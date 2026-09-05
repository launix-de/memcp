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
import "github.com/launix-de/memcp/scm"

const batchCapacityRows = 128

// tryScanBatchRewrite detects a nested scan inside the mapreduce callback of an outer scan
// and rewrites it into a batched version: the outer scan accumulates rows into
// a buffer, and the inner scan becomes scan_batch consuming the buffer via #N
// pseudo-columns. Returns the rewritten AST or nil if the pattern doesn't match.
// tryScanOrderBatchRewrite attempts batch rewrite for scan_order's mapfn.
// DISABLED pending separate review — scan_order batch semantics need the
// ordered-result-preservation path and are different from plain scan. Leave
// scan_order unbatched for now.
func tryScanOrderBatchRewrite(v []scm.Scmer) scm.Scmer {
	return scm.NewNil()
}

func tryScanBatchRewrite(v []scm.Scmer) scm.Scmer {
	// scan: [fn, tx, tbl, accessSchema, accessValues, filtercols, filterfn,
	// mapcols, mapreduce, neutral, combine, isOuter]
	// v[2] (tbl) is always a table reference — shape-agnostic (TagTable at
	// runtime, (table schema tbl) list or tbl:schema:name symbol at optimize
	// time). We trust that and just pass it through unchanged.
	if len(v) < 9 {
		return scm.NewNil()
	}
	if len(v) > 11 && scm.ToBool(v[11]) {
		return scm.NewNil()
	}
	return tryScanBatchRewriteMapReduce(v, 7, 8)
}

