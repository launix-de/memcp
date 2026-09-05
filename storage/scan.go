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
import "sort"
import "strings"
import "sync"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"

type scanError struct {
	r     interface{}
	stack string
}

const scanAnalyzeScratchCapacity = 8

const scanAccessSchemaHeaderSize = 1
const scanAccessBoundaryStride = 4

const scanAccessConsumerScan = "scan"
const scanAccessConsumerCoveredScan = "scan_covered"

const scanAccessHeaderMagic = 0x15 << 44

type scanAccessSchemaMeta struct {
	count, projections, mapperSlot int
	consumer                       string
}

type scanAccessBoundaryMeta struct {
	lowerSlot, upperSlot int
	flags                int64
}

func newScanAccessBoundaryMeta(lowerSlot, upperSlot int, flags int64) scm.Scmer {
	if lowerSlot < -1<<15 || lowerSlot >= 1<<15 || upperSlot < -1<<15 || upperSlot >= 1<<15 || flags < 0 || flags >= 1<<16 {
		panic("scan access boundary metadata exceeds packed limits")
	}
	return scm.NewInt(int64(lowerSlot+1<<15) | int64(upperSlot+1<<15)<<16 | flags<<32)
}

func decodeScanAccessBoundaryMeta(value scm.Scmer) scanAccessBoundaryMeta {
	raw := scm.ToInt(value)
	return scanAccessBoundaryMeta{
		lowerSlot: int(raw&0xffff) - 1<<15,
		upperSlot: int(raw>>16&0xffff) - 1<<15,
		flags:     int64(raw >> 32 & 0xffff),
	}
}

func newScanAccessHeader(count int, consumer string, projections int, mapperSlot int) scm.Scmer {
	consumerID := int64(0)
	switch consumer {
	case scanAccessConsumerScan:
	case scanAccessConsumerCoveredScan:
		consumerID = 1
	case "exists":
		consumerID = 2
	case "value":
		consumerID = 3
	case "map":
		consumerID = 4
	default:
		panic("unknown scan access consumer " + consumer)
	}
	if count < 0 || count >= 1<<12 || projections < 0 || projections >= 1<<12 || mapperSlot < -1 || mapperSlot >= 1<<12-1 {
		panic("scan access header exceeds packed limits")
	}
	return scm.NewInt(int64(scanAccessHeaderMagic) | consumerID<<40 | int64(count)<<28 | int64(projections)<<16 | int64(mapperSlot+1))
}

func decodeScanAccessHeader(value scm.Scmer) (scanAccessSchemaMeta, bool) {
	raw := scm.ToInt(value)
	if raw&(0xff<<44) != scanAccessHeaderMagic {
		return scanAccessSchemaMeta{}, false
	}
	consumer := scanAccessConsumerScan
	switch raw >> 40 & 0xf {
	case 0:
	case 1:
		consumer = scanAccessConsumerCoveredScan
	case 2:
		consumer = "exists"
	case 3:
		consumer = "value"
	case 4:
		consumer = "map"
	default:
		return scanAccessSchemaMeta{}, false
	}
	return scanAccessSchemaMeta{count: int(raw >> 28 & 0xfff), projections: int(raw >> 16 & 0xfff), mapperSlot: int(raw&0xfff) - 1, consumer: consumer}, true
}

var emptyScanAccessSchema = newScanAccessSchema(scanAccessConsumerScan, nil, -1)

// scanAnalyzeScratch owns the short-lived physical analyzer output until all
// parallel shard consumers have completed. A pool is preferable to a caller
// stack array here: the shard callback escapes into goroutines, which would
// force the complete stack array onto the heap on every scan. Most SQL scans
// fit in these inline buffers; unusual wide predicates retain the ordinary
// append fallback without changing analyzer semantics.
type scanAnalyzeScratch struct {
	runtime         scanAccessRuntime
	batchBoundaries [scanAnalyzeScratchCapacity]columnboundaries
	lower           [scanAnalyzeScratchCapacity]scm.Scmer
}

var scanAnalyzeScratchPool = sync.Pool{
	New: func() any { return new(scanAnalyzeScratch) },
}

func acquireScanAnalyzeScratch() *scanAnalyzeScratch {
	return scanAnalyzeScratchPool.Get().(*scanAnalyzeScratch)
}

func releaseScanAnalyzeScratch(scratch *scanAnalyzeScratch) {
	scratch.runtime = scanAccessRuntime{}
	clear(scratch.batchBoundaries[:])
	clear(scratch.lower[:])
	scanAnalyzeScratchPool.Put(scratch)
}

func (s scanError) Error() string {
	return fmt.Sprint(s.r)
}

func buildOuterNullCallbackRow(callbackCols []string) []scm.Scmer {
	return make([]scm.Scmer, len(callbackCols))
}

/* TODO: interface Scannable (scan + scan_order) and (table schema tbl) to get a scannable */

// optimizeScanShared propagates the accumulator type through the combined
// (accumulator, column...) callback and then through the shard combiner.
func optimizeScanShared(v []scm.Scmer, oc *scm.OptimizerContext, mapReduceIdx, neutralIdx, combineIdx, outerIdx int) (scm.Scmer, *scm.TypeDescriptor) {
	rawMapReduce := v[mapReduceIdx]
	rawCombine := scm.NewNil()
	if len(v) > combineIdx {
		rawCombine = v[combineIdx]
	}

	// Optimize scalar/operator arguments independently of callback ownership.
	for i := 1; i <= mapReduceIdx && i < len(v); i++ {
		if i != mapReduceIdx {
			v[i], _ = oc.OptimizeSub(v[i], true)
		}
	}
	neutralType := unknownScanType()
	if len(v) > neutralIdx {
		v[neutralIdx], neutralType = oc.OptimizeSub(v[neutralIdx], true)
		neutralType = normalizeScanType(neutralType)
	}
	oc.Ome.IncrLoopDepth()
	columnTypes := []*scm.TypeDescriptor(nil)
	if params, _, ok := scanLambdaParts(rawMapReduce); ok && len(params) > 1 {
		columnTypes = make([]*scm.TypeDescriptor, len(params)-1)
		for i := range columnTypes {
			columnTypes[i] = unknownScanType()
		}
	}
	optimizedMapReduce, reduceType := oc.OptimizeReducerCallback(rawMapReduce, neutralType, columnTypes...)
	v[mapReduceIdx] = optimizedMapReduce
	if !rawCombine.IsNil() {
		v[combineIdx], reduceType = oc.OptimizeReducerCallback(rawCombine, reduceType, reduceType)
	}
	if len(v) > outerIdx {
		v[outerIdx], _ = oc.OptimizeSub(v[outerIdx], true)
	}
	oc.Ome.DecrLoopDepth()
	return scm.NewSlice(v), reduceType
}

func unknownScanType() *scm.TypeDescriptor {
	return &scm.TypeDescriptor{Kind: "any", Length: scm.UnknownLength}
}

func normalizeScanType(td *scm.TypeDescriptor) *scm.TypeDescriptor {
	if td == nil {
		return unknownScanType()
	}
	return td
}

func optimizeScan(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	if rewritten := tryScanInvariantFilterRewrite(v); !rewritten.IsNil() {
		if result, td, accepted := oc.OptimizeRewrite(scm.NewSlice(v), rewritten, useResult, scm.OptimizerRewriteContract{
			Name: "scan-invariant-filter", PreconditionsMet: true, MaxGrowthNodes: 64,
		}); accepted {
			return result, td
		}
	}
	if rewritten := tryScanExistsRewrite(v); !rewritten.IsNil() {
		if result, td, accepted := oc.OptimizeRewrite(scm.NewSlice(v), rewritten, useResult, scm.OptimizerRewriteContract{
			Name: "scan-exists", PreconditionsMet: true, MaxGrowthNodes: 64,
		}); accepted {
			return result, td
		}
	}
	if rewritten := tryScanBatchRewrite(v); !rewritten.IsNil() {
		if result, td, accepted := oc.OptimizeRewrite(scm.NewSlice(v), rewritten, useResult, scm.OptimizerRewriteContract{
			Name: "scan-batch", PreconditionsMet: true, MaxGrowthNodes: 256,
		}); accepted {
			return result, td
		}
	}
	return optimizeScanShared(v, oc, 8, 9, 10, 11)
}

func optimizeScanExists(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	return oc.ApplyDefaultOptimization(v, useResult)
}

func optimizeScanSelectivity(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	return oc.ApplyDefaultOptimization(v, useResult)
}

func scanStaticListElements(expr scm.Scmer) ([]scm.Scmer, bool) {
	items, ok := scmerSlice(expr)
	if !ok {
		return nil, false
	}
	if len(items) == 2 && scanSymbolIs(items[0], "quote") {
		quoted, quotedOK := scmerSlice(items[1])
		if !quotedOK {
			return nil, false
		}
		return quoted, true
	} else if len(items) > 0 && scanSymbolIs(items[0], "list") {
		return items[1:], true
	}
	return items, true
}

func scanStaticColumns(expr scm.Scmer) ([]scm.Scmer, bool) {
	items, ok := scanStaticListElements(expr)
	if !ok {
		return nil, false
	}
	columns := make([]scm.Scmer, len(items))
	for i, item := range items {
		item = item.WithoutSourceInfo()
		if !item.IsString() {
			return nil, false
		}
		columns[i] = item
	}
	return columns, true
}

func scanAccessValuesExpr(values []scm.Scmer) scm.Scmer {
	if len(values) == 0 {
		// Empty slices are self-evaluating in SCM. Keeping the canonical empty
		// value vector literal avoids two quote nodes in every access-free scan.
		return scm.NewSlice(nil)
	}
	staticValues := make([]scm.Scmer, len(values))
	for i, value := range values {
		value = value.WithoutSourceInfo()
		if value.IsSymbol() {
			return scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("list")}, values...))
		}
		if value.IsSlice() {
			items := value.Slice()
			if len(items) != 2 || !scanSymbolIs(items[0], "quote") {
				return scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("list")}, values...))
			}
			value = items[1]
		}
		staticValues[i] = value
	}
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice(staticValues)})
}

