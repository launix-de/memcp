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

import "sort"
import "runtime"
import "container/heap"
import "github.com/launix-de/memcp/scm"

func optimizeScanJoinOrder(v []scm.Scmer, oc *scm.OptimizerContext, _ bool) (scm.Scmer, *scm.TypeDescriptor) {
	const mapIdx, reduceIdx, neutralIdx, reduce2Idx, outerIdx, notFoundIdx = 14, 15, 16, 17, 18, 19
	rawMap := v[mapIdx]
	rawReduce := scm.NewNil()
	rawReduce2 := scm.NewNil()
	if len(v) > reduceIdx {
		rawReduce = v[reduceIdx]
	}
	if len(v) > reduce2Idx {
		rawReduce2 = v[reduce2Idx]
	}
	for i := 1; i < mapIdx && i < len(v); i++ {
		v[i], _ = oc.OptimizeSub(v[i], true)
	}
	neutralType := unknownScanType()
	if len(v) > neutralIdx {
		v[neutralIdx], neutralType = oc.OptimizeSub(v[neutralIdx], true)
		neutralType = normalizeScanType(neutralType)
	}
	if !rawReduce.IsNil() {
		oc.SetCallbackReturnFlow(scm.CallbackReturnFlow(rawMap, rawReduce, 1))
	}
	oc.Ome.IncrLoopDepth()
	optimizedMap, mapType := oc.OptimizeSub(rawMap, true)
	v[mapIdx] = optimizedMap
	mapType = normalizeScanType(mapType)
	resultType := neutralType
	if !rawReduce.IsNil() {
		v[reduceIdx], resultType = oc.OptimizeReducerCallback(rawReduce, neutralType, mapType)
	}
	if !rawReduce2.IsNil() {
		v[reduce2Idx], resultType = oc.OptimizeReducerCallback(rawReduce2, resultType, resultType)
	}
	if len(v) > outerIdx {
		v[outerIdx], _ = oc.OptimizeSub(v[outerIdx], true)
	}
	if len(v) > notFoundIdx {
		v[notFoundIdx], _ = oc.OptimizeSub(v[notFoundIdx], true)
	}
	oc.Ome.DecrLoopDepth()
	return scm.NewSlice(v), resultType
}

// scanJoinOrderColumn identifies one physical column in the joined tuple.
// Table is the zero-based position in scanJoinOrderSpec.inputs.
type scanJoinOrderColumn struct {
	table  int
	column string
}

// scanJoinOrderInput describes one left-deep equi-join input. The first input
// is the ORDER BY driver. Every later input joins targetKeyCols against values
// taken from the already assembled tuple through sourceKeyCols.
type scanJoinOrderInput struct {
	table         *table
	filterCols    []string
	filter        scm.Scmer
	sourceKeyCols []scanJoinOrderColumn
	targetKeyCols []string

	readCols     []string
	readColIndex map[string]int
	lateMapCols  []string
	lateMapIndex map[string]int
}

type scanJoinOrderSpec struct {
	inputs []scanJoinOrderInput

	orderCols          []scanJoinOrderColumn
	orderDirs          []func(...scm.Scmer) scm.Scmer
	limitPartitionCols int
	offset             int
	limit              int

	joinFilterCols []scanJoinOrderColumn
	joinFilter     scm.Scmer
	mapCols        []scanJoinOrderColumn
	mapFn          scm.Scmer
	reduceFn       scm.Scmer
	reduce2Fn      scm.Scmer
	neutral        scm.Scmer
	isOuter        bool
	notFoundValue  scm.Scmer
}

func scanJoinOrderUsesDriverOrder(spec *scanJoinOrderSpec) bool {
	for _, ref := range spec.orderCols {
		if ref.table != 0 {
			return false
		}
	}
	return true
}

type scanJoinOrderRecord struct {
	shard  *storageShard
	recid  uint32
	values []scm.Scmer
}

type scanJoinOrderTuple struct {
	records []*scanJoinOrderRecord
	order   int
}

type scanJoinOrderShardRange struct {
	lower          scm.Scmer
	hasLower       bool
	lowerExclusive bool
	upper          scm.Scmer
	hasUpper       bool
}

type scanJoinOrderShardStream struct {
	shard   *storageShard
	records []*scanJoinOrderRecord
	ranges  map[string]scanJoinOrderShardRange
}

type scanJoinOrderTopKCollector struct {
	spec  *scanJoinOrderSpec
	keep  int
	items []*scanJoinOrderTuple
}

func (collector *scanJoinOrderTopKCollector) Len() int { return len(collector.items) }
func (collector *scanJoinOrderTopKCollector) Less(i int, j int) bool {
	// Reverse final order: heap[0] is the worst retained tuple.
	return lessScanJoinOrderTuple(collector.spec, collector.items[j], collector.items[i])
}
func (collector *scanJoinOrderTopKCollector) Swap(i int, j int) {
	collector.items[i], collector.items[j] = collector.items[j], collector.items[i]
}
func (collector *scanJoinOrderTopKCollector) Push(value any) {
	collector.items = append(collector.items, value.(*scanJoinOrderTuple))
}
func (collector *scanJoinOrderTopKCollector) Pop() any {
	last := len(collector.items) - 1
	value := collector.items[last]
	collector.items[last] = nil
	collector.items = collector.items[:last]
	return value
}
func (collector *scanJoinOrderTopKCollector) add(tuple *scanJoinOrderTuple) {
	if collector.keep < 0 {
		collector.items = append(collector.items, tuple)
		return
	}
	if collector.keep == 0 {
		return
	}
	if len(collector.items) < collector.keep {
		heap.Push(collector, tuple)
		return
	}
	if lessScanJoinOrderTuple(collector.spec, tuple, collector.items[0]) {
		collector.items[0] = tuple
		heap.Fix(collector, 0)
	}
}

func (collector *scanJoinOrderTopKCollector) sorted() []*scanJoinOrderTuple {
	sort.SliceStable(collector.items, func(i int, j int) bool {
		return lessScanJoinOrderTuple(collector.spec, collector.items[i], collector.items[j])
	})
	return collector.items
}

func scanJoinTrue() scm.Scmer {
	return scm.NewFunc(func(...scm.Scmer) scm.Scmer { return scm.NewBool(true) })
}

func scanJoinOrderOuterResult(spec *scanJoinOrderSpec, result scm.Scmer) scm.Scmer {
	mapProgram := scm.PrepareSerialProc(spec.mapFn)
	reduce := spec.reduceFn
	if reduce.IsNil() {
		reduce = scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] })
	}
	reduceProgram := scm.PrepareSerialProc(reduce)
	nulls := make([]scm.Scmer, len(spec.mapCols))
	for i := range nulls {
		nulls[i] = scm.NewNil()
	}
	args := [2]scm.Scmer{result, mapProgram.Call(nulls)}
	return reduceProgram.Call(args[:])
}

func appendScanJoinColumn(cols []string, indexes map[string]int, col string) ([]string, map[string]int) {
	if indexes == nil {
		indexes = make(map[string]int)
	}
	if _, exists := indexes[col]; !exists {
		indexes[col] = len(cols)
		cols = append(cols, col)
	}
	return cols, indexes
}

