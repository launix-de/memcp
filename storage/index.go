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

import "sort"
import "sync"
import "sync/atomic"
import "time"
import "strings"

import "github.com/google/btree"
import "github.com/launix-de/memcp/scm"

type indexPair struct {
	itemid int // -1 for reference items
	data   []scm.Scmer
}

// (no op) numeric helper removed; collations now use golang.org/x/text/collate for ordering

// colGetter retrieves a single index-column value for a given record ID.
// For raw columns it reads directly from ColumnStorage; for computed columns
// it evaluates the mapFn over the source column storages.
type colGetter struct {
	raw     ColumnStorage                // non-nil for raw columns
	mapCols []ColumnStorage              // non-nil for computed columns
	mapFn   func(...scm.Scmer) scm.Scmer // non-nil for computed columns
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
	ColMapCols  [][]string  // per-column source col names; nil entry means raw column
	ColMapFn    []scm.Scmer // per-column compute fn; IsNil() entry means raw column
	ColMatchers []BoundaryMatcher // per-column matcher; nil entry = equal/range (sorted)
	// matcherDataCache[colIdx][cacheKey] stores BoundaryMatcherData per column.
	// Each column's matcher can cache multiple entries (e.g. refinement chain
	// "%h%" → "%he%" → "%hei%"). Populated lazily by iterate() via
	// ProduceIndex(); evicted together with the index.
	matcherDataCache []map[string]BoundaryMatcherData
	Savings          float64 // store the amount of time savings here -> add selectivity (outputted / size) on each
	Native      bool        // true when data is physically sorted by this index (zero-cost)
	mainIndexes StorageInt  // we can do binary searches here (unused when Native)
	// deltaBtree holds delta inserts in index-column order. Contract:
	// when active==true, deltaBtree is non-nil. It is built during index
	// construction and kept in sync by shard.insertDataset on every insert.
	// Deleted rows are intentionally kept in the btree — the scan layer
	// filters them via t.deletions / transaction visibility overlays,
	// because a DELETE can be rolled back by a concurrent transaction.
	deltaBtree *btree.BTreeG[indexPair]
	t          *storageShard
	active     bool
	lastHit    atomic.Uint32 // last binary search position for sorted access pattern optimization
	mu         sync.Mutex
}

// buildGetters returns per-column value getters for this index, reading from the
// shard under a currently-held RLock. Must be called with s.t.mu.RLock held.
func (s *StorageIndex) buildGetters() []colGetter {
	getters := make([]colGetter, len(s.Cols))
	for i, col := range s.Cols {
		if len(s.ColMapFn) > i && !s.ColMapFn[i].IsNil() {
			// computed column: read mapCols and apply mapFn
			mapColStorages := make([]ColumnStorage, len(s.ColMapCols[i]))
			for j, mc := range s.ColMapCols[i] {
				mapColStorages[j] = s.t.getColumnStorageRLocked(mc)
			}
			fn := scm.OptimizeProcToSerialFunction(s.ColMapFn[i])
			getters[i] = colGetter{mapCols: mapColStorages, mapFn: fn}
		} else {
			getters[i] = colGetter{raw: s.t.getColumnStorageRLocked(col)}
		}
	}
	return getters
}

// matcherKindEqual checks if two matchers have the same kind for index deduplication.
func matcherKindEqual(a, b BoundaryMatcher) bool {
	return a.Kind() == b.Kind()
}

func (idx *StorageIndex) ComputeSize() uint {
	var sz uint = 24 * 8 // heuristic
	if !idx.Native {
		sz += idx.mainIndexes.ComputeSize()
	}
	for _, colCache := range idx.matcherDataCache {
		sz += 8 // map header pointer in slice
		for k, data := range colCache {
			sz += uint(len(k)) + 16 + 8 // key string (data+header) + map bucket overhead
			sz += 16 + data.ComputeSize() // interface header + payload
		}
	}
	// TODO: deltaBtree
	return sz
}