func shiftCompiledScanAccessSlots(schemaValue scm.Scmer, shift int) scm.Scmer {
	schema := schemaValue.Slice()
	if len(schema) == 0 {
		return schemaValue
	}
	shifted := append([]scm.Scmer(nil), schema...)
	meta, valid := decodeScanAccessHeader(shifted[0])
	if !valid {
		panic("invalid scan access header")
	}
	for offset, count := scanAccessSchemaHeaderSize, meta.count; count > 0; offset, count = offset+scanAccessBoundaryStride, count-1 {
		boundaryMeta := decodeScanAccessBoundaryMeta(shifted[offset+2])
		if boundaryMeta.lowerSlot >= 0 {
			boundaryMeta.lowerSlot += shift
		}
		if boundaryMeta.upperSlot >= 0 {
			boundaryMeta.upperSlot += shift
		}
		if mapperSlot := boundaryMeta.flags >> 3; mapperSlot > 0 {
			boundaryMeta.flags = boundaryMeta.flags&7 | (mapperSlot+int64(shift))<<3
		}
		shifted[offset+2] = newScanAccessBoundaryMeta(boundaryMeta.lowerSlot, boundaryMeta.upperSlot, boundaryMeta.flags)
	}
	return scm.NewSlice(shifted)
}

func compileScanAccessList(columnListsExpr, filtersExpr scm.Scmer, allowBatch bool) ([]scm.Scmer, []scm.Scmer, []bool, bool) {
	columnLists, columnsOK := scanStaticListElements(columnListsExpr)
	filters, filtersOK := scanStaticListElements(filtersExpr)
	if !columnsOK || !filtersOK || len(columnLists) != len(filters) {
		return nil, nil, nil, false
	}
	schemas := make([]scm.Scmer, len(columnLists))
	bindings := make([]scm.Scmer, 0)
	compiledEntries := make([]bool, len(columnLists))
	compiledAny := false
	for i := range columnLists {
		schema, sourceBindings, ok := compileScanAccessMode(columnLists[i], filters[i], allowBatch)
		schemas[i] = shiftCompiledScanAccessSlots(schema, len(bindings))
		if ok {
			bindings = append(bindings, sourceBindings...)
			compiledEntries[i] = true
			compiledAny = true
		}
	}
	return schemas, bindings, compiledEntries, compiledAny
}

func pruneScanResidualList(columnListsExpr, filtersExpr scm.Scmer, compiled []bool, allowBatch bool) (scm.Scmer, scm.Scmer) {
	columnLists, columnsOK := scanStaticListElements(columnListsExpr)
	filters, filtersOK := scanStaticListElements(filtersExpr)
	if !columnsOK || !filtersOK || len(columnLists) != len(filters) || len(compiled) != len(filters) {
		return columnListsExpr, filtersExpr
	}
	prunedColumns := make([]scm.Scmer, len(columnLists)+1)
	prunedFilters := make([]scm.Scmer, len(filters)+1)
	prunedColumns[0], prunedFilters[0] = scm.NewSymbol("list"), scm.NewSymbol("list")
	for i := range filters {
		prunedColumns[i+1], prunedFilters[i+1] = columnLists[i], filters[i]
		if compiled[i] {
			prunedColumns[i+1], prunedFilters[i+1] = pruneScanResidual(columnLists[i], filters[i], allowBatch)
		}
	}
	return scm.NewSlice(prunedColumns), scm.NewSlice(prunedFilters)
}

func markCoveredScanAccessSchemas(schemas []scm.Scmer, filtersExpr scm.Scmer) []scm.Scmer {
	filters, ok := scanStaticListElements(filtersExpr)
	if !ok || len(filters) != len(schemas) {
		return schemas
	}
	for i, filter := range filters {
		schemas[i] = markCoveredScanAccessSchema(schemas[i], filter)
	}
	return schemas
}

func markCoveredScanAccessSchema(schema, residual scm.Scmer) scm.Scmer {
	if !schema.IsSlice() || len(schema.Slice()) < scanAccessSchemaHeaderSize {
		return schema
	}
	_, body, lambda := scanLambdaParts(residual)
	if !lambda || !(body.SymbolEquals("true") || (body.IsBool() && body.Bool())) {
		return schema
	}
	items := append([]scm.Scmer(nil), schema.Slice()...)
	meta, valid := decodeScanAccessHeader(items[0])
	if !valid {
		return schema
	}
	items[0] = newScanAccessHeader(meta.count, scanAccessConsumerCoveredScan, meta.projections, meta.mapperSlot)
	return scm.NewSlice(items)
}

func scanAccessCoversResidual(access scanAccess) bool {
	return access.runtime != nil && access.runtime.filterCovered || access.plannerFilterCovered
}

// A zero-argument predicate is row-independent in Scheme's functional model.
// Evaluate it once at scan setup even when an optimizer/JIT wrapper prevents
// PrepareSerialProc from representing it as SerialProcConstant.
func scanConditionAlwaysTrue(program *scm.SerialProc, parameterCount int) bool {
	if program.Kind == scm.SerialProcConstant {
		return scm.ToBool(program.Value)
	}
	return parameterCount == 0 && scm.ToBool(program.Call(nil))
}

func scanParamColumn(expr scm.Scmer, params, columns []scm.Scmer) (string, bool) {
	for i, param := range params {
		if i >= len(columns) {
			break
		}
		name, named := scanSymbolName(param)
		if named && scanExprIsLambdaParam(expr, name, i) {
			column := columns[i].String()
			if _, batch := parseBatchPseudoColName(column); !batch {
				return column, true
			}
		}
	}
	return "", false
}

func scanExprUsesParams(expr scm.Scmer, params []scm.Scmer) bool {
	for i, param := range params {
		name, named := scanSymbolName(param)
		if named && scanExprIsLambdaParam(expr, name, i) {
			return true
		}
	}
	items, ok := scmerSlice(expr)
	if !ok {
		return false
	}
	if len(items) > 0 && scanSymbolIs(items[0], "quote") {
		return false
	}
	for _, item := range items {
		if scanExprUsesParams(item, params) {
			return true
		}
	}
	return false
}

type compiledScanBoundary struct {
	kind           string
	column         string
	lower          scm.Scmer
	upper          scm.Scmer
	lowerSet       bool
	upperSet       bool
	lowerBatch     bool
	upperBatch     bool
	lowerBatchSlot int
	upperBatchSlot int
	lowerInclusive bool
	upperInclusive bool
	nullSafe       bool
	collation      string
	mapCols        []string
	mapFn          scm.Scmer
}

func scanCompiledColumn(expr scm.Scmer, params, columns []scm.Scmer) (string, []string, scm.Scmer, bool) {
	if column, ok := scanParamColumn(expr, params, columns); ok {
		return column, nil, scm.NewNil(), true
	}
	expr = expr.WithoutSourceInfo()
	if !expr.IsSlice() || isIndependent(params, expr) || !scanComputedExpressionSafe(expr, params) {
		return "", nil, scm.NewNil(), false
	}
	mapCols := make([]string, len(columns))
	for i, column := range columns {
		if !column.IsString() {
			return "", nil, scm.NewNil(), false
		}
		mapCols[i] = column.String()
		if isScanPseudoColName(mapCols[i]) && computedExprUsesParameter(expr, params, i) {
			return "", nil, scm.NewNil(), false
		}
	}
	mapFn := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice(append([]scm.Scmer(nil), params...)),
		expr,
	})
	return canonicalColName(expr, params, mapCols), mapCols, mapFn, true
}

func scanComputedExpressionSafe(expr scm.Scmer, params []scm.Scmer) bool {
	expr = expr.WithoutSourceInfo()
	if isIndependent(params, expr) {
		return scanExprSafeToHoist(expr, false)
	}
	if expr.IsSymbol() || expr.IsNthLocalVar() {
		return true
	}
	if !expr.IsSlice() {
		return false
	}
	items := expr.Slice()
	if len(items) == 0 || items[0].IsNthLocalVar() {
		return false
	}
	declaration := scm.DeclarationForValue(items[0])
	if declaration == nil || !declaration.IsFoldable() {
		return false
	}
	for _, item := range items[1:] {
		if !scanComputedExpressionSafe(item, params) {
			return false
		}
	}
	return true
}

func compileComputedScanIndex(mapper scm.Scmer, mapCols []string) scm.Scmer {
	ctx, ok := conditionAnalyzeContext(mapCols, mapper)
	if !ok {
		panic("compiled scan index mapper must be a procedure")
	}
	expr := ctx.materializeComputedExpr(ctx.proc.Body)
	if !isRawDataset(ctx.params, expr) {
		panic("compiled scan index mapper contains a runtime-dependent formula")
	}
	cols, fn := buildComputedFn(expr, ctx.proc.Params, ctx.proc.En, mapCols)
	if fn.IsNil() || len(cols) != len(mapCols) {
		panic("compiled scan index mapper could not be materialized")
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewString(canonicalColName(expr, ctx.params, mapCols)),
		fn,
	})
}

func scanBatchParamSlot(expr scm.Scmer, params, columns []scm.Scmer) (int, bool) {
	for i, param := range params {
		if i >= len(columns) || !columns[i].IsString() {
			break
		}
		name, named := scanSymbolName(param)
		if !named || !scanExprIsLambdaParam(expr, name, i) {
			continue
		}
		return parseBatchPseudoColName(columns[i].String())
	}
	return 0, false
}