// tryScanBatchRewriteMapReduce rewrites a non-reducing fused callback. The
// accumulator parameter must be unused; callbacks that carry query state keep
// their original row-at-a-time semantics.
func tryScanBatchRewriteMapReduce(v []scm.Scmer, mapcolsIdx, mapReduceIdx int) scm.Scmer {

	mapReduceParams, mapBody := extractLambdaParts(v[mapReduceIdx])
	if len(mapReduceParams) == 0 {
		return scm.NewNil()
	}
	accumulator := scmerHeadString(mapReduceParams[0])
	if accumulator == "" || astContainsSymbol(mapBody, map[string]bool{accumulator: true}) {
		return scm.NewNil()
	}

	// The first callback parameter is the accumulator; only physical columns
	// become fields in the batch buffer.
	outerLabels := extractLabels(mapReduceParams[1:])
	stride := len(outerLabels)
	if stride == 0 {
		return scm.NewNil()
	}

	// Skip DML scans: $update and other $ params are functions, not data columns
	for _, name := range outerLabels {
		if len(name) > 0 && name[0] == '$' {
			return scm.NewNil()
		}
	}

	// Find first nested scan in mapfn body (shallow search only)
	innerScanSlice, replacer := findFirstScan(mapBody)
	if innerScanSlice == nil {
		return scm.NewNil()
	}

	// Inner scan — v[2] is always a table reference (see tryScanBatchRewrite);
	// we only check arity and that it's not an outer scan.
	if len(innerScanSlice) < 9 {
		return scm.NewNil()
	}
	if len(innerScanSlice) > 11 && scm.ToBool(innerScanSlice[11]) {
		return scm.NewNil()
	}
	// Batch rewriting delays the inner mapper until a flush. Keep effectful
	// mappers in their original nested pipeline so result emission, cache
	// initialization, and other declared effects retain row-by-row semantics.
	if scanExprMayHaveSideEffects(innerScanSlice[8]) {
		return scm.NewNil()
	}

	// The inner scan must actually reference at least one outer param
	// (otherwise it's a cross-join where batching adds overhead for no gain
	// and can break GROUP BY keytable logic).
	outerParamSet := make(map[string]bool, len(outerLabels))
	for _, name := range outerLabels {
		outerParamSet[name] = true
	}
	hasOuterRef := false
	outerSlots := make(map[int]bool, len(outerLabels))
	for i := range outerLabels {
		outerSlots[i+1] = true
	}
	// Check access values, residual filter, and mapfn for outer references.
	if len(innerScanSlice) > 4 {
		hasOuterRef = hasOuterRef || astContainsSymbol(innerScanSlice[4], outerParamSet) || astContainsOuterSlot(innerScanSlice[4], outerSlots)
	}
	if len(innerScanSlice) > 6 {
		hasOuterRef = hasOuterRef || astContainsSymbol(innerScanSlice[6], outerParamSet) || astContainsOuterSlot(innerScanSlice[6], outerSlots)
	}
	if len(innerScanSlice) > 8 {
		hasOuterRef = hasOuterRef || astContainsSymbol(innerScanSlice[8], outerParamSet) || astContainsOuterSlot(innerScanSlice[8], outerSlots)
	}
	if !hasOuterRef {
		return scm.NewNil()
	}

	// Build replacement mapping: outer param symbol → #N symbol
	replaceMap := make(map[string]string, stride)
	replaceSlots := make(map[int]string, stride)
	batchPseudocols := make([]scm.Scmer, stride)
	batchParams := make([]scm.Scmer, stride)
	for i, name := range outerLabels {
		pseudo := fmt.Sprintf("#%d", i)
		replaceMap[name] = pseudo
		replaceSlots[i+1] = pseudo
		batchPseudocols[i] = scm.NewString(pseudo)
		batchParams[i] = scm.NewSymbol(pseudo)
	}

	// Rewrite inner scan → scan_batch
	rewrittenInner := rewriteInnerScanToBatch(innerScanSlice, batchPseudocols, batchParams, replaceMap, replaceSlots, stride)
	if rewrittenInner == nil {
		return scm.NewNil()
	}

	// Replace inner scan in mapfn body with the rewritten scan_batch
	newBody := replacer(scm.NewSlice(rewrittenInner))

	// Build __inner_flush lambda: (lambda (__batchbuf) newBody)
	innerFlushLambda := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("__batchbuf")}),
		newBody,
	})

	// Build the fused outer callback directly as
	// (lambda (batchdata cols...) ... __batchbuf). The scan supplies batchdata
	// in args[0], so no intermediate mapped row or parameter copy is needed.
	outerMapReduceParams := make([]scm.Scmer, stride+1)
	outerMapReduceParams[0] = scm.NewSymbol("batchdata")
	consValues := make([]scm.Scmer, stride+2)
	consValues[0] = scm.NewSymbol("cons")
	consValues[1] = scm.NewSymbol("__batchbuf0")
	for i, name := range outerLabels {
		outerMapReduceParams[i+1] = scm.NewSymbol(name)
		consValues[i+2] = scm.NewSymbol(name)
	}
	appendValues := []scm.Scmer{scm.NewSymbol("apply"), scm.NewSymbol("append_mut"), scm.NewSlice(consValues)}

	batchCapacity := scm.NewInt(int64(stride * batchCapacityRows))

	outerMapReduce := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice(outerMapReduceParams),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("define"), scm.NewSymbol("__batchbuf0"),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("nil?"), scm.NewSymbol("batchdata")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
					scm.NewSymbol("batchdata")})}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("define"), scm.NewSymbol("__batchbuf"),
				scm.NewSlice(appendValues)}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol(">="),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("count"), scm.NewSymbol("__batchbuf")}),
					batchCapacity}),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("begin"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("__inner_flush"), scm.NewSymbol("__batchbuf")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("reset_mut"), scm.NewSymbol("__batchbuf")})}),
				scm.NewBool(true)}),
			scm.NewSymbol("__batchbuf"),
		}),
	})

	// Build the shard combiner: (lambda (acc shardbuf)
	//   (begin
	//     (if (or (nil? shardbuf) (equal? (count shardbuf) 0)) true (__inner_flush shardbuf))
	//     nil))
	outerCombine := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("acc"), scm.NewSymbol("shardbuf")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("or"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("nil?"), scm.NewSymbol("shardbuf")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("equal?"),
						scm.NewSlice([]scm.Scmer{scm.NewSymbol("count"), scm.NewSymbol("shardbuf")}),
						scm.NewInt(0)})}),
				scm.NewBool(true),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("__inner_flush"), scm.NewSymbol("shardbuf")})}),
			scm.NewNil(),
		}),
	})

	// Build the outer scan call with its fused callback and shard combiner.
	outerArgs := make([]scm.Scmer, 0, mapReduceIdx+4)
	for i := 0; i <= mapcolsIdx; i++ {
		outerArgs = append(outerArgs, v[i]) // scan, tx, schema, tbl, ..., mapcols
	}
	outerArgs = append(outerArgs, outerMapReduce, scm.NewNil(), outerCombine)
	outerArgs = append(outerArgs, scm.NewBool(false)) // isOuter
	outerScan := scm.NewSlice(outerArgs)

	// Wrap: (begin (define __inner_flush ...) outer_scan)
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("begin"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("define"), scm.NewSymbol("__inner_flush"), innerFlushLambda}),
		outerScan,
	})
}

