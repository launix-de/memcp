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
import "time"
import "runtime/debug"
import "strings"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

type scanError struct {
	r     interface{}
	stack string
}

func (s scanError) Error() string {
	return fmt.Sprint(s.r)
}

func buildOuterNullCallbackRow(callbackCols []string) []scm.Scmer {
	return make([]scm.Scmer, len(callbackCols))
}

/* TODO: interface Scannable (scan + scan_order) and (table schema tbl) to get a scannable */

// optimizeScan is the Optimize hook for the scan declaration.
// It explicitly controls callback ownership for the reduce and reduce2 lambdas,
// ensuring the accumulator parameter is marked as owned (enabling _mut swaps
// like set_assoc → set_assoc_mut inside the reduce body).
func optimizeScanShared(v []scm.Scmer, oc *scm.OptimizerContext, mapEnd, reduceIdx, neutralIdx, reduce2Idx, outerIdx int) (scm.Scmer, *scm.TypeDescriptor) {
	// Non-callback args (tx, table, filtercols, mapcols, etc.) at loop depth 0
	for i := 1; i <= mapEnd && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	// Callback lambdas execute per-row → increment loop depth so the optimizer
	// does not inline hoisted defines (like table pointers) back into the loop.
	oc.Ome.IncrLoopDepth()
	if len(v) > reduceIdx && !v[reduceIdx].IsNil() {
		oc.SetCallbackOwned([]bool{true, false})
		v[reduceIdx], _ = oc.OptimizeSub(v[reduceIdx], true)
	}
	if len(v) > neutralIdx {
		v[neutralIdx], _ = oc.OptimizeSub(v[neutralIdx], true)
	}
	if len(v) > reduce2Idx && !v[reduce2Idx].IsNil() {
		oc.SetCallbackOwned([]bool{true, false})
		v[reduce2Idx], _ = oc.OptimizeSub(v[reduce2Idx], true)
	}
	if len(v) > outerIdx {
		v[outerIdx], _ = oc.OptimizeSub(v[outerIdx], true)
	}
	oc.Ome.DecrLoopDepth()
	return scm.NewSlice(v), nil
}

func optimizeScan(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	if rewritten := tryScanExistsRewrite(v); !rewritten.IsNil() {
		return oc.OptimizeSub(rewritten, useResult)
	}
	if rewritten := tryScanBatchRewrite(v); !rewritten.IsNil() {
		return oc.OptimizeSub(rewritten, useResult)
	}
	return optimizeScanShared(v, oc, 6, 7, 8, 9, 10)
}

func tryScanExistsRewrite(v []scm.Scmer) scm.Scmer {
	// scan: [fn, tx, table, filtercols, filterfn, mapcols, mapfn, reduce, neutral, reduce2, isOuter]
	if len(v) < 9 {
		return scm.NewNil()
	}
	if len(v) > 10 && scm.ToBool(v[10]) {
		return scm.NewNil()
	}
	if len(v) > 9 && !v[9].IsNil() {
		return scm.NewNil()
	}
	if !scanFalseNeutral(v[8]) || !scanExistsMap(v[6]) || !scanExistsOrReducer(v[7]) {
		return scm.NewNil()
	}
	if scanMapColsHaveSideEffects(v[5]) || scanExprMayHaveSideEffects(v[4]) {
		return scm.NewNil()
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan_exists"),
		v[1],
		v[2],
		v[3],
		v[4],
	})
}

func scanFalseNeutral(v scm.Scmer) bool {
	if v.IsBool() {
		return !v.Bool()
	}
	return false
}

func scanExistsMap(v scm.Scmer) bool {
	_, body, ok := scanLambdaParts(v)
	return ok && scanExprIsTrue(body)
}

func scanExistsOrReducer(v scm.Scmer) bool {
	params, body, ok := scanLambdaParts(v)
	if !ok || len(params) != 2 {
		return false
	}
	left, ok1 := scanSymbolName(params[0])
	right, ok2 := scanSymbolName(params[1])
	if !ok1 || !ok2 {
		return false
	}
	return scanExprIsOrOf(body, left, right)
}

