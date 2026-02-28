/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "io"
import "sync"
import "reflect"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

// StorageComputeProxy wraps a main storage with lazy-evaluation support.
// Values are computed on demand via a computor lambda and cached in a delta map
// until Compress() materializes them into a compressed main storage.
type StorageComputeProxy struct {
	main       ColumnStorage                      // after Compress() — typically StorageSCMER or compressed type
	delta      map[uint32]scm.Scmer               // sparse overwrites (lazy-computed values before Compress)
	validMask  NonLockingReadMap.NonBlockingBitMap // 1=valid, 0=needs compute
	compressed bool                                // true after Compress() → skip validMask, read from main
	computor   scm.Scmer                           // computation lambda
	inputCols  []string                            // column names the computor reads
	shard      *storageShard                       // back-reference for reading input columns
	colName    string                              // own column name (for cycle protection)
	mu         sync.RWMutex                        // protects delta map + compressed flag
	count      uint32                              // total row count at creation
}

func (p *StorageComputeProxy) String() string {
	return "compute-proxy"
}

func (p *StorageComputeProxy) ComputeSize() uint {
	var sz uint = 128 // struct overhead
	sz += p.validMask.ComputeSize()
	sz += uint(len(p.delta)) * 24 // rough estimate per map entry
	if p.main != nil {
		sz += p.main.ComputeSize()
	}
	return sz
}

// GetValue returns the value at idx, computing on demand if necessary.
func (p *StorageComputeProxy) GetValue(idx uint32) scm.Scmer {
	// Fast path 1: fully compressed → all values in main
	if p.compressed {
		return p.main.GetValue(idx)
	}

	// Fast path 2: valid bit set → value in delta or main
	if p.validMask.Get(idx) {
		p.mu.RLock()
		if val, ok := p.delta[idx]; ok {
			p.mu.RUnlock()
			return val
		}
		p.mu.RUnlock()
		if p.main != nil {
			return p.main.GetValue(idx)
		}
	}

	// Slow path: compute on demand
	colvalues := make([]scm.Scmer, len(p.inputCols))
	for i, col := range p.inputCols {
		colvalues[i] = p.shard.getColumnStorageOrPanic(col).GetValue(idx)
	}
	val := scm.Apply(p.computor, colvalues...)

	p.mu.Lock()
	p.delta[idx] = val
	p.mu.Unlock()
	p.validMask.Set(idx, true)

	return val
}

func (p *StorageComputeProxy) GetCachedReader() ColumnReader {
	return p
}

// Compress materializes all values into a compressed main storage.
func (p *StorageComputeProxy) Compress() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Early exit if already compressed and no dirty delta
	if p.compressed && len(p.delta) == 0 {
		return
	}

	fn := scm.OptimizeProcToSerialFunction(p.computor)
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = newCachedColumnReader(p.shard.getColumnStorageOrPanic(col))
	}

	colvalues := make([]scm.Scmer, len(p.inputCols))
	getValue := func(idx uint32) scm.Scmer {
		if val, ok := p.delta[idx]; ok {
			return val
		}
		if p.main != nil && p.validMask.Get(idx) {
			return p.main.GetValue(idx)
		}
		// compute
		for j := range readers {
			colvalues[j] = readers[j].GetValue(idx)
		}
		return fn(colvalues...)
	}

	// Standard proposeCompression loop (same as shard rebuild)
	var newcol ColumnStorage = new(StorageSCMER)
	for {
		newcol.prepare()
		for i := uint32(0); i < p.count; i++ {
			newcol.scan(i, getValue(i))
		}
		proposed := newcol.proposeCompression(p.count)
		if proposed == nil {
			break
		}
		newcol = proposed
	}
	newcol.init(p.count)
	for i := uint32(0); i < p.count; i++ {
		newcol.build(i, getValue(i))
	}
	newcol.finish()

	p.main = newcol
	p.delta = make(map[uint32]scm.Scmer)
	p.validMask.Reset()
	p.compressed = true
}