func compileScanComparison(node scm.Scmer, params, columns []scm.Scmer, allowBatch bool) (compiledScanBoundary, bool) {
	items, ok := scmerSlice(node)
	if !ok {
		return compiledScanBoundary{}, false
	}
	if len(items) != 3 {
		return compiledScanBoundary{}, false
	}
	operator, named := scanSymbolName(items[0])
	if !named {
		return compiledScanBoundary{}, false
	}
	leftColumn, leftMapCols, leftMapFn, leftIsColumn := scanCompiledColumn(items[1], params, columns)
	rightColumn, rightMapCols, rightMapFn, rightIsColumn := scanCompiledColumn(items[2], params, columns)
	if leftIsColumn == rightIsColumn {
		return compiledScanBoundary{}, false
	}
	column := leftColumn
	mapCols, mapFn := leftMapCols, leftMapFn
	value := items[2]
	reversed := false
	if rightIsColumn {
		column, value, reversed = rightColumn, items[1], true
		mapCols, mapFn = rightMapCols, rightMapFn
	}
	batchSlot, batchValue := 0, false
	if allowBatch {
		batchSlot, batchValue = scanBatchParamSlot(value, params, columns)
	}
	if !batchValue && scanExprUsesParams(value, params) {
		return compiledScanBoundary{}, false
	}
	value = scanLiftOutOfLambda(value)
	switch operator {
	case "equal?", "equal??":
		collation := ""
		if operator == "equal??" {
			collation = "utf8mb4_general_ci"
		}
		return compiledScanBoundary{kind: "equal", column: column, lower: value, upper: value, lowerSet: true, upperSet: true,
			lowerBatch: batchValue, upperBatch: batchValue, lowerBatchSlot: batchSlot, upperBatchSlot: batchSlot,
			lowerInclusive: true, upperInclusive: true, nullSafe: operator == "equal??", collation: collation, mapCols: mapCols, mapFn: mapFn}, true
	case "<", "<=", ">", ">=":
		inclusive := operator == "<=" || operator == ">="
		lower := operator == ">" || operator == ">="
		if reversed {
			lower = !lower
		}
		boundary := compiledScanBoundary{kind: "range", column: column, mapCols: mapCols, mapFn: mapFn}
		if lower {
			boundary.lower, boundary.lowerSet, boundary.lowerInclusive = value, true, inclusive
			boundary.lowerBatch, boundary.lowerBatchSlot = batchValue, batchSlot
		} else {
			boundary.upper, boundary.upperSet, boundary.upperInclusive = value, true, inclusive
			boundary.upperBatch, boundary.upperBatchSlot = batchValue, batchSlot
		}
		return boundary, true
	default:
		return compiledScanBoundary{}, false
	}
}

func compileScanSpecialBoundary(node scm.Scmer, params, columns []scm.Scmer) (compiledScanBoundary, bool) {
	items, ok := scmerSlice(node)
	if !ok {
		return compiledScanBoundary{}, false
	}
	if len(items) == 2 && scanSymbolIs(items[0], "nil?") {
		if column, ok := scanParamColumn(items[1], params, columns); ok {
			return compiledScanBoundary{kind: "equal", column: column, lower: scm.NewNil(), upper: scm.NewNil(), lowerSet: true, upperSet: true, nullSafe: true,
				lowerInclusive: true, upperInclusive: true}, true
		}
	}
	if len(items) >= 3 && len(items) <= 4 && scanSymbolIs(items[0], "strlike") {
		column, ok := scanParamColumn(items[1], params, columns)
		if !ok || scanExprUsesParams(items[2], params) {
			return compiledScanBoundary{}, false
		}
		collation := "utf8mb4_general_ci"
		if len(items) == 4 {
			collationValue := items[3].WithoutSourceInfo()
			if !collationValue.IsString() {
				return compiledScanBoundary{}, false
			}
			collation = strings.ToLower(collationValue.String())
		}
		pattern := scanLiftOutOfLambda(items[2])
		return compiledScanBoundary{kind: "like", column: column, lower: pattern, upper: pattern,
			lowerSet: true, upperSet: true, lowerInclusive: true, upperInclusive: true, collation: collation}, true
	}
	if len(items) == 2 {
		column, parameter := scanParamColumn(items[0], params, columns)
		if parameter && column == "$recset_contains" && !scanExprUsesParams(items[1], params) {
			value := scanLiftOutOfLambda(items[1])
			return compiledScanBoundary{kind: "recset", column: column, lower: value, upper: value,
				lowerSet: true, upperSet: true, lowerInclusive: true, upperInclusive: true}, true
		}
	}
	return compiledScanBoundary{}, false
}

func collectCompiledScanBoundaries(node scm.Scmer, params, columns []scm.Scmer, result []compiledScanBoundary, allowBatch bool) ([]compiledScanBoundary, bool) {
	if items, sliced := scmerSlice(node); sliced {
		if len(items) > 1 && scanSymbolIs(items[0], "and") {
			for _, child := range items[1:] {
				var valid bool
				result, valid = collectCompiledScanBoundaries(child, params, columns, result, allowBatch)
				if !valid {
					return result, false
				}
			}
			return result, true
		}
	}
	boundary, ok := compileScanComparison(node, params, columns, allowBatch)
	if !ok {
		boundary, ok = compileScanSpecialBoundary(node, params, columns)
	}
	if ok {
		for i, existing := range result {
			if existing.column == boundary.column {
				if existing.kind != "range" || boundary.kind != "range" {
					// One physical index dimension can carry only one matcher. Keep
					// the first useful probe and leave duplicate or conflicting
					// predicates to the residual filter; rejecting the complete
					// access program here would turn correlated point lookups into
					// full scans.
					return result, true
				}
				if (existing.lowerSet || existing.lowerBatch) && (boundary.lowerSet || boundary.lowerBatch) {
					return result, false
				}
				if (existing.upperSet || existing.upperBatch) && (boundary.upperSet || boundary.upperBatch) {
					return result, false
				}
				if boundary.lowerSet || boundary.lowerBatch {
					result[i].lower = boundary.lower
					result[i].lowerSet = boundary.lowerSet
					result[i].lowerBatch = boundary.lowerBatch
					result[i].lowerBatchSlot = boundary.lowerBatchSlot
					result[i].lowerInclusive = boundary.lowerInclusive
				}
				if boundary.upperSet || boundary.upperBatch {
					result[i].upper = boundary.upper
					result[i].upperSet = boundary.upperSet
					result[i].upperBatch = boundary.upperBatch
					result[i].upperBatchSlot = boundary.upperBatchSlot
					result[i].upperInclusive = boundary.upperInclusive
				}
				return result, true
			}
		}
		return append(result, boundary), true
	}
	return result, true
}

func compileScanAccess(columnExpr, filterExpr scm.Scmer) (scm.Scmer, []scm.Scmer, bool) {
	return compileScanAccessMode(columnExpr, filterExpr, false)
}

func scanLiteralDefinitelyNonNil(value scm.Scmer) bool {
	return value.IsBool() || value.IsInt() || value.IsFloat() || value.IsDate() || value.IsString() || value.IsBSON()
}

func compiledScanBoundaryCovers(have, want compiledScanBoundary) bool {
	if have.kind != want.kind || have.column != want.column || have.collation != want.collation || have.nullSafe != want.nullSafe {
		return false
	}
	if want.lowerSet && (!have.lowerSet || have.lowerInclusive != want.lowerInclusive ||
		!scm.Equal(have.lower.WithoutSourceInfo(), want.lower.WithoutSourceInfo())) {
		return false
	}
	if want.upperSet && (!have.upperSet || have.upperInclusive != want.upperInclusive ||
		!scm.Equal(have.upper.WithoutSourceInfo(), want.upper.WithoutSourceInfo())) {
		return false
	}
	if want.lowerBatch && (!have.lowerBatch || have.lowerInclusive != want.lowerInclusive || have.lowerBatchSlot != want.lowerBatchSlot) {
		return false
	}
	if want.upperBatch && (!have.upperBatch || have.upperInclusive != want.upperInclusive || have.upperBatchSlot != want.upperBatchSlot) {
		return false
	}
	return true
}

// pruneScanResidual removes only predicates which the exact physical
// enumerator guarantees. Candidate hooks such as LIKE and RecSet remain in the
// residual callback, as do expressions the access compiler did not recognize.
// Unused callback parameters are dropped so the scan does not fetch columns
// which were needed only to establish an exact seek or range.
func pruneScanResidual(columnExpr, filterExpr scm.Scmer, allowBatch bool) (scm.Scmer, scm.Scmer) {
	columns, columnsOK := scanStaticColumns(columnExpr)
	params, body, lambdaOK := scanLambdaParts(filterExpr)
	if !columnsOK || !lambdaOK || len(params) != len(columns) {
		return columnExpr, filterExpr
	}
	compiled, valid := collectCompiledScanBoundaries(body, params, columns, nil, allowBatch)
	if !valid {
		return columnExpr, filterExpr
	}
	sort.SliceStable(compiled, func(i, j int) bool {
		iSorted := compiled[i].kind == "equal" || compiled[i].kind == "range"
		jSorted := compiled[j].kind == "equal" || compiled[j].kind == "range"
		if iSorted != jSorted {
			return iSorted
		}
		if (compiled[i].kind == "equal") != (compiled[j].kind == "equal") {
			return compiled[i].kind == "equal"
		}
		return compiled[i].column < compiled[j].column
	})
	covered := make(map[string]compiledScanBoundary, len(compiled))
	for _, boundary := range compiled {
		switch boundary.kind {
		case "equal":
			covered[boundary.column] = boundary
		case "range":
			covered[boundary.column] = boundary
			// A lexicographic index can enforce only one range suffix.
			// Later range columns remain residual predicates.
			goto coverageComplete
		default:
			goto coverageComplete
		}
	}
coverageComplete:
	var prune func(scm.Scmer) scm.Scmer
	prune = func(node scm.Scmer) scm.Scmer {
		if items, ok := scmerSlice(node); ok && len(items) > 1 && scanSymbolIs(items[0], "and") {
			residual := make([]scm.Scmer, 1, len(items))
			residual[0] = items[0]
			for _, child := range items[1:] {
				remaining := prune(child)
				if !remaining.SymbolEquals("true") && !(remaining.IsBool() && remaining.Bool()) {
					residual = append(residual, remaining)
				}
			}
			switch len(residual) {
			case 1:
				return scm.NewBool(true)
			case 2:
				return residual[1]
			default:
				return scm.NewSlice(residual)
			}
		}
		boundary, exact := compileScanComparison(node, params, columns, allowBatch)
		have, physicallyCovered := covered[boundary.column]
		if exact && physicallyCovered && compiledScanBoundaryCovers(have, boundary) &&
			(!boundary.nullSafe || scanLiteralDefinitelyNonNil(boundary.lower)) {
			return scm.NewBool(true)
		}
		return node
	}

	residualBody := prune(body)
	residualParams := make([]scm.Scmer, 0, len(params))
	residualColumns := make([]scm.Scmer, 0, len(columns))
	for i, param := range params {
		if scanExprUsesParams(residualBody, []scm.Scmer{param}) {
			residualParams = append(residualParams, param)
			residualColumns = append(residualColumns, columns[i])
		}
	}
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice(residualColumns)}),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("lambda"), scm.NewSlice(residualParams), residualBody})
}