func scanLambdaParts(v scm.Scmer) ([]scm.Scmer, scm.Scmer, bool) {
	if !v.IsSlice() {
		return nil, scm.NewNil(), false
	}
	items := v.Slice()
	if len(items) < 3 || !scanSymbolIs(items[0], "lambda") {
		return nil, scm.NewNil(), false
	}
	paramsExpr := items[1]
	if paramsExpr.IsNil() {
		return []scm.Scmer{}, items[2], true
	}
	if !paramsExpr.IsSlice() {
		return nil, scm.NewNil(), false
	}
	return paramsExpr.Slice(), items[2], true
}

func scanExprIsTrue(v scm.Scmer) bool {
	return v.IsBool() && v.Bool()
}

func scanExprIsOrOf(v scm.Scmer, left, right string) bool {
	if !v.IsSlice() {
		return false
	}
	items := v.Slice()
	if len(items) < 3 || !scanSymbolIs(items[0], "or") {
		return false
	}
	seenLeft := false
	seenRight := false
	for _, item := range items[1:] {
		if item.IsBool() && !item.Bool() {
			continue
		}
		if scanExprIsLambdaParam(item, left, 0) {
			seenLeft = true
			continue
		}
		if scanExprIsLambdaParam(item, right, 1) {
			seenRight = true
			continue
		}
		return false
	}
	return seenLeft && seenRight
}

func scanExprIsLambdaParam(v scm.Scmer, name string, idx int) bool {
	if v.IsNthLocalVar() {
		return int(v.NthLocalVar()) == idx
	}
	s, ok := scanSymbolName(v)
	return ok && s == name
}

func scanSymbolIs(v scm.Scmer, name string) bool {
	s, ok := scanSymbolName(v)
	return ok && s == name
}

func scanSymbolName(v scm.Scmer) (string, bool) {
	if v.GetTag() == scm.TagSymbol {
		return v.String(), true
	}
	if !v.IsSlice() {
		return "", false
	}
	items := v.Slice()
	if len(items) == 2 && items[0].GetTag() == scm.TagSymbol && items[0].String() == "quote" && items[1].GetTag() == scm.TagSymbol {
		return items[1].String(), true
	}
	return "", false
}

func scanMapColsHaveSideEffects(v scm.Scmer) bool {
	if v.IsNil() {
		return false
	}
	if !v.IsSlice() {
		return true
	}
	for _, item := range v.Slice() {
		if !item.IsString() {
			return true
		}
		col := item.String()
		if strings.HasPrefix(col, "$") {
			return true
		}
	}
	return false
}

func scanExprMayHaveSideEffects(v scm.Scmer) bool {
	if name, ok := scanSymbolName(v); ok {
		switch name {
		case "set", "define", "insert", "update", "delete", "createcolumn", "dropcolumn", "createtable", "droptable", "createkey", "dropkey", "resultrow", "print", "error", "$update":
			return true
		default:
			return false
		}
	}
	if !v.IsSlice() {
		return false
	}
	for _, item := range v.Slice() {
		if scanExprMayHaveSideEffects(item) {
			return true
		}
	}
	return false
}

func optimizeScanBatch(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	return optimizeScanShared(v, oc, 8, 9, 10, 11, 12)
}

// scanResult bundles per-shard outputs to minimize allocations and type assertions.
type scanResult struct {
	res        scm.Scmer
	outCount   int64
	inputCount int64
	err        scanError // err.r != nil indicates an error
}

func (t *table) scanExists(currentTx *TxContext, conditionCols []string, condition scm.Scmer) bool {
	ss := SessionStateFromTx(currentTx)
	if ss != nil && ss.IsKilled() {
		panic("query killed")
	}
	touchTempColumns(t, conditionCols, nil)
	boundaries := extractBoundaries(conditionCols, condition)
	reorderByFrequency(boundaries, t)
	lower, upperLast := indexFromBoundaries(boundaries)
	for _, b := range boundaries {
		t.AddPartitioningScore([]string{b.col})
	}

	values := make(chan scanResult, 4)
	var found atomic.Bool
	done := t.iterateShardsParallel(boundaries, func(s *storageShard, solo bool) {
		if found.Load() {
			values <- scanResult{}
			return
		}
		defer func() {
			if r := recover(); r != nil {
				values <- scanResult{err: scanError{r, string(debug.Stack())}}
			}
		}()
		if s.scanExists(boundaries, lower, upperLast, conditionCols, condition, currentTx, ss, &found) {
			found.Store(true)
			values <- scanResult{outCount: 1}
			return
		}
		values <- scanResult{}
	})
	if done == nil {
		close(values)
	} else {
		go func() {
			<-done
			close(values)
		}()
	}

	var scanErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if scanErr.r == nil {
				scanErr = msg.err
			}
			continue
		}
		if msg.outCount > 0 {
			found.Store(true)
		}
	}
	if scanErr.r != nil {
		panic(scanErr)
	}
	return found.Load()
}

