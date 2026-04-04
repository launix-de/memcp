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
import "fmt"
import "sync"
import "sync/atomic"
import "time"
import "reflect"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"
import "github.com/launix-de/NonLockingReadMap"

type storageComputeVariant struct {
	main       ColumnStorage
	delta      map[uint32]scm.Scmer
	validMask  NonLockingReadMap.NonBlockingBitMap
	compressed bool
	count      uint32

	invalidateNsSinceRead atomic.Int64
	lastRecomputeNs       atomic.Int64
	lastUsed              atomic.Int64
	mu                    sync.RWMutex
}

func newStorageComputeVariant(count uint32) *storageComputeVariant {
	return &storageComputeVariant{
		delta: make(map[uint32]scm.Scmer),
		count: count,
	}
}

type computeVariantReader struct {
	proxy   *StorageComputeProxy
	variant *storageComputeVariant
	readers []ColumnReader
	tx      *TxContext
}

func applyWithTx(tx *TxContext, fn scm.Scmer, args ...scm.Scmer) scm.Scmer {
	if fn.IsProc() {
		proc := *fn.Proc()
		proc.En = bindSessionEnv(proc.En, txSessionScmer(tx))
		fn = scm.NewProcStruct(proc)
	}
	if tx == nil || tx.Session.IsNil() {
		return scm.Apply(fn, args...)
	}
	return scm.WithSession(tx.Session, scm.NewFunc(func(a ...scm.Scmer) scm.Scmer {
		return scm.Apply(fn, args...)
	}))
}

// StorageComputeProxy wraps a main storage with lazy-evaluation support.
// Values are computed on demand via a computor lambda and cached in a delta map
// until Compress() materializes them into a compressed main storage.
type StorageComputeProxy struct {
	main       ColumnStorage                       // after Compress() — typically StorageSCMER or compressed type
	delta      map[uint32]scm.Scmer                // sparse overwrites (lazy-computed values before Compress)
	validMask  NonLockingReadMap.NonBlockingBitMap // 1=valid, 0=needs compute
	compressed bool                                // true after Compress() → skip validMask, read from main
	computor   scm.Scmer                           // computation lambda
	inputCols  []string                            // column names the computor reads
	shard      *storageShard                       // back-reference for reading input columns
	colName    string                              // own column name (for cycle protection)
	mu         sync.RWMutex                        // protects delta map + compressed flag
	count      uint32                              // total row count at creation
	// ORC support: when isOrdered=true, single-row lazy compute is disabled.
	// Validity is tracked per-row via validMask (1=valid, 0=needs compute).
	// Invalidation sets bits to 0; on-demand recompute sets them back to 1.
	isOrdered bool
	// Invalidation telemetry: tracks cumulative cost of selective invalidations
	// since last read. When invalidation cost exceeds last recompute cost,
	// switches to InvalidateAll() to avoid death-by-a-thousand-cuts.
	invalidateNsSinceRead atomic.Int64 // cumulative invalidation nanoseconds since last read
	lastRecomputeNs       atomic.Int64 // nanoseconds of the last full/suffix recompute
	sessionKeys           []string
	variants              map[string]*storageComputeVariant
	variantsMu            sync.RWMutex
}

func (p *StorageComputeProxy) hasSessionVariants() bool {
	return len(p.sessionKeys) > 0
}

func (p *StorageComputeProxy) sessionVariantKey(tx *TxContext) string {
	if tx == nil || !p.hasSessionVariants() {
		return ""
	}
	keyExpr := make([]scm.Scmer, 0, len(p.sessionKeys)*2+1)
	keyExpr = append(keyExpr, scm.NewSymbol("list"))
	for _, key := range p.sessionKeys {
		keyExpr = append(keyExpr, scm.NewString(key), tx.SessionValue(key))
	}
	return encodeScmerToString(scm.NewSlice(keyExpr), nil, nil)
}

