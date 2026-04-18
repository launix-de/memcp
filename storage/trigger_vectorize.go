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

import "github.com/launix-de/memcp/scm"

// VectorizeTrigger analyzes a trigger body and produces a vectorized version
// that operates on a batch of rows in a single scan. Returns nil if the trigger
// body cannot be vectorized (falls back to per-row execution).
//
// Currently recognizes the prejoin DELETE pattern:
//
//	(lambda (OLD NEW) (scan nil (table schema tbl) condCols
//	    (lambda (cols...) (equal? col (get_assoc OLD key)))
//	    (list "$update") (lambda ($update) ($update)) + 0 nil false))
//
// Vectorized form:
//
//	(lambda (OLD_batch NEW_batch)
//	    (define vals (map OLD_batch (lambda (OLD) (get_assoc OLD key))))
//	    (scan nil (table schema tbl) condCols
//	        (lambda (cols...) (has? vals col))
//	        (list "$update") (lambda ($update) ($update)) + 0 nil false))
func VectorizeTrigger(triggerFn scm.Scmer) scm.Scmer {
	// Extract the proc body
	if !triggerFn.IsProc() {
		return scm.NewNil()
	}
	defer func() {
		if r := recover(); r != nil {
			// Vectorization failed — not vectorizable
		}
	}()
	proc := triggerFn.Proc()
	body := proc.Body

	// Check if body is a scan call
	if !body.IsSlice() {
		return scm.NewNil()
	}
	items := body.Slice()
	if len(items) < 5 {
		return scm.NewNil()
	}
	// Check for (scan tx table condCols filterFn ...)
	// Handle both symbol "scan" and resolved tagFunc references.
	headName := ""
	if items[0].IsSymbol() {
		headName = items[0].String()
	} else if d := scm.DeclarationForValue(items[0]); d != nil {
		headName = d.Name
	}
	if headName != "scan" {
		return scm.NewNil()
	}

	// Extract filter function (items[4])
	// After eval, this is a Proc. In AST form, it's a (lambda ...) list.
	filterFn := items[4]
	if !filterFn.IsProc() && !filterFn.IsSlice() {
		return scm.NewNil()
	}

	// Check if filter contains (equal? col (get_assoc OLD key)) pattern
	key := extractGetAssocOldKey(filterFn)
	if key == "" {
		return scm.NewNil()
	}

	// Check that this is a $update scan (DELETE pattern)
	if len(items) < 6 {
		return scm.NewNil()
	}
	// Check that callback columns contain "$update" (DELETE pattern).
	// The mapCols can be (list "$update") as AST or ["$update"] as evaluated list.
	if !containsUpdateCol(items[5]) {
		return scm.NewNil()
	}

	// Build vectorized function:
	// (lambda (OLD_batch NEW_batch)
	//   for each OLD in OLD_batch: collect get_assoc OLD key into a set
	//   then do a single scan with has? filter)
	// Pre-compute the column param index at vectorization time (not per-call)
	colParamIdx := findEqualParamIdx(filterFn)
	if colParamIdx < 0 {
		return scm.NewNil()
	}

	return scm.NewFunc(func(args ...scm.Scmer) scm.Scmer {
		// args[0] = OLD_batch (columnar dict-of-lists: {"col": [v1,v2,...], ...})
		// args[1] = NEW_batch (nil for DELETE)
		oldBatch := args[0]
		if oldBatch.IsNil() {
			return scm.NewNil()
		}

		// Extract the column values from the columnar batch via get_assoc
		valsList := scm.Apply(scm.NewSymbol("get_assoc"), oldBatch, scm.NewString(key))
		if valsList.IsNil() {
			return scm.NewNil()
		}

		// Build a fast lookup set for the IN-check
		var vals []scm.Scmer
		if valsList.IsSlice() {
			vals = valsList.Slice()
		} else {
			// Single value (shouldn't happen with columnar, but handle it)
			vals = []scm.Scmer{valsList}
		}
		if len(vals) == 0 {
			return scm.NewNil()
		}

		// Replace the filter function with a batch-aware IN-check
		newFilter := scm.NewFunc(func(filterArgs ...scm.Scmer) scm.Scmer {
			if colParamIdx >= len(filterArgs) {
				return scm.NewBool(false)
			}
			colVal := filterArgs[colParamIdx]
			for _, v := range vals {
				if !scm.Less(colVal, v) && !scm.Less(v, colVal) {
					return scm.NewBool(true)
				}
			}
			return scm.NewBool(false)
		})

		// Build new scan call with the batch filter
		newItems := make([]scm.Scmer, len(items))
		copy(newItems, items)
		newItems[4] = newFilter

		// Execute the single batched scan
		return scm.Eval(scm.NewSlice(newItems), proc.En)
	})
}

