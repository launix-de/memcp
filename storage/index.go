/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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

import "fmt"
import "math/bits"
import "sort"
import "sync"
import "time"
import "strings"
import "sync/atomic"

import "github.com/google/btree"
import "github.com/carli2/hybridsort"
import "github.com/launix-de/memcp/scm"

type indexPair struct {
	itemid      int // -1 for reference items
	data        []scm.Scmer
	reference   *scanIndexBounds
	compareCols int // reference-only prefix length; zero compares the complete key
}

type storageIndexState struct {
	mainIndexes StorageInt
	// mainIndexPositions is the compressed inverse permutation of mainIndexes:
	// record ID -> position in this index. It is built only when a sparse RecSet
	// repeatedly dominates an ordered scan and belongs to the same cache/variant
	// lifecycle as the forward permutation.
	mainIndexPositions StorageInt
	deltaBtree         *btree.BTreeG[indexPair]
	active             bool
	savings            float64
	minVals            []scm.Scmer
	maxVals            []scm.Scmer
	indexHooks         []IndexHook
	indexHookBytes     atomic.Int64
	precomputedDelta   bool
	computedRevisions  []computedRevision
}

type indexIterationOptions struct {
	orderedLimit         int
	boundaryCoveredLimit bool
}

type computedRevision struct {
	proxy    *StorageComputeProxy
	revision uint64
}

// (no op) numeric helper removed; collations now use golang.org/x/text/collate for ordering

// colGetter retrieves a single index-column value for a given record ID.
// For raw columns it reads directly from ColumnStorage; for computed columns
// it evaluates the mapFn over the source column storages.
type colGetter struct {
	raw     ColumnReader                 // non-nil for raw columns
	mapCols []ColumnReader               // non-nil for computed columns
	mapFn   func(...scm.Scmer) scm.Scmer // non-nil for computed columns
}

const inlineIndexGetters = 8

type indexGetterScratch struct {
	getters     [inlineIndexGetters]colGetter
	indexBounds scanIndexBounds
}

var indexGetterScratchPool = sync.Pool{
	New: func() any { return new(indexGetterScratch) },
}

func (g colGetter) get(recid uint32) scm.Scmer {
	if g.mapFn != nil {
		vals := make([]scm.Scmer, len(g.mapCols))
		for i, cs := range g.mapCols {
			vals[i] = cs.GetValue(recid)
		}
		return g.mapFn(vals...)
	}
	return g.raw.GetValue(recid)
}

type StorageIndex struct {
	Cols []string // sort equal-cols alphabetically, so similar conditions are canonical
	// ColMapCols[i] and ColMapFn[i] are set for computed index columns (col starts with ".").
	// Both are nil for raw columns.
	ColMapCols [][]string  // per-column source col names; nil entry means raw column
	ColMapFn   []scm.Scmer // per-column compute fn; IsNil() entry means raw column
	// ColMatchers stores only custom, non-sorted analyzers. A nil entry (or an
	// absent slice) denotes an ordinary sorted column; equal/range belong to the
	// query boundary and are deliberately not persisted as index metadata.
	ColMatchers []IndexAnalyzer
	// ColOrder is the immutable per-column strict relation used by build, delta
	// merge, lookup, and ordered scans. Each callback owns collation, direction,
	// and NULL placement; storage must not infer or wrap those semantics.
	ColOrder []func(scm.Scmer, scm.Scmer) bool
	// ColOrderMeta identifies the source relation for equality and reuse checks.
	ColOrderMeta []string
	Savings      float64 // store the amount of time savings here -> add selectivity (outputted / size) on each
	Native       bool    // true when data is physically sorted by this index (zero-cost)
	t            *storageShard
	lastHit      atomic.Uint32 // last search position for sorted access pattern optimization
	mu           sync.Mutex
	baseState    storageIndexState
}

func orderRelationMeta(order func(...scm.Scmer) scm.Scmer) string {
	if collation, reverse, ok := scm.LookupCollate(order); ok {
		if reverse {
			return collation + ":desc"
		}
		return collation + ":asc"
	}
	if order == nil {
		return ""
	}
	return fmt.Sprintf("callback:%x", scm.FunctionIdentity(order))
}

func canonicalColumnOrder(t *table, col string) (func(scm.Scmer, scm.Scmer) bool, string) {
	collation := "bin"
	for _, definition := range t.Columns {
		if definition.Name == col {
			if definition.Collation != "" {
				collation = definition.Collation
			}
			break
		}
	}
	value := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString(collation), scm.NewBool(false))
	order := scm.OptimizeProcToSerialFunction(value)
	return scm.OrderRelationLess(order), orderRelationMeta(order)
}

func boundaryOrder(t *table, boundary columnboundaries) (func(scm.Scmer, scm.Scmer) bool, string) {
	if boundary.order != nil {
		meta := boundary.orderMeta
		if meta == "" {
			meta = orderRelationMeta(boundary.order)
		}
		return scm.OrderRelationLess(boundary.order), meta
	}
	// Materialized group keys currently carry "any" column metadata. Preserve
	// the EqualSQL order encoded by the compiled predicate there. Base-table
	// columns retain their declared canonical order; their residual callback is
	// still authoritative for SQL equality semantics.
	if boundary.collation != "" && strings.HasPrefix(t.Name, ".grp:") &&
		(boundary.lower.IsString() || boundary.lower.IsSymbol()) {
		value := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString(boundary.collation), scm.NewBool(false))
		order := scm.OptimizeProcToSerialFunction(value)
		return scm.OrderRelationLess(order), orderRelationMeta(order)
	}
	return canonicalColumnOrder(t, boundary.col)
}

func ascendingOrderMetaMatches(meta, collation string) bool {
	return len(meta) == len(collation)+len(":asc") &&
		meta[:len(collation)] == collation && meta[len(collation):] == ":asc"
}

func indexOrderMatchesBoundary(t *table, index *StorageIndex, column int, boundary columnboundaries) bool {
	if column >= len(index.ColOrderMeta) {
		return false
	}
	meta := index.ColOrderMeta[column]
	if boundary.order != nil {
		required := boundary.orderMeta
		if required == "" {
			required = orderRelationMeta(boundary.order)
		}
		return meta == required
	}
	if boundary.collation != "" && strings.HasPrefix(t.Name, ".grp:") &&
		(boundary.lower.IsString() || boundary.lower.IsSymbol()) {
		return ascendingOrderMetaMatches(meta, boundary.collation)
	}
	collation := "bin"
	for _, definition := range t.Columns {
		if definition.Name == boundary.col {
			if definition.Collation != "" {
				collation = definition.Collation
			}
			break
		}
	}
	return ascendingOrderMetaMatches(meta, collation)
}

func (s *StorageIndex) lessAt(col int, a, b scm.Scmer) bool {
	if col < len(s.ColOrder) && s.ColOrder[col] != nil {
		return s.ColOrder[col](a, b)
	}
	return scm.Less(a, b)
}

func (s *StorageIndex) compareAt(col int, a, b scm.Scmer) int {
	if s.lessAt(col, a, b) {
		return -1
	}
	if s.lessAt(col, b, a) {
		return 1
	}
	return 0
}

func (s *StorageIndex) usesNaturalAscendingOrder(col int) bool {
	if col >= len(s.ColOrder) || s.ColOrder[col] == nil {
		return true
	}
	if col >= len(s.ColOrderMeta) {
		return false
	}
	meta := s.ColOrderMeta[col]
	if !strings.HasSuffix(meta, ":asc") {
		return false
	}
	switch strings.TrimSuffix(meta, ":asc") {
	case "bin", "binary", "utf8", "utf8mb4":
		return true
	default:
		return false
	}
}

// buildGetters returns per-column value getters for this index, reading from the
// shard under a currently-held RLock. Must be called with s.t.mu.RLock held.
func (s *StorageIndex) buildGetters(_ *TxContext, storage []colGetter) []colGetter {
	var getters []colGetter
	if cap(storage) >= len(s.Cols) {
		getters = storage[:len(s.Cols)]
		clear(getters)
	} else {
		getters = make([]colGetter, len(s.Cols))
	}
	for i, col := range s.Cols {
		if !s.columnIsSorted(i) && isScanPseudoColName(col) {
			continue
		}
		if len(s.ColMapFn) > i && !s.ColMapFn[i].IsNil() {
			// computed column: read mapCols and apply mapFn
			mapColReaders := make([]ColumnReader, len(s.ColMapCols[i]))
			for j, mc := range s.ColMapCols[i] {
				if isScanPseudoColName(mc) {
					mapColReaders[j] = ColumnReaderFunc(func(uint32) scm.Scmer { return scm.NewNil() })
					continue
				}
				cs := s.t.getColumnStorageRLocked(mc)
				mapColReaders[j] = newCachedColumnReaderTx(cs, nil)
			}
			mapFn := s.ColMapFn[i]
			fn := scm.OptimizeProcToSerialFunction(mapFn)
			getters[i] = colGetter{mapCols: mapColReaders, mapFn: fn}
		} else {
			cs := s.t.getColumnStorageRLocked(col)
			getters[i] = colGetter{raw: newCachedColumnReaderTx(cs, nil)}
		}
	}
	return getters
}

