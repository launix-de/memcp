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
import "reflect"
import "github.com/google/btree"
import "github.com/launix-de/memcp/scm"

type indexPair struct {
	itemid int // -1 for reference items
	data []scm.Scmer
}

type StorageIndex struct {
	cols []string // sort equal-cols alphabetically, so similar conditions are canonical
	savings float64 // store the amount of time savings here -> add selectivity (outputted / size) on each
	mainIndexes StorageInt // we can do binary searches here
	deltaBtree *btree.BTreeG[indexPair]
	t *storageShard
	inactive bool
}

// iterates over items
func (t *storageShard) iterateIndex(cols boundaries, maxInsertIndex int) chan uint {

	// check if we found conditions
	if len(cols) > 0 {
		//fmt.Println("conditions:", cols)
		// sort columns -> at first, the lower==upper alphabetically; then one lower!=upper according to best selectivity; discard the rest
		sort.Slice(cols, func (i, j int) bool {
			if cols[i].lower == cols[i].upper && cols[j].lower != cols[j].upper {
				return true // put equal?-conditions leftmost
			}
			return cols[i].col < cols[j].col // otherwise: alphabetically
		})

		// build up lower and upper bounds of index
		for {
			if len(cols) >= 2 && cols[len(cols)-2].lower != cols[len(cols)-2].upper {
				// remove last col -> we cant have two ranged cols
				cols = cols[:len(cols)-1]
			} else {
				break // finished -> pure index
			}
		}
		// find out boundaries
		lower := make([]scm.Scmer, len(cols))
		for i, v := range cols {
			lower[i] = v.lower
		}
		upperLast := cols[len(cols)-1].upper
		//fmt.Println(cols, lower, upperLast) // debug output if we found the right boundaries

		// find an index that has at least the columns in that order we're searching for
		// if the index is inactive, use the other one
		for _, index := range t.indexes {
			// naive index search algo; TODO: improve
			if len(index.cols) >= len(cols) {
				for i := 0; i < len(cols); i++ {
					if cols[i].col != index.cols[i] {
						continue // this index does not fit
					}
				}
				// this index fits!
				return index.iterate(lower, upperLast, maxInsertIndex)
			}
		}

		// otherwise: create new index
		index := new(StorageIndex)
		index.cols = make([]string, len(cols))
		for i, v := range cols {
			index.cols[i] = v.col
		}
		index.savings = 0.0 // count how many cost we wasted so we decide when to build the index
		index.inactive = true // tell the engine that index has to be built first
		index.t = t
		t.indexes = append(t.indexes, index)
		return index.iterate(lower, upperLast, maxInsertIndex)
	}

	// otherwise: iterate over all items
	result := make(chan uint, 64)
	go func() {
		for i := uint(0); i < t.main_count; i++ {
			result <- i
		}
		for i := 0; i < maxInsertIndex; i++ {
			result <- t.main_count + uint(i)
		}
		close(result)
	}()
	return result
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
func (s *StorageIndex) iterate(lower []scm.Scmer, upperLast scm.Scmer, maxInsertIndex int) chan uint {
	result := make(chan uint, 64)

	// find columns in storage
	cols := make([]ColumnStorage, len(s.cols))
	for i, c := range s.cols {
		cols[i] = s.t.columns[c]
	}

	savings_threshold := 2.0 // building an index costs 1x the time as traversing the list
	s.savings = s.savings + 1.0 // mark that we could save time
	go func() {
		if s.inactive {
			// index is not built yet
			if s.savings < savings_threshold {
				// iterate over all items because we don't want to store the index
				for i := uint(0); i < s.t.main_count; i++ {
					result <- i
				}
				close(result)
				return
			} else {
				// rebuild index
				fmt.Println("building index on", s.t.t.Name, "over", s.cols)

				// main storage
				tmp := make([]uint, s.t.main_count)
				for i := uint(0); i < s.t.main_count; i++ {
					tmp[i] = i // fill with natural order
				}
				// sort indexes
				sort.Slice(tmp, func (i, j int) bool {
					for _, c := range cols {
						a := c.GetValue(tmp[i])
						b := c.GetValue(tmp[j])
						if scm.Less(a, b) {
							return true // less
						} else if !reflect.DeepEqual(a, b) {
							return false // greater
						}
						// otherwise: next iteration
					}
					return false // fully equal
				})
				// store sorted values into compressed format
				s.mainIndexes.prepare()
				for i, v := range tmp {
					s.mainIndexes.scan(uint(i), v)
				}
				s.mainIndexes.init(uint(len(tmp)))
				for i, v := range tmp {
					s.mainIndexes.build(uint(i), v)
				}
				s.mainIndexes.finish()

				// delta storage
				s.deltaBtree = btree.NewG[indexPair](8, func (i, j indexPair) bool {
					for _, col := range s.cols {
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
						} else if !reflect.DeepEqual(a, b) {
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

				s.inactive = false // mark as ready
			}
		}

		// bisect where the lower bound is found
		idx := sort.Search(int(s.t.main_count), func (idx int) bool {
			idx2 := uint(int64(s.mainIndexes.GetValueUInt(uint(idx))) + s.mainIndexes.offset)
			for i, c := range cols {
				a := lower[i]
				b := c.GetValue(uint(idx2))
				if scm.Less(a, b) {
					return true // less
				} else if !reflect.DeepEqual(a, b) {
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
			if uint(idx) >= s.t.main_count {
				break
			}
			idx2 := uint(int64(s.mainIndexes.GetValueUInt(uint(idx))) + s.mainIndexes.offset)
			// check for index bounds
			for i, c := range cols {
				a := c.GetValue(uint(idx2))
				if i == len(cols) - 1 {
					if scm.Less(upperLast, a) {
						break iteration // stop traversing when we exceed the < part of last col
					}
				} else if !reflect.DeepEqual(a, lower[i]) {
					break iteration // stop traversing when we exceed the equal-part
				}
				// otherwise: next col
			}
			// output recordid
			result <- uint(idx2)
			idx++
			// TODO: stop on limit
		}

		// delta storage -> scan btree (but we can also eject all items, it won't break the code)
		if len(s.t.inserts) > 0 { // avoid building objects if there is no delta
			delta_lower := make(dataset, 2 * len(s.cols))
			delta_upper := make(dataset, 2 * len(s.cols))
			for i := 0; i < len(s.cols); i++ {
				delta_lower[2 * i] = s.cols[i]
				delta_lower[2 * i + 1] = lower[i]
				delta_upper[2 * i] = s.cols[i]
				delta_upper[2 * i + 1] = lower[i]
			}
			delta_upper[len(delta_upper)-1] = upperLast
			// scan less than
			s.deltaBtree.AscendRange(indexPair{-1, delta_lower}, indexPair{-1, delta_upper}, func (p indexPair) bool {
				result <- s.t.main_count + uint(p.itemid)
				return true // don't stop iteration
				// TODO: stop on limit
			})
			// find exact fit, too
			if p, ok := s.deltaBtree.Get(indexPair{-1, delta_upper}); ok {
				result <- s.t.main_count + uint(p.itemid)
			}
			/*for i := 0; i < maxInsertIndex; i++ {
				result <- s.t.main_count + uint(i)
			}*/
		}
		close(result)
	}()
	return result
}
