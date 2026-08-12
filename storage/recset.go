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
import "runtime/debug"
import "sort"
import "sync/atomic"
import "unsafe"
import "github.com/launix-de/memcp/scm"

// TagRecSet is the custom Scmer tag for query-local RecSet handles.
const TagRecSet = 102

type recSetShard struct {
	shard  *storageShard
	recids []uint32
	count  int64
}

func (s *recSetShard) contains(recid uint32) bool {
	if s == nil {
		return false
	}
	pos := sort.Search(len(s.recids), func(i int) bool { return s.recids[i] >= recid })
	return pos < len(s.recids) && s.recids[pos] == recid
}

// recSet is a query-local, non-persistent subset of one table represented by
// physical record IDs. It is intentionally table-shaped so the planner can swap
// repeated scans over the same filtered base relation for a scan over this value.
type recSet struct {
	tx     *TxContext
	table  *table
	shards []recSetShard
	count  int64
}

func NewRecSetScmer(rs *recSet) scm.Scmer {
	return scm.NewCustom(TagRecSet, unsafe.Pointer(rs))
}

func RecSetFromScmer(s scm.Scmer) *recSet {
	return (*recSet)(s.Custom(TagRecSet))
}

func (r *recSet) String() string {
	if r == nil || r.table == nil {
		return "(recset nil)"
	}
	return fmt.Sprintf("(recset %q %q %d)", r.table.schema.Name, r.table.Name, r.count)
}

func (r *recSet) shardEntry(shard *storageShard) *recSetShard {
	if r == nil || shard == nil || r.table != shard.t {
		return nil
	}
	for i := range r.shards {
		rs := &r.shards[i]
		if rs.shard == shard {
			return rs
		}
	}
	return nil
}

func (r *recSet) contains(shard *storageShard, recid uint32) bool {
	return r.shardEntry(shard).contains(recid)
}

func recSetUnion(items []*recSet) *recSet {
	var base *table
	var tx *TxContext
	for _, rs := range items {
		if rs == nil || rs.table == nil {
			continue
		}
		if base == nil {
			base = rs.table
			tx = rs.tx
		} else if base != rs.table {
			panic("recset_union: all recsets must belong to the same table")
		}
	}
	result := &recSet{tx: tx, table: base}
	if base == nil {
		return result
	}

	byShard := make(map[*storageShard][]uint32)
	for _, rs := range items {
		if rs == nil || rs.table == nil {
			continue
		}
		for i := range rs.shards {
			part := rs.shards[i]
			if part.count == 0 {
				continue
			}
			byShard[part.shard] = append(byShard[part.shard], part.recids...)
		}
	}
	for shard, recids := range byShard {
		sort.Slice(recids, func(i, j int) bool { return recids[i] < recids[j] })
		write := 0
		var last uint32
		for i, recid := range recids {
			if i > 0 && recid == last {
				continue
			}
			recids[write] = recid
			write++
			last = recid
		}
		recids = recids[:write]
		result.count += int64(len(recids))
		result.shards = append(result.shards, recSetShard{shard: shard, recids: recids, count: int64(len(recids))})
	}
	sort.Slice(result.shards, func(i, j int) bool {
		return result.shards[i].shard.uuid.String() < result.shards[j].shard.uuid.String()
	})
	return result
}

func recSetContainsClosure(shard *storageShard) *func(uint32, ...scm.Scmer) scm.Scmer {
	var cachedRecSet *recSet
	var cachedShard *recSetShard
	fn := func(recid uint32, args ...scm.Scmer) scm.Scmer {
		if len(args) != 1 || !args[0].IsCustom(TagRecSet) {
			return scm.NewBool(false)
		}
		rs := RecSetFromScmer(args[0])
		if rs != cachedRecSet {
			cachedRecSet = rs
			cachedShard = rs.shardEntry(shard)
		}
		return scm.NewBool(cachedShard.contains(recid))
	}
	return &fn
}

func recSetAlreadyMatchedClosure() *func(uint32, ...scm.Scmer) scm.Scmer {
	fn := func(_ uint32, _ ...scm.Scmer) scm.Scmer {
		return scm.NewBool(true)
	}
	return &fn
}

type recSetBuildResult struct {
	part recSetShard
	err  scanError
}

type recSetKeyResult struct {
	keys [][]scm.Scmer
	err  scanError
}