// map reduce implementation based on scheme scripts
func (t *table) scan(currentTx *TxContext, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool) scm.Scmer {
	return t.scanWithBatch(currentTx, conditionCols, condition, callbackCols, callback, aggregate, neutral, aggregate2, isOuter, 0, nil)
}

func (t *table) scanWithBatch(currentTx *TxContext, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool, stride int, batchdata []scm.Scmer) scm.Scmer {
	ss := SessionStateFromTx(currentTx)
	if ss != nil && ss.IsKilled() {
		panic("query killed")
	}
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}
	if hasMutationCallback && !t.hasMutationOwner() {
		t.mutationMu.Lock()
		defer t.mutationMu.Unlock()
		t.enterMutationOwner()
		defer t.exitMutationOwner()
	}
	if t.tableLockOwner.Load() != nil {
		t.waitTableLock(ss, hasMutationCallback)
	}
	// touch temp columns so CacheManager knows they're still in use
	touchTempColumns(t, conditionCols, callbackCols)
	// Measure analysis time (boundary extraction, sharding hints)
	analyzeStart := time.Now()
	/* analyze query */
	boundaries := extractBoundaries(conditionCols, condition)
	reorderByFrequency(boundaries, t)
	lower, upperLast := indexFromBoundaries(boundaries)
	if Settings.ScanDebugging {
		dbg := fmt.Sprintf("[SCAN] %s.%s", t.schema.Name, t.Name)
		for _, b := range boundaries {
			dbg += fmt.Sprintf(" %s:[%v..%v]", b.col, b.lower, b.upper)
		}
		dbg += fmt.Sprintf(" lower=%v upper=%v", lower, upperLast)
		fmt.Println(dbg)
	}
	// give sharding hints
	for _, b := range boundaries {
		t.AddPartitioningScore([]string{b.col})
	}

	analyzeNs := time.Since(analyzeStart).Nanoseconds()
	// Measure execution time (parallel shard scans + collection)
	execStart := time.Now()
	var outCount int64
	var inputCount int64
	values := make(chan scanResult, 4)
	done := t.iterateShardsParallel(boundaries, func(s *storageShard, solo bool) {
		// Kill check at shard-scheduling point: ss is a closure variable, no GLS lookup needed.
		// This keeps the worker pool draining quickly on tables with many shards.
		if ss != nil && ss.IsKilled() {
			panic("query killed")
		}
		defer func() {
			if r := recover(); r != nil {
				values <- scanResult{err: scanError{r, string(debug.Stack())}}
			}
		}()
		res, cnt := s.scan(boundaries, lower, upperLast, conditionCols, condition, callbackCols, callback, aggregate, neutral, stride, batchdata, currentTx, ss)
		values <- scanResult{res: res, outCount: cnt, inputCount: int64(s.Count())}
	})
	if done == nil {
		close(values)
	} else {
		go func() {
			<-done
			close(values)
		}()
	}

	akkumulator := neutral
	hadValue := false
	var scanErr scanError
	if !aggregate2.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(aggregate2)
		for msg := range values {
			if msg.err.r != nil {
				if scanErr.r == nil {
					scanErr = msg.err
				}
				continue
			}
			if scanErr.r != nil {
				continue
			}
			inputCount += msg.inputCount
			outCount += msg.outCount
			if msg.outCount > 0 {
				akkumulator = fn(akkumulator, msg.res)
				hadValue = true
			}
		}
		if scanErr.r == nil && !hadValue && isOuter {
			nullRow := buildOuterNullCallbackRow(callbackCols)
			akkumulator = fn(akkumulator, scm.Apply(callback, nullRow...)) // outer join: push one NULL row
		}
	} else if !aggregate.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(aggregate)
		for msg := range values {
			if msg.err.r != nil {
				if scanErr.r == nil {
					scanErr = msg.err
				}
				continue
			}
			if scanErr.r != nil {
				continue
			}
			inputCount += msg.inputCount
			outCount += msg.outCount
			if msg.outCount > 0 {
				akkumulator = fn(akkumulator, msg.res)
				hadValue = true
			}
		}
		if scanErr.r == nil && !hadValue && isOuter {
			nullRow := buildOuterNullCallbackRow(callbackCols)
			akkumulator = fn(akkumulator, scm.Apply(callback, nullRow...)) // outer join: push one NULL row
		}
	} else {
		for msg := range values {
			if msg.err.r != nil {
				if scanErr.r == nil {
					scanErr = msg.err
				}
				continue
			}
			if scanErr.r != nil {
				continue
			}
			inputCount += msg.inputCount
			outCount += msg.outCount
			hadValue = hadValue || msg.outCount > 0
		}
		if scanErr.r == nil && !hadValue && isOuter {
			nullRow := buildOuterNullCallbackRow(callbackCols)
			scm.Apply(callback, nullRow...) // outer join: push one NULL row
		}
	}
	if scanErr.r != nil {
		panic(scanErr)
	}
	// log statistics (best-effort, async so it doesn't add latency)
	execNs := time.Since(execStart).Nanoseconds()
	if Settings.ScanDebugging || inputCount > int64(Settings.AnalyzeMinItems) {
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
			indexColsEnc := boundaryIndexCols(boundaries)
			safeLogScan(t.schema.Name, t.Name, false, filterEnc, "", indexColsEnc, inputCount, outCount, anNs, exNs)
		}(analyzeNs, execNs)
	}
	return akkumulator
}