func (p *StorageComputeProxy) currentVariant(tx *TxContext, create bool) *storageComputeVariant {
	if !p.hasSessionVariants() {
		return nil
	}
	key := p.sessionVariantKey(tx)
	p.variantsMu.RLock()
	variant := p.variants[key]
	p.variantsMu.RUnlock()
	if variant == nil && create {
		p.variantsMu.Lock()
		variant = p.variants[key]
		if variant == nil {
			variant = newStorageComputeVariant(p.count)
			if p.variants == nil {
				p.variants = make(map[string]*storageComputeVariant)
			}
			p.variants[key] = variant
		}
		p.variantsMu.Unlock()
	}
	if variant != nil {
		variant.lastUsed.Store(time.Now().UnixNano())
	}
	return variant
}

// cloneComputeProxyRows ports a compute/ORC proxy onto a rebuilt shard without
// evaluating the computor. Cached rows stay cached, invalid rows stay lazy.
func cloneComputeProxyRows(oldProxy *StorageComputeProxy, newShard *storageShard, oldRowIDs []uint32) *StorageComputeProxy {
	newProxy := &StorageComputeProxy{
		delta:     make(map[uint32]scm.Scmer),
		computor:  oldProxy.computor,
		inputCols: oldProxy.inputCols,
		shard:     newShard,
		colName:   oldProxy.colName,
		count:     uint32(len(oldRowIDs)),
		isOrdered: oldProxy.isOrdered,
	}
	appendComputeProxyRows(newProxy, oldProxy, oldRowIDs, 0)
	return newProxy
}

func appendComputeProxyRows(newProxy *StorageComputeProxy, oldProxy *StorageComputeProxy, oldRowIDs []uint32, startIdx uint32) uint32 {
	if oldProxy.isOrdered && oldProxy.shard != nil && oldProxy.shard.t != nil {
		// Ordered-reduce proxies are populated under table.orcMu. Rebuild must
		// not snapshot them mid-recompute, otherwise a background rebuild can
		// publish an all-invalid proxy even though a foreground query just
		// materialized the values on the old shard.
		oldProxy.shard.t.orcMu.Lock()
		defer oldProxy.shard.t.orcMu.Unlock()
	}
	oldProxy.mu.RLock()
	defer oldProxy.mu.RUnlock()
	newIdx := startIdx
	for _, oldIdx := range oldRowIDs {
		val, inDelta := oldProxy.delta[oldIdx]
		if !inDelta {
			// Proxy row ids may refer to forwarded delta rows that were inserted
			// after the proxy's main storage was materialized. Those rows are only
			// safe to port if they have an explicit cached delta entry; otherwise
			// they must stay lazy-invalid on the rebuilt shard instead of reading
			// past the old main storage.
			if oldIdx >= oldProxy.count {
				newIdx++
				continue
			}
			if !oldProxy.compressed && !oldProxy.validMask.Get(uint(oldIdx)) {
				newIdx++
				continue
			}
			if oldProxy.main == nil {
				newIdx++
				continue
			}
			val = oldProxy.main.GetValue(oldIdx)
		}
		newProxy.delta[newIdx] = val
		newProxy.validMask.Set(uint(newIdx), true)
		newIdx++
	}
	return newIdx
}

func (p *StorageComputeProxy) String() string {
	return "compute-proxy"
}

func (r *computeVariantReader) GetValue(idx uint32) scm.Scmer {
	p := r.proxy
	v := r.variant

	v.mu.RLock()
	if val, ok := v.delta[idx]; ok {
		v.mu.RUnlock()
		return val
	}
	v.mu.RUnlock()

	if v.compressed && idx < v.count && v.main != nil {
		return v.main.GetValue(idx)
	}
	if v.validMask.Get(uint(idx)) && idx < v.count && v.main != nil {
		return v.main.GetValue(idx)
	}

	colvalues := make([]scm.Scmer, len(r.readers))
	for i := range r.readers {
		colvalues[i] = r.readers[i].GetValue(idx)
	}
	val := applyWithTx(r.tx, p.computor, colvalues...)

	v.mu.Lock()
	v.delta[idx] = val
	v.mu.Unlock()
	v.validMask.Set(uint(idx), true)

	return val
}

