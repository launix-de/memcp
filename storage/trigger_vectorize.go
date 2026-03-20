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
//   (lambda (OLD NEW) (scan schema tbl condCols
//       (lambda (cols...) (equal? col (get_assoc OLD key)))
//       (list "$update") (lambda ($update) ($update)) + 0 nil false))
//
// Vectorized form:
//   (lambda (OLD_batch NEW_batch)
//       (define vals (map OLD_batch (lambda (OLD) (get_assoc OLD key))))
//       (scan schema tbl condCols
//           (lambda (cols...) (has? vals col))
//           (list "$update") (lambda ($update) ($update)) + 0 nil false))
func VectorizeTrigger(triggerFn scm.Scmer) scm.Scmer {
	// Extract the proc body
	if !triggerFn.IsProc() {
		return scm.NewNil()
	}
	defer func() {
		if r := recover(); r != nil {
			// Vectorization failed — not vectorizable, return nil
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
	// Check for (scan schema tbl condCols filterFn ...)
	isScan := (items[0].IsSymbol() && items[0].String() == "scan") ||
		(scm.DeclarationForValue(items[0]) != nil && scm.DeclarationForValue(items[0]).Name == "scan")
	if !isScan {
		return scm.NewNil()
	}

	// Extract filter function (items[4])
	filterFn := items[4]
	if !filterFn.IsProc() && !filterFn.IsSlice() {
		return scm.NewNil()
	}

	// Check if filter contains (equal? col (get_assoc OLD key)) pattern
	// and extract the key
	key := extractGetAssocOldKey(filterFn)
	if key == "" {
		return scm.NewNil()
	}

	// Check that this is a $update scan (DELETE pattern)
	if len(items) < 6 {
		return scm.NewNil()
	}
	mapCols := items[5]
	if !mapCols.IsSlice() {
		return scm.NewNil()
	}
	mapColsSlice := mapCols.Slice()
	hasUpdate := false
	for _, mc := range mapColsSlice {
		if mc.IsString() && mc.String() == "$update" {
			hasUpdate = true
		}
	}
	if !hasUpdate {
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
	isEqual := false
	if items[0].IsSymbol() {
		name := items[0].String()
		isEqual = name == "equal?" || name == "equal??"
	}
	if !isEqual {
		d := scm.DeclarationForValue(items[0])
		isEqual = d != nil && (d.Name == "equal?" || d.Name == "equal??")
	}

	if isEqual && len(items) == 3 {
		if k := extractGetAssocOldFromArg(items[2]); k != "" {
			return k
		}
		if k := extractGetAssocOldFromArg(items[1]); k != "" {
			return k
		}
	}

	// Check for (and ...) — recurse into children
	isAnd := items[0].IsSymbol() && items[0].String() == "and"
	if isAnd {
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
	isGetAssoc := false
	if items[0].IsSymbol() {
		isGetAssoc = items[0].String() == "get_assoc"
	}
	if !isGetAssoc {
		d := scm.DeclarationForValue(items[0])
		isGetAssoc = d != nil && d.Name == "get_assoc"
	}
	if !isGetAssoc {
		return ""
	}
	// items[1] should be OLD (a symbol)
	if !items[1].IsSymbol() || items[1].String() != "OLD" {
		// Could be NthLocalVar(0) after optimization
		if !items[1].IsNthLocalVar() || items[1].NthLocalVar() != 0 {
			return ""
		}
	}
	// items[2] should be a string key
	if !items[2].IsString() {
		return ""
	}
	return items[2].String()
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

	isEqual := false
	if items[0].IsSymbol() {
		name := items[0].String()
		isEqual = name == "equal?" || name == "equal??"
	}
	if !isEqual {
		d := scm.DeclarationForValue(items[0])
		isEqual = d != nil && (d.Name == "equal?" || d.Name == "equal??")
	}

	if isEqual && len(items) == 3 {
		// Check which side is a param reference
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

	// Recurse into (and ...)
	if items[0].IsSymbol() && items[0].String() == "and" {
		for _, child := range items[1:] {
			if idx := findEqualParamInExpr(child, params); idx >= 0 {
				return idx
			}
		}
	}

	return -1
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