func (t *table) recSetShardResultBufferSize() int {
	t.shardModeMu.RLock()
	defer t.shardModeMu.RUnlock()
	size := len(t.PShards)
	if t.ShardMode == ShardModeFree {
		size = len(t.Shards)
	}
	if size < 1 {
		return 1
	}
	return size
}

func (t *table) scanRecSet(currentTx *TxContext, conditionCols []string, condition scm.Scmer) *recSet {
	ss := SessionStateFromTx(currentTx)
	querySeq := scm.CurrentQuerySeq()
	boundaries := extractBoundaries(conditionCols, condition)
	boundaries, recsetFilter := splitRecSetBoundary(boundaries, t)
	reorderByFrequency(boundaries, t)
	lower, upperLast := indexFromBoundaries(boundaries)
	result := &recSet{tx: currentTx, table: t}

	values := make(chan recSetBuildResult, t.recSetShardResultBufferSize())
	done := t.iterateShardsParallel(currentTx, boundaries, func(shard *storageShard, solo bool) {
		withTxSession(currentTx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- recSetBuildResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			// Cancellation contract: check only at the scheduling boundary, before entering
			// the shard. Once entered, a shard runs atomically without cancellation checks.
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			values <- recSetBuildResult{
				part: shard.collectRecSet(boundaries, lower, upperLast, conditionCols, condition, currentTx, ss, recsetFilter),
			}
			return scm.NewNil()
		})
	})
	if done == nil {
		close(values)
	} else {
		go func() {
			<-done
			close(values)
		}()
	}

	var buildErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if buildErr.r == nil {
				buildErr = msg.err
			}
			continue
		}
		if buildErr.r != nil {
			continue
		}
		result.count += msg.part.count
		result.shards = append(result.shards, msg.part)
	}
	if buildErr.r != nil {
		panic(buildErr)
	}
	return result
}

func (t *storageShard) collectRecSet(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState, recsetFilter *recSet) recSetShard {
	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
	var recsetPart *recSetShard
	if recsetFilter != nil {
		recsetPart = recsetFilter.shardEntry(t)
		if recsetPart == nil || recsetPart.count == 0 {
			return recSetShard{shard: t}
		}
	}
	recsetBoundaryCoversCondition := recsetPart != nil && recSetBoundaryCallCount(conditionCols, condition) == 1

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
	cNeedsCachedReader := make([]bool, len(conditionCols))
	conditionGetters := make([]mapArgGetter, len(conditionCols))
	for i, k := range conditionCols {
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

	maxInsertIndex := len(t.inserts)
	visibleUpper := t.main_count + uint32(maxInsertIndex)
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	recids := make([]uint32, 0, 64)
	var buf [1024]uint32
	t.iterateIndex(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf[:], true, func(batch []uint32) bool {
		for _, idx := range batch {
			if recsetPart != nil && !recsetPart.contains(idx) {
				continue
			}
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
			if idx < t.main_count {
				for i, c := range cReaders {
					if getter := conditionGetters[i]; getter != nil {
						cdataset[i] = getter(idx, 0)
					} else if cNeedsCachedReader[i] {
						cdataset[i] = c.GetValue(idx)
					} else {
						cdataset[i] = ccols[i].GetValue(idx)
					}
				}
			} else {
				for i, col := range conditionCols {
					if getter := conditionGetters[i]; getter != nil {
						cdataset[i] = getter(idx, 0)
					} else if cNeedsCachedReader[i] {
						cdataset[i] = cReaders[i].GetValue(idx)
					} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
						cdataset[i] = ccols[i].GetValue(idx)
					} else {
						cdataset[i] = t.getDelta(int(idx-t.main_count), col)
					}
				}
			}
			if scm.ToBool(conditionFn(cdataset...)) {
				recids = append(recids, idx)
			}
		}
		return true
	})
	sort.Slice(recids, func(i, j int) bool { return recids[i] < recids[j] })
	return recSetShard{shard: t, recids: recids, count: int64(len(recids))}
}

func (r *recSet) projectJoin(currentTx *TxContext, sourceKeyCols []string, target *table, targetKeyCols []string) *recSet {
	if r == nil || r.table == nil {
		return &recSet{tx: currentTx, table: target}
	}
	if len(sourceKeyCols) == 0 || len(sourceKeyCols) != len(targetKeyCols) {
		panic("recset_project_join: source and target key columns must have the same non-zero length")
	}
	if currentTx == nil {
		currentTx = r.tx
	}
	ss := SessionStateFromTx(currentTx)
	keys := r.collectProjectJoinKeys(currentTx, sourceKeyCols, ss)
	return target.projectJoinKeysToRecSet(currentTx, targetKeyCols, keys, ss)
}

