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
							return NewSlice([]Scmer{inner[0], inner[1], rv[2]}), td
						}
					}
				}
			}
		}
	}
	return result, td
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
		"list", "constructs a list from its arguments",
		0, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"items", "any", "items to put into the list", nil},
		}, "list",
		List,
		true, false, nil,
		nil,
	})

	Declare(&Globalenv, &Declaration{
		"count", "counts the number of elements in the list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", NoEscape},
		}, "int",
		func(a ...Scmer) Scmer {
			if a[0].GetTag() == tagSlice {
				return NewInt(int64(len(a[0].Slice())))
			}
			if a[0].GetTag() == tagFastDict {
				return NewInt(int64(len(a[0].FastDict().Pairs)))
			}
			panic("count expects a list")
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := args[0]
			d1 := ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(6))}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 6)
				ctx.W.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			d3 := d2
			ctx.EnsureDesc(&d3)
			if d3.Loc != LocImm && d3.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d3.Loc == LocImm {
				if d3.Imm.Bool() {
					ctx.W.MarkLabel(lbl4)
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d3.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl4)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
			}
			ctx.FreeDesc(&d2)
			ctx.W.MarkLabel(lbl3)
			d4 := args[0]
			d5 := ctx.EmitGetTagDesc(&d4, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d5)
			var d6 JITValueDesc
			if d5.Loc == LocImm {
				d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d5.Imm.Int()) == uint64(14))}
			} else {
				r1 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d5.Reg, 14)
				ctx.W.EmitSetcc(r1, CcE)
				d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d6)
			}
			ctx.FreeDesc(&d5)
			d7 := d6
			ctx.EnsureDesc(&d7)
			if d7.Loc != LocImm && d7.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.MarkLabel(lbl8)
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.MarkLabel(lbl9)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl8)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl2)
			d8 := args[0]
			var d9 JITValueDesc
			if d8.Loc == LocImm {
				slice := d8.Imm.Slice()
				d9 = JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: NewInt(int64(len(slice)))}
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r2, d8.Reg2)
				ctx.W.EmitShlRegImm8(r2, 16)
				ctx.W.EmitShrRegImm8(r2, 16)
				ctx.FreeReg(d8.Reg2)
				d9 = JITValueDesc{Loc: LocRegPair, Reg: d8.Reg, Reg2: r2}
				ctx.BindReg(d8.Reg, &d9)
				ctx.BindReg(r2, &d9)
			}
			ctx.FreeDesc(&d8)
			var d10 JITValueDesc
			if d9.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d9.StackOff))}
			} else {
				ctx.EnsureDesc(&d9)
				if d9.Loc == LocRegPair {
					d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg2}
					ctx.BindReg(d9.Reg2, &d10)
					ctx.BindReg(d9.Reg2, &d10)
				} else if d9.Loc == LocReg {
					d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg}
					ctx.BindReg(d9.Reg, &d10)
					ctx.BindReg(d9.Reg, &d10)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			ctx.W.EmitMakeInt(result, d10)
			if d10.Loc == LocReg { ctx.FreeReg(d10.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl6)
			d12 := args[0]
			var d13 JITValueDesc
			if d12.Loc == LocImm {
				panic("FastDict: LocImm not expected at JIT compile time")
			} else {
				ctx.FreeReg(d12.Reg2)
				d13 = JITValueDesc{Loc: LocReg, Reg: d12.Reg}
				ctx.BindReg(d12.Reg, &d13)
			}
			ctx.FreeDesc(&d12)
			var d14 JITValueDesc
			ctx.EnsureDesc(&d13)
			if d13.Loc == LocImm {
				fieldAddr := uintptr(d13.Imm.Int()) + 0
				r3 := ctx.AllocReg()
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r3, fieldAddr)
				ctx.W.EmitMovRegMem64(r4, fieldAddr+8)
				d14 = JITValueDesc{Loc: LocRegPair, Reg: r3, Reg2: r4}
				ctx.BindReg(r3, &d14)
				ctx.BindReg(r4, &d14)
			} else {
				off := int32(0)
				baseReg := d13.Reg
				r5 := ctx.AllocRegExcept(baseReg)
				r6 := ctx.AllocRegExcept(baseReg, r5)
				ctx.W.EmitMovRegMem(r5, baseReg, off)
				ctx.W.EmitMovRegMem(r6, baseReg, off+8)
				d14 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d14)
				ctx.BindReg(r6, &d14)
			}
			ctx.FreeDesc(&d13)
			var d15 JITValueDesc
			if d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d14.StackOff))}
			} else {
				ctx.EnsureDesc(&d14)
				if d14.Loc == LocRegPair {
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg2}
					ctx.BindReg(d14.Reg2, &d15)
					ctx.BindReg(d14.Reg2, &d15)
				} else if d14.Loc == LocReg {
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg}
					ctx.BindReg(d14.Reg, &d15)
					ctx.BindReg(d14.Reg, &d15)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			ctx.W.EmitMakeInt(result, d15)
			if d15.Loc == LocReg { ctx.FreeReg(d15.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			return result
		}, /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */
	})
	Declare(&Globalenv, &Declaration{
		"nth", "get the nth item of a list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", NoEscape},
			DeclarationParameter{"index", "number", "index beginning from 0", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "nth")
			idx := int(a[1].Int())
			if idx < 0 || idx >= len(list) {
				panic("nth index out of range")
			}
			return list[idx]
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"slice", "extract a sublist from start (inclusive) to end (exclusive).\n(slice list start end) returns elements list[start..end).",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", NoEscape},
			DeclarationParameter{"start", "number", "start index (inclusive)", nil},
			DeclarationParameter{"end", "number", "end index (exclusive)", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"append", "appends items to a list and return the extended list.\nThe original list stays unharmed.",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", nil},
			DeclarationParameter{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			base := append([]Scmer{}, asSlice(a[0], "append")...)
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("append_mut")},
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
	Declare(&Globalenv, &Declaration{
		"append_unique", "appends items to a list but only if they are new.\nThe original list stays unharmed.",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", nil},
			DeclarationParameter{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("append_unique_mut")},
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
	Declare(&Globalenv, &Declaration{
		"cons", "constructs a list from a head and a tail list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"car", "any", "new head element", nil},
			DeclarationParameter{"cdr", "list", "tail that is appended after car", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
			car := a[0]
			if a[1].GetTag() == tagSlice {
				return NewSlice(append([]Scmer{car}, a[1].Slice()...))
			}
			return NewSlice([]Scmer{car, a[1]})
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"car", "extracts the head of a list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list", NoEscape},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "car")
			if len(list) == 0 {
				panic("car on empty list")
			}
			return list[0]
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"cdr", "extracts the tail of a list\nThe tail of a list is a list with all items except the head.",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list", NoEscape},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cdr")
			if len(list) == 0 {
				return NewSlice([]Scmer{})
			}
			return NewSlice(list[1:])
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"cadr", "extracts the second element of a list.\nEquivalent to (car (cdr x)).",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list", NoEscape},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cadr")
			if len(list) < 2 {
				panic("cadr on list with fewer than 2 elements")
			}
			return list[1]
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"zip", "swaps the dimension of a list of lists. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as the components that will be zipped into the sub list",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "any", "list of lists of items", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
	})
	Declare(&Globalenv, &Declaration{
		"merge", "flattens a list of lists into a list containing all the subitems. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as lists that will be merged into one",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "any", "list of lists of items", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
	})
	Declare(&Globalenv, &Declaration{
		"merge_unique", "flattens a list of lists into a list containing all the subitems. Duplicates are filtered out.",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list of lists of items", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
	})
	Declare(&Globalenv, &Declaration{
		"has?", "checks if a list has a certain item (equal?)",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"haystack", "list", "list to search in", NoEscape},
			DeclarationParameter{"needle", "any", "item to search for", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "has?")
			for _, v := range list {
				if Equal(a[1], v) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"filter", "returns a list that only contains elements that pass the filter function",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be filtered", NoEscape},
			DeclarationParameter{"condition", "func", "filter condition func(any)->bool", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("filter_mut")},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"map", "returns a list that contains the results of a map function that is applied to the list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be mapped", NoEscape},
			DeclarationParameter{"map", "func", "map function func(any)->any that is applied to each item", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "map")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(v)
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: optimizeMap},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"mapIndex", "returns a list that contains the results of a map function that is applied to the list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be mapped", NoEscape},
			DeclarationParameter{"map", "func", "map function func(i, any)->any that is applied to each item", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "mapIndex")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("mapIndex_mut")},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"reduce", "returns a list that contains the result of a map function",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be reduced", NoEscape},
			DeclarationParameter{"reduce", "func", "reduce function func(any any)->any where the first parameter is the accumulator, the second is a list item", &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Transfer: true}, nil}}},
			DeclarationParameter{"neutral", "any", "(optional) initial value of the accumulator, defaults to nil", nil},
		}, "any",
		func(a ...Scmer) Scmer {
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
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"produce", "returns a list that contains produced items - it works like for(state = startstate, condition(state), state = iterator(state)) {yield state}",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"startstate", "any", "start state to begin with", nil},
			DeclarationParameter{"condition", "func", "func that returns true whether the state will be inserted into the result or the loop is stopped", nil},
			DeclarationParameter{"iterator", "func", "func that produces the next state", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t0[:0:int] */, /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"produceN", "returns a list with numbers from 0..n-1, optionally mapped through a function",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"n", "number", "number of elements to produce", nil},
			DeclarationParameter{"fn", "func", "(optional) map function applied to each index", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: MakeSlice: make []Scmer t5 t5 */, /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */
	})
	Declare(&Globalenv, &Declaration{
		"list?", "checks if a value is a list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to check", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].IsSlice())
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d2 := d0
			d2.ID = 0
			d1 := ctx.EmitTagEquals(&d2, tagSlice, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d1.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d1)
			} else {
				ctx.W.EmitMakeBool(result, d1)
				ctx.FreeReg(d1.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
		}, /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */
	})
	Declare(&Globalenv, &Declaration{
		"contains?", "checks if a value is in a list; uses the equal?? operator",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list to check", NoEscape},
			DeclarationParameter{"value", "any", "value to check", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			arr := asSlice(a[0], "contains?")
			for _, v := range arr {
				if Equal(v, a[1]) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	// dictionary functions
	DeclareTitle("Associative Lists / Dictionaries")

	Declare(&Globalenv, &Declaration{
		"filter_assoc", "returns a filtered dictionary according to a filter function",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be filtered", NoEscape},
			DeclarationParameter{"condition", "func", "filter function func(string any)->bool where the first parameter is the key, the second is the value", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("filter_assoc_mut")},
		nil /* TODO: Slice on non-desc: slice t1[:0:int] */, /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"map_assoc", "returns a mapped dictionary according to a map function\nKeys will stay the same but values are mapped.",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be mapped", NoEscape},
			DeclarationParameter{"map", "func", "map function func(string any)->any where the first parameter is the key, the second is the value. It must return the new value.", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("map_assoc_mut")},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"reduce_assoc", "reduces a dictionary according to a reduce function",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be reduced", NoEscape},
			DeclarationParameter{"reduce", "func", "reduce function func(any string any)->any where the first parameter is the accumulator, second is key, third is value. It must return the new accumulator.", &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Transfer: true}, nil, nil}}},
			DeclarationParameter{"neutral", "any", "initial value for the accumulator", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			result := a[2]
			reduce := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "reduce_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					result = reduce(result, slice[i], slice[i+1])
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool { result = reduce(result, k, v); return true })
			}
			return result
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"has_assoc?", "checks if a dictionary has a key present",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be checked", NoEscape},
			DeclarationParameter{"key", "string", "key to test", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
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
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"get_assoc", "gets a value from a dictionary by key, returns nil if not found",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary to look up", NoEscape},
			DeclarationParameter{"key", "any", "key to look up", nil},
			DeclarationParameter{"default", "any", "optional default value if key not found", nil},
		}, "any",
		func(a ...Scmer) Scmer {
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
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"extract_assoc", "applies a function (key value) on the dictionary and returns the results as a flat list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be checked", NoEscape},
			DeclarationParameter{"map", "func", "func(string any)->any that flattens down each element", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("extract_assoc_mut")},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"set_assoc", "returns a new dictionary where a single value has been changed.\nThe original dictionary is not modified.",
		3, 4,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "input dictionary", nil},
			DeclarationParameter{"key", "string", "key that has to be set", nil},
			DeclarationParameter{"value", "any", "new value to set", nil},
			DeclarationParameter{"merge", "func", "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value. It must return the merged value that shall be physically stored in the new dictionary.", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("set_assoc_mut")},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"merge_assoc", "returns a dictionary where all keys from dict1 and all keys from dict2 are present.\nIf a key is present in both inputs, the second one will be dominant so the first value will be overwritten unless you provide a merge function",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict1", "list", "first input dictionary that has to be changed. You must not use this value again.", nil},
			DeclarationParameter{"dict2", "list", "input dictionary that contains the new values that have to be added", nil},
			DeclarationParameter{"merge", "func", "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value from dict2. It must return the merged value that shall be pysically stored in the new dictionary.", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("merge_assoc_mut")},
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
	})

	// _mut variants: optimizer-only, forbidden from .scm code
	// Tier 1: same-length, zero-copy

	Declare(&Globalenv, &Declaration{
		"map_mut", "in-place map (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"list", "list", "owned list to map in-place", nil},
			{"map", "func", "map function", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(v)
			}
			return NewSlice(list)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: unsupported builtin: SliceData */, /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
	})

	Declare(&Globalenv, &Declaration{
		"mapIndex_mut", "in-place mapIndex (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"list", "list", "owned list to map in-place", nil},
			{"map", "func", "map function func(i, any)->any", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(list)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: unsupported builtin: SliceData */, /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
	})

	Declare(&Globalenv, &Declaration{
		"map_assoc_mut", "in-place map_assoc (optimizer-only, slice path only)",
		2, 2,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to map in-place", nil},
			{"map", "func", "map function func(key, value)->value", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	// Tier 2: shrinking, write-cursor

	Declare(&Globalenv, &Declaration{
		"filter_mut", "in-place filter (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"list", "list", "owned list to filter in-place", nil},
			{"condition", "func", "filter condition func(any)->bool", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: unsupported builtin: SliceData */, /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
	})

	Declare(&Globalenv, &Declaration{
		"filter_assoc_mut", "in-place filter_assoc (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to filter in-place", nil},
			{"condition", "func", "filter function func(key, value)->bool", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"extract_assoc_mut", "in-place extract_assoc (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to extract from in-place", nil},
			{"map", "func", "func(key, value)->any that extracts each element", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"set_assoc_mut", "in-place set_assoc (optimizer-only, mutates input directly)",
		3, 4,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to mutate", nil},
			{"key", "string", "key to set", nil},
			{"value", "any", "new value", nil},
			{"merge", "func", "(optional) merge function", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			var mergeFn func(Scmer, Scmer) Scmer
			if len(a) > 3 {
				mfn := OptimizeProcToSerialFunction(a[3])
				mergeFn = func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
			}
			// Always operate on FastDict; promote slice/nil inputs
			var fd *FastDict
			if a[0].IsFastDict() {
				fd = a[0].FastDict()
			} else if a[0].IsSlice() {
				list := a[0].Slice()
				fd = NewFastDictValue(len(list)/2 + 4)
				for i := 0; i+1 < len(list); i += 2 {
					fd.Set(list[i], list[i+1], nil)
				}
			} else {
				fd = NewFastDictValue(8)
			}
			fd.Set(a[1], a[2], mergeFn)
			return NewFastDict(fd)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d0)
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() > 3)}
			} else {
				r1 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d0.Reg, 3)
				ctx.W.EmitSetcc(r1, CcG)
				d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d1)
			}
			ctx.FreeDesc(&d0)
			d2 := d1
			ctx.EnsureDesc(&d2)
			if d2.Loc != LocImm && d2.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d2.Loc == LocImm {
				if d2.Imm.Bool() {
					ctx.W.MarkLabel(lbl4)
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.MarkLabel(lbl5)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl4)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl5)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
				ctx.W.EmitJmp(lbl3)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl3)
			d3 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 := args[0]
			d6 := d4
			d6.ID = 0
			d5 := ctx.EmitTagEquals(&d6, tagFastDict, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d4)
			d7 := d5
			ctx.EnsureDesc(&d7)
			if d7.Loc != LocImm && d7.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.MarkLabel(lbl8)
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.MarkLabel(lbl9)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl8)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl2)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d8 := args[3]
			d9 := ctx.EmitGoCallScalar(GoFuncAddr(OptimizeProcToSerialFunction), []JITValueDesc{d8}, 1)
			ctx.FreeDesc(&d8)
			ctx.FreeDesc(&d9)
			d10 := ctx.EmitGoCallScalar(GoFuncAddr(JITBuildMergeClosure), []JITValueDesc{d9}, 1)
			d11 := d10
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d11)
			ctx.EmitStoreToStack(d11, 0)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl7)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d12 := args[0]
			d14 := d12
			d14.ID = 0
			d13 := ctx.EmitTagEquals(&d14, tagSlice, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d12)
			d15 := d13
			ctx.EnsureDesc(&d15)
			if d15.Loc != LocImm && d15.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d15.Loc == LocImm {
				if d15.Imm.Bool() {
					ctx.W.MarkLabel(lbl12)
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d15.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl12)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d13)
			ctx.W.MarkLabel(lbl6)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d16 := args[0]
			var d17 JITValueDesc
			if d16.Loc == LocImm {
				panic("FastDict: LocImm not expected at JIT compile time")
			} else {
				ctx.FreeReg(d16.Reg2)
				d17 = JITValueDesc{Loc: LocReg, Reg: d16.Reg}
				ctx.BindReg(d16.Reg, &d17)
			}
			ctx.FreeDesc(&d16)
			lbl14 := ctx.W.ReserveLabel()
			d18 := d17
			if d18.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d18)
			ctx.EmitStoreToStack(d18, 8)
			ctx.W.MarkLabel(lbl14)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d19 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d20 := args[1]
			d21 := args[2]
			ctx.EnsureDesc(&d3)
			ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d19, d20, d21, d3})
			ctx.FreeDesc(&d20)
			ctx.FreeDesc(&d21)
			ctx.FreeDesc(&d3)
			var d22 JITValueDesc
			if d19.Loc == LocImm {
				panic("NewFastDict: LocImm not expected at JIT compile time")
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(r2, uint64(tagFastDict) << 48)
				d22 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d19.Reg, Reg2: r2}
				ctx.BindReg(d19.Reg, &d22)
				ctx.BindReg(r2, &d22)
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d22)
			if d22.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d22, &result)
				result.Type = d22.Type
			} else {
				switch d22.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d22)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d22)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d22)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d22, &result)
					result.Type = d22.Type
				}
			}
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl11)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(8)}
			d24 := ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d23}, 1)
			d25 := d24
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			ctx.EmitStoreToStack(d25, 8)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl10)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d26 := args[0]
			var d27 JITValueDesc
			if d26.Loc == LocImm {
				slice := d26.Imm.Slice()
				d27 = JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: NewInt(int64(len(slice)))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r3, d26.Reg2)
				ctx.W.EmitShlRegImm8(r3, 16)
				ctx.W.EmitShrRegImm8(r3, 16)
				ctx.FreeReg(d26.Reg2)
				d27 = JITValueDesc{Loc: LocRegPair, Reg: d26.Reg, Reg2: r3}
				ctx.BindReg(d26.Reg, &d27)
				ctx.BindReg(r3, &d27)
			}
			ctx.FreeDesc(&d26)
			var d28 JITValueDesc
			if d27.Loc == LocImm {
				d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d27.StackOff))}
			} else {
				ctx.EnsureDesc(&d27)
				if d27.Loc == LocRegPair {
					d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg2}
					ctx.BindReg(d27.Reg2, &d28)
					ctx.BindReg(d27.Reg2, &d28)
				} else if d27.Loc == LocReg {
					d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg}
					ctx.BindReg(d27.Reg, &d28)
					ctx.BindReg(d27.Reg, &d28)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d28)
			var d29 JITValueDesc
			if d28.Loc == LocImm {
				d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d28.Imm.Int() / 2)}
			} else {
				ctx.W.EmitShrRegImm8(d28.Reg, 1)
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg}
				ctx.BindReg(d28.Reg, &d29)
			}
			if d29.Loc == LocReg && d28.Loc == LocReg && d29.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			var d30 JITValueDesc
			if d29.Loc == LocImm {
				d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int() + 4)}
			} else {
				ctx.W.EmitAddRegImm32(d29.Reg, int32(4))
				d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg}
				ctx.BindReg(d29.Reg, &d30)
			}
			if d30.Loc == LocReg && d29.Loc == LocReg && d30.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d30)
			d31 := ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d30}, 1)
			ctx.FreeDesc(&d30)
			lbl15 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 16)
			ctx.W.MarkLabel(lbl15)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d32 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d32)
			var d33 JITValueDesc
			if d32.Loc == LocImm {
				d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d32.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(scratch, d32.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			}
			if d33.Loc == LocReg && d32.Loc == LocReg && d33.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = LocNone
			}
			var d34 JITValueDesc
			if d27.Loc == LocImm {
				d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d27.StackOff))}
			} else {
				ctx.EnsureDesc(&d27)
				if d27.Loc == LocRegPair {
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg2}
					ctx.BindReg(d27.Reg2, &d34)
					ctx.BindReg(d27.Reg2, &d34)
				} else if d27.Loc == LocReg {
					d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg}
					ctx.BindReg(d27.Reg, &d34)
					ctx.BindReg(d27.Reg, &d34)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d34)
			var d35 JITValueDesc
			if d33.Loc == LocImm && d34.Loc == LocImm {
				d35 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d33.Imm.Int() < d34.Imm.Int())}
			} else if d34.Loc == LocImm {
				r4 := ctx.AllocReg()
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d33.Reg, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitCmpInt64(d33.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r4, CcL)
				d35 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d35)
			} else if d33.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d33.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d34.Reg)
				ctx.W.EmitSetcc(r5, CcL)
				d35 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d35)
			} else {
				r6 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d33.Reg, d34.Reg)
				ctx.W.EmitSetcc(r6, CcL)
				d35 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d35)
			}
			ctx.FreeDesc(&d33)
			ctx.FreeDesc(&d34)
			d36 := d35
			ctx.EnsureDesc(&d36)
			if d36.Loc != LocImm && d36.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d36.Loc == LocImm {
				if d36.Imm.Bool() {
					ctx.W.MarkLabel(lbl17)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl18)
			d37 := d31
			if d37.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d37)
			ctx.EmitStoreToStack(d37, 8)
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d36.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl17)
				ctx.W.EmitJmp(lbl18)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl18)
			d38 := d31
			if d38.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d38)
			ctx.EmitStoreToStack(d38, 8)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl16)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d32 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d32)
			r7 := ctx.AllocReg()
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d27)
			if d32.Loc == LocImm {
				ctx.W.EmitMovRegImm64(r7, uint64(d32.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r7, d32.Reg)
				ctx.W.EmitShlRegImm8(r7, 4)
			}
			if d27.Loc == LocImm {
				ctx.W.EmitMovRegImm64(RegR11, uint64(d27.Imm.Int()))
				ctx.W.EmitAddInt64(r7, RegR11)
			} else {
				ctx.W.EmitAddInt64(r7, d27.Reg)
			}
			r8 := ctx.AllocRegExcept(r7)
			r9 := ctx.AllocRegExcept(r7, r8)
			ctx.W.EmitMovRegMem(r8, r7, 0)
			ctx.W.EmitMovRegMem(r9, r7, 8)
			ctx.FreeReg(r7)
			d39 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r8, Reg2: r9}
			ctx.BindReg(r8, &d39)
			ctx.BindReg(r9, &d39)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d32)
			var d40 JITValueDesc
			if d32.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d32.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(scratch, d32.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			}
			if d40.Loc == LocReg && d32.Loc == LocReg && d40.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = LocNone
			}
			ctx.EnsureDesc(&d40)
			r10 := ctx.AllocReg()
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d27)
			if d40.Loc == LocImm {
				ctx.W.EmitMovRegImm64(r10, uint64(d40.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r10, d40.Reg)
				ctx.W.EmitShlRegImm8(r10, 4)
			}
			if d27.Loc == LocImm {
				ctx.W.EmitMovRegImm64(RegR11, uint64(d27.Imm.Int()))
				ctx.W.EmitAddInt64(r10, RegR11)
			} else {
				ctx.W.EmitAddInt64(r10, d27.Reg)
			}
			r11 := ctx.AllocRegExcept(r10)
			r12 := ctx.AllocRegExcept(r10, r11)
			ctx.W.EmitMovRegMem(r11, r10, 0)
			ctx.W.EmitMovRegMem(r12, r10, 8)
			ctx.FreeReg(r10)
			d41 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r11, Reg2: r12}
			ctx.BindReg(r11, &d41)
			ctx.BindReg(r12, &d41)
			ctx.FreeDesc(&d40)
			d42 := JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d31, d39, d41, d42})
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d32)
			var d43 JITValueDesc
			if d32.Loc == LocImm {
				d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d32.Imm.Int() + 2)}
			} else {
				ctx.W.EmitAddRegImm32(d32.Reg, int32(2))
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d32.Reg}
				ctx.BindReg(d32.Reg, &d43)
			}
			if d43.Loc == LocReg && d32.Loc == LocReg && d43.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = LocNone
			}
			ctx.FreeDesc(&d32)
			d44 := d43
			if d44.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d44)
			ctx.EmitStoreToStack(d44, 16)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(24))
			ctx.W.EmitAddRSP32(int32(24))
			return result
		},
	})

	// Tier 3: append/grow

	Declare(&Globalenv, &Declaration{
		"append_mut", "in-place append (optimizer-only)",
		2, 1000,
		[]DeclarationParameter{
			{"list", "list", "owned base list", nil},
			{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			base := asSlice(a[0], "append_mut")
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"append_unique_mut", "in-place append_unique (optimizer-only)",
		2, 1000,
		[]DeclarationParameter{
			{"list", "list", "owned base list", nil},
			{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t1[:] */, /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"merge_assoc_mut", "in-place merge_assoc (optimizer-only)",
		2, 3,
		[]DeclarationParameter{
			{"dict1", "list", "owned first dictionary", nil},
			{"dict2", "list", "dictionary with new values", nil},
			{"merge", "func", "(optional) merge function", nil},
		}, "list",
		func(a ...Scmer) Scmer {
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
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
	})
}
