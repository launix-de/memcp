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

type computeProxyReader struct {
	proxy   *StorageComputeProxy
	readers []ColumnReader
	tx      *TxContext
	values  []scm.Scmer
}

type orderedComputeProxyReader struct {
	proxy *StorageComputeProxy
	tx    *TxContext
}

func (r *orderedComputeProxyReader) GetValue(idx uint32) scm.Scmer {
	return r.proxy.getValueTx(r.tx, idx)
}

func (r *orderedComputeProxyReader) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	for i := uint32(0); i < count; i++ {
		target[int(i)*stride] = r.proxy.getValueTx(r.tx, recid+i)
	}
}

func (r *orderedComputeProxyReader) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	for i, recid := range recids {
		target[i*stride] = r.proxy.getValueTx(r.tx, recid)
	}
}

func applyWithTx(tx *TxContext, fn scm.Scmer, args ...scm.Scmer) scm.Scmer {
	if fn.IsProc() {
		proc := *fn.Proc()
		runtimeSession := txSessionScmer(tx)
		// Physical computed-column lambdas can outlive the query which created
		// them. Rebind their explicit execution parameters to the current consumer
		// without inserting an environment level: optimized nested closures address
		// captured numbered variables by their exact lexical depth.
		physicalTx := scm.NewNil()
		if tx != nil {
			physicalTx = scm.NewAny(tx)
		}
		proc.En = bindExecutionEnv(proc.En, runtimeSession, physicalTx)
		fn = scm.NewProcStruct(proc)
	}
	return scm.Apply(fn, args...)
}

// StorageComputeProxy is a complete logical column with lazy physical values.
// Creating the proxy installs the column definition first; it does not require
// every row value to be materialized before the column can be read.
//
// Materialization contract:
//   - Compress eagerly prepares every row when all values are expected to be
//     consumed.
//   - CompressFiltered only prewarms rows selected by its filter because the
//     caller expects to consume those rows soon. The filter is a preparation
//     hint, not part of the column definition or its value semantics.
//   - GetValue repairs an ordinary missing or invalidated value pointwise and
//     caches it. A row omitted by prewarming is neither NULL nor an error.
//   - An ordered reduction column cannot be repaired pointwise. Its GetValue
//     path recomputes the ordered dependency range required for the requested
//     value, which may include a suffix or the whole column.
//   - Invalidation marks cached values unusable; later reads repair the affected
//     dependency closure without changing the column's logical domain.
//
// Keep this distinction intact when changing planner setup, DDL, rebuild, or
// scan code: selective preparation may change work scheduling, never which
// values the StorageComputeProxy represents.
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
	// Invalidation telemetry compares measured selective maintenance with the
	// last complete or suffix recompute. It is an operator-local runtime gate:
	// exact row invalidation is retained when it is cheaper, while broad fanout
	// can fall back to InvalidateAll instead of serially recomputing most rows.
	invalidateNsSinceRead atomic.Int64  // cumulative invalidation nanoseconds since last read
	lastRecomputeNs       atomic.Int64  // nanoseconds of the last full/suffix recompute
	revision              atomic.Uint64 // logical value changes; index readers use this for lazy invalidation
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

// GetValueRange and GetValueMulti mirror StorageComputeProxy's own bulk
// fast path: delegate straight to v.main in one call when every row is
// already valid and materialized there, otherwise fall back to this
// reader's own GetValue per row (full delta/compute repair logic).
func (r *computeVariantReader) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	v := r.variant
	if v.compressed && v.main != nil && uint64(recid)+uint64(count) <= uint64(v.count) {
		v.mu.RLock()
		deltaEmpty := len(v.delta) == 0
		v.mu.RUnlock()
		if deltaEmpty {
			v.main.GetValueRange(recid, count, target, stride)
			return
		}
	}
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = r.GetValue(recid + k)
		idx += stride
	}
}

func (r *computeVariantReader) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if len(recids) == 0 {
		return
	}
	v := r.variant
	if v.compressed && v.main != nil {
		allMain := true
		for _, recid := range recids {
			if recid >= v.count {
				allMain = false
				break
			}
		}
		if allMain {
			v.mu.RLock()
			deltaEmpty := len(v.delta) == 0
			v.mu.RUnlock()
			if deltaEmpty {
				v.main.GetValueMulti(recids, target, stride)
				return
			}
		}
	}
	idx := 0
	for _, recid := range recids {
		target[idx] = r.GetValue(recid)
		idx += stride
	}
}

