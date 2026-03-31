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
import "sync"
import "time"
import "runtime/debug"
import "github.com/launix-de/memcp/scm"

type scanError struct {
	r     interface{}
	stack string
}

func (s scanError) Error() string {
	return fmt.Sprint(s.r) + "\n" + s.stack // room for improvement
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
	for i := 1; i <= mapEnd && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	if len(v) > reduceIdx && !v[reduceIdx].IsNil() {
		oc.SetCallbackOwned([]bool{true, false}) // acc is owned
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
	return scm.NewSlice(v), nil
}

func optimizeScan(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	return optimizeScanShared(v, oc, 6, 7, 8, 9, 10)
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

// map reduce implementation based on scheme scripts
func (t *table) scan(conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool) scm.Scmer {
	return t.scanWithBatch(conditionCols, condition, callbackCols, callback, aggregate, neutral, aggregate2, isOuter, 0, nil)
}

func (t *table) scanWithBatch(conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool, stride int, batchdata []scm.Scmer) scm.Scmer {
	ss := scm.GetCurrentSessionState()
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
	results := make([]scanResult, 0, 8)
	var resultsMu sync.Mutex
	t.iterateShardsParallel(boundaries, func(s *storageShard, solo bool) {
		// Kill check at shard-scheduling point: ss is a closure variable, no GLS lookup needed.
		// This keeps the worker pool draining quickly on tables with many shards.
		if ss != nil && ss.IsKilled() {
			panic("query killed")
		}
		msg := scanResult{}
		defer func() {
			if r := recover(); r != nil {
				msg = scanResult{err: scanError{r, string(debug.Stack())}}
			}
			if solo {
				results = append(results, msg)
				return
			}
			resultsMu.Lock()
			results = append(results, msg)
			resultsMu.Unlock()
		}()
		res, cnt := s.scan(boundaries, lower, upperLast, conditionCols, condition, callbackCols, callback, aggregate, neutral, stride, batchdata)
		msg = scanResult{res: res, outCount: cnt, inputCount: int64(s.Count())}
	})

	akkumulator := neutral
	hadValue := false
	var scanErr scanError
	if !aggregate2.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(aggregate2)
		for _, msg := range results {
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
		for _, msg := range results {
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
		for _, msg := range results {
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

func (t *storageShard) scan(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer) (scm.Scmer, int64) {
	if stride > 0 {
		return t.scanBatch(boundaries, lower, upperLast, conditionCols, condition, callbackCols, callback, aggregate, neutral, stride, batchdata)
	}
	akkumulator := neutral
	var outCount int64

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
	currentTx := CurrentTx()
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
			t.t.waitTableLock(scm.GetCurrentSessionState(), true)
		}
	}
	skipShardReadLock := ownsWrite || lockMutationExclusively
	t.ensureMainCount(skipShardReadLock)

	// condition column readers
	ccols := make([]ColumnStorage, len(conditionCols))
	for i, k := range conditionCols {
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	// MapReducer for map+reduce phase (builds column readers internally)
	mapper := t.openMapReducerEx(callbackCols, callback, aggregate, skipShardReadLock, 0, nil)
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
			t.t.waitTableLock(scm.GetCurrentSessionState(), hasMutationCallback)
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

	t.iterateIndex(boundaries, lower, upperLast, maxInsertIndex, buf[:], func(batch []uint32) bool {
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
				for i, k := range ccols {
					cdataset[i] = k.GetValue(effectiveIdx)
				}
			} else {
				for i, k := range conditionCols {
					if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
						cdataset[i] = ccols[i].GetValue(effectiveIdx)
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

func (t *storageShard) scanBatch(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer) (scm.Scmer, int64) {
	akkumulator := neutral
	var outCount int64
	ss := scm.GetCurrentSessionState()

	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}

	t.ensureLoaded()
	currentTx := CurrentTx()
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
			t.t.waitTableLock(scm.GetCurrentSessionState(), true)
		}
	}
	skipShardReadLock := ownsWrite || lockMutationExclusively
	t.ensureMainCount(skipShardReadLock)

	ccols := make([]ColumnStorage, len(conditionCols))
	conditionBatchSubidx := make([]int, len(conditionCols))
	for i, k := range conditionCols {
		if subidx, ok := parseBatchPseudoColName(k); ok {
			conditionBatchSubidx[i] = subidx + 1
			continue
		}
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))

	mapper := t.openMapReducerEx(callbackCols, callback, aggregate, skipShardReadLock, stride, batchdata)
	defer mapper.Close()

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.tableLockOwner.Load() != nil {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(scm.GetCurrentSessionState(), hasMutationCallback)
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

		t.iterateIndex(activeBoundaries, activeLower, activeUpperLast, maxInsertIndex, buf[:], func(batch []uint32) bool {
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
						} else {
							cdataset[i] = k.GetValue(effectiveIdx)
						}
					}
				} else {
					for i, k := range conditionCols {
						if subidx := conditionBatchSubidx[i] - 1; subidx >= 0 {
							cdataset[i] = batchdata[batchid*stride+subidx]
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