func compileScanAccessMode(columnExpr, filterExpr scm.Scmer, allowBatch bool) (scm.Scmer, []scm.Scmer, bool) {
	columns, columnsOK := scanStaticColumns(columnExpr)
	params, body, lambdaOK := scanLambdaParts(filterExpr)
	if !columnsOK || !lambdaOK || len(params) != len(columns) {
		return newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, false
	}
	compiled, valid := collectCompiledScanBoundaries(body, params, columns, nil, allowBatch)
	if !valid || len(compiled) == 0 {
		return newScanAccessSchema(scanAccessConsumerScan, nil, -1), nil, false
	}
	sort.SliceStable(compiled, func(i, j int) bool {
		iSorted := compiled[i].kind == "equal" || compiled[i].kind == "range"
		jSorted := compiled[j].kind == "equal" || compiled[j].kind == "range"
		if iSorted != jSorted {
			return iSorted
		}
		if (compiled[i].kind == "equal") != (compiled[j].kind == "equal") {
			return compiled[i].kind == "equal"
		}
		return compiled[i].column < compiled[j].column
	})
	var mapCols []string
	for _, boundary := range compiled {
		if !boundary.mapFn.IsNil() {
			mapCols = boundary.mapCols
			break
		}
	}
	schema := make([]scm.Scmer, 0, scanAccessSchemaHeaderSize+len(compiled)*scanAccessBoundaryStride+len(mapCols))
	schema = append(schema, newScanAccessHeader(len(compiled), scanAccessConsumerScan, len(mapCols), -1))
	bindings := make([]scm.Scmer, 0, len(compiled)*2)
	for _, boundary := range compiled {
		lowerSlot, upperSlot := int64(-1), int64(-1)
		if boundary.lowerBatch {
			lowerSlot = int64(-2 - boundary.lowerBatchSlot)
		} else if boundary.lowerSet {
			lowerSlot = int64(len(bindings))
			bindings = append(bindings, boundary.lower)
		}
		if boundary.upperBatch {
			upperSlot = int64(-2 - boundary.upperBatchSlot)
		} else if boundary.upperSet {
			if boundary.kind != "range" {
				upperSlot = lowerSlot
			} else {
				upperSlot = int64(len(bindings))
				bindings = append(bindings, boundary.upper)
			}
		}
		flags := int64(0)
		if boundary.lowerInclusive {
			flags |= 1
		}
		if boundary.upperInclusive {
			flags |= 2
		}
		if boundary.nullSafe {
			flags |= 4
		}
		if !boundary.mapFn.IsNil() {
			mapperSlot := len(bindings)
			mapColValues := make([]scm.Scmer, len(boundary.mapCols))
			for i, column := range boundary.mapCols {
				mapColValues[i] = scm.NewString(column)
			}
			bindings = append(bindings, scm.NewSlice([]scm.Scmer{
				scm.NewSymbol("compile_scan_computed_index"),
				boundary.mapFn,
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice(mapColValues)}),
			}))
			flags |= int64(mapperSlot+1) << 3
		}
		schema = append(schema,
			scm.NewString(boundary.kind), scm.NewString(boundary.column),
			newScanAccessBoundaryMeta(int(lowerSlot), int(upperSlot), flags), scm.NewString(boundary.collation))
	}
	for _, column := range mapCols {
		schema = append(schema, scm.NewString(column))
	}
	return scm.NewSlice(schema), bindings, true
}

func newScanAccessSchema(consumer string, projections []scm.Scmer, mapperSlot int) scm.Scmer {
	if consumer == scanAccessConsumerScan && len(projections) == 0 && mapperSlot < 0 {
		return scm.NewSlice(nil)
	}
	schema := make([]scm.Scmer, 0, scanAccessSchemaHeaderSize+len(projections))
	schema = append(schema, newScanAccessHeader(0, consumer, len(projections), mapperSlot))
	schema = append(schema, projections...)
	return scm.NewSlice(schema)
}

// tryScanInvariantFilterRewrite selects a row-independent IF branch once per
// scan invocation instead of evaluating it for every candidate row.
func tryScanInvariantFilterRewrite(v []scm.Scmer) scm.Scmer {
	// Every scan form shares tx/table/access-schema/access-values/filtercols/
	// filterfn at indices 1..6.
	if len(v) < 7 {
		return scm.NewNil()
	}
	lambda, ok := scmerSlice(v[6])
	if !ok || len(lambda) < 3 || !scanSymbolIs(lambda[0], "lambda") {
		return scm.NewNil()
	}
	_, body, ok := scanLambdaParts(v[6])
	if !ok {
		return scm.NewNil()
	}

	wrappedOptimize := false
	if bodyItems, bodyOK := scmerSlice(body); bodyOK && len(bodyItems) == 2 && scanSymbolIs(bodyItems[0], "optimize") {
		wrappedOptimize = true
		body = bodyItems[1]
	}
	conditional, ok := scmerSlice(body)
	if !ok || len(conditional) != 4 || !scanSymbolIs(conditional[0], "if") {
		return scm.NewNil()
	}
	condition := conditional[1]
	if !scanExprSafeToHoist(condition, false) {
		return scm.NewNil()
	}

	makeBranch := func(branch scm.Scmer) scm.Scmer {
		if wrappedOptimize {
			branch = scm.NewSlice([]scm.Scmer{scm.NewSymbol("optimize"), branch})
		}
		branchLambda := append([]scm.Scmer(nil), lambda...)
		branchLambda[2] = branch
		return scm.NewSlice(branchLambda)
	}

	rewritten := append([]scm.Scmer(nil), v...)
	rewritten[6] = scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("if"),
		scanLiftOutOfLambda(condition),
		makeBranch(conditional[2]),
		makeBranch(conditional[3]),
	})
	return scm.NewSlice(rewritten)
}

func scanExprSafeToHoist(expr scm.Scmer, belowOuter bool) bool {
	if expr.IsNthLocalVar() {
		return belowOuter
	}
	if _, ok := scanSymbolName(expr); ok {
		return false
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 {
		return true
	}
	if scanSymbolIs(items[0], "quote") {
		return true
	}
	if depth, inner, ok := scanOuterReference(expr); ok {
		return depth > 0 && scanExprSafeToHoist(inner, true)
	}
	name, named := scanSymbolName(items[0])
	if !named {
		return false
	}
	switch name {
	case "session", "equal?", "equal??", "nil?", "not", "sql_not", "and", "or", "coalesceNil", "bool?", "int?", "float?", "string?", "<", "<=", ">", ">=":
	default:
		return false
	}
	for _, item := range items[1:] {
		if !scanExprSafeToHoist(item, belowOuter) {
			return false
		}
	}
	return true
}

func scanLiftOutOfLambda(expr scm.Scmer) scm.Scmer {
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 || scanSymbolIs(items[0], "quote") {
		return expr
	}
	if depth, inner, ok := scanOuterReference(expr); ok && depth > 0 {
		if depth == 1 {
			return inner
		}
		return scm.NewSlice([]scm.Scmer{items[0], scm.NewInt(int64(depth - 1)), inner})
	}
	lifted := make([]scm.Scmer, len(items))
	for i, item := range items {
		lifted[i] = scanLiftOutOfLambda(item)
	}
	return scm.NewSlice(lifted)
}

func tryScanExistsRewrite(v []scm.Scmer) scm.Scmer {
	// scan: [fn, tx, table, accessSchema, accessValues, filtercols, filterfn,
	// mapcols, mapreduce, neutral, combine, isOuter]
	if len(v) < 12 {
		return scm.NewNil()
	}
	if scm.ToBool(v[11]) {
		return scm.NewNil()
	}
	if !scanFalseNeutral(v[9]) || !scanExistsMapReduce(v[8]) || !scanExistsOrReducer(v[10]) {
		return scm.NewNil()
	}
	if scanMapColsHaveSideEffects(v[7]) || scanExprMayHaveSideEffects(v[6]) {
		return scm.NewNil()
	}
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("scan_exists"),
		v[1],
		v[2],
		v[3],
		v[4],
		v[5],
		v[6],
	})
}

func scanFalseNeutral(v scm.Scmer) bool {
	if v.IsBool() {
		return !v.Bool()
	}
	return false
}