func (t *storageShard) scanExists(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState, stop *atomic.Bool) bool {
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}
	conditionFn := scm.OptimizeProcToSerialFunction(condition)

	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwner()
	t.ensureMainCount(skipShardReadLock)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	cNeedsTxReader := make([]bool, len(conditionCols))
	for i, k := range conditionCols {
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
		if proxy, ok := ccols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			cNeedsTxReader[i] = true
		}
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.tableLockOwner.Load() != nil {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, false)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()

	acidMode := currentTx != nil && currentTx.Mode == TxACID
	mainCount := t.main_count
	maxInsertIndex := len(t.inserts)
	visibleUpper := mainCount + uint32(maxInsertIndex)
	found := false

	var buf [8]uint32
	t.iterateIndex(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf[:], true, func(batch []uint32) bool {
		if stop != nil && stop.Load() {
			return false
		}
		if ss != nil && ss.IsKilled() {
			panic("query killed")
		}
		for _, idx := range batch {
			if idx >= visibleUpper {
				continue
			}
			if acidMode {
				if !currentTx.IsVisible(t, idx) {
					continue
				}
			} else if t.deletions.Get(uint(idx)) {
				continue
			}
			if idx < mainCount {
				for i, c := range cReaders {
					if cNeedsTxReader[i] {
						cdataset[i] = c.GetValue(idx)
					} else {
						cdataset[i] = ccols[i].GetValue(idx)
					}
				}
			} else {
				for i, col := range conditionCols {
					if cNeedsTxReader[i] {
						cdataset[i] = cReaders[i].GetValue(idx)
					} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
						cdataset[i] = ccols[i].GetValue(idx)
					} else {
						cdataset[i] = t.getDelta(int(idx-mainCount), col)
					}
				}
			}
			if scm.ToBool(conditionFn(cdataset...)) {
				found = true
				return false
			}
		}
		return true
	})

	return found
}

