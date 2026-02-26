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
import "time"
import "strings"
import "runtime/debug"
import "container/heap"
import "github.com/jtolds/gls"
import "github.com/launix-de/memcp/scm"

// optimizeScanOrder is the Optimize hook for the scan_order declaration.
// scan_order args: 1=schema, 2=table, 3=filterCols, 4=filter, 5=sortcols, 6=sortdirs,
// 7=offset, 8=limit, 9=mapCols, 10=map, 11=reduce, 12=neutral, 13=isOuter
func optimizeScanOrder(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	// Optimize args 1-10 normally
	for i := 1; i <= 10 && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	// Arg 11 (reduce callback): set callback ownership before optimizing
	if len(v) > 11 && !v[11].IsNil() {
		oc.SetCallbackOwned([]bool{true, false}) // acc is owned
		v[11], _ = oc.OptimizeSub(v[11], true)
	}
	// Arg 12 (neutral)
	if len(v) > 12 {
		v[12], _ = oc.OptimizeSub(v[12], true)
	}
	// Arg 13 (isOuter)
	if len(v) > 13 {
		v[13], _ = oc.OptimizeSub(v[13], true)
	}
	return scm.NewSlice(v), nil
}

type shardqueue struct {
	shard    *storageShard
	items    []uint32 // TODO: refactor to chan, so we can block generating too much entries
	err      scanError
	scols    []func(uint32) scm.Scmer // sort criteria column reader
	sortdirs []func(...scm.Scmer) scm.Scmer
	mapper   *ShardMapReducer
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
	// touch temp columns so CacheManager knows they're still in use
	touchTempColumns(t, conditionCols, callbackCols)
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

	var outCount int64

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
	hadValue := false
	// initialize MapReducers: pre-allocate args per shard
	for _, sq := range q.q {
		sq.mapper = sq.shard.OpenMapReducer(callbackCols, callback, aggregate)
	}

	var buf [1024]uint32 // stack-allocated batch buffer (4 KB, fits in L1)
	bufN := 0
	var bufShard *shardqueue
	for len(q.q) > 0 {
		qx := q.q[0]

		if len(qx.items) == 0 {
			heap.Pop(&q)
			continue
		}

		if offset > 0 {
			// offset skip (typically small)
			qx.items = qx.items[1:]
			offset--
			if len(qx.items) > 0 {
				heap.Fix(&q, 0)
			} else {
				heap.Pop(&q)
			}
			continue
		}

		if limit == 0 {
			break
		}

		// Pop one item from the global merge
		item := qx.items[0]
		qx.items = qx.items[1:]

		// If shard changed, flush the buffer to the previous shard's mapper
		if bufShard != nil && bufShard != qx {
			akkumulator = bufShard.mapper.Stream(akkumulator, buf[:bufN])
			hadValue = true
			bufN = 0
		}

		// Accumulate item into buffer
		bufShard = qx
		buf[bufN] = item
		bufN++
		outCount++
		if limit >= 0 {
			limit--
		}

		// Flush if buffer full
		if bufN == len(buf) {
			akkumulator = bufShard.mapper.Stream(akkumulator, buf[:bufN])
			hadValue = true
			bufN = 0
		}

		// Re-heapify or remove exhausted shard
		if len(qx.items) > 0 {
			heap.Fix(&q, 0)
		} else {
			heap.Pop(&q)
		}
	}
	// Flush remaining buffer
	if bufN > 0 && bufShard != nil {
		akkumulator = bufShard.mapper.Stream(akkumulator, buf[:bufN])
		hadValue = true
	}
	if !hadValue && isOuter {
		callbackFn := scm.OptimizeProcToSerialFunction(callback)
		aggregateFn := scm.OptimizeProcToSerialFunction(aggregate)
		nullRow := make([]scm.Scmer, len(callbackCols))
		for i := range nullRow {
			nullRow[i] = scm.NewNil()
		}
		akkumulator = aggregateFn(akkumulator, callbackFn(nullRow...)) // outer join: call once with NULLs
	}
	execNs := time.Since(execStart).Nanoseconds()
	// log statistics for ordered scan (best-effort, async) if threshold met
	if inputCount > int64(Settings.AnalyzeMinItems) {
		go func(anNs, exNs int64) {
			defer func() { _ = recover() }()
			filterEnc := ""
			if proc, ok := condition.Any().(scm.Proc); ok {
				var params []scm.Scmer
				if proc.Params.IsSlice() {
					params = proc.Params.Slice()
				} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
					params = arr
				}
				filterEnc = encodeScmerToString(proc.Body, conditionCols, params)
			}
			var sb strings.Builder
			for i, sc := range sortcols {
				if i > 0 {
					sb.WriteByte('|')
				}
				if sc.IsString() {
					sb.WriteString(sc.String())
				} else {
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
	for i := range cdataset {
		cdataset[i] = scm.NewNil()
	}

	// prepare sort criteria so they can be queried easily
	result.scols = make([]func(uint32) scm.Scmer, len(sortcols))
	for i, scol := range sortcols {
		if scol.IsString() {
			colname := scol.String()
			result.scols[i] = t.ColumnReader(colname)
			continue
		}
		if proc, ok := scol.Any().(scm.Proc); ok {
			var params []scm.Scmer
			if proc.Params.IsSlice() {
				params = proc.Params.Slice()
			} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
				params = arr
			}
			largs := make([]func(uint32) scm.Scmer, len(params))
			for j, param := range params {
				name := ""
				if param.IsSymbol() {
					name = param.String()
				} else if sym, ok := param.Any().(scm.Symbol); ok {
					name = string(sym)
				} else {
					name = scm.String(param)
				}
				largs[j] = t.ColumnReader(name)
			}
			procFn := scm.OptimizeProcToSerialFunction(scol)
			result.scols[i] = func(idx uint32) scm.Scmer {
				vals := make([]scm.Scmer, len(largs))
				for j, getter := range largs {
					vals[j] = getter(idx)
				}
				return procFn(vals...)
			}
			continue
		}
		panic("unknown sort criteria: " + scm.String(scol))
	}

	// If a sort column has a column-level collation and sortdir is the default < or >,
	// replace the comparator with the appropriate collator-based comparator to honor
	// column collation without explicit ORDER BY COLLATE.
	// Build an adjusted sortdirs slice for this scan.
	adjustedSortdirs := make([]func(...scm.Scmer) scm.Scmer, len(sortdirs))
	for i := range sortdirs {
		adjustedSortdirs[i] = sortdirs[i]
		colname := ""
		if sortcols[i].IsString() {
			colname = sortcols[i].String()
		} else if sym, ok := sortcols[i].Any().(scm.Symbol); ok {
			colname = string(sym)
		} else {
			continue
		}
		// find column definition
		coll := ""
		for _, c := range t.t.Columns {
			if c.Name == colname {
				coll = c.Collation
				break
			}
		}
		if coll == "" {
			continue
		}
		// Only actionable collations: those with a language suffix or explicit 'bin'.
		if !(strings.Contains(coll, "_") || strings.EqualFold(coll, "bin")) {
			continue
		}
		// If sortdirs[i] already is a collate closure, respect it (explicit ORDER BY COLLATE)
		if _, _, isCollate := scm.LookupCollate(sortdirs[i]); isCollate {
			continue
		}
		// Derive reverse flag by probing comparator semantics (robust across pointer differences)
		reverse := false // ASC by default
		defer func() { _ = recover() }()
		// If dir(1,2) is true, comparator behaves like '<' (ASC) -> reverse=false
		// Else if dir(2,1) is true, comparator behaves like '>' (DESC) -> reverse=true
		if res := sortdirs[i](scm.NewInt(1), scm.NewInt(2)); scm.ToBool(res) {
			reverse = false
		} else if res2 := sortdirs[i](scm.NewInt(2), scm.NewInt(1)); scm.ToBool(res2) {
			reverse = true
		}
		// Build comparator via (collate coll reverse?)
		cmpScm := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString(coll), scm.NewBool(reverse))
		cmpFn := scm.OptimizeProcToSerialFunction(cmpScm)
		adjustedSortdirs[i] = cmpFn
	}

	// main storage
	ccols := make([]ColumnStorage, len(conditionCols))
	for i, k := range conditionCols { // iterate over columns
		ccols[i] = t.getColumnStorageOrPanic(k)
	}

	// initialize main_count lazily if needed
	t.ensureMainCount(false)
	// scan loop in read lock
	var maxInsertIndex int
	func() {
		t.mu.RLock()         // lock whole shard for reading since we frequently read deletions
		defer t.mu.RUnlock() // finished reading
		// remember current insert status (so don't scan things that are inserted during map)
		maxInsertIndex = len(t.inserts)

		// iterate over items (indexed)
		// TODO(memcp): iterateIndexSorted(boundaries, sortcols) to emit tuples in ORDER BY sequence.
		currentTx := CurrentTx()
		var buf [1024]uint32
		resultCap := 1024
		result.items = make([]uint32, resultCap)
		resultN := 0
		t.iterateIndex(boundaries, lower, upperLast, maxInsertIndex, buf[:], func(batch []uint32) bool {
			// filter in-place: overwrite batch with passing IDs
			outN := 0
			for _, idx := range batch {
				if currentTx != nil && currentTx.Mode == TxACID {
					if !currentTx.IsVisible(t, idx) {
						continue
					}
				} else {
					if t.deletions.Get(idx) {
						continue // item is on delete list
					}
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
					continue // condition did not match
				}

				batch[outN] = idx
				outN++
			}
			// grow result if needed, then flush filtered batch
			for resultN+outN > resultCap {
				resultCap *= 2
				newItems := make([]uint32, resultCap)
				copy(newItems, result.items[:resultN])
				result.items = newItems
			}
			copy(result.items[resultN:], batch[:outN])
			resultN += outN
			return true
		})
		result.items = result.items[:resultN]
	}()

	// and now sort result!
	result.sortdirs = adjustedSortdirs
	// TODO: find conditions when exactly we don't need to sort anymore.
	// The sort can be skipped when ALL of these hold:
	// 1. The index used by iterateIndex covers the ORDER BY columns in
	//    the same order (the index's Cols prefix matches sortcols).
	// 2. The sort directions match (ASC for the index's natural order).
	// 3. There are no delta inserts (maxInsertIndex == 0), OR the delta
	//    items were merged in sorted order by the streaming merge in
	//    StorageIndex.iterate (which they are — but the condition filter
	//    in the callback above can discard items, so the output is still
	//    sorted, just with gaps).
	// 4. With Optimization 1 (Native sort): if the shard's physical row
	//    order matches ORDER BY and there are no deltas, the sort is free.
	// When these conditions are met, the same knowledge could also be
	// used to exit early during iterateIndex (stop after OFFSET+LIMIT).
	if (maxInsertIndex > 0 || true) && len(sortcols) > 0 {
		sort.Sort(result)
		// or: quicksort but those segments above offset+limit can be omitted
	}
	return
}
