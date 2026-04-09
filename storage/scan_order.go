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
import "github.com/carli2/hybridsort"
import "time"
import "strings"
import "runtime/debug"
import "container/heap"
import "github.com/launix-de/memcp/scm"

func optimizeScanOrderMulti(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	// scan_order_multi args: 0=fn, 1=tx, 2=schemas, 3=tables, 4=filterCols, 5=filterFns,
	// 6=sortcols, 7=sortdirs, 8=partCols, 9=offset, 10=limit, 11=mapCols, 12=mapFns,
	// 13=reduce, 14=neutral, 15=isOuter
	for i := 1; i <= 12 && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	if len(v) > 13 && !v[13].IsNil() {
		oc.SetCallbackOwned([]bool{true, false})
		v[13], _ = oc.OptimizeSub(v[13], true)
	}
	if len(v) > 14 {
		v[14], _ = oc.OptimizeSub(v[14], true)
	}
	if len(v) > 15 {
		v[15], _ = oc.OptimizeSub(v[15], true)
	}
	return scm.NewSlice(v), nil
}

func optimizeScanOrder(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	// NOTE: scan_order has no reduce2, so batch-rewrite cannot flush the last
	// partial batch. Disabled until scan_order gains reduce2 or an alternative
	// flush mechanism is implemented.
	// if rewritten := tryScanOrderBatchRewrite(v); !rewritten.IsNil() {
	// 	return oc.OptimizeSub(rewritten, useResult)
	// }
	mapEnd, reduceIdx, neutralIdx, outerIdx := 12, 13, 14, 15
	for i := 1; i <= mapEnd && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	if len(v) > reduceIdx && !v[reduceIdx].IsNil() {
		oc.SetCallbackOwned([]bool{true, false})
		v[reduceIdx], _ = oc.OptimizeSub(v[reduceIdx], true)
	}
	if len(v) > neutralIdx {
		v[neutralIdx], _ = oc.OptimizeSub(v[neutralIdx], true)
	}
	if len(v) > outerIdx {
		v[outerIdx], _ = oc.OptimizeSub(v[outerIdx], true)
	}
	return scm.NewSlice(v), nil
}

// pkEqual compares two partition key slices element-wise.
func pkEqual(a, b []scm.Scmer) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !scm.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

// skipPartition uses binary search to skip all remaining items in the current
// partition of a shardqueue. Since items are sorted by (partition_key, order_key),
// all items of the same partition are contiguous — sort.Search finds the first
// item of the next partition in O(log n).
func skipPartition(q *globalqueue, qx *shardqueue, pk []scm.Scmer, n int) {
	idx := sort.Search(len(qx.items), func(i int) bool {
		for c := 0; c < n; c++ {
			if !scm.Equal(qx.scols[c](qx.items[i]), pk[c]) {
				return true
			}
		}
		return false
	})
	qx.items = qx.items[idx:]
	if len(qx.items) > 0 {
		heap.Fix(q, 0)
	} else {
		heap.Pop(q)
	}
}

