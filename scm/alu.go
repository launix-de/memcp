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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
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
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(3))}
			} else {
				r1 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpRegImm32(d1.Reg, 3)
				ctx.W.EmitSetcc(r1, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d2)
			}
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
			}
			ctx.FreeDesc(&d2)
			ctx.W.MarkLabel(lbl3)
			ctx.EnsureDesc(&d1)
			var d4 JITValueDesc
			if d1.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(4))}
			} else {
				r2 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r2, CcE)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d4)
			}
			d5 := d4
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocImm && d5.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.MarkLabel(lbl7)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.MarkLabel(lbl8)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl7)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl7)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d4)
			ctx.W.MarkLabel(lbl2)
			d6 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			ctx.W.EmitMakeBool(result, d6)
			if d6.Loc == LocReg { ctx.FreeReg(d6.Reg) }
			result.Type = tagBool
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl6)
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d1)
			var d7 JITValueDesc
			if d1.Loc == LocImm {
				d7 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(15))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 15)
				ctx.W.EmitSetcc(r3, CcE)
				d7 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d7)
			}
			ctx.FreeDesc(&d1)
			d8 := d7
			if d8.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d8)
			ctx.EmitStoreToStack(d8, 0)
			ctx.W.EmitJmp(lbl2)
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
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			lbl2 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 8)
			ctx.W.MarkLabel(lbl2)
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
			d4 := d3
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocImm && d4.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl5)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl4)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d5 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d5)
			var d6 JITValueDesc
			if d1.Loc == LocImm && d5.Loc == LocImm {
				d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == d5.Imm.Int())}
			} else if d5.Loc == LocImm {
				r4 := ctx.AllocRegExcept(d1.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r4, CcE)
				d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d6)
			} else if d1.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d5.Reg)
				ctx.W.EmitSetcc(r5, CcE)
				d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d6)
			} else {
				r6 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d5.Reg)
				ctx.W.EmitSetcc(r6, CcE)
				d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d6)
			}
			ctx.FreeDesc(&d5)
			d7 := d6
			ctx.EnsureDesc(&d7)
			if d7.Loc != LocImm && d7.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d7.Loc == LocImm {
				if d7.Imm.Bool() {
					ctx.W.MarkLabel(lbl9)
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.MarkLabel(lbl10)
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl3)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d1)
			var d8 JITValueDesc
			if d1.Loc == LocImm {
				idx := int(d1.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d8 = args[idx]
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
				lbl11 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r7, ai.Reg)
						ctx.W.EmitMovRegReg(r8, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r7, tmp.Reg)
						ctx.W.EmitMovRegReg(r8, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl11)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl11)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d8 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r7, Reg2: r8}
				ctx.BindReg(r7, &d8)
				ctx.BindReg(r8, &d8)
			}
			d10 := d8
			d10.ID = 0
			d9 := ctx.EmitTagEquals(&d10, tagInt, JITValueDesc{Loc: LocAny})
			d11 := d9
			ctx.EnsureDesc(&d11)
			if d11.Loc != LocImm && d11.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d11.Loc == LocImm {
				if d11.Imm.Bool() {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d11.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d9)
			ctx.W.MarkLabel(lbl8)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d12 JITValueDesc
			if d0.Loc == LocImm {
				d12 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d0.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d0.Reg)
				d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d12)
			}
			lbl15 := ctx.W.ReserveLabel()
			d13 := d1
			if d13.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, 16)
			d14 := d12
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 24)
			ctx.W.MarkLabel(lbl15)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d15 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d17 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d17)
			var d18 JITValueDesc
			if d15.Loc == LocImm && d17.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d17.Imm.Int())}
			} else if d17.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d15.Reg)
				if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d15.Reg, int32(d17.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
					ctx.W.EmitCmpInt64(d15.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcL)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d18)
			} else if d15.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d17.Reg)
				ctx.W.EmitSetcc(r10, CcL)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d18)
			} else {
				r11 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitCmpInt64(d15.Reg, d17.Reg)
				ctx.W.EmitSetcc(r11, CcL)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d18)
			}
			ctx.FreeDesc(&d17)
			d19 := d18
			ctx.EnsureDesc(&d19)
			if d19.Loc != LocImm && d19.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d19.Loc == LocImm {
				if d19.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d19.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d18)
			ctx.W.MarkLabel(lbl7)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.W.EmitMakeInt(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			var d20 JITValueDesc
			if d8.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int())}
			} else if d8.Type == tagInt && d8.Loc == LocRegPair {
				ctx.FreeReg(d8.Reg)
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg2}
				ctx.BindReg(d8.Reg2, &d20)
				ctx.BindReg(d8.Reg2, &d20)
			} else if d8.Type == tagInt && d8.Loc == LocReg {
				d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d8.Reg}
				ctx.BindReg(d8.Reg, &d20)
				ctx.BindReg(d8.Reg, &d20)
			} else {
				d20 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d8}, 1)
				d20.Type = tagInt
				ctx.BindReg(d20.Reg, &d20)
			}
			ctx.FreeDesc(&d8)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d20)
			var d21 JITValueDesc
			if d0.Loc == LocImm && d20.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d20.Imm.Int())}
			} else if d20.Loc == LocImm && d20.Imm.Int() == 0 {
				r12 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r12, d0.Reg)
				d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d21)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d20.Reg}
				ctx.BindReg(d20.Reg, &d21)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d20.Reg)
				d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d20.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r13 := ctx.AllocRegExcept(d0.Reg, d20.Reg)
				ctx.W.EmitMovRegReg(r13, d0.Reg)
				ctx.W.EmitAddInt64(r13, d20.Reg)
				d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
				ctx.BindReg(r13, &d21)
			}
			if d21.Loc == LocReg && d0.Loc == LocReg && d21.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d20)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d22 JITValueDesc
			if d1.Loc == LocImm {
				d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			}
			if d22.Loc == LocReg && d1.Loc == LocReg && d22.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			d23 := d21
			if d23.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			ctx.EmitStoreToStack(d23, 0)
			d24 := d22
			if d24.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d24)
			ctx.EmitStoreToStack(d24, 8)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl17)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d16)
			ctx.W.EmitMakeFloat(result, d16)
			if d16.Loc == LocReg { ctx.FreeReg(d16.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl16)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d15)
			var d25 JITValueDesc
			if d15.Loc == LocImm {
				idx := int(d15.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d25 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d15.Reg] {
					ctx.ProtectReg(d15.Reg)
					seen[d15.Reg] = true
					protected = append(protected, d15.Reg)
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
				lbl20 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d15.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r14, ai.Reg)
						ctx.W.EmitMovRegReg(r15, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r14, tmp.Reg)
						ctx.W.EmitMovRegReg(r15, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl20)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl20)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d25 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r14, Reg2: r15}
				ctx.BindReg(r14, &d25)
				ctx.BindReg(r15, &d25)
			}
			d27 := d25
			d27.ID = 0
			d26 := ctx.EmitTagEquals(&d27, tagNil, JITValueDesc{Loc: LocAny})
			d28 := d26
			ctx.EnsureDesc(&d28)
			if d28.Loc != LocImm && d28.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			if d28.Loc == LocImm {
				if d28.Imm.Bool() {
					ctx.W.MarkLabel(lbl23)
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.MarkLabel(lbl24)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d28.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl23)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl24)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d26)
			ctx.W.MarkLabel(lbl22)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			var d29 JITValueDesc
			if d25.Loc == LocImm {
				d29 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d25.Imm.Float())}
			} else if d25.Type == tagFloat && d25.Loc == LocReg {
				d29 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d25.Reg}
				ctx.BindReg(d25.Reg, &d29)
				ctx.BindReg(d25.Reg, &d29)
			} else if d25.Type == tagFloat && d25.Loc == LocRegPair {
				ctx.FreeReg(d25.Reg)
				d29 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d25.Reg2}
				ctx.BindReg(d25.Reg2, &d29)
				ctx.BindReg(d25.Reg2, &d29)
			} else {
				d29 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d25}, 1)
				d29.Type = tagFloat
				ctx.BindReg(d29.Reg, &d29)
			}
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d29)
			var d30 JITValueDesc
			if d16.Loc == LocImm && d29.Loc == LocImm {
				d30 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d16.Imm.Float() + d29.Imm.Float())}
			} else if d16.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				_, xBits := d16.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitAddFloat64(scratch, d29.Reg)
				d30 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else if d29.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				_, yBits := d29.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitAddFloat64(scratch, RegR11)
				d30 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else {
				r16 := ctx.AllocRegExcept(d16.Reg, d29.Reg)
				ctx.W.EmitMovRegReg(r16, d16.Reg)
				ctx.W.EmitAddFloat64(r16, d29.Reg)
				d30 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
				ctx.BindReg(r16, &d30)
			}
			if d30.Loc == LocReg && d16.Loc == LocReg && d30.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			var d31 JITValueDesc
			if d15.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(scratch, d15.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			}
			if d31.Loc == LocReg && d15.Loc == LocReg && d31.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = LocNone
			}
			d32 := d31
			if d32.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d32)
			ctx.EmitStoreToStack(d32, 16)
			d33 := d30
			if d33.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d33)
			ctx.EmitStoreToStack(d33, 24)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl21)
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d15 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
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
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl2 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 0)
			ctx.W.MarkLabel(lbl2)
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
			d4 := d3
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocImm && d4.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl5)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl4)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 := args[0]
			d7 := d5
			d7.ID = 0
			d6 := ctx.EmitTagEquals(&d7, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d5)
			d8 := d6
			ctx.EnsureDesc(&d8)
			if d8.Loc != LocImm && d8.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d8.Loc == LocImm {
				if d8.Imm.Bool() {
					ctx.W.MarkLabel(lbl9)
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.MarkLabel(lbl10)
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d8.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl9)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl3)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d2)
			var d9 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d9 = args[idx]
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
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl11 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl11)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl11)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d9 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d9)
				ctx.BindReg(r5, &d9)
			}
			d11 := d9
			d11.ID = 0
			d10 := ctx.EmitTagEquals(&d11, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d9)
			d12 := d10
			ctx.EnsureDesc(&d12)
			if d12.Loc != LocImm && d12.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d12.Loc == LocImm {
				if d12.Imm.Bool() {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.MarkLabel(lbl14)
			d13 := d2
			if d13.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, 0)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d12.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl14)
			d14 := d2
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 0)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl8)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
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
			lbl15 := ctx.W.ReserveLabel()
			d17 := d16
			if d17.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			ctx.EmitStoreToStack(d17, 40)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 48)
			ctx.W.MarkLabel(lbl15)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d18 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d20 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			var d21 JITValueDesc
			if d19.Loc == LocImm && d20.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.Int() < d20.Imm.Int())}
			} else if d20.Loc == LocImm {
				r6 := ctx.AllocRegExcept(d19.Reg)
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d19.Reg, int32(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
					ctx.W.EmitCmpInt64(d19.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r6, CcL)
				d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d21)
			} else if d19.Loc == LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d20.Reg)
				ctx.W.EmitSetcc(r7, CcL)
				d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d21)
			} else {
				r8 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitCmpInt64(d19.Reg, d20.Reg)
				ctx.W.EmitSetcc(r8, CcL)
				d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d21)
			}
			ctx.FreeDesc(&d20)
			d22 := d21
			ctx.EnsureDesc(&d22)
			if d22.Loc != LocImm && d22.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d22.Loc == LocImm {
				if d22.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d22.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d21)
			ctx.W.MarkLabel(lbl7)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d23 := args[0]
			var d24 JITValueDesc
			if d23.Loc == LocImm {
				d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d23.Imm.Int())}
			} else if d23.Type == tagInt && d23.Loc == LocRegPair {
				ctx.FreeReg(d23.Reg)
				d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d23.Reg2}
				ctx.BindReg(d23.Reg2, &d24)
				ctx.BindReg(d23.Reg2, &d24)
			} else if d23.Type == tagInt && d23.Loc == LocReg {
				d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d23.Reg}
				ctx.BindReg(d23.Reg, &d24)
				ctx.BindReg(d23.Reg, &d24)
			} else {
				d24 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d23}, 1)
				d24.Type = tagInt
				ctx.BindReg(d24.Reg, &d24)
			}
			ctx.FreeDesc(&d23)
			lbl20 := ctx.W.ReserveLabel()
			d25 := d24
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			ctx.EmitStoreToStack(d25, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 16)
			ctx.W.MarkLabel(lbl20)
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d26 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d28 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d28)
			var d29 JITValueDesc
			if d27.Loc == LocImm && d28.Loc == LocImm {
				d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d27.Imm.Int() < d28.Imm.Int())}
			} else if d28.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d27.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d27.Reg, int32(d28.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d28.Imm.Int()))
					ctx.W.EmitCmpInt64(d27.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcL)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d29)
			} else if d27.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d27.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d28.Reg)
				ctx.W.EmitSetcc(r10, CcL)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d29)
			} else {
				r11 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitCmpInt64(d27.Reg, d28.Reg)
				ctx.W.EmitSetcc(r11, CcL)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d29)
			}
			ctx.FreeDesc(&d28)
			d30 := d29
			ctx.EnsureDesc(&d30)
			if d30.Loc != LocImm && d30.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			if d30.Loc == LocImm {
				if d30.Imm.Bool() {
					ctx.W.MarkLabel(lbl23)
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.MarkLabel(lbl24)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl23)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl24)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d29)
			ctx.W.MarkLabel(lbl12)
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl17)
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			ctx.W.EmitMakeFloat(result, d18)
			if d18.Loc == LocReg { ctx.FreeReg(d18.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl16)
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d19)
			var d31 JITValueDesc
			if d19.Loc == LocImm {
				idx := int(d19.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d31 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d19.Reg] {
					ctx.ProtectReg(d19.Reg)
					seen[d19.Reg] = true
					protected = append(protected, d19.Reg)
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
				lbl25 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d19.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r12, ai.Reg)
						ctx.W.EmitMovRegReg(r13, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r12, tmp.Reg)
						ctx.W.EmitMovRegReg(r13, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl25)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl25)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d31 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r12, Reg2: r13}
				ctx.BindReg(r12, &d31)
				ctx.BindReg(r13, &d31)
			}
			var d32 JITValueDesc
			if d31.Loc == LocImm {
				d32 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d31.Imm.Float())}
			} else if d31.Type == tagFloat && d31.Loc == LocReg {
				d32 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d31.Reg}
				ctx.BindReg(d31.Reg, &d32)
				ctx.BindReg(d31.Reg, &d32)
			} else if d31.Type == tagFloat && d31.Loc == LocRegPair {
				ctx.FreeReg(d31.Reg)
				d32 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d31.Reg2}
				ctx.BindReg(d31.Reg2, &d32)
				ctx.BindReg(d31.Reg2, &d32)
			} else {
				d32 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d31}, 1)
				d32.Type = tagFloat
				ctx.BindReg(d32.Reg, &d32)
			}
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d32)
			var d33 JITValueDesc
			if d18.Loc == LocImm && d32.Loc == LocImm {
				d33 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d18.Imm.Float() - d32.Imm.Float())}
			} else if d18.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				_, xBits := d18.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d32.Reg)
				d33 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d32.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				_, yBits := d32.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d33 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r14 := ctx.AllocRegExcept(d18.Reg, d32.Reg)
				ctx.W.EmitMovRegReg(r14, d18.Reg)
				ctx.W.EmitSubFloat64(r14, d32.Reg)
				d33 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r14}
				ctx.BindReg(r14, &d33)
			}
			if d33.Loc == LocReg && d18.Loc == LocReg && d33.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			var d34 JITValueDesc
			if d19.Loc == LocImm {
				d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(scratch, d19.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d34)
			}
			if d34.Loc == LocReg && d19.Loc == LocReg && d34.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = LocNone
			}
			d35 := d33
			if d35.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d35)
			ctx.EmitStoreToStack(d35, 40)
			d36 := d34
			if d36.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, 48)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl22)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d37 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d37)
			var d38 JITValueDesc
			if d27.Loc == LocImm && d37.Loc == LocImm {
				d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d27.Imm.Int() == d37.Imm.Int())}
			} else if d37.Loc == LocImm {
				r15 := ctx.AllocRegExcept(d27.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d27.Reg, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitCmpInt64(d27.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r15, CcE)
				d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d38)
			} else if d27.Loc == LocImm {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d27.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d37.Reg)
				ctx.W.EmitSetcc(r16, CcE)
				d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d38)
			} else {
				r17 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitCmpInt64(d27.Reg, d37.Reg)
				ctx.W.EmitSetcc(r17, CcE)
				d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d38)
			}
			ctx.FreeDesc(&d37)
			d39 := d38
			ctx.EnsureDesc(&d39)
			if d39.Loc != LocImm && d39.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d39.Loc == LocImm {
				if d39.Imm.Bool() {
					ctx.W.MarkLabel(lbl28)
					ctx.W.EmitJmp(lbl26)
				} else {
					ctx.W.MarkLabel(lbl29)
					ctx.W.EmitJmp(lbl27)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d39.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl28)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d38)
			ctx.W.MarkLabel(lbl21)
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d27)
			var d40 JITValueDesc
			if d27.Loc == LocImm {
				idx := int(d27.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d40 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d27.Reg] {
					ctx.ProtectReg(d27.Reg)
					seen[d27.Reg] = true
					protected = append(protected, d27.Reg)
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
				lbl30 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d27.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r18, ai.Reg)
						ctx.W.EmitMovRegReg(r19, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r18, tmp.Reg)
						ctx.W.EmitMovRegReg(r19, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
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
				d40 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d40)
				ctx.BindReg(r19, &d40)
			}
			d42 := d40
			d42.ID = 0
			d41 := ctx.EmitTagEquals(&d42, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d40)
			d43 := d41
			ctx.EnsureDesc(&d43)
			if d43.Loc != LocImm && d43.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d43.Loc == LocImm {
				if d43.Imm.Bool() {
					ctx.W.MarkLabel(lbl32)
					ctx.W.EmitJmp(lbl31)
				} else {
					ctx.W.MarkLabel(lbl33)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl32)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl33)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d41)
			ctx.W.MarkLabel(lbl27)
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d26)
			var d44 JITValueDesc
			if d26.Loc == LocImm {
				d44 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d26.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d26.Reg)
				d44 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d26.Reg}
				ctx.BindReg(d26.Reg, &d44)
			}
			lbl34 := ctx.W.ReserveLabel()
			d45 := d27
			if d45.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d45)
			ctx.EmitStoreToStack(d45, 24)
			d46 := d44
			if d46.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d46)
			ctx.EmitStoreToStack(d46, 32)
			ctx.W.MarkLabel(lbl34)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d47 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d48 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d49 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			var d50 JITValueDesc
			if d47.Loc == LocImm && d49.Loc == LocImm {
				d50 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d47.Imm.Int() < d49.Imm.Int())}
			} else if d49.Loc == LocImm {
				r20 := ctx.AllocRegExcept(d47.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d47.Reg, int32(d49.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d49.Imm.Int()))
					ctx.W.EmitCmpInt64(d47.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r20, CcL)
				d50 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
				ctx.BindReg(r20, &d50)
			} else if d47.Loc == LocImm {
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d47.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d49.Reg)
				ctx.W.EmitSetcc(r21, CcL)
				d50 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
				ctx.BindReg(r21, &d50)
			} else {
				r22 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitCmpInt64(d47.Reg, d49.Reg)
				ctx.W.EmitSetcc(r22, CcL)
				d50 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d50)
			}
			ctx.FreeDesc(&d49)
			d51 := d50
			ctx.EnsureDesc(&d51)
			if d51.Loc != LocImm && d51.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			if d51.Loc == LocImm {
				if d51.Imm.Bool() {
					ctx.W.MarkLabel(lbl37)
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.MarkLabel(lbl38)
					ctx.W.EmitJmp(lbl36)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d51.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl37)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl37)
				ctx.W.EmitJmp(lbl35)
				ctx.W.MarkLabel(lbl38)
				ctx.W.EmitJmp(lbl36)
			}
			ctx.FreeDesc(&d50)
			ctx.W.MarkLabel(lbl26)
			d48 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d47 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d26)
			ctx.W.EmitMakeInt(result, d26)
			if d26.Loc == LocReg { ctx.FreeReg(d26.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl31)
			d47 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d48 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d27)
			var d52 JITValueDesc
			if d27.Loc == LocImm {
				idx := int(d27.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d52 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d27.Reg] {
					ctx.ProtectReg(d27.Reg)
					seen[d27.Reg] = true
					protected = append(protected, d27.Reg)
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
				r23 := ctx.AllocReg()
				r24 := ctx.AllocRegExcept(r23)
				lbl39 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d27.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r23, ai.Reg)
						ctx.W.EmitMovRegReg(r24, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r23, tmp.Reg)
						ctx.W.EmitMovRegReg(r24, tmp.Reg2)
						typ = tmp.Type
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r23, Reg2: r24}
						ctx.BindReg(r23, &pair)
						ctx.BindReg(r24, &pair)
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
							ctx.W.EmitMovRegImm64(r23, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r24, auxWord)
						}
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl39)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl39)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d52 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r23, Reg2: r24}
				ctx.BindReg(r23, &d52)
				ctx.BindReg(r24, &d52)
			}
			var d53 JITValueDesc
			if d52.Loc == LocImm {
				d53 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d52.Imm.Int())}
			} else if d52.Type == tagInt && d52.Loc == LocRegPair {
				ctx.FreeReg(d52.Reg)
				d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d52.Reg2}
				ctx.BindReg(d52.Reg2, &d53)
				ctx.BindReg(d52.Reg2, &d53)
			} else if d52.Type == tagInt && d52.Loc == LocReg {
				d53 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d52.Reg}
				ctx.BindReg(d52.Reg, &d53)
				ctx.BindReg(d52.Reg, &d53)
			} else {
				d53 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d52}, 1)
				d53.Type = tagInt
				ctx.BindReg(d53.Reg, &d53)
			}
			ctx.FreeDesc(&d52)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d53)
			var d54 JITValueDesc
			if d26.Loc == LocImm && d53.Loc == LocImm {
				d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d26.Imm.Int() - d53.Imm.Int())}
			} else if d53.Loc == LocImm && d53.Imm.Int() == 0 {
				r25 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r25, d26.Reg)
				d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
				ctx.BindReg(r25, &d54)
			} else if d26.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d53.Reg)
				d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else if d53.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(scratch, d26.Reg)
				if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d53.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d53.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else {
				r26 := ctx.AllocRegExcept(d26.Reg, d53.Reg)
				ctx.W.EmitMovRegReg(r26, d26.Reg)
				ctx.W.EmitSubInt64(r26, d53.Reg)
				d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r26}
				ctx.BindReg(r26, &d54)
			}
			if d54.Loc == LocReg && d26.Loc == LocReg && d54.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = LocNone
			}
			ctx.FreeDesc(&d53)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d27)
			var d55 JITValueDesc
			if d27.Loc == LocImm {
				d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d27.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(scratch, d27.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d55 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			}
			if d55.Loc == LocReg && d27.Loc == LocReg && d55.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = LocNone
			}
			d56 := d54
			if d56.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d56)
			ctx.EmitStoreToStack(d56, 8)
			d57 := d55
			if d57.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			ctx.EmitStoreToStack(d57, 16)
			ctx.W.EmitJmp(lbl20)
			ctx.W.MarkLabel(lbl36)
			d47 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d48 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d48)
			ctx.W.EmitMakeFloat(result, d48)
			if d48.Loc == LocReg { ctx.FreeReg(d48.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl35)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d47 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d48 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d18 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d19 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d47)
			var d58 JITValueDesc
			if d47.Loc == LocImm {
				idx := int(d47.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d58 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d47.Reg] {
					ctx.ProtectReg(d47.Reg)
					seen[d47.Reg] = true
					protected = append(protected, d47.Reg)
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
				r27 := ctx.AllocReg()
				r28 := ctx.AllocRegExcept(r27)
				lbl40 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d47.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r27, ai.Reg)
						ctx.W.EmitMovRegReg(r28, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r27, tmp.Reg)
						ctx.W.EmitMovRegReg(r28, tmp.Reg2)
						typ = tmp.Type
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r27, Reg2: r28}
						ctx.BindReg(r27, &pair)
						ctx.BindReg(r28, &pair)
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
							ctx.W.EmitMovRegImm64(r27, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r28, auxWord)
						}
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl40)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl40)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d58 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r27, Reg2: r28}
				ctx.BindReg(r27, &d58)
				ctx.BindReg(r28, &d58)
			}
			var d59 JITValueDesc
			if d58.Loc == LocImm {
				d59 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d58.Imm.Float())}
			} else if d58.Type == tagFloat && d58.Loc == LocReg {
				d59 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d58.Reg}
				ctx.BindReg(d58.Reg, &d59)
				ctx.BindReg(d58.Reg, &d59)
			} else if d58.Type == tagFloat && d58.Loc == LocRegPair {
				ctx.FreeReg(d58.Reg)
				d59 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d58.Reg2}
				ctx.BindReg(d58.Reg2, &d59)
				ctx.BindReg(d58.Reg2, &d59)
			} else {
				d59 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d58}, 1)
				d59.Type = tagFloat
				ctx.BindReg(d59.Reg, &d59)
			}
			ctx.FreeDesc(&d58)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d59)
			var d60 JITValueDesc
			if d48.Loc == LocImm && d59.Loc == LocImm {
				d60 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d48.Imm.Float() - d59.Imm.Float())}
			} else if d48.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				_, xBits := d48.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d59.Reg)
				d60 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else if d59.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(scratch, d48.Reg)
				_, yBits := d59.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d60 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else {
				r29 := ctx.AllocRegExcept(d48.Reg, d59.Reg)
				ctx.W.EmitMovRegReg(r29, d48.Reg)
				ctx.W.EmitSubFloat64(r29, d59.Reg)
				d60 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r29}
				ctx.BindReg(r29, &d60)
			}
			if d60.Loc == LocReg && d48.Loc == LocReg && d60.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			var d61 JITValueDesc
			if d47.Loc == LocImm {
				d61 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(scratch, d47.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d61 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			}
			if d61.Loc == LocReg && d47.Loc == LocReg && d61.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = LocNone
			}
			d62 := d61
			if d62.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d62)
			ctx.EmitStoreToStack(d62, 24)
			d63 := d60
			if d63.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d63)
			ctx.EmitStoreToStack(d63, 32)
			ctx.W.EmitJmp(lbl34)
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
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl2 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 0)
			ctx.W.MarkLabel(lbl2)
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
			d4 := d3
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocImm && d4.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl5)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl4)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			lbl7 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 16)
			ctx.W.MarkLabel(lbl7)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d7 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d7)
			var d8 JITValueDesc
			if d6.Loc == LocImm && d7.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() < d7.Imm.Int())}
			} else if d7.Loc == LocImm {
				r4 := ctx.AllocRegExcept(d6.Reg)
				if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
					ctx.W.EmitCmpInt64(d6.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r4, CcL)
				d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d8)
			} else if d6.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d7.Reg)
				ctx.W.EmitSetcc(r5, CcL)
				d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d8)
			} else {
				r6 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitCmpInt64(d6.Reg, d7.Reg)
				ctx.W.EmitSetcc(r6, CcL)
				d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d8)
			}
			ctx.FreeDesc(&d7)
			d9 := d8
			ctx.EnsureDesc(&d9)
			if d9.Loc != LocImm && d9.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d9.Loc == LocImm {
				if d9.Imm.Bool() {
					ctx.W.MarkLabel(lbl10)
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.MarkLabel(lbl11)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d8)
			ctx.W.MarkLabel(lbl3)
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d2)
			var d10 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d10 = args[idx]
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
				lbl12 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r7, ai.Reg)
						ctx.W.EmitMovRegReg(r8, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r7, tmp.Reg)
						ctx.W.EmitMovRegReg(r8, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl12)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl12)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d10 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r7, Reg2: r8}
				ctx.BindReg(r7, &d10)
				ctx.BindReg(r8, &d10)
			}
			d12 := d10
			d12.ID = 0
			d11 := ctx.EmitTagEquals(&d12, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d10)
			d13 := d11
			ctx.EnsureDesc(&d13)
			if d13.Loc != LocImm && d13.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d13.Loc == LocImm {
				if d13.Imm.Bool() {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.MarkLabel(lbl15)
			d14 := d2
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 0)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d13.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl14)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl15)
			d15 := d2
			if d15.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d11)
			ctx.W.MarkLabel(lbl9)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d16 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d16)
			var d17 JITValueDesc
			if d6.Loc == LocImm && d16.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() == d16.Imm.Int())}
			} else if d16.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d6.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d6.Reg, int32(d16.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
					ctx.W.EmitCmpInt64(d6.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r9, CcE)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d17)
			} else if d6.Loc == LocImm {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d16.Reg)
				ctx.W.EmitSetcc(r10, CcE)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d17)
			} else {
				r11 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitCmpInt64(d6.Reg, d16.Reg)
				ctx.W.EmitSetcc(r11, CcE)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d17)
			}
			ctx.FreeDesc(&d16)
			d18 := d17
			ctx.EnsureDesc(&d18)
			if d18.Loc != LocImm && d18.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d18.Loc == LocImm {
				if d18.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d18.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d17)
			ctx.W.MarkLabel(lbl8)
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			var d19 JITValueDesc
			if d6.Loc == LocImm {
				idx := int(d6.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d19 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d6.Reg] {
					ctx.ProtectReg(d6.Reg)
					seen[d6.Reg] = true
					protected = append(protected, d6.Reg)
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
				lbl20 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d6.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r12, ai.Reg)
						ctx.W.EmitMovRegReg(r13, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r12, tmp.Reg)
						ctx.W.EmitMovRegReg(r13, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl20)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl20)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d19 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r12, Reg2: r13}
				ctx.BindReg(r12, &d19)
				ctx.BindReg(r13, &d19)
			}
			d21 := d19
			d21.ID = 0
			d20 := ctx.EmitTagEquals(&d21, tagInt, JITValueDesc{Loc: LocAny})
			d22 := d20
			ctx.EnsureDesc(&d22)
			if d22.Loc != LocImm && d22.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			if d22.Loc == LocImm {
				if d22.Imm.Bool() {
					ctx.W.MarkLabel(lbl23)
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.MarkLabel(lbl24)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d22.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl23)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl24)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d20)
			ctx.W.MarkLabel(lbl13)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl17)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			var d23 JITValueDesc
			if d5.Loc == LocImm {
				d23 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d5.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d5.Reg)
				d23 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d5.Reg}
				ctx.BindReg(d5.Reg, &d23)
			}
			lbl25 := ctx.W.ReserveLabel()
			d24 := d6
			if d24.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d24)
			ctx.EmitStoreToStack(d24, 32)
			d25 := d23
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			ctx.EmitStoreToStack(d25, 40)
			ctx.W.MarkLabel(lbl25)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d26 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d28 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d28)
			var d29 JITValueDesc
			if d26.Loc == LocImm && d28.Loc == LocImm {
				d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d26.Imm.Int() < d28.Imm.Int())}
			} else if d28.Loc == LocImm {
				r14 := ctx.AllocRegExcept(d26.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d26.Reg, int32(d28.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d28.Imm.Int()))
					ctx.W.EmitCmpInt64(d26.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r14, CcL)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
				ctx.BindReg(r14, &d29)
			} else if d26.Loc == LocImm {
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d26.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d28.Reg)
				ctx.W.EmitSetcc(r15, CcL)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d29)
			} else {
				r16 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitCmpInt64(d26.Reg, d28.Reg)
				ctx.W.EmitSetcc(r16, CcL)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d29)
			}
			ctx.FreeDesc(&d28)
			d30 := d29
			ctx.EnsureDesc(&d30)
			if d30.Loc != LocImm && d30.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d30.Loc == LocImm {
				if d30.Imm.Bool() {
					ctx.W.MarkLabel(lbl28)
					ctx.W.EmitJmp(lbl26)
				} else {
					ctx.W.MarkLabel(lbl29)
					ctx.W.EmitJmp(lbl27)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl28)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d29)
			ctx.W.MarkLabel(lbl16)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			ctx.W.EmitMakeInt(result, d5)
			if d5.Loc == LocReg { ctx.FreeReg(d5.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl22)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d32 := d19
			d32.ID = 0
			d31 := ctx.EmitTagEquals(&d32, tagFloat, JITValueDesc{Loc: LocAny})
			d33 := d31
			ctx.EnsureDesc(&d33)
			if d33.Loc != LocImm && d33.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d33.Loc == LocImm {
				if d33.Imm.Bool() {
					ctx.W.MarkLabel(lbl31)
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.W.MarkLabel(lbl32)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d33.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl31)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d31)
			ctx.W.MarkLabel(lbl21)
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			var d34 JITValueDesc
			if d19.Loc == LocImm {
				d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d19.Imm.Int())}
			} else if d19.Type == tagInt && d19.Loc == LocRegPair {
				ctx.FreeReg(d19.Reg)
				d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2}
				ctx.BindReg(d19.Reg2, &d34)
				ctx.BindReg(d19.Reg2, &d34)
			} else if d19.Type == tagInt && d19.Loc == LocReg {
				d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg}
				ctx.BindReg(d19.Reg, &d34)
				ctx.BindReg(d19.Reg, &d34)
			} else {
				d34 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d19}, 1)
				d34.Type = tagInt
				ctx.BindReg(d34.Reg, &d34)
			}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d34)
			var d35 JITValueDesc
			if d5.Loc == LocImm && d34.Loc == LocImm {
				d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() * d34.Imm.Int())}
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d34.Reg)
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d34.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, RegR11)
				}
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else {
				r17 := ctx.AllocRegExcept(d5.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r17, d5.Reg)
				ctx.W.EmitImulInt64(r17, d34.Reg)
				d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
				ctx.BindReg(r17, &d35)
			}
			if d35.Loc == LocReg && d5.Loc == LocReg && d35.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d34)
			lbl33 := ctx.W.ReserveLabel()
			d36 := d35
			if d36.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, 24)
			ctx.W.MarkLabel(lbl33)
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d37 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			var d38 JITValueDesc
			if d6.Loc == LocImm {
				d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			}
			if d38.Loc == LocReg && d6.Loc == LocReg && d38.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = LocNone
			}
			d39 := d37
			if d39.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d39)
			ctx.EmitStoreToStack(d39, 8)
			d40 := d38
			if d40.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d40)
			ctx.EmitStoreToStack(d40, 16)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl27)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d37 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d27)
			ctx.W.EmitMakeFloat(result, d27)
			if d27.Loc == LocReg { ctx.FreeReg(d27.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl26)
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d37 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d26)
			var d41 JITValueDesc
			if d26.Loc == LocImm {
				idx := int(d26.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d41 = args[idx]
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d26.Reg] {
					ctx.ProtectReg(d26.Reg)
					seen[d26.Reg] = true
					protected = append(protected, d26.Reg)
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
				lbl34 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d26.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r18, ai.Reg)
						ctx.W.EmitMovRegReg(r19, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r18, tmp.Reg)
						ctx.W.EmitMovRegReg(r19, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
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
				d41 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d41)
				ctx.BindReg(r19, &d41)
			}
			var d42 JITValueDesc
			if d41.Loc == LocImm {
				d42 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d41.Imm.Float())}
			} else if d41.Type == tagFloat && d41.Loc == LocReg {
				d42 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d41.Reg}
				ctx.BindReg(d41.Reg, &d42)
				ctx.BindReg(d41.Reg, &d42)
			} else if d41.Type == tagFloat && d41.Loc == LocRegPair {
				ctx.FreeReg(d41.Reg)
				d42 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d41.Reg2}
				ctx.BindReg(d41.Reg2, &d42)
				ctx.BindReg(d41.Reg2, &d42)
			} else {
				d42 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d41}, 1)
				d42.Type = tagFloat
				ctx.BindReg(d42.Reg, &d42)
			}
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d42)
			var d43 JITValueDesc
			if d27.Loc == LocImm && d42.Loc == LocImm {
				d43 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d27.Imm.Float() * d42.Imm.Float())}
			} else if d27.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				_, xBits := d27.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitMulFloat64(scratch, d42.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(scratch, d27.Reg)
				_, yBits := d42.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitMulFloat64(scratch, RegR11)
				d43 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r20 := ctx.AllocRegExcept(d27.Reg, d42.Reg)
				ctx.W.EmitMovRegReg(r20, d27.Reg)
				ctx.W.EmitMulFloat64(r20, d42.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r20}
				ctx.BindReg(r20, &d43)
			}
			if d43.Loc == LocReg && d27.Loc == LocReg && d43.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = LocNone
			}
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d26)
			var d44 JITValueDesc
			if d26.Loc == LocImm {
				d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d26.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(scratch, d26.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			}
			if d44.Loc == LocReg && d26.Loc == LocReg && d44.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = LocNone
			}
			d45 := d44
			if d45.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d45)
			ctx.EmitStoreToStack(d45, 32)
			d46 := d43
			if d46.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d46)
			ctx.EmitStoreToStack(d46, 40)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl30)
			d37 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			var d47 JITValueDesc
			if d19.Loc == LocImm {
				d47 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d19.Imm.Float())}
			} else if d19.Type == tagFloat && d19.Loc == LocReg {
				d47 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d19.Reg}
				ctx.BindReg(d19.Reg, &d47)
				ctx.BindReg(d19.Reg, &d47)
			} else if d19.Type == tagFloat && d19.Loc == LocRegPair {
				ctx.FreeReg(d19.Reg)
				d47 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d19.Reg2}
				ctx.BindReg(d19.Reg2, &d47)
				ctx.BindReg(d19.Reg2, &d47)
			} else {
				d47 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d19}, 1)
				d47.Type = tagFloat
				ctx.BindReg(d47.Reg, &d47)
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d47)
			var d48 JITValueDesc
			if d47.Loc == LocImm {
				d48 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d47.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d47)
				var truncSrc Reg
				if d47.Loc == LocRegPair {
					ctx.FreeReg(d47.Reg)
					truncSrc = d47.Reg2
				} else {
					truncSrc = d47.Reg
				}
				truncInt := ctx.AllocRegExcept(truncSrc)
				ctx.W.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
				ctx.W.EmitCvtInt64ToFloat64(RegX0, truncInt)
				d48 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
				ctx.BindReg(truncInt, &d48)
				ctx.BindReg(truncInt, &d48)
			}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			var d49 JITValueDesc
			if d47.Loc == LocImm && d48.Loc == LocImm {
				d49 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d47.Imm.Float() == d48.Imm.Float())}
			} else if d48.Loc == LocImm {
				r21 := ctx.AllocRegExcept(d47.Reg)
				_, yBits := d48.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitCmpFloat64Setcc(r21, d47.Reg, RegR11, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
				ctx.BindReg(r21, &d49)
			} else if d47.Loc == LocImm {
				r22 := ctx.AllocRegExcept(d48.Reg)
				_, xBits := d47.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, xBits)
				ctx.W.EmitCmpFloat64Setcc(r22, RegR11, d48.Reg, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d49)
			} else {
				r23 := ctx.AllocRegExcept(d47.Reg, d48.Reg)
				ctx.W.EmitCmpFloat64Setcc(r23, d47.Reg, d48.Reg, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d49)
			}
			ctx.FreeDesc(&d48)
			d50 := d49
			ctx.EnsureDesc(&d50)
			if d50.Loc != LocImm && d50.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			if d50.Loc == LocImm {
				if d50.Imm.Bool() {
					ctx.W.MarkLabel(lbl36)
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.MarkLabel(lbl37)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d50.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl36)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl36)
				ctx.W.EmitJmp(lbl35)
				ctx.W.MarkLabel(lbl37)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d49)
			ctx.W.MarkLabel(lbl35)
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d37 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d26 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d27 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			var d51 JITValueDesc
			if d47.Loc == LocImm {
				d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d47.Imm.Float()))}
			} else {
				r24 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r24, d47.Reg)
				d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r24}
				ctx.BindReg(r24, &d51)
			}
			ctx.FreeDesc(&d47)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d51)
			var d52 JITValueDesc
			if d5.Loc == LocImm && d51.Loc == LocImm {
				d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d5.Imm.Int() * d51.Imm.Int())}
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d5.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d51.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d51.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, RegR11)
				}
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else {
				r25 := ctx.AllocRegExcept(d5.Reg, d51.Reg)
				ctx.W.EmitMovRegReg(r25, d5.Reg)
				ctx.W.EmitImulInt64(r25, d51.Reg)
				d52 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r25}
				ctx.BindReg(r25, &d52)
			}
			if d52.Loc == LocReg && d5.Loc == LocReg && d52.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d51)
			d53 := d52
			if d53.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d53)
			ctx.EmitStoreToStack(d53, 24)
			ctx.W.EmitJmp(lbl33)
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
		nil /* TODO: Slice on non-desc: slice a[1:int:] */, /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[1]
			d1 := args[0]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d2.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d2.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d2.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d2.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[1]
			d1 := args[0]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d2.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d2.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d2.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d2.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Equal arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Equal arg1)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d1 := args[1]
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (EqualSQL arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (EqualSQL arg1)")
			}
			d2 := ctx.EmitGoCallScalar(GoFuncAddr(EqualSQL), []JITValueDesc{d0, d1}, 2)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d2, &result)
				result.Type = d2.Type
			} else {
				switch d2.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d2)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d2)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d2)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					panic("jit: single-block scalar return with unknown type")
				}
			}
			return result
			}
			return bbs[0].Render()
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
		nil /* TODO: unsupported compare const kind: (Scmer).String(t22) */, /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
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
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d2 := d0
			d2.ID = 0
			d1 := ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d1.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d1.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d1.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d2 := d0
			d2.ID = 0
			d1 := ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d1.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d1.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d1.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
			d0 := args[0]
			d2 := d0
			d2.ID = 0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
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
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl2 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 16)
			ctx.W.MarkLabel(lbl2)
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
			d5 := d4
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocImm && d5.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl5)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d4)
			ctx.W.MarkLabel(lbl4)
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
			ctx.W.MarkLabel(lbl3)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d3)
			var d6 JITValueDesc
			if d3.Loc == LocImm {
				idx := int(d3.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d6 = args[idx]
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
				lbl7 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl7)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl7)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d6 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d6)
				ctx.BindReg(r5, &d6)
			}
			d8 := d1
			d8.ID = 0
			d7 := ctx.EmitTagEquals(&d8, tagNil, JITValueDesc{Loc: LocAny})
			d9 := d7
			ctx.EnsureDesc(&d9)
			if d9.Loc != LocImm && d9.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d9.Loc == LocImm {
				if d9.Imm.Bool() {
					ctx.W.MarkLabel(lbl10)
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.MarkLabel(lbl11)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl9)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d11 := d6
			d11.ID = 0
			d10 := ctx.EmitTagEquals(&d11, tagNil, JITValueDesc{Loc: LocAny})
			d12 := d10
			ctx.EnsureDesc(&d12)
			if d12.Loc != LocImm && d12.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d12.Loc == LocImm {
				if d12.Imm.Bool() {
					ctx.W.MarkLabel(lbl13)
			d13 := d1
			if d13.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			if d13.Loc == LocRegPair || d13.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d13, 0)
			} else {
				ctx.EmitStoreToStack(d13, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d14 := d3
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 16)
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d12.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl13)
			d15 := d1
			if d15.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			if d15.Loc == LocRegPair || d15.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d15, 0)
			} else {
				ctx.EmitStoreToStack(d15, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d16 := d3
			if d16.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, 16)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl8)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d17 := d6
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
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl12)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			if d6.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d6.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d6)
				} else if d6.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d6)
				} else if d6.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d6)
				} else if d6.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d6.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d6 = tmpPair
			} else if d6.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocRegExcept(d6.Reg), Reg2: ctx.AllocRegExcept(d6.Reg)}
				switch d6.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d6)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d6)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d6)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d6)
				d6 = tmpPair
			}
			if d6.Loc != LocRegPair && d6.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d19 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d6, d1}, 1)
			d20 := d19
			ctx.EnsureDesc(&d20)
			if d20.Loc != LocImm && d20.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			if d20.Loc == LocImm {
				if d20.Imm.Bool() {
					ctx.W.MarkLabel(lbl16)
					ctx.W.EmitJmp(lbl15)
				} else {
					ctx.W.MarkLabel(lbl17)
			d21 := d1
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
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d20.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl16)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl17)
			d23 := d1
			if d23.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocRegPair || d23.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d23, 0)
			} else {
				ctx.EmitStoreToStack(d23, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d24 := d3
			if d24.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d24)
			ctx.EmitStoreToStack(d24, 16)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d19)
			ctx.W.MarkLabel(lbl15)
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d25 := d6
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			if d25.Loc == LocRegPair || d25.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d25, 0)
			} else {
				ctx.EmitStoreToStack(d25, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d26 := d3
			if d26.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, 16)
			ctx.W.EmitJmp(lbl2)
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
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			lbl2 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 16)
			ctx.W.MarkLabel(lbl2)
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
			d5 := d4
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocImm && d5.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl5)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d4)
			ctx.W.MarkLabel(lbl4)
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
			ctx.W.MarkLabel(lbl3)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d3)
			var d6 JITValueDesc
			if d3.Loc == LocImm {
				idx := int(d3.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d6 = args[idx]
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
				lbl7 := ctx.W.ReserveLabel()
				typ := uint16(JITTypeUnknown)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
						typ = ai.Type
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						typ = tmp.Type
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
						typ = ai.Imm.GetTag()
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl7)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.EmitByte(0xCC) // unreachable: dynamic args index out of range
				ctx.W.MarkLabel(lbl7)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d6 = JITValueDesc{Loc: LocRegPair, Type: typ, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d6)
				ctx.BindReg(r5, &d6)
			}
			d8 := d1
			d8.ID = 0
			d7 := ctx.EmitTagEquals(&d8, tagNil, JITValueDesc{Loc: LocAny})
			d9 := d7
			ctx.EnsureDesc(&d9)
			if d9.Loc != LocImm && d9.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d9.Loc == LocImm {
				if d9.Imm.Bool() {
					ctx.W.MarkLabel(lbl10)
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.MarkLabel(lbl11)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl10)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d7)
			ctx.W.MarkLabel(lbl9)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d11 := d6
			d11.ID = 0
			d10 := ctx.EmitTagEquals(&d11, tagNil, JITValueDesc{Loc: LocAny})
			d12 := d10
			ctx.EnsureDesc(&d12)
			if d12.Loc != LocImm && d12.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d12.Loc == LocImm {
				if d12.Imm.Bool() {
					ctx.W.MarkLabel(lbl13)
			d13 := d1
			if d13.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			if d13.Loc == LocRegPair || d13.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d13, 0)
			} else {
				ctx.EmitStoreToStack(d13, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d14 := d3
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 16)
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d12.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl13)
			d15 := d1
			if d15.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			if d15.Loc == LocRegPair || d15.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d15, 0)
			} else {
				ctx.EmitStoreToStack(d15, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d16 := d3
			if d16.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, 16)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl8)
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d17 := d6
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
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl12)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			if d6.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d6.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d6)
				} else if d6.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d6)
				} else if d6.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d6)
				} else if d6.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d6.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d6 = tmpPair
			} else if d6.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d6.Type, Reg: ctx.AllocRegExcept(d6.Reg), Reg2: ctx.AllocRegExcept(d6.Reg)}
				switch d6.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d6)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d6)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d6)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d6)
				d6 = tmpPair
			}
			if d6.Loc != LocRegPair && d6.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d19 := ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d1, d6}, 1)
			d20 := d19
			ctx.EnsureDesc(&d20)
			if d20.Loc != LocImm && d20.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			if d20.Loc == LocImm {
				if d20.Imm.Bool() {
					ctx.W.MarkLabel(lbl16)
					ctx.W.EmitJmp(lbl15)
				} else {
					ctx.W.MarkLabel(lbl17)
			d21 := d1
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
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d20.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl16)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl17)
			d23 := d1
			if d23.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocRegPair || d23.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d23, 0)
			} else {
				ctx.EmitStoreToStack(d23, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d24 := d3
			if d24.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d24)
			ctx.EmitStoreToStack(d24, 16)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d19)
			ctx.W.MarkLabel(lbl15)
			d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d25 := d6
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			if d25.Loc == LocRegPair || d25.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d25, 0)
			} else {
				ctx.EmitStoreToStack(d25, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d26 := d3
			if d26.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, 16)
			ctx.W.EmitJmp(lbl2)
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
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
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Floor(d1.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d1)
				var d3 JITValueDesc
				if d1.Loc == LocRegPair {
					ctx.FreeReg(d1.Reg)
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg2}
					ctx.BindReg(d1.Reg2, &d3)
					ctx.BindReg(d1.Reg2, &d3)
				} else {
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
					ctx.BindReg(d1.Reg, &d3)
					ctx.BindReg(d1.Reg, &d3)
				}
				d2 = ctx.EmitGoCallScalar(GoFuncAddr(JITFloorBits), []JITValueDesc{d3}, 1)
				d2.Type = tagFloat
				ctx.BindReg(d2.Reg, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
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
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Ceil(d1.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d1)
				var d3 JITValueDesc
				if d1.Loc == LocRegPair {
					ctx.FreeReg(d1.Reg)
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg2}
					ctx.BindReg(d1.Reg2, &d3)
					ctx.BindReg(d1.Reg2, &d3)
				} else {
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
					ctx.BindReg(d1.Reg, &d3)
					ctx.BindReg(d1.Reg, &d3)
				}
				d2 = ctx.EmitGoCallScalar(GoFuncAddr(JITCeilBits), []JITValueDesc{d3}, 1)
				d2.Type = tagFloat
				ctx.BindReg(d2.Reg, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			}
			return bbs[0].Render()
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
			var bbs [1]BBDescriptor
			bbs[0].Render = func() JITValueDesc {
			lbl0 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl0)
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
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			}
			return bbs[0].Render()
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
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := args[0]
			d2 := d0
			d2.ID = 0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			d3 := d1
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
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl3)
			d4 := args[0]
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float())}
			} else if d4.Type == tagFloat && d4.Loc == LocReg {
				d5 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d4.Reg}
				ctx.BindReg(d4.Reg, &d5)
				ctx.BindReg(d4.Reg, &d5)
			} else if d4.Type == tagFloat && d4.Loc == LocRegPair {
				ctx.FreeReg(d4.Reg)
				d5 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d4.Reg2}
				ctx.BindReg(d4.Reg2, &d5)
				ctx.BindReg(d4.Reg2, &d5)
			} else {
				d5 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d4}, 1)
				d5.Type = tagFloat
				ctx.BindReg(d5.Reg, &d5)
			}
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d5)
			var d6 JITValueDesc
			if d5.Loc == LocImm {
				d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Float() < 0)}
			} else {
				r1 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegImm64(RegR11, uint64(0))
				ctx.W.EmitCmpFloat64Setcc(r1, d5.Reg, RegR11, CcL)
				d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d6)
			}
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
			d8 := d5
			if d8.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d8)
			ctx.EmitStoreToStack(d8, 0)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl8)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
			d9 := d5
			if d9.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d9)
			ctx.EmitStoreToStack(d9, 0)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl2)
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			d10 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d11 := args[0]
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d11)
			if d11.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d11.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d11)
				} else if d11.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d11)
				} else if d11.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d11)
				} else if d11.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d11.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d11 = tmpPair
			} else if d11.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocRegExcept(d11.Reg), Reg2: ctx.AllocRegExcept(d11.Reg)}
				switch d11.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d11)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d11)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d11)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d11)
				d11 = tmpPair
			}
			if d11.Loc != LocRegPair && d11.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d12 := ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d11}, 1)
			ctx.FreeDesc(&d11)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			var d13 JITValueDesc
			if d10.Loc == LocImm {
				d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.Imm.Float()))}
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r2, d10.Reg)
				d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
				ctx.BindReg(r2, &d13)
			}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d13)
			var d14 JITValueDesc
			if d12.Loc == LocImm && d13.Loc == LocImm {
				d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() == d13.Imm.Int())}
			} else if d13.Loc == LocImm {
				r3 := ctx.AllocReg()
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d12.Reg, int32(d13.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
					ctx.W.EmitCmpInt64(d12.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r3, CcE)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d14)
			} else if d12.Loc == LocImm {
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d13.Reg)
				ctx.W.EmitSetcc(r4, CcE)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d14)
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d12.Reg, d13.Reg)
				ctx.W.EmitSetcc(r5, CcE)
				d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d14)
			}
			ctx.FreeDesc(&d12)
			ctx.FreeDesc(&d13)
			d15 := d14
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
			ctx.FreeDesc(&d14)
			ctx.W.MarkLabel(lbl6)
			d10 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d16 JITValueDesc
			if d5.Loc == LocImm {
				if d5.Type == tagFloat {
					d16 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d5.Imm.Float())}
				} else {
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d5.Imm.Int())}
				}
			} else {
				if d5.Type == tagFloat {
					r6 := ctx.AllocRegExcept(d5.Reg)
					ctx.W.EmitMovRegImm64(r6, 0)
					ctx.W.EmitSubFloat64(r6, d5.Reg)
					d16 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
					ctx.BindReg(r6, &d16)
				} else {
					r7 := ctx.AllocRegExcept(d5.Reg)
					ctx.W.EmitMovRegImm64(r7, 0)
					ctx.W.EmitSubInt64(r7, d5.Reg)
					d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r7}
					ctx.BindReg(r7, &d16)
				}
			}
			d17 := d16
			if d17.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			ctx.EmitStoreToStack(d17, 0)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl11)
			d10 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			ctx.W.EmitMakeFloat(result, d10)
			if d10.Loc == LocReg { ctx.FreeReg(d10.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			d10 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d18 := args[0]
			var d19 JITValueDesc
			if d18.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d18.Imm.Float())}
			} else if d18.Type == tagFloat && d18.Loc == LocReg {
				d19 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d18.Reg}
				ctx.BindReg(d18.Reg, &d19)
				ctx.BindReg(d18.Reg, &d19)
			} else if d18.Type == tagFloat && d18.Loc == LocRegPair {
				ctx.FreeReg(d18.Reg)
				d19 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d18.Reg2}
				ctx.BindReg(d18.Reg2, &d19)
				ctx.BindReg(d18.Reg2, &d19)
			} else {
				d19 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d18}, 1)
				d19.Type = tagFloat
				ctx.BindReg(d19.Reg, &d19)
			}
			ctx.FreeDesc(&d18)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d10)
			var d20 JITValueDesc
			if d19.Loc == LocImm && d10.Loc == LocImm {
				d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.Float() == d10.Imm.Float())}
			} else if d10.Loc == LocImm {
				r8 := ctx.AllocReg()
				_, yBits := d10.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitCmpFloat64Setcc(r8, d19.Reg, RegR11, CcE)
				d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d20)
			} else if d19.Loc == LocImm {
				r9 := ctx.AllocRegExcept(d10.Reg)
				_, xBits := d19.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, xBits)
				ctx.W.EmitCmpFloat64Setcc(r9, RegR11, d10.Reg, CcE)
				d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d20)
			} else {
				r10 := ctx.AllocRegExcept(d19.Reg, d10.Reg)
				ctx.W.EmitCmpFloat64Setcc(r10, d19.Reg, d10.Reg, CcE)
				d20 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d20)
			}
			ctx.FreeDesc(&d19)
			d21 := d20
			ctx.EnsureDesc(&d21)
			if d21.Loc != LocImm && d21.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d21.Loc == LocImm {
				if d21.Imm.Bool() {
					ctx.W.MarkLabel(lbl15)
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.MarkLabel(lbl16)
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d21.Reg, 0)
				ctx.W.EmitJcc(CcNE, lbl15)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d20)
			ctx.W.MarkLabel(lbl14)
			d10 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			var d22 JITValueDesc
			if d10.Loc == LocImm {
				d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.Imm.Float()))}
			} else {
				r11 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r11, d10.Reg)
				d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d22)
			}
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			ctx.W.EmitMakeInt(result, d22)
			if d22.Loc == LocReg { ctx.FreeReg(d22.Reg) }
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
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			d0 := args[0]
			d2 := d0
			d2.ID = 0
			d1 := ctx.EmitTagEquals(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			d3 := d1
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
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl3)
			d4 := args[0]
			var d5 JITValueDesc
			if d4.Loc == LocImm {
				d5 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float())}
			} else if d4.Type == tagFloat && d4.Loc == LocReg {
				d5 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d4.Reg}
				ctx.BindReg(d4.Reg, &d5)
				ctx.BindReg(d4.Reg, &d5)
			} else if d4.Type == tagFloat && d4.Loc == LocRegPair {
				ctx.FreeReg(d4.Reg)
				d5 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d4.Reg2}
				ctx.BindReg(d4.Reg2, &d5)
				ctx.BindReg(d4.Reg2, &d5)
			} else {
				d5 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d4}, 1)
				d5.Type = tagFloat
				ctx.BindReg(d5.Reg, &d5)
			}
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d5)
			var d6 JITValueDesc
			if d5.Loc == LocImm {
				d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d5.Imm.Float() < 0)}
			} else {
				r0 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegImm64(RegR11, uint64(0))
				ctx.W.EmitCmpFloat64Setcc(r0, d5.Reg, RegR11, CcL)
				d6 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d6)
			}
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
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			ctx.EnsureDesc(&d5)
			var d8 JITValueDesc
			if d5.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d5.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d5)
				var d9 JITValueDesc
				if d5.Loc == LocRegPair {
					ctx.FreeReg(d5.Reg)
					d9 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d5.Reg2}
					ctx.BindReg(d5.Reg2, &d9)
					ctx.BindReg(d5.Reg2, &d9)
				} else {
					d9 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d5.Reg}
					ctx.BindReg(d5.Reg, &d9)
					ctx.BindReg(d5.Reg, &d9)
				}
				d8 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d9}, 1)
				d8.Type = tagFloat
				ctx.BindReg(d8.Reg, &d8)
			}
			ctx.FreeDesc(&d5)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d8)
			ctx.W.EmitMakeFloat(result, d8)
			if d8.Loc == LocReg { ctx.FreeReg(d8.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl6)
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
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
}