func (r *computeProxyReader) GetValue(idx uint32) scm.Scmer {
	p := r.proxy

	p.mu.RLock()
	if val, ok := p.delta[idx]; ok {
		p.mu.RUnlock()
		return val
	}
	p.mu.RUnlock()

	if p.compressed && idx < p.count && p.main != nil {
		return p.main.GetValue(idx)
	}
	if p.validMask.Get(uint(idx)) && idx < p.count && p.main != nil {
		return p.main.GetValue(idx)
	}

	for i := range r.readers {
		r.values[i] = r.readers[i].GetValue(idx)
	}
	val := applyWithTx(r.tx, p.computor, r.values...)

	p.mu.Lock()
	p.delta[idx] = val
	p.mu.Unlock()
	p.validMask.Set(uint(idx), true)
	return val
}

// GetValueRange and GetValueMulti mirror computeVariantReader's bulk fast
// path against p (the proxy) instead of a session variant.
func (r *computeProxyReader) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	p := r.proxy
	if p.compressed && p.main != nil && uint64(recid)+uint64(count) <= uint64(p.count) {
		p.mu.RLock()
		deltaEmpty := len(p.delta) == 0
		p.mu.RUnlock()
		if deltaEmpty {
			p.main.GetValueRange(recid, count, target, stride)
			return
		}
	}
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = r.GetValue(recid + k)
		idx += stride
	}
}

func (r *computeProxyReader) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if len(recids) == 0 {
		return
	}
	p := r.proxy
	if p.compressed && p.main != nil {
		allMain := true
		for _, recid := range recids {
			if recid >= p.count {
				allMain = false
				break
			}
		}
		if allMain {
			p.mu.RLock()
			deltaEmpty := len(p.delta) == 0
			p.mu.RUnlock()
			if deltaEmpty {
				p.main.GetValueMulti(recids, target, stride)
				return
			}
		}
	}
	idx := 0
	for _, recid := range recids {
		target[idx] = r.GetValue(recid)
		idx += stride
	}
}