type shardqueue struct {
	shard        *storageShard
	items        []uint32 // TODO: refactor to chan, so we can block generating too much entries
	err          scanError
	scols        []func(uint32) scm.Scmer // sort criteria column reader
	sortdirs     []func(...scm.Scmer) scm.Scmer
	mapper       *ShardMapReducer
	callbackCols []string  // per-table map columns (for multi-table merge)
	callback     scm.Scmer // per-table map function (for multi-table merge)
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
	if i >= len(s.items) || j >= len(s.items) {
		return i < j
	}
	cmpCount := len(s.scols)
	if len(s.sortdirs) < cmpCount {
		cmpCount = len(s.sortdirs)
	}
	for c := 0; c < cmpCount; c++ {
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

type topKHeap struct {
	items []uint32
	less  func(a, b uint32) bool
}

func (h *topKHeap) Len() int {
	return len(h.items)
}

func (h *topKHeap) Less(i, j int) bool {
	// Reverse the user-facing ordering so heap[0] stays the current worst item.
	return h.less(h.items[j], h.items[i])
}

func (h *topKHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

func (h *topKHeap) Push(x any) {
	h.items = append(h.items, x.(uint32))
}

func (h *topKHeap) Pop() any {
	n := len(h.items)
	item := h.items[n-1]
	h.items = h.items[:n-1]
	return item
}

// sort interface for global shard-queue
func (s *globalqueue) Len() int {
	return len(s.q)
}
func (s *globalqueue) Less(i, j int) bool {
	if i >= len(s.q) || j >= len(s.q) {
		return i < j
	}
	if len(s.q[i].items) == 0 {
		return false
	}
	if len(s.q[j].items) == 0 {
		return true
	}
	cmpCount := len(s.q[i].scols)
	if len(s.q[j].scols) < cmpCount {
		cmpCount = len(s.q[j].scols)
	}
	if len(s.q[i].sortdirs) < cmpCount {
		cmpCount = len(s.q[i].sortdirs)
	}
	if len(s.q[j].sortdirs) < cmpCount {
		cmpCount = len(s.q[j].sortdirs)
	}
	for c := 0; c < cmpCount; c++ {
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

func topKByOrder(items []uint32, keep int, less func(a, b uint32) bool) []uint32 {
	if keep <= 0 || len(items) == 0 {
		return nil
	}
	if keep >= len(items) {
		out := append([]uint32(nil), items...)
		hybridsort.Slice(out, func(i, j int) bool {
			return less(out[i], out[j])
		})
		return out
	}
	h := &topKHeap{less: less}
	for _, item := range items {
		if h.Len() < keep {
			heap.Push(h, item)
			continue
		}
		if less(item, h.items[0]) {
			h.items[0] = item
			heap.Fix(h, 0)
		}
	}
	out := append([]uint32(nil), h.items...)
	hybridsort.Slice(out, func(i, j int) bool {
		return less(out[i], out[j])
	})
	return out
}

// TODO: helper function for priority-q. golangs implementation is kinda quirky, so do our own. container/heap especially lacks the function to test the value at front instead of popping it

// scanOrderTableSpec holds per-table parameters for scanOrderMulti.
type scanOrderTableSpec struct {
	table         *table
	conditionCols []string
	condition     scm.Scmer
	sortcols      []scm.Scmer
	callbackCols  []string
	callback      scm.Scmer
}

// extendBoundariesWithSortCols appends sort columns to the boundaries when all
// existing filter boundaries are point lookups and the comparators are
// index-order compatible (ASC). This lets the shard return rows already sorted
// by ORDER BY, reducing the cross-shard merge to merging pre-sorted runs.
func extendBoundariesWithSortCols(b boundaries, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer) boundaries {
	if len(b) == 0 {
		return b
	}
	allEq := true
	for _, bi := range b {
		if !boundaryIsPoint(bi) {
			allEq = false
			break
		}
	}
	canAppendSortPrefix := len(sortcols) > 0
	for i := range sortcols {
		if i >= len(sortdirs) || sortdirs[i] == nil {
			continue // default ASC
		}
		asc := false
		probeOK := true
		func() {
			defer func() {
				if r := recover(); r != nil {
					probeOK = false
				}
			}()
			if scm.ToBool(sortdirs[i](scm.NewInt(1), scm.NewInt(2))) &&
				!scm.ToBool(sortdirs[i](scm.NewInt(2), scm.NewInt(1))) {
				asc = true
			}
		}()
		if !probeOK || !asc {
			canAppendSortPrefix = false
			break
		}
	}
	if !allEq || !canAppendSortPrefix {
		return b
	}
	for _, scol := range sortcols {
		if scol.IsString() {
			col := scol.String()
			already := false
			for _, bi := range b {
				if bi.col == col {
					already = true
					break
				}
			}
			if !already {
				b = append(b, columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), upper: scm.NewNil()})
			}
			continue
		}
		proc, ok := scol.Any().(scm.Proc)
		if !ok && scol.IsProc() {
			proc = *scol.Proc()
			ok = true
		}
		if !ok {
			continue
		}
		var procParams []scm.Scmer
		if proc.Params.IsSlice() {
			procParams = proc.Params.Slice()
		}
		if len(procParams) == 0 {
			continue
		}
		sortCondCols := make([]string, len(procParams))
		for j, param := range procParams {
			if param.IsSymbol() {
				sortCondCols[j] = param.String()
			} else {
				sortCondCols[j] = scm.String(param)
			}
		}
		if !isRawDataset(procParams, proc.Body) {
			continue
		}
		canon := canonicalColName(proc.Body, procParams, sortCondCols)
		mc, mf := buildComputedFn(proc.Body, proc.Params, proc.En, sortCondCols)
		if mf.IsNil() || mc == nil {
			continue
		}
		already := false
		for _, bi := range b {
			if bi.col == canon {
				already = true
				break
			}
		}
		if !already {
			b = append(b, columnboundaries{col: canon, matcher: RangeMatcher, lower: scm.NewNil(), upper: scm.NewNil(), mapCols: mc, mapFn: mf})
		}
	}
	return b
}

// scanOrderMulti performs an ordered scan across one or more tables, merging
// results from all tables' shards into a single globally sorted stream.
// Each table has its own filter, sort columns and map function, but sort
// directions, offset/limit, reduce and neutral are shared.
func scanOrderMulti(currentTx *TxContext, tables []scanOrderTableSpec, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool) scm.Scmer {
	ss := SessionStateFromTx(currentTx)
	if ss != nil && ss.IsKilled() {
		panic("query killed")
	}

	total_limit := -1
	if limitPartitionCols == 0 && limit >= 0 {
		total_limit = offset + limit
	}

	var q globalqueue
	q_ := make(chan scanOrderResult, len(tables)*4)
	var inputCount int64
	var wg sync.WaitGroup

	// Launch shard-parallel scans for each table
	for ti := range tables {
		spec := &tables[ti]
		t := spec.table
		touchTempColumns(t, spec.conditionCols, spec.callbackCols)

		// Boundary analysis per table
		analyzeStart := time.Now()
		bounds := extractBoundaries(spec.conditionCols, spec.condition)
		bounds = dropUnsafeComputeBoundaries(t, bounds)
		reorderByFrequency(bounds, t)
		bounds = extendBoundariesWithSortCols(bounds, spec.sortcols, sortdirs)
		lower, upperLast := indexFromBoundaries(bounds)

		if Settings.ScanDebugging {
			dbg := fmt.Sprintf("[SCAN_ORDER_MULTI] %s.%s", t.schema.Name, t.Name)
			for _, b := range bounds {
				dbg += fmt.Sprintf(" %s:[%v..%v]", b.col, b.lower, b.upper)
			}
			dbg += fmt.Sprintf(" lower=%v upper=%v", lower, upperLast)
			fmt.Println(dbg)
		}

		for _, b := range bounds {
			t.AddPartitioningScore([]string{b.col})
		}
		analyzeNs := time.Since(analyzeStart).Nanoseconds()

		// Capture closure variables
		callbackCols := spec.callbackCols
		callback := spec.callback
		conditionCols := spec.conditionCols
		condition := spec.condition
		sortcols := spec.sortcols
		tableBounds := bounds

		done := t.iterateShardsParallel(tableBounds, func(s *storageShard, solo bool) {
			if ss != nil && ss.IsKilled() {
				panic("query killed")
			}
			defer func() {
				if r := recover(); r != nil {
					q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
				}
			}()
			res := s.scan_order(tableBounds, lower, upperLast, conditionCols, condition, sortcols, sortdirs, limitPartitionCols, offset, total_limit, callbackCols, currentTx, ss)
			res.callbackCols = callbackCols
			res.callback = callback
			q_ <- scanOrderResult{res: res, inputCount: int64(s.Count()), scanCount: int64(len(res.items))}
		})
		if done != nil {
			wg.Add(1)
			go func(ch <-chan struct{}) {
				<-ch
				wg.Done()
			}(done)
		}

		// Per-table logging (best-effort, async) — inputCount is 0 here (set
		// after merge), so this fires only when ScanDebugging is enabled.
		if Settings.ScanDebugging {
			go func(tbl *table, cCols []string, cond scm.Scmer, scols []scm.Scmer, bnds boundaries, anNs int64) {
				defer func() { _ = recover() }()
				filterEnc := ""
				if proc, ok := cond.Any().(scm.Proc); ok {
					var params []scm.Scmer
					if proc.Params.IsSlice() {
						params = proc.Params.Slice()
					} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
						params = arr
					}
					filterEnc = encodeScmerToString(proc.Body, cCols, params)
				}
				var sb strings.Builder
				for i, sc := range scols {
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
				indexColsEnc := boundaryIndexCols(bnds)
				safeLogScan(tbl.schema.Name, tbl.Name, true, filterEnc, orderEnc, indexColsEnc, 0, 0, anNs, 0)
			}(t, conditionCols, condition, sortcols, tableBounds, analyzeNs)
		}
	}

	// Close result channel when all tables' shard scans complete
	go func() {
		wg.Wait()
		close(q_)
	}()

	// Collect shard results into globalqueue
	var scanErr scanError
	for msg := range q_ {
		if msg.err.r != nil {
			if scanErr.r == nil {
				scanErr = msg.err
			}
			continue
		}
		if scanErr.r != nil {
			continue
		}
		if msg.res != nil && len(msg.res.items) > 0 {
			heap.Push(&q, msg.res)
		}
		inputCount += msg.inputCount
	}
	if scanErr.r != nil {
		panic(scanErr)
	}

	// Merge-collect phase: merge sorted shardqueues from all tables
	akkumulator := neutral
	hadValue := false
	// initialize MapReducers per shard (each shard uses its table's callbackCols/callback)
	for _, sq := range q.q {
		sq.mapper = sq.shard.OpenMapReducer(sq.callbackCols, sq.callback, aggregate, false, 0, nil, currentTx)
	}

	var buf [1024]uint32 // stack-allocated batch buffer (4 KB, fits in L1)
	bufN := 0
	var bufShard *shardqueue
	breakCaught := false

	// Per-partition offset/limit state. When limitPartitionCols == 0 this
	// degenerates to a single partition covering all rows (= global limit).
	var prevPK []scm.Scmer
	partOffset := offset
	partLimit := limit

	for !breakCaught && len(q.q) > 0 {
		qx := q.q[0]

		if len(qx.items) == 0 {
			heap.Pop(&q)
			continue
		}

		// Extract partition key from leading sort columns (empty slice when limitPartitionCols == 0)
		peekItem := qx.items[0]
		curPK := make([]scm.Scmer, limitPartitionCols)
		for c := 0; c < limitPartitionCols && c < len(qx.scols); c++ {
			curPK[c] = qx.scols[c](peekItem)
		}
		// Detect partition change (first row or key differs)
		if prevPK == nil || !pkEqual(prevPK, curPK) {
			// Flush buffer before partition switch
			if bufN > 0 && bufShard != nil {
				akkumulator, breakCaught = streamOrBreak(bufShard.mapper, akkumulator, buf[:bufN])
				hadValue = true
				bufN = 0
				if breakCaught {
					break
				}
			}
			partOffset = offset
			partLimit = limit
			prevPK = curPK
		}
		// Per-partition offset skip
		if partOffset > 0 {
			partOffset--
			qx.items = qx.items[1:]
			if len(qx.items) > 0 {
				heap.Fix(&q, 0)
			} else {
				heap.Pop(&q)
			}
			continue
		}
		// Per-partition limit exhausted
		if partLimit == 0 {
			if limitPartitionCols > 0 {
				// Bulk-skip rest of partition via binary search (O(log n))
				skipPartition(&q, qx, prevPK, limitPartitionCols)
				continue // proceed to next partition
			}
			// limitPartitionCols == 0: single partition = all done
			break
		}
		partLimit--

		// Pop one item from the global merge
		item := qx.items[0]
		qx.items = qx.items[1:]

		// If shard changed, flush the buffer to the previous shard's mapper
		if bufShard != nil && bufShard != qx {
			akkumulator, breakCaught = streamOrBreak(bufShard.mapper, akkumulator, buf[:bufN])
			hadValue = true
			bufN = 0
			if breakCaught {
				break
			}
		}

		// Accumulate item into buffer
		bufShard = qx
		buf[bufN] = item
		bufN++

		// Flush if buffer full
		if bufN == len(buf) {
			akkumulator, breakCaught = streamOrBreak(bufShard.mapper, akkumulator, buf[:bufN])
			hadValue = true
			bufN = 0
			if breakCaught {
				break
			}
		}

		// Re-heapify or remove exhausted shard
		if len(qx.items) > 0 {
			heap.Fix(&q, 0)
		} else {
			heap.Pop(&q)
		}
	}
	// Flush remaining buffer
	if !breakCaught && bufN > 0 && bufShard != nil {
		akkumulator, _ = streamOrBreak(bufShard.mapper, akkumulator, buf[:bufN])
		hadValue = true
	}
	if !hadValue && isOuter && len(tables) > 0 {
		cbCols := tables[0].callbackCols
		cb := tables[0].callback
		if aggregate.IsNil() {
			scm.Apply(cb, buildOuterNullCallbackRow(cbCols)...)
		} else {
			var nullMapper *ShardMapReducer
			if len(q.q) > 0 {
				nullMapper = q.q[0].mapper
			} else if len(tables[0].table.Shards) > 0 {
				nullMapper = tables[0].table.Shards[0].OpenMapReducer(cbCols, cb, aggregate, false, 0, nil, currentTx)
				defer nullMapper.Close()
			}
			if nullMapper != nil {
				akkumulator = nullMapper.reduceFn(akkumulator, outerNullMappedResult(nullMapper, neutral, aggregate, cbCols))
			}
		}
	}
	return akkumulator
}

// scan_order delegates to scanOrderMulti with a single-element table spec.
func (t *table) scan_order(currentTx *TxContext, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool) scm.Scmer {
	return scanOrderMulti(currentTx, []scanOrderTableSpec{{
		table:         t,
		conditionCols: conditionCols,
		condition:     condition,
		sortcols:      sortcols,
		callbackCols:  callbackCols,
		callback:      callback,
	}}, sortdirs, limitPartitionCols, offset, limit, aggregate, neutral, isOuter)
}

// streamOrBreak calls mapper.Stream and catches a breakSentinel panic (from $break
// pseudo-columns). When a break is caught, the current accumulator is returned and
// broke=true signals the merge loop to stop iteration.
func streamOrBreak(mapper *ShardMapReducer, acc scm.Scmer, recids []uint32) (result scm.Scmer, broke bool) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(breakSentinel); ok {
				broke = true
				result = acc
			} else {
				panic(r) // re-panic for all other errors
			}
		}
	}()
	result = mapper.Stream(acc, recids, nil)
	return
}

