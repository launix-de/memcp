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
	mainIndexes StorageInt // we can do binary searches here
	// deltaBtree holds delta inserts in index-column order. Contract:
	// when active==true, deltaBtree is non-nil. It is built during index
	// construction and kept in sync by shard.insertDataset on every insert.
	// Deleted rows are intentionally kept in the btree — the scan layer
	// filters them via t.deletions / transaction visibility overlays,
	// because a DELETE can be rolled back by a concurrent transaction.
	deltaBtree *btree.BTreeG[indexPair]
	t          *storageShard
	active     bool
	mu         sync.Mutex
}

func (idx *StorageIndex) ComputeSize() uint {
	var sz uint = 24 * 8 // heuristic
	sz += idx.mainIndexes.ComputeSize()
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

func (s *StorageIndex) rowWithinBounds(cmpCols int, lower []scm.Scmer, upperLast scm.Scmer, getter func(int) scm.Scmer) (inRange bool, beyond bool) {
	for i := 0; i < cmpCols; i++ {
		v := getter(i)
		if i == cmpCols-1 {
			if !upperLast.IsNil() && scm.Less(upperLast, v) {
				return false, true
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
				index.iterate(lower, upperLast, maxInsertIndex, buf, callback)
				return
			}
		skip_index:
		}

		// otherwise: create new index
		t.indexMutex.Lock()
		if len(old_indexes) != len(t.Indexes) {
			t.indexMutex.Unlock()
			goto retry_indexscan // someone has added a index in the meantime: recheck
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
		index.iterate(lower, upperLast, maxInsertIndex, buf, callback)
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

func rebuildIndexes(t1 *storageShard, t2 *storageShard) {
	// TODO rebuild index in database rebuild
	// check if indexes share same prefix -> leave out the shorter one
	// savings = 0.9 * savings (decrease)
	// according to memory pressure -> threshold for discard savings
	// -> mark inactive if we can don't want to store this index
	// if two indexes are prefixed, give up the shorter one and add to savings
	// (also consider incremental indexes??)
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

// iterate over index using a caller-provided buffer for batching
func (s *StorageIndex) iterate(lower []scm.Scmer, upperLast scm.Scmer, maxInsertIndex int, buf []uint32, callback func([]uint32) bool) {

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
			fmt.Println("building index on", s.t.t.Name, "over", s.Cols)

			// main storage
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
			// fill deltaBtree with global record IDs (no locking required; we are already in a readlock)
			for i, data := range s.t.inserts {
				s.deltaBtree.ReplaceOrInsert(indexPair{int(s.t.main_count) + i, data})
			}

			s.active = true // mark as ready
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
	s.mu.Unlock()

	// bisect where the lower bound is found
	// Only compare as many columns as provided in 'lower' (index can have more cols)
	cmpCols := len(lower)
	mainIdx := sort.Search(int(s.t.main_count), func(idx int) bool {
		idx2 := uint32(int64(snapMainIndexes.GetValueUInt(uint32(idx))) + snapMainIndexes.offset)
		for i := 0; i < cmpCols; i++ {
			a := lower[i]
			b := cols[i].GetValue(idx2)
			// TODO: respect !lowerEqual
			if scm.Less(a, b) {
				return true // less
			} else if scm.Less(b, a) {
				return false // greater
			}
			// otherwise: next iteration
		}
		return true // fully equal
	})

	nextMain := func() (uint32, bool) {
		for {
			if uint32(mainIdx) >= s.t.main_count {
				return 0, false
			}
			recid := uint32(int64(snapMainIndexes.GetValueUInt(uint32(mainIdx))) + snapMainIndexes.offset)
			mainIdx++
			inRange, beyond := s.rowWithinBounds(cmpCols, lower, upperLast, func(i int) scm.Scmer {
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
			inRange, beyond := s.rowWithinBounds(cmpCols, lower, upperLast, func(i int) scm.Scmer {
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
