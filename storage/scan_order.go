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
import "runtime/debug"
import "container/heap"
import "github.com/launix-de/memcp/scm"

type shardqueue struct {
	shard *storageShard
	items []uint // TODO: refactor to chan, so we can block generating too much entries
	err scanError
	mcols []func(uint) scm.Scmer // map column reader
	scols []func(uint) scm.Scmer // sort criteria column reader
	sortdirs []bool
}

// sort interface for shardqueue (local) (TODO: heap could be more efficient because early-out will be cheaper)
func (s *shardqueue) Len() int {
	return len(s.items)
}
func (s *shardqueue) Less(i, j int) bool {
	for c := 0; c < len(s.scols); c++ {
		comparison := scm.Compare(s.scols[c](s.items[i]), s.scols[c](s.items[j]))
		if (comparison < 0) != s.sortdirs[c] {
			return true
		}
		if (comparison > 0) != s.sortdirs[c] {
			return false
		}
		// otherwise: move on to c++
	}
	return false // equal is not less
}
func (s *shardqueue) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}

type globalqueue struct {
	q []*shardqueue
}

// sort interface for global shard-queue
func (s *globalqueue) Len() int {
	return len(s.q)
}
func (s *globalqueue) Less(i, j int) bool {
	for c := 0; c < len(s.q[i].scols); c++ {
		comparison := scm.Compare(s.q[i].scols[c](s.q[i].items[0]), s.q[j].scols[c](s.q[j].items[0]))
		if (comparison < 0) != s.q[i].sortdirs[c] {
			return true
		}
		if (comparison > 0) != s.q[i].sortdirs[c] {
			return false
		}
		// otherwise: move on to c++
	}
	return false // equal is not less
}
func (s *globalqueue) Swap(i, j int) {
	s.q[i], s.q[j] = s.q[j], s.q[i]
}
func (s *globalqueue) Push(x_ any) {
	x := x_.(*shardqueue)
	s.q = append(s.q, x)
}
func (s *globalqueue) Pop() any {
	result := s.q[len(s.q)-1]
	s.q[len(s.q)-1] = nil // already free the memory, so GC can also run during an uncompleted ordered scan
	s.q = s.q[0:len(s.q)-1]
	return result
}


// TODO: helper function for priority-q. golangs implementation is kinda quirky, so do our own. container/heap especially lacks the function to test the value at front instead of popping it

// map reduce implementation based on scheme scripts
func (t *table) scan_order(conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []bool, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer) scm.Scmer {

	/* analyze condition query */
	boundaries := extractBoundaries(conditionCols, condition)
	// TODO: append sortcols to boundaries

	// TODO: sortcols that are not just simple columns but complex lambda expressions could be temporarily materialized to trade memory for execution time
	// --> sortcols can then be rewritten to strings

	callbackFn := scm.OptimizeProcToSerialFunction(callback)
	aggregateFn := func(...scm.Scmer) scm.Scmer {return nil}
	if aggregate != nil {
		aggregateFn = scm.OptimizeProcToSerialFunction(aggregate)
	}

	// total_limit helps the shard-scanners to early-out
	total_limit := -1
	if limit >= 0 {
		total_limit = offset + limit
	}

	var q globalqueue
	q_ := make(chan *shardqueue, 1)
	rest := 0
	for _, s := range t.Shards { // TODO: replace for loop with a more efficient algo that takes column boundaries to only pick the few of possibly thousands of shards that are within the min-max bounds
		// parallel scan over shards
		go func(s *storageShard) {
			defer func () {
				if r := recover(); r != nil {
					// fmt.Println("panic during scan:", r, string(debug.Stack()))
					q_ <- &shardqueue{s, nil, scanError{r, string(debug.Stack())}, nil, nil, nil}
				}
			}()
			q_ <- s.scan_order(boundaries, conditionCols, condition, sortcols, sortdirs, total_limit, callbackCols)
		}(s)
		rest = rest + 1
	}
	// collect all subchans
	for i := 0; i < rest; i++ {
		qe := <- q_
		if qe.err.r != nil {
			panic(qe.err) // propagate errors that occur inside inner scan
		}
		if len(qe.items) > 0 {
			heap.Push(&q, qe) // add to heap
		}
	}

	// collect values from parallel scan
	akkumulator := neutral
	// TODO: do queue polling instead of this naive testing code
	for len(q.q) > 0 {
		qx := q.q[0] // draw a random queue element

		if len(qx.items) == 0 {
			heap.Pop(&q) // remove empty element from stack
			continue
		}

		idx := qx.items[0] // draw the smallest element from shard
		qx.items = qx.items[1:]

		if offset > 0 {
			// skip offset
			offset--
		} else {
			if limit == 0 {
				return akkumulator
			}
			if limit > 0 {
				limit--
			}

			// prepare args for map function (map is guaranteed to run in order)
			mapargs := make([]scm.Scmer, len(callbackCols))
			for i, reader := range qx.mcols {
				mapargs[i] = reader(idx) // read column value into map argument
			}
			// call map function
			value := callbackFn(mapargs...)

			// aggregate
			akkumulator = aggregateFn(akkumulator, value)
		}

		if len(qx.items) > 0 {
			heap.Fix(&q, 0) // sink up since we have the next value
		} else {
			// sub-queue is empty -> remove
			heap.Pop(&q)
		}
	}
	return akkumulator
}

