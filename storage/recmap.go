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

func recMapReadTarget(currentTx *TxContext, target recMapTarget, columns []string,
	mapper, ifNull scm.Scmer) scm.Scmer {
	if target.shard == nil {
		return scm.Apply(ifNull)
	}
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
	return scm.Apply(mapper, values...)
}

type recMapRecordKey struct {
	shard *storageShard
	recid uint32
}

type recMapValuePosition struct {
	source recMapRecordKey
	target recMapTarget
	values []scm.Scmer
}

// newRecMapValueMapper materializes values reached through a RecMap one shard
// at a time. The returned lookup is immutable and lock-free; it is suitable
// for another operator's row callback without re-acquiring storage rights in
// that loop.
func newRecMapValueMapper(currentTx *TxContext, mapping *recMap, columns []string,
	mapper, ifNull scm.Scmer) scm.Scmer {
	fallback := scm.Apply(ifNull)
	values := make(map[recMapRecordKey]scm.Scmer)
	groups := make(map[*storageShard][]recMapValuePosition)
	if mapping != nil {
		values = make(map[recMapRecordKey]scm.Scmer, mapping.count)
		for partIndex := range mapping.shards {
			part := &mapping.shards[partIndex]
			for rowIndex, recid := range part.sourceRecIDs {
				source := recMapRecordKey{shard: part.sourceShard, recid: recid}
				target := part.targets[rowIndex]
				if target.shard == nil {
					values[source] = fallback
					continue
				}
				groups[target.shard] = append(groups[target.shard], recMapValuePosition{
					source: source, target: target,
				})
			}
		}
	}
	prepared := scm.PrepareSerialProc(mapper)
	for shard, positions := range groups {
		release := shard.acquireReadForScan(currentTx)
		func() {
			defer release()
			shard.ensureLoaded()
			skipShardReadLock := shard.hasWriteOwnerForTx(currentTx)
			shard.ensureMainCount(skipShardReadLock)
			storages := make([]ColumnStorage, len(columns))
			for i, column := range columns {
				storages[i] = shard.getColumnStorageOrPanic(column, skipShardReadLock, currentTx)
			}
			if !skipShardReadLock {
				shard.mu.RLock()
				defer shard.mu.RUnlock()
			}
			for positionIndex := range positions {
				position := &positions[positionIndex]
				position.values = make([]scm.Scmer, len(columns))
				for i, column := range columns {
					if position.target.recid < shard.main_count {
						position.values[i] = storages[i].GetValue(position.target.recid)
					} else if _, proxy := storages[i].(*StorageComputeProxy); proxy {
						position.values[i] = storages[i].GetValue(position.target.recid)
					} else {
						position.values[i] = shard.getDelta(int(position.target.recid-shard.main_count), column)
					}
				}
			}
		}()
		for _, position := range positions {
			values[position.source] = prepared.Call(position.values)
		}
	}
	return scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if len(args) != 1 {
			panic("recmap value mapper expects one source record-ref")
		}
		source := recordRefFromScmer(args[0])
		if value, ok := values[recMapRecordKey{shard: source.shard, recid: source.recid}]; ok {
			return value
		}
		return fallback
	})
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
		columns := scmerSliceToStrings(mustScmerSlice(args[1], "$recmap_call columns"))
		return recMapReadTarget(currentTx, lookup.part.targets[position], columns, args[2], args[3])
	}
	return &fn
}

