/*
Copyright (C) 2026  Carl-Philip Hänsch

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

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/launix-de/memcp/scm"
	"io"
	"math/bits"
	"unsafe"
)

/*
	StorageEnum: k-ary rANS entropy-coded columnar storage for low-cardinality
	columns (up to 8 distinct values including NULL).

	Compared to PFOR (StorageInt), rANS encodes each symbol with a cost
	proportional to -log2(probability). For a boolean column that is 99% false,
	PFOR uses 1 bit/elem, while rANS uses ~0.08 bits/elem — a 12x improvement.

	Storage layout:
	  data[]    — uint64 rANS-coded chunks (variable elements per chunk)
	  jumpL1[]  — uint32 absolute cumulative counts every stride chunks
	  jumpL2[]  — uint16 relative cumulative counts per chunk

	Access patterns:
	  With cache hint:  O(chunk_size) — skips binary search, decodes from chunk start
	  Random access:    O(log(chunks) + chunk_size) via binary search on jump index
	  With per-thread EnumDecodeCache: O(1) sequential via GetValueCached
*/

const enumBitShift = 8
const enumBitMask = ^uint64(0) >> (64 - enumBitShift) // 0xFF
const enumBitModulo = uint64(1) << enumBitShift       // 256
const enumMaxSymbols = 8

type StorageEnum struct {
	// rANS coded payload
	data []uint64

	// 2-level jump index
	jumpL1       []uint32
	jumpL2       []uint16
	jumpL1Stride int

	// symbol table
	values     [enumMaxSymbols]scm.Scmer
	k          uint8 // number of symbols (including NULL if present)
	thresholds [enumMaxSymbols - 1]uint64
	widths     [enumMaxSymbols]uint64
	invWidths  [enumMaxSymbols]uint64

	count uint64

	// scan-phase temporaries
	scanFreqs [enumMaxSymbols]uint64
	scanTotal uint64

	// build-phase temporaries: we need to reverse-buffer elements
	// because rANS encodes in reverse order
	buildBuf []scm.Scmer
}

func enumFastDivMod(n, d, inv uint64) (q, r uint64) {
	q, _ = bits.Mul64(n, inv)
	r = n - q*d
	if r >= d {
		q++
		r -= d
	}
	return
}

func (s *StorageEnum) String() string {
	return fmt.Sprintf("enum[%d]", s.k)
}

func (s *StorageEnum) ComputeSize() uint {
	var sz uint = 200 // struct overhead estimate
	sz += 8 * uint(len(s.data))
	sz += 4 * uint(len(s.jumpL1))
	sz += 2 * uint(len(s.jumpL2))
	for i := uint8(0); i < s.k; i++ {
		sz += scm.ComputeSize(s.values[i])
	}
	return sz
}

// --- rANS codec helpers ---

func (s *StorageEnum) symbolLo(idx int) uint64 {
	if idx == 0 {
		return 0
	}
	return s.thresholds[idx-1]
}

func (s *StorageEnum) findValue(val scm.Scmer) int {
	for i := uint8(0); i < s.k; i++ {
		// strict: NULL only matches NULL
		if val.IsNil() == s.values[i].IsNil() && (val.IsNil() || scm.Equal(s.values[i], val)) {
			return int(i)
		}
	}
	panic(fmt.Sprintf("StorageEnum: value %v not in symbol set", val))
}

func (s *StorageEnum) decodeSymbol(slice uint64) int {
	for i := uint8(0); i < s.k-1; i++ {
		if slice < s.thresholds[i] {
			return int(i)
		}
	}
	return int(s.k) - 1
}

func (s *StorageEnum) jumpCum(j int) int {
	g := j / s.jumpL1Stride
	base := uint32(0)
	if g > 0 {
		base = s.jumpL1[g-1]
	}
	return int(base) + int(s.jumpL2[j])
}

func (s *StorageEnum) decodeOne(buffer uint64) (scm.Scmer, uint64) {
	slice := buffer & enumBitMask
	symIdx := s.decodeSymbol(slice)
	width := s.widths[symIdx]
	return s.values[symIdx], (buffer>>enumBitShift)*width + slice - s.symbolLo(symIdx)
}

// --- ColumnStorage interface ---

func (s *StorageEnum) prepare() {
	s.scanFreqs = [enumMaxSymbols]uint64{}
	s.scanTotal = 0
	s.k = 0
}