// extractLambdaParts returns (params_slice, body) from a lambda AST node,
// or (nil, nil) if the node is not a lambda. Handles both (lambda (p1 p2) body)
// and (lambda () body) where () may be nil or an empty slice.
func extractLambdaParts(expr scm.Scmer) (params []scm.Scmer, body scm.Scmer) {
	if !expr.IsSlice() {
		return nil, scm.NewNil()
	}
	sl := expr.Slice()
	if len(sl) < 3 {
		return nil, scm.NewNil()
	}
	if scmerHeadString(sl[0]) != "lambda" {
		return nil, scm.NewNil()
	}
	if sl[1].IsSlice() {
		return sl[1].Slice(), sl[2]
	}
	// nil, false, or any other non-slice param list → 0-arity lambda
	return []scm.Scmer{}, sl[2]
}

// extractLabels extracts string names from a lambda parameter list.
func extractLabels(params []scm.Scmer) []string {
	names := make([]string, 0, len(params))
	for _, p := range params {
		if p.IsSymbol() {
			names = append(names, p.String())
		} else if sym, ok := p.Any().(scm.Symbol); ok {
			names = append(names, string(sym))
		} else {
			names = append(names, scm.String(p))
		}
	}
	return names
}

// findFirstScan does a SHALLOW walk of an AST to find the first reachable
// (scan ...) or (scan_batch ...) call that represents the inner table of a
// nested-loop join. Only recurses through begin/begin_mut and if — does NOT
// enter !begin (scalar subselect wrappers), lambda bodies, define/set values,
// or any other constructs.
func findFirstScan(expr scm.Scmer) (scanSlice []scm.Scmer, replacer func(scm.Scmer) scm.Scmer) {
	if !expr.IsSlice() {
		return nil, nil
	}
	sl := expr.Slice()
	if len(sl) == 0 {
		return nil, nil
	}
	headStr := scmerHeadString(sl[0])

	// An existing scan_batch has already crossed this rewrite boundary. Treating
	// it as a fresh scan would nest stride/batch arguments on every optimizer pass.
	if headStr == "scan" {
		return sl, func(replacement scm.Scmer) scm.Scmer { return replacement }
	}

	switch headStr {
	case "begin", "begin_mut":
		for i := 1; i < len(sl); i++ {
			inner, innerReplacer := findFirstScan(sl[i])
			if inner != nil {
				idx := i
				return inner, func(replacement scm.Scmer) scm.Scmer {
					newSl := make([]scm.Scmer, len(sl))
					copy(newSl, sl)
					newSl[idx] = innerReplacer(replacement)
					return scm.NewSlice(newSl)
				}
			}
		}
	case "if":
		for i := 2; i < len(sl); i++ {
			inner, innerReplacer := findFirstScan(sl[i])
			if inner != nil {
				idx := i
				return inner, func(replacement scm.Scmer) scm.Scmer {
					newSl := make([]scm.Scmer, len(sl))
					copy(newSl, sl)
					newSl[idx] = innerReplacer(replacement)
					return scm.NewSlice(newSl)
				}
			}
		}
	}
	// Do NOT recurse into !begin, lambda, define, set, nth, resultrow, or anything else.
	return nil, nil
}