// recMapBuildRows is owned by one serial shard scan. The table-level combine
// collects completed builders on its serial consumer; no row-loop lock is
// needed, and the later mapper owns each builder on one fanout worker.
type recMapBuildRows struct {
	rows   []recMapSourceRow
	mapper *scm.SerialProc
	width  int
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

// newRecMapEquiFirstOfMapper resolves one source key against alternative target
// key layouts. It is the batch equivalent of an equality OR such as
// (target.a = key OR target.b = key): the target is projected once per layout,
// while every source tuple retains scalar-first 0..1 semantics.
func newRecMapEquiFirstOfMapper(currentTx *TxContext, target *table, targetKeyAlternatives [][]string, sourceKeyFn scm.Scmer) scm.Scmer {
	return scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if len(args) != 1 || !args[0].IsSlice() {
			panic("recmap equi-first-of mapper expects a batch of source tuples")
		}
		if len(targetKeyAlternatives) == 0 {
			panic("recmap equi-first-of mapper expects target key alternatives")
		}
		width := len(targetKeyAlternatives[0])
		for _, columns := range targetKeyAlternatives {
			if len(columns) != width {
				panic("recmap equi-first-of target keys must have equal widths")
			}
		}
		var mapper scm.SerialProc
		if !sourceKeyFn.IsNil() {
			mapper = scm.PrepareSerialProc(sourceKeyFn)
		}
		batch := args[0].Slice()
		keys := make([][]scm.Scmer, len(batch))
		order := make([]int, 0, len(batch))
		for i, input := range batch {
			if !input.IsSlice() {
				panic("recmap equi-first-of mapper expects source column tuples")
			}
			key := input.Slice()
			if !sourceKeyFn.IsNil() {
				mapped := mapper.Call(key)
				if !mapped.IsSlice() {
					panic("recmap equi-first-of mapper must return a target-key tuple")
				}
				key = mapped.Slice()
			}
			if len(key) != width {
				panic("recmap equi-first-of mapper returned the wrong target-key width")
			}
			keys[i] = key
			if !recMapKeyHasNull(key) {
				order = append(order, i)
			}
		}
		sort.Slice(order, func(i, j int) bool { return compareProjectKey(keys[order[i]], keys[order[j]]) < 0 })
		unique := recSetProjectKeys{width: width, values: make([]scm.Scmer, 0, len(order)*width)}
		for i, index := range order {
			if i == 0 || compareProjectKey(keys[order[i-1]], keys[index]) != 0 {
				unique.values = append(unique.values, keys[index]...)
			}
		}
		output := make([]scm.Scmer, len(batch))
		if unique.count() == 0 {
			return scm.NewSlice(output)
		}
		for _, targetKeyCols := range targetKeyAlternatives {
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
		}
		return scm.NewSlice(output)
	})
}