func (r *recSet) collectProjectJoinKeys(currentTx *TxContext, sourceKeyCols []string, ss *scm.SessionState) [][]scm.Scmer {
	querySeq := scm.CurrentQuerySeq()
	values := make(chan recSetKeyResult, len(r.shards))
	closeAfter := 0
	for i := range r.shards {
		part := r.shards[i]
		if part.count == 0 {
			continue
		}
		closeAfter++
		go func(part recSetShard) {
			withTxSession(currentTx, func() scm.Scmer {
				defer func() {
					if rec := recover(); rec != nil {
						values <- recSetKeyResult{err: scanError{rec, string(debug.Stack())}}
					}
				}()
				// Cancellation contract: check only at the scheduling boundary, before entering
				// the shard. Once entered, a shard runs atomically without cancellation checks.
				if ss != nil && ss.IsKilledSeq(querySeq) {
					panic("query killed")
				}
				values <- recSetKeyResult{keys: part.shard.collectProjectJoinKeys(part.recids, sourceKeyCols, currentTx, ss)}
				return scm.NewNil()
			})
		}(part)
	}
	keys := make([][]scm.Scmer, 0, r.count)
	var keyErr scanError
	for i := 0; i < closeAfter; i++ {
		msg := <-values
		if msg.err.r != nil {
			if keyErr.r == nil {
				keyErr = msg.err
			}
			continue
		}
		keys = append(keys, msg.keys...)
	}
	if keyErr.r != nil {
		panic(keyErr)
	}
	return keys
}

func (t *storageShard) collectProjectJoinKeys(recids []uint32, sourceKeyCols []string, currentTx *TxContext, ss *scm.SessionState) [][]scm.Scmer {
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)

	cols := make([]ColumnStorage, len(sourceKeyCols))
	readers := make([]ColumnReader, len(sourceKeyCols))
	needsTxReader := make([]bool, len(sourceKeyCols))
	for i, col := range sourceKeyCols {
		cols[i] = t.getColumnStorageOrPanicEx(col, skipShardReadLock, currentTx)
		readers[i] = newCachedColumnReaderTx(cols[i], currentTx)
		if proxy, ok := cols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			needsTxReader[i] = true
		}
	}

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
	visibleUpper := mainCount + uint32(len(t.inserts))
	keys := make([][]scm.Scmer, 0, len(recids))
	for _, idx := range recids {
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
		key := make([]scm.Scmer, len(sourceKeyCols))
		skip := false
		if idx < mainCount {
			for i, col := range cols {
				if needsTxReader[i] {
					key[i] = readers[i].GetValue(idx)
				} else {
					key[i] = col.GetValue(idx)
				}
				if key[i].IsNil() {
					skip = true
				}
			}
		} else {
			for i, col := range sourceKeyCols {
				if needsTxReader[i] {
					key[i] = readers[i].GetValue(idx)
				} else if _, isProxy := cols[i].(*StorageComputeProxy); isProxy {
					key[i] = cols[i].GetValue(idx)
				} else {
					key[i] = t.getDelta(int(idx-mainCount), col)
				}
				if key[i].IsNil() {
					skip = true
				}
			}
		}
		if !skip {
			keys = append(keys, key)
		}
	}
	return keys
}

func (t *table) projectJoinKeysToRecSet(currentTx *TxContext, targetKeyCols []string, keys [][]scm.Scmer, ss *scm.SessionState) *recSet {
	result := &recSet{tx: currentTx, table: t}
	if len(keys) == 0 {
		return result
	}
	querySeq := scm.CurrentQuerySeq()
	type targetPartResult struct {
		part recSetShard
		err  scanError
	}
	values := make(chan targetPartResult, t.recSetShardResultBufferSize())
	done := t.iterateShardsParallel(currentTx, nil, func(shard *storageShard, solo bool) {
		withTxSession(currentTx, func() scm.Scmer {
			defer func() {
				if rec := recover(); rec != nil {
					values <- targetPartResult{err: scanError{rec, string(debug.Stack())}}
				}
			}()
			// Cancellation contract: check only at the scheduling boundary, before entering
			// the shard. Once entered, a shard runs atomically without cancellation checks.
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			values <- targetPartResult{part: shard.projectJoinKeysPart(currentTx, targetKeyCols, keys, ss)}
			return scm.NewNil()
		})
	})
	if done == nil {
		close(values)
	} else {
		go func() {
			<-done
			close(values)
		}()
	}
	var joinErr scanError
	for msg := range values {
		if msg.err.r != nil {
			if joinErr.r == nil {
				joinErr = msg.err
			}
			continue
		}
		if joinErr.r != nil {
			continue
		}
		result.count += msg.part.count
		result.shards = append(result.shards, msg.part)
	}
	if joinErr.r != nil {
		panic(joinErr)
	}
	return result
}