// scmerHeadString extracts the string name of a list head (symbol).
func scmerHeadString(head scm.Scmer) string {
	if head.IsSymbol() {
		return head.String()
	}
	if declaration := scm.DeclarationForValue(head); declaration != nil {
		return declaration.Name
	}
	if sym, ok := head.Any().(scm.Symbol); ok {
		return string(sym)
	}
	return ""
}

// astContainsSymbol checks whether any symbol in the given set appears as a
// free variable reference anywhere in the AST.
func astContainsSymbol(expr scm.Scmer, symbols map[string]bool) bool {
	if expr.IsSymbol() {
		return symbols[expr.String()]
	}
	if sym, ok := expr.Any().(scm.Symbol); ok {
		return symbols[string(sym)]
	}
	if !expr.IsSlice() {
		return false
	}
	for _, child := range expr.Slice() {
		if astContainsSymbol(child, symbols) {
			return true
		}
	}
	return false
}

func astContainsOuterSlot(expr scm.Scmer, slots map[int]bool) bool {
	if !expr.IsSlice() {
		return false
	}
	items := expr.Slice()
	if len(items) == 3 && scanSymbolIs(items[0], "outer") && scm.ToInt(items[1]) == 1 && items[2].IsNthLocalVar() {
		return slots[int(items[2].NthLocalVar())]
	}
	for _, item := range items {
		if astContainsOuterSlot(item, slots) {
			return true
		}
	}
	return false
}

// rewriteInnerScanToBatch rewrites a (scan ...) call to (scan_batch ...) by:
// 1. Changing the head to scan_batch
// 2. Appending #N pseudo-columns to filtercols and mapcols
// 3. Extending filterfn and mapfn lambdas with #N params
// 4. Replacing outer param symbols in filter/map bodies with #N symbols
// 5. Inserting stride and __batchbuf after mapreduce
func rewriteInnerScanToBatch(inner []scm.Scmer, pseudocols, pseudoparams []scm.Scmer, replaceMap map[string]string, replaceSlots map[int]string, stride int) []scm.Scmer {
	// inner = [scan, tx, tbl, accessSchema, accessValues, filtercols, filterfn,
	// mapcols, mapreduce, neutral, combine, isOuter]
	result := make([]scm.Scmer, 0, len(inner)+2)

	// [0] scan_batch
	result = append(result, scm.NewSymbol("scan_batch"))
	// [1..2] tx, tbl
	result = append(result, inner[1], inner[2])
	filterColumns := appendToScmerList(inner[5], pseudocols)
	filterFn := extendAndRewriteLambda(inner[6], pseudoparams, replaceMap, replaceSlots)
	accessSchema, accessValues, ok := rewriteBatchScanAccess(inner[3], inner[4], replaceMap, replaceSlots)
	if !ok {
		return nil
	}
	// A direct SCM caller may provide an empty access schema. Compile the
	// rewritten residual once here so scan_batch still receives the one ABI.
	if schemaItems, schemaOK := scanStaticListElements(accessSchema); schemaOK && scanAccessSchemaIsEmpty(schemaItems) {
		if compiledSchema, bindings, compiled := compileScanAccessMode(filterColumns, filterFn, true); compiled {
			accessSchema = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), compiledSchema})
			accessValues = scanAccessValuesExpr(bindings)
			_, residual := pruneScanResidual(filterColumns, filterFn, true)
			accessSchema = scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), markCoveredScanAccessSchema(compiledSchema, residual)})
		}
	}
	// [3..4] common access schema and runtime values
	result = append(result, accessSchema, accessValues)
	// [5] filtercols: append #N
	result = append(result, filterColumns)
	// [6] filterfn: extend params + replace body symbols
	result = append(result, filterFn)
	// [7] mapcols: append #N
	result = append(result, appendToScmerList(inner[7], pseudocols))
	// [8] mapreduce: extend params + replace body symbols
	result = append(result, extendAndRewriteLambda(inner[8], pseudoparams, replaceMap, replaceSlots))
	// [9] stride
	result = append(result, scm.NewInt(int64(stride)))
	// [10] batchdata (symbol __batchbuf from the flush lambda)
	result = append(result, scm.NewSymbol("__batchbuf"))
	// [11..] neutral, combine, isOuter from original
	for i := 9; i < len(inner) && i <= 11; i++ {
		result = append(result, inner[i])
	}
	return result
}

