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

func optimizedExactListLength(expr Scmer, oc *OptimizerContext) int {
	optimized, ti := OptimizeEx(expr, oc.Env, oc.Ome, true)
	if length := ti.Length(); length > 0 {
		return length
	}
	if length := exactListLengthFromExpr(optimized); length >= 0 {
		return length
	}
	if materialized, changed := materializeCodeLiteral(expr); changed {
		materializedOptimized, _ := OptimizeEx(materialized, oc.Env, oc.Ome, true)
		if length := exactListLengthFromExpr(materializedOptimized); length >= 0 {
			return length
		}
		return exactListLengthFromExpr(materialized)
	}
	return exactListLengthFromExpr(expr)
}

func optimizedExactAssocLength(expr Scmer, oc *OptimizerContext) int {
	optimized, ti := OptimizeEx(expr, oc.Env, oc.Ome, true)
	if length := ti.Length(); length > 0 {
		return length
	}
	if length := exactAssocLengthFromExpr(optimized); length >= 0 {
		return length
	}
	if materialized, changed := materializeCodeLiteral(expr); changed {
		materializedOptimized, _ := OptimizeEx(materialized, oc.Env, oc.Ome, true)
		if length := exactAssocLengthFromExpr(materializedOptimized); length >= 0 {
			return length
		}
		return exactAssocLengthFromExpr(materialized)
	}
	return exactAssocLengthFromExpr(expr)
}

func lambdaBodyResultExpr(expr Scmer) (Scmer, bool) {
	if stripped, ok := scmerStripSourceInfo(expr); ok {
		expr = stripped
	}
	inner, ok := scmerSlice(expr)
	if !ok || len(inner) < 3 || !scmerIsSymbol(inner[0], "lambda") {
		return NewNil(), false
	}
	body := inner[2]
	if stripped, ok := scmerStripSourceInfo(body); ok {
		body = stripped
	}
	if bodySlice, ok := scmerSlice(body); ok && len(bodySlice) > 0 && (scmerIsSymbol(bodySlice[0], "begin") || scmerIsSymbol(bodySlice[0], "!begin")) {
		if len(bodySlice) == 1 {
			return NewNil(), true
		}
		return bodySlice[len(bodySlice)-1], true
	}
	return body, true
}

func exactCallbackListLength(expr Scmer, oc *OptimizerContext) int {
	if body, ok := lambdaBodyResultExpr(expr); ok {
		return optimizedExactListLength(body, oc)
	}
	if decl := DeclarationForValue(expr); decl != nil && decl.Type != nil && decl.Type.Return != nil && decl.Type.Return.Length > 0 {
		return decl.Type.Return.Length
	}
	return UnknownLength
}

func exactFlattenedMergeLength(expr Scmer, oc *OptimizerContext) int {
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
				return exactFlattenedMergeLength(inner[1], oc)
			}
			return UnknownLength
		case "list":
			total := 0
			for _, item := range inner[1:] {
				itemLen := optimizedExactListLength(item, oc)
				if itemLen < 0 {
					return UnknownLength
				}
				total += itemLen
			}
			return total
		case "map", "map_mut", "parallel_map", "parallel_map_mut", "mapIndex", "mapIndex_mut":
			if len(inner) >= 3 {
				inputLen := optimizedExactListLength(inner[1], oc)
				itemLen := exactCallbackListLength(inner[2], oc)
				if inputLen >= 0 && itemLen >= 0 {
					return inputLen * itemLen
				}
			}
			return UnknownLength
		case "extract_assoc", "extract_assoc_mut":
			if len(inner) >= 3 {
				inputLen := optimizedExactAssocLength(inner[1], oc)
				itemLen := exactCallbackListLength(inner[2], oc)
				if inputLen >= 0 && itemLen >= 0 {
					return inputLen * itemLen
				}
			}
			return UnknownLength
		case "produceN", "produceN_mut", "parallelN", "parallelN_mut":
			if len(inner) >= 3 {
				if count := int(ToInt(inner[1])); count >= 0 {
					itemLen := exactCallbackListLength(inner[2], oc)
					if itemLen >= 0 {
						return count * itemLen
					}
				}
			}
			return UnknownLength
		case "merge":
			return optimizedExactListLength(expr, oc)
		}
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
	if len(v) == 2 && !exprMayHaveSideEffects(v[1]) {
		if length := optimizedExactListLength(v[1], oc); length >= 0 {
			return NewInt(int64(length)), &TypeDescriptor{Kind: "int", Transfer: true, Const: true, Length: UnknownLength}
		}
	}
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if rv, ok := scmerSlice(result); ok && len(rv) == 2 {
		if length := optimizedExactListLength(rv[1], oc); length >= 0 && !exprMayHaveSideEffects(rv[1]) {
			return NewInt(int64(length)), &TypeDescriptor{Kind: "int", Transfer: true, Const: true, Length: UnknownLength}
		}
	}
	return result, td
}

