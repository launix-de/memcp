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
import "math"
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
	// 14=reduce, 15=neutral, 16=isOuter, 17=notFoundValue
	rawReduce := scm.NewNil()
	if len(v) > 14 {
		rawReduce = v[14]
	}
	for i := 1; i <= 13 && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	neutralType := unknownScanType()
	if len(v) > 15 {
		v[15], neutralType = oc.OptimizeSub(v[15], true)
		neutralType = normalizeScanType(neutralType)
	}
	oc.Ome.IncrLoopDepth()
	if !rawReduce.IsNil() {
		v[14], _ = oc.OptimizeReducerCallback(rawReduce, neutralType, unknownScanType())
	}
	if len(v) > 16 {
		v[16], _ = oc.OptimizeSub(v[16], true)
	}
	if len(v) > 17 {
		v[17], _ = oc.OptimizeSub(v[17], true)
	}
	oc.Ome.DecrLoopDepth()
	return scm.NewSlice(v), nil
}

func optimizeScanOrder(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	if rewritten := tryScanInvariantFilterRewrite(v); !rewritten.IsNil() {
		if result, td, accepted := oc.OptimizeRewrite(scm.NewSlice(v), rewritten, useResult, scm.OptimizerRewriteContract{
			Name: "scan-order-invariant-filter", PreconditionsMet: true, MaxGrowthNodes: 64,
		}); accepted {
			return result, td
		}
	}
	// NOTE: scan_order has no reduce2, so batch-rewrite cannot flush the last
	// partial batch. Disabled until scan_order gains reduce2 or an alternative
	// flush mechanism is implemented.
	// if rewritten := tryScanOrderBatchRewrite(v); !rewritten.IsNil() {
	// 	return oc.OptimizeSub(rewritten, useResult)
	// }
	mapEnd, reduceIdx, neutralIdx, outerIdx, notFoundIdx, postOrderColsIdx, postOrderFilterIdx := 11, 12, 13, 14, 15, 16, 17
	rawReduce := scm.NewNil()
	if len(v) > reduceIdx {
		rawReduce = v[reduceIdx]
	}
	for i := 1; i <= mapEnd && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	neutralType := unknownScanType()
	if len(v) > neutralIdx {
		v[neutralIdx], neutralType = oc.OptimizeSub(v[neutralIdx], true)
		neutralType = normalizeScanType(neutralType)
	}
	oc.Ome.IncrLoopDepth()
	if !rawReduce.IsNil() {
		// The value shape differs between shard-local and shard-collect phases,
		// but the accumulator starts from the same structured neutral value.
		v[reduceIdx], _ = oc.OptimizeReducerCallback(rawReduce, neutralType, unknownScanType())
	}
	if len(v) > outerIdx {
		v[outerIdx], _ = oc.OptimizeSub(v[outerIdx], true)
	}
	if len(v) > notFoundIdx {
		v[notFoundIdx], _ = oc.OptimizeSub(v[notFoundIdx], true)
	}
	if len(v) > postOrderColsIdx {
		v[postOrderColsIdx], _ = oc.OptimizeSub(v[postOrderColsIdx], true)
	}
	if len(v) > postOrderFilterIdx {
		v[postOrderFilterIdx], _ = oc.OptimizeSub(v[postOrderFilterIdx], true)
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
	shard           *storageShard
	items           []uint32 // TODO: refactor to chan, so we can block generating too much entries
	universe        uint32   // visible recid upper bound captured by the shard scan
	candidateCount  int64
	err             scanError
	scols           []func(uint32) scm.Scmer // sort criteria column reader
	sortdirs        []func(...scm.Scmer) scm.Scmer
	sortless        []func(scm.Scmer, scm.Scmer) bool
	mapper          *ShardMapReducer
	postOrderMapper *ShardMapReducer
	callbackCols    []string  // per-table map columns (for multi-table merge)
	callback        scm.Scmer // per-table map function (for multi-table merge)
	tableIdx        int       // index into scanOrderMulti tables slice; 0 for single-table scan_order
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
	if len(s.sortless) < cmpCount {
		cmpCount = len(s.sortless)
	}
	for c := 0; c < cmpCount; c++ {
		a := s.scols[c](s.items[i])
		b := s.scols[c](s.items[j])
		if s.sortless[c](a, b) {
			return true
		} else if s.sortless[c](b, a) {
			return false
		} // else: go to next level
		// otherwise: move on to c++
	}
	return s.items[i] < s.items[j]
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
	if len(s.q[i].sortless) < cmpCount {
		cmpCount = len(s.q[i].sortless)
	}
	if len(s.q[j].sortless) < cmpCount {
		cmpCount = len(s.q[j].sortless)
	}
	for c := 0; c < cmpCount; c++ {
		a := s.q[i].scols[c](s.q[i].items[0])
		b := s.q[j].scols[c](s.q[j].items[0])
		if s.q[i].sortless[c](a, b) {
			return true
		} else if s.q[i].sortless[c](b, a) {
			return false
		} // else: go to next level
		// otherwise: move on to c++
	}
	// SQL leaves peer ordering unspecified, but adaptive batch windows need the
	// same physical order on every continuation. Shard UUID plus recid supplies
	// that internal total order without reading or sorting another column. This
	// also makes the empty-ORDER path repeatable while retaining greedy scans.
	if s.q[i].tableIdx != s.q[j].tableIdx {
		return s.q[i].tableIdx < s.q[j].tableIdx
	}
	for k := range s.q[i].shard.uuid {
		if s.q[i].shard.uuid[k] != s.q[j].shard.uuid[k] {
			return s.q[i].shard.uuid[k] < s.q[j].shard.uuid[k]
		}
	}
	return s.q[i].items[0] < s.q[j].items[0]
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
	table           *table
	recset          *recSet
	conditionCols   []string
	condition       scm.Scmer
	acceptCols      []string
	accept          scm.Scmer
	sortcols        []scm.Scmer
	callbackCols    []string
	callback        scm.Scmer
	postOrderCols   []string
	postOrderFilter scm.Scmer
	// recordVisitor is an internal sink used by scan_order_batch_accept. When
	// present, globally ordered record IDs are delivered directly instead of
	// being mapped. The visitor must consume the slice before returning.
	recordVisitor func(*shardqueue, []uint32)
	// perTableOffset / perTableLimit: -1 disables per-table limiting for this
	// table; otherwise the first `perTableOffset` rows (in merge order) are
	// skipped and at most `perTableLimit` rows are emitted from this table.
	// NOTE: only well-defined when per-table sort direction matches the global
	// merge direction (shared sortdirs). Callers must enforce this.
	perTableOffset int
	perTableLimit  int
}

func streamScanOrderItems(spec *scanOrderTableSpec, queue *shardqueue, acc scm.Scmer, recids []uint32) (scm.Scmer, bool) {
	if spec.recordVisitor != nil {
		spec.recordVisitor(queue, recids)
		return acc, false
	}
	return streamOrBreak(queue.mapper, acc, recids)
}

func (s *scanOrderTableSpec) backingTable() *table {
	if s.recset != nil {
		return s.recset.table
	}
	return s.table
}

// extendBoundariesWithSortCols inserts sort columns before candidate matchers
// when all existing filter boundaries are point lookups. The relation callback
// is the complete ordering contract, including collation, direction and NULLs.
func extendBoundariesWithSortCols(b boundaries, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer) (boundaries, bool) {
	original := b
	allEq := true
	for _, bi := range b {
		if !boundaryIsPoint(bi) {
			allEq = false
			break
		}
	}
	canAppendSortPrefix := len(sortcols) > 0 && len(sortdirs) >= len(sortcols)
	if !allEq || !canAppendSortPrefix {
		return b, false
	}
	insertSortedBoundary := func(bound columnboundaries) {
		at := len(b)
		for i := range b {
			if !b[i].matcher.IsSorted() {
				at = i
				break
			}
		}
		b = append(b, columnboundaries{})
		copy(b[at+1:], b[at:])
		b[at] = bound
	}
	for i, scol := range sortcols {
		if sortdirs[i] == nil {
			return original, false
		}
		orderMeta := orderRelationMeta(sortdirs[i])
		if scol.IsString() {
			col := scol.String()
			already := false
			for _, bi := range b {
				if bi.col == col && bi.matcher.IsSorted() {
					already = true
					break
				}
			}
			if !already {
				insertSortedBoundary(columnboundaries{col: col, matcher: RangeMatcher, lower: scm.NewNil(), upper: scm.NewNil(), order: sortdirs[i], orderMeta: orderMeta})
			}
			continue
		}
		proc, ok := scol.Any().(scm.Proc)
		if !ok && scol.IsProc() {
			proc = *scol.Proc()
			ok = true
		}
		if !ok {
			return original, false
		}
		var procParams []scm.Scmer
		if proc.Params.IsSlice() {
			procParams = proc.Params.Slice()
		}
		if len(procParams) == 0 {
			return original, false
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
			return original, false
		}
		canon := canonicalColName(proc.Body, procParams, sortCondCols)
		mc, mf := buildComputedFn(proc.Body, proc.Params, proc.En, sortCondCols)
		if mf.IsNil() || mc == nil {
			return original, false
		}
		already := false
		for _, bi := range b {
			if bi.col == canon && bi.matcher.IsSorted() {
				already = true
				break
			}
		}
		if !already {
			insertSortedBoundary(columnboundaries{col: canon, matcher: RangeMatcher, lower: scm.NewNil(), upper: scm.NewNil(), order: sortdirs[i], orderMeta: orderMeta, mapCols: mc, mapFn: mf})
		}
	}
	return b, len(sortcols) > 0
}

func indexCoversBoundaryOrder(index *StorageIndex, active bool, bounds boundaries, effectiveCols int) bool {
	if !active || index == nil {
		return false
	}
	orderCols := 0
	for i, boundary := range bounds {
		if boundary.order == nil || boundaryIsPoint(boundary) {
			continue
		}
		orderCols++
		if i >= effectiveCols || i >= len(index.Cols) || i >= len(index.ColOrder) {
			return false
		}
		orderMeta := boundary.orderMeta
		if orderMeta == "" {
			orderMeta = orderRelationMeta(boundary.order)
		}
		if index.Cols[i] != boundary.col || i >= len(index.ColOrderMeta) || index.ColOrderMeta[i] != orderMeta {
			return false
		}
	}
	return orderCols > 0
}

// orderedIndexUsageWeight estimates how much of one full scan an ordered
// index would save. Building the permutation costs O(n log n), while an
// unindexed Top-k scan costs O(n log k). Savings therefore accumulate slowly
// for tiny windows and at the normal rate for large offsets or full scans.
func orderedIndexUsageWeight(rows, kept int) float64 {
	if rows <= 1 || kept < 0 || kept >= rows {
		return 1
	}
	if kept < 1 {
		kept = 1
	}
	weight := 2 * math.Max(1, math.Log2(float64(kept)+1)) / math.Log2(float64(rows)+1)
	if weight > 1 {
		return 1
	}
	return weight
}

// orderedScanIndexUsageWeight distinguishes a selective access path from a
// pure ORDER BY Top-k. A restrictive boundary can avoid the complete scan, so
// it must train the autoindex at the normal rate even when the result window is
// tiny. Unbounded order boundaries only accelerate Top-k and retain the lower
// build weight above.
func orderedScanIndexUsageWeight(bounds boundaries, rows, kept int) float64 {
	for _, bound := range bounds {
		if !boundaryIsUnboundedOrder(bound) {
			return 1
		}
	}
	return orderedIndexUsageWeight(rows, kept)
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

func recSetHooksCoverCondition(bounds boundaries, lower []scm.Scmer, backingTable *table, conditionCols []string, condition scm.Scmer) bool {
	want := recSetBoundaryCallCount(conditionCols, condition)
	if want == 0 {
		return false
	}
	covered := 0
	for i := 0; i < len(lower) && i < len(bounds); i++ {
		bound := bounds[i]
		if !matcherKindEqual(bound.matcher, RecSetMatcher) {
			continue
		}
		if !bound.lower.IsCustom(TagRecSet) {
			return false
		}
		set := RecSetFromScmer(bound.lower)
		if set == nil || set.table != backingTable {
			return false
		}
		covered++
	}
	return covered == want
}

// scanOrderMulti performs an ordered scan across one or more tables, merging
// results from all tables' shards into a single globally sorted stream.
// Each table has its own filter, sort columns and map function, but sort
// directions, offset/limit, reduce and neutral are shared.
func scanOrderMulti(currentTx *TxContext, tables []scanOrderTableSpec, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool, notFoundValue scm.Scmer) scm.Scmer {
	execStart := time.Now()
	ss := SessionStateFromTx(currentTx)

	total_limit := -1
	if limitPartitionCols == 0 && limit >= 0 {
		total_limit = offset + limit
	}

	var q globalqueue
	resultBufferSize := 0
	for i := range tables {
		if tables[i].recset != nil {
			resultBufferSize += len(tables[i].recset.shards)
		} else {
			resultBufferSize += tables[i].backingTable().shardResultBufferSize()
		}
	}
	if resultBufferSize < 1 {
		resultBufferSize = 1
	}
	q_ := make(chan scanOrderResult, resultBufferSize)
	doneChannels := make([]<-chan struct{}, 0, len(tables))
	querySeq := querySeqFromTx(currentTx)
	stats := make([]scanOrderStats, len(tables))

	// Launch shard-parallel scans for each table
	for ti := range tables {
		spec := &tables[ti]
		t := spec.backingTable()
		touchTempColumns(t, spec.conditionCols, spec.callbackCols)
		touchTempColumns(t, spec.acceptCols, nil)

		// Per-table top-K hint: when perTableLimit is set, each shard only
		// needs to return the top (perTableOffset + perTableLimit) rows in
		// per-table sort order. This prunes work at the shard level before
		// the merge enforces the per-table offset/limit globally.
		shardTotalLimit := total_limit
		// A post-order predicate can reject any prefix of the ordered stream.
		// Shards therefore cannot truncate before the global merge has applied it.
		if !spec.postOrderFilter.IsNil() {
			shardTotalLimit = -1
		}
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
		reorderByFrequency(bounds, t)
		bounds, _ = extendBoundariesWithSortCols(bounds, spec.sortcols, sortdirs)
		bounds = appendRecSetBoundary(bounds, spec.recset)
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
		acceptCols := spec.acceptCols
		accept := spec.accept
		sortcols := spec.sortcols
		tableBounds := bounds
		tableIdx := ti
		shardLimit := shardTotalLimit

		if spec.recset != nil {
			if len(sortcols) > 0 {
				// Ordered RecSet parts intentionally form one sequential producer.
				// The result channel is sized for every part, so a one-worker path
				// can execute directly without a goroutine or waiter.
				runFanoutTasks(currentTx, 1, func(_ int, _ bool) {
					parts := spec.recset.shards
					withTxSession(currentTx, func() scm.Scmer {
						defer func() {
							if r := recover(); r != nil {
								q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
							}
						}()
						for _, part := range parts {
							if part.count == 0 {
								continue
							}
							// Cancellation contract: check only at the scheduling boundary, before entering
							// the shard. Once entered, a shard runs atomically without cancellation checks.
							if ss != nil && ss.IsKilledSeq(querySeq) {
								panic("query killed")
							}
							func(part recSetShard) {
								defer func() {
									if r := recover(); r != nil {
										q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
									}
								}()
								res := part.shard.scan_order(tableBounds, lower, upperLast, conditionCols, condition, acceptCols, accept, sortcols, sortdirs, limitPartitionCols, offset, shardLimit, callbackCols, currentTx, ss)
								res.callbackCols = callbackCols
								res.callback = callback
								res.tableIdx = tableIdx
								q_ <- scanOrderResult{res: res, inputCount: part.count, candidateCount: part.count, outputCount: int64(len(res.items))}
							}(part)
						}
						return scm.NewNil()
					})
				})
			} else {
				activeParts := make([]int, 0, len(spec.recset.shards))
				for i := range spec.recset.shards {
					if spec.recset.shards[i].count > 0 {
						activeParts = append(activeParts, i)
					}
				}
				done := runFanoutTasks(currentTx, len(activeParts), func(taskIndex int, _ bool) {
					part := spec.recset.shards[activeParts[taskIndex]]
					withTxSession(currentTx, func() scm.Scmer {
						defer func() {
							if r := recover(); r != nil {
								q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
							}
						}()
						// Cancellation contract: check only at the scheduling boundary, before entering
						// the shard. Once entered, a shard runs atomically without cancellation checks.
						if ss != nil && ss.IsKilledSeq(querySeq) {
							panic("query killed")
						}
						res := part.shard.scan_order(tableBounds, lower, upperLast, conditionCols, condition, acceptCols, accept, sortcols, sortdirs, limitPartitionCols, offset, shardLimit, callbackCols, currentTx, ss)
						res.callbackCols = callbackCols
						res.callback = callback
						res.tableIdx = tableIdx
						q_ <- scanOrderResult{res: res, inputCount: part.count, candidateCount: part.count, outputCount: int64(len(res.items))}
						return scm.NewNil()
					})
				})
				if done != nil {
					doneChannels = append(doneChannels, done)
				}
			}
		} else {
			done := t.iterateShardsParallel(currentTx, tableBounds, func(s *storageShard, solo bool) {
				defer func() {
					if r := recover(); r != nil {
						q_ <- scanOrderResult{err: scanError{r, string(debug.Stack())}}
					}
				}()
				// Cancellation contract: check only at the scheduling boundary, before entering
				// the shard. Once entered, a shard runs atomically without cancellation checks.
				if ss != nil && ss.IsKilledSeq(querySeq) {
					panic("query killed")
				}
				res := s.scan_order(tableBounds, lower, upperLast, conditionCols, condition, acceptCols, accept, sortcols, sortdirs, limitPartitionCols, offset, shardLimit, callbackCols, currentTx, ss)
				res.callbackCols = callbackCols
				res.callback = callback
				res.tableIdx = tableIdx
				q_ <- scanOrderResult{res: res, inputCount: int64(s.Count()), candidateCount: res.candidateCount, outputCount: int64(len(res.items))}
			})
			if done != nil {
				doneChannels = append(doneChannels, done)
			}
		}

	}

	// Synchronous scans have already filled the bounded result channel. Only a
	// real fanout needs a coordinator goroutine while this caller consumes.
	if len(doneChannels) == 0 {
		close(q_)
	} else {
		go func() {
			for _, done := range doneChannels {
				<-done
			}
			close(q_)
		}()
	}

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
			if tables[msg.res.tableIdx].postOrderFilter.IsNil() {
				tableStats.outputCount += msg.outputCount
			}
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
		spec := &tables[sq.tableIdx]
		if spec.recordVisitor == nil {
			sq.mapper = sq.shard.OpenMapReducer(sq.callbackCols, sq.callback, aggregate, false, 0, nil, currentTx)
		}
		if !spec.postOrderFilter.IsNil() {
			sq.postOrderMapper = sq.shard.OpenMapReducer(
				spec.postOrderCols,
				spec.postOrderFilter,
				scm.NewFunc(func(args ...scm.Scmer) scm.Scmer { return args[1] }),
				false, 0, nil, currentTx,
			)
		}
	}

	// The optional record visitor makes this batch escape even though ordered
	// scans consume it synchronously. Keep one exclusive, pointer-free 4 KiB
	// workspace per active merge instead of allocating it for every invocation.
	buf, pooledFullBuf, pooledPointBuf := acquireScanIDBuffer(defaultScanBufferSize)
	defer releaseScanIDBuffer(pooledFullBuf, pooledPointBuf)
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

		ti := qx.tableIdx
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
				akkumulator, breakCaught = streamScanOrderItems(&tables[bufShard.tableIdx], bufShard, akkumulator, buf[:bufN])
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

		// This predicate intentionally runs after ordering, but before offsets and
		// limits are counted. It supports expensive nested lookup acceptance without
		// turning callback-controlled panics into an execution-control mechanism.
		if qx.postOrderMapper != nil && !scm.ToBool(qx.postOrderMapper.MapOne(peekItem)) {
			qx.items = qx.items[1:]
			if len(qx.items) > 0 {
				heap.Fix(&q, 0)
			} else {
				heap.Pop(&q)
			}
			continue
		}
		if qx.postOrderMapper != nil {
			stats[ti].outputCount++
		}

		// Per-table offsets and limits count only post-order accepted records.
		if ti < len(tablePartLimit) && tablePartLimit[ti] == 0 {
			heap.Pop(&q)
			continue
		}
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
			akkumulator, breakCaught = streamScanOrderItems(&tables[bufShard.tableIdx], bufShard, akkumulator, buf[:bufN])
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
			akkumulator, breakCaught = streamScanOrderItems(&tables[bufShard.tableIdx], bufShard, akkumulator, buf[:bufN])
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
		akkumulator, _ = streamScanOrderItems(&tables[bufShard.tableIdx], bufShard, akkumulator, buf[:bufN])
		hadValue = true
	}
	if !hadValue && isOuter && len(tables) > 0 {
		cbCols := tables[0].callbackCols
		cb := tables[0].callback
		callbackProgram := scm.PrepareSerialProc(cb)
		aggregateProgram := scm.PrepareSerialProc(aggregate)
		nullRow := buildOuterNullCallbackRow(cbCols)
		var aggregateArgs [2]scm.Scmer
		aggregateArgs[0] = akkumulator
		aggregateArgs[1] = callbackProgram.Call(nullRow)
		akkumulator = aggregateProgram.Call(aggregateArgs[:])
	}
	if !hadValue && !isOuter {
		akkumulator = notFoundValue
	}
	execNs := time.Since(execStart).Nanoseconds()
	for i := range tables {
		tableStats := stats[i]
		if !Settings.ScanDebugging && tableStats.candidateCount <= int64(Settings.AnalyzeMinItems) {
			continue
		}
		spec := &tables[i]
		tbl := spec.backingTable()
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
func (t *table) scan_order(currentTx *TxContext, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool, notFoundValue scm.Scmer, postOrderCols []string, postOrderFilter scm.Scmer) scm.Scmer {
	if postOrderFilter.IsNil() && len(sortcols) == 0 && limitPartitionCols == 0 && offset == 0 && limit == 1 && !aggregate.IsNil() && !isOuter {
		return t.scanOrderFirst(currentTx, conditionCols, condition, callbackCols, callback, aggregate, neutral, notFoundValue)
	}
	return scanOrderMulti(currentTx, []scanOrderTableSpec{{
		table:           t,
		conditionCols:   conditionCols,
		condition:       condition,
		sortcols:        sortcols,
		callbackCols:    callbackCols,
		callback:        callback,
		postOrderCols:   postOrderCols,
		postOrderFilter: postOrderFilter,
		perTableOffset:  -1,
		perTableLimit:   -1,
	}}, sortdirs, limitPartitionCols, offset, limit, aggregate, neutral, isOuter, notFoundValue)
}

// scanOrderFirst is the no-order LIMIT 1 specialization of scan_order. It
// preserves the scan operator contract while avoiding the queues, channels,
// and global merge needed by the general ordered multi-shard implementation.
func (t *table) scanOrderFirst(currentTx *TxContext, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, notFoundValue scm.Scmer) scm.Scmer {
	ss := SessionStateFromTx(currentTx)
	querySeq := querySeqFromTx(currentTx)
	bounds := extractBoundaries(conditionCols, condition)
	reorderByFrequency(bounds, t)
	lower, upperLast := indexFromBoundaries(bounds)

	var mu sync.Mutex
	var foundShard *storageShard
	var foundID uint32
	var firstErr scanError
	done := t.iterateShardsParallel(currentTx, bounds, func(shard *storageShard, _ bool) {
		defer func() {
			if recovered := recover(); recovered != nil {
				mu.Lock()
				if firstErr.r == nil {
					firstErr = scanError{recovered, string(debug.Stack())}
				}
				mu.Unlock()
			}
		}()
		mu.Lock()
		finished := foundShard != nil || firstErr.r != nil
		mu.Unlock()
		if finished {
			return
		}
		if ss != nil && ss.IsKilledSeq(querySeq) {
			panic("query killed")
		}
		recid, present := shard.scanFirstRecord(bounds, lower, upperLast, conditionCols, condition, currentTx, ss, nil)
		if !present {
			return
		}
		mu.Lock()
		if foundShard == nil {
			foundShard = shard
			foundID = recid
		}
		mu.Unlock()
	})
	if done != nil {
		<-done
	}
	if firstErr.r != nil {
		panic(firstErr)
	}
	if foundShard == nil {
		return notFoundValue
	}
	mapperAlreadyLocked := foundShard.hasWriteOwnerForTx(currentTx)
	var mapperStorage ShardMapReducer
	var mapperWorkspace shardMapReducerWorkspace
	mapper := &mapperStorage
	if mapReducerCanUseReadWorkspace(callbackCols) {
		prepareReadMapReducerStorage(&mapperStorage, &mapperWorkspace, len(callbackCols))
		foundShard.initReadMapReducer(&mapperStorage, callbackCols, callback, aggregate, mapperAlreadyLocked, currentTx)
	} else {
		mapper = foundShard.OpenMapReducer(callbackCols, callback, aggregate, mapperAlreadyLocked, 0, nil, currentTx)
	}
	result := mapper.Stream(neutral, []uint32{foundID}, nil)
	mapper.FlushSideEffects()
	return result
}

func (r *recSet) scan_order(currentTx *TxContext, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool, notFoundValue scm.Scmer, postOrderCols []string, postOrderFilter scm.Scmer) scm.Scmer {
	if currentTx == nil {
		currentTx = r.tx
	}
	return scanOrderMulti(currentTx, []scanOrderTableSpec{{
		recset:          r,
		conditionCols:   conditionCols,
		condition:       condition,
		sortcols:        sortcols,
		callbackCols:    callbackCols,
		callback:        callback,
		postOrderCols:   postOrderCols,
		postOrderFilter: postOrderFilter,
		perTableOffset:  -1,
		perTableLimit:   -1,
	}}, sortdirs, limitPartitionCols, offset, limit, aggregate, neutral, isOuter, notFoundValue)
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

func (t *storageShard) scan_order(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, acceptCols []string, accept scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, currentTx *TxContext, ss *scm.SessionState) (result *shardqueue) {
	result = new(shardqueue)
	result.shard = t
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}
	recsetBoundaryCoversCondition := recSetHooksCoverCondition(boundaries, lower, t.t, conditionCols, condition)
	conditionProgram := scm.PrepareSerialProc(condition)
	conditionAlwaysTrue := conditionProgram.Kind == scm.SerialProcConstant && scm.ToBool(conditionProgram.Value)
	var acceptProgram *scm.SerialProc
	if !accept.IsNil() {
		prepared := scm.PrepareSerialProc(accept)
		if prepared.Kind != scm.SerialProcConstant || !scm.ToBool(prepared.Value) {
			acceptProgram = &prepared
		}
	}

	// prepare filter function
	cdataset := make([]scm.Scmer, len(conditionCols))
	for i := range cdataset {
		cdataset[i] = scm.NewNil()
	}
	adataset := make([]scm.Scmer, len(acceptCols))
	for i := range adataset {
		adataset[i] = scm.NewNil()
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
				if name == "$tx" {
					txValue := scm.NewAny(currentTx)
					largs[j] = func(uint32) scm.Scmer { return txValue }
				} else {
					largs[j] = t.ColumnReaderTx(currentTx, name)
				}
			}
			procFn := scm.PrepareSerialProc(scol)
			vals := make([]scm.Scmer, len(largs))
			result.scols[i] = func(idx uint32) scm.Scmer {
				for j, getter := range largs {
					vals[j] = getter(idx)
				}
				return procFn.Call(vals)
			}
			continue
		}
		panic("unknown sort criteria: " + scm.String(scol))
	}

	adjustedSortdirs := sortdirs

	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	if t.t.hasTableLock() {
		t.t.waitTableLock(ss, false)
	}

	// main storage — use skipShardReadLock to avoid redundant hasWriteOwner() per column
	var ccols []ColumnStorage
	var cReaders []ColumnReader
	var cNeedsCachedReader []bool
	var conditionGetters []mapArgGetter
	if !conditionAlwaysTrue {
		ccols = make([]ColumnStorage, len(conditionCols))
		cReaders = make([]ColumnReader, len(conditionCols))
		cNeedsCachedReader = make([]bool, len(conditionCols))
		conditionGetters = make([]mapArgGetter, len(conditionCols))
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
			ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
			cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
			if _, ok := ccols[i].(*StorageComputeProxy); ok {
				cNeedsCachedReader[i] = true
			}
		}
	}
	acols := make([]ColumnStorage, len(acceptCols))
	aReaders := make([]ColumnReader, len(acceptCols))
	aNeedsCachedReader := make([]bool, len(acceptCols))
	for i, column := range acceptCols {
		acols[i] = t.getColumnStorageOrPanicEx(column, skipShardReadLock, currentTx)
		aReaders[i] = newCachedColumnReaderTx(acols[i], currentTx)
		if _, ok := acols[i].(*StorageComputeProxy); ok {
			aNeedsCachedReader[i] = true
		}
	}
	// initialize main_count lazily if needed
	t.ensureMainCount(skipShardReadLock)
	// scan loop in read lock
	var maxInsertIndex int
	var visibleUpper uint32
	resultAlreadySorted := len(sortcols) == 0
	func() {
		shardLocked := false
		if !skipShardReadLock {
			t.mu.RLock()
			shardLocked = true
			// Table lock check must happen AFTER shard RLock — race-safe synchronization
			// point (mirrors storageShard.scan logic for the TOCTOU fix).
			if t.t.hasTableLock() {
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
		result.universe = visibleUpper

		// iterate over items (indexed)
		// TODO(memcp): iterateIndexSorted(boundaries, sortcols) to emit tuples in ORDER BY sequence.
		buf, pooledFullBuf, pooledPointBuf := acquireScanIDBuffer(defaultScanBufferSize)
		defer releaseScanIDBuffer(pooledFullBuf, pooledPointBuf)
		resultCap := 1024
		if limitPartitionCols == 0 && limit >= 0 && limit < resultCap {
			resultCap = limit
		}
		if resultCap < 1 {
			resultCap = 1
		}
		result.items = make([]uint32, resultCap)
		resultN := 0
		usageWeight := orderedScanIndexUsageWeight(boundaries, int(visibleUpper), limit)
		// Reused across batches: survived/mainIds scratch lists and one value
		// buffer per non-getter condition column, so a batch's main-storage rows
		// are fetched with one GetValueMulti call per column instead of one
		// GetValue call per row per column.
		var survivedBuf, mainIdsBuf, acceptMainIdsBuf []uint32
		colBufs := make([][]scm.Scmer, len(conditionCols))
		acceptColBufs := make([][]scm.Scmer, len(acceptCols))
		t.iterateIndexOrdered(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf, usageWeight, limit, func(index *StorageIndex, active bool) {
			if len(sortcols) > 0 {
				resultAlreadySorted = indexCoversBoundaryOrder(index, active, boundaries, len(lower))
			}
		}, func(batch []uint32) bool {
			result.candidateCount += int64(len(batch))

			// pass 1: data-independent skip checks (recset/visibility/deletion),
			// producing the ordered surviving-id list without touching column data.
			survived := survivedBuf[:0]
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
				survived = append(survived, idx)
			}
			survivedBuf = survived

			outN := len(survived)
			if conditionAlwaysTrue {
				copy(batch, survived)
			} else {
				// pass 2: bulk-fetch every non-getter condition column for the
				// main-storage survivors of this batch, one call per column.
				mainIds := mainIdsBuf[:0]
				if conditionProgram.Kind != scm.SerialProcNativeArgConstant {
					for _, idx := range survived {
						if idx < t.main_count {
							mainIds = append(mainIds, idx)
						}
					}
				}
				mainIdsBuf = mainIds
				for i := range ccols {
					if conditionProgram.Kind == scm.SerialProcNativeArgConstant {
						continue
					}
					if conditionGetters[i] != nil {
						continue
					}
					if cap(colBufs[i]) < len(mainIds) {
						colBufs[i] = make([]scm.Scmer, len(mainIds))
					}
					colBufs[i] = colBufs[i][:len(mainIds)]
					if len(mainIds) == 0 {
						continue
					}
					if cNeedsCachedReader[i] {
						cReaders[i].GetValueMulti(mainIds, colBufs[i], 1)
					} else {
						ccols[i].GetValueMulti(mainIds, colBufs[i], 1)
					}
				}

				// Pass 3 dispatches once per batch. Simple binary predicates read
				// only their argument column and call the native primitive directly;
				// general expressions retain the existing interpreter adapter.
				if conditionProgram.Kind == scm.SerialProcNativeArgConstant {
					outN = t.filterNativeArgConstantScanBatch(survived, conditionCols, ccols, cReaders, conditionGetters, &conditionProgram)
					copy(batch, survived[:outN])
				} else {
					outN = 0
					mainBufIdx := 0
					for _, idx := range survived {
						if idx < t.main_count {
							for i := range ccols {
								if conditionGetters[i] != nil {
									cdataset[i] = conditionGetters[i](idx, 0)
								} else {
									cdataset[i] = colBufs[i][mainBufIdx]
								}
							}
							mainBufIdx++
						} else {
							for i, k := range conditionCols {
								if conditionGetters[i] != nil {
									cdataset[i] = conditionGetters[i](idx, 0)
								} else if cNeedsCachedReader[i] {
									cdataset[i] = cReaders[i].GetValue(idx)
								} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
									cdataset[i] = ccols[i].GetValue(idx)
								} else {
									cdataset[i] = t.getDelta(int(idx-t.main_count), k)
								}
							}
						}
						if !scm.ToBool(conditionProgram.Call(cdataset)) {
							continue
						}
						batch[outN] = idx
						outN++
					}
				}
			}

			// The accept predicate is a continuation supplied by scan_join_order.
			// Apply it only to rows that passed the ordinary condition, and fetch
			// its columns in batches so the ordered hot path remains columnar.
			if acceptProgram != nil && outN > 0 {
				acceptMainIds := acceptMainIdsBuf[:0]
				for _, idx := range batch[:outN] {
					if idx < t.main_count {
						acceptMainIds = append(acceptMainIds, idx)
					}
				}
				acceptMainIdsBuf = acceptMainIds
				for i := range acols {
					if cap(acceptColBufs[i]) < len(acceptMainIds) {
						acceptColBufs[i] = make([]scm.Scmer, len(acceptMainIds))
					}
					acceptColBufs[i] = acceptColBufs[i][:len(acceptMainIds)]
					if len(acceptMainIds) == 0 {
						continue
					}
					if aNeedsCachedReader[i] {
						aReaders[i].GetValueMulti(acceptMainIds, acceptColBufs[i], 1)
					} else {
						acols[i].GetValueMulti(acceptMainIds, acceptColBufs[i], 1)
					}
				}

				acceptedN := 0
				mainBufIdx := 0
				for _, idx := range batch[:outN] {
					if idx < t.main_count {
						for i := range acols {
							adataset[i] = acceptColBufs[i][mainBufIdx]
						}
						mainBufIdx++
					} else {
						for i, column := range acceptCols {
							if aNeedsCachedReader[i] {
								adataset[i] = aReaders[i].GetValue(idx)
							} else if _, isProxy := acols[i].(*StorageComputeProxy); isProxy {
								adataset[i] = acols[i].GetValue(idx)
							} else {
								adataset[i] = t.getDelta(int(idx-t.main_count), column)
							}
						}
					}
					if !scm.ToBool(acceptProgram.Call(adataset)) {
						continue
					}
					batch[acceptedN] = idx
					acceptedN++
				}
				outN = acceptedN
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
			if resultAlreadySorted && limit >= 0 && limitPartitionCols == 0 && resultN >= limit {
				return false
			}
			return true
		})
		result.items = result.items[:resultN]
	}()

	// and now sort result!
	result.sortdirs = adjustedSortdirs
	result.sortless = make([]func(scm.Scmer, scm.Scmer) bool, len(adjustedSortdirs))
	for i, relation := range adjustedSortdirs {
		result.sortless[i] = scm.OrderRelationLess(relation)
	}
	itemPos := make(map[uint32]int, len(result.items))
	for i, idx := range result.items {
		itemPos[idx] = i
	}
	// Computed sort callbacks may contain nested point scans. Evaluate each key
	// once per candidate instead of repeating the callback for every comparison.
	// The cache is local to this physical scan and is released with its queue.
	for c, sortcol := range sortcols {
		if sortcol.IsString() {
			continue
		}
		values := make([]scm.Scmer, len(result.items))
		for i, idx := range result.items {
			values[i] = result.scols[c](idx)
		}
		result.scols[c] = func(idx uint32) scm.Scmer {
			return values[itemPos[idx]]
		}
	}
	lessByID := func(a, b uint32) bool {
		cmpCount := len(result.scols)
		if len(result.sortless) < cmpCount {
			cmpCount = len(result.sortless)
		}
		for c := 0; c < cmpCount; c++ {
			av := result.scols[c](a)
			bv := result.scols[c](b)
			if result.sortless[c](av, bv) {
				return true
			}
			if result.sortless[c](bv, av) {
				return false
			}
		}
		return a < b
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
	if len(sortcols) > 0 && !resultAlreadySorted {
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