func (s *StorageEnum) scan(i uint, value scm.Scmer) {
	s.scanTotal++
	// find existing symbol (strict: NULL only matches NULL)
	for j := uint8(0); j < s.k; j++ {
		if value.IsNil() == s.values[j].IsNil() && (value.IsNil() || scm.Equal(s.values[j], value)) {
			s.scanFreqs[j]++
			return
		}
	}
	// new symbol
	if s.k < enumMaxSymbols {
		s.values[s.k] = value
		s.scanFreqs[s.k] = 1
		s.k++
	}
}

func (s *StorageEnum) proposeCompression(i uint) ColumnStorage {
	return nil // terminal
}

func (s *StorageEnum) init(i uint) {
	s.count = uint64(i)

	if s.k < 2 {
		// degenerate: 0 or 1 distinct values; pad to 2 symbols
		if s.k == 0 {
			s.values[0] = scm.NewNil()
			s.values[1] = scm.NewBool(false)
			s.scanFreqs[0] = s.scanTotal
			s.scanFreqs[1] = 0
		} else {
			// 1 symbol: add a dummy
			s.values[s.k] = scm.NewNil()
			s.scanFreqs[s.k] = 0
		}
		s.k = 2
	}

	// Build slot widths from frequencies
	k := int(s.k)
	total := uint64(0)
	for j := 0; j < k; j++ {
		total += s.scanFreqs[j]
	}
	if total == 0 {
		total = 1
	}

	slots := [enumMaxSymbols]uint64{}
	remaining := int(enumBitModulo) - k
	for j := 0; j < k; j++ {
		slots[j] = 1 // minimum 1
	}
	distributed := 0
	for j := 0; j < k; j++ {
		extra := int(s.scanFreqs[j]) * remaining / int(total)
		slots[j] += uint64(extra)
		distributed += extra
	}
	leftover := remaining - distributed
	if leftover > 0 {
		maxIdx := 0
		for j := 1; j < k; j++ {
			if s.scanFreqs[j] > s.scanFreqs[maxIdx] {
				maxIdx = j
			}
		}
		slots[maxIdx] += uint64(leftover)
	}

	// Build thresholds, widths, inverse table
	cum := uint64(0)
	for j := 0; j < k; j++ {
		s.widths[j] = slots[j]
		s.invWidths[j] = ^uint64(0) / slots[j]
		if j < k-1 {
			cum += slots[j]
			s.thresholds[j] = cum
		}
	}

	// Allocate build buffer (freed in finish)
	s.buildBuf = make([]scm.Scmer, i)
}

func (s *StorageEnum) build(i uint, value scm.Scmer) {
	s.buildBuf[i] = value
}

func (s *StorageEnum) finish() {
	n := int(s.count)
	s.data = s.data[:0]
	var chunkSizes []int

	var buffer uint64
	bufferlen := 0

	// encode in reverse order (rANS requirement)
	for i := n - 1; i >= 0; i-- {
		symIdx := s.findValue(s.buildBuf[i])
		lo := s.symbolLo(symIdx)
		width := s.widths[symIdx]
		inv := s.invWidths[symIdx]

		bufferx, rest := enumFastDivMod(buffer, width, inv)
		if bufferx > ^uint64(0)>>enumBitShift {
			s.data = append(s.data, buffer)
			chunkSizes = append(chunkSizes, bufferlen)
			buffer = 0
			bufferlen = 0
			bufferx = 0
		}
		buffer = (bufferx << enumBitShift) + lo + rest
		bufferlen++
	}
	s.data = append(s.data, buffer)
	chunkSizes = append(chunkSizes, bufferlen)

	// Free build buffer
	s.buildBuf = nil

	// Build 2-level jump index
	numChunks := len(s.data)

	// Auto-tune stride
	maxCS := 0
	for _, cs := range chunkSizes {
		if cs > maxCS {
			maxCS = cs
		}
	}
	s.jumpL1Stride = 1
	for s.jumpL1Stride*2*maxCS <= 65535 {
		s.jumpL1Stride *= 2
	}
	if s.jumpL1Stride < 1 {
		s.jumpL1Stride = 1
	}

	numGroups := (numChunks + s.jumpL1Stride - 1) / s.jumpL1Stride
	s.jumpL1 = make([]uint32, numGroups)
	s.jumpL2 = make([]uint16, numChunks)

	cumAbs := uint32(0)
	groupBase := uint32(0)
	for j := 0; j < numChunks; j++ {
		cumAbs += uint32(chunkSizes[numChunks-1-j])
		s.jumpL2[j] = uint16(cumAbs - groupBase)
		if (j+1)%s.jumpL1Stride == 0 {
			g := (j + 1) / s.jumpL1Stride
			s.jumpL1[g-1] = cumAbs
			groupBase = cumAbs
		}
	}
	if numChunks%s.jumpL1Stride != 0 {
		s.jumpL1[numGroups-1] = cumAbs
	}
}