// computedRevisionsRLocked snapshots the logical generations of computed
// columns used as index keys. The caller already owns the shard read lock.
func (s *StorageIndex) computedRevisionsRLocked() []computedRevision {
	result := make([]computedRevision, 0)
	add := func(col string) {
		if isScanPseudoColName(col) {
			return
		}
		proxy, ok := s.t.getColumnStorageRLocked(col).(*StorageComputeProxy)
		if !ok {
			return
		}
		for _, existing := range result {
			if existing.proxy == proxy {
				return
			}
		}
		result = append(result, computedRevision{proxy: proxy, revision: proxy.revision.Load()})
	}
	for i, col := range s.Cols {
		if len(s.ColMapFn) > i && !s.ColMapFn[i].IsNil() {
			for _, mapCol := range s.ColMapCols[i] {
				add(mapCol)
			}
			continue
		}
		add(col)
	}
	return result
}

func sameComputedRevisions(a, b []computedRevision) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// matcherKindEqual checks if two matchers have the same kind.
func matcherKindEqual(a, b IndexAnalyzer) bool {
	return a.Kind() == b.Kind()
}

// indexMatcherCompatible reports whether an index column can serve a query
// boundary. Equal and range boundaries both use the same sorted index data;
// only custom row matchers require an exact analyzer kind.
func indexMatcherCompatible(query, indexed IndexAnalyzer) bool {
	if query == nil || query.IsSorted() {
		return indexed == nil || indexed.IsSorted()
	}
	return indexed != nil && matcherKindEqual(query, indexed)
}

func (idx *StorageIndex) columnIsSorted(i int) bool {
	return len(idx.ColMatchers) <= i || idx.ColMatchers[i] == nil || idx.ColMatchers[i].IsSorted()
}

func (idx *StorageIndex) addSavings(state *storageIndexState, usageWeight float64) float64 {
	if usageWeight > 0 {
		idx.Savings += usageWeight
	}
	return idx.Savings
}

func (idx *StorageIndex) ComputeSize() uint {
	var sz uint = 24 * 8 // heuristic
	for _, state := range []*storageIndexState{&idx.baseState} {
		if !idx.Native {
			sz += state.mainIndexes.ComputeSize()
		}
		if state.mainIndexPositions.count > 0 {
			sz += state.mainIndexPositions.ComputeSize()
		}
		sz += uint(state.indexHookBytes.Load())
		sz += idx.computeDeltaBtreeSize(state)
	}
	return sz
}

func (idx *StorageIndex) evictionOffer(currentSize int64) evictionOffer {
	return evictionOffer{fullBytes: currentSize}
}

func (idx *StorageIndex) evict(mode evictionMode, currentSize int64, _ *[numEvictableTypes]int64) evictionResult {
	if !idx.mu.TryLock() {
		return evictionResult{}
	}
	defer idx.mu.Unlock()
	if mode == evictPartial {
		return evictionResult{}
	}
	idx.baseState = storageIndexState{}
	return evictionResult{freedBytes: currentSize, fullyEvicted: true, success: true}
}

func (idx *StorageIndex) computeDeltaBtreeSize(state *storageIndexState) uint {
	if state == nil || state.deltaBtree == nil {
		return 0
	}
	// The B-tree owns nodes and indexPair values. For normal delta rows,
	// indexPair.data points at shard inserts that are already counted by the
	// shard; count only the slice header there. A future precomputed payload
	// would have to be accounted separately.
	var sz uint = 64 + uint(state.deltaBtree.Len())*64
	state.deltaBtree.Ascend(func(item indexPair) bool {
		if state.precomputedDelta {
			sz += scm.ComputeSize(scm.NewAny(item.data))
		} else {
			sz += 24
		}
		return true
	})
	return sz
}

// String describes the physical index. Equal/range are query properties, not
// distinct index kinds, so only custom analyzers are included in the output.
func (idx *StorageIndex) String() string {
	var b strings.Builder
	for i, col := range idx.Cols {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(col)
		if len(idx.ColMatchers) > i && idx.ColMatchers[i] != nil && !idx.ColMatchers[i].IsSorted() {
			b.WriteByte('(')
			b.WriteString(idx.ColMatchers[i].Kind())
			b.WriteByte(')')
		}
	}
	return b.String()
}

// getDeltaValue returns the raw column value for a delta row.
func (s *StorageIndex) getDeltaValue(data []scm.Scmer, col string) scm.Scmer {
	colpos, ok := s.t.deltaColumns[col]
	if ok && colpos < len(data) {
		return data[colpos]
	}
	return scm.NewNil()
}

// getDeltaColValue returns the index-column value for a delta row at column index colIdx.
// For computed columns it reads the source cols and applies the mapFn.
func (s *StorageIndex) getDeltaColValue(recid uint32, data []scm.Scmer, colIdx int) scm.Scmer {
	return s.getDeltaColValueTx(nil, recid, data, colIdx)
}

func (s *StorageIndex) getDeltaColValueTx(tx *TxContext, recid uint32, data []scm.Scmer, colIdx int) scm.Scmer {
	if len(s.ColMapFn) > colIdx && !s.ColMapFn[colIdx].IsNil() {
		fn := scm.OptimizeProcToSerialFunction(s.ColMapFn[colIdx])
		vals := make([]scm.Scmer, len(s.ColMapCols[colIdx]))
		for i, mc := range s.ColMapCols[colIdx] {
			if isScanPseudoColName(mc) {
				vals[i] = scm.NewNil()
				continue
			}
			cs := s.t.getColumnStorageRLocked(mc)
			if proxy, ok := cs.(*StorageComputeProxy); ok {
				if !proxy.isOrdered {
					vals[i] = proxy.getValueRLocked(tx, recid)
				} else {
					vals[i] = s.t.ColumnReaderTx(tx, mc)(recid)
				}
			} else {
				vals[i] = s.getDeltaValue(data, mc)
			}
		}
		return fn(vals...)
	}
	cs := s.t.getColumnStorageRLocked(s.Cols[colIdx])
	if proxy, ok := cs.(*StorageComputeProxy); ok {
		if !proxy.isOrdered {
			return proxy.getValueRLocked(tx, recid)
		}
		return s.t.ColumnReaderTx(tx, s.Cols[colIdx])(recid)
	}
	return s.getDeltaValue(data, s.Cols[colIdx])
}

// rowWithinBounds checks sorted (equal/range) columns using lower/upper directly.
// Non-sorted columns (LIKE etc.) are skipped — handled by block-level skipping
// in iterate(); the scan layer applies the full condition afterwards.
func (s *StorageIndex) boundKernel(bounds scanAccess, cmpCols int) (firstSorted int, lastSorted int, sortedMask uint64, unboundedMask uint64) {
	firstSorted, lastSorted = -1, -1
	for i := 0; i < cmpCols; i++ {
		if !s.columnIsSorted(i) {
			continue
		}
		if firstSorted < 0 {
			firstSorted = i
		}
		lastSorted = i
		if i < 64 {
			sortedMask |= uint64(1) << i
			bound := bounds.boundary(i)
			if boundaryIsUnboundedOrder(bound) || boundaryIsUnboundedRange(bound) {
				unboundedMask |= uint64(1) << i
			}
		}
	}
	return
}

func (s *StorageIndex) rowWithinBounds(bounds scanAccess, indexBounds scanIndexBounds, cmpCols int, lastSorted int, sortedMask uint64, unboundedMask uint64, lowerInclusive bool, upperInclusive bool, getter func(int) scm.Scmer) (inRange bool, beyond bool) {
	for i := 0; i < cmpCols; i++ {
		if i < 64 && sortedMask&(uint64(1)<<i) == 0 {
			continue // non-sorted: block-skip handles this, scan() filters exact
		}
		if i >= 64 && !s.columnIsSorted(i) {
			continue
		}
		v := getter(i)
		if i < 64 {
			if unboundedMask&(uint64(1)<<i) != 0 {
				continue
			}
		} else if i < bounds.len() {
			bound := bounds.boundary(i)
			if boundaryIsUnboundedOrder(bound) || boundaryIsUnboundedRange(bound) {
				continue // ordering or residual-only range, not an index restriction
			}
		}
		if i == lastSorted {
			upperLast := indexBounds.upperLast()
			if !upperLast.IsNil() {
				if upperInclusive {
					if s.compareAt(i, upperLast, v) < 0 {
						return false, true
					}
				} else if s.compareAt(i, v, upperLast) >= 0 {
					return false, true
				}
			}
			if indexBounds.len() > i && !indexBounds.lower(i).IsNil() {
				comparison := s.compareAt(i, v, indexBounds.lower(i))
				if comparison < 0 || (comparison == 0 && !lowerInclusive) {
					return false, false
				}
			}
			continue
		}
		cmp := s.compareAt(i, v, indexBounds.lower(i))
		if cmp == 0 {
			continue
		}
		if cmp < 0 {
			return false, false
		}
		return false, true
	}
	return true, false
}

// queryIndexPrefixLen returns the part of the query boundary list which is
// physically represented by this index. Query-local hooks are orthogonal to
// the stored index key: in particular, a trailing $recset_contains boundary
// must never be interpreted as the next column of a longer reusable index.
// Matching the column and matcher kind here preserves prefix reuse without
// coupling an invocation-only matcher to an unrelated physical suffix.
func (s *StorageIndex) queryIndexPrefixLen(bounds scanAccess, indexBounds scanIndexBounds) int {
	limit := indexBounds.len()
	if limit > bounds.len() {
		limit = bounds.len()
	}
	if limit > len(s.Cols) {
		limit = len(s.Cols)
	}
	for i := 0; i < limit; i++ {
		bound := bounds.boundary(i)
		if bound.col != s.Cols[i] {
			return i
		}
		var indexedMatcher IndexAnalyzer
		if i < len(s.ColMatchers) {
			indexedMatcher = s.ColMatchers[i]
		}
		if !indexMatcherCompatible(bound.matcher, indexedMatcher) {
			return i
		}
	}
	return limit
}