// CompressFiltered eagerly computes only rows matching the filter predicate.
// Unmatched rows stay lazy and are computed on demand via GetValue.
func (p *StorageComputeProxy) CompressFiltered(filterCols []string, filter scm.Scmer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fn := scm.OptimizeProcToSerialFunction(p.computor)
	filterFn := scm.OptimizeProcToSerialFunction(filter)

	filterReaders := make([]ColumnReader, len(filterCols))
	for i, col := range filterCols {
		filterReaders[i] = newCachedColumnReader(p.shard.getColumnStorageOrPanic(col))
	}
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = newCachedColumnReader(p.shard.getColumnStorageOrPanic(col))
	}

	filterValues := make([]scm.Scmer, len(filterCols))
	colvalues := make([]scm.Scmer, len(p.inputCols))
	for i := uint32(0); i < p.count; i++ {
		for j := range filterReaders {
			filterValues[j] = filterReaders[j].GetValue(i)
		}
		if scm.ToBool(filterFn(filterValues...)) {
			for j := range readers {
				colvalues[j] = readers[j].GetValue(i)
			}
			p.delta[i] = fn(colvalues...)
			p.validMask.Set(i, true)
		}
	}
	// Don't set compressed=true → unmatched rows stay lazy for on-demand GetValue
}

// Invalidate marks a single row as needing recomputation.
func (p *StorageComputeProxy) Invalidate(idx uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// Try in-place update if main supports SetValue (StorageSCMER)
	if p.compressed {
		if scmer, ok := p.main.(*StorageSCMER); ok {
			// recompute single value and write directly
			colvalues := make([]scm.Scmer, len(p.inputCols))
			for i, col := range p.inputCols {
				colvalues[i] = p.shard.getColumnStorageOrPanic(col).GetValue(idx)
			}
			val := scm.Apply(p.computor, colvalues...)
			scmer.SetValue(idx, val)
			return // stay compressed, no bitmap change needed
		}
		// main is compressed type → can't SetValue → fall back to lazy
		p.compressed = false
	}
	p.validMask.Set(idx, false)
	delete(p.delta, idx)
}

// InvalidateAll marks all rows as needing recomputation.
func (p *StorageComputeProxy) InvalidateAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.compressed = false
	p.validMask.Reset()
	p.delta = make(map[uint32]scm.Scmer)
}

// proposeCompression returns nil — the proxy does not participate in shard
// rebuild's compression loop. During rebuild, the old proxy is READ via
// GetCachedReader/GetValue and a NEW storage is built by the rebuild loop.
func (p *StorageComputeProxy) proposeCompression(i uint32) ColumnStorage {
	return nil
}

func (p *StorageComputeProxy) prepare() {
	panic("StorageComputeProxy should not be used as rebuild target")
}
func (p *StorageComputeProxy) scan(i uint32, value scm.Scmer) {
	panic("StorageComputeProxy should not be used as rebuild target")
}
func (p *StorageComputeProxy) init(i uint32) {
	panic("StorageComputeProxy should not be used as rebuild target")
}
func (p *StorageComputeProxy) build(i uint32, value scm.Scmer) {
	panic("StorageComputeProxy should not be used as rebuild target")
}
func (p *StorageComputeProxy) finish() {
	panic("StorageComputeProxy should not be used as rebuild target")
}