func (p *StorageComputeProxy) GetCachedReaderTx(tx *TxContext) ColumnReader {
	if !p.hasSessionVariants() {
		return p
	}
	variant := p.currentVariant(tx, true)
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(col, tx))
	}
	return &computeVariantReader{
		proxy:   p,
		variant: variant,
		readers: readers,
		tx:      tx,
	}
}

func (p *StorageComputeProxy) forEachVariant(fn func(*storageComputeVariant)) {
	p.variantsMu.RLock()
	variants := make([]*storageComputeVariant, 0, len(p.variants))
	for _, variant := range p.variants {
		variants = append(variants, variant)
	}
	p.variantsMu.RUnlock()
	for _, variant := range variants {
		fn(variant)
	}
}

func (p *StorageComputeProxy) compressVariant(v *storageComputeVariant, tx *TxContext) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.compressed && len(v.delta) == 0 {
		return
	}
	if v.count == 0 {
		v.compressed = true
		return
	}

	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(col, tx))
	}

	colvalues := make([]scm.Scmer, len(p.inputCols))
	getValue := func(idx uint32) scm.Scmer {
		if val, ok := v.delta[idx]; ok {
			return val
		}
		if v.main != nil && v.validMask.Get(uint(idx)) {
			return v.main.GetValue(idx)
		}
		for j := range readers {
			colvalues[j] = readers[j].GetValue(idx)
		}
		return applyWithTx(tx, p.computor, colvalues...)
	}

	var newcol ColumnStorage = new(StorageSCMER)
	for {
		newcol.prepare()
		for i := uint32(0); i < v.count; i++ {
			newcol.scan(i, getValue(i))
		}
		proposed := newcol.proposeCompression(v.count)
		if proposed == nil {
			break
		}
		newcol = proposed
	}
	newcol.init(v.count)
	for i := uint32(0); i < v.count; i++ {
		newcol.build(i, getValue(i))
	}
	newcol.finish()

	v.main = newcol
	v.delta = make(map[uint32]scm.Scmer)
	v.validMask.Reset()
	v.compressed = true
}

func (p *StorageComputeProxy) compressFilteredVariant(v *storageComputeVariant, tx *TxContext, filterCols []string, filter scm.Scmer) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.count == 0 {
		return
	}

	filterFn := scm.OptimizeProcToSerialFunction(filter)
	filterReaders := make([]ColumnReader, len(filterCols))
	for i, col := range filterCols {
		filterReaders[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(col, tx))
	}
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(col, tx))
	}

	filterValues := make([]scm.Scmer, len(filterCols))
	colvalues := make([]scm.Scmer, len(p.inputCols))
	for i := uint32(0); i < v.count; i++ {
		for j := range filterReaders {
			filterValues[j] = filterReaders[j].GetValue(i)
		}
		if scm.ToBool(filterFn(filterValues...)) {
			for j := range readers {
				colvalues[j] = readers[j].GetValue(i)
			}
			v.delta[i] = applyWithTx(tx, p.computor, colvalues...)
			v.validMask.Set(uint(i), true)
		}
	}
}