func (s *StorageIndex) compareMainAndDelta(state *storageIndexState, mainRecid uint32, mainGetters []colGetter, delta indexPair) int {
	for i := range s.Cols {
		if !s.columnIsSorted(i) {
			continue
		}
		mainVal := mainGetters[i].get(mainRecid)
		deltaVal := delta.data[i]
		if !state.precomputedDelta {
			deltaVal = s.getDeltaColValue(uint32(delta.itemid), delta.data, i)
		}
		if s.lessAt(i, mainVal, deltaVal) {
			return -1
		}
		if s.lessAt(i, deltaVal, mainVal) {
			return 1
		}
	}
	deltaRecid := uint32(delta.itemid)
	if mainRecid < deltaRecid {
		return -1
	}
	if mainRecid > deltaRecid {
		return 1
	}
	return 0
}

// iterates over items using a caller-provided buffer for batching.
// The callback receives batches of record IDs and returns false to stop iteration.
// Buffer size controls early-out granularity: use small buffers (e.g. [8]uint32)
// for existence checks, large buffers (e.g. [1024]uint32) for full scans.
func (t *storageShard) iterateIndex(tx *TxContext, cols scanAccess, maxInsertIndex int, buf []uint32, usageWeight float64, selected func(*StorageIndex, bool), callback func([]uint32) bool) {
	t.iterateIndexEx(tx, cols, maxInsertIndex, buf, usageWeight, false, nil, nil, nil, selected, callback)
}

// iterateIndexEstimate additionally reports the complete candidate range found
// by an active sorted index. Delta rows are included conservatively because
// they are merged after the main range and may still satisfy the predicate.
func (t *storageShard) iterateIndexEstimate(tx *TxContext, cols scanAccess, maxInsertIndex int, buf []uint32, candidateSpan *int64, selected func(*StorageIndex, bool), callback func([]uint32) bool) {
	t.iterateIndexEx(tx, cols, maxInsertIndex, buf, 0, false, nil, nil, candidateSpan, selected, callback)
}

func (t *storageShard) iterateIndexOrdered(tx *TxContext, cols scanAccess, maxInsertIndex int, buf []uint32, usageWeight float64, limit int, boundaryCoveredLimit bool, selected func(*StorageIndex, bool), callback func([]uint32) bool) {
	t.iterateIndexEx(tx, cols, maxInsertIndex, buf, usageWeight, false, nil, &indexIterationOptions{orderedLimit: limit, boundaryCoveredLimit: boundaryCoveredLimit}, nil, selected, callback)
}

func (t *storageShard) iterateIndexForce(tx *TxContext, cols scanAccess, maxInsertIndex int, buf []uint32, countUsage bool, callback func([]uint32) bool) {
	usageWeight := 0.0
	if countUsage {
		usageWeight = 1.0
	}
	t.iterateIndexEx(tx, cols, maxInsertIndex, buf, usageWeight, true, nil, nil, nil, nil, callback)
}

func (t *storageShard) iterateIndexMatchAware(tx *TxContext, cols scanAccess, maxInsertIndex int, buf []uint32, countUsage bool, exactMain *bool, callback func([]uint32) bool) {
	usageWeight := 0.0
	if countUsage {
		usageWeight = 1.0
	}
	t.iterateIndexEx(tx, cols, maxInsertIndex, buf, usageWeight, false, exactMain, nil, nil, nil, callback)
}

func effectiveBoundaryInclusiveness(cols scanAccess, indexBounds scanIndexBounds) (bool, bool) {
	if indexBounds.len() == 0 {
		return true, true
	}
	for i := indexBounds.len() - 1; i >= 0; i-- {
		bound := cols.boundary(i)
		if bound.matcher.IsSorted() {
			return bound.lowerInclusive, bound.upperInclusive
		}
	}
	return true, true
}

func (t *storageShard) iterateIndexEx(tx *TxContext, cols scanAccess, maxInsertIndex int, buf []uint32, usageWeight float64, forceBuild bool, exactMain *bool, options *indexIterationOptions, candidateSpan *int64, selected func(*StorageIndex, bool), callback func([]uint32) bool) {
	indexBounds := newScanIndexBounds(cols)
	if exactMain != nil {
		*exactMain = false
	}
	// cols is already sorted by 1st rank: equality before range; 2nd rank alphabet
	// A complete unique point prefix yields at most one live row. Building or
	// retaining a full-column candidate hook cannot narrow that driver further;
	// leave the suffix to the mandatory residual predicate.
	if t.t.hasBoundUniquePoint(cols) {
		sortedEnd := 0
		for sortedEnd < cols.len() && cols.boundary(sortedEnd).matcher.IsSorted() {
			sortedEnd++
		}
		if sortedEnd < cols.len() && indexBounds.len() > sortedEnd {
			indexBounds = indexBounds.truncate(sortedEnd)
		}
	}

	// The index view may shorten access when more than one range column is
	// present because an ordered index can use only one range suffix. Read the
	// flags from that effective last boundary, not from a later condition that
	// is evaluated only by the scan predicate.
	lowerIncl, upperIncl := effectiveBoundaryInclusiveness(cols, indexBounds)
	// Exact RecSet membership is a query-bound overlay. A prepared base index
	// needs to cover only the sorted/access prefix; RecSetMatcher binds to that
	// index at invocation time and must not create one index identity per source
	// carrier. Retain a lone RecSet boundary so unordered scans can still use the
	// common matcher machinery without a zero-column StorageIndex.
	indexCols := indexBounds.len()
	for indexCols > 1 && indexCols <= cols.len() && matcherKindEqual(cols.boundary(indexCols-1).matcher, RecSetMatcher) {
		indexCols--
	}

	// check if we found conditions
	if indexBounds.len() > 0 {
		// find an index that has at least the columns in that order we're searching for
		// if the index is inactive, use the other one
	retry_indexscan:
		old_indexes := t.Indexes
		for _, index := range old_indexes {
			// naive index search algo; TODO: improve
			if len(index.Cols) >= indexCols {
				for i := 0; i < indexCols; i++ {
					bound := cols.boundary(i)
					if bound.col != index.Cols[i] {
						goto skip_index // column mismatch
					}
					var indexedMatcher IndexAnalyzer
					if len(index.ColMatchers) > i {
						indexedMatcher = index.ColMatchers[i]
					}
					if !indexMatcherCompatible(bound.matcher, indexedMatcher) {
						goto skip_index // matcher kind mismatch
					}
					if bound.matcher.IsSorted() {
						if !indexOrderMatchesBoundary(t.t, index, i, bound) {
							goto skip_index
						}
					}
				}
				// this index fits!
				index.iterate(tx, cols, indexBounds, lowerIncl, upperIncl, maxInsertIndex, buf, usageWeight, forceBuild, exactMain, options, candidateSpan, selected, callback)
				return
			}
		skip_index:
		}

		// otherwise: create new index (but first check for prefix coverage)
		t.indexMutex.Lock()
		if len(old_indexes) != len(t.Indexes) {
			t.indexMutex.Unlock()
			goto retry_indexscan // someone has added a index in the meantime: recheck
		}
		// check if an existing longer index already covers these columns as a prefix
		for _, index := range t.Indexes {
			if len(index.Cols) >= indexCols {
				covered := true
				for i := 0; i < indexCols; i++ {
					bound := cols.boundary(i)
					if bound.col != index.Cols[i] {
						covered = false
						break
					}
					var indexedMatcher IndexAnalyzer
					if len(index.ColMatchers) > i {
						indexedMatcher = index.ColMatchers[i]
					}
					if !indexMatcherCompatible(bound.matcher, indexedMatcher) {
						covered = false
						break
					}
					if bound.matcher.IsSorted() {
						if !indexOrderMatchesBoundary(t.t, index, i, bound) {
							covered = false
							break
						}
					}
				}
				if covered {
					// longer index covers this query; use it instead of creating a shorter one
					t.indexMutex.Unlock()
					index.iterate(tx, cols, indexBounds, lowerIncl, upperIncl, maxInsertIndex, buf, usageWeight, forceBuild, exactMain, options, candidateSpan, selected, callback)
					return
				}
			}
		}
		index := new(StorageIndex)
		index.Cols = make([]string, indexCols)
		index.ColMapCols = make([][]string, indexCols)
		index.ColMapFn = make([]scm.Scmer, indexCols)
		index.ColOrder = make([]func(scm.Scmer, scm.Scmer) bool, indexCols)
		index.ColOrderMeta = make([]string, indexCols)
		for i := 0; i < indexCols; i++ {
			bound := cols.boundary(i)
			index.Cols[i] = bound.col
			index.ColMapCols[i] = bound.mapCols // nil for raw columns
			index.ColMapFn[i] = bound.mapFn     // IsNil() for raw columns
			if bound.matcher.IsSorted() {
				index.ColOrder[i], index.ColOrderMeta[i] = boundaryOrder(t.t, bound)
			} else {
				if index.ColMatchers == nil {
					index.ColMatchers = make([]IndexAnalyzer, indexCols)
				}
				index.ColMatchers[i] = bound.matcher
			}
		}
		index.Savings = 0.0            // count how many cost we wasted so we decide when to build the index
		index.baseState.active = false // tell the engine that index has to be built first
		index.t = t
		index.Native = true
		for i := range index.Cols {
			if index.columnIsSorted(i) {
				index.Native = false
				break
			}
		}
		t.Indexes = append(t.Indexes, index)
		t.indexMutex.Unlock()
		index.iterate(tx, cols, indexBounds, lowerIncl, upperIncl, maxInsertIndex, buf, usageWeight, forceBuild, exactMain, options, candidateSpan, selected, callback)
		return
	}

	// otherwise: iterate over all items in batches
	if selected != nil {
		selected(nil, false)
	}
	bufN := 0
	for i := uint32(0); i < t.main_count; i++ {
		buf[bufN] = i
		bufN++
		if bufN == len(buf) {
			if !callback(buf[:bufN]) {
				return
			}
			bufN = 0
		}
	}
	for i := 0; i < maxInsertIndex; i++ {
		buf[bufN] = t.main_count + uint32(i)
		bufN++
		if bufN == len(buf) {
			if !callback(buf[:bufN]) {
				return
			}
			bufN = 0
		}
	}
	if bufN > 0 {
		callback(buf[:bufN])
	}
}

