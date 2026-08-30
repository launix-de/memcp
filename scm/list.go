/*
Copyright (C) 2023-2026  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

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
/*
 * A minimal Scheme interpreter, as seen in lis.py and SICP
 * http://norvig.com/lispy.html
 * http://mitpress.mit.edu/sicp/full-text/sicp/book/node77.html
 *
 * Pieter Kelchtermans 2013
 * LICENSE: WTFPL 2.0
 */
package scm

import "fmt"
import "runtime"
import "strconv"
import "sync"
import "sync/atomic"
import "github.com/carli2/hybridsort"
import "github.com/jtolds/gls"

func descriptorWithLength(td *TypeDescriptor, length int) *TypeDescriptor {
	if td == nil {
		td = &TypeDescriptor{}
	}
	out := *td
	if length > 0 {
		out.Length = length
	} else {
		out.Length = UnknownLength
	}
	return &out
}

// setOptimizedCallLength annotates the fresh top-level descriptor returned for
// the current operator call. Unlike child descriptors projected from TypeInfo,
// this descriptor is exclusively owned by the caller hook.
func setOptimizedCallLength(td *TypeDescriptor, length int) *TypeDescriptor {
	if td == nil {
		td = &TypeDescriptor{}
	}
	if length > 0 {
		td.Length = length
	} else {
		td.Length = UnknownLength
	}
	return td
}

func setOptimizedCallElement(td *TypeDescriptor, element TypeInfo) *TypeDescriptor {
	// Structured callback metadata already belongs to the recursively optimized
	// child. The top-level call descriptor is fresh, so only the immutable child
	// subtree is shared. Atomic element facts remain in compact TypeInfo until
	// TypeInfo itself has an inline structured representation.
	if td == nil || element.Extra == nil {
		return td
	}
	td.Element = element.Extra
	return td
}

func callbackResultType(expr Scmer, callback TypeInfo) TypeInfo {
	if callback.Kind() == KindFunc && callback.Extra != nil && callback.Extra.Return != nil {
		return TypeInfoFromTD(callback.Extra.Return)
	}
	if callback.Kind() != KindAny || callback.Extra != nil {
		return callback
	}
	if decl := DeclarationForValue(expr); decl != nil && decl.Type != nil && decl.Type.Return != nil {
		return TypeInfoFromTD(decl.Type.Return)
	}
	return callback
}

func materializeCodeLiteral(expr Scmer) (Scmer, bool) {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if inner, ok := scmerSlice(expr); ok {
		if len(inner) == 2 && scmerIsSymbol(inner[0], "quote") {
			materialized, _ := materializeCodeLiteral(inner[1])
			return materialized, true
		}
		if len(inner) == 2 && scmerIsSymbol(inner[0], "symbol") {
			if sym, ok := scmerSymbol(inner[1]); ok {
				return NewSymbol(string(sym)), true
			}
			if inner[1].IsString() {
				return NewSymbol(inner[1].String()), true
			}
		}
		out := make([]Scmer, len(inner))
		changed := false
		for i, item := range inner {
			materialized, itemChanged := materializeCodeLiteral(item)
			out[i] = materialized
			changed = changed || itemChanged
		}
		if changed {
			return NewSlice(out), true
		}
	}
	return expr, false
}

// listExpressionCallName recognizes both source symbols and native procedure
// heads produced by recursive optimization. Length analysis runs after child
// optimization, so treating a native `list` head as list data adds the call
// head itself to the inferred length.
func listExpressionCallName(head Scmer) (string, bool) {
	if sym, ok := scmerSymbol(head); ok {
		return string(sym), true
	}
	if decl := DeclarationForValue(head); decl != nil {
		return decl.Name, true
	}
	return "", false
}

func exactListLengthFromExpr(expr Scmer) int {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if inner, ok := scmerSlice(expr); ok {
		if len(inner) == 0 {
			return UnknownLength
		}
		if callName, ok := listExpressionCallName(inner[0]); ok {
			switch callName {
			case "quote":
				if len(inner) == 2 {
					return exactListLengthFromExpr(inner[1])
				}
				return UnknownLength
			case "!list":
				if len(inner) >= 3 {
					if count := int(ToInt(inner[2])); count >= 0 {
						return count
					}
				}
				return UnknownLength
			case "list":
				return len(inner) - 1
			case "append", "append_mut":
				if len(inner) < 2 {
					return UnknownLength
				}
				baseLength := exactListLengthFromExpr(inner[1])
				if baseLength >= 0 {
					return baseLength + len(inner) - 2
				}
				return UnknownLength
			case "cons":
				if len(inner) == 3 {
					tailLength := exactListLengthFromExpr(inner[2])
					if tailLength >= 0 {
						return tailLength + 1
					}
				}
				return UnknownLength
			case "map", "map_mut", "parallel_map", "parallel_map_mut", "mapIndex", "mapIndex_mut", "reverse", "reverse_mut":
				if len(inner) >= 2 {
					return exactListLengthFromExpr(inner[1])
				}
				return UnknownLength
			case "cdr":
				if len(inner) == 2 {
					sourceLen := exactListLengthFromExpr(inner[1])
					if sourceLen >= 0 {
						return sourceLen - 1
					}
				}
				return UnknownLength
			case "produceN", "produceN_mut", "parallelN", "parallelN_mut":
				if len(inner) >= 2 {
					if count := int(ToInt(inner[1])); count >= 0 {
						return count
					}
				}
				return UnknownLength
			case "merge":
				if len(inner) == 2 {
					arg := inner[1]
					if outer, ok := scmerSlice(arg); ok && len(outer) > 0 {
						outerCall, isCall := listExpressionCallName(outer[0])
						if !isCall || outerCall != "list" {
							return UnknownLength
						}
						total := 0
						for _, item := range outer[1:] {
							itemLen := exactListLengthFromExpr(item)
							if itemLen < 0 {
								return UnknownLength
							}
							total += itemLen
						}
						return total
					}
					return UnknownLength
				}
				total := 0
				for _, arg := range inner[1:] {
					itemLen := exactListLengthFromExpr(arg)
					if itemLen < 0 {
						return UnknownLength
					}
					total += itemLen
				}
				return total
			case "extract_assoc", "extract_assoc_mut":
				if len(inner) >= 2 {
					return exactAssocLengthFromExpr(inner[1])
				}
				return UnknownLength
			case "zip":
				if len(inner) == 2 {
					arg := inner[1]
					if outer, ok := scmerSlice(arg); ok && len(outer) > 0 {
						outerCall, isCall := listExpressionCallName(outer[0])
						if !isCall || outerCall != "list" {
							return UnknownLength
						}
						expected := UnknownLength
						for _, item := range outer[1:] {
							itemLen := exactListLengthFromExpr(item)
							if itemLen < 0 {
								return UnknownLength
							}
							if expected == UnknownLength {
								expected = itemLen
								continue
							}
							if itemLen != expected {
								return UnknownLength
							}
						}
						return expected
					}
					return UnknownLength
				}
				minLen := UnknownLength
				for _, arg := range inner[1:] {
					itemLen := exactListLengthFromExpr(arg)
					if itemLen < 0 {
						return UnknownLength
					}
					if minLen == UnknownLength || itemLen < minLen {
						minLen = itemLen
					}
				}
				return minLen
			}
			if decl := DeclarationForValue(inner[0]); decl != nil && decl.Type != nil && decl.Type.Return != nil && decl.Type.Return.Length > 0 {
				return decl.Type.Return.Length
			}
			return UnknownLength
		}
		return len(inner)
	}
	return UnknownLength
}

func exactAssocLengthFromExpr(expr Scmer) int {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if inner, ok := scmerSlice(expr); ok {
		if len(inner) == 0 {
			return UnknownLength
		}
		if callName, ok := listExpressionCallName(inner[0]); ok {
			switch callName {
			case "quote":
				if len(inner) == 2 {
					return exactAssocLengthFromExpr(inner[1])
				}
				return UnknownLength
			case "!list":
				if len(inner) >= 3 {
					if count := int(ToInt(inner[2])); count >= 0 && count%2 == 0 {
						return count / 2
					}
				}
				return UnknownLength
			case "list":
				if (len(inner)-1)%2 == 0 {
					return (len(inner) - 1) / 2
				}
				return UnknownLength
			case "map_assoc", "map_assoc_mut":
				if len(inner) >= 2 {
					return exactAssocLengthFromExpr(inner[1])
				}
				return UnknownLength
			case "set_assoc", "set_assoc_mut":
				if len(inner) >= 4 {
					sourceLen := exactAssocLengthFromExpr(inner[1])
					_ = sourceLen
				}
				return UnknownLength
			}
			if decl := DeclarationForValue(inner[0]); decl != nil && decl.Type != nil && decl.Type.Return != nil && decl.Type.Return.Length > 0 {
				return decl.Type.Return.Length
			}
		}
	}
	return UnknownLength
}

// exactOptimizedListArgumentLength consumes the code and TypeInfo returned by
// the argument's existing recursive optimizer call. It must not invoke the
// optimizer again merely to recover metadata.
func exactOptimizedListArgumentLength(expr Scmer, ti TypeInfo) int {
	if length := exactListLengthFromExpr(expr); length >= 0 {
		return length
	}
	if materialized, changed := materializeCodeLiteral(expr); changed {
		if length := exactListLengthFromExpr(materialized); length >= 0 {
			return length
		}
	}
	if ti.length >= 0 {
		return ti.length
	}
	if ti.Extra != nil && ti.Extra.Length >= 0 {
		return ti.Extra.Length
	}
	return UnknownLength
}

// exactOptimizedAssocArgumentLength is the assoc counterpart of
// exactOptimizedListArgumentLength and has the same single-visit contract.
func exactOptimizedAssocArgumentLength(expr Scmer, ti TypeInfo) int {
	if length := exactAssocLengthFromExpr(expr); length >= 0 {
		return length
	}
	if materialized, changed := materializeCodeLiteral(expr); changed {
		if length := exactAssocLengthFromExpr(materialized); length >= 0 {
			return length
		}
	}
	if ti.length >= 0 {
		return ti.length
	}
	if ti.Extra != nil && ti.Extra.Length >= 0 {
		return ti.Extra.Length
	}
	return UnknownLength
}

func optimizedArgumentType(types []TypeInfo, index int) TypeInfo {
	if index < 0 || index >= len(types) {
		return tiZero
	}
	return types[index]
}

func exactFlattenedMergeArgumentLength(expr Scmer, ti TypeInfo) int {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	inner, ok := scmerSlice(expr)
	if !ok || len(inner) == 0 {
		return UnknownLength
	}
	if callName, ok := listExpressionCallName(inner[0]); ok {
		switch callName {
		case "quote":
			if len(inner) == 2 {
				return exactFlattenedMergeArgumentLength(inner[1], ti)
			}
			return UnknownLength
		case "list":
			total := 0
			for i, item := range inner[1:] {
				itemType := tiZero
				if ti.Extra != nil && ti.Extra.Keys != nil {
					itemType = TypeInfoFromTD(ti.Extra.Keys[strconv.Itoa(i)])
				}
				itemLen := exactOptimizedListArgumentLength(item, itemType)
				if itemLen < 0 {
					return UnknownLength
				}
				total += itemLen
			}
			return total
		}
	}
	outerLength := exactOptimizedListArgumentLength(expr, ti)
	if outerLength >= 0 && ti.Extra != nil && ti.Extra.Element != nil && ti.Extra.Element.Length >= 0 {
		return outerLength * ti.Extra.Element.Length
	}
	return UnknownLength
}

func exprMayHaveSideEffects(expr Scmer) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	inner, ok := scmerSlice(expr)
	if !ok || len(inner) == 0 {
		return false
	}
	if decl := DeclarationForValue(inner[0]); decl != nil && decl.Type != nil && decl.Type.HasSideEffects {
		return true
	}
	for _, part := range inner[1:] {
		if exprMayHaveSideEffects(part) {
			return true
		}
	}
	return false
}

func optimizeListCall(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	td = descriptorWithLength(td, len(v)-1)
	if !td.Const {
		td.Transfer = true
	}
	return result, td
}

func optimizeCount(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	if len(v) == 2 {
		optimized, argumentType := OptimizeEx(v[1], oc.Env, oc.Ome, true)
		v[1] = optimized
		if length := exactOptimizedListArgumentLength(optimized, argumentType); length >= 0 && !exprMayHaveSideEffects(optimized) {
			return NewInt(int64(length)), &TypeDescriptor{Kind: "int", Transfer: true, Const: true, Length: UnknownLength}
		}
		return NewSlice(v), &TypeDescriptor{Kind: "int", Length: UnknownLength}
	}
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	if rv, ok := scmerSlice(result); ok && len(rv) == 2 {
		if length := exactOptimizedListArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1)); length >= 0 && !exprMayHaveSideEffects(rv[1]) {
			return NewInt(int64(length)), &TypeDescriptor{Kind: "int", Transfer: true, Const: true, Length: UnknownLength}
		}
	}
	return result, td
}

func optimizeFixedLengthInput(mutName string) func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	return func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
		call := oc.applyDefaultOptimizationWithTypes(v, useResult, mutName)
		result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
		if rv, ok := scmerSlice(result); ok && len(rv) >= 2 {
			return result, setOptimizedCallLength(td, exactOptimizedListArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1)))
		}
		return result, td
	}
}

func optimizeAssocFixedLengthInput(mutName string) func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	return func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
		call := oc.applyDefaultOptimizationWithTypes(v, useResult, mutName)
		result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
		if rv, ok := scmerSlice(result); ok && len(rv) >= 2 {
			return result, setOptimizedCallLength(td, exactOptimizedAssocArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1)))
		}
		return result, td
	}
}

func optimizeExtractAssoc(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "extract_assoc_mut")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	if rv, ok := scmerSlice(result); ok && len(rv) >= 2 {
		return result, setOptimizedCallLength(td, exactOptimizedAssocArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1)))
	}
	return result, td
}

func optimizeCdr(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	if rv, ok := scmerSlice(result); ok && len(rv) == 2 {
		if length := exactOptimizedListArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1)); length >= 0 {
			return result, setOptimizedCallLength(td, length-1)
		}
	}
	return result, td
}

func optimizeZip(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 2 {
		return result, td
	}
	if len(rv) == 2 {
		argExpr := rv[1]
		if argList, ok := scmerSlice(argExpr); ok && len(argList) > 0 {
			expected := UnknownLength
			for i, item := range argList[1:] {
				itemType := tiZero
				if outerType := optimizedArgumentType(argumentTypes, 1); outerType.Extra != nil && outerType.Extra.Keys != nil {
					itemType = TypeInfoFromTD(outerType.Extra.Keys[strconv.Itoa(i)])
				}
				itemLen := exactOptimizedListArgumentLength(item, itemType)
				if itemLen < 0 {
					return result, td
				}
				if expected == UnknownLength {
					expected = itemLen
					continue
				}
				if itemLen != expected {
					return result, td
				}
			}
			if expected > 0 {
				return result, setOptimizedCallLength(td, expected)
			}
		}
		return result, td
	}
	minLen := UnknownLength
	for i, arg := range rv[1:] {
		length := exactOptimizedListArgumentLength(arg, optimizedArgumentType(argumentTypes, i+1))
		if length < 0 {
			return result, td
		}
		if minLen == UnknownLength || length < minLen {
			minLen = length
		}
	}
	return result, setOptimizedCallLength(td, minLen)
}

// optimizedMergeSegments returns the list-of-lists input of an optimized merge
// call without materializing its flattened result. Variadic merge calls need a
// small segment catalog; unary calls already carry that catalog as their sole
// argument.
func optimizedMergeSegments(v Scmer) (Scmer, bool) {
	rv, ok := scmerSlice(v)
	if !ok || len(rv) < 2 || !scmerIsSymbol(rv[0], "merge") {
		return NewNil(), false
	}
	if len(rv) == 2 {
		return rv[1], true
	}
	segments := make([]Scmer, 1, len(rv))
	segments[0] = NewSymbol("list")
	segments = append(segments, rv[1:]...)
	return NewSlice(segments), true
}

// mergeValidationSafeDeferredArgument reports whether evaluating a reduce
// argument before merge's list validation is unobservable. Fused calls evaluate
// their reducer and neutral value before validating the segment catalog, whereas
// the unfused spelling finishes evaluating merge first. Dynamic lookups are not
// safe here even when they have already been numbered by the optimizer.
func mergeValidationSafeDeferredArgument(v Scmer, allowLambda bool) bool {
	if stripped, ok := scmerStripSourceInfo(v); ok {
		v = stripped
	}
	inner, ok := scmerSlice(v)
	if !ok {
		return v.IsNil() || v.IsBool() || v.IsInt() || v.IsFloat() || v.IsString()
	}
	if len(inner) == 0 {
		return true
	}
	return scmerIsSymbol(inner[0], "quote") || (allowLambda && scmerIsSymbol(inner[0], "lambda"))
}

// optimizerLambdaWithBody preserves the optimized lambda frame while replacing
// its body. This avoids another optimization or variable-remapping pass.
func optimizerLambdaWithBody(expr Scmer, body Scmer) (Scmer, bool) {
	items, ok := scmerSlice(expr)
	if !ok || len(items) < 3 || !scmerIsSymbol(items[0], "lambda") {
		return NewNil(), false
	}
	rewritten := append([]Scmer(nil), items...)
	rewritten[2] = body
	return NewSlice(rewritten), true
}

// optimizerLambdaParamReference accepts both source symbols and already lowered
// local-variable references.
func optimizerLambdaParamReference(expr Scmer, params []Scmer, index int) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsNthLocalVar() {
		return int(expr.NthLocalVar()) == index
	}
	if index < 0 || index >= len(params) {
		return false
	}
	param, ok := scmerSymbol(params[index])
	return ok && scmerIsSymbol(expr, string(param))
}

// optimizerNonNilPredicate recognizes the planner's exact (not (nil? value))
// filter without analyzing the callback subtree again.
func optimizerNonNilPredicate(expr Scmer) bool {
	params, body, ok := optimizerLambdaParts(expr)
	if !ok || len(params) != 1 {
		return false
	}
	notCall, ok := scmerSlice(body)
	if !ok || len(notCall) != 2 || !scmerIsSymbol(notCall[0], "not") {
		return false
	}
	nilCall, ok := scmerSlice(notCall[1])
	return ok && len(nilCall) == 2 && scmerIsSymbol(nilCall[0], "nil?") &&
		optimizerLambdaParamReference(nilCall[1], params, 0)
}

func optimizerBoolLiteral(expr Scmer) (bool, bool) {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if expr.IsBool() {
		return expr.Bool(), true
	}
	if scmerIsSymbol(expr, "true") {
		return true, true
	}
	if scmerIsSymbol(expr, "false") {
		return false, true
	}
	return false, false
}

func optimizerNilLiteral(expr Scmer) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	return expr.IsNil() || scmerIsSymbol(expr, "nil")
}

func optimizerZeroLiteral(expr Scmer) bool {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	return (expr.IsInt() && expr.Int() == 0) || (expr.IsFloat() && expr.Float() == 0)
}

// optimizePlannerReduceFold lowers common planner folds to physical loops. It
// only inspects the already optimized reducer's root shape, keeping the whole
// optimization pass linear in the expression size.
func optimizePlannerReduceFold(rv []Scmer, td *TypeDescriptor) (Scmer, *TypeDescriptor, bool) {
	if len(rv) != 4 {
		return NewNil(), nil, false
	}
	params, body, ok := optimizerLambdaParts(rv[2])
	if !ok || len(params) < 2 {
		return NewNil(), nil, false
	}
	bodyItems, ok := scmerSlice(body)
	if !ok || len(bodyItems) < 3 {
		return NewNil(), nil, false
	}
	if len(bodyItems) == 3 && scmerIsSymbol(bodyItems[0], "+") &&
		optimizerLambdaParamReference(bodyItems[1], params, 0) && optimizerZeroLiteral(rv[3]) {
		callback, ok := optimizerLambdaWithBody(rv[2], bodyItems[2])
		if !ok {
			return NewNil(), nil, false
		}
		return NewSlice([]Scmer{NewSymbol("sum_map"), rv[1], callback, rv[3]}), &TypeDescriptor{Kind: "number"}, true
	}
	if (scmerIsSymbol(bodyItems[0], "or") || scmerIsSymbol(bodyItems[0], "and")) &&
		optimizerLambdaParamReference(bodyItems[1], params, 0) {
		wantNeutral := scmerIsSymbol(bodyItems[0], "and")
		neutral, ok := optimizerBoolLiteral(rv[3])
		if !ok || neutral != wantNeutral {
			return NewNil(), nil, false
		}
		candidate := bodyItems[2]
		if len(bodyItems) > 3 {
			candidate = NewSlice(append([]Scmer{bodyItems[0]}, bodyItems[2:]...))
		}
		callback, ok := optimizerLambdaWithBody(rv[2], candidate)
		if !ok {
			return NewNil(), nil, false
		}
		name := "reduce_any"
		if wantNeutral {
			name = "reduce_all"
		}
		return NewSlice([]Scmer{NewSymbol(name), rv[1], callback}), &TypeDescriptor{Kind: "bool"}, true
	}

	if len(bodyItems) == 4 && scmerIsSymbol(bodyItems[0], "if") && optimizerNilLiteral(rv[3]) {
		condition, ok := scmerSlice(bodyItems[1])
		if !ok || len(condition) != 2 || !scmerIsSymbol(condition[0], "not") {
			return NewNil(), nil, false
		}
		nilCall, ok := scmerSlice(condition[1])
		if !ok || len(nilCall) != 2 || !scmerIsSymbol(nilCall[0], "nil?") ||
			!optimizerLambdaParamReference(nilCall[1], params, 0) ||
			!optimizerLambdaParamReference(bodyItems[2], params, 0) {
			return NewNil(), nil, false
		}
		callback, ok := optimizerLambdaWithBody(rv[2], bodyItems[3])
		if !ok {
			return NewNil(), nil, false
		}
		return NewSlice([]Scmer{NewSymbol("find_map_notnull"), rv[1], callback}), td, true
	}

	return NewNil(), nil, false
}

// optimizeReduce keeps merge as a segmented producer when its flattened value
// is consumed exactly once by reduce. reduce_segments validates all segments
// before invoking the callback, matching merge's existing error order.
func optimizeReduce(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 3 || len(rv) > 4 || !scmerIsSymbol(rv[0], "reduce") {
		return result, td
	}
	if len(rv) >= 3 {
		if inner, ok := scmerSlice(rv[1]); ok && len(inner) == 3 {
			switch {
			case scmerIsSymbol(inner[0], "map") || scmerIsSymbol(inner[0], "map_mut"):
				if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(rv[2]) || (len(rv) == 4 && exprMayHaveSideEffects(rv[3])) {
					return result, td
				}
				fused := []Scmer{NewSymbol("reduce_map"), inner[1], inner[2], rv[2]}
				if len(rv) == 4 {
					fused = append(fused, rv[3])
				}
				return NewSlice(fused), td
			case scmerIsSymbol(inner[0], "filter") || scmerIsSymbol(inner[0], "filter_mut"):
				if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(rv[2]) || (len(rv) == 4 && exprMayHaveSideEffects(rv[3])) {
					return result, td
				}
				fused := []Scmer{NewSymbol("reduce_filter"), inner[1], inner[2], rv[2]}
				if len(rv) == 4 {
					fused = append(fused, rv[3])
				}
				return NewSlice(fused), td
			}
		}
		if inner, ok := scmerSlice(rv[1]); ok && len(inner) == 4 {
			switch {
			case scmerIsSymbol(inner[0], "filter_map"):
				if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(inner[3]) ||
					exprMayHaveSideEffects(rv[2]) || (len(rv) == 4 && exprMayHaveSideEffects(rv[3])) {
					return result, td
				}
				fused := []Scmer{NewSymbol("reduce_map_filter"), inner[1], inner[2], inner[3], rv[2]}
				if len(rv) == 4 {
					fused = append(fused, rv[3])
				}
				return NewSlice(fused), td
			case scmerIsSymbol(inner[0], "map_filter"):
				if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(inner[3]) ||
					exprMayHaveSideEffects(rv[2]) || (len(rv) == 4 && exprMayHaveSideEffects(rv[3])) {
					return result, td
				}
				fused := []Scmer{NewSymbol("reduce_filter_map"), inner[1], inner[2], inner[3], rv[2]}
				if len(rv) == 4 {
					fused = append(fused, rv[3])
				}
				return NewSlice(fused), td
			}
		}
	}
	if inner, ok := scmerSlice(rv[1]); ok && len(inner) == 3 && scmerIsSymbol(inner[0], "merge") {
		if !mergeValidationSafeDeferredArgument(rv[2], true) ||
			(len(rv) == 4 && !mergeValidationSafeDeferredArgument(rv[3], false)) ||
			exprMayHaveSideEffects(rv[2]) || (len(rv) == 4 && exprMayHaveSideEffects(rv[3])) {
			return result, td
		}
		fused := []Scmer{NewSymbol("reduce_merge2"), inner[1], inner[2], rv[2]}
		if len(rv) == 4 {
			fused = append(fused, rv[3])
		}
		return NewSlice(fused), td
	}
	segments, ok := optimizedMergeSegments(rv[1])
	if ok && mergeValidationSafeDeferredArgument(rv[2], true) &&
		(len(rv) != 4 || mergeValidationSafeDeferredArgument(rv[3], false)) {
		fused := make([]Scmer, 0, len(rv))
		fused = append(fused, NewSymbol("reduce_segments"), segments)
		fused = append(fused, rv[2:]...)
		return NewSlice(fused), td
	}
	if fused, fusedType, ok := optimizePlannerReduceFold(rv, td); ok {
		return fused, fusedType
	}
	return result, td
}

// optimizeMap is the optimizer hook for `map`. It applies default optimization
// (including FirstParameterMutable swap to map_mut), then fuses
// (map (produceN N) fn) → (produceN N fn) to eliminate the intermediate list.
func optimizeMap(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Run default optimization first (handles map → map_mut swap etc.)
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "map_mut")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	elementType := tiZero
	if len(v) >= 3 {
		elementType = callbackResultType(v[2], optimizedArgumentType(argumentTypes, 2))
	}
	// Check if the optimized result is still a call to map/map_mut
	if result.IsSlice() {
		rv := result.Slice()
		if len(rv) == 3 {
			if sym, ok := scmerSymbol(rv[0]); ok && (sym == "map" || sym == "map_mut") {
				// Check if arg 1 is a (produceN N) call
				if rv[1].IsSlice() {
					inner := rv[1].Slice()
					if len(inner) == 2 {
						if isym, ok := scmerSymbol(inner[0]); ok && isym == "produceN" {
							// Fuse: (map (produceN N) fn) → (produceN N fn)
							if count := int(ToInt(inner[1])); count > 0 {
								return NewSlice([]Scmer{inner[0], inner[1], rv[2]}), setOptimizedCallElement(setOptimizedCallLength(td, count), elementType)
							}
							return NewSlice([]Scmer{inner[0], inner[1], rv[2]}), setOptimizedCallElement(td, elementType)
						}
					}
				}
				// Fuse: (map (filter x p) f) → (map_filter x p f)
				if inner, ok := scmerSlice(rv[1]); ok && len(inner) == 3 {
					isym, ok := scmerSymbol(inner[0])
					if !ok {
						return result, td
					}
					switch isym {
					case "filter", "filter_mut":
						if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(rv[2]) {
							return result, td
						}
						return NewSlice([]Scmer{NewSymbol("map_filter"), inner[1], inner[2], rv[2]}), setOptimizedCallElement(FreshAlloc, elementType)
					case "map", "map_mut":
						if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(rv[2]) {
							return result, td
						}
						innerType := callbackResultType(inner[2], optimizedArgumentType(argumentTypes, 1))
						fusedType := callbackResultType(rv[2], innerType)
						return NewSlice([]Scmer{NewSymbol("map_map"), inner[1], inner[2], rv[2]}), setOptimizedCallElement(setOptimizedCallLength(td, exactOptimizedListArgumentLength(inner[1], optimizedArgumentType(argumentTypes, 1))), fusedType)
					}
				}
			}
		}
	}
	if rv, ok := scmerSlice(result); ok && len(rv) == 3 {
		return result, setOptimizedCallElement(setOptimizedCallLength(td, exactOptimizedListArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1))), elementType)
	}
	return result, td
}

// optimizeFilter fuses a serial map followed by a filter into one physical
// traversal. The Scheme expression remains declarative; the fused operator is
// optimizer-only and does not expose a separate surface-language primitive.
func optimizeFilter(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.applyDefaultOptimization(v, useResult, "filter_mut")
	rv, ok := scmerSlice(result)
	if !ok || len(rv) != 3 || (!scmerIsSymbol(rv[0], "filter") && !scmerIsSymbol(rv[0], "filter_mut")) {
		return result, td
	}
	inner, ok := scmerSlice(rv[1])
	if !ok || len(inner) != 3 {
		return result, td
	}
	switch {
	case scmerIsSymbol(inner[0], "map") || scmerIsSymbol(inner[0], "map_mut"):
		if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(rv[2]) {
			return result, td
		}
		if optimizerNonNilPredicate(rv[2]) {
			return NewSlice([]Scmer{NewSymbol("map_filter_notnull"), inner[1], inner[2]}), descriptorWithLength(FreshAlloc, UnknownLength)
		}
		return NewSlice([]Scmer{NewSymbol("filter_map"), inner[1], inner[2], rv[2]}), descriptorWithLength(FreshAlloc, UnknownLength)
	case scmerIsSymbol(inner[0], "filter") || scmerIsSymbol(inner[0], "filter_mut"):
		if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(rv[2]) {
			return result, td
		}
		return NewSlice([]Scmer{NewSymbol("filter_filter"), inner[1], inner[2], rv[2]}), descriptorWithLength(FreshAlloc, UnknownLength)
	}
	return result, td
}

// optimizeProduceN rewrites (produceN ...) to (produceN_mut ... nil) when the
// result is unused, so runtime can avoid result allocation.
func optimizeProduceN(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	length := UnknownLength
	if len(v) >= 2 {
		if count := int(ToInt(v[1])); count > 0 {
			length = count
		}
	}
	if useResult || !result.IsSlice() {
		return result, descriptorWithLength(td, length)
	}
	rv := result.Slice()
	if len(rv) < 2 {
		return result, descriptorWithLength(td, length)
	}
	if sym, ok := scmerSymbol(rv[0]); !ok || sym != "produceN" {
		return result, descriptorWithLength(td, length)
	}
	out := make([]Scmer, 0, len(rv)+1)
	out = append(out, NewSymbol("produceN_mut"))
	out = append(out, rv[1:]...)
	if len(rv) == 2 {
		out = append(out, NewNil())
	}
	return NewSlice(out), descriptorWithLength(&TypeDescriptor{}, length)
}

// optimizeParallelN rewrites (parallelN ...) to (parallelN_mut ... nil) when
// the result is unused, so runtime can avoid result allocation.
func optimizeParallelN(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	length := UnknownLength
	if len(v) >= 2 {
		if count := int(ToInt(v[1])); count > 0 {
			length = count
		}
	}
	if useResult || !result.IsSlice() {
		return result, descriptorWithLength(td, length)
	}
	rv := result.Slice()
	if len(rv) < 3 {
		return result, descriptorWithLength(td, length)
	}
	if sym, ok := scmerSymbol(rv[0]); !ok || sym != "parallelN" {
		return result, descriptorWithLength(td, length)
	}
	out := make([]Scmer, 0, len(rv)+1)
	out = append(out, NewSymbol("parallelN_mut"))
	out = append(out, rv[1:]...)
	if len(rv) == 3 {
		out = append(out, NewNil())
	}
	return NewSlice(out), descriptorWithLength(&TypeDescriptor{}, length)
}

func asSlice(v Scmer, ctx string) []Scmer {
	// Treat nil as empty list so higher-level code can be concise
	if v.IsNil() {
		return []Scmer{}
	}
	if v.IsSlice() {
		return v.Slice()
	}
	panic(fmt.Sprintf("%s expects a list, got %s", ctx, v.String()))
}

// jitAsSlice keeps the generated Go-call ABI compact. The surrounding builtin
// remains SSA-emitted; only the existing list validation helper is called.
func jitAsSlice(v Scmer) []Scmer {
	return asSlice(v, "jit list operation")
}

func asAssoc(v Scmer, ctx string) ([]Scmer, *FastDict) {
	// Treat nil as empty dictionary (assoc list)
	if v.IsNil() {
		return []Scmer{}, nil
	}
	if v.IsSlice() {
		return v.Slice(), nil
	}
	if v.IsFastDict() {
		return nil, v.FastDict()
	}
	panic(fmt.Sprintf("%s expects a dictionary", ctx))
}

const orderedUniqueHashThreshold = 8

type orderedUniqueBuilder struct {
	linear       []Scmer
	dict         *FastDict
	hashDomain   uint8
	hashDisabled bool
}

func orderedUniqueCanonical(value Scmer) (Scmer, uint8, bool) {
	// FastDict hashes type tags while Equal deliberately coerces across types.
	// Hash only a proven homogeneous domain; add degrades to the exact linear
	// Equal path as soon as a value leaves that domain.
	switch value.GetTag() {
	case tagString, tagSymbol, tagCString, tagBString:
		return NewString(value.String()), 1, true
	case tagInt:
		return value, 2, true
	case tagFloat:
		if value.Float() == 0 {
			return NewFloat(0), 3, true
		}
		return value, 3, true
	case tagBool:
		return value, 4, true
	case tagNil:
		return value, 5, true
	case tagDate:
		return value, 6, true
	default:
		return NewNil(), 0, false
	}
}

func keepOrderedUniqueFirst(oldValue, _ Scmer) Scmer {
	return oldValue
}

func (builder *orderedUniqueBuilder) promote() {
	if builder.hashDisabled || len(builder.linear) < orderedUniqueHashThreshold {
		return
	}
	dict := NewFastDictValue(len(builder.linear))
	domain := uint8(0)
	for _, value := range builder.linear {
		key, valueDomain, ok := orderedUniqueCanonical(value)
		if !ok || (domain != 0 && domain != valueDomain) {
			builder.hashDisabled = true
			return
		}
		domain = valueDomain
		dict.Set(key, value, keepOrderedUniqueFirst)
	}
	builder.linear = nil
	builder.dict = dict
	builder.hashDomain = domain
}

func (builder *orderedUniqueBuilder) degrade() {
	pairs := builder.dict.Pairs
	builder.linear = make([]Scmer, len(pairs)/2)
	for i := range builder.linear {
		builder.linear[i] = pairs[i*2+1]
	}
	builder.dict = nil
	builder.hashDisabled = true
}

func (builder *orderedUniqueBuilder) add(value Scmer) {
	if builder.dict != nil {
		key, domain, ok := orderedUniqueCanonical(value)
		if ok && domain == builder.hashDomain {
			builder.dict.Set(key, value, keepOrderedUniqueFirst)
			return
		}
		builder.degrade()
	}
	for _, existing := range builder.linear {
		if Equal(value, existing) {
			return
		}
	}
	builder.linear = append(builder.linear, value)
	builder.promote()
}

func (builder *orderedUniqueBuilder) result() []Scmer {
	if builder.dict == nil {
		return builder.linear
	}
	pairs := builder.dict.Pairs
	length := len(pairs) / 2
	// The builder owns the transient dictionary. Compact its insertion-ordered
	// values over the disposable keys instead of allocating another result.
	for i := 0; i < length; i++ {
		pairs[i] = pairs[i*2+1]
	}
	clear(pairs[length:])
	return pairs[:length:length]
}

func init_list() {
	// list functions
	DeclareTitle("Lists")

	// list is already in Globalenv.Vars (scm.go init); register it
	// in declarations so serialization can resolve the function pointer.
	Declare(&Globalenv, &Declaration{
		Name:            "list",
		RetainsCallArgs: true,

		Fn: List,
		Type: &TypeDescriptor{Kind: "func", Description: "constructs a list from its arguments",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "items", Description: "items to put into the list", Variadic: true},
			},
			Return:         &TypeDescriptor{Kind: "list", Length: UnknownLength},
			Const:          true,
			Optimize:       optimizeListCall,
			JITVirtualArgs: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				argPinned0 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned0 = append(argPinned0, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned0 = append(argPinned0, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned0 = append(argPinned0, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned0 {
						ctx.UnprotectReg(r)
					}
				}()
				d1 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: append([]JITValueDesc(nil), args...)}
				return jitMaterializeVirtualSlice(ctx, d1, result)
				return result
			},
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "count",

		Fn: func(a ...Scmer) Scmer {
			if a[0].GetTag() == tagSlice {
				return NewInt(int64(len(a[0].Slice())))
			}
			if a[0].GetTag() == tagFastDict {
				fd := a[0].FastDict()
				if fd == nil {
					return NewInt(0)
				}
				return NewInt(int64(len(fd.Pairs)))
			}
			panic("count expects a list")
		},
		Type: &TypeDescriptor{Kind: "func", Description: "counts the number of elements in the list",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "base list", NoEscape: true},
			},
			Return:   &TypeDescriptor{Kind: "int"},
			Const:    true,
			Optimize: optimizeCount,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "nth",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "nth")
			idx := int(a[1].Int())
			if idx < 0 || idx >= len(list) {
				panic("nth index out of range")
			}
			return list[idx]
		},
		Type: &TypeDescriptor{Kind: "func", Description: "get the nth item of a list",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "base list", NoEscape: true},
				{Kind: "number", Label: "index", Description: "index beginning from 0"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [4]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					var d1 JITValueDesc
					if d0.Type == tagSlice {
						d1 = jitKnownSliceHeader(ctx, &d0)
					} else {
						d1 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d0}, 3)
					}
					ctx.BindReg(d1.Reg, &d1)
					ctx.BindReg(d1.Reg2, &d1)
					ctx.BindReg(d1.Reg3, &d1)
					ctx.FreeDesc(&d0)
					d2 = args[1]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int())}
					} else if d2.Type == tagInt && d2.Loc == LocRegPair {
						ctx.FreeReg(d2.Reg)
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg2}
						ctx.BindReg(d2.Reg2, &d3)
						ctx.BindReg(d2.Reg2, &d3)
					} else if d2.Type == tagInt && d2.Loc == LocReg {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
						ctx.BindReg(d2.Reg, &d3)
						ctx.BindReg(d2.Reg, &d3)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d2}, 1)
						d3.Type = tagInt
						ctx.BindReg(d3.Reg, &d3)
					}
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d5 JITValueDesc
					if d3.Loc == LocImm {
						d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < 0)}
					} else {
						r0 := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitCmpRegImm32(d3.Reg, 0)
						ctx.EmitSetcc(r0, CcL)
						d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d5)
					}
					d6 = d5
					ctx.EnsureDesc(&d6)
					if d6.Loc != LocImm && d6.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d6.Loc == LocImm {
						if d6.Imm.Bool() {
							ps7 := PhiState{General: ps.General}
							ps7.OverlayValues = make([]JITValueDesc, 7)
							ps7.OverlayValues[0] = d0
							ps7.OverlayValues[1] = d1
							ps7.OverlayValues[2] = d2
							ps7.OverlayValues[3] = d3
							ps7.OverlayValues[4] = d4
							ps7.OverlayValues[5] = d5
							ps7.OverlayValues[6] = d6
							return bbs[1].RenderPS(ps7)
						}
						ps8 := PhiState{General: ps.General}
						ps8.OverlayValues = make([]JITValueDesc, 7)
						ps8.OverlayValues[0] = d0
						ps8.OverlayValues[1] = d1
						ps8.OverlayValues[2] = d2
						ps8.OverlayValues[3] = d3
						ps8.OverlayValues[4] = d4
						ps8.OverlayValues[5] = d5
						ps8.OverlayValues[6] = d6
						return bbs[3].RenderPS(ps8)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, 0)
					ctx.EmitJcc(CcNE, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 7)
					ps9.OverlayValues[0] = d0
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[2] = d2
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					ps9.OverlayValues[6] = d6
					ps10 := PhiState{General: true}
					ps10.OverlayValues = make([]JITValueDesc, 7)
					ps10.OverlayValues[0] = d0
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					snap11 := d0
					snap12 := d1
					snap13 := d2
					snap14 := d3
					snap15 := d4
					snap16 := d5
					snap17 := d6
					alloc18 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps10)
					}
					ctx.RestoreAllocState(alloc18)
					d0 = snap11
					d1 = snap12
					d2 = snap13
					d3 = snap14
					d4 = snap15
					d5 = snap16
					d6 = snap17
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps9)
					}
					return result
					ctx.FreeDesc(&d5)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					ctx.ReclaimUntrackedRegs()
					d19 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("nth index out of range")}
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d19.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d19)
						} else if d19.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d19)
						} else if d19.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d19)
						} else if d19.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d19.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d19 = tmpPair
					} else if d19.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d19.Type, Reg: ctx.AllocRegExcept(d19.Reg), Reg2: ctx.AllocRegExcept(d19.Reg)}
						switch d19.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d19)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d19)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d19)
						default:
							panic("jit: panic arg scalar type unknown for Scmer pair")
						}
						ctx.FreeDesc(&d19)
						d19 = tmpPair
					}
					if d19.Loc != LocRegPair && d19.Loc != LocStackPair {
						panic("jit: panic arg expects Scmer pair")
					}
					ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{d19})
					ctx.FreeDesc(&d19)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d1)
					if d3.Loc == LocImm {
						ctx.EmitMovRegImm64(r1, uint64(d3.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r1, d3.Reg)
						ctx.EmitShlRegImm8(r1, 4)
					}
					if d1.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(r1, RegR11)
					} else {
						ctx.EmitAddInt64(r1, d1.Reg)
					}
					r2 := ctx.AllocRegExcept(r1)
					r3 := ctx.AllocRegExcept(r1, r2)
					ctx.EmitMovRegMem(r2, r1, 0)
					ctx.EmitMovRegMem(r3, r1, 8)
					ctx.FreeReg(r1)
					d20 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
					ctx.BindReg(r2, &d20)
					ctx.BindReg(r3, &d20)
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d20, &result)
						result.Type = d20.Type
					} else {
						switch d20.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d20)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d20)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d20)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d20, &result)
							result.Type = d20.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					var d21 JITValueDesc
					if d1.SliceSizeKnown {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.KnownSliceLen))}
					} else if d1.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.StackOff))}
					} else {
						ctx.EnsureDesc(&d1)
						if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
							d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
							ctx.BindReg(d1.Reg2, &d21)
							ctx.BindReg(d1.Reg2, &d21)
						} else if d1.Loc == LocReg {
							d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
							ctx.BindReg(d1.Reg, &d21)
							ctx.BindReg(d1.Reg, &d21)
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d21)
					var d22 JITValueDesc
					if d3.Loc == LocImm && d21.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() >= d21.Imm.Int())}
					} else if d21.Loc == LocImm {
						r4 := ctx.AllocReg()
						if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d3.Reg, int32(d21.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
							ctx.EmitCmpInt64(d3.Reg, RegR11)
						}
						ctx.EmitSetcc(r4, CcGE)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d22)
					} else if d3.Loc == LocImm {
						r5 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d21.Reg)
						ctx.EmitSetcc(r5, CcGE)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d22)
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitCmpInt64(d3.Reg, d21.Reg)
						ctx.EmitSetcc(r6, CcGE)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
						ctx.BindReg(r6, &d22)
					}
					ctx.FreeDesc(&d3)
					ctx.FreeDesc(&d21)
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							ps24 := PhiState{General: ps.General}
							ps24.OverlayValues = make([]JITValueDesc, 24)
							ps24.OverlayValues[0] = d0
							ps24.OverlayValues[1] = d1
							ps24.OverlayValues[2] = d2
							ps24.OverlayValues[3] = d3
							ps24.OverlayValues[4] = d4
							ps24.OverlayValues[5] = d5
							ps24.OverlayValues[6] = d6
							ps24.OverlayValues[19] = d19
							ps24.OverlayValues[20] = d20
							ps24.OverlayValues[21] = d21
							ps24.OverlayValues[22] = d22
							ps24.OverlayValues[23] = d23
							return bbs[1].RenderPS(ps24)
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 24)
						ps25.OverlayValues[0] = d0
						ps25.OverlayValues[1] = d1
						ps25.OverlayValues[2] = d2
						ps25.OverlayValues[3] = d3
						ps25.OverlayValues[4] = d4
						ps25.OverlayValues[5] = d5
						ps25.OverlayValues[6] = d6
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[23] = d23
						return bbs[2].RenderPS(ps25)
					}
					if !ps.General {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJcc(CcNE, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl3)
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 24)
					ps26.OverlayValues[0] = d0
					ps26.OverlayValues[1] = d1
					ps26.OverlayValues[2] = d2
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[4] = d4
					ps26.OverlayValues[5] = d5
					ps26.OverlayValues[6] = d6
					ps26.OverlayValues[19] = d19
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[22] = d22
					ps26.OverlayValues[23] = d23
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 24)
					ps27.OverlayValues[0] = d0
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[6] = d6
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					snap28 := d0
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d6
					snap35 := d19
					snap36 := d20
					snap37 := d21
					snap38 := d22
					snap39 := d23
					alloc40 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps27)
					}
					ctx.RestoreAllocState(alloc40)
					d0 = snap28
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d6 = snap34
					d19 = snap35
					d20 = snap36
					d21 = snap37
					d22 = snap38
					d23 = snap39
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps26)
					}
					return result
					ctx.FreeDesc(&d22)
					return result
				}
				argPinned41 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned41 = append(argPinned41, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned41 = append(argPinned41, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned41 = append(argPinned41, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned41 = append(argPinned41, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned41 {
						ctx.UnprotectReg(r)
					}
				}()
				ps42 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps42)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "nth_mut",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "nth_mut")
			idx := int(a[1].Int())
			if idx < 0 || idx >= len(list) {
				panic("nth_mut index out of range")
			}
			list[idx] = a[2]
			return NewSlice(list)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "sets the nth item of an owned list in-place and returns the mutated list",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned base list"},
				{Kind: "number", Label: "index", Description: "index beginning from 0"},
				{Kind: "any", Label: "value", Description: "new value"},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "slice",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "slice")
			start := int(a[1].Int())
			end := int(a[2].Int())
			if start < 0 {
				start = 0
			}
			if end > len(list) {
				end = len(list)
			}
			if start >= end {
				return NewSlice([]Scmer{})
			}
			result := make([]Scmer, end-start)
			copy(result, list[start:end])
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "extract a sublist from start (inclusive) to end (exclusive).\n(slice list start end) returns elements list[start..end).",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "base list", NoEscape: true},
				{Kind: "number", Label: "start", Description: "start index (inclusive)"},
				{Kind: "number", Label: "end", Description: "end index (exclusive)"},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reverse",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "reverse")
			n := len(list)
			result := make([]Scmer, n)
			for i := 0; i < n; i++ {
				result[i] = list[n-1-i]
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a new list with elements in reversed order.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list to reverse", NoEscape: true},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 optimizeFixedLengthInput("reverse_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "append",

		Fn: func(a ...Scmer) Scmer {
			base := append([]Scmer{}, asSlice(a[0], "append")...)
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "appends items to a list and return the extended list.\nThe original list stays unharmed.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "base list"},
				{Kind: "any", Label: "item...", Description: "items to add", Variadic: true},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 optimizeAppend,
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "append_unique",

		Fn: func(a ...Scmer) Scmer {
			list := append([]Scmer{}, asSlice(a[0], "append_unique")...)
			for _, el := range a[1:] {
				for _, el2 := range list {
					if Equal(el, el2) {
						// ignore duplicates
						goto skipItem
					}
				}
				list = append(list, el)
			skipItem:
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "appends items to a list but only if they are new.\nThe original list stays unharmed.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "base list"},
				{Kind: "any", Label: "item...", Description: "items to add", Variadic: true},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 FirstParameterMutable("append_unique_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cons",

		Fn: func(a ...Scmer) Scmer {
			car := a[0]
			if a[1].GetTag() == tagSlice {
				return NewSlice(append([]Scmer{car}, a[1].Slice()...))
			}
			return NewSlice([]Scmer{car, a[1]})
		},
		Type: &TypeDescriptor{Kind: "func", Description: "constructs a list from a head and a tail list",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "car", Description: "new head element"},
				{Kind: "list", Label: "cdr", Description: "tail that is appended after car", NoEscape: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeCons,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "car",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "car")
			if len(list) == 0 {
				panic("car on empty list")
			}
			return list[0]
		},
		Type: &TypeDescriptor{Kind: "func", Description: "extracts the head of a list",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d1 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d0}, 3)
					ctx.BindReg(d1.Reg, &d1)
					ctx.BindReg(d1.Reg2, &d1)
					ctx.BindReg(d1.Reg3, &d1)
					ctx.FreeDesc(&d0)
					var d2 JITValueDesc
					if d1.Loc == LocImm {
						d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.StackOff))}
					} else {
						ctx.EnsureDesc(&d1)
						if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
							d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
							ctx.BindReg(d1.Reg2, &d2)
							ctx.BindReg(d1.Reg2, &d2)
						} else if d1.Loc == LocReg {
							d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
							ctx.BindReg(d1.Reg, &d2)
							ctx.BindReg(d1.Reg, &d2)
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r0, CcE)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.EnsureDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[0] = d0
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						ps6 := PhiState{General: ps.General}
						ps6.OverlayValues = make([]JITValueDesc, 5)
						ps6.OverlayValues[0] = d0
						ps6.OverlayValues[1] = d1
						ps6.OverlayValues[2] = d2
						ps6.OverlayValues[3] = d3
						ps6.OverlayValues[4] = d4
						return bbs[2].RenderPS(ps6)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d4.Reg, 0)
					ctx.EmitJcc(CcNE, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps7 := PhiState{General: true}
					ps7.OverlayValues = make([]JITValueDesc, 5)
					ps7.OverlayValues[0] = d0
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					ps7.OverlayValues[4] = d4
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 5)
					ps8.OverlayValues[0] = d0
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[2] = d2
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					snap9 := d0
					snap10 := d1
					snap11 := d2
					snap12 := d3
					snap13 := d4
					alloc14 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps8)
					}
					ctx.RestoreAllocState(alloc14)
					d0 = snap9
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps7)
					}
					return result
					ctx.FreeDesc(&d3)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					ctx.ReclaimUntrackedRegs()
					d15 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("car on empty list")}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d15.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d15)
						} else if d15.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d15)
						} else if d15.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d15)
						} else if d15.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d15.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d15 = tmpPair
					} else if d15.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d15.Type, Reg: ctx.AllocRegExcept(d15.Reg), Reg2: ctx.AllocRegExcept(d15.Reg)}
						switch d15.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d15)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d15)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d15)
						default:
							panic("jit: panic arg scalar type unknown for Scmer pair")
						}
						ctx.FreeDesc(&d15)
						d15 = tmpPair
					}
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: panic arg expects Scmer pair")
					}
					ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{d15})
					ctx.FreeDesc(&d15)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					ctx.ReclaimUntrackedRegs()
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d1)
					if d16.Loc == LocImm {
						ctx.EmitMovRegImm64(r1, uint64(d16.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r1, d16.Reg)
						ctx.EmitShlRegImm8(r1, 4)
					}
					if d1.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(r1, RegR11)
					} else {
						ctx.EmitAddInt64(r1, d1.Reg)
					}
					r2 := ctx.AllocRegExcept(r1)
					r3 := ctx.AllocRegExcept(r1, r2)
					ctx.EmitMovRegMem(r2, r1, 0)
					ctx.EmitMovRegMem(r3, r1, 8)
					ctx.FreeReg(r1)
					d17 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
					ctx.BindReg(r2, &d17)
					ctx.BindReg(r3, &d17)
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d17, &result)
						result.Type = d17.Type
					} else {
						switch d17.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d17)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d17)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d17)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d17, &result)
							result.Type = d17.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned18 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned18 = append(argPinned18, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned18 = append(argPinned18, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned18 = append(argPinned18, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned18 = append(argPinned18, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned18 {
						ctx.UnprotectReg(r)
					}
				}()
				ps19 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps19)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cdr",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cdr")
			if len(list) == 0 {
				return NewSlice([]Scmer{})
			}
			return NewSlice(list[1:])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "extracts the tail of a list\nThe tail of a list is a list with all items except the head.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list", NoEscape: true},
			},
			// cdr shares the input slice's backing array; its result is borrowed.
			Return:   &TypeDescriptor{Kind: "list"},
			Const:    true,
			Optimize: optimizeCdr,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d1 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d0}, 3)
					ctx.BindReg(d1.Reg, &d1)
					ctx.BindReg(d1.Reg2, &d1)
					ctx.BindReg(d1.Reg3, &d1)
					ctx.FreeDesc(&d0)
					var d2 JITValueDesc
					if d1.Loc == LocImm {
						d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.StackOff))}
					} else {
						ctx.EnsureDesc(&d1)
						if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
							d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
							ctx.BindReg(d1.Reg2, &d2)
							ctx.BindReg(d1.Reg2, &d2)
						} else if d1.Loc == LocReg {
							d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
							ctx.BindReg(d1.Reg, &d2)
							ctx.BindReg(d1.Reg, &d2)
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == 0)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 0)
						ctx.EmitSetcc(r0, CcE)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.EnsureDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[0] = d0
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						ps6 := PhiState{General: ps.General}
						ps6.OverlayValues = make([]JITValueDesc, 5)
						ps6.OverlayValues[0] = d0
						ps6.OverlayValues[1] = d1
						ps6.OverlayValues[2] = d2
						ps6.OverlayValues[3] = d3
						ps6.OverlayValues[4] = d4
						return bbs[2].RenderPS(ps6)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d4.Reg, 0)
					ctx.EmitJcc(CcNE, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps7 := PhiState{General: true}
					ps7.OverlayValues = make([]JITValueDesc, 5)
					ps7.OverlayValues[0] = d0
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					ps7.OverlayValues[4] = d4
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 5)
					ps8.OverlayValues[0] = d0
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[2] = d2
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					snap9 := d0
					snap10 := d1
					snap11 := d2
					snap12 := d3
					snap13 := d4
					alloc14 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps8)
					}
					ctx.RestoreAllocState(alloc14)
					d0 = snap9
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps7)
					}
					return result
					ctx.FreeDesc(&d3)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					r3 := ctx.AllocRegExcept(r1, r2)
					ctx.EmitMovRegImm64(r1, 0)
					ctx.EmitMovRegImm64(r2, 0)
					ctx.EmitMovRegImm64(r3, 0)
					d15 = JITValueDesc{Loc: LocRegTriple, Reg: r1, Reg2: r2, Reg3: r3}
					ctx.BindReg(r1, &d15)
					ctx.BindReg(r2, &d15)
					ctx.BindReg(r3, &d15)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocRegTriple && d15.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (NewSlice arg0)")
					}
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewSlice), []JITValueDesc{d15}, 2)
					ctx.BindReg(d16.Reg, &d16)
					ctx.BindReg(d16.Reg2, &d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d16, &result)
						result.Type = d16.Type
					} else {
						switch d16.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d16)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d16)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d16)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d16, &result)
							result.Type = d16.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					ctx.ReclaimUntrackedRegs()
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					var d18 JITValueDesc
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
						d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
						ctx.BindReg(d1.Reg2, &d18)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d18)
					var d20 JITValueDesc
					if d18.Loc == LocImm && d17.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d18.Imm.Int() - d17.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d18.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d18.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d18.Reg)
						}
						if d17.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
							ctx.EmitSubInt64(r4, RegR11)
						} else {
							ctx.EmitSubInt64(r4, d17.Reg)
						}
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d20)
					}
					var d21 JITValueDesc
					if d1.Loc == LocImm && d17.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + d17.Imm.Int()*16)}
					} else {
						r5 := ctx.AllocReg()
						if d1.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d1.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d1.Reg)
						}
						if d17.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()*16))
							ctx.EmitAddInt64(r5, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r5, d17.Reg)
							ctx.EmitMovRegReg(offsetReg, d17.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r5, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d21)
					}
					var d22 JITValueDesc
					r6 := ctx.AllocReg()
					r7 := ctx.AllocReg()
					if d21.Loc == LocImm {
						ctx.EmitMovRegImm64(r6, uint64(d21.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r6, d21.Reg)
						ctx.FreeReg(d21.Reg)
					}
					if d20.Loc == LocImm {
						ctx.EmitMovRegImm64(r7, uint64(d20.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r7, d20.Reg)
						ctx.FreeReg(d20.Reg)
					}
					r8 := ctx.AllocRegExcept(r6, r7)
					ctx.EmitMovRegReg(r8, d1.Reg3)
					if d17.Loc == LocImm {
						if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(r8, int32(d17.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
							ctx.EmitSubInt64(r8, RegR11)
						}
					} else {
						ctx.EmitSubInt64(r8, d17.Reg)
					}
					d22 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
					ctx.BindReg(r6, &d22)
					ctx.BindReg(r7, &d22)
					ctx.BindReg(r8, &d22)
					ctx.EnsureDesc(&d22)
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocRegTriple && d22.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (NewSlice arg0)")
					}
					d23 = ctx.EmitGoCallScalar(GoFuncAddr(NewSlice), []JITValueDesc{d22}, 2)
					ctx.BindReg(d23.Reg, &d23)
					ctx.BindReg(d23.Reg2, &d23)
					ctx.EnsureDesc(&d23)
					if d23.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d23, &result)
						result.Type = d23.Type
					} else {
						switch d23.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d23)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d23)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d23)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d23, &result)
							result.Type = d23.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned24 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned24 = append(argPinned24, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned24 = append(argPinned24, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned24 = append(argPinned24, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned24 = append(argPinned24, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned24 {
						ctx.UnprotectReg(r)
					}
				}()
				ps25 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps25)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cadr",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cadr")
			if len(list) < 2 {
				panic("cadr on list with fewer than 2 elements")
			}
			return list[1]
		},
		Type: &TypeDescriptor{Kind: "func", Description: "extracts the second element of a list.\nEquivalent to (car (cdr x)).",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d1 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d0}, 3)
					ctx.BindReg(d1.Reg, &d1)
					ctx.BindReg(d1.Reg2, &d1)
					ctx.BindReg(d1.Reg3, &d1)
					ctx.FreeDesc(&d0)
					var d2 JITValueDesc
					if d1.Loc == LocImm {
						d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d1.StackOff))}
					} else {
						ctx.EnsureDesc(&d1)
						if d1.Loc == LocRegPair || d1.Loc == LocRegTriple {
							d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2}
							ctx.BindReg(d1.Reg2, &d2)
							ctx.BindReg(d1.Reg2, &d2)
						} else if d1.Loc == LocReg {
							d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
							ctx.BindReg(d1.Reg, &d2)
							ctx.BindReg(d1.Reg, &d2)
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					var d3 JITValueDesc
					if d2.Loc == LocImm {
						d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d2.Reg, 2)
						ctx.EmitSetcc(r0, CcL)
						d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d3)
					}
					ctx.FreeDesc(&d2)
					d4 = d3
					ctx.EnsureDesc(&d4)
					if d4.Loc != LocImm && d4.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d4.Loc == LocImm {
						if d4.Imm.Bool() {
							ps5 := PhiState{General: ps.General}
							ps5.OverlayValues = make([]JITValueDesc, 5)
							ps5.OverlayValues[0] = d0
							ps5.OverlayValues[1] = d1
							ps5.OverlayValues[2] = d2
							ps5.OverlayValues[3] = d3
							ps5.OverlayValues[4] = d4
							return bbs[1].RenderPS(ps5)
						}
						ps6 := PhiState{General: ps.General}
						ps6.OverlayValues = make([]JITValueDesc, 5)
						ps6.OverlayValues[0] = d0
						ps6.OverlayValues[1] = d1
						ps6.OverlayValues[2] = d2
						ps6.OverlayValues[3] = d3
						ps6.OverlayValues[4] = d4
						return bbs[2].RenderPS(ps6)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d4.Reg, 0)
					ctx.EmitJcc(CcNE, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps7 := PhiState{General: true}
					ps7.OverlayValues = make([]JITValueDesc, 5)
					ps7.OverlayValues[0] = d0
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					ps7.OverlayValues[4] = d4
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 5)
					ps8.OverlayValues[0] = d0
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[2] = d2
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					snap9 := d0
					snap10 := d1
					snap11 := d2
					snap12 := d3
					snap13 := d4
					alloc14 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps8)
					}
					ctx.RestoreAllocState(alloc14)
					d0 = snap9
					d1 = snap10
					d2 = snap11
					d3 = snap12
					d4 = snap13
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps7)
					}
					return result
					ctx.FreeDesc(&d3)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					ctx.ReclaimUntrackedRegs()
					d15 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("cadr on list with fewer than 2 elements")}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d15.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d15)
						} else if d15.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d15)
						} else if d15.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d15)
						} else if d15.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d15.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d15 = tmpPair
					} else if d15.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d15.Type, Reg: ctx.AllocRegExcept(d15.Reg), Reg2: ctx.AllocRegExcept(d15.Reg)}
						switch d15.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d15)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d15)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d15)
						default:
							panic("jit: panic arg scalar type unknown for Scmer pair")
						}
						ctx.FreeDesc(&d15)
						d15 = tmpPair
					}
					if d15.Loc != LocRegPair && d15.Loc != LocStackPair {
						panic("jit: panic arg expects Scmer pair")
					}
					ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{d15})
					ctx.FreeDesc(&d15)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					ctx.ReclaimUntrackedRegs()
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					r1 := ctx.AllocReg()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d1)
					if d16.Loc == LocImm {
						ctx.EmitMovRegImm64(r1, uint64(d16.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r1, d16.Reg)
						ctx.EmitShlRegImm8(r1, 4)
					}
					if d1.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitAddInt64(r1, RegR11)
					} else {
						ctx.EmitAddInt64(r1, d1.Reg)
					}
					r2 := ctx.AllocRegExcept(r1)
					r3 := ctx.AllocRegExcept(r1, r2)
					ctx.EmitMovRegMem(r2, r1, 0)
					ctx.EmitMovRegMem(r3, r1, 8)
					ctx.FreeReg(r1)
					d17 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
					ctx.BindReg(r2, &d17)
					ctx.BindReg(r3, &d17)
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d17, &result)
						result.Type = d17.Type
					} else {
						switch d17.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d17)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d17)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d17)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d17, &result)
							result.Type = d17.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned18 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned18 = append(argPinned18, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned18 = append(argPinned18, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned18 = append(argPinned18, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned18 = append(argPinned18, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned18 {
						ctx.UnprotectReg(r)
					}
				}()
				ps19 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps19)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "zip",

		Fn: func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "zip")
			}
			if len(lists) == 0 {
				return NewSlice([]Scmer{})
			}
			first := asSlice(lists[0], "zip element")
			size := len(first)
			result := make([]Scmer, size)
			for i := 0; i < size; i++ {
				subresult := make([]Scmer, len(lists))
				for j, v := range lists {
					current := asSlice(v, "zip item")
					if i >= len(current) {
						panic("zip expects lists of equal length")
					}
					subresult[j] = current[i]
				}
				result[i] = NewSlice(subresult)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "swaps the dimension of a list of lists. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as the components that will be zipped into the sub list",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "list", Description: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeZip,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge",

		Fn: func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "merge")
			}
			size := 0
			for _, v := range lists {
				size += len(asSlice(v, "merge item"))
			}
			result := make([]Scmer, 0, size)
			for _, v := range lists {
				result = append(result, asSlice(v, "merge item")...)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "flattens a list of lists into a list containing all the subitems. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as lists that will be merged into one",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "list", Description: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeMerge,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge_unique",

		Fn: func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "merge_unique")
			}
			size := 0
			for _, v := range lists {
				size += len(asSlice(v, "merge_unique item"))
			}
			result := make([]Scmer, 0, size)
			for _, v := range lists {
				for _, el := range asSlice(v, "merge_unique item") {
					duplicate := false
					for _, existing := range result {
						if Equal(el, existing) {
							duplicate = true
							break
						}
					}
					if !duplicate {
						result = append(result, el)
					}
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "flattens a list of lists into a list containing all the subitems. Duplicates are filtered out.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 optimizeMergeUnique,
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "has?",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "has?")
			for _, v := range list {
				if Equal(a[1], v) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "checks if a list has a certain item (equal?)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "haystack", Description: "list to search in", NoEscape: true},
				{Kind: "any", Label: "needle", Description: "item to search for"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "filter",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "filter")
			result := make([]Scmer, 0, len(input))
			fn := OptimizeProcToSerialFunction(a[1])
			for _, v := range input {
				if fn(v).Bool() {
					result = append(result, v)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list that only contains elements that pass the filter function",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be filtered", NoEscape: true},
				{Kind: "func", Label: "condition", Description: "returns whether an item should be included", Params: []*TypeDescriptor{{Kind: "any", Label: "item", Description: "current list item"}}, Return: &TypeDescriptor{Kind: "bool", Label: "included", Description: "whether to include the item"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 optimizeFilter,
			OptimizeFirstArgTransfer: true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d50 JITValueDesc
				_ = d50
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d112 JITValueDesc
				_ = d112
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(40))
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				var bbs [5]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[0].VisitCount >= 0 {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
				}
				bbs[0].VisitCount++
				if ps.General {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_0 = bbs[0].Address
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				ctx.ReclaimUntrackedRegs()
				d3 = args[0]
				d3.ID = 0
				var d4 JITValueDesc
				if d3.Type == tagSlice {
					d4 = jitKnownSliceHeader(ctx, &d3)
				} else {
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d3}, 3)
				}
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.BindReg(d4.Reg3, &d4)
				ctx.FreeDesc(&d3)
				var d5 JITValueDesc
				if d4.SliceSizeKnown {
					d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d5 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d5)
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d5)
				d7 = ctx.EmitGoCallScalar(GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d6, d5}, 3)
				ctx.BindReg(d7.Reg, &d7)
				ctx.BindReg(d7.Reg2, &d7)
				ctx.BindReg(d7.Reg3, &d7)
				ctx.FreeDesc(&d5)
				d8 = args[1]
				d8.ID = 0
				var d9 JITValueDesc
				if d8.Loc == LocLambdaTemplate {
					d9 = d8
				} else {
					d9 = ctx.RequestOptimizedCallback(1)
				}
				ctx.FreeDesc(&d8)
				var d10 JITValueDesc
				if d4.SliceSizeKnown {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.EnsureDesc(&d7)
				if d7.Loc == LocReg {
					ctx.ProtectReg(d7.Reg)
				} else if d7.Loc == LocRegPair {
					ctx.ProtectReg(d7.Reg)
					ctx.ProtectReg(d7.Reg2)
				}
				d11 = d7
				if d11.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d11)
				if d11.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d11.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d11.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d11.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(24))
				if d7.Loc == LocReg {
					ctx.UnprotectReg(d7.Reg)
				} else if d7.Loc == LocRegPair {
					ctx.UnprotectReg(d7.Reg)
					ctx.UnprotectReg(d7.Reg2)
				}
				ps12 := PhiState{General: ps.General}
				ps12.OverlayValues = make([]JITValueDesc, 12)
				ps12.OverlayValues[1] = d1
				ps12.OverlayValues[2] = d2
				ps12.OverlayValues[3] = d3
				ps12.OverlayValues[4] = d4
				ps12.OverlayValues[5] = d5
				ps12.OverlayValues[6] = d6
				ps12.OverlayValues[7] = d7
				ps12.OverlayValues[8] = d8
				ps12.OverlayValues[9] = d9
				ps12.OverlayValues[10] = d10
				ps12.OverlayValues[11] = d11
				ps12.PhiValues = make([]JITValueDesc, 2)
				d13 = d7
				ps12.PhiValues[0] = d13
				d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
				ps12.PhiValues[1] = d14
				if ps12.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps12)
				return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d15 := ps.PhiValues[0]
						ctx.EnsureDesc(&d15)
						ctx.EmitStoreRegMem(d15.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d15.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d15.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d16 := ps.PhiValues[1]
						ctx.EnsureDesc(&d16)
						ctx.EmitStoreToStack(d16, int32(bbs[1].PhiBase)+int32(24))
					}
					if bbs[1].VisitCount >= 0 {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
				}
				bbs[1].VisitCount++
				if ps.General {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_1 = bbs[1].Address
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
					d14 = ps.OverlayValues[14]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d1 = ps.PhiValues[0]
				}
				if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d2 = ps.PhiValues[1]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs17 := make([]Reg, 0, 3)
				seenBlockPinnedRegs18 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs18
				for _, r := range []Reg{d7.Reg, d7.Reg2, d7.Reg3} {
					live := d7.Loc == LocRegTriple && (r == d7.Reg || r == d7.Reg2 || r == d7.Reg3)
					if live && !seenBlockPinnedRegs18[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs18[r] = true
						blockPinnedRegs17 = append(blockPinnedRegs17, r)
					}
				}
				unpinBlockRegs19 := func() { for _, r := range blockPinnedRegs17 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs19()
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				var d20 JITValueDesc
				if d2.Loc == LocImm {
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d2.Reg)
					ctx.EmitMovRegReg(scratch, d2.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d20)
				}
				if d20.Loc == LocReg && d2.Loc == LocReg && d20.Reg == d2.Reg {
					ctx.TransferReg(d2.Reg)
					d2.Loc = LocNone
				}
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d20)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d20)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d20)
				ctx.EnsureDesc(&d10)
				var d21 JITValueDesc
				if d20.Loc == LocImm && d10.Loc == LocImm {
					d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.Int() < d10.Imm.Int())}
				} else if d10.Loc == LocImm {
					r0 := ctx.AllocRegExcept(d20.Reg)
					if d10.Imm.Int() >= -2147483648 && d10.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d20.Reg, int32(d10.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
						ctx.EmitCmpInt64(d20.Reg, RegR11)
					}
					ctx.EmitSetcc(r0, CcL)
					d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
					ctx.BindReg(r0, &d21)
				} else if d20.Loc == LocImm {
					r1 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d10.Reg)
					ctx.EmitSetcc(r1, CcL)
					d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
					ctx.BindReg(r1, &d21)
				} else {
					r2 := ctx.AllocRegExcept(d20.Reg)
					ctx.EmitCmpInt64(d20.Reg, d10.Reg)
					ctx.EmitSetcc(r2, CcL)
					d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
					ctx.BindReg(r2, &d21)
				}
				ctx.FreeDesc(&d10)
				d22 = d21
				ctx.EnsureDesc(&d22)
				if d22.Loc != LocImm && d22.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d22.Loc == LocImm {
					if d22.Imm.Bool() {
				ps23 := PhiState{General: ps.General}
				ps23.OverlayValues = make([]JITValueDesc, 23)
				ps23.OverlayValues[1] = d1
				ps23.OverlayValues[2] = d2
				ps23.OverlayValues[3] = d3
				ps23.OverlayValues[4] = d4
				ps23.OverlayValues[5] = d5
				ps23.OverlayValues[6] = d6
				ps23.OverlayValues[7] = d7
				ps23.OverlayValues[8] = d8
				ps23.OverlayValues[9] = d9
				ps23.OverlayValues[10] = d10
				ps23.OverlayValues[11] = d11
				ps23.OverlayValues[13] = d13
				ps23.OverlayValues[14] = d14
				ps23.OverlayValues[15] = d15
				ps23.OverlayValues[16] = d16
				ps23.OverlayValues[20] = d20
				ps23.OverlayValues[21] = d21
				ps23.OverlayValues[22] = d22
						return bbs[2].RenderPS(ps23)
					}
				ps24 := PhiState{General: ps.General}
				ps24.OverlayValues = make([]JITValueDesc, 23)
				ps24.OverlayValues[1] = d1
				ps24.OverlayValues[2] = d2
				ps24.OverlayValues[3] = d3
				ps24.OverlayValues[4] = d4
				ps24.OverlayValues[5] = d5
				ps24.OverlayValues[6] = d6
				ps24.OverlayValues[7] = d7
				ps24.OverlayValues[8] = d8
				ps24.OverlayValues[9] = d9
				ps24.OverlayValues[10] = d10
				ps24.OverlayValues[11] = d11
				ps24.OverlayValues[13] = d13
				ps24.OverlayValues[14] = d14
				ps24.OverlayValues[15] = d15
				ps24.OverlayValues[16] = d16
				ps24.OverlayValues[20] = d20
				ps24.OverlayValues[21] = d21
				ps24.OverlayValues[22] = d22
					return bbs[3].RenderPS(ps24)
				}
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d25 := ps.PhiValues[0]
						ctx.EnsureDesc(&d25)
						ctx.EmitStoreRegMem(d25.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d25.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d25.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d26 := ps.PhiValues[1]
						ctx.EnsureDesc(&d26)
						ctx.EmitStoreToStack(d26, int32(bbs[1].PhiBase)+int32(24))
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
				lbl6 := ctx.ReserveLabel()
				lbl7 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d22.Reg, 0)
				ctx.EmitJcc(CcNE, lbl6)
				ctx.EmitJmp(lbl7)
				ctx.MarkLabel(lbl6)
				ctx.EmitJmp(lbl3)
				ctx.MarkLabel(lbl7)
				ctx.EmitJmp(lbl4)
				ps27 := PhiState{General: true}
				ps27.OverlayValues = make([]JITValueDesc, 27)
				ps27.OverlayValues[1] = d1
				ps27.OverlayValues[2] = d2
				ps27.OverlayValues[3] = d3
				ps27.OverlayValues[4] = d4
				ps27.OverlayValues[5] = d5
				ps27.OverlayValues[6] = d6
				ps27.OverlayValues[7] = d7
				ps27.OverlayValues[8] = d8
				ps27.OverlayValues[9] = d9
				ps27.OverlayValues[10] = d10
				ps27.OverlayValues[11] = d11
				ps27.OverlayValues[13] = d13
				ps27.OverlayValues[14] = d14
				ps27.OverlayValues[15] = d15
				ps27.OverlayValues[16] = d16
				ps27.OverlayValues[20] = d20
				ps27.OverlayValues[21] = d21
				ps27.OverlayValues[22] = d22
				ps27.OverlayValues[25] = d25
				ps27.OverlayValues[26] = d26
				ps28 := PhiState{General: true}
				ps28.OverlayValues = make([]JITValueDesc, 27)
				ps28.OverlayValues[1] = d1
				ps28.OverlayValues[2] = d2
				ps28.OverlayValues[3] = d3
				ps28.OverlayValues[4] = d4
				ps28.OverlayValues[5] = d5
				ps28.OverlayValues[6] = d6
				ps28.OverlayValues[7] = d7
				ps28.OverlayValues[8] = d8
				ps28.OverlayValues[9] = d9
				ps28.OverlayValues[10] = d10
				ps28.OverlayValues[11] = d11
				ps28.OverlayValues[13] = d13
				ps28.OverlayValues[14] = d14
				ps28.OverlayValues[15] = d15
				ps28.OverlayValues[16] = d16
				ps28.OverlayValues[20] = d20
				ps28.OverlayValues[21] = d21
				ps28.OverlayValues[22] = d22
				ps28.OverlayValues[25] = d25
				ps28.OverlayValues[26] = d26
				snap29 := d1
				snap30 := d2
				snap31 := d3
				snap32 := d4
				snap33 := d5
				snap34 := d6
				snap35 := d7
				snap36 := d8
				snap37 := d9
				snap38 := d10
				snap39 := d11
				snap40 := d13
				snap41 := d14
				snap42 := d15
				snap43 := d16
				snap44 := d20
				snap45 := d21
				snap46 := d22
				snap47 := d25
				snap48 := d26
				alloc49 := ctx.SnapshotAllocState()
				if !bbs[3].Rendered {
					bbs[3].RenderPS(ps28)
				}
				ctx.RestoreAllocState(alloc49)
				d1 = snap29
				d2 = snap30
				d3 = snap31
				d4 = snap32
				d5 = snap33
				d6 = snap34
				d7 = snap35
				d8 = snap36
				d9 = snap37
				d10 = snap38
				d11 = snap39
				d13 = snap40
				d14 = snap41
				d15 = snap42
				d16 = snap43
				d20 = snap44
				d21 = snap45
				d22 = snap46
				d25 = snap47
				d26 = snap48
				if !bbs[2].Rendered {
					return bbs[2].RenderPS(ps27)
				}
				return result
				ctx.FreeDesc(&d21)
				return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[2].VisitCount >= 0 {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
				}
				bbs[2].VisitCount++
				if ps.General {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_2 = bbs[2].Address
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
					d14 = ps.OverlayValues[14]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
					d20 = ps.OverlayValues[20]
				}
				if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
					d21 = ps.OverlayValues[21]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
					d25 = ps.OverlayValues[25]
				}
				if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
					d26 = ps.OverlayValues[26]
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d20)
				r3 := ctx.AllocReg()
				ctx.EnsureDesc(&d20)
				ctx.EnsureDesc(&d4)
				if d20.Loc == LocImm {
					ctx.EmitMovRegImm64(r3, uint64(d20.Imm.Int()) * 16)
				} else {
					ctx.EmitMovRegReg(r3, d20.Reg)
					ctx.EmitShlRegImm8(r3, 4)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitAddInt64(r3, RegR11)
				} else {
					ctx.EmitAddInt64(r3, d4.Reg)
				}
				r4 := ctx.AllocRegExcept(r3)
				r5 := ctx.AllocRegExcept(r3, r4)
				ctx.EmitMovRegMem(r4, r3, 0)
				ctx.EmitMovRegMem(r5, r3, 8)
				ctx.FreeReg(r3)
				d50 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d50)
				ctx.BindReg(r5, &d50)
				stackArray51 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d50)
				ctx.EnsureDesc(&d50)
				ctx.EmitStoreScmerToStack(d50, int32(stackArray51)+int32(0))
				r6 := ctx.AllocReg()
				r7 := ctx.AllocRegExcept(r6)
				r8 := ctx.AllocRegExcept(r6, r7)
				ctx.EmitLeaRegMem(r6, RegRSP, int32(stackArray51))
				ctx.EmitMovRegImm64(r7, uint64(1))
				ctx.EmitMovRegImm64(r8, uint64(1))
				d52 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r6, &d52)
				ctx.BindReg(r7, &d52)
				ctx.BindReg(r8, &d52)
				callbackArgs54 := make([]JITValueDesc, 1)
				callbackArgs54[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray51)+0}
				var d53 JITValueDesc
				ctx.FreeDesc(&d52)
				if d9.Loc == LocLambdaTemplate && d9.Lambda != nil {
					callbackResultOff55 := ctx.AllocSpill(16)
					ctx.setStackPointer(jitStackRootFrameBP, callbackResultOff55, true)
					outerRegs56 := ctx.PreserveOuterRegs()
					d53 = JITEmitProcInlineWithOuter(ctx, &d9.Lambda.Proc, d9.Lambda.Outer, callbackArgs54, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreRegMem(d53.Reg, RegRBP, callbackResultOff55)
					ctx.EmitStoreRegMem(d53.Reg2, RegRBP, callbackResultOff55+8)
					ctx.RestoreOuterRegs(outerRegs56)
					d53 = JITValueDesc{Loc: LocStackPair, Type: d53.Type, StackOff: callbackResultOff55, NoHeapPointer: d53.NoHeapPointer}
					liveRegs57 := make([]Reg, 0, 21)
					seenLiveRegs58 := make(map[Reg]bool)
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := (d1.Loc == LocReg && r == d1.Reg) ||
							(d1.Loc == LocRegPair && (r == d1.Reg || r == d1.Reg2)) ||
							(d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3))
						if live && !seenLiveRegs58[r] {
							ctx.ProtectReg(r)
							seenLiveRegs58[r] = true
							liveRegs57 = append(liveRegs57, r)
						}
					}
					for _, r := range []Reg{d20.Reg, d20.Reg2, d20.Reg3} {
						live := (d20.Loc == LocReg && r == d20.Reg) ||
							(d20.Loc == LocRegPair && (r == d20.Reg || r == d20.Reg2)) ||
							(d20.Loc == LocRegTriple && (r == d20.Reg || r == d20.Reg2 || r == d20.Reg3))
						if live && !seenLiveRegs58[r] {
							ctx.ProtectReg(r)
							seenLiveRegs58[r] = true
							liveRegs57 = append(liveRegs57, r)
						}
					}
					for _, r := range []Reg{d4.Reg, d4.Reg2, d4.Reg3} {
						live := (d4.Loc == LocReg && r == d4.Reg) ||
							(d4.Loc == LocRegPair && (r == d4.Reg || r == d4.Reg2)) ||
							(d4.Loc == LocRegTriple && (r == d4.Reg || r == d4.Reg2 || r == d4.Reg3))
						if live && !seenLiveRegs58[r] {
							ctx.ProtectReg(r)
							seenLiveRegs58[r] = true
							liveRegs57 = append(liveRegs57, r)
						}
					}
					for _, r := range []Reg{d50.Reg, d50.Reg2, d50.Reg3} {
						live := (d50.Loc == LocReg && r == d50.Reg) ||
							(d50.Loc == LocRegPair && (r == d50.Reg || r == d50.Reg2)) ||
							(d50.Loc == LocRegTriple && (r == d50.Reg || r == d50.Reg2 || r == d50.Reg3))
						if live && !seenLiveRegs58[r] {
							ctx.ProtectReg(r)
							seenLiveRegs58[r] = true
							liveRegs57 = append(liveRegs57, r)
						}
					}
					for _, r := range []Reg{d52.Reg, d52.Reg2, d52.Reg3} {
						live := (d52.Loc == LocReg && r == d52.Reg) ||
							(d52.Loc == LocRegPair && (r == d52.Reg || r == d52.Reg2)) ||
							(d52.Loc == LocRegTriple && (r == d52.Reg || r == d52.Reg2 || r == d52.Reg3))
						if live && !seenLiveRegs58[r] {
							ctx.ProtectReg(r)
							seenLiveRegs58[r] = true
							liveRegs57 = append(liveRegs57, r)
						}
					}
					for _, r := range []Reg{d7.Reg, d7.Reg2, d7.Reg3} {
						live := (d7.Loc == LocReg && r == d7.Reg) ||
							(d7.Loc == LocRegPair && (r == d7.Reg || r == d7.Reg2)) ||
							(d7.Loc == LocRegTriple && (r == d7.Reg || r == d7.Reg2 || r == d7.Reg3))
						if live && !seenLiveRegs58[r] {
							ctx.ProtectReg(r)
							seenLiveRegs58[r] = true
							liveRegs57 = append(liveRegs57, r)
						}
					}
					for _, r := range []Reg{d9.Reg, d9.Reg2, d9.Reg3} {
						live := (d9.Loc == LocReg && r == d9.Reg) ||
							(d9.Loc == LocRegPair && (r == d9.Reg || r == d9.Reg2)) ||
							(d9.Loc == LocRegTriple && (r == d9.Reg || r == d9.Reg2 || r == d9.Reg3))
						if live && !seenLiveRegs58[r] {
							ctx.ProtectReg(r)
							seenLiveRegs58[r] = true
							liveRegs57 = append(liveRegs57, r)
						}
					}
					ctx.EnsureDesc(&d53)
					for _, r := range liveRegs57 { ctx.UnprotectReg(r) }
				} else {
					callbackCallArgs := make([]JITValueDesc, 0, 2)
					callbackCallArgs = append(callbackCallArgs, d9)
					callbackCallArgs = append(callbackCallArgs, callbackArgs54...)
					d53 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
				}
				d60 = d53
				d60.ID = 0
				d59 = ctx.EmitBoolDesc(&d60, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d53)
				d61 = d59
				ctx.EnsureDesc(&d61)
				if d61.Loc != LocImm && d61.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d61.Loc == LocImm {
					if d61.Imm.Bool() {
				ps62 := PhiState{General: ps.General}
				ps62.OverlayValues = make([]JITValueDesc, 62)
				ps62.OverlayValues[1] = d1
				ps62.OverlayValues[2] = d2
				ps62.OverlayValues[3] = d3
				ps62.OverlayValues[4] = d4
				ps62.OverlayValues[5] = d5
				ps62.OverlayValues[6] = d6
				ps62.OverlayValues[7] = d7
				ps62.OverlayValues[8] = d8
				ps62.OverlayValues[9] = d9
				ps62.OverlayValues[10] = d10
				ps62.OverlayValues[11] = d11
				ps62.OverlayValues[13] = d13
				ps62.OverlayValues[14] = d14
				ps62.OverlayValues[15] = d15
				ps62.OverlayValues[16] = d16
				ps62.OverlayValues[20] = d20
				ps62.OverlayValues[21] = d21
				ps62.OverlayValues[22] = d22
				ps62.OverlayValues[25] = d25
				ps62.OverlayValues[26] = d26
				ps62.OverlayValues[50] = d50
				ps62.OverlayValues[52] = d52
				ps62.OverlayValues[53] = d53
				ps62.OverlayValues[59] = d59
				ps62.OverlayValues[60] = d60
				ps62.OverlayValues[61] = d61
						return bbs[4].RenderPS(ps62)
					}
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d20)
				if d20.Loc == LocReg {
					ctx.ProtectReg(d20.Reg)
				} else if d20.Loc == LocRegPair {
					ctx.ProtectReg(d20.Reg)
					ctx.ProtectReg(d20.Reg2)
				}
				d63 = d1
				if d63.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d63)
				if d63.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d63.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d63.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d63.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d64 = d20
				if d64.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d64)
				ctx.EmitStoreToStack(d64, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d20.Loc == LocReg {
					ctx.UnprotectReg(d20.Reg)
				} else if d20.Loc == LocRegPair {
					ctx.UnprotectReg(d20.Reg)
					ctx.UnprotectReg(d20.Reg2)
				}
				ps65 := PhiState{General: ps.General}
				ps65.OverlayValues = make([]JITValueDesc, 65)
				ps65.OverlayValues[1] = d1
				ps65.OverlayValues[2] = d2
				ps65.OverlayValues[3] = d3
				ps65.OverlayValues[4] = d4
				ps65.OverlayValues[5] = d5
				ps65.OverlayValues[6] = d6
				ps65.OverlayValues[7] = d7
				ps65.OverlayValues[8] = d8
				ps65.OverlayValues[9] = d9
				ps65.OverlayValues[10] = d10
				ps65.OverlayValues[11] = d11
				ps65.OverlayValues[13] = d13
				ps65.OverlayValues[14] = d14
				ps65.OverlayValues[15] = d15
				ps65.OverlayValues[16] = d16
				ps65.OverlayValues[20] = d20
				ps65.OverlayValues[21] = d21
				ps65.OverlayValues[22] = d22
				ps65.OverlayValues[25] = d25
				ps65.OverlayValues[26] = d26
				ps65.OverlayValues[50] = d50
				ps65.OverlayValues[52] = d52
				ps65.OverlayValues[53] = d53
				ps65.OverlayValues[59] = d59
				ps65.OverlayValues[60] = d60
				ps65.OverlayValues[61] = d61
				ps65.OverlayValues[63] = d63
				ps65.OverlayValues[64] = d64
				ps65.PhiValues = make([]JITValueDesc, 2)
				d66 = d1
				ps65.PhiValues[0] = d66
				d67 = d20
				ps65.PhiValues[1] = d67
					return bbs[1].RenderPS(ps65)
				}
				if !ps.General {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
				lbl8 := ctx.ReserveLabel()
				lbl9 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d61.Reg, 0)
				ctx.EmitJcc(CcNE, lbl8)
				ctx.EmitJmp(lbl9)
				ctx.MarkLabel(lbl8)
				ctx.EmitJmp(lbl5)
				ctx.MarkLabel(lbl9)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d20)
				if d20.Loc == LocReg {
					ctx.ProtectReg(d20.Reg)
				} else if d20.Loc == LocRegPair {
					ctx.ProtectReg(d20.Reg)
					ctx.ProtectReg(d20.Reg2)
				}
				d68 = d1
				if d68.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d68)
				if d68.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d68.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d68.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d68.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d69 = d20
				if d69.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d69)
				ctx.EmitStoreToStack(d69, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d20.Loc == LocReg {
					ctx.UnprotectReg(d20.Reg)
				} else if d20.Loc == LocRegPair {
					ctx.UnprotectReg(d20.Reg)
					ctx.UnprotectReg(d20.Reg2)
				}
				ctx.EmitJmp(lbl2)
				ps70 := PhiState{General: true}
				ps70.OverlayValues = make([]JITValueDesc, 70)
				ps70.OverlayValues[1] = d1
				ps70.OverlayValues[2] = d2
				ps70.OverlayValues[3] = d3
				ps70.OverlayValues[4] = d4
				ps70.OverlayValues[5] = d5
				ps70.OverlayValues[6] = d6
				ps70.OverlayValues[7] = d7
				ps70.OverlayValues[8] = d8
				ps70.OverlayValues[9] = d9
				ps70.OverlayValues[10] = d10
				ps70.OverlayValues[11] = d11
				ps70.OverlayValues[13] = d13
				ps70.OverlayValues[14] = d14
				ps70.OverlayValues[15] = d15
				ps70.OverlayValues[16] = d16
				ps70.OverlayValues[20] = d20
				ps70.OverlayValues[21] = d21
				ps70.OverlayValues[22] = d22
				ps70.OverlayValues[25] = d25
				ps70.OverlayValues[26] = d26
				ps70.OverlayValues[50] = d50
				ps70.OverlayValues[52] = d52
				ps70.OverlayValues[53] = d53
				ps70.OverlayValues[59] = d59
				ps70.OverlayValues[60] = d60
				ps70.OverlayValues[61] = d61
				ps70.OverlayValues[63] = d63
				ps70.OverlayValues[64] = d64
				ps70.OverlayValues[66] = d66
				ps70.OverlayValues[67] = d67
				ps70.OverlayValues[68] = d68
				ps70.OverlayValues[69] = d69
				ps71 := PhiState{General: true}
				ps71.OverlayValues = make([]JITValueDesc, 70)
				ps71.OverlayValues[1] = d1
				ps71.OverlayValues[2] = d2
				ps71.OverlayValues[3] = d3
				ps71.OverlayValues[4] = d4
				ps71.OverlayValues[5] = d5
				ps71.OverlayValues[6] = d6
				ps71.OverlayValues[7] = d7
				ps71.OverlayValues[8] = d8
				ps71.OverlayValues[9] = d9
				ps71.OverlayValues[10] = d10
				ps71.OverlayValues[11] = d11
				ps71.OverlayValues[13] = d13
				ps71.OverlayValues[14] = d14
				ps71.OverlayValues[15] = d15
				ps71.OverlayValues[16] = d16
				ps71.OverlayValues[20] = d20
				ps71.OverlayValues[21] = d21
				ps71.OverlayValues[22] = d22
				ps71.OverlayValues[25] = d25
				ps71.OverlayValues[26] = d26
				ps71.OverlayValues[50] = d50
				ps71.OverlayValues[52] = d52
				ps71.OverlayValues[53] = d53
				ps71.OverlayValues[59] = d59
				ps71.OverlayValues[60] = d60
				ps71.OverlayValues[61] = d61
				ps71.OverlayValues[63] = d63
				ps71.OverlayValues[64] = d64
				ps71.OverlayValues[66] = d66
				ps71.OverlayValues[67] = d67
				ps71.OverlayValues[68] = d68
				ps71.OverlayValues[69] = d69
				ps71.PhiValues = make([]JITValueDesc, 2)
				d72 = d1
				ps71.PhiValues[0] = d72
				d73 = d20
				ps71.PhiValues[1] = d73
				snap74 := d1
				snap75 := d2
				snap76 := d3
				snap77 := d4
				snap78 := d5
				snap79 := d6
				snap80 := d7
				snap81 := d8
				snap82 := d9
				snap83 := d10
				snap84 := d11
				snap85 := d13
				snap86 := d14
				snap87 := d15
				snap88 := d16
				snap89 := d20
				snap90 := d21
				snap91 := d22
				snap92 := d25
				snap93 := d26
				snap94 := d50
				snap95 := d52
				snap96 := d53
				snap97 := d59
				snap98 := d60
				snap99 := d61
				snap100 := d63
				snap101 := d64
				snap102 := d66
				snap103 := d67
				snap104 := d68
				snap105 := d69
				snap106 := d72
				snap107 := d73
				alloc108 := ctx.SnapshotAllocState()
				if !bbs[1].Rendered {
					bbs[1].RenderPS(ps71)
				}
				ctx.RestoreAllocState(alloc108)
				d1 = snap74
				d2 = snap75
				d3 = snap76
				d4 = snap77
				d5 = snap78
				d6 = snap79
				d7 = snap80
				d8 = snap81
				d9 = snap82
				d10 = snap83
				d11 = snap84
				d13 = snap85
				d14 = snap86
				d15 = snap87
				d16 = snap88
				d20 = snap89
				d21 = snap90
				d22 = snap91
				d25 = snap92
				d26 = snap93
				d50 = snap94
				d52 = snap95
				d53 = snap96
				d59 = snap97
				d60 = snap98
				d61 = snap99
				d63 = snap100
				d64 = snap101
				d66 = snap102
				d67 = snap103
				d68 = snap104
				d69 = snap105
				d72 = snap106
				d73 = snap107
				if !bbs[4].Rendered {
					return bbs[4].RenderPS(ps70)
				}
				return result
				ctx.FreeDesc(&d59)
				return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[3].VisitCount >= 0 {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
				}
				bbs[3].VisitCount++
				if ps.General {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_3 = bbs[3].Address
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
					d14 = ps.OverlayValues[14]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
					d20 = ps.OverlayValues[20]
				}
				if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
					d21 = ps.OverlayValues[21]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
					d25 = ps.OverlayValues[25]
				}
				if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
					d26 = ps.OverlayValues[26]
				}
				if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
					d50 = ps.OverlayValues[50]
				}
				if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
					d52 = ps.OverlayValues[52]
				}
				if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
					d53 = ps.OverlayValues[53]
				}
				if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
					d59 = ps.OverlayValues[59]
				}
				if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
					d60 = ps.OverlayValues[60]
				}
				if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
					d61 = ps.OverlayValues[61]
				}
				if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
					d63 = ps.OverlayValues[63]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
					d66 = ps.OverlayValues[66]
				}
				if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
					d67 = ps.OverlayValues[67]
				}
				if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
					d68 = ps.OverlayValues[68]
				}
				if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
					d69 = ps.OverlayValues[69]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs109 := make([]Reg, 0, 3)
				seenBlockPinnedRegs110 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs110
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs110[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs110[r] = true
						blockPinnedRegs109 = append(blockPinnedRegs109, r)
					}
				}
				unpinBlockRegs111 := func() { for _, r := range blockPinnedRegs109 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs111()
				d112 = ctx.EmitNewSliceFromGoSlice(&d1)
				ctx.EnsureDesc(&d112)
				if d112.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d112, &result)
					result.Type = d112.Type
				} else {
					switch d112.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d112)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d112)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d112)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						ctx.EmitMovPairToResult(&d112, &result)
						result.Type = d112.Type
					}
				}
				ctx.EmitJmp(lbl0)
				return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[4].VisitCount >= 0 {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
				}
				bbs[4].VisitCount++
				if ps.General {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_4 = bbs[4].Address
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
					d14 = ps.OverlayValues[14]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
					d20 = ps.OverlayValues[20]
				}
				if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
					d21 = ps.OverlayValues[21]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
					d25 = ps.OverlayValues[25]
				}
				if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
					d26 = ps.OverlayValues[26]
				}
				if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
					d50 = ps.OverlayValues[50]
				}
				if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
					d52 = ps.OverlayValues[52]
				}
				if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
					d53 = ps.OverlayValues[53]
				}
				if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
					d59 = ps.OverlayValues[59]
				}
				if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
					d60 = ps.OverlayValues[60]
				}
				if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
					d61 = ps.OverlayValues[61]
				}
				if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
					d63 = ps.OverlayValues[63]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
					d66 = ps.OverlayValues[66]
				}
				if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
					d67 = ps.OverlayValues[67]
				}
				if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
					d68 = ps.OverlayValues[68]
				}
				if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
					d69 = ps.OverlayValues[69]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
					d112 = ps.OverlayValues[112]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs113 := make([]Reg, 0, 3)
				seenBlockPinnedRegs114 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs114
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs114[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs114[r] = true
						blockPinnedRegs113 = append(blockPinnedRegs113, r)
					}
				}
				unpinBlockRegs115 := func() { for _, r := range blockPinnedRegs113 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs115()
				stackArray116 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d50)
				ctx.EnsureDesc(&d50)
				ctx.EmitStoreScmerToStack(d50, int32(stackArray116)+int32(0))
				ctx.FreeDesc(&d50)
				r9 := ctx.AllocReg()
				r10 := ctx.AllocRegExcept(r9)
				r11 := ctx.AllocRegExcept(r9, r10)
				ctx.EmitLeaRegMem(r9, RegRSP, int32(stackArray116))
				ctx.EmitMovRegImm64(r10, uint64(1))
				ctx.EmitMovRegImm64(r11, uint64(1))
				d117 = JITValueDesc{Loc: LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r9, &d117)
				ctx.BindReg(r10, &d117)
				ctx.BindReg(r11, &d117)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple { panic("jit: append requires a Go slice header") }
				lbl10 := ctx.ReserveLabel()
				ctx.EmitCmpInt64(d1.Reg2, d1.Reg3)
				ctx.EmitJcc(CcB, lbl10)
				ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{{Loc: LocImm, Type: tagString, Imm: NewString("jit: generated append exceeded its fixed capacity")}})
				ctx.MarkLabel(lbl10)
				d118 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2, NoHeapPointer: true}
				ctx.BindReg(d1.Reg2, &d118)
				d119 = ctx.EmitSliceElementAddress(&d1, &d118, int32(16))
				d120 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray116)}
				ctx.EmitStoreScmerAt(&d119, &d120)
				ctx.FreeDesc(&d119)
				ctx.EmitAddRegImm32(d1.Reg2, 1)
				d121 = d1
				ctx.BindReg(d121.Reg, &d121)
				ctx.BindReg(d121.Reg2, &d121)
				ctx.BindReg(d121.Reg3, &d121)
				ctx.EnsureDesc(&d20)
				if d20.Loc == LocReg {
					ctx.ProtectReg(d20.Reg)
				} else if d20.Loc == LocRegPair {
					ctx.ProtectReg(d20.Reg)
					ctx.ProtectReg(d20.Reg2)
				}
				ctx.EnsureDesc(&d121)
				if d121.Loc == LocReg {
					ctx.ProtectReg(d121.Reg)
				} else if d121.Loc == LocRegPair {
					ctx.ProtectReg(d121.Reg)
					ctx.ProtectReg(d121.Reg2)
				}
				d122 = d121
				if d122.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d122)
				if d122.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d122.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d122.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d122.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d123 = d20
				if d123.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d123)
				ctx.EmitStoreToStack(d123, int32(bbs[1].PhiBase)+int32(24))
				if d20.Loc == LocReg {
					ctx.UnprotectReg(d20.Reg)
				} else if d20.Loc == LocRegPair {
					ctx.UnprotectReg(d20.Reg)
					ctx.UnprotectReg(d20.Reg2)
				}
				if d121.Loc == LocReg {
					ctx.UnprotectReg(d121.Reg)
				} else if d121.Loc == LocRegPair {
					ctx.UnprotectReg(d121.Reg)
					ctx.UnprotectReg(d121.Reg2)
				}
				ps124 := PhiState{General: ps.General}
				ps124.OverlayValues = make([]JITValueDesc, 124)
				ps124.OverlayValues[1] = d1
				ps124.OverlayValues[2] = d2
				ps124.OverlayValues[3] = d3
				ps124.OverlayValues[4] = d4
				ps124.OverlayValues[5] = d5
				ps124.OverlayValues[6] = d6
				ps124.OverlayValues[7] = d7
				ps124.OverlayValues[8] = d8
				ps124.OverlayValues[9] = d9
				ps124.OverlayValues[10] = d10
				ps124.OverlayValues[11] = d11
				ps124.OverlayValues[13] = d13
				ps124.OverlayValues[14] = d14
				ps124.OverlayValues[15] = d15
				ps124.OverlayValues[16] = d16
				ps124.OverlayValues[20] = d20
				ps124.OverlayValues[21] = d21
				ps124.OverlayValues[22] = d22
				ps124.OverlayValues[25] = d25
				ps124.OverlayValues[26] = d26
				ps124.OverlayValues[50] = d50
				ps124.OverlayValues[52] = d52
				ps124.OverlayValues[53] = d53
				ps124.OverlayValues[59] = d59
				ps124.OverlayValues[60] = d60
				ps124.OverlayValues[61] = d61
				ps124.OverlayValues[63] = d63
				ps124.OverlayValues[64] = d64
				ps124.OverlayValues[66] = d66
				ps124.OverlayValues[67] = d67
				ps124.OverlayValues[68] = d68
				ps124.OverlayValues[69] = d69
				ps124.OverlayValues[72] = d72
				ps124.OverlayValues[73] = d73
				ps124.OverlayValues[112] = d112
				ps124.OverlayValues[117] = d117
				ps124.OverlayValues[118] = d118
				ps124.OverlayValues[119] = d119
				ps124.OverlayValues[120] = d120
				ps124.OverlayValues[121] = d121
				ps124.OverlayValues[122] = d122
				ps124.OverlayValues[123] = d123
				ps124.PhiValues = make([]JITValueDesc, 2)
				d125 = d121
				ps124.PhiValues[0] = d125
				d126 = d20
				ps124.PhiValues[1] = d126
				if ps124.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps124)
				return result
				}
				argPinned127 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned127 = append(argPinned127, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned127 = append(argPinned127, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned127 = append(argPinned127, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned127 = append(argPinned127, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned127 {
						ctx.UnprotectReg(r)
					}
				}()
				ps128 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps128)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(40))
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "find",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "find")
			fn := OptimizeProcToSerialFunction(a[1])
			for _, v := range input {
				if fn(v).Bool() {
					return v
				}
			}
			if len(a) >= 3 {
				return a[2]
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the first list element that passes the condition function, or nil/default if none matches",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list to search", NoEscape: true},
				{Kind: "func", Label: "condition", Description: "predicate applied until the first match", Params: []*TypeDescriptor{{Kind: "any", Label: "item", Description: "current list item"}}, Return: &TypeDescriptor{Kind: "bool", Label: "matches", Description: "whether the item matches"}},
				{Kind: "any", Label: "default", Description: "optional default value if nothing matches", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "map")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(v)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list that contains the results of a map function that is applied to the list",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be mapped", NoEscape: true},
				{Kind: "func", Label: "map", Description: "transforms each item", Params: []*TypeDescriptor{{Kind: "any", Label: "item", Description: "current list item"}}, Return: &TypeDescriptor{Kind: "any", Label: "mapped_item", Description: "transformed item"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 optimizeMap,
			OptimizeFirstArgTransfer: true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d18 JITValueDesc
				_ = d18
				var d37 JITValueDesc
				_ = d37
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.FreeDesc(&d2)
					var d4 JITValueDesc
					if d3.SliceSizeKnown {
						d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d4)
					d5 = ctx.RequestPreallocatedSlice(0)
					d6 = jitKnownSliceHeader(ctx, &d5)
					ctx.FreeDesc(&d4)
					d7 = args[1]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else {
						d8 = ctx.RequestOptimizedCallback(1)
					}
					ctx.FreeDesc(&d7)
					var d9 JITValueDesc
					if d3.SliceSizeKnown {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					ps10 := PhiState{General: ps.General}
					ps10.OverlayValues = make([]JITValueDesc, 10)
					ps10.OverlayValues[1] = d1
					ps10.OverlayValues[2] = d2
					ps10.OverlayValues[3] = d3
					ps10.OverlayValues[4] = d4
					ps10.OverlayValues[5] = d5
					ps10.OverlayValues[6] = d6
					ps10.OverlayValues[7] = d7
					ps10.OverlayValues[8] = d8
					ps10.OverlayValues[9] = d9
					ps10.PhiValues = make([]JITValueDesc, 1)
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps10.PhiValues[0] = d11
					if ps10.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps10)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d12 := ps.PhiValues[0]
							ctx.EnsureDesc(&d12)
							ctx.EmitStoreToStack(d12, int32(bbs[1].PhiBase)+int32(0))
						}
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d13 JITValueDesc
					if d1.Loc == LocImm {
						d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d13)
					}
					if d13.Loc == LocReg && d1.Loc == LocReg && d13.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d9)
					var d14 JITValueDesc
					if d13.Loc == LocImm && d9.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d9.Imm.Int())}
					} else if d9.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d13.Reg)
						if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d9.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d9.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, RegR11)
						}
						ctx.EmitSetcc(r0, CcL)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d14)
					} else if d13.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d9.Reg)
						ctx.EmitSetcc(r1, CcL)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d14)
					} else {
						r2 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpInt64(d13.Reg, d9.Reg)
						ctx.EmitSetcc(r2, CcL)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d14)
					}
					ctx.FreeDesc(&d9)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							ps16 := PhiState{General: ps.General}
							ps16.OverlayValues = make([]JITValueDesc, 16)
							ps16.OverlayValues[1] = d1
							ps16.OverlayValues[2] = d2
							ps16.OverlayValues[3] = d3
							ps16.OverlayValues[4] = d4
							ps16.OverlayValues[5] = d5
							ps16.OverlayValues[6] = d6
							ps16.OverlayValues[7] = d7
							ps16.OverlayValues[8] = d8
							ps16.OverlayValues[9] = d9
							ps16.OverlayValues[11] = d11
							ps16.OverlayValues[12] = d12
							ps16.OverlayValues[13] = d13
							ps16.OverlayValues[14] = d14
							ps16.OverlayValues[15] = d15
							return bbs[2].RenderPS(ps16)
						}
						ps17 := PhiState{General: ps.General}
						ps17.OverlayValues = make([]JITValueDesc, 16)
						ps17.OverlayValues[1] = d1
						ps17.OverlayValues[2] = d2
						ps17.OverlayValues[3] = d3
						ps17.OverlayValues[4] = d4
						ps17.OverlayValues[5] = d5
						ps17.OverlayValues[6] = d6
						ps17.OverlayValues[7] = d7
						ps17.OverlayValues[8] = d8
						ps17.OverlayValues[9] = d9
						ps17.OverlayValues[11] = d11
						ps17.OverlayValues[12] = d12
						ps17.OverlayValues[13] = d13
						ps17.OverlayValues[14] = d14
						ps17.OverlayValues[15] = d15
						return bbs[3].RenderPS(ps17)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d18 := ps.PhiValues[0]
							ctx.EnsureDesc(&d18)
							ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJcc(CcNE, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 19)
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[3] = d3
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[5] = d5
					ps19.OverlayValues[6] = d6
					ps19.OverlayValues[7] = d7
					ps19.OverlayValues[8] = d8
					ps19.OverlayValues[9] = d9
					ps19.OverlayValues[11] = d11
					ps19.OverlayValues[12] = d12
					ps19.OverlayValues[13] = d13
					ps19.OverlayValues[14] = d14
					ps19.OverlayValues[15] = d15
					ps19.OverlayValues[18] = d18
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 19)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[8] = d8
					ps20.OverlayValues[9] = d9
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[18] = d18
					snap21 := d1
					snap22 := d2
					snap23 := d3
					snap24 := d4
					snap25 := d5
					snap26 := d6
					snap27 := d7
					snap28 := d8
					snap29 := d9
					snap30 := d11
					snap31 := d12
					snap32 := d13
					snap33 := d14
					snap34 := d15
					snap35 := d18
					alloc36 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps20)
					}
					ctx.RestoreAllocState(alloc36)
					d1 = snap21
					d2 = snap22
					d3 = snap23
					d4 = snap24
					d5 = snap25
					d6 = snap26
					d7 = snap27
					d8 = snap28
					d9 = snap29
					d11 = snap30
					d12 = snap31
					d13 = snap32
					d14 = snap33
					d15 = snap34
					d18 = snap35
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps19)
					}
					return result
					ctx.FreeDesc(&d14)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					r3 := ctx.AllocReg()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d3)
					if d13.Loc == LocImm {
						ctx.EmitMovRegImm64(r3, uint64(d13.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r3, d13.Reg)
						ctx.EmitShlRegImm8(r3, 4)
					}
					if d3.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
						ctx.EmitAddInt64(r3, RegR11)
					} else {
						ctx.EmitAddInt64(r3, d3.Reg)
					}
					r4 := ctx.AllocRegExcept(r3)
					r5 := ctx.AllocRegExcept(r3, r4)
					ctx.EmitMovRegMem(r4, r3, 0)
					ctx.EmitMovRegMem(r5, r3, 8)
					ctx.FreeReg(r3)
					d37 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
					ctx.BindReg(r4, &d37)
					ctx.BindReg(r5, &d37)
					stackArray38 := ctx.AllocStack(int32(16))
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.EmitStoreScmerToStack(d37, int32(stackArray38)+int32(0))
					ctx.FreeDesc(&d37)
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					r8 := ctx.AllocRegExcept(r6, r7)
					ctx.EmitLeaRegMem(r6, RegRSP, int32(stackArray38))
					ctx.EmitMovRegImm64(r7, uint64(1))
					ctx.EmitMovRegImm64(r8, uint64(1))
					d39 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					ctx.BindReg(r6, &d39)
					ctx.BindReg(r7, &d39)
					ctx.BindReg(r8, &d39)
					callbackArgs41 := make([]JITValueDesc, 1)
					callbackArgs41[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray38) + 0}
					var d40 JITValueDesc
					ctx.FreeDesc(&d39)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						outerRegs42 := ctx.PreserveOuterRegs()
						d40 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, callbackArgs41, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
						ctx.RestoreOuterRegs(outerRegs42)
					} else {
						callbackCallArgs := make([]JITValueDesc, 0, 2)
						callbackCallArgs = append(callbackCallArgs, d8)
						callbackCallArgs = append(callbackCallArgs, callbackArgs41...)
						d40 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d40)
					d43 = ctx.EmitSliceElementAddress(&d6, &d13, int32(16))
					ctx.EmitStoreScmerAt(&d43, &d40)
					ctx.FreeDesc(&d43)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocReg {
						ctx.ProtectReg(d13.Reg)
					} else if d13.Loc == LocRegPair {
						ctx.ProtectReg(d13.Reg)
						ctx.ProtectReg(d13.Reg2)
					}
					d44 = d13
					if d44.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d44)
					ctx.EmitStoreToStack(d44, int32(bbs[1].PhiBase)+int32(0))
					if d13.Loc == LocReg {
						ctx.UnprotectReg(d13.Reg)
					} else if d13.Loc == LocRegPair {
						ctx.UnprotectReg(d13.Reg)
						ctx.UnprotectReg(d13.Reg2)
					}
					ps45 := PhiState{General: ps.General}
					ps45.OverlayValues = make([]JITValueDesc, 45)
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[6] = d6
					ps45.OverlayValues[7] = d7
					ps45.OverlayValues[8] = d8
					ps45.OverlayValues[9] = d9
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[12] = d12
					ps45.OverlayValues[13] = d13
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[15] = d15
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[37] = d37
					ps45.OverlayValues[39] = d39
					ps45.OverlayValues[40] = d40
					ps45.OverlayValues[43] = d43
					ps45.OverlayValues[44] = d44
					ps45.PhiValues = make([]JITValueDesc, 1)
					d46 = d13
					ps45.PhiValues[0] = d46
					if ps45.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps45)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					ctx.ReclaimUntrackedRegs()
					d47 = ctx.EmitNewSliceFromGoSlice(&d6)
					ctx.EnsureDesc(&d47)
					if d47.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d47, &result)
						result.Type = d47.Type
					} else {
						switch d47.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d47)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d47)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d47)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d47, &result)
							result.Type = d47.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned48 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned48 = append(argPinned48, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned48 = append(argPinned48, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned48 = append(argPinned48, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned48 = append(argPinned48, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned48 {
						ctx.UnprotectReg(r)
					}
				}()
				ps49 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps49)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallel_map",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "parallel_map")
			if len(list) <= 1 {
				// fast path: no parallelism needed
				result := make([]Scmer, len(list))
				if len(list) == 1 {
					fn := OptimizeProcToSerialFunction(a[1])
					result[0] = fn(list[0])
				}
				return NewSlice(result)
			}
			results := make([]Scmer, len(list))
			workers := runtime.NumCPU()
			if workers > len(list) {
				workers = len(list)
			}
			jobs := make(chan int, workers)
			var wg sync.WaitGroup
			var firstErr atomic.Value
			wg.Add(workers)
			for w := 0; w < workers; w++ {
				gls.Go(func() {
					defer wg.Done()
					fn := OptimizeProcToSerialFunction(a[1])
					for i := range jobs {
						if firstErr.Load() != nil {
							continue // drain remaining jobs
						}
						func() {
							defer func() {
								if r := recover(); r != nil {
									firstErr.CompareAndSwap(nil, r)
								}
							}()
							results[i] = fn(list[i])
						}()
					}
				})
			}
			for i := range list {
				jobs <- i
			}
			close(jobs)
			wg.Wait()
			if err := firstErr.Load(); err != nil {
				panic(err)
			}
			return NewSlice(results)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "like map, but applies fn to each element in parallel using a worker pool limited to runtime.NumCPU()",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list to map over in parallel", NoEscape: true},
				{Kind: "func", Label: "fn", Description: "function applied to each element", Params: []*TypeDescriptor{{Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:                   FreshAlloc,
			Optimize:                 optimizeFixedLengthInput("parallel_map_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallel_map_mut",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "parallel_map_mut")
			if len(list) <= 1 {
				result := make([]Scmer, len(list))
				if len(list) == 1 {
					fn := OptimizeProcToSerialFunction(a[1])
					result[0] = fn(list[0])
				}
				return NewSlice(result)
			}
			results := make([]Scmer, len(list))
			workers := runtime.NumCPU()
			if workers > len(list) {
				workers = len(list)
			}
			jobs := make(chan int, workers)
			var wg sync.WaitGroup
			var firstErr atomic.Value
			wg.Add(workers)
			for w := 0; w < workers; w++ {
				gls.Go(func() {
					defer wg.Done()
					fn := OptimizeProcToSerialFunction(a[1])
					for i := range jobs {
						if firstErr.Load() != nil {
							continue
						}
						func() {
							defer func() {
								if r := recover(); r != nil {
									firstErr.CompareAndSwap(nil, r)
								}
							}()
							results[i] = fn(list[i])
						}()
					}
				})
			}
			for i := range list {
				jobs <- i
			}
			close(jobs)
			wg.Wait()
			if err := firstErr.Load(); err != nil {
				panic(err)
			}
			return NewSlice(results)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "like parallel_map, but signals the optimizer that fn may have side effects",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list to map over in parallel", NoEscape: true},
				{Kind: "func", Label: "fn", Description: "function with side effects applied to each element", Params: []*TypeDescriptor{{Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mapIndex",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "mapIndex")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list that contains the results of a map function that is applied to the list",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be mapped", NoEscape: true},
				{Kind: "func", Label: "map", Description: "transforms each item with its index", Params: []*TypeDescriptor{{Kind: "int", Label: "index", Description: "zero-based item index"}, {Kind: "any", Label: "item", Description: "current list item"}}, Return: &TypeDescriptor{Kind: "any", Label: "mapped_item", Description: "transformed item"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 optimizeFixedLengthInput("mapIndex_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "reduce")
			fn := OptimizeProcToSerialFunction(a[1])
			result := NewNil()
			i := 0
			if len(a) > 2 {
				result = a[2]
			} else if len(list) > 0 {
				result = list[0]
				i = 1
			}
			for i < len(list) {
				result = fn(result, list[i])
				i++
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list that contains the result of a map function",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be reduced", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Kind: "any", Transfer: true, Label: "acc", Description: "current accumulator"}, {Kind: "any", Label: "item", Description: "current list item"}}, Label: "reduce", Description: "combines the accumulator with each list item", Return: &TypeDescriptor{Kind: "any", Label: "acc", Description: "next accumulator"}},
				{Kind: "any", Label: "neutral", Description: "(optional) initial value of the accumulator, defaults to nil", Optional: true},
			},
			Return:   &TypeDescriptor{Kind: "any"},
			Const:    true,
			Optimize: optimizeReduce,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d35 JITValueDesc
				_ = d35
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				var bbs [7]BBDescriptor
				bbs[6].PhiBase = int32(phiBase0) + int32(0)
				bbs[6].PhiCount = uint16(2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					d3 = args[0]
					d3.ID = 0
					var d4 JITValueDesc
					if d3.Type == tagSlice {
						d4 = jitKnownSliceHeader(ctx, &d3)
					} else {
						d4 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d3}, 3)
					}
					ctx.BindReg(d4.Reg, &d4)
					ctx.BindReg(d4.Reg2, &d4)
					ctx.BindReg(d4.Reg3, &d4)
					ctx.FreeDesc(&d3)
					d5 = args[1]
					d5.ID = 0
					var d6 JITValueDesc
					if d5.Loc == LocLambdaTemplate {
						d6 = d5
					} else {
						d6 = ctx.RequestOptimizedCallback(1)
					}
					ctx.FreeDesc(&d5)
					d7 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d8)
					var d9 JITValueDesc
					if d8.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d8.Reg, 2)
						ctx.EmitSetcc(r0, CcG)
						d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d9)
					}
					ctx.FreeDesc(&d8)
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							ps11 := PhiState{General: ps.General}
							ps11.OverlayValues = make([]JITValueDesc, 11)
							ps11.OverlayValues[1] = d1
							ps11.OverlayValues[2] = d2
							ps11.OverlayValues[3] = d3
							ps11.OverlayValues[4] = d4
							ps11.OverlayValues[5] = d5
							ps11.OverlayValues[6] = d6
							ps11.OverlayValues[7] = d7
							ps11.OverlayValues[8] = d8
							ps11.OverlayValues[9] = d9
							ps11.OverlayValues[10] = d10
							return bbs[1].RenderPS(ps11)
						}
						ps12 := PhiState{General: ps.General}
						ps12.OverlayValues = make([]JITValueDesc, 11)
						ps12.OverlayValues[1] = d1
						ps12.OverlayValues[2] = d2
						ps12.OverlayValues[3] = d3
						ps12.OverlayValues[4] = d4
						ps12.OverlayValues[5] = d5
						ps12.OverlayValues[6] = d6
						ps12.OverlayValues[7] = d7
						ps12.OverlayValues[8] = d8
						ps12.OverlayValues[9] = d9
						ps12.OverlayValues[10] = d10
						return bbs[2].RenderPS(ps12)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d10.Reg, 0)
					ctx.EmitJcc(CcNE, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl3)
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 11)
					ps13.OverlayValues[1] = d1
					ps13.OverlayValues[2] = d2
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[4] = d4
					ps13.OverlayValues[5] = d5
					ps13.OverlayValues[6] = d6
					ps13.OverlayValues[7] = d7
					ps13.OverlayValues[8] = d8
					ps13.OverlayValues[9] = d9
					ps13.OverlayValues[10] = d10
					ps14 := PhiState{General: true}
					ps14.OverlayValues = make([]JITValueDesc, 11)
					ps14.OverlayValues[1] = d1
					ps14.OverlayValues[2] = d2
					ps14.OverlayValues[3] = d3
					ps14.OverlayValues[4] = d4
					ps14.OverlayValues[5] = d5
					ps14.OverlayValues[6] = d6
					ps14.OverlayValues[7] = d7
					ps14.OverlayValues[8] = d8
					ps14.OverlayValues[9] = d9
					ps14.OverlayValues[10] = d10
					snap15 := d1
					snap16 := d2
					snap17 := d3
					snap18 := d4
					snap19 := d5
					snap20 := d6
					snap21 := d7
					snap22 := d8
					snap23 := d9
					snap24 := d10
					alloc25 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps14)
					}
					ctx.RestoreAllocState(alloc25)
					d1 = snap15
					d2 = snap16
					d3 = snap17
					d4 = snap18
					d5 = snap19
					d6 = snap20
					d7 = snap21
					d8 = snap22
					d9 = snap23
					d10 = snap24
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps13)
					}
					return result
					ctx.FreeDesc(&d9)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					ctx.ReclaimUntrackedRegs()
					d26 = args[2]
					d26.ID = 0
					ctx.EnsureDesc(&d26)
					if d26.Loc == LocReg {
						ctx.ProtectReg(d26.Reg)
					} else if d26.Loc == LocRegPair {
						ctx.ProtectReg(d26.Reg)
						ctx.ProtectReg(d26.Reg2)
					}
					d27 = d26
					if d27.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d27)
					if d27.Loc == LocRegPair || d27.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d27, int32(bbs[6].PhiBase)+int32(0))
					} else {
						ctx.EmitStoreToStack(d27, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[6].PhiBase)+int32(0))+8)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(16))
					if d26.Loc == LocReg {
						ctx.UnprotectReg(d26.Reg)
					} else if d26.Loc == LocRegPair {
						ctx.UnprotectReg(d26.Reg)
						ctx.UnprotectReg(d26.Reg2)
					}
					ps28 := PhiState{General: ps.General}
					ps28.OverlayValues = make([]JITValueDesc, 28)
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[6] = d6
					ps28.OverlayValues[7] = d7
					ps28.OverlayValues[8] = d8
					ps28.OverlayValues[9] = d9
					ps28.OverlayValues[10] = d10
					ps28.OverlayValues[26] = d26
					ps28.OverlayValues[27] = d27
					ps28.PhiValues = make([]JITValueDesc, 2)
					d29 = d26
					ps28.PhiValues[0] = d29
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps28.PhiValues[1] = d30
					if ps28.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps28)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					ctx.ReclaimUntrackedRegs()
					var d31 JITValueDesc
					if d4.SliceSizeKnown {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
					} else if d4.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
					} else {
						ctx.EnsureDesc(&d4)
						if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
						} else if d4.Loc == LocReg {
							d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d31)
					var d32 JITValueDesc
					if d31.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d31.Imm.Int() > 0)}
					} else {
						r1 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d31.Reg, 0)
						ctx.EmitSetcc(r1, CcG)
						d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d32)
					}
					ctx.FreeDesc(&d31)
					d33 = d32
					ctx.EnsureDesc(&d33)
					if d33.Loc != LocImm && d33.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d33.Loc == LocImm {
						if d33.Imm.Bool() {
							ps34 := PhiState{General: ps.General}
							ps34.OverlayValues = make([]JITValueDesc, 34)
							ps34.OverlayValues[1] = d1
							ps34.OverlayValues[2] = d2
							ps34.OverlayValues[3] = d3
							ps34.OverlayValues[4] = d4
							ps34.OverlayValues[5] = d5
							ps34.OverlayValues[6] = d6
							ps34.OverlayValues[7] = d7
							ps34.OverlayValues[8] = d8
							ps34.OverlayValues[9] = d9
							ps34.OverlayValues[10] = d10
							ps34.OverlayValues[26] = d26
							ps34.OverlayValues[27] = d27
							ps34.OverlayValues[29] = d29
							ps34.OverlayValues[30] = d30
							ps34.OverlayValues[31] = d31
							ps34.OverlayValues[32] = d32
							ps34.OverlayValues[33] = d33
							return bbs[3].RenderPS(ps34)
						}
						ctx.EnsureDesc(&d7)
						if d7.Loc == LocReg {
							ctx.ProtectReg(d7.Reg)
						} else if d7.Loc == LocRegPair {
							ctx.ProtectReg(d7.Reg)
							ctx.ProtectReg(d7.Reg2)
						}
						d35 = d7
						if d35.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d35)
						if d35.Loc == LocRegPair || d35.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d35, int32(bbs[6].PhiBase)+int32(0))
						} else {
							ctx.EmitStoreToStack(d35, int32(bbs[6].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[6].PhiBase)+int32(0))+8)
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(16))
						if d7.Loc == LocReg {
							ctx.UnprotectReg(d7.Reg)
						} else if d7.Loc == LocRegPair {
							ctx.UnprotectReg(d7.Reg)
							ctx.UnprotectReg(d7.Reg2)
						}
						ps36 := PhiState{General: ps.General}
						ps36.OverlayValues = make([]JITValueDesc, 36)
						ps36.OverlayValues[1] = d1
						ps36.OverlayValues[2] = d2
						ps36.OverlayValues[3] = d3
						ps36.OverlayValues[4] = d4
						ps36.OverlayValues[5] = d5
						ps36.OverlayValues[6] = d6
						ps36.OverlayValues[7] = d7
						ps36.OverlayValues[8] = d8
						ps36.OverlayValues[9] = d9
						ps36.OverlayValues[10] = d10
						ps36.OverlayValues[26] = d26
						ps36.OverlayValues[27] = d27
						ps36.OverlayValues[29] = d29
						ps36.OverlayValues[30] = d30
						ps36.OverlayValues[31] = d31
						ps36.OverlayValues[32] = d32
						ps36.OverlayValues[33] = d33
						ps36.OverlayValues[35] = d35
						ps36.PhiValues = make([]JITValueDesc, 2)
						d37 = d7
						ps36.PhiValues[0] = d37
						d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
						ps36.PhiValues[1] = d38
						return bbs[6].RenderPS(ps36)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d33.Reg, 0)
					ctx.EmitJcc(CcNE, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl11)
					ctx.EnsureDesc(&d7)
					if d7.Loc == LocReg {
						ctx.ProtectReg(d7.Reg)
					} else if d7.Loc == LocRegPair {
						ctx.ProtectReg(d7.Reg)
						ctx.ProtectReg(d7.Reg2)
					}
					d39 = d7
					if d39.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d39)
					if d39.Loc == LocRegPair || d39.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d39, int32(bbs[6].PhiBase)+int32(0))
					} else {
						ctx.EmitStoreToStack(d39, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[6].PhiBase)+int32(0))+8)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(16))
					if d7.Loc == LocReg {
						ctx.UnprotectReg(d7.Reg)
					} else if d7.Loc == LocRegPair {
						ctx.UnprotectReg(d7.Reg)
						ctx.UnprotectReg(d7.Reg2)
					}
					ctx.EmitJmp(lbl7)
					ps40 := PhiState{General: true}
					ps40.OverlayValues = make([]JITValueDesc, 40)
					ps40.OverlayValues[1] = d1
					ps40.OverlayValues[2] = d2
					ps40.OverlayValues[3] = d3
					ps40.OverlayValues[4] = d4
					ps40.OverlayValues[5] = d5
					ps40.OverlayValues[6] = d6
					ps40.OverlayValues[7] = d7
					ps40.OverlayValues[8] = d8
					ps40.OverlayValues[9] = d9
					ps40.OverlayValues[10] = d10
					ps40.OverlayValues[26] = d26
					ps40.OverlayValues[27] = d27
					ps40.OverlayValues[29] = d29
					ps40.OverlayValues[30] = d30
					ps40.OverlayValues[31] = d31
					ps40.OverlayValues[32] = d32
					ps40.OverlayValues[33] = d33
					ps40.OverlayValues[35] = d35
					ps40.OverlayValues[37] = d37
					ps40.OverlayValues[38] = d38
					ps40.OverlayValues[39] = d39
					ps41 := PhiState{General: true}
					ps41.OverlayValues = make([]JITValueDesc, 40)
					ps41.OverlayValues[1] = d1
					ps41.OverlayValues[2] = d2
					ps41.OverlayValues[3] = d3
					ps41.OverlayValues[4] = d4
					ps41.OverlayValues[5] = d5
					ps41.OverlayValues[6] = d6
					ps41.OverlayValues[7] = d7
					ps41.OverlayValues[8] = d8
					ps41.OverlayValues[9] = d9
					ps41.OverlayValues[10] = d10
					ps41.OverlayValues[26] = d26
					ps41.OverlayValues[27] = d27
					ps41.OverlayValues[29] = d29
					ps41.OverlayValues[30] = d30
					ps41.OverlayValues[31] = d31
					ps41.OverlayValues[32] = d32
					ps41.OverlayValues[33] = d33
					ps41.OverlayValues[35] = d35
					ps41.OverlayValues[37] = d37
					ps41.OverlayValues[38] = d38
					ps41.OverlayValues[39] = d39
					ps41.PhiValues = make([]JITValueDesc, 2)
					d42 = d7
					ps41.PhiValues[0] = d42
					d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps41.PhiValues[1] = d43
					snap44 := d1
					snap45 := d2
					snap46 := d3
					snap47 := d4
					snap48 := d5
					snap49 := d6
					snap50 := d7
					snap51 := d8
					snap52 := d9
					snap53 := d10
					snap54 := d26
					snap55 := d27
					snap56 := d29
					snap57 := d30
					snap58 := d31
					snap59 := d32
					snap60 := d33
					snap61 := d35
					snap62 := d37
					snap63 := d38
					snap64 := d39
					snap65 := d42
					snap66 := d43
					alloc67 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps41)
					}
					ctx.RestoreAllocState(alloc67)
					d1 = snap44
					d2 = snap45
					d3 = snap46
					d4 = snap47
					d5 = snap48
					d6 = snap49
					d7 = snap50
					d8 = snap51
					d9 = snap52
					d10 = snap53
					d26 = snap54
					d27 = snap55
					d29 = snap56
					d30 = snap57
					d31 = snap58
					d32 = snap59
					d33 = snap60
					d35 = snap61
					d37 = snap62
					d38 = snap63
					d39 = snap64
					d42 = snap65
					d43 = snap66
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps40)
					}
					return result
					ctx.FreeDesc(&d32)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					ctx.ReclaimUntrackedRegs()
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					r2 := ctx.AllocReg()
					ctx.EnsureDesc(&d68)
					ctx.EnsureDesc(&d4)
					if d68.Loc == LocImm {
						ctx.EmitMovRegImm64(r2, uint64(d68.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r2, d68.Reg)
						ctx.EmitShlRegImm8(r2, 4)
					}
					if d4.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
						ctx.EmitAddInt64(r2, RegR11)
					} else {
						ctx.EmitAddInt64(r2, d4.Reg)
					}
					r3 := ctx.AllocRegExcept(r2)
					r4 := ctx.AllocRegExcept(r2, r3)
					ctx.EmitMovRegMem(r3, r2, 0)
					ctx.EmitMovRegMem(r4, r2, 8)
					ctx.FreeReg(r2)
					d69 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r3, Reg2: r4}
					ctx.BindReg(r3, &d69)
					ctx.BindReg(r4, &d69)
					ctx.EnsureDesc(&d69)
					if d69.Loc == LocReg {
						ctx.ProtectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.ProtectReg(d69.Reg)
						ctx.ProtectReg(d69.Reg2)
					}
					d70 = d69
					if d70.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d70)
					if d70.Loc == LocRegPair || d70.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d70, int32(bbs[6].PhiBase)+int32(0))
					} else {
						ctx.EmitStoreToStack(d70, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[6].PhiBase)+int32(0))+8)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(bbs[6].PhiBase)+int32(16))
					if d69.Loc == LocReg {
						ctx.UnprotectReg(d69.Reg)
					} else if d69.Loc == LocRegPair {
						ctx.UnprotectReg(d69.Reg)
						ctx.UnprotectReg(d69.Reg2)
					}
					ps71 := PhiState{General: ps.General}
					ps71.OverlayValues = make([]JITValueDesc, 71)
					ps71.OverlayValues[1] = d1
					ps71.OverlayValues[2] = d2
					ps71.OverlayValues[3] = d3
					ps71.OverlayValues[4] = d4
					ps71.OverlayValues[5] = d5
					ps71.OverlayValues[6] = d6
					ps71.OverlayValues[7] = d7
					ps71.OverlayValues[8] = d8
					ps71.OverlayValues[9] = d9
					ps71.OverlayValues[10] = d10
					ps71.OverlayValues[26] = d26
					ps71.OverlayValues[27] = d27
					ps71.OverlayValues[29] = d29
					ps71.OverlayValues[30] = d30
					ps71.OverlayValues[31] = d31
					ps71.OverlayValues[32] = d32
					ps71.OverlayValues[33] = d33
					ps71.OverlayValues[35] = d35
					ps71.OverlayValues[37] = d37
					ps71.OverlayValues[38] = d38
					ps71.OverlayValues[39] = d39
					ps71.OverlayValues[42] = d42
					ps71.OverlayValues[43] = d43
					ps71.OverlayValues[68] = d68
					ps71.OverlayValues[69] = d69
					ps71.OverlayValues[70] = d70
					ps71.PhiValues = make([]JITValueDesc, 2)
					d72 = d69
					ps71.PhiValues[0] = d72
					d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ps71.PhiValues[1] = d73
					if ps71.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps71)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
							ps.General = true
							return bbs[4].RenderPS(ps)
						}
					}
					bbs[4].VisitCount++
					if ps.General {
						if bbs[4].Rendered {
							ctx.EmitJmp(lbl5)
							return result
						}
						bbs[4].Rendered = true
						bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_4 = bbs[4].Address
						ctx.MarkLabel(lbl5)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					r5 := ctx.AllocReg()
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d4)
					if d2.Loc == LocImm {
						ctx.EmitMovRegImm64(r5, uint64(d2.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r5, d2.Reg)
						ctx.EmitShlRegImm8(r5, 4)
					}
					if d4.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
						ctx.EmitAddInt64(r5, RegR11)
					} else {
						ctx.EmitAddInt64(r5, d4.Reg)
					}
					r6 := ctx.AllocRegExcept(r5)
					r7 := ctx.AllocRegExcept(r5, r6)
					ctx.EmitMovRegMem(r6, r5, 0)
					ctx.EmitMovRegMem(r7, r5, 8)
					ctx.FreeReg(r5)
					d74 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d74)
					ctx.BindReg(r7, &d74)
					stackArray75 := ctx.AllocStack(int32(32))
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EmitStoreScmerToStack(d1, int32(stackArray75)+int32(0))
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					ctx.EmitStoreScmerToStack(d74, int32(stackArray75)+int32(16))
					ctx.FreeDesc(&d74)
					r8 := ctx.AllocReg()
					r9 := ctx.AllocRegExcept(r8)
					r10 := ctx.AllocRegExcept(r8, r9)
					ctx.EmitLeaRegMem(r8, RegRSP, int32(stackArray75))
					ctx.EmitMovRegImm64(r9, uint64(2))
					ctx.EmitMovRegImm64(r10, uint64(2))
					d76 = JITValueDesc{Loc: LocRegTriple, Reg: r8, Reg2: r9, Reg3: r10, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					ctx.BindReg(r8, &d76)
					ctx.BindReg(r9, &d76)
					ctx.BindReg(r10, &d76)
					callbackArgs78 := make([]JITValueDesc, 2)
					callbackArgs78[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray75) + 0}
					callbackArgs78[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray75) + 16}
					var d77 JITValueDesc
					ctx.FreeDesc(&d76)
					if d6.Loc == LocLambdaTemplate && d6.Lambda != nil {
						outerRegs79 := ctx.PreserveOuterRegs()
						d77 = JITEmitProcInlineWithOuter(ctx, &d6.Lambda.Proc, d6.Lambda.Outer, callbackArgs78, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
						ctx.RestoreOuterRegs(outerRegs79)
					} else {
						callbackCallArgs := make([]JITValueDesc, 0, 3)
						callbackCallArgs = append(callbackCallArgs, d6)
						callbackCallArgs = append(callbackCallArgs, callbackArgs78...)
						d77 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d80 JITValueDesc
					if d2.Loc == LocImm {
						d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d80)
					}
					if d80.Loc == LocReg && d2.Loc == LocReg && d80.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d77)
					if d77.Loc == LocReg {
						ctx.ProtectReg(d77.Reg)
					} else if d77.Loc == LocRegPair {
						ctx.ProtectReg(d77.Reg)
						ctx.ProtectReg(d77.Reg2)
					}
					ctx.EnsureDesc(&d80)
					if d80.Loc == LocReg {
						ctx.ProtectReg(d80.Reg)
					} else if d80.Loc == LocRegPair {
						ctx.ProtectReg(d80.Reg)
						ctx.ProtectReg(d80.Reg2)
					}
					d81 = d77
					if d81.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d81)
					if d81.Loc == LocRegPair || d81.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d81, int32(bbs[6].PhiBase)+int32(0))
					} else {
						ctx.EmitStoreToStack(d81, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[6].PhiBase)+int32(0))+8)
					}
					d82 = d80
					if d82.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d82)
					ctx.EmitStoreToStack(d82, int32(bbs[6].PhiBase)+int32(16))
					if d77.Loc == LocReg {
						ctx.UnprotectReg(d77.Reg)
					} else if d77.Loc == LocRegPair {
						ctx.UnprotectReg(d77.Reg)
						ctx.UnprotectReg(d77.Reg2)
					}
					if d80.Loc == LocReg {
						ctx.UnprotectReg(d80.Reg)
					} else if d80.Loc == LocRegPair {
						ctx.UnprotectReg(d80.Reg)
						ctx.UnprotectReg(d80.Reg2)
					}
					ps83 := PhiState{General: ps.General}
					ps83.OverlayValues = make([]JITValueDesc, 83)
					ps83.OverlayValues[1] = d1
					ps83.OverlayValues[2] = d2
					ps83.OverlayValues[3] = d3
					ps83.OverlayValues[4] = d4
					ps83.OverlayValues[5] = d5
					ps83.OverlayValues[6] = d6
					ps83.OverlayValues[7] = d7
					ps83.OverlayValues[8] = d8
					ps83.OverlayValues[9] = d9
					ps83.OverlayValues[10] = d10
					ps83.OverlayValues[26] = d26
					ps83.OverlayValues[27] = d27
					ps83.OverlayValues[29] = d29
					ps83.OverlayValues[30] = d30
					ps83.OverlayValues[31] = d31
					ps83.OverlayValues[32] = d32
					ps83.OverlayValues[33] = d33
					ps83.OverlayValues[35] = d35
					ps83.OverlayValues[37] = d37
					ps83.OverlayValues[38] = d38
					ps83.OverlayValues[39] = d39
					ps83.OverlayValues[42] = d42
					ps83.OverlayValues[43] = d43
					ps83.OverlayValues[68] = d68
					ps83.OverlayValues[69] = d69
					ps83.OverlayValues[70] = d70
					ps83.OverlayValues[72] = d72
					ps83.OverlayValues[73] = d73
					ps83.OverlayValues[74] = d74
					ps83.OverlayValues[76] = d76
					ps83.OverlayValues[77] = d77
					ps83.OverlayValues[80] = d80
					ps83.OverlayValues[81] = d81
					ps83.OverlayValues[82] = d82
					ps83.PhiValues = make([]JITValueDesc, 2)
					d84 = d77
					ps83.PhiValues[0] = d84
					d85 = d80
					ps83.PhiValues[1] = d85
					if ps83.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps83)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
							ps.General = true
							return bbs[5].RenderPS(ps)
						}
					}
					bbs[5].VisitCount++
					if ps.General {
						if bbs[5].Rendered {
							ctx.EmitJmp(lbl6)
							return result
						}
						bbs[5].Rendered = true
						bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_5 = bbs[5].Address
						ctx.MarkLabel(lbl6)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d1, &result)
						result.Type = d1.Type
					} else {
						switch d1.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d1)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d1)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d1)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d1, &result)
							result.Type = d1.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d86 := ps.PhiValues[0]
							ctx.EnsureDesc(&d86)
							ctx.EmitStoreScmerToStack(d86, int32(bbs[6].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d87 := ps.PhiValues[1]
							ctx.EnsureDesc(&d87)
							ctx.EmitStoreToStack(d87, int32(bbs[6].PhiBase)+int32(16))
						}
						if bbs[6].VisitCount >= 0 {
							ps.General = true
							return bbs[6].RenderPS(ps)
						}
					}
					bbs[6].VisitCount++
					if ps.General {
						if bbs[6].Rendered {
							ctx.EmitJmp(lbl7)
							return result
						}
						bbs[6].Rendered = true
						bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_6 = bbs[6].Address
						ctx.MarkLabel(lbl7)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d2 = ps.PhiValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					var d88 JITValueDesc
					if d4.SliceSizeKnown {
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
					} else if d4.Loc == LocImm {
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
					} else {
						ctx.EnsureDesc(&d4)
						if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
							d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
						} else if d4.Loc == LocReg {
							d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d88)
					var d89 JITValueDesc
					if d2.Loc == LocImm && d88.Loc == LocImm {
						d89 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d88.Imm.Int())}
					} else if d88.Loc == LocImm {
						r11 := ctx.AllocReg()
						if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d88.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d88.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, RegR11)
						}
						ctx.EmitSetcc(r11, CcL)
						d89 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
						ctx.BindReg(r11, &d89)
					} else if d2.Loc == LocImm {
						r12 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d88.Reg)
						ctx.EmitSetcc(r12, CcL)
						d89 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
						ctx.BindReg(r12, &d89)
					} else {
						r13 := ctx.AllocReg()
						ctx.EmitCmpInt64(d2.Reg, d88.Reg)
						ctx.EmitSetcc(r13, CcL)
						d89 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
						ctx.BindReg(r13, &d89)
					}
					ctx.FreeDesc(&d2)
					ctx.FreeDesc(&d88)
					d90 = d89
					ctx.EnsureDesc(&d90)
					if d90.Loc != LocImm && d90.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d90.Loc == LocImm {
						if d90.Imm.Bool() {
							ps91 := PhiState{General: ps.General}
							ps91.OverlayValues = make([]JITValueDesc, 91)
							ps91.OverlayValues[1] = d1
							ps91.OverlayValues[2] = d2
							ps91.OverlayValues[3] = d3
							ps91.OverlayValues[4] = d4
							ps91.OverlayValues[5] = d5
							ps91.OverlayValues[6] = d6
							ps91.OverlayValues[7] = d7
							ps91.OverlayValues[8] = d8
							ps91.OverlayValues[9] = d9
							ps91.OverlayValues[10] = d10
							ps91.OverlayValues[26] = d26
							ps91.OverlayValues[27] = d27
							ps91.OverlayValues[29] = d29
							ps91.OverlayValues[30] = d30
							ps91.OverlayValues[31] = d31
							ps91.OverlayValues[32] = d32
							ps91.OverlayValues[33] = d33
							ps91.OverlayValues[35] = d35
							ps91.OverlayValues[37] = d37
							ps91.OverlayValues[38] = d38
							ps91.OverlayValues[39] = d39
							ps91.OverlayValues[42] = d42
							ps91.OverlayValues[43] = d43
							ps91.OverlayValues[68] = d68
							ps91.OverlayValues[69] = d69
							ps91.OverlayValues[70] = d70
							ps91.OverlayValues[72] = d72
							ps91.OverlayValues[73] = d73
							ps91.OverlayValues[74] = d74
							ps91.OverlayValues[76] = d76
							ps91.OverlayValues[77] = d77
							ps91.OverlayValues[80] = d80
							ps91.OverlayValues[81] = d81
							ps91.OverlayValues[82] = d82
							ps91.OverlayValues[84] = d84
							ps91.OverlayValues[85] = d85
							ps91.OverlayValues[86] = d86
							ps91.OverlayValues[87] = d87
							ps91.OverlayValues[88] = d88
							ps91.OverlayValues[89] = d89
							ps91.OverlayValues[90] = d90
							return bbs[4].RenderPS(ps91)
						}
						ps92 := PhiState{General: ps.General}
						ps92.OverlayValues = make([]JITValueDesc, 91)
						ps92.OverlayValues[1] = d1
						ps92.OverlayValues[2] = d2
						ps92.OverlayValues[3] = d3
						ps92.OverlayValues[4] = d4
						ps92.OverlayValues[5] = d5
						ps92.OverlayValues[6] = d6
						ps92.OverlayValues[7] = d7
						ps92.OverlayValues[8] = d8
						ps92.OverlayValues[9] = d9
						ps92.OverlayValues[10] = d10
						ps92.OverlayValues[26] = d26
						ps92.OverlayValues[27] = d27
						ps92.OverlayValues[29] = d29
						ps92.OverlayValues[30] = d30
						ps92.OverlayValues[31] = d31
						ps92.OverlayValues[32] = d32
						ps92.OverlayValues[33] = d33
						ps92.OverlayValues[35] = d35
						ps92.OverlayValues[37] = d37
						ps92.OverlayValues[38] = d38
						ps92.OverlayValues[39] = d39
						ps92.OverlayValues[42] = d42
						ps92.OverlayValues[43] = d43
						ps92.OverlayValues[68] = d68
						ps92.OverlayValues[69] = d69
						ps92.OverlayValues[70] = d70
						ps92.OverlayValues[72] = d72
						ps92.OverlayValues[73] = d73
						ps92.OverlayValues[74] = d74
						ps92.OverlayValues[76] = d76
						ps92.OverlayValues[77] = d77
						ps92.OverlayValues[80] = d80
						ps92.OverlayValues[81] = d81
						ps92.OverlayValues[82] = d82
						ps92.OverlayValues[84] = d84
						ps92.OverlayValues[85] = d85
						ps92.OverlayValues[86] = d86
						ps92.OverlayValues[87] = d87
						ps92.OverlayValues[88] = d88
						ps92.OverlayValues[89] = d89
						ps92.OverlayValues[90] = d90
						return bbs[5].RenderPS(ps92)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d93 := ps.PhiValues[0]
							ctx.EnsureDesc(&d93)
							ctx.EmitStoreScmerToStack(d93, int32(bbs[6].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d94 := ps.PhiValues[1]
							ctx.EnsureDesc(&d94)
							ctx.EmitStoreToStack(d94, int32(bbs[6].PhiBase)+int32(16))
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d90.Reg, 0)
					ctx.EmitJcc(CcNE, lbl12)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl6)
					ps95 := PhiState{General: true}
					ps95.OverlayValues = make([]JITValueDesc, 95)
					ps95.OverlayValues[1] = d1
					ps95.OverlayValues[2] = d2
					ps95.OverlayValues[3] = d3
					ps95.OverlayValues[4] = d4
					ps95.OverlayValues[5] = d5
					ps95.OverlayValues[6] = d6
					ps95.OverlayValues[7] = d7
					ps95.OverlayValues[8] = d8
					ps95.OverlayValues[9] = d9
					ps95.OverlayValues[10] = d10
					ps95.OverlayValues[26] = d26
					ps95.OverlayValues[27] = d27
					ps95.OverlayValues[29] = d29
					ps95.OverlayValues[30] = d30
					ps95.OverlayValues[31] = d31
					ps95.OverlayValues[32] = d32
					ps95.OverlayValues[33] = d33
					ps95.OverlayValues[35] = d35
					ps95.OverlayValues[37] = d37
					ps95.OverlayValues[38] = d38
					ps95.OverlayValues[39] = d39
					ps95.OverlayValues[42] = d42
					ps95.OverlayValues[43] = d43
					ps95.OverlayValues[68] = d68
					ps95.OverlayValues[69] = d69
					ps95.OverlayValues[70] = d70
					ps95.OverlayValues[72] = d72
					ps95.OverlayValues[73] = d73
					ps95.OverlayValues[74] = d74
					ps95.OverlayValues[76] = d76
					ps95.OverlayValues[77] = d77
					ps95.OverlayValues[80] = d80
					ps95.OverlayValues[81] = d81
					ps95.OverlayValues[82] = d82
					ps95.OverlayValues[84] = d84
					ps95.OverlayValues[85] = d85
					ps95.OverlayValues[86] = d86
					ps95.OverlayValues[87] = d87
					ps95.OverlayValues[88] = d88
					ps95.OverlayValues[89] = d89
					ps95.OverlayValues[90] = d90
					ps95.OverlayValues[93] = d93
					ps95.OverlayValues[94] = d94
					ps96 := PhiState{General: true}
					ps96.OverlayValues = make([]JITValueDesc, 95)
					ps96.OverlayValues[1] = d1
					ps96.OverlayValues[2] = d2
					ps96.OverlayValues[3] = d3
					ps96.OverlayValues[4] = d4
					ps96.OverlayValues[5] = d5
					ps96.OverlayValues[6] = d6
					ps96.OverlayValues[7] = d7
					ps96.OverlayValues[8] = d8
					ps96.OverlayValues[9] = d9
					ps96.OverlayValues[10] = d10
					ps96.OverlayValues[26] = d26
					ps96.OverlayValues[27] = d27
					ps96.OverlayValues[29] = d29
					ps96.OverlayValues[30] = d30
					ps96.OverlayValues[31] = d31
					ps96.OverlayValues[32] = d32
					ps96.OverlayValues[33] = d33
					ps96.OverlayValues[35] = d35
					ps96.OverlayValues[37] = d37
					ps96.OverlayValues[38] = d38
					ps96.OverlayValues[39] = d39
					ps96.OverlayValues[42] = d42
					ps96.OverlayValues[43] = d43
					ps96.OverlayValues[68] = d68
					ps96.OverlayValues[69] = d69
					ps96.OverlayValues[70] = d70
					ps96.OverlayValues[72] = d72
					ps96.OverlayValues[73] = d73
					ps96.OverlayValues[74] = d74
					ps96.OverlayValues[76] = d76
					ps96.OverlayValues[77] = d77
					ps96.OverlayValues[80] = d80
					ps96.OverlayValues[81] = d81
					ps96.OverlayValues[82] = d82
					ps96.OverlayValues[84] = d84
					ps96.OverlayValues[85] = d85
					ps96.OverlayValues[86] = d86
					ps96.OverlayValues[87] = d87
					ps96.OverlayValues[88] = d88
					ps96.OverlayValues[89] = d89
					ps96.OverlayValues[90] = d90
					ps96.OverlayValues[93] = d93
					ps96.OverlayValues[94] = d94
					snap97 := d1
					snap98 := d2
					snap99 := d3
					snap100 := d4
					snap101 := d5
					snap102 := d6
					snap103 := d7
					snap104 := d8
					snap105 := d9
					snap106 := d10
					snap107 := d26
					snap108 := d27
					snap109 := d29
					snap110 := d30
					snap111 := d31
					snap112 := d32
					snap113 := d33
					snap114 := d35
					snap115 := d37
					snap116 := d38
					snap117 := d39
					snap118 := d42
					snap119 := d43
					snap120 := d68
					snap121 := d69
					snap122 := d70
					snap123 := d72
					snap124 := d73
					snap125 := d74
					snap126 := d76
					snap127 := d77
					snap128 := d80
					snap129 := d81
					snap130 := d82
					snap131 := d84
					snap132 := d85
					snap133 := d86
					snap134 := d87
					snap135 := d88
					snap136 := d89
					snap137 := d90
					snap138 := d93
					snap139 := d94
					alloc140 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps96)
					}
					ctx.RestoreAllocState(alloc140)
					d1 = snap97
					d2 = snap98
					d3 = snap99
					d4 = snap100
					d5 = snap101
					d6 = snap102
					d7 = snap103
					d8 = snap104
					d9 = snap105
					d10 = snap106
					d26 = snap107
					d27 = snap108
					d29 = snap109
					d30 = snap110
					d31 = snap111
					d32 = snap112
					d33 = snap113
					d35 = snap114
					d37 = snap115
					d38 = snap116
					d39 = snap117
					d42 = snap118
					d43 = snap119
					d68 = snap120
					d69 = snap121
					d70 = snap122
					d72 = snap123
					d73 = snap124
					d74 = snap125
					d76 = snap126
					d77 = snap127
					d80 = snap128
					d81 = snap129
					d82 = snap130
					d84 = snap131
					d85 = snap132
					d86 = snap133
					d87 = snap134
					d88 = snap135
					d89 = snap136
					d90 = snap137
					d93 = snap138
					d94 = snap139
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps95)
					}
					return result
					ctx.FreeDesc(&d89)
					return result
				}
				argPinned141 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned141 = append(argPinned141, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned141 = append(argPinned141, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned141 = append(argPinned141, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned141 = append(argPinned141, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned141 {
						ctx.UnprotectReg(r)
					}
				}()
				ps142 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps142)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(32))
				return result
			},
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "produce",

		Fn: func(a ...Scmer) Scmer {
			result := make([]Scmer, 0)
			state := a[0]
			condition := OptimizeProcToSerialFunction(a[1])
			iterator := OptimizeProcToSerialFunction(a[2])
			for condition(state).Bool() {
				result = append(result, state)
				state = iterator(state)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list that contains produced items - it works like for(state = startstate, condition(state), state = iterator(state)) {yield state}",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "startstate", Description: "start state to begin with"},
				{Kind: "func", Label: "condition", Description: "func that returns true whether the state will be inserted into the result or the loop is stopped", Params: []*TypeDescriptor{{Kind: "any", Label: "state"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", Label: "iterator", Description: "func that produces the next state", Params: []*TypeDescriptor{{Kind: "any", Label: "state"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "produceN",

		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			result := make([]Scmer, n)
			if len(a) > 1 && !a[1].IsNil() {
				// fused produceN+map: generate and transform in one pass
				fn := OptimizeProcToSerialFunction(a[1])
				for i := 0; i < n; i++ {
					result[i] = fn(NewInt(int64(i)))
				}
			} else {
				for i := 0; i < n; i++ {
					result[i] = NewInt(int64(i))
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list with numbers from 0..n-1, optionally mapped through a function",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "n", Description: "number of elements to produce"},
				{Kind: "func", Label: "fn", Description: "(optional) map function applied to each index", Optional: true, Params: []*TypeDescriptor{{Kind: "int", Label: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeProduceN,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallelN",

		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			result := make([]Scmer, n)
			fn := a[1]
			needsSerializedCall := fn.GetTag() == tagFunc || fn.GetTag() == tagFuncEnv
			var fnMu sync.Mutex
			callFn := func(i int) Scmer {
				if needsSerializedCall {
					fnMu.Lock()
					defer fnMu.Unlock()
				}
				return Apply(fn, NewInt(int64(i)))
			}
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			if workers > n {
				workers = n
			}
			jobs := make(chan int, workers)
			errs := make(chan any, workers)
			var wg sync.WaitGroup
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						func() {
							defer func() {
								if r := recover(); r != nil {
									errs <- r
								}
							}()
							result[i] = callFn(i)
						}()
					}
				}()
			}
			for i := 0; i < n; i++ {
				jobs <- i
			}
			close(jobs)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					panic(err)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a list with numbers from 0..n-1 mapped in parallel through a function",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "n", Description: "number of elements to produce"},
				{Kind: "func", Label: "fn", Description: "map function applied to each index in parallel", Params: []*TypeDescriptor{{Kind: "int", Label: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeParallelN,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "produceN_mut",

		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			fn := OptimizeProcToSerialFunction(a[1])
			if len(a) < 3 || a[2].IsNil() {
				for i := 0; i < n; i++ {
					fn(NewInt(int64(i)))
				}
				return NewNil()
			}
			result := asSlice(a[2], "produceN_mut target")
			if len(result) < n {
				panic("produceN_mut target too small")
			}
			result = result[:n]
			for i := 0; i < n; i++ {
				result[i] = fn(NewInt(int64(i)))
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place produceN variant (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "n", Description: "number of elements to produce"},
				{Kind: "func", Label: "fn", Description: "map function applied to each index", Params: []*TypeDescriptor{{Kind: "int", Label: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "list", Label: "target", Description: "(optional) preallocated target list", NoEscape: true, Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "list"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallelN_mut",

		Fn: func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			fn := a[1]
			needsSerializedCall := fn.GetTag() == tagFunc || fn.GetTag() == tagFuncEnv
			var fnMu sync.Mutex
			callFn := func(i int) Scmer {
				if needsSerializedCall {
					fnMu.Lock()
					defer fnMu.Unlock()
				}
				return Apply(fn, NewInt(int64(i)))
			}
			workers := runtime.NumCPU()
			if workers < 1 {
				workers = 1
			}
			if workers > n {
				workers = n
			}
			if len(a) < 3 || a[2].IsNil() {
				jobs := make(chan int, workers)
				errs := make(chan any, workers)
				var wg sync.WaitGroup
				for w := 0; w < workers; w++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for i := range jobs {
							func() {
								defer func() {
									if r := recover(); r != nil {
										errs <- r
									}
								}()
								callFn(i)
							}()
						}
					}()
				}
				for i := 0; i < n; i++ {
					jobs <- i
				}
				close(jobs)
				wg.Wait()
				close(errs)
				for err := range errs {
					if err != nil {
						panic(err)
					}
				}
				return NewNil()
			}
			result := asSlice(a[2], "parallelN_mut target")
			if len(result) < n {
				panic("parallelN_mut target too small")
			}
			result = result[:n]
			jobs := make(chan int, workers)
			errs := make(chan any, workers)
			var wg sync.WaitGroup
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						func() {
							defer func() {
								if r := recover(); r != nil {
									errs <- r
								}
							}()
							result[i] = callFn(i)
						}()
					}
				}()
			}
			for i := 0; i < n; i++ {
				jobs <- i
			}
			close(jobs)
			wg.Wait()
			close(errs)
			for err := range errs {
				if err != nil {
					panic(err)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place parallelN variant (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "n", Description: "number of elements to produce"},
				{Kind: "func", Label: "fn", Description: "map function applied to each index in parallel", Params: []*TypeDescriptor{{Kind: "int", Label: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "list", Label: "target", Description: "(optional) preallocated target list", NoEscape: true, Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "list"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "list?",

		Fn: func(a ...Scmer) Scmer {
			if a[0].IsSlice() {
				return NewBool(true)
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "checks if a value is a list",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "value", Description: "value to check"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [3]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					ctx.ReclaimUntrackedRegs()
					d0 = args[0]
					d0.ID = 0
					d2 = d0
					d2.ID = 0
					d1 = ctx.EmitTagEqualsBorrowed(&d2, tagSlice, JITValueDesc{Loc: LocAny})
					ctx.FreeDesc(&d0)
					d3 = d1
					ctx.EnsureDesc(&d3)
					if d3.Loc != LocImm && d3.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d3.Loc == LocImm {
						if d3.Imm.Bool() {
							ps4 := PhiState{General: ps.General}
							ps4.OverlayValues = make([]JITValueDesc, 4)
							ps4.OverlayValues[0] = d0
							ps4.OverlayValues[1] = d1
							ps4.OverlayValues[2] = d2
							ps4.OverlayValues[3] = d3
							return bbs[1].RenderPS(ps4)
						}
						ps5 := PhiState{General: ps.General}
						ps5.OverlayValues = make([]JITValueDesc, 4)
						ps5.OverlayValues[0] = d0
						ps5.OverlayValues[1] = d1
						ps5.OverlayValues[2] = d2
						ps5.OverlayValues[3] = d3
						return bbs[2].RenderPS(ps5)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl4 := ctx.ReserveLabel()
					lbl5 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, 0)
					ctx.EmitJcc(CcNE, lbl4)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl4)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ps6 := PhiState{General: true}
					ps6.OverlayValues = make([]JITValueDesc, 4)
					ps6.OverlayValues[0] = d0
					ps6.OverlayValues[1] = d1
					ps6.OverlayValues[2] = d2
					ps6.OverlayValues[3] = d3
					ps7 := PhiState{General: true}
					ps7.OverlayValues = make([]JITValueDesc, 4)
					ps7.OverlayValues[0] = d0
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					snap8 := d0
					snap9 := d1
					snap10 := d2
					snap11 := d3
					alloc12 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps7)
					}
					ctx.RestoreAllocState(alloc12)
					d0 = snap8
					d1 = snap9
					d2 = snap10
					d3 = snap11
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps6)
					}
					return result
					ctx.FreeDesc(&d1)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.EnsureDesc(&d13)
					ctx.EmitMakeBool(result, d13)
					if d13.Loc == LocReg {
						ctx.FreeReg(d13.Reg)
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					ctx.ReclaimUntrackedRegs()
					d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
					ctx.EnsureDesc(&d14)
					ctx.EmitMakeBool(result, d14)
					if d14.Loc == LocReg {
						ctx.FreeReg(d14.Reg)
					}
					result.Type = tagBool
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned15 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned15 = append(argPinned15, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned15 = append(argPinned15, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned15 = append(argPinned15, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned15 = append(argPinned15, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned15 {
						ctx.UnprotectReg(r)
					}
				}()
				ps16 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps16)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "contains?",

		Fn: func(a ...Scmer) Scmer {
			arr := asSlice(a[0], "contains?")
			for _, v := range arr {
				if Equal(v, a[1]) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "checks if a value is in a list; uses the equal?? operator",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list to check", NoEscape: true},
				{Kind: "any", Label: "value", Description: "value to check"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_in",

		Fn: func(a ...Scmer) Scmer {
			values := asSlice(a[0], "sql_in")
			if a[1].IsNil() {
				return NewNil()
			}
			unknown := false
			for _, value := range values {
				equal := EqualSQL(value, a[1])
				if equal.IsNil() {
					unknown = true
				} else if equal.Bool() {
					return NewBool(true)
				}
			}
			if unknown {
				return NewNil()
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tests SQL IN-list membership and returns nil when NULL makes the result UNKNOWN",
			Params: []*TypeDescriptor{{Kind: "list", Label: "values", Description: "SQL IN-list values", NoEscape: true}, {Kind: "any", Label: "value", Description: "value to find"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})

	// dictionary functions
	DeclareTitle("Associative Lists / Dictionaries")

	Declare(&Globalenv, &Declaration{
		Name: "filter_assoc",

		Fn: func(a ...Scmer) Scmer {
			result := make([]Scmer, 0)
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "filter_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						result = append(result, slice[i], slice[i+1])
					}
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = append(result, k, v)
					}
					return true
				})
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a filtered dictionary according to a filter function",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary that has to be filtered", NoEscape: true},
				{Kind: "func", Label: "condition", Description: "returns whether a dictionary entry should be included", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "bool", Label: "included", Description: "whether to include the entry"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 FirstParameterMutable("filter_assoc_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "find_assoc",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "find_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						return NewSlice([]Scmer{slice[i], slice[i+1]})
					}
				}
			} else {
				var result Scmer
				found := false
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = NewSlice([]Scmer{k, v})
						found = true
						return false
					}
					return true
				})
				if found {
					return result
				}
			}
			if len(a) >= 3 {
				return a[2]
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns the first key/value pair that passes the condition function, or nil/default if none matches",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary to search", NoEscape: true},
				{Kind: "func", Label: "condition", Description: "predicate applied until the first matching dictionary entry", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "bool", Label: "matches", Description: "whether the entry matches"}},
				{Kind: "any", Label: "default", Description: "optional default value if nothing matches", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map_assoc",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "map_assoc"); fd == nil {
				result := make([]Scmer, len(slice))
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						result[i] = v
						key = v
					} else {
						result[i] = fn(key, v)
					}
				}
				return NewSlice(result)
			} else {
				result := make([]Scmer, 0, len(fd.Pairs))
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, k, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a mapped dictionary according to a map function\nKeys will stay the same but values are mapped.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary that has to be mapped", NoEscape: true},
				{Kind: "func", Label: "map", Description: "transforms each dictionary value", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "any", Label: "mapped_value", Description: "replacement value"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeAssocFixedLengthInput("map_assoc_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_assoc",

		Fn: func(a ...Scmer) Scmer {
			result := a[2]
			reduce := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "reduce_assoc"); fd == nil {
				if len(slice)%2 != 0 {
					panic(fmt.Sprintf("reduce_assoc received odd-length dict (%d): %s", len(slice), SerializeToString(a[0], &Globalenv)))
				}
				for i := 0; i < len(slice); i += 2 {
					result = reduce(result, slice[i], slice[i+1])
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool { result = reduce(result, k, v); return true })
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "reduces a dictionary according to a reduce function",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary that has to be reduced", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Kind: "any", Transfer: true, Label: "acc", Description: "current accumulator"}, {Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Label: "reduce", Description: "combines the accumulator with each dictionary entry", Return: &TypeDescriptor{Kind: "any", Label: "acc", Description: "next accumulator"}},
				{Kind: "any", Label: "neutral", Description: "initial value for the accumulator"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "make_structural_index",

		Fn: NewStructuralIndex,
		Type: &TypeDescriptor{Kind: "func", Description: "Builds an immutable structural-expression index. It eagerly hashes every key and every node under roots, then returns a parallel-safe lookup function that maps an equal expression to its zero-based key position or nil.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "keys", Description: "immutable structural expressions to index"},
				{Kind: "list", Label: "roots", Description: "immutable expression roots whose descendant hashes are precomputed"},
			},
			Return: &TypeDescriptor{
				Kind:        "func",
				Label:       "lookup",
				Description: "looks up the indexed position of a structurally equal expression",
				Params: []*TypeDescriptor{
					{Kind: "any", Label: "expression", Description: "a key, root, descendant of a declared root, or scalar expression"},
				},
				Return: &TypeDescriptor{Kind: "int|nil", Label: "position", Description: "zero-based key position, or nil when the expression is not indexed"},
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "make_structural_catalog",

		Fn: NewStructuralCatalog,
		Type: &TypeDescriptor{Kind: "func", Description: "Creates an atomic compile-local structural catalog. Look up with (catalog key), insert with (catalog key value), or freeze with (catalog) for parallel-safe read-only lookup.",
			Params: []*TypeDescriptor{
				{Kind: "bool|symbol", Label: "mode", Description: "true forces collisions for tests; ast selects type-stable compiler equality", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "func", Label: "catalog", Description: "atomic structural-expression lookup and update function",
				Params: []*TypeDescriptor{
					{Kind: "any", Label: "key", Description: "expression to look up; omit to freeze the catalog", Optional: true},
					{Kind: "any", Label: "value", Description: "value to store for key", Optional: true},
				},
				Return: &TypeDescriptor{Kind: "any|func", Label: "result", Description: "stored value, lookup result, or frozen lookup function",
					Params: []*TypeDescriptor{
						{Kind: "any", Label: "key", Description: "expression to look up in the frozen catalog"},
					},
					Return: &TypeDescriptor{Kind: "any", Label: "value", Description: "value stored for a structurally equal expression, or nil"},
				},
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "has_assoc?",

		Fn: func(a ...Scmer) Scmer {
			if slice, fd := asAssoc(a[0], "has_assoc?"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if Equal(slice[i], a[1]) {
						return NewBool(true)
					}
				}
			} else {
				if _, ok := fd.Get(a[1]); ok {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "checks if a dictionary has a key present",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary that has to be checked", NoEscape: true},
				{Kind: "string", Label: "key", Description: "key to test"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "get_assoc",

		Fn: func(a ...Scmer) Scmer {
			if slice, fd := asAssoc(a[0], "get_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if Equal(slice[i], a[1]) {
						return slice[i+1]
					}
				}
			} else {
				if v, ok := fd.Get(a[1]); ok {
					return v
				}
			}
			// Return default value if provided, otherwise nil
			if len(a) >= 3 {
				return a[2]
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "gets a value from a dictionary by key, returns nil if not found",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary to look up", NoEscape: true},
				{Kind: "any", Label: "key", Description: "key to look up"},
				{Kind: "any", Label: "default", Description: "optional default value if key not found", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "get_assoc_pairlist",

		Fn: func(a ...Scmer) Scmer {
			for _, entry := range asSlice(a[0], "get_assoc_pairlist") {
				if !entry.IsSlice() {
					continue
				}
				row := entry.Slice()
				if len(row) == 0 || !Equal(row[0], a[1]) {
					continue
				}
				if len(row) == 2 {
					return row[1]
				}
				return NewSlice(row[1:])
			}
			return a[2]
		},
		Type: &TypeDescriptor{Kind: "func", Description: "gets a value from a list of key/value rows without flattening the rows",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "rows", Description: "list whose rows contain a key followed by one or more values", NoEscape: true},
				{Kind: "any", Label: "key", Description: "key compared with the first item of each row"},
				{Kind: "any", Label: "default", Description: "value returned when no row contains the key"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "extract_assoc",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "extract_assoc"); fd == nil {
				result := make([]Scmer, len(slice)/2)
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						key = v
					} else {
						result[i/2] = fn(key, v)
					}
				}
				return NewSlice(result)
			} else {
				result := make([]Scmer, 0, len(fd.Pairs)/2)
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "applies a function (key value) on the dictionary and returns the results as a flat list",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary that has to be checked", NoEscape: true},
				{Kind: "func", Label: "map", Description: "extracts one element per dictionary entry", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "any", Label: "element", Description: "element extracted from the entry"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 optimizeExtractAssoc,
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "set_assoc",

		Fn: func(a ...Scmer) Scmer {
			var mergeFn func(Scmer, Scmer) Scmer
			if len(a) > 3 {
				mfn := OptimizeProcToSerialFunction(a[3])
				mergeFn = func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
			}
			slice, fd := asAssoc(a[0], "set_assoc")
			if fd == nil {
				// defensive copy — set_assoc must not mutate the original
				list := append([]Scmer{}, slice...)
				for i := 0; i < len(list); i += 2 {
					if Equal(list[i], a[1]) {
						if mergeFn != nil {
							list[i+1] = mergeFn(list[i+1], a[2])
						} else {
							list[i+1] = a[2]
						}
						return NewSlice(list)
					}
				}
				list = append(list, a[1], a[2])
				if len(list) >= 10 {
					fd := NewFastDictValue(len(list)/2 + 4)
					for i := 0; i < len(list); i += 2 {
						fd.Set(list[i], list[i+1], nil)
					}
					return NewFastDict(fd)
				}
				return NewSlice(list)
			} else {
				fd = fd.Copy()
				fd.Set(a[1], a[2], mergeFn)
				return NewFastDict(fd)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a new dictionary where a single value has been changed.\nThe original dictionary is not modified.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "input dictionary"},
				{Kind: "string", Label: "key", Description: "key that has to be set"},
				{Kind: "any", Label: "value", Description: "new value to set"},
				{Kind: "func", Label: "merge", Description: "combines values when an existing entry is overwritten", Optional: true, Params: []*TypeDescriptor{{Kind: "any", Label: "old", Description: "existing value"}, {Kind: "any", Label: "new", Description: "replacement value"}}, Return: &TypeDescriptor{Kind: "any", Label: "merged", Description: "value stored in the new dictionary"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 FirstParameterMutable("set_assoc_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge_assoc",

		Fn: func(a ...Scmer) Scmer {
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc"])
			dst := a[0]
			if slice, fd := asAssoc(a[1], "merge_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if len(a) > 2 {
						dst = setAssoc(dst, slice[i], slice[i+1], a[2])
					} else {
						dst = setAssoc(dst, slice[i], slice[i+1])
					}
				}
			} else {
				if len(a) > 2 {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v, a[2]); return true })
				} else {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v); return true })
				}
			}
			return dst
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a dictionary where all keys from dict1 and all keys from dict2 are present.\nIf a key is present in both inputs, the second one will be dominant so the first value will be overwritten unless you provide a merge function",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict1", Description: "first input dictionary that has to be changed. You must not use this value again."},
				{Kind: "list", Label: "dict2", Description: "input dictionary that contains the new values that have to be added"},
				{Kind: "func", Label: "merge", Description: "combines values when both dictionaries contain an entry", Optional: true, Params: []*TypeDescriptor{{Kind: "any", Label: "old", Description: "value from the first dictionary"}, {Kind: "any", Label: "new", Description: "value from the second dictionary"}}, Return: &TypeDescriptor{Kind: "any", Label: "merged", Description: "value stored in the merged dictionary"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 FirstParameterMutable("merge_assoc_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})

	// Fused physical operators: optimizer-only, forbidden from .scm code.
	Declare(&Globalenv, &Declaration{
		Name: "reduce_segments",

		Fn: func(a ...Scmer) Scmer {
			segments := asSlice(a[0], "reduce_segments")
			for _, segment := range segments {
				asSlice(segment, "reduce_segments item")
			}
			fn := OptimizeProcToSerialFunction(a[1])
			result := NewNil()
			hasResult := len(a) > 2
			if hasResult {
				result = a[2]
			}
			for _, segment := range segments {
				for _, item := range asSlice(segment, "reduce_segments item") {
					if !hasResult {
						result = item
						hasResult = true
						continue
					}
					result = fn(result, item)
				}
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "reduces ordered list segments without flattening them (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "segments", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Transfer: true, Label: "acc"}, {Label: "item"}}, Label: "reduce", Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "neutral", Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "filter_map",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "filter_map")
			mapper := OptimizeProcToSerialFunction(a[1])
			predicate := OptimizeProcToSerialFunction(a[2])
			result := make([]Scmer, 0, len(input))
			for _, item := range input {
				mapped := mapper(item)
				if predicate(mapped).Bool() {
					result = append(result, mapped)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial map and filter (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "condition", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d54 JITValueDesc
				_ = d54
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d128 JITValueDesc
				_ = d128
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d141 JITValueDesc
				_ = d141
				var d142 JITValueDesc
				_ = d142
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(40))
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				var bbs [5]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[0].VisitCount >= 0 {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
				}
				bbs[0].VisitCount++
				if ps.General {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_0 = bbs[0].Address
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				ctx.ReclaimUntrackedRegs()
				d3 = args[0]
				d3.ID = 0
				var d4 JITValueDesc
				if d3.Type == tagSlice {
					d4 = jitKnownSliceHeader(ctx, &d3)
				} else {
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d3}, 3)
				}
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.BindReg(d4.Reg3, &d4)
				ctx.FreeDesc(&d3)
				d5 = args[1]
				d5.ID = 0
				var d6 JITValueDesc
				if d5.Loc == LocLambdaTemplate {
					d6 = d5
				} else {
					d6 = ctx.RequestOptimizedCallback(1)
				}
				ctx.FreeDesc(&d5)
				d7 = args[2]
				d7.ID = 0
				var d8 JITValueDesc
				if d7.Loc == LocLambdaTemplate {
					d8 = d7
				} else {
					d8 = ctx.RequestOptimizedCallback(2)
				}
				ctx.FreeDesc(&d7)
				var d9 JITValueDesc
				if d4.SliceSizeKnown {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d9)
				d11 = ctx.EmitGoCallScalar(GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d10, d9}, 3)
				ctx.BindReg(d11.Reg, &d11)
				ctx.BindReg(d11.Reg2, &d11)
				ctx.BindReg(d11.Reg3, &d11)
				ctx.FreeDesc(&d9)
				var d12 JITValueDesc
				if d4.SliceSizeKnown {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.EnsureDesc(&d11)
				if d11.Loc == LocReg {
					ctx.ProtectReg(d11.Reg)
				} else if d11.Loc == LocRegPair {
					ctx.ProtectReg(d11.Reg)
					ctx.ProtectReg(d11.Reg2)
				}
				d13 = d11
				if d13.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d13)
				if d13.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d13.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d13.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d13.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(24))
				if d11.Loc == LocReg {
					ctx.UnprotectReg(d11.Reg)
				} else if d11.Loc == LocRegPair {
					ctx.UnprotectReg(d11.Reg)
					ctx.UnprotectReg(d11.Reg2)
				}
				ps14 := PhiState{General: ps.General}
				ps14.OverlayValues = make([]JITValueDesc, 14)
				ps14.OverlayValues[1] = d1
				ps14.OverlayValues[2] = d2
				ps14.OverlayValues[3] = d3
				ps14.OverlayValues[4] = d4
				ps14.OverlayValues[5] = d5
				ps14.OverlayValues[6] = d6
				ps14.OverlayValues[7] = d7
				ps14.OverlayValues[8] = d8
				ps14.OverlayValues[9] = d9
				ps14.OverlayValues[10] = d10
				ps14.OverlayValues[11] = d11
				ps14.OverlayValues[12] = d12
				ps14.OverlayValues[13] = d13
				ps14.PhiValues = make([]JITValueDesc, 2)
				d15 = d11
				ps14.PhiValues[0] = d15
				d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
				ps14.PhiValues[1] = d16
				if ps14.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps14)
				return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d17 := ps.PhiValues[0]
						ctx.EnsureDesc(&d17)
						ctx.EmitStoreRegMem(d17.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d17.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d17.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d18 := ps.PhiValues[1]
						ctx.EnsureDesc(&d18)
						ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(24))
					}
					if bbs[1].VisitCount >= 0 {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
				}
				bbs[1].VisitCount++
				if ps.General {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_1 = bbs[1].Address
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d1 = ps.PhiValues[0]
				}
				if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d2 = ps.PhiValues[1]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs19 := make([]Reg, 0, 3)
				seenBlockPinnedRegs20 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs20
				for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
					live := d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3)
					if live && !seenBlockPinnedRegs20[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs20[r] = true
						blockPinnedRegs19 = append(blockPinnedRegs19, r)
					}
				}
				unpinBlockRegs21 := func() { for _, r := range blockPinnedRegs19 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs21()
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				var d22 JITValueDesc
				if d2.Loc == LocImm {
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d2.Reg)
					ctx.EmitMovRegReg(scratch, d2.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d22)
				}
				if d22.Loc == LocReg && d2.Loc == LocReg && d22.Reg == d2.Reg {
					ctx.TransferReg(d2.Reg)
					d2.Loc = LocNone
				}
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				var d23 JITValueDesc
				if d22.Loc == LocImm && d12.Loc == LocImm {
					d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d22.Imm.Int() < d12.Imm.Int())}
				} else if d12.Loc == LocImm {
					r0 := ctx.AllocRegExcept(d22.Reg)
					if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d22.Reg, int32(d12.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
						ctx.EmitCmpInt64(d22.Reg, RegR11)
					}
					ctx.EmitSetcc(r0, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
					ctx.BindReg(r0, &d23)
				} else if d22.Loc == LocImm {
					r1 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d12.Reg)
					ctx.EmitSetcc(r1, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
					ctx.BindReg(r1, &d23)
				} else {
					r2 := ctx.AllocRegExcept(d22.Reg)
					ctx.EmitCmpInt64(d22.Reg, d12.Reg)
					ctx.EmitSetcc(r2, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
					ctx.BindReg(r2, &d23)
				}
				ctx.FreeDesc(&d12)
				d24 = d23
				ctx.EnsureDesc(&d24)
				if d24.Loc != LocImm && d24.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d24.Loc == LocImm {
					if d24.Imm.Bool() {
				ps25 := PhiState{General: ps.General}
				ps25.OverlayValues = make([]JITValueDesc, 25)
				ps25.OverlayValues[1] = d1
				ps25.OverlayValues[2] = d2
				ps25.OverlayValues[3] = d3
				ps25.OverlayValues[4] = d4
				ps25.OverlayValues[5] = d5
				ps25.OverlayValues[6] = d6
				ps25.OverlayValues[7] = d7
				ps25.OverlayValues[8] = d8
				ps25.OverlayValues[9] = d9
				ps25.OverlayValues[10] = d10
				ps25.OverlayValues[11] = d11
				ps25.OverlayValues[12] = d12
				ps25.OverlayValues[13] = d13
				ps25.OverlayValues[15] = d15
				ps25.OverlayValues[16] = d16
				ps25.OverlayValues[17] = d17
				ps25.OverlayValues[18] = d18
				ps25.OverlayValues[22] = d22
				ps25.OverlayValues[23] = d23
				ps25.OverlayValues[24] = d24
						return bbs[2].RenderPS(ps25)
					}
				ps26 := PhiState{General: ps.General}
				ps26.OverlayValues = make([]JITValueDesc, 25)
				ps26.OverlayValues[1] = d1
				ps26.OverlayValues[2] = d2
				ps26.OverlayValues[3] = d3
				ps26.OverlayValues[4] = d4
				ps26.OverlayValues[5] = d5
				ps26.OverlayValues[6] = d6
				ps26.OverlayValues[7] = d7
				ps26.OverlayValues[8] = d8
				ps26.OverlayValues[9] = d9
				ps26.OverlayValues[10] = d10
				ps26.OverlayValues[11] = d11
				ps26.OverlayValues[12] = d12
				ps26.OverlayValues[13] = d13
				ps26.OverlayValues[15] = d15
				ps26.OverlayValues[16] = d16
				ps26.OverlayValues[17] = d17
				ps26.OverlayValues[18] = d18
				ps26.OverlayValues[22] = d22
				ps26.OverlayValues[23] = d23
				ps26.OverlayValues[24] = d24
					return bbs[3].RenderPS(ps26)
				}
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d27 := ps.PhiValues[0]
						ctx.EnsureDesc(&d27)
						ctx.EmitStoreRegMem(d27.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d27.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d27.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d28 := ps.PhiValues[1]
						ctx.EnsureDesc(&d28)
						ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(24))
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
				lbl6 := ctx.ReserveLabel()
				lbl7 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d24.Reg, 0)
				ctx.EmitJcc(CcNE, lbl6)
				ctx.EmitJmp(lbl7)
				ctx.MarkLabel(lbl6)
				ctx.EmitJmp(lbl3)
				ctx.MarkLabel(lbl7)
				ctx.EmitJmp(lbl4)
				ps29 := PhiState{General: true}
				ps29.OverlayValues = make([]JITValueDesc, 29)
				ps29.OverlayValues[1] = d1
				ps29.OverlayValues[2] = d2
				ps29.OverlayValues[3] = d3
				ps29.OverlayValues[4] = d4
				ps29.OverlayValues[5] = d5
				ps29.OverlayValues[6] = d6
				ps29.OverlayValues[7] = d7
				ps29.OverlayValues[8] = d8
				ps29.OverlayValues[9] = d9
				ps29.OverlayValues[10] = d10
				ps29.OverlayValues[11] = d11
				ps29.OverlayValues[12] = d12
				ps29.OverlayValues[13] = d13
				ps29.OverlayValues[15] = d15
				ps29.OverlayValues[16] = d16
				ps29.OverlayValues[17] = d17
				ps29.OverlayValues[18] = d18
				ps29.OverlayValues[22] = d22
				ps29.OverlayValues[23] = d23
				ps29.OverlayValues[24] = d24
				ps29.OverlayValues[27] = d27
				ps29.OverlayValues[28] = d28
				ps30 := PhiState{General: true}
				ps30.OverlayValues = make([]JITValueDesc, 29)
				ps30.OverlayValues[1] = d1
				ps30.OverlayValues[2] = d2
				ps30.OverlayValues[3] = d3
				ps30.OverlayValues[4] = d4
				ps30.OverlayValues[5] = d5
				ps30.OverlayValues[6] = d6
				ps30.OverlayValues[7] = d7
				ps30.OverlayValues[8] = d8
				ps30.OverlayValues[9] = d9
				ps30.OverlayValues[10] = d10
				ps30.OverlayValues[11] = d11
				ps30.OverlayValues[12] = d12
				ps30.OverlayValues[13] = d13
				ps30.OverlayValues[15] = d15
				ps30.OverlayValues[16] = d16
				ps30.OverlayValues[17] = d17
				ps30.OverlayValues[18] = d18
				ps30.OverlayValues[22] = d22
				ps30.OverlayValues[23] = d23
				ps30.OverlayValues[24] = d24
				ps30.OverlayValues[27] = d27
				ps30.OverlayValues[28] = d28
				snap31 := d1
				snap32 := d2
				snap33 := d3
				snap34 := d4
				snap35 := d5
				snap36 := d6
				snap37 := d7
				snap38 := d8
				snap39 := d9
				snap40 := d10
				snap41 := d11
				snap42 := d12
				snap43 := d13
				snap44 := d15
				snap45 := d16
				snap46 := d17
				snap47 := d18
				snap48 := d22
				snap49 := d23
				snap50 := d24
				snap51 := d27
				snap52 := d28
				alloc53 := ctx.SnapshotAllocState()
				if !bbs[3].Rendered {
					bbs[3].RenderPS(ps30)
				}
				ctx.RestoreAllocState(alloc53)
				d1 = snap31
				d2 = snap32
				d3 = snap33
				d4 = snap34
				d5 = snap35
				d6 = snap36
				d7 = snap37
				d8 = snap38
				d9 = snap39
				d10 = snap40
				d11 = snap41
				d12 = snap42
				d13 = snap43
				d15 = snap44
				d16 = snap45
				d17 = snap46
				d18 = snap47
				d22 = snap48
				d23 = snap49
				d24 = snap50
				d27 = snap51
				d28 = snap52
				if !bbs[2].Rendered {
					return bbs[2].RenderPS(ps29)
				}
				return result
				ctx.FreeDesc(&d23)
				return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[2].VisitCount >= 0 {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
				}
				bbs[2].VisitCount++
				if ps.General {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_2 = bbs[2].Address
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d22)
				r3 := ctx.AllocReg()
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d4)
				if d22.Loc == LocImm {
					ctx.EmitMovRegImm64(r3, uint64(d22.Imm.Int()) * 16)
				} else {
					ctx.EmitMovRegReg(r3, d22.Reg)
					ctx.EmitShlRegImm8(r3, 4)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitAddInt64(r3, RegR11)
				} else {
					ctx.EmitAddInt64(r3, d4.Reg)
				}
				r4 := ctx.AllocRegExcept(r3)
				r5 := ctx.AllocRegExcept(r3, r4)
				ctx.EmitMovRegMem(r4, r3, 0)
				ctx.EmitMovRegMem(r5, r3, 8)
				ctx.FreeReg(r3)
				d54 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d54)
				ctx.BindReg(r5, &d54)
				stackArray55 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreScmerToStack(d54, int32(stackArray55)+int32(0))
				ctx.FreeDesc(&d54)
				r6 := ctx.AllocReg()
				r7 := ctx.AllocRegExcept(r6)
				r8 := ctx.AllocRegExcept(r6, r7)
				ctx.EmitLeaRegMem(r6, RegRSP, int32(stackArray55))
				ctx.EmitMovRegImm64(r7, uint64(1))
				ctx.EmitMovRegImm64(r8, uint64(1))
				d56 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r6, &d56)
				ctx.BindReg(r7, &d56)
				ctx.BindReg(r8, &d56)
				callbackArgs58 := make([]JITValueDesc, 1)
				callbackArgs58[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray55)+0}
				var d57 JITValueDesc
				ctx.FreeDesc(&d56)
				if d6.Loc == LocLambdaTemplate && d6.Lambda != nil {
					callbackResultOff59 := ctx.AllocSpill(16)
					ctx.setStackPointer(jitStackRootFrameBP, callbackResultOff59, true)
					outerRegs60 := ctx.PreserveOuterRegs()
					d57 = JITEmitProcInlineWithOuter(ctx, &d6.Lambda.Proc, d6.Lambda.Outer, callbackArgs58, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					ctx.EnsureDesc(&d57)
					ctx.EmitStoreRegMem(d57.Reg, RegRBP, callbackResultOff59)
					ctx.EmitStoreRegMem(d57.Reg2, RegRBP, callbackResultOff59+8)
					ctx.RestoreOuterRegs(outerRegs60)
					d57 = JITValueDesc{Loc: LocStackPair, Type: d57.Type, StackOff: callbackResultOff59, NoHeapPointer: d57.NoHeapPointer}
					liveRegs61 := make([]Reg, 0, 21)
					seenLiveRegs62 := make(map[Reg]bool)
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := (d1.Loc == LocReg && r == d1.Reg) ||
							(d1.Loc == LocRegPair && (r == d1.Reg || r == d1.Reg2)) ||
							(d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := (d11.Loc == LocReg && r == d11.Reg) ||
							(d11.Loc == LocRegPair && (r == d11.Reg || r == d11.Reg2)) ||
							(d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d22.Reg, d22.Reg2, d22.Reg3} {
						live := (d22.Loc == LocReg && r == d22.Reg) ||
							(d22.Loc == LocRegPair && (r == d22.Reg || r == d22.Reg2)) ||
							(d22.Loc == LocRegTriple && (r == d22.Reg || r == d22.Reg2 || r == d22.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d4.Reg, d4.Reg2, d4.Reg3} {
						live := (d4.Loc == LocReg && r == d4.Reg) ||
							(d4.Loc == LocRegPair && (r == d4.Reg || r == d4.Reg2)) ||
							(d4.Loc == LocRegTriple && (r == d4.Reg || r == d4.Reg2 || r == d4.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d56.Reg, d56.Reg2, d56.Reg3} {
						live := (d56.Loc == LocReg && r == d56.Reg) ||
							(d56.Loc == LocRegPair && (r == d56.Reg || r == d56.Reg2)) ||
							(d56.Loc == LocRegTriple && (r == d56.Reg || r == d56.Reg2 || r == d56.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d6.Reg, d6.Reg2, d6.Reg3} {
						live := (d6.Loc == LocReg && r == d6.Reg) ||
							(d6.Loc == LocRegPair && (r == d6.Reg || r == d6.Reg2)) ||
							(d6.Loc == LocRegTriple && (r == d6.Reg || r == d6.Reg2 || r == d6.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d8.Reg, d8.Reg2, d8.Reg3} {
						live := (d8.Loc == LocReg && r == d8.Reg) ||
							(d8.Loc == LocRegPair && (r == d8.Reg || r == d8.Reg2)) ||
							(d8.Loc == LocRegTriple && (r == d8.Reg || r == d8.Reg2 || r == d8.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					ctx.EnsureDesc(&d57)
					for _, r := range liveRegs61 { ctx.UnprotectReg(r) }
				} else {
					callbackCallArgs := make([]JITValueDesc, 0, 2)
					callbackCallArgs = append(callbackCallArgs, d6)
					callbackCallArgs = append(callbackCallArgs, callbackArgs58...)
					d57 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
				}
				stackArray63 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d57)
				ctx.EnsureDesc(&d57)
				ctx.EmitStoreScmerToStack(d57, int32(stackArray63)+int32(0))
				r9 := ctx.AllocReg()
				r10 := ctx.AllocRegExcept(r9)
				r11 := ctx.AllocRegExcept(r9, r10)
				ctx.EmitLeaRegMem(r9, RegRSP, int32(stackArray63))
				ctx.EmitMovRegImm64(r10, uint64(1))
				ctx.EmitMovRegImm64(r11, uint64(1))
				d64 = JITValueDesc{Loc: LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r9, &d64)
				ctx.BindReg(r10, &d64)
				ctx.BindReg(r11, &d64)
				callbackArgs66 := make([]JITValueDesc, 1)
				callbackArgs66[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray63)+0}
				var d65 JITValueDesc
				ctx.FreeDesc(&d64)
				if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
					callbackResultOff67 := ctx.AllocSpill(16)
					ctx.setStackPointer(jitStackRootFrameBP, callbackResultOff67, true)
					outerRegs68 := ctx.PreserveOuterRegs()
					d65 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, callbackArgs66, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					ctx.EnsureDesc(&d65)
					ctx.EmitStoreRegMem(d65.Reg, RegRBP, callbackResultOff67)
					ctx.EmitStoreRegMem(d65.Reg2, RegRBP, callbackResultOff67+8)
					ctx.RestoreOuterRegs(outerRegs68)
					d65 = JITValueDesc{Loc: LocStackPair, Type: d65.Type, StackOff: callbackResultOff67, NoHeapPointer: d65.NoHeapPointer}
					liveRegs69 := make([]Reg, 0, 21)
					seenLiveRegs70 := make(map[Reg]bool)
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := (d1.Loc == LocReg && r == d1.Reg) ||
							(d1.Loc == LocRegPair && (r == d1.Reg || r == d1.Reg2)) ||
							(d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3))
						if live && !seenLiveRegs70[r] {
							ctx.ProtectReg(r)
							seenLiveRegs70[r] = true
							liveRegs69 = append(liveRegs69, r)
						}
					}
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := (d11.Loc == LocReg && r == d11.Reg) ||
							(d11.Loc == LocRegPair && (r == d11.Reg || r == d11.Reg2)) ||
							(d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3))
						if live && !seenLiveRegs70[r] {
							ctx.ProtectReg(r)
							seenLiveRegs70[r] = true
							liveRegs69 = append(liveRegs69, r)
						}
					}
					for _, r := range []Reg{d22.Reg, d22.Reg2, d22.Reg3} {
						live := (d22.Loc == LocReg && r == d22.Reg) ||
							(d22.Loc == LocRegPair && (r == d22.Reg || r == d22.Reg2)) ||
							(d22.Loc == LocRegTriple && (r == d22.Reg || r == d22.Reg2 || r == d22.Reg3))
						if live && !seenLiveRegs70[r] {
							ctx.ProtectReg(r)
							seenLiveRegs70[r] = true
							liveRegs69 = append(liveRegs69, r)
						}
					}
					for _, r := range []Reg{d4.Reg, d4.Reg2, d4.Reg3} {
						live := (d4.Loc == LocReg && r == d4.Reg) ||
							(d4.Loc == LocRegPair && (r == d4.Reg || r == d4.Reg2)) ||
							(d4.Loc == LocRegTriple && (r == d4.Reg || r == d4.Reg2 || r == d4.Reg3))
						if live && !seenLiveRegs70[r] {
							ctx.ProtectReg(r)
							seenLiveRegs70[r] = true
							liveRegs69 = append(liveRegs69, r)
						}
					}
					for _, r := range []Reg{d57.Reg, d57.Reg2, d57.Reg3} {
						live := (d57.Loc == LocReg && r == d57.Reg) ||
							(d57.Loc == LocRegPair && (r == d57.Reg || r == d57.Reg2)) ||
							(d57.Loc == LocRegTriple && (r == d57.Reg || r == d57.Reg2 || r == d57.Reg3))
						if live && !seenLiveRegs70[r] {
							ctx.ProtectReg(r)
							seenLiveRegs70[r] = true
							liveRegs69 = append(liveRegs69, r)
						}
					}
					for _, r := range []Reg{d64.Reg, d64.Reg2, d64.Reg3} {
						live := (d64.Loc == LocReg && r == d64.Reg) ||
							(d64.Loc == LocRegPair && (r == d64.Reg || r == d64.Reg2)) ||
							(d64.Loc == LocRegTriple && (r == d64.Reg || r == d64.Reg2 || r == d64.Reg3))
						if live && !seenLiveRegs70[r] {
							ctx.ProtectReg(r)
							seenLiveRegs70[r] = true
							liveRegs69 = append(liveRegs69, r)
						}
					}
					for _, r := range []Reg{d8.Reg, d8.Reg2, d8.Reg3} {
						live := (d8.Loc == LocReg && r == d8.Reg) ||
							(d8.Loc == LocRegPair && (r == d8.Reg || r == d8.Reg2)) ||
							(d8.Loc == LocRegTriple && (r == d8.Reg || r == d8.Reg2 || r == d8.Reg3))
						if live && !seenLiveRegs70[r] {
							ctx.ProtectReg(r)
							seenLiveRegs70[r] = true
							liveRegs69 = append(liveRegs69, r)
						}
					}
					ctx.EnsureDesc(&d65)
					for _, r := range liveRegs69 { ctx.UnprotectReg(r) }
				} else {
					callbackCallArgs := make([]JITValueDesc, 0, 2)
					callbackCallArgs = append(callbackCallArgs, d8)
					callbackCallArgs = append(callbackCallArgs, callbackArgs66...)
					d65 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
				}
				d72 = d65
				d72.ID = 0
				d71 = ctx.EmitBoolDesc(&d72, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d65)
				d73 = d71
				ctx.EnsureDesc(&d73)
				if d73.Loc != LocImm && d73.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d73.Loc == LocImm {
					if d73.Imm.Bool() {
				ps74 := PhiState{General: ps.General}
				ps74.OverlayValues = make([]JITValueDesc, 74)
				ps74.OverlayValues[1] = d1
				ps74.OverlayValues[2] = d2
				ps74.OverlayValues[3] = d3
				ps74.OverlayValues[4] = d4
				ps74.OverlayValues[5] = d5
				ps74.OverlayValues[6] = d6
				ps74.OverlayValues[7] = d7
				ps74.OverlayValues[8] = d8
				ps74.OverlayValues[9] = d9
				ps74.OverlayValues[10] = d10
				ps74.OverlayValues[11] = d11
				ps74.OverlayValues[12] = d12
				ps74.OverlayValues[13] = d13
				ps74.OverlayValues[15] = d15
				ps74.OverlayValues[16] = d16
				ps74.OverlayValues[17] = d17
				ps74.OverlayValues[18] = d18
				ps74.OverlayValues[22] = d22
				ps74.OverlayValues[23] = d23
				ps74.OverlayValues[24] = d24
				ps74.OverlayValues[27] = d27
				ps74.OverlayValues[28] = d28
				ps74.OverlayValues[54] = d54
				ps74.OverlayValues[56] = d56
				ps74.OverlayValues[57] = d57
				ps74.OverlayValues[64] = d64
				ps74.OverlayValues[65] = d65
				ps74.OverlayValues[71] = d71
				ps74.OverlayValues[72] = d72
				ps74.OverlayValues[73] = d73
						return bbs[4].RenderPS(ps74)
					}
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d75 = d1
				if d75.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d75)
				if d75.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d75.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d75.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d75.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d76 = d22
				if d76.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d76)
				ctx.EmitStoreToStack(d76, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ps77 := PhiState{General: ps.General}
				ps77.OverlayValues = make([]JITValueDesc, 77)
				ps77.OverlayValues[1] = d1
				ps77.OverlayValues[2] = d2
				ps77.OverlayValues[3] = d3
				ps77.OverlayValues[4] = d4
				ps77.OverlayValues[5] = d5
				ps77.OverlayValues[6] = d6
				ps77.OverlayValues[7] = d7
				ps77.OverlayValues[8] = d8
				ps77.OverlayValues[9] = d9
				ps77.OverlayValues[10] = d10
				ps77.OverlayValues[11] = d11
				ps77.OverlayValues[12] = d12
				ps77.OverlayValues[13] = d13
				ps77.OverlayValues[15] = d15
				ps77.OverlayValues[16] = d16
				ps77.OverlayValues[17] = d17
				ps77.OverlayValues[18] = d18
				ps77.OverlayValues[22] = d22
				ps77.OverlayValues[23] = d23
				ps77.OverlayValues[24] = d24
				ps77.OverlayValues[27] = d27
				ps77.OverlayValues[28] = d28
				ps77.OverlayValues[54] = d54
				ps77.OverlayValues[56] = d56
				ps77.OverlayValues[57] = d57
				ps77.OverlayValues[64] = d64
				ps77.OverlayValues[65] = d65
				ps77.OverlayValues[71] = d71
				ps77.OverlayValues[72] = d72
				ps77.OverlayValues[73] = d73
				ps77.OverlayValues[75] = d75
				ps77.OverlayValues[76] = d76
				ps77.PhiValues = make([]JITValueDesc, 2)
				d78 = d1
				ps77.PhiValues[0] = d78
				d79 = d22
				ps77.PhiValues[1] = d79
					return bbs[1].RenderPS(ps77)
				}
				if !ps.General {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
				lbl8 := ctx.ReserveLabel()
				lbl9 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d73.Reg, 0)
				ctx.EmitJcc(CcNE, lbl8)
				ctx.EmitJmp(lbl9)
				ctx.MarkLabel(lbl8)
				ctx.EmitJmp(lbl5)
				ctx.MarkLabel(lbl9)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d80 = d1
				if d80.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d80)
				if d80.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d80.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d80.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d80.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d81 = d22
				if d81.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d81)
				ctx.EmitStoreToStack(d81, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ctx.EmitJmp(lbl2)
				ps82 := PhiState{General: true}
				ps82.OverlayValues = make([]JITValueDesc, 82)
				ps82.OverlayValues[1] = d1
				ps82.OverlayValues[2] = d2
				ps82.OverlayValues[3] = d3
				ps82.OverlayValues[4] = d4
				ps82.OverlayValues[5] = d5
				ps82.OverlayValues[6] = d6
				ps82.OverlayValues[7] = d7
				ps82.OverlayValues[8] = d8
				ps82.OverlayValues[9] = d9
				ps82.OverlayValues[10] = d10
				ps82.OverlayValues[11] = d11
				ps82.OverlayValues[12] = d12
				ps82.OverlayValues[13] = d13
				ps82.OverlayValues[15] = d15
				ps82.OverlayValues[16] = d16
				ps82.OverlayValues[17] = d17
				ps82.OverlayValues[18] = d18
				ps82.OverlayValues[22] = d22
				ps82.OverlayValues[23] = d23
				ps82.OverlayValues[24] = d24
				ps82.OverlayValues[27] = d27
				ps82.OverlayValues[28] = d28
				ps82.OverlayValues[54] = d54
				ps82.OverlayValues[56] = d56
				ps82.OverlayValues[57] = d57
				ps82.OverlayValues[64] = d64
				ps82.OverlayValues[65] = d65
				ps82.OverlayValues[71] = d71
				ps82.OverlayValues[72] = d72
				ps82.OverlayValues[73] = d73
				ps82.OverlayValues[75] = d75
				ps82.OverlayValues[76] = d76
				ps82.OverlayValues[78] = d78
				ps82.OverlayValues[79] = d79
				ps82.OverlayValues[80] = d80
				ps82.OverlayValues[81] = d81
				ps83 := PhiState{General: true}
				ps83.OverlayValues = make([]JITValueDesc, 82)
				ps83.OverlayValues[1] = d1
				ps83.OverlayValues[2] = d2
				ps83.OverlayValues[3] = d3
				ps83.OverlayValues[4] = d4
				ps83.OverlayValues[5] = d5
				ps83.OverlayValues[6] = d6
				ps83.OverlayValues[7] = d7
				ps83.OverlayValues[8] = d8
				ps83.OverlayValues[9] = d9
				ps83.OverlayValues[10] = d10
				ps83.OverlayValues[11] = d11
				ps83.OverlayValues[12] = d12
				ps83.OverlayValues[13] = d13
				ps83.OverlayValues[15] = d15
				ps83.OverlayValues[16] = d16
				ps83.OverlayValues[17] = d17
				ps83.OverlayValues[18] = d18
				ps83.OverlayValues[22] = d22
				ps83.OverlayValues[23] = d23
				ps83.OverlayValues[24] = d24
				ps83.OverlayValues[27] = d27
				ps83.OverlayValues[28] = d28
				ps83.OverlayValues[54] = d54
				ps83.OverlayValues[56] = d56
				ps83.OverlayValues[57] = d57
				ps83.OverlayValues[64] = d64
				ps83.OverlayValues[65] = d65
				ps83.OverlayValues[71] = d71
				ps83.OverlayValues[72] = d72
				ps83.OverlayValues[73] = d73
				ps83.OverlayValues[75] = d75
				ps83.OverlayValues[76] = d76
				ps83.OverlayValues[78] = d78
				ps83.OverlayValues[79] = d79
				ps83.OverlayValues[80] = d80
				ps83.OverlayValues[81] = d81
				ps83.PhiValues = make([]JITValueDesc, 2)
				d84 = d1
				ps83.PhiValues[0] = d84
				d85 = d22
				ps83.PhiValues[1] = d85
				snap86 := d1
				snap87 := d2
				snap88 := d3
				snap89 := d4
				snap90 := d5
				snap91 := d6
				snap92 := d7
				snap93 := d8
				snap94 := d9
				snap95 := d10
				snap96 := d11
				snap97 := d12
				snap98 := d13
				snap99 := d15
				snap100 := d16
				snap101 := d17
				snap102 := d18
				snap103 := d22
				snap104 := d23
				snap105 := d24
				snap106 := d27
				snap107 := d28
				snap108 := d54
				snap109 := d56
				snap110 := d57
				snap111 := d64
				snap112 := d65
				snap113 := d71
				snap114 := d72
				snap115 := d73
				snap116 := d75
				snap117 := d76
				snap118 := d78
				snap119 := d79
				snap120 := d80
				snap121 := d81
				snap122 := d84
				snap123 := d85
				alloc124 := ctx.SnapshotAllocState()
				if !bbs[1].Rendered {
					bbs[1].RenderPS(ps83)
				}
				ctx.RestoreAllocState(alloc124)
				d1 = snap86
				d2 = snap87
				d3 = snap88
				d4 = snap89
				d5 = snap90
				d6 = snap91
				d7 = snap92
				d8 = snap93
				d9 = snap94
				d10 = snap95
				d11 = snap96
				d12 = snap97
				d13 = snap98
				d15 = snap99
				d16 = snap100
				d17 = snap101
				d18 = snap102
				d22 = snap103
				d23 = snap104
				d24 = snap105
				d27 = snap106
				d28 = snap107
				d54 = snap108
				d56 = snap109
				d57 = snap110
				d64 = snap111
				d65 = snap112
				d71 = snap113
				d72 = snap114
				d73 = snap115
				d75 = snap116
				d76 = snap117
				d78 = snap118
				d79 = snap119
				d80 = snap120
				d81 = snap121
				d84 = snap122
				d85 = snap123
				if !bbs[4].Rendered {
					return bbs[4].RenderPS(ps82)
				}
				return result
				ctx.FreeDesc(&d71)
				return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[3].VisitCount >= 0 {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
				}
				bbs[3].VisitCount++
				if ps.General {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_3 = bbs[3].Address
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
					d54 = ps.OverlayValues[54]
				}
				if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
					d56 = ps.OverlayValues[56]
				}
				if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
					d57 = ps.OverlayValues[57]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
					d65 = ps.OverlayValues[65]
				}
				if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
					d71 = ps.OverlayValues[71]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
					d75 = ps.OverlayValues[75]
				}
				if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
					d76 = ps.OverlayValues[76]
				}
				if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
					d78 = ps.OverlayValues[78]
				}
				if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
					d79 = ps.OverlayValues[79]
				}
				if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
					d80 = ps.OverlayValues[80]
				}
				if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
					d81 = ps.OverlayValues[81]
				}
				if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
					d84 = ps.OverlayValues[84]
				}
				if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
					d85 = ps.OverlayValues[85]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs125 := make([]Reg, 0, 3)
				seenBlockPinnedRegs126 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs126
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs126[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs126[r] = true
						blockPinnedRegs125 = append(blockPinnedRegs125, r)
					}
				}
				unpinBlockRegs127 := func() { for _, r := range blockPinnedRegs125 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs127()
				d128 = ctx.EmitNewSliceFromGoSlice(&d1)
				ctx.EnsureDesc(&d128)
				if d128.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d128, &result)
					result.Type = d128.Type
				} else {
					switch d128.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d128)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d128)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d128)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						ctx.EmitMovPairToResult(&d128, &result)
						result.Type = d128.Type
					}
				}
				ctx.EmitJmp(lbl0)
				return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[4].VisitCount >= 0 {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
				}
				bbs[4].VisitCount++
				if ps.General {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_4 = bbs[4].Address
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
					d54 = ps.OverlayValues[54]
				}
				if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
					d56 = ps.OverlayValues[56]
				}
				if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
					d57 = ps.OverlayValues[57]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
					d65 = ps.OverlayValues[65]
				}
				if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
					d71 = ps.OverlayValues[71]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
					d75 = ps.OverlayValues[75]
				}
				if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
					d76 = ps.OverlayValues[76]
				}
				if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
					d78 = ps.OverlayValues[78]
				}
				if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
					d79 = ps.OverlayValues[79]
				}
				if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
					d80 = ps.OverlayValues[80]
				}
				if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
					d81 = ps.OverlayValues[81]
				}
				if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
					d84 = ps.OverlayValues[84]
				}
				if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
					d85 = ps.OverlayValues[85]
				}
				if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
					d128 = ps.OverlayValues[128]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs129 := make([]Reg, 0, 3)
				seenBlockPinnedRegs130 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs130
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs130[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs130[r] = true
						blockPinnedRegs129 = append(blockPinnedRegs129, r)
					}
				}
				unpinBlockRegs131 := func() { for _, r := range blockPinnedRegs129 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs131()
				stackArray132 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d57)
				ctx.EnsureDesc(&d57)
				ctx.EmitStoreScmerToStack(d57, int32(stackArray132)+int32(0))
				ctx.FreeDesc(&d57)
				r12 := ctx.AllocReg()
				r13 := ctx.AllocRegExcept(r12)
				r14 := ctx.AllocRegExcept(r12, r13)
				ctx.EmitLeaRegMem(r12, RegRSP, int32(stackArray132))
				ctx.EmitMovRegImm64(r13, uint64(1))
				ctx.EmitMovRegImm64(r14, uint64(1))
				d133 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r12, &d133)
				ctx.BindReg(r13, &d133)
				ctx.BindReg(r14, &d133)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple { panic("jit: append requires a Go slice header") }
				lbl10 := ctx.ReserveLabel()
				ctx.EmitCmpInt64(d1.Reg2, d1.Reg3)
				ctx.EmitJcc(CcB, lbl10)
				ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{{Loc: LocImm, Type: tagString, Imm: NewString("jit: generated append exceeded its fixed capacity")}})
				ctx.MarkLabel(lbl10)
				d134 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2, NoHeapPointer: true}
				ctx.BindReg(d1.Reg2, &d134)
				d135 = ctx.EmitSliceElementAddress(&d1, &d134, int32(16))
				d136 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray132)}
				ctx.EmitStoreScmerAt(&d135, &d136)
				ctx.FreeDesc(&d135)
				ctx.EmitAddRegImm32(d1.Reg2, 1)
				d137 = d1
				ctx.BindReg(d137.Reg, &d137)
				ctx.BindReg(d137.Reg2, &d137)
				ctx.BindReg(d137.Reg3, &d137)
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				ctx.EnsureDesc(&d137)
				if d137.Loc == LocReg {
					ctx.ProtectReg(d137.Reg)
				} else if d137.Loc == LocRegPair {
					ctx.ProtectReg(d137.Reg)
					ctx.ProtectReg(d137.Reg2)
				}
				d138 = d137
				if d138.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d138)
				if d138.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d138.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d138.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d138.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d139 = d22
				if d139.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d139)
				ctx.EmitStoreToStack(d139, int32(bbs[1].PhiBase)+int32(24))
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				if d137.Loc == LocReg {
					ctx.UnprotectReg(d137.Reg)
				} else if d137.Loc == LocRegPair {
					ctx.UnprotectReg(d137.Reg)
					ctx.UnprotectReg(d137.Reg2)
				}
				ps140 := PhiState{General: ps.General}
				ps140.OverlayValues = make([]JITValueDesc, 140)
				ps140.OverlayValues[1] = d1
				ps140.OverlayValues[2] = d2
				ps140.OverlayValues[3] = d3
				ps140.OverlayValues[4] = d4
				ps140.OverlayValues[5] = d5
				ps140.OverlayValues[6] = d6
				ps140.OverlayValues[7] = d7
				ps140.OverlayValues[8] = d8
				ps140.OverlayValues[9] = d9
				ps140.OverlayValues[10] = d10
				ps140.OverlayValues[11] = d11
				ps140.OverlayValues[12] = d12
				ps140.OverlayValues[13] = d13
				ps140.OverlayValues[15] = d15
				ps140.OverlayValues[16] = d16
				ps140.OverlayValues[17] = d17
				ps140.OverlayValues[18] = d18
				ps140.OverlayValues[22] = d22
				ps140.OverlayValues[23] = d23
				ps140.OverlayValues[24] = d24
				ps140.OverlayValues[27] = d27
				ps140.OverlayValues[28] = d28
				ps140.OverlayValues[54] = d54
				ps140.OverlayValues[56] = d56
				ps140.OverlayValues[57] = d57
				ps140.OverlayValues[64] = d64
				ps140.OverlayValues[65] = d65
				ps140.OverlayValues[71] = d71
				ps140.OverlayValues[72] = d72
				ps140.OverlayValues[73] = d73
				ps140.OverlayValues[75] = d75
				ps140.OverlayValues[76] = d76
				ps140.OverlayValues[78] = d78
				ps140.OverlayValues[79] = d79
				ps140.OverlayValues[80] = d80
				ps140.OverlayValues[81] = d81
				ps140.OverlayValues[84] = d84
				ps140.OverlayValues[85] = d85
				ps140.OverlayValues[128] = d128
				ps140.OverlayValues[133] = d133
				ps140.OverlayValues[134] = d134
				ps140.OverlayValues[135] = d135
				ps140.OverlayValues[136] = d136
				ps140.OverlayValues[137] = d137
				ps140.OverlayValues[138] = d138
				ps140.OverlayValues[139] = d139
				ps140.PhiValues = make([]JITValueDesc, 2)
				d141 = d137
				ps140.PhiValues[0] = d141
				d142 = d22
				ps140.PhiValues[1] = d142
				if ps140.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps140)
				return result
				}
				argPinned143 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned143 = append(argPinned143, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned143 = append(argPinned143, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned143 = append(argPinned143, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned143 = append(argPinned143, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned143 {
						ctx.UnprotectReg(r)
					}
				}()
				ps144 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps144)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(40))
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map_filter_notnull",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "map_filter_notnull")
			mapper := OptimizeProcToSerialFunction(a[1])
			result := make([]Scmer, 0, len(input))
			for _, item := range input {
				mapped := mapper(item)
				if !mapped.IsNil() {
					result = append(result, mapped)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial map and non-nil filter (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sum_map",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "sum_map")
			mapper := OptimizeProcToSerialFunction(a[1])
			result := a[2]
			for _, item := range input {
				mapped := mapper(result, item)
				if result.IsNil() || mapped.IsNil() {
					result = NewNil()
				} else if result.IsInt() && mapped.IsInt() {
					result = NewInt(result.Int() + mapped.Int())
				} else {
					result = NewFloat(result.Float() + mapped.Float())
				}
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial map and numeric sum reduction (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any", Label: "sum"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "number"}},
				{Kind: "number", Label: "zero"},
			},
			Return:    &TypeDescriptor{Kind: "number"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_any",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "reduce_any")
			candidate := OptimizeProcToSerialFunction(a[1])
			state := NewBool(false)
			for _, item := range input {
				value := candidate(state, item)
				if value.IsNil() {
					state = NewNil()
				} else if value.Bool() {
					return NewBool(true)
				}
			}
			return state
		},
		Type: &TypeDescriptor{Kind: "func", Description: "short-circuiting three-valued OR reduction (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "candidate", Params: []*TypeDescriptor{{Kind: "any", Label: "state"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "bool"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_all",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "reduce_all")
			candidate := OptimizeProcToSerialFunction(a[1])
			state := NewBool(true)
			for _, item := range input {
				value := candidate(state, item)
				if value.IsNil() {
					state = NewNil()
				} else if !value.Bool() {
					return NewBool(false)
				}
			}
			return state
		},
		Type: &TypeDescriptor{Kind: "func", Description: "short-circuiting three-valued AND reduction (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "candidate", Params: []*TypeDescriptor{{Kind: "any", Label: "state"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "bool"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "find_map_notnull",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "find_map_notnull")
			candidate := OptimizeProcToSerialFunction(a[1])
			for _, item := range input {
				value := candidate(NewNil(), item)
				if !value.IsNil() {
					return value
				}
			}
			return NewNil()
		},
		Type: &TypeDescriptor{Kind: "func", Description: "maps until the first non-nil result (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "candidate", Params: []*TypeDescriptor{{Kind: "any", Label: "state"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map_filter",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "map_filter")
			predicate := OptimizeProcToSerialFunction(a[1])
			mapper := OptimizeProcToSerialFunction(a[2])
			result := make([]Scmer, 0, len(input))
			for _, item := range input {
				if predicate(item).Bool() {
					result = append(result, mapper(item))
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial filter and map (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "condition", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d54 JITValueDesc
				_ = d54
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d118 JITValueDesc
				_ = d118
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
				var d139 JITValueDesc
				_ = d139
				var d140 JITValueDesc
				_ = d140
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(40))
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				var bbs [5]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[0].VisitCount >= 0 {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
				}
				bbs[0].VisitCount++
				if ps.General {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_0 = bbs[0].Address
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				ctx.ReclaimUntrackedRegs()
				d3 = args[0]
				d3.ID = 0
				var d4 JITValueDesc
				if d3.Type == tagSlice {
					d4 = jitKnownSliceHeader(ctx, &d3)
				} else {
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d3}, 3)
				}
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.BindReg(d4.Reg3, &d4)
				ctx.FreeDesc(&d3)
				d5 = args[1]
				d5.ID = 0
				var d6 JITValueDesc
				if d5.Loc == LocLambdaTemplate {
					d6 = d5
				} else {
					d6 = ctx.RequestOptimizedCallback(1)
				}
				ctx.FreeDesc(&d5)
				d7 = args[2]
				d7.ID = 0
				var d8 JITValueDesc
				if d7.Loc == LocLambdaTemplate {
					d8 = d7
				} else {
					d8 = ctx.RequestOptimizedCallback(2)
				}
				ctx.FreeDesc(&d7)
				var d9 JITValueDesc
				if d4.SliceSizeKnown {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d9)
				d11 = ctx.EmitGoCallScalar(GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d10, d9}, 3)
				ctx.BindReg(d11.Reg, &d11)
				ctx.BindReg(d11.Reg2, &d11)
				ctx.BindReg(d11.Reg3, &d11)
				ctx.FreeDesc(&d9)
				var d12 JITValueDesc
				if d4.SliceSizeKnown {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.EnsureDesc(&d11)
				if d11.Loc == LocReg {
					ctx.ProtectReg(d11.Reg)
				} else if d11.Loc == LocRegPair {
					ctx.ProtectReg(d11.Reg)
					ctx.ProtectReg(d11.Reg2)
				}
				d13 = d11
				if d13.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d13)
				if d13.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d13.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d13.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d13.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(24))
				if d11.Loc == LocReg {
					ctx.UnprotectReg(d11.Reg)
				} else if d11.Loc == LocRegPair {
					ctx.UnprotectReg(d11.Reg)
					ctx.UnprotectReg(d11.Reg2)
				}
				ps14 := PhiState{General: ps.General}
				ps14.OverlayValues = make([]JITValueDesc, 14)
				ps14.OverlayValues[1] = d1
				ps14.OverlayValues[2] = d2
				ps14.OverlayValues[3] = d3
				ps14.OverlayValues[4] = d4
				ps14.OverlayValues[5] = d5
				ps14.OverlayValues[6] = d6
				ps14.OverlayValues[7] = d7
				ps14.OverlayValues[8] = d8
				ps14.OverlayValues[9] = d9
				ps14.OverlayValues[10] = d10
				ps14.OverlayValues[11] = d11
				ps14.OverlayValues[12] = d12
				ps14.OverlayValues[13] = d13
				ps14.PhiValues = make([]JITValueDesc, 2)
				d15 = d11
				ps14.PhiValues[0] = d15
				d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
				ps14.PhiValues[1] = d16
				if ps14.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps14)
				return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d17 := ps.PhiValues[0]
						ctx.EnsureDesc(&d17)
						ctx.EmitStoreRegMem(d17.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d17.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d17.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d18 := ps.PhiValues[1]
						ctx.EnsureDesc(&d18)
						ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(24))
					}
					if bbs[1].VisitCount >= 0 {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
				}
				bbs[1].VisitCount++
				if ps.General {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_1 = bbs[1].Address
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d1 = ps.PhiValues[0]
				}
				if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d2 = ps.PhiValues[1]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs19 := make([]Reg, 0, 3)
				seenBlockPinnedRegs20 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs20
				for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
					live := d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3)
					if live && !seenBlockPinnedRegs20[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs20[r] = true
						blockPinnedRegs19 = append(blockPinnedRegs19, r)
					}
				}
				unpinBlockRegs21 := func() { for _, r := range blockPinnedRegs19 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs21()
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				var d22 JITValueDesc
				if d2.Loc == LocImm {
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d2.Reg)
					ctx.EmitMovRegReg(scratch, d2.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d22)
				}
				if d22.Loc == LocReg && d2.Loc == LocReg && d22.Reg == d2.Reg {
					ctx.TransferReg(d2.Reg)
					d2.Loc = LocNone
				}
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				var d23 JITValueDesc
				if d22.Loc == LocImm && d12.Loc == LocImm {
					d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d22.Imm.Int() < d12.Imm.Int())}
				} else if d12.Loc == LocImm {
					r0 := ctx.AllocRegExcept(d22.Reg)
					if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d22.Reg, int32(d12.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
						ctx.EmitCmpInt64(d22.Reg, RegR11)
					}
					ctx.EmitSetcc(r0, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
					ctx.BindReg(r0, &d23)
				} else if d22.Loc == LocImm {
					r1 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d12.Reg)
					ctx.EmitSetcc(r1, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
					ctx.BindReg(r1, &d23)
				} else {
					r2 := ctx.AllocRegExcept(d22.Reg)
					ctx.EmitCmpInt64(d22.Reg, d12.Reg)
					ctx.EmitSetcc(r2, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
					ctx.BindReg(r2, &d23)
				}
				ctx.FreeDesc(&d12)
				d24 = d23
				ctx.EnsureDesc(&d24)
				if d24.Loc != LocImm && d24.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d24.Loc == LocImm {
					if d24.Imm.Bool() {
				ps25 := PhiState{General: ps.General}
				ps25.OverlayValues = make([]JITValueDesc, 25)
				ps25.OverlayValues[1] = d1
				ps25.OverlayValues[2] = d2
				ps25.OverlayValues[3] = d3
				ps25.OverlayValues[4] = d4
				ps25.OverlayValues[5] = d5
				ps25.OverlayValues[6] = d6
				ps25.OverlayValues[7] = d7
				ps25.OverlayValues[8] = d8
				ps25.OverlayValues[9] = d9
				ps25.OverlayValues[10] = d10
				ps25.OverlayValues[11] = d11
				ps25.OverlayValues[12] = d12
				ps25.OverlayValues[13] = d13
				ps25.OverlayValues[15] = d15
				ps25.OverlayValues[16] = d16
				ps25.OverlayValues[17] = d17
				ps25.OverlayValues[18] = d18
				ps25.OverlayValues[22] = d22
				ps25.OverlayValues[23] = d23
				ps25.OverlayValues[24] = d24
						return bbs[2].RenderPS(ps25)
					}
				ps26 := PhiState{General: ps.General}
				ps26.OverlayValues = make([]JITValueDesc, 25)
				ps26.OverlayValues[1] = d1
				ps26.OverlayValues[2] = d2
				ps26.OverlayValues[3] = d3
				ps26.OverlayValues[4] = d4
				ps26.OverlayValues[5] = d5
				ps26.OverlayValues[6] = d6
				ps26.OverlayValues[7] = d7
				ps26.OverlayValues[8] = d8
				ps26.OverlayValues[9] = d9
				ps26.OverlayValues[10] = d10
				ps26.OverlayValues[11] = d11
				ps26.OverlayValues[12] = d12
				ps26.OverlayValues[13] = d13
				ps26.OverlayValues[15] = d15
				ps26.OverlayValues[16] = d16
				ps26.OverlayValues[17] = d17
				ps26.OverlayValues[18] = d18
				ps26.OverlayValues[22] = d22
				ps26.OverlayValues[23] = d23
				ps26.OverlayValues[24] = d24
					return bbs[3].RenderPS(ps26)
				}
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d27 := ps.PhiValues[0]
						ctx.EnsureDesc(&d27)
						ctx.EmitStoreRegMem(d27.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d27.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d27.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d28 := ps.PhiValues[1]
						ctx.EnsureDesc(&d28)
						ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(24))
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
				lbl6 := ctx.ReserveLabel()
				lbl7 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d24.Reg, 0)
				ctx.EmitJcc(CcNE, lbl6)
				ctx.EmitJmp(lbl7)
				ctx.MarkLabel(lbl6)
				ctx.EmitJmp(lbl3)
				ctx.MarkLabel(lbl7)
				ctx.EmitJmp(lbl4)
				ps29 := PhiState{General: true}
				ps29.OverlayValues = make([]JITValueDesc, 29)
				ps29.OverlayValues[1] = d1
				ps29.OverlayValues[2] = d2
				ps29.OverlayValues[3] = d3
				ps29.OverlayValues[4] = d4
				ps29.OverlayValues[5] = d5
				ps29.OverlayValues[6] = d6
				ps29.OverlayValues[7] = d7
				ps29.OverlayValues[8] = d8
				ps29.OverlayValues[9] = d9
				ps29.OverlayValues[10] = d10
				ps29.OverlayValues[11] = d11
				ps29.OverlayValues[12] = d12
				ps29.OverlayValues[13] = d13
				ps29.OverlayValues[15] = d15
				ps29.OverlayValues[16] = d16
				ps29.OverlayValues[17] = d17
				ps29.OverlayValues[18] = d18
				ps29.OverlayValues[22] = d22
				ps29.OverlayValues[23] = d23
				ps29.OverlayValues[24] = d24
				ps29.OverlayValues[27] = d27
				ps29.OverlayValues[28] = d28
				ps30 := PhiState{General: true}
				ps30.OverlayValues = make([]JITValueDesc, 29)
				ps30.OverlayValues[1] = d1
				ps30.OverlayValues[2] = d2
				ps30.OverlayValues[3] = d3
				ps30.OverlayValues[4] = d4
				ps30.OverlayValues[5] = d5
				ps30.OverlayValues[6] = d6
				ps30.OverlayValues[7] = d7
				ps30.OverlayValues[8] = d8
				ps30.OverlayValues[9] = d9
				ps30.OverlayValues[10] = d10
				ps30.OverlayValues[11] = d11
				ps30.OverlayValues[12] = d12
				ps30.OverlayValues[13] = d13
				ps30.OverlayValues[15] = d15
				ps30.OverlayValues[16] = d16
				ps30.OverlayValues[17] = d17
				ps30.OverlayValues[18] = d18
				ps30.OverlayValues[22] = d22
				ps30.OverlayValues[23] = d23
				ps30.OverlayValues[24] = d24
				ps30.OverlayValues[27] = d27
				ps30.OverlayValues[28] = d28
				snap31 := d1
				snap32 := d2
				snap33 := d3
				snap34 := d4
				snap35 := d5
				snap36 := d6
				snap37 := d7
				snap38 := d8
				snap39 := d9
				snap40 := d10
				snap41 := d11
				snap42 := d12
				snap43 := d13
				snap44 := d15
				snap45 := d16
				snap46 := d17
				snap47 := d18
				snap48 := d22
				snap49 := d23
				snap50 := d24
				snap51 := d27
				snap52 := d28
				alloc53 := ctx.SnapshotAllocState()
				if !bbs[3].Rendered {
					bbs[3].RenderPS(ps30)
				}
				ctx.RestoreAllocState(alloc53)
				d1 = snap31
				d2 = snap32
				d3 = snap33
				d4 = snap34
				d5 = snap35
				d6 = snap36
				d7 = snap37
				d8 = snap38
				d9 = snap39
				d10 = snap40
				d11 = snap41
				d12 = snap42
				d13 = snap43
				d15 = snap44
				d16 = snap45
				d17 = snap46
				d18 = snap47
				d22 = snap48
				d23 = snap49
				d24 = snap50
				d27 = snap51
				d28 = snap52
				if !bbs[2].Rendered {
					return bbs[2].RenderPS(ps29)
				}
				return result
				ctx.FreeDesc(&d23)
				return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[2].VisitCount >= 0 {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
				}
				bbs[2].VisitCount++
				if ps.General {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_2 = bbs[2].Address
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d22)
				r3 := ctx.AllocReg()
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d4)
				if d22.Loc == LocImm {
					ctx.EmitMovRegImm64(r3, uint64(d22.Imm.Int()) * 16)
				} else {
					ctx.EmitMovRegReg(r3, d22.Reg)
					ctx.EmitShlRegImm8(r3, 4)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitAddInt64(r3, RegR11)
				} else {
					ctx.EmitAddInt64(r3, d4.Reg)
				}
				r4 := ctx.AllocRegExcept(r3)
				r5 := ctx.AllocRegExcept(r3, r4)
				ctx.EmitMovRegMem(r4, r3, 0)
				ctx.EmitMovRegMem(r5, r3, 8)
				ctx.FreeReg(r3)
				d54 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d54)
				ctx.BindReg(r5, &d54)
				stackArray55 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreScmerToStack(d54, int32(stackArray55)+int32(0))
				r6 := ctx.AllocReg()
				r7 := ctx.AllocRegExcept(r6)
				r8 := ctx.AllocRegExcept(r6, r7)
				ctx.EmitLeaRegMem(r6, RegRSP, int32(stackArray55))
				ctx.EmitMovRegImm64(r7, uint64(1))
				ctx.EmitMovRegImm64(r8, uint64(1))
				d56 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r6, &d56)
				ctx.BindReg(r7, &d56)
				ctx.BindReg(r8, &d56)
				callbackArgs58 := make([]JITValueDesc, 1)
				callbackArgs58[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray55)+0}
				var d57 JITValueDesc
				ctx.FreeDesc(&d56)
				if d6.Loc == LocLambdaTemplate && d6.Lambda != nil {
					callbackResultOff59 := ctx.AllocSpill(16)
					ctx.setStackPointer(jitStackRootFrameBP, callbackResultOff59, true)
					outerRegs60 := ctx.PreserveOuterRegs()
					d57 = JITEmitProcInlineWithOuter(ctx, &d6.Lambda.Proc, d6.Lambda.Outer, callbackArgs58, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					ctx.EnsureDesc(&d57)
					ctx.EmitStoreRegMem(d57.Reg, RegRBP, callbackResultOff59)
					ctx.EmitStoreRegMem(d57.Reg2, RegRBP, callbackResultOff59+8)
					ctx.RestoreOuterRegs(outerRegs60)
					d57 = JITValueDesc{Loc: LocStackPair, Type: d57.Type, StackOff: callbackResultOff59, NoHeapPointer: d57.NoHeapPointer}
					liveRegs61 := make([]Reg, 0, 24)
					seenLiveRegs62 := make(map[Reg]bool)
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := (d1.Loc == LocReg && r == d1.Reg) ||
							(d1.Loc == LocRegPair && (r == d1.Reg || r == d1.Reg2)) ||
							(d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := (d11.Loc == LocReg && r == d11.Reg) ||
							(d11.Loc == LocRegPair && (r == d11.Reg || r == d11.Reg2)) ||
							(d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d22.Reg, d22.Reg2, d22.Reg3} {
						live := (d22.Loc == LocReg && r == d22.Reg) ||
							(d22.Loc == LocRegPair && (r == d22.Reg || r == d22.Reg2)) ||
							(d22.Loc == LocRegTriple && (r == d22.Reg || r == d22.Reg2 || r == d22.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d4.Reg, d4.Reg2, d4.Reg3} {
						live := (d4.Loc == LocReg && r == d4.Reg) ||
							(d4.Loc == LocRegPair && (r == d4.Reg || r == d4.Reg2)) ||
							(d4.Loc == LocRegTriple && (r == d4.Reg || r == d4.Reg2 || r == d4.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d54.Reg, d54.Reg2, d54.Reg3} {
						live := (d54.Loc == LocReg && r == d54.Reg) ||
							(d54.Loc == LocRegPair && (r == d54.Reg || r == d54.Reg2)) ||
							(d54.Loc == LocRegTriple && (r == d54.Reg || r == d54.Reg2 || r == d54.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d56.Reg, d56.Reg2, d56.Reg3} {
						live := (d56.Loc == LocReg && r == d56.Reg) ||
							(d56.Loc == LocRegPair && (r == d56.Reg || r == d56.Reg2)) ||
							(d56.Loc == LocRegTriple && (r == d56.Reg || r == d56.Reg2 || r == d56.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d6.Reg, d6.Reg2, d6.Reg3} {
						live := (d6.Loc == LocReg && r == d6.Reg) ||
							(d6.Loc == LocRegPair && (r == d6.Reg || r == d6.Reg2)) ||
							(d6.Loc == LocRegTriple && (r == d6.Reg || r == d6.Reg2 || r == d6.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d8.Reg, d8.Reg2, d8.Reg3} {
						live := (d8.Loc == LocReg && r == d8.Reg) ||
							(d8.Loc == LocRegPair && (r == d8.Reg || r == d8.Reg2)) ||
							(d8.Loc == LocRegTriple && (r == d8.Reg || r == d8.Reg2 || r == d8.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					ctx.EnsureDesc(&d57)
					for _, r := range liveRegs61 { ctx.UnprotectReg(r) }
				} else {
					callbackCallArgs := make([]JITValueDesc, 0, 2)
					callbackCallArgs = append(callbackCallArgs, d6)
					callbackCallArgs = append(callbackCallArgs, callbackArgs58...)
					d57 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
				}
				d64 = d57
				d64.ID = 0
				d63 = ctx.EmitBoolDesc(&d64, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d57)
				d65 = d63
				ctx.EnsureDesc(&d65)
				if d65.Loc != LocImm && d65.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d65.Loc == LocImm {
					if d65.Imm.Bool() {
				ps66 := PhiState{General: ps.General}
				ps66.OverlayValues = make([]JITValueDesc, 66)
				ps66.OverlayValues[1] = d1
				ps66.OverlayValues[2] = d2
				ps66.OverlayValues[3] = d3
				ps66.OverlayValues[4] = d4
				ps66.OverlayValues[5] = d5
				ps66.OverlayValues[6] = d6
				ps66.OverlayValues[7] = d7
				ps66.OverlayValues[8] = d8
				ps66.OverlayValues[9] = d9
				ps66.OverlayValues[10] = d10
				ps66.OverlayValues[11] = d11
				ps66.OverlayValues[12] = d12
				ps66.OverlayValues[13] = d13
				ps66.OverlayValues[15] = d15
				ps66.OverlayValues[16] = d16
				ps66.OverlayValues[17] = d17
				ps66.OverlayValues[18] = d18
				ps66.OverlayValues[22] = d22
				ps66.OverlayValues[23] = d23
				ps66.OverlayValues[24] = d24
				ps66.OverlayValues[27] = d27
				ps66.OverlayValues[28] = d28
				ps66.OverlayValues[54] = d54
				ps66.OverlayValues[56] = d56
				ps66.OverlayValues[57] = d57
				ps66.OverlayValues[63] = d63
				ps66.OverlayValues[64] = d64
				ps66.OverlayValues[65] = d65
						return bbs[4].RenderPS(ps66)
					}
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d67 = d1
				if d67.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d67)
				if d67.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d67.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d67.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d67.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d68 = d22
				if d68.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d68)
				ctx.EmitStoreToStack(d68, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ps69 := PhiState{General: ps.General}
				ps69.OverlayValues = make([]JITValueDesc, 69)
				ps69.OverlayValues[1] = d1
				ps69.OverlayValues[2] = d2
				ps69.OverlayValues[3] = d3
				ps69.OverlayValues[4] = d4
				ps69.OverlayValues[5] = d5
				ps69.OverlayValues[6] = d6
				ps69.OverlayValues[7] = d7
				ps69.OverlayValues[8] = d8
				ps69.OverlayValues[9] = d9
				ps69.OverlayValues[10] = d10
				ps69.OverlayValues[11] = d11
				ps69.OverlayValues[12] = d12
				ps69.OverlayValues[13] = d13
				ps69.OverlayValues[15] = d15
				ps69.OverlayValues[16] = d16
				ps69.OverlayValues[17] = d17
				ps69.OverlayValues[18] = d18
				ps69.OverlayValues[22] = d22
				ps69.OverlayValues[23] = d23
				ps69.OverlayValues[24] = d24
				ps69.OverlayValues[27] = d27
				ps69.OverlayValues[28] = d28
				ps69.OverlayValues[54] = d54
				ps69.OverlayValues[56] = d56
				ps69.OverlayValues[57] = d57
				ps69.OverlayValues[63] = d63
				ps69.OverlayValues[64] = d64
				ps69.OverlayValues[65] = d65
				ps69.OverlayValues[67] = d67
				ps69.OverlayValues[68] = d68
				ps69.PhiValues = make([]JITValueDesc, 2)
				d70 = d1
				ps69.PhiValues[0] = d70
				d71 = d22
				ps69.PhiValues[1] = d71
					return bbs[1].RenderPS(ps69)
				}
				if !ps.General {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
				lbl8 := ctx.ReserveLabel()
				lbl9 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d65.Reg, 0)
				ctx.EmitJcc(CcNE, lbl8)
				ctx.EmitJmp(lbl9)
				ctx.MarkLabel(lbl8)
				ctx.EmitJmp(lbl5)
				ctx.MarkLabel(lbl9)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d72 = d1
				if d72.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d72)
				if d72.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d72.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d72.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d72.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d73 = d22
				if d73.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d73)
				ctx.EmitStoreToStack(d73, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ctx.EmitJmp(lbl2)
				ps74 := PhiState{General: true}
				ps74.OverlayValues = make([]JITValueDesc, 74)
				ps74.OverlayValues[1] = d1
				ps74.OverlayValues[2] = d2
				ps74.OverlayValues[3] = d3
				ps74.OverlayValues[4] = d4
				ps74.OverlayValues[5] = d5
				ps74.OverlayValues[6] = d6
				ps74.OverlayValues[7] = d7
				ps74.OverlayValues[8] = d8
				ps74.OverlayValues[9] = d9
				ps74.OverlayValues[10] = d10
				ps74.OverlayValues[11] = d11
				ps74.OverlayValues[12] = d12
				ps74.OverlayValues[13] = d13
				ps74.OverlayValues[15] = d15
				ps74.OverlayValues[16] = d16
				ps74.OverlayValues[17] = d17
				ps74.OverlayValues[18] = d18
				ps74.OverlayValues[22] = d22
				ps74.OverlayValues[23] = d23
				ps74.OverlayValues[24] = d24
				ps74.OverlayValues[27] = d27
				ps74.OverlayValues[28] = d28
				ps74.OverlayValues[54] = d54
				ps74.OverlayValues[56] = d56
				ps74.OverlayValues[57] = d57
				ps74.OverlayValues[63] = d63
				ps74.OverlayValues[64] = d64
				ps74.OverlayValues[65] = d65
				ps74.OverlayValues[67] = d67
				ps74.OverlayValues[68] = d68
				ps74.OverlayValues[70] = d70
				ps74.OverlayValues[71] = d71
				ps74.OverlayValues[72] = d72
				ps74.OverlayValues[73] = d73
				ps75 := PhiState{General: true}
				ps75.OverlayValues = make([]JITValueDesc, 74)
				ps75.OverlayValues[1] = d1
				ps75.OverlayValues[2] = d2
				ps75.OverlayValues[3] = d3
				ps75.OverlayValues[4] = d4
				ps75.OverlayValues[5] = d5
				ps75.OverlayValues[6] = d6
				ps75.OverlayValues[7] = d7
				ps75.OverlayValues[8] = d8
				ps75.OverlayValues[9] = d9
				ps75.OverlayValues[10] = d10
				ps75.OverlayValues[11] = d11
				ps75.OverlayValues[12] = d12
				ps75.OverlayValues[13] = d13
				ps75.OverlayValues[15] = d15
				ps75.OverlayValues[16] = d16
				ps75.OverlayValues[17] = d17
				ps75.OverlayValues[18] = d18
				ps75.OverlayValues[22] = d22
				ps75.OverlayValues[23] = d23
				ps75.OverlayValues[24] = d24
				ps75.OverlayValues[27] = d27
				ps75.OverlayValues[28] = d28
				ps75.OverlayValues[54] = d54
				ps75.OverlayValues[56] = d56
				ps75.OverlayValues[57] = d57
				ps75.OverlayValues[63] = d63
				ps75.OverlayValues[64] = d64
				ps75.OverlayValues[65] = d65
				ps75.OverlayValues[67] = d67
				ps75.OverlayValues[68] = d68
				ps75.OverlayValues[70] = d70
				ps75.OverlayValues[71] = d71
				ps75.OverlayValues[72] = d72
				ps75.OverlayValues[73] = d73
				ps75.PhiValues = make([]JITValueDesc, 2)
				d76 = d1
				ps75.PhiValues[0] = d76
				d77 = d22
				ps75.PhiValues[1] = d77
				snap78 := d1
				snap79 := d2
				snap80 := d3
				snap81 := d4
				snap82 := d5
				snap83 := d6
				snap84 := d7
				snap85 := d8
				snap86 := d9
				snap87 := d10
				snap88 := d11
				snap89 := d12
				snap90 := d13
				snap91 := d15
				snap92 := d16
				snap93 := d17
				snap94 := d18
				snap95 := d22
				snap96 := d23
				snap97 := d24
				snap98 := d27
				snap99 := d28
				snap100 := d54
				snap101 := d56
				snap102 := d57
				snap103 := d63
				snap104 := d64
				snap105 := d65
				snap106 := d67
				snap107 := d68
				snap108 := d70
				snap109 := d71
				snap110 := d72
				snap111 := d73
				snap112 := d76
				snap113 := d77
				alloc114 := ctx.SnapshotAllocState()
				if !bbs[1].Rendered {
					bbs[1].RenderPS(ps75)
				}
				ctx.RestoreAllocState(alloc114)
				d1 = snap78
				d2 = snap79
				d3 = snap80
				d4 = snap81
				d5 = snap82
				d6 = snap83
				d7 = snap84
				d8 = snap85
				d9 = snap86
				d10 = snap87
				d11 = snap88
				d12 = snap89
				d13 = snap90
				d15 = snap91
				d16 = snap92
				d17 = snap93
				d18 = snap94
				d22 = snap95
				d23 = snap96
				d24 = snap97
				d27 = snap98
				d28 = snap99
				d54 = snap100
				d56 = snap101
				d57 = snap102
				d63 = snap103
				d64 = snap104
				d65 = snap105
				d67 = snap106
				d68 = snap107
				d70 = snap108
				d71 = snap109
				d72 = snap110
				d73 = snap111
				d76 = snap112
				d77 = snap113
				if !bbs[4].Rendered {
					return bbs[4].RenderPS(ps74)
				}
				return result
				ctx.FreeDesc(&d63)
				return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[3].VisitCount >= 0 {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
				}
				bbs[3].VisitCount++
				if ps.General {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_3 = bbs[3].Address
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
					d54 = ps.OverlayValues[54]
				}
				if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
					d56 = ps.OverlayValues[56]
				}
				if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
					d57 = ps.OverlayValues[57]
				}
				if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
					d63 = ps.OverlayValues[63]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
					d65 = ps.OverlayValues[65]
				}
				if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
					d67 = ps.OverlayValues[67]
				}
				if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
					d68 = ps.OverlayValues[68]
				}
				if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
					d70 = ps.OverlayValues[70]
				}
				if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
					d71 = ps.OverlayValues[71]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
					d76 = ps.OverlayValues[76]
				}
				if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
					d77 = ps.OverlayValues[77]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs115 := make([]Reg, 0, 3)
				seenBlockPinnedRegs116 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs116
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs116[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs116[r] = true
						blockPinnedRegs115 = append(blockPinnedRegs115, r)
					}
				}
				unpinBlockRegs117 := func() { for _, r := range blockPinnedRegs115 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs117()
				d118 = ctx.EmitNewSliceFromGoSlice(&d1)
				ctx.EnsureDesc(&d118)
				if d118.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d118, &result)
					result.Type = d118.Type
				} else {
					switch d118.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d118)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d118)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d118)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						ctx.EmitMovPairToResult(&d118, &result)
						result.Type = d118.Type
					}
				}
				ctx.EmitJmp(lbl0)
				return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[4].VisitCount >= 0 {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
				}
				bbs[4].VisitCount++
				if ps.General {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_4 = bbs[4].Address
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
					d54 = ps.OverlayValues[54]
				}
				if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
					d56 = ps.OverlayValues[56]
				}
				if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
					d57 = ps.OverlayValues[57]
				}
				if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
					d63 = ps.OverlayValues[63]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
					d65 = ps.OverlayValues[65]
				}
				if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
					d67 = ps.OverlayValues[67]
				}
				if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
					d68 = ps.OverlayValues[68]
				}
				if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
					d70 = ps.OverlayValues[70]
				}
				if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
					d71 = ps.OverlayValues[71]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
					d76 = ps.OverlayValues[76]
				}
				if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
					d77 = ps.OverlayValues[77]
				}
				if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
					d118 = ps.OverlayValues[118]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs119 := make([]Reg, 0, 3)
				seenBlockPinnedRegs120 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs120
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs120[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs120[r] = true
						blockPinnedRegs119 = append(blockPinnedRegs119, r)
					}
				}
				unpinBlockRegs121 := func() { for _, r := range blockPinnedRegs119 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs121()
				stackArray122 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreScmerToStack(d54, int32(stackArray122)+int32(0))
				ctx.FreeDesc(&d54)
				r9 := ctx.AllocReg()
				r10 := ctx.AllocRegExcept(r9)
				r11 := ctx.AllocRegExcept(r9, r10)
				ctx.EmitLeaRegMem(r9, RegRSP, int32(stackArray122))
				ctx.EmitMovRegImm64(r10, uint64(1))
				ctx.EmitMovRegImm64(r11, uint64(1))
				d123 = JITValueDesc{Loc: LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r9, &d123)
				ctx.BindReg(r10, &d123)
				ctx.BindReg(r11, &d123)
				callbackArgs125 := make([]JITValueDesc, 1)
				callbackArgs125[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray122)+0}
				var d124 JITValueDesc
				ctx.FreeDesc(&d123)
				if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
					callbackResultOff126 := ctx.AllocSpill(16)
					ctx.setStackPointer(jitStackRootFrameBP, callbackResultOff126, true)
					outerRegs127 := ctx.PreserveOuterRegs()
					d124 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, callbackArgs125, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					ctx.EnsureDesc(&d124)
					ctx.EmitStoreRegMem(d124.Reg, RegRBP, callbackResultOff126)
					ctx.EmitStoreRegMem(d124.Reg2, RegRBP, callbackResultOff126+8)
					ctx.RestoreOuterRegs(outerRegs127)
					d124 = JITValueDesc{Loc: LocStackPair, Type: d124.Type, StackOff: callbackResultOff126, NoHeapPointer: d124.NoHeapPointer}
					liveRegs128 := make([]Reg, 0, 21)
					seenLiveRegs129 := make(map[Reg]bool)
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := (d1.Loc == LocReg && r == d1.Reg) ||
							(d1.Loc == LocRegPair && (r == d1.Reg || r == d1.Reg2)) ||
							(d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3))
						if live && !seenLiveRegs129[r] {
							ctx.ProtectReg(r)
							seenLiveRegs129[r] = true
							liveRegs128 = append(liveRegs128, r)
						}
					}
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := (d11.Loc == LocReg && r == d11.Reg) ||
							(d11.Loc == LocRegPair && (r == d11.Reg || r == d11.Reg2)) ||
							(d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3))
						if live && !seenLiveRegs129[r] {
							ctx.ProtectReg(r)
							seenLiveRegs129[r] = true
							liveRegs128 = append(liveRegs128, r)
						}
					}
					for _, r := range []Reg{d118.Reg, d118.Reg2, d118.Reg3} {
						live := (d118.Loc == LocReg && r == d118.Reg) ||
							(d118.Loc == LocRegPair && (r == d118.Reg || r == d118.Reg2)) ||
							(d118.Loc == LocRegTriple && (r == d118.Reg || r == d118.Reg2 || r == d118.Reg3))
						if live && !seenLiveRegs129[r] {
							ctx.ProtectReg(r)
							seenLiveRegs129[r] = true
							liveRegs128 = append(liveRegs128, r)
						}
					}
					for _, r := range []Reg{d123.Reg, d123.Reg2, d123.Reg3} {
						live := (d123.Loc == LocReg && r == d123.Reg) ||
							(d123.Loc == LocRegPair && (r == d123.Reg || r == d123.Reg2)) ||
							(d123.Loc == LocRegTriple && (r == d123.Reg || r == d123.Reg2 || r == d123.Reg3))
						if live && !seenLiveRegs129[r] {
							ctx.ProtectReg(r)
							seenLiveRegs129[r] = true
							liveRegs128 = append(liveRegs128, r)
						}
					}
					for _, r := range []Reg{d22.Reg, d22.Reg2, d22.Reg3} {
						live := (d22.Loc == LocReg && r == d22.Reg) ||
							(d22.Loc == LocRegPair && (r == d22.Reg || r == d22.Reg2)) ||
							(d22.Loc == LocRegTriple && (r == d22.Reg || r == d22.Reg2 || r == d22.Reg3))
						if live && !seenLiveRegs129[r] {
							ctx.ProtectReg(r)
							seenLiveRegs129[r] = true
							liveRegs128 = append(liveRegs128, r)
						}
					}
					for _, r := range []Reg{d4.Reg, d4.Reg2, d4.Reg3} {
						live := (d4.Loc == LocReg && r == d4.Reg) ||
							(d4.Loc == LocRegPair && (r == d4.Reg || r == d4.Reg2)) ||
							(d4.Loc == LocRegTriple && (r == d4.Reg || r == d4.Reg2 || r == d4.Reg3))
						if live && !seenLiveRegs129[r] {
							ctx.ProtectReg(r)
							seenLiveRegs129[r] = true
							liveRegs128 = append(liveRegs128, r)
						}
					}
					for _, r := range []Reg{d8.Reg, d8.Reg2, d8.Reg3} {
						live := (d8.Loc == LocReg && r == d8.Reg) ||
							(d8.Loc == LocRegPair && (r == d8.Reg || r == d8.Reg2)) ||
							(d8.Loc == LocRegTriple && (r == d8.Reg || r == d8.Reg2 || r == d8.Reg3))
						if live && !seenLiveRegs129[r] {
							ctx.ProtectReg(r)
							seenLiveRegs129[r] = true
							liveRegs128 = append(liveRegs128, r)
						}
					}
					ctx.EnsureDesc(&d124)
					for _, r := range liveRegs128 { ctx.UnprotectReg(r) }
				} else {
					callbackCallArgs := make([]JITValueDesc, 0, 2)
					callbackCallArgs = append(callbackCallArgs, d8)
					callbackCallArgs = append(callbackCallArgs, callbackArgs125...)
					d124 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
				}
				stackArray130 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d124)
				ctx.EnsureDesc(&d124)
				ctx.EmitStoreScmerToStack(d124, int32(stackArray130)+int32(0))
				ctx.FreeDesc(&d124)
				r12 := ctx.AllocReg()
				r13 := ctx.AllocRegExcept(r12)
				r14 := ctx.AllocRegExcept(r12, r13)
				ctx.EmitLeaRegMem(r12, RegRSP, int32(stackArray130))
				ctx.EmitMovRegImm64(r13, uint64(1))
				ctx.EmitMovRegImm64(r14, uint64(1))
				d131 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r12, &d131)
				ctx.BindReg(r13, &d131)
				ctx.BindReg(r14, &d131)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple { panic("jit: append requires a Go slice header") }
				lbl10 := ctx.ReserveLabel()
				ctx.EmitCmpInt64(d1.Reg2, d1.Reg3)
				ctx.EmitJcc(CcB, lbl10)
				ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{{Loc: LocImm, Type: tagString, Imm: NewString("jit: generated append exceeded its fixed capacity")}})
				ctx.MarkLabel(lbl10)
				d132 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2, NoHeapPointer: true}
				ctx.BindReg(d1.Reg2, &d132)
				d133 = ctx.EmitSliceElementAddress(&d1, &d132, int32(16))
				d134 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray130)}
				ctx.EmitStoreScmerAt(&d133, &d134)
				ctx.FreeDesc(&d133)
				ctx.EmitAddRegImm32(d1.Reg2, 1)
				d135 = d1
				ctx.BindReg(d135.Reg, &d135)
				ctx.BindReg(d135.Reg2, &d135)
				ctx.BindReg(d135.Reg3, &d135)
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				ctx.EnsureDesc(&d135)
				if d135.Loc == LocReg {
					ctx.ProtectReg(d135.Reg)
				} else if d135.Loc == LocRegPair {
					ctx.ProtectReg(d135.Reg)
					ctx.ProtectReg(d135.Reg2)
				}
				d136 = d135
				if d136.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d136)
				if d136.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d136.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d136.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d136.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d137 = d22
				if d137.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d137)
				ctx.EmitStoreToStack(d137, int32(bbs[1].PhiBase)+int32(24))
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				if d135.Loc == LocReg {
					ctx.UnprotectReg(d135.Reg)
				} else if d135.Loc == LocRegPair {
					ctx.UnprotectReg(d135.Reg)
					ctx.UnprotectReg(d135.Reg2)
				}
				ps138 := PhiState{General: ps.General}
				ps138.OverlayValues = make([]JITValueDesc, 138)
				ps138.OverlayValues[1] = d1
				ps138.OverlayValues[2] = d2
				ps138.OverlayValues[3] = d3
				ps138.OverlayValues[4] = d4
				ps138.OverlayValues[5] = d5
				ps138.OverlayValues[6] = d6
				ps138.OverlayValues[7] = d7
				ps138.OverlayValues[8] = d8
				ps138.OverlayValues[9] = d9
				ps138.OverlayValues[10] = d10
				ps138.OverlayValues[11] = d11
				ps138.OverlayValues[12] = d12
				ps138.OverlayValues[13] = d13
				ps138.OverlayValues[15] = d15
				ps138.OverlayValues[16] = d16
				ps138.OverlayValues[17] = d17
				ps138.OverlayValues[18] = d18
				ps138.OverlayValues[22] = d22
				ps138.OverlayValues[23] = d23
				ps138.OverlayValues[24] = d24
				ps138.OverlayValues[27] = d27
				ps138.OverlayValues[28] = d28
				ps138.OverlayValues[54] = d54
				ps138.OverlayValues[56] = d56
				ps138.OverlayValues[57] = d57
				ps138.OverlayValues[63] = d63
				ps138.OverlayValues[64] = d64
				ps138.OverlayValues[65] = d65
				ps138.OverlayValues[67] = d67
				ps138.OverlayValues[68] = d68
				ps138.OverlayValues[70] = d70
				ps138.OverlayValues[71] = d71
				ps138.OverlayValues[72] = d72
				ps138.OverlayValues[73] = d73
				ps138.OverlayValues[76] = d76
				ps138.OverlayValues[77] = d77
				ps138.OverlayValues[118] = d118
				ps138.OverlayValues[123] = d123
				ps138.OverlayValues[124] = d124
				ps138.OverlayValues[131] = d131
				ps138.OverlayValues[132] = d132
				ps138.OverlayValues[133] = d133
				ps138.OverlayValues[134] = d134
				ps138.OverlayValues[135] = d135
				ps138.OverlayValues[136] = d136
				ps138.OverlayValues[137] = d137
				ps138.PhiValues = make([]JITValueDesc, 2)
				d139 = d135
				ps138.PhiValues[0] = d139
				d140 = d22
				ps138.PhiValues[1] = d140
				if ps138.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps138)
				return result
				}
				argPinned141 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned141 = append(argPinned141, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned141 = append(argPinned141, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned141 = append(argPinned141, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned141 = append(argPinned141, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned141 {
						ctx.UnprotectReg(r)
					}
				}()
				ps142 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps142)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(40))
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map_map",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "map_map")
			first := OptimizeProcToSerialFunction(a[1])
			second := OptimizeProcToSerialFunction(a[2])
			result := make([]Scmer, len(input))
			for i, item := range input {
				result[i] = second(first(item))
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial map and map (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_map",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "reduce_map")
			mapper := OptimizeProcToSerialFunction(a[1])
			reducer := OptimizeProcToSerialFunction(a[2])
			result := NewNil()
			hasResult := false
			if len(a) > 3 {
				result = a[3]
				hasResult = true
			}
			for _, item := range input {
				mapped := mapper(item)
				if !hasResult {
					result = mapped
					hasResult = true
					continue
				}
				result = reducer(result, mapped)
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial map and reduce (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be reduced", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Params: []*TypeDescriptor{{Kind: "any", Transfer: true, Label: "acc", Description: "current accumulator"}, {Kind: "any", Label: "item", Description: "mapped item"}}, Label: "reduce", Return: &TypeDescriptor{Kind: "any", Label: "acc", Description: "next accumulator"}, Description: "combines the accumulator with each mapped item"},
				{Kind: "any", Label: "neutral", Description: "(optional) initial value of the accumulator, defaults to nil", Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_filter",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "reduce_filter")
			predicate := OptimizeProcToSerialFunction(a[1])
			reducer := OptimizeProcToSerialFunction(a[2])
			result := NewNil()
			hasResult := false
			if len(a) > 3 {
				result = a[3]
				hasResult = true
			}
			for _, item := range input {
				if !predicate(item).Bool() {
					continue
				}
				if !hasResult {
					result = item
					hasResult = true
					continue
				}
				result = reducer(result, item)
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial filter and reduce (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be reduced", NoEscape: true},
				{Kind: "func", Label: "filter", Params: []*TypeDescriptor{{Label: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", Params: []*TypeDescriptor{{Kind: "any", Transfer: true, Label: "acc", Description: "current accumulator"}, {Kind: "any", Label: "item", Description: "current list item"}}, Label: "reduce", Return: &TypeDescriptor{Kind: "any", Label: "acc", Description: "next accumulator"}, Description: "combines the accumulator with each list item"},
				{Kind: "any", Label: "neutral", Description: "(optional) initial value of the accumulator, defaults to nil", Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_merge2",

		Fn: func(a ...Scmer) Scmer {
			if len(a) < 3 {
				panic("reduce_merge2 expects at least two input lists and a reduce function")
			}
			lhs := asSlice(a[0], "reduce_merge2 lhs")
			rhs := asSlice(a[1], "reduce_merge2 rhs")
			reducer := OptimizeProcToSerialFunction(a[2])
			result := NewNil()
			hasResult := false
			if len(a) > 3 {
				result = a[3]
				hasResult = true
			}
			for _, item := range lhs {
				if !hasResult {
					result = item
					hasResult = true
					continue
				}
				result = reducer(result, item)
			}
			for _, item := range rhs {
				if !hasResult {
					result = item
					hasResult = true
					continue
				}
				result = reducer(result, item)
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial merge of two lists and reduce (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "lhs", Description: "first list to reduce"},
				{Kind: "list", Label: "rhs", Description: "second list to reduce"},
				{Kind: "func", Label: "reduce", Params: []*TypeDescriptor{{Kind: "any", Transfer: true, Label: "acc", Description: "current accumulator"}, {Kind: "any", Label: "item", Description: "mapped item"}}, Return: &TypeDescriptor{Kind: "any", Label: "acc", Description: "next accumulator"}, Description: "combines the accumulator with each mapped item"},
				{Kind: "any", Label: "neutral", Description: "optional initial value of the accumulator, defaults to nil", Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_map_filter",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "reduce_map_filter")
			mapper := OptimizeProcToSerialFunction(a[1])
			predicate := OptimizeProcToSerialFunction(a[2])
			reducer := OptimizeProcToSerialFunction(a[3])
			result := NewNil()
			hasResult := false
			if len(a) > 4 {
				result = a[4]
				hasResult = true
			}
			for _, item := range input {
				mapped := mapper(item)
				if !predicate(mapped).Bool() {
					continue
				}
				if !hasResult {
					result = mapped
					hasResult = true
					continue
				}
				result = reducer(result, mapped)
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial map then filter and reduce (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be reduced", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "filter", Params: []*TypeDescriptor{{Label: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", Params: []*TypeDescriptor{{Kind: "any", Transfer: true, Label: "acc", Description: "current accumulator"}, {Kind: "any", Label: "item", Description: "mapped item"}}, Label: "reduce", Return: &TypeDescriptor{Kind: "any", Label: "acc", Description: "next accumulator"}, Description: "combines the accumulator with each mapped item"},
				{Kind: "any", Label: "neutral", Description: "(optional) initial value of the accumulator, defaults to nil", Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_filter_map",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "reduce_filter_map")
			predicate := OptimizeProcToSerialFunction(a[1])
			mapper := OptimizeProcToSerialFunction(a[2])
			reducer := OptimizeProcToSerialFunction(a[3])
			result := NewNil()
			hasResult := false
			if len(a) > 4 {
				result = a[4]
				hasResult = true
			}
			for _, item := range input {
				if !predicate(item).Bool() {
					continue
				}
				mapped := mapper(item)
				if !hasResult {
					result = mapped
					hasResult = true
					continue
				}
				result = reducer(result, mapped)
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial filter then map and reduce (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "list that has to be reduced", NoEscape: true},
				{Kind: "func", Label: "filter", Params: []*TypeDescriptor{{Label: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Params: []*TypeDescriptor{{Kind: "any", Transfer: true, Label: "acc", Description: "current accumulator"}, {Kind: "any", Label: "item", Description: "mapped item"}}, Label: "reduce", Return: &TypeDescriptor{Kind: "any", Label: "acc", Description: "next accumulator"}, Description: "combines the accumulator with each mapped item"},
				{Kind: "any", Label: "neutral", Description: "(optional) initial value of the accumulator, defaults to nil", Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "filter_filter",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "filter_filter")
			left := OptimizeProcToSerialFunction(a[1])
			right := OptimizeProcToSerialFunction(a[2])
			result := make([]Scmer, 0, len(input))
			for _, item := range input {
				if left(item).Bool() && right(item).Bool() {
					result = append(result, item)
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial filter and filter (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "filter", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", Label: "filter", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:    FreshAlloc,
			Forbidden: true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d54 JITValueDesc
				_ = d54
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d118 JITValueDesc
				_ = d118
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d141 JITValueDesc
				_ = d141
				var d142 JITValueDesc
				_ = d142
				var d143 JITValueDesc
				_ = d143
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d150 JITValueDesc
				_ = d150
				var d151 JITValueDesc
				_ = d151
				var d154 JITValueDesc
				_ = d154
				var d155 JITValueDesc
				_ = d155
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(40))
				d1 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				var bbs [6]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(2)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[0].VisitCount >= 0 {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
				}
				bbs[0].VisitCount++
				if ps.General {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_0 = bbs[0].Address
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				ctx.ReclaimUntrackedRegs()
				d3 = args[0]
				d3.ID = 0
				var d4 JITValueDesc
				if d3.Type == tagSlice {
					d4 = jitKnownSliceHeader(ctx, &d3)
				} else {
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d3}, 3)
				}
				ctx.BindReg(d4.Reg, &d4)
				ctx.BindReg(d4.Reg2, &d4)
				ctx.BindReg(d4.Reg3, &d4)
				ctx.FreeDesc(&d3)
				d5 = args[1]
				d5.ID = 0
				var d6 JITValueDesc
				if d5.Loc == LocLambdaTemplate {
					d6 = d5
				} else {
					d6 = ctx.RequestOptimizedCallback(1)
				}
				ctx.FreeDesc(&d5)
				d7 = args[2]
				d7.ID = 0
				var d8 JITValueDesc
				if d7.Loc == LocLambdaTemplate {
					d8 = d7
				} else {
					d8 = ctx.RequestOptimizedCallback(2)
				}
				ctx.FreeDesc(&d7)
				var d9 JITValueDesc
				if d4.SliceSizeKnown {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d9)
				ctx.EnsureDesc(&d10)
				ctx.EnsureDesc(&d9)
				d11 = ctx.EmitGoCallScalar(GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d10, d9}, 3)
				ctx.BindReg(d11.Reg, &d11)
				ctx.BindReg(d11.Reg2, &d11)
				ctx.BindReg(d11.Reg3, &d11)
				ctx.FreeDesc(&d9)
				var d12 JITValueDesc
				if d4.SliceSizeKnown {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
				} else if d4.Loc == LocImm {
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
				} else {
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
					} else if d4.Loc == LocReg {
						d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.EnsureDesc(&d11)
				if d11.Loc == LocReg {
					ctx.ProtectReg(d11.Reg)
				} else if d11.Loc == LocRegPair {
					ctx.ProtectReg(d11.Reg)
					ctx.ProtectReg(d11.Reg2)
				}
				d13 = d11
				if d13.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d13)
				if d13.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d13.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d13.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d13.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(24))
				if d11.Loc == LocReg {
					ctx.UnprotectReg(d11.Reg)
				} else if d11.Loc == LocRegPair {
					ctx.UnprotectReg(d11.Reg)
					ctx.UnprotectReg(d11.Reg2)
				}
				ps14 := PhiState{General: ps.General}
				ps14.OverlayValues = make([]JITValueDesc, 14)
				ps14.OverlayValues[1] = d1
				ps14.OverlayValues[2] = d2
				ps14.OverlayValues[3] = d3
				ps14.OverlayValues[4] = d4
				ps14.OverlayValues[5] = d5
				ps14.OverlayValues[6] = d6
				ps14.OverlayValues[7] = d7
				ps14.OverlayValues[8] = d8
				ps14.OverlayValues[9] = d9
				ps14.OverlayValues[10] = d10
				ps14.OverlayValues[11] = d11
				ps14.OverlayValues[12] = d12
				ps14.OverlayValues[13] = d13
				ps14.PhiValues = make([]JITValueDesc, 2)
				d15 = d11
				ps14.PhiValues[0] = d15
				d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
				ps14.PhiValues[1] = d16
				if ps14.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps14)
				return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d17 := ps.PhiValues[0]
						ctx.EnsureDesc(&d17)
						ctx.EmitStoreRegMem(d17.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d17.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d17.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d18 := ps.PhiValues[1]
						ctx.EnsureDesc(&d18)
						ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(24))
					}
					if bbs[1].VisitCount >= 0 {
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
				}
				bbs[1].VisitCount++
				if ps.General {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_1 = bbs[1].Address
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d1 = ps.PhiValues[0]
				}
				if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d2 = ps.PhiValues[1]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs19 := make([]Reg, 0, 3)
				seenBlockPinnedRegs20 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs20
				for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
					live := d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3)
					if live && !seenBlockPinnedRegs20[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs20[r] = true
						blockPinnedRegs19 = append(blockPinnedRegs19, r)
					}
				}
				unpinBlockRegs21 := func() { for _, r := range blockPinnedRegs19 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs21()
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				var d22 JITValueDesc
				if d2.Loc == LocImm {
					d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d2.Reg)
					ctx.EmitMovRegReg(scratch, d2.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d22)
				}
				if d22.Loc == LocReg && d2.Loc == LocReg && d22.Reg == d2.Reg {
					ctx.TransferReg(d2.Reg)
					d2.Loc = LocNone
				}
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d12)
				var d23 JITValueDesc
				if d22.Loc == LocImm && d12.Loc == LocImm {
					d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d22.Imm.Int() < d12.Imm.Int())}
				} else if d12.Loc == LocImm {
					r0 := ctx.AllocRegExcept(d22.Reg)
					if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d22.Reg, int32(d12.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
						ctx.EmitCmpInt64(d22.Reg, RegR11)
					}
					ctx.EmitSetcc(r0, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
					ctx.BindReg(r0, &d23)
				} else if d22.Loc == LocImm {
					r1 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d12.Reg)
					ctx.EmitSetcc(r1, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
					ctx.BindReg(r1, &d23)
				} else {
					r2 := ctx.AllocRegExcept(d22.Reg)
					ctx.EmitCmpInt64(d22.Reg, d12.Reg)
					ctx.EmitSetcc(r2, CcL)
					d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
					ctx.BindReg(r2, &d23)
				}
				ctx.FreeDesc(&d12)
				d24 = d23
				ctx.EnsureDesc(&d24)
				if d24.Loc != LocImm && d24.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d24.Loc == LocImm {
					if d24.Imm.Bool() {
				ps25 := PhiState{General: ps.General}
				ps25.OverlayValues = make([]JITValueDesc, 25)
				ps25.OverlayValues[1] = d1
				ps25.OverlayValues[2] = d2
				ps25.OverlayValues[3] = d3
				ps25.OverlayValues[4] = d4
				ps25.OverlayValues[5] = d5
				ps25.OverlayValues[6] = d6
				ps25.OverlayValues[7] = d7
				ps25.OverlayValues[8] = d8
				ps25.OverlayValues[9] = d9
				ps25.OverlayValues[10] = d10
				ps25.OverlayValues[11] = d11
				ps25.OverlayValues[12] = d12
				ps25.OverlayValues[13] = d13
				ps25.OverlayValues[15] = d15
				ps25.OverlayValues[16] = d16
				ps25.OverlayValues[17] = d17
				ps25.OverlayValues[18] = d18
				ps25.OverlayValues[22] = d22
				ps25.OverlayValues[23] = d23
				ps25.OverlayValues[24] = d24
						return bbs[2].RenderPS(ps25)
					}
				ps26 := PhiState{General: ps.General}
				ps26.OverlayValues = make([]JITValueDesc, 25)
				ps26.OverlayValues[1] = d1
				ps26.OverlayValues[2] = d2
				ps26.OverlayValues[3] = d3
				ps26.OverlayValues[4] = d4
				ps26.OverlayValues[5] = d5
				ps26.OverlayValues[6] = d6
				ps26.OverlayValues[7] = d7
				ps26.OverlayValues[8] = d8
				ps26.OverlayValues[9] = d9
				ps26.OverlayValues[10] = d10
				ps26.OverlayValues[11] = d11
				ps26.OverlayValues[12] = d12
				ps26.OverlayValues[13] = d13
				ps26.OverlayValues[15] = d15
				ps26.OverlayValues[16] = d16
				ps26.OverlayValues[17] = d17
				ps26.OverlayValues[18] = d18
				ps26.OverlayValues[22] = d22
				ps26.OverlayValues[23] = d23
				ps26.OverlayValues[24] = d24
					return bbs[3].RenderPS(ps26)
				}
				if !ps.General {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d27 := ps.PhiValues[0]
						ctx.EnsureDesc(&d27)
						ctx.EmitStoreRegMem(d27.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
						ctx.EmitStoreRegMem(d27.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
						ctx.EmitStoreRegMem(d27.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d28 := ps.PhiValues[1]
						ctx.EnsureDesc(&d28)
						ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(24))
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
				lbl7 := ctx.ReserveLabel()
				lbl8 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d24.Reg, 0)
				ctx.EmitJcc(CcNE, lbl7)
				ctx.EmitJmp(lbl8)
				ctx.MarkLabel(lbl7)
				ctx.EmitJmp(lbl3)
				ctx.MarkLabel(lbl8)
				ctx.EmitJmp(lbl4)
				ps29 := PhiState{General: true}
				ps29.OverlayValues = make([]JITValueDesc, 29)
				ps29.OverlayValues[1] = d1
				ps29.OverlayValues[2] = d2
				ps29.OverlayValues[3] = d3
				ps29.OverlayValues[4] = d4
				ps29.OverlayValues[5] = d5
				ps29.OverlayValues[6] = d6
				ps29.OverlayValues[7] = d7
				ps29.OverlayValues[8] = d8
				ps29.OverlayValues[9] = d9
				ps29.OverlayValues[10] = d10
				ps29.OverlayValues[11] = d11
				ps29.OverlayValues[12] = d12
				ps29.OverlayValues[13] = d13
				ps29.OverlayValues[15] = d15
				ps29.OverlayValues[16] = d16
				ps29.OverlayValues[17] = d17
				ps29.OverlayValues[18] = d18
				ps29.OverlayValues[22] = d22
				ps29.OverlayValues[23] = d23
				ps29.OverlayValues[24] = d24
				ps29.OverlayValues[27] = d27
				ps29.OverlayValues[28] = d28
				ps30 := PhiState{General: true}
				ps30.OverlayValues = make([]JITValueDesc, 29)
				ps30.OverlayValues[1] = d1
				ps30.OverlayValues[2] = d2
				ps30.OverlayValues[3] = d3
				ps30.OverlayValues[4] = d4
				ps30.OverlayValues[5] = d5
				ps30.OverlayValues[6] = d6
				ps30.OverlayValues[7] = d7
				ps30.OverlayValues[8] = d8
				ps30.OverlayValues[9] = d9
				ps30.OverlayValues[10] = d10
				ps30.OverlayValues[11] = d11
				ps30.OverlayValues[12] = d12
				ps30.OverlayValues[13] = d13
				ps30.OverlayValues[15] = d15
				ps30.OverlayValues[16] = d16
				ps30.OverlayValues[17] = d17
				ps30.OverlayValues[18] = d18
				ps30.OverlayValues[22] = d22
				ps30.OverlayValues[23] = d23
				ps30.OverlayValues[24] = d24
				ps30.OverlayValues[27] = d27
				ps30.OverlayValues[28] = d28
				snap31 := d1
				snap32 := d2
				snap33 := d3
				snap34 := d4
				snap35 := d5
				snap36 := d6
				snap37 := d7
				snap38 := d8
				snap39 := d9
				snap40 := d10
				snap41 := d11
				snap42 := d12
				snap43 := d13
				snap44 := d15
				snap45 := d16
				snap46 := d17
				snap47 := d18
				snap48 := d22
				snap49 := d23
				snap50 := d24
				snap51 := d27
				snap52 := d28
				alloc53 := ctx.SnapshotAllocState()
				if !bbs[3].Rendered {
					bbs[3].RenderPS(ps30)
				}
				ctx.RestoreAllocState(alloc53)
				d1 = snap31
				d2 = snap32
				d3 = snap33
				d4 = snap34
				d5 = snap35
				d6 = snap36
				d7 = snap37
				d8 = snap38
				d9 = snap39
				d10 = snap40
				d11 = snap41
				d12 = snap42
				d13 = snap43
				d15 = snap44
				d16 = snap45
				d17 = snap46
				d18 = snap47
				d22 = snap48
				d23 = snap49
				d24 = snap50
				d27 = snap51
				d28 = snap52
				if !bbs[2].Rendered {
					return bbs[2].RenderPS(ps29)
				}
				return result
				ctx.FreeDesc(&d23)
				return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[2].VisitCount >= 0 {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
				}
				bbs[2].VisitCount++
				if ps.General {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_2 = bbs[2].Address
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d22)
				r3 := ctx.AllocReg()
				ctx.EnsureDesc(&d22)
				ctx.EnsureDesc(&d4)
				if d22.Loc == LocImm {
					ctx.EmitMovRegImm64(r3, uint64(d22.Imm.Int()) * 16)
				} else {
					ctx.EmitMovRegReg(r3, d22.Reg)
					ctx.EmitShlRegImm8(r3, 4)
				}
				if d4.Loc == LocImm {
					ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.EmitAddInt64(r3, RegR11)
				} else {
					ctx.EmitAddInt64(r3, d4.Reg)
				}
				r4 := ctx.AllocRegExcept(r3)
				r5 := ctx.AllocRegExcept(r3, r4)
				ctx.EmitMovRegMem(r4, r3, 0)
				ctx.EmitMovRegMem(r5, r3, 8)
				ctx.FreeReg(r3)
				d54 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d54)
				ctx.BindReg(r5, &d54)
				stackArray55 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreScmerToStack(d54, int32(stackArray55)+int32(0))
				r6 := ctx.AllocReg()
				r7 := ctx.AllocRegExcept(r6)
				r8 := ctx.AllocRegExcept(r6, r7)
				ctx.EmitLeaRegMem(r6, RegRSP, int32(stackArray55))
				ctx.EmitMovRegImm64(r7, uint64(1))
				ctx.EmitMovRegImm64(r8, uint64(1))
				d56 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r6, &d56)
				ctx.BindReg(r7, &d56)
				ctx.BindReg(r8, &d56)
				callbackArgs58 := make([]JITValueDesc, 1)
				callbackArgs58[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray55)+0}
				var d57 JITValueDesc
				ctx.FreeDesc(&d56)
				if d6.Loc == LocLambdaTemplate && d6.Lambda != nil {
					callbackResultOff59 := ctx.AllocSpill(16)
					ctx.setStackPointer(jitStackRootFrameBP, callbackResultOff59, true)
					outerRegs60 := ctx.PreserveOuterRegs()
					d57 = JITEmitProcInlineWithOuter(ctx, &d6.Lambda.Proc, d6.Lambda.Outer, callbackArgs58, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					ctx.EnsureDesc(&d57)
					ctx.EmitStoreRegMem(d57.Reg, RegRBP, callbackResultOff59)
					ctx.EmitStoreRegMem(d57.Reg2, RegRBP, callbackResultOff59+8)
					ctx.RestoreOuterRegs(outerRegs60)
					d57 = JITValueDesc{Loc: LocStackPair, Type: d57.Type, StackOff: callbackResultOff59, NoHeapPointer: d57.NoHeapPointer}
					liveRegs61 := make([]Reg, 0, 24)
					seenLiveRegs62 := make(map[Reg]bool)
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := (d1.Loc == LocReg && r == d1.Reg) ||
							(d1.Loc == LocRegPair && (r == d1.Reg || r == d1.Reg2)) ||
							(d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := (d11.Loc == LocReg && r == d11.Reg) ||
							(d11.Loc == LocRegPair && (r == d11.Reg || r == d11.Reg2)) ||
							(d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d22.Reg, d22.Reg2, d22.Reg3} {
						live := (d22.Loc == LocReg && r == d22.Reg) ||
							(d22.Loc == LocRegPair && (r == d22.Reg || r == d22.Reg2)) ||
							(d22.Loc == LocRegTriple && (r == d22.Reg || r == d22.Reg2 || r == d22.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d4.Reg, d4.Reg2, d4.Reg3} {
						live := (d4.Loc == LocReg && r == d4.Reg) ||
							(d4.Loc == LocRegPair && (r == d4.Reg || r == d4.Reg2)) ||
							(d4.Loc == LocRegTriple && (r == d4.Reg || r == d4.Reg2 || r == d4.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d54.Reg, d54.Reg2, d54.Reg3} {
						live := (d54.Loc == LocReg && r == d54.Reg) ||
							(d54.Loc == LocRegPair && (r == d54.Reg || r == d54.Reg2)) ||
							(d54.Loc == LocRegTriple && (r == d54.Reg || r == d54.Reg2 || r == d54.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d56.Reg, d56.Reg2, d56.Reg3} {
						live := (d56.Loc == LocReg && r == d56.Reg) ||
							(d56.Loc == LocRegPair && (r == d56.Reg || r == d56.Reg2)) ||
							(d56.Loc == LocRegTriple && (r == d56.Reg || r == d56.Reg2 || r == d56.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d6.Reg, d6.Reg2, d6.Reg3} {
						live := (d6.Loc == LocReg && r == d6.Reg) ||
							(d6.Loc == LocRegPair && (r == d6.Reg || r == d6.Reg2)) ||
							(d6.Loc == LocRegTriple && (r == d6.Reg || r == d6.Reg2 || r == d6.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					for _, r := range []Reg{d8.Reg, d8.Reg2, d8.Reg3} {
						live := (d8.Loc == LocReg && r == d8.Reg) ||
							(d8.Loc == LocRegPair && (r == d8.Reg || r == d8.Reg2)) ||
							(d8.Loc == LocRegTriple && (r == d8.Reg || r == d8.Reg2 || r == d8.Reg3))
						if live && !seenLiveRegs62[r] {
							ctx.ProtectReg(r)
							seenLiveRegs62[r] = true
							liveRegs61 = append(liveRegs61, r)
						}
					}
					ctx.EnsureDesc(&d57)
					for _, r := range liveRegs61 { ctx.UnprotectReg(r) }
				} else {
					callbackCallArgs := make([]JITValueDesc, 0, 2)
					callbackCallArgs = append(callbackCallArgs, d6)
					callbackCallArgs = append(callbackCallArgs, callbackArgs58...)
					d57 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
				}
				d64 = d57
				d64.ID = 0
				d63 = ctx.EmitBoolDesc(&d64, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d57)
				d65 = d63
				ctx.EnsureDesc(&d65)
				if d65.Loc != LocImm && d65.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d65.Loc == LocImm {
					if d65.Imm.Bool() {
				ps66 := PhiState{General: ps.General}
				ps66.OverlayValues = make([]JITValueDesc, 66)
				ps66.OverlayValues[1] = d1
				ps66.OverlayValues[2] = d2
				ps66.OverlayValues[3] = d3
				ps66.OverlayValues[4] = d4
				ps66.OverlayValues[5] = d5
				ps66.OverlayValues[6] = d6
				ps66.OverlayValues[7] = d7
				ps66.OverlayValues[8] = d8
				ps66.OverlayValues[9] = d9
				ps66.OverlayValues[10] = d10
				ps66.OverlayValues[11] = d11
				ps66.OverlayValues[12] = d12
				ps66.OverlayValues[13] = d13
				ps66.OverlayValues[15] = d15
				ps66.OverlayValues[16] = d16
				ps66.OverlayValues[17] = d17
				ps66.OverlayValues[18] = d18
				ps66.OverlayValues[22] = d22
				ps66.OverlayValues[23] = d23
				ps66.OverlayValues[24] = d24
				ps66.OverlayValues[27] = d27
				ps66.OverlayValues[28] = d28
				ps66.OverlayValues[54] = d54
				ps66.OverlayValues[56] = d56
				ps66.OverlayValues[57] = d57
				ps66.OverlayValues[63] = d63
				ps66.OverlayValues[64] = d64
				ps66.OverlayValues[65] = d65
						return bbs[5].RenderPS(ps66)
					}
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d67 = d1
				if d67.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d67)
				if d67.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d67.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d67.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d67.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d68 = d22
				if d68.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d68)
				ctx.EmitStoreToStack(d68, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ps69 := PhiState{General: ps.General}
				ps69.OverlayValues = make([]JITValueDesc, 69)
				ps69.OverlayValues[1] = d1
				ps69.OverlayValues[2] = d2
				ps69.OverlayValues[3] = d3
				ps69.OverlayValues[4] = d4
				ps69.OverlayValues[5] = d5
				ps69.OverlayValues[6] = d6
				ps69.OverlayValues[7] = d7
				ps69.OverlayValues[8] = d8
				ps69.OverlayValues[9] = d9
				ps69.OverlayValues[10] = d10
				ps69.OverlayValues[11] = d11
				ps69.OverlayValues[12] = d12
				ps69.OverlayValues[13] = d13
				ps69.OverlayValues[15] = d15
				ps69.OverlayValues[16] = d16
				ps69.OverlayValues[17] = d17
				ps69.OverlayValues[18] = d18
				ps69.OverlayValues[22] = d22
				ps69.OverlayValues[23] = d23
				ps69.OverlayValues[24] = d24
				ps69.OverlayValues[27] = d27
				ps69.OverlayValues[28] = d28
				ps69.OverlayValues[54] = d54
				ps69.OverlayValues[56] = d56
				ps69.OverlayValues[57] = d57
				ps69.OverlayValues[63] = d63
				ps69.OverlayValues[64] = d64
				ps69.OverlayValues[65] = d65
				ps69.OverlayValues[67] = d67
				ps69.OverlayValues[68] = d68
				ps69.PhiValues = make([]JITValueDesc, 2)
				d70 = d1
				ps69.PhiValues[0] = d70
				d71 = d22
				ps69.PhiValues[1] = d71
					return bbs[1].RenderPS(ps69)
				}
				if !ps.General {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
				lbl9 := ctx.ReserveLabel()
				lbl10 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d65.Reg, 0)
				ctx.EmitJcc(CcNE, lbl9)
				ctx.EmitJmp(lbl10)
				ctx.MarkLabel(lbl9)
				ctx.EmitJmp(lbl6)
				ctx.MarkLabel(lbl10)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d72 = d1
				if d72.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d72)
				if d72.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d72.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d72.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d72.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d73 = d22
				if d73.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d73)
				ctx.EmitStoreToStack(d73, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ctx.EmitJmp(lbl2)
				ps74 := PhiState{General: true}
				ps74.OverlayValues = make([]JITValueDesc, 74)
				ps74.OverlayValues[1] = d1
				ps74.OverlayValues[2] = d2
				ps74.OverlayValues[3] = d3
				ps74.OverlayValues[4] = d4
				ps74.OverlayValues[5] = d5
				ps74.OverlayValues[6] = d6
				ps74.OverlayValues[7] = d7
				ps74.OverlayValues[8] = d8
				ps74.OverlayValues[9] = d9
				ps74.OverlayValues[10] = d10
				ps74.OverlayValues[11] = d11
				ps74.OverlayValues[12] = d12
				ps74.OverlayValues[13] = d13
				ps74.OverlayValues[15] = d15
				ps74.OverlayValues[16] = d16
				ps74.OverlayValues[17] = d17
				ps74.OverlayValues[18] = d18
				ps74.OverlayValues[22] = d22
				ps74.OverlayValues[23] = d23
				ps74.OverlayValues[24] = d24
				ps74.OverlayValues[27] = d27
				ps74.OverlayValues[28] = d28
				ps74.OverlayValues[54] = d54
				ps74.OverlayValues[56] = d56
				ps74.OverlayValues[57] = d57
				ps74.OverlayValues[63] = d63
				ps74.OverlayValues[64] = d64
				ps74.OverlayValues[65] = d65
				ps74.OverlayValues[67] = d67
				ps74.OverlayValues[68] = d68
				ps74.OverlayValues[70] = d70
				ps74.OverlayValues[71] = d71
				ps74.OverlayValues[72] = d72
				ps74.OverlayValues[73] = d73
				ps75 := PhiState{General: true}
				ps75.OverlayValues = make([]JITValueDesc, 74)
				ps75.OverlayValues[1] = d1
				ps75.OverlayValues[2] = d2
				ps75.OverlayValues[3] = d3
				ps75.OverlayValues[4] = d4
				ps75.OverlayValues[5] = d5
				ps75.OverlayValues[6] = d6
				ps75.OverlayValues[7] = d7
				ps75.OverlayValues[8] = d8
				ps75.OverlayValues[9] = d9
				ps75.OverlayValues[10] = d10
				ps75.OverlayValues[11] = d11
				ps75.OverlayValues[12] = d12
				ps75.OverlayValues[13] = d13
				ps75.OverlayValues[15] = d15
				ps75.OverlayValues[16] = d16
				ps75.OverlayValues[17] = d17
				ps75.OverlayValues[18] = d18
				ps75.OverlayValues[22] = d22
				ps75.OverlayValues[23] = d23
				ps75.OverlayValues[24] = d24
				ps75.OverlayValues[27] = d27
				ps75.OverlayValues[28] = d28
				ps75.OverlayValues[54] = d54
				ps75.OverlayValues[56] = d56
				ps75.OverlayValues[57] = d57
				ps75.OverlayValues[63] = d63
				ps75.OverlayValues[64] = d64
				ps75.OverlayValues[65] = d65
				ps75.OverlayValues[67] = d67
				ps75.OverlayValues[68] = d68
				ps75.OverlayValues[70] = d70
				ps75.OverlayValues[71] = d71
				ps75.OverlayValues[72] = d72
				ps75.OverlayValues[73] = d73
				ps75.PhiValues = make([]JITValueDesc, 2)
				d76 = d1
				ps75.PhiValues[0] = d76
				d77 = d22
				ps75.PhiValues[1] = d77
				snap78 := d1
				snap79 := d2
				snap80 := d3
				snap81 := d4
				snap82 := d5
				snap83 := d6
				snap84 := d7
				snap85 := d8
				snap86 := d9
				snap87 := d10
				snap88 := d11
				snap89 := d12
				snap90 := d13
				snap91 := d15
				snap92 := d16
				snap93 := d17
				snap94 := d18
				snap95 := d22
				snap96 := d23
				snap97 := d24
				snap98 := d27
				snap99 := d28
				snap100 := d54
				snap101 := d56
				snap102 := d57
				snap103 := d63
				snap104 := d64
				snap105 := d65
				snap106 := d67
				snap107 := d68
				snap108 := d70
				snap109 := d71
				snap110 := d72
				snap111 := d73
				snap112 := d76
				snap113 := d77
				alloc114 := ctx.SnapshotAllocState()
				if !bbs[1].Rendered {
					bbs[1].RenderPS(ps75)
				}
				ctx.RestoreAllocState(alloc114)
				d1 = snap78
				d2 = snap79
				d3 = snap80
				d4 = snap81
				d5 = snap82
				d6 = snap83
				d7 = snap84
				d8 = snap85
				d9 = snap86
				d10 = snap87
				d11 = snap88
				d12 = snap89
				d13 = snap90
				d15 = snap91
				d16 = snap92
				d17 = snap93
				d18 = snap94
				d22 = snap95
				d23 = snap96
				d24 = snap97
				d27 = snap98
				d28 = snap99
				d54 = snap100
				d56 = snap101
				d57 = snap102
				d63 = snap103
				d64 = snap104
				d65 = snap105
				d67 = snap106
				d68 = snap107
				d70 = snap108
				d71 = snap109
				d72 = snap110
				d73 = snap111
				d76 = snap112
				d77 = snap113
				if !bbs[5].Rendered {
					return bbs[5].RenderPS(ps74)
				}
				return result
				ctx.FreeDesc(&d63)
				return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[3].VisitCount >= 0 {
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
				}
				bbs[3].VisitCount++
				if ps.General {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_3 = bbs[3].Address
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
					d54 = ps.OverlayValues[54]
				}
				if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
					d56 = ps.OverlayValues[56]
				}
				if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
					d57 = ps.OverlayValues[57]
				}
				if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
					d63 = ps.OverlayValues[63]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
					d65 = ps.OverlayValues[65]
				}
				if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
					d67 = ps.OverlayValues[67]
				}
				if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
					d68 = ps.OverlayValues[68]
				}
				if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
					d70 = ps.OverlayValues[70]
				}
				if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
					d71 = ps.OverlayValues[71]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
					d76 = ps.OverlayValues[76]
				}
				if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
					d77 = ps.OverlayValues[77]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs115 := make([]Reg, 0, 3)
				seenBlockPinnedRegs116 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs116
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs116[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs116[r] = true
						blockPinnedRegs115 = append(blockPinnedRegs115, r)
					}
				}
				unpinBlockRegs117 := func() { for _, r := range blockPinnedRegs115 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs117()
				d118 = ctx.EmitNewSliceFromGoSlice(&d1)
				ctx.EnsureDesc(&d118)
				if d118.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d118, &result)
					result.Type = d118.Type
				} else {
					switch d118.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d118)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d118)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d118)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						ctx.EmitMovPairToResult(&d118, &result)
						result.Type = d118.Type
					}
				}
				ctx.EmitJmp(lbl0)
				return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[4].VisitCount >= 0 {
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
				}
				bbs[4].VisitCount++
				if ps.General {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_4 = bbs[4].Address
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
					d54 = ps.OverlayValues[54]
				}
				if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
					d56 = ps.OverlayValues[56]
				}
				if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
					d57 = ps.OverlayValues[57]
				}
				if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
					d63 = ps.OverlayValues[63]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
					d65 = ps.OverlayValues[65]
				}
				if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
					d67 = ps.OverlayValues[67]
				}
				if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
					d68 = ps.OverlayValues[68]
				}
				if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
					d70 = ps.OverlayValues[70]
				}
				if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
					d71 = ps.OverlayValues[71]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
					d76 = ps.OverlayValues[76]
				}
				if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
					d77 = ps.OverlayValues[77]
				}
				if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
					d118 = ps.OverlayValues[118]
				}
				ctx.ReclaimUntrackedRegs()
				blockPinnedRegs119 := make([]Reg, 0, 3)
				seenBlockPinnedRegs120 := make(map[Reg]bool)
				_ = seenBlockPinnedRegs120
				for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
					live := d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3)
					if live && !seenBlockPinnedRegs120[r] {
						ctx.ProtectReg(r)
						seenBlockPinnedRegs120[r] = true
						blockPinnedRegs119 = append(blockPinnedRegs119, r)
					}
				}
				unpinBlockRegs121 := func() { for _, r := range blockPinnedRegs119 { ctx.UnprotectReg(r) } }
				defer unpinBlockRegs121()
				stackArray122 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreScmerToStack(d54, int32(stackArray122)+int32(0))
				r9 := ctx.AllocReg()
				r10 := ctx.AllocRegExcept(r9)
				r11 := ctx.AllocRegExcept(r9, r10)
				ctx.EmitLeaRegMem(r9, RegRSP, int32(stackArray122))
				ctx.EmitMovRegImm64(r10, uint64(1))
				ctx.EmitMovRegImm64(r11, uint64(1))
				d123 = JITValueDesc{Loc: LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r9, &d123)
				ctx.BindReg(r10, &d123)
				ctx.BindReg(r11, &d123)
				ctx.EnsureDesc(&d1)
				if d1.Loc != LocRegTriple { panic("jit: append requires a Go slice header") }
				lbl11 := ctx.ReserveLabel()
				ctx.EmitCmpInt64(d1.Reg2, d1.Reg3)
				ctx.EmitJcc(CcB, lbl11)
				ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{{Loc: LocImm, Type: tagString, Imm: NewString("jit: generated append exceeded its fixed capacity")}})
				ctx.MarkLabel(lbl11)
				d124 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg2, NoHeapPointer: true}
				ctx.BindReg(d1.Reg2, &d124)
				d125 = ctx.EmitSliceElementAddress(&d1, &d124, int32(16))
				d126 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray122)}
				ctx.EmitStoreScmerAt(&d125, &d126)
				ctx.FreeDesc(&d125)
				ctx.EmitAddRegImm32(d1.Reg2, 1)
				d127 = d1
				ctx.BindReg(d127.Reg, &d127)
				ctx.BindReg(d127.Reg2, &d127)
				ctx.BindReg(d127.Reg3, &d127)
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				ctx.EnsureDesc(&d127)
				if d127.Loc == LocReg {
					ctx.ProtectReg(d127.Reg)
				} else if d127.Loc == LocRegPair {
					ctx.ProtectReg(d127.Reg)
					ctx.ProtectReg(d127.Reg2)
				}
				d128 = d127
				if d128.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d128)
				if d128.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d128.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d128.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d128.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d129 = d22
				if d129.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d129)
				ctx.EmitStoreToStack(d129, int32(bbs[1].PhiBase)+int32(24))
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				if d127.Loc == LocReg {
					ctx.UnprotectReg(d127.Reg)
				} else if d127.Loc == LocRegPair {
					ctx.UnprotectReg(d127.Reg)
					ctx.UnprotectReg(d127.Reg2)
				}
				ps130 := PhiState{General: ps.General}
				ps130.OverlayValues = make([]JITValueDesc, 130)
				ps130.OverlayValues[1] = d1
				ps130.OverlayValues[2] = d2
				ps130.OverlayValues[3] = d3
				ps130.OverlayValues[4] = d4
				ps130.OverlayValues[5] = d5
				ps130.OverlayValues[6] = d6
				ps130.OverlayValues[7] = d7
				ps130.OverlayValues[8] = d8
				ps130.OverlayValues[9] = d9
				ps130.OverlayValues[10] = d10
				ps130.OverlayValues[11] = d11
				ps130.OverlayValues[12] = d12
				ps130.OverlayValues[13] = d13
				ps130.OverlayValues[15] = d15
				ps130.OverlayValues[16] = d16
				ps130.OverlayValues[17] = d17
				ps130.OverlayValues[18] = d18
				ps130.OverlayValues[22] = d22
				ps130.OverlayValues[23] = d23
				ps130.OverlayValues[24] = d24
				ps130.OverlayValues[27] = d27
				ps130.OverlayValues[28] = d28
				ps130.OverlayValues[54] = d54
				ps130.OverlayValues[56] = d56
				ps130.OverlayValues[57] = d57
				ps130.OverlayValues[63] = d63
				ps130.OverlayValues[64] = d64
				ps130.OverlayValues[65] = d65
				ps130.OverlayValues[67] = d67
				ps130.OverlayValues[68] = d68
				ps130.OverlayValues[70] = d70
				ps130.OverlayValues[71] = d71
				ps130.OverlayValues[72] = d72
				ps130.OverlayValues[73] = d73
				ps130.OverlayValues[76] = d76
				ps130.OverlayValues[77] = d77
				ps130.OverlayValues[118] = d118
				ps130.OverlayValues[123] = d123
				ps130.OverlayValues[124] = d124
				ps130.OverlayValues[125] = d125
				ps130.OverlayValues[126] = d126
				ps130.OverlayValues[127] = d127
				ps130.OverlayValues[128] = d128
				ps130.OverlayValues[129] = d129
				ps130.PhiValues = make([]JITValueDesc, 2)
				d131 = d127
				ps130.PhiValues[0] = d131
				d132 = d22
				ps130.PhiValues[1] = d132
				if ps130.General && bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				return bbs[1].RenderPS(ps130)
				return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
				if !ps.General {
					if bbs[5].VisitCount >= 0 {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
				}
				bbs[5].VisitCount++
				if ps.General {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					bbpos_0_5 = bbs[5].Address
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
				}
				d1 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase0)+int32(0)}
				d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0)+int32(24)}
				if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
					d1 = ps.OverlayValues[1]
				}
				if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
					d2 = ps.OverlayValues[2]
				}
				if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
					d3 = ps.OverlayValues[3]
				}
				if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
					d4 = ps.OverlayValues[4]
				}
				if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
					d5 = ps.OverlayValues[5]
				}
				if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
					d6 = ps.OverlayValues[6]
				}
				if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
					d7 = ps.OverlayValues[7]
				}
				if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
					d8 = ps.OverlayValues[8]
				}
				if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
					d9 = ps.OverlayValues[9]
				}
				if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
					d10 = ps.OverlayValues[10]
				}
				if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
					d11 = ps.OverlayValues[11]
				}
				if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
					d12 = ps.OverlayValues[12]
				}
				if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
					d13 = ps.OverlayValues[13]
				}
				if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
					d15 = ps.OverlayValues[15]
				}
				if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
					d16 = ps.OverlayValues[16]
				}
				if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
					d17 = ps.OverlayValues[17]
				}
				if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
					d18 = ps.OverlayValues[18]
				}
				if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
					d22 = ps.OverlayValues[22]
				}
				if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
					d23 = ps.OverlayValues[23]
				}
				if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
					d24 = ps.OverlayValues[24]
				}
				if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
					d27 = ps.OverlayValues[27]
				}
				if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
					d28 = ps.OverlayValues[28]
				}
				if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
					d54 = ps.OverlayValues[54]
				}
				if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
					d56 = ps.OverlayValues[56]
				}
				if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
					d57 = ps.OverlayValues[57]
				}
				if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
					d63 = ps.OverlayValues[63]
				}
				if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
					d64 = ps.OverlayValues[64]
				}
				if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
					d65 = ps.OverlayValues[65]
				}
				if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
					d67 = ps.OverlayValues[67]
				}
				if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
					d68 = ps.OverlayValues[68]
				}
				if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
					d70 = ps.OverlayValues[70]
				}
				if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
					d71 = ps.OverlayValues[71]
				}
				if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
					d72 = ps.OverlayValues[72]
				}
				if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
					d73 = ps.OverlayValues[73]
				}
				if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
					d76 = ps.OverlayValues[76]
				}
				if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
					d77 = ps.OverlayValues[77]
				}
				if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
					d118 = ps.OverlayValues[118]
				}
				if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
					d123 = ps.OverlayValues[123]
				}
				if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
					d124 = ps.OverlayValues[124]
				}
				if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
					d125 = ps.OverlayValues[125]
				}
				if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
					d126 = ps.OverlayValues[126]
				}
				if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
					d127 = ps.OverlayValues[127]
				}
				if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
					d128 = ps.OverlayValues[128]
				}
				if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
					d129 = ps.OverlayValues[129]
				}
				if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
					d131 = ps.OverlayValues[131]
				}
				if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
					d132 = ps.OverlayValues[132]
				}
				ctx.ReclaimUntrackedRegs()
				stackArray133 := ctx.AllocStack(int32(16))
				ctx.EnsureDesc(&d54)
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreScmerToStack(d54, int32(stackArray133)+int32(0))
				ctx.FreeDesc(&d54)
				r12 := ctx.AllocReg()
				r13 := ctx.AllocRegExcept(r12)
				r14 := ctx.AllocRegExcept(r12, r13)
				ctx.EmitLeaRegMem(r12, RegRSP, int32(stackArray133))
				ctx.EmitMovRegImm64(r13, uint64(1))
				ctx.EmitMovRegImm64(r14, uint64(1))
				d134 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				ctx.BindReg(r12, &d134)
				ctx.BindReg(r13, &d134)
				ctx.BindReg(r14, &d134)
				callbackArgs136 := make([]JITValueDesc, 1)
				callbackArgs136[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray133)+0}
				var d135 JITValueDesc
				ctx.FreeDesc(&d134)
				if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
					callbackResultOff137 := ctx.AllocSpill(16)
					ctx.setStackPointer(jitStackRootFrameBP, callbackResultOff137, true)
					outerRegs138 := ctx.PreserveOuterRegs()
					d135 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, callbackArgs136, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					ctx.EnsureDesc(&d135)
					ctx.EmitStoreRegMem(d135.Reg, RegRBP, callbackResultOff137)
					ctx.EmitStoreRegMem(d135.Reg2, RegRBP, callbackResultOff137+8)
					ctx.RestoreOuterRegs(outerRegs138)
					d135 = JITValueDesc{Loc: LocStackPair, Type: d135.Type, StackOff: callbackResultOff137, NoHeapPointer: d135.NoHeapPointer}
					liveRegs139 := make([]Reg, 0, 24)
					seenLiveRegs140 := make(map[Reg]bool)
					for _, r := range []Reg{d1.Reg, d1.Reg2, d1.Reg3} {
						live := (d1.Loc == LocReg && r == d1.Reg) ||
							(d1.Loc == LocRegPair && (r == d1.Reg || r == d1.Reg2)) ||
							(d1.Loc == LocRegTriple && (r == d1.Reg || r == d1.Reg2 || r == d1.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					for _, r := range []Reg{d11.Reg, d11.Reg2, d11.Reg3} {
						live := (d11.Loc == LocReg && r == d11.Reg) ||
							(d11.Loc == LocRegPair && (r == d11.Reg || r == d11.Reg2)) ||
							(d11.Loc == LocRegTriple && (r == d11.Reg || r == d11.Reg2 || r == d11.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					for _, r := range []Reg{d118.Reg, d118.Reg2, d118.Reg3} {
						live := (d118.Loc == LocReg && r == d118.Reg) ||
							(d118.Loc == LocRegPair && (r == d118.Reg || r == d118.Reg2)) ||
							(d118.Loc == LocRegTriple && (r == d118.Reg || r == d118.Reg2 || r == d118.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					for _, r := range []Reg{d127.Reg, d127.Reg2, d127.Reg3} {
						live := (d127.Loc == LocReg && r == d127.Reg) ||
							(d127.Loc == LocRegPair && (r == d127.Reg || r == d127.Reg2)) ||
							(d127.Loc == LocRegTriple && (r == d127.Reg || r == d127.Reg2 || r == d127.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					for _, r := range []Reg{d134.Reg, d134.Reg2, d134.Reg3} {
						live := (d134.Loc == LocReg && r == d134.Reg) ||
							(d134.Loc == LocRegPair && (r == d134.Reg || r == d134.Reg2)) ||
							(d134.Loc == LocRegTriple && (r == d134.Reg || r == d134.Reg2 || r == d134.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					for _, r := range []Reg{d22.Reg, d22.Reg2, d22.Reg3} {
						live := (d22.Loc == LocReg && r == d22.Reg) ||
							(d22.Loc == LocRegPair && (r == d22.Reg || r == d22.Reg2)) ||
							(d22.Loc == LocRegTriple && (r == d22.Reg || r == d22.Reg2 || r == d22.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					for _, r := range []Reg{d4.Reg, d4.Reg2, d4.Reg3} {
						live := (d4.Loc == LocReg && r == d4.Reg) ||
							(d4.Loc == LocRegPair && (r == d4.Reg || r == d4.Reg2)) ||
							(d4.Loc == LocRegTriple && (r == d4.Reg || r == d4.Reg2 || r == d4.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					for _, r := range []Reg{d8.Reg, d8.Reg2, d8.Reg3} {
						live := (d8.Loc == LocReg && r == d8.Reg) ||
							(d8.Loc == LocRegPair && (r == d8.Reg || r == d8.Reg2)) ||
							(d8.Loc == LocRegTriple && (r == d8.Reg || r == d8.Reg2 || r == d8.Reg3))
						if live && !seenLiveRegs140[r] {
							ctx.ProtectReg(r)
							seenLiveRegs140[r] = true
							liveRegs139 = append(liveRegs139, r)
						}
					}
					ctx.EnsureDesc(&d135)
					for _, r := range liveRegs139 { ctx.UnprotectReg(r) }
				} else {
					callbackCallArgs := make([]JITValueDesc, 0, 2)
					callbackCallArgs = append(callbackCallArgs, d8)
					callbackCallArgs = append(callbackCallArgs, callbackArgs136...)
					d135 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
				}
				d142 = d135
				d142.ID = 0
				d141 = ctx.EmitBoolDesc(&d142, JITValueDesc{Loc: LocAny})
				ctx.FreeDesc(&d135)
				d143 = d141
				ctx.EnsureDesc(&d143)
				if d143.Loc != LocImm && d143.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				if d143.Loc == LocImm {
					if d143.Imm.Bool() {
				ps144 := PhiState{General: ps.General}
				ps144.OverlayValues = make([]JITValueDesc, 144)
				ps144.OverlayValues[1] = d1
				ps144.OverlayValues[2] = d2
				ps144.OverlayValues[3] = d3
				ps144.OverlayValues[4] = d4
				ps144.OverlayValues[5] = d5
				ps144.OverlayValues[6] = d6
				ps144.OverlayValues[7] = d7
				ps144.OverlayValues[8] = d8
				ps144.OverlayValues[9] = d9
				ps144.OverlayValues[10] = d10
				ps144.OverlayValues[11] = d11
				ps144.OverlayValues[12] = d12
				ps144.OverlayValues[13] = d13
				ps144.OverlayValues[15] = d15
				ps144.OverlayValues[16] = d16
				ps144.OverlayValues[17] = d17
				ps144.OverlayValues[18] = d18
				ps144.OverlayValues[22] = d22
				ps144.OverlayValues[23] = d23
				ps144.OverlayValues[24] = d24
				ps144.OverlayValues[27] = d27
				ps144.OverlayValues[28] = d28
				ps144.OverlayValues[54] = d54
				ps144.OverlayValues[56] = d56
				ps144.OverlayValues[57] = d57
				ps144.OverlayValues[63] = d63
				ps144.OverlayValues[64] = d64
				ps144.OverlayValues[65] = d65
				ps144.OverlayValues[67] = d67
				ps144.OverlayValues[68] = d68
				ps144.OverlayValues[70] = d70
				ps144.OverlayValues[71] = d71
				ps144.OverlayValues[72] = d72
				ps144.OverlayValues[73] = d73
				ps144.OverlayValues[76] = d76
				ps144.OverlayValues[77] = d77
				ps144.OverlayValues[118] = d118
				ps144.OverlayValues[123] = d123
				ps144.OverlayValues[124] = d124
				ps144.OverlayValues[125] = d125
				ps144.OverlayValues[126] = d126
				ps144.OverlayValues[127] = d127
				ps144.OverlayValues[128] = d128
				ps144.OverlayValues[129] = d129
				ps144.OverlayValues[131] = d131
				ps144.OverlayValues[132] = d132
				ps144.OverlayValues[134] = d134
				ps144.OverlayValues[135] = d135
				ps144.OverlayValues[141] = d141
				ps144.OverlayValues[142] = d142
				ps144.OverlayValues[143] = d143
						return bbs[4].RenderPS(ps144)
					}
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d145 = d1
				if d145.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d145)
				if d145.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d145.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d145.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d145.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d146 = d22
				if d146.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d146)
				ctx.EmitStoreToStack(d146, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ps147 := PhiState{General: ps.General}
				ps147.OverlayValues = make([]JITValueDesc, 147)
				ps147.OverlayValues[1] = d1
				ps147.OverlayValues[2] = d2
				ps147.OverlayValues[3] = d3
				ps147.OverlayValues[4] = d4
				ps147.OverlayValues[5] = d5
				ps147.OverlayValues[6] = d6
				ps147.OverlayValues[7] = d7
				ps147.OverlayValues[8] = d8
				ps147.OverlayValues[9] = d9
				ps147.OverlayValues[10] = d10
				ps147.OverlayValues[11] = d11
				ps147.OverlayValues[12] = d12
				ps147.OverlayValues[13] = d13
				ps147.OverlayValues[15] = d15
				ps147.OverlayValues[16] = d16
				ps147.OverlayValues[17] = d17
				ps147.OverlayValues[18] = d18
				ps147.OverlayValues[22] = d22
				ps147.OverlayValues[23] = d23
				ps147.OverlayValues[24] = d24
				ps147.OverlayValues[27] = d27
				ps147.OverlayValues[28] = d28
				ps147.OverlayValues[54] = d54
				ps147.OverlayValues[56] = d56
				ps147.OverlayValues[57] = d57
				ps147.OverlayValues[63] = d63
				ps147.OverlayValues[64] = d64
				ps147.OverlayValues[65] = d65
				ps147.OverlayValues[67] = d67
				ps147.OverlayValues[68] = d68
				ps147.OverlayValues[70] = d70
				ps147.OverlayValues[71] = d71
				ps147.OverlayValues[72] = d72
				ps147.OverlayValues[73] = d73
				ps147.OverlayValues[76] = d76
				ps147.OverlayValues[77] = d77
				ps147.OverlayValues[118] = d118
				ps147.OverlayValues[123] = d123
				ps147.OverlayValues[124] = d124
				ps147.OverlayValues[125] = d125
				ps147.OverlayValues[126] = d126
				ps147.OverlayValues[127] = d127
				ps147.OverlayValues[128] = d128
				ps147.OverlayValues[129] = d129
				ps147.OverlayValues[131] = d131
				ps147.OverlayValues[132] = d132
				ps147.OverlayValues[134] = d134
				ps147.OverlayValues[135] = d135
				ps147.OverlayValues[141] = d141
				ps147.OverlayValues[142] = d142
				ps147.OverlayValues[143] = d143
				ps147.OverlayValues[145] = d145
				ps147.OverlayValues[146] = d146
				ps147.PhiValues = make([]JITValueDesc, 2)
				d148 = d1
				ps147.PhiValues[0] = d148
				d149 = d22
				ps147.PhiValues[1] = d149
					return bbs[1].RenderPS(ps147)
				}
				if !ps.General {
					ps.General = true
					return bbs[5].RenderPS(ps)
				}
				lbl12 := ctx.ReserveLabel()
				lbl13 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d143.Reg, 0)
				ctx.EmitJcc(CcNE, lbl12)
				ctx.EmitJmp(lbl13)
				ctx.MarkLabel(lbl12)
				ctx.EmitJmp(lbl5)
				ctx.MarkLabel(lbl13)
				ctx.EnsureDesc(&d1)
				if d1.Loc == LocReg {
					ctx.ProtectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.ProtectReg(d1.Reg)
					ctx.ProtectReg(d1.Reg2)
				}
				ctx.EnsureDesc(&d22)
				if d22.Loc == LocReg {
					ctx.ProtectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.ProtectReg(d22.Reg)
					ctx.ProtectReg(d22.Reg2)
				}
				d150 = d1
				if d150.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d150)
				if d150.Loc != LocRegTriple { panic("jit: slice phi source is not a triple") }
				ctx.EmitStoreRegMem(d150.Reg, RegRSP, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreRegMem(d150.Reg2, RegRSP, int32(bbs[1].PhiBase)+int32(0)+8)
				ctx.EmitStoreRegMem(d150.Reg3, RegRSP, int32(bbs[1].PhiBase)+int32(0)+16)
				d151 = d22
				if d151.Loc == LocNone { panic("jit: phi source has no location") }
				ctx.EnsureDesc(&d151)
				ctx.EmitStoreToStack(d151, int32(bbs[1].PhiBase)+int32(24))
				if d1.Loc == LocReg {
					ctx.UnprotectReg(d1.Reg)
				} else if d1.Loc == LocRegPair {
					ctx.UnprotectReg(d1.Reg)
					ctx.UnprotectReg(d1.Reg2)
				}
				if d22.Loc == LocReg {
					ctx.UnprotectReg(d22.Reg)
				} else if d22.Loc == LocRegPair {
					ctx.UnprotectReg(d22.Reg)
					ctx.UnprotectReg(d22.Reg2)
				}
				ctx.EmitJmp(lbl2)
				ps152 := PhiState{General: true}
				ps152.OverlayValues = make([]JITValueDesc, 152)
				ps152.OverlayValues[1] = d1
				ps152.OverlayValues[2] = d2
				ps152.OverlayValues[3] = d3
				ps152.OverlayValues[4] = d4
				ps152.OverlayValues[5] = d5
				ps152.OverlayValues[6] = d6
				ps152.OverlayValues[7] = d7
				ps152.OverlayValues[8] = d8
				ps152.OverlayValues[9] = d9
				ps152.OverlayValues[10] = d10
				ps152.OverlayValues[11] = d11
				ps152.OverlayValues[12] = d12
				ps152.OverlayValues[13] = d13
				ps152.OverlayValues[15] = d15
				ps152.OverlayValues[16] = d16
				ps152.OverlayValues[17] = d17
				ps152.OverlayValues[18] = d18
				ps152.OverlayValues[22] = d22
				ps152.OverlayValues[23] = d23
				ps152.OverlayValues[24] = d24
				ps152.OverlayValues[27] = d27
				ps152.OverlayValues[28] = d28
				ps152.OverlayValues[54] = d54
				ps152.OverlayValues[56] = d56
				ps152.OverlayValues[57] = d57
				ps152.OverlayValues[63] = d63
				ps152.OverlayValues[64] = d64
				ps152.OverlayValues[65] = d65
				ps152.OverlayValues[67] = d67
				ps152.OverlayValues[68] = d68
				ps152.OverlayValues[70] = d70
				ps152.OverlayValues[71] = d71
				ps152.OverlayValues[72] = d72
				ps152.OverlayValues[73] = d73
				ps152.OverlayValues[76] = d76
				ps152.OverlayValues[77] = d77
				ps152.OverlayValues[118] = d118
				ps152.OverlayValues[123] = d123
				ps152.OverlayValues[124] = d124
				ps152.OverlayValues[125] = d125
				ps152.OverlayValues[126] = d126
				ps152.OverlayValues[127] = d127
				ps152.OverlayValues[128] = d128
				ps152.OverlayValues[129] = d129
				ps152.OverlayValues[131] = d131
				ps152.OverlayValues[132] = d132
				ps152.OverlayValues[134] = d134
				ps152.OverlayValues[135] = d135
				ps152.OverlayValues[141] = d141
				ps152.OverlayValues[142] = d142
				ps152.OverlayValues[143] = d143
				ps152.OverlayValues[145] = d145
				ps152.OverlayValues[146] = d146
				ps152.OverlayValues[148] = d148
				ps152.OverlayValues[149] = d149
				ps152.OverlayValues[150] = d150
				ps152.OverlayValues[151] = d151
				ps153 := PhiState{General: true}
				ps153.OverlayValues = make([]JITValueDesc, 152)
				ps153.OverlayValues[1] = d1
				ps153.OverlayValues[2] = d2
				ps153.OverlayValues[3] = d3
				ps153.OverlayValues[4] = d4
				ps153.OverlayValues[5] = d5
				ps153.OverlayValues[6] = d6
				ps153.OverlayValues[7] = d7
				ps153.OverlayValues[8] = d8
				ps153.OverlayValues[9] = d9
				ps153.OverlayValues[10] = d10
				ps153.OverlayValues[11] = d11
				ps153.OverlayValues[12] = d12
				ps153.OverlayValues[13] = d13
				ps153.OverlayValues[15] = d15
				ps153.OverlayValues[16] = d16
				ps153.OverlayValues[17] = d17
				ps153.OverlayValues[18] = d18
				ps153.OverlayValues[22] = d22
				ps153.OverlayValues[23] = d23
				ps153.OverlayValues[24] = d24
				ps153.OverlayValues[27] = d27
				ps153.OverlayValues[28] = d28
				ps153.OverlayValues[54] = d54
				ps153.OverlayValues[56] = d56
				ps153.OverlayValues[57] = d57
				ps153.OverlayValues[63] = d63
				ps153.OverlayValues[64] = d64
				ps153.OverlayValues[65] = d65
				ps153.OverlayValues[67] = d67
				ps153.OverlayValues[68] = d68
				ps153.OverlayValues[70] = d70
				ps153.OverlayValues[71] = d71
				ps153.OverlayValues[72] = d72
				ps153.OverlayValues[73] = d73
				ps153.OverlayValues[76] = d76
				ps153.OverlayValues[77] = d77
				ps153.OverlayValues[118] = d118
				ps153.OverlayValues[123] = d123
				ps153.OverlayValues[124] = d124
				ps153.OverlayValues[125] = d125
				ps153.OverlayValues[126] = d126
				ps153.OverlayValues[127] = d127
				ps153.OverlayValues[128] = d128
				ps153.OverlayValues[129] = d129
				ps153.OverlayValues[131] = d131
				ps153.OverlayValues[132] = d132
				ps153.OverlayValues[134] = d134
				ps153.OverlayValues[135] = d135
				ps153.OverlayValues[141] = d141
				ps153.OverlayValues[142] = d142
				ps153.OverlayValues[143] = d143
				ps153.OverlayValues[145] = d145
				ps153.OverlayValues[146] = d146
				ps153.OverlayValues[148] = d148
				ps153.OverlayValues[149] = d149
				ps153.OverlayValues[150] = d150
				ps153.OverlayValues[151] = d151
				ps153.PhiValues = make([]JITValueDesc, 2)
				d154 = d1
				ps153.PhiValues[0] = d154
				d155 = d22
				ps153.PhiValues[1] = d155
				snap156 := d1
				snap157 := d2
				snap158 := d3
				snap159 := d4
				snap160 := d5
				snap161 := d6
				snap162 := d7
				snap163 := d8
				snap164 := d9
				snap165 := d10
				snap166 := d11
				snap167 := d12
				snap168 := d13
				snap169 := d15
				snap170 := d16
				snap171 := d17
				snap172 := d18
				snap173 := d22
				snap174 := d23
				snap175 := d24
				snap176 := d27
				snap177 := d28
				snap178 := d54
				snap179 := d56
				snap180 := d57
				snap181 := d63
				snap182 := d64
				snap183 := d65
				snap184 := d67
				snap185 := d68
				snap186 := d70
				snap187 := d71
				snap188 := d72
				snap189 := d73
				snap190 := d76
				snap191 := d77
				snap192 := d118
				snap193 := d123
				snap194 := d124
				snap195 := d125
				snap196 := d126
				snap197 := d127
				snap198 := d128
				snap199 := d129
				snap200 := d131
				snap201 := d132
				snap202 := d134
				snap203 := d135
				snap204 := d141
				snap205 := d142
				snap206 := d143
				snap207 := d145
				snap208 := d146
				snap209 := d148
				snap210 := d149
				snap211 := d150
				snap212 := d151
				snap213 := d154
				snap214 := d155
				alloc215 := ctx.SnapshotAllocState()
				if !bbs[1].Rendered {
					bbs[1].RenderPS(ps153)
				}
				ctx.RestoreAllocState(alloc215)
				d1 = snap156
				d2 = snap157
				d3 = snap158
				d4 = snap159
				d5 = snap160
				d6 = snap161
				d7 = snap162
				d8 = snap163
				d9 = snap164
				d10 = snap165
				d11 = snap166
				d12 = snap167
				d13 = snap168
				d15 = snap169
				d16 = snap170
				d17 = snap171
				d18 = snap172
				d22 = snap173
				d23 = snap174
				d24 = snap175
				d27 = snap176
				d28 = snap177
				d54 = snap178
				d56 = snap179
				d57 = snap180
				d63 = snap181
				d64 = snap182
				d65 = snap183
				d67 = snap184
				d68 = snap185
				d70 = snap186
				d71 = snap187
				d72 = snap188
				d73 = snap189
				d76 = snap190
				d77 = snap191
				d118 = snap192
				d123 = snap193
				d124 = snap194
				d125 = snap195
				d126 = snap196
				d127 = snap197
				d128 = snap198
				d129 = snap199
				d131 = snap200
				d132 = snap201
				d134 = snap202
				d135 = snap203
				d141 = snap204
				d142 = snap205
				d143 = snap206
				d145 = snap207
				d146 = snap208
				d148 = snap209
				d149 = snap210
				d150 = snap211
				d151 = snap212
				d154 = snap213
				d155 = snap214
				if !bbs[4].Rendered {
					return bbs[4].RenderPS(ps152)
				}
				return result
				ctx.FreeDesc(&d141)
				return result
				}
				argPinned216 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned216 = append(argPinned216, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned216 = append(argPinned216, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned216 = append(argPinned216, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned216 = append(argPinned216, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned216 {
						ctx.UnprotectReg(r)
					}
				}()
				ps217 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps217)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(40))
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cons_map",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[1], "cons_map")
			mapper := OptimizeProcToSerialFunction(a[2])
			result := make([]Scmer, len(input)+1)
			result[0] = a[0]
			for i, item := range input {
				result[i+1] = mapper(item)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "constructs a list head while mapping its tail (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "head"},
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d21 JITValueDesc
				_ = d21
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[1]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.FreeDesc(&d2)
					d4 = args[2]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else {
						d5 = ctx.RequestOptimizedCallback(2)
					}
					ctx.FreeDesc(&d4)
					var d6 JITValueDesc
					if d3.SliceSizeKnown {
						d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					var d7 JITValueDesc
					if d6.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitMovRegReg(scratch, d6.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d7)
					}
					if d7.Loc == LocReg && d6.Loc == LocReg && d7.Reg == d6.Reg {
						ctx.TransferReg(d6.Reg)
						d6.Loc = LocNone
					}
					ctx.FreeDesc(&d6)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					d8 = ctx.EmitGoCallScalar(GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d7, d7}, 3)
					ctx.BindReg(d8.Reg, &d8)
					ctx.BindReg(d8.Reg2, &d8)
					ctx.BindReg(d8.Reg3, &d8)
					ctx.FreeDesc(&d7)
					d9 = args[0]
					d9.ID = 0
					d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d9)
					d11 = ctx.EmitSliceElementAddress(&d8, &d10, int32(16))
					ctx.EmitStoreScmerAt(&d11, &d9)
					ctx.FreeDesc(&d11)
					ctx.FreeDesc(&d9)
					var d12 JITValueDesc
					if d3.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					ps13 := PhiState{General: ps.General}
					ps13.OverlayValues = make([]JITValueDesc, 13)
					ps13.OverlayValues[1] = d1
					ps13.OverlayValues[2] = d2
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[4] = d4
					ps13.OverlayValues[5] = d5
					ps13.OverlayValues[6] = d6
					ps13.OverlayValues[7] = d7
					ps13.OverlayValues[8] = d8
					ps13.OverlayValues[9] = d9
					ps13.OverlayValues[10] = d10
					ps13.OverlayValues[11] = d11
					ps13.OverlayValues[12] = d12
					ps13.PhiValues = make([]JITValueDesc, 1)
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps13.PhiValues[0] = d14
					if ps13.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps13)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d15 := ps.PhiValues[0]
							ctx.EnsureDesc(&d15)
							ctx.EmitStoreToStack(d15, int32(bbs[1].PhiBase)+int32(0))
						}
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d16 JITValueDesc
					if d1.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d16)
					}
					if d16.Loc == LocReg && d1.Loc == LocReg && d16.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d12)
					var d17 JITValueDesc
					if d16.Loc == LocImm && d12.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() < d12.Imm.Int())}
					} else if d12.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d16.Reg)
						if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d16.Reg, int32(d12.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
							ctx.EmitCmpInt64(d16.Reg, RegR11)
						}
						ctx.EmitSetcc(r0, CcL)
						d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d17)
					} else if d16.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d12.Reg)
						ctx.EmitSetcc(r1, CcL)
						d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d17)
					} else {
						r2 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpInt64(d16.Reg, d12.Reg)
						ctx.EmitSetcc(r2, CcL)
						d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d17)
					}
					ctx.FreeDesc(&d12)
					d18 = d17
					ctx.EnsureDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							ps19 := PhiState{General: ps.General}
							ps19.OverlayValues = make([]JITValueDesc, 19)
							ps19.OverlayValues[1] = d1
							ps19.OverlayValues[2] = d2
							ps19.OverlayValues[3] = d3
							ps19.OverlayValues[4] = d4
							ps19.OverlayValues[5] = d5
							ps19.OverlayValues[6] = d6
							ps19.OverlayValues[7] = d7
							ps19.OverlayValues[8] = d8
							ps19.OverlayValues[9] = d9
							ps19.OverlayValues[10] = d10
							ps19.OverlayValues[11] = d11
							ps19.OverlayValues[12] = d12
							ps19.OverlayValues[14] = d14
							ps19.OverlayValues[15] = d15
							ps19.OverlayValues[16] = d16
							ps19.OverlayValues[17] = d17
							ps19.OverlayValues[18] = d18
							return bbs[2].RenderPS(ps19)
						}
						ps20 := PhiState{General: ps.General}
						ps20.OverlayValues = make([]JITValueDesc, 19)
						ps20.OverlayValues[1] = d1
						ps20.OverlayValues[2] = d2
						ps20.OverlayValues[3] = d3
						ps20.OverlayValues[4] = d4
						ps20.OverlayValues[5] = d5
						ps20.OverlayValues[6] = d6
						ps20.OverlayValues[7] = d7
						ps20.OverlayValues[8] = d8
						ps20.OverlayValues[9] = d9
						ps20.OverlayValues[10] = d10
						ps20.OverlayValues[11] = d11
						ps20.OverlayValues[12] = d12
						ps20.OverlayValues[14] = d14
						ps20.OverlayValues[15] = d15
						ps20.OverlayValues[16] = d16
						ps20.OverlayValues[17] = d17
						ps20.OverlayValues[18] = d18
						return bbs[3].RenderPS(ps20)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d21 := ps.PhiValues[0]
							ctx.EnsureDesc(&d21)
							ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d18.Reg, 0)
					ctx.EmitJcc(CcNE, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ps22 := PhiState{General: true}
					ps22.OverlayValues = make([]JITValueDesc, 22)
					ps22.OverlayValues[1] = d1
					ps22.OverlayValues[2] = d2
					ps22.OverlayValues[3] = d3
					ps22.OverlayValues[4] = d4
					ps22.OverlayValues[5] = d5
					ps22.OverlayValues[6] = d6
					ps22.OverlayValues[7] = d7
					ps22.OverlayValues[8] = d8
					ps22.OverlayValues[9] = d9
					ps22.OverlayValues[10] = d10
					ps22.OverlayValues[11] = d11
					ps22.OverlayValues[12] = d12
					ps22.OverlayValues[14] = d14
					ps22.OverlayValues[15] = d15
					ps22.OverlayValues[16] = d16
					ps22.OverlayValues[17] = d17
					ps22.OverlayValues[18] = d18
					ps22.OverlayValues[21] = d21
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 22)
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[2] = d2
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[5] = d5
					ps23.OverlayValues[6] = d6
					ps23.OverlayValues[7] = d7
					ps23.OverlayValues[8] = d8
					ps23.OverlayValues[9] = d9
					ps23.OverlayValues[10] = d10
					ps23.OverlayValues[11] = d11
					ps23.OverlayValues[12] = d12
					ps23.OverlayValues[14] = d14
					ps23.OverlayValues[15] = d15
					ps23.OverlayValues[16] = d16
					ps23.OverlayValues[17] = d17
					ps23.OverlayValues[18] = d18
					ps23.OverlayValues[21] = d21
					snap24 := d1
					snap25 := d2
					snap26 := d3
					snap27 := d4
					snap28 := d5
					snap29 := d6
					snap30 := d7
					snap31 := d8
					snap32 := d9
					snap33 := d10
					snap34 := d11
					snap35 := d12
					snap36 := d14
					snap37 := d15
					snap38 := d16
					snap39 := d17
					snap40 := d18
					snap41 := d21
					alloc42 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps23)
					}
					ctx.RestoreAllocState(alloc42)
					d1 = snap24
					d2 = snap25
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d6 = snap29
					d7 = snap30
					d8 = snap31
					d9 = snap32
					d10 = snap33
					d11 = snap34
					d12 = snap35
					d14 = snap36
					d15 = snap37
					d16 = snap38
					d17 = snap39
					d18 = snap40
					d21 = snap41
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps22)
					}
					return result
					ctx.FreeDesc(&d17)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					r3 := ctx.AllocReg()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d3)
					if d16.Loc == LocImm {
						ctx.EmitMovRegImm64(r3, uint64(d16.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r3, d16.Reg)
						ctx.EmitShlRegImm8(r3, 4)
					}
					if d3.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
						ctx.EmitAddInt64(r3, RegR11)
					} else {
						ctx.EmitAddInt64(r3, d3.Reg)
					}
					r4 := ctx.AllocRegExcept(r3)
					r5 := ctx.AllocRegExcept(r3, r4)
					ctx.EmitMovRegMem(r4, r3, 0)
					ctx.EmitMovRegMem(r5, r3, 8)
					ctx.FreeReg(r3)
					d43 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
					ctx.BindReg(r4, &d43)
					ctx.BindReg(r5, &d43)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					var d44 JITValueDesc
					if d16.Loc == LocImm {
						d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d16.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitMovRegReg(scratch, d16.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d44)
					}
					if d44.Loc == LocReg && d16.Loc == LocReg && d44.Reg == d16.Reg {
						ctx.TransferReg(d16.Reg)
						d16.Loc = LocNone
					}
					stackArray45 := ctx.AllocStack(int32(16))
					ctx.EnsureDesc(&d43)
					ctx.EnsureDesc(&d43)
					ctx.EmitStoreScmerToStack(d43, int32(stackArray45)+int32(0))
					ctx.FreeDesc(&d43)
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					r8 := ctx.AllocRegExcept(r6, r7)
					ctx.EmitLeaRegMem(r6, RegRSP, int32(stackArray45))
					ctx.EmitMovRegImm64(r7, uint64(1))
					ctx.EmitMovRegImm64(r8, uint64(1))
					d46 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					ctx.BindReg(r6, &d46)
					ctx.BindReg(r7, &d46)
					ctx.BindReg(r8, &d46)
					callbackArgs48 := make([]JITValueDesc, 1)
					callbackArgs48[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray45) + 0}
					var d47 JITValueDesc
					ctx.FreeDesc(&d46)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						outerRegs49 := ctx.PreserveOuterRegs()
						d47 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, callbackArgs48, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
						ctx.RestoreOuterRegs(outerRegs49)
					} else {
						callbackCallArgs := make([]JITValueDesc, 0, 2)
						callbackCallArgs = append(callbackCallArgs, d5)
						callbackCallArgs = append(callbackCallArgs, callbackArgs48...)
						d47 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					}
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d47)
					d50 = ctx.EmitSliceElementAddress(&d8, &d44, int32(16))
					ctx.EmitStoreScmerAt(&d50, &d47)
					ctx.FreeDesc(&d50)
					ctx.FreeDesc(&d44)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocReg {
						ctx.ProtectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.ProtectReg(d16.Reg)
						ctx.ProtectReg(d16.Reg2)
					}
					d51 = d16
					if d51.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d51)
					ctx.EmitStoreToStack(d51, int32(bbs[1].PhiBase)+int32(0))
					if d16.Loc == LocReg {
						ctx.UnprotectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.UnprotectReg(d16.Reg)
						ctx.UnprotectReg(d16.Reg2)
					}
					ps52 := PhiState{General: ps.General}
					ps52.OverlayValues = make([]JITValueDesc, 52)
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[8] = d8
					ps52.OverlayValues[9] = d9
					ps52.OverlayValues[10] = d10
					ps52.OverlayValues[11] = d11
					ps52.OverlayValues[12] = d12
					ps52.OverlayValues[14] = d14
					ps52.OverlayValues[15] = d15
					ps52.OverlayValues[16] = d16
					ps52.OverlayValues[17] = d17
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[43] = d43
					ps52.OverlayValues[44] = d44
					ps52.OverlayValues[46] = d46
					ps52.OverlayValues[47] = d47
					ps52.OverlayValues[50] = d50
					ps52.OverlayValues[51] = d51
					ps52.PhiValues = make([]JITValueDesc, 1)
					d53 = d16
					ps52.PhiValues[0] = d53
					if ps52.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps52)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					ctx.ReclaimUntrackedRegs()
					d54 = ctx.EmitNewSliceFromGoSlice(&d8)
					ctx.EnsureDesc(&d54)
					if d54.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d54, &result)
						result.Type = d54.Type
					} else {
						switch d54.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d54)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d54)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d54)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d54, &result)
							result.Type = d54.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned55 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned55 = append(argPinned55, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned55 = append(argPinned55, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned55 = append(argPinned55, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned55 = append(argPinned55, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned55 {
						ctx.UnprotectReg(r)
					}
				}()
				ps56 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps56)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "flat_map",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "flat_map")
			mapper := OptimizeProcToSerialFunction(a[1])
			width := int(ToInt(a[2]))
			result := make([]Scmer, 0, len(input)*width)
			for _, item := range input {
				result = append(result, asSlice(mapper(item), "flat_map result")...)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused fixed-width serial map and flatten (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "list"}},
				{Kind: "int", Label: "width"},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "flat_map_unique",

		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "flat_map_unique")
			mapper := PrepareSerialProc(a[1])
			var mapperArgs [1]Scmer
			builder := orderedUniqueBuilder{}
			for _, item := range input {
				mapperArgs[0] = item
				for _, mapped := range asSlice(mapper.Call(mapperArgs[:]), "flat_map_unique result") {
					builder.add(mapped)
				}
			}
			return NewSlice(builder.result())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "fused serial map, flatten, and stable unique collection (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "list"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	// _mut variants: optimizer-only, forbidden from .scm code
	// Tier 1: same-length, zero-copy

	Declare(&Globalenv, &Declaration{
		Name: "map_mut",

		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(v)
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place map (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned list to map in-place"},
				{Kind: "func", Label: "map", Description: "map function", Params: []*TypeDescriptor{{Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d12 JITValueDesc
				_ = d12
				var d15 JITValueDesc
				_ = d15
				var d31 JITValueDesc
				_ = d31
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
							ps.General = true
							return bbs[0].RenderPS(ps)
						}
					}
					bbs[0].VisitCount++
					if ps.General {
						if bbs[0].Rendered {
							ctx.EmitJmp(lbl1)
							return result
						}
						bbs[0].Rendered = true
						bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_0 = bbs[0].Address
						ctx.MarkLabel(lbl1)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					d3 = jitKnownSliceHeader(ctx, &d2)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else {
						d5 = ctx.RequestOptimizedCallback(1)
					}
					ctx.FreeDesc(&d4)
					var d6 JITValueDesc
					if d3.SliceSizeKnown {
						d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d6 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					ps7 := PhiState{General: ps.General}
					ps7.OverlayValues = make([]JITValueDesc, 7)
					ps7.OverlayValues[1] = d1
					ps7.OverlayValues[2] = d2
					ps7.OverlayValues[3] = d3
					ps7.OverlayValues[4] = d4
					ps7.OverlayValues[5] = d5
					ps7.OverlayValues[6] = d6
					ps7.PhiValues = make([]JITValueDesc, 1)
					d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps7.PhiValues[0] = d8
					if ps7.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps7)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d9 := ps.PhiValues[0]
							ctx.EnsureDesc(&d9)
							ctx.EmitStoreToStack(d9, int32(bbs[1].PhiBase)+int32(0))
						}
						if bbs[1].VisitCount >= 0 {
							ps.General = true
							return bbs[1].RenderPS(ps)
						}
					}
					bbs[1].VisitCount++
					if ps.General {
						if bbs[1].Rendered {
							ctx.EmitJmp(lbl2)
							return result
						}
						bbs[1].Rendered = true
						bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_1 = bbs[1].Address
						ctx.MarkLabel(lbl2)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d10 JITValueDesc
					if d1.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d10)
					}
					if d10.Loc == LocReg && d1.Loc == LocReg && d10.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d6)
					var d11 JITValueDesc
					if d10.Loc == LocImm && d6.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() < d6.Imm.Int())}
					} else if d6.Loc == LocImm {
						r0 := ctx.AllocRegExcept(d10.Reg)
						if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d10.Reg, int32(d6.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
							ctx.EmitCmpInt64(d10.Reg, RegR11)
						}
						ctx.EmitSetcc(r0, CcL)
						d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d11)
					} else if d10.Loc == LocImm {
						r1 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d6.Reg)
						ctx.EmitSetcc(r1, CcL)
						d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						r2 := ctx.AllocRegExcept(d10.Reg)
						ctx.EmitCmpInt64(d10.Reg, d6.Reg)
						ctx.EmitSetcc(r2, CcL)
						d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d11)
					}
					ctx.FreeDesc(&d6)
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							ps13 := PhiState{General: ps.General}
							ps13.OverlayValues = make([]JITValueDesc, 13)
							ps13.OverlayValues[1] = d1
							ps13.OverlayValues[2] = d2
							ps13.OverlayValues[3] = d3
							ps13.OverlayValues[4] = d4
							ps13.OverlayValues[5] = d5
							ps13.OverlayValues[6] = d6
							ps13.OverlayValues[8] = d8
							ps13.OverlayValues[9] = d9
							ps13.OverlayValues[10] = d10
							ps13.OverlayValues[11] = d11
							ps13.OverlayValues[12] = d12
							return bbs[2].RenderPS(ps13)
						}
						ps14 := PhiState{General: ps.General}
						ps14.OverlayValues = make([]JITValueDesc, 13)
						ps14.OverlayValues[1] = d1
						ps14.OverlayValues[2] = d2
						ps14.OverlayValues[3] = d3
						ps14.OverlayValues[4] = d4
						ps14.OverlayValues[5] = d5
						ps14.OverlayValues[6] = d6
						ps14.OverlayValues[8] = d8
						ps14.OverlayValues[9] = d9
						ps14.OverlayValues[10] = d10
						ps14.OverlayValues[11] = d11
						ps14.OverlayValues[12] = d12
						return bbs[3].RenderPS(ps14)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d15 := ps.PhiValues[0]
							ctx.EnsureDesc(&d15)
							ctx.EmitStoreToStack(d15, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					lbl5 := ctx.ReserveLabel()
					lbl6 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					ctx.EmitJcc(CcNE, lbl5)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl5)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl4)
					ps16 := PhiState{General: true}
					ps16.OverlayValues = make([]JITValueDesc, 16)
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[2] = d2
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[5] = d5
					ps16.OverlayValues[6] = d6
					ps16.OverlayValues[8] = d8
					ps16.OverlayValues[9] = d9
					ps16.OverlayValues[10] = d10
					ps16.OverlayValues[11] = d11
					ps16.OverlayValues[12] = d12
					ps16.OverlayValues[15] = d15
					ps17 := PhiState{General: true}
					ps17.OverlayValues = make([]JITValueDesc, 16)
					ps17.OverlayValues[1] = d1
					ps17.OverlayValues[2] = d2
					ps17.OverlayValues[3] = d3
					ps17.OverlayValues[4] = d4
					ps17.OverlayValues[5] = d5
					ps17.OverlayValues[6] = d6
					ps17.OverlayValues[8] = d8
					ps17.OverlayValues[9] = d9
					ps17.OverlayValues[10] = d10
					ps17.OverlayValues[11] = d11
					ps17.OverlayValues[12] = d12
					ps17.OverlayValues[15] = d15
					snap18 := d1
					snap19 := d2
					snap20 := d3
					snap21 := d4
					snap22 := d5
					snap23 := d6
					snap24 := d8
					snap25 := d9
					snap26 := d10
					snap27 := d11
					snap28 := d12
					snap29 := d15
					alloc30 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps17)
					}
					ctx.RestoreAllocState(alloc30)
					d1 = snap18
					d2 = snap19
					d3 = snap20
					d4 = snap21
					d5 = snap22
					d6 = snap23
					d8 = snap24
					d9 = snap25
					d10 = snap26
					d11 = snap27
					d12 = snap28
					d15 = snap29
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps16)
					}
					return result
					ctx.FreeDesc(&d11)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
							ps.General = true
							return bbs[2].RenderPS(ps)
						}
					}
					bbs[2].VisitCount++
					if ps.General {
						if bbs[2].Rendered {
							ctx.EmitJmp(lbl3)
							return result
						}
						bbs[2].Rendered = true
						bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_2 = bbs[2].Address
						ctx.MarkLabel(lbl3)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d10)
					r3 := ctx.AllocReg()
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d3)
					if d10.Loc == LocImm {
						ctx.EmitMovRegImm64(r3, uint64(d10.Imm.Int())*16)
					} else {
						ctx.EmitMovRegReg(r3, d10.Reg)
						ctx.EmitShlRegImm8(r3, 4)
					}
					if d3.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
						ctx.EmitAddInt64(r3, RegR11)
					} else {
						ctx.EmitAddInt64(r3, d3.Reg)
					}
					r4 := ctx.AllocRegExcept(r3)
					r5 := ctx.AllocRegExcept(r3, r4)
					ctx.EmitMovRegMem(r4, r3, 0)
					ctx.EmitMovRegMem(r5, r3, 8)
					ctx.FreeReg(r3)
					d31 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
					ctx.BindReg(r4, &d31)
					ctx.BindReg(r5, &d31)
					stackArray32 := ctx.AllocStack(int32(16))
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d31)
					ctx.EmitStoreScmerToStack(d31, int32(stackArray32)+int32(0))
					ctx.FreeDesc(&d31)
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					r8 := ctx.AllocRegExcept(r6, r7)
					ctx.EmitLeaRegMem(r6, RegRSP, int32(stackArray32))
					ctx.EmitMovRegImm64(r7, uint64(1))
					ctx.EmitMovRegImm64(r8, uint64(1))
					d33 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					ctx.BindReg(r6, &d33)
					ctx.BindReg(r7, &d33)
					ctx.BindReg(r8, &d33)
					callbackArgs35 := make([]JITValueDesc, 1)
					callbackArgs35[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray32) + 0}
					var d34 JITValueDesc
					ctx.FreeDesc(&d33)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						outerRegs36 := ctx.PreserveOuterRegs()
						d34 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, callbackArgs35, ctx.SliceBase, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
						ctx.RestoreOuterRegs(outerRegs36)
					} else {
						callbackCallArgs := make([]JITValueDesc, 0, 2)
						callbackCallArgs = append(callbackCallArgs, d5)
						callbackCallArgs = append(callbackCallArgs, callbackArgs35...)
						d34 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
					}
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d34)
					d37 = ctx.EmitSliceElementAddress(&d3, &d10, int32(16))
					ctx.EmitStoreScmerAt(&d37, &d34)
					ctx.FreeDesc(&d37)
					ctx.FreeDesc(&d34)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					d38 = d10
					if d38.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d38)
					ctx.EmitStoreToStack(d38, int32(bbs[1].PhiBase)+int32(0))
					if d10.Loc == LocReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
					ps39 := PhiState{General: ps.General}
					ps39.OverlayValues = make([]JITValueDesc, 39)
					ps39.OverlayValues[1] = d1
					ps39.OverlayValues[2] = d2
					ps39.OverlayValues[3] = d3
					ps39.OverlayValues[4] = d4
					ps39.OverlayValues[5] = d5
					ps39.OverlayValues[6] = d6
					ps39.OverlayValues[8] = d8
					ps39.OverlayValues[9] = d9
					ps39.OverlayValues[10] = d10
					ps39.OverlayValues[11] = d11
					ps39.OverlayValues[12] = d12
					ps39.OverlayValues[15] = d15
					ps39.OverlayValues[31] = d31
					ps39.OverlayValues[33] = d33
					ps39.OverlayValues[34] = d34
					ps39.OverlayValues[37] = d37
					ps39.OverlayValues[38] = d38
					ps39.PhiValues = make([]JITValueDesc, 1)
					d40 = d10
					ps39.PhiValues[0] = d40
					if ps39.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps39)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
							ps.General = true
							return bbs[3].RenderPS(ps)
						}
					}
					bbs[3].VisitCount++
					if ps.General {
						if bbs[3].Rendered {
							ctx.EmitJmp(lbl4)
							return result
						}
						bbs[3].Rendered = true
						bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_3 = bbs[3].Address
						ctx.MarkLabel(lbl4)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					ctx.ReclaimUntrackedRegs()
					d41 = ctx.EmitNewSliceFromGoSlice(&d3)
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d41, &result)
						result.Type = d41.Type
					} else {
						switch d41.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d41)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d41)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d41)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d41, &result)
							result.Type = d41.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				argPinned42 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned42 = append(argPinned42, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned42 = append(argPinned42, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned42 = append(argPinned42, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned42 = append(argPinned42, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned42 {
						ctx.UnprotectReg(r)
					}
				}()
				ps43 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps43)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "mapIndex_mut",

		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place mapIndex (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned list to map in-place"},
				{Kind: "func", Label: "map", Description: "transforms each item with its index", Params: []*TypeDescriptor{{Kind: "int", Label: "index", Description: "zero-based item index"}, {Kind: "any", Label: "item", Description: "current list item"}}, Return: &TypeDescriptor{Kind: "any", Label: "mapped_item", Description: "transformed item"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "map_assoc_mut",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "map_assoc_mut"); fd == nil {
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						key = v
					} else {
						slice[i] = fn(key, v)
					}
				}
				return NewSlice(slice)
			} else {
				// FastDict path: cannot mutate in-place, fall back to allocating
				result := make([]Scmer, 0, len(fd.Pairs))
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, k, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place map_assoc (optimizer-only, slice path only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "owned dictionary to map in-place"},
				{Kind: "func", Label: "map", Description: "transforms each dictionary value", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "any", Label: "mapped_value", Description: "replacement value"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	// Tier 2: shrinking, write-cursor

	Declare(&Globalenv, &Declaration{
		Name: "filter_mut",

		Fn: func(a ...Scmer) Scmer {
			input := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			w := 0
			for _, v := range input {
				if fn(v).Bool() {
					input[w] = v
					w++
				}
			}
			return NewSlice(input[:w])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place filter (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned list to filter in-place"},
				{Kind: "func", Label: "condition", Description: "returns whether an item should be included", Params: []*TypeDescriptor{{Kind: "any", Label: "item", Description: "current list item"}}, Return: &TypeDescriptor{Kind: "bool", Label: "included", Description: "whether to include the item"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "reverse_mut",

		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place reverse (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned list to reverse in-place"},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "filter_assoc_mut",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "filter_assoc_mut"); fd == nil {
				w := 0
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						slice[w] = slice[i]
						slice[w+1] = slice[i+1]
						w += 2
					}
				}
				return NewSlice(slice[:w])
			} else {
				result := make([]Scmer, 0)
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = append(result, k, v)
					}
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place filter_assoc (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "owned dictionary to filter in-place"},
				{Kind: "func", Label: "condition", Description: "returns whether a dictionary entry should be included", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "bool", Label: "included", Description: "whether to include the entry"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "extract_assoc_mut",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "extract_assoc_mut"); fd == nil {
				w := 0
				for i := 0; i < len(slice); i += 2 {
					slice[w] = fn(slice[i], slice[i+1])
					w++
				}
				return NewSlice(slice[:w])
			} else {
				result := make([]Scmer, 0, len(fd.Pairs)/2)
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place extract_assoc (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "owned dictionary to extract from in-place"},
				{Kind: "func", Label: "map", Description: "extracts one element per dictionary entry", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "entry key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "any", Label: "element", Description: "element extracted from the entry"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "set_assoc_mut",

		Fn: func(a ...Scmer) Scmer {
			var mergeFn func(Scmer, Scmer) Scmer
			if len(a) > 3 {
				mfn := OptimizeProcToSerialFunction(a[3])
				mergeFn = func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
			}
			slice, fd := asAssoc(a[0], "set_assoc_mut")
			if fd == nil {
				// Small associations are slice-backed and may originate from a
				// reducer neutral shared by parallel shards. Copy the bounded
				// representation before changing or extending it; once promoted
				// to a FastDict, ownership is exclusive and updates stay in place.
				list := append([]Scmer(nil), slice...)
				for i := 0; i < len(list); i += 2 {
					if Equal(list[i], a[1]) {
						if mergeFn != nil {
							list[i+1] = mergeFn(list[i+1], a[2])
						} else {
							list[i+1] = a[2]
						}
						return NewSlice(list)
					}
				}
				list = append(list, a[1], a[2])
				if len(list) >= 10 {
					fd := NewFastDictValue(len(list)/2 + 4)
					for i := 0; i < len(list); i += 2 {
						fd.Set(list[i], list[i+1], nil)
					}
					return NewFastDict(fd)
				}
				return NewSlice(list)
			} else {
				fd.Set(a[1], a[2], mergeFn)
				return NewFastDict(fd)
			}
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place set_assoc (optimizer-only, mutates input directly)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "owned dictionary to mutate"},
				{Kind: "string", Label: "key", Description: "key to set"},
				{Kind: "any", Label: "value", Description: "new value"},
				{Kind: "func", Label: "merge", Description: "(optional) merge function", Optional: true, Params: []*TypeDescriptor{{Kind: "any", Label: "old"}, {Kind: "any", Label: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	// Tier 3: append/grow

	Declare(&Globalenv, &Declaration{
		Name: "append_mut",

		Fn: func(a ...Scmer) Scmer {
			base := asSlice(a[0], "append_mut")
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place append (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned base list"},
				{Kind: "any", Label: "item...", Description: "items to add", Variadic: true},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "append_unique_mut",

		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "append_unique_mut")
			for _, el := range a[1:] {
				for _, el2 := range list {
					if Equal(el, el2) {
						goto skipItem
					}
				}
				list = append(list, el)
			skipItem:
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place append_unique (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned base list"},
				{Kind: "any", Label: "item...", Description: "items to add", Variadic: true},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "merge_unique_mut",

		Fn: func(a ...Scmer) Scmer {
			if len(a) == 1 {
				lists := asSlice(a[0], "merge_unique_mut")
				inputs := append([]Scmer{}, lists...)
				result := lists[:0]
				for _, v := range inputs {
					for _, el := range asSlice(v, "merge_unique_mut item") {
						duplicate := false
						for _, existing := range result {
							if Equal(el, existing) {
								duplicate = true
								break
							}
						}
						if !duplicate {
							result = append(result, el)
						}
					}
				}
				return NewSlice(result)
			}
			base := asSlice(a[0], "merge_unique_mut")
			inputs := append([]Scmer{}, base...)
			result := base[:0]
			for _, el := range inputs {
				duplicate := false
				for _, existing := range result {
					if Equal(el, existing) {
						duplicate = true
						break
					}
				}
				if !duplicate {
					result = append(result, el)
				}
			}
			for _, v := range a[1:] {
				for _, el := range asSlice(v, "merge_unique_mut item") {
					duplicate := false
					for _, existing := range result {
						if Equal(el, existing) {
							duplicate = true
							break
						}
					}
					if !duplicate {
						result = append(result, el)
					}
				}
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place merge_unique (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned base list or owned list of lists", Variadic: true},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "reset_mut",

		Fn: func(a ...Scmer) Scmer {
			base := asSlice(a[0], "reset_mut")
			if base == nil {
				return NewSlice([]Scmer{})
			}
			return NewSlice(base[:0:cap(base)])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "resets an owned list to len=0 while preserving capacity",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", Description: "owned base list"},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "merge_assoc_mut",

		Fn: func(a ...Scmer) Scmer {
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc_mut"])
			dst := a[0]
			if slice, fd := asAssoc(a[1], "merge_assoc_mut"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if len(a) > 2 {
						dst = setAssoc(dst, slice[i], slice[i+1], a[2])
					} else {
						dst = setAssoc(dst, slice[i], slice[i+1])
					}
				}
			} else {
				if len(a) > 2 {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v, a[2]); return true })
				} else {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v); return true })
				}
			}
			return dst
		},
		Type: &TypeDescriptor{Kind: "func", Description: "in-place merge_assoc (optimizer-only)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict1", Description: "owned first dictionary"},
				{Kind: "list", Label: "dict2", Description: "dictionary with new values"},
				{Kind: "func", Label: "merge", Description: "(optional) merge function", Optional: true, Params: []*TypeDescriptor{{Kind: "any", Label: "old"}, {Kind: "any", Label: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sort",

		Fn: func(a ...Scmer) Scmer {
			src := a[0].Slice()
			cmp := a[1]
			dst := make([]Scmer, len(src))
			copy(dst, src)
			hybridsort.SliceStable(dst, func(i, j int) bool {
				return ToBool(Apply(cmp, dst[i], dst[j]))
			})
			return NewSlice(dst)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a sorted copy of a list using a comparator (lambda (a b) truthy/falsy)",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "comparator", Params: []*TypeDescriptor{{Kind: "any"}, {Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:                   FreshAlloc,
			Const:                    true,
			Optimize:                 FirstParameterMutable("sort_mut"),
			OptimizeFirstArgTransfer: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sort_mut",

		Fn: func(a ...Scmer) Scmer {
			src := a[0].Slice()
			cmp := a[1]
			hybridsort.SliceStable(src, func(i, j int) bool {
				return ToBool(Apply(cmp, src[i], src[j]))
			})
			return a[0]
		},
		Type: &TypeDescriptor{Kind: "func", Description: "sorts a list in-place using a comparator (lambda (a b) truthy/falsy)",
			HasSideEffects: true,
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list"},
				{Kind: "func", Label: "comparator", Params: []*TypeDescriptor{{Kind: "any"}, {Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return: &TypeDescriptor{Kind: "list"},

			JITEmit: nil,
		},
	})
}

func optimizeAppend(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "append_mut")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 2 {
		return result, td
	}
	baseLength := exactOptimizedListArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1))
	if baseLength >= 0 {
		td = setOptimizedCallLength(td, baseLength+len(rv)-2)
	}
	if len(rv) > 2 {
		if base, ok := scmerSlice(rv[1]); ok && len(base) > 0 && scmerIsSymbol(base[0], "list") {
			merged := make([]Scmer, 0, len(base)+len(rv)-2)
			merged = append(merged, NewSymbol("list"))
			merged = append(merged, base[1:]...)
			merged = append(merged, rv[2:]...)
			return NewSlice(merged), descriptorWithLength(FreshAlloc, len(merged)-1)
		}
	}
	return result, td
}

// optimizeMerge keeps the standard optimization pipeline, then uses exact
// positive list lengths to annotate the flattened result and collapse direct
// list-of-items merges into a single list constructor.
func optimizeMerge(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 2 {
		return result, td
	}
	if len(rv) == 3 && optimizedArgumentType(argumentTypes, 1).Transfer() {
		if singleton, ok := optimizedSingletonListItem(rv[2]); ok {
			length := UnknownLength
			if baseLength := exactOptimizedListArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1)); baseLength >= 0 {
				length = baseLength + 1
			}
			return NewSlice([]Scmer{NewSymbol("append_mut"), rv[1], singleton}), descriptorWithLength(FreshAlloc, length)
		}
	}
	if len(rv) == 2 {
		if producer, ok := scmerSlice(rv[1]); ok && len(producer) == 3 {
			if exprMayHaveSideEffects(producer[2]) {
				return result, td
			}
			producerType := optimizedArgumentType(argumentTypes, 1)
			itemLength := UnknownLength
			if producerType.Extra != nil && producerType.Extra.Element != nil {
				itemLength = producerType.Extra.Element.Length
			}
			if itemLength >= 0 {
				switch {
				case scmerIsSymbol(producer[0], "map"), scmerIsSymbol(producer[0], "map_mut"):
					resultLength := UnknownLength
					if inputLength := exactOptimizedListArgumentLength(rv[1], producerType); inputLength >= 0 {
						resultLength = inputLength * itemLength
					}
					fused := NewSlice([]Scmer{NewSymbol("flat_map"), producer[1], producer[2], NewInt(int64(itemLength))})
					return fused, descriptorWithLength(FreshAlloc, resultLength)
				}
			}
		}
		if outer, ok := scmerSlice(rv[1]); ok && len(outer) > 0 && scmerIsSymbol(outer[0], "list") {
			merged := make([]Scmer, 0, len(outer)+1)
			merged = append(merged, NewSymbol("list"))
			allDirectListItems := true
			for _, item := range outer[1:] {
				itemSlice, ok := scmerSlice(item)
				if !ok || len(itemSlice) == 0 || !scmerIsSymbol(itemSlice[0], "list") {
					allDirectListItems = false
					break
				}
				merged = append(merged, itemSlice[1:]...)
			}
			if allDirectListItems {
				return NewSlice(merged), descriptorWithLength(FreshAlloc, len(merged)-1)
			}
		}
		if totalLength := exactFlattenedMergeArgumentLength(rv[1], optimizedArgumentType(argumentTypes, 1)); totalLength > 0 {
			return result, setOptimizedCallLength(td, totalLength)
		}
		return result, td
	}
	totalLength := 0
	allExact := len(rv) > 2
	allDirectListArgs := len(rv) > 2
	for i, arg := range rv[1:] {
		length := exactOptimizedListArgumentLength(arg, optimizedArgumentType(argumentTypes, i+1))
		if length >= 0 {
			totalLength += length
		} else {
			allExact = false
		}
		inner, ok := scmerSlice(arg)
		if !ok || len(inner) == 0 || !scmerIsSymbol(inner[0], "list") {
			allDirectListArgs = false
		}
	}
	if allDirectListArgs {
		merged := make([]Scmer, 0, totalLength+1)
		merged = append(merged, NewSymbol("list"))
		for _, arg := range rv[1:] {
			merged = append(merged, arg.Slice()[1:]...)
		}
		return NewSlice(merged), descriptorWithLength(FreshAlloc, len(merged)-1)
	}
	if allExact {
		return result, setOptimizedCallLength(td, totalLength)
	}
	return result, td
}

// optimizedSingletonListItem extracts the item from an already optimized
// one-element list constructor. The !list form is the frame-local equivalent
// emitted for a NoEscape merge argument.
func optimizedSingletonListItem(expr Scmer) (Scmer, bool) {
	items, ok := scmerSlice(expr)
	if !ok {
		return NewNil(), false
	}
	if len(items) == 2 && scmerIsSymbol(items[0], "list") {
		return items[1], true
	}
	if len(items) == 4 && scmerIsSymbol(items[0], "!list") && items[2].IsInt() && items[2].Int() == 1 {
		return items[3], true
	}
	return NewNil(), false
}

// optimizeMergeUnique streams a mapped list-of-lists into an ordered unique
// collector when the existing bottom-up type information proves that every
// mapper result is a list. It does not revisit the mapper body.
func optimizeMergeUnique(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "merge_unique_mut")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	rv, ok := scmerSlice(result)
	if !ok || len(rv) != 2 {
		return result, td
	}
	producer, ok := scmerSlice(rv[1])
	if !ok || len(producer) != 3 ||
		(!scmerIsSymbol(producer[0], "map") && !scmerIsSymbol(producer[0], "map_mut")) ||
		exprMayHaveSideEffects(producer[2]) {
		return result, td
	}
	producerType := optimizedArgumentType(argumentTypes, 1)
	if producerType.Extra == nil || producerType.Extra.Element == nil || producerType.Extra.Element.Kind != "list" {
		return result, td
	}
	elementType := tiZero
	if producerType.Extra.Element.Element != nil {
		elementType = TypeInfoFromTD(producerType.Extra.Element.Element)
	}
	return NewSlice([]Scmer{NewSymbol("flat_map_unique"), producer[1], producer[2]}),
		setOptimizedCallElement(descriptorWithLength(FreshAlloc, UnknownLength), elementType)
}

func flattenConsList(v []Scmer) ([]Scmer, bool) {
	if len(v) != 3 {
		return nil, false
	}
	tail := v[2]
	if tail.GetTag() != tagSlice && !tail.IsSourceInfo() {
		return nil, false
	}
	if stripped, ok := scmerStripSourceInfo(tail); ok {
		tail = stripped
	}
	tailExpr, ok := scmerSlice(tail)
	if !ok || len(tailExpr) != 3 || !scmerIsSymbol(tailExpr[0], "cons") {
		return nil, false
	}

	headCount := 0
	tailCount := 0
	current := v
	for len(current) == 3 && scmerIsSymbol(current[0], "cons") {
		headCount++
		tail := current[2]
		if stripped, ok := scmerStripSourceInfo(tail); ok {
			tail = stripped
		}
		tailExpr, ok := scmerSlice(tail)
		if !ok || len(tailExpr) == 0 {
			return nil, false
		}
		if scmerIsSymbol(tailExpr[0], "cons") {
			current = tailExpr
			continue
		}
		if scmerIsSymbol(tailExpr[0], "list") {
			tailCount = len(tailExpr) - 1
			break
		}
		if len(tailExpr) == 2 && scmerIsSymbol(tailExpr[0], "quote") {
			if quoted, ok := scmerSlice(tailExpr[1]); ok && len(quoted) == 0 {
				break
			}
		}
		return nil, false
	}
	if headCount == 0 {
		return nil, false
	}

	items := make([]Scmer, 1, 1+headCount+tailCount)
	items[0] = NewSymbol("list")
	current = v
	for len(current) == 3 && scmerIsSymbol(current[0], "cons") {
		items = append(items, current[1])
		tail := current[2]
		if stripped, ok := scmerStripSourceInfo(tail); ok {
			tail = stripped
		}
		tailExpr, _ := scmerSlice(tail)
		if scmerIsSymbol(tailExpr[0], "cons") {
			current = tailExpr
			continue
		}
		if scmerIsSymbol(tailExpr[0], "list") {
			items = append(items, tailExpr[1:]...)
		}
		return items, true
	}
	return nil, false
}

// optimizeCons eliminates intermediate tail lists when their producer is known:
//
//	(cons head (map list fn)) → (cons_map head list fn)
//	(cons head (list a b c))  → (list head a b c)
func optimizeCons(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	if flattened, ok := flattenConsList(v); ok {
		return oc.ApplyDefaultOptimization(flattened, useResult)
	}
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	if rSlice, ok := scmerSlice(result); ok && len(rSlice) == 3 {
		tail := rSlice[2]
		if inner, ok2 := scmerSlice(tail); ok2 && len(inner) >= 1 {
			if len(inner) == 3 && (scmerIsSymbol(inner[0], "map") || scmerIsSymbol(inner[0], "map_mut")) {
				length := UnknownLength
				if tailLength := exactOptimizedListArgumentLength(tail, optimizedArgumentType(argumentTypes, 2)); tailLength >= 0 {
					length = tailLength + 1
				}
				return NewSlice([]Scmer{NewSymbol("cons_map"), rSlice[1], inner[1], inner[2]}), descriptorWithLength(FreshAlloc, length)
			}
			// (cons head (list a b c)) → (list head a b c)
			if scmerIsSymbol(inner[0], "list") {
				merged := make([]Scmer, 0, len(inner)+1)
				merged = append(merged, NewSymbol("list"))
				merged = append(merged, rSlice[1])    // head
				merged = append(merged, inner[1:]...) // tail items
				return NewSlice(merged), descriptorWithLength(FreshAlloc, len(merged)-1)
			}
		}
		if tailLength := exactOptimizedListArgumentLength(tail, optimizedArgumentType(argumentTypes, 2)); tailLength >= 0 {
			return result, setOptimizedCallLength(td, tailLength+1)
		}
	}
	return result, td
}