// orcCol returns the column definition for this ORC proxy's column.
func (p *StorageComputeProxy) orcCol() *column {
	for _, c := range p.shard.t.Columns {
		if c.Name == p.colName {
			return c
		}
	}
	return nil
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
	// ORC path: validity tracked per-row via validMask.
	if p.isOrdered {
		if !p.validMask.Get(uint(idx)) {
			// During an active recompute (scan_order reading ORC values):
			// Return nil for invalid rows (reducer uses nil to detect "needs compute").
			// Return cached value for valid rows (enables Phase 1 skip + Phase 3 convergence).
			if atomic.LoadInt32(&p.shard.t.orcRecomputing) > 0 {
				if p.validMask.Get(uint(idx)) {
					// Valid row: return cached value for convergence check
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
				return scm.NewNil() // invalid row
			}
			// Invalid row → on-demand incremental recompute
			p.shard.t.orcMu.Lock()
			if !p.validMask.Get(uint(idx)) {
				p.shard.t.incrementalRecomputeORC(p.colName, p.shard, idx)
			}
			p.shard.t.orcMu.Unlock()
		}
		// Valid: return from delta or main
		p.mu.RLock()
		if val, ok := p.delta[idx]; ok {
			p.mu.RUnlock()
			return val
		}
		p.mu.RUnlock()
		if idx < p.count && p.main != nil {
			return p.main.GetValue(idx)
		}
		return scm.NewNil()
	}

	// Delta entries shadow main storage and are also used for rows appended
	// after the proxy's materialized main storage was built.
	p.mu.RLock()
	if val, ok := p.delta[idx]; ok {
		p.mu.RUnlock()
		return val
	}
	p.mu.RUnlock()

	// Fast path 1: fully compressed → value is in main storage for main rows.
	if p.compressed && idx < p.count && p.main != nil {
		return p.main.GetValue(idx)
	}

	// Fast path 2: valid bit set → value is cached in main storage for main rows.
	if p.validMask.Get(uint(idx)) && idx < p.count && p.main != nil {
		return p.main.GetValue(idx)
	}

	// Slow path: compute on demand
	colvalues := make([]scm.Scmer, len(p.inputCols))
	for i, col := range p.inputCols {
		// Delta rows must be read via the shard-level ColumnReader; direct
		// ColumnStorage access only understands main-row indexes.
		colvalues[i] = p.shard.ColumnReader(col)(idx)
	}
	val := scm.Apply(p.computor, colvalues...)

	p.mu.Lock()
	p.delta[idx] = val
	p.mu.Unlock()
	p.validMask.Set(uint(idx), true)

	return val
}

func (p *StorageComputeProxy) GetCachedReader() ColumnReader {
	return p.GetCachedReaderTx(CurrentTx())
}

// Compress materializes all values into a compressed main storage.
func (p *StorageComputeProxy) Compress() {
	tx := CurrentTx()
	if variant := p.currentVariant(tx, true); variant != nil {
		p.compressVariant(variant, tx)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// Early exit if already compressed and no dirty delta
	if p.compressed && len(p.delta) == 0 {
		return
	}

	// Empty shard: nothing to compute, avoid accessing input column storages
	if p.count == 0 {
		p.compressed = true
		return
	}

	fn := scm.OptimizeProcToSerialFunction(p.computor)
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = newCachedColumnReaderTx(p.shard.getColumnStorageOrPanic(col), tx)
	}

	colvalues := make([]scm.Scmer, len(p.inputCols))
	getValue := func(idx uint32) scm.Scmer {
		if val, ok := p.delta[idx]; ok {
			return val
		}
		if p.main != nil && p.validMask.Get(uint(idx)) {
			return p.main.GetValue(idx)
		}
		// compute
		for j := range readers {
			colvalues[j] = readers[j].GetValue(idx)
		}
		result := fn(colvalues...)
		return result
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
	tx := CurrentTx()
	if variant := p.currentVariant(tx, true); variant != nil {
		p.compressFilteredVariant(variant, tx, filterCols, filter)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// Empty shard: nothing to compute
	if p.count == 0 {
		return
	}

	fn := scm.OptimizeProcToSerialFunction(p.computor)
	filterFn := scm.OptimizeProcToSerialFunction(filter)

	filterReaders := make([]ColumnReader, len(filterCols))
	for i, col := range filterCols {
		filterReaders[i] = newCachedColumnReaderTx(p.shard.getColumnStorageOrPanic(col), tx)
	}
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = newCachedColumnReaderTx(p.shard.getColumnStorageOrPanic(col), tx)
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
			p.validMask.Set(uint(i), true)
		}
	}
	// Don't set compressed=true → unmatched rows stay lazy for on-demand GetValue
}

// Invalidate marks a single row as needing recomputation.
func (p *StorageComputeProxy) Invalidate(idx uint32) {
	if p.hasSessionVariants() {
		p.forEachVariant(func(v *storageComputeVariant) {
			v.mu.Lock()
			defer v.mu.Unlock()
			if v.compressed {
				if scmer, ok := v.main.(*StorageSCMER); ok {
					colvalues := make([]scm.Scmer, len(p.inputCols))
					for i, col := range p.inputCols {
						colvalues[i] = p.shard.ColumnReaderTx(col, CurrentTx())(idx)
					}
					val := applyWithTx(CurrentTx(), p.computor, colvalues...)
					scmer.SetValue(idx, val)
					return
				}
				v.compressed = false
			}
			v.validMask.Set(uint(idx), false)
			delete(v.delta, idx)
		})
		return
	}
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
			val := applyWithTx(CurrentTx(), p.computor, colvalues...)
			scmer.SetValue(idx, val)
			return // stay compressed, no bitmap change needed
		}
		// main is compressed type → can't SetValue → fall back to lazy
		p.compressed = false
	}
	p.validMask.Set(uint(idx), false)
	delete(p.delta, idx)
}

