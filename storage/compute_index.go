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
		if items[0].IsSymbol() && items[0].String() == "!list" && len(items) >= 3 {
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
		if decl == nil || !decl.Foldable {
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
		if items[0].IsSymbol() && items[0].String() == "outer" {
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
				if val.IsInt() || val.IsFloat() || val.IsString() {
					return val, true
				}
			}
		}
		return scm.NewNil(), false
	}
	// (outer sym): look up sym in env
	if expr.IsSlice() {
		items := expr.Slice()
		if len(items) == 2 && items[0].IsSymbol() && items[0].String() == "outer" {
			sym := scm.Symbol(items[1].String())
			e := env.FindRead(sym)
			if e != nil {
				if val, exists := e.Vars[sym]; exists {
					if val.IsInt() || val.IsFloat() || val.IsString() {
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
		if len(items2) >= 3 && items2[0].IsSymbol() && items2[0].String() == "!list" {
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
		result.Proc().NumVars = len(conditionCols)
	}
	// mapCols = all conditionCols (lambda takes all params in order)
	return conditionCols, result
}