func prepareScanJoinOrderSpec(spec *scanJoinOrderSpec) {
	if len(spec.inputs) == 0 {
		panic("scan_join_order: at least one input table is required")
	}
	if spec.offset < 0 || spec.limit < -1 {
		panic("scan_join_order: invalid offset or limit")
	}
	if spec.limitPartitionCols != 0 {
		panic("scan_join_order: partitioned limits are not supported")
	}
	if !spec.reduce2Fn.IsNil() && (spec.offset != 0 || spec.limit != -1) {
		panic("scan_join_order: reduce2 requires an unlimited scan without offset")
	}
	if len(spec.orderCols) != len(spec.orderDirs) {
		panic("scan_join_order: order columns and directions must have equal length")
	}
	for i := range spec.inputs {
		input := &spec.inputs[i]
		if input.table == nil {
			panic("scan_join_order: input table is nil")
		}
		if input.filter.IsNil() {
			input.filter = scanJoinTrue()
		}
		if i == 0 {
			if len(input.sourceKeyCols) != 0 || len(input.targetKeyCols) != 0 {
				panic("scan_join_order: driver input must not declare join keys")
			}
		} else if len(input.sourceKeyCols) == 0 || len(input.sourceKeyCols) != len(input.targetKeyCols) {
			panic("scan_join_order: each inner input needs equally wide source and target join keys")
		}
		for _, ref := range input.sourceKeyCols {
			if ref.table < 0 || ref.table >= i {
				panic("scan_join_order: source join columns must reference an earlier input")
			}
		}
		for _, col := range input.filterCols {
			input.readCols, input.readColIndex = appendScanJoinColumn(input.readCols, input.readColIndex, col)
		}
		for _, col := range input.targetKeyCols {
			input.readCols, input.readColIndex = appendScanJoinColumn(input.readCols, input.readColIndex, col)
		}
	}
	allRefs := make([]scanJoinOrderColumn, 0, len(spec.orderCols)+len(spec.joinFilterCols))
	allRefs = append(allRefs, spec.orderCols...)
	allRefs = append(allRefs, spec.joinFilterCols...)
	for i := 1; i < len(spec.inputs); i++ {
		allRefs = append(allRefs, spec.inputs[i].sourceKeyCols...)
		spec.inputs[i].table.AddPartitioningScore(spec.inputs[i].targetKeyCols)
		for _, ref := range spec.inputs[i].sourceKeyCols {
			spec.inputs[ref.table].table.AddPartitioningScore([]string{ref.column})
		}
	}
	for _, ref := range allRefs {
		if ref.table < 0 || ref.table >= len(spec.inputs) {
			panic("scan_join_order: joined column references an unknown input")
		}
		input := &spec.inputs[ref.table]
		input.readCols, input.readColIndex = appendScanJoinColumn(input.readCols, input.readColIndex, ref.column)
	}
	for _, ref := range spec.mapCols {
		if ref.table < 0 || ref.table >= len(spec.inputs) {
			panic("scan_join_order: mapped column references an unknown input")
		}
		input := &spec.inputs[ref.table]
		if _, prepared := input.readColIndex[ref.column]; !prepared {
			input.lateMapCols, input.lateMapIndex = appendScanJoinColumn(input.lateMapCols, input.lateMapIndex, ref.column)
		}
	}
}

func collectScanJoinOrderRecords(currentTx *TxContext, input *scanJoinOrderInput, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer, offset int, limit int, acceptCols []string, accept scm.Scmer) []orderedBatchRecord {
	records := make([]orderedBatchRecord, 0)
	source := scanOrderTableSpec{
		table:          input.table,
		conditionCols:  input.filterCols,
		condition:      input.filter,
		acceptCols:     acceptCols,
		accept:         accept,
		sortcols:       sortcols,
		callbackCols:   nil,
		callback:       scm.NewNil(),
		perTableOffset: -1,
		perTableLimit:  -1,
	}
	source.recordVisitor = func(queue *shardqueue, recids []uint32) {
		for _, recid := range recids {
			records = append(records, orderedBatchRecord{shard: queue.shard, recid: recid})
		}
	}
	scanOrderMulti(currentTx, []scanOrderTableSpec{source}, sortdirs, 0, offset, limit,
		scm.NewNil(), scm.NewNil(), false, scm.NewNil())
	return records
}

func materializeScanJoinRecords(currentTx *TxContext, input *scanJoinOrderInput, refs []orderedBatchRecord) []*scanJoinOrderRecord {
	result := make([]*scanJoinOrderRecord, 0, len(refs))
	mappers := make(map[*storageShard]*ShardMapReducer)
	identity := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		return scm.NewSlice(append([]scm.Scmer(nil), values...))
	})
	defer func() {
		for _, mapper := range mappers {
			mapper.Close()
		}
	}()
	for _, ref := range refs {
		mapper := mappers[ref.shard]
		if mapper == nil {
			mapper = ref.shard.OpenMapReducer(input.readCols, identity, scm.NewNil(), false, 0, nil, currentTx)
			mappers[ref.shard] = mapper
		}
		values := mapper.MapOne(ref.recid)
		result = append(result, &scanJoinOrderRecord{shard: ref.shard, recid: ref.recid, values: values.Slice()})
	}
	return result
}

type scanJoinOrderDriverConstraint struct {
	positions []int
	keys      [][]scm.Scmer
}

// scanJoinOrderCandidateDriver narrows the ordered driver with join keys that
// survived directly joined inner-table filters. It is returned separately
// from the driver predicate so every shard owns both optimized callbacks while
// the original predicate remains available for boundary extraction.
func scanJoinOrderCandidateDriver(spec *scanJoinOrderSpec, innerRecords [][]*scanJoinOrderRecord) (scanJoinOrderInput, []string, scm.Scmer) {
	driver := spec.inputs[0]
	columns := make([]string, 0)
	columnIndexes := make(map[string]int)
	constraints := make([]scanJoinOrderDriverConstraint, 0, len(spec.inputs)-1)
	for inputIndex := 1; inputIndex < len(spec.inputs); inputIndex++ {
		inner := &spec.inputs[inputIndex]
		positions := make([]int, len(inner.sourceKeyCols))
		direct := true
		for i, ref := range inner.sourceKeyCols {
			if ref.table != 0 {
				direct = false
				break
			}
			position, found := columnIndexes[ref.column]
			if !found {
				position = len(columns)
				columnIndexes[ref.column] = position
				columns = append(columns, ref.column)
			}
			positions[i] = position
		}
		if !direct {
			continue
		}
		keys := make([][]scm.Scmer, 0, len(innerRecords[inputIndex]))
		for _, record := range innerRecords[inputIndex] {
			key := scanJoinOrderRecordKey(inner, record, inner.targetKeyCols, nil)
			if scanJoinOrderKeyHasNull(key) || (len(keys) > 0 && compareProjectKey(keys[len(keys)-1], key) == 0) {
				continue
			}
			keys = append(keys, append([]scm.Scmer(nil), key...))
		}
		constraints = append(constraints, scanJoinOrderDriverConstraint{positions: positions, keys: keys})
	}
	if len(constraints) == 0 {
		return driver, nil, scm.NewNil()
	}
	predicate := scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
		for _, constraint := range constraints {
			key := make([]scm.Scmer, len(constraint.positions))
			for i, position := range constraint.positions {
				key[i] = values[position]
			}
			if scanJoinOrderKeyHasNull(key) {
				return scm.NewBool(false)
			}
			position := sort.Search(len(constraint.keys), func(i int) bool {
				return compareProjectKey(constraint.keys[i], key) >= 0
			})
			if position == len(constraint.keys) || compareProjectKey(constraint.keys[position], key) != 0 {
				return scm.NewBool(false)
			}
		}
		return scm.NewBool(true)
	})
	if len(driver.filterCols) == 0 {
		driver.filterCols = columns
		driver.filter = predicate
		return driver, nil, scm.NewNil()
	}
	return driver, columns, predicate
}