func scanExistsMapReduce(v scm.Scmer) bool {
	params, body, ok := scanLambdaParts(v)
	if !ok || len(params) == 0 {
		return false
	}
	acc, ok := scanSymbolName(params[0])
	if !ok || !body.IsSlice() {
		return false
	}
	items := body.Slice()
	if len(items) < 3 || !scanSymbolIs(items[0], "or") {
		return false
	}
	seenAcc := false
	seenTrue := false
	for _, item := range items[1:] {
		if item.IsSymbol() && item.String() == acc {
			seenAcc = true
		}
		if scanExprIsTrue(item) {
			seenTrue = true
		}
	}
	return seenAcc && seenTrue
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
	items, ok := scmerSlice(v)
	if !ok {
		return nil, scm.NewNil(), false
	}
	if len(items) < 3 || !scanSymbolIs(items[0], "lambda") {
		return nil, scm.NewNil(), false
	}
	paramsExpr := items[1]
	if paramsExpr.IsNil() {
		return []scm.Scmer{}, items[2], true
	}
	params, paramsOK := scmerSlice(paramsExpr)
	if !paramsOK {
		return nil, scm.NewNil(), false
	}
	return params, items[2], true
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

func scanOuterReference(v scm.Scmer) (int, scm.Scmer, bool) {
	items, ok := scmerSlice(v)
	if !ok || len(items) != 3 || !scanSymbolIs(items[0], "outer") {
		return 0, scm.NewNil(), false
	}
	depthValue := items[1].WithoutSourceInfo()
	var depth int64
	switch {
	case depthValue.IsInt():
		depth = depthValue.Int()
	case depthValue.IsFloat():
		depth = depthValue.Int()
		if depthValue.Float() != float64(depth) {
			return 0, scm.NewNil(), false
		}
	default:
		return 0, scm.NewNil(), false
	}
	if depth < 0 || int64(int(depth)) != depth {
		return 0, scm.NewNil(), false
	}
	return int(depth), items[2], true
}

func scanSymbolName(v scm.Scmer) (string, bool) {
	v = v.WithoutSourceInfo()
	if v.GetTag() == scm.TagSymbol {
		return v.String(), true
	}
	if declaration := scm.DeclarationForValue(v); declaration != nil && declaration.IsSpecialForm {
		return declaration.Name, true
	}
	if !v.IsSlice() {
		return "", false
	}
	items := v.Slice()
	if len(items) == 2 && items[0].SymbolEquals("quote") && items[1].GetTag() == scm.TagSymbol {
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
	if declaration := scm.DeclarationForValue(v); declaration != nil && declaration.Type != nil && declaration.Type.HasSideEffects {
		return true
	}
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
	if rewritten := tryScanInvariantFilterRewrite(v); !rewritten.IsNil() {
		if result, td, accepted := oc.OptimizeRewrite(scm.NewSlice(v), rewritten, useResult, scm.OptimizerRewriteContract{
			Name: "scan-batch-invariant-filter", PreconditionsMet: true, MaxGrowthNodes: 64,
		}); accepted {
			return result, td
		}
	}
	return optimizeScanShared(v, oc, 8, 11, 12, 13)
}

// scanResult bundles per-shard outputs to minimize allocations and type assertions.
type scanResult struct {
	res            scm.Scmer
	outCount       int64
	inputCount     int64
	candidateCount int64
	err            scanError // err.r != nil indicates an error
}

// scanResultCollector keeps the one-shard path synchronous. Parallel scans
// lazily allocate their channel on the first non-solo callback; a scan whose
// selected topology contains one shard writes its sole result directly.
type scanResultCollector struct {
	channelSize int
	once        sync.Once
	parallel    chan scanResult
	solo        scanResult
	soloSet     bool
	soloRead    bool
}

func (c *scanResultCollector) send(solo bool, result scanResult) {
	if solo {
		c.solo = result
		c.soloSet = true
		return
	}
	c.once.Do(func() {
		c.parallel = make(chan scanResult, c.channelSize)
	})
	c.parallel <- result
}

func (c *scanResultCollector) finish(done <-chan struct{}) {
	if done != nil {
		<-done
	}
	if c.parallel != nil {
		close(c.parallel)
	}
}

func (c *scanResultCollector) next() (scanResult, bool) {
	if c.parallel != nil {
		result, ok := <-c.parallel
		return result, ok
	}
	if !c.soloSet || c.soloRead {
		return scanResult{}, false
	}
	c.soloRead = true
	return c.solo, true
}

const (
	defaultScanBufferSize     = 1024
	uniquePointScanBufferSize = 8
)

type fullScanIDBuffer struct {
	values [defaultScanBufferSize]uint32
}

type pointScanIDBuffer struct {
	values [uniquePointScanBufferSize]uint32
}

// The index iterator callback makes a dynamically sized []uint32 escape even
// though it is consumed before scan returns. Reuse the two calibrated batch
// sizes instead of paying for a heap allocation on every scan. sync.Pool keeps
// concurrent scans independent and permits the runtime to discard idle memory.
var fullScanIDBufferPool = sync.Pool{
	New: func() any { return new(fullScanIDBuffer) },
}

var pointScanIDBufferPool = sync.Pool{
	New: func() any { return new(pointScanIDBuffer) },
}

func acquireScanIDBuffer(size int) ([]uint32, *fullScanIDBuffer, *pointScanIDBuffer) {
	if size == defaultScanBufferSize {
		buffer := fullScanIDBufferPool.Get().(*fullScanIDBuffer)
		return buffer.values[:], buffer, nil
	}
	if size == uniquePointScanBufferSize {
		buffer := pointScanIDBufferPool.Get().(*pointScanIDBuffer)
		return buffer.values[:], nil, buffer
	}
	return make([]uint32, size), nil, nil
}

func releaseScanIDBuffer(full *fullScanIDBuffer, point *pointScanIDBuffer) {
	if full != nil {
		fullScanIDBufferPool.Put(full)
	}
	if point != nil {
		pointScanIDBufferPool.Put(point)
	}
}

// scanBufferSize keeps full scans batched while avoiding a 4 KiB allocation
// for the common join case where an exact unique key can yield at most one
// currently visible row. A few slots remain for stale index entries left by
// updates; iterateIndex still visits further batches when necessary.
func (t *table) scanBufferSize(boundaries scanAccess) int {
	if t.hasBoundUniquePoint(boundaries) {
		return uniquePointScanBufferSize
	}
	return defaultScanBufferSize
}

func (t *table) hasBoundUniquePoint(boundaries scanAccess) bool {
	for _, unique := range t.Unique {
		covered := true
		for _, col := range unique.Cols {
			matched := false
			for i := 0; i < boundaries.len(); i++ {
				boundary := boundaries.boundary(i)
				if boundary.col != col || !matcherKindEqual(boundary.matcher, EqualMatcher) ||
					boundary.lowerBatch || boundary.upperBatch || boundary.lower.IsNil() || boundary.upper.IsNil() ||
					!boundary.lowerInclusive || !boundary.upperInclusive ||
					!boundaryValueEqual(boundary.lower, boundary.upper) {
					continue
				}
				matched = true
				break
			}
			if !matched {
				covered = false
				break
			}
		}
		if covered && len(unique.Cols) > 0 {
			return true
		}
	}
	return false
}

func (t *table) hasUniqueColumns(columns []string) bool {
	for _, unique := range t.Unique {
		if len(unique.Cols) != len(columns) {
			continue
		}
		matched := true
		for i, column := range columns {
			if unique.Cols[i] != column {
				matched = false
				break
			}
		}
		if matched && len(columns) > 0 {
			return true
		}
	}
	return false
}

func (t *table) scanExists(currentTx *TxContext, accessSchema scm.Scmer, accessValues []scm.Scmer, conditionCols []string, condition scm.Scmer) bool {
	return t.scanExistsFrom(currentTx, nil, accessSchema, accessValues, conditionCols, condition)
}

func (t *table) scanExistsFrom(currentTx *TxContext, source *recSet, accessSchema scm.Scmer, accessValues []scm.Scmer, conditionCols []string, condition scm.Scmer) bool {
	ss := SessionStateFromTx(currentTx)
	querySeq := querySeqFromTx(currentTx)
	touchTempColumns(t, conditionCols, nil)
	suffix := appendRecSetBoundary(nil, source)
	access, compiled := scanAccessFromScheme(accessSchema, accessValues, suffix)
	if !compiled {
		panic("scan_exists received an invalid compiled access schema")
	}
	if access.impossible() {
		return false
	}
	var scratch *scanAnalyzeScratch
	if access.len() > 0 {
		scratch = acquireScanAnalyzeScratch()
		defer releaseScanAnalyzeScratch(scratch)
		access = access.useScratch(scratch)
	}
	var lowerStorage []scm.Scmer
	if scratch != nil {
		lowerStorage = scratch.lower[:0]
	}
	lower, upperLast := indexFromScanAccessInto(lowerStorage, access)
	for i := 0; i < access.len(); i++ {
		b := access.boundary(i)
		t.AddPartitioningScore([]string{b.col})
	}

	values := scanResultCollector{channelSize: t.shardResultBufferSize()}
	var found atomic.Bool
	done := t.iterateShardsParallel(currentTx, access, func(s *storageShard, solo bool) {
		if found.Load() {
			values.send(solo, scanResult{})
			return
		}
		defer func() {
			if r := recover(); r != nil {
				values.send(solo, scanResult{err: scanError{r, string(debug.Stack())}})
			}
		}()
		// Cancellation contract: check only at the scheduling boundary, before entering
		// the shard. Once entered, a shard runs atomically without cancellation checks.
		if ss != nil && ss.IsKilledSeq(querySeq) {
			panic("query killed")
		}
		if s.scanExists(access, lower, upperLast, conditionCols, condition, currentTx, ss, &found) {
			found.Store(true)
			values.send(solo, scanResult{outCount: 1})
			return
		}
		values.send(solo, scanResult{})
	})
	values.finish(done)

	var scanErr scanError
	for msg, ok := values.next(); ok; msg, ok = values.next() {
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

// Fused map-reduce implementation based on Scheme callbacks.
func (t *table) scan(currentTx *TxContext, accessSchema scm.Scmer, accessValues []scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, combine scm.Scmer, isOuter bool) scm.Scmer {
	return t.scanWithBatchFrom(currentTx, nil, accessSchema, accessValues, scanAccess{}, conditionCols, condition, callbackCols, mapReduce, neutral, combine, isOuter, 0, nil)
}

func (t *table) scanWithBatch(currentTx *TxContext, accessSchema scm.Scmer, accessValues []scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, combine scm.Scmer, isOuter bool, stride int, batchdata []scm.Scmer) scm.Scmer {
	return t.scanWithBatchFrom(currentTx, nil, accessSchema, accessValues, scanAccess{}, conditionCols, condition, callbackCols, mapReduce, neutral, combine, isOuter, stride, batchdata)
}

func (t *table) scanWithBatchFrom(currentTx *TxContext, source *recSet, accessSchema scm.Scmer, accessValues []scm.Scmer, requiredAccess scanAccess, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, combine scm.Scmer, isOuter bool, stride int, batchdata []scm.Scmer) scm.Scmer {
	ss := SessionStateFromTx(currentTx)
	querySeq := querySeqFromTx(currentTx)
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}
	if hasMutationCallback {
		t.mutationMu.Lock()
		defer t.mutationMu.Unlock()
	}
	if t.hasTableLock() {
		t.waitTableLock(ss, querySeqFromTx(currentTx), hasMutationCallback)
	}
	// touch temp columns so CacheManager knows they're still in use
	touchTempColumns(t, conditionCols, callbackCols)
	// Measure analysis time (boundary extraction, sharding hints)
	analyzeStart := time.Now()
	/* analyze query */
	var suffix boundaries
	if requiredAccess.native != nil {
		suffix = requiredAccess.native
	} else if requiredAccess.runtime != nil {
		suffix = requiredAccess.runtime.suffix
	}
	if source != nil {
		// requiredAccess may be shared by every batch invocation. Copy only when
		// a RecSet boundary must be appended; the common dynamic point probe can
		// borrow its immutable suffix directly.
		suffix = append(boundaries(nil), suffix...)
		suffix = appendRecSetBoundary(suffix, source)
	}
	access, compiled := scanAccessFromScheme(accessSchema, accessValues, suffix)
	if !compiled {
		panic("scan received an invalid compiled access schema")
	}
	if access.impossible() {
		if !isOuter {
			return neutral
		}
		nullArgs := make([]scm.Scmer, len(callbackCols)+1)
		nullArgs[0] = neutral
		return scm.Apply(mapReduce, nullArgs...)
	}
	var scratch *scanAnalyzeScratch
	if access.len() > 0 {
		scratch = acquireScanAnalyzeScratch()
		defer releaseScanAnalyzeScratch(scratch)
		access = access.useScratch(scratch)
	}
	var lowerStorage []scm.Scmer
	if scratch != nil {
		lowerStorage = scratch.lower[:0]
	}
	lower, upperLast := indexFromScanAccessInto(lowerStorage, access)
	if Settings.ScanDebugging {
		dbg := fmt.Sprintf("[SCAN] %s.%s", t.schema.Name, t.Name)
		for i := 0; i < access.len(); i++ {
			b := access.boundary(i)
			dbg += fmt.Sprintf(" %s:[%v..%v]", b.col, b.lower, b.upper)
		}
		dbg += fmt.Sprintf(" lower=%v upper=%v", lower, upperLast)
		fmt.Println(dbg)
	}
	// give sharding hints
	for i := 0; i < access.len(); i++ {
		b := access.boundary(i)
		t.AddPartitioningScore([]string{b.col})
	}

	analyzeNs := time.Since(analyzeStart).Nanoseconds()
	// Measure execution time (parallel shard scans + collection)
	execStart := time.Now()
	var outCount int64
	var inputCount int64
	var candidateCount int64
	values := scanResultCollector{channelSize: t.shardResultBufferSize()}
	done := t.iterateShardsParallel(currentTx, access, func(s *storageShard, solo bool) {
		defer func() {
			if r := recover(); r != nil {
				values.send(solo, scanResult{err: scanError{r, string(debug.Stack())}})
			}
		}()
		// Cancellation contract: check only at the scheduling boundary, before entering
		// the shard. Once entered, a shard runs atomically without cancellation checks.
		if ss != nil && ss.IsKilledSeq(querySeq) {
			panic("query killed")
		}
		res, shardOutCount, shardCandidateCount := s.scan(access, lower, upperLast, conditionCols, condition, callbackCols, mapReduce, neutral, stride, batchdata, currentTx, ss)
		values.send(solo, scanResult{res: res, outCount: shardOutCount, inputCount: int64(s.Count()), candidateCount: shardCandidateCount})
	})
	values.finish(done)

	akkumulator := neutral
	hadValue := false
	var scanErr scanError
	if !combine.IsNil() {
		fn := scm.OptimizeProcToSerialFunction(combine)
		for msg, ok := values.next(); ok; msg, ok = values.next() {
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
			candidateCount += msg.candidateCount
			outCount += msg.outCount
			if msg.outCount > 0 {
				akkumulator = fn(akkumulator, msg.res)
				hadValue = true
			}
		}
		if scanErr.r == nil && !hadValue && isOuter {
			nullArgs := make([]scm.Scmer, len(callbackCols)+1)
			nullArgs[0] = akkumulator
			akkumulator = scm.Apply(mapReduce, nullArgs...)
		}
	} else {
		for msg, ok := values.next(); ok; msg, ok = values.next() {
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
			candidateCount += msg.candidateCount
			outCount += msg.outCount
			hadValue = hadValue || msg.outCount > 0
		}
		if scanErr.r == nil && !hadValue && isOuter {
			nullArgs := make([]scm.Scmer, len(callbackCols)+1)
			nullArgs[0] = akkumulator
			akkumulator = scm.Apply(mapReduce, nullArgs...)
		}
	}
	if scanErr.r != nil {
		panic(scanErr)
	}
	// log statistics (best-effort, async so it doesn't add latency)
	execNs := time.Since(execStart).Nanoseconds()
	if Settings.ScanDebugging || candidateCount > int64(Settings.AnalyzeMinItems) {
		// Boundaries may live in pooled scan scratch. Encode them before the
		// asynchronous logger starts so no reference outlives this scan.
		indexColsEnc := scanAccessIndexCols(access)
		go func(anNs, exNs int64, indexColsEnc string) {
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
			safeLogScan(t.schema.Name, t.Name, false, filterEnc, "", indexColsEnc, inputCount, candidateCount, outCount, anNs, exNs)
		}(analyzeNs, execNs, indexColsEnc)
	}
	return akkumulator
}

func (t *storageShard) scanExists(boundaries scanAccess, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState, stop *atomic.Bool) bool {
	_, found := t.scanFirstRecord(boundaries, lower, upperLast, conditionCols, condition, currentTx, ss, stop)
	return found
}

func (t *storageShard) filterVisibleScanBatch(batch []uint32, visibleUpper uint32, hasMutationCallback bool, currentTx *TxContext, mutationSeen map[uint32]struct{}) int {
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
			continue
		}
		batch[outN] = effectiveIdx
		outN++
	}
	return outN
}

func (t *storageShard) filterConditionScanBatch(batch []uint32, conditionCols []string, ccols []ColumnStorage, cReaders []ColumnReader, conditionGetters []mapArgGetter, cdataset []scm.Scmer, condition *scm.SerialProc) int {
	outN := 0
	for _, effectiveIdx := range batch {
		if effectiveIdx < t.main_count {
			for i, reader := range cReaders {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(effectiveIdx, 0)
				} else {
					cdataset[i] = reader.GetValue(effectiveIdx)
				}
			}
		} else {
			for i, col := range conditionCols {
				if getter := conditionGetters[i]; getter != nil {
					cdataset[i] = getter(effectiveIdx, 0)
				} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
					cdataset[i] = cReaders[i].GetValue(effectiveIdx)
				} else {
					cdataset[i] = t.getDelta(int(effectiveIdx-t.main_count), col)
				}
			}
		}
		if !scm.ToBool(condition.Call(cdataset)) {
			continue
		}
		batch[outN] = effectiveIdx
		outN++
	}
	return outN
}

