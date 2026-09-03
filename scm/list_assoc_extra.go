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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc"].Fn, args, result)
				}
				declaration := declarations["group_assoc"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var stackArray53 int32
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var callbackResultOff57 int32
				var d60 JITValueDesc
				_ = d60
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
						ctx.EmitSetcc(r0, CondSignedLess)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					d13 = d12
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d13.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d12)
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
						ctx.EmitSetcc(r2, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d17.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d17.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d22)
					}
					ctx.FreeDesc(&d17)
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 27)
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[7] = d7
					ps27.OverlayValues[8] = d8
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[11] = d11
					ps27.OverlayValues[12] = d12
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[14] = d14
					ps27.OverlayValues[15] = d15
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[17] = d17
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					ps27.OverlayValues[26] = d26
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 27)
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[7] = d7
					ps28.OverlayValues[8] = d8
					ps28.OverlayValues[10] = d10
					ps28.OverlayValues[11] = d11
					ps28.OverlayValues[12] = d12
					ps28.OverlayValues[13] = d13
					ps28.OverlayValues[14] = d14
					ps28.OverlayValues[15] = d15
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[26] = d26
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d7
					snap35 := d8
					snap36 := d10
					snap37 := d11
					snap38 := d12
					snap39 := d13
					snap40 := d14
					snap41 := d15
					snap42 := d16
					snap43 := d17
					snap44 := d19
					snap45 := d20
					snap46 := d21
					snap47 := d22
					snap48 := d23
					snap49 := d26
					alloc50 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps28)
					}
					ctx.RestoreAllocState(alloc50)
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
					d10 = snap36
					d11 = snap37
					d12 = snap38
					d13 = snap39
					d14 = snap40
					d15 = snap41
					d16 = snap42
					d17 = snap43
					d19 = snap44
					d20 = snap45
					d21 = snap46
					d22 = snap47
					d23 = snap48
					d26 = snap49
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps27)
					}
					return result
					ctx.FreeDesc(&d22)
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
					d52 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d52)
					r5 := ctx.AllocRegExcept(d52.Reg)
					ctx.EmitMovRegMem(r5, d52.Reg, 8)
					ctx.EmitMovRegMem(d52.Reg, d52.Reg, 0)
					d51 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d52.Reg, Reg2: r5}
					ctx.BindReg(d52.Reg, &d51)
					ctx.BindReg(r5, &d51)
					stackArray53 = ctx.AllocStack(int32(16))
					_ = stackArray53
					ctx.SyncDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray53)+int32(0))
					d54 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d54
					callbackArgs56 := make([]JITValueDesc, 1)
					callbackArgs56[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray53) + 0}
					var d55 JITValueDesc
					callbackResultOff57 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff57))
					ctx.FreeDesc(&d54)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs58 := ctx.StabilizeCallbackArgs(callbackArgs56)
						ctx.ReclaimUntrackedRegs()
						outerRegs59 := ctx.PreserveOuterRegs()
						d55 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs58, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0})
						ctx.RestoreOuterRegs(outerRegs59)
						ctx.ReclaimUntrackedRegs()
					} else {
						d60, knownBuiltin61 := jitEmitKnownDeclaration(ctx, d5, callbackArgs56, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0})
						if knownBuiltin61 {
							d55 = d60
						} else {
							ctx.Coverage.DynamicCalls++
							d62 := jitCopyScmerToPair(ctx, d5)
							callbackCallArgs := make([]JITValueDesc, 0, 2)
							callbackCallArgs = append(callbackCallArgs, d62)
							callbackCallArgs = append(callbackCallArgs, callbackArgs56...)
							d55 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d55, int32(callbackResultOff57))
							ctx.FreeDesc(&d55)
							d55 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0}
						}
					}
					d63 = args[3]
					d63.ID = 0
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d55)
					d55 = JITPrepareScmerGoArg(ctx, d55)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					d51 = JITPrepareScmerGoArg(ctx, d51)
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d63)
					d63 = JITPrepareScmerGoArg(ctx, d63)
					ctx.EnsureDesc(&d8)
					ctx.EnsureDesc(&d8)
					if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d55)
					ctx.SyncDesc(&d51)
					ctx.SyncDesc(&d63)
					ctx.SyncDesc(&d8)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).ReduceValue), []JITValueDesc{d16, d55, d51, d63, d8})
					ctx.FreeDesc(&d55)
					ctx.FreeDesc(&d51)
					ctx.FreeDesc(&d63)
					if ps.General {
						ctx.SyncDesc(&d21)
						if d21.Loc == LocReg {
							ctx.ProtectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.ProtectReg(d21.Reg)
							ctx.ProtectReg(d21.Reg2)
						}
						d64 = d21
						if d64.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d64)
						ctx.EmitStoreToStack(d64, int32(bbs[1].PhiBase)+int32(0))
						if d21.Loc == LocReg {
							ctx.UnprotectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.UnprotectReg(d21.Reg)
							ctx.UnprotectReg(d21.Reg2)
						}
					}
					ps65 := PhiState{General: ps.General}
					ps65.OverlayValues = make([]JITValueDesc, 65)
					ps65.OverlayValues[1] = d1
					ps65.OverlayValues[2] = d2
					ps65.OverlayValues[3] = d3
					ps65.OverlayValues[4] = d4
					ps65.OverlayValues[5] = d5
					ps65.OverlayValues[7] = d7
					ps65.OverlayValues[8] = d8
					ps65.OverlayValues[10] = d10
					ps65.OverlayValues[11] = d11
					ps65.OverlayValues[12] = d12
					ps65.OverlayValues[13] = d13
					ps65.OverlayValues[14] = d14
					ps65.OverlayValues[15] = d15
					ps65.OverlayValues[16] = d16
					ps65.OverlayValues[17] = d17
					ps65.OverlayValues[19] = d19
					ps65.OverlayValues[20] = d20
					ps65.OverlayValues[21] = d21
					ps65.OverlayValues[22] = d22
					ps65.OverlayValues[23] = d23
					ps65.OverlayValues[26] = d26
					ps65.OverlayValues[51] = d51
					ps65.OverlayValues[52] = d52
					ps65.OverlayValues[54] = d54
					ps65.OverlayValues[55] = d55
					ps65.OverlayValues[60] = d60
					ps65.OverlayValues[62] = d62
					ps65.OverlayValues[63] = d63
					ps65.OverlayValues[64] = d64
					ps65.PhiValues = make([]JITValueDesc, 1)
					d66 = d21
					ps65.PhiValues[0] = d66
					if ps65.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps65)
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
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
					}
					if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
						d63 = ps.OverlayValues[63]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
					}
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					ctx.ReclaimUntrackedRegs()
					var d67 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d67 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r6}
						ctx.BindReg(d16.Reg, &d67)
						ctx.BindReg(r6, &d67)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d67)
						ctx.BindReg(r6, &d67)
						d16.Loc = LocNone
					}
					ctx.FreeDesc(&d16)
					ctx.SyncDesc(&d67)
					if d67.Loc == LocRegPair || d67.Loc == LocStackPair || d67.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d67, &result)
						result.Type = d67.Type
					} else {
						switch d67.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d67)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d67)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d67)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d67, &result)
							result.Type = d67.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps68 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps68)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_append"].Fn, args, result)
				}
				declaration := declarations["group_assoc_append"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var stackArray53 int32
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var callbackResultOff57 int32
				var d60 JITValueDesc
				_ = d60
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var stackArray64 int32
				var d65 JITValueDesc
				_ = d65
				var d66 JITValueDesc
				_ = d66
				var callbackResultOff68 int32
				var d71 JITValueDesc
				_ = d71
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
						ctx.EmitSetcc(r0, CondSignedLess)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					d13 = d12
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d13.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d12)
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
						ctx.EmitSetcc(r2, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d17.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d17.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d22)
					}
					ctx.FreeDesc(&d17)
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 27)
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[7] = d7
					ps27.OverlayValues[8] = d8
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[11] = d11
					ps27.OverlayValues[12] = d12
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[14] = d14
					ps27.OverlayValues[15] = d15
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[17] = d17
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					ps27.OverlayValues[26] = d26
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 27)
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[7] = d7
					ps28.OverlayValues[8] = d8
					ps28.OverlayValues[10] = d10
					ps28.OverlayValues[11] = d11
					ps28.OverlayValues[12] = d12
					ps28.OverlayValues[13] = d13
					ps28.OverlayValues[14] = d14
					ps28.OverlayValues[15] = d15
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[26] = d26
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d7
					snap35 := d8
					snap36 := d10
					snap37 := d11
					snap38 := d12
					snap39 := d13
					snap40 := d14
					snap41 := d15
					snap42 := d16
					snap43 := d17
					snap44 := d19
					snap45 := d20
					snap46 := d21
					snap47 := d22
					snap48 := d23
					snap49 := d26
					alloc50 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps28)
					}
					ctx.RestoreAllocState(alloc50)
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
					d10 = snap36
					d11 = snap37
					d12 = snap38
					d13 = snap39
					d14 = snap40
					d15 = snap41
					d16 = snap42
					d17 = snap43
					d19 = snap44
					d20 = snap45
					d21 = snap46
					d22 = snap47
					d23 = snap48
					d26 = snap49
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps27)
					}
					return result
					ctx.FreeDesc(&d22)
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
					d52 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d52)
					r5 := ctx.AllocRegExcept(d52.Reg)
					ctx.EmitMovRegMem(r5, d52.Reg, 8)
					ctx.EmitMovRegMem(d52.Reg, d52.Reg, 0)
					d51 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d52.Reg, Reg2: r5}
					ctx.BindReg(d52.Reg, &d51)
					ctx.BindReg(r5, &d51)
					stackArray53 = ctx.AllocStack(int32(16))
					_ = stackArray53
					ctx.SyncDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray53)+int32(0))
					d54 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d54
					callbackArgs56 := make([]JITValueDesc, 1)
					callbackArgs56[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray53) + 0}
					var d55 JITValueDesc
					callbackResultOff57 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff57))
					ctx.FreeDesc(&d54)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs58 := ctx.StabilizeCallbackArgs(callbackArgs56)
						ctx.ReclaimUntrackedRegs()
						outerRegs59 := ctx.PreserveOuterRegs()
						d55 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs58, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0})
						ctx.RestoreOuterRegs(outerRegs59)
						ctx.ReclaimUntrackedRegs()
					} else {
						d60, knownBuiltin61 := jitEmitKnownDeclaration(ctx, d5, callbackArgs56, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0})
						if knownBuiltin61 {
							d55 = d60
						} else {
							ctx.Coverage.DynamicCalls++
							d62 := jitCopyScmerToPair(ctx, d5)
							callbackCallArgs := make([]JITValueDesc, 0, 2)
							callbackCallArgs = append(callbackCallArgs, d62)
							callbackCallArgs = append(callbackCallArgs, callbackArgs56...)
							d55 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d55, int32(callbackResultOff57))
							ctx.FreeDesc(&d55)
							d55 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0}
						}
					}
					d63 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray64 = ctx.AllocStack(int32(32))
					_ = stackArray64
					ctx.SyncDesc(&d63)
					ctx.EmitStoreScmerToStack(d63, int32(stackArray64)+int32(0))
					ctx.FreeDesc(&d63)
					ctx.SyncDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray64)+int32(16))
					ctx.FreeDesc(&d51)
					d65 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d65
					callbackArgs67 := make([]JITValueDesc, 2)
					callbackArgs67[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray64) + 0}
					callbackArgs67[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray64) + 16}
					var d66 JITValueDesc
					callbackResultOff68 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff68))
					ctx.FreeDesc(&d65)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs69 := ctx.StabilizeCallbackArgs(callbackArgs67)
						ctx.ReclaimUntrackedRegs()
						outerRegs70 := ctx.PreserveOuterRegs()
						d66 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs69, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff68), ID: 0})
						ctx.RestoreOuterRegs(outerRegs70)
						ctx.ReclaimUntrackedRegs()
					} else {
						d71, knownBuiltin72 := jitEmitKnownDeclaration(ctx, d8, callbackArgs67, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff68), ID: 0})
						if knownBuiltin72 {
							d66 = d71
						} else {
							ctx.Coverage.DynamicCalls++
							d73 := jitCopyScmerToPair(ctx, d8)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d73)
							callbackCallArgs = append(callbackCallArgs, callbackArgs67...)
							d66 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d66, int32(callbackResultOff68))
							ctx.FreeDesc(&d66)
							d66 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff68), ID: 0}
						}
					}
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d55)
					d55 = JITPrepareScmerGoArg(ctx, d55)
					ctx.EnsureDesc(&d66)
					ctx.EnsureDesc(&d66)
					d66 = JITPrepareScmerGoArg(ctx, d66)
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d55)
					ctx.SyncDesc(&d66)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d16, d55, d66})
					ctx.FreeDesc(&d55)
					ctx.FreeDesc(&d66)
					if ps.General {
						ctx.SyncDesc(&d21)
						if d21.Loc == LocReg {
							ctx.ProtectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.ProtectReg(d21.Reg)
							ctx.ProtectReg(d21.Reg2)
						}
						d74 = d21
						if d74.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d74)
						ctx.EmitStoreToStack(d74, int32(bbs[1].PhiBase)+int32(0))
						if d21.Loc == LocReg {
							ctx.UnprotectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.UnprotectReg(d21.Reg)
							ctx.UnprotectReg(d21.Reg2)
						}
					}
					ps75 := PhiState{General: ps.General}
					ps75.OverlayValues = make([]JITValueDesc, 75)
					ps75.OverlayValues[1] = d1
					ps75.OverlayValues[2] = d2
					ps75.OverlayValues[3] = d3
					ps75.OverlayValues[4] = d4
					ps75.OverlayValues[5] = d5
					ps75.OverlayValues[7] = d7
					ps75.OverlayValues[8] = d8
					ps75.OverlayValues[10] = d10
					ps75.OverlayValues[11] = d11
					ps75.OverlayValues[12] = d12
					ps75.OverlayValues[13] = d13
					ps75.OverlayValues[14] = d14
					ps75.OverlayValues[15] = d15
					ps75.OverlayValues[16] = d16
					ps75.OverlayValues[17] = d17
					ps75.OverlayValues[19] = d19
					ps75.OverlayValues[20] = d20
					ps75.OverlayValues[21] = d21
					ps75.OverlayValues[22] = d22
					ps75.OverlayValues[23] = d23
					ps75.OverlayValues[26] = d26
					ps75.OverlayValues[51] = d51
					ps75.OverlayValues[52] = d52
					ps75.OverlayValues[54] = d54
					ps75.OverlayValues[55] = d55
					ps75.OverlayValues[60] = d60
					ps75.OverlayValues[62] = d62
					ps75.OverlayValues[63] = d63
					ps75.OverlayValues[65] = d65
					ps75.OverlayValues[66] = d66
					ps75.OverlayValues[71] = d71
					ps75.OverlayValues[73] = d73
					ps75.OverlayValues[74] = d74
					ps75.PhiValues = make([]JITValueDesc, 1)
					d76 = d21
					ps75.PhiValues[0] = d76
					if ps75.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps75)
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
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
						d54 = ps.OverlayValues[54]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
						d62 = ps.OverlayValues[62]
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
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
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
					ctx.ReclaimUntrackedRegs()
					var d77 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d77 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r6}
						ctx.BindReg(d16.Reg, &d77)
						ctx.BindReg(r6, &d77)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d77)
						ctx.BindReg(r6, &d77)
						d16.Loc = LocNone
					}
					ctx.FreeDesc(&d16)
					ctx.SyncDesc(&d77)
					if d77.Loc == LocRegPair || d77.Loc == LocStackPair || d77.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d77, &result)
						result.Type = d77.Type
					} else {
						switch d77.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d77)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d77)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d77)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d77, &result)
							result.Type = d77.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps78 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps78)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_append_reduce"].Fn, args, result)
				}
				declaration := declarations["group_assoc_append_reduce"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d51 JITValueDesc
				_ = d51
				var d52 JITValueDesc
				_ = d52
				var d53 JITValueDesc
				_ = d53
				var stackArray54 int32
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var callbackResultOff58 int32
				var d61 JITValueDesc
				_ = d61
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var stackArray65 int32
				var d66 JITValueDesc
				_ = d66
				var d67 JITValueDesc
				_ = d67
				var callbackResultOff69 int32
				var d72 JITValueDesc
				_ = d72
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
						ctx.EmitSetcc(r0, CondSignedLess)
						d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d12)
					}
					ctx.ReclaimUntrackedRegs()
					d13 = d12
					ctx.EnsureDesc(&d13)
					if d13.Loc != LocImm && d13.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d13.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d12)
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
						ctx.EmitSetcc(r2, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d17.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d17.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d22 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d22)
					}
					ctx.FreeDesc(&d17)
					d23 = d22
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps27 := PhiState{General: true}
					ps27.OverlayValues = make([]JITValueDesc, 27)
					ps27.OverlayValues[1] = d1
					ps27.OverlayValues[2] = d2
					ps27.OverlayValues[3] = d3
					ps27.OverlayValues[4] = d4
					ps27.OverlayValues[5] = d5
					ps27.OverlayValues[7] = d7
					ps27.OverlayValues[8] = d8
					ps27.OverlayValues[10] = d10
					ps27.OverlayValues[11] = d11
					ps27.OverlayValues[12] = d12
					ps27.OverlayValues[13] = d13
					ps27.OverlayValues[14] = d14
					ps27.OverlayValues[15] = d15
					ps27.OverlayValues[16] = d16
					ps27.OverlayValues[17] = d17
					ps27.OverlayValues[19] = d19
					ps27.OverlayValues[20] = d20
					ps27.OverlayValues[21] = d21
					ps27.OverlayValues[22] = d22
					ps27.OverlayValues[23] = d23
					ps27.OverlayValues[26] = d26
					ps28 := PhiState{General: true}
					ps28.OverlayValues = make([]JITValueDesc, 27)
					ps28.OverlayValues[1] = d1
					ps28.OverlayValues[2] = d2
					ps28.OverlayValues[3] = d3
					ps28.OverlayValues[4] = d4
					ps28.OverlayValues[5] = d5
					ps28.OverlayValues[7] = d7
					ps28.OverlayValues[8] = d8
					ps28.OverlayValues[10] = d10
					ps28.OverlayValues[11] = d11
					ps28.OverlayValues[12] = d12
					ps28.OverlayValues[13] = d13
					ps28.OverlayValues[14] = d14
					ps28.OverlayValues[15] = d15
					ps28.OverlayValues[16] = d16
					ps28.OverlayValues[17] = d17
					ps28.OverlayValues[19] = d19
					ps28.OverlayValues[20] = d20
					ps28.OverlayValues[21] = d21
					ps28.OverlayValues[22] = d22
					ps28.OverlayValues[23] = d23
					ps28.OverlayValues[26] = d26
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d7
					snap35 := d8
					snap36 := d10
					snap37 := d11
					snap38 := d12
					snap39 := d13
					snap40 := d14
					snap41 := d15
					snap42 := d16
					snap43 := d17
					snap44 := d19
					snap45 := d20
					snap46 := d21
					snap47 := d22
					snap48 := d23
					snap49 := d26
					alloc50 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps28)
					}
					ctx.RestoreAllocState(alloc50)
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
					d10 = snap36
					d11 = snap37
					d12 = snap38
					d13 = snap39
					d14 = snap40
					d15 = snap41
					d16 = snap42
					d17 = snap43
					d19 = snap44
					d20 = snap45
					d21 = snap46
					d22 = snap47
					d23 = snap48
					d26 = snap49
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps27)
					}
					return result
					ctx.FreeDesc(&d22)
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
					d52 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d52)
					r5 := ctx.AllocRegExcept(d52.Reg)
					ctx.EmitMovRegMem(r5, d52.Reg, 8)
					ctx.EmitMovRegMem(d52.Reg, d52.Reg, 0)
					d51 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d52.Reg, Reg2: r5}
					ctx.BindReg(d52.Reg, &d51)
					ctx.BindReg(r5, &d51)
					d53 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray54 = ctx.AllocStack(int32(32))
					_ = stackArray54
					ctx.SyncDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray54)+int32(0))
					ctx.FreeDesc(&d53)
					ctx.SyncDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray54)+int32(16))
					d55 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d55
					callbackArgs57 := make([]JITValueDesc, 2)
					callbackArgs57[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray54) + 0}
					callbackArgs57[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray54) + 16}
					var d56 JITValueDesc
					callbackResultOff58 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff58))
					ctx.FreeDesc(&d55)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs59 := ctx.StabilizeCallbackArgs(callbackArgs57)
						ctx.ReclaimUntrackedRegs()
						outerRegs60 := ctx.PreserveOuterRegs()
						d56 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs59, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff58), ID: 0})
						ctx.RestoreOuterRegs(outerRegs60)
						ctx.ReclaimUntrackedRegs()
					} else {
						d61, knownBuiltin62 := jitEmitKnownDeclaration(ctx, d5, callbackArgs57, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff58), ID: 0})
						if knownBuiltin62 {
							d56 = d61
						} else {
							ctx.Coverage.DynamicCalls++
							d63 := jitCopyScmerToPair(ctx, d5)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d63)
							callbackCallArgs = append(callbackCallArgs, callbackArgs57...)
							d56 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d56, int32(callbackResultOff58))
							ctx.FreeDesc(&d56)
							d56 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff58), ID: 0}
						}
					}
					d64 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray65 = ctx.AllocStack(int32(32))
					_ = stackArray65
					ctx.SyncDesc(&d64)
					ctx.EmitStoreScmerToStack(d64, int32(stackArray65)+int32(0))
					ctx.FreeDesc(&d64)
					ctx.SyncDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray65)+int32(16))
					ctx.FreeDesc(&d51)
					d66 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d66
					callbackArgs68 := make([]JITValueDesc, 2)
					callbackArgs68[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray65) + 0}
					callbackArgs68[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray65) + 16}
					var d67 JITValueDesc
					callbackResultOff69 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff69))
					ctx.FreeDesc(&d66)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs70 := ctx.StabilizeCallbackArgs(callbackArgs68)
						ctx.ReclaimUntrackedRegs()
						outerRegs71 := ctx.PreserveOuterRegs()
						d67 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs70, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff69), ID: 0})
						ctx.RestoreOuterRegs(outerRegs71)
						ctx.ReclaimUntrackedRegs()
					} else {
						d72, knownBuiltin73 := jitEmitKnownDeclaration(ctx, d8, callbackArgs68, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff69), ID: 0})
						if knownBuiltin73 {
							d67 = d72
						} else {
							ctx.Coverage.DynamicCalls++
							d74 := jitCopyScmerToPair(ctx, d8)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d74)
							callbackCallArgs = append(callbackCallArgs, callbackArgs68...)
							d67 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d67, int32(callbackResultOff69))
							ctx.FreeDesc(&d67)
							d67 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff69), ID: 0}
						}
					}
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d56)
					d56 = JITPrepareScmerGoArg(ctx, d56)
					ctx.EnsureDesc(&d67)
					ctx.EnsureDesc(&d67)
					d67 = JITPrepareScmerGoArg(ctx, d67)
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d56)
					ctx.SyncDesc(&d67)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d16, d56, d67})
					ctx.FreeDesc(&d56)
					ctx.FreeDesc(&d67)
					if ps.General {
						ctx.SyncDesc(&d21)
						if d21.Loc == LocReg {
							ctx.ProtectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.ProtectReg(d21.Reg)
							ctx.ProtectReg(d21.Reg2)
						}
						d75 = d21
						if d75.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d75)
						ctx.EmitStoreToStack(d75, int32(bbs[1].PhiBase)+int32(0))
						if d21.Loc == LocReg {
							ctx.UnprotectReg(d21.Reg)
						} else if d21.Loc == LocRegPair {
							ctx.UnprotectReg(d21.Reg)
							ctx.UnprotectReg(d21.Reg2)
						}
					}
					ps76 := PhiState{General: ps.General}
					ps76.OverlayValues = make([]JITValueDesc, 76)
					ps76.OverlayValues[1] = d1
					ps76.OverlayValues[2] = d2
					ps76.OverlayValues[3] = d3
					ps76.OverlayValues[4] = d4
					ps76.OverlayValues[5] = d5
					ps76.OverlayValues[7] = d7
					ps76.OverlayValues[8] = d8
					ps76.OverlayValues[10] = d10
					ps76.OverlayValues[11] = d11
					ps76.OverlayValues[12] = d12
					ps76.OverlayValues[13] = d13
					ps76.OverlayValues[14] = d14
					ps76.OverlayValues[15] = d15
					ps76.OverlayValues[16] = d16
					ps76.OverlayValues[17] = d17
					ps76.OverlayValues[19] = d19
					ps76.OverlayValues[20] = d20
					ps76.OverlayValues[21] = d21
					ps76.OverlayValues[22] = d22
					ps76.OverlayValues[23] = d23
					ps76.OverlayValues[26] = d26
					ps76.OverlayValues[51] = d51
					ps76.OverlayValues[52] = d52
					ps76.OverlayValues[53] = d53
					ps76.OverlayValues[55] = d55
					ps76.OverlayValues[56] = d56
					ps76.OverlayValues[61] = d61
					ps76.OverlayValues[63] = d63
					ps76.OverlayValues[64] = d64
					ps76.OverlayValues[66] = d66
					ps76.OverlayValues[67] = d67
					ps76.OverlayValues[72] = d72
					ps76.OverlayValues[74] = d74
					ps76.OverlayValues[75] = d75
					ps76.PhiValues = make([]JITValueDesc, 1)
					d77 = d21
					ps76.PhiValues[0] = d77
					if ps76.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps76)
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
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != LocNone {
						d52 = ps.OverlayValues[52]
					}
					if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
						d53 = ps.OverlayValues[53]
					}
					if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != LocNone {
						d55 = ps.OverlayValues[55]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
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
					if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
						d66 = ps.OverlayValues[66]
					}
					if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
						d67 = ps.OverlayValues[67]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
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
					ctx.ReclaimUntrackedRegs()
					var d78 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d78 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r6}
						ctx.BindReg(d16.Reg, &d78)
						ctx.BindReg(r6, &d78)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d78)
						ctx.BindReg(r6, &d78)
						d16.Loc = LocNone
					}
					ctx.FreeDesc(&d16)
					ctx.SyncDesc(&d78)
					if d78.Loc == LocRegPair || d78.Loc == LocStackPair || d78.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d78, &result)
						result.Type = d78.Type
					} else {
						switch d78.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d78)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d78)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d78)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d78, &result)
							result.Type = d78.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps79 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps79)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_count"].Fn, args, result)
				}
				declaration := declarations["group_assoc_count"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var stackArray48 int32
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var callbackResultOff52 int32
				var d55 JITValueDesc
				_ = d55
				var d57 JITValueDesc
				_ = d57
				var d58 JITValueDesc
				_ = d58
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
						ctx.EmitSetcc(r0, CondSignedLess)
						d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d9)
					}
					ctx.ReclaimUntrackedRegs()
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d10.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d9)
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
						ctx.EmitSetcc(r2, CondSignedLess)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d19)
					} else if d18.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d18.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d14.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d19)
					} else {
						r4 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpInt64(d18.Reg, d14.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d19)
					}
					ctx.FreeDesc(&d14)
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 24)
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[2] = d2
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[7] = d7
					ps24.OverlayValues[8] = d8
					ps24.OverlayValues[9] = d9
					ps24.OverlayValues[10] = d10
					ps24.OverlayValues[11] = d11
					ps24.OverlayValues[12] = d12
					ps24.OverlayValues[13] = d13
					ps24.OverlayValues[14] = d14
					ps24.OverlayValues[16] = d16
					ps24.OverlayValues[17] = d17
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					ps24.OverlayValues[23] = d23
					ps25 := PhiState{General: true}
					ps25.OverlayValues = make([]JITValueDesc, 24)
					ps25.OverlayValues[1] = d1
					ps25.OverlayValues[2] = d2
					ps25.OverlayValues[3] = d3
					ps25.OverlayValues[4] = d4
					ps25.OverlayValues[5] = d5
					ps25.OverlayValues[7] = d7
					ps25.OverlayValues[8] = d8
					ps25.OverlayValues[9] = d9
					ps25.OverlayValues[10] = d10
					ps25.OverlayValues[11] = d11
					ps25.OverlayValues[12] = d12
					ps25.OverlayValues[13] = d13
					ps25.OverlayValues[14] = d14
					ps25.OverlayValues[16] = d16
					ps25.OverlayValues[17] = d17
					ps25.OverlayValues[18] = d18
					ps25.OverlayValues[19] = d19
					ps25.OverlayValues[20] = d20
					ps25.OverlayValues[23] = d23
					snap26 := d1
					snap27 := d2
					snap28 := d3
					snap29 := d4
					snap30 := d5
					snap31 := d7
					snap32 := d8
					snap33 := d9
					snap34 := d10
					snap35 := d11
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d16
					snap40 := d17
					snap41 := d18
					snap42 := d19
					snap43 := d20
					snap44 := d23
					alloc45 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps25)
					}
					ctx.RestoreAllocState(alloc45)
					d1 = snap26
					d2 = snap27
					d3 = snap28
					d4 = snap29
					d5 = snap30
					d7 = snap31
					d8 = snap32
					d9 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d16 = snap39
					d17 = snap40
					d18 = snap41
					d19 = snap42
					d20 = snap43
					d23 = snap44
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps24)
					}
					return result
					ctx.FreeDesc(&d19)
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
					d47 = ctx.EmitSliceElementAddress(&d3, &d18, 16)
					ctx.EnsureDesc(&d47)
					r5 := ctx.AllocRegExcept(d47.Reg)
					ctx.EmitMovRegMem(r5, d47.Reg, 8)
					ctx.EmitMovRegMem(d47.Reg, d47.Reg, 0)
					d46 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d47.Reg, Reg2: r5}
					ctx.BindReg(d47.Reg, &d46)
					ctx.BindReg(r5, &d46)
					stackArray48 = ctx.AllocStack(int32(16))
					_ = stackArray48
					ctx.SyncDesc(&d46)
					ctx.EmitStoreScmerToStack(d46, int32(stackArray48)+int32(0))
					ctx.FreeDesc(&d46)
					d49 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d49
					callbackArgs51 := make([]JITValueDesc, 1)
					callbackArgs51[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray48) + 0}
					var d50 JITValueDesc
					callbackResultOff52 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff52))
					ctx.FreeDesc(&d49)
					ctx.StabilizeDescAcrossNestedCall(&d18)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs53 := ctx.StabilizeCallbackArgs(callbackArgs51)
						ctx.ReclaimUntrackedRegs()
						outerRegs54 := ctx.PreserveOuterRegs()
						d50 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs53, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff52), ID: 0})
						ctx.RestoreOuterRegs(outerRegs54)
						ctx.ReclaimUntrackedRegs()
					} else {
						d55, knownBuiltin56 := jitEmitKnownDeclaration(ctx, d5, callbackArgs51, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff52), ID: 0})
						if knownBuiltin56 {
							d50 = d55
						} else {
							ctx.Coverage.DynamicCalls++
							d57 := jitCopyScmerToPair(ctx, d5)
							callbackCallArgs := make([]JITValueDesc, 0, 2)
							callbackCallArgs = append(callbackCallArgs, d57)
							callbackCallArgs = append(callbackCallArgs, callbackArgs51...)
							d50 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback1), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d50, int32(callbackResultOff52))
							ctx.FreeDesc(&d50)
							d50 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff52), ID: 0}
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d50)
					ctx.EnsureDesc(&d50)
					d50 = JITPrepareScmerGoArg(ctx, d50)
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d50)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d13, d50})
					ctx.FreeDesc(&d50)
					if ps.General {
						ctx.SyncDesc(&d18)
						if d18.Loc == LocReg {
							ctx.ProtectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.ProtectReg(d18.Reg)
							ctx.ProtectReg(d18.Reg2)
						}
						d58 = d18
						if d58.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d58)
						ctx.EmitStoreToStack(d58, int32(bbs[1].PhiBase)+int32(0))
						if d18.Loc == LocReg {
							ctx.UnprotectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.UnprotectReg(d18.Reg)
							ctx.UnprotectReg(d18.Reg2)
						}
					}
					ps59 := PhiState{General: ps.General}
					ps59.OverlayValues = make([]JITValueDesc, 59)
					ps59.OverlayValues[1] = d1
					ps59.OverlayValues[2] = d2
					ps59.OverlayValues[3] = d3
					ps59.OverlayValues[4] = d4
					ps59.OverlayValues[5] = d5
					ps59.OverlayValues[7] = d7
					ps59.OverlayValues[8] = d8
					ps59.OverlayValues[9] = d9
					ps59.OverlayValues[10] = d10
					ps59.OverlayValues[11] = d11
					ps59.OverlayValues[12] = d12
					ps59.OverlayValues[13] = d13
					ps59.OverlayValues[14] = d14
					ps59.OverlayValues[16] = d16
					ps59.OverlayValues[17] = d17
					ps59.OverlayValues[18] = d18
					ps59.OverlayValues[19] = d19
					ps59.OverlayValues[20] = d20
					ps59.OverlayValues[23] = d23
					ps59.OverlayValues[46] = d46
					ps59.OverlayValues[47] = d47
					ps59.OverlayValues[49] = d49
					ps59.OverlayValues[50] = d50
					ps59.OverlayValues[55] = d55
					ps59.OverlayValues[57] = d57
					ps59.OverlayValues[58] = d58
					ps59.PhiValues = make([]JITValueDesc, 1)
					d60 = d18
					ps59.PhiValues[0] = d60
					if ps59.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps59)
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
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
						d49 = ps.OverlayValues[49]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
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
					if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
						d60 = ps.OverlayValues[60]
					}
					ctx.ReclaimUntrackedRegs()
					var d61 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d61 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d13.Reg, Reg2: r6}
						ctx.BindReg(d13.Reg, &d61)
						ctx.BindReg(r6, &d61)
						ctx.TransferReg(d13.Reg)
						ctx.BindReg(d13.Reg, &d61)
						ctx.BindReg(r6, &d61)
						d13.Loc = LocNone
					}
					ctx.FreeDesc(&d13)
					ctx.SyncDesc(&d61)
					if d61.Loc == LocRegPair || d61.Loc == LocStackPair || d61.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d61, &result)
						result.Type = d61.Type
					} else {
						switch d61.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d61)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d61)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d61)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d61, &result)
							result.Type = d61.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps62 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps62)
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
				if !jitEnabled {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_count_reduce"].Fn, args, result)
				}
				declaration := declarations["group_assoc_count_reduce"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
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
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var stackArray49 int32
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				var callbackResultOff53 int32
				var d56 JITValueDesc
				_ = d56
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
				bbs[1].PhiCount = uint16(1)
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
						ctx.EmitSetcc(r0, CondSignedLess)
						d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d9)
					}
					ctx.ReclaimUntrackedRegs()
					d10 = d9
					ctx.EnsureDesc(&d10)
					if d10.Loc != LocImm && d10.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
						ctx.EmitCmpRegImm32(d10.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					ctx.FreeDesc(&d9)
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
						ctx.EmitSetcc(r2, CondSignedLess)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d19)
					} else if d18.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d18.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d14.Reg)
						ctx.EmitSetcc(r3, CondSignedLess)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
						ctx.BindReg(r3, &d19)
					} else {
						r4 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpInt64(d18.Reg, d14.Reg)
						ctx.EmitSetcc(r4, CondSignedLess)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
						ctx.BindReg(r4, &d19)
					}
					ctx.FreeDesc(&d14)
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
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
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl11)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl12)
					ctx.EmitJmp(lbl4)
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 24)
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[2] = d2
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[7] = d7
					ps24.OverlayValues[8] = d8
					ps24.OverlayValues[9] = d9
					ps24.OverlayValues[10] = d10
					ps24.OverlayValues[11] = d11
					ps24.OverlayValues[12] = d12
					ps24.OverlayValues[13] = d13
					ps24.OverlayValues[14] = d14
					ps24.OverlayValues[16] = d16
					ps24.OverlayValues[17] = d17
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					ps24.OverlayValues[23] = d23
					ps25 := PhiState{General: true}
					ps25.OverlayValues = make([]JITValueDesc, 24)
					ps25.OverlayValues[1] = d1
					ps25.OverlayValues[2] = d2
					ps25.OverlayValues[3] = d3
					ps25.OverlayValues[4] = d4
					ps25.OverlayValues[5] = d5
					ps25.OverlayValues[7] = d7
					ps25.OverlayValues[8] = d8
					ps25.OverlayValues[9] = d9
					ps25.OverlayValues[10] = d10
					ps25.OverlayValues[11] = d11
					ps25.OverlayValues[12] = d12
					ps25.OverlayValues[13] = d13
					ps25.OverlayValues[14] = d14
					ps25.OverlayValues[16] = d16
					ps25.OverlayValues[17] = d17
					ps25.OverlayValues[18] = d18
					ps25.OverlayValues[19] = d19
					ps25.OverlayValues[20] = d20
					ps25.OverlayValues[23] = d23
					snap26 := d1
					snap27 := d2
					snap28 := d3
					snap29 := d4
					snap30 := d5
					snap31 := d7
					snap32 := d8
					snap33 := d9
					snap34 := d10
					snap35 := d11
					snap36 := d12
					snap37 := d13
					snap38 := d14
					snap39 := d16
					snap40 := d17
					snap41 := d18
					snap42 := d19
					snap43 := d20
					snap44 := d23
					alloc45 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps25)
					}
					ctx.RestoreAllocState(alloc45)
					d1 = snap26
					d2 = snap27
					d3 = snap28
					d4 = snap29
					d5 = snap30
					d7 = snap31
					d8 = snap32
					d9 = snap33
					d10 = snap34
					d11 = snap35
					d12 = snap36
					d13 = snap37
					d14 = snap38
					d16 = snap39
					d17 = snap40
					d18 = snap41
					d19 = snap42
					d20 = snap43
					d23 = snap44
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps24)
					}
					return result
					ctx.FreeDesc(&d19)
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
					d47 = ctx.EmitSliceElementAddress(&d3, &d18, 16)
					ctx.EnsureDesc(&d47)
					r5 := ctx.AllocRegExcept(d47.Reg)
					ctx.EmitMovRegMem(r5, d47.Reg, 8)
					ctx.EmitMovRegMem(d47.Reg, d47.Reg, 0)
					d46 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d47.Reg, Reg2: r5}
					ctx.BindReg(d47.Reg, &d46)
					ctx.BindReg(r5, &d46)
					d48 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					stackArray49 = ctx.AllocStack(int32(32))
					_ = stackArray49
					ctx.SyncDesc(&d48)
					ctx.EmitStoreScmerToStack(d48, int32(stackArray49)+int32(0))
					ctx.FreeDesc(&d48)
					ctx.SyncDesc(&d46)
					ctx.EmitStoreScmerToStack(d46, int32(stackArray49)+int32(16))
					ctx.FreeDesc(&d46)
					d50 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d50
					callbackArgs52 := make([]JITValueDesc, 2)
					callbackArgs52[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray49) + 0}
					callbackArgs52[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray49) + 16}
					var d51 JITValueDesc
					callbackResultOff53 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff53))
					ctx.FreeDesc(&d50)
					ctx.StabilizeDescAcrossNestedCall(&d18)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs54 := ctx.StabilizeCallbackArgs(callbackArgs52)
						ctx.ReclaimUntrackedRegs()
						outerRegs55 := ctx.PreserveOuterRegs()
						d51 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs54, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff53), ID: 0})
						ctx.RestoreOuterRegs(outerRegs55)
						ctx.ReclaimUntrackedRegs()
					} else {
						d56, knownBuiltin57 := jitEmitKnownDeclaration(ctx, d5, callbackArgs52, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff53), ID: 0})
						if knownBuiltin57 {
							d51 = d56
						} else {
							ctx.Coverage.DynamicCalls++
							d58 := jitCopyScmerToPair(ctx, d5)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d58)
							callbackCallArgs = append(callbackCallArgs, callbackArgs52...)
							d51 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d51, int32(callbackResultOff53))
							ctx.FreeDesc(&d51)
							d51 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff53), ID: 0}
						}
					}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					d51 = JITPrepareScmerGoArg(ctx, d51)
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d51)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d13, d51})
					ctx.FreeDesc(&d51)
					if ps.General {
						ctx.SyncDesc(&d18)
						if d18.Loc == LocReg {
							ctx.ProtectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.ProtectReg(d18.Reg)
							ctx.ProtectReg(d18.Reg2)
						}
						d59 = d18
						if d59.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d59)
						ctx.EmitStoreToStack(d59, int32(bbs[1].PhiBase)+int32(0))
						if d18.Loc == LocReg {
							ctx.UnprotectReg(d18.Reg)
						} else if d18.Loc == LocRegPair {
							ctx.UnprotectReg(d18.Reg)
							ctx.UnprotectReg(d18.Reg2)
						}
					}
					ps60 := PhiState{General: ps.General}
					ps60.OverlayValues = make([]JITValueDesc, 60)
					ps60.OverlayValues[1] = d1
					ps60.OverlayValues[2] = d2
					ps60.OverlayValues[3] = d3
					ps60.OverlayValues[4] = d4
					ps60.OverlayValues[5] = d5
					ps60.OverlayValues[7] = d7
					ps60.OverlayValues[8] = d8
					ps60.OverlayValues[9] = d9
					ps60.OverlayValues[10] = d10
					ps60.OverlayValues[11] = d11
					ps60.OverlayValues[12] = d12
					ps60.OverlayValues[13] = d13
					ps60.OverlayValues[14] = d14
					ps60.OverlayValues[16] = d16
					ps60.OverlayValues[17] = d17
					ps60.OverlayValues[18] = d18
					ps60.OverlayValues[19] = d19
					ps60.OverlayValues[20] = d20
					ps60.OverlayValues[23] = d23
					ps60.OverlayValues[46] = d46
					ps60.OverlayValues[47] = d47
					ps60.OverlayValues[48] = d48
					ps60.OverlayValues[50] = d50
					ps60.OverlayValues[51] = d51
					ps60.OverlayValues[56] = d56
					ps60.OverlayValues[58] = d58
					ps60.OverlayValues[59] = d59
					ps60.PhiValues = make([]JITValueDesc, 1)
					d61 = d18
					ps60.PhiValues[0] = d61
					if ps60.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps60)
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
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
						d50 = ps.OverlayValues[50]
					}
					if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
						d51 = ps.OverlayValues[51]
					}
					if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != LocNone {
						d56 = ps.OverlayValues[56]
					}
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
						d61 = ps.OverlayValues[61]
					}
					ctx.ReclaimUntrackedRegs()
					var d62 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d62 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d13.Reg, Reg2: r6}
						ctx.BindReg(d13.Reg, &d62)
						ctx.BindReg(r6, &d62)
						ctx.TransferReg(d13.Reg)
						ctx.BindReg(d13.Reg, &d62)
						ctx.BindReg(r6, &d62)
						d13.Loc = LocNone
					}
					ctx.FreeDesc(&d13)
					ctx.SyncDesc(&d62)
					if d62.Loc == LocRegPair || d62.Loc == LocStackPair || d62.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d62, &result)
						result.Type = d62.Type
					} else {
						switch d62.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d62)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d62)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d62)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d62, &result)
							result.Type = d62.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps63 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps63)
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
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["mapkey_assoc"].Fn, args, result)
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
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				ctx.Coverage.NativeCalls++
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["mapkey_assoc_mut"].Fn, args, result)
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: false,
			JITInlineCost:      65535,
		},
	})
}