func scanJoinOrderRecordValue(input *scanJoinOrderInput, record *scanJoinOrderRecord, col string) scm.Scmer {
	position, found := input.readColIndex[col]
	if !found {
		panic("scan_join_order: column was not prepared")
	}
	return record.values[position]
}

func scanJoinOrderTupleValues(spec *scanJoinOrderSpec, tuple *scanJoinOrderTuple, refs []scanJoinOrderColumn, dst []scm.Scmer) []scm.Scmer {
	if cap(dst) < len(refs) {
		dst = make([]scm.Scmer, len(refs))
	} else {
		dst = dst[:len(refs)]
	}
	for i, ref := range refs {
		dst[i] = scanJoinOrderRecordValue(&spec.inputs[ref.table], tuple.records[ref.table], ref.column)
	}
	return dst
}

type scanJoinOrderMapReaders struct {
	currentTx *TxContext
	spec      *scanJoinOrderSpec
	mappers   []map[*storageShard]*ShardMapReducer
	identity  scm.Scmer
}

func newScanJoinOrderMapReaders(currentTx *TxContext, spec *scanJoinOrderSpec) *scanJoinOrderMapReaders {
	return &scanJoinOrderMapReaders{
		currentTx: currentTx,
		spec:      spec,
		mappers:   make([]map[*storageShard]*ShardMapReducer, len(spec.inputs)),
		identity: scm.NewFunc(func(values ...scm.Scmer) scm.Scmer {
			return scm.NewSlice(append([]scm.Scmer(nil), values...))
		}),
	}
}

func (readers *scanJoinOrderMapReaders) close() {
	for _, inputMappers := range readers.mappers {
		for _, mapper := range inputMappers {
			mapper.Close()
		}
	}
}

func (readers *scanJoinOrderMapReaders) values(tuple *scanJoinOrderTuple, dst []scm.Scmer) []scm.Scmer {
	if cap(dst) < len(readers.spec.mapCols) {
		dst = make([]scm.Scmer, len(readers.spec.mapCols))
	} else {
		dst = dst[:len(readers.spec.mapCols)]
	}
	lateValues := make([][]scm.Scmer, len(readers.spec.inputs))
	for i, ref := range readers.spec.mapCols {
		input := &readers.spec.inputs[ref.table]
		record := tuple.records[ref.table]
		if position, prepared := input.readColIndex[ref.column]; prepared {
			dst[i] = record.values[position]
			continue
		}
		if lateValues[ref.table] == nil {
			inputMappers := readers.mappers[ref.table]
			if inputMappers == nil {
				inputMappers = make(map[*storageShard]*ShardMapReducer)
				readers.mappers[ref.table] = inputMappers
			}
			mapper := inputMappers[record.shard]
			if mapper == nil {
				mapper = record.shard.OpenMapReducer(input.lateMapCols, readers.identity, scm.NewNil(), false, 0, nil, readers.currentTx)
				inputMappers[record.shard] = mapper
			}
			lateValues[ref.table] = mapper.MapOne(record.recid).Slice()
		}
		dst[i] = lateValues[ref.table][input.lateMapIndex[ref.column]]
	}
	return dst
}

func scanJoinOrderKeyHasNull(key []scm.Scmer) bool {
	for _, value := range key {
		if value.IsNil() {
			return true
		}
	}
	return false
}

func scanJoinOrderRecordKey(input *scanJoinOrderInput, record *scanJoinOrderRecord, cols []string, dst []scm.Scmer) []scm.Scmer {
	if cap(dst) < len(cols) {
		dst = make([]scm.Scmer, len(cols))
	} else {
		dst = dst[:len(cols)]
	}
	for i, col := range cols {
		dst[i] = scanJoinOrderRecordValue(input, record, col)
	}
	return dst
}

func scanJoinOrderTupleKey(spec *scanJoinOrderSpec, tuple *scanJoinOrderTuple, refs []scanJoinOrderColumn, dst []scm.Scmer) []scm.Scmer {
	return scanJoinOrderTupleValues(spec, tuple, refs, dst)
}

func joinScanOrderInputRange(spec *scanJoinOrderSpec, tuples []*scanJoinOrderTuple, inputIndex int, inner []*scanJoinOrderRecord) []*scanJoinOrderTuple {
	input := &spec.inputs[inputIndex]
	result := make([]*scanJoinOrderTuple, 0, len(tuples))
	for _, tuple := range tuples {
		key := scanJoinOrderTupleKey(spec, tuple, input.sourceKeyCols, nil)
		if scanJoinOrderKeyHasNull(key) {
			continue
		}
		start := sort.Search(len(inner), func(i int) bool {
			candidate := scanJoinOrderRecordKey(input, inner[i], input.targetKeyCols, nil)
			return compareProjectKey(candidate, key) >= 0
		})
		for position := start; position < len(inner); position++ {
			candidate := scanJoinOrderRecordKey(input, inner[position], input.targetKeyCols, nil)
			comparison := compareProjectKey(candidate, key)
			if comparison != 0 {
				break
			}
			records := make([]*scanJoinOrderRecord, len(tuple.records)+1)
			copy(records, tuple.records)
			records[len(tuple.records)] = inner[position]
			result = append(result, &scanJoinOrderTuple{records: records, order: tuple.order})
		}
	}
	return result
}

type scanJoinOrderPartition struct {
	innerStart int
	innerEnd   int
	upperKey   []scm.Scmer
	tuples     []*scanJoinOrderTuple
}

func scanJoinOrderWorkerCount(tupleCount int, innerCount int) int {
	work := tupleCount + innerCount
	if tupleCount < 256 || innerCount < 256 || work < 4096 {
		return 1
	}
	workers := runtime.GOMAXPROCS(0) / 2
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	return workers
}

// scanJoinOrderPartitions cuts only between complete duplicate-key groups.
// This is what makes it safe to execute a join partition independently: one
// SQL equi-key and its complete cross product can never straddle workers.
func scanJoinOrderPartitions(input *scanJoinOrderInput, inner []*scanJoinOrderRecord, count int) []scanJoinOrderPartition {
	if count <= 1 || len(inner) == 0 {
		return []scanJoinOrderPartition{{innerEnd: len(inner)}}
	}
	result := make([]scanJoinOrderPartition, 0, count)
	start := 0
	for part := 1; part <= count && start < len(inner); part++ {
		end := len(inner) * part / count
		if part == count || end >= len(inner) {
			end = len(inner)
		} else {
			previous := scanJoinOrderRecordKey(input, inner[end-1], input.targetKeyCols, nil)
			for end < len(inner) {
				current := scanJoinOrderRecordKey(input, inner[end], input.targetKeyCols, nil)
				if compareProjectKey(previous, current) != 0 {
					break
				}
				end++
			}
		}
		if end <= start {
			continue
		}
		upper := scanJoinOrderRecordKey(input, inner[end-1], input.targetKeyCols, nil)
		result = append(result, scanJoinOrderPartition{
			innerStart: start,
			innerEnd:   end,
			upperKey:   append([]scm.Scmer(nil), upper...),
		})
		start = end
	}
	return result
}