func (t *storageShard) scan(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64) {
	if stride > 0 {
		return t.scanBatch(boundaries, lower, upperLast, conditionCols, condition, callbackCols, callback, aggregate, neutral, stride, batchdata, currentTx, ss)
	}
	akkumulator := neutral
	var outCount int64
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}

	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}

	// Ensure shard is loaded from disk before accessing columns.
	// ensureLoaded() must run before getColumnStorageOrPanic so that COLD
	// shards have their column map populated by load(t) first.
	// ensureMainCount then loads at least one column to initialize main_count.
	t.ensureLoaded()
	ownsWrite := t.hasWriteOwner()
	lockMutationExclusively := hasMutationCallback && !ownsWrite
	writeLocked := false
	if lockMutationExclusively {
		t.mu.Lock()
		writeLocked = true
		defer func() {
			if writeLocked {
				t.mu.Unlock()
			}
		}()
		t.enterWriteOwner()
		defer func() {
			if writeLocked {
				t.exitWriteOwner()
			}
		}()
		if currentTx != nil {
			currentTx.EnterShardWrite(t)
			defer currentTx.ExitShardWrite(t)
		}
		// Table lock check for mutation path: lockTable() stores owner while holding
		// all shard write locks, so checking after our own t.mu.Lock() is TOCTOU-safe.
		// waitTableLock only uses tableLockMu (not t.mu), so no deadlock.
		if t.t.tableLockOwner.Load() != nil {
			t.t.waitTableLock(ss, true)
		}
	}
	skipShardReadLock := ownsWrite || lockMutationExclusively
	t.ensureMainCount(skipShardReadLock)

	// condition column readers
	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	cNeedsTxReader := make([]bool, len(conditionCols))
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	for i, k := range conditionCols {
		if k == "$recset_contains" {
			fnptr := recSetContainsClosure(t)
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
	cdataset := make([]scm.Scmer, len(conditionCols))

	// MapReducer for map+reduce phase (builds column readers internally)
	mapper := t.OpenMapReducer(callbackCols, callback, aggregate, skipShardReadLock, 0, nil, currentTx)
	defer mapper.Close()
	// Use a guarded lock that will always be released on panic to avoid leaked locks.
	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		// Table lock check must happen AFTER shard RLock to close the TOCTOU window:
		// lockTable() sets tableLockOwner while holding shard write locks, so any
		// scan that gets past RLock is guaranteed to see a non-nil owner if a
		// LOCK TABLES was issued before this scan acquired the shard read lock.
		if t.t.tableLockOwner.Load() != nil {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, hasMutationCallback)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()
	maxInsertIndex := len(t.inserts)
	visibleUpper := t.main_count + uint32(maxInsertIndex)
	var pendingRecids []uint32
	var mutationSeen map[uint32]struct{}
	if hasMutationCallback {
		mutationSeen = make(map[uint32]struct{}, 128)
	}

	// filter phase: iterateIndex fills stack buffer, callback filters in-place and flushes to MapReducer
	var buf [1024]uint32
	hadValue := false

	t.iterateIndex(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf[:], true, func(batch []uint32) bool {
		// filter in-place: overwrite batch with passing IDs
		outN := 0
		for _, idx := range batch {
			effectiveIdx := idx
			if effectiveIdx >= visibleUpper {
				continue
			}
			if hasMutationCallback && (currentTx == nil || currentTx.Mode != TxACID) {
				if t.deletions.Get(uint(effectiveIdx)) {
					if followIdx, ok := t.resolveVisiblePrimaryRecidLocked(effectiveIdx); ok {
						effectiveIdx = followIdx
					} else {
						continue
					}
				}
				// Multiple stale index entries can resolve to the same current row.
				// Mutate each current row at most once per statement.
				if _, ok := mutationSeen[effectiveIdx]; ok {
					continue
				}
				mutationSeen[effectiveIdx] = struct{}{}
			}
			if currentTx != nil && currentTx.Mode == TxACID {
				if !currentTx.IsVisible(t, effectiveIdx) {
					continue
				}
			} else if t.deletions.Get(uint(effectiveIdx)) {
				continue // item is on delete list
			}

			// condition check
			if effectiveIdx < t.main_count {
				for i, k := range cReaders {
					if getter := conditionGetters[i]; getter != nil {
						cdataset[i] = getter(effectiveIdx, 0)
					} else {
						cdataset[i] = k.GetValue(effectiveIdx)
					}
				}
			} else {
				for i, k := range conditionCols {
					if getter := conditionGetters[i]; getter != nil {
						cdataset[i] = getter(effectiveIdx, 0)
					} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
						cdataset[i] = cReaders[i].GetValue(effectiveIdx)
					} else {
						cdataset[i] = t.getDelta(int(effectiveIdx-t.main_count), k)
					}
				}
			}
			var condResult bool
			var condVal scm.Scmer
			condVal = conditionFn(cdataset...)
			condResult = scm.ToBool(condVal)
			if !condResult {
				continue
			}

			batch[outN] = effectiveIdx
			outN++
		}
		if outN > 0 {
			if hasMutationCallback {
				pendingRecids = append(pendingRecids, batch[:outN]...)
				outCount += int64(outN)
				hadValue = true
			} else {
				// release lock for map+reduce (UpdateFunction needs write lock)
				if locked {
					t.mu.RUnlock()
					locked = false
				}
				outCount += int64(outN)
				akkumulator = mapper.Stream(akkumulator, batch[:outN], nil)
				hadValue = true
				if !skipShardReadLock {
					t.mu.RLock()
					locked = true
				}
			}
		}
		return true
	})

	// finished reading
	if locked {
		t.mu.RUnlock()
		locked = false
	}
	if !hadValue {
		// Release locks before flushing trigger batch
		if locked {
			t.mu.RUnlock()
			locked = false
		}
		mapper.FlushSideEffects()
		return scm.NewNil(), outCount
	}
	if hasMutationCallback && len(pendingRecids) > 0 {
		// Release exclusive lock before map+reduce phase: mapFn may contain
		// nested scans on the same table (e.g. EXISTS inside UPDATE).
		// The mapper re-acquires mu.Lock() per batch internally via
		// processMainBlock/processDeltaBlock when shardWriteLocked=false.
		// Table-level mutationMu still serializes concurrent mutations.
		if writeLocked {
			t.exitWriteOwner()
			t.mu.Unlock()
			writeLocked = false
			mapper.SetShardWriteLocked(false)
		}
		for i := 0; i < len(pendingRecids); i += len(buf) {
			j := i + len(buf)
			if j > len(pendingRecids) {
				j = len(pendingRecids)
			}
			akkumulator = mapper.Stream(akkumulator, pendingRecids[i:j], nil)
		}
	}
	// Release locks before flushing trigger batch to avoid deadlocks
	// (trigger handlers may scan other tables that need locks)
	if locked {
		t.mu.RUnlock()
		locked = false
	}
	if writeLocked {
		t.exitWriteOwner()
		t.mu.Unlock()
		writeLocked = false
	}
	mapper.FlushSideEffects()
	return akkumulator, outCount
}

