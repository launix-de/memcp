/*
Copyright (C) 2025-2026  Carl-Philip Hänsch

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
package scm

import (
	"math"
	"strings"
)

func init_vector() {
	// string functions
	DeclareTitle("Vectors")

	Declare(&Globalenv, &Declaration{
		Name: "dot",

		Fn: func(a ...Scmer) Scmer {
			var result float64
			v1 := asSlice(a[0], "dot v1")
			v2 := asSlice(a[1], "dot v2")
			mode := "DOT"
			if len(a) > 2 {
				mode = strings.ToUpper(String(a[2]))
			}
			if mode == "COSINE" {
				// COSINE
				var lena float64 = 0
				var lenb float64 = 0
				for i := 0; i < len(v1) && i < len(v2); i++ {
					w1 := ToFloat(v1[i])
					w2 := ToFloat(v2[i])
					lena += w1 * w1
					lenb += w2 * w2
					result += w1 * w2
				}
				result = result / math.Sqrt(lena*lenb)
			} else {
				// DOT AND EUCLIDEAN
				for i := 0; i < len(v1) && i < len(v2); i++ {
					result += ToFloat(v1[i]) * ToFloat(v2[i])
				}
				if mode == "EUCLIDEAN" {
					result = math.Sqrt(result)
				}
			}
			return NewFloat(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "produced the dot product",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "list", Label: "v1", Description: "vector1"}, &TypeDescriptor{Kind: "list", Label: "v2", Description: "vector2"}, &TypeDescriptor{Kind: "string", Label: "mode", Description: "DOT, COSINE, EUCLIDEAN, default is DOT", Optional: true}},
			Return: &TypeDescriptor{Kind: "number"},
			Const:  true,

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["dot"].Fn, args, result)
				}
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
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d18 JITValueDesc
				_ = d18
				var d21 JITValueDesc
				_ = d21
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d55 JITValueDesc
				_ = d55
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d108 JITValueDesc
				_ = d108
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d167 JITValueDesc
				_ = d167
				var d168 JITValueDesc
				_ = d168
				var d169 JITValueDesc
				_ = d169
				var d170 JITValueDesc
				_ = d170
				var d171 JITValueDesc
				_ = d171
				var d172 JITValueDesc
				_ = d172
				var d173 JITValueDesc
				_ = d173
				var d174 JITValueDesc
				_ = d174
				var d175 JITValueDesc
				_ = d175
				var d176 JITValueDesc
				_ = d176
				var d177 JITValueDesc
				_ = d177
				var d178 JITValueDesc
				_ = d178
				var d180 JITValueDesc
				_ = d180
				var d181 JITValueDesc
				_ = d181
				var d182 JITValueDesc
				_ = d182
				var d183 JITValueDesc
				_ = d183
				var d185 JITValueDesc
				_ = d185
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var d264 JITValueDesc
				_ = d264
				var d265 JITValueDesc
				_ = d265
				var d266 JITValueDesc
				_ = d266
				var d267 JITValueDesc
				_ = d267
				var d268 JITValueDesc
				_ = d268
				var d271 JITValueDesc
				_ = d271
				var d272 JITValueDesc
				_ = d272
				var d354 JITValueDesc
				_ = d354
				var d355 JITValueDesc
				_ = d355
				var d356 JITValueDesc
				_ = d356
				var d357 JITValueDesc
				_ = d357
				var d358 JITValueDesc
				_ = d358
				var d359 JITValueDesc
				_ = d359
				var d360 JITValueDesc
				_ = d360
				var d361 JITValueDesc
				_ = d361
				var d362 JITValueDesc
				_ = d362
				var d363 JITValueDesc
				_ = d363
				var d364 JITValueDesc
				_ = d364
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				var d368 JITValueDesc
				_ = d368
				var d369 JITValueDesc
				_ = d369
				var d370 JITValueDesc
				_ = d370
				var d372 JITValueDesc
				_ = d372
				var d374 JITValueDesc
				_ = d374
				var d375 JITValueDesc
				_ = d375
				var d378 JITValueDesc
				_ = d378
				var d478 JITValueDesc
				_ = d478
				var d479 JITValueDesc
				_ = d479
				var d480 JITValueDesc
				_ = d480
				var d587 JITValueDesc
				_ = d587
				var d588 JITValueDesc
				_ = d588
				var d589 JITValueDesc
				_ = d589
				var d591 JITValueDesc
				_ = d591
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(128))
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				d3 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
				_ = d3
				d4 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
				_ = d4
				d5 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
				_ = d5
				d6 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
				_ = d6
				d7 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
				_ = d7
				d8 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
				_ = d8
				var bbs [15]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[4].PhiCount = uint16(1)
				bbs[6].PhiBase = int32(phiBase0) + int32(32)
				bbs[6].PhiCount = uint16(4)
				bbs[10].PhiBase = int32(phiBase0) + int32(96)
				bbs[10].PhiCount = uint16(2)
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
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				bbpos_0_13 := int32(-1)
				_ = bbpos_0_13
				lbl14 := ctx.ReserveLabel()
				bbpos_0_14 := int32(-1)
				_ = bbpos_0_14
				lbl15 := ctx.ReserveLabel()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					ctx.ReclaimUntrackedRegs()
					d9 = args[0]
					d9.ID = 0
					var d10 JITValueDesc
					if d9.Type == tagSlice {
						d10 = jitKnownSliceHeader(ctx, &d9)
					} else {
						d10 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d9}, 3)
					}
					ctx.BindReg(d10.Reg, &d10)
					ctx.BindReg(d10.Reg2, &d10)
					ctx.BindReg(d10.Reg3, &d10)
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.FreeDesc(&d9)
					d11 = args[1]
					d11.ID = 0
					var d12 JITValueDesc
					if d11.Type == tagSlice {
						d12 = jitKnownSliceHeader(ctx, &d11)
					} else {
						d12 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d11}, 3)
					}
					ctx.BindReg(d12.Reg, &d12)
					ctx.BindReg(d12.Reg2, &d12)
					ctx.BindReg(d12.Reg3, &d12)
					ctx.StabilizeDescForControlFlow(&d12)
					ctx.FreeDesc(&d11)
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d13)
					var d14 JITValueDesc
					if d13.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() > 2)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d13.Reg, 2)
						ctx.EmitSetcc(r0, CondSignedGreater)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d14)
					}
					ctx.FreeDesc(&d13)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							if ps.General {
							}
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
							ps16.OverlayValues[10] = d10
							ps16.OverlayValues[11] = d11
							ps16.OverlayValues[12] = d12
							ps16.OverlayValues[13] = d13
							ps16.OverlayValues[14] = d14
							ps16.OverlayValues[15] = d15
							return bbs[1].RenderPS(ps16)
						}
						if ps.General {
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
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
						ps17.OverlayValues[10] = d10
						ps17.OverlayValues[11] = d11
						ps17.OverlayValues[12] = d12
						ps17.OverlayValues[13] = d13
						ps17.OverlayValues[14] = d14
						ps17.OverlayValues[15] = d15
						ps17.PhiValues = make([]JITValueDesc, 1)
						d18 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
						ps17.PhiValues[0] = d18
						return bbs[2].RenderPS(ps17)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d15.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl17)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
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
					ps19.OverlayValues[10] = d10
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
					ps20.OverlayValues[10] = d10
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[18] = d18
					ps20.PhiValues = make([]JITValueDesc, 1)
					d21 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
					ps20.PhiValues[0] = d21
					snap22 := d1
					snap23 := d2
					snap24 := d3
					snap25 := d4
					snap26 := d5
					snap27 := d6
					snap28 := d7
					snap29 := d8
					snap30 := d9
					snap31 := d10
					snap32 := d11
					snap33 := d12
					snap34 := d13
					snap35 := d14
					snap36 := d15
					snap37 := d18
					snap38 := d21
					alloc39 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps20)
					}
					ctx.RestoreAllocState(alloc39)
					d1 = snap22
					d2 = snap23
					d3 = snap24
					d4 = snap25
					d5 = snap26
					d6 = snap27
					d7 = snap28
					d8 = snap29
					d9 = snap30
					d10 = snap31
					d11 = snap32
					d12 = snap33
					d13 = snap34
					d14 = snap35
					d15 = snap36
					d18 = snap37
					d21 = snap38
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps19)
					}
					return result
					ctx.FreeDesc(&d14)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					ctx.ReclaimUntrackedRegs()
					d40 = args[2]
					d40.ID = 0
					d42 = d40
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						tag := d42.Imm.GetTag()
						switch tag {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d42)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d42)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d42)
						case tagNil:
							ctx.EmitMakeNil(tmpPair)
						default:
							ptrWord, auxWord := d42.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d42 = tmpPair
					} else if d42.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(d42.Reg), Reg2: ctx.AllocRegExcept(d42.Reg)}
						switch d42.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d42)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d42)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d42)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&d42)
						d42 = tmpPair
					} else if d42.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d42.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d42.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocRegExcept(tmpScalar.Reg), Reg2: ctx.AllocRegExcept(tmpScalar.Reg)}
						switch tmpScalar.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, tmpScalar)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, tmpScalar)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, tmpScalar)
						default:
							panic("jit: Scmer.String requires Scmer pair receiver")
						}
						ctx.FreeDesc(&tmpScalar)
						d42 = tmpPair
					}
					if d42.Loc != LocRegPair && d42.Loc != LocStackPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d41 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d42}, 2)
					ctx.FreeDesc(&d40)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d41)
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d41.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d41.Imm)
						ptrWord, _ := d41.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d41.Imm.String())))
						d41 = tmpPair
					} else if d41.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d41.Type, Reg: ctx.AllocRegExcept(d41.Reg), Reg2: ctx.AllocRegExcept(d41.Reg)}
						switch d41.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d41)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d41)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d41)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d41)
						d41 = tmpPair
					}
					if d41.Loc != LocRegPair && d41.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d41)
					d43 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d41}, 2)
					d43.NoHeapPointer = false
					ctx.BindReg(d43.Reg, &d43)
					ctx.BindReg(d43.Reg2, &d43)
					ctx.StabilizeDescForControlFlow(&d43)
					if ps.General {
						ctx.SyncDesc(&d43)
						if d43.Loc == LocReg {
							ctx.ProtectReg(d43.Reg)
						} else if d43.Loc == LocRegPair {
							ctx.ProtectReg(d43.Reg)
							ctx.ProtectReg(d43.Reg2)
						}
						d44 = d43
						if d44.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d44)
						if d44.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d44, int32(bbs[2].PhiBase)+int32(0), 2)
						} else if d44.Loc == LocInputPair {
							ctx.EnsureDesc(&d44)
							ctx.EmitStoreScmerToStack(d44, int32(bbs[2].PhiBase)+int32(0))
						} else if d44.Loc == LocRegPair || d44.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d44, int32(bbs[2].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d44)
							ctx.EmitStoreToStack(d44, int32(bbs[2].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
						}
						if d43.Loc == LocReg {
							ctx.UnprotectReg(d43.Reg)
						} else if d43.Loc == LocRegPair {
							ctx.UnprotectReg(d43.Reg)
							ctx.UnprotectReg(d43.Reg2)
						}
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
					ps45.OverlayValues[10] = d10
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[12] = d12
					ps45.OverlayValues[13] = d13
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[15] = d15
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[21] = d21
					ps45.OverlayValues[40] = d40
					ps45.OverlayValues[41] = d41
					ps45.OverlayValues[42] = d42
					ps45.OverlayValues[43] = d43
					ps45.OverlayValues[44] = d44
					ps45.PhiValues = make([]JITValueDesc, 1)
					d46 = d43
					ps45.PhiValues[0] = d46
					if ps45.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps45)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d47 := ps.PhiValues[0]
							ctx.EnsureDesc(&d47)
							ctx.EmitStoreScmerToStack(d47, int32(bbs[2].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					var d48 JITValueDesc
					if d1.Loc == LocImm {
						ctx.TrackImm(d1.Imm)
						ptrWord, _ := d1.Imm.RawWords()
						d48 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d48.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d48.Reg2, uint64(len(d1.Imm.String())))
						ctx.BindReg(d48.Reg, &d48)
						ctx.BindReg(d48.Reg2, &d48)
					} else {
						d48 = d1
					}
					d49 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("COSINE")}
					var d50 JITValueDesc
					if d49.Loc == LocImm {
						ctx.TrackImm(d49.Imm)
						ptrWord, _ := d49.Imm.RawWords()
						d50 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d50.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d50.Reg2, uint64(len(d49.Imm.String())))
						ctx.BindReg(d50.Reg, &d50)
						ctx.BindReg(d50.Reg2, &d50)
					} else {
						d50 = d49
					}
					d51 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d48, d50}, 1)
					ctx.EmitAndRegImm32(d51.Reg, 1)
					d51.Type = tagBool
					ctx.BindReg(d51.Reg, &d51)
					d52 = d51
					ctx.EnsureDesc(&d52)
					if d52.Loc != LocImm && d52.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d52.Loc == LocImm {
						if d52.Imm.Bool() {
							if ps.General {
							}
							ps53 := PhiState{General: ps.General}
							ps53.OverlayValues = make([]JITValueDesc, 53)
							ps53.OverlayValues[1] = d1
							ps53.OverlayValues[2] = d2
							ps53.OverlayValues[3] = d3
							ps53.OverlayValues[4] = d4
							ps53.OverlayValues[5] = d5
							ps53.OverlayValues[6] = d6
							ps53.OverlayValues[7] = d7
							ps53.OverlayValues[8] = d8
							ps53.OverlayValues[9] = d9
							ps53.OverlayValues[10] = d10
							ps53.OverlayValues[11] = d11
							ps53.OverlayValues[12] = d12
							ps53.OverlayValues[13] = d13
							ps53.OverlayValues[14] = d14
							ps53.OverlayValues[15] = d15
							ps53.OverlayValues[18] = d18
							ps53.OverlayValues[21] = d21
							ps53.OverlayValues[40] = d40
							ps53.OverlayValues[41] = d41
							ps53.OverlayValues[42] = d42
							ps53.OverlayValues[43] = d43
							ps53.OverlayValues[44] = d44
							ps53.OverlayValues[46] = d46
							ps53.OverlayValues[47] = d47
							ps53.OverlayValues[48] = d48
							ps53.OverlayValues[49] = d49
							ps53.OverlayValues[50] = d50
							ps53.OverlayValues[51] = d51
							ps53.OverlayValues[52] = d52
							return bbs[3].RenderPS(ps53)
						}
						if ps.General {
						}
						ps54 := PhiState{General: ps.General}
						ps54.OverlayValues = make([]JITValueDesc, 53)
						ps54.OverlayValues[1] = d1
						ps54.OverlayValues[2] = d2
						ps54.OverlayValues[3] = d3
						ps54.OverlayValues[4] = d4
						ps54.OverlayValues[5] = d5
						ps54.OverlayValues[6] = d6
						ps54.OverlayValues[7] = d7
						ps54.OverlayValues[8] = d8
						ps54.OverlayValues[9] = d9
						ps54.OverlayValues[10] = d10
						ps54.OverlayValues[11] = d11
						ps54.OverlayValues[12] = d12
						ps54.OverlayValues[13] = d13
						ps54.OverlayValues[14] = d14
						ps54.OverlayValues[15] = d15
						ps54.OverlayValues[18] = d18
						ps54.OverlayValues[21] = d21
						ps54.OverlayValues[40] = d40
						ps54.OverlayValues[41] = d41
						ps54.OverlayValues[42] = d42
						ps54.OverlayValues[43] = d43
						ps54.OverlayValues[44] = d44
						ps54.OverlayValues[46] = d46
						ps54.OverlayValues[47] = d47
						ps54.OverlayValues[48] = d48
						ps54.OverlayValues[49] = d49
						ps54.OverlayValues[50] = d50
						ps54.OverlayValues[51] = d51
						ps54.OverlayValues[52] = d52
						return bbs[5].RenderPS(ps54)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d55 := ps.PhiValues[0]
							ctx.EnsureDesc(&d55)
							ctx.EmitStoreScmerToStack(d55, int32(bbs[2].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d52.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl6)
					ps56 := PhiState{General: true}
					ps56.OverlayValues = make([]JITValueDesc, 56)
					ps56.OverlayValues[1] = d1
					ps56.OverlayValues[2] = d2
					ps56.OverlayValues[3] = d3
					ps56.OverlayValues[4] = d4
					ps56.OverlayValues[5] = d5
					ps56.OverlayValues[6] = d6
					ps56.OverlayValues[7] = d7
					ps56.OverlayValues[8] = d8
					ps56.OverlayValues[9] = d9
					ps56.OverlayValues[10] = d10
					ps56.OverlayValues[11] = d11
					ps56.OverlayValues[12] = d12
					ps56.OverlayValues[13] = d13
					ps56.OverlayValues[14] = d14
					ps56.OverlayValues[15] = d15
					ps56.OverlayValues[18] = d18
					ps56.OverlayValues[21] = d21
					ps56.OverlayValues[40] = d40
					ps56.OverlayValues[41] = d41
					ps56.OverlayValues[42] = d42
					ps56.OverlayValues[43] = d43
					ps56.OverlayValues[44] = d44
					ps56.OverlayValues[46] = d46
					ps56.OverlayValues[47] = d47
					ps56.OverlayValues[48] = d48
					ps56.OverlayValues[49] = d49
					ps56.OverlayValues[50] = d50
					ps56.OverlayValues[51] = d51
					ps56.OverlayValues[52] = d52
					ps56.OverlayValues[55] = d55
					ps57 := PhiState{General: true}
					ps57.OverlayValues = make([]JITValueDesc, 56)
					ps57.OverlayValues[1] = d1
					ps57.OverlayValues[2] = d2
					ps57.OverlayValues[3] = d3
					ps57.OverlayValues[4] = d4
					ps57.OverlayValues[5] = d5
					ps57.OverlayValues[6] = d6
					ps57.OverlayValues[7] = d7
					ps57.OverlayValues[8] = d8
					ps57.OverlayValues[9] = d9
					ps57.OverlayValues[10] = d10
					ps57.OverlayValues[11] = d11
					ps57.OverlayValues[12] = d12
					ps57.OverlayValues[13] = d13
					ps57.OverlayValues[14] = d14
					ps57.OverlayValues[15] = d15
					ps57.OverlayValues[18] = d18
					ps57.OverlayValues[21] = d21
					ps57.OverlayValues[40] = d40
					ps57.OverlayValues[41] = d41
					ps57.OverlayValues[42] = d42
					ps57.OverlayValues[43] = d43
					ps57.OverlayValues[44] = d44
					ps57.OverlayValues[46] = d46
					ps57.OverlayValues[47] = d47
					ps57.OverlayValues[48] = d48
					ps57.OverlayValues[49] = d49
					ps57.OverlayValues[50] = d50
					ps57.OverlayValues[51] = d51
					ps57.OverlayValues[52] = d52
					ps57.OverlayValues[55] = d55
					snap58 := d1
					snap59 := d2
					snap60 := d3
					snap61 := d4
					snap62 := d5
					snap63 := d6
					snap64 := d7
					snap65 := d8
					snap66 := d9
					snap67 := d10
					snap68 := d11
					snap69 := d12
					snap70 := d13
					snap71 := d14
					snap72 := d15
					snap73 := d18
					snap74 := d21
					snap75 := d40
					snap76 := d41
					snap77 := d42
					snap78 := d43
					snap79 := d44
					snap80 := d46
					snap81 := d47
					snap82 := d48
					snap83 := d49
					snap84 := d50
					snap85 := d51
					snap86 := d52
					snap87 := d55
					alloc88 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps57)
					}
					ctx.RestoreAllocState(alloc88)
					d1 = snap58
					d2 = snap59
					d3 = snap60
					d4 = snap61
					d5 = snap62
					d6 = snap63
					d7 = snap64
					d8 = snap65
					d9 = snap66
					d10 = snap67
					d11 = snap68
					d12 = snap69
					d13 = snap70
					d14 = snap71
					d15 = snap72
					d18 = snap73
					d21 = snap74
					d40 = snap75
					d41 = snap76
					d42 = snap77
					d43 = snap78
					d44 = snap79
					d46 = snap80
					d47 = snap81
					d48 = snap82
					d49 = snap83
					d50 = snap84
					d51 = snap85
					d52 = snap86
					d55 = snap87
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps56)
					}
					return result
					ctx.FreeDesc(&d51)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}, int32(bbs[6].PhiBase)+int32(16))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}, int32(bbs[6].PhiBase)+int32(32))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(48))
					}
					ps89 := PhiState{General: ps.General}
					ps89.OverlayValues = make([]JITValueDesc, 56)
					ps89.OverlayValues[1] = d1
					ps89.OverlayValues[2] = d2
					ps89.OverlayValues[3] = d3
					ps89.OverlayValues[4] = d4
					ps89.OverlayValues[5] = d5
					ps89.OverlayValues[6] = d6
					ps89.OverlayValues[7] = d7
					ps89.OverlayValues[8] = d8
					ps89.OverlayValues[9] = d9
					ps89.OverlayValues[10] = d10
					ps89.OverlayValues[11] = d11
					ps89.OverlayValues[12] = d12
					ps89.OverlayValues[13] = d13
					ps89.OverlayValues[14] = d14
					ps89.OverlayValues[15] = d15
					ps89.OverlayValues[18] = d18
					ps89.OverlayValues[21] = d21
					ps89.OverlayValues[40] = d40
					ps89.OverlayValues[41] = d41
					ps89.OverlayValues[42] = d42
					ps89.OverlayValues[43] = d43
					ps89.OverlayValues[44] = d44
					ps89.OverlayValues[46] = d46
					ps89.OverlayValues[47] = d47
					ps89.OverlayValues[48] = d48
					ps89.OverlayValues[49] = d49
					ps89.OverlayValues[50] = d50
					ps89.OverlayValues[51] = d51
					ps89.OverlayValues[52] = d52
					ps89.OverlayValues[55] = d55
					ps89.PhiValues = make([]JITValueDesc, 4)
					d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps89.PhiValues[0] = d90
					d91 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps89.PhiValues[1] = d91
					d92 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps89.PhiValues[2] = d92
					d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps89.PhiValues[3] = d93
					if ps89.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps89)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d94 := ps.PhiValues[0]
							ctx.EnsureDesc(&d94)
							ctx.EmitStoreToStack(d94, int32(bbs[4].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						ctx.EmitMakeFloat(result, d2)
					} else {
						ctx.EmitMovToReg(result.Reg2, d2)
						d95 := JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d95)
						if d2.Loc == LocReg && d2.Reg != result.Reg2 {
							ctx.FreeReg(d2.Reg)
						}
					}
					result.Type = tagFloat
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[10].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[10].PhiBase)+int32(16))
					}
					ps96 := PhiState{General: ps.General}
					ps96.OverlayValues = make([]JITValueDesc, 96)
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
					ps96.OverlayValues[11] = d11
					ps96.OverlayValues[12] = d12
					ps96.OverlayValues[13] = d13
					ps96.OverlayValues[14] = d14
					ps96.OverlayValues[15] = d15
					ps96.OverlayValues[18] = d18
					ps96.OverlayValues[21] = d21
					ps96.OverlayValues[40] = d40
					ps96.OverlayValues[41] = d41
					ps96.OverlayValues[42] = d42
					ps96.OverlayValues[43] = d43
					ps96.OverlayValues[44] = d44
					ps96.OverlayValues[46] = d46
					ps96.OverlayValues[47] = d47
					ps96.OverlayValues[48] = d48
					ps96.OverlayValues[49] = d49
					ps96.OverlayValues[50] = d50
					ps96.OverlayValues[51] = d51
					ps96.OverlayValues[52] = d52
					ps96.OverlayValues[55] = d55
					ps96.OverlayValues[90] = d90
					ps96.OverlayValues[91] = d91
					ps96.OverlayValues[92] = d92
					ps96.OverlayValues[93] = d93
					ps96.OverlayValues[94] = d94
					ps96.OverlayValues[95] = d95
					ps96.PhiValues = make([]JITValueDesc, 2)
					d97 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps96.PhiValues[0] = d97
					d98 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps96.PhiValues[1] = d98
					if ps96.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps96)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d99 := ps.PhiValues[0]
							ctx.EnsureDesc(&d99)
							ctx.EmitStoreToStack(d99, int32(bbs[6].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d100 := ps.PhiValues[1]
							ctx.EnsureDesc(&d100)
							ctx.EmitStoreToStack(d100, int32(bbs[6].PhiBase)+int32(16))
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d101 := ps.PhiValues[2]
							ctx.EnsureDesc(&d101)
							ctx.EmitStoreToStack(d101, int32(bbs[6].PhiBase)+int32(32))
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d102 := ps.PhiValues[3]
							ctx.EnsureDesc(&d102)
							ctx.EmitStoreToStack(d102, int32(bbs[6].PhiBase)+int32(48))
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d4 = ps.PhiValues[1]
					}
					if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
						d5 = ps.PhiValues[2]
					}
					if !ps.General && len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
						d6 = ps.PhiValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d6)
					var d103 JITValueDesc
					if d10.SliceSizeKnown {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.KnownSliceLen))}
					} else if d10.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.StackOff))}
					} else if d10.Loc == LocStackTriple {
						d103 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d10.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d10)
						if d10.Loc == LocRegPair || d10.Loc == LocRegTriple {
							d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg2, ID: 0}
						} else if d10.Loc == LocReg {
							d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d103)
					var d104 JITValueDesc
					if d6.Loc == LocImm && d103.Loc == LocImm {
						d104 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() < d103.Imm.Int())}
					} else if d103.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d6.Reg)
						if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d6.Reg, int32(d103.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d103.Imm.Int()))
							ctx.EmitCmpInt64(d6.Reg, RegR11)
						}
						ctx.EmitSetcc(r1, CondSignedLess)
						d104 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d104)
					} else if d6.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d103.Reg)
						ctx.EmitSetcc(r2, CondSignedLess)
						d104 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d104)
					} else {
						r3 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpInt64(d6.Reg, d103.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d104 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d104)
					}
					ctx.FreeDesc(&d103)
					d105 = d104
					ctx.EnsureDesc(&d105)
					if d105.Loc != LocImm && d105.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d105.Loc == LocImm {
						if d105.Imm.Bool() {
							if ps.General {
							}
							ps106 := PhiState{General: ps.General}
							ps106.OverlayValues = make([]JITValueDesc, 106)
							ps106.OverlayValues[1] = d1
							ps106.OverlayValues[2] = d2
							ps106.OverlayValues[3] = d3
							ps106.OverlayValues[4] = d4
							ps106.OverlayValues[5] = d5
							ps106.OverlayValues[6] = d6
							ps106.OverlayValues[7] = d7
							ps106.OverlayValues[8] = d8
							ps106.OverlayValues[9] = d9
							ps106.OverlayValues[10] = d10
							ps106.OverlayValues[11] = d11
							ps106.OverlayValues[12] = d12
							ps106.OverlayValues[13] = d13
							ps106.OverlayValues[14] = d14
							ps106.OverlayValues[15] = d15
							ps106.OverlayValues[18] = d18
							ps106.OverlayValues[21] = d21
							ps106.OverlayValues[40] = d40
							ps106.OverlayValues[41] = d41
							ps106.OverlayValues[42] = d42
							ps106.OverlayValues[43] = d43
							ps106.OverlayValues[44] = d44
							ps106.OverlayValues[46] = d46
							ps106.OverlayValues[47] = d47
							ps106.OverlayValues[48] = d48
							ps106.OverlayValues[49] = d49
							ps106.OverlayValues[50] = d50
							ps106.OverlayValues[51] = d51
							ps106.OverlayValues[52] = d52
							ps106.OverlayValues[55] = d55
							ps106.OverlayValues[90] = d90
							ps106.OverlayValues[91] = d91
							ps106.OverlayValues[92] = d92
							ps106.OverlayValues[93] = d93
							ps106.OverlayValues[94] = d94
							ps106.OverlayValues[95] = d95
							ps106.OverlayValues[97] = d97
							ps106.OverlayValues[98] = d98
							ps106.OverlayValues[99] = d99
							ps106.OverlayValues[100] = d100
							ps106.OverlayValues[101] = d101
							ps106.OverlayValues[102] = d102
							ps106.OverlayValues[103] = d103
							ps106.OverlayValues[104] = d104
							ps106.OverlayValues[105] = d105
							return bbs[9].RenderPS(ps106)
						}
						if ps.General {
						}
						ps107 := PhiState{General: ps.General}
						ps107.OverlayValues = make([]JITValueDesc, 106)
						ps107.OverlayValues[1] = d1
						ps107.OverlayValues[2] = d2
						ps107.OverlayValues[3] = d3
						ps107.OverlayValues[4] = d4
						ps107.OverlayValues[5] = d5
						ps107.OverlayValues[6] = d6
						ps107.OverlayValues[7] = d7
						ps107.OverlayValues[8] = d8
						ps107.OverlayValues[9] = d9
						ps107.OverlayValues[10] = d10
						ps107.OverlayValues[11] = d11
						ps107.OverlayValues[12] = d12
						ps107.OverlayValues[13] = d13
						ps107.OverlayValues[14] = d14
						ps107.OverlayValues[15] = d15
						ps107.OverlayValues[18] = d18
						ps107.OverlayValues[21] = d21
						ps107.OverlayValues[40] = d40
						ps107.OverlayValues[41] = d41
						ps107.OverlayValues[42] = d42
						ps107.OverlayValues[43] = d43
						ps107.OverlayValues[44] = d44
						ps107.OverlayValues[46] = d46
						ps107.OverlayValues[47] = d47
						ps107.OverlayValues[48] = d48
						ps107.OverlayValues[49] = d49
						ps107.OverlayValues[50] = d50
						ps107.OverlayValues[51] = d51
						ps107.OverlayValues[52] = d52
						ps107.OverlayValues[55] = d55
						ps107.OverlayValues[90] = d90
						ps107.OverlayValues[91] = d91
						ps107.OverlayValues[92] = d92
						ps107.OverlayValues[93] = d93
						ps107.OverlayValues[94] = d94
						ps107.OverlayValues[95] = d95
						ps107.OverlayValues[97] = d97
						ps107.OverlayValues[98] = d98
						ps107.OverlayValues[99] = d99
						ps107.OverlayValues[100] = d100
						ps107.OverlayValues[101] = d101
						ps107.OverlayValues[102] = d102
						ps107.OverlayValues[103] = d103
						ps107.OverlayValues[104] = d104
						ps107.OverlayValues[105] = d105
						return bbs[8].RenderPS(ps107)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d108 := ps.PhiValues[0]
							ctx.EnsureDesc(&d108)
							ctx.EmitStoreToStack(d108, int32(bbs[6].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d109 := ps.PhiValues[1]
							ctx.EnsureDesc(&d109)
							ctx.EmitStoreToStack(d109, int32(bbs[6].PhiBase)+int32(16))
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d110 := ps.PhiValues[2]
							ctx.EnsureDesc(&d110)
							ctx.EmitStoreToStack(d110, int32(bbs[6].PhiBase)+int32(32))
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d111 := ps.PhiValues[3]
							ctx.EnsureDesc(&d111)
							ctx.EmitStoreToStack(d111, int32(bbs[6].PhiBase)+int32(48))
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d105.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl9)
					ps112 := PhiState{General: true}
					ps112.OverlayValues = make([]JITValueDesc, 112)
					ps112.OverlayValues[1] = d1
					ps112.OverlayValues[2] = d2
					ps112.OverlayValues[3] = d3
					ps112.OverlayValues[4] = d4
					ps112.OverlayValues[5] = d5
					ps112.OverlayValues[6] = d6
					ps112.OverlayValues[7] = d7
					ps112.OverlayValues[8] = d8
					ps112.OverlayValues[9] = d9
					ps112.OverlayValues[10] = d10
					ps112.OverlayValues[11] = d11
					ps112.OverlayValues[12] = d12
					ps112.OverlayValues[13] = d13
					ps112.OverlayValues[14] = d14
					ps112.OverlayValues[15] = d15
					ps112.OverlayValues[18] = d18
					ps112.OverlayValues[21] = d21
					ps112.OverlayValues[40] = d40
					ps112.OverlayValues[41] = d41
					ps112.OverlayValues[42] = d42
					ps112.OverlayValues[43] = d43
					ps112.OverlayValues[44] = d44
					ps112.OverlayValues[46] = d46
					ps112.OverlayValues[47] = d47
					ps112.OverlayValues[48] = d48
					ps112.OverlayValues[49] = d49
					ps112.OverlayValues[50] = d50
					ps112.OverlayValues[51] = d51
					ps112.OverlayValues[52] = d52
					ps112.OverlayValues[55] = d55
					ps112.OverlayValues[90] = d90
					ps112.OverlayValues[91] = d91
					ps112.OverlayValues[92] = d92
					ps112.OverlayValues[93] = d93
					ps112.OverlayValues[94] = d94
					ps112.OverlayValues[95] = d95
					ps112.OverlayValues[97] = d97
					ps112.OverlayValues[98] = d98
					ps112.OverlayValues[99] = d99
					ps112.OverlayValues[100] = d100
					ps112.OverlayValues[101] = d101
					ps112.OverlayValues[102] = d102
					ps112.OverlayValues[103] = d103
					ps112.OverlayValues[104] = d104
					ps112.OverlayValues[105] = d105
					ps112.OverlayValues[108] = d108
					ps112.OverlayValues[109] = d109
					ps112.OverlayValues[110] = d110
					ps112.OverlayValues[111] = d111
					ps113 := PhiState{General: true}
					ps113.OverlayValues = make([]JITValueDesc, 112)
					ps113.OverlayValues[1] = d1
					ps113.OverlayValues[2] = d2
					ps113.OverlayValues[3] = d3
					ps113.OverlayValues[4] = d4
					ps113.OverlayValues[5] = d5
					ps113.OverlayValues[6] = d6
					ps113.OverlayValues[7] = d7
					ps113.OverlayValues[8] = d8
					ps113.OverlayValues[9] = d9
					ps113.OverlayValues[10] = d10
					ps113.OverlayValues[11] = d11
					ps113.OverlayValues[12] = d12
					ps113.OverlayValues[13] = d13
					ps113.OverlayValues[14] = d14
					ps113.OverlayValues[15] = d15
					ps113.OverlayValues[18] = d18
					ps113.OverlayValues[21] = d21
					ps113.OverlayValues[40] = d40
					ps113.OverlayValues[41] = d41
					ps113.OverlayValues[42] = d42
					ps113.OverlayValues[43] = d43
					ps113.OverlayValues[44] = d44
					ps113.OverlayValues[46] = d46
					ps113.OverlayValues[47] = d47
					ps113.OverlayValues[48] = d48
					ps113.OverlayValues[49] = d49
					ps113.OverlayValues[50] = d50
					ps113.OverlayValues[51] = d51
					ps113.OverlayValues[52] = d52
					ps113.OverlayValues[55] = d55
					ps113.OverlayValues[90] = d90
					ps113.OverlayValues[91] = d91
					ps113.OverlayValues[92] = d92
					ps113.OverlayValues[93] = d93
					ps113.OverlayValues[94] = d94
					ps113.OverlayValues[95] = d95
					ps113.OverlayValues[97] = d97
					ps113.OverlayValues[98] = d98
					ps113.OverlayValues[99] = d99
					ps113.OverlayValues[100] = d100
					ps113.OverlayValues[101] = d101
					ps113.OverlayValues[102] = d102
					ps113.OverlayValues[103] = d103
					ps113.OverlayValues[104] = d104
					ps113.OverlayValues[105] = d105
					ps113.OverlayValues[108] = d108
					ps113.OverlayValues[109] = d109
					ps113.OverlayValues[110] = d110
					ps113.OverlayValues[111] = d111
					snap114 := d1
					snap115 := d2
					snap116 := d3
					snap117 := d4
					snap118 := d5
					snap119 := d6
					snap120 := d7
					snap121 := d8
					snap122 := d9
					snap123 := d10
					snap124 := d11
					snap125 := d12
					snap126 := d13
					snap127 := d14
					snap128 := d15
					snap129 := d18
					snap130 := d21
					snap131 := d40
					snap132 := d41
					snap133 := d42
					snap134 := d43
					snap135 := d44
					snap136 := d46
					snap137 := d47
					snap138 := d48
					snap139 := d49
					snap140 := d50
					snap141 := d51
					snap142 := d52
					snap143 := d55
					snap144 := d90
					snap145 := d91
					snap146 := d92
					snap147 := d93
					snap148 := d94
					snap149 := d95
					snap150 := d97
					snap151 := d98
					snap152 := d99
					snap153 := d100
					snap154 := d101
					snap155 := d102
					snap156 := d103
					snap157 := d104
					snap158 := d105
					snap159 := d108
					snap160 := d109
					snap161 := d110
					snap162 := d111
					alloc163 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps113)
					}
					ctx.RestoreAllocState(alloc163)
					d1 = snap114
					d2 = snap115
					d3 = snap116
					d4 = snap117
					d5 = snap118
					d6 = snap119
					d7 = snap120
					d8 = snap121
					d9 = snap122
					d10 = snap123
					d11 = snap124
					d12 = snap125
					d13 = snap126
					d14 = snap127
					d15 = snap128
					d18 = snap129
					d21 = snap130
					d40 = snap131
					d41 = snap132
					d42 = snap133
					d43 = snap134
					d44 = snap135
					d46 = snap136
					d47 = snap137
					d48 = snap138
					d49 = snap139
					d50 = snap140
					d51 = snap141
					d52 = snap142
					d55 = snap143
					d90 = snap144
					d91 = snap145
					d92 = snap146
					d93 = snap147
					d94 = snap148
					d95 = snap149
					d97 = snap150
					d98 = snap151
					d99 = snap152
					d100 = snap153
					d101 = snap154
					d102 = snap155
					d103 = snap156
					d104 = snap157
					d105 = snap158
					d108 = snap159
					d109 = snap160
					d110 = snap161
					d111 = snap162
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps112)
					}
					return result
					ctx.FreeDesc(&d104)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[7].VisitCount >= 0 {
							ps.General = true
							return bbs[7].RenderPS(ps)
						}
					}
					bbs[7].VisitCount++
					if ps.General {
						if bbs[7].Rendered {
							ctx.EmitJmp(lbl8)
							return result
						}
						bbs[7].Rendered = true
						bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_7 = bbs[7].Address
						ctx.MarkLabel(lbl8)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					d165 = ctx.EmitSliceElementAddress(&d10, &d6, 16)
					ctx.EnsureDesc(&d165)
					r4 := ctx.AllocRegExcept(d165.Reg)
					ctx.EmitMovRegMem(r4, d165.Reg, 8)
					ctx.EmitMovRegMem(d165.Reg, d165.Reg, 0)
					d164 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d165.Reg, Reg2: r4}
					ctx.BindReg(d165.Reg, &d164)
					ctx.BindReg(r4, &d164)
					ctx.EnsureDesc(&d164)
					d166 = d164
					_ = d166
					ctx.StabilizeDescForControlFlow(&d166)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d167 JITValueDesc
					if d166.Loc == LocImm {
						d167 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d166.Imm.Float())}
					} else if d166.Type == tagFloat && d166.Loc == LocReg {
						d167 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d166.Reg}
						ctx.BindReg(d166.Reg, &d167)
						ctx.BindReg(d166.Reg, &d167)
					} else if d166.Type == tagFloat && d166.Loc == LocRegPair {
						ctx.FreeReg(d166.Reg)
						d167 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d166.Reg2}
						ctx.BindReg(d166.Reg2, &d167)
						ctx.BindReg(d166.Reg2, &d167)
					} else {
						d167 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d166}, 1)
						d167.Type = tagFloat
						ctx.BindReg(d167.Reg, &d167)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d167)
					ctx.FreeDesc(&d164)
					ctx.EnsureDesc(&d6)
					d169 = ctx.EmitSliceElementAddress(&d12, &d6, 16)
					ctx.EnsureDesc(&d169)
					r5 := ctx.AllocRegExcept(d169.Reg)
					ctx.EmitMovRegMem(r5, d169.Reg, 8)
					ctx.EmitMovRegMem(d169.Reg, d169.Reg, 0)
					d168 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d169.Reg, Reg2: r5}
					ctx.BindReg(d169.Reg, &d168)
					ctx.BindReg(r5, &d168)
					ctx.EnsureDesc(&d168)
					d170 = d168
					_ = d170
					ctx.StabilizeDescForControlFlow(&d170)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d171 JITValueDesc
					if d170.Loc == LocImm {
						d171 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d170.Imm.Float())}
					} else if d170.Type == tagFloat && d170.Loc == LocReg {
						d171 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d170.Reg}
						ctx.BindReg(d170.Reg, &d171)
						ctx.BindReg(d170.Reg, &d171)
					} else if d170.Type == tagFloat && d170.Loc == LocRegPair {
						ctx.FreeReg(d170.Reg)
						d171 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d170.Reg2}
						ctx.BindReg(d170.Reg2, &d171)
						ctx.BindReg(d170.Reg2, &d171)
					} else {
						d171 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d170}, 1)
						d171.Type = tagFloat
						ctx.BindReg(d171.Reg, &d171)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d171)
					ctx.FreeDesc(&d168)
					ctx.EnsureDesc(&d167)
					ctx.EnsureDesc(&d167)
					ctx.EnsureDesc(&d167)
					ctx.EnsureDesc(&d167)
					var d172 JITValueDesc
					if d167.Loc == LocImm {
						d172 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d167.Imm.Float() * d167.Imm.Float())}
					} else if d167.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d167.Reg)
						_, xBits := d167.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d167.Reg)
						d172 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d172)
					} else if d167.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d167.Reg)
						ctx.EmitMovRegReg(scratch, d167.Reg)
						_, yBits := d167.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d172 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d172)
					} else {
						r6 := ctx.AllocRegExcept(d167.Reg, d167.Reg)
						ctx.EmitMovRegReg(r6, d167.Reg)
						ctx.EmitMulFloat64(r6, d167.Reg)
						d172 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
						ctx.BindReg(r6, &d172)
					}
					if d172.Loc == LocReg && d167.Loc == LocReg && d172.Reg == d167.Reg {
						ctx.TransferReg(d167.Reg)
						d167.Loc = LocNone
					}
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d172)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d172)
					var d173 JITValueDesc
					if d4.Loc == LocImm && d172.Loc == LocImm {
						d173 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float() + d172.Imm.Float())}
					} else if d4.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d172.Reg)
						_, xBits := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d172.Reg)
						d173 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d173)
					} else if d172.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						_, yBits := d172.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d173 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d173)
					} else {
						r7 := ctx.AllocRegExcept(d4.Reg, d172.Reg)
						ctx.EmitMovRegReg(r7, d4.Reg)
						ctx.EmitAddFloat64(r7, d172.Reg)
						d173 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r7}
						ctx.BindReg(r7, &d173)
					}
					if d173.Loc == LocReg && d4.Loc == LocReg && d173.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d173)
					ctx.EmitStoreToStack(d173, int32(bbs[6].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d173)
					ctx.FreeDesc(&d172)
					ctx.EnsureDesc(&d171)
					ctx.EnsureDesc(&d171)
					ctx.EnsureDesc(&d171)
					ctx.EnsureDesc(&d171)
					var d174 JITValueDesc
					if d171.Loc == LocImm {
						d174 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d171.Imm.Float() * d171.Imm.Float())}
					} else if d171.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d171.Reg)
						_, xBits := d171.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d171.Reg)
						d174 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d174)
					} else if d171.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d171.Reg)
						ctx.EmitMovRegReg(scratch, d171.Reg)
						_, yBits := d171.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d174 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d174)
					} else {
						r8 := ctx.AllocRegExcept(d171.Reg, d171.Reg)
						ctx.EmitMovRegReg(r8, d171.Reg)
						ctx.EmitMulFloat64(r8, d171.Reg)
						d174 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r8}
						ctx.BindReg(r8, &d174)
					}
					if d174.Loc == LocReg && d171.Loc == LocReg && d174.Reg == d171.Reg {
						ctx.TransferReg(d171.Reg)
						d171.Loc = LocNone
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d174)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d174)
					var d175 JITValueDesc
					if d5.Loc == LocImm && d174.Loc == LocImm {
						d175 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d5.Imm.Float() + d174.Imm.Float())}
					} else if d5.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d174.Reg)
						_, xBits := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d174.Reg)
						d175 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d175)
					} else if d174.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(scratch, d5.Reg)
						_, yBits := d174.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d175 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d175)
					} else {
						r9 := ctx.AllocRegExcept(d5.Reg, d174.Reg)
						ctx.EmitMovRegReg(r9, d5.Reg)
						ctx.EmitAddFloat64(r9, d174.Reg)
						d175 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r9}
						ctx.BindReg(r9, &d175)
					}
					if d175.Loc == LocReg && d5.Loc == LocReg && d175.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					ctx.EnsureDesc(&d175)
					ctx.EmitStoreToStack(d175, int32(bbs[6].PhiBase)+int32(32))
					ctx.StabilizeDescForControlFlow(&d175)
					ctx.FreeDesc(&d174)
					ctx.EnsureDesc(&d167)
					ctx.EnsureDesc(&d171)
					ctx.EnsureDesc(&d167)
					ctx.EnsureDesc(&d171)
					var d176 JITValueDesc
					if d167.Loc == LocImm && d171.Loc == LocImm {
						d176 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d167.Imm.Float() * d171.Imm.Float())}
					} else if d167.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d171.Reg)
						_, xBits := d167.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d171.Reg)
						d176 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d176)
					} else if d171.Loc == LocImm {
						_, yBits := d171.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d167.Reg, RegR11)
						d176 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d167.Reg}
						ctx.BindReg(d167.Reg, &d176)
					} else {
						ctx.EmitMulFloat64(d167.Reg, d171.Reg)
						d176 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d167.Reg}
						ctx.BindReg(d167.Reg, &d176)
					}
					if d176.Loc == LocReg && d167.Loc == LocReg && d176.Reg == d167.Reg {
						ctx.TransferReg(d167.Reg)
						d167.Loc = LocNone
					}
					ctx.FreeDesc(&d167)
					ctx.FreeDesc(&d171)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d176)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d176)
					var d177 JITValueDesc
					if d3.Loc == LocImm && d176.Loc == LocImm {
						d177 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float() + d176.Imm.Float())}
					} else if d3.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d176.Reg)
						_, xBits := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d176.Reg)
						d177 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d177)
					} else if d176.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						_, yBits := d176.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d177 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d177)
					} else {
						r10 := ctx.AllocRegExcept(d3.Reg, d176.Reg)
						ctx.EmitMovRegReg(r10, d3.Reg)
						ctx.EmitAddFloat64(r10, d176.Reg)
						d177 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r10}
						ctx.BindReg(r10, &d177)
					}
					if d177.Loc == LocReg && d3.Loc == LocReg && d177.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d177)
					ctx.EmitStoreToStack(d177, int32(bbs[6].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d177)
					ctx.FreeDesc(&d176)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					var d178 JITValueDesc
					if d6.Loc == LocImm {
						d178 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitMovRegReg(scratch, d6.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d178 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d178)
					}
					if d178.Loc == LocReg && d6.Loc == LocReg && d178.Reg == d6.Reg {
						ctx.TransferReg(d6.Reg)
						d6.Loc = LocNone
					}
					ctx.EnsureDesc(&d178)
					ctx.EmitStoreToStack(d178, int32(bbs[6].PhiBase)+int32(48))
					ctx.StabilizeDescForControlFlow(&d178)
					if ps.General {
					}
					ps179 := PhiState{General: ps.General}
					ps179.OverlayValues = make([]JITValueDesc, 179)
					ps179.OverlayValues[1] = d1
					ps179.OverlayValues[2] = d2
					ps179.OverlayValues[3] = d3
					ps179.OverlayValues[4] = d4
					ps179.OverlayValues[5] = d5
					ps179.OverlayValues[6] = d6
					ps179.OverlayValues[7] = d7
					ps179.OverlayValues[8] = d8
					ps179.OverlayValues[9] = d9
					ps179.OverlayValues[10] = d10
					ps179.OverlayValues[11] = d11
					ps179.OverlayValues[12] = d12
					ps179.OverlayValues[13] = d13
					ps179.OverlayValues[14] = d14
					ps179.OverlayValues[15] = d15
					ps179.OverlayValues[18] = d18
					ps179.OverlayValues[21] = d21
					ps179.OverlayValues[40] = d40
					ps179.OverlayValues[41] = d41
					ps179.OverlayValues[42] = d42
					ps179.OverlayValues[43] = d43
					ps179.OverlayValues[44] = d44
					ps179.OverlayValues[46] = d46
					ps179.OverlayValues[47] = d47
					ps179.OverlayValues[48] = d48
					ps179.OverlayValues[49] = d49
					ps179.OverlayValues[50] = d50
					ps179.OverlayValues[51] = d51
					ps179.OverlayValues[52] = d52
					ps179.OverlayValues[55] = d55
					ps179.OverlayValues[90] = d90
					ps179.OverlayValues[91] = d91
					ps179.OverlayValues[92] = d92
					ps179.OverlayValues[93] = d93
					ps179.OverlayValues[94] = d94
					ps179.OverlayValues[95] = d95
					ps179.OverlayValues[97] = d97
					ps179.OverlayValues[98] = d98
					ps179.OverlayValues[99] = d99
					ps179.OverlayValues[100] = d100
					ps179.OverlayValues[101] = d101
					ps179.OverlayValues[102] = d102
					ps179.OverlayValues[103] = d103
					ps179.OverlayValues[104] = d104
					ps179.OverlayValues[105] = d105
					ps179.OverlayValues[108] = d108
					ps179.OverlayValues[109] = d109
					ps179.OverlayValues[110] = d110
					ps179.OverlayValues[111] = d111
					ps179.OverlayValues[164] = d164
					ps179.OverlayValues[165] = d165
					ps179.OverlayValues[166] = d166
					ps179.OverlayValues[167] = d167
					ps179.OverlayValues[168] = d168
					ps179.OverlayValues[169] = d169
					ps179.OverlayValues[170] = d170
					ps179.OverlayValues[171] = d171
					ps179.OverlayValues[172] = d172
					ps179.OverlayValues[173] = d173
					ps179.OverlayValues[174] = d174
					ps179.OverlayValues[175] = d175
					ps179.OverlayValues[176] = d176
					ps179.OverlayValues[177] = d177
					ps179.OverlayValues[178] = d178
					ps179.PhiValues = make([]JITValueDesc, 4)
					if ps179.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps179)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != LocNone {
						d169 = ps.OverlayValues[169]
					}
					if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != LocNone {
						d170 = ps.OverlayValues[170]
					}
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d5)
					var d180 JITValueDesc
					if d4.Loc == LocImm && d5.Loc == LocImm {
						d180 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float() * d5.Imm.Float())}
					} else if d4.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						_, xBits := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d5.Reg)
						d180 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d180)
					} else if d5.Loc == LocImm {
						_, yBits := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d4.Reg, RegR11)
						d180 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d180)
					} else {
						ctx.EmitMulFloat64(d4.Reg, d5.Reg)
						d180 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d4.Reg}
						ctx.BindReg(d4.Reg, &d180)
					}
					if d180.Loc == LocReg && d4.Loc == LocReg && d180.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.FreeDesc(&d4)
					ctx.FreeDesc(&d5)
					ctx.EnsureDesc(&d180)
					var d181 JITValueDesc
					if d180.Loc == LocImm {
						d181 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d180.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d180)
						var d182 JITValueDesc
						if d180.Loc == LocRegPair {
							ctx.FreeReg(d180.Reg)
							d182 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d180.Reg2}
							ctx.BindReg(d180.Reg2, &d182)
							ctx.BindReg(d180.Reg2, &d182)
						} else {
							d182 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d180.Reg}
							ctx.BindReg(d180.Reg, &d182)
							ctx.BindReg(d180.Reg, &d182)
						}
						d181 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d182}, 1)
						d181.Type = tagFloat
						ctx.BindReg(d181.Reg, &d181)
					}
					ctx.FreeDesc(&d180)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d181)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d181)
					var d183 JITValueDesc
					if d3.Loc == LocImm && d181.Loc == LocImm {
						d183 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float() / d181.Imm.Float())}
					} else if d3.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d181.Reg)
						_, xBits := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d181.Reg)
						d183 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d183)
					} else if d181.Loc == LocImm {
						_, yBits := d181.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitDivFloat64(d3.Reg, RegR11)
						d183 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d3.Reg}
						ctx.BindReg(d3.Reg, &d183)
					} else {
						ctx.EmitDivFloat64(d3.Reg, d181.Reg)
						d183 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d3.Reg}
						ctx.BindReg(d3.Reg, &d183)
					}
					if d183.Loc == LocReg && d3.Loc == LocReg && d183.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d183)
					ctx.EmitStoreToStack(d183, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d183)
					ctx.FreeDesc(&d3)
					ctx.FreeDesc(&d181)
					if ps.General {
					}
					ps184 := PhiState{General: ps.General}
					ps184.OverlayValues = make([]JITValueDesc, 184)
					ps184.OverlayValues[1] = d1
					ps184.OverlayValues[2] = d2
					ps184.OverlayValues[3] = d3
					ps184.OverlayValues[4] = d4
					ps184.OverlayValues[5] = d5
					ps184.OverlayValues[6] = d6
					ps184.OverlayValues[7] = d7
					ps184.OverlayValues[8] = d8
					ps184.OverlayValues[9] = d9
					ps184.OverlayValues[10] = d10
					ps184.OverlayValues[11] = d11
					ps184.OverlayValues[12] = d12
					ps184.OverlayValues[13] = d13
					ps184.OverlayValues[14] = d14
					ps184.OverlayValues[15] = d15
					ps184.OverlayValues[18] = d18
					ps184.OverlayValues[21] = d21
					ps184.OverlayValues[40] = d40
					ps184.OverlayValues[41] = d41
					ps184.OverlayValues[42] = d42
					ps184.OverlayValues[43] = d43
					ps184.OverlayValues[44] = d44
					ps184.OverlayValues[46] = d46
					ps184.OverlayValues[47] = d47
					ps184.OverlayValues[48] = d48
					ps184.OverlayValues[49] = d49
					ps184.OverlayValues[50] = d50
					ps184.OverlayValues[51] = d51
					ps184.OverlayValues[52] = d52
					ps184.OverlayValues[55] = d55
					ps184.OverlayValues[90] = d90
					ps184.OverlayValues[91] = d91
					ps184.OverlayValues[92] = d92
					ps184.OverlayValues[93] = d93
					ps184.OverlayValues[94] = d94
					ps184.OverlayValues[95] = d95
					ps184.OverlayValues[97] = d97
					ps184.OverlayValues[98] = d98
					ps184.OverlayValues[99] = d99
					ps184.OverlayValues[100] = d100
					ps184.OverlayValues[101] = d101
					ps184.OverlayValues[102] = d102
					ps184.OverlayValues[103] = d103
					ps184.OverlayValues[104] = d104
					ps184.OverlayValues[105] = d105
					ps184.OverlayValues[108] = d108
					ps184.OverlayValues[109] = d109
					ps184.OverlayValues[110] = d110
					ps184.OverlayValues[111] = d111
					ps184.OverlayValues[164] = d164
					ps184.OverlayValues[165] = d165
					ps184.OverlayValues[166] = d166
					ps184.OverlayValues[167] = d167
					ps184.OverlayValues[168] = d168
					ps184.OverlayValues[169] = d169
					ps184.OverlayValues[170] = d170
					ps184.OverlayValues[171] = d171
					ps184.OverlayValues[172] = d172
					ps184.OverlayValues[173] = d173
					ps184.OverlayValues[174] = d174
					ps184.OverlayValues[175] = d175
					ps184.OverlayValues[176] = d176
					ps184.OverlayValues[177] = d177
					ps184.OverlayValues[178] = d178
					ps184.OverlayValues[180] = d180
					ps184.OverlayValues[181] = d181
					ps184.OverlayValues[182] = d182
					ps184.OverlayValues[183] = d183
					ps184.PhiValues = make([]JITValueDesc, 1)
					if ps184.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps184)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != LocNone {
						d169 = ps.OverlayValues[169]
					}
					if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != LocNone {
						d170 = ps.OverlayValues[170]
					}
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					ctx.ReclaimUntrackedRegs()
					var d185 JITValueDesc
					if d12.SliceSizeKnown {
						d185 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.KnownSliceLen))}
					} else if d12.Loc == LocImm {
						d185 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.StackOff))}
					} else if d12.Loc == LocStackTriple {
						d185 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d12.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d12)
						if d12.Loc == LocRegPair || d12.Loc == LocRegTriple {
							d185 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg2, ID: 0}
						} else if d12.Loc == LocReg {
							d185 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d185)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d185)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d185)
					var d186 JITValueDesc
					if d6.Loc == LocImm && d185.Loc == LocImm {
						d186 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() < d185.Imm.Int())}
					} else if d185.Loc == LocImm {
						r11 := ctx.AllocReg()
						if d185.Imm.Int() >= -2147483648 && d185.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d6.Reg, int32(d185.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d185.Imm.Int()))
							ctx.EmitCmpInt64(d6.Reg, RegR11)
						}
						ctx.EmitSetcc(r11, CondSignedLess)
						d186 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
						ctx.BindReg(r11, &d186)
					} else if d6.Loc == LocImm {
						r12 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d185.Reg)
						ctx.EmitSetcc(r12, CondSignedLess)
						d186 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
						ctx.BindReg(r12, &d186)
					} else {
						r13 := ctx.AllocReg()
						ctx.EmitCmpInt64(d6.Reg, d185.Reg)
						ctx.EmitSetcc(r13, CondSignedLess)
						d186 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
						ctx.BindReg(r13, &d186)
					}
					ctx.FreeDesc(&d6)
					ctx.FreeDesc(&d185)
					d187 = d186
					ctx.EnsureDesc(&d187)
					if d187.Loc != LocImm && d187.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d187.Loc == LocImm {
						if d187.Imm.Bool() {
							if ps.General {
							}
							ps188 := PhiState{General: ps.General}
							ps188.OverlayValues = make([]JITValueDesc, 188)
							ps188.OverlayValues[1] = d1
							ps188.OverlayValues[2] = d2
							ps188.OverlayValues[3] = d3
							ps188.OverlayValues[4] = d4
							ps188.OverlayValues[5] = d5
							ps188.OverlayValues[6] = d6
							ps188.OverlayValues[7] = d7
							ps188.OverlayValues[8] = d8
							ps188.OverlayValues[9] = d9
							ps188.OverlayValues[10] = d10
							ps188.OverlayValues[11] = d11
							ps188.OverlayValues[12] = d12
							ps188.OverlayValues[13] = d13
							ps188.OverlayValues[14] = d14
							ps188.OverlayValues[15] = d15
							ps188.OverlayValues[18] = d18
							ps188.OverlayValues[21] = d21
							ps188.OverlayValues[40] = d40
							ps188.OverlayValues[41] = d41
							ps188.OverlayValues[42] = d42
							ps188.OverlayValues[43] = d43
							ps188.OverlayValues[44] = d44
							ps188.OverlayValues[46] = d46
							ps188.OverlayValues[47] = d47
							ps188.OverlayValues[48] = d48
							ps188.OverlayValues[49] = d49
							ps188.OverlayValues[50] = d50
							ps188.OverlayValues[51] = d51
							ps188.OverlayValues[52] = d52
							ps188.OverlayValues[55] = d55
							ps188.OverlayValues[90] = d90
							ps188.OverlayValues[91] = d91
							ps188.OverlayValues[92] = d92
							ps188.OverlayValues[93] = d93
							ps188.OverlayValues[94] = d94
							ps188.OverlayValues[95] = d95
							ps188.OverlayValues[97] = d97
							ps188.OverlayValues[98] = d98
							ps188.OverlayValues[99] = d99
							ps188.OverlayValues[100] = d100
							ps188.OverlayValues[101] = d101
							ps188.OverlayValues[102] = d102
							ps188.OverlayValues[103] = d103
							ps188.OverlayValues[104] = d104
							ps188.OverlayValues[105] = d105
							ps188.OverlayValues[108] = d108
							ps188.OverlayValues[109] = d109
							ps188.OverlayValues[110] = d110
							ps188.OverlayValues[111] = d111
							ps188.OverlayValues[164] = d164
							ps188.OverlayValues[165] = d165
							ps188.OverlayValues[166] = d166
							ps188.OverlayValues[167] = d167
							ps188.OverlayValues[168] = d168
							ps188.OverlayValues[169] = d169
							ps188.OverlayValues[170] = d170
							ps188.OverlayValues[171] = d171
							ps188.OverlayValues[172] = d172
							ps188.OverlayValues[173] = d173
							ps188.OverlayValues[174] = d174
							ps188.OverlayValues[175] = d175
							ps188.OverlayValues[176] = d176
							ps188.OverlayValues[177] = d177
							ps188.OverlayValues[178] = d178
							ps188.OverlayValues[180] = d180
							ps188.OverlayValues[181] = d181
							ps188.OverlayValues[182] = d182
							ps188.OverlayValues[183] = d183
							ps188.OverlayValues[185] = d185
							ps188.OverlayValues[186] = d186
							ps188.OverlayValues[187] = d187
							return bbs[7].RenderPS(ps188)
						}
						if ps.General {
						}
						ps189 := PhiState{General: ps.General}
						ps189.OverlayValues = make([]JITValueDesc, 188)
						ps189.OverlayValues[1] = d1
						ps189.OverlayValues[2] = d2
						ps189.OverlayValues[3] = d3
						ps189.OverlayValues[4] = d4
						ps189.OverlayValues[5] = d5
						ps189.OverlayValues[6] = d6
						ps189.OverlayValues[7] = d7
						ps189.OverlayValues[8] = d8
						ps189.OverlayValues[9] = d9
						ps189.OverlayValues[10] = d10
						ps189.OverlayValues[11] = d11
						ps189.OverlayValues[12] = d12
						ps189.OverlayValues[13] = d13
						ps189.OverlayValues[14] = d14
						ps189.OverlayValues[15] = d15
						ps189.OverlayValues[18] = d18
						ps189.OverlayValues[21] = d21
						ps189.OverlayValues[40] = d40
						ps189.OverlayValues[41] = d41
						ps189.OverlayValues[42] = d42
						ps189.OverlayValues[43] = d43
						ps189.OverlayValues[44] = d44
						ps189.OverlayValues[46] = d46
						ps189.OverlayValues[47] = d47
						ps189.OverlayValues[48] = d48
						ps189.OverlayValues[49] = d49
						ps189.OverlayValues[50] = d50
						ps189.OverlayValues[51] = d51
						ps189.OverlayValues[52] = d52
						ps189.OverlayValues[55] = d55
						ps189.OverlayValues[90] = d90
						ps189.OverlayValues[91] = d91
						ps189.OverlayValues[92] = d92
						ps189.OverlayValues[93] = d93
						ps189.OverlayValues[94] = d94
						ps189.OverlayValues[95] = d95
						ps189.OverlayValues[97] = d97
						ps189.OverlayValues[98] = d98
						ps189.OverlayValues[99] = d99
						ps189.OverlayValues[100] = d100
						ps189.OverlayValues[101] = d101
						ps189.OverlayValues[102] = d102
						ps189.OverlayValues[103] = d103
						ps189.OverlayValues[104] = d104
						ps189.OverlayValues[105] = d105
						ps189.OverlayValues[108] = d108
						ps189.OverlayValues[109] = d109
						ps189.OverlayValues[110] = d110
						ps189.OverlayValues[111] = d111
						ps189.OverlayValues[164] = d164
						ps189.OverlayValues[165] = d165
						ps189.OverlayValues[166] = d166
						ps189.OverlayValues[167] = d167
						ps189.OverlayValues[168] = d168
						ps189.OverlayValues[169] = d169
						ps189.OverlayValues[170] = d170
						ps189.OverlayValues[171] = d171
						ps189.OverlayValues[172] = d172
						ps189.OverlayValues[173] = d173
						ps189.OverlayValues[174] = d174
						ps189.OverlayValues[175] = d175
						ps189.OverlayValues[176] = d176
						ps189.OverlayValues[177] = d177
						ps189.OverlayValues[178] = d178
						ps189.OverlayValues[180] = d180
						ps189.OverlayValues[181] = d181
						ps189.OverlayValues[182] = d182
						ps189.OverlayValues[183] = d183
						ps189.OverlayValues[185] = d185
						ps189.OverlayValues[186] = d186
						ps189.OverlayValues[187] = d187
						return bbs[8].RenderPS(ps189)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d187.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl9)
					ps190 := PhiState{General: true}
					ps190.OverlayValues = make([]JITValueDesc, 188)
					ps190.OverlayValues[1] = d1
					ps190.OverlayValues[2] = d2
					ps190.OverlayValues[3] = d3
					ps190.OverlayValues[4] = d4
					ps190.OverlayValues[5] = d5
					ps190.OverlayValues[6] = d6
					ps190.OverlayValues[7] = d7
					ps190.OverlayValues[8] = d8
					ps190.OverlayValues[9] = d9
					ps190.OverlayValues[10] = d10
					ps190.OverlayValues[11] = d11
					ps190.OverlayValues[12] = d12
					ps190.OverlayValues[13] = d13
					ps190.OverlayValues[14] = d14
					ps190.OverlayValues[15] = d15
					ps190.OverlayValues[18] = d18
					ps190.OverlayValues[21] = d21
					ps190.OverlayValues[40] = d40
					ps190.OverlayValues[41] = d41
					ps190.OverlayValues[42] = d42
					ps190.OverlayValues[43] = d43
					ps190.OverlayValues[44] = d44
					ps190.OverlayValues[46] = d46
					ps190.OverlayValues[47] = d47
					ps190.OverlayValues[48] = d48
					ps190.OverlayValues[49] = d49
					ps190.OverlayValues[50] = d50
					ps190.OverlayValues[51] = d51
					ps190.OverlayValues[52] = d52
					ps190.OverlayValues[55] = d55
					ps190.OverlayValues[90] = d90
					ps190.OverlayValues[91] = d91
					ps190.OverlayValues[92] = d92
					ps190.OverlayValues[93] = d93
					ps190.OverlayValues[94] = d94
					ps190.OverlayValues[95] = d95
					ps190.OverlayValues[97] = d97
					ps190.OverlayValues[98] = d98
					ps190.OverlayValues[99] = d99
					ps190.OverlayValues[100] = d100
					ps190.OverlayValues[101] = d101
					ps190.OverlayValues[102] = d102
					ps190.OverlayValues[103] = d103
					ps190.OverlayValues[104] = d104
					ps190.OverlayValues[105] = d105
					ps190.OverlayValues[108] = d108
					ps190.OverlayValues[109] = d109
					ps190.OverlayValues[110] = d110
					ps190.OverlayValues[111] = d111
					ps190.OverlayValues[164] = d164
					ps190.OverlayValues[165] = d165
					ps190.OverlayValues[166] = d166
					ps190.OverlayValues[167] = d167
					ps190.OverlayValues[168] = d168
					ps190.OverlayValues[169] = d169
					ps190.OverlayValues[170] = d170
					ps190.OverlayValues[171] = d171
					ps190.OverlayValues[172] = d172
					ps190.OverlayValues[173] = d173
					ps190.OverlayValues[174] = d174
					ps190.OverlayValues[175] = d175
					ps190.OverlayValues[176] = d176
					ps190.OverlayValues[177] = d177
					ps190.OverlayValues[178] = d178
					ps190.OverlayValues[180] = d180
					ps190.OverlayValues[181] = d181
					ps190.OverlayValues[182] = d182
					ps190.OverlayValues[183] = d183
					ps190.OverlayValues[185] = d185
					ps190.OverlayValues[186] = d186
					ps190.OverlayValues[187] = d187
					ps191 := PhiState{General: true}
					ps191.OverlayValues = make([]JITValueDesc, 188)
					ps191.OverlayValues[1] = d1
					ps191.OverlayValues[2] = d2
					ps191.OverlayValues[3] = d3
					ps191.OverlayValues[4] = d4
					ps191.OverlayValues[5] = d5
					ps191.OverlayValues[6] = d6
					ps191.OverlayValues[7] = d7
					ps191.OverlayValues[8] = d8
					ps191.OverlayValues[9] = d9
					ps191.OverlayValues[10] = d10
					ps191.OverlayValues[11] = d11
					ps191.OverlayValues[12] = d12
					ps191.OverlayValues[13] = d13
					ps191.OverlayValues[14] = d14
					ps191.OverlayValues[15] = d15
					ps191.OverlayValues[18] = d18
					ps191.OverlayValues[21] = d21
					ps191.OverlayValues[40] = d40
					ps191.OverlayValues[41] = d41
					ps191.OverlayValues[42] = d42
					ps191.OverlayValues[43] = d43
					ps191.OverlayValues[44] = d44
					ps191.OverlayValues[46] = d46
					ps191.OverlayValues[47] = d47
					ps191.OverlayValues[48] = d48
					ps191.OverlayValues[49] = d49
					ps191.OverlayValues[50] = d50
					ps191.OverlayValues[51] = d51
					ps191.OverlayValues[52] = d52
					ps191.OverlayValues[55] = d55
					ps191.OverlayValues[90] = d90
					ps191.OverlayValues[91] = d91
					ps191.OverlayValues[92] = d92
					ps191.OverlayValues[93] = d93
					ps191.OverlayValues[94] = d94
					ps191.OverlayValues[95] = d95
					ps191.OverlayValues[97] = d97
					ps191.OverlayValues[98] = d98
					ps191.OverlayValues[99] = d99
					ps191.OverlayValues[100] = d100
					ps191.OverlayValues[101] = d101
					ps191.OverlayValues[102] = d102
					ps191.OverlayValues[103] = d103
					ps191.OverlayValues[104] = d104
					ps191.OverlayValues[105] = d105
					ps191.OverlayValues[108] = d108
					ps191.OverlayValues[109] = d109
					ps191.OverlayValues[110] = d110
					ps191.OverlayValues[111] = d111
					ps191.OverlayValues[164] = d164
					ps191.OverlayValues[165] = d165
					ps191.OverlayValues[166] = d166
					ps191.OverlayValues[167] = d167
					ps191.OverlayValues[168] = d168
					ps191.OverlayValues[169] = d169
					ps191.OverlayValues[170] = d170
					ps191.OverlayValues[171] = d171
					ps191.OverlayValues[172] = d172
					ps191.OverlayValues[173] = d173
					ps191.OverlayValues[174] = d174
					ps191.OverlayValues[175] = d175
					ps191.OverlayValues[176] = d176
					ps191.OverlayValues[177] = d177
					ps191.OverlayValues[178] = d178
					ps191.OverlayValues[180] = d180
					ps191.OverlayValues[181] = d181
					ps191.OverlayValues[182] = d182
					ps191.OverlayValues[183] = d183
					ps191.OverlayValues[185] = d185
					ps191.OverlayValues[186] = d186
					ps191.OverlayValues[187] = d187
					snap192 := d1
					snap193 := d2
					snap194 := d3
					snap195 := d4
					snap196 := d5
					snap197 := d6
					snap198 := d7
					snap199 := d8
					snap200 := d9
					snap201 := d10
					snap202 := d11
					snap203 := d12
					snap204 := d13
					snap205 := d14
					snap206 := d15
					snap207 := d18
					snap208 := d21
					snap209 := d40
					snap210 := d41
					snap211 := d42
					snap212 := d43
					snap213 := d44
					snap214 := d46
					snap215 := d47
					snap216 := d48
					snap217 := d49
					snap218 := d50
					snap219 := d51
					snap220 := d52
					snap221 := d55
					snap222 := d90
					snap223 := d91
					snap224 := d92
					snap225 := d93
					snap226 := d94
					snap227 := d95
					snap228 := d97
					snap229 := d98
					snap230 := d99
					snap231 := d100
					snap232 := d101
					snap233 := d102
					snap234 := d103
					snap235 := d104
					snap236 := d105
					snap237 := d108
					snap238 := d109
					snap239 := d110
					snap240 := d111
					snap241 := d164
					snap242 := d165
					snap243 := d166
					snap244 := d167
					snap245 := d168
					snap246 := d169
					snap247 := d170
					snap248 := d171
					snap249 := d172
					snap250 := d173
					snap251 := d174
					snap252 := d175
					snap253 := d176
					snap254 := d177
					snap255 := d178
					snap256 := d180
					snap257 := d181
					snap258 := d182
					snap259 := d183
					snap260 := d185
					snap261 := d186
					snap262 := d187
					alloc263 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps191)
					}
					ctx.RestoreAllocState(alloc263)
					d1 = snap192
					d2 = snap193
					d3 = snap194
					d4 = snap195
					d5 = snap196
					d6 = snap197
					d7 = snap198
					d8 = snap199
					d9 = snap200
					d10 = snap201
					d11 = snap202
					d12 = snap203
					d13 = snap204
					d14 = snap205
					d15 = snap206
					d18 = snap207
					d21 = snap208
					d40 = snap209
					d41 = snap210
					d42 = snap211
					d43 = snap212
					d44 = snap213
					d46 = snap214
					d47 = snap215
					d48 = snap216
					d49 = snap217
					d50 = snap218
					d51 = snap219
					d52 = snap220
					d55 = snap221
					d90 = snap222
					d91 = snap223
					d92 = snap224
					d93 = snap225
					d94 = snap226
					d95 = snap227
					d97 = snap228
					d98 = snap229
					d99 = snap230
					d100 = snap231
					d101 = snap232
					d102 = snap233
					d103 = snap234
					d104 = snap235
					d105 = snap236
					d108 = snap237
					d109 = snap238
					d110 = snap239
					d111 = snap240
					d164 = snap241
					d165 = snap242
					d166 = snap243
					d167 = snap244
					d168 = snap245
					d169 = snap246
					d170 = snap247
					d171 = snap248
					d172 = snap249
					d173 = snap250
					d174 = snap251
					d175 = snap252
					d176 = snap253
					d177 = snap254
					d178 = snap255
					d180 = snap256
					d181 = snap257
					d182 = snap258
					d183 = snap259
					d185 = snap260
					d186 = snap261
					d187 = snap262
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps190)
					}
					return result
					ctx.FreeDesc(&d186)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d264 := ps.PhiValues[0]
							ctx.EnsureDesc(&d264)
							ctx.EmitStoreToStack(d264, int32(bbs[10].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d265 := ps.PhiValues[1]
							ctx.EnsureDesc(&d265)
							ctx.EmitStoreToStack(d265, int32(bbs[10].PhiBase)+int32(16))
						}
						if bbs[10].VisitCount >= 0 {
							ps.General = true
							return bbs[10].RenderPS(ps)
						}
					}
					bbs[10].VisitCount++
					if ps.General {
						if bbs[10].Rendered {
							ctx.EmitJmp(lbl11)
							return result
						}
						bbs[10].Rendered = true
						bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_10 = bbs[10].Address
						ctx.MarkLabel(lbl11)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != LocNone {
						d169 = ps.OverlayValues[169]
					}
					if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != LocNone {
						d170 = ps.OverlayValues[170]
					}
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != LocNone {
						d264 = ps.OverlayValues[264]
					}
					if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != LocNone {
						d265 = ps.OverlayValues[265]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d7 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d8 = ps.PhiValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.StabilizeDescForControlFlow(&d8)
					var d266 JITValueDesc
					if d10.SliceSizeKnown {
						d266 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.KnownSliceLen))}
					} else if d10.Loc == LocImm {
						d266 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.StackOff))}
					} else if d10.Loc == LocStackTriple {
						d266 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d10.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d10)
						if d10.Loc == LocRegPair || d10.Loc == LocRegTriple {
							d266 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg2, ID: 0}
						} else if d10.Loc == LocReg {
							d266 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d266)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d266)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d266)
					var d267 JITValueDesc
					if d8.Loc == LocImm && d266.Loc == LocImm {
						d267 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < d266.Imm.Int())}
					} else if d266.Loc == LocImm {
						r14 := ctx.AllocRegExcept(d8.Reg)
						if d266.Imm.Int() >= -2147483648 && d266.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d8.Reg, int32(d266.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d266.Imm.Int()))
							ctx.EmitCmpInt64(d8.Reg, RegR11)
						}
						ctx.EmitSetcc(r14, CondSignedLess)
						d267 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d267)
					} else if d8.Loc == LocImm {
						r15 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d266.Reg)
						ctx.EmitSetcc(r15, CondSignedLess)
						d267 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
						ctx.BindReg(r15, &d267)
					} else {
						r16 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpInt64(d8.Reg, d266.Reg)
						ctx.EmitSetcc(r16, CondSignedLess)
						d267 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
						ctx.BindReg(r16, &d267)
					}
					ctx.FreeDesc(&d266)
					d268 = d267
					ctx.EnsureDesc(&d268)
					if d268.Loc != LocImm && d268.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d268.Loc == LocImm {
						if d268.Imm.Bool() {
							if ps.General {
							}
							ps269 := PhiState{General: ps.General}
							ps269.OverlayValues = make([]JITValueDesc, 269)
							ps269.OverlayValues[1] = d1
							ps269.OverlayValues[2] = d2
							ps269.OverlayValues[3] = d3
							ps269.OverlayValues[4] = d4
							ps269.OverlayValues[5] = d5
							ps269.OverlayValues[6] = d6
							ps269.OverlayValues[7] = d7
							ps269.OverlayValues[8] = d8
							ps269.OverlayValues[9] = d9
							ps269.OverlayValues[10] = d10
							ps269.OverlayValues[11] = d11
							ps269.OverlayValues[12] = d12
							ps269.OverlayValues[13] = d13
							ps269.OverlayValues[14] = d14
							ps269.OverlayValues[15] = d15
							ps269.OverlayValues[18] = d18
							ps269.OverlayValues[21] = d21
							ps269.OverlayValues[40] = d40
							ps269.OverlayValues[41] = d41
							ps269.OverlayValues[42] = d42
							ps269.OverlayValues[43] = d43
							ps269.OverlayValues[44] = d44
							ps269.OverlayValues[46] = d46
							ps269.OverlayValues[47] = d47
							ps269.OverlayValues[48] = d48
							ps269.OverlayValues[49] = d49
							ps269.OverlayValues[50] = d50
							ps269.OverlayValues[51] = d51
							ps269.OverlayValues[52] = d52
							ps269.OverlayValues[55] = d55
							ps269.OverlayValues[90] = d90
							ps269.OverlayValues[91] = d91
							ps269.OverlayValues[92] = d92
							ps269.OverlayValues[93] = d93
							ps269.OverlayValues[94] = d94
							ps269.OverlayValues[95] = d95
							ps269.OverlayValues[97] = d97
							ps269.OverlayValues[98] = d98
							ps269.OverlayValues[99] = d99
							ps269.OverlayValues[100] = d100
							ps269.OverlayValues[101] = d101
							ps269.OverlayValues[102] = d102
							ps269.OverlayValues[103] = d103
							ps269.OverlayValues[104] = d104
							ps269.OverlayValues[105] = d105
							ps269.OverlayValues[108] = d108
							ps269.OverlayValues[109] = d109
							ps269.OverlayValues[110] = d110
							ps269.OverlayValues[111] = d111
							ps269.OverlayValues[164] = d164
							ps269.OverlayValues[165] = d165
							ps269.OverlayValues[166] = d166
							ps269.OverlayValues[167] = d167
							ps269.OverlayValues[168] = d168
							ps269.OverlayValues[169] = d169
							ps269.OverlayValues[170] = d170
							ps269.OverlayValues[171] = d171
							ps269.OverlayValues[172] = d172
							ps269.OverlayValues[173] = d173
							ps269.OverlayValues[174] = d174
							ps269.OverlayValues[175] = d175
							ps269.OverlayValues[176] = d176
							ps269.OverlayValues[177] = d177
							ps269.OverlayValues[178] = d178
							ps269.OverlayValues[180] = d180
							ps269.OverlayValues[181] = d181
							ps269.OverlayValues[182] = d182
							ps269.OverlayValues[183] = d183
							ps269.OverlayValues[185] = d185
							ps269.OverlayValues[186] = d186
							ps269.OverlayValues[187] = d187
							ps269.OverlayValues[264] = d264
							ps269.OverlayValues[265] = d265
							ps269.OverlayValues[266] = d266
							ps269.OverlayValues[267] = d267
							ps269.OverlayValues[268] = d268
							return bbs[13].RenderPS(ps269)
						}
						if ps.General {
						}
						ps270 := PhiState{General: ps.General}
						ps270.OverlayValues = make([]JITValueDesc, 269)
						ps270.OverlayValues[1] = d1
						ps270.OverlayValues[2] = d2
						ps270.OverlayValues[3] = d3
						ps270.OverlayValues[4] = d4
						ps270.OverlayValues[5] = d5
						ps270.OverlayValues[6] = d6
						ps270.OverlayValues[7] = d7
						ps270.OverlayValues[8] = d8
						ps270.OverlayValues[9] = d9
						ps270.OverlayValues[10] = d10
						ps270.OverlayValues[11] = d11
						ps270.OverlayValues[12] = d12
						ps270.OverlayValues[13] = d13
						ps270.OverlayValues[14] = d14
						ps270.OverlayValues[15] = d15
						ps270.OverlayValues[18] = d18
						ps270.OverlayValues[21] = d21
						ps270.OverlayValues[40] = d40
						ps270.OverlayValues[41] = d41
						ps270.OverlayValues[42] = d42
						ps270.OverlayValues[43] = d43
						ps270.OverlayValues[44] = d44
						ps270.OverlayValues[46] = d46
						ps270.OverlayValues[47] = d47
						ps270.OverlayValues[48] = d48
						ps270.OverlayValues[49] = d49
						ps270.OverlayValues[50] = d50
						ps270.OverlayValues[51] = d51
						ps270.OverlayValues[52] = d52
						ps270.OverlayValues[55] = d55
						ps270.OverlayValues[90] = d90
						ps270.OverlayValues[91] = d91
						ps270.OverlayValues[92] = d92
						ps270.OverlayValues[93] = d93
						ps270.OverlayValues[94] = d94
						ps270.OverlayValues[95] = d95
						ps270.OverlayValues[97] = d97
						ps270.OverlayValues[98] = d98
						ps270.OverlayValues[99] = d99
						ps270.OverlayValues[100] = d100
						ps270.OverlayValues[101] = d101
						ps270.OverlayValues[102] = d102
						ps270.OverlayValues[103] = d103
						ps270.OverlayValues[104] = d104
						ps270.OverlayValues[105] = d105
						ps270.OverlayValues[108] = d108
						ps270.OverlayValues[109] = d109
						ps270.OverlayValues[110] = d110
						ps270.OverlayValues[111] = d111
						ps270.OverlayValues[164] = d164
						ps270.OverlayValues[165] = d165
						ps270.OverlayValues[166] = d166
						ps270.OverlayValues[167] = d167
						ps270.OverlayValues[168] = d168
						ps270.OverlayValues[169] = d169
						ps270.OverlayValues[170] = d170
						ps270.OverlayValues[171] = d171
						ps270.OverlayValues[172] = d172
						ps270.OverlayValues[173] = d173
						ps270.OverlayValues[174] = d174
						ps270.OverlayValues[175] = d175
						ps270.OverlayValues[176] = d176
						ps270.OverlayValues[177] = d177
						ps270.OverlayValues[178] = d178
						ps270.OverlayValues[180] = d180
						ps270.OverlayValues[181] = d181
						ps270.OverlayValues[182] = d182
						ps270.OverlayValues[183] = d183
						ps270.OverlayValues[185] = d185
						ps270.OverlayValues[186] = d186
						ps270.OverlayValues[187] = d187
						ps270.OverlayValues[264] = d264
						ps270.OverlayValues[265] = d265
						ps270.OverlayValues[266] = d266
						ps270.OverlayValues[267] = d267
						ps270.OverlayValues[268] = d268
						return bbs[12].RenderPS(ps270)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d271 := ps.PhiValues[0]
							ctx.EnsureDesc(&d271)
							ctx.EmitStoreToStack(d271, int32(bbs[10].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d272 := ps.PhiValues[1]
							ctx.EnsureDesc(&d272)
							ctx.EmitStoreToStack(d272, int32(bbs[10].PhiBase)+int32(16))
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d268.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl13)
					ps273 := PhiState{General: true}
					ps273.OverlayValues = make([]JITValueDesc, 273)
					ps273.OverlayValues[1] = d1
					ps273.OverlayValues[2] = d2
					ps273.OverlayValues[3] = d3
					ps273.OverlayValues[4] = d4
					ps273.OverlayValues[5] = d5
					ps273.OverlayValues[6] = d6
					ps273.OverlayValues[7] = d7
					ps273.OverlayValues[8] = d8
					ps273.OverlayValues[9] = d9
					ps273.OverlayValues[10] = d10
					ps273.OverlayValues[11] = d11
					ps273.OverlayValues[12] = d12
					ps273.OverlayValues[13] = d13
					ps273.OverlayValues[14] = d14
					ps273.OverlayValues[15] = d15
					ps273.OverlayValues[18] = d18
					ps273.OverlayValues[21] = d21
					ps273.OverlayValues[40] = d40
					ps273.OverlayValues[41] = d41
					ps273.OverlayValues[42] = d42
					ps273.OverlayValues[43] = d43
					ps273.OverlayValues[44] = d44
					ps273.OverlayValues[46] = d46
					ps273.OverlayValues[47] = d47
					ps273.OverlayValues[48] = d48
					ps273.OverlayValues[49] = d49
					ps273.OverlayValues[50] = d50
					ps273.OverlayValues[51] = d51
					ps273.OverlayValues[52] = d52
					ps273.OverlayValues[55] = d55
					ps273.OverlayValues[90] = d90
					ps273.OverlayValues[91] = d91
					ps273.OverlayValues[92] = d92
					ps273.OverlayValues[93] = d93
					ps273.OverlayValues[94] = d94
					ps273.OverlayValues[95] = d95
					ps273.OverlayValues[97] = d97
					ps273.OverlayValues[98] = d98
					ps273.OverlayValues[99] = d99
					ps273.OverlayValues[100] = d100
					ps273.OverlayValues[101] = d101
					ps273.OverlayValues[102] = d102
					ps273.OverlayValues[103] = d103
					ps273.OverlayValues[104] = d104
					ps273.OverlayValues[105] = d105
					ps273.OverlayValues[108] = d108
					ps273.OverlayValues[109] = d109
					ps273.OverlayValues[110] = d110
					ps273.OverlayValues[111] = d111
					ps273.OverlayValues[164] = d164
					ps273.OverlayValues[165] = d165
					ps273.OverlayValues[166] = d166
					ps273.OverlayValues[167] = d167
					ps273.OverlayValues[168] = d168
					ps273.OverlayValues[169] = d169
					ps273.OverlayValues[170] = d170
					ps273.OverlayValues[171] = d171
					ps273.OverlayValues[172] = d172
					ps273.OverlayValues[173] = d173
					ps273.OverlayValues[174] = d174
					ps273.OverlayValues[175] = d175
					ps273.OverlayValues[176] = d176
					ps273.OverlayValues[177] = d177
					ps273.OverlayValues[178] = d178
					ps273.OverlayValues[180] = d180
					ps273.OverlayValues[181] = d181
					ps273.OverlayValues[182] = d182
					ps273.OverlayValues[183] = d183
					ps273.OverlayValues[185] = d185
					ps273.OverlayValues[186] = d186
					ps273.OverlayValues[187] = d187
					ps273.OverlayValues[264] = d264
					ps273.OverlayValues[265] = d265
					ps273.OverlayValues[266] = d266
					ps273.OverlayValues[267] = d267
					ps273.OverlayValues[268] = d268
					ps273.OverlayValues[271] = d271
					ps273.OverlayValues[272] = d272
					ps274 := PhiState{General: true}
					ps274.OverlayValues = make([]JITValueDesc, 273)
					ps274.OverlayValues[1] = d1
					ps274.OverlayValues[2] = d2
					ps274.OverlayValues[3] = d3
					ps274.OverlayValues[4] = d4
					ps274.OverlayValues[5] = d5
					ps274.OverlayValues[6] = d6
					ps274.OverlayValues[7] = d7
					ps274.OverlayValues[8] = d8
					ps274.OverlayValues[9] = d9
					ps274.OverlayValues[10] = d10
					ps274.OverlayValues[11] = d11
					ps274.OverlayValues[12] = d12
					ps274.OverlayValues[13] = d13
					ps274.OverlayValues[14] = d14
					ps274.OverlayValues[15] = d15
					ps274.OverlayValues[18] = d18
					ps274.OverlayValues[21] = d21
					ps274.OverlayValues[40] = d40
					ps274.OverlayValues[41] = d41
					ps274.OverlayValues[42] = d42
					ps274.OverlayValues[43] = d43
					ps274.OverlayValues[44] = d44
					ps274.OverlayValues[46] = d46
					ps274.OverlayValues[47] = d47
					ps274.OverlayValues[48] = d48
					ps274.OverlayValues[49] = d49
					ps274.OverlayValues[50] = d50
					ps274.OverlayValues[51] = d51
					ps274.OverlayValues[52] = d52
					ps274.OverlayValues[55] = d55
					ps274.OverlayValues[90] = d90
					ps274.OverlayValues[91] = d91
					ps274.OverlayValues[92] = d92
					ps274.OverlayValues[93] = d93
					ps274.OverlayValues[94] = d94
					ps274.OverlayValues[95] = d95
					ps274.OverlayValues[97] = d97
					ps274.OverlayValues[98] = d98
					ps274.OverlayValues[99] = d99
					ps274.OverlayValues[100] = d100
					ps274.OverlayValues[101] = d101
					ps274.OverlayValues[102] = d102
					ps274.OverlayValues[103] = d103
					ps274.OverlayValues[104] = d104
					ps274.OverlayValues[105] = d105
					ps274.OverlayValues[108] = d108
					ps274.OverlayValues[109] = d109
					ps274.OverlayValues[110] = d110
					ps274.OverlayValues[111] = d111
					ps274.OverlayValues[164] = d164
					ps274.OverlayValues[165] = d165
					ps274.OverlayValues[166] = d166
					ps274.OverlayValues[167] = d167
					ps274.OverlayValues[168] = d168
					ps274.OverlayValues[169] = d169
					ps274.OverlayValues[170] = d170
					ps274.OverlayValues[171] = d171
					ps274.OverlayValues[172] = d172
					ps274.OverlayValues[173] = d173
					ps274.OverlayValues[174] = d174
					ps274.OverlayValues[175] = d175
					ps274.OverlayValues[176] = d176
					ps274.OverlayValues[177] = d177
					ps274.OverlayValues[178] = d178
					ps274.OverlayValues[180] = d180
					ps274.OverlayValues[181] = d181
					ps274.OverlayValues[182] = d182
					ps274.OverlayValues[183] = d183
					ps274.OverlayValues[185] = d185
					ps274.OverlayValues[186] = d186
					ps274.OverlayValues[187] = d187
					ps274.OverlayValues[264] = d264
					ps274.OverlayValues[265] = d265
					ps274.OverlayValues[266] = d266
					ps274.OverlayValues[267] = d267
					ps274.OverlayValues[268] = d268
					ps274.OverlayValues[271] = d271
					ps274.OverlayValues[272] = d272
					snap275 := d1
					snap276 := d2
					snap277 := d3
					snap278 := d4
					snap279 := d5
					snap280 := d6
					snap281 := d7
					snap282 := d8
					snap283 := d9
					snap284 := d10
					snap285 := d11
					snap286 := d12
					snap287 := d13
					snap288 := d14
					snap289 := d15
					snap290 := d18
					snap291 := d21
					snap292 := d40
					snap293 := d41
					snap294 := d42
					snap295 := d43
					snap296 := d44
					snap297 := d46
					snap298 := d47
					snap299 := d48
					snap300 := d49
					snap301 := d50
					snap302 := d51
					snap303 := d52
					snap304 := d55
					snap305 := d90
					snap306 := d91
					snap307 := d92
					snap308 := d93
					snap309 := d94
					snap310 := d95
					snap311 := d97
					snap312 := d98
					snap313 := d99
					snap314 := d100
					snap315 := d101
					snap316 := d102
					snap317 := d103
					snap318 := d104
					snap319 := d105
					snap320 := d108
					snap321 := d109
					snap322 := d110
					snap323 := d111
					snap324 := d164
					snap325 := d165
					snap326 := d166
					snap327 := d167
					snap328 := d168
					snap329 := d169
					snap330 := d170
					snap331 := d171
					snap332 := d172
					snap333 := d173
					snap334 := d174
					snap335 := d175
					snap336 := d176
					snap337 := d177
					snap338 := d178
					snap339 := d180
					snap340 := d181
					snap341 := d182
					snap342 := d183
					snap343 := d185
					snap344 := d186
					snap345 := d187
					snap346 := d264
					snap347 := d265
					snap348 := d266
					snap349 := d267
					snap350 := d268
					snap351 := d271
					snap352 := d272
					alloc353 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps274)
					}
					ctx.RestoreAllocState(alloc353)
					d1 = snap275
					d2 = snap276
					d3 = snap277
					d4 = snap278
					d5 = snap279
					d6 = snap280
					d7 = snap281
					d8 = snap282
					d9 = snap283
					d10 = snap284
					d11 = snap285
					d12 = snap286
					d13 = snap287
					d14 = snap288
					d15 = snap289
					d18 = snap290
					d21 = snap291
					d40 = snap292
					d41 = snap293
					d42 = snap294
					d43 = snap295
					d44 = snap296
					d46 = snap297
					d47 = snap298
					d48 = snap299
					d49 = snap300
					d50 = snap301
					d51 = snap302
					d52 = snap303
					d55 = snap304
					d90 = snap305
					d91 = snap306
					d92 = snap307
					d93 = snap308
					d94 = snap309
					d95 = snap310
					d97 = snap311
					d98 = snap312
					d99 = snap313
					d100 = snap314
					d101 = snap315
					d102 = snap316
					d103 = snap317
					d104 = snap318
					d105 = snap319
					d108 = snap320
					d109 = snap321
					d110 = snap322
					d111 = snap323
					d164 = snap324
					d165 = snap325
					d166 = snap326
					d167 = snap327
					d168 = snap328
					d169 = snap329
					d170 = snap330
					d171 = snap331
					d172 = snap332
					d173 = snap333
					d174 = snap334
					d175 = snap335
					d176 = snap336
					d177 = snap337
					d178 = snap338
					d180 = snap339
					d181 = snap340
					d182 = snap341
					d183 = snap342
					d185 = snap343
					d186 = snap344
					d187 = snap345
					d264 = snap346
					d265 = snap347
					d266 = snap348
					d267 = snap349
					d268 = snap350
					d271 = snap351
					d272 = snap352
					if !bbs[13].Rendered {
						return bbs[13].RenderPS(ps273)
					}
					return result
					ctx.FreeDesc(&d267)
					return result
				}
				bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[11].VisitCount >= 0 {
							ps.General = true
							return bbs[11].RenderPS(ps)
						}
					}
					bbs[11].VisitCount++
					if ps.General {
						if bbs[11].Rendered {
							ctx.EmitJmp(lbl12)
							return result
						}
						bbs[11].Rendered = true
						bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_11 = bbs[11].Address
						ctx.MarkLabel(lbl12)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != LocNone {
						d169 = ps.OverlayValues[169]
					}
					if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != LocNone {
						d170 = ps.OverlayValues[170]
					}
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != LocNone {
						d264 = ps.OverlayValues[264]
					}
					if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != LocNone {
						d265 = ps.OverlayValues[265]
					}
					if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != LocNone {
						d266 = ps.OverlayValues[266]
					}
					if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != LocNone {
						d267 = ps.OverlayValues[267]
					}
					if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != LocNone {
						d268 = ps.OverlayValues[268]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					d355 = ctx.EmitSliceElementAddress(&d10, &d8, 16)
					ctx.EnsureDesc(&d355)
					r17 := ctx.AllocRegExcept(d355.Reg)
					ctx.EmitMovRegMem(r17, d355.Reg, 8)
					ctx.EmitMovRegMem(d355.Reg, d355.Reg, 0)
					d354 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d355.Reg, Reg2: r17}
					ctx.BindReg(d355.Reg, &d354)
					ctx.BindReg(r17, &d354)
					ctx.EnsureDesc(&d354)
					d356 = d354
					_ = d356
					ctx.StabilizeDescForControlFlow(&d356)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d357 JITValueDesc
					if d356.Loc == LocImm {
						d357 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d356.Imm.Float())}
					} else if d356.Type == tagFloat && d356.Loc == LocReg {
						d357 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d356.Reg}
						ctx.BindReg(d356.Reg, &d357)
						ctx.BindReg(d356.Reg, &d357)
					} else if d356.Type == tagFloat && d356.Loc == LocRegPair {
						ctx.FreeReg(d356.Reg)
						d357 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d356.Reg2}
						ctx.BindReg(d356.Reg2, &d357)
						ctx.BindReg(d356.Reg2, &d357)
					} else {
						d357 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d356}, 1)
						d357.Type = tagFloat
						ctx.BindReg(d357.Reg, &d357)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d357)
					ctx.FreeDesc(&d354)
					ctx.EnsureDesc(&d8)
					d359 = ctx.EmitSliceElementAddress(&d12, &d8, 16)
					ctx.EnsureDesc(&d359)
					r18 := ctx.AllocRegExcept(d359.Reg)
					ctx.EmitMovRegMem(r18, d359.Reg, 8)
					ctx.EmitMovRegMem(d359.Reg, d359.Reg, 0)
					d358 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d359.Reg, Reg2: r18}
					ctx.BindReg(d359.Reg, &d358)
					ctx.BindReg(r18, &d358)
					ctx.EnsureDesc(&d358)
					d360 = d358
					_ = d360
					ctx.StabilizeDescForControlFlow(&d360)
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d361 JITValueDesc
					if d360.Loc == LocImm {
						d361 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d360.Imm.Float())}
					} else if d360.Type == tagFloat && d360.Loc == LocReg {
						d361 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d360.Reg}
						ctx.BindReg(d360.Reg, &d361)
						ctx.BindReg(d360.Reg, &d361)
					} else if d360.Type == tagFloat && d360.Loc == LocRegPair {
						ctx.FreeReg(d360.Reg)
						d361 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d360.Reg2}
						ctx.BindReg(d360.Reg2, &d361)
						ctx.BindReg(d360.Reg2, &d361)
					} else {
						d361 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d360}, 1)
						d361.Type = tagFloat
						ctx.BindReg(d361.Reg, &d361)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d361)
					ctx.FreeDesc(&d358)
					ctx.EnsureDesc(&d357)
					ctx.EnsureDesc(&d361)
					ctx.EnsureDesc(&d357)
					ctx.EnsureDesc(&d361)
					var d362 JITValueDesc
					if d357.Loc == LocImm && d361.Loc == LocImm {
						d362 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d357.Imm.Float() * d361.Imm.Float())}
					} else if d357.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d361.Reg)
						_, xBits := d357.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d361.Reg)
						d362 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d362)
					} else if d361.Loc == LocImm {
						_, yBits := d361.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d357.Reg, RegR11)
						d362 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d357.Reg}
						ctx.BindReg(d357.Reg, &d362)
					} else {
						ctx.EmitMulFloat64(d357.Reg, d361.Reg)
						d362 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d357.Reg}
						ctx.BindReg(d357.Reg, &d362)
					}
					if d362.Loc == LocReg && d357.Loc == LocReg && d362.Reg == d357.Reg {
						ctx.TransferReg(d357.Reg)
						d357.Loc = LocNone
					}
					ctx.FreeDesc(&d357)
					ctx.FreeDesc(&d361)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d362)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d362)
					var d363 JITValueDesc
					if d7.Loc == LocImm && d362.Loc == LocImm {
						d363 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d7.Imm.Float() + d362.Imm.Float())}
					} else if d7.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d362.Reg)
						_, xBits := d7.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d362.Reg)
						d363 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d363)
					} else if d362.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegReg(scratch, d7.Reg)
						_, yBits := d362.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d363 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d363)
					} else {
						r19 := ctx.AllocRegExcept(d7.Reg, d362.Reg)
						ctx.EmitMovRegReg(r19, d7.Reg)
						ctx.EmitAddFloat64(r19, d362.Reg)
						d363 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r19}
						ctx.BindReg(r19, &d363)
					}
					if d363.Loc == LocReg && d7.Loc == LocReg && d363.Reg == d7.Reg {
						ctx.TransferReg(d7.Reg)
						d7.Loc = LocNone
					}
					ctx.EnsureDesc(&d363)
					ctx.EmitStoreToStack(d363, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d363)
					ctx.FreeDesc(&d362)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d8)
					var d364 JITValueDesc
					if d8.Loc == LocImm {
						d364 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitMovRegReg(scratch, d8.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d364 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d364)
					}
					if d364.Loc == LocReg && d8.Loc == LocReg && d364.Reg == d8.Reg {
						ctx.TransferReg(d8.Reg)
						d8.Loc = LocNone
					}
					ctx.EnsureDesc(&d364)
					ctx.EmitStoreToStack(d364, int32(bbs[10].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d364)
					if ps.General {
					}
					ps365 := PhiState{General: ps.General}
					ps365.OverlayValues = make([]JITValueDesc, 365)
					ps365.OverlayValues[1] = d1
					ps365.OverlayValues[2] = d2
					ps365.OverlayValues[3] = d3
					ps365.OverlayValues[4] = d4
					ps365.OverlayValues[5] = d5
					ps365.OverlayValues[6] = d6
					ps365.OverlayValues[7] = d7
					ps365.OverlayValues[8] = d8
					ps365.OverlayValues[9] = d9
					ps365.OverlayValues[10] = d10
					ps365.OverlayValues[11] = d11
					ps365.OverlayValues[12] = d12
					ps365.OverlayValues[13] = d13
					ps365.OverlayValues[14] = d14
					ps365.OverlayValues[15] = d15
					ps365.OverlayValues[18] = d18
					ps365.OverlayValues[21] = d21
					ps365.OverlayValues[40] = d40
					ps365.OverlayValues[41] = d41
					ps365.OverlayValues[42] = d42
					ps365.OverlayValues[43] = d43
					ps365.OverlayValues[44] = d44
					ps365.OverlayValues[46] = d46
					ps365.OverlayValues[47] = d47
					ps365.OverlayValues[48] = d48
					ps365.OverlayValues[49] = d49
					ps365.OverlayValues[50] = d50
					ps365.OverlayValues[51] = d51
					ps365.OverlayValues[52] = d52
					ps365.OverlayValues[55] = d55
					ps365.OverlayValues[90] = d90
					ps365.OverlayValues[91] = d91
					ps365.OverlayValues[92] = d92
					ps365.OverlayValues[93] = d93
					ps365.OverlayValues[94] = d94
					ps365.OverlayValues[95] = d95
					ps365.OverlayValues[97] = d97
					ps365.OverlayValues[98] = d98
					ps365.OverlayValues[99] = d99
					ps365.OverlayValues[100] = d100
					ps365.OverlayValues[101] = d101
					ps365.OverlayValues[102] = d102
					ps365.OverlayValues[103] = d103
					ps365.OverlayValues[104] = d104
					ps365.OverlayValues[105] = d105
					ps365.OverlayValues[108] = d108
					ps365.OverlayValues[109] = d109
					ps365.OverlayValues[110] = d110
					ps365.OverlayValues[111] = d111
					ps365.OverlayValues[164] = d164
					ps365.OverlayValues[165] = d165
					ps365.OverlayValues[166] = d166
					ps365.OverlayValues[167] = d167
					ps365.OverlayValues[168] = d168
					ps365.OverlayValues[169] = d169
					ps365.OverlayValues[170] = d170
					ps365.OverlayValues[171] = d171
					ps365.OverlayValues[172] = d172
					ps365.OverlayValues[173] = d173
					ps365.OverlayValues[174] = d174
					ps365.OverlayValues[175] = d175
					ps365.OverlayValues[176] = d176
					ps365.OverlayValues[177] = d177
					ps365.OverlayValues[178] = d178
					ps365.OverlayValues[180] = d180
					ps365.OverlayValues[181] = d181
					ps365.OverlayValues[182] = d182
					ps365.OverlayValues[183] = d183
					ps365.OverlayValues[185] = d185
					ps365.OverlayValues[186] = d186
					ps365.OverlayValues[187] = d187
					ps365.OverlayValues[264] = d264
					ps365.OverlayValues[265] = d265
					ps365.OverlayValues[266] = d266
					ps365.OverlayValues[267] = d267
					ps365.OverlayValues[268] = d268
					ps365.OverlayValues[271] = d271
					ps365.OverlayValues[272] = d272
					ps365.OverlayValues[354] = d354
					ps365.OverlayValues[355] = d355
					ps365.OverlayValues[356] = d356
					ps365.OverlayValues[357] = d357
					ps365.OverlayValues[358] = d358
					ps365.OverlayValues[359] = d359
					ps365.OverlayValues[360] = d360
					ps365.OverlayValues[361] = d361
					ps365.OverlayValues[362] = d362
					ps365.OverlayValues[363] = d363
					ps365.OverlayValues[364] = d364
					ps365.PhiValues = make([]JITValueDesc, 2)
					if ps365.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps365)
					return result
				}
				bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[12].VisitCount >= 0 {
							ps.General = true
							return bbs[12].RenderPS(ps)
						}
					}
					bbs[12].VisitCount++
					if ps.General {
						if bbs[12].Rendered {
							ctx.EmitJmp(lbl13)
							return result
						}
						bbs[12].Rendered = true
						bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_12 = bbs[12].Address
						ctx.MarkLabel(lbl13)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != LocNone {
						d169 = ps.OverlayValues[169]
					}
					if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != LocNone {
						d170 = ps.OverlayValues[170]
					}
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != LocNone {
						d264 = ps.OverlayValues[264]
					}
					if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != LocNone {
						d265 = ps.OverlayValues[265]
					}
					if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != LocNone {
						d266 = ps.OverlayValues[266]
					}
					if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != LocNone {
						d267 = ps.OverlayValues[267]
					}
					if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != LocNone {
						d268 = ps.OverlayValues[268]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
					}
					if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != LocNone {
						d361 = ps.OverlayValues[361]
					}
					if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != LocNone {
						d362 = ps.OverlayValues[362]
					}
					if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != LocNone {
						d363 = ps.OverlayValues[363]
					}
					if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != LocNone {
						d364 = ps.OverlayValues[364]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					var d366 JITValueDesc
					if d1.Loc == LocImm {
						ctx.TrackImm(d1.Imm)
						ptrWord, _ := d1.Imm.RawWords()
						d366 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d366.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d366.Reg2, uint64(len(d1.Imm.String())))
						ctx.BindReg(d366.Reg, &d366)
						ctx.BindReg(d366.Reg2, &d366)
					} else {
						d366 = d1
					}
					d367 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("EUCLIDEAN")}
					var d368 JITValueDesc
					if d367.Loc == LocImm {
						ctx.TrackImm(d367.Imm)
						ptrWord, _ := d367.Imm.RawWords()
						d368 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d368.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d368.Reg2, uint64(len(d367.Imm.String())))
						ctx.BindReg(d368.Reg, &d368)
						ctx.BindReg(d368.Reg2, &d368)
					} else {
						d368 = d367
					}
					d369 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d366, d368}, 1)
					ctx.EmitAndRegImm32(d369.Reg, 1)
					d369.Type = tagBool
					ctx.BindReg(d369.Reg, &d369)
					ctx.FreeDesc(&d1)
					d370 = d369
					ctx.EnsureDesc(&d370)
					if d370.Loc != LocImm && d370.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d370.Loc == LocImm {
						if d370.Imm.Bool() {
							if ps.General {
							}
							ps371 := PhiState{General: ps.General}
							ps371.OverlayValues = make([]JITValueDesc, 371)
							ps371.OverlayValues[1] = d1
							ps371.OverlayValues[2] = d2
							ps371.OverlayValues[3] = d3
							ps371.OverlayValues[4] = d4
							ps371.OverlayValues[5] = d5
							ps371.OverlayValues[6] = d6
							ps371.OverlayValues[7] = d7
							ps371.OverlayValues[8] = d8
							ps371.OverlayValues[9] = d9
							ps371.OverlayValues[10] = d10
							ps371.OverlayValues[11] = d11
							ps371.OverlayValues[12] = d12
							ps371.OverlayValues[13] = d13
							ps371.OverlayValues[14] = d14
							ps371.OverlayValues[15] = d15
							ps371.OverlayValues[18] = d18
							ps371.OverlayValues[21] = d21
							ps371.OverlayValues[40] = d40
							ps371.OverlayValues[41] = d41
							ps371.OverlayValues[42] = d42
							ps371.OverlayValues[43] = d43
							ps371.OverlayValues[44] = d44
							ps371.OverlayValues[46] = d46
							ps371.OverlayValues[47] = d47
							ps371.OverlayValues[48] = d48
							ps371.OverlayValues[49] = d49
							ps371.OverlayValues[50] = d50
							ps371.OverlayValues[51] = d51
							ps371.OverlayValues[52] = d52
							ps371.OverlayValues[55] = d55
							ps371.OverlayValues[90] = d90
							ps371.OverlayValues[91] = d91
							ps371.OverlayValues[92] = d92
							ps371.OverlayValues[93] = d93
							ps371.OverlayValues[94] = d94
							ps371.OverlayValues[95] = d95
							ps371.OverlayValues[97] = d97
							ps371.OverlayValues[98] = d98
							ps371.OverlayValues[99] = d99
							ps371.OverlayValues[100] = d100
							ps371.OverlayValues[101] = d101
							ps371.OverlayValues[102] = d102
							ps371.OverlayValues[103] = d103
							ps371.OverlayValues[104] = d104
							ps371.OverlayValues[105] = d105
							ps371.OverlayValues[108] = d108
							ps371.OverlayValues[109] = d109
							ps371.OverlayValues[110] = d110
							ps371.OverlayValues[111] = d111
							ps371.OverlayValues[164] = d164
							ps371.OverlayValues[165] = d165
							ps371.OverlayValues[166] = d166
							ps371.OverlayValues[167] = d167
							ps371.OverlayValues[168] = d168
							ps371.OverlayValues[169] = d169
							ps371.OverlayValues[170] = d170
							ps371.OverlayValues[171] = d171
							ps371.OverlayValues[172] = d172
							ps371.OverlayValues[173] = d173
							ps371.OverlayValues[174] = d174
							ps371.OverlayValues[175] = d175
							ps371.OverlayValues[176] = d176
							ps371.OverlayValues[177] = d177
							ps371.OverlayValues[178] = d178
							ps371.OverlayValues[180] = d180
							ps371.OverlayValues[181] = d181
							ps371.OverlayValues[182] = d182
							ps371.OverlayValues[183] = d183
							ps371.OverlayValues[185] = d185
							ps371.OverlayValues[186] = d186
							ps371.OverlayValues[187] = d187
							ps371.OverlayValues[264] = d264
							ps371.OverlayValues[265] = d265
							ps371.OverlayValues[266] = d266
							ps371.OverlayValues[267] = d267
							ps371.OverlayValues[268] = d268
							ps371.OverlayValues[271] = d271
							ps371.OverlayValues[272] = d272
							ps371.OverlayValues[354] = d354
							ps371.OverlayValues[355] = d355
							ps371.OverlayValues[356] = d356
							ps371.OverlayValues[357] = d357
							ps371.OverlayValues[358] = d358
							ps371.OverlayValues[359] = d359
							ps371.OverlayValues[360] = d360
							ps371.OverlayValues[361] = d361
							ps371.OverlayValues[362] = d362
							ps371.OverlayValues[363] = d363
							ps371.OverlayValues[364] = d364
							ps371.OverlayValues[366] = d366
							ps371.OverlayValues[367] = d367
							ps371.OverlayValues[368] = d368
							ps371.OverlayValues[369] = d369
							ps371.OverlayValues[370] = d370
							return bbs[14].RenderPS(ps371)
						}
						if ps.General {
							ctx.SyncDesc(&d7)
							if d7.Loc == LocReg {
								ctx.ProtectReg(d7.Reg)
							} else if d7.Loc == LocRegPair {
								ctx.ProtectReg(d7.Reg)
								ctx.ProtectReg(d7.Reg2)
							}
							d372 = d7
							if d372.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d372)
							ctx.EmitStoreToStack(d372, int32(bbs[4].PhiBase)+int32(0))
							if d7.Loc == LocReg {
								ctx.UnprotectReg(d7.Reg)
							} else if d7.Loc == LocRegPair {
								ctx.UnprotectReg(d7.Reg)
								ctx.UnprotectReg(d7.Reg2)
							}
						}
						ps373 := PhiState{General: ps.General}
						ps373.OverlayValues = make([]JITValueDesc, 373)
						ps373.OverlayValues[1] = d1
						ps373.OverlayValues[2] = d2
						ps373.OverlayValues[3] = d3
						ps373.OverlayValues[4] = d4
						ps373.OverlayValues[5] = d5
						ps373.OverlayValues[6] = d6
						ps373.OverlayValues[7] = d7
						ps373.OverlayValues[8] = d8
						ps373.OverlayValues[9] = d9
						ps373.OverlayValues[10] = d10
						ps373.OverlayValues[11] = d11
						ps373.OverlayValues[12] = d12
						ps373.OverlayValues[13] = d13
						ps373.OverlayValues[14] = d14
						ps373.OverlayValues[15] = d15
						ps373.OverlayValues[18] = d18
						ps373.OverlayValues[21] = d21
						ps373.OverlayValues[40] = d40
						ps373.OverlayValues[41] = d41
						ps373.OverlayValues[42] = d42
						ps373.OverlayValues[43] = d43
						ps373.OverlayValues[44] = d44
						ps373.OverlayValues[46] = d46
						ps373.OverlayValues[47] = d47
						ps373.OverlayValues[48] = d48
						ps373.OverlayValues[49] = d49
						ps373.OverlayValues[50] = d50
						ps373.OverlayValues[51] = d51
						ps373.OverlayValues[52] = d52
						ps373.OverlayValues[55] = d55
						ps373.OverlayValues[90] = d90
						ps373.OverlayValues[91] = d91
						ps373.OverlayValues[92] = d92
						ps373.OverlayValues[93] = d93
						ps373.OverlayValues[94] = d94
						ps373.OverlayValues[95] = d95
						ps373.OverlayValues[97] = d97
						ps373.OverlayValues[98] = d98
						ps373.OverlayValues[99] = d99
						ps373.OverlayValues[100] = d100
						ps373.OverlayValues[101] = d101
						ps373.OverlayValues[102] = d102
						ps373.OverlayValues[103] = d103
						ps373.OverlayValues[104] = d104
						ps373.OverlayValues[105] = d105
						ps373.OverlayValues[108] = d108
						ps373.OverlayValues[109] = d109
						ps373.OverlayValues[110] = d110
						ps373.OverlayValues[111] = d111
						ps373.OverlayValues[164] = d164
						ps373.OverlayValues[165] = d165
						ps373.OverlayValues[166] = d166
						ps373.OverlayValues[167] = d167
						ps373.OverlayValues[168] = d168
						ps373.OverlayValues[169] = d169
						ps373.OverlayValues[170] = d170
						ps373.OverlayValues[171] = d171
						ps373.OverlayValues[172] = d172
						ps373.OverlayValues[173] = d173
						ps373.OverlayValues[174] = d174
						ps373.OverlayValues[175] = d175
						ps373.OverlayValues[176] = d176
						ps373.OverlayValues[177] = d177
						ps373.OverlayValues[178] = d178
						ps373.OverlayValues[180] = d180
						ps373.OverlayValues[181] = d181
						ps373.OverlayValues[182] = d182
						ps373.OverlayValues[183] = d183
						ps373.OverlayValues[185] = d185
						ps373.OverlayValues[186] = d186
						ps373.OverlayValues[187] = d187
						ps373.OverlayValues[264] = d264
						ps373.OverlayValues[265] = d265
						ps373.OverlayValues[266] = d266
						ps373.OverlayValues[267] = d267
						ps373.OverlayValues[268] = d268
						ps373.OverlayValues[271] = d271
						ps373.OverlayValues[272] = d272
						ps373.OverlayValues[354] = d354
						ps373.OverlayValues[355] = d355
						ps373.OverlayValues[356] = d356
						ps373.OverlayValues[357] = d357
						ps373.OverlayValues[358] = d358
						ps373.OverlayValues[359] = d359
						ps373.OverlayValues[360] = d360
						ps373.OverlayValues[361] = d361
						ps373.OverlayValues[362] = d362
						ps373.OverlayValues[363] = d363
						ps373.OverlayValues[364] = d364
						ps373.OverlayValues[366] = d366
						ps373.OverlayValues[367] = d367
						ps373.OverlayValues[368] = d368
						ps373.OverlayValues[369] = d369
						ps373.OverlayValues[370] = d370
						ps373.OverlayValues[372] = d372
						ps373.PhiValues = make([]JITValueDesc, 1)
						d374 = d7
						ps373.PhiValues[0] = d374
						return bbs[4].RenderPS(ps373)
					}
					if !ps.General {
						ps.General = true
						return bbs[12].RenderPS(ps)
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d370.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl27)
					ctx.SyncDesc(&d7)
					if d7.Loc == LocReg {
						ctx.ProtectReg(d7.Reg)
					} else if d7.Loc == LocRegPair {
						ctx.ProtectReg(d7.Reg)
						ctx.ProtectReg(d7.Reg2)
					}
					d375 = d7
					if d375.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d375)
					ctx.EmitStoreToStack(d375, int32(bbs[4].PhiBase)+int32(0))
					if d7.Loc == LocReg {
						ctx.UnprotectReg(d7.Reg)
					} else if d7.Loc == LocRegPair {
						ctx.UnprotectReg(d7.Reg)
						ctx.UnprotectReg(d7.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ps376 := PhiState{General: true}
					ps376.OverlayValues = make([]JITValueDesc, 376)
					ps376.OverlayValues[1] = d1
					ps376.OverlayValues[2] = d2
					ps376.OverlayValues[3] = d3
					ps376.OverlayValues[4] = d4
					ps376.OverlayValues[5] = d5
					ps376.OverlayValues[6] = d6
					ps376.OverlayValues[7] = d7
					ps376.OverlayValues[8] = d8
					ps376.OverlayValues[9] = d9
					ps376.OverlayValues[10] = d10
					ps376.OverlayValues[11] = d11
					ps376.OverlayValues[12] = d12
					ps376.OverlayValues[13] = d13
					ps376.OverlayValues[14] = d14
					ps376.OverlayValues[15] = d15
					ps376.OverlayValues[18] = d18
					ps376.OverlayValues[21] = d21
					ps376.OverlayValues[40] = d40
					ps376.OverlayValues[41] = d41
					ps376.OverlayValues[42] = d42
					ps376.OverlayValues[43] = d43
					ps376.OverlayValues[44] = d44
					ps376.OverlayValues[46] = d46
					ps376.OverlayValues[47] = d47
					ps376.OverlayValues[48] = d48
					ps376.OverlayValues[49] = d49
					ps376.OverlayValues[50] = d50
					ps376.OverlayValues[51] = d51
					ps376.OverlayValues[52] = d52
					ps376.OverlayValues[55] = d55
					ps376.OverlayValues[90] = d90
					ps376.OverlayValues[91] = d91
					ps376.OverlayValues[92] = d92
					ps376.OverlayValues[93] = d93
					ps376.OverlayValues[94] = d94
					ps376.OverlayValues[95] = d95
					ps376.OverlayValues[97] = d97
					ps376.OverlayValues[98] = d98
					ps376.OverlayValues[99] = d99
					ps376.OverlayValues[100] = d100
					ps376.OverlayValues[101] = d101
					ps376.OverlayValues[102] = d102
					ps376.OverlayValues[103] = d103
					ps376.OverlayValues[104] = d104
					ps376.OverlayValues[105] = d105
					ps376.OverlayValues[108] = d108
					ps376.OverlayValues[109] = d109
					ps376.OverlayValues[110] = d110
					ps376.OverlayValues[111] = d111
					ps376.OverlayValues[164] = d164
					ps376.OverlayValues[165] = d165
					ps376.OverlayValues[166] = d166
					ps376.OverlayValues[167] = d167
					ps376.OverlayValues[168] = d168
					ps376.OverlayValues[169] = d169
					ps376.OverlayValues[170] = d170
					ps376.OverlayValues[171] = d171
					ps376.OverlayValues[172] = d172
					ps376.OverlayValues[173] = d173
					ps376.OverlayValues[174] = d174
					ps376.OverlayValues[175] = d175
					ps376.OverlayValues[176] = d176
					ps376.OverlayValues[177] = d177
					ps376.OverlayValues[178] = d178
					ps376.OverlayValues[180] = d180
					ps376.OverlayValues[181] = d181
					ps376.OverlayValues[182] = d182
					ps376.OverlayValues[183] = d183
					ps376.OverlayValues[185] = d185
					ps376.OverlayValues[186] = d186
					ps376.OverlayValues[187] = d187
					ps376.OverlayValues[264] = d264
					ps376.OverlayValues[265] = d265
					ps376.OverlayValues[266] = d266
					ps376.OverlayValues[267] = d267
					ps376.OverlayValues[268] = d268
					ps376.OverlayValues[271] = d271
					ps376.OverlayValues[272] = d272
					ps376.OverlayValues[354] = d354
					ps376.OverlayValues[355] = d355
					ps376.OverlayValues[356] = d356
					ps376.OverlayValues[357] = d357
					ps376.OverlayValues[358] = d358
					ps376.OverlayValues[359] = d359
					ps376.OverlayValues[360] = d360
					ps376.OverlayValues[361] = d361
					ps376.OverlayValues[362] = d362
					ps376.OverlayValues[363] = d363
					ps376.OverlayValues[364] = d364
					ps376.OverlayValues[366] = d366
					ps376.OverlayValues[367] = d367
					ps376.OverlayValues[368] = d368
					ps376.OverlayValues[369] = d369
					ps376.OverlayValues[370] = d370
					ps376.OverlayValues[372] = d372
					ps376.OverlayValues[374] = d374
					ps376.OverlayValues[375] = d375
					ps377 := PhiState{General: true}
					ps377.OverlayValues = make([]JITValueDesc, 376)
					ps377.OverlayValues[1] = d1
					ps377.OverlayValues[2] = d2
					ps377.OverlayValues[3] = d3
					ps377.OverlayValues[4] = d4
					ps377.OverlayValues[5] = d5
					ps377.OverlayValues[6] = d6
					ps377.OverlayValues[7] = d7
					ps377.OverlayValues[8] = d8
					ps377.OverlayValues[9] = d9
					ps377.OverlayValues[10] = d10
					ps377.OverlayValues[11] = d11
					ps377.OverlayValues[12] = d12
					ps377.OverlayValues[13] = d13
					ps377.OverlayValues[14] = d14
					ps377.OverlayValues[15] = d15
					ps377.OverlayValues[18] = d18
					ps377.OverlayValues[21] = d21
					ps377.OverlayValues[40] = d40
					ps377.OverlayValues[41] = d41
					ps377.OverlayValues[42] = d42
					ps377.OverlayValues[43] = d43
					ps377.OverlayValues[44] = d44
					ps377.OverlayValues[46] = d46
					ps377.OverlayValues[47] = d47
					ps377.OverlayValues[48] = d48
					ps377.OverlayValues[49] = d49
					ps377.OverlayValues[50] = d50
					ps377.OverlayValues[51] = d51
					ps377.OverlayValues[52] = d52
					ps377.OverlayValues[55] = d55
					ps377.OverlayValues[90] = d90
					ps377.OverlayValues[91] = d91
					ps377.OverlayValues[92] = d92
					ps377.OverlayValues[93] = d93
					ps377.OverlayValues[94] = d94
					ps377.OverlayValues[95] = d95
					ps377.OverlayValues[97] = d97
					ps377.OverlayValues[98] = d98
					ps377.OverlayValues[99] = d99
					ps377.OverlayValues[100] = d100
					ps377.OverlayValues[101] = d101
					ps377.OverlayValues[102] = d102
					ps377.OverlayValues[103] = d103
					ps377.OverlayValues[104] = d104
					ps377.OverlayValues[105] = d105
					ps377.OverlayValues[108] = d108
					ps377.OverlayValues[109] = d109
					ps377.OverlayValues[110] = d110
					ps377.OverlayValues[111] = d111
					ps377.OverlayValues[164] = d164
					ps377.OverlayValues[165] = d165
					ps377.OverlayValues[166] = d166
					ps377.OverlayValues[167] = d167
					ps377.OverlayValues[168] = d168
					ps377.OverlayValues[169] = d169
					ps377.OverlayValues[170] = d170
					ps377.OverlayValues[171] = d171
					ps377.OverlayValues[172] = d172
					ps377.OverlayValues[173] = d173
					ps377.OverlayValues[174] = d174
					ps377.OverlayValues[175] = d175
					ps377.OverlayValues[176] = d176
					ps377.OverlayValues[177] = d177
					ps377.OverlayValues[178] = d178
					ps377.OverlayValues[180] = d180
					ps377.OverlayValues[181] = d181
					ps377.OverlayValues[182] = d182
					ps377.OverlayValues[183] = d183
					ps377.OverlayValues[185] = d185
					ps377.OverlayValues[186] = d186
					ps377.OverlayValues[187] = d187
					ps377.OverlayValues[264] = d264
					ps377.OverlayValues[265] = d265
					ps377.OverlayValues[266] = d266
					ps377.OverlayValues[267] = d267
					ps377.OverlayValues[268] = d268
					ps377.OverlayValues[271] = d271
					ps377.OverlayValues[272] = d272
					ps377.OverlayValues[354] = d354
					ps377.OverlayValues[355] = d355
					ps377.OverlayValues[356] = d356
					ps377.OverlayValues[357] = d357
					ps377.OverlayValues[358] = d358
					ps377.OverlayValues[359] = d359
					ps377.OverlayValues[360] = d360
					ps377.OverlayValues[361] = d361
					ps377.OverlayValues[362] = d362
					ps377.OverlayValues[363] = d363
					ps377.OverlayValues[364] = d364
					ps377.OverlayValues[366] = d366
					ps377.OverlayValues[367] = d367
					ps377.OverlayValues[368] = d368
					ps377.OverlayValues[369] = d369
					ps377.OverlayValues[370] = d370
					ps377.OverlayValues[372] = d372
					ps377.OverlayValues[374] = d374
					ps377.OverlayValues[375] = d375
					ps377.PhiValues = make([]JITValueDesc, 1)
					d378 = d7
					ps377.PhiValues[0] = d378
					snap379 := d1
					snap380 := d2
					snap381 := d3
					snap382 := d4
					snap383 := d5
					snap384 := d6
					snap385 := d7
					snap386 := d8
					snap387 := d9
					snap388 := d10
					snap389 := d11
					snap390 := d12
					snap391 := d13
					snap392 := d14
					snap393 := d15
					snap394 := d18
					snap395 := d21
					snap396 := d40
					snap397 := d41
					snap398 := d42
					snap399 := d43
					snap400 := d44
					snap401 := d46
					snap402 := d47
					snap403 := d48
					snap404 := d49
					snap405 := d50
					snap406 := d51
					snap407 := d52
					snap408 := d55
					snap409 := d90
					snap410 := d91
					snap411 := d92
					snap412 := d93
					snap413 := d94
					snap414 := d95
					snap415 := d97
					snap416 := d98
					snap417 := d99
					snap418 := d100
					snap419 := d101
					snap420 := d102
					snap421 := d103
					snap422 := d104
					snap423 := d105
					snap424 := d108
					snap425 := d109
					snap426 := d110
					snap427 := d111
					snap428 := d164
					snap429 := d165
					snap430 := d166
					snap431 := d167
					snap432 := d168
					snap433 := d169
					snap434 := d170
					snap435 := d171
					snap436 := d172
					snap437 := d173
					snap438 := d174
					snap439 := d175
					snap440 := d176
					snap441 := d177
					snap442 := d178
					snap443 := d180
					snap444 := d181
					snap445 := d182
					snap446 := d183
					snap447 := d185
					snap448 := d186
					snap449 := d187
					snap450 := d264
					snap451 := d265
					snap452 := d266
					snap453 := d267
					snap454 := d268
					snap455 := d271
					snap456 := d272
					snap457 := d354
					snap458 := d355
					snap459 := d356
					snap460 := d357
					snap461 := d358
					snap462 := d359
					snap463 := d360
					snap464 := d361
					snap465 := d362
					snap466 := d363
					snap467 := d364
					snap468 := d366
					snap469 := d367
					snap470 := d368
					snap471 := d369
					snap472 := d370
					snap473 := d372
					snap474 := d374
					snap475 := d375
					snap476 := d378
					alloc477 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps377)
					}
					ctx.RestoreAllocState(alloc477)
					d1 = snap379
					d2 = snap380
					d3 = snap381
					d4 = snap382
					d5 = snap383
					d6 = snap384
					d7 = snap385
					d8 = snap386
					d9 = snap387
					d10 = snap388
					d11 = snap389
					d12 = snap390
					d13 = snap391
					d14 = snap392
					d15 = snap393
					d18 = snap394
					d21 = snap395
					d40 = snap396
					d41 = snap397
					d42 = snap398
					d43 = snap399
					d44 = snap400
					d46 = snap401
					d47 = snap402
					d48 = snap403
					d49 = snap404
					d50 = snap405
					d51 = snap406
					d52 = snap407
					d55 = snap408
					d90 = snap409
					d91 = snap410
					d92 = snap411
					d93 = snap412
					d94 = snap413
					d95 = snap414
					d97 = snap415
					d98 = snap416
					d99 = snap417
					d100 = snap418
					d101 = snap419
					d102 = snap420
					d103 = snap421
					d104 = snap422
					d105 = snap423
					d108 = snap424
					d109 = snap425
					d110 = snap426
					d111 = snap427
					d164 = snap428
					d165 = snap429
					d166 = snap430
					d167 = snap431
					d168 = snap432
					d169 = snap433
					d170 = snap434
					d171 = snap435
					d172 = snap436
					d173 = snap437
					d174 = snap438
					d175 = snap439
					d176 = snap440
					d177 = snap441
					d178 = snap442
					d180 = snap443
					d181 = snap444
					d182 = snap445
					d183 = snap446
					d185 = snap447
					d186 = snap448
					d187 = snap449
					d264 = snap450
					d265 = snap451
					d266 = snap452
					d267 = snap453
					d268 = snap454
					d271 = snap455
					d272 = snap456
					d354 = snap457
					d355 = snap458
					d356 = snap459
					d357 = snap460
					d358 = snap461
					d359 = snap462
					d360 = snap463
					d361 = snap464
					d362 = snap465
					d363 = snap466
					d364 = snap467
					d366 = snap468
					d367 = snap469
					d368 = snap470
					d369 = snap471
					d370 = snap472
					d372 = snap473
					d374 = snap474
					d375 = snap475
					d378 = snap476
					if !bbs[14].Rendered {
						return bbs[14].RenderPS(ps376)
					}
					return result
					ctx.FreeDesc(&d369)
					return result
				}
				bbs[13].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[13].VisitCount >= 0 {
							ps.General = true
							return bbs[13].RenderPS(ps)
						}
					}
					bbs[13].VisitCount++
					if ps.General {
						if bbs[13].Rendered {
							ctx.EmitJmp(lbl14)
							return result
						}
						bbs[13].Rendered = true
						bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_13 = bbs[13].Address
						ctx.MarkLabel(lbl14)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != LocNone {
						d169 = ps.OverlayValues[169]
					}
					if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != LocNone {
						d170 = ps.OverlayValues[170]
					}
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != LocNone {
						d264 = ps.OverlayValues[264]
					}
					if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != LocNone {
						d265 = ps.OverlayValues[265]
					}
					if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != LocNone {
						d266 = ps.OverlayValues[266]
					}
					if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != LocNone {
						d267 = ps.OverlayValues[267]
					}
					if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != LocNone {
						d268 = ps.OverlayValues[268]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
					}
					if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != LocNone {
						d361 = ps.OverlayValues[361]
					}
					if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != LocNone {
						d362 = ps.OverlayValues[362]
					}
					if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != LocNone {
						d363 = ps.OverlayValues[363]
					}
					if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != LocNone {
						d364 = ps.OverlayValues[364]
					}
					if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != LocNone {
						d366 = ps.OverlayValues[366]
					}
					if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != LocNone {
						d367 = ps.OverlayValues[367]
					}
					if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != LocNone {
						d368 = ps.OverlayValues[368]
					}
					if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != LocNone {
						d369 = ps.OverlayValues[369]
					}
					if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != LocNone {
						d370 = ps.OverlayValues[370]
					}
					if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != LocNone {
						d372 = ps.OverlayValues[372]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					ctx.ReclaimUntrackedRegs()
					var d478 JITValueDesc
					if d12.SliceSizeKnown {
						d478 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.KnownSliceLen))}
					} else if d12.Loc == LocImm {
						d478 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.StackOff))}
					} else if d12.Loc == LocStackTriple {
						d478 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d12.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d12)
						if d12.Loc == LocRegPair || d12.Loc == LocRegTriple {
							d478 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg2, ID: 0}
						} else if d12.Loc == LocReg {
							d478 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d478)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d478)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d478)
					var d479 JITValueDesc
					if d8.Loc == LocImm && d478.Loc == LocImm {
						d479 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < d478.Imm.Int())}
					} else if d478.Loc == LocImm {
						r20 := ctx.AllocReg()
						if d478.Imm.Int() >= -2147483648 && d478.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d8.Reg, int32(d478.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d478.Imm.Int()))
							ctx.EmitCmpInt64(d8.Reg, RegR11)
						}
						ctx.EmitSetcc(r20, CondSignedLess)
						d479 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
						ctx.BindReg(r20, &d479)
					} else if d8.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d478.Reg)
						ctx.EmitSetcc(r21, CondSignedLess)
						d479 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
						ctx.BindReg(r21, &d479)
					} else {
						r22 := ctx.AllocReg()
						ctx.EmitCmpInt64(d8.Reg, d478.Reg)
						ctx.EmitSetcc(r22, CondSignedLess)
						d479 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
						ctx.BindReg(r22, &d479)
					}
					ctx.FreeDesc(&d8)
					ctx.FreeDesc(&d478)
					d480 = d479
					ctx.EnsureDesc(&d480)
					if d480.Loc != LocImm && d480.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d480.Loc == LocImm {
						if d480.Imm.Bool() {
							if ps.General {
							}
							ps481 := PhiState{General: ps.General}
							ps481.OverlayValues = make([]JITValueDesc, 481)
							ps481.OverlayValues[1] = d1
							ps481.OverlayValues[2] = d2
							ps481.OverlayValues[3] = d3
							ps481.OverlayValues[4] = d4
							ps481.OverlayValues[5] = d5
							ps481.OverlayValues[6] = d6
							ps481.OverlayValues[7] = d7
							ps481.OverlayValues[8] = d8
							ps481.OverlayValues[9] = d9
							ps481.OverlayValues[10] = d10
							ps481.OverlayValues[11] = d11
							ps481.OverlayValues[12] = d12
							ps481.OverlayValues[13] = d13
							ps481.OverlayValues[14] = d14
							ps481.OverlayValues[15] = d15
							ps481.OverlayValues[18] = d18
							ps481.OverlayValues[21] = d21
							ps481.OverlayValues[40] = d40
							ps481.OverlayValues[41] = d41
							ps481.OverlayValues[42] = d42
							ps481.OverlayValues[43] = d43
							ps481.OverlayValues[44] = d44
							ps481.OverlayValues[46] = d46
							ps481.OverlayValues[47] = d47
							ps481.OverlayValues[48] = d48
							ps481.OverlayValues[49] = d49
							ps481.OverlayValues[50] = d50
							ps481.OverlayValues[51] = d51
							ps481.OverlayValues[52] = d52
							ps481.OverlayValues[55] = d55
							ps481.OverlayValues[90] = d90
							ps481.OverlayValues[91] = d91
							ps481.OverlayValues[92] = d92
							ps481.OverlayValues[93] = d93
							ps481.OverlayValues[94] = d94
							ps481.OverlayValues[95] = d95
							ps481.OverlayValues[97] = d97
							ps481.OverlayValues[98] = d98
							ps481.OverlayValues[99] = d99
							ps481.OverlayValues[100] = d100
							ps481.OverlayValues[101] = d101
							ps481.OverlayValues[102] = d102
							ps481.OverlayValues[103] = d103
							ps481.OverlayValues[104] = d104
							ps481.OverlayValues[105] = d105
							ps481.OverlayValues[108] = d108
							ps481.OverlayValues[109] = d109
							ps481.OverlayValues[110] = d110
							ps481.OverlayValues[111] = d111
							ps481.OverlayValues[164] = d164
							ps481.OverlayValues[165] = d165
							ps481.OverlayValues[166] = d166
							ps481.OverlayValues[167] = d167
							ps481.OverlayValues[168] = d168
							ps481.OverlayValues[169] = d169
							ps481.OverlayValues[170] = d170
							ps481.OverlayValues[171] = d171
							ps481.OverlayValues[172] = d172
							ps481.OverlayValues[173] = d173
							ps481.OverlayValues[174] = d174
							ps481.OverlayValues[175] = d175
							ps481.OverlayValues[176] = d176
							ps481.OverlayValues[177] = d177
							ps481.OverlayValues[178] = d178
							ps481.OverlayValues[180] = d180
							ps481.OverlayValues[181] = d181
							ps481.OverlayValues[182] = d182
							ps481.OverlayValues[183] = d183
							ps481.OverlayValues[185] = d185
							ps481.OverlayValues[186] = d186
							ps481.OverlayValues[187] = d187
							ps481.OverlayValues[264] = d264
							ps481.OverlayValues[265] = d265
							ps481.OverlayValues[266] = d266
							ps481.OverlayValues[267] = d267
							ps481.OverlayValues[268] = d268
							ps481.OverlayValues[271] = d271
							ps481.OverlayValues[272] = d272
							ps481.OverlayValues[354] = d354
							ps481.OverlayValues[355] = d355
							ps481.OverlayValues[356] = d356
							ps481.OverlayValues[357] = d357
							ps481.OverlayValues[358] = d358
							ps481.OverlayValues[359] = d359
							ps481.OverlayValues[360] = d360
							ps481.OverlayValues[361] = d361
							ps481.OverlayValues[362] = d362
							ps481.OverlayValues[363] = d363
							ps481.OverlayValues[364] = d364
							ps481.OverlayValues[366] = d366
							ps481.OverlayValues[367] = d367
							ps481.OverlayValues[368] = d368
							ps481.OverlayValues[369] = d369
							ps481.OverlayValues[370] = d370
							ps481.OverlayValues[372] = d372
							ps481.OverlayValues[374] = d374
							ps481.OverlayValues[375] = d375
							ps481.OverlayValues[378] = d378
							ps481.OverlayValues[478] = d478
							ps481.OverlayValues[479] = d479
							ps481.OverlayValues[480] = d480
							return bbs[11].RenderPS(ps481)
						}
						if ps.General {
						}
						ps482 := PhiState{General: ps.General}
						ps482.OverlayValues = make([]JITValueDesc, 481)
						ps482.OverlayValues[1] = d1
						ps482.OverlayValues[2] = d2
						ps482.OverlayValues[3] = d3
						ps482.OverlayValues[4] = d4
						ps482.OverlayValues[5] = d5
						ps482.OverlayValues[6] = d6
						ps482.OverlayValues[7] = d7
						ps482.OverlayValues[8] = d8
						ps482.OverlayValues[9] = d9
						ps482.OverlayValues[10] = d10
						ps482.OverlayValues[11] = d11
						ps482.OverlayValues[12] = d12
						ps482.OverlayValues[13] = d13
						ps482.OverlayValues[14] = d14
						ps482.OverlayValues[15] = d15
						ps482.OverlayValues[18] = d18
						ps482.OverlayValues[21] = d21
						ps482.OverlayValues[40] = d40
						ps482.OverlayValues[41] = d41
						ps482.OverlayValues[42] = d42
						ps482.OverlayValues[43] = d43
						ps482.OverlayValues[44] = d44
						ps482.OverlayValues[46] = d46
						ps482.OverlayValues[47] = d47
						ps482.OverlayValues[48] = d48
						ps482.OverlayValues[49] = d49
						ps482.OverlayValues[50] = d50
						ps482.OverlayValues[51] = d51
						ps482.OverlayValues[52] = d52
						ps482.OverlayValues[55] = d55
						ps482.OverlayValues[90] = d90
						ps482.OverlayValues[91] = d91
						ps482.OverlayValues[92] = d92
						ps482.OverlayValues[93] = d93
						ps482.OverlayValues[94] = d94
						ps482.OverlayValues[95] = d95
						ps482.OverlayValues[97] = d97
						ps482.OverlayValues[98] = d98
						ps482.OverlayValues[99] = d99
						ps482.OverlayValues[100] = d100
						ps482.OverlayValues[101] = d101
						ps482.OverlayValues[102] = d102
						ps482.OverlayValues[103] = d103
						ps482.OverlayValues[104] = d104
						ps482.OverlayValues[105] = d105
						ps482.OverlayValues[108] = d108
						ps482.OverlayValues[109] = d109
						ps482.OverlayValues[110] = d110
						ps482.OverlayValues[111] = d111
						ps482.OverlayValues[164] = d164
						ps482.OverlayValues[165] = d165
						ps482.OverlayValues[166] = d166
						ps482.OverlayValues[167] = d167
						ps482.OverlayValues[168] = d168
						ps482.OverlayValues[169] = d169
						ps482.OverlayValues[170] = d170
						ps482.OverlayValues[171] = d171
						ps482.OverlayValues[172] = d172
						ps482.OverlayValues[173] = d173
						ps482.OverlayValues[174] = d174
						ps482.OverlayValues[175] = d175
						ps482.OverlayValues[176] = d176
						ps482.OverlayValues[177] = d177
						ps482.OverlayValues[178] = d178
						ps482.OverlayValues[180] = d180
						ps482.OverlayValues[181] = d181
						ps482.OverlayValues[182] = d182
						ps482.OverlayValues[183] = d183
						ps482.OverlayValues[185] = d185
						ps482.OverlayValues[186] = d186
						ps482.OverlayValues[187] = d187
						ps482.OverlayValues[264] = d264
						ps482.OverlayValues[265] = d265
						ps482.OverlayValues[266] = d266
						ps482.OverlayValues[267] = d267
						ps482.OverlayValues[268] = d268
						ps482.OverlayValues[271] = d271
						ps482.OverlayValues[272] = d272
						ps482.OverlayValues[354] = d354
						ps482.OverlayValues[355] = d355
						ps482.OverlayValues[356] = d356
						ps482.OverlayValues[357] = d357
						ps482.OverlayValues[358] = d358
						ps482.OverlayValues[359] = d359
						ps482.OverlayValues[360] = d360
						ps482.OverlayValues[361] = d361
						ps482.OverlayValues[362] = d362
						ps482.OverlayValues[363] = d363
						ps482.OverlayValues[364] = d364
						ps482.OverlayValues[366] = d366
						ps482.OverlayValues[367] = d367
						ps482.OverlayValues[368] = d368
						ps482.OverlayValues[369] = d369
						ps482.OverlayValues[370] = d370
						ps482.OverlayValues[372] = d372
						ps482.OverlayValues[374] = d374
						ps482.OverlayValues[375] = d375
						ps482.OverlayValues[378] = d378
						ps482.OverlayValues[478] = d478
						ps482.OverlayValues[479] = d479
						ps482.OverlayValues[480] = d480
						return bbs[12].RenderPS(ps482)
					}
					if !ps.General {
						ps.General = true
						return bbs[13].RenderPS(ps)
					}
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d480.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl28)
					ctx.EmitJmp(lbl29)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl13)
					ps483 := PhiState{General: true}
					ps483.OverlayValues = make([]JITValueDesc, 481)
					ps483.OverlayValues[1] = d1
					ps483.OverlayValues[2] = d2
					ps483.OverlayValues[3] = d3
					ps483.OverlayValues[4] = d4
					ps483.OverlayValues[5] = d5
					ps483.OverlayValues[6] = d6
					ps483.OverlayValues[7] = d7
					ps483.OverlayValues[8] = d8
					ps483.OverlayValues[9] = d9
					ps483.OverlayValues[10] = d10
					ps483.OverlayValues[11] = d11
					ps483.OverlayValues[12] = d12
					ps483.OverlayValues[13] = d13
					ps483.OverlayValues[14] = d14
					ps483.OverlayValues[15] = d15
					ps483.OverlayValues[18] = d18
					ps483.OverlayValues[21] = d21
					ps483.OverlayValues[40] = d40
					ps483.OverlayValues[41] = d41
					ps483.OverlayValues[42] = d42
					ps483.OverlayValues[43] = d43
					ps483.OverlayValues[44] = d44
					ps483.OverlayValues[46] = d46
					ps483.OverlayValues[47] = d47
					ps483.OverlayValues[48] = d48
					ps483.OverlayValues[49] = d49
					ps483.OverlayValues[50] = d50
					ps483.OverlayValues[51] = d51
					ps483.OverlayValues[52] = d52
					ps483.OverlayValues[55] = d55
					ps483.OverlayValues[90] = d90
					ps483.OverlayValues[91] = d91
					ps483.OverlayValues[92] = d92
					ps483.OverlayValues[93] = d93
					ps483.OverlayValues[94] = d94
					ps483.OverlayValues[95] = d95
					ps483.OverlayValues[97] = d97
					ps483.OverlayValues[98] = d98
					ps483.OverlayValues[99] = d99
					ps483.OverlayValues[100] = d100
					ps483.OverlayValues[101] = d101
					ps483.OverlayValues[102] = d102
					ps483.OverlayValues[103] = d103
					ps483.OverlayValues[104] = d104
					ps483.OverlayValues[105] = d105
					ps483.OverlayValues[108] = d108
					ps483.OverlayValues[109] = d109
					ps483.OverlayValues[110] = d110
					ps483.OverlayValues[111] = d111
					ps483.OverlayValues[164] = d164
					ps483.OverlayValues[165] = d165
					ps483.OverlayValues[166] = d166
					ps483.OverlayValues[167] = d167
					ps483.OverlayValues[168] = d168
					ps483.OverlayValues[169] = d169
					ps483.OverlayValues[170] = d170
					ps483.OverlayValues[171] = d171
					ps483.OverlayValues[172] = d172
					ps483.OverlayValues[173] = d173
					ps483.OverlayValues[174] = d174
					ps483.OverlayValues[175] = d175
					ps483.OverlayValues[176] = d176
					ps483.OverlayValues[177] = d177
					ps483.OverlayValues[178] = d178
					ps483.OverlayValues[180] = d180
					ps483.OverlayValues[181] = d181
					ps483.OverlayValues[182] = d182
					ps483.OverlayValues[183] = d183
					ps483.OverlayValues[185] = d185
					ps483.OverlayValues[186] = d186
					ps483.OverlayValues[187] = d187
					ps483.OverlayValues[264] = d264
					ps483.OverlayValues[265] = d265
					ps483.OverlayValues[266] = d266
					ps483.OverlayValues[267] = d267
					ps483.OverlayValues[268] = d268
					ps483.OverlayValues[271] = d271
					ps483.OverlayValues[272] = d272
					ps483.OverlayValues[354] = d354
					ps483.OverlayValues[355] = d355
					ps483.OverlayValues[356] = d356
					ps483.OverlayValues[357] = d357
					ps483.OverlayValues[358] = d358
					ps483.OverlayValues[359] = d359
					ps483.OverlayValues[360] = d360
					ps483.OverlayValues[361] = d361
					ps483.OverlayValues[362] = d362
					ps483.OverlayValues[363] = d363
					ps483.OverlayValues[364] = d364
					ps483.OverlayValues[366] = d366
					ps483.OverlayValues[367] = d367
					ps483.OverlayValues[368] = d368
					ps483.OverlayValues[369] = d369
					ps483.OverlayValues[370] = d370
					ps483.OverlayValues[372] = d372
					ps483.OverlayValues[374] = d374
					ps483.OverlayValues[375] = d375
					ps483.OverlayValues[378] = d378
					ps483.OverlayValues[478] = d478
					ps483.OverlayValues[479] = d479
					ps483.OverlayValues[480] = d480
					ps484 := PhiState{General: true}
					ps484.OverlayValues = make([]JITValueDesc, 481)
					ps484.OverlayValues[1] = d1
					ps484.OverlayValues[2] = d2
					ps484.OverlayValues[3] = d3
					ps484.OverlayValues[4] = d4
					ps484.OverlayValues[5] = d5
					ps484.OverlayValues[6] = d6
					ps484.OverlayValues[7] = d7
					ps484.OverlayValues[8] = d8
					ps484.OverlayValues[9] = d9
					ps484.OverlayValues[10] = d10
					ps484.OverlayValues[11] = d11
					ps484.OverlayValues[12] = d12
					ps484.OverlayValues[13] = d13
					ps484.OverlayValues[14] = d14
					ps484.OverlayValues[15] = d15
					ps484.OverlayValues[18] = d18
					ps484.OverlayValues[21] = d21
					ps484.OverlayValues[40] = d40
					ps484.OverlayValues[41] = d41
					ps484.OverlayValues[42] = d42
					ps484.OverlayValues[43] = d43
					ps484.OverlayValues[44] = d44
					ps484.OverlayValues[46] = d46
					ps484.OverlayValues[47] = d47
					ps484.OverlayValues[48] = d48
					ps484.OverlayValues[49] = d49
					ps484.OverlayValues[50] = d50
					ps484.OverlayValues[51] = d51
					ps484.OverlayValues[52] = d52
					ps484.OverlayValues[55] = d55
					ps484.OverlayValues[90] = d90
					ps484.OverlayValues[91] = d91
					ps484.OverlayValues[92] = d92
					ps484.OverlayValues[93] = d93
					ps484.OverlayValues[94] = d94
					ps484.OverlayValues[95] = d95
					ps484.OverlayValues[97] = d97
					ps484.OverlayValues[98] = d98
					ps484.OverlayValues[99] = d99
					ps484.OverlayValues[100] = d100
					ps484.OverlayValues[101] = d101
					ps484.OverlayValues[102] = d102
					ps484.OverlayValues[103] = d103
					ps484.OverlayValues[104] = d104
					ps484.OverlayValues[105] = d105
					ps484.OverlayValues[108] = d108
					ps484.OverlayValues[109] = d109
					ps484.OverlayValues[110] = d110
					ps484.OverlayValues[111] = d111
					ps484.OverlayValues[164] = d164
					ps484.OverlayValues[165] = d165
					ps484.OverlayValues[166] = d166
					ps484.OverlayValues[167] = d167
					ps484.OverlayValues[168] = d168
					ps484.OverlayValues[169] = d169
					ps484.OverlayValues[170] = d170
					ps484.OverlayValues[171] = d171
					ps484.OverlayValues[172] = d172
					ps484.OverlayValues[173] = d173
					ps484.OverlayValues[174] = d174
					ps484.OverlayValues[175] = d175
					ps484.OverlayValues[176] = d176
					ps484.OverlayValues[177] = d177
					ps484.OverlayValues[178] = d178
					ps484.OverlayValues[180] = d180
					ps484.OverlayValues[181] = d181
					ps484.OverlayValues[182] = d182
					ps484.OverlayValues[183] = d183
					ps484.OverlayValues[185] = d185
					ps484.OverlayValues[186] = d186
					ps484.OverlayValues[187] = d187
					ps484.OverlayValues[264] = d264
					ps484.OverlayValues[265] = d265
					ps484.OverlayValues[266] = d266
					ps484.OverlayValues[267] = d267
					ps484.OverlayValues[268] = d268
					ps484.OverlayValues[271] = d271
					ps484.OverlayValues[272] = d272
					ps484.OverlayValues[354] = d354
					ps484.OverlayValues[355] = d355
					ps484.OverlayValues[356] = d356
					ps484.OverlayValues[357] = d357
					ps484.OverlayValues[358] = d358
					ps484.OverlayValues[359] = d359
					ps484.OverlayValues[360] = d360
					ps484.OverlayValues[361] = d361
					ps484.OverlayValues[362] = d362
					ps484.OverlayValues[363] = d363
					ps484.OverlayValues[364] = d364
					ps484.OverlayValues[366] = d366
					ps484.OverlayValues[367] = d367
					ps484.OverlayValues[368] = d368
					ps484.OverlayValues[369] = d369
					ps484.OverlayValues[370] = d370
					ps484.OverlayValues[372] = d372
					ps484.OverlayValues[374] = d374
					ps484.OverlayValues[375] = d375
					ps484.OverlayValues[378] = d378
					ps484.OverlayValues[478] = d478
					ps484.OverlayValues[479] = d479
					ps484.OverlayValues[480] = d480
					snap485 := d1
					snap486 := d2
					snap487 := d3
					snap488 := d4
					snap489 := d5
					snap490 := d6
					snap491 := d7
					snap492 := d8
					snap493 := d9
					snap494 := d10
					snap495 := d11
					snap496 := d12
					snap497 := d13
					snap498 := d14
					snap499 := d15
					snap500 := d18
					snap501 := d21
					snap502 := d40
					snap503 := d41
					snap504 := d42
					snap505 := d43
					snap506 := d44
					snap507 := d46
					snap508 := d47
					snap509 := d48
					snap510 := d49
					snap511 := d50
					snap512 := d51
					snap513 := d52
					snap514 := d55
					snap515 := d90
					snap516 := d91
					snap517 := d92
					snap518 := d93
					snap519 := d94
					snap520 := d95
					snap521 := d97
					snap522 := d98
					snap523 := d99
					snap524 := d100
					snap525 := d101
					snap526 := d102
					snap527 := d103
					snap528 := d104
					snap529 := d105
					snap530 := d108
					snap531 := d109
					snap532 := d110
					snap533 := d111
					snap534 := d164
					snap535 := d165
					snap536 := d166
					snap537 := d167
					snap538 := d168
					snap539 := d169
					snap540 := d170
					snap541 := d171
					snap542 := d172
					snap543 := d173
					snap544 := d174
					snap545 := d175
					snap546 := d176
					snap547 := d177
					snap548 := d178
					snap549 := d180
					snap550 := d181
					snap551 := d182
					snap552 := d183
					snap553 := d185
					snap554 := d186
					snap555 := d187
					snap556 := d264
					snap557 := d265
					snap558 := d266
					snap559 := d267
					snap560 := d268
					snap561 := d271
					snap562 := d272
					snap563 := d354
					snap564 := d355
					snap565 := d356
					snap566 := d357
					snap567 := d358
					snap568 := d359
					snap569 := d360
					snap570 := d361
					snap571 := d362
					snap572 := d363
					snap573 := d364
					snap574 := d366
					snap575 := d367
					snap576 := d368
					snap577 := d369
					snap578 := d370
					snap579 := d372
					snap580 := d374
					snap581 := d375
					snap582 := d378
					snap583 := d478
					snap584 := d479
					snap585 := d480
					alloc586 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps484)
					}
					ctx.RestoreAllocState(alloc586)
					d1 = snap485
					d2 = snap486
					d3 = snap487
					d4 = snap488
					d5 = snap489
					d6 = snap490
					d7 = snap491
					d8 = snap492
					d9 = snap493
					d10 = snap494
					d11 = snap495
					d12 = snap496
					d13 = snap497
					d14 = snap498
					d15 = snap499
					d18 = snap500
					d21 = snap501
					d40 = snap502
					d41 = snap503
					d42 = snap504
					d43 = snap505
					d44 = snap506
					d46 = snap507
					d47 = snap508
					d48 = snap509
					d49 = snap510
					d50 = snap511
					d51 = snap512
					d52 = snap513
					d55 = snap514
					d90 = snap515
					d91 = snap516
					d92 = snap517
					d93 = snap518
					d94 = snap519
					d95 = snap520
					d97 = snap521
					d98 = snap522
					d99 = snap523
					d100 = snap524
					d101 = snap525
					d102 = snap526
					d103 = snap527
					d104 = snap528
					d105 = snap529
					d108 = snap530
					d109 = snap531
					d110 = snap532
					d111 = snap533
					d164 = snap534
					d165 = snap535
					d166 = snap536
					d167 = snap537
					d168 = snap538
					d169 = snap539
					d170 = snap540
					d171 = snap541
					d172 = snap542
					d173 = snap543
					d174 = snap544
					d175 = snap545
					d176 = snap546
					d177 = snap547
					d178 = snap548
					d180 = snap549
					d181 = snap550
					d182 = snap551
					d183 = snap552
					d185 = snap553
					d186 = snap554
					d187 = snap555
					d264 = snap556
					d265 = snap557
					d266 = snap558
					d267 = snap559
					d268 = snap560
					d271 = snap561
					d272 = snap562
					d354 = snap563
					d355 = snap564
					d356 = snap565
					d357 = snap566
					d358 = snap567
					d359 = snap568
					d360 = snap569
					d361 = snap570
					d362 = snap571
					d363 = snap572
					d364 = snap573
					d366 = snap574
					d367 = snap575
					d368 = snap576
					d369 = snap577
					d370 = snap578
					d372 = snap579
					d374 = snap580
					d375 = snap581
					d378 = snap582
					d478 = snap583
					d479 = snap584
					d480 = snap585
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps483)
					}
					return result
					ctx.FreeDesc(&d479)
					return result
				}
				bbs[14].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[14].VisitCount >= 0 {
							ps.General = true
							return bbs[14].RenderPS(ps)
						}
					}
					bbs[14].VisitCount++
					if ps.General {
						if bbs[14].Rendered {
							ctx.EmitJmp(lbl15)
							return result
						}
						bbs[14].Rendered = true
						bbs[14].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_14 = bbs[14].Address
						ctx.MarkLabel(lbl15)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					d7 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != LocNone {
						d164 = ps.OverlayValues[164]
					}
					if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != LocNone {
						d165 = ps.OverlayValues[165]
					}
					if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != LocNone {
						d166 = ps.OverlayValues[166]
					}
					if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != LocNone {
						d167 = ps.OverlayValues[167]
					}
					if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != LocNone {
						d168 = ps.OverlayValues[168]
					}
					if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != LocNone {
						d169 = ps.OverlayValues[169]
					}
					if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != LocNone {
						d170 = ps.OverlayValues[170]
					}
					if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != LocNone {
						d171 = ps.OverlayValues[171]
					}
					if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != LocNone {
						d172 = ps.OverlayValues[172]
					}
					if len(ps.OverlayValues) > 173 && ps.OverlayValues[173].Loc != LocNone {
						d173 = ps.OverlayValues[173]
					}
					if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != LocNone {
						d174 = ps.OverlayValues[174]
					}
					if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != LocNone {
						d175 = ps.OverlayValues[175]
					}
					if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != LocNone {
						d176 = ps.OverlayValues[176]
					}
					if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != LocNone {
						d177 = ps.OverlayValues[177]
					}
					if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != LocNone {
						d178 = ps.OverlayValues[178]
					}
					if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != LocNone {
						d180 = ps.OverlayValues[180]
					}
					if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != LocNone {
						d181 = ps.OverlayValues[181]
					}
					if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != LocNone {
						d182 = ps.OverlayValues[182]
					}
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != LocNone {
						d185 = ps.OverlayValues[185]
					}
					if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
						d186 = ps.OverlayValues[186]
					}
					if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
						d187 = ps.OverlayValues[187]
					}
					if len(ps.OverlayValues) > 264 && ps.OverlayValues[264].Loc != LocNone {
						d264 = ps.OverlayValues[264]
					}
					if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != LocNone {
						d265 = ps.OverlayValues[265]
					}
					if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != LocNone {
						d266 = ps.OverlayValues[266]
					}
					if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != LocNone {
						d267 = ps.OverlayValues[267]
					}
					if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != LocNone {
						d268 = ps.OverlayValues[268]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
						d354 = ps.OverlayValues[354]
					}
					if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
						d355 = ps.OverlayValues[355]
					}
					if len(ps.OverlayValues) > 356 && ps.OverlayValues[356].Loc != LocNone {
						d356 = ps.OverlayValues[356]
					}
					if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
						d357 = ps.OverlayValues[357]
					}
					if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
						d358 = ps.OverlayValues[358]
					}
					if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
						d359 = ps.OverlayValues[359]
					}
					if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
						d360 = ps.OverlayValues[360]
					}
					if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != LocNone {
						d361 = ps.OverlayValues[361]
					}
					if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != LocNone {
						d362 = ps.OverlayValues[362]
					}
					if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != LocNone {
						d363 = ps.OverlayValues[363]
					}
					if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != LocNone {
						d364 = ps.OverlayValues[364]
					}
					if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != LocNone {
						d366 = ps.OverlayValues[366]
					}
					if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != LocNone {
						d367 = ps.OverlayValues[367]
					}
					if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != LocNone {
						d368 = ps.OverlayValues[368]
					}
					if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != LocNone {
						d369 = ps.OverlayValues[369]
					}
					if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != LocNone {
						d370 = ps.OverlayValues[370]
					}
					if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != LocNone {
						d372 = ps.OverlayValues[372]
					}
					if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != LocNone {
						d374 = ps.OverlayValues[374]
					}
					if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != LocNone {
						d375 = ps.OverlayValues[375]
					}
					if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != LocNone {
						d378 = ps.OverlayValues[378]
					}
					if len(ps.OverlayValues) > 478 && ps.OverlayValues[478].Loc != LocNone {
						d478 = ps.OverlayValues[478]
					}
					if len(ps.OverlayValues) > 479 && ps.OverlayValues[479].Loc != LocNone {
						d479 = ps.OverlayValues[479]
					}
					if len(ps.OverlayValues) > 480 && ps.OverlayValues[480].Loc != LocNone {
						d480 = ps.OverlayValues[480]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					var d587 JITValueDesc
					if d7.Loc == LocImm {
						d587 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d7.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d7)
						var d588 JITValueDesc
						if d7.Loc == LocRegPair {
							ctx.FreeReg(d7.Reg)
							d588 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d7.Reg2}
							ctx.BindReg(d7.Reg2, &d588)
							ctx.BindReg(d7.Reg2, &d588)
						} else {
							d588 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d7.Reg}
							ctx.BindReg(d7.Reg, &d588)
							ctx.BindReg(d7.Reg, &d588)
						}
						d587 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d588}, 1)
						d587.Type = tagFloat
						ctx.BindReg(d587.Reg, &d587)
					}
					ctx.StabilizeDescForControlFlow(&d587)
					if ps.General {
						ctx.SyncDesc(&d587)
						if d587.Loc == LocReg {
							ctx.ProtectReg(d587.Reg)
						} else if d587.Loc == LocRegPair {
							ctx.ProtectReg(d587.Reg)
							ctx.ProtectReg(d587.Reg2)
						}
						d589 = d587
						if d589.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d589)
						ctx.EmitStoreToStack(d589, int32(bbs[4].PhiBase)+int32(0))
						if d587.Loc == LocReg {
							ctx.UnprotectReg(d587.Reg)
						} else if d587.Loc == LocRegPair {
							ctx.UnprotectReg(d587.Reg)
							ctx.UnprotectReg(d587.Reg2)
						}
					}
					ps590 := PhiState{General: ps.General}
					ps590.OverlayValues = make([]JITValueDesc, 590)
					ps590.OverlayValues[1] = d1
					ps590.OverlayValues[2] = d2
					ps590.OverlayValues[3] = d3
					ps590.OverlayValues[4] = d4
					ps590.OverlayValues[5] = d5
					ps590.OverlayValues[6] = d6
					ps590.OverlayValues[7] = d7
					ps590.OverlayValues[8] = d8
					ps590.OverlayValues[9] = d9
					ps590.OverlayValues[10] = d10
					ps590.OverlayValues[11] = d11
					ps590.OverlayValues[12] = d12
					ps590.OverlayValues[13] = d13
					ps590.OverlayValues[14] = d14
					ps590.OverlayValues[15] = d15
					ps590.OverlayValues[18] = d18
					ps590.OverlayValues[21] = d21
					ps590.OverlayValues[40] = d40
					ps590.OverlayValues[41] = d41
					ps590.OverlayValues[42] = d42
					ps590.OverlayValues[43] = d43
					ps590.OverlayValues[44] = d44
					ps590.OverlayValues[46] = d46
					ps590.OverlayValues[47] = d47
					ps590.OverlayValues[48] = d48
					ps590.OverlayValues[49] = d49
					ps590.OverlayValues[50] = d50
					ps590.OverlayValues[51] = d51
					ps590.OverlayValues[52] = d52
					ps590.OverlayValues[55] = d55
					ps590.OverlayValues[90] = d90
					ps590.OverlayValues[91] = d91
					ps590.OverlayValues[92] = d92
					ps590.OverlayValues[93] = d93
					ps590.OverlayValues[94] = d94
					ps590.OverlayValues[95] = d95
					ps590.OverlayValues[97] = d97
					ps590.OverlayValues[98] = d98
					ps590.OverlayValues[99] = d99
					ps590.OverlayValues[100] = d100
					ps590.OverlayValues[101] = d101
					ps590.OverlayValues[102] = d102
					ps590.OverlayValues[103] = d103
					ps590.OverlayValues[104] = d104
					ps590.OverlayValues[105] = d105
					ps590.OverlayValues[108] = d108
					ps590.OverlayValues[109] = d109
					ps590.OverlayValues[110] = d110
					ps590.OverlayValues[111] = d111
					ps590.OverlayValues[164] = d164
					ps590.OverlayValues[165] = d165
					ps590.OverlayValues[166] = d166
					ps590.OverlayValues[167] = d167
					ps590.OverlayValues[168] = d168
					ps590.OverlayValues[169] = d169
					ps590.OverlayValues[170] = d170
					ps590.OverlayValues[171] = d171
					ps590.OverlayValues[172] = d172
					ps590.OverlayValues[173] = d173
					ps590.OverlayValues[174] = d174
					ps590.OverlayValues[175] = d175
					ps590.OverlayValues[176] = d176
					ps590.OverlayValues[177] = d177
					ps590.OverlayValues[178] = d178
					ps590.OverlayValues[180] = d180
					ps590.OverlayValues[181] = d181
					ps590.OverlayValues[182] = d182
					ps590.OverlayValues[183] = d183
					ps590.OverlayValues[185] = d185
					ps590.OverlayValues[186] = d186
					ps590.OverlayValues[187] = d187
					ps590.OverlayValues[264] = d264
					ps590.OverlayValues[265] = d265
					ps590.OverlayValues[266] = d266
					ps590.OverlayValues[267] = d267
					ps590.OverlayValues[268] = d268
					ps590.OverlayValues[271] = d271
					ps590.OverlayValues[272] = d272
					ps590.OverlayValues[354] = d354
					ps590.OverlayValues[355] = d355
					ps590.OverlayValues[356] = d356
					ps590.OverlayValues[357] = d357
					ps590.OverlayValues[358] = d358
					ps590.OverlayValues[359] = d359
					ps590.OverlayValues[360] = d360
					ps590.OverlayValues[361] = d361
					ps590.OverlayValues[362] = d362
					ps590.OverlayValues[363] = d363
					ps590.OverlayValues[364] = d364
					ps590.OverlayValues[366] = d366
					ps590.OverlayValues[367] = d367
					ps590.OverlayValues[368] = d368
					ps590.OverlayValues[369] = d369
					ps590.OverlayValues[370] = d370
					ps590.OverlayValues[372] = d372
					ps590.OverlayValues[374] = d374
					ps590.OverlayValues[375] = d375
					ps590.OverlayValues[378] = d378
					ps590.OverlayValues[478] = d478
					ps590.OverlayValues[479] = d479
					ps590.OverlayValues[480] = d480
					ps590.OverlayValues[587] = d587
					ps590.OverlayValues[588] = d588
					ps590.OverlayValues[589] = d589
					ps590.PhiValues = make([]JITValueDesc, 1)
					d591 = d587
					ps590.PhiValues[0] = d591
					if ps590.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps590)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps592 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps592)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(128))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  80,
		},
	})
}
