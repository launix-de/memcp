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
				declaration := declarations["dot"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
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
				var d24 JITValueDesc
				_ = d24
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d30 JITValueDesc
				_ = d30
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d66 JITValueDesc
				_ = d66
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				var d114 JITValueDesc
				_ = d114
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d179 JITValueDesc
				_ = d179
				var d180 JITValueDesc
				_ = d180
				var d181 JITValueDesc
				_ = d181
				var d182 JITValueDesc
				_ = d182
				var d183 JITValueDesc
				_ = d183
				var d184 JITValueDesc
				_ = d184
				var d185 JITValueDesc
				_ = d185
				var d186 JITValueDesc
				_ = d186
				var d187 JITValueDesc
				_ = d187
				var d188 JITValueDesc
				_ = d188
				var d189 JITValueDesc
				_ = d189
				var d190 JITValueDesc
				_ = d190
				var d191 JITValueDesc
				_ = d191
				var d192 JITValueDesc
				_ = d192
				var d193 JITValueDesc
				_ = d193
				var d194 JITValueDesc
				_ = d194
				var d195 JITValueDesc
				_ = d195
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d199 JITValueDesc
				_ = d199
				var d200 JITValueDesc
				_ = d200
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d203 JITValueDesc
				_ = d203
				var d204 JITValueDesc
				_ = d204
				var d205 JITValueDesc
				_ = d205
				var d206 JITValueDesc
				_ = d206
				var d208 JITValueDesc
				_ = d208
				var d209 JITValueDesc
				_ = d209
				var d210 JITValueDesc
				_ = d210
				var d297 JITValueDesc
				_ = d297
				var d298 JITValueDesc
				_ = d298
				var d299 JITValueDesc
				_ = d299
				var d300 JITValueDesc
				_ = d300
				var d301 JITValueDesc
				_ = d301
				var d304 JITValueDesc
				_ = d304
				var d305 JITValueDesc
				_ = d305
				var d397 JITValueDesc
				_ = d397
				var d398 JITValueDesc
				_ = d398
				var d399 JITValueDesc
				_ = d399
				var d400 JITValueDesc
				_ = d400
				var d401 JITValueDesc
				_ = d401
				var d402 JITValueDesc
				_ = d402
				var d403 JITValueDesc
				_ = d403
				var d404 JITValueDesc
				_ = d404
				var d405 JITValueDesc
				_ = d405
				var d406 JITValueDesc
				_ = d406
				var d407 JITValueDesc
				_ = d407
				var d408 JITValueDesc
				_ = d408
				var d409 JITValueDesc
				_ = d409
				var d411 JITValueDesc
				_ = d411
				var d412 JITValueDesc
				_ = d412
				var d413 JITValueDesc
				_ = d413
				var d414 JITValueDesc
				_ = d414
				var d415 JITValueDesc
				_ = d415
				var d416 JITValueDesc
				_ = d416
				var d417 JITValueDesc
				_ = d417
				var d419 JITValueDesc
				_ = d419
				var d421 JITValueDesc
				_ = d421
				var d422 JITValueDesc
				_ = d422
				var d425 JITValueDesc
				_ = d425
				var d539 JITValueDesc
				_ = d539
				var d540 JITValueDesc
				_ = d540
				var d541 JITValueDesc
				_ = d541
				var d662 JITValueDesc
				_ = d662
				var d663 JITValueDesc
				_ = d663
				var d664 JITValueDesc
				_ = d664
				var d666 JITValueDesc
				_ = d666
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(128))
				var bbs [15]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[2].PhiCount = uint16(1)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[4].PhiCount = uint16(1)
				bbs[6].PhiBase = int32(phiBase0) + int32(32)
				bbs[6].PhiCount = uint16(4)
				bbs[10].PhiBase = int32(phiBase0) + int32(96)
				bbs[10].PhiCount = uint16(2)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 64}, {Color: 1, Width: 1, Cost: 35}, {Color: 2, Width: 1, Cost: 17}, {Color: 3, Width: 1, Cost: 17}}, Count: 4})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[1]
				}
				var r1 Reg
				phiHomeOK3 := registerHomes1.Available&(uint16(1)<<3) == uint16(1)<<3
				if phiHomeOK3 {
					r1 = registerHomes1.Registers[3]
				}
				var r2 Reg
				phiHomeOK4 := registerHomes1.Available&(uint16(1)<<2) == uint16(1)<<2
				if phiHomeOK4 {
					r2 = registerHomes1.Registers[2]
				}
				var r3 Reg
				phiHomeOK5 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK5 {
					r3 = registerHomes1.Registers[0]
				}
				var r4 Reg
				phiHomeOK6 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
				if phiHomeOK6 {
					r4 = registerHomes1.Registers[1]
				}
				var r5 Reg
				phiHomeOK7 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK7 {
					r5 = registerHomes1.Registers[0]
				}
				d8 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d8
				d9 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
				_ = d9
				var d10 JITValueDesc
				if phiHomeOK2 {
					d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
					ctx.BindReg(r0, &d10)
				} else {
					d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
				}
				_ = d10
				var d11 JITValueDesc
				if phiHomeOK3 {
					d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
					ctx.BindReg(r1, &d11)
				} else {
					d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
				}
				_ = d11
				var d12 JITValueDesc
				if phiHomeOK4 {
					d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
					ctx.BindReg(r2, &d12)
				} else {
					d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
				}
				_ = d12
				var d13 JITValueDesc
				if phiHomeOK5 {
					d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
					ctx.BindReg(r3, &d13)
				} else {
					d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
				}
				_ = d13
				var d14 JITValueDesc
				if phiHomeOK6 {
					d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
					ctx.BindReg(r4, &d14)
				} else {
					d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
				}
				_ = d14
				var d15 JITValueDesc
				if phiHomeOK7 {
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
					ctx.BindReg(r5, &d15)
				} else {
					d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
				}
				_ = d15
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				resultRegsProtected := result.Loc == LocRegPair
				if resultRegsProtected {
					ctx.ProtectReg(result.Reg)
					ctx.ProtectReg(result.Reg2)
				}
				lbl0 := ctx.ReserveLabel()
				bbpos_0_0 := int32(-1)
				_ = bbpos_0_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_0_1 := int32(-1)
				_ = bbpos_0_1
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_0_2 := int32(-1)
				_ = bbpos_0_2
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_0_3 := int32(-1)
				_ = bbpos_0_3
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_0_13 := int32(-1)
				_ = bbpos_0_13
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_0_14 := int32(-1)
				_ = bbpos_0_14
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					ctx.ReclaimUntrackedRegs()
					d16 = args[0]
					d16.ID = 0
					var d17 JITValueDesc
					if d16.Type == tagSlice {
						d17 = jitKnownSliceHeader(ctx, &d16)
					} else {
						d17 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d16}, 3)
					}
					ctx.BindReg(d17.Reg, &d17)
					ctx.BindReg(d17.Reg2, &d17)
					ctx.BindReg(d17.Reg3, &d17)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.FreeDesc(&d16)
					d18 = args[1]
					d18.ID = 0
					var d19 JITValueDesc
					if d18.Type == tagSlice {
						d19 = jitKnownSliceHeader(ctx, &d18)
					} else {
						d19 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d18}, 3)
					}
					ctx.BindReg(d19.Reg, &d19)
					ctx.BindReg(d19.Reg2, &d19)
					ctx.BindReg(d19.Reg3, &d19)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d18)
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d20)
					var d21 JITValueDesc
					if d20.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.Int() > 2)}
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d20.Reg, 2)
						ctx.EmitSetcc(r6, CondSignedGreater)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
						ctx.BindReg(r6, &d21)
					}
					ctx.FreeDesc(&d20)
					d22 = d21
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							if ps.General {
							}
							ps23 := PhiState{General: ps.General}
							ps23.OverlayValues = make([]JITValueDesc, 23)
							ps23.OverlayValues[8] = d8
							ps23.OverlayValues[9] = d9
							ps23.OverlayValues[10] = d10
							ps23.OverlayValues[11] = d11
							ps23.OverlayValues[12] = d12
							ps23.OverlayValues[13] = d13
							ps23.OverlayValues[14] = d14
							ps23.OverlayValues[15] = d15
							ps23.OverlayValues[16] = d16
							ps23.OverlayValues[17] = d17
							ps23.OverlayValues[18] = d18
							ps23.OverlayValues[19] = d19
							ps23.OverlayValues[20] = d20
							ps23.OverlayValues[21] = d21
							ps23.OverlayValues[22] = d22
							return bbs[1].RenderPS(ps23)
						}
						if ps.General {
							d24 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
							ctx.EmitStoreScmerToStack(d24, int32(bbs[2].PhiBase)+int32(0))
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 25)
						ps25.OverlayValues[8] = d8
						ps25.OverlayValues[9] = d9
						ps25.OverlayValues[10] = d10
						ps25.OverlayValues[11] = d11
						ps25.OverlayValues[12] = d12
						ps25.OverlayValues[13] = d13
						ps25.OverlayValues[14] = d14
						ps25.OverlayValues[15] = d15
						ps25.OverlayValues[16] = d16
						ps25.OverlayValues[17] = d17
						ps25.OverlayValues[18] = d18
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[24] = d24
						ps25.PhiValues = make([]JITValueDesc, 1)
						d26 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
						ps25.PhiValues[0] = d26
						return bbs[2].RenderPS(ps25)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl17)
					d27 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
					ctx.EmitStoreScmerToStack(d27, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 28)
					ps28.OverlayValues[8] = d8
					ps28.OverlayValues[9] = d9
					ps28.OverlayValues[10] = d10
					ps28.OverlayValues[11] = d11
					ps28.OverlayValues[12] = d12
					ps28.OverlayValues[13] = d13
					ps28.OverlayValues[14] = d14
					ps28.OverlayValues[15] = d15
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[18] = d18
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[24] = d24
					ps28.OverlayValues[26] = d26
					ps28.OverlayValues[27] = d27
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 28)
					ps29.OverlayValues[8] = d8
					ps29.OverlayValues[9] = d9
					ps29.OverlayValues[10] = d10
					ps29.OverlayValues[11] = d11
					ps29.OverlayValues[12] = d12
					ps29.OverlayValues[13] = d13
					ps29.OverlayValues[14] = d14
					ps29.OverlayValues[15] = d15
					ps29.OverlayValues[16] = d16
					ps29.OverlayValues[17] = d17
					ps29.OverlayValues[18] = d18
					ps29.OverlayValues[19] = d19
					ps29.OverlayValues[20] = d20
					ps29.OverlayValues[21] = d21
					ps29.OverlayValues[22] = d22
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[26] = d26
					ps29.OverlayValues[27] = d27
					ps29.PhiValues = make([]JITValueDesc, 1)
					d30 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
					ps29.PhiValues[0] = d30
					snap31 := d8
					snap32 := d9
					snap33 := d10
					snap34 := d11
					snap35 := d12
					snap36 := d13
					snap37 := d14
					snap38 := d15
					snap39 := d16
					snap40 := d17
					snap41 := d18
					snap42 := d19
					snap43 := d20
					snap44 := d21
					snap45 := d22
					snap46 := d24
					snap47 := d26
					snap48 := d27
					snap49 := d30
					alloc50 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps29)
					}
					ctx.RestoreAllocState(alloc50)
					d8 = snap31
					d9 = snap32
					d10 = snap33
					d11 = snap34
					d12 = snap35
					d13 = snap36
					d14 = snap37
					d15 = snap38
					d16 = snap39
					d17 = snap40
					d18 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d24 = snap46
					d26 = snap47
					d27 = snap48
					d30 = snap49
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps28)
					}
					return result
					ctx.FreeDesc(&d21)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					ctx.ReclaimUntrackedRegs()
					d51 = args[2]
					d51.ID = 0
					d53 = d51
					ctx.SyncDesc(&d53)
					if d53.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d53.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d53.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d53 = tmpScalar
					}
					d53 = JITPrepareScmerGoArg(ctx, d53)
					if d53.Loc != LocRegPair && d53.Loc != LocStackPair && d53.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d52 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d53}, 2)
					ctx.FreeDesc(&d51)
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d52)
					if d52.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d52.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d52.Imm)
						ptrWord, _ := d52.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d52.Imm.String())))
						d52 = tmpPair
					} else if d52.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d52.Type, Reg: ctx.AllocRegExcept(d52.Reg), Reg2: ctx.AllocRegExcept(d52.Reg)}
						switch d52.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d52)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d52)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d52)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d52)
						d52 = tmpPair
					}
					if d52.Loc != LocRegPair && d52.Loc != LocStackPair && d52.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d52)
					d54 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d52}, 2)
					d54.NoHeapPointer = false
					ctx.BindReg(d54.Reg, &d54)
					ctx.BindReg(d54.Reg2, &d54)
					ctx.StabilizeDescForControlFlow(&d54)
					if ps.General {
						ctx.SyncDesc(&d54)
						if d54.Loc == LocReg {
							ctx.ProtectReg(d54.Reg)
						} else if d54.Loc == LocRegPair {
							ctx.ProtectReg(d54.Reg)
							ctx.ProtectReg(d54.Reg2)
						}
						d55 = d54
						if d55.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EmitStoreScmerToStack(d55, int32(bbs[2].PhiBase)+int32(0))
						if d54.Loc == LocReg {
							ctx.UnprotectReg(d54.Reg)
						} else if d54.Loc == LocRegPair {
							ctx.UnprotectReg(d54.Reg)
							ctx.UnprotectReg(d54.Reg2)
						}
					}
					ps56 := PhiState{General: ps.General}
					ps56.OverlayValues = make([]JITValueDesc, 56)
					ps56.OverlayValues[8] = d8
					ps56.OverlayValues[9] = d9
					ps56.OverlayValues[10] = d10
					ps56.OverlayValues[11] = d11
					ps56.OverlayValues[12] = d12
					ps56.OverlayValues[13] = d13
					ps56.OverlayValues[14] = d14
					ps56.OverlayValues[15] = d15
					ps56.OverlayValues[16] = d16
					ps56.OverlayValues[17] = d17
					ps56.OverlayValues[18] = d18
					ps56.OverlayValues[19] = d19
					ps56.OverlayValues[20] = d20
					ps56.OverlayValues[21] = d21
					ps56.OverlayValues[22] = d22
					ps56.OverlayValues[24] = d24
					ps56.OverlayValues[26] = d26
					ps56.OverlayValues[27] = d27
					ps56.OverlayValues[30] = d30
					ps56.OverlayValues[51] = d51
					ps56.OverlayValues[52] = d52
					ps56.OverlayValues[53] = d53
					ps56.OverlayValues[54] = d54
					ps56.OverlayValues[55] = d55
					ps56.PhiValues = make([]JITValueDesc, 1)
					d57 = d54
					ps56.PhiValues[0] = d57
					if ps56.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps56)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d58 := ps.PhiValues[0]
							ctx.EmitStoreScmerToStack(d58, int32(bbs[2].PhiBase)+int32(0))
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d8 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d8)
					var d59 JITValueDesc
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d59 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d59.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d59.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d59.Reg, &d59)
						ctx.BindReg(d59.Reg2, &d59)
					} else {
						d59 = d8
					}
					d60 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("COSINE")}
					var d61 JITValueDesc
					if d60.Loc == LocImm {
						ctx.TrackImm(d60.Imm)
						ptrWord, _ := d60.Imm.RawWords()
						d61 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d61.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d61.Reg2, uint64(len(d60.Imm.String())))
						ctx.BindReg(d61.Reg, &d61)
						ctx.BindReg(d61.Reg2, &d61)
					} else {
						d61 = d60
					}
					d62 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d59, d61}, 1)
					ctx.EmitAndRegImm32(d62.Reg, 1)
					d62.Type = tagBool
					ctx.BindReg(d62.Reg, &d62)
					d63 = d62
					ctx.EnsureDesc(&d63)
					if d63.Loc != LocImm && d63.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d63.Loc == LocImm {
						if d63.Imm.Bool() {
							if ps.General {
							}
							ps64 := PhiState{General: ps.General}
							ps64.OverlayValues = make([]JITValueDesc, 64)
							ps64.OverlayValues[8] = d8
							ps64.OverlayValues[9] = d9
							ps64.OverlayValues[10] = d10
							ps64.OverlayValues[11] = d11
							ps64.OverlayValues[12] = d12
							ps64.OverlayValues[13] = d13
							ps64.OverlayValues[14] = d14
							ps64.OverlayValues[15] = d15
							ps64.OverlayValues[16] = d16
							ps64.OverlayValues[17] = d17
							ps64.OverlayValues[18] = d18
							ps64.OverlayValues[19] = d19
							ps64.OverlayValues[20] = d20
							ps64.OverlayValues[21] = d21
							ps64.OverlayValues[22] = d22
							ps64.OverlayValues[24] = d24
							ps64.OverlayValues[26] = d26
							ps64.OverlayValues[27] = d27
							ps64.OverlayValues[30] = d30
							ps64.OverlayValues[51] = d51
							ps64.OverlayValues[52] = d52
							ps64.OverlayValues[53] = d53
							ps64.OverlayValues[54] = d54
							ps64.OverlayValues[55] = d55
							ps64.OverlayValues[57] = d57
							ps64.OverlayValues[58] = d58
							ps64.OverlayValues[59] = d59
							ps64.OverlayValues[60] = d60
							ps64.OverlayValues[61] = d61
							ps64.OverlayValues[62] = d62
							ps64.OverlayValues[63] = d63
							return bbs[3].RenderPS(ps64)
						}
						if ps.General {
						}
						ps65 := PhiState{General: ps.General}
						ps65.OverlayValues = make([]JITValueDesc, 64)
						ps65.OverlayValues[8] = d8
						ps65.OverlayValues[9] = d9
						ps65.OverlayValues[10] = d10
						ps65.OverlayValues[11] = d11
						ps65.OverlayValues[12] = d12
						ps65.OverlayValues[13] = d13
						ps65.OverlayValues[14] = d14
						ps65.OverlayValues[15] = d15
						ps65.OverlayValues[16] = d16
						ps65.OverlayValues[17] = d17
						ps65.OverlayValues[18] = d18
						ps65.OverlayValues[19] = d19
						ps65.OverlayValues[20] = d20
						ps65.OverlayValues[21] = d21
						ps65.OverlayValues[22] = d22
						ps65.OverlayValues[24] = d24
						ps65.OverlayValues[26] = d26
						ps65.OverlayValues[27] = d27
						ps65.OverlayValues[30] = d30
						ps65.OverlayValues[51] = d51
						ps65.OverlayValues[52] = d52
						ps65.OverlayValues[53] = d53
						ps65.OverlayValues[54] = d54
						ps65.OverlayValues[55] = d55
						ps65.OverlayValues[57] = d57
						ps65.OverlayValues[58] = d58
						ps65.OverlayValues[59] = d59
						ps65.OverlayValues[60] = d60
						ps65.OverlayValues[61] = d61
						ps65.OverlayValues[62] = d62
						ps65.OverlayValues[63] = d63
						return bbs[5].RenderPS(ps65)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d66 := ps.PhiValues[0]
							ctx.EmitStoreScmerToStack(d66, int32(bbs[2].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d63.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl6)
					ps67 := PhiState{General: true}
					ps67.OverlayValues = make([]JITValueDesc, 67)
					ps67.OverlayValues[8] = d8
					ps67.OverlayValues[9] = d9
					ps67.OverlayValues[10] = d10
					ps67.OverlayValues[11] = d11
					ps67.OverlayValues[12] = d12
					ps67.OverlayValues[13] = d13
					ps67.OverlayValues[14] = d14
					ps67.OverlayValues[15] = d15
					ps67.OverlayValues[16] = d16
					ps67.OverlayValues[17] = d17
					ps67.OverlayValues[18] = d18
					ps67.OverlayValues[19] = d19
					ps67.OverlayValues[20] = d20
					ps67.OverlayValues[21] = d21
					ps67.OverlayValues[22] = d22
					ps67.OverlayValues[24] = d24
					ps67.OverlayValues[26] = d26
					ps67.OverlayValues[27] = d27
					ps67.OverlayValues[30] = d30
					ps67.OverlayValues[51] = d51
					ps67.OverlayValues[52] = d52
					ps67.OverlayValues[53] = d53
					ps67.OverlayValues[54] = d54
					ps67.OverlayValues[55] = d55
					ps67.OverlayValues[57] = d57
					ps67.OverlayValues[58] = d58
					ps67.OverlayValues[59] = d59
					ps67.OverlayValues[60] = d60
					ps67.OverlayValues[61] = d61
					ps67.OverlayValues[62] = d62
					ps67.OverlayValues[63] = d63
					ps67.OverlayValues[66] = d66
					ps68 := PhiState{General: true}
					ps68.OverlayValues = make([]JITValueDesc, 67)
					ps68.OverlayValues[8] = d8
					ps68.OverlayValues[9] = d9
					ps68.OverlayValues[10] = d10
					ps68.OverlayValues[11] = d11
					ps68.OverlayValues[12] = d12
					ps68.OverlayValues[13] = d13
					ps68.OverlayValues[14] = d14
					ps68.OverlayValues[15] = d15
					ps68.OverlayValues[16] = d16
					ps68.OverlayValues[17] = d17
					ps68.OverlayValues[18] = d18
					ps68.OverlayValues[19] = d19
					ps68.OverlayValues[20] = d20
					ps68.OverlayValues[21] = d21
					ps68.OverlayValues[22] = d22
					ps68.OverlayValues[24] = d24
					ps68.OverlayValues[26] = d26
					ps68.OverlayValues[27] = d27
					ps68.OverlayValues[30] = d30
					ps68.OverlayValues[51] = d51
					ps68.OverlayValues[52] = d52
					ps68.OverlayValues[53] = d53
					ps68.OverlayValues[54] = d54
					ps68.OverlayValues[55] = d55
					ps68.OverlayValues[57] = d57
					ps68.OverlayValues[58] = d58
					ps68.OverlayValues[59] = d59
					ps68.OverlayValues[60] = d60
					ps68.OverlayValues[61] = d61
					ps68.OverlayValues[62] = d62
					ps68.OverlayValues[63] = d63
					ps68.OverlayValues[66] = d66
					snap69 := d8
					snap70 := d9
					snap71 := d10
					snap72 := d11
					snap73 := d12
					snap74 := d13
					snap75 := d14
					snap76 := d15
					snap77 := d16
					snap78 := d17
					snap79 := d18
					snap80 := d19
					snap81 := d20
					snap82 := d21
					snap83 := d22
					snap84 := d24
					snap85 := d26
					snap86 := d27
					snap87 := d30
					snap88 := d51
					snap89 := d52
					snap90 := d53
					snap91 := d54
					snap92 := d55
					snap93 := d57
					snap94 := d58
					snap95 := d59
					snap96 := d60
					snap97 := d61
					snap98 := d62
					snap99 := d63
					snap100 := d66
					alloc101 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps68)
					}
					ctx.RestoreAllocState(alloc101)
					d8 = snap69
					d9 = snap70
					d10 = snap71
					d11 = snap72
					d12 = snap73
					d13 = snap74
					d14 = snap75
					d15 = snap76
					d16 = snap77
					d17 = snap78
					d18 = snap79
					d19 = snap80
					d20 = snap81
					d21 = snap82
					d22 = snap83
					d24 = snap84
					d26 = snap85
					d27 = snap86
					d30 = snap87
					d51 = snap88
					d52 = snap89
					d53 = snap90
					d54 = snap91
					d55 = snap92
					d57 = snap93
					d58 = snap94
					d59 = snap95
					d60 = snap96
					d61 = snap97
					d62 = snap98
					d63 = snap99
					d66 = snap100
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps67)
					}
					return result
					ctx.FreeDesc(&d62)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
						}
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}, int32(bbs[6].PhiBase)+int32(16))
						}
						if phiHomeOK4 {
							ctx.EmitMovToReg(r2, JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}, int32(bbs[6].PhiBase)+int32(32))
						}
						if phiHomeOK5 {
							ctx.EmitMovToReg(r3, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(48))
						}
					}
					ps102 := PhiState{General: ps.General}
					ps102.OverlayValues = make([]JITValueDesc, 67)
					ps102.OverlayValues[8] = d8
					ps102.OverlayValues[9] = d9
					ps102.OverlayValues[10] = d10
					ps102.OverlayValues[11] = d11
					ps102.OverlayValues[12] = d12
					ps102.OverlayValues[13] = d13
					ps102.OverlayValues[14] = d14
					ps102.OverlayValues[15] = d15
					ps102.OverlayValues[16] = d16
					ps102.OverlayValues[17] = d17
					ps102.OverlayValues[18] = d18
					ps102.OverlayValues[19] = d19
					ps102.OverlayValues[20] = d20
					ps102.OverlayValues[21] = d21
					ps102.OverlayValues[22] = d22
					ps102.OverlayValues[24] = d24
					ps102.OverlayValues[26] = d26
					ps102.OverlayValues[27] = d27
					ps102.OverlayValues[30] = d30
					ps102.OverlayValues[51] = d51
					ps102.OverlayValues[52] = d52
					ps102.OverlayValues[53] = d53
					ps102.OverlayValues[54] = d54
					ps102.OverlayValues[55] = d55
					ps102.OverlayValues[57] = d57
					ps102.OverlayValues[58] = d58
					ps102.OverlayValues[59] = d59
					ps102.OverlayValues[60] = d60
					ps102.OverlayValues[61] = d61
					ps102.OverlayValues[62] = d62
					ps102.OverlayValues[63] = d63
					ps102.OverlayValues[66] = d66
					ps102.PhiValues = make([]JITValueDesc, 4)
					d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps102.PhiValues[0] = d103
					d104 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps102.PhiValues[1] = d104
					d105 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps102.PhiValues[2] = d105
					d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps102.PhiValues[3] = d106
					if ps102.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps102)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d107 := ps.PhiValues[0]
							ctx.EmitStoreToStack(d107, int32(bbs[4].PhiBase)+int32(0))
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d9 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						ctx.EmitMakeFloat(result, d9)
					} else {
						ctx.EmitMovToReg(result.Reg2, d9)
						d108 := JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d108)
						if d9.Loc == LocReg && d9.Reg != result.Reg2 {
							ctx.FreeReg(d9.Reg)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						if phiHomeOK6 {
							ctx.EmitMovToReg(r4, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[10].PhiBase)+int32(0))
						}
						if phiHomeOK7 {
							ctx.EmitMovToReg(r5, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[10].PhiBase)+int32(16))
						}
					}
					ps109 := PhiState{General: ps.General}
					ps109.OverlayValues = make([]JITValueDesc, 109)
					ps109.OverlayValues[8] = d8
					ps109.OverlayValues[9] = d9
					ps109.OverlayValues[10] = d10
					ps109.OverlayValues[11] = d11
					ps109.OverlayValues[12] = d12
					ps109.OverlayValues[13] = d13
					ps109.OverlayValues[14] = d14
					ps109.OverlayValues[15] = d15
					ps109.OverlayValues[16] = d16
					ps109.OverlayValues[17] = d17
					ps109.OverlayValues[18] = d18
					ps109.OverlayValues[19] = d19
					ps109.OverlayValues[20] = d20
					ps109.OverlayValues[21] = d21
					ps109.OverlayValues[22] = d22
					ps109.OverlayValues[24] = d24
					ps109.OverlayValues[26] = d26
					ps109.OverlayValues[27] = d27
					ps109.OverlayValues[30] = d30
					ps109.OverlayValues[51] = d51
					ps109.OverlayValues[52] = d52
					ps109.OverlayValues[53] = d53
					ps109.OverlayValues[54] = d54
					ps109.OverlayValues[55] = d55
					ps109.OverlayValues[57] = d57
					ps109.OverlayValues[58] = d58
					ps109.OverlayValues[59] = d59
					ps109.OverlayValues[60] = d60
					ps109.OverlayValues[61] = d61
					ps109.OverlayValues[62] = d62
					ps109.OverlayValues[63] = d63
					ps109.OverlayValues[66] = d66
					ps109.OverlayValues[103] = d103
					ps109.OverlayValues[104] = d104
					ps109.OverlayValues[105] = d105
					ps109.OverlayValues[106] = d106
					ps109.OverlayValues[107] = d107
					ps109.OverlayValues[108] = d108
					ps109.PhiValues = make([]JITValueDesc, 2)
					d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps109.PhiValues[0] = d110
					d111 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps109.PhiValues[1] = d111
					if ps109.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps109)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d112 := ps.PhiValues[0]
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d112)
							} else {
								ctx.EmitStoreToStack(d112, int32(bbs[6].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d113 := ps.PhiValues[1]
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d113)
							} else {
								ctx.EmitStoreToStack(d113, int32(bbs[6].PhiBase)+int32(16))
							}
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d114 := ps.PhiValues[2]
							if phiHomeOK4 {
								ctx.EmitMovToReg(r2, d114)
							} else {
								ctx.EmitStoreToStack(d114, int32(bbs[6].PhiBase)+int32(32))
							}
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d115 := ps.PhiValues[3]
							if phiHomeOK5 {
								ctx.EmitMovToReg(r3, d115)
							} else {
								ctx.EmitStoreToStack(d115, int32(bbs[6].PhiBase)+int32(48))
							}
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d10 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d11 = ps.PhiValues[1]
					}
					if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
						d12 = ps.PhiValues[2]
					}
					if !ps.General && len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
						d13 = ps.PhiValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					var d116 JITValueDesc
					if d17.SliceSizeKnown {
						d116 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d116 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d116 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d116 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d116 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d116)
					ctx.EnsureDescsTogether(&d13, &d116)
					var d117 JITValueDesc
					if d13.Loc == LocImm && d116.Loc == LocImm {
						d117 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d116.Imm.Int())}
					} else if d116.Loc == LocImm {
						r7 := ctx.AllocRegExcept(d13.Reg)
						if d116.Imm.Int() >= -2147483648 && d116.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d116.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d116.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, RegR11)
						}
						ctx.EmitSetcc(r7, CondSignedLess)
						d117 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
						ctx.BindReg(r7, &d117)
					} else if d13.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d116.Reg)
						ctx.EmitSetcc(r8, CondSignedLess)
						d117 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d117)
					} else {
						r9 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpInt64(d13.Reg, d116.Reg)
						ctx.EmitSetcc(r9, CondSignedLess)
						d117 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d117)
					}
					ctx.FreeDesc(&d116)
					d118 = d117
					ctx.EnsureDesc(&d118)
					if d118.Loc != LocImm && d118.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d118.Loc == LocImm {
						if d118.Imm.Bool() {
							if ps.General {
							}
							ps119 := PhiState{General: ps.General}
							ps119.OverlayValues = make([]JITValueDesc, 119)
							ps119.OverlayValues[8] = d8
							ps119.OverlayValues[9] = d9
							ps119.OverlayValues[10] = d10
							ps119.OverlayValues[11] = d11
							ps119.OverlayValues[12] = d12
							ps119.OverlayValues[13] = d13
							ps119.OverlayValues[14] = d14
							ps119.OverlayValues[15] = d15
							ps119.OverlayValues[16] = d16
							ps119.OverlayValues[17] = d17
							ps119.OverlayValues[18] = d18
							ps119.OverlayValues[19] = d19
							ps119.OverlayValues[20] = d20
							ps119.OverlayValues[21] = d21
							ps119.OverlayValues[22] = d22
							ps119.OverlayValues[24] = d24
							ps119.OverlayValues[26] = d26
							ps119.OverlayValues[27] = d27
							ps119.OverlayValues[30] = d30
							ps119.OverlayValues[51] = d51
							ps119.OverlayValues[52] = d52
							ps119.OverlayValues[53] = d53
							ps119.OverlayValues[54] = d54
							ps119.OverlayValues[55] = d55
							ps119.OverlayValues[57] = d57
							ps119.OverlayValues[58] = d58
							ps119.OverlayValues[59] = d59
							ps119.OverlayValues[60] = d60
							ps119.OverlayValues[61] = d61
							ps119.OverlayValues[62] = d62
							ps119.OverlayValues[63] = d63
							ps119.OverlayValues[66] = d66
							ps119.OverlayValues[103] = d103
							ps119.OverlayValues[104] = d104
							ps119.OverlayValues[105] = d105
							ps119.OverlayValues[106] = d106
							ps119.OverlayValues[107] = d107
							ps119.OverlayValues[108] = d108
							ps119.OverlayValues[110] = d110
							ps119.OverlayValues[111] = d111
							ps119.OverlayValues[112] = d112
							ps119.OverlayValues[113] = d113
							ps119.OverlayValues[114] = d114
							ps119.OverlayValues[115] = d115
							ps119.OverlayValues[116] = d116
							ps119.OverlayValues[117] = d117
							ps119.OverlayValues[118] = d118
							return bbs[9].RenderPS(ps119)
						}
						if ps.General {
						}
						ps120 := PhiState{General: ps.General}
						ps120.OverlayValues = make([]JITValueDesc, 119)
						ps120.OverlayValues[8] = d8
						ps120.OverlayValues[9] = d9
						ps120.OverlayValues[10] = d10
						ps120.OverlayValues[11] = d11
						ps120.OverlayValues[12] = d12
						ps120.OverlayValues[13] = d13
						ps120.OverlayValues[14] = d14
						ps120.OverlayValues[15] = d15
						ps120.OverlayValues[16] = d16
						ps120.OverlayValues[17] = d17
						ps120.OverlayValues[18] = d18
						ps120.OverlayValues[19] = d19
						ps120.OverlayValues[20] = d20
						ps120.OverlayValues[21] = d21
						ps120.OverlayValues[22] = d22
						ps120.OverlayValues[24] = d24
						ps120.OverlayValues[26] = d26
						ps120.OverlayValues[27] = d27
						ps120.OverlayValues[30] = d30
						ps120.OverlayValues[51] = d51
						ps120.OverlayValues[52] = d52
						ps120.OverlayValues[53] = d53
						ps120.OverlayValues[54] = d54
						ps120.OverlayValues[55] = d55
						ps120.OverlayValues[57] = d57
						ps120.OverlayValues[58] = d58
						ps120.OverlayValues[59] = d59
						ps120.OverlayValues[60] = d60
						ps120.OverlayValues[61] = d61
						ps120.OverlayValues[62] = d62
						ps120.OverlayValues[63] = d63
						ps120.OverlayValues[66] = d66
						ps120.OverlayValues[103] = d103
						ps120.OverlayValues[104] = d104
						ps120.OverlayValues[105] = d105
						ps120.OverlayValues[106] = d106
						ps120.OverlayValues[107] = d107
						ps120.OverlayValues[108] = d108
						ps120.OverlayValues[110] = d110
						ps120.OverlayValues[111] = d111
						ps120.OverlayValues[112] = d112
						ps120.OverlayValues[113] = d113
						ps120.OverlayValues[114] = d114
						ps120.OverlayValues[115] = d115
						ps120.OverlayValues[116] = d116
						ps120.OverlayValues[117] = d117
						ps120.OverlayValues[118] = d118
						return bbs[8].RenderPS(ps120)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d121 := ps.PhiValues[0]
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d121)
							} else {
								ctx.EmitStoreToStack(d121, int32(bbs[6].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d122 := ps.PhiValues[1]
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d122)
							} else {
								ctx.EmitStoreToStack(d122, int32(bbs[6].PhiBase)+int32(16))
							}
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d123 := ps.PhiValues[2]
							if phiHomeOK4 {
								ctx.EmitMovToReg(r2, d123)
							} else {
								ctx.EmitStoreToStack(d123, int32(bbs[6].PhiBase)+int32(32))
							}
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d124 := ps.PhiValues[3]
							if phiHomeOK5 {
								ctx.EmitMovToReg(r3, d124)
							} else {
								ctx.EmitStoreToStack(d124, int32(bbs[6].PhiBase)+int32(48))
							}
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d118.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl9)
					ps125 := PhiState{General: true}
					ps125.OverlayValues = make([]JITValueDesc, 125)
					ps125.OverlayValues[8] = d8
					ps125.OverlayValues[9] = d9
					ps125.OverlayValues[10] = d10
					ps125.OverlayValues[11] = d11
					ps125.OverlayValues[12] = d12
					ps125.OverlayValues[13] = d13
					ps125.OverlayValues[14] = d14
					ps125.OverlayValues[15] = d15
					ps125.OverlayValues[16] = d16
					ps125.OverlayValues[17] = d17
					ps125.OverlayValues[18] = d18
					ps125.OverlayValues[19] = d19
					ps125.OverlayValues[20] = d20
					ps125.OverlayValues[21] = d21
					ps125.OverlayValues[22] = d22
					ps125.OverlayValues[24] = d24
					ps125.OverlayValues[26] = d26
					ps125.OverlayValues[27] = d27
					ps125.OverlayValues[30] = d30
					ps125.OverlayValues[51] = d51
					ps125.OverlayValues[52] = d52
					ps125.OverlayValues[53] = d53
					ps125.OverlayValues[54] = d54
					ps125.OverlayValues[55] = d55
					ps125.OverlayValues[57] = d57
					ps125.OverlayValues[58] = d58
					ps125.OverlayValues[59] = d59
					ps125.OverlayValues[60] = d60
					ps125.OverlayValues[61] = d61
					ps125.OverlayValues[62] = d62
					ps125.OverlayValues[63] = d63
					ps125.OverlayValues[66] = d66
					ps125.OverlayValues[103] = d103
					ps125.OverlayValues[104] = d104
					ps125.OverlayValues[105] = d105
					ps125.OverlayValues[106] = d106
					ps125.OverlayValues[107] = d107
					ps125.OverlayValues[108] = d108
					ps125.OverlayValues[110] = d110
					ps125.OverlayValues[111] = d111
					ps125.OverlayValues[112] = d112
					ps125.OverlayValues[113] = d113
					ps125.OverlayValues[114] = d114
					ps125.OverlayValues[115] = d115
					ps125.OverlayValues[116] = d116
					ps125.OverlayValues[117] = d117
					ps125.OverlayValues[118] = d118
					ps125.OverlayValues[121] = d121
					ps125.OverlayValues[122] = d122
					ps125.OverlayValues[123] = d123
					ps125.OverlayValues[124] = d124
					ps126 := PhiState{General: true}
					ps126.OverlayValues = make([]JITValueDesc, 125)
					ps126.OverlayValues[8] = d8
					ps126.OverlayValues[9] = d9
					ps126.OverlayValues[10] = d10
					ps126.OverlayValues[11] = d11
					ps126.OverlayValues[12] = d12
					ps126.OverlayValues[13] = d13
					ps126.OverlayValues[14] = d14
					ps126.OverlayValues[15] = d15
					ps126.OverlayValues[16] = d16
					ps126.OverlayValues[17] = d17
					ps126.OverlayValues[18] = d18
					ps126.OverlayValues[19] = d19
					ps126.OverlayValues[20] = d20
					ps126.OverlayValues[21] = d21
					ps126.OverlayValues[22] = d22
					ps126.OverlayValues[24] = d24
					ps126.OverlayValues[26] = d26
					ps126.OverlayValues[27] = d27
					ps126.OverlayValues[30] = d30
					ps126.OverlayValues[51] = d51
					ps126.OverlayValues[52] = d52
					ps126.OverlayValues[53] = d53
					ps126.OverlayValues[54] = d54
					ps126.OverlayValues[55] = d55
					ps126.OverlayValues[57] = d57
					ps126.OverlayValues[58] = d58
					ps126.OverlayValues[59] = d59
					ps126.OverlayValues[60] = d60
					ps126.OverlayValues[61] = d61
					ps126.OverlayValues[62] = d62
					ps126.OverlayValues[63] = d63
					ps126.OverlayValues[66] = d66
					ps126.OverlayValues[103] = d103
					ps126.OverlayValues[104] = d104
					ps126.OverlayValues[105] = d105
					ps126.OverlayValues[106] = d106
					ps126.OverlayValues[107] = d107
					ps126.OverlayValues[108] = d108
					ps126.OverlayValues[110] = d110
					ps126.OverlayValues[111] = d111
					ps126.OverlayValues[112] = d112
					ps126.OverlayValues[113] = d113
					ps126.OverlayValues[114] = d114
					ps126.OverlayValues[115] = d115
					ps126.OverlayValues[116] = d116
					ps126.OverlayValues[117] = d117
					ps126.OverlayValues[118] = d118
					ps126.OverlayValues[121] = d121
					ps126.OverlayValues[122] = d122
					ps126.OverlayValues[123] = d123
					ps126.OverlayValues[124] = d124
					snap127 := d8
					snap128 := d9
					snap129 := d10
					snap130 := d11
					snap131 := d12
					snap132 := d13
					snap133 := d14
					snap134 := d15
					snap135 := d16
					snap136 := d17
					snap137 := d18
					snap138 := d19
					snap139 := d20
					snap140 := d21
					snap141 := d22
					snap142 := d24
					snap143 := d26
					snap144 := d27
					snap145 := d30
					snap146 := d51
					snap147 := d52
					snap148 := d53
					snap149 := d54
					snap150 := d55
					snap151 := d57
					snap152 := d58
					snap153 := d59
					snap154 := d60
					snap155 := d61
					snap156 := d62
					snap157 := d63
					snap158 := d66
					snap159 := d103
					snap160 := d104
					snap161 := d105
					snap162 := d106
					snap163 := d107
					snap164 := d108
					snap165 := d110
					snap166 := d111
					snap167 := d112
					snap168 := d113
					snap169 := d114
					snap170 := d115
					snap171 := d116
					snap172 := d117
					snap173 := d118
					snap174 := d121
					snap175 := d122
					snap176 := d123
					snap177 := d124
					alloc178 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps126)
					}
					ctx.RestoreAllocState(alloc178)
					d8 = snap127
					d9 = snap128
					d10 = snap129
					d11 = snap130
					d12 = snap131
					d13 = snap132
					d14 = snap133
					d15 = snap134
					d16 = snap135
					d17 = snap136
					d18 = snap137
					d19 = snap138
					d20 = snap139
					d21 = snap140
					d22 = snap141
					d24 = snap142
					d26 = snap143
					d27 = snap144
					d30 = snap145
					d51 = snap146
					d52 = snap147
					d53 = snap148
					d54 = snap149
					d55 = snap150
					d57 = snap151
					d58 = snap152
					d59 = snap153
					d60 = snap154
					d61 = snap155
					d62 = snap156
					d63 = snap157
					d66 = snap158
					d103 = snap159
					d104 = snap160
					d105 = snap161
					d106 = snap162
					d107 = snap163
					d108 = snap164
					d110 = snap165
					d111 = snap166
					d112 = snap167
					d113 = snap168
					d114 = snap169
					d115 = snap170
					d116 = snap171
					d117 = snap172
					d118 = snap173
					d121 = snap174
					d122 = snap175
					d123 = snap176
					d124 = snap177
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps125)
					}
					return result
					ctx.FreeDesc(&d117)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					d180 = ctx.EmitSliceElementAddress(&d17, &d13, 16)
					ctx.EnsureDesc(&d180)
					r10 := ctx.AllocRegExcept(d180.Reg)
					ctx.EmitMovRegMem(r10, d180.Reg, 8)
					ctx.EmitMovRegMem(d180.Reg, d180.Reg, 0)
					d179 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d180.Reg, Reg2: r10}
					ctx.BindReg(d180.Reg, &d179)
					ctx.BindReg(r10, &d179)
					ctx.EnsureDesc(&d179)
					d181 = d179
					_ = d181
					ctx.StabilizeDescForControlFlow(&d181)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl22 := ctx.ReserveLabel()
					_ = lbl22
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d182 JITValueDesc
					if d181.Loc == LocImm {
						d182 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d181.Imm.Float())}
					} else if d181.Type == tagFloat && d181.Loc == LocReg {
						d182 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d181.Reg}
						ctx.BindReg(d181.Reg, &d182)
						ctx.BindReg(d181.Reg, &d182)
					} else if d181.Type == tagFloat && d181.Loc == LocRegPair {
						ctx.FreeReg(d181.Reg)
						d182 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d181.Reg2}
						ctx.BindReg(d181.Reg2, &d182)
						ctx.BindReg(d181.Reg2, &d182)
					} else {
						d182 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d181}, 1)
						d182.Type = tagFloat
						ctx.BindReg(d182.Reg, &d182)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d182)
					ctx.FreeDesc(&d179)
					ctx.EnsureDesc(&d13)
					d184 = ctx.EmitSliceElementAddress(&d19, &d13, 16)
					ctx.EnsureDesc(&d184)
					r11 := ctx.AllocRegExcept(d184.Reg)
					ctx.EmitMovRegMem(r11, d184.Reg, 8)
					ctx.EmitMovRegMem(d184.Reg, d184.Reg, 0)
					d183 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d184.Reg, Reg2: r11}
					ctx.BindReg(d184.Reg, &d183)
					ctx.BindReg(r11, &d183)
					ctx.EnsureDesc(&d183)
					d185 = d183
					_ = d185
					ctx.StabilizeDescForControlFlow(&d185)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl23 := ctx.ReserveLabel()
					_ = lbl23
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d186 JITValueDesc
					if d185.Loc == LocImm {
						d186 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d185.Imm.Float())}
					} else if d185.Type == tagFloat && d185.Loc == LocReg {
						d186 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d185.Reg}
						ctx.BindReg(d185.Reg, &d186)
						ctx.BindReg(d185.Reg, &d186)
					} else if d185.Type == tagFloat && d185.Loc == LocRegPair {
						ctx.FreeReg(d185.Reg)
						d186 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d185.Reg2}
						ctx.BindReg(d185.Reg2, &d186)
						ctx.BindReg(d185.Reg2, &d186)
					} else {
						d186 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d185}, 1)
						d186.Type = tagFloat
						ctx.BindReg(d186.Reg, &d186)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d186)
					ctx.FreeDesc(&d183)
					ctx.EnsureDesc(&d182)
					ctx.EnsureDesc(&d182)
					ctx.EnsureDescsTogether(&d182, &d182)
					var d187 JITValueDesc
					if d182.Loc == LocImm {
						d187 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d182.Imm.Float() * d182.Imm.Float())}
					} else if d182.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d182.Reg)
						_, xBits := d182.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d182.Reg)
						d187 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d187)
					} else if d182.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d182.Reg)
						ctx.EmitMovRegReg(scratch, d182.Reg)
						_, yBits := d182.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d187 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d187)
					} else {
						r12 := ctx.AllocRegExcept(d182.Reg, d182.Reg)
						ctx.EmitMovRegReg(r12, d182.Reg)
						ctx.EmitMulFloat64(r12, d182.Reg)
						d187 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
						ctx.BindReg(r12, &d187)
					}
					if d187.Loc == LocReg && d182.Loc == LocReg && d187.Reg == d182.Reg {
						ctx.TransferReg(d182.Reg)
						d182.Loc = LocNone
					}
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d187)
					ctx.EnsureDescsTogether(&d11, &d187)
					var d188 JITValueDesc
					if d11.Loc == LocImm && d187.Loc == LocImm {
						d188 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d11.Imm.Float() + d187.Imm.Float())}
					} else if d11.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 && r1 != d187.Reg {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d187.Reg)
						}
						_, xBits := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d187.Reg)
						d188 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d188)
					} else if d187.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d11.Reg)
						}
						ctx.EmitMovRegReg(scratch, d11.Reg)
						_, yBits := d187.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d188 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d188)
					} else {
						var r13 Reg
						if phiHomeOK3 && r1 != d187.Reg {
							r13 = r1
						} else {
							r13 = ctx.AllocRegExcept(d11.Reg, d187.Reg)
						}
						ctx.EmitMovRegReg(r13, d11.Reg)
						ctx.EmitAddFloat64(r13, d187.Reg)
						d188 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r13}
						ctx.BindReg(r13, &d188)
					}
					if d188.Loc == LocReg && d11.Loc == LocReg && d188.Reg == d11.Reg {
						ctx.TransferReg(d11.Reg)
						d11.Loc = LocNone
					}
					ctx.FreeDesc(&d187)
					ctx.EnsureDesc(&d186)
					ctx.EnsureDesc(&d186)
					ctx.EnsureDescsTogether(&d186, &d186)
					var d189 JITValueDesc
					if d186.Loc == LocImm {
						d189 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d186.Imm.Float() * d186.Imm.Float())}
					} else if d186.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d186.Reg)
						_, xBits := d186.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d186.Reg)
						d189 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d189)
					} else if d186.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d186.Reg)
						ctx.EmitMovRegReg(scratch, d186.Reg)
						_, yBits := d186.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d189 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d189)
					} else {
						r14 := ctx.AllocRegExcept(d186.Reg, d186.Reg)
						ctx.EmitMovRegReg(r14, d186.Reg)
						ctx.EmitMulFloat64(r14, d186.Reg)
						d189 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r14}
						ctx.BindReg(r14, &d189)
					}
					if d189.Loc == LocReg && d186.Loc == LocReg && d189.Reg == d186.Reg {
						ctx.TransferReg(d186.Reg)
						d186.Loc = LocNone
					}
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d189)
					ctx.EnsureDescsTogether(&d12, &d189)
					var d190 JITValueDesc
					if d12.Loc == LocImm && d189.Loc == LocImm {
						d190 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d12.Imm.Float() + d189.Imm.Float())}
					} else if d12.Loc == LocImm {
						var scratch Reg
						if phiHomeOK4 && r2 != d189.Reg {
							scratch = r2
						} else {
							scratch = ctx.AllocRegExcept(d189.Reg)
						}
						_, xBits := d12.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d189.Reg)
						d190 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d190)
					} else if d189.Loc == LocImm {
						var scratch Reg
						if phiHomeOK4 {
							scratch = r2
						} else {
							scratch = ctx.AllocRegExcept(d12.Reg)
						}
						ctx.EmitMovRegReg(scratch, d12.Reg)
						_, yBits := d189.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d190 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d190)
					} else {
						var r15 Reg
						if phiHomeOK4 && r2 != d189.Reg {
							r15 = r2
						} else {
							r15 = ctx.AllocRegExcept(d12.Reg, d189.Reg)
						}
						ctx.EmitMovRegReg(r15, d12.Reg)
						ctx.EmitAddFloat64(r15, d189.Reg)
						d190 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r15}
						ctx.BindReg(r15, &d190)
					}
					if d190.Loc == LocReg && d12.Loc == LocReg && d190.Reg == d12.Reg {
						ctx.TransferReg(d12.Reg)
						d12.Loc = LocNone
					}
					ctx.FreeDesc(&d189)
					ctx.EnsureDesc(&d182)
					ctx.EnsureDesc(&d186)
					ctx.EnsureDescsTogether(&d182, &d186)
					var d191 JITValueDesc
					if d182.Loc == LocImm && d186.Loc == LocImm {
						d191 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d182.Imm.Float() * d186.Imm.Float())}
					} else if d182.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d186.Reg)
						_, xBits := d182.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d186.Reg)
						d191 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d191)
					} else if d186.Loc == LocImm {
						_, yBits := d186.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d182.Reg, RegR11)
						d191 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d182.Reg}
						ctx.BindReg(d182.Reg, &d191)
					} else {
						ctx.EmitMulFloat64(d182.Reg, d186.Reg)
						d191 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d182.Reg}
						ctx.BindReg(d182.Reg, &d191)
					}
					if d191.Loc == LocReg && d182.Loc == LocReg && d191.Reg == d182.Reg {
						ctx.TransferReg(d182.Reg)
						d182.Loc = LocNone
					}
					ctx.FreeDesc(&d182)
					ctx.FreeDesc(&d186)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d191)
					ctx.EnsureDescsTogether(&d10, &d191)
					var d192 JITValueDesc
					if d10.Loc == LocImm && d191.Loc == LocImm {
						d192 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float() + d191.Imm.Float())}
					} else if d10.Loc == LocImm {
						var scratch Reg
						if phiHomeOK2 && r0 != d191.Reg {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d191.Reg)
						}
						_, xBits := d10.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d191.Reg)
						d192 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d192)
					} else if d191.Loc == LocImm {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d10.Reg)
						}
						ctx.EmitMovRegReg(scratch, d10.Reg)
						_, yBits := d191.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d192 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d192)
					} else {
						var r16 Reg
						if phiHomeOK2 && r0 != d191.Reg {
							r16 = r0
						} else {
							r16 = ctx.AllocRegExcept(d10.Reg, d191.Reg)
						}
						ctx.EmitMovRegReg(r16, d10.Reg)
						ctx.EmitAddFloat64(r16, d191.Reg)
						d192 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
						ctx.BindReg(r16, &d192)
					}
					if d192.Loc == LocReg && d10.Loc == LocReg && d192.Reg == d10.Reg {
						ctx.TransferReg(d10.Reg)
						d10.Loc = LocNone
					}
					ctx.FreeDesc(&d191)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					var d193 JITValueDesc
					if d13.Loc == LocImm {
						d193 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK5 {
							scratch = r3
						} else {
							scratch = ctx.AllocRegExcept(d13.Reg)
						}
						ctx.EmitMovRegReg(scratch, d13.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d193 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d193)
					}
					if d193.Loc == LocReg && d13.Loc == LocReg && d193.Reg == d13.Reg {
						ctx.TransferReg(d13.Reg)
						d13.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d188)
						if d188.Loc == LocReg {
							ctx.ProtectReg(d188.Reg)
						} else if d188.Loc == LocRegPair {
							ctx.ProtectReg(d188.Reg)
							ctx.ProtectReg(d188.Reg2)
						}
						ctx.SyncDesc(&d190)
						if d190.Loc == LocReg {
							ctx.ProtectReg(d190.Reg)
						} else if d190.Loc == LocRegPair {
							ctx.ProtectReg(d190.Reg)
							ctx.ProtectReg(d190.Reg2)
						}
						ctx.SyncDesc(&d192)
						if d192.Loc == LocReg {
							ctx.ProtectReg(d192.Reg)
						} else if d192.Loc == LocRegPair {
							ctx.ProtectReg(d192.Reg)
							ctx.ProtectReg(d192.Reg2)
						}
						d194 = d192
						if d194.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d194)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d194)
						} else {
							ctx.EmitStoreToStack(d194, int32(bbs[6].PhiBase)+int32(0))
						}
						d195 = d188
						if d195.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d195)
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, d195)
						} else {
							ctx.EmitStoreToStack(d195, int32(bbs[6].PhiBase)+int32(16))
						}
						d196 = d190
						if d196.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d196)
						if phiHomeOK4 {
							ctx.EmitMovToReg(r2, d196)
						} else {
							ctx.EmitStoreToStack(d196, int32(bbs[6].PhiBase)+int32(32))
						}
						if d188.Loc == LocReg {
							ctx.UnprotectReg(d188.Reg)
						} else if d188.Loc == LocRegPair {
							ctx.UnprotectReg(d188.Reg)
							ctx.UnprotectReg(d188.Reg2)
						}
						if d190.Loc == LocReg {
							ctx.UnprotectReg(d190.Reg)
						} else if d190.Loc == LocRegPair {
							ctx.UnprotectReg(d190.Reg)
							ctx.UnprotectReg(d190.Reg2)
						}
						if d192.Loc == LocReg {
							ctx.UnprotectReg(d192.Reg)
						} else if d192.Loc == LocRegPair {
							ctx.UnprotectReg(d192.Reg)
							ctx.UnprotectReg(d192.Reg2)
						}
						ctx.SyncDesc(&d193)
						if d193.Loc == LocReg {
							ctx.ProtectReg(d193.Reg)
						} else if d193.Loc == LocRegPair {
							ctx.ProtectReg(d193.Reg)
							ctx.ProtectReg(d193.Reg2)
						}
						d197 = d193
						if d197.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d197)
						if phiHomeOK5 {
							ctx.EmitMovToReg(r3, d197)
						} else {
							ctx.EmitStoreToStack(d197, int32(bbs[6].PhiBase)+int32(48))
						}
						if d193.Loc == LocReg {
							ctx.UnprotectReg(d193.Reg)
						} else if d193.Loc == LocRegPair {
							ctx.UnprotectReg(d193.Reg)
							ctx.UnprotectReg(d193.Reg2)
						}
					}
					ps198 := PhiState{General: ps.General}
					ps198.OverlayValues = make([]JITValueDesc, 198)
					ps198.OverlayValues[8] = d8
					ps198.OverlayValues[9] = d9
					ps198.OverlayValues[10] = d10
					ps198.OverlayValues[11] = d11
					ps198.OverlayValues[12] = d12
					ps198.OverlayValues[13] = d13
					ps198.OverlayValues[14] = d14
					ps198.OverlayValues[15] = d15
					ps198.OverlayValues[16] = d16
					ps198.OverlayValues[17] = d17
					ps198.OverlayValues[18] = d18
					ps198.OverlayValues[19] = d19
					ps198.OverlayValues[20] = d20
					ps198.OverlayValues[21] = d21
					ps198.OverlayValues[22] = d22
					ps198.OverlayValues[24] = d24
					ps198.OverlayValues[26] = d26
					ps198.OverlayValues[27] = d27
					ps198.OverlayValues[30] = d30
					ps198.OverlayValues[51] = d51
					ps198.OverlayValues[52] = d52
					ps198.OverlayValues[53] = d53
					ps198.OverlayValues[54] = d54
					ps198.OverlayValues[55] = d55
					ps198.OverlayValues[57] = d57
					ps198.OverlayValues[58] = d58
					ps198.OverlayValues[59] = d59
					ps198.OverlayValues[60] = d60
					ps198.OverlayValues[61] = d61
					ps198.OverlayValues[62] = d62
					ps198.OverlayValues[63] = d63
					ps198.OverlayValues[66] = d66
					ps198.OverlayValues[103] = d103
					ps198.OverlayValues[104] = d104
					ps198.OverlayValues[105] = d105
					ps198.OverlayValues[106] = d106
					ps198.OverlayValues[107] = d107
					ps198.OverlayValues[108] = d108
					ps198.OverlayValues[110] = d110
					ps198.OverlayValues[111] = d111
					ps198.OverlayValues[112] = d112
					ps198.OverlayValues[113] = d113
					ps198.OverlayValues[114] = d114
					ps198.OverlayValues[115] = d115
					ps198.OverlayValues[116] = d116
					ps198.OverlayValues[117] = d117
					ps198.OverlayValues[118] = d118
					ps198.OverlayValues[121] = d121
					ps198.OverlayValues[122] = d122
					ps198.OverlayValues[123] = d123
					ps198.OverlayValues[124] = d124
					ps198.OverlayValues[179] = d179
					ps198.OverlayValues[180] = d180
					ps198.OverlayValues[181] = d181
					ps198.OverlayValues[182] = d182
					ps198.OverlayValues[183] = d183
					ps198.OverlayValues[184] = d184
					ps198.OverlayValues[185] = d185
					ps198.OverlayValues[186] = d186
					ps198.OverlayValues[187] = d187
					ps198.OverlayValues[188] = d188
					ps198.OverlayValues[189] = d189
					ps198.OverlayValues[190] = d190
					ps198.OverlayValues[191] = d191
					ps198.OverlayValues[192] = d192
					ps198.OverlayValues[193] = d193
					ps198.OverlayValues[194] = d194
					ps198.OverlayValues[195] = d195
					ps198.OverlayValues[196] = d196
					ps198.OverlayValues[197] = d197
					ps198.PhiValues = make([]JITValueDesc, 4)
					d199 = d192
					ps198.PhiValues[0] = d199
					d200 = d188
					ps198.PhiValues[1] = d200
					d201 = d190
					ps198.PhiValues[2] = d201
					d202 = d193
					ps198.PhiValues[3] = d202
					if ps198.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps198)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
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
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDescsTogether(&d11, &d12)
					var d203 JITValueDesc
					if d11.Loc == LocImm && d12.Loc == LocImm {
						d203 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d11.Imm.Float() * d12.Imm.Float())}
					} else if d11.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d12.Reg)
						_, xBits := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d12.Reg)
						d203 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d203)
					} else if d12.Loc == LocImm {
						_, yBits := d12.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d11.Reg, RegR11)
						d203 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d11.Reg}
						ctx.BindReg(d11.Reg, &d203)
					} else {
						ctx.EmitMulFloat64(d11.Reg, d12.Reg)
						d203 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d11.Reg}
						ctx.BindReg(d11.Reg, &d203)
					}
					if d203.Loc == LocReg && d11.Loc == LocReg && d203.Reg == d11.Reg {
						ctx.TransferReg(d11.Reg)
						d11.Loc = LocNone
					}
					ctx.FreeDesc(&d11)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d203)
					var d204 JITValueDesc
					if d203.Loc == LocImm {
						d204 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d203.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d203)
						var d205 JITValueDesc
						if d203.Loc == LocRegPair {
							ctx.FreeReg(d203.Reg)
							d205 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d203.Reg2}
							ctx.BindReg(d203.Reg2, &d205)
							ctx.BindReg(d203.Reg2, &d205)
						} else {
							d205 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d203.Reg}
							ctx.BindReg(d203.Reg, &d205)
							ctx.BindReg(d203.Reg, &d205)
						}
						d204 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d205}, 1)
						d204.Type = tagFloat
						ctx.BindReg(d204.Reg, &d204)
					}
					ctx.FreeDesc(&d203)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d204)
					ctx.EnsureDescsTogether(&d10, &d204)
					var d206 JITValueDesc
					if d10.Loc == LocImm && d204.Loc == LocImm {
						d206 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float() / d204.Imm.Float())}
					} else if d10.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d204.Reg)
						_, xBits := d10.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d204.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d206)
					} else if d204.Loc == LocImm {
						_, yBits := d204.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitDivFloat64(d10.Reg, RegR11)
						d206 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg}
						ctx.BindReg(d10.Reg, &d206)
					} else {
						ctx.EmitDivFloat64(d10.Reg, d204.Reg)
						d206 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg}
						ctx.BindReg(d10.Reg, &d206)
					}
					if d206.Loc == LocReg && d10.Loc == LocReg && d206.Reg == d10.Reg {
						ctx.TransferReg(d10.Reg)
						d10.Loc = LocNone
					}
					ctx.EnsureDesc(&d206)
					ctx.EmitStoreToStack(d206, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d206)
					ctx.FreeDesc(&d10)
					ctx.FreeDesc(&d204)
					if ps.General {
					}
					ps207 := PhiState{General: ps.General}
					ps207.OverlayValues = make([]JITValueDesc, 207)
					ps207.OverlayValues[8] = d8
					ps207.OverlayValues[9] = d9
					ps207.OverlayValues[10] = d10
					ps207.OverlayValues[11] = d11
					ps207.OverlayValues[12] = d12
					ps207.OverlayValues[13] = d13
					ps207.OverlayValues[14] = d14
					ps207.OverlayValues[15] = d15
					ps207.OverlayValues[16] = d16
					ps207.OverlayValues[17] = d17
					ps207.OverlayValues[18] = d18
					ps207.OverlayValues[19] = d19
					ps207.OverlayValues[20] = d20
					ps207.OverlayValues[21] = d21
					ps207.OverlayValues[22] = d22
					ps207.OverlayValues[24] = d24
					ps207.OverlayValues[26] = d26
					ps207.OverlayValues[27] = d27
					ps207.OverlayValues[30] = d30
					ps207.OverlayValues[51] = d51
					ps207.OverlayValues[52] = d52
					ps207.OverlayValues[53] = d53
					ps207.OverlayValues[54] = d54
					ps207.OverlayValues[55] = d55
					ps207.OverlayValues[57] = d57
					ps207.OverlayValues[58] = d58
					ps207.OverlayValues[59] = d59
					ps207.OverlayValues[60] = d60
					ps207.OverlayValues[61] = d61
					ps207.OverlayValues[62] = d62
					ps207.OverlayValues[63] = d63
					ps207.OverlayValues[66] = d66
					ps207.OverlayValues[103] = d103
					ps207.OverlayValues[104] = d104
					ps207.OverlayValues[105] = d105
					ps207.OverlayValues[106] = d106
					ps207.OverlayValues[107] = d107
					ps207.OverlayValues[108] = d108
					ps207.OverlayValues[110] = d110
					ps207.OverlayValues[111] = d111
					ps207.OverlayValues[112] = d112
					ps207.OverlayValues[113] = d113
					ps207.OverlayValues[114] = d114
					ps207.OverlayValues[115] = d115
					ps207.OverlayValues[116] = d116
					ps207.OverlayValues[117] = d117
					ps207.OverlayValues[118] = d118
					ps207.OverlayValues[121] = d121
					ps207.OverlayValues[122] = d122
					ps207.OverlayValues[123] = d123
					ps207.OverlayValues[124] = d124
					ps207.OverlayValues[179] = d179
					ps207.OverlayValues[180] = d180
					ps207.OverlayValues[181] = d181
					ps207.OverlayValues[182] = d182
					ps207.OverlayValues[183] = d183
					ps207.OverlayValues[184] = d184
					ps207.OverlayValues[185] = d185
					ps207.OverlayValues[186] = d186
					ps207.OverlayValues[187] = d187
					ps207.OverlayValues[188] = d188
					ps207.OverlayValues[189] = d189
					ps207.OverlayValues[190] = d190
					ps207.OverlayValues[191] = d191
					ps207.OverlayValues[192] = d192
					ps207.OverlayValues[193] = d193
					ps207.OverlayValues[194] = d194
					ps207.OverlayValues[195] = d195
					ps207.OverlayValues[196] = d196
					ps207.OverlayValues[197] = d197
					ps207.OverlayValues[199] = d199
					ps207.OverlayValues[200] = d200
					ps207.OverlayValues[201] = d201
					ps207.OverlayValues[202] = d202
					ps207.OverlayValues[203] = d203
					ps207.OverlayValues[204] = d204
					ps207.OverlayValues[205] = d205
					ps207.OverlayValues[206] = d206
					ps207.PhiValues = make([]JITValueDesc, 1)
					if ps207.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps207)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
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
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					ctx.ReclaimUntrackedRegs()
					var d208 JITValueDesc
					if d19.SliceSizeKnown {
						d208 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d208 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d208 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d208 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d208 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d208)
					ctx.EnsureDescsTogether(&d13, &d208)
					var d209 JITValueDesc
					if d13.Loc == LocImm && d208.Loc == LocImm {
						d209 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d208.Imm.Int())}
					} else if d208.Loc == LocImm {
						r17 := ctx.AllocReg()
						if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d208.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d208.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, RegR11)
						}
						ctx.EmitSetcc(r17, CondSignedLess)
						d209 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
						ctx.BindReg(r17, &d209)
					} else if d13.Loc == LocImm {
						r18 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d208.Reg)
						ctx.EmitSetcc(r18, CondSignedLess)
						d209 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
						ctx.BindReg(r18, &d209)
					} else {
						r19 := ctx.AllocReg()
						ctx.EmitCmpInt64(d13.Reg, d208.Reg)
						ctx.EmitSetcc(r19, CondSignedLess)
						d209 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r19}
						ctx.BindReg(r19, &d209)
					}
					ctx.FreeDesc(&d13)
					ctx.FreeDesc(&d208)
					d210 = d209
					ctx.EnsureDesc(&d210)
					if d210.Loc != LocImm && d210.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d210.Loc == LocImm {
						if d210.Imm.Bool() {
							if ps.General {
							}
							ps211 := PhiState{General: ps.General}
							ps211.OverlayValues = make([]JITValueDesc, 211)
							ps211.OverlayValues[8] = d8
							ps211.OverlayValues[9] = d9
							ps211.OverlayValues[10] = d10
							ps211.OverlayValues[11] = d11
							ps211.OverlayValues[12] = d12
							ps211.OverlayValues[13] = d13
							ps211.OverlayValues[14] = d14
							ps211.OverlayValues[15] = d15
							ps211.OverlayValues[16] = d16
							ps211.OverlayValues[17] = d17
							ps211.OverlayValues[18] = d18
							ps211.OverlayValues[19] = d19
							ps211.OverlayValues[20] = d20
							ps211.OverlayValues[21] = d21
							ps211.OverlayValues[22] = d22
							ps211.OverlayValues[24] = d24
							ps211.OverlayValues[26] = d26
							ps211.OverlayValues[27] = d27
							ps211.OverlayValues[30] = d30
							ps211.OverlayValues[51] = d51
							ps211.OverlayValues[52] = d52
							ps211.OverlayValues[53] = d53
							ps211.OverlayValues[54] = d54
							ps211.OverlayValues[55] = d55
							ps211.OverlayValues[57] = d57
							ps211.OverlayValues[58] = d58
							ps211.OverlayValues[59] = d59
							ps211.OverlayValues[60] = d60
							ps211.OverlayValues[61] = d61
							ps211.OverlayValues[62] = d62
							ps211.OverlayValues[63] = d63
							ps211.OverlayValues[66] = d66
							ps211.OverlayValues[103] = d103
							ps211.OverlayValues[104] = d104
							ps211.OverlayValues[105] = d105
							ps211.OverlayValues[106] = d106
							ps211.OverlayValues[107] = d107
							ps211.OverlayValues[108] = d108
							ps211.OverlayValues[110] = d110
							ps211.OverlayValues[111] = d111
							ps211.OverlayValues[112] = d112
							ps211.OverlayValues[113] = d113
							ps211.OverlayValues[114] = d114
							ps211.OverlayValues[115] = d115
							ps211.OverlayValues[116] = d116
							ps211.OverlayValues[117] = d117
							ps211.OverlayValues[118] = d118
							ps211.OverlayValues[121] = d121
							ps211.OverlayValues[122] = d122
							ps211.OverlayValues[123] = d123
							ps211.OverlayValues[124] = d124
							ps211.OverlayValues[179] = d179
							ps211.OverlayValues[180] = d180
							ps211.OverlayValues[181] = d181
							ps211.OverlayValues[182] = d182
							ps211.OverlayValues[183] = d183
							ps211.OverlayValues[184] = d184
							ps211.OverlayValues[185] = d185
							ps211.OverlayValues[186] = d186
							ps211.OverlayValues[187] = d187
							ps211.OverlayValues[188] = d188
							ps211.OverlayValues[189] = d189
							ps211.OverlayValues[190] = d190
							ps211.OverlayValues[191] = d191
							ps211.OverlayValues[192] = d192
							ps211.OverlayValues[193] = d193
							ps211.OverlayValues[194] = d194
							ps211.OverlayValues[195] = d195
							ps211.OverlayValues[196] = d196
							ps211.OverlayValues[197] = d197
							ps211.OverlayValues[199] = d199
							ps211.OverlayValues[200] = d200
							ps211.OverlayValues[201] = d201
							ps211.OverlayValues[202] = d202
							ps211.OverlayValues[203] = d203
							ps211.OverlayValues[204] = d204
							ps211.OverlayValues[205] = d205
							ps211.OverlayValues[206] = d206
							ps211.OverlayValues[208] = d208
							ps211.OverlayValues[209] = d209
							ps211.OverlayValues[210] = d210
							return bbs[7].RenderPS(ps211)
						}
						if ps.General {
						}
						ps212 := PhiState{General: ps.General}
						ps212.OverlayValues = make([]JITValueDesc, 211)
						ps212.OverlayValues[8] = d8
						ps212.OverlayValues[9] = d9
						ps212.OverlayValues[10] = d10
						ps212.OverlayValues[11] = d11
						ps212.OverlayValues[12] = d12
						ps212.OverlayValues[13] = d13
						ps212.OverlayValues[14] = d14
						ps212.OverlayValues[15] = d15
						ps212.OverlayValues[16] = d16
						ps212.OverlayValues[17] = d17
						ps212.OverlayValues[18] = d18
						ps212.OverlayValues[19] = d19
						ps212.OverlayValues[20] = d20
						ps212.OverlayValues[21] = d21
						ps212.OverlayValues[22] = d22
						ps212.OverlayValues[24] = d24
						ps212.OverlayValues[26] = d26
						ps212.OverlayValues[27] = d27
						ps212.OverlayValues[30] = d30
						ps212.OverlayValues[51] = d51
						ps212.OverlayValues[52] = d52
						ps212.OverlayValues[53] = d53
						ps212.OverlayValues[54] = d54
						ps212.OverlayValues[55] = d55
						ps212.OverlayValues[57] = d57
						ps212.OverlayValues[58] = d58
						ps212.OverlayValues[59] = d59
						ps212.OverlayValues[60] = d60
						ps212.OverlayValues[61] = d61
						ps212.OverlayValues[62] = d62
						ps212.OverlayValues[63] = d63
						ps212.OverlayValues[66] = d66
						ps212.OverlayValues[103] = d103
						ps212.OverlayValues[104] = d104
						ps212.OverlayValues[105] = d105
						ps212.OverlayValues[106] = d106
						ps212.OverlayValues[107] = d107
						ps212.OverlayValues[108] = d108
						ps212.OverlayValues[110] = d110
						ps212.OverlayValues[111] = d111
						ps212.OverlayValues[112] = d112
						ps212.OverlayValues[113] = d113
						ps212.OverlayValues[114] = d114
						ps212.OverlayValues[115] = d115
						ps212.OverlayValues[116] = d116
						ps212.OverlayValues[117] = d117
						ps212.OverlayValues[118] = d118
						ps212.OverlayValues[121] = d121
						ps212.OverlayValues[122] = d122
						ps212.OverlayValues[123] = d123
						ps212.OverlayValues[124] = d124
						ps212.OverlayValues[179] = d179
						ps212.OverlayValues[180] = d180
						ps212.OverlayValues[181] = d181
						ps212.OverlayValues[182] = d182
						ps212.OverlayValues[183] = d183
						ps212.OverlayValues[184] = d184
						ps212.OverlayValues[185] = d185
						ps212.OverlayValues[186] = d186
						ps212.OverlayValues[187] = d187
						ps212.OverlayValues[188] = d188
						ps212.OverlayValues[189] = d189
						ps212.OverlayValues[190] = d190
						ps212.OverlayValues[191] = d191
						ps212.OverlayValues[192] = d192
						ps212.OverlayValues[193] = d193
						ps212.OverlayValues[194] = d194
						ps212.OverlayValues[195] = d195
						ps212.OverlayValues[196] = d196
						ps212.OverlayValues[197] = d197
						ps212.OverlayValues[199] = d199
						ps212.OverlayValues[200] = d200
						ps212.OverlayValues[201] = d201
						ps212.OverlayValues[202] = d202
						ps212.OverlayValues[203] = d203
						ps212.OverlayValues[204] = d204
						ps212.OverlayValues[205] = d205
						ps212.OverlayValues[206] = d206
						ps212.OverlayValues[208] = d208
						ps212.OverlayValues[209] = d209
						ps212.OverlayValues[210] = d210
						return bbs[8].RenderPS(ps212)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d210.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl9)
					ps213 := PhiState{General: true}
					ps213.OverlayValues = make([]JITValueDesc, 211)
					ps213.OverlayValues[8] = d8
					ps213.OverlayValues[9] = d9
					ps213.OverlayValues[10] = d10
					ps213.OverlayValues[11] = d11
					ps213.OverlayValues[12] = d12
					ps213.OverlayValues[13] = d13
					ps213.OverlayValues[14] = d14
					ps213.OverlayValues[15] = d15
					ps213.OverlayValues[16] = d16
					ps213.OverlayValues[17] = d17
					ps213.OverlayValues[18] = d18
					ps213.OverlayValues[19] = d19
					ps213.OverlayValues[20] = d20
					ps213.OverlayValues[21] = d21
					ps213.OverlayValues[22] = d22
					ps213.OverlayValues[24] = d24
					ps213.OverlayValues[26] = d26
					ps213.OverlayValues[27] = d27
					ps213.OverlayValues[30] = d30
					ps213.OverlayValues[51] = d51
					ps213.OverlayValues[52] = d52
					ps213.OverlayValues[53] = d53
					ps213.OverlayValues[54] = d54
					ps213.OverlayValues[55] = d55
					ps213.OverlayValues[57] = d57
					ps213.OverlayValues[58] = d58
					ps213.OverlayValues[59] = d59
					ps213.OverlayValues[60] = d60
					ps213.OverlayValues[61] = d61
					ps213.OverlayValues[62] = d62
					ps213.OverlayValues[63] = d63
					ps213.OverlayValues[66] = d66
					ps213.OverlayValues[103] = d103
					ps213.OverlayValues[104] = d104
					ps213.OverlayValues[105] = d105
					ps213.OverlayValues[106] = d106
					ps213.OverlayValues[107] = d107
					ps213.OverlayValues[108] = d108
					ps213.OverlayValues[110] = d110
					ps213.OverlayValues[111] = d111
					ps213.OverlayValues[112] = d112
					ps213.OverlayValues[113] = d113
					ps213.OverlayValues[114] = d114
					ps213.OverlayValues[115] = d115
					ps213.OverlayValues[116] = d116
					ps213.OverlayValues[117] = d117
					ps213.OverlayValues[118] = d118
					ps213.OverlayValues[121] = d121
					ps213.OverlayValues[122] = d122
					ps213.OverlayValues[123] = d123
					ps213.OverlayValues[124] = d124
					ps213.OverlayValues[179] = d179
					ps213.OverlayValues[180] = d180
					ps213.OverlayValues[181] = d181
					ps213.OverlayValues[182] = d182
					ps213.OverlayValues[183] = d183
					ps213.OverlayValues[184] = d184
					ps213.OverlayValues[185] = d185
					ps213.OverlayValues[186] = d186
					ps213.OverlayValues[187] = d187
					ps213.OverlayValues[188] = d188
					ps213.OverlayValues[189] = d189
					ps213.OverlayValues[190] = d190
					ps213.OverlayValues[191] = d191
					ps213.OverlayValues[192] = d192
					ps213.OverlayValues[193] = d193
					ps213.OverlayValues[194] = d194
					ps213.OverlayValues[195] = d195
					ps213.OverlayValues[196] = d196
					ps213.OverlayValues[197] = d197
					ps213.OverlayValues[199] = d199
					ps213.OverlayValues[200] = d200
					ps213.OverlayValues[201] = d201
					ps213.OverlayValues[202] = d202
					ps213.OverlayValues[203] = d203
					ps213.OverlayValues[204] = d204
					ps213.OverlayValues[205] = d205
					ps213.OverlayValues[206] = d206
					ps213.OverlayValues[208] = d208
					ps213.OverlayValues[209] = d209
					ps213.OverlayValues[210] = d210
					ps214 := PhiState{General: true}
					ps214.OverlayValues = make([]JITValueDesc, 211)
					ps214.OverlayValues[8] = d8
					ps214.OverlayValues[9] = d9
					ps214.OverlayValues[10] = d10
					ps214.OverlayValues[11] = d11
					ps214.OverlayValues[12] = d12
					ps214.OverlayValues[13] = d13
					ps214.OverlayValues[14] = d14
					ps214.OverlayValues[15] = d15
					ps214.OverlayValues[16] = d16
					ps214.OverlayValues[17] = d17
					ps214.OverlayValues[18] = d18
					ps214.OverlayValues[19] = d19
					ps214.OverlayValues[20] = d20
					ps214.OverlayValues[21] = d21
					ps214.OverlayValues[22] = d22
					ps214.OverlayValues[24] = d24
					ps214.OverlayValues[26] = d26
					ps214.OverlayValues[27] = d27
					ps214.OverlayValues[30] = d30
					ps214.OverlayValues[51] = d51
					ps214.OverlayValues[52] = d52
					ps214.OverlayValues[53] = d53
					ps214.OverlayValues[54] = d54
					ps214.OverlayValues[55] = d55
					ps214.OverlayValues[57] = d57
					ps214.OverlayValues[58] = d58
					ps214.OverlayValues[59] = d59
					ps214.OverlayValues[60] = d60
					ps214.OverlayValues[61] = d61
					ps214.OverlayValues[62] = d62
					ps214.OverlayValues[63] = d63
					ps214.OverlayValues[66] = d66
					ps214.OverlayValues[103] = d103
					ps214.OverlayValues[104] = d104
					ps214.OverlayValues[105] = d105
					ps214.OverlayValues[106] = d106
					ps214.OverlayValues[107] = d107
					ps214.OverlayValues[108] = d108
					ps214.OverlayValues[110] = d110
					ps214.OverlayValues[111] = d111
					ps214.OverlayValues[112] = d112
					ps214.OverlayValues[113] = d113
					ps214.OverlayValues[114] = d114
					ps214.OverlayValues[115] = d115
					ps214.OverlayValues[116] = d116
					ps214.OverlayValues[117] = d117
					ps214.OverlayValues[118] = d118
					ps214.OverlayValues[121] = d121
					ps214.OverlayValues[122] = d122
					ps214.OverlayValues[123] = d123
					ps214.OverlayValues[124] = d124
					ps214.OverlayValues[179] = d179
					ps214.OverlayValues[180] = d180
					ps214.OverlayValues[181] = d181
					ps214.OverlayValues[182] = d182
					ps214.OverlayValues[183] = d183
					ps214.OverlayValues[184] = d184
					ps214.OverlayValues[185] = d185
					ps214.OverlayValues[186] = d186
					ps214.OverlayValues[187] = d187
					ps214.OverlayValues[188] = d188
					ps214.OverlayValues[189] = d189
					ps214.OverlayValues[190] = d190
					ps214.OverlayValues[191] = d191
					ps214.OverlayValues[192] = d192
					ps214.OverlayValues[193] = d193
					ps214.OverlayValues[194] = d194
					ps214.OverlayValues[195] = d195
					ps214.OverlayValues[196] = d196
					ps214.OverlayValues[197] = d197
					ps214.OverlayValues[199] = d199
					ps214.OverlayValues[200] = d200
					ps214.OverlayValues[201] = d201
					ps214.OverlayValues[202] = d202
					ps214.OverlayValues[203] = d203
					ps214.OverlayValues[204] = d204
					ps214.OverlayValues[205] = d205
					ps214.OverlayValues[206] = d206
					ps214.OverlayValues[208] = d208
					ps214.OverlayValues[209] = d209
					ps214.OverlayValues[210] = d210
					snap215 := d8
					snap216 := d9
					snap217 := d10
					snap218 := d11
					snap219 := d12
					snap220 := d13
					snap221 := d14
					snap222 := d15
					snap223 := d16
					snap224 := d17
					snap225 := d18
					snap226 := d19
					snap227 := d20
					snap228 := d21
					snap229 := d22
					snap230 := d24
					snap231 := d26
					snap232 := d27
					snap233 := d30
					snap234 := d51
					snap235 := d52
					snap236 := d53
					snap237 := d54
					snap238 := d55
					snap239 := d57
					snap240 := d58
					snap241 := d59
					snap242 := d60
					snap243 := d61
					snap244 := d62
					snap245 := d63
					snap246 := d66
					snap247 := d103
					snap248 := d104
					snap249 := d105
					snap250 := d106
					snap251 := d107
					snap252 := d108
					snap253 := d110
					snap254 := d111
					snap255 := d112
					snap256 := d113
					snap257 := d114
					snap258 := d115
					snap259 := d116
					snap260 := d117
					snap261 := d118
					snap262 := d121
					snap263 := d122
					snap264 := d123
					snap265 := d124
					snap266 := d179
					snap267 := d180
					snap268 := d181
					snap269 := d182
					snap270 := d183
					snap271 := d184
					snap272 := d185
					snap273 := d186
					snap274 := d187
					snap275 := d188
					snap276 := d189
					snap277 := d190
					snap278 := d191
					snap279 := d192
					snap280 := d193
					snap281 := d194
					snap282 := d195
					snap283 := d196
					snap284 := d197
					snap285 := d199
					snap286 := d200
					snap287 := d201
					snap288 := d202
					snap289 := d203
					snap290 := d204
					snap291 := d205
					snap292 := d206
					snap293 := d208
					snap294 := d209
					snap295 := d210
					alloc296 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps214)
					}
					ctx.RestoreAllocState(alloc296)
					d8 = snap215
					d9 = snap216
					d10 = snap217
					d11 = snap218
					d12 = snap219
					d13 = snap220
					d14 = snap221
					d15 = snap222
					d16 = snap223
					d17 = snap224
					d18 = snap225
					d19 = snap226
					d20 = snap227
					d21 = snap228
					d22 = snap229
					d24 = snap230
					d26 = snap231
					d27 = snap232
					d30 = snap233
					d51 = snap234
					d52 = snap235
					d53 = snap236
					d54 = snap237
					d55 = snap238
					d57 = snap239
					d58 = snap240
					d59 = snap241
					d60 = snap242
					d61 = snap243
					d62 = snap244
					d63 = snap245
					d66 = snap246
					d103 = snap247
					d104 = snap248
					d105 = snap249
					d106 = snap250
					d107 = snap251
					d108 = snap252
					d110 = snap253
					d111 = snap254
					d112 = snap255
					d113 = snap256
					d114 = snap257
					d115 = snap258
					d116 = snap259
					d117 = snap260
					d118 = snap261
					d121 = snap262
					d122 = snap263
					d123 = snap264
					d124 = snap265
					d179 = snap266
					d180 = snap267
					d181 = snap268
					d182 = snap269
					d183 = snap270
					d184 = snap271
					d185 = snap272
					d186 = snap273
					d187 = snap274
					d188 = snap275
					d189 = snap276
					d190 = snap277
					d191 = snap278
					d192 = snap279
					d193 = snap280
					d194 = snap281
					d195 = snap282
					d196 = snap283
					d197 = snap284
					d199 = snap285
					d200 = snap286
					d201 = snap287
					d202 = snap288
					d203 = snap289
					d204 = snap290
					d205 = snap291
					d206 = snap292
					d208 = snap293
					d209 = snap294
					d210 = snap295
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps213)
					}
					return result
					ctx.FreeDesc(&d209)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d297 := ps.PhiValues[0]
							if phiHomeOK6 {
								ctx.EmitMovToReg(r4, d297)
							} else {
								ctx.EmitStoreToStack(d297, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d298 := ps.PhiValues[1]
							if phiHomeOK7 {
								ctx.EmitMovToReg(r5, d298)
							} else {
								ctx.EmitStoreToStack(d298, int32(bbs[10].PhiBase)+int32(16))
							}
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
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
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d14 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d15 = ps.PhiValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					var d299 JITValueDesc
					if d17.SliceSizeKnown {
						d299 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d299 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d299 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d299 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d299 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d299)
					ctx.EnsureDescsTogether(&d15, &d299)
					var d300 JITValueDesc
					if d15.Loc == LocImm && d299.Loc == LocImm {
						d300 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d299.Imm.Int())}
					} else if d299.Loc == LocImm {
						r20 := ctx.AllocRegExcept(d15.Reg)
						if d299.Imm.Int() >= -2147483648 && d299.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d299.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d299.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, RegR11)
						}
						ctx.EmitSetcc(r20, CondSignedLess)
						d300 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
						ctx.BindReg(r20, &d300)
					} else if d15.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d299.Reg)
						ctx.EmitSetcc(r21, CondSignedLess)
						d300 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
						ctx.BindReg(r21, &d300)
					} else {
						r22 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpInt64(d15.Reg, d299.Reg)
						ctx.EmitSetcc(r22, CondSignedLess)
						d300 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
						ctx.BindReg(r22, &d300)
					}
					ctx.FreeDesc(&d299)
					d301 = d300
					ctx.EnsureDesc(&d301)
					if d301.Loc != LocImm && d301.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d301.Loc == LocImm {
						if d301.Imm.Bool() {
							if ps.General {
							}
							ps302 := PhiState{General: ps.General}
							ps302.OverlayValues = make([]JITValueDesc, 302)
							ps302.OverlayValues[8] = d8
							ps302.OverlayValues[9] = d9
							ps302.OverlayValues[10] = d10
							ps302.OverlayValues[11] = d11
							ps302.OverlayValues[12] = d12
							ps302.OverlayValues[13] = d13
							ps302.OverlayValues[14] = d14
							ps302.OverlayValues[15] = d15
							ps302.OverlayValues[16] = d16
							ps302.OverlayValues[17] = d17
							ps302.OverlayValues[18] = d18
							ps302.OverlayValues[19] = d19
							ps302.OverlayValues[20] = d20
							ps302.OverlayValues[21] = d21
							ps302.OverlayValues[22] = d22
							ps302.OverlayValues[24] = d24
							ps302.OverlayValues[26] = d26
							ps302.OverlayValues[27] = d27
							ps302.OverlayValues[30] = d30
							ps302.OverlayValues[51] = d51
							ps302.OverlayValues[52] = d52
							ps302.OverlayValues[53] = d53
							ps302.OverlayValues[54] = d54
							ps302.OverlayValues[55] = d55
							ps302.OverlayValues[57] = d57
							ps302.OverlayValues[58] = d58
							ps302.OverlayValues[59] = d59
							ps302.OverlayValues[60] = d60
							ps302.OverlayValues[61] = d61
							ps302.OverlayValues[62] = d62
							ps302.OverlayValues[63] = d63
							ps302.OverlayValues[66] = d66
							ps302.OverlayValues[103] = d103
							ps302.OverlayValues[104] = d104
							ps302.OverlayValues[105] = d105
							ps302.OverlayValues[106] = d106
							ps302.OverlayValues[107] = d107
							ps302.OverlayValues[108] = d108
							ps302.OverlayValues[110] = d110
							ps302.OverlayValues[111] = d111
							ps302.OverlayValues[112] = d112
							ps302.OverlayValues[113] = d113
							ps302.OverlayValues[114] = d114
							ps302.OverlayValues[115] = d115
							ps302.OverlayValues[116] = d116
							ps302.OverlayValues[117] = d117
							ps302.OverlayValues[118] = d118
							ps302.OverlayValues[121] = d121
							ps302.OverlayValues[122] = d122
							ps302.OverlayValues[123] = d123
							ps302.OverlayValues[124] = d124
							ps302.OverlayValues[179] = d179
							ps302.OverlayValues[180] = d180
							ps302.OverlayValues[181] = d181
							ps302.OverlayValues[182] = d182
							ps302.OverlayValues[183] = d183
							ps302.OverlayValues[184] = d184
							ps302.OverlayValues[185] = d185
							ps302.OverlayValues[186] = d186
							ps302.OverlayValues[187] = d187
							ps302.OverlayValues[188] = d188
							ps302.OverlayValues[189] = d189
							ps302.OverlayValues[190] = d190
							ps302.OverlayValues[191] = d191
							ps302.OverlayValues[192] = d192
							ps302.OverlayValues[193] = d193
							ps302.OverlayValues[194] = d194
							ps302.OverlayValues[195] = d195
							ps302.OverlayValues[196] = d196
							ps302.OverlayValues[197] = d197
							ps302.OverlayValues[199] = d199
							ps302.OverlayValues[200] = d200
							ps302.OverlayValues[201] = d201
							ps302.OverlayValues[202] = d202
							ps302.OverlayValues[203] = d203
							ps302.OverlayValues[204] = d204
							ps302.OverlayValues[205] = d205
							ps302.OverlayValues[206] = d206
							ps302.OverlayValues[208] = d208
							ps302.OverlayValues[209] = d209
							ps302.OverlayValues[210] = d210
							ps302.OverlayValues[297] = d297
							ps302.OverlayValues[298] = d298
							ps302.OverlayValues[299] = d299
							ps302.OverlayValues[300] = d300
							ps302.OverlayValues[301] = d301
							return bbs[13].RenderPS(ps302)
						}
						if ps.General {
						}
						ps303 := PhiState{General: ps.General}
						ps303.OverlayValues = make([]JITValueDesc, 302)
						ps303.OverlayValues[8] = d8
						ps303.OverlayValues[9] = d9
						ps303.OverlayValues[10] = d10
						ps303.OverlayValues[11] = d11
						ps303.OverlayValues[12] = d12
						ps303.OverlayValues[13] = d13
						ps303.OverlayValues[14] = d14
						ps303.OverlayValues[15] = d15
						ps303.OverlayValues[16] = d16
						ps303.OverlayValues[17] = d17
						ps303.OverlayValues[18] = d18
						ps303.OverlayValues[19] = d19
						ps303.OverlayValues[20] = d20
						ps303.OverlayValues[21] = d21
						ps303.OverlayValues[22] = d22
						ps303.OverlayValues[24] = d24
						ps303.OverlayValues[26] = d26
						ps303.OverlayValues[27] = d27
						ps303.OverlayValues[30] = d30
						ps303.OverlayValues[51] = d51
						ps303.OverlayValues[52] = d52
						ps303.OverlayValues[53] = d53
						ps303.OverlayValues[54] = d54
						ps303.OverlayValues[55] = d55
						ps303.OverlayValues[57] = d57
						ps303.OverlayValues[58] = d58
						ps303.OverlayValues[59] = d59
						ps303.OverlayValues[60] = d60
						ps303.OverlayValues[61] = d61
						ps303.OverlayValues[62] = d62
						ps303.OverlayValues[63] = d63
						ps303.OverlayValues[66] = d66
						ps303.OverlayValues[103] = d103
						ps303.OverlayValues[104] = d104
						ps303.OverlayValues[105] = d105
						ps303.OverlayValues[106] = d106
						ps303.OverlayValues[107] = d107
						ps303.OverlayValues[108] = d108
						ps303.OverlayValues[110] = d110
						ps303.OverlayValues[111] = d111
						ps303.OverlayValues[112] = d112
						ps303.OverlayValues[113] = d113
						ps303.OverlayValues[114] = d114
						ps303.OverlayValues[115] = d115
						ps303.OverlayValues[116] = d116
						ps303.OverlayValues[117] = d117
						ps303.OverlayValues[118] = d118
						ps303.OverlayValues[121] = d121
						ps303.OverlayValues[122] = d122
						ps303.OverlayValues[123] = d123
						ps303.OverlayValues[124] = d124
						ps303.OverlayValues[179] = d179
						ps303.OverlayValues[180] = d180
						ps303.OverlayValues[181] = d181
						ps303.OverlayValues[182] = d182
						ps303.OverlayValues[183] = d183
						ps303.OverlayValues[184] = d184
						ps303.OverlayValues[185] = d185
						ps303.OverlayValues[186] = d186
						ps303.OverlayValues[187] = d187
						ps303.OverlayValues[188] = d188
						ps303.OverlayValues[189] = d189
						ps303.OverlayValues[190] = d190
						ps303.OverlayValues[191] = d191
						ps303.OverlayValues[192] = d192
						ps303.OverlayValues[193] = d193
						ps303.OverlayValues[194] = d194
						ps303.OverlayValues[195] = d195
						ps303.OverlayValues[196] = d196
						ps303.OverlayValues[197] = d197
						ps303.OverlayValues[199] = d199
						ps303.OverlayValues[200] = d200
						ps303.OverlayValues[201] = d201
						ps303.OverlayValues[202] = d202
						ps303.OverlayValues[203] = d203
						ps303.OverlayValues[204] = d204
						ps303.OverlayValues[205] = d205
						ps303.OverlayValues[206] = d206
						ps303.OverlayValues[208] = d208
						ps303.OverlayValues[209] = d209
						ps303.OverlayValues[210] = d210
						ps303.OverlayValues[297] = d297
						ps303.OverlayValues[298] = d298
						ps303.OverlayValues[299] = d299
						ps303.OverlayValues[300] = d300
						ps303.OverlayValues[301] = d301
						return bbs[12].RenderPS(ps303)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d304 := ps.PhiValues[0]
							if phiHomeOK6 {
								ctx.EmitMovToReg(r4, d304)
							} else {
								ctx.EmitStoreToStack(d304, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d305 := ps.PhiValues[1]
							if phiHomeOK7 {
								ctx.EmitMovToReg(r5, d305)
							} else {
								ctx.EmitStoreToStack(d305, int32(bbs[10].PhiBase)+int32(16))
							}
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d301.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl13)
					ps306 := PhiState{General: true}
					ps306.OverlayValues = make([]JITValueDesc, 306)
					ps306.OverlayValues[8] = d8
					ps306.OverlayValues[9] = d9
					ps306.OverlayValues[10] = d10
					ps306.OverlayValues[11] = d11
					ps306.OverlayValues[12] = d12
					ps306.OverlayValues[13] = d13
					ps306.OverlayValues[14] = d14
					ps306.OverlayValues[15] = d15
					ps306.OverlayValues[16] = d16
					ps306.OverlayValues[17] = d17
					ps306.OverlayValues[18] = d18
					ps306.OverlayValues[19] = d19
					ps306.OverlayValues[20] = d20
					ps306.OverlayValues[21] = d21
					ps306.OverlayValues[22] = d22
					ps306.OverlayValues[24] = d24
					ps306.OverlayValues[26] = d26
					ps306.OverlayValues[27] = d27
					ps306.OverlayValues[30] = d30
					ps306.OverlayValues[51] = d51
					ps306.OverlayValues[52] = d52
					ps306.OverlayValues[53] = d53
					ps306.OverlayValues[54] = d54
					ps306.OverlayValues[55] = d55
					ps306.OverlayValues[57] = d57
					ps306.OverlayValues[58] = d58
					ps306.OverlayValues[59] = d59
					ps306.OverlayValues[60] = d60
					ps306.OverlayValues[61] = d61
					ps306.OverlayValues[62] = d62
					ps306.OverlayValues[63] = d63
					ps306.OverlayValues[66] = d66
					ps306.OverlayValues[103] = d103
					ps306.OverlayValues[104] = d104
					ps306.OverlayValues[105] = d105
					ps306.OverlayValues[106] = d106
					ps306.OverlayValues[107] = d107
					ps306.OverlayValues[108] = d108
					ps306.OverlayValues[110] = d110
					ps306.OverlayValues[111] = d111
					ps306.OverlayValues[112] = d112
					ps306.OverlayValues[113] = d113
					ps306.OverlayValues[114] = d114
					ps306.OverlayValues[115] = d115
					ps306.OverlayValues[116] = d116
					ps306.OverlayValues[117] = d117
					ps306.OverlayValues[118] = d118
					ps306.OverlayValues[121] = d121
					ps306.OverlayValues[122] = d122
					ps306.OverlayValues[123] = d123
					ps306.OverlayValues[124] = d124
					ps306.OverlayValues[179] = d179
					ps306.OverlayValues[180] = d180
					ps306.OverlayValues[181] = d181
					ps306.OverlayValues[182] = d182
					ps306.OverlayValues[183] = d183
					ps306.OverlayValues[184] = d184
					ps306.OverlayValues[185] = d185
					ps306.OverlayValues[186] = d186
					ps306.OverlayValues[187] = d187
					ps306.OverlayValues[188] = d188
					ps306.OverlayValues[189] = d189
					ps306.OverlayValues[190] = d190
					ps306.OverlayValues[191] = d191
					ps306.OverlayValues[192] = d192
					ps306.OverlayValues[193] = d193
					ps306.OverlayValues[194] = d194
					ps306.OverlayValues[195] = d195
					ps306.OverlayValues[196] = d196
					ps306.OverlayValues[197] = d197
					ps306.OverlayValues[199] = d199
					ps306.OverlayValues[200] = d200
					ps306.OverlayValues[201] = d201
					ps306.OverlayValues[202] = d202
					ps306.OverlayValues[203] = d203
					ps306.OverlayValues[204] = d204
					ps306.OverlayValues[205] = d205
					ps306.OverlayValues[206] = d206
					ps306.OverlayValues[208] = d208
					ps306.OverlayValues[209] = d209
					ps306.OverlayValues[210] = d210
					ps306.OverlayValues[297] = d297
					ps306.OverlayValues[298] = d298
					ps306.OverlayValues[299] = d299
					ps306.OverlayValues[300] = d300
					ps306.OverlayValues[301] = d301
					ps306.OverlayValues[304] = d304
					ps306.OverlayValues[305] = d305
					ps307 := PhiState{General: true}
					ps307.OverlayValues = make([]JITValueDesc, 306)
					ps307.OverlayValues[8] = d8
					ps307.OverlayValues[9] = d9
					ps307.OverlayValues[10] = d10
					ps307.OverlayValues[11] = d11
					ps307.OverlayValues[12] = d12
					ps307.OverlayValues[13] = d13
					ps307.OverlayValues[14] = d14
					ps307.OverlayValues[15] = d15
					ps307.OverlayValues[16] = d16
					ps307.OverlayValues[17] = d17
					ps307.OverlayValues[18] = d18
					ps307.OverlayValues[19] = d19
					ps307.OverlayValues[20] = d20
					ps307.OverlayValues[21] = d21
					ps307.OverlayValues[22] = d22
					ps307.OverlayValues[24] = d24
					ps307.OverlayValues[26] = d26
					ps307.OverlayValues[27] = d27
					ps307.OverlayValues[30] = d30
					ps307.OverlayValues[51] = d51
					ps307.OverlayValues[52] = d52
					ps307.OverlayValues[53] = d53
					ps307.OverlayValues[54] = d54
					ps307.OverlayValues[55] = d55
					ps307.OverlayValues[57] = d57
					ps307.OverlayValues[58] = d58
					ps307.OverlayValues[59] = d59
					ps307.OverlayValues[60] = d60
					ps307.OverlayValues[61] = d61
					ps307.OverlayValues[62] = d62
					ps307.OverlayValues[63] = d63
					ps307.OverlayValues[66] = d66
					ps307.OverlayValues[103] = d103
					ps307.OverlayValues[104] = d104
					ps307.OverlayValues[105] = d105
					ps307.OverlayValues[106] = d106
					ps307.OverlayValues[107] = d107
					ps307.OverlayValues[108] = d108
					ps307.OverlayValues[110] = d110
					ps307.OverlayValues[111] = d111
					ps307.OverlayValues[112] = d112
					ps307.OverlayValues[113] = d113
					ps307.OverlayValues[114] = d114
					ps307.OverlayValues[115] = d115
					ps307.OverlayValues[116] = d116
					ps307.OverlayValues[117] = d117
					ps307.OverlayValues[118] = d118
					ps307.OverlayValues[121] = d121
					ps307.OverlayValues[122] = d122
					ps307.OverlayValues[123] = d123
					ps307.OverlayValues[124] = d124
					ps307.OverlayValues[179] = d179
					ps307.OverlayValues[180] = d180
					ps307.OverlayValues[181] = d181
					ps307.OverlayValues[182] = d182
					ps307.OverlayValues[183] = d183
					ps307.OverlayValues[184] = d184
					ps307.OverlayValues[185] = d185
					ps307.OverlayValues[186] = d186
					ps307.OverlayValues[187] = d187
					ps307.OverlayValues[188] = d188
					ps307.OverlayValues[189] = d189
					ps307.OverlayValues[190] = d190
					ps307.OverlayValues[191] = d191
					ps307.OverlayValues[192] = d192
					ps307.OverlayValues[193] = d193
					ps307.OverlayValues[194] = d194
					ps307.OverlayValues[195] = d195
					ps307.OverlayValues[196] = d196
					ps307.OverlayValues[197] = d197
					ps307.OverlayValues[199] = d199
					ps307.OverlayValues[200] = d200
					ps307.OverlayValues[201] = d201
					ps307.OverlayValues[202] = d202
					ps307.OverlayValues[203] = d203
					ps307.OverlayValues[204] = d204
					ps307.OverlayValues[205] = d205
					ps307.OverlayValues[206] = d206
					ps307.OverlayValues[208] = d208
					ps307.OverlayValues[209] = d209
					ps307.OverlayValues[210] = d210
					ps307.OverlayValues[297] = d297
					ps307.OverlayValues[298] = d298
					ps307.OverlayValues[299] = d299
					ps307.OverlayValues[300] = d300
					ps307.OverlayValues[301] = d301
					ps307.OverlayValues[304] = d304
					ps307.OverlayValues[305] = d305
					snap308 := d8
					snap309 := d9
					snap310 := d10
					snap311 := d11
					snap312 := d12
					snap313 := d13
					snap314 := d14
					snap315 := d15
					snap316 := d16
					snap317 := d17
					snap318 := d18
					snap319 := d19
					snap320 := d20
					snap321 := d21
					snap322 := d22
					snap323 := d24
					snap324 := d26
					snap325 := d27
					snap326 := d30
					snap327 := d51
					snap328 := d52
					snap329 := d53
					snap330 := d54
					snap331 := d55
					snap332 := d57
					snap333 := d58
					snap334 := d59
					snap335 := d60
					snap336 := d61
					snap337 := d62
					snap338 := d63
					snap339 := d66
					snap340 := d103
					snap341 := d104
					snap342 := d105
					snap343 := d106
					snap344 := d107
					snap345 := d108
					snap346 := d110
					snap347 := d111
					snap348 := d112
					snap349 := d113
					snap350 := d114
					snap351 := d115
					snap352 := d116
					snap353 := d117
					snap354 := d118
					snap355 := d121
					snap356 := d122
					snap357 := d123
					snap358 := d124
					snap359 := d179
					snap360 := d180
					snap361 := d181
					snap362 := d182
					snap363 := d183
					snap364 := d184
					snap365 := d185
					snap366 := d186
					snap367 := d187
					snap368 := d188
					snap369 := d189
					snap370 := d190
					snap371 := d191
					snap372 := d192
					snap373 := d193
					snap374 := d194
					snap375 := d195
					snap376 := d196
					snap377 := d197
					snap378 := d199
					snap379 := d200
					snap380 := d201
					snap381 := d202
					snap382 := d203
					snap383 := d204
					snap384 := d205
					snap385 := d206
					snap386 := d208
					snap387 := d209
					snap388 := d210
					snap389 := d297
					snap390 := d298
					snap391 := d299
					snap392 := d300
					snap393 := d301
					snap394 := d304
					snap395 := d305
					alloc396 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps307)
					}
					ctx.RestoreAllocState(alloc396)
					d8 = snap308
					d9 = snap309
					d10 = snap310
					d11 = snap311
					d12 = snap312
					d13 = snap313
					d14 = snap314
					d15 = snap315
					d16 = snap316
					d17 = snap317
					d18 = snap318
					d19 = snap319
					d20 = snap320
					d21 = snap321
					d22 = snap322
					d24 = snap323
					d26 = snap324
					d27 = snap325
					d30 = snap326
					d51 = snap327
					d52 = snap328
					d53 = snap329
					d54 = snap330
					d55 = snap331
					d57 = snap332
					d58 = snap333
					d59 = snap334
					d60 = snap335
					d61 = snap336
					d62 = snap337
					d63 = snap338
					d66 = snap339
					d103 = snap340
					d104 = snap341
					d105 = snap342
					d106 = snap343
					d107 = snap344
					d108 = snap345
					d110 = snap346
					d111 = snap347
					d112 = snap348
					d113 = snap349
					d114 = snap350
					d115 = snap351
					d116 = snap352
					d117 = snap353
					d118 = snap354
					d121 = snap355
					d122 = snap356
					d123 = snap357
					d124 = snap358
					d179 = snap359
					d180 = snap360
					d181 = snap361
					d182 = snap362
					d183 = snap363
					d184 = snap364
					d185 = snap365
					d186 = snap366
					d187 = snap367
					d188 = snap368
					d189 = snap369
					d190 = snap370
					d191 = snap371
					d192 = snap372
					d193 = snap373
					d194 = snap374
					d195 = snap375
					d196 = snap376
					d197 = snap377
					d199 = snap378
					d200 = snap379
					d201 = snap380
					d202 = snap381
					d203 = snap382
					d204 = snap383
					d205 = snap384
					d206 = snap385
					d208 = snap386
					d209 = snap387
					d210 = snap388
					d297 = snap389
					d298 = snap390
					d299 = snap391
					d300 = snap392
					d301 = snap393
					d304 = snap394
					d305 = snap395
					if !bbs[13].Rendered {
						return bbs[13].RenderPS(ps306)
					}
					return result
					ctx.FreeDesc(&d300)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
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
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != LocNone {
						d300 = ps.OverlayValues[300]
					}
					if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != LocNone {
						d301 = ps.OverlayValues[301]
					}
					if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != LocNone {
						d304 = ps.OverlayValues[304]
					}
					if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != LocNone {
						d305 = ps.OverlayValues[305]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					d398 = ctx.EmitSliceElementAddress(&d17, &d15, 16)
					ctx.EnsureDesc(&d398)
					r23 := ctx.AllocRegExcept(d398.Reg)
					ctx.EmitMovRegMem(r23, d398.Reg, 8)
					ctx.EmitMovRegMem(d398.Reg, d398.Reg, 0)
					d397 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d398.Reg, Reg2: r23}
					ctx.BindReg(d398.Reg, &d397)
					ctx.BindReg(r23, &d397)
					ctx.EnsureDesc(&d397)
					d399 = d397
					_ = d399
					ctx.StabilizeDescForControlFlow(&d399)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl28 := ctx.ReserveLabel()
					_ = lbl28
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl28)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d400 JITValueDesc
					if d399.Loc == LocImm {
						d400 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d399.Imm.Float())}
					} else if d399.Type == tagFloat && d399.Loc == LocReg {
						d400 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d399.Reg}
						ctx.BindReg(d399.Reg, &d400)
						ctx.BindReg(d399.Reg, &d400)
					} else if d399.Type == tagFloat && d399.Loc == LocRegPair {
						ctx.FreeReg(d399.Reg)
						d400 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d399.Reg2}
						ctx.BindReg(d399.Reg2, &d400)
						ctx.BindReg(d399.Reg2, &d400)
					} else {
						d400 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d399}, 1)
						d400.Type = tagFloat
						ctx.BindReg(d400.Reg, &d400)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d400)
					ctx.FreeDesc(&d397)
					ctx.EnsureDesc(&d15)
					d402 = ctx.EmitSliceElementAddress(&d19, &d15, 16)
					ctx.EnsureDesc(&d402)
					r24 := ctx.AllocRegExcept(d402.Reg)
					ctx.EmitMovRegMem(r24, d402.Reg, 8)
					ctx.EmitMovRegMem(d402.Reg, d402.Reg, 0)
					d401 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d402.Reg, Reg2: r24}
					ctx.BindReg(d402.Reg, &d401)
					ctx.BindReg(r24, &d401)
					ctx.EnsureDesc(&d401)
					d403 = d401
					_ = d403
					ctx.StabilizeDescForControlFlow(&d403)
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl29 := ctx.ReserveLabel()
					_ = lbl29
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl29)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d404 JITValueDesc
					if d403.Loc == LocImm {
						d404 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d403.Imm.Float())}
					} else if d403.Type == tagFloat && d403.Loc == LocReg {
						d404 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d403.Reg}
						ctx.BindReg(d403.Reg, &d404)
						ctx.BindReg(d403.Reg, &d404)
					} else if d403.Type == tagFloat && d403.Loc == LocRegPair {
						ctx.FreeReg(d403.Reg)
						d404 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d403.Reg2}
						ctx.BindReg(d403.Reg2, &d404)
						ctx.BindReg(d403.Reg2, &d404)
					} else {
						d404 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d403}, 1)
						d404.Type = tagFloat
						ctx.BindReg(d404.Reg, &d404)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d404)
					ctx.FreeDesc(&d401)
					ctx.EnsureDesc(&d400)
					ctx.EnsureDesc(&d404)
					ctx.EnsureDescsTogether(&d400, &d404)
					var d405 JITValueDesc
					if d400.Loc == LocImm && d404.Loc == LocImm {
						d405 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d400.Imm.Float() * d404.Imm.Float())}
					} else if d400.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d404.Reg)
						_, xBits := d400.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d404.Reg)
						d405 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d405)
					} else if d404.Loc == LocImm {
						_, yBits := d404.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d400.Reg, RegR11)
						d405 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d400.Reg}
						ctx.BindReg(d400.Reg, &d405)
					} else {
						ctx.EmitMulFloat64(d400.Reg, d404.Reg)
						d405 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d400.Reg}
						ctx.BindReg(d400.Reg, &d405)
					}
					if d405.Loc == LocReg && d400.Loc == LocReg && d405.Reg == d400.Reg {
						ctx.TransferReg(d400.Reg)
						d400.Loc = LocNone
					}
					ctx.FreeDesc(&d400)
					ctx.FreeDesc(&d404)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d405)
					ctx.EnsureDescsTogether(&d14, &d405)
					var d406 JITValueDesc
					if d14.Loc == LocImm && d405.Loc == LocImm {
						d406 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d14.Imm.Float() + d405.Imm.Float())}
					} else if d14.Loc == LocImm {
						var scratch Reg
						if phiHomeOK6 && r4 != d405.Reg {
							scratch = r4
						} else {
							scratch = ctx.AllocRegExcept(d405.Reg)
						}
						_, xBits := d14.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d405.Reg)
						d406 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d406)
					} else if d405.Loc == LocImm {
						var scratch Reg
						if phiHomeOK6 {
							scratch = r4
						} else {
							scratch = ctx.AllocRegExcept(d14.Reg)
						}
						ctx.EmitMovRegReg(scratch, d14.Reg)
						_, yBits := d405.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d406 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d406)
					} else {
						var r25 Reg
						if phiHomeOK6 && r4 != d405.Reg {
							r25 = r4
						} else {
							r25 = ctx.AllocRegExcept(d14.Reg, d405.Reg)
						}
						ctx.EmitMovRegReg(r25, d14.Reg)
						ctx.EmitAddFloat64(r25, d405.Reg)
						d406 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r25}
						ctx.BindReg(r25, &d406)
					}
					if d406.Loc == LocReg && d14.Loc == LocReg && d406.Reg == d14.Reg {
						ctx.TransferReg(d14.Reg)
						d14.Loc = LocNone
					}
					ctx.FreeDesc(&d405)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					var d407 JITValueDesc
					if d15.Loc == LocImm {
						d407 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK7 {
							scratch = r5
						} else {
							scratch = ctx.AllocRegExcept(d15.Reg)
						}
						ctx.EmitMovRegReg(scratch, d15.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d407 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d407)
					}
					if d407.Loc == LocReg && d15.Loc == LocReg && d407.Reg == d15.Reg {
						ctx.TransferReg(d15.Reg)
						d15.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d406)
						if d406.Loc == LocReg {
							ctx.ProtectReg(d406.Reg)
						} else if d406.Loc == LocRegPair {
							ctx.ProtectReg(d406.Reg)
							ctx.ProtectReg(d406.Reg2)
						}
						ctx.SyncDesc(&d407)
						if d407.Loc == LocReg {
							ctx.ProtectReg(d407.Reg)
						} else if d407.Loc == LocRegPair {
							ctx.ProtectReg(d407.Reg)
							ctx.ProtectReg(d407.Reg2)
						}
						d408 = d406
						if d408.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d408)
						if phiHomeOK6 {
							ctx.EmitMovToReg(r4, d408)
						} else {
							ctx.EmitStoreToStack(d408, int32(bbs[10].PhiBase)+int32(0))
						}
						d409 = d407
						if d409.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d409)
						if phiHomeOK7 {
							ctx.EmitMovToReg(r5, d409)
						} else {
							ctx.EmitStoreToStack(d409, int32(bbs[10].PhiBase)+int32(16))
						}
						if d406.Loc == LocReg {
							ctx.UnprotectReg(d406.Reg)
						} else if d406.Loc == LocRegPair {
							ctx.UnprotectReg(d406.Reg)
							ctx.UnprotectReg(d406.Reg2)
						}
						if d407.Loc == LocReg {
							ctx.UnprotectReg(d407.Reg)
						} else if d407.Loc == LocRegPair {
							ctx.UnprotectReg(d407.Reg)
							ctx.UnprotectReg(d407.Reg2)
						}
					}
					ps410 := PhiState{General: ps.General}
					ps410.OverlayValues = make([]JITValueDesc, 410)
					ps410.OverlayValues[8] = d8
					ps410.OverlayValues[9] = d9
					ps410.OverlayValues[10] = d10
					ps410.OverlayValues[11] = d11
					ps410.OverlayValues[12] = d12
					ps410.OverlayValues[13] = d13
					ps410.OverlayValues[14] = d14
					ps410.OverlayValues[15] = d15
					ps410.OverlayValues[16] = d16
					ps410.OverlayValues[17] = d17
					ps410.OverlayValues[18] = d18
					ps410.OverlayValues[19] = d19
					ps410.OverlayValues[20] = d20
					ps410.OverlayValues[21] = d21
					ps410.OverlayValues[22] = d22
					ps410.OverlayValues[24] = d24
					ps410.OverlayValues[26] = d26
					ps410.OverlayValues[27] = d27
					ps410.OverlayValues[30] = d30
					ps410.OverlayValues[51] = d51
					ps410.OverlayValues[52] = d52
					ps410.OverlayValues[53] = d53
					ps410.OverlayValues[54] = d54
					ps410.OverlayValues[55] = d55
					ps410.OverlayValues[57] = d57
					ps410.OverlayValues[58] = d58
					ps410.OverlayValues[59] = d59
					ps410.OverlayValues[60] = d60
					ps410.OverlayValues[61] = d61
					ps410.OverlayValues[62] = d62
					ps410.OverlayValues[63] = d63
					ps410.OverlayValues[66] = d66
					ps410.OverlayValues[103] = d103
					ps410.OverlayValues[104] = d104
					ps410.OverlayValues[105] = d105
					ps410.OverlayValues[106] = d106
					ps410.OverlayValues[107] = d107
					ps410.OverlayValues[108] = d108
					ps410.OverlayValues[110] = d110
					ps410.OverlayValues[111] = d111
					ps410.OverlayValues[112] = d112
					ps410.OverlayValues[113] = d113
					ps410.OverlayValues[114] = d114
					ps410.OverlayValues[115] = d115
					ps410.OverlayValues[116] = d116
					ps410.OverlayValues[117] = d117
					ps410.OverlayValues[118] = d118
					ps410.OverlayValues[121] = d121
					ps410.OverlayValues[122] = d122
					ps410.OverlayValues[123] = d123
					ps410.OverlayValues[124] = d124
					ps410.OverlayValues[179] = d179
					ps410.OverlayValues[180] = d180
					ps410.OverlayValues[181] = d181
					ps410.OverlayValues[182] = d182
					ps410.OverlayValues[183] = d183
					ps410.OverlayValues[184] = d184
					ps410.OverlayValues[185] = d185
					ps410.OverlayValues[186] = d186
					ps410.OverlayValues[187] = d187
					ps410.OverlayValues[188] = d188
					ps410.OverlayValues[189] = d189
					ps410.OverlayValues[190] = d190
					ps410.OverlayValues[191] = d191
					ps410.OverlayValues[192] = d192
					ps410.OverlayValues[193] = d193
					ps410.OverlayValues[194] = d194
					ps410.OverlayValues[195] = d195
					ps410.OverlayValues[196] = d196
					ps410.OverlayValues[197] = d197
					ps410.OverlayValues[199] = d199
					ps410.OverlayValues[200] = d200
					ps410.OverlayValues[201] = d201
					ps410.OverlayValues[202] = d202
					ps410.OverlayValues[203] = d203
					ps410.OverlayValues[204] = d204
					ps410.OverlayValues[205] = d205
					ps410.OverlayValues[206] = d206
					ps410.OverlayValues[208] = d208
					ps410.OverlayValues[209] = d209
					ps410.OverlayValues[210] = d210
					ps410.OverlayValues[297] = d297
					ps410.OverlayValues[298] = d298
					ps410.OverlayValues[299] = d299
					ps410.OverlayValues[300] = d300
					ps410.OverlayValues[301] = d301
					ps410.OverlayValues[304] = d304
					ps410.OverlayValues[305] = d305
					ps410.OverlayValues[397] = d397
					ps410.OverlayValues[398] = d398
					ps410.OverlayValues[399] = d399
					ps410.OverlayValues[400] = d400
					ps410.OverlayValues[401] = d401
					ps410.OverlayValues[402] = d402
					ps410.OverlayValues[403] = d403
					ps410.OverlayValues[404] = d404
					ps410.OverlayValues[405] = d405
					ps410.OverlayValues[406] = d406
					ps410.OverlayValues[407] = d407
					ps410.OverlayValues[408] = d408
					ps410.OverlayValues[409] = d409
					ps410.PhiValues = make([]JITValueDesc, 2)
					d411 = d406
					ps410.PhiValues[0] = d411
					d412 = d407
					ps410.PhiValues[1] = d412
					if ps410.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps410)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
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
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != LocNone {
						d300 = ps.OverlayValues[300]
					}
					if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != LocNone {
						d301 = ps.OverlayValues[301]
					}
					if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != LocNone {
						d304 = ps.OverlayValues[304]
					}
					if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != LocNone {
						d305 = ps.OverlayValues[305]
					}
					if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != LocNone {
						d397 = ps.OverlayValues[397]
					}
					if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != LocNone {
						d398 = ps.OverlayValues[398]
					}
					if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != LocNone {
						d399 = ps.OverlayValues[399]
					}
					if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != LocNone {
						d400 = ps.OverlayValues[400]
					}
					if len(ps.OverlayValues) > 401 && ps.OverlayValues[401].Loc != LocNone {
						d401 = ps.OverlayValues[401]
					}
					if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != LocNone {
						d402 = ps.OverlayValues[402]
					}
					if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != LocNone {
						d403 = ps.OverlayValues[403]
					}
					if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != LocNone {
						d404 = ps.OverlayValues[404]
					}
					if len(ps.OverlayValues) > 405 && ps.OverlayValues[405].Loc != LocNone {
						d405 = ps.OverlayValues[405]
					}
					if len(ps.OverlayValues) > 406 && ps.OverlayValues[406].Loc != LocNone {
						d406 = ps.OverlayValues[406]
					}
					if len(ps.OverlayValues) > 407 && ps.OverlayValues[407].Loc != LocNone {
						d407 = ps.OverlayValues[407]
					}
					if len(ps.OverlayValues) > 408 && ps.OverlayValues[408].Loc != LocNone {
						d408 = ps.OverlayValues[408]
					}
					if len(ps.OverlayValues) > 409 && ps.OverlayValues[409].Loc != LocNone {
						d409 = ps.OverlayValues[409]
					}
					if len(ps.OverlayValues) > 411 && ps.OverlayValues[411].Loc != LocNone {
						d411 = ps.OverlayValues[411]
					}
					if len(ps.OverlayValues) > 412 && ps.OverlayValues[412].Loc != LocNone {
						d412 = ps.OverlayValues[412]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					var d413 JITValueDesc
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d413 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d413.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d413.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d413.Reg, &d413)
						ctx.BindReg(d413.Reg2, &d413)
					} else {
						d413 = d8
					}
					d414 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("EUCLIDEAN")}
					var d415 JITValueDesc
					if d414.Loc == LocImm {
						ctx.TrackImm(d414.Imm)
						ptrWord, _ := d414.Imm.RawWords()
						d415 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d415.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d415.Reg2, uint64(len(d414.Imm.String())))
						ctx.BindReg(d415.Reg, &d415)
						ctx.BindReg(d415.Reg2, &d415)
					} else {
						d415 = d414
					}
					d416 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d413, d415}, 1)
					ctx.EmitAndRegImm32(d416.Reg, 1)
					d416.Type = tagBool
					ctx.BindReg(d416.Reg, &d416)
					ctx.FreeDesc(&d8)
					d417 = d416
					ctx.EnsureDesc(&d417)
					if d417.Loc != LocImm && d417.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d417.Loc == LocImm {
						if d417.Imm.Bool() {
							if ps.General {
							}
							ps418 := PhiState{General: ps.General}
							ps418.OverlayValues = make([]JITValueDesc, 418)
							ps418.OverlayValues[8] = d8
							ps418.OverlayValues[9] = d9
							ps418.OverlayValues[10] = d10
							ps418.OverlayValues[11] = d11
							ps418.OverlayValues[12] = d12
							ps418.OverlayValues[13] = d13
							ps418.OverlayValues[14] = d14
							ps418.OverlayValues[15] = d15
							ps418.OverlayValues[16] = d16
							ps418.OverlayValues[17] = d17
							ps418.OverlayValues[18] = d18
							ps418.OverlayValues[19] = d19
							ps418.OverlayValues[20] = d20
							ps418.OverlayValues[21] = d21
							ps418.OverlayValues[22] = d22
							ps418.OverlayValues[24] = d24
							ps418.OverlayValues[26] = d26
							ps418.OverlayValues[27] = d27
							ps418.OverlayValues[30] = d30
							ps418.OverlayValues[51] = d51
							ps418.OverlayValues[52] = d52
							ps418.OverlayValues[53] = d53
							ps418.OverlayValues[54] = d54
							ps418.OverlayValues[55] = d55
							ps418.OverlayValues[57] = d57
							ps418.OverlayValues[58] = d58
							ps418.OverlayValues[59] = d59
							ps418.OverlayValues[60] = d60
							ps418.OverlayValues[61] = d61
							ps418.OverlayValues[62] = d62
							ps418.OverlayValues[63] = d63
							ps418.OverlayValues[66] = d66
							ps418.OverlayValues[103] = d103
							ps418.OverlayValues[104] = d104
							ps418.OverlayValues[105] = d105
							ps418.OverlayValues[106] = d106
							ps418.OverlayValues[107] = d107
							ps418.OverlayValues[108] = d108
							ps418.OverlayValues[110] = d110
							ps418.OverlayValues[111] = d111
							ps418.OverlayValues[112] = d112
							ps418.OverlayValues[113] = d113
							ps418.OverlayValues[114] = d114
							ps418.OverlayValues[115] = d115
							ps418.OverlayValues[116] = d116
							ps418.OverlayValues[117] = d117
							ps418.OverlayValues[118] = d118
							ps418.OverlayValues[121] = d121
							ps418.OverlayValues[122] = d122
							ps418.OverlayValues[123] = d123
							ps418.OverlayValues[124] = d124
							ps418.OverlayValues[179] = d179
							ps418.OverlayValues[180] = d180
							ps418.OverlayValues[181] = d181
							ps418.OverlayValues[182] = d182
							ps418.OverlayValues[183] = d183
							ps418.OverlayValues[184] = d184
							ps418.OverlayValues[185] = d185
							ps418.OverlayValues[186] = d186
							ps418.OverlayValues[187] = d187
							ps418.OverlayValues[188] = d188
							ps418.OverlayValues[189] = d189
							ps418.OverlayValues[190] = d190
							ps418.OverlayValues[191] = d191
							ps418.OverlayValues[192] = d192
							ps418.OverlayValues[193] = d193
							ps418.OverlayValues[194] = d194
							ps418.OverlayValues[195] = d195
							ps418.OverlayValues[196] = d196
							ps418.OverlayValues[197] = d197
							ps418.OverlayValues[199] = d199
							ps418.OverlayValues[200] = d200
							ps418.OverlayValues[201] = d201
							ps418.OverlayValues[202] = d202
							ps418.OverlayValues[203] = d203
							ps418.OverlayValues[204] = d204
							ps418.OverlayValues[205] = d205
							ps418.OverlayValues[206] = d206
							ps418.OverlayValues[208] = d208
							ps418.OverlayValues[209] = d209
							ps418.OverlayValues[210] = d210
							ps418.OverlayValues[297] = d297
							ps418.OverlayValues[298] = d298
							ps418.OverlayValues[299] = d299
							ps418.OverlayValues[300] = d300
							ps418.OverlayValues[301] = d301
							ps418.OverlayValues[304] = d304
							ps418.OverlayValues[305] = d305
							ps418.OverlayValues[397] = d397
							ps418.OverlayValues[398] = d398
							ps418.OverlayValues[399] = d399
							ps418.OverlayValues[400] = d400
							ps418.OverlayValues[401] = d401
							ps418.OverlayValues[402] = d402
							ps418.OverlayValues[403] = d403
							ps418.OverlayValues[404] = d404
							ps418.OverlayValues[405] = d405
							ps418.OverlayValues[406] = d406
							ps418.OverlayValues[407] = d407
							ps418.OverlayValues[408] = d408
							ps418.OverlayValues[409] = d409
							ps418.OverlayValues[411] = d411
							ps418.OverlayValues[412] = d412
							ps418.OverlayValues[413] = d413
							ps418.OverlayValues[414] = d414
							ps418.OverlayValues[415] = d415
							ps418.OverlayValues[416] = d416
							ps418.OverlayValues[417] = d417
							return bbs[14].RenderPS(ps418)
						}
						if ps.General {
							ctx.SyncDesc(&d14)
							if d14.Loc == LocReg {
								ctx.ProtectReg(d14.Reg)
							} else if d14.Loc == LocRegPair {
								ctx.ProtectReg(d14.Reg)
								ctx.ProtectReg(d14.Reg2)
							}
							d419 = d14
							if d419.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d419)
							ctx.EmitStoreToStack(d419, int32(bbs[4].PhiBase)+int32(0))
							if d14.Loc == LocReg {
								ctx.UnprotectReg(d14.Reg)
							} else if d14.Loc == LocRegPair {
								ctx.UnprotectReg(d14.Reg)
								ctx.UnprotectReg(d14.Reg2)
							}
						}
						ps420 := PhiState{General: ps.General}
						ps420.OverlayValues = make([]JITValueDesc, 420)
						ps420.OverlayValues[8] = d8
						ps420.OverlayValues[9] = d9
						ps420.OverlayValues[10] = d10
						ps420.OverlayValues[11] = d11
						ps420.OverlayValues[12] = d12
						ps420.OverlayValues[13] = d13
						ps420.OverlayValues[14] = d14
						ps420.OverlayValues[15] = d15
						ps420.OverlayValues[16] = d16
						ps420.OverlayValues[17] = d17
						ps420.OverlayValues[18] = d18
						ps420.OverlayValues[19] = d19
						ps420.OverlayValues[20] = d20
						ps420.OverlayValues[21] = d21
						ps420.OverlayValues[22] = d22
						ps420.OverlayValues[24] = d24
						ps420.OverlayValues[26] = d26
						ps420.OverlayValues[27] = d27
						ps420.OverlayValues[30] = d30
						ps420.OverlayValues[51] = d51
						ps420.OverlayValues[52] = d52
						ps420.OverlayValues[53] = d53
						ps420.OverlayValues[54] = d54
						ps420.OverlayValues[55] = d55
						ps420.OverlayValues[57] = d57
						ps420.OverlayValues[58] = d58
						ps420.OverlayValues[59] = d59
						ps420.OverlayValues[60] = d60
						ps420.OverlayValues[61] = d61
						ps420.OverlayValues[62] = d62
						ps420.OverlayValues[63] = d63
						ps420.OverlayValues[66] = d66
						ps420.OverlayValues[103] = d103
						ps420.OverlayValues[104] = d104
						ps420.OverlayValues[105] = d105
						ps420.OverlayValues[106] = d106
						ps420.OverlayValues[107] = d107
						ps420.OverlayValues[108] = d108
						ps420.OverlayValues[110] = d110
						ps420.OverlayValues[111] = d111
						ps420.OverlayValues[112] = d112
						ps420.OverlayValues[113] = d113
						ps420.OverlayValues[114] = d114
						ps420.OverlayValues[115] = d115
						ps420.OverlayValues[116] = d116
						ps420.OverlayValues[117] = d117
						ps420.OverlayValues[118] = d118
						ps420.OverlayValues[121] = d121
						ps420.OverlayValues[122] = d122
						ps420.OverlayValues[123] = d123
						ps420.OverlayValues[124] = d124
						ps420.OverlayValues[179] = d179
						ps420.OverlayValues[180] = d180
						ps420.OverlayValues[181] = d181
						ps420.OverlayValues[182] = d182
						ps420.OverlayValues[183] = d183
						ps420.OverlayValues[184] = d184
						ps420.OverlayValues[185] = d185
						ps420.OverlayValues[186] = d186
						ps420.OverlayValues[187] = d187
						ps420.OverlayValues[188] = d188
						ps420.OverlayValues[189] = d189
						ps420.OverlayValues[190] = d190
						ps420.OverlayValues[191] = d191
						ps420.OverlayValues[192] = d192
						ps420.OverlayValues[193] = d193
						ps420.OverlayValues[194] = d194
						ps420.OverlayValues[195] = d195
						ps420.OverlayValues[196] = d196
						ps420.OverlayValues[197] = d197
						ps420.OverlayValues[199] = d199
						ps420.OverlayValues[200] = d200
						ps420.OverlayValues[201] = d201
						ps420.OverlayValues[202] = d202
						ps420.OverlayValues[203] = d203
						ps420.OverlayValues[204] = d204
						ps420.OverlayValues[205] = d205
						ps420.OverlayValues[206] = d206
						ps420.OverlayValues[208] = d208
						ps420.OverlayValues[209] = d209
						ps420.OverlayValues[210] = d210
						ps420.OverlayValues[297] = d297
						ps420.OverlayValues[298] = d298
						ps420.OverlayValues[299] = d299
						ps420.OverlayValues[300] = d300
						ps420.OverlayValues[301] = d301
						ps420.OverlayValues[304] = d304
						ps420.OverlayValues[305] = d305
						ps420.OverlayValues[397] = d397
						ps420.OverlayValues[398] = d398
						ps420.OverlayValues[399] = d399
						ps420.OverlayValues[400] = d400
						ps420.OverlayValues[401] = d401
						ps420.OverlayValues[402] = d402
						ps420.OverlayValues[403] = d403
						ps420.OverlayValues[404] = d404
						ps420.OverlayValues[405] = d405
						ps420.OverlayValues[406] = d406
						ps420.OverlayValues[407] = d407
						ps420.OverlayValues[408] = d408
						ps420.OverlayValues[409] = d409
						ps420.OverlayValues[411] = d411
						ps420.OverlayValues[412] = d412
						ps420.OverlayValues[413] = d413
						ps420.OverlayValues[414] = d414
						ps420.OverlayValues[415] = d415
						ps420.OverlayValues[416] = d416
						ps420.OverlayValues[417] = d417
						ps420.OverlayValues[419] = d419
						ps420.PhiValues = make([]JITValueDesc, 1)
						d421 = d14
						ps420.PhiValues[0] = d421
						return bbs[4].RenderPS(ps420)
					}
					if !ps.General {
						ps.General = true
						return bbs[12].RenderPS(ps)
					}
					lbl30 := ctx.ReserveLabel()
					lbl31 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d417.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl30)
					ctx.EmitJmp(lbl31)
					ctx.MarkLabel(lbl30)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl31)
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d422 = d14
					if d422.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d422)
					ctx.EmitStoreToStack(d422, int32(bbs[4].PhiBase)+int32(0))
					if d14.Loc == LocReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ps423 := PhiState{General: true}
					ps423.OverlayValues = make([]JITValueDesc, 423)
					ps423.OverlayValues[8] = d8
					ps423.OverlayValues[9] = d9
					ps423.OverlayValues[10] = d10
					ps423.OverlayValues[11] = d11
					ps423.OverlayValues[12] = d12
					ps423.OverlayValues[13] = d13
					ps423.OverlayValues[14] = d14
					ps423.OverlayValues[15] = d15
					ps423.OverlayValues[16] = d16
					ps423.OverlayValues[17] = d17
					ps423.OverlayValues[18] = d18
					ps423.OverlayValues[19] = d19
					ps423.OverlayValues[20] = d20
					ps423.OverlayValues[21] = d21
					ps423.OverlayValues[22] = d22
					ps423.OverlayValues[24] = d24
					ps423.OverlayValues[26] = d26
					ps423.OverlayValues[27] = d27
					ps423.OverlayValues[30] = d30
					ps423.OverlayValues[51] = d51
					ps423.OverlayValues[52] = d52
					ps423.OverlayValues[53] = d53
					ps423.OverlayValues[54] = d54
					ps423.OverlayValues[55] = d55
					ps423.OverlayValues[57] = d57
					ps423.OverlayValues[58] = d58
					ps423.OverlayValues[59] = d59
					ps423.OverlayValues[60] = d60
					ps423.OverlayValues[61] = d61
					ps423.OverlayValues[62] = d62
					ps423.OverlayValues[63] = d63
					ps423.OverlayValues[66] = d66
					ps423.OverlayValues[103] = d103
					ps423.OverlayValues[104] = d104
					ps423.OverlayValues[105] = d105
					ps423.OverlayValues[106] = d106
					ps423.OverlayValues[107] = d107
					ps423.OverlayValues[108] = d108
					ps423.OverlayValues[110] = d110
					ps423.OverlayValues[111] = d111
					ps423.OverlayValues[112] = d112
					ps423.OverlayValues[113] = d113
					ps423.OverlayValues[114] = d114
					ps423.OverlayValues[115] = d115
					ps423.OverlayValues[116] = d116
					ps423.OverlayValues[117] = d117
					ps423.OverlayValues[118] = d118
					ps423.OverlayValues[121] = d121
					ps423.OverlayValues[122] = d122
					ps423.OverlayValues[123] = d123
					ps423.OverlayValues[124] = d124
					ps423.OverlayValues[179] = d179
					ps423.OverlayValues[180] = d180
					ps423.OverlayValues[181] = d181
					ps423.OverlayValues[182] = d182
					ps423.OverlayValues[183] = d183
					ps423.OverlayValues[184] = d184
					ps423.OverlayValues[185] = d185
					ps423.OverlayValues[186] = d186
					ps423.OverlayValues[187] = d187
					ps423.OverlayValues[188] = d188
					ps423.OverlayValues[189] = d189
					ps423.OverlayValues[190] = d190
					ps423.OverlayValues[191] = d191
					ps423.OverlayValues[192] = d192
					ps423.OverlayValues[193] = d193
					ps423.OverlayValues[194] = d194
					ps423.OverlayValues[195] = d195
					ps423.OverlayValues[196] = d196
					ps423.OverlayValues[197] = d197
					ps423.OverlayValues[199] = d199
					ps423.OverlayValues[200] = d200
					ps423.OverlayValues[201] = d201
					ps423.OverlayValues[202] = d202
					ps423.OverlayValues[203] = d203
					ps423.OverlayValues[204] = d204
					ps423.OverlayValues[205] = d205
					ps423.OverlayValues[206] = d206
					ps423.OverlayValues[208] = d208
					ps423.OverlayValues[209] = d209
					ps423.OverlayValues[210] = d210
					ps423.OverlayValues[297] = d297
					ps423.OverlayValues[298] = d298
					ps423.OverlayValues[299] = d299
					ps423.OverlayValues[300] = d300
					ps423.OverlayValues[301] = d301
					ps423.OverlayValues[304] = d304
					ps423.OverlayValues[305] = d305
					ps423.OverlayValues[397] = d397
					ps423.OverlayValues[398] = d398
					ps423.OverlayValues[399] = d399
					ps423.OverlayValues[400] = d400
					ps423.OverlayValues[401] = d401
					ps423.OverlayValues[402] = d402
					ps423.OverlayValues[403] = d403
					ps423.OverlayValues[404] = d404
					ps423.OverlayValues[405] = d405
					ps423.OverlayValues[406] = d406
					ps423.OverlayValues[407] = d407
					ps423.OverlayValues[408] = d408
					ps423.OverlayValues[409] = d409
					ps423.OverlayValues[411] = d411
					ps423.OverlayValues[412] = d412
					ps423.OverlayValues[413] = d413
					ps423.OverlayValues[414] = d414
					ps423.OverlayValues[415] = d415
					ps423.OverlayValues[416] = d416
					ps423.OverlayValues[417] = d417
					ps423.OverlayValues[419] = d419
					ps423.OverlayValues[421] = d421
					ps423.OverlayValues[422] = d422
					ps424 := PhiState{General: true}
					ps424.OverlayValues = make([]JITValueDesc, 423)
					ps424.OverlayValues[8] = d8
					ps424.OverlayValues[9] = d9
					ps424.OverlayValues[10] = d10
					ps424.OverlayValues[11] = d11
					ps424.OverlayValues[12] = d12
					ps424.OverlayValues[13] = d13
					ps424.OverlayValues[14] = d14
					ps424.OverlayValues[15] = d15
					ps424.OverlayValues[16] = d16
					ps424.OverlayValues[17] = d17
					ps424.OverlayValues[18] = d18
					ps424.OverlayValues[19] = d19
					ps424.OverlayValues[20] = d20
					ps424.OverlayValues[21] = d21
					ps424.OverlayValues[22] = d22
					ps424.OverlayValues[24] = d24
					ps424.OverlayValues[26] = d26
					ps424.OverlayValues[27] = d27
					ps424.OverlayValues[30] = d30
					ps424.OverlayValues[51] = d51
					ps424.OverlayValues[52] = d52
					ps424.OverlayValues[53] = d53
					ps424.OverlayValues[54] = d54
					ps424.OverlayValues[55] = d55
					ps424.OverlayValues[57] = d57
					ps424.OverlayValues[58] = d58
					ps424.OverlayValues[59] = d59
					ps424.OverlayValues[60] = d60
					ps424.OverlayValues[61] = d61
					ps424.OverlayValues[62] = d62
					ps424.OverlayValues[63] = d63
					ps424.OverlayValues[66] = d66
					ps424.OverlayValues[103] = d103
					ps424.OverlayValues[104] = d104
					ps424.OverlayValues[105] = d105
					ps424.OverlayValues[106] = d106
					ps424.OverlayValues[107] = d107
					ps424.OverlayValues[108] = d108
					ps424.OverlayValues[110] = d110
					ps424.OverlayValues[111] = d111
					ps424.OverlayValues[112] = d112
					ps424.OverlayValues[113] = d113
					ps424.OverlayValues[114] = d114
					ps424.OverlayValues[115] = d115
					ps424.OverlayValues[116] = d116
					ps424.OverlayValues[117] = d117
					ps424.OverlayValues[118] = d118
					ps424.OverlayValues[121] = d121
					ps424.OverlayValues[122] = d122
					ps424.OverlayValues[123] = d123
					ps424.OverlayValues[124] = d124
					ps424.OverlayValues[179] = d179
					ps424.OverlayValues[180] = d180
					ps424.OverlayValues[181] = d181
					ps424.OverlayValues[182] = d182
					ps424.OverlayValues[183] = d183
					ps424.OverlayValues[184] = d184
					ps424.OverlayValues[185] = d185
					ps424.OverlayValues[186] = d186
					ps424.OverlayValues[187] = d187
					ps424.OverlayValues[188] = d188
					ps424.OverlayValues[189] = d189
					ps424.OverlayValues[190] = d190
					ps424.OverlayValues[191] = d191
					ps424.OverlayValues[192] = d192
					ps424.OverlayValues[193] = d193
					ps424.OverlayValues[194] = d194
					ps424.OverlayValues[195] = d195
					ps424.OverlayValues[196] = d196
					ps424.OverlayValues[197] = d197
					ps424.OverlayValues[199] = d199
					ps424.OverlayValues[200] = d200
					ps424.OverlayValues[201] = d201
					ps424.OverlayValues[202] = d202
					ps424.OverlayValues[203] = d203
					ps424.OverlayValues[204] = d204
					ps424.OverlayValues[205] = d205
					ps424.OverlayValues[206] = d206
					ps424.OverlayValues[208] = d208
					ps424.OverlayValues[209] = d209
					ps424.OverlayValues[210] = d210
					ps424.OverlayValues[297] = d297
					ps424.OverlayValues[298] = d298
					ps424.OverlayValues[299] = d299
					ps424.OverlayValues[300] = d300
					ps424.OverlayValues[301] = d301
					ps424.OverlayValues[304] = d304
					ps424.OverlayValues[305] = d305
					ps424.OverlayValues[397] = d397
					ps424.OverlayValues[398] = d398
					ps424.OverlayValues[399] = d399
					ps424.OverlayValues[400] = d400
					ps424.OverlayValues[401] = d401
					ps424.OverlayValues[402] = d402
					ps424.OverlayValues[403] = d403
					ps424.OverlayValues[404] = d404
					ps424.OverlayValues[405] = d405
					ps424.OverlayValues[406] = d406
					ps424.OverlayValues[407] = d407
					ps424.OverlayValues[408] = d408
					ps424.OverlayValues[409] = d409
					ps424.OverlayValues[411] = d411
					ps424.OverlayValues[412] = d412
					ps424.OverlayValues[413] = d413
					ps424.OverlayValues[414] = d414
					ps424.OverlayValues[415] = d415
					ps424.OverlayValues[416] = d416
					ps424.OverlayValues[417] = d417
					ps424.OverlayValues[419] = d419
					ps424.OverlayValues[421] = d421
					ps424.OverlayValues[422] = d422
					ps424.PhiValues = make([]JITValueDesc, 1)
					d425 = d14
					ps424.PhiValues[0] = d425
					snap426 := d8
					snap427 := d9
					snap428 := d10
					snap429 := d11
					snap430 := d12
					snap431 := d13
					snap432 := d14
					snap433 := d15
					snap434 := d16
					snap435 := d17
					snap436 := d18
					snap437 := d19
					snap438 := d20
					snap439 := d21
					snap440 := d22
					snap441 := d24
					snap442 := d26
					snap443 := d27
					snap444 := d30
					snap445 := d51
					snap446 := d52
					snap447 := d53
					snap448 := d54
					snap449 := d55
					snap450 := d57
					snap451 := d58
					snap452 := d59
					snap453 := d60
					snap454 := d61
					snap455 := d62
					snap456 := d63
					snap457 := d66
					snap458 := d103
					snap459 := d104
					snap460 := d105
					snap461 := d106
					snap462 := d107
					snap463 := d108
					snap464 := d110
					snap465 := d111
					snap466 := d112
					snap467 := d113
					snap468 := d114
					snap469 := d115
					snap470 := d116
					snap471 := d117
					snap472 := d118
					snap473 := d121
					snap474 := d122
					snap475 := d123
					snap476 := d124
					snap477 := d179
					snap478 := d180
					snap479 := d181
					snap480 := d182
					snap481 := d183
					snap482 := d184
					snap483 := d185
					snap484 := d186
					snap485 := d187
					snap486 := d188
					snap487 := d189
					snap488 := d190
					snap489 := d191
					snap490 := d192
					snap491 := d193
					snap492 := d194
					snap493 := d195
					snap494 := d196
					snap495 := d197
					snap496 := d199
					snap497 := d200
					snap498 := d201
					snap499 := d202
					snap500 := d203
					snap501 := d204
					snap502 := d205
					snap503 := d206
					snap504 := d208
					snap505 := d209
					snap506 := d210
					snap507 := d297
					snap508 := d298
					snap509 := d299
					snap510 := d300
					snap511 := d301
					snap512 := d304
					snap513 := d305
					snap514 := d397
					snap515 := d398
					snap516 := d399
					snap517 := d400
					snap518 := d401
					snap519 := d402
					snap520 := d403
					snap521 := d404
					snap522 := d405
					snap523 := d406
					snap524 := d407
					snap525 := d408
					snap526 := d409
					snap527 := d411
					snap528 := d412
					snap529 := d413
					snap530 := d414
					snap531 := d415
					snap532 := d416
					snap533 := d417
					snap534 := d419
					snap535 := d421
					snap536 := d422
					snap537 := d425
					alloc538 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps424)
					}
					ctx.RestoreAllocState(alloc538)
					d8 = snap426
					d9 = snap427
					d10 = snap428
					d11 = snap429
					d12 = snap430
					d13 = snap431
					d14 = snap432
					d15 = snap433
					d16 = snap434
					d17 = snap435
					d18 = snap436
					d19 = snap437
					d20 = snap438
					d21 = snap439
					d22 = snap440
					d24 = snap441
					d26 = snap442
					d27 = snap443
					d30 = snap444
					d51 = snap445
					d52 = snap446
					d53 = snap447
					d54 = snap448
					d55 = snap449
					d57 = snap450
					d58 = snap451
					d59 = snap452
					d60 = snap453
					d61 = snap454
					d62 = snap455
					d63 = snap456
					d66 = snap457
					d103 = snap458
					d104 = snap459
					d105 = snap460
					d106 = snap461
					d107 = snap462
					d108 = snap463
					d110 = snap464
					d111 = snap465
					d112 = snap466
					d113 = snap467
					d114 = snap468
					d115 = snap469
					d116 = snap470
					d117 = snap471
					d118 = snap472
					d121 = snap473
					d122 = snap474
					d123 = snap475
					d124 = snap476
					d179 = snap477
					d180 = snap478
					d181 = snap479
					d182 = snap480
					d183 = snap481
					d184 = snap482
					d185 = snap483
					d186 = snap484
					d187 = snap485
					d188 = snap486
					d189 = snap487
					d190 = snap488
					d191 = snap489
					d192 = snap490
					d193 = snap491
					d194 = snap492
					d195 = snap493
					d196 = snap494
					d197 = snap495
					d199 = snap496
					d200 = snap497
					d201 = snap498
					d202 = snap499
					d203 = snap500
					d204 = snap501
					d205 = snap502
					d206 = snap503
					d208 = snap504
					d209 = snap505
					d210 = snap506
					d297 = snap507
					d298 = snap508
					d299 = snap509
					d300 = snap510
					d301 = snap511
					d304 = snap512
					d305 = snap513
					d397 = snap514
					d398 = snap515
					d399 = snap516
					d400 = snap517
					d401 = snap518
					d402 = snap519
					d403 = snap520
					d404 = snap521
					d405 = snap522
					d406 = snap523
					d407 = snap524
					d408 = snap525
					d409 = snap526
					d411 = snap527
					d412 = snap528
					d413 = snap529
					d414 = snap530
					d415 = snap531
					d416 = snap532
					d417 = snap533
					d419 = snap534
					d421 = snap535
					d422 = snap536
					d425 = snap537
					if !bbs[14].Rendered {
						return bbs[14].RenderPS(ps423)
					}
					return result
					ctx.FreeDesc(&d416)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
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
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != LocNone {
						d300 = ps.OverlayValues[300]
					}
					if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != LocNone {
						d301 = ps.OverlayValues[301]
					}
					if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != LocNone {
						d304 = ps.OverlayValues[304]
					}
					if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != LocNone {
						d305 = ps.OverlayValues[305]
					}
					if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != LocNone {
						d397 = ps.OverlayValues[397]
					}
					if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != LocNone {
						d398 = ps.OverlayValues[398]
					}
					if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != LocNone {
						d399 = ps.OverlayValues[399]
					}
					if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != LocNone {
						d400 = ps.OverlayValues[400]
					}
					if len(ps.OverlayValues) > 401 && ps.OverlayValues[401].Loc != LocNone {
						d401 = ps.OverlayValues[401]
					}
					if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != LocNone {
						d402 = ps.OverlayValues[402]
					}
					if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != LocNone {
						d403 = ps.OverlayValues[403]
					}
					if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != LocNone {
						d404 = ps.OverlayValues[404]
					}
					if len(ps.OverlayValues) > 405 && ps.OverlayValues[405].Loc != LocNone {
						d405 = ps.OverlayValues[405]
					}
					if len(ps.OverlayValues) > 406 && ps.OverlayValues[406].Loc != LocNone {
						d406 = ps.OverlayValues[406]
					}
					if len(ps.OverlayValues) > 407 && ps.OverlayValues[407].Loc != LocNone {
						d407 = ps.OverlayValues[407]
					}
					if len(ps.OverlayValues) > 408 && ps.OverlayValues[408].Loc != LocNone {
						d408 = ps.OverlayValues[408]
					}
					if len(ps.OverlayValues) > 409 && ps.OverlayValues[409].Loc != LocNone {
						d409 = ps.OverlayValues[409]
					}
					if len(ps.OverlayValues) > 411 && ps.OverlayValues[411].Loc != LocNone {
						d411 = ps.OverlayValues[411]
					}
					if len(ps.OverlayValues) > 412 && ps.OverlayValues[412].Loc != LocNone {
						d412 = ps.OverlayValues[412]
					}
					if len(ps.OverlayValues) > 413 && ps.OverlayValues[413].Loc != LocNone {
						d413 = ps.OverlayValues[413]
					}
					if len(ps.OverlayValues) > 414 && ps.OverlayValues[414].Loc != LocNone {
						d414 = ps.OverlayValues[414]
					}
					if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != LocNone {
						d415 = ps.OverlayValues[415]
					}
					if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != LocNone {
						d416 = ps.OverlayValues[416]
					}
					if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != LocNone {
						d417 = ps.OverlayValues[417]
					}
					if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != LocNone {
						d419 = ps.OverlayValues[419]
					}
					if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != LocNone {
						d421 = ps.OverlayValues[421]
					}
					if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != LocNone {
						d422 = ps.OverlayValues[422]
					}
					if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != LocNone {
						d425 = ps.OverlayValues[425]
					}
					ctx.ReclaimUntrackedRegs()
					var d539 JITValueDesc
					if d19.SliceSizeKnown {
						d539 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d539 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d539 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d539 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d539 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d539)
					ctx.EnsureDescsTogether(&d15, &d539)
					var d540 JITValueDesc
					if d15.Loc == LocImm && d539.Loc == LocImm {
						d540 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d539.Imm.Int())}
					} else if d539.Loc == LocImm {
						r26 := ctx.AllocReg()
						if d539.Imm.Int() >= -2147483648 && d539.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d539.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d539.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, RegR11)
						}
						ctx.EmitSetcc(r26, CondSignedLess)
						d540 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d540)
					} else if d15.Loc == LocImm {
						r27 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d539.Reg)
						ctx.EmitSetcc(r27, CondSignedLess)
						d540 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r27}
						ctx.BindReg(r27, &d540)
					} else {
						r28 := ctx.AllocReg()
						ctx.EmitCmpInt64(d15.Reg, d539.Reg)
						ctx.EmitSetcc(r28, CondSignedLess)
						d540 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r28}
						ctx.BindReg(r28, &d540)
					}
					ctx.FreeDesc(&d15)
					ctx.FreeDesc(&d539)
					d541 = d540
					ctx.EnsureDesc(&d541)
					if d541.Loc != LocImm && d541.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d541.Loc == LocImm {
						if d541.Imm.Bool() {
							if ps.General {
							}
							ps542 := PhiState{General: ps.General}
							ps542.OverlayValues = make([]JITValueDesc, 542)
							ps542.OverlayValues[8] = d8
							ps542.OverlayValues[9] = d9
							ps542.OverlayValues[10] = d10
							ps542.OverlayValues[11] = d11
							ps542.OverlayValues[12] = d12
							ps542.OverlayValues[13] = d13
							ps542.OverlayValues[14] = d14
							ps542.OverlayValues[15] = d15
							ps542.OverlayValues[16] = d16
							ps542.OverlayValues[17] = d17
							ps542.OverlayValues[18] = d18
							ps542.OverlayValues[19] = d19
							ps542.OverlayValues[20] = d20
							ps542.OverlayValues[21] = d21
							ps542.OverlayValues[22] = d22
							ps542.OverlayValues[24] = d24
							ps542.OverlayValues[26] = d26
							ps542.OverlayValues[27] = d27
							ps542.OverlayValues[30] = d30
							ps542.OverlayValues[51] = d51
							ps542.OverlayValues[52] = d52
							ps542.OverlayValues[53] = d53
							ps542.OverlayValues[54] = d54
							ps542.OverlayValues[55] = d55
							ps542.OverlayValues[57] = d57
							ps542.OverlayValues[58] = d58
							ps542.OverlayValues[59] = d59
							ps542.OverlayValues[60] = d60
							ps542.OverlayValues[61] = d61
							ps542.OverlayValues[62] = d62
							ps542.OverlayValues[63] = d63
							ps542.OverlayValues[66] = d66
							ps542.OverlayValues[103] = d103
							ps542.OverlayValues[104] = d104
							ps542.OverlayValues[105] = d105
							ps542.OverlayValues[106] = d106
							ps542.OverlayValues[107] = d107
							ps542.OverlayValues[108] = d108
							ps542.OverlayValues[110] = d110
							ps542.OverlayValues[111] = d111
							ps542.OverlayValues[112] = d112
							ps542.OverlayValues[113] = d113
							ps542.OverlayValues[114] = d114
							ps542.OverlayValues[115] = d115
							ps542.OverlayValues[116] = d116
							ps542.OverlayValues[117] = d117
							ps542.OverlayValues[118] = d118
							ps542.OverlayValues[121] = d121
							ps542.OverlayValues[122] = d122
							ps542.OverlayValues[123] = d123
							ps542.OverlayValues[124] = d124
							ps542.OverlayValues[179] = d179
							ps542.OverlayValues[180] = d180
							ps542.OverlayValues[181] = d181
							ps542.OverlayValues[182] = d182
							ps542.OverlayValues[183] = d183
							ps542.OverlayValues[184] = d184
							ps542.OverlayValues[185] = d185
							ps542.OverlayValues[186] = d186
							ps542.OverlayValues[187] = d187
							ps542.OverlayValues[188] = d188
							ps542.OverlayValues[189] = d189
							ps542.OverlayValues[190] = d190
							ps542.OverlayValues[191] = d191
							ps542.OverlayValues[192] = d192
							ps542.OverlayValues[193] = d193
							ps542.OverlayValues[194] = d194
							ps542.OverlayValues[195] = d195
							ps542.OverlayValues[196] = d196
							ps542.OverlayValues[197] = d197
							ps542.OverlayValues[199] = d199
							ps542.OverlayValues[200] = d200
							ps542.OverlayValues[201] = d201
							ps542.OverlayValues[202] = d202
							ps542.OverlayValues[203] = d203
							ps542.OverlayValues[204] = d204
							ps542.OverlayValues[205] = d205
							ps542.OverlayValues[206] = d206
							ps542.OverlayValues[208] = d208
							ps542.OverlayValues[209] = d209
							ps542.OverlayValues[210] = d210
							ps542.OverlayValues[297] = d297
							ps542.OverlayValues[298] = d298
							ps542.OverlayValues[299] = d299
							ps542.OverlayValues[300] = d300
							ps542.OverlayValues[301] = d301
							ps542.OverlayValues[304] = d304
							ps542.OverlayValues[305] = d305
							ps542.OverlayValues[397] = d397
							ps542.OverlayValues[398] = d398
							ps542.OverlayValues[399] = d399
							ps542.OverlayValues[400] = d400
							ps542.OverlayValues[401] = d401
							ps542.OverlayValues[402] = d402
							ps542.OverlayValues[403] = d403
							ps542.OverlayValues[404] = d404
							ps542.OverlayValues[405] = d405
							ps542.OverlayValues[406] = d406
							ps542.OverlayValues[407] = d407
							ps542.OverlayValues[408] = d408
							ps542.OverlayValues[409] = d409
							ps542.OverlayValues[411] = d411
							ps542.OverlayValues[412] = d412
							ps542.OverlayValues[413] = d413
							ps542.OverlayValues[414] = d414
							ps542.OverlayValues[415] = d415
							ps542.OverlayValues[416] = d416
							ps542.OverlayValues[417] = d417
							ps542.OverlayValues[419] = d419
							ps542.OverlayValues[421] = d421
							ps542.OverlayValues[422] = d422
							ps542.OverlayValues[425] = d425
							ps542.OverlayValues[539] = d539
							ps542.OverlayValues[540] = d540
							ps542.OverlayValues[541] = d541
							return bbs[11].RenderPS(ps542)
						}
						if ps.General {
						}
						ps543 := PhiState{General: ps.General}
						ps543.OverlayValues = make([]JITValueDesc, 542)
						ps543.OverlayValues[8] = d8
						ps543.OverlayValues[9] = d9
						ps543.OverlayValues[10] = d10
						ps543.OverlayValues[11] = d11
						ps543.OverlayValues[12] = d12
						ps543.OverlayValues[13] = d13
						ps543.OverlayValues[14] = d14
						ps543.OverlayValues[15] = d15
						ps543.OverlayValues[16] = d16
						ps543.OverlayValues[17] = d17
						ps543.OverlayValues[18] = d18
						ps543.OverlayValues[19] = d19
						ps543.OverlayValues[20] = d20
						ps543.OverlayValues[21] = d21
						ps543.OverlayValues[22] = d22
						ps543.OverlayValues[24] = d24
						ps543.OverlayValues[26] = d26
						ps543.OverlayValues[27] = d27
						ps543.OverlayValues[30] = d30
						ps543.OverlayValues[51] = d51
						ps543.OverlayValues[52] = d52
						ps543.OverlayValues[53] = d53
						ps543.OverlayValues[54] = d54
						ps543.OverlayValues[55] = d55
						ps543.OverlayValues[57] = d57
						ps543.OverlayValues[58] = d58
						ps543.OverlayValues[59] = d59
						ps543.OverlayValues[60] = d60
						ps543.OverlayValues[61] = d61
						ps543.OverlayValues[62] = d62
						ps543.OverlayValues[63] = d63
						ps543.OverlayValues[66] = d66
						ps543.OverlayValues[103] = d103
						ps543.OverlayValues[104] = d104
						ps543.OverlayValues[105] = d105
						ps543.OverlayValues[106] = d106
						ps543.OverlayValues[107] = d107
						ps543.OverlayValues[108] = d108
						ps543.OverlayValues[110] = d110
						ps543.OverlayValues[111] = d111
						ps543.OverlayValues[112] = d112
						ps543.OverlayValues[113] = d113
						ps543.OverlayValues[114] = d114
						ps543.OverlayValues[115] = d115
						ps543.OverlayValues[116] = d116
						ps543.OverlayValues[117] = d117
						ps543.OverlayValues[118] = d118
						ps543.OverlayValues[121] = d121
						ps543.OverlayValues[122] = d122
						ps543.OverlayValues[123] = d123
						ps543.OverlayValues[124] = d124
						ps543.OverlayValues[179] = d179
						ps543.OverlayValues[180] = d180
						ps543.OverlayValues[181] = d181
						ps543.OverlayValues[182] = d182
						ps543.OverlayValues[183] = d183
						ps543.OverlayValues[184] = d184
						ps543.OverlayValues[185] = d185
						ps543.OverlayValues[186] = d186
						ps543.OverlayValues[187] = d187
						ps543.OverlayValues[188] = d188
						ps543.OverlayValues[189] = d189
						ps543.OverlayValues[190] = d190
						ps543.OverlayValues[191] = d191
						ps543.OverlayValues[192] = d192
						ps543.OverlayValues[193] = d193
						ps543.OverlayValues[194] = d194
						ps543.OverlayValues[195] = d195
						ps543.OverlayValues[196] = d196
						ps543.OverlayValues[197] = d197
						ps543.OverlayValues[199] = d199
						ps543.OverlayValues[200] = d200
						ps543.OverlayValues[201] = d201
						ps543.OverlayValues[202] = d202
						ps543.OverlayValues[203] = d203
						ps543.OverlayValues[204] = d204
						ps543.OverlayValues[205] = d205
						ps543.OverlayValues[206] = d206
						ps543.OverlayValues[208] = d208
						ps543.OverlayValues[209] = d209
						ps543.OverlayValues[210] = d210
						ps543.OverlayValues[297] = d297
						ps543.OverlayValues[298] = d298
						ps543.OverlayValues[299] = d299
						ps543.OverlayValues[300] = d300
						ps543.OverlayValues[301] = d301
						ps543.OverlayValues[304] = d304
						ps543.OverlayValues[305] = d305
						ps543.OverlayValues[397] = d397
						ps543.OverlayValues[398] = d398
						ps543.OverlayValues[399] = d399
						ps543.OverlayValues[400] = d400
						ps543.OverlayValues[401] = d401
						ps543.OverlayValues[402] = d402
						ps543.OverlayValues[403] = d403
						ps543.OverlayValues[404] = d404
						ps543.OverlayValues[405] = d405
						ps543.OverlayValues[406] = d406
						ps543.OverlayValues[407] = d407
						ps543.OverlayValues[408] = d408
						ps543.OverlayValues[409] = d409
						ps543.OverlayValues[411] = d411
						ps543.OverlayValues[412] = d412
						ps543.OverlayValues[413] = d413
						ps543.OverlayValues[414] = d414
						ps543.OverlayValues[415] = d415
						ps543.OverlayValues[416] = d416
						ps543.OverlayValues[417] = d417
						ps543.OverlayValues[419] = d419
						ps543.OverlayValues[421] = d421
						ps543.OverlayValues[422] = d422
						ps543.OverlayValues[425] = d425
						ps543.OverlayValues[539] = d539
						ps543.OverlayValues[540] = d540
						ps543.OverlayValues[541] = d541
						return bbs[12].RenderPS(ps543)
					}
					if !ps.General {
						ps.General = true
						return bbs[13].RenderPS(ps)
					}
					lbl32 := ctx.ReserveLabel()
					lbl33 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d541.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl32)
					ctx.EmitJmp(lbl33)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl13)
					ps544 := PhiState{General: true}
					ps544.OverlayValues = make([]JITValueDesc, 542)
					ps544.OverlayValues[8] = d8
					ps544.OverlayValues[9] = d9
					ps544.OverlayValues[10] = d10
					ps544.OverlayValues[11] = d11
					ps544.OverlayValues[12] = d12
					ps544.OverlayValues[13] = d13
					ps544.OverlayValues[14] = d14
					ps544.OverlayValues[15] = d15
					ps544.OverlayValues[16] = d16
					ps544.OverlayValues[17] = d17
					ps544.OverlayValues[18] = d18
					ps544.OverlayValues[19] = d19
					ps544.OverlayValues[20] = d20
					ps544.OverlayValues[21] = d21
					ps544.OverlayValues[22] = d22
					ps544.OverlayValues[24] = d24
					ps544.OverlayValues[26] = d26
					ps544.OverlayValues[27] = d27
					ps544.OverlayValues[30] = d30
					ps544.OverlayValues[51] = d51
					ps544.OverlayValues[52] = d52
					ps544.OverlayValues[53] = d53
					ps544.OverlayValues[54] = d54
					ps544.OverlayValues[55] = d55
					ps544.OverlayValues[57] = d57
					ps544.OverlayValues[58] = d58
					ps544.OverlayValues[59] = d59
					ps544.OverlayValues[60] = d60
					ps544.OverlayValues[61] = d61
					ps544.OverlayValues[62] = d62
					ps544.OverlayValues[63] = d63
					ps544.OverlayValues[66] = d66
					ps544.OverlayValues[103] = d103
					ps544.OverlayValues[104] = d104
					ps544.OverlayValues[105] = d105
					ps544.OverlayValues[106] = d106
					ps544.OverlayValues[107] = d107
					ps544.OverlayValues[108] = d108
					ps544.OverlayValues[110] = d110
					ps544.OverlayValues[111] = d111
					ps544.OverlayValues[112] = d112
					ps544.OverlayValues[113] = d113
					ps544.OverlayValues[114] = d114
					ps544.OverlayValues[115] = d115
					ps544.OverlayValues[116] = d116
					ps544.OverlayValues[117] = d117
					ps544.OverlayValues[118] = d118
					ps544.OverlayValues[121] = d121
					ps544.OverlayValues[122] = d122
					ps544.OverlayValues[123] = d123
					ps544.OverlayValues[124] = d124
					ps544.OverlayValues[179] = d179
					ps544.OverlayValues[180] = d180
					ps544.OverlayValues[181] = d181
					ps544.OverlayValues[182] = d182
					ps544.OverlayValues[183] = d183
					ps544.OverlayValues[184] = d184
					ps544.OverlayValues[185] = d185
					ps544.OverlayValues[186] = d186
					ps544.OverlayValues[187] = d187
					ps544.OverlayValues[188] = d188
					ps544.OverlayValues[189] = d189
					ps544.OverlayValues[190] = d190
					ps544.OverlayValues[191] = d191
					ps544.OverlayValues[192] = d192
					ps544.OverlayValues[193] = d193
					ps544.OverlayValues[194] = d194
					ps544.OverlayValues[195] = d195
					ps544.OverlayValues[196] = d196
					ps544.OverlayValues[197] = d197
					ps544.OverlayValues[199] = d199
					ps544.OverlayValues[200] = d200
					ps544.OverlayValues[201] = d201
					ps544.OverlayValues[202] = d202
					ps544.OverlayValues[203] = d203
					ps544.OverlayValues[204] = d204
					ps544.OverlayValues[205] = d205
					ps544.OverlayValues[206] = d206
					ps544.OverlayValues[208] = d208
					ps544.OverlayValues[209] = d209
					ps544.OverlayValues[210] = d210
					ps544.OverlayValues[297] = d297
					ps544.OverlayValues[298] = d298
					ps544.OverlayValues[299] = d299
					ps544.OverlayValues[300] = d300
					ps544.OverlayValues[301] = d301
					ps544.OverlayValues[304] = d304
					ps544.OverlayValues[305] = d305
					ps544.OverlayValues[397] = d397
					ps544.OverlayValues[398] = d398
					ps544.OverlayValues[399] = d399
					ps544.OverlayValues[400] = d400
					ps544.OverlayValues[401] = d401
					ps544.OverlayValues[402] = d402
					ps544.OverlayValues[403] = d403
					ps544.OverlayValues[404] = d404
					ps544.OverlayValues[405] = d405
					ps544.OverlayValues[406] = d406
					ps544.OverlayValues[407] = d407
					ps544.OverlayValues[408] = d408
					ps544.OverlayValues[409] = d409
					ps544.OverlayValues[411] = d411
					ps544.OverlayValues[412] = d412
					ps544.OverlayValues[413] = d413
					ps544.OverlayValues[414] = d414
					ps544.OverlayValues[415] = d415
					ps544.OverlayValues[416] = d416
					ps544.OverlayValues[417] = d417
					ps544.OverlayValues[419] = d419
					ps544.OverlayValues[421] = d421
					ps544.OverlayValues[422] = d422
					ps544.OverlayValues[425] = d425
					ps544.OverlayValues[539] = d539
					ps544.OverlayValues[540] = d540
					ps544.OverlayValues[541] = d541
					ps545 := PhiState{General: true}
					ps545.OverlayValues = make([]JITValueDesc, 542)
					ps545.OverlayValues[8] = d8
					ps545.OverlayValues[9] = d9
					ps545.OverlayValues[10] = d10
					ps545.OverlayValues[11] = d11
					ps545.OverlayValues[12] = d12
					ps545.OverlayValues[13] = d13
					ps545.OverlayValues[14] = d14
					ps545.OverlayValues[15] = d15
					ps545.OverlayValues[16] = d16
					ps545.OverlayValues[17] = d17
					ps545.OverlayValues[18] = d18
					ps545.OverlayValues[19] = d19
					ps545.OverlayValues[20] = d20
					ps545.OverlayValues[21] = d21
					ps545.OverlayValues[22] = d22
					ps545.OverlayValues[24] = d24
					ps545.OverlayValues[26] = d26
					ps545.OverlayValues[27] = d27
					ps545.OverlayValues[30] = d30
					ps545.OverlayValues[51] = d51
					ps545.OverlayValues[52] = d52
					ps545.OverlayValues[53] = d53
					ps545.OverlayValues[54] = d54
					ps545.OverlayValues[55] = d55
					ps545.OverlayValues[57] = d57
					ps545.OverlayValues[58] = d58
					ps545.OverlayValues[59] = d59
					ps545.OverlayValues[60] = d60
					ps545.OverlayValues[61] = d61
					ps545.OverlayValues[62] = d62
					ps545.OverlayValues[63] = d63
					ps545.OverlayValues[66] = d66
					ps545.OverlayValues[103] = d103
					ps545.OverlayValues[104] = d104
					ps545.OverlayValues[105] = d105
					ps545.OverlayValues[106] = d106
					ps545.OverlayValues[107] = d107
					ps545.OverlayValues[108] = d108
					ps545.OverlayValues[110] = d110
					ps545.OverlayValues[111] = d111
					ps545.OverlayValues[112] = d112
					ps545.OverlayValues[113] = d113
					ps545.OverlayValues[114] = d114
					ps545.OverlayValues[115] = d115
					ps545.OverlayValues[116] = d116
					ps545.OverlayValues[117] = d117
					ps545.OverlayValues[118] = d118
					ps545.OverlayValues[121] = d121
					ps545.OverlayValues[122] = d122
					ps545.OverlayValues[123] = d123
					ps545.OverlayValues[124] = d124
					ps545.OverlayValues[179] = d179
					ps545.OverlayValues[180] = d180
					ps545.OverlayValues[181] = d181
					ps545.OverlayValues[182] = d182
					ps545.OverlayValues[183] = d183
					ps545.OverlayValues[184] = d184
					ps545.OverlayValues[185] = d185
					ps545.OverlayValues[186] = d186
					ps545.OverlayValues[187] = d187
					ps545.OverlayValues[188] = d188
					ps545.OverlayValues[189] = d189
					ps545.OverlayValues[190] = d190
					ps545.OverlayValues[191] = d191
					ps545.OverlayValues[192] = d192
					ps545.OverlayValues[193] = d193
					ps545.OverlayValues[194] = d194
					ps545.OverlayValues[195] = d195
					ps545.OverlayValues[196] = d196
					ps545.OverlayValues[197] = d197
					ps545.OverlayValues[199] = d199
					ps545.OverlayValues[200] = d200
					ps545.OverlayValues[201] = d201
					ps545.OverlayValues[202] = d202
					ps545.OverlayValues[203] = d203
					ps545.OverlayValues[204] = d204
					ps545.OverlayValues[205] = d205
					ps545.OverlayValues[206] = d206
					ps545.OverlayValues[208] = d208
					ps545.OverlayValues[209] = d209
					ps545.OverlayValues[210] = d210
					ps545.OverlayValues[297] = d297
					ps545.OverlayValues[298] = d298
					ps545.OverlayValues[299] = d299
					ps545.OverlayValues[300] = d300
					ps545.OverlayValues[301] = d301
					ps545.OverlayValues[304] = d304
					ps545.OverlayValues[305] = d305
					ps545.OverlayValues[397] = d397
					ps545.OverlayValues[398] = d398
					ps545.OverlayValues[399] = d399
					ps545.OverlayValues[400] = d400
					ps545.OverlayValues[401] = d401
					ps545.OverlayValues[402] = d402
					ps545.OverlayValues[403] = d403
					ps545.OverlayValues[404] = d404
					ps545.OverlayValues[405] = d405
					ps545.OverlayValues[406] = d406
					ps545.OverlayValues[407] = d407
					ps545.OverlayValues[408] = d408
					ps545.OverlayValues[409] = d409
					ps545.OverlayValues[411] = d411
					ps545.OverlayValues[412] = d412
					ps545.OverlayValues[413] = d413
					ps545.OverlayValues[414] = d414
					ps545.OverlayValues[415] = d415
					ps545.OverlayValues[416] = d416
					ps545.OverlayValues[417] = d417
					ps545.OverlayValues[419] = d419
					ps545.OverlayValues[421] = d421
					ps545.OverlayValues[422] = d422
					ps545.OverlayValues[425] = d425
					ps545.OverlayValues[539] = d539
					ps545.OverlayValues[540] = d540
					ps545.OverlayValues[541] = d541
					snap546 := d8
					snap547 := d9
					snap548 := d10
					snap549 := d11
					snap550 := d12
					snap551 := d13
					snap552 := d14
					snap553 := d15
					snap554 := d16
					snap555 := d17
					snap556 := d18
					snap557 := d19
					snap558 := d20
					snap559 := d21
					snap560 := d22
					snap561 := d24
					snap562 := d26
					snap563 := d27
					snap564 := d30
					snap565 := d51
					snap566 := d52
					snap567 := d53
					snap568 := d54
					snap569 := d55
					snap570 := d57
					snap571 := d58
					snap572 := d59
					snap573 := d60
					snap574 := d61
					snap575 := d62
					snap576 := d63
					snap577 := d66
					snap578 := d103
					snap579 := d104
					snap580 := d105
					snap581 := d106
					snap582 := d107
					snap583 := d108
					snap584 := d110
					snap585 := d111
					snap586 := d112
					snap587 := d113
					snap588 := d114
					snap589 := d115
					snap590 := d116
					snap591 := d117
					snap592 := d118
					snap593 := d121
					snap594 := d122
					snap595 := d123
					snap596 := d124
					snap597 := d179
					snap598 := d180
					snap599 := d181
					snap600 := d182
					snap601 := d183
					snap602 := d184
					snap603 := d185
					snap604 := d186
					snap605 := d187
					snap606 := d188
					snap607 := d189
					snap608 := d190
					snap609 := d191
					snap610 := d192
					snap611 := d193
					snap612 := d194
					snap613 := d195
					snap614 := d196
					snap615 := d197
					snap616 := d199
					snap617 := d200
					snap618 := d201
					snap619 := d202
					snap620 := d203
					snap621 := d204
					snap622 := d205
					snap623 := d206
					snap624 := d208
					snap625 := d209
					snap626 := d210
					snap627 := d297
					snap628 := d298
					snap629 := d299
					snap630 := d300
					snap631 := d301
					snap632 := d304
					snap633 := d305
					snap634 := d397
					snap635 := d398
					snap636 := d399
					snap637 := d400
					snap638 := d401
					snap639 := d402
					snap640 := d403
					snap641 := d404
					snap642 := d405
					snap643 := d406
					snap644 := d407
					snap645 := d408
					snap646 := d409
					snap647 := d411
					snap648 := d412
					snap649 := d413
					snap650 := d414
					snap651 := d415
					snap652 := d416
					snap653 := d417
					snap654 := d419
					snap655 := d421
					snap656 := d422
					snap657 := d425
					snap658 := d539
					snap659 := d540
					snap660 := d541
					alloc661 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps545)
					}
					ctx.RestoreAllocState(alloc661)
					d8 = snap546
					d9 = snap547
					d10 = snap548
					d11 = snap549
					d12 = snap550
					d13 = snap551
					d14 = snap552
					d15 = snap553
					d16 = snap554
					d17 = snap555
					d18 = snap556
					d19 = snap557
					d20 = snap558
					d21 = snap559
					d22 = snap560
					d24 = snap561
					d26 = snap562
					d27 = snap563
					d30 = snap564
					d51 = snap565
					d52 = snap566
					d53 = snap567
					d54 = snap568
					d55 = snap569
					d57 = snap570
					d58 = snap571
					d59 = snap572
					d60 = snap573
					d61 = snap574
					d62 = snap575
					d63 = snap576
					d66 = snap577
					d103 = snap578
					d104 = snap579
					d105 = snap580
					d106 = snap581
					d107 = snap582
					d108 = snap583
					d110 = snap584
					d111 = snap585
					d112 = snap586
					d113 = snap587
					d114 = snap588
					d115 = snap589
					d116 = snap590
					d117 = snap591
					d118 = snap592
					d121 = snap593
					d122 = snap594
					d123 = snap595
					d124 = snap596
					d179 = snap597
					d180 = snap598
					d181 = snap599
					d182 = snap600
					d183 = snap601
					d184 = snap602
					d185 = snap603
					d186 = snap604
					d187 = snap605
					d188 = snap606
					d189 = snap607
					d190 = snap608
					d191 = snap609
					d192 = snap610
					d193 = snap611
					d194 = snap612
					d195 = snap613
					d196 = snap614
					d197 = snap615
					d199 = snap616
					d200 = snap617
					d201 = snap618
					d202 = snap619
					d203 = snap620
					d204 = snap621
					d205 = snap622
					d206 = snap623
					d208 = snap624
					d209 = snap625
					d210 = snap626
					d297 = snap627
					d298 = snap628
					d299 = snap629
					d300 = snap630
					d301 = snap631
					d304 = snap632
					d305 = snap633
					d397 = snap634
					d398 = snap635
					d399 = snap636
					d400 = snap637
					d401 = snap638
					d402 = snap639
					d403 = snap640
					d404 = snap641
					d405 = snap642
					d406 = snap643
					d407 = snap644
					d408 = snap645
					d409 = snap646
					d411 = snap647
					d412 = snap648
					d413 = snap649
					d414 = snap650
					d415 = snap651
					d416 = snap652
					d417 = snap653
					d419 = snap654
					d421 = snap655
					d422 = snap656
					d425 = snap657
					d539 = snap658
					d540 = snap659
					d541 = snap660
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps544)
					}
					return result
					ctx.FreeDesc(&d540)
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
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0}
						ctx.BindReg(r0, &d10)
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1}
						ctx.BindReg(r1, &d11)
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
						ctx.BindReg(r2, &d12)
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d13)
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4}
						ctx.BindReg(r4, &d14)
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d15)
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if !ps.General && len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
						d14 = ps.OverlayValues[14]
					}
					if !ps.General && len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					if len(ps.OverlayValues) > 108 && ps.OverlayValues[108].Loc != LocNone {
						d108 = ps.OverlayValues[108]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != LocNone {
						d179 = ps.OverlayValues[179]
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
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
						d188 = ps.OverlayValues[188]
					}
					if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
						d189 = ps.OverlayValues[189]
					}
					if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
						d190 = ps.OverlayValues[190]
					}
					if len(ps.OverlayValues) > 191 && ps.OverlayValues[191].Loc != LocNone {
						d191 = ps.OverlayValues[191]
					}
					if len(ps.OverlayValues) > 192 && ps.OverlayValues[192].Loc != LocNone {
						d192 = ps.OverlayValues[192]
					}
					if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
						d193 = ps.OverlayValues[193]
					}
					if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
						d194 = ps.OverlayValues[194]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 200 && ps.OverlayValues[200].Loc != LocNone {
						d200 = ps.OverlayValues[200]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
						d210 = ps.OverlayValues[210]
					}
					if len(ps.OverlayValues) > 297 && ps.OverlayValues[297].Loc != LocNone {
						d297 = ps.OverlayValues[297]
					}
					if len(ps.OverlayValues) > 298 && ps.OverlayValues[298].Loc != LocNone {
						d298 = ps.OverlayValues[298]
					}
					if len(ps.OverlayValues) > 299 && ps.OverlayValues[299].Loc != LocNone {
						d299 = ps.OverlayValues[299]
					}
					if len(ps.OverlayValues) > 300 && ps.OverlayValues[300].Loc != LocNone {
						d300 = ps.OverlayValues[300]
					}
					if len(ps.OverlayValues) > 301 && ps.OverlayValues[301].Loc != LocNone {
						d301 = ps.OverlayValues[301]
					}
					if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != LocNone {
						d304 = ps.OverlayValues[304]
					}
					if len(ps.OverlayValues) > 305 && ps.OverlayValues[305].Loc != LocNone {
						d305 = ps.OverlayValues[305]
					}
					if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != LocNone {
						d397 = ps.OverlayValues[397]
					}
					if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != LocNone {
						d398 = ps.OverlayValues[398]
					}
					if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != LocNone {
						d399 = ps.OverlayValues[399]
					}
					if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != LocNone {
						d400 = ps.OverlayValues[400]
					}
					if len(ps.OverlayValues) > 401 && ps.OverlayValues[401].Loc != LocNone {
						d401 = ps.OverlayValues[401]
					}
					if len(ps.OverlayValues) > 402 && ps.OverlayValues[402].Loc != LocNone {
						d402 = ps.OverlayValues[402]
					}
					if len(ps.OverlayValues) > 403 && ps.OverlayValues[403].Loc != LocNone {
						d403 = ps.OverlayValues[403]
					}
					if len(ps.OverlayValues) > 404 && ps.OverlayValues[404].Loc != LocNone {
						d404 = ps.OverlayValues[404]
					}
					if len(ps.OverlayValues) > 405 && ps.OverlayValues[405].Loc != LocNone {
						d405 = ps.OverlayValues[405]
					}
					if len(ps.OverlayValues) > 406 && ps.OverlayValues[406].Loc != LocNone {
						d406 = ps.OverlayValues[406]
					}
					if len(ps.OverlayValues) > 407 && ps.OverlayValues[407].Loc != LocNone {
						d407 = ps.OverlayValues[407]
					}
					if len(ps.OverlayValues) > 408 && ps.OverlayValues[408].Loc != LocNone {
						d408 = ps.OverlayValues[408]
					}
					if len(ps.OverlayValues) > 409 && ps.OverlayValues[409].Loc != LocNone {
						d409 = ps.OverlayValues[409]
					}
					if len(ps.OverlayValues) > 411 && ps.OverlayValues[411].Loc != LocNone {
						d411 = ps.OverlayValues[411]
					}
					if len(ps.OverlayValues) > 412 && ps.OverlayValues[412].Loc != LocNone {
						d412 = ps.OverlayValues[412]
					}
					if len(ps.OverlayValues) > 413 && ps.OverlayValues[413].Loc != LocNone {
						d413 = ps.OverlayValues[413]
					}
					if len(ps.OverlayValues) > 414 && ps.OverlayValues[414].Loc != LocNone {
						d414 = ps.OverlayValues[414]
					}
					if len(ps.OverlayValues) > 415 && ps.OverlayValues[415].Loc != LocNone {
						d415 = ps.OverlayValues[415]
					}
					if len(ps.OverlayValues) > 416 && ps.OverlayValues[416].Loc != LocNone {
						d416 = ps.OverlayValues[416]
					}
					if len(ps.OverlayValues) > 417 && ps.OverlayValues[417].Loc != LocNone {
						d417 = ps.OverlayValues[417]
					}
					if len(ps.OverlayValues) > 419 && ps.OverlayValues[419].Loc != LocNone {
						d419 = ps.OverlayValues[419]
					}
					if len(ps.OverlayValues) > 421 && ps.OverlayValues[421].Loc != LocNone {
						d421 = ps.OverlayValues[421]
					}
					if len(ps.OverlayValues) > 422 && ps.OverlayValues[422].Loc != LocNone {
						d422 = ps.OverlayValues[422]
					}
					if len(ps.OverlayValues) > 425 && ps.OverlayValues[425].Loc != LocNone {
						d425 = ps.OverlayValues[425]
					}
					if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != LocNone {
						d539 = ps.OverlayValues[539]
					}
					if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != LocNone {
						d540 = ps.OverlayValues[540]
					}
					if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != LocNone {
						d541 = ps.OverlayValues[541]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d14)
					var d662 JITValueDesc
					if d14.Loc == LocImm {
						d662 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d14.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d14)
						var d663 JITValueDesc
						if d14.Loc == LocRegPair {
							ctx.FreeReg(d14.Reg)
							d663 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg2}
							ctx.BindReg(d14.Reg2, &d663)
							ctx.BindReg(d14.Reg2, &d663)
						} else {
							d663 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg}
							ctx.BindReg(d14.Reg, &d663)
							ctx.BindReg(d14.Reg, &d663)
						}
						d662 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d663}, 1)
						d662.Type = tagFloat
						ctx.BindReg(d662.Reg, &d662)
					}
					ctx.StabilizeDescForControlFlow(&d662)
					if ps.General {
						ctx.SyncDesc(&d662)
						if d662.Loc == LocReg {
							ctx.ProtectReg(d662.Reg)
						} else if d662.Loc == LocRegPair {
							ctx.ProtectReg(d662.Reg)
							ctx.ProtectReg(d662.Reg2)
						}
						d664 = d662
						if d664.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d664)
						ctx.EmitStoreToStack(d664, int32(bbs[4].PhiBase)+int32(0))
						if d662.Loc == LocReg {
							ctx.UnprotectReg(d662.Reg)
						} else if d662.Loc == LocRegPair {
							ctx.UnprotectReg(d662.Reg)
							ctx.UnprotectReg(d662.Reg2)
						}
					}
					ps665 := PhiState{General: ps.General}
					ps665.OverlayValues = make([]JITValueDesc, 665)
					ps665.OverlayValues[8] = d8
					ps665.OverlayValues[9] = d9
					ps665.OverlayValues[10] = d10
					ps665.OverlayValues[11] = d11
					ps665.OverlayValues[12] = d12
					ps665.OverlayValues[13] = d13
					ps665.OverlayValues[14] = d14
					ps665.OverlayValues[15] = d15
					ps665.OverlayValues[16] = d16
					ps665.OverlayValues[17] = d17
					ps665.OverlayValues[18] = d18
					ps665.OverlayValues[19] = d19
					ps665.OverlayValues[20] = d20
					ps665.OverlayValues[21] = d21
					ps665.OverlayValues[22] = d22
					ps665.OverlayValues[24] = d24
					ps665.OverlayValues[26] = d26
					ps665.OverlayValues[27] = d27
					ps665.OverlayValues[30] = d30
					ps665.OverlayValues[51] = d51
					ps665.OverlayValues[52] = d52
					ps665.OverlayValues[53] = d53
					ps665.OverlayValues[54] = d54
					ps665.OverlayValues[55] = d55
					ps665.OverlayValues[57] = d57
					ps665.OverlayValues[58] = d58
					ps665.OverlayValues[59] = d59
					ps665.OverlayValues[60] = d60
					ps665.OverlayValues[61] = d61
					ps665.OverlayValues[62] = d62
					ps665.OverlayValues[63] = d63
					ps665.OverlayValues[66] = d66
					ps665.OverlayValues[103] = d103
					ps665.OverlayValues[104] = d104
					ps665.OverlayValues[105] = d105
					ps665.OverlayValues[106] = d106
					ps665.OverlayValues[107] = d107
					ps665.OverlayValues[108] = d108
					ps665.OverlayValues[110] = d110
					ps665.OverlayValues[111] = d111
					ps665.OverlayValues[112] = d112
					ps665.OverlayValues[113] = d113
					ps665.OverlayValues[114] = d114
					ps665.OverlayValues[115] = d115
					ps665.OverlayValues[116] = d116
					ps665.OverlayValues[117] = d117
					ps665.OverlayValues[118] = d118
					ps665.OverlayValues[121] = d121
					ps665.OverlayValues[122] = d122
					ps665.OverlayValues[123] = d123
					ps665.OverlayValues[124] = d124
					ps665.OverlayValues[179] = d179
					ps665.OverlayValues[180] = d180
					ps665.OverlayValues[181] = d181
					ps665.OverlayValues[182] = d182
					ps665.OverlayValues[183] = d183
					ps665.OverlayValues[184] = d184
					ps665.OverlayValues[185] = d185
					ps665.OverlayValues[186] = d186
					ps665.OverlayValues[187] = d187
					ps665.OverlayValues[188] = d188
					ps665.OverlayValues[189] = d189
					ps665.OverlayValues[190] = d190
					ps665.OverlayValues[191] = d191
					ps665.OverlayValues[192] = d192
					ps665.OverlayValues[193] = d193
					ps665.OverlayValues[194] = d194
					ps665.OverlayValues[195] = d195
					ps665.OverlayValues[196] = d196
					ps665.OverlayValues[197] = d197
					ps665.OverlayValues[199] = d199
					ps665.OverlayValues[200] = d200
					ps665.OverlayValues[201] = d201
					ps665.OverlayValues[202] = d202
					ps665.OverlayValues[203] = d203
					ps665.OverlayValues[204] = d204
					ps665.OverlayValues[205] = d205
					ps665.OverlayValues[206] = d206
					ps665.OverlayValues[208] = d208
					ps665.OverlayValues[209] = d209
					ps665.OverlayValues[210] = d210
					ps665.OverlayValues[297] = d297
					ps665.OverlayValues[298] = d298
					ps665.OverlayValues[299] = d299
					ps665.OverlayValues[300] = d300
					ps665.OverlayValues[301] = d301
					ps665.OverlayValues[304] = d304
					ps665.OverlayValues[305] = d305
					ps665.OverlayValues[397] = d397
					ps665.OverlayValues[398] = d398
					ps665.OverlayValues[399] = d399
					ps665.OverlayValues[400] = d400
					ps665.OverlayValues[401] = d401
					ps665.OverlayValues[402] = d402
					ps665.OverlayValues[403] = d403
					ps665.OverlayValues[404] = d404
					ps665.OverlayValues[405] = d405
					ps665.OverlayValues[406] = d406
					ps665.OverlayValues[407] = d407
					ps665.OverlayValues[408] = d408
					ps665.OverlayValues[409] = d409
					ps665.OverlayValues[411] = d411
					ps665.OverlayValues[412] = d412
					ps665.OverlayValues[413] = d413
					ps665.OverlayValues[414] = d414
					ps665.OverlayValues[415] = d415
					ps665.OverlayValues[416] = d416
					ps665.OverlayValues[417] = d417
					ps665.OverlayValues[419] = d419
					ps665.OverlayValues[421] = d421
					ps665.OverlayValues[422] = d422
					ps665.OverlayValues[425] = d425
					ps665.OverlayValues[539] = d539
					ps665.OverlayValues[540] = d540
					ps665.OverlayValues[541] = d541
					ps665.OverlayValues[662] = d662
					ps665.OverlayValues[663] = d663
					ps665.OverlayValues[664] = d664
					ps665.PhiValues = make([]JITValueDesc, 1)
					d666 = d662
					ps665.PhiValues[0] = d666
					if ps665.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps665)
					return result
				}
				ps667 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps667)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  80,
		},
	})
}