// EnumDecodeCache holds per-goroutine rANS decode state for O(1) sequential access.
// Allocate one per worker goroutine and pass to GetValueCached.
type EnumDecodeCache struct {
	fwdChunk int
	start    int
	pos      int
	buf      uint64
	valid    bool
}

// cachedEnumReader wraps a StorageEnum with a private EnumDecodeCache.
// Returned by StorageEnum.GetCachedReader(). Must not be shared between goroutines.
type cachedEnumReader struct {
	s     *StorageEnum
	cache EnumDecodeCache
}

func (r *cachedEnumReader) GetValue(i uint) scm.Scmer {
	return r.s.GetValueCached(i, &r.cache)
}

func (s *StorageEnum) GetCachedReader() ColumnReader {
	return &cachedEnumReader{s: s}
}

// GetValue is safe for concurrent use — it is fully read-only on the struct.
// Uses binary search + sequential decode from chunk start. For O(1) sequential
// access, use GetCachedReader() which returns a per-goroutine cached wrapper.
func (s *StorageEnum) GetValue(i uint) scm.Scmer {
	idx := int(i)
	fwdIdx := s.findChunk(idx)
	chunkStart := 0
	if fwdIdx > 0 {
		chunkStart = s.jumpCum(fwdIdx - 1)
	}
	dataIdx := len(s.data) - 1 - fwdIdx
	buffer := s.data[dataIdx]
	posInChunk := idx - chunkStart
	var result scm.Scmer
	for j := 0; j <= posInChunk; j++ {
		result, buffer = s.decodeOne(buffer)
	}
	return result
}

// GetValueCached provides O(1) sequential access using a per-goroutine cache.
// The cache must not be shared between goroutines.
func (s *StorageEnum) GetValueCached(i uint, c *EnumDecodeCache) scm.Scmer {
	idx := int(i)

	if c.valid {
		chunkEnd := s.jumpCum(c.fwdChunk)
		// fast path: index is ahead of cache position in same chunk
		if idx >= c.start+c.pos && idx < chunkEnd {
			buffer := c.buf
			var result scm.Scmer
			target := idx - c.start
			for j := c.pos; j <= target; j++ {
				result, buffer = s.decodeOne(buffer)
			}
			c.pos = target + 1
			c.buf = buffer
			return result
		}
		// next chunk fast path
		if idx >= chunkEnd {
			nextFwd := c.fwdChunk + 1
			if nextFwd < len(s.jumpL2) && idx < s.jumpCum(nextFwd) {
				dataIdx := len(s.data) - 1 - nextFwd
				buffer := s.data[dataIdx]
				posInChunk := idx - chunkEnd
				var result scm.Scmer
				for j := 0; j <= posInChunk; j++ {
					result, buffer = s.decodeOne(buffer)
				}
				c.fwdChunk = nextFwd
				c.start = chunkEnd
				c.pos = posInChunk + 1
				c.buf = buffer
				return result
			}
		}
	}

	// Binary search fallback
	fwdIdx := s.findChunk(idx)
	chunkStart := 0
	if fwdIdx > 0 {
		chunkStart = s.jumpCum(fwdIdx - 1)
	}

	dataIdx := len(s.data) - 1 - fwdIdx
	buffer := s.data[dataIdx]
	posInChunk := idx - chunkStart
	var result scm.Scmer
	for j := 0; j <= posInChunk; j++ {
		result, buffer = s.decodeOne(buffer)
	}

	c.valid = true
	c.fwdChunk = fwdIdx
	c.start = chunkStart
	c.pos = posInChunk + 1
	c.buf = buffer
	return result
}