func optimizeFixedLengthInput(mutName string) func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	return func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
		result, td := oc.applyDefaultOptimization(v, useResult, mutName)
		if rv, ok := scmerSlice(result); ok && len(rv) >= 2 {
			return result, descriptorWithLength(td, optimizedExactListLength(rv[1], oc))
		}
		return result, td
	}
}

func optimizeAssocFixedLengthInput(mutName string) func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	return func(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
		result, td := oc.applyDefaultOptimization(v, useResult, mutName)
		if rv, ok := scmerSlice(result); ok && len(rv) >= 2 {
			return result, descriptorWithLength(td, optimizedExactAssocLength(rv[1], oc))
		}
		return result, td
	}
}

func optimizeExtractAssoc(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.applyDefaultOptimization(v, useResult, "extract_assoc_mut")
	if rv, ok := scmerSlice(result); ok && len(rv) >= 2 {
		return result, descriptorWithLength(td, optimizedExactAssocLength(rv[1], oc))
	}
	return result, td
}

func optimizeCdr(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	if rv, ok := scmerSlice(result); ok && len(rv) == 2 {
		if length := optimizedExactListLength(rv[1], oc); length >= 0 {
			return result, descriptorWithLength(td, length-1)
		}
	}
	return result, td
}

func optimizeZip(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 2 {
		return result, td
	}
	if len(rv) == 2 {
		argExpr := rv[1]
		if argList, ok := scmerSlice(argExpr); ok && len(argList) > 0 {
			expected := UnknownLength
			for _, item := range argList[1:] {
				itemLen := optimizedExactListLength(item, oc)
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
				return result, descriptorWithLength(td, expected)
			}
		}
		return result, td
	}
	minLen := UnknownLength
	for _, arg := range rv[1:] {
		length := optimizedExactListLength(arg, oc)
		if length < 0 {
			return result, td
		}
		if minLen == UnknownLength || length < minLen {
			minLen = length
		}
	}
	return result, descriptorWithLength(td, minLen)
}

// optimizeMap is the optimizer hook for `map`. It applies default optimization
// (including FirstParameterMutable swap to map_mut), then fuses
// (map (produceN N) fn) → (produceN N fn) to eliminate the intermediate list.
func optimizeMap(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Run default optimization first (handles map → map_mut swap etc.)
	result, td := oc.applyDefaultOptimization(v, useResult, "map_mut")
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
								return NewSlice([]Scmer{inner[0], inner[1], rv[2]}), descriptorWithLength(td, count)
							}
							return NewSlice([]Scmer{inner[0], inner[1], rv[2]}), td
						}
					}
				}
			}
		}
	}
	if rv, ok := scmerSlice(result); ok && len(rv) == 3 {
		return result, descriptorWithLength(td, optimizedExactListLength(rv[1], oc))
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

			JITEmit: nil,
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
			Return: &TypeDescriptor{Kind: "any"},
			Const:  true,

			JITEmit: nil,
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

			JITEmit: nil,
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
	result, td := oc.applyDefaultOptimization(v, useResult, "append_mut")
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 2 {
		return result, td
	}
	baseLength := optimizedExactListLength(rv[1], oc)
	if baseLength >= 0 {
		td = descriptorWithLength(td, baseLength+len(rv)-2)
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
	result, td := oc.ApplyDefaultOptimization(v, useResult)
	rv, ok := scmerSlice(result)
	if !ok || len(rv) < 2 {
		return result, td
	}
	if len(rv) == 2 {
		if producer, ok := scmerSlice(rv[1]); ok && len(producer) == 3 {
			if exprMayHaveSideEffects(producer[2]) {
				return result, td
			}
			itemLength := exactCallbackListLength(producer[2], oc)
			if itemLength >= 0 {
				switch {
				case scmerIsSymbol(producer[0], "map"), scmerIsSymbol(producer[0], "map_mut"):
					inputLength := optimizedExactListLength(producer[1], oc)
					resultLength := UnknownLength
					if inputLength >= 0 {
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
		if totalLength := exactFlattenedMergeLength(rv[1], oc); totalLength > 0 {
			return result, descriptorWithLength(td, totalLength)
		}
		return result, td
	}
	totalLength := 0
	allExact := len(rv) > 2
	allDirectListArgs := len(rv) > 2
	for _, arg := range rv[1:] {
		length := optimizedExactListLength(arg, oc)
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
		return result, descriptorWithLength(td, totalLength)
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
	result, td := oc.ApplyDefaultOptimization(v, useResult)
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
		if tailLength := optimizedExactListLength(tail, oc); tailLength >= 0 {
			return result, descriptorWithLength(td, tailLength+1)
		}
	}
	return result, td
}
