/*
Copyright (C) 2023  Carl-Philip Hänsch

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
func (t *storageShard) iterateIndex(cols boundaries, lower []scm.Scmer, upperLast scm.Scmer, maxInsertIndex int, callback func(uint32)) {
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
				index.iterate(lower, upperLast, maxInsertIndex, callback)
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
		index.iterate(lower, upperLast, maxInsertIndex, callback)
		return
	}

	// otherwise: iterate over all items
	for i := uint32(0); i < t.main_count; i++ {
		callback(i)
	}
	for i := 0; i < maxInsertIndex; i++ {
		callback(t.main_count + uint32(i))
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

// iterate over index
func (s *StorageIndex) iterate(lower []scm.Scmer, upperLast scm.Scmer, maxInsertIndex int, callback func(uint32)) {

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
			for i := uint32(0); i < s.t.main_count; i++ {
				callback(i)
			}
			for i := 0; i < maxInsertIndex; i++ {
				callback(s.t.main_count + uint32(i))
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
				return false // fully equal
			})
			// fill deltaBtree (no locking required; we are already in a readlock)
			for i, data := range s.t.inserts {
				s.deltaBtree.ReplaceOrInsert(indexPair{i, data})
			}

			s.active = true // mark as ready
			s.mu.Unlock()
		}
	}
start_scan:

	// bisect where the lower bound is found
	// Only compare as many columns as provided in 'lower' (index can have more cols)
	cmpCols := len(lower)
	idx := sort.Search(int(s.t.main_count), func(idx int) bool {
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
	// now iterate over all items as long as we stay in sync
iteration:
	for {
		// check for array bounds
		if uint32(idx) >= s.t.main_count {
			break
		}
		idx2 := uint32(int64(s.mainIndexes.GetValueUInt(uint32(idx))) + s.mainIndexes.offset)
		// check for index bounds
		for i := 0; i < cmpCols; i++ {
			a := cols[i].GetValue(idx2)
			if i == cmpCols-1 {
				if !upperLast.IsNil() && scm.Less(upperLast, a) { // TODO: respect !upperEqual
					break iteration // stop traversing when we exceed the < part of last col
				}
			} else if !scm.Equal(a, lower[i]) {
				break iteration // stop traversing when we exceed the equal-part
			}
			// otherwise: next col
		}
		// TODO: merge with delta btree in order to preserve index order
		// output recordid
		callback(idx2)
		idx++
		// TODO: stop on limit
	}

	// delta storage -> scan btree (but we can also eject all items, it won't break the code)
	if len(s.t.inserts) > 0 { // avoid building objects if there is no delta
		/* TODO: use our own compressed delta Bheap-tree
		delta_lower := make(dataset, 2 * len(s.Cols))
		delta_upper := make(dataset, 2 * len(s.Cols))
		for i := 0; i < len(s.Cols); i++ {
			delta_lower[2 * i] = s.Cols[i]
			delta_lower[2 * i + 1] = lower[i]
			delta_upper[2 * i] = s.Cols[i]
			delta_upper[2 * i + 1] = lower[i]
		}
		delta_upper[len(delta_upper)-1] = upperLast
		// scan less than
		s.deltaBtree.AscendRange(indexPair{-1, delta_lower}, indexPair{-1, delta_upper}, func (p indexPair) bool {
			callback(s.t.main_count + uint(p.itemid))
			return true // don't stop iteration
			// TODO: stop on limit
		})
		// find exact fit, too
		if p, ok := s.deltaBtree.Get(indexPair{-1, delta_upper}); ok {
			callback(s.t.main_count + uint(p.itemid))
		}
		*/
		// fallback: output all items
		for i := 0; i < maxInsertIndex; i++ {
			callback(s.t.main_count + uint32(i))
		}
	}
}
