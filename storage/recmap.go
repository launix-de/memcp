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

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/launix-de/memcp/scm"
)

// TagRecMap is the custom Scmer tag for query-local functional row mappings.
const TagRecMap = 104

// TagRecordRef identifies one query-local physical row. The pointer is the
// owning shard and the compact custom ID is its record ID.
const TagRecordRef = 105

const TagRecMapBuild = 106

// Mapper calls cross into Scheme only with a bounded tuple batch. The Go
// source buffers and final RecMap may grow with the selected domain, but no
// Scheme list is used as a relation-sized intermediate.
const recMapMapperBatchSize = 1024

// recMapTarget is a physical row identity valid for the query which built it.
// A nil shard denotes the SQL-NULL result of an outer/scalar-first lookup.
type recMapTarget struct {
	shard *storageShard
	recid uint32
}

func newRecordRef(shard *storageShard, recid uint32) scm.Scmer {
	return scm.NewCustomID(TagRecordRef, unsafe.Pointer(shard), recid)
}

func recordRefFromScmer(value scm.Scmer) recMapTarget {
	if !value.IsCustom(TagRecordRef) {
		panic("expected a record-ref, got " + scm.String(value))
	}
	return recMapTarget{shard: (*storageShard)(value.Custom(TagRecordRef)), recid: value.CustomID(TagRecordRef)}
}

// recMapShard stores only accepted source rows. sourceRecIDs is sorted, so a
// caller supplying a narrow RecSet window retains only that window.
type recMapShard struct {
	sourceShard  *storageShard
	sourceRecIDs []uint32
	targets      []recMapTarget
}

// recMap is an immutable query-local source-row -> optional target-row carrier.
// Like recSet it retains no transaction or cancellation state and must never be
// persisted or reused by another query. Physical row identities can change on
// rebuild, so a persistent cache needs a separate generation contract.
type recMap struct {
	source *table
	target *table
	shards []recMapShard
	count  int64
}

func NewRecMapScmer(rm *recMap) scm.Scmer {
	return scm.NewCustom(TagRecMap, unsafe.Pointer(rm))
}

func RecMapFromScmer(value scm.Scmer) *recMap {
	return (*recMap)(value.Custom(TagRecMap))
}

func (r *recMap) String() string {
	if r == nil || r.source == nil || r.target == nil {
		return "(recmap nil)"
	}
	return fmt.Sprintf("(recmap %q.%q %q.%q %d)",
		r.source.schema.Name, r.source.Name, r.target.schema.Name, r.target.Name, r.count)
}

func (r *recMap) lookup(shard *storageShard, recid uint32) (recMapTarget, bool) {
	if r == nil || shard == nil {
		return recMapTarget{}, false
	}
	for i := range r.shards {
		part := &r.shards[i]
		if part.sourceShard != shard {
			continue
		}
		position := sort.Search(len(part.sourceRecIDs), func(i int) bool {
			return part.sourceRecIDs[i] >= recid
		})
		if position == len(part.sourceRecIDs) || part.sourceRecIDs[position] != recid {
			return recMapTarget{}, false
		}
		return part.targets[position], true
	}
	return recMapTarget{}, false
}