// newRecMapHashFirstOfMapper evaluates target key alternatives once per target
// row and retains the resulting query-local hash image for every source batch.
// Unlike the direct projection mapper, alternatives may contain scalar probe
// results reached from the target row (for example a time-dependent owner).
func newRecMapHashFirstOfMapper(currentTx *TxContext, target *table, targetCols []string,
	targetKeyFn scm.Scmer, sourceKeyFn scm.Scmer) scm.Scmer {
	var buildOnce sync.Once
	var targetRows []recMapSourceRow
	var buildPanic any
	var targetKeyWidth int
	build := func() {
		defer func() { buildPanic = recover() }()
		parts := make([]*recMapBuildRows, 0)
		callbackCols := append([]string{"$record_ref"}, targetCols...)
		reduce := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			if len(args) != len(callbackCols)+1 {
				panic("recmap hash mapper received an invalid target reducer frame")
			}
			var rows *recMapBuildRows
			if args[0].IsNil() {
				mapper := scm.PrepareSerialProc(targetKeyFn)
				rows = &recMapBuildRows{mapper: &mapper}
			} else if args[0].IsCustom(TagRecMapBuild) {
				rows = (*recMapBuildRows)(args[0].Custom(TagRecMapBuild))
			} else {
				panic("recmap hash mapper received an invalid shard accumulator")
			}
			ref := recordRefFromScmer(args[1])
			alternatives := rows.mapper.Call(args[2:])
			if !alternatives.IsSlice() {
				panic("recmap hash target mapper must return target-key alternatives")
			}
			for _, rawKey := range alternatives.Slice() {
				if !rawKey.IsSlice() {
					panic("recmap hash target mapper must return target-key tuples")
				}
				key := rawKey.Slice()
				if !recMapKeyHasNull(key) {
					if rows.width == 0 {
						rows.width = len(key)
					} else if len(key) != rows.width {
						panic("recmap hash target mapper returned unequal key widths")
					}
					rows.rows = append(rows.rows, recMapSourceRow{target: ref, key: append([]scm.Scmer(nil), key...)})
				}
			}
			return scm.NewCustom(TagRecMapBuild, unsafe.Pointer(rows))
		})
		combine := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			if len(args) != 2 || !args[1].IsCustom(TagRecMapBuild) {
				panic("recmap hash mapper received an invalid shard result")
			}
			parts = append(parts, (*recMapBuildRows)(args[1].Custom(TagRecMapBuild)))
			return args[0]
		})
		target.scan(currentTx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
			scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
			callbackCols, reduce, scm.NewNil(), combine, false)
		for _, part := range parts {
			if targetKeyWidth == 0 {
				targetKeyWidth = part.width
			} else if part.width != 0 && part.width != targetKeyWidth {
				panic("recmap hash target mapper returned unequal key widths")
			}
			targetRows = append(targetRows, part.rows...)
		}
		sort.SliceStable(targetRows, func(i, j int) bool {
			return compareProjectKey(targetRows[i].key, targetRows[j].key) < 0
		})
	}
	return scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if len(args) != 1 || !args[0].IsSlice() {
			panic("recmap hash mapper expects a batch of source tuples")
		}
		buildOnce.Do(build)
		if buildPanic != nil {
			panic(buildPanic)
		}
		var mapper scm.SerialProc
		if !sourceKeyFn.IsNil() {
			mapper = scm.PrepareSerialProc(sourceKeyFn)
		}
		batch := args[0].Slice()
		keys := make([][]scm.Scmer, len(batch))
		order := make([]int, 0, len(batch))
		for i, input := range batch {
			if !input.IsSlice() {
				panic("recmap hash mapper expects source column tuples")
			}
			key := input.Slice()
			if !sourceKeyFn.IsNil() {
				mapped := mapper.Call(key)
				if !mapped.IsSlice() {
					panic("recmap hash source mapper must return a target-key tuple")
				}
				key = mapped.Slice()
			}
			if targetKeyWidth != 0 && len(key) != targetKeyWidth {
				panic("recmap hash source mapper returned the wrong target-key width")
			}
			keys[i] = key
			if !recMapKeyHasNull(key) {
				order = append(order, i)
			}
		}
		sort.Slice(order, func(i, j int) bool { return compareProjectKey(keys[order[i]], keys[order[j]]) < 0 })
		output := make([]scm.Scmer, len(batch))
		for _, row := range targetRows {
			position := sort.Search(len(order), func(i int) bool {
				return compareProjectKey(keys[order[i]], row.key) >= 0
			})
			for position < len(order) && compareProjectKey(keys[order[position]], row.key) == 0 {
				if output[order[position]].IsNil() {
					output[order[position]] = newRecordRef(row.target.shard, row.target.recid)
				}
				position++
			}
		}
		return scm.NewSlice(output)
	})
}

