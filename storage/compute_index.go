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

import "github.com/launix-de/memcp/scm"

// sessionReadKey recognizes the planner's runtime representation of a
// read-only query binding. Mutating session calls have a third argument and
// are deliberately not query-invariant values.
func sessionReadKey(expr scm.Scmer) (string, bool) {
	if !expr.IsSlice() {
		return "", false
	}
	items := expr.Slice()
	if len(items) != 2 || !items[0].IsSymbol() || items[0].String() != "session" || !items[1].IsString() {
		return "", false
	}
	return items[1].String(), true
}

func hasSessionRead(expr scm.Scmer) bool {
	if _, ok := sessionReadKey(expr); ok {
		return true
	}
	if expr.IsProc() {
		return hasSessionRead(expr.Proc().Body)
	}
	if expr.IsSlice() {
		for _, item := range expr.Slice() {
			if hasSessionRead(item) {
				return true
			}
		}
	}
	return false
}

// hasImplicitComputeContext reports whether a persistent computed-column
// callback reads request-local execution state. Computed values are shared by
// every reader and therefore cannot close over either binding. Query-specific
// values must be materialized as ordinary input columns and passed as callback
// parameters instead.
func hasImplicitComputeContext(expr scm.Scmer) bool {
	return hasImplicitComputeContextUsing(expr, nil)
}

func hasImplicitComputeContextUsing(expr scm.Scmer, bound map[string]bool) bool {
	if expr.IsSourceInfo() {
		return hasImplicitComputeContextUsing(expr.WithoutSourceInfo(), bound)
	}
	if expr.IsProc() {
		proc := expr.Proc()
		return hasImplicitComputeContextUsing(proc.Body, bindComputeParams(bound, proc.Params))
	}
	if expr.IsSymbol() {
		name := expr.String()
		return isComputeContextSymbol(name) && !bound[name]
	}
	if !expr.IsSlice() {
		return false
	}
	items := expr.Slice()
	if len(items) == 0 || items[0].SymbolEquals("quote") {
		return false
	}
	if items[0].SymbolEquals("lambda") && len(items) >= 3 {
		return hasImplicitComputeContextUsing(items[2], bindComputeParams(bound, items[1]))
	}
	for _, item := range items {
		if hasImplicitComputeContextUsing(item, bound) {
			return true
		}
	}
	return false
}

func isComputeContextSymbol(name string) bool {
	return name == "session" || name == "tx" || name == "__memcp_tx"
}

func bindComputeParams(bound map[string]bool, params scm.Scmer) map[string]bool {
	result := make(map[string]bool, len(bound)+4)
	for name, present := range bound {
		result[name] = present
	}
	if params.IsSymbol() {
		result[params.String()] = true
		return result
	}
	if params.IsSlice() {
		for _, param := range params.Slice() {
			if param.IsSymbol() {
				result[param.String()] = true
			}
		}
	}
	return result
}

// containsNthLocalVar reports whether expr contains at least one optimizer-local
// variable reference (var i). This is used to decide whether Proc.NumVars must
// be set for serial execution.
func containsNthLocalVar(expr scm.Scmer) bool {
	if expr.IsNthLocalVar() {
		return true
	}
	if expr.IsSlice() {
		for _, it := range expr.Slice() {
			if containsNthLocalVar(it) {
				return true
			}
		}
	}
	return false
}

// isRawDataset reports whether expr uses only:
//   - param symbols (or NthLocalVar within param range)
//   - constants (int, float, string, bool, nil)
//   - pure function calls (function is not a param reference, all args are rawDataset)
//
// Returns false for outer refs, scan calls, standalone lambdas, and unknown symbols.
func isRawDataset(params []scm.Scmer, expr scm.Scmer) bool {
	// constants
	if expr.IsInt() || expr.IsFloat() || expr.IsString() || expr.IsBool() || expr.IsNil() {
		return true
	}
	// param symbol reference
	if expr.IsSymbol() {
		for _, p := range params {
			if p.IsSymbol() && p.String() == expr.String() {
				return true
			}
		}
		return false // unknown symbol (not a param)
	}
	// NthLocalVar param reference
	if expr.IsNthLocalVar() {
		return int(expr.NthLocalVar()) >= 0 && int(expr.NthLocalVar()) < len(params)
	}
	// Request-local bindings cannot define a shared computed index. The planner
	// must project them into explicit row parameters before index analysis.
	if _, ok := sessionReadKey(expr); ok {
		return false
	}
	// function call: look up declaration and require Foldable.
	// DeclarationForValue handles both unoptimized (symbol) and
	// optimizer-resolved (tagFunc/tagFuncEnv) forms via the same path.
	if expr.IsSlice() {
		items := expr.Slice()
		if len(items) == 0 {
			return true
		}
		// calling a param as function is not safe
		if items[0].IsNthLocalVar() {
			return false
		}
		// !list special form: pure alloc-free optimization of (list expr...).
		// Valid when count == number of value exprs. Check only the value exprs for rawDataset.
		if items[0].SymbolEquals("!list") && len(items) >= 3 {
			count := int(scm.ToInt(items[2]))
			if count == len(items)-3 {
				for _, item := range items[3:] {
					if !isRawDataset(params, item) {
						return false
					}
				}
				return true
			}
		}
		// the function must have a foldable declaration
		decl := scm.DeclarationForValue(items[0])
		if decl == nil || !decl.IsFoldable() {
			return false
		}
		// all arguments must be rawDataset
		for _, item := range items[1:] {
			if !isRawDataset(params, item) {
				return false
			}
		}
		return true
	}
	return false
}

