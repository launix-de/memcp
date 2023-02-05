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
import "github.com/launix-de/memcp/scm"

type StorageIndex struct {
	cols []string // sort equal-cols alphabetically, so similar conditions are canonical
	savings float64 // store the amount of time savings here -> add selectivity (outputted / size) on each
	sortedItems StorageInt // we can do binary searches here
	t *storageShard
	inactive bool
}

type columnboundaries struct{
	col string
	lower scm.Scmer
	upper scm.Scmer
}

func (t *storageShard) iterateIndex(condition scm.Scmer) chan uint {
	cols := make([]columnboundaries, 0, 4)
	// analyze condition for AND clauses, equal? < > <= >= BETWEEN
	traverseCondition := func (condition scm.Scmer) {
		switch v := condition.(scm.Proc).Body.(type) {
			case []scm.Scmer:
				if v[0] == scm.Symbol("equal?") {
					switch v1 := v[1].(type) {
						case scm.Symbol:
							switch v2 := v[2].(type) {
								case float64, string:
									// equals column vs. constant
									cols = append(cols, columnboundaries{string(v1), v2, v2})
							}
						// TODO: equals constant vs. column
					}
				}
				// TODO: AND -> recursive traverse
				// TODO: OR -> merge multiple
		}
	}
	traverseCondition(condition) // recursive analysis over condition

	// check if we found conditions
	if len(cols) > 0 {
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
				return index.iterate(lower, upperLast)
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
		return index.iterate(lower, upperLast)
	}

	// otherwise: iterate over all items
	result := make(chan uint, 64)
	go func() {
		for i := uint(0); i < t.main_count; i++ {
			result <- i
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

// sort function for scmer
func scmerLess(a, b scm.Scmer) bool {
	// TODO: 2D check if NULL etc.
	switch a.(type) {
		case float64:
			return a.(float64) < b.(float64)
		case string:
			return a.(string) < b.(string)
		// are there any other types??
	}
	return false
}

// iterate over index
func (s *StorageIndex) iterate(lower []scm.Scmer, upperLast scm.Scmer) chan uint {
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
				fmt.Println("building index on", s.t.t.name, "over", s.cols)
				tmp := make([]uint, s.t.main_count)
				for i := uint(0); i < s.t.main_count; i++ {
					tmp[i] = i // fill with natural order
				}
				// sort indexes
				sort.Slice(tmp, func (i, j int) bool {
					for _, c := range cols {
						a := c.getValue(tmp[i])
						b := c.getValue(tmp[j])
						if scmerLess(a, b) {
							return true // less
						} else if !reflect.DeepEqual(a, b) {
							return false // greater
						}
						// otherwise: next iteration
					}
					return false // fully equal
				})
				// store sorted values into compressed format
				s.sortedItems.prepare()
				for i, v := range tmp {
					s.sortedItems.scan(uint(i), v)
				}
				s.sortedItems.init(uint(len(tmp)))
				for i, v := range tmp {
					s.sortedItems.build(uint(i), v)
				}
				s.sortedItems.finish()
				s.inactive = false // mark as ready
			}
		}
		/* code for go 1.19
		i, found := sort.Find(s.t.main_count, func (idx int) int {
			for i, c := range cols {
				a := lower[i]
				b := c.getValue(uint(idx))
				if scmerLess(a, b) {
					return -1 // less
				} else if !reflect.DeepEqual(a, b) {
					return 1 // greater
				}
				// otherwise: next iteration
			}
			return 0 // fully equal
		})
		*/
		// bisect where the lower bound is found
		idx := sort.Search(int(s.t.main_count), func (idx int) bool {
			idx2 := s.sortedItems.getValueUInt(uint(idx))
			for i, c := range cols {
				a := lower[i]
				b := c.getValue(uint(idx2))
				if scmerLess(a, b) {
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
			idx2 := s.sortedItems.getValueUInt(uint(idx))
			// check for index bounds
			for i, c := range cols {
				a := c.getValue(uint(idx2))
				if i == len(cols) - 1 {
					if scmerLess(upperLast, a) {
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
		}
		close(result)
	}()
	return result
}
