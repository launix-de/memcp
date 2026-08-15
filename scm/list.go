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

func exactListLengthFromExpr(expr Scmer) int {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	if inner, ok := scmerSlice(expr); ok {
		if len(inner) == 0 {
			return UnknownLength
		}
		if sym, ok := scmerSymbol(inner[0]); ok {
			switch sym {
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
					if outer, ok := scmerSlice(arg); ok && len(outer) > 0 && scmerIsSymbol(outer[0], "list") {
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
					if outer, ok := scmerSlice(arg); ok && len(outer) > 0 && scmerIsSymbol(outer[0], "list") {
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
		if sym, ok := scmerSymbol(inner[0]); ok {
			switch sym {
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
	if sym, ok := scmerSymbol(inner[0]); ok {
		switch sym {
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

// mergeValidationSafeArgument reports whether evaluating an argument before
// merge's list validation is unobservable. Fused calls evaluate their ordinary
// arguments before reduce_segments validates the segment catalog, whereas the
// unfused spelling validates during evaluation of reduce's first argument.
func mergeValidationSafeArgument(v Scmer, allowLambda bool) bool {
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

// optimizeReduce keeps merge as a segmented producer when its flattened value
// is consumed exactly once by reduce. reduce_segments validates all segments
// before invoking the callback, matching merge's existing error order.
func optimizeReduce(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 3 || len(rv) > 4 || !scmerIsSymbol(rv[0], "reduce") {
		return result, td
	}
	segments, ok := optimizedMergeSegments(rv[1])
	if !ok || !mergeValidationSafeArgument(rv[2], true) || (len(rv) == 4 && !mergeValidationSafeArgument(rv[3], false)) {
		return result, td
	}
	fused := make([]Scmer, 0, len(rv))
	fused = append(fused, NewSymbol("reduce_segments"), segments)
	fused = append(fused, rv[2:]...)
	return NewSlice(fused), td
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
	if !ok || len(inner) != 3 || (!scmerIsSymbol(inner[0], "map") && !scmerIsSymbol(inner[0], "map_mut")) {
		return result, td
	}
	if exprMayHaveSideEffects(inner[2]) || exprMayHaveSideEffects(rv[2]) {
		return result, td
	}
	fused := NewSlice([]Scmer{NewSymbol("filter_map"), inner[1], inner[2], rv[2]})
	return fused, descriptorWithLength(FreshAlloc, UnknownLength)
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

func init_list() {
	// list functions
	DeclareTitle("Lists")

	// list is already in Globalenv.Vars (scm.go init); register it
	// in declarations so serialization can resolve the function pointer.
	Declare(&Globalenv, &Declaration{
		Name: "list",
		Desc: "constructs a list from its arguments",
		Fn:   List,
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "items", ParamDesc: "items to put into the list", Variadic: true},
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
		Desc: "counts the number of elements in the list",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list", NoEscape: true},
			},
			Return:   &TypeDescriptor{Kind: "int"},
			Const:    true,
			Optimize: optimizeCount,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "nth",
		Desc: "get the nth item of a list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "nth")
			idx := int(a[1].Int())
			if idx < 0 || idx >= len(list) {
				panic("nth index out of range")
			}
			return list[idx]
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list", NoEscape: true},
				{Kind: "number", ParamName: "index", ParamDesc: "index beginning from 0"},
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
		Desc: "sets the nth item of an owned list in-place and returns the mutated list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "nth_mut")
			idx := int(a[1].Int())
			if idx < 0 || idx >= len(list) {
				panic("nth_mut index out of range")
			}
			list[idx] = a[2]
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned base list"},
				{Kind: "number", ParamName: "index", ParamDesc: "index beginning from 0"},
				{Kind: "any", ParamName: "value", ParamDesc: "new value"},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "slice",
		Desc: "extract a sublist from start (inclusive) to end (exclusive).\n(slice list start end) returns elements list[start..end).",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list", NoEscape: true},
				{Kind: "number", ParamName: "start", ParamDesc: "start index (inclusive)"},
				{Kind: "number", ParamName: "end", ParamDesc: "end index (exclusive)"},
			},
			Return: &TypeDescriptor{Kind: "list"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reverse",
		Desc: "returns a new list with elements in reversed order.",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "reverse")
			n := len(list)
			result := make([]Scmer, n)
			for i := 0; i < n; i++ {
				result[i] = list[n-1-i]
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to reverse", NoEscape: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeFixedLengthInput("reverse_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "append",
		Desc: "appends items to a list and return the extended list.\nThe original list stays unharmed.",
		Fn: func(a ...Scmer) Scmer {
			base := append([]Scmer{}, asSlice(a[0], "append")...)
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeAppend,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "append_unique",
		Desc: "appends items to a list but only if they are new.\nThe original list stays unharmed.",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: FirstParameterMutable("append_unique_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "cons",
		Desc: "constructs a list from a head and a tail list",
		Fn: func(a ...Scmer) Scmer {
			car := a[0]
			if a[1].GetTag() == tagSlice {
				return NewSlice(append([]Scmer{car}, a[1].Slice()...))
			}
			return NewSlice([]Scmer{car, a[1]})
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "car", ParamDesc: "new head element"},
				{Kind: "list", ParamName: "cdr", ParamDesc: "tail that is appended after car", NoEscape: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeCons,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "car",
		Desc: "extracts the head of a list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "car")
			if len(list) == 0 {
				panic("car on empty list")
			}
			return list[0]
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list", NoEscape: true},
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
		Desc: "extracts the tail of a list\nThe tail of a list is a list with all items except the head.",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cdr")
			if len(list) == 0 {
				return NewSlice([]Scmer{})
			}
			return NewSlice(list[1:])
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list", NoEscape: true},
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
		Desc: "extracts the second element of a list.\nEquivalent to (car (cdr x)).",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cadr")
			if len(list) < 2 {
				panic("cadr on list with fewer than 2 elements")
			}
			return list[1]
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list", NoEscape: true},
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
		Desc: "swaps the dimension of a list of lists. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as the components that will be zipped into the sub list",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "list", ParamDesc: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeZip,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge",
		Desc: "flattens a list of lists into a list containing all the subitems. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as lists that will be merged into one",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "list", ParamDesc: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeMerge,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge_unique",
		Desc: "flattens a list of lists into a list containing all the subitems. Duplicates are filtered out.",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list of lists of items", NoEscape: true, Variadic: true},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeMergeUnique,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "has?",
		Desc: "checks if a list has a certain item (equal?)",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "has?")
			for _, v := range list {
				if Equal(a[1], v) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "haystack", ParamDesc: "list to search in", NoEscape: true},
				{Kind: "any", ParamName: "needle", ParamDesc: "item to search for"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "filter",
		Desc: "returns a list that only contains elements that pass the filter function",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be filtered", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter condition func(item)->bool", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeFilter,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "find",
		Desc: "returns the first list element that passes the condition function, or nil/default if none matches",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to search", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "predicate func(any)->bool that is applied until the first match", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "any", ParamName: "default", ParamDesc: "optional default value if nothing matches", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map",
		Desc: "returns a list that contains the results of a map function that is applied to the list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "map")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(v)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be mapped", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(any)->any that is applied to each item", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeMap,

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
		Desc: "like map, but applies fn to each element in parallel using a worker pool limited to runtime.NumCPU()",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to map over in parallel", NoEscape: true},
				{Kind: "func", ParamName: "fn", ParamDesc: "function applied to each element", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Optimize: optimizeFixedLengthInput("parallel_map_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallel_map_mut",
		Desc: "like parallel_map, but signals the optimizer that fn may have side effects",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to map over in parallel", NoEscape: true},
				{Kind: "func", ParamName: "fn", ParamDesc: "function with side effects applied to each element", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mapIndex",
		Desc: "returns a list that contains the results of a map function that is applied to the list",
		Fn: func(a ...Scmer) Scmer {
			list := asSlice(a[0], "mapIndex")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(result)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be mapped", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(i, any)->any that is applied to each item", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}, {Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeFixedLengthInput("mapIndex_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce",
		Desc: "returns a list that contains the result of a map function",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list that has to be reduced", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Transfer: true, ParamName: "acc"}, {ParamName: "item"}}, ParamName: "reduce", ParamDesc: "reduce function func(any any)->any where the first parameter is the accumulator, the second is a list item", Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", ParamName: "neutral", ParamDesc: "(optional) initial value of the accumulator, defaults to nil", Optional: true},
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
		Desc: "returns a list that contains produced items - it works like for(state = startstate, condition(state), state = iterator(state)) {yield state}",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "startstate", ParamDesc: "start state to begin with"},
				{Kind: "func", ParamName: "condition", ParamDesc: "func that returns true whether the state will be inserted into the result or the loop is stopped", Params: []*TypeDescriptor{{Kind: "any", ParamName: "state"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "func", ParamName: "iterator", ParamDesc: "func that produces the next state", Params: []*TypeDescriptor{{Kind: "any", ParamName: "state"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: FreshAlloc,
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "produceN",
		Desc: "returns a list with numbers from 0..n-1, optionally mapped through a function",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "(optional) map function applied to each index", Optional: true, Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeProduceN,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallelN",
		Desc: "returns a list with numbers from 0..n-1 mapped in parallel through a function",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "map function applied to each index in parallel", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeParallelN,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "produceN_mut",
		Desc: "in-place produceN variant (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "map function applied to each index", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "list", ParamName: "target", ParamDesc: "(optional) preallocated target list", NoEscape: true, Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "list"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "parallelN_mut",
		Desc: "in-place parallelN variant (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "number", ParamName: "n", ParamDesc: "number of elements to produce"},
				{Kind: "func", ParamName: "fn", ParamDesc: "map function applied to each index in parallel", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "list", ParamName: "target", ParamDesc: "(optional) preallocated target list", NoEscape: true, Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "list"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "list?",
		Desc: "checks if a value is a list",
		Fn: func(a ...Scmer) Scmer {
			if a[0].IsSlice() {
				return NewBool(true)
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "any", ParamName: "value", ParamDesc: "value to check"},
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
		Desc: "checks if a value is in a list; uses the equal?? operator",
		Fn: func(a ...Scmer) Scmer {
			arr := asSlice(a[0], "contains?")
			for _, v := range arr {
				if Equal(v, a[1]) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "list to check", NoEscape: true},
				{Kind: "any", ParamName: "value", ParamDesc: "value to check"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sql_in",
		Desc: "tests SQL IN-list membership and returns nil when NULL makes the result UNKNOWN",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{{Kind: "list", ParamName: "values", ParamDesc: "SQL IN-list values", NoEscape: true}, {Kind: "any", ParamName: "value", ParamDesc: "value to find"}},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})

	// dictionary functions
	DeclareTitle("Associative Lists / Dictionaries")

	Declare(&Globalenv, &Declaration{
		Name: "filter_assoc",
		Desc: "returns a filtered dictionary according to a filter function",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be filtered", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter function func(string any)->bool where the first parameter is the key, the second is the value", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: FirstParameterMutable("filter_assoc_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "find_assoc",
		Desc: "returns the first key/value pair that passes the condition function, or nil/default if none matches",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary to search", NoEscape: true},
				{Kind: "func", ParamName: "condition", ParamDesc: "predicate func(string any)->bool that is applied until the first match", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "bool"}},
				{Kind: "any", ParamName: "default", ParamDesc: "optional default value if nothing matches", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "map_assoc",
		Desc: "returns a mapped dictionary according to a map function\nKeys will stay the same but values are mapped.",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be mapped", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(string any)->any where the first parameter is the key, the second is the value. It must return the new value.", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeAssocFixedLengthInput("map_assoc_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "reduce_assoc",
		Desc: "reduces a dictionary according to a reduce function",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be reduced", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Transfer: true, ParamName: "acc"}, {ParamName: "key"}, {ParamName: "value"}}, ParamName: "reduce", ParamDesc: "reduce function func(any string any)->any where the first parameter is the accumulator, second is key, third is value. It must return the new accumulator.", Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", ParamName: "neutral", ParamDesc: "initial value for the accumulator"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "make_structural_index",
		Desc: "Builds an immutable structural-expression index. It eagerly hashes every key and every node under roots, then returns a parallel-safe lookup function that maps an equal expression to its zero-based key position or nil.",
		Fn:   NewStructuralIndex,
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "keys", ParamDesc: "immutable structural expressions to index"},
				{Kind: "list", ParamName: "roots", ParamDesc: "immutable expression roots whose descendant hashes are precomputed"},
			},
			Return: &TypeDescriptor{
				Kind: "func",
				Params: []*TypeDescriptor{
					{Kind: "any", ParamName: "expression", ParamDesc: "a key, root, descendant of a declared root, or scalar expression"},
				},
				Return: &TypeDescriptor{Kind: "int|nil"},
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "make_structural_catalog",
		Desc: "Creates an atomic compile-local structural catalog. Look up with (catalog key), insert with (catalog key value), or freeze with (catalog) for parallel-safe read-only lookup.",
		Fn:   NewStructuralCatalog,
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "bool", ParamName: "force_collision", ParamDesc: "test-only: place every key in one bucket", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "func"},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "has_assoc?",
		Desc: "checks if a dictionary has a key present",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be checked", NoEscape: true},
				{Kind: "string", ParamName: "key", ParamDesc: "key to test"},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "get_assoc",
		Desc: "gets a value from a dictionary by key, returns nil if not found",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary to look up", NoEscape: true},
				{Kind: "any", ParamName: "key", ParamDesc: "key to look up"},
				{Kind: "any", ParamName: "default", ParamDesc: "optional default value if key not found", Optional: true},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "get_assoc_pairlist",
		Desc: "gets a value from a list of key/value rows without flattening the rows",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "rows", ParamDesc: "list whose rows contain a key followed by one or more values", NoEscape: true},
				{Kind: "any", ParamName: "key", ParamDesc: "key compared with the first item of each row"},
				{Kind: "any", ParamName: "default", ParamDesc: "value returned when no row contains the key"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "extract_assoc",
		Desc: "applies a function (key value) on the dictionary and returns the results as a flat list",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "dictionary that has to be checked", NoEscape: true},
				{Kind: "func", ParamName: "map", ParamDesc: "func(key, value)->any that extracts one element per key-value pair", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: optimizeExtractAssoc,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "set_assoc",
		Desc: "returns a new dictionary where a single value has been changed.\nThe original dictionary is not modified.",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "input dictionary"},
				{Kind: "string", ParamName: "key", ParamDesc: "key that has to be set"},
				{Kind: "any", ParamName: "value", ParamDesc: "new value to set"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value. It must return the merged value that shall be physically stored in the new dictionary.", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: FirstParameterMutable("set_assoc_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "merge_assoc",
		Desc: "returns a dictionary where all keys from dict1 and all keys from dict2 are present.\nIf a key is present in both inputs, the second one will be dominant so the first value will be overwritten unless you provide a merge function",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict1", ParamDesc: "first input dictionary that has to be changed. You must not use this value again."},
				{Kind: "list", ParamName: "dict2", ParamDesc: "input dictionary that contains the new values that have to be added"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value from dict2. It must return the merged value that shall be pysically stored in the new dictionary.", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: FirstParameterMutable("merge_assoc_mut"),

			JITEmit: nil,
		},
	})

	// Fused physical operators: optimizer-only, forbidden from .scm code.
	Declare(&Globalenv, &Declaration{
		Name: "reduce_segments",
		Desc: "reduces ordered list segments without flattening them (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "segments", NoEscape: true},
				{Kind: "func", Params: []*TypeDescriptor{{Transfer: true, ParamName: "acc"}, {ParamName: "item"}}, ParamName: "reduce", Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", ParamName: "neutral", Optional: true},
			},
			Return:    &TypeDescriptor{Kind: "any"},
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "filter_map",
		Desc: "fused serial map and filter (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", NoEscape: true},
				{Kind: "func", ParamName: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", ParamName: "condition", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "flat_map",
		Desc: "fused fixed-width serial map and flatten (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", NoEscape: true},
				{Kind: "func", ParamName: "map", Params: []*TypeDescriptor{{Kind: "any"}}, Return: &TypeDescriptor{Kind: "list"}},
				{Kind: "int", ParamName: "width"},
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
		Desc: "in-place map (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(v)
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to map in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "map function", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
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
		Desc: "in-place mapIndex (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to map in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(i, any)->any", Params: []*TypeDescriptor{{Kind: "int", ParamName: "index"}, {Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "map_assoc_mut",
		Desc: "in-place map_assoc (optimizer-only, slice path only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to map in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "map function func(key, value)->value", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
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
		Desc: "in-place filter (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to filter in-place"},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter condition func(any)->bool", Params: []*TypeDescriptor{{Kind: "any", ParamName: "item"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "reverse_mut",
		Desc: "in-place reverse (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			list := a[0].Slice()
			for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
				list[i], list[j] = list[j], list[i]
			}
			return NewSlice(list)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned list to reverse in-place"},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "filter_assoc_mut",
		Desc: "in-place filter_assoc (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to filter in-place"},
				{Kind: "func", ParamName: "condition", ParamDesc: "filter function func(key, value)->bool", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "extract_assoc_mut",
		Desc: "in-place extract_assoc (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to extract from in-place"},
				{Kind: "func", ParamName: "map", ParamDesc: "func(key, value)->any that extracts each element", Params: []*TypeDescriptor{{Kind: "string", ParamName: "key"}, {Kind: "any", ParamName: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "set_assoc_mut",
		Desc: "in-place set_assoc (optimizer-only, mutates input directly)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict", ParamDesc: "owned dictionary to mutate"},
				{Kind: "string", ParamName: "key", ParamDesc: "key to set"},
				{Kind: "any", ParamName: "value", ParamDesc: "new value"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) merge function", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
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
		Desc: "in-place append (optimizer-only)",
		Fn: func(a ...Scmer) Scmer {
			base := asSlice(a[0], "append_mut")
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "append_unique_mut",
		Desc: "in-place append_unique (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned base list"},
				{Kind: "any", ParamName: "item...", ParamDesc: "items to add", Variadic: true},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "merge_unique_mut",
		Desc: "in-place merge_unique (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned base list or owned list of lists", Variadic: true},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "reset_mut",
		Desc: "resets an owned list to len=0 while preserving capacity",
		Fn: func(a ...Scmer) Scmer {
			base := asSlice(a[0], "reset_mut")
			if base == nil {
				return NewSlice([]Scmer{})
			}
			return NewSlice(base[:0:cap(base)])
		},
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", ParamDesc: "owned base list"},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "merge_assoc_mut",
		Desc: "in-place merge_assoc (optimizer-only)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "dict1", ParamDesc: "owned first dictionary"},
				{Kind: "list", ParamName: "dict2", ParamDesc: "dictionary with new values"},
				{Kind: "func", ParamName: "merge", ParamDesc: "(optional) merge function", Optional: true, Params: []*TypeDescriptor{{Kind: "any", ParamName: "old"}, {Kind: "any", ParamName: "new"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sort",
		Desc: "returns a sorted copy of a list using a comparator (lambda (a b) truthy/falsy)",
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
		Type: &TypeDescriptor{
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list", NoEscape: true},
				{Kind: "func", ParamName: "comparator", Params: []*TypeDescriptor{{Kind: "any"}, {Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
			},
			Return:   FreshAlloc,
			Const:    true,
			Optimize: FirstParameterMutable("sort_mut"),

			JITEmit: nil,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "sort_mut",
		Desc: "sorts a list in-place using a comparator (lambda (a b) truthy/falsy)",
		Fn: func(a ...Scmer) Scmer {
			src := a[0].Slice()
			cmp := a[1]
			hybridsort.SliceStable(src, func(i, j int) bool {
				return ToBool(Apply(cmp, src[i], src[j]))
			})
			return a[0]
		},
		Type: &TypeDescriptor{
			HasSideEffects: true,
			Params: []*TypeDescriptor{
				{Kind: "list", ParamName: "list"},
				{Kind: "func", ParamName: "comparator", Params: []*TypeDescriptor{{Kind: "any"}, {Kind: "any"}}, Return: &TypeDescriptor{Kind: "bool"}},
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

// optimizeMergeUnique keeps merge_unique on the standard variadic path, but
// additionally treats a direct (list ...) first argument as fresh so the
// optimizer can swap to merge_unique_mut without changing the global list
// return contract.
func optimizeMergeUnique(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	firstArgListLiteral := false
	if len(v) > 2 {
		arg1 := v[1]
		if stripped, ok := scmerStripSourceInfo(arg1); ok {
			arg1 = stripped
		}
		if inner, ok := scmerSlice(arg1); ok && len(inner) > 0 && scmerIsSymbol(inner[0], "list") {
			firstArgListLiteral = true
		}
	}

	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if !firstArgListLiteral {
		return result, td
	}

	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 2 || !scmerIsSymbol(rv[0], "merge_unique") {
		return result, td
	}
	rv[0] = NewSymbol("merge_unique_mut")
	if td == nil {
		td = &TypeDescriptor{}
	}
	td.Transfer = true
	return NewSlice(rv), td
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

// optimizeCons rewrites cons when the tail is a freshly allocated list:
//
//	(cons head (map list fn)) → (cons head (map_mut list fn))  — already handled by _mut
//	(cons head (list a b c))  → (list head a b c)              — avoid double allocation
func optimizeCons(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	if flattened, ok := flattenConsList(v); ok {
		return oc.ApplyDefaultOptimization(flattened, useResult)
	}
	call := oc.applyDefaultOptimizationWithTypes(v, useResult, "")
	result, td, argumentTypes := call.code, call.typeInfo, call.argumentTypes
	if rSlice, ok := scmerSlice(result); ok && len(rSlice) == 3 {
		tail := rSlice[2]
		if inner, ok2 := scmerSlice(tail); ok2 && len(inner) >= 1 {
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