// scanJoinOrderShardPartitions retains an existing leading range partition on
// a single-column join key. The same pivot semantics are used by shard pruning,
// so each worker stays within the smallest available set of physical shards.
// Composite and non-leading dimensions fall back to query-local key ranges.
func scanJoinOrderShardPartitions(input *scanJoinOrderInput, inner []*scanJoinOrderRecord) ([]scanJoinOrderPartition, bool) {
	if len(input.targetKeyCols) != 1 || len(inner) == 0 {
		return nil, false
	}
	topology := input.table.activeTopology()
	if topology.mode != ShardModePartition || len(topology.dimensions) == 0 ||
		topology.dimensions[0].Column != input.targetKeyCols[0] ||
		topology.dimensions[0].NumPartitions < 2 {
		return nil, false
	}
	result := make([]scanJoinOrderPartition, 0, topology.dimensions[0].NumPartitions)
	start := 0
	for _, pivot := range topology.dimensions[0].Pivots {
		end := sort.Search(len(inner), func(i int) bool {
			key := scanJoinOrderRecordKey(input, inner[i], input.targetKeyCols, nil)
			return scm.Less(pivot, key[0])
		})
		if end > start {
			upper := scanJoinOrderRecordKey(input, inner[end-1], input.targetKeyCols, nil)
			result = append(result, scanJoinOrderPartition{
				innerStart: start,
				innerEnd:   end,
				upperKey:   append([]scm.Scmer(nil), upper...),
			})
		}
		start = end
	}
	if start < len(inner) {
		upper := scanJoinOrderRecordKey(input, inner[len(inner)-1], input.targetKeyCols, nil)
		result = append(result, scanJoinOrderPartition{
			innerStart: start,
			innerEnd:   len(inner),
			upperKey:   append([]scm.Scmer(nil), upper...),
		})
	}
	return result, len(result) > 0
}

func mergeScanJoinOrderResults(left []*scanJoinOrderTuple, right []*scanJoinOrderTuple) []*scanJoinOrderTuple {
	result := make([]*scanJoinOrderTuple, 0, len(left)+len(right))
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		if left[i].order <= right[j].order {
			result = append(result, left[i])
			i++
		} else {
			result = append(result, right[j])
			j++
		}
	}
	result = append(result, left[i:]...)
	result = append(result, right[j:]...)
	return result
}

func mergeScanJoinOrderPartitions(results [][]*scanJoinOrderTuple) []*scanJoinOrderTuple {
	for len(results) > 1 {
		next := make([][]*scanJoinOrderTuple, 0, (len(results)+1)/2)
		for i := 0; i < len(results); i += 2 {
			if i+1 == len(results) {
				next = append(next, results[i])
			} else {
				next = append(next, mergeScanJoinOrderResults(results[i], results[i+1]))
			}
		}
		results = next
	}
	if len(results) == 0 {
		return nil
	}
	for i, tuple := range results[0] {
		tuple.order = i
	}
	return results[0]
}

func joinScanOrderInput(currentTx *TxContext, spec *scanJoinOrderSpec, tuples []*scanJoinOrderTuple, inputIndex int, inner []*scanJoinOrderRecord) []*scanJoinOrderTuple {
	if len(tuples) == 0 || len(inner) == 0 {
		return nil
	}
	input := &spec.inputs[inputIndex]
	partitions, physical := scanJoinOrderShardPartitions(input, inner)
	if !physical {
		partitions = scanJoinOrderPartitions(input, inner, scanJoinOrderWorkerCount(len(tuples), len(inner)))
	}
	for _, tuple := range tuples {
		key := scanJoinOrderTupleKey(spec, tuple, input.sourceKeyCols, nil)
		if scanJoinOrderKeyHasNull(key) {
			continue
		}
		partition := sort.Search(len(partitions), func(i int) bool {
			return compareProjectKey(partitions[i].upperKey, key) >= 0
		})
		if partition < len(partitions) {
			partitions[partition].tuples = append(partitions[partition].tuples, tuple)
		}
	}
	results := make([][]*scanJoinOrderTuple, len(partitions))
	done := runFanoutTasks(currentTx, len(partitions), func(partition int, _ bool) {
		part := &partitions[partition]
		results[partition] = joinScanOrderInputRange(spec, part.tuples, inputIndex, inner[part.innerStart:part.innerEnd])
	})
	if done != nil {
		<-done
	}
	return mergeScanJoinOrderPartitions(results)
}

func lessScanJoinOrderTuple(spec *scanJoinOrderSpec, left *scanJoinOrderTuple, right *scanJoinOrderTuple) bool {
	leftValues := scanJoinOrderTupleValues(spec, left, spec.orderCols, nil)
	rightValues := scanJoinOrderTupleValues(spec, right, spec.orderCols, nil)
	for i, relation := range spec.orderDirs {
		if scm.ToBool(relation(leftValues[i], rightValues[i])) {
			return true
		}
		if scm.ToBool(relation(rightValues[i], leftValues[i])) {
			return false
		}
	}
	for i := range left.records {
		leftRecord := left.records[i]
		rightRecord := right.records[i]
		for b := range leftRecord.shard.uuid {
			if leftRecord.shard.uuid[b] != rightRecord.shard.uuid[b] {
				return leftRecord.shard.uuid[b] < rightRecord.shard.uuid[b]
			}
		}
		if leftRecord.recid != rightRecord.recid {
			return leftRecord.recid < rightRecord.recid
		}
	}
	return false
}

func reduceScanJoinOrderTuples(currentTx *TxContext, spec *scanJoinOrderSpec, tuples []*scanJoinOrderTuple, result scm.Scmer, emitted *int) (scm.Scmer, bool) {
	if len(tuples) == 0 {
		return result, false
	}
	mapProgram := scm.PrepareSerialProc(spec.mapFn)
	reduceProgram := scm.PrepareSerialProc(spec.reduceFn)
	if spec.reduceFn.IsNil() {
		reduceProgram = scm.PrepareSerialProc(scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }))
	}
	var reduceArgs [2]scm.Scmer
	reduceValue := func(acc scm.Scmer, value scm.Scmer) scm.Scmer {
		reduceArgs[0], reduceArgs[1] = acc, value
		return reduceProgram.Call(reduceArgs[:])
	}
	if spec.reduce2Fn.IsNil() {
		// Keep scan_join_order aligned with the scan callback pipeline: COUNT(*)
		// is one addition per accepted batch, not one callback pair per row.
		if mapProgram.Kind == scm.SerialProcConstant && mapProgram.Value.IsInt() && mapProgram.Value.Int() == 1 && reduceProgram.IsNative(scm.Symbol("+")) {
			reduceArgs[0] = result
			reduceArgs[1] = scm.NewInt(int64(len(tuples)))
			result = reduceProgram.Function(reduceArgs[:]...)
			*emitted += len(tuples)
			return result, true
		}
		readers := newScanJoinOrderMapReaders(currentTx, spec)
		defer readers.close()
		var args []scm.Scmer
		for _, tuple := range tuples {
			args = readers.values(tuple, args)
			result = reduceValue(result, mapProgram.Call(args))
			*emitted++
		}
		return result, true
	}

	workers := scanJoinOrderWorkerCount(len(tuples), len(tuples))
	chunkSize := (len(tuples) + workers - 1) / workers
	tasks := (len(tuples) + chunkSize - 1) / chunkSize
	partials := make([]scm.Scmer, tasks)
	done := runFanoutTasks(currentTx, tasks, func(task int, _ bool) {
		localMap := scm.PrepareSerialProc(spec.mapFn)
		localReduce := scm.PrepareSerialProc(spec.reduceFn)
		if spec.reduceFn.IsNil() {
			localReduce = scm.PrepareSerialProc(scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] }))
		}
		readers := newScanJoinOrderMapReaders(currentTx, spec)
		defer readers.close()
		partial := spec.neutral
		var localReduceArgs [2]scm.Scmer
		start := task * chunkSize
		end := start + chunkSize
		if end > len(tuples) {
			end = len(tuples)
		}
		if localMap.Kind == scm.SerialProcConstant && localMap.Value.IsInt() && localMap.Value.Int() == 1 && localReduce.IsNative(scm.Symbol("+")) {
			localReduceArgs[0] = partial
			localReduceArgs[1] = scm.NewInt(int64(end - start))
			partials[task] = localReduce.Function(localReduceArgs[:]...)
			return
		}
		var args []scm.Scmer
		for i := start; i < end; i++ {
			args = readers.values(tuples[i], args)
			localReduceArgs[0] = partial
			localReduceArgs[1] = localMap.Call(args)
			partial = localReduce.Call(localReduceArgs[:])
		}
		partials[task] = partial
	})
	if done != nil {
		<-done
	}
	reduce2Program := scm.PrepareSerialProc(spec.reduce2Fn)
	var reduce2Args [2]scm.Scmer
	for _, partial := range partials {
		reduce2Args[0], reduce2Args[1] = result, partial
		result = reduce2Program.Call(reduce2Args[:])
	}
	*emitted += len(tuples)
	return result, true
}

