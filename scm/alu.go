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
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl8)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d6 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r7, Reg2: r8}
				ctx.BindReg(r7, &d6)
				ctx.BindReg(r8, &d6)
			}
			d8 := d6
			d7 := ctx.EmitTagEquals(&d8, tagInt, JITValueDesc{Loc: LocAny})
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			var d9 JITValueDesc
			if d0.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d0.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d0.Reg)
				d9 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d9)
			}
			lbl11 := ctx.W.ReserveLabel()
			d10 := d1
			if d10.Loc == LocNone { panic("jit: phi source has no location") }
			if d10.Loc == LocStack || d10.Loc == LocStackPair { ctx.EnsureDesc(&d10) }
			ctx.EmitStoreToStack(d10, 16)
			d11 := d9
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			if d11.Loc == LocStack || d11.Loc == LocStackPair { ctx.EnsureDesc(&d11) }
			ctx.EmitStoreToStack(d11, 24)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl11)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d14 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			var d15 JITValueDesc
			if d12.Loc == LocImm && d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < d14.Imm.Int())}
			} else if d14.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d12.Reg)
				if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d12.Reg, int32(d14.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
					ctx.W.EmitCmpInt64(d12.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d15)
			} else if d12.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d14.Reg)
				ctx.W.EmitSetcc(r10, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d15)
			} else {
				r11 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitCmpInt64(d12.Reg, d14.Reg)
				ctx.W.EmitSetcc(r11, CcL)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d15)
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
			ctx.W.MarkLabel(lbl5)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			ctx.W.EmitMakeInt(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl9)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			var d16 JITValueDesc
			if d6.Loc == LocImm {
				d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int())}
			} else if d6.Type == tagInt && d6.Loc == LocRegPair {
				ctx.FreeReg(d6.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2}
				ctx.BindReg(d6.Reg2, &d16)
				ctx.BindReg(d6.Reg2, &d16)
			} else if d6.Type == tagInt && d6.Loc == LocReg {
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg}
				ctx.BindReg(d6.Reg, &d16)
				ctx.BindReg(d6.Reg, &d16)
			} else {
				d16 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d6}, 1)
				d16.Type = tagInt
				ctx.BindReg(d16.Reg, &d16)
			}
			ctx.FreeDesc(&d6)
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d0.Loc == LocStack || d0.Loc == LocStackPair { ctx.EnsureDesc(&d0) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			var d17 JITValueDesc
			if d0.Loc == LocImm && d16.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d16.Imm.Int())}
			} else if d16.Loc == LocImm && d16.Imm.Int() == 0 {
				r12 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r12, d0.Reg)
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d17)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg}
				ctx.BindReg(d16.Reg, &d17)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d16.Reg)
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else if d16.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d16.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else {
				r13 := ctx.AllocRegExcept(d0.Reg, d16.Reg)
				ctx.W.EmitMovRegReg(r13, d0.Reg)
				ctx.W.EmitAddInt64(r13, d16.Reg)
				d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
				ctx.BindReg(r13, &d17)
			}
			if d17.Loc == LocReg && d0.Loc == LocReg && d17.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d16)
			if d1.Loc == LocStack || d1.Loc == LocStackPair { ctx.EnsureDesc(&d1) }
			var d18 JITValueDesc
			if d1.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			}
			if d18.Loc == LocReg && d1.Loc == LocReg && d18.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			d19 := d17
			if d19.Loc == LocNone { panic("jit: phi source has no location") }
			if d19.Loc == LocStack || d19.Loc == LocStackPair { ctx.EnsureDesc(&d19) }
			ctx.EmitStoreToStack(d19, 0)
			d20 := d18
			if d20.Loc == LocNone { panic("jit: phi source has no location") }
			if d20.Loc == LocStack || d20.Loc == LocStackPair { ctx.EnsureDesc(&d20) }
			ctx.EmitStoreToStack(d20, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl13)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			ctx.W.EmitMakeFloat(result, d13)
			if d13.Loc == LocReg { ctx.FreeReg(d13.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			var d21 JITValueDesc
			if d12.Loc == LocImm {
				idx := int(d12.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d21 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d12.Reg] {
					ctx.ProtectReg(d12.Reg)
					seen[d12.Reg] = true
					protected = append(protected, d12.Reg)
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
					ctx.W.EmitCmpRegImm32(d12.Reg, int32(i))
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
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl15)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d21 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r14, Reg2: r15}
				ctx.BindReg(r14, &d21)
				ctx.BindReg(r15, &d21)
			}
			d23 := d21
			d22 := ctx.EmitTagEquals(&d23, tagNil, JITValueDesc{Loc: LocAny})
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d22.Loc == LocImm {
				if d22.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d22.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d22)
			ctx.W.MarkLabel(lbl17)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			var d24 JITValueDesc
			if d21.Loc == LocImm {
				d24 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d21.Imm.Float())}
			} else if d21.Type == tagFloat && d21.Loc == LocReg {
				d24 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d21.Reg}
				ctx.BindReg(d21.Reg, &d24)
				ctx.BindReg(d21.Reg, &d24)
			} else if d21.Type == tagFloat && d21.Loc == LocRegPair {
				ctx.FreeReg(d21.Reg)
				d24 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d21.Reg2}
				ctx.BindReg(d21.Reg2, &d24)
				ctx.BindReg(d21.Reg2, &d24)
			} else {
				d24 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d21}, 1)
				d24.Type = tagFloat
				ctx.BindReg(d24.Reg, &d24)
			}
			ctx.FreeDesc(&d21)
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d24.Loc == LocStack || d24.Loc == LocStackPair { ctx.EnsureDesc(&d24) }
			if d13.Loc == LocStack || d13.Loc == LocStackPair { ctx.EnsureDesc(&d13) }
			if d24.Loc == LocStack || d24.Loc == LocStackPair { ctx.EnsureDesc(&d24) }
			var d25 JITValueDesc
			if d13.Loc == LocImm && d24.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d13.Imm.Float() + d24.Imm.Float())}
			} else if d13.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				_, xBits := d13.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitAddFloat64(scratch, d24.Reg)
				d25 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else if d24.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(scratch, d13.Reg)
				_, yBits := d24.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitAddFloat64(scratch, RegR11)
				d25 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else {
				r16 := ctx.AllocRegExcept(d13.Reg, d24.Reg)
				ctx.W.EmitMovRegReg(r16, d13.Reg)
				ctx.W.EmitAddFloat64(r16, d24.Reg)
				d25 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
				ctx.BindReg(r16, &d25)
			}
			if d25.Loc == LocReg && d13.Loc == LocReg && d25.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = LocNone
			}
			ctx.FreeDesc(&d24)
			if d12.Loc == LocStack || d12.Loc == LocStackPair { ctx.EnsureDesc(&d12) }
			var d26 JITValueDesc
			if d12.Loc == LocImm {
				d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(scratch, d12.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			}
			if d26.Loc == LocReg && d12.Loc == LocReg && d26.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = LocNone
			}
			d27 := d26
			if d27.Loc == LocNone { panic("jit: phi source has no location") }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			ctx.EmitStoreToStack(d27, 16)
			d28 := d25
			if d28.Loc == LocNone { panic("jit: phi source has no location") }
			if d28.Loc == LocStack || d28.Loc == LocStackPair { ctx.EnsureDesc(&d28) }
			ctx.EmitStoreToStack(d28, 24)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl16)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 := args[0]
			d6 := d4
			d5 := ctx.EmitTagEquals(&d6, tagInt, JITValueDesc{Loc: LocAny})
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if d2.Loc == LocStack || d2.Loc == LocStackPair { ctx.EnsureDesc(&d2) }
			var d7 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d7 = args[idx]
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
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl8)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d7 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d7)
				ctx.BindReg(r5, &d7)
			}
			d9 := d7
			d8 := ctx.EmitTagEquals(&d9, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d7)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d8.Loc == LocImm {
				if d8.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d10 := d2
			if d10.Loc == LocNone { panic("jit: phi source has no location") }
			if d10.Loc == LocStack || d10.Loc == LocStackPair { ctx.EnsureDesc(&d10) }
			ctx.EmitStoreToStack(d10, 0)
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d8.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
			d11 := d2
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			if d11.Loc == LocStack || d11.Loc == LocStackPair { ctx.EnsureDesc(&d11) }
			ctx.EmitStoreToStack(d11, 0)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d8)
			ctx.W.MarkLabel(lbl6)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d12 := args[0]
			var d13 JITValueDesc
			if d12.Loc == LocImm {
				d13 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d12.Imm.Float())}
			} else if d12.Type == tagFloat && d12.Loc == LocReg {
				d13 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d12.Reg}
				ctx.BindReg(d12.Reg, &d13)
				ctx.BindReg(d12.Reg, &d13)
			} else if d12.Type == tagFloat && d12.Loc == LocRegPair {
				ctx.FreeReg(d12.Reg)
				d13 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d12.Reg2}
				ctx.BindReg(d12.Reg2, &d13)
				ctx.BindReg(d12.Reg2, &d13)
			} else {
				d13 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d12}, 1)
				d13.Type = tagFloat
				ctx.BindReg(d13.Reg, &d13)
			}
			ctx.FreeDesc(&d12)
			lbl11 := ctx.W.ReserveLabel()
			d14 := d13
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			if d14.Loc == LocStack || d14.Loc == LocStackPair { ctx.EnsureDesc(&d14) }
			ctx.EmitStoreToStack(d14, 40)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl11)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d15 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d17 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d17.Loc == LocStack || d17.Loc == LocStackPair { ctx.EnsureDesc(&d17) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d17.Loc == LocStack || d17.Loc == LocStackPair { ctx.EnsureDesc(&d17) }
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			if d17.Loc == LocStack || d17.Loc == LocStackPair { ctx.EnsureDesc(&d17) }
			var d18 JITValueDesc
			if d16.Loc == LocImm && d17.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() < d17.Imm.Int())}
			} else if d17.Loc == LocImm {
				r6 := ctx.AllocRegExcept(d16.Reg)
				if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d16.Reg, int32(d17.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
					ctx.W.EmitCmpInt64(d16.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r6, CcL)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d18)
			} else if d16.Loc == LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d17.Reg)
				ctx.W.EmitSetcc(r7, CcL)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d18)
			} else {
				r8 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitCmpInt64(d16.Reg, d17.Reg)
				ctx.W.EmitSetcc(r8, CcL)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d18)
			}
			ctx.FreeDesc(&d17)
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d18.Loc == LocImm {
				if d18.Imm.Bool() {
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d18.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d18)
			ctx.W.MarkLabel(lbl5)
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 := args[0]
			var d20 JITValueDesc
			if d19.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int())}
			} else if d19.Type == tagInt && d19.Loc == LocRegPair {
				ctx.FreeReg(d19.Reg)
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2}
				ctx.BindReg(d19.Reg2, &d20)
				ctx.BindReg(d19.Reg2, &d20)
			} else if d19.Type == tagInt && d19.Loc == LocReg {
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg}
				ctx.BindReg(d19.Reg, &d20)
				ctx.BindReg(d19.Reg, &d20)
			} else {
				d20 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d19}, 1)
				d20.Type = tagInt
				ctx.BindReg(d20.Reg, &d20)
			}
			ctx.FreeDesc(&d19)
			lbl15 := ctx.W.ReserveLabel()
			d21 := d20
			if d21.Loc == LocNone { panic("jit: phi source has no location") }
			if d21.Loc == LocStack || d21.Loc == LocStackPair { ctx.EnsureDesc(&d21) }
			ctx.EmitStoreToStack(d21, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 16)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl15)
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d24 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			if d24.Loc == LocStack || d24.Loc == LocStackPair { ctx.EnsureDesc(&d24) }
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			if d24.Loc == LocStack || d24.Loc == LocStackPair { ctx.EnsureDesc(&d24) }
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			if d24.Loc == LocStack || d24.Loc == LocStackPair { ctx.EnsureDesc(&d24) }
			var d25 JITValueDesc
			if d23.Loc == LocImm && d24.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d24.Imm.Int())}
			} else if d24.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d23.Reg)
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d23.Reg, int32(d24.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d24.Imm.Int()))
					ctx.W.EmitCmpInt64(d23.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcL)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d25)
			} else if d23.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d24.Reg)
				ctx.W.EmitSetcc(r10, CcL)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d25)
			} else {
				r11 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitCmpInt64(d23.Reg, d24.Reg)
				ctx.W.EmitSetcc(r11, CcL)
				d25 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d25)
			}
			ctx.FreeDesc(&d24)
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d25.Loc == LocImm {
				if d25.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d25.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d25)
			ctx.W.MarkLabel(lbl9)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl13)
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			ctx.W.EmitMakeFloat(result, d15)
			if d15.Loc == LocReg { ctx.FreeReg(d15.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			var d26 JITValueDesc
			if d16.Loc == LocImm {
				idx := int(d16.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d26 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d16.Reg] {
					ctx.ProtectReg(d16.Reg)
					seen[d16.Reg] = true
					protected = append(protected, d16.Reg)
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
					ctx.W.EmitCmpRegImm32(d16.Reg, int32(i))
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
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl19)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d26 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r12, Reg2: r13}
				ctx.BindReg(r12, &d26)
				ctx.BindReg(r13, &d26)
			}
			var d27 JITValueDesc
			if d26.Loc == LocImm {
				d27 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d26.Imm.Float())}
			} else if d26.Type == tagFloat && d26.Loc == LocReg {
				d27 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d26.Reg}
				ctx.BindReg(d26.Reg, &d27)
				ctx.BindReg(d26.Reg, &d27)
			} else if d26.Type == tagFloat && d26.Loc == LocRegPair {
				ctx.FreeReg(d26.Reg)
				d27 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d26.Reg2}
				ctx.BindReg(d26.Reg2, &d27)
				ctx.BindReg(d26.Reg2, &d27)
			} else {
				d27 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d26}, 1)
				d27.Type = tagFloat
				ctx.BindReg(d27.Reg, &d27)
			}
			ctx.FreeDesc(&d26)
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			if d15.Loc == LocStack || d15.Loc == LocStackPair { ctx.EnsureDesc(&d15) }
			if d27.Loc == LocStack || d27.Loc == LocStackPair { ctx.EnsureDesc(&d27) }
			var d28 JITValueDesc
			if d15.Loc == LocImm && d27.Loc == LocImm {
				d28 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d15.Imm.Float() - d27.Imm.Float())}
			} else if d15.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				_, xBits := d15.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d27.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else if d27.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(scratch, d15.Reg)
				_, yBits := d27.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d28 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else {
				r14 := ctx.AllocRegExcept(d15.Reg, d27.Reg)
				ctx.W.EmitMovRegReg(r14, d15.Reg)
				ctx.W.EmitSubFloat64(r14, d27.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r14}
				ctx.BindReg(r14, &d28)
			}
			if d28.Loc == LocReg && d15.Loc == LocReg && d28.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = LocNone
			}
			ctx.FreeDesc(&d27)
			if d16.Loc == LocStack || d16.Loc == LocStackPair { ctx.EnsureDesc(&d16) }
			var d29 JITValueDesc
			if d16.Loc == LocImm {
				d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d16.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d29 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			}
			if d29.Loc == LocReg && d16.Loc == LocReg && d29.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = LocNone
			}
			d30 := d28
			if d30.Loc == LocNone { panic("jit: phi source has no location") }
			if d30.Loc == LocStack || d30.Loc == LocStackPair { ctx.EnsureDesc(&d30) }
			ctx.EmitStoreToStack(d30, 40)
			d31 := d29
			if d31.Loc == LocNone { panic("jit: phi source has no location") }
			if d31.Loc == LocStack || d31.Loc == LocStackPair { ctx.EnsureDesc(&d31) }
			ctx.EmitStoreToStack(d31, 48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl17)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d32 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			if d32.Loc == LocStack || d32.Loc == LocStackPair { ctx.EnsureDesc(&d32) }
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			if d32.Loc == LocStack || d32.Loc == LocStackPair { ctx.EnsureDesc(&d32) }
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			if d32.Loc == LocStack || d32.Loc == LocStackPair { ctx.EnsureDesc(&d32) }
			var d33 JITValueDesc
			if d23.Loc == LocImm && d32.Loc == LocImm {
				d33 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() == d32.Imm.Int())}
			} else if d32.Loc == LocImm {
				r15 := ctx.AllocRegExcept(d23.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d23.Reg, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitCmpInt64(d23.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r15, CcE)
				d33 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d33)
			} else if d23.Loc == LocImm {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d32.Reg)
				ctx.W.EmitSetcc(r16, CcE)
				d33 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d33)
			} else {
				r17 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitCmpInt64(d23.Reg, d32.Reg)
				ctx.W.EmitSetcc(r17, CcE)
				d33 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d33)
			}
			ctx.FreeDesc(&d32)
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d33.Loc == LocImm {
				if d33.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d33.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d33)
			ctx.W.MarkLabel(lbl16)
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			var d34 JITValueDesc
			if d23.Loc == LocImm {
				idx := int(d23.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d34 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d23.Reg] {
					ctx.ProtectReg(d23.Reg)
					seen[d23.Reg] = true
					protected = append(protected, d23.Reg)
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
				r18 := ctx.AllocReg()
				r19 := ctx.AllocRegExcept(r18)
				lbl23 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d23.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r18, ai.Reg)
						ctx.W.EmitMovRegReg(r19, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r18, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r19, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl23)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl23)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d34 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d34)
				ctx.BindReg(r19, &d34)
			}
			d36 := d34
			d35 := ctx.EmitTagEquals(&d36, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d34)
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d35.Loc == LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl25)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl21)
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			var d37 JITValueDesc
			if d22.Loc == LocImm {
				d37 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d22.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d22.Reg)
				d37 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d37)
			}
			lbl26 := ctx.W.ReserveLabel()
			d38 := d23
			if d38.Loc == LocNone { panic("jit: phi source has no location") }
			if d38.Loc == LocStack || d38.Loc == LocStackPair { ctx.EnsureDesc(&d38) }
			ctx.EmitStoreToStack(d38, 24)
			d39 := d37
			if d39.Loc == LocNone { panic("jit: phi source has no location") }
			if d39.Loc == LocStack || d39.Loc == LocStackPair { ctx.EnsureDesc(&d39) }
			ctx.EmitStoreToStack(d39, 32)
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl26)
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d40 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d41 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d42 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			if d40.Loc == LocStack || d40.Loc == LocStackPair { ctx.EnsureDesc(&d40) }
			if d42.Loc == LocStack || d42.Loc == LocStackPair { ctx.EnsureDesc(&d42) }
			if d40.Loc == LocStack || d40.Loc == LocStackPair { ctx.EnsureDesc(&d40) }
			if d42.Loc == LocStack || d42.Loc == LocStackPair { ctx.EnsureDesc(&d42) }
			if d40.Loc == LocStack || d40.Loc == LocStackPair { ctx.EnsureDesc(&d40) }
			if d42.Loc == LocStack || d42.Loc == LocStackPair { ctx.EnsureDesc(&d42) }
			var d43 JITValueDesc
			if d40.Loc == LocImm && d42.Loc == LocImm {
				d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d40.Imm.Int() < d42.Imm.Int())}
			} else if d42.Loc == LocImm {
				r20 := ctx.AllocRegExcept(d40.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d40.Reg, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitCmpInt64(d40.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r20, CcL)
				d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
				ctx.BindReg(r20, &d43)
			} else if d40.Loc == LocImm {
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d42.Reg)
				ctx.W.EmitSetcc(r21, CcL)
				d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
				ctx.BindReg(r21, &d43)
			} else {
				r22 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitCmpInt64(d40.Reg, d42.Reg)
				ctx.W.EmitSetcc(r22, CcL)
				d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d43)
			}
			ctx.FreeDesc(&d42)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d43.Loc == LocImm {
				if d43.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl29)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d43)
			ctx.W.MarkLabel(lbl20)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d40 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d41 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			ctx.W.EmitMakeInt(result, d22)
			if d22.Loc == LocReg { ctx.FreeReg(d22.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl24)
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d40 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d41 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			var d44 JITValueDesc
			if d23.Loc == LocImm {
				idx := int(d23.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d44 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d23.Reg] {
					ctx.ProtectReg(d23.Reg)
					seen[d23.Reg] = true
					protected = append(protected, d23.Reg)
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
				r23 := ctx.AllocReg()
				r24 := ctx.AllocRegExcept(r23)
				lbl30 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d23.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r23, ai.Reg)
						ctx.W.EmitMovRegReg(r24, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r23, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r24, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl30)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl30)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d44 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r23, Reg2: r24}
				ctx.BindReg(r23, &d44)
				ctx.BindReg(r24, &d44)
			}
			var d45 JITValueDesc
			if d44.Loc == LocImm {
				d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int())}
			} else if d44.Type == tagInt && d44.Loc == LocRegPair {
				ctx.FreeReg(d44.Reg)
				d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2}
				ctx.BindReg(d44.Reg2, &d45)
				ctx.BindReg(d44.Reg2, &d45)
			} else if d44.Type == tagInt && d44.Loc == LocReg {
				d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg}
				ctx.BindReg(d44.Reg, &d45)
				ctx.BindReg(d44.Reg, &d45)
			} else {
				d45 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d44}, 1)
				d45.Type = tagInt
				ctx.BindReg(d45.Reg, &d45)
			}
			ctx.FreeDesc(&d44)
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d45.Loc == LocStack || d45.Loc == LocStackPair { ctx.EnsureDesc(&d45) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d45.Loc == LocStack || d45.Loc == LocStackPair { ctx.EnsureDesc(&d45) }
			if d22.Loc == LocStack || d22.Loc == LocStackPair { ctx.EnsureDesc(&d22) }
			if d45.Loc == LocStack || d45.Loc == LocStackPair { ctx.EnsureDesc(&d45) }
			var d46 JITValueDesc
			if d22.Loc == LocImm && d45.Loc == LocImm {
				d46 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d22.Imm.Int() - d45.Imm.Int())}
			} else if d45.Loc == LocImm && d45.Imm.Int() == 0 {
				r25 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(r25, d22.Reg)
				d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
				ctx.BindReg(r25, &d46)
			} else if d22.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d45.Reg)
				d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else if d45.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(scratch, d22.Reg)
				if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d45.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d45.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else {
				r26 := ctx.AllocRegExcept(d22.Reg, d45.Reg)
				ctx.W.EmitMovRegReg(r26, d22.Reg)
				ctx.W.EmitSubInt64(r26, d45.Reg)
				d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
				ctx.BindReg(r26, &d46)
			}
			if d46.Loc == LocReg && d22.Loc == LocReg && d46.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = LocNone
			}
			ctx.FreeDesc(&d45)
			if d23.Loc == LocStack || d23.Loc == LocStackPair { ctx.EnsureDesc(&d23) }
			var d47 JITValueDesc
			if d23.Loc == LocImm {
				d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(scratch, d23.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			}
			if d47.Loc == LocReg && d23.Loc == LocReg && d47.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = LocNone
			}
			d48 := d46
			if d48.Loc == LocNone { panic("jit: phi source has no location") }
			if d48.Loc == LocStack || d48.Loc == LocStackPair { ctx.EnsureDesc(&d48) }
			ctx.EmitStoreToStack(d48, 8)
			d49 := d47
			if d49.Loc == LocNone { panic("jit: phi source has no location") }
			if d49.Loc == LocStack || d49.Loc == LocStackPair { ctx.EnsureDesc(&d49) }
			ctx.EmitStoreToStack(d49, 16)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl28)
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d40 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d41 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			if d41.Loc == LocStack || d41.Loc == LocStackPair { ctx.EnsureDesc(&d41) }
			if d41.Loc == LocStack || d41.Loc == LocStackPair { ctx.EnsureDesc(&d41) }
			ctx.W.EmitMakeFloat(result, d41)
			if d41.Loc == LocReg { ctx.FreeReg(d41.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl27)
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d40 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d41 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if d40.Loc == LocStack || d40.Loc == LocStackPair { ctx.EnsureDesc(&d40) }
			var d50 JITValueDesc
			if d40.Loc == LocImm {
				idx := int(d40.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d50 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d40.Reg] {
					ctx.ProtectReg(d40.Reg)
					seen[d40.Reg] = true
					protected = append(protected, d40.Reg)
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
				r27 := ctx.AllocReg()
				r28 := ctx.AllocRegExcept(r27)
				lbl31 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d40.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r27, ai.Reg)
						ctx.W.EmitMovRegReg(r28, ai.Reg2)
					case LocImm:
						ptrWord, auxWord := ai.Imm.RawWords()
						ctx.W.EmitMovRegImm64(r27, uint64(ptrWord))
						ctx.W.EmitMovRegImm64(r28, auxWord)
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl31)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl31)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d50 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r27, Reg2: r28}
				ctx.BindReg(r27, &d50)
				ctx.BindReg(r28, &d50)
			}
			var d51 JITValueDesc
			if d50.Loc == LocImm {
				d51 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d50.Imm.Float())}
			} else if d50.Type == tagFloat && d50.Loc == LocReg {
				d51 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d50.Reg}
				ctx.BindReg(d50.Reg, &d51)
				ctx.BindReg(d50.Reg, &d51)
			} else if d50.Type == tagFloat && d50.Loc == LocRegPair {
				ctx.FreeReg(d50.Reg)
				d51 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d50.Reg2}
				ctx.BindReg(d50.Reg2, &d51)
				ctx.BindReg(d50.Reg2, &d51)
			} else {
				d51 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d50}, 1)
				d51.Type = tagFloat
				ctx.BindReg(d51.Reg, &d51)
			}
			ctx.FreeDesc(&d50)
			if d41.Loc == LocStack || d41.Loc == LocStackPair { ctx.EnsureDesc(&d41) }
			if d51.Loc == LocStack || d51.Loc == LocStackPair { ctx.EnsureDesc(&d51) }
			if d41.Loc == LocStack || d41.Loc == LocStackPair { ctx.EnsureDesc(&d41) }
			if d51.Loc == LocStack || d51.Loc == LocStackPair { ctx.EnsureDesc(&d51) }
			var d52 JITValueDesc
			if d41.Loc == LocImm && d51.Loc == LocImm {
				d52 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d41.Imm.Float() - d51.Imm.Float())}
			} else if d41.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				_, xBits := d41.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(scratch, d41.Reg)
				_, yBits := d51.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d52 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else {
				r29 := ctx.AllocRegExcept(d41.Reg, d51.Reg)
				ctx.W.EmitMovRegReg(r29, d41.Reg)
				ctx.W.EmitSubFloat64(r29, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r29}
				ctx.BindReg(r29, &d52)
			}
			if d52.Loc == LocReg && d41.Loc == LocReg && d52.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = LocNone
			}
			ctx.FreeDesc(&d51)
			if d40.Loc == LocStack || d40.Loc == LocStackPair { ctx.EnsureDesc(&d40) }
			var d53 JITValueDesc
			if d40.Loc == LocImm {
				d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d40.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(scratch, d40.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d53)
			}
			if d53.Loc == LocReg && d40.Loc == LocReg && d53.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = LocNone
			}
			d54 := d53
			if d54.Loc == LocNone { panic("jit: phi source has no location") }
			if d54.Loc == LocStack || d54.Loc == LocStackPair { ctx.EnsureDesc(&d54) }
			ctx.EmitStoreToStack(d54, 24)
			d55 := d52
			if d55.Loc == LocNone { panic("jit: phi source has no location") }
			if d55.Loc == LocStack || d55.Loc == LocStackPair { ctx.EnsureDesc(&d55) }
			ctx.EmitStoreToStack(d55, 32)
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
		nil /* TODO: unsupported call: archTrunc(x) */, /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */
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
		nil /* TODO: Slice on non-desc: slice a[1:int:] */, /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t0[t3] */, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		nil /* TODO: IndexAddr on non-parameter: &t0[t3] */, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		nil /* TODO: IndexAddr on non-parameter: &t0[t3] */, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		nil /* TODO: IndexAddr on non-parameter: &t0[t3] */, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		nil /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */
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
		nil /* TODO: unsupported compare const kind: 0:float64 */, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		nil /* TODO: Index: s[t1] */, /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
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
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
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
		nil /* TODO: unsupported compare const kind: 0:float64 */, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		nil /* TODO: unsupported compare const kind: 0:float64 */, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
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
		nil /* TODO: IndexAddr on non-parameter: &t0[t3] */, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		nil /* TODO: IndexAddr on non-parameter: &t0[t3] */, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		nil /* TODO: unsupported call: archFloor(x) */, /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */
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
		nil /* TODO: unsupported call: archCeil(x) */, /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */
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
		nil /* TODO: unsupported BinOp &^ */, /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */
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
		nil /* TODO: unsupported compare const kind: 0:float64 */, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		nil /* TODO: unsupported compare const kind: 0:float64 */, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
}