func recMapCallClosure(shard *storageShard, currentTx *TxContext) *func(uint32, ...scm.Scmer) scm.Scmer {
	type cachedLookup struct {
		mapping *recMap
		part    *recMapShard
	}
	var cached atomic.Pointer[cachedLookup]
	fn := func(recid uint32, args ...scm.Scmer) scm.Scmer {
		if len(args) != 4 || !args[0].IsCustom(TagRecMap) {
			panic("$recmap_call expects recmap, columns, mapperFn, and ifNullFn")
		}
		rm := RecMapFromScmer(args[0])
		lookup := cached.Load()
		if lookup == nil || lookup.mapping != rm {
			lookup = &cachedLookup{mapping: rm}
			for i := range rm.shards {
				if rm.shards[i].sourceShard == shard {
					lookup.part = &rm.shards[i]
					break
				}
			}
			cached.Store(lookup)
		}
		if lookup.part == nil {
			return scm.Apply(args[3])
		}
		position := sort.Search(len(lookup.part.sourceRecIDs), func(i int) bool {
			return lookup.part.sourceRecIDs[i] >= recid
		})
		if position == len(lookup.part.sourceRecIDs) || lookup.part.sourceRecIDs[position] != recid {
			return scm.Apply(args[3])
		}
		target := lookup.part.targets[position]
		if target.shard == nil {
			return scm.Apply(args[3])
		}
		columns := scmerSliceToStrings(mustScmerSlice(args[1], "$recmap_call columns"))
		release := target.shard.acquireReadForScan(currentTx)
		defer release()
		target.shard.ensureLoaded()
		skipShardReadLock := target.shard.hasWriteOwnerForTx(currentTx)
		target.shard.ensureMainCount(skipShardReadLock)
		storages := make([]ColumnStorage, len(columns))
		for i, column := range columns {
			storages[i] = target.shard.getColumnStorageOrPanic(column, skipShardReadLock, currentTx)
		}
		if !skipShardReadLock {
			target.shard.mu.RLock()
			defer target.shard.mu.RUnlock()
		}
		values := make([]scm.Scmer, len(columns))
		for i, column := range columns {
			if target.recid < target.shard.main_count {
				values[i] = storages[i].GetValue(target.recid)
			} else if _, proxy := storages[i].(*StorageComputeProxy); proxy {
				values[i] = storages[i].GetValue(target.recid)
			} else {
				values[i] = target.shard.getDelta(int(target.recid-target.shard.main_count), column)
			}
		}
		return scm.Apply(args[2], values...)
	}
	return &fn
}

// recMapBuildRows is owned by one serial shard scan. The table-level combine
// collects completed builders on its serial consumer; no row-loop lock is
// needed, and the later mapper owns each builder on one fanout worker.
type recMapBuildRows struct {
	rows []recMapSourceRow
}

// newRecMapEquiFirstMapper resolves each bounded source-shard batch in one
// target projection. Source shards invoke the mapper concurrently; a scalar
// subscan per source record is intentionally not supported.
func newRecMapEquiFirstMapper(currentTx *TxContext, target *table, targetKeyCols []string, sourceKeyFn scm.Scmer) scm.Scmer {
	return scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if len(args) != 1 || !args[0].IsSlice() {
			panic("recmap equi mapper expects a batch of source tuples")
		}
		// General Scheme callbacks own mutable call frames. Prepare one per
		// shard worker; the target projection itself can then run in parallel.
		var mapper scm.SerialProc
		if !sourceKeyFn.IsNil() {
			mapper = scm.PrepareSerialProc(sourceKeyFn)
		}
		batch := args[0].Slice()
		keys := make([][]scm.Scmer, len(batch))
		order := make([]int, 0, len(batch))
		for i, input := range batch {
			if !input.IsSlice() {
				panic("recmap equi mapper expects source column tuples")
			}
			key := input.Slice()
			if !sourceKeyFn.IsNil() {
				mapped := mapper.Call(key)
				if !mapped.IsSlice() {
					panic("recmap equi mapper must return a target-key tuple")
				}
				key = mapped.Slice()
			}
			if len(key) != len(targetKeyCols) {
				panic("recmap equi mapper returned the wrong target-key width")
			}
			keys[i] = key
			if !recMapKeyHasNull(key) {
				order = append(order, i)
			}
		}
		sort.Slice(order, func(i, j int) bool { return compareProjectKey(keys[order[i]], keys[order[j]]) < 0 })
		unique := recSetProjectKeys{width: len(targetKeyCols), values: make([]scm.Scmer, 0, len(order)*len(targetKeyCols))}
		for i, index := range order {
			if i == 0 || compareProjectKey(keys[order[i-1]], keys[index]) != 0 {
				unique.values = append(unique.values, keys[index]...)
			}
		}
		output := make([]scm.Scmer, len(batch))
		if unique.count() == 0 {
			return scm.NewSlice(output)
		}
		targetRows := target.projectJoinKeysToRecSet(currentTx, targetKeyCols, unique, SessionStateFromTx(currentTx)).collectRecMapRows(currentTx, targetKeyCols, nil)
		sort.SliceStable(targetRows, func(i, j int) bool { return compareProjectKey(targetRows[i].key, targetRows[j].key) < 0 })
		for _, row := range targetRows {
			position := sort.Search(len(order), func(i int) bool { return compareProjectKey(keys[order[i]], row.key) >= 0 })
			for position < len(order) && compareProjectKey(keys[order[position]], row.key) == 0 {
				if output[order[position]].IsNil() {
					output[order[position]] = newRecordRef(row.shard, row.recid)
				}
				position++
			}
		}
		return scm.NewSlice(output)
	})
}

