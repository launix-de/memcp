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

const compiledScanAccessVersion = "scan_access_v1"

// scanAnalyzeScratch owns the short-lived physical analyzer output until all
// parallel shard consumers have completed. A pool is preferable to a caller
// stack array here: the shard callback escapes into goroutines, which would
// force the complete stack array onto the heap on every scan. Most SQL scans
// fit in these inline buffers; unusual wide predicates retain the ordinary
// append fallback without changing analyzer semantics.
type scanAnalyzeScratch struct {
	boundaries [scanAnalyzeScratchCapacity]columnboundaries
	lower      [scanAnalyzeScratchCapacity]scm.Scmer
}

var scanAnalyzeScratchPool = sync.Pool{
	New: func() any { return new(scanAnalyzeScratch) },
}

func acquireScanAnalyzeScratch() *scanAnalyzeScratch {
	return scanAnalyzeScratchPool.Get().(*scanAnalyzeScratch)
}

func releaseScanAnalyzeScratch(scratch *scanAnalyzeScratch) {
	clear(scratch.boundaries[:])
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
	if len(v) == 10 {
		if schema, bindings, ok := compileScanAccess(v[3], v[4]); ok {
			v = append(v,
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), schema}),
				oc.OptimizeNoEscapeList(bindings))
		}
	}
	return optimizeScanShared(v, oc, 6, 7, 8, 9)
}

func optimizeScanExists(v []scm.Scmer, oc *scm.OptimizerContext, useResult bool) (scm.Scmer, *scm.TypeDescriptor) {
	if len(v) == 5 {
		if schema, bindings, ok := compileScanAccess(v[3], v[4]); ok {
			v = append(v,
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), schema}),
				scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("list")}, bindings...)))
		}
	}
	return oc.ApplyDefaultOptimization(v, useResult)
}

func scanStaticColumns(expr scm.Scmer) ([]scm.Scmer, bool) {
	if !expr.IsSlice() {
		return nil, false
	}
	items := expr.Slice()
	if len(items) == 2 && scanSymbolIs(items[0], "quote") && items[1].IsSlice() {
		items = items[1].Slice()
	} else if len(items) > 0 && scanSymbolIs(items[0], "list") {
		items = items[1:]
	}
	for _, item := range items {
		if !item.IsString() {
			return nil, false
		}
	}
	return items, true
}