func scanJoinOrderBatch(currentTx *TxContext, spec *scanJoinOrderSpec, driverRefs []orderedBatchRecord, innerRecords [][]*scanJoinOrderRecord) []*scanJoinOrderTuple {
	driver := materializeScanJoinRecords(currentTx, &spec.inputs[0], driverRefs)
	tuples := make([]*scanJoinOrderTuple, len(driver))
	for i, record := range driver {
		tuples[i] = &scanJoinOrderTuple{records: []*scanJoinOrderRecord{record}, order: i}
	}
	for inputIndex := 1; inputIndex < len(spec.inputs) && len(tuples) > 0; inputIndex++ {
		tuples = joinScanOrderInput(currentTx, spec, tuples, inputIndex, innerRecords[inputIndex])
	}
	if spec.joinFilter.IsNil() || len(tuples) == 0 {
		return tuples
	}
	filterProgram := scm.PrepareSerialProc(spec.joinFilter)
	if filterProgram.Kind == scm.SerialProcConstant {
		if scm.ToBool(filterProgram.Value) {
			return tuples
		}
		return tuples[:0]
	}
	filtered := tuples[:0]
	var args []scm.Scmer
	for _, tuple := range tuples {
		args = scanJoinOrderTupleValues(spec, tuple, spec.joinFilterCols, args)
		if scm.ToBool(filterProgram.Call(args)) {
			filtered = append(filtered, tuple)
		}
	}
	return filtered
}

func scanJoinOrderMaterialized(currentTx *TxContext, spec scanJoinOrderSpec) scm.Scmer {
	prepareScanJoinOrderSpec(&spec)
	if spec.limit == 0 {
		return spec.notFoundValue
	}

	innerRecords := make([][]*scanJoinOrderRecord, len(spec.inputs))
	for inputIndex := 1; inputIndex < len(spec.inputs); inputIndex++ {
		input := &spec.inputs[inputIndex]
		sortcols := make([]scm.Scmer, len(input.targetKeyCols))
		sortdirs := make([]func(...scm.Scmer) scm.Scmer, len(input.targetKeyCols))
		for i, col := range input.targetKeyCols {
			sortcols[i] = scm.NewString(col)
			sortdirs[i] = func(values ...scm.Scmer) scm.Scmer {
				return scm.NewBool(scm.Less(values[0], values[1]))
			}
		}
		refs := collectScanJoinOrderRecords(currentTx, input, sortcols, sortdirs, 0, -1, nil, scm.NewNil())
		innerRecords[inputIndex] = materializeScanJoinRecords(currentTx, input, refs)
	}

	driverOrdered := scanJoinOrderUsesDriverOrder(&spec)
	driverInput := spec.inputs[0]
	var driverAcceptCols []string
	driverAccept := scm.NewNil()
	if len(spec.inputs) > 1 {
		driverInput, driverAcceptCols, driverAccept = scanJoinOrderCandidateDriver(&spec, innerRecords)
	}
	driverSortCols := make([]scm.Scmer, 0, len(spec.orderCols))
	if driverOrdered {
		for _, ref := range spec.orderCols {
			driverSortCols = append(driverSortCols, scm.NewString(ref.column))
		}
	}
	result := spec.neutral
	driverOffset := 0
	accepted := 0
	emitted := 0
	hadValue := false
	target := spec.offset + spec.limit
	batchSize := target
	if spec.limit < 0 || !driverOrdered {
		batchSize = -1
		target = int(^uint(0) >> 1)
	} else if batchSize < 1 {
		batchSize = 1
	}
	for accepted < target {
		driverRefs := collectScanJoinOrderRecords(currentTx, &driverInput, driverSortCols, spec.orderDirs, driverOffset, batchSize, driverAcceptCols, driverAccept)
		if len(driverRefs) == 0 {
			break
		}
		tuples := scanJoinOrderBatch(currentTx, &spec, driverRefs, innerRecords)
		if !driverOrdered {
			sort.SliceStable(tuples, func(i int, j int) bool {
				return lessScanJoinOrderTuple(&spec, tuples[i], tuples[j])
			})
		}
		first := 0
		if accepted < spec.offset {
			first = spec.offset - accepted
			if first > len(tuples) {
				first = len(tuples)
			}
		}
		accepted += len(tuples)
		last := len(tuples)
		if spec.limit >= 0 && last-first > spec.limit-emitted {
			last = first + spec.limit - emitted
		}
		var batchHadValue bool
		result, batchHadValue = reduceScanJoinOrderTuples(currentTx, &spec, tuples[first:last], result, &emitted)
		hadValue = hadValue || batchHadValue
		if spec.limit >= 0 && emitted >= spec.limit {
			break
		}
		driverOffset += len(driverRefs)
		if batchSize < 0 || len(driverRefs) < batchSize {
			break
		}
		batchSize *= 2
	}
	if !hadValue && spec.isOuter {
		return scanJoinOrderOuterResult(&spec, result)
	}
	if !hadValue {
		return spec.notFoundValue
	}
	return result
}

func scanJoinOrderStreamRanges(input *scanJoinOrderInput, shard *storageShard) map[string]scanJoinOrderShardRange {
	topology := input.table.activeTopology()
	if topology.mode != ShardModePartition || len(topology.dimensions) == 0 {
		return nil
	}
	shardIndex := -1
	for i, candidate := range topology.shards {
		if candidate == shard {
			shardIndex = i
			break
		}
	}
	if shardIndex < 0 {
		return nil
	}
	result := make(map[string]scanJoinOrderShardRange, len(topology.dimensions))
	stride := len(topology.shards)
	for _, dimension := range topology.dimensions {
		stride /= dimension.NumPartitions
		partition := (shardIndex / stride) % dimension.NumPartitions
		valueRange := scanJoinOrderShardRange{}
		if partition > 0 {
			valueRange.lower = dimension.Pivots[partition-1]
			valueRange.hasLower = true
			valueRange.lowerExclusive = true
		}
		if partition < len(dimension.Pivots) {
			valueRange.upper = dimension.Pivots[partition]
			valueRange.hasUpper = true
		}
		result[dimension.Column] = valueRange
	}
	return result
}