// indexHasComputedCol returns true if any of the index's columns is a computed column.
func indexHasComputedCol(s *storageShard, idx *StorageIndex) bool {
	for _, col := range idx.Cols {
		for _, c := range s.t.Columns {
			if c.Name == col && len(c.ComputorInputCols) > 0 {
				return true
			}
		}
	}
	// also check new-style computed index columns
	for i := range idx.Cols {
		if len(idx.ColMapFn) > i && !idx.ColMapFn[i].IsNil() {
			return true
		}
	}
	return false
}

// snapshotIndexesForRebuild copies the mutable index metadata while the caller
// holds the source shard lock. Rebuild can then rank the candidates without
// retaining or reacquiring the source lock.
func snapshotIndexesForRebuild(indexes []*StorageIndex) []*StorageIndex {
	if len(indexes) == 0 {
		return nil
	}
	candidates := make([]*StorageIndex, 0, len(indexes))
	for _, idx := range indexes {
		clone := new(StorageIndex)
		clone.Cols = append([]string(nil), idx.Cols...)
		clone.ColMapCols = idx.ColMapCols // shallow copy OK (immutable per-col slices)
		clone.ColMapFn = idx.ColMapFn     // shallow copy OK
		for i, matcher := range idx.ColMatchers {
			if matcher == nil || matcher.IsSorted() {
				continue
			}
			if clone.ColMatchers == nil {
				clone.ColMatchers = make([]IndexAnalyzer, len(idx.ColMatchers))
			}
			clone.ColMatchers[i] = matcher
		}
		clone.ColOrder = append([]func(scm.Scmer, scm.Scmer) bool(nil), idx.ColOrder...)
		clone.ColOrderMeta = append([]string(nil), idx.ColOrderMeta...)
		clone.Savings = idx.Savings * 0.9
		clone.baseState.active = false
		candidates = append(candidates, clone)
	}
	return candidates
}

func rebuildIndexes(candidates []*StorageIndex, t2 *storageShard) {
	if len(candidates) == 0 {
		return
	}

	// 1. Attach the snapshotted metadata to the new shard.
	for _, candidate := range candidates {
		candidate.t = t2
	}

	// 2. Prefix dedup: sort by len(Cols) descending so longer indexes absorb shorter ones
	hybridsort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i].Cols) > len(candidates[j].Cols)
	})
	removed := make([]bool, len(candidates))
	for i, longer := range candidates {
		if removed[i] {
			continue
		}
		for j := i + 1; j < len(candidates); j++ {
			if removed[j] {
				continue
			}
			shorter := candidates[j]
			if len(shorter.Cols) > len(longer.Cols) {
				continue
			}
			isPrefix := true
			for k := 0; k < len(shorter.Cols); k++ {
				if shorter.Cols[k] != longer.Cols[k] {
					isPrefix = false
					break
				}
				var shorterMatcher, longerMatcher IndexAnalyzer
				if len(shorter.ColMatchers) > k {
					shorterMatcher = shorter.ColMatchers[k]
				}
				if len(longer.ColMatchers) > k {
					longerMatcher = longer.ColMatchers[k]
				}
				if !indexMatcherCompatible(shorterMatcher, longerMatcher) {
					isPrefix = false
					break
				}
				if len(shorter.ColOrderMeta) <= k || len(longer.ColOrderMeta) <= k ||
					shorter.ColOrderMeta[k] != longer.ColOrderMeta[k] {
					isPrefix = false
					break
				}
			}
			if isPrefix {
				longer.Savings += shorter.Savings
				removed[j] = true
			}
		}
	}

	// 3. Assign surviving candidates to new shard, mark hottest as Native
	result := make([]*StorageIndex, 0, len(candidates))
	for i, idx := range candidates {
		if !removed[i] {
			result = append(result, idx)
		}
	}
	// pick the highest-Savings index to physically sort by
	// Native indexes are forbidden on computed columns because their values
	// can change via Invalidate/SetValue which would break the physical sort order.
	bestSavings := 4.0 // minimum threshold for physical sort
	bestIdx := -1
	for i, idx := range result {
		if idx.Savings > bestSavings && !indexHasComputedCol(t2, idx) {
			bestSavings = idx.Savings
			bestIdx = i
		}
	}
	if bestIdx >= 0 {
		result[bestIdx].Native = true
	}
	t2.Indexes = result
}

// fullScan iterates all record IDs (main + delta) in natural order.
// Used when the index is not built yet or was evicted.
func (s *StorageIndex) fullScan(maxInsertIndex int, buf []uint32, matchers []IndexRowMatcher, callback func([]uint32) bool) {
	bufN := 0
	for i := uint32(0); i < s.t.main_count; i++ {
		buf[bufN] = i
		bufN++
		if bufN == len(buf) {
			if !emitRowMatchers(matchers, buf[:bufN], callback) {
				return
			}
			bufN = 0
		}
	}
	for i := 0; i < maxInsertIndex; i++ {
		buf[bufN] = s.t.main_count + uint32(i)
		bufN++
		if bufN == len(buf) {
			if !emitRowMatchers(matchers, buf[:bufN], callback) {
				return
			}
			bufN = 0
		}
	}
	if bufN > 0 {
		emitRowMatchers(matchers, buf[:bufN], callback)
	}
}

