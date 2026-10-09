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
			// All loop accumulators are native float64 values until the final
			// NewFloat boundary. Keep them in the backend FP register class.
			JITNativeFP: true,

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
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d130 JITValueDesc
				_ = d130
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d135 JITValueDesc
				_ = d135
				var d136 JITValueDesc
				_ = d136
				var d137 JITValueDesc
				_ = d137
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
				var d144 JITValueDesc
				_ = d144
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
				var d207 JITValueDesc
				_ = d207
				var d208 JITValueDesc
				_ = d208
				var d209 JITValueDesc
				_ = d209
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
				var d366 JITValueDesc
				_ = d366
				var d446 JITValueDesc
				_ = d446
				var d447 JITValueDesc
				_ = d447
				var d448 JITValueDesc
				_ = d448
				var d531 JITValueDesc
				_ = d531
				var d532 JITValueDesc
				_ = d532
				var d533 JITValueDesc
				_ = d533
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(128))
				branchSerial := ctx.branchSerial
				_ = branchSerial
				returnType := uint8(JITTypeUnknown)
				returnTypeSeen := false
				mergeReturnType := func(t uint8) {
					if !returnTypeSeen {
						returnType, returnTypeSeen = t, true
					} else if returnType != t {
						returnType = JITTypeUnknown
					}
				}
				var bbs [15]BBDescriptor
				bbs[2].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiBase = int32(phiBase0) + int32(16)
				bbs[6].PhiBase = int32(phiBase0) + int32(32)
				bbs[10].PhiBase = int32(phiBase0) + int32(96)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 64}, {Color: 1, Width: 1, Class: JITRegisterClassFP, Cost: 35}, {Color: 2, Width: 1, Class: JITRegisterClassFP, Cost: 17}, {Color: 3, Width: 1, Class: JITRegisterClassFP, Cost: 17}}, Count: 4})
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
					d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
				} else {
					d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
				}
				_ = d10
				var d11 JITValueDesc
				if phiHomeOK3 {
					d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
				} else {
					d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
				}
				_ = d11
				var d12 JITValueDesc
				if phiHomeOK4 {
					d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
				} else {
					d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
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
					d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
				} else {
					d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
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
				bbs[0].Render = func() JITValueDesc {
					if bbs[0].Rendered {
						ctx.EmitJmp(lbl1)
						return result
					}
					bbs[0].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl1)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					d16 = args[0]
					d16.ID = 0
					if d16.Type == tagSlice {
						d17 = jitKnownSliceHeader(ctx, &d16)
					} else {
						d17 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d16}, 3)
					}
					if d17.Loc == LocRegTriple {
						ctx.BindReg(d17.Reg, &d17)
						ctx.BindReg(d17.Reg2, &d17)
						ctx.BindReg(d17.Reg3, &d17)
					}
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.FreeDesc(&d16)
					d18 = args[1]
					d18.ID = 0
					if d18.Type == tagSlice {
						d19 = jitKnownSliceHeader(ctx, &d18)
					} else {
						d19 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d18}, 3)
					}
					if d19.Loc == LocRegTriple {
						ctx.BindReg(d19.Reg, &d19)
						ctx.BindReg(d19.Reg2, &d19)
						ctx.BindReg(d19.Reg3, &d19)
					}
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d18)
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
					ctx.EnsureDesc(&d20)
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
							return bbs[1].Render()
						}
						ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
						return bbs[2].Render()
					}
					lbl16 := ctx.ReserveLabel()
					ctx.EmitJump(d22.Condition, lbl2)
					ctx.EmitJmp(lbl16)
					ctx.FreeDesc(&d21)
					snap23 := d8
					snap24 := d9
					snap25 := d10
					snap26 := d11
					snap27 := d12
					snap28 := d13
					snap29 := d14
					snap30 := d15
					snap31 := d16
					snap32 := d17
					snap33 := d18
					snap34 := d19
					snap35 := d20
					snap36 := d21
					snap37 := d22
					alloc38 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl16)
					ctx.EmitStoreScmerToStack(JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("DOT")}, int32(bbs[2].PhiBase)+int32(0))
					ctx.EmitJmp(lbl3)
					ctx.RestoreAllocState(alloc38)
					d8 = snap23
					d9 = snap24
					d10 = snap25
					d11 = snap26
					d12 = snap27
					d13 = snap28
					d14 = snap29
					d15 = snap30
					d16 = snap31
					d17 = snap32
					d18 = snap33
					d19 = snap34
					d20 = snap35
					d21 = snap36
					d22 = snap37
					if !bbs[2].Rendered {
						snap39 := d8
						snap40 := d9
						snap41 := d10
						snap42 := d11
						snap43 := d12
						snap44 := d13
						snap45 := d14
						snap46 := d15
						snap47 := d16
						snap48 := d17
						snap49 := d18
						snap50 := d19
						snap51 := d20
						snap52 := d21
						snap53 := d22
						alloc54 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc54)
						d8 = snap39
						d9 = snap40
						d10 = snap41
						d11 = snap42
						d12 = snap43
						d13 = snap44
						d14 = snap45
						d15 = snap46
						d16 = snap47
						d17 = snap48
						d18 = snap49
						d19 = snap50
						d20 = snap51
						d21 = snap52
						d22 = snap53
					}
					if !bbs[1].Rendered {
						return bbs[1].Render()
					}
					return result
					return result
				}
				bbs[1].Render = func() JITValueDesc {
					if bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					bbs[1].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl2)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					d55 = args[2]
					d55.ID = 0
					d57 = d55
					ctx.SyncDesc(&d57)
					if d57.Loc == LocMem {
						tmpScalar := JITValueDesc{Loc: LocReg, Type: d57.Type, Reg: ctx.AllocReg()}
						scratch := ctx.AllocRegExcept(tmpScalar.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d57.MemPtr))
						ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
						ctx.FreeReg(scratch)
						ctx.BindReg(tmpScalar.Reg, &tmpScalar)
						d57 = tmpScalar
					}
					d57 = JITPrepareScmerGoArg(ctx, d57)
					if d57.Loc != LocRegPair && d57.Loc != LocStackPair && d57.Loc != LocInputPair {
						panic("jit: Scmer.String receiver not materialized as pair")
					}
					d56 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.String), []JITValueDesc{d57}, 2)
					ctx.FreeDesc(&d55)
					ctx.EnsureDesc(&d56)
					if d56.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d56.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d56.Imm)
						ptrWord, _ := d56.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d56.Imm.String())))
						d56 = tmpPair
					} else if d56.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d56.Type, Reg: ctx.AllocRegExcept(d56.Reg), Reg2: ctx.AllocRegExcept(d56.Reg)}
						switch d56.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d56)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d56)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d56)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d56)
						d56 = tmpPair
					}
					if d56.Loc != LocRegPair && d56.Loc != LocStackPair && d56.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (strings.ToUpper arg0)")
					}
					ctx.SyncDesc(&d56)
					d58 = ctx.EmitGoCallScalar(GoFuncAddr(strings.ToUpper), []JITValueDesc{d56}, 2)
					d58.NoHeapPointer = false
					ctx.BindReg(d58.Reg, &d58)
					ctx.BindReg(d58.Reg2, &d58)
					ctx.StabilizeDescForControlFlow(&d58)
					ctx.SyncDesc(&d58)
					if d58.Loc == LocReg || d58.Loc == LocFPReg {
						ctx.ProtectReg(d58.Reg)
					} else if d58.Loc == LocRegPair {
						ctx.ProtectReg(d58.Reg)
						ctx.ProtectReg(d58.Reg2)
					}
					d59 = d58
					if d59.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d59)
					if d59.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d59, int32(bbs[2].PhiBase)+int32(0), 2)
					} else if d59.Loc == LocInputPair {
						ctx.EnsureDesc(&d59)
						ctx.EmitStoreScmerToStack(d59, int32(bbs[2].PhiBase)+int32(0))
					} else if d59.Loc == LocRegPair || d59.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d59, int32(bbs[2].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d59)
						ctx.EmitStoreToStack(d59, int32(bbs[2].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[2].PhiBase)+int32(0))+8)
					}
					if d58.Loc == LocReg || d58.Loc == LocFPReg {
						ctx.UnprotectReg(d58.Reg)
					} else if d58.Loc == LocRegPair {
						ctx.UnprotectReg(d58.Reg)
						ctx.UnprotectReg(d58.Reg2)
					}
					return bbs[2].Render()
					return result
				}
				bbs[2].Render = func() JITValueDesc {
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					bbs[2].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl3)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d60 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d60.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d60.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d60.Reg, &d60)
						ctx.BindReg(d60.Reg2, &d60)
					} else {
						d60 = d8
					}
					d61 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("COSINE")}
					if d61.Loc == LocImm {
						ctx.TrackImm(d61.Imm)
						ptrWord, _ := d61.Imm.RawWords()
						d62 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d62.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d62.Reg2, uint64(len(d61.Imm.String())))
						ctx.BindReg(d62.Reg, &d62)
						ctx.BindReg(d62.Reg2, &d62)
					} else {
						d62 = d61
					}
					d63 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d60, d62}, 1)
					ctx.EmitAndRegImm32(d63.Reg, 1)
					d63.Type = tagBool
					ctx.BindReg(d63.Reg, &d63)
					d64 = d63
					ctx.EnsureDesc(&d64)
					if d64.Loc != LocImm && d64.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d64.Loc == LocImm {
						if d64.Imm.Bool() {
							return bbs[3].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitCmpRegImm32(d64.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl4)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
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
						snap80 := d55
						snap81 := d56
						snap82 := d57
						snap83 := d58
						snap84 := d59
						snap85 := d60
						snap86 := d61
						snap87 := d62
						snap88 := d63
						snap89 := d64
						alloc90 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc90)
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
						d55 = snap80
						d56 = snap81
						d57 = snap82
						d58 = snap83
						d59 = snap84
						d60 = snap85
						d61 = snap86
						d62 = snap87
						d63 = snap88
						d64 = snap89
					}
					if !bbs[3].Rendered {
						return bbs[3].Render()
					}
					return result
					ctx.FreeDesc(&d63)
					return result
				}
				bbs[3].Render = func() JITValueDesc {
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					bbs[3].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl4)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
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
					return bbs[6].Render()
					return result
				}
				bbs[4].Render = func() JITValueDesc {
					if bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					bbs[4].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl5)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						ctx.EmitMakeFloat(result, d9)
					} else {
						ctx.EmitMovToReg(result.Reg2, d9)
						d91 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: result.Reg2, ID: 0}
						ctx.EmitMakeFloat(result, d91)
						if d9.Loc == LocReg && d9.Reg != result.Reg2 {
							ctx.FreeReg(d9.Reg)
						}
					}
					result.Type = tagFloat
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[5].Render = func() JITValueDesc {
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
						return result
					}
					bbs[5].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
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
					return bbs[10].Render()
					return result
				}
				bbs[6].Render = func() JITValueDesc {
					if bbs[6].Rendered {
						ctx.EmitJmp(lbl7)
						return result
					}
					bbs[6].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
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
					if d17.SliceSizeKnown {
						d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d92 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d92)
					ctx.EnsureDescsTogether(&d13, &d92)
					if d13.Loc == LocImm && d92.Loc == LocImm {
						d93 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d92.Imm.Int())}
					} else if d92.Loc == LocImm {
						r7 := ctx.AllocRegExcept(d13.Reg)
						if d92.Imm.Int() >= -2147483648 && d92.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d92.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d92.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, ctx.ScratchReg)
						}
						d93 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r7, Condition: CondSignedLess}
						ctx.BindReg(r7, &d93)
					} else if d13.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d92.Reg)
						d93 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r8, Condition: CondSignedLess}
						ctx.BindReg(r8, &d93)
					} else {
						r9 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpInt64(d13.Reg, d92.Reg)
						d93 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLess}
						ctx.BindReg(r9, &d93)
					}
					ctx.FreeDesc(&d92)
					d94 = d93
					ctx.EnsureDesc(&d94)
					if d94.Loc != LocImm && d94.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d94.Loc == LocImm {
						if d94.Imm.Bool() {
							return bbs[9].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d94.Condition, lbl10)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d93)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap95 := d8
						snap96 := d9
						snap97 := d10
						snap98 := d11
						snap99 := d12
						snap100 := d13
						snap101 := d14
						snap102 := d15
						snap103 := d16
						snap104 := d17
						snap105 := d18
						snap106 := d19
						snap107 := d20
						snap108 := d21
						snap109 := d22
						snap110 := d55
						snap111 := d56
						snap112 := d57
						snap113 := d58
						snap114 := d59
						snap115 := d60
						snap116 := d61
						snap117 := d62
						snap118 := d63
						snap119 := d64
						snap120 := d91
						snap121 := d92
						snap122 := d93
						snap123 := d94
						alloc124 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc124)
						d8 = snap95
						d9 = snap96
						d10 = snap97
						d11 = snap98
						d12 = snap99
						d13 = snap100
						d14 = snap101
						d15 = snap102
						d16 = snap103
						d17 = snap104
						d18 = snap105
						d19 = snap106
						d20 = snap107
						d21 = snap108
						d22 = snap109
						d55 = snap110
						d56 = snap111
						d57 = snap112
						d58 = snap113
						d59 = snap114
						d60 = snap115
						d61 = snap116
						d62 = snap117
						d63 = snap118
						d64 = snap119
						d91 = snap120
						d92 = snap121
						d93 = snap122
						d94 = snap123
					}
					if !bbs[9].Rendered {
						return bbs[9].Render()
					}
					return result
					return result
				}
				bbs[7].Render = func() JITValueDesc {
					if bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					bbs[7].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					d126 = ctx.EmitSliceElementAddress(&d17, &d13, 16)
					ctx.EnsureDesc(&d126)
					r10 := ctx.AllocRegExcept(d126.Reg)
					ctx.EmitMovRegMem(r10, d126.Reg, 8)
					ctx.EmitMovRegMem(d126.Reg, d126.Reg, 0)
					d125 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d126.Reg, Reg2: r10}
					ctx.BindReg(d126.Reg, &d125)
					ctx.BindReg(r10, &d125)
					ctx.EnsureDesc(&d125)
					d127 = d125
					_ = d127
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl17 := ctx.ReserveLabel()
					_ = lbl17
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d128 = ctx.EmitFloatDesc(d127)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d128)
					ctx.FreeDesc(&d125)
					ctx.EnsureDesc(&d13)
					d130 = ctx.EmitSliceElementAddress(&d19, &d13, 16)
					ctx.EnsureDesc(&d130)
					r11 := ctx.AllocRegExcept(d130.Reg)
					ctx.EmitMovRegMem(r11, d130.Reg, 8)
					ctx.EmitMovRegMem(d130.Reg, d130.Reg, 0)
					d129 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d130.Reg, Reg2: r11}
					ctx.BindReg(d130.Reg, &d129)
					ctx.BindReg(r11, &d129)
					ctx.EnsureDesc(&d129)
					d131 = d129
					_ = d131
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl18 := ctx.ReserveLabel()
					_ = lbl18
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d132 = ctx.EmitFloatDesc(d131)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d132)
					ctx.FreeDesc(&d129)
					ctx.EnsureDesc(&d128)
					ctx.EnsureDesc(&d128)
					d133 = ctx.EmitFloatBinary(&d128, &d128, JITFloatMul)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d133)
					d134 = ctx.EmitFloatBinary(&d11, &d133, JITFloatAdd)
					ctx.FreeDesc(&d133)
					ctx.EnsureDesc(&d132)
					ctx.EnsureDesc(&d132)
					d135 = ctx.EmitFloatBinary(&d132, &d132, JITFloatMul)
					ctx.EnsureDesc(&d12)
					ctx.EnsureDesc(&d135)
					d136 = ctx.EmitFloatBinary(&d12, &d135, JITFloatAdd)
					ctx.FreeDesc(&d135)
					ctx.EnsureDesc(&d128)
					ctx.EnsureDesc(&d132)
					d137 = ctx.EmitFloatBinary(&d128, &d132, JITFloatMul)
					ctx.FreeDesc(&d128)
					ctx.FreeDesc(&d132)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d137)
					d138 = ctx.EmitFloatBinary(&d10, &d137, JITFloatAdd)
					ctx.FreeDesc(&d137)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						d139 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK5 {
							scratch = r3
						} else {
							scratch = ctx.AllocRegExcept(d13.Reg)
						}
						ctx.EmitMovRegReg(scratch, d13.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d139 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d139)
					}
					if d139.Loc == LocReg && d13.Loc == LocReg && d139.Reg == d13.Reg {
						ctx.TransferReg(d13.Reg)
						d13.Loc = LocNone
					}
					ctx.SyncDesc(&d134)
					if d134.Loc == LocReg || d134.Loc == LocFPReg {
						ctx.ProtectReg(d134.Reg)
					} else if d134.Loc == LocRegPair {
						ctx.ProtectReg(d134.Reg)
						ctx.ProtectReg(d134.Reg2)
					}
					ctx.SyncDesc(&d136)
					if d136.Loc == LocReg || d136.Loc == LocFPReg {
						ctx.ProtectReg(d136.Reg)
					} else if d136.Loc == LocRegPair {
						ctx.ProtectReg(d136.Reg)
						ctx.ProtectReg(d136.Reg2)
					}
					ctx.SyncDesc(&d138)
					if d138.Loc == LocReg || d138.Loc == LocFPReg {
						ctx.ProtectReg(d138.Reg)
					} else if d138.Loc == LocRegPair {
						ctx.ProtectReg(d138.Reg)
						ctx.ProtectReg(d138.Reg2)
					}
					d140 = d138
					if d140.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d140)
					if phiHomeOK2 {
						ctx.EmitMovToReg(r0, d140)
					} else {
						ctx.EmitStoreToStack(d140, int32(bbs[6].PhiBase)+int32(0))
					}
					d141 = d134
					if d141.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d141)
					if phiHomeOK3 {
						ctx.EmitMovToReg(r1, d141)
					} else {
						ctx.EmitStoreToStack(d141, int32(bbs[6].PhiBase)+int32(16))
					}
					d142 = d136
					if d142.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d142)
					if phiHomeOK4 {
						ctx.EmitMovToReg(r2, d142)
					} else {
						ctx.EmitStoreToStack(d142, int32(bbs[6].PhiBase)+int32(32))
					}
					if d134.Loc == LocReg || d134.Loc == LocFPReg {
						ctx.UnprotectReg(d134.Reg)
					} else if d134.Loc == LocRegPair {
						ctx.UnprotectReg(d134.Reg)
						ctx.UnprotectReg(d134.Reg2)
					}
					if d136.Loc == LocReg || d136.Loc == LocFPReg {
						ctx.UnprotectReg(d136.Reg)
					} else if d136.Loc == LocRegPair {
						ctx.UnprotectReg(d136.Reg)
						ctx.UnprotectReg(d136.Reg2)
					}
					if d138.Loc == LocReg || d138.Loc == LocFPReg {
						ctx.UnprotectReg(d138.Reg)
					} else if d138.Loc == LocRegPair {
						ctx.UnprotectReg(d138.Reg)
						ctx.UnprotectReg(d138.Reg2)
					}
					ctx.SyncDesc(&d139)
					if d139.Loc == LocReg || d139.Loc == LocFPReg {
						ctx.ProtectReg(d139.Reg)
					} else if d139.Loc == LocRegPair {
						ctx.ProtectReg(d139.Reg)
						ctx.ProtectReg(d139.Reg2)
					}
					d143 = d139
					if d143.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d143)
					if phiHomeOK5 {
						ctx.EmitMovToReg(r3, d143)
					} else {
						ctx.EmitStoreToStack(d143, int32(bbs[6].PhiBase)+int32(48))
					}
					if d139.Loc == LocReg || d139.Loc == LocFPReg {
						ctx.UnprotectReg(d139.Reg)
					} else if d139.Loc == LocRegPair {
						ctx.UnprotectReg(d139.Reg)
						ctx.UnprotectReg(d139.Reg2)
					}
					return bbs[6].Render()
					return result
				}
				bbs[8].Render = func() JITValueDesc {
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
						return result
					}
					bbs[8].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d12)
					d144 = ctx.EmitFloatBinary(&d11, &d12, JITFloatMul)
					ctx.EnsureDesc(&d144)
					if d144.Loc == LocImm {
						d145 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d144.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d144)
						if d144.Loc == LocRegPair {
							ctx.FreeReg(d144.Reg)
							d146 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d144.Reg2}
							ctx.BindReg(d144.Reg2, &d146)
							ctx.BindReg(d144.Reg2, &d146)
						} else {
							d146 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d144.Reg}
							ctx.BindReg(d144.Reg, &d146)
							ctx.BindReg(d144.Reg, &d146)
						}
						d145 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d146}, 1)
						d145.Type = tagFloat
						ctx.BindReg(d145.Reg, &d145)
					}
					ctx.FreeDesc(&d144)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d145)
					d147 = ctx.EmitFloatBinary(&d10, &d145, JITFloatDiv)
					ctx.EnsureDesc(&d147)
					ctx.EmitStoreToStack(d147, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d147)
					ctx.FreeDesc(&d145)
					return bbs[4].Render()
					return result
				}
				bbs[9].Render = func() JITValueDesc {
					if bbs[9].Rendered {
						ctx.EmitJmp(lbl10)
						return result
					}
					bbs[9].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					if d19.SliceSizeKnown {
						d148 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d148 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d148 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d148 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d148)
					ctx.EnsureDescsTogether(&d13, &d148)
					if d13.Loc == LocImm && d148.Loc == LocImm {
						d149 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < d148.Imm.Int())}
					} else if d148.Loc == LocImm {
						r12 := ctx.AllocRegExcept(d13.Reg)
						if d148.Imm.Int() >= -2147483648 && d148.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d13.Reg, int32(d148.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d148.Imm.Int()))
							ctx.EmitCmpInt64(d13.Reg, ctx.ScratchReg)
						}
						d149 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r12, Condition: CondSignedLess}
						ctx.BindReg(r12, &d149)
					} else if d13.Loc == LocImm {
						r13 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d148.Reg)
						d149 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r13, Condition: CondSignedLess}
						ctx.BindReg(r13, &d149)
					} else {
						r14 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpInt64(d13.Reg, d148.Reg)
						d149 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
						ctx.BindReg(r14, &d149)
					}
					ctx.FreeDesc(&d148)
					d150 = d149
					ctx.EnsureDesc(&d150)
					if d150.Loc != LocImm && d150.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d150.Loc == LocImm {
						if d150.Imm.Bool() {
							return bbs[7].Render()
						}
						return bbs[8].Render()
					}
					ctx.EmitJump(d150.Condition, lbl8)
					if bbs[8].Rendered {
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d149)
					ctx.FlushRegisterMoves()
					if !bbs[8].Rendered {
						snap151 := d8
						snap152 := d9
						snap153 := d10
						snap154 := d11
						snap155 := d12
						snap156 := d13
						snap157 := d14
						snap158 := d15
						snap159 := d16
						snap160 := d17
						snap161 := d18
						snap162 := d19
						snap163 := d20
						snap164 := d21
						snap165 := d22
						snap166 := d55
						snap167 := d56
						snap168 := d57
						snap169 := d58
						snap170 := d59
						snap171 := d60
						snap172 := d61
						snap173 := d62
						snap174 := d63
						snap175 := d64
						snap176 := d91
						snap177 := d92
						snap178 := d93
						snap179 := d94
						snap180 := d125
						snap181 := d126
						snap182 := d127
						snap183 := d128
						snap184 := d129
						snap185 := d130
						snap186 := d131
						snap187 := d132
						snap188 := d133
						snap189 := d134
						snap190 := d135
						snap191 := d136
						snap192 := d137
						snap193 := d138
						snap194 := d139
						snap195 := d140
						snap196 := d141
						snap197 := d142
						snap198 := d143
						snap199 := d144
						snap200 := d145
						snap201 := d146
						snap202 := d147
						snap203 := d148
						snap204 := d149
						snap205 := d150
						alloc206 := ctx.SnapshotAllocState()
						bbs[8].Render()
						ctx.RestoreAllocState(alloc206)
						d8 = snap151
						d9 = snap152
						d10 = snap153
						d11 = snap154
						d12 = snap155
						d13 = snap156
						d14 = snap157
						d15 = snap158
						d16 = snap159
						d17 = snap160
						d18 = snap161
						d19 = snap162
						d20 = snap163
						d21 = snap164
						d22 = snap165
						d55 = snap166
						d56 = snap167
						d57 = snap168
						d58 = snap169
						d59 = snap170
						d60 = snap171
						d61 = snap172
						d62 = snap173
						d63 = snap174
						d64 = snap175
						d91 = snap176
						d92 = snap177
						d93 = snap178
						d94 = snap179
						d125 = snap180
						d126 = snap181
						d127 = snap182
						d128 = snap183
						d129 = snap184
						d130 = snap185
						d131 = snap186
						d132 = snap187
						d133 = snap188
						d134 = snap189
						d135 = snap190
						d136 = snap191
						d137 = snap192
						d138 = snap193
						d139 = snap194
						d140 = snap195
						d141 = snap196
						d142 = snap197
						d143 = snap198
						d144 = snap199
						d145 = snap200
						d146 = snap201
						d147 = snap202
						d148 = snap203
						d149 = snap204
						d150 = snap205
					}
					if !bbs[7].Rendered {
						return bbs[7].Render()
					}
					return result
					return result
				}
				bbs[10].Render = func() JITValueDesc {
					if bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					bbs[10].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					if phiHomeOK6 && d14.Loc == LocReg {
						ctx.BindReg(r4, &d14)
					}
					if phiHomeOK7 && d15.Loc == LocReg {
						ctx.BindReg(r5, &d15)
					}
					ctx.ReclaimUntrackedRegs()
					if d17.SliceSizeKnown {
						d207 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
					} else if d17.Loc == LocImm {
						d207 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
					} else if d17.Loc == LocStackTriple {
						d207 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
							d207 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
						} else if d17.Loc == LocReg {
							d207 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d207)
					ctx.EnsureDescsTogether(&d15, &d207)
					if d15.Loc == LocImm && d207.Loc == LocImm {
						d208 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d207.Imm.Int())}
					} else if d207.Loc == LocImm {
						r15 := ctx.AllocRegExcept(d15.Reg)
						if d207.Imm.Int() >= -2147483648 && d207.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d207.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d207.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, ctx.ScratchReg)
						}
						d208 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondSignedLess}
						ctx.BindReg(r15, &d208)
					} else if d15.Loc == LocImm {
						r16 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d207.Reg)
						d208 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
						ctx.BindReg(r16, &d208)
					} else {
						r17 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpInt64(d15.Reg, d207.Reg)
						d208 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedLess}
						ctx.BindReg(r17, &d208)
					}
					ctx.FreeDesc(&d207)
					d209 = d208
					ctx.EnsureDesc(&d209)
					if d209.Loc != LocImm && d209.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d209.Loc == LocImm {
						if d209.Imm.Bool() {
							return bbs[13].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitJump(d209.Condition, lbl14)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d208)
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap210 := d8
						snap211 := d9
						snap212 := d10
						snap213 := d11
						snap214 := d12
						snap215 := d13
						snap216 := d14
						snap217 := d15
						snap218 := d16
						snap219 := d17
						snap220 := d18
						snap221 := d19
						snap222 := d20
						snap223 := d21
						snap224 := d22
						snap225 := d55
						snap226 := d56
						snap227 := d57
						snap228 := d58
						snap229 := d59
						snap230 := d60
						snap231 := d61
						snap232 := d62
						snap233 := d63
						snap234 := d64
						snap235 := d91
						snap236 := d92
						snap237 := d93
						snap238 := d94
						snap239 := d125
						snap240 := d126
						snap241 := d127
						snap242 := d128
						snap243 := d129
						snap244 := d130
						snap245 := d131
						snap246 := d132
						snap247 := d133
						snap248 := d134
						snap249 := d135
						snap250 := d136
						snap251 := d137
						snap252 := d138
						snap253 := d139
						snap254 := d140
						snap255 := d141
						snap256 := d142
						snap257 := d143
						snap258 := d144
						snap259 := d145
						snap260 := d146
						snap261 := d147
						snap262 := d148
						snap263 := d149
						snap264 := d150
						snap265 := d207
						snap266 := d208
						snap267 := d209
						alloc268 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc268)
						d8 = snap210
						d9 = snap211
						d10 = snap212
						d11 = snap213
						d12 = snap214
						d13 = snap215
						d14 = snap216
						d15 = snap217
						d16 = snap218
						d17 = snap219
						d18 = snap220
						d19 = snap221
						d20 = snap222
						d21 = snap223
						d22 = snap224
						d55 = snap225
						d56 = snap226
						d57 = snap227
						d58 = snap228
						d59 = snap229
						d60 = snap230
						d61 = snap231
						d62 = snap232
						d63 = snap233
						d64 = snap234
						d91 = snap235
						d92 = snap236
						d93 = snap237
						d94 = snap238
						d125 = snap239
						d126 = snap240
						d127 = snap241
						d128 = snap242
						d129 = snap243
						d130 = snap244
						d131 = snap245
						d132 = snap246
						d133 = snap247
						d134 = snap248
						d135 = snap249
						d136 = snap250
						d137 = snap251
						d138 = snap252
						d139 = snap253
						d140 = snap254
						d141 = snap255
						d142 = snap256
						d143 = snap257
						d144 = snap258
						d145 = snap259
						d146 = snap260
						d147 = snap261
						d148 = snap262
						d149 = snap263
						d150 = snap264
						d207 = snap265
						d208 = snap266
						d209 = snap267
					}
					if !bbs[13].Rendered {
						return bbs[13].Render()
					}
					return result
					return result
				}
				bbs[11].Render = func() JITValueDesc {
					if bbs[11].Rendered {
						ctx.EmitJmp(lbl12)
						return result
					}
					bbs[11].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					d270 = ctx.EmitSliceElementAddress(&d17, &d15, 16)
					ctx.EnsureDesc(&d270)
					r18 := ctx.AllocRegExcept(d270.Reg)
					ctx.EmitMovRegMem(r18, d270.Reg, 8)
					ctx.EmitMovRegMem(d270.Reg, d270.Reg, 0)
					d269 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d270.Reg, Reg2: r18}
					ctx.BindReg(d270.Reg, &d269)
					ctx.BindReg(r18, &d269)
					ctx.EnsureDesc(&d269)
					d271 = d269
					_ = d271
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					lbl19 := ctx.ReserveLabel()
					_ = lbl19
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl19)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d272 = ctx.EmitFloatDesc(d271)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d272)
					ctx.FreeDesc(&d269)
					ctx.EnsureDesc(&d15)
					d274 = ctx.EmitSliceElementAddress(&d19, &d15, 16)
					ctx.EnsureDesc(&d274)
					r19 := ctx.AllocRegExcept(d274.Reg)
					ctx.EmitMovRegMem(r19, d274.Reg, 8)
					ctx.EmitMovRegMem(d274.Reg, d274.Reg, 0)
					d273 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d274.Reg, Reg2: r19}
					ctx.BindReg(d274.Reg, &d273)
					ctx.BindReg(r19, &d273)
					ctx.EnsureDesc(&d273)
					d275 = d273
					_ = d275
					bbpos_4_0 := int32(-1)
					_ = bbpos_4_0
					lbl20 := ctx.ReserveLabel()
					_ = lbl20
					bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl20)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d276 = ctx.EmitFloatDesc(d275)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d276)
					ctx.FreeDesc(&d273)
					ctx.EnsureDesc(&d272)
					ctx.EnsureDesc(&d276)
					d277 = ctx.EmitFloatBinary(&d272, &d276, JITFloatMul)
					ctx.FreeDesc(&d272)
					ctx.FreeDesc(&d276)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d277)
					d278 = ctx.EmitFloatBinary(&d14, &d277, JITFloatAdd)
					ctx.FreeDesc(&d277)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						d279 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int() + 1)}
					} else {
						var scratch Reg
						if phiHomeOK7 {
							scratch = r5
						} else {
							scratch = ctx.AllocRegExcept(d15.Reg)
						}
						ctx.EmitMovRegReg(scratch, d15.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d279 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d279)
					}
					if d279.Loc == LocReg && d15.Loc == LocReg && d279.Reg == d15.Reg {
						ctx.TransferReg(d15.Reg)
						d15.Loc = LocNone
					}
					ctx.SyncDesc(&d278)
					if d278.Loc == LocReg || d278.Loc == LocFPReg {
						ctx.ProtectReg(d278.Reg)
					} else if d278.Loc == LocRegPair {
						ctx.ProtectReg(d278.Reg)
						ctx.ProtectReg(d278.Reg2)
					}
					ctx.SyncDesc(&d279)
					if d279.Loc == LocReg || d279.Loc == LocFPReg {
						ctx.ProtectReg(d279.Reg)
					} else if d279.Loc == LocRegPair {
						ctx.ProtectReg(d279.Reg)
						ctx.ProtectReg(d279.Reg2)
					}
					d280 = d278
					if d280.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d280)
					if phiHomeOK6 {
						ctx.EmitMovToReg(r4, d280)
					} else {
						ctx.EmitStoreToStack(d280, int32(bbs[10].PhiBase)+int32(0))
					}
					d281 = d279
					if d281.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d281)
					if phiHomeOK7 {
						ctx.EmitMovToReg(r5, d281)
					} else {
						ctx.EmitStoreToStack(d281, int32(bbs[10].PhiBase)+int32(16))
					}
					if d278.Loc == LocReg || d278.Loc == LocFPReg {
						ctx.UnprotectReg(d278.Reg)
					} else if d278.Loc == LocRegPair {
						ctx.UnprotectReg(d278.Reg)
						ctx.UnprotectReg(d278.Reg2)
					}
					if d279.Loc == LocReg || d279.Loc == LocFPReg {
						ctx.UnprotectReg(d279.Reg)
					} else if d279.Loc == LocRegPair {
						ctx.UnprotectReg(d279.Reg)
						ctx.UnprotectReg(d279.Reg2)
					}
					return bbs[10].Render()
					return result
				}
				bbs[12].Render = func() JITValueDesc {
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
						return result
					}
					bbs[12].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocImm {
						ctx.TrackImm(d8.Imm)
						ptrWord, _ := d8.Imm.RawWords()
						d282 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d282.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d282.Reg2, uint64(len(d8.Imm.String())))
						ctx.BindReg(d282.Reg, &d282)
						ctx.BindReg(d282.Reg2, &d282)
					} else {
						d282 = d8
					}
					d283 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("EUCLIDEAN")}
					if d283.Loc == LocImm {
						ctx.TrackImm(d283.Imm)
						ptrWord, _ := d283.Imm.RawWords()
						d284 = JITValueDesc{Loc: LocRegPair, Type: tagString, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.EmitMovRegImm64(d284.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(d284.Reg2, uint64(len(d283.Imm.String())))
						ctx.BindReg(d284.Reg, &d284)
						ctx.BindReg(d284.Reg2, &d284)
					} else {
						d284 = d283
					}
					d285 = ctx.EmitGoCallScalar(GoFuncAddr(JITStringEqual), []JITValueDesc{d282, d284}, 1)
					ctx.EmitAndRegImm32(d285.Reg, 1)
					d285.Type = tagBool
					ctx.BindReg(d285.Reg, &d285)
					d286 = d285
					ctx.EnsureDesc(&d286)
					if d286.Loc != LocImm && d286.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d286.Loc == LocImm {
						if d286.Imm.Bool() {
							return bbs[14].Render()
						}
						ctx.SyncDesc(&d14)
						if d14.Loc == LocReg || d14.Loc == LocFPReg {
							ctx.ProtectReg(d14.Reg)
						} else if d14.Loc == LocRegPair {
							ctx.ProtectReg(d14.Reg)
							ctx.ProtectReg(d14.Reg2)
						}
						d287 = d14
						if d287.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d287)
						ctx.EmitStoreToStack(d287, int32(bbs[4].PhiBase)+int32(0))
						if d14.Loc == LocReg || d14.Loc == LocFPReg {
							ctx.UnprotectReg(d14.Reg)
						} else if d14.Loc == LocRegPair {
							ctx.UnprotectReg(d14.Reg)
							ctx.UnprotectReg(d14.Reg2)
						}
						return bbs[4].Render()
					}
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d286.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl21)
					snap288 := d8
					snap289 := d9
					snap290 := d10
					snap291 := d11
					snap292 := d12
					snap293 := d13
					snap294 := d14
					snap295 := d15
					snap296 := d16
					snap297 := d17
					snap298 := d18
					snap299 := d19
					snap300 := d20
					snap301 := d21
					snap302 := d22
					snap303 := d55
					snap304 := d56
					snap305 := d57
					snap306 := d58
					snap307 := d59
					snap308 := d60
					snap309 := d61
					snap310 := d62
					snap311 := d63
					snap312 := d64
					snap313 := d91
					snap314 := d92
					snap315 := d93
					snap316 := d94
					snap317 := d125
					snap318 := d126
					snap319 := d127
					snap320 := d128
					snap321 := d129
					snap322 := d130
					snap323 := d131
					snap324 := d132
					snap325 := d133
					snap326 := d134
					snap327 := d135
					snap328 := d136
					snap329 := d137
					snap330 := d138
					snap331 := d139
					snap332 := d140
					snap333 := d141
					snap334 := d142
					snap335 := d143
					snap336 := d144
					snap337 := d145
					snap338 := d146
					snap339 := d147
					snap340 := d148
					snap341 := d149
					snap342 := d150
					snap343 := d207
					snap344 := d208
					snap345 := d209
					snap346 := d269
					snap347 := d270
					snap348 := d271
					snap349 := d272
					snap350 := d273
					snap351 := d274
					snap352 := d275
					snap353 := d276
					snap354 := d277
					snap355 := d278
					snap356 := d279
					snap357 := d280
					snap358 := d281
					snap359 := d282
					snap360 := d283
					snap361 := d284
					snap362 := d285
					snap363 := d286
					snap364 := d287
					alloc365 := ctx.SnapshotAllocState()
					ctx.MarkLabel(lbl21)
					ctx.SyncDesc(&d14)
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.ProtectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.ProtectReg(d14.Reg)
						ctx.ProtectReg(d14.Reg2)
					}
					d366 = d14
					if d366.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d366)
					ctx.EmitStoreToStack(d366, int32(bbs[4].PhiBase)+int32(0))
					if d14.Loc == LocReg || d14.Loc == LocFPReg {
						ctx.UnprotectReg(d14.Reg)
					} else if d14.Loc == LocRegPair {
						ctx.UnprotectReg(d14.Reg)
						ctx.UnprotectReg(d14.Reg2)
					}
					ctx.EmitJmp(lbl5)
					ctx.RestoreAllocState(alloc365)
					d8 = snap288
					d9 = snap289
					d10 = snap290
					d11 = snap291
					d12 = snap292
					d13 = snap293
					d14 = snap294
					d15 = snap295
					d16 = snap296
					d17 = snap297
					d18 = snap298
					d19 = snap299
					d20 = snap300
					d21 = snap301
					d22 = snap302
					d55 = snap303
					d56 = snap304
					d57 = snap305
					d58 = snap306
					d59 = snap307
					d60 = snap308
					d61 = snap309
					d62 = snap310
					d63 = snap311
					d64 = snap312
					d91 = snap313
					d92 = snap314
					d93 = snap315
					d94 = snap316
					d125 = snap317
					d126 = snap318
					d127 = snap319
					d128 = snap320
					d129 = snap321
					d130 = snap322
					d131 = snap323
					d132 = snap324
					d133 = snap325
					d134 = snap326
					d135 = snap327
					d136 = snap328
					d137 = snap329
					d138 = snap330
					d139 = snap331
					d140 = snap332
					d141 = snap333
					d142 = snap334
					d143 = snap335
					d144 = snap336
					d145 = snap337
					d146 = snap338
					d147 = snap339
					d148 = snap340
					d149 = snap341
					d150 = snap342
					d207 = snap343
					d208 = snap344
					d209 = snap345
					d269 = snap346
					d270 = snap347
					d271 = snap348
					d272 = snap349
					d273 = snap350
					d274 = snap351
					d275 = snap352
					d276 = snap353
					d277 = snap354
					d278 = snap355
					d279 = snap356
					d280 = snap357
					d281 = snap358
					d282 = snap359
					d283 = snap360
					d284 = snap361
					d285 = snap362
					d286 = snap363
					d287 = snap364
					if !bbs[4].Rendered {
						snap367 := d8
						snap368 := d9
						snap369 := d10
						snap370 := d11
						snap371 := d12
						snap372 := d13
						snap373 := d14
						snap374 := d15
						snap375 := d16
						snap376 := d17
						snap377 := d18
						snap378 := d19
						snap379 := d20
						snap380 := d21
						snap381 := d22
						snap382 := d55
						snap383 := d56
						snap384 := d57
						snap385 := d58
						snap386 := d59
						snap387 := d60
						snap388 := d61
						snap389 := d62
						snap390 := d63
						snap391 := d64
						snap392 := d91
						snap393 := d92
						snap394 := d93
						snap395 := d94
						snap396 := d125
						snap397 := d126
						snap398 := d127
						snap399 := d128
						snap400 := d129
						snap401 := d130
						snap402 := d131
						snap403 := d132
						snap404 := d133
						snap405 := d134
						snap406 := d135
						snap407 := d136
						snap408 := d137
						snap409 := d138
						snap410 := d139
						snap411 := d140
						snap412 := d141
						snap413 := d142
						snap414 := d143
						snap415 := d144
						snap416 := d145
						snap417 := d146
						snap418 := d147
						snap419 := d148
						snap420 := d149
						snap421 := d150
						snap422 := d207
						snap423 := d208
						snap424 := d209
						snap425 := d269
						snap426 := d270
						snap427 := d271
						snap428 := d272
						snap429 := d273
						snap430 := d274
						snap431 := d275
						snap432 := d276
						snap433 := d277
						snap434 := d278
						snap435 := d279
						snap436 := d280
						snap437 := d281
						snap438 := d282
						snap439 := d283
						snap440 := d284
						snap441 := d285
						snap442 := d286
						snap443 := d287
						snap444 := d366
						alloc445 := ctx.SnapshotAllocState()
						bbs[4].Render()
						ctx.RestoreAllocState(alloc445)
						d8 = snap367
						d9 = snap368
						d10 = snap369
						d11 = snap370
						d12 = snap371
						d13 = snap372
						d14 = snap373
						d15 = snap374
						d16 = snap375
						d17 = snap376
						d18 = snap377
						d19 = snap378
						d20 = snap379
						d21 = snap380
						d22 = snap381
						d55 = snap382
						d56 = snap383
						d57 = snap384
						d58 = snap385
						d59 = snap386
						d60 = snap387
						d61 = snap388
						d62 = snap389
						d63 = snap390
						d64 = snap391
						d91 = snap392
						d92 = snap393
						d93 = snap394
						d94 = snap395
						d125 = snap396
						d126 = snap397
						d127 = snap398
						d128 = snap399
						d129 = snap400
						d130 = snap401
						d131 = snap402
						d132 = snap403
						d133 = snap404
						d134 = snap405
						d135 = snap406
						d136 = snap407
						d137 = snap408
						d138 = snap409
						d139 = snap410
						d140 = snap411
						d141 = snap412
						d142 = snap413
						d143 = snap414
						d144 = snap415
						d145 = snap416
						d146 = snap417
						d147 = snap418
						d148 = snap419
						d149 = snap420
						d150 = snap421
						d207 = snap422
						d208 = snap423
						d209 = snap424
						d269 = snap425
						d270 = snap426
						d271 = snap427
						d272 = snap428
						d273 = snap429
						d274 = snap430
						d275 = snap431
						d276 = snap432
						d277 = snap433
						d278 = snap434
						d279 = snap435
						d280 = snap436
						d281 = snap437
						d282 = snap438
						d283 = snap439
						d284 = snap440
						d285 = snap441
						d286 = snap442
						d287 = snap443
						d366 = snap444
					}
					if !bbs[14].Rendered {
						return bbs[14].Render()
					}
					return result
					ctx.FreeDesc(&d285)
					return result
				}
				bbs[13].Render = func() JITValueDesc {
					if bbs[13].Rendered {
						ctx.EmitJmp(lbl14)
						return result
					}
					bbs[13].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					if d19.SliceSizeKnown {
						d446 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d446 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d446 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d446 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d446 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d446)
					ctx.EnsureDescsTogether(&d15, &d446)
					if d15.Loc == LocImm && d446.Loc == LocImm {
						d447 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.Int() < d446.Imm.Int())}
					} else if d446.Loc == LocImm {
						r20 := ctx.AllocRegExcept(d15.Reg)
						if d446.Imm.Int() >= -2147483648 && d446.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d15.Reg, int32(d446.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d446.Imm.Int()))
							ctx.EmitCmpInt64(d15.Reg, ctx.ScratchReg)
						}
						d447 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r20, Condition: CondSignedLess}
						ctx.BindReg(r20, &d447)
					} else if d15.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d15.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d446.Reg)
						d447 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r21, Condition: CondSignedLess}
						ctx.BindReg(r21, &d447)
					} else {
						r22 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpInt64(d15.Reg, d446.Reg)
						d447 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedLess}
						ctx.BindReg(r22, &d447)
					}
					ctx.FreeDesc(&d446)
					d448 = d447
					ctx.EnsureDesc(&d448)
					if d448.Loc != LocImm && d448.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d448.Loc == LocImm {
						if d448.Imm.Bool() {
							return bbs[11].Render()
						}
						return bbs[12].Render()
					}
					ctx.EmitJump(d448.Condition, lbl12)
					if bbs[12].Rendered {
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d447)
					ctx.FlushRegisterMoves()
					if !bbs[12].Rendered {
						snap449 := d8
						snap450 := d9
						snap451 := d10
						snap452 := d11
						snap453 := d12
						snap454 := d13
						snap455 := d14
						snap456 := d15
						snap457 := d16
						snap458 := d17
						snap459 := d18
						snap460 := d19
						snap461 := d20
						snap462 := d21
						snap463 := d22
						snap464 := d55
						snap465 := d56
						snap466 := d57
						snap467 := d58
						snap468 := d59
						snap469 := d60
						snap470 := d61
						snap471 := d62
						snap472 := d63
						snap473 := d64
						snap474 := d91
						snap475 := d92
						snap476 := d93
						snap477 := d94
						snap478 := d125
						snap479 := d126
						snap480 := d127
						snap481 := d128
						snap482 := d129
						snap483 := d130
						snap484 := d131
						snap485 := d132
						snap486 := d133
						snap487 := d134
						snap488 := d135
						snap489 := d136
						snap490 := d137
						snap491 := d138
						snap492 := d139
						snap493 := d140
						snap494 := d141
						snap495 := d142
						snap496 := d143
						snap497 := d144
						snap498 := d145
						snap499 := d146
						snap500 := d147
						snap501 := d148
						snap502 := d149
						snap503 := d150
						snap504 := d207
						snap505 := d208
						snap506 := d209
						snap507 := d269
						snap508 := d270
						snap509 := d271
						snap510 := d272
						snap511 := d273
						snap512 := d274
						snap513 := d275
						snap514 := d276
						snap515 := d277
						snap516 := d278
						snap517 := d279
						snap518 := d280
						snap519 := d281
						snap520 := d282
						snap521 := d283
						snap522 := d284
						snap523 := d285
						snap524 := d286
						snap525 := d287
						snap526 := d366
						snap527 := d446
						snap528 := d447
						snap529 := d448
						alloc530 := ctx.SnapshotAllocState()
						bbs[12].Render()
						ctx.RestoreAllocState(alloc530)
						d8 = snap449
						d9 = snap450
						d10 = snap451
						d11 = snap452
						d12 = snap453
						d13 = snap454
						d14 = snap455
						d15 = snap456
						d16 = snap457
						d17 = snap458
						d18 = snap459
						d19 = snap460
						d20 = snap461
						d21 = snap462
						d22 = snap463
						d55 = snap464
						d56 = snap465
						d57 = snap466
						d58 = snap467
						d59 = snap468
						d60 = snap469
						d61 = snap470
						d62 = snap471
						d63 = snap472
						d64 = snap473
						d91 = snap474
						d92 = snap475
						d93 = snap476
						d94 = snap477
						d125 = snap478
						d126 = snap479
						d127 = snap480
						d128 = snap481
						d129 = snap482
						d130 = snap483
						d131 = snap484
						d132 = snap485
						d133 = snap486
						d134 = snap487
						d135 = snap488
						d136 = snap489
						d137 = snap490
						d138 = snap491
						d139 = snap492
						d140 = snap493
						d141 = snap494
						d142 = snap495
						d143 = snap496
						d144 = snap497
						d145 = snap498
						d146 = snap499
						d147 = snap500
						d148 = snap501
						d149 = snap502
						d150 = snap503
						d207 = snap504
						d208 = snap505
						d209 = snap506
						d269 = snap507
						d270 = snap508
						d271 = snap509
						d272 = snap510
						d273 = snap511
						d274 = snap512
						d275 = snap513
						d276 = snap514
						d277 = snap515
						d278 = snap516
						d279 = snap517
						d280 = snap518
						d281 = snap519
						d282 = snap520
						d283 = snap521
						d284 = snap522
						d285 = snap523
						d286 = snap524
						d287 = snap525
						d366 = snap526
						d446 = snap527
						d447 = snap528
						d448 = snap529
					}
					if !bbs[11].Rendered {
						return bbs[11].Render()
					}
					return result
					return result
				}
				bbs[14].Render = func() JITValueDesc {
					if bbs[14].Rendered {
						ctx.EmitJmp(lbl15)
						return result
					}
					bbs[14].Rendered = true
					ctx.FlushRegisterMoves()
					bbpos_0_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d8 = JITValueDesc{Loc: LocStackPair, Type: tagString, StackOff: int32(phiBase0) + int32(0)}
					d9 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(phiBase0) + int32(16)}
					if phiHomeOK2 {
						d10 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r0, ID: 0}
					} else {
						d10 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(32)}
					}
					if phiHomeOK3 {
						d11 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r1, ID: 0}
					} else {
						d11 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(48)}
					}
					if phiHomeOK4 {
						d12 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r2, ID: 0}
					} else {
						d12 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(64)}
					}
					if phiHomeOK5 {
						d13 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3, ID: 0}
					} else {
						d13 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(80)}
					}
					if phiHomeOK6 {
						d14 = JITValueDesc{Loc: LocFPReg, Type: tagFloat, RegClass: JITRegisterClassFP, Reg: r4, ID: 0}
					} else {
						d14 = JITValueDesc{Loc: LocStack, Type: tagFloat, RegClass: JITRegisterClassFP, StackOff: int32(phiBase0) + int32(96)}
					}
					if phiHomeOK7 {
						d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5, ID: 0}
					} else {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(112)}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						d531 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d14.Imm.Float()))}
					} else {
						ctx.EnsureDesc(&d14)
						if d14.Loc == LocRegPair {
							ctx.FreeReg(d14.Reg)
							d532 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg2}
							ctx.BindReg(d14.Reg2, &d532)
							ctx.BindReg(d14.Reg2, &d532)
						} else {
							d532 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg}
							ctx.BindReg(d14.Reg, &d532)
							ctx.BindReg(d14.Reg, &d532)
						}
						d531 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d532}, 1)
						d531.Type = tagFloat
						ctx.BindReg(d531.Reg, &d531)
					}
					ctx.StabilizeDescForControlFlow(&d531)
					ctx.SyncDesc(&d531)
					if d531.Loc == LocReg || d531.Loc == LocFPReg {
						ctx.ProtectReg(d531.Reg)
					} else if d531.Loc == LocRegPair {
						ctx.ProtectReg(d531.Reg)
						ctx.ProtectReg(d531.Reg2)
					}
					d533 = d531
					if d533.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d533)
					ctx.EmitStoreToStack(d533, int32(bbs[4].PhiBase)+int32(0))
					if d531.Loc == LocReg || d531.Loc == LocFPReg {
						ctx.UnprotectReg(d531.Reg)
					} else if d531.Loc == LocRegPair {
						ctx.UnprotectReg(d531.Reg)
						ctx.UnprotectReg(d531.Reg2)
					}
					return bbs[4].Render()
					return result
				}
				returned := bbs[0].Render()
				if ctx.hasBooleanFlags(returned) {
					if resultRegsProtected {
						ctx.UnprotectReg(result.Reg2)
						ctx.UnprotectReg(result.Reg)
					}
					return returned
				}
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				result.Type = returnType
				result.ReturnTypeMerged = returnTypeSeen
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  80,
		},
	})
}
