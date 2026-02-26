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
	deltaBtree  *btree.BTreeG[indexPair]
	t           *storageShard
	active      bool
	mu          sync.Mutex
}

/*
 TODO: n-ary heap btree implementation for better cache exploitation

 - len(tree) = len(data)
 - completely pointerless
 - each node has up to 32 items
 - all nodes except the last node are fully occupied
 - the first item is guaranteed to be the smallest item
 - the next smallest item is its first child
 - if the node is a leaf, the next smallest item is its sibling
 - if the item is the last in the node, the next smallest item is the sibling of its parent
 - forward iteration algorithm:
    if 32 * i < len(heap) {
	    // visit child
	    i = 32 * i
    } else {
	    // move to next sibling
	    i = i + 1
	    while i % 32 == 0 {
		    // end of node -> go to parent's next
		    i = i / 32
		    if i == 0 {
			    return "finished"
		    }
	    }
    }
 - backward iteration algorithm:
    if i == 0 {
            return "finished"
    }
    if i % 32 == 0 {
            // end of node -> go to parent's next
            i = i / 32
    } else {
	    // move to next sibling
	    i = i - 1
	    while 32 * i + 31 < len(heap) {
		    // visit highest child
		    i = 32 * i + 31
	    }
    }
  - bisect algorithm: given a searchvalue, find the lowest index where all items below that point are less than searchvalue
     - start with left=0, right=min(32, len(heap))
     - pivot = (left+right)/2
     - if Less(item[pivot], searchvalue) { left = pivot } else { right = pivot }
     - if left = right:
       - if not left*32 <= len(heap): return left
       - if left == 0: return 0
       - left = (left-1) * 32, right = left + 32 // search items left from left that might be smaller
       - recurse
  - bisect for a greater-equal variant (find the highest index where all items above are not less than searchvalue)

*/

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

// iterate over index using a caller-provided buffer for batching
func (s *StorageIndex) iterate(lower []scm.Scmer, upperLast scm.Scmer, maxInsertIndex int, buf []uint32, callback func([]uint32) bool) {

	// find columns in storage
	cols := make([]ColumnStorage, len(s.Cols))
	for i, c := range s.Cols {
		cols[i] = s.t.getColumnStorageOrPanic(c)
	}
	// no collation-specific helpers in the current implementation

	savings_threshold := 2.0    // building an index costs 1x the time as traversing the list
	s.Savings = s.Savings + 1.0 // mark that we could save time
	if !s.active {
		// index is not built yet
		if s.Savings < savings_threshold {
			// iterate over all items because we don't want to store the index
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

	// bisect where the lower bound is found
	// Only compare as many columns as provided in 'lower' (index can have more cols)
	cmpCols := len(lower)
	mainIdx := sort.Search(int(s.t.main_count), func(idx int) bool {
		idx2 := uint32(int64(s.mainIndexes.GetValueUInt(uint32(idx))) + s.mainIndexes.offset)
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
			recid := uint32(int64(s.mainIndexes.GetValueUInt(uint32(mainIdx))) + s.mainIndexes.offset)
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

	deltaItems := make([]indexPair, 0)
	if maxInsertIndex > 0 && s.deltaBtree != nil {
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
		s.deltaBtree.AscendGreaterOrEqual(indexPair{-1, refLower}, func(p indexPair) bool {
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
			if inRange {
				deltaItems = append(deltaItems, p)
			}
			return !beyond
		})
	}

	bufN := 0
	if len(deltaItems) == 0 {
		for recid, ok := nextMain(); ok; recid, ok = nextMain() {
			buf[bufN] = recid
			bufN++
			if bufN == len(buf) {
				if !callback(buf[:bufN]) {
					return
				}
				bufN = 0
			}
		}
		if maxInsertIndex > 0 && s.deltaBtree == nil {
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
		}
		if bufN > 0 {
			callback(buf[:bufN])
		}
		return
	}

	di := 0
	mainRecid, mainOk := nextMain()
	for mainOk || di < len(deltaItems) {
		var id uint32
		if !mainOk {
			id = uint32(deltaItems[di].itemid)
			di++
		} else if di >= len(deltaItems) {
			id = mainRecid
			mainRecid, mainOk = nextMain()
		} else {
			cmp := s.compareMainAndDelta(mainRecid, cols, deltaItems[di])
			if cmp <= 0 {
				id = mainRecid
				mainRecid, mainOk = nextMain()
			} else {
				id = uint32(deltaItems[di].itemid)
				di++
			}
		}
		buf[bufN] = id
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
