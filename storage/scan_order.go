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
import "time"
import "sort"
import "strings"
import "runtime/debug"
import "container/heap"
import "sync/atomic"
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"

type shardqueue struct {
	shard    *storageShard
	items    []uint // TODO: refactor to chan, so we can block generating too much entries
	err      scanError
	mcols    []func(uint) scm.Scmer // map column reader
	scols    []func(uint) scm.Scmer // sort criteria column reader
	sortdirs []func(...scm.Scmer) scm.Scmer
}

// scanOrderResult bundles per-shard outputs for ordered scans.
type scanOrderResult struct {
	res        *shardqueue
	err        scanError // err.r != nil indicates an error
	inputCount int64
	scanCount  int64
}

// sort interface for shardqueue (local) (TODO: heap could be more efficient because early-out will be cheaper)
func (s *shardqueue) Len() int {
	return len(s.items)
}
func (s *shardqueue) Less(i, j int) bool {
	for c := 0; c < len(s.scols); c++ {
		a := s.scols[c](s.items[i])
		b := s.scols[c](s.items[j])
		if scm.ToBool(s.sortdirs[c](a, b)) {
			return true
		} else if scm.ToBool(s.sortdirs[c](b, a)) {
			return false
		} // else: go to next level
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
		a := s.q[i].scols[c](s.q[i].items[0])
		b := s.q[j].scols[c](s.q[j].items[0])
		if scm.ToBool(s.q[i].sortdirs[c](a, b)) {
			return true
		} else if scm.ToBool(s.q[i].sortdirs[c](b, a)) {
			return false
		} // else: go to next level
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
	s.q = s.q[0 : len(s.q)-1]
	return result
}

// TODO: helper function for priority-q. golangs implementation is kinda quirky, so do our own. container/heap especially lacks the function to test the value at front instead of popping it

// map reduce implementation based on scheme scripts
func (t *table) scan_order(conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool) scm.Scmer {
	// Measure analysis time
	analyzeStart := time.Now()
	/* TODO(memcp): Range-based braking & vectorization
	   - Maintain a top-k threshold (k = offset+limit) on the global queue and stop scanning when no shard's next-best key can beat threshold.
	   - Vectorize predicate/key evaluation with selection vectors to reduce branching and allocations (batch evaluate condition, compact indices, then project/aggregate).
	   - Pre-bind comparators (ASC/DESC) per sort key to avoid dynamic checks; reuse argument slices in hot loops.
	*/
	/* analyze condition query */
	boundaries := extractBoundaries(conditionCols, condition)
	lower, upperLast := indexFromBoundaries(boundaries)
	// TODO: append sortcols to boundaries

	// TODO: sortcols that are not just simple columns but complex lambda expressions could be temporarily materialized to trade memory for execution time
	// --> sortcols can then be rewritten to strings

	// give sharding hints
	for _, b := range boundaries {
		t.AddPartitioningScore([]string{b.col})
	}
	analyzeNs := time.Since(analyzeStart).Nanoseconds()

	callbackFn := scm.OptimizeProcToSerialFunction(callback)
	// wrap map callback to count outputs
	var outCount int64
	innerCallback := callbackFn
	callbackFn = func(args ...scm.Scmer) scm.Scmer {
		atomic.AddInt64(&outCount, 1)
		return innerCallback(args...)
	}
	aggregateFn := func(...scm.Scmer) scm.Scmer { return nil }
	if aggregate != nil {
		aggregateFn = scm.OptimizeProcToSerialFunction(aggregate)
	}

	// total_limit helps the shard-scanners to early-out
	total_limit := -1
	if limit >= 0 {
		total_limit = offset + limit
	}

	// TODO(memcp): Parallel braking
	// - Introduce a shared (atomic) global threshold for the k-th element (k = total_limit) in multi-key space.
	// - Option 1 (preferred): implement ordered per-shard iteration (iterateIndexSorted by sortcols). Each shard streams next-best tuple; if next-best >= threshold, early-stop shard.
	// - Option 2 (interim): keep a per-shard local top-k heap while scanning unsorted; prune using global threshold; sort local top-k only.

	// TODO(memcp): Selection vectors & vectorization
	// - Batch predicate evaluation into a selection vector; compact indices; then project/aggregate only selected rows.
	// - Pre-bind ASC/DESC comparators; reuse argument slices to avoid allocations.

	// Measure execution time of the ordered scan
	execStart := time.Now()
	var q globalqueue
	q_ := make(chan scanOrderResult, 1)
	var inputCount int64
	gls.Go(func() {
		t.iterateShards(boundaries, func(s *storageShard) {
			// parallel scan over shards
			defer func() {
				if r := recover(); r != nil {
					// fmt.Println("panic during scan:", r, string(debug.Stack()))
					q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
				}
			}()
			res := s.scan_order(boundaries, lower, upperLast, conditionCols, condition, sortcols, sortdirs, total_limit, callbackCols)
			q_ <- scanOrderResult{res: res, inputCount: int64(s.Count()), scanCount: int64(len(res.items))}
		})
		close(q_)
	})
	// collect all subchans
	for msg := range q_ {
		if msg.err.r != nil {
			panic(msg.err) // propagate errors that occur inside inner scan
		}
		if msg.res != nil && len(msg.res.items) > 0 {
			heap.Push(&q, msg.res) // add to heap
		}
		inputCount += msg.inputCount
	}

	// collect values from parallel scan
	akkumulator := neutral
	// TODO: do queue polling instead of this naive testing code
	hadValue := false
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
			hadValue = true
		}

		if len(qx.items) > 0 {
			heap.Fix(&q, 0) // sink up since we have the next value
		} else {
			// sub-queue is empty -> remove
			heap.Pop(&q)
		}
	}
	if !hadValue && isOuter {
		akkumulator = aggregateFn(akkumulator, callbackFn(make([]scm.Scmer, len(callbackCols)))) // outer join: call once with NULLs
	}
	execNs := time.Since(execStart).Nanoseconds()
	// log statistics for ordered scan (best-effort, async) if threshold met
	if inputCount > int64(Settings.AnalyzeMinItems) {
		go func(anNs, exNs int64) {
			defer func() { _ = recover() }()
			filterEnc := ""
			if p, ok := condition.(scm.Proc); ok {
				filterEnc = encodeScmerToString(p.Body, conditionCols, p.Params.([]scm.Scmer))
			}
			var sb strings.Builder
			for i, sc := range sortcols {
				if i > 0 {
					sb.WriteByte('|')
				}
				switch v := sc.(type) {
				case string:
					// Plain column name
					sb.WriteString(v)
				default:
					// Use helper to encode unknowns/lambdas as '?'
					encodeScmer(sc, &sb, nil, nil)
				}
			}
			orderEnc := sb.String()
			safeLogScan(t.schema.Name, t.Name, true, filterEnc, orderEnc, inputCount, outCount, anNs, exNs)
		}(analyzeNs, execNs)
	}
	return akkumulator
}

