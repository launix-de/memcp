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

// tryScanBatchRewrite detects a nested scan inside the mapfn of an outer scan
// and rewrites it into a batched version: the outer scan accumulates rows into
// a buffer, and the inner scan becomes scan_batch consuming the buffer via #N
// pseudo-columns. Returns the rewritten AST or nil if the pattern doesn't match.
func tryScanBatchRewrite(v []scm.Scmer) scm.Scmer {
	// v = [scan, tx, schema, tbl, filtercols, filterfn, mapcols, mapfn, reduce, neutral, reduce2, isOuter]
	if len(v) < 8 {
		return scm.NewNil()
	}
	// Must be a real table (string), not a list/virtual table
	if !v[3].IsString() {
		return scm.NewNil()
	}
	// Must not be outer join
	if len(v) > 11 && scm.ToBool(v[11]) {
		return scm.NewNil()
	}
	// Outer reduce must be nil (no aggregation on outer scan — SELECT, not DML)
	if len(v) > 8 && !v[8].IsNil() {
		return scm.NewNil()
	}

	// mapfn must be a lambda: (lambda (params...) body) or (quote lambda) etc.
	mapParams, mapBody := extractLambdaParts(v[7])
	if mapParams == nil {
		return scm.NewNil()
	}

	// Extract outer param symbol names
	outerParamNames := extractParamNames(mapParams)
	stride := len(outerParamNames)
	if stride == 0 {
		return scm.NewNil()
	}

	// Skip DML scans: $update and other $ params are functions, not data columns
	for _, name := range outerParamNames {
		if len(name) > 0 && name[0] == '$' {
			return scm.NewNil()
		}
	}

	// Find first nested scan in mapfn body
	innerScanSlice, replacer := findFirstScan(mapBody)
	if innerScanSlice == nil {
		return scm.NewNil()
	}

	// Inner scan must also be a real table (not list) and not outer
	if len(innerScanSlice) < 8 || !innerScanSlice[3].IsString() {
		return scm.NewNil()
	}
	if len(innerScanSlice) > 11 && scm.ToBool(innerScanSlice[11]) {
		return scm.NewNil()
	}

	// Build replacement mapping: outer param symbol → #N symbol
	replaceMap := make(map[string]string, stride)
	batchPseudocols := make([]scm.Scmer, stride)
	batchParams := make([]scm.Scmer, stride)
	for i, name := range outerParamNames {
		pseudo := fmt.Sprintf("#%d", i)
		replaceMap[name] = pseudo
		batchPseudocols[i] = scm.NewString(pseudo)
		batchParams[i] = scm.NewSymbol(pseudo)
	}

	// Rewrite inner scan → scan_batch
	rewrittenInner := rewriteInnerScanToBatch(innerScanSlice, batchPseudocols, batchParams, replaceMap, stride)

	// Replace inner scan in mapfn body with the rewritten scan_batch
	newBody := replacer(scm.NewSlice(rewrittenInner))

	// Build __inner_flush lambda: (lambda (__batchbuf) newBody)
	innerFlushLambda := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("__batchbuf")}),
		newBody,
	})

	// Build outer mapfn: (lambda (params...) (begin (define __record (list)) (append_mut __record params...)))
	outerMapParams := make([]scm.Scmer, stride)
	appendArgs := make([]scm.Scmer, stride+2)
	appendArgs[0] = scm.NewSymbol("append_mut")
	appendArgs[1] = scm.NewSymbol("__record")
	for i, name := range outerParamNames {
		outerMapParams[i] = scm.NewSymbol(name)
		appendArgs[i+2] = scm.NewSymbol(name)
	}
	outerMapfn := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice(outerMapParams),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("define"), scm.NewSymbol("__record"), scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")})}),
			scm.NewSlice(appendArgs),
		}),
	})

	batchCapacity := scm.NewInt(int64(stride * batchCapacityRows))

	// Build outer reduce: (lambda (batchdata rowvals)
	//   (begin
	//     (define __batchbuf0 (if (nil? batchdata) (list) batchdata))
	//     (define __batchbuf (apply append_mut (cons __batchbuf0 rowvals)))
	//     (if (>= (count __batchbuf) batch_capacity)
	//       (begin (__inner_flush __batchbuf) (reset_mut __batchbuf))
	//       true)
	//     __batchbuf))
	outerReduce := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("batchdata"), scm.NewSymbol("rowvals")}),
		scm.NewSlice([]scm.Scmer{
			scm.NewSymbol("begin"),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("define"), scm.NewSymbol("__batchbuf0"),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("if"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("nil?"), scm.NewSymbol("batchdata")}),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("list")}),
					scm.NewSymbol("batchdata")})}),
			scm.NewSlice([]scm.Scmer{scm.NewSymbol("define"), scm.NewSymbol("__batchbuf"),
				scm.NewSlice([]scm.Scmer{scm.NewSymbol("apply"), scm.NewSymbol("append_mut"),
					scm.NewSlice([]scm.Scmer{scm.NewSymbol("cons"), scm.NewSymbol("__batchbuf0"), scm.NewSymbol("rowvals")})})}),
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

	// Build outer reduce2: (lambda (acc shardbuf)
	//   (begin
	//     (if (or (nil? shardbuf) (equal? (count shardbuf) 0)) true (__inner_flush shardbuf))
	//     nil))
	outerReduce2 := scm.NewSlice([]scm.Scmer{
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

	// Build the outer scan call
	outerScan := scm.NewSlice([]scm.Scmer{
		v[0], v[1], v[2], v[3], // scan, tx, schema, tbl
		v[4], v[5], v[6], // filtercols, filterfn, mapcols
		outerMapfn,
		outerReduce,
		scm.NewNil(), // neutral
		outerReduce2,
		scm.NewBool(false), // isOuter
	})

	// Wrap: (begin (define __inner_flush ...) outer_scan)
	return scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("begin"),
		scm.NewSlice([]scm.Scmer{scm.NewSymbol("define"), scm.NewSymbol("__inner_flush"), innerFlushLambda}),
		outerScan,
	})
}