func (t *storageShard) filterNativeArgConstantScanBatch(batch []uint32, conditionCols []string, ccols []ColumnStorage, cReaders []ColumnReader, conditionGetters []mapArgGetter, condition *scm.SerialProc) int {
	argument := int(condition.Argument)
	if argument < 0 || argument >= len(conditionCols) {
		panic("serial filter argument outside condition columns")
	}
	// Passing two interface values directly to an escaping variadic function
	// makes the compiler allocate its argument slice for every row. Keep one
	// caller-owned frame on the stack and overwrite it in the loop instead.
	var call [2]scm.Scmer
	outN := 0
	for _, idx := range batch {
		var value scm.Scmer
		if getter := conditionGetters[argument]; getter != nil {
			value = getter(idx, 0)
		} else if idx < t.main_count {
			value = cReaders[argument].GetValue(idx)
		} else if _, isProxy := ccols[argument].(*StorageComputeProxy); isProxy {
			value = cReaders[argument].GetValue(idx)
		} else {
			value = t.getDelta(int(idx-t.main_count), conditionCols[argument])
		}
		if condition.ConstantFirst {
			call[0], call[1] = condition.Value, value
		} else {
			call[0], call[1] = value, condition.Value
		}
		if !scm.ToBool(condition.Function(call[:]...)) {
			continue
		}
		batch[outN] = idx
		outN++
	}
	return outN
}

func (t *storageShard) filterVisibleBatchedScanBatch(batch []uint32, batchIDs []uint32, batchID uint32, visibleUpper uint32, hasMutationCallback bool, currentTx *TxContext, mutationSeen map[uint64]struct{}) int {
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
			key := uint64(batchID)<<32 | uint64(effectiveIdx)
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
		batch[outN] = effectiveIdx
		batchIDs[outN] = batchID
		outN++
	}
	return outN
}

