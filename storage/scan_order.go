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
	// scan_order_multi args: 0=fn, 1=tx, 2=tables, 3=filterCols, 4=filterFns,
	// 5=sortcols, 6=sortdirs, 7=perTableOffset, 8=perTableLimit,
	// 9=partCols, 10=offset, 11=limit, 12=mapCols, 13=mapFns,
	// 14=reduce, 15=neutral, 16=isOuter
	for i := 1; i <= 13 && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	oc.Ome.IncrLoopDepth()
	if len(v) > 14 && !v[14].IsNil() {
		oc.SetCallbackOwned([]bool{true, false})
		v[14], _ = oc.OptimizeSub(v[14], true)
	}
	if len(v) > 15 {
		v[15], _ = oc.OptimizeSub(v[15], true)
	}
	if len(v) > 16 {
		v[16], _ = oc.OptimizeSub(v[16], true)
	}
	oc.Ome.DecrLoopDepth()
	return scm.NewSlice(v), nil
}

func optimizeScanOrder(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	// NOTE: scan_order has no reduce2, so batch-rewrite cannot flush the last
	// partial batch. Disabled until scan_order gains reduce2 or an alternative
	// flush mechanism is implemented.
	// if rewritten := tryScanOrderBatchRewrite(v); !rewritten.IsNil() {
	// 	return oc.OptimizeSub(rewritten, useResult)
	// }
	mapEnd, reduceIdx, neutralIdx, outerIdx := 11, 12, 13, 14
	for i := 1; i <= mapEnd && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	oc.Ome.IncrLoopDepth()
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
	oc.Ome.DecrLoopDepth()
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
	shard          *storageShard
	items          []uint32 // TODO: refactor to chan, so we can block generating too much entries
	candidateCount int64
	err            scanError
	scols          []func(uint32) scm.Scmer // sort criteria column reader
	sortdirs       []func(...scm.Scmer) scm.Scmer
	mapper         *ShardMapReducer
	callbackCols   []string  // per-table map columns (for multi-table merge)
	callback       scm.Scmer // per-table map function (for multi-table merge)
	tableIdx       int       // index into scanOrderMulti tables slice; 0 for single-table scan_order
}

// scanOrderResult bundles per-shard outputs for ordered scans.
type scanOrderResult struct {
	res            *shardqueue
	err            scanError // err.r != nil indicates an error
	inputCount     int64
	candidateCount int64
	outputCount    int64
}

type scanOrderStats struct {
	boundaries     boundaries
	inputCount     int64
	candidateCount int64
	outputCount    int64
	analyzeNs      int64
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
	recset        *recSet
	conditionCols []string
	condition     scm.Scmer
	sortcols      []scm.Scmer
	callbackCols  []string
	callback      scm.Scmer
	// perTableOffset / perTableLimit: -1 disables per-table limiting for this
	// table; otherwise the first `perTableOffset` rows (in merge order) are
	// skipped and at most `perTableLimit` rows are emitted from this table.
	// NOTE: only well-defined when per-table sort direction matches the global
	// merge direction (shared sortdirs). Callers must enforce this.
	perTableOffset int
	perTableLimit  int
}

func (s *scanOrderTableSpec) carrierTable() *table {
	if s.recset != nil {
		return s.recset.table
	}
	return s.table
}

// extendBoundariesWithSortCols appends sort columns to the boundaries when all
// existing filter boundaries are point lookups and the comparators are
// index-order compatible (ASC). This lets the shard return rows already sorted
// by ORDER BY, reducing the cross-shard merge to merging pre-sorted runs.
func extendBoundariesWithSortCols(b boundaries, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer) (boundaries, bool) {
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
		return b, false
	}
	addedSortCols := 0
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
				addedSortCols++
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
			addedSortCols++
		}
	}
	return b, addedSortCols > 0
}