type recMapSourceRow struct {
	shard  *storageShard
	recid  uint32
	key    []scm.Scmer
	value  []scm.Scmer
	target recMapTarget
}

type recMapOrderRow struct {
	sourceShard *storageShard
	sourceRecID uint32
	target      recMapTarget
	values      []scm.Scmer
}

// orderRecSet selects an exact source-row window by values reached through the
// mapping. sortSides contains "source" or "target" for every column. This is
// deliberately query-local: the returned RecSet and all physical row IDs have
// the same lifetime as the RecMap which produced them.
func (r *recMap) orderRecSet(currentTx *TxContext, sortSides, sortColumns []string,
	sortDirections []func(...scm.Scmer) scm.Scmer, offset, limit int) *recSet {
	if r == nil || r.source == nil || r.target == nil || offset < 0 || limit < 0 ||
		len(sortSides) != len(sortColumns) || len(sortColumns) != len(sortDirections) {
		panic("recmap_order_recset: invalid mapping, sort specification, offset, or limit")
	}
	result := &recSet{table: r.source}
	if limit == 0 || r.count == 0 {
		return result
	}
	rows := make([]recMapOrderRow, 0, r.count)
	for partIndex := range r.shards {
		part := &r.shards[partIndex]
		for rowIndex, recid := range part.sourceRecIDs {
			rows = append(rows, recMapOrderRow{sourceShard: part.sourceShard,
				sourceRecID: recid, target: part.targets[rowIndex], values: make([]scm.Scmer, len(sortColumns))})
		}
	}
	fillRecMapOrderValues(currentTx, rows, sortSides, sortColumns)
	less := make([]func(scm.Scmer, scm.Scmer) bool, len(sortDirections))
	for i, relation := range sortDirections {
		less[i] = scm.OrderRelationLess(relation)
	}
	sort.Slice(rows, func(i, j int) bool {
		for column := range less {
			if less[column](rows[i].values[column], rows[j].values[column]) {
				return true
			}
			if less[column](rows[j].values[column], rows[i].values[column]) {
				return false
			}
		}
		for k := range rows[i].sourceShard.uuid {
			if rows[i].sourceShard.uuid[k] != rows[j].sourceShard.uuid[k] {
				return rows[i].sourceShard.uuid[k] < rows[j].sourceShard.uuid[k]
			}
		}
		return rows[i].sourceRecID < rows[j].sourceRecID
	})
	if offset >= len(rows) {
		return result
	}
	end := len(rows)
	if limit < len(rows)-offset {
		end = offset + limit
	}
	result.order = &recSetOrder{ranks: make(map[*storageShard]map[uint32]int64)}
	for rank, row := range rows[offset:end] {
		shardRanks := result.order.ranks[row.sourceShard]
		if shardRanks == nil {
			shardRanks = make(map[uint32]int64)
			result.order.ranks[row.sourceShard] = shardRanks
		}
		shardRanks[row.sourceRecID] = int64(rank)
	}
	byShard := make(map[*storageShard][]uint32)
	for _, row := range rows[offset:end] {
		byShard[row.sourceShard] = append(byShard[row.sourceShard], row.sourceRecID)
	}
	for shard, ids := range byShard {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		release := shard.GetRead()
		shard.mu.RLock()
		universe := shard.main_count + uint32(len(shard.inserts))
		shard.mu.RUnlock()
		release()
		part := newRecSetShardFromSortedIDs(shard, universe, ids)
		result.count += part.count
		result.shards = append(result.shards, part)
	}
	return result
}