func (t *storageShard) scanFirstRecord(boundaries scanAccess, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState, stop *atomic.Bool) (uint32, bool) {
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}
	conditionProgram := scm.PrepareSerialProc(condition)
	conditionAlwaysTrue := scanConditionAlwaysTrue(&conditionProgram, len(conditionCols)) ||
		scanAccessProvesCondition(conditionCols, condition, boundaries)

	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
	t.ensureScanAccessColumns(boundaries, skipShardReadLock, currentTx)
	recsetBoundaryCoversCondition := recSetHooksCoverCondition(boundaries, lower, t.t, conditionCols, condition)

	var ccols []ColumnStorage
	var cReaders []ColumnReader
	var cNeedsCachedReader []bool
	var conditionGetters []mapArgGetter
	var cdataset []scm.Scmer
	if !conditionAlwaysTrue {
		ccols = make([]ColumnStorage, len(conditionCols))
		cReaders = make([]ColumnReader, len(conditionCols))
		cNeedsCachedReader = make([]bool, len(conditionCols))
		conditionGetters = make([]mapArgGetter, len(conditionCols))
		for i, k := range conditionCols {
			if k == "$recset_contains" {
				fnptr := recSetContainsClosure(t)
				if recsetBoundaryCoversCondition {
					fnptr = recSetAlreadyMatchedClosure()
				}
				conditionGetters[i] = func(id uint32, _ uint32) scm.Scmer {
					return scm.NewClosure(fnptr, id)
				}
				continue
			}
			ccols[i] = t.getColumnStorageOrPanic(k, skipShardReadLock, currentTx)
			cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
			if _, ok := ccols[i].(*StorageComputeProxy); ok {
				cNeedsCachedReader[i] = true
			}
		}
		cdataset = make([]scm.Scmer, len(conditionCols))
	}

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, querySeqFromTx(currentTx), false)
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
	var foundID uint32

	buf, pooledFullBuf, pooledPointBuf := acquireScanIDBuffer(uniquePointScanBufferSize)
	defer releaseScanIDBuffer(pooledFullBuf, pooledPointBuf)
	t.iterateIndex(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf, 1, nil, func(batch []uint32) bool {
		if stop != nil && stop.Load() {
			return false
		}
		if conditionAlwaysTrue {
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
				found = true
				foundID = idx
				return false
			}
			return true
		}
		if conditionProgram.Kind == scm.SerialProcNativeArgConstant {
			var candidate [1]uint32
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
				candidate[0] = idx
				if t.filterNativeArgConstantScanBatch(candidate[:], conditionCols, ccols, cReaders, conditionGetters, &conditionProgram) == 0 {
					continue
				}
				found = true
				foundID = idx
				return false
			}
			return true
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
						cdataset[i] = t.getDelta(int(idx-mainCount), col)
					}
				}
			}
			if scm.ToBool(conditionProgram.Call(cdataset)) {
				found = true
				foundID = idx
				return false
			}
		}
		return true
	})

	return foundID, found
}

