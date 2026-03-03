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
			ctx.EnsureDesc(&d1)
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
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
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
			ctx.EnsureDesc(&d1)
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
			ctx.EnsureDesc(&d1)
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
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			ctx.W.EmitMakeBool(result, d4)
			if d4.Loc == LocReg { ctx.FreeReg(d4.Reg) }
			result.Type = tagBool
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl4)
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d1)
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
			ctx.EnsureDesc(&d6)
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
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
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
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d4)
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
			ctx.EnsureDesc(&d1)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
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
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r7, tmp.Reg)
						ctx.W.EmitMovRegReg(r8, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
						ctx.BindReg(r7, &pair)
						ctx.BindReg(r8, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r7, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r8, auxWord)
						}
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
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
			ctx.EnsureDesc(&d10)
			ctx.EmitStoreToStack(d10, 16)
			d11 := d9
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d11)
			ctx.EmitStoreToStack(d11, 24)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl11)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d14 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d14)
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
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.W.EmitMakeInt(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl9)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			var d16 JITValueDesc
			if d6.Loc == LocImm {
				d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int())}
			} else if d6.Type == tagInt && d6.Loc == LocRegPair {
				ctx.FreeReg(d6.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2}
				ctx.BindReg(d6.Reg2, &d16)
				ctx.BindReg(d6.Reg2, &d16)
			} else if d6.Loc == LocRegPair {
				tmpTag := d6
				isInt := ctx.EmitTagEquals(&tmpTag, tagInt, JITValueDesc{Loc: LocAny})
				lblInt := ctx.W.ReserveLabel()
				lblFallback := ctx.W.ReserveLabel()
				lblJoin := ctx.W.ReserveLabel()
				if isInt.Loc == LocImm {
					if isInt.Imm.Bool() {
						ctx.W.EmitJmp(lblInt)
					} else {
						ctx.W.EmitJmp(lblFallback)
					}
				} else {
					ctx.W.EmitCmpRegImm32(isInt.Reg, 0)
					ctx.W.EmitJcc(CcNE, lblInt)
					ctx.W.EmitJmp(lblFallback)
				}
				ctx.FreeDesc(&isInt)
				ctx.W.MarkLabel(lblInt)
				ctx.FreeReg(d6.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2}
				ctx.BindReg(d6.Reg2, &d16)
				ctx.BindReg(d6.Reg2, &d16)
				ctx.W.EmitJmp(lblJoin)
				ctx.W.MarkLabel(lblFallback)
				d16 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d6}, 1)
				d16.Type = tagInt
				ctx.BindReg(d16.Reg, &d16)
				ctx.W.MarkLabel(lblJoin)
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
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d16)
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
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
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
			ctx.EnsureDesc(&d19)
			ctx.EmitStoreToStack(d19, 0)
			d20 := d18
			if d20.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl13)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d13)
			ctx.W.EmitMakeFloat(result, d13)
			if d13.Loc == LocReg { ctx.FreeReg(d13.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d12)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
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
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r14, tmp.Reg)
						ctx.W.EmitMovRegReg(r15, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r14, Reg2: r15}
						ctx.BindReg(r14, &pair)
						ctx.BindReg(r15, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r14, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r15, auxWord)
						}
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
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d24)
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
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d12)
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
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, 16)
			d28 := d25
			if d28.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			ctx.EmitStoreToStack(d28, 24)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl16)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d12 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
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
			// Binary hot path first, so JIT specialization for two arguments avoids
			// the variadic loop CFG entirely.
			if len(a) == 2 {
				if a[0].IsNil() || a[1].IsNil() {
					return NewNil()
				}
				if a[0].IsInt() && a[1].IsInt() {
					return NewInt(a[0].Int() - a[1].Int())
				}
				return NewFloat(a[0].Float() - a[1].Float())
			}
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
			ctx.EnsureDesc(&d0)
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Int() == 2)}
			} else {
				r1 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d0.Reg, 2)
				ctx.W.EmitSetcc(r1, CcE)
				d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d1)
			}
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d2 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl4 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 0)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl4)
			d3 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d4 JITValueDesc
			if d3.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d3.Reg, int32(1))
				d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg}
				ctx.BindReg(d3.Reg, &d4)
			}
			if d4.Loc == LocReg && d3.Loc == LocReg && d4.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = LocNone
			}
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			var d5 JITValueDesc
			if d4.Loc == LocImm && d2.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d2.Imm.Int())}
			} else if d2.Loc == LocImm {
				r2 := ctx.AllocRegExcept(d4.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d4.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d4.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r2, CcL)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d5)
			} else if d4.Loc == LocImm {
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d2.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d5)
			} else {
				r4 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitCmpInt64(d4.Reg, d2.Reg)
				ctx.W.EmitSetcc(r4, CcL)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d5)
			}
			ctx.FreeDesc(&d2)
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
			ctx.W.MarkLabel(lbl1)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d6 := args[0]
			d8 := d6
			d7 := ctx.EmitTagEquals(&d8, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d6)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl6)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d9 := args[0]
			d11 := d9
			d10 := ctx.EmitTagEquals(&d11, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d9)
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d10.Loc == LocImm {
				if d10.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d10.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl5)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d12 JITValueDesc
			if d4.Loc == LocImm {
				idx := int(d4.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d12 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d4.Reg] {
					ctx.ProtectReg(d4.Reg)
					seen[d4.Reg] = true
					protected = append(protected, d4.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r5 := ctx.AllocReg()
				r6 := ctx.AllocRegExcept(r5)
				lbl14 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d4.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r5, ai.Reg)
						ctx.W.EmitMovRegReg(r6, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r5, tmp.Reg)
						ctx.W.EmitMovRegReg(r6, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
						ctx.BindReg(r5, &pair)
						ctx.BindReg(r6, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r5, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r6, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl14)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl14)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d12 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d12)
				ctx.BindReg(r6, &d12)
			}
			d14 := d12
			d13 := ctx.EmitTagEquals(&d14, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d12)
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d13.Loc == LocImm {
				if d13.Imm.Bool() {
					ctx.W.EmitJmp(lbl15)
				} else {
			d15 := d4
			if d15.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d13.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl16)
			d16 := d4
			if d16.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, 0)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl15)
			}
			ctx.FreeDesc(&d13)
			ctx.W.MarkLabel(lbl9)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d17 := args[1]
			d19 := d17
			d18 := ctx.EmitTagEquals(&d19, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d17)
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d18.Loc == LocImm {
				if d18.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d18.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d18)
			ctx.W.MarkLabel(lbl8)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d20 := args[0]
			var d21 JITValueDesc
			if d20.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d20.Imm.Float())}
			} else if d20.Type == tagFloat && d20.Loc == LocReg {
				d21 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d20.Reg}
				ctx.BindReg(d20.Reg, &d21)
				ctx.BindReg(d20.Reg, &d21)
			} else if d20.Type == tagFloat && d20.Loc == LocRegPair {
				ctx.FreeReg(d20.Reg)
				d21 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d20.Reg2}
				ctx.BindReg(d20.Reg2, &d21)
				ctx.BindReg(d20.Reg2, &d21)
			} else {
				d21 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d20}, 1)
				d21.Type = tagFloat
				ctx.BindReg(d21.Reg, &d21)
			}
			ctx.FreeDesc(&d20)
			lbl19 := ctx.W.ReserveLabel()
			d22 := d21
			if d22.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d22)
			ctx.EmitStoreToStack(d22, 40)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 48)
			ctx.W.EmitJmp(lbl19)
			ctx.W.MarkLabel(lbl19)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d23 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d25 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d25)
			var d26 JITValueDesc
			if d24.Loc == LocImm && d25.Loc == LocImm {
				d26 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d24.Imm.Int() < d25.Imm.Int())}
			} else if d25.Loc == LocImm {
				r7 := ctx.AllocRegExcept(d24.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d24.Reg, int32(d25.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d25.Imm.Int()))
					ctx.W.EmitCmpInt64(d24.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r7, CcL)
				d26 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d26)
			} else if d24.Loc == LocImm {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d24.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d25.Reg)
				ctx.W.EmitSetcc(r8, CcL)
				d26 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d26)
			} else {
				r9 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitCmpInt64(d24.Reg, d25.Reg)
				ctx.W.EmitSetcc(r9, CcL)
				d26 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d26)
			}
			ctx.FreeDesc(&d25)
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d26.Loc == LocImm {
				if d26.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d26.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d26)
			ctx.W.MarkLabel(lbl11)
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d27 := args[0]
			var d28 JITValueDesc
			if d27.Loc == LocImm {
				d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d27.Imm.Int())}
			} else if d27.Type == tagInt && d27.Loc == LocRegPair {
				ctx.FreeReg(d27.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg2}
				ctx.BindReg(d27.Reg2, &d28)
				ctx.BindReg(d27.Reg2, &d28)
			} else if d27.Loc == LocRegPair {
				tmpTag := d27
				isInt := ctx.EmitTagEquals(&tmpTag, tagInt, JITValueDesc{Loc: LocAny})
				lblInt := ctx.W.ReserveLabel()
				lblFallback := ctx.W.ReserveLabel()
				lblJoin := ctx.W.ReserveLabel()
				if isInt.Loc == LocImm {
					if isInt.Imm.Bool() {
						ctx.W.EmitJmp(lblInt)
					} else {
						ctx.W.EmitJmp(lblFallback)
					}
				} else {
					ctx.W.EmitCmpRegImm32(isInt.Reg, 0)
					ctx.W.EmitJcc(CcNE, lblInt)
					ctx.W.EmitJmp(lblFallback)
				}
				ctx.FreeDesc(&isInt)
				ctx.W.MarkLabel(lblInt)
				ctx.FreeReg(d27.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg2}
				ctx.BindReg(d27.Reg2, &d28)
				ctx.BindReg(d27.Reg2, &d28)
				ctx.W.EmitJmp(lblJoin)
				ctx.W.MarkLabel(lblFallback)
				d28 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d27}, 1)
				d28.Type = tagInt
				ctx.BindReg(d28.Reg, &d28)
				ctx.W.MarkLabel(lblJoin)
			} else if d27.Type == tagInt && d27.Loc == LocReg {
				d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d27.Reg}
				ctx.BindReg(d27.Reg, &d28)
				ctx.BindReg(d27.Reg, &d28)
			} else {
				d28 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d27}, 1)
				d28.Type = tagInt
				ctx.BindReg(d28.Reg, &d28)
			}
			ctx.FreeDesc(&d27)
			lbl23 := ctx.W.ReserveLabel()
			d29 := d28
			if d29.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 16)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl23)
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d32 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d32)
			var d33 JITValueDesc
			if d31.Loc == LocImm && d32.Loc == LocImm {
				d33 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d31.Imm.Int() < d32.Imm.Int())}
			} else if d32.Loc == LocImm {
				r10 := ctx.AllocRegExcept(d31.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d31.Reg, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitCmpInt64(d31.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r10, CcL)
				d33 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d33)
			} else if d31.Loc == LocImm {
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d31.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d32.Reg)
				ctx.W.EmitSetcc(r11, CcL)
				d33 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d33)
			} else {
				r12 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitCmpInt64(d31.Reg, d32.Reg)
				ctx.W.EmitSetcc(r12, CcL)
				d33 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
				ctx.BindReg(r12, &d33)
			}
			ctx.FreeDesc(&d32)
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d33.Loc == LocImm {
				if d33.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d33.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl26)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d33)
			ctx.W.MarkLabel(lbl15)
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl17)
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d34 := args[0]
			d36 := d34
			d35 := ctx.EmitTagEquals(&d36, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d34)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d35.Loc == LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl29)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl21)
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			ctx.W.EmitMakeFloat(result, d23)
			if d23.Loc == LocReg { ctx.FreeReg(d23.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl20)
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d24)
			var d37 JITValueDesc
			if d24.Loc == LocImm {
				idx := int(d24.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d37 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d24.Reg] {
					ctx.ProtectReg(d24.Reg)
					seen[d24.Reg] = true
					protected = append(protected, d24.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r13 := ctx.AllocReg()
				r14 := ctx.AllocRegExcept(r13)
				lbl30 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d24.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r13, ai.Reg)
						ctx.W.EmitMovRegReg(r14, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r13, tmp.Reg)
						ctx.W.EmitMovRegReg(r14, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r13, Reg2: r14}
						ctx.BindReg(r13, &pair)
						ctx.BindReg(r14, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r13, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r14, auxWord)
						}
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
				d37 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r13, Reg2: r14}
				ctx.BindReg(r13, &d37)
				ctx.BindReg(r14, &d37)
			}
			var d38 JITValueDesc
			if d37.Loc == LocImm {
				d38 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d37.Imm.Float())}
			} else if d37.Type == tagFloat && d37.Loc == LocReg {
				d38 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d37.Reg}
				ctx.BindReg(d37.Reg, &d38)
				ctx.BindReg(d37.Reg, &d38)
			} else if d37.Type == tagFloat && d37.Loc == LocRegPair {
				ctx.FreeReg(d37.Reg)
				d38 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d37.Reg2}
				ctx.BindReg(d37.Reg2, &d38)
				ctx.BindReg(d37.Reg2, &d38)
			} else {
				d38 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d37}, 1)
				d38.Type = tagFloat
				ctx.BindReg(d38.Reg, &d38)
			}
			ctx.FreeDesc(&d37)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d38)
			var d39 JITValueDesc
			if d23.Loc == LocImm && d38.Loc == LocImm {
				d39 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d23.Imm.Float() - d38.Imm.Float())}
			} else if d23.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				_, xBits := d23.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d38.Reg)
				d39 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else if d38.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(scratch, d23.Reg)
				_, yBits := d38.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d39 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else {
				r15 := ctx.AllocRegExcept(d23.Reg, d38.Reg)
				ctx.W.EmitMovRegReg(r15, d23.Reg)
				ctx.W.EmitSubFloat64(r15, d38.Reg)
				d39 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r15}
				ctx.BindReg(r15, &d39)
			}
			if d39.Loc == LocReg && d23.Loc == LocReg && d39.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d40 JITValueDesc
			if d24.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d24.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(scratch, d24.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			}
			if d40.Loc == LocReg && d24.Loc == LocReg && d40.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = LocNone
			}
			d41 := d39
			if d41.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d41)
			ctx.EmitStoreToStack(d41, 40)
			d42 := d40
			if d42.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d42)
			ctx.EmitStoreToStack(d42, 48)
			ctx.W.EmitJmp(lbl19)
			ctx.W.MarkLabel(lbl25)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d43 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d43)
			var d44 JITValueDesc
			if d31.Loc == LocImm && d43.Loc == LocImm {
				d44 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d31.Imm.Int() == d43.Imm.Int())}
			} else if d43.Loc == LocImm {
				r16 := ctx.AllocRegExcept(d31.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d31.Reg, int32(d43.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
					ctx.W.EmitCmpInt64(d31.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r16, CcE)
				d44 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d44)
			} else if d31.Loc == LocImm {
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d31.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d43.Reg)
				ctx.W.EmitSetcc(r17, CcE)
				d44 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d44)
			} else {
				r18 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitCmpInt64(d31.Reg, d43.Reg)
				ctx.W.EmitSetcc(r18, CcE)
				d44 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
				ctx.BindReg(r18, &d44)
			}
			ctx.FreeDesc(&d43)
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d44.Loc == LocImm {
				if d44.Imm.Bool() {
					ctx.W.EmitJmp(lbl31)
				} else {
					ctx.W.EmitJmp(lbl32)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d44.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl33)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl33)
				ctx.W.EmitJmp(lbl31)
			}
			ctx.FreeDesc(&d44)
			ctx.W.MarkLabel(lbl24)
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d31)
			var d45 JITValueDesc
			if d31.Loc == LocImm {
				idx := int(d31.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d45 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d31.Reg] {
					ctx.ProtectReg(d31.Reg)
					seen[d31.Reg] = true
					protected = append(protected, d31.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r19 := ctx.AllocReg()
				r20 := ctx.AllocRegExcept(r19)
				lbl34 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d31.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r19, ai.Reg)
						ctx.W.EmitMovRegReg(r20, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r19, tmp.Reg)
						ctx.W.EmitMovRegReg(r20, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r19, Reg2: r20}
						ctx.BindReg(r19, &pair)
						ctx.BindReg(r20, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r19, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r20, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl34)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl34)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d45 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r19, Reg2: r20}
				ctx.BindReg(r19, &d45)
				ctx.BindReg(r20, &d45)
			}
			d47 := d45
			d46 := ctx.EmitTagEquals(&d47, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d45)
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			if d46.Loc == LocImm {
				if d46.Imm.Bool() {
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl36)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl36)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d46)
			ctx.W.MarkLabel(lbl28)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d48 := args[0]
			var d49 JITValueDesc
			if d48.Loc == LocImm {
				d49 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d48.Imm.Float())}
			} else if d48.Type == tagFloat && d48.Loc == LocReg {
				d49 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d48.Reg}
				ctx.BindReg(d48.Reg, &d49)
				ctx.BindReg(d48.Reg, &d49)
			} else if d48.Type == tagFloat && d48.Loc == LocRegPair {
				ctx.FreeReg(d48.Reg)
				d49 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d48.Reg2}
				ctx.BindReg(d48.Reg2, &d49)
				ctx.BindReg(d48.Reg2, &d49)
			} else {
				d49 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d48}, 1)
				d49.Type = tagFloat
				ctx.BindReg(d49.Reg, &d49)
			}
			ctx.FreeDesc(&d48)
			d50 := args[1]
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
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d51)
			var d52 JITValueDesc
			if d49.Loc == LocImm && d51.Loc == LocImm {
				d52 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d49.Imm.Float() - d51.Imm.Float())}
			} else if d49.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				_, xBits := d49.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(scratch, d49.Reg)
				_, yBits := d51.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d52 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else {
				r21 := ctx.AllocRegExcept(d49.Reg, d51.Reg)
				ctx.W.EmitMovRegReg(r21, d49.Reg)
				ctx.W.EmitSubFloat64(r21, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r21}
				ctx.BindReg(r21, &d52)
			}
			if d52.Loc == LocReg && d49.Loc == LocReg && d52.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.FreeDesc(&d51)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d52)
			ctx.W.EmitMakeFloat(result, d52)
			if d52.Loc == LocReg { ctx.FreeReg(d52.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl27)
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d53 := args[1]
			d55 := d53
			d54 := ctx.EmitTagEquals(&d55, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d53)
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			if d54.Loc == LocImm {
				if d54.Imm.Bool() {
					ctx.W.EmitJmp(lbl37)
				} else {
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d54.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl38)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl38)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d54)
			ctx.W.MarkLabel(lbl32)
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			var d56 JITValueDesc
			if d30.Loc == LocImm {
				d56 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d30.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d30.Reg)
				d56 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d30.Reg}
				ctx.BindReg(d30.Reg, &d56)
			}
			lbl39 := ctx.W.ReserveLabel()
			d57 := d31
			if d57.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			ctx.EmitStoreToStack(d57, 24)
			d58 := d56
			if d58.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d58)
			ctx.EmitStoreToStack(d58, 32)
			ctx.W.EmitJmp(lbl39)
			ctx.W.MarkLabel(lbl39)
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d59 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d60 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d61 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d61)
			var d62 JITValueDesc
			if d59.Loc == LocImm && d61.Loc == LocImm {
				d62 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d59.Imm.Int() < d61.Imm.Int())}
			} else if d61.Loc == LocImm {
				r22 := ctx.AllocRegExcept(d59.Reg)
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d59.Reg, int32(d61.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d61.Imm.Int()))
					ctx.W.EmitCmpInt64(d59.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r22, CcL)
				d62 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d62)
			} else if d59.Loc == LocImm {
				r23 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d59.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d61.Reg)
				ctx.W.EmitSetcc(r23, CcL)
				d62 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d62)
			} else {
				r24 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitCmpInt64(d59.Reg, d61.Reg)
				ctx.W.EmitSetcc(r24, CcL)
				d62 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
				ctx.BindReg(r24, &d62)
			}
			ctx.FreeDesc(&d61)
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d62.Loc == LocImm {
				if d62.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
					ctx.W.EmitJmp(lbl41)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d62.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl42)
				ctx.W.EmitJmp(lbl41)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d62)
			ctx.W.MarkLabel(lbl31)
			d59 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d60 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			ctx.W.EmitMakeInt(result, d30)
			if d30.Loc == LocReg { ctx.FreeReg(d30.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl35)
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d59 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d60 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d31)
			var d63 JITValueDesc
			if d31.Loc == LocImm {
				idx := int(d31.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d63 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d31.Reg] {
					ctx.ProtectReg(d31.Reg)
					seen[d31.Reg] = true
					protected = append(protected, d31.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r25 := ctx.AllocReg()
				r26 := ctx.AllocRegExcept(r25)
				lbl43 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d31.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r25, ai.Reg)
						ctx.W.EmitMovRegReg(r26, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r25, tmp.Reg)
						ctx.W.EmitMovRegReg(r26, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r25, Reg2: r26}
						ctx.BindReg(r25, &pair)
						ctx.BindReg(r26, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r25, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r26, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl43)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl43)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d63 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r25, Reg2: r26}
				ctx.BindReg(r25, &d63)
				ctx.BindReg(r26, &d63)
			}
			var d64 JITValueDesc
			if d63.Loc == LocImm {
				d64 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d63.Imm.Int())}
			} else if d63.Type == tagInt && d63.Loc == LocRegPair {
				ctx.FreeReg(d63.Reg)
				d64 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d63.Reg2}
				ctx.BindReg(d63.Reg2, &d64)
				ctx.BindReg(d63.Reg2, &d64)
			} else if d63.Loc == LocRegPair {
				tmpTag := d63
				isInt := ctx.EmitTagEquals(&tmpTag, tagInt, JITValueDesc{Loc: LocAny})
				lblInt := ctx.W.ReserveLabel()
				lblFallback := ctx.W.ReserveLabel()
				lblJoin := ctx.W.ReserveLabel()
				if isInt.Loc == LocImm {
					if isInt.Imm.Bool() {
						ctx.W.EmitJmp(lblInt)
					} else {
						ctx.W.EmitJmp(lblFallback)
					}
				} else {
					ctx.W.EmitCmpRegImm32(isInt.Reg, 0)
					ctx.W.EmitJcc(CcNE, lblInt)
					ctx.W.EmitJmp(lblFallback)
				}
				ctx.FreeDesc(&isInt)
				ctx.W.MarkLabel(lblInt)
				ctx.FreeReg(d63.Reg)
				d64 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d63.Reg2}
				ctx.BindReg(d63.Reg2, &d64)
				ctx.BindReg(d63.Reg2, &d64)
				ctx.W.EmitJmp(lblJoin)
				ctx.W.MarkLabel(lblFallback)
				d64 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d63}, 1)
				d64.Type = tagInt
				ctx.BindReg(d64.Reg, &d64)
				ctx.W.MarkLabel(lblJoin)
			} else if d63.Type == tagInt && d63.Loc == LocReg {
				d64 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d63.Reg}
				ctx.BindReg(d63.Reg, &d64)
				ctx.BindReg(d63.Reg, &d64)
			} else {
				d64 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d63}, 1)
				d64.Type = tagInt
				ctx.BindReg(d64.Reg, &d64)
			}
			ctx.FreeDesc(&d63)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d64)
			var d65 JITValueDesc
			if d30.Loc == LocImm && d64.Loc == LocImm {
				d65 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d30.Imm.Int() - d64.Imm.Int())}
			} else if d64.Loc == LocImm && d64.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r27, d30.Reg)
				d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r27}
				ctx.BindReg(r27, &d65)
			} else if d30.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d64.Reg)
				d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else if d64.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(scratch, d30.Reg)
				if d64.Imm.Int() >= -2147483648 && d64.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d64.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d64.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else {
				r28 := ctx.AllocRegExcept(d30.Reg, d64.Reg)
				ctx.W.EmitMovRegReg(r28, d30.Reg)
				ctx.W.EmitSubInt64(r28, d64.Reg)
				d65 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r28}
				ctx.BindReg(r28, &d65)
			}
			if d65.Loc == LocReg && d30.Loc == LocReg && d65.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = LocNone
			}
			ctx.FreeDesc(&d64)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d31)
			var d66 JITValueDesc
			if d31.Loc == LocImm {
				d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(scratch, d31.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d66)
			}
			if d66.Loc == LocReg && d31.Loc == LocReg && d66.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = LocNone
			}
			d67 := d65
			if d67.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d67)
			ctx.EmitStoreToStack(d67, 8)
			d68 := d66
			if d68.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d68)
			ctx.EmitStoreToStack(d68, 16)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl37)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d59 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d60 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d69 := args[0]
			var d70 JITValueDesc
			if d69.Loc == LocImm {
				d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d69.Imm.Int())}
			} else if d69.Type == tagInt && d69.Loc == LocRegPair {
				ctx.FreeReg(d69.Reg)
				d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg2}
				ctx.BindReg(d69.Reg2, &d70)
				ctx.BindReg(d69.Reg2, &d70)
			} else if d69.Loc == LocRegPair {
				tmpTag := d69
				isInt := ctx.EmitTagEquals(&tmpTag, tagInt, JITValueDesc{Loc: LocAny})
				lblInt := ctx.W.ReserveLabel()
				lblFallback := ctx.W.ReserveLabel()
				lblJoin := ctx.W.ReserveLabel()
				if isInt.Loc == LocImm {
					if isInt.Imm.Bool() {
						ctx.W.EmitJmp(lblInt)
					} else {
						ctx.W.EmitJmp(lblFallback)
					}
				} else {
					ctx.W.EmitCmpRegImm32(isInt.Reg, 0)
					ctx.W.EmitJcc(CcNE, lblInt)
					ctx.W.EmitJmp(lblFallback)
				}
				ctx.FreeDesc(&isInt)
				ctx.W.MarkLabel(lblInt)
				ctx.FreeReg(d69.Reg)
				d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg2}
				ctx.BindReg(d69.Reg2, &d70)
				ctx.BindReg(d69.Reg2, &d70)
				ctx.W.EmitJmp(lblJoin)
				ctx.W.MarkLabel(lblFallback)
				d70 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d69}, 1)
				d70.Type = tagInt
				ctx.BindReg(d70.Reg, &d70)
				ctx.W.MarkLabel(lblJoin)
			} else if d69.Type == tagInt && d69.Loc == LocReg {
				d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d69.Reg}
				ctx.BindReg(d69.Reg, &d70)
				ctx.BindReg(d69.Reg, &d70)
			} else {
				d70 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d69}, 1)
				d70.Type = tagInt
				ctx.BindReg(d70.Reg, &d70)
			}
			ctx.FreeDesc(&d69)
			d71 := args[1]
			var d72 JITValueDesc
			if d71.Loc == LocImm {
				d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int())}
			} else if d71.Type == tagInt && d71.Loc == LocRegPair {
				ctx.FreeReg(d71.Reg)
				d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg2}
				ctx.BindReg(d71.Reg2, &d72)
				ctx.BindReg(d71.Reg2, &d72)
			} else if d71.Loc == LocRegPair {
				tmpTag := d71
				isInt := ctx.EmitTagEquals(&tmpTag, tagInt, JITValueDesc{Loc: LocAny})
				lblInt := ctx.W.ReserveLabel()
				lblFallback := ctx.W.ReserveLabel()
				lblJoin := ctx.W.ReserveLabel()
				if isInt.Loc == LocImm {
					if isInt.Imm.Bool() {
						ctx.W.EmitJmp(lblInt)
					} else {
						ctx.W.EmitJmp(lblFallback)
					}
				} else {
					ctx.W.EmitCmpRegImm32(isInt.Reg, 0)
					ctx.W.EmitJcc(CcNE, lblInt)
					ctx.W.EmitJmp(lblFallback)
				}
				ctx.FreeDesc(&isInt)
				ctx.W.MarkLabel(lblInt)
				ctx.FreeReg(d71.Reg)
				d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg2}
				ctx.BindReg(d71.Reg2, &d72)
				ctx.BindReg(d71.Reg2, &d72)
				ctx.W.EmitJmp(lblJoin)
				ctx.W.MarkLabel(lblFallback)
				d72 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d71}, 1)
				d72.Type = tagInt
				ctx.BindReg(d72.Reg, &d72)
				ctx.W.MarkLabel(lblJoin)
			} else if d71.Type == tagInt && d71.Loc == LocReg {
				d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
				ctx.BindReg(d71.Reg, &d72)
				ctx.BindReg(d71.Reg, &d72)
			} else {
				d72 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d71}, 1)
				d72.Type = tagInt
				ctx.BindReg(d72.Reg, &d72)
			}
			ctx.FreeDesc(&d71)
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d72)
			var d73 JITValueDesc
			if d70.Loc == LocImm && d72.Loc == LocImm {
				d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() - d72.Imm.Int())}
			} else if d72.Loc == LocImm && d72.Imm.Int() == 0 {
				r29 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(r29, d70.Reg)
				d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r29}
				ctx.BindReg(r29, &d73)
			} else if d70.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d70.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d72.Reg)
				d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else if d72.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(scratch, d70.Reg)
				if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d72.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d72.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else {
				r30 := ctx.AllocRegExcept(d70.Reg, d72.Reg)
				ctx.W.EmitMovRegReg(r30, d70.Reg)
				ctx.W.EmitSubInt64(r30, d72.Reg)
				d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r30}
				ctx.BindReg(r30, &d73)
			}
			if d73.Loc == LocReg && d70.Loc == LocReg && d73.Reg == d70.Reg {
				ctx.TransferReg(d70.Reg)
				d70.Loc = LocNone
			}
			ctx.FreeDesc(&d70)
			ctx.FreeDesc(&d72)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d73)
			ctx.W.EmitMakeInt(result, d73)
			if d73.Loc == LocReg { ctx.FreeReg(d73.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl41)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d59 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d60 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d60)
			ctx.W.EmitMakeFloat(result, d60)
			if d60.Loc == LocReg { ctx.FreeReg(d60.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl40)
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d31 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d59 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d60 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d23 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d24 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d59)
			var d74 JITValueDesc
			if d59.Loc == LocImm {
				idx := int(d59.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d74 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d59.Reg] {
					ctx.ProtectReg(d59.Reg)
					seen[d59.Reg] = true
					protected = append(protected, d59.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r31 := ctx.AllocReg()
				r32 := ctx.AllocRegExcept(r31)
				lbl44 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d59.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r31, ai.Reg)
						ctx.W.EmitMovRegReg(r32, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r31, tmp.Reg)
						ctx.W.EmitMovRegReg(r32, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
						ctx.BindReg(r31, &pair)
						ctx.BindReg(r32, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r31, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r32, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl44)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl44)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d74 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r31, Reg2: r32}
				ctx.BindReg(r31, &d74)
				ctx.BindReg(r32, &d74)
			}
			var d75 JITValueDesc
			if d74.Loc == LocImm {
				d75 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d74.Imm.Float())}
			} else if d74.Type == tagFloat && d74.Loc == LocReg {
				d75 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d74.Reg}
				ctx.BindReg(d74.Reg, &d75)
				ctx.BindReg(d74.Reg, &d75)
			} else if d74.Type == tagFloat && d74.Loc == LocRegPair {
				ctx.FreeReg(d74.Reg)
				d75 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d74.Reg2}
				ctx.BindReg(d74.Reg2, &d75)
				ctx.BindReg(d74.Reg2, &d75)
			} else {
				d75 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d74}, 1)
				d75.Type = tagFloat
				ctx.BindReg(d75.Reg, &d75)
			}
			ctx.FreeDesc(&d74)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d75)
			var d76 JITValueDesc
			if d60.Loc == LocImm && d75.Loc == LocImm {
				d76 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d60.Imm.Float() - d75.Imm.Float())}
			} else if d60.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d75.Reg)
				_, xBits := d60.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d75.Reg)
				d76 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			} else if d75.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegReg(scratch, d60.Reg)
				_, yBits := d75.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d76 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			} else {
				r33 := ctx.AllocRegExcept(d60.Reg, d75.Reg)
				ctx.W.EmitMovRegReg(r33, d60.Reg)
				ctx.W.EmitSubFloat64(r33, d75.Reg)
				d76 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r33}
				ctx.BindReg(r33, &d76)
			}
			if d76.Loc == LocReg && d60.Loc == LocReg && d76.Reg == d60.Reg {
				ctx.TransferReg(d60.Reg)
				d60.Loc = LocNone
			}
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d59)
			var d77 JITValueDesc
			if d59.Loc == LocImm {
				d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d59.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(scratch, d59.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			}
			if d77.Loc == LocReg && d59.Loc == LocReg && d77.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = LocNone
			}
			d78 := d77
			if d78.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d78)
			ctx.EmitStoreToStack(d78, 24)
			d79 := d76
			if d79.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d79)
			ctx.EmitStoreToStack(d79, 32)
			ctx.W.EmitJmp(lbl39)
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
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
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
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d0)
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
			lbl5 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 16)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl5)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d6 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d6)
			var d7 JITValueDesc
			if d5.Loc == LocImm && d6.Loc == LocImm {
				d7 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() < d6.Imm.Int())}
			} else if d6.Loc == LocImm {
				r4 := ctx.AllocRegExcept(d5.Reg)
				if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d5.Reg, int32(d6.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
					ctx.W.EmitCmpInt64(d5.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r4, CcL)
				d7 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d7)
			} else if d5.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d6.Reg)
				ctx.W.EmitSetcc(r5, CcL)
				d7 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d7)
			} else {
				r6 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitCmpInt64(d5.Reg, d6.Reg)
				ctx.W.EmitSetcc(r6, CcL)
				d7 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d7)
			}
			ctx.FreeDesc(&d6)
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl8)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl2)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d2)
			var d8 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d8 = args[idx]
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r7 := ctx.AllocReg()
				r8 := ctx.AllocRegExcept(r7)
				lbl9 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r7, ai.Reg)
						ctx.W.EmitMovRegReg(r8, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r7, tmp.Reg)
						ctx.W.EmitMovRegReg(r8, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
						ctx.BindReg(r7, &pair)
						ctx.BindReg(r8, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r7, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r8, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl9)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl9)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d8 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r7, Reg2: r8}
				ctx.BindReg(r7, &d8)
				ctx.BindReg(r8, &d8)
			}
			d10 := d8
			d9 := ctx.EmitTagEquals(&d10, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d8)
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d9.Loc == LocImm {
				if d9.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
			d11 := d2
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d11)
			ctx.EmitStoreToStack(d11, 0)
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl11)
			d12 := d2
			if d12.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			ctx.EmitStoreToStack(d12, 0)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d9)
			ctx.W.MarkLabel(lbl7)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d13 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d13)
			var d14 JITValueDesc
			if d5.Loc == LocImm && d13.Loc == LocImm {
				d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Int() == d13.Imm.Int())}
			} else if d13.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d5.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d5.Reg, int32(d13.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
					ctx.W.EmitCmpInt64(d5.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcE)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d14)
			} else if d5.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d13.Reg)
				ctx.W.EmitSetcc(r10, CcE)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d14)
			} else {
				r11 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitCmpInt64(d5.Reg, d13.Reg)
				ctx.W.EmitSetcc(r11, CcE)
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
			ctx.W.MarkLabel(lbl6)
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d5)
			var d15 JITValueDesc
			if d5.Loc == LocImm {
				idx := int(d5.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d15 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d5.Reg] {
					ctx.ProtectReg(d5.Reg)
					seen[d5.Reg] = true
					protected = append(protected, d5.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r12 := ctx.AllocReg()
				r13 := ctx.AllocRegExcept(r12)
				lbl15 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d5.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r12, ai.Reg)
						ctx.W.EmitMovRegReg(r13, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r12, tmp.Reg)
						ctx.W.EmitMovRegReg(r13, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r12, Reg2: r13}
						ctx.BindReg(r12, &pair)
						ctx.BindReg(r13, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r12, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r13, auxWord)
						}
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
				d15 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r12, Reg2: r13}
				ctx.BindReg(r12, &d15)
				ctx.BindReg(r13, &d15)
			}
			d17 := d15
			d16 := ctx.EmitTagEquals(&d17, tagInt, JITValueDesc{Loc: LocAny})
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d16.Loc == LocImm {
				if d16.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl10)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl13)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d18 JITValueDesc
			if d4.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d4.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d4.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d4.Reg}
				ctx.BindReg(d4.Reg, &d18)
			}
			lbl19 := ctx.W.ReserveLabel()
			d19 := d5
			if d19.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d19)
			ctx.EmitStoreToStack(d19, 32)
			d20 := d18
			if d20.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, 40)
			ctx.W.EmitJmp(lbl19)
			ctx.W.MarkLabel(lbl19)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d21 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d23 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d23)
			var d24 JITValueDesc
			if d21.Loc == LocImm && d23.Loc == LocImm {
				d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() < d23.Imm.Int())}
			} else if d23.Loc == LocImm {
				r14 := ctx.AllocRegExcept(d21.Reg)
				if d23.Imm.Int() >= -2147483648 && d23.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d21.Reg, int32(d23.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
					ctx.W.EmitCmpInt64(d21.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r14, CcL)
				d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
				ctx.BindReg(r14, &d24)
			} else if d21.Loc == LocImm {
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d23.Reg)
				ctx.W.EmitSetcc(r15, CcL)
				d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d24)
			} else {
				r16 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitCmpInt64(d21.Reg, d23.Reg)
				ctx.W.EmitSetcc(r16, CcL)
				d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d24)
			}
			ctx.FreeDesc(&d23)
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d24.Loc == LocImm {
				if d24.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d24.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d24)
			ctx.W.MarkLabel(lbl12)
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			ctx.W.EmitMakeInt(result, d4)
			if d4.Loc == LocReg { ctx.FreeReg(d4.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl17)
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d26 := d15
			d25 := ctx.EmitTagEquals(&d26, tagFloat, JITValueDesc{Loc: LocAny})
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			if d25.Loc == LocImm {
				if d25.Imm.Bool() {
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d25.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl24)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl24)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d25)
			ctx.W.MarkLabel(lbl16)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			var d27 JITValueDesc
			if d15.Loc == LocImm {
				d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int())}
			} else if d15.Type == tagInt && d15.Loc == LocRegPair {
				ctx.FreeReg(d15.Reg)
				d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg2}
				ctx.BindReg(d15.Reg2, &d27)
				ctx.BindReg(d15.Reg2, &d27)
			} else if d15.Loc == LocRegPair {
				tmpTag := d15
				isInt := ctx.EmitTagEquals(&tmpTag, tagInt, JITValueDesc{Loc: LocAny})
				lblInt := ctx.W.ReserveLabel()
				lblFallback := ctx.W.ReserveLabel()
				lblJoin := ctx.W.ReserveLabel()
				if isInt.Loc == LocImm {
					if isInt.Imm.Bool() {
						ctx.W.EmitJmp(lblInt)
					} else {
						ctx.W.EmitJmp(lblFallback)
					}
				} else {
					ctx.W.EmitCmpRegImm32(isInt.Reg, 0)
					ctx.W.EmitJcc(CcNE, lblInt)
					ctx.W.EmitJmp(lblFallback)
				}
				ctx.FreeDesc(&isInt)
				ctx.W.MarkLabel(lblInt)
				ctx.FreeReg(d15.Reg)
				d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg2}
				ctx.BindReg(d15.Reg2, &d27)
				ctx.BindReg(d15.Reg2, &d27)
				ctx.W.EmitJmp(lblJoin)
				ctx.W.MarkLabel(lblFallback)
				d27 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d15}, 1)
				d27.Type = tagInt
				ctx.BindReg(d27.Reg, &d27)
				ctx.W.MarkLabel(lblJoin)
			} else if d15.Type == tagInt && d15.Loc == LocReg {
				d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d15.Reg}
				ctx.BindReg(d15.Reg, &d27)
				ctx.BindReg(d15.Reg, &d27)
			} else {
				d27 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d15}, 1)
				d27.Type = tagInt
				ctx.BindReg(d27.Reg, &d27)
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d27)
			var d28 JITValueDesc
			if d4.Loc == LocImm && d27.Loc == LocImm {
				d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() * d27.Imm.Int())}
			} else if d4.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d27.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else if d27.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d27.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d27.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, RegR11)
				}
				d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else {
				r17 := ctx.AllocRegExcept(d4.Reg, d27.Reg)
				ctx.W.EmitMovRegReg(r17, d4.Reg)
				ctx.W.EmitImulInt64(r17, d27.Reg)
				d28 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
				ctx.BindReg(r17, &d28)
			}
			if d28.Loc == LocReg && d4.Loc == LocReg && d28.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.FreeDesc(&d27)
			lbl25 := ctx.W.ReserveLabel()
			d29 := d28
			if d29.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 24)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl25)
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d30 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			var d31 JITValueDesc
			if d5.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			}
			if d31.Loc == LocReg && d5.Loc == LocReg && d31.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			d32 := d30
			if d32.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d32)
			ctx.EmitStoreToStack(d32, 8)
			d33 := d31
			if d33.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d33)
			ctx.EmitStoreToStack(d33, 16)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl21)
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			ctx.W.EmitMakeFloat(result, d22)
			if d22.Loc == LocReg { ctx.FreeReg(d22.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl20)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d21)
			var d34 JITValueDesc
			if d21.Loc == LocImm {
				idx := int(d21.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d34 = args[idx]
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r18 := ctx.AllocReg()
				r19 := ctx.AllocRegExcept(r18)
				lbl26 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d21.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r18, ai.Reg)
						ctx.W.EmitMovRegReg(r19, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r18, tmp.Reg)
						ctx.W.EmitMovRegReg(r19, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r18, Reg2: r19}
						ctx.BindReg(r18, &pair)
						ctx.BindReg(r19, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r18, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r19, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl26)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl26)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d34 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d34)
				ctx.BindReg(r19, &d34)
			}
			var d35 JITValueDesc
			if d34.Loc == LocImm {
				d35 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d34.Imm.Float())}
			} else if d34.Type == tagFloat && d34.Loc == LocReg {
				d35 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d34.Reg}
				ctx.BindReg(d34.Reg, &d35)
				ctx.BindReg(d34.Reg, &d35)
			} else if d34.Type == tagFloat && d34.Loc == LocRegPair {
				ctx.FreeReg(d34.Reg)
				d35 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d34.Reg2}
				ctx.BindReg(d34.Reg2, &d35)
				ctx.BindReg(d34.Reg2, &d35)
			} else {
				d35 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d34}, 1)
				d35.Type = tagFloat
				ctx.BindReg(d35.Reg, &d35)
			}
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d35)
			var d36 JITValueDesc
			if d22.Loc == LocImm && d35.Loc == LocImm {
				d36 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d22.Imm.Float() * d35.Imm.Float())}
			} else if d22.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d35.Reg)
				_, xBits := d22.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitMulFloat64(scratch, d35.Reg)
				d36 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d35.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(scratch, d22.Reg)
				_, yBits := d35.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitMulFloat64(scratch, RegR11)
				d36 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else {
				r20 := ctx.AllocRegExcept(d22.Reg, d35.Reg)
				ctx.W.EmitMovRegReg(r20, d22.Reg)
				ctx.W.EmitMulFloat64(r20, d35.Reg)
				d36 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r20}
				ctx.BindReg(r20, &d36)
			}
			if d36.Loc == LocReg && d22.Loc == LocReg && d36.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			var d37 JITValueDesc
			if d21.Loc == LocImm {
				d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d21.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(scratch, d21.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d37)
			}
			if d37.Loc == LocReg && d21.Loc == LocReg && d37.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = LocNone
			}
			d38 := d37
			if d38.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d38)
			ctx.EmitStoreToStack(d38, 32)
			d39 := d36
			if d39.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d39)
			ctx.EmitStoreToStack(d39, 40)
			ctx.W.EmitJmp(lbl19)
			ctx.W.MarkLabel(lbl23)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			var d40 JITValueDesc
			if d15.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d15.Imm.Float())}
			} else if d15.Type == tagFloat && d15.Loc == LocReg {
				d40 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d15.Reg}
				ctx.BindReg(d15.Reg, &d40)
				ctx.BindReg(d15.Reg, &d40)
			} else if d15.Type == tagFloat && d15.Loc == LocRegPair {
				ctx.FreeReg(d15.Reg)
				d40 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d15.Reg2}
				ctx.BindReg(d15.Reg2, &d40)
				ctx.BindReg(d15.Reg2, &d40)
			} else {
				d40 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d15}, 1)
				d40.Type = tagFloat
				ctx.BindReg(d40.Reg, &d40)
			}
			ctx.FreeDesc(&d15)
			ctx.EnsureDesc(&d40)
			var d41 JITValueDesc
			if d40.Loc == LocImm {
				d41 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d40.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d40)
				var truncSrc Reg
				if d40.Loc == LocRegPair {
					ctx.FreeReg(d40.Reg)
					truncSrc = d40.Reg2
				} else {
					truncSrc = d40.Reg
				}
				truncInt := ctx.AllocRegExcept(truncSrc)
				ctx.W.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
				ctx.W.EmitCvtInt64ToFloat64(RegX0, truncInt)
				d41 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
				ctx.BindReg(truncInt, &d41)
				ctx.BindReg(truncInt, &d41)
			}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			var d42 JITValueDesc
			if d40.Loc == LocImm && d41.Loc == LocImm {
				d42 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d40.Imm.Float() == d41.Imm.Float())}
			} else if d41.Loc == LocImm {
				r21 := ctx.AllocRegExcept(d40.Reg)
				_, yBits := d41.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitCmpFloat64Setcc(r21, d40.Reg, RegR11, CcE)
				d42 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
				ctx.BindReg(r21, &d42)
			} else if d40.Loc == LocImm {
				r22 := ctx.AllocRegExcept(d41.Reg)
				_, xBits := d40.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, xBits)
				ctx.W.EmitCmpFloat64Setcc(r22, RegR11, d41.Reg, CcE)
				d42 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d42)
			} else {
				r23 := ctx.AllocRegExcept(d40.Reg, d41.Reg)
				ctx.W.EmitCmpFloat64Setcc(r23, d40.Reg, d41.Reg, CcE)
				d42 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d42)
			}
			ctx.FreeDesc(&d41)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			if d42.Loc == LocImm {
				if d42.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d42.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl28)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d42)
			ctx.W.MarkLabel(lbl27)
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d30 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d22 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d43 JITValueDesc
			if d40.Loc == LocImm {
				d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.Imm.Float()))}
			} else {
				r24 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r24, d40.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
				ctx.BindReg(r24, &d43)
			}
			ctx.FreeDesc(&d40)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d43)
			var d44 JITValueDesc
			if d4.Loc == LocImm && d43.Loc == LocImm {
				d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() * d43.Imm.Int())}
			} else if d4.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d43.Reg)
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else if d43.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d43.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d43.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, RegR11)
				}
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else {
				r25 := ctx.AllocRegExcept(d4.Reg, d43.Reg)
				ctx.W.EmitMovRegReg(r25, d4.Reg)
				ctx.W.EmitImulInt64(r25, d43.Reg)
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
				ctx.BindReg(r25, &d44)
			}
			if d44.Loc == LocReg && d4.Loc == LocReg && d44.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.FreeDesc(&d43)
			d45 := d44
			if d45.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d45)
			ctx.EmitStoreToStack(d45, 24)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(48))
			ctx.W.EmitAddRSP32(int32(48))
			return result
		}, /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */
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
		nil /* TODO: Slice on non-desc: slice a[1:int:] */, /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[1]
			d1 := args[0]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d2.Imm.Bool())}
			} else {
				ctx.EnsureDesc(&d2)
				negReg := ctx.AllocReg()
				if d2.Loc == LocRegPair {
					ctx.W.EmitCmpRegImm32(d2.Reg2, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.W.EmitCmpRegImm32(d2.Reg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! on non-register descriptor")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[1]
			d1 := args[0]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d2.Imm.Bool())}
			} else {
				ctx.EnsureDesc(&d2)
				negReg := ctx.AllocReg()
				if d2.Loc == LocRegPair {
					ctx.W.EmitCmpRegImm32(d2.Reg2, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.W.EmitCmpRegImm32(d2.Reg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! on non-register descriptor")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(EqualSQL), []JITValueDesc{d0, d1}, 2)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d2, &result)
			} else {
				switch d2.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d2)
				case tagInt:
					ctx.W.EmitMakeInt(result, d2)
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d2)
				case tagNil:
					ctx.W.EmitMakeNil(result)
				default:
					panic("jit: single-block scalar return with unknown type")
				}
			}
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		nil /* TODO: unsupported compare const kind: (Scmer).String(t22) */, /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
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
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Bool())}
			} else if d0.Type == tagBool && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else if d0.Type == tagBool && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{d0}, 1)
				d1.Type = tagBool
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				ctx.EnsureDesc(&d1)
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.W.EmitCmpRegImm32(d1.Reg2, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d2)
				} else if d1.Loc == LocReg {
					ctx.W.EmitCmpRegImm32(d1.Reg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d2)
				} else {
					panic("UnOp ! on non-register descriptor")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d0.Imm.Bool())}
			} else if d0.Type == tagBool && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else if d0.Type == tagBool && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Bool), []JITValueDesc{d0}, 1)
				d1.Type = tagBool
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				ctx.EnsureDesc(&d1)
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.W.EmitCmpRegImm32(d1.Reg2, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d2)
				} else if d1.Loc == LocReg {
					ctx.W.EmitCmpRegImm32(d1.Reg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d2)
				} else {
					panic("UnOp ! on non-register descriptor")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
			ctx.EnsureDesc(&d1)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d1.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d1)
			} else {
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl1 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d2.Reg, int32(1))
				d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
				ctx.BindReg(d2.Reg, &d3)
			}
			if d3.Loc == LocReg && d2.Loc == LocReg && d3.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d0)
			var d4 JITValueDesc
			if d3.Loc == LocImm && d0.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d0.Imm.Int())}
			} else if d0.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d3.Reg)
				if d0.Imm.Int() >= -2147483648 && d0.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(d0.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.W.EmitCmpInt64(d3.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d4)
			} else if d3.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d0.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d4)
			} else {
				r3 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitCmpInt64(d3.Reg, d0.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d4)
			}
			ctx.FreeDesc(&d0)
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d4)
			ctx.W.MarkLabel(lbl3)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d1, &result)
				result.Type = d1.Type
			} else {
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d1)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d1)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d1)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d1, &result)
					result.Type = d1.Type
				}
			}
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl2)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d3)
			var d5 JITValueDesc
			if d3.Loc == LocImm {
				idx := int(d3.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d5 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d3.Reg] {
					ctx.ProtectReg(d3.Reg)
					seen[d3.Reg] = true
					protected = append(protected, d3.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl5 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl5)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl5)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d5 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d5)
				ctx.BindReg(r5, &d5)
			}
			d7 := d1
			d6 := ctx.EmitTagEquals(&d7, tagNil, JITValueDesc{Loc: LocAny})
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d6.Loc == LocImm {
				if d6.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d6.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl8)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl7)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d9 := d5
			d8 := ctx.EmitTagEquals(&d9, tagNil, JITValueDesc{Loc: LocAny})
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d8.Loc == LocImm {
				if d8.Imm.Bool() {
			d10 := d1
			if d10.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocRegPair || d10.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d10, 0)
			} else {
				ctx.EmitStoreToStack(d10, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d11 := d3
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d11)
			ctx.EmitStoreToStack(d11, 16)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d8.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl10)
			d12 := d1
			if d12.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			if d12.Loc == LocRegPair || d12.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d12, 0)
			} else {
				ctx.EmitStoreToStack(d12, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d13 := d3
			if d13.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, 16)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d8)
			ctx.W.MarkLabel(lbl6)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d14 := d5
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			if d14.Loc == LocRegPair || d14.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d14, 0)
			} else {
				ctx.EmitStoreToStack(d14, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d15 := d3
			if d15.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl9)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d16 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d5, d1}, 1)
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d16.Loc == LocImm {
				if d16.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
			d17 := d1
			if d17.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			if d17.Loc == LocRegPair || d17.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d17, 0)
			} else {
				ctx.EmitStoreToStack(d17, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d18 := d3
			if d18.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d18)
			ctx.EmitStoreToStack(d18, 16)
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl12)
			d19 := d1
			if d19.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d19)
			if d19.Loc == LocRegPair || d19.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d19, 0)
			} else {
				ctx.EmitStoreToStack(d19, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d20 := d3
			if d20.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, 16)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl11)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d21 := d5
			if d21.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d21)
			if d21.Loc == LocRegPair || d21.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d21, 0)
			} else {
				ctx.EmitStoreToStack(d21, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d22 := d3
			if d22.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d22)
			ctx.EmitStoreToStack(d22, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(24))
			ctx.W.EmitAddRSP32(int32(24))
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl1 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d2.Reg, int32(1))
				d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d2.Reg}
				ctx.BindReg(d2.Reg, &d3)
			}
			if d3.Loc == LocReg && d2.Loc == LocReg && d3.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d0)
			var d4 JITValueDesc
			if d3.Loc == LocImm && d0.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d0.Imm.Int())}
			} else if d0.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d3.Reg)
				if d0.Imm.Int() >= -2147483648 && d0.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(d0.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d0.Imm.Int()))
					ctx.W.EmitCmpInt64(d3.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d4)
			} else if d3.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d0.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d4)
			} else {
				r3 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitCmpInt64(d3.Reg, d0.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d4)
			}
			ctx.FreeDesc(&d0)
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d4)
			ctx.W.MarkLabel(lbl3)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d1, &result)
				result.Type = d1.Type
			} else {
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d1)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d1)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d1)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d1, &result)
					result.Type = d1.Type
				}
			}
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl2)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d3)
			var d5 JITValueDesc
			if d3.Loc == LocImm {
				idx := int(d3.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d5 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d3.Reg] {
					ctx.ProtectReg(d3.Reg)
					seen[d3.Reg] = true
					protected = append(protected, d3.Reg)
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
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl5 := ctx.W.ReserveLabel()
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl5)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl5)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d5 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d5)
				ctx.BindReg(r5, &d5)
			}
			d7 := d1
			d6 := ctx.EmitTagEquals(&d7, tagNil, JITValueDesc{Loc: LocAny})
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d6.Loc == LocImm {
				if d6.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d6.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl8)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl7)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d9 := d5
			d8 := ctx.EmitTagEquals(&d9, tagNil, JITValueDesc{Loc: LocAny})
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d8.Loc == LocImm {
				if d8.Imm.Bool() {
			d10 := d1
			if d10.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocRegPair || d10.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d10, 0)
			} else {
				ctx.EmitStoreToStack(d10, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d11 := d3
			if d11.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d11)
			ctx.EmitStoreToStack(d11, 16)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d8.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl10)
			d12 := d1
			if d12.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			if d12.Loc == LocRegPair || d12.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d12, 0)
			} else {
				ctx.EmitStoreToStack(d12, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d13 := d3
			if d13.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, 16)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d8)
			ctx.W.MarkLabel(lbl6)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d14 := d5
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			if d14.Loc == LocRegPair || d14.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d14, 0)
			} else {
				ctx.EmitStoreToStack(d14, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d15 := d3
			if d15.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl9)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocRegPair && d5.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d16 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d1, d5}, 1)
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d16.Loc == LocImm {
				if d16.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
			d17 := d1
			if d17.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			if d17.Loc == LocRegPair || d17.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d17, 0)
			} else {
				ctx.EmitStoreToStack(d17, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d18 := d3
			if d18.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d18)
			ctx.EmitStoreToStack(d18, 16)
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl12)
			d19 := d1
			if d19.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d19)
			if d19.Loc == LocRegPair || d19.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d19, 0)
			} else {
				ctx.EmitStoreToStack(d19, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d20 := d3
			if d20.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, 16)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl11)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d21 := d5
			if d21.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d21)
			if d21.Loc == LocRegPair || d21.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d21, 0)
			} else {
				ctx.EmitStoreToStack(d21, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d22 := d3
			if d22.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d22)
			ctx.EmitStoreToStack(d22, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(24))
			ctx.W.EmitAddRSP32(int32(24))
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d0.Imm.Float())}
			} else if d0.Type == tagFloat && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else if d0.Type == tagFloat && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d0}, 1)
				d1.Type = tagFloat
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair || d1.Loc == LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(math.Floor), []JITValueDesc{d1}, 1)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d0.Imm.Float())}
			} else if d0.Type == tagFloat && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else if d0.Type == tagFloat && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d0}, 1)
				d1.Type = tagFloat
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair || d1.Loc == LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(math.Ceil), []JITValueDesc{d1}, 1)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			d0 := args[0]
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d0.Imm.Float())}
			} else if d0.Type == tagFloat && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else if d0.Type == tagFloat && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d0}, 1)
				d1.Type = tagFloat
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair || d1.Loc == LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(math.Round), []JITValueDesc{d1}, 1)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			return result
		}, /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d3 := args[0]
			var d4 JITValueDesc
			if d3.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float())}
			} else if d3.Type == tagFloat && d3.Loc == LocReg {
				d4 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d3.Reg}
				ctx.BindReg(d3.Reg, &d4)
				ctx.BindReg(d3.Reg, &d4)
			} else if d3.Type == tagFloat && d3.Loc == LocRegPair {
				ctx.FreeReg(d3.Reg)
				d4 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d3.Reg2}
				ctx.BindReg(d3.Reg2, &d4)
				ctx.BindReg(d3.Reg2, &d4)
			} else {
				d4 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d3}, 1)
				d4.Type = tagFloat
				ctx.BindReg(d4.Reg, &d4)
			}
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d4)
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Float() < 0)}
			} else {
				r1 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegImm64(RegR11, uint64(0))
				ctx.W.EmitCmpFloat64Setcc(r1, d4.Reg, RegR11, CcL)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d5)
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
			d6 := d4
			if d6.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d6)
			ctx.EmitStoreToStack(d6, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl6)
			d7 := d4
			if d7.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d7)
			ctx.EmitStoreToStack(d7, 0)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			d8 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d9 := args[0]
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d9)
			if d9.Loc != LocRegPair && d9.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value")
			}
			d10 := ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d9}, 1)
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d8)
			var d11 JITValueDesc
			if d8.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d8.Imm.Float()))}
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r2, d8.Reg)
				d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
				ctx.BindReg(r2, &d11)
			}
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d11)
			var d12 JITValueDesc
			if d10.Loc == LocImm && d11.Loc == LocImm {
				d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() == d11.Imm.Int())}
			} else if d11.Loc == LocImm {
				r3 := ctx.AllocReg()
				if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d10.Reg, int32(d11.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d11.Imm.Int()))
					ctx.W.EmitCmpInt64(d10.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r3, CcE)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d12)
			} else if d10.Loc == LocImm {
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d11.Reg)
				ctx.W.EmitSetcc(r4, CcE)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d12)
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d10.Reg, d11.Reg)
				ctx.W.EmitSetcc(r5, CcE)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d12)
			}
			ctx.FreeDesc(&d10)
			ctx.FreeDesc(&d11)
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d12.Loc == LocImm {
				if d12.Imm.Bool() {
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d12.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d12)
			ctx.W.MarkLabel(lbl4)
			d8 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d13 JITValueDesc
			if d4.Loc == LocImm {
				if d4.Type == tagFloat {
					d13 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d4.Imm.Float())}
				} else {
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d4.Imm.Int())}
				}
			} else {
				if d4.Type == tagFloat {
					r6 := ctx.AllocRegExcept(d4.Reg)
					ctx.W.EmitMovRegImm64(r6, 0)
					ctx.W.EmitSubFloat64(r6, d4.Reg)
					d13 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
					ctx.BindReg(r6, &d13)
				} else {
					r7 := ctx.AllocRegExcept(d4.Reg)
					ctx.W.EmitMovRegImm64(r7, 0)
					ctx.W.EmitSubInt64(r7, d4.Reg)
					d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
					ctx.BindReg(r7, &d13)
				}
			}
			d14 := d13
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl8)
			d8 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d8)
			ctx.W.EmitMakeFloat(result, d8)
			if d8.Loc == LocReg { ctx.FreeReg(d8.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			d8 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d15 := args[0]
			var d16 JITValueDesc
			if d15.Loc == LocImm {
				d16 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d15.Imm.Float())}
			} else if d15.Type == tagFloat && d15.Loc == LocReg {
				d16 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d15.Reg}
				ctx.BindReg(d15.Reg, &d16)
				ctx.BindReg(d15.Reg, &d16)
			} else if d15.Type == tagFloat && d15.Loc == LocRegPair {
				ctx.FreeReg(d15.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d15.Reg2}
				ctx.BindReg(d15.Reg2, &d16)
				ctx.BindReg(d15.Reg2, &d16)
			} else {
				d16 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d15}, 1)
				d16.Type = tagFloat
				ctx.BindReg(d16.Reg, &d16)
			}
			ctx.FreeDesc(&d15)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d8)
			var d17 JITValueDesc
			if d16.Loc == LocImm && d8.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Float() == d8.Imm.Float())}
			} else if d8.Loc == LocImm {
				r8 := ctx.AllocReg()
				_, yBits := d8.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitCmpFloat64Setcc(r8, d16.Reg, RegR11, CcE)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d17)
			} else if d16.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d8.Reg)
				_, xBits := d16.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, xBits)
				ctx.W.EmitCmpFloat64Setcc(r9, RegR11, d8.Reg, CcE)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d17)
			} else {
				r10 := ctx.AllocRegExcept(d16.Reg, d8.Reg)
				ctx.W.EmitCmpFloat64Setcc(r10, d16.Reg, d8.Reg, CcE)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d17)
			}
			ctx.FreeDesc(&d16)
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d17.Loc == LocImm {
				if d17.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d17.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl11)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d17)
			ctx.W.MarkLabel(lbl10)
			d8 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d8)
			var d18 JITValueDesc
			if d8.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d8.Imm.Float()))}
			} else {
				r11 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r11, d8.Reg)
				d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d18)
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			ctx.W.EmitMakeInt(result, d18)
			if d18.Loc == LocReg { ctx.FreeReg(d18.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(8))
			ctx.W.EmitAddRSP32(int32(8))
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			d0 := args[0]
			d2 := d0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d1.Loc == LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl2)
			d3 := args[0]
			var d4 JITValueDesc
			if d3.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float())}
			} else if d3.Type == tagFloat && d3.Loc == LocReg {
				d4 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d3.Reg}
				ctx.BindReg(d3.Reg, &d4)
				ctx.BindReg(d3.Reg, &d4)
			} else if d3.Type == tagFloat && d3.Loc == LocRegPair {
				ctx.FreeReg(d3.Reg)
				d4 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d3.Reg2}
				ctx.BindReg(d3.Reg2, &d4)
				ctx.BindReg(d3.Reg2, &d4)
			} else {
				d4 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d3}, 1)
				d4.Type = tagFloat
				ctx.BindReg(d4.Reg, &d4)
			}
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d4)
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Float() < 0)}
			} else {
				r0 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegImm64(RegR11, uint64(0))
				ctx.W.EmitCmpFloat64Setcc(r0, d4.Reg, RegR11, CcL)
				d5 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d5)
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl6)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			ctx.EnsureDesc(&d4)
			d6 := ctx.EmitGoCallScalar(GoFuncAddr(math.Sqrt), []JITValueDesc{d4}, 1)
			d6.Type = tagFloat
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			ctx.W.EmitMakeFloat(result, d6)
			if d6.Loc == LocReg { ctx.FreeReg(d6.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl4)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
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
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
}