func (t *storageShard) projectJoinKeysPart(currentTx *TxContext, targetKeyCols []string, keys [][]scm.Scmer, ss *scm.SessionState) recSetShard {
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
	targetCols := make([]ColumnStorage, len(targetKeyCols))
	targetReaders := make([]ColumnReader, len(targetKeyCols))
	targetNeedsTxReader := make([]bool, len(targetKeyCols))
	for i, col := range targetKeyCols {
		targetCols[i] = t.getColumnStorageOrPanicEx(col, skipShardReadLock, currentTx)
		targetReaders[i] = newCachedColumnReaderTx(targetCols[i], currentTx)
		if proxy, ok := targetCols[i].(*StorageComputeProxy); ok && proxy.hasSessionVariants() {
			targetNeedsTxReader[i] = true
		}
	}

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

	maxInsertIndex := len(t.inserts)
	visibleUpper := t.main_count + uint32(maxInsertIndex)
	acidMode := currentTx != nil && currentTx.Mode == TxACID
	seen := make(map[uint32]struct{})
	recids := make([]uint32, 0, 64)
	var buf [1024]uint32
	for _, key := range keys {
		bounds := make(boundaries, len(targetKeyCols))
		for i, col := range targetKeyCols {
			bounds[i] = columnboundaries{
				col:            col,
				matcher:        EqualMatcher,
				lower:          key[i],
				lowerInclusive: true,
				upper:          key[i],
				upperInclusive: true,
			}
		}
		reorderByFrequency(bounds, t.t)
		lower, upperLast := indexFromBoundaries(bounds)
		t.iterateIndexForce(currentTx, bounds, lower, upperLast, maxInsertIndex, buf[:], true, func(batch []uint32) bool {
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
				if _, ok := seen[idx]; ok {
					continue
				}
				if !t.projectJoinTargetMatches(idx, key, targetKeyCols, targetCols, targetReaders, targetNeedsTxReader, currentTx) {
					continue
				}
				seen[idx] = struct{}{}
				recids = append(recids, idx)
			}
			return true
		})
	}
	sort.Slice(recids, func(i, j int) bool { return recids[i] < recids[j] })
	return recSetShard{shard: t, recids: recids, count: int64(len(recids))}
}

func (t *storageShard) projectJoinTargetMatches(idx uint32, key []scm.Scmer, targetKeyCols []string, targetCols []ColumnStorage, targetReaders []ColumnReader, targetNeedsTxReader []bool, currentTx *TxContext) bool {
	for i, expected := range key {
		var actual scm.Scmer
		if idx < t.main_count {
			if targetNeedsTxReader[i] {
				actual = targetReaders[i].GetValue(idx)
			} else {
				actual = targetCols[i].GetValue(idx)
			}
		} else if targetNeedsTxReader[i] {
			actual = targetReaders[i].GetValue(idx)
		} else if _, isProxy := targetCols[i].(*StorageComputeProxy); isProxy {
			actual = targetCols[i].GetValue(idx)
		} else {
			actual = t.getDelta(int(idx-t.main_count), targetKeyCols[i])
		}
		if !scm.Equal(actual, expected) {
			return false
		}
	}
	return true
}