func fillRecMapOrderValues(currentTx *TxContext, rows []recMapOrderRow, sides, columns []string) {
	for column, side := range sides {
		groups := make(map[*storageShard][]int)
		for rowIndex := range rows {
			shard := rows[rowIndex].sourceShard
			if side == "target" {
				shard = rows[rowIndex].target.shard
			} else if side != "source" {
				panic("recmap_order_recset: sort side must be source or target")
			}
			if shard != nil {
				groups[shard] = append(groups[shard], rowIndex)
			}
		}
		for shard, indexes := range groups {
			release := shard.acquireReadForScan(currentTx)
			func() {
				defer release()
				shard.ensureLoaded()
				skipLock := shard.hasWriteOwnerForTx(currentTx)
				shard.ensureMainCount(skipLock)
				storage := shard.getColumnStorageOrPanic(columns[column], skipLock, currentTx)
				if !skipLock {
					shard.mu.RLock()
					defer shard.mu.RUnlock()
				}
				for _, rowIndex := range indexes {
					recid := rows[rowIndex].sourceRecID
					if side == "target" {
						recid = rows[rowIndex].target.recid
					}
					if recid < shard.main_count {
						rows[rowIndex].values[column] = storage.GetValue(recid)
					} else if _, proxy := storage.(*StorageComputeProxy); proxy {
						rows[rowIndex].values[column] = storage.GetValue(recid)
					} else {
						rows[rowIndex].values[column] = shard.getDelta(int(recid-shard.main_count), columns[column])
					}
				}
			}()
		}
	}
}

