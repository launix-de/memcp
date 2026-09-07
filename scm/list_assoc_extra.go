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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
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
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d28 JITValueDesc
				_ = d28
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var stackArray77 int32
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var callbackResultOff81 int32
				var d84 JITValueDesc
				_ = d84
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 12}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d3
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d4}, 3)
					}
					ctx.BindReg(d5.Reg, &d5)
					ctx.BindReg(d5.Reg2, &d5)
					ctx.BindReg(d5.Reg3, &d5)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d6 = args[1]
					d6.ID = 0
					var d7 JITValueDesc
					if d6.Loc == LocLambdaTemplate {
						d7 = d6
					} else if d6.Loc == LocImm {
						optimizedCallback8 := NewFunc(OptimizeProcToSerialFunction(d6.Imm))
						ctx.TrackImm(optimizedCallback8)
						d7 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback8, Rooted: true}
					} else {
						if d6.Loc == LocInputPair && int(d6.StackOff) < ctx.InputArgCount {
							d7 = ctx.RequestOptimizedCallback(int(d6.StackOff))
						} else {
							d7 = jitCopyScmerToPair(ctx, d6)
						}
					}
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.FreeDesc(&d6)
					d9 = args[2]
					d9.ID = 0
					var d10 JITValueDesc
					if d9.Loc == LocLambdaTemplate {
						d10 = d9
					} else if d9.Loc == LocImm {
						optimizedCallback11 := NewFunc(OptimizeProcToSerialFunction(d9.Imm))
						ctx.TrackImm(optimizedCallback11)
						d10 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback11, Rooted: true}
					} else {
						if d9.Loc == LocInputPair && int(d9.StackOff) < ctx.InputArgCount {
							d10 = ctx.RequestOptimizedCallback(int(d9.StackOff))
						} else {
							d10 = jitCopyScmerToPair(ctx, d9)
						}
					}
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.FreeDesc(&d9)
					var d12 JITValueDesc
					if d5.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d12 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d12)
					d13 = d12
					_ = d13
					ctx.StabilizeDescForControlFlow(&d13)
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
					ctx.EnsureDesc(&d13)
					var d14 JITValueDesc
					if d13.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < 32)}
					} else {
						r1 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d15.Condition, lbl9)
						ctx.EmitJmp(lbl10)
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
					r2 := ctx.AllocReg()
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d16)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d13)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d17 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d17)
					ctx.BindReg(r2, &d17)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d17}, 1)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					var d19 JITValueDesc
					if d5.SliceSizeKnown {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d19)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
						}
					}
					ps20 := PhiState{General: ps.General}
					ps20.OverlayValues = make([]JITValueDesc, 20)
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[9] = d9
					ps20.OverlayValues[10] = d10
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[16] = d16
					ps20.OverlayValues[17] = d17
					ps20.OverlayValues[18] = d18
					ps20.OverlayValues[19] = d19
					ps20.PhiValues = make([]JITValueDesc, 1)
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps20.PhiValues[0] = d21
					if ps20.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps20)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d22 := ps.PhiValues[0]
							ctx.EnsureDesc(&d22)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d22)
							} else {
								ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d23 JITValueDesc
					if d3.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d23)
					}
					if d23.Loc == LocReg && d3.Loc == LocReg && d23.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d23, &d19)
					var d24 JITValueDesc
					if d23.Loc == LocImm && d19.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d23.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, RegR11)
						}
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d24)
					} else if d23.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d24)
					} else {
						r5 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d24)
					}
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							if ps.General {
							}
							ps26 := PhiState{General: ps.General}
							ps26.OverlayValues = make([]JITValueDesc, 26)
							ps26.OverlayValues[3] = d3
							ps26.OverlayValues[4] = d4
							ps26.OverlayValues[5] = d5
							ps26.OverlayValues[6] = d6
							ps26.OverlayValues[7] = d7
							ps26.OverlayValues[9] = d9
							ps26.OverlayValues[10] = d10
							ps26.OverlayValues[12] = d12
							ps26.OverlayValues[13] = d13
							ps26.OverlayValues[14] = d14
							ps26.OverlayValues[15] = d15
							ps26.OverlayValues[16] = d16
							ps26.OverlayValues[17] = d17
							ps26.OverlayValues[18] = d18
							ps26.OverlayValues[19] = d19
							ps26.OverlayValues[21] = d21
							ps26.OverlayValues[22] = d22
							ps26.OverlayValues[23] = d23
							ps26.OverlayValues[24] = d24
							ps26.OverlayValues[25] = d25
							return bbs[2].RenderPS(ps26)
						}
						if ps.General {
						}
						ps27 := PhiState{General: ps.General}
						ps27.OverlayValues = make([]JITValueDesc, 26)
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[6] = d6
						ps27.OverlayValues[7] = d7
						ps27.OverlayValues[9] = d9
						ps27.OverlayValues[10] = d10
						ps27.OverlayValues[12] = d12
						ps27.OverlayValues[13] = d13
						ps27.OverlayValues[14] = d14
						ps27.OverlayValues[15] = d15
						ps27.OverlayValues[16] = d16
						ps27.OverlayValues[17] = d17
						ps27.OverlayValues[18] = d18
						ps27.OverlayValues[19] = d19
						ps27.OverlayValues[21] = d21
						ps27.OverlayValues[22] = d22
						ps27.OverlayValues[23] = d23
						ps27.OverlayValues[24] = d24
						ps27.OverlayValues[25] = d25
						return bbs[3].RenderPS(ps27)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d28 := ps.PhiValues[0]
							ctx.EnsureDesc(&d28)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d28)
							} else {
								ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d25.Condition, lbl3)
					snap29 := d3
					snap30 := d4
					snap31 := d5
					snap32 := d6
					snap33 := d7
					snap34 := d9
					snap35 := d10
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d17
					snap42 := d18
					snap43 := d19
					snap44 := d21
					snap45 := d22
					snap46 := d23
					snap47 := d24
					snap48 := d25
					snap49 := d28
					alloc50 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc50)
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d6 = snap32
					d7 = snap33
					d9 = snap34
					d10 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d18 = snap42
					d19 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d24 = snap47
					d25 = snap48
					d28 = snap49
					ctx.RestoreAllocState(alloc50)
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d6 = snap32
					d7 = snap33
					d9 = snap34
					d10 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d18 = snap42
					d19 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d24 = snap47
					d25 = snap48
					d28 = snap49
					ps51 := PhiState{General: true}
					ps51.OverlayValues = make([]JITValueDesc, 29)
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[6] = d6
					ps51.OverlayValues[7] = d7
					ps51.OverlayValues[9] = d9
					ps51.OverlayValues[10] = d10
					ps51.OverlayValues[12] = d12
					ps51.OverlayValues[13] = d13
					ps51.OverlayValues[14] = d14
					ps51.OverlayValues[15] = d15
					ps51.OverlayValues[16] = d16
					ps51.OverlayValues[17] = d17
					ps51.OverlayValues[18] = d18
					ps51.OverlayValues[19] = d19
					ps51.OverlayValues[21] = d21
					ps51.OverlayValues[22] = d22
					ps51.OverlayValues[23] = d23
					ps51.OverlayValues[24] = d24
					ps51.OverlayValues[25] = d25
					ps51.OverlayValues[28] = d28
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 29)
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[9] = d9
					ps52.OverlayValues[10] = d10
					ps52.OverlayValues[12] = d12
					ps52.OverlayValues[13] = d13
					ps52.OverlayValues[14] = d14
					ps52.OverlayValues[15] = d15
					ps52.OverlayValues[16] = d16
					ps52.OverlayValues[17] = d17
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[19] = d19
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[22] = d22
					ps52.OverlayValues[23] = d23
					ps52.OverlayValues[24] = d24
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[28] = d28
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d7
					snap58 := d9
					snap59 := d10
					snap60 := d12
					snap61 := d13
					snap62 := d14
					snap63 := d15
					snap64 := d16
					snap65 := d17
					snap66 := d18
					snap67 := d19
					snap68 := d21
					snap69 := d22
					snap70 := d23
					snap71 := d24
					snap72 := d25
					snap73 := d28
					alloc74 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps52)
					}
					ctx.RestoreAllocState(alloc74)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d9 = snap58
					d10 = snap59
					d12 = snap60
					d13 = snap61
					d14 = snap62
					d15 = snap63
					d16 = snap64
					d17 = snap65
					d18 = snap66
					d19 = snap67
					d21 = snap68
					d22 = snap69
					d23 = snap70
					d24 = snap71
					d25 = snap72
					d28 = snap73
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps51)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					d76 = ctx.EmitSliceElementAddress(&d5, &d23, 16)
					ctx.EnsureDesc(&d76)
					r6 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMem(r6, d76.Reg, 8)
					ctx.EmitMovRegMem(d76.Reg, d76.Reg, 0)
					d75 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d76.Reg, Reg2: r6}
					ctx.BindReg(d76.Reg, &d75)
					ctx.BindReg(r6, &d75)
					stackArray77 = ctx.AllocStack(int32(16))
					_ = stackArray77
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray77)+int32(0))
					d78 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d78
					callbackArgs80 := make([]JITValueDesc, 1)
					callbackArgs80[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray77) + 0}
					var d79 JITValueDesc
					callbackResultOff81 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff81))
					ctx.FreeDesc(&d78)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs82 := ctx.StabilizeCallbackArgs(callbackArgs80)
						ctx.ReclaimUntrackedRegs()
						outerRegs83 := ctx.PreserveOuterRegs()
						d79 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs82, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						ctx.RestoreOuterRegs(outerRegs83)
						ctx.ReclaimUntrackedRegs()
					} else {
						d84, knownBuiltin85 := jitEmitKnownDeclaration(ctx, d7, callbackArgs80, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						if knownBuiltin85 {
							d79 = d84
						} else {
							ctx.Coverage.DynamicCalls++
							d86 := jitCopyScmerToPair(ctx, d7)
							d79 = jitEmitDynamicCallableAt(ctx, d86, callbackArgs80, int32(stackArray77), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						}
					}
					d87 = args[3]
					d87.ID = 0
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d79)
					d79 = JITPrepareScmerGoArg(ctx, d79)
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					d75 = JITPrepareScmerGoArg(ctx, d75)
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d87)
					d87 = JITPrepareScmerGoArg(ctx, d87)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d79)
					ctx.SyncDesc(&d75)
					ctx.SyncDesc(&d87)
					ctx.SyncDesc(&d10)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).ReduceValue), []JITValueDesc{d18, d79, d75, d87, d10})
					ctx.FreeDesc(&d79)
					ctx.FreeDesc(&d75)
					ctx.FreeDesc(&d87)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d88 = d23
						if d88.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d88)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d88)
						} else {
							ctx.EmitStoreToStack(d88, int32(bbs[1].PhiBase)+int32(0))
						}
						if d23.Loc == LocReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps89 := PhiState{General: ps.General}
					ps89.OverlayValues = make([]JITValueDesc, 89)
					ps89.OverlayValues[3] = d3
					ps89.OverlayValues[4] = d4
					ps89.OverlayValues[5] = d5
					ps89.OverlayValues[6] = d6
					ps89.OverlayValues[7] = d7
					ps89.OverlayValues[9] = d9
					ps89.OverlayValues[10] = d10
					ps89.OverlayValues[12] = d12
					ps89.OverlayValues[13] = d13
					ps89.OverlayValues[14] = d14
					ps89.OverlayValues[15] = d15
					ps89.OverlayValues[16] = d16
					ps89.OverlayValues[17] = d17
					ps89.OverlayValues[18] = d18
					ps89.OverlayValues[19] = d19
					ps89.OverlayValues[21] = d21
					ps89.OverlayValues[22] = d22
					ps89.OverlayValues[23] = d23
					ps89.OverlayValues[24] = d24
					ps89.OverlayValues[25] = d25
					ps89.OverlayValues[28] = d28
					ps89.OverlayValues[75] = d75
					ps89.OverlayValues[76] = d76
					ps89.OverlayValues[78] = d78
					ps89.OverlayValues[79] = d79
					ps89.OverlayValues[84] = d84
					ps89.OverlayValues[86] = d86
					ps89.OverlayValues[87] = d87
					ps89.OverlayValues[88] = d88
					ps89.PhiValues = make([]JITValueDesc, 1)
					d90 = d23
					ps89.PhiValues[0] = d90
					if ps89.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps89)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
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
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					ctx.ReclaimUntrackedRegs()
					var d91 JITValueDesc
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d91 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r7}
						ctx.BindReg(d18.Reg, &d91)
						ctx.BindReg(r7, &d91)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d91)
						ctx.BindReg(r7, &d91)
						d18.Loc = LocNone
					}
					ctx.SyncDesc(&d91)
					if d91.Loc == LocRegPair || d91.Loc == LocStackPair || d91.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d91, &result)
						result.Type = d91.Type
					} else {
						switch d91.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d91)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d91)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d91)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d91, &result)
							result.Type = d91.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps92 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps92)
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
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
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d28 JITValueDesc
				_ = d28
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var stackArray77 int32
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var callbackResultOff81 int32
				var d84 JITValueDesc
				_ = d84
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var stackArray88 int32
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var callbackResultOff92 int32
				var d95 JITValueDesc
				_ = d95
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 12}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d3
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d4}, 3)
					}
					ctx.BindReg(d5.Reg, &d5)
					ctx.BindReg(d5.Reg2, &d5)
					ctx.BindReg(d5.Reg3, &d5)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d6 = args[1]
					d6.ID = 0
					var d7 JITValueDesc
					if d6.Loc == LocLambdaTemplate {
						d7 = d6
					} else if d6.Loc == LocImm {
						optimizedCallback8 := NewFunc(OptimizeProcToSerialFunction(d6.Imm))
						ctx.TrackImm(optimizedCallback8)
						d7 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback8, Rooted: true}
					} else {
						if d6.Loc == LocInputPair && int(d6.StackOff) < ctx.InputArgCount {
							d7 = ctx.RequestOptimizedCallback(int(d6.StackOff))
						} else {
							d7 = jitCopyScmerToPair(ctx, d6)
						}
					}
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.FreeDesc(&d6)
					d9 = args[2]
					d9.ID = 0
					var d10 JITValueDesc
					if d9.Loc == LocLambdaTemplate {
						d10 = d9
					} else if d9.Loc == LocImm {
						optimizedCallback11 := NewFunc(OptimizeProcToSerialFunction(d9.Imm))
						ctx.TrackImm(optimizedCallback11)
						d10 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback11, Rooted: true}
					} else {
						if d9.Loc == LocInputPair && int(d9.StackOff) < ctx.InputArgCount {
							d10 = ctx.RequestOptimizedCallback(int(d9.StackOff))
						} else {
							d10 = jitCopyScmerToPair(ctx, d9)
						}
					}
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.FreeDesc(&d9)
					var d12 JITValueDesc
					if d5.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d12 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d12)
					d13 = d12
					_ = d13
					ctx.StabilizeDescForControlFlow(&d13)
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
					ctx.EnsureDesc(&d13)
					var d14 JITValueDesc
					if d13.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < 32)}
					} else {
						r1 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d15.Condition, lbl9)
						ctx.EmitJmp(lbl10)
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
					r2 := ctx.AllocReg()
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d16)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d13)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d17 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d17)
					ctx.BindReg(r2, &d17)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d17}, 1)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					var d19 JITValueDesc
					if d5.SliceSizeKnown {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d19)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
						}
					}
					ps20 := PhiState{General: ps.General}
					ps20.OverlayValues = make([]JITValueDesc, 20)
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[9] = d9
					ps20.OverlayValues[10] = d10
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[16] = d16
					ps20.OverlayValues[17] = d17
					ps20.OverlayValues[18] = d18
					ps20.OverlayValues[19] = d19
					ps20.PhiValues = make([]JITValueDesc, 1)
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps20.PhiValues[0] = d21
					if ps20.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps20)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d22 := ps.PhiValues[0]
							ctx.EnsureDesc(&d22)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d22)
							} else {
								ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d23 JITValueDesc
					if d3.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d23)
					}
					if d23.Loc == LocReg && d3.Loc == LocReg && d23.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d23, &d19)
					var d24 JITValueDesc
					if d23.Loc == LocImm && d19.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d23.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, RegR11)
						}
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d24)
					} else if d23.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d24)
					} else {
						r5 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d24)
					}
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							if ps.General {
							}
							ps26 := PhiState{General: ps.General}
							ps26.OverlayValues = make([]JITValueDesc, 26)
							ps26.OverlayValues[3] = d3
							ps26.OverlayValues[4] = d4
							ps26.OverlayValues[5] = d5
							ps26.OverlayValues[6] = d6
							ps26.OverlayValues[7] = d7
							ps26.OverlayValues[9] = d9
							ps26.OverlayValues[10] = d10
							ps26.OverlayValues[12] = d12
							ps26.OverlayValues[13] = d13
							ps26.OverlayValues[14] = d14
							ps26.OverlayValues[15] = d15
							ps26.OverlayValues[16] = d16
							ps26.OverlayValues[17] = d17
							ps26.OverlayValues[18] = d18
							ps26.OverlayValues[19] = d19
							ps26.OverlayValues[21] = d21
							ps26.OverlayValues[22] = d22
							ps26.OverlayValues[23] = d23
							ps26.OverlayValues[24] = d24
							ps26.OverlayValues[25] = d25
							return bbs[2].RenderPS(ps26)
						}
						if ps.General {
						}
						ps27 := PhiState{General: ps.General}
						ps27.OverlayValues = make([]JITValueDesc, 26)
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[6] = d6
						ps27.OverlayValues[7] = d7
						ps27.OverlayValues[9] = d9
						ps27.OverlayValues[10] = d10
						ps27.OverlayValues[12] = d12
						ps27.OverlayValues[13] = d13
						ps27.OverlayValues[14] = d14
						ps27.OverlayValues[15] = d15
						ps27.OverlayValues[16] = d16
						ps27.OverlayValues[17] = d17
						ps27.OverlayValues[18] = d18
						ps27.OverlayValues[19] = d19
						ps27.OverlayValues[21] = d21
						ps27.OverlayValues[22] = d22
						ps27.OverlayValues[23] = d23
						ps27.OverlayValues[24] = d24
						ps27.OverlayValues[25] = d25
						return bbs[3].RenderPS(ps27)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d28 := ps.PhiValues[0]
							ctx.EnsureDesc(&d28)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d28)
							} else {
								ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d25.Condition, lbl3)
					snap29 := d3
					snap30 := d4
					snap31 := d5
					snap32 := d6
					snap33 := d7
					snap34 := d9
					snap35 := d10
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d17
					snap42 := d18
					snap43 := d19
					snap44 := d21
					snap45 := d22
					snap46 := d23
					snap47 := d24
					snap48 := d25
					snap49 := d28
					alloc50 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc50)
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d6 = snap32
					d7 = snap33
					d9 = snap34
					d10 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d18 = snap42
					d19 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d24 = snap47
					d25 = snap48
					d28 = snap49
					ctx.RestoreAllocState(alloc50)
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d6 = snap32
					d7 = snap33
					d9 = snap34
					d10 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d18 = snap42
					d19 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d24 = snap47
					d25 = snap48
					d28 = snap49
					ps51 := PhiState{General: true}
					ps51.OverlayValues = make([]JITValueDesc, 29)
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[6] = d6
					ps51.OverlayValues[7] = d7
					ps51.OverlayValues[9] = d9
					ps51.OverlayValues[10] = d10
					ps51.OverlayValues[12] = d12
					ps51.OverlayValues[13] = d13
					ps51.OverlayValues[14] = d14
					ps51.OverlayValues[15] = d15
					ps51.OverlayValues[16] = d16
					ps51.OverlayValues[17] = d17
					ps51.OverlayValues[18] = d18
					ps51.OverlayValues[19] = d19
					ps51.OverlayValues[21] = d21
					ps51.OverlayValues[22] = d22
					ps51.OverlayValues[23] = d23
					ps51.OverlayValues[24] = d24
					ps51.OverlayValues[25] = d25
					ps51.OverlayValues[28] = d28
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 29)
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[9] = d9
					ps52.OverlayValues[10] = d10
					ps52.OverlayValues[12] = d12
					ps52.OverlayValues[13] = d13
					ps52.OverlayValues[14] = d14
					ps52.OverlayValues[15] = d15
					ps52.OverlayValues[16] = d16
					ps52.OverlayValues[17] = d17
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[19] = d19
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[22] = d22
					ps52.OverlayValues[23] = d23
					ps52.OverlayValues[24] = d24
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[28] = d28
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d7
					snap58 := d9
					snap59 := d10
					snap60 := d12
					snap61 := d13
					snap62 := d14
					snap63 := d15
					snap64 := d16
					snap65 := d17
					snap66 := d18
					snap67 := d19
					snap68 := d21
					snap69 := d22
					snap70 := d23
					snap71 := d24
					snap72 := d25
					snap73 := d28
					alloc74 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps52)
					}
					ctx.RestoreAllocState(alloc74)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d9 = snap58
					d10 = snap59
					d12 = snap60
					d13 = snap61
					d14 = snap62
					d15 = snap63
					d16 = snap64
					d17 = snap65
					d18 = snap66
					d19 = snap67
					d21 = snap68
					d22 = snap69
					d23 = snap70
					d24 = snap71
					d25 = snap72
					d28 = snap73
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps51)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					d76 = ctx.EmitSliceElementAddress(&d5, &d23, 16)
					ctx.EnsureDesc(&d76)
					r6 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMem(r6, d76.Reg, 8)
					ctx.EmitMovRegMem(d76.Reg, d76.Reg, 0)
					d75 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d76.Reg, Reg2: r6}
					ctx.BindReg(d76.Reg, &d75)
					ctx.BindReg(r6, &d75)
					stackArray77 = ctx.AllocStack(int32(16))
					_ = stackArray77
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray77)+int32(0))
					d78 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d78
					callbackArgs80 := make([]JITValueDesc, 1)
					callbackArgs80[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray77) + 0}
					var d79 JITValueDesc
					callbackResultOff81 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff81))
					ctx.FreeDesc(&d78)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs82 := ctx.StabilizeCallbackArgs(callbackArgs80)
						ctx.ReclaimUntrackedRegs()
						outerRegs83 := ctx.PreserveOuterRegs()
						d79 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs82, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						ctx.RestoreOuterRegs(outerRegs83)
						ctx.ReclaimUntrackedRegs()
					} else {
						d84, knownBuiltin85 := jitEmitKnownDeclaration(ctx, d7, callbackArgs80, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						if knownBuiltin85 {
							d79 = d84
						} else {
							ctx.Coverage.DynamicCalls++
							d86 := jitCopyScmerToPair(ctx, d7)
							d79 = jitEmitDynamicCallableAt(ctx, d86, callbackArgs80, int32(stackArray77), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						}
					}
					d87 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray88 = ctx.AllocStack(int32(32))
					_ = stackArray88
					ctx.SyncDesc(&d87)
					ctx.EmitStoreScmerToStack(d87, int32(stackArray88)+int32(0))
					ctx.FreeDesc(&d87)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray88)+int32(16))
					ctx.FreeDesc(&d75)
					d89 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d89
					callbackArgs91 := make([]JITValueDesc, 2)
					callbackArgs91[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray88) + 0}
					callbackArgs91[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray88) + 16}
					var d90 JITValueDesc
					callbackResultOff92 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff92))
					ctx.FreeDesc(&d89)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d10.Loc == LocLambdaTemplate && d10.Lambda != nil {
						stableCallbackArgs93 := ctx.StabilizeCallbackArgs(callbackArgs91)
						ctx.ReclaimUntrackedRegs()
						outerRegs94 := ctx.PreserveOuterRegs()
						d90 = JITEmitProcInlineWithOuter(ctx, &d10.Lambda.Proc, d10.Lambda.Outer, stableCallbackArgs93, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff92), ID: 0})
						ctx.RestoreOuterRegs(outerRegs94)
						ctx.ReclaimUntrackedRegs()
					} else {
						d95, knownBuiltin96 := jitEmitKnownDeclaration(ctx, d10, callbackArgs91, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff92), ID: 0})
						if knownBuiltin96 {
							d90 = d95
						} else {
							ctx.Coverage.DynamicCalls++
							d97 := jitCopyScmerToPair(ctx, d10)
							d90 = jitEmitDynamicCallableAt(ctx, d97, callbackArgs91, int32(stackArray88), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff92), ID: 0})
						}
					}
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d79)
					d79 = JITPrepareScmerGoArg(ctx, d79)
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d90)
					d90 = JITPrepareScmerGoArg(ctx, d90)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d79)
					ctx.SyncDesc(&d90)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d79, d90})
					ctx.FreeDesc(&d79)
					ctx.FreeDesc(&d90)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d98 = d23
						if d98.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d98)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d98)
						} else {
							ctx.EmitStoreToStack(d98, int32(bbs[1].PhiBase)+int32(0))
						}
						if d23.Loc == LocReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps99 := PhiState{General: ps.General}
					ps99.OverlayValues = make([]JITValueDesc, 99)
					ps99.OverlayValues[3] = d3
					ps99.OverlayValues[4] = d4
					ps99.OverlayValues[5] = d5
					ps99.OverlayValues[6] = d6
					ps99.OverlayValues[7] = d7
					ps99.OverlayValues[9] = d9
					ps99.OverlayValues[10] = d10
					ps99.OverlayValues[12] = d12
					ps99.OverlayValues[13] = d13
					ps99.OverlayValues[14] = d14
					ps99.OverlayValues[15] = d15
					ps99.OverlayValues[16] = d16
					ps99.OverlayValues[17] = d17
					ps99.OverlayValues[18] = d18
					ps99.OverlayValues[19] = d19
					ps99.OverlayValues[21] = d21
					ps99.OverlayValues[22] = d22
					ps99.OverlayValues[23] = d23
					ps99.OverlayValues[24] = d24
					ps99.OverlayValues[25] = d25
					ps99.OverlayValues[28] = d28
					ps99.OverlayValues[75] = d75
					ps99.OverlayValues[76] = d76
					ps99.OverlayValues[78] = d78
					ps99.OverlayValues[79] = d79
					ps99.OverlayValues[84] = d84
					ps99.OverlayValues[86] = d86
					ps99.OverlayValues[87] = d87
					ps99.OverlayValues[89] = d89
					ps99.OverlayValues[90] = d90
					ps99.OverlayValues[95] = d95
					ps99.OverlayValues[97] = d97
					ps99.OverlayValues[98] = d98
					ps99.PhiValues = make([]JITValueDesc, 1)
					d100 = d23
					ps99.PhiValues[0] = d100
					if ps99.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps99)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
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
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
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
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					ctx.ReclaimUntrackedRegs()
					var d101 JITValueDesc
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d101 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r7}
						ctx.BindReg(d18.Reg, &d101)
						ctx.BindReg(r7, &d101)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d101)
						ctx.BindReg(r7, &d101)
						d18.Loc = LocNone
					}
					ctx.SyncDesc(&d101)
					if d101.Loc == LocRegPair || d101.Loc == LocStackPair || d101.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d101, &result)
						result.Type = d101.Type
					} else {
						switch d101.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d101)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d101)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d101)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d101, &result)
							result.Type = d101.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps102 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps102)
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
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
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d28 JITValueDesc
				_ = d28
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var stackArray78 int32
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var callbackResultOff82 int32
				var d85 JITValueDesc
				_ = d85
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var stackArray89 int32
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var callbackResultOff93 int32
				var d96 JITValueDesc
				_ = d96
				var d98 JITValueDesc
				_ = d98
				var d99 JITValueDesc
				_ = d99
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 12}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d3
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d4}, 3)
					}
					ctx.BindReg(d5.Reg, &d5)
					ctx.BindReg(d5.Reg2, &d5)
					ctx.BindReg(d5.Reg3, &d5)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d6 = args[1]
					d6.ID = 0
					var d7 JITValueDesc
					if d6.Loc == LocLambdaTemplate {
						d7 = d6
					} else if d6.Loc == LocImm {
						optimizedCallback8 := NewFunc(OptimizeProcToSerialFunction(d6.Imm))
						ctx.TrackImm(optimizedCallback8)
						d7 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback8, Rooted: true}
					} else {
						if d6.Loc == LocInputPair && int(d6.StackOff) < ctx.InputArgCount {
							d7 = ctx.RequestOptimizedCallback(int(d6.StackOff))
						} else {
							d7 = jitCopyScmerToPair(ctx, d6)
						}
					}
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.FreeDesc(&d6)
					d9 = args[2]
					d9.ID = 0
					var d10 JITValueDesc
					if d9.Loc == LocLambdaTemplate {
						d10 = d9
					} else if d9.Loc == LocImm {
						optimizedCallback11 := NewFunc(OptimizeProcToSerialFunction(d9.Imm))
						ctx.TrackImm(optimizedCallback11)
						d10 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback11, Rooted: true}
					} else {
						if d9.Loc == LocInputPair && int(d9.StackOff) < ctx.InputArgCount {
							d10 = ctx.RequestOptimizedCallback(int(d9.StackOff))
						} else {
							d10 = jitCopyScmerToPair(ctx, d9)
						}
					}
					ctx.StabilizeDescForControlFlow(&d10)
					ctx.FreeDesc(&d9)
					var d12 JITValueDesc
					if d5.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d12 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d12)
					d13 = d12
					_ = d13
					ctx.StabilizeDescForControlFlow(&d13)
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
					ctx.EnsureDesc(&d13)
					var d14 JITValueDesc
					if d13.Loc == LocImm {
						d14 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < 32)}
					} else {
						r1 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d15.Loc == LocImm {
						if d15.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d15.Condition, lbl9)
						ctx.EmitJmp(lbl10)
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
					r2 := ctx.AllocReg()
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d16)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d13)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d17 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d17)
					ctx.BindReg(r2, &d17)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d17}, 1)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					var d19 JITValueDesc
					if d5.SliceSizeKnown {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d19)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
						}
					}
					ps20 := PhiState{General: ps.General}
					ps20.OverlayValues = make([]JITValueDesc, 20)
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[9] = d9
					ps20.OverlayValues[10] = d10
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[16] = d16
					ps20.OverlayValues[17] = d17
					ps20.OverlayValues[18] = d18
					ps20.OverlayValues[19] = d19
					ps20.PhiValues = make([]JITValueDesc, 1)
					d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps20.PhiValues[0] = d21
					if ps20.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps20)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d22 := ps.PhiValues[0]
							ctx.EnsureDesc(&d22)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d22)
							} else {
								ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
						d21 = ps.OverlayValues[21]
					}
					if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
						d22 = ps.OverlayValues[22]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d23 JITValueDesc
					if d3.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d23)
					}
					if d23.Loc == LocReg && d3.Loc == LocReg && d23.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d23, &d19)
					var d24 JITValueDesc
					if d23.Loc == LocImm && d19.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d23.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, RegR11)
						}
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d24)
					} else if d23.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d24)
					} else {
						r5 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d24)
					}
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							if ps.General {
							}
							ps26 := PhiState{General: ps.General}
							ps26.OverlayValues = make([]JITValueDesc, 26)
							ps26.OverlayValues[3] = d3
							ps26.OverlayValues[4] = d4
							ps26.OverlayValues[5] = d5
							ps26.OverlayValues[6] = d6
							ps26.OverlayValues[7] = d7
							ps26.OverlayValues[9] = d9
							ps26.OverlayValues[10] = d10
							ps26.OverlayValues[12] = d12
							ps26.OverlayValues[13] = d13
							ps26.OverlayValues[14] = d14
							ps26.OverlayValues[15] = d15
							ps26.OverlayValues[16] = d16
							ps26.OverlayValues[17] = d17
							ps26.OverlayValues[18] = d18
							ps26.OverlayValues[19] = d19
							ps26.OverlayValues[21] = d21
							ps26.OverlayValues[22] = d22
							ps26.OverlayValues[23] = d23
							ps26.OverlayValues[24] = d24
							ps26.OverlayValues[25] = d25
							return bbs[2].RenderPS(ps26)
						}
						if ps.General {
						}
						ps27 := PhiState{General: ps.General}
						ps27.OverlayValues = make([]JITValueDesc, 26)
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[6] = d6
						ps27.OverlayValues[7] = d7
						ps27.OverlayValues[9] = d9
						ps27.OverlayValues[10] = d10
						ps27.OverlayValues[12] = d12
						ps27.OverlayValues[13] = d13
						ps27.OverlayValues[14] = d14
						ps27.OverlayValues[15] = d15
						ps27.OverlayValues[16] = d16
						ps27.OverlayValues[17] = d17
						ps27.OverlayValues[18] = d18
						ps27.OverlayValues[19] = d19
						ps27.OverlayValues[21] = d21
						ps27.OverlayValues[22] = d22
						ps27.OverlayValues[23] = d23
						ps27.OverlayValues[24] = d24
						ps27.OverlayValues[25] = d25
						return bbs[3].RenderPS(ps27)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d28 := ps.PhiValues[0]
							ctx.EnsureDesc(&d28)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d28)
							} else {
								ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d25.Condition, lbl3)
					snap29 := d3
					snap30 := d4
					snap31 := d5
					snap32 := d6
					snap33 := d7
					snap34 := d9
					snap35 := d10
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d17
					snap42 := d18
					snap43 := d19
					snap44 := d21
					snap45 := d22
					snap46 := d23
					snap47 := d24
					snap48 := d25
					snap49 := d28
					alloc50 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc50)
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d6 = snap32
					d7 = snap33
					d9 = snap34
					d10 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d18 = snap42
					d19 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d24 = snap47
					d25 = snap48
					d28 = snap49
					ctx.RestoreAllocState(alloc50)
					d3 = snap29
					d4 = snap30
					d5 = snap31
					d6 = snap32
					d7 = snap33
					d9 = snap34
					d10 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d17 = snap41
					d18 = snap42
					d19 = snap43
					d21 = snap44
					d22 = snap45
					d23 = snap46
					d24 = snap47
					d25 = snap48
					d28 = snap49
					ps51 := PhiState{General: true}
					ps51.OverlayValues = make([]JITValueDesc, 29)
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[6] = d6
					ps51.OverlayValues[7] = d7
					ps51.OverlayValues[9] = d9
					ps51.OverlayValues[10] = d10
					ps51.OverlayValues[12] = d12
					ps51.OverlayValues[13] = d13
					ps51.OverlayValues[14] = d14
					ps51.OverlayValues[15] = d15
					ps51.OverlayValues[16] = d16
					ps51.OverlayValues[17] = d17
					ps51.OverlayValues[18] = d18
					ps51.OverlayValues[19] = d19
					ps51.OverlayValues[21] = d21
					ps51.OverlayValues[22] = d22
					ps51.OverlayValues[23] = d23
					ps51.OverlayValues[24] = d24
					ps51.OverlayValues[25] = d25
					ps51.OverlayValues[28] = d28
					ps52 := PhiState{General: true}
					ps52.OverlayValues = make([]JITValueDesc, 29)
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[6] = d6
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[9] = d9
					ps52.OverlayValues[10] = d10
					ps52.OverlayValues[12] = d12
					ps52.OverlayValues[13] = d13
					ps52.OverlayValues[14] = d14
					ps52.OverlayValues[15] = d15
					ps52.OverlayValues[16] = d16
					ps52.OverlayValues[17] = d17
					ps52.OverlayValues[18] = d18
					ps52.OverlayValues[19] = d19
					ps52.OverlayValues[21] = d21
					ps52.OverlayValues[22] = d22
					ps52.OverlayValues[23] = d23
					ps52.OverlayValues[24] = d24
					ps52.OverlayValues[25] = d25
					ps52.OverlayValues[28] = d28
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d7
					snap58 := d9
					snap59 := d10
					snap60 := d12
					snap61 := d13
					snap62 := d14
					snap63 := d15
					snap64 := d16
					snap65 := d17
					snap66 := d18
					snap67 := d19
					snap68 := d21
					snap69 := d22
					snap70 := d23
					snap71 := d24
					snap72 := d25
					snap73 := d28
					alloc74 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps52)
					}
					ctx.RestoreAllocState(alloc74)
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d9 = snap58
					d10 = snap59
					d12 = snap60
					d13 = snap61
					d14 = snap62
					d15 = snap63
					d16 = snap64
					d17 = snap65
					d18 = snap66
					d19 = snap67
					d21 = snap68
					d22 = snap69
					d23 = snap70
					d24 = snap71
					d25 = snap72
					d28 = snap73
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps51)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d23)
					d76 = ctx.EmitSliceElementAddress(&d5, &d23, 16)
					ctx.EnsureDesc(&d76)
					r6 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMem(r6, d76.Reg, 8)
					ctx.EmitMovRegMem(d76.Reg, d76.Reg, 0)
					d75 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d76.Reg, Reg2: r6}
					ctx.BindReg(d76.Reg, &d75)
					ctx.BindReg(r6, &d75)
					d77 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray78 = ctx.AllocStack(int32(32))
					_ = stackArray78
					ctx.SyncDesc(&d77)
					ctx.EmitStoreScmerToStack(d77, int32(stackArray78)+int32(0))
					ctx.FreeDesc(&d77)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray78)+int32(16))
					d79 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d79
					callbackArgs81 := make([]JITValueDesc, 2)
					callbackArgs81[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray78) + 0}
					callbackArgs81[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray78) + 16}
					var d80 JITValueDesc
					callbackResultOff82 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff82))
					ctx.FreeDesc(&d79)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs83 := ctx.StabilizeCallbackArgs(callbackArgs81)
						ctx.ReclaimUntrackedRegs()
						outerRegs84 := ctx.PreserveOuterRegs()
						d80 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs83, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff82), ID: 0})
						ctx.RestoreOuterRegs(outerRegs84)
						ctx.ReclaimUntrackedRegs()
					} else {
						d85, knownBuiltin86 := jitEmitKnownDeclaration(ctx, d7, callbackArgs81, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff82), ID: 0})
						if knownBuiltin86 {
							d80 = d85
						} else {
							ctx.Coverage.DynamicCalls++
							d87 := jitCopyScmerToPair(ctx, d7)
							d80 = jitEmitDynamicCallableAt(ctx, d87, callbackArgs81, int32(stackArray78), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff82), ID: 0})
						}
					}
					d88 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray89 = ctx.AllocStack(int32(32))
					_ = stackArray89
					ctx.SyncDesc(&d88)
					ctx.EmitStoreScmerToStack(d88, int32(stackArray89)+int32(0))
					ctx.FreeDesc(&d88)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray89)+int32(16))
					ctx.FreeDesc(&d75)
					d90 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d90
					callbackArgs92 := make([]JITValueDesc, 2)
					callbackArgs92[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray89) + 0}
					callbackArgs92[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray89) + 16}
					var d91 JITValueDesc
					callbackResultOff93 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff93))
					ctx.FreeDesc(&d90)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d10.Loc == LocLambdaTemplate && d10.Lambda != nil {
						stableCallbackArgs94 := ctx.StabilizeCallbackArgs(callbackArgs92)
						ctx.ReclaimUntrackedRegs()
						outerRegs95 := ctx.PreserveOuterRegs()
						d91 = JITEmitProcInlineWithOuter(ctx, &d10.Lambda.Proc, d10.Lambda.Outer, stableCallbackArgs94, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff93), ID: 0})
						ctx.RestoreOuterRegs(outerRegs95)
						ctx.ReclaimUntrackedRegs()
					} else {
						d96, knownBuiltin97 := jitEmitKnownDeclaration(ctx, d10, callbackArgs92, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff93), ID: 0})
						if knownBuiltin97 {
							d91 = d96
						} else {
							ctx.Coverage.DynamicCalls++
							d98 := jitCopyScmerToPair(ctx, d10)
							d91 = jitEmitDynamicCallableAt(ctx, d98, callbackArgs92, int32(stackArray89), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff93), ID: 0})
						}
					}
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d80)
					d80 = JITPrepareScmerGoArg(ctx, d80)
					ctx.EnsureDesc(&d91)
					ctx.EnsureDesc(&d91)
					d91 = JITPrepareScmerGoArg(ctx, d91)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d80)
					ctx.SyncDesc(&d91)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d80, d91})
					ctx.FreeDesc(&d80)
					ctx.FreeDesc(&d91)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d99 = d23
						if d99.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d99)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d99)
						} else {
							ctx.EmitStoreToStack(d99, int32(bbs[1].PhiBase)+int32(0))
						}
						if d23.Loc == LocReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps100 := PhiState{General: ps.General}
					ps100.OverlayValues = make([]JITValueDesc, 100)
					ps100.OverlayValues[3] = d3
					ps100.OverlayValues[4] = d4
					ps100.OverlayValues[5] = d5
					ps100.OverlayValues[6] = d6
					ps100.OverlayValues[7] = d7
					ps100.OverlayValues[9] = d9
					ps100.OverlayValues[10] = d10
					ps100.OverlayValues[12] = d12
					ps100.OverlayValues[13] = d13
					ps100.OverlayValues[14] = d14
					ps100.OverlayValues[15] = d15
					ps100.OverlayValues[16] = d16
					ps100.OverlayValues[17] = d17
					ps100.OverlayValues[18] = d18
					ps100.OverlayValues[19] = d19
					ps100.OverlayValues[21] = d21
					ps100.OverlayValues[22] = d22
					ps100.OverlayValues[23] = d23
					ps100.OverlayValues[24] = d24
					ps100.OverlayValues[25] = d25
					ps100.OverlayValues[28] = d28
					ps100.OverlayValues[75] = d75
					ps100.OverlayValues[76] = d76
					ps100.OverlayValues[77] = d77
					ps100.OverlayValues[79] = d79
					ps100.OverlayValues[80] = d80
					ps100.OverlayValues[85] = d85
					ps100.OverlayValues[87] = d87
					ps100.OverlayValues[88] = d88
					ps100.OverlayValues[90] = d90
					ps100.OverlayValues[91] = d91
					ps100.OverlayValues[96] = d96
					ps100.OverlayValues[98] = d98
					ps100.OverlayValues[99] = d99
					ps100.PhiValues = make([]JITValueDesc, 1)
					d101 = d23
					ps100.PhiValues[0] = d101
					if ps100.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps100)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
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
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
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
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					ctx.ReclaimUntrackedRegs()
					var d102 JITValueDesc
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d102 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r7}
						ctx.BindReg(d18.Reg, &d102)
						ctx.BindReg(r7, &d102)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d102)
						ctx.BindReg(r7, &d102)
						d18.Loc = LocNone
					}
					ctx.SyncDesc(&d102)
					if d102.Loc == LocRegPair || d102.Loc == LocStackPair || d102.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d102, &result)
						result.Type = d102.Type
					} else {
						switch d102.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d102)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d102)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d102)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d102, &result)
							result.Type = d102.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps103 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps103)
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
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
				var d16 JITValueDesc
				_ = d16
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
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var stackArray70 int32
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var callbackResultOff74 int32
				var d77 JITValueDesc
				_ = d77
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 12}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d3
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d4}, 3)
					}
					ctx.BindReg(d5.Reg, &d5)
					ctx.BindReg(d5.Reg2, &d5)
					ctx.BindReg(d5.Reg3, &d5)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d6 = args[1]
					d6.ID = 0
					var d7 JITValueDesc
					if d6.Loc == LocLambdaTemplate {
						d7 = d6
					} else if d6.Loc == LocImm {
						optimizedCallback8 := NewFunc(OptimizeProcToSerialFunction(d6.Imm))
						ctx.TrackImm(optimizedCallback8)
						d7 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback8, Rooted: true}
					} else {
						if d6.Loc == LocInputPair && int(d6.StackOff) < ctx.InputArgCount {
							d7 = ctx.RequestOptimizedCallback(int(d6.StackOff))
						} else {
							d7 = jitCopyScmerToPair(ctx, d6)
						}
					}
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.FreeDesc(&d6)
					var d9 JITValueDesc
					if d5.SliceSizeKnown {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d9)
					d10 = d9
					_ = d10
					ctx.StabilizeDescForControlFlow(&d10)
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
					ctx.EnsureDesc(&d10)
					var d11 JITValueDesc
					if d10.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() < 32)}
					} else {
						r1 := ctx.AllocRegExcept(d10.Reg)
						ctx.EmitCmpRegImm32(d10.Reg, 32)
						d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d11)
					}
					ctx.ReclaimUntrackedRegs()
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d12.Condition, lbl9)
						ctx.EmitJmp(lbl10)
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
					r2 := ctx.AllocReg()
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d13)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d10)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d14 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d14)
					ctx.BindReg(r2, &d14)
					ctx.FreeDesc(&d9)
					ctx.EnsureDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d14}, 1)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.FreeDesc(&d14)
					var d16 JITValueDesc
					if d5.SliceSizeKnown {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d16)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
						}
					}
					ps17 := PhiState{General: ps.General}
					ps17.OverlayValues = make([]JITValueDesc, 17)
					ps17.OverlayValues[3] = d3
					ps17.OverlayValues[4] = d4
					ps17.OverlayValues[5] = d5
					ps17.OverlayValues[6] = d6
					ps17.OverlayValues[7] = d7
					ps17.OverlayValues[9] = d9
					ps17.OverlayValues[10] = d10
					ps17.OverlayValues[11] = d11
					ps17.OverlayValues[12] = d12
					ps17.OverlayValues[13] = d13
					ps17.OverlayValues[14] = d14
					ps17.OverlayValues[15] = d15
					ps17.OverlayValues[16] = d16
					ps17.PhiValues = make([]JITValueDesc, 1)
					d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps17.PhiValues[0] = d18
					if ps17.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps17)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d19 := ps.PhiValues[0]
							ctx.EnsureDesc(&d19)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d19)
							} else {
								ctx.EmitStoreToStack(d19, int32(bbs[1].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d20 JITValueDesc
					if d3.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d20)
					}
					if d20.Loc == LocReg && d3.Loc == LocReg && d20.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDescsTogether(&d20, &d16)
					var d21 JITValueDesc
					if d20.Loc == LocImm && d16.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.Int() < d16.Imm.Int())}
					} else if d16.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d20.Reg)
						if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d20.Reg, int32(d16.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
							ctx.EmitCmpInt64(d20.Reg, RegR11)
						}
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d21)
					} else if d20.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d16.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d21)
					} else {
						r5 := ctx.AllocRegExcept(d20.Reg)
						ctx.EmitCmpInt64(d20.Reg, d16.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d21)
					}
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
							ps23.OverlayValues[3] = d3
							ps23.OverlayValues[4] = d4
							ps23.OverlayValues[5] = d5
							ps23.OverlayValues[6] = d6
							ps23.OverlayValues[7] = d7
							ps23.OverlayValues[9] = d9
							ps23.OverlayValues[10] = d10
							ps23.OverlayValues[11] = d11
							ps23.OverlayValues[12] = d12
							ps23.OverlayValues[13] = d13
							ps23.OverlayValues[14] = d14
							ps23.OverlayValues[15] = d15
							ps23.OverlayValues[16] = d16
							ps23.OverlayValues[18] = d18
							ps23.OverlayValues[19] = d19
							ps23.OverlayValues[20] = d20
							ps23.OverlayValues[21] = d21
							ps23.OverlayValues[22] = d22
							return bbs[2].RenderPS(ps23)
						}
						if ps.General {
						}
						ps24 := PhiState{General: ps.General}
						ps24.OverlayValues = make([]JITValueDesc, 23)
						ps24.OverlayValues[3] = d3
						ps24.OverlayValues[4] = d4
						ps24.OverlayValues[5] = d5
						ps24.OverlayValues[6] = d6
						ps24.OverlayValues[7] = d7
						ps24.OverlayValues[9] = d9
						ps24.OverlayValues[10] = d10
						ps24.OverlayValues[11] = d11
						ps24.OverlayValues[12] = d12
						ps24.OverlayValues[13] = d13
						ps24.OverlayValues[14] = d14
						ps24.OverlayValues[15] = d15
						ps24.OverlayValues[16] = d16
						ps24.OverlayValues[18] = d18
						ps24.OverlayValues[19] = d19
						ps24.OverlayValues[20] = d20
						ps24.OverlayValues[21] = d21
						ps24.OverlayValues[22] = d22
						return bbs[3].RenderPS(ps24)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d25 := ps.PhiValues[0]
							ctx.EnsureDesc(&d25)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d25)
							} else {
								ctx.EmitStoreToStack(d25, int32(bbs[1].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d22.Condition, lbl3)
					snap26 := d3
					snap27 := d4
					snap28 := d5
					snap29 := d6
					snap30 := d7
					snap31 := d9
					snap32 := d10
					snap33 := d11
					snap34 := d12
					snap35 := d13
					snap36 := d14
					snap37 := d15
					snap38 := d16
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d21
					snap43 := d22
					snap44 := d25
					alloc45 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc45)
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d6 = snap29
					d7 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d16 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d22 = snap43
					d25 = snap44
					ctx.RestoreAllocState(alloc45)
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d6 = snap29
					d7 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d16 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d22 = snap43
					d25 = snap44
					ps46 := PhiState{General: true}
					ps46.OverlayValues = make([]JITValueDesc, 26)
					ps46.OverlayValues[3] = d3
					ps46.OverlayValues[4] = d4
					ps46.OverlayValues[5] = d5
					ps46.OverlayValues[6] = d6
					ps46.OverlayValues[7] = d7
					ps46.OverlayValues[9] = d9
					ps46.OverlayValues[10] = d10
					ps46.OverlayValues[11] = d11
					ps46.OverlayValues[12] = d12
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[15] = d15
					ps46.OverlayValues[16] = d16
					ps46.OverlayValues[18] = d18
					ps46.OverlayValues[19] = d19
					ps46.OverlayValues[20] = d20
					ps46.OverlayValues[21] = d21
					ps46.OverlayValues[22] = d22
					ps46.OverlayValues[25] = d25
					ps47 := PhiState{General: true}
					ps47.OverlayValues = make([]JITValueDesc, 26)
					ps47.OverlayValues[3] = d3
					ps47.OverlayValues[4] = d4
					ps47.OverlayValues[5] = d5
					ps47.OverlayValues[6] = d6
					ps47.OverlayValues[7] = d7
					ps47.OverlayValues[9] = d9
					ps47.OverlayValues[10] = d10
					ps47.OverlayValues[11] = d11
					ps47.OverlayValues[12] = d12
					ps47.OverlayValues[13] = d13
					ps47.OverlayValues[14] = d14
					ps47.OverlayValues[15] = d15
					ps47.OverlayValues[16] = d16
					ps47.OverlayValues[18] = d18
					ps47.OverlayValues[19] = d19
					ps47.OverlayValues[20] = d20
					ps47.OverlayValues[21] = d21
					ps47.OverlayValues[22] = d22
					ps47.OverlayValues[25] = d25
					snap48 := d3
					snap49 := d4
					snap50 := d5
					snap51 := d6
					snap52 := d7
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d12
					snap57 := d13
					snap58 := d14
					snap59 := d15
					snap60 := d16
					snap61 := d18
					snap62 := d19
					snap63 := d20
					snap64 := d21
					snap65 := d22
					snap66 := d25
					alloc67 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps47)
					}
					ctx.RestoreAllocState(alloc67)
					d3 = snap48
					d4 = snap49
					d5 = snap50
					d6 = snap51
					d7 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d12 = snap56
					d13 = snap57
					d14 = snap58
					d15 = snap59
					d16 = snap60
					d18 = snap61
					d19 = snap62
					d20 = snap63
					d21 = snap64
					d22 = snap65
					d25 = snap66
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps46)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					d69 = ctx.EmitSliceElementAddress(&d5, &d20, 16)
					ctx.EnsureDesc(&d69)
					r6 := ctx.AllocRegExcept(d69.Reg)
					ctx.EmitMovRegMem(r6, d69.Reg, 8)
					ctx.EmitMovRegMem(d69.Reg, d69.Reg, 0)
					d68 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d69.Reg, Reg2: r6}
					ctx.BindReg(d69.Reg, &d68)
					ctx.BindReg(r6, &d68)
					stackArray70 = ctx.AllocStack(int32(16))
					_ = stackArray70
					ctx.SyncDesc(&d68)
					ctx.EmitStoreScmerToStack(d68, int32(stackArray70)+int32(0))
					ctx.FreeDesc(&d68)
					d71 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d71
					callbackArgs73 := make([]JITValueDesc, 1)
					callbackArgs73[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray70) + 0}
					var d72 JITValueDesc
					callbackResultOff74 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff74))
					ctx.FreeDesc(&d71)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs75 := ctx.StabilizeCallbackArgs(callbackArgs73)
						ctx.ReclaimUntrackedRegs()
						outerRegs76 := ctx.PreserveOuterRegs()
						d72 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs75, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff74), ID: 0})
						ctx.RestoreOuterRegs(outerRegs76)
						ctx.ReclaimUntrackedRegs()
					} else {
						d77, knownBuiltin78 := jitEmitKnownDeclaration(ctx, d7, callbackArgs73, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff74), ID: 0})
						if knownBuiltin78 {
							d72 = d77
						} else {
							ctx.Coverage.DynamicCalls++
							d79 := jitCopyScmerToPair(ctx, d7)
							d72 = jitEmitDynamicCallableAt(ctx, d79, callbackArgs73, int32(stackArray70), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff74), ID: 0})
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d72)
					d72 = JITPrepareScmerGoArg(ctx, d72)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d72)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d15, d72})
					ctx.FreeDesc(&d72)
					if ps.General {
						ctx.SyncDesc(&d20)
						if d20.Loc == LocReg {
							ctx.ProtectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.ProtectReg(d20.Reg)
							ctx.ProtectReg(d20.Reg2)
						}
						d80 = d20
						if d80.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d80)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d80)
						} else {
							ctx.EmitStoreToStack(d80, int32(bbs[1].PhiBase)+int32(0))
						}
						if d20.Loc == LocReg {
							ctx.UnprotectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.UnprotectReg(d20.Reg)
							ctx.UnprotectReg(d20.Reg2)
						}
					}
					ps81 := PhiState{General: ps.General}
					ps81.OverlayValues = make([]JITValueDesc, 81)
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[4] = d4
					ps81.OverlayValues[5] = d5
					ps81.OverlayValues[6] = d6
					ps81.OverlayValues[7] = d7
					ps81.OverlayValues[9] = d9
					ps81.OverlayValues[10] = d10
					ps81.OverlayValues[11] = d11
					ps81.OverlayValues[12] = d12
					ps81.OverlayValues[13] = d13
					ps81.OverlayValues[14] = d14
					ps81.OverlayValues[15] = d15
					ps81.OverlayValues[16] = d16
					ps81.OverlayValues[18] = d18
					ps81.OverlayValues[19] = d19
					ps81.OverlayValues[20] = d20
					ps81.OverlayValues[21] = d21
					ps81.OverlayValues[22] = d22
					ps81.OverlayValues[25] = d25
					ps81.OverlayValues[68] = d68
					ps81.OverlayValues[69] = d69
					ps81.OverlayValues[71] = d71
					ps81.OverlayValues[72] = d72
					ps81.OverlayValues[77] = d77
					ps81.OverlayValues[79] = d79
					ps81.OverlayValues[80] = d80
					ps81.PhiValues = make([]JITValueDesc, 1)
					d82 = d20
					ps81.PhiValues[0] = d82
					if ps81.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps81)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					ctx.ReclaimUntrackedRegs()
					var d83 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d83 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d15.Reg, Reg2: r7}
						ctx.BindReg(d15.Reg, &d83)
						ctx.BindReg(r7, &d83)
						ctx.TransferReg(d15.Reg)
						ctx.BindReg(d15.Reg, &d83)
						ctx.BindReg(r7, &d83)
						d15.Loc = LocNone
					}
					ctx.SyncDesc(&d83)
					if d83.Loc == LocRegPair || d83.Loc == LocStackPair || d83.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d83, &result)
						result.Type = d83.Type
					} else {
						switch d83.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d83)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d83)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d83)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d83, &result)
							result.Type = d83.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps84 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps84)
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
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
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
				var d16 JITValueDesc
				_ = d16
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
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var d70 JITValueDesc
				_ = d70
				var stackArray71 int32
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var callbackResultOff75 int32
				var d78 JITValueDesc
				_ = d78
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				registerHomes1 := ctx.AllocRegisterHomes(JITRegisterPlan{Slots: [16]JITRegisterSlot{{Color: 0, Width: 1, Cost: 12}}, Count: 1})
				defer ctx.ReleaseRegisterHomes(registerHomes1)
				var r0 Reg
				phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
				if phiHomeOK2 {
					r0 = registerHomes1.Registers[0]
				}
				var d3 JITValueDesc
				if phiHomeOK2 {
					d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
				} else {
					d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				}
				_ = d3
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					ctx.ReclaimUntrackedRegs()
					d4 = args[0]
					d4.ID = 0
					var d5 JITValueDesc
					if d4.Type == tagSlice {
						d5 = jitKnownSliceHeader(ctx, &d4)
					} else {
						d5 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d4}, 3)
					}
					ctx.BindReg(d5.Reg, &d5)
					ctx.BindReg(d5.Reg2, &d5)
					ctx.BindReg(d5.Reg3, &d5)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.FreeDesc(&d4)
					d6 = args[1]
					d6.ID = 0
					var d7 JITValueDesc
					if d6.Loc == LocLambdaTemplate {
						d7 = d6
					} else if d6.Loc == LocImm {
						optimizedCallback8 := NewFunc(OptimizeProcToSerialFunction(d6.Imm))
						ctx.TrackImm(optimizedCallback8)
						d7 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback8, Rooted: true}
					} else {
						if d6.Loc == LocInputPair && int(d6.StackOff) < ctx.InputArgCount {
							d7 = ctx.RequestOptimizedCallback(int(d6.StackOff))
						} else {
							d7 = jitCopyScmerToPair(ctx, d6)
						}
					}
					ctx.StabilizeDescForControlFlow(&d7)
					ctx.FreeDesc(&d6)
					var d9 JITValueDesc
					if d5.SliceSizeKnown {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d9)
					d10 = d9
					_ = d10
					ctx.StabilizeDescForControlFlow(&d10)
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
					ctx.EnsureDesc(&d10)
					var d11 JITValueDesc
					if d10.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() < 32)}
					} else {
						r1 := ctx.AllocRegExcept(d10.Reg)
						ctx.EmitCmpRegImm32(d10.Reg, 32)
						d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondSignedLess}
						ctx.BindReg(r1, &d11)
					}
					ctx.ReclaimUntrackedRegs()
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d12.Loc == LocImm {
						if d12.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d12.Condition, lbl9)
						ctx.EmitJmp(lbl10)
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
					r2 := ctx.AllocReg()
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d13)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r2, d10)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d14 = JITValueDesc{Loc: LocReg, Reg: r2}
					ctx.BindReg(r2, &d14)
					ctx.BindReg(r2, &d14)
					ctx.FreeDesc(&d9)
					ctx.EnsureDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d14}, 1)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.FreeDesc(&d14)
					var d16 JITValueDesc
					if d5.SliceSizeKnown {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.KnownSliceLen))}
					} else if d5.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d5.StackOff))}
					} else if d5.Loc == LocStackTriple {
						d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d5.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d5)
						if d5.Loc == LocRegPair || d5.Loc == LocRegTriple {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg2, ID: 0}
						} else if d5.Loc == LocReg {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d5.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d16)
					if ps.General {
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)})
						} else {
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
						}
					}
					ps17 := PhiState{General: ps.General}
					ps17.OverlayValues = make([]JITValueDesc, 17)
					ps17.OverlayValues[3] = d3
					ps17.OverlayValues[4] = d4
					ps17.OverlayValues[5] = d5
					ps17.OverlayValues[6] = d6
					ps17.OverlayValues[7] = d7
					ps17.OverlayValues[9] = d9
					ps17.OverlayValues[10] = d10
					ps17.OverlayValues[11] = d11
					ps17.OverlayValues[12] = d12
					ps17.OverlayValues[13] = d13
					ps17.OverlayValues[14] = d14
					ps17.OverlayValues[15] = d15
					ps17.OverlayValues[16] = d16
					ps17.PhiValues = make([]JITValueDesc, 1)
					d18 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps17.PhiValues[0] = d18
					if ps17.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps17)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d19 := ps.PhiValues[0]
							ctx.EnsureDesc(&d19)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d19)
							} else {
								ctx.EmitStoreToStack(d19, int32(bbs[1].PhiBase)+int32(0))
							}
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d3 = ps.PhiValues[0]
					}
					if phiHomeOK2 && d3.Loc == LocReg {
						ctx.BindReg(r0, &d3)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					var d20 JITValueDesc
					if d3.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d3.Reg)
						ctx.EmitMovRegReg(scratch, d3.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d20)
					}
					if d20.Loc == LocReg && d3.Loc == LocReg && d20.Reg == d3.Reg {
						ctx.TransferReg(d3.Reg)
						d3.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDescsTogether(&d20, &d16)
					var d21 JITValueDesc
					if d20.Loc == LocImm && d16.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.Int() < d16.Imm.Int())}
					} else if d16.Loc == LocImm {
						r3 := ctx.AllocRegExcept(d20.Reg)
						if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d20.Reg, int32(d16.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
							ctx.EmitCmpInt64(d20.Reg, RegR11)
						}
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d21)
					} else if d20.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d16.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d21)
					} else {
						r5 := ctx.AllocRegExcept(d20.Reg)
						ctx.EmitCmpInt64(d20.Reg, d16.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r5, Condition: CondSignedLess}
						ctx.BindReg(r5, &d21)
					}
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
							ps23.OverlayValues[3] = d3
							ps23.OverlayValues[4] = d4
							ps23.OverlayValues[5] = d5
							ps23.OverlayValues[6] = d6
							ps23.OverlayValues[7] = d7
							ps23.OverlayValues[9] = d9
							ps23.OverlayValues[10] = d10
							ps23.OverlayValues[11] = d11
							ps23.OverlayValues[12] = d12
							ps23.OverlayValues[13] = d13
							ps23.OverlayValues[14] = d14
							ps23.OverlayValues[15] = d15
							ps23.OverlayValues[16] = d16
							ps23.OverlayValues[18] = d18
							ps23.OverlayValues[19] = d19
							ps23.OverlayValues[20] = d20
							ps23.OverlayValues[21] = d21
							ps23.OverlayValues[22] = d22
							return bbs[2].RenderPS(ps23)
						}
						if ps.General {
						}
						ps24 := PhiState{General: ps.General}
						ps24.OverlayValues = make([]JITValueDesc, 23)
						ps24.OverlayValues[3] = d3
						ps24.OverlayValues[4] = d4
						ps24.OverlayValues[5] = d5
						ps24.OverlayValues[6] = d6
						ps24.OverlayValues[7] = d7
						ps24.OverlayValues[9] = d9
						ps24.OverlayValues[10] = d10
						ps24.OverlayValues[11] = d11
						ps24.OverlayValues[12] = d12
						ps24.OverlayValues[13] = d13
						ps24.OverlayValues[14] = d14
						ps24.OverlayValues[15] = d15
						ps24.OverlayValues[16] = d16
						ps24.OverlayValues[18] = d18
						ps24.OverlayValues[19] = d19
						ps24.OverlayValues[20] = d20
						ps24.OverlayValues[21] = d21
						ps24.OverlayValues[22] = d22
						return bbs[3].RenderPS(ps24)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d25 := ps.PhiValues[0]
							ctx.EnsureDesc(&d25)
							if phiHomeOK2 {
								ctx.EmitMovToReg(r0, d25)
							} else {
								ctx.EmitStoreToStack(d25, int32(bbs[1].PhiBase)+int32(0))
							}
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d22.Condition, lbl3)
					snap26 := d3
					snap27 := d4
					snap28 := d5
					snap29 := d6
					snap30 := d7
					snap31 := d9
					snap32 := d10
					snap33 := d11
					snap34 := d12
					snap35 := d13
					snap36 := d14
					snap37 := d15
					snap38 := d16
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d21
					snap43 := d22
					snap44 := d25
					alloc45 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc45)
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d6 = snap29
					d7 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d16 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d22 = snap43
					d25 = snap44
					ctx.RestoreAllocState(alloc45)
					d3 = snap26
					d4 = snap27
					d5 = snap28
					d6 = snap29
					d7 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d16 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d22 = snap43
					d25 = snap44
					ps46 := PhiState{General: true}
					ps46.OverlayValues = make([]JITValueDesc, 26)
					ps46.OverlayValues[3] = d3
					ps46.OverlayValues[4] = d4
					ps46.OverlayValues[5] = d5
					ps46.OverlayValues[6] = d6
					ps46.OverlayValues[7] = d7
					ps46.OverlayValues[9] = d9
					ps46.OverlayValues[10] = d10
					ps46.OverlayValues[11] = d11
					ps46.OverlayValues[12] = d12
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[15] = d15
					ps46.OverlayValues[16] = d16
					ps46.OverlayValues[18] = d18
					ps46.OverlayValues[19] = d19
					ps46.OverlayValues[20] = d20
					ps46.OverlayValues[21] = d21
					ps46.OverlayValues[22] = d22
					ps46.OverlayValues[25] = d25
					ps47 := PhiState{General: true}
					ps47.OverlayValues = make([]JITValueDesc, 26)
					ps47.OverlayValues[3] = d3
					ps47.OverlayValues[4] = d4
					ps47.OverlayValues[5] = d5
					ps47.OverlayValues[6] = d6
					ps47.OverlayValues[7] = d7
					ps47.OverlayValues[9] = d9
					ps47.OverlayValues[10] = d10
					ps47.OverlayValues[11] = d11
					ps47.OverlayValues[12] = d12
					ps47.OverlayValues[13] = d13
					ps47.OverlayValues[14] = d14
					ps47.OverlayValues[15] = d15
					ps47.OverlayValues[16] = d16
					ps47.OverlayValues[18] = d18
					ps47.OverlayValues[19] = d19
					ps47.OverlayValues[20] = d20
					ps47.OverlayValues[21] = d21
					ps47.OverlayValues[22] = d22
					ps47.OverlayValues[25] = d25
					snap48 := d3
					snap49 := d4
					snap50 := d5
					snap51 := d6
					snap52 := d7
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d12
					snap57 := d13
					snap58 := d14
					snap59 := d15
					snap60 := d16
					snap61 := d18
					snap62 := d19
					snap63 := d20
					snap64 := d21
					snap65 := d22
					snap66 := d25
					alloc67 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps47)
					}
					ctx.RestoreAllocState(alloc67)
					d3 = snap48
					d4 = snap49
					d5 = snap50
					d6 = snap51
					d7 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d12 = snap56
					d13 = snap57
					d14 = snap58
					d15 = snap59
					d16 = snap60
					d18 = snap61
					d19 = snap62
					d20 = snap63
					d21 = snap64
					d22 = snap65
					d25 = snap66
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps46)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d20)
					d69 = ctx.EmitSliceElementAddress(&d5, &d20, 16)
					ctx.EnsureDesc(&d69)
					r6 := ctx.AllocRegExcept(d69.Reg)
					ctx.EmitMovRegMem(r6, d69.Reg, 8)
					ctx.EmitMovRegMem(d69.Reg, d69.Reg, 0)
					d68 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d69.Reg, Reg2: r6}
					ctx.BindReg(d69.Reg, &d68)
					ctx.BindReg(r6, &d68)
					d70 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray71 = ctx.AllocStack(int32(32))
					_ = stackArray71
					ctx.SyncDesc(&d70)
					ctx.EmitStoreScmerToStack(d70, int32(stackArray71)+int32(0))
					ctx.FreeDesc(&d70)
					ctx.SyncDesc(&d68)
					ctx.EmitStoreScmerToStack(d68, int32(stackArray71)+int32(16))
					ctx.FreeDesc(&d68)
					d72 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d72
					callbackArgs74 := make([]JITValueDesc, 2)
					callbackArgs74[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray71) + 0}
					callbackArgs74[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray71) + 16}
					var d73 JITValueDesc
					callbackResultOff75 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff75))
					ctx.FreeDesc(&d72)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs76 := ctx.StabilizeCallbackArgs(callbackArgs74)
						ctx.ReclaimUntrackedRegs()
						outerRegs77 := ctx.PreserveOuterRegs()
						d73 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs76, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff75), ID: 0})
						ctx.RestoreOuterRegs(outerRegs77)
						ctx.ReclaimUntrackedRegs()
					} else {
						d78, knownBuiltin79 := jitEmitKnownDeclaration(ctx, d7, callbackArgs74, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff75), ID: 0})
						if knownBuiltin79 {
							d73 = d78
						} else {
							ctx.Coverage.DynamicCalls++
							d80 := jitCopyScmerToPair(ctx, d7)
							d73 = jitEmitDynamicCallableAt(ctx, d80, callbackArgs74, int32(stackArray71), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff75), ID: 0})
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					d73 = JITPrepareScmerGoArg(ctx, d73)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d73)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d15, d73})
					ctx.FreeDesc(&d73)
					if ps.General {
						ctx.SyncDesc(&d20)
						if d20.Loc == LocReg {
							ctx.ProtectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.ProtectReg(d20.Reg)
							ctx.ProtectReg(d20.Reg2)
						}
						d81 = d20
						if d81.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d81)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d81)
						} else {
							ctx.EmitStoreToStack(d81, int32(bbs[1].PhiBase)+int32(0))
						}
						if d20.Loc == LocReg {
							ctx.UnprotectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.UnprotectReg(d20.Reg)
							ctx.UnprotectReg(d20.Reg2)
						}
					}
					ps82 := PhiState{General: ps.General}
					ps82.OverlayValues = make([]JITValueDesc, 82)
					ps82.OverlayValues[3] = d3
					ps82.OverlayValues[4] = d4
					ps82.OverlayValues[5] = d5
					ps82.OverlayValues[6] = d6
					ps82.OverlayValues[7] = d7
					ps82.OverlayValues[9] = d9
					ps82.OverlayValues[10] = d10
					ps82.OverlayValues[11] = d11
					ps82.OverlayValues[12] = d12
					ps82.OverlayValues[13] = d13
					ps82.OverlayValues[14] = d14
					ps82.OverlayValues[15] = d15
					ps82.OverlayValues[16] = d16
					ps82.OverlayValues[18] = d18
					ps82.OverlayValues[19] = d19
					ps82.OverlayValues[20] = d20
					ps82.OverlayValues[21] = d21
					ps82.OverlayValues[22] = d22
					ps82.OverlayValues[25] = d25
					ps82.OverlayValues[68] = d68
					ps82.OverlayValues[69] = d69
					ps82.OverlayValues[70] = d70
					ps82.OverlayValues[72] = d72
					ps82.OverlayValues[73] = d73
					ps82.OverlayValues[78] = d78
					ps82.OverlayValues[80] = d80
					ps82.OverlayValues[81] = d81
					ps82.PhiValues = make([]JITValueDesc, 1)
					d83 = d20
					ps82.PhiValues[0] = d83
					if ps82.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps82)
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
					if phiHomeOK2 {
						d3 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r0, ID: 0}
					} else {
						d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					}
					if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
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
					if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
						d16 = ps.OverlayValues[16]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
					}
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					ctx.ReclaimUntrackedRegs()
					var d84 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d84 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d15.Reg, Reg2: r7}
						ctx.BindReg(d15.Reg, &d84)
						ctx.BindReg(r7, &d84)
						ctx.TransferReg(d15.Reg)
						ctx.BindReg(d15.Reg, &d84)
						ctx.BindReg(r7, &d84)
						d15.Loc = LocNone
					}
					ctx.SyncDesc(&d84)
					if d84.Loc == LocRegPair || d84.Loc == LocStackPair || d84.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d84, &result)
						result.Type = d84.Type
					} else {
						switch d84.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d84)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d84)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d84)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d84, &result)
							result.Type = d84.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps85 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps85)
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