func (r *recSet) scan(currentTx *TxContext, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, aggregate2 scm.Scmer, isOuter bool) scm.Scmer {
	if r == nil {
		return neutral
	}
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") || (len(c) > 5 && c[:5] == "$set:") || (len(c) > 12 && c[:12] == "$invalidate:") {
			panic("recset scan mutation callbacks are not implemented")
		}
	}
	if currentTx == nil {
		currentTx = r.tx
	}
	ss := SessionStateFromTx(currentTx)
	querySeq := scm.CurrentQuerySeq()
	values := make(chan scanResult, len(r.shards))
	for i := range r.shards {
		rsShard := r.shards[i]
		if rsShard.count == 0 {
			continue
		}
		go func(part recSetShard) {
			withTxSession(currentTx, func() scm.Scmer {
				defer func() {
					if rec := recover(); rec != nil {
						values <- scanResult{err: scanError{rec, string(debug.Stack())}}
					}
				}()
				// Cancellation contract: check only at the scheduling boundary, before entering
				// the shard. Once entered, a shard runs atomically without cancellation checks.
				if ss != nil && ss.IsKilledSeq(querySeq) {
					panic("query killed")
				}
				res, cnt := part.shard.scanRecSetPart(part.recids, conditionCols, condition, callbackCols, callback, aggregate, neutral, currentTx, ss)
				values <- scanResult{res: res, outCount: cnt, inputCount: part.count}
				return scm.NewNil()
			})
		}(rsShard)
	}
	closeAfter := 0
	for _, part := range r.shards {
		if part.count > 0 {
			closeAfter++
		}
	}
	akkumulator := neutral
	hadValue := false
	var scanErr scanError
	if !aggregate2.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(aggregate2)
		for i := 0; i < closeAfter; i++ {
			msg := <-values
			if msg.err.r != nil {
				if scanErr.r == nil {
					scanErr = msg.err
				}
				continue
			}
			if msg.outCount > 0 {
				akkumulator = fn(akkumulator, msg.res)
				hadValue = true
			}
		}
	} else if !aggregate.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(aggregate)
		for i := 0; i < closeAfter; i++ {
			msg := <-values
			if msg.err.r != nil {
				if scanErr.r == nil {
					scanErr = msg.err
				}
				continue
			}
			if msg.outCount > 0 {
				akkumulator = fn(akkumulator, msg.res)
				hadValue = true
			}
		}
	} else {
		for i := 0; i < closeAfter; i++ {
			msg := <-values
			if msg.err.r != nil {
				if scanErr.r == nil {
					scanErr = msg.err
				}
				continue
			}
			hadValue = hadValue || msg.outCount > 0
		}
	}
	if scanErr.r != nil {
		panic(scanErr)
	}
	if !hadValue && isOuter {
		nullRow := buildOuterNullCallbackRow(callbackCols)
		if !aggregate2.IsNil() {
			fn := scm.OptimizeProcToSerialFunction(aggregate2)
			akkumulator = fn(akkumulator, scm.Apply(callback, nullRow...))
		} else if !aggregate.IsNil() {
			fn := scm.OptimizeProcToSerialFunction(aggregate)
			akkumulator = fn(akkumulator, scm.Apply(callback, nullRow...))
		} else {
			scm.Apply(callback, nullRow...)
		}
	}
	return akkumulator
}

func (r *recSet) scanExists(currentTx *TxContext, conditionCols []string, condition scm.Scmer) bool {
	if r == nil {
		return false
	}
	if currentTx == nil {
		currentTx = r.tx
	}
	ss := SessionStateFromTx(currentTx)
	querySeq := scm.CurrentQuerySeq()
	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	type existsResult struct {
		found bool
		err   scanError
	}
	values := make(chan existsResult, len(r.shards))
	var stop atomic.Bool
	closeAfter := 0
	for i := range r.shards {
		part := &r.shards[i]
		if part.count == 0 {
			continue
		}
		closeAfter++
		go func(part recSetShard) {
			withTxSession(currentTx, func() scm.Scmer {
				defer func() {
					if rec := recover(); rec != nil {
						values <- existsResult{err: scanError{rec, string(debug.Stack())}}
					}
				}()
				// Cancellation contract: check only at the scheduling boundary, before entering
				// the shard. Once entered, a shard runs atomically without cancellation checks.
				if ss != nil && ss.IsKilledSeq(querySeq) {
					panic("query killed")
				}
				found := part.shard.recSetPartExists(part.recids, conditionCols, conditionFn, currentTx, ss, &stop)
				if found {
					stop.Store(true)
				}
				values <- existsResult{found: found}
				return scm.NewNil()
			})
		}(*part)
	}
	var existsErr scanError
	found := false
	for i := 0; i < closeAfter; i++ {
		msg := <-values
		if msg.err.r != nil {
			if existsErr.r == nil {
				existsErr = msg.err
			}
			continue
		}
		if msg.found {
			found = true
		}
	}
	if existsErr.r != nil {
		panic(existsErr)
	}
	return found
}