// buildIndex constructs the shared index data structures.
// cols must contain value getters for each index column in order.
// The caller must hold s.mu.Lock() or have exclusive access.
func (s *StorageIndex) buildIndex(state *storageIndexState, cols []colGetter, tx *TxContext) {
	startRevisions := s.computedRevisionsRLocked()
	if !s.Native {
		// main storage: build sort-order index
		tmp := make([]uint32, s.t.main_count)
		relations := make([]func(scm.Scmer, scm.Scmer) bool, len(cols))
		for i := range relations {
			if i < len(s.ColOrder) && s.ColOrder[i] != nil {
				relations[i] = s.ColOrder[i]
			} else {
				relations[i] = scm.Less
			}
		}
		for i := uint32(0); i < s.t.main_count; i++ {
			tmp[i] = i // fill with natural order
		}
		// Pre-materialize sort-key values with one sequential pass per sorted
		// column, instead of calling g.get(recid) from inside the sort
		// comparator. Compressed encodings like StorageSeq keep a single-slot
		// pivot cache tuned for sequential access; decoding here keeps every
		// read on that cache's fast path, while decoding from the comparator
		// visits recids in the sort's effectively random order and forces a
		// fresh bisection on nearly every call (measured: >99% of buildIndex
		// time for a StorageSeq-encoded column).
		// Single flat, row-major buffer (materialized[recid*numCols+colIdx]):
		// one contiguous allocation total. The comparator (called ~N*log(N)
		// times) reads every sorted column of the same recid together, so
		// keeping a row's values in one cache line matters more than keeping
		// each column's materialization write sequential (called only N*K
		// times total, i.e. far less often).
		numCols := len(cols)
		mainCount := uint(s.t.main_count)
		materialized := make([]scm.Scmer, mainCount*uint(numCols))
		hasSortCol := make([]bool, numCols)
		for colIdx, g := range cols {
			if !s.columnIsSorted(colIdx) {
				continue // query-level overlay column: never read by the comparator below
			}
			hasSortCol[colIdx] = true
			for i := uint32(0); i < s.t.main_count; i++ {
				materialized[uint(i)*uint(numCols)+uint(colIdx)] = g.get(i)
			}
		}
		// sort indexes; skip non-sorted matcher columns (they don't affect
		// sort order, they are query-level overlays for pruning)
		hybridsort.HybridSort(tmp, func(a, b uint32) bool {
			rowA := uint(a) * uint(numCols)
			rowB := uint(b) * uint(numCols)
			for colIdx := range cols {
				if !hasSortCol[colIdx] {
					continue
				}
				va := materialized[rowA+uint(colIdx)]
				vb := materialized[rowB+uint(colIdx)]
				if relations[colIdx](va, vb) {
					return true // less
				} else if relations[colIdx](vb, va) {
					return false // greater
				}
				// otherwise: next iteration
			}
			return false // fully equal
		})
		// store sorted values into compressed format
		state.mainIndexes.prepare()
		for i, v := range tmp {
			state.mainIndexes.scan(uint32(i), scm.NewInt(int64(v)))
		}
		state.mainIndexes.init(uint32(len(tmp)))
		for i, v := range tmp {
			state.mainIndexes.build(uint32(i), scm.NewInt(int64(v)))
		}
		state.mainIndexes.finish()
		// Capture min/max from sorted permutation for interpolation search
		if len(tmp) > 0 {
			state.minVals = make([]scm.Scmer, len(cols))
			state.maxVals = make([]scm.Scmer, len(cols))
			for i, g := range cols {
				if !s.columnIsSorted(i) {
					continue
				}
				state.minVals[i] = g.get(tmp[0])
				state.maxVals[i] = g.get(tmp[len(tmp)-1])
			}
		}
	} else if s.t.main_count > 0 {
		// Native index: identity order, recid 0 = min, recid N-1 = max
		state.minVals = make([]scm.Scmer, len(cols))
		state.maxVals = make([]scm.Scmer, len(cols))
		for i, g := range cols {
			if !s.columnIsSorted(i) {
				continue
			}
			state.minVals[i] = g.get(0)
			state.maxVals[i] = g.get(s.t.main_count - 1)
		}
	}

	// Deploy query-independent, shard-local custom indexes once. Their concrete
	// caches remain hidden behind IndexHook; the scan only sees bound matchers.
	state.indexHooks = make([]IndexHook, len(s.ColMatchers))
	var indexHookBytes int64
	for colIdx, analyzer := range s.ColMatchers {
		if analyzer == nil || analyzer.IsSorted() {
			continue
		}
		// Repeated predicates of the same index kind bind independently but
		// share their shard-local cache.
		for previous := 0; previous < colIdx; previous++ {
			if s.Cols[previous] == s.Cols[colIdx] && s.ColMatchers[previous] != nil && matcherKindEqual(s.ColMatchers[previous], analyzer) {
				state.indexHooks[colIdx] = state.indexHooks[previous]
				break
			}
		}
		if state.indexHooks[colIdx] != nil {
			continue
		}
		var reader ColumnReader
		if colIdx < len(cols) {
			reader = cols[colIdx].raw
		}
		hook := analyzer.Deploy(IndexDeployContext{
			MainCount: s.t.main_count,
			Column:    reader,
			shard:     s.t,
		}, true)
		state.indexHooks[colIdx] = hook
		if hook != nil {
			indexHookBytes += int64(hook.ComputeSize())
		}
	}
	state.indexHookBytes.Store(indexHookBytes)
	// (previously: else: Native index comment)

	// delta storage — comparator uses getDeltaColValue so computed columns work;
	// skip non-sorted matcher columns (they don't participate in sort order)
	state.deltaBtree = btree.NewG[indexPair](8, func(a, b indexPair) bool {
		compareCols := len(s.Cols)
		if a.compareCols > 0 && a.compareCols < compareCols {
			compareCols = a.compareCols
		}
		if b.compareCols > 0 && b.compareCols < compareCols {
			compareCols = b.compareCols
		}
		for colIdx := 0; colIdx < compareCols; colIdx++ {
			if !s.columnIsSorted(colIdx) {
				continue
			}
			var av, bv scm.Scmer
			if a.itemid == -1 {
				av = a.reference.lower(colIdx)
			} else if state.precomputedDelta {
				av = a.data[colIdx]
			} else {
				av = s.getDeltaColValue(uint32(a.itemid), a.data, colIdx)
			}
			if b.itemid == -1 {
				bv = b.reference.lower(colIdx)
			} else if state.precomputedDelta {
				bv = b.data[colIdx]
			} else {
				bv = s.getDeltaColValue(uint32(b.itemid), b.data, colIdx)
			}
			if s.lessAt(colIdx, av, bv) {
				return true // less
			} else if s.lessAt(colIdx, bv, av) {
				return false // greater
			}
			// otherwise: next iteration
		}
		// tiebreak by itemid so duplicate key values are never "equal"
		// (prevents ReplaceOrInsert from dropping rows with same key)
		return a.itemid < b.itemid
	})
	// fill deltaBtree with global record IDs
	for i, data := range s.t.inserts {
		recid := s.t.main_count + uint32(i)
		state.deltaBtree.ReplaceOrInsert(indexPair{itemid: int(recid), data: data})
	}

	endRevisions := s.computedRevisionsRLocked()
	state.computedRevisions = endRevisions
	state.active = sameComputedRevisions(startRevisions, endRevisions)
}

// buildMainIndexPositionsLocked lazily constructs the inverse of mainIndexes
// as a compressed StorageInt. The forward index remains the only mandatory
// structure. Sparse ordered RecSet scans pay this O(N) pass only on their first
// actual use, then share it through the normal index cache lifecycle.
//
// The caller holds s.mu. Decode uses StorageInt's allocation-free uint32 range
// API into one reusable stack buffer; no Scmer boxing or per-row allocation is
// introduced by the inversion pass.
func (s *StorageIndex) buildMainIndexPositionsLocked(state *storageIndexState) uint {
	if s.Native || state.mainIndexPositions.count > 0 || s.t.main_count == 0 {
		return 0
	}
	count := s.t.main_count
	state.mainIndexPositions.initValuesUInt32(count, 0, count-1)
	var recids [1024]uint32
	for base := uint32(0); base < count; {
		chunkCount := count - base
		if chunkCount > uint32(len(recids)) {
			chunkCount = uint32(len(recids))
		}
		chunk := recids[:chunkCount]
		state.mainIndexes.GetValuesUInt32Range(base, chunkCount, chunk, 1)
		for offset, recid := range chunk {
			state.mainIndexPositions.buildValueUInt32(recid, base+uint32(offset))
		}
		base += chunkCount
	}
	return state.mainIndexPositions.ComputeSize()
}

func recSetSortWork(rows int64) int64 {
	if rows <= 1 {
		return rows
	}
	return rows * int64(bits.Len64(uint64(rows-1)))
}

const (
	// BenchmarkRecSetBoundaryCrossover measures about 4 ns per RecSet row/log2
	// sorting unit and at least 6 ns per positive/range membership candidate on
	// the reference CPU. Integer picoseconds keep this hot decision allocation-
	// free and make its independently measured constants explicit.
	orderedRecSetSortUnitPs     = int64(4_000)
	orderedIndexMembershipRowPs = int64(6_000)
)

func orderedIndexExpectedRows(recsetRows, universeRows, indexSpanRows int64, limit int) int64 {
	if recsetRows <= 0 || indexSpanRows <= 0 {
		return 0
	}
	if limit <= 0 || int64(limit) >= recsetRows {
		return indexSpanRows
	}
	expected := (int64(limit)*universeRows + recsetRows - 1) / recsetRows
	if expected > indexSpanRows {
		return indexSpanRows
	}
	return expected
}

func orderedRecSetSwitchRows(recsetRows int64) int64 {
	workPs := recSetSortWork(recsetRows) * orderedRecSetSortUnitPs
	return (workPs + orderedIndexMembershipRowPs - 1) / orderedIndexMembershipRowPs
}

// orderedRecSetDominates compares two execution kernels, not two relational
// plans. The base-index side estimates how far an ordered iterator must walk
// to find LIMIT hits at the RecSet's observed density, capped by the effective
// access-bound min/max interval. The inverse side bulk-loads RecID positions
// and sorts them. scan and scan_order deliberately use different crossovers.
// Keep the constants synchronized with BenchmarkRecSetBoundaryCrossover.
func orderedRecSetDominates(recsetRows, universeRows, indexSpanRows int64, limit int) bool {
	if recsetRows <= 0 {
		return true
	}
	if indexSpanRows <= 0 || recsetRows >= indexSpanRows {
		return false
	}
	expectedRows := orderedIndexExpectedRows(recsetRows, universeRows, indexSpanRows, limit)
	return orderedRecSetSwitchRows(recsetRows) < expectedRows
}

func unorderedRecSetDominates(recsetRows, indexSpanRows int64) bool {
	return recsetRows >= 0 && recsetRows < indexSpanRows
}

