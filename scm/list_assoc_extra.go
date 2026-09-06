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
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var stackArray55 int32
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var callbackResultOff59 int32
				var d62 JITValueDesc
				_ = d62
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
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
						ctx.EmitSetcc(r1, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d15.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d14)
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
						ctx.EmitSetcc(r3, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d24)
					} else if d23.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d24)
					} else {
						r5 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						ctx.EmitSetcc(r5, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d24)
					}
					ctx.FreeDesc(&d19)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 29)
					ps29.OverlayValues[3] = d3
					ps29.OverlayValues[4] = d4
					ps29.OverlayValues[5] = d5
					ps29.OverlayValues[6] = d6
					ps29.OverlayValues[7] = d7
					ps29.OverlayValues[9] = d9
					ps29.OverlayValues[10] = d10
					ps29.OverlayValues[12] = d12
					ps29.OverlayValues[13] = d13
					ps29.OverlayValues[14] = d14
					ps29.OverlayValues[15] = d15
					ps29.OverlayValues[16] = d16
					ps29.OverlayValues[17] = d17
					ps29.OverlayValues[18] = d18
					ps29.OverlayValues[19] = d19
					ps29.OverlayValues[21] = d21
					ps29.OverlayValues[22] = d22
					ps29.OverlayValues[23] = d23
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[25] = d25
					ps29.OverlayValues[28] = d28
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 29)
					ps30.OverlayValues[3] = d3
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[6] = d6
					ps30.OverlayValues[7] = d7
					ps30.OverlayValues[9] = d9
					ps30.OverlayValues[10] = d10
					ps30.OverlayValues[12] = d12
					ps30.OverlayValues[13] = d13
					ps30.OverlayValues[14] = d14
					ps30.OverlayValues[15] = d15
					ps30.OverlayValues[16] = d16
					ps30.OverlayValues[17] = d17
					ps30.OverlayValues[18] = d18
					ps30.OverlayValues[19] = d19
					ps30.OverlayValues[21] = d21
					ps30.OverlayValues[22] = d22
					ps30.OverlayValues[23] = d23
					ps30.OverlayValues[24] = d24
					ps30.OverlayValues[25] = d25
					ps30.OverlayValues[28] = d28
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d6
					snap35 := d7
					snap36 := d9
					snap37 := d10
					snap38 := d12
					snap39 := d13
					snap40 := d14
					snap41 := d15
					snap42 := d16
					snap43 := d17
					snap44 := d18
					snap45 := d19
					snap46 := d21
					snap47 := d22
					snap48 := d23
					snap49 := d24
					snap50 := d25
					snap51 := d28
					alloc52 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps30)
					}
					ctx.RestoreAllocState(alloc52)
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d6 = snap34
					d7 = snap35
					d9 = snap36
					d10 = snap37
					d12 = snap38
					d13 = snap39
					d14 = snap40
					d15 = snap41
					d16 = snap42
					d17 = snap43
					d18 = snap44
					d19 = snap45
					d21 = snap46
					d22 = snap47
					d23 = snap48
					d24 = snap49
					d25 = snap50
					d28 = snap51
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps29)
					}
					return result
					ctx.FreeDesc(&d24)
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
					d54 = ctx.EmitSliceElementAddress(&d5, &d23, 16)
					ctx.EnsureDesc(&d54)
					r6 := ctx.AllocRegExcept(d54.Reg)
					ctx.EmitMovRegMem(r6, d54.Reg, 8)
					ctx.EmitMovRegMem(d54.Reg, d54.Reg, 0)
					d53 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d54.Reg, Reg2: r6}
					ctx.BindReg(d54.Reg, &d53)
					ctx.BindReg(r6, &d53)
					stackArray55 = ctx.AllocStack(int32(16))
					_ = stackArray55
					ctx.SyncDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray55)+int32(0))
					d56 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d56
					callbackArgs58 := make([]JITValueDesc, 1)
					callbackArgs58[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray55) + 0}
					var d57 JITValueDesc
					callbackResultOff59 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff59))
					ctx.FreeDesc(&d56)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs60 := ctx.StabilizeCallbackArgs(callbackArgs58)
						ctx.ReclaimUntrackedRegs()
						outerRegs61 := ctx.PreserveOuterRegs()
						d57 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs60, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						ctx.RestoreOuterRegs(outerRegs61)
						ctx.ReclaimUntrackedRegs()
					} else {
						d62, knownBuiltin63 := jitEmitKnownDeclaration(ctx, d7, callbackArgs58, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						if knownBuiltin63 {
							d57 = d62
						} else {
							ctx.Coverage.DynamicCalls++
							d64 := jitCopyScmerToPair(ctx, d7)
							d57 = jitEmitDynamicCallableAt(ctx, d64, callbackArgs58, int32(stackArray55), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						}
					}
					d65 = args[3]
					d65.ID = 0
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d57)
					ctx.EnsureDesc(&d57)
					d57 = JITPrepareScmerGoArg(ctx, d57)
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d53)
					d53 = JITPrepareScmerGoArg(ctx, d53)
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					d65 = JITPrepareScmerGoArg(ctx, d65)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocRegPair || d10.Loc == LocStackPair || d10.Loc == LocRegTriple || d10.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d57)
					ctx.SyncDesc(&d53)
					ctx.SyncDesc(&d65)
					ctx.SyncDesc(&d10)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).ReduceValue), []JITValueDesc{d18, d57, d53, d65, d10})
					ctx.FreeDesc(&d57)
					ctx.FreeDesc(&d53)
					ctx.FreeDesc(&d65)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d66 = d23
						if d66.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d66)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d66)
						} else {
							ctx.EmitStoreToStack(d66, int32(bbs[1].PhiBase)+int32(0))
						}
						if d23.Loc == LocReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps67 := PhiState{General: ps.General}
					ps67.OverlayValues = make([]JITValueDesc, 67)
					ps67.OverlayValues[3] = d3
					ps67.OverlayValues[4] = d4
					ps67.OverlayValues[5] = d5
					ps67.OverlayValues[6] = d6
					ps67.OverlayValues[7] = d7
					ps67.OverlayValues[9] = d9
					ps67.OverlayValues[10] = d10
					ps67.OverlayValues[12] = d12
					ps67.OverlayValues[13] = d13
					ps67.OverlayValues[14] = d14
					ps67.OverlayValues[15] = d15
					ps67.OverlayValues[16] = d16
					ps67.OverlayValues[17] = d17
					ps67.OverlayValues[18] = d18
					ps67.OverlayValues[19] = d19
					ps67.OverlayValues[21] = d21
					ps67.OverlayValues[22] = d22
					ps67.OverlayValues[23] = d23
					ps67.OverlayValues[24] = d24
					ps67.OverlayValues[25] = d25
					ps67.OverlayValues[28] = d28
					ps67.OverlayValues[53] = d53
					ps67.OverlayValues[54] = d54
					ps67.OverlayValues[56] = d56
					ps67.OverlayValues[57] = d57
					ps67.OverlayValues[62] = d62
					ps67.OverlayValues[64] = d64
					ps67.OverlayValues[65] = d65
					ps67.OverlayValues[66] = d66
					ps67.PhiValues = make([]JITValueDesc, 1)
					d68 = d23
					ps67.PhiValues[0] = d68
					if ps67.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps67)
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					ctx.ReclaimUntrackedRegs()
					var d69 JITValueDesc
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d69 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r7}
						ctx.BindReg(d18.Reg, &d69)
						ctx.BindReg(r7, &d69)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d69)
						ctx.BindReg(r7, &d69)
						d18.Loc = LocNone
					}
					ctx.FreeDesc(&d18)
					ctx.SyncDesc(&d69)
					if d69.Loc == LocRegPair || d69.Loc == LocStackPair || d69.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d69, &result)
						result.Type = d69.Type
					} else {
						switch d69.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d69)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d69)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d69)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d69, &result)
							result.Type = d69.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps70 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps70)
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
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var stackArray55 int32
				var d56 JITValueDesc
				_ = d56
				var d57 JITValueDesc
				_ = d57
				var callbackResultOff59 int32
				var d62 JITValueDesc
				_ = d62
				var d64 JITValueDesc
				_ = d64
				var d65 JITValueDesc
				_ = d65
				var stackArray66 int32
				var d67 JITValueDesc
				_ = d67
				var d68 JITValueDesc
				_ = d68
				var callbackResultOff70 int32
				var d73 JITValueDesc
				_ = d73
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
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
						ctx.EmitSetcc(r1, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d15.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d14)
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
						ctx.EmitSetcc(r3, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d24)
					} else if d23.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d24)
					} else {
						r5 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						ctx.EmitSetcc(r5, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d24)
					}
					ctx.FreeDesc(&d19)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 29)
					ps29.OverlayValues[3] = d3
					ps29.OverlayValues[4] = d4
					ps29.OverlayValues[5] = d5
					ps29.OverlayValues[6] = d6
					ps29.OverlayValues[7] = d7
					ps29.OverlayValues[9] = d9
					ps29.OverlayValues[10] = d10
					ps29.OverlayValues[12] = d12
					ps29.OverlayValues[13] = d13
					ps29.OverlayValues[14] = d14
					ps29.OverlayValues[15] = d15
					ps29.OverlayValues[16] = d16
					ps29.OverlayValues[17] = d17
					ps29.OverlayValues[18] = d18
					ps29.OverlayValues[19] = d19
					ps29.OverlayValues[21] = d21
					ps29.OverlayValues[22] = d22
					ps29.OverlayValues[23] = d23
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[25] = d25
					ps29.OverlayValues[28] = d28
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 29)
					ps30.OverlayValues[3] = d3
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[6] = d6
					ps30.OverlayValues[7] = d7
					ps30.OverlayValues[9] = d9
					ps30.OverlayValues[10] = d10
					ps30.OverlayValues[12] = d12
					ps30.OverlayValues[13] = d13
					ps30.OverlayValues[14] = d14
					ps30.OverlayValues[15] = d15
					ps30.OverlayValues[16] = d16
					ps30.OverlayValues[17] = d17
					ps30.OverlayValues[18] = d18
					ps30.OverlayValues[19] = d19
					ps30.OverlayValues[21] = d21
					ps30.OverlayValues[22] = d22
					ps30.OverlayValues[23] = d23
					ps30.OverlayValues[24] = d24
					ps30.OverlayValues[25] = d25
					ps30.OverlayValues[28] = d28
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d6
					snap35 := d7
					snap36 := d9
					snap37 := d10
					snap38 := d12
					snap39 := d13
					snap40 := d14
					snap41 := d15
					snap42 := d16
					snap43 := d17
					snap44 := d18
					snap45 := d19
					snap46 := d21
					snap47 := d22
					snap48 := d23
					snap49 := d24
					snap50 := d25
					snap51 := d28
					alloc52 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps30)
					}
					ctx.RestoreAllocState(alloc52)
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d6 = snap34
					d7 = snap35
					d9 = snap36
					d10 = snap37
					d12 = snap38
					d13 = snap39
					d14 = snap40
					d15 = snap41
					d16 = snap42
					d17 = snap43
					d18 = snap44
					d19 = snap45
					d21 = snap46
					d22 = snap47
					d23 = snap48
					d24 = snap49
					d25 = snap50
					d28 = snap51
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps29)
					}
					return result
					ctx.FreeDesc(&d24)
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
					d54 = ctx.EmitSliceElementAddress(&d5, &d23, 16)
					ctx.EnsureDesc(&d54)
					r6 := ctx.AllocRegExcept(d54.Reg)
					ctx.EmitMovRegMem(r6, d54.Reg, 8)
					ctx.EmitMovRegMem(d54.Reg, d54.Reg, 0)
					d53 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d54.Reg, Reg2: r6}
					ctx.BindReg(d54.Reg, &d53)
					ctx.BindReg(r6, &d53)
					stackArray55 = ctx.AllocStack(int32(16))
					_ = stackArray55
					ctx.SyncDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray55)+int32(0))
					d56 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d56
					callbackArgs58 := make([]JITValueDesc, 1)
					callbackArgs58[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray55) + 0}
					var d57 JITValueDesc
					callbackResultOff59 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff59))
					ctx.FreeDesc(&d56)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs60 := ctx.StabilizeCallbackArgs(callbackArgs58)
						ctx.ReclaimUntrackedRegs()
						outerRegs61 := ctx.PreserveOuterRegs()
						d57 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs60, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						ctx.RestoreOuterRegs(outerRegs61)
						ctx.ReclaimUntrackedRegs()
					} else {
						d62, knownBuiltin63 := jitEmitKnownDeclaration(ctx, d7, callbackArgs58, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						if knownBuiltin63 {
							d57 = d62
						} else {
							ctx.Coverage.DynamicCalls++
							d64 := jitCopyScmerToPair(ctx, d7)
							d57 = jitEmitDynamicCallableAt(ctx, d64, callbackArgs58, int32(stackArray55), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						}
					}
					d65 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray66 = ctx.AllocStack(int32(32))
					_ = stackArray66
					ctx.SyncDesc(&d65)
					ctx.EmitStoreScmerToStack(d65, int32(stackArray66)+int32(0))
					ctx.FreeDesc(&d65)
					ctx.SyncDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray66)+int32(16))
					ctx.FreeDesc(&d53)
					d67 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d67
					callbackArgs69 := make([]JITValueDesc, 2)
					callbackArgs69[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray66) + 0}
					callbackArgs69[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray66) + 16}
					var d68 JITValueDesc
					callbackResultOff70 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff70))
					ctx.FreeDesc(&d67)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d10.Loc == LocLambdaTemplate && d10.Lambda != nil {
						stableCallbackArgs71 := ctx.StabilizeCallbackArgs(callbackArgs69)
						ctx.ReclaimUntrackedRegs()
						outerRegs72 := ctx.PreserveOuterRegs()
						d68 = JITEmitProcInlineWithOuter(ctx, &d10.Lambda.Proc, d10.Lambda.Outer, stableCallbackArgs71, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff70), ID: 0})
						ctx.RestoreOuterRegs(outerRegs72)
						ctx.ReclaimUntrackedRegs()
					} else {
						d73, knownBuiltin74 := jitEmitKnownDeclaration(ctx, d10, callbackArgs69, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff70), ID: 0})
						if knownBuiltin74 {
							d68 = d73
						} else {
							ctx.Coverage.DynamicCalls++
							d75 := jitCopyScmerToPair(ctx, d10)
							d68 = jitEmitDynamicCallableAt(ctx, d75, callbackArgs69, int32(stackArray66), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff70), ID: 0})
						}
					}
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d57)
					ctx.EnsureDesc(&d57)
					d57 = JITPrepareScmerGoArg(ctx, d57)
					ctx.EnsureDesc(&d68)
					ctx.EnsureDesc(&d68)
					d68 = JITPrepareScmerGoArg(ctx, d68)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d57)
					ctx.SyncDesc(&d68)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d57, d68})
					ctx.FreeDesc(&d57)
					ctx.FreeDesc(&d68)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d76 = d23
						if d76.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d76)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d76)
						} else {
							ctx.EmitStoreToStack(d76, int32(bbs[1].PhiBase)+int32(0))
						}
						if d23.Loc == LocReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps77 := PhiState{General: ps.General}
					ps77.OverlayValues = make([]JITValueDesc, 77)
					ps77.OverlayValues[3] = d3
					ps77.OverlayValues[4] = d4
					ps77.OverlayValues[5] = d5
					ps77.OverlayValues[6] = d6
					ps77.OverlayValues[7] = d7
					ps77.OverlayValues[9] = d9
					ps77.OverlayValues[10] = d10
					ps77.OverlayValues[12] = d12
					ps77.OverlayValues[13] = d13
					ps77.OverlayValues[14] = d14
					ps77.OverlayValues[15] = d15
					ps77.OverlayValues[16] = d16
					ps77.OverlayValues[17] = d17
					ps77.OverlayValues[18] = d18
					ps77.OverlayValues[19] = d19
					ps77.OverlayValues[21] = d21
					ps77.OverlayValues[22] = d22
					ps77.OverlayValues[23] = d23
					ps77.OverlayValues[24] = d24
					ps77.OverlayValues[25] = d25
					ps77.OverlayValues[28] = d28
					ps77.OverlayValues[53] = d53
					ps77.OverlayValues[54] = d54
					ps77.OverlayValues[56] = d56
					ps77.OverlayValues[57] = d57
					ps77.OverlayValues[62] = d62
					ps77.OverlayValues[64] = d64
					ps77.OverlayValues[65] = d65
					ps77.OverlayValues[67] = d67
					ps77.OverlayValues[68] = d68
					ps77.OverlayValues[73] = d73
					ps77.OverlayValues[75] = d75
					ps77.OverlayValues[76] = d76
					ps77.PhiValues = make([]JITValueDesc, 1)
					d78 = d23
					ps77.PhiValues[0] = d78
					if ps77.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps77)
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
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
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
					ctx.ReclaimUntrackedRegs()
					var d79 JITValueDesc
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d79 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r7}
						ctx.BindReg(d18.Reg, &d79)
						ctx.BindReg(r7, &d79)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d79)
						ctx.BindReg(r7, &d79)
						d18.Loc = LocNone
					}
					ctx.FreeDesc(&d18)
					ctx.SyncDesc(&d79)
					if d79.Loc == LocRegPair || d79.Loc == LocStackPair || d79.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d79, &result)
						result.Type = d79.Type
					} else {
						switch d79.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d79)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d79)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d79)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d79, &result)
							result.Type = d79.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps80 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps80)
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
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var stackArray56 int32
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var callbackResultOff60 int32
				var d63 JITValueDesc
				_ = d63
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var stackArray67 int32
				var d68 JITValueDesc
				_ = d68
				var d69 JITValueDesc
				_ = d69
				var callbackResultOff71 int32
				var d74 JITValueDesc
				_ = d74
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
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
						ctx.EmitSetcc(r1, CondSignedLess)
						d14 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d14)
					}
					ctx.ReclaimUntrackedRegs()
					d15 = d14
					ctx.EnsureDesc(&d15)
					if d15.Loc != LocImm && d15.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d15.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d14)
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
						ctx.EmitSetcc(r3, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d24)
					} else if d23.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d24)
					} else {
						r5 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						ctx.EmitSetcc(r5, CondSignedLess)
						d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d24)
					}
					ctx.FreeDesc(&d19)
					d25 = d24
					ctx.EnsureDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d25.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps29 := PhiState{General: true}
					ps29.OverlayValues = make([]JITValueDesc, 29)
					ps29.OverlayValues[3] = d3
					ps29.OverlayValues[4] = d4
					ps29.OverlayValues[5] = d5
					ps29.OverlayValues[6] = d6
					ps29.OverlayValues[7] = d7
					ps29.OverlayValues[9] = d9
					ps29.OverlayValues[10] = d10
					ps29.OverlayValues[12] = d12
					ps29.OverlayValues[13] = d13
					ps29.OverlayValues[14] = d14
					ps29.OverlayValues[15] = d15
					ps29.OverlayValues[16] = d16
					ps29.OverlayValues[17] = d17
					ps29.OverlayValues[18] = d18
					ps29.OverlayValues[19] = d19
					ps29.OverlayValues[21] = d21
					ps29.OverlayValues[22] = d22
					ps29.OverlayValues[23] = d23
					ps29.OverlayValues[24] = d24
					ps29.OverlayValues[25] = d25
					ps29.OverlayValues[28] = d28
					ps30 := PhiState{General: true}
					ps30.OverlayValues = make([]JITValueDesc, 29)
					ps30.OverlayValues[3] = d3
					ps30.OverlayValues[4] = d4
					ps30.OverlayValues[5] = d5
					ps30.OverlayValues[6] = d6
					ps30.OverlayValues[7] = d7
					ps30.OverlayValues[9] = d9
					ps30.OverlayValues[10] = d10
					ps30.OverlayValues[12] = d12
					ps30.OverlayValues[13] = d13
					ps30.OverlayValues[14] = d14
					ps30.OverlayValues[15] = d15
					ps30.OverlayValues[16] = d16
					ps30.OverlayValues[17] = d17
					ps30.OverlayValues[18] = d18
					ps30.OverlayValues[19] = d19
					ps30.OverlayValues[21] = d21
					ps30.OverlayValues[22] = d22
					ps30.OverlayValues[23] = d23
					ps30.OverlayValues[24] = d24
					ps30.OverlayValues[25] = d25
					ps30.OverlayValues[28] = d28
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d6
					snap35 := d7
					snap36 := d9
					snap37 := d10
					snap38 := d12
					snap39 := d13
					snap40 := d14
					snap41 := d15
					snap42 := d16
					snap43 := d17
					snap44 := d18
					snap45 := d19
					snap46 := d21
					snap47 := d22
					snap48 := d23
					snap49 := d24
					snap50 := d25
					snap51 := d28
					alloc52 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps30)
					}
					ctx.RestoreAllocState(alloc52)
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d6 = snap34
					d7 = snap35
					d9 = snap36
					d10 = snap37
					d12 = snap38
					d13 = snap39
					d14 = snap40
					d15 = snap41
					d16 = snap42
					d17 = snap43
					d18 = snap44
					d19 = snap45
					d21 = snap46
					d22 = snap47
					d23 = snap48
					d24 = snap49
					d25 = snap50
					d28 = snap51
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps29)
					}
					return result
					ctx.FreeDesc(&d24)
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
					d54 = ctx.EmitSliceElementAddress(&d5, &d23, 16)
					ctx.EnsureDesc(&d54)
					r6 := ctx.AllocRegExcept(d54.Reg)
					ctx.EmitMovRegMem(r6, d54.Reg, 8)
					ctx.EmitMovRegMem(d54.Reg, d54.Reg, 0)
					d53 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d54.Reg, Reg2: r6}
					ctx.BindReg(d54.Reg, &d53)
					ctx.BindReg(r6, &d53)
					d55 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray56 = ctx.AllocStack(int32(32))
					_ = stackArray56
					ctx.SyncDesc(&d55)
					ctx.EmitStoreScmerToStack(d55, int32(stackArray56)+int32(0))
					ctx.FreeDesc(&d55)
					ctx.SyncDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray56)+int32(16))
					d57 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d57
					callbackArgs59 := make([]JITValueDesc, 2)
					callbackArgs59[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray56) + 0}
					callbackArgs59[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray56) + 16}
					var d58 JITValueDesc
					callbackResultOff60 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff60))
					ctx.FreeDesc(&d57)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs61 := ctx.StabilizeCallbackArgs(callbackArgs59)
						ctx.ReclaimUntrackedRegs()
						outerRegs62 := ctx.PreserveOuterRegs()
						d58 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs61, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff60), ID: 0})
						ctx.RestoreOuterRegs(outerRegs62)
						ctx.ReclaimUntrackedRegs()
					} else {
						d63, knownBuiltin64 := jitEmitKnownDeclaration(ctx, d7, callbackArgs59, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff60), ID: 0})
						if knownBuiltin64 {
							d58 = d63
						} else {
							ctx.Coverage.DynamicCalls++
							d65 := jitCopyScmerToPair(ctx, d7)
							d58 = jitEmitDynamicCallableAt(ctx, d65, callbackArgs59, int32(stackArray56), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff60), ID: 0})
						}
					}
					d66 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray67 = ctx.AllocStack(int32(32))
					_ = stackArray67
					ctx.SyncDesc(&d66)
					ctx.EmitStoreScmerToStack(d66, int32(stackArray67)+int32(0))
					ctx.FreeDesc(&d66)
					ctx.SyncDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray67)+int32(16))
					ctx.FreeDesc(&d53)
					d68 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d68
					callbackArgs70 := make([]JITValueDesc, 2)
					callbackArgs70[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray67) + 0}
					callbackArgs70[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray67) + 16}
					var d69 JITValueDesc
					callbackResultOff71 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff71))
					ctx.FreeDesc(&d68)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d10.Loc == LocLambdaTemplate && d10.Lambda != nil {
						stableCallbackArgs72 := ctx.StabilizeCallbackArgs(callbackArgs70)
						ctx.ReclaimUntrackedRegs()
						outerRegs73 := ctx.PreserveOuterRegs()
						d69 = JITEmitProcInlineWithOuter(ctx, &d10.Lambda.Proc, d10.Lambda.Outer, stableCallbackArgs72, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff71), ID: 0})
						ctx.RestoreOuterRegs(outerRegs73)
						ctx.ReclaimUntrackedRegs()
					} else {
						d74, knownBuiltin75 := jitEmitKnownDeclaration(ctx, d10, callbackArgs70, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff71), ID: 0})
						if knownBuiltin75 {
							d69 = d74
						} else {
							ctx.Coverage.DynamicCalls++
							d76 := jitCopyScmerToPair(ctx, d10)
							d69 = jitEmitDynamicCallableAt(ctx, d76, callbackArgs70, int32(stackArray67), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff71), ID: 0})
						}
					}
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d58)
					d58 = JITPrepareScmerGoArg(ctx, d58)
					ctx.EnsureDesc(&d69)
					ctx.EnsureDesc(&d69)
					d69 = JITPrepareScmerGoArg(ctx, d69)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d58)
					ctx.SyncDesc(&d69)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d58, d69})
					ctx.FreeDesc(&d58)
					ctx.FreeDesc(&d69)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d77 = d23
						if d77.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d77)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d77)
						} else {
							ctx.EmitStoreToStack(d77, int32(bbs[1].PhiBase)+int32(0))
						}
						if d23.Loc == LocReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps78 := PhiState{General: ps.General}
					ps78.OverlayValues = make([]JITValueDesc, 78)
					ps78.OverlayValues[3] = d3
					ps78.OverlayValues[4] = d4
					ps78.OverlayValues[5] = d5
					ps78.OverlayValues[6] = d6
					ps78.OverlayValues[7] = d7
					ps78.OverlayValues[9] = d9
					ps78.OverlayValues[10] = d10
					ps78.OverlayValues[12] = d12
					ps78.OverlayValues[13] = d13
					ps78.OverlayValues[14] = d14
					ps78.OverlayValues[15] = d15
					ps78.OverlayValues[16] = d16
					ps78.OverlayValues[17] = d17
					ps78.OverlayValues[18] = d18
					ps78.OverlayValues[19] = d19
					ps78.OverlayValues[21] = d21
					ps78.OverlayValues[22] = d22
					ps78.OverlayValues[23] = d23
					ps78.OverlayValues[24] = d24
					ps78.OverlayValues[25] = d25
					ps78.OverlayValues[28] = d28
					ps78.OverlayValues[53] = d53
					ps78.OverlayValues[54] = d54
					ps78.OverlayValues[55] = d55
					ps78.OverlayValues[57] = d57
					ps78.OverlayValues[58] = d58
					ps78.OverlayValues[63] = d63
					ps78.OverlayValues[65] = d65
					ps78.OverlayValues[66] = d66
					ps78.OverlayValues[68] = d68
					ps78.OverlayValues[69] = d69
					ps78.OverlayValues[74] = d74
					ps78.OverlayValues[76] = d76
					ps78.OverlayValues[77] = d77
					ps78.PhiValues = make([]JITValueDesc, 1)
					d79 = d23
					ps78.PhiValues[0] = d79
					if ps78.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps78)
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
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != LocNone {
						d65 = ps.OverlayValues[65]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
						d68 = ps.OverlayValues[68]
					}
					if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
						d69 = ps.OverlayValues[69]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					ctx.ReclaimUntrackedRegs()
					var d80 JITValueDesc
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d80 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r7}
						ctx.BindReg(d18.Reg, &d80)
						ctx.BindReg(r7, &d80)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d80)
						ctx.BindReg(r7, &d80)
						d18.Loc = LocNone
					}
					ctx.FreeDesc(&d18)
					ctx.SyncDesc(&d80)
					if d80.Loc == LocRegPair || d80.Loc == LocStackPair || d80.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d80, &result)
						result.Type = d80.Type
					} else {
						switch d80.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d80)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d80)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d80)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d80, &result)
							result.Type = d80.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps81 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps81)
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
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var stackArray50 int32
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var callbackResultOff54 int32
				var d57 JITValueDesc
				_ = d57
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
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
						ctx.EmitSetcc(r1, CondSignedLess)
						d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d11)
					}
					ctx.ReclaimUntrackedRegs()
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d12.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d11)
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
						ctx.EmitSetcc(r3, CondSignedLess)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d21)
					} else if d20.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d16.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d21)
					} else {
						r5 := ctx.AllocRegExcept(d20.Reg)
						ctx.EmitCmpInt64(d20.Reg, d16.Reg)
						ctx.EmitSetcc(r5, CondSignedLess)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d21)
					}
					ctx.FreeDesc(&d16)
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 26)
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[4] = d4
					ps26.OverlayValues[5] = d5
					ps26.OverlayValues[6] = d6
					ps26.OverlayValues[7] = d7
					ps26.OverlayValues[9] = d9
					ps26.OverlayValues[10] = d10
					ps26.OverlayValues[11] = d11
					ps26.OverlayValues[12] = d12
					ps26.OverlayValues[13] = d13
					ps26.OverlayValues[14] = d14
					ps26.OverlayValues[15] = d15
					ps26.OverlayValues[16] = d16
					ps26.OverlayValues[18] = d18
					ps26.OverlayValues[19] = d19
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[22] = d22
					ps26.OverlayValues[25] = d25
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 26)
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[6] = d6
					ps27.OverlayValues[7] = d7
					ps27.OverlayValues[9] = d9
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[11] = d11
					ps27.OverlayValues[12] = d12
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[14] = d14
					ps27.OverlayValues[15] = d15
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[18] = d18
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[25] = d25
					snap28 := d3
					snap29 := d4
					snap30 := d5
					snap31 := d6
					snap32 := d7
					snap33 := d9
					snap34 := d10
					snap35 := d11
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d18
					snap42 := d19
					snap43 := d20
					snap44 := d21
					snap45 := d22
					snap46 := d25
					alloc47 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps27)
					}
					ctx.RestoreAllocState(alloc47)
					d3 = snap28
					d4 = snap29
					d5 = snap30
					d6 = snap31
					d7 = snap32
					d9 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d18 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d25 = snap46
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps26)
					}
					return result
					ctx.FreeDesc(&d21)
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
					d49 = ctx.EmitSliceElementAddress(&d5, &d20, 16)
					ctx.EnsureDesc(&d49)
					r6 := ctx.AllocRegExcept(d49.Reg)
					ctx.EmitMovRegMem(r6, d49.Reg, 8)
					ctx.EmitMovRegMem(d49.Reg, d49.Reg, 0)
					d48 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d49.Reg, Reg2: r6}
					ctx.BindReg(d49.Reg, &d48)
					ctx.BindReg(r6, &d48)
					stackArray50 = ctx.AllocStack(int32(16))
					_ = stackArray50
					ctx.SyncDesc(&d48)
					ctx.EmitStoreScmerToStack(d48, int32(stackArray50)+int32(0))
					ctx.FreeDesc(&d48)
					d51 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d51
					callbackArgs53 := make([]JITValueDesc, 1)
					callbackArgs53[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray50) + 0}
					var d52 JITValueDesc
					callbackResultOff54 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff54))
					ctx.FreeDesc(&d51)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs55 := ctx.StabilizeCallbackArgs(callbackArgs53)
						ctx.ReclaimUntrackedRegs()
						outerRegs56 := ctx.PreserveOuterRegs()
						d52 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs55, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff54), ID: 0})
						ctx.RestoreOuterRegs(outerRegs56)
						ctx.ReclaimUntrackedRegs()
					} else {
						d57, knownBuiltin58 := jitEmitKnownDeclaration(ctx, d7, callbackArgs53, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff54), ID: 0})
						if knownBuiltin58 {
							d52 = d57
						} else {
							ctx.Coverage.DynamicCalls++
							d59 := jitCopyScmerToPair(ctx, d7)
							d52 = jitEmitDynamicCallableAt(ctx, d59, callbackArgs53, int32(stackArray50), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff54), ID: 0})
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d52)
					d52 = JITPrepareScmerGoArg(ctx, d52)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d52)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d15, d52})
					ctx.FreeDesc(&d52)
					if ps.General {
						ctx.SyncDesc(&d20)
						if d20.Loc == LocReg {
							ctx.ProtectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.ProtectReg(d20.Reg)
							ctx.ProtectReg(d20.Reg2)
						}
						d60 = d20
						if d60.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d60)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d60)
						} else {
							ctx.EmitStoreToStack(d60, int32(bbs[1].PhiBase)+int32(0))
						}
						if d20.Loc == LocReg {
							ctx.UnprotectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.UnprotectReg(d20.Reg)
							ctx.UnprotectReg(d20.Reg2)
						}
					}
					ps61 := PhiState{General: ps.General}
					ps61.OverlayValues = make([]JITValueDesc, 61)
					ps61.OverlayValues[3] = d3
					ps61.OverlayValues[4] = d4
					ps61.OverlayValues[5] = d5
					ps61.OverlayValues[6] = d6
					ps61.OverlayValues[7] = d7
					ps61.OverlayValues[9] = d9
					ps61.OverlayValues[10] = d10
					ps61.OverlayValues[11] = d11
					ps61.OverlayValues[12] = d12
					ps61.OverlayValues[13] = d13
					ps61.OverlayValues[14] = d14
					ps61.OverlayValues[15] = d15
					ps61.OverlayValues[16] = d16
					ps61.OverlayValues[18] = d18
					ps61.OverlayValues[19] = d19
					ps61.OverlayValues[20] = d20
					ps61.OverlayValues[21] = d21
					ps61.OverlayValues[22] = d22
					ps61.OverlayValues[25] = d25
					ps61.OverlayValues[48] = d48
					ps61.OverlayValues[49] = d49
					ps61.OverlayValues[51] = d51
					ps61.OverlayValues[52] = d52
					ps61.OverlayValues[57] = d57
					ps61.OverlayValues[59] = d59
					ps61.OverlayValues[60] = d60
					ps61.PhiValues = make([]JITValueDesc, 1)
					d62 = d20
					ps61.PhiValues[0] = d62
					if ps61.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps61)
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
						d57 = ps.OverlayValues[57]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					ctx.ReclaimUntrackedRegs()
					var d63 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d63 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d15.Reg, Reg2: r7}
						ctx.BindReg(d15.Reg, &d63)
						ctx.BindReg(r7, &d63)
						ctx.TransferReg(d15.Reg)
						ctx.BindReg(d15.Reg, &d63)
						ctx.BindReg(r7, &d63)
						d15.Loc = LocNone
					}
					ctx.FreeDesc(&d15)
					ctx.SyncDesc(&d63)
					if d63.Loc == LocRegPair || d63.Loc == LocStackPair || d63.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d63, &result)
						result.Type = d63.Type
					} else {
						switch d63.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d63)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d63)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d63)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d63, &result)
							result.Type = d63.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps64 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps64)
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
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var stackArray51 int32
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var callbackResultOff55 int32
				var d58 JITValueDesc
				_ = d58
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
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
						ctx.EmitSetcc(r1, CondSignedLess)
						d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
						ctx.BindReg(r1, &d11)
					}
					ctx.ReclaimUntrackedRegs()
					d12 = d11
					ctx.EnsureDesc(&d12)
					if d12.Loc != LocImm && d12.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d12.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d11)
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
						ctx.EmitSetcc(r3, CondSignedLess)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d21)
					} else if d20.Loc == LocImm {
						r4 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d16.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d21)
					} else {
						r5 := ctx.AllocRegExcept(d20.Reg)
						ctx.EmitCmpInt64(d20.Reg, d16.Reg)
						ctx.EmitSetcc(r5, CondSignedLess)
						d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
						ctx.BindReg(r5, &d21)
					}
					ctx.FreeDesc(&d16)
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d22.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps26 := PhiState{General: true}
					ps26.OverlayValues = make([]JITValueDesc, 26)
					ps26.OverlayValues[3] = d3
					ps26.OverlayValues[4] = d4
					ps26.OverlayValues[5] = d5
					ps26.OverlayValues[6] = d6
					ps26.OverlayValues[7] = d7
					ps26.OverlayValues[9] = d9
					ps26.OverlayValues[10] = d10
					ps26.OverlayValues[11] = d11
					ps26.OverlayValues[12] = d12
					ps26.OverlayValues[13] = d13
					ps26.OverlayValues[14] = d14
					ps26.OverlayValues[15] = d15
					ps26.OverlayValues[16] = d16
					ps26.OverlayValues[18] = d18
					ps26.OverlayValues[19] = d19
					ps26.OverlayValues[20] = d20
					ps26.OverlayValues[21] = d21
					ps26.OverlayValues[22] = d22
					ps26.OverlayValues[25] = d25
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 26)
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[6] = d6
					ps27.OverlayValues[7] = d7
					ps27.OverlayValues[9] = d9
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[11] = d11
					ps27.OverlayValues[12] = d12
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[14] = d14
					ps27.OverlayValues[15] = d15
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[18] = d18
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[25] = d25
					snap28 := d3
					snap29 := d4
					snap30 := d5
					snap31 := d6
					snap32 := d7
					snap33 := d9
					snap34 := d10
					snap35 := d11
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d15
					snap40 := d16
					snap41 := d18
					snap42 := d19
					snap43 := d20
					snap44 := d21
					snap45 := d22
					snap46 := d25
					alloc47 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps27)
					}
					ctx.RestoreAllocState(alloc47)
					d3 = snap28
					d4 = snap29
					d5 = snap30
					d6 = snap31
					d7 = snap32
					d9 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d15 = snap39
					d16 = snap40
					d18 = snap41
					d19 = snap42
					d20 = snap43
					d21 = snap44
					d22 = snap45
					d25 = snap46
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps26)
					}
					return result
					ctx.FreeDesc(&d21)
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
					d49 = ctx.EmitSliceElementAddress(&d5, &d20, 16)
					ctx.EnsureDesc(&d49)
					r6 := ctx.AllocRegExcept(d49.Reg)
					ctx.EmitMovRegMem(r6, d49.Reg, 8)
					ctx.EmitMovRegMem(d49.Reg, d49.Reg, 0)
					d48 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d49.Reg, Reg2: r6}
					ctx.BindReg(d49.Reg, &d48)
					ctx.BindReg(r6, &d48)
					d50 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray51 = ctx.AllocStack(int32(32))
					_ = stackArray51
					ctx.SyncDesc(&d50)
					ctx.EmitStoreScmerToStack(d50, int32(stackArray51)+int32(0))
					ctx.FreeDesc(&d50)
					ctx.SyncDesc(&d48)
					ctx.EmitStoreScmerToStack(d48, int32(stackArray51)+int32(16))
					ctx.FreeDesc(&d48)
					d52 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d52
					callbackArgs54 := make([]JITValueDesc, 2)
					callbackArgs54[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray51) + 0}
					callbackArgs54[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray51) + 16}
					var d53 JITValueDesc
					callbackResultOff55 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff55))
					ctx.FreeDesc(&d52)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d7.Loc == LocLambdaTemplate && d7.Lambda != nil {
						stableCallbackArgs56 := ctx.StabilizeCallbackArgs(callbackArgs54)
						ctx.ReclaimUntrackedRegs()
						outerRegs57 := ctx.PreserveOuterRegs()
						d53 = JITEmitProcInlineWithOuter(ctx, &d7.Lambda.Proc, d7.Lambda.Outer, stableCallbackArgs56, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff55), ID: 0})
						ctx.RestoreOuterRegs(outerRegs57)
						ctx.ReclaimUntrackedRegs()
					} else {
						d58, knownBuiltin59 := jitEmitKnownDeclaration(ctx, d7, callbackArgs54, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff55), ID: 0})
						if knownBuiltin59 {
							d53 = d58
						} else {
							ctx.Coverage.DynamicCalls++
							d60 := jitCopyScmerToPair(ctx, d7)
							d53 = jitEmitDynamicCallableAt(ctx, d60, callbackArgs54, int32(stackArray51), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff55), ID: 0})
						}
					}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d53)
					d53 = JITPrepareScmerGoArg(ctx, d53)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d53)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d15, d53})
					ctx.FreeDesc(&d53)
					if ps.General {
						ctx.SyncDesc(&d20)
						if d20.Loc == LocReg {
							ctx.ProtectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.ProtectReg(d20.Reg)
							ctx.ProtectReg(d20.Reg2)
						}
						d61 = d20
						if d61.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d61)
						if phiHomeOK2 {
							ctx.EmitMovToReg(r0, d61)
						} else {
							ctx.EmitStoreToStack(d61, int32(bbs[1].PhiBase)+int32(0))
						}
						if d20.Loc == LocReg {
							ctx.UnprotectReg(d20.Reg)
						} else if d20.Loc == LocRegPair {
							ctx.UnprotectReg(d20.Reg)
							ctx.UnprotectReg(d20.Reg2)
						}
					}
					ps62 := PhiState{General: ps.General}
					ps62.OverlayValues = make([]JITValueDesc, 62)
					ps62.OverlayValues[3] = d3
					ps62.OverlayValues[4] = d4
					ps62.OverlayValues[5] = d5
					ps62.OverlayValues[6] = d6
					ps62.OverlayValues[7] = d7
					ps62.OverlayValues[9] = d9
					ps62.OverlayValues[10] = d10
					ps62.OverlayValues[11] = d11
					ps62.OverlayValues[12] = d12
					ps62.OverlayValues[13] = d13
					ps62.OverlayValues[14] = d14
					ps62.OverlayValues[15] = d15
					ps62.OverlayValues[16] = d16
					ps62.OverlayValues[18] = d18
					ps62.OverlayValues[19] = d19
					ps62.OverlayValues[20] = d20
					ps62.OverlayValues[21] = d21
					ps62.OverlayValues[22] = d22
					ps62.OverlayValues[25] = d25
					ps62.OverlayValues[48] = d48
					ps62.OverlayValues[49] = d49
					ps62.OverlayValues[50] = d50
					ps62.OverlayValues[52] = d52
					ps62.OverlayValues[53] = d53
					ps62.OverlayValues[58] = d58
					ps62.OverlayValues[60] = d60
					ps62.OverlayValues[61] = d61
					ps62.PhiValues = make([]JITValueDesc, 1)
					d63 = d20
					ps62.PhiValues[0] = d63
					if ps62.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps62)
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
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
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
					ctx.ReclaimUntrackedRegs()
					var d64 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r7 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, makeAux(tagFastDict, 0))
						d64 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d15.Reg, Reg2: r7}
						ctx.BindReg(d15.Reg, &d64)
						ctx.BindReg(r7, &d64)
						ctx.TransferReg(d15.Reg)
						ctx.BindReg(d15.Reg, &d64)
						ctx.BindReg(r7, &d64)
						d15.Loc = LocNone
					}
					ctx.FreeDesc(&d15)
					ctx.SyncDesc(&d64)
					if d64.Loc == LocRegPair || d64.Loc == LocStackPair || d64.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d64, &result)
						result.Type = d64.Type
					} else {
						switch d64.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d64)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d64)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d64)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d64, &result)
							result.Type = d64.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				ps65 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps65)
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