func (t *storageShard) scan_order(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limit int, callbackCols []string) (result *shardqueue) {
	result = new(shardqueue)
	result.shard = t

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
			// complex lambda columns (TODO: either remove lambda columns or add colname mapping)
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
		ccols[i] = t.getColumnStorageOrPanic(k)
	}

	// initialize main_count lazily if needed
	t.ensureMainCount()
	// scan loop in read lock
	var maxInsertIndex int
	func() {
		t.mu.RLock()         // lock whole shard for reading since we frequently read deletions
		defer t.mu.RUnlock() // finished reading
		// remember current insert status (so don't scan things that are inserted during map)
		maxInsertIndex = len(t.inserts)

		// iterate over items (indexed)
		// TODO(memcp): iterateIndexSorted(boundaries, sortcols) to emit tuples in ORDER BY sequence.
		t.iterateIndex(boundaries, lower, upperLast, maxInsertIndex, func(idx uint) {
			if t.deletions.Get(idx) {
				return // item is on delete list
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
					cdataset[i] = t.getDelta(int(idx-t.main_count), k) // fill value
				}
			}
			// check condition
			if !scm.ToBool(conditionFn(cdataset...)) {
				return // condition did not match
			}

			result.items = append(result.items, idx)
		})
	}()

	// and now sort result!
	result.sortdirs = sortdirs
	// TODO: find conditions when exactly we don't need to sort anymore (fully covered indexes, no inserts); the same condition could be used to exit early during iterateIndex
	if (maxInsertIndex > 0 || true) && len(sortcols) > 0 {
		sort.Sort(result)
		// or: quicksort but those segments above offset+limit can be omitted
	}
	return
}