// IncrementalUpdate adds delta to the cached value at idx.
// If the row is not valid (not yet computed), this is a no-op (the next read will compute fresh).
// This avoids shard rebuilds by modifying the cached value in-place.
func (p *StorageComputeProxy) IncrementalUpdate(idx uint32, delta scm.Scmer) {
	if p.hasSessionVariants() {
		p.forEachVariant(func(v *storageComputeVariant) {
			v.mu.Lock()
			defer v.mu.Unlock()
			if !v.compressed && !v.validMask.Get(uint(idx)) {
				return
			}
			var oldVal scm.Scmer
			if val, ok := v.delta[idx]; ok {
				oldVal = val
			} else if v.main != nil {
				oldVal = v.main.GetValue(idx)
			} else {
				return
			}
			var newVal scm.Scmer
			if oldVal.IsInt() && delta.IsInt() {
				newVal = scm.NewInt(oldVal.Int() + delta.Int())
			} else if oldVal.IsNil() || delta.IsNil() {
				newVal = scm.NewNil()
			} else {
				newVal = scm.NewFloat(oldVal.Float() + delta.Float())
			}
			v.delta[idx] = newVal
			if v.compressed {
				v.compressed = false
				for i := uint32(0); i < v.count; i++ {
					v.validMask.Set(uint(i), true)
				}
			}
		})
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.compressed && !p.validMask.Get(uint(idx)) {
		return // not valid → will be computed fresh on next read
	}
	var oldVal scm.Scmer
	if v, ok := p.delta[idx]; ok {
		oldVal = v
	} else if p.main != nil {
		oldVal = p.main.GetValue(idx)
	} else {
		return
	}
	// Add oldVal + delta using Go arithmetic (avoids Scheme runtime overhead)
	var newVal scm.Scmer
	if oldVal.IsInt() && delta.IsInt() {
		newVal = scm.NewInt(oldVal.Int() + delta.Int())
	} else if oldVal.IsNil() || delta.IsNil() {
		newVal = scm.NewNil()
	} else {
		newVal = scm.NewFloat(oldVal.Float() + delta.Float())
	}
	p.delta[idx] = newVal
	if p.compressed {
		p.compressed = false
		// All rows were valid while compressed (values in main). Now that we're
		// non-compressed, mark all rows as valid so IncrementalUpdate works for
		// other indices too. Values not in delta will fall through to main.
		for i := uint32(0); i < p.count; i++ {
			p.validMask.Set(uint(i), true)
		}
	}
}

// SetValue writes val directly to the cached value at idx, bypassing recomputation.
// If the shard is compressed and main is a StorageSCMER, the value is written in-place.
// Otherwise the value is written to the delta map.
func (p *StorageComputeProxy) SetValue(idx uint32, val scm.Scmer) {
	if p.hasSessionVariants() {
		p.forEachVariant(func(v *storageComputeVariant) {
			v.mu.Lock()
			defer v.mu.Unlock()
			if v.compressed && v.main != nil {
				if scmer, ok := v.main.(*StorageSCMER); ok {
					scmer.SetValue(idx, val)
					return
				}
				v.compressed = false
				for i := uint32(0); i < v.count; i++ {
					v.validMask.Set(uint(i), true)
				}
			}
			v.delta[idx] = val
			v.validMask.Set(uint(idx), true)
		})
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.compressed && p.main != nil {
		if scmer, ok := p.main.(*StorageSCMER); ok {
			scmer.SetValue(idx, val)
			return
		}
		// main is a compressed type → fall back to delta; mark all rows valid
		// so that GetValue falls through to main for rows not in delta.
		p.compressed = false
		for i := uint32(0); i < p.count; i++ {
			p.validMask.Set(uint(i), true)
		}
	}
	p.delta[idx] = val
	p.validMask.Set(uint(idx), true)
}

// InvalidateAll marks all rows as needing recomputation (resets validMask).
func (p *StorageComputeProxy) InvalidateAll() {
	if p.hasSessionVariants() {
		p.forEachVariant(func(v *storageComputeVariant) {
			v.mu.Lock()
			defer v.mu.Unlock()
			v.compressed = false
			v.validMask.Reset()
			v.delta = make(map[uint32]scm.Scmer)
		})
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.compressed = false
	p.validMask.Reset()
	p.delta = make(map[uint32]scm.Scmer)
}

// ShouldSkipSelectiveInvalidation returns true when cumulative invalidation
// cost exceeds the last recompute cost. The caller should then skip the
// selective invalidation (the column is already dirty enough for a full
// recompute on the next read).
func (p *StorageComputeProxy) ShouldSkipSelectiveInvalidation() bool {
	invNs := p.invalidateNsSinceRead.Load()
	recompNs := p.lastRecomputeNs.Load()
	if recompNs == 0 {
		return false // no baseline yet, allow selective
	}
	return invNs > recompNs
}

// AddInvalidationCost adds nanoseconds to the invalidation telemetry counter.
func (p *StorageComputeProxy) AddInvalidationCost(ns int64) {
	p.invalidateNsSinceRead.Add(ns)
}

// ResetInvalidationTelemetry resets the invalidation counter (called after read/recompute).
func (p *StorageComputeProxy) ResetInvalidationTelemetry(recomputeNs int64) {
	p.invalidateNsSinceRead.Store(0)
	if recomputeNs > 0 {
		p.lastRecomputeNs.Store(recomputeNs)
	}
}

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
// storageComputeProxyVersion is the current binary format version for StorageComputeProxy.
// Increment this constant and add a new deserializeComputeProxyV* helper whenever the
// layout after the magic byte changes. Never delete old helpers.
const storageComputeProxyVersion = 2

// StorageComputeProxy binary layout (magic byte 50 consumed by shard loader):
//
//	[version uint8]         ← first byte read by Deserialize
//	[count uint32]
//	[numCols uint16]
//	[inputCols: numCols × (uint16 length + bytes)]
//	[computorLen uint32]
//	[computorJSON: computorLen bytes]
//	[compressed uint8]      ← 1=compressed, 0=delta only
//	[hasMain uint8]         ← 1=has main storage, 0=delta map
//	  if hasMain: [magic uint8 + full serialized main storage]
//	  else:       [deltaLen uint32] [delta: deltaLen × (uint32 idx + uint32 valLen + JSON bytes)]
//	[validCount uint32]     ← number of set bits in validMask
//	[validMask: validCount × uint32 indices]
//
// Version history:
//
//	0: layout as above.
//	1: adds [isOrdered uint8] after validMask, but validMask indices were written
//	   via binary.Write(uint), so persisted data may miss the bitmap payload.
//	2: writes validMask indices as uint32 and keeps the trailing isOrdered byte.
func (p *StorageComputeProxy) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(50))                         // magic byte
	binary.Write(f, binary.LittleEndian, uint8(storageComputeProxyVersion)) // version byte
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
		p.main.Serialize(f)                            // nested — includes its own magic byte
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
	p.validMask.Iterate(func(idx uint) {
		binary.Write(f, binary.LittleEndian, uint32(idx))
	})

	// v1: isOrdered flag
	var isOrderedByte uint8
	if p.isOrdered {
		isOrderedByte = 1
	}
	binary.Write(f, binary.LittleEndian, isOrderedByte)
}

// Deserialize reads the proxy from the given reader.
// Note: magic byte 50 is already consumed by the caller.
func (p *StorageComputeProxy) Deserialize(f io.Reader) uint {
	var version uint8
	binary.Read(f, binary.LittleEndian, &version)
	switch version {
	case 0:
		return p.deserializeComputeProxyV0(f)
	case 1:
		return p.deserializeComputeProxyV1(f)
	case 2:
		return p.deserializeComputeProxyV2(f)
	default:
		panic(fmt.Sprintf("StorageComputeProxy: unknown version %d", version))
	}
}

func (p *StorageComputeProxy) restoreValidMaskFromPayload() {
	p.validMask = NonLockingReadMap.NonBlockingBitMap{}
	if p.compressed && p.main != nil {
		for idx := uint32(0); idx < p.count; idx++ {
			p.validMask.Set(uint(idx), true)
		}
		return
	}
	for idx := range p.delta {
		p.validMask.Set(uint(idx), true)
	}
}

func (p *StorageComputeProxy) readValidMaskV2(f io.Reader) error {
	p.validMask = NonLockingReadMap.NonBlockingBitMap{}
	var validCount uint32
	if err := binary.Read(f, binary.LittleEndian, &validCount); err != nil {
		return err
	}
	for i := uint32(0); i < validCount; i++ {
		var idx uint32
		if err := binary.Read(f, binary.LittleEndian, &idx); err != nil {
			return err
		}
		p.validMask.Set(uint(idx), true)
	}
	return nil
}

func (p *StorageComputeProxy) deserializeComputeProxyV0(f io.Reader) uint {
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

	if p.delta == nil {
		p.delta = make(map[uint32]scm.Scmer)
	}
	if err := p.readValidMaskV2(f); err != nil {
		// Legacy proxy files before v2 wrote the bitmap payload incorrectly.
		// Reconstruct validity from the persisted value payload instead of
		// treating every row as invalid after reload.
		p.restoreValidMaskFromPayload()
	}

	// shard/column runtime bindings are restored by the post-load hook in
	// ensureColumnLoaded.
	return uint(p.count)
}

func (p *StorageComputeProxy) deserializeComputeProxyV1(f io.Reader) uint {
	n := p.deserializeComputeProxyV0(f)
	// v1 adds isOrdered byte after validMask
	var isOrderedByte uint8
	binary.Read(f, binary.LittleEndian, &isOrderedByte)
	p.isOrdered = isOrderedByte != 0
	return n
}

func (p *StorageComputeProxy) deserializeComputeProxyV2(f io.Reader) uint {
	n := p.deserializeComputeProxyV0(f)
	var isOrderedByte uint8
	binary.Read(f, binary.LittleEndian, &isOrderedByte)
	p.isOrdered = isOrderedByte != 0
	return n
}