func (t *storageShard) scanBatch(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64) {
	akkumulator := neutral
	var outCount int64
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}

	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}

	t.ensureLoaded()
	ownsWrite := t.hasWriteOwner()
	lockMutationExclusively := hasMutationCallback && !ownsWrite
	writeLocked := false
	if lockMutationExclusively {
		t.mu.Lock()
		writeLocked = true
		defer func() {
			if writeLocked {
				t.mu.Unlock()
			}
		}()
		t.enterWriteOwner()
		defer func() {
			if writeLocked {
				t.exitWriteOwner()
			}
		}()
		if currentTx != nil {
			currentTx.EnterShardWrite(t)
			defer currentTx.ExitShardWrite(t)
		}
		if t.t.tableLockOwner.Load() != nil {
			t.t.waitTableLock(ss, true)
		}
	}
	skipShardReadLock := ownsWrite || lockMutationExclusively
	t.ensureMainCount(skipShardReadLock)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	cNeedsTxReader := make([]bool, len(conditionCols))
	conditionBatchSubidx := make([]int, len(conditionCols))
	for i, k := range conditionCols {
		if subidx, ok := parseBatchPseudoColName(k); ok {
			conditionBatchSubidx[i] = subidx + 1
			continue
		}
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
		if proxy, ok := ccols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			cNeedsTxReader[i] = true
		}
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	mapper := t.OpenMapReducer(callbackCols, callback, aggregate, skipShardReadLock, stride, batchdata, currentTx)
	defer mapper.Close()

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.tableLockOwner.Load() != nil {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, hasMutationCallback)
			t.mu.RLock()
			locked = true
		}
	}
	defer func() {
		if locked {
			t.mu.RUnlock()
		}
	}()
	maxInsertIndex := len(t.inserts)
	visibleUpper := t.main_count + uint32(maxInsertIndex)
	var pendingRecids []uint32
	var pendingBatchids []uint32
	var mutationSeen map[uint64]struct{}
	if hasMutationCallback {
		mutationSeen = make(map[uint64]struct{}, 128)
	}

	var buf [1024]uint32
	var batchBuf [1024]uint32
	hadValue := false
	batchCount := len(batchdata) / stride
	batchBoundaries := hasBatchBoundaries(boundaries)

	for batchid := 0; batchid < batchCount; batchid++ {
		if ss != nil && ss.IsKilled() {
			panic("query killed")
		}
		activeBoundaries := boundaries
		activeLower := lower
		activeUpperLast := upperLast
		if batchBoundaries {
			activeBoundaries = materializeBatchBoundaries(boundaries, stride, batchdata, uint32(batchid))
			activeLower, activeUpperLast = indexFromBoundaries(activeBoundaries)
		}

		t.iterateIndex(currentTx, activeBoundaries, activeLower, activeUpperLast, maxInsertIndex, buf[:], true, func(batch []uint32) bool {
			if ss != nil && ss.IsKilled() {
				panic("query killed")
			}
			outN := 0
			for _, idx := range batch {
				effectiveIdx := idx
				if effectiveIdx >= visibleUpper {
					continue
				}
				if hasMutationCallback && (currentTx == nil || currentTx.Mode != TxACID) {
					if t.deletions.Get(uint(effectiveIdx)) {
						if followIdx, ok := t.resolveVisiblePrimaryRecidLocked(effectiveIdx); ok {
							effectiveIdx = followIdx
						} else {
							continue
						}
					}
					key := (uint64(uint32(batchid)) << 32) | uint64(effectiveIdx)
					if _, ok := mutationSeen[key]; ok {
						continue
					}
					mutationSeen[key] = struct{}{}
				}
				if currentTx != nil && currentTx.Mode == TxACID {
					if !currentTx.IsVisible(t, effectiveIdx) {
						continue
					}
				} else if t.deletions.Get(uint(effectiveIdx)) {
					continue
				}

				if effectiveIdx < t.main_count {
					for i, k := range ccols {
						if subidx := conditionBatchSubidx[i] - 1; subidx >= 0 {
							cdataset[i] = batchdata[batchid*stride+subidx]
						} else if cNeedsTxReader[i] {
							cdataset[i] = cReaders[i].GetValue(effectiveIdx)
						} else {
							cdataset[i] = k.GetValue(effectiveIdx)
						}
					}
				} else {
					for i, k := range conditionCols {
						if subidx := conditionBatchSubidx[i] - 1; subidx >= 0 {
							cdataset[i] = batchdata[batchid*stride+subidx]
						} else if cNeedsTxReader[i] {
							cdataset[i] = cReaders[i].GetValue(effectiveIdx)
						} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
							cdataset[i] = ccols[i].GetValue(effectiveIdx)
						} else {
							cdataset[i] = t.getDelta(int(effectiveIdx-t.main_count), k)
						}
					}
				}
				if !scm.ToBool(conditionFn(cdataset...)) {
					continue
				}

				batch[outN] = effectiveIdx
				batchBuf[outN] = uint32(batchid)
				outN++
			}
			if outN > 0 {
				if hasMutationCallback {
					pendingRecids = append(pendingRecids, batch[:outN]...)
					pendingBatchids = append(pendingBatchids, batchBuf[:outN]...)
					outCount += int64(outN)
					hadValue = true
				} else {
					if locked {
						t.mu.RUnlock()
						locked = false
					}
					outCount += int64(outN)
					akkumulator = mapper.Stream(akkumulator, batch[:outN], batchBuf[:outN])
					hadValue = true
					if !skipShardReadLock {
						t.mu.RLock()
						locked = true
					}
				}
			}
			return true
		})
	}

	if locked {
		t.mu.RUnlock()
		locked = false
	}
	if !hadValue {
		mapper.FlushSideEffects()
		return scm.NewNil(), outCount
	}
	if hasMutationCallback && len(pendingRecids) > 0 {
		if writeLocked {
			t.exitWriteOwner()
			t.mu.Unlock()
			writeLocked = false
			mapper.SetShardWriteLocked(false)
		}
		for i := 0; i < len(pendingRecids); i += len(buf) {
			j := i + len(buf)
			if j > len(pendingRecids) {
				j = len(pendingRecids)
			}
			akkumulator = mapper.Stream(akkumulator, pendingRecids[i:j], pendingBatchids[i:j])
		}
	}
	if locked {
		t.mu.RUnlock()
		locked = false
	}
	if writeLocked {
		t.exitWriteOwner()
		t.mu.Unlock()
		writeLocked = false
	}
	mapper.FlushSideEffects()
	return akkumulator, outCount
}