func scanAccessSchemaIsEmpty(schema []scm.Scmer) bool {
	if len(schema) == 0 {
		return true
	}
	meta, valid := decodeScanAccessHeader(schema[0])
	return valid && meta.count == 0
}

// rewriteBatchScanAccess converts runtime access-value slots which directly
// reference an outer row into the negative #N slots understood by scan_batch.
// More complex outer-dependent access expressions are deliberately rejected:
// they need a distinct vector expression contract, not per-row evaluation in
// the storage loop.
func rewriteBatchScanAccess(schemaExpr, valuesExpr scm.Scmer, mapping map[string]string, slots map[int]string) (scm.Scmer, scm.Scmer, bool) {
	schema, ok := scanStaticListElements(schemaExpr)
	if !ok {
		return scm.NewNil(), scm.NewNil(), false
	}
	if len(schema) == 0 {
		return schemaExpr, valuesExpr, true
	}
	if len(schema) < scanAccessSchemaHeaderSize {
		return scm.NewNil(), scm.NewNil(), false
	}
	if _, valid := decodeScanAccessHeader(schema[0]); !valid {
		return scm.NewNil(), scm.NewNil(), false
	}
	values, ok := scanStaticListElements(valuesExpr)
	if !ok {
		return scm.NewNil(), scm.NewNil(), false
	}
	rewrittenSchema := append([]scm.Scmer(nil), schema...)
	rewrittenValues := append([]scm.Scmer(nil), values...)
	for valueSlot, value := range values {
		rewritten := replaceSymbolsInAST(value, mapping, slots)
		name, named := scanSymbolName(rewritten)
		batchSlot, batch := 0, false
		if named {
			batchSlot, batch = parseBatchPseudoColName(name)
		}
		if !batch {
			if rewritten != value {
				return scm.NewNil(), scm.NewNil(), false
			}
			rewrittenValues[valueSlot] = value
			continue
		}
		rewrittenValues[valueSlot] = scm.NewNil()
		encodedSlot := -2 - batchSlot
		header, valid := decodeScanAccessHeader(rewrittenSchema[0])
		if !valid {
			return scm.NewNil(), scm.NewNil(), false
		}
		boundaryCount := header.count
		for boundary := 0; boundary < boundaryCount; boundary++ {
			base := scanAccessSchemaHeaderSize + boundary*scanAccessBoundaryStride
			meta := decodeScanAccessBoundaryMeta(rewrittenSchema[base+2])
			if meta.lowerSlot == valueSlot {
				meta.lowerSlot = encodedSlot
			}
			if meta.upperSlot == valueSlot {
				meta.upperSlot = encodedSlot
			}
			rewrittenSchema[base+2] = newScanAccessBoundaryMeta(meta.lowerSlot, meta.upperSlot, meta.flags)
		}
	}
	return scm.NewSlice([]scm.Scmer{scm.NewSymbol("quote"), scm.NewSlice(rewrittenSchema)}),
		scm.NewSlice(append([]scm.Scmer{scm.NewSymbol("list")}, rewrittenValues...)), true
}