// extractGetAssocOldKey checks if a filter function contains the pattern
// (equal? param (get_assoc OLD "key")) and returns the key string.
// Returns "" if the pattern is not found.
func extractGetAssocOldKey(filterFn scm.Scmer) string {
	var body scm.Scmer
	if filterFn.IsProc() {
		body = filterFn.Proc().Body
	} else if filterFn.IsSlice() {
		items := filterFn.Slice()
		if len(items) >= 3 && items[0].IsSymbol() && items[0].String() == "lambda" {
			body = items[2]
		}
	}
	if body.IsNil() || !body.IsSlice() {
		return ""
	}
	return findGetAssocOldInExpr(body)
}

// findGetAssocOldInExpr recursively searches for (get_assoc OLD "key") in an expression.
func findGetAssocOldInExpr(expr scm.Scmer) string {
	if !expr.IsSlice() {
		return ""
	}
	items := expr.Slice()
	if len(items) == 0 {
		return ""
	}

	// Check for (equal? X (get_assoc OLD key)) or (equal? (get_assoc OLD key) X)
	headName := declName(items[0])
	isEqual := headName == "equal?" || headName == "equal??"

	if isEqual && len(items) == 3 {
		if k := extractGetAssocOldFromArg(items[2]); k != "" {
			return k
		}
		if k := extractGetAssocOldFromArg(items[1]); k != "" {
			return k
		}
	}

	// Check for (and ...) — recurse into children
	if declName(items[0]) == "and" {
		for _, child := range items[1:] {
			if k := findGetAssocOldInExpr(child); k != "" {
				return k
			}
		}
	}

	return ""
}

// extractGetAssocOldFromArg checks if an expression is (get_assoc OLD "key").
func extractGetAssocOldFromArg(expr scm.Scmer) string {
	if !expr.IsSlice() {
		return ""
	}
	items := expr.Slice()
	if len(items) != 3 {
		return ""
	}
	// Check for (get_assoc OLD "key")
	if declName(items[0]) != "get_assoc" {
		return ""
	}
	// items[1] should be OLD (a symbol)
	if !items[1].IsSymbol() || items[1].String() != "OLD" {
		// Could be NthLocalVar(0) after optimization
		if !items[1].IsNthLocalVar() || items[1].NthLocalVar() != 0 {
			return ""
		}
	}
	// items[2] should be a string key or a symbol key
	if items[2].IsString() {
		return items[2].String()
	}
	if items[2].IsSymbol() {
		return items[2].String()
	}
	return ""
}

// findEqualParamIdx finds which parameter index in the filter lambda is compared
// to (get_assoc OLD ...). Returns -1 if not found.
func findEqualParamIdx(filterFn scm.Scmer) int {
	var params []scm.Scmer
	var body scm.Scmer
	if filterFn.IsProc() {
		if filterFn.Proc().Params.IsSlice() {
			params = filterFn.Proc().Params.Slice()
		}
		body = filterFn.Proc().Body
	} else if filterFn.IsSlice() {
		items := filterFn.Slice()
		if len(items) >= 3 && items[0].IsSymbol() && items[0].String() == "lambda" {
			if items[1].IsSlice() {
				params = items[1].Slice()
			}
			body = items[2]
		}
	}
	if len(params) == 0 || body.IsNil() {
		return -1
	}
	return findEqualParamInExpr(body, params)
}

func findEqualParamInExpr(expr scm.Scmer, params []scm.Scmer) int {
	if !expr.IsSlice() {
		return -1
	}
	items := expr.Slice()
	if len(items) == 0 {
		return -1
	}

	hn := declName(items[0])
	isEqual := hn == "equal?" || hn == "equal??"

	if isEqual && len(items) == 3 {
		if idx := matchParam(items[1], params); idx >= 0 {
			if extractGetAssocOldFromArg(items[2]) != "" {
				return idx
			}
		}
		if idx := matchParam(items[2], params); idx >= 0 {
			if extractGetAssocOldFromArg(items[1]) != "" {
				return idx
			}
		}
	}

	if hn == "and" {
		for _, child := range items[1:] {
			if idx := findEqualParamInExpr(child, params); idx >= 0 {
				return idx
			}
		}
	}

	return -1
}

// declName returns the declaration name for a Scmer value, handling both
// symbols and resolved tagFunc references.
func declName(v scm.Scmer) string {
	if v.IsSymbol() {
		return v.String()
	}
	if d := scm.DeclarationForValue(v); d != nil {
		return d.Name
	}
	return ""
}

// containsUpdateCol checks if an expression contains "$update" as a string,
// either as a direct element or inside a (list ...) AST node.
func containsUpdateCol(expr scm.Scmer) bool {
	if expr.IsString() && expr.String() == "$update" {
		return true
	}
	if expr.IsSlice() {
		for _, item := range expr.Slice() {
			if containsUpdateCol(item) {
				return true
			}
		}
	}
	return false
}

func matchParam(expr scm.Scmer, params []scm.Scmer) int {
	if expr.IsSymbol() {
		name := expr.String()
		for i, p := range params {
			if p.IsSymbol() && p.String() == name {
				return i
			}
		}
	}
	if expr.IsNthLocalVar() {
		idx := int(expr.NthLocalVar())
		if idx < len(params) {
			return idx
		}
	}
	return -1
}