func scanJoinOrderRangesOverlap(left scanJoinOrderShardRange, right scanJoinOrderShardRange) bool {
	if left.hasUpper && right.hasLower {
		if scm.Less(left.upper, right.lower) ||
			(!scm.Less(left.upper, right.lower) && !scm.Less(right.lower, left.upper) && right.lowerExclusive) {
			return false
		}
	}
	if right.hasUpper && left.hasLower {
		if scm.Less(right.upper, left.lower) ||
			(!scm.Less(right.upper, left.lower) && !scm.Less(left.lower, right.upper) && left.lowerExclusive) {
			return false
		}
	}
	return true
}

func collectScanJoinOrderShardStreams(currentTx *TxContext, input *scanJoinOrderInput, sortcols []scm.Scmer, sortdirs []func(...scm.Scmer) scm.Scmer) []*scanJoinOrderShardStream {
	bounds := extractBoundaries(input.filterCols, input.filter)
	reorderByFrequency(bounds, input.table)
	bounds, _ = extendBoundariesWithSortCols(bounds, sortcols, sortdirs)
	lower, upperLast := indexFromBoundaries(bounds)
	for _, boundary := range bounds {
		input.table.AddPartitioningScore([]string{boundary.col})
	}

	values := make(chan *scanJoinOrderShardStream, input.table.shardResultBufferSize())
	ss := SessionStateFromTx(currentTx)
	done := input.table.iterateShardsParallel(currentTx, bounds, func(shard *storageShard, _ bool) {
		queue := shard.scan_order(bounds, lower, upperLast, input.filterCols, input.filter,
			nil, scm.NewNil(),
			sortcols, sortdirs, 0, 0, -1, input.readCols, currentTx, ss)
		refs := make([]orderedBatchRecord, len(queue.items))
		for i, recid := range queue.items {
			refs[i] = orderedBatchRecord{shard: shard, recid: recid}
		}
		values <- &scanJoinOrderShardStream{
			shard:   shard,
			records: materializeScanJoinRecords(currentTx, input, refs),
			ranges:  scanJoinOrderStreamRanges(input, shard),
		}
	})
	if done != nil {
		<-done
	}
	close(values)
	result := make([]*scanJoinOrderShardStream, 0)
	for stream := range values {
		if len(stream.records) > 0 {
			result = append(result, stream)
		}
	}
	return result
}

func scanJoinOrderStreamsCompatible(spec *scanJoinOrderSpec, selected []*scanJoinOrderShardStream, inputIndex int, candidate *scanJoinOrderShardStream) bool {
	input := &spec.inputs[inputIndex]
	for keyIndex, source := range input.sourceKeyCols {
		sourceRange, sourcePartitioned := selected[source.table].ranges[source.column]
		targetRange, targetPartitioned := candidate.ranges[input.targetKeyCols[keyIndex]]
		if sourcePartitioned && targetPartitioned && !scanJoinOrderRangesOverlap(sourceRange, targetRange) {
			return false
		}
	}
	return true
}

func scanJoinOrderShardCombinations(spec *scanJoinOrderSpec, streams [][]*scanJoinOrderShardStream) [][]*scanJoinOrderShardStream {
	result := make([][]*scanJoinOrderShardStream, 0)
	selected := make([]*scanJoinOrderShardStream, len(streams))
	var visit func(int)
	visit = func(inputIndex int) {
		if inputIndex == len(streams) {
			combination := append([]*scanJoinOrderShardStream(nil), selected...)
			result = append(result, combination)
			return
		}
		for _, candidate := range streams[inputIndex] {
			if inputIndex > 0 && !scanJoinOrderStreamsCompatible(spec, selected, inputIndex, candidate) {
				continue
			}
			selected[inputIndex] = candidate
			visit(inputIndex + 1)
		}
	}
	visit(0)
	return result
}

func sortAndCapScanJoinOrderTuples(spec *scanJoinOrderSpec, tuples []*scanJoinOrderTuple, keep int) []*scanJoinOrderTuple {
	sort.SliceStable(tuples, func(i int, j int) bool {
		return lessScanJoinOrderTuple(spec, tuples[i], tuples[j])
	})
	if keep >= 0 && len(tuples) > keep {
		tuples = tuples[:keep]
	}
	return tuples
}

func runScanJoinOrderShardCombination(currentTx *TxContext, spec *scanJoinOrderSpec, combination []*scanJoinOrderShardStream, keep int) []*scanJoinOrderTuple {
	_ = currentTx
	collector := &scanJoinOrderTopKCollector{spec: spec, keep: keep}
	var filterProgram *scm.SerialProc
	if !spec.joinFilter.IsNil() {
		prepared := scm.PrepareSerialProc(spec.joinFilter)
		if prepared.Kind == scm.SerialProcConstant {
			if !scm.ToBool(prepared.Value) {
				return nil
			}
		} else {
			filterProgram = &prepared
		}
	}
	records := make([]*scanJoinOrderRecord, len(combination))
	var enumerate func(int)
	enumerate = func(inputIndex int) {
		if inputIndex == len(combination) {
			tuple := &scanJoinOrderTuple{records: append([]*scanJoinOrderRecord(nil), records...)}
			if filterProgram != nil {
				args := scanJoinOrderTupleValues(spec, tuple, spec.joinFilterCols, nil)
				if !scm.ToBool(filterProgram.Call(args)) {
					return
				}
			}
			collector.add(tuple)
			return
		}
		input := &spec.inputs[inputIndex]
		partial := &scanJoinOrderTuple{records: records[:inputIndex]}
		key := scanJoinOrderTupleKey(spec, partial, input.sourceKeyCols, nil)
		if scanJoinOrderKeyHasNull(key) {
			return
		}
		inner := combination[inputIndex].records
		start := sort.Search(len(inner), func(i int) bool {
			candidate := scanJoinOrderRecordKey(input, inner[i], input.targetKeyCols, nil)
			return compareProjectKey(candidate, key) >= 0
		})
		for position := start; position < len(inner); position++ {
			candidate := scanJoinOrderRecordKey(input, inner[position], input.targetKeyCols, nil)
			if compareProjectKey(candidate, key) != 0 {
				break
			}
			records[inputIndex] = inner[position]
			enumerate(inputIndex + 1)
		}
	}
	driverOrdered := scanJoinOrderUsesDriverOrder(spec)
	for _, driverRecord := range combination[0].records {
		records[0] = driverRecord
		enumerate(1)
		// All future driver records sort no better than this completed group.
		// Inner duplicates for the current driver have already been exhausted.
		if driverOrdered && keep >= 0 && collector.Len() >= keep {
			break
		}
	}
	return collector.sorted()
}