// extractLambdaParts returns (params_slice, body) from a lambda AST node,
// or (nil, nil) if the node is not a lambda.
func extractLambdaParts(expr scm.Scmer) (params []scm.Scmer, body scm.Scmer) {
	if !expr.IsSlice() {
		return nil, scm.NewNil()
	}
	sl := expr.Slice()
	if len(sl) < 3 {
		return nil, scm.NewNil()
	}
	head := sl[0]
	isLambda := false
	if head.IsSymbol() && head.String() == "lambda" {
		isLambda = true
	} else if sym, ok := head.Any().(scm.Symbol); ok && string(sym) == "lambda" {
		isLambda = true
	}
	if !isLambda {
		return nil, scm.NewNil()
	}
	if !sl[1].IsSlice() {
		return nil, scm.NewNil()
	}
	return sl[1].Slice(), sl[2]
}

// extractParamNames extracts string names from a lambda parameter list.
func extractParamNames(params []scm.Scmer) []string {
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

// findFirstScan walks an AST to find the first reachable (scan ...) or (scan_batch ...) call.
// Returns the scan's slice and a replacer function that substitutes it in the original tree.
func findFirstScan(expr scm.Scmer) (scanSlice []scm.Scmer, replacer func(scm.Scmer) scm.Scmer) {
	if !expr.IsSlice() {
		return nil, nil
	}
	sl := expr.Slice()
	if len(sl) == 0 {
		return nil, nil
	}
	head := sl[0]
	headStr := ""
	if head.IsSymbol() {
		headStr = head.String()
	} else if sym, ok := head.Any().(scm.Symbol); ok {
		headStr = string(sym)
	}

	// Direct scan/scan_batch match
	if headStr == "scan" || headStr == "scan_batch" {
		return sl, func(replacement scm.Scmer) scm.Scmer { return replacement }
	}

	// Recurse into known SHALLOW wrapper forms only.
	// We only look through begin, if, define, set, nth — NOT into !begin
	// (which wraps scalar subselects) or other complex constructs.
	switch headStr {
	case "begin", "begin_mut":
		// Search through forms for a direct nested scan
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
		// Search then-branch and else-branch
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
	return nil, nil
}

// rewriteInnerScanToBatch rewrites a (scan ...) call to (scan_batch ...) by:
// 1. Changing the head to scan_batch
// 2. Appending #N pseudo-columns to filtercols and mapcols
// 3. Extending filterfn and mapfn lambdas with #N params
// 4. Replacing outer param symbols in filter/map bodies with #N symbols
// 5. Inserting stride and __batchbuf after mapfn
func rewriteInnerScanToBatch(inner []scm.Scmer, pseudocols, pseudoparams []scm.Scmer, replaceMap map[string]string, stride int) []scm.Scmer {
	// inner = [scan, tx, schema, tbl, filtercols, filterfn, mapcols, mapfn, reduce, neutral, reduce2, isOuter]
	result := make([]scm.Scmer, 0, len(inner)+2)

	// [0] scan_batch
	result = append(result, scm.NewSymbol("scan_batch"))
	// [1..3] tx, schema, tbl
	result = append(result, inner[1], inner[2], inner[3])
	// [4] filtercols: append #N
	result = append(result, appendToScmerList(inner[4], pseudocols))
	// [5] filterfn: extend params + replace body symbols
	result = append(result, extendAndRewriteLambda(inner[5], pseudoparams, replaceMap))
	// [6] mapcols: append #N
	result = append(result, appendToScmerList(inner[6], pseudocols))
	// [7] mapfn: extend params + replace body symbols
	result = append(result, extendAndRewriteLambda(inner[7], pseudoparams, replaceMap))
	// [8] stride
	result = append(result, scm.NewInt(int64(stride)))
	// [9] batchdata (symbol __batchbuf from the flush lambda)
	result = append(result, scm.NewSymbol("__batchbuf"))
	// [10..] reduce, neutral, reduce2, isOuter from original
	for i := 8; i < len(inner); i++ {
		result = append(result, inner[i])
	}
	return result
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
func extendAndRewriteLambda(lambdaExpr scm.Scmer, extraParams []scm.Scmer, replaceMap map[string]string) scm.Scmer {
	params, body := extractLambdaParts(lambdaExpr)
	if params == nil {
		return lambdaExpr
	}
	// Extend params
	newParams := make([]scm.Scmer, len(params)+len(extraParams))
	copy(newParams, params)
	copy(newParams[len(params):], extraParams)
	// Replace symbols in body
	newBody := replaceSymbolsInAST(body, replaceMap)
	// Reconstruct lambda — preserve head form (symbol vs quote)
	sl := lambdaExpr.Slice()
	if len(sl) >= 4 {
		// (lambda params body numvars) — also has numvars
		return scm.NewSlice([]scm.Scmer{sl[0], scm.NewSlice(newParams), newBody, sl[3]})
	}
	return scm.NewSlice([]scm.Scmer{sl[0], scm.NewSlice(newParams), newBody})
}

// replaceSymbolsInAST walks an AST and replaces symbol references according to the mapping.
func replaceSymbolsInAST(expr scm.Scmer, mapping map[string]string) scm.Scmer {
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
	// Don't recurse into nested lambda param lists (only body)
	head := sl[0]
	headStr := ""
	if head.IsSymbol() {
		headStr = head.String()
	} else if sym, ok := head.Any().(scm.Symbol); ok {
		headStr = string(sym)
	}
	if headStr == "lambda" && len(sl) >= 3 {
		// Only replace in body (sl[2]), not in params (sl[1])
		newBody := replaceSymbolsInAST(sl[2], mapping)
		if len(sl) >= 4 {
			return scm.NewSlice([]scm.Scmer{sl[0], sl[1], newBody, sl[3]})
		}
		return scm.NewSlice([]scm.Scmer{sl[0], sl[1], newBody})
	}

	changed := false
	newSl := make([]scm.Scmer, len(sl))
	for i, elem := range sl {
		newSl[i] = replaceSymbolsInAST(elem, mapping)
		if newSl[i] != elem {
			changed = true
		}
	}
	if !changed {
		return expr
	}
	return scm.NewSlice(newSl)
}
