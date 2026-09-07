/*
Copyright (C) 2026  Carl-Philip Hänsch

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

func groupAssocCapacity(inputLength int) int {
	const initialGroups = 32
	if inputLength < initialGroups {
		return inputLength
	}
	return initialGroups
}

func init_list_assoc_extra() {
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc")
			key := OptimizeProcToSerialFunction(a[1])
			reduce := OptimizeProcToSerialFunction(a[2])
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				result.ReduceValue(key(item), item, a[3], reduce)
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "groups list elements by key and reduces every group from a neutral value",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "key", Params: []*TypeDescriptor{{Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "reducer", Params: []*TypeDescriptor{{Kind: "any", Label: "current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "neutral"},
			},
			Return: &TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["group_assoc"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d26 JITValueDesc
				_ = d26
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var stackArray75 int32
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var callbackResultOff79 int32
				var d82 JITValueDesc
				_ = d82
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else if d4.Loc == LocImm {
						optimizedCallback6 := NewFunc(OptimizeProcToSerialFunction(d4.Imm))
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := NewFunc(OptimizeProcToSerialFunction(d7.Imm))
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						if d7.Loc == LocInputPair && int(d7.StackOff) < ctx.InputArgCount {
							d8 = ctx.RequestOptimizedCallback(int(d7.StackOff))
						} else {
							d8 = jitCopyScmerToPair(ctx, d7)
						}
					}
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.FreeDesc(&d7)
					var d10 JITValueDesc
					if d3.SliceSizeKnown {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d10 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					var d12 JITValueDesc
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitCmpRegImm32(d11.Reg, 32)
						d12 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					d13 = d12
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d13.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d12)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d14)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d11)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d15 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d15)
					ctx.BindReg(r1, &d15)
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d15}, 1)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d15)
					var d17 JITValueDesc
					if d3.SliceSizeKnown {
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d17)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps18 := PhiState{General: ps.General}
					ps18.OverlayValues = make([]JITValueDesc, 18)
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[3] = d3
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[7] = d7
					ps18.OverlayValues[8] = d8
					ps18.OverlayValues[10] = d10
					ps18.OverlayValues[11] = d11
					ps18.OverlayValues[12] = d12
					ps18.OverlayValues[13] = d13
					ps18.OverlayValues[14] = d14
					ps18.OverlayValues[15] = d15
					ps18.OverlayValues[16] = d16
					ps18.OverlayValues[17] = d17
					ps18.PhiValues = make([]JITValueDesc, 1)
					d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps18.PhiValues[0] = d19
					if ps18.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps18)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d20 := ps.PhiValues[0]
							ctx.EnsureDesc(&d20)
							ctx.EmitStoreToStack(d20, int32(bbs[1].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d21 JITValueDesc
					if d1.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d21)
					}
					if d21.Loc == LocReg && d1.Loc == LocReg && d21.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDescsTogether(&d21, &d17)
					var d22 JITValueDesc
					if d21.Loc == LocImm && d17.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() < d17.Imm.Int())}
					} else if d17.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d21.Reg)
						if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d21.Reg, int32(d17.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
							ctx.EmitCmpInt64(d21.Reg, RegR11)
						}
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d17.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d17.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d22)
					}
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							if ps.General {
							}
							ps24 := PhiState{General: ps.General}
							ps24.OverlayValues = make([]JITValueDesc, 24)
							ps24.OverlayValues[1] = d1
							ps24.OverlayValues[2] = d2
							ps24.OverlayValues[3] = d3
							ps24.OverlayValues[4] = d4
							ps24.OverlayValues[5] = d5
							ps24.OverlayValues[7] = d7
							ps24.OverlayValues[8] = d8
							ps24.OverlayValues[10] = d10
							ps24.OverlayValues[11] = d11
							ps24.OverlayValues[12] = d12
							ps24.OverlayValues[13] = d13
							ps24.OverlayValues[14] = d14
							ps24.OverlayValues[15] = d15
							ps24.OverlayValues[16] = d16
							ps24.OverlayValues[17] = d17
							ps24.OverlayValues[19] = d19
							ps24.OverlayValues[20] = d20
							ps24.OverlayValues[21] = d21
							ps24.OverlayValues[22] = d22
							ps24.OverlayValues[23] = d23
							return bbs[2].RenderPS(ps24)
						}
						if ps.General {
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 24)
						ps25.OverlayValues[1] = d1
						ps25.OverlayValues[2] = d2
						ps25.OverlayValues[3] = d3
						ps25.OverlayValues[4] = d4
						ps25.OverlayValues[5] = d5
						ps25.OverlayValues[7] = d7
						ps25.OverlayValues[8] = d8
						ps25.OverlayValues[10] = d10
						ps25.OverlayValues[11] = d11
						ps25.OverlayValues[12] = d12
						ps25.OverlayValues[13] = d13
						ps25.OverlayValues[14] = d14
						ps25.OverlayValues[15] = d15
						ps25.OverlayValues[16] = d16
						ps25.OverlayValues[17] = d17
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[23] = d23
						return bbs[3].RenderPS(ps25)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d26 := ps.PhiValues[0]
							ctx.EnsureDesc(&d26)
							ctx.EmitStoreToStack(d26, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d23.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d22)
					snap27 := d1
					snap28 := d2
					snap29 := d3
					snap30 := d4
					snap31 := d5
					snap32 := d7
					snap33 := d8
					snap34 := d10
					snap35 := d11
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d17
					snap42 := d19
					snap43 := d20
					snap44 := d21
					snap45 := d22
					snap46 := d23
					snap47 := d26
					alloc48 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc48)
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d7 = snap32
					d8 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d26 = snap47
					ctx.RestoreAllocState(alloc48)
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d7 = snap32
					d8 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d26 = snap47
					ps49 := PhiState{General: true}
					ps49.OverlayValues = make([]JITValueDesc, 27)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[7] = d7
					ps49.OverlayValues[8] = d8
					ps49.OverlayValues[10] = d10
					ps49.OverlayValues[11] = d11
					ps49.OverlayValues[12] = d12
					ps49.OverlayValues[13] = d13
					ps49.OverlayValues[14] = d14
					ps49.OverlayValues[15] = d15
					ps49.OverlayValues[16] = d16
					ps49.OverlayValues[17] = d17
					ps49.OverlayValues[19] = d19
					ps49.OverlayValues[20] = d20
					ps49.OverlayValues[21] = d21
					ps49.OverlayValues[22] = d22
					ps49.OverlayValues[23] = d23
					ps49.OverlayValues[26] = d26
					ps50 := PhiState{General: true}
					ps50.OverlayValues = make([]JITValueDesc, 27)
					ps50.OverlayValues[1] = d1
					ps50.OverlayValues[2] = d2
					ps50.OverlayValues[3] = d3
					ps50.OverlayValues[4] = d4
					ps50.OverlayValues[5] = d5
					ps50.OverlayValues[7] = d7
					ps50.OverlayValues[8] = d8
					ps50.OverlayValues[10] = d10
					ps50.OverlayValues[11] = d11
					ps50.OverlayValues[12] = d12
					ps50.OverlayValues[13] = d13
					ps50.OverlayValues[14] = d14
					ps50.OverlayValues[15] = d15
					ps50.OverlayValues[16] = d16
					ps50.OverlayValues[17] = d17
					ps50.OverlayValues[19] = d19
					ps50.OverlayValues[20] = d20
					ps50.OverlayValues[21] = d21
					ps50.OverlayValues[22] = d22
					ps50.OverlayValues[23] = d23
					ps50.OverlayValues[26] = d26
					snap51 := d1
					snap52 := d2
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d7
					snap57 := d8
					snap58 := d10
					snap59 := d11
					snap60 := d12
					snap61 := d13
					snap62 := d14
					snap63 := d15
					snap64 := d16
					snap65 := d17
					snap66 := d19
					snap67 := d20
					snap68 := d21
					snap69 := d22
					snap70 := d23
					snap71 := d26
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps50)
					}
					ctx.RestoreAllocState(alloc72)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d7 = snap56
					d8 = snap57
					d10 = snap58
					d11 = snap59
					d12 = snap60
					d13 = snap61
					d14 = snap62
					d15 = snap63
					d16 = snap64
					d17 = snap65
					d19 = snap66
					d20 = snap67
					d21 = snap68
					d22 = snap69
					d23 = snap70
					d26 = snap71
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps49)
					}
					return result
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					d74 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d74)
					r5 := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitMovRegMem(r5, d74.Reg, 8)
					ctx.EmitMovRegMem(d74.Reg, d74.Reg, 0)
					d73 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d74.Reg, Reg2: r5}
					ctx.BindReg(d74.Reg, &d73)
					ctx.BindReg(r5, &d73)
					stackArray75 = ctx.AllocStack(int32(16))
					_ = stackArray75
					ctx.SyncDesc(&d73)
					ctx.EmitStoreScmerToStack(d73, int32(stackArray75)+int32(0))
					d76 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d76
					callbackArgs78 := make([]JITValueDesc, 1)
					callbackArgs78[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray75) + 0}
					var d77 JITValueDesc
					callbackResultOff79 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff79))
					ctx.FreeDesc(&d76)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs80 := ctx.StabilizeCallbackArgs(callbackArgs78)
						ctx.ReclaimUntrackedRegs()
						outerRegs81 := ctx.PreserveOuterRegs()
						d77 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs80, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff79), ID: 0})
						ctx.RestoreOuterRegs(outerRegs81)
						ctx.ReclaimUntrackedRegs()
					} else {
						d82, knownBuiltin83 := jitEmitKnownDeclaration(ctx, d5, callbackArgs78, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff79), ID: 0})
						if knownBuiltin83 {
							d77 = d82
						} else {
							ctx.Coverage.DynamicCalls++
							d84 := jitCopyScmerToPair(ctx, d5)
							d77 = jitEmitDynamicCallableAt(ctx, d84, callbackArgs78, int32(stackArray75), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff79), ID: 0})
						}
					}
					d85 = args[3]
					d85.ID = 0
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d77 = JITPrepareScmerGoArg(ctx, d77)
					d73 = JITPrepareScmerGoArg(ctx, d73)
					d85 = JITPrepareScmerGoArg(ctx, d85)
					if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d77)
					ctx.SyncDesc(&d73)
					ctx.SyncDesc(&d85)
					ctx.SyncDesc(&d8)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).ReduceValue), []JITValueDesc{d16, d77, d73, d85, d8})
					ctx.FreeDesc(&d77)
					ctx.FreeDesc(&d73)
					ctx.FreeDesc(&d85)
					if ps.General {
						ctx.SyncDesc(&d21)
						if d21.Loc == LocReg {
							ctx.ProtectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.ProtectReg(d21.Reg)
							ctx.ProtectReg(d21.Reg2)
						}
						d86 = d21
						if d86.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d86)
						ctx.EmitStoreToStack(d86, int32(bbs[1].PhiBase)+int32(0))
						if d21.Loc == LocReg {
							ctx.UnprotectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.UnprotectReg(d21.Reg)
							ctx.UnprotectReg(d21.Reg2)
						}
					}
					ps87 := PhiState{General: ps.General}
					ps87.OverlayValues = make([]JITValueDesc, 87)
					ps87.OverlayValues[1] = d1
					ps87.OverlayValues[2] = d2
					ps87.OverlayValues[3] = d3
					ps87.OverlayValues[4] = d4
					ps87.OverlayValues[5] = d5
					ps87.OverlayValues[7] = d7
					ps87.OverlayValues[8] = d8
					ps87.OverlayValues[10] = d10
					ps87.OverlayValues[11] = d11
					ps87.OverlayValues[12] = d12
					ps87.OverlayValues[13] = d13
					ps87.OverlayValues[14] = d14
					ps87.OverlayValues[15] = d15
					ps87.OverlayValues[16] = d16
					ps87.OverlayValues[17] = d17
					ps87.OverlayValues[19] = d19
					ps87.OverlayValues[20] = d20
					ps87.OverlayValues[21] = d21
					ps87.OverlayValues[22] = d22
					ps87.OverlayValues[23] = d23
					ps87.OverlayValues[26] = d26
					ps87.OverlayValues[73] = d73
					ps87.OverlayValues[74] = d74
					ps87.OverlayValues[76] = d76
					ps87.OverlayValues[77] = d77
					ps87.OverlayValues[82] = d82
					ps87.OverlayValues[84] = d84
					ps87.OverlayValues[85] = d85
					ps87.OverlayValues[86] = d86
					ps87.PhiValues = make([]JITValueDesc, 1)
					d88 = d21
					ps87.PhiValues[0] = d88
					if ps87.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps87)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					ctx.ReclaimUntrackedRegs()
					var d89 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d89 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r6}
						ctx.BindReg(d16.Reg, &d89)
						ctx.BindReg(r6, &d89)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d89)
						ctx.BindReg(r6, &d89)
						d16.Loc = LocNone
					}
					ctx.SyncDesc(&d89)
					if d89.Loc == LocRegPair || d89.Loc == LocStackPair || d89.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d89, &result)
						result.Type = d89.Type
					} else {
						switch d89.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d89)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d89)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d89)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d89, &result)
							result.Type = d89.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps90 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps90)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCallbacks: true,
			JITInlineCost:      35,
		},
		Optimize: optimizeGroupAssoc,
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_append",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_append")
			key := OptimizeProcToSerialFunction(a[1])
			value := OptimizeProcToSerialFunction(a[2])
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				result.AppendValue(key(item), value(NewNil(), item))
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only append reduction into grouped lists",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "key", Params: []*TypeDescriptor{{Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "value", Params: []*TypeDescriptor{{Kind: "any", Label: "unused_current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength, Element: &TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength}},
			Const:     true,
			Forbidden: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["group_assoc_append"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d26 JITValueDesc
				_ = d26
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var stackArray75 int32
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var callbackResultOff79 int32
				var d82 JITValueDesc
				_ = d82
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var stackArray86 int32
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var callbackResultOff90 int32
				var d93 JITValueDesc
				_ = d93
				var d95 JITValueDesc
				_ = d95
				var d96 JITValueDesc
				_ = d96
				var d98 JITValueDesc
				_ = d98
				var d99 JITValueDesc
				_ = d99
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else if d4.Loc == LocImm {
						optimizedCallback6 := NewFunc(OptimizeProcToSerialFunction(d4.Imm))
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := NewFunc(OptimizeProcToSerialFunction(d7.Imm))
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						if d7.Loc == LocInputPair && int(d7.StackOff) < ctx.InputArgCount {
							d8 = ctx.RequestOptimizedCallback(int(d7.StackOff))
						} else {
							d8 = jitCopyScmerToPair(ctx, d7)
						}
					}
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.FreeDesc(&d7)
					var d10 JITValueDesc
					if d3.SliceSizeKnown {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d10 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					var d12 JITValueDesc
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitCmpRegImm32(d11.Reg, 32)
						d12 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					d13 = d12
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d13.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d12)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d14)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d11)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d15 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d15)
					ctx.BindReg(r1, &d15)
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d15}, 1)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d15)
					var d17 JITValueDesc
					if d3.SliceSizeKnown {
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d17)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps18 := PhiState{General: ps.General}
					ps18.OverlayValues = make([]JITValueDesc, 18)
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[3] = d3
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[7] = d7
					ps18.OverlayValues[8] = d8
					ps18.OverlayValues[10] = d10
					ps18.OverlayValues[11] = d11
					ps18.OverlayValues[12] = d12
					ps18.OverlayValues[13] = d13
					ps18.OverlayValues[14] = d14
					ps18.OverlayValues[15] = d15
					ps18.OverlayValues[16] = d16
					ps18.OverlayValues[17] = d17
					ps18.PhiValues = make([]JITValueDesc, 1)
					d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps18.PhiValues[0] = d19
					if ps18.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps18)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d20 := ps.PhiValues[0]
							ctx.EnsureDesc(&d20)
							ctx.EmitStoreToStack(d20, int32(bbs[1].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d21 JITValueDesc
					if d1.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d21)
					}
					if d21.Loc == LocReg && d1.Loc == LocReg && d21.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDescsTogether(&d21, &d17)
					var d22 JITValueDesc
					if d21.Loc == LocImm && d17.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() < d17.Imm.Int())}
					} else if d17.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d21.Reg)
						if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d21.Reg, int32(d17.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
							ctx.EmitCmpInt64(d21.Reg, RegR11)
						}
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d17.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d17.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d22)
					}
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							if ps.General {
							}
							ps24 := PhiState{General: ps.General}
							ps24.OverlayValues = make([]JITValueDesc, 24)
							ps24.OverlayValues[1] = d1
							ps24.OverlayValues[2] = d2
							ps24.OverlayValues[3] = d3
							ps24.OverlayValues[4] = d4
							ps24.OverlayValues[5] = d5
							ps24.OverlayValues[7] = d7
							ps24.OverlayValues[8] = d8
							ps24.OverlayValues[10] = d10
							ps24.OverlayValues[11] = d11
							ps24.OverlayValues[12] = d12
							ps24.OverlayValues[13] = d13
							ps24.OverlayValues[14] = d14
							ps24.OverlayValues[15] = d15
							ps24.OverlayValues[16] = d16
							ps24.OverlayValues[17] = d17
							ps24.OverlayValues[19] = d19
							ps24.OverlayValues[20] = d20
							ps24.OverlayValues[21] = d21
							ps24.OverlayValues[22] = d22
							ps24.OverlayValues[23] = d23
							return bbs[2].RenderPS(ps24)
						}
						if ps.General {
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 24)
						ps25.OverlayValues[1] = d1
						ps25.OverlayValues[2] = d2
						ps25.OverlayValues[3] = d3
						ps25.OverlayValues[4] = d4
						ps25.OverlayValues[5] = d5
						ps25.OverlayValues[7] = d7
						ps25.OverlayValues[8] = d8
						ps25.OverlayValues[10] = d10
						ps25.OverlayValues[11] = d11
						ps25.OverlayValues[12] = d12
						ps25.OverlayValues[13] = d13
						ps25.OverlayValues[14] = d14
						ps25.OverlayValues[15] = d15
						ps25.OverlayValues[16] = d16
						ps25.OverlayValues[17] = d17
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[23] = d23
						return bbs[3].RenderPS(ps25)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d26 := ps.PhiValues[0]
							ctx.EnsureDesc(&d26)
							ctx.EmitStoreToStack(d26, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d23.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d22)
					snap27 := d1
					snap28 := d2
					snap29 := d3
					snap30 := d4
					snap31 := d5
					snap32 := d7
					snap33 := d8
					snap34 := d10
					snap35 := d11
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d17
					snap42 := d19
					snap43 := d20
					snap44 := d21
					snap45 := d22
					snap46 := d23
					snap47 := d26
					alloc48 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc48)
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d7 = snap32
					d8 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d26 = snap47
					ctx.RestoreAllocState(alloc48)
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d7 = snap32
					d8 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d26 = snap47
					ps49 := PhiState{General: true}
					ps49.OverlayValues = make([]JITValueDesc, 27)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[7] = d7
					ps49.OverlayValues[8] = d8
					ps49.OverlayValues[10] = d10
					ps49.OverlayValues[11] = d11
					ps49.OverlayValues[12] = d12
					ps49.OverlayValues[13] = d13
					ps49.OverlayValues[14] = d14
					ps49.OverlayValues[15] = d15
					ps49.OverlayValues[16] = d16
					ps49.OverlayValues[17] = d17
					ps49.OverlayValues[19] = d19
					ps49.OverlayValues[20] = d20
					ps49.OverlayValues[21] = d21
					ps49.OverlayValues[22] = d22
					ps49.OverlayValues[23] = d23
					ps49.OverlayValues[26] = d26
					ps50 := PhiState{General: true}
					ps50.OverlayValues = make([]JITValueDesc, 27)
					ps50.OverlayValues[1] = d1
					ps50.OverlayValues[2] = d2
					ps50.OverlayValues[3] = d3
					ps50.OverlayValues[4] = d4
					ps50.OverlayValues[5] = d5
					ps50.OverlayValues[7] = d7
					ps50.OverlayValues[8] = d8
					ps50.OverlayValues[10] = d10
					ps50.OverlayValues[11] = d11
					ps50.OverlayValues[12] = d12
					ps50.OverlayValues[13] = d13
					ps50.OverlayValues[14] = d14
					ps50.OverlayValues[15] = d15
					ps50.OverlayValues[16] = d16
					ps50.OverlayValues[17] = d17
					ps50.OverlayValues[19] = d19
					ps50.OverlayValues[20] = d20
					ps50.OverlayValues[21] = d21
					ps50.OverlayValues[22] = d22
					ps50.OverlayValues[23] = d23
					ps50.OverlayValues[26] = d26
					snap51 := d1
					snap52 := d2
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d7
					snap57 := d8
					snap58 := d10
					snap59 := d11
					snap60 := d12
					snap61 := d13
					snap62 := d14
					snap63 := d15
					snap64 := d16
					snap65 := d17
					snap66 := d19
					snap67 := d20
					snap68 := d21
					snap69 := d22
					snap70 := d23
					snap71 := d26
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps50)
					}
					ctx.RestoreAllocState(alloc72)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d7 = snap56
					d8 = snap57
					d10 = snap58
					d11 = snap59
					d12 = snap60
					d13 = snap61
					d14 = snap62
					d15 = snap63
					d16 = snap64
					d17 = snap65
					d19 = snap66
					d20 = snap67
					d21 = snap68
					d22 = snap69
					d23 = snap70
					d26 = snap71
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps49)
					}
					return result
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					d74 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d74)
					r5 := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitMovRegMem(r5, d74.Reg, 8)
					ctx.EmitMovRegMem(d74.Reg, d74.Reg, 0)
					d73 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d74.Reg, Reg2: r5}
					ctx.BindReg(d74.Reg, &d73)
					ctx.BindReg(r5, &d73)
					stackArray75 = ctx.AllocStack(int32(16))
					_ = stackArray75
					ctx.SyncDesc(&d73)
					ctx.EmitStoreScmerToStack(d73, int32(stackArray75)+int32(0))
					d76 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d76
					callbackArgs78 := make([]JITValueDesc, 1)
					callbackArgs78[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray75) + 0}
					var d77 JITValueDesc
					callbackResultOff79 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff79))
					ctx.FreeDesc(&d76)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs80 := ctx.StabilizeCallbackArgs(callbackArgs78)
						ctx.ReclaimUntrackedRegs()
						outerRegs81 := ctx.PreserveOuterRegs()
						d77 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs80, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff79), ID: 0})
						ctx.RestoreOuterRegs(outerRegs81)
						ctx.ReclaimUntrackedRegs()
					} else {
						d82, knownBuiltin83 := jitEmitKnownDeclaration(ctx, d5, callbackArgs78, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff79), ID: 0})
						if knownBuiltin83 {
							d77 = d82
						} else {
							ctx.Coverage.DynamicCalls++
							d84 := jitCopyScmerToPair(ctx, d5)
							d77 = jitEmitDynamicCallableAt(ctx, d84, callbackArgs78, int32(stackArray75), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff79), ID: 0})
						}
					}
					d85 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray86 = ctx.AllocStack(int32(32))
					_ = stackArray86
					ctx.SyncDesc(&d85)
					ctx.EmitStoreScmerToStack(d85, int32(stackArray86)+int32(0))
					ctx.FreeDesc(&d85)
					ctx.SyncDesc(&d73)
					ctx.EmitStoreScmerToStack(d73, int32(stackArray86)+int32(16))
					ctx.FreeDesc(&d73)
					d87 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d87
					callbackArgs89 := make([]JITValueDesc, 2)
					callbackArgs89[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray86) + 0}
					callbackArgs89[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray86) + 16}
					var d88 JITValueDesc
					callbackResultOff90 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff90))
					ctx.FreeDesc(&d87)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs91 := ctx.StabilizeCallbackArgs(callbackArgs89)
						ctx.ReclaimUntrackedRegs()
						outerRegs92 := ctx.PreserveOuterRegs()
						d88 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs91, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff90), ID: 0})
						ctx.RestoreOuterRegs(outerRegs92)
						ctx.ReclaimUntrackedRegs()
					} else {
						d93, knownBuiltin94 := jitEmitKnownDeclaration(ctx, d8, callbackArgs89, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff90), ID: 0})
						if knownBuiltin94 {
							d88 = d93
						} else {
							ctx.Coverage.DynamicCalls++
							d95 := jitCopyScmerToPair(ctx, d8)
							d88 = jitEmitDynamicCallableAt(ctx, d95, callbackArgs89, int32(stackArray86), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff90), ID: 0})
						}
					}
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d77 = JITPrepareScmerGoArg(ctx, d77)
					d88 = JITPrepareScmerGoArg(ctx, d88)
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d77)
					ctx.SyncDesc(&d88)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d16, d77, d88})
					ctx.FreeDesc(&d77)
					ctx.FreeDesc(&d88)
					if ps.General {
						ctx.SyncDesc(&d21)
						if d21.Loc == LocReg {
							ctx.ProtectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.ProtectReg(d21.Reg)
							ctx.ProtectReg(d21.Reg2)
						}
						d96 = d21
						if d96.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d96)
						ctx.EmitStoreToStack(d96, int32(bbs[1].PhiBase)+int32(0))
						if d21.Loc == LocReg {
							ctx.UnprotectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.UnprotectReg(d21.Reg)
							ctx.UnprotectReg(d21.Reg2)
						}
					}
					ps97 := PhiState{General: ps.General}
					ps97.OverlayValues = make([]JITValueDesc, 97)
					ps97.OverlayValues[1] = d1
					ps97.OverlayValues[2] = d2
					ps97.OverlayValues[3] = d3
					ps97.OverlayValues[4] = d4
					ps97.OverlayValues[5] = d5
					ps97.OverlayValues[7] = d7
					ps97.OverlayValues[8] = d8
					ps97.OverlayValues[10] = d10
					ps97.OverlayValues[11] = d11
					ps97.OverlayValues[12] = d12
					ps97.OverlayValues[13] = d13
					ps97.OverlayValues[14] = d14
					ps97.OverlayValues[15] = d15
					ps97.OverlayValues[16] = d16
					ps97.OverlayValues[17] = d17
					ps97.OverlayValues[19] = d19
					ps97.OverlayValues[20] = d20
					ps97.OverlayValues[21] = d21
					ps97.OverlayValues[22] = d22
					ps97.OverlayValues[23] = d23
					ps97.OverlayValues[26] = d26
					ps97.OverlayValues[73] = d73
					ps97.OverlayValues[74] = d74
					ps97.OverlayValues[76] = d76
					ps97.OverlayValues[77] = d77
					ps97.OverlayValues[82] = d82
					ps97.OverlayValues[84] = d84
					ps97.OverlayValues[85] = d85
					ps97.OverlayValues[87] = d87
					ps97.OverlayValues[88] = d88
					ps97.OverlayValues[93] = d93
					ps97.OverlayValues[95] = d95
					ps97.OverlayValues[96] = d96
					ps97.PhiValues = make([]JITValueDesc, 1)
					d98 = d21
					ps97.PhiValues[0] = d98
					if ps97.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps97)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					ctx.ReclaimUntrackedRegs()
					var d99 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d99 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r6}
						ctx.BindReg(d16.Reg, &d99)
						ctx.BindReg(r6, &d99)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d99)
						ctx.BindReg(r6, &d99)
						d16.Loc = LocNone
					}
					ctx.SyncDesc(&d99)
					if d99.Loc == LocRegPair || d99.Loc == LocStackPair || d99.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d99, &result)
						result.Type = d99.Type
					} else {
						switch d99.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d99)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d99)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d99)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d99, &result)
							result.Type = d99.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps100 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps100)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCallbacks: true,
			JITInlineCost:      41,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_append_reduce",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_append_reduce")
			key := OptimizeProcToSerialFunction(a[1])
			value := OptimizeProcToSerialFunction(a[2])
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				result.AppendValue(key(NewNil(), item), value(NewNil(), item))
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only append reduction from a normalized two-parameter reducer",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "key", Params: []*TypeDescriptor{{Kind: "any", Label: "unused_current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "value", Params: []*TypeDescriptor{{Kind: "any", Label: "unused_current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength, Element: &TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength}},
			Const:     true,
			Forbidden: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["group_assoc_append_reduce"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d26 JITValueDesc
				_ = d26
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var stackArray76 int32
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var callbackResultOff80 int32
				var d83 JITValueDesc
				_ = d83
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var stackArray87 int32
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var callbackResultOff91 int32
				var d94 JITValueDesc
				_ = d94
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else if d4.Loc == LocImm {
						optimizedCallback6 := NewFunc(OptimizeProcToSerialFunction(d4.Imm))
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := NewFunc(OptimizeProcToSerialFunction(d7.Imm))
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						if d7.Loc == LocInputPair && int(d7.StackOff) < ctx.InputArgCount {
							d8 = ctx.RequestOptimizedCallback(int(d7.StackOff))
						} else {
							d8 = jitCopyScmerToPair(ctx, d7)
						}
					}
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.FreeDesc(&d7)
					var d10 JITValueDesc
					if d3.SliceSizeKnown {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d10 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d10)
					d11 = d10
					_ = d11
					ctx.StabilizeDescForControlFlow(&d11)
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					var d12 JITValueDesc
					if d11.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d11.Reg)
						ctx.EmitCmpRegImm32(d11.Reg, 32)
						d12 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					d13 = d12
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d13.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d12)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d14)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d11)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d15 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d15)
					ctx.BindReg(r1, &d15)
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d15)
					d16 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d15}, 1)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d15)
					var d17 JITValueDesc
					if d3.SliceSizeKnown {
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d17)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps18 := PhiState{General: ps.General}
					ps18.OverlayValues = make([]JITValueDesc, 18)
					ps18.OverlayValues[1] = d1
					ps18.OverlayValues[2] = d2
					ps18.OverlayValues[3] = d3
					ps18.OverlayValues[4] = d4
					ps18.OverlayValues[5] = d5
					ps18.OverlayValues[7] = d7
					ps18.OverlayValues[8] = d8
					ps18.OverlayValues[10] = d10
					ps18.OverlayValues[11] = d11
					ps18.OverlayValues[12] = d12
					ps18.OverlayValues[13] = d13
					ps18.OverlayValues[14] = d14
					ps18.OverlayValues[15] = d15
					ps18.OverlayValues[16] = d16
					ps18.OverlayValues[17] = d17
					ps18.PhiValues = make([]JITValueDesc, 1)
					d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps18.PhiValues[0] = d19
					if ps18.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps18)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d20 := ps.PhiValues[0]
							ctx.EnsureDesc(&d20)
							ctx.EmitStoreToStack(d20, int32(bbs[1].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d21 JITValueDesc
					if d1.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d21 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d21)
					}
					if d21.Loc == LocReg && d1.Loc == LocReg && d21.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDescsTogether(&d21, &d17)
					var d22 JITValueDesc
					if d21.Loc == LocImm && d17.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() < d17.Imm.Int())}
					} else if d17.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d21.Reg)
						if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d21.Reg, int32(d17.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d17.Imm.Int()))
							ctx.EmitCmpInt64(d21.Reg, RegR11)
						}
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d17.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d17.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d22)
					}
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							if ps.General {
							}
							ps24 := PhiState{General: ps.General}
							ps24.OverlayValues = make([]JITValueDesc, 24)
							ps24.OverlayValues[1] = d1
							ps24.OverlayValues[2] = d2
							ps24.OverlayValues[3] = d3
							ps24.OverlayValues[4] = d4
							ps24.OverlayValues[5] = d5
							ps24.OverlayValues[7] = d7
							ps24.OverlayValues[8] = d8
							ps24.OverlayValues[10] = d10
							ps24.OverlayValues[11] = d11
							ps24.OverlayValues[12] = d12
							ps24.OverlayValues[13] = d13
							ps24.OverlayValues[14] = d14
							ps24.OverlayValues[15] = d15
							ps24.OverlayValues[16] = d16
							ps24.OverlayValues[17] = d17
							ps24.OverlayValues[19] = d19
							ps24.OverlayValues[20] = d20
							ps24.OverlayValues[21] = d21
							ps24.OverlayValues[22] = d22
							ps24.OverlayValues[23] = d23
							return bbs[2].RenderPS(ps24)
						}
						if ps.General {
						}
						ps25 := PhiState{General: ps.General}
						ps25.OverlayValues = make([]JITValueDesc, 24)
						ps25.OverlayValues[1] = d1
						ps25.OverlayValues[2] = d2
						ps25.OverlayValues[3] = d3
						ps25.OverlayValues[4] = d4
						ps25.OverlayValues[5] = d5
						ps25.OverlayValues[7] = d7
						ps25.OverlayValues[8] = d8
						ps25.OverlayValues[10] = d10
						ps25.OverlayValues[11] = d11
						ps25.OverlayValues[12] = d12
						ps25.OverlayValues[13] = d13
						ps25.OverlayValues[14] = d14
						ps25.OverlayValues[15] = d15
						ps25.OverlayValues[16] = d16
						ps25.OverlayValues[17] = d17
						ps25.OverlayValues[19] = d19
						ps25.OverlayValues[20] = d20
						ps25.OverlayValues[21] = d21
						ps25.OverlayValues[22] = d22
						ps25.OverlayValues[23] = d23
						return bbs[3].RenderPS(ps25)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d26 := ps.PhiValues[0]
							ctx.EnsureDesc(&d26)
							ctx.EmitStoreToStack(d26, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d23.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d22)
					snap27 := d1
					snap28 := d2
					snap29 := d3
					snap30 := d4
					snap31 := d5
					snap32 := d7
					snap33 := d8
					snap34 := d10
					snap35 := d11
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d17
					snap42 := d19
					snap43 := d20
					snap44 := d21
					snap45 := d22
					snap46 := d23
					snap47 := d26
					alloc48 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc48)
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d7 = snap32
					d8 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d26 = snap47
					ctx.RestoreAllocState(alloc48)
					d1 = snap27
					d2 = snap28
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d7 = snap32
					d8 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d26 = snap47
					ps49 := PhiState{General: true}
					ps49.OverlayValues = make([]JITValueDesc, 27)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[7] = d7
					ps49.OverlayValues[8] = d8
					ps49.OverlayValues[10] = d10
					ps49.OverlayValues[11] = d11
					ps49.OverlayValues[12] = d12
					ps49.OverlayValues[13] = d13
					ps49.OverlayValues[14] = d14
					ps49.OverlayValues[15] = d15
					ps49.OverlayValues[16] = d16
					ps49.OverlayValues[17] = d17
					ps49.OverlayValues[19] = d19
					ps49.OverlayValues[20] = d20
					ps49.OverlayValues[21] = d21
					ps49.OverlayValues[22] = d22
					ps49.OverlayValues[23] = d23
					ps49.OverlayValues[26] = d26
					ps50 := PhiState{General: true}
					ps50.OverlayValues = make([]JITValueDesc, 27)
					ps50.OverlayValues[1] = d1
					ps50.OverlayValues[2] = d2
					ps50.OverlayValues[3] = d3
					ps50.OverlayValues[4] = d4
					ps50.OverlayValues[5] = d5
					ps50.OverlayValues[7] = d7
					ps50.OverlayValues[8] = d8
					ps50.OverlayValues[10] = d10
					ps50.OverlayValues[11] = d11
					ps50.OverlayValues[12] = d12
					ps50.OverlayValues[13] = d13
					ps50.OverlayValues[14] = d14
					ps50.OverlayValues[15] = d15
					ps50.OverlayValues[16] = d16
					ps50.OverlayValues[17] = d17
					ps50.OverlayValues[19] = d19
					ps50.OverlayValues[20] = d20
					ps50.OverlayValues[21] = d21
					ps50.OverlayValues[22] = d22
					ps50.OverlayValues[23] = d23
					ps50.OverlayValues[26] = d26
					snap51 := d1
					snap52 := d2
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d7
					snap57 := d8
					snap58 := d10
					snap59 := d11
					snap60 := d12
					snap61 := d13
					snap62 := d14
					snap63 := d15
					snap64 := d16
					snap65 := d17
					snap66 := d19
					snap67 := d20
					snap68 := d21
					snap69 := d22
					snap70 := d23
					snap71 := d26
					alloc72 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps50)
					}
					ctx.RestoreAllocState(alloc72)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d7 = snap56
					d8 = snap57
					d10 = snap58
					d11 = snap59
					d12 = snap60
					d13 = snap61
					d14 = snap62
					d15 = snap63
					d16 = snap64
					d17 = snap65
					d19 = snap66
					d20 = snap67
					d21 = snap68
					d22 = snap69
					d23 = snap70
					d26 = snap71
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps49)
					}
					return result
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					d74 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d74)
					r5 := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitMovRegMem(r5, d74.Reg, 8)
					ctx.EmitMovRegMem(d74.Reg, d74.Reg, 0)
					d73 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d74.Reg, Reg2: r5}
					ctx.BindReg(d74.Reg, &d73)
					ctx.BindReg(r5, &d73)
					d75 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray76 = ctx.AllocStack(int32(32))
					_ = stackArray76
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray76)+int32(0))
					ctx.FreeDesc(&d75)
					ctx.SyncDesc(&d73)
					ctx.EmitStoreScmerToStack(d73, int32(stackArray76)+int32(16))
					d77 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d77
					callbackArgs79 := make([]JITValueDesc, 2)
					callbackArgs79[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray76) + 0}
					callbackArgs79[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray76) + 16}
					var d78 JITValueDesc
					callbackResultOff80 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff80))
					ctx.FreeDesc(&d77)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs81 := ctx.StabilizeCallbackArgs(callbackArgs79)
						ctx.ReclaimUntrackedRegs()
						outerRegs82 := ctx.PreserveOuterRegs()
						d78 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs81, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff80), ID: 0})
						ctx.RestoreOuterRegs(outerRegs82)
						ctx.ReclaimUntrackedRegs()
					} else {
						d83, knownBuiltin84 := jitEmitKnownDeclaration(ctx, d5, callbackArgs79, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff80), ID: 0})
						if knownBuiltin84 {
							d78 = d83
						} else {
							ctx.Coverage.DynamicCalls++
							d85 := jitCopyScmerToPair(ctx, d5)
							d78 = jitEmitDynamicCallableAt(ctx, d85, callbackArgs79, int32(stackArray76), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff80), ID: 0})
						}
					}
					d86 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray87 = ctx.AllocStack(int32(32))
					_ = stackArray87
					ctx.SyncDesc(&d86)
					ctx.EmitStoreScmerToStack(d86, int32(stackArray87)+int32(0))
					ctx.FreeDesc(&d86)
					ctx.SyncDesc(&d73)
					ctx.EmitStoreScmerToStack(d73, int32(stackArray87)+int32(16))
					ctx.FreeDesc(&d73)
					d88 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d88
					callbackArgs90 := make([]JITValueDesc, 2)
					callbackArgs90[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray87) + 0}
					callbackArgs90[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray87) + 16}
					var d89 JITValueDesc
					callbackResultOff91 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff91))
					ctx.FreeDesc(&d88)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs92 := ctx.StabilizeCallbackArgs(callbackArgs90)
						ctx.ReclaimUntrackedRegs()
						outerRegs93 := ctx.PreserveOuterRegs()
						d89 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs92, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff91), ID: 0})
						ctx.RestoreOuterRegs(outerRegs93)
						ctx.ReclaimUntrackedRegs()
					} else {
						d94, knownBuiltin95 := jitEmitKnownDeclaration(ctx, d8, callbackArgs90, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff91), ID: 0})
						if knownBuiltin95 {
							d89 = d94
						} else {
							ctx.Coverage.DynamicCalls++
							d96 := jitCopyScmerToPair(ctx, d8)
							d89 = jitEmitDynamicCallableAt(ctx, d96, callbackArgs90, int32(stackArray87), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff91), ID: 0})
						}
					}
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d78 = JITPrepareScmerGoArg(ctx, d78)
					d89 = JITPrepareScmerGoArg(ctx, d89)
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d78)
					ctx.SyncDesc(&d89)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d16, d78, d89})
					ctx.FreeDesc(&d78)
					ctx.FreeDesc(&d89)
					if ps.General {
						ctx.SyncDesc(&d21)
						if d21.Loc == LocReg {
							ctx.ProtectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.ProtectReg(d21.Reg)
							ctx.ProtectReg(d21.Reg2)
						}
						d97 = d21
						if d97.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d97)
						ctx.EmitStoreToStack(d97, int32(bbs[1].PhiBase)+int32(0))
						if d21.Loc == LocReg {
							ctx.UnprotectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.UnprotectReg(d21.Reg)
							ctx.UnprotectReg(d21.Reg2)
						}
					}
					ps98 := PhiState{General: ps.General}
					ps98.OverlayValues = make([]JITValueDesc, 98)
					ps98.OverlayValues[1] = d1
					ps98.OverlayValues[2] = d2
					ps98.OverlayValues[3] = d3
					ps98.OverlayValues[4] = d4
					ps98.OverlayValues[5] = d5
					ps98.OverlayValues[7] = d7
					ps98.OverlayValues[8] = d8
					ps98.OverlayValues[10] = d10
					ps98.OverlayValues[11] = d11
					ps98.OverlayValues[12] = d12
					ps98.OverlayValues[13] = d13
					ps98.OverlayValues[14] = d14
					ps98.OverlayValues[15] = d15
					ps98.OverlayValues[16] = d16
					ps98.OverlayValues[17] = d17
					ps98.OverlayValues[19] = d19
					ps98.OverlayValues[20] = d20
					ps98.OverlayValues[21] = d21
					ps98.OverlayValues[22] = d22
					ps98.OverlayValues[23] = d23
					ps98.OverlayValues[26] = d26
					ps98.OverlayValues[73] = d73
					ps98.OverlayValues[74] = d74
					ps98.OverlayValues[75] = d75
					ps98.OverlayValues[77] = d77
					ps98.OverlayValues[78] = d78
					ps98.OverlayValues[83] = d83
					ps98.OverlayValues[85] = d85
					ps98.OverlayValues[86] = d86
					ps98.OverlayValues[88] = d88
					ps98.OverlayValues[89] = d89
					ps98.OverlayValues[94] = d94
					ps98.OverlayValues[96] = d96
					ps98.OverlayValues[97] = d97
					ps98.PhiValues = make([]JITValueDesc, 1)
					d99 = d21
					ps98.PhiValues[0] = d99
					if ps98.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps98)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
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
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					ctx.ReclaimUntrackedRegs()
					var d100 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d100 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r6}
						ctx.BindReg(d16.Reg, &d100)
						ctx.BindReg(r6, &d100)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d100)
						ctx.BindReg(r6, &d100)
						d16.Loc = LocNone
					}
					ctx.SyncDesc(&d100)
					if d100.Loc == LocRegPair || d100.Loc == LocStackPair || d100.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d100, &result)
						result.Type = d100.Type
					} else {
						switch d100.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d100)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d100)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d100)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d100, &result)
							result.Type = d100.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps101 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps101)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCallbacks: true,
			JITInlineCost:      44,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_multi_append_reduce",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_multi_append_reduce")
			legs := (len(a) - 1) / 2
			keys := make([]func(...Scmer) Scmer, legs)
			values := make([]func(...Scmer) Scmer, legs)
			for i := 0; i < legs; i++ {
				keys[i] = OptimizeProcToSerialFunction(a[1+2*i])
				values[i] = OptimizeProcToSerialFunction(a[2+2*i])
			}
			result := NewFastDictValue(groupAssocCapacity(len(input) * legs))
			for _, item := range input {
				for i := 0; i < legs; i++ {
					result.AppendValue(keys[i](NewNil(), item), values[i](NewNil(), item))
				}
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only multi-leg append reduction: applies an ordered sequence of (key, value) extractor pairs to every item, preserving item-major/leg-minor insertion order so results match the equivalent chain of set_assoc/append calls",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "extractor...", Description: "alternating key/value extractor functions, one pair per leg", Variadic: true, Params: []*TypeDescriptor{{Kind: "any", Label: "unused_current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength, Element: &TypeDescriptor{Kind: "list", Transfer: true, Length: UnknownLength}},
			Const:     true,
			Forbidden: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				declaration := declarations["group_assoc_multi_append_reduce"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_multi_count_reduce",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_multi_count_reduce")
			legs := len(a) - 1
			keys := make([]func(...Scmer) Scmer, legs)
			for i := 0; i < legs; i++ {
				keys[i] = OptimizeProcToSerialFunction(a[1+i])
			}
			result := NewFastDictValue(groupAssocCapacity(len(input) * legs))
			for _, item := range input {
				for i := 0; i < legs; i++ {
					result.IncrementCount(keys[i](NewNil(), item))
				}
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only multi-leg counting reduction: increments a count per key extractor per item in one pass",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "key...", Description: "one key extractor per leg", Variadic: true, Params: []*TypeDescriptor{{Kind: "any", Label: "unused_current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength, Element: &TypeDescriptor{Kind: "int", Transfer: true}},
			Const:     true,
			Forbidden: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				ctx.Coverage.NativeCalls++
				declaration := declarations["group_assoc_multi_count_reduce"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_count",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_count")
			key := OptimizeProcToSerialFunction(a[1])
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				result.IncrementCount(key(item))
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only integer counting reduction by key",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "key", Params: []*TypeDescriptor{{Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength, Element: &TypeDescriptor{Kind: "int", Transfer: true}},
			Const:     true,
			Forbidden: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["group_assoc_count"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
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
				var d23 JITValueDesc
				_ = d23
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var stackArray68 int32
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var callbackResultOff72 int32
				var d75 JITValueDesc
				_ = d75
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else if d4.Loc == LocImm {
						optimizedCallback6 := NewFunc(OptimizeProcToSerialFunction(d4.Imm))
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					var d7 JITValueDesc
					if d3.SliceSizeKnown {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d7)
					d8 = d7
					_ = d8
					ctx.StabilizeDescForControlFlow(&d8)
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					var d9 JITValueDesc
					if d8.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpRegImm32(d8.Reg, 32)
						d9 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d9)
					}
					ctx.ReclaimUntrackedRegs()
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d10.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d9)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d11)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d8)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d12 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d12)
					ctx.BindReg(r1, &d12)
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d12)
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d12}, 1)
					ctx.StabilizeDescForControlFlow(&d13)
					ctx.FreeDesc(&d12)
					var d14 JITValueDesc
					if d3.SliceSizeKnown {
						d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d14 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d14)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps15 := PhiState{General: ps.General}
					ps15.OverlayValues = make([]JITValueDesc, 15)
					ps15.OverlayValues[1] = d1
					ps15.OverlayValues[2] = d2
					ps15.OverlayValues[3] = d3
					ps15.OverlayValues[4] = d4
					ps15.OverlayValues[5] = d5
					ps15.OverlayValues[7] = d7
					ps15.OverlayValues[8] = d8
					ps15.OverlayValues[9] = d9
					ps15.OverlayValues[10] = d10
					ps15.OverlayValues[11] = d11
					ps15.OverlayValues[12] = d12
					ps15.OverlayValues[13] = d13
					ps15.OverlayValues[14] = d14
					ps15.PhiValues = make([]JITValueDesc, 1)
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps15.PhiValues[0] = d16
					if ps15.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps15)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d17 := ps.PhiValues[0]
							ctx.EnsureDesc(&d17)
							ctx.EmitStoreToStack(d17, int32(bbs[1].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d18 JITValueDesc
					if d1.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d18)
					}
					if d18.Loc == LocReg && d1.Loc == LocReg && d18.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDescsTogether(&d18, &d14)
					var d19 JITValueDesc
					if d18.Loc == LocImm && d14.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d18.Imm.Int() < d14.Imm.Int())}
					} else if d14.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d18.Reg)
						if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d18.Reg, int32(d14.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
							ctx.EmitCmpInt64(d18.Reg, RegR11)
						}
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d19)
					} else if d18.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d18.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d14.Reg)
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d19)
					} else {
						r4 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpInt64(d18.Reg, d14.Reg)
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d19)
					}
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							if ps.General {
							}
							ps21 := PhiState{General: ps.General}
							ps21.OverlayValues = make([]JITValueDesc, 21)
							ps21.OverlayValues[1] = d1
							ps21.OverlayValues[2] = d2
							ps21.OverlayValues[3] = d3
							ps21.OverlayValues[4] = d4
							ps21.OverlayValues[5] = d5
							ps21.OverlayValues[7] = d7
							ps21.OverlayValues[8] = d8
							ps21.OverlayValues[9] = d9
							ps21.OverlayValues[10] = d10
							ps21.OverlayValues[11] = d11
							ps21.OverlayValues[12] = d12
							ps21.OverlayValues[13] = d13
							ps21.OverlayValues[14] = d14
							ps21.OverlayValues[16] = d16
							ps21.OverlayValues[17] = d17
							ps21.OverlayValues[18] = d18
							ps21.OverlayValues[19] = d19
							ps21.OverlayValues[20] = d20
							return bbs[2].RenderPS(ps21)
						}
						if ps.General {
						}
						ps22 := PhiState{General: ps.General}
						ps22.OverlayValues = make([]JITValueDesc, 21)
						ps22.OverlayValues[1] = d1
						ps22.OverlayValues[2] = d2
						ps22.OverlayValues[3] = d3
						ps22.OverlayValues[4] = d4
						ps22.OverlayValues[5] = d5
						ps22.OverlayValues[7] = d7
						ps22.OverlayValues[8] = d8
						ps22.OverlayValues[9] = d9
						ps22.OverlayValues[10] = d10
						ps22.OverlayValues[11] = d11
						ps22.OverlayValues[12] = d12
						ps22.OverlayValues[13] = d13
						ps22.OverlayValues[14] = d14
						ps22.OverlayValues[16] = d16
						ps22.OverlayValues[17] = d17
						ps22.OverlayValues[18] = d18
						ps22.OverlayValues[19] = d19
						ps22.OverlayValues[20] = d20
						return bbs[3].RenderPS(ps22)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d23 := ps.PhiValues[0]
							ctx.EnsureDesc(&d23)
							ctx.EmitStoreToStack(d23, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d20.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d19)
					snap24 := d1
					snap25 := d2
					snap26 := d3
					snap27 := d4
					snap28 := d5
					snap29 := d7
					snap30 := d8
					snap31 := d9
					snap32 := d10
					snap33 := d11
					snap34 := d12
					snap35 := d13
					snap36 := d14
					snap37 := d16
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d23
					alloc43 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc43)
					d1 = snap24
					d2 = snap25
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d7 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d23 = snap42
					ctx.RestoreAllocState(alloc43)
					d1 = snap24
					d2 = snap25
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d7 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d23 = snap42
					ps44 := PhiState{General: true}
					ps44.OverlayValues = make([]JITValueDesc, 24)
					ps44.OverlayValues[1] = d1
					ps44.OverlayValues[2] = d2
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[4] = d4
					ps44.OverlayValues[5] = d5
					ps44.OverlayValues[7] = d7
					ps44.OverlayValues[8] = d8
					ps44.OverlayValues[9] = d9
					ps44.OverlayValues[10] = d10
					ps44.OverlayValues[11] = d11
					ps44.OverlayValues[12] = d12
					ps44.OverlayValues[13] = d13
					ps44.OverlayValues[14] = d14
					ps44.OverlayValues[16] = d16
					ps44.OverlayValues[17] = d17
					ps44.OverlayValues[18] = d18
					ps44.OverlayValues[19] = d19
					ps44.OverlayValues[20] = d20
					ps44.OverlayValues[23] = d23
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 24)
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[7] = d7
					ps45.OverlayValues[8] = d8
					ps45.OverlayValues[9] = d9
					ps45.OverlayValues[10] = d10
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[12] = d12
					ps45.OverlayValues[13] = d13
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[16] = d16
					ps45.OverlayValues[17] = d17
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[19] = d19
					ps45.OverlayValues[20] = d20
					ps45.OverlayValues[23] = d23
					snap46 := d1
					snap47 := d2
					snap48 := d3
					snap49 := d4
					snap50 := d5
					snap51 := d7
					snap52 := d8
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d12
					snap57 := d13
					snap58 := d14
					snap59 := d16
					snap60 := d17
					snap61 := d18
					snap62 := d19
					snap63 := d20
					snap64 := d23
					alloc65 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps45)
					}
					ctx.RestoreAllocState(alloc65)
					d1 = snap46
					d2 = snap47
					d3 = snap48
					d4 = snap49
					d5 = snap50
					d7 = snap51
					d8 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d12 = snap56
					d13 = snap57
					d14 = snap58
					d16 = snap59
					d17 = snap60
					d18 = snap61
					d19 = snap62
					d20 = snap63
					d23 = snap64
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps44)
					}
					return result
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					d67 = ctx.EmitSliceElementAddress(&d3, &d18, 16)
					ctx.EnsureDesc(&d67)
					r5 := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitMovRegMem(r5, d67.Reg, 8)
					ctx.EmitMovRegMem(d67.Reg, d67.Reg, 0)
					d66 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d67.Reg, Reg2: r5}
					ctx.BindReg(d67.Reg, &d66)
					ctx.BindReg(r5, &d66)
					stackArray68 = ctx.AllocStack(int32(16))
					_ = stackArray68
					ctx.SyncDesc(&d66)
					ctx.EmitStoreScmerToStack(d66, int32(stackArray68)+int32(0))
					ctx.FreeDesc(&d66)
					d69 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d69
					callbackArgs71 := make([]JITValueDesc, 1)
					callbackArgs71[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray68) + 0}
					var d70 JITValueDesc
					callbackResultOff72 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff72))
					ctx.FreeDesc(&d69)
					ctx.StabilizeDescAcrossNestedCall(&d18)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs73 := ctx.StabilizeCallbackArgs(callbackArgs71)
						ctx.ReclaimUntrackedRegs()
						outerRegs74 := ctx.PreserveOuterRegs()
						d70 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs73, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff72), ID: 0})
						ctx.RestoreOuterRegs(outerRegs74)
						ctx.ReclaimUntrackedRegs()
					} else {
						d75, knownBuiltin76 := jitEmitKnownDeclaration(ctx, d5, callbackArgs71, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff72), ID: 0})
						if knownBuiltin76 {
							d70 = d75
						} else {
							ctx.Coverage.DynamicCalls++
							d77 := jitCopyScmerToPair(ctx, d5)
							d70 = jitEmitDynamicCallableAt(ctx, d77, callbackArgs71, int32(stackArray68), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff72), ID: 0})
						}
					}
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d70 = JITPrepareScmerGoArg(ctx, d70)
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d70)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d13, d70})
					ctx.FreeDesc(&d70)
					if ps.General {
						ctx.SyncDesc(&d18)
						if d18.Loc == LocReg {
							ctx.ProtectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.ProtectReg(d18.Reg)
							ctx.ProtectReg(d18.Reg2)
						}
						d78 = d18
						if d78.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d78)
						ctx.EmitStoreToStack(d78, int32(bbs[1].PhiBase)+int32(0))
						if d18.Loc == LocReg {
							ctx.UnprotectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.UnprotectReg(d18.Reg)
							ctx.UnprotectReg(d18.Reg2)
						}
					}
					ps79 := PhiState{General: ps.General}
					ps79.OverlayValues = make([]JITValueDesc, 79)
					ps79.OverlayValues[1] = d1
					ps79.OverlayValues[2] = d2
					ps79.OverlayValues[3] = d3
					ps79.OverlayValues[4] = d4
					ps79.OverlayValues[5] = d5
					ps79.OverlayValues[7] = d7
					ps79.OverlayValues[8] = d8
					ps79.OverlayValues[9] = d9
					ps79.OverlayValues[10] = d10
					ps79.OverlayValues[11] = d11
					ps79.OverlayValues[12] = d12
					ps79.OverlayValues[13] = d13
					ps79.OverlayValues[14] = d14
					ps79.OverlayValues[16] = d16
					ps79.OverlayValues[17] = d17
					ps79.OverlayValues[18] = d18
					ps79.OverlayValues[19] = d19
					ps79.OverlayValues[20] = d20
					ps79.OverlayValues[23] = d23
					ps79.OverlayValues[66] = d66
					ps79.OverlayValues[67] = d67
					ps79.OverlayValues[69] = d69
					ps79.OverlayValues[70] = d70
					ps79.OverlayValues[75] = d75
					ps79.OverlayValues[77] = d77
					ps79.OverlayValues[78] = d78
					ps79.PhiValues = make([]JITValueDesc, 1)
					d80 = d18
					ps79.PhiValues[0] = d80
					if ps79.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps79)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					ctx.ReclaimUntrackedRegs()
					var d81 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d81 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d13.Reg, Reg2: r6}
						ctx.BindReg(d13.Reg, &d81)
						ctx.BindReg(r6, &d81)
						ctx.TransferReg(d13.Reg)
						ctx.BindReg(d13.Reg, &d81)
						ctx.BindReg(r6, &d81)
						d13.Loc = LocNone
					}
					ctx.SyncDesc(&d81)
					if d81.Loc == LocRegPair || d81.Loc == LocStackPair || d81.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d81, &result)
						result.Type = d81.Type
					} else {
						switch d81.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d81)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d81)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d81)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d81, &result)
							result.Type = d81.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps82 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps82)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCallbacks: true,
			JITInlineCost:      30,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_count_reduce",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_count_reduce")
			key := OptimizeProcToSerialFunction(a[1])
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				result.IncrementCount(key(NewNil(), item))
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only counting from a normalized two-parameter reducer",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "key", Params: []*TypeDescriptor{{Kind: "any", Label: "unused_current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return:    &TypeDescriptor{Kind: "assoc", Transfer: true, Length: UnknownLength, Element: &TypeDescriptor{Kind: "int", Transfer: true}},
			Const:     true,
			Forbidden: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["group_assoc_count_reduce"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
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
				var d23 JITValueDesc
				_ = d23
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var stackArray69 int32
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var callbackResultOff73 int32
				var d76 JITValueDesc
				_ = d76
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Loc == LocLambdaTemplate {
						d5 = d4
					} else if d4.Loc == LocImm {
						optimizedCallback6 := NewFunc(OptimizeProcToSerialFunction(d4.Imm))
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					var d7 JITValueDesc
					if d3.SliceSizeKnown {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d7)
					d8 = d7
					_ = d8
					ctx.StabilizeDescForControlFlow(&d8)
					lbl5 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl6 := ctx.ReserveLabel()
					_ = lbl6
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl7 := ctx.ReserveLabel()
					_ = lbl7
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl6)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					var d9 JITValueDesc
					if d8.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d8.Reg)
						ctx.EmitCmpRegImm32(d8.Reg, 32)
						d9 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d9)
					}
					ctx.ReclaimUntrackedRegs()
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d10.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d9)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d11)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d8)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d12 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d12)
					ctx.BindReg(r1, &d12)
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d12)
					d13 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d12}, 1)
					ctx.StabilizeDescForControlFlow(&d13)
					ctx.FreeDesc(&d12)
					var d14 JITValueDesc
					if d3.SliceSizeKnown {
						d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d14 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d14)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps15 := PhiState{General: ps.General}
					ps15.OverlayValues = make([]JITValueDesc, 15)
					ps15.OverlayValues[1] = d1
					ps15.OverlayValues[2] = d2
					ps15.OverlayValues[3] = d3
					ps15.OverlayValues[4] = d4
					ps15.OverlayValues[5] = d5
					ps15.OverlayValues[7] = d7
					ps15.OverlayValues[8] = d8
					ps15.OverlayValues[9] = d9
					ps15.OverlayValues[10] = d10
					ps15.OverlayValues[11] = d11
					ps15.OverlayValues[12] = d12
					ps15.OverlayValues[13] = d13
					ps15.OverlayValues[14] = d14
					ps15.PhiValues = make([]JITValueDesc, 1)
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps15.PhiValues[0] = d16
					if ps15.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps15)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d17 := ps.PhiValues[0]
							ctx.EnsureDesc(&d17)
							ctx.EmitStoreToStack(d17, int32(bbs[1].PhiBase)+int32(0))
						}
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d18 JITValueDesc
					if d1.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d18 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d18)
					}
					if d18.Loc == LocReg && d1.Loc == LocReg && d18.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDescsTogether(&d18, &d14)
					var d19 JITValueDesc
					if d18.Loc == LocImm && d14.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d18.Imm.Int() < d14.Imm.Int())}
					} else if d14.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d18.Reg)
						if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d18.Reg, int32(d14.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d14.Imm.Int()))
							ctx.EmitCmpInt64(d18.Reg, RegR11)
						}
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d19)
					} else if d18.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d18.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d14.Reg)
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d19)
					} else {
						r4 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpInt64(d18.Reg, d14.Reg)
						d19 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d19)
					}
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							if ps.General {
							}
							ps21 := PhiState{General: ps.General}
							ps21.OverlayValues = make([]JITValueDesc, 21)
							ps21.OverlayValues[1] = d1
							ps21.OverlayValues[2] = d2
							ps21.OverlayValues[3] = d3
							ps21.OverlayValues[4] = d4
							ps21.OverlayValues[5] = d5
							ps21.OverlayValues[7] = d7
							ps21.OverlayValues[8] = d8
							ps21.OverlayValues[9] = d9
							ps21.OverlayValues[10] = d10
							ps21.OverlayValues[11] = d11
							ps21.OverlayValues[12] = d12
							ps21.OverlayValues[13] = d13
							ps21.OverlayValues[14] = d14
							ps21.OverlayValues[16] = d16
							ps21.OverlayValues[17] = d17
							ps21.OverlayValues[18] = d18
							ps21.OverlayValues[19] = d19
							ps21.OverlayValues[20] = d20
							return bbs[2].RenderPS(ps21)
						}
						if ps.General {
						}
						ps22 := PhiState{General: ps.General}
						ps22.OverlayValues = make([]JITValueDesc, 21)
						ps22.OverlayValues[1] = d1
						ps22.OverlayValues[2] = d2
						ps22.OverlayValues[3] = d3
						ps22.OverlayValues[4] = d4
						ps22.OverlayValues[5] = d5
						ps22.OverlayValues[7] = d7
						ps22.OverlayValues[8] = d8
						ps22.OverlayValues[9] = d9
						ps22.OverlayValues[10] = d10
						ps22.OverlayValues[11] = d11
						ps22.OverlayValues[12] = d12
						ps22.OverlayValues[13] = d13
						ps22.OverlayValues[14] = d14
						ps22.OverlayValues[16] = d16
						ps22.OverlayValues[17] = d17
						ps22.OverlayValues[18] = d18
						ps22.OverlayValues[19] = d19
						ps22.OverlayValues[20] = d20
						return bbs[3].RenderPS(ps22)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d23 := ps.PhiValues[0]
							ctx.EnsureDesc(&d23)
							ctx.EmitStoreToStack(d23, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d20.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d19)
					snap24 := d1
					snap25 := d2
					snap26 := d3
					snap27 := d4
					snap28 := d5
					snap29 := d7
					snap30 := d8
					snap31 := d9
					snap32 := d10
					snap33 := d11
					snap34 := d12
					snap35 := d13
					snap36 := d14
					snap37 := d16
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d23
					alloc43 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc43)
					d1 = snap24
					d2 = snap25
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d7 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d23 = snap42
					ctx.RestoreAllocState(alloc43)
					d1 = snap24
					d2 = snap25
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d7 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d16 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d23 = snap42
					ps44 := PhiState{General: true}
					ps44.OverlayValues = make([]JITValueDesc, 24)
					ps44.OverlayValues[1] = d1
					ps44.OverlayValues[2] = d2
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[4] = d4
					ps44.OverlayValues[5] = d5
					ps44.OverlayValues[7] = d7
					ps44.OverlayValues[8] = d8
					ps44.OverlayValues[9] = d9
					ps44.OverlayValues[10] = d10
					ps44.OverlayValues[11] = d11
					ps44.OverlayValues[12] = d12
					ps44.OverlayValues[13] = d13
					ps44.OverlayValues[14] = d14
					ps44.OverlayValues[16] = d16
					ps44.OverlayValues[17] = d17
					ps44.OverlayValues[18] = d18
					ps44.OverlayValues[19] = d19
					ps44.OverlayValues[20] = d20
					ps44.OverlayValues[23] = d23
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 24)
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[7] = d7
					ps45.OverlayValues[8] = d8
					ps45.OverlayValues[9] = d9
					ps45.OverlayValues[10] = d10
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[12] = d12
					ps45.OverlayValues[13] = d13
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[16] = d16
					ps45.OverlayValues[17] = d17
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[19] = d19
					ps45.OverlayValues[20] = d20
					ps45.OverlayValues[23] = d23
					snap46 := d1
					snap47 := d2
					snap48 := d3
					snap49 := d4
					snap50 := d5
					snap51 := d7
					snap52 := d8
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d12
					snap57 := d13
					snap58 := d14
					snap59 := d16
					snap60 := d17
					snap61 := d18
					snap62 := d19
					snap63 := d20
					snap64 := d23
					alloc65 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps45)
					}
					ctx.RestoreAllocState(alloc65)
					d1 = snap46
					d2 = snap47
					d3 = snap48
					d4 = snap49
					d5 = snap50
					d7 = snap51
					d8 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d12 = snap56
					d13 = snap57
					d14 = snap58
					d16 = snap59
					d17 = snap60
					d18 = snap61
					d19 = snap62
					d20 = snap63
					d23 = snap64
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps44)
					}
					return result
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					d67 = ctx.EmitSliceElementAddress(&d3, &d18, 16)
					ctx.EnsureDesc(&d67)
					r5 := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitMovRegMem(r5, d67.Reg, 8)
					ctx.EmitMovRegMem(d67.Reg, d67.Reg, 0)
					d66 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d67.Reg, Reg2: r5}
					ctx.BindReg(d67.Reg, &d66)
					ctx.BindReg(r5, &d66)
					d68 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray69 = ctx.AllocStack(int32(32))
					_ = stackArray69
					ctx.SyncDesc(&d68)
					ctx.EmitStoreScmerToStack(d68, int32(stackArray69)+int32(0))
					ctx.FreeDesc(&d68)
					ctx.SyncDesc(&d66)
					ctx.EmitStoreScmerToStack(d66, int32(stackArray69)+int32(16))
					ctx.FreeDesc(&d66)
					d70 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d70
					callbackArgs72 := make([]JITValueDesc, 2)
					callbackArgs72[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray69) + 0}
					callbackArgs72[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray69) + 16}
					var d71 JITValueDesc
					callbackResultOff73 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff73))
					ctx.FreeDesc(&d70)
					ctx.StabilizeDescAcrossNestedCall(&d18)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs74 := ctx.StabilizeCallbackArgs(callbackArgs72)
						ctx.ReclaimUntrackedRegs()
						outerRegs75 := ctx.PreserveOuterRegs()
						d71 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs74, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff73), ID: 0})
						ctx.RestoreOuterRegs(outerRegs75)
						ctx.ReclaimUntrackedRegs()
					} else {
						d76, knownBuiltin77 := jitEmitKnownDeclaration(ctx, d5, callbackArgs72, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff73), ID: 0})
						if knownBuiltin77 {
							d71 = d76
						} else {
							ctx.Coverage.DynamicCalls++
							d78 := jitCopyScmerToPair(ctx, d5)
							d71 = jitEmitDynamicCallableAt(ctx, d78, callbackArgs72, int32(stackArray69), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff73), ID: 0})
						}
					}
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d71 = JITPrepareScmerGoArg(ctx, d71)
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d71)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d13, d71})
					ctx.FreeDesc(&d71)
					if ps.General {
						ctx.SyncDesc(&d18)
						if d18.Loc == LocReg {
							ctx.ProtectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.ProtectReg(d18.Reg)
							ctx.ProtectReg(d18.Reg2)
						}
						d79 = d18
						if d79.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d79)
						ctx.EmitStoreToStack(d79, int32(bbs[1].PhiBase)+int32(0))
						if d18.Loc == LocReg {
							ctx.UnprotectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.UnprotectReg(d18.Reg)
							ctx.UnprotectReg(d18.Reg2)
						}
					}
					ps80 := PhiState{General: ps.General}
					ps80.OverlayValues = make([]JITValueDesc, 80)
					ps80.OverlayValues[1] = d1
					ps80.OverlayValues[2] = d2
					ps80.OverlayValues[3] = d3
					ps80.OverlayValues[4] = d4
					ps80.OverlayValues[5] = d5
					ps80.OverlayValues[7] = d7
					ps80.OverlayValues[8] = d8
					ps80.OverlayValues[9] = d9
					ps80.OverlayValues[10] = d10
					ps80.OverlayValues[11] = d11
					ps80.OverlayValues[12] = d12
					ps80.OverlayValues[13] = d13
					ps80.OverlayValues[14] = d14
					ps80.OverlayValues[16] = d16
					ps80.OverlayValues[17] = d17
					ps80.OverlayValues[18] = d18
					ps80.OverlayValues[19] = d19
					ps80.OverlayValues[20] = d20
					ps80.OverlayValues[23] = d23
					ps80.OverlayValues[66] = d66
					ps80.OverlayValues[67] = d67
					ps80.OverlayValues[68] = d68
					ps80.OverlayValues[70] = d70
					ps80.OverlayValues[71] = d71
					ps80.OverlayValues[76] = d76
					ps80.OverlayValues[78] = d78
					ps80.OverlayValues[79] = d79
					ps80.PhiValues = make([]JITValueDesc, 1)
					d81 = d18
					ps80.PhiValues[0] = d81
					if ps80.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps80)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
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
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					ctx.ReclaimUntrackedRegs()
					var d82 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d82 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d13.Reg, Reg2: r6}
						ctx.BindReg(d13.Reg, &d82)
						ctx.BindReg(r6, &d82)
						ctx.TransferReg(d13.Reg)
						ctx.BindReg(d13.Reg, &d82)
						ctx.BindReg(r6, &d82)
						d13.Loc = LocNone
					}
					ctx.SyncDesc(&d82)
					if d82.Loc == LocRegPair || d82.Loc == LocStackPair || d82.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d82, &result)
						result.Type = d82.Type
					} else {
						switch d82.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d82)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d82)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d82)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d82, &result)
							result.Type = d82.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps83 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps83)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				if resultRegsProtected {
					ctx.UnprotectReg(result.Reg2)
					ctx.UnprotectReg(result.Reg)
				}
				return result
			},
			JITInlineCallbacks: true,
			JITInlineCost:      33,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mapkey_assoc",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc"])
			result := NewSlice(nil)
			if slice, fd := asAssoc(a[0], "mapkey_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					result = setAssoc(result, fn(slice[i], slice[i+1]), slice[i+1])
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool {
					result = setAssoc(result, fn(k, v), v)
					return true
				})
			}
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "returns a mapped dictionary according to a map function\nValues stay the same but keys are mapped.",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "dictionary whose keys have to be mapped", NoEscape: true},
				{Kind: "func", Label: "map", Description: "computes a replacement key for each dictionary entry", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "existing key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "any", Label: "new_key", Description: "replacement key"}},
			},
			Return: FreshAlloc,
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["mapkey_assoc"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
		},
		Optimize:                 FirstParameterMutable("mapkey_assoc_mut"),
		OptimizeFirstArgTransfer: true,
	})
	Declare(&Globalenv, &Declaration{
		Name: "mapkey_assoc_mut",

		Fn: func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc_mut"])
			slice, fd := asAssoc(a[0], "mapkey_assoc_mut")
			if fd == nil {
				orig := append([]Scmer{}, slice...)
				result := NewSlice(slice[:0])
				for i := 0; i < len(orig); i += 2 {
					result = setAssoc(result, fn(orig[i], orig[i+1]), orig[i+1])
				}
				return result
			}
			result := NewSlice(nil)
			fd.Iterate(func(k, v Scmer) bool {
				result = setAssoc(result, fn(k, v), v)
				return true
			})
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "optimizer-only key remap for dictionaries",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "dict", Description: "owned dictionary whose keys have to be remapped"},
				{Kind: "func", Label: "map", Description: "computes a replacement key for each dictionary entry", Params: []*TypeDescriptor{{Kind: "string", Label: "key", Description: "existing key"}, {Kind: "any", Label: "value", Description: "entry value"}}, Return: &TypeDescriptor{Kind: "any", Label: "new_key", Description: "replacement key"}},
			},
			Return:    FreshAlloc,
			Const:     true,
			Forbidden: true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				declaration := declarations["mapkey_assoc_mut"]
				return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
		},
	})
}