// appendToScmerList appends extra items to a (list ...) AST node.
func appendToScmerList(listExpr scm.Scmer, extras []scm.Scmer) scm.Scmer {
	if !listExpr.IsSlice() {
		return listExpr
	}
	sl := listExpr.Slice()
	newSl := make([]scm.Scmer, len(sl)+len(extras))
	copy(newSl, sl)
	copy(newSl[len(sl):], extras)
	return scm.NewSlice(newSl)
}

// extendAndRewriteLambda extends a lambda with extra params and replaces
// symbols in its body according to replaceMap.
func extendAndRewriteLambda(lambdaExpr scm.Scmer, extraParams []scm.Scmer, replaceMap map[string]string, replaceSlots map[int]string) scm.Scmer {
	if !lambdaExpr.IsSlice() {
		return lambdaExpr
	}
	sl := lambdaExpr.Slice()
	if len(sl) < 3 || scmerHeadString(sl[0]) != "lambda" {
		return lambdaExpr
	}

	// Extract existing params (may be a list or nil for 0-arity)
	var params []scm.Scmer
	if sl[1].IsSlice() {
		params = sl[1].Slice()
	}
	body := sl[2]

	// Extend params
	newParams := make([]scm.Scmer, len(params)+len(extraParams))
	copy(newParams, params)
	copy(newParams[len(params):], extraParams)

	// Replace symbols in body
	newBody := replaceSymbolsInAST(body, replaceMap, replaceSlots)

	// Handle numvars (4th element): increase by number of extra params
	if len(sl) >= 4 && !sl[3].IsNil() {
		oldNumVars := int(sl[3].Int())
		newNumVars := oldNumVars + len(extraParams)
		return scm.NewSlice([]scm.Scmer{sl[0], scm.NewSlice(newParams), newBody, scm.NewInt(int64(newNumVars))})
	}
	return scm.NewSlice([]scm.Scmer{sl[0], scm.NewSlice(newParams), newBody})
}

// replaceSymbolsInAST walks an AST and replaces symbol references according to the mapping.
func replaceSymbolsInAST(expr scm.Scmer, mapping map[string]string, slots map[int]string) scm.Scmer {
	if expr.IsSymbol() {
		name := expr.String()
		if replacement, ok := mapping[name]; ok {
			return scm.NewSymbol(replacement)
		}
		return expr
	}
	if sym, ok := expr.Any().(scm.Symbol); ok {
		if replacement, okm := mapping[string(sym)]; okm {
			return scm.NewSymbol(replacement)
		}
		return expr
	}
	if !expr.IsSlice() {
		return expr
	}
	sl := expr.Slice()
	if len(sl) == 0 {
		return expr
	}
	if len(sl) == 3 && scanSymbolIs(sl[0], "outer") && scm.ToInt(sl[1]) == 1 && sl[2].IsNthLocalVar() {
		if replacement, ok := slots[int(sl[2].NthLocalVar())]; ok {
			return scm.NewSymbol(replacement)
		}
	}
	// Don't recurse into nested lambda param lists (only body)
	head := sl[0]
	headStr := scmerHeadString(head)
	if headStr == "lambda" && len(sl) >= 3 {
		// Only replace in body (sl[2]), not in params (sl[1])
		newBody := replaceSymbolsInAST(sl[2], mapping, slots)
		if len(sl) >= 4 {
			return scm.NewSlice([]scm.Scmer{sl[0], sl[1], newBody, sl[3]})
		}
		return scm.NewSlice([]scm.Scmer{sl[0], sl[1], newBody})
	}

	changed := false
	newSl := make([]scm.Scmer, len(sl))
	for i, elem := range sl {
		newSl[i] = replaceSymbolsInAST(elem, mapping, slots)
		if newSl[i] != elem {
			changed = true
		}
	}
	if !changed {
		return expr
	}
	return scm.NewSlice(newSl)
}