// isIndependent reports whether expr does NOT reference any param symbol or NthLocalVar.
// Constants, outer refs, and pure function calls on independent args are OK.
func isIndependent(params []scm.Scmer, expr scm.Scmer) bool {
	// constants
	if expr.IsInt() || expr.IsFloat() || expr.IsString() || expr.IsBool() || expr.IsNil() {
		return true
	}
	// param symbol reference — NOT independent
	if expr.IsSymbol() {
		for _, p := range params {
			if p.IsSymbol() && p.String() == expr.String() {
				return false
			}
		}
		return true // not a param → outer var or global function
	}
	// NthLocalVar in param range — NOT independent
	if expr.IsNthLocalVar() {
		idx := int(expr.NthLocalVar())
		return idx < 0 || idx >= len(params)
	}
	// function call or list
	if expr.IsSlice() {
		items := expr.Slice()
		if len(items) == 0 {
			return true
		}
		// (outer ...) is independent
		if items[0].SymbolEquals("outer") {
			return true
		}
		for _, item := range items {
			if !isIndependent(params, item) {
				return false
			}
		}
		return true
	}
	// Proc: might capture outer state; conservatively not independent
	if expr.IsProc() {
		return false
	}
	return true
}

// hasExplicitOuterReference reports whether expr contains an optimizer-lowered
// capture. The expression is part of a procedure body, so evaluating such a
// capture must start in a synthetic call frame rather than directly in the
// procedure's creation environment.
func hasExplicitOuterReference(expr scm.Scmer) bool {
	if expr.IsProc() {
		return false
	}
	if !expr.IsSlice() {
		return false
	}
	if _, _, ok := scanOuterReference(expr); ok {
		return true
	}
	for _, item := range expr.Slice() {
		if hasExplicitOuterReference(item) {
			return true
		}
	}
	return false
}

// evalIndependentScmer evaluates an expression that doesn't depend on row params.
// Returns (value, true) when evaluation succeeds with a scalar result.
func evalIndependentScmer(expr scm.Scmer, env *scm.Env) (result scm.Scmer, ok bool) {
	// fast path: literal
	if expr.IsInt() || expr.IsFloat() || expr.IsString() {
		return expr, true
	}
	// nil literal
	if expr.IsNil() {
		return expr, true
	}
	// bool literal
	if expr.IsBool() {
		return expr, true
	}
	// symbol: look up in env chain
	if expr.IsSymbol() {
		e := env.FindRead(scm.Symbol(expr.String()))
		if e != nil {
			if val, exists := e.Vars[scm.Symbol(expr.String())]; exists {
				if val.IsInt() || val.IsFloat() || val.IsString() || val.IsBool() {
					return val, true
				}
			}
		}
		return scm.NewNil(), false
	}
	// (outer depth sym): look up sym in the selected environment
	if expr.IsSlice() {
		if depth, value, valid := scanOuterReference(expr); valid && depth > 0 {
			value = value.WithoutSourceInfo()
			if !value.IsSymbol() {
				return scm.NewNil(), false
			}
			for level := 1; level < depth && env != nil; level++ {
				env = env.Outer
			}
			if env == nil {
				return scm.NewNil(), false
			}
			sym := scm.Symbol(value.String())
			e := env.FindRead(sym)
			if e != nil {
				if val, exists := e.Vars[sym]; exists {
					if val.IsInt() || val.IsFloat() || val.IsString() || val.IsBool() {
						return val, true
					}
				}
			}
			return scm.NewNil(), false
		}
	}
	// !list special form: (!list NthLocalVar(start) count expr...)
	// Evaluate items[3:] directly without needing VarsNumbered context.
	if expr.IsSlice() {
		items2 := expr.Slice()
		if len(items2) >= 3 && items2[0].SymbolEquals("!list") {
			count := int(scm.ToInt(scm.Scmer(items2[2])))
			vals := make([]scm.Scmer, 0, count)
			for i := 0; i < count && i+3 < len(items2); i++ {
				v, ok2 := evalIndependentScmer(items2[i+3], env)
				if !ok2 {
					return scm.NewNil(), false
				}
				vals = append(vals, v)
			}
			return scm.NewSlice(vals), true
		}
	}
	// general case: try Eval (for pure function calls like YEAR(NOW()))
	defer func() {
		if r := recover(); r != nil {
			result = scm.NewNil()
			ok = false
		}
	}()
	res := scm.Eval(expr, env)
	if res.IsInt() || res.IsFloat() || res.IsString() || res.IsBool() || res.IsNil() {
		return res, true
	}
	return scm.NewNil(), false
}

