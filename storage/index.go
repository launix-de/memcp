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

type StorageIndex struct {
	Cols        []string   // sort equal-cols alphabetically, so similar conditions are canonical
	Savings     float64    // store the amount of time savings here -> add selectivity (outputted / size) on each
	Native      bool       // true when data is physically sorted by this index (zero-cost)
	mainIndexes StorageInt // we can do binary searches here (unused when Native)
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

func (idx *StorageIndex) ComputeSize() uint {
	var sz uint = 24 * 8 // heuristic
	if !idx.Native {
		sz += idx.mainIndexes.ComputeSize()
	}
	// TODO: deltaBtree
	return sz
}

// pretty-print
func (idx *StorageIndex) String() string {
	return strings.Join(idx.Cols, "|")
}

// iterates over items
func (s *StorageIndex) getDeltaValue(data []scm.Scmer, col string) scm.Scmer {
	colpos, ok := s.t.deltaColumns[col]
	if ok && colpos < len(data) {
		return data[colpos]
	}
	return scm.NewNil()
}

func (s *StorageIndex) rowWithinBounds(cmpCols int, lower []scm.Scmer, upperLast scm.Scmer, upperInclusive bool, getter func(int) scm.Scmer) (inRange bool, beyond bool) {
	for i := 0; i < cmpCols; i++ {
		v := getter(i)
		if i == cmpCols-1 {
			if !upperLast.IsNil() {
				if upperInclusive {
					if scm.Less(upperLast, v) {
						return false, true
					}
				} else if !scm.Less(v, upperLast) {
					return false, true
				}
			}
			continue
		}
		if scm.Equal(v, lower[i]) {
			continue
		}
		if scm.Less(v, lower[i]) {
			return false, false
		}
		return false, true
	}
	return true, false
}

func (s *StorageIndex) compareMainAndDelta(mainRecid uint32, mainCols []ColumnStorage, delta indexPair) int {
	for i, col := range s.Cols {
		mainVal := mainCols[i].GetValue(mainRecid)
		deltaVal := s.getDeltaValue(delta.data, col)
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
					if cols[i].col != index.Cols[i] {
						goto skip_index // this index does not fit
					}
				}
				// this index fits!
				index.iterate(lower, upperLast, lowerIncl, upperIncl, maxInsertIndex, buf, callback)
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
					if cols[i].col != index.Cols[i] {
						covered = false
						break
					}
				}
				if covered {
					// longer index covers this query; use it instead of creating a shorter one
					t.indexMutex.Unlock()
					index.iterate(lower, upperLast, lowerIncl, upperIncl, maxInsertIndex, buf, callback)
					return
				}
			}
		}
		index := new(StorageIndex)
		index.Cols = make([]string, len(lower))
		for i := range lower {
			index.Cols[i] = cols[i].col
		}
		index.Savings = 0.0  // count how many cost we wasted so we decide when to build the index
		index.active = false // tell the engine that index has to be built first
		index.t = t
		t.Indexes = append(t.Indexes, index)
		t.indexMutex.Unlock()
		index.iterate(lower, upperLast, lowerIncl, upperIncl, maxInsertIndex, buf, callback)
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
// cols must contain column storages for each index column in order.
// The caller must hold s.mu.Lock() or have exclusive access.
func (s *StorageIndex) buildIndex(cols []ColumnStorage) {
	fmt.Println("building index on", s.t.t.Name, "over", s.Cols)

	if !s.Native {
		// main storage: build sort-order index
		tmp := make([]uint32, s.t.main_count)
		for i := uint32(0); i < s.t.main_count; i++ {
			tmp[i] = i // fill with natural order
		}
		// sort indexes
		sort.Slice(tmp, func(i, j int) bool {
			for _, c := range cols {
				a := c.GetValue(tmp[i])
				b := c.GetValue(tmp[j])
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

	// delta storage
	s.deltaBtree = btree.NewG[indexPair](8, func(i, j indexPair) bool {
		for _, col := range s.Cols {
			colpos, ok := s.t.deltaColumns[col]
			if !ok {
				continue // non-existing column -> don't compare
			}
			var a, b scm.Scmer
			if colpos < len(i.data) {
				a = i.data[colpos]
			}
			if colpos < len(j.data) {
				b = j.data[colpos]
			}
			if scm.Less(a, b) {
				return true // less
			} else if !scm.Equal(a, b) {
				return false // greater
			}
			// otherwise: next iteration
		}
		// tiebreak by itemid so duplicate key values are never "equal"
		// (prevents ReplaceOrInsert from dropping rows with same key)
		return i.itemid < j.itemid
	})
	// fill deltaBtree with global record IDs
	for i, data := range s.t.inserts {
		s.deltaBtree.ReplaceOrInsert(indexPair{int(s.t.main_count) + i, data})
	}

	s.active = true // mark as ready
}

// iterate over index using a caller-provided buffer for batching
func (s *StorageIndex) iterate(lower []scm.Scmer, upperLast scm.Scmer, lowerInclusive bool, upperInclusive bool, maxInsertIndex int, buf []uint32, callback func([]uint32) bool) {

	// find columns in storage — use RLocked variant because the caller
	// (scan, scan_order, GetRecordidForUnique) already holds s.t.mu.RLock().
	// Re-acquiring RLock via getColumnStorageOrPanic would deadlock when a
	// concurrent writer is waiting for s.t.mu.Lock() (write-preferring RWMutex).
	cols := make([]ColumnStorage, len(s.Cols))
	for i, c := range s.Cols {
		cols[i] = s.t.getColumnStorageRLocked(c)
	}
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
			// rebuild index
			s.mu.Lock()
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
	s.mu.Lock()
	if !s.active {
		// Index was evicted between our initial check and here.
		s.mu.Unlock()
		s.fullScan(maxInsertIndex, buf, callback)
		return
	}
	snapMainIndexes := s.mainIndexes
	snapDeltaBtree := s.deltaBtree
	isNative := s.Native
	s.mu.Unlock()

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
	// Use last-hit hint to narrow binary search range (helps sorted outer loops)
	searchLo := 0
	searchN := int(s.t.main_count)
	if hint := int(s.lastHit.Load()); hint > 0 && hint < searchN && cmpCols > 0 {
		hintVal := cols[0].GetValue(getRecid(hint))
		if scm.Less(hintVal, lower[0]) {
			searchLo = hint
			searchN -= hint
		} else if scm.Less(lower[0], hintVal) {
			searchN = hint + 1
		}
	}
	mainIdx := searchLo + sort.Search(searchN, func(idx int) bool {
		idx2 := getRecid(searchLo + idx)
		for i := 0; i < cmpCols; i++ {
			a := lower[i]
			b := cols[i].GetValue(idx2)
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
	if !lowerInclusive && cmpCols > 0 && !lower[cmpCols-1].IsNil() {
		for uint32(mainIdx) < s.t.main_count {
			recid := getRecid(mainIdx)
			if !scm.Equal(cols[cmpCols-1].GetValue(recid), lower[cmpCols-1]) {
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
			inRange, beyond := s.rowWithinBounds(cmpCols, lower, upperLast, upperInclusive, func(i int) scm.Scmer {
				return cols[i].GetValue(recid)
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
		snapDeltaBtree.AscendGreaterOrEqual(indexPair{-1, refLower}, func(p indexPair) bool {
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
			inRange, beyond := s.rowWithinBounds(cmpCols, lower, upperLast, upperInclusive, func(i int) scm.Scmer {
				return s.getDeltaValue(p.data, s.Cols[i])
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
		})
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
	idx.mu.Unlock()
	return true
}

func indexLastUsed(ptr any) time.Time {
	// use the parent shard's lastAccessed as proxy
	return ptr.(*StorageIndex).t.lastAccessed
}

func indexGetScore(ptr any) float64 {
	return ptr.(*StorageIndex).Savings
}
