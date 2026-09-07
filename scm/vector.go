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
				var d45 JITValueDesc
				_ = d45
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d79 JITValueDesc
				_ = d79
				var d145 JITValueDesc
				_ = d145
				var d146 JITValueDesc
				_ = d146
				var d147 JITValueDesc
				_ = d147
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d150 JITValueDesc
				_ = d150
				var d152 JITValueDesc
				_ = d152
				var d153 JITValueDesc
				_ = d153
				var d154 JITValueDesc
				_ = d154
				var d155 JITValueDesc
				_ = d155
				var d156 JITValueDesc
				_ = d156
				var d157 JITValueDesc
				_ = d157
				var d158 JITValueDesc
				_ = d158
				var d159 JITValueDesc
				_ = d159
				var d160 JITValueDesc
				_ = d160
				var d163 JITValueDesc
				_ = d163
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d269 JITValueDesc
				_ = d269
				var d270 JITValueDesc
				_ = d270
				var d271 JITValueDesc
				_ = d271
				var d272 JITValueDesc
				_ = d272
				var d273 JITValueDesc
				_ = d273
				var d274 JITValueDesc
				_ = d274
				var d275 JITValueDesc
				_ = d275
				var d276 JITValueDesc
				_ = d276
				var d277 JITValueDesc
				_ = d277
				var d278 JITValueDesc
				_ = d278
				var d279 JITValueDesc
				_ = d279
				var d280 JITValueDesc
				_ = d280
				var d281 JITValueDesc
				_ = d281
				var d282 JITValueDesc
				_ = d282
				var d283 JITValueDesc
				_ = d283
				var d284 JITValueDesc
				_ = d284
				var d285 JITValueDesc
				_ = d285
				var d286 JITValueDesc
				_ = d286
				var d287 JITValueDesc
				_ = d287
				var d289 JITValueDesc
				_ = d289
				var d290 JITValueDesc
				_ = d290
				var d291 JITValueDesc
				_ = d291
				var d292 JITValueDesc
				_ = d292
				var d293 JITValueDesc
				_ = d293
				var d294 JITValueDesc
				_ = d294
				var d295 JITValueDesc
				_ = d295
				var d296 JITValueDesc
				_ = d296
				var d298 JITValueDesc
				_ = d298
				var d299 JITValueDesc
				_ = d299
				var d300 JITValueDesc
				_ = d300
				var d465 JITValueDesc
				_ = d465
				var d466 JITValueDesc
				_ = d466
				var d467 JITValueDesc
				_ = d467
				var d468 JITValueDesc
				_ = d468
				var d469 JITValueDesc
				_ = d469
				var d472 JITValueDesc
				_ = d472
				var d473 JITValueDesc
				_ = d473
				var d650 JITValueDesc
				_ = d650
				var d651 JITValueDesc
				_ = d651
				var d652 JITValueDesc
				_ = d652
				var d653 JITValueDesc
				_ = d653
				var d654 JITValueDesc
				_ = d654
				var d655 JITValueDesc
				_ = d655
				var d656 JITValueDesc
				_ = d656
				var d657 JITValueDesc
				_ = d657
				var d658 JITValueDesc
				_ = d658
				var d659 JITValueDesc
				_ = d659
				var d660 JITValueDesc
				_ = d660
				var d661 JITValueDesc
				_ = d661
				var d662 JITValueDesc
				_ = d662
				var d664 JITValueDesc
				_ = d664
				var d665 JITValueDesc
				_ = d665
				var d666 JITValueDesc
				_ = d666
				var d667 JITValueDesc
				_ = d667
				var d668 JITValueDesc
				_ = d668
				var d669 JITValueDesc
				_ = d669
				var d670 JITValueDesc
				_ = d670
				var d672 JITValueDesc
				_ = d672
				var d674 JITValueDesc
				_ = d674
				var d784 JITValueDesc
				_ = d784
				var d787 JITValueDesc
				_ = d787
				var d899 JITValueDesc
				_ = d899
				var d900 JITValueDesc
				_ = d900
				var d901 JITValueDesc
				_ = d901
				var d1134 JITValueDesc
				_ = d1134
				var d1135 JITValueDesc
				_ = d1135
				var d1136 JITValueDesc
				_ = d1136
				var d1138 JITValueDesc
				_ = d1138
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
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r6, Condition: CondSignedGreater}
						ctx.BindReg(r6, &d21)
					}
					ctx.FreeDesc(&d20)
					d22 = d21
					ctx.EnsureDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					ctx.EmitJump(d22.Condition, lbl2)
					ctx.EmitJmp(lbl16)
					snap26 := d8
					snap27 := d9
					snap28 := d10
					snap29 := d11
					snap30 := d12
					snap31 := d13
					snap32 := d14
					snap33 := d15
					snap34 := d16
					snap35 := d17
					snap36 := d18
					snap37 := d19
					snap38 := d20
					snap39 := d21
					snap40 := d22
					snap41 := d25
					alloc42 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc42)
					d8 = snap26
					d9 = snap27
					d10 = snap28
					d11 = snap29
					d12 = snap30
					d13 = snap31
					d14 = snap32
					d15 = snap33
					d16 = snap34
					d17 = snap35
					d18 = snap36
					d19 = snap37
					d20 = snap38
					d21 = snap39
					d22 = snap40
					d25 = snap41
					ctx.MarkLabel(lbl16)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc42)
					d8 = snap26
					d9 = snap27
					d10 = snap28
					d11 = snap29
					d12 = snap30
					d13 = snap31
					d14 = snap32
					d15 = snap33
					d16 = snap34
					d17 = snap35
					d18 = snap36
					d19 = snap37
					d20 = snap38
					d21 = snap39
					d22 = snap40
					d25 = snap41
					ps43 := PhiState{General: true}
					ps43.OverlayValues = make([]JITValueDesc, 26)
					ps43.OverlayValues[8] = d8
					ps43.OverlayValues[9] = d9
					ps43.OverlayValues[10] = d10
					ps43.OverlayValues[11] = d11
					ps43.OverlayValues[12] = d12
					ps43.OverlayValues[13] = d13
					ps43.OverlayValues[14] = d14
					ps43.OverlayValues[15] = d15
					ps43.OverlayValues[16] = d16
					ps43.OverlayValues[17] = d17
					ps43.OverlayValues[18] = d18
					ps43.OverlayValues[19] = d19
					ps43.OverlayValues[20] = d20
					ps43.OverlayValues[21] = d21
					ps43.OverlayValues[22] = d22
					ps43.OverlayValues[25] = d25
					ps44 := PhiState{General: true}
					ps44.OverlayValues = make([]JITValueDesc, 26)
					ps44.OverlayValues[8] = d8
					ps44.OverlayValues[9] = d9
					ps44.OverlayValues[10] = d10
					ps44.OverlayValues[11] = d11
					ps44.OverlayValues[12] = d12
					ps44.OverlayValues[13] = d13
					ps44.OverlayValues[14] = d14
					ps44.OverlayValues[15] = d15
					ps44.OverlayValues[16] = d16
					ps44.OverlayValues[17] = d17
					ps44.OverlayValues[18] = d18
					ps44.OverlayValues[19] = d19
					ps44.OverlayValues[20] = d20
					ps44.OverlayValues[21] = d21
					ps44.OverlayValues[22] = d22
					ps44.OverlayValues[25] = d25
					ps44.PhiValues = make([]JITValueDesc, 1)
					d45 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
					ps44.PhiValues[0] = d45
					snap46 := d8
					snap47 := d9
					snap48 := d10
					snap49 := d11
					snap50 := d12
					snap51 := d13
					snap52 := d14
					snap53 := d15
					snap54 := d16
					snap55 := d17
					snap56 := d18
					snap57 := d19
					snap58 := d20
					snap59 := d21
					snap60 := d22
					snap61 := d25
					snap62 := d45
					alloc63 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps44)
					}
					ctx.RestoreAllocState(alloc63)
					d8 = snap46
					d9 = snap47
					d10 = snap48
					d11 = snap49
					d12 = snap50
					d13 = snap51
					d14 = snap52
					d15 = snap53
					d16 = snap54
					d17 = snap55
					d18 = snap56
					d19 = snap57
					d20 = snap58
					d21 = snap59
					d22 = snap60
					d25 = snap61
					d45 = snap62
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps43)
					}
					return result
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					ctx.ReclaimUntrackedRegs()
					d64 = args[2]
					d64.ID = 0
					d66 = d64
					ctx.SyncDesc(&d66)
					if d66.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d66.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d66.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d66 = tmpScalar
					}
					d66 = JITPrepareScmerGoArg(ctx, d66)
					if d66.Loc != LocRegPair && d66.Loc != LocStackPair && d66.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d65 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d66}, 2)
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d65.Imm)
						ptrWord, _ := d65.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d65.Imm.String())))
						d65 = tmpPair
					} else if d65.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d65.Type, Reg: ctx.AllocRegExcept(d65.Reg), Reg2: ctx.AllocRegExcept(d65.Reg)}
						switch d65.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d65)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d65)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d65)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d65)
						d65 = tmpPair
					}
					if d65.Loc != LocRegPair && d65.Loc != LocStackPair && d65.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d65)
					d67 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d65}, 2)
					d67.NoHeapPointer = false
					ctx.BindReg(d67.Reg, &d67)
					ctx.BindReg(d67.Reg2, &d67)
					ctx.StabilizeDescForControlFlow(&d67)
					if ps.General {
						ctx.SyncDesc(&d67)
						if d67.Loc == LocReg {
							ctx.ProtectReg(d67.Reg)
						} else if d67.Loc == LocRegPair {
							ctx.ProtectReg(d67.Reg)
							ctx.ProtectReg(d67.Reg2)
						}
						d68 = d67
						if d68.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d68)
						if d68.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d68, int32(bbs[2].PhiBase)+int32(0), 2)
						} else if d68.Loc == LocInputPair {
							ctx.EnsureDesc(&d68)
							ctx.EmitStoreScmerToStack(d68, int32(bbs[2].PhiBase)+int32(0))
						} else if d68.Loc == LocRegPair || d68.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d68, int32(bbs[2].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d68)
							ctx.EmitStoreToStack(d68, int32(bbs[2].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
						}
						if d67.Loc == LocReg {
							ctx.UnprotectReg(d67.Reg)
						} else if d67.Loc == LocRegPair {
							ctx.UnprotectReg(d67.Reg)
							ctx.UnprotectReg(d67.Reg2)
						}
					}
					ps69 := PhiState{General: ps.General}
					ps69.OverlayValues = make([]JITValueDesc, 69)
					ps69.OverlayValues[8] = d8
					ps69.OverlayValues[9] = d9
					ps69.OverlayValues[10] = d10
					ps69.OverlayValues[11] = d11
					ps69.OverlayValues[12] = d12
					ps69.OverlayValues[13] = d13
					ps69.OverlayValues[14] = d14
					ps69.OverlayValues[15] = d15
					ps69.OverlayValues[16] = d16
					ps69.OverlayValues[17] = d17
					ps69.OverlayValues[18] = d18
					ps69.OverlayValues[19] = d19
					ps69.OverlayValues[20] = d20
					ps69.OverlayValues[21] = d21
					ps69.OverlayValues[22] = d22
					ps69.OverlayValues[25] = d25
					ps69.OverlayValues[45] = d45
					ps69.OverlayValues[64] = d64
					ps69.OverlayValues[65] = d65
					ps69.OverlayValues[66] = d66
					ps69.OverlayValues[67] = d67
					ps69.OverlayValues[68] = d68
					ps69.PhiValues = make([]JITValueDesc, 1)
					d70 = d67
					ps69.PhiValues[0] = d70
					if ps69.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps69)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d71 := ps.PhiValues[0]
							ctx.EnsureDesc(&d71)
							ctx.EmitStoreScmerToStack(d71, int32(bbs[2].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d8 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d8)
					var d72 JITValueDesc
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d72 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d72.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d72.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d72.Reg, &d72)
						ctx.BindReg(d72.Reg2, &d72)
					} else {
						d72 = d8
					}
					d73 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("COSINE")}
					var d74 JITValueDesc
					if d73.Loc == LocImm {
						ctx.TrackImm(d73.Imm)
						ptrWord, _ := d73.Imm.RawWords()
						d74 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d74.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d74.Reg2, uint64(len(d73.Imm.String())))
						ctx.BindReg(d74.Reg, &d74)
						ctx.BindReg(d74.Reg2, &d74)
					} else {
						d74 = d73
					}
					d75 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d72, d74}, 1)
					ctx.EmitAndRegImm32(d75.Reg, 1)
					d75.Type = tagBool
					ctx.BindReg(d75.Reg, &d75)
					d76 = d75
					ctx.EnsureDesc(&d76)
					if d76.Loc != LocImm && d76.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d76.Loc == LocImm {
						if d76.Imm.Bool() {
							if ps.General {
							}
							ps77 := PhiState{General: ps.General}
							ps77.OverlayValues = make([]JITValueDesc, 77)
							ps77.OverlayValues[8] = d8
							ps77.OverlayValues[9] = d9
							ps77.OverlayValues[10] = d10
							ps77.OverlayValues[11] = d11
							ps77.OverlayValues[12] = d12
							ps77.OverlayValues[13] = d13
							ps77.OverlayValues[14] = d14
							ps77.OverlayValues[15] = d15
							ps77.OverlayValues[16] = d16
							ps77.OverlayValues[17] = d17
							ps77.OverlayValues[18] = d18
							ps77.OverlayValues[19] = d19
							ps77.OverlayValues[20] = d20
							ps77.OverlayValues[21] = d21
							ps77.OverlayValues[22] = d22
							ps77.OverlayValues[25] = d25
							ps77.OverlayValues[45] = d45
							ps77.OverlayValues[64] = d64
							ps77.OverlayValues[65] = d65
							ps77.OverlayValues[66] = d66
							ps77.OverlayValues[67] = d67
							ps77.OverlayValues[68] = d68
							ps77.OverlayValues[70] = d70
							ps77.OverlayValues[71] = d71
							ps77.OverlayValues[72] = d72
							ps77.OverlayValues[73] = d73
							ps77.OverlayValues[74] = d74
							ps77.OverlayValues[75] = d75
							ps77.OverlayValues[76] = d76
							return bbs[3].RenderPS(ps77)
						}
						if ps.General {
						}
						ps78 := PhiState{General: ps.General}
						ps78.OverlayValues = make([]JITValueDesc, 77)
						ps78.OverlayValues[8] = d8
						ps78.OverlayValues[9] = d9
						ps78.OverlayValues[10] = d10
						ps78.OverlayValues[11] = d11
						ps78.OverlayValues[12] = d12
						ps78.OverlayValues[13] = d13
						ps78.OverlayValues[14] = d14
						ps78.OverlayValues[15] = d15
						ps78.OverlayValues[16] = d16
						ps78.OverlayValues[17] = d17
						ps78.OverlayValues[18] = d18
						ps78.OverlayValues[19] = d19
						ps78.OverlayValues[20] = d20
						ps78.OverlayValues[21] = d21
						ps78.OverlayValues[22] = d22
						ps78.OverlayValues[25] = d25
						ps78.OverlayValues[45] = d45
						ps78.OverlayValues[64] = d64
						ps78.OverlayValues[65] = d65
						ps78.OverlayValues[66] = d66
						ps78.OverlayValues[67] = d67
						ps78.OverlayValues[68] = d68
						ps78.OverlayValues[70] = d70
						ps78.OverlayValues[71] = d71
						ps78.OverlayValues[72] = d72
						ps78.OverlayValues[73] = d73
						ps78.OverlayValues[74] = d74
						ps78.OverlayValues[75] = d75
						ps78.OverlayValues[76] = d76
						return bbs[5].RenderPS(ps78)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d79 := ps.PhiValues[0]
							ctx.EnsureDesc(&d79)
							ctx.EmitStoreScmerToStack(d79, int32(bbs[2].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d76.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					snap80 := d8
					snap81 := d9
					snap82 := d10
					snap83 := d11
					snap84 := d12
					snap85 := d13
					snap86 := d14
					snap87 := d15
					snap88 := d16
					snap89 := d17
					snap90 := d18
					snap91 := d19
					snap92 := d20
					snap93 := d21
					snap94 := d22
					snap95 := d25
					snap96 := d45
					snap97 := d64
					snap98 := d65
					snap99 := d66
					snap100 := d67
					snap101 := d68
					snap102 := d70
					snap103 := d71
					snap104 := d72
					snap105 := d73
					snap106 := d74
					snap107 := d75
					snap108 := d76
					snap109 := d79
					alloc110 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc110)
					d8 = snap80
					d9 = snap81
					d10 = snap82
					d11 = snap83
					d12 = snap84
					d13 = snap85
					d14 = snap86
					d15 = snap87
					d16 = snap88
					d17 = snap89
					d18 = snap90
					d19 = snap91
					d20 = snap92
					d21 = snap93
					d22 = snap94
					d25 = snap95
					d45 = snap96
					d64 = snap97
					d65 = snap98
					d66 = snap99
					d67 = snap100
					d68 = snap101
					d70 = snap102
					d71 = snap103
					d72 = snap104
					d73 = snap105
					d74 = snap106
					d75 = snap107
					d76 = snap108
					d79 = snap109
					ctx.RestoreAllocState(alloc110)
					d8 = snap80
					d9 = snap81
					d10 = snap82
					d11 = snap83
					d12 = snap84
					d13 = snap85
					d14 = snap86
					d15 = snap87
					d16 = snap88
					d17 = snap89
					d18 = snap90
					d19 = snap91
					d20 = snap92
					d21 = snap93
					d22 = snap94
					d25 = snap95
					d45 = snap96
					d64 = snap97
					d65 = snap98
					d66 = snap99
					d67 = snap100
					d68 = snap101
					d70 = snap102
					d71 = snap103
					d72 = snap104
					d73 = snap105
					d74 = snap106
					d75 = snap107
					d76 = snap108
					d79 = snap109
					ps111 := PhiState{General: true}
					ps111.OverlayValues = make([]JITValueDesc, 80)
					ps111.OverlayValues[8] = d8
					ps111.OverlayValues[9] = d9
					ps111.OverlayValues[10] = d10
					ps111.OverlayValues[11] = d11
					ps111.OverlayValues[12] = d12
					ps111.OverlayValues[13] = d13
					ps111.OverlayValues[14] = d14
					ps111.OverlayValues[15] = d15
					ps111.OverlayValues[16] = d16
					ps111.OverlayValues[17] = d17
					ps111.OverlayValues[18] = d18
					ps111.OverlayValues[19] = d19
					ps111.OverlayValues[20] = d20
					ps111.OverlayValues[21] = d21
					ps111.OverlayValues[22] = d22
					ps111.OverlayValues[25] = d25
					ps111.OverlayValues[45] = d45
					ps111.OverlayValues[64] = d64
					ps111.OverlayValues[65] = d65
					ps111.OverlayValues[66] = d66
					ps111.OverlayValues[67] = d67
					ps111.OverlayValues[68] = d68
					ps111.OverlayValues[70] = d70
					ps111.OverlayValues[71] = d71
					ps111.OverlayValues[72] = d72
					ps111.OverlayValues[73] = d73
					ps111.OverlayValues[74] = d74
					ps111.OverlayValues[75] = d75
					ps111.OverlayValues[76] = d76
					ps111.OverlayValues[79] = d79
					ps112 := PhiState{General: true}
					ps112.OverlayValues = make([]JITValueDesc, 80)
					ps112.OverlayValues[8] = d8
					ps112.OverlayValues[9] = d9
					ps112.OverlayValues[10] = d10
					ps112.OverlayValues[11] = d11
					ps112.OverlayValues[12] = d12
					ps112.OverlayValues[13] = d13
					ps112.OverlayValues[14] = d14
					ps112.OverlayValues[15] = d15
					ps112.OverlayValues[16] = d16
					ps112.OverlayValues[17] = d17
					ps112.OverlayValues[18] = d18
					ps112.OverlayValues[19] = d19
					ps112.OverlayValues[20] = d20
					ps112.OverlayValues[21] = d21
					ps112.OverlayValues[22] = d22
					ps112.OverlayValues[25] = d25
					ps112.OverlayValues[45] = d45
					ps112.OverlayValues[64] = d64
					ps112.OverlayValues[65] = d65
					ps112.OverlayValues[66] = d66
					ps112.OverlayValues[67] = d67
					ps112.OverlayValues[68] = d68
					ps112.OverlayValues[70] = d70
					ps112.OverlayValues[71] = d71
					ps112.OverlayValues[72] = d72
					ps112.OverlayValues[73] = d73
					ps112.OverlayValues[74] = d74
					ps112.OverlayValues[75] = d75
					ps112.OverlayValues[76] = d76
					ps112.OverlayValues[79] = d79
					snap113 := d8
					snap114 := d9
					snap115 := d10
					snap116 := d11
					snap117 := d12
					snap118 := d13
					snap119 := d14
					snap120 := d15
					snap121 := d16
					snap122 := d17
					snap123 := d18
					snap124 := d19
					snap125 := d20
					snap126 := d21
					snap127 := d22
					snap128 := d25
					snap129 := d45
					snap130 := d64
					snap131 := d65
					snap132 := d66
					snap133 := d67
					snap134 := d68
					snap135 := d70
					snap136 := d71
					snap137 := d72
					snap138 := d73
					snap139 := d74
					snap140 := d75
					snap141 := d76
					snap142 := d79
					alloc143 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps112)
					}
					ctx.RestoreAllocState(alloc143)
					d8 = snap113
					d9 = snap114
					d10 = snap115
					d11 = snap116
					d12 = snap117
					d13 = snap118
					d14 = snap119
					d15 = snap120
					d16 = snap121
					d17 = snap122
					d18 = snap123
					d19 = snap124
					d20 = snap125
					d21 = snap126
					d22 = snap127
					d25 = snap128
					d45 = snap129
					d64 = snap130
					d65 = snap131
					d66 = snap132
					d67 = snap133
					d68 = snap134
					d70 = snap135
					d71 = snap136
					d72 = snap137
					d73 = snap138
					d74 = snap139
					d75 = snap140
					d76 = snap141
					d79 = snap142
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps111)
					}
					return result
					ctx.FreeDesc(&d75)
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
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
					ps144 := PhiState{General: ps.General}
					ps144.OverlayValues = make([]JITValueDesc, 80)
					ps144.OverlayValues[8] = d8
					ps144.OverlayValues[9] = d9
					ps144.OverlayValues[10] = d10
					ps144.OverlayValues[11] = d11
					ps144.OverlayValues[12] = d12
					ps144.OverlayValues[13] = d13
					ps144.OverlayValues[14] = d14
					ps144.OverlayValues[15] = d15
					ps144.OverlayValues[16] = d16
					ps144.OverlayValues[17] = d17
					ps144.OverlayValues[18] = d18
					ps144.OverlayValues[19] = d19
					ps144.OverlayValues[20] = d20
					ps144.OverlayValues[21] = d21
					ps144.OverlayValues[22] = d22
					ps144.OverlayValues[25] = d25
					ps144.OverlayValues[45] = d45
					ps144.OverlayValues[64] = d64
					ps144.OverlayValues[65] = d65
					ps144.OverlayValues[66] = d66
					ps144.OverlayValues[67] = d67
					ps144.OverlayValues[68] = d68
					ps144.OverlayValues[70] = d70
					ps144.OverlayValues[71] = d71
					ps144.OverlayValues[72] = d72
					ps144.OverlayValues[73] = d73
					ps144.OverlayValues[74] = d74
					ps144.OverlayValues[75] = d75
					ps144.OverlayValues[76] = d76
					ps144.OverlayValues[79] = d79
					ps144.PhiValues = make([]JITValueDesc, 4)
					d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps144.PhiValues[0] = d145
					d146 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps144.PhiValues[1] = d146
					d147 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps144.PhiValues[2] = d147
					d148 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps144.PhiValues[3] = d148
					if ps144.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps144)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d149 := ps.PhiValues[0]
							ctx.EnsureDesc(&d149)
							ctx.EmitStoreToStack(d149, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
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
						d150 := JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d150)
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
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
					ps151 := PhiState{General: ps.General}
					ps151.OverlayValues = make([]JITValueDesc, 151)
					ps151.OverlayValues[8] = d8
					ps151.OverlayValues[9] = d9
					ps151.OverlayValues[10] = d10
					ps151.OverlayValues[11] = d11
					ps151.OverlayValues[12] = d12
					ps151.OverlayValues[13] = d13
					ps151.OverlayValues[14] = d14
					ps151.OverlayValues[15] = d15
					ps151.OverlayValues[16] = d16
					ps151.OverlayValues[17] = d17
					ps151.OverlayValues[18] = d18
					ps151.OverlayValues[19] = d19
					ps151.OverlayValues[20] = d20
					ps151.OverlayValues[21] = d21
					ps151.OverlayValues[22] = d22
					ps151.OverlayValues[25] = d25
					ps151.OverlayValues[45] = d45
					ps151.OverlayValues[64] = d64
					ps151.OverlayValues[65] = d65
					ps151.OverlayValues[66] = d66
					ps151.OverlayValues[67] = d67
					ps151.OverlayValues[68] = d68
					ps151.OverlayValues[70] = d70
					ps151.OverlayValues[71] = d71
					ps151.OverlayValues[72] = d72
					ps151.OverlayValues[73] = d73
					ps151.OverlayValues[74] = d74
					ps151.OverlayValues[75] = d75
					ps151.OverlayValues[76] = d76
					ps151.OverlayValues[79] = d79
					ps151.OverlayValues[145] = d145
					ps151.OverlayValues[146] = d146
					ps151.OverlayValues[147] = d147
					ps151.OverlayValues[148] = d148
					ps151.OverlayValues[149] = d149
					ps151.OverlayValues[150] = d150
					ps151.PhiValues = make([]JITValueDesc, 2)
					d152 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps151.PhiValues[0] = d152
					d153 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps151.PhiValues[1] = d153
					if ps151.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps151)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d154 := ps.PhiValues[0]
							ctx.EnsureDesc(&d154)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d154)
							} else {
								ctx.EmitStoreToStack(d154, int32(bbs[6].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d155 := ps.PhiValues[1]
							ctx.EnsureDesc(&d155)
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d155)
							} else {
								ctx.EmitStoreToStack(d155, int32(bbs[6].PhiBase)+int32(16))
							}
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d156 := ps.PhiValues[2]
							ctx.EnsureDesc(&d156)
							if phiHomeOK4 {
								ctx.EmitMovToReg(r2, d156)
							} else {
								ctx.EmitStoreToStack(d156, int32(bbs[6].PhiBase)+int32(32))
							}
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d157 := ps.PhiValues[3]
							ctx.EnsureDesc(&d157)
							if phiHomeOK5 {
								ctx.EmitMovToReg(r3, d157)
							} else {
								ctx.EmitStoreToStack(d157, int32(bbs[6].PhiBase)+int32(48))
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
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
					var d158 JITValueDesc
					if d17.SliceSizeKnown {
						d158 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d158 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d158 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d158 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d158 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d158)
					ctx.EnsureDescsTogether(&d13, &d158)
					var d159 JITValueDesc
					if d13.Loc == LocImm && d158.Loc == LocImm {
						d159 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d158.Imm.Int())}
					} else if d158.Loc == LocImm {
						r7 := ctx.AllocRegExcept(d13.Reg)
						if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d158.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d158.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, RegR11)
						}
						d159 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedLess}
						ctx.BindReg(r7, &d159)
					} else if d13.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d158.Reg)
						d159 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondSignedLess}
						ctx.BindReg(r8, &d159)
					} else {
						r9 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpInt64(d13.Reg, d158.Reg)
						d159 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLess}
						ctx.BindReg(r9, &d159)
					}
					ctx.FreeDesc(&d158)
					d160 = d159
					ctx.EnsureDesc(&d160)
					if d160.Loc != LocImm && d160.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d160.Loc == LocImm {
						if d160.Imm.Bool() {
							if ps.General {
							}
							ps161 := PhiState{General: ps.General}
							ps161.OverlayValues = make([]JITValueDesc, 161)
							ps161.OverlayValues[8] = d8
							ps161.OverlayValues[9] = d9
							ps161.OverlayValues[10] = d10
							ps161.OverlayValues[11] = d11
							ps161.OverlayValues[12] = d12
							ps161.OverlayValues[13] = d13
							ps161.OverlayValues[14] = d14
							ps161.OverlayValues[15] = d15
							ps161.OverlayValues[16] = d16
							ps161.OverlayValues[17] = d17
							ps161.OverlayValues[18] = d18
							ps161.OverlayValues[19] = d19
							ps161.OverlayValues[20] = d20
							ps161.OverlayValues[21] = d21
							ps161.OverlayValues[22] = d22
							ps161.OverlayValues[25] = d25
							ps161.OverlayValues[45] = d45
							ps161.OverlayValues[64] = d64
							ps161.OverlayValues[65] = d65
							ps161.OverlayValues[66] = d66
							ps161.OverlayValues[67] = d67
							ps161.OverlayValues[68] = d68
							ps161.OverlayValues[70] = d70
							ps161.OverlayValues[71] = d71
							ps161.OverlayValues[72] = d72
							ps161.OverlayValues[73] = d73
							ps161.OverlayValues[74] = d74
							ps161.OverlayValues[75] = d75
							ps161.OverlayValues[76] = d76
							ps161.OverlayValues[79] = d79
							ps161.OverlayValues[145] = d145
							ps161.OverlayValues[146] = d146
							ps161.OverlayValues[147] = d147
							ps161.OverlayValues[148] = d148
							ps161.OverlayValues[149] = d149
							ps161.OverlayValues[150] = d150
							ps161.OverlayValues[152] = d152
							ps161.OverlayValues[153] = d153
							ps161.OverlayValues[154] = d154
							ps161.OverlayValues[155] = d155
							ps161.OverlayValues[156] = d156
							ps161.OverlayValues[157] = d157
							ps161.OverlayValues[158] = d158
							ps161.OverlayValues[159] = d159
							ps161.OverlayValues[160] = d160
							return bbs[9].RenderPS(ps161)
						}
						if ps.General {
						}
						ps162 := PhiState{General: ps.General}
						ps162.OverlayValues = make([]JITValueDesc, 161)
						ps162.OverlayValues[8] = d8
						ps162.OverlayValues[9] = d9
						ps162.OverlayValues[10] = d10
						ps162.OverlayValues[11] = d11
						ps162.OverlayValues[12] = d12
						ps162.OverlayValues[13] = d13
						ps162.OverlayValues[14] = d14
						ps162.OverlayValues[15] = d15
						ps162.OverlayValues[16] = d16
						ps162.OverlayValues[17] = d17
						ps162.OverlayValues[18] = d18
						ps162.OverlayValues[19] = d19
						ps162.OverlayValues[20] = d20
						ps162.OverlayValues[21] = d21
						ps162.OverlayValues[22] = d22
						ps162.OverlayValues[25] = d25
						ps162.OverlayValues[45] = d45
						ps162.OverlayValues[64] = d64
						ps162.OverlayValues[65] = d65
						ps162.OverlayValues[66] = d66
						ps162.OverlayValues[67] = d67
						ps162.OverlayValues[68] = d68
						ps162.OverlayValues[70] = d70
						ps162.OverlayValues[71] = d71
						ps162.OverlayValues[72] = d72
						ps162.OverlayValues[73] = d73
						ps162.OverlayValues[74] = d74
						ps162.OverlayValues[75] = d75
						ps162.OverlayValues[76] = d76
						ps162.OverlayValues[79] = d79
						ps162.OverlayValues[145] = d145
						ps162.OverlayValues[146] = d146
						ps162.OverlayValues[147] = d147
						ps162.OverlayValues[148] = d148
						ps162.OverlayValues[149] = d149
						ps162.OverlayValues[150] = d150
						ps162.OverlayValues[152] = d152
						ps162.OverlayValues[153] = d153
						ps162.OverlayValues[154] = d154
						ps162.OverlayValues[155] = d155
						ps162.OverlayValues[156] = d156
						ps162.OverlayValues[157] = d157
						ps162.OverlayValues[158] = d158
						ps162.OverlayValues[159] = d159
						ps162.OverlayValues[160] = d160
						return bbs[8].RenderPS(ps162)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d163 := ps.PhiValues[0]
							ctx.EnsureDesc(&d163)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d163)
							} else {
								ctx.EmitStoreToStack(d163, int32(bbs[6].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d164 := ps.PhiValues[1]
							ctx.EnsureDesc(&d164)
							if phiHomeOK3 {
								ctx.EmitMovToReg(r1, d164)
							} else {
								ctx.EmitStoreToStack(d164, int32(bbs[6].PhiBase)+int32(16))
							}
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d165 := ps.PhiValues[2]
							ctx.EnsureDesc(&d165)
							if phiHomeOK4 {
								ctx.EmitMovToReg(r2, d165)
							} else {
								ctx.EmitStoreToStack(d165, int32(bbs[6].PhiBase)+int32(32))
							}
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d166 := ps.PhiValues[3]
							ctx.EnsureDesc(&d166)
							if phiHomeOK5 {
								ctx.EmitMovToReg(r3, d166)
							} else {
								ctx.EmitStoreToStack(d166, int32(bbs[6].PhiBase)+int32(48))
							}
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitJump(d160.Condition, lbl10)
					snap167 := d8
					snap168 := d9
					snap169 := d10
					snap170 := d11
					snap171 := d12
					snap172 := d13
					snap173 := d14
					snap174 := d15
					snap175 := d16
					snap176 := d17
					snap177 := d18
					snap178 := d19
					snap179 := d20
					snap180 := d21
					snap181 := d22
					snap182 := d25
					snap183 := d45
					snap184 := d64
					snap185 := d65
					snap186 := d66
					snap187 := d67
					snap188 := d68
					snap189 := d70
					snap190 := d71
					snap191 := d72
					snap192 := d73
					snap193 := d74
					snap194 := d75
					snap195 := d76
					snap196 := d79
					snap197 := d145
					snap198 := d146
					snap199 := d147
					snap200 := d148
					snap201 := d149
					snap202 := d150
					snap203 := d152
					snap204 := d153
					snap205 := d154
					snap206 := d155
					snap207 := d156
					snap208 := d157
					snap209 := d158
					snap210 := d159
					snap211 := d160
					snap212 := d163
					snap213 := d164
					snap214 := d165
					snap215 := d166
					alloc216 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc216)
					d8 = snap167
					d9 = snap168
					d10 = snap169
					d11 = snap170
					d12 = snap171
					d13 = snap172
					d14 = snap173
					d15 = snap174
					d16 = snap175
					d17 = snap176
					d18 = snap177
					d19 = snap178
					d20 = snap179
					d21 = snap180
					d22 = snap181
					d25 = snap182
					d45 = snap183
					d64 = snap184
					d65 = snap185
					d66 = snap186
					d67 = snap187
					d68 = snap188
					d70 = snap189
					d71 = snap190
					d72 = snap191
					d73 = snap192
					d74 = snap193
					d75 = snap194
					d76 = snap195
					d79 = snap196
					d145 = snap197
					d146 = snap198
					d147 = snap199
					d148 = snap200
					d149 = snap201
					d150 = snap202
					d152 = snap203
					d153 = snap204
					d154 = snap205
					d155 = snap206
					d156 = snap207
					d157 = snap208
					d158 = snap209
					d159 = snap210
					d160 = snap211
					d163 = snap212
					d164 = snap213
					d165 = snap214
					d166 = snap215
					ctx.RestoreAllocState(alloc216)
					d8 = snap167
					d9 = snap168
					d10 = snap169
					d11 = snap170
					d12 = snap171
					d13 = snap172
					d14 = snap173
					d15 = snap174
					d16 = snap175
					d17 = snap176
					d18 = snap177
					d19 = snap178
					d20 = snap179
					d21 = snap180
					d22 = snap181
					d25 = snap182
					d45 = snap183
					d64 = snap184
					d65 = snap185
					d66 = snap186
					d67 = snap187
					d68 = snap188
					d70 = snap189
					d71 = snap190
					d72 = snap191
					d73 = snap192
					d74 = snap193
					d75 = snap194
					d76 = snap195
					d79 = snap196
					d145 = snap197
					d146 = snap198
					d147 = snap199
					d148 = snap200
					d149 = snap201
					d150 = snap202
					d152 = snap203
					d153 = snap204
					d154 = snap205
					d155 = snap206
					d156 = snap207
					d157 = snap208
					d158 = snap209
					d159 = snap210
					d160 = snap211
					d163 = snap212
					d164 = snap213
					d165 = snap214
					d166 = snap215
					ps217 := PhiState{General: true}
					ps217.OverlayValues = make([]JITValueDesc, 167)
					ps217.OverlayValues[8] = d8
					ps217.OverlayValues[9] = d9
					ps217.OverlayValues[10] = d10
					ps217.OverlayValues[11] = d11
					ps217.OverlayValues[12] = d12
					ps217.OverlayValues[13] = d13
					ps217.OverlayValues[14] = d14
					ps217.OverlayValues[15] = d15
					ps217.OverlayValues[16] = d16
					ps217.OverlayValues[17] = d17
					ps217.OverlayValues[18] = d18
					ps217.OverlayValues[19] = d19
					ps217.OverlayValues[20] = d20
					ps217.OverlayValues[21] = d21
					ps217.OverlayValues[22] = d22
					ps217.OverlayValues[25] = d25
					ps217.OverlayValues[45] = d45
					ps217.OverlayValues[64] = d64
					ps217.OverlayValues[65] = d65
					ps217.OverlayValues[66] = d66
					ps217.OverlayValues[67] = d67
					ps217.OverlayValues[68] = d68
					ps217.OverlayValues[70] = d70
					ps217.OverlayValues[71] = d71
					ps217.OverlayValues[72] = d72
					ps217.OverlayValues[73] = d73
					ps217.OverlayValues[74] = d74
					ps217.OverlayValues[75] = d75
					ps217.OverlayValues[76] = d76
					ps217.OverlayValues[79] = d79
					ps217.OverlayValues[145] = d145
					ps217.OverlayValues[146] = d146
					ps217.OverlayValues[147] = d147
					ps217.OverlayValues[148] = d148
					ps217.OverlayValues[149] = d149
					ps217.OverlayValues[150] = d150
					ps217.OverlayValues[152] = d152
					ps217.OverlayValues[153] = d153
					ps217.OverlayValues[154] = d154
					ps217.OverlayValues[155] = d155
					ps217.OverlayValues[156] = d156
					ps217.OverlayValues[157] = d157
					ps217.OverlayValues[158] = d158
					ps217.OverlayValues[159] = d159
					ps217.OverlayValues[160] = d160
					ps217.OverlayValues[163] = d163
					ps217.OverlayValues[164] = d164
					ps217.OverlayValues[165] = d165
					ps217.OverlayValues[166] = d166
					ps218 := PhiState{General: true}
					ps218.OverlayValues = make([]JITValueDesc, 167)
					ps218.OverlayValues[8] = d8
					ps218.OverlayValues[9] = d9
					ps218.OverlayValues[10] = d10
					ps218.OverlayValues[11] = d11
					ps218.OverlayValues[12] = d12
					ps218.OverlayValues[13] = d13
					ps218.OverlayValues[14] = d14
					ps218.OverlayValues[15] = d15
					ps218.OverlayValues[16] = d16
					ps218.OverlayValues[17] = d17
					ps218.OverlayValues[18] = d18
					ps218.OverlayValues[19] = d19
					ps218.OverlayValues[20] = d20
					ps218.OverlayValues[21] = d21
					ps218.OverlayValues[22] = d22
					ps218.OverlayValues[25] = d25
					ps218.OverlayValues[45] = d45
					ps218.OverlayValues[64] = d64
					ps218.OverlayValues[65] = d65
					ps218.OverlayValues[66] = d66
					ps218.OverlayValues[67] = d67
					ps218.OverlayValues[68] = d68
					ps218.OverlayValues[70] = d70
					ps218.OverlayValues[71] = d71
					ps218.OverlayValues[72] = d72
					ps218.OverlayValues[73] = d73
					ps218.OverlayValues[74] = d74
					ps218.OverlayValues[75] = d75
					ps218.OverlayValues[76] = d76
					ps218.OverlayValues[79] = d79
					ps218.OverlayValues[145] = d145
					ps218.OverlayValues[146] = d146
					ps218.OverlayValues[147] = d147
					ps218.OverlayValues[148] = d148
					ps218.OverlayValues[149] = d149
					ps218.OverlayValues[150] = d150
					ps218.OverlayValues[152] = d152
					ps218.OverlayValues[153] = d153
					ps218.OverlayValues[154] = d154
					ps218.OverlayValues[155] = d155
					ps218.OverlayValues[156] = d156
					ps218.OverlayValues[157] = d157
					ps218.OverlayValues[158] = d158
					ps218.OverlayValues[159] = d159
					ps218.OverlayValues[160] = d160
					ps218.OverlayValues[163] = d163
					ps218.OverlayValues[164] = d164
					ps218.OverlayValues[165] = d165
					ps218.OverlayValues[166] = d166
					snap219 := d8
					snap220 := d9
					snap221 := d10
					snap222 := d11
					snap223 := d12
					snap224 := d13
					snap225 := d14
					snap226 := d15
					snap227 := d16
					snap228 := d17
					snap229 := d18
					snap230 := d19
					snap231 := d20
					snap232 := d21
					snap233 := d22
					snap234 := d25
					snap235 := d45
					snap236 := d64
					snap237 := d65
					snap238 := d66
					snap239 := d67
					snap240 := d68
					snap241 := d70
					snap242 := d71
					snap243 := d72
					snap244 := d73
					snap245 := d74
					snap246 := d75
					snap247 := d76
					snap248 := d79
					snap249 := d145
					snap250 := d146
					snap251 := d147
					snap252 := d148
					snap253 := d149
					snap254 := d150
					snap255 := d152
					snap256 := d153
					snap257 := d154
					snap258 := d155
					snap259 := d156
					snap260 := d157
					snap261 := d158
					snap262 := d159
					snap263 := d160
					snap264 := d163
					snap265 := d164
					snap266 := d165
					snap267 := d166
					alloc268 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps218)
					}
					ctx.RestoreAllocState(alloc268)
					d8 = snap219
					d9 = snap220
					d10 = snap221
					d11 = snap222
					d12 = snap223
					d13 = snap224
					d14 = snap225
					d15 = snap226
					d16 = snap227
					d17 = snap228
					d18 = snap229
					d19 = snap230
					d20 = snap231
					d21 = snap232
					d22 = snap233
					d25 = snap234
					d45 = snap235
					d64 = snap236
					d65 = snap237
					d66 = snap238
					d67 = snap239
					d68 = snap240
					d70 = snap241
					d71 = snap242
					d72 = snap243
					d73 = snap244
					d74 = snap245
					d75 = snap246
					d76 = snap247
					d79 = snap248
					d145 = snap249
					d146 = snap250
					d147 = snap251
					d148 = snap252
					d149 = snap253
					d150 = snap254
					d152 = snap255
					d153 = snap256
					d154 = snap257
					d155 = snap258
					d156 = snap259
					d157 = snap260
					d158 = snap261
					d159 = snap262
					d160 = snap263
					d163 = snap264
					d164 = snap265
					d165 = snap266
					d166 = snap267
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps217)
					}
					return result
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					d270 = ctx.EmitSliceElementAddress(&d17, &d13, 16)
					ctx.EnsureDesc(&d270)
					r10 := ctx.AllocRegExcept(d270.Reg)
					ctx.EmitMovRegMem(r10, d270.Reg, 8)
					ctx.EmitMovRegMem(d270.Reg, d270.Reg, 0)
					d269 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d270.Reg, Reg2: r10}
					ctx.BindReg(d270.Reg, &d269)
					ctx.BindReg(r10, &d269)
					ctx.EnsureDesc(&d269)
					d271 = d269
					_ = d271
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d272 JITValueDesc
					if d271.Loc == LocImm {
						d272 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d271.Imm.Float())}
					} else if d271.Type == tagFloat && d271.Loc == LocReg {
						d272 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d271.Reg}
						ctx.BindReg(d271.Reg, &d272)
						ctx.BindReg(d271.Reg, &d272)
					} else if d271.Type == tagFloat && d271.Loc == LocRegPair {
						ctx.FreeReg(d271.Reg)
						d272 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d271.Reg2}
						ctx.BindReg(d271.Reg2, &d272)
						ctx.BindReg(d271.Reg2, &d272)
					} else {
						d272 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d271}, 1)
						d272.Type = tagFloat
						ctx.BindReg(d272.Reg, &d272)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d272)
					ctx.FreeDesc(&d269)
					ctx.EnsureDesc(&d13)
					d274 = ctx.EmitSliceElementAddress(&d19, &d13, 16)
					ctx.EnsureDesc(&d274)
					r11 := ctx.AllocRegExcept(d274.Reg)
					ctx.EmitMovRegMem(r11, d274.Reg, 8)
					ctx.EmitMovRegMem(d274.Reg, d274.Reg, 0)
					d273 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d274.Reg, Reg2: r11}
					ctx.BindReg(d274.Reg, &d273)
					ctx.BindReg(r11, &d273)
					ctx.EnsureDesc(&d273)
					d275 = d273
					_ = d275
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl18 := ctx.ReserveLabel()
					_ = lbl18
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d276 JITValueDesc
					if d275.Loc == LocImm {
						d276 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d275.Imm.Float())}
					} else if d275.Type == tagFloat && d275.Loc == LocReg {
						d276 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d275.Reg}
						ctx.BindReg(d275.Reg, &d276)
						ctx.BindReg(d275.Reg, &d276)
					} else if d275.Type == tagFloat && d275.Loc == LocRegPair {
						ctx.FreeReg(d275.Reg)
						d276 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d275.Reg2}
						ctx.BindReg(d275.Reg2, &d276)
						ctx.BindReg(d275.Reg2, &d276)
					} else {
						d276 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d275}, 1)
						d276.Type = tagFloat
						ctx.BindReg(d276.Reg, &d276)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d276)
					ctx.FreeDesc(&d273)
					ctx.EnsureDesc(&d272)
					ctx.EnsureDesc(&d272)
					ctx.EnsureDescsTogether(&d272, &d272)
					var d277 JITValueDesc
					if d272.Loc == LocImm {
						d277 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d272.Imm.Float() * d272.Imm.Float())}
					} else if d272.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d272.Reg)
						_, xBits := d272.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d272.Reg)
						d277 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d277)
					} else if d272.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d272.Reg)
						ctx.EmitMovRegReg(scratch, d272.Reg)
						_, yBits := d272.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d277 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d277)
					} else {
						r12 := ctx.AllocRegExcept(d272.Reg, d272.Reg)
						ctx.EmitMovRegReg(r12, d272.Reg)
						ctx.EmitMulFloat64(r12, d272.Reg)
						d277 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
						ctx.BindReg(r12, &d277)
					}
					if d277.Loc == LocReg && d272.Loc == LocReg && d277.Reg == d272.Reg {
						ctx.TransferReg(d272.Reg)
						d272.Loc = LocNone
					}
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d277)
					ctx.EnsureDescsTogether(&d11, &d277)
					var d278 JITValueDesc
					if d11.Loc == LocImm && d277.Loc == LocImm {
						d278 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d11.Imm.Float() + d277.Imm.Float())}
					} else if d11.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 && r1 != d277.Reg {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d277.Reg)
						}
						_, xBits := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d277.Reg)
						d278 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d278)
					} else if d277.Loc == LocImm {
						var scratch Reg
						if phiHomeOK3 {
							scratch = r1
						} else {
							scratch = ctx.AllocRegExcept(d11.Reg)
						}
						ctx.EmitMovRegReg(scratch, d11.Reg)
						_, yBits := d277.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d278 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d278)
					} else {
						var r13 Reg
						if phiHomeOK3 && r1 != d277.Reg {
							r13 = r1
						} else {
							r13 = ctx.AllocRegExcept(d11.Reg, d277.Reg)
						}
						ctx.EmitMovRegReg(r13, d11.Reg)
						ctx.EmitAddFloat64(r13, d277.Reg)
						d278 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r13}
						ctx.BindReg(r13, &d278)
					}
					if d278.Loc == LocReg && d11.Loc == LocReg && d278.Reg == d11.Reg {
						ctx.TransferReg(d11.Reg)
						d11.Loc = LocNone
					}
					ctx.FreeDesc(&d277)
					ctx.EnsureDesc(&d276)
					ctx.EnsureDesc(&d276)
					ctx.EnsureDescsTogether(&d276, &d276)
					var d279 JITValueDesc
					if d276.Loc == LocImm {
						d279 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d276.Imm.Float() * d276.Imm.Float())}
					} else if d276.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d276.Reg)
						_, xBits := d276.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d276.Reg)
						d279 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d279)
					} else if d276.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d276.Reg)
						ctx.EmitMovRegReg(scratch, d276.Reg)
						_, yBits := d276.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d279 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d279)
					} else {
						r14 := ctx.AllocRegExcept(d276.Reg, d276.Reg)
						ctx.EmitMovRegReg(r14, d276.Reg)
						ctx.EmitMulFloat64(r14, d276.Reg)
						d279 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r14}
						ctx.BindReg(r14, &d279)
					}
					if d279.Loc == LocReg && d276.Loc == LocReg && d279.Reg == d276.Reg {
						ctx.TransferReg(d276.Reg)
						d276.Loc = LocNone
					}
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d279)
					ctx.EnsureDescsTogether(&d12, &d279)
					var d280 JITValueDesc
					if d12.Loc == LocImm && d279.Loc == LocImm {
						d280 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d12.Imm.Float() + d279.Imm.Float())}
					} else if d12.Loc == LocImm {
						var scratch Reg
						if phiHomeOK4 && r2 != d279.Reg {
							scratch = r2
						} else {
							scratch = ctx.AllocRegExcept(d279.Reg)
						}
						_, xBits := d12.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d279.Reg)
						d280 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d280)
					} else if d279.Loc == LocImm {
						var scratch Reg
						if phiHomeOK4 {
							scratch = r2
						} else {
							scratch = ctx.AllocRegExcept(d12.Reg)
						}
						ctx.EmitMovRegReg(scratch, d12.Reg)
						_, yBits := d279.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d280 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d280)
					} else {
						var r15 Reg
						if phiHomeOK4 && r2 != d279.Reg {
							r15 = r2
						} else {
							r15 = ctx.AllocRegExcept(d12.Reg, d279.Reg)
						}
						ctx.EmitMovRegReg(r15, d12.Reg)
						ctx.EmitAddFloat64(r15, d279.Reg)
						d280 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r15}
						ctx.BindReg(r15, &d280)
					}
					if d280.Loc == LocReg && d12.Loc == LocReg && d280.Reg == d12.Reg {
						ctx.TransferReg(d12.Reg)
						d12.Loc = LocNone
					}
					ctx.FreeDesc(&d279)
					ctx.EnsureDesc(&d272)
					ctx.EnsureDesc(&d276)
					ctx.EnsureDescsTogether(&d272, &d276)
					var d281 JITValueDesc
					if d272.Loc == LocImm && d276.Loc == LocImm {
						d281 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d272.Imm.Float() * d276.Imm.Float())}
					} else if d272.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d276.Reg)
						_, xBits := d272.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d276.Reg)
						d281 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d281)
					} else if d276.Loc == LocImm {
						_, yBits := d276.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d272.Reg, RegR11)
						d281 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d272.Reg}
						ctx.BindReg(d272.Reg, &d281)
					} else {
						ctx.EmitMulFloat64(d272.Reg, d276.Reg)
						d281 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d272.Reg}
						ctx.BindReg(d272.Reg, &d281)
					}
					if d281.Loc == LocReg && d272.Loc == LocReg && d281.Reg == d272.Reg {
						ctx.TransferReg(d272.Reg)
						d272.Loc = LocNone
					}
					ctx.FreeDesc(&d272)
					ctx.FreeDesc(&d276)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d281)
					ctx.EnsureDescsTogether(&d10, &d281)
					var d282 JITValueDesc
					if d10.Loc == LocImm && d281.Loc == LocImm {
						d282 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float() + d281.Imm.Float())}
					} else if d10.Loc == LocImm {
						var scratch Reg
						if phiHomeOK2 && r0 != d281.Reg {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d281.Reg)
						}
						_, xBits := d10.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d281.Reg)
						d282 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d282)
					} else if d281.Loc == LocImm {
						var scratch Reg
						if phiHomeOK2 {
							scratch = r0
						} else {
							scratch = ctx.AllocRegExcept(d10.Reg)
						}
						ctx.EmitMovRegReg(scratch, d10.Reg)
						_, yBits := d281.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d282 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d282)
					} else {
						var r16 Reg
						if phiHomeOK2 && r0 != d281.Reg {
							r16 = r0
						} else {
							r16 = ctx.AllocRegExcept(d10.Reg, d281.Reg)
						}
						ctx.EmitMovRegReg(r16, d10.Reg)
						ctx.EmitAddFloat64(r16, d281.Reg)
						d282 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
						ctx.BindReg(r16, &d282)
					}
					if d282.Loc == LocReg && d10.Loc == LocReg && d282.Reg == d10.Reg {
						ctx.TransferReg(d10.Reg)
						d10.Loc = LocNone
					}
					ctx.FreeDesc(&d281)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					var d283 JITValueDesc
					if d13.Loc == LocImm {
						d283 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK5 {
							scratch = r3
						} else {
							scratch = ctx.AllocRegExcept(d13.Reg)
						}
						ctx.EmitMovRegReg(scratch, d13.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d283 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d283)
					}
					if d283.Loc == LocReg && d13.Loc == LocReg && d283.Reg == d13.Reg {
						ctx.TransferReg(d13.Reg)
						d13.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d278)
						if d278.Loc == LocReg {
							ctx.ProtectReg(d278.Reg)
						} else if d278.Loc == LocRegPair {
							ctx.ProtectReg(d278.Reg)
							ctx.ProtectReg(d278.Reg2)
						}
						ctx.SyncDesc(&d280)
						if d280.Loc == LocReg {
							ctx.ProtectReg(d280.Reg)
						} else if d280.Loc == LocRegPair {
							ctx.ProtectReg(d280.Reg)
							ctx.ProtectReg(d280.Reg2)
						}
						ctx.SyncDesc(&d282)
						if d282.Loc == LocReg {
							ctx.ProtectReg(d282.Reg)
						} else if d282.Loc == LocRegPair {
							ctx.ProtectReg(d282.Reg)
							ctx.ProtectReg(d282.Reg2)
						}
						d284 = d282
						if d284.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d284)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d284)
						} else {
							ctx.EmitStoreToStack(d284, int32(bbs[6].PhiBase)+int32(0))
						}
						d285 = d278
						if d285.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d285)
						if phiHomeOK3 {
							ctx.EmitMovToReg(r1, d285)
						} else {
							ctx.EmitStoreToStack(d285, int32(bbs[6].PhiBase)+int32(16))
						}
						d286 = d280
						if d286.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d286)
						if phiHomeOK4 {
							ctx.EmitMovToReg(r2, d286)
						} else {
							ctx.EmitStoreToStack(d286, int32(bbs[6].PhiBase)+int32(32))
						}
						if d278.Loc == LocReg {
							ctx.UnprotectReg(d278.Reg)
						} else if d278.Loc == LocRegPair {
							ctx.UnprotectReg(d278.Reg)
							ctx.UnprotectReg(d278.Reg2)
						}
						if d280.Loc == LocReg {
							ctx.UnprotectReg(d280.Reg)
						} else if d280.Loc == LocRegPair {
							ctx.UnprotectReg(d280.Reg)
							ctx.UnprotectReg(d280.Reg2)
						}
						if d282.Loc == LocReg {
							ctx.UnprotectReg(d282.Reg)
						} else if d282.Loc == LocRegPair {
							ctx.UnprotectReg(d282.Reg)
							ctx.UnprotectReg(d282.Reg2)
						}
						ctx.SyncDesc(&d283)
						if d283.Loc == LocReg {
							ctx.ProtectReg(d283.Reg)
						} else if d283.Loc == LocRegPair {
							ctx.ProtectReg(d283.Reg)
							ctx.ProtectReg(d283.Reg2)
						}
						d287 = d283
						if d287.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d287)
						if phiHomeOK5 {
							ctx.EmitMovToReg(r3, d287)
						} else {
							ctx.EmitStoreToStack(d287, int32(bbs[6].PhiBase)+int32(48))
						}
						if d283.Loc == LocReg {
							ctx.UnprotectReg(d283.Reg)
						} else if d283.Loc == LocRegPair {
							ctx.UnprotectReg(d283.Reg)
							ctx.UnprotectReg(d283.Reg2)
						}
					}
					ps288 := PhiState{General: ps.General}
					ps288.OverlayValues = make([]JITValueDesc, 288)
					ps288.OverlayValues[8] = d8
					ps288.OverlayValues[9] = d9
					ps288.OverlayValues[10] = d10
					ps288.OverlayValues[11] = d11
					ps288.OverlayValues[12] = d12
					ps288.OverlayValues[13] = d13
					ps288.OverlayValues[14] = d14
					ps288.OverlayValues[15] = d15
					ps288.OverlayValues[16] = d16
					ps288.OverlayValues[17] = d17
					ps288.OverlayValues[18] = d18
					ps288.OverlayValues[19] = d19
					ps288.OverlayValues[20] = d20
					ps288.OverlayValues[21] = d21
					ps288.OverlayValues[22] = d22
					ps288.OverlayValues[25] = d25
					ps288.OverlayValues[45] = d45
					ps288.OverlayValues[64] = d64
					ps288.OverlayValues[65] = d65
					ps288.OverlayValues[66] = d66
					ps288.OverlayValues[67] = d67
					ps288.OverlayValues[68] = d68
					ps288.OverlayValues[70] = d70
					ps288.OverlayValues[71] = d71
					ps288.OverlayValues[72] = d72
					ps288.OverlayValues[73] = d73
					ps288.OverlayValues[74] = d74
					ps288.OverlayValues[75] = d75
					ps288.OverlayValues[76] = d76
					ps288.OverlayValues[79] = d79
					ps288.OverlayValues[145] = d145
					ps288.OverlayValues[146] = d146
					ps288.OverlayValues[147] = d147
					ps288.OverlayValues[148] = d148
					ps288.OverlayValues[149] = d149
					ps288.OverlayValues[150] = d150
					ps288.OverlayValues[152] = d152
					ps288.OverlayValues[153] = d153
					ps288.OverlayValues[154] = d154
					ps288.OverlayValues[155] = d155
					ps288.OverlayValues[156] = d156
					ps288.OverlayValues[157] = d157
					ps288.OverlayValues[158] = d158
					ps288.OverlayValues[159] = d159
					ps288.OverlayValues[160] = d160
					ps288.OverlayValues[163] = d163
					ps288.OverlayValues[164] = d164
					ps288.OverlayValues[165] = d165
					ps288.OverlayValues[166] = d166
					ps288.OverlayValues[269] = d269
					ps288.OverlayValues[270] = d270
					ps288.OverlayValues[271] = d271
					ps288.OverlayValues[272] = d272
					ps288.OverlayValues[273] = d273
					ps288.OverlayValues[274] = d274
					ps288.OverlayValues[275] = d275
					ps288.OverlayValues[276] = d276
					ps288.OverlayValues[277] = d277
					ps288.OverlayValues[278] = d278
					ps288.OverlayValues[279] = d279
					ps288.OverlayValues[280] = d280
					ps288.OverlayValues[281] = d281
					ps288.OverlayValues[282] = d282
					ps288.OverlayValues[283] = d283
					ps288.OverlayValues[284] = d284
					ps288.OverlayValues[285] = d285
					ps288.OverlayValues[286] = d286
					ps288.OverlayValues[287] = d287
					ps288.PhiValues = make([]JITValueDesc, 4)
					d289 = d282
					ps288.PhiValues[0] = d289
					d290 = d278
					ps288.PhiValues[1] = d290
					d291 = d280
					ps288.PhiValues[2] = d291
					d292 = d283
					ps288.PhiValues[3] = d292
					if ps288.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps288)
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != LocNone {
						d269 = ps.OverlayValues[269]
					}
					if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
						d270 = ps.OverlayValues[270]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != LocNone {
						d279 = ps.OverlayValues[279]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != LocNone {
						d281 = ps.OverlayValues[281]
					}
					if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != LocNone {
						d282 = ps.OverlayValues[282]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
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
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDescsTogether(&d11, &d12)
					var d293 JITValueDesc
					if d11.Loc == LocImm && d12.Loc == LocImm {
						d293 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d11.Imm.Float() * d12.Imm.Float())}
					} else if d11.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d12.Reg)
						_, xBits := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d12.Reg)
						d293 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d293)
					} else if d12.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitMovRegReg(scratch, d11.Reg)
						_, yBits := d12.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d293 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d293)
					} else {
						r17 := ctx.AllocRegExcept(d11.Reg, d12.Reg)
						ctx.EmitMovRegReg(r17, d11.Reg)
						ctx.EmitMulFloat64(r17, d12.Reg)
						d293 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r17}
						ctx.BindReg(r17, &d293)
					}
					if d293.Loc == LocReg && d11.Loc == LocReg && d293.Reg == d11.Reg {
						ctx.TransferReg(d11.Reg)
						d11.Loc = LocNone
					}
					ctx.EnsureDesc(&d293)
					var d294 JITValueDesc
					if d293.Loc == LocImm {
						d294 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d293.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d293)
						var d295 JITValueDesc
						if d293.Loc == LocRegPair {
							ctx.FreeReg(d293.Reg)
							d295 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d293.Reg2}
							ctx.BindReg(d293.Reg2, &d295)
							ctx.BindReg(d293.Reg2, &d295)
						} else {
							d295 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d293.Reg}
							ctx.BindReg(d293.Reg, &d295)
							ctx.BindReg(d293.Reg, &d295)
						}
						d294 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d295}, 1)
						d294.Type = tagFloat
						ctx.BindReg(d294.Reg, &d294)
					}
					ctx.FreeDesc(&d293)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d294)
					ctx.EnsureDescsTogether(&d10, &d294)
					var d296 JITValueDesc
					if d10.Loc == LocImm && d294.Loc == LocImm {
						d296 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float() / d294.Imm.Float())}
					} else if d10.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d294.Reg)
						_, xBits := d10.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d294.Reg)
						d296 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d296)
					} else if d294.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d10.Reg)
						ctx.EmitMovRegReg(scratch, d10.Reg)
						_, yBits := d294.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitDivFloat64(scratch, RegR11)
						d296 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d296)
					} else {
						r18 := ctx.AllocRegExcept(d10.Reg, d294.Reg)
						ctx.EmitMovRegReg(r18, d10.Reg)
						ctx.EmitDivFloat64(r18, d294.Reg)
						d296 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r18}
						ctx.BindReg(r18, &d296)
					}
					if d296.Loc == LocReg && d10.Loc == LocReg && d296.Reg == d10.Reg {
						ctx.TransferReg(d10.Reg)
						d10.Loc = LocNone
					}
					ctx.EnsureDesc(&d296)
					ctx.EmitStoreToStack(d296, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d296)
					ctx.FreeDesc(&d294)
					if ps.General {
					}
					ps297 := PhiState{General: ps.General}
					ps297.OverlayValues = make([]JITValueDesc, 297)
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
					ps297.OverlayValues[45] = d45
					ps297.OverlayValues[64] = d64
					ps297.OverlayValues[65] = d65
					ps297.OverlayValues[66] = d66
					ps297.OverlayValues[67] = d67
					ps297.OverlayValues[68] = d68
					ps297.OverlayValues[70] = d70
					ps297.OverlayValues[71] = d71
					ps297.OverlayValues[72] = d72
					ps297.OverlayValues[73] = d73
					ps297.OverlayValues[74] = d74
					ps297.OverlayValues[75] = d75
					ps297.OverlayValues[76] = d76
					ps297.OverlayValues[79] = d79
					ps297.OverlayValues[145] = d145
					ps297.OverlayValues[146] = d146
					ps297.OverlayValues[147] = d147
					ps297.OverlayValues[148] = d148
					ps297.OverlayValues[149] = d149
					ps297.OverlayValues[150] = d150
					ps297.OverlayValues[152] = d152
					ps297.OverlayValues[153] = d153
					ps297.OverlayValues[154] = d154
					ps297.OverlayValues[155] = d155
					ps297.OverlayValues[156] = d156
					ps297.OverlayValues[157] = d157
					ps297.OverlayValues[158] = d158
					ps297.OverlayValues[159] = d159
					ps297.OverlayValues[160] = d160
					ps297.OverlayValues[163] = d163
					ps297.OverlayValues[164] = d164
					ps297.OverlayValues[165] = d165
					ps297.OverlayValues[166] = d166
					ps297.OverlayValues[269] = d269
					ps297.OverlayValues[270] = d270
					ps297.OverlayValues[271] = d271
					ps297.OverlayValues[272] = d272
					ps297.OverlayValues[273] = d273
					ps297.OverlayValues[274] = d274
					ps297.OverlayValues[275] = d275
					ps297.OverlayValues[276] = d276
					ps297.OverlayValues[277] = d277
					ps297.OverlayValues[278] = d278
					ps297.OverlayValues[279] = d279
					ps297.OverlayValues[280] = d280
					ps297.OverlayValues[281] = d281
					ps297.OverlayValues[282] = d282
					ps297.OverlayValues[283] = d283
					ps297.OverlayValues[284] = d284
					ps297.OverlayValues[285] = d285
					ps297.OverlayValues[286] = d286
					ps297.OverlayValues[287] = d287
					ps297.OverlayValues[289] = d289
					ps297.OverlayValues[290] = d290
					ps297.OverlayValues[291] = d291
					ps297.OverlayValues[292] = d292
					ps297.OverlayValues[293] = d293
					ps297.OverlayValues[294] = d294
					ps297.OverlayValues[295] = d295
					ps297.OverlayValues[296] = d296
					ps297.PhiValues = make([]JITValueDesc, 1)
					if ps297.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps297)
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != LocNone {
						d269 = ps.OverlayValues[269]
					}
					if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
						d270 = ps.OverlayValues[270]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != LocNone {
						d279 = ps.OverlayValues[279]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != LocNone {
						d281 = ps.OverlayValues[281]
					}
					if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != LocNone {
						d282 = ps.OverlayValues[282]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
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
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
					}
					ctx.ReclaimUntrackedRegs()
					var d298 JITValueDesc
					if d19.SliceSizeKnown {
						d298 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d298 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d298 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d298 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d298 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d298)
					ctx.EnsureDescsTogether(&d13, &d298)
					var d299 JITValueDesc
					if d13.Loc == LocImm && d298.Loc == LocImm {
						d299 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d298.Imm.Int())}
					} else if d298.Loc == LocImm {
						r19 := ctx.AllocRegExcept(d13.Reg)
						if d298.Imm.Int() >= -2147483648 && d298.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d298.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d298.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, RegR11)
						}
						d299 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondSignedLess}
						ctx.BindReg(r19, &d299)
					} else if d13.Loc == LocImm {
						r20 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d298.Reg)
						d299 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
						ctx.BindReg(r20, &d299)
					} else {
						r21 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpInt64(d13.Reg, d298.Reg)
						d299 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedLess}
						ctx.BindReg(r21, &d299)
					}
					ctx.FreeDesc(&d298)
					d300 = d299
					ctx.EnsureDesc(&d300)
					if d300.Loc != LocImm && d300.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d300.Loc == LocImm {
						if d300.Imm.Bool() {
							if ps.General {
							}
							ps301 := PhiState{General: ps.General}
							ps301.OverlayValues = make([]JITValueDesc, 301)
							ps301.OverlayValues[8] = d8
							ps301.OverlayValues[9] = d9
							ps301.OverlayValues[10] = d10
							ps301.OverlayValues[11] = d11
							ps301.OverlayValues[12] = d12
							ps301.OverlayValues[13] = d13
							ps301.OverlayValues[14] = d14
							ps301.OverlayValues[15] = d15
							ps301.OverlayValues[16] = d16
							ps301.OverlayValues[17] = d17
							ps301.OverlayValues[18] = d18
							ps301.OverlayValues[19] = d19
							ps301.OverlayValues[20] = d20
							ps301.OverlayValues[21] = d21
							ps301.OverlayValues[22] = d22
							ps301.OverlayValues[25] = d25
							ps301.OverlayValues[45] = d45
							ps301.OverlayValues[64] = d64
							ps301.OverlayValues[65] = d65
							ps301.OverlayValues[66] = d66
							ps301.OverlayValues[67] = d67
							ps301.OverlayValues[68] = d68
							ps301.OverlayValues[70] = d70
							ps301.OverlayValues[71] = d71
							ps301.OverlayValues[72] = d72
							ps301.OverlayValues[73] = d73
							ps301.OverlayValues[74] = d74
							ps301.OverlayValues[75] = d75
							ps301.OverlayValues[76] = d76
							ps301.OverlayValues[79] = d79
							ps301.OverlayValues[145] = d145
							ps301.OverlayValues[146] = d146
							ps301.OverlayValues[147] = d147
							ps301.OverlayValues[148] = d148
							ps301.OverlayValues[149] = d149
							ps301.OverlayValues[150] = d150
							ps301.OverlayValues[152] = d152
							ps301.OverlayValues[153] = d153
							ps301.OverlayValues[154] = d154
							ps301.OverlayValues[155] = d155
							ps301.OverlayValues[156] = d156
							ps301.OverlayValues[157] = d157
							ps301.OverlayValues[158] = d158
							ps301.OverlayValues[159] = d159
							ps301.OverlayValues[160] = d160
							ps301.OverlayValues[163] = d163
							ps301.OverlayValues[164] = d164
							ps301.OverlayValues[165] = d165
							ps301.OverlayValues[166] = d166
							ps301.OverlayValues[269] = d269
							ps301.OverlayValues[270] = d270
							ps301.OverlayValues[271] = d271
							ps301.OverlayValues[272] = d272
							ps301.OverlayValues[273] = d273
							ps301.OverlayValues[274] = d274
							ps301.OverlayValues[275] = d275
							ps301.OverlayValues[276] = d276
							ps301.OverlayValues[277] = d277
							ps301.OverlayValues[278] = d278
							ps301.OverlayValues[279] = d279
							ps301.OverlayValues[280] = d280
							ps301.OverlayValues[281] = d281
							ps301.OverlayValues[282] = d282
							ps301.OverlayValues[283] = d283
							ps301.OverlayValues[284] = d284
							ps301.OverlayValues[285] = d285
							ps301.OverlayValues[286] = d286
							ps301.OverlayValues[287] = d287
							ps301.OverlayValues[289] = d289
							ps301.OverlayValues[290] = d290
							ps301.OverlayValues[291] = d291
							ps301.OverlayValues[292] = d292
							ps301.OverlayValues[293] = d293
							ps301.OverlayValues[294] = d294
							ps301.OverlayValues[295] = d295
							ps301.OverlayValues[296] = d296
							ps301.OverlayValues[298] = d298
							ps301.OverlayValues[299] = d299
							ps301.OverlayValues[300] = d300
							return bbs[7].RenderPS(ps301)
						}
						if ps.General {
						}
						ps302 := PhiState{General: ps.General}
						ps302.OverlayValues = make([]JITValueDesc, 301)
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
						ps302.OverlayValues[25] = d25
						ps302.OverlayValues[45] = d45
						ps302.OverlayValues[64] = d64
						ps302.OverlayValues[65] = d65
						ps302.OverlayValues[66] = d66
						ps302.OverlayValues[67] = d67
						ps302.OverlayValues[68] = d68
						ps302.OverlayValues[70] = d70
						ps302.OverlayValues[71] = d71
						ps302.OverlayValues[72] = d72
						ps302.OverlayValues[73] = d73
						ps302.OverlayValues[74] = d74
						ps302.OverlayValues[75] = d75
						ps302.OverlayValues[76] = d76
						ps302.OverlayValues[79] = d79
						ps302.OverlayValues[145] = d145
						ps302.OverlayValues[146] = d146
						ps302.OverlayValues[147] = d147
						ps302.OverlayValues[148] = d148
						ps302.OverlayValues[149] = d149
						ps302.OverlayValues[150] = d150
						ps302.OverlayValues[152] = d152
						ps302.OverlayValues[153] = d153
						ps302.OverlayValues[154] = d154
						ps302.OverlayValues[155] = d155
						ps302.OverlayValues[156] = d156
						ps302.OverlayValues[157] = d157
						ps302.OverlayValues[158] = d158
						ps302.OverlayValues[159] = d159
						ps302.OverlayValues[160] = d160
						ps302.OverlayValues[163] = d163
						ps302.OverlayValues[164] = d164
						ps302.OverlayValues[165] = d165
						ps302.OverlayValues[166] = d166
						ps302.OverlayValues[269] = d269
						ps302.OverlayValues[270] = d270
						ps302.OverlayValues[271] = d271
						ps302.OverlayValues[272] = d272
						ps302.OverlayValues[273] = d273
						ps302.OverlayValues[274] = d274
						ps302.OverlayValues[275] = d275
						ps302.OverlayValues[276] = d276
						ps302.OverlayValues[277] = d277
						ps302.OverlayValues[278] = d278
						ps302.OverlayValues[279] = d279
						ps302.OverlayValues[280] = d280
						ps302.OverlayValues[281] = d281
						ps302.OverlayValues[282] = d282
						ps302.OverlayValues[283] = d283
						ps302.OverlayValues[284] = d284
						ps302.OverlayValues[285] = d285
						ps302.OverlayValues[286] = d286
						ps302.OverlayValues[287] = d287
						ps302.OverlayValues[289] = d289
						ps302.OverlayValues[290] = d290
						ps302.OverlayValues[291] = d291
						ps302.OverlayValues[292] = d292
						ps302.OverlayValues[293] = d293
						ps302.OverlayValues[294] = d294
						ps302.OverlayValues[295] = d295
						ps302.OverlayValues[296] = d296
						ps302.OverlayValues[298] = d298
						ps302.OverlayValues[299] = d299
						ps302.OverlayValues[300] = d300
						return bbs[8].RenderPS(ps302)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					ctx.EmitJump(d300.Condition, lbl8)
					snap303 := d8
					snap304 := d9
					snap305 := d10
					snap306 := d11
					snap307 := d12
					snap308 := d13
					snap309 := d14
					snap310 := d15
					snap311 := d16
					snap312 := d17
					snap313 := d18
					snap314 := d19
					snap315 := d20
					snap316 := d21
					snap317 := d22
					snap318 := d25
					snap319 := d45
					snap320 := d64
					snap321 := d65
					snap322 := d66
					snap323 := d67
					snap324 := d68
					snap325 := d70
					snap326 := d71
					snap327 := d72
					snap328 := d73
					snap329 := d74
					snap330 := d75
					snap331 := d76
					snap332 := d79
					snap333 := d145
					snap334 := d146
					snap335 := d147
					snap336 := d148
					snap337 := d149
					snap338 := d150
					snap339 := d152
					snap340 := d153
					snap341 := d154
					snap342 := d155
					snap343 := d156
					snap344 := d157
					snap345 := d158
					snap346 := d159
					snap347 := d160
					snap348 := d163
					snap349 := d164
					snap350 := d165
					snap351 := d166
					snap352 := d269
					snap353 := d270
					snap354 := d271
					snap355 := d272
					snap356 := d273
					snap357 := d274
					snap358 := d275
					snap359 := d276
					snap360 := d277
					snap361 := d278
					snap362 := d279
					snap363 := d280
					snap364 := d281
					snap365 := d282
					snap366 := d283
					snap367 := d284
					snap368 := d285
					snap369 := d286
					snap370 := d287
					snap371 := d289
					snap372 := d290
					snap373 := d291
					snap374 := d292
					snap375 := d293
					snap376 := d294
					snap377 := d295
					snap378 := d296
					snap379 := d298
					snap380 := d299
					snap381 := d300
					alloc382 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc382)
					d8 = snap303
					d9 = snap304
					d10 = snap305
					d11 = snap306
					d12 = snap307
					d13 = snap308
					d14 = snap309
					d15 = snap310
					d16 = snap311
					d17 = snap312
					d18 = snap313
					d19 = snap314
					d20 = snap315
					d21 = snap316
					d22 = snap317
					d25 = snap318
					d45 = snap319
					d64 = snap320
					d65 = snap321
					d66 = snap322
					d67 = snap323
					d68 = snap324
					d70 = snap325
					d71 = snap326
					d72 = snap327
					d73 = snap328
					d74 = snap329
					d75 = snap330
					d76 = snap331
					d79 = snap332
					d145 = snap333
					d146 = snap334
					d147 = snap335
					d148 = snap336
					d149 = snap337
					d150 = snap338
					d152 = snap339
					d153 = snap340
					d154 = snap341
					d155 = snap342
					d156 = snap343
					d157 = snap344
					d158 = snap345
					d159 = snap346
					d160 = snap347
					d163 = snap348
					d164 = snap349
					d165 = snap350
					d166 = snap351
					d269 = snap352
					d270 = snap353
					d271 = snap354
					d272 = snap355
					d273 = snap356
					d274 = snap357
					d275 = snap358
					d276 = snap359
					d277 = snap360
					d278 = snap361
					d279 = snap362
					d280 = snap363
					d281 = snap364
					d282 = snap365
					d283 = snap366
					d284 = snap367
					d285 = snap368
					d286 = snap369
					d287 = snap370
					d289 = snap371
					d290 = snap372
					d291 = snap373
					d292 = snap374
					d293 = snap375
					d294 = snap376
					d295 = snap377
					d296 = snap378
					d298 = snap379
					d299 = snap380
					d300 = snap381
					ctx.RestoreAllocState(alloc382)
					d8 = snap303
					d9 = snap304
					d10 = snap305
					d11 = snap306
					d12 = snap307
					d13 = snap308
					d14 = snap309
					d15 = snap310
					d16 = snap311
					d17 = snap312
					d18 = snap313
					d19 = snap314
					d20 = snap315
					d21 = snap316
					d22 = snap317
					d25 = snap318
					d45 = snap319
					d64 = snap320
					d65 = snap321
					d66 = snap322
					d67 = snap323
					d68 = snap324
					d70 = snap325
					d71 = snap326
					d72 = snap327
					d73 = snap328
					d74 = snap329
					d75 = snap330
					d76 = snap331
					d79 = snap332
					d145 = snap333
					d146 = snap334
					d147 = snap335
					d148 = snap336
					d149 = snap337
					d150 = snap338
					d152 = snap339
					d153 = snap340
					d154 = snap341
					d155 = snap342
					d156 = snap343
					d157 = snap344
					d158 = snap345
					d159 = snap346
					d160 = snap347
					d163 = snap348
					d164 = snap349
					d165 = snap350
					d166 = snap351
					d269 = snap352
					d270 = snap353
					d271 = snap354
					d272 = snap355
					d273 = snap356
					d274 = snap357
					d275 = snap358
					d276 = snap359
					d277 = snap360
					d278 = snap361
					d279 = snap362
					d280 = snap363
					d281 = snap364
					d282 = snap365
					d283 = snap366
					d284 = snap367
					d285 = snap368
					d286 = snap369
					d287 = snap370
					d289 = snap371
					d290 = snap372
					d291 = snap373
					d292 = snap374
					d293 = snap375
					d294 = snap376
					d295 = snap377
					d296 = snap378
					d298 = snap379
					d299 = snap380
					d300 = snap381
					ps383 := PhiState{General: true}
					ps383.OverlayValues = make([]JITValueDesc, 301)
					ps383.OverlayValues[8] = d8
					ps383.OverlayValues[9] = d9
					ps383.OverlayValues[10] = d10
					ps383.OverlayValues[11] = d11
					ps383.OverlayValues[12] = d12
					ps383.OverlayValues[13] = d13
					ps383.OverlayValues[14] = d14
					ps383.OverlayValues[15] = d15
					ps383.OverlayValues[16] = d16
					ps383.OverlayValues[17] = d17
					ps383.OverlayValues[18] = d18
					ps383.OverlayValues[19] = d19
					ps383.OverlayValues[20] = d20
					ps383.OverlayValues[21] = d21
					ps383.OverlayValues[22] = d22
					ps383.OverlayValues[25] = d25
					ps383.OverlayValues[45] = d45
					ps383.OverlayValues[64] = d64
					ps383.OverlayValues[65] = d65
					ps383.OverlayValues[66] = d66
					ps383.OverlayValues[67] = d67
					ps383.OverlayValues[68] = d68
					ps383.OverlayValues[70] = d70
					ps383.OverlayValues[71] = d71
					ps383.OverlayValues[72] = d72
					ps383.OverlayValues[73] = d73
					ps383.OverlayValues[74] = d74
					ps383.OverlayValues[75] = d75
					ps383.OverlayValues[76] = d76
					ps383.OverlayValues[79] = d79
					ps383.OverlayValues[145] = d145
					ps383.OverlayValues[146] = d146
					ps383.OverlayValues[147] = d147
					ps383.OverlayValues[148] = d148
					ps383.OverlayValues[149] = d149
					ps383.OverlayValues[150] = d150
					ps383.OverlayValues[152] = d152
					ps383.OverlayValues[153] = d153
					ps383.OverlayValues[154] = d154
					ps383.OverlayValues[155] = d155
					ps383.OverlayValues[156] = d156
					ps383.OverlayValues[157] = d157
					ps383.OverlayValues[158] = d158
					ps383.OverlayValues[159] = d159
					ps383.OverlayValues[160] = d160
					ps383.OverlayValues[163] = d163
					ps383.OverlayValues[164] = d164
					ps383.OverlayValues[165] = d165
					ps383.OverlayValues[166] = d166
					ps383.OverlayValues[269] = d269
					ps383.OverlayValues[270] = d270
					ps383.OverlayValues[271] = d271
					ps383.OverlayValues[272] = d272
					ps383.OverlayValues[273] = d273
					ps383.OverlayValues[274] = d274
					ps383.OverlayValues[275] = d275
					ps383.OverlayValues[276] = d276
					ps383.OverlayValues[277] = d277
					ps383.OverlayValues[278] = d278
					ps383.OverlayValues[279] = d279
					ps383.OverlayValues[280] = d280
					ps383.OverlayValues[281] = d281
					ps383.OverlayValues[282] = d282
					ps383.OverlayValues[283] = d283
					ps383.OverlayValues[284] = d284
					ps383.OverlayValues[285] = d285
					ps383.OverlayValues[286] = d286
					ps383.OverlayValues[287] = d287
					ps383.OverlayValues[289] = d289
					ps383.OverlayValues[290] = d290
					ps383.OverlayValues[291] = d291
					ps383.OverlayValues[292] = d292
					ps383.OverlayValues[293] = d293
					ps383.OverlayValues[294] = d294
					ps383.OverlayValues[295] = d295
					ps383.OverlayValues[296] = d296
					ps383.OverlayValues[298] = d298
					ps383.OverlayValues[299] = d299
					ps383.OverlayValues[300] = d300
					ps384 := PhiState{General: true}
					ps384.OverlayValues = make([]JITValueDesc, 301)
					ps384.OverlayValues[8] = d8
					ps384.OverlayValues[9] = d9
					ps384.OverlayValues[10] = d10
					ps384.OverlayValues[11] = d11
					ps384.OverlayValues[12] = d12
					ps384.OverlayValues[13] = d13
					ps384.OverlayValues[14] = d14
					ps384.OverlayValues[15] = d15
					ps384.OverlayValues[16] = d16
					ps384.OverlayValues[17] = d17
					ps384.OverlayValues[18] = d18
					ps384.OverlayValues[19] = d19
					ps384.OverlayValues[20] = d20
					ps384.OverlayValues[21] = d21
					ps384.OverlayValues[22] = d22
					ps384.OverlayValues[25] = d25
					ps384.OverlayValues[45] = d45
					ps384.OverlayValues[64] = d64
					ps384.OverlayValues[65] = d65
					ps384.OverlayValues[66] = d66
					ps384.OverlayValues[67] = d67
					ps384.OverlayValues[68] = d68
					ps384.OverlayValues[70] = d70
					ps384.OverlayValues[71] = d71
					ps384.OverlayValues[72] = d72
					ps384.OverlayValues[73] = d73
					ps384.OverlayValues[74] = d74
					ps384.OverlayValues[75] = d75
					ps384.OverlayValues[76] = d76
					ps384.OverlayValues[79] = d79
					ps384.OverlayValues[145] = d145
					ps384.OverlayValues[146] = d146
					ps384.OverlayValues[147] = d147
					ps384.OverlayValues[148] = d148
					ps384.OverlayValues[149] = d149
					ps384.OverlayValues[150] = d150
					ps384.OverlayValues[152] = d152
					ps384.OverlayValues[153] = d153
					ps384.OverlayValues[154] = d154
					ps384.OverlayValues[155] = d155
					ps384.OverlayValues[156] = d156
					ps384.OverlayValues[157] = d157
					ps384.OverlayValues[158] = d158
					ps384.OverlayValues[159] = d159
					ps384.OverlayValues[160] = d160
					ps384.OverlayValues[163] = d163
					ps384.OverlayValues[164] = d164
					ps384.OverlayValues[165] = d165
					ps384.OverlayValues[166] = d166
					ps384.OverlayValues[269] = d269
					ps384.OverlayValues[270] = d270
					ps384.OverlayValues[271] = d271
					ps384.OverlayValues[272] = d272
					ps384.OverlayValues[273] = d273
					ps384.OverlayValues[274] = d274
					ps384.OverlayValues[275] = d275
					ps384.OverlayValues[276] = d276
					ps384.OverlayValues[277] = d277
					ps384.OverlayValues[278] = d278
					ps384.OverlayValues[279] = d279
					ps384.OverlayValues[280] = d280
					ps384.OverlayValues[281] = d281
					ps384.OverlayValues[282] = d282
					ps384.OverlayValues[283] = d283
					ps384.OverlayValues[284] = d284
					ps384.OverlayValues[285] = d285
					ps384.OverlayValues[286] = d286
					ps384.OverlayValues[287] = d287
					ps384.OverlayValues[289] = d289
					ps384.OverlayValues[290] = d290
					ps384.OverlayValues[291] = d291
					ps384.OverlayValues[292] = d292
					ps384.OverlayValues[293] = d293
					ps384.OverlayValues[294] = d294
					ps384.OverlayValues[295] = d295
					ps384.OverlayValues[296] = d296
					ps384.OverlayValues[298] = d298
					ps384.OverlayValues[299] = d299
					ps384.OverlayValues[300] = d300
					snap385 := d8
					snap386 := d9
					snap387 := d10
					snap388 := d11
					snap389 := d12
					snap390 := d13
					snap391 := d14
					snap392 := d15
					snap393 := d16
					snap394 := d17
					snap395 := d18
					snap396 := d19
					snap397 := d20
					snap398 := d21
					snap399 := d22
					snap400 := d25
					snap401 := d45
					snap402 := d64
					snap403 := d65
					snap404 := d66
					snap405 := d67
					snap406 := d68
					snap407 := d70
					snap408 := d71
					snap409 := d72
					snap410 := d73
					snap411 := d74
					snap412 := d75
					snap413 := d76
					snap414 := d79
					snap415 := d145
					snap416 := d146
					snap417 := d147
					snap418 := d148
					snap419 := d149
					snap420 := d150
					snap421 := d152
					snap422 := d153
					snap423 := d154
					snap424 := d155
					snap425 := d156
					snap426 := d157
					snap427 := d158
					snap428 := d159
					snap429 := d160
					snap430 := d163
					snap431 := d164
					snap432 := d165
					snap433 := d166
					snap434 := d269
					snap435 := d270
					snap436 := d271
					snap437 := d272
					snap438 := d273
					snap439 := d274
					snap440 := d275
					snap441 := d276
					snap442 := d277
					snap443 := d278
					snap444 := d279
					snap445 := d280
					snap446 := d281
					snap447 := d282
					snap448 := d283
					snap449 := d284
					snap450 := d285
					snap451 := d286
					snap452 := d287
					snap453 := d289
					snap454 := d290
					snap455 := d291
					snap456 := d292
					snap457 := d293
					snap458 := d294
					snap459 := d295
					snap460 := d296
					snap461 := d298
					snap462 := d299
					snap463 := d300
					alloc464 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps384)
					}
					ctx.RestoreAllocState(alloc464)
					d8 = snap385
					d9 = snap386
					d10 = snap387
					d11 = snap388
					d12 = snap389
					d13 = snap390
					d14 = snap391
					d15 = snap392
					d16 = snap393
					d17 = snap394
					d18 = snap395
					d19 = snap396
					d20 = snap397
					d21 = snap398
					d22 = snap399
					d25 = snap400
					d45 = snap401
					d64 = snap402
					d65 = snap403
					d66 = snap404
					d67 = snap405
					d68 = snap406
					d70 = snap407
					d71 = snap408
					d72 = snap409
					d73 = snap410
					d74 = snap411
					d75 = snap412
					d76 = snap413
					d79 = snap414
					d145 = snap415
					d146 = snap416
					d147 = snap417
					d148 = snap418
					d149 = snap419
					d150 = snap420
					d152 = snap421
					d153 = snap422
					d154 = snap423
					d155 = snap424
					d156 = snap425
					d157 = snap426
					d158 = snap427
					d159 = snap428
					d160 = snap429
					d163 = snap430
					d164 = snap431
					d165 = snap432
					d166 = snap433
					d269 = snap434
					d270 = snap435
					d271 = snap436
					d272 = snap437
					d273 = snap438
					d274 = snap439
					d275 = snap440
					d276 = snap441
					d277 = snap442
					d278 = snap443
					d279 = snap444
					d280 = snap445
					d281 = snap446
					d282 = snap447
					d283 = snap448
					d284 = snap449
					d285 = snap450
					d286 = snap451
					d287 = snap452
					d289 = snap453
					d290 = snap454
					d291 = snap455
					d292 = snap456
					d293 = snap457
					d294 = snap458
					d295 = snap459
					d296 = snap460
					d298 = snap461
					d299 = snap462
					d300 = snap463
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps383)
					}
					return result
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d465 := ps.PhiValues[0]
							ctx.EnsureDesc(&d465)
							if phiHomeOK6 {
								ctx.EmitMovToReg(r4, d465)
							} else {
								ctx.EmitStoreToStack(d465, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d466 := ps.PhiValues[1]
							ctx.EnsureDesc(&d466)
							if phiHomeOK7 {
								ctx.EmitMovToReg(r5, d466)
							} else {
								ctx.EmitStoreToStack(d466, int32(bbs[10].PhiBase)+int32(16))
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != LocNone {
						d269 = ps.OverlayValues[269]
					}
					if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
						d270 = ps.OverlayValues[270]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != LocNone {
						d279 = ps.OverlayValues[279]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != LocNone {
						d281 = ps.OverlayValues[281]
					}
					if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != LocNone {
						d282 = ps.OverlayValues[282]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
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
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != LocNone {
						d465 = ps.OverlayValues[465]
					}
					if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != LocNone {
						d466 = ps.OverlayValues[466]
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
					var d467 JITValueDesc
					if d17.SliceSizeKnown {
						d467 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d467 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d467 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d467 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d467 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d467)
					ctx.EnsureDescsTogether(&d15, &d467)
					var d468 JITValueDesc
					if d15.Loc == LocImm && d467.Loc == LocImm {
						d468 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d467.Imm.Int())}
					} else if d467.Loc == LocImm {
						r22 := ctx.AllocRegExcept(d15.Reg)
						if d467.Imm.Int() >= -2147483648 && d467.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d467.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d467.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, RegR11)
						}
						d468 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedLess}
						ctx.BindReg(r22, &d468)
					} else if d15.Loc == LocImm {
						r23 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d467.Reg)
						d468 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondSignedLess}
						ctx.BindReg(r23, &d468)
					} else {
						r24 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpInt64(d15.Reg, d467.Reg)
						d468 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedLess}
						ctx.BindReg(r24, &d468)
					}
					ctx.FreeDesc(&d467)
					d469 = d468
					ctx.EnsureDesc(&d469)
					if d469.Loc != LocImm && d469.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d469.Loc == LocImm {
						if d469.Imm.Bool() {
							if ps.General {
							}
							ps470 := PhiState{General: ps.General}
							ps470.OverlayValues = make([]JITValueDesc, 470)
							ps470.OverlayValues[8] = d8
							ps470.OverlayValues[9] = d9
							ps470.OverlayValues[10] = d10
							ps470.OverlayValues[11] = d11
							ps470.OverlayValues[12] = d12
							ps470.OverlayValues[13] = d13
							ps470.OverlayValues[14] = d14
							ps470.OverlayValues[15] = d15
							ps470.OverlayValues[16] = d16
							ps470.OverlayValues[17] = d17
							ps470.OverlayValues[18] = d18
							ps470.OverlayValues[19] = d19
							ps470.OverlayValues[20] = d20
							ps470.OverlayValues[21] = d21
							ps470.OverlayValues[22] = d22
							ps470.OverlayValues[25] = d25
							ps470.OverlayValues[45] = d45
							ps470.OverlayValues[64] = d64
							ps470.OverlayValues[65] = d65
							ps470.OverlayValues[66] = d66
							ps470.OverlayValues[67] = d67
							ps470.OverlayValues[68] = d68
							ps470.OverlayValues[70] = d70
							ps470.OverlayValues[71] = d71
							ps470.OverlayValues[72] = d72
							ps470.OverlayValues[73] = d73
							ps470.OverlayValues[74] = d74
							ps470.OverlayValues[75] = d75
							ps470.OverlayValues[76] = d76
							ps470.OverlayValues[79] = d79
							ps470.OverlayValues[145] = d145
							ps470.OverlayValues[146] = d146
							ps470.OverlayValues[147] = d147
							ps470.OverlayValues[148] = d148
							ps470.OverlayValues[149] = d149
							ps470.OverlayValues[150] = d150
							ps470.OverlayValues[152] = d152
							ps470.OverlayValues[153] = d153
							ps470.OverlayValues[154] = d154
							ps470.OverlayValues[155] = d155
							ps470.OverlayValues[156] = d156
							ps470.OverlayValues[157] = d157
							ps470.OverlayValues[158] = d158
							ps470.OverlayValues[159] = d159
							ps470.OverlayValues[160] = d160
							ps470.OverlayValues[163] = d163
							ps470.OverlayValues[164] = d164
							ps470.OverlayValues[165] = d165
							ps470.OverlayValues[166] = d166
							ps470.OverlayValues[269] = d269
							ps470.OverlayValues[270] = d270
							ps470.OverlayValues[271] = d271
							ps470.OverlayValues[272] = d272
							ps470.OverlayValues[273] = d273
							ps470.OverlayValues[274] = d274
							ps470.OverlayValues[275] = d275
							ps470.OverlayValues[276] = d276
							ps470.OverlayValues[277] = d277
							ps470.OverlayValues[278] = d278
							ps470.OverlayValues[279] = d279
							ps470.OverlayValues[280] = d280
							ps470.OverlayValues[281] = d281
							ps470.OverlayValues[282] = d282
							ps470.OverlayValues[283] = d283
							ps470.OverlayValues[284] = d284
							ps470.OverlayValues[285] = d285
							ps470.OverlayValues[286] = d286
							ps470.OverlayValues[287] = d287
							ps470.OverlayValues[289] = d289
							ps470.OverlayValues[290] = d290
							ps470.OverlayValues[291] = d291
							ps470.OverlayValues[292] = d292
							ps470.OverlayValues[293] = d293
							ps470.OverlayValues[294] = d294
							ps470.OverlayValues[295] = d295
							ps470.OverlayValues[296] = d296
							ps470.OverlayValues[298] = d298
							ps470.OverlayValues[299] = d299
							ps470.OverlayValues[300] = d300
							ps470.OverlayValues[465] = d465
							ps470.OverlayValues[466] = d466
							ps470.OverlayValues[467] = d467
							ps470.OverlayValues[468] = d468
							ps470.OverlayValues[469] = d469
							return bbs[13].RenderPS(ps470)
						}
						if ps.General {
						}
						ps471 := PhiState{General: ps.General}
						ps471.OverlayValues = make([]JITValueDesc, 470)
						ps471.OverlayValues[8] = d8
						ps471.OverlayValues[9] = d9
						ps471.OverlayValues[10] = d10
						ps471.OverlayValues[11] = d11
						ps471.OverlayValues[12] = d12
						ps471.OverlayValues[13] = d13
						ps471.OverlayValues[14] = d14
						ps471.OverlayValues[15] = d15
						ps471.OverlayValues[16] = d16
						ps471.OverlayValues[17] = d17
						ps471.OverlayValues[18] = d18
						ps471.OverlayValues[19] = d19
						ps471.OverlayValues[20] = d20
						ps471.OverlayValues[21] = d21
						ps471.OverlayValues[22] = d22
						ps471.OverlayValues[25] = d25
						ps471.OverlayValues[45] = d45
						ps471.OverlayValues[64] = d64
						ps471.OverlayValues[65] = d65
						ps471.OverlayValues[66] = d66
						ps471.OverlayValues[67] = d67
						ps471.OverlayValues[68] = d68
						ps471.OverlayValues[70] = d70
						ps471.OverlayValues[71] = d71
						ps471.OverlayValues[72] = d72
						ps471.OverlayValues[73] = d73
						ps471.OverlayValues[74] = d74
						ps471.OverlayValues[75] = d75
						ps471.OverlayValues[76] = d76
						ps471.OverlayValues[79] = d79
						ps471.OverlayValues[145] = d145
						ps471.OverlayValues[146] = d146
						ps471.OverlayValues[147] = d147
						ps471.OverlayValues[148] = d148
						ps471.OverlayValues[149] = d149
						ps471.OverlayValues[150] = d150
						ps471.OverlayValues[152] = d152
						ps471.OverlayValues[153] = d153
						ps471.OverlayValues[154] = d154
						ps471.OverlayValues[155] = d155
						ps471.OverlayValues[156] = d156
						ps471.OverlayValues[157] = d157
						ps471.OverlayValues[158] = d158
						ps471.OverlayValues[159] = d159
						ps471.OverlayValues[160] = d160
						ps471.OverlayValues[163] = d163
						ps471.OverlayValues[164] = d164
						ps471.OverlayValues[165] = d165
						ps471.OverlayValues[166] = d166
						ps471.OverlayValues[269] = d269
						ps471.OverlayValues[270] = d270
						ps471.OverlayValues[271] = d271
						ps471.OverlayValues[272] = d272
						ps471.OverlayValues[273] = d273
						ps471.OverlayValues[274] = d274
						ps471.OverlayValues[275] = d275
						ps471.OverlayValues[276] = d276
						ps471.OverlayValues[277] = d277
						ps471.OverlayValues[278] = d278
						ps471.OverlayValues[279] = d279
						ps471.OverlayValues[280] = d280
						ps471.OverlayValues[281] = d281
						ps471.OverlayValues[282] = d282
						ps471.OverlayValues[283] = d283
						ps471.OverlayValues[284] = d284
						ps471.OverlayValues[285] = d285
						ps471.OverlayValues[286] = d286
						ps471.OverlayValues[287] = d287
						ps471.OverlayValues[289] = d289
						ps471.OverlayValues[290] = d290
						ps471.OverlayValues[291] = d291
						ps471.OverlayValues[292] = d292
						ps471.OverlayValues[293] = d293
						ps471.OverlayValues[294] = d294
						ps471.OverlayValues[295] = d295
						ps471.OverlayValues[296] = d296
						ps471.OverlayValues[298] = d298
						ps471.OverlayValues[299] = d299
						ps471.OverlayValues[300] = d300
						ps471.OverlayValues[465] = d465
						ps471.OverlayValues[466] = d466
						ps471.OverlayValues[467] = d467
						ps471.OverlayValues[468] = d468
						ps471.OverlayValues[469] = d469
						return bbs[12].RenderPS(ps471)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d472 := ps.PhiValues[0]
							ctx.EnsureDesc(&d472)
							if phiHomeOK6 {
								ctx.EmitMovToReg(r4, d472)
							} else {
								ctx.EmitStoreToStack(d472, int32(bbs[10].PhiBase)+int32(0))
							}
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d473 := ps.PhiValues[1]
							ctx.EnsureDesc(&d473)
							if phiHomeOK7 {
								ctx.EmitMovToReg(r5, d473)
							} else {
								ctx.EmitStoreToStack(d473, int32(bbs[10].PhiBase)+int32(16))
							}
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					ctx.EmitJump(d469.Condition, lbl14)
					snap474 := d8
					snap475 := d9
					snap476 := d10
					snap477 := d11
					snap478 := d12
					snap479 := d13
					snap480 := d14
					snap481 := d15
					snap482 := d16
					snap483 := d17
					snap484 := d18
					snap485 := d19
					snap486 := d20
					snap487 := d21
					snap488 := d22
					snap489 := d25
					snap490 := d45
					snap491 := d64
					snap492 := d65
					snap493 := d66
					snap494 := d67
					snap495 := d68
					snap496 := d70
					snap497 := d71
					snap498 := d72
					snap499 := d73
					snap500 := d74
					snap501 := d75
					snap502 := d76
					snap503 := d79
					snap504 := d145
					snap505 := d146
					snap506 := d147
					snap507 := d148
					snap508 := d149
					snap509 := d150
					snap510 := d152
					snap511 := d153
					snap512 := d154
					snap513 := d155
					snap514 := d156
					snap515 := d157
					snap516 := d158
					snap517 := d159
					snap518 := d160
					snap519 := d163
					snap520 := d164
					snap521 := d165
					snap522 := d166
					snap523 := d269
					snap524 := d270
					snap525 := d271
					snap526 := d272
					snap527 := d273
					snap528 := d274
					snap529 := d275
					snap530 := d276
					snap531 := d277
					snap532 := d278
					snap533 := d279
					snap534 := d280
					snap535 := d281
					snap536 := d282
					snap537 := d283
					snap538 := d284
					snap539 := d285
					snap540 := d286
					snap541 := d287
					snap542 := d289
					snap543 := d290
					snap544 := d291
					snap545 := d292
					snap546 := d293
					snap547 := d294
					snap548 := d295
					snap549 := d296
					snap550 := d298
					snap551 := d299
					snap552 := d300
					snap553 := d465
					snap554 := d466
					snap555 := d467
					snap556 := d468
					snap557 := d469
					snap558 := d472
					snap559 := d473
					alloc560 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc560)
					d8 = snap474
					d9 = snap475
					d10 = snap476
					d11 = snap477
					d12 = snap478
					d13 = snap479
					d14 = snap480
					d15 = snap481
					d16 = snap482
					d17 = snap483
					d18 = snap484
					d19 = snap485
					d20 = snap486
					d21 = snap487
					d22 = snap488
					d25 = snap489
					d45 = snap490
					d64 = snap491
					d65 = snap492
					d66 = snap493
					d67 = snap494
					d68 = snap495
					d70 = snap496
					d71 = snap497
					d72 = snap498
					d73 = snap499
					d74 = snap500
					d75 = snap501
					d76 = snap502
					d79 = snap503
					d145 = snap504
					d146 = snap505
					d147 = snap506
					d148 = snap507
					d149 = snap508
					d150 = snap509
					d152 = snap510
					d153 = snap511
					d154 = snap512
					d155 = snap513
					d156 = snap514
					d157 = snap515
					d158 = snap516
					d159 = snap517
					d160 = snap518
					d163 = snap519
					d164 = snap520
					d165 = snap521
					d166 = snap522
					d269 = snap523
					d270 = snap524
					d271 = snap525
					d272 = snap526
					d273 = snap527
					d274 = snap528
					d275 = snap529
					d276 = snap530
					d277 = snap531
					d278 = snap532
					d279 = snap533
					d280 = snap534
					d281 = snap535
					d282 = snap536
					d283 = snap537
					d284 = snap538
					d285 = snap539
					d286 = snap540
					d287 = snap541
					d289 = snap542
					d290 = snap543
					d291 = snap544
					d292 = snap545
					d293 = snap546
					d294 = snap547
					d295 = snap548
					d296 = snap549
					d298 = snap550
					d299 = snap551
					d300 = snap552
					d465 = snap553
					d466 = snap554
					d467 = snap555
					d468 = snap556
					d469 = snap557
					d472 = snap558
					d473 = snap559
					ctx.RestoreAllocState(alloc560)
					d8 = snap474
					d9 = snap475
					d10 = snap476
					d11 = snap477
					d12 = snap478
					d13 = snap479
					d14 = snap480
					d15 = snap481
					d16 = snap482
					d17 = snap483
					d18 = snap484
					d19 = snap485
					d20 = snap486
					d21 = snap487
					d22 = snap488
					d25 = snap489
					d45 = snap490
					d64 = snap491
					d65 = snap492
					d66 = snap493
					d67 = snap494
					d68 = snap495
					d70 = snap496
					d71 = snap497
					d72 = snap498
					d73 = snap499
					d74 = snap500
					d75 = snap501
					d76 = snap502
					d79 = snap503
					d145 = snap504
					d146 = snap505
					d147 = snap506
					d148 = snap507
					d149 = snap508
					d150 = snap509
					d152 = snap510
					d153 = snap511
					d154 = snap512
					d155 = snap513
					d156 = snap514
					d157 = snap515
					d158 = snap516
					d159 = snap517
					d160 = snap518
					d163 = snap519
					d164 = snap520
					d165 = snap521
					d166 = snap522
					d269 = snap523
					d270 = snap524
					d271 = snap525
					d272 = snap526
					d273 = snap527
					d274 = snap528
					d275 = snap529
					d276 = snap530
					d277 = snap531
					d278 = snap532
					d279 = snap533
					d280 = snap534
					d281 = snap535
					d282 = snap536
					d283 = snap537
					d284 = snap538
					d285 = snap539
					d286 = snap540
					d287 = snap541
					d289 = snap542
					d290 = snap543
					d291 = snap544
					d292 = snap545
					d293 = snap546
					d294 = snap547
					d295 = snap548
					d296 = snap549
					d298 = snap550
					d299 = snap551
					d300 = snap552
					d465 = snap553
					d466 = snap554
					d467 = snap555
					d468 = snap556
					d469 = snap557
					d472 = snap558
					d473 = snap559
					ps561 := PhiState{General: true}
					ps561.OverlayValues = make([]JITValueDesc, 474)
					ps561.OverlayValues[8] = d8
					ps561.OverlayValues[9] = d9
					ps561.OverlayValues[10] = d10
					ps561.OverlayValues[11] = d11
					ps561.OverlayValues[12] = d12
					ps561.OverlayValues[13] = d13
					ps561.OverlayValues[14] = d14
					ps561.OverlayValues[15] = d15
					ps561.OverlayValues[16] = d16
					ps561.OverlayValues[17] = d17
					ps561.OverlayValues[18] = d18
					ps561.OverlayValues[19] = d19
					ps561.OverlayValues[20] = d20
					ps561.OverlayValues[21] = d21
					ps561.OverlayValues[22] = d22
					ps561.OverlayValues[25] = d25
					ps561.OverlayValues[45] = d45
					ps561.OverlayValues[64] = d64
					ps561.OverlayValues[65] = d65
					ps561.OverlayValues[66] = d66
					ps561.OverlayValues[67] = d67
					ps561.OverlayValues[68] = d68
					ps561.OverlayValues[70] = d70
					ps561.OverlayValues[71] = d71
					ps561.OverlayValues[72] = d72
					ps561.OverlayValues[73] = d73
					ps561.OverlayValues[74] = d74
					ps561.OverlayValues[75] = d75
					ps561.OverlayValues[76] = d76
					ps561.OverlayValues[79] = d79
					ps561.OverlayValues[145] = d145
					ps561.OverlayValues[146] = d146
					ps561.OverlayValues[147] = d147
					ps561.OverlayValues[148] = d148
					ps561.OverlayValues[149] = d149
					ps561.OverlayValues[150] = d150
					ps561.OverlayValues[152] = d152
					ps561.OverlayValues[153] = d153
					ps561.OverlayValues[154] = d154
					ps561.OverlayValues[155] = d155
					ps561.OverlayValues[156] = d156
					ps561.OverlayValues[157] = d157
					ps561.OverlayValues[158] = d158
					ps561.OverlayValues[159] = d159
					ps561.OverlayValues[160] = d160
					ps561.OverlayValues[163] = d163
					ps561.OverlayValues[164] = d164
					ps561.OverlayValues[165] = d165
					ps561.OverlayValues[166] = d166
					ps561.OverlayValues[269] = d269
					ps561.OverlayValues[270] = d270
					ps561.OverlayValues[271] = d271
					ps561.OverlayValues[272] = d272
					ps561.OverlayValues[273] = d273
					ps561.OverlayValues[274] = d274
					ps561.OverlayValues[275] = d275
					ps561.OverlayValues[276] = d276
					ps561.OverlayValues[277] = d277
					ps561.OverlayValues[278] = d278
					ps561.OverlayValues[279] = d279
					ps561.OverlayValues[280] = d280
					ps561.OverlayValues[281] = d281
					ps561.OverlayValues[282] = d282
					ps561.OverlayValues[283] = d283
					ps561.OverlayValues[284] = d284
					ps561.OverlayValues[285] = d285
					ps561.OverlayValues[286] = d286
					ps561.OverlayValues[287] = d287
					ps561.OverlayValues[289] = d289
					ps561.OverlayValues[290] = d290
					ps561.OverlayValues[291] = d291
					ps561.OverlayValues[292] = d292
					ps561.OverlayValues[293] = d293
					ps561.OverlayValues[294] = d294
					ps561.OverlayValues[295] = d295
					ps561.OverlayValues[296] = d296
					ps561.OverlayValues[298] = d298
					ps561.OverlayValues[299] = d299
					ps561.OverlayValues[300] = d300
					ps561.OverlayValues[465] = d465
					ps561.OverlayValues[466] = d466
					ps561.OverlayValues[467] = d467
					ps561.OverlayValues[468] = d468
					ps561.OverlayValues[469] = d469
					ps561.OverlayValues[472] = d472
					ps561.OverlayValues[473] = d473
					ps562 := PhiState{General: true}
					ps562.OverlayValues = make([]JITValueDesc, 474)
					ps562.OverlayValues[8] = d8
					ps562.OverlayValues[9] = d9
					ps562.OverlayValues[10] = d10
					ps562.OverlayValues[11] = d11
					ps562.OverlayValues[12] = d12
					ps562.OverlayValues[13] = d13
					ps562.OverlayValues[14] = d14
					ps562.OverlayValues[15] = d15
					ps562.OverlayValues[16] = d16
					ps562.OverlayValues[17] = d17
					ps562.OverlayValues[18] = d18
					ps562.OverlayValues[19] = d19
					ps562.OverlayValues[20] = d20
					ps562.OverlayValues[21] = d21
					ps562.OverlayValues[22] = d22
					ps562.OverlayValues[25] = d25
					ps562.OverlayValues[45] = d45
					ps562.OverlayValues[64] = d64
					ps562.OverlayValues[65] = d65
					ps562.OverlayValues[66] = d66
					ps562.OverlayValues[67] = d67
					ps562.OverlayValues[68] = d68
					ps562.OverlayValues[70] = d70
					ps562.OverlayValues[71] = d71
					ps562.OverlayValues[72] = d72
					ps562.OverlayValues[73] = d73
					ps562.OverlayValues[74] = d74
					ps562.OverlayValues[75] = d75
					ps562.OverlayValues[76] = d76
					ps562.OverlayValues[79] = d79
					ps562.OverlayValues[145] = d145
					ps562.OverlayValues[146] = d146
					ps562.OverlayValues[147] = d147
					ps562.OverlayValues[148] = d148
					ps562.OverlayValues[149] = d149
					ps562.OverlayValues[150] = d150
					ps562.OverlayValues[152] = d152
					ps562.OverlayValues[153] = d153
					ps562.OverlayValues[154] = d154
					ps562.OverlayValues[155] = d155
					ps562.OverlayValues[156] = d156
					ps562.OverlayValues[157] = d157
					ps562.OverlayValues[158] = d158
					ps562.OverlayValues[159] = d159
					ps562.OverlayValues[160] = d160
					ps562.OverlayValues[163] = d163
					ps562.OverlayValues[164] = d164
					ps562.OverlayValues[165] = d165
					ps562.OverlayValues[166] = d166
					ps562.OverlayValues[269] = d269
					ps562.OverlayValues[270] = d270
					ps562.OverlayValues[271] = d271
					ps562.OverlayValues[272] = d272
					ps562.OverlayValues[273] = d273
					ps562.OverlayValues[274] = d274
					ps562.OverlayValues[275] = d275
					ps562.OverlayValues[276] = d276
					ps562.OverlayValues[277] = d277
					ps562.OverlayValues[278] = d278
					ps562.OverlayValues[279] = d279
					ps562.OverlayValues[280] = d280
					ps562.OverlayValues[281] = d281
					ps562.OverlayValues[282] = d282
					ps562.OverlayValues[283] = d283
					ps562.OverlayValues[284] = d284
					ps562.OverlayValues[285] = d285
					ps562.OverlayValues[286] = d286
					ps562.OverlayValues[287] = d287
					ps562.OverlayValues[289] = d289
					ps562.OverlayValues[290] = d290
					ps562.OverlayValues[291] = d291
					ps562.OverlayValues[292] = d292
					ps562.OverlayValues[293] = d293
					ps562.OverlayValues[294] = d294
					ps562.OverlayValues[295] = d295
					ps562.OverlayValues[296] = d296
					ps562.OverlayValues[298] = d298
					ps562.OverlayValues[299] = d299
					ps562.OverlayValues[300] = d300
					ps562.OverlayValues[465] = d465
					ps562.OverlayValues[466] = d466
					ps562.OverlayValues[467] = d467
					ps562.OverlayValues[468] = d468
					ps562.OverlayValues[469] = d469
					ps562.OverlayValues[472] = d472
					ps562.OverlayValues[473] = d473
					snap563 := d8
					snap564 := d9
					snap565 := d10
					snap566 := d11
					snap567 := d12
					snap568 := d13
					snap569 := d14
					snap570 := d15
					snap571 := d16
					snap572 := d17
					snap573 := d18
					snap574 := d19
					snap575 := d20
					snap576 := d21
					snap577 := d22
					snap578 := d25
					snap579 := d45
					snap580 := d64
					snap581 := d65
					snap582 := d66
					snap583 := d67
					snap584 := d68
					snap585 := d70
					snap586 := d71
					snap587 := d72
					snap588 := d73
					snap589 := d74
					snap590 := d75
					snap591 := d76
					snap592 := d79
					snap593 := d145
					snap594 := d146
					snap595 := d147
					snap596 := d148
					snap597 := d149
					snap598 := d150
					snap599 := d152
					snap600 := d153
					snap601 := d154
					snap602 := d155
					snap603 := d156
					snap604 := d157
					snap605 := d158
					snap606 := d159
					snap607 := d160
					snap608 := d163
					snap609 := d164
					snap610 := d165
					snap611 := d166
					snap612 := d269
					snap613 := d270
					snap614 := d271
					snap615 := d272
					snap616 := d273
					snap617 := d274
					snap618 := d275
					snap619 := d276
					snap620 := d277
					snap621 := d278
					snap622 := d279
					snap623 := d280
					snap624 := d281
					snap625 := d282
					snap626 := d283
					snap627 := d284
					snap628 := d285
					snap629 := d286
					snap630 := d287
					snap631 := d289
					snap632 := d290
					snap633 := d291
					snap634 := d292
					snap635 := d293
					snap636 := d294
					snap637 := d295
					snap638 := d296
					snap639 := d298
					snap640 := d299
					snap641 := d300
					snap642 := d465
					snap643 := d466
					snap644 := d467
					snap645 := d468
					snap646 := d469
					snap647 := d472
					snap648 := d473
					alloc649 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps562)
					}
					ctx.RestoreAllocState(alloc649)
					d8 = snap563
					d9 = snap564
					d10 = snap565
					d11 = snap566
					d12 = snap567
					d13 = snap568
					d14 = snap569
					d15 = snap570
					d16 = snap571
					d17 = snap572
					d18 = snap573
					d19 = snap574
					d20 = snap575
					d21 = snap576
					d22 = snap577
					d25 = snap578
					d45 = snap579
					d64 = snap580
					d65 = snap581
					d66 = snap582
					d67 = snap583
					d68 = snap584
					d70 = snap585
					d71 = snap586
					d72 = snap587
					d73 = snap588
					d74 = snap589
					d75 = snap590
					d76 = snap591
					d79 = snap592
					d145 = snap593
					d146 = snap594
					d147 = snap595
					d148 = snap596
					d149 = snap597
					d150 = snap598
					d152 = snap599
					d153 = snap600
					d154 = snap601
					d155 = snap602
					d156 = snap603
					d157 = snap604
					d158 = snap605
					d159 = snap606
					d160 = snap607
					d163 = snap608
					d164 = snap609
					d165 = snap610
					d166 = snap611
					d269 = snap612
					d270 = snap613
					d271 = snap614
					d272 = snap615
					d273 = snap616
					d274 = snap617
					d275 = snap618
					d276 = snap619
					d277 = snap620
					d278 = snap621
					d279 = snap622
					d280 = snap623
					d281 = snap624
					d282 = snap625
					d283 = snap626
					d284 = snap627
					d285 = snap628
					d286 = snap629
					d287 = snap630
					d289 = snap631
					d290 = snap632
					d291 = snap633
					d292 = snap634
					d293 = snap635
					d294 = snap636
					d295 = snap637
					d296 = snap638
					d298 = snap639
					d299 = snap640
					d300 = snap641
					d465 = snap642
					d466 = snap643
					d467 = snap644
					d468 = snap645
					d469 = snap646
					d472 = snap647
					d473 = snap648
					if !bbs[13].Rendered {
						return bbs[13].RenderPS(ps561)
					}
					return result
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != LocNone {
						d269 = ps.OverlayValues[269]
					}
					if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
						d270 = ps.OverlayValues[270]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != LocNone {
						d279 = ps.OverlayValues[279]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != LocNone {
						d281 = ps.OverlayValues[281]
					}
					if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != LocNone {
						d282 = ps.OverlayValues[282]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
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
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != LocNone {
						d465 = ps.OverlayValues[465]
					}
					if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != LocNone {
						d466 = ps.OverlayValues[466]
					}
					if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != LocNone {
						d467 = ps.OverlayValues[467]
					}
					if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != LocNone {
						d468 = ps.OverlayValues[468]
					}
					if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != LocNone {
						d469 = ps.OverlayValues[469]
					}
					if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != LocNone {
						d472 = ps.OverlayValues[472]
					}
					if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != LocNone {
						d473 = ps.OverlayValues[473]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					d651 = ctx.EmitSliceElementAddress(&d17, &d15, 16)
					ctx.EnsureDesc(&d651)
					r25 := ctx.AllocRegExcept(d651.Reg)
					ctx.EmitMovRegMem(r25, d651.Reg, 8)
					ctx.EmitMovRegMem(d651.Reg, d651.Reg, 0)
					d650 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d651.Reg, Reg2: r25}
					ctx.BindReg(d651.Reg, &d650)
					ctx.BindReg(r25, &d650)
					ctx.EnsureDesc(&d650)
					d652 = d650
					_ = d652
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl19 := ctx.ReserveLabel()
					_ = lbl19
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d653 JITValueDesc
					if d652.Loc == LocImm {
						d653 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d652.Imm.Float())}
					} else if d652.Type == tagFloat && d652.Loc == LocReg {
						d653 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d652.Reg}
						ctx.BindReg(d652.Reg, &d653)
						ctx.BindReg(d652.Reg, &d653)
					} else if d652.Type == tagFloat && d652.Loc == LocRegPair {
						ctx.FreeReg(d652.Reg)
						d653 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d652.Reg2}
						ctx.BindReg(d652.Reg2, &d653)
						ctx.BindReg(d652.Reg2, &d653)
					} else {
						d653 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d652}, 1)
						d653.Type = tagFloat
						ctx.BindReg(d653.Reg, &d653)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d653)
					ctx.FreeDesc(&d650)
					ctx.EnsureDesc(&d15)
					d655 = ctx.EmitSliceElementAddress(&d19, &d15, 16)
					ctx.EnsureDesc(&d655)
					r26 := ctx.AllocRegExcept(d655.Reg)
					ctx.EmitMovRegMem(r26, d655.Reg, 8)
					ctx.EmitMovRegMem(d655.Reg, d655.Reg, 0)
					d654 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d655.Reg, Reg2: r26}
					ctx.BindReg(d655.Reg, &d654)
					ctx.BindReg(r26, &d654)
					ctx.EnsureDesc(&d654)
					d656 = d654
					_ = d656
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl20 := ctx.ReserveLabel()
					_ = lbl20
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d657 JITValueDesc
					if d656.Loc == LocImm {
						d657 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d656.Imm.Float())}
					} else if d656.Type == tagFloat && d656.Loc == LocReg {
						d657 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d656.Reg}
						ctx.BindReg(d656.Reg, &d657)
						ctx.BindReg(d656.Reg, &d657)
					} else if d656.Type == tagFloat && d656.Loc == LocRegPair {
						ctx.FreeReg(d656.Reg)
						d657 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d656.Reg2}
						ctx.BindReg(d656.Reg2, &d657)
						ctx.BindReg(d656.Reg2, &d657)
					} else {
						d657 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d656}, 1)
						d657.Type = tagFloat
						ctx.BindReg(d657.Reg, &d657)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d657)
					ctx.FreeDesc(&d654)
					ctx.EnsureDesc(&d653)
					ctx.EnsureDesc(&d657)
					ctx.EnsureDescsTogether(&d653, &d657)
					var d658 JITValueDesc
					if d653.Loc == LocImm && d657.Loc == LocImm {
						d658 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d653.Imm.Float() * d657.Imm.Float())}
					} else if d653.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d657.Reg)
						_, xBits := d653.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d657.Reg)
						d658 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d658)
					} else if d657.Loc == LocImm {
						_, yBits := d657.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d653.Reg, RegR11)
						d658 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d653.Reg}
						ctx.BindReg(d653.Reg, &d658)
					} else {
						ctx.EmitMulFloat64(d653.Reg, d657.Reg)
						d658 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d653.Reg}
						ctx.BindReg(d653.Reg, &d658)
					}
					if d658.Loc == LocReg && d653.Loc == LocReg && d658.Reg == d653.Reg {
						ctx.TransferReg(d653.Reg)
						d653.Loc = LocNone
					}
					ctx.FreeDesc(&d653)
					ctx.FreeDesc(&d657)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d658)
					ctx.EnsureDescsTogether(&d14, &d658)
					var d659 JITValueDesc
					if d14.Loc == LocImm && d658.Loc == LocImm {
						d659 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d14.Imm.Float() + d658.Imm.Float())}
					} else if d14.Loc == LocImm {
						var scratch Reg
						if phiHomeOK6 && r4 != d658.Reg {
							scratch = r4
						} else {
							scratch = ctx.AllocRegExcept(d658.Reg)
						}
						_, xBits := d14.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d658.Reg)
						d659 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d659)
					} else if d658.Loc == LocImm {
						var scratch Reg
						if phiHomeOK6 {
							scratch = r4
						} else {
							scratch = ctx.AllocRegExcept(d14.Reg)
						}
						ctx.EmitMovRegReg(scratch, d14.Reg)
						_, yBits := d658.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d659 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d659)
					} else {
						var r27 Reg
						if phiHomeOK6 && r4 != d658.Reg {
							r27 = r4
						} else {
							r27 = ctx.AllocRegExcept(d14.Reg, d658.Reg)
						}
						ctx.EmitMovRegReg(r27, d14.Reg)
						ctx.EmitAddFloat64(r27, d658.Reg)
						d659 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r27}
						ctx.BindReg(r27, &d659)
					}
					if d659.Loc == LocReg && d14.Loc == LocReg && d659.Reg == d14.Reg {
						ctx.TransferReg(d14.Reg)
						d14.Loc = LocNone
					}
					ctx.FreeDesc(&d658)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					var d660 JITValueDesc
					if d15.Loc == LocImm {
						d660 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK7 {
							scratch = r5
						} else {
							scratch = ctx.AllocRegExcept(d15.Reg)
						}
						ctx.EmitMovRegReg(scratch, d15.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d660 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d660)
					}
					if d660.Loc == LocReg && d15.Loc == LocReg && d660.Reg == d15.Reg {
						ctx.TransferReg(d15.Reg)
						d15.Loc = LocNone
					}
					if ps.General {
						ctx.SyncDesc(&d659)
						if d659.Loc == LocReg {
							ctx.ProtectReg(d659.Reg)
						} else if d659.Loc == LocRegPair {
							ctx.ProtectReg(d659.Reg)
							ctx.ProtectReg(d659.Reg2)
						}
						ctx.SyncDesc(&d660)
						if d660.Loc == LocReg {
							ctx.ProtectReg(d660.Reg)
						} else if d660.Loc == LocRegPair {
							ctx.ProtectReg(d660.Reg)
							ctx.ProtectReg(d660.Reg2)
						}
						d661 = d659
						if d661.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d661)
						if phiHomeOK6 {
							ctx.EmitMovToReg(r4, d661)
						} else {
							ctx.EmitStoreToStack(d661, int32(bbs[10].PhiBase)+int32(0))
						}
						d662 = d660
						if d662.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d662)
						if phiHomeOK7 {
							ctx.EmitMovToReg(r5, d662)
						} else {
							ctx.EmitStoreToStack(d662, int32(bbs[10].PhiBase)+int32(16))
						}
						if d659.Loc == LocReg {
							ctx.UnprotectReg(d659.Reg)
						} else if d659.Loc == LocRegPair {
							ctx.UnprotectReg(d659.Reg)
							ctx.UnprotectReg(d659.Reg2)
						}
						if d660.Loc == LocReg {
							ctx.UnprotectReg(d660.Reg)
						} else if d660.Loc == LocRegPair {
							ctx.UnprotectReg(d660.Reg)
							ctx.UnprotectReg(d660.Reg2)
						}
					}
					ps663 := PhiState{General: ps.General}
					ps663.OverlayValues = make([]JITValueDesc, 663)
					ps663.OverlayValues[8] = d8
					ps663.OverlayValues[9] = d9
					ps663.OverlayValues[10] = d10
					ps663.OverlayValues[11] = d11
					ps663.OverlayValues[12] = d12
					ps663.OverlayValues[13] = d13
					ps663.OverlayValues[14] = d14
					ps663.OverlayValues[15] = d15
					ps663.OverlayValues[16] = d16
					ps663.OverlayValues[17] = d17
					ps663.OverlayValues[18] = d18
					ps663.OverlayValues[19] = d19
					ps663.OverlayValues[20] = d20
					ps663.OverlayValues[21] = d21
					ps663.OverlayValues[22] = d22
					ps663.OverlayValues[25] = d25
					ps663.OverlayValues[45] = d45
					ps663.OverlayValues[64] = d64
					ps663.OverlayValues[65] = d65
					ps663.OverlayValues[66] = d66
					ps663.OverlayValues[67] = d67
					ps663.OverlayValues[68] = d68
					ps663.OverlayValues[70] = d70
					ps663.OverlayValues[71] = d71
					ps663.OverlayValues[72] = d72
					ps663.OverlayValues[73] = d73
					ps663.OverlayValues[74] = d74
					ps663.OverlayValues[75] = d75
					ps663.OverlayValues[76] = d76
					ps663.OverlayValues[79] = d79
					ps663.OverlayValues[145] = d145
					ps663.OverlayValues[146] = d146
					ps663.OverlayValues[147] = d147
					ps663.OverlayValues[148] = d148
					ps663.OverlayValues[149] = d149
					ps663.OverlayValues[150] = d150
					ps663.OverlayValues[152] = d152
					ps663.OverlayValues[153] = d153
					ps663.OverlayValues[154] = d154
					ps663.OverlayValues[155] = d155
					ps663.OverlayValues[156] = d156
					ps663.OverlayValues[157] = d157
					ps663.OverlayValues[158] = d158
					ps663.OverlayValues[159] = d159
					ps663.OverlayValues[160] = d160
					ps663.OverlayValues[163] = d163
					ps663.OverlayValues[164] = d164
					ps663.OverlayValues[165] = d165
					ps663.OverlayValues[166] = d166
					ps663.OverlayValues[269] = d269
					ps663.OverlayValues[270] = d270
					ps663.OverlayValues[271] = d271
					ps663.OverlayValues[272] = d272
					ps663.OverlayValues[273] = d273
					ps663.OverlayValues[274] = d274
					ps663.OverlayValues[275] = d275
					ps663.OverlayValues[276] = d276
					ps663.OverlayValues[277] = d277
					ps663.OverlayValues[278] = d278
					ps663.OverlayValues[279] = d279
					ps663.OverlayValues[280] = d280
					ps663.OverlayValues[281] = d281
					ps663.OverlayValues[282] = d282
					ps663.OverlayValues[283] = d283
					ps663.OverlayValues[284] = d284
					ps663.OverlayValues[285] = d285
					ps663.OverlayValues[286] = d286
					ps663.OverlayValues[287] = d287
					ps663.OverlayValues[289] = d289
					ps663.OverlayValues[290] = d290
					ps663.OverlayValues[291] = d291
					ps663.OverlayValues[292] = d292
					ps663.OverlayValues[293] = d293
					ps663.OverlayValues[294] = d294
					ps663.OverlayValues[295] = d295
					ps663.OverlayValues[296] = d296
					ps663.OverlayValues[298] = d298
					ps663.OverlayValues[299] = d299
					ps663.OverlayValues[300] = d300
					ps663.OverlayValues[465] = d465
					ps663.OverlayValues[466] = d466
					ps663.OverlayValues[467] = d467
					ps663.OverlayValues[468] = d468
					ps663.OverlayValues[469] = d469
					ps663.OverlayValues[472] = d472
					ps663.OverlayValues[473] = d473
					ps663.OverlayValues[650] = d650
					ps663.OverlayValues[651] = d651
					ps663.OverlayValues[652] = d652
					ps663.OverlayValues[653] = d653
					ps663.OverlayValues[654] = d654
					ps663.OverlayValues[655] = d655
					ps663.OverlayValues[656] = d656
					ps663.OverlayValues[657] = d657
					ps663.OverlayValues[658] = d658
					ps663.OverlayValues[659] = d659
					ps663.OverlayValues[660] = d660
					ps663.OverlayValues[661] = d661
					ps663.OverlayValues[662] = d662
					ps663.PhiValues = make([]JITValueDesc, 2)
					d664 = d659
					ps663.PhiValues[0] = d664
					d665 = d660
					ps663.PhiValues[1] = d665
					if ps663.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps663)
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != LocNone {
						d269 = ps.OverlayValues[269]
					}
					if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
						d270 = ps.OverlayValues[270]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != LocNone {
						d279 = ps.OverlayValues[279]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != LocNone {
						d281 = ps.OverlayValues[281]
					}
					if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != LocNone {
						d282 = ps.OverlayValues[282]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
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
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != LocNone {
						d465 = ps.OverlayValues[465]
					}
					if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != LocNone {
						d466 = ps.OverlayValues[466]
					}
					if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != LocNone {
						d467 = ps.OverlayValues[467]
					}
					if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != LocNone {
						d468 = ps.OverlayValues[468]
					}
					if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != LocNone {
						d469 = ps.OverlayValues[469]
					}
					if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != LocNone {
						d472 = ps.OverlayValues[472]
					}
					if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != LocNone {
						d473 = ps.OverlayValues[473]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					if len(ps.OverlayValues) > 661 && ps.OverlayValues[661].Loc != LocNone {
						d661 = ps.OverlayValues[661]
					}
					if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != LocNone {
						d662 = ps.OverlayValues[662]
					}
					if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != LocNone {
						d664 = ps.OverlayValues[664]
					}
					if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != LocNone {
						d665 = ps.OverlayValues[665]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					var d666 JITValueDesc
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d666 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d666.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d666.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d666.Reg, &d666)
						ctx.BindReg(d666.Reg2, &d666)
					} else {
						d666 = d8
					}
					d667 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("EUCLIDEAN")}
					var d668 JITValueDesc
					if d667.Loc == LocImm {
						ctx.TrackImm(d667.Imm)
						ptrWord, _ := d667.Imm.RawWords()
						d668 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d668.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d668.Reg2, uint64(len(d667.Imm.String())))
						ctx.BindReg(d668.Reg, &d668)
						ctx.BindReg(d668.Reg2, &d668)
					} else {
						d668 = d667
					}
					d669 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d666, d668}, 1)
					ctx.EmitAndRegImm32(d669.Reg, 1)
					d669.Type = tagBool
					ctx.BindReg(d669.Reg, &d669)
					d670 = d669
					ctx.EnsureDesc(&d670)
					if d670.Loc != LocImm && d670.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d670.Loc == LocImm {
						if d670.Imm.Bool() {
							if ps.General {
							}
							ps671 := PhiState{General: ps.General}
							ps671.OverlayValues = make([]JITValueDesc, 671)
							ps671.OverlayValues[8] = d8
							ps671.OverlayValues[9] = d9
							ps671.OverlayValues[10] = d10
							ps671.OverlayValues[11] = d11
							ps671.OverlayValues[12] = d12
							ps671.OverlayValues[13] = d13
							ps671.OverlayValues[14] = d14
							ps671.OverlayValues[15] = d15
							ps671.OverlayValues[16] = d16
							ps671.OverlayValues[17] = d17
							ps671.OverlayValues[18] = d18
							ps671.OverlayValues[19] = d19
							ps671.OverlayValues[20] = d20
							ps671.OverlayValues[21] = d21
							ps671.OverlayValues[22] = d22
							ps671.OverlayValues[25] = d25
							ps671.OverlayValues[45] = d45
							ps671.OverlayValues[64] = d64
							ps671.OverlayValues[65] = d65
							ps671.OverlayValues[66] = d66
							ps671.OverlayValues[67] = d67
							ps671.OverlayValues[68] = d68
							ps671.OverlayValues[70] = d70
							ps671.OverlayValues[71] = d71
							ps671.OverlayValues[72] = d72
							ps671.OverlayValues[73] = d73
							ps671.OverlayValues[74] = d74
							ps671.OverlayValues[75] = d75
							ps671.OverlayValues[76] = d76
							ps671.OverlayValues[79] = d79
							ps671.OverlayValues[145] = d145
							ps671.OverlayValues[146] = d146
							ps671.OverlayValues[147] = d147
							ps671.OverlayValues[148] = d148
							ps671.OverlayValues[149] = d149
							ps671.OverlayValues[150] = d150
							ps671.OverlayValues[152] = d152
							ps671.OverlayValues[153] = d153
							ps671.OverlayValues[154] = d154
							ps671.OverlayValues[155] = d155
							ps671.OverlayValues[156] = d156
							ps671.OverlayValues[157] = d157
							ps671.OverlayValues[158] = d158
							ps671.OverlayValues[159] = d159
							ps671.OverlayValues[160] = d160
							ps671.OverlayValues[163] = d163
							ps671.OverlayValues[164] = d164
							ps671.OverlayValues[165] = d165
							ps671.OverlayValues[166] = d166
							ps671.OverlayValues[269] = d269
							ps671.OverlayValues[270] = d270
							ps671.OverlayValues[271] = d271
							ps671.OverlayValues[272] = d272
							ps671.OverlayValues[273] = d273
							ps671.OverlayValues[274] = d274
							ps671.OverlayValues[275] = d275
							ps671.OverlayValues[276] = d276
							ps671.OverlayValues[277] = d277
							ps671.OverlayValues[278] = d278
							ps671.OverlayValues[279] = d279
							ps671.OverlayValues[280] = d280
							ps671.OverlayValues[281] = d281
							ps671.OverlayValues[282] = d282
							ps671.OverlayValues[283] = d283
							ps671.OverlayValues[284] = d284
							ps671.OverlayValues[285] = d285
							ps671.OverlayValues[286] = d286
							ps671.OverlayValues[287] = d287
							ps671.OverlayValues[289] = d289
							ps671.OverlayValues[290] = d290
							ps671.OverlayValues[291] = d291
							ps671.OverlayValues[292] = d292
							ps671.OverlayValues[293] = d293
							ps671.OverlayValues[294] = d294
							ps671.OverlayValues[295] = d295
							ps671.OverlayValues[296] = d296
							ps671.OverlayValues[298] = d298
							ps671.OverlayValues[299] = d299
							ps671.OverlayValues[300] = d300
							ps671.OverlayValues[465] = d465
							ps671.OverlayValues[466] = d466
							ps671.OverlayValues[467] = d467
							ps671.OverlayValues[468] = d468
							ps671.OverlayValues[469] = d469
							ps671.OverlayValues[472] = d472
							ps671.OverlayValues[473] = d473
							ps671.OverlayValues[650] = d650
							ps671.OverlayValues[651] = d651
							ps671.OverlayValues[652] = d652
							ps671.OverlayValues[653] = d653
							ps671.OverlayValues[654] = d654
							ps671.OverlayValues[655] = d655
							ps671.OverlayValues[656] = d656
							ps671.OverlayValues[657] = d657
							ps671.OverlayValues[658] = d658
							ps671.OverlayValues[659] = d659
							ps671.OverlayValues[660] = d660
							ps671.OverlayValues[661] = d661
							ps671.OverlayValues[662] = d662
							ps671.OverlayValues[664] = d664
							ps671.OverlayValues[665] = d665
							ps671.OverlayValues[666] = d666
							ps671.OverlayValues[667] = d667
							ps671.OverlayValues[668] = d668
							ps671.OverlayValues[669] = d669
							ps671.OverlayValues[670] = d670
							return bbs[14].RenderPS(ps671)
						}
						if ps.General {
							ctx.SyncDesc(&d14)
							if d14.Loc == LocReg {
								ctx.ProtectReg(d14.Reg)
							} else if d14.Loc == LocRegPair {
								ctx.ProtectReg(d14.Reg)
								ctx.ProtectReg(d14.Reg2)
							}
							d672 = d14
							if d672.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d672)
							ctx.EmitStoreToStack(d672, int32(bbs[4].PhiBase)+int32(0))
							if d14.Loc == LocReg {
								ctx.UnprotectReg(d14.Reg)
							} else if d14.Loc == LocRegPair {
								ctx.UnprotectReg(d14.Reg)
								ctx.UnprotectReg(d14.Reg2)
							}
						}
						ps673 := PhiState{General: ps.General}
						ps673.OverlayValues = make([]JITValueDesc, 673)
						ps673.OverlayValues[8] = d8
						ps673.OverlayValues[9] = d9
						ps673.OverlayValues[10] = d10
						ps673.OverlayValues[11] = d11
						ps673.OverlayValues[12] = d12
						ps673.OverlayValues[13] = d13
						ps673.OverlayValues[14] = d14
						ps673.OverlayValues[15] = d15
						ps673.OverlayValues[16] = d16
						ps673.OverlayValues[17] = d17
						ps673.OverlayValues[18] = d18
						ps673.OverlayValues[19] = d19
						ps673.OverlayValues[20] = d20
						ps673.OverlayValues[21] = d21
						ps673.OverlayValues[22] = d22
						ps673.OverlayValues[25] = d25
						ps673.OverlayValues[45] = d45
						ps673.OverlayValues[64] = d64
						ps673.OverlayValues[65] = d65
						ps673.OverlayValues[66] = d66
						ps673.OverlayValues[67] = d67
						ps673.OverlayValues[68] = d68
						ps673.OverlayValues[70] = d70
						ps673.OverlayValues[71] = d71
						ps673.OverlayValues[72] = d72
						ps673.OverlayValues[73] = d73
						ps673.OverlayValues[74] = d74
						ps673.OverlayValues[75] = d75
						ps673.OverlayValues[76] = d76
						ps673.OverlayValues[79] = d79
						ps673.OverlayValues[145] = d145
						ps673.OverlayValues[146] = d146
						ps673.OverlayValues[147] = d147
						ps673.OverlayValues[148] = d148
						ps673.OverlayValues[149] = d149
						ps673.OverlayValues[150] = d150
						ps673.OverlayValues[152] = d152
						ps673.OverlayValues[153] = d153
						ps673.OverlayValues[154] = d154
						ps673.OverlayValues[155] = d155
						ps673.OverlayValues[156] = d156
						ps673.OverlayValues[157] = d157
						ps673.OverlayValues[158] = d158
						ps673.OverlayValues[159] = d159
						ps673.OverlayValues[160] = d160
						ps673.OverlayValues[163] = d163
						ps673.OverlayValues[164] = d164
						ps673.OverlayValues[165] = d165
						ps673.OverlayValues[166] = d166
						ps673.OverlayValues[269] = d269
						ps673.OverlayValues[270] = d270
						ps673.OverlayValues[271] = d271
						ps673.OverlayValues[272] = d272
						ps673.OverlayValues[273] = d273
						ps673.OverlayValues[274] = d274
						ps673.OverlayValues[275] = d275
						ps673.OverlayValues[276] = d276
						ps673.OverlayValues[277] = d277
						ps673.OverlayValues[278] = d278
						ps673.OverlayValues[279] = d279
						ps673.OverlayValues[280] = d280
						ps673.OverlayValues[281] = d281
						ps673.OverlayValues[282] = d282
						ps673.OverlayValues[283] = d283
						ps673.OverlayValues[284] = d284
						ps673.OverlayValues[285] = d285
						ps673.OverlayValues[286] = d286
						ps673.OverlayValues[287] = d287
						ps673.OverlayValues[289] = d289
						ps673.OverlayValues[290] = d290
						ps673.OverlayValues[291] = d291
						ps673.OverlayValues[292] = d292
						ps673.OverlayValues[293] = d293
						ps673.OverlayValues[294] = d294
						ps673.OverlayValues[295] = d295
						ps673.OverlayValues[296] = d296
						ps673.OverlayValues[298] = d298
						ps673.OverlayValues[299] = d299
						ps673.OverlayValues[300] = d300
						ps673.OverlayValues[465] = d465
						ps673.OverlayValues[466] = d466
						ps673.OverlayValues[467] = d467
						ps673.OverlayValues[468] = d468
						ps673.OverlayValues[469] = d469
						ps673.OverlayValues[472] = d472
						ps673.OverlayValues[473] = d473
						ps673.OverlayValues[650] = d650
						ps673.OverlayValues[651] = d651
						ps673.OverlayValues[652] = d652
						ps673.OverlayValues[653] = d653
						ps673.OverlayValues[654] = d654
						ps673.OverlayValues[655] = d655
						ps673.OverlayValues[656] = d656
						ps673.OverlayValues[657] = d657
						ps673.OverlayValues[658] = d658
						ps673.OverlayValues[659] = d659
						ps673.OverlayValues[660] = d660
						ps673.OverlayValues[661] = d661
						ps673.OverlayValues[662] = d662
						ps673.OverlayValues[664] = d664
						ps673.OverlayValues[665] = d665
						ps673.OverlayValues[666] = d666
						ps673.OverlayValues[667] = d667
						ps673.OverlayValues[668] = d668
						ps673.OverlayValues[669] = d669
						ps673.OverlayValues[670] = d670
						ps673.OverlayValues[672] = d672
						ps673.PhiValues = make([]JITValueDesc, 1)
						d674 = d14
						ps673.PhiValues[0] = d674
						return bbs[4].RenderPS(ps673)
					}
					if !ps.General {
						ps.General = true
						return bbs[12].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d670.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl21)
					snap675 := d8
					snap676 := d9
					snap677 := d10
					snap678 := d11
					snap679 := d12
					snap680 := d13
					snap681 := d14
					snap682 := d15
					snap683 := d16
					snap684 := d17
					snap685 := d18
					snap686 := d19
					snap687 := d20
					snap688 := d21
					snap689 := d22
					snap690 := d25
					snap691 := d45
					snap692 := d64
					snap693 := d65
					snap694 := d66
					snap695 := d67
					snap696 := d68
					snap697 := d70
					snap698 := d71
					snap699 := d72
					snap700 := d73
					snap701 := d74
					snap702 := d75
					snap703 := d76
					snap704 := d79
					snap705 := d145
					snap706 := d146
					snap707 := d147
					snap708 := d148
					snap709 := d149
					snap710 := d150
					snap711 := d152
					snap712 := d153
					snap713 := d154
					snap714 := d155
					snap715 := d156
					snap716 := d157
					snap717 := d158
					snap718 := d159
					snap719 := d160
					snap720 := d163
					snap721 := d164
					snap722 := d165
					snap723 := d166
					snap724 := d269
					snap725 := d270
					snap726 := d271
					snap727 := d272
					snap728 := d273
					snap729 := d274
					snap730 := d275
					snap731 := d276
					snap732 := d277
					snap733 := d278
					snap734 := d279
					snap735 := d280
					snap736 := d281
					snap737 := d282
					snap738 := d283
					snap739 := d284
					snap740 := d285
					snap741 := d286
					snap742 := d287
					snap743 := d289
					snap744 := d290
					snap745 := d291
					snap746 := d292
					snap747 := d293
					snap748 := d294
					snap749 := d295
					snap750 := d296
					snap751 := d298
					snap752 := d299
					snap753 := d300
					snap754 := d465
					snap755 := d466
					snap756 := d467
					snap757 := d468
					snap758 := d469
					snap759 := d472
					snap760 := d473
					snap761 := d650
					snap762 := d651
					snap763 := d652
					snap764 := d653
					snap765 := d654
					snap766 := d655
					snap767 := d656
					snap768 := d657
					snap769 := d658
					snap770 := d659
					snap771 := d660
					snap772 := d661
					snap773 := d662
					snap774 := d664
					snap775 := d665
					snap776 := d666
					snap777 := d667
					snap778 := d668
					snap779 := d669
					snap780 := d670
					snap781 := d672
					snap782 := d674
					alloc783 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc783)
					d8 = snap675
					d9 = snap676
					d10 = snap677
					d11 = snap678
					d12 = snap679
					d13 = snap680
					d14 = snap681
					d15 = snap682
					d16 = snap683
					d17 = snap684
					d18 = snap685
					d19 = snap686
					d20 = snap687
					d21 = snap688
					d22 = snap689
					d25 = snap690
					d45 = snap691
					d64 = snap692
					d65 = snap693
					d66 = snap694
					d67 = snap695
					d68 = snap696
					d70 = snap697
					d71 = snap698
					d72 = snap699
					d73 = snap700
					d74 = snap701
					d75 = snap702
					d76 = snap703
					d79 = snap704
					d145 = snap705
					d146 = snap706
					d147 = snap707
					d148 = snap708
					d149 = snap709
					d150 = snap710
					d152 = snap711
					d153 = snap712
					d154 = snap713
					d155 = snap714
					d156 = snap715
					d157 = snap716
					d158 = snap717
					d159 = snap718
					d160 = snap719
					d163 = snap720
					d164 = snap721
					d165 = snap722
					d166 = snap723
					d269 = snap724
					d270 = snap725
					d271 = snap726
					d272 = snap727
					d273 = snap728
					d274 = snap729
					d275 = snap730
					d276 = snap731
					d277 = snap732
					d278 = snap733
					d279 = snap734
					d280 = snap735
					d281 = snap736
					d282 = snap737
					d283 = snap738
					d284 = snap739
					d285 = snap740
					d286 = snap741
					d287 = snap742
					d289 = snap743
					d290 = snap744
					d291 = snap745
					d292 = snap746
					d293 = snap747
					d294 = snap748
					d295 = snap749
					d296 = snap750
					d298 = snap751
					d299 = snap752
					d300 = snap753
					d465 = snap754
					d466 = snap755
					d467 = snap756
					d468 = snap757
					d469 = snap758
					d472 = snap759
					d473 = snap760
					d650 = snap761
					d651 = snap762
					d652 = snap763
					d653 = snap764
					d654 = snap765
					d655 = snap766
					d656 = snap767
					d657 = snap768
					d658 = snap769
					d659 = snap770
					d660 = snap771
					d661 = snap772
					d662 = snap773
					d664 = snap774
					d665 = snap775
					d666 = snap776
					d667 = snap777
					d668 = snap778
					d669 = snap779
					d670 = snap780
					d672 = snap781
					d674 = snap782
					ctx.MarkLabel(lbl21)
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d784 = d14
					if d784.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d784)
					ctx.EmitStoreToStack(d784, int32(bbs[4].PhiBase)+int32(0))
					if d14.Loc == LocReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc783)
					d8 = snap675
					d9 = snap676
					d10 = snap677
					d11 = snap678
					d12 = snap679
					d13 = snap680
					d14 = snap681
					d15 = snap682
					d16 = snap683
					d17 = snap684
					d18 = snap685
					d19 = snap686
					d20 = snap687
					d21 = snap688
					d22 = snap689
					d25 = snap690
					d45 = snap691
					d64 = snap692
					d65 = snap693
					d66 = snap694
					d67 = snap695
					d68 = snap696
					d70 = snap697
					d71 = snap698
					d72 = snap699
					d73 = snap700
					d74 = snap701
					d75 = snap702
					d76 = snap703
					d79 = snap704
					d145 = snap705
					d146 = snap706
					d147 = snap707
					d148 = snap708
					d149 = snap709
					d150 = snap710
					d152 = snap711
					d153 = snap712
					d154 = snap713
					d155 = snap714
					d156 = snap715
					d157 = snap716
					d158 = snap717
					d159 = snap718
					d160 = snap719
					d163 = snap720
					d164 = snap721
					d165 = snap722
					d166 = snap723
					d269 = snap724
					d270 = snap725
					d271 = snap726
					d272 = snap727
					d273 = snap728
					d274 = snap729
					d275 = snap730
					d276 = snap731
					d277 = snap732
					d278 = snap733
					d279 = snap734
					d280 = snap735
					d281 = snap736
					d282 = snap737
					d283 = snap738
					d284 = snap739
					d285 = snap740
					d286 = snap741
					d287 = snap742
					d289 = snap743
					d290 = snap744
					d291 = snap745
					d292 = snap746
					d293 = snap747
					d294 = snap748
					d295 = snap749
					d296 = snap750
					d298 = snap751
					d299 = snap752
					d300 = snap753
					d465 = snap754
					d466 = snap755
					d467 = snap756
					d468 = snap757
					d469 = snap758
					d472 = snap759
					d473 = snap760
					d650 = snap761
					d651 = snap762
					d652 = snap763
					d653 = snap764
					d654 = snap765
					d655 = snap766
					d656 = snap767
					d657 = snap768
					d658 = snap769
					d659 = snap770
					d660 = snap771
					d661 = snap772
					d662 = snap773
					d664 = snap774
					d665 = snap775
					d666 = snap776
					d667 = snap777
					d668 = snap778
					d669 = snap779
					d670 = snap780
					d672 = snap781
					d674 = snap782
					ps785 := PhiState{General: true}
					ps785.OverlayValues = make([]JITValueDesc, 785)
					ps785.OverlayValues[8] = d8
					ps785.OverlayValues[9] = d9
					ps785.OverlayValues[10] = d10
					ps785.OverlayValues[11] = d11
					ps785.OverlayValues[12] = d12
					ps785.OverlayValues[13] = d13
					ps785.OverlayValues[14] = d14
					ps785.OverlayValues[15] = d15
					ps785.OverlayValues[16] = d16
					ps785.OverlayValues[17] = d17
					ps785.OverlayValues[18] = d18
					ps785.OverlayValues[19] = d19
					ps785.OverlayValues[20] = d20
					ps785.OverlayValues[21] = d21
					ps785.OverlayValues[22] = d22
					ps785.OverlayValues[25] = d25
					ps785.OverlayValues[45] = d45
					ps785.OverlayValues[64] = d64
					ps785.OverlayValues[65] = d65
					ps785.OverlayValues[66] = d66
					ps785.OverlayValues[67] = d67
					ps785.OverlayValues[68] = d68
					ps785.OverlayValues[70] = d70
					ps785.OverlayValues[71] = d71
					ps785.OverlayValues[72] = d72
					ps785.OverlayValues[73] = d73
					ps785.OverlayValues[74] = d74
					ps785.OverlayValues[75] = d75
					ps785.OverlayValues[76] = d76
					ps785.OverlayValues[79] = d79
					ps785.OverlayValues[145] = d145
					ps785.OverlayValues[146] = d146
					ps785.OverlayValues[147] = d147
					ps785.OverlayValues[148] = d148
					ps785.OverlayValues[149] = d149
					ps785.OverlayValues[150] = d150
					ps785.OverlayValues[152] = d152
					ps785.OverlayValues[153] = d153
					ps785.OverlayValues[154] = d154
					ps785.OverlayValues[155] = d155
					ps785.OverlayValues[156] = d156
					ps785.OverlayValues[157] = d157
					ps785.OverlayValues[158] = d158
					ps785.OverlayValues[159] = d159
					ps785.OverlayValues[160] = d160
					ps785.OverlayValues[163] = d163
					ps785.OverlayValues[164] = d164
					ps785.OverlayValues[165] = d165
					ps785.OverlayValues[166] = d166
					ps785.OverlayValues[269] = d269
					ps785.OverlayValues[270] = d270
					ps785.OverlayValues[271] = d271
					ps785.OverlayValues[272] = d272
					ps785.OverlayValues[273] = d273
					ps785.OverlayValues[274] = d274
					ps785.OverlayValues[275] = d275
					ps785.OverlayValues[276] = d276
					ps785.OverlayValues[277] = d277
					ps785.OverlayValues[278] = d278
					ps785.OverlayValues[279] = d279
					ps785.OverlayValues[280] = d280
					ps785.OverlayValues[281] = d281
					ps785.OverlayValues[282] = d282
					ps785.OverlayValues[283] = d283
					ps785.OverlayValues[284] = d284
					ps785.OverlayValues[285] = d285
					ps785.OverlayValues[286] = d286
					ps785.OverlayValues[287] = d287
					ps785.OverlayValues[289] = d289
					ps785.OverlayValues[290] = d290
					ps785.OverlayValues[291] = d291
					ps785.OverlayValues[292] = d292
					ps785.OverlayValues[293] = d293
					ps785.OverlayValues[294] = d294
					ps785.OverlayValues[295] = d295
					ps785.OverlayValues[296] = d296
					ps785.OverlayValues[298] = d298
					ps785.OverlayValues[299] = d299
					ps785.OverlayValues[300] = d300
					ps785.OverlayValues[465] = d465
					ps785.OverlayValues[466] = d466
					ps785.OverlayValues[467] = d467
					ps785.OverlayValues[468] = d468
					ps785.OverlayValues[469] = d469
					ps785.OverlayValues[472] = d472
					ps785.OverlayValues[473] = d473
					ps785.OverlayValues[650] = d650
					ps785.OverlayValues[651] = d651
					ps785.OverlayValues[652] = d652
					ps785.OverlayValues[653] = d653
					ps785.OverlayValues[654] = d654
					ps785.OverlayValues[655] = d655
					ps785.OverlayValues[656] = d656
					ps785.OverlayValues[657] = d657
					ps785.OverlayValues[658] = d658
					ps785.OverlayValues[659] = d659
					ps785.OverlayValues[660] = d660
					ps785.OverlayValues[661] = d661
					ps785.OverlayValues[662] = d662
					ps785.OverlayValues[664] = d664
					ps785.OverlayValues[665] = d665
					ps785.OverlayValues[666] = d666
					ps785.OverlayValues[667] = d667
					ps785.OverlayValues[668] = d668
					ps785.OverlayValues[669] = d669
					ps785.OverlayValues[670] = d670
					ps785.OverlayValues[672] = d672
					ps785.OverlayValues[674] = d674
					ps785.OverlayValues[784] = d784
					ps786 := PhiState{General: true}
					ps786.OverlayValues = make([]JITValueDesc, 785)
					ps786.OverlayValues[8] = d8
					ps786.OverlayValues[9] = d9
					ps786.OverlayValues[10] = d10
					ps786.OverlayValues[11] = d11
					ps786.OverlayValues[12] = d12
					ps786.OverlayValues[13] = d13
					ps786.OverlayValues[14] = d14
					ps786.OverlayValues[15] = d15
					ps786.OverlayValues[16] = d16
					ps786.OverlayValues[17] = d17
					ps786.OverlayValues[18] = d18
					ps786.OverlayValues[19] = d19
					ps786.OverlayValues[20] = d20
					ps786.OverlayValues[21] = d21
					ps786.OverlayValues[22] = d22
					ps786.OverlayValues[25] = d25
					ps786.OverlayValues[45] = d45
					ps786.OverlayValues[64] = d64
					ps786.OverlayValues[65] = d65
					ps786.OverlayValues[66] = d66
					ps786.OverlayValues[67] = d67
					ps786.OverlayValues[68] = d68
					ps786.OverlayValues[70] = d70
					ps786.OverlayValues[71] = d71
					ps786.OverlayValues[72] = d72
					ps786.OverlayValues[73] = d73
					ps786.OverlayValues[74] = d74
					ps786.OverlayValues[75] = d75
					ps786.OverlayValues[76] = d76
					ps786.OverlayValues[79] = d79
					ps786.OverlayValues[145] = d145
					ps786.OverlayValues[146] = d146
					ps786.OverlayValues[147] = d147
					ps786.OverlayValues[148] = d148
					ps786.OverlayValues[149] = d149
					ps786.OverlayValues[150] = d150
					ps786.OverlayValues[152] = d152
					ps786.OverlayValues[153] = d153
					ps786.OverlayValues[154] = d154
					ps786.OverlayValues[155] = d155
					ps786.OverlayValues[156] = d156
					ps786.OverlayValues[157] = d157
					ps786.OverlayValues[158] = d158
					ps786.OverlayValues[159] = d159
					ps786.OverlayValues[160] = d160
					ps786.OverlayValues[163] = d163
					ps786.OverlayValues[164] = d164
					ps786.OverlayValues[165] = d165
					ps786.OverlayValues[166] = d166
					ps786.OverlayValues[269] = d269
					ps786.OverlayValues[270] = d270
					ps786.OverlayValues[271] = d271
					ps786.OverlayValues[272] = d272
					ps786.OverlayValues[273] = d273
					ps786.OverlayValues[274] = d274
					ps786.OverlayValues[275] = d275
					ps786.OverlayValues[276] = d276
					ps786.OverlayValues[277] = d277
					ps786.OverlayValues[278] = d278
					ps786.OverlayValues[279] = d279
					ps786.OverlayValues[280] = d280
					ps786.OverlayValues[281] = d281
					ps786.OverlayValues[282] = d282
					ps786.OverlayValues[283] = d283
					ps786.OverlayValues[284] = d284
					ps786.OverlayValues[285] = d285
					ps786.OverlayValues[286] = d286
					ps786.OverlayValues[287] = d287
					ps786.OverlayValues[289] = d289
					ps786.OverlayValues[290] = d290
					ps786.OverlayValues[291] = d291
					ps786.OverlayValues[292] = d292
					ps786.OverlayValues[293] = d293
					ps786.OverlayValues[294] = d294
					ps786.OverlayValues[295] = d295
					ps786.OverlayValues[296] = d296
					ps786.OverlayValues[298] = d298
					ps786.OverlayValues[299] = d299
					ps786.OverlayValues[300] = d300
					ps786.OverlayValues[465] = d465
					ps786.OverlayValues[466] = d466
					ps786.OverlayValues[467] = d467
					ps786.OverlayValues[468] = d468
					ps786.OverlayValues[469] = d469
					ps786.OverlayValues[472] = d472
					ps786.OverlayValues[473] = d473
					ps786.OverlayValues[650] = d650
					ps786.OverlayValues[651] = d651
					ps786.OverlayValues[652] = d652
					ps786.OverlayValues[653] = d653
					ps786.OverlayValues[654] = d654
					ps786.OverlayValues[655] = d655
					ps786.OverlayValues[656] = d656
					ps786.OverlayValues[657] = d657
					ps786.OverlayValues[658] = d658
					ps786.OverlayValues[659] = d659
					ps786.OverlayValues[660] = d660
					ps786.OverlayValues[661] = d661
					ps786.OverlayValues[662] = d662
					ps786.OverlayValues[664] = d664
					ps786.OverlayValues[665] = d665
					ps786.OverlayValues[666] = d666
					ps786.OverlayValues[667] = d667
					ps786.OverlayValues[668] = d668
					ps786.OverlayValues[669] = d669
					ps786.OverlayValues[670] = d670
					ps786.OverlayValues[672] = d672
					ps786.OverlayValues[674] = d674
					ps786.OverlayValues[784] = d784
					ps786.PhiValues = make([]JITValueDesc, 1)
					d787 = d14
					ps786.PhiValues[0] = d787
					snap788 := d8
					snap789 := d9
					snap790 := d10
					snap791 := d11
					snap792 := d12
					snap793 := d13
					snap794 := d14
					snap795 := d15
					snap796 := d16
					snap797 := d17
					snap798 := d18
					snap799 := d19
					snap800 := d20
					snap801 := d21
					snap802 := d22
					snap803 := d25
					snap804 := d45
					snap805 := d64
					snap806 := d65
					snap807 := d66
					snap808 := d67
					snap809 := d68
					snap810 := d70
					snap811 := d71
					snap812 := d72
					snap813 := d73
					snap814 := d74
					snap815 := d75
					snap816 := d76
					snap817 := d79
					snap818 := d145
					snap819 := d146
					snap820 := d147
					snap821 := d148
					snap822 := d149
					snap823 := d150
					snap824 := d152
					snap825 := d153
					snap826 := d154
					snap827 := d155
					snap828 := d156
					snap829 := d157
					snap830 := d158
					snap831 := d159
					snap832 := d160
					snap833 := d163
					snap834 := d164
					snap835 := d165
					snap836 := d166
					snap837 := d269
					snap838 := d270
					snap839 := d271
					snap840 := d272
					snap841 := d273
					snap842 := d274
					snap843 := d275
					snap844 := d276
					snap845 := d277
					snap846 := d278
					snap847 := d279
					snap848 := d280
					snap849 := d281
					snap850 := d282
					snap851 := d283
					snap852 := d284
					snap853 := d285
					snap854 := d286
					snap855 := d287
					snap856 := d289
					snap857 := d290
					snap858 := d291
					snap859 := d292
					snap860 := d293
					snap861 := d294
					snap862 := d295
					snap863 := d296
					snap864 := d298
					snap865 := d299
					snap866 := d300
					snap867 := d465
					snap868 := d466
					snap869 := d467
					snap870 := d468
					snap871 := d469
					snap872 := d472
					snap873 := d473
					snap874 := d650
					snap875 := d651
					snap876 := d652
					snap877 := d653
					snap878 := d654
					snap879 := d655
					snap880 := d656
					snap881 := d657
					snap882 := d658
					snap883 := d659
					snap884 := d660
					snap885 := d661
					snap886 := d662
					snap887 := d664
					snap888 := d665
					snap889 := d666
					snap890 := d667
					snap891 := d668
					snap892 := d669
					snap893 := d670
					snap894 := d672
					snap895 := d674
					snap896 := d784
					snap897 := d787
					alloc898 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps786)
					}
					ctx.RestoreAllocState(alloc898)
					d8 = snap788
					d9 = snap789
					d10 = snap790
					d11 = snap791
					d12 = snap792
					d13 = snap793
					d14 = snap794
					d15 = snap795
					d16 = snap796
					d17 = snap797
					d18 = snap798
					d19 = snap799
					d20 = snap800
					d21 = snap801
					d22 = snap802
					d25 = snap803
					d45 = snap804
					d64 = snap805
					d65 = snap806
					d66 = snap807
					d67 = snap808
					d68 = snap809
					d70 = snap810
					d71 = snap811
					d72 = snap812
					d73 = snap813
					d74 = snap814
					d75 = snap815
					d76 = snap816
					d79 = snap817
					d145 = snap818
					d146 = snap819
					d147 = snap820
					d148 = snap821
					d149 = snap822
					d150 = snap823
					d152 = snap824
					d153 = snap825
					d154 = snap826
					d155 = snap827
					d156 = snap828
					d157 = snap829
					d158 = snap830
					d159 = snap831
					d160 = snap832
					d163 = snap833
					d164 = snap834
					d165 = snap835
					d166 = snap836
					d269 = snap837
					d270 = snap838
					d271 = snap839
					d272 = snap840
					d273 = snap841
					d274 = snap842
					d275 = snap843
					d276 = snap844
					d277 = snap845
					d278 = snap846
					d279 = snap847
					d280 = snap848
					d281 = snap849
					d282 = snap850
					d283 = snap851
					d284 = snap852
					d285 = snap853
					d286 = snap854
					d287 = snap855
					d289 = snap856
					d290 = snap857
					d291 = snap858
					d292 = snap859
					d293 = snap860
					d294 = snap861
					d295 = snap862
					d296 = snap863
					d298 = snap864
					d299 = snap865
					d300 = snap866
					d465 = snap867
					d466 = snap868
					d467 = snap869
					d468 = snap870
					d469 = snap871
					d472 = snap872
					d473 = snap873
					d650 = snap874
					d651 = snap875
					d652 = snap876
					d653 = snap877
					d654 = snap878
					d655 = snap879
					d656 = snap880
					d657 = snap881
					d658 = snap882
					d659 = snap883
					d660 = snap884
					d661 = snap885
					d662 = snap886
					d664 = snap887
					d665 = snap888
					d666 = snap889
					d667 = snap890
					d668 = snap891
					d669 = snap892
					d670 = snap893
					d672 = snap894
					d674 = snap895
					d784 = snap896
					d787 = snap897
					if !bbs[14].Rendered {
						return bbs[14].RenderPS(ps785)
					}
					return result
					ctx.FreeDesc(&d669)
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != LocNone {
						d269 = ps.OverlayValues[269]
					}
					if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
						d270 = ps.OverlayValues[270]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != LocNone {
						d279 = ps.OverlayValues[279]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != LocNone {
						d281 = ps.OverlayValues[281]
					}
					if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != LocNone {
						d282 = ps.OverlayValues[282]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
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
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != LocNone {
						d465 = ps.OverlayValues[465]
					}
					if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != LocNone {
						d466 = ps.OverlayValues[466]
					}
					if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != LocNone {
						d467 = ps.OverlayValues[467]
					}
					if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != LocNone {
						d468 = ps.OverlayValues[468]
					}
					if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != LocNone {
						d469 = ps.OverlayValues[469]
					}
					if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != LocNone {
						d472 = ps.OverlayValues[472]
					}
					if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != LocNone {
						d473 = ps.OverlayValues[473]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					if len(ps.OverlayValues) > 661 && ps.OverlayValues[661].Loc != LocNone {
						d661 = ps.OverlayValues[661]
					}
					if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != LocNone {
						d662 = ps.OverlayValues[662]
					}
					if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != LocNone {
						d664 = ps.OverlayValues[664]
					}
					if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != LocNone {
						d665 = ps.OverlayValues[665]
					}
					if len(ps.OverlayValues) > 666 && ps.OverlayValues[666].Loc != LocNone {
						d666 = ps.OverlayValues[666]
					}
					if len(ps.OverlayValues) > 667 && ps.OverlayValues[667].Loc != LocNone {
						d667 = ps.OverlayValues[667]
					}
					if len(ps.OverlayValues) > 668 && ps.OverlayValues[668].Loc != LocNone {
						d668 = ps.OverlayValues[668]
					}
					if len(ps.OverlayValues) > 669 && ps.OverlayValues[669].Loc != LocNone {
						d669 = ps.OverlayValues[669]
					}
					if len(ps.OverlayValues) > 670 && ps.OverlayValues[670].Loc != LocNone {
						d670 = ps.OverlayValues[670]
					}
					if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != LocNone {
						d672 = ps.OverlayValues[672]
					}
					if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != LocNone {
						d674 = ps.OverlayValues[674]
					}
					if len(ps.OverlayValues) > 784 && ps.OverlayValues[784].Loc != LocNone {
						d784 = ps.OverlayValues[784]
					}
					if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != LocNone {
						d787 = ps.OverlayValues[787]
					}
					ctx.ReclaimUntrackedRegs()
					var d899 JITValueDesc
					if d19.SliceSizeKnown {
						d899 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d899 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d899 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d899 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d899 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d899)
					ctx.EnsureDescsTogether(&d15, &d899)
					var d900 JITValueDesc
					if d15.Loc == LocImm && d899.Loc == LocImm {
						d900 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d899.Imm.Int())}
					} else if d899.Loc == LocImm {
						r28 := ctx.AllocRegExcept(d15.Reg)
						if d899.Imm.Int() >= -2147483648 && d899.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d899.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d899.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, RegR11)
						}
						d900 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r28, Condition: CondSignedLess}
						ctx.BindReg(r28, &d900)
					} else if d15.Loc == LocImm {
						r29 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d899.Reg)
						d900 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r29, Condition: CondSignedLess}
						ctx.BindReg(r29, &d900)
					} else {
						r30 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpInt64(d15.Reg, d899.Reg)
						d900 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r30, Condition: CondSignedLess}
						ctx.BindReg(r30, &d900)
					}
					ctx.FreeDesc(&d899)
					d901 = d900
					ctx.EnsureDesc(&d901)
					if d901.Loc != LocImm && d901.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d901.Loc == LocImm {
						if d901.Imm.Bool() {
							if ps.General {
							}
							ps902 := PhiState{General: ps.General}
							ps902.OverlayValues = make([]JITValueDesc, 902)
							ps902.OverlayValues[8] = d8
							ps902.OverlayValues[9] = d9
							ps902.OverlayValues[10] = d10
							ps902.OverlayValues[11] = d11
							ps902.OverlayValues[12] = d12
							ps902.OverlayValues[13] = d13
							ps902.OverlayValues[14] = d14
							ps902.OverlayValues[15] = d15
							ps902.OverlayValues[16] = d16
							ps902.OverlayValues[17] = d17
							ps902.OverlayValues[18] = d18
							ps902.OverlayValues[19] = d19
							ps902.OverlayValues[20] = d20
							ps902.OverlayValues[21] = d21
							ps902.OverlayValues[22] = d22
							ps902.OverlayValues[25] = d25
							ps902.OverlayValues[45] = d45
							ps902.OverlayValues[64] = d64
							ps902.OverlayValues[65] = d65
							ps902.OverlayValues[66] = d66
							ps902.OverlayValues[67] = d67
							ps902.OverlayValues[68] = d68
							ps902.OverlayValues[70] = d70
							ps902.OverlayValues[71] = d71
							ps902.OverlayValues[72] = d72
							ps902.OverlayValues[73] = d73
							ps902.OverlayValues[74] = d74
							ps902.OverlayValues[75] = d75
							ps902.OverlayValues[76] = d76
							ps902.OverlayValues[79] = d79
							ps902.OverlayValues[145] = d145
							ps902.OverlayValues[146] = d146
							ps902.OverlayValues[147] = d147
							ps902.OverlayValues[148] = d148
							ps902.OverlayValues[149] = d149
							ps902.OverlayValues[150] = d150
							ps902.OverlayValues[152] = d152
							ps902.OverlayValues[153] = d153
							ps902.OverlayValues[154] = d154
							ps902.OverlayValues[155] = d155
							ps902.OverlayValues[156] = d156
							ps902.OverlayValues[157] = d157
							ps902.OverlayValues[158] = d158
							ps902.OverlayValues[159] = d159
							ps902.OverlayValues[160] = d160
							ps902.OverlayValues[163] = d163
							ps902.OverlayValues[164] = d164
							ps902.OverlayValues[165] = d165
							ps902.OverlayValues[166] = d166
							ps902.OverlayValues[269] = d269
							ps902.OverlayValues[270] = d270
							ps902.OverlayValues[271] = d271
							ps902.OverlayValues[272] = d272
							ps902.OverlayValues[273] = d273
							ps902.OverlayValues[274] = d274
							ps902.OverlayValues[275] = d275
							ps902.OverlayValues[276] = d276
							ps902.OverlayValues[277] = d277
							ps902.OverlayValues[278] = d278
							ps902.OverlayValues[279] = d279
							ps902.OverlayValues[280] = d280
							ps902.OverlayValues[281] = d281
							ps902.OverlayValues[282] = d282
							ps902.OverlayValues[283] = d283
							ps902.OverlayValues[284] = d284
							ps902.OverlayValues[285] = d285
							ps902.OverlayValues[286] = d286
							ps902.OverlayValues[287] = d287
							ps902.OverlayValues[289] = d289
							ps902.OverlayValues[290] = d290
							ps902.OverlayValues[291] = d291
							ps902.OverlayValues[292] = d292
							ps902.OverlayValues[293] = d293
							ps902.OverlayValues[294] = d294
							ps902.OverlayValues[295] = d295
							ps902.OverlayValues[296] = d296
							ps902.OverlayValues[298] = d298
							ps902.OverlayValues[299] = d299
							ps902.OverlayValues[300] = d300
							ps902.OverlayValues[465] = d465
							ps902.OverlayValues[466] = d466
							ps902.OverlayValues[467] = d467
							ps902.OverlayValues[468] = d468
							ps902.OverlayValues[469] = d469
							ps902.OverlayValues[472] = d472
							ps902.OverlayValues[473] = d473
							ps902.OverlayValues[650] = d650
							ps902.OverlayValues[651] = d651
							ps902.OverlayValues[652] = d652
							ps902.OverlayValues[653] = d653
							ps902.OverlayValues[654] = d654
							ps902.OverlayValues[655] = d655
							ps902.OverlayValues[656] = d656
							ps902.OverlayValues[657] = d657
							ps902.OverlayValues[658] = d658
							ps902.OverlayValues[659] = d659
							ps902.OverlayValues[660] = d660
							ps902.OverlayValues[661] = d661
							ps902.OverlayValues[662] = d662
							ps902.OverlayValues[664] = d664
							ps902.OverlayValues[665] = d665
							ps902.OverlayValues[666] = d666
							ps902.OverlayValues[667] = d667
							ps902.OverlayValues[668] = d668
							ps902.OverlayValues[669] = d669
							ps902.OverlayValues[670] = d670
							ps902.OverlayValues[672] = d672
							ps902.OverlayValues[674] = d674
							ps902.OverlayValues[784] = d784
							ps902.OverlayValues[787] = d787
							ps902.OverlayValues[899] = d899
							ps902.OverlayValues[900] = d900
							ps902.OverlayValues[901] = d901
							return bbs[11].RenderPS(ps902)
						}
						if ps.General {
						}
						ps903 := PhiState{General: ps.General}
						ps903.OverlayValues = make([]JITValueDesc, 902)
						ps903.OverlayValues[8] = d8
						ps903.OverlayValues[9] = d9
						ps903.OverlayValues[10] = d10
						ps903.OverlayValues[11] = d11
						ps903.OverlayValues[12] = d12
						ps903.OverlayValues[13] = d13
						ps903.OverlayValues[14] = d14
						ps903.OverlayValues[15] = d15
						ps903.OverlayValues[16] = d16
						ps903.OverlayValues[17] = d17
						ps903.OverlayValues[18] = d18
						ps903.OverlayValues[19] = d19
						ps903.OverlayValues[20] = d20
						ps903.OverlayValues[21] = d21
						ps903.OverlayValues[22] = d22
						ps903.OverlayValues[25] = d25
						ps903.OverlayValues[45] = d45
						ps903.OverlayValues[64] = d64
						ps903.OverlayValues[65] = d65
						ps903.OverlayValues[66] = d66
						ps903.OverlayValues[67] = d67
						ps903.OverlayValues[68] = d68
						ps903.OverlayValues[70] = d70
						ps903.OverlayValues[71] = d71
						ps903.OverlayValues[72] = d72
						ps903.OverlayValues[73] = d73
						ps903.OverlayValues[74] = d74
						ps903.OverlayValues[75] = d75
						ps903.OverlayValues[76] = d76
						ps903.OverlayValues[79] = d79
						ps903.OverlayValues[145] = d145
						ps903.OverlayValues[146] = d146
						ps903.OverlayValues[147] = d147
						ps903.OverlayValues[148] = d148
						ps903.OverlayValues[149] = d149
						ps903.OverlayValues[150] = d150
						ps903.OverlayValues[152] = d152
						ps903.OverlayValues[153] = d153
						ps903.OverlayValues[154] = d154
						ps903.OverlayValues[155] = d155
						ps903.OverlayValues[156] = d156
						ps903.OverlayValues[157] = d157
						ps903.OverlayValues[158] = d158
						ps903.OverlayValues[159] = d159
						ps903.OverlayValues[160] = d160
						ps903.OverlayValues[163] = d163
						ps903.OverlayValues[164] = d164
						ps903.OverlayValues[165] = d165
						ps903.OverlayValues[166] = d166
						ps903.OverlayValues[269] = d269
						ps903.OverlayValues[270] = d270
						ps903.OverlayValues[271] = d271
						ps903.OverlayValues[272] = d272
						ps903.OverlayValues[273] = d273
						ps903.OverlayValues[274] = d274
						ps903.OverlayValues[275] = d275
						ps903.OverlayValues[276] = d276
						ps903.OverlayValues[277] = d277
						ps903.OverlayValues[278] = d278
						ps903.OverlayValues[279] = d279
						ps903.OverlayValues[280] = d280
						ps903.OverlayValues[281] = d281
						ps903.OverlayValues[282] = d282
						ps903.OverlayValues[283] = d283
						ps903.OverlayValues[284] = d284
						ps903.OverlayValues[285] = d285
						ps903.OverlayValues[286] = d286
						ps903.OverlayValues[287] = d287
						ps903.OverlayValues[289] = d289
						ps903.OverlayValues[290] = d290
						ps903.OverlayValues[291] = d291
						ps903.OverlayValues[292] = d292
						ps903.OverlayValues[293] = d293
						ps903.OverlayValues[294] = d294
						ps903.OverlayValues[295] = d295
						ps903.OverlayValues[296] = d296
						ps903.OverlayValues[298] = d298
						ps903.OverlayValues[299] = d299
						ps903.OverlayValues[300] = d300
						ps903.OverlayValues[465] = d465
						ps903.OverlayValues[466] = d466
						ps903.OverlayValues[467] = d467
						ps903.OverlayValues[468] = d468
						ps903.OverlayValues[469] = d469
						ps903.OverlayValues[472] = d472
						ps903.OverlayValues[473] = d473
						ps903.OverlayValues[650] = d650
						ps903.OverlayValues[651] = d651
						ps903.OverlayValues[652] = d652
						ps903.OverlayValues[653] = d653
						ps903.OverlayValues[654] = d654
						ps903.OverlayValues[655] = d655
						ps903.OverlayValues[656] = d656
						ps903.OverlayValues[657] = d657
						ps903.OverlayValues[658] = d658
						ps903.OverlayValues[659] = d659
						ps903.OverlayValues[660] = d660
						ps903.OverlayValues[661] = d661
						ps903.OverlayValues[662] = d662
						ps903.OverlayValues[664] = d664
						ps903.OverlayValues[665] = d665
						ps903.OverlayValues[666] = d666
						ps903.OverlayValues[667] = d667
						ps903.OverlayValues[668] = d668
						ps903.OverlayValues[669] = d669
						ps903.OverlayValues[670] = d670
						ps903.OverlayValues[672] = d672
						ps903.OverlayValues[674] = d674
						ps903.OverlayValues[784] = d784
						ps903.OverlayValues[787] = d787
						ps903.OverlayValues[899] = d899
						ps903.OverlayValues[900] = d900
						ps903.OverlayValues[901] = d901
						return bbs[12].RenderPS(ps903)
					}
					if !ps.General {
						ps.General = true
						return bbs[13].RenderPS(ps)
					}
					ctx.EmitJump(d901.Condition, lbl12)
					snap904 := d8
					snap905 := d9
					snap906 := d10
					snap907 := d11
					snap908 := d12
					snap909 := d13
					snap910 := d14
					snap911 := d15
					snap912 := d16
					snap913 := d17
					snap914 := d18
					snap915 := d19
					snap916 := d20
					snap917 := d21
					snap918 := d22
					snap919 := d25
					snap920 := d45
					snap921 := d64
					snap922 := d65
					snap923 := d66
					snap924 := d67
					snap925 := d68
					snap926 := d70
					snap927 := d71
					snap928 := d72
					snap929 := d73
					snap930 := d74
					snap931 := d75
					snap932 := d76
					snap933 := d79
					snap934 := d145
					snap935 := d146
					snap936 := d147
					snap937 := d148
					snap938 := d149
					snap939 := d150
					snap940 := d152
					snap941 := d153
					snap942 := d154
					snap943 := d155
					snap944 := d156
					snap945 := d157
					snap946 := d158
					snap947 := d159
					snap948 := d160
					snap949 := d163
					snap950 := d164
					snap951 := d165
					snap952 := d166
					snap953 := d269
					snap954 := d270
					snap955 := d271
					snap956 := d272
					snap957 := d273
					snap958 := d274
					snap959 := d275
					snap960 := d276
					snap961 := d277
					snap962 := d278
					snap963 := d279
					snap964 := d280
					snap965 := d281
					snap966 := d282
					snap967 := d283
					snap968 := d284
					snap969 := d285
					snap970 := d286
					snap971 := d287
					snap972 := d289
					snap973 := d290
					snap974 := d291
					snap975 := d292
					snap976 := d293
					snap977 := d294
					snap978 := d295
					snap979 := d296
					snap980 := d298
					snap981 := d299
					snap982 := d300
					snap983 := d465
					snap984 := d466
					snap985 := d467
					snap986 := d468
					snap987 := d469
					snap988 := d472
					snap989 := d473
					snap990 := d650
					snap991 := d651
					snap992 := d652
					snap993 := d653
					snap994 := d654
					snap995 := d655
					snap996 := d656
					snap997 := d657
					snap998 := d658
					snap999 := d659
					snap1000 := d660
					snap1001 := d661
					snap1002 := d662
					snap1003 := d664
					snap1004 := d665
					snap1005 := d666
					snap1006 := d667
					snap1007 := d668
					snap1008 := d669
					snap1009 := d670
					snap1010 := d672
					snap1011 := d674
					snap1012 := d784
					snap1013 := d787
					snap1014 := d899
					snap1015 := d900
					snap1016 := d901
					alloc1017 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc1017)
					d8 = snap904
					d9 = snap905
					d10 = snap906
					d11 = snap907
					d12 = snap908
					d13 = snap909
					d14 = snap910
					d15 = snap911
					d16 = snap912
					d17 = snap913
					d18 = snap914
					d19 = snap915
					d20 = snap916
					d21 = snap917
					d22 = snap918
					d25 = snap919
					d45 = snap920
					d64 = snap921
					d65 = snap922
					d66 = snap923
					d67 = snap924
					d68 = snap925
					d70 = snap926
					d71 = snap927
					d72 = snap928
					d73 = snap929
					d74 = snap930
					d75 = snap931
					d76 = snap932
					d79 = snap933
					d145 = snap934
					d146 = snap935
					d147 = snap936
					d148 = snap937
					d149 = snap938
					d150 = snap939
					d152 = snap940
					d153 = snap941
					d154 = snap942
					d155 = snap943
					d156 = snap944
					d157 = snap945
					d158 = snap946
					d159 = snap947
					d160 = snap948
					d163 = snap949
					d164 = snap950
					d165 = snap951
					d166 = snap952
					d269 = snap953
					d270 = snap954
					d271 = snap955
					d272 = snap956
					d273 = snap957
					d274 = snap958
					d275 = snap959
					d276 = snap960
					d277 = snap961
					d278 = snap962
					d279 = snap963
					d280 = snap964
					d281 = snap965
					d282 = snap966
					d283 = snap967
					d284 = snap968
					d285 = snap969
					d286 = snap970
					d287 = snap971
					d289 = snap972
					d290 = snap973
					d291 = snap974
					d292 = snap975
					d293 = snap976
					d294 = snap977
					d295 = snap978
					d296 = snap979
					d298 = snap980
					d299 = snap981
					d300 = snap982
					d465 = snap983
					d466 = snap984
					d467 = snap985
					d468 = snap986
					d469 = snap987
					d472 = snap988
					d473 = snap989
					d650 = snap990
					d651 = snap991
					d652 = snap992
					d653 = snap993
					d654 = snap994
					d655 = snap995
					d656 = snap996
					d657 = snap997
					d658 = snap998
					d659 = snap999
					d660 = snap1000
					d661 = snap1001
					d662 = snap1002
					d664 = snap1003
					d665 = snap1004
					d666 = snap1005
					d667 = snap1006
					d668 = snap1007
					d669 = snap1008
					d670 = snap1009
					d672 = snap1010
					d674 = snap1011
					d784 = snap1012
					d787 = snap1013
					d899 = snap1014
					d900 = snap1015
					d901 = snap1016
					ctx.RestoreAllocState(alloc1017)
					d8 = snap904
					d9 = snap905
					d10 = snap906
					d11 = snap907
					d12 = snap908
					d13 = snap909
					d14 = snap910
					d15 = snap911
					d16 = snap912
					d17 = snap913
					d18 = snap914
					d19 = snap915
					d20 = snap916
					d21 = snap917
					d22 = snap918
					d25 = snap919
					d45 = snap920
					d64 = snap921
					d65 = snap922
					d66 = snap923
					d67 = snap924
					d68 = snap925
					d70 = snap926
					d71 = snap927
					d72 = snap928
					d73 = snap929
					d74 = snap930
					d75 = snap931
					d76 = snap932
					d79 = snap933
					d145 = snap934
					d146 = snap935
					d147 = snap936
					d148 = snap937
					d149 = snap938
					d150 = snap939
					d152 = snap940
					d153 = snap941
					d154 = snap942
					d155 = snap943
					d156 = snap944
					d157 = snap945
					d158 = snap946
					d159 = snap947
					d160 = snap948
					d163 = snap949
					d164 = snap950
					d165 = snap951
					d166 = snap952
					d269 = snap953
					d270 = snap954
					d271 = snap955
					d272 = snap956
					d273 = snap957
					d274 = snap958
					d275 = snap959
					d276 = snap960
					d277 = snap961
					d278 = snap962
					d279 = snap963
					d280 = snap964
					d281 = snap965
					d282 = snap966
					d283 = snap967
					d284 = snap968
					d285 = snap969
					d286 = snap970
					d287 = snap971
					d289 = snap972
					d290 = snap973
					d291 = snap974
					d292 = snap975
					d293 = snap976
					d294 = snap977
					d295 = snap978
					d296 = snap979
					d298 = snap980
					d299 = snap981
					d300 = snap982
					d465 = snap983
					d466 = snap984
					d467 = snap985
					d468 = snap986
					d469 = snap987
					d472 = snap988
					d473 = snap989
					d650 = snap990
					d651 = snap991
					d652 = snap992
					d653 = snap993
					d654 = snap994
					d655 = snap995
					d656 = snap996
					d657 = snap997
					d658 = snap998
					d659 = snap999
					d660 = snap1000
					d661 = snap1001
					d662 = snap1002
					d664 = snap1003
					d665 = snap1004
					d666 = snap1005
					d667 = snap1006
					d668 = snap1007
					d669 = snap1008
					d670 = snap1009
					d672 = snap1010
					d674 = snap1011
					d784 = snap1012
					d787 = snap1013
					d899 = snap1014
					d900 = snap1015
					d901 = snap1016
					ps1018 := PhiState{General: true}
					ps1018.OverlayValues = make([]JITValueDesc, 902)
					ps1018.OverlayValues[8] = d8
					ps1018.OverlayValues[9] = d9
					ps1018.OverlayValues[10] = d10
					ps1018.OverlayValues[11] = d11
					ps1018.OverlayValues[12] = d12
					ps1018.OverlayValues[13] = d13
					ps1018.OverlayValues[14] = d14
					ps1018.OverlayValues[15] = d15
					ps1018.OverlayValues[16] = d16
					ps1018.OverlayValues[17] = d17
					ps1018.OverlayValues[18] = d18
					ps1018.OverlayValues[19] = d19
					ps1018.OverlayValues[20] = d20
					ps1018.OverlayValues[21] = d21
					ps1018.OverlayValues[22] = d22
					ps1018.OverlayValues[25] = d25
					ps1018.OverlayValues[45] = d45
					ps1018.OverlayValues[64] = d64
					ps1018.OverlayValues[65] = d65
					ps1018.OverlayValues[66] = d66
					ps1018.OverlayValues[67] = d67
					ps1018.OverlayValues[68] = d68
					ps1018.OverlayValues[70] = d70
					ps1018.OverlayValues[71] = d71
					ps1018.OverlayValues[72] = d72
					ps1018.OverlayValues[73] = d73
					ps1018.OverlayValues[74] = d74
					ps1018.OverlayValues[75] = d75
					ps1018.OverlayValues[76] = d76
					ps1018.OverlayValues[79] = d79
					ps1018.OverlayValues[145] = d145
					ps1018.OverlayValues[146] = d146
					ps1018.OverlayValues[147] = d147
					ps1018.OverlayValues[148] = d148
					ps1018.OverlayValues[149] = d149
					ps1018.OverlayValues[150] = d150
					ps1018.OverlayValues[152] = d152
					ps1018.OverlayValues[153] = d153
					ps1018.OverlayValues[154] = d154
					ps1018.OverlayValues[155] = d155
					ps1018.OverlayValues[156] = d156
					ps1018.OverlayValues[157] = d157
					ps1018.OverlayValues[158] = d158
					ps1018.OverlayValues[159] = d159
					ps1018.OverlayValues[160] = d160
					ps1018.OverlayValues[163] = d163
					ps1018.OverlayValues[164] = d164
					ps1018.OverlayValues[165] = d165
					ps1018.OverlayValues[166] = d166
					ps1018.OverlayValues[269] = d269
					ps1018.OverlayValues[270] = d270
					ps1018.OverlayValues[271] = d271
					ps1018.OverlayValues[272] = d272
					ps1018.OverlayValues[273] = d273
					ps1018.OverlayValues[274] = d274
					ps1018.OverlayValues[275] = d275
					ps1018.OverlayValues[276] = d276
					ps1018.OverlayValues[277] = d277
					ps1018.OverlayValues[278] = d278
					ps1018.OverlayValues[279] = d279
					ps1018.OverlayValues[280] = d280
					ps1018.OverlayValues[281] = d281
					ps1018.OverlayValues[282] = d282
					ps1018.OverlayValues[283] = d283
					ps1018.OverlayValues[284] = d284
					ps1018.OverlayValues[285] = d285
					ps1018.OverlayValues[286] = d286
					ps1018.OverlayValues[287] = d287
					ps1018.OverlayValues[289] = d289
					ps1018.OverlayValues[290] = d290
					ps1018.OverlayValues[291] = d291
					ps1018.OverlayValues[292] = d292
					ps1018.OverlayValues[293] = d293
					ps1018.OverlayValues[294] = d294
					ps1018.OverlayValues[295] = d295
					ps1018.OverlayValues[296] = d296
					ps1018.OverlayValues[298] = d298
					ps1018.OverlayValues[299] = d299
					ps1018.OverlayValues[300] = d300
					ps1018.OverlayValues[465] = d465
					ps1018.OverlayValues[466] = d466
					ps1018.OverlayValues[467] = d467
					ps1018.OverlayValues[468] = d468
					ps1018.OverlayValues[469] = d469
					ps1018.OverlayValues[472] = d472
					ps1018.OverlayValues[473] = d473
					ps1018.OverlayValues[650] = d650
					ps1018.OverlayValues[651] = d651
					ps1018.OverlayValues[652] = d652
					ps1018.OverlayValues[653] = d653
					ps1018.OverlayValues[654] = d654
					ps1018.OverlayValues[655] = d655
					ps1018.OverlayValues[656] = d656
					ps1018.OverlayValues[657] = d657
					ps1018.OverlayValues[658] = d658
					ps1018.OverlayValues[659] = d659
					ps1018.OverlayValues[660] = d660
					ps1018.OverlayValues[661] = d661
					ps1018.OverlayValues[662] = d662
					ps1018.OverlayValues[664] = d664
					ps1018.OverlayValues[665] = d665
					ps1018.OverlayValues[666] = d666
					ps1018.OverlayValues[667] = d667
					ps1018.OverlayValues[668] = d668
					ps1018.OverlayValues[669] = d669
					ps1018.OverlayValues[670] = d670
					ps1018.OverlayValues[672] = d672
					ps1018.OverlayValues[674] = d674
					ps1018.OverlayValues[784] = d784
					ps1018.OverlayValues[787] = d787
					ps1018.OverlayValues[899] = d899
					ps1018.OverlayValues[900] = d900
					ps1018.OverlayValues[901] = d901
					ps1019 := PhiState{General: true}
					ps1019.OverlayValues = make([]JITValueDesc, 902)
					ps1019.OverlayValues[8] = d8
					ps1019.OverlayValues[9] = d9
					ps1019.OverlayValues[10] = d10
					ps1019.OverlayValues[11] = d11
					ps1019.OverlayValues[12] = d12
					ps1019.OverlayValues[13] = d13
					ps1019.OverlayValues[14] = d14
					ps1019.OverlayValues[15] = d15
					ps1019.OverlayValues[16] = d16
					ps1019.OverlayValues[17] = d17
					ps1019.OverlayValues[18] = d18
					ps1019.OverlayValues[19] = d19
					ps1019.OverlayValues[20] = d20
					ps1019.OverlayValues[21] = d21
					ps1019.OverlayValues[22] = d22
					ps1019.OverlayValues[25] = d25
					ps1019.OverlayValues[45] = d45
					ps1019.OverlayValues[64] = d64
					ps1019.OverlayValues[65] = d65
					ps1019.OverlayValues[66] = d66
					ps1019.OverlayValues[67] = d67
					ps1019.OverlayValues[68] = d68
					ps1019.OverlayValues[70] = d70
					ps1019.OverlayValues[71] = d71
					ps1019.OverlayValues[72] = d72
					ps1019.OverlayValues[73] = d73
					ps1019.OverlayValues[74] = d74
					ps1019.OverlayValues[75] = d75
					ps1019.OverlayValues[76] = d76
					ps1019.OverlayValues[79] = d79
					ps1019.OverlayValues[145] = d145
					ps1019.OverlayValues[146] = d146
					ps1019.OverlayValues[147] = d147
					ps1019.OverlayValues[148] = d148
					ps1019.OverlayValues[149] = d149
					ps1019.OverlayValues[150] = d150
					ps1019.OverlayValues[152] = d152
					ps1019.OverlayValues[153] = d153
					ps1019.OverlayValues[154] = d154
					ps1019.OverlayValues[155] = d155
					ps1019.OverlayValues[156] = d156
					ps1019.OverlayValues[157] = d157
					ps1019.OverlayValues[158] = d158
					ps1019.OverlayValues[159] = d159
					ps1019.OverlayValues[160] = d160
					ps1019.OverlayValues[163] = d163
					ps1019.OverlayValues[164] = d164
					ps1019.OverlayValues[165] = d165
					ps1019.OverlayValues[166] = d166
					ps1019.OverlayValues[269] = d269
					ps1019.OverlayValues[270] = d270
					ps1019.OverlayValues[271] = d271
					ps1019.OverlayValues[272] = d272
					ps1019.OverlayValues[273] = d273
					ps1019.OverlayValues[274] = d274
					ps1019.OverlayValues[275] = d275
					ps1019.OverlayValues[276] = d276
					ps1019.OverlayValues[277] = d277
					ps1019.OverlayValues[278] = d278
					ps1019.OverlayValues[279] = d279
					ps1019.OverlayValues[280] = d280
					ps1019.OverlayValues[281] = d281
					ps1019.OverlayValues[282] = d282
					ps1019.OverlayValues[283] = d283
					ps1019.OverlayValues[284] = d284
					ps1019.OverlayValues[285] = d285
					ps1019.OverlayValues[286] = d286
					ps1019.OverlayValues[287] = d287
					ps1019.OverlayValues[289] = d289
					ps1019.OverlayValues[290] = d290
					ps1019.OverlayValues[291] = d291
					ps1019.OverlayValues[292] = d292
					ps1019.OverlayValues[293] = d293
					ps1019.OverlayValues[294] = d294
					ps1019.OverlayValues[295] = d295
					ps1019.OverlayValues[296] = d296
					ps1019.OverlayValues[298] = d298
					ps1019.OverlayValues[299] = d299
					ps1019.OverlayValues[300] = d300
					ps1019.OverlayValues[465] = d465
					ps1019.OverlayValues[466] = d466
					ps1019.OverlayValues[467] = d467
					ps1019.OverlayValues[468] = d468
					ps1019.OverlayValues[469] = d469
					ps1019.OverlayValues[472] = d472
					ps1019.OverlayValues[473] = d473
					ps1019.OverlayValues[650] = d650
					ps1019.OverlayValues[651] = d651
					ps1019.OverlayValues[652] = d652
					ps1019.OverlayValues[653] = d653
					ps1019.OverlayValues[654] = d654
					ps1019.OverlayValues[655] = d655
					ps1019.OverlayValues[656] = d656
					ps1019.OverlayValues[657] = d657
					ps1019.OverlayValues[658] = d658
					ps1019.OverlayValues[659] = d659
					ps1019.OverlayValues[660] = d660
					ps1019.OverlayValues[661] = d661
					ps1019.OverlayValues[662] = d662
					ps1019.OverlayValues[664] = d664
					ps1019.OverlayValues[665] = d665
					ps1019.OverlayValues[666] = d666
					ps1019.OverlayValues[667] = d667
					ps1019.OverlayValues[668] = d668
					ps1019.OverlayValues[669] = d669
					ps1019.OverlayValues[670] = d670
					ps1019.OverlayValues[672] = d672
					ps1019.OverlayValues[674] = d674
					ps1019.OverlayValues[784] = d784
					ps1019.OverlayValues[787] = d787
					ps1019.OverlayValues[899] = d899
					ps1019.OverlayValues[900] = d900
					ps1019.OverlayValues[901] = d901
					snap1020 := d8
					snap1021 := d9
					snap1022 := d10
					snap1023 := d11
					snap1024 := d12
					snap1025 := d13
					snap1026 := d14
					snap1027 := d15
					snap1028 := d16
					snap1029 := d17
					snap1030 := d18
					snap1031 := d19
					snap1032 := d20
					snap1033 := d21
					snap1034 := d22
					snap1035 := d25
					snap1036 := d45
					snap1037 := d64
					snap1038 := d65
					snap1039 := d66
					snap1040 := d67
					snap1041 := d68
					snap1042 := d70
					snap1043 := d71
					snap1044 := d72
					snap1045 := d73
					snap1046 := d74
					snap1047 := d75
					snap1048 := d76
					snap1049 := d79
					snap1050 := d145
					snap1051 := d146
					snap1052 := d147
					snap1053 := d148
					snap1054 := d149
					snap1055 := d150
					snap1056 := d152
					snap1057 := d153
					snap1058 := d154
					snap1059 := d155
					snap1060 := d156
					snap1061 := d157
					snap1062 := d158
					snap1063 := d159
					snap1064 := d160
					snap1065 := d163
					snap1066 := d164
					snap1067 := d165
					snap1068 := d166
					snap1069 := d269
					snap1070 := d270
					snap1071 := d271
					snap1072 := d272
					snap1073 := d273
					snap1074 := d274
					snap1075 := d275
					snap1076 := d276
					snap1077 := d277
					snap1078 := d278
					snap1079 := d279
					snap1080 := d280
					snap1081 := d281
					snap1082 := d282
					snap1083 := d283
					snap1084 := d284
					snap1085 := d285
					snap1086 := d286
					snap1087 := d287
					snap1088 := d289
					snap1089 := d290
					snap1090 := d291
					snap1091 := d292
					snap1092 := d293
					snap1093 := d294
					snap1094 := d295
					snap1095 := d296
					snap1096 := d298
					snap1097 := d299
					snap1098 := d300
					snap1099 := d465
					snap1100 := d466
					snap1101 := d467
					snap1102 := d468
					snap1103 := d469
					snap1104 := d472
					snap1105 := d473
					snap1106 := d650
					snap1107 := d651
					snap1108 := d652
					snap1109 := d653
					snap1110 := d654
					snap1111 := d655
					snap1112 := d656
					snap1113 := d657
					snap1114 := d658
					snap1115 := d659
					snap1116 := d660
					snap1117 := d661
					snap1118 := d662
					snap1119 := d664
					snap1120 := d665
					snap1121 := d666
					snap1122 := d667
					snap1123 := d668
					snap1124 := d669
					snap1125 := d670
					snap1126 := d672
					snap1127 := d674
					snap1128 := d784
					snap1129 := d787
					snap1130 := d899
					snap1131 := d900
					snap1132 := d901
					alloc1133 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps1019)
					}
					ctx.RestoreAllocState(alloc1133)
					d8 = snap1020
					d9 = snap1021
					d10 = snap1022
					d11 = snap1023
					d12 = snap1024
					d13 = snap1025
					d14 = snap1026
					d15 = snap1027
					d16 = snap1028
					d17 = snap1029
					d18 = snap1030
					d19 = snap1031
					d20 = snap1032
					d21 = snap1033
					d22 = snap1034
					d25 = snap1035
					d45 = snap1036
					d64 = snap1037
					d65 = snap1038
					d66 = snap1039
					d67 = snap1040
					d68 = snap1041
					d70 = snap1042
					d71 = snap1043
					d72 = snap1044
					d73 = snap1045
					d74 = snap1046
					d75 = snap1047
					d76 = snap1048
					d79 = snap1049
					d145 = snap1050
					d146 = snap1051
					d147 = snap1052
					d148 = snap1053
					d149 = snap1054
					d150 = snap1055
					d152 = snap1056
					d153 = snap1057
					d154 = snap1058
					d155 = snap1059
					d156 = snap1060
					d157 = snap1061
					d158 = snap1062
					d159 = snap1063
					d160 = snap1064
					d163 = snap1065
					d164 = snap1066
					d165 = snap1067
					d166 = snap1068
					d269 = snap1069
					d270 = snap1070
					d271 = snap1071
					d272 = snap1072
					d273 = snap1073
					d274 = snap1074
					d275 = snap1075
					d276 = snap1076
					d277 = snap1077
					d278 = snap1078
					d279 = snap1079
					d280 = snap1080
					d281 = snap1081
					d282 = snap1082
					d283 = snap1083
					d284 = snap1084
					d285 = snap1085
					d286 = snap1086
					d287 = snap1087
					d289 = snap1088
					d290 = snap1089
					d291 = snap1090
					d292 = snap1091
					d293 = snap1092
					d294 = snap1093
					d295 = snap1094
					d296 = snap1095
					d298 = snap1096
					d299 = snap1097
					d300 = snap1098
					d465 = snap1099
					d466 = snap1100
					d467 = snap1101
					d468 = snap1102
					d469 = snap1103
					d472 = snap1104
					d473 = snap1105
					d650 = snap1106
					d651 = snap1107
					d652 = snap1108
					d653 = snap1109
					d654 = snap1110
					d655 = snap1111
					d656 = snap1112
					d657 = snap1113
					d658 = snap1114
					d659 = snap1115
					d660 = snap1116
					d661 = snap1117
					d662 = snap1118
					d664 = snap1119
					d665 = snap1120
					d666 = snap1121
					d667 = snap1122
					d668 = snap1123
					d669 = snap1124
					d670 = snap1125
					d672 = snap1126
					d674 = snap1127
					d784 = snap1128
					d787 = snap1129
					d899 = snap1130
					d900 = snap1131
					d901 = snap1132
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps1018)
					}
					return result
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
						d146 = ps.OverlayValues[146]
					}
					if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
						d147 = ps.OverlayValues[147]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
					}
					if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
						d154 = ps.OverlayValues[154]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 156 && ps.OverlayValues[156].Loc != LocNone {
						d156 = ps.OverlayValues[156]
					}
					if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
						d157 = ps.OverlayValues[157]
					}
					if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
						d158 = ps.OverlayValues[158]
					}
					if len(ps.OverlayValues) > 159 && ps.OverlayValues[159].Loc != LocNone {
						d159 = ps.OverlayValues[159]
					}
					if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != LocNone {
						d160 = ps.OverlayValues[160]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
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
					if len(ps.OverlayValues) > 269 && ps.OverlayValues[269].Loc != LocNone {
						d269 = ps.OverlayValues[269]
					}
					if len(ps.OverlayValues) > 270 && ps.OverlayValues[270].Loc != LocNone {
						d270 = ps.OverlayValues[270]
					}
					if len(ps.OverlayValues) > 271 && ps.OverlayValues[271].Loc != LocNone {
						d271 = ps.OverlayValues[271]
					}
					if len(ps.OverlayValues) > 272 && ps.OverlayValues[272].Loc != LocNone {
						d272 = ps.OverlayValues[272]
					}
					if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
						d273 = ps.OverlayValues[273]
					}
					if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
						d274 = ps.OverlayValues[274]
					}
					if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
						d275 = ps.OverlayValues[275]
					}
					if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
						d276 = ps.OverlayValues[276]
					}
					if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != LocNone {
						d277 = ps.OverlayValues[277]
					}
					if len(ps.OverlayValues) > 278 && ps.OverlayValues[278].Loc != LocNone {
						d278 = ps.OverlayValues[278]
					}
					if len(ps.OverlayValues) > 279 && ps.OverlayValues[279].Loc != LocNone {
						d279 = ps.OverlayValues[279]
					}
					if len(ps.OverlayValues) > 280 && ps.OverlayValues[280].Loc != LocNone {
						d280 = ps.OverlayValues[280]
					}
					if len(ps.OverlayValues) > 281 && ps.OverlayValues[281].Loc != LocNone {
						d281 = ps.OverlayValues[281]
					}
					if len(ps.OverlayValues) > 282 && ps.OverlayValues[282].Loc != LocNone {
						d282 = ps.OverlayValues[282]
					}
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 286 && ps.OverlayValues[286].Loc != LocNone {
						d286 = ps.OverlayValues[286]
					}
					if len(ps.OverlayValues) > 287 && ps.OverlayValues[287].Loc != LocNone {
						d287 = ps.OverlayValues[287]
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
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 295 && ps.OverlayValues[295].Loc != LocNone {
						d295 = ps.OverlayValues[295]
					}
					if len(ps.OverlayValues) > 296 && ps.OverlayValues[296].Loc != LocNone {
						d296 = ps.OverlayValues[296]
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
					if len(ps.OverlayValues) > 465 && ps.OverlayValues[465].Loc != LocNone {
						d465 = ps.OverlayValues[465]
					}
					if len(ps.OverlayValues) > 466 && ps.OverlayValues[466].Loc != LocNone {
						d466 = ps.OverlayValues[466]
					}
					if len(ps.OverlayValues) > 467 && ps.OverlayValues[467].Loc != LocNone {
						d467 = ps.OverlayValues[467]
					}
					if len(ps.OverlayValues) > 468 && ps.OverlayValues[468].Loc != LocNone {
						d468 = ps.OverlayValues[468]
					}
					if len(ps.OverlayValues) > 469 && ps.OverlayValues[469].Loc != LocNone {
						d469 = ps.OverlayValues[469]
					}
					if len(ps.OverlayValues) > 472 && ps.OverlayValues[472].Loc != LocNone {
						d472 = ps.OverlayValues[472]
					}
					if len(ps.OverlayValues) > 473 && ps.OverlayValues[473].Loc != LocNone {
						d473 = ps.OverlayValues[473]
					}
					if len(ps.OverlayValues) > 650 && ps.OverlayValues[650].Loc != LocNone {
						d650 = ps.OverlayValues[650]
					}
					if len(ps.OverlayValues) > 651 && ps.OverlayValues[651].Loc != LocNone {
						d651 = ps.OverlayValues[651]
					}
					if len(ps.OverlayValues) > 652 && ps.OverlayValues[652].Loc != LocNone {
						d652 = ps.OverlayValues[652]
					}
					if len(ps.OverlayValues) > 653 && ps.OverlayValues[653].Loc != LocNone {
						d653 = ps.OverlayValues[653]
					}
					if len(ps.OverlayValues) > 654 && ps.OverlayValues[654].Loc != LocNone {
						d654 = ps.OverlayValues[654]
					}
					if len(ps.OverlayValues) > 655 && ps.OverlayValues[655].Loc != LocNone {
						d655 = ps.OverlayValues[655]
					}
					if len(ps.OverlayValues) > 656 && ps.OverlayValues[656].Loc != LocNone {
						d656 = ps.OverlayValues[656]
					}
					if len(ps.OverlayValues) > 657 && ps.OverlayValues[657].Loc != LocNone {
						d657 = ps.OverlayValues[657]
					}
					if len(ps.OverlayValues) > 658 && ps.OverlayValues[658].Loc != LocNone {
						d658 = ps.OverlayValues[658]
					}
					if len(ps.OverlayValues) > 659 && ps.OverlayValues[659].Loc != LocNone {
						d659 = ps.OverlayValues[659]
					}
					if len(ps.OverlayValues) > 660 && ps.OverlayValues[660].Loc != LocNone {
						d660 = ps.OverlayValues[660]
					}
					if len(ps.OverlayValues) > 661 && ps.OverlayValues[661].Loc != LocNone {
						d661 = ps.OverlayValues[661]
					}
					if len(ps.OverlayValues) > 662 && ps.OverlayValues[662].Loc != LocNone {
						d662 = ps.OverlayValues[662]
					}
					if len(ps.OverlayValues) > 664 && ps.OverlayValues[664].Loc != LocNone {
						d664 = ps.OverlayValues[664]
					}
					if len(ps.OverlayValues) > 665 && ps.OverlayValues[665].Loc != LocNone {
						d665 = ps.OverlayValues[665]
					}
					if len(ps.OverlayValues) > 666 && ps.OverlayValues[666].Loc != LocNone {
						d666 = ps.OverlayValues[666]
					}
					if len(ps.OverlayValues) > 667 && ps.OverlayValues[667].Loc != LocNone {
						d667 = ps.OverlayValues[667]
					}
					if len(ps.OverlayValues) > 668 && ps.OverlayValues[668].Loc != LocNone {
						d668 = ps.OverlayValues[668]
					}
					if len(ps.OverlayValues) > 669 && ps.OverlayValues[669].Loc != LocNone {
						d669 = ps.OverlayValues[669]
					}
					if len(ps.OverlayValues) > 670 && ps.OverlayValues[670].Loc != LocNone {
						d670 = ps.OverlayValues[670]
					}
					if len(ps.OverlayValues) > 672 && ps.OverlayValues[672].Loc != LocNone {
						d672 = ps.OverlayValues[672]
					}
					if len(ps.OverlayValues) > 674 && ps.OverlayValues[674].Loc != LocNone {
						d674 = ps.OverlayValues[674]
					}
					if len(ps.OverlayValues) > 784 && ps.OverlayValues[784].Loc != LocNone {
						d784 = ps.OverlayValues[784]
					}
					if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != LocNone {
						d787 = ps.OverlayValues[787]
					}
					if len(ps.OverlayValues) > 899 && ps.OverlayValues[899].Loc != LocNone {
						d899 = ps.OverlayValues[899]
					}
					if len(ps.OverlayValues) > 900 && ps.OverlayValues[900].Loc != LocNone {
						d900 = ps.OverlayValues[900]
					}
					if len(ps.OverlayValues) > 901 && ps.OverlayValues[901].Loc != LocNone {
						d901 = ps.OverlayValues[901]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d14)
					var d1134 JITValueDesc
					if d14.Loc == LocImm {
						d1134 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d14.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d14)
						var d1135 JITValueDesc
						if d14.Loc == LocRegPair {
							ctx.FreeReg(d14.Reg)
							d1135 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg2}
							ctx.BindReg(d14.Reg2, &d1135)
							ctx.BindReg(d14.Reg2, &d1135)
						} else {
							d1135 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg}
							ctx.BindReg(d14.Reg, &d1135)
							ctx.BindReg(d14.Reg, &d1135)
						}
						d1134 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d1135}, 1)
						d1134.Type = tagFloat
						ctx.BindReg(d1134.Reg, &d1134)
					}
					ctx.StabilizeDescForControlFlow(&d1134)
					if ps.General {
						ctx.SyncDesc(&d1134)
						if d1134.Loc == LocReg {
							ctx.ProtectReg(d1134.Reg)
						} else if d1134.Loc == LocRegPair {
							ctx.ProtectReg(d1134.Reg)
							ctx.ProtectReg(d1134.Reg2)
						}
						d1136 = d1134
						if d1136.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d1136)
						ctx.EmitStoreToStack(d1136, int32(bbs[4].PhiBase)+int32(0))
						if d1134.Loc == LocReg {
							ctx.UnprotectReg(d1134.Reg)
						} else if d1134.Loc == LocRegPair {
							ctx.UnprotectReg(d1134.Reg)
							ctx.UnprotectReg(d1134.Reg2)
						}
					}
					ps1137 := PhiState{General: ps.General}
					ps1137.OverlayValues = make([]JITValueDesc, 1137)
					ps1137.OverlayValues[8] = d8
					ps1137.OverlayValues[9] = d9
					ps1137.OverlayValues[10] = d10
					ps1137.OverlayValues[11] = d11
					ps1137.OverlayValues[12] = d12
					ps1137.OverlayValues[13] = d13
					ps1137.OverlayValues[14] = d14
					ps1137.OverlayValues[15] = d15
					ps1137.OverlayValues[16] = d16
					ps1137.OverlayValues[17] = d17
					ps1137.OverlayValues[18] = d18
					ps1137.OverlayValues[19] = d19
					ps1137.OverlayValues[20] = d20
					ps1137.OverlayValues[21] = d21
					ps1137.OverlayValues[22] = d22
					ps1137.OverlayValues[25] = d25
					ps1137.OverlayValues[45] = d45
					ps1137.OverlayValues[64] = d64
					ps1137.OverlayValues[65] = d65
					ps1137.OverlayValues[66] = d66
					ps1137.OverlayValues[67] = d67
					ps1137.OverlayValues[68] = d68
					ps1137.OverlayValues[70] = d70
					ps1137.OverlayValues[71] = d71
					ps1137.OverlayValues[72] = d72
					ps1137.OverlayValues[73] = d73
					ps1137.OverlayValues[74] = d74
					ps1137.OverlayValues[75] = d75
					ps1137.OverlayValues[76] = d76
					ps1137.OverlayValues[79] = d79
					ps1137.OverlayValues[145] = d145
					ps1137.OverlayValues[146] = d146
					ps1137.OverlayValues[147] = d147
					ps1137.OverlayValues[148] = d148
					ps1137.OverlayValues[149] = d149
					ps1137.OverlayValues[150] = d150
					ps1137.OverlayValues[152] = d152
					ps1137.OverlayValues[153] = d153
					ps1137.OverlayValues[154] = d154
					ps1137.OverlayValues[155] = d155
					ps1137.OverlayValues[156] = d156
					ps1137.OverlayValues[157] = d157
					ps1137.OverlayValues[158] = d158
					ps1137.OverlayValues[159] = d159
					ps1137.OverlayValues[160] = d160
					ps1137.OverlayValues[163] = d163
					ps1137.OverlayValues[164] = d164
					ps1137.OverlayValues[165] = d165
					ps1137.OverlayValues[166] = d166
					ps1137.OverlayValues[269] = d269
					ps1137.OverlayValues[270] = d270
					ps1137.OverlayValues[271] = d271
					ps1137.OverlayValues[272] = d272
					ps1137.OverlayValues[273] = d273
					ps1137.OverlayValues[274] = d274
					ps1137.OverlayValues[275] = d275
					ps1137.OverlayValues[276] = d276
					ps1137.OverlayValues[277] = d277
					ps1137.OverlayValues[278] = d278
					ps1137.OverlayValues[279] = d279
					ps1137.OverlayValues[280] = d280
					ps1137.OverlayValues[281] = d281
					ps1137.OverlayValues[282] = d282
					ps1137.OverlayValues[283] = d283
					ps1137.OverlayValues[284] = d284
					ps1137.OverlayValues[285] = d285
					ps1137.OverlayValues[286] = d286
					ps1137.OverlayValues[287] = d287
					ps1137.OverlayValues[289] = d289
					ps1137.OverlayValues[290] = d290
					ps1137.OverlayValues[291] = d291
					ps1137.OverlayValues[292] = d292
					ps1137.OverlayValues[293] = d293
					ps1137.OverlayValues[294] = d294
					ps1137.OverlayValues[295] = d295
					ps1137.OverlayValues[296] = d296
					ps1137.OverlayValues[298] = d298
					ps1137.OverlayValues[299] = d299
					ps1137.OverlayValues[300] = d300
					ps1137.OverlayValues[465] = d465
					ps1137.OverlayValues[466] = d466
					ps1137.OverlayValues[467] = d467
					ps1137.OverlayValues[468] = d468
					ps1137.OverlayValues[469] = d469
					ps1137.OverlayValues[472] = d472
					ps1137.OverlayValues[473] = d473
					ps1137.OverlayValues[650] = d650
					ps1137.OverlayValues[651] = d651
					ps1137.OverlayValues[652] = d652
					ps1137.OverlayValues[653] = d653
					ps1137.OverlayValues[654] = d654
					ps1137.OverlayValues[655] = d655
					ps1137.OverlayValues[656] = d656
					ps1137.OverlayValues[657] = d657
					ps1137.OverlayValues[658] = d658
					ps1137.OverlayValues[659] = d659
					ps1137.OverlayValues[660] = d660
					ps1137.OverlayValues[661] = d661
					ps1137.OverlayValues[662] = d662
					ps1137.OverlayValues[664] = d664
					ps1137.OverlayValues[665] = d665
					ps1137.OverlayValues[666] = d666
					ps1137.OverlayValues[667] = d667
					ps1137.OverlayValues[668] = d668
					ps1137.OverlayValues[669] = d669
					ps1137.OverlayValues[670] = d670
					ps1137.OverlayValues[672] = d672
					ps1137.OverlayValues[674] = d674
					ps1137.OverlayValues[784] = d784
					ps1137.OverlayValues[787] = d787
					ps1137.OverlayValues[899] = d899
					ps1137.OverlayValues[900] = d900
					ps1137.OverlayValues[901] = d901
					ps1137.OverlayValues[1134] = d1134
					ps1137.OverlayValues[1135] = d1135
					ps1137.OverlayValues[1136] = d1136
					ps1137.PhiValues = make([]JITValueDesc, 1)
					d1138 = d1134
					ps1137.PhiValues[0] = d1138
					if ps1137.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps1137)
					return result
				}
				ps1139 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps1139)
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