// evalIndependentProcBodyScmer evaluates an independent fragment extracted
// from proc.Body. An explicit (outer 1 ...) is relative to the procedure call
// frame which normally exists while the body runs, not directly to proc.En.
// Build that otherwise-empty frame only for expressions which need it; common
// literal and session-read boundaries retain the allocation-free path.
func evalIndependentProcBodyScmer(expr scm.Scmer, proc *scm.Proc) (scm.Scmer, bool) {
	captureBase, captures := proc.JITCapturedLocals()
	if len(captures) == 0 && !hasExplicitOuterReference(expr) {
		return evalIndependentScmer(expr, proc.En)
	}
	bodyEnv := scm.Env{Outer: proc.En}
	if len(captures) != 0 {
		bodyEnv.VarsNumbered = make([]scm.Scmer, captureBase+len(captures))
		copy(bodyEnv.VarsNumbered[captureBase:], captures)
	}
	return evalIndependentScmer(expr, &bodyEnv)
}

// canonicalColName builds a stable canonical name for a computed index column.
// The name starts with "." to distinguish it from real column names.
func canonicalColName(expr scm.Scmer, params []scm.Scmer, conditionCols []string) string {
	return "." + encodeScmerToString(expr, conditionCols, params)
}

// buildComputedFn builds a compute function for a rawDataset formula expression.
// It returns the list of input column names (mapCols) and a callable mapFn.
// mapFn is called with values for mapCols in order and returns the computed value.
// Returns (nil, nil-Scmer) if the formula cannot be compiled.
func buildComputedFn(formulaExpr scm.Scmer, origParams scm.Scmer, env *scm.Env, conditionCols []string) (mapCols []string, mapFn scm.Scmer) {
	if !origParams.IsSlice() {
		return nil, scm.NewNil()
	}
	params := origParams.Slice()
	for i, col := range conditionCols {
		if isScanPseudoColName(col) && computedExprUsesParameter(formulaExpr, params, i) {
			// Pseudo columns are bound per scan run and cannot become persistent
			// computed index inputs. Their own analyzer remains available as a
			// row matcher boundary.
			return nil, scm.NewNil()
		}
	}
	// De-optimize any !list special forms back to plain (list ...) so the lambda
	// does not depend on VarsNumbered slots beyond its params.
	formulaExpr = scm.DeoptimizeExpr(formulaExpr)
	// Build (lambda origParams formulaExpr) in the proc's environment so
	// outer variable references are preserved.
	lambdaForm := scm.NewSlice([]scm.Scmer{
		scm.NewSymbol("lambda"),
		origParams,
		formulaExpr,
	})
	var result scm.Scmer
	func() {
		defer func() { recover() }()
		result = scm.Eval(lambdaForm, env)
	}()
	if result.IsNil() {
		return nil, scm.NewNil()
	}
	// The body may already contain NthLocalVar references (when the condition lambda
	// was pre-compiled by the optimizer). Ensure NumVars is set so that
	// OptimizeProcToSerialFunction uses VarsNumbered instead of Vars[sym], which
	// would leave NthLocalVar(i) unresolvable and cause an index-out-of-range panic.
	if result.IsProc() {
		// Important: only set NumVars when the body actually uses NthLocalVar.
		// For symbol-based bodies, forcing NumVars would skip symbol bindings in
		// OptimizeProcToSerialFunction and make every param read as nil.
		if containsNthLocalVar(result.Proc().Body) {
			result.Proc().NumVars = len(conditionCols)
		} else {
			result.Proc().NumVars = 0
		}
	}
	// mapCols = all conditionCols (lambda takes all params in order)
	return conditionCols, result
}

func computedExprUsesParameter(expr scm.Scmer, params []scm.Scmer, index int) bool {
	expr = expr.WithoutSourceInfo()
	if expr.IsNthLocalVar() {
		return int(expr.NthLocalVar()) == index
	}
	if expr.IsSymbol() && index < len(params) {
		param := params[index].WithoutSourceInfo()
		return param.IsSymbol() && expr.String() == param.String()
	}
	items, ok := scmerSlice(expr)
	if !ok || len(items) == 0 || scanSymbolIs(items[0], "quote") {
		return false
	}
	for _, item := range items {
		if computedExprUsesParameter(item, params, index) {
			return true
		}
	}
	return false
}