// pretty-print
func (idx *StorageIndex) String() string {
	return strings.Join(idx.Cols, "|")
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
func (s *StorageIndex) getDeltaColValue(data []scm.Scmer, colIdx int) scm.Scmer {
	if len(s.ColMapFn) > colIdx && !s.ColMapFn[colIdx].IsNil() {
		fn := scm.OptimizeProcToSerialFunction(s.ColMapFn[colIdx])
		vals := make([]scm.Scmer, len(s.ColMapCols[colIdx]))
		for i, mc := range s.ColMapCols[colIdx] {
			vals[i] = s.getDeltaValue(data, mc)
		}
		return fn(vals...)
	}
	return s.getDeltaValue(data, s.Cols[colIdx])
}

func (s *StorageIndex) rowWithinBounds(cmpCols int, matcherData []BoundaryMatcherData, getter func(int) scm.Scmer) (inRange bool, beyond bool) {
	for i := 0; i < cmpCols; i++ {
		v := getter(i)
		if !matcherData[i].Match(v) {
			return false, matcherData[i].Beyond(v)
		}
	}
	return true, false
}

func (s *StorageIndex) compareMainAndDelta(mainRecid uint32, mainGetters []colGetter, delta indexPair) int {
	for i := range s.Cols {
		if len(s.ColMatchers) > i && !s.ColMatchers[i].IsSorted() {
			continue
		}
		mainVal := mainGetters[i].get(mainRecid)
		deltaVal := s.getDeltaColValue(delta.data, i)
		if scm.Less(mainVal, deltaVal) {
			return -1
		}
		if scm.Less(deltaVal, mainVal) {
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
func (t *storageShard) iterateIndex(cols boundaries, lower []scm.Scmer, upperLast scm.Scmer, maxInsertIndex int, buf []uint32, callback func([]uint32) bool) {
	// cols is already sorted by 1st rank: equality before range; 2nd rank alphabet

	// extract inclusiveness for the range column (last boundary)
	lowerIncl, upperIncl := true, true
	if len(cols) > 0 {
		lowerIncl = cols[len(cols)-1].lowerInclusive
		upperIncl = cols[len(cols)-1].upperInclusive
	}

	// extract query-specific matchers and per-column uppers from boundaries
	var queryMatchers []BoundaryMatcher
	var uppers []scm.Scmer
	if len(lower) > 0 {
		queryMatchers = make([]BoundaryMatcher, len(lower))
		uppers = make([]scm.Scmer, len(lower))
		for i := range lower {
			queryMatchers[i] = cols[i].matcher
			uppers[i] = cols[i].upper
		}
	}

	// check if we found conditions
	if len(lower) > 0 {
		// find an index that has at least the columns in that order we're searching for
		// if the index is inactive, use the other one
	retry_indexscan:
		old_indexes := t.Indexes
		for _, index := range old_indexes {
			// naive index search algo; TODO: improve
			if len(index.Cols) >= len(lower) {
				for i := 0; i < len(lower); i++ {
					if cols[i].col != index.Cols[i] || !matcherKindEqual(cols[i].matcher, index.ColMatchers[i]) {
						goto skip_index // this index does not fit
					}
				}
				// this index fits!
				index.iterate(lower, uppers, upperLast, lowerIncl, upperIncl, queryMatchers, maxInsertIndex, buf, callback)
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
			if len(index.Cols) >= len(lower) {
				covered := true
				for i := 0; i < len(lower); i++ {
					if cols[i].col != index.Cols[i] || !matcherKindEqual(cols[i].matcher, index.ColMatchers[i]) {
						covered = false
						break
					}
				}
				if covered {
					// longer index covers this query; use it instead of creating a shorter one
					t.indexMutex.Unlock()
					index.iterate(lower, uppers, upperLast, lowerIncl, upperIncl, queryMatchers, maxInsertIndex, buf, callback)
					return
				}
			}
		}
		// Don't create an index for tiny shards — a bisect on N rows costs
		// more than a full scan, and the 4 heap allocs dominate the scan time.
		threshold := Settings.IndexThreshold
		if threshold <= 0 {
			threshold = 5
		}
		if int(t.main_count)+maxInsertIndex < threshold {
			t.indexMutex.Unlock()
			// inline full scan (same as the len(lower)==0 path below)
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
			return
		}
		index := new(StorageIndex)
		index.Cols = make([]string, len(lower))
		index.ColMapCols = make([][]string, len(lower))
		index.ColMapFn = make([]scm.Scmer, len(lower))
		index.ColMatchers = make([]BoundaryMatcher, len(lower))
		for i := range lower {
			index.Cols[i] = cols[i].col
			index.ColMapCols[i] = cols[i].mapCols // nil for raw columns
			index.ColMapFn[i] = cols[i].mapFn     // IsNil() for raw columns
			index.ColMatchers[i] = cols[i].matcher // nil for equal/range
		}
		index.Savings = 0.0  // count how many cost we wasted so we decide when to build the index
		index.active = false // tell the engine that index has to be built first
		index.t = t
		t.Indexes = append(t.Indexes, index)
		t.indexMutex.Unlock()
		index.iterate(lower, uppers, upperLast, lowerIncl, upperIncl, queryMatchers, maxInsertIndex, buf, callback)
		return
	}

	// otherwise: iterate over all items in batches
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

func rebuildIndexes(t1 *storageShard, t2 *storageShard) {
	if len(t1.Indexes) == 0 {
		return
	}

	// 1. Clone index metadata from old shard, decay Savings, mark inactive
	candidates := make([]*StorageIndex, 0, len(t1.Indexes))
	for _, idx := range t1.Indexes {
		clone := new(StorageIndex)
		clone.Cols = make([]string, len(idx.Cols))
		copy(clone.Cols, idx.Cols)
		clone.ColMapCols = idx.ColMapCols // shallow copy OK (immutable per-col slices)
		clone.ColMapFn = idx.ColMapFn     // shallow copy OK
		clone.Savings = idx.Savings * 0.9
		clone.t = t2
		clone.active = false
		candidates = append(candidates, clone)
	}

	// 2. Prefix dedup: sort by len(Cols) descending so longer indexes absorb shorter ones
	sort.Slice(candidates, func(i, j int) bool {
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
func (s *StorageIndex) fullScan(maxInsertIndex int, buf []uint32, callback func([]uint32) bool) {
	bufN := 0
	for i := uint32(0); i < s.t.main_count; i++ {
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
		buf[bufN] = s.t.main_count + uint32(i)
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

// buildIndex constructs the index data structures (mainIndexes and deltaBtree).
// cols must contain value getters for each index column in order.
// The caller must hold s.mu.Lock() or have exclusive access.
func (s *StorageIndex) buildIndex(cols []colGetter) {
	if !s.Native {
		// main storage: build sort-order index
		tmp := make([]uint32, s.t.main_count)
		for i := uint32(0); i < s.t.main_count; i++ {
			tmp[i] = i // fill with natural order
		}
		// sort indexes; skip non-sorted matcher columns (they don't affect
		// sort order, they are query-level overlays for pruning)
		sort.Slice(tmp, func(i, j int) bool {
			for colIdx, g := range cols {
				if len(s.ColMatchers) > colIdx && !s.ColMatchers[colIdx].IsSorted() {
					continue
				}
				a := g.get(tmp[i])
				b := g.get(tmp[j])
				if scm.Less(a, b) {
					return true // less
				} else if !scm.Equal(a, b) {
					return false // greater
				}
				// otherwise: next iteration
			}
			return false // fully equal
		})
		// store sorted values into compressed format
		s.mainIndexes.prepare()
		for i, v := range tmp {
			s.mainIndexes.scan(uint32(i), scm.NewInt(int64(v)))
		}
		s.mainIndexes.init(uint32(len(tmp)))
		for i, v := range tmp {
			s.mainIndexes.build(uint32(i), scm.NewInt(int64(v)))
		}
		s.mainIndexes.finish()
	}
	// else: Native index — data is physically sorted, no mainIndexes needed

	// delta storage — comparator uses getDeltaColValue so computed columns work;
	// skip non-sorted matcher columns (they don't participate in sort order)
	s.deltaBtree = btree.NewG[indexPair](8, func(a, b indexPair) bool {
		for colIdx := range s.Cols {
			if len(s.ColMatchers) > colIdx && !s.ColMatchers[colIdx].IsSorted() {
				continue
			}
			av := s.getDeltaColValue(a.data, colIdx)
			bv := s.getDeltaColValue(b.data, colIdx)
			if scm.Less(av, bv) {
				return true // less
			} else if !scm.Equal(av, bv) {
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
		s.deltaBtree.ReplaceOrInsert(indexPair{int(s.t.main_count) + i, data})
	}

	s.active = true // mark as ready
}

// iterate over index using a caller-provided buffer for batching
func (s *StorageIndex) iterate(lower []scm.Scmer, uppers []scm.Scmer, upperLast scm.Scmer, lowerInclusive bool, upperInclusive bool, queryMatchers []BoundaryMatcher, maxInsertIndex int, buf []uint32, callback func([]uint32) bool) {

	// Build column getters — use RLocked variant because the caller
	// (scan, scan_order, GetRecordidForUnique) already holds s.t.mu.RLock().
	// Re-acquiring RLock via getColumnStorageOrPanic would deadlock when a
	// concurrent writer is waiting for s.t.mu.Lock() (write-preferring RWMutex).
	cols := s.buildGetters()
	// no collation-specific helpers in the current implementation

	savings_threshold := 2.0    // building an index costs 1x the time as traversing the list
	s.Savings = s.Savings + 1.0 // mark that we could save time
	if !s.active {
		// index is not built yet
		if s.Savings < savings_threshold {
			// iterate over all items because we don't want to store the index
			s.fullScan(maxInsertIndex, buf, callback)
			return
		} else {
			// Rebuild index without blocking on index mutex contention.
			// Under heavy parallel UPDATE load, waiting here can stall requests.
			// Falling back to a single full scan keeps progress while another
			// goroutine builds or updates this index.
			if !s.mu.TryLock() {
				s.fullScan(maxInsertIndex, buf, callback)
				return
			}
			if s.active {
				// someone has built it in the meantime
				s.mu.Unlock()
				goto start_scan
			}
			s.buildIndex(cols)
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
		s.fullScan(maxInsertIndex, buf, callback)
		return
	}
	if !s.active {
		// Index was evicted between our initial check and here.
		s.mu.Unlock()
		s.fullScan(maxInsertIndex, buf, callback)
		return
	}
	snapMainIndexes := s.mainIndexes
	var snapDeltaBtree *btree.BTreeG[indexPair]
	if s.deltaBtree != nil {
		// Clone under the index lock so scans can iterate a stable delta snapshot
		// while concurrent inserts keep mutating the live tree.
		snapDeltaBtree = s.deltaBtree.Clone()
	}
	isNative := s.Native
	s.mu.Unlock()

	// Produce per-column matcher data (cached across queries with same pattern).
	var matcherData []BoundaryMatcherData
	if len(queryMatchers) > 0 {
		matcherData = make([]BoundaryMatcherData, len(queryMatchers))
		for i, qm := range queryMatchers {
			matcherData[i] = qm.ProduceData(s, i, lower[i], uppers[i])
		}
	}

	// record-ID lookup: identity when data is physically sorted (Native), index dereference otherwise
	getRecid := func(idx int) uint32 {
		return uint32(int64(snapMainIndexes.GetValueUInt(uint32(idx))) + snapMainIndexes.offset)
	}
	if isNative {
		getRecid = func(idx int) uint32 { return uint32(idx) }
	}

	// bisect where the lower bound is found
	// Only compare as many columns as provided in 'lower' (index can have more cols)
	cmpCols := len(lower)
	// Use last-hit hint to narrow binary search range (helps sorted outer loops).
	// The hint is advisory: if stale or from a concurrent goroutine, we safely
	// fall through to an unnarrowed search. No correctness dependency on the hint.
	// LIKE columns cannot participate in binary search (pattern doesn't map to sort order).
	searchLo := 0
	searchN := int(s.t.main_count)
	firstSearchable := len(queryMatchers) == 0 || queryMatchers[0].IsSorted()
	if hint := int(s.lastHit.Load()); hint > 0 && hint < searchN && cmpCols > 0 && firstSearchable && !lower[0].IsNil() {
		hintVal := cols[0].get(getRecid(hint))
		if !hintVal.IsNil() {
			if scm.Less(hintVal, lower[0]) {
				searchLo = hint
				searchN -= hint
			} else if scm.Less(lower[0], hintVal) {
				searchN = hint + 1
			}
		}
	}
	mainIdx := searchLo + sort.Search(searchN, func(idx int) bool {
		idx2 := getRecid(searchLo + idx)
		for i := 0; i < cmpCols; i++ {
			if len(queryMatchers) > i && !queryMatchers[i].IsSorted() {
				continue // non-sorted matcher cols: no binary search, filter later
			}
			a := lower[i]
			b := cols[i].get(idx2)
			if scm.Less(a, b) {
				return true // less
			} else if scm.Less(b, a) {
				return false // greater
			}
			// otherwise: next iteration
		}
		return true // fully equal
	})
	s.lastHit.Store(uint32(mainIdx))
	// skip past equal values when lower bound is exclusive (col > 5)
	// LIKE columns don't have lower/upper semantics, so skip this optimization.
	lastHasMatcher := len(queryMatchers) >= cmpCols && !queryMatchers[cmpCols-1].IsSorted()
	if !lowerInclusive && !lastHasMatcher && cmpCols > 0 && !lower[cmpCols-1].IsNil() {
		for uint32(mainIdx) < s.t.main_count {
			recid := getRecid(mainIdx)
			if !scm.Equal(cols[cmpCols-1].get(recid), lower[cmpCols-1]) {
				break
			}
			mainIdx++
		}
	}

	nextMain := func() (uint32, bool) {
		for {
			if uint32(mainIdx) >= s.t.main_count {
				return 0, false
			}
			recid := getRecid(mainIdx)
			mainIdx++
			inRange, beyond := s.rowWithinBounds(cmpCols, matcherData, func(i int) scm.Scmer {
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
			if !callback(buf[:bufN]) {
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
			inRange, beyond := s.rowWithinBounds(cmpCols, matcherData, func(i int) scm.Scmer {
				return s.getDeltaColValue(p.data, i)
			})
			if !inRange {
				return !beyond
			}
			// drain main items that sort before this delta item
			for mainOk && !stopped {
				cmp := s.compareMainAndDelta(mainRecid, cols, p)
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
		// don't map to sort order), so scan all.
		hasUnsearchableInBounds := false
		for i := 0; i < cmpCols; i++ {
			if (len(s.ColMapFn) > i && !s.ColMapFn[i].IsNil()) || (len(queryMatchers) > i && !queryMatchers[i].IsSorted()) {
				hasUnsearchableInBounds = true
				break
			}
		}
		if hasUnsearchableInBounds {
			snapDeltaBtree.Ascend(iterFn)
		} else {
			maxCol := 0
			for _, col := range s.Cols {
				if pos, ok := s.t.deltaColumns[col]; ok && pos+1 > maxCol {
					maxCol = pos + 1
				}
			}
			refLower := make([]scm.Scmer, maxCol)
			for i := 0; i < cmpCols; i++ {
				col := s.Cols[i]
				if pos, ok := s.t.deltaColumns[col]; ok {
					refLower[pos] = lower[i]
				}
			}
			snapDeltaBtree.AscendGreaterOrEqual(indexPair{-1, refLower}, iterFn)
		}
	}

	// drain remaining main items
	for mainOk && !stopped {
		emit(mainRecid)
		mainRecid, mainOk = nextMain()
	}
	if bufN > 0 && !stopped {
		callback(buf[:bufN])
	}
}

// indexCleanup is called by the CacheManager when evicting an index.
// Returns false if the index lock cannot be acquired (non-blocking).
func indexCleanup(ptr any, freedByType *[numEvictableTypes]int64) bool {
	idx := ptr.(*StorageIndex)
	if !idx.mu.TryLock() {
		return false // index is in use, skip eviction
	}
	idx.active = false
	idx.mainIndexes = StorageInt{}
	idx.deltaBtree = nil
	idx.matcherDataCache = nil
	idx.mu.Unlock()
	return true
}

func indexLastUsed(ptr any) time.Time {
	// use the parent shard's lastAccessed as proxy
	return time.Unix(0, int64(atomic.LoadUint64(&ptr.(*StorageIndex).t.lastAccessed)))
}

func indexGetScore(ptr any) float64 {
	return ptr.(*StorageIndex).Savings
}