func reduceScanJoinOrderShardCombination(currentTx *TxContext, spec *scanJoinOrderSpec, combination []*scanJoinOrderShardStream) (scm.Scmer, int) {
	result := spec.neutral
	count := 0
	mapProgram := scm.PrepareSerialProc(spec.mapFn)
	reduce := spec.reduceFn
	if reduce.IsNil() {
		reduce = scm.NewFunc(func(values ...scm.Scmer) scm.Scmer { return values[1] })
	}
	reduceProgram := scm.PrepareSerialProc(reduce)
	countPipeline := mapProgram.Kind == scm.SerialProcConstant && mapProgram.Value.IsInt() && mapProgram.Value.Int() == 1 && reduceProgram.IsNative(scm.Symbol("+"))
	var reduceArgs [2]scm.Scmer
	readers := newScanJoinOrderMapReaders(currentTx, spec)
	defer readers.close()
	var filterProgram *scm.SerialProc
	if !spec.joinFilter.IsNil() {
		prepared := scm.PrepareSerialProc(spec.joinFilter)
		if prepared.Kind == scm.SerialProcConstant {
			if !scm.ToBool(prepared.Value) {
				return result, 0
			}
		} else {
			filterProgram = &prepared
		}
	}
	records := make([]*scanJoinOrderRecord, len(combination))
	var enumerate func(int)
	enumerate = func(inputIndex int) {
		if inputIndex == len(combination) {
			tuple := &scanJoinOrderTuple{records: records}
			if filterProgram != nil {
				filterArgs := scanJoinOrderTupleValues(spec, tuple, spec.joinFilterCols, nil)
				if !scm.ToBool(filterProgram.Call(filterArgs)) {
					return
				}
			}
			if countPipeline {
				count++
				return
			}
			mapArgs := readers.values(tuple, nil)
			reduceArgs[0] = result
			reduceArgs[1] = mapProgram.Call(mapArgs)
			result = reduceProgram.Call(reduceArgs[:])
			count++
			return
		}
		input := &spec.inputs[inputIndex]
		partial := &scanJoinOrderTuple{records: records[:inputIndex]}
		key := scanJoinOrderTupleKey(spec, partial, input.sourceKeyCols, nil)
		if scanJoinOrderKeyHasNull(key) {
			return
		}
		inner := combination[inputIndex].records
		start := sort.Search(len(inner), func(i int) bool {
			candidate := scanJoinOrderRecordKey(input, inner[i], input.targetKeyCols, nil)
			return compareProjectKey(candidate, key) >= 0
		})
		for position := start; position < len(inner); position++ {
			candidate := scanJoinOrderRecordKey(input, inner[position], input.targetKeyCols, nil)
			if compareProjectKey(candidate, key) != 0 {
				break
			}
			records[inputIndex] = inner[position]
			enumerate(inputIndex + 1)
		}
	}
	for _, driverRecord := range combination[0].records {
		records[0] = driverRecord
		enumerate(1)
	}
	if countPipeline && count > 0 {
		reduceArgs[0] = result
		reduceArgs[1] = scm.NewInt(int64(count))
		result = reduceProgram.Function(reduceArgs[:]...)
	}
	return result, count
}

func mergeTopKScanJoinOrderTuples(spec *scanJoinOrderSpec, left []*scanJoinOrderTuple, right []*scanJoinOrderTuple, keep int) []*scanJoinOrderTuple {
	capacity := len(left) + len(right)
	if keep >= 0 && capacity > keep {
		capacity = keep
	}
	result := make([]*scanJoinOrderTuple, 0, capacity)
	i, j := 0, 0
	for (keep < 0 || len(result) < keep) && i < len(left) && j < len(right) {
		if lessScanJoinOrderTuple(spec, right[j], left[i]) {
			result = append(result, right[j])
			j++
		} else {
			result = append(result, left[i])
			i++
		}
	}
	for (keep < 0 || len(result) < keep) && i < len(left) {
		result = append(result, left[i])
		i++
	}
	for (keep < 0 || len(result) < keep) && j < len(right) {
		result = append(result, right[j])
		j++
	}
	return result
}

func collectTopKScanJoinOrderTuples(spec *scanJoinOrderSpec, runnerResults [][]*scanJoinOrderTuple, keep int) []*scanJoinOrderTuple {
	for len(runnerResults) > 1 {
		next := make([][]*scanJoinOrderTuple, 0, (len(runnerResults)+1)/2)
		for i := 0; i < len(runnerResults); i += 2 {
			if i+1 == len(runnerResults) {
				next = append(next, runnerResults[i])
			} else {
				next = append(next, mergeTopKScanJoinOrderTuples(spec, runnerResults[i], runnerResults[i+1], keep))
			}
		}
		runnerResults = next
	}
	if len(runnerResults) == 0 {
		return nil
	}
	return runnerResults[0]
}

func scanJoinOrder(currentTx *TxContext, spec scanJoinOrderSpec) scm.Scmer {
	if spec.limit >= 0 && scanJoinOrderUsesDriverOrder(&spec) {
		return scanJoinOrderMaterialized(currentTx, spec)
	}
	prepareScanJoinOrderSpec(&spec)
	if spec.limit == 0 {
		return spec.notFoundValue
	}
	keep := -1
	if spec.limit >= 0 {
		if spec.offset > int(^uint(0)>>1)-spec.limit {
			panic("scan_join_order: offset plus limit overflows")
		}
		keep = spec.offset + spec.limit
	}

	streams := make([][]*scanJoinOrderShardStream, len(spec.inputs))
	for inputIndex := range spec.inputs {
		input := &spec.inputs[inputIndex]
		var sortcols []scm.Scmer
		var sortdirs []func(...scm.Scmer) scm.Scmer
		if inputIndex == 0 && scanJoinOrderUsesDriverOrder(&spec) {
			sortcols = make([]scm.Scmer, len(spec.orderCols))
			for i, ref := range spec.orderCols {
				sortcols[i] = scm.NewString(ref.column)
			}
			sortdirs = spec.orderDirs
		} else if inputIndex > 0 {
			sortcols = make([]scm.Scmer, len(input.targetKeyCols))
			sortdirs = make([]func(...scm.Scmer) scm.Scmer, len(input.targetKeyCols))
			for i, col := range input.targetKeyCols {
				sortcols[i] = scm.NewString(col)
				sortdirs[i] = func(values ...scm.Scmer) scm.Scmer {
					return scm.NewBool(scm.Less(values[0], values[1]))
				}
			}
		}
		streams[inputIndex] = collectScanJoinOrderShardStreams(currentTx, input, sortcols, sortdirs)
		if len(streams[inputIndex]) == 0 {
			return spec.notFoundValue
		}
	}

	combinations := scanJoinOrderShardCombinations(&spec, streams)
	if !spec.reduce2Fn.IsNil() {
		partials := make([]scm.Scmer, len(combinations))
		counts := make([]int, len(combinations))
		done := runFanoutTasks(currentTx, len(combinations), func(runner int, _ bool) {
			partials[runner], counts[runner] = reduceScanJoinOrderShardCombination(currentTx, &spec, combinations[runner])
		})
		if done != nil {
			<-done
		}
		reduce2Program := scm.PrepareSerialProc(spec.reduce2Fn)
		var reduce2Args [2]scm.Scmer
		result := spec.neutral
		total := 0
		for runner, partial := range partials {
			if counts[runner] == 0 {
				continue
			}
			reduce2Args[0], reduce2Args[1] = result, partial
			result = reduce2Program.Call(reduce2Args[:])
			total += counts[runner]
		}
		if total > 0 {
			return result
		}
		if spec.isOuter {
			return scanJoinOrderOuterResult(&spec, spec.neutral)
		}
		return spec.notFoundValue
	}
	runnerResults := make([][]*scanJoinOrderTuple, len(combinations))
	done := runFanoutTasks(currentTx, len(combinations), func(runner int, _ bool) {
		runnerResults[runner] = runScanJoinOrderShardCombination(currentTx, &spec, combinations[runner], keep)
	})
	if done != nil {
		<-done
	}
	joined := collectTopKScanJoinOrderTuples(&spec, runnerResults, keep)
	if spec.offset > len(joined) {
		joined = nil
	} else {
		joined = joined[spec.offset:]
	}
	if spec.limit >= 0 && len(joined) > spec.limit {
		joined = joined[:spec.limit]
	}
	if len(joined) == 0 {
		if spec.isOuter {
			return scanJoinOrderOuterResult(&spec, spec.neutral)
		}
		return spec.notFoundValue
	}
	emitted := 0
	result, _ := reduceScanJoinOrderTuples(currentTx, &spec, joined, spec.neutral, &emitted)
	return result
}