func scanRecMap(currentTx *TxContext, source scm.Scmer, accessSchema scm.Scmer, accessValues []scm.Scmer,
	filterCols []string, filterFn scm.Scmer, mapCols []string, mapFn scm.Scmer, target *table) *recMap {
	var sourceTable *table
	var sourceRecSet *recSet
	if source.IsCustom(TagRecSet) {
		sourceRecSet = RecSetFromScmer(source)
		if sourceRecSet != nil {
			sourceTable = sourceRecSet.table
		}
	} else {
		sourceTable = TableFromScmer(source)
	}
	result := &recMap{source: sourceTable, target: target}
	if sourceTable == nil {
		return result
	}

	parts := make([]*recMapBuildRows, 0)
	callbackCols := append([]string{"$record_ref"}, mapCols...)
	reduce := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if len(args) != len(callbackCols)+1 {
			panic("scan_recmap received an invalid map-reducer frame")
		}
		var build *recMapBuildRows
		if args[0].IsNil() {
			build = &recMapBuildRows{}
		} else if args[0].IsCustom(TagRecMapBuild) {
			build = (*recMapBuildRows)(args[0].Custom(TagRecMapBuild))
		} else {
			panic("scan_recmap received an invalid shard accumulator")
		}
		sourceRef := recordRefFromScmer(args[1])
		build.rows = append(build.rows, recMapSourceRow{shard: sourceRef.shard, recid: sourceRef.recid,
			key: append([]scm.Scmer(nil), args[2:]...)})
		return scm.NewCustom(TagRecMapBuild, unsafe.Pointer(build))
	})
	combine := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if len(args) != 2 || !args[1].IsCustom(TagRecMapBuild) {
			panic("scan_recmap received an invalid shard result")
		}
		parts = append(parts, (*recMapBuildRows)(args[1].Custom(TagRecMapBuild)))
		return args[0]
	})
	if sourceRecSet != nil {
		sourceRecSet.scan(currentTx, accessSchema, accessValues, filterCols, filterFn,
			callbackCols, reduce, scm.NewNil(), combine, false)
	} else {
		sourceTable.scan(currentTx, accessSchema, accessValues, filterCols, filterFn,
			callbackCols, reduce, scm.NewNil(), combine, false)
	}
	if len(parts) == 0 {
		return result
	}
	result.shards = make([]recMapShard, len(parts))
	var firstPanic any
	var panicMu sync.Mutex
	done := runFanoutTasks(currentTx, len(parts), func(index int, _ bool) {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicMu.Lock()
				if firstPanic == nil {
					firstPanic = recovered
				}
				panicMu.Unlock()
			}
		}()
		rows := parts[index].rows
		sort.Slice(rows, func(i, j int) bool { return rows[i].recid < rows[j].recid })
		part := recMapShard{sourceShard: rows[0].shard, sourceRecIDs: make([]uint32, len(rows)), targets: make([]recMapTarget, len(rows))}
		for i := range rows {
			part.sourceRecIDs[i] = rows[i].recid
		}
		for start := 0; start < len(rows); start += recMapMapperBatchSize {
			end := start + recMapMapperBatchSize
			if end > len(rows) {
				end = len(rows)
			}
			inputs := make([]scm.Scmer, end-start)
			for i := start; i < end; i++ {
				inputs[i-start] = scm.NewSlice(rows[i].key)
			}
			mapped := scm.Apply(mapFn, scm.NewSlice(inputs))
			if !mapped.IsSlice() || len(mapped.Slice()) != len(inputs) {
				panic("scan_recmap mapper must return one target record-ref or nil per accepted source row")
			}
			for i, ref := range mapped.Slice() {
				if ref.IsNil() {
					continue
				}
				part.targets[start+i] = recordRefFromScmer(ref)
				if part.targets[start+i].shard == nil || part.targets[start+i].shard.t != target {
					panic("scan_recmap mapper returned a record-ref from another target table")
				}
			}
		}
		result.shards[index] = part
	})
	if done != nil {
		<-done
	}
	if firstPanic != nil {
		panic(firstPanic)
	}
	for _, part := range result.shards {
		result.count += int64(len(part.sourceRecIDs))
	}
	return result
}

func (r *recSet) collectRecMapSourceRows(currentTx *TxContext, sourceKeyCols []string) []recMapSourceRow {
	return r.collectRecMapRows(currentTx, sourceKeyCols, nil)
}

