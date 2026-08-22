/*
Copyright (C) 2026  Carl-Philip Hänsch

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
import "github.com/launix-de/memcp/scm"

type orderedBatchRecord struct {
	shard *storageShard
	recid uint32
}

type orderedBatchPart struct {
	shard    *storageShard
	universe uint32
	recids   []uint32
}

// collectOrderedCandidateBatch uses scan_order's normal shard Top-K and global
// merge. Repeated calls request disjoint windows; batch growth is handled by
// scanOrderBatchAccept. The ordered record vector remains authoritative because
// a RecSet itself deliberately carries membership, not order.
func collectOrderedCandidateBatch(currentTx *TxContext, source scanOrderTableSpec, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, offset int, limit int) ([]orderedBatchRecord, *recSet) {
	table := source.backingTable()
	parts := make(map[*storageShard]*orderedBatchPart)
	partOrder := make([]*orderedBatchPart, 0)
	records := make([]orderedBatchRecord, 0, limit)
	source.conditionCols = nil
	source.condition = scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
	source.sortcols = sortcols
	source.callbackCols = nil
	source.callback = scm.NewNil()
	source.postOrderCols = nil
	source.postOrderFilter = scm.NewNil()
	source.perTableOffset = -1
	source.perTableLimit = -1
	source.recordVisitor = func(queue *shardqueue, recids []uint32) {
		part := parts[queue.shard]
		if part == nil {
			part = &orderedBatchPart{shard: queue.shard, universe: queue.universe}
			parts[queue.shard] = part
			partOrder = append(partOrder, part)
		}
		if queue.universe > part.universe {
			part.universe = queue.universe
		}
		for _, recid := range recids {
			records = append(records, orderedBatchRecord{shard: queue.shard, recid: recid})
			part.recids = append(part.recids, recid)
		}
	}

	scanOrderMulti(currentTx, []scanOrderTableSpec{source}, sortdirs, 0, offset, limit,
		scm.NewNil(), scm.NewNil(), false, scm.NewNil())

	batch := &recSet{tx: currentTx, table: table, shards: make([]recSetShard, 0, len(partOrder))}
	for _, part := range partOrder {
		sort.Slice(part.recids, func(i, j int) bool { return part.recids[i] < part.recids[j] })
		shardPart := newRecSetShardFromSortedIDs(part.shard, part.universe, part.recids)
		batch.count += shardPart.count
		batch.shards = append(batch.shards, shardPart)
	}
	return records, batch
}

func validateAcceptedBatch(currentTx *TxContext, batch *recSet, acceptedValue scm.Scmer) *recSet {
	if !acceptedValue.IsCustom(TagRecSet) {
		panic("scan_order_batch_accept: batch filter must return a recset")
	}
	accepted := RecSetFromScmer(acceptedValue)
	if accepted == nil || accepted.table != batch.table {
		panic("scan_order_batch_accept: batch filter must return a recset of the input table")
	}
	if accepted.tx != currentTx {
		panic("scan_order_batch_accept: batch filter returned a recset from a different transaction")
	}
	for i := range accepted.shards {
		part := &accepted.shards[i]
		if part.shard == nil || part.shard.t != batch.table {
			panic("scan_order_batch_accept: batch filter returned a recset of the input table")
		}
		part.forEachID(func(recid uint32) bool {
			if !batch.contains(part.shard, recid) {
				panic("scan_order_batch_accept: batch filter result is not a subset of its input batch")
			}
			return true
		})
	}
	return accepted
}

func optimizeScanOrderBatchAccept(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	const mapEnd = 10
	const reduceIdx = 11
	const neutralIdx = 12
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
		v[reduceIdx], _ = oc.OptimizeReducerCallback(rawReduce, neutralType, unknownScanType())
	}
	for i := 13; i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	oc.Ome.DecrLoopDepth()
	return scm.NewSlice(v), nil
}

// scanOrderBatchAccept incrementally requests ordered candidate windows. The
// first window has offset+limit records; every following window is twice as
// large. The filter sees an exact RecSet for only that window and returns an
// accepted subset. SQL OFFSET/LIMIT count accepted records, while driverOffset
// counts every candidate already examined.
func scanOrderBatchAccept(currentTx *TxContext, source scanOrderTableSpec, batchFilter scm.Scmer, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, limitPartitionCols int, offset int, limit int, callbackCols []string, callback scm.Scmer, aggregate scm.Scmer, neutral scm.Scmer, isOuter bool, notFoundValue scm.Scmer) scm.Scmer {
	if limitPartitionCols != 0 {
		panic("scan_order_batch_accept: partitioned limits are not supported")
	}
	if offset < 0 {
		panic("scan_order_batch_accept: offset must not be negative")
	}
	if limit < 0 {
		panic("scan_order_batch_accept: limit must be finite and non-negative")
	}
	if len(sortcols) != len(sortdirs) {
		panic("scan_order_batch_accept: sortcols and sortdirs must have equal length")
	}
	if source.recset != nil {
		if currentTx == nil {
			currentTx = source.recset.tx
		} else if currentTx != source.recset.tx {
			panic("scan_order_batch_accept: input recset belongs to a different transaction")
		}
	}
	if source.backingTable() == nil {
		panic("scan_order_batch_accept: input must have a backing table")
	}
	if offset > int(^uint(0)>>1)-limit {
		panic("scan_order_batch_accept: offset plus limit overflows")
	}

	result := neutral
	hadValue := false
	if limit == 0 {
		if isOuter {
			callbackFn := scm.OptimizeProcToSerialFunction(callback)
			reduceFn := scm.OptimizeProcToSerialFunction(aggregate)
			return reduceFn(result, callbackFn(buildOuterNullCallbackRow(callbackCols)...))
		}
		return notFoundValue
	}

	filterFn := scm.OptimizeProcToSerialFunction(batchFilter)
	mappers := make(map[*storageShard]*ShardMapReducer)
	defer func() {
		for _, mapper := range mappers {
			mapper.Close()
			mapper.FlushSideEffects()
		}
	}()
	mapperFor := func(shard *storageShard) *ShardMapReducer {
		mapper := mappers[shard]
		if mapper == nil {
			mapper = shard.OpenMapReducer(callbackCols, callback, aggregate, false, 0, nil, currentTx)
			mappers[shard] = mapper
		}
		return mapper
	}

	target := offset + limit
	batchSize := target
	if batchSize < 1 {
		batchSize = 1
	}
	driverOffset := 0
	acceptedCount := 0
	emittedCount := 0
	ss := SessionStateFromTx(currentTx)
	querySeq := querySeqFromTx(currentTx)

	for acceptedCount < target {
		if ss != nil && ss.IsKilledSeq(querySeq) {
			panic("query killed")
		}
		records, batch := collectOrderedCandidateBatch(currentTx, source, sortcols, sortdirs, driverOffset, batchSize)
		if len(records) == 0 {
			break
		}
		accepted := validateAcceptedBatch(currentTx, batch, filterFn(NewRecSetScmer(batch)))

		var pendingShard *storageShard
		pending := make([]uint32, 0, len(records))
		flush := func() bool {
			if len(pending) == 0 {
				return false
			}
			var broke bool
			result, broke = streamOrBreak(mapperFor(pendingShard), result, pending)
			pending = pending[:0]
			return broke
		}
		breakCaught := false
		for _, record := range records {
			if !accepted.contains(record.shard, record.recid) {
				continue
			}
			acceptedCount++
			if acceptedCount <= offset {
				continue
			}
			if emittedCount >= limit {
				break
			}
			if pendingShard != nil && pendingShard != record.shard && flush() {
				breakCaught = true
				break
			}
			pendingShard = record.shard
			pending = append(pending, record.recid)
			emittedCount++
			hadValue = true
		}
		if !breakCaught {
			breakCaught = flush()
		}
		if breakCaught || acceptedCount >= target {
			break
		}

		driverOffset += len(records)
		if len(records) < batchSize {
			break
		}
		if batchSize > (int(^uint(0)>>1)-driverOffset)/2 {
			batchSize = int(^uint(0)>>1) - driverOffset
		} else {
			batchSize *= 2
		}
		if batchSize <= 0 {
			panic(fmt.Sprintf("scan_order_batch_accept: candidate offset %d exceeds platform limits", driverOffset))
		}
	}

	if !hadValue && isOuter {
		callbackFn := scm.OptimizeProcToSerialFunction(callback)
		reduceFn := scm.OptimizeProcToSerialFunction(aggregate)
		result = reduceFn(result, callbackFn(buildOuterNullCallbackRow(callbackCols)...))
		hadValue = true
	}
	if !hadValue && !isOuter {
		return notFoundValue
	}
	return result
}