func decodeScanJoinOrderColumn(value scm.Scmer, label string) scanJoinOrderColumn {
	items := mustScmerSlice(value, label)
	if len(items) != 2 {
		panic("scan_join_order: a joined column reference must be (table_index column_name)")
	}
	return scanJoinOrderColumn{table: int(scm.ToInt(items[0])), column: scm.String(items[1])}
}

func decodeScanJoinOrderColumns(value scm.Scmer, label string) []scanJoinOrderColumn {
	items := mustScmerSlice(value, label)
	result := make([]scanJoinOrderColumn, len(items))
	for i, item := range items {
		result[i] = decodeScanJoinOrderColumn(item, label)
	}
	return result
}

func decodeScanJoinOrderInputs(tables []scm.Scmer, filterColumns scm.Scmer, filterFns scm.Scmer, joins scm.Scmer) []scanJoinOrderInput {
	filterCols := mustScmerSlice(filterColumns, "filterColumns")
	filters := mustScmerSlice(filterFns, "filterFns")
	joinItems := mustScmerSlice(joins, "joins")
	if len(filterCols) != len(tables) || len(filters) != len(tables) || len(joinItems) != len(tables)-1 {
		panic("scan_join_order: filter arrays must match tables and joins must have tables-1 entries")
	}
	result := make([]scanJoinOrderInput, len(tables))
	for i, tableValue := range tables {
		result[i] = scanJoinOrderInput{
			table:      TableFromScmer(tableValue),
			filterCols: scmerSliceToStrings(mustScmerSlice(filterCols[i], "filterColumns[i]")),
			filter:     filters[i],
		}
		if i == 0 {
			continue
		}
		clauses := mustScmerSlice(joinItems[i-1], "joins[i]")
		result[i].sourceKeyCols = make([]scanJoinOrderColumn, len(clauses))
		result[i].targetKeyCols = make([]string, len(clauses))
		for clauseIndex, clauseValue := range clauses {
			clause := mustScmerSlice(clauseValue, "join clause")
			if len(clause) != 3 {
				panic("scan_join_order: a join clause must be (outer_table_index outer_column inner_column)")
			}
			result[i].sourceKeyCols[clauseIndex] = scanJoinOrderColumn{
				table:  int(scm.ToInt(clause[0])),
				column: scm.String(clause[1]),
			}
			result[i].targetKeyCols[clauseIndex] = scm.String(clause[2])
		}
	}
	return result
}

func declareScanJoinOrder(en *scm.Env) {
	joinedColumn := &scm.TypeDescriptor{
		Kind:        "list",
		Label:       "joined column",
		Description: "pair (zero-based table index, physical column name)",
	}
	scm.Declare(en, &scm.Declaration{
		Name: "scan_join_order",
		Fn: func(a ...scm.Scmer) scm.Scmer {
			tables := mustScmerSlice(a[1], "tables")
			sortDirections := mustScmerSlice(a[8], "sortDirections")
			orderDirs := make([]func(...scm.Scmer) scm.Scmer, len(sortDirections))
			for i, direction := range sortDirections {
				orderDirs[i] = scm.OptimizeProcToSerialFunction(direction)
			}
			neutral := scm.NewNil()
			if len(a) > 15 {
				neutral = a[15]
			}
			reduce2 := scm.NewNil()
			if len(a) > 16 {
				reduce2 = a[16]
			}
			isOuter := len(a) > 17 && scm.ToBool(a[17])
			notFoundValue := neutral
			if len(a) > 18 {
				notFoundValue = a[18]
			}
			return scanJoinOrder(scmerToTxContext(a[0]), scanJoinOrderSpec{
				inputs:             decodeScanJoinOrderInputs(tables, a[2], a[3], a[4]),
				joinFilterCols:     decodeScanJoinOrderColumns(a[5], "joinFilterColumns"),
				joinFilter:         a[6],
				orderCols:          decodeScanJoinOrderColumns(a[7], "orderColumns"),
				orderDirs:          orderDirs,
				limitPartitionCols: int(scm.ToInt(a[9])),
				offset:             int(scm.ToInt(a[10])),
				limit:              int(scm.ToInt(a[11])),
				mapCols:            decodeScanJoinOrderColumns(a[12], "mapColumns"),
				mapFn:              a[13],
				reduceFn:           a[14],
				reduce2Fn:          reduce2,
				neutral:            neutral,
				isOuter:            isOuter,
				notFoundValue:      notFoundValue,
			})
		},
		Type: &scm.TypeDescriptor{
			Kind:           "func",
			Description:    "scans a left-deep equi-join in final ORDER BY order, executes each inner table through equality-filter plus join-key ordered access, and applies OFFSET/LIMIT only to joined rows",
			HasSideEffects: true,
			Params: []*scm.TypeDescriptor{
				{Kind: "any", Label: "tx", Description: "transaction context"},
				{Kind: "list", Label: "tables", Description: "driver table followed by inner equi-join tables"},
				{Kind: "list", Label: "filterColumns", Description: "one physical filter-column list per table"},
				{Kind: "list", Label: "filterFns", Description: "one table-local filter callback per table"},
				{Kind: "list", Label: "joins", Description: "one join description per inner table; every clause is (outer_table_index outer_column inner_column)"},
				{Kind: "list", Label: "joinFilterColumns", Description: "joined columns supplied to the residual join filter", Element: joinedColumn},
				{Kind: "func|nil", Label: "joinFilter", Description: "residual predicate over the complete joined tuple, or nil"},
				{Kind: "list", Label: "orderColumns", Description: "joined column references used by final ORDER BY", Element: joinedColumn},
				{Kind: "list", Label: "sortDirections", Description: "one ordering relation per order column"},
				{Kind: "int", Label: "limitPartitionCols", Description: "reserved for scan_order compatibility; currently must be 0"},
				{Kind: "int", Label: "offset", Description: "joined rows skipped in final order"},
				{Kind: "int", Label: "limit", Description: "joined rows emitted in final order; -1 is unlimited"},
				{Kind: "list", Label: "mapColumns", Description: "joined columns supplied to the sole outer map callback", Element: joinedColumn},
				{Kind: "func", Label: "map", Description: "map callback evaluated only for final OFFSET/LIMIT rows"},
				{Kind: "func|nil", Label: "reduce", Description: "global reducer; when reduce2 is supplied, this becomes the runner-local first-stage reducer", Optional: true},
				{Kind: "any", Label: "neutral", Description: "optional reducer neutral value", Optional: true},
				{Kind: "func|nil", Label: "reduce2", Description: "optional global second-stage reducer combining runner-local reduce results; permitted only with offset 0 and unlimited limit -1", Optional: true},
				{Kind: "bool", Label: "isOuter", Description: "optional whole-scan NULL fallback", Optional: true},
				{Kind: "any", Label: "notFoundValue", Description: "optional result when no joined row is emitted", Optional: true},
			},
			Return:   &scm.TypeDescriptor{Kind: "any"},
			Optimize: optimizeScanJoinOrder,
		},
	})
}