func (r *recSet) collectRecMapRows(currentTx *TxContext, sourceKeyCols []string, valueCols []string) []recMapSourceRow {
	if r == nil || r.table == nil || r.count == 0 {
		return nil
	}
	rows := make([]recMapSourceRow, 0, r.count)
	ss := SessionStateFromTx(currentTx)
	querySeq := querySeqFromTx(currentTx)
	for partIndex := range r.shards {
		part := &r.shards[partIndex]
		if part.count == 0 {
			continue
		}
		shard := part.shard
		release := shard.acquireReadForScan(currentTx)
		func() {
			defer release()
			if ss != nil && ss.IsKilledSeq(querySeq) {
				panic("query killed")
			}
			shard.ensureLoaded()
			skipShardReadLock := shard.hasWriteOwnerForTx(currentTx)
			shard.ensureMainCount(skipShardReadLock)
			allCols := append(append([]string(nil), sourceKeyCols...), valueCols...)
			columns := make([]ColumnStorage, len(allCols))
			for i, column := range allCols {
				columns[i] = shard.getColumnStorageOrPanic(column, skipShardReadLock, currentTx)
			}
			if !skipShardReadLock {
				shard.mu.RLock()
				if shard.t.hasTableLock() {
					shard.mu.RUnlock()
					shard.t.waitTableLock(ss, querySeq, false)
					shard.mu.RLock()
				}
				defer shard.mu.RUnlock()
			}
			acidMode := currentTx != nil && currentTx.Mode == TxACID
			visibleUpper := shard.main_count + uint32(len(shard.inserts))
			part.forEachID(func(recid uint32) bool {
				if recid >= visibleUpper {
					return true
				}
				if acidMode {
					if !currentTx.IsVisible(shard, recid) {
						return true
					}
				} else if shard.deletions.Get(uint(recid)) {
					return true
				}
				key := make([]scm.Scmer, len(sourceKeyCols))
				value := make([]scm.Scmer, len(valueCols))
				for i, column := range allCols {
					var item scm.Scmer
					if recid < shard.main_count {
						item = columns[i].GetValue(recid)
					} else if _, proxy := columns[i].(*StorageComputeProxy); proxy {
						item = columns[i].GetValue(recid)
					} else {
						item = shard.getDelta(int(recid-shard.main_count), column)
					}
					if i < len(sourceKeyCols) {
						key[i] = item
					} else {
						value[i-len(sourceKeyCols)] = item
					}
				}
				rows = append(rows, recMapSourceRow{shard: shard, recid: recid, key: key, value: value})
				return true
			})
		}()
	}
	return rows
}

func recMapKeyHasNull(key []scm.Scmer) bool {
	for _, value := range key {
		if value.IsNil() {
			return true
		}
	}
	return false
}

func (r *recMap) image() *recSet {
	if r == nil || r.target == nil {
		return &recSet{}
	}
	result := &recSet{table: r.target}
	byShard := make(map[*storageShard][]uint32)
	for _, part := range r.shards {
		for _, target := range part.targets {
			if target.shard != nil {
				byShard[target.shard] = append(byShard[target.shard], target.recid)
			}
		}
	}
	for shard, ids := range byShard {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		write := 0
		for _, id := range ids {
			if write == 0 || ids[write-1] != id {
				ids[write] = id
				write++
			}
		}
		ids = ids[:write]
		release := shard.GetRead()
		shard.mu.RLock()
		universe := shard.main_count + uint32(len(shard.inserts))
		shard.mu.RUnlock()
		release()
		part := newRecSetShardFromSortedIDs(shard, universe, ids)
		result.count += part.count
		result.shards = append(result.shards, part)
	}
	return result
}

func composeRecMaps(left, right *recMap) *recMap {
	if left == nil || right == nil || left.target != right.source {
		panic("recmap_compose: left target must be right source")
	}
	result := &recMap{source: left.source, target: right.target, count: left.count}
	rightParts := make(map[*storageShard]*recMapShard, len(right.shards))
	for i := range right.shards {
		rightParts[right.shards[i].sourceShard] = &right.shards[i]
	}
	for _, leftPart := range left.shards {
		part := recMapShard{
			sourceShard:  leftPart.sourceShard,
			sourceRecIDs: append([]uint32(nil), leftPart.sourceRecIDs...),
			targets:      make([]recMapTarget, len(leftPart.targets)),
		}
		for i, intermediate := range leftPart.targets {
			if intermediate.shard == nil {
				continue
			}
			rightPart := rightParts[intermediate.shard]
			if rightPart == nil {
				continue
			}
			position := sort.Search(len(rightPart.sourceRecIDs), func(i int) bool {
				return rightPart.sourceRecIDs[i] >= intermediate.recid
			})
			if position < len(rightPart.sourceRecIDs) && rightPart.sourceRecIDs[position] == intermediate.recid {
				part.targets[i] = rightPart.targets[position]
			}
		}
		result.shards = append(result.shards, part)
	}
	return result
}