func (t *storageShard) scan(boundaries scanAccess, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64, int64) {
	if stride > 0 {
		return t.scanBatch(boundaries, lower, upperLast, conditionCols, condition, callbackCols, mapReduce, neutral, stride, batchdata, currentTx, ss)
	}
	akkumulator := neutral
	var outCount int64
	var candidateCount int64
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}

	conditionProgram := scm.PrepareSerialProc(condition)
	conditionAlwaysTrue := scanConditionAlwaysTrue(&conditionProgram, len(conditionCols))
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}
	// An index may return a historical record ID that visibility resolution
	// forwards to a newer primary record whose predicate columns changed. A
	// read may trust its exact access proof, but a mutation must recheck the
	// visible row before applying update/delete pseudo-columns.
	conditionAlwaysTrue = conditionAlwaysTrue || !hasMutationCallback &&
		scanAccessProvesCondition(conditionCols, condition, boundaries)

	// Ensure shard is loaded from disk before accessing columns.
	// ensureLoaded() must run before getColumnStorageOrPanic so that COLD
	// shards have their column map populated by load(t) first.
	// ensureMainCount then loads at least one column to initialize main_count.
	t.ensureLoaded()
	t.ensureMainCount(false)
	t.ensureScanAccessColumns(boundaries, false, currentTx)
	// Most scans do not read an ordered computed column. Keep discovery on the
	// caller's stack and inspect the two existing column slices directly, so the
	// correctness preflight below adds no heap work to an ordinary scan.
	var orderedProxies []*StorageComputeProxy
	if t.hasOrderedScanProxy(conditionCols, currentTx) || t.hasOrderedScanProxy(callbackCols, currentTx) {
		var orderedProxyStorage [4]*StorageComputeProxy
		orderedProxies = t.appendOrderedScanProxies(orderedProxyStorage[:0], conditionCols, currentTx)
		orderedProxies = t.appendOrderedScanProxies(orderedProxies, callbackCols, currentTx)
	}
	prepareOrdered := func() {
		if len(orderedProxies) == 0 {
			return
		}
		t.mu.RLock()
		upper := t.main_count + uint32(len(t.inserts))
		recids := make([]uint32, 0, upper)
		for id := uint32(0); id < upper; id++ {
			if currentTx != nil && currentTx.Mode == TxACID {
				if !currentTx.IsVisible(t, id) {
					continue
				}
			} else if t.deletions.Get(uint(id)) {
				continue
			}
			recids = append(recids, id)
		}
		t.mu.RUnlock()
		for _, proxy := range orderedProxies {
			for _, id := range recids {
				if !proxy.validMask.Get(uint(id)) {
					proxy.GetValue(id)
				}
			}
		}
	}
	orderedReadyLocked := func() bool {
		if len(orderedProxies) == 0 {
			return true
		}
		upper := t.main_count + uint32(len(t.inserts))
		for id := uint32(0); id < upper; id++ {
			if currentTx != nil && currentTx.Mode == TxACID {
				if !currentTx.IsVisible(t, id) {
					continue
				}
			} else if t.deletions.Get(uint(id)) {
				continue
			}
			for _, proxy := range orderedProxies {
				if !proxy.validMask.Get(uint(id)) {
					return false
				}
			}
		}
		return true
	}
	ownsWrite := false
	lockMutationExclusively := hasMutationCallback && !ownsWrite
	writeLocked := false
	if lockMutationExclusively {
		for {
			prepareOrdered()
			t.lockForMutation(currentTx)
			if orderedReadyLocked() {
				writeLocked = true
				break
			}
			t.mu.Unlock()
		}
		defer func() {
			if writeLocked {
				t.mu.Unlock()
			}
		}()
		if currentTx != nil {
			currentTx.EnterShardWrite(t)
			defer currentTx.ExitShardWrite(t)
		}
	}
	skipShardReadLock := ownsWrite || lockMutationExclusively
	recsetBoundaryCoversCondition := recSetHooksCoverCondition(boundaries, lower, t.t, conditionCols, condition)

	// condition column readers
	var ccols []ColumnStorage
	var cReaders []ColumnReader
	var conditionGetters []mapArgGetter
	var cdataset []scm.Scmer
	if !conditionAlwaysTrue {
		ccols = make([]ColumnStorage, len(conditionCols))
		cReaders = make([]ColumnReader, len(conditionCols))
		conditionGetters = make([]mapArgGetter, len(conditionCols))
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
			ccols[i] = t.getColumnStorageOrPanic(k, skipShardReadLock, currentTx)
			cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
		}
		cdataset = make([]scm.Scmer, len(conditionCols))
	}

	// MapReducer for the fused callback phase (builds column readers internally)
	var mapperStorage ShardMapReducer
	var mapperWorkspace shardMapReducerWorkspace
	mapper := &mapperStorage
	if mapReducerCanUseReadWorkspace(callbackCols) {
		prepareReadMapReducerStorage(&mapperStorage, &mapperWorkspace, len(callbackCols))
		t.initReadMapReducer(&mapperStorage, callbackCols, mapReduce, skipShardReadLock, currentTx)
	} else {
		mapper = t.OpenMapReducer(callbackCols, mapReduce, skipShardReadLock, 0, nil, currentTx)
	}
	defer mapper.Close()
	// Use a guarded lock that will always be released on panic to avoid leaked locks.
	locked := false
	if !skipShardReadLock {
		acquireVerifiedReadLock := func() {
			for {
				prepareOrdered()
				t.mu.RLock()
				if orderedReadyLocked() {
					locked = true
					return
				}
				t.mu.RUnlock()
			}
		}
		acquireVerifiedReadLock()
		// Table lock check must happen AFTER shard RLock to close the TOCTOU window:
		// WRITE publication holds every shard write lock, so a scan that gets past
		// RLock observes it. READ locks do not block ordinary scans.
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, querySeqFromTx(currentTx), hasMutationCallback)
			// A writer may have invalidated an ordered value while this scan was
			// waiting. Prepare and verify again instead of blindly reacquiring RLock.
			acquireVerifiedReadLock()
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

	// filter phase: iterateIndex fills the reusable buffer, callback filters in-place and flushes to MapReducer
	buf, pooledFullBuf, pooledPointBuf := acquireScanIDBuffer(t.t.scanBufferSize(boundaries))
	defer releaseScanIDBuffer(pooledFullBuf, pooledPointBuf)
	hadValue := false

	t.iterateIndex(currentTx, boundaries, lower, upperLast, maxInsertIndex, buf, 1, nil, func(batch []uint32) bool {
		candidateCount += int64(len(batch))
		outN := t.filterVisibleScanBatch(batch, visibleUpper, hasMutationCallback, currentTx, mutationSeen)
		if !conditionAlwaysTrue && outN > 0 {
			if conditionProgram.Kind == scm.SerialProcNativeArgConstant {
				outN = t.filterNativeArgConstantScanBatch(batch[:outN], conditionCols, ccols, cReaders, conditionGetters, &conditionProgram)
			} else {
				outN = t.filterConditionScanBatch(batch[:outN], conditionCols, ccols, cReaders, conditionGetters, cdataset, &conditionProgram)
			}
		}
		if outN > 0 {
			if hasMutationCallback {
				pendingRecids = append(pendingRecids, batch[:outN]...)
				outCount += int64(outN)
				hadValue = true
			} else {
				// release lock for the fused callback (UpdateFunction needs write lock)
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
		return scm.NewNil(), outCount, candidateCount
	}
	if hasMutationCallback && len(pendingRecids) > 0 {
		// Release exclusive lock before the fused callback: it may contain
		// nested scans on the same table (e.g. EXISTS inside UPDATE).
		// The mapper re-acquires mu.Lock() per batch internally via
		// processMainBlock/processDeltaBlock when shardWriteLocked=false.
		// Table-level mutationMu still serializes concurrent mutations.
		if writeLocked {
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
		t.mu.Unlock()
		writeLocked = false
	}
	mapper.FlushSideEffects()
	return akkumulator, outCount, candidateCount
}

// appendOrderedScanProxies appends the ORC-backed columns that a scan may read.
// They must be made valid before the shard lock is acquired: an on-demand ORC
// repair takes shard write locks and therefore cannot run under a scan-held
// shard read or write lock. The caller supplies stack capacity for the normal
// case; linear deduplication is cheaper than allocating a map for a handful of
// projected columns.
func (t *storageShard) appendOrderedScanProxies(result []*StorageComputeProxy, cols []string, currentTx *TxContext) []*StorageComputeProxy {
	for _, col := range cols {
		if col == "$recset_contains" || col == "$update" || col == "$break" ||
			strings.HasPrefix(col, "NEW.") || strings.HasPrefix(col, "$invalidate:") ||
			strings.HasPrefix(col, "$increment:") || strings.HasPrefix(col, "$set:") {
			continue
		}
		if _, ok := parseBatchPseudoColName(col); ok {
			continue
		}
		proxy, ok := t.getColumnStorageOrPanic(col, false, currentTx).(*StorageComputeProxy)
		if !ok || !proxy.isOrdered {
			continue
		}
		seen := false
		for _, existing := range result {
			if existing == proxy {
				seen = true
				break
			}
		}
		if seen {
			continue
		}
		result = append(result, proxy)
	}
	return result
}

// hasOrderedScanProxy keeps the overwhelmingly common non-ORC path free of
// proxy-list storage. It deliberately performs only the same stable column
// lookup appendOrderedScanProxies would perform; ORC scans pay the second pass.
func (t *storageShard) hasOrderedScanProxy(cols []string, currentTx *TxContext) bool {
	if !t.t.hasOrderedColumns.Load() {
		return false
	}
	for _, col := range cols {
		if col == "$recset_contains" || col == "$update" || col == "$break" ||
			strings.HasPrefix(col, "NEW.") || strings.HasPrefix(col, "$invalidate:") ||
			strings.HasPrefix(col, "$increment:") || strings.HasPrefix(col, "$set:") {
			continue
		}
		if _, ok := parseBatchPseudoColName(col); ok {
			continue
		}
		proxy, ok := t.getColumnStorageOrPanic(col, false, currentTx).(*StorageComputeProxy)
		if ok && proxy.isOrdered {
			return true
		}
	}
	return false
}

func (t *storageShard) scanBatch(boundaries scanAccess, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64, int64) {
	akkumulator := neutral
	var outCount int64
	var candidateCount int64
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}

	conditionProgram := scm.PrepareSerialProc(condition)
	conditionAlwaysTrue := scanConditionAlwaysTrue(&conditionProgram, len(conditionCols))
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}
	conditionAlwaysTrue = conditionAlwaysTrue || !hasMutationCallback &&
		scanAccessProvesCondition(conditionCols, condition, boundaries)

	t.ensureLoaded()
	ownsWrite := false
	lockMutationExclusively := hasMutationCallback && !ownsWrite
	writeLocked := false
	if lockMutationExclusively {
		t.lockForMutation(currentTx)
		writeLocked = true
		defer func() {
			if writeLocked {
				t.mu.Unlock()
			}
		}()
		if currentTx != nil {
			currentTx.EnterShardWrite(t)
			defer currentTx.ExitShardWrite(t)
		}
	}
	skipShardReadLock := ownsWrite || lockMutationExclusively
	t.ensureMainCount(skipShardReadLock)
	t.ensureScanAccessColumns(boundaries, skipShardReadLock, currentTx)
	recsetBoundaryCoversCondition := recSetHooksCoverCondition(boundaries, lower, t.t, conditionCols, condition)

	var ccols []ColumnStorage
	var cReaders []ColumnReader
	var cNeedsCachedReader []bool
	var conditionBatchSubidx []int
	var conditionGetters []mapArgGetter
	var cdataset []scm.Scmer
	if !conditionAlwaysTrue {
		ccols = make([]ColumnStorage, len(conditionCols))
		cReaders = make([]ColumnReader, len(conditionCols))
		cNeedsCachedReader = make([]bool, len(conditionCols))
		conditionBatchSubidx = make([]int, len(conditionCols))
		conditionGetters = make([]mapArgGetter, len(conditionCols))
		for i, k := range conditionCols {
			if k == "$recset_contains" {
				fnptr := recSetContainsClosure(t)
				if recsetBoundaryCoversCondition {
					fnptr = recSetAlreadyMatchedClosure()
				}
				conditionGetters[i] = func(id uint32, _ uint32) scm.Scmer {
					return scm.NewClosure(fnptr, id)
				}
				continue
			}
			if subidx, ok := parseBatchPseudoColName(k); ok {
				conditionBatchSubidx[i] = subidx + 1
				continue
			}
			ccols[i] = t.getColumnStorageOrPanic(k, skipShardReadLock, currentTx)
			cReaders[i] = newCachedColumnReaderTx(ccols[i], currentTx)
			if _, ok := ccols[i].(*StorageComputeProxy); ok {
				cNeedsCachedReader[i] = true
			}
		}
		cdataset = make([]scm.Scmer, len(conditionCols))
	}

	var mapperStorage ShardMapReducer
	var mapperWorkspace shardMapReducerWorkspace
	mapper := &mapperStorage
	if stride == 0 && mapReducerCanUseReadWorkspace(callbackCols) {
		prepareReadMapReducerStorage(&mapperStorage, &mapperWorkspace, len(callbackCols))
		t.initReadMapReducer(&mapperStorage, callbackCols, mapReduce, skipShardReadLock, currentTx)
	} else {
		mapper = t.OpenMapReducer(callbackCols, mapReduce, skipShardReadLock, stride, batchdata, currentTx)
	}
	defer mapper.Close()

	locked := false
	if !skipShardReadLock {
		t.mu.RLock()
		locked = true
		if t.t.hasTableLock() {
			t.mu.RUnlock()
			locked = false
			t.t.waitTableLock(ss, querySeqFromTx(currentTx), hasMutationCallback)
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

	buf, pooledFullBuf, pooledPointBuf := acquireScanIDBuffer(defaultScanBufferSize)
	defer releaseScanIDBuffer(pooledFullBuf, pooledPointBuf)
	var batchBuf [1024]uint32
	hadValue := false
	batchCount := len(batchdata) / stride
	batchBoundaries := hasBatchScanAccess(boundaries)
	var batchAccessScratch *scanAnalyzeScratch
	if batchBoundaries {
		batchAccessScratch = acquireScanAnalyzeScratch()
		defer releaseScanAnalyzeScratch(batchAccessScratch)
	}
	activeBoundaries := scanAccess{plannerFilterCovered: scanAccessCoversResidual(boundaries)}

	for batchid := 0; batchid < batchCount; batchid++ {
		if boundaries.impossibleBatch(stride, batchdata, batchid) {
			continue
		}
		currentBoundaries := boundaries
		activeLower := lower
		activeUpperLast := upperLast
		if batchBoundaries {
			activeBoundaries.native = materializeBatchScanAccessInto(batchAccessScratch.batchBoundaries[:0], boundaries, stride, batchdata, uint32(batchid))
			currentBoundaries = activeBoundaries
			activeLower, activeUpperLast = indexFromScanAccessInto(batchAccessScratch.lower[:0], currentBoundaries)
		}

		t.iterateIndex(currentTx, currentBoundaries, activeLower, activeUpperLast, maxInsertIndex, buf, 1, nil, func(batch []uint32) bool {
			candidateCount += int64(len(batch))
			outN := t.filterVisibleBatchedScanBatch(batch, batchBuf[:], uint32(batchid), visibleUpper, hasMutationCallback, currentTx, mutationSeen)
			if !conditionAlwaysTrue {
				filteredN := 0
				for _, effectiveIdx := range batch[:outN] {
					if effectiveIdx < t.main_count {
						for i, k := range ccols {
							if subidx := conditionBatchSubidx[i] - 1; subidx >= 0 {
								cdataset[i] = batchdata[batchid*stride+subidx]
							} else if getter := conditionGetters[i]; getter != nil {
								cdataset[i] = getter(effectiveIdx, uint32(batchid))
							} else if cNeedsCachedReader[i] {
								cdataset[i] = cReaders[i].GetValue(effectiveIdx)
							} else {
								cdataset[i] = k.GetValue(effectiveIdx)
							}
						}
					} else {
						for i, k := range conditionCols {
							if subidx := conditionBatchSubidx[i] - 1; subidx >= 0 {
								cdataset[i] = batchdata[batchid*stride+subidx]
							} else if getter := conditionGetters[i]; getter != nil {
								cdataset[i] = getter(effectiveIdx, uint32(batchid))
							} else if cNeedsCachedReader[i] {
								cdataset[i] = cReaders[i].GetValue(effectiveIdx)
							} else if _, isProxy := ccols[i].(*StorageComputeProxy); isProxy {
								cdataset[i] = ccols[i].GetValue(effectiveIdx)
							} else {
								cdataset[i] = t.getDelta(int(effectiveIdx-t.main_count), k)
							}
						}
					}
					if !scm.ToBool(conditionProgram.Call(cdataset)) {
						continue
					}
					batch[filteredN] = effectiveIdx
					filteredN++
				}
				outN = filteredN
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
		return scm.NewNil(), outCount, candidateCount
	}
	if hasMutationCallback && len(pendingRecids) > 0 {
		if writeLocked {
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
		t.mu.Unlock()
		writeLocked = false
	}
	mapper.FlushSideEffects()
	return akkumulator, outCount, candidateCount
}
