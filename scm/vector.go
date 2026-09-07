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
				var d38 JITValueDesc
				_ = d38
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
				var d63 JITValueDesc
				_ = d63
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
				var d69 JITValueDesc
				_ = d69
				var d72 JITValueDesc
				_ = d72
				var d138 JITValueDesc
				_ = d138
				var d139 JITValueDesc
				_ = d139
				var d140 JITValueDesc
				_ = d140
				var d141 JITValueDesc
				_ = d141
				var d142 JITValueDesc
				_ = d142
				var d143 JITValueDesc
				_ = d143
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
				var d151 JITValueDesc
				_ = d151
				var d152 JITValueDesc
				_ = d152
				var d153 JITValueDesc
				_ = d153
				var d156 JITValueDesc
				_ = d156
				var d157 JITValueDesc
				_ = d157
				var d158 JITValueDesc
				_ = d158
				var d159 JITValueDesc
				_ = d159
				var d262 JITValueDesc
				_ = d262
				var d263 JITValueDesc
				_ = d263
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
				var d278 JITValueDesc
				_ = d278
				var d279 JITValueDesc
				_ = d279
				var d280 JITValueDesc
				_ = d280
				var d281 JITValueDesc
				_ = d281
				var d283 JITValueDesc
				_ = d283
				var d284 JITValueDesc
				_ = d284
				var d285 JITValueDesc
				_ = d285
				var d434 JITValueDesc
				_ = d434
				var d435 JITValueDesc
				_ = d435
				var d436 JITValueDesc
				_ = d436
				var d437 JITValueDesc
				_ = d437
				var d438 JITValueDesc
				_ = d438
				var d441 JITValueDesc
				_ = d441
				var d442 JITValueDesc
				_ = d442
				var d603 JITValueDesc
				_ = d603
				var d604 JITValueDesc
				_ = d604
				var d605 JITValueDesc
				_ = d605
				var d606 JITValueDesc
				_ = d606
				var d607 JITValueDesc
				_ = d607
				var d608 JITValueDesc
				_ = d608
				var d609 JITValueDesc
				_ = d609
				var d610 JITValueDesc
				_ = d610
				var d611 JITValueDesc
				_ = d611
				var d612 JITValueDesc
				_ = d612
				var d613 JITValueDesc
				_ = d613
				var d615 JITValueDesc
				_ = d615
				var d616 JITValueDesc
				_ = d616
				var d617 JITValueDesc
				_ = d617
				var d618 JITValueDesc
				_ = d618
				var d619 JITValueDesc
				_ = d619
				var d621 JITValueDesc
				_ = d621
				var d623 JITValueDesc
				_ = d623
				var d721 JITValueDesc
				_ = d721
				var d724 JITValueDesc
				_ = d724
				var d824 JITValueDesc
				_ = d824
				var d825 JITValueDesc
				_ = d825
				var d826 JITValueDesc
				_ = d826
				var d1035 JITValueDesc
				_ = d1035
				var d1036 JITValueDesc
				_ = d1036
				var d1037 JITValueDesc
				_ = d1037
				var d1039 JITValueDesc
				_ = d1039
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
				d1 := JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
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
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedGreater}
						ctx.BindReg(r0, &d14)
					}
					ctx.FreeDesc(&d13)
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
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
					ctx.EmitJump(d15.Condition, lbl2)
					ctx.EmitJmp(lbl16)
					ctx.FreeDesc(&d14)
					snap19 := d1
					snap20 := d2
					snap21 := d3
					snap22 := d4
					snap23 := d5
					snap24 := d6
					snap25 := d7
					snap26 := d8
					snap27 := d9
					snap28 := d10
					snap29 := d11
					snap30 := d12
					snap31 := d13
					snap32 := d14
					snap33 := d15
					snap34 := d18
					alloc35 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc35)
					d1 = snap19
					d2 = snap20
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d6 = snap24
					d7 = snap25
					d8 = snap26
					d9 = snap27
					d10 = snap28
					d11 = snap29
					d12 = snap30
					d13 = snap31
					d14 = snap32
					d15 = snap33
					d18 = snap34
					ctx.MarkLabel(lbl16)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc35)
					d1 = snap19
					d2 = snap20
					d3 = snap21
					d4 = snap22
					d5 = snap23
					d6 = snap24
					d7 = snap25
					d8 = snap26
					d9 = snap27
					d10 = snap28
					d11 = snap29
					d12 = snap30
					d13 = snap31
					d14 = snap32
					d15 = snap33
					d18 = snap34
					ps36 := PhiState{General: true}
					ps36.OverlayValues = make([]JITValueDesc, 19)
					ps36.OverlayValues[1] = d1
					ps36.OverlayValues[2] = d2
					ps36.OverlayValues[3] = d3
					ps36.OverlayValues[4] = d4
					ps36.OverlayValues[5] = d5
					ps36.OverlayValues[6] = d6
					ps36.OverlayValues[7] = d7
					ps36.OverlayValues[8] = d8
					ps36.OverlayValues[9] = d9
					ps36.OverlayValues[10] = d10
					ps36.OverlayValues[11] = d11
					ps36.OverlayValues[12] = d12
					ps36.OverlayValues[13] = d13
					ps36.OverlayValues[14] = d14
					ps36.OverlayValues[15] = d15
					ps36.OverlayValues[18] = d18
					ps37 := PhiState{General: true}
					ps37.OverlayValues = make([]JITValueDesc, 19)
					ps37.OverlayValues[1] = d1
					ps37.OverlayValues[2] = d2
					ps37.OverlayValues[3] = d3
					ps37.OverlayValues[4] = d4
					ps37.OverlayValues[5] = d5
					ps37.OverlayValues[6] = d6
					ps37.OverlayValues[7] = d7
					ps37.OverlayValues[8] = d8
					ps37.OverlayValues[9] = d9
					ps37.OverlayValues[10] = d10
					ps37.OverlayValues[11] = d11
					ps37.OverlayValues[12] = d12
					ps37.OverlayValues[13] = d13
					ps37.OverlayValues[14] = d14
					ps37.OverlayValues[15] = d15
					ps37.OverlayValues[18] = d18
					ps37.PhiValues = make([]JITValueDesc, 1)
					d38 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}
					ps37.PhiValues[0] = d38
					snap39 := d1
					snap40 := d2
					snap41 := d3
					snap42 := d4
					snap43 := d5
					snap44 := d6
					snap45 := d7
					snap46 := d8
					snap47 := d9
					snap48 := d10
					snap49 := d11
					snap50 := d12
					snap51 := d13
					snap52 := d14
					snap53 := d15
					snap54 := d18
					snap55 := d38
					alloc56 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps37)
					}
					ctx.RestoreAllocState(alloc56)
					d1 = snap39
					d2 = snap40
					d3 = snap41
					d4 = snap42
					d5 = snap43
					d6 = snap44
					d7 = snap45
					d8 = snap46
					d9 = snap47
					d10 = snap48
					d11 = snap49
					d12 = snap50
					d13 = snap51
					d14 = snap52
					d15 = snap53
					d18 = snap54
					d38 = snap55
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps36)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					ctx.ReclaimUntrackedRegs()
					d57 = args[2]
					d57.ID = 0
					d59 = d57
					ctx.SyncDesc(&d59)
					if d59.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d59.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d59.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d59 = tmpScalar
					}
					d59 = JITPrepareScmerGoArg(ctx, d59)
					if d59.Loc != LocRegPair && d59.Loc != LocStackPair && d59.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d58 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d59}, 2)
					ctx.FreeDesc(&d57)
					ctx.EnsureDesc(&d58)
					if d58.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d58.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d58.Imm)
						ptrWord, _ := d58.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d58.Imm.String())))
						d58 = tmpPair
					} else if d58.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d58.Type, Reg: ctx.AllocRegExcept(d58.Reg), Reg2: ctx.AllocRegExcept(d58.Reg)}
						switch d58.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d58)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d58)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d58)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d58)
						d58 = tmpPair
					}
					if d58.Loc != LocRegPair && d58.Loc != LocStackPair && d58.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d58)
					d60 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d58}, 2)
					d60.NoHeapPointer = false
					ctx.BindReg(d60.Reg, &d60)
					ctx.BindReg(d60.Reg2, &d60)
					ctx.StabilizeDescForControlFlow(&d60)
					if ps.General {
						ctx.SyncDesc(&d60)
						if d60.Loc == LocReg {
							ctx.ProtectReg(d60.Reg)
						} else if d60.Loc == LocRegPair {
							ctx.ProtectReg(d60.Reg)
							ctx.ProtectReg(d60.Reg2)
						}
						d61 = d60
						if d61.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d61)
						if d61.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d61, int32(bbs[2].PhiBase)+int32(0), 2)
						} else if d61.Loc == LocInputPair {
							ctx.EnsureDesc(&d61)
							ctx.EmitStoreScmerToStack(d61, int32(bbs[2].PhiBase)+int32(0))
						} else if d61.Loc == LocRegPair || d61.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d61, int32(bbs[2].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d61)
							ctx.EmitStoreToStack(d61, int32(bbs[2].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
						}
						if d60.Loc == LocReg {
							ctx.UnprotectReg(d60.Reg)
						} else if d60.Loc == LocRegPair {
							ctx.UnprotectReg(d60.Reg)
							ctx.UnprotectReg(d60.Reg2)
						}
					}
					ps62 := PhiState{General: ps.General}
					ps62.OverlayValues = make([]JITValueDesc, 62)
					ps62.OverlayValues[1] = d1
					ps62.OverlayValues[2] = d2
					ps62.OverlayValues[3] = d3
					ps62.OverlayValues[4] = d4
					ps62.OverlayValues[5] = d5
					ps62.OverlayValues[6] = d6
					ps62.OverlayValues[7] = d7
					ps62.OverlayValues[8] = d8
					ps62.OverlayValues[9] = d9
					ps62.OverlayValues[10] = d10
					ps62.OverlayValues[11] = d11
					ps62.OverlayValues[12] = d12
					ps62.OverlayValues[13] = d13
					ps62.OverlayValues[14] = d14
					ps62.OverlayValues[15] = d15
					ps62.OverlayValues[18] = d18
					ps62.OverlayValues[38] = d38
					ps62.OverlayValues[57] = d57
					ps62.OverlayValues[58] = d58
					ps62.OverlayValues[59] = d59
					ps62.OverlayValues[60] = d60
					ps62.OverlayValues[61] = d61
					ps62.PhiValues = make([]JITValueDesc, 1)
					d63 = d60
					ps62.PhiValues[0] = d63
					if ps62.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps62)
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d64 := ps.PhiValues[0]
							ctx.EnsureDesc(&d64)
							ctx.EmitStoreScmerToStack(d64, int32(bbs[2].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					var d65 JITValueDesc
					if d1.Loc == LocImm {
						ctx.TrackImm(d1.Imm)
						ptrWord, _ := d1.Imm.RawWords()
						d65 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d65.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d65.Reg2, uint64(len(d1.Imm.String())))
						ctx.BindReg(d65.Reg, &d65)
						ctx.BindReg(d65.Reg2, &d65)
					} else {
						d65 = d1
					}
					d66 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("COSINE")}
					var d67 JITValueDesc
					if d66.Loc == LocImm {
						ctx.TrackImm(d66.Imm)
						ptrWord, _ := d66.Imm.RawWords()
						d67 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d67.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d67.Reg2, uint64(len(d66.Imm.String())))
						ctx.BindReg(d67.Reg, &d67)
						ctx.BindReg(d67.Reg2, &d67)
					} else {
						d67 = d66
					}
					d68 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d65, d67}, 1)
					ctx.EmitAndRegImm32(d68.Reg, 1)
					d68.Type = tagBool
					ctx.BindReg(d68.Reg, &d68)
					d69 = d68
					ctx.EnsureDesc(&d69)
					if d69.Loc != LocImm && d69.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d69.Loc == LocImm {
						if d69.Imm.Bool() {
							if ps.General {
							}
							ps70 := PhiState{General: ps.General}
							ps70.OverlayValues = make([]JITValueDesc, 70)
							ps70.OverlayValues[1] = d1
							ps70.OverlayValues[2] = d2
							ps70.OverlayValues[3] = d3
							ps70.OverlayValues[4] = d4
							ps70.OverlayValues[5] = d5
							ps70.OverlayValues[6] = d6
							ps70.OverlayValues[7] = d7
							ps70.OverlayValues[8] = d8
							ps70.OverlayValues[9] = d9
							ps70.OverlayValues[10] = d10
							ps70.OverlayValues[11] = d11
							ps70.OverlayValues[12] = d12
							ps70.OverlayValues[13] = d13
							ps70.OverlayValues[14] = d14
							ps70.OverlayValues[15] = d15
							ps70.OverlayValues[18] = d18
							ps70.OverlayValues[38] = d38
							ps70.OverlayValues[57] = d57
							ps70.OverlayValues[58] = d58
							ps70.OverlayValues[59] = d59
							ps70.OverlayValues[60] = d60
							ps70.OverlayValues[61] = d61
							ps70.OverlayValues[63] = d63
							ps70.OverlayValues[64] = d64
							ps70.OverlayValues[65] = d65
							ps70.OverlayValues[66] = d66
							ps70.OverlayValues[67] = d67
							ps70.OverlayValues[68] = d68
							ps70.OverlayValues[69] = d69
							return bbs[3].RenderPS(ps70)
						}
						if ps.General {
						}
						ps71 := PhiState{General: ps.General}
						ps71.OverlayValues = make([]JITValueDesc, 70)
						ps71.OverlayValues[1] = d1
						ps71.OverlayValues[2] = d2
						ps71.OverlayValues[3] = d3
						ps71.OverlayValues[4] = d4
						ps71.OverlayValues[5] = d5
						ps71.OverlayValues[6] = d6
						ps71.OverlayValues[7] = d7
						ps71.OverlayValues[8] = d8
						ps71.OverlayValues[9] = d9
						ps71.OverlayValues[10] = d10
						ps71.OverlayValues[11] = d11
						ps71.OverlayValues[12] = d12
						ps71.OverlayValues[13] = d13
						ps71.OverlayValues[14] = d14
						ps71.OverlayValues[15] = d15
						ps71.OverlayValues[18] = d18
						ps71.OverlayValues[38] = d38
						ps71.OverlayValues[57] = d57
						ps71.OverlayValues[58] = d58
						ps71.OverlayValues[59] = d59
						ps71.OverlayValues[60] = d60
						ps71.OverlayValues[61] = d61
						ps71.OverlayValues[63] = d63
						ps71.OverlayValues[64] = d64
						ps71.OverlayValues[65] = d65
						ps71.OverlayValues[66] = d66
						ps71.OverlayValues[67] = d67
						ps71.OverlayValues[68] = d68
						ps71.OverlayValues[69] = d69
						return bbs[5].RenderPS(ps71)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d72 := ps.PhiValues[0]
							ctx.EnsureDesc(&d72)
							ctx.EmitStoreScmerToStack(d72, int32(bbs[2].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					ctx.EmitCmpRegImm32(d69.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					snap73 := d1
					snap74 := d2
					snap75 := d3
					snap76 := d4
					snap77 := d5
					snap78 := d6
					snap79 := d7
					snap80 := d8
					snap81 := d9
					snap82 := d10
					snap83 := d11
					snap84 := d12
					snap85 := d13
					snap86 := d14
					snap87 := d15
					snap88 := d18
					snap89 := d38
					snap90 := d57
					snap91 := d58
					snap92 := d59
					snap93 := d60
					snap94 := d61
					snap95 := d63
					snap96 := d64
					snap97 := d65
					snap98 := d66
					snap99 := d67
					snap100 := d68
					snap101 := d69
					snap102 := d72
					alloc103 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc103)
					d1 = snap73
					d2 = snap74
					d3 = snap75
					d4 = snap76
					d5 = snap77
					d6 = snap78
					d7 = snap79
					d8 = snap80
					d9 = snap81
					d10 = snap82
					d11 = snap83
					d12 = snap84
					d13 = snap85
					d14 = snap86
					d15 = snap87
					d18 = snap88
					d38 = snap89
					d57 = snap90
					d58 = snap91
					d59 = snap92
					d60 = snap93
					d61 = snap94
					d63 = snap95
					d64 = snap96
					d65 = snap97
					d66 = snap98
					d67 = snap99
					d68 = snap100
					d69 = snap101
					d72 = snap102
					ctx.RestoreAllocState(alloc103)
					d1 = snap73
					d2 = snap74
					d3 = snap75
					d4 = snap76
					d5 = snap77
					d6 = snap78
					d7 = snap79
					d8 = snap80
					d9 = snap81
					d10 = snap82
					d11 = snap83
					d12 = snap84
					d13 = snap85
					d14 = snap86
					d15 = snap87
					d18 = snap88
					d38 = snap89
					d57 = snap90
					d58 = snap91
					d59 = snap92
					d60 = snap93
					d61 = snap94
					d63 = snap95
					d64 = snap96
					d65 = snap97
					d66 = snap98
					d67 = snap99
					d68 = snap100
					d69 = snap101
					d72 = snap102
					ps104 := PhiState{General: true}
					ps104.OverlayValues = make([]JITValueDesc, 73)
					ps104.OverlayValues[1] = d1
					ps104.OverlayValues[2] = d2
					ps104.OverlayValues[3] = d3
					ps104.OverlayValues[4] = d4
					ps104.OverlayValues[5] = d5
					ps104.OverlayValues[6] = d6
					ps104.OverlayValues[7] = d7
					ps104.OverlayValues[8] = d8
					ps104.OverlayValues[9] = d9
					ps104.OverlayValues[10] = d10
					ps104.OverlayValues[11] = d11
					ps104.OverlayValues[12] = d12
					ps104.OverlayValues[13] = d13
					ps104.OverlayValues[14] = d14
					ps104.OverlayValues[15] = d15
					ps104.OverlayValues[18] = d18
					ps104.OverlayValues[38] = d38
					ps104.OverlayValues[57] = d57
					ps104.OverlayValues[58] = d58
					ps104.OverlayValues[59] = d59
					ps104.OverlayValues[60] = d60
					ps104.OverlayValues[61] = d61
					ps104.OverlayValues[63] = d63
					ps104.OverlayValues[64] = d64
					ps104.OverlayValues[65] = d65
					ps104.OverlayValues[66] = d66
					ps104.OverlayValues[67] = d67
					ps104.OverlayValues[68] = d68
					ps104.OverlayValues[69] = d69
					ps104.OverlayValues[72] = d72
					ps105 := PhiState{General: true}
					ps105.OverlayValues = make([]JITValueDesc, 73)
					ps105.OverlayValues[1] = d1
					ps105.OverlayValues[2] = d2
					ps105.OverlayValues[3] = d3
					ps105.OverlayValues[4] = d4
					ps105.OverlayValues[5] = d5
					ps105.OverlayValues[6] = d6
					ps105.OverlayValues[7] = d7
					ps105.OverlayValues[8] = d8
					ps105.OverlayValues[9] = d9
					ps105.OverlayValues[10] = d10
					ps105.OverlayValues[11] = d11
					ps105.OverlayValues[12] = d12
					ps105.OverlayValues[13] = d13
					ps105.OverlayValues[14] = d14
					ps105.OverlayValues[15] = d15
					ps105.OverlayValues[18] = d18
					ps105.OverlayValues[38] = d38
					ps105.OverlayValues[57] = d57
					ps105.OverlayValues[58] = d58
					ps105.OverlayValues[59] = d59
					ps105.OverlayValues[60] = d60
					ps105.OverlayValues[61] = d61
					ps105.OverlayValues[63] = d63
					ps105.OverlayValues[64] = d64
					ps105.OverlayValues[65] = d65
					ps105.OverlayValues[66] = d66
					ps105.OverlayValues[67] = d67
					ps105.OverlayValues[68] = d68
					ps105.OverlayValues[69] = d69
					ps105.OverlayValues[72] = d72
					snap106 := d1
					snap107 := d2
					snap108 := d3
					snap109 := d4
					snap110 := d5
					snap111 := d6
					snap112 := d7
					snap113 := d8
					snap114 := d9
					snap115 := d10
					snap116 := d11
					snap117 := d12
					snap118 := d13
					snap119 := d14
					snap120 := d15
					snap121 := d18
					snap122 := d38
					snap123 := d57
					snap124 := d58
					snap125 := d59
					snap126 := d60
					snap127 := d61
					snap128 := d63
					snap129 := d64
					snap130 := d65
					snap131 := d66
					snap132 := d67
					snap133 := d68
					snap134 := d69
					snap135 := d72
					alloc136 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps105)
					}
					ctx.RestoreAllocState(alloc136)
					d1 = snap106
					d2 = snap107
					d3 = snap108
					d4 = snap109
					d5 = snap110
					d6 = snap111
					d7 = snap112
					d8 = snap113
					d9 = snap114
					d10 = snap115
					d11 = snap116
					d12 = snap117
					d13 = snap118
					d14 = snap119
					d15 = snap120
					d18 = snap121
					d38 = snap122
					d57 = snap123
					d58 = snap124
					d59 = snap125
					d60 = snap126
					d61 = snap127
					d63 = snap128
					d64 = snap129
					d65 = snap130
					d66 = snap131
					d67 = snap132
					d68 = snap133
					d69 = snap134
					d72 = snap135
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps104)
					}
					return result
					ctx.FreeDesc(&d68)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}, int32(bbs[6].PhiBase)+int32(16))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}, int32(bbs[6].PhiBase)+int32(32))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[6].PhiBase)+int32(48))
					}
					ps137 := PhiState{General: ps.General}
					ps137.OverlayValues = make([]JITValueDesc, 73)
					ps137.OverlayValues[1] = d1
					ps137.OverlayValues[2] = d2
					ps137.OverlayValues[3] = d3
					ps137.OverlayValues[4] = d4
					ps137.OverlayValues[5] = d5
					ps137.OverlayValues[6] = d6
					ps137.OverlayValues[7] = d7
					ps137.OverlayValues[8] = d8
					ps137.OverlayValues[9] = d9
					ps137.OverlayValues[10] = d10
					ps137.OverlayValues[11] = d11
					ps137.OverlayValues[12] = d12
					ps137.OverlayValues[13] = d13
					ps137.OverlayValues[14] = d14
					ps137.OverlayValues[15] = d15
					ps137.OverlayValues[18] = d18
					ps137.OverlayValues[38] = d38
					ps137.OverlayValues[57] = d57
					ps137.OverlayValues[58] = d58
					ps137.OverlayValues[59] = d59
					ps137.OverlayValues[60] = d60
					ps137.OverlayValues[61] = d61
					ps137.OverlayValues[63] = d63
					ps137.OverlayValues[64] = d64
					ps137.OverlayValues[65] = d65
					ps137.OverlayValues[66] = d66
					ps137.OverlayValues[67] = d67
					ps137.OverlayValues[68] = d68
					ps137.OverlayValues[69] = d69
					ps137.OverlayValues[72] = d72
					ps137.PhiValues = make([]JITValueDesc, 4)
					d138 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps137.PhiValues[0] = d138
					d139 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps137.PhiValues[1] = d139
					d140 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(0)}
					ps137.PhiValues[2] = d140
					d141 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps137.PhiValues[3] = d141
					if ps137.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps137)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d142 := ps.PhiValues[0]
							ctx.EnsureDesc(&d142)
							ctx.EmitStoreToStack(d142, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
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
						d143 := JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d143)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[10].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[10].PhiBase)+int32(16))
					}
					ps144 := PhiState{General: ps.General}
					ps144.OverlayValues = make([]JITValueDesc, 144)
					ps144.OverlayValues[1] = d1
					ps144.OverlayValues[2] = d2
					ps144.OverlayValues[3] = d3
					ps144.OverlayValues[4] = d4
					ps144.OverlayValues[5] = d5
					ps144.OverlayValues[6] = d6
					ps144.OverlayValues[7] = d7
					ps144.OverlayValues[8] = d8
					ps144.OverlayValues[9] = d9
					ps144.OverlayValues[10] = d10
					ps144.OverlayValues[11] = d11
					ps144.OverlayValues[12] = d12
					ps144.OverlayValues[13] = d13
					ps144.OverlayValues[14] = d14
					ps144.OverlayValues[15] = d15
					ps144.OverlayValues[18] = d18
					ps144.OverlayValues[38] = d38
					ps144.OverlayValues[57] = d57
					ps144.OverlayValues[58] = d58
					ps144.OverlayValues[59] = d59
					ps144.OverlayValues[60] = d60
					ps144.OverlayValues[61] = d61
					ps144.OverlayValues[63] = d63
					ps144.OverlayValues[64] = d64
					ps144.OverlayValues[65] = d65
					ps144.OverlayValues[66] = d66
					ps144.OverlayValues[67] = d67
					ps144.OverlayValues[68] = d68
					ps144.OverlayValues[69] = d69
					ps144.OverlayValues[72] = d72
					ps144.OverlayValues[138] = d138
					ps144.OverlayValues[139] = d139
					ps144.OverlayValues[140] = d140
					ps144.OverlayValues[141] = d141
					ps144.OverlayValues[142] = d142
					ps144.OverlayValues[143] = d143
					ps144.PhiValues = make([]JITValueDesc, 2)
					d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps144.PhiValues[0] = d145
					d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps144.PhiValues[1] = d146
					if ps144.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps144)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d147 := ps.PhiValues[0]
							ctx.EnsureDesc(&d147)
							ctx.EmitStoreToStack(d147, int32(bbs[6].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d148 := ps.PhiValues[1]
							ctx.EnsureDesc(&d148)
							ctx.EmitStoreToStack(d148, int32(bbs[6].PhiBase)+int32(16))
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d149 := ps.PhiValues[2]
							ctx.EnsureDesc(&d149)
							ctx.EmitStoreToStack(d149, int32(bbs[6].PhiBase)+int32(32))
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d150 := ps.PhiValues[3]
							ctx.EnsureDesc(&d150)
							ctx.EmitStoreToStack(d150, int32(bbs[6].PhiBase)+int32(48))
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					var d151 JITValueDesc
					if d10.SliceSizeKnown {
						d151 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.KnownSliceLen))}
					} else if d10.Loc == LocImm {
						d151 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.StackOff))}
					} else if d10.Loc == LocStackTriple {
						d151 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d10.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d10)
						if d10.Loc == LocRegPair || d10.Loc == LocRegTriple {
							d151 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg2, ID: 0}
						} else if d10.Loc == LocReg {
							d151 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d151)
					ctx.EnsureDescsTogether(&d6, &d151)
					var d152 JITValueDesc
					if d6.Loc == LocImm && d151.Loc == LocImm {
						d152 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() < d151.Imm.Int())}
					} else if d151.Loc == LocImm {
						r1 := ctx.AllocRegExcept(d6.Reg)
						if d151.Imm.Int() >= -2147483648 && d151.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d6.Reg, int32(d151.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d151.Imm.Int()))
							ctx.EmitCmpInt64(d6.Reg, RegR11)
						}
						d152 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d152)
					} else if d6.Loc == LocImm {
						r2 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d151.Reg)
						d152 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d152)
					} else {
						r3 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpInt64(d6.Reg, d151.Reg)
						d152 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d152)
					}
					ctx.FreeDesc(&d151)
					d153 = d152
					ctx.EnsureDesc(&d153)
					if d153.Loc != LocImm && d153.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d153.Loc == LocImm {
						if d153.Imm.Bool() {
							if ps.General {
							}
							ps154 := PhiState{General: ps.General}
							ps154.OverlayValues = make([]JITValueDesc, 154)
							ps154.OverlayValues[1] = d1
							ps154.OverlayValues[2] = d2
							ps154.OverlayValues[3] = d3
							ps154.OverlayValues[4] = d4
							ps154.OverlayValues[5] = d5
							ps154.OverlayValues[6] = d6
							ps154.OverlayValues[7] = d7
							ps154.OverlayValues[8] = d8
							ps154.OverlayValues[9] = d9
							ps154.OverlayValues[10] = d10
							ps154.OverlayValues[11] = d11
							ps154.OverlayValues[12] = d12
							ps154.OverlayValues[13] = d13
							ps154.OverlayValues[14] = d14
							ps154.OverlayValues[15] = d15
							ps154.OverlayValues[18] = d18
							ps154.OverlayValues[38] = d38
							ps154.OverlayValues[57] = d57
							ps154.OverlayValues[58] = d58
							ps154.OverlayValues[59] = d59
							ps154.OverlayValues[60] = d60
							ps154.OverlayValues[61] = d61
							ps154.OverlayValues[63] = d63
							ps154.OverlayValues[64] = d64
							ps154.OverlayValues[65] = d65
							ps154.OverlayValues[66] = d66
							ps154.OverlayValues[67] = d67
							ps154.OverlayValues[68] = d68
							ps154.OverlayValues[69] = d69
							ps154.OverlayValues[72] = d72
							ps154.OverlayValues[138] = d138
							ps154.OverlayValues[139] = d139
							ps154.OverlayValues[140] = d140
							ps154.OverlayValues[141] = d141
							ps154.OverlayValues[142] = d142
							ps154.OverlayValues[143] = d143
							ps154.OverlayValues[145] = d145
							ps154.OverlayValues[146] = d146
							ps154.OverlayValues[147] = d147
							ps154.OverlayValues[148] = d148
							ps154.OverlayValues[149] = d149
							ps154.OverlayValues[150] = d150
							ps154.OverlayValues[151] = d151
							ps154.OverlayValues[152] = d152
							ps154.OverlayValues[153] = d153
							return bbs[9].RenderPS(ps154)
						}
						if ps.General {
						}
						ps155 := PhiState{General: ps.General}
						ps155.OverlayValues = make([]JITValueDesc, 154)
						ps155.OverlayValues[1] = d1
						ps155.OverlayValues[2] = d2
						ps155.OverlayValues[3] = d3
						ps155.OverlayValues[4] = d4
						ps155.OverlayValues[5] = d5
						ps155.OverlayValues[6] = d6
						ps155.OverlayValues[7] = d7
						ps155.OverlayValues[8] = d8
						ps155.OverlayValues[9] = d9
						ps155.OverlayValues[10] = d10
						ps155.OverlayValues[11] = d11
						ps155.OverlayValues[12] = d12
						ps155.OverlayValues[13] = d13
						ps155.OverlayValues[14] = d14
						ps155.OverlayValues[15] = d15
						ps155.OverlayValues[18] = d18
						ps155.OverlayValues[38] = d38
						ps155.OverlayValues[57] = d57
						ps155.OverlayValues[58] = d58
						ps155.OverlayValues[59] = d59
						ps155.OverlayValues[60] = d60
						ps155.OverlayValues[61] = d61
						ps155.OverlayValues[63] = d63
						ps155.OverlayValues[64] = d64
						ps155.OverlayValues[65] = d65
						ps155.OverlayValues[66] = d66
						ps155.OverlayValues[67] = d67
						ps155.OverlayValues[68] = d68
						ps155.OverlayValues[69] = d69
						ps155.OverlayValues[72] = d72
						ps155.OverlayValues[138] = d138
						ps155.OverlayValues[139] = d139
						ps155.OverlayValues[140] = d140
						ps155.OverlayValues[141] = d141
						ps155.OverlayValues[142] = d142
						ps155.OverlayValues[143] = d143
						ps155.OverlayValues[145] = d145
						ps155.OverlayValues[146] = d146
						ps155.OverlayValues[147] = d147
						ps155.OverlayValues[148] = d148
						ps155.OverlayValues[149] = d149
						ps155.OverlayValues[150] = d150
						ps155.OverlayValues[151] = d151
						ps155.OverlayValues[152] = d152
						ps155.OverlayValues[153] = d153
						return bbs[8].RenderPS(ps155)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d156 := ps.PhiValues[0]
							ctx.EnsureDesc(&d156)
							ctx.EmitStoreToStack(d156, int32(bbs[6].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d157 := ps.PhiValues[1]
							ctx.EnsureDesc(&d157)
							ctx.EmitStoreToStack(d157, int32(bbs[6].PhiBase)+int32(16))
						}
						if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != LocNone {
							d158 := ps.PhiValues[2]
							ctx.EnsureDesc(&d158)
							ctx.EmitStoreToStack(d158, int32(bbs[6].PhiBase)+int32(32))
						}
						if len(ps.PhiValues) > 3 && ps.PhiValues[3].Loc != LocNone {
							d159 := ps.PhiValues[3]
							ctx.EnsureDesc(&d159)
							ctx.EmitStoreToStack(d159, int32(bbs[6].PhiBase)+int32(48))
						}
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					ctx.EmitJump(d153.Condition, lbl10)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d152)
					snap160 := d1
					snap161 := d2
					snap162 := d3
					snap163 := d4
					snap164 := d5
					snap165 := d6
					snap166 := d7
					snap167 := d8
					snap168 := d9
					snap169 := d10
					snap170 := d11
					snap171 := d12
					snap172 := d13
					snap173 := d14
					snap174 := d15
					snap175 := d18
					snap176 := d38
					snap177 := d57
					snap178 := d58
					snap179 := d59
					snap180 := d60
					snap181 := d61
					snap182 := d63
					snap183 := d64
					snap184 := d65
					snap185 := d66
					snap186 := d67
					snap187 := d68
					snap188 := d69
					snap189 := d72
					snap190 := d138
					snap191 := d139
					snap192 := d140
					snap193 := d141
					snap194 := d142
					snap195 := d143
					snap196 := d145
					snap197 := d146
					snap198 := d147
					snap199 := d148
					snap200 := d149
					snap201 := d150
					snap202 := d151
					snap203 := d152
					snap204 := d153
					snap205 := d156
					snap206 := d157
					snap207 := d158
					snap208 := d159
					alloc209 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc209)
					d1 = snap160
					d2 = snap161
					d3 = snap162
					d4 = snap163
					d5 = snap164
					d6 = snap165
					d7 = snap166
					d8 = snap167
					d9 = snap168
					d10 = snap169
					d11 = snap170
					d12 = snap171
					d13 = snap172
					d14 = snap173
					d15 = snap174
					d18 = snap175
					d38 = snap176
					d57 = snap177
					d58 = snap178
					d59 = snap179
					d60 = snap180
					d61 = snap181
					d63 = snap182
					d64 = snap183
					d65 = snap184
					d66 = snap185
					d67 = snap186
					d68 = snap187
					d69 = snap188
					d72 = snap189
					d138 = snap190
					d139 = snap191
					d140 = snap192
					d141 = snap193
					d142 = snap194
					d143 = snap195
					d145 = snap196
					d146 = snap197
					d147 = snap198
					d148 = snap199
					d149 = snap200
					d150 = snap201
					d151 = snap202
					d152 = snap203
					d153 = snap204
					d156 = snap205
					d157 = snap206
					d158 = snap207
					d159 = snap208
					ctx.RestoreAllocState(alloc209)
					d1 = snap160
					d2 = snap161
					d3 = snap162
					d4 = snap163
					d5 = snap164
					d6 = snap165
					d7 = snap166
					d8 = snap167
					d9 = snap168
					d10 = snap169
					d11 = snap170
					d12 = snap171
					d13 = snap172
					d14 = snap173
					d15 = snap174
					d18 = snap175
					d38 = snap176
					d57 = snap177
					d58 = snap178
					d59 = snap179
					d60 = snap180
					d61 = snap181
					d63 = snap182
					d64 = snap183
					d65 = snap184
					d66 = snap185
					d67 = snap186
					d68 = snap187
					d69 = snap188
					d72 = snap189
					d138 = snap190
					d139 = snap191
					d140 = snap192
					d141 = snap193
					d142 = snap194
					d143 = snap195
					d145 = snap196
					d146 = snap197
					d147 = snap198
					d148 = snap199
					d149 = snap200
					d150 = snap201
					d151 = snap202
					d152 = snap203
					d153 = snap204
					d156 = snap205
					d157 = snap206
					d158 = snap207
					d159 = snap208
					ps210 := PhiState{General: true}
					ps210.OverlayValues = make([]JITValueDesc, 160)
					ps210.OverlayValues[1] = d1
					ps210.OverlayValues[2] = d2
					ps210.OverlayValues[3] = d3
					ps210.OverlayValues[4] = d4
					ps210.OverlayValues[5] = d5
					ps210.OverlayValues[6] = d6
					ps210.OverlayValues[7] = d7
					ps210.OverlayValues[8] = d8
					ps210.OverlayValues[9] = d9
					ps210.OverlayValues[10] = d10
					ps210.OverlayValues[11] = d11
					ps210.OverlayValues[12] = d12
					ps210.OverlayValues[13] = d13
					ps210.OverlayValues[14] = d14
					ps210.OverlayValues[15] = d15
					ps210.OverlayValues[18] = d18
					ps210.OverlayValues[38] = d38
					ps210.OverlayValues[57] = d57
					ps210.OverlayValues[58] = d58
					ps210.OverlayValues[59] = d59
					ps210.OverlayValues[60] = d60
					ps210.OverlayValues[61] = d61
					ps210.OverlayValues[63] = d63
					ps210.OverlayValues[64] = d64
					ps210.OverlayValues[65] = d65
					ps210.OverlayValues[66] = d66
					ps210.OverlayValues[67] = d67
					ps210.OverlayValues[68] = d68
					ps210.OverlayValues[69] = d69
					ps210.OverlayValues[72] = d72
					ps210.OverlayValues[138] = d138
					ps210.OverlayValues[139] = d139
					ps210.OverlayValues[140] = d140
					ps210.OverlayValues[141] = d141
					ps210.OverlayValues[142] = d142
					ps210.OverlayValues[143] = d143
					ps210.OverlayValues[145] = d145
					ps210.OverlayValues[146] = d146
					ps210.OverlayValues[147] = d147
					ps210.OverlayValues[148] = d148
					ps210.OverlayValues[149] = d149
					ps210.OverlayValues[150] = d150
					ps210.OverlayValues[151] = d151
					ps210.OverlayValues[152] = d152
					ps210.OverlayValues[153] = d153
					ps210.OverlayValues[156] = d156
					ps210.OverlayValues[157] = d157
					ps210.OverlayValues[158] = d158
					ps210.OverlayValues[159] = d159
					ps211 := PhiState{General: true}
					ps211.OverlayValues = make([]JITValueDesc, 160)
					ps211.OverlayValues[1] = d1
					ps211.OverlayValues[2] = d2
					ps211.OverlayValues[3] = d3
					ps211.OverlayValues[4] = d4
					ps211.OverlayValues[5] = d5
					ps211.OverlayValues[6] = d6
					ps211.OverlayValues[7] = d7
					ps211.OverlayValues[8] = d8
					ps211.OverlayValues[9] = d9
					ps211.OverlayValues[10] = d10
					ps211.OverlayValues[11] = d11
					ps211.OverlayValues[12] = d12
					ps211.OverlayValues[13] = d13
					ps211.OverlayValues[14] = d14
					ps211.OverlayValues[15] = d15
					ps211.OverlayValues[18] = d18
					ps211.OverlayValues[38] = d38
					ps211.OverlayValues[57] = d57
					ps211.OverlayValues[58] = d58
					ps211.OverlayValues[59] = d59
					ps211.OverlayValues[60] = d60
					ps211.OverlayValues[61] = d61
					ps211.OverlayValues[63] = d63
					ps211.OverlayValues[64] = d64
					ps211.OverlayValues[65] = d65
					ps211.OverlayValues[66] = d66
					ps211.OverlayValues[67] = d67
					ps211.OverlayValues[68] = d68
					ps211.OverlayValues[69] = d69
					ps211.OverlayValues[72] = d72
					ps211.OverlayValues[138] = d138
					ps211.OverlayValues[139] = d139
					ps211.OverlayValues[140] = d140
					ps211.OverlayValues[141] = d141
					ps211.OverlayValues[142] = d142
					ps211.OverlayValues[143] = d143
					ps211.OverlayValues[145] = d145
					ps211.OverlayValues[146] = d146
					ps211.OverlayValues[147] = d147
					ps211.OverlayValues[148] = d148
					ps211.OverlayValues[149] = d149
					ps211.OverlayValues[150] = d150
					ps211.OverlayValues[151] = d151
					ps211.OverlayValues[152] = d152
					ps211.OverlayValues[153] = d153
					ps211.OverlayValues[156] = d156
					ps211.OverlayValues[157] = d157
					ps211.OverlayValues[158] = d158
					ps211.OverlayValues[159] = d159
					snap212 := d1
					snap213 := d2
					snap214 := d3
					snap215 := d4
					snap216 := d5
					snap217 := d6
					snap218 := d7
					snap219 := d8
					snap220 := d9
					snap221 := d10
					snap222 := d11
					snap223 := d12
					snap224 := d13
					snap225 := d14
					snap226 := d15
					snap227 := d18
					snap228 := d38
					snap229 := d57
					snap230 := d58
					snap231 := d59
					snap232 := d60
					snap233 := d61
					snap234 := d63
					snap235 := d64
					snap236 := d65
					snap237 := d66
					snap238 := d67
					snap239 := d68
					snap240 := d69
					snap241 := d72
					snap242 := d138
					snap243 := d139
					snap244 := d140
					snap245 := d141
					snap246 := d142
					snap247 := d143
					snap248 := d145
					snap249 := d146
					snap250 := d147
					snap251 := d148
					snap252 := d149
					snap253 := d150
					snap254 := d151
					snap255 := d152
					snap256 := d153
					snap257 := d156
					snap258 := d157
					snap259 := d158
					snap260 := d159
					alloc261 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps211)
					}
					ctx.RestoreAllocState(alloc261)
					d1 = snap212
					d2 = snap213
					d3 = snap214
					d4 = snap215
					d5 = snap216
					d6 = snap217
					d7 = snap218
					d8 = snap219
					d9 = snap220
					d10 = snap221
					d11 = snap222
					d12 = snap223
					d13 = snap224
					d14 = snap225
					d15 = snap226
					d18 = snap227
					d38 = snap228
					d57 = snap229
					d58 = snap230
					d59 = snap231
					d60 = snap232
					d61 = snap233
					d63 = snap234
					d64 = snap235
					d65 = snap236
					d66 = snap237
					d67 = snap238
					d68 = snap239
					d69 = snap240
					d72 = snap241
					d138 = snap242
					d139 = snap243
					d140 = snap244
					d141 = snap245
					d142 = snap246
					d143 = snap247
					d145 = snap248
					d146 = snap249
					d147 = snap250
					d148 = snap251
					d149 = snap252
					d150 = snap253
					d151 = snap254
					d152 = snap255
					d153 = snap256
					d156 = snap257
					d157 = snap258
					d158 = snap259
					d159 = snap260
					if !bbs[9].Rendered {
						return bbs[9].RenderPS(ps210)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d6)
					d263 = ctx.EmitSliceElementAddress(&d10, &d6, 16)
					ctx.EnsureDesc(&d263)
					r4 := ctx.AllocRegExcept(d263.Reg)
					ctx.EmitMovRegMem(r4, d263.Reg, 8)
					ctx.EmitMovRegMem(d263.Reg, d263.Reg, 0)
					d262 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d263.Reg, Reg2: r4}
					ctx.BindReg(d263.Reg, &d262)
					ctx.BindReg(r4, &d262)
					ctx.EnsureDesc(&d262)
					d264 = d262
					_ = d264
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d265 JITValueDesc
					if d264.Loc == LocImm {
						d265 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d264.Imm.Float())}
					} else if d264.Type == tagFloat && d264.Loc == LocReg {
						d265 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d264.Reg}
						ctx.BindReg(d264.Reg, &d265)
						ctx.BindReg(d264.Reg, &d265)
					} else if d264.Type == tagFloat && d264.Loc == LocRegPair {
						ctx.FreeReg(d264.Reg)
						d265 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d264.Reg2}
						ctx.BindReg(d264.Reg2, &d265)
						ctx.BindReg(d264.Reg2, &d265)
					} else {
						d265 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d264}, 1)
						d265.Type = tagFloat
						ctx.BindReg(d265.Reg, &d265)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d265)
					ctx.FreeDesc(&d262)
					ctx.EnsureDesc(&d6)
					d267 = ctx.EmitSliceElementAddress(&d12, &d6, 16)
					ctx.EnsureDesc(&d267)
					r5 := ctx.AllocRegExcept(d267.Reg)
					ctx.EmitMovRegMem(r5, d267.Reg, 8)
					ctx.EmitMovRegMem(d267.Reg, d267.Reg, 0)
					d266 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d267.Reg, Reg2: r5}
					ctx.BindReg(d267.Reg, &d266)
					ctx.BindReg(r5, &d266)
					ctx.EnsureDesc(&d266)
					d268 = d266
					_ = d268
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl18 := ctx.ReserveLabel()
					_ = lbl18
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d269 JITValueDesc
					if d268.Loc == LocImm {
						d269 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d268.Imm.Float())}
					} else if d268.Type == tagFloat && d268.Loc == LocReg {
						d269 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d268.Reg}
						ctx.BindReg(d268.Reg, &d269)
						ctx.BindReg(d268.Reg, &d269)
					} else if d268.Type == tagFloat && d268.Loc == LocRegPair {
						ctx.FreeReg(d268.Reg)
						d269 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d268.Reg2}
						ctx.BindReg(d268.Reg2, &d269)
						ctx.BindReg(d268.Reg2, &d269)
					} else {
						d269 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d268}, 1)
						d269.Type = tagFloat
						ctx.BindReg(d269.Reg, &d269)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d269)
					ctx.FreeDesc(&d266)
					ctx.EnsureDesc(&d265)
					ctx.EnsureDesc(&d265)
					ctx.EnsureDescsTogether(&d265, &d265)
					var d270 JITValueDesc
					if d265.Loc == LocImm {
						d270 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d265.Imm.Float() * d265.Imm.Float())}
					} else if d265.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d265.Reg)
						_, xBits := d265.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d265.Reg)
						d270 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d270)
					} else if d265.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d265.Reg)
						ctx.EmitMovRegReg(scratch, d265.Reg)
						_, yBits := d265.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d270 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d270)
					} else {
						r6 := ctx.AllocRegExcept(d265.Reg, d265.Reg)
						ctx.EmitMovRegReg(r6, d265.Reg)
						ctx.EmitMulFloat64(r6, d265.Reg)
						d270 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r6}
						ctx.BindReg(r6, &d270)
					}
					if d270.Loc == LocReg && d265.Loc == LocReg && d270.Reg == d265.Reg {
						ctx.TransferReg(d265.Reg)
						d265.Loc = LocNone
					}
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d270)
					ctx.EnsureDescsTogether(&d4, &d270)
					var d271 JITValueDesc
					if d4.Loc == LocImm && d270.Loc == LocImm {
						d271 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float() + d270.Imm.Float())}
					} else if d4.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d270.Reg)
						_, xBits := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d270.Reg)
						d271 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d271)
					} else if d270.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						_, yBits := d270.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d271 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d271)
					} else {
						r7 := ctx.AllocRegExcept(d4.Reg, d270.Reg)
						ctx.EmitMovRegReg(r7, d4.Reg)
						ctx.EmitAddFloat64(r7, d270.Reg)
						d271 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r7}
						ctx.BindReg(r7, &d271)
					}
					if d271.Loc == LocReg && d4.Loc == LocReg && d271.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d271)
					ctx.EmitStoreToStack(d271, int32(bbs[6].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d271)
					ctx.FreeDesc(&d270)
					ctx.EnsureDesc(&d269)
					ctx.EnsureDesc(&d269)
					ctx.EnsureDescsTogether(&d269, &d269)
					var d272 JITValueDesc
					if d269.Loc == LocImm {
						d272 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d269.Imm.Float() * d269.Imm.Float())}
					} else if d269.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d269.Reg)
						_, xBits := d269.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d269.Reg)
						d272 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d272)
					} else if d269.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d269.Reg)
						ctx.EmitMovRegReg(scratch, d269.Reg)
						_, yBits := d269.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d272 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d272)
					} else {
						r8 := ctx.AllocRegExcept(d269.Reg, d269.Reg)
						ctx.EmitMovRegReg(r8, d269.Reg)
						ctx.EmitMulFloat64(r8, d269.Reg)
						d272 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r8}
						ctx.BindReg(r8, &d272)
					}
					if d272.Loc == LocReg && d269.Loc == LocReg && d272.Reg == d269.Reg {
						ctx.TransferReg(d269.Reg)
						d269.Loc = LocNone
					}
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d272)
					ctx.EnsureDescsTogether(&d5, &d272)
					var d273 JITValueDesc
					if d5.Loc == LocImm && d272.Loc == LocImm {
						d273 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d5.Imm.Float() + d272.Imm.Float())}
					} else if d5.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d272.Reg)
						_, xBits := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d272.Reg)
						d273 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d273)
					} else if d272.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						ctx.EmitMovRegReg(scratch, d5.Reg)
						_, yBits := d272.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d273 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d273)
					} else {
						r9 := ctx.AllocRegExcept(d5.Reg, d272.Reg)
						ctx.EmitMovRegReg(r9, d5.Reg)
						ctx.EmitAddFloat64(r9, d272.Reg)
						d273 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r9}
						ctx.BindReg(r9, &d273)
					}
					if d273.Loc == LocReg && d5.Loc == LocReg && d273.Reg == d5.Reg {
						ctx.TransferReg(d5.Reg)
						d5.Loc = LocNone
					}
					ctx.EnsureDesc(&d273)
					ctx.EmitStoreToStack(d273, int32(bbs[6].PhiBase)+int32(32))
					ctx.StabilizeDescForControlFlow(&d273)
					ctx.FreeDesc(&d272)
					ctx.EnsureDesc(&d265)
					ctx.EnsureDesc(&d269)
					ctx.EnsureDescsTogether(&d265, &d269)
					var d274 JITValueDesc
					if d265.Loc == LocImm && d269.Loc == LocImm {
						d274 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d265.Imm.Float() * d269.Imm.Float())}
					} else if d265.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d269.Reg)
						_, xBits := d265.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d269.Reg)
						d274 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d274)
					} else if d269.Loc == LocImm {
						_, yBits := d269.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d265.Reg, RegR11)
						d274 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d265.Reg}
						ctx.BindReg(d265.Reg, &d274)
					} else {
						ctx.EmitMulFloat64(d265.Reg, d269.Reg)
						d274 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d265.Reg}
						ctx.BindReg(d265.Reg, &d274)
					}
					if d274.Loc == LocReg && d265.Loc == LocReg && d274.Reg == d265.Reg {
						ctx.TransferReg(d265.Reg)
						d265.Loc = LocNone
					}
					ctx.FreeDesc(&d265)
					ctx.FreeDesc(&d269)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d274)
					ctx.EnsureDescsTogether(&d3, &d274)
					var d275 JITValueDesc
					if d3.Loc == LocImm && d274.Loc == LocImm {
						d275 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float() + d274.Imm.Float())}
					} else if d3.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d274.Reg)
						_, xBits := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d274.Reg)
						d275 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d275)
					} else if d274.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						_, yBits := d274.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d275 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d275)
					} else {
						r10 := ctx.AllocRegExcept(d3.Reg, d274.Reg)
						ctx.EmitMovRegReg(r10, d3.Reg)
						ctx.EmitAddFloat64(r10, d274.Reg)
						d275 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r10}
						ctx.BindReg(r10, &d275)
					}
					if d275.Loc == LocReg && d3.Loc == LocReg && d275.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d275)
					ctx.EmitStoreToStack(d275, int32(bbs[6].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d275)
					ctx.FreeDesc(&d274)
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d6)
					var d276 JITValueDesc
					if d6.Loc == LocImm {
						d276 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitMovRegReg(scratch, d6.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d276 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d276)
					}
					if d276.Loc == LocReg && d6.Loc == LocReg && d276.Reg == d6.Reg {
						ctx.TransferReg(d6.Reg)
						d6.Loc = LocNone
					}
					ctx.EnsureDesc(&d276)
					ctx.EmitStoreToStack(d276, int32(bbs[6].PhiBase)+int32(48))
					ctx.StabilizeDescForControlFlow(&d276)
					if ps.General {
					}
					ps277 := PhiState{General: ps.General}
					ps277.OverlayValues = make([]JITValueDesc, 277)
					ps277.OverlayValues[1] = d1
					ps277.OverlayValues[2] = d2
					ps277.OverlayValues[3] = d3
					ps277.OverlayValues[4] = d4
					ps277.OverlayValues[5] = d5
					ps277.OverlayValues[6] = d6
					ps277.OverlayValues[7] = d7
					ps277.OverlayValues[8] = d8
					ps277.OverlayValues[9] = d9
					ps277.OverlayValues[10] = d10
					ps277.OverlayValues[11] = d11
					ps277.OverlayValues[12] = d12
					ps277.OverlayValues[13] = d13
					ps277.OverlayValues[14] = d14
					ps277.OverlayValues[15] = d15
					ps277.OverlayValues[18] = d18
					ps277.OverlayValues[38] = d38
					ps277.OverlayValues[57] = d57
					ps277.OverlayValues[58] = d58
					ps277.OverlayValues[59] = d59
					ps277.OverlayValues[60] = d60
					ps277.OverlayValues[61] = d61
					ps277.OverlayValues[63] = d63
					ps277.OverlayValues[64] = d64
					ps277.OverlayValues[65] = d65
					ps277.OverlayValues[66] = d66
					ps277.OverlayValues[67] = d67
					ps277.OverlayValues[68] = d68
					ps277.OverlayValues[69] = d69
					ps277.OverlayValues[72] = d72
					ps277.OverlayValues[138] = d138
					ps277.OverlayValues[139] = d139
					ps277.OverlayValues[140] = d140
					ps277.OverlayValues[141] = d141
					ps277.OverlayValues[142] = d142
					ps277.OverlayValues[143] = d143
					ps277.OverlayValues[145] = d145
					ps277.OverlayValues[146] = d146
					ps277.OverlayValues[147] = d147
					ps277.OverlayValues[148] = d148
					ps277.OverlayValues[149] = d149
					ps277.OverlayValues[150] = d150
					ps277.OverlayValues[151] = d151
					ps277.OverlayValues[152] = d152
					ps277.OverlayValues[153] = d153
					ps277.OverlayValues[156] = d156
					ps277.OverlayValues[157] = d157
					ps277.OverlayValues[158] = d158
					ps277.OverlayValues[159] = d159
					ps277.OverlayValues[262] = d262
					ps277.OverlayValues[263] = d263
					ps277.OverlayValues[264] = d264
					ps277.OverlayValues[265] = d265
					ps277.OverlayValues[266] = d266
					ps277.OverlayValues[267] = d267
					ps277.OverlayValues[268] = d268
					ps277.OverlayValues[269] = d269
					ps277.OverlayValues[270] = d270
					ps277.OverlayValues[271] = d271
					ps277.OverlayValues[272] = d272
					ps277.OverlayValues[273] = d273
					ps277.OverlayValues[274] = d274
					ps277.OverlayValues[275] = d275
					ps277.OverlayValues[276] = d276
					ps277.PhiValues = make([]JITValueDesc, 4)
					if ps277.General && bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					return bbs[6].RenderPS(ps277)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDescsTogether(&d4, &d5)
					var d278 JITValueDesc
					if d4.Loc == LocImm && d5.Loc == LocImm {
						d278 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float() * d5.Imm.Float())}
					} else if d4.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d5.Reg)
						_, xBits := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d5.Reg)
						d278 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d278)
					} else if d5.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d4.Reg)
						ctx.EmitMovRegReg(scratch, d4.Reg)
						_, yBits := d5.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(scratch, RegR11)
						d278 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d278)
					} else {
						r11 := ctx.AllocRegExcept(d4.Reg, d5.Reg)
						ctx.EmitMovRegReg(r11, d4.Reg)
						ctx.EmitMulFloat64(r11, d5.Reg)
						d278 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r11}
						ctx.BindReg(r11, &d278)
					}
					if d278.Loc == LocReg && d4.Loc == LocReg && d278.Reg == d4.Reg {
						ctx.TransferReg(d4.Reg)
						d4.Loc = LocNone
					}
					ctx.EnsureDesc(&d278)
					var d279 JITValueDesc
					if d278.Loc == LocImm {
						d279 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d278.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d278)
						var d280 JITValueDesc
						if d278.Loc == LocRegPair {
							ctx.FreeReg(d278.Reg)
							d280 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d278.Reg2}
							ctx.BindReg(d278.Reg2, &d280)
							ctx.BindReg(d278.Reg2, &d280)
						} else {
							d280 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d278.Reg}
							ctx.BindReg(d278.Reg, &d280)
							ctx.BindReg(d278.Reg, &d280)
						}
						d279 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d280}, 1)
						d279.Type = tagFloat
						ctx.BindReg(d279.Reg, &d279)
					}
					ctx.FreeDesc(&d278)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d279)
					ctx.EnsureDescsTogether(&d3, &d279)
					var d281 JITValueDesc
					if d3.Loc == LocImm && d279.Loc == LocImm {
						d281 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float() / d279.Imm.Float())}
					} else if d3.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d279.Reg)
						_, xBits := d3.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitDivFloat64(scratch, d279.Reg)
						d281 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d281)
					} else if d279.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						_, yBits := d279.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitDivFloat64(scratch, RegR11)
						d281 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d281)
					} else {
						r12 := ctx.AllocRegExcept(d3.Reg, d279.Reg)
						ctx.EmitMovRegReg(r12, d3.Reg)
						ctx.EmitDivFloat64(r12, d279.Reg)
						d281 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r12}
						ctx.BindReg(r12, &d281)
					}
					if d281.Loc == LocReg && d3.Loc == LocReg && d281.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.EnsureDesc(&d281)
					ctx.EmitStoreToStack(d281, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d281)
					ctx.FreeDesc(&d279)
					if ps.General {
					}
					ps282 := PhiState{General: ps.General}
					ps282.OverlayValues = make([]JITValueDesc, 282)
					ps282.OverlayValues[1] = d1
					ps282.OverlayValues[2] = d2
					ps282.OverlayValues[3] = d3
					ps282.OverlayValues[4] = d4
					ps282.OverlayValues[5] = d5
					ps282.OverlayValues[6] = d6
					ps282.OverlayValues[7] = d7
					ps282.OverlayValues[8] = d8
					ps282.OverlayValues[9] = d9
					ps282.OverlayValues[10] = d10
					ps282.OverlayValues[11] = d11
					ps282.OverlayValues[12] = d12
					ps282.OverlayValues[13] = d13
					ps282.OverlayValues[14] = d14
					ps282.OverlayValues[15] = d15
					ps282.OverlayValues[18] = d18
					ps282.OverlayValues[38] = d38
					ps282.OverlayValues[57] = d57
					ps282.OverlayValues[58] = d58
					ps282.OverlayValues[59] = d59
					ps282.OverlayValues[60] = d60
					ps282.OverlayValues[61] = d61
					ps282.OverlayValues[63] = d63
					ps282.OverlayValues[64] = d64
					ps282.OverlayValues[65] = d65
					ps282.OverlayValues[66] = d66
					ps282.OverlayValues[67] = d67
					ps282.OverlayValues[68] = d68
					ps282.OverlayValues[69] = d69
					ps282.OverlayValues[72] = d72
					ps282.OverlayValues[138] = d138
					ps282.OverlayValues[139] = d139
					ps282.OverlayValues[140] = d140
					ps282.OverlayValues[141] = d141
					ps282.OverlayValues[142] = d142
					ps282.OverlayValues[143] = d143
					ps282.OverlayValues[145] = d145
					ps282.OverlayValues[146] = d146
					ps282.OverlayValues[147] = d147
					ps282.OverlayValues[148] = d148
					ps282.OverlayValues[149] = d149
					ps282.OverlayValues[150] = d150
					ps282.OverlayValues[151] = d151
					ps282.OverlayValues[152] = d152
					ps282.OverlayValues[153] = d153
					ps282.OverlayValues[156] = d156
					ps282.OverlayValues[157] = d157
					ps282.OverlayValues[158] = d158
					ps282.OverlayValues[159] = d159
					ps282.OverlayValues[262] = d262
					ps282.OverlayValues[263] = d263
					ps282.OverlayValues[264] = d264
					ps282.OverlayValues[265] = d265
					ps282.OverlayValues[266] = d266
					ps282.OverlayValues[267] = d267
					ps282.OverlayValues[268] = d268
					ps282.OverlayValues[269] = d269
					ps282.OverlayValues[270] = d270
					ps282.OverlayValues[271] = d271
					ps282.OverlayValues[272] = d272
					ps282.OverlayValues[273] = d273
					ps282.OverlayValues[274] = d274
					ps282.OverlayValues[275] = d275
					ps282.OverlayValues[276] = d276
					ps282.OverlayValues[278] = d278
					ps282.OverlayValues[279] = d279
					ps282.OverlayValues[280] = d280
					ps282.OverlayValues[281] = d281
					ps282.PhiValues = make([]JITValueDesc, 1)
					if ps282.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps282)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
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
					ctx.ReclaimUntrackedRegs()
					var d283 JITValueDesc
					if d12.SliceSizeKnown {
						d283 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.KnownSliceLen))}
					} else if d12.Loc == LocImm {
						d283 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.StackOff))}
					} else if d12.Loc == LocStackTriple {
						d283 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d12.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d12)
						if d12.Loc == LocRegPair || d12.Loc == LocRegTriple {
							d283 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg2, ID: 0}
						} else if d12.Loc == LocReg {
							d283 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d6)
					ctx.EnsureDesc(&d283)
					ctx.EnsureDescsTogether(&d6, &d283)
					var d284 JITValueDesc
					if d6.Loc == LocImm && d283.Loc == LocImm {
						d284 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() < d283.Imm.Int())}
					} else if d283.Loc == LocImm {
						r13 := ctx.AllocRegExcept(d6.Reg)
						if d283.Imm.Int() >= -2147483648 && d283.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d6.Reg, int32(d283.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d283.Imm.Int()))
							ctx.EmitCmpInt64(d6.Reg, RegR11)
						}
						d284 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
						ctx.BindReg(r13, &d284)
					} else if d6.Loc == LocImm {
						r14 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d283.Reg)
						d284 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
						ctx.BindReg(r14, &d284)
					} else {
						r15 := ctx.AllocRegExcept(d6.Reg)
						ctx.EmitCmpInt64(d6.Reg, d283.Reg)
						d284 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondSignedLess}
						ctx.BindReg(r15, &d284)
					}
					ctx.FreeDesc(&d283)
					d285 = d284
					ctx.EnsureDesc(&d285)
					if d285.Loc != LocImm && d285.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d285.Loc == LocImm {
						if d285.Imm.Bool() {
							if ps.General {
							}
							ps286 := PhiState{General: ps.General}
							ps286.OverlayValues = make([]JITValueDesc, 286)
							ps286.OverlayValues[1] = d1
							ps286.OverlayValues[2] = d2
							ps286.OverlayValues[3] = d3
							ps286.OverlayValues[4] = d4
							ps286.OverlayValues[5] = d5
							ps286.OverlayValues[6] = d6
							ps286.OverlayValues[7] = d7
							ps286.OverlayValues[8] = d8
							ps286.OverlayValues[9] = d9
							ps286.OverlayValues[10] = d10
							ps286.OverlayValues[11] = d11
							ps286.OverlayValues[12] = d12
							ps286.OverlayValues[13] = d13
							ps286.OverlayValues[14] = d14
							ps286.OverlayValues[15] = d15
							ps286.OverlayValues[18] = d18
							ps286.OverlayValues[38] = d38
							ps286.OverlayValues[57] = d57
							ps286.OverlayValues[58] = d58
							ps286.OverlayValues[59] = d59
							ps286.OverlayValues[60] = d60
							ps286.OverlayValues[61] = d61
							ps286.OverlayValues[63] = d63
							ps286.OverlayValues[64] = d64
							ps286.OverlayValues[65] = d65
							ps286.OverlayValues[66] = d66
							ps286.OverlayValues[67] = d67
							ps286.OverlayValues[68] = d68
							ps286.OverlayValues[69] = d69
							ps286.OverlayValues[72] = d72
							ps286.OverlayValues[138] = d138
							ps286.OverlayValues[139] = d139
							ps286.OverlayValues[140] = d140
							ps286.OverlayValues[141] = d141
							ps286.OverlayValues[142] = d142
							ps286.OverlayValues[143] = d143
							ps286.OverlayValues[145] = d145
							ps286.OverlayValues[146] = d146
							ps286.OverlayValues[147] = d147
							ps286.OverlayValues[148] = d148
							ps286.OverlayValues[149] = d149
							ps286.OverlayValues[150] = d150
							ps286.OverlayValues[151] = d151
							ps286.OverlayValues[152] = d152
							ps286.OverlayValues[153] = d153
							ps286.OverlayValues[156] = d156
							ps286.OverlayValues[157] = d157
							ps286.OverlayValues[158] = d158
							ps286.OverlayValues[159] = d159
							ps286.OverlayValues[262] = d262
							ps286.OverlayValues[263] = d263
							ps286.OverlayValues[264] = d264
							ps286.OverlayValues[265] = d265
							ps286.OverlayValues[266] = d266
							ps286.OverlayValues[267] = d267
							ps286.OverlayValues[268] = d268
							ps286.OverlayValues[269] = d269
							ps286.OverlayValues[270] = d270
							ps286.OverlayValues[271] = d271
							ps286.OverlayValues[272] = d272
							ps286.OverlayValues[273] = d273
							ps286.OverlayValues[274] = d274
							ps286.OverlayValues[275] = d275
							ps286.OverlayValues[276] = d276
							ps286.OverlayValues[278] = d278
							ps286.OverlayValues[279] = d279
							ps286.OverlayValues[280] = d280
							ps286.OverlayValues[281] = d281
							ps286.OverlayValues[283] = d283
							ps286.OverlayValues[284] = d284
							ps286.OverlayValues[285] = d285
							return bbs[7].RenderPS(ps286)
						}
						if ps.General {
						}
						ps287 := PhiState{General: ps.General}
						ps287.OverlayValues = make([]JITValueDesc, 286)
						ps287.OverlayValues[1] = d1
						ps287.OverlayValues[2] = d2
						ps287.OverlayValues[3] = d3
						ps287.OverlayValues[4] = d4
						ps287.OverlayValues[5] = d5
						ps287.OverlayValues[6] = d6
						ps287.OverlayValues[7] = d7
						ps287.OverlayValues[8] = d8
						ps287.OverlayValues[9] = d9
						ps287.OverlayValues[10] = d10
						ps287.OverlayValues[11] = d11
						ps287.OverlayValues[12] = d12
						ps287.OverlayValues[13] = d13
						ps287.OverlayValues[14] = d14
						ps287.OverlayValues[15] = d15
						ps287.OverlayValues[18] = d18
						ps287.OverlayValues[38] = d38
						ps287.OverlayValues[57] = d57
						ps287.OverlayValues[58] = d58
						ps287.OverlayValues[59] = d59
						ps287.OverlayValues[60] = d60
						ps287.OverlayValues[61] = d61
						ps287.OverlayValues[63] = d63
						ps287.OverlayValues[64] = d64
						ps287.OverlayValues[65] = d65
						ps287.OverlayValues[66] = d66
						ps287.OverlayValues[67] = d67
						ps287.OverlayValues[68] = d68
						ps287.OverlayValues[69] = d69
						ps287.OverlayValues[72] = d72
						ps287.OverlayValues[138] = d138
						ps287.OverlayValues[139] = d139
						ps287.OverlayValues[140] = d140
						ps287.OverlayValues[141] = d141
						ps287.OverlayValues[142] = d142
						ps287.OverlayValues[143] = d143
						ps287.OverlayValues[145] = d145
						ps287.OverlayValues[146] = d146
						ps287.OverlayValues[147] = d147
						ps287.OverlayValues[148] = d148
						ps287.OverlayValues[149] = d149
						ps287.OverlayValues[150] = d150
						ps287.OverlayValues[151] = d151
						ps287.OverlayValues[152] = d152
						ps287.OverlayValues[153] = d153
						ps287.OverlayValues[156] = d156
						ps287.OverlayValues[157] = d157
						ps287.OverlayValues[158] = d158
						ps287.OverlayValues[159] = d159
						ps287.OverlayValues[262] = d262
						ps287.OverlayValues[263] = d263
						ps287.OverlayValues[264] = d264
						ps287.OverlayValues[265] = d265
						ps287.OverlayValues[266] = d266
						ps287.OverlayValues[267] = d267
						ps287.OverlayValues[268] = d268
						ps287.OverlayValues[269] = d269
						ps287.OverlayValues[270] = d270
						ps287.OverlayValues[271] = d271
						ps287.OverlayValues[272] = d272
						ps287.OverlayValues[273] = d273
						ps287.OverlayValues[274] = d274
						ps287.OverlayValues[275] = d275
						ps287.OverlayValues[276] = d276
						ps287.OverlayValues[278] = d278
						ps287.OverlayValues[279] = d279
						ps287.OverlayValues[280] = d280
						ps287.OverlayValues[281] = d281
						ps287.OverlayValues[283] = d283
						ps287.OverlayValues[284] = d284
						ps287.OverlayValues[285] = d285
						return bbs[8].RenderPS(ps287)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					ctx.EmitJump(d285.Condition, lbl8)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d284)
					snap288 := d1
					snap289 := d2
					snap290 := d3
					snap291 := d4
					snap292 := d5
					snap293 := d6
					snap294 := d7
					snap295 := d8
					snap296 := d9
					snap297 := d10
					snap298 := d11
					snap299 := d12
					snap300 := d13
					snap301 := d14
					snap302 := d15
					snap303 := d18
					snap304 := d38
					snap305 := d57
					snap306 := d58
					snap307 := d59
					snap308 := d60
					snap309 := d61
					snap310 := d63
					snap311 := d64
					snap312 := d65
					snap313 := d66
					snap314 := d67
					snap315 := d68
					snap316 := d69
					snap317 := d72
					snap318 := d138
					snap319 := d139
					snap320 := d140
					snap321 := d141
					snap322 := d142
					snap323 := d143
					snap324 := d145
					snap325 := d146
					snap326 := d147
					snap327 := d148
					snap328 := d149
					snap329 := d150
					snap330 := d151
					snap331 := d152
					snap332 := d153
					snap333 := d156
					snap334 := d157
					snap335 := d158
					snap336 := d159
					snap337 := d262
					snap338 := d263
					snap339 := d264
					snap340 := d265
					snap341 := d266
					snap342 := d267
					snap343 := d268
					snap344 := d269
					snap345 := d270
					snap346 := d271
					snap347 := d272
					snap348 := d273
					snap349 := d274
					snap350 := d275
					snap351 := d276
					snap352 := d278
					snap353 := d279
					snap354 := d280
					snap355 := d281
					snap356 := d283
					snap357 := d284
					snap358 := d285
					alloc359 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc359)
					d1 = snap288
					d2 = snap289
					d3 = snap290
					d4 = snap291
					d5 = snap292
					d6 = snap293
					d7 = snap294
					d8 = snap295
					d9 = snap296
					d10 = snap297
					d11 = snap298
					d12 = snap299
					d13 = snap300
					d14 = snap301
					d15 = snap302
					d18 = snap303
					d38 = snap304
					d57 = snap305
					d58 = snap306
					d59 = snap307
					d60 = snap308
					d61 = snap309
					d63 = snap310
					d64 = snap311
					d65 = snap312
					d66 = snap313
					d67 = snap314
					d68 = snap315
					d69 = snap316
					d72 = snap317
					d138 = snap318
					d139 = snap319
					d140 = snap320
					d141 = snap321
					d142 = snap322
					d143 = snap323
					d145 = snap324
					d146 = snap325
					d147 = snap326
					d148 = snap327
					d149 = snap328
					d150 = snap329
					d151 = snap330
					d152 = snap331
					d153 = snap332
					d156 = snap333
					d157 = snap334
					d158 = snap335
					d159 = snap336
					d262 = snap337
					d263 = snap338
					d264 = snap339
					d265 = snap340
					d266 = snap341
					d267 = snap342
					d268 = snap343
					d269 = snap344
					d270 = snap345
					d271 = snap346
					d272 = snap347
					d273 = snap348
					d274 = snap349
					d275 = snap350
					d276 = snap351
					d278 = snap352
					d279 = snap353
					d280 = snap354
					d281 = snap355
					d283 = snap356
					d284 = snap357
					d285 = snap358
					ctx.RestoreAllocState(alloc359)
					d1 = snap288
					d2 = snap289
					d3 = snap290
					d4 = snap291
					d5 = snap292
					d6 = snap293
					d7 = snap294
					d8 = snap295
					d9 = snap296
					d10 = snap297
					d11 = snap298
					d12 = snap299
					d13 = snap300
					d14 = snap301
					d15 = snap302
					d18 = snap303
					d38 = snap304
					d57 = snap305
					d58 = snap306
					d59 = snap307
					d60 = snap308
					d61 = snap309
					d63 = snap310
					d64 = snap311
					d65 = snap312
					d66 = snap313
					d67 = snap314
					d68 = snap315
					d69 = snap316
					d72 = snap317
					d138 = snap318
					d139 = snap319
					d140 = snap320
					d141 = snap321
					d142 = snap322
					d143 = snap323
					d145 = snap324
					d146 = snap325
					d147 = snap326
					d148 = snap327
					d149 = snap328
					d150 = snap329
					d151 = snap330
					d152 = snap331
					d153 = snap332
					d156 = snap333
					d157 = snap334
					d158 = snap335
					d159 = snap336
					d262 = snap337
					d263 = snap338
					d264 = snap339
					d265 = snap340
					d266 = snap341
					d267 = snap342
					d268 = snap343
					d269 = snap344
					d270 = snap345
					d271 = snap346
					d272 = snap347
					d273 = snap348
					d274 = snap349
					d275 = snap350
					d276 = snap351
					d278 = snap352
					d279 = snap353
					d280 = snap354
					d281 = snap355
					d283 = snap356
					d284 = snap357
					d285 = snap358
					ps360 := PhiState{General: true}
					ps360.OverlayValues = make([]JITValueDesc, 286)
					ps360.OverlayValues[1] = d1
					ps360.OverlayValues[2] = d2
					ps360.OverlayValues[3] = d3
					ps360.OverlayValues[4] = d4
					ps360.OverlayValues[5] = d5
					ps360.OverlayValues[6] = d6
					ps360.OverlayValues[7] = d7
					ps360.OverlayValues[8] = d8
					ps360.OverlayValues[9] = d9
					ps360.OverlayValues[10] = d10
					ps360.OverlayValues[11] = d11
					ps360.OverlayValues[12] = d12
					ps360.OverlayValues[13] = d13
					ps360.OverlayValues[14] = d14
					ps360.OverlayValues[15] = d15
					ps360.OverlayValues[18] = d18
					ps360.OverlayValues[38] = d38
					ps360.OverlayValues[57] = d57
					ps360.OverlayValues[58] = d58
					ps360.OverlayValues[59] = d59
					ps360.OverlayValues[60] = d60
					ps360.OverlayValues[61] = d61
					ps360.OverlayValues[63] = d63
					ps360.OverlayValues[64] = d64
					ps360.OverlayValues[65] = d65
					ps360.OverlayValues[66] = d66
					ps360.OverlayValues[67] = d67
					ps360.OverlayValues[68] = d68
					ps360.OverlayValues[69] = d69
					ps360.OverlayValues[72] = d72
					ps360.OverlayValues[138] = d138
					ps360.OverlayValues[139] = d139
					ps360.OverlayValues[140] = d140
					ps360.OverlayValues[141] = d141
					ps360.OverlayValues[142] = d142
					ps360.OverlayValues[143] = d143
					ps360.OverlayValues[145] = d145
					ps360.OverlayValues[146] = d146
					ps360.OverlayValues[147] = d147
					ps360.OverlayValues[148] = d148
					ps360.OverlayValues[149] = d149
					ps360.OverlayValues[150] = d150
					ps360.OverlayValues[151] = d151
					ps360.OverlayValues[152] = d152
					ps360.OverlayValues[153] = d153
					ps360.OverlayValues[156] = d156
					ps360.OverlayValues[157] = d157
					ps360.OverlayValues[158] = d158
					ps360.OverlayValues[159] = d159
					ps360.OverlayValues[262] = d262
					ps360.OverlayValues[263] = d263
					ps360.OverlayValues[264] = d264
					ps360.OverlayValues[265] = d265
					ps360.OverlayValues[266] = d266
					ps360.OverlayValues[267] = d267
					ps360.OverlayValues[268] = d268
					ps360.OverlayValues[269] = d269
					ps360.OverlayValues[270] = d270
					ps360.OverlayValues[271] = d271
					ps360.OverlayValues[272] = d272
					ps360.OverlayValues[273] = d273
					ps360.OverlayValues[274] = d274
					ps360.OverlayValues[275] = d275
					ps360.OverlayValues[276] = d276
					ps360.OverlayValues[278] = d278
					ps360.OverlayValues[279] = d279
					ps360.OverlayValues[280] = d280
					ps360.OverlayValues[281] = d281
					ps360.OverlayValues[283] = d283
					ps360.OverlayValues[284] = d284
					ps360.OverlayValues[285] = d285
					ps361 := PhiState{General: true}
					ps361.OverlayValues = make([]JITValueDesc, 286)
					ps361.OverlayValues[1] = d1
					ps361.OverlayValues[2] = d2
					ps361.OverlayValues[3] = d3
					ps361.OverlayValues[4] = d4
					ps361.OverlayValues[5] = d5
					ps361.OverlayValues[6] = d6
					ps361.OverlayValues[7] = d7
					ps361.OverlayValues[8] = d8
					ps361.OverlayValues[9] = d9
					ps361.OverlayValues[10] = d10
					ps361.OverlayValues[11] = d11
					ps361.OverlayValues[12] = d12
					ps361.OverlayValues[13] = d13
					ps361.OverlayValues[14] = d14
					ps361.OverlayValues[15] = d15
					ps361.OverlayValues[18] = d18
					ps361.OverlayValues[38] = d38
					ps361.OverlayValues[57] = d57
					ps361.OverlayValues[58] = d58
					ps361.OverlayValues[59] = d59
					ps361.OverlayValues[60] = d60
					ps361.OverlayValues[61] = d61
					ps361.OverlayValues[63] = d63
					ps361.OverlayValues[64] = d64
					ps361.OverlayValues[65] = d65
					ps361.OverlayValues[66] = d66
					ps361.OverlayValues[67] = d67
					ps361.OverlayValues[68] = d68
					ps361.OverlayValues[69] = d69
					ps361.OverlayValues[72] = d72
					ps361.OverlayValues[138] = d138
					ps361.OverlayValues[139] = d139
					ps361.OverlayValues[140] = d140
					ps361.OverlayValues[141] = d141
					ps361.OverlayValues[142] = d142
					ps361.OverlayValues[143] = d143
					ps361.OverlayValues[145] = d145
					ps361.OverlayValues[146] = d146
					ps361.OverlayValues[147] = d147
					ps361.OverlayValues[148] = d148
					ps361.OverlayValues[149] = d149
					ps361.OverlayValues[150] = d150
					ps361.OverlayValues[151] = d151
					ps361.OverlayValues[152] = d152
					ps361.OverlayValues[153] = d153
					ps361.OverlayValues[156] = d156
					ps361.OverlayValues[157] = d157
					ps361.OverlayValues[158] = d158
					ps361.OverlayValues[159] = d159
					ps361.OverlayValues[262] = d262
					ps361.OverlayValues[263] = d263
					ps361.OverlayValues[264] = d264
					ps361.OverlayValues[265] = d265
					ps361.OverlayValues[266] = d266
					ps361.OverlayValues[267] = d267
					ps361.OverlayValues[268] = d268
					ps361.OverlayValues[269] = d269
					ps361.OverlayValues[270] = d270
					ps361.OverlayValues[271] = d271
					ps361.OverlayValues[272] = d272
					ps361.OverlayValues[273] = d273
					ps361.OverlayValues[274] = d274
					ps361.OverlayValues[275] = d275
					ps361.OverlayValues[276] = d276
					ps361.OverlayValues[278] = d278
					ps361.OverlayValues[279] = d279
					ps361.OverlayValues[280] = d280
					ps361.OverlayValues[281] = d281
					ps361.OverlayValues[283] = d283
					ps361.OverlayValues[284] = d284
					ps361.OverlayValues[285] = d285
					snap362 := d1
					snap363 := d2
					snap364 := d3
					snap365 := d4
					snap366 := d5
					snap367 := d6
					snap368 := d7
					snap369 := d8
					snap370 := d9
					snap371 := d10
					snap372 := d11
					snap373 := d12
					snap374 := d13
					snap375 := d14
					snap376 := d15
					snap377 := d18
					snap378 := d38
					snap379 := d57
					snap380 := d58
					snap381 := d59
					snap382 := d60
					snap383 := d61
					snap384 := d63
					snap385 := d64
					snap386 := d65
					snap387 := d66
					snap388 := d67
					snap389 := d68
					snap390 := d69
					snap391 := d72
					snap392 := d138
					snap393 := d139
					snap394 := d140
					snap395 := d141
					snap396 := d142
					snap397 := d143
					snap398 := d145
					snap399 := d146
					snap400 := d147
					snap401 := d148
					snap402 := d149
					snap403 := d150
					snap404 := d151
					snap405 := d152
					snap406 := d153
					snap407 := d156
					snap408 := d157
					snap409 := d158
					snap410 := d159
					snap411 := d262
					snap412 := d263
					snap413 := d264
					snap414 := d265
					snap415 := d266
					snap416 := d267
					snap417 := d268
					snap418 := d269
					snap419 := d270
					snap420 := d271
					snap421 := d272
					snap422 := d273
					snap423 := d274
					snap424 := d275
					snap425 := d276
					snap426 := d278
					snap427 := d279
					snap428 := d280
					snap429 := d281
					snap430 := d283
					snap431 := d284
					snap432 := d285
					alloc433 := ctx.SnapshotAllocState()
					if !bbs[8].Rendered {
						bbs[8].RenderPS(ps361)
					}
					ctx.RestoreAllocState(alloc433)
					d1 = snap362
					d2 = snap363
					d3 = snap364
					d4 = snap365
					d5 = snap366
					d6 = snap367
					d7 = snap368
					d8 = snap369
					d9 = snap370
					d10 = snap371
					d11 = snap372
					d12 = snap373
					d13 = snap374
					d14 = snap375
					d15 = snap376
					d18 = snap377
					d38 = snap378
					d57 = snap379
					d58 = snap380
					d59 = snap381
					d60 = snap382
					d61 = snap383
					d63 = snap384
					d64 = snap385
					d65 = snap386
					d66 = snap387
					d67 = snap388
					d68 = snap389
					d69 = snap390
					d72 = snap391
					d138 = snap392
					d139 = snap393
					d140 = snap394
					d141 = snap395
					d142 = snap396
					d143 = snap397
					d145 = snap398
					d146 = snap399
					d147 = snap400
					d148 = snap401
					d149 = snap402
					d150 = snap403
					d151 = snap404
					d152 = snap405
					d153 = snap406
					d156 = snap407
					d157 = snap408
					d158 = snap409
					d159 = snap410
					d262 = snap411
					d263 = snap412
					d264 = snap413
					d265 = snap414
					d266 = snap415
					d267 = snap416
					d268 = snap417
					d269 = snap418
					d270 = snap419
					d271 = snap420
					d272 = snap421
					d273 = snap422
					d274 = snap423
					d275 = snap424
					d276 = snap425
					d278 = snap426
					d279 = snap427
					d280 = snap428
					d281 = snap429
					d283 = snap430
					d284 = snap431
					d285 = snap432
					if !bbs[7].Rendered {
						return bbs[7].RenderPS(ps360)
					}
					return result
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d434 := ps.PhiValues[0]
							ctx.EnsureDesc(&d434)
							ctx.EmitStoreToStack(d434, int32(bbs[10].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d435 := ps.PhiValues[1]
							ctx.EnsureDesc(&d435)
							ctx.EmitStoreToStack(d435, int32(bbs[10].PhiBase)+int32(16))
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
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
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
						d434 = ps.OverlayValues[434]
					}
					if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
						d435 = ps.OverlayValues[435]
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
					var d436 JITValueDesc
					if d10.SliceSizeKnown {
						d436 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.KnownSliceLen))}
					} else if d10.Loc == LocImm {
						d436 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.StackOff))}
					} else if d10.Loc == LocStackTriple {
						d436 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d10.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d10)
						if d10.Loc == LocRegPair || d10.Loc == LocRegTriple {
							d436 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg2, ID: 0}
						} else if d10.Loc == LocReg {
							d436 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d436)
					ctx.EnsureDescsTogether(&d8, &d436)
					var d437 JITValueDesc
					if d8.Loc == LocImm && d436.Loc == LocImm {
						d437 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < d436.Imm.Int())}
					} else if d436.Loc == LocImm {
						r16 := ctx.AllocRegExcept(d8.Reg)
						if d436.Imm.Int() >= -2147483648 && d436.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d8.Reg, int32(d436.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d436.Imm.Int()))
							ctx.EmitCmpInt64(d8.Reg, RegR11)
						}
						d437 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
						ctx.BindReg(r16, &d437)
					} else if d8.Loc == LocImm {
						r17 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d436.Reg)
						d437 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedLess}
						ctx.BindReg(r17, &d437)
					} else {
						r18 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpInt64(d8.Reg, d436.Reg)
						d437 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedLess}
						ctx.BindReg(r18, &d437)
					}
					ctx.FreeDesc(&d436)
					d438 = d437
					ctx.EnsureDesc(&d438)
					if d438.Loc != LocImm && d438.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d438.Loc == LocImm {
						if d438.Imm.Bool() {
							if ps.General {
							}
							ps439 := PhiState{General: ps.General}
							ps439.OverlayValues = make([]JITValueDesc, 439)
							ps439.OverlayValues[1] = d1
							ps439.OverlayValues[2] = d2
							ps439.OverlayValues[3] = d3
							ps439.OverlayValues[4] = d4
							ps439.OverlayValues[5] = d5
							ps439.OverlayValues[6] = d6
							ps439.OverlayValues[7] = d7
							ps439.OverlayValues[8] = d8
							ps439.OverlayValues[9] = d9
							ps439.OverlayValues[10] = d10
							ps439.OverlayValues[11] = d11
							ps439.OverlayValues[12] = d12
							ps439.OverlayValues[13] = d13
							ps439.OverlayValues[14] = d14
							ps439.OverlayValues[15] = d15
							ps439.OverlayValues[18] = d18
							ps439.OverlayValues[38] = d38
							ps439.OverlayValues[57] = d57
							ps439.OverlayValues[58] = d58
							ps439.OverlayValues[59] = d59
							ps439.OverlayValues[60] = d60
							ps439.OverlayValues[61] = d61
							ps439.OverlayValues[63] = d63
							ps439.OverlayValues[64] = d64
							ps439.OverlayValues[65] = d65
							ps439.OverlayValues[66] = d66
							ps439.OverlayValues[67] = d67
							ps439.OverlayValues[68] = d68
							ps439.OverlayValues[69] = d69
							ps439.OverlayValues[72] = d72
							ps439.OverlayValues[138] = d138
							ps439.OverlayValues[139] = d139
							ps439.OverlayValues[140] = d140
							ps439.OverlayValues[141] = d141
							ps439.OverlayValues[142] = d142
							ps439.OverlayValues[143] = d143
							ps439.OverlayValues[145] = d145
							ps439.OverlayValues[146] = d146
							ps439.OverlayValues[147] = d147
							ps439.OverlayValues[148] = d148
							ps439.OverlayValues[149] = d149
							ps439.OverlayValues[150] = d150
							ps439.OverlayValues[151] = d151
							ps439.OverlayValues[152] = d152
							ps439.OverlayValues[153] = d153
							ps439.OverlayValues[156] = d156
							ps439.OverlayValues[157] = d157
							ps439.OverlayValues[158] = d158
							ps439.OverlayValues[159] = d159
							ps439.OverlayValues[262] = d262
							ps439.OverlayValues[263] = d263
							ps439.OverlayValues[264] = d264
							ps439.OverlayValues[265] = d265
							ps439.OverlayValues[266] = d266
							ps439.OverlayValues[267] = d267
							ps439.OverlayValues[268] = d268
							ps439.OverlayValues[269] = d269
							ps439.OverlayValues[270] = d270
							ps439.OverlayValues[271] = d271
							ps439.OverlayValues[272] = d272
							ps439.OverlayValues[273] = d273
							ps439.OverlayValues[274] = d274
							ps439.OverlayValues[275] = d275
							ps439.OverlayValues[276] = d276
							ps439.OverlayValues[278] = d278
							ps439.OverlayValues[279] = d279
							ps439.OverlayValues[280] = d280
							ps439.OverlayValues[281] = d281
							ps439.OverlayValues[283] = d283
							ps439.OverlayValues[284] = d284
							ps439.OverlayValues[285] = d285
							ps439.OverlayValues[434] = d434
							ps439.OverlayValues[435] = d435
							ps439.OverlayValues[436] = d436
							ps439.OverlayValues[437] = d437
							ps439.OverlayValues[438] = d438
							return bbs[13].RenderPS(ps439)
						}
						if ps.General {
						}
						ps440 := PhiState{General: ps.General}
						ps440.OverlayValues = make([]JITValueDesc, 439)
						ps440.OverlayValues[1] = d1
						ps440.OverlayValues[2] = d2
						ps440.OverlayValues[3] = d3
						ps440.OverlayValues[4] = d4
						ps440.OverlayValues[5] = d5
						ps440.OverlayValues[6] = d6
						ps440.OverlayValues[7] = d7
						ps440.OverlayValues[8] = d8
						ps440.OverlayValues[9] = d9
						ps440.OverlayValues[10] = d10
						ps440.OverlayValues[11] = d11
						ps440.OverlayValues[12] = d12
						ps440.OverlayValues[13] = d13
						ps440.OverlayValues[14] = d14
						ps440.OverlayValues[15] = d15
						ps440.OverlayValues[18] = d18
						ps440.OverlayValues[38] = d38
						ps440.OverlayValues[57] = d57
						ps440.OverlayValues[58] = d58
						ps440.OverlayValues[59] = d59
						ps440.OverlayValues[60] = d60
						ps440.OverlayValues[61] = d61
						ps440.OverlayValues[63] = d63
						ps440.OverlayValues[64] = d64
						ps440.OverlayValues[65] = d65
						ps440.OverlayValues[66] = d66
						ps440.OverlayValues[67] = d67
						ps440.OverlayValues[68] = d68
						ps440.OverlayValues[69] = d69
						ps440.OverlayValues[72] = d72
						ps440.OverlayValues[138] = d138
						ps440.OverlayValues[139] = d139
						ps440.OverlayValues[140] = d140
						ps440.OverlayValues[141] = d141
						ps440.OverlayValues[142] = d142
						ps440.OverlayValues[143] = d143
						ps440.OverlayValues[145] = d145
						ps440.OverlayValues[146] = d146
						ps440.OverlayValues[147] = d147
						ps440.OverlayValues[148] = d148
						ps440.OverlayValues[149] = d149
						ps440.OverlayValues[150] = d150
						ps440.OverlayValues[151] = d151
						ps440.OverlayValues[152] = d152
						ps440.OverlayValues[153] = d153
						ps440.OverlayValues[156] = d156
						ps440.OverlayValues[157] = d157
						ps440.OverlayValues[158] = d158
						ps440.OverlayValues[159] = d159
						ps440.OverlayValues[262] = d262
						ps440.OverlayValues[263] = d263
						ps440.OverlayValues[264] = d264
						ps440.OverlayValues[265] = d265
						ps440.OverlayValues[266] = d266
						ps440.OverlayValues[267] = d267
						ps440.OverlayValues[268] = d268
						ps440.OverlayValues[269] = d269
						ps440.OverlayValues[270] = d270
						ps440.OverlayValues[271] = d271
						ps440.OverlayValues[272] = d272
						ps440.OverlayValues[273] = d273
						ps440.OverlayValues[274] = d274
						ps440.OverlayValues[275] = d275
						ps440.OverlayValues[276] = d276
						ps440.OverlayValues[278] = d278
						ps440.OverlayValues[279] = d279
						ps440.OverlayValues[280] = d280
						ps440.OverlayValues[281] = d281
						ps440.OverlayValues[283] = d283
						ps440.OverlayValues[284] = d284
						ps440.OverlayValues[285] = d285
						ps440.OverlayValues[434] = d434
						ps440.OverlayValues[435] = d435
						ps440.OverlayValues[436] = d436
						ps440.OverlayValues[437] = d437
						ps440.OverlayValues[438] = d438
						return bbs[12].RenderPS(ps440)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d441 := ps.PhiValues[0]
							ctx.EnsureDesc(&d441)
							ctx.EmitStoreToStack(d441, int32(bbs[10].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d442 := ps.PhiValues[1]
							ctx.EnsureDesc(&d442)
							ctx.EmitStoreToStack(d442, int32(bbs[10].PhiBase)+int32(16))
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					ctx.EmitJump(d438.Condition, lbl14)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d437)
					snap443 := d1
					snap444 := d2
					snap445 := d3
					snap446 := d4
					snap447 := d5
					snap448 := d6
					snap449 := d7
					snap450 := d8
					snap451 := d9
					snap452 := d10
					snap453 := d11
					snap454 := d12
					snap455 := d13
					snap456 := d14
					snap457 := d15
					snap458 := d18
					snap459 := d38
					snap460 := d57
					snap461 := d58
					snap462 := d59
					snap463 := d60
					snap464 := d61
					snap465 := d63
					snap466 := d64
					snap467 := d65
					snap468 := d66
					snap469 := d67
					snap470 := d68
					snap471 := d69
					snap472 := d72
					snap473 := d138
					snap474 := d139
					snap475 := d140
					snap476 := d141
					snap477 := d142
					snap478 := d143
					snap479 := d145
					snap480 := d146
					snap481 := d147
					snap482 := d148
					snap483 := d149
					snap484 := d150
					snap485 := d151
					snap486 := d152
					snap487 := d153
					snap488 := d156
					snap489 := d157
					snap490 := d158
					snap491 := d159
					snap492 := d262
					snap493 := d263
					snap494 := d264
					snap495 := d265
					snap496 := d266
					snap497 := d267
					snap498 := d268
					snap499 := d269
					snap500 := d270
					snap501 := d271
					snap502 := d272
					snap503 := d273
					snap504 := d274
					snap505 := d275
					snap506 := d276
					snap507 := d278
					snap508 := d279
					snap509 := d280
					snap510 := d281
					snap511 := d283
					snap512 := d284
					snap513 := d285
					snap514 := d434
					snap515 := d435
					snap516 := d436
					snap517 := d437
					snap518 := d438
					snap519 := d441
					snap520 := d442
					alloc521 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc521)
					d1 = snap443
					d2 = snap444
					d3 = snap445
					d4 = snap446
					d5 = snap447
					d6 = snap448
					d7 = snap449
					d8 = snap450
					d9 = snap451
					d10 = snap452
					d11 = snap453
					d12 = snap454
					d13 = snap455
					d14 = snap456
					d15 = snap457
					d18 = snap458
					d38 = snap459
					d57 = snap460
					d58 = snap461
					d59 = snap462
					d60 = snap463
					d61 = snap464
					d63 = snap465
					d64 = snap466
					d65 = snap467
					d66 = snap468
					d67 = snap469
					d68 = snap470
					d69 = snap471
					d72 = snap472
					d138 = snap473
					d139 = snap474
					d140 = snap475
					d141 = snap476
					d142 = snap477
					d143 = snap478
					d145 = snap479
					d146 = snap480
					d147 = snap481
					d148 = snap482
					d149 = snap483
					d150 = snap484
					d151 = snap485
					d152 = snap486
					d153 = snap487
					d156 = snap488
					d157 = snap489
					d158 = snap490
					d159 = snap491
					d262 = snap492
					d263 = snap493
					d264 = snap494
					d265 = snap495
					d266 = snap496
					d267 = snap497
					d268 = snap498
					d269 = snap499
					d270 = snap500
					d271 = snap501
					d272 = snap502
					d273 = snap503
					d274 = snap504
					d275 = snap505
					d276 = snap506
					d278 = snap507
					d279 = snap508
					d280 = snap509
					d281 = snap510
					d283 = snap511
					d284 = snap512
					d285 = snap513
					d434 = snap514
					d435 = snap515
					d436 = snap516
					d437 = snap517
					d438 = snap518
					d441 = snap519
					d442 = snap520
					ctx.RestoreAllocState(alloc521)
					d1 = snap443
					d2 = snap444
					d3 = snap445
					d4 = snap446
					d5 = snap447
					d6 = snap448
					d7 = snap449
					d8 = snap450
					d9 = snap451
					d10 = snap452
					d11 = snap453
					d12 = snap454
					d13 = snap455
					d14 = snap456
					d15 = snap457
					d18 = snap458
					d38 = snap459
					d57 = snap460
					d58 = snap461
					d59 = snap462
					d60 = snap463
					d61 = snap464
					d63 = snap465
					d64 = snap466
					d65 = snap467
					d66 = snap468
					d67 = snap469
					d68 = snap470
					d69 = snap471
					d72 = snap472
					d138 = snap473
					d139 = snap474
					d140 = snap475
					d141 = snap476
					d142 = snap477
					d143 = snap478
					d145 = snap479
					d146 = snap480
					d147 = snap481
					d148 = snap482
					d149 = snap483
					d150 = snap484
					d151 = snap485
					d152 = snap486
					d153 = snap487
					d156 = snap488
					d157 = snap489
					d158 = snap490
					d159 = snap491
					d262 = snap492
					d263 = snap493
					d264 = snap494
					d265 = snap495
					d266 = snap496
					d267 = snap497
					d268 = snap498
					d269 = snap499
					d270 = snap500
					d271 = snap501
					d272 = snap502
					d273 = snap503
					d274 = snap504
					d275 = snap505
					d276 = snap506
					d278 = snap507
					d279 = snap508
					d280 = snap509
					d281 = snap510
					d283 = snap511
					d284 = snap512
					d285 = snap513
					d434 = snap514
					d435 = snap515
					d436 = snap516
					d437 = snap517
					d438 = snap518
					d441 = snap519
					d442 = snap520
					ps522 := PhiState{General: true}
					ps522.OverlayValues = make([]JITValueDesc, 443)
					ps522.OverlayValues[1] = d1
					ps522.OverlayValues[2] = d2
					ps522.OverlayValues[3] = d3
					ps522.OverlayValues[4] = d4
					ps522.OverlayValues[5] = d5
					ps522.OverlayValues[6] = d6
					ps522.OverlayValues[7] = d7
					ps522.OverlayValues[8] = d8
					ps522.OverlayValues[9] = d9
					ps522.OverlayValues[10] = d10
					ps522.OverlayValues[11] = d11
					ps522.OverlayValues[12] = d12
					ps522.OverlayValues[13] = d13
					ps522.OverlayValues[14] = d14
					ps522.OverlayValues[15] = d15
					ps522.OverlayValues[18] = d18
					ps522.OverlayValues[38] = d38
					ps522.OverlayValues[57] = d57
					ps522.OverlayValues[58] = d58
					ps522.OverlayValues[59] = d59
					ps522.OverlayValues[60] = d60
					ps522.OverlayValues[61] = d61
					ps522.OverlayValues[63] = d63
					ps522.OverlayValues[64] = d64
					ps522.OverlayValues[65] = d65
					ps522.OverlayValues[66] = d66
					ps522.OverlayValues[67] = d67
					ps522.OverlayValues[68] = d68
					ps522.OverlayValues[69] = d69
					ps522.OverlayValues[72] = d72
					ps522.OverlayValues[138] = d138
					ps522.OverlayValues[139] = d139
					ps522.OverlayValues[140] = d140
					ps522.OverlayValues[141] = d141
					ps522.OverlayValues[142] = d142
					ps522.OverlayValues[143] = d143
					ps522.OverlayValues[145] = d145
					ps522.OverlayValues[146] = d146
					ps522.OverlayValues[147] = d147
					ps522.OverlayValues[148] = d148
					ps522.OverlayValues[149] = d149
					ps522.OverlayValues[150] = d150
					ps522.OverlayValues[151] = d151
					ps522.OverlayValues[152] = d152
					ps522.OverlayValues[153] = d153
					ps522.OverlayValues[156] = d156
					ps522.OverlayValues[157] = d157
					ps522.OverlayValues[158] = d158
					ps522.OverlayValues[159] = d159
					ps522.OverlayValues[262] = d262
					ps522.OverlayValues[263] = d263
					ps522.OverlayValues[264] = d264
					ps522.OverlayValues[265] = d265
					ps522.OverlayValues[266] = d266
					ps522.OverlayValues[267] = d267
					ps522.OverlayValues[268] = d268
					ps522.OverlayValues[269] = d269
					ps522.OverlayValues[270] = d270
					ps522.OverlayValues[271] = d271
					ps522.OverlayValues[272] = d272
					ps522.OverlayValues[273] = d273
					ps522.OverlayValues[274] = d274
					ps522.OverlayValues[275] = d275
					ps522.OverlayValues[276] = d276
					ps522.OverlayValues[278] = d278
					ps522.OverlayValues[279] = d279
					ps522.OverlayValues[280] = d280
					ps522.OverlayValues[281] = d281
					ps522.OverlayValues[283] = d283
					ps522.OverlayValues[284] = d284
					ps522.OverlayValues[285] = d285
					ps522.OverlayValues[434] = d434
					ps522.OverlayValues[435] = d435
					ps522.OverlayValues[436] = d436
					ps522.OverlayValues[437] = d437
					ps522.OverlayValues[438] = d438
					ps522.OverlayValues[441] = d441
					ps522.OverlayValues[442] = d442
					ps523 := PhiState{General: true}
					ps523.OverlayValues = make([]JITValueDesc, 443)
					ps523.OverlayValues[1] = d1
					ps523.OverlayValues[2] = d2
					ps523.OverlayValues[3] = d3
					ps523.OverlayValues[4] = d4
					ps523.OverlayValues[5] = d5
					ps523.OverlayValues[6] = d6
					ps523.OverlayValues[7] = d7
					ps523.OverlayValues[8] = d8
					ps523.OverlayValues[9] = d9
					ps523.OverlayValues[10] = d10
					ps523.OverlayValues[11] = d11
					ps523.OverlayValues[12] = d12
					ps523.OverlayValues[13] = d13
					ps523.OverlayValues[14] = d14
					ps523.OverlayValues[15] = d15
					ps523.OverlayValues[18] = d18
					ps523.OverlayValues[38] = d38
					ps523.OverlayValues[57] = d57
					ps523.OverlayValues[58] = d58
					ps523.OverlayValues[59] = d59
					ps523.OverlayValues[60] = d60
					ps523.OverlayValues[61] = d61
					ps523.OverlayValues[63] = d63
					ps523.OverlayValues[64] = d64
					ps523.OverlayValues[65] = d65
					ps523.OverlayValues[66] = d66
					ps523.OverlayValues[67] = d67
					ps523.OverlayValues[68] = d68
					ps523.OverlayValues[69] = d69
					ps523.OverlayValues[72] = d72
					ps523.OverlayValues[138] = d138
					ps523.OverlayValues[139] = d139
					ps523.OverlayValues[140] = d140
					ps523.OverlayValues[141] = d141
					ps523.OverlayValues[142] = d142
					ps523.OverlayValues[143] = d143
					ps523.OverlayValues[145] = d145
					ps523.OverlayValues[146] = d146
					ps523.OverlayValues[147] = d147
					ps523.OverlayValues[148] = d148
					ps523.OverlayValues[149] = d149
					ps523.OverlayValues[150] = d150
					ps523.OverlayValues[151] = d151
					ps523.OverlayValues[152] = d152
					ps523.OverlayValues[153] = d153
					ps523.OverlayValues[156] = d156
					ps523.OverlayValues[157] = d157
					ps523.OverlayValues[158] = d158
					ps523.OverlayValues[159] = d159
					ps523.OverlayValues[262] = d262
					ps523.OverlayValues[263] = d263
					ps523.OverlayValues[264] = d264
					ps523.OverlayValues[265] = d265
					ps523.OverlayValues[266] = d266
					ps523.OverlayValues[267] = d267
					ps523.OverlayValues[268] = d268
					ps523.OverlayValues[269] = d269
					ps523.OverlayValues[270] = d270
					ps523.OverlayValues[271] = d271
					ps523.OverlayValues[272] = d272
					ps523.OverlayValues[273] = d273
					ps523.OverlayValues[274] = d274
					ps523.OverlayValues[275] = d275
					ps523.OverlayValues[276] = d276
					ps523.OverlayValues[278] = d278
					ps523.OverlayValues[279] = d279
					ps523.OverlayValues[280] = d280
					ps523.OverlayValues[281] = d281
					ps523.OverlayValues[283] = d283
					ps523.OverlayValues[284] = d284
					ps523.OverlayValues[285] = d285
					ps523.OverlayValues[434] = d434
					ps523.OverlayValues[435] = d435
					ps523.OverlayValues[436] = d436
					ps523.OverlayValues[437] = d437
					ps523.OverlayValues[438] = d438
					ps523.OverlayValues[441] = d441
					ps523.OverlayValues[442] = d442
					snap524 := d1
					snap525 := d2
					snap526 := d3
					snap527 := d4
					snap528 := d5
					snap529 := d6
					snap530 := d7
					snap531 := d8
					snap532 := d9
					snap533 := d10
					snap534 := d11
					snap535 := d12
					snap536 := d13
					snap537 := d14
					snap538 := d15
					snap539 := d18
					snap540 := d38
					snap541 := d57
					snap542 := d58
					snap543 := d59
					snap544 := d60
					snap545 := d61
					snap546 := d63
					snap547 := d64
					snap548 := d65
					snap549 := d66
					snap550 := d67
					snap551 := d68
					snap552 := d69
					snap553 := d72
					snap554 := d138
					snap555 := d139
					snap556 := d140
					snap557 := d141
					snap558 := d142
					snap559 := d143
					snap560 := d145
					snap561 := d146
					snap562 := d147
					snap563 := d148
					snap564 := d149
					snap565 := d150
					snap566 := d151
					snap567 := d152
					snap568 := d153
					snap569 := d156
					snap570 := d157
					snap571 := d158
					snap572 := d159
					snap573 := d262
					snap574 := d263
					snap575 := d264
					snap576 := d265
					snap577 := d266
					snap578 := d267
					snap579 := d268
					snap580 := d269
					snap581 := d270
					snap582 := d271
					snap583 := d272
					snap584 := d273
					snap585 := d274
					snap586 := d275
					snap587 := d276
					snap588 := d278
					snap589 := d279
					snap590 := d280
					snap591 := d281
					snap592 := d283
					snap593 := d284
					snap594 := d285
					snap595 := d434
					snap596 := d435
					snap597 := d436
					snap598 := d437
					snap599 := d438
					snap600 := d441
					snap601 := d442
					alloc602 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps523)
					}
					ctx.RestoreAllocState(alloc602)
					d1 = snap524
					d2 = snap525
					d3 = snap526
					d4 = snap527
					d5 = snap528
					d6 = snap529
					d7 = snap530
					d8 = snap531
					d9 = snap532
					d10 = snap533
					d11 = snap534
					d12 = snap535
					d13 = snap536
					d14 = snap537
					d15 = snap538
					d18 = snap539
					d38 = snap540
					d57 = snap541
					d58 = snap542
					d59 = snap543
					d60 = snap544
					d61 = snap545
					d63 = snap546
					d64 = snap547
					d65 = snap548
					d66 = snap549
					d67 = snap550
					d68 = snap551
					d69 = snap552
					d72 = snap553
					d138 = snap554
					d139 = snap555
					d140 = snap556
					d141 = snap557
					d142 = snap558
					d143 = snap559
					d145 = snap560
					d146 = snap561
					d147 = snap562
					d148 = snap563
					d149 = snap564
					d150 = snap565
					d151 = snap566
					d152 = snap567
					d153 = snap568
					d156 = snap569
					d157 = snap570
					d158 = snap571
					d159 = snap572
					d262 = snap573
					d263 = snap574
					d264 = snap575
					d265 = snap576
					d266 = snap577
					d267 = snap578
					d268 = snap579
					d269 = snap580
					d270 = snap581
					d271 = snap582
					d272 = snap583
					d273 = snap584
					d274 = snap585
					d275 = snap586
					d276 = snap587
					d278 = snap588
					d279 = snap589
					d280 = snap590
					d281 = snap591
					d283 = snap592
					d284 = snap593
					d285 = snap594
					d434 = snap595
					d435 = snap596
					d436 = snap597
					d437 = snap598
					d438 = snap599
					d441 = snap600
					d442 = snap601
					if !bbs[13].Rendered {
						return bbs[13].RenderPS(ps522)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
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
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
						d434 = ps.OverlayValues[434]
					}
					if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
						d435 = ps.OverlayValues[435]
					}
					if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != LocNone {
						d436 = ps.OverlayValues[436]
					}
					if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != LocNone {
						d437 = ps.OverlayValues[437]
					}
					if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != LocNone {
						d438 = ps.OverlayValues[438]
					}
					if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != LocNone {
						d441 = ps.OverlayValues[441]
					}
					if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
						d442 = ps.OverlayValues[442]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					d604 = ctx.EmitSliceElementAddress(&d10, &d8, 16)
					ctx.EnsureDesc(&d604)
					r19 := ctx.AllocRegExcept(d604.Reg)
					ctx.EmitMovRegMem(r19, d604.Reg, 8)
					ctx.EmitMovRegMem(d604.Reg, d604.Reg, 0)
					d603 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d604.Reg, Reg2: r19}
					ctx.BindReg(d604.Reg, &d603)
					ctx.BindReg(r19, &d603)
					ctx.EnsureDesc(&d603)
					d605 = d603
					_ = d605
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl19 := ctx.ReserveLabel()
					_ = lbl19
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d606 JITValueDesc
					if d605.Loc == LocImm {
						d606 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d605.Imm.Float())}
					} else if d605.Type == tagFloat && d605.Loc == LocReg {
						d606 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d605.Reg}
						ctx.BindReg(d605.Reg, &d606)
						ctx.BindReg(d605.Reg, &d606)
					} else if d605.Type == tagFloat && d605.Loc == LocRegPair {
						ctx.FreeReg(d605.Reg)
						d606 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d605.Reg2}
						ctx.BindReg(d605.Reg2, &d606)
						ctx.BindReg(d605.Reg2, &d606)
					} else {
						d606 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d605}, 1)
						d606.Type = tagFloat
						ctx.BindReg(d606.Reg, &d606)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d606)
					ctx.FreeDesc(&d603)
					ctx.EnsureDesc(&d8)
					d608 = ctx.EmitSliceElementAddress(&d12, &d8, 16)
					ctx.EnsureDesc(&d608)
					r20 := ctx.AllocRegExcept(d608.Reg)
					ctx.EmitMovRegMem(r20, d608.Reg, 8)
					ctx.EmitMovRegMem(d608.Reg, d608.Reg, 0)
					d607 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d608.Reg, Reg2: r20}
					ctx.BindReg(d608.Reg, &d607)
					ctx.BindReg(r20, &d607)
					ctx.EnsureDesc(&d607)
					d609 = d607
					_ = d609
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl20 := ctx.ReserveLabel()
					_ = lbl20
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d610 JITValueDesc
					if d609.Loc == LocImm {
						d610 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d609.Imm.Float())}
					} else if d609.Type == tagFloat && d609.Loc == LocReg {
						d610 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d609.Reg}
						ctx.BindReg(d609.Reg, &d610)
						ctx.BindReg(d609.Reg, &d610)
					} else if d609.Type == tagFloat && d609.Loc == LocRegPair {
						ctx.FreeReg(d609.Reg)
						d610 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d609.Reg2}
						ctx.BindReg(d609.Reg2, &d610)
						ctx.BindReg(d609.Reg2, &d610)
					} else {
						d610 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d609}, 1)
						d610.Type = tagFloat
						ctx.BindReg(d610.Reg, &d610)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d610)
					ctx.FreeDesc(&d607)
					ctx.EnsureDesc(&d606)
					ctx.EnsureDesc(&d610)
					ctx.EnsureDescsTogether(&d606, &d610)
					var d611 JITValueDesc
					if d606.Loc == LocImm && d610.Loc == LocImm {
						d611 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d606.Imm.Float() * d610.Imm.Float())}
					} else if d606.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d610.Reg)
						_, xBits := d606.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitMulFloat64(scratch, d610.Reg)
						d611 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d611)
					} else if d610.Loc == LocImm {
						_, yBits := d610.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitMulFloat64(d606.Reg, RegR11)
						d611 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d606.Reg}
						ctx.BindReg(d606.Reg, &d611)
					} else {
						ctx.EmitMulFloat64(d606.Reg, d610.Reg)
						d611 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d606.Reg}
						ctx.BindReg(d606.Reg, &d611)
					}
					if d611.Loc == LocReg && d606.Loc == LocReg && d611.Reg == d606.Reg {
						ctx.TransferReg(d606.Reg)
						d606.Loc = LocNone
					}
					ctx.FreeDesc(&d606)
					ctx.FreeDesc(&d610)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d611)
					ctx.EnsureDescsTogether(&d7, &d611)
					var d612 JITValueDesc
					if d7.Loc == LocImm && d611.Loc == LocImm {
						d612 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d7.Imm.Float() + d611.Imm.Float())}
					} else if d7.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d611.Reg)
						_, xBits := d7.Imm.RawWords()
						ctx.EmitMovRegImm64(scratch, xBits)
						ctx.EmitAddFloat64(scratch, d611.Reg)
						d612 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d612)
					} else if d611.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d7.Reg)
						ctx.EmitMovRegReg(scratch, d7.Reg)
						_, yBits := d611.Imm.RawWords()
						ctx.EmitMovRegImm64(RegR11, yBits)
						ctx.EmitAddFloat64(scratch, RegR11)
						d612 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
						ctx.BindReg(scratch, &d612)
					} else {
						r21 := ctx.AllocRegExcept(d7.Reg, d611.Reg)
						ctx.EmitMovRegReg(r21, d7.Reg)
						ctx.EmitAddFloat64(r21, d611.Reg)
						d612 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r21}
						ctx.BindReg(r21, &d612)
					}
					if d612.Loc == LocReg && d7.Loc == LocReg && d612.Reg == d7.Reg {
						ctx.TransferReg(d7.Reg)
						d7.Loc = LocNone
					}
					ctx.EnsureDesc(&d612)
					ctx.EmitStoreToStack(d612, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d612)
					ctx.FreeDesc(&d611)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d8)
					var d613 JITValueDesc
					if d8.Loc == LocImm {
						d613 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d8.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitMovRegReg(scratch, d8.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d613 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d613)
					}
					if d613.Loc == LocReg && d8.Loc == LocReg && d613.Reg == d8.Reg {
						ctx.TransferReg(d8.Reg)
						d8.Loc = LocNone
					}
					ctx.EnsureDesc(&d613)
					ctx.EmitStoreToStack(d613, int32(bbs[10].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d613)
					if ps.General {
					}
					ps614 := PhiState{General: ps.General}
					ps614.OverlayValues = make([]JITValueDesc, 614)
					ps614.OverlayValues[1] = d1
					ps614.OverlayValues[2] = d2
					ps614.OverlayValues[3] = d3
					ps614.OverlayValues[4] = d4
					ps614.OverlayValues[5] = d5
					ps614.OverlayValues[6] = d6
					ps614.OverlayValues[7] = d7
					ps614.OverlayValues[8] = d8
					ps614.OverlayValues[9] = d9
					ps614.OverlayValues[10] = d10
					ps614.OverlayValues[11] = d11
					ps614.OverlayValues[12] = d12
					ps614.OverlayValues[13] = d13
					ps614.OverlayValues[14] = d14
					ps614.OverlayValues[15] = d15
					ps614.OverlayValues[18] = d18
					ps614.OverlayValues[38] = d38
					ps614.OverlayValues[57] = d57
					ps614.OverlayValues[58] = d58
					ps614.OverlayValues[59] = d59
					ps614.OverlayValues[60] = d60
					ps614.OverlayValues[61] = d61
					ps614.OverlayValues[63] = d63
					ps614.OverlayValues[64] = d64
					ps614.OverlayValues[65] = d65
					ps614.OverlayValues[66] = d66
					ps614.OverlayValues[67] = d67
					ps614.OverlayValues[68] = d68
					ps614.OverlayValues[69] = d69
					ps614.OverlayValues[72] = d72
					ps614.OverlayValues[138] = d138
					ps614.OverlayValues[139] = d139
					ps614.OverlayValues[140] = d140
					ps614.OverlayValues[141] = d141
					ps614.OverlayValues[142] = d142
					ps614.OverlayValues[143] = d143
					ps614.OverlayValues[145] = d145
					ps614.OverlayValues[146] = d146
					ps614.OverlayValues[147] = d147
					ps614.OverlayValues[148] = d148
					ps614.OverlayValues[149] = d149
					ps614.OverlayValues[150] = d150
					ps614.OverlayValues[151] = d151
					ps614.OverlayValues[152] = d152
					ps614.OverlayValues[153] = d153
					ps614.OverlayValues[156] = d156
					ps614.OverlayValues[157] = d157
					ps614.OverlayValues[158] = d158
					ps614.OverlayValues[159] = d159
					ps614.OverlayValues[262] = d262
					ps614.OverlayValues[263] = d263
					ps614.OverlayValues[264] = d264
					ps614.OverlayValues[265] = d265
					ps614.OverlayValues[266] = d266
					ps614.OverlayValues[267] = d267
					ps614.OverlayValues[268] = d268
					ps614.OverlayValues[269] = d269
					ps614.OverlayValues[270] = d270
					ps614.OverlayValues[271] = d271
					ps614.OverlayValues[272] = d272
					ps614.OverlayValues[273] = d273
					ps614.OverlayValues[274] = d274
					ps614.OverlayValues[275] = d275
					ps614.OverlayValues[276] = d276
					ps614.OverlayValues[278] = d278
					ps614.OverlayValues[279] = d279
					ps614.OverlayValues[280] = d280
					ps614.OverlayValues[281] = d281
					ps614.OverlayValues[283] = d283
					ps614.OverlayValues[284] = d284
					ps614.OverlayValues[285] = d285
					ps614.OverlayValues[434] = d434
					ps614.OverlayValues[435] = d435
					ps614.OverlayValues[436] = d436
					ps614.OverlayValues[437] = d437
					ps614.OverlayValues[438] = d438
					ps614.OverlayValues[441] = d441
					ps614.OverlayValues[442] = d442
					ps614.OverlayValues[603] = d603
					ps614.OverlayValues[604] = d604
					ps614.OverlayValues[605] = d605
					ps614.OverlayValues[606] = d606
					ps614.OverlayValues[607] = d607
					ps614.OverlayValues[608] = d608
					ps614.OverlayValues[609] = d609
					ps614.OverlayValues[610] = d610
					ps614.OverlayValues[611] = d611
					ps614.OverlayValues[612] = d612
					ps614.OverlayValues[613] = d613
					ps614.PhiValues = make([]JITValueDesc, 2)
					if ps614.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps614)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
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
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
						d434 = ps.OverlayValues[434]
					}
					if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
						d435 = ps.OverlayValues[435]
					}
					if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != LocNone {
						d436 = ps.OverlayValues[436]
					}
					if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != LocNone {
						d437 = ps.OverlayValues[437]
					}
					if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != LocNone {
						d438 = ps.OverlayValues[438]
					}
					if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != LocNone {
						d441 = ps.OverlayValues[441]
					}
					if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
						d442 = ps.OverlayValues[442]
					}
					if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != LocNone {
						d603 = ps.OverlayValues[603]
					}
					if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != LocNone {
						d604 = ps.OverlayValues[604]
					}
					if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != LocNone {
						d605 = ps.OverlayValues[605]
					}
					if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != LocNone {
						d606 = ps.OverlayValues[606]
					}
					if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != LocNone {
						d607 = ps.OverlayValues[607]
					}
					if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != LocNone {
						d608 = ps.OverlayValues[608]
					}
					if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != LocNone {
						d609 = ps.OverlayValues[609]
					}
					if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != LocNone {
						d610 = ps.OverlayValues[610]
					}
					if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != LocNone {
						d611 = ps.OverlayValues[611]
					}
					if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != LocNone {
						d612 = ps.OverlayValues[612]
					}
					if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != LocNone {
						d613 = ps.OverlayValues[613]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					var d615 JITValueDesc
					if d1.Loc == LocImm {
						ctx.TrackImm(d1.Imm)
						ptrWord, _ := d1.Imm.RawWords()
						d615 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d615.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d615.Reg2, uint64(len(d1.Imm.String())))
						ctx.BindReg(d615.Reg, &d615)
						ctx.BindReg(d615.Reg2, &d615)
					} else {
						d615 = d1
					}
					d616 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("EUCLIDEAN")}
					var d617 JITValueDesc
					if d616.Loc == LocImm {
						ctx.TrackImm(d616.Imm)
						ptrWord, _ := d616.Imm.RawWords()
						d617 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d617.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d617.Reg2, uint64(len(d616.Imm.String())))
						ctx.BindReg(d617.Reg, &d617)
						ctx.BindReg(d617.Reg2, &d617)
					} else {
						d617 = d616
					}
					d618 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d615, d617}, 1)
					ctx.EmitAndRegImm32(d618.Reg, 1)
					d618.Type = tagBool
					ctx.BindReg(d618.Reg, &d618)
					d619 = d618
					ctx.EnsureDesc(&d619)
					if d619.Loc != LocImm && d619.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d619.Loc == LocImm {
						if d619.Imm.Bool() {
							if ps.General {
							}
							ps620 := PhiState{General: ps.General}
							ps620.OverlayValues = make([]JITValueDesc, 620)
							ps620.OverlayValues[1] = d1
							ps620.OverlayValues[2] = d2
							ps620.OverlayValues[3] = d3
							ps620.OverlayValues[4] = d4
							ps620.OverlayValues[5] = d5
							ps620.OverlayValues[6] = d6
							ps620.OverlayValues[7] = d7
							ps620.OverlayValues[8] = d8
							ps620.OverlayValues[9] = d9
							ps620.OverlayValues[10] = d10
							ps620.OverlayValues[11] = d11
							ps620.OverlayValues[12] = d12
							ps620.OverlayValues[13] = d13
							ps620.OverlayValues[14] = d14
							ps620.OverlayValues[15] = d15
							ps620.OverlayValues[18] = d18
							ps620.OverlayValues[38] = d38
							ps620.OverlayValues[57] = d57
							ps620.OverlayValues[58] = d58
							ps620.OverlayValues[59] = d59
							ps620.OverlayValues[60] = d60
							ps620.OverlayValues[61] = d61
							ps620.OverlayValues[63] = d63
							ps620.OverlayValues[64] = d64
							ps620.OverlayValues[65] = d65
							ps620.OverlayValues[66] = d66
							ps620.OverlayValues[67] = d67
							ps620.OverlayValues[68] = d68
							ps620.OverlayValues[69] = d69
							ps620.OverlayValues[72] = d72
							ps620.OverlayValues[138] = d138
							ps620.OverlayValues[139] = d139
							ps620.OverlayValues[140] = d140
							ps620.OverlayValues[141] = d141
							ps620.OverlayValues[142] = d142
							ps620.OverlayValues[143] = d143
							ps620.OverlayValues[145] = d145
							ps620.OverlayValues[146] = d146
							ps620.OverlayValues[147] = d147
							ps620.OverlayValues[148] = d148
							ps620.OverlayValues[149] = d149
							ps620.OverlayValues[150] = d150
							ps620.OverlayValues[151] = d151
							ps620.OverlayValues[152] = d152
							ps620.OverlayValues[153] = d153
							ps620.OverlayValues[156] = d156
							ps620.OverlayValues[157] = d157
							ps620.OverlayValues[158] = d158
							ps620.OverlayValues[159] = d159
							ps620.OverlayValues[262] = d262
							ps620.OverlayValues[263] = d263
							ps620.OverlayValues[264] = d264
							ps620.OverlayValues[265] = d265
							ps620.OverlayValues[266] = d266
							ps620.OverlayValues[267] = d267
							ps620.OverlayValues[268] = d268
							ps620.OverlayValues[269] = d269
							ps620.OverlayValues[270] = d270
							ps620.OverlayValues[271] = d271
							ps620.OverlayValues[272] = d272
							ps620.OverlayValues[273] = d273
							ps620.OverlayValues[274] = d274
							ps620.OverlayValues[275] = d275
							ps620.OverlayValues[276] = d276
							ps620.OverlayValues[278] = d278
							ps620.OverlayValues[279] = d279
							ps620.OverlayValues[280] = d280
							ps620.OverlayValues[281] = d281
							ps620.OverlayValues[283] = d283
							ps620.OverlayValues[284] = d284
							ps620.OverlayValues[285] = d285
							ps620.OverlayValues[434] = d434
							ps620.OverlayValues[435] = d435
							ps620.OverlayValues[436] = d436
							ps620.OverlayValues[437] = d437
							ps620.OverlayValues[438] = d438
							ps620.OverlayValues[441] = d441
							ps620.OverlayValues[442] = d442
							ps620.OverlayValues[603] = d603
							ps620.OverlayValues[604] = d604
							ps620.OverlayValues[605] = d605
							ps620.OverlayValues[606] = d606
							ps620.OverlayValues[607] = d607
							ps620.OverlayValues[608] = d608
							ps620.OverlayValues[609] = d609
							ps620.OverlayValues[610] = d610
							ps620.OverlayValues[611] = d611
							ps620.OverlayValues[612] = d612
							ps620.OverlayValues[613] = d613
							ps620.OverlayValues[615] = d615
							ps620.OverlayValues[616] = d616
							ps620.OverlayValues[617] = d617
							ps620.OverlayValues[618] = d618
							ps620.OverlayValues[619] = d619
							return bbs[14].RenderPS(ps620)
						}
						if ps.General {
							ctx.SyncDesc(&d7)
							if d7.Loc == LocReg {
								ctx.ProtectReg(d7.Reg)
							} else if d7.Loc == LocRegPair {
								ctx.ProtectReg(d7.Reg)
								ctx.ProtectReg(d7.Reg2)
							}
							d621 = d7
							if d621.Loc == LocNone {
								panic("jit: phi source has no location")
							}
							ctx.EnsureDesc(&d621)
							ctx.EmitStoreToStack(d621, int32(bbs[4].PhiBase)+int32(0))
							if d7.Loc == LocReg {
								ctx.UnprotectReg(d7.Reg)
							} else if d7.Loc == LocRegPair {
								ctx.UnprotectReg(d7.Reg)
								ctx.UnprotectReg(d7.Reg2)
							}
						}
						ps622 := PhiState{General: ps.General}
						ps622.OverlayValues = make([]JITValueDesc, 622)
						ps622.OverlayValues[1] = d1
						ps622.OverlayValues[2] = d2
						ps622.OverlayValues[3] = d3
						ps622.OverlayValues[4] = d4
						ps622.OverlayValues[5] = d5
						ps622.OverlayValues[6] = d6
						ps622.OverlayValues[7] = d7
						ps622.OverlayValues[8] = d8
						ps622.OverlayValues[9] = d9
						ps622.OverlayValues[10] = d10
						ps622.OverlayValues[11] = d11
						ps622.OverlayValues[12] = d12
						ps622.OverlayValues[13] = d13
						ps622.OverlayValues[14] = d14
						ps622.OverlayValues[15] = d15
						ps622.OverlayValues[18] = d18
						ps622.OverlayValues[38] = d38
						ps622.OverlayValues[57] = d57
						ps622.OverlayValues[58] = d58
						ps622.OverlayValues[59] = d59
						ps622.OverlayValues[60] = d60
						ps622.OverlayValues[61] = d61
						ps622.OverlayValues[63] = d63
						ps622.OverlayValues[64] = d64
						ps622.OverlayValues[65] = d65
						ps622.OverlayValues[66] = d66
						ps622.OverlayValues[67] = d67
						ps622.OverlayValues[68] = d68
						ps622.OverlayValues[69] = d69
						ps622.OverlayValues[72] = d72
						ps622.OverlayValues[138] = d138
						ps622.OverlayValues[139] = d139
						ps622.OverlayValues[140] = d140
						ps622.OverlayValues[141] = d141
						ps622.OverlayValues[142] = d142
						ps622.OverlayValues[143] = d143
						ps622.OverlayValues[145] = d145
						ps622.OverlayValues[146] = d146
						ps622.OverlayValues[147] = d147
						ps622.OverlayValues[148] = d148
						ps622.OverlayValues[149] = d149
						ps622.OverlayValues[150] = d150
						ps622.OverlayValues[151] = d151
						ps622.OverlayValues[152] = d152
						ps622.OverlayValues[153] = d153
						ps622.OverlayValues[156] = d156
						ps622.OverlayValues[157] = d157
						ps622.OverlayValues[158] = d158
						ps622.OverlayValues[159] = d159
						ps622.OverlayValues[262] = d262
						ps622.OverlayValues[263] = d263
						ps622.OverlayValues[264] = d264
						ps622.OverlayValues[265] = d265
						ps622.OverlayValues[266] = d266
						ps622.OverlayValues[267] = d267
						ps622.OverlayValues[268] = d268
						ps622.OverlayValues[269] = d269
						ps622.OverlayValues[270] = d270
						ps622.OverlayValues[271] = d271
						ps622.OverlayValues[272] = d272
						ps622.OverlayValues[273] = d273
						ps622.OverlayValues[274] = d274
						ps622.OverlayValues[275] = d275
						ps622.OverlayValues[276] = d276
						ps622.OverlayValues[278] = d278
						ps622.OverlayValues[279] = d279
						ps622.OverlayValues[280] = d280
						ps622.OverlayValues[281] = d281
						ps622.OverlayValues[283] = d283
						ps622.OverlayValues[284] = d284
						ps622.OverlayValues[285] = d285
						ps622.OverlayValues[434] = d434
						ps622.OverlayValues[435] = d435
						ps622.OverlayValues[436] = d436
						ps622.OverlayValues[437] = d437
						ps622.OverlayValues[438] = d438
						ps622.OverlayValues[441] = d441
						ps622.OverlayValues[442] = d442
						ps622.OverlayValues[603] = d603
						ps622.OverlayValues[604] = d604
						ps622.OverlayValues[605] = d605
						ps622.OverlayValues[606] = d606
						ps622.OverlayValues[607] = d607
						ps622.OverlayValues[608] = d608
						ps622.OverlayValues[609] = d609
						ps622.OverlayValues[610] = d610
						ps622.OverlayValues[611] = d611
						ps622.OverlayValues[612] = d612
						ps622.OverlayValues[613] = d613
						ps622.OverlayValues[615] = d615
						ps622.OverlayValues[616] = d616
						ps622.OverlayValues[617] = d617
						ps622.OverlayValues[618] = d618
						ps622.OverlayValues[619] = d619
						ps622.OverlayValues[621] = d621
						ps622.PhiValues = make([]JITValueDesc, 1)
						d623 = d7
						ps622.PhiValues[0] = d623
						return bbs[4].RenderPS(ps622)
					}
					if !ps.General {
						ps.General = true
						return bbs[12].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d619.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl21)
					snap624 := d1
					snap625 := d2
					snap626 := d3
					snap627 := d4
					snap628 := d5
					snap629 := d6
					snap630 := d7
					snap631 := d8
					snap632 := d9
					snap633 := d10
					snap634 := d11
					snap635 := d12
					snap636 := d13
					snap637 := d14
					snap638 := d15
					snap639 := d18
					snap640 := d38
					snap641 := d57
					snap642 := d58
					snap643 := d59
					snap644 := d60
					snap645 := d61
					snap646 := d63
					snap647 := d64
					snap648 := d65
					snap649 := d66
					snap650 := d67
					snap651 := d68
					snap652 := d69
					snap653 := d72
					snap654 := d138
					snap655 := d139
					snap656 := d140
					snap657 := d141
					snap658 := d142
					snap659 := d143
					snap660 := d145
					snap661 := d146
					snap662 := d147
					snap663 := d148
					snap664 := d149
					snap665 := d150
					snap666 := d151
					snap667 := d152
					snap668 := d153
					snap669 := d156
					snap670 := d157
					snap671 := d158
					snap672 := d159
					snap673 := d262
					snap674 := d263
					snap675 := d264
					snap676 := d265
					snap677 := d266
					snap678 := d267
					snap679 := d268
					snap680 := d269
					snap681 := d270
					snap682 := d271
					snap683 := d272
					snap684 := d273
					snap685 := d274
					snap686 := d275
					snap687 := d276
					snap688 := d278
					snap689 := d279
					snap690 := d280
					snap691 := d281
					snap692 := d283
					snap693 := d284
					snap694 := d285
					snap695 := d434
					snap696 := d435
					snap697 := d436
					snap698 := d437
					snap699 := d438
					snap700 := d441
					snap701 := d442
					snap702 := d603
					snap703 := d604
					snap704 := d605
					snap705 := d606
					snap706 := d607
					snap707 := d608
					snap708 := d609
					snap709 := d610
					snap710 := d611
					snap711 := d612
					snap712 := d613
					snap713 := d615
					snap714 := d616
					snap715 := d617
					snap716 := d618
					snap717 := d619
					snap718 := d621
					snap719 := d623
					alloc720 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc720)
					d1 = snap624
					d2 = snap625
					d3 = snap626
					d4 = snap627
					d5 = snap628
					d6 = snap629
					d7 = snap630
					d8 = snap631
					d9 = snap632
					d10 = snap633
					d11 = snap634
					d12 = snap635
					d13 = snap636
					d14 = snap637
					d15 = snap638
					d18 = snap639
					d38 = snap640
					d57 = snap641
					d58 = snap642
					d59 = snap643
					d60 = snap644
					d61 = snap645
					d63 = snap646
					d64 = snap647
					d65 = snap648
					d66 = snap649
					d67 = snap650
					d68 = snap651
					d69 = snap652
					d72 = snap653
					d138 = snap654
					d139 = snap655
					d140 = snap656
					d141 = snap657
					d142 = snap658
					d143 = snap659
					d145 = snap660
					d146 = snap661
					d147 = snap662
					d148 = snap663
					d149 = snap664
					d150 = snap665
					d151 = snap666
					d152 = snap667
					d153 = snap668
					d156 = snap669
					d157 = snap670
					d158 = snap671
					d159 = snap672
					d262 = snap673
					d263 = snap674
					d264 = snap675
					d265 = snap676
					d266 = snap677
					d267 = snap678
					d268 = snap679
					d269 = snap680
					d270 = snap681
					d271 = snap682
					d272 = snap683
					d273 = snap684
					d274 = snap685
					d275 = snap686
					d276 = snap687
					d278 = snap688
					d279 = snap689
					d280 = snap690
					d281 = snap691
					d283 = snap692
					d284 = snap693
					d285 = snap694
					d434 = snap695
					d435 = snap696
					d436 = snap697
					d437 = snap698
					d438 = snap699
					d441 = snap700
					d442 = snap701
					d603 = snap702
					d604 = snap703
					d605 = snap704
					d606 = snap705
					d607 = snap706
					d608 = snap707
					d609 = snap708
					d610 = snap709
					d611 = snap710
					d612 = snap711
					d613 = snap712
					d615 = snap713
					d616 = snap714
					d617 = snap715
					d618 = snap716
					d619 = snap717
					d621 = snap718
					d623 = snap719
					ctx.MarkLabel(lbl21)
					ctx.SyncDesc(&d7)
					if d7.Loc == LocReg {
						ctx.ProtectReg(d7.Reg)
					} else if d7.Loc == LocRegPair {
						ctx.ProtectReg(d7.Reg)
						ctx.ProtectReg(d7.Reg2)
					}
					d721 = d7
					if d721.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d721)
					ctx.EmitStoreToStack(d721, int32(bbs[4].PhiBase)+int32(0))
					if d7.Loc == LocReg {
						ctx.UnprotectReg(d7.Reg)
					} else if d7.Loc == LocRegPair {
						ctx.UnprotectReg(d7.Reg)
						ctx.UnprotectReg(d7.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc720)
					d1 = snap624
					d2 = snap625
					d3 = snap626
					d4 = snap627
					d5 = snap628
					d6 = snap629
					d7 = snap630
					d8 = snap631
					d9 = snap632
					d10 = snap633
					d11 = snap634
					d12 = snap635
					d13 = snap636
					d14 = snap637
					d15 = snap638
					d18 = snap639
					d38 = snap640
					d57 = snap641
					d58 = snap642
					d59 = snap643
					d60 = snap644
					d61 = snap645
					d63 = snap646
					d64 = snap647
					d65 = snap648
					d66 = snap649
					d67 = snap650
					d68 = snap651
					d69 = snap652
					d72 = snap653
					d138 = snap654
					d139 = snap655
					d140 = snap656
					d141 = snap657
					d142 = snap658
					d143 = snap659
					d145 = snap660
					d146 = snap661
					d147 = snap662
					d148 = snap663
					d149 = snap664
					d150 = snap665
					d151 = snap666
					d152 = snap667
					d153 = snap668
					d156 = snap669
					d157 = snap670
					d158 = snap671
					d159 = snap672
					d262 = snap673
					d263 = snap674
					d264 = snap675
					d265 = snap676
					d266 = snap677
					d267 = snap678
					d268 = snap679
					d269 = snap680
					d270 = snap681
					d271 = snap682
					d272 = snap683
					d273 = snap684
					d274 = snap685
					d275 = snap686
					d276 = snap687
					d278 = snap688
					d279 = snap689
					d280 = snap690
					d281 = snap691
					d283 = snap692
					d284 = snap693
					d285 = snap694
					d434 = snap695
					d435 = snap696
					d436 = snap697
					d437 = snap698
					d438 = snap699
					d441 = snap700
					d442 = snap701
					d603 = snap702
					d604 = snap703
					d605 = snap704
					d606 = snap705
					d607 = snap706
					d608 = snap707
					d609 = snap708
					d610 = snap709
					d611 = snap710
					d612 = snap711
					d613 = snap712
					d615 = snap713
					d616 = snap714
					d617 = snap715
					d618 = snap716
					d619 = snap717
					d621 = snap718
					d623 = snap719
					ps722 := PhiState{General: true}
					ps722.OverlayValues = make([]JITValueDesc, 722)
					ps722.OverlayValues[1] = d1
					ps722.OverlayValues[2] = d2
					ps722.OverlayValues[3] = d3
					ps722.OverlayValues[4] = d4
					ps722.OverlayValues[5] = d5
					ps722.OverlayValues[6] = d6
					ps722.OverlayValues[7] = d7
					ps722.OverlayValues[8] = d8
					ps722.OverlayValues[9] = d9
					ps722.OverlayValues[10] = d10
					ps722.OverlayValues[11] = d11
					ps722.OverlayValues[12] = d12
					ps722.OverlayValues[13] = d13
					ps722.OverlayValues[14] = d14
					ps722.OverlayValues[15] = d15
					ps722.OverlayValues[18] = d18
					ps722.OverlayValues[38] = d38
					ps722.OverlayValues[57] = d57
					ps722.OverlayValues[58] = d58
					ps722.OverlayValues[59] = d59
					ps722.OverlayValues[60] = d60
					ps722.OverlayValues[61] = d61
					ps722.OverlayValues[63] = d63
					ps722.OverlayValues[64] = d64
					ps722.OverlayValues[65] = d65
					ps722.OverlayValues[66] = d66
					ps722.OverlayValues[67] = d67
					ps722.OverlayValues[68] = d68
					ps722.OverlayValues[69] = d69
					ps722.OverlayValues[72] = d72
					ps722.OverlayValues[138] = d138
					ps722.OverlayValues[139] = d139
					ps722.OverlayValues[140] = d140
					ps722.OverlayValues[141] = d141
					ps722.OverlayValues[142] = d142
					ps722.OverlayValues[143] = d143
					ps722.OverlayValues[145] = d145
					ps722.OverlayValues[146] = d146
					ps722.OverlayValues[147] = d147
					ps722.OverlayValues[148] = d148
					ps722.OverlayValues[149] = d149
					ps722.OverlayValues[150] = d150
					ps722.OverlayValues[151] = d151
					ps722.OverlayValues[152] = d152
					ps722.OverlayValues[153] = d153
					ps722.OverlayValues[156] = d156
					ps722.OverlayValues[157] = d157
					ps722.OverlayValues[158] = d158
					ps722.OverlayValues[159] = d159
					ps722.OverlayValues[262] = d262
					ps722.OverlayValues[263] = d263
					ps722.OverlayValues[264] = d264
					ps722.OverlayValues[265] = d265
					ps722.OverlayValues[266] = d266
					ps722.OverlayValues[267] = d267
					ps722.OverlayValues[268] = d268
					ps722.OverlayValues[269] = d269
					ps722.OverlayValues[270] = d270
					ps722.OverlayValues[271] = d271
					ps722.OverlayValues[272] = d272
					ps722.OverlayValues[273] = d273
					ps722.OverlayValues[274] = d274
					ps722.OverlayValues[275] = d275
					ps722.OverlayValues[276] = d276
					ps722.OverlayValues[278] = d278
					ps722.OverlayValues[279] = d279
					ps722.OverlayValues[280] = d280
					ps722.OverlayValues[281] = d281
					ps722.OverlayValues[283] = d283
					ps722.OverlayValues[284] = d284
					ps722.OverlayValues[285] = d285
					ps722.OverlayValues[434] = d434
					ps722.OverlayValues[435] = d435
					ps722.OverlayValues[436] = d436
					ps722.OverlayValues[437] = d437
					ps722.OverlayValues[438] = d438
					ps722.OverlayValues[441] = d441
					ps722.OverlayValues[442] = d442
					ps722.OverlayValues[603] = d603
					ps722.OverlayValues[604] = d604
					ps722.OverlayValues[605] = d605
					ps722.OverlayValues[606] = d606
					ps722.OverlayValues[607] = d607
					ps722.OverlayValues[608] = d608
					ps722.OverlayValues[609] = d609
					ps722.OverlayValues[610] = d610
					ps722.OverlayValues[611] = d611
					ps722.OverlayValues[612] = d612
					ps722.OverlayValues[613] = d613
					ps722.OverlayValues[615] = d615
					ps722.OverlayValues[616] = d616
					ps722.OverlayValues[617] = d617
					ps722.OverlayValues[618] = d618
					ps722.OverlayValues[619] = d619
					ps722.OverlayValues[621] = d621
					ps722.OverlayValues[623] = d623
					ps722.OverlayValues[721] = d721
					ps723 := PhiState{General: true}
					ps723.OverlayValues = make([]JITValueDesc, 722)
					ps723.OverlayValues[1] = d1
					ps723.OverlayValues[2] = d2
					ps723.OverlayValues[3] = d3
					ps723.OverlayValues[4] = d4
					ps723.OverlayValues[5] = d5
					ps723.OverlayValues[6] = d6
					ps723.OverlayValues[7] = d7
					ps723.OverlayValues[8] = d8
					ps723.OverlayValues[9] = d9
					ps723.OverlayValues[10] = d10
					ps723.OverlayValues[11] = d11
					ps723.OverlayValues[12] = d12
					ps723.OverlayValues[13] = d13
					ps723.OverlayValues[14] = d14
					ps723.OverlayValues[15] = d15
					ps723.OverlayValues[18] = d18
					ps723.OverlayValues[38] = d38
					ps723.OverlayValues[57] = d57
					ps723.OverlayValues[58] = d58
					ps723.OverlayValues[59] = d59
					ps723.OverlayValues[60] = d60
					ps723.OverlayValues[61] = d61
					ps723.OverlayValues[63] = d63
					ps723.OverlayValues[64] = d64
					ps723.OverlayValues[65] = d65
					ps723.OverlayValues[66] = d66
					ps723.OverlayValues[67] = d67
					ps723.OverlayValues[68] = d68
					ps723.OverlayValues[69] = d69
					ps723.OverlayValues[72] = d72
					ps723.OverlayValues[138] = d138
					ps723.OverlayValues[139] = d139
					ps723.OverlayValues[140] = d140
					ps723.OverlayValues[141] = d141
					ps723.OverlayValues[142] = d142
					ps723.OverlayValues[143] = d143
					ps723.OverlayValues[145] = d145
					ps723.OverlayValues[146] = d146
					ps723.OverlayValues[147] = d147
					ps723.OverlayValues[148] = d148
					ps723.OverlayValues[149] = d149
					ps723.OverlayValues[150] = d150
					ps723.OverlayValues[151] = d151
					ps723.OverlayValues[152] = d152
					ps723.OverlayValues[153] = d153
					ps723.OverlayValues[156] = d156
					ps723.OverlayValues[157] = d157
					ps723.OverlayValues[158] = d158
					ps723.OverlayValues[159] = d159
					ps723.OverlayValues[262] = d262
					ps723.OverlayValues[263] = d263
					ps723.OverlayValues[264] = d264
					ps723.OverlayValues[265] = d265
					ps723.OverlayValues[266] = d266
					ps723.OverlayValues[267] = d267
					ps723.OverlayValues[268] = d268
					ps723.OverlayValues[269] = d269
					ps723.OverlayValues[270] = d270
					ps723.OverlayValues[271] = d271
					ps723.OverlayValues[272] = d272
					ps723.OverlayValues[273] = d273
					ps723.OverlayValues[274] = d274
					ps723.OverlayValues[275] = d275
					ps723.OverlayValues[276] = d276
					ps723.OverlayValues[278] = d278
					ps723.OverlayValues[279] = d279
					ps723.OverlayValues[280] = d280
					ps723.OverlayValues[281] = d281
					ps723.OverlayValues[283] = d283
					ps723.OverlayValues[284] = d284
					ps723.OverlayValues[285] = d285
					ps723.OverlayValues[434] = d434
					ps723.OverlayValues[435] = d435
					ps723.OverlayValues[436] = d436
					ps723.OverlayValues[437] = d437
					ps723.OverlayValues[438] = d438
					ps723.OverlayValues[441] = d441
					ps723.OverlayValues[442] = d442
					ps723.OverlayValues[603] = d603
					ps723.OverlayValues[604] = d604
					ps723.OverlayValues[605] = d605
					ps723.OverlayValues[606] = d606
					ps723.OverlayValues[607] = d607
					ps723.OverlayValues[608] = d608
					ps723.OverlayValues[609] = d609
					ps723.OverlayValues[610] = d610
					ps723.OverlayValues[611] = d611
					ps723.OverlayValues[612] = d612
					ps723.OverlayValues[613] = d613
					ps723.OverlayValues[615] = d615
					ps723.OverlayValues[616] = d616
					ps723.OverlayValues[617] = d617
					ps723.OverlayValues[618] = d618
					ps723.OverlayValues[619] = d619
					ps723.OverlayValues[621] = d621
					ps723.OverlayValues[623] = d623
					ps723.OverlayValues[721] = d721
					ps723.PhiValues = make([]JITValueDesc, 1)
					d724 = d7
					ps723.PhiValues[0] = d724
					snap725 := d1
					snap726 := d2
					snap727 := d3
					snap728 := d4
					snap729 := d5
					snap730 := d6
					snap731 := d7
					snap732 := d8
					snap733 := d9
					snap734 := d10
					snap735 := d11
					snap736 := d12
					snap737 := d13
					snap738 := d14
					snap739 := d15
					snap740 := d18
					snap741 := d38
					snap742 := d57
					snap743 := d58
					snap744 := d59
					snap745 := d60
					snap746 := d61
					snap747 := d63
					snap748 := d64
					snap749 := d65
					snap750 := d66
					snap751 := d67
					snap752 := d68
					snap753 := d69
					snap754 := d72
					snap755 := d138
					snap756 := d139
					snap757 := d140
					snap758 := d141
					snap759 := d142
					snap760 := d143
					snap761 := d145
					snap762 := d146
					snap763 := d147
					snap764 := d148
					snap765 := d149
					snap766 := d150
					snap767 := d151
					snap768 := d152
					snap769 := d153
					snap770 := d156
					snap771 := d157
					snap772 := d158
					snap773 := d159
					snap774 := d262
					snap775 := d263
					snap776 := d264
					snap777 := d265
					snap778 := d266
					snap779 := d267
					snap780 := d268
					snap781 := d269
					snap782 := d270
					snap783 := d271
					snap784 := d272
					snap785 := d273
					snap786 := d274
					snap787 := d275
					snap788 := d276
					snap789 := d278
					snap790 := d279
					snap791 := d280
					snap792 := d281
					snap793 := d283
					snap794 := d284
					snap795 := d285
					snap796 := d434
					snap797 := d435
					snap798 := d436
					snap799 := d437
					snap800 := d438
					snap801 := d441
					snap802 := d442
					snap803 := d603
					snap804 := d604
					snap805 := d605
					snap806 := d606
					snap807 := d607
					snap808 := d608
					snap809 := d609
					snap810 := d610
					snap811 := d611
					snap812 := d612
					snap813 := d613
					snap814 := d615
					snap815 := d616
					snap816 := d617
					snap817 := d618
					snap818 := d619
					snap819 := d621
					snap820 := d623
					snap821 := d721
					snap822 := d724
					alloc823 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps723)
					}
					ctx.RestoreAllocState(alloc823)
					d1 = snap725
					d2 = snap726
					d3 = snap727
					d4 = snap728
					d5 = snap729
					d6 = snap730
					d7 = snap731
					d8 = snap732
					d9 = snap733
					d10 = snap734
					d11 = snap735
					d12 = snap736
					d13 = snap737
					d14 = snap738
					d15 = snap739
					d18 = snap740
					d38 = snap741
					d57 = snap742
					d58 = snap743
					d59 = snap744
					d60 = snap745
					d61 = snap746
					d63 = snap747
					d64 = snap748
					d65 = snap749
					d66 = snap750
					d67 = snap751
					d68 = snap752
					d69 = snap753
					d72 = snap754
					d138 = snap755
					d139 = snap756
					d140 = snap757
					d141 = snap758
					d142 = snap759
					d143 = snap760
					d145 = snap761
					d146 = snap762
					d147 = snap763
					d148 = snap764
					d149 = snap765
					d150 = snap766
					d151 = snap767
					d152 = snap768
					d153 = snap769
					d156 = snap770
					d157 = snap771
					d158 = snap772
					d159 = snap773
					d262 = snap774
					d263 = snap775
					d264 = snap776
					d265 = snap777
					d266 = snap778
					d267 = snap779
					d268 = snap780
					d269 = snap781
					d270 = snap782
					d271 = snap783
					d272 = snap784
					d273 = snap785
					d274 = snap786
					d275 = snap787
					d276 = snap788
					d278 = snap789
					d279 = snap790
					d280 = snap791
					d281 = snap792
					d283 = snap793
					d284 = snap794
					d285 = snap795
					d434 = snap796
					d435 = snap797
					d436 = snap798
					d437 = snap799
					d438 = snap800
					d441 = snap801
					d442 = snap802
					d603 = snap803
					d604 = snap804
					d605 = snap805
					d606 = snap806
					d607 = snap807
					d608 = snap808
					d609 = snap809
					d610 = snap810
					d611 = snap811
					d612 = snap812
					d613 = snap813
					d615 = snap814
					d616 = snap815
					d617 = snap816
					d618 = snap817
					d619 = snap818
					d621 = snap819
					d623 = snap820
					d721 = snap821
					d724 = snap822
					if !bbs[14].Rendered {
						return bbs[14].RenderPS(ps722)
					}
					return result
					ctx.FreeDesc(&d618)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
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
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
						d434 = ps.OverlayValues[434]
					}
					if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
						d435 = ps.OverlayValues[435]
					}
					if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != LocNone {
						d436 = ps.OverlayValues[436]
					}
					if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != LocNone {
						d437 = ps.OverlayValues[437]
					}
					if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != LocNone {
						d438 = ps.OverlayValues[438]
					}
					if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != LocNone {
						d441 = ps.OverlayValues[441]
					}
					if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
						d442 = ps.OverlayValues[442]
					}
					if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != LocNone {
						d603 = ps.OverlayValues[603]
					}
					if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != LocNone {
						d604 = ps.OverlayValues[604]
					}
					if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != LocNone {
						d605 = ps.OverlayValues[605]
					}
					if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != LocNone {
						d606 = ps.OverlayValues[606]
					}
					if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != LocNone {
						d607 = ps.OverlayValues[607]
					}
					if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != LocNone {
						d608 = ps.OverlayValues[608]
					}
					if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != LocNone {
						d609 = ps.OverlayValues[609]
					}
					if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != LocNone {
						d610 = ps.OverlayValues[610]
					}
					if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != LocNone {
						d611 = ps.OverlayValues[611]
					}
					if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != LocNone {
						d612 = ps.OverlayValues[612]
					}
					if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != LocNone {
						d613 = ps.OverlayValues[613]
					}
					if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != LocNone {
						d615 = ps.OverlayValues[615]
					}
					if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != LocNone {
						d616 = ps.OverlayValues[616]
					}
					if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != LocNone {
						d617 = ps.OverlayValues[617]
					}
					if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != LocNone {
						d618 = ps.OverlayValues[618]
					}
					if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != LocNone {
						d619 = ps.OverlayValues[619]
					}
					if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != LocNone {
						d621 = ps.OverlayValues[621]
					}
					if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != LocNone {
						d623 = ps.OverlayValues[623]
					}
					if len(ps.OverlayValues) > 721 && ps.OverlayValues[721].Loc != LocNone {
						d721 = ps.OverlayValues[721]
					}
					if len(ps.OverlayValues) > 724 && ps.OverlayValues[724].Loc != LocNone {
						d724 = ps.OverlayValues[724]
					}
					ctx.ReclaimUntrackedRegs()
					var d824 JITValueDesc
					if d12.SliceSizeKnown {
						d824 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.KnownSliceLen))}
					} else if d12.Loc == LocImm {
						d824 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d12.StackOff))}
					} else if d12.Loc == LocStackTriple {
						d824 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d12.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d12)
						if d12.Loc == LocRegPair || d12.Loc == LocRegTriple {
							d824 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg2, ID: 0}
						} else if d12.Loc == LocReg {
							d824 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d824)
					ctx.EnsureDescsTogether(&d8, &d824)
					var d825 JITValueDesc
					if d8.Loc == LocImm && d824.Loc == LocImm {
						d825 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < d824.Imm.Int())}
					} else if d824.Loc == LocImm {
						r22 := ctx.AllocRegExcept(d8.Reg)
						if d824.Imm.Int() >= -2147483648 && d824.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d8.Reg, int32(d824.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d824.Imm.Int()))
							ctx.EmitCmpInt64(d8.Reg, RegR11)
						}
						d825 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedLess}
						ctx.BindReg(r22, &d825)
					} else if d8.Loc == LocImm {
						r23 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d824.Reg)
						d825 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondSignedLess}
						ctx.BindReg(r23, &d825)
					} else {
						r24 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpInt64(d8.Reg, d824.Reg)
						d825 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedLess}
						ctx.BindReg(r24, &d825)
					}
					ctx.FreeDesc(&d824)
					d826 = d825
					ctx.EnsureDesc(&d826)
					if d826.Loc != LocImm && d826.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d826.Loc == LocImm {
						if d826.Imm.Bool() {
							if ps.General {
							}
							ps827 := PhiState{General: ps.General}
							ps827.OverlayValues = make([]JITValueDesc, 827)
							ps827.OverlayValues[1] = d1
							ps827.OverlayValues[2] = d2
							ps827.OverlayValues[3] = d3
							ps827.OverlayValues[4] = d4
							ps827.OverlayValues[5] = d5
							ps827.OverlayValues[6] = d6
							ps827.OverlayValues[7] = d7
							ps827.OverlayValues[8] = d8
							ps827.OverlayValues[9] = d9
							ps827.OverlayValues[10] = d10
							ps827.OverlayValues[11] = d11
							ps827.OverlayValues[12] = d12
							ps827.OverlayValues[13] = d13
							ps827.OverlayValues[14] = d14
							ps827.OverlayValues[15] = d15
							ps827.OverlayValues[18] = d18
							ps827.OverlayValues[38] = d38
							ps827.OverlayValues[57] = d57
							ps827.OverlayValues[58] = d58
							ps827.OverlayValues[59] = d59
							ps827.OverlayValues[60] = d60
							ps827.OverlayValues[61] = d61
							ps827.OverlayValues[63] = d63
							ps827.OverlayValues[64] = d64
							ps827.OverlayValues[65] = d65
							ps827.OverlayValues[66] = d66
							ps827.OverlayValues[67] = d67
							ps827.OverlayValues[68] = d68
							ps827.OverlayValues[69] = d69
							ps827.OverlayValues[72] = d72
							ps827.OverlayValues[138] = d138
							ps827.OverlayValues[139] = d139
							ps827.OverlayValues[140] = d140
							ps827.OverlayValues[141] = d141
							ps827.OverlayValues[142] = d142
							ps827.OverlayValues[143] = d143
							ps827.OverlayValues[145] = d145
							ps827.OverlayValues[146] = d146
							ps827.OverlayValues[147] = d147
							ps827.OverlayValues[148] = d148
							ps827.OverlayValues[149] = d149
							ps827.OverlayValues[150] = d150
							ps827.OverlayValues[151] = d151
							ps827.OverlayValues[152] = d152
							ps827.OverlayValues[153] = d153
							ps827.OverlayValues[156] = d156
							ps827.OverlayValues[157] = d157
							ps827.OverlayValues[158] = d158
							ps827.OverlayValues[159] = d159
							ps827.OverlayValues[262] = d262
							ps827.OverlayValues[263] = d263
							ps827.OverlayValues[264] = d264
							ps827.OverlayValues[265] = d265
							ps827.OverlayValues[266] = d266
							ps827.OverlayValues[267] = d267
							ps827.OverlayValues[268] = d268
							ps827.OverlayValues[269] = d269
							ps827.OverlayValues[270] = d270
							ps827.OverlayValues[271] = d271
							ps827.OverlayValues[272] = d272
							ps827.OverlayValues[273] = d273
							ps827.OverlayValues[274] = d274
							ps827.OverlayValues[275] = d275
							ps827.OverlayValues[276] = d276
							ps827.OverlayValues[278] = d278
							ps827.OverlayValues[279] = d279
							ps827.OverlayValues[280] = d280
							ps827.OverlayValues[281] = d281
							ps827.OverlayValues[283] = d283
							ps827.OverlayValues[284] = d284
							ps827.OverlayValues[285] = d285
							ps827.OverlayValues[434] = d434
							ps827.OverlayValues[435] = d435
							ps827.OverlayValues[436] = d436
							ps827.OverlayValues[437] = d437
							ps827.OverlayValues[438] = d438
							ps827.OverlayValues[441] = d441
							ps827.OverlayValues[442] = d442
							ps827.OverlayValues[603] = d603
							ps827.OverlayValues[604] = d604
							ps827.OverlayValues[605] = d605
							ps827.OverlayValues[606] = d606
							ps827.OverlayValues[607] = d607
							ps827.OverlayValues[608] = d608
							ps827.OverlayValues[609] = d609
							ps827.OverlayValues[610] = d610
							ps827.OverlayValues[611] = d611
							ps827.OverlayValues[612] = d612
							ps827.OverlayValues[613] = d613
							ps827.OverlayValues[615] = d615
							ps827.OverlayValues[616] = d616
							ps827.OverlayValues[617] = d617
							ps827.OverlayValues[618] = d618
							ps827.OverlayValues[619] = d619
							ps827.OverlayValues[621] = d621
							ps827.OverlayValues[623] = d623
							ps827.OverlayValues[721] = d721
							ps827.OverlayValues[724] = d724
							ps827.OverlayValues[824] = d824
							ps827.OverlayValues[825] = d825
							ps827.OverlayValues[826] = d826
							return bbs[11].RenderPS(ps827)
						}
						if ps.General {
						}
						ps828 := PhiState{General: ps.General}
						ps828.OverlayValues = make([]JITValueDesc, 827)
						ps828.OverlayValues[1] = d1
						ps828.OverlayValues[2] = d2
						ps828.OverlayValues[3] = d3
						ps828.OverlayValues[4] = d4
						ps828.OverlayValues[5] = d5
						ps828.OverlayValues[6] = d6
						ps828.OverlayValues[7] = d7
						ps828.OverlayValues[8] = d8
						ps828.OverlayValues[9] = d9
						ps828.OverlayValues[10] = d10
						ps828.OverlayValues[11] = d11
						ps828.OverlayValues[12] = d12
						ps828.OverlayValues[13] = d13
						ps828.OverlayValues[14] = d14
						ps828.OverlayValues[15] = d15
						ps828.OverlayValues[18] = d18
						ps828.OverlayValues[38] = d38
						ps828.OverlayValues[57] = d57
						ps828.OverlayValues[58] = d58
						ps828.OverlayValues[59] = d59
						ps828.OverlayValues[60] = d60
						ps828.OverlayValues[61] = d61
						ps828.OverlayValues[63] = d63
						ps828.OverlayValues[64] = d64
						ps828.OverlayValues[65] = d65
						ps828.OverlayValues[66] = d66
						ps828.OverlayValues[67] = d67
						ps828.OverlayValues[68] = d68
						ps828.OverlayValues[69] = d69
						ps828.OverlayValues[72] = d72
						ps828.OverlayValues[138] = d138
						ps828.OverlayValues[139] = d139
						ps828.OverlayValues[140] = d140
						ps828.OverlayValues[141] = d141
						ps828.OverlayValues[142] = d142
						ps828.OverlayValues[143] = d143
						ps828.OverlayValues[145] = d145
						ps828.OverlayValues[146] = d146
						ps828.OverlayValues[147] = d147
						ps828.OverlayValues[148] = d148
						ps828.OverlayValues[149] = d149
						ps828.OverlayValues[150] = d150
						ps828.OverlayValues[151] = d151
						ps828.OverlayValues[152] = d152
						ps828.OverlayValues[153] = d153
						ps828.OverlayValues[156] = d156
						ps828.OverlayValues[157] = d157
						ps828.OverlayValues[158] = d158
						ps828.OverlayValues[159] = d159
						ps828.OverlayValues[262] = d262
						ps828.OverlayValues[263] = d263
						ps828.OverlayValues[264] = d264
						ps828.OverlayValues[265] = d265
						ps828.OverlayValues[266] = d266
						ps828.OverlayValues[267] = d267
						ps828.OverlayValues[268] = d268
						ps828.OverlayValues[269] = d269
						ps828.OverlayValues[270] = d270
						ps828.OverlayValues[271] = d271
						ps828.OverlayValues[272] = d272
						ps828.OverlayValues[273] = d273
						ps828.OverlayValues[274] = d274
						ps828.OverlayValues[275] = d275
						ps828.OverlayValues[276] = d276
						ps828.OverlayValues[278] = d278
						ps828.OverlayValues[279] = d279
						ps828.OverlayValues[280] = d280
						ps828.OverlayValues[281] = d281
						ps828.OverlayValues[283] = d283
						ps828.OverlayValues[284] = d284
						ps828.OverlayValues[285] = d285
						ps828.OverlayValues[434] = d434
						ps828.OverlayValues[435] = d435
						ps828.OverlayValues[436] = d436
						ps828.OverlayValues[437] = d437
						ps828.OverlayValues[438] = d438
						ps828.OverlayValues[441] = d441
						ps828.OverlayValues[442] = d442
						ps828.OverlayValues[603] = d603
						ps828.OverlayValues[604] = d604
						ps828.OverlayValues[605] = d605
						ps828.OverlayValues[606] = d606
						ps828.OverlayValues[607] = d607
						ps828.OverlayValues[608] = d608
						ps828.OverlayValues[609] = d609
						ps828.OverlayValues[610] = d610
						ps828.OverlayValues[611] = d611
						ps828.OverlayValues[612] = d612
						ps828.OverlayValues[613] = d613
						ps828.OverlayValues[615] = d615
						ps828.OverlayValues[616] = d616
						ps828.OverlayValues[617] = d617
						ps828.OverlayValues[618] = d618
						ps828.OverlayValues[619] = d619
						ps828.OverlayValues[621] = d621
						ps828.OverlayValues[623] = d623
						ps828.OverlayValues[721] = d721
						ps828.OverlayValues[724] = d724
						ps828.OverlayValues[824] = d824
						ps828.OverlayValues[825] = d825
						ps828.OverlayValues[826] = d826
						return bbs[12].RenderPS(ps828)
					}
					if !ps.General {
						ps.General = true
						return bbs[13].RenderPS(ps)
					}
					ctx.EmitJump(d826.Condition, lbl12)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d825)
					snap829 := d1
					snap830 := d2
					snap831 := d3
					snap832 := d4
					snap833 := d5
					snap834 := d6
					snap835 := d7
					snap836 := d8
					snap837 := d9
					snap838 := d10
					snap839 := d11
					snap840 := d12
					snap841 := d13
					snap842 := d14
					snap843 := d15
					snap844 := d18
					snap845 := d38
					snap846 := d57
					snap847 := d58
					snap848 := d59
					snap849 := d60
					snap850 := d61
					snap851 := d63
					snap852 := d64
					snap853 := d65
					snap854 := d66
					snap855 := d67
					snap856 := d68
					snap857 := d69
					snap858 := d72
					snap859 := d138
					snap860 := d139
					snap861 := d140
					snap862 := d141
					snap863 := d142
					snap864 := d143
					snap865 := d145
					snap866 := d146
					snap867 := d147
					snap868 := d148
					snap869 := d149
					snap870 := d150
					snap871 := d151
					snap872 := d152
					snap873 := d153
					snap874 := d156
					snap875 := d157
					snap876 := d158
					snap877 := d159
					snap878 := d262
					snap879 := d263
					snap880 := d264
					snap881 := d265
					snap882 := d266
					snap883 := d267
					snap884 := d268
					snap885 := d269
					snap886 := d270
					snap887 := d271
					snap888 := d272
					snap889 := d273
					snap890 := d274
					snap891 := d275
					snap892 := d276
					snap893 := d278
					snap894 := d279
					snap895 := d280
					snap896 := d281
					snap897 := d283
					snap898 := d284
					snap899 := d285
					snap900 := d434
					snap901 := d435
					snap902 := d436
					snap903 := d437
					snap904 := d438
					snap905 := d441
					snap906 := d442
					snap907 := d603
					snap908 := d604
					snap909 := d605
					snap910 := d606
					snap911 := d607
					snap912 := d608
					snap913 := d609
					snap914 := d610
					snap915 := d611
					snap916 := d612
					snap917 := d613
					snap918 := d615
					snap919 := d616
					snap920 := d617
					snap921 := d618
					snap922 := d619
					snap923 := d621
					snap924 := d623
					snap925 := d721
					snap926 := d724
					snap927 := d824
					snap928 := d825
					snap929 := d826
					alloc930 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc930)
					d1 = snap829
					d2 = snap830
					d3 = snap831
					d4 = snap832
					d5 = snap833
					d6 = snap834
					d7 = snap835
					d8 = snap836
					d9 = snap837
					d10 = snap838
					d11 = snap839
					d12 = snap840
					d13 = snap841
					d14 = snap842
					d15 = snap843
					d18 = snap844
					d38 = snap845
					d57 = snap846
					d58 = snap847
					d59 = snap848
					d60 = snap849
					d61 = snap850
					d63 = snap851
					d64 = snap852
					d65 = snap853
					d66 = snap854
					d67 = snap855
					d68 = snap856
					d69 = snap857
					d72 = snap858
					d138 = snap859
					d139 = snap860
					d140 = snap861
					d141 = snap862
					d142 = snap863
					d143 = snap864
					d145 = snap865
					d146 = snap866
					d147 = snap867
					d148 = snap868
					d149 = snap869
					d150 = snap870
					d151 = snap871
					d152 = snap872
					d153 = snap873
					d156 = snap874
					d157 = snap875
					d158 = snap876
					d159 = snap877
					d262 = snap878
					d263 = snap879
					d264 = snap880
					d265 = snap881
					d266 = snap882
					d267 = snap883
					d268 = snap884
					d269 = snap885
					d270 = snap886
					d271 = snap887
					d272 = snap888
					d273 = snap889
					d274 = snap890
					d275 = snap891
					d276 = snap892
					d278 = snap893
					d279 = snap894
					d280 = snap895
					d281 = snap896
					d283 = snap897
					d284 = snap898
					d285 = snap899
					d434 = snap900
					d435 = snap901
					d436 = snap902
					d437 = snap903
					d438 = snap904
					d441 = snap905
					d442 = snap906
					d603 = snap907
					d604 = snap908
					d605 = snap909
					d606 = snap910
					d607 = snap911
					d608 = snap912
					d609 = snap913
					d610 = snap914
					d611 = snap915
					d612 = snap916
					d613 = snap917
					d615 = snap918
					d616 = snap919
					d617 = snap920
					d618 = snap921
					d619 = snap922
					d621 = snap923
					d623 = snap924
					d721 = snap925
					d724 = snap926
					d824 = snap927
					d825 = snap928
					d826 = snap929
					ctx.RestoreAllocState(alloc930)
					d1 = snap829
					d2 = snap830
					d3 = snap831
					d4 = snap832
					d5 = snap833
					d6 = snap834
					d7 = snap835
					d8 = snap836
					d9 = snap837
					d10 = snap838
					d11 = snap839
					d12 = snap840
					d13 = snap841
					d14 = snap842
					d15 = snap843
					d18 = snap844
					d38 = snap845
					d57 = snap846
					d58 = snap847
					d59 = snap848
					d60 = snap849
					d61 = snap850
					d63 = snap851
					d64 = snap852
					d65 = snap853
					d66 = snap854
					d67 = snap855
					d68 = snap856
					d69 = snap857
					d72 = snap858
					d138 = snap859
					d139 = snap860
					d140 = snap861
					d141 = snap862
					d142 = snap863
					d143 = snap864
					d145 = snap865
					d146 = snap866
					d147 = snap867
					d148 = snap868
					d149 = snap869
					d150 = snap870
					d151 = snap871
					d152 = snap872
					d153 = snap873
					d156 = snap874
					d157 = snap875
					d158 = snap876
					d159 = snap877
					d262 = snap878
					d263 = snap879
					d264 = snap880
					d265 = snap881
					d266 = snap882
					d267 = snap883
					d268 = snap884
					d269 = snap885
					d270 = snap886
					d271 = snap887
					d272 = snap888
					d273 = snap889
					d274 = snap890
					d275 = snap891
					d276 = snap892
					d278 = snap893
					d279 = snap894
					d280 = snap895
					d281 = snap896
					d283 = snap897
					d284 = snap898
					d285 = snap899
					d434 = snap900
					d435 = snap901
					d436 = snap902
					d437 = snap903
					d438 = snap904
					d441 = snap905
					d442 = snap906
					d603 = snap907
					d604 = snap908
					d605 = snap909
					d606 = snap910
					d607 = snap911
					d608 = snap912
					d609 = snap913
					d610 = snap914
					d611 = snap915
					d612 = snap916
					d613 = snap917
					d615 = snap918
					d616 = snap919
					d617 = snap920
					d618 = snap921
					d619 = snap922
					d621 = snap923
					d623 = snap924
					d721 = snap925
					d724 = snap926
					d824 = snap927
					d825 = snap928
					d826 = snap929
					ps931 := PhiState{General: true}
					ps931.OverlayValues = make([]JITValueDesc, 827)
					ps931.OverlayValues[1] = d1
					ps931.OverlayValues[2] = d2
					ps931.OverlayValues[3] = d3
					ps931.OverlayValues[4] = d4
					ps931.OverlayValues[5] = d5
					ps931.OverlayValues[6] = d6
					ps931.OverlayValues[7] = d7
					ps931.OverlayValues[8] = d8
					ps931.OverlayValues[9] = d9
					ps931.OverlayValues[10] = d10
					ps931.OverlayValues[11] = d11
					ps931.OverlayValues[12] = d12
					ps931.OverlayValues[13] = d13
					ps931.OverlayValues[14] = d14
					ps931.OverlayValues[15] = d15
					ps931.OverlayValues[18] = d18
					ps931.OverlayValues[38] = d38
					ps931.OverlayValues[57] = d57
					ps931.OverlayValues[58] = d58
					ps931.OverlayValues[59] = d59
					ps931.OverlayValues[60] = d60
					ps931.OverlayValues[61] = d61
					ps931.OverlayValues[63] = d63
					ps931.OverlayValues[64] = d64
					ps931.OverlayValues[65] = d65
					ps931.OverlayValues[66] = d66
					ps931.OverlayValues[67] = d67
					ps931.OverlayValues[68] = d68
					ps931.OverlayValues[69] = d69
					ps931.OverlayValues[72] = d72
					ps931.OverlayValues[138] = d138
					ps931.OverlayValues[139] = d139
					ps931.OverlayValues[140] = d140
					ps931.OverlayValues[141] = d141
					ps931.OverlayValues[142] = d142
					ps931.OverlayValues[143] = d143
					ps931.OverlayValues[145] = d145
					ps931.OverlayValues[146] = d146
					ps931.OverlayValues[147] = d147
					ps931.OverlayValues[148] = d148
					ps931.OverlayValues[149] = d149
					ps931.OverlayValues[150] = d150
					ps931.OverlayValues[151] = d151
					ps931.OverlayValues[152] = d152
					ps931.OverlayValues[153] = d153
					ps931.OverlayValues[156] = d156
					ps931.OverlayValues[157] = d157
					ps931.OverlayValues[158] = d158
					ps931.OverlayValues[159] = d159
					ps931.OverlayValues[262] = d262
					ps931.OverlayValues[263] = d263
					ps931.OverlayValues[264] = d264
					ps931.OverlayValues[265] = d265
					ps931.OverlayValues[266] = d266
					ps931.OverlayValues[267] = d267
					ps931.OverlayValues[268] = d268
					ps931.OverlayValues[269] = d269
					ps931.OverlayValues[270] = d270
					ps931.OverlayValues[271] = d271
					ps931.OverlayValues[272] = d272
					ps931.OverlayValues[273] = d273
					ps931.OverlayValues[274] = d274
					ps931.OverlayValues[275] = d275
					ps931.OverlayValues[276] = d276
					ps931.OverlayValues[278] = d278
					ps931.OverlayValues[279] = d279
					ps931.OverlayValues[280] = d280
					ps931.OverlayValues[281] = d281
					ps931.OverlayValues[283] = d283
					ps931.OverlayValues[284] = d284
					ps931.OverlayValues[285] = d285
					ps931.OverlayValues[434] = d434
					ps931.OverlayValues[435] = d435
					ps931.OverlayValues[436] = d436
					ps931.OverlayValues[437] = d437
					ps931.OverlayValues[438] = d438
					ps931.OverlayValues[441] = d441
					ps931.OverlayValues[442] = d442
					ps931.OverlayValues[603] = d603
					ps931.OverlayValues[604] = d604
					ps931.OverlayValues[605] = d605
					ps931.OverlayValues[606] = d606
					ps931.OverlayValues[607] = d607
					ps931.OverlayValues[608] = d608
					ps931.OverlayValues[609] = d609
					ps931.OverlayValues[610] = d610
					ps931.OverlayValues[611] = d611
					ps931.OverlayValues[612] = d612
					ps931.OverlayValues[613] = d613
					ps931.OverlayValues[615] = d615
					ps931.OverlayValues[616] = d616
					ps931.OverlayValues[617] = d617
					ps931.OverlayValues[618] = d618
					ps931.OverlayValues[619] = d619
					ps931.OverlayValues[621] = d621
					ps931.OverlayValues[623] = d623
					ps931.OverlayValues[721] = d721
					ps931.OverlayValues[724] = d724
					ps931.OverlayValues[824] = d824
					ps931.OverlayValues[825] = d825
					ps931.OverlayValues[826] = d826
					ps932 := PhiState{General: true}
					ps932.OverlayValues = make([]JITValueDesc, 827)
					ps932.OverlayValues[1] = d1
					ps932.OverlayValues[2] = d2
					ps932.OverlayValues[3] = d3
					ps932.OverlayValues[4] = d4
					ps932.OverlayValues[5] = d5
					ps932.OverlayValues[6] = d6
					ps932.OverlayValues[7] = d7
					ps932.OverlayValues[8] = d8
					ps932.OverlayValues[9] = d9
					ps932.OverlayValues[10] = d10
					ps932.OverlayValues[11] = d11
					ps932.OverlayValues[12] = d12
					ps932.OverlayValues[13] = d13
					ps932.OverlayValues[14] = d14
					ps932.OverlayValues[15] = d15
					ps932.OverlayValues[18] = d18
					ps932.OverlayValues[38] = d38
					ps932.OverlayValues[57] = d57
					ps932.OverlayValues[58] = d58
					ps932.OverlayValues[59] = d59
					ps932.OverlayValues[60] = d60
					ps932.OverlayValues[61] = d61
					ps932.OverlayValues[63] = d63
					ps932.OverlayValues[64] = d64
					ps932.OverlayValues[65] = d65
					ps932.OverlayValues[66] = d66
					ps932.OverlayValues[67] = d67
					ps932.OverlayValues[68] = d68
					ps932.OverlayValues[69] = d69
					ps932.OverlayValues[72] = d72
					ps932.OverlayValues[138] = d138
					ps932.OverlayValues[139] = d139
					ps932.OverlayValues[140] = d140
					ps932.OverlayValues[141] = d141
					ps932.OverlayValues[142] = d142
					ps932.OverlayValues[143] = d143
					ps932.OverlayValues[145] = d145
					ps932.OverlayValues[146] = d146
					ps932.OverlayValues[147] = d147
					ps932.OverlayValues[148] = d148
					ps932.OverlayValues[149] = d149
					ps932.OverlayValues[150] = d150
					ps932.OverlayValues[151] = d151
					ps932.OverlayValues[152] = d152
					ps932.OverlayValues[153] = d153
					ps932.OverlayValues[156] = d156
					ps932.OverlayValues[157] = d157
					ps932.OverlayValues[158] = d158
					ps932.OverlayValues[159] = d159
					ps932.OverlayValues[262] = d262
					ps932.OverlayValues[263] = d263
					ps932.OverlayValues[264] = d264
					ps932.OverlayValues[265] = d265
					ps932.OverlayValues[266] = d266
					ps932.OverlayValues[267] = d267
					ps932.OverlayValues[268] = d268
					ps932.OverlayValues[269] = d269
					ps932.OverlayValues[270] = d270
					ps932.OverlayValues[271] = d271
					ps932.OverlayValues[272] = d272
					ps932.OverlayValues[273] = d273
					ps932.OverlayValues[274] = d274
					ps932.OverlayValues[275] = d275
					ps932.OverlayValues[276] = d276
					ps932.OverlayValues[278] = d278
					ps932.OverlayValues[279] = d279
					ps932.OverlayValues[280] = d280
					ps932.OverlayValues[281] = d281
					ps932.OverlayValues[283] = d283
					ps932.OverlayValues[284] = d284
					ps932.OverlayValues[285] = d285
					ps932.OverlayValues[434] = d434
					ps932.OverlayValues[435] = d435
					ps932.OverlayValues[436] = d436
					ps932.OverlayValues[437] = d437
					ps932.OverlayValues[438] = d438
					ps932.OverlayValues[441] = d441
					ps932.OverlayValues[442] = d442
					ps932.OverlayValues[603] = d603
					ps932.OverlayValues[604] = d604
					ps932.OverlayValues[605] = d605
					ps932.OverlayValues[606] = d606
					ps932.OverlayValues[607] = d607
					ps932.OverlayValues[608] = d608
					ps932.OverlayValues[609] = d609
					ps932.OverlayValues[610] = d610
					ps932.OverlayValues[611] = d611
					ps932.OverlayValues[612] = d612
					ps932.OverlayValues[613] = d613
					ps932.OverlayValues[615] = d615
					ps932.OverlayValues[616] = d616
					ps932.OverlayValues[617] = d617
					ps932.OverlayValues[618] = d618
					ps932.OverlayValues[619] = d619
					ps932.OverlayValues[621] = d621
					ps932.OverlayValues[623] = d623
					ps932.OverlayValues[721] = d721
					ps932.OverlayValues[724] = d724
					ps932.OverlayValues[824] = d824
					ps932.OverlayValues[825] = d825
					ps932.OverlayValues[826] = d826
					snap933 := d1
					snap934 := d2
					snap935 := d3
					snap936 := d4
					snap937 := d5
					snap938 := d6
					snap939 := d7
					snap940 := d8
					snap941 := d9
					snap942 := d10
					snap943 := d11
					snap944 := d12
					snap945 := d13
					snap946 := d14
					snap947 := d15
					snap948 := d18
					snap949 := d38
					snap950 := d57
					snap951 := d58
					snap952 := d59
					snap953 := d60
					snap954 := d61
					snap955 := d63
					snap956 := d64
					snap957 := d65
					snap958 := d66
					snap959 := d67
					snap960 := d68
					snap961 := d69
					snap962 := d72
					snap963 := d138
					snap964 := d139
					snap965 := d140
					snap966 := d141
					snap967 := d142
					snap968 := d143
					snap969 := d145
					snap970 := d146
					snap971 := d147
					snap972 := d148
					snap973 := d149
					snap974 := d150
					snap975 := d151
					snap976 := d152
					snap977 := d153
					snap978 := d156
					snap979 := d157
					snap980 := d158
					snap981 := d159
					snap982 := d262
					snap983 := d263
					snap984 := d264
					snap985 := d265
					snap986 := d266
					snap987 := d267
					snap988 := d268
					snap989 := d269
					snap990 := d270
					snap991 := d271
					snap992 := d272
					snap993 := d273
					snap994 := d274
					snap995 := d275
					snap996 := d276
					snap997 := d278
					snap998 := d279
					snap999 := d280
					snap1000 := d281
					snap1001 := d283
					snap1002 := d284
					snap1003 := d285
					snap1004 := d434
					snap1005 := d435
					snap1006 := d436
					snap1007 := d437
					snap1008 := d438
					snap1009 := d441
					snap1010 := d442
					snap1011 := d603
					snap1012 := d604
					snap1013 := d605
					snap1014 := d606
					snap1015 := d607
					snap1016 := d608
					snap1017 := d609
					snap1018 := d610
					snap1019 := d611
					snap1020 := d612
					snap1021 := d613
					snap1022 := d615
					snap1023 := d616
					snap1024 := d617
					snap1025 := d618
					snap1026 := d619
					snap1027 := d621
					snap1028 := d623
					snap1029 := d721
					snap1030 := d724
					snap1031 := d824
					snap1032 := d825
					snap1033 := d826
					alloc1034 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps932)
					}
					ctx.RestoreAllocState(alloc1034)
					d1 = snap933
					d2 = snap934
					d3 = snap935
					d4 = snap936
					d5 = snap937
					d6 = snap938
					d7 = snap939
					d8 = snap940
					d9 = snap941
					d10 = snap942
					d11 = snap943
					d12 = snap944
					d13 = snap945
					d14 = snap946
					d15 = snap947
					d18 = snap948
					d38 = snap949
					d57 = snap950
					d58 = snap951
					d59 = snap952
					d60 = snap953
					d61 = snap954
					d63 = snap955
					d64 = snap956
					d65 = snap957
					d66 = snap958
					d67 = snap959
					d68 = snap960
					d69 = snap961
					d72 = snap962
					d138 = snap963
					d139 = snap964
					d140 = snap965
					d141 = snap966
					d142 = snap967
					d143 = snap968
					d145 = snap969
					d146 = snap970
					d147 = snap971
					d148 = snap972
					d149 = snap973
					d150 = snap974
					d151 = snap975
					d152 = snap976
					d153 = snap977
					d156 = snap978
					d157 = snap979
					d158 = snap980
					d159 = snap981
					d262 = snap982
					d263 = snap983
					d264 = snap984
					d265 = snap985
					d266 = snap986
					d267 = snap987
					d268 = snap988
					d269 = snap989
					d270 = snap990
					d271 = snap991
					d272 = snap992
					d273 = snap993
					d274 = snap994
					d275 = snap995
					d276 = snap996
					d278 = snap997
					d279 = snap998
					d280 = snap999
					d281 = snap1000
					d283 = snap1001
					d284 = snap1002
					d285 = snap1003
					d434 = snap1004
					d435 = snap1005
					d436 = snap1006
					d437 = snap1007
					d438 = snap1008
					d441 = snap1009
					d442 = snap1010
					d603 = snap1011
					d604 = snap1012
					d605 = snap1013
					d606 = snap1014
					d607 = snap1015
					d608 = snap1016
					d609 = snap1017
					d610 = snap1018
					d611 = snap1019
					d612 = snap1020
					d613 = snap1021
					d615 = snap1022
					d616 = snap1023
					d617 = snap1024
					d618 = snap1025
					d619 = snap1026
					d621 = snap1027
					d623 = snap1028
					d721 = snap1029
					d724 = snap1030
					d824 = snap1031
					d825 = snap1032
					d826 = snap1033
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps931)
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
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
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
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != LocNone {
						d140 = ps.OverlayValues[140]
					}
					if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
						d141 = ps.OverlayValues[141]
					}
					if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
						d142 = ps.OverlayValues[142]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
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
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
						d153 = ps.OverlayValues[153]
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
					if len(ps.OverlayValues) > 262 && ps.OverlayValues[262].Loc != LocNone {
						d262 = ps.OverlayValues[262]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
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
					if len(ps.OverlayValues) > 283 && ps.OverlayValues[283].Loc != LocNone {
						d283 = ps.OverlayValues[283]
					}
					if len(ps.OverlayValues) > 284 && ps.OverlayValues[284].Loc != LocNone {
						d284 = ps.OverlayValues[284]
					}
					if len(ps.OverlayValues) > 285 && ps.OverlayValues[285].Loc != LocNone {
						d285 = ps.OverlayValues[285]
					}
					if len(ps.OverlayValues) > 434 && ps.OverlayValues[434].Loc != LocNone {
						d434 = ps.OverlayValues[434]
					}
					if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
						d435 = ps.OverlayValues[435]
					}
					if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != LocNone {
						d436 = ps.OverlayValues[436]
					}
					if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != LocNone {
						d437 = ps.OverlayValues[437]
					}
					if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != LocNone {
						d438 = ps.OverlayValues[438]
					}
					if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != LocNone {
						d441 = ps.OverlayValues[441]
					}
					if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
						d442 = ps.OverlayValues[442]
					}
					if len(ps.OverlayValues) > 603 && ps.OverlayValues[603].Loc != LocNone {
						d603 = ps.OverlayValues[603]
					}
					if len(ps.OverlayValues) > 604 && ps.OverlayValues[604].Loc != LocNone {
						d604 = ps.OverlayValues[604]
					}
					if len(ps.OverlayValues) > 605 && ps.OverlayValues[605].Loc != LocNone {
						d605 = ps.OverlayValues[605]
					}
					if len(ps.OverlayValues) > 606 && ps.OverlayValues[606].Loc != LocNone {
						d606 = ps.OverlayValues[606]
					}
					if len(ps.OverlayValues) > 607 && ps.OverlayValues[607].Loc != LocNone {
						d607 = ps.OverlayValues[607]
					}
					if len(ps.OverlayValues) > 608 && ps.OverlayValues[608].Loc != LocNone {
						d608 = ps.OverlayValues[608]
					}
					if len(ps.OverlayValues) > 609 && ps.OverlayValues[609].Loc != LocNone {
						d609 = ps.OverlayValues[609]
					}
					if len(ps.OverlayValues) > 610 && ps.OverlayValues[610].Loc != LocNone {
						d610 = ps.OverlayValues[610]
					}
					if len(ps.OverlayValues) > 611 && ps.OverlayValues[611].Loc != LocNone {
						d611 = ps.OverlayValues[611]
					}
					if len(ps.OverlayValues) > 612 && ps.OverlayValues[612].Loc != LocNone {
						d612 = ps.OverlayValues[612]
					}
					if len(ps.OverlayValues) > 613 && ps.OverlayValues[613].Loc != LocNone {
						d613 = ps.OverlayValues[613]
					}
					if len(ps.OverlayValues) > 615 && ps.OverlayValues[615].Loc != LocNone {
						d615 = ps.OverlayValues[615]
					}
					if len(ps.OverlayValues) > 616 && ps.OverlayValues[616].Loc != LocNone {
						d616 = ps.OverlayValues[616]
					}
					if len(ps.OverlayValues) > 617 && ps.OverlayValues[617].Loc != LocNone {
						d617 = ps.OverlayValues[617]
					}
					if len(ps.OverlayValues) > 618 && ps.OverlayValues[618].Loc != LocNone {
						d618 = ps.OverlayValues[618]
					}
					if len(ps.OverlayValues) > 619 && ps.OverlayValues[619].Loc != LocNone {
						d619 = ps.OverlayValues[619]
					}
					if len(ps.OverlayValues) > 621 && ps.OverlayValues[621].Loc != LocNone {
						d621 = ps.OverlayValues[621]
					}
					if len(ps.OverlayValues) > 623 && ps.OverlayValues[623].Loc != LocNone {
						d623 = ps.OverlayValues[623]
					}
					if len(ps.OverlayValues) > 721 && ps.OverlayValues[721].Loc != LocNone {
						d721 = ps.OverlayValues[721]
					}
					if len(ps.OverlayValues) > 724 && ps.OverlayValues[724].Loc != LocNone {
						d724 = ps.OverlayValues[724]
					}
					if len(ps.OverlayValues) > 824 && ps.OverlayValues[824].Loc != LocNone {
						d824 = ps.OverlayValues[824]
					}
					if len(ps.OverlayValues) > 825 && ps.OverlayValues[825].Loc != LocNone {
						d825 = ps.OverlayValues[825]
					}
					if len(ps.OverlayValues) > 826 && ps.OverlayValues[826].Loc != LocNone {
						d826 = ps.OverlayValues[826]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d7)
					var d1035 JITValueDesc
					if d7.Loc == LocImm {
						d1035 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d7.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d7)
						var d1036 JITValueDesc
						if d7.Loc == LocRegPair {
							ctx.FreeReg(d7.Reg)
							d1036 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d7.Reg2}
							ctx.BindReg(d7.Reg2, &d1036)
							ctx.BindReg(d7.Reg2, &d1036)
						} else {
							d1036 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d7.Reg}
							ctx.BindReg(d7.Reg, &d1036)
							ctx.BindReg(d7.Reg, &d1036)
						}
						d1035 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d1036}, 1)
						d1035.Type = tagFloat
						ctx.BindReg(d1035.Reg, &d1035)
					}
					ctx.StabilizeDescForControlFlow(&d1035)
					if ps.General {
						ctx.SyncDesc(&d1035)
						if d1035.Loc == LocReg {
							ctx.ProtectReg(d1035.Reg)
						} else if d1035.Loc == LocRegPair {
							ctx.ProtectReg(d1035.Reg)
							ctx.ProtectReg(d1035.Reg2)
						}
						d1037 = d1035
						if d1037.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d1037)
						ctx.EmitStoreToStack(d1037, int32(bbs[4].PhiBase)+int32(0))
						if d1035.Loc == LocReg {
							ctx.UnprotectReg(d1035.Reg)
						} else if d1035.Loc == LocRegPair {
							ctx.UnprotectReg(d1035.Reg)
							ctx.UnprotectReg(d1035.Reg2)
						}
					}
					ps1038 := PhiState{General: ps.General}
					ps1038.OverlayValues = make([]JITValueDesc, 1038)
					ps1038.OverlayValues[1] = d1
					ps1038.OverlayValues[2] = d2
					ps1038.OverlayValues[3] = d3
					ps1038.OverlayValues[4] = d4
					ps1038.OverlayValues[5] = d5
					ps1038.OverlayValues[6] = d6
					ps1038.OverlayValues[7] = d7
					ps1038.OverlayValues[8] = d8
					ps1038.OverlayValues[9] = d9
					ps1038.OverlayValues[10] = d10
					ps1038.OverlayValues[11] = d11
					ps1038.OverlayValues[12] = d12
					ps1038.OverlayValues[13] = d13
					ps1038.OverlayValues[14] = d14
					ps1038.OverlayValues[15] = d15
					ps1038.OverlayValues[18] = d18
					ps1038.OverlayValues[38] = d38
					ps1038.OverlayValues[57] = d57
					ps1038.OverlayValues[58] = d58
					ps1038.OverlayValues[59] = d59
					ps1038.OverlayValues[60] = d60
					ps1038.OverlayValues[61] = d61
					ps1038.OverlayValues[63] = d63
					ps1038.OverlayValues[64] = d64
					ps1038.OverlayValues[65] = d65
					ps1038.OverlayValues[66] = d66
					ps1038.OverlayValues[67] = d67
					ps1038.OverlayValues[68] = d68
					ps1038.OverlayValues[69] = d69
					ps1038.OverlayValues[72] = d72
					ps1038.OverlayValues[138] = d138
					ps1038.OverlayValues[139] = d139
					ps1038.OverlayValues[140] = d140
					ps1038.OverlayValues[141] = d141
					ps1038.OverlayValues[142] = d142
					ps1038.OverlayValues[143] = d143
					ps1038.OverlayValues[145] = d145
					ps1038.OverlayValues[146] = d146
					ps1038.OverlayValues[147] = d147
					ps1038.OverlayValues[148] = d148
					ps1038.OverlayValues[149] = d149
					ps1038.OverlayValues[150] = d150
					ps1038.OverlayValues[151] = d151
					ps1038.OverlayValues[152] = d152
					ps1038.OverlayValues[153] = d153
					ps1038.OverlayValues[156] = d156
					ps1038.OverlayValues[157] = d157
					ps1038.OverlayValues[158] = d158
					ps1038.OverlayValues[159] = d159
					ps1038.OverlayValues[262] = d262
					ps1038.OverlayValues[263] = d263
					ps1038.OverlayValues[264] = d264
					ps1038.OverlayValues[265] = d265
					ps1038.OverlayValues[266] = d266
					ps1038.OverlayValues[267] = d267
					ps1038.OverlayValues[268] = d268
					ps1038.OverlayValues[269] = d269
					ps1038.OverlayValues[270] = d270
					ps1038.OverlayValues[271] = d271
					ps1038.OverlayValues[272] = d272
					ps1038.OverlayValues[273] = d273
					ps1038.OverlayValues[274] = d274
					ps1038.OverlayValues[275] = d275
					ps1038.OverlayValues[276] = d276
					ps1038.OverlayValues[278] = d278
					ps1038.OverlayValues[279] = d279
					ps1038.OverlayValues[280] = d280
					ps1038.OverlayValues[281] = d281
					ps1038.OverlayValues[283] = d283
					ps1038.OverlayValues[284] = d284
					ps1038.OverlayValues[285] = d285
					ps1038.OverlayValues[434] = d434
					ps1038.OverlayValues[435] = d435
					ps1038.OverlayValues[436] = d436
					ps1038.OverlayValues[437] = d437
					ps1038.OverlayValues[438] = d438
					ps1038.OverlayValues[441] = d441
					ps1038.OverlayValues[442] = d442
					ps1038.OverlayValues[603] = d603
					ps1038.OverlayValues[604] = d604
					ps1038.OverlayValues[605] = d605
					ps1038.OverlayValues[606] = d606
					ps1038.OverlayValues[607] = d607
					ps1038.OverlayValues[608] = d608
					ps1038.OverlayValues[609] = d609
					ps1038.OverlayValues[610] = d610
					ps1038.OverlayValues[611] = d611
					ps1038.OverlayValues[612] = d612
					ps1038.OverlayValues[613] = d613
					ps1038.OverlayValues[615] = d615
					ps1038.OverlayValues[616] = d616
					ps1038.OverlayValues[617] = d617
					ps1038.OverlayValues[618] = d618
					ps1038.OverlayValues[619] = d619
					ps1038.OverlayValues[621] = d621
					ps1038.OverlayValues[623] = d623
					ps1038.OverlayValues[721] = d721
					ps1038.OverlayValues[724] = d724
					ps1038.OverlayValues[824] = d824
					ps1038.OverlayValues[825] = d825
					ps1038.OverlayValues[826] = d826
					ps1038.OverlayValues[1035] = d1035
					ps1038.OverlayValues[1036] = d1036
					ps1038.OverlayValues[1037] = d1037
					ps1038.PhiValues = make([]JITValueDesc, 1)
					d1039 = d1035
					ps1038.PhiValues[0] = d1039
					if ps1038.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps1038)
					return result
				}
				ps1040 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps1040)
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