// findChunk returns the chunk index containing element idx via binary search.
func (s *StorageEnum) findChunk(idx int) int {
	lo, hi := 0, len(s.jumpL2)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if s.jumpCum(mid) <= idx {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// --- Serialization ---

func (s *StorageEnum) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(40)) // magic byte 40 = StorageEnum
	binary.Write(f, binary.LittleEndian, uint8(s.k))
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint32(s.jumpL1Stride))
	binary.Write(f, binary.LittleEndian, uint64(len(s.data)))
	binary.Write(f, binary.LittleEndian, uint64(len(s.jumpL1)))
	binary.Write(f, binary.LittleEndian, uint64(len(s.jumpL2)))

	// symbol frequencies for rebuilding widths/thresholds
	for j := uint8(0); j < s.k; j++ {
		binary.Write(f, binary.LittleEndian, s.scanFreqs[j])
	}

	// symbol values as JSON lines
	for j := uint8(0); j < s.k; j++ {
		b, _ := json.Marshal(s.values[j])
		binary.Write(f, binary.LittleEndian, uint32(len(b)))
		f.Write(b)
	}

	// data chunks
	if len(s.data) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.data[0])), 8*len(s.data)))
	}
	// jumpL1
	if len(s.jumpL1) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.jumpL1[0])), 4*len(s.jumpL1)))
	}
	// jumpL2
	if len(s.jumpL2) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.jumpL2[0])), 2*len(s.jumpL2)))
	}
}

func (s *StorageEnum) Deserialize(f io.Reader) uint {
	binary.Read(f, binary.LittleEndian, &s.k)
	binary.Read(f, binary.LittleEndian, &s.count)
	var stride uint32
	binary.Read(f, binary.LittleEndian, &stride)
	s.jumpL1Stride = int(stride)
	var dataLen, l1Len, l2Len uint64
	binary.Read(f, binary.LittleEndian, &dataLen)
	binary.Read(f, binary.LittleEndian, &l1Len)
	binary.Read(f, binary.LittleEndian, &l2Len)

	// Read frequencies
	for j := uint8(0); j < s.k; j++ {
		binary.Read(f, binary.LittleEndian, &s.scanFreqs[j])
	}

	// Read symbol values
	for j := uint8(0); j < s.k; j++ {
		var vlen uint32
		binary.Read(f, binary.LittleEndian, &vlen)
		buf := make([]byte, vlen)
		io.ReadFull(f, buf)
		var v any
		json.Unmarshal(buf, &v)
		s.values[j] = scm.TransformFromJSON(v)
	}

	// Rebuild widths/thresholds from frequencies
	s.rebuildCodec()

	// Read data
	if dataLen > 0 {
		raw := make([]byte, dataLen*8)
		io.ReadFull(f, raw)
		s.data = unsafe.Slice((*uint64)(unsafe.Pointer(&raw[0])), dataLen)
	}
	// Read jumpL1
	if l1Len > 0 {
		raw := make([]byte, l1Len*4)
		io.ReadFull(f, raw)
		s.jumpL1 = unsafe.Slice((*uint32)(unsafe.Pointer(&raw[0])), l1Len)
	}
	// Read jumpL2
	if l2Len > 0 {
		raw := make([]byte, l2Len*2)
		io.ReadFull(f, raw)
		s.jumpL2 = unsafe.Slice((*uint16)(unsafe.Pointer(&raw[0])), l2Len)
	}

	return uint(s.count)
}

// rebuildCodec reconstructs thresholds/widths/invWidths from scanFreqs.
func (s *StorageEnum) rebuildCodec() {
	k := int(s.k)
	if k < 2 {
		k = 2
		s.k = 2
	}
	total := uint64(0)
	for j := 0; j < k; j++ {
		total += s.scanFreqs[j]
	}
	if total == 0 {
		total = 1
	}

	slots := [enumMaxSymbols]uint64{}
	remaining := int(enumBitModulo) - k
	for j := 0; j < k; j++ {
		slots[j] = 1
	}
	distributed := 0
	for j := 0; j < k; j++ {
		extra := int(s.scanFreqs[j]) * remaining / int(total)
		slots[j] += uint64(extra)
		distributed += extra
	}
	leftover := remaining - distributed
	if leftover > 0 {
		maxIdx := 0
		for j := 1; j < k; j++ {
			if s.scanFreqs[j] > s.scanFreqs[maxIdx] {
				maxIdx = j
			}
		}
		slots[maxIdx] += uint64(leftover)
	}

	cum := uint64(0)
	for j := 0; j < k; j++ {
		s.widths[j] = slots[j]
		s.invWidths[j] = ^uint64(0) / slots[j]
		if j < k-1 {
			cum += slots[j]
			s.thresholds[j] = cum
		}
	}
}