// iterateRecSetFirst emits one exact RecSet boundary in this index's order.
// The query-local buffers contain only uint32 values. For an active inverse
// permutation the hot path bulk-decodes RecID -> index position, sorts those
// positions, then bulk-decodes the forward permutation back to RecIDs. If no
// inverse exists yet, the same bounded buffer is sorted by the index key
// callbacks; this keeps a cold sparse query from degrading to a full scan.
func (s *StorageIndex) iterateRecSetFirst(tx *TxContext, state *storageIndexState, part *recSetShard, bounds scanAccess, indexBounds scanIndexBounds, upperInclusive bool, mainStart int, mainEnd int, maxInsertIndex int, buf []uint32, cols []colGetter, persistent bool, ordered bool, exactMain *bool, callback func([]uint32) bool) {
	if part == nil || part.count == 0 {
		return
	}
	var localItems [1024]uint32
	items := localItems[:0]
	if part.count > int64(cap(localItems)) {
		items = make([]uint32, 0, int(part.count))
	}
	visibleUpper := s.t.main_count + uint32(maxInsertIndex)
	part.forEachID(func(recid uint32) bool {
		if recid < visibleUpper {
			items = append(items, recid)
		}
		return true
	})
	if len(items) == 0 {
		return
	}

	usedInverse := ordered && persistent && !s.Native &&
		state != nil && state.mainIndexPositions.count == uint64(s.t.main_count)
	if usedInverse {
		mainCount := sort.Search(len(items), func(i int) bool { return items[i] >= s.t.main_count })
		mainItems := items[:mainCount]
		deltaItems := items[mainCount:]
		var localPositions [1024]uint32
		positions := localPositions[:0]
		if len(mainItems) > len(localPositions) {
			positions = make([]uint32, len(mainItems))
		} else {
			positions = localPositions[:len(mainItems)]
		}
		state.mainIndexPositions.GetValuesUInt32Multi(mainItems, positions, 1)
		// Access/equality bounds may cover only a narrow part of the ordered
		// index. Compact before sorting so the RecSet kernel pays only for the
		// same physical min/max interval as the sequential kernel.
		kept := positions[:0]
		for _, position := range positions {
			if int(position) >= mainStart && int(position) < mainEnd {
				kept = append(kept, position)
			}
		}
		positions = kept
		hybridsort.Slice(positions, func(i, j int) bool { return positions[i] < positions[j] })
		mainItems = mainItems[:len(positions)]
		state.mainIndexes.GetValuesUInt32Multi(positions, mainItems, 1)

		valueAt := func(recid uint32, col int) scm.Scmer {
			if recid < s.t.main_count {
				return cols[col].get(recid)
			}
			return s.getDeltaColValueTx(tx, recid, s.t.inserts[recid-s.t.main_count], col)
		}
		lessRecID := func(leftID, rightID uint32) bool {
			for col := range s.Cols {
				if !s.columnIsSorted(col) {
					continue
				}
				left := valueAt(leftID, col)
				right := valueAt(rightID, col)
				if s.lessAt(col, left, right) {
					return true
				}
				if s.lessAt(col, right, left) {
					return false
				}
			}
			return leftID < rightID
		}
		if len(deltaItems) > 0 {
			cmpCols := s.queryIndexPrefixLen(bounds, indexBounds)
			_, lastSorted, sortedMask, unboundedMask := s.boundKernel(bounds, cmpCols)
			lowerInclusive, _ := effectiveBoundaryInclusiveness(bounds, indexBounds)
			keptDelta := deltaItems[:0]
			for _, recid := range deltaItems {
				inRange, _ := s.rowWithinBounds(bounds, indexBounds, cmpCols, lastSorted, sortedMask, unboundedMask, lowerInclusive, upperInclusive, func(col int) scm.Scmer {
					return valueAt(recid, col)
				})
				if inRange {
					keptDelta = append(keptDelta, recid)
				}
			}
			deltaItems = keptDelta
			hybridsort.Slice(deltaItems, func(i, j int) bool {
				return lessRecID(deltaItems[i], deltaItems[j])
			})
		}

		// Main and delta orders are individually sorted but may interlace. Merge
		// them into one reusable uint32 buffer so ORDER/LIMIT observes exactly the
		// same sequence as the ordinary index+delta B-tree streaming merge.
		var localMerged [1024]uint32
		merged := localMerged[:0]
		if len(mainItems)+len(deltaItems) > cap(localMerged) {
			merged = make([]uint32, 0, len(mainItems)+len(deltaItems))
		}
		mainPos, deltaPos := 0, 0
		for mainPos < len(mainItems) && deltaPos < len(deltaItems) {
			if lessRecID(deltaItems[deltaPos], mainItems[mainPos]) {
				merged = append(merged, deltaItems[deltaPos])
				deltaPos++
			} else {
				merged = append(merged, mainItems[mainPos])
				mainPos++
			}
		}
		merged = append(merged, mainItems[mainPos:]...)
		merged = append(merged, deltaItems[deltaPos:]...)
		items = merged
	} else if ordered {
		valueAt := func(recid uint32, col int) scm.Scmer {
			if recid < s.t.main_count {
				return cols[col].get(recid)
			}
			return s.getDeltaColValueTx(tx, recid, s.t.inserts[recid-s.t.main_count], col)
		}
		hybridsort.Slice(items, func(i, j int) bool {
			for col := range s.Cols {
				if !s.columnIsSorted(col) {
					continue
				}
				left := valueAt(items[i], col)
				right := valueAt(items[j], col)
				if s.lessAt(col, left, right) {
					return true
				}
				if s.lessAt(col, right, left) {
					return false
				}
			}
			return items[i] < items[j]
		})
	}

	matchers := s.bindRowMatchers(tx, bounds, indexBounds, upperInclusive, cols, func() []IndexHook {
		if persistent && state != nil {
			return state.indexHooks
		}
		return nil
	}(), persistent, exactMain)
	for len(items) > 0 {
		count := len(items)
		if count > len(buf) {
			count = len(buf)
		}
		copy(buf, items[:count])
		if !emitRowMatchers(matchers, buf[:count], callback) {
			return
		}
		items = items[count:]
	}
}

// bindRowMatchers returns only matcher state. The terminal callback deliberately
// stays outside this value: storing it in a returned wrapper closure makes the
// complete shard scan state escape even though index iteration is synchronous.
func (s *StorageIndex) bindRowMatchers(tx *TxContext, bounds scanAccess, indexBounds scanIndexBounds, upperInclusive bool, cols []colGetter, hooks []IndexHook, persistent bool, exactMain *bool) []IndexRowMatcher {
	var matchers []IndexRowMatcher
	if !persistent {
		matchers = s.bindColdRangeMatcher(tx, bounds, indexBounds, upperInclusive, cols)
	}
	for colIdx := 0; colIdx < bounds.len(); colIdx++ {
		bound := bounds.boundary(colIdx)
		if bound.matcher == nil || bound.matcher.IsSorted() {
			continue
		}
		var hook IndexHook
		alignedWithIndex := colIdx < len(s.Cols) && s.Cols[colIdx] == bound.col
		if alignedWithIndex && colIdx < len(s.ColMatchers) {
			alignedWithIndex = indexMatcherCompatible(bound.matcher, s.ColMatchers[colIdx])
		} else if alignedWithIndex {
			alignedWithIndex = bound.matcher.IsSorted()
		}
		if persistent && alignedWithIndex && colIdx < len(hooks) {
			hook = hooks[colIdx]
		}
		if hook == nil {
			var reader ColumnReader
			if alignedWithIndex && colIdx < len(cols) {
				reader = cols[colIdx].raw
			} else if !isScanPseudoColName(bound.col) && bound.mapFn.IsNil() {
				reader = newCachedColumnReaderTx(s.t.getColumnStorageRLocked(bound.col), tx)
			}
			hook = bound.matcher.Deploy(IndexDeployContext{
				MainCount: s.t.main_count,
				Column:    reader,
				shard:     s.t,
			}, false)
		}
		if hook == nil {
			continue
		}
		matcher := hook.Bind(bound.lower)
		if matcher != nil {
			matchers = append(matchers, matcher)
		}
	}
	// Candidate matchers never replace the original predicate. This is
	// required for approximate indexes such as n-grams and harmless for exact
	// matchers such as RecSet membership.
	if len(matchers) > 0 && exactMain != nil {
		*exactMain = false
	}
	return matchers
}

// Keep the capturing cold-index range matcher out of bindRowMatchers. If it is
// inlined into that function, Go makes the access view escape even for the hot
// persistent-index branch where the closure is never constructed.
//
//go:noinline
func (s *StorageIndex) bindColdRangeMatcher(tx *TxContext, bounds scanAccess, indexBounds scanIndexBounds, upperInclusive bool, cols []colGetter) []IndexRowMatcher {
	cmpCols := s.queryIndexPrefixLen(bounds, indexBounds)
	if cmpCols == 0 {
		return nil
	}
	_, lastSorted, sortedMask, unboundedMask := s.boundKernel(bounds, cmpCols)
	lowerInclusive, _ := effectiveBoundaryInclusiveness(bounds, indexBounds)
	return []IndexRowMatcher{func(ids []uint32) []uint32 {
		out := 0
		for _, recid := range ids {
			valueAt := func(col int) scm.Scmer {
				if recid < s.t.main_count {
					return cols[col].get(recid)
				}
				return s.getDeltaColValueTx(tx, recid, s.t.inserts[recid-s.t.main_count], col)
			}
			inRange, _ := s.rowWithinBounds(bounds, indexBounds, cmpCols, lastSorted, sortedMask, unboundedMask, lowerInclusive, upperInclusive, valueAt)
			if inRange && !lowerInclusive {
				lastSorted := cmpCols - 1
				for lastSorted >= 0 && !s.columnIsSorted(lastSorted) {
					lastSorted--
				}
				if lastSorted >= 0 && !indexBounds.lower(lastSorted).IsNil() &&
					s.compareAt(lastSorted, valueAt(lastSorted), indexBounds.lower(lastSorted)) == 0 {
					inRange = false
				}
			}
			if inRange {
				ids[out] = recid
				out++
			}
		}
		return ids[:out]
	}}
}

func emitRowMatchers(matchers []IndexRowMatcher, ids []uint32, callback func([]uint32) bool) bool {
	for _, matcher := range matchers {
		ids = matcher(ids)
		if len(ids) == 0 {
			return true
		}
	}
	return callback(ids)
}

func (s *StorageIndex) estimateHookCandidates(tx *TxContext, bounds scanAccess) (uint32, uint32, bool) {
	state := &s.baseState
	if state == nil {
		return 0, 0, false
	}
	s.mu.Lock()
	if !state.active {
		s.mu.Unlock()
		return 0, 0, false
	}
	hooks := append([]IndexHook(nil), state.indexHooks...)
	s.mu.Unlock()

	var candidates uint32
	var universe uint32
	found := false
	for colIdx := 0; colIdx < bounds.len(); colIdx++ {
		bound := bounds.boundary(colIdx)
		if bound.matcher == nil || bound.matcher.IsSorted() || colIdx >= len(hooks) {
			continue
		}
		estimator, ok := hooks[colIdx].(IndexCandidateEstimator)
		if !ok {
			continue
		}
		count, hookUniverse, ok := estimator.EstimateCandidates(bound.lower)
		if !ok {
			continue
		}
		if !found || count < candidates {
			candidates = count
		}
		if hookUniverse > universe {
			universe = hookUniverse
		}
		found = true
	}
	return candidates, universe, found
}