// Serialize writes the proxy to the given writer.
func (p *StorageComputeProxy) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(50)) // magic byte
	binary.Write(f, binary.LittleEndian, p.count)

	// inputCols
	binary.Write(f, binary.LittleEndian, uint16(len(p.inputCols)))
	for _, col := range p.inputCols {
		b := []byte(col)
		binary.Write(f, binary.LittleEndian, uint16(len(b)))
		f.Write(b)
	}

	// computor as JSON
	computorJSON, err := json.Marshal(p.computor)
	if err != nil {
		panic(err)
	}
	binary.Write(f, binary.LittleEndian, uint32(len(computorJSON)))
	f.Write(computorJSON)

	// compressed flag
	if p.compressed {
		binary.Write(f, binary.LittleEndian, uint8(1))
	} else {
		binary.Write(f, binary.LittleEndian, uint8(0))
	}

	// main storage or delta
	if p.compressed && p.main != nil {
		binary.Write(f, binary.LittleEndian, uint8(1)) // has main
		p.main.Serialize(f)                             // nested — includes its own magic byte
	} else {
		binary.Write(f, binary.LittleEndian, uint8(0)) // no main
		// write delta
		binary.Write(f, binary.LittleEndian, uint32(len(p.delta)))
		for idx, val := range p.delta {
			binary.Write(f, binary.LittleEndian, idx)
			valJSON, err := json.Marshal(val)
			if err != nil {
				panic(err)
			}
			binary.Write(f, binary.LittleEndian, uint32(len(valJSON)))
			f.Write(valJSON)
		}
	}

	// validMask: serialize set bits
	validCount := p.validMask.Count()
	binary.Write(f, binary.LittleEndian, uint32(validCount))
	p.validMask.Iterate(func(idx uint32) {
		binary.Write(f, binary.LittleEndian, idx)
	})
}

// Deserialize reads the proxy from the given reader.
// Note: magic byte 50 is already consumed by the caller.
func (p *StorageComputeProxy) Deserialize(f io.Reader) uint {
	binary.Read(f, binary.LittleEndian, &p.count)

	// inputCols
	var numCols uint16
	binary.Read(f, binary.LittleEndian, &numCols)
	p.inputCols = make([]string, numCols)
	for i := uint16(0); i < numCols; i++ {
		var slen uint16
		binary.Read(f, binary.LittleEndian, &slen)
		buf := make([]byte, slen)
		io.ReadFull(f, buf)
		p.inputCols[i] = string(buf)
	}

	// computor from JSON
	var computorLen uint32
	binary.Read(f, binary.LittleEndian, &computorLen)
	computorBuf := make([]byte, computorLen)
	io.ReadFull(f, computorBuf)
	var computorRaw any
	json.Unmarshal(computorBuf, &computorRaw)
	p.computor = scm.TransformFromJSON(computorRaw)

	// compressed flag
	var compressedFlag uint8
	binary.Read(f, binary.LittleEndian, &compressedFlag)
	p.compressed = compressedFlag != 0

	// main or delta
	var hasMain uint8
	binary.Read(f, binary.LittleEndian, &hasMain)
	if hasMain != 0 {
		var magicbyte uint8
		binary.Read(f, binary.LittleEndian, &magicbyte)
		main := reflect.New(storages[magicbyte]).Interface().(ColumnStorage)
		main.Deserialize(f)
		p.main = main
	} else {
		var deltaLen uint32
		binary.Read(f, binary.LittleEndian, &deltaLen)
		p.delta = make(map[uint32]scm.Scmer, deltaLen)
		for i := uint32(0); i < deltaLen; i++ {
			var idx uint32
			binary.Read(f, binary.LittleEndian, &idx)
			var valLen uint32
			binary.Read(f, binary.LittleEndian, &valLen)
			valBuf := make([]byte, valLen)
			io.ReadFull(f, valBuf)
			var valRaw any
			json.Unmarshal(valBuf, &valRaw)
			p.delta[idx] = scm.TransformFromJSON(valRaw)
		}
	}

	// validMask
	if p.delta == nil {
		p.delta = make(map[uint32]scm.Scmer)
	}
	var validCount uint32
	binary.Read(f, binary.LittleEndian, &validCount)
	for i := uint32(0); i < validCount; i++ {
		var idx uint32
		binary.Read(f, binary.LittleEndian, &idx)
		p.validMask.Set(idx, true)
	}

	// shard back-reference is set by post-load hook in ensureColumnLoaded
	return uint(p.count)
}