func recSetBoundaryCallCount(conditionCols []string, condition scm.Scmer) int {
	var p scm.Proc
	if condition.IsProc() {
		p = *condition.Proc()
	} else if si, ok := condition.Any().(scm.Proc); ok {
		p = si
	} else {
		return 0
	}
	var params []scm.Scmer
	if p.Params.IsSlice() {
		params = p.Params.Slice()
	}
	paramIsRecSetContains := func(node scm.Scmer) bool {
		if node.IsSymbol() {
			name := node.String()
			for i, sym := range params {
				if i < len(conditionCols) && sym.IsSymbol() && sym.String() == name {
					return conditionCols[i] == "$recset_contains"
				}
			}
		}
		if node.IsNthLocalVar() {
			idx := int(node.NthLocalVar())
			return idx < len(conditionCols) && conditionCols[idx] == "$recset_contains"
		}
		return false
	}
	var walk func(scm.Scmer) int
	walk = func(node scm.Scmer) int {
		if !node.IsSlice() {
			return 0
		}
		items := node.Slice()
		if len(items) == 0 {
			return 0
		}
		count := 0
		if paramIsRecSetContains(items[0]) {
			count++
		}
		for _, item := range items[1:] {
			count += walk(item)
		}
		return count
	}
	return walk(p.Body)
}