// iterate over index using a caller-provided buffer for batching
func (s *StorageIndex) iterate(tx *TxContext, bounds scanAccess, indexBounds scanIndexBounds, lowerInclusive bool, upperInclusive bool, maxInsertIndex int, buf []uint32, usageWeight float64, forceBuild bool, exactMain *bool, options *indexIterationOptions, candidateSpan *int64, selected func(*StorageIndex, bool), callback func([]uint32) bool) {

	// Build column getters — use RLocked variant because the caller
	// (scan, scan_order, GetRecordidForUnique) already holds s.t.mu.RLock().
	// Re-acquiring RLock via getColumnStorageOrPanic would deadlock when a
	// concurrent writer is waiting for s.t.mu.Lock() (write-preferring RWMutex).
	getterScratch := indexGetterScratchPool.Get().(*indexGetterScratch)
	cols := s.buildGetters(tx, getterScratch.getters[:])
	getterScratch.indexBounds = indexBounds
	defer func() {
		clear(getterScratch.getters[:])
		getterScratch.indexBounds = scanIndexBounds{}
		indexGetterScratchPool.Put(getterScratch)
	}()
	state := &s.baseState
	currentRevisions := s.computedRevisionsRLocked()
	s.mu.Lock()
	if state.active && !sameComputedRevisions(state.computedRevisions, currentRevisions) {
		state.active = false
	}
	stateActive := state.active
	s.mu.Unlock()
	// no collation-specific helpers in the current implementation
	recsetPart, hasRecSetBoundary := smallestRecSetBoundary(bounds, s.t)
	// An exact source with no rows in this shard is terminal. In particular,
	// do not build either the forward or inverse index merely to prove that an
	// empty RecSet produces no candidates.
	if hasRecSetBoundary && (recsetPart == nil || recsetPart.count == 0) {
		if selected != nil {
			selected(s, options != nil)
		}
		return
	}
	indexSpanRows := int64(s.t.main_count) + int64(maxInsertIndex)
	preferRecSet := false
	if hasRecSetBoundary {
		rows := int64(0)
		if recsetPart != nil {
			rows = recsetPart.count
		}
		if options != nil {
			preferRecSet = orderedRecSetDominates(rows, int64(recsetPart.universe), indexSpanRows, options.orderedLimit)
		} else {
			preferRecSet = unorderedRecSetDominates(rows, indexSpanRows)
		}
	}

	// An unordered exact RecSet already is the cheapest possible candidate
	// iterator when it is smaller than the base-index span. Building a forward
	// index cannot provide order or reduce the candidate count; in particular,
	// repeated batch-local scan_recset calls must not eventually build a
	// synthetic $recset_contains index merely because their savings score grew.
	// Ordered scans remain below this dominance rule because their measured
	// base-walk versus inverse-position crossover is a real runtime choice.
	if preferRecSet && options == nil {
		if selected != nil {
			selected(s, false)
		}
		s.iterateRecSetFirst(tx, nil, recsetPart, bounds, indexBounds,
			upperInclusive, 0, int(s.t.main_count), maxInsertIndex, buf, cols, false, false, exactMain, callback)
		return
	}
	savingsThreshold := 2.0 // building an index costs 1x the time as traversing the list
	savings := s.addSavings(state, usageWeight)
	if !stateActive {
		// index is not built yet
		if savings < savingsThreshold && !forceBuild {
			if preferRecSet {
				if selected != nil {
					selected(s, options != nil)
				}
				s.iterateRecSetFirst(tx, nil, recsetPart, bounds, indexBounds,
					upperInclusive, 0, int(s.t.main_count), maxInsertIndex, buf, cols, false, options != nil, exactMain, callback)
				return
			}
			// iterate over all items because we don't want to store the index
			if selected != nil {
				selected(s, false)
			}
			matchers := s.bindRowMatchers(tx, bounds, indexBounds, upperInclusive, cols, nil, false, exactMain)
			s.fullScan(maxInsertIndex, buf, matchers, callback)
			return
		} else {
			// Rebuild index without blocking on index mutex contention.
			// Under heavy parallel UPDATE load, waiting here can stall requests.
			// Falling back to a single full scan keeps progress while another
			// goroutine builds or updates this index.
			if !s.mu.TryLock() {
				if selected != nil {
					selected(s, false)
				}
				matchers := s.bindRowMatchers(tx, bounds, indexBounds, upperInclusive, cols, nil, false, exactMain)
				s.fullScan(maxInsertIndex, buf, matchers, callback)
				return
			}
			if state.active {
				// someone has built it in the meantime
				s.mu.Unlock()
				goto start_scan
			}
			s.buildIndex(state, cols, tx)
			s.mu.Unlock()
			// register with CacheManager
			GlobalCache.AddItem(s, int64(s.ComputeSize()), TypeIndex, indexCleanup, indexLastUsed, indexGetScore)
		}
	}
start_scan:

	// Snapshot index state under the lock to prevent a TOCTOU race with
	// indexCleanup, which may set active=false / mainIndexes={} / deltaBtree=nil
	// concurrently. The snapshot keeps the backing data alive via GC references.
	if !s.mu.TryLock() {
		if selected != nil {
			selected(s, false)
		}
		matchers := s.bindRowMatchers(tx, bounds, indexBounds, upperInclusive, cols, nil, false, exactMain)
		s.fullScan(maxInsertIndex, buf, matchers, callback)
		return
	}
	if !state.active {
		// Index was evicted between our initial check and here.
		s.mu.Unlock()
		if selected != nil {
			selected(s, false)
		}
		matchers := s.bindRowMatchers(tx, bounds, indexBounds, upperInclusive, cols, nil, false, exactMain)
		s.fullScan(maxInsertIndex, buf, matchers, callback)
		return
	}
	snapMainIndexes := state.mainIndexes
	// No clone needed: the shard's RLock (held by caller) prevents concurrent
	// inserts from modifying deltaBtree. The index mutex protects against
	// eviction only; the btree data is stable under RLock.
	snapDeltaBtree := state.deltaBtree
	snapIndexHooks := state.indexHooks
	isNative := s.Native
	s.mu.Unlock()
	if selected != nil {
		selected(s, true)
	}
	// A fully index-covered filter cannot reject an otherwise visible row. When
	// this active index also supplies ORDER BY, emit only the requested prefix
	// per callback so LIMIT can brake inside the index walk. The caller-owned
	// pooled buffer remains allocated at its normal size; only its visible slice
	// is shortened, so the hot path adds no allocation.
	cmpCols := s.queryIndexPrefixLen(bounds, indexBounds)
	firstSorted, lastSorted, sortedMask, unboundedMask := s.boundKernel(bounds, cmpCols)
	if options != nil && options.boundaryCoveredLimit && options.orderedLimit > 0 &&
		options.orderedLimit < len(buf) && indexCoversBoundaryOrder(s, true, bounds, cmpCols) {
		buf = buf[:options.orderedLimit]
	}

	// record-ID lookup: identity when data is physically sorted (Native), index dereference otherwise
	getRecid := func(idx int) uint32 {
		return uint32(int64(snapMainIndexes.GetValueUInt(uint32(idx))) + snapMainIndexes.offset)
	}
	if isNative {
		getRecid = func(idx int) uint32 { return uint32(idx) }
	}

	// Bisect only over the query prefix physically represented by this index.
	// Invocation-only hooks remain in bounds for bindRowMatchers below, but do
	// not occupy a key position in a reusable longer index.

	// Find the leading physical sort key. Non-sorted matchers such as LIKE are
	// deliberately present in the logical boundary list but do not participate
	// in buildIndex ordering and must never be used for binary search.
	// Use last-hit hint to narrow binary search range (helps sorted outer loops).
	// The hint is advisory: if stale or from a concurrent goroutine, we safely
	// fall through to an unnarrowed search. No correctness dependency on the hint.
	// LIKE columns cannot participate in binary search (pattern doesn't map to sort order).
	searchLo := 0
	searchN := int(s.t.main_count)
	if hint := int(s.lastHit.Load()); hint > 0 && hint < searchN && firstSorted >= 0 && !indexBounds.lower(firstSorted).IsNil() {
		hintVal := cols[firstSorted].get(getRecid(hint))
		if !hintVal.IsNil() {
			if s.compareAt(firstSorted, hintVal, indexBounds.lower(firstSorted)) < 0 {
				searchLo = hint
				searchN -= hint
			} else if s.compareAt(firstSorted, indexBounds.lower(firstSorted), hintVal) < 0 {
				searchN = hint + 1
			}
		}
	}
	mainIdx := 0
	if firstSorted >= 0 && !indexBounds.lower(firstSorted).IsNil() {
		if s.usesNaturalAscendingOrder(firstSorted) {
			var interpMin, interpMax scm.Scmer
			if len(state.minVals) > firstSorted {
				interpMin = state.minVals[firstSorted]
				interpMax = state.maxVals[firstSorted]
			}
			mainIdx = interpolationSearch(searchLo, searchN, indexBounds.lower(firstSorted), interpMin, interpMax,
				func(idx int) scm.Scmer {
					return cols[firstSorted].get(getRecid(idx))
				})
		} else {
			mainIdx = searchLo + sort.Search(searchN, func(idx int) bool {
				value := cols[firstSorted].get(getRecid(searchLo + idx))
				return s.compareAt(firstSorted, value, indexBounds.lower(firstSorted)) >= 0
			})
		}
	}
	s.lastHit.Store(uint32(mainIdx))
	// skip past equal values when lower bound is exclusive (col > 5)
	// LIKE columns don't have lower/upper semantics, so skip this optimization.
	if !lowerInclusive && lastSorted >= 0 && !indexBounds.lower(lastSorted).IsNil() {
		for uint32(mainIdx) < s.t.main_count {
			recid := getRecid(mainIdx)
			if s.compareAt(lastSorted, cols[lastSorted].get(recid), indexBounds.lower(lastSorted)) != 0 {
				break
			}
			mainIdx++
		}
	}

	// Resolve the effective main-index min/max interval before choosing the
	// RecSet traversal kernel. A narrow equality/range prefix can make an index
	// membership walk cheaper even when the RecSet is small relative to the
	// whole table; whole-table density is therefore the wrong crossover input.
	mainEnd := int(s.t.main_count)
	if lastSorted >= 0 && !indexBounds.upperLast().IsNil() && mainIdx < mainEnd {
		mainEnd = mainIdx + sort.Search(mainEnd-mainIdx, func(offset int) bool {
			recid := getRecid(mainIdx + offset)
			_, beyond := s.rowWithinBounds(bounds, indexBounds, cmpCols, lastSorted, sortedMask, unboundedMask, lowerInclusive, upperInclusive, func(col int) scm.Scmer {
				return cols[col].get(recid)
			})
			return beyond
		})
	}
	mainStart := mainIdx
	indexSpanRows = int64(mainEnd-mainStart) + int64(maxInsertIndex)
	if candidateSpan != nil {
		*candidateSpan = indexSpanRows
	}
	if hasRecSetBoundary {
		rows := int64(0)
		if recsetPart != nil {
			rows = recsetPart.count
		}
		if options != nil {
			preferRecSet = orderedRecSetDominates(rows, int64(recsetPart.universe), indexSpanRows, options.orderedLimit)
		} else {
			preferRecSet = unorderedRecSetDominates(rows, indexSpanRows)
		}
	}
	runRecSetKernel := func(start, end int, target func([]uint32) bool) {
		var inverseBytes uint
		s.mu.Lock()
		if state.active && options != nil {
			inverseBytes = s.buildMainIndexPositionsLocked(state)
		}
		sparseState := storageIndexState{
			mainIndexes:        state.mainIndexes,
			mainIndexPositions: state.mainIndexPositions,
			indexHooks:         append([]IndexHook(nil), state.indexHooks...),
		}
		s.mu.Unlock()
		if inverseBytes > 0 {
			GlobalCache.UpdateSizeAsync(s, int64(inverseBytes))
		}
		s.iterateRecSetFirst(tx, &sparseState, recsetPart, bounds, indexBounds,
			upperInclusive, start, end, maxInsertIndex, buf, cols, true, options != nil, exactMain, target)
	}
	if preferRecSet {
		runRecSetKernel(mainIdx, mainEnd, callback)
		return
	}

	rawCallback := callback
	matchers := s.bindRowMatchers(tx, bounds, indexBounds, upperInclusive, cols, snapIndexHooks, true, exactMain)
	// For one constrained sorted key, the two binary searches define the exact
	// main-row interval. Non-sorted access hooks still run in emitRowMatchers.
	// Composite prefixes keep their row checks because their lower search uses
	// only the first sorted key.
	mainRangeCovered := false
	if firstSorted >= 0 && lastSorted < 64 {
		constrainedSorted := sortedMask &^ unboundedMask
		mainRangeCovered = constrainedSorted == uint64(1)<<firstSorted
	}
	adaptiveSwitchRows := int64(0)
	if options != nil && hasRecSetBoundary && maxInsertIndex == 0 {
		adaptiveSwitchRows = orderedRecSetSwitchRows(recsetPart.count)
		if adaptiveSwitchRows >= indexSpanRows {
			adaptiveSwitchRows = 0
		}
	}

	nextMain := func() (uint32, bool) {
		for {
			if mainIdx >= mainEnd {
				return 0, false
			}
			recid := getRecid(mainIdx)
			mainIdx++
			if mainRangeCovered {
				return recid, true
			}
			inRange, beyond := s.rowWithinBounds(bounds, indexBounds, cmpCols, lastSorted, sortedMask, unboundedMask, lowerInclusive, upperInclusive, func(i int) scm.Scmer {
				return cols[i].get(recid)
			})
			if inRange {
				return recid, true
			}
			if beyond {
				return 0, false
			}
		}
	}

	// Streaming merge of main (via nextMain) and delta (via deltaBtree).
	// Both iterators produce items in index-column order; the merge
	// interleaves them to maintain global sort order without intermediate
	// materialization of delta items.
	//
	// NOTE on deletions: deleted rows are NOT filtered here. The deltaBtree
	// intentionally retains items whose underlying row has been marked as
	// deleted (t.deletions). Filtering happens in the scan layer (scan.go,
	// scan_order.go) which checks t.deletions and the transaction visibility
	// overlay. This is by design: a DELETE may be rolled back by a concurrent
	// transaction, so the index must keep all rows and let the scan layer
	// decide visibility per-transaction.
	bufN := 0
	stopped := false
	emit := func(id uint32) {
		buf[bufN] = id
		bufN++
		if bufN == len(buf) {
			if !emitRowMatchers(matchers, buf[:bufN], callback) {
				stopped = true
			}
			bufN = 0
		}
	}

	mainRecid, mainOk := nextMain()

	if maxInsertIndex > 0 && snapDeltaBtree != nil {
		// iterFn handles each delta item in btree order.
		iterFn := func(p indexPair) bool {
			if stopped {
				return false
			}
			if p.itemid < 0 {
				return true
			}
			recid := uint32(p.itemid)
			if recid < s.t.main_count || p.itemid-int(s.t.main_count) >= maxInsertIndex {
				return true
			}
			inRange, beyond := s.rowWithinBounds(bounds, indexBounds, cmpCols, lastSorted, sortedMask, unboundedMask, lowerInclusive, upperInclusive, func(i int) scm.Scmer {
				if state.precomputedDelta {
					return p.data[i]
				}
				return s.getDeltaColValue(recid, p.data, i)
			})
			if !inRange {
				return !beyond
			}
			// drain main items that sort before this delta item
			for mainOk && !stopped {
				cmp := s.compareMainAndDelta(state, mainRecid, cols, p)
				if cmp > 0 {
					break // delta item comes first
				}
				emit(mainRecid)
				mainRecid, mainOk = nextMain()
			}
			// emit this delta item
			if !stopped {
				emit(uint32(p.itemid))
			}
			return !beyond && !stopped
		}

		// For computed or non-sorted matcher columns, AscendGreaterOrEqual cannot be
		// used (computed col names have no entry in deltaColumns, matcher patterns
		// don't map to sort order), so scan all. Prefix lookups remain seekable: the
		// reference comparator stops at cmpCols, making the missing suffix unbounded.
		hasUnsearchableInBounds := state.precomputedDelta
		for i := 0; i < cmpCols; i++ {
			if indexBounds.lower(i).IsNil() || (len(s.ColMapFn) > i && !s.ColMapFn[i].IsNil()) || (bounds.len() > i && !bounds.boundary(i).matcher.IsSorted()) {
				hasUnsearchableInBounds = true
				break
			}
		}
		if hasUnsearchableInBounds {
			snapDeltaBtree.Ascend(iterFn)
		} else {
			// Reference pairs are marked with itemid -1 and interpreted in
			// index-column order by the comparator, so lower is directly
			// seekable without a per-probe reordered copy.
			snapDeltaBtree.AscendGreaterOrEqual(indexPair{itemid: -1, reference: &getterScratch.indexBounds, compareCols: cmpCols}, iterFn)
		}
	}

	// drain remaining main items
	for mainOk && !stopped {
		emit(mainRecid)
		mainRecid, mainOk = nextMain()
		if adaptiveSwitchRows > 0 && int64(mainIdx-mainStart) >= adaptiveSwitchRows {
			// Flush the already ordered prefix first. A false callback means the
			// residual filter has filled LIMIT and no switch is necessary. A true
			// callback proves that its real acceptance rate was lower than the
			// optimistic LIMIT-only estimate, so continue from the next un-emitted
			// index position through the inverse RecSet kernel.
			if bufN > 0 {
				if !emitRowMatchers(matchers, buf[:bufN], callback) {
					stopped = true
				}
				bufN = 0
			}
			if !stopped {
				remainingStart := mainIdx
				if mainOk {
					remainingStart-- // nextMain has prefetched this position
				}
				runRecSetKernel(remainingStart, mainEnd, rawCallback)
			}
			return
		}
	}
	if bufN > 0 && !stopped {
		emitRowMatchers(matchers, buf[:bufN], callback)
	}
}

// indexCleanup is called by the CacheManager when evicting an index.
// Returns false if the index lock cannot be acquired (non-blocking).
func indexCleanup(ptr any, freedByType *[numEvictableTypes]int64) bool {
	idx := ptr.(*StorageIndex)
	return idx.evict(evictFull, 0, freedByType).success
}

func indexLastUsed(ptr any) time.Time {
	// use the parent shard's lastAccessed as proxy
	return time.Unix(0, int64(atomic.LoadUint64(&ptr.(*StorageIndex).t.lastAccessed)))
}

func indexGetScore(ptr any) float64 {
	return ptr.(*StorageIndex).Savings
}