func (t *storageShard) scan_order(boundaries boundaries, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []bool, limit int, callbackCols []string) (result *shardqueue) {
	result = new(shardqueue)
	result.shard = t
	// TODO: mergesort sink instead of list-append-sort would allow early-out

	conditionFn := scm.OptimizeProcToSerialFunction(condition)

	// prepare filter function
	cdataset := make([]scm.Scmer, len(conditionCols))

	// prepare sort criteria so they can be queried easily
	result.scols = make([]func(uint) scm.Scmer, len(sortcols))
	for i, scol := range sortcols {
		if colname, ok := scol.(string); ok {
			// naive column sort
			result.scols[i] = t.ColumnReader(colname)
		} else if proc, ok := scol.(scm.Proc); ok {
			// complex lambda columns
			largs := make([]func(uint) scm.Scmer, len(proc.Params.([]scm.Scmer))) // allocate only once, reuse in loop
			for j, param := range proc.Params.([]scm.Scmer) {
				largs[j] = t.ColumnReader(string(param.(scm.Symbol)))
			}
			procFn := scm.OptimizeProcToSerialFunction(proc)
			result.scols[i] = func(idx uint) scm.Scmer {
				largs_ := make([]scm.Scmer, len(largs))
				for i, getter := range largs {
					// fetch columns used for getter
					largs_[i] = getter(idx)
				}
				// execute getter
				return procFn(largs_...)
			}
		} else {
			panic("unknown sort criteria: " + fmt.Sprint(scol))
		}
	}

	// prepare map columns (but only caller will use them)
	result.mcols = make([]func(uint) scm.Scmer, len(callbackCols))
	for i, arg := range callbackCols {
		if arg == "$update" {
			result.mcols[i] = func(idx uint) scm.Scmer {
				return t.UpdateFunction(idx, true)
			}
		} else {
			result.mcols[i] = t.ColumnReader(arg)
		}
	}

	// main storage
	ccols := make([]ColumnStorage, len(conditionCols))
	for i, k := range conditionCols { // iterate over columns
		var ok bool
		ccols[i], ok = t.columns[k] // find storage
		if !ok {
			panic("Column does not exist: `" + t.t.schema.Name + "`.`" + t.t.Name + "`.`" + k + "`")
		}
	}

	// scan loop in read lock
	var maxInsertIndex int
	func () {
		t.mu.RLock() // lock whole shard for reading since we frequently read deletions
		defer t.mu.RUnlock() // finished reading
		// remember current insert status (so don't scan things that are inserted during map)
		maxInsertIndex = len(t.inserts)

		// iterate over items (indexed)
		for idx := range t.iterateIndex(boundaries, maxInsertIndex) { // TODO: iterateIndexSorted
			if t.deletions.Get(idx) {
				continue // item is on delete list
			}

			if idx < t.main_count {
				// value from main storage
				// check condition
				for i, k := range ccols { // iterate over columns
					cdataset[i] = k.GetValue(idx)
				}
			} else {
				// value from delta storage
				// prepare&call condition function
				for i, k := range conditionCols { // iterate over columns
					cdataset[i] = t.getDelta(int(idx - t.main_count), k) // fill value
				}
			}
			// check condition
			if (!scm.ToBool(conditionFn(cdataset...))) {
				continue // condition did not match
			}

			result.items = append(result.items, idx)
		}
	}()

	// and now sort result!
	result.sortdirs = sortdirs
	// TODO: find conditions when exactly we don't need to sort anymore (fully covered indexes, no inserts); the same condition could be used to exit early during iterateIndex
	if (maxInsertIndex > 0 || true) && len(sortcols) > 0 {
		sort.Sort(result)
	}
	return
}