func scanParamColumn(expr scm.Scmer, params, columns []scm.Scmer) (string, bool) {
	for i, param := range params {
		if i >= len(columns) {
			break
		}
		name, named := scanSymbolName(param)
		if named && scanExprIsLambdaParam(expr, name, i) {
			return columns[i].String(), true
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
	if !expr.IsSlice() {
		return false
	}
	items := expr.Slice()
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
	lowerInclusive bool
	upperInclusive bool
}

func compileScanComparison(node scm.Scmer, params, columns []scm.Scmer) (compiledScanBoundary, bool) {
	if !node.IsSlice() {
		return compiledScanBoundary{}, false
	}
	items := node.Slice()
	if len(items) != 3 {
		return compiledScanBoundary{}, false
	}
	operator, named := scanSymbolName(items[0])
	if !named {
		return compiledScanBoundary{}, false
	}
	leftColumn, leftIsColumn := scanParamColumn(items[1], params, columns)
	rightColumn, rightIsColumn := scanParamColumn(items[2], params, columns)
	if leftIsColumn == rightIsColumn {
		return compiledScanBoundary{}, false
	}
	column := leftColumn
	value := items[2]
	reversed := false
	if rightIsColumn {
		column, value, reversed = rightColumn, items[1], true
	}
	if scanExprUsesParams(value, params) {
		return compiledScanBoundary{}, false
	}
	if value.IsNil() {
		return compiledScanBoundary{}, false
	}
	value = scanLiftOutOfLambda(value)
	switch operator {
	case "equal?", "equal??":
		return compiledScanBoundary{kind: "equal", column: column, lower: value, upper: value, lowerInclusive: true, upperInclusive: true}, true
	case "<", "<=", ">", ">=":
		inclusive := operator == "<=" || operator == ">="
		lower := operator == ">" || operator == ">="
		if reversed {
			lower = !lower
		}
		boundary := compiledScanBoundary{kind: "range", column: column}
		if lower {
			boundary.lower, boundary.lowerInclusive = value, inclusive
		} else {
			boundary.upper, boundary.upperInclusive = value, inclusive
		}
		return boundary, true
	default:
		return compiledScanBoundary{}, false
	}
}

func collectCompiledScanBoundaries(node scm.Scmer, params, columns []scm.Scmer, result []compiledScanBoundary) ([]compiledScanBoundary, bool) {
	if node.IsSlice() {
		items := node.Slice()
		if len(items) > 1 && scanSymbolIs(items[0], "and") {
			for _, child := range items[1:] {
				var valid bool
				result, valid = collectCompiledScanBoundaries(child, params, columns, result)
				if !valid {
					return result, false
				}
			}
			return result, true
		}
	}
	if boundary, ok := compileScanComparison(node, params, columns); ok {
		for i, existing := range result {
			if existing.column == boundary.column {
				if existing.kind != "range" || boundary.kind != "range" {
					return result, false
				}
				if !existing.lower.IsNil() && !boundary.lower.IsNil() {
					return result, false
				}
				if !existing.upper.IsNil() && !boundary.upper.IsNil() {
					return result, false
				}
				if !boundary.lower.IsNil() {
					result[i].lower = boundary.lower
					result[i].lowerInclusive = boundary.lowerInclusive
				}
				if !boundary.upper.IsNil() {
					result[i].upper = boundary.upper
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
	columns, columnsOK := scanStaticColumns(columnExpr)
	params, body, lambdaOK := scanLambdaParts(filterExpr)
	if !columnsOK || !lambdaOK || len(params) != len(columns) {
		return scm.NewNil(), nil, false
	}
	compiled, valid := collectCompiledScanBoundaries(body, params, columns, nil)
	if !valid || len(compiled) == 0 {
		return scm.NewNil(), nil, false
	}
	sort.SliceStable(compiled, func(i, j int) bool {
		if (compiled[i].kind == "equal") != (compiled[j].kind == "equal") {
			return compiled[i].kind == "equal"
		}
		return compiled[i].column < compiled[j].column
	})
	schema := make([]scm.Scmer, 0, 2+len(compiled)*5)
	schema = append(schema, scm.NewString(compiledScanAccessVersion), scm.NewInt(int64(len(compiled))))
	bindings := make([]scm.Scmer, 0, len(compiled)*2)
	for _, boundary := range compiled {
		lowerSlot, upperSlot := int64(-1), int64(-1)
		if !boundary.lower.IsNil() {
			lowerSlot = int64(len(bindings))
			bindings = append(bindings, boundary.lower)
		}
		if !boundary.upper.IsNil() {
			if boundary.kind == "equal" {
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
		schema = append(schema,
			scm.NewString(boundary.kind), scm.NewString(boundary.column),
			scm.NewInt(lowerSlot), scm.NewInt(upperSlot), scm.NewInt(flags))
	}
	return scm.NewSlice(schema), bindings, true
}

func bindCompiledScanAccess(schemaValue scm.Scmer, values []scm.Scmer, target boundaries) (boundaries, bool) {
	if !schemaValue.IsSlice() {
		return target, false
	}
	schema := schemaValue.Slice()
	if len(schema) < 2 || schema[0].String() != compiledScanAccessVersion {
		return target, false
	}
	count := int(scm.ToInt(schema[1]))
	if count <= 0 || len(schema) != 2+count*5 {
		panic("scan access schema has an invalid boundary count")
	}
	for offset := 2; offset < len(schema); offset += 5 {
		lowerSlot, upperSlot := int(scm.ToInt(schema[offset+2])), int(scm.ToInt(schema[offset+3]))
		flags := scm.ToInt(schema[offset+4])
		boundary := columnboundaries{
			col:            schema[offset+1].String(),
			lowerInclusive: flags&1 != 0,
			upperInclusive: flags&2 != 0,
		}
		switch schema[offset].String() {
		case "equal":
			boundary.matcher = EqualMatcher
		case "range":
			boundary.matcher = RangeMatcher
		default:
			panic("scan access schema has an unknown matcher")
		}
		if lowerSlot >= 0 {
			if lowerSlot >= len(values) {
				panic("scan access lower slot is out of bounds")
			}
			boundary.lower = values[lowerSlot]
		}
		if upperSlot >= 0 {
			if upperSlot >= len(values) {
				panic("scan access upper slot is out of bounds")
			}
			boundary.upper = values[upperSlot]
		}
		target = append(target, boundary)
	}
	return target, true
}

// tryScanInvariantFilterRewrite selects a row-independent IF branch once per
// scan invocation instead of evaluating it for every candidate row.
func tryScanInvariantFilterRewrite(v []scm.Scmer) scm.Scmer {
	// scan and scan_order share tx/table/filtercols/filterfn at indices 1..4.
	if len(v) < 5 {
		return scm.NewNil()
	}
	lambda, ok := scmerSlice(v[4])
	if !ok || len(lambda) < 3 || !scanSymbolIs(lambda[0], "lambda") {
		return scm.NewNil()
	}
	_, body, ok := scanLambdaParts(v[4])
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
	rewritten[4] = scm.NewSlice([]scm.Scmer{
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
	// scan: [fn, tx, table, filtercols, filterfn, mapcols, mapreduce, neutral, combine, isOuter]
	if len(v) < 10 {
		return scm.NewNil()
	}
	if scm.ToBool(v[9]) {
		return scm.NewNil()
	}
	if !scanFalseNeutral(v[7]) || !scanExistsMapReduce(v[6]) || !scanExistsOrReducer(v[8]) {
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
	return optimizeScanShared(v, oc, 6, 9, 10, 11)
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
func (t *table) scanBufferSize(boundaries boundaries) int {
	if t.hasBoundUniquePoint(boundaries) {
		return uniquePointScanBufferSize
	}
	return defaultScanBufferSize
}

func (t *table) hasBoundUniquePoint(boundaries boundaries) bool {
	for _, unique := range t.Unique {
		covered := true
		for _, col := range unique.Cols {
			matched := false
			for _, boundary := range boundaries {
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

func (t *table) scanExists(currentTx *TxContext, conditionCols []string, condition scm.Scmer) bool {
	return t.scanExistsFrom(currentTx, nil, conditionCols, condition, scm.NewNil(), nil)
}

func (t *table) scanExistsFrom(currentTx *TxContext, source *recSet, conditionCols []string, condition scm.Scmer, accessSchema scm.Scmer, accessValues []scm.Scmer) bool {
	ss := SessionStateFromTx(currentTx)
	querySeq := querySeqFromTx(currentTx)
	touchTempColumns(t, conditionCols, nil)
	var scratch *scanAnalyzeScratch
	var boundaries boundaries
	if !accessSchema.IsNil() || source != nil || conditionMayHaveBoundaries(condition) {
		scratch = acquireScanAnalyzeScratch()
		defer releaseScanAnalyzeScratch(scratch)
		if !accessSchema.IsNil() {
			var compiled bool
			boundaries, compiled = bindCompiledScanAccess(accessSchema, accessValues, scratch.boundaries[:0])
			if !compiled {
				panic("scan_exists received an invalid compiled access schema")
			}
		} else {
			boundaries = extractBoundariesInto(scratch.boundaries[:0], conditionCols, condition)
		}
	}
	if accessSchema.IsNil() {
		reorderByFrequency(boundaries, t)
	}
	boundaries = appendRecSetBoundary(boundaries, source)
	var lowerStorage []scm.Scmer
	if scratch != nil {
		lowerStorage = scratch.lower[:0]
	}
	lower, upperLast := indexFromBoundariesInto(lowerStorage, boundaries)
	for _, b := range boundaries {
		t.AddPartitioningScore([]string{b.col})
	}

	values := scanResultCollector{channelSize: t.shardResultBufferSize()}
	var found atomic.Bool
	done := t.iterateShardsParallel(currentTx, boundaries, func(s *storageShard, solo bool) {
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
		if s.scanExists(boundaries, lower, upperLast, conditionCols, condition, currentTx, ss, &found) {
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
func (t *table) scan(currentTx *TxContext, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, combine scm.Scmer, isOuter bool) scm.Scmer {
	return t.scanWithBatchFrom(currentTx, nil, conditionCols, condition, callbackCols, mapReduce, neutral, combine, isOuter, 0, nil, nil, scm.NewNil(), nil)
}

func (t *table) scanWithBatch(currentTx *TxContext, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, combine scm.Scmer, isOuter bool, stride int, batchdata []scm.Scmer) scm.Scmer {
	return t.scanWithBatchFrom(currentTx, nil, conditionCols, condition, callbackCols, mapReduce, neutral, combine, isOuter, stride, batchdata, nil, scm.NewNil(), nil)
}

func (t *table) scanWithBatchFrom(currentTx *TxContext, source *recSet, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, combine scm.Scmer, isOuter bool, stride int, batchdata []scm.Scmer, requiredBoundaries boundaries, accessSchema scm.Scmer, accessValues []scm.Scmer) scm.Scmer {
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
	var scratch *scanAnalyzeScratch
	var boundaries boundaries
	if !accessSchema.IsNil() || source != nil || conditionMayHaveBoundaries(condition) {
		scratch = acquireScanAnalyzeScratch()
		defer releaseScanAnalyzeScratch(scratch)
		if !accessSchema.IsNil() {
			var compiled bool
			boundaries, compiled = bindCompiledScanAccess(accessSchema, accessValues, scratch.boundaries[:0])
			if !compiled {
				panic("scan received an invalid compiled access schema")
			}
		} else {
			boundaries = extractBoundariesInto(scratch.boundaries[:0], conditionCols, condition)
		}
	}
	boundaries = append(boundaries, requiredBoundaries...)
	if accessSchema.IsNil() {
		reorderByFrequency(boundaries, t)
	}
	boundaries = appendRecSetBoundary(boundaries, source)
	var lowerStorage []scm.Scmer
	if scratch != nil {
		lowerStorage = scratch.lower[:0]
	}
	lower, upperLast := indexFromBoundariesInto(lowerStorage, boundaries)
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
	var candidateCount int64
	values := scanResultCollector{channelSize: t.shardResultBufferSize()}
	done := t.iterateShardsParallel(currentTx, boundaries, func(s *storageShard, solo bool) {
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
		res, shardOutCount, shardCandidateCount := s.scan(boundaries, lower, upperLast, conditionCols, condition, callbackCols, mapReduce, neutral, stride, batchdata, currentTx, ss)
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
		indexColsEnc := boundaryIndexCols(boundaries)
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

func (t *storageShard) scanExists(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState, stop *atomic.Bool) bool {
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

func (t *storageShard) scanFirstRecord(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, currentTx *TxContext, ss *scm.SessionState, stop *atomic.Bool) (uint32, bool) {
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}
	conditionProgram := scm.PrepareSerialProc(condition)
	conditionAlwaysTrue := conditionProgram.Kind == scm.SerialProcConstant && scm.ToBool(conditionProgram.Value)

	t.ensureLoaded()
	skipShardReadLock := t.hasWriteOwnerForTx(currentTx)
	t.ensureMainCount(skipShardReadLock)
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

func (t *storageShard) scan(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64, int64) {
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
	conditionAlwaysTrue := conditionProgram.Kind == scm.SerialProcConstant && scm.ToBool(conditionProgram.Value)
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
	t.ensureMainCount(false)
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

func (t *storageShard) scanBatch(boundaries boundaries, lower []scm.Scmer, upperLast scm.Scmer, conditionCols []string, condition scm.Scmer, callbackCols []string, mapReduce scm.Scmer, neutral scm.Scmer, stride int, batchdata []scm.Scmer, currentTx *TxContext, ss *scm.SessionState) (scm.Scmer, int64, int64) {
	akkumulator := neutral
	var outCount int64
	var candidateCount int64
	if ss == nil {
		ss = SessionStateFromTx(currentTx)
	}

	conditionProgram := scm.PrepareSerialProc(condition)
	conditionAlwaysTrue := conditionProgram.Kind == scm.SerialProcConstant && scm.ToBool(conditionProgram.Value)
	hasMutationCallback := false
	for _, c := range callbackCols {
		if c == "$update" || (len(c) > 11 && c[:11] == "$increment:") {
			hasMutationCallback = true
			break
		}
	}

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
	batchBoundaries := hasBatchBoundaries(boundaries)

	for batchid := 0; batchid < batchCount; batchid++ {
		activeBoundaries := boundaries
		activeLower := lower
		activeUpperLast := upperLast
		if batchBoundaries {
			activeBoundaries = materializeBatchBoundaries(boundaries, stride, batchdata, uint32(batchid))
			activeLower, activeUpperLast = indexFromBoundaries(activeBoundaries)
		}

		t.iterateIndex(currentTx, activeBoundaries, activeLower, activeUpperLast, maxInsertIndex, buf, 1, nil, func(batch []uint32) bool {
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
