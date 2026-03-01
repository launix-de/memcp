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

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"strings"
)

func init_alu() {
	// string functions
	DeclareTitle("Arithmetic / Logic")

	Declare(&Globalenv, &Declaration{
		"int?", "tells if the value is a integer",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagInt)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			d0 := args[0]
			d1 := ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == 4)}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
			}
			if d2.Loc == LocImm {
				if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: d2.Imm} }
				ctx.W.EmitMakeBool(result, d2)
			} else {
				if result.Loc == LocAny { return d2 }
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"number?", "tells if the value is a number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			tag := a[0].GetTag()
			return NewBool(tag == tagFloat || tag == tagInt || tag == tagDate)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			r0 := ctx.AllocReg()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d1 := ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == 3)}
			} else {
				r1 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 3)
				ctx.W.EmitSetcc(r1, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
			}
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d2.Loc == LocImm {
				if d2.Imm.Bool() {
					ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Imm: NewInt(1)})
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Imm: NewInt(1)})
				ctx.W.EmitJmp(lbl1)
			}
			ctx.W.MarkLabel(lbl2)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == 4)}
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r2, CcE)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d3.Loc == LocImm {
				if d3.Imm.Bool() {
					ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Imm: NewInt(1)})
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d3.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl5)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl5)
				ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Imm: NewInt(1)})
				ctx.W.EmitJmp(lbl1)
			}
			ctx.W.MarkLabel(lbl1)
			d4 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r0}
			ctx.W.EmitMakeBool(result, d4)
			if d4.Loc == LocReg { ctx.FreeReg(d4.Reg) }
			result.Type = tagBool
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl4)
			var d5 JITValueDesc
			if d1.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == 15)}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 15)
				ctx.W.EmitSetcc(r3, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
			}
			ctx.EmitMovToReg(r0, d5)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"+", "adds two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values to add", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Fast path: accumulate ints until first non-int, then promote to float if needed
			var sumInt int64
			i := 0
			for i < len(a) {
				v := a[i]
				if v.IsInt() {
					sumInt += v.Int()
					i++
					continue
				}
				break
			}
			if i == len(a) {
				return NewInt(sumInt)
			}
			// Promote to float and continue
			sumFloat := float64(sumInt)
			for ; i < len(a); i++ {
				v := a[i]
				if v.IsNil() {
					return NewNil()
				}
				sumFloat += v.Float()
			}
			return NewFloat(sumFloat)
		},
		true, false, &TypeDescriptor{Optimize: optimizeAssociative},
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			r0 := ctx.AllocReg()
			r1 := ctx.AllocReg()
			r2 := ctx.AllocReg()
			r3 := ctx.AllocReg()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Imm: NewInt(0)})
			ctx.EmitMovToReg(r1, JITValueDesc{Loc: LocImm, Imm: NewInt(0)})
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r0}
			d1 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r1}
			d2 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d3 JITValueDesc
			if d1.Loc == LocImm && d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d2.Imm.Int())}
			} else if d2.Loc == LocImm {
				r4 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
				ctx.W.EmitSetcc(r4, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
			} else if d1.Loc == LocImm {
				r5 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d2.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r5, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
			} else {
				r6 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d1.Reg, d2.Reg)
				ctx.W.EmitSetcc(r6, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d3.Loc == LocImm {
				if d3.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d3.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.W.MarkLabel(lbl3)
			d4 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d5 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == d4.Imm.Int())}
			} else if d4.Loc == LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, int32(d4.Imm.Int()))
				ctx.W.EmitSetcc(r7, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
			} else if d1.Loc == LocImm {
				r8 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d4.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r8, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
			} else {
				r9 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d1.Reg, d4.Reg)
				ctx.W.EmitSetcc(r9, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
			}
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl7)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.W.MarkLabel(lbl2)
			r10 := ctx.AllocReg()
			ctx.W.emitMovRegReg(r10, d1.Reg)
			ctx.W.EmitShlRegImm8(r10, 4)
			ctx.W.EmitAddInt64(r10, ctx.SliceBase)
			r11 := ctx.AllocReg()
			r12 := ctx.AllocReg()
			ctx.W.emitMovRegMem(r11, r10, 0)
			ctx.W.emitMovRegMem(r12, r10, 8)
			ctx.FreeReg(r10)
			d6 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r11, Reg2: r12}
			d7 := ctx.EmitTagEquals(&d6, tagInt, JITValueDesc{Loc: LocAny})
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.W.MarkLabel(lbl6)
			var d8 JITValueDesc
			if d0.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d0.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(d0.Reg, d0.Reg)
				d8 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
			}
			ctx.EmitMovToReg(r2, d1)
			ctx.EmitMovToReg(r3, d8)
			lbl10 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl5)
			ctx.W.EmitMakeInt(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl8)
			var d9 JITValueDesc
			if d6.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int())}
			} else {
				ctx.FreeReg(d6.Reg)
				d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2}
			}
			var d10 JITValueDesc
			if d0.Loc == LocImm && d9.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d9.Imm.Int())}
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d9.Reg)
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d9.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitAddInt64(d0.Reg, scratch)
				ctx.FreeReg(scratch)
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d0.Reg}
			} else {
				ctx.W.EmitAddInt64(d0.Reg, d9.Reg)
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d0.Reg}
			}
			var d11 JITValueDesc
			if d1.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d1.Reg, scratch)
				ctx.FreeReg(scratch)
				d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
			}
			ctx.EmitMovToReg(r0, d10)
			ctx.EmitMovToReg(r1, d11)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl10)
			d12 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r2}
			d13 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r3}
			d14 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d15 JITValueDesc
			if d12.Loc == LocImm && d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < d14.Imm.Int())}
			} else if d14.Loc == LocImm {
				r13 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d12.Reg, int32(d14.Imm.Int()))
				ctx.W.EmitSetcc(r13, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
			} else if d12.Loc == LocImm {
				r14 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d14.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r14, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
			} else {
				r15 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d12.Reg, d14.Reg)
				ctx.W.EmitSetcc(r15, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d15.Loc == LocImm {
				if d15.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d15.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.W.MarkLabel(lbl12)
			ctx.W.EmitMakeFloat(result, d13)
			if d13.Loc == LocReg { ctx.FreeReg(d13.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl11)
			r16 := ctx.AllocReg()
			ctx.W.emitMovRegReg(r16, d12.Reg)
			ctx.W.EmitShlRegImm8(r16, 4)
			ctx.W.EmitAddInt64(r16, ctx.SliceBase)
			r17 := ctx.AllocReg()
			r18 := ctx.AllocReg()
			ctx.W.emitMovRegMem(r17, r16, 0)
			ctx.W.emitMovRegMem(r18, r16, 8)
			ctx.FreeReg(r16)
			d16 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r17, Reg2: r18}
			d17 := ctx.EmitTagEquals(&d16, tagNil, JITValueDesc{Loc: LocAny})
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d17.Loc == LocImm {
				if d17.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d17.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.W.MarkLabel(lbl15)
			var d18 JITValueDesc
			if d16.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d16.Imm.Float())}
			} else {
				ctx.FreeReg(d16.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d16.Reg2}
			}
			var d19 JITValueDesc
			if d13.Loc == LocImm && d18.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + d18.Imm.Int())}
			} else if d13.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d18.Reg)
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d18.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
				ctx.W.EmitAddInt64(d13.Reg, scratch)
				ctx.FreeReg(scratch)
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg}
			} else {
				ctx.W.EmitAddInt64(d13.Reg, d18.Reg)
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg}
			}
			var d20 JITValueDesc
			if d12.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d12.Reg, scratch)
				ctx.FreeReg(scratch)
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg}
			}
			ctx.EmitMovToReg(r2, d20)
			ctx.EmitMovToReg(r3, d19)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl14)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"-", "subtracts two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			// Int-first, then promote to float if needed
			if a[0].IsInt() {
				diffInt := a[0].Int()
				i := 1
				for i < len(a) && a[i].IsInt() {
					diffInt -= a[i].Int()
					i++
				}
				if i == len(a) {
					return NewInt(diffInt)
				}
				diffFloat := float64(diffInt)
				for ; i < len(a); i++ {
					diffFloat -= a[i].Float()
				}
				return NewFloat(diffFloat)
			}
			// Float mode from the start
			diffFloat := a[0].Float()
			for i := 1; i < len(a); i++ {
				diffFloat -= a[i].Float()
			}
			return NewFloat(diffFloat)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			r0 := ctx.AllocReg()
			r1 := ctx.AllocReg()
			r2 := ctx.AllocReg()
			r3 := ctx.AllocReg()
			r4 := ctx.AllocReg()
			r5 := ctx.AllocReg()
			r6 := ctx.AllocReg()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Imm: NewInt(-1)})
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			d1 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r0}
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d1.Reg, scratch)
				ctx.FreeReg(scratch)
				d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
			}
			var d3 JITValueDesc
			if d2.Loc == LocImm && d0.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d0.Imm.Int())}
			} else if d0.Loc == LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d2.Reg, int32(d0.Imm.Int()))
				ctx.W.EmitSetcc(r7, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
			} else if d2.Loc == LocImm {
				r8 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d0.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r8, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
			} else {
				r9 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d2.Reg, d0.Reg)
				ctx.W.EmitSetcc(r9, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d3.Loc == LocImm {
				if d3.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d3.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.W.MarkLabel(lbl3)
			d4 := args[0]
			d5 := ctx.EmitTagEquals(&d4, tagInt, JITValueDesc{Loc: LocAny})
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl7)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.W.MarkLabel(lbl2)
			r10 := ctx.AllocReg()
			ctx.W.emitMovRegReg(r10, d2.Reg)
			ctx.W.EmitShlRegImm8(r10, 4)
			ctx.W.EmitAddInt64(r10, ctx.SliceBase)
			r11 := ctx.AllocReg()
			r12 := ctx.AllocReg()
			ctx.W.emitMovRegMem(r11, r10, 0)
			ctx.W.emitMovRegMem(r12, r10, 8)
			ctx.FreeReg(r10)
			d6 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r11, Reg2: r12}
			d7 := ctx.EmitTagEquals(&d6, tagNil, JITValueDesc{Loc: LocAny})
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.EmitMovToReg(r0, d2)
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
				ctx.EmitMovToReg(r0, d2)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.W.MarkLabel(lbl6)
			d8 := args[0]
			var d9 JITValueDesc
			if d8.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d8.Imm.Float())}
			} else {
				ctx.FreeReg(d8.Reg)
				d9 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d8.Reg2}
			}
			ctx.EmitMovToReg(r5, d9)
			ctx.EmitMovToReg(r6, JITValueDesc{Loc: LocImm, Imm: NewInt(1)})
			lbl10 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl5)
			d10 := args[0]
			var d11 JITValueDesc
			if d10.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d10.Imm.Int())}
			} else {
				ctx.FreeReg(d10.Reg)
				d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg2}
			}
			ctx.EmitMovToReg(r1, d11)
			ctx.EmitMovToReg(r2, JITValueDesc{Loc: LocImm, Imm: NewInt(1)})
			lbl11 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl8)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			d12 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r5}
			d13 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r6}
			d14 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d15 JITValueDesc
			if d13.Loc == LocImm && d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d14.Imm.Int())}
			} else if d14.Loc == LocImm {
				r13 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d13.Reg, int32(d14.Imm.Int()))
				ctx.W.EmitSetcc(r13, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
			} else if d13.Loc == LocImm {
				r14 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d14.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r14, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
			} else {
				r15 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d13.Reg, d14.Reg)
				ctx.W.EmitSetcc(r15, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d15.Loc == LocImm {
				if d15.Imm.Bool() {
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d15.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.W.MarkLabel(lbl11)
			d16 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r1}
			d17 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r2}
			d18 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d19 JITValueDesc
			if d17.Loc == LocImm && d18.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() < d18.Imm.Int())}
			} else if d18.Loc == LocImm {
				r16 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d17.Reg, int32(d18.Imm.Int()))
				ctx.W.EmitSetcc(r16, CcL)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
			} else if d17.Loc == LocImm {
				r17 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d18.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r17, CcL)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
			} else {
				r18 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d17.Reg, d18.Reg)
				ctx.W.EmitSetcc(r18, CcL)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			if d19.Loc == LocImm {
				if d19.Imm.Bool() {
					ctx.W.EmitJmp(lbl15)
				} else {
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d19.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl17)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl15)
			}
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitMakeFloat(result, d12)
			if d12.Loc == LocReg { ctx.FreeReg(d12.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			r19 := ctx.AllocReg()
			ctx.W.emitMovRegReg(r19, d13.Reg)
			ctx.W.EmitShlRegImm8(r19, 4)
			ctx.W.EmitAddInt64(r19, ctx.SliceBase)
			r20 := ctx.AllocReg()
			r21 := ctx.AllocReg()
			ctx.W.emitMovRegMem(r20, r19, 0)
			ctx.W.emitMovRegMem(r21, r19, 8)
			ctx.FreeReg(r19)
			d20 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r20, Reg2: r21}
			var d21 JITValueDesc
			if d20.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d20.Imm.Float())}
			} else {
				ctx.FreeReg(d20.Reg)
				d21 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d20.Reg2}
			}
			var d22 JITValueDesc
			if d12.Loc == LocImm && d21.Loc == LocImm {
				d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() - d21.Imm.Int())}
			} else if d12.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d21.Reg)
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d21.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
				ctx.W.EmitSubInt64(d12.Reg, scratch)
				ctx.FreeReg(scratch)
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg}
			} else {
				ctx.W.EmitSubInt64(d12.Reg, d21.Reg)
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg}
			}
			var d23 JITValueDesc
			if d13.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d13.Reg, scratch)
				ctx.FreeReg(scratch)
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg}
			}
			ctx.EmitMovToReg(r5, d22)
			ctx.EmitMovToReg(r6, d23)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl16)
			d24 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d25 JITValueDesc
			if d17.Loc == LocImm && d24.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() == d24.Imm.Int())}
			} else if d24.Loc == LocImm {
				r22 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d17.Reg, int32(d24.Imm.Int()))
				ctx.W.EmitSetcc(r22, CcE)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
			} else if d17.Loc == LocImm {
				r23 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d24.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r23, CcE)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
			} else {
				r24 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d17.Reg, d24.Reg)
				ctx.W.EmitSetcc(r24, CcE)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
			}
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			if d25.Loc == LocImm {
				if d25.Imm.Bool() {
					ctx.W.EmitJmp(lbl18)
				} else {
					ctx.W.EmitJmp(lbl19)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d25.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl20)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl20)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.W.MarkLabel(lbl15)
			r25 := ctx.AllocReg()
			ctx.W.emitMovRegReg(r25, d17.Reg)
			ctx.W.EmitShlRegImm8(r25, 4)
			ctx.W.EmitAddInt64(r25, ctx.SliceBase)
			r26 := ctx.AllocReg()
			r27 := ctx.AllocReg()
			ctx.W.emitMovRegMem(r26, r25, 0)
			ctx.W.emitMovRegMem(r27, r25, 8)
			ctx.FreeReg(r25)
			d26 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r26, Reg2: r27}
			d27 := ctx.EmitTagEquals(&d26, tagInt, JITValueDesc{Loc: LocAny})
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d27.Loc == LocImm {
				if d27.Imm.Bool() {
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl22)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl21)
			}
			ctx.W.MarkLabel(lbl19)
			var d28 JITValueDesc
			if d16.Loc == LocImm {
				d28 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d16.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(d16.Reg, d16.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d16.Reg}
			}
			ctx.EmitMovToReg(r3, d17)
			ctx.EmitMovToReg(r4, d28)
			lbl23 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl18)
			ctx.W.EmitMakeInt(result, d16)
			if d16.Loc == LocReg { ctx.FreeReg(d16.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl21)
			r28 := ctx.AllocReg()
			ctx.W.emitMovRegReg(r28, d17.Reg)
			ctx.W.EmitShlRegImm8(r28, 4)
			ctx.W.EmitAddInt64(r28, ctx.SliceBase)
			r29 := ctx.AllocReg()
			r30 := ctx.AllocReg()
			ctx.W.emitMovRegMem(r29, r28, 0)
			ctx.W.emitMovRegMem(r30, r28, 8)
			ctx.FreeReg(r28)
			d29 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r29, Reg2: r30}
			var d30 JITValueDesc
			if d29.Loc == LocImm {
				d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int())}
			} else {
				ctx.FreeReg(d29.Reg)
				d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg2}
			}
			var d31 JITValueDesc
			if d16.Loc == LocImm && d30.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d16.Imm.Int() - d30.Imm.Int())}
			} else if d16.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d30.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d30.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitSubInt64(d16.Reg, scratch)
				ctx.FreeReg(scratch)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg}
			} else {
				ctx.W.EmitSubInt64(d16.Reg, d30.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg}
			}
			var d32 JITValueDesc
			if d17.Loc == LocImm {
				d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d17.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d17.Reg, scratch)
				ctx.FreeReg(scratch)
				d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg}
			}
			ctx.EmitMovToReg(r1, d31)
			ctx.EmitMovToReg(r2, d32)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl23)
			d33 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r3}
			d34 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r4}
			d35 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d36 JITValueDesc
			if d33.Loc == LocImm && d35.Loc == LocImm {
				d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d33.Imm.Int() < d35.Imm.Int())}
			} else if d35.Loc == LocImm {
				r31 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d33.Reg, int32(d35.Imm.Int()))
				ctx.W.EmitSetcc(r31, CcL)
				d36 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r31}
			} else if d33.Loc == LocImm {
				r32 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d35.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r32, CcL)
				d36 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r32}
			} else {
				r33 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d33.Reg, d35.Reg)
				ctx.W.EmitSetcc(r33, CcL)
				d36 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r33}
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d36.Loc == LocImm {
				if d36.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d36.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl26)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.W.MarkLabel(lbl25)
			ctx.W.EmitMakeFloat(result, d34)
			if d34.Loc == LocReg { ctx.FreeReg(d34.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl24)
			r34 := ctx.AllocReg()
			ctx.W.emitMovRegReg(r34, d33.Reg)
			ctx.W.EmitShlRegImm8(r34, 4)
			ctx.W.EmitAddInt64(r34, ctx.SliceBase)
			r35 := ctx.AllocReg()
			r36 := ctx.AllocReg()
			ctx.W.emitMovRegMem(r35, r34, 0)
			ctx.W.emitMovRegMem(r36, r34, 8)
			ctx.FreeReg(r34)
			d37 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r35, Reg2: r36}
			var d38 JITValueDesc
			if d37.Loc == LocImm {
				d38 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d37.Imm.Float())}
			} else {
				ctx.FreeReg(d37.Reg)
				d38 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d37.Reg2}
			}
			var d39 JITValueDesc
			if d34.Loc == LocImm && d38.Loc == LocImm {
				d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d34.Imm.Int() - d38.Imm.Int())}
			} else if d34.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d38.Reg)
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d38.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitSubInt64(d34.Reg, scratch)
				ctx.FreeReg(scratch)
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg}
			} else {
				ctx.W.EmitSubInt64(d34.Reg, d38.Reg)
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg}
			}
			var d40 JITValueDesc
			if d33.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d33.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d33.Reg, scratch)
				ctx.FreeReg(scratch)
				d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg}
			}
			ctx.EmitMovToReg(r3, d40)
			ctx.EmitMovToReg(r4, d39)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"*", "multiplies two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit (SQL-style): if any arg is nil, result is nil
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			// Try integer mode: treat float operands with zero fractional part as integers
			prodInt := int64(1)
			i := 0
			for ; i < len(a); i++ {
				v := a[i]
				if v.IsInt() {
					prodInt *= v.Int()
					continue
				}
				if v.IsFloat() {
					f := v.Float()
					if f == math.Trunc(f) {
						prodInt *= int64(f)
						continue
					}
				}
				break // non-integer number encountered -> switch to float mode
			}
			if i == len(a) {
				return NewInt(prodInt)
			}
			// Float mode: include any prior integer product and continue in float
			prodFloat := float64(prodInt)
			for ; i < len(a); i++ {
				prodFloat *= a[i].Float()
			}
			return NewFloat(prodFloat)
		},
		true, false, &TypeDescriptor{Optimize: optimizeAssociative},
		nil /* TODO: unsupported call: math.Trunc(t22) */,
	})
	Declare(&Globalenv, &Declaration{
		"/", "divides two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			v := a[0].Float()
			for _, i := range a[1:] {
				v /= i.Float()
			}
			return NewFloat(v)
		},
		true, false, nil,
		nil /* TODO: Slice: slice a[1:int:] */,
	})
	Declare(&Globalenv, &Declaration{
		"<=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!Less(a[1], a[0]))
		},
		true, false, nil,
		nil /* TODO: unsupported call: Less(t1, t3) */,
	})
	Declare(&Globalenv, &Declaration{
		"<", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Less(a[0], a[1]))
		},
		true, false, nil,
		nil /* TODO: unsupported call: Less(t1, t3) */,
	})
	Declare(&Globalenv, &Declaration{
		">", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Less(a[1], a[0]))
		},
		true, false, nil,
		nil /* TODO: unsupported call: Less(t1, t3) */,
	})
	Declare(&Globalenv, &Declaration{
		">=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!Less(a[0], a[1]))
		},
		true, false, nil,
		nil /* TODO: unsupported call: Less(t1, t3) */,
	})
	Declare(&Globalenv, &Declaration{
		"equal?", "compares two values of the same type, (equal? nil nil) is true",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Equal(a[0], a[1]))
		},
		true, false, nil,
		nil /* TODO: unsupported call: Equal(t1, t3) */,
	})
	Declare(&Globalenv, &Declaration{
		"equal??", "performs a SQL compliant sloppy equality check on primitive values (number, int, string, bool. nil), strings are compared case insensitive, (equal? nil nil) is nil",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return EqualSQL(a[0], a[1])
		},
		true, false, nil,
		nil /* TODO: unsupported call: EqualSQL(t1, t3) */,
	})
	Declare(&Globalenv, &Declaration{
		"equal_collate", "performs SQL equality with a specified collation (e.g. *_ci case-insensitive, *_bin case-sensitive); returns nil if either arg is nil",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"a", "any", "left side", nil},
			DeclarationParameter{"b", "any", "right side", nil},
			DeclarationParameter{"collation", "string", "collation name", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			coll := strings.ToLower(String(a[2]))
			ta := a[0].GetTag()
			tb := a[1].GetTag()
			if (ta == tagString || ta == tagSymbol) && (tb == tagString || tb == tagSymbol) {
				as := a[0].String()
				bs := a[1].String()
				if strings.Contains(coll, "_ci") {
					return NewBool(strings.EqualFold(as, bs))
				}
				return NewBool(as == bs)
			}
			return EqualSQL(a[0], a[1])
		},
		true, false, nil,
		nil /* TODO: unsupported call: String(t5) */,
	})
	Declare(&Globalenv, &Declaration{
		"notequal_collate", "performs SQL inequality with a specified collation; returns nil if either arg is nil",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"a", "any", "left side", nil},
			DeclarationParameter{"b", "any", "right side", nil},
			DeclarationParameter{"collation", "string", "collation name", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			r := Globalenv.Vars["equal_collate"].Func()(a[0], a[1], a[2])
			if r.IsNil() {
				return r
			}
			return NewBool(!r.Bool())
		},
		true, false, nil,
		nil /* TODO: FieldAddr: &Globalenv.Vars [#0] */,
	})
	Declare(&Globalenv, &Declaration{
		"!", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).Bool(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"not", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
		},
		true, false, nil,
		nil /* TODO: unsupported call: (Scmer).Bool(t1) */,
	})
	Declare(&Globalenv, &Declaration{
		"nil?", "returns true if value is nil",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].IsNil())
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			d0 := args[0]
			d1 := ctx.EmitTagEquals(&d0, tagNil, JITValueDesc{Loc: LocAny})
			if d1.Loc == LocImm {
				if result.Loc == LocAny { return JITValueDesc{Loc: LocImm, Imm: d1.Imm} }
				ctx.W.EmitMakeBool(result, d1)
			} else {
				if result.Loc == LocAny { return d1 }
				ctx.W.EmitMakeBool(result, d1)
				ctx.FreeReg(d1.Reg)
			}
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"min", "returns the smallest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value", nil},
		}, "number|string",
		func(a ...Scmer) Scmer {
			var result Scmer
			for _, v := range a {
				if result.IsNil() {
					result = v
				} else if !v.IsNil() && Less(v, result) {
					result = v
				}
			}
			return result
		},
		true, false, nil,
		nil /* TODO: unsupported return type for phi [0: Scmer{}:Scmer, 4: t6, 5: t1, 7: t1, 6: t6] #result */,
	})
	Declare(&Globalenv, &Declaration{
		"max", "returns the highest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value", nil},
		}, "number|string",
		func(a ...Scmer) Scmer {
			var result Scmer
			for _, v := range a {
				if result.IsNil() {
					result = v
				} else if !v.IsNil() && Less(result, v) {
					result = v
				}
			}
			return result
		},
		true, false, nil,
		nil /* TODO: unsupported return type for phi [0: Scmer{}:Scmer, 4: t6, 5: t1, 7: t1, 6: t6] #result */,
	})
	Declare(&Globalenv, &Declaration{
		"floor", "rounds the number down",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Floor(a[0].Float()))
		},
		true, false, nil,
		nil /* TODO: unsupported call: math.Floor(t2) */,
	})
	Declare(&Globalenv, &Declaration{
		"ceil", "rounds the number up",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Ceil(a[0].Float()))
		},
		true, false, nil,
		nil /* TODO: unsupported call: math.Ceil(t2) */,
	})
	Declare(&Globalenv, &Declaration{
		"round", "rounds the number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Round(a[0].Float()))
		},
		true, false, nil,
		nil /* TODO: unsupported call: math.Round(t2) */,
	})
	Declare(&Globalenv, &Declaration{
		"sql_abs", "SQL ABS(): returns absolute value, NULL-safe",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			v := a[0].Float()
			if v < 0 {
				v = -v
			}
			// preserve int type
			if ToInt(a[0]) == int(v) && a[0].Float() == v {
				return NewInt(int64(v))
			}
			return NewFloat(v)
		},
		true, false, nil,
		nil /* TODO: unsupported call: ToInt(t11) */,
	})
	Declare(&Globalenv, &Declaration{
		"sqrt", "returns the square root of a number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			v := a[0].Float()
			if v < 0 {
				return NewNil()
			}
			return NewFloat(math.Sqrt(v))
		},
		true, false, nil,
		nil /* TODO: unsupported call: math.Sqrt(t6) */,
	})
	Declare(&Globalenv, &Declaration{
		"sql_rand", "SQL RAND(): returns a random float in [0,1)",
		0, 0,
		[]DeclarationParameter{}, "number",
		func(a ...Scmer) Scmer {
			var buf [8]byte
			if _, err := crand.Read(buf[:]); err != nil {
				panic("sql_rand: " + err.Error())
			}
			// 53 random bits map exactly into float64 mantissa range.
			u := binary.LittleEndian.Uint64(buf[:]) >> 11
			return NewFloat(float64(u) / (1 << 53))
		},
		true, false, nil,
		nil /* TODO: Alloc: new [8]byte (buf) */,
	})
}