func (t *storageShard) recSetPartExists(recids []uint32, conditionCols []string, conditionFn func(...scm.Scmer) scm.Scmer, currentTx *TxContext, ss *scm.SessionState, stop *atomic.Bool) bool {
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
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
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
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
	visibleUpper := mainCount + uint32(len(t.inserts))
	for _, idx := range recids {
		if stop != nil && stop.Load() {
			return false
		}
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
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else {
					cdataset[i] = c.GetValue(idx)
				}
			}
		} else {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = cReaders[i].GetValue(idx)
				} else {
					cdataset[i] = t.getDelta(int(idx-mainCount), col)
				}
			}
		}
		if scm.ToBool(conditionFn(cdataset...)) {
			return true
		}
	}
	return false
}

func (t *storageShard) scanRecSetPart(recids []uint32, conditionCols []string, condition scm.Scmer, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64) {
	conditionFn := scm.OptimizeProcToSerialFunction(condition)
	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)

	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
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
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
	}
	cdataset := make([]scm.Scmer, len(conditionCols))
	mapper := t.OpenMapReducer(callbackCols, callback, aggregate, skipShardReadLock, 0, nil, currentTx)
	defer mapper.Close()

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
	visibleUpper := mainCount + uint32(len(t.inserts))
	akkumulator := neutral
	var outCount int64
	var buf [1024]uint32
	pending := buf[:0]
	for _, idx := range recids {
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
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else {
					cdataset[i] = c.GetValue(idx)
				}
			}
		} else {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = cReaders[i].GetValue(idx)
				} else {
					cdataset[i] = t.getDelta(int(idx-mainCount), col)
				}
			}
		}
		if !scm.ToBool(conditionFn(cdataset...)) {
			continue
		}
		pending = append(pending, idx)
		outCount++
		if len(pending) == cap(buf) {
			if locked {
				t.mu.RUnlock()
				locked = false
			}
			akkumulator = mapper.Stream(akkumulator, pending, nil)
			pending = buf[:0]
			if !skipShardReadLock {
				t.mu.RLock()
				locked = true
			}
		}
	}
	if len(pending) > 0 {
		if locked {
			t.mu.RUnlock()
			locked = false
		}
		akkumulator = mapper.Stream(akkumulator, pending, nil)
	}
	if locked {
		t.mu.RUnlock()
		locked = false
	}
	mapper.FlushSideEffects()
	if outCount == 0 {
		return scm.NewNil(), 0
	}
	return akkumulator, outCount
}

func (t *storageShard) scan_order_recids(recids []uint32, conditionCols []string, condition scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, currentTx *TxContext, ss *scm.SessionState) *shardqueue {
	result := &shardqueue{shard: t}
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

	result.scols = make([]func(uint32) scm.Scmer, len(sortcols))
	for i, scol := range sortcols {
		if scol.IsString() {
			result.scols[i] = t.ColumnReaderTx(currentTx, scol.String())
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
	result.sortdirs = make([]func(...scm.Scmer) scm.Scmer, len(sortcols))
	for i := range sortcols {
		if i < len(sortdirs) && sortdirs[i] != nil {
			result.sortdirs[i] = wrapScanOrderComparator(sortdirs[i])
		} else {
			result.sortdirs[i] = wrapScanOrderComparator(defaultSortDir)
		}
	}

	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
	ccols := make([]ColumnStorage, len(conditionCols))
	cReaders := make([]ColumnReader, len(conditionCols))
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
		ccols[i] = t.getColumnStorageOrPanicEx(k, skipShardReadLock, currentTx)
		cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
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
	visibleUpper := mainCount + uint32(len(t.inserts))
	result.items = make([]uint32, 0, len(recids))
	for _, idx := range recids {
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
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else {
					cdataset[i] = c.GetValue(idx)
				}
			}
		} else {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(idx, 0)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = cReaders[i].GetValue(idx)
				} else {
					cdataset[i] = t.getDelta(int(idx-mainCount), col)
				}
			}
		}
		if scm.ToBool(conditionFn(cdataset...)) {
			result.items = append(result.items, idx)
		}
	}

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
	if len(sortcols) > 0 {
		if limit >= 0 && limitPartitionCols == 0 {
			result.items = topKByOrder(result.items, offset+limit, lessByID)
		} else {
			sort.Slice(result.items, func(i, j int) bool {
				return lessByID(result.items[i], result.items[j])
			})
		}
	}
	if limit >= 0 {
		perPart := offset + limit
		if perPart < 0 {
			perPart = len(result.items)
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
	_ = callbackCols
	return result
}