func (t *storageShard) scan_order(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, currentTx *TxContext, ss *scm.SessionState) (result *shardqueue) {
	result = new(shardqueue)
	result.shard = t
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}
	defaultSortDir := func(args ...scm.Scmer) scm.Scmer {
		if len(args) < 2 {
			return scm.NewBool(false)
		}
		return scm.NewBool(scm.Less(args[0], args[1]))
	}

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
			result.scols[i] = t.ColumnReaderTx(currentTx, colname)
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
				largs[j] = t.ColumnReaderTx(currentTx, name)
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
	adjustedSortdirs := make([]func(...scm.Scmer) scm.Scmer, len(sortcols))
	for i := range sortcols {
		dir := defaultSortDir
		if i < len(sortdirs) && sortdirs[i] != nil {
			dir = sortdirs[i]
		}
		adjustedSortdirs[i] = dir
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
		// Derive reverse flag by probing comparator semantics (robust across pointer differences).
		// Keep panic recovery strictly local to this probe: a function-wide defer-recover
		// here would swallow unrelated panics from scan/filter/map and surface as empty
		// result sets instead of proper SQL errors.
		reverse := false // ASC by default
		probeOK := true
		func() {
			defer func() {
				if r := recover(); r != nil {
					probeOK = false
				}
			}()
			// If dir(1,2) is true, comparator behaves like '<' (ASC) -> reverse=false
			// Else if dir(2,1) is true, comparator behaves like '>' (DESC) -> reverse=true
			if res := dir(scm.NewInt(1), scm.NewInt(2)); scm.ToBool(res) {
				reverse = false
			} else if res2 := dir(scm.NewInt(2), scm.NewInt(1)); scm.ToBool(res2) {
				reverse = true
			}
		}()
		if !probeOK {
			continue
		}
		// Build comparator via (collate coll reverse?)
		cmpScm := scm.Apply(scm.Globalenv.Vars[scm.Symbol("collate")], scm.NewString(coll), scm.NewBool(reverse))
		cmpFn := scm.OptimizeProcToSerialFunction(cmpScm)
		adjustedSortdirs[i] = cmpFn
	}

	skipShardReadLock := t.hasWriteOwner() || (currentTx != nil && currentTx.HasShardWrite(t))
	if t.t.tableLockOwner.Load() != nil {
		t.t.waitTableLock(ss, false)
	}

	// main storage — use skipShardReadLock to avoid redundant hasWriteOwner() per column
	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	cNeedsTxReader := make([]bool, len(conditionCols))
	for i, k := range conditionCols { // iterate over columns
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
		if proxy, ok := ccols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			cNeedsTxReader[i] = true
		}
	}
	// initialize main_count lazily if needed
	t.ensureMainCount(skipShardReadLock)
	// scan loop in read lock
	var maxInsertIndex int
	var visibleUpper uint32
	func() {
		shardLocked := false
		if !skipShardReadLock {
			t.mu.RLock()
			shardLocked = true
			// Table lock check must happen AFTER shard RLock — race-safe synchronization
			// point (mirrors storageShard.scan logic for the TOCTOU fix).
			if t.t.tableLockOwner.Load() != nil {
				t.mu.RUnlock()
				shardLocked = false
				t.t.waitTableLock(ss, false)
				t.mu.RLock()
				shardLocked = true
			}
		}
		defer func() {
			if shardLocked {
				t.mu.RUnlock()
			}
		}()
		// remember current insert status (so don't scan things that are inserted during map)
		maxInsertIndex = len(t.inserts)
		visibleUpper = t.main_count + uint32(maxInsertIndex)

		// iterate over items (indexed)
		// TODO(memcp): iterateIndexSorted(boundaries, sortcols) to emit tuples in ORDER BY sequence.
		var buf [1024]uint32
		resultCap := 1024
		result.items = make([]uint32, resultCap)
		resultN := 0
		t.iterateIndex(boundaries, lower, upperLast, maxInsertIndex, buf[:], func(batch []uint32) bool {
			// filter in-place: overwrite batch with passing IDs
			outN := 0
			for _, idx := range batch {
				if idx >= visibleUpper {
					continue
				}
				if currentTx != nil && currentTx.Mode == TxACID {
					if !currentTx.IsVisible(t, idx) {
						continue
					}
				} else if t.deletions.Get(uint(idx)) {
					continue // item is on delete list
				}

				if idx < t.main_count {
					// value from main storage
					// check condition
					for i, k := range ccols { // iterate over columns
						if cNeedsTxReader[i] {
							cdataset[i] = cReaders[i].GetValue(idx)
						} else {
							cdataset[i] = k.GetValue(idx)
						}
					}
				} else {
					// value from delta storage
					// prepare&call condition function
					for i, k := range conditionCols { // iterate over columns
						if cNeedsTxReader[i] {
							cdataset[i] = cReaders[i].GetValue(idx)
						} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
							cdataset[i] = ccols[i].GetValue(idx)
						} else {
							cdataset[i] = t.getDelta(int(idx-t.main_count), k) // fill value
						}
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
	itemPos := make(map[uint32]int, len(result.items))
	for i, idx := range result.items {
		itemPos[idx] = i
	}
	lessByID := func(a, b uint32) bool {
		cmpCount := len(result.scols)
		if len(result.sortdirs) < cmpCount {
			cmpCount = len(result.sortdirs)
		}
		for c := 0; c < cmpCount; c++ {
			av := result.scols[c](a)
			bv := result.scols[c](b)
			if scm.ToBool(result.sortdirs[c](av, bv)) {
				return true
			}
			if scm.ToBool(result.sortdirs[c](bv, av)) {
				return false
			}
		}
		return itemPos[a] < itemPos[b]
	}
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
	if len(sortcols) > 0 {
		if limit >= 0 && limitPartitionCols == 0 {
			// ORDER BY ... LIMIT only needs the best k rows from each shard.
			// Keeping all matching rows and fully sorting them makes small-LIMIT
			// queries degenerate into an expensive full sort with dynamic Scheme
			// comparators, which dominated the multishard regression.
			result.items = topKByOrder(result.items, offset+limit, lessByID)
		} else {
			hybridsort.Slice(result.items, func(i, j int) bool {
				return lessByID(result.items[i], result.items[j])
			})
		}
	}
	// Shard-local per-partition pruning: keep at most offset+limit items per
	// partition. This reduces what goes into the cross-shard globalqueue merge.
	// When limitPartitionCols == 0 this is a single partition (= global limit).
	if limit >= 0 {
		perPart := offset + limit
		if perPart < 0 {
			perPart = len(result.items) // overflow guard
		}
		var pruned []uint32
		var prevPK []scm.Scmer
		partCount := 0
		for _, idx := range result.items {
			curPK := make([]scm.Scmer, limitPartitionCols)
			for c := 0; c < limitPartitionCols; c++ {
				curPK[c] = result.scols[c](idx)
			}
			if prevPK == nil || !pkEqual(prevPK, curPK) {
				partCount = 0
				prevPK = curPK
			}
			if partCount < perPart {
				pruned = append(pruned, idx)
			}
			partCount++
		}
		result.items = pruned
	}
	return
}