// newRecMapRangeFirstMapper resolves a batch of (point keys..., bound) tuples
// against one ordered target relation. The target is scanned once for the
// query-local mapper; individual source rows only binary-search that image.
// Relations <= and < choose the greatest qualifying value, while >= and >
// choose the smallest one.
func newRecMapRangeFirstMapper(currentTx *TxContext, target *table, targetPointCols []string,
	targetRangeCol, relation string, sourceKeyFn scm.Scmer) scm.Scmer {
	if len(targetPointCols) == 0 {
		panic("recmap range-first mapper expects point-key columns")
	}
	if relation != "<=" && relation != "<" && relation != ">=" && relation != ">" {
		panic("recmap range-first mapper expects <, <=, >, or >=")
	}
	var buildOnce sync.Once
	var targetRows []recMapSourceRow
	var buildPanic any
	build := func() {
		defer func() { buildPanic = recover() }()
		parts := make([]*recMapBuildRows, 0)
		columns := append(append([]string{"$record_ref"}, targetPointCols...), targetRangeCol)
		reduce := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			if len(args) != len(columns)+1 {
				panic("recmap range-first mapper received an invalid target reducer frame")
			}
			var rows *recMapBuildRows
			if args[0].IsNil() {
				rows = &recMapBuildRows{}
			} else if args[0].IsCustom(TagRecMapBuild) {
				rows = (*recMapBuildRows)(args[0].Custom(TagRecMapBuild))
			} else {
				panic("recmap range-first mapper received an invalid shard accumulator")
			}
			key := append([]scm.Scmer(nil), args[2:]...)
			if !recMapKeyHasNull(key) {
				rows.rows = append(rows.rows, recMapSourceRow{
					target: recordRefFromScmer(args[1]), key: key,
				})
			}
			return scm.NewCustom(TagRecMapBuild, unsafe.Pointer(rows))
		})
		combine := scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
			if len(args) != 2 || !args[1].IsCustom(TagRecMapBuild) {
				panic("recmap range-first mapper received an invalid shard result")
			}
			parts = append(parts, (*recMapBuildRows)(args[1].Custom(TagRecMapBuild)))
			return args[0]
		})
		target.scan(currentTx, newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, nil,
			scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) }),
			columns, reduce, scm.NewNil(), combine, false)
		for _, part := range parts {
			targetRows = append(targetRows, part.rows...)
		}
		sort.SliceStable(targetRows, func(i, j int) bool {
			return compareProjectKey(targetRows[i].key, targetRows[j].key) < 0
		})
	}
	return scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		if len(args) != 1 || !args[0].IsSlice() {
			panic("recmap range-first mapper expects a batch of source tuples")
		}
		buildOnce.Do(build)
		if buildPanic != nil {
			panic(buildPanic)
		}
		var mapper scm.SerialProc
		if !sourceKeyFn.IsNil() {
			mapper = scm.PrepareSerialProc(sourceKeyFn)
		}
		batch := args[0].Slice()
		output := make([]scm.Scmer, len(batch))
		pointWidth := len(targetPointCols)
		for i, input := range batch {
			if !input.IsSlice() {
				panic("recmap range-first mapper expects source column tuples")
			}
			key := input.Slice()
			if !sourceKeyFn.IsNil() {
				mapped := mapper.Call(key)
				if !mapped.IsSlice() {
					panic("recmap range-first source mapper must return a key tuple")
				}
				key = mapped.Slice()
			}
			if len(key) != pointWidth+1 || recMapKeyHasNull(key) {
				continue
			}
			point := key[:pointWidth]
			bound := key[pointWidth:]
			start := sort.Search(len(targetRows), func(index int) bool {
				return compareProjectKey(targetRows[index].key[:pointWidth], point) >= 0
			})
			best := -1
			for index := start; index < len(targetRows) &&
				compareProjectKey(targetRows[index].key[:pointWidth], point) == 0; index++ {
				cmp := compareProjectKey(targetRows[index].key[pointWidth:], bound)
				qualifies := (relation == "<=" && cmp <= 0) || (relation == "<" && cmp < 0) ||
					(relation == ">=" && cmp >= 0) || (relation == ">" && cmp > 0)
				if !qualifies {
					continue
				}
				best = index
				if relation == ">=" || relation == ">" {
					break
				}
			}
			if best >= 0 {
				output[i] = newRecordRef(targetRows[best].target.shard, targetRows[best].target.recid)
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

// extendRecMap keeps the original source identity while deriving another
// target from both original-source columns and columns reached by a previous
// mapping. Values are gathered shard-wise before invoking the mapper, so an
// FK chain remains a batch operation instead of rebuilding a RecMap per row.
func extendRecMap(currentTx *TxContext, previous *recMap, sourceCols, previousTargetCols []string,
	mapFn scm.Scmer, target *table) *recMap {
	result := &recMap{target: target}
	if previous == nil || previous.source == nil {
		return result
	}
	result.source = previous.source
	rows := make([]recMapSourceRow, 0, previous.count)
	targetPositions := make(map[*storageShard][]int)
	for partIndex := range previous.shards {
		part := &previous.shards[partIndex]
		func() {
			release := part.sourceShard.acquireReadForScan(currentTx)
			defer release()
			part.sourceShard.ensureLoaded()
			skipLock := part.sourceShard.hasWriteOwnerForTx(currentTx)
			part.sourceShard.ensureMainCount(skipLock)
			storages := make([]ColumnStorage, len(sourceCols))
			for i, column := range sourceCols {
				storages[i] = part.sourceShard.getColumnStorageOrPanic(column, skipLock, currentTx)
			}
			if !skipLock {
				part.sourceShard.mu.RLock()
				defer part.sourceShard.mu.RUnlock()
			}
			for i, recid := range part.sourceRecIDs {
				values := make([]scm.Scmer, len(sourceCols), len(sourceCols)+len(previousTargetCols))
				for columnIndex, column := range sourceCols {
					if recid < part.sourceShard.main_count {
						values[columnIndex] = storages[columnIndex].GetValue(recid)
					} else if _, proxy := storages[columnIndex].(*StorageComputeProxy); proxy {
						values[columnIndex] = storages[columnIndex].GetValue(recid)
					} else {
						values[columnIndex] = part.sourceShard.getDelta(int(recid-part.sourceShard.main_count), column)
					}
				}
				rows = append(rows, recMapSourceRow{shard: part.sourceShard, recid: recid, key: values, target: part.targets[i]})
				position := len(rows) - 1
				if part.targets[i].shard != nil {
					targetPositions[part.targets[i].shard] = append(targetPositions[part.targets[i].shard], position)
				}
			}
		}()
	}
	for shard, positions := range targetPositions {
		func() {
			release := shard.acquireReadForScan(currentTx)
			defer release()
			shard.ensureLoaded()
			skipLock := shard.hasWriteOwnerForTx(currentTx)
			shard.ensureMainCount(skipLock)
			storages := make([]ColumnStorage, len(previousTargetCols))
			for i, column := range previousTargetCols {
				storages[i] = shard.getColumnStorageOrPanic(column, skipLock, currentTx)
			}
			if !skipLock {
				shard.mu.RLock()
				defer shard.mu.RUnlock()
			}
			for _, position := range positions {
				previousTarget := rows[position].target
				for columnIndex, column := range previousTargetCols {
					var value scm.Scmer
					if previousTarget.recid < shard.main_count {
						value = storages[columnIndex].GetValue(previousTarget.recid)
					} else if _, proxy := storages[columnIndex].(*StorageComputeProxy); proxy {
						value = storages[columnIndex].GetValue(previousTarget.recid)
					} else {
						value = shard.getDelta(int(previousTarget.recid-shard.main_count), column)
					}
					rows[position].key = append(rows[position].key, value)
				}
			}
		}()
	}
	for i := range rows {
		rows[i].target = recMapTarget{}
		if len(rows[i].key) != len(sourceCols)+len(previousTargetCols) {
			rows[i].key = append(rows[i].key, make([]scm.Scmer, len(previousTargetCols))...)
		}
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
			panic("recmap extend mapper must return one target record-ref or nil per source row")
		}
		for i, ref := range mapped.Slice() {
			if ref.IsNil() {
				continue
			}
			rows[start+i].target = recordRefFromScmer(ref)
			if rows[start+i].target.shard == nil || rows[start+i].target.shard.t != target {
				panic("recmap extend mapper returned a record-ref from another target table")
			}
		}
	}
	for _, previousPart := range previous.shards {
		part := recMapShard{sourceShard: previousPart.sourceShard}
		for i := range rows {
			if rows[i].shard == previousPart.sourceShard {
				part.sourceRecIDs = append(part.sourceRecIDs, rows[i].recid)
				part.targets = append(part.targets, rows[i].target)
			}
		}
		result.count += int64(len(part.sourceRecIDs))
		result.shards = append(result.shards, part)
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