func (p *StorageComputeProxy) GetCachedReaderTx(tx *TxContext) ColumnReader {
	if p.isOrdered {
		return &orderedComputeProxyReader{proxy: p, tx: tx}
	}
	// Bind input readers before the physical scan acquires the shard read lock.
	// A cache miss may compute a value, but it must stay within the already
	// acquired shard capability instead of re-entering GetRead/ColumnReaderTx.
	// This also keeps the reader executable on a future remote shard owner.
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}
	variant := p.currentVariant(tx, true)
	if variant == nil {
		return &computeProxyReader{
			proxy:   p,
			readers: readers,
			tx:      tx,
			values:  make([]scm.Scmer, len(readers)),
		}
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

func (p *StorageComputeProxy) visibleDeltaRecids() []uint32 {
	p.shard.mu.RLock()
	recids := make([]uint32, 0, len(p.shard.inserts))
	for recid := p.shard.main_count; recid < p.shard.main_count+uint32(len(p.shard.inserts)); recid++ {
		if !p.shard.deletions.Get(uint(recid)) {
			recids = append(recids, recid)
		}
	}
	p.shard.mu.RUnlock()
	return recids
}

func (p *StorageComputeProxy) needsUnfilteredPreparation() bool {
	recids := p.visibleDeltaRecids()
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.compressed {
		return true
	}
	for _, recid := range recids {
		if _, present := p.delta[recid]; !present {
			return true
		}
	}
	return false
}

func (p *StorageComputeProxy) prewarmDeltaRows(tx *TxContext, filterCols []string, filter scm.Scmer, onlyMissing bool) {
	filterReaders := make([]ColumnReader, len(filterCols))
	for i, col := range filterCols {
		filterReaders[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}
	inputReaders := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		inputReaders[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}

	recids := p.visibleDeltaRecids()
	if onlyMissing {
		p.mu.RLock()
		missing := recids[:0]
		for _, recid := range recids {
			if _, present := p.delta[recid]; !present {
				missing = append(missing, recid)
			}
		}
		p.mu.RUnlock()
		recids = missing
	}
	prepared := make(map[uint32]scm.Scmer)
	filterValues := make([]scm.Scmer, len(filterReaders))
	inputValues := make([]scm.Scmer, len(inputReaders))
	for _, recid := range recids {
		if !filter.IsNil() {
			for i, reader := range filterReaders {
				filterValues[i] = reader.GetValue(recid)
			}
			if !scm.ToBool(applyWithTx(tx, filter, filterValues...)) {
				continue
			}
		}
		for i, reader := range inputReaders {
			inputValues[i] = reader.GetValue(recid)
		}
		prepared[recid] = applyWithTx(tx, p.computor, inputValues...)
	}
	p.mu.Lock()
	for recid, value := range prepared {
		if onlyMissing {
			if _, present := p.delta[recid]; present {
				continue
			}
		}
		p.delta[recid] = value
		p.validMask.Set(uint(recid), true)
	}
	p.mu.Unlock()
}

func (p *StorageComputeProxy) prewarmVariantDeltaRows(v *storageComputeVariant, tx *TxContext, filterCols []string, filter scm.Scmer, onlyMissing bool) {
	filterReaders := make([]ColumnReader, len(filterCols))
	for i, col := range filterCols {
		filterReaders[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}
	inputReaders := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		inputReaders[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}

	recids := p.visibleDeltaRecids()
	if onlyMissing {
		v.mu.RLock()
		missing := recids[:0]
		for _, recid := range recids {
			if _, present := v.delta[recid]; !present {
				missing = append(missing, recid)
			}
		}
		v.mu.RUnlock()
		recids = missing
	}
	prepared := make(map[uint32]scm.Scmer)
	filterValues := make([]scm.Scmer, len(filterReaders))
	inputValues := make([]scm.Scmer, len(inputReaders))
	for _, recid := range recids {
		if !filter.IsNil() {
			for i, reader := range filterReaders {
				filterValues[i] = reader.GetValue(recid)
			}
			if !scm.ToBool(applyWithTx(tx, filter, filterValues...)) {
				continue
			}
		}
		for i, reader := range inputReaders {
			inputValues[i] = reader.GetValue(recid)
		}
		prepared[recid] = applyWithTx(tx, p.computor, inputValues...)
	}
	v.mu.Lock()
	for recid, value := range prepared {
		if onlyMissing {
			if _, present := v.delta[recid]; present {
				continue
			}
		}
		v.delta[recid] = value
		v.validMask.Set(uint(recid), true)
	}
	v.mu.Unlock()
}

func (p *StorageComputeProxy) compressVariant(v *storageComputeVariant, tx *TxContext) {
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}
	func() {
		v.mu.Lock()
		defer v.mu.Unlock()
		if v.compressed {
			return
		}
		if v.count == 0 {
			v.compressed = true
			return
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
		for recid := range v.delta {
			if recid < v.count {
				delete(v.delta, recid)
			}
		}
		v.validMask.Reset()
		v.compressed = true
	}()
	p.prewarmVariantDeltaRows(v, tx, nil, scm.NewNil(), true)
}

func (p *StorageComputeProxy) compressFilteredVariant(v *storageComputeVariant, tx *TxContext, filterCols []string, filter scm.Scmer) {
	filterProgram := scm.PrepareSerialProc(filter)
	filterReaders := make([]ColumnReader, len(filterCols))
	for i, col := range filterCols {
		filterReaders[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = ColumnReaderFunc(p.shard.ColumnReaderTx(tx, col))
	}

	func() {
		v.mu.Lock()
		defer v.mu.Unlock()
		filterValues := make([]scm.Scmer, len(filterCols))
		colvalues := make([]scm.Scmer, len(p.inputCols))
		for i := uint32(0); i < v.count; i++ {
			for j := range filterReaders {
				filterValues[j] = filterReaders[j].GetValue(i)
			}
			if scm.ToBool(filterProgram.Call(filterValues)) {
				for j := range readers {
					colvalues[j] = readers[j].GetValue(i)
				}
				v.delta[i] = applyWithTx(tx, p.computor, colvalues...)
				v.validMask.Set(uint(i), true)
			}
		}
	}()
	p.prewarmVariantDeltaRows(v, tx, filterCols, filter, false)
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
	p.mu.RLock()
	sz += uint(len(p.delta)) * 24 // rough estimate per map entry
	if p.main != nil {
		sz += p.main.ComputeSize()
	}
	p.mu.RUnlock()

	// Session-bound variants are owned by this proxy as well. Snapshot the map
	// under variantsMu, then measure each variant under its own lock so size
	// accounting never races lazy materialization or invalidation.
	p.variantsMu.RLock()
	variants := make([]*storageComputeVariant, 0, len(p.variants))
	for _, variant := range p.variants {
		variants = append(variants, variant)
	}
	p.variantsMu.RUnlock()
	for _, variant := range variants {
		variant.mu.RLock()
		sz += 96 + variant.validMask.ComputeSize()
		sz += uint(len(variant.delta)) * 24
		if variant.main != nil {
			sz += variant.main.ComputeSize()
		}
		variant.mu.RUnlock()
	}
	return sz
}

// GetValue returns the logical value at idx. For an ordinary proxy, a cache miss
// or invalidated value computes only this row from the current inputs and caches
// it. For an ordered reduction column, the branch below repairs the required
// ordered dependency range instead. Neither kind may treat an unprepared row as
// absent or expose a transient invalid-cache sentinel as its logical value.
func (p *StorageComputeProxy) GetValue(idx uint32) scm.Scmer {
	return p.getValueTx(nil, idx)
}

func (p *StorageComputeProxy) getValueTx(tx *TxContext, idx uint32) scm.Scmer {
	// ORC path: validity tracked per-row via validMask.
	if p.isOrdered {
		if !p.validMask.Get(uint(idx)) {
			// Invalid row → on-demand incremental recompute (or wait for the
			// ongoing one to complete).
			p.shard.t.orcMu.Lock()
			if !p.validMask.Get(uint(idx)) {
				p.shard.t.incrementalRecomputeORC(p.colName, p.shard, idx, tx)
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
		colvalues[i] = p.shard.ColumnReaderTx(tx, col)(idx)
	}
	val := applyWithTx(tx, p.computor, colvalues...)

	p.mu.Lock()
	p.delta[idx] = val
	p.mu.Unlock()
	p.validMask.Set(uint(idx), true)

	return val
}

// storedORCValue returns the currently published ORC value without triggering
// repair. Recompute scans request it through the explicit $orc_stored pseudo
// column; ordinary readers continue to wait for or initiate repair in getValueTx.
func (p *StorageComputeProxy) storedORCValue(idx uint32) scm.Scmer {
	if !p.validMask.Get(uint(idx)) {
		return scm.NewNil()
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if val, ok := p.delta[idx]; ok {
		return val
	}
	if idx < p.count && p.main != nil {
		return p.main.GetValue(idx)
	}
	return scm.NewNil()
}

// getValueRLocked evaluates an ordinary computed column while the caller owns
// the shard read lock. It must not re-enter shard.mu: Go's writer-preferring
// RWMutex would deadlock if a writer queued between the two read acquisitions.
func (p *StorageComputeProxy) getValueRLocked(tx *TxContext, idx uint32) scm.Scmer {
	if variant := p.currentVariant(tx, true); variant != nil {
		variant.mu.RLock()
		if value, present := variant.delta[idx]; present {
			variant.mu.RUnlock()
			return value
		}
		main := variant.main
		count := variant.count
		compressed := variant.compressed
		variant.mu.RUnlock()
		if idx < count && main != nil && (compressed || variant.validMask.Get(uint(idx))) {
			return main.GetValue(idx)
		}

		values := make([]scm.Scmer, len(p.inputCols))
		for i, col := range p.inputCols {
			cs := p.shard.getColumnStorageRLocked(col)
			if dependency, ok := cs.(*StorageComputeProxy); ok && !dependency.isOrdered {
				values[i] = dependency.getValueRLocked(tx, idx)
				continue
			}
			if idx < p.shard.main_count {
				values[i] = newCachedColumnReaderTx(cs, tx).GetValue(idx)
				continue
			}
			deltaIndex := int(idx - p.shard.main_count)
			if columnIndex, ok := p.shard.deltaColumns[col]; ok && deltaIndex < len(p.shard.inserts) && columnIndex < len(p.shard.inserts[deltaIndex]) {
				values[i] = p.shard.inserts[deltaIndex][columnIndex]
			} else {
				values[i] = scm.NewNil()
			}
		}
		value := applyWithTx(tx, p.computor, values...)
		variant.mu.Lock()
		variant.delta[idx] = value
		variant.mu.Unlock()
		variant.validMask.Set(uint(idx), true)
		return value
	}

	p.mu.RLock()
	if value, present := p.delta[idx]; present {
		p.mu.RUnlock()
		return value
	}
	main := p.main
	count := p.count
	compressed := p.compressed
	p.mu.RUnlock()
	if idx < count && main != nil && (compressed || p.validMask.Get(uint(idx))) {
		return main.GetValue(idx)
	}

	values := make([]scm.Scmer, len(p.inputCols))
	for i, col := range p.inputCols {
		cs := p.shard.getColumnStorageRLocked(col)
		if dependency, ok := cs.(*StorageComputeProxy); ok && !dependency.isOrdered {
			values[i] = dependency.getValueRLocked(tx, idx)
			continue
		}
		if idx < p.shard.main_count {
			values[i] = newCachedColumnReaderTx(cs, tx).GetValue(idx)
			continue
		}
		deltaIndex := int(idx - p.shard.main_count)
		if columnIndex, ok := p.shard.deltaColumns[col]; ok && deltaIndex < len(p.shard.inserts) && columnIndex < len(p.shard.inserts[deltaIndex]) {
			values[i] = p.shard.inserts[deltaIndex][columnIndex]
		} else {
			values[i] = scm.NewNil()
		}
	}
	value := applyWithTx(tx, p.computor, values...)
	p.mu.Lock()
	p.delta[idx] = value
	p.mu.Unlock()
	p.validMask.Set(uint(idx), true)
	return value
}

// GetValueRange and GetValueMulti take the fast path — one bulk call
// straight into p.main — only when every requested row is guaranteed to
// come from main with no repair work: not an ORC column, no pending delta
// overrides, and the proxy is fully compressed. That mirrors GetValue's own
// "fast path 1" above. Any other case (ORC, live delta entries, rows beyond
// the compressed main, or a row still needing on-demand compute) falls back
// to the existing per-row GetValue, which already contains the full
// invalidation/session/delta repair logic; duplicating that logic here for
// the sake of batching would risk subtly diverging from it.
func (p *StorageComputeProxy) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	if !p.isOrdered && p.compressed && p.main != nil && uint64(recid)+uint64(count) <= uint64(p.count) {
		p.mu.RLock()
		deltaEmpty := len(p.delta) == 0
		p.mu.RUnlock()
		if deltaEmpty {
			p.main.GetValueRange(recid, count, target, stride)
			return
		}
	}
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = p.GetValue(recid + k)
		idx += stride
	}
}

func (p *StorageComputeProxy) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if len(recids) == 0 {
		return
	}
	if !p.isOrdered && p.compressed && p.main != nil {
		allMain := true
		for _, recid := range recids {
			if recid >= p.count {
				allMain = false
				break
			}
		}
		if allMain {
			p.mu.RLock()
			deltaEmpty := len(p.delta) == 0
			p.mu.RUnlock()
			if deltaEmpty {
				p.main.GetValueMulti(recids, target, stride)
				return
			}
		}
	}
	idx := 0
	for _, recid := range recids {
		target[idx] = p.GetValue(recid)
		idx += stride
	}
}

func (p *StorageComputeProxy) GetCachedReader() ColumnReader {
	return p.GetCachedReaderTx(nil)
}

// Compress materializes all values into a compressed main storage.
func (p *StorageComputeProxy) Compress(tx *TxContext) {
	compressStart := time.Now()
	compressedNow := false
	if variant := p.currentVariant(tx, true); variant != nil {
		p.compressVariant(variant, tx)
		return
	}
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = newCachedColumnReaderTx(p.shard.getColumnStorageOrPanic(col, false, tx), tx)
	}
	func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.compressed {
			return
		}
		if p.count == 0 {
			p.compressed = true
			compressedNow = true
			return
		}

		colvalues := make([]scm.Scmer, len(p.inputCols))
		getValue := func(idx uint32) scm.Scmer {
			if val, ok := p.delta[idx]; ok {
				return val
			}
			if p.main != nil && p.validMask.Get(uint(idx)) {
				return p.main.GetValue(idx)
			}
			for j := range readers {
				colvalues[j] = readers[j].GetValue(idx)
			}
			return applyWithTx(tx, p.computor, colvalues...)
		}

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
		for recid := range p.delta {
			if recid < p.count {
				delete(p.delta, recid)
			}
		}
		p.validMask.Reset()
		p.compressed = true
		compressedNow = true
	}()
	p.prewarmDeltaRows(tx, nil, scm.NewNil(), true)
	if compressedNow {
		p.ResetInvalidationTelemetry(time.Since(compressStart).Nanoseconds())
	}
}

// CompressFiltered prewarms an ordinary computed column only for rows matching
// filter because the caller has signalled that those values will likely be read
// immediately. The filter does not narrow the logical column and is not a
// read-time predicate: unmatched rows remain valid lazy values and GetValue
// materializes each one pointwise on first read. Ordered reduction columns have
// dependency-aware preparation and repair paths instead.
func (p *StorageComputeProxy) CompressFiltered(tx *TxContext, filterCols []string, filter scm.Scmer) {
	if variant := p.currentVariant(tx, true); variant != nil {
		p.compressFilteredVariant(variant, tx, filterCols, filter)
		return
	}
	filterReaders := make([]ColumnReader, len(filterCols))
	for i, col := range filterCols {
		filterReaders[i] = newCachedColumnReaderTx(p.shard.getColumnStorageOrPanic(col, false, tx), tx)
	}
	readers := make([]ColumnReader, len(p.inputCols))
	for i, col := range p.inputCols {
		readers[i] = newCachedColumnReaderTx(p.shard.getColumnStorageOrPanic(col, false, tx), tx)
	}

	func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		filterValues := make([]scm.Scmer, len(filterCols))
		colvalues := make([]scm.Scmer, len(p.inputCols))
		for i := uint32(0); i < p.count; i++ {
			for j := range filterReaders {
				filterValues[j] = filterReaders[j].GetValue(i)
			}
			if scm.ToBool(applyWithTx(tx, filter, filterValues...)) {
				for j := range readers {
					colvalues[j] = readers[j].GetValue(i)
				}
				p.delta[i] = applyWithTx(tx, p.computor, colvalues...)
				p.validMask.Set(uint(i), true)
			}
		}
	}()
	p.prewarmDeltaRows(tx, filterCols, filter, false)
	// Don't set compressed=true → unmatched rows stay lazy for on-demand GetValue
}

// Invalidate marks a single row as needing recomputation.
func (p *StorageComputeProxy) Invalidate(idx uint32) {
	p.InvalidateTx(nil, idx)
}

// InvalidateTx marks a single row stale and uses tx for any immediate repair.
func (p *StorageComputeProxy) InvalidateTx(tx *TxContext, idx uint32) {
	p.revision.Add(1)
	if p.hasSessionVariants() {
		p.forEachVariant(func(v *storageComputeVariant) {
			v.mu.Lock()
			defer v.mu.Unlock()
			if v.compressed {
				if scmer, ok := v.main.(*StorageSCMER); ok {
					if idx >= v.count {
						v.validMask.Set(uint(idx), false)
						delete(v.delta, idx)
						return
					}
					colvalues := make([]scm.Scmer, len(p.inputCols))
					for i, col := range p.inputCols {
						colvalues[i] = p.shard.ColumnReaderTx(tx, col)(idx)
					}
					val := applyWithTx(tx, p.computor, colvalues...)
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
			if idx >= p.count {
				p.validMask.Set(uint(idx), false)
				delete(p.delta, idx)
				return
			}
			// recompute single value and write directly
			colvalues := make([]scm.Scmer, len(p.inputCols))
			for i, col := range p.inputCols {
				colvalues[i] = p.shard.getColumnStorageOrPanic(col, false, tx).GetValue(idx)
			}
			val := applyWithTx(tx, p.computor, colvalues...)
			scmer.SetValue(idx, val)
			return // stay compressed, no bitmap change needed
		}
		// Compressed immutable storages cannot update in place. Keep the compact
		// base column and install one sparse override instead of switching the
		// complete proxy back to lazy mode: validMask is intentionally empty after
		// Compress(), so that transition would make every unrelated row look stale.
		if idx < p.count {
			colvalues := make([]scm.Scmer, len(p.inputCols))
			for i, col := range p.inputCols {
				colvalues[i] = p.shard.getColumnStorageOrPanic(col, false, tx).GetValue(idx)
			}
			p.delta[idx] = applyWithTx(tx, p.computor, colvalues...)
			return
		}
	}
	p.validMask.Set(uint(idx), false)
	delete(p.delta, idx)
}

// InvalidateRows chooses between exact point repair and complete lazy
// invalidation from measurements of this proxy's own computor. Dominating
// cases remain unconditional: small batches are repaired directly. For a
// broad batch, a short measured sample is extrapolated and compared with the
// last complete materialization. This avoids both a global cost constant and
// a schema-specific heuristic; the same physical operator adapts to cheap and
// expensive lookup bodies and to changing table cardinalities.
func (p *StorageComputeProxy) InvalidateRows(recids map[uint32]struct{}) {
	p.InvalidateRowsTx(nil, recids)
}

// InvalidateRowsTx invalidates a measured row subset with explicit tx-bound repairs.
func (p *StorageComputeProxy) InvalidateRowsTx(tx *TxContext, recids map[uint32]struct{}) {
	if len(recids) == 0 {
		return
	}
	const sampleRows = 32
	if len(recids) <= sampleRows || p.hasSessionVariants() {
		for recid := range recids {
			p.InvalidateTx(tx, recid)
		}
		return
	}

	fullRecomputeNs := p.lastRecomputeNs.Load()
	if fullRecomputeNs <= 0 {
		for recid := range recids {
			p.InvalidateTx(tx, recid)
		}
		return
	}

	ids := make([]uint32, 0, len(recids))
	for recid := range recids {
		ids = append(ids, recid)
	}
	started := time.Now()
	for _, recid := range ids[:sampleRows] {
		p.InvalidateTx(tx, recid)
	}
	sampleNs := time.Since(started).Nanoseconds()
	estimatedNs := float64(sampleNs) * float64(len(ids)) / float64(sampleRows)
	if estimatedNs >= float64(fullRecomputeNs) {
		p.InvalidateAll()
		return
	}
	for _, recid := range ids[sampleRows:] {
		p.InvalidateTx(tx, recid)
	}
}

// IncrementalUpdate adds delta to the cached value at idx.
// If the row is not valid yet, materialize it from the current post-mutation
// state instead of silently dropping the update. This keeps FK-reuse and other
// incremental aggregate caches correct across empty<->non-empty transitions.
func (p *StorageComputeProxy) IncrementalUpdate(idx uint32, delta scm.Scmer) {
	p.IncrementalUpdateTx(nil, idx, delta)
}

// IncrementalUpdateTx updates one cached value using an explicit tx for lazy repair.
func (p *StorageComputeProxy) IncrementalUpdateTx(tx *TxContext, idx uint32, delta scm.Scmer) {
	p.revision.Add(1)
	if p.hasSessionVariants() {
		p.forEachVariant(func(v *storageComputeVariant) {
			v.mu.Lock()
			if !v.compressed && !v.validMask.Get(uint(idx)) {
				v.mu.Unlock()
				p.getValueTx(tx, idx)
				return
			}
			var oldVal scm.Scmer
			if val, ok := v.delta[idx]; ok {
				oldVal = val
			} else if idx < v.count && v.main != nil {
				oldVal = v.main.GetValue(idx)
			} else {
				v.validMask.Set(uint(idx), false)
				v.mu.Unlock()
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
			v.mu.Unlock()
		})
		return
	}
	p.mu.Lock()
	if !p.compressed && !p.validMask.Get(uint(idx)) {
		p.mu.Unlock()
		// Recompute the affected row from the already-mutated source state so the
		// cache converges immediately even when this row had never been
		// materialized before.
		p.getValueTx(tx, idx)
		return
	}
	var oldVal scm.Scmer
	if v, ok := p.delta[idx]; ok {
		oldVal = v
	} else if idx < p.count && p.main != nil {
		oldVal = p.main.GetValue(idx)
	} else {
		p.mu.Unlock()
		// The row was appended after main storage was compressed. Its source
		// state already contains this trigger update, so materialize that state
		// once instead of applying delta a second time.
		p.getValueTx(tx, idx)
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
	p.mu.Unlock()
}

// SetValue writes val directly to the cached value at idx, bypassing recomputation.
// If the shard is compressed and main is a StorageSCMER, the value is written in-place.
// Otherwise the value is written to the delta map.
func (p *StorageComputeProxy) SetValue(idx uint32, val scm.Scmer) {
	p.revision.Add(1)
	if p.hasSessionVariants() {
		p.forEachVariant(func(v *storageComputeVariant) {
			v.mu.Lock()
			defer v.mu.Unlock()
			if v.compressed && v.main != nil {
				if scmer, ok := v.main.(*StorageSCMER); ok && idx < v.count {
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
		if scmer, ok := p.main.(*StorageSCMER); ok && idx < p.count {
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
	p.revision.Add(1)
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

func (s *StorageComputeProxy) DistinctCount() uint {
	if s.main == nil {
		return 0
	}
	return s.main.DistinctCount()
}

// JITEmit preserves lazy compute semantics by calling the ordinary reader.
func (s *StorageComputeProxy) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	var idxInt scm.JITValueDesc
	if idx.Loc == scm.LocImm {
		idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
	} else if idx.Loc == scm.LocRegPair {
		ctx.FreeReg(idx.Reg)
		idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
		ctx.BindReg(idx.Reg2, &idxInt)
	} else {
		idxInt = idx
	}
	if idxInt.Loc == scm.LocImm {
		idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.EnsureDesc(&idxInt)
		if idxInt.Loc != scm.LocReg {
			panic("jit: idxInt not in register")
		}
		ctx.EmitShlRegImm8(idxInt.Reg, 32)
		ctx.EmitShrRegImm8(idxInt.Reg, 32)
		ctx.BindReg(idxInt.Reg, &idxInt)
	}
	ctx.EnsureDesc(&thisptr)
	ctx.EnsureDesc(&thisptr)
	if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
		panic("jit: generic call arg expects 1-word value")
	}
	d0 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
	if d0.Loc == scm.LocRegPair || d0.Loc == scm.LocStackPair || d0.Loc == scm.LocRegTriple || d0.Loc == scm.LocStackTriple {
		panic("jit: generic call arg expects 1-word value")
	}
	ctx.EnsureDesc(&idxInt)
	ctx.EnsureDesc(&idxInt)
	if idxInt.Loc == scm.LocRegPair || idxInt.Loc == scm.LocStackPair || idxInt.Loc == scm.LocRegTriple || idxInt.Loc == scm.LocStackTriple {
		panic("jit: generic call arg expects 1-word value")
	}
	ctx.SyncDesc(&thisptr)
	ctx.SyncDesc(&d0)
	ctx.SyncDesc(&idxInt)
	d1 := ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageComputeProxy).getValueTx), []scm.JITValueDesc{thisptr, d0, idxInt}, 2)
	d1.NoHeapPointer = false
	ctx.BindReg(d1.Reg, &d1)
	ctx.BindReg(d1.Reg2, &d1)
	ctx.FreeDesc(&d0)
	ctx.FreeDesc(&idxInt)
	if d1.Loc == scm.LocImm {
		if result.Loc == scm.LocAny {
			return d1
		}
	}
	if result.Loc == scm.LocAny {
		result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.BindReg(result.Reg, &result)
		ctx.BindReg(result.Reg2, &result)
	}
	ctx.SyncDesc(&d1)
	if d1.Loc == scm.LocRegPair || d1.Loc == scm.LocStackPair || d1.Loc == scm.LocInputPair {
		ctx.EmitMovPairToResult(&d1, &result)
		result.Type = d1.Type
	} else {
		switch d1.Type {
		case scm.TagBool:
			ctx.EmitMakeBool(result, d1)
			result.Type = scm.TagBool
		case scm.TagInt:
			ctx.EmitMakeInt(result, d1)
			result.Type = scm.TagInt
		case scm.TagFloat:
			ctx.EmitMakeFloat(result, d1)
			result.Type = scm.TagFloat
		case scm.TagNil:
			ctx.EmitMakeNil(result)
			result.Type = scm.TagNil
		default:
			panic("jit: single-block scalar return with unknown type")
		}
	}
	return result
	return result
}
