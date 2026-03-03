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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			d1 := ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(4))}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d1 := ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(3))}
			} else {
				r1 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpRegImm32(d1.Reg, 3)
				ctx.W.EmitSetcc(r1, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d2)
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
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(4))}
			} else {
				r2 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r2, CcE)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d3)
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
			d4 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			ctx.W.EmitMakeBool(result, d4)
			if d4.Loc == LocReg { ctx.FreeReg(d4.Reg) }
			result.Type = tagBool
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl4)
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d5 JITValueDesc
			if d1.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(15))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 15)
				ctx.W.EmitSetcc(r3, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d5)
			}
			ctx.FreeDesc(&d1)
			d6 := d5
			if d6.Loc == LocNone { panic("jit: phi source has no location") }
			if d6.Loc == LocStack || d6.Loc == LocStackPair { ctx.EnsureDesc(&d6) }
			ctx.EmitStoreToStack(d6, 0)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			var d3 JITValueDesc
			if d1.Loc == LocImm && d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d2.Imm.Int())}
			} else if d2.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d1.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d3)
			} else if d1.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d2.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d3)
			} else {
				r3 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d2.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d3)
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
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			if d4.Loc == LocStack || d4.Loc == LocStackPair { ctx.EnsureDesc(&d4) }
			var d5 JITValueDesc
			if d1.Loc == LocImm && d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == d4.Imm.Int())}
			} else if d4.Loc == LocImm {
				r4 := ctx.AllocRegExcept(d1.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r4, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d5)
			} else if d1.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d4.Reg)
				ctx.W.EmitSetcc(r5, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d5)
			} else {
				r6 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d4.Reg)
				ctx.W.EmitSetcc(r6, CcE)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d5)
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
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d6 JITValueDesc
			if d1.Loc == LocImm {
				idx := int(d1.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d6 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d1.Reg] {
					ctx.ProtectReg(d1.Reg)
					seen[d1.Reg] = true
					protected = append(protected, d1.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					}
				}
				r7 := ctx.AllocReg()
				r8 := ctx.AllocRegExcept(r7)
				lbl8 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r7, ai.Reg)
						ctx.W.EmitMovRegReg(r8, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r7, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r8, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl8)
					ctx.W.MarkLabel(nextLbl)
				}
				panic("jitgen: dynamic args index out of range")
				ctx.W.MarkLabel(lbl8)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d6 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r7, Reg2: r8}
				ctx.BindReg(r7, &d6)
				ctx.BindReg(r8, &d6)
			}
			d7 := ctx.EmitTagEquals(&d6, tagInt, JITValueDesc{Loc: LocAny})
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl6)
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			var d8 JITValueDesc
			if d0.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d0.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d0.Reg)
				d8 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d8)
			}
			lbl11 := ctx.W.ReserveLabel()
			d9 := d1
			if d9.Loc == LocNone { panic("jit: phi source has no location") }
			if d9.Loc == LocStack || d9.Loc == LocStackPair { ctx.EnsureDesc(&d9) }
			ctx.EmitStoreToStack(d9, 16)
			d10 := d8
			if d10.Loc == LocNone { panic("jit: phi source has no location") }
			if d10.Loc == LocStack || d10.Loc == LocStackPair { ctx.EnsureDesc(&d10) }
			ctx.EmitStoreToStack(d10, 24)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl11)
			d11 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d12 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d13 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d11.Loc == LocStack || d11.Loc == LocStackPair { ctx.EnsureDesc(&d11) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d11.Loc == LocStack || d11.Loc == LocStackPair { ctx.EnsureDesc(&d11) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d11.Loc == LocStack || d11.Loc == LocStackPair { ctx.EnsureDesc(&d11) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			var d14 JITValueDesc
			if d11.Loc == LocImm && d13.Loc == LocImm {
				d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Int() < d13.Imm.Int())}
			} else if d13.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d11.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d11.Reg, int32(d13.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
					ctx.W.EmitCmpInt64(d11.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcL)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d14)
			} else if d11.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d13.Reg)
				ctx.W.EmitSetcc(r10, CcL)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d14)
			} else {
				r11 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitCmpInt64(d11.Reg, d13.Reg)
				ctx.W.EmitSetcc(r11, CcL)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d14)
			}
			ctx.FreeDesc(&d13)
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d14.Loc == LocImm {
				if d14.Imm.Bool() {
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d14.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d14)
			ctx.W.MarkLabel(lbl5)
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			ctx.W.EmitMakeInt(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl9)
			var d15 JITValueDesc
			if d6.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int())}
			} else if d6.Type == tagInt && d6.Loc == LocRegPair {
				ctx.FreeReg(d6.Reg)
				d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2}
				ctx.BindReg(d6.Reg2, &d15)
				ctx.BindReg(d6.Reg2, &d15)
			} else if d6.Type == tagInt && d6.Loc == LocReg {
				d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg}
				ctx.BindReg(d6.Reg, &d15)
				ctx.BindReg(d6.Reg, &d15)
			} else {
				d15 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d6}, 1)
				d15.Type = tagInt
				ctx.BindReg(d15.Reg, &d15)
			}
			ctx.FreeDesc(&d6)
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			var d16 JITValueDesc
			if d0.Loc == LocImm && d15.Loc == LocImm {
				d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d15.Imm.Int())}
			} else if d15.Loc == LocImm && d15.Imm.Int() == 0 {
				r12 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r12, d0.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d16)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg}
				ctx.BindReg(d15.Reg, &d16)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d15.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d16)
			} else if d15.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d15.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d16)
			} else {
				r13 := ctx.AllocRegExcept(d0.Reg, d15.Reg)
				ctx.W.EmitMovRegReg(r13, d0.Reg)
				ctx.W.EmitAddInt64(r13, d15.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
				ctx.BindReg(r13, &d16)
			}
			if d16.Loc == LocReg && d0.Loc == LocReg && d16.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d15)
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d17 JITValueDesc
			if d1.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			}
			if d17.Loc == LocReg && d1.Loc == LocReg && d17.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			d18 := d16
			if d18.Loc == LocNone { panic("jit: phi source has no location") }
			if d18.Loc == LocStack || d18.Loc == LocStackPair { ctx.EnsureDesc(&d18) }
			ctx.EmitStoreToStack(d18, 0)
			d19 := d17
			if d19.Loc == LocNone { panic("jit: phi source has no location") }
			if d19.Loc == LocStack || d19.Loc == LocStackPair { ctx.EnsureDesc(&d19) }
			ctx.EmitStoreToStack(d19, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl13)
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			ctx.W.EmitMakeFloat(result, d12)
			if d12.Loc == LocReg { ctx.FreeReg(d12.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			if d11.Loc == LocStack || d11.Loc == LocStackPair { ctx.EnsureDesc(&d11) }
			var d20 JITValueDesc
			if d11.Loc == LocImm {
				idx := int(d11.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d20 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d11.Reg] {
					ctx.ProtectReg(d11.Reg)
					seen[d11.Reg] = true
					protected = append(protected, d11.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					}
				}
				r14 := ctx.AllocReg()
				r15 := ctx.AllocRegExcept(r14)
				lbl15 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d11.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r14, ai.Reg)
						ctx.W.EmitMovRegReg(r15, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r14, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r15, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl15)
					ctx.W.MarkLabel(nextLbl)
				}
				panic("jitgen: dynamic args index out of range")
				ctx.W.MarkLabel(lbl15)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d20 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r14, Reg2: r15}
				ctx.BindReg(r14, &d20)
				ctx.BindReg(r15, &d20)
			}
			d21 := ctx.EmitTagEquals(&d20, tagNil, JITValueDesc{Loc: LocAny})
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d21.Loc == LocImm {
				if d21.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d21.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d21)
			ctx.W.MarkLabel(lbl17)
			var d22 JITValueDesc
			if d20.Loc == LocImm {
				d22 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d20.Imm.Float())}
			} else if d20.Type == tagFloat && d20.Loc == LocRegPair {
				ctx.FreeReg(d20.Reg)
				d22 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d20.Reg2}
				ctx.BindReg(d20.Reg2, &d22)
				ctx.BindReg(d20.Reg2, &d22)
			} else if d20.Type == tagFloat && d20.Loc == LocReg {
				d22 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d20.Reg}
				ctx.BindReg(d20.Reg, &d22)
				ctx.BindReg(d20.Reg, &d22)
			} else {
				d22 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Float), []JITValueDesc{d20}, 1)
				d22.Type = tagFloat
				ctx.BindReg(d22.Reg, &d22)
			}
			ctx.FreeDesc(&d20)
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 JITValueDesc
			if d12.Loc == LocImm && d22.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + d22.Imm.Int())}
			} else if d22.Loc == LocImm && d22.Imm.Int() == 0 {
				r16 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r16, d12.Reg)
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
				ctx.BindReg(r16, &d23)
			} else if d12.Loc == LocImm && d12.Imm.Int() == 0 {
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d23)
			} else if d12.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d22.Reg)
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d22.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d22.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r17 := ctx.AllocRegExcept(d12.Reg, d22.Reg)
				ctx.W.EmitMovRegReg(r17, d12.Reg)
				ctx.W.EmitAddInt64(r17, d22.Reg)
				d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
				ctx.BindReg(r17, &d23)
			}
			if d23.Loc == LocReg && d12.Loc == LocReg && d23.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = LocNone
			}
			ctx.FreeDesc(&d22)
			if d11.Loc == LocStack || d11.Loc == LocStackPair { ctx.EnsureDesc(&d11) }
			var d24 JITValueDesc
			if d11.Loc == LocImm {
				d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d11.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(scratch, d11.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			}
			if d24.Loc == LocReg && d11.Loc == LocReg && d24.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = LocNone
			}
			d25 := d24
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			if d25.Loc == LocStack || d25.Loc == LocStackPair { ctx.EnsureDesc(&d25) }
			ctx.EmitStoreToStack(d25, 16)
			d26 := d23
			if d26.Loc == LocNone { panic("jit: phi source has no location") }
			if d26.Loc == LocStack || d26.Loc == LocStackPair { ctx.EnsureDesc(&d26) }
			ctx.EmitStoreToStack(d26, 24)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl16)
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl1 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 0)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d1.Reg, int32(1))
				d2 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d2)
			}
			if d2.Loc == LocReg && d1.Loc == LocReg && d2.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d1)
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			var d3 JITValueDesc
			if d2.Loc == LocImm && d0.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d0.Imm.Int())}
			} else if d0.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d2.Reg)
				if d0.Imm.Int() >= -2147483648 && d0.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d0.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d3)
			} else if d2.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d0.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d3)
			} else {
				r3 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d0.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d3)
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
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			var d6 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d6 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d2.Reg] {
					ctx.ProtectReg(d2.Reg)
					seen[d2.Reg] = true
					protected = append(protected, d2.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					}
				}
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl8 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r4, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r5, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl8)
					ctx.W.MarkLabel(nextLbl)
				}
				panic("jitgen: dynamic args index out of range")
				ctx.W.MarkLabel(lbl8)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d6 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d6)
				ctx.BindReg(r5, &d6)
			}
			d7 := ctx.EmitTagEquals(&d6, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d6)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d8 := d2
			if d8.Loc == LocNone { panic("jit: phi source has no location") }
			if d8.Loc == LocStack || d8.Loc == LocStackPair { ctx.EnsureDesc(&d8) }
			ctx.EmitStoreToStack(d8, 0)
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
			d9 := d2
			if d9.Loc == LocNone { panic("jit: phi source has no location") }
			if d9.Loc == LocStack || d9.Loc == LocStackPair { ctx.EnsureDesc(&d9) }
			ctx.EmitStoreToStack(d9, 0)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl6)
			d10 := args[0]
			var d11 JITValueDesc
			if d10.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float())}
			} else if d10.Type == tagFloat && d10.Loc == LocRegPair {
				ctx.FreeReg(d10.Reg)
				d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg2}
				ctx.BindReg(d10.Reg2, &d11)
				ctx.BindReg(d10.Reg2, &d11)
			} else if d10.Type == tagFloat && d10.Loc == LocReg {
				d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg}
				ctx.BindReg(d10.Reg, &d11)
				ctx.BindReg(d10.Reg, &d11)
			} else {
				d11 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Float), []JITValueDesc{d10}, 1)
				d11.Type = tagFloat
				ctx.BindReg(d11.Reg, &d11)
			}
			ctx.FreeDesc(&d10)
			lbl11 := ctx.W.ReserveLabel()
			d12 := d11
			if d12.Loc == LocNone { panic("jit: phi source has no location") }
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			ctx.EmitStoreToStack(d12, 40)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl11)
			d13 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d14 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d15 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			var d16 JITValueDesc
			if d14.Loc == LocImm && d15.Loc == LocImm {
				d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.Int() < d15.Imm.Int())}
			} else if d15.Loc == LocImm {
				r6 := ctx.AllocRegExcept(d14.Reg)
				if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d14.Reg, int32(d15.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
					ctx.W.EmitCmpInt64(d14.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r6, CcL)
				d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d16)
			} else if d14.Loc == LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d15.Reg)
				ctx.W.EmitSetcc(r7, CcL)
				d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d16)
			} else {
				r8 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitCmpInt64(d14.Reg, d15.Reg)
				ctx.W.EmitSetcc(r8, CcL)
				d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d16)
			}
			ctx.FreeDesc(&d15)
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d16.Loc == LocImm {
				if d16.Imm.Bool() {
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl5)
			d17 := args[0]
			var d18 JITValueDesc
			if d17.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d17.Imm.Int())}
			} else if d17.Type == tagInt && d17.Loc == LocRegPair {
				ctx.FreeReg(d17.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2}
				ctx.BindReg(d17.Reg2, &d18)
				ctx.BindReg(d17.Reg2, &d18)
			} else if d17.Type == tagInt && d17.Loc == LocReg {
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg}
				ctx.BindReg(d17.Reg, &d18)
				ctx.BindReg(d17.Reg, &d18)
			} else {
				d18 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d17}, 1)
				d18.Type = tagInt
				ctx.BindReg(d18.Reg, &d18)
			}
			ctx.FreeDesc(&d17)
			lbl15 := ctx.W.ReserveLabel()
			d19 := d18
			if d19.Loc == LocNone { panic("jit: phi source has no location") }
			if d19.Loc == LocStack || d19.Loc == LocStackPair { ctx.EnsureDesc(&d19) }
			ctx.EmitStoreToStack(d19, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 16)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl15)
			d20 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d21 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d22 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 JITValueDesc
			if d21.Loc == LocImm && d22.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() < d22.Imm.Int())}
			} else if d22.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d21.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d21.Reg, int32(d22.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d22.Imm.Int()))
					ctx.W.EmitCmpInt64(d21.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcL)
				d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d23)
			} else if d21.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d22.Reg)
				ctx.W.EmitSetcc(r10, CcL)
				d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d23)
			} else {
				r11 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitCmpInt64(d21.Reg, d22.Reg)
				ctx.W.EmitSetcc(r11, CcL)
				d23 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d23)
			}
			ctx.FreeDesc(&d22)
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d23.Loc == LocImm {
				if d23.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d23.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d23)
			ctx.W.MarkLabel(lbl9)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl13)
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			ctx.W.EmitMakeFloat(result, d13)
			if d13.Loc == LocReg { ctx.FreeReg(d13.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			var d24 JITValueDesc
			if d14.Loc == LocImm {
				idx := int(d14.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d24 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d14.Reg] {
					ctx.ProtectReg(d14.Reg)
					seen[d14.Reg] = true
					protected = append(protected, d14.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					}
				}
				r12 := ctx.AllocReg()
				r13 := ctx.AllocRegExcept(r12)
				lbl19 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d14.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r12, ai.Reg)
						ctx.W.EmitMovRegReg(r13, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r12, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r13, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl19)
					ctx.W.MarkLabel(nextLbl)
				}
				panic("jitgen: dynamic args index out of range")
				ctx.W.MarkLabel(lbl19)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d24 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r12, Reg2: r13}
				ctx.BindReg(r12, &d24)
				ctx.BindReg(r13, &d24)
			}
			var d25 JITValueDesc
			if d24.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d24.Imm.Float())}
			} else if d24.Type == tagFloat && d24.Loc == LocRegPair {
				ctx.FreeReg(d24.Reg)
				d25 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d24.Reg2}
				ctx.BindReg(d24.Reg2, &d25)
				ctx.BindReg(d24.Reg2, &d25)
			} else if d24.Type == tagFloat && d24.Loc == LocReg {
				d25 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d24.Reg}
				ctx.BindReg(d24.Reg, &d25)
				ctx.BindReg(d24.Reg, &d25)
			} else {
				d25 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Float), []JITValueDesc{d24}, 1)
				d25.Type = tagFloat
				ctx.BindReg(d25.Reg, &d25)
			}
			ctx.FreeDesc(&d24)
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d25.Loc == LocStack || d25.Loc == LocStackPair { ctx.EnsureDesc(&d25) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d25.Loc == LocStack || d25.Loc == LocStackPair { ctx.EnsureDesc(&d25) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d25.Loc == LocStack || d25.Loc == LocStackPair { ctx.EnsureDesc(&d25) }
			var d26 JITValueDesc
			if d13.Loc == LocImm && d25.Loc == LocImm {
				d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() - d25.Imm.Int())}
			} else if d25.Loc == LocImm && d25.Imm.Int() == 0 {
				r14 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r14, d13.Reg)
				d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
				ctx.BindReg(r14, &d26)
			} else if d13.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d25.Reg)
				d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else if d25.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(scratch, d13.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d25.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else {
				r15 := ctx.AllocRegExcept(d13.Reg, d25.Reg)
				ctx.W.EmitMovRegReg(r15, d13.Reg)
				ctx.W.EmitSubInt64(r15, d25.Reg)
				d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
				ctx.BindReg(r15, &d26)
			}
			if d26.Loc == LocReg && d13.Loc == LocReg && d26.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = LocNone
			}
			ctx.FreeDesc(&d25)
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			var d27 JITValueDesc
			if d14.Loc == LocImm {
				d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d14.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(scratch, d14.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			}
			if d27.Loc == LocReg && d14.Loc == LocReg && d27.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = LocNone
			}
			d28 := d26
			if d28.Loc == LocNone { panic("jit: phi source has no location") }
			if d28.Loc == LocStack || d28.Loc == LocStackPair { ctx.EnsureDesc(&d28) }
			ctx.EmitStoreToStack(d28, 40)
			d29 := d27
			if d29.Loc == LocNone { panic("jit: phi source has no location") }
			if d29.Loc == LocStack || d29.Loc == LocStackPair { ctx.EnsureDesc(&d29) }
			ctx.EmitStoreToStack(d29, 48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl17)
			d30 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			if d30.Loc == LocStack || d30.Loc == LocStackPair { ctx.EnsureDesc(&d30) }
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			if d30.Loc == LocStack || d30.Loc == LocStackPair { ctx.EnsureDesc(&d30) }
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			if d30.Loc == LocStack || d30.Loc == LocStackPair { ctx.EnsureDesc(&d30) }
			var d31 JITValueDesc
			if d21.Loc == LocImm && d30.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() == d30.Imm.Int())}
			} else if d30.Loc == LocImm {
				r16 := ctx.AllocRegExcept(d21.Reg)
				if d30.Imm.Int() >= -2147483648 && d30.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d21.Reg, int32(d30.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d30.Imm.Int()))
					ctx.W.EmitCmpInt64(d21.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r16, CcE)
				d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d31)
			} else if d21.Loc == LocImm {
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d30.Reg)
				ctx.W.EmitSetcc(r17, CcE)
				d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d31)
			} else {
				r18 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitCmpInt64(d21.Reg, d30.Reg)
				ctx.W.EmitSetcc(r18, CcE)
				d31 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
				ctx.BindReg(r18, &d31)
			}
			ctx.FreeDesc(&d30)
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d31.Loc == LocImm {
				if d31.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d31.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d31)
			ctx.W.MarkLabel(lbl16)
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			var d32 JITValueDesc
			if d21.Loc == LocImm {
				idx := int(d21.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d32 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d21.Reg] {
					ctx.ProtectReg(d21.Reg)
					seen[d21.Reg] = true
					protected = append(protected, d21.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					}
				}
				r19 := ctx.AllocReg()
				r20 := ctx.AllocRegExcept(r19)
				lbl23 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d21.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r19, ai.Reg)
						ctx.W.EmitMovRegReg(r20, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r19, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r20, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl23)
					ctx.W.MarkLabel(nextLbl)
				}
				panic("jitgen: dynamic args index out of range")
				ctx.W.MarkLabel(lbl23)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d32 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r19, Reg2: r20}
				ctx.BindReg(r19, &d32)
				ctx.BindReg(r20, &d32)
			}
			d33 := ctx.EmitTagEquals(&d32, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d32)
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d33.Loc == LocImm {
				if d33.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d33.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl25)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d33)
			ctx.W.MarkLabel(lbl21)
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			var d34 JITValueDesc
			if d20.Loc == LocImm {
				d34 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d20.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d20.Reg)
				d34 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d20.Reg}
				ctx.BindReg(d20.Reg, &d34)
			}
			lbl26 := ctx.W.ReserveLabel()
			d35 := d21
			if d35.Loc == LocNone { panic("jit: phi source has no location") }
			if d35.Loc == LocStack || d35.Loc == LocStackPair { ctx.EnsureDesc(&d35) }
			ctx.EmitStoreToStack(d35, 24)
			d36 := d34
			if d36.Loc == LocNone { panic("jit: phi source has no location") }
			if d36.Loc == LocStack || d36.Loc == LocStackPair { ctx.EnsureDesc(&d36) }
			ctx.EmitStoreToStack(d36, 32)
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl26)
			d37 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d38 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d39 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d37.Loc == LocStack || d37.Loc == LocStackPair { ctx.EnsureDesc(&d37) }
			if d39.Loc == LocStack || d39.Loc == LocStackPair { ctx.EnsureDesc(&d39) }
			if d37.Loc == LocStack || d37.Loc == LocStackPair { ctx.EnsureDesc(&d37) }
			if d39.Loc == LocStack || d39.Loc == LocStackPair { ctx.EnsureDesc(&d39) }
			if d37.Loc == LocStack || d37.Loc == LocStackPair { ctx.EnsureDesc(&d37) }
			if d39.Loc == LocStack || d39.Loc == LocStackPair { ctx.EnsureDesc(&d39) }
			var d40 JITValueDesc
			if d37.Loc == LocImm && d39.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.Int() < d39.Imm.Int())}
			} else if d39.Loc == LocImm {
				r21 := ctx.AllocRegExcept(d37.Reg)
				if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d37.Reg, int32(d39.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d39.Imm.Int()))
					ctx.W.EmitCmpInt64(d37.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r21, CcL)
				d40 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
				ctx.BindReg(r21, &d40)
			} else if d37.Loc == LocImm {
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d37.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d39.Reg)
				ctx.W.EmitSetcc(r22, CcL)
				d40 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d40)
			} else {
				r23 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitCmpInt64(d37.Reg, d39.Reg)
				ctx.W.EmitSetcc(r23, CcL)
				d40 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d40)
			}
			ctx.FreeDesc(&d39)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d40.Loc == LocImm {
				if d40.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d40.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl29)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d40)
			ctx.W.MarkLabel(lbl20)
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			ctx.W.EmitMakeInt(result, d20)
			if d20.Loc == LocReg { ctx.FreeReg(d20.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl24)
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			var d41 JITValueDesc
			if d21.Loc == LocImm {
				idx := int(d21.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d41 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d21.Reg] {
					ctx.ProtectReg(d21.Reg)
					seen[d21.Reg] = true
					protected = append(protected, d21.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					}
				}
				r24 := ctx.AllocReg()
				r25 := ctx.AllocRegExcept(r24)
				lbl30 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d21.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r24, ai.Reg)
						ctx.W.EmitMovRegReg(r25, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r24, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r25, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl30)
					ctx.W.MarkLabel(nextLbl)
				}
				panic("jitgen: dynamic args index out of range")
				ctx.W.MarkLabel(lbl30)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d41 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r24, Reg2: r25}
				ctx.BindReg(r24, &d41)
				ctx.BindReg(r25, &d41)
			}
			var d42 JITValueDesc
			if d41.Loc == LocImm {
				d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d41.Imm.Int())}
			} else if d41.Type == tagInt && d41.Loc == LocRegPair {
				ctx.FreeReg(d41.Reg)
				d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2}
				ctx.BindReg(d41.Reg2, &d42)
				ctx.BindReg(d41.Reg2, &d42)
			} else if d41.Type == tagInt && d41.Loc == LocReg {
				d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg}
				ctx.BindReg(d41.Reg, &d42)
				ctx.BindReg(d41.Reg, &d42)
			} else {
				d42 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d41}, 1)
				d42.Type = tagInt
				ctx.BindReg(d42.Reg, &d42)
			}
			ctx.FreeDesc(&d41)
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			if d42.Loc == LocStack || d42.Loc == LocStackPair { ctx.EnsureDesc(&d42) }
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			if d42.Loc == LocStack || d42.Loc == LocStackPair { ctx.EnsureDesc(&d42) }
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			if d42.Loc == LocStack || d42.Loc == LocStackPair { ctx.EnsureDesc(&d42) }
			var d43 JITValueDesc
			if d20.Loc == LocImm && d42.Loc == LocImm {
				d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d20.Imm.Int() - d42.Imm.Int())}
			} else if d42.Loc == LocImm && d42.Imm.Int() == 0 {
				r26 := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(r26, d20.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
				ctx.BindReg(r26, &d43)
			} else if d20.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d42.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(scratch, d20.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r27 := ctx.AllocRegExcept(d20.Reg, d42.Reg)
				ctx.W.EmitMovRegReg(r27, d20.Reg)
				ctx.W.EmitSubInt64(r27, d42.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r27}
				ctx.BindReg(r27, &d43)
			}
			if d43.Loc == LocReg && d20.Loc == LocReg && d43.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = LocNone
			}
			ctx.FreeDesc(&d42)
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			var d44 JITValueDesc
			if d21.Loc == LocImm {
				d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d21.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(scratch, d21.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			}
			if d44.Loc == LocReg && d21.Loc == LocReg && d44.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = LocNone
			}
			d45 := d43
			if d45.Loc == LocNone { panic("jit: phi source has no location") }
			if d45.Loc == LocStack || d45.Loc == LocStackPair { ctx.EnsureDesc(&d45) }
			ctx.EmitStoreToStack(d45, 8)
			d46 := d44
			if d46.Loc == LocNone { panic("jit: phi source has no location") }
			if d46.Loc == LocStack || d46.Loc == LocStackPair { ctx.EnsureDesc(&d46) }
			ctx.EmitStoreToStack(d46, 16)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl28)
			if d38.Loc == LocStack || d38.Loc == LocStackPair { ctx.EnsureDesc(&d38) }
			if d38.Loc == LocStack || d38.Loc == LocStackPair { ctx.EnsureDesc(&d38) }
			ctx.W.EmitMakeFloat(result, d38)
			if d38.Loc == LocReg { ctx.FreeReg(d38.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl27)
			if d37.Loc == LocStack || d37.Loc == LocStackPair { ctx.EnsureDesc(&d37) }
			var d47 JITValueDesc
			if d37.Loc == LocImm {
				idx := int(d37.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d47 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d37.Reg] {
					ctx.ProtectReg(d37.Reg)
					seen[d37.Reg] = true
					protected = append(protected, d37.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					}
				}
				r28 := ctx.AllocReg()
				r29 := ctx.AllocRegExcept(r28)
				lbl31 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d37.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r28, ai.Reg)
						ctx.W.EmitMovRegReg(r29, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r28, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r29, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl31)
					ctx.W.MarkLabel(nextLbl)
				}
				panic("jitgen: dynamic args index out of range")
				ctx.W.MarkLabel(lbl31)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d47 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r28, Reg2: r29}
				ctx.BindReg(r28, &d47)
				ctx.BindReg(r29, &d47)
			}
			var d48 JITValueDesc
			if d47.Loc == LocImm {
				d48 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d47.Imm.Float())}
			} else if d47.Type == tagFloat && d47.Loc == LocRegPair {
				ctx.FreeReg(d47.Reg)
				d48 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d47.Reg2}
				ctx.BindReg(d47.Reg2, &d48)
				ctx.BindReg(d47.Reg2, &d48)
			} else if d47.Type == tagFloat && d47.Loc == LocReg {
				d48 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d47.Reg}
				ctx.BindReg(d47.Reg, &d48)
				ctx.BindReg(d47.Reg, &d48)
			} else {
				d48 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Float), []JITValueDesc{d47}, 1)
				d48.Type = tagFloat
				ctx.BindReg(d48.Reg, &d48)
			}
			ctx.FreeDesc(&d47)
			if d38.Loc == LocStack || d38.Loc == LocStackPair { ctx.EnsureDesc(&d38) }
			if d48.Loc == LocStack || d48.Loc == LocStackPair { ctx.EnsureDesc(&d48) }
			if d38.Loc == LocStack || d38.Loc == LocStackPair { ctx.EnsureDesc(&d38) }
			if d48.Loc == LocStack || d48.Loc == LocStackPair { ctx.EnsureDesc(&d48) }
			if d38.Loc == LocStack || d38.Loc == LocStackPair { ctx.EnsureDesc(&d38) }
			if d48.Loc == LocStack || d48.Loc == LocStackPair { ctx.EnsureDesc(&d48) }
			var d49 JITValueDesc
			if d38.Loc == LocImm && d48.Loc == LocImm {
				d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d38.Imm.Int() - d48.Imm.Int())}
			} else if d48.Loc == LocImm && d48.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r30, d38.Reg)
				d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
				ctx.BindReg(r30, &d49)
			} else if d38.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d48.Reg)
				d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else if d48.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d48.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else {
				r31 := ctx.AllocRegExcept(d38.Reg, d48.Reg)
				ctx.W.EmitMovRegReg(r31, d38.Reg)
				ctx.W.EmitSubInt64(r31, d48.Reg)
				d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r31}
				ctx.BindReg(r31, &d49)
			}
			if d49.Loc == LocReg && d38.Loc == LocReg && d49.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = LocNone
			}
			ctx.FreeDesc(&d48)
			if d37.Loc == LocStack || d37.Loc == LocStackPair { ctx.EnsureDesc(&d37) }
			var d50 JITValueDesc
			if d37.Loc == LocImm {
				d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(scratch, d37.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			}
			if d50.Loc == LocReg && d37.Loc == LocReg && d50.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = LocNone
			}
			d51 := d50
			if d51.Loc == LocNone { panic("jit: phi source has no location") }
			if d51.Loc == LocStack || d51.Loc == LocStackPair { ctx.EnsureDesc(&d51) }
			ctx.EmitStoreToStack(d51, 24)
			d52 := d49
			if d52.Loc == LocNone { panic("jit: phi source has no location") }
			if d52.Loc == LocStack || d52.Loc == LocStackPair { ctx.EnsureDesc(&d52) }
			ctx.EmitStoreToStack(d52, 32)
			ctx.W.EmitJmp(lbl26)
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
		nil /* TODO: unsupported call: archTrunc(x) */,
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
		nil /* TODO: Slice on non-desc: slice a[1:int:] */,
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
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */,
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
		nil /* TODO: unsupported compare const kind: 0:float64 */,
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
		nil /* TODO: unsupported compare const kind: 0:float64 */,
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
		nil /* TODO: unsupported compare const kind: 0:float64 */,
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			d1 := ctx.EmitTagEquals(&d0, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
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
		nil /* TODO: unsupported call: archFloor(x) */,
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
		nil /* TODO: unsupported call: archCeil(x) */,
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
		nil /* TODO: unsupported BinOp &^ */,
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
		nil /* TODO: unsupported compare const kind: 0:float64 */,
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
		nil /* TODO: unsupported compare const kind: 0:float64 */,
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
		nil /* TODO: Slice on non-desc: slice t0[:] */,
	})
}
