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
			ctx.FreeDesc(&d0)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(4))}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
			}
			ctx.FreeDesc(&d1)
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
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d1 := ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(3))}
			} else {
				r1 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpRegImm32(d1.Reg, 3)
				ctx.W.EmitSetcc(r1, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
			}
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d2.Loc == LocImm {
				if d2.Imm.Bool() {
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d2)
			ctx.W.MarkLabel(lbl2)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(4))}
			} else {
				r2 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r2, CcE)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d3.Loc == LocImm {
				if d3.Imm.Bool() {
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d3.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl5)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl5)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl1)
			r3 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r3, 0)
			ctx.ProtectReg(r3)
			d4 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r3}
			ctx.UnprotectReg(r3)
			ctx.W.EmitMakeBool(result, d4)
			if d4.Loc == LocReg { ctx.FreeReg(d4.Reg) }
			result.Type = tagBool
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl4)
			var d5 JITValueDesc
			if d1.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(15))}
			} else {
				r4 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 15)
				ctx.W.EmitSetcc(r4, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
			}
			ctx.FreeDesc(&d1)
			ctx.EmitStoreToStack(d5, 0)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(8))
			ctx.W.EmitAddRSP32(int32(8))
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
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 8)
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			r1 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r1, 0)
			ctx.ProtectReg(r1)
			d0 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r1}
			r2 := ctx.AllocRegExcept(r1)
			ctx.EmitLoadFromStack(r2, 8)
			ctx.ProtectReg(r2)
			d1 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r2}
			ctx.UnprotectReg(r1)
			ctx.UnprotectReg(r2)
			d2 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d3 JITValueDesc
			if d1.Loc == LocImm && d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d2.Imm.Int())}
			} else if d2.Loc == LocImm {
				r3 := ctx.AllocRegExcept(d1.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r3, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
			} else if d1.Loc == LocImm {
				r4 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d2.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r4, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
			} else {
				r5 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d2.Reg)
				ctx.W.EmitSetcc(r5, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
			}
			ctx.FreeDesc(&d2)
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
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl3)
			d4 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d5 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == d4.Imm.Int())}
			} else if d4.Loc == LocImm {
				r6 := ctx.AllocRegExcept(d1.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d4.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r6, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
			} else if d1.Loc == LocImm {
				r7 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d4.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r7, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
			} else {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d4.Reg)
				ctx.W.EmitSetcc(r8, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
			}
			ctx.FreeDesc(&d4)
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
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl2)
			r9 := ctx.AllocReg()
			ctx.W.EmitMovRegReg(r9, d1.Reg)
			ctx.W.EmitShlRegImm8(r9, 4)
			ctx.W.EmitAddInt64(r9, ctx.SliceBase)
			r10 := ctx.AllocReg()
			r11 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r10, r9, 0)
			ctx.W.EmitMovRegMem(r11, r9, 8)
			ctx.FreeReg(r9)
			d6 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r10, Reg2: r11}
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
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl6)
			var d8 JITValueDesc
			if d0.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d0.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d0.Reg)
				d8 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
			}
			ctx.EmitStoreToStack(d1, 16)
			ctx.EmitStoreToStack(d8, 24)
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
			ctx.FreeDesc(&d6)
			var d10 JITValueDesc
			if d0.Loc == LocImm && d9.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d9.Imm.Int())}
			} else if d9.Loc == LocImm && d9.Imm.Int() == 0 {
				r12 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r12, d0.Reg)
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d9.Reg}
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d9.Reg)
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d9.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d9.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else {
				r13 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r13, d0.Reg)
				ctx.W.EmitAddInt64(r13, d9.Reg)
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
			}
			if d10.Loc == LocReg && d0.Loc == LocReg && d10.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d9)
			var d11 JITValueDesc
			if d1.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			}
			if d11.Loc == LocReg && d1.Loc == LocReg && d11.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.EmitStoreToStack(d10, 0)
			ctx.EmitStoreToStack(d11, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl10)
			r14 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r14, 16)
			ctx.ProtectReg(r14)
			d12 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r14}
			r15 := ctx.AllocRegExcept(r14)
			ctx.EmitLoadFromStack(r15, 24)
			ctx.ProtectReg(r15)
			d13 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r15}
			ctx.UnprotectReg(r14)
			ctx.UnprotectReg(r15)
			d14 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d15 JITValueDesc
			if d12.Loc == LocImm && d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < d14.Imm.Int())}
			} else if d14.Loc == LocImm {
				r16 := ctx.AllocRegExcept(d12.Reg)
				if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d12.Reg, int32(d14.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
					ctx.W.EmitCmpInt64(d12.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r16, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
			} else if d12.Loc == LocImm {
				r17 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d14.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r17, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
			} else {
				r18 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitCmpInt64(d12.Reg, d14.Reg)
				ctx.W.EmitSetcc(r18, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
			}
			ctx.FreeDesc(&d14)
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
			ctx.FreeDesc(&d15)
			ctx.W.MarkLabel(lbl12)
			ctx.W.EmitMakeFloat(result, d13)
			if d13.Loc == LocReg { ctx.FreeReg(d13.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl11)
			r19 := ctx.AllocReg()
			ctx.W.EmitMovRegReg(r19, d12.Reg)
			ctx.W.EmitShlRegImm8(r19, 4)
			ctx.W.EmitAddInt64(r19, ctx.SliceBase)
			r20 := ctx.AllocReg()
			r21 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r20, r19, 0)
			ctx.W.EmitMovRegMem(r21, r19, 8)
			ctx.FreeReg(r19)
			d16 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r20, Reg2: r21}
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
			ctx.FreeDesc(&d17)
			ctx.W.MarkLabel(lbl15)
			var d18 JITValueDesc
			if d16.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d16.Imm.Float())}
			} else {
				ctx.FreeReg(d16.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d16.Reg2}
			}
			ctx.FreeDesc(&d16)
			var d19 JITValueDesc
			if d13.Loc == LocImm && d18.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + d18.Imm.Int())}
			} else if d18.Loc == LocImm && d18.Imm.Int() == 0 {
				r22 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r22, d13.Reg)
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r22}
			} else if d13.Loc == LocImm && d13.Imm.Int() == 0 {
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d18.Reg}
			} else if d13.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d18.Reg)
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d18.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(scratch, d13.Reg)
				if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d18.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d18.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else {
				r23 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r23, d13.Reg)
				ctx.W.EmitAddInt64(r23, d18.Reg)
				d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r23}
			}
			if d19.Loc == LocReg && d13.Loc == LocReg && d19.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = LocNone
			}
			ctx.FreeDesc(&d18)
			var d20 JITValueDesc
			if d12.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			}
			if d20.Loc == LocReg && d12.Loc == LocReg && d20.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = LocNone
			}
			ctx.EmitStoreToStack(d20, 16)
			ctx.EmitStoreToStack(d19, 24)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl14)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(32))
			ctx.W.EmitAddRSP32(int32(32))
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
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 0)
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			r1 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r1, 0)
			ctx.ProtectReg(r1)
			d1 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r1}
			ctx.UnprotectReg(r1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d1.Reg, int32(1))
				d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
			}
			if d2.Loc == LocReg && d1.Loc == LocReg && d2.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d1)
			var d3 JITValueDesc
			if d2.Loc == LocImm && d0.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d0.Imm.Int())}
			} else if d0.Loc == LocImm {
				r2 := ctx.AllocRegExcept(d2.Reg)
				if d0.Imm.Int() >= -2147483648 && d0.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d0.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r2, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
			} else if d2.Loc == LocImm {
				r3 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d0.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r3, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
			} else {
				r4 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d0.Reg)
				ctx.W.EmitSetcc(r4, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
			}
			ctx.FreeDesc(&d0)
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
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl3)
			d4 := args[0]
			d5 := ctx.EmitTagEquals(&d4, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d4)
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
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl2)
			r5 := ctx.AllocReg()
			ctx.W.EmitMovRegReg(r5, d2.Reg)
			ctx.W.EmitShlRegImm8(r5, 4)
			ctx.W.EmitAddInt64(r5, ctx.SliceBase)
			r6 := ctx.AllocReg()
			r7 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r6, r5, 0)
			ctx.W.EmitMovRegMem(r7, r5, 8)
			ctx.FreeReg(r5)
			d6 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
			d7 := ctx.EmitTagEquals(&d6, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d6)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
			ctx.EmitStoreToStack(d2, 0)
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
			ctx.EmitStoreToStack(d2, 0)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl6)
			d8 := args[0]
			var d9 JITValueDesc
			if d8.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d8.Imm.Float())}
			} else {
				ctx.FreeReg(d8.Reg)
				d9 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d8.Reg2}
			}
			ctx.FreeDesc(&d8)
			ctx.EmitStoreToStack(d9, 40)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 48)
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
			ctx.FreeDesc(&d10)
			ctx.EmitStoreToStack(d11, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 16)
			lbl11 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl8)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			r8 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r8, 40)
			ctx.ProtectReg(r8)
			d12 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r8}
			r9 := ctx.AllocRegExcept(r8)
			ctx.EmitLoadFromStack(r9, 48)
			ctx.ProtectReg(r9)
			d13 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r9}
			ctx.UnprotectReg(r8)
			ctx.UnprotectReg(r9)
			d14 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d15 JITValueDesc
			if d13.Loc == LocImm && d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d14.Imm.Int())}
			} else if d14.Loc == LocImm {
				r10 := ctx.AllocRegExcept(d13.Reg)
				if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d13.Reg, int32(d14.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
					ctx.W.EmitCmpInt64(d13.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r10, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
			} else if d13.Loc == LocImm {
				r11 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d14.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r11, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
			} else {
				r12 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitCmpInt64(d13.Reg, d14.Reg)
				ctx.W.EmitSetcc(r12, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
			}
			ctx.FreeDesc(&d14)
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
			ctx.FreeDesc(&d15)
			ctx.W.MarkLabel(lbl11)
			r13 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r13, 8)
			ctx.ProtectReg(r13)
			d16 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r13}
			r14 := ctx.AllocRegExcept(r13)
			ctx.EmitLoadFromStack(r14, 16)
			ctx.ProtectReg(r14)
			d17 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r14}
			ctx.UnprotectReg(r13)
			ctx.UnprotectReg(r14)
			d18 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d19 JITValueDesc
			if d17.Loc == LocImm && d18.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() < d18.Imm.Int())}
			} else if d18.Loc == LocImm {
				r15 := ctx.AllocRegExcept(d17.Reg)
				if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d17.Reg, int32(d18.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
					ctx.W.EmitCmpInt64(d17.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r15, CcL)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
			} else if d17.Loc == LocImm {
				r16 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d18.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r16, CcL)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
			} else {
				r17 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitCmpInt64(d17.Reg, d18.Reg)
				ctx.W.EmitSetcc(r17, CcL)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
			}
			ctx.FreeDesc(&d18)
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
			ctx.FreeDesc(&d19)
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitMakeFloat(result, d12)
			if d12.Loc == LocReg { ctx.FreeReg(d12.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			r18 := ctx.AllocReg()
			ctx.W.EmitMovRegReg(r18, d13.Reg)
			ctx.W.EmitShlRegImm8(r18, 4)
			ctx.W.EmitAddInt64(r18, ctx.SliceBase)
			r19 := ctx.AllocReg()
			r20 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r19, r18, 0)
			ctx.W.EmitMovRegMem(r20, r18, 8)
			ctx.FreeReg(r18)
			d20 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r19, Reg2: r20}
			var d21 JITValueDesc
			if d20.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d20.Imm.Float())}
			} else {
				ctx.FreeReg(d20.Reg)
				d21 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d20.Reg2}
			}
			ctx.FreeDesc(&d20)
			var d22 JITValueDesc
			if d12.Loc == LocImm && d21.Loc == LocImm {
				d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() - d21.Imm.Int())}
			} else if d21.Loc == LocImm && d21.Imm.Int() == 0 {
				r21 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r21, d12.Reg)
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r21}
			} else if d12.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d21.Reg)
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d21.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d21.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else {
				r22 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r22, d12.Reg)
				ctx.W.EmitSubInt64(r22, d21.Reg)
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r22}
			}
			if d22.Loc == LocReg && d12.Loc == LocReg && d22.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = LocNone
			}
			ctx.FreeDesc(&d21)
			var d23 JITValueDesc
			if d13.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(scratch, d13.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			}
			if d23.Loc == LocReg && d13.Loc == LocReg && d23.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = LocNone
			}
			ctx.EmitStoreToStack(d22, 40)
			ctx.EmitStoreToStack(d23, 48)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl16)
			d24 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d25 JITValueDesc
			if d17.Loc == LocImm && d24.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() == d24.Imm.Int())}
			} else if d24.Loc == LocImm {
				r23 := ctx.AllocRegExcept(d17.Reg)
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d17.Reg, int32(d24.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
					ctx.W.EmitCmpInt64(d17.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r23, CcE)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
			} else if d17.Loc == LocImm {
				r24 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d24.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r24, CcE)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
			} else {
				r25 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitCmpInt64(d17.Reg, d24.Reg)
				ctx.W.EmitSetcc(r25, CcE)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
			}
			ctx.FreeDesc(&d24)
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
			ctx.FreeDesc(&d25)
			ctx.W.MarkLabel(lbl15)
			r26 := ctx.AllocReg()
			ctx.W.EmitMovRegReg(r26, d17.Reg)
			ctx.W.EmitShlRegImm8(r26, 4)
			ctx.W.EmitAddInt64(r26, ctx.SliceBase)
			r27 := ctx.AllocReg()
			r28 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r27, r26, 0)
			ctx.W.EmitMovRegMem(r28, r26, 8)
			ctx.FreeReg(r26)
			d26 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r27, Reg2: r28}
			d27 := ctx.EmitTagEquals(&d26, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d26)
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
			ctx.FreeDesc(&d27)
			ctx.W.MarkLabel(lbl19)
			var d28 JITValueDesc
			if d16.Loc == LocImm {
				d28 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d16.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d16.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d16.Reg}
			}
			ctx.EmitStoreToStack(d17, 24)
			ctx.EmitStoreToStack(d28, 32)
			lbl23 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl18)
			ctx.W.EmitMakeInt(result, d16)
			if d16.Loc == LocReg { ctx.FreeReg(d16.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl21)
			r29 := ctx.AllocReg()
			ctx.W.EmitMovRegReg(r29, d17.Reg)
			ctx.W.EmitShlRegImm8(r29, 4)
			ctx.W.EmitAddInt64(r29, ctx.SliceBase)
			r30 := ctx.AllocReg()
			r31 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r30, r29, 0)
			ctx.W.EmitMovRegMem(r31, r29, 8)
			ctx.FreeReg(r29)
			d29 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r30, Reg2: r31}
			var d30 JITValueDesc
			if d29.Loc == LocImm {
				d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int())}
			} else {
				ctx.FreeReg(d29.Reg)
				d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg2}
			}
			ctx.FreeDesc(&d29)
			var d31 JITValueDesc
			if d16.Loc == LocImm && d30.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d16.Imm.Int() - d30.Imm.Int())}
			} else if d30.Loc == LocImm && d30.Imm.Int() == 0 {
				r32 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r32, d16.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r32}
			} else if d16.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d30.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d30.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				if d30.Imm.Int() >= -2147483648 && d30.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d30.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d30.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else {
				r33 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r33, d16.Reg)
				ctx.W.EmitSubInt64(r33, d30.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r33}
			}
			if d31.Loc == LocReg && d16.Loc == LocReg && d31.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = LocNone
			}
			ctx.FreeDesc(&d30)
			var d32 JITValueDesc
			if d17.Loc == LocImm {
				d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d17.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(scratch, d17.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			}
			if d32.Loc == LocReg && d17.Loc == LocReg && d32.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = LocNone
			}
			ctx.EmitStoreToStack(d31, 8)
			ctx.EmitStoreToStack(d32, 16)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl23)
			r34 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r34, 24)
			ctx.ProtectReg(r34)
			d33 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r34}
			r35 := ctx.AllocRegExcept(r34)
			ctx.EmitLoadFromStack(r35, 32)
			ctx.ProtectReg(r35)
			d34 := JITValueDesc{Loc: LocReg, Type: JITTypeUnknown, Reg: r35}
			ctx.UnprotectReg(r34)
			ctx.UnprotectReg(r35)
			d35 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			var d36 JITValueDesc
			if d33.Loc == LocImm && d35.Loc == LocImm {
				d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d33.Imm.Int() < d35.Imm.Int())}
			} else if d35.Loc == LocImm {
				r36 := ctx.AllocRegExcept(d33.Reg)
				if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d33.Reg, int32(d35.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
					ctx.W.EmitCmpInt64(d33.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r36, CcL)
				d36 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r36}
			} else if d33.Loc == LocImm {
				r37 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d35.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r37, CcL)
				d36 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r37}
			} else {
				r38 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitCmpInt64(d33.Reg, d35.Reg)
				ctx.W.EmitSetcc(r38, CcL)
				d36 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r38}
			}
			ctx.FreeDesc(&d35)
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
			ctx.FreeDesc(&d36)
			ctx.W.MarkLabel(lbl25)
			ctx.W.EmitMakeFloat(result, d34)
			if d34.Loc == LocReg { ctx.FreeReg(d34.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl24)
			r39 := ctx.AllocReg()
			ctx.W.EmitMovRegReg(r39, d33.Reg)
			ctx.W.EmitShlRegImm8(r39, 4)
			ctx.W.EmitAddInt64(r39, ctx.SliceBase)
			r40 := ctx.AllocReg()
			r41 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r40, r39, 0)
			ctx.W.EmitMovRegMem(r41, r39, 8)
			ctx.FreeReg(r39)
			d37 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r40, Reg2: r41}
			var d38 JITValueDesc
			if d37.Loc == LocImm {
				d38 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d37.Imm.Float())}
			} else {
				ctx.FreeReg(d37.Reg)
				d38 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d37.Reg2}
			}
			ctx.FreeDesc(&d37)
			var d39 JITValueDesc
			if d34.Loc == LocImm && d38.Loc == LocImm {
				d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d34.Imm.Int() - d38.Imm.Int())}
			} else if d38.Loc == LocImm && d38.Imm.Int() == 0 {
				r42 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(r42, d34.Reg)
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r42}
			} else if d34.Loc == LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d38.Reg)
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else if d38.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(scratch, d34.Reg)
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d38.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d38.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			} else {
				r43 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(r43, d34.Reg)
				ctx.W.EmitSubInt64(r43, d38.Reg)
				d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r43}
			}
			if d39.Loc == LocReg && d34.Loc == LocReg && d39.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = LocNone
			}
			ctx.FreeDesc(&d38)
			var d40 JITValueDesc
			if d33.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d33.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(scratch, d33.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
			}
			if d40.Loc == LocReg && d33.Loc == LocReg && d40.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = LocNone
			}
			ctx.EmitStoreToStack(d40, 24)
			ctx.EmitStoreToStack(d39, 32)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(56))
			ctx.W.EmitAddRSP32(int32(56))
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
		nil /* TODO: If condition is not a desc: true:untyped bool */,
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
		nil /* TODO: runtime error: invalid memory address or nil pointer dereference */,
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
		nil /* TODO: len on non-parameter: len(t0) */,
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
		nil /* TODO: len on non-parameter: len(t0) */,
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
		nil /* TODO: len on non-parameter: len(t0) */,
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
		nil /* TODO: len on non-parameter: len(t0) */,
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
		nil /* TODO: FieldAddr on non-receiver: &t0.ptr [#0] */,
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
		nil /* TODO: FieldAddr on non-receiver: &t0.aux [#1] */,
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
		nil /* TODO: len on non-parameter: len(s) */,
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
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */,
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
		nil /* TODO: FieldAddr on non-receiver: &t0.aux [#1] */,
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
		nil /* TODO: FieldAddr on non-receiver: &t0.aux [#1] */,
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
			ctx.FreeDesc(&d0)
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
		nil /* TODO: len on non-parameter: len(t0) */,
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
		nil /* TODO: len on non-parameter: len(t0) */,
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
		nil /* TODO: If condition is not a desc: true:untyped bool */,
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
		nil /* TODO: If condition is not a desc: true:untyped bool */,
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
		nil /* TODO: unsupported Convert *float64 → unsafe.Pointer */,
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
		nil /* TODO: unsupported Convert float64 → int */,
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
		nil /* TODO: unsupported Convert *float64 → unsafe.Pointer */,
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
		nil /* TODO: runtime error: invalid memory address or nil pointer dereference */,
	})
}
