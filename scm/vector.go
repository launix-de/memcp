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
				var d25 JITValueDesc
				_ = d25
				var d28 JITValueDesc
				_ = d28
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
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d62 JITValueDesc
				_ = d62
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
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
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
				var d198 JITValueDesc
				_ = d198
				var d200 JITValueDesc
				_ = d200
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d287 JITValueDesc
				_ = d287
				var d288 JITValueDesc
				_ = d288
				var d289 JITValueDesc
				_ = d289
				var d290 JITValueDesc
				_ = d290
				var d291 JITValueDesc
				_ = d291
				var d294 JITValueDesc
				_ = d294
				var d295 JITValueDesc
				_ = d295
				var d385 JITValueDesc
				_ = d385
				var d386 JITValueDesc
				_ = d386
				var d387 JITValueDesc
				_ = d387
				var d388 JITValueDesc
				_ = d388
				var d389 JITValueDesc
				_ = d389
				var d390 JITValueDesc
				_ = d390
				var d391 JITValueDesc
				_ = d391
				var d392 JITValueDesc
				_ = d392
				var d393 JITValueDesc
				_ = d393
				var d394 JITValueDesc
				_ = d394
				var d395 JITValueDesc
				_ = d395
				var d396 JITValueDesc
				_ = d396
				var d397 JITValueDesc
				_ = d397
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
				var d407 JITValueDesc
				_ = d407
				var d409 JITValueDesc
				_ = d409
				var d410 JITValueDesc
				_ = d410
				var d413 JITValueDesc
				_ = d413
				var d525 JITValueDesc
				_ = d525
				var d526 JITValueDesc
				_ = d526
				var d527 JITValueDesc
				_ = d527
				var d646 JITValueDesc
				_ = d646
				var d647 JITValueDesc
				_ = d647
				var d648 JITValueDesc
				_ = d648
				var d650 JITValueDesc
				_ = d650
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
				phiHomeOK3 := registerHomes1.Available&(uint16(1)<<2) == uint16(1)<<2
				if phiHomeOK3 {
					r1 = registerHomes1.Registers[2]
				}
				var r2 Reg
				phiHomeOK4 := registerHomes1.Available&(uint16(1)<<3) == uint16(1)<<3
				if phiHomeOK4 {
					r2 = registerHomes1.Registers[3]
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
					d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
				} else {
					d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
				}
				_ = d10
				var d11 JITValueDesc
				if phiHomeOK3 {
					d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
				} else {
					d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
				}
				_ = d11
				var d12 JITValueDesc
				if phiHomeOK4 {
					d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
				} else {
					d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
				}
				_ = d12
				var d13 JITValueDesc
				if phiHomeOK5 {
					d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
				} else {
					d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
				}
				_ = d13
				var d14 JITValueDesc
				if phiHomeOK6 {
					d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
				} else {
					d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
				}
				_ = d14
				var d15 JITValueDesc
				if phiHomeOK7 {
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
							ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
						}
						ps24 := PhiState{General: ps.General}
						ps24.OverlayValues = make([]JITValueDesc, 23)
						ps24.OverlayValues[8] = d8
						ps24.OverlayValues[9] = d9
						ps24.OverlayValues[10] = d10
						ps24.OverlayValues[11] = d11
						ps24.OverlayValues[12] = d12
						ps24.OverlayValues[13] = d13
						ps24.OverlayValues[14] = d14
						ps24.OverlayValues[15] = d15
						ps24.OverlayValues[16] = d16
						ps24.OverlayValues[17] = d17
						ps24.OverlayValues[18] = d18
						ps24.OverlayValues[19] = d19
						ps24.OverlayValues[20] = d20
						ps24.OverlayValues[21] = d21
						ps24.OverlayValues[22] = d22
						ps24.PhiValues = make([]JITValueDesc, 1)
						d25 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
						ps24.PhiValues[0] = d25
						return bbs[2].RenderPS(ps24)
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
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 26)
					ps26.OverlayValues[8] = d8
					ps26.OverlayValues[9] = d9
					ps26.OverlayValues[10] = d10
					ps26.OverlayValues[11] = d11
					ps26.OverlayValues[12] = d12
					ps26.OverlayValues[13] = d13
					ps26.OverlayValues[14] = d14
					ps26.OverlayValues[15] = d15
					ps26.OverlayValues[16] = d16
					ps26.OverlayValues[17] = d17
					ps26.OverlayValues[18] = d18
					ps26.OverlayValues[19] = d19
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[22] = d22
					ps26.OverlayValues[25] = d25
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 26)
					ps27.OverlayValues[8] = d8
					ps27.OverlayValues[9] = d9
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[11] = d11
					ps27.OverlayValues[12] = d12
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[14] = d14
					ps27.OverlayValues[15] = d15
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[17] = d17
					ps27.OverlayValues[18] = d18
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[25] = d25
					ps27.PhiValues = make([]JITValueDesc, 1)
					d28 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
					ps27.PhiValues[0] = d28
					snap29 := d8
					snap30 := d9
					snap31 := d10
					snap32 := d11
					snap33 := d12
					snap34 := d13
					snap35 := d14
					snap36 := d15
					snap37 := d16
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d21
					snap43 := d22
					snap44 := d25
					snap45 := d28
					alloc46 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps27)
					}
					ctx.RestoreAllocState(alloc46)
					d8 = snap29
					d9 = snap30
					d10 = snap31
					d11 = snap32
					d12 = snap33
					d13 = snap34
					d14 = snap35
					d15 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d22 = snap43
					d25 = snap44
					d28 = snap45
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps26)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					ctx.ReclaimUntrackedRegs()
					d47 = args[2]
					d47.ID = 0
					d49 = d47
					ctx.SyncDesc(&d49)
					if d49.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d49.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d49.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d49 = tmpScalar
					}
					d49 = JITPrepareScmerGoArg(ctx, d49)
					if d49.Loc != LocRegPair && d49.Loc != LocStackPair && d49.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d48 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d49}, 2)
					ctx.FreeDesc(&d47)
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					if d48.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d48.Imm)
						ptrWord, _ := d48.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d48.Imm.String())))
						d48 = tmpPair
					} else if d48.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d48.Type, Reg: ctx.AllocRegExcept(d48.Reg), Reg2: ctx.AllocRegExcept(d48.Reg)}
						switch d48.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d48)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d48)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d48)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d48)
						d48 = tmpPair
					}
					if d48.Loc != LocRegPair && d48.Loc != LocStackPair && d48.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d48)
					d50 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d48}, 2)
					d50.NoHeapPointer = false
					ctx.BindReg(d50.Reg, &d50)
					ctx.BindReg(d50.Reg2, &d50)
					ctx.StabilizeDescForControlFlow(&d50)
					if ps.General {
						ctx.SyncDesc(&d50)
						if d50.Loc == LocReg {
							ctx.ProtectReg(d50.Reg)
						} else if d50.Loc == LocRegPair {
							ctx.ProtectReg(d50.Reg)
							ctx.ProtectReg(d50.Reg2)
						}
						d51 = d50
						if d51.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d51)
						ctx.EmitStoreScmerToStack(d51, int32(bbs[2].PhiBase)+int32(0))
						if d50.Loc == LocReg {
							ctx.UnprotectReg(d50.Reg)
						} else if d50.Loc == LocRegPair {
							ctx.UnprotectReg(d50.Reg)
							ctx.UnprotectReg(d50.Reg2)
						}
					}
					ps52 := PhiState{General: ps.General}
					ps52.OverlayValues = make([]JITValueDesc, 52)
					ps52.OverlayValues[8] = d8
					ps52.OverlayValues[9] = d9
					ps52.OverlayValues[10] = d10
					ps52.OverlayValues[11] = d11
					ps52.OverlayValues[12] = d12
					ps52.OverlayValues[13] = d13
					ps52.OverlayValues[14] = d14
					ps52.OverlayValues[15] = d15
					ps52.OverlayValues[16] = d16
					ps52.OverlayValues[17] = d17
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[19] = d19
					ps52.OverlayValues[20] = d20
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[22] = d22
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[28] = d28
					ps52.OverlayValues[47] = d47
					ps52.OverlayValues[48] = d48
					ps52.OverlayValues[49] = d49
					ps52.OverlayValues[50] = d50
					ps52.OverlayValues[51] = d51
					ps52.PhiValues = make([]JITValueDesc, 1)
					d53 = d50
					ps52.PhiValues[0] = d53
					if ps52.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps52)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d54 := ps.PhiValues[0]
							ctx.EnsureDesc(&d54)
							ctx.EmitStoreScmerToStack(d54, int32(bbs[2].PhiBase)+int32(0))
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d8 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d8)
					var d55 JITValueDesc
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d55 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d55.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d55.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d55.Reg, &d55)
						ctx.BindReg(d55.Reg2, &d55)
					} else {
						d55 = d8
					}
					d56 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("COSINE")}
					var d57 JITValueDesc
					if d56.Loc == LocImm {
						ctx.TrackImm(d56.Imm)
						ptrWord, _ := d56.Imm.RawWords()
						d57 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d57.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d57.Reg2, uint64(len(d56.Imm.String())))
						ctx.BindReg(d57.Reg, &d57)
						ctx.BindReg(d57.Reg2, &d57)
					} else {
						d57 = d56
					}
					d58 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d55, d57}, 1)
					ctx.EmitAndRegImm32(d58.Reg, 1)
					d58.Type = tagBool
					ctx.BindReg(d58.Reg, &d58)
					d59 = d58
					ctx.EnsureDesc(&d59)
					if d59.Loc != LocImm && d59.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d59.Loc == LocImm {
						if d59.Imm.Bool() {
							if ps.General {
							}
							ps60 := PhiState{General: ps.General}
							ps60.OverlayValues = make([]JITValueDesc, 60)
							ps60.OverlayValues[8] = d8
							ps60.OverlayValues[9] = d9
							ps60.OverlayValues[10] = d10
							ps60.OverlayValues[11] = d11
							ps60.OverlayValues[12] = d12
							ps60.OverlayValues[13] = d13
							ps60.OverlayValues[14] = d14
							ps60.OverlayValues[15] = d15
							ps60.OverlayValues[16] = d16
							ps60.OverlayValues[17] = d17
							ps60.OverlayValues[18] = d18
							ps60.OverlayValues[19] = d19
							ps60.OverlayValues[20] = d20
							ps60.OverlayValues[21] = d21
							ps60.OverlayValues[22] = d22
							ps60.OverlayValues[25] = d25
							ps60.OverlayValues[28] = d28
							ps60.OverlayValues[47] = d47
							ps60.OverlayValues[48] = d48
							ps60.OverlayValues[49] = d49
							ps60.OverlayValues[50] = d50
							ps60.OverlayValues[51] = d51
							ps60.OverlayValues[53] = d53
							ps60.OverlayValues[54] = d54
							ps60.OverlayValues[55] = d55
							ps60.OverlayValues[56] = d56
							ps60.OverlayValues[57] = d57
							ps60.OverlayValues[58] = d58
							ps60.OverlayValues[59] = d59
							return bbs[3].RenderPS(ps60)
						}
						if ps.General {
						}
						ps61 := PhiState{General: ps.General}
						ps61.OverlayValues = make([]JITValueDesc, 60)
						ps61.OverlayValues[8] = d8
						ps61.OverlayValues[9] = d9
						ps61.OverlayValues[10] = d10
						ps61.OverlayValues[11] = d11
						ps61.OverlayValues[12] = d12
						ps61.OverlayValues[13] = d13
						ps61.OverlayValues[14] = d14
						ps61.OverlayValues[15] = d15
						ps61.OverlayValues[16] = d16
						ps61.OverlayValues[17] = d17
						ps61.OverlayValues[18] = d18
						ps61.OverlayValues[19] = d19
						ps61.OverlayValues[20] = d20
						ps61.OverlayValues[21] = d21
						ps61.OverlayValues[22] = d22
						ps61.OverlayValues[25] = d25
						ps61.OverlayValues[28] = d28
						ps61.OverlayValues[47] = d47
						ps61.OverlayValues[48] = d48
						ps61.OverlayValues[49] = d49
						ps61.OverlayValues[50] = d50
						ps61.OverlayValues[51] = d51
						ps61.OverlayValues[53] = d53
						ps61.OverlayValues[54] = d54
						ps61.OverlayValues[55] = d55
						ps61.OverlayValues[56] = d56
						ps61.OverlayValues[57] = d57
						ps61.OverlayValues[58] = d58
						ps61.OverlayValues[59] = d59
						return bbs[5].RenderPS(ps61)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d62 := ps.PhiValues[0]
							ctx.EnsureDesc(&d62)
							ctx.EmitStoreScmerToStack(d62, int32(bbs[2].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d59.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl6)
					ps63 := PhiState{General: true}
					ps63.OverlayValues = make([]JITValueDesc, 63)
					ps63.OverlayValues[8] = d8
					ps63.OverlayValues[9] = d9
					ps63.OverlayValues[10] = d10
					ps63.OverlayValues[11] = d11
					ps63.OverlayValues[12] = d12
					ps63.OverlayValues[13] = d13
					ps63.OverlayValues[14] = d14
					ps63.OverlayValues[15] = d15
					ps63.OverlayValues[16] = d16
					ps63.OverlayValues[17] = d17
					ps63.OverlayValues[18] = d18
					ps63.OverlayValues[19] = d19
					ps63.OverlayValues[20] = d20
					ps63.OverlayValues[21] = d21
					ps63.OverlayValues[22] = d22
					ps63.OverlayValues[25] = d25
					ps63.OverlayValues[28] = d28
					ps63.OverlayValues[47] = d47
					ps63.OverlayValues[48] = d48
					ps63.OverlayValues[49] = d49
					ps63.OverlayValues[50] = d50
					ps63.OverlayValues[51] = d51
					ps63.OverlayValues[53] = d53
					ps63.OverlayValues[54] = d54
					ps63.OverlayValues[55] = d55
					ps63.OverlayValues[56] = d56
					ps63.OverlayValues[57] = d57
					ps63.OverlayValues[58] = d58
					ps63.OverlayValues[59] = d59
					ps63.OverlayValues[62] = d62
					ps64 := PhiState{General: true}
					ps64.OverlayValues = make([]JITValueDesc, 63)
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
					ps64.OverlayValues[25] = d25
					ps64.OverlayValues[28] = d28
					ps64.OverlayValues[47] = d47
					ps64.OverlayValues[48] = d48
					ps64.OverlayValues[49] = d49
					ps64.OverlayValues[50] = d50
					ps64.OverlayValues[51] = d51
					ps64.OverlayValues[53] = d53
					ps64.OverlayValues[54] = d54
					ps64.OverlayValues[55] = d55
					ps64.OverlayValues[56] = d56
					ps64.OverlayValues[57] = d57
					ps64.OverlayValues[58] = d58
					ps64.OverlayValues[59] = d59
					ps64.OverlayValues[62] = d62
					snap65 := d8
					snap66 := d9
					snap67 := d10
					snap68 := d11
					snap69 := d12
					snap70 := d13
					snap71 := d14
					snap72 := d15
					snap73 := d16
					snap74 := d17
					snap75 := d18
					snap76 := d19
					snap77 := d20
					snap78 := d21
					snap79 := d22
					snap80 := d25
					snap81 := d28
					snap82 := d47
					snap83 := d48
					snap84 := d49
					snap85 := d50
					snap86 := d51
					snap87 := d53
					snap88 := d54
					snap89 := d55
					snap90 := d56
					snap91 := d57
					snap92 := d58
					snap93 := d59
					snap94 := d62
					alloc95 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps64)
					}
					ctx.RestoreAllocState(alloc95)
					d8 = snap65
					d9 = snap66
					d10 = snap67
					d11 = snap68
					d12 = snap69
					d13 = snap70
					d14 = snap71
					d15 = snap72
					d16 = snap73
					d17 = snap74
					d18 = snap75
					d19 = snap76
					d20 = snap77
					d21 = snap78
					d22 = snap79
					d25 = snap80
					d28 = snap81
					d47 = snap82
					d48 = snap83
					d49 = snap84
					d50 = snap85
					d51 = snap86
					d53 = snap87
					d54 = snap88
					d55 = snap89
					d56 = snap90
					d57 = snap91
					d58 = snap92
					d59 = snap93
					d62 = snap94
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps63)
					}
					return result
					ctx.FreeDesc(&d58)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					ps96 := PhiState{General: ps.General}
					ps96.OverlayValues = make([]JITValueDesc, 63)
					ps96.OverlayValues[8] = d8
					ps96.OverlayValues[9] = d9
					ps96.OverlayValues[10] = d10
					ps96.OverlayValues[11] = d11
					ps96.OverlayValues[12] = d12
					ps96.OverlayValues[13] = d13
					ps96.OverlayValues[14] = d14
					ps96.OverlayValues[15] = d15
					ps96.OverlayValues[16] = d16
					ps96.OverlayValues[17] = d17
					ps96.OverlayValues[18] = d18
					ps96.OverlayValues[19] = d19
					ps96.OverlayValues[20] = d20
					ps96.OverlayValues[21] = d21
					ps96.OverlayValues[22] = d22
					ps96.OverlayValues[25] = d25
					ps96.OverlayValues[28] = d28
					ps96.OverlayValues[47] = d47
					ps96.OverlayValues[48] = d48
					ps96.OverlayValues[49] = d49
					ps96.OverlayValues[50] = d50
					ps96.OverlayValues[51] = d51
					ps96.OverlayValues[53] = d53
					ps96.OverlayValues[54] = d54
					ps96.OverlayValues[55] = d55
					ps96.OverlayValues[56] = d56
					ps96.OverlayValues[57] = d57
					ps96.OverlayValues[58] = d58
					ps96.OverlayValues[59] = d59
					ps96.OverlayValues[62] = d62
					ps96.PhiValues = make([]JITValueDesc, 4)
					d97 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps96.PhiValues[0] = d97
					d98 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps96.PhiValues[1] = d98
					d99 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps96.PhiValues[2] = d99
					d100 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps96.PhiValues[3] = d100
					if ps96.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps96)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d101 := ps.PhiValues[0]
							ctx.EnsureDesc(&d101)
							ctx.EmitStoreToStack(d101, int32(bbs[4].PhiBase)+int32(0))
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d9 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						ctx.EmitMakeFloat(result, d9)
					} else {
						ctx.EmitMovToReg(result.Reg2, d9)
						d102 := JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d102)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					ps103 := PhiState{General: ps.General}
					ps103.OverlayValues = make([]JITValueDesc, 103)
					ps103.OverlayValues[8] = d8
					ps103.OverlayValues[9] = d9
					ps103.OverlayValues[10] = d10
					ps103.OverlayValues[11] = d11
					ps103.OverlayValues[12] = d12
					ps103.OverlayValues[13] = d13
					ps103.OverlayValues[14] = d14
					ps103.OverlayValues[15] = d15
					ps103.OverlayValues[16] = d16
					ps103.OverlayValues[17] = d17
					ps103.OverlayValues[18] = d18
					ps103.OverlayValues[19] = d19
					ps103.OverlayValues[20] = d20
					ps103.OverlayValues[21] = d21
					ps103.OverlayValues[22] = d22
					ps103.OverlayValues[25] = d25
					ps103.OverlayValues[28] = d28
					ps103.OverlayValues[47] = d47
					ps103.OverlayValues[48] = d48
					ps103.OverlayValues[49] = d49
					ps103.OverlayValues[50] = d50
					ps103.OverlayValues[51] = d51
					ps103.OverlayValues[53] = d53
					ps103.OverlayValues[54] = d54
					ps103.OverlayValues[55] = d55
					ps103.OverlayValues[56] = d56
					ps103.OverlayValues[57] = d57
					ps103.OverlayValues[58] = d58
					ps103.OverlayValues[59] = d59
					ps103.OverlayValues[62] = d62
					ps103.OverlayValues[97] = d97
					ps103.OverlayValues[98] = d98
					ps103.OverlayValues[99] = d99
					ps103.OverlayValues[100] = d100
					ps103.OverlayValues[101] = d101
					ps103.OverlayValues[102] = d102
					ps103.PhiValues = make([]JITValueDesc, 2)
					d104 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps103.PhiValues[0] = d104
					d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps103.PhiValues[1] = d105
					if ps103.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps103)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d106 := ps.PhiValues[0]
							ctx.EnsureDesc(&d106)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d106)
							} else {
								ctx.EmitStoreToStack(d106, int32(bbs[6].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d107 := ps.PhiValues[1]
							ctx.EnsureDesc(&d107)
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d107)
							} else {
								ctx.EmitStoreToStack(d107, int32(bbs[6].PhiBase)+int32(16))
							}
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d108 := ps.PhiValues[2]
							ctx.EnsureDesc(&d108)
							if phiHomeOK4 {
								ctx.EmitMovToReg(r2, d108)
							} else {
								ctx.EmitStoreToStack(d108, int32(bbs[6].PhiBase)+int32(32))
							}
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d109 := ps.PhiValues[3]
							ctx.EnsureDesc(&d109)
							if phiHomeOK5 {
								ctx.EmitMovToReg(r3, d109)
							} else {
								ctx.EmitStoreToStack(d109, int32(bbs[6].PhiBase)+int32(48))
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if phiHomeOK2 && d10.Loc == LocReg {
						ctx.BindReg(r0, &d10)
					}
					if phiHomeOK3 && d11.Loc == LocReg {
						ctx.BindReg(r1, &d11)
					}
					if phiHomeOK4 && d12.Loc == LocReg {
						ctx.BindReg(r2, &d12)
					}
					if phiHomeOK5 && d13.Loc == LocReg {
						ctx.BindReg(r3, &d13)
					}
					ctx.ReclaimUntrackedRegs()
					var d110 JITValueDesc
					if d17.SliceSizeKnown {
						d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d110 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d110 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d110 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d110)
					ctx.EnsureDescsTogether(&d13, &d110)
					var d111 JITValueDesc
					if d13.Loc == LocImm && d110.Loc == LocImm {
						d111 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d110.Imm.Int())}
					} else if d110.Loc == LocImm {
						r7 := ctx.AllocRegExcept(d13.Reg)
						if d110.Imm.Int() >= -2147483648 && d110.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d110.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d110.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, RegR11)
						}
						ctx.EmitSetcc(r7, CondSignedLess)
						d111 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
						ctx.BindReg(r7, &d111)
					} else if d13.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d110.Reg)
						ctx.EmitSetcc(r8, CondSignedLess)
						d111 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d111)
					} else {
						r9 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpInt64(d13.Reg, d110.Reg)
						ctx.EmitSetcc(r9, CondSignedLess)
						d111 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d111)
					}
					ctx.FreeDesc(&d110)
					d112 = d111
					ctx.EnsureDesc(&d112)
					if d112.Loc != LocImm && d112.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d112.Loc == LocImm {
						if d112.Imm.Bool() {
							if ps.General {
							}
							ps113 := PhiState{General: ps.General}
							ps113.OverlayValues = make([]JITValueDesc, 113)
							ps113.OverlayValues[8] = d8
							ps113.OverlayValues[9] = d9
							ps113.OverlayValues[10] = d10
							ps113.OverlayValues[11] = d11
							ps113.OverlayValues[12] = d12
							ps113.OverlayValues[13] = d13
							ps113.OverlayValues[14] = d14
							ps113.OverlayValues[15] = d15
							ps113.OverlayValues[16] = d16
							ps113.OverlayValues[17] = d17
							ps113.OverlayValues[18] = d18
							ps113.OverlayValues[19] = d19
							ps113.OverlayValues[20] = d20
							ps113.OverlayValues[21] = d21
							ps113.OverlayValues[22] = d22
							ps113.OverlayValues[25] = d25
							ps113.OverlayValues[28] = d28
							ps113.OverlayValues[47] = d47
							ps113.OverlayValues[48] = d48
							ps113.OverlayValues[49] = d49
							ps113.OverlayValues[50] = d50
							ps113.OverlayValues[51] = d51
							ps113.OverlayValues[53] = d53
							ps113.OverlayValues[54] = d54
							ps113.OverlayValues[55] = d55
							ps113.OverlayValues[56] = d56
							ps113.OverlayValues[57] = d57
							ps113.OverlayValues[58] = d58
							ps113.OverlayValues[59] = d59
							ps113.OverlayValues[62] = d62
							ps113.OverlayValues[97] = d97
							ps113.OverlayValues[98] = d98
							ps113.OverlayValues[99] = d99
							ps113.OverlayValues[100] = d100
							ps113.OverlayValues[101] = d101
							ps113.OverlayValues[102] = d102
							ps113.OverlayValues[104] = d104
							ps113.OverlayValues[105] = d105
							ps113.OverlayValues[106] = d106
							ps113.OverlayValues[107] = d107
							ps113.OverlayValues[108] = d108
							ps113.OverlayValues[109] = d109
							ps113.OverlayValues[110] = d110
							ps113.OverlayValues[111] = d111
							ps113.OverlayValues[112] = d112
							return bbs[9].RenderPS(ps113)
						}
						if ps.General {
						}
						ps114 := PhiState{General: ps.General}
						ps114.OverlayValues = make([]JITValueDesc, 113)
						ps114.OverlayValues[8] = d8
						ps114.OverlayValues[9] = d9
						ps114.OverlayValues[10] = d10
						ps114.OverlayValues[11] = d11
						ps114.OverlayValues[12] = d12
						ps114.OverlayValues[13] = d13
						ps114.OverlayValues[14] = d14
						ps114.OverlayValues[15] = d15
						ps114.OverlayValues[16] = d16
						ps114.OverlayValues[17] = d17
						ps114.OverlayValues[18] = d18
						ps114.OverlayValues[19] = d19
						ps114.OverlayValues[20] = d20
						ps114.OverlayValues[21] = d21
						ps114.OverlayValues[22] = d22
						ps114.OverlayValues[25] = d25
						ps114.OverlayValues[28] = d28
						ps114.OverlayValues[47] = d47
						ps114.OverlayValues[48] = d48
						ps114.OverlayValues[49] = d49
						ps114.OverlayValues[50] = d50
						ps114.OverlayValues[51] = d51
						ps114.OverlayValues[53] = d53
						ps114.OverlayValues[54] = d54
						ps114.OverlayValues[55] = d55
						ps114.OverlayValues[56] = d56
						ps114.OverlayValues[57] = d57
						ps114.OverlayValues[58] = d58
						ps114.OverlayValues[59] = d59
						ps114.OverlayValues[62] = d62
						ps114.OverlayValues[97] = d97
						ps114.OverlayValues[98] = d98
						ps114.OverlayValues[99] = d99
						ps114.OverlayValues[100] = d100
						ps114.OverlayValues[101] = d101
						ps114.OverlayValues[102] = d102
						ps114.OverlayValues[104] = d104
						ps114.OverlayValues[105] = d105
						ps114.OverlayValues[106] = d106
						ps114.OverlayValues[107] = d107
						ps114.OverlayValues[108] = d108
						ps114.OverlayValues[109] = d109
						ps114.OverlayValues[110] = d110
						ps114.OverlayValues[111] = d111
						ps114.OverlayValues[112] = d112
						return bbs[8].RenderPS(ps114)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d115 := ps.PhiValues[0]
							ctx.EnsureDesc(&d115)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d115)
							} else {
								ctx.EmitStoreToStack(d115, int32(bbs[6].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d116 := ps.PhiValues[1]
							ctx.EnsureDesc(&d116)
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d116)
							} else {
								ctx.EmitStoreToStack(d116, int32(bbs[6].PhiBase)+int32(16))
							}
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d117 := ps.PhiValues[2]
							ctx.EnsureDesc(&d117)
							if phiHomeOK4 {
								ctx.EmitMovToReg(r2, d117)
							} else {
								ctx.EmitStoreToStack(d117, int32(bbs[6].PhiBase)+int32(32))
							}
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d118 := ps.PhiValues[3]
							ctx.EnsureDesc(&d118)
							if phiHomeOK5 {
								ctx.EmitMovToReg(r3, d118)
							} else {
								ctx.EmitStoreToStack(d118, int32(bbs[6].PhiBase)+int32(48))
							}
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d112.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl9)
					ps119 := PhiState{General: true}
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
					ps119.OverlayValues[25] = d25
					ps119.OverlayValues[28] = d28
					ps119.OverlayValues[47] = d47
					ps119.OverlayValues[48] = d48
					ps119.OverlayValues[49] = d49
					ps119.OverlayValues[50] = d50
					ps119.OverlayValues[51] = d51
					ps119.OverlayValues[53] = d53
					ps119.OverlayValues[54] = d54
					ps119.OverlayValues[55] = d55
					ps119.OverlayValues[56] = d56
					ps119.OverlayValues[57] = d57
					ps119.OverlayValues[58] = d58
					ps119.OverlayValues[59] = d59
					ps119.OverlayValues[62] = d62
					ps119.OverlayValues[97] = d97
					ps119.OverlayValues[98] = d98
					ps119.OverlayValues[99] = d99
					ps119.OverlayValues[100] = d100
					ps119.OverlayValues[101] = d101
					ps119.OverlayValues[102] = d102
					ps119.OverlayValues[104] = d104
					ps119.OverlayValues[105] = d105
					ps119.OverlayValues[106] = d106
					ps119.OverlayValues[107] = d107
					ps119.OverlayValues[108] = d108
					ps119.OverlayValues[109] = d109
					ps119.OverlayValues[110] = d110
					ps119.OverlayValues[111] = d111
					ps119.OverlayValues[112] = d112
					ps119.OverlayValues[115] = d115
					ps119.OverlayValues[116] = d116
					ps119.OverlayValues[117] = d117
					ps119.OverlayValues[118] = d118
					ps120 := PhiState{General: true}
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
					ps120.OverlayValues[25] = d25
					ps120.OverlayValues[28] = d28
					ps120.OverlayValues[47] = d47
					ps120.OverlayValues[48] = d48
					ps120.OverlayValues[49] = d49
					ps120.OverlayValues[50] = d50
					ps120.OverlayValues[51] = d51
					ps120.OverlayValues[53] = d53
					ps120.OverlayValues[54] = d54
					ps120.OverlayValues[55] = d55
					ps120.OverlayValues[56] = d56
					ps120.OverlayValues[57] = d57
					ps120.OverlayValues[58] = d58
					ps120.OverlayValues[59] = d59
					ps120.OverlayValues[62] = d62
					ps120.OverlayValues[97] = d97
					ps120.OverlayValues[98] = d98
					ps120.OverlayValues[99] = d99
					ps120.OverlayValues[100] = d100
					ps120.OverlayValues[101] = d101
					ps120.OverlayValues[102] = d102
					ps120.OverlayValues[104] = d104
					ps120.OverlayValues[105] = d105
					ps120.OverlayValues[106] = d106
					ps120.OverlayValues[107] = d107
					ps120.OverlayValues[108] = d108
					ps120.OverlayValues[109] = d109
					ps120.OverlayValues[110] = d110
					ps120.OverlayValues[111] = d111
					ps120.OverlayValues[112] = d112
					ps120.OverlayValues[115] = d115
					ps120.OverlayValues[116] = d116
					ps120.OverlayValues[117] = d117
					ps120.OverlayValues[118] = d118
					snap121 := d8
					snap122 := d9
					snap123 := d10
					snap124 := d11
					snap125 := d12
					snap126 := d13
					snap127 := d14
					snap128 := d15
					snap129 := d16
					snap130 := d17
					snap131 := d18
					snap132 := d19
					snap133 := d20
					snap134 := d21
					snap135 := d22
					snap136 := d25
					snap137 := d28
					snap138 := d47
					snap139 := d48
					snap140 := d49
					snap141 := d50
					snap142 := d51
					snap143 := d53
					snap144 := d54
					snap145 := d55
					snap146 := d56
					snap147 := d57
					snap148 := d58
					snap149 := d59
					snap150 := d62
					snap151 := d97
					snap152 := d98
					snap153 := d99
					snap154 := d100
					snap155 := d101
					snap156 := d102
					snap157 := d104
					snap158 := d105
					snap159 := d106
					snap160 := d107
					snap161 := d108
					snap162 := d109
					snap163 := d110
					snap164 := d111
					snap165 := d112
					snap166 := d115
					snap167 := d116
					snap168 := d117
					snap169 := d118
					alloc170 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps120)
					}
					ctx.RestoreAllocState(alloc170)
					d8 = snap121
					d9 = snap122
					d10 = snap123
					d11 = snap124
					d12 = snap125
					d13 = snap126
					d14 = snap127
					d15 = snap128
					d16 = snap129
					d17 = snap130
					d18 = snap131
					d19 = snap132
					d20 = snap133
					d21 = snap134
					d22 = snap135
					d25 = snap136
					d28 = snap137
					d47 = snap138
					d48 = snap139
					d49 = snap140
					d50 = snap141
					d51 = snap142
					d53 = snap143
					d54 = snap144
					d55 = snap145
					d56 = snap146
					d57 = snap147
					d58 = snap148
					d59 = snap149
					d62 = snap150
					d97 = snap151
					d98 = snap152
					d99 = snap153
					d100 = snap154
					d101 = snap155
					d102 = snap156
					d104 = snap157
					d105 = snap158
					d106 = snap159
					d107 = snap160
					d108 = snap161
					d109 = snap162
					d110 = snap163
					d111 = snap164
					d112 = snap165
					d115 = snap166
					d116 = snap167
					d117 = snap168
					d118 = snap169
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps119)
					}
					return result
					ctx.FreeDesc(&d111)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					d172 = ctx.EmitSliceElementAddress(&d17, &d13, 16)
					ctx.EnsureDesc(&d172)
					r10 := ctx.AllocRegExcept(d172.Reg)
					ctx.EmitMovRegMem(r10, d172.Reg, 8)
					ctx.EmitMovRegMem(d172.Reg, d172.Reg, 0)
					d171 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d172.Reg, Reg2: r10}
					ctx.BindReg(d172.Reg, &d171)
					ctx.BindReg(r10, &d171)
					ctx.EnsureDesc(&d171)
					d173 = d171
					_ = d173
					ctx.StabilizeDescForControlFlow(&d173)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl22 := ctx.ReserveLabel()
					_ = lbl22
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d174 JITValueDesc
					if d173.Loc == LocImm {
						d174 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d173.Imm.Float())}
					} else if d173.Type == tagFloat && d173.Loc == LocReg {
						d174 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d173.Reg}
						ctx.BindReg(d173.Reg, &d174)
						ctx.BindReg(d173.Reg, &d174)
					} else if d173.Type == tagFloat && d173.Loc == LocRegPair {
						ctx.FreeReg(d173.Reg)
						d174 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d173.Reg2}
						ctx.BindReg(d173.Reg2, &d174)
						ctx.BindReg(d173.Reg2, &d174)
					} else {
						d174 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d173}, 1)
						d174.Type = tagFloat
						ctx.BindReg(d174.Reg, &d174)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d174)
					ctx.FreeDesc(&d171)
					ctx.EnsureDesc(&d13)
					d176 = ctx.EmitSliceElementAddress(&d19, &d13, 16)
					ctx.EnsureDesc(&d176)
					r11 := ctx.AllocRegExcept(d176.Reg)
					ctx.EmitMovRegMem(r11, d176.Reg, 8)
					ctx.EmitMovRegMem(d176.Reg, d176.Reg, 0)
					d175 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d176.Reg, Reg2: r11}
					ctx.BindReg(d176.Reg, &d175)
					ctx.BindReg(r11, &d175)
					ctx.EnsureDesc(&d175)
					d177 = d175
					_ = d177
					ctx.StabilizeDescForControlFlow(&d177)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl23 := ctx.ReserveLabel()
					_ = lbl23
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d178 JITValueDesc
					if d177.Loc == LocImm {
						d178 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d177.Imm.Float())}
					} else if d177.Type == tagFloat && d177.Loc == LocReg {
						d178 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d177.Reg}
						ctx.BindReg(d177.Reg, &d178)
						ctx.BindReg(d177.Reg, &d178)
					} else if d177.Type == tagFloat && d177.Loc == LocRegPair {
						ctx.FreeReg(d177.Reg)
						d178 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d177.Reg2}
						ctx.BindReg(d177.Reg2, &d178)
						ctx.BindReg(d177.Reg2, &d178)
					} else {
						d178 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d177}, 1)
						d178.Type = tagFloat
						ctx.BindReg(d178.Reg, &d178)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d178)
					ctx.FreeDesc(&d175)
					ctx.EnsureDesc(&d174)
					ctx.EnsureDesc(&d174)
					ctx.EnsureDescsTogether(&d174, &d174)
					var d179 JITValueDesc
					if d174.Loc == LocImm {
						d179 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d174.Imm.Float() * d174.Imm.Float())}
					} else if d174.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d174.Reg)
						_, xBits := d174.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d174.Reg)
						d179 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d179)
					} else if d174.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d174.Reg)
						ctx.EmitMovRegReg(scratch, d174.Reg)
						_, yBits := d174.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d179 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d179)
					} else {
						r12 := ctx.AllocRegExcept(d174.Reg, d174.Reg)
						ctx.EmitMovRegReg(r12, d174.Reg)
						ctx.EmitMulFloat64(r12, d174.Reg)
						d179 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
						ctx.BindReg(r12, &d179)
					}
					if d179.Loc == LocReg && d174.Loc == LocReg && d179.Reg == d174.Reg {
						ctx.TransferReg(d174.Reg)
						d174.Loc = LocNone
					}
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d179)
					ctx.EnsureDescsTogether(&d11, &d179)
					var d180 JITValueDesc
					if d11.Loc == LocImm && d179.Loc == LocImm {
						d180 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d11.Imm.Float() + d179.Imm.Float())}
					} else if d11.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 && r1 != d179.Reg {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d179.Reg)
						}
						_, xBits := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d179.Reg)
						d180 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d180)
					} else if d179.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d11.Reg)
						}
						ctx.EmitMovRegReg(scratch, d11.Reg)
						_, yBits := d179.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d180 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d180)
					} else {
						var r13 Reg
						if phiHomeOK3 && r1 != d179.Reg {
							r13 = r1
						} else {
							r13 = ctx.AllocRegExcept(d11.Reg, d179.Reg)
						}
						ctx.EmitMovRegReg(r13, d11.Reg)
						ctx.EmitAddFloat64(r13, d179.Reg)
						d180 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r13}
						ctx.BindReg(r13, &d180)
					}
					if d180.Loc == LocReg && d11.Loc == LocReg && d180.Reg == d11.Reg {
						ctx.TransferReg(d11.Reg)
						d11.Loc = LocNone
					}
					ctx.FreeDesc(&d179)
					ctx.EnsureDesc(&d178)
					ctx.EnsureDesc(&d178)
					ctx.EnsureDescsTogether(&d178, &d178)
					var d181 JITValueDesc
					if d178.Loc == LocImm {
						d181 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d178.Imm.Float() * d178.Imm.Float())}
					} else if d178.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d178.Reg)
						_, xBits := d178.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d178.Reg)
						d181 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d181)
					} else if d178.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d178.Reg)
						ctx.EmitMovRegReg(scratch, d178.Reg)
						_, yBits := d178.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d181 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d181)
					} else {
						r14 := ctx.AllocRegExcept(d178.Reg, d178.Reg)
						ctx.EmitMovRegReg(r14, d178.Reg)
						ctx.EmitMulFloat64(r14, d178.Reg)
						d181 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r14}
						ctx.BindReg(r14, &d181)
					}
					if d181.Loc == LocReg && d178.Loc == LocReg && d181.Reg == d178.Reg {
						ctx.TransferReg(d178.Reg)
						d178.Loc = LocNone
					}
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d181)
					ctx.EnsureDescsTogether(&d12, &d181)
					var d182 JITValueDesc
					if d12.Loc == LocImm && d181.Loc == LocImm {
						d182 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d12.Imm.Float() + d181.Imm.Float())}
					} else if d12.Loc == LocImm {
						var scratch Reg
						if phiHomeOK4 && r2 != d181.Reg {
							scratch = r2
						} else {
							scratch = ctx.AllocRegExcept(d181.Reg)
						}
						_, xBits := d12.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d181.Reg)
						d182 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d182)
					} else if d181.Loc == LocImm {
						var scratch Reg
						if phiHomeOK4 {
							scratch = r2
						} else {
							scratch = ctx.AllocRegExcept(d12.Reg)
						}
						ctx.EmitMovRegReg(scratch, d12.Reg)
						_, yBits := d181.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d182 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d182)
					} else {
						var r15 Reg
						if phiHomeOK4 && r2 != d181.Reg {
							r15 = r2
						} else {
							r15 = ctx.AllocRegExcept(d12.Reg, d181.Reg)
						}
						ctx.EmitMovRegReg(r15, d12.Reg)
						ctx.EmitAddFloat64(r15, d181.Reg)
						d182 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r15}
						ctx.BindReg(r15, &d182)
					}
					if d182.Loc == LocReg && d12.Loc == LocReg && d182.Reg == d12.Reg {
						ctx.TransferReg(d12.Reg)
						d12.Loc = LocNone
					}
					ctx.FreeDesc(&d181)
					ctx.EnsureDesc(&d174)
					ctx.EnsureDesc(&d178)
					ctx.EnsureDescsTogether(&d174, &d178)
					var d183 JITValueDesc
					if d174.Loc == LocImm && d178.Loc == LocImm {
						d183 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d174.Imm.Float() * d178.Imm.Float())}
					} else if d174.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d178.Reg)
						_, xBits := d174.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d178.Reg)
						d183 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d183)
					} else if d178.Loc == LocImm {
						_, yBits := d178.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d174.Reg, RegR11)
						d183 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d174.Reg}
						ctx.BindReg(d174.Reg, &d183)
					} else {
						ctx.EmitMulFloat64(d174.Reg, d178.Reg)
						d183 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d174.Reg}
						ctx.BindReg(d174.Reg, &d183)
					}
					if d183.Loc == LocReg && d174.Loc == LocReg && d183.Reg == d174.Reg {
						ctx.TransferReg(d174.Reg)
						d174.Loc = LocNone
					}
					ctx.FreeDesc(&d174)
					ctx.FreeDesc(&d178)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d183)
					ctx.EnsureDescsTogether(&d10, &d183)
					var d184 JITValueDesc
					if d10.Loc == LocImm && d183.Loc == LocImm {
						d184 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float() + d183.Imm.Float())}
					} else if d10.Loc == LocImm {
						var scratch Reg
						if phiHomeOK2 && r0 != d183.Reg {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d183.Reg)
						}
						_, xBits := d10.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d183.Reg)
						d184 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d184)
					} else if d183.Loc == LocImm {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d10.Reg)
						}
						ctx.EmitMovRegReg(scratch, d10.Reg)
						_, yBits := d183.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d184 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d184)
					} else {
						var r16 Reg
						if phiHomeOK2 && r0 != d183.Reg {
							r16 = r0
						} else {
							r16 = ctx.AllocRegExcept(d10.Reg, d183.Reg)
						}
						ctx.EmitMovRegReg(r16, d10.Reg)
						ctx.EmitAddFloat64(r16, d183.Reg)
						d184 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
						ctx.BindReg(r16, &d184)
					}
					if d184.Loc == LocReg && d10.Loc == LocReg && d184.Reg == d10.Reg {
						ctx.TransferReg(d10.Reg)
						d10.Loc = LocNone
					}
					ctx.FreeDesc(&d183)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					var d185 JITValueDesc
					if d13.Loc == LocImm {
						d185 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK5 {
							scratch = r3
						} else {
							scratch = ctx.AllocRegExcept(d13.Reg)
						}
						ctx.EmitMovRegReg(scratch, d13.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d185 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d185)
					}
					if d185.Loc == LocReg && d13.Loc == LocReg && d185.Reg == d13.Reg {
						ctx.TransferReg(d13.Reg)
						d13.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d180)
						if d180.Loc == LocReg {
							ctx.ProtectReg(d180.Reg)
						} else if d180.Loc == LocRegPair {
							ctx.ProtectReg(d180.Reg)
							ctx.ProtectReg(d180.Reg2)
						}
						ctx.SyncDesc(&d182)
						if d182.Loc == LocReg {
							ctx.ProtectReg(d182.Reg)
						} else if d182.Loc == LocRegPair {
							ctx.ProtectReg(d182.Reg)
							ctx.ProtectReg(d182.Reg2)
						}
						ctx.SyncDesc(&d184)
						if d184.Loc == LocReg {
							ctx.ProtectReg(d184.Reg)
						} else if d184.Loc == LocRegPair {
							ctx.ProtectReg(d184.Reg)
							ctx.ProtectReg(d184.Reg2)
						}
						d186 = d184
						if d186.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d186)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d186)
						} else {
							ctx.EmitStoreToStack(d186, int32(bbs[6].PhiBase)+int32(0))
						}
						d187 = d180
						if d187.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d187)
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, d187)
						} else {
							ctx.EmitStoreToStack(d187, int32(bbs[6].PhiBase)+int32(16))
						}
						d188 = d182
						if d188.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d188)
						if phiHomeOK4 {
							ctx.EmitMovToReg(r2, d188)
						} else {
							ctx.EmitStoreToStack(d188, int32(bbs[6].PhiBase)+int32(32))
						}
						if d180.Loc == LocReg {
							ctx.UnprotectReg(d180.Reg)
						} else if d180.Loc == LocRegPair {
							ctx.UnprotectReg(d180.Reg)
							ctx.UnprotectReg(d180.Reg2)
						}
						if d182.Loc == LocReg {
							ctx.UnprotectReg(d182.Reg)
						} else if d182.Loc == LocRegPair {
							ctx.UnprotectReg(d182.Reg)
							ctx.UnprotectReg(d182.Reg2)
						}
						if d184.Loc == LocReg {
							ctx.UnprotectReg(d184.Reg)
						} else if d184.Loc == LocRegPair {
							ctx.UnprotectReg(d184.Reg)
							ctx.UnprotectReg(d184.Reg2)
						}
						ctx.SyncDesc(&d185)
						if d185.Loc == LocReg {
							ctx.ProtectReg(d185.Reg)
						} else if d185.Loc == LocRegPair {
							ctx.ProtectReg(d185.Reg)
							ctx.ProtectReg(d185.Reg2)
						}
						d189 = d185
						if d189.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d189)
						if phiHomeOK5 {
							ctx.EmitMovToReg(r3, d189)
						} else {
							ctx.EmitStoreToStack(d189, int32(bbs[6].PhiBase)+int32(48))
						}
						if d185.Loc == LocReg {
							ctx.UnprotectReg(d185.Reg)
						} else if d185.Loc == LocRegPair {
							ctx.UnprotectReg(d185.Reg)
							ctx.UnprotectReg(d185.Reg2)
						}
					}
					ps190 := PhiState{General: ps.General}
					ps190.OverlayValues = make([]JITValueDesc, 190)
					ps190.OverlayValues[8] = d8
					ps190.OverlayValues[9] = d9
					ps190.OverlayValues[10] = d10
					ps190.OverlayValues[11] = d11
					ps190.OverlayValues[12] = d12
					ps190.OverlayValues[13] = d13
					ps190.OverlayValues[14] = d14
					ps190.OverlayValues[15] = d15
					ps190.OverlayValues[16] = d16
					ps190.OverlayValues[17] = d17
					ps190.OverlayValues[18] = d18
					ps190.OverlayValues[19] = d19
					ps190.OverlayValues[20] = d20
					ps190.OverlayValues[21] = d21
					ps190.OverlayValues[22] = d22
					ps190.OverlayValues[25] = d25
					ps190.OverlayValues[28] = d28
					ps190.OverlayValues[47] = d47
					ps190.OverlayValues[48] = d48
					ps190.OverlayValues[49] = d49
					ps190.OverlayValues[50] = d50
					ps190.OverlayValues[51] = d51
					ps190.OverlayValues[53] = d53
					ps190.OverlayValues[54] = d54
					ps190.OverlayValues[55] = d55
					ps190.OverlayValues[56] = d56
					ps190.OverlayValues[57] = d57
					ps190.OverlayValues[58] = d58
					ps190.OverlayValues[59] = d59
					ps190.OverlayValues[62] = d62
					ps190.OverlayValues[97] = d97
					ps190.OverlayValues[98] = d98
					ps190.OverlayValues[99] = d99
					ps190.OverlayValues[100] = d100
					ps190.OverlayValues[101] = d101
					ps190.OverlayValues[102] = d102
					ps190.OverlayValues[104] = d104
					ps190.OverlayValues[105] = d105
					ps190.OverlayValues[106] = d106
					ps190.OverlayValues[107] = d107
					ps190.OverlayValues[108] = d108
					ps190.OverlayValues[109] = d109
					ps190.OverlayValues[110] = d110
					ps190.OverlayValues[111] = d111
					ps190.OverlayValues[112] = d112
					ps190.OverlayValues[115] = d115
					ps190.OverlayValues[116] = d116
					ps190.OverlayValues[117] = d117
					ps190.OverlayValues[118] = d118
					ps190.OverlayValues[171] = d171
					ps190.OverlayValues[172] = d172
					ps190.OverlayValues[173] = d173
					ps190.OverlayValues[174] = d174
					ps190.OverlayValues[175] = d175
					ps190.OverlayValues[176] = d176
					ps190.OverlayValues[177] = d177
					ps190.OverlayValues[178] = d178
					ps190.OverlayValues[179] = d179
					ps190.OverlayValues[180] = d180
					ps190.OverlayValues[181] = d181
					ps190.OverlayValues[182] = d182
					ps190.OverlayValues[183] = d183
					ps190.OverlayValues[184] = d184
					ps190.OverlayValues[185] = d185
					ps190.OverlayValues[186] = d186
					ps190.OverlayValues[187] = d187
					ps190.OverlayValues[188] = d188
					ps190.OverlayValues[189] = d189
					ps190.PhiValues = make([]JITValueDesc, 4)
					d191 = d184
					ps190.PhiValues[0] = d191
					d192 = d180
					ps190.PhiValues[1] = d192
					d193 = d182
					ps190.PhiValues[2] = d193
					d194 = d185
					ps190.PhiValues[3] = d194
					if ps190.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps190)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDescsTogether(&d11, &d12)
					var d195 JITValueDesc
					if d11.Loc == LocImm && d12.Loc == LocImm {
						d195 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d11.Imm.Float() * d12.Imm.Float())}
					} else if d11.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d12.Reg)
						_, xBits := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d12.Reg)
						d195 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d195)
					} else if d12.Loc == LocImm {
						_, yBits := d12.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d11.Reg, RegR11)
						d195 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d11.Reg}
						ctx.BindReg(d11.Reg, &d195)
					} else {
						ctx.EmitMulFloat64(d11.Reg, d12.Reg)
						d195 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d11.Reg}
						ctx.BindReg(d11.Reg, &d195)
					}
					if d195.Loc == LocReg && d11.Loc == LocReg && d195.Reg == d11.Reg {
						ctx.TransferReg(d11.Reg)
						d11.Loc = LocNone
					}
					ctx.FreeDesc(&d11)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d195)
					var d196 JITValueDesc
					if d195.Loc == LocImm {
						d196 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d195.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d195)
						var d197 JITValueDesc
						if d195.Loc == LocRegPair {
							ctx.FreeReg(d195.Reg)
							d197 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d195.Reg2}
							ctx.BindReg(d195.Reg2, &d197)
							ctx.BindReg(d195.Reg2, &d197)
						} else {
							d197 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d195.Reg}
							ctx.BindReg(d195.Reg, &d197)
							ctx.BindReg(d195.Reg, &d197)
						}
						d196 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d197}, 1)
						d196.Type = tagFloat
						ctx.BindReg(d196.Reg, &d196)
					}
					ctx.FreeDesc(&d195)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d196)
					ctx.EnsureDescsTogether(&d10, &d196)
					var d198 JITValueDesc
					if d10.Loc == LocImm && d196.Loc == LocImm {
						d198 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float() / d196.Imm.Float())}
					} else if d10.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d196.Reg)
						_, xBits := d10.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d196.Reg)
						d198 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d198)
					} else if d196.Loc == LocImm {
						_, yBits := d196.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitDivFloat64(d10.Reg, RegR11)
						d198 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg}
						ctx.BindReg(d10.Reg, &d198)
					} else {
						ctx.EmitDivFloat64(d10.Reg, d196.Reg)
						d198 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg}
						ctx.BindReg(d10.Reg, &d198)
					}
					if d198.Loc == LocReg && d10.Loc == LocReg && d198.Reg == d10.Reg {
						ctx.TransferReg(d10.Reg)
						d10.Loc = LocNone
					}
					ctx.EnsureDesc(&d198)
					ctx.EmitStoreToStack(d198, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d198)
					ctx.FreeDesc(&d10)
					ctx.FreeDesc(&d196)
					if ps.General {
					}
					ps199 := PhiState{General: ps.General}
					ps199.OverlayValues = make([]JITValueDesc, 199)
					ps199.OverlayValues[8] = d8
					ps199.OverlayValues[9] = d9
					ps199.OverlayValues[10] = d10
					ps199.OverlayValues[11] = d11
					ps199.OverlayValues[12] = d12
					ps199.OverlayValues[13] = d13
					ps199.OverlayValues[14] = d14
					ps199.OverlayValues[15] = d15
					ps199.OverlayValues[16] = d16
					ps199.OverlayValues[17] = d17
					ps199.OverlayValues[18] = d18
					ps199.OverlayValues[19] = d19
					ps199.OverlayValues[20] = d20
					ps199.OverlayValues[21] = d21
					ps199.OverlayValues[22] = d22
					ps199.OverlayValues[25] = d25
					ps199.OverlayValues[28] = d28
					ps199.OverlayValues[47] = d47
					ps199.OverlayValues[48] = d48
					ps199.OverlayValues[49] = d49
					ps199.OverlayValues[50] = d50
					ps199.OverlayValues[51] = d51
					ps199.OverlayValues[53] = d53
					ps199.OverlayValues[54] = d54
					ps199.OverlayValues[55] = d55
					ps199.OverlayValues[56] = d56
					ps199.OverlayValues[57] = d57
					ps199.OverlayValues[58] = d58
					ps199.OverlayValues[59] = d59
					ps199.OverlayValues[62] = d62
					ps199.OverlayValues[97] = d97
					ps199.OverlayValues[98] = d98
					ps199.OverlayValues[99] = d99
					ps199.OverlayValues[100] = d100
					ps199.OverlayValues[101] = d101
					ps199.OverlayValues[102] = d102
					ps199.OverlayValues[104] = d104
					ps199.OverlayValues[105] = d105
					ps199.OverlayValues[106] = d106
					ps199.OverlayValues[107] = d107
					ps199.OverlayValues[108] = d108
					ps199.OverlayValues[109] = d109
					ps199.OverlayValues[110] = d110
					ps199.OverlayValues[111] = d111
					ps199.OverlayValues[112] = d112
					ps199.OverlayValues[115] = d115
					ps199.OverlayValues[116] = d116
					ps199.OverlayValues[117] = d117
					ps199.OverlayValues[118] = d118
					ps199.OverlayValues[171] = d171
					ps199.OverlayValues[172] = d172
					ps199.OverlayValues[173] = d173
					ps199.OverlayValues[174] = d174
					ps199.OverlayValues[175] = d175
					ps199.OverlayValues[176] = d176
					ps199.OverlayValues[177] = d177
					ps199.OverlayValues[178] = d178
					ps199.OverlayValues[179] = d179
					ps199.OverlayValues[180] = d180
					ps199.OverlayValues[181] = d181
					ps199.OverlayValues[182] = d182
					ps199.OverlayValues[183] = d183
					ps199.OverlayValues[184] = d184
					ps199.OverlayValues[185] = d185
					ps199.OverlayValues[186] = d186
					ps199.OverlayValues[187] = d187
					ps199.OverlayValues[188] = d188
					ps199.OverlayValues[189] = d189
					ps199.OverlayValues[191] = d191
					ps199.OverlayValues[192] = d192
					ps199.OverlayValues[193] = d193
					ps199.OverlayValues[194] = d194
					ps199.OverlayValues[195] = d195
					ps199.OverlayValues[196] = d196
					ps199.OverlayValues[197] = d197
					ps199.OverlayValues[198] = d198
					ps199.PhiValues = make([]JITValueDesc, 1)
					if ps199.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps199)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					ctx.ReclaimUntrackedRegs()
					var d200 JITValueDesc
					if d19.SliceSizeKnown {
						d200 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d200 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d200 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d200 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d200 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d200)
					ctx.EnsureDescsTogether(&d13, &d200)
					var d201 JITValueDesc
					if d13.Loc == LocImm && d200.Loc == LocImm {
						d201 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d200.Imm.Int())}
					} else if d200.Loc == LocImm {
						r17 := ctx.AllocReg()
						if d200.Imm.Int() >= -2147483648 && d200.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d200.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d200.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, RegR11)
						}
						ctx.EmitSetcc(r17, CondSignedLess)
						d201 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
						ctx.BindReg(r17, &d201)
					} else if d13.Loc == LocImm {
						r18 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d200.Reg)
						ctx.EmitSetcc(r18, CondSignedLess)
						d201 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r18}
						ctx.BindReg(r18, &d201)
					} else {
						r19 := ctx.AllocReg()
						ctx.EmitCmpInt64(d13.Reg, d200.Reg)
						ctx.EmitSetcc(r19, CondSignedLess)
						d201 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r19}
						ctx.BindReg(r19, &d201)
					}
					ctx.FreeDesc(&d13)
					ctx.FreeDesc(&d200)
					d202 = d201
					ctx.EnsureDesc(&d202)
					if d202.Loc != LocImm && d202.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d202.Loc == LocImm {
						if d202.Imm.Bool() {
							if ps.General {
							}
							ps203 := PhiState{General: ps.General}
							ps203.OverlayValues = make([]JITValueDesc, 203)
							ps203.OverlayValues[8] = d8
							ps203.OverlayValues[9] = d9
							ps203.OverlayValues[10] = d10
							ps203.OverlayValues[11] = d11
							ps203.OverlayValues[12] = d12
							ps203.OverlayValues[13] = d13
							ps203.OverlayValues[14] = d14
							ps203.OverlayValues[15] = d15
							ps203.OverlayValues[16] = d16
							ps203.OverlayValues[17] = d17
							ps203.OverlayValues[18] = d18
							ps203.OverlayValues[19] = d19
							ps203.OverlayValues[20] = d20
							ps203.OverlayValues[21] = d21
							ps203.OverlayValues[22] = d22
							ps203.OverlayValues[25] = d25
							ps203.OverlayValues[28] = d28
							ps203.OverlayValues[47] = d47
							ps203.OverlayValues[48] = d48
							ps203.OverlayValues[49] = d49
							ps203.OverlayValues[50] = d50
							ps203.OverlayValues[51] = d51
							ps203.OverlayValues[53] = d53
							ps203.OverlayValues[54] = d54
							ps203.OverlayValues[55] = d55
							ps203.OverlayValues[56] = d56
							ps203.OverlayValues[57] = d57
							ps203.OverlayValues[58] = d58
							ps203.OverlayValues[59] = d59
							ps203.OverlayValues[62] = d62
							ps203.OverlayValues[97] = d97
							ps203.OverlayValues[98] = d98
							ps203.OverlayValues[99] = d99
							ps203.OverlayValues[100] = d100
							ps203.OverlayValues[101] = d101
							ps203.OverlayValues[102] = d102
							ps203.OverlayValues[104] = d104
							ps203.OverlayValues[105] = d105
							ps203.OverlayValues[106] = d106
							ps203.OverlayValues[107] = d107
							ps203.OverlayValues[108] = d108
							ps203.OverlayValues[109] = d109
							ps203.OverlayValues[110] = d110
							ps203.OverlayValues[111] = d111
							ps203.OverlayValues[112] = d112
							ps203.OverlayValues[115] = d115
							ps203.OverlayValues[116] = d116
							ps203.OverlayValues[117] = d117
							ps203.OverlayValues[118] = d118
							ps203.OverlayValues[171] = d171
							ps203.OverlayValues[172] = d172
							ps203.OverlayValues[173] = d173
							ps203.OverlayValues[174] = d174
							ps203.OverlayValues[175] = d175
							ps203.OverlayValues[176] = d176
							ps203.OverlayValues[177] = d177
							ps203.OverlayValues[178] = d178
							ps203.OverlayValues[179] = d179
							ps203.OverlayValues[180] = d180
							ps203.OverlayValues[181] = d181
							ps203.OverlayValues[182] = d182
							ps203.OverlayValues[183] = d183
							ps203.OverlayValues[184] = d184
							ps203.OverlayValues[185] = d185
							ps203.OverlayValues[186] = d186
							ps203.OverlayValues[187] = d187
							ps203.OverlayValues[188] = d188
							ps203.OverlayValues[189] = d189
							ps203.OverlayValues[191] = d191
							ps203.OverlayValues[192] = d192
							ps203.OverlayValues[193] = d193
							ps203.OverlayValues[194] = d194
							ps203.OverlayValues[195] = d195
							ps203.OverlayValues[196] = d196
							ps203.OverlayValues[197] = d197
							ps203.OverlayValues[198] = d198
							ps203.OverlayValues[200] = d200
							ps203.OverlayValues[201] = d201
							ps203.OverlayValues[202] = d202
							return bbs[7].RenderPS(ps203)
						}
						if ps.General {
						}
						ps204 := PhiState{General: ps.General}
						ps204.OverlayValues = make([]JITValueDesc, 203)
						ps204.OverlayValues[8] = d8
						ps204.OverlayValues[9] = d9
						ps204.OverlayValues[10] = d10
						ps204.OverlayValues[11] = d11
						ps204.OverlayValues[12] = d12
						ps204.OverlayValues[13] = d13
						ps204.OverlayValues[14] = d14
						ps204.OverlayValues[15] = d15
						ps204.OverlayValues[16] = d16
						ps204.OverlayValues[17] = d17
						ps204.OverlayValues[18] = d18
						ps204.OverlayValues[19] = d19
						ps204.OverlayValues[20] = d20
						ps204.OverlayValues[21] = d21
						ps204.OverlayValues[22] = d22
						ps204.OverlayValues[25] = d25
						ps204.OverlayValues[28] = d28
						ps204.OverlayValues[47] = d47
						ps204.OverlayValues[48] = d48
						ps204.OverlayValues[49] = d49
						ps204.OverlayValues[50] = d50
						ps204.OverlayValues[51] = d51
						ps204.OverlayValues[53] = d53
						ps204.OverlayValues[54] = d54
						ps204.OverlayValues[55] = d55
						ps204.OverlayValues[56] = d56
						ps204.OverlayValues[57] = d57
						ps204.OverlayValues[58] = d58
						ps204.OverlayValues[59] = d59
						ps204.OverlayValues[62] = d62
						ps204.OverlayValues[97] = d97
						ps204.OverlayValues[98] = d98
						ps204.OverlayValues[99] = d99
						ps204.OverlayValues[100] = d100
						ps204.OverlayValues[101] = d101
						ps204.OverlayValues[102] = d102
						ps204.OverlayValues[104] = d104
						ps204.OverlayValues[105] = d105
						ps204.OverlayValues[106] = d106
						ps204.OverlayValues[107] = d107
						ps204.OverlayValues[108] = d108
						ps204.OverlayValues[109] = d109
						ps204.OverlayValues[110] = d110
						ps204.OverlayValues[111] = d111
						ps204.OverlayValues[112] = d112
						ps204.OverlayValues[115] = d115
						ps204.OverlayValues[116] = d116
						ps204.OverlayValues[117] = d117
						ps204.OverlayValues[118] = d118
						ps204.OverlayValues[171] = d171
						ps204.OverlayValues[172] = d172
						ps204.OverlayValues[173] = d173
						ps204.OverlayValues[174] = d174
						ps204.OverlayValues[175] = d175
						ps204.OverlayValues[176] = d176
						ps204.OverlayValues[177] = d177
						ps204.OverlayValues[178] = d178
						ps204.OverlayValues[179] = d179
						ps204.OverlayValues[180] = d180
						ps204.OverlayValues[181] = d181
						ps204.OverlayValues[182] = d182
						ps204.OverlayValues[183] = d183
						ps204.OverlayValues[184] = d184
						ps204.OverlayValues[185] = d185
						ps204.OverlayValues[186] = d186
						ps204.OverlayValues[187] = d187
						ps204.OverlayValues[188] = d188
						ps204.OverlayValues[189] = d189
						ps204.OverlayValues[191] = d191
						ps204.OverlayValues[192] = d192
						ps204.OverlayValues[193] = d193
						ps204.OverlayValues[194] = d194
						ps204.OverlayValues[195] = d195
						ps204.OverlayValues[196] = d196
						ps204.OverlayValues[197] = d197
						ps204.OverlayValues[198] = d198
						ps204.OverlayValues[200] = d200
						ps204.OverlayValues[201] = d201
						ps204.OverlayValues[202] = d202
						return bbs[8].RenderPS(ps204)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d202.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl9)
					ps205 := PhiState{General: true}
					ps205.OverlayValues = make([]JITValueDesc, 203)
					ps205.OverlayValues[8] = d8
					ps205.OverlayValues[9] = d9
					ps205.OverlayValues[10] = d10
					ps205.OverlayValues[11] = d11
					ps205.OverlayValues[12] = d12
					ps205.OverlayValues[13] = d13
					ps205.OverlayValues[14] = d14
					ps205.OverlayValues[15] = d15
					ps205.OverlayValues[16] = d16
					ps205.OverlayValues[17] = d17
					ps205.OverlayValues[18] = d18
					ps205.OverlayValues[19] = d19
					ps205.OverlayValues[20] = d20
					ps205.OverlayValues[21] = d21
					ps205.OverlayValues[22] = d22
					ps205.OverlayValues[25] = d25
					ps205.OverlayValues[28] = d28
					ps205.OverlayValues[47] = d47
					ps205.OverlayValues[48] = d48
					ps205.OverlayValues[49] = d49
					ps205.OverlayValues[50] = d50
					ps205.OverlayValues[51] = d51
					ps205.OverlayValues[53] = d53
					ps205.OverlayValues[54] = d54
					ps205.OverlayValues[55] = d55
					ps205.OverlayValues[56] = d56
					ps205.OverlayValues[57] = d57
					ps205.OverlayValues[58] = d58
					ps205.OverlayValues[59] = d59
					ps205.OverlayValues[62] = d62
					ps205.OverlayValues[97] = d97
					ps205.OverlayValues[98] = d98
					ps205.OverlayValues[99] = d99
					ps205.OverlayValues[100] = d100
					ps205.OverlayValues[101] = d101
					ps205.OverlayValues[102] = d102
					ps205.OverlayValues[104] = d104
					ps205.OverlayValues[105] = d105
					ps205.OverlayValues[106] = d106
					ps205.OverlayValues[107] = d107
					ps205.OverlayValues[108] = d108
					ps205.OverlayValues[109] = d109
					ps205.OverlayValues[110] = d110
					ps205.OverlayValues[111] = d111
					ps205.OverlayValues[112] = d112
					ps205.OverlayValues[115] = d115
					ps205.OverlayValues[116] = d116
					ps205.OverlayValues[117] = d117
					ps205.OverlayValues[118] = d118
					ps205.OverlayValues[171] = d171
					ps205.OverlayValues[172] = d172
					ps205.OverlayValues[173] = d173
					ps205.OverlayValues[174] = d174
					ps205.OverlayValues[175] = d175
					ps205.OverlayValues[176] = d176
					ps205.OverlayValues[177] = d177
					ps205.OverlayValues[178] = d178
					ps205.OverlayValues[179] = d179
					ps205.OverlayValues[180] = d180
					ps205.OverlayValues[181] = d181
					ps205.OverlayValues[182] = d182
					ps205.OverlayValues[183] = d183
					ps205.OverlayValues[184] = d184
					ps205.OverlayValues[185] = d185
					ps205.OverlayValues[186] = d186
					ps205.OverlayValues[187] = d187
					ps205.OverlayValues[188] = d188
					ps205.OverlayValues[189] = d189
					ps205.OverlayValues[191] = d191
					ps205.OverlayValues[192] = d192
					ps205.OverlayValues[193] = d193
					ps205.OverlayValues[194] = d194
					ps205.OverlayValues[195] = d195
					ps205.OverlayValues[196] = d196
					ps205.OverlayValues[197] = d197
					ps205.OverlayValues[198] = d198
					ps205.OverlayValues[200] = d200
					ps205.OverlayValues[201] = d201
					ps205.OverlayValues[202] = d202
					ps206 := PhiState{General: true}
					ps206.OverlayValues = make([]JITValueDesc, 203)
					ps206.OverlayValues[8] = d8
					ps206.OverlayValues[9] = d9
					ps206.OverlayValues[10] = d10
					ps206.OverlayValues[11] = d11
					ps206.OverlayValues[12] = d12
					ps206.OverlayValues[13] = d13
					ps206.OverlayValues[14] = d14
					ps206.OverlayValues[15] = d15
					ps206.OverlayValues[16] = d16
					ps206.OverlayValues[17] = d17
					ps206.OverlayValues[18] = d18
					ps206.OverlayValues[19] = d19
					ps206.OverlayValues[20] = d20
					ps206.OverlayValues[21] = d21
					ps206.OverlayValues[22] = d22
					ps206.OverlayValues[25] = d25
					ps206.OverlayValues[28] = d28
					ps206.OverlayValues[47] = d47
					ps206.OverlayValues[48] = d48
					ps206.OverlayValues[49] = d49
					ps206.OverlayValues[50] = d50
					ps206.OverlayValues[51] = d51
					ps206.OverlayValues[53] = d53
					ps206.OverlayValues[54] = d54
					ps206.OverlayValues[55] = d55
					ps206.OverlayValues[56] = d56
					ps206.OverlayValues[57] = d57
					ps206.OverlayValues[58] = d58
					ps206.OverlayValues[59] = d59
					ps206.OverlayValues[62] = d62
					ps206.OverlayValues[97] = d97
					ps206.OverlayValues[98] = d98
					ps206.OverlayValues[99] = d99
					ps206.OverlayValues[100] = d100
					ps206.OverlayValues[101] = d101
					ps206.OverlayValues[102] = d102
					ps206.OverlayValues[104] = d104
					ps206.OverlayValues[105] = d105
					ps206.OverlayValues[106] = d106
					ps206.OverlayValues[107] = d107
					ps206.OverlayValues[108] = d108
					ps206.OverlayValues[109] = d109
					ps206.OverlayValues[110] = d110
					ps206.OverlayValues[111] = d111
					ps206.OverlayValues[112] = d112
					ps206.OverlayValues[115] = d115
					ps206.OverlayValues[116] = d116
					ps206.OverlayValues[117] = d117
					ps206.OverlayValues[118] = d118
					ps206.OverlayValues[171] = d171
					ps206.OverlayValues[172] = d172
					ps206.OverlayValues[173] = d173
					ps206.OverlayValues[174] = d174
					ps206.OverlayValues[175] = d175
					ps206.OverlayValues[176] = d176
					ps206.OverlayValues[177] = d177
					ps206.OverlayValues[178] = d178
					ps206.OverlayValues[179] = d179
					ps206.OverlayValues[180] = d180
					ps206.OverlayValues[181] = d181
					ps206.OverlayValues[182] = d182
					ps206.OverlayValues[183] = d183
					ps206.OverlayValues[184] = d184
					ps206.OverlayValues[185] = d185
					ps206.OverlayValues[186] = d186
					ps206.OverlayValues[187] = d187
					ps206.OverlayValues[188] = d188
					ps206.OverlayValues[189] = d189
					ps206.OverlayValues[191] = d191
					ps206.OverlayValues[192] = d192
					ps206.OverlayValues[193] = d193
					ps206.OverlayValues[194] = d194
					ps206.OverlayValues[195] = d195
					ps206.OverlayValues[196] = d196
					ps206.OverlayValues[197] = d197
					ps206.OverlayValues[198] = d198
					ps206.OverlayValues[200] = d200
					ps206.OverlayValues[201] = d201
					ps206.OverlayValues[202] = d202
					snap207 := d8
					snap208 := d9
					snap209 := d10
					snap210 := d11
					snap211 := d12
					snap212 := d13
					snap213 := d14
					snap214 := d15
					snap215 := d16
					snap216 := d17
					snap217 := d18
					snap218 := d19
					snap219 := d20
					snap220 := d21
					snap221 := d22
					snap222 := d25
					snap223 := d28
					snap224 := d47
					snap225 := d48
					snap226 := d49
					snap227 := d50
					snap228 := d51
					snap229 := d53
					snap230 := d54
					snap231 := d55
					snap232 := d56
					snap233 := d57
					snap234 := d58
					snap235 := d59
					snap236 := d62
					snap237 := d97
					snap238 := d98
					snap239 := d99
					snap240 := d100
					snap241 := d101
					snap242 := d102
					snap243 := d104
					snap244 := d105
					snap245 := d106
					snap246 := d107
					snap247 := d108
					snap248 := d109
					snap249 := d110
					snap250 := d111
					snap251 := d112
					snap252 := d115
					snap253 := d116
					snap254 := d117
					snap255 := d118
					snap256 := d171
					snap257 := d172
					snap258 := d173
					snap259 := d174
					snap260 := d175
					snap261 := d176
					snap262 := d177
					snap263 := d178
					snap264 := d179
					snap265 := d180
					snap266 := d181
					snap267 := d182
					snap268 := d183
					snap269 := d184
					snap270 := d185
					snap271 := d186
					snap272 := d187
					snap273 := d188
					snap274 := d189
					snap275 := d191
					snap276 := d192
					snap277 := d193
					snap278 := d194
					snap279 := d195
					snap280 := d196
					snap281 := d197
					snap282 := d198
					snap283 := d200
					snap284 := d201
					snap285 := d202
					alloc286 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps206)
					}
					ctx.RestoreAllocState(alloc286)
					d8 = snap207
					d9 = snap208
					d10 = snap209
					d11 = snap210
					d12 = snap211
					d13 = snap212
					d14 = snap213
					d15 = snap214
					d16 = snap215
					d17 = snap216
					d18 = snap217
					d19 = snap218
					d20 = snap219
					d21 = snap220
					d22 = snap221
					d25 = snap222
					d28 = snap223
					d47 = snap224
					d48 = snap225
					d49 = snap226
					d50 = snap227
					d51 = snap228
					d53 = snap229
					d54 = snap230
					d55 = snap231
					d56 = snap232
					d57 = snap233
					d58 = snap234
					d59 = snap235
					d62 = snap236
					d97 = snap237
					d98 = snap238
					d99 = snap239
					d100 = snap240
					d101 = snap241
					d102 = snap242
					d104 = snap243
					d105 = snap244
					d106 = snap245
					d107 = snap246
					d108 = snap247
					d109 = snap248
					d110 = snap249
					d111 = snap250
					d112 = snap251
					d115 = snap252
					d116 = snap253
					d117 = snap254
					d118 = snap255
					d171 = snap256
					d172 = snap257
					d173 = snap258
					d174 = snap259
					d175 = snap260
					d176 = snap261
					d177 = snap262
					d178 = snap263
					d179 = snap264
					d180 = snap265
					d181 = snap266
					d182 = snap267
					d183 = snap268
					d184 = snap269
					d185 = snap270
					d186 = snap271
					d187 = snap272
					d188 = snap273
					d189 = snap274
					d191 = snap275
					d192 = snap276
					d193 = snap277
					d194 = snap278
					d195 = snap279
					d196 = snap280
					d197 = snap281
					d198 = snap282
					d200 = snap283
					d201 = snap284
					d202 = snap285
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps205)
					}
					return result
					ctx.FreeDesc(&d201)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d287 := ps.PhiValues[0]
							ctx.EnsureDesc(&d287)
							if phiHomeOK6 {
								ctx.EmitMovToReg(r4, d287)
							} else {
								ctx.EmitStoreToStack(d287, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d288 := ps.PhiValues[1]
							ctx.EnsureDesc(&d288)
							if phiHomeOK7 {
								ctx.EmitMovToReg(r5, d288)
							} else {
								ctx.EmitStoreToStack(d288, int32(bbs[10].PhiBase)+int32(16))
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
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
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d14 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d15 = ps.PhiValues[1]
					}
					if phiHomeOK6 && d14.Loc == LocReg {
						ctx.BindReg(r4, &d14)
					}
					if phiHomeOK7 && d15.Loc == LocReg {
						ctx.BindReg(r5, &d15)
					}
					ctx.ReclaimUntrackedRegs()
					var d289 JITValueDesc
					if d17.SliceSizeKnown {
						d289 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d289 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d289 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d289 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d289 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d289)
					ctx.EnsureDescsTogether(&d15, &d289)
					var d290 JITValueDesc
					if d15.Loc == LocImm && d289.Loc == LocImm {
						d290 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d289.Imm.Int())}
					} else if d289.Loc == LocImm {
						r20 := ctx.AllocRegExcept(d15.Reg)
						if d289.Imm.Int() >= -2147483648 && d289.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d289.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d289.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, RegR11)
						}
						ctx.EmitSetcc(r20, CondSignedLess)
						d290 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
						ctx.BindReg(r20, &d290)
					} else if d15.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d289.Reg)
						ctx.EmitSetcc(r21, CondSignedLess)
						d290 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
						ctx.BindReg(r21, &d290)
					} else {
						r22 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpInt64(d15.Reg, d289.Reg)
						ctx.EmitSetcc(r22, CondSignedLess)
						d290 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
						ctx.BindReg(r22, &d290)
					}
					ctx.FreeDesc(&d289)
					d291 = d290
					ctx.EnsureDesc(&d291)
					if d291.Loc != LocImm && d291.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d291.Loc == LocImm {
						if d291.Imm.Bool() {
							if ps.General {
							}
							ps292 := PhiState{General: ps.General}
							ps292.OverlayValues = make([]JITValueDesc, 292)
							ps292.OverlayValues[8] = d8
							ps292.OverlayValues[9] = d9
							ps292.OverlayValues[10] = d10
							ps292.OverlayValues[11] = d11
							ps292.OverlayValues[12] = d12
							ps292.OverlayValues[13] = d13
							ps292.OverlayValues[14] = d14
							ps292.OverlayValues[15] = d15
							ps292.OverlayValues[16] = d16
							ps292.OverlayValues[17] = d17
							ps292.OverlayValues[18] = d18
							ps292.OverlayValues[19] = d19
							ps292.OverlayValues[20] = d20
							ps292.OverlayValues[21] = d21
							ps292.OverlayValues[22] = d22
							ps292.OverlayValues[25] = d25
							ps292.OverlayValues[28] = d28
							ps292.OverlayValues[47] = d47
							ps292.OverlayValues[48] = d48
							ps292.OverlayValues[49] = d49
							ps292.OverlayValues[50] = d50
							ps292.OverlayValues[51] = d51
							ps292.OverlayValues[53] = d53
							ps292.OverlayValues[54] = d54
							ps292.OverlayValues[55] = d55
							ps292.OverlayValues[56] = d56
							ps292.OverlayValues[57] = d57
							ps292.OverlayValues[58] = d58
							ps292.OverlayValues[59] = d59
							ps292.OverlayValues[62] = d62
							ps292.OverlayValues[97] = d97
							ps292.OverlayValues[98] = d98
							ps292.OverlayValues[99] = d99
							ps292.OverlayValues[100] = d100
							ps292.OverlayValues[101] = d101
							ps292.OverlayValues[102] = d102
							ps292.OverlayValues[104] = d104
							ps292.OverlayValues[105] = d105
							ps292.OverlayValues[106] = d106
							ps292.OverlayValues[107] = d107
							ps292.OverlayValues[108] = d108
							ps292.OverlayValues[109] = d109
							ps292.OverlayValues[110] = d110
							ps292.OverlayValues[111] = d111
							ps292.OverlayValues[112] = d112
							ps292.OverlayValues[115] = d115
							ps292.OverlayValues[116] = d116
							ps292.OverlayValues[117] = d117
							ps292.OverlayValues[118] = d118
							ps292.OverlayValues[171] = d171
							ps292.OverlayValues[172] = d172
							ps292.OverlayValues[173] = d173
							ps292.OverlayValues[174] = d174
							ps292.OverlayValues[175] = d175
							ps292.OverlayValues[176] = d176
							ps292.OverlayValues[177] = d177
							ps292.OverlayValues[178] = d178
							ps292.OverlayValues[179] = d179
							ps292.OverlayValues[180] = d180
							ps292.OverlayValues[181] = d181
							ps292.OverlayValues[182] = d182
							ps292.OverlayValues[183] = d183
							ps292.OverlayValues[184] = d184
							ps292.OverlayValues[185] = d185
							ps292.OverlayValues[186] = d186
							ps292.OverlayValues[187] = d187
							ps292.OverlayValues[188] = d188
							ps292.OverlayValues[189] = d189
							ps292.OverlayValues[191] = d191
							ps292.OverlayValues[192] = d192
							ps292.OverlayValues[193] = d193
							ps292.OverlayValues[194] = d194
							ps292.OverlayValues[195] = d195
							ps292.OverlayValues[196] = d196
							ps292.OverlayValues[197] = d197
							ps292.OverlayValues[198] = d198
							ps292.OverlayValues[200] = d200
							ps292.OverlayValues[201] = d201
							ps292.OverlayValues[202] = d202
							ps292.OverlayValues[287] = d287
							ps292.OverlayValues[288] = d288
							ps292.OverlayValues[289] = d289
							ps292.OverlayValues[290] = d290
							ps292.OverlayValues[291] = d291
							return bbs[13].RenderPS(ps292)
						}
						if ps.General {
						}
						ps293 := PhiState{General: ps.General}
						ps293.OverlayValues = make([]JITValueDesc, 292)
						ps293.OverlayValues[8] = d8
						ps293.OverlayValues[9] = d9
						ps293.OverlayValues[10] = d10
						ps293.OverlayValues[11] = d11
						ps293.OverlayValues[12] = d12
						ps293.OverlayValues[13] = d13
						ps293.OverlayValues[14] = d14
						ps293.OverlayValues[15] = d15
						ps293.OverlayValues[16] = d16
						ps293.OverlayValues[17] = d17
						ps293.OverlayValues[18] = d18
						ps293.OverlayValues[19] = d19
						ps293.OverlayValues[20] = d20
						ps293.OverlayValues[21] = d21
						ps293.OverlayValues[22] = d22
						ps293.OverlayValues[25] = d25
						ps293.OverlayValues[28] = d28
						ps293.OverlayValues[47] = d47
						ps293.OverlayValues[48] = d48
						ps293.OverlayValues[49] = d49
						ps293.OverlayValues[50] = d50
						ps293.OverlayValues[51] = d51
						ps293.OverlayValues[53] = d53
						ps293.OverlayValues[54] = d54
						ps293.OverlayValues[55] = d55
						ps293.OverlayValues[56] = d56
						ps293.OverlayValues[57] = d57
						ps293.OverlayValues[58] = d58
						ps293.OverlayValues[59] = d59
						ps293.OverlayValues[62] = d62
						ps293.OverlayValues[97] = d97
						ps293.OverlayValues[98] = d98
						ps293.OverlayValues[99] = d99
						ps293.OverlayValues[100] = d100
						ps293.OverlayValues[101] = d101
						ps293.OverlayValues[102] = d102
						ps293.OverlayValues[104] = d104
						ps293.OverlayValues[105] = d105
						ps293.OverlayValues[106] = d106
						ps293.OverlayValues[107] = d107
						ps293.OverlayValues[108] = d108
						ps293.OverlayValues[109] = d109
						ps293.OverlayValues[110] = d110
						ps293.OverlayValues[111] = d111
						ps293.OverlayValues[112] = d112
						ps293.OverlayValues[115] = d115
						ps293.OverlayValues[116] = d116
						ps293.OverlayValues[117] = d117
						ps293.OverlayValues[118] = d118
						ps293.OverlayValues[171] = d171
						ps293.OverlayValues[172] = d172
						ps293.OverlayValues[173] = d173
						ps293.OverlayValues[174] = d174
						ps293.OverlayValues[175] = d175
						ps293.OverlayValues[176] = d176
						ps293.OverlayValues[177] = d177
						ps293.OverlayValues[178] = d178
						ps293.OverlayValues[179] = d179
						ps293.OverlayValues[180] = d180
						ps293.OverlayValues[181] = d181
						ps293.OverlayValues[182] = d182
						ps293.OverlayValues[183] = d183
						ps293.OverlayValues[184] = d184
						ps293.OverlayValues[185] = d185
						ps293.OverlayValues[186] = d186
						ps293.OverlayValues[187] = d187
						ps293.OverlayValues[188] = d188
						ps293.OverlayValues[189] = d189
						ps293.OverlayValues[191] = d191
						ps293.OverlayValues[192] = d192
						ps293.OverlayValues[193] = d193
						ps293.OverlayValues[194] = d194
						ps293.OverlayValues[195] = d195
						ps293.OverlayValues[196] = d196
						ps293.OverlayValues[197] = d197
						ps293.OverlayValues[198] = d198
						ps293.OverlayValues[200] = d200
						ps293.OverlayValues[201] = d201
						ps293.OverlayValues[202] = d202
						ps293.OverlayValues[287] = d287
						ps293.OverlayValues[288] = d288
						ps293.OverlayValues[289] = d289
						ps293.OverlayValues[290] = d290
						ps293.OverlayValues[291] = d291
						return bbs[12].RenderPS(ps293)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d294 := ps.PhiValues[0]
							ctx.EnsureDesc(&d294)
							if phiHomeOK6 {
								ctx.EmitMovToReg(r4, d294)
							} else {
								ctx.EmitStoreToStack(d294, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d295 := ps.PhiValues[1]
							ctx.EnsureDesc(&d295)
							if phiHomeOK7 {
								ctx.EmitMovToReg(r5, d295)
							} else {
								ctx.EmitStoreToStack(d295, int32(bbs[10].PhiBase)+int32(16))
							}
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d291.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl26)
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl14)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl13)
					ps296 := PhiState{General: true}
					ps296.OverlayValues = make([]JITValueDesc, 296)
					ps296.OverlayValues[8] = d8
					ps296.OverlayValues[9] = d9
					ps296.OverlayValues[10] = d10
					ps296.OverlayValues[11] = d11
					ps296.OverlayValues[12] = d12
					ps296.OverlayValues[13] = d13
					ps296.OverlayValues[14] = d14
					ps296.OverlayValues[15] = d15
					ps296.OverlayValues[16] = d16
					ps296.OverlayValues[17] = d17
					ps296.OverlayValues[18] = d18
					ps296.OverlayValues[19] = d19
					ps296.OverlayValues[20] = d20
					ps296.OverlayValues[21] = d21
					ps296.OverlayValues[22] = d22
					ps296.OverlayValues[25] = d25
					ps296.OverlayValues[28] = d28
					ps296.OverlayValues[47] = d47
					ps296.OverlayValues[48] = d48
					ps296.OverlayValues[49] = d49
					ps296.OverlayValues[50] = d50
					ps296.OverlayValues[51] = d51
					ps296.OverlayValues[53] = d53
					ps296.OverlayValues[54] = d54
					ps296.OverlayValues[55] = d55
					ps296.OverlayValues[56] = d56
					ps296.OverlayValues[57] = d57
					ps296.OverlayValues[58] = d58
					ps296.OverlayValues[59] = d59
					ps296.OverlayValues[62] = d62
					ps296.OverlayValues[97] = d97
					ps296.OverlayValues[98] = d98
					ps296.OverlayValues[99] = d99
					ps296.OverlayValues[100] = d100
					ps296.OverlayValues[101] = d101
					ps296.OverlayValues[102] = d102
					ps296.OverlayValues[104] = d104
					ps296.OverlayValues[105] = d105
					ps296.OverlayValues[106] = d106
					ps296.OverlayValues[107] = d107
					ps296.OverlayValues[108] = d108
					ps296.OverlayValues[109] = d109
					ps296.OverlayValues[110] = d110
					ps296.OverlayValues[111] = d111
					ps296.OverlayValues[112] = d112
					ps296.OverlayValues[115] = d115
					ps296.OverlayValues[116] = d116
					ps296.OverlayValues[117] = d117
					ps296.OverlayValues[118] = d118
					ps296.OverlayValues[171] = d171
					ps296.OverlayValues[172] = d172
					ps296.OverlayValues[173] = d173
					ps296.OverlayValues[174] = d174
					ps296.OverlayValues[175] = d175
					ps296.OverlayValues[176] = d176
					ps296.OverlayValues[177] = d177
					ps296.OverlayValues[178] = d178
					ps296.OverlayValues[179] = d179
					ps296.OverlayValues[180] = d180
					ps296.OverlayValues[181] = d181
					ps296.OverlayValues[182] = d182
					ps296.OverlayValues[183] = d183
					ps296.OverlayValues[184] = d184
					ps296.OverlayValues[185] = d185
					ps296.OverlayValues[186] = d186
					ps296.OverlayValues[187] = d187
					ps296.OverlayValues[188] = d188
					ps296.OverlayValues[189] = d189
					ps296.OverlayValues[191] = d191
					ps296.OverlayValues[192] = d192
					ps296.OverlayValues[193] = d193
					ps296.OverlayValues[194] = d194
					ps296.OverlayValues[195] = d195
					ps296.OverlayValues[196] = d196
					ps296.OverlayValues[197] = d197
					ps296.OverlayValues[198] = d198
					ps296.OverlayValues[200] = d200
					ps296.OverlayValues[201] = d201
					ps296.OverlayValues[202] = d202
					ps296.OverlayValues[287] = d287
					ps296.OverlayValues[288] = d288
					ps296.OverlayValues[289] = d289
					ps296.OverlayValues[290] = d290
					ps296.OverlayValues[291] = d291
					ps296.OverlayValues[294] = d294
					ps296.OverlayValues[295] = d295
					ps297 := PhiState{General: true}
					ps297.OverlayValues = make([]JITValueDesc, 296)
					ps297.OverlayValues[8] = d8
					ps297.OverlayValues[9] = d9
					ps297.OverlayValues[10] = d10
					ps297.OverlayValues[11] = d11
					ps297.OverlayValues[12] = d12
					ps297.OverlayValues[13] = d13
					ps297.OverlayValues[14] = d14
					ps297.OverlayValues[15] = d15
					ps297.OverlayValues[16] = d16
					ps297.OverlayValues[17] = d17
					ps297.OverlayValues[18] = d18
					ps297.OverlayValues[19] = d19
					ps297.OverlayValues[20] = d20
					ps297.OverlayValues[21] = d21
					ps297.OverlayValues[22] = d22
					ps297.OverlayValues[25] = d25
					ps297.OverlayValues[28] = d28
					ps297.OverlayValues[47] = d47
					ps297.OverlayValues[48] = d48
					ps297.OverlayValues[49] = d49
					ps297.OverlayValues[50] = d50
					ps297.OverlayValues[51] = d51
					ps297.OverlayValues[53] = d53
					ps297.OverlayValues[54] = d54
					ps297.OverlayValues[55] = d55
					ps297.OverlayValues[56] = d56
					ps297.OverlayValues[57] = d57
					ps297.OverlayValues[58] = d58
					ps297.OverlayValues[59] = d59
					ps297.OverlayValues[62] = d62
					ps297.OverlayValues[97] = d97
					ps297.OverlayValues[98] = d98
					ps297.OverlayValues[99] = d99
					ps297.OverlayValues[100] = d100
					ps297.OverlayValues[101] = d101
					ps297.OverlayValues[102] = d102
					ps297.OverlayValues[104] = d104
					ps297.OverlayValues[105] = d105
					ps297.OverlayValues[106] = d106
					ps297.OverlayValues[107] = d107
					ps297.OverlayValues[108] = d108
					ps297.OverlayValues[109] = d109
					ps297.OverlayValues[110] = d110
					ps297.OverlayValues[111] = d111
					ps297.OverlayValues[112] = d112
					ps297.OverlayValues[115] = d115
					ps297.OverlayValues[116] = d116
					ps297.OverlayValues[117] = d117
					ps297.OverlayValues[118] = d118
					ps297.OverlayValues[171] = d171
					ps297.OverlayValues[172] = d172
					ps297.OverlayValues[173] = d173
					ps297.OverlayValues[174] = d174
					ps297.OverlayValues[175] = d175
					ps297.OverlayValues[176] = d176
					ps297.OverlayValues[177] = d177
					ps297.OverlayValues[178] = d178
					ps297.OverlayValues[179] = d179
					ps297.OverlayValues[180] = d180
					ps297.OverlayValues[181] = d181
					ps297.OverlayValues[182] = d182
					ps297.OverlayValues[183] = d183
					ps297.OverlayValues[184] = d184
					ps297.OverlayValues[185] = d185
					ps297.OverlayValues[186] = d186
					ps297.OverlayValues[187] = d187
					ps297.OverlayValues[188] = d188
					ps297.OverlayValues[189] = d189
					ps297.OverlayValues[191] = d191
					ps297.OverlayValues[192] = d192
					ps297.OverlayValues[193] = d193
					ps297.OverlayValues[194] = d194
					ps297.OverlayValues[195] = d195
					ps297.OverlayValues[196] = d196
					ps297.OverlayValues[197] = d197
					ps297.OverlayValues[198] = d198
					ps297.OverlayValues[200] = d200
					ps297.OverlayValues[201] = d201
					ps297.OverlayValues[202] = d202
					ps297.OverlayValues[287] = d287
					ps297.OverlayValues[288] = d288
					ps297.OverlayValues[289] = d289
					ps297.OverlayValues[290] = d290
					ps297.OverlayValues[291] = d291
					ps297.OverlayValues[294] = d294
					ps297.OverlayValues[295] = d295
					snap298 := d8
					snap299 := d9
					snap300 := d10
					snap301 := d11
					snap302 := d12
					snap303 := d13
					snap304 := d14
					snap305 := d15
					snap306 := d16
					snap307 := d17
					snap308 := d18
					snap309 := d19
					snap310 := d20
					snap311 := d21
					snap312 := d22
					snap313 := d25
					snap314 := d28
					snap315 := d47
					snap316 := d48
					snap317 := d49
					snap318 := d50
					snap319 := d51
					snap320 := d53
					snap321 := d54
					snap322 := d55
					snap323 := d56
					snap324 := d57
					snap325 := d58
					snap326 := d59
					snap327 := d62
					snap328 := d97
					snap329 := d98
					snap330 := d99
					snap331 := d100
					snap332 := d101
					snap333 := d102
					snap334 := d104
					snap335 := d105
					snap336 := d106
					snap337 := d107
					snap338 := d108
					snap339 := d109
					snap340 := d110
					snap341 := d111
					snap342 := d112
					snap343 := d115
					snap344 := d116
					snap345 := d117
					snap346 := d118
					snap347 := d171
					snap348 := d172
					snap349 := d173
					snap350 := d174
					snap351 := d175
					snap352 := d176
					snap353 := d177
					snap354 := d178
					snap355 := d179
					snap356 := d180
					snap357 := d181
					snap358 := d182
					snap359 := d183
					snap360 := d184
					snap361 := d185
					snap362 := d186
					snap363 := d187
					snap364 := d188
					snap365 := d189
					snap366 := d191
					snap367 := d192
					snap368 := d193
					snap369 := d194
					snap370 := d195
					snap371 := d196
					snap372 := d197
					snap373 := d198
					snap374 := d200
					snap375 := d201
					snap376 := d202
					snap377 := d287
					snap378 := d288
					snap379 := d289
					snap380 := d290
					snap381 := d291
					snap382 := d294
					snap383 := d295
					alloc384 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps297)
					}
					ctx.RestoreAllocState(alloc384)
					d8 = snap298
					d9 = snap299
					d10 = snap300
					d11 = snap301
					d12 = snap302
					d13 = snap303
					d14 = snap304
					d15 = snap305
					d16 = snap306
					d17 = snap307
					d18 = snap308
					d19 = snap309
					d20 = snap310
					d21 = snap311
					d22 = snap312
					d25 = snap313
					d28 = snap314
					d47 = snap315
					d48 = snap316
					d49 = snap317
					d50 = snap318
					d51 = snap319
					d53 = snap320
					d54 = snap321
					d55 = snap322
					d56 = snap323
					d57 = snap324
					d58 = snap325
					d59 = snap326
					d62 = snap327
					d97 = snap328
					d98 = snap329
					d99 = snap330
					d100 = snap331
					d101 = snap332
					d102 = snap333
					d104 = snap334
					d105 = snap335
					d106 = snap336
					d107 = snap337
					d108 = snap338
					d109 = snap339
					d110 = snap340
					d111 = snap341
					d112 = snap342
					d115 = snap343
					d116 = snap344
					d117 = snap345
					d118 = snap346
					d171 = snap347
					d172 = snap348
					d173 = snap349
					d174 = snap350
					d175 = snap351
					d176 = snap352
					d177 = snap353
					d178 = snap354
					d179 = snap355
					d180 = snap356
					d181 = snap357
					d182 = snap358
					d183 = snap359
					d184 = snap360
					d185 = snap361
					d186 = snap362
					d187 = snap363
					d188 = snap364
					d189 = snap365
					d191 = snap366
					d192 = snap367
					d193 = snap368
					d194 = snap369
					d195 = snap370
					d196 = snap371
					d197 = snap372
					d198 = snap373
					d200 = snap374
					d201 = snap375
					d202 = snap376
					d287 = snap377
					d288 = snap378
					d289 = snap379
					d290 = snap380
					d291 = snap381
					d294 = snap382
					d295 = snap383
					if !bbs[13].Rendered {
						return bbs[13].RenderPS(ps296)
					}
					return result
					ctx.FreeDesc(&d290)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
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
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					d386 = ctx.EmitSliceElementAddress(&d17, &d15, 16)
					ctx.EnsureDesc(&d386)
					r23 := ctx.AllocRegExcept(d386.Reg)
					ctx.EmitMovRegMem(r23, d386.Reg, 8)
					ctx.EmitMovRegMem(d386.Reg, d386.Reg, 0)
					d385 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d386.Reg, Reg2: r23}
					ctx.BindReg(d386.Reg, &d385)
					ctx.BindReg(r23, &d385)
					ctx.EnsureDesc(&d385)
					d387 = d385
					_ = d387
					ctx.StabilizeDescForControlFlow(&d387)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl28 := ctx.ReserveLabel()
					_ = lbl28
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl28)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d388 JITValueDesc
					if d387.Loc == LocImm {
						d388 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d387.Imm.Float())}
					} else if d387.Type == tagFloat && d387.Loc == LocReg {
						d388 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d387.Reg}
						ctx.BindReg(d387.Reg, &d388)
						ctx.BindReg(d387.Reg, &d388)
					} else if d387.Type == tagFloat && d387.Loc == LocRegPair {
						ctx.FreeReg(d387.Reg)
						d388 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d387.Reg2}
						ctx.BindReg(d387.Reg2, &d388)
						ctx.BindReg(d387.Reg2, &d388)
					} else {
						d388 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d387}, 1)
						d388.Type = tagFloat
						ctx.BindReg(d388.Reg, &d388)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d388)
					ctx.FreeDesc(&d385)
					ctx.EnsureDesc(&d15)
					d390 = ctx.EmitSliceElementAddress(&d19, &d15, 16)
					ctx.EnsureDesc(&d390)
					r24 := ctx.AllocRegExcept(d390.Reg)
					ctx.EmitMovRegMem(r24, d390.Reg, 8)
					ctx.EmitMovRegMem(d390.Reg, d390.Reg, 0)
					d389 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d390.Reg, Reg2: r24}
					ctx.BindReg(d390.Reg, &d389)
					ctx.BindReg(r24, &d389)
					ctx.EnsureDesc(&d389)
					d391 = d389
					_ = d391
					ctx.StabilizeDescForControlFlow(&d391)
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl29 := ctx.ReserveLabel()
					_ = lbl29
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl29)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d392 JITValueDesc
					if d391.Loc == LocImm {
						d392 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d391.Imm.Float())}
					} else if d391.Type == tagFloat && d391.Loc == LocReg {
						d392 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d391.Reg}
						ctx.BindReg(d391.Reg, &d392)
						ctx.BindReg(d391.Reg, &d392)
					} else if d391.Type == tagFloat && d391.Loc == LocRegPair {
						ctx.FreeReg(d391.Reg)
						d392 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d391.Reg2}
						ctx.BindReg(d391.Reg2, &d392)
						ctx.BindReg(d391.Reg2, &d392)
					} else {
						d392 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d391}, 1)
						d392.Type = tagFloat
						ctx.BindReg(d392.Reg, &d392)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d392)
					ctx.FreeDesc(&d389)
					ctx.EnsureDesc(&d388)
					ctx.EnsureDesc(&d392)
					ctx.EnsureDescsTogether(&d388, &d392)
					var d393 JITValueDesc
					if d388.Loc == LocImm && d392.Loc == LocImm {
						d393 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d388.Imm.Float() * d392.Imm.Float())}
					} else if d388.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d392.Reg)
						_, xBits := d388.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d392.Reg)
						d393 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d393)
					} else if d392.Loc == LocImm {
						_, yBits := d392.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d388.Reg, RegR11)
						d393 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d388.Reg}
						ctx.BindReg(d388.Reg, &d393)
					} else {
						ctx.EmitMulFloat64(d388.Reg, d392.Reg)
						d393 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d388.Reg}
						ctx.BindReg(d388.Reg, &d393)
					}
					if d393.Loc == LocReg && d388.Loc == LocReg && d393.Reg == d388.Reg {
						ctx.TransferReg(d388.Reg)
						d388.Loc = LocNone
					}
					ctx.FreeDesc(&d388)
					ctx.FreeDesc(&d392)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d393)
					ctx.EnsureDescsTogether(&d14, &d393)
					var d394 JITValueDesc
					if d14.Loc == LocImm && d393.Loc == LocImm {
						d394 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d14.Imm.Float() + d393.Imm.Float())}
					} else if d14.Loc == LocImm {
						var scratch Reg
						if phiHomeOK6 && r4 != d393.Reg {
							scratch = r4
						} else {
							scratch = ctx.AllocRegExcept(d393.Reg)
						}
						_, xBits := d14.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d393.Reg)
						d394 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d394)
					} else if d393.Loc == LocImm {
						var scratch Reg
						if phiHomeOK6 {
							scratch = r4
						} else {
							scratch = ctx.AllocRegExcept(d14.Reg)
						}
						ctx.EmitMovRegReg(scratch, d14.Reg)
						_, yBits := d393.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d394 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d394)
					} else {
						var r25 Reg
						if phiHomeOK6 && r4 != d393.Reg {
							r25 = r4
						} else {
							r25 = ctx.AllocRegExcept(d14.Reg, d393.Reg)
						}
						ctx.EmitMovRegReg(r25, d14.Reg)
						ctx.EmitAddFloat64(r25, d393.Reg)
						d394 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r25}
						ctx.BindReg(r25, &d394)
					}
					if d394.Loc == LocReg && d14.Loc == LocReg && d394.Reg == d14.Reg {
						ctx.TransferReg(d14.Reg)
						d14.Loc = LocNone
					}
					ctx.FreeDesc(&d393)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					var d395 JITValueDesc
					if d15.Loc == LocImm {
						d395 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK7 {
							scratch = r5
						} else {
							scratch = ctx.AllocRegExcept(d15.Reg)
						}
						ctx.EmitMovRegReg(scratch, d15.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d395 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d395)
					}
					if d395.Loc == LocReg && d15.Loc == LocReg && d395.Reg == d15.Reg {
						ctx.TransferReg(d15.Reg)
						d15.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d394)
						if d394.Loc == LocReg {
							ctx.ProtectReg(d394.Reg)
						} else if d394.Loc == LocRegPair {
							ctx.ProtectReg(d394.Reg)
							ctx.ProtectReg(d394.Reg2)
						}
						ctx.SyncDesc(&d395)
						if d395.Loc == LocReg {
							ctx.ProtectReg(d395.Reg)
						} else if d395.Loc == LocRegPair {
							ctx.ProtectReg(d395.Reg)
							ctx.ProtectReg(d395.Reg2)
						}
						d396 = d394
						if d396.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d396)
						if phiHomeOK6 {
							ctx.EmitMovToReg(r4, d396)
						} else {
							ctx.EmitStoreToStack(d396, int32(bbs[10].PhiBase)+int32(0))
						}
						d397 = d395
						if d397.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d397)
						if phiHomeOK7 {
							ctx.EmitMovToReg(r5, d397)
						} else {
							ctx.EmitStoreToStack(d397, int32(bbs[10].PhiBase)+int32(16))
						}
						if d394.Loc == LocReg {
							ctx.UnprotectReg(d394.Reg)
						} else if d394.Loc == LocRegPair {
							ctx.UnprotectReg(d394.Reg)
							ctx.UnprotectReg(d394.Reg2)
						}
						if d395.Loc == LocReg {
							ctx.UnprotectReg(d395.Reg)
						} else if d395.Loc == LocRegPair {
							ctx.UnprotectReg(d395.Reg)
							ctx.UnprotectReg(d395.Reg2)
						}
					}
					ps398 := PhiState{General: ps.General}
					ps398.OverlayValues = make([]JITValueDesc, 398)
					ps398.OverlayValues[8] = d8
					ps398.OverlayValues[9] = d9
					ps398.OverlayValues[10] = d10
					ps398.OverlayValues[11] = d11
					ps398.OverlayValues[12] = d12
					ps398.OverlayValues[13] = d13
					ps398.OverlayValues[14] = d14
					ps398.OverlayValues[15] = d15
					ps398.OverlayValues[16] = d16
					ps398.OverlayValues[17] = d17
					ps398.OverlayValues[18] = d18
					ps398.OverlayValues[19] = d19
					ps398.OverlayValues[20] = d20
					ps398.OverlayValues[21] = d21
					ps398.OverlayValues[22] = d22
					ps398.OverlayValues[25] = d25
					ps398.OverlayValues[28] = d28
					ps398.OverlayValues[47] = d47
					ps398.OverlayValues[48] = d48
					ps398.OverlayValues[49] = d49
					ps398.OverlayValues[50] = d50
					ps398.OverlayValues[51] = d51
					ps398.OverlayValues[53] = d53
					ps398.OverlayValues[54] = d54
					ps398.OverlayValues[55] = d55
					ps398.OverlayValues[56] = d56
					ps398.OverlayValues[57] = d57
					ps398.OverlayValues[58] = d58
					ps398.OverlayValues[59] = d59
					ps398.OverlayValues[62] = d62
					ps398.OverlayValues[97] = d97
					ps398.OverlayValues[98] = d98
					ps398.OverlayValues[99] = d99
					ps398.OverlayValues[100] = d100
					ps398.OverlayValues[101] = d101
					ps398.OverlayValues[102] = d102
					ps398.OverlayValues[104] = d104
					ps398.OverlayValues[105] = d105
					ps398.OverlayValues[106] = d106
					ps398.OverlayValues[107] = d107
					ps398.OverlayValues[108] = d108
					ps398.OverlayValues[109] = d109
					ps398.OverlayValues[110] = d110
					ps398.OverlayValues[111] = d111
					ps398.OverlayValues[112] = d112
					ps398.OverlayValues[115] = d115
					ps398.OverlayValues[116] = d116
					ps398.OverlayValues[117] = d117
					ps398.OverlayValues[118] = d118
					ps398.OverlayValues[171] = d171
					ps398.OverlayValues[172] = d172
					ps398.OverlayValues[173] = d173
					ps398.OverlayValues[174] = d174
					ps398.OverlayValues[175] = d175
					ps398.OverlayValues[176] = d176
					ps398.OverlayValues[177] = d177
					ps398.OverlayValues[178] = d178
					ps398.OverlayValues[179] = d179
					ps398.OverlayValues[180] = d180
					ps398.OverlayValues[181] = d181
					ps398.OverlayValues[182] = d182
					ps398.OverlayValues[183] = d183
					ps398.OverlayValues[184] = d184
					ps398.OverlayValues[185] = d185
					ps398.OverlayValues[186] = d186
					ps398.OverlayValues[187] = d187
					ps398.OverlayValues[188] = d188
					ps398.OverlayValues[189] = d189
					ps398.OverlayValues[191] = d191
					ps398.OverlayValues[192] = d192
					ps398.OverlayValues[193] = d193
					ps398.OverlayValues[194] = d194
					ps398.OverlayValues[195] = d195
					ps398.OverlayValues[196] = d196
					ps398.OverlayValues[197] = d197
					ps398.OverlayValues[198] = d198
					ps398.OverlayValues[200] = d200
					ps398.OverlayValues[201] = d201
					ps398.OverlayValues[202] = d202
					ps398.OverlayValues[287] = d287
					ps398.OverlayValues[288] = d288
					ps398.OverlayValues[289] = d289
					ps398.OverlayValues[290] = d290
					ps398.OverlayValues[291] = d291
					ps398.OverlayValues[294] = d294
					ps398.OverlayValues[295] = d295
					ps398.OverlayValues[385] = d385
					ps398.OverlayValues[386] = d386
					ps398.OverlayValues[387] = d387
					ps398.OverlayValues[388] = d388
					ps398.OverlayValues[389] = d389
					ps398.OverlayValues[390] = d390
					ps398.OverlayValues[391] = d391
					ps398.OverlayValues[392] = d392
					ps398.OverlayValues[393] = d393
					ps398.OverlayValues[394] = d394
					ps398.OverlayValues[395] = d395
					ps398.OverlayValues[396] = d396
					ps398.OverlayValues[397] = d397
					ps398.PhiValues = make([]JITValueDesc, 2)
					d399 = d394
					ps398.PhiValues[0] = d399
					d400 = d395
					ps398.PhiValues[1] = d400
					if ps398.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps398)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
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
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != LocNone {
						d390 = ps.OverlayValues[390]
					}
					if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != LocNone {
						d391 = ps.OverlayValues[391]
					}
					if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != LocNone {
						d392 = ps.OverlayValues[392]
					}
					if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != LocNone {
						d393 = ps.OverlayValues[393]
					}
					if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != LocNone {
						d394 = ps.OverlayValues[394]
					}
					if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != LocNone {
						d395 = ps.OverlayValues[395]
					}
					if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != LocNone {
						d396 = ps.OverlayValues[396]
					}
					if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != LocNone {
						d397 = ps.OverlayValues[397]
					}
					if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != LocNone {
						d399 = ps.OverlayValues[399]
					}
					if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != LocNone {
						d400 = ps.OverlayValues[400]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					var d401 JITValueDesc
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d401 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d401.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d401.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d401.Reg, &d401)
						ctx.BindReg(d401.Reg2, &d401)
					} else {
						d401 = d8
					}
					d402 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("EUCLIDEAN")}
					var d403 JITValueDesc
					if d402.Loc == LocImm {
						ctx.TrackImm(d402.Imm)
						ptrWord, _ := d402.Imm.RawWords()
						d403 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d403.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d403.Reg2, uint64(len(d402.Imm.String())))
						ctx.BindReg(d403.Reg, &d403)
						ctx.BindReg(d403.Reg2, &d403)
					} else {
						d403 = d402
					}
					d404 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d401, d403}, 1)
					ctx.EmitAndRegImm32(d404.Reg, 1)
					d404.Type = tagBool
					ctx.BindReg(d404.Reg, &d404)
					ctx.FreeDesc(&d8)
					d405 = d404
					ctx.EnsureDesc(&d405)
					if d405.Loc != LocImm && d405.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d405.Loc == LocImm {
						if d405.Imm.Bool() {
							if ps.General {
							}
							ps406 := PhiState{General: ps.General}
							ps406.OverlayValues = make([]JITValueDesc, 406)
							ps406.OverlayValues[8] = d8
							ps406.OverlayValues[9] = d9
							ps406.OverlayValues[10] = d10
							ps406.OverlayValues[11] = d11
							ps406.OverlayValues[12] = d12
							ps406.OverlayValues[13] = d13
							ps406.OverlayValues[14] = d14
							ps406.OverlayValues[15] = d15
							ps406.OverlayValues[16] = d16
							ps406.OverlayValues[17] = d17
							ps406.OverlayValues[18] = d18
							ps406.OverlayValues[19] = d19
							ps406.OverlayValues[20] = d20
							ps406.OverlayValues[21] = d21
							ps406.OverlayValues[22] = d22
							ps406.OverlayValues[25] = d25
							ps406.OverlayValues[28] = d28
							ps406.OverlayValues[47] = d47
							ps406.OverlayValues[48] = d48
							ps406.OverlayValues[49] = d49
							ps406.OverlayValues[50] = d50
							ps406.OverlayValues[51] = d51
							ps406.OverlayValues[53] = d53
							ps406.OverlayValues[54] = d54
							ps406.OverlayValues[55] = d55
							ps406.OverlayValues[56] = d56
							ps406.OverlayValues[57] = d57
							ps406.OverlayValues[58] = d58
							ps406.OverlayValues[59] = d59
							ps406.OverlayValues[62] = d62
							ps406.OverlayValues[97] = d97
							ps406.OverlayValues[98] = d98
							ps406.OverlayValues[99] = d99
							ps406.OverlayValues[100] = d100
							ps406.OverlayValues[101] = d101
							ps406.OverlayValues[102] = d102
							ps406.OverlayValues[104] = d104
							ps406.OverlayValues[105] = d105
							ps406.OverlayValues[106] = d106
							ps406.OverlayValues[107] = d107
							ps406.OverlayValues[108] = d108
							ps406.OverlayValues[109] = d109
							ps406.OverlayValues[110] = d110
							ps406.OverlayValues[111] = d111
							ps406.OverlayValues[112] = d112
							ps406.OverlayValues[115] = d115
							ps406.OverlayValues[116] = d116
							ps406.OverlayValues[117] = d117
							ps406.OverlayValues[118] = d118
							ps406.OverlayValues[171] = d171
							ps406.OverlayValues[172] = d172
							ps406.OverlayValues[173] = d173
							ps406.OverlayValues[174] = d174
							ps406.OverlayValues[175] = d175
							ps406.OverlayValues[176] = d176
							ps406.OverlayValues[177] = d177
							ps406.OverlayValues[178] = d178
							ps406.OverlayValues[179] = d179
							ps406.OverlayValues[180] = d180
							ps406.OverlayValues[181] = d181
							ps406.OverlayValues[182] = d182
							ps406.OverlayValues[183] = d183
							ps406.OverlayValues[184] = d184
							ps406.OverlayValues[185] = d185
							ps406.OverlayValues[186] = d186
							ps406.OverlayValues[187] = d187
							ps406.OverlayValues[188] = d188
							ps406.OverlayValues[189] = d189
							ps406.OverlayValues[191] = d191
							ps406.OverlayValues[192] = d192
							ps406.OverlayValues[193] = d193
							ps406.OverlayValues[194] = d194
							ps406.OverlayValues[195] = d195
							ps406.OverlayValues[196] = d196
							ps406.OverlayValues[197] = d197
							ps406.OverlayValues[198] = d198
							ps406.OverlayValues[200] = d200
							ps406.OverlayValues[201] = d201
							ps406.OverlayValues[202] = d202
							ps406.OverlayValues[287] = d287
							ps406.OverlayValues[288] = d288
							ps406.OverlayValues[289] = d289
							ps406.OverlayValues[290] = d290
							ps406.OverlayValues[291] = d291
							ps406.OverlayValues[294] = d294
							ps406.OverlayValues[295] = d295
							ps406.OverlayValues[385] = d385
							ps406.OverlayValues[386] = d386
							ps406.OverlayValues[387] = d387
							ps406.OverlayValues[388] = d388
							ps406.OverlayValues[389] = d389
							ps406.OverlayValues[390] = d390
							ps406.OverlayValues[391] = d391
							ps406.OverlayValues[392] = d392
							ps406.OverlayValues[393] = d393
							ps406.OverlayValues[394] = d394
							ps406.OverlayValues[395] = d395
							ps406.OverlayValues[396] = d396
							ps406.OverlayValues[397] = d397
							ps406.OverlayValues[399] = d399
							ps406.OverlayValues[400] = d400
							ps406.OverlayValues[401] = d401
							ps406.OverlayValues[402] = d402
							ps406.OverlayValues[403] = d403
							ps406.OverlayValues[404] = d404
							ps406.OverlayValues[405] = d405
							return bbs[14].RenderPS(ps406)
						}
						if ps.General {
							ctx.SyncDesc(&d14)
							if d14.Loc == LocReg {
								ctx.ProtectReg(d14.Reg)
							} else if d14.Loc == LocRegPair {
								ctx.ProtectReg(d14.Reg)
								ctx.ProtectReg(d14.Reg2)
							}
							d407 = d14
							if d407.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d407)
							ctx.EmitStoreToStack(d407, int32(bbs[4].PhiBase)+int32(0))
							if d14.Loc == LocReg {
								ctx.UnprotectReg(d14.Reg)
							} else if d14.Loc == LocRegPair {
								ctx.UnprotectReg(d14.Reg)
								ctx.UnprotectReg(d14.Reg2)
							}
						}
						ps408 := PhiState{General: ps.General}
						ps408.OverlayValues = make([]JITValueDesc, 408)
						ps408.OverlayValues[8] = d8
						ps408.OverlayValues[9] = d9
						ps408.OverlayValues[10] = d10
						ps408.OverlayValues[11] = d11
						ps408.OverlayValues[12] = d12
						ps408.OverlayValues[13] = d13
						ps408.OverlayValues[14] = d14
						ps408.OverlayValues[15] = d15
						ps408.OverlayValues[16] = d16
						ps408.OverlayValues[17] = d17
						ps408.OverlayValues[18] = d18
						ps408.OverlayValues[19] = d19
						ps408.OverlayValues[20] = d20
						ps408.OverlayValues[21] = d21
						ps408.OverlayValues[22] = d22
						ps408.OverlayValues[25] = d25
						ps408.OverlayValues[28] = d28
						ps408.OverlayValues[47] = d47
						ps408.OverlayValues[48] = d48
						ps408.OverlayValues[49] = d49
						ps408.OverlayValues[50] = d50
						ps408.OverlayValues[51] = d51
						ps408.OverlayValues[53] = d53
						ps408.OverlayValues[54] = d54
						ps408.OverlayValues[55] = d55
						ps408.OverlayValues[56] = d56
						ps408.OverlayValues[57] = d57
						ps408.OverlayValues[58] = d58
						ps408.OverlayValues[59] = d59
						ps408.OverlayValues[62] = d62
						ps408.OverlayValues[97] = d97
						ps408.OverlayValues[98] = d98
						ps408.OverlayValues[99] = d99
						ps408.OverlayValues[100] = d100
						ps408.OverlayValues[101] = d101
						ps408.OverlayValues[102] = d102
						ps408.OverlayValues[104] = d104
						ps408.OverlayValues[105] = d105
						ps408.OverlayValues[106] = d106
						ps408.OverlayValues[107] = d107
						ps408.OverlayValues[108] = d108
						ps408.OverlayValues[109] = d109
						ps408.OverlayValues[110] = d110
						ps408.OverlayValues[111] = d111
						ps408.OverlayValues[112] = d112
						ps408.OverlayValues[115] = d115
						ps408.OverlayValues[116] = d116
						ps408.OverlayValues[117] = d117
						ps408.OverlayValues[118] = d118
						ps408.OverlayValues[171] = d171
						ps408.OverlayValues[172] = d172
						ps408.OverlayValues[173] = d173
						ps408.OverlayValues[174] = d174
						ps408.OverlayValues[175] = d175
						ps408.OverlayValues[176] = d176
						ps408.OverlayValues[177] = d177
						ps408.OverlayValues[178] = d178
						ps408.OverlayValues[179] = d179
						ps408.OverlayValues[180] = d180
						ps408.OverlayValues[181] = d181
						ps408.OverlayValues[182] = d182
						ps408.OverlayValues[183] = d183
						ps408.OverlayValues[184] = d184
						ps408.OverlayValues[185] = d185
						ps408.OverlayValues[186] = d186
						ps408.OverlayValues[187] = d187
						ps408.OverlayValues[188] = d188
						ps408.OverlayValues[189] = d189
						ps408.OverlayValues[191] = d191
						ps408.OverlayValues[192] = d192
						ps408.OverlayValues[193] = d193
						ps408.OverlayValues[194] = d194
						ps408.OverlayValues[195] = d195
						ps408.OverlayValues[196] = d196
						ps408.OverlayValues[197] = d197
						ps408.OverlayValues[198] = d198
						ps408.OverlayValues[200] = d200
						ps408.OverlayValues[201] = d201
						ps408.OverlayValues[202] = d202
						ps408.OverlayValues[287] = d287
						ps408.OverlayValues[288] = d288
						ps408.OverlayValues[289] = d289
						ps408.OverlayValues[290] = d290
						ps408.OverlayValues[291] = d291
						ps408.OverlayValues[294] = d294
						ps408.OverlayValues[295] = d295
						ps408.OverlayValues[385] = d385
						ps408.OverlayValues[386] = d386
						ps408.OverlayValues[387] = d387
						ps408.OverlayValues[388] = d388
						ps408.OverlayValues[389] = d389
						ps408.OverlayValues[390] = d390
						ps408.OverlayValues[391] = d391
						ps408.OverlayValues[392] = d392
						ps408.OverlayValues[393] = d393
						ps408.OverlayValues[394] = d394
						ps408.OverlayValues[395] = d395
						ps408.OverlayValues[396] = d396
						ps408.OverlayValues[397] = d397
						ps408.OverlayValues[399] = d399
						ps408.OverlayValues[400] = d400
						ps408.OverlayValues[401] = d401
						ps408.OverlayValues[402] = d402
						ps408.OverlayValues[403] = d403
						ps408.OverlayValues[404] = d404
						ps408.OverlayValues[405] = d405
						ps408.OverlayValues[407] = d407
						ps408.PhiValues = make([]JITValueDesc, 1)
						d409 = d14
						ps408.PhiValues[0] = d409
						return bbs[4].RenderPS(ps408)
					}
					if !ps.General {
						ps.General = true
						return bbs[12].RenderPS(ps)
					}
					lbl30 := ctx.ReserveLabel()
					lbl31 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d405.Reg, 0)
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
					d410 = d14
					if d410.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d410)
					ctx.EmitStoreToStack(d410, int32(bbs[4].PhiBase)+int32(0))
					if d14.Loc == LocReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ps411 := PhiState{General: true}
					ps411.OverlayValues = make([]JITValueDesc, 411)
					ps411.OverlayValues[8] = d8
					ps411.OverlayValues[9] = d9
					ps411.OverlayValues[10] = d10
					ps411.OverlayValues[11] = d11
					ps411.OverlayValues[12] = d12
					ps411.OverlayValues[13] = d13
					ps411.OverlayValues[14] = d14
					ps411.OverlayValues[15] = d15
					ps411.OverlayValues[16] = d16
					ps411.OverlayValues[17] = d17
					ps411.OverlayValues[18] = d18
					ps411.OverlayValues[19] = d19
					ps411.OverlayValues[20] = d20
					ps411.OverlayValues[21] = d21
					ps411.OverlayValues[22] = d22
					ps411.OverlayValues[25] = d25
					ps411.OverlayValues[28] = d28
					ps411.OverlayValues[47] = d47
					ps411.OverlayValues[48] = d48
					ps411.OverlayValues[49] = d49
					ps411.OverlayValues[50] = d50
					ps411.OverlayValues[51] = d51
					ps411.OverlayValues[53] = d53
					ps411.OverlayValues[54] = d54
					ps411.OverlayValues[55] = d55
					ps411.OverlayValues[56] = d56
					ps411.OverlayValues[57] = d57
					ps411.OverlayValues[58] = d58
					ps411.OverlayValues[59] = d59
					ps411.OverlayValues[62] = d62
					ps411.OverlayValues[97] = d97
					ps411.OverlayValues[98] = d98
					ps411.OverlayValues[99] = d99
					ps411.OverlayValues[100] = d100
					ps411.OverlayValues[101] = d101
					ps411.OverlayValues[102] = d102
					ps411.OverlayValues[104] = d104
					ps411.OverlayValues[105] = d105
					ps411.OverlayValues[106] = d106
					ps411.OverlayValues[107] = d107
					ps411.OverlayValues[108] = d108
					ps411.OverlayValues[109] = d109
					ps411.OverlayValues[110] = d110
					ps411.OverlayValues[111] = d111
					ps411.OverlayValues[112] = d112
					ps411.OverlayValues[115] = d115
					ps411.OverlayValues[116] = d116
					ps411.OverlayValues[117] = d117
					ps411.OverlayValues[118] = d118
					ps411.OverlayValues[171] = d171
					ps411.OverlayValues[172] = d172
					ps411.OverlayValues[173] = d173
					ps411.OverlayValues[174] = d174
					ps411.OverlayValues[175] = d175
					ps411.OverlayValues[176] = d176
					ps411.OverlayValues[177] = d177
					ps411.OverlayValues[178] = d178
					ps411.OverlayValues[179] = d179
					ps411.OverlayValues[180] = d180
					ps411.OverlayValues[181] = d181
					ps411.OverlayValues[182] = d182
					ps411.OverlayValues[183] = d183
					ps411.OverlayValues[184] = d184
					ps411.OverlayValues[185] = d185
					ps411.OverlayValues[186] = d186
					ps411.OverlayValues[187] = d187
					ps411.OverlayValues[188] = d188
					ps411.OverlayValues[189] = d189
					ps411.OverlayValues[191] = d191
					ps411.OverlayValues[192] = d192
					ps411.OverlayValues[193] = d193
					ps411.OverlayValues[194] = d194
					ps411.OverlayValues[195] = d195
					ps411.OverlayValues[196] = d196
					ps411.OverlayValues[197] = d197
					ps411.OverlayValues[198] = d198
					ps411.OverlayValues[200] = d200
					ps411.OverlayValues[201] = d201
					ps411.OverlayValues[202] = d202
					ps411.OverlayValues[287] = d287
					ps411.OverlayValues[288] = d288
					ps411.OverlayValues[289] = d289
					ps411.OverlayValues[290] = d290
					ps411.OverlayValues[291] = d291
					ps411.OverlayValues[294] = d294
					ps411.OverlayValues[295] = d295
					ps411.OverlayValues[385] = d385
					ps411.OverlayValues[386] = d386
					ps411.OverlayValues[387] = d387
					ps411.OverlayValues[388] = d388
					ps411.OverlayValues[389] = d389
					ps411.OverlayValues[390] = d390
					ps411.OverlayValues[391] = d391
					ps411.OverlayValues[392] = d392
					ps411.OverlayValues[393] = d393
					ps411.OverlayValues[394] = d394
					ps411.OverlayValues[395] = d395
					ps411.OverlayValues[396] = d396
					ps411.OverlayValues[397] = d397
					ps411.OverlayValues[399] = d399
					ps411.OverlayValues[400] = d400
					ps411.OverlayValues[401] = d401
					ps411.OverlayValues[402] = d402
					ps411.OverlayValues[403] = d403
					ps411.OverlayValues[404] = d404
					ps411.OverlayValues[405] = d405
					ps411.OverlayValues[407] = d407
					ps411.OverlayValues[409] = d409
					ps411.OverlayValues[410] = d410
					ps412 := PhiState{General: true}
					ps412.OverlayValues = make([]JITValueDesc, 411)
					ps412.OverlayValues[8] = d8
					ps412.OverlayValues[9] = d9
					ps412.OverlayValues[10] = d10
					ps412.OverlayValues[11] = d11
					ps412.OverlayValues[12] = d12
					ps412.OverlayValues[13] = d13
					ps412.OverlayValues[14] = d14
					ps412.OverlayValues[15] = d15
					ps412.OverlayValues[16] = d16
					ps412.OverlayValues[17] = d17
					ps412.OverlayValues[18] = d18
					ps412.OverlayValues[19] = d19
					ps412.OverlayValues[20] = d20
					ps412.OverlayValues[21] = d21
					ps412.OverlayValues[22] = d22
					ps412.OverlayValues[25] = d25
					ps412.OverlayValues[28] = d28
					ps412.OverlayValues[47] = d47
					ps412.OverlayValues[48] = d48
					ps412.OverlayValues[49] = d49
					ps412.OverlayValues[50] = d50
					ps412.OverlayValues[51] = d51
					ps412.OverlayValues[53] = d53
					ps412.OverlayValues[54] = d54
					ps412.OverlayValues[55] = d55
					ps412.OverlayValues[56] = d56
					ps412.OverlayValues[57] = d57
					ps412.OverlayValues[58] = d58
					ps412.OverlayValues[59] = d59
					ps412.OverlayValues[62] = d62
					ps412.OverlayValues[97] = d97
					ps412.OverlayValues[98] = d98
					ps412.OverlayValues[99] = d99
					ps412.OverlayValues[100] = d100
					ps412.OverlayValues[101] = d101
					ps412.OverlayValues[102] = d102
					ps412.OverlayValues[104] = d104
					ps412.OverlayValues[105] = d105
					ps412.OverlayValues[106] = d106
					ps412.OverlayValues[107] = d107
					ps412.OverlayValues[108] = d108
					ps412.OverlayValues[109] = d109
					ps412.OverlayValues[110] = d110
					ps412.OverlayValues[111] = d111
					ps412.OverlayValues[112] = d112
					ps412.OverlayValues[115] = d115
					ps412.OverlayValues[116] = d116
					ps412.OverlayValues[117] = d117
					ps412.OverlayValues[118] = d118
					ps412.OverlayValues[171] = d171
					ps412.OverlayValues[172] = d172
					ps412.OverlayValues[173] = d173
					ps412.OverlayValues[174] = d174
					ps412.OverlayValues[175] = d175
					ps412.OverlayValues[176] = d176
					ps412.OverlayValues[177] = d177
					ps412.OverlayValues[178] = d178
					ps412.OverlayValues[179] = d179
					ps412.OverlayValues[180] = d180
					ps412.OverlayValues[181] = d181
					ps412.OverlayValues[182] = d182
					ps412.OverlayValues[183] = d183
					ps412.OverlayValues[184] = d184
					ps412.OverlayValues[185] = d185
					ps412.OverlayValues[186] = d186
					ps412.OverlayValues[187] = d187
					ps412.OverlayValues[188] = d188
					ps412.OverlayValues[189] = d189
					ps412.OverlayValues[191] = d191
					ps412.OverlayValues[192] = d192
					ps412.OverlayValues[193] = d193
					ps412.OverlayValues[194] = d194
					ps412.OverlayValues[195] = d195
					ps412.OverlayValues[196] = d196
					ps412.OverlayValues[197] = d197
					ps412.OverlayValues[198] = d198
					ps412.OverlayValues[200] = d200
					ps412.OverlayValues[201] = d201
					ps412.OverlayValues[202] = d202
					ps412.OverlayValues[287] = d287
					ps412.OverlayValues[288] = d288
					ps412.OverlayValues[289] = d289
					ps412.OverlayValues[290] = d290
					ps412.OverlayValues[291] = d291
					ps412.OverlayValues[294] = d294
					ps412.OverlayValues[295] = d295
					ps412.OverlayValues[385] = d385
					ps412.OverlayValues[386] = d386
					ps412.OverlayValues[387] = d387
					ps412.OverlayValues[388] = d388
					ps412.OverlayValues[389] = d389
					ps412.OverlayValues[390] = d390
					ps412.OverlayValues[391] = d391
					ps412.OverlayValues[392] = d392
					ps412.OverlayValues[393] = d393
					ps412.OverlayValues[394] = d394
					ps412.OverlayValues[395] = d395
					ps412.OverlayValues[396] = d396
					ps412.OverlayValues[397] = d397
					ps412.OverlayValues[399] = d399
					ps412.OverlayValues[400] = d400
					ps412.OverlayValues[401] = d401
					ps412.OverlayValues[402] = d402
					ps412.OverlayValues[403] = d403
					ps412.OverlayValues[404] = d404
					ps412.OverlayValues[405] = d405
					ps412.OverlayValues[407] = d407
					ps412.OverlayValues[409] = d409
					ps412.OverlayValues[410] = d410
					ps412.PhiValues = make([]JITValueDesc, 1)
					d413 = d14
					ps412.PhiValues[0] = d413
					snap414 := d8
					snap415 := d9
					snap416 := d10
					snap417 := d11
					snap418 := d12
					snap419 := d13
					snap420 := d14
					snap421 := d15
					snap422 := d16
					snap423 := d17
					snap424 := d18
					snap425 := d19
					snap426 := d20
					snap427 := d21
					snap428 := d22
					snap429 := d25
					snap430 := d28
					snap431 := d47
					snap432 := d48
					snap433 := d49
					snap434 := d50
					snap435 := d51
					snap436 := d53
					snap437 := d54
					snap438 := d55
					snap439 := d56
					snap440 := d57
					snap441 := d58
					snap442 := d59
					snap443 := d62
					snap444 := d97
					snap445 := d98
					snap446 := d99
					snap447 := d100
					snap448 := d101
					snap449 := d102
					snap450 := d104
					snap451 := d105
					snap452 := d106
					snap453 := d107
					snap454 := d108
					snap455 := d109
					snap456 := d110
					snap457 := d111
					snap458 := d112
					snap459 := d115
					snap460 := d116
					snap461 := d117
					snap462 := d118
					snap463 := d171
					snap464 := d172
					snap465 := d173
					snap466 := d174
					snap467 := d175
					snap468 := d176
					snap469 := d177
					snap470 := d178
					snap471 := d179
					snap472 := d180
					snap473 := d181
					snap474 := d182
					snap475 := d183
					snap476 := d184
					snap477 := d185
					snap478 := d186
					snap479 := d187
					snap480 := d188
					snap481 := d189
					snap482 := d191
					snap483 := d192
					snap484 := d193
					snap485 := d194
					snap486 := d195
					snap487 := d196
					snap488 := d197
					snap489 := d198
					snap490 := d200
					snap491 := d201
					snap492 := d202
					snap493 := d287
					snap494 := d288
					snap495 := d289
					snap496 := d290
					snap497 := d291
					snap498 := d294
					snap499 := d295
					snap500 := d385
					snap501 := d386
					snap502 := d387
					snap503 := d388
					snap504 := d389
					snap505 := d390
					snap506 := d391
					snap507 := d392
					snap508 := d393
					snap509 := d394
					snap510 := d395
					snap511 := d396
					snap512 := d397
					snap513 := d399
					snap514 := d400
					snap515 := d401
					snap516 := d402
					snap517 := d403
					snap518 := d404
					snap519 := d405
					snap520 := d407
					snap521 := d409
					snap522 := d410
					snap523 := d413
					alloc524 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps412)
					}
					ctx.RestoreAllocState(alloc524)
					d8 = snap414
					d9 = snap415
					d10 = snap416
					d11 = snap417
					d12 = snap418
					d13 = snap419
					d14 = snap420
					d15 = snap421
					d16 = snap422
					d17 = snap423
					d18 = snap424
					d19 = snap425
					d20 = snap426
					d21 = snap427
					d22 = snap428
					d25 = snap429
					d28 = snap430
					d47 = snap431
					d48 = snap432
					d49 = snap433
					d50 = snap434
					d51 = snap435
					d53 = snap436
					d54 = snap437
					d55 = snap438
					d56 = snap439
					d57 = snap440
					d58 = snap441
					d59 = snap442
					d62 = snap443
					d97 = snap444
					d98 = snap445
					d99 = snap446
					d100 = snap447
					d101 = snap448
					d102 = snap449
					d104 = snap450
					d105 = snap451
					d106 = snap452
					d107 = snap453
					d108 = snap454
					d109 = snap455
					d110 = snap456
					d111 = snap457
					d112 = snap458
					d115 = snap459
					d116 = snap460
					d117 = snap461
					d118 = snap462
					d171 = snap463
					d172 = snap464
					d173 = snap465
					d174 = snap466
					d175 = snap467
					d176 = snap468
					d177 = snap469
					d178 = snap470
					d179 = snap471
					d180 = snap472
					d181 = snap473
					d182 = snap474
					d183 = snap475
					d184 = snap476
					d185 = snap477
					d186 = snap478
					d187 = snap479
					d188 = snap480
					d189 = snap481
					d191 = snap482
					d192 = snap483
					d193 = snap484
					d194 = snap485
					d195 = snap486
					d196 = snap487
					d197 = snap488
					d198 = snap489
					d200 = snap490
					d201 = snap491
					d202 = snap492
					d287 = snap493
					d288 = snap494
					d289 = snap495
					d290 = snap496
					d291 = snap497
					d294 = snap498
					d295 = snap499
					d385 = snap500
					d386 = snap501
					d387 = snap502
					d388 = snap503
					d389 = snap504
					d390 = snap505
					d391 = snap506
					d392 = snap507
					d393 = snap508
					d394 = snap509
					d395 = snap510
					d396 = snap511
					d397 = snap512
					d399 = snap513
					d400 = snap514
					d401 = snap515
					d402 = snap516
					d403 = snap517
					d404 = snap518
					d405 = snap519
					d407 = snap520
					d409 = snap521
					d410 = snap522
					d413 = snap523
					if !bbs[14].Rendered {
						return bbs[14].RenderPS(ps411)
					}
					return result
					ctx.FreeDesc(&d404)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
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
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != LocNone {
						d390 = ps.OverlayValues[390]
					}
					if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != LocNone {
						d391 = ps.OverlayValues[391]
					}
					if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != LocNone {
						d392 = ps.OverlayValues[392]
					}
					if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != LocNone {
						d393 = ps.OverlayValues[393]
					}
					if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != LocNone {
						d394 = ps.OverlayValues[394]
					}
					if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != LocNone {
						d395 = ps.OverlayValues[395]
					}
					if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != LocNone {
						d396 = ps.OverlayValues[396]
					}
					if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != LocNone {
						d397 = ps.OverlayValues[397]
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
					if len(ps.OverlayValues) > 407 && ps.OverlayValues[407].Loc != LocNone {
						d407 = ps.OverlayValues[407]
					}
					if len(ps.OverlayValues) > 409 && ps.OverlayValues[409].Loc != LocNone {
						d409 = ps.OverlayValues[409]
					}
					if len(ps.OverlayValues) > 410 && ps.OverlayValues[410].Loc != LocNone {
						d410 = ps.OverlayValues[410]
					}
					if len(ps.OverlayValues) > 413 && ps.OverlayValues[413].Loc != LocNone {
						d413 = ps.OverlayValues[413]
					}
					ctx.ReclaimUntrackedRegs()
					var d525 JITValueDesc
					if d19.SliceSizeKnown {
						d525 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d525 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d525 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d525 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d525 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d525)
					ctx.EnsureDescsTogether(&d15, &d525)
					var d526 JITValueDesc
					if d15.Loc == LocImm && d525.Loc == LocImm {
						d526 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d525.Imm.Int())}
					} else if d525.Loc == LocImm {
						r26 := ctx.AllocReg()
						if d525.Imm.Int() >= -2147483648 && d525.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d525.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d525.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, RegR11)
						}
						ctx.EmitSetcc(r26, CondSignedLess)
						d526 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d526)
					} else if d15.Loc == LocImm {
						r27 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d525.Reg)
						ctx.EmitSetcc(r27, CondSignedLess)
						d526 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r27}
						ctx.BindReg(r27, &d526)
					} else {
						r28 := ctx.AllocReg()
						ctx.EmitCmpInt64(d15.Reg, d525.Reg)
						ctx.EmitSetcc(r28, CondSignedLess)
						d526 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r28}
						ctx.BindReg(r28, &d526)
					}
					ctx.FreeDesc(&d15)
					ctx.FreeDesc(&d525)
					d527 = d526
					ctx.EnsureDesc(&d527)
					if d527.Loc != LocImm && d527.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d527.Loc == LocImm {
						if d527.Imm.Bool() {
							if ps.General {
							}
							ps528 := PhiState{General: ps.General}
							ps528.OverlayValues = make([]JITValueDesc, 528)
							ps528.OverlayValues[8] = d8
							ps528.OverlayValues[9] = d9
							ps528.OverlayValues[10] = d10
							ps528.OverlayValues[11] = d11
							ps528.OverlayValues[12] = d12
							ps528.OverlayValues[13] = d13
							ps528.OverlayValues[14] = d14
							ps528.OverlayValues[15] = d15
							ps528.OverlayValues[16] = d16
							ps528.OverlayValues[17] = d17
							ps528.OverlayValues[18] = d18
							ps528.OverlayValues[19] = d19
							ps528.OverlayValues[20] = d20
							ps528.OverlayValues[21] = d21
							ps528.OverlayValues[22] = d22
							ps528.OverlayValues[25] = d25
							ps528.OverlayValues[28] = d28
							ps528.OverlayValues[47] = d47
							ps528.OverlayValues[48] = d48
							ps528.OverlayValues[49] = d49
							ps528.OverlayValues[50] = d50
							ps528.OverlayValues[51] = d51
							ps528.OverlayValues[53] = d53
							ps528.OverlayValues[54] = d54
							ps528.OverlayValues[55] = d55
							ps528.OverlayValues[56] = d56
							ps528.OverlayValues[57] = d57
							ps528.OverlayValues[58] = d58
							ps528.OverlayValues[59] = d59
							ps528.OverlayValues[62] = d62
							ps528.OverlayValues[97] = d97
							ps528.OverlayValues[98] = d98
							ps528.OverlayValues[99] = d99
							ps528.OverlayValues[100] = d100
							ps528.OverlayValues[101] = d101
							ps528.OverlayValues[102] = d102
							ps528.OverlayValues[104] = d104
							ps528.OverlayValues[105] = d105
							ps528.OverlayValues[106] = d106
							ps528.OverlayValues[107] = d107
							ps528.OverlayValues[108] = d108
							ps528.OverlayValues[109] = d109
							ps528.OverlayValues[110] = d110
							ps528.OverlayValues[111] = d111
							ps528.OverlayValues[112] = d112
							ps528.OverlayValues[115] = d115
							ps528.OverlayValues[116] = d116
							ps528.OverlayValues[117] = d117
							ps528.OverlayValues[118] = d118
							ps528.OverlayValues[171] = d171
							ps528.OverlayValues[172] = d172
							ps528.OverlayValues[173] = d173
							ps528.OverlayValues[174] = d174
							ps528.OverlayValues[175] = d175
							ps528.OverlayValues[176] = d176
							ps528.OverlayValues[177] = d177
							ps528.OverlayValues[178] = d178
							ps528.OverlayValues[179] = d179
							ps528.OverlayValues[180] = d180
							ps528.OverlayValues[181] = d181
							ps528.OverlayValues[182] = d182
							ps528.OverlayValues[183] = d183
							ps528.OverlayValues[184] = d184
							ps528.OverlayValues[185] = d185
							ps528.OverlayValues[186] = d186
							ps528.OverlayValues[187] = d187
							ps528.OverlayValues[188] = d188
							ps528.OverlayValues[189] = d189
							ps528.OverlayValues[191] = d191
							ps528.OverlayValues[192] = d192
							ps528.OverlayValues[193] = d193
							ps528.OverlayValues[194] = d194
							ps528.OverlayValues[195] = d195
							ps528.OverlayValues[196] = d196
							ps528.OverlayValues[197] = d197
							ps528.OverlayValues[198] = d198
							ps528.OverlayValues[200] = d200
							ps528.OverlayValues[201] = d201
							ps528.OverlayValues[202] = d202
							ps528.OverlayValues[287] = d287
							ps528.OverlayValues[288] = d288
							ps528.OverlayValues[289] = d289
							ps528.OverlayValues[290] = d290
							ps528.OverlayValues[291] = d291
							ps528.OverlayValues[294] = d294
							ps528.OverlayValues[295] = d295
							ps528.OverlayValues[385] = d385
							ps528.OverlayValues[386] = d386
							ps528.OverlayValues[387] = d387
							ps528.OverlayValues[388] = d388
							ps528.OverlayValues[389] = d389
							ps528.OverlayValues[390] = d390
							ps528.OverlayValues[391] = d391
							ps528.OverlayValues[392] = d392
							ps528.OverlayValues[393] = d393
							ps528.OverlayValues[394] = d394
							ps528.OverlayValues[395] = d395
							ps528.OverlayValues[396] = d396
							ps528.OverlayValues[397] = d397
							ps528.OverlayValues[399] = d399
							ps528.OverlayValues[400] = d400
							ps528.OverlayValues[401] = d401
							ps528.OverlayValues[402] = d402
							ps528.OverlayValues[403] = d403
							ps528.OverlayValues[404] = d404
							ps528.OverlayValues[405] = d405
							ps528.OverlayValues[407] = d407
							ps528.OverlayValues[409] = d409
							ps528.OverlayValues[410] = d410
							ps528.OverlayValues[413] = d413
							ps528.OverlayValues[525] = d525
							ps528.OverlayValues[526] = d526
							ps528.OverlayValues[527] = d527
							return bbs[11].RenderPS(ps528)
						}
						if ps.General {
						}
						ps529 := PhiState{General: ps.General}
						ps529.OverlayValues = make([]JITValueDesc, 528)
						ps529.OverlayValues[8] = d8
						ps529.OverlayValues[9] = d9
						ps529.OverlayValues[10] = d10
						ps529.OverlayValues[11] = d11
						ps529.OverlayValues[12] = d12
						ps529.OverlayValues[13] = d13
						ps529.OverlayValues[14] = d14
						ps529.OverlayValues[15] = d15
						ps529.OverlayValues[16] = d16
						ps529.OverlayValues[17] = d17
						ps529.OverlayValues[18] = d18
						ps529.OverlayValues[19] = d19
						ps529.OverlayValues[20] = d20
						ps529.OverlayValues[21] = d21
						ps529.OverlayValues[22] = d22
						ps529.OverlayValues[25] = d25
						ps529.OverlayValues[28] = d28
						ps529.OverlayValues[47] = d47
						ps529.OverlayValues[48] = d48
						ps529.OverlayValues[49] = d49
						ps529.OverlayValues[50] = d50
						ps529.OverlayValues[51] = d51
						ps529.OverlayValues[53] = d53
						ps529.OverlayValues[54] = d54
						ps529.OverlayValues[55] = d55
						ps529.OverlayValues[56] = d56
						ps529.OverlayValues[57] = d57
						ps529.OverlayValues[58] = d58
						ps529.OverlayValues[59] = d59
						ps529.OverlayValues[62] = d62
						ps529.OverlayValues[97] = d97
						ps529.OverlayValues[98] = d98
						ps529.OverlayValues[99] = d99
						ps529.OverlayValues[100] = d100
						ps529.OverlayValues[101] = d101
						ps529.OverlayValues[102] = d102
						ps529.OverlayValues[104] = d104
						ps529.OverlayValues[105] = d105
						ps529.OverlayValues[106] = d106
						ps529.OverlayValues[107] = d107
						ps529.OverlayValues[108] = d108
						ps529.OverlayValues[109] = d109
						ps529.OverlayValues[110] = d110
						ps529.OverlayValues[111] = d111
						ps529.OverlayValues[112] = d112
						ps529.OverlayValues[115] = d115
						ps529.OverlayValues[116] = d116
						ps529.OverlayValues[117] = d117
						ps529.OverlayValues[118] = d118
						ps529.OverlayValues[171] = d171
						ps529.OverlayValues[172] = d172
						ps529.OverlayValues[173] = d173
						ps529.OverlayValues[174] = d174
						ps529.OverlayValues[175] = d175
						ps529.OverlayValues[176] = d176
						ps529.OverlayValues[177] = d177
						ps529.OverlayValues[178] = d178
						ps529.OverlayValues[179] = d179
						ps529.OverlayValues[180] = d180
						ps529.OverlayValues[181] = d181
						ps529.OverlayValues[182] = d182
						ps529.OverlayValues[183] = d183
						ps529.OverlayValues[184] = d184
						ps529.OverlayValues[185] = d185
						ps529.OverlayValues[186] = d186
						ps529.OverlayValues[187] = d187
						ps529.OverlayValues[188] = d188
						ps529.OverlayValues[189] = d189
						ps529.OverlayValues[191] = d191
						ps529.OverlayValues[192] = d192
						ps529.OverlayValues[193] = d193
						ps529.OverlayValues[194] = d194
						ps529.OverlayValues[195] = d195
						ps529.OverlayValues[196] = d196
						ps529.OverlayValues[197] = d197
						ps529.OverlayValues[198] = d198
						ps529.OverlayValues[200] = d200
						ps529.OverlayValues[201] = d201
						ps529.OverlayValues[202] = d202
						ps529.OverlayValues[287] = d287
						ps529.OverlayValues[288] = d288
						ps529.OverlayValues[289] = d289
						ps529.OverlayValues[290] = d290
						ps529.OverlayValues[291] = d291
						ps529.OverlayValues[294] = d294
						ps529.OverlayValues[295] = d295
						ps529.OverlayValues[385] = d385
						ps529.OverlayValues[386] = d386
						ps529.OverlayValues[387] = d387
						ps529.OverlayValues[388] = d388
						ps529.OverlayValues[389] = d389
						ps529.OverlayValues[390] = d390
						ps529.OverlayValues[391] = d391
						ps529.OverlayValues[392] = d392
						ps529.OverlayValues[393] = d393
						ps529.OverlayValues[394] = d394
						ps529.OverlayValues[395] = d395
						ps529.OverlayValues[396] = d396
						ps529.OverlayValues[397] = d397
						ps529.OverlayValues[399] = d399
						ps529.OverlayValues[400] = d400
						ps529.OverlayValues[401] = d401
						ps529.OverlayValues[402] = d402
						ps529.OverlayValues[403] = d403
						ps529.OverlayValues[404] = d404
						ps529.OverlayValues[405] = d405
						ps529.OverlayValues[407] = d407
						ps529.OverlayValues[409] = d409
						ps529.OverlayValues[410] = d410
						ps529.OverlayValues[413] = d413
						ps529.OverlayValues[525] = d525
						ps529.OverlayValues[526] = d526
						ps529.OverlayValues[527] = d527
						return bbs[12].RenderPS(ps529)
					}
					if !ps.General {
						ps.General = true
						return bbs[13].RenderPS(ps)
					}
					lbl32 := ctx.ReserveLabel()
					lbl33 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d527.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl32)
					ctx.EmitJmp(lbl33)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl13)
					ps530 := PhiState{General: true}
					ps530.OverlayValues = make([]JITValueDesc, 528)
					ps530.OverlayValues[8] = d8
					ps530.OverlayValues[9] = d9
					ps530.OverlayValues[10] = d10
					ps530.OverlayValues[11] = d11
					ps530.OverlayValues[12] = d12
					ps530.OverlayValues[13] = d13
					ps530.OverlayValues[14] = d14
					ps530.OverlayValues[15] = d15
					ps530.OverlayValues[16] = d16
					ps530.OverlayValues[17] = d17
					ps530.OverlayValues[18] = d18
					ps530.OverlayValues[19] = d19
					ps530.OverlayValues[20] = d20
					ps530.OverlayValues[21] = d21
					ps530.OverlayValues[22] = d22
					ps530.OverlayValues[25] = d25
					ps530.OverlayValues[28] = d28
					ps530.OverlayValues[47] = d47
					ps530.OverlayValues[48] = d48
					ps530.OverlayValues[49] = d49
					ps530.OverlayValues[50] = d50
					ps530.OverlayValues[51] = d51
					ps530.OverlayValues[53] = d53
					ps530.OverlayValues[54] = d54
					ps530.OverlayValues[55] = d55
					ps530.OverlayValues[56] = d56
					ps530.OverlayValues[57] = d57
					ps530.OverlayValues[58] = d58
					ps530.OverlayValues[59] = d59
					ps530.OverlayValues[62] = d62
					ps530.OverlayValues[97] = d97
					ps530.OverlayValues[98] = d98
					ps530.OverlayValues[99] = d99
					ps530.OverlayValues[100] = d100
					ps530.OverlayValues[101] = d101
					ps530.OverlayValues[102] = d102
					ps530.OverlayValues[104] = d104
					ps530.OverlayValues[105] = d105
					ps530.OverlayValues[106] = d106
					ps530.OverlayValues[107] = d107
					ps530.OverlayValues[108] = d108
					ps530.OverlayValues[109] = d109
					ps530.OverlayValues[110] = d110
					ps530.OverlayValues[111] = d111
					ps530.OverlayValues[112] = d112
					ps530.OverlayValues[115] = d115
					ps530.OverlayValues[116] = d116
					ps530.OverlayValues[117] = d117
					ps530.OverlayValues[118] = d118
					ps530.OverlayValues[171] = d171
					ps530.OverlayValues[172] = d172
					ps530.OverlayValues[173] = d173
					ps530.OverlayValues[174] = d174
					ps530.OverlayValues[175] = d175
					ps530.OverlayValues[176] = d176
					ps530.OverlayValues[177] = d177
					ps530.OverlayValues[178] = d178
					ps530.OverlayValues[179] = d179
					ps530.OverlayValues[180] = d180
					ps530.OverlayValues[181] = d181
					ps530.OverlayValues[182] = d182
					ps530.OverlayValues[183] = d183
					ps530.OverlayValues[184] = d184
					ps530.OverlayValues[185] = d185
					ps530.OverlayValues[186] = d186
					ps530.OverlayValues[187] = d187
					ps530.OverlayValues[188] = d188
					ps530.OverlayValues[189] = d189
					ps530.OverlayValues[191] = d191
					ps530.OverlayValues[192] = d192
					ps530.OverlayValues[193] = d193
					ps530.OverlayValues[194] = d194
					ps530.OverlayValues[195] = d195
					ps530.OverlayValues[196] = d196
					ps530.OverlayValues[197] = d197
					ps530.OverlayValues[198] = d198
					ps530.OverlayValues[200] = d200
					ps530.OverlayValues[201] = d201
					ps530.OverlayValues[202] = d202
					ps530.OverlayValues[287] = d287
					ps530.OverlayValues[288] = d288
					ps530.OverlayValues[289] = d289
					ps530.OverlayValues[290] = d290
					ps530.OverlayValues[291] = d291
					ps530.OverlayValues[294] = d294
					ps530.OverlayValues[295] = d295
					ps530.OverlayValues[385] = d385
					ps530.OverlayValues[386] = d386
					ps530.OverlayValues[387] = d387
					ps530.OverlayValues[388] = d388
					ps530.OverlayValues[389] = d389
					ps530.OverlayValues[390] = d390
					ps530.OverlayValues[391] = d391
					ps530.OverlayValues[392] = d392
					ps530.OverlayValues[393] = d393
					ps530.OverlayValues[394] = d394
					ps530.OverlayValues[395] = d395
					ps530.OverlayValues[396] = d396
					ps530.OverlayValues[397] = d397
					ps530.OverlayValues[399] = d399
					ps530.OverlayValues[400] = d400
					ps530.OverlayValues[401] = d401
					ps530.OverlayValues[402] = d402
					ps530.OverlayValues[403] = d403
					ps530.OverlayValues[404] = d404
					ps530.OverlayValues[405] = d405
					ps530.OverlayValues[407] = d407
					ps530.OverlayValues[409] = d409
					ps530.OverlayValues[410] = d410
					ps530.OverlayValues[413] = d413
					ps530.OverlayValues[525] = d525
					ps530.OverlayValues[526] = d526
					ps530.OverlayValues[527] = d527
					ps531 := PhiState{General: true}
					ps531.OverlayValues = make([]JITValueDesc, 528)
					ps531.OverlayValues[8] = d8
					ps531.OverlayValues[9] = d9
					ps531.OverlayValues[10] = d10
					ps531.OverlayValues[11] = d11
					ps531.OverlayValues[12] = d12
					ps531.OverlayValues[13] = d13
					ps531.OverlayValues[14] = d14
					ps531.OverlayValues[15] = d15
					ps531.OverlayValues[16] = d16
					ps531.OverlayValues[17] = d17
					ps531.OverlayValues[18] = d18
					ps531.OverlayValues[19] = d19
					ps531.OverlayValues[20] = d20
					ps531.OverlayValues[21] = d21
					ps531.OverlayValues[22] = d22
					ps531.OverlayValues[25] = d25
					ps531.OverlayValues[28] = d28
					ps531.OverlayValues[47] = d47
					ps531.OverlayValues[48] = d48
					ps531.OverlayValues[49] = d49
					ps531.OverlayValues[50] = d50
					ps531.OverlayValues[51] = d51
					ps531.OverlayValues[53] = d53
					ps531.OverlayValues[54] = d54
					ps531.OverlayValues[55] = d55
					ps531.OverlayValues[56] = d56
					ps531.OverlayValues[57] = d57
					ps531.OverlayValues[58] = d58
					ps531.OverlayValues[59] = d59
					ps531.OverlayValues[62] = d62
					ps531.OverlayValues[97] = d97
					ps531.OverlayValues[98] = d98
					ps531.OverlayValues[99] = d99
					ps531.OverlayValues[100] = d100
					ps531.OverlayValues[101] = d101
					ps531.OverlayValues[102] = d102
					ps531.OverlayValues[104] = d104
					ps531.OverlayValues[105] = d105
					ps531.OverlayValues[106] = d106
					ps531.OverlayValues[107] = d107
					ps531.OverlayValues[108] = d108
					ps531.OverlayValues[109] = d109
					ps531.OverlayValues[110] = d110
					ps531.OverlayValues[111] = d111
					ps531.OverlayValues[112] = d112
					ps531.OverlayValues[115] = d115
					ps531.OverlayValues[116] = d116
					ps531.OverlayValues[117] = d117
					ps531.OverlayValues[118] = d118
					ps531.OverlayValues[171] = d171
					ps531.OverlayValues[172] = d172
					ps531.OverlayValues[173] = d173
					ps531.OverlayValues[174] = d174
					ps531.OverlayValues[175] = d175
					ps531.OverlayValues[176] = d176
					ps531.OverlayValues[177] = d177
					ps531.OverlayValues[178] = d178
					ps531.OverlayValues[179] = d179
					ps531.OverlayValues[180] = d180
					ps531.OverlayValues[181] = d181
					ps531.OverlayValues[182] = d182
					ps531.OverlayValues[183] = d183
					ps531.OverlayValues[184] = d184
					ps531.OverlayValues[185] = d185
					ps531.OverlayValues[186] = d186
					ps531.OverlayValues[187] = d187
					ps531.OverlayValues[188] = d188
					ps531.OverlayValues[189] = d189
					ps531.OverlayValues[191] = d191
					ps531.OverlayValues[192] = d192
					ps531.OverlayValues[193] = d193
					ps531.OverlayValues[194] = d194
					ps531.OverlayValues[195] = d195
					ps531.OverlayValues[196] = d196
					ps531.OverlayValues[197] = d197
					ps531.OverlayValues[198] = d198
					ps531.OverlayValues[200] = d200
					ps531.OverlayValues[201] = d201
					ps531.OverlayValues[202] = d202
					ps531.OverlayValues[287] = d287
					ps531.OverlayValues[288] = d288
					ps531.OverlayValues[289] = d289
					ps531.OverlayValues[290] = d290
					ps531.OverlayValues[291] = d291
					ps531.OverlayValues[294] = d294
					ps531.OverlayValues[295] = d295
					ps531.OverlayValues[385] = d385
					ps531.OverlayValues[386] = d386
					ps531.OverlayValues[387] = d387
					ps531.OverlayValues[388] = d388
					ps531.OverlayValues[389] = d389
					ps531.OverlayValues[390] = d390
					ps531.OverlayValues[391] = d391
					ps531.OverlayValues[392] = d392
					ps531.OverlayValues[393] = d393
					ps531.OverlayValues[394] = d394
					ps531.OverlayValues[395] = d395
					ps531.OverlayValues[396] = d396
					ps531.OverlayValues[397] = d397
					ps531.OverlayValues[399] = d399
					ps531.OverlayValues[400] = d400
					ps531.OverlayValues[401] = d401
					ps531.OverlayValues[402] = d402
					ps531.OverlayValues[403] = d403
					ps531.OverlayValues[404] = d404
					ps531.OverlayValues[405] = d405
					ps531.OverlayValues[407] = d407
					ps531.OverlayValues[409] = d409
					ps531.OverlayValues[410] = d410
					ps531.OverlayValues[413] = d413
					ps531.OverlayValues[525] = d525
					ps531.OverlayValues[526] = d526
					ps531.OverlayValues[527] = d527
					snap532 := d8
					snap533 := d9
					snap534 := d10
					snap535 := d11
					snap536 := d12
					snap537 := d13
					snap538 := d14
					snap539 := d15
					snap540 := d16
					snap541 := d17
					snap542 := d18
					snap543 := d19
					snap544 := d20
					snap545 := d21
					snap546 := d22
					snap547 := d25
					snap548 := d28
					snap549 := d47
					snap550 := d48
					snap551 := d49
					snap552 := d50
					snap553 := d51
					snap554 := d53
					snap555 := d54
					snap556 := d55
					snap557 := d56
					snap558 := d57
					snap559 := d58
					snap560 := d59
					snap561 := d62
					snap562 := d97
					snap563 := d98
					snap564 := d99
					snap565 := d100
					snap566 := d101
					snap567 := d102
					snap568 := d104
					snap569 := d105
					snap570 := d106
					snap571 := d107
					snap572 := d108
					snap573 := d109
					snap574 := d110
					snap575 := d111
					snap576 := d112
					snap577 := d115
					snap578 := d116
					snap579 := d117
					snap580 := d118
					snap581 := d171
					snap582 := d172
					snap583 := d173
					snap584 := d174
					snap585 := d175
					snap586 := d176
					snap587 := d177
					snap588 := d178
					snap589 := d179
					snap590 := d180
					snap591 := d181
					snap592 := d182
					snap593 := d183
					snap594 := d184
					snap595 := d185
					snap596 := d186
					snap597 := d187
					snap598 := d188
					snap599 := d189
					snap600 := d191
					snap601 := d192
					snap602 := d193
					snap603 := d194
					snap604 := d195
					snap605 := d196
					snap606 := d197
					snap607 := d198
					snap608 := d200
					snap609 := d201
					snap610 := d202
					snap611 := d287
					snap612 := d288
					snap613 := d289
					snap614 := d290
					snap615 := d291
					snap616 := d294
					snap617 := d295
					snap618 := d385
					snap619 := d386
					snap620 := d387
					snap621 := d388
					snap622 := d389
					snap623 := d390
					snap624 := d391
					snap625 := d392
					snap626 := d393
					snap627 := d394
					snap628 := d395
					snap629 := d396
					snap630 := d397
					snap631 := d399
					snap632 := d400
					snap633 := d401
					snap634 := d402
					snap635 := d403
					snap636 := d404
					snap637 := d405
					snap638 := d407
					snap639 := d409
					snap640 := d410
					snap641 := d413
					snap642 := d525
					snap643 := d526
					snap644 := d527
					alloc645 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps531)
					}
					ctx.RestoreAllocState(alloc645)
					d8 = snap532
					d9 = snap533
					d10 = snap534
					d11 = snap535
					d12 = snap536
					d13 = snap537
					d14 = snap538
					d15 = snap539
					d16 = snap540
					d17 = snap541
					d18 = snap542
					d19 = snap543
					d20 = snap544
					d21 = snap545
					d22 = snap546
					d25 = snap547
					d28 = snap548
					d47 = snap549
					d48 = snap550
					d49 = snap551
					d50 = snap552
					d51 = snap553
					d53 = snap554
					d54 = snap555
					d55 = snap556
					d56 = snap557
					d57 = snap558
					d58 = snap559
					d59 = snap560
					d62 = snap561
					d97 = snap562
					d98 = snap563
					d99 = snap564
					d100 = snap565
					d101 = snap566
					d102 = snap567
					d104 = snap568
					d105 = snap569
					d106 = snap570
					d107 = snap571
					d108 = snap572
					d109 = snap573
					d110 = snap574
					d111 = snap575
					d112 = snap576
					d115 = snap577
					d116 = snap578
					d117 = snap579
					d118 = snap580
					d171 = snap581
					d172 = snap582
					d173 = snap583
					d174 = snap584
					d175 = snap585
					d176 = snap586
					d177 = snap587
					d178 = snap588
					d179 = snap589
					d180 = snap590
					d181 = snap591
					d182 = snap592
					d183 = snap593
					d184 = snap594
					d185 = snap595
					d186 = snap596
					d187 = snap597
					d188 = snap598
					d189 = snap599
					d191 = snap600
					d192 = snap601
					d193 = snap602
					d194 = snap603
					d195 = snap604
					d196 = snap605
					d197 = snap606
					d198 = snap607
					d200 = snap608
					d201 = snap609
					d202 = snap610
					d287 = snap611
					d288 = snap612
					d289 = snap613
					d290 = snap614
					d291 = snap615
					d294 = snap616
					d295 = snap617
					d385 = snap618
					d386 = snap619
					d387 = snap620
					d388 = snap621
					d389 = snap622
					d390 = snap623
					d391 = snap624
					d392 = snap625
					d393 = snap626
					d394 = snap627
					d395 = snap628
					d396 = snap629
					d397 = snap630
					d399 = snap631
					d400 = snap632
					d401 = snap633
					d402 = snap634
					d403 = snap635
					d404 = snap636
					d405 = snap637
					d407 = snap638
					d409 = snap639
					d410 = snap640
					d413 = snap641
					d525 = snap642
					d526 = snap643
					d527 = snap644
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps530)
					}
					return result
					ctx.FreeDesc(&d526)
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
						d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
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
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 109 && ps.OverlayValues[109].Loc != LocNone {
						d109 = ps.OverlayValues[109]
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
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
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
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
					}
					if len(ps.OverlayValues) > 288 && ps.OverlayValues[288].Loc != LocNone {
						d288 = ps.OverlayValues[288]
					}
					if len(ps.OverlayValues) > 289 && ps.OverlayValues[289].Loc != LocNone {
						d289 = ps.OverlayValues[289]
					}
					if len(ps.OverlayValues) > 290 && ps.OverlayValues[290].Loc != LocNone {
						d290 = ps.OverlayValues[290]
					}
					if len(ps.OverlayValues) > 291 && ps.OverlayValues[291].Loc != LocNone {
						d291 = ps.OverlayValues[291]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != LocNone {
						d385 = ps.OverlayValues[385]
					}
					if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != LocNone {
						d386 = ps.OverlayValues[386]
					}
					if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != LocNone {
						d387 = ps.OverlayValues[387]
					}
					if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != LocNone {
						d388 = ps.OverlayValues[388]
					}
					if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != LocNone {
						d389 = ps.OverlayValues[389]
					}
					if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != LocNone {
						d390 = ps.OverlayValues[390]
					}
					if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != LocNone {
						d391 = ps.OverlayValues[391]
					}
					if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != LocNone {
						d392 = ps.OverlayValues[392]
					}
					if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != LocNone {
						d393 = ps.OverlayValues[393]
					}
					if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != LocNone {
						d394 = ps.OverlayValues[394]
					}
					if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != LocNone {
						d395 = ps.OverlayValues[395]
					}
					if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != LocNone {
						d396 = ps.OverlayValues[396]
					}
					if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != LocNone {
						d397 = ps.OverlayValues[397]
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
					if len(ps.OverlayValues) > 407 && ps.OverlayValues[407].Loc != LocNone {
						d407 = ps.OverlayValues[407]
					}
					if len(ps.OverlayValues) > 409 && ps.OverlayValues[409].Loc != LocNone {
						d409 = ps.OverlayValues[409]
					}
					if len(ps.OverlayValues) > 410 && ps.OverlayValues[410].Loc != LocNone {
						d410 = ps.OverlayValues[410]
					}
					if len(ps.OverlayValues) > 413 && ps.OverlayValues[413].Loc != LocNone {
						d413 = ps.OverlayValues[413]
					}
					if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != LocNone {
						d525 = ps.OverlayValues[525]
					}
					if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != LocNone {
						d526 = ps.OverlayValues[526]
					}
					if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != LocNone {
						d527 = ps.OverlayValues[527]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d14)
					var d646 JITValueDesc
					if d14.Loc == LocImm {
						d646 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d14.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d14)
						var d647 JITValueDesc
						if d14.Loc == LocRegPair {
							ctx.FreeReg(d14.Reg)
							d647 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg2}
							ctx.BindReg(d14.Reg2, &d647)
							ctx.BindReg(d14.Reg2, &d647)
						} else {
							d647 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg}
							ctx.BindReg(d14.Reg, &d647)
							ctx.BindReg(d14.Reg, &d647)
						}
						d646 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d647}, 1)
						d646.Type = tagFloat
						ctx.BindReg(d646.Reg, &d646)
					}
					ctx.StabilizeDescForControlFlow(&d646)
					if ps.General {
						ctx.SyncDesc(&d646)
						if d646.Loc == LocReg {
							ctx.ProtectReg(d646.Reg)
						} else if d646.Loc == LocRegPair {
							ctx.ProtectReg(d646.Reg)
							ctx.ProtectReg(d646.Reg2)
						}
						d648 = d646
						if d648.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d648)
						ctx.EmitStoreToStack(d648, int32(bbs[4].PhiBase)+int32(0))
						if d646.Loc == LocReg {
							ctx.UnprotectReg(d646.Reg)
						} else if d646.Loc == LocRegPair {
							ctx.UnprotectReg(d646.Reg)
							ctx.UnprotectReg(d646.Reg2)
						}
					}
					ps649 := PhiState{General: ps.General}
					ps649.OverlayValues = make([]JITValueDesc, 649)
					ps649.OverlayValues[8] = d8
					ps649.OverlayValues[9] = d9
					ps649.OverlayValues[10] = d10
					ps649.OverlayValues[11] = d11
					ps649.OverlayValues[12] = d12
					ps649.OverlayValues[13] = d13
					ps649.OverlayValues[14] = d14
					ps649.OverlayValues[15] = d15
					ps649.OverlayValues[16] = d16
					ps649.OverlayValues[17] = d17
					ps649.OverlayValues[18] = d18
					ps649.OverlayValues[19] = d19
					ps649.OverlayValues[20] = d20
					ps649.OverlayValues[21] = d21
					ps649.OverlayValues[22] = d22
					ps649.OverlayValues[25] = d25
					ps649.OverlayValues[28] = d28
					ps649.OverlayValues[47] = d47
					ps649.OverlayValues[48] = d48
					ps649.OverlayValues[49] = d49
					ps649.OverlayValues[50] = d50
					ps649.OverlayValues[51] = d51
					ps649.OverlayValues[53] = d53
					ps649.OverlayValues[54] = d54
					ps649.OverlayValues[55] = d55
					ps649.OverlayValues[56] = d56
					ps649.OverlayValues[57] = d57
					ps649.OverlayValues[58] = d58
					ps649.OverlayValues[59] = d59
					ps649.OverlayValues[62] = d62
					ps649.OverlayValues[97] = d97
					ps649.OverlayValues[98] = d98
					ps649.OverlayValues[99] = d99
					ps649.OverlayValues[100] = d100
					ps649.OverlayValues[101] = d101
					ps649.OverlayValues[102] = d102
					ps649.OverlayValues[104] = d104
					ps649.OverlayValues[105] = d105
					ps649.OverlayValues[106] = d106
					ps649.OverlayValues[107] = d107
					ps649.OverlayValues[108] = d108
					ps649.OverlayValues[109] = d109
					ps649.OverlayValues[110] = d110
					ps649.OverlayValues[111] = d111
					ps649.OverlayValues[112] = d112
					ps649.OverlayValues[115] = d115
					ps649.OverlayValues[116] = d116
					ps649.OverlayValues[117] = d117
					ps649.OverlayValues[118] = d118
					ps649.OverlayValues[171] = d171
					ps649.OverlayValues[172] = d172
					ps649.OverlayValues[173] = d173
					ps649.OverlayValues[174] = d174
					ps649.OverlayValues[175] = d175
					ps649.OverlayValues[176] = d176
					ps649.OverlayValues[177] = d177
					ps649.OverlayValues[178] = d178
					ps649.OverlayValues[179] = d179
					ps649.OverlayValues[180] = d180
					ps649.OverlayValues[181] = d181
					ps649.OverlayValues[182] = d182
					ps649.OverlayValues[183] = d183
					ps649.OverlayValues[184] = d184
					ps649.OverlayValues[185] = d185
					ps649.OverlayValues[186] = d186
					ps649.OverlayValues[187] = d187
					ps649.OverlayValues[188] = d188
					ps649.OverlayValues[189] = d189
					ps649.OverlayValues[191] = d191
					ps649.OverlayValues[192] = d192
					ps649.OverlayValues[193] = d193
					ps649.OverlayValues[194] = d194
					ps649.OverlayValues[195] = d195
					ps649.OverlayValues[196] = d196
					ps649.OverlayValues[197] = d197
					ps649.OverlayValues[198] = d198
					ps649.OverlayValues[200] = d200
					ps649.OverlayValues[201] = d201
					ps649.OverlayValues[202] = d202
					ps649.OverlayValues[287] = d287
					ps649.OverlayValues[288] = d288
					ps649.OverlayValues[289] = d289
					ps649.OverlayValues[290] = d290
					ps649.OverlayValues[291] = d291
					ps649.OverlayValues[294] = d294
					ps649.OverlayValues[295] = d295
					ps649.OverlayValues[385] = d385
					ps649.OverlayValues[386] = d386
					ps649.OverlayValues[387] = d387
					ps649.OverlayValues[388] = d388
					ps649.OverlayValues[389] = d389
					ps649.OverlayValues[390] = d390
					ps649.OverlayValues[391] = d391
					ps649.OverlayValues[392] = d392
					ps649.OverlayValues[393] = d393
					ps649.OverlayValues[394] = d394
					ps649.OverlayValues[395] = d395
					ps649.OverlayValues[396] = d396
					ps649.OverlayValues[397] = d397
					ps649.OverlayValues[399] = d399
					ps649.OverlayValues[400] = d400
					ps649.OverlayValues[401] = d401
					ps649.OverlayValues[402] = d402
					ps649.OverlayValues[403] = d403
					ps649.OverlayValues[404] = d404
					ps649.OverlayValues[405] = d405
					ps649.OverlayValues[407] = d407
					ps649.OverlayValues[409] = d409
					ps649.OverlayValues[410] = d410
					ps649.OverlayValues[413] = d413
					ps649.OverlayValues[525] = d525
					ps649.OverlayValues[526] = d526
					ps649.OverlayValues[527] = d527
					ps649.OverlayValues[646] = d646
					ps649.OverlayValues[647] = d647
					ps649.OverlayValues[648] = d648
					ps649.PhiValues = make([]JITValueDesc, 1)
					d650 = d646
					ps649.PhiValues[0] = d650
					if ps649.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps649)
					return result
				}
				ps651 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps651)
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