// scanOrderMulti performs an ordered scan across one or more tables, merging
// results from all tables' shards into a single globally sorted stream.
// Each table has its own filter, sort columns and map function, but sort
// directions, offset/limit, reduce and neutral are shared.
func scanOrderMulti(currentTx *TxContext, tables []scanOrderTableSpec, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool) scm.Scmer {
	execStart := time.Now()
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
	var wg sync.WaitGroup
	querySeq := scm.CurrentQuerySeq()
	stats := make([]scanOrderStats, len(tables))

	// Launch shard-parallel scans for each table
	for ti := range tables {
		spec := &tables[ti]
		t := spec.carrierTable()
		touchTempColumns(t, spec.conditionCols, spec.callbackCols)

		// Per-table top-K hint: when perTableLimit is set, each shard only
		// needs to return the top (perTableOffset + perTableLimit) rows in
		// per-table sort order. This prunes work at the shard level before
		// the merge enforces the per-table offset/limit globally.
		shardTotalLimit := total_limit
		if spec.perTableLimit >= 0 {
			ptTotal := spec.perTableLimit
			if spec.perTableOffset > 0 {
				ptTotal += spec.perTableOffset
			}
			if shardTotalLimit < 0 || ptTotal < shardTotalLimit {
				shardTotalLimit = ptTotal
			}
		}

		// Boundary analysis per table
		analyzeStart := time.Now()
		bounds := extractBoundaries(spec.conditionCols, spec.condition)
		bounds, recsetBoundary := splitRecSetBoundary(bounds, t)
		reorderByFrequency(bounds, t)
		bounds, indexCoversOrder := extendBoundariesWithSortCols(bounds, spec.sortcols, sortdirs)
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
		stats[ti].boundaries = bounds
		stats[ti].analyzeNs = analyzeNs

		// Capture closure variables
		callbackCols := spec.callbackCols
		callback := spec.callback
		conditionCols := spec.conditionCols
		condition := spec.condition
		sortcols := spec.sortcols
		tableBounds := bounds
		tableIdx := ti
		shardLimit := shardTotalLimit
		orderedEarlyLimit := indexCoversOrder && shardLimit >= 0 && limitPartitionCols == 0
		recsetFilter := recsetBoundary

		if spec.recset != nil {
			if len(sortcols) > 0 {
				wg.Add(1)
				go func(parts []recSetShard) {
					withTxSession(currentTx, func() scm.Scmer {
						defer wg.Done()
						defer func() {
							if r := recover(); r != nil {
								q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
							}
						}()
						for _, part := range parts {
							if part.count == 0 {
								continue
							}
							if ss != nil && ss.IsKilledSeq(querySeq) {
								panic("query killed")
							}
							func(part recSetShard) {
								defer func() {
									if r := recover(); r != nil {
										q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
									}
								}()
								res := part.shard.scan_order_recids(part.recids, conditionCols, condition, sortcols, sortdirs, limitPartitionCols, offset, shardLimit, callbackCols, currentTx, ss, querySeq)
								res.callbackCols = callbackCols
								res.callback = callback
								res.tableIdx = tableIdx
								q_ <- scanOrderResult{res: res, inputCount: part.count, candidateCount: part.count, outputCount: int64(len(res.items))}
							}(part)
						}
						return scm.NewNil()
					})
				}(spec.recset.shards)
			} else {
				for i := range spec.recset.shards {
					part := spec.recset.shards[i]
					if part.count == 0 {
						continue
					}
					wg.Add(1)
					go func(part recSetShard) {
						withTxSession(currentTx, func() scm.Scmer {
							defer wg.Done()
							if ss != nil && ss.IsKilledSeq(querySeq) {
								panic("query killed")
							}
							defer func() {
								if r := recover(); r != nil {
									q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
								}
							}()
							res := part.shard.scan_order_recids(part.recids, conditionCols, condition, sortcols, sortdirs, limitPartitionCols, offset, shardLimit, callbackCols, currentTx, ss, querySeq)
							res.callbackCols = callbackCols
							res.callback = callback
							res.tableIdx = tableIdx
							q_ <- scanOrderResult{res: res, inputCount: part.count, candidateCount: part.count, outputCount: int64(len(res.items))}
							return scm.NewNil()
						})
					}(part)
				}
			}
		} else {
			done := t.iterateShardsParallel(tableBounds, func(s *storageShard, solo bool) {
				if ss != nil && ss.IsKilled() {
					panic("query killed")
				}
				defer func() {
					if r := recover(); r != nil {
						q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
					}
				}()
				res := s.scan_order(tableBounds, lower, upperLast, conditionCols, condition, sortcols, sortdirs, limitPartitionCols, offset, shardLimit, callbackCols, currentTx, ss, orderedEarlyLimit, recsetFilter)
				res.callbackCols = callbackCols
				res.callback = callback
				res.tableIdx = tableIdx
				q_ <- scanOrderResult{res: res, inputCount: int64(s.Count()), candidateCount: res.candidateCount, outputCount: int64(len(res.items))}
			})
			if done != nil {
				wg.Add(1)
				go func(ch <-chan struct{}) {
					<-ch
					wg.Done()
				}(done)
			}
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
		if msg.res != nil && msg.res.tableIdx >= 0 && msg.res.tableIdx < len(stats) {
			tableStats := &stats[msg.res.tableIdx]
			tableStats.inputCount += msg.inputCount
			tableStats.candidateCount += msg.candidateCount
			tableStats.outputCount += msg.outputCount
		}
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

	// Per-table offset/limit state. Applied BEFORE partition/global logic.
	// -1 entries disable per-table limiting for that table.
	tablePartOffset := make([]int, len(tables))
	tablePartLimit := make([]int, len(tables))
	for ti := range tables {
		tablePartOffset[ti] = tables[ti].perTableOffset
		tablePartLimit[ti] = tables[ti].perTableLimit
	}

	for !breakCaught && len(q.q) > 0 {
		qx := q.q[0]

		if len(qx.items) == 0 {
			heap.Pop(&q)
			continue
		}

		// Per-table limit: discard remaining shardqueue items once the
		// per-table quota is exhausted. Sibling shardqueues of the same
		// tableIdx are dropped as they reach the heap top.
		ti := qx.tableIdx
		if ti < len(tablePartLimit) && tablePartLimit[ti] == 0 {
			heap.Pop(&q)
			continue
		}
		// Per-table offset skip: consume leading items without emitting.
		if ti < len(tablePartOffset) && tablePartOffset[ti] > 0 {
			tablePartOffset[ti]--
			qx.items = qx.items[1:]
			if len(qx.items) > 0 {
				heap.Fix(&q, 0)
			} else {
				heap.Pop(&q)
			}
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
		if ti < len(tablePartLimit) && tablePartLimit[ti] > 0 {
			tablePartLimit[ti]--
		}

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
		callbackFn := scm.OptimizeProcToSerialFunction(cb)
		aggregateFn := scm.OptimizeProcToSerialFunction(aggregate)
		nullRow := buildOuterNullCallbackRow(cbCols)
		akkumulator = aggregateFn(akkumulator, callbackFn(nullRow...))
	}
	execNs := time.Since(execStart).Nanoseconds()
	for i := range tables {
		tableStats := stats[i]
		if !Settings.ScanDebugging && tableStats.candidateCount <= int64(Settings.AnalyzeMinItems) {
			continue
		}
		spec := &tables[i]
		tbl := spec.carrierTable()
		filterEnc := ""
		if proc, ok := spec.condition.Any().(scm.Proc); ok {
			var params []scm.Scmer
			if proc.Params.IsSlice() {
				params = proc.Params.Slice()
			} else if arr, ok := proc.Params.Any().([]scm.Scmer); ok {
				params = arr
			}
			filterEnc = encodeScmerToString(proc.Body, spec.conditionCols, params)
		}
		var sb strings.Builder
		for j, sortcol := range spec.sortcols {
			if j > 0 {
				sb.WriteByte('|')
			}
			if sortcol.IsString() {
				sb.WriteString(sortcol.String())
			} else {
				encodeScmer(sortcol, &sb, nil, nil)
			}
		}
		orderEnc := sb.String()
		indexColsEnc := boundaryIndexCols(tableStats.boundaries)
		go safeLogScan(tbl.schema.Name, tbl.Name, true, filterEnc, orderEnc, indexColsEnc, tableStats.inputCount, tableStats.candidateCount, tableStats.outputCount, tableStats.analyzeNs, execNs)
	}
	return akkumulator
}

// scan_order delegates to scanOrderMulti with a single-element table spec.
func (t *table) scan_order(currentTx *TxContext, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool) scm.Scmer {
	return scanOrderMulti(currentTx, []scanOrderTableSpec{{
		table:          t,
		conditionCols:  conditionCols,
		condition:      condition,
		sortcols:       sortcols,
		callbackCols:   callbackCols,
		callback:       callback,
		perTableOffset: -1,
		perTableLimit:  -1,
	}}, sortdirs, limitPartitionCols, offset, limit, aggregate, neutral, isOuter)
}

func (r *recSet) scan_order(currentTx *TxContext, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool) scm.Scmer {
	if currentTx == nil {
		currentTx = r.tx
	}
	return scanOrderMulti(currentTx, []scanOrderTableSpec{{
		recset:         r,
		conditionCols:  conditionCols,
		condition:      condition,
		sortcols:       sortcols,
		callbackCols:   callbackCols,
		callback:       callback,
		perTableOffset: -1,
		perTableLimit:  -1,
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

func (t *storageShard) scan_order(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, currentTx *TxContext, ss *scm.SessionState, orderedEarlyLimit bool, recsetFilter *recSet) (result *shardqueue) {
	result = new(shardqueue)
	result.shard = t
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}
	var recsetPart *recSetShard
	if recsetFilter != nil {
		recsetPart = recsetFilter.shardEntry(t)
		if recsetPart == nil || recsetPart.count == 0 {
			result.items = nil
			return
		}
	}
	recsetBoundaryCoversCondition := recsetPart != nil && recSetBoundaryCallCount(conditionCols, condition) == 1
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
		if scol.IsProc() {
			proc := scol.Proc()
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
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	for i, k := range conditionCols { // iterate over columns
		if k == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
			if recsetBoundaryCoversCondition {
				fnptr = recSetAlreadyMatchedClosure()
			}
			getter := func(id uint32, batchid uint32) scm.Scmer {
				return scm.NewClosure(fnptr, id)
			}
			conditionGetters[i] = getter
			continue
		}
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
		t.iterateIndex(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf[:], true, func(batch []uint32) bool {
			result.candidateCount += int64(len(batch))
			// filter in-place: overwrite batch with passing IDs
			outN := 0
			for _, idx := range batch {
				if recsetPart != nil && !recsetPart.contains(idx) {
					continue
				}
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
						if conditionGetters[i] != nil {
							cdataset[i] = conditionGetters[i](idx, 0)
						} else if cNeedsTxReader[i] {
							cdataset[i] = cReaders[i].GetValue(idx)
						} else {
							cdataset[i] = k.GetValue(idx)
						}
					}
				} else {
					// value from delta storage
					// prepare&call condition function
					for i, k := range conditionCols { // iterate over columns
						if conditionGetters[i] != nil {
							cdataset[i] = conditionGetters[i](idx, 0)
						} else if cNeedsTxReader[i] {
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
			if orderedEarlyLimit && resultN >= limit {
				return false
			}
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
