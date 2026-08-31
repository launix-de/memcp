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
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc"].Fn, args, result)
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
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
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
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var stackArray92 int32
				var d93 JITValueDesc
				_ = d93
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
				var d112 JITValueDesc
				_ = d112
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
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var stackArray123 int32
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var callbackResultOff127 int32
				var d130 JITValueDesc
				_ = d130
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d135 JITValueDesc
				_ = d135
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
						d5 = ctx.RequestOptimizedCallback(1)
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
						d8 = ctx.RequestOptimizedCallback(2)
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
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
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
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitJmp(lbl6)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitCmpRegImm32(d13.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl6)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d12)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
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
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d21)
					ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
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
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl11)
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
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray53)+int32(0))
					d54 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d54
					callbackArgs56 := make([]JITValueDesc, 1)
					callbackArgs56[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray53) + 0}
					var d55 JITValueDesc
					callbackResultOff57 = ctx.AllocStack(16)
					ctx.FreeDesc(&d54)
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
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d8)
					d64 = d55
					_ = d64
					ctx.StabilizeDescForControlFlow(&d64)
					d65 = d51
					_ = d65
					ctx.StabilizeDescForControlFlow(&d65)
					d66 = d63
					_ = d66
					ctx.StabilizeDescForControlFlow(&d66)
					d67 = d8
					_ = d67
					ctx.StabilizeDescForControlFlow(&d67)
					r6 := d16.Loc == LocReg || d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r7 := d16.Reg
					if r6 {
						ctx.ProtectReg(r7)
					}
					r8 := d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r9 := d16.Reg2
					if r8 {
						ctx.ProtectReg(r9)
					}
					r10 := d16.Loc == LocRegTriple
					r11 := d16.Reg3
					if r10 {
						ctx.ProtectReg(r11)
					}
					lbl12 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					bbpos_2_4 := int32(-1)
					_ = bbpos_2_4
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d68 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r12 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r12, fieldAddr)
						d68 = JITValueDesc{Loc: LocReg, Reg: r12}
						ctx.BindReg(r12, &d68)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r13 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r13, baseReg, off)
						d68 = JITValueDesc{Loc: LocReg, Reg: r13}
						ctx.BindReg(r13, &d68)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d68)
					var d69 JITValueDesc
					if d68.Loc == LocImm {
						d69 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d68.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d68)
						if d68.Loc != LocReg && d68.Loc != LocRegPair && d68.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r14 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d68.Reg, 0)
						ctx.EmitSetcc(r14, CondEqual)
						d69 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d69)
					}
					ctx.FreeDesc(&d68)
					ctx.ReclaimUntrackedRegs()
					d70 = d69
					ctx.EnsureDesc(&d70)
					if d70.Loc != LocImm && d70.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d70.Loc == LocImm {
						if d70.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						}
					} else {
						ctx.EmitCmpRegImm32(d70.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d69)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					if d64.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d64.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d64.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d64)
						} else if d64.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d64)
						} else if d64.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d64)
						} else if d64.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d64.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d64 = tmpPair
					} else if d64.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d64.Type, Reg: ctx.AllocRegExcept(d64.Reg), Reg2: ctx.AllocRegExcept(d64.Reg)}
						switch d64.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d64)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d64)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d64)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d64)
						d64 = tmpPair
					}
					if d64.Loc != LocRegPair && d64.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (HashKey arg0)")
					}
					ctx.SyncDesc(&d64)
					d71 = ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d64}, 1)
					ctx.BindReg(d71.Reg, &d71)
					ctx.StabilizeDescForControlFlow(&d71)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					if d64.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d64.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d64.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d64)
						} else if d64.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d64)
						} else if d64.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d64)
						} else if d64.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d64.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d64 = tmpPair
					} else if d64.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d64.Type, Reg: ctx.AllocRegExcept(d64.Reg), Reg2: ctx.AllocRegExcept(d64.Reg)}
						switch d64.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d64)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d64)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d64)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d64)
						d64 = tmpPair
					}
					if d64.Loc != LocRegPair && d64.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*FastDict).findPos arg1)")
					}
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d71)
					if d71.Loc == LocRegPair || d71.Loc == LocStackPair || d71.Loc == LocRegTriple || d71.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d64)
					ctx.SyncDesc(&d71)
					callResults72 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d16, d64, d71}, []uint8{1, 1}, []uint8{0, 0})
					d73 = callResults72[0]
					_ = d73
					d74 = callResults72[1]
					_ = d74
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d73)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d75 = d74
					ctx.EnsureDesc(&d75)
					if d75.Loc != LocImm && d75.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d75.Loc == LocImm {
						if d75.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitCmpRegImm32(d75.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d74)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					stackArray76 = ctx.AllocStack(int32(32))
					_ = stackArray76
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d66)
					ctx.EnsureDesc(&d66)
					ctx.EmitStoreScmerToStack(d66, int32(stackArray76)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EmitStoreScmerToStack(d65, int32(stackArray76)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d77 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d77
					ctx.ReclaimUntrackedRegs()
					callbackArgs79 := make([]JITValueDesc, 2)
					callbackArgs79[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray76) + 0}
					callbackArgs79[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray76) + 16}
					var d78 JITValueDesc
					callbackResultOff80 = ctx.AllocStack(16)
					ctx.FreeDesc(&d77)
					if d67.Loc == LocLambdaTemplate && d67.Lambda != nil {
						stableCallbackArgs81 := ctx.StabilizeCallbackArgs(callbackArgs79)
						ctx.ReclaimUntrackedRegs()
						outerRegs82 := ctx.PreserveOuterRegs()
						d78 = JITEmitProcInlineWithOuter(ctx, &d67.Lambda.Proc, d67.Lambda.Outer, stableCallbackArgs81, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff80), ID: 0})
						ctx.RestoreOuterRegs(outerRegs82)
						ctx.ReclaimUntrackedRegs()
					} else {
						d83, knownBuiltin84 := jitEmitKnownDeclaration(ctx, d67, callbackArgs79, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff80), ID: 0})
						if knownBuiltin84 {
							d78 = d83
						} else {
							d85 := jitCopyScmerToPair(ctx, d67)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d85)
							callbackCallArgs = append(callbackCallArgs, callbackArgs79...)
							d78 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d78, int32(callbackResultOff80))
							ctx.FreeDesc(&d78)
							d78 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff80), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d71)
					d86 = d64
					_ = d86
					ctx.StabilizeDescForControlFlow(&d86)
					d87 = d78
					_ = d87
					ctx.StabilizeDescForControlFlow(&d87)
					d88 = d71
					_ = d88
					ctx.StabilizeDescForControlFlow(&d88)
					r15 := d16.Loc == LocReg || d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r16 := d16.Reg
					if r15 {
						ctx.ProtectReg(r16)
					}
					r17 := d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r18 := d16.Reg2
					if r17 {
						ctx.ProtectReg(r18)
					}
					r19 := d16.Loc == LocRegTriple
					r20 := d16.Reg3
					if r19 {
						ctx.ProtectReg(r20)
					}
					r21 := d64.Loc == LocReg || d64.Loc == LocRegPair || d64.Loc == LocRegTriple
					r22 := d64.Reg
					if r21 {
						ctx.ProtectReg(r22)
					}
					r23 := d64.Loc == LocRegPair || d64.Loc == LocRegTriple
					r24 := d64.Reg2
					if r23 {
						ctx.ProtectReg(r24)
					}
					r25 := d64.Loc == LocRegTriple
					r26 := d64.Reg3
					if r25 {
						ctx.ProtectReg(r26)
					}
					lbl21 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					bbpos_3_3 := int32(-1)
					_ = bbpos_3_3
					bbpos_3_4 := int32(-1)
					_ = bbpos_3_4
					bbpos_3_5 := int32(-1)
					_ = bbpos_3_5
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d89 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r27 := ctx.AllocReg()
						r28 := ctx.AllocRegExcept(r27)
						r29 := ctx.AllocRegExcept(r27, r28)
						ctx.EmitMovRegMem64(r27, fieldAddr)
						ctx.EmitMovRegMem64(r28, fieldAddr+8)
						ctx.EmitMovRegMem64(r29, fieldAddr+16)
						d89 = JITValueDesc{Loc: LocRegTriple, Reg: r27, Reg2: r28, Reg3: r29}
						ctx.BindReg(r27, &d89)
						ctx.BindReg(r28, &d89)
						ctx.BindReg(r29, &d89)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r30 := ctx.AllocRegExcept(baseReg)
						r31 := ctx.AllocRegExcept(baseReg, r30)
						r32 := ctx.AllocRegExcept(baseReg, r30, r31)
						ctx.EmitMovRegMem(r30, baseReg, off)
						ctx.EmitMovRegMem(r31, baseReg, off+8)
						ctx.EmitMovRegMem(r32, baseReg, off+16)
						d89 = JITValueDesc{Loc: LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
						ctx.BindReg(r30, &d89)
						ctx.BindReg(r31, &d89)
						ctx.BindReg(r32, &d89)
					}
					ctx.ReclaimUntrackedRegs()
					var d90 JITValueDesc
					if d89.SliceSizeKnown {
						d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d89.KnownSliceLen))}
					} else if d89.Loc == LocImm {
						d90 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d89.StackOff))}
					} else if d89.Loc == LocStackTriple {
						d90 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d89.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d89)
						if d89.Loc == LocRegPair || d89.Loc == LocRegTriple {
							d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d89.Reg2, ID: 0}
						} else if d89.Loc == LocReg {
							d90 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d89.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d90)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d91 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r33 := ctx.AllocReg()
						r34 := ctx.AllocRegExcept(r33)
						r35 := ctx.AllocRegExcept(r33, r34)
						ctx.EmitMovRegMem64(r33, fieldAddr)
						ctx.EmitMovRegMem64(r34, fieldAddr+8)
						ctx.EmitMovRegMem64(r35, fieldAddr+16)
						d91 = JITValueDesc{Loc: LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
						ctx.BindReg(r33, &d91)
						ctx.BindReg(r34, &d91)
						ctx.BindReg(r35, &d91)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r36 := ctx.AllocRegExcept(baseReg)
						r37 := ctx.AllocRegExcept(baseReg, r36)
						r38 := ctx.AllocRegExcept(baseReg, r36, r37)
						ctx.EmitMovRegMem(r36, baseReg, off)
						ctx.EmitMovRegMem(r37, baseReg, off+8)
						ctx.EmitMovRegMem(r38, baseReg, off+16)
						d91 = JITValueDesc{Loc: LocRegTriple, Reg: r36, Reg2: r37, Reg3: r38}
						ctx.BindReg(r36, &d91)
						ctx.BindReg(r37, &d91)
						ctx.BindReg(r38, &d91)
					}
					ctx.ReclaimUntrackedRegs()
					stackArray92 = ctx.AllocStack(int32(32))
					_ = stackArray92
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d86)
					ctx.EnsureDesc(&d86)
					ctx.EmitStoreScmerToStack(d86, int32(stackArray92)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d87)
					ctx.EmitStoreScmerToStack(d87, int32(stackArray92)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d93 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d93
					ctx.ReclaimUntrackedRegs()
					r39 := ctx.AllocReg()
					r40 := ctx.AllocRegExcept(r39)
					r41 := ctx.AllocRegExcept(r39, r40)
					d94 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r39, Reg2: r40, Reg3: r41}
					ctx.BindReg(r39, &d94)
					ctx.BindReg(r40, &d94)
					ctx.BindReg(r41, &d94)
					ctx.BindReg(r39, &d94)
					ctx.BindReg(r40, &d94)
					ctx.BindReg(r41, &d94)
					ctx.EmitLeaRegMem(d94.Reg, ctx.StackReg, int32(stackArray92))
					ctx.EmitMovRegImm64(d94.Reg2, uint64(2))
					ctx.EmitMovRegImm64(d94.Reg3, uint64(2))
					callResults95 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d91, d94}, []uint8{3}, []uint8{1})
					d96 = callResults95[0]
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d96)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d96)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value []Scmer) { base.Pairs = value }), []JITValueDesc{d16, d96})
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d97 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r42 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r42, fieldAddr)
						d97 = JITValueDesc{Loc: LocReg, Reg: r42}
						ctx.BindReg(r42, &d97)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r43 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r43, baseReg, off)
						d97 = JITValueDesc{Loc: LocReg, Reg: r43}
						ctx.BindReg(r43, &d97)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d88)
					lookupResults98 := JITEmitGoCallResults(ctx, GoFuncAddr(func(m map[uint64]int, k uint64) (int, bool) { value, ok := m[k]; return value, ok }), []JITValueDesc{d97, d88}, []uint8{1, 1}, []uint8{0, 0})
					d99 = lookupResults98[0]
					d100 = lookupResults98[1]
					ctx.EmitAndRegImm32(d100.Reg, 1)
					d100.Type = tagBool
					ctx.FreeDesc(&d97)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d101 = d100
					ctx.EnsureDesc(&d101)
					if d101.Loc != LocImm && d101.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d101.Loc == LocImm {
						if d101.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl22)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitCmpRegImm32(d101.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
					ctx.FreeDesc(&d100)
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d102 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r44 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r44, fieldAddr)
						d102 = JITValueDesc{Loc: LocReg, Reg: r44}
						ctx.BindReg(r44, &d102)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r45 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r45, baseReg, off)
						d102 = JITValueDesc{Loc: LocReg, Reg: r45}
						ctx.BindReg(r45, &d102)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d102)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d90)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64]int, key uint64, value int) { m[key] = value }), []JITValueDesc{d102, d88, d90})
					ctx.FreeDesc(&d102)
					ctx.ReclaimUntrackedRegs()
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl21)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d103 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r46 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r46, fieldAddr)
						d103 = JITValueDesc{Loc: LocReg, Reg: r46}
						ctx.BindReg(r46, &d103)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r47 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r47, baseReg, off)
						d103 = JITValueDesc{Loc: LocReg, Reg: r47}
						ctx.BindReg(r47, &d103)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d103)
					var d104 JITValueDesc
					if d103.Loc == LocImm {
						d104 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d103.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d103)
						if d103.Loc != LocReg && d103.Loc != LocRegPair && d103.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r48 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d103.Reg, 0)
						ctx.EmitSetcc(r48, CondEqual)
						d104 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r48}
						ctx.BindReg(r48, &d104)
					}
					ctx.FreeDesc(&d103)
					ctx.ReclaimUntrackedRegs()
					d105 = d104
					ctx.EnsureDesc(&d105)
					if d105.Loc != LocImm && d105.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					if d105.Loc == LocImm {
						if d105.Imm.Bool() {
							ctx.MarkLabel(lbl28)
							ctx.EmitJmp(lbl26)
						} else {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl27)
						}
					} else {
						ctx.EmitCmpRegImm32(d105.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl28)
						ctx.EmitJmp(lbl29)
						ctx.MarkLabel(lbl28)
						ctx.EmitJmp(lbl26)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
					}
					ctx.FreeDesc(&d104)
					bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d106 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r49 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r49, fieldAddr)
						d106 = JITValueDesc{Loc: LocReg, Reg: r49}
						ctx.BindReg(r49, &d106)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r50 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r50, baseReg, off)
						d106 = JITValueDesc{Loc: LocReg, Reg: r50}
						ctx.BindReg(r50, &d106)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d107 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r51 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r51, fieldAddr)
						d107 = JITValueDesc{Loc: LocReg, Reg: r51}
						ctx.BindReg(r51, &d107)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r52 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r52, baseReg, off)
						d107 = JITValueDesc{Loc: LocReg, Reg: r52}
						ctx.BindReg(r52, &d107)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d107)
					ctx.EnsureDesc(&d88)
					d108 = ctx.EmitGoCallScalar(GoFuncAddr(func(m map[uint64][]int, k uint64) []int { return m[k] }), []JITValueDesc{d107, d88}, 3)
					ctx.FreeDesc(&d107)
					ctx.ReclaimUntrackedRegs()
					d109 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]int { return new([1]int) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d90)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]int, index int, value int) { dst[index] = value }), []JITValueDesc{d109, d110, d90})
					ctx.FreeDesc(&d90)
					ctx.ReclaimUntrackedRegs()
					sliceResults111 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]int) []int { return value[0:1:1] }), []JITValueDesc{d109}, []uint8{3}, []uint8{1})
					d112 = sliceResults111[0]
					ctx.ReclaimUntrackedRegs()
					callResults113 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []int) []int { return append(dst, src...) }), []JITValueDesc{d108, d112}, []uint8{3}, []uint8{1})
					d114 = callResults113[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d114)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64][]int, key uint64, value []int) { m[key] = value }), []JITValueDesc{d106, d88, d114})
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_3_2)
					bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d115 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64][]int { return make(map[uint64][]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d115)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d115)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64][]int) { base.collisions = value }), []JITValueDesc{d16, d115})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl21)
					if r15 {
						ctx.UnprotectReg(r16)
					}
					if r17 {
						ctx.UnprotectReg(r18)
					}
					if r19 {
						ctx.UnprotectReg(r20)
					}
					if r21 {
						ctx.UnprotectReg(r22)
					}
					if r23 {
						ctx.UnprotectReg(r24)
					}
					if r25 {
						ctx.UnprotectReg(r26)
					}
					ctx.FreeDesc(&d78)
					ctx.FreeDesc(&d71)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d116 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d116)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d116)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d16, d116})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d117 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r53 := ctx.AllocReg()
						r54 := ctx.AllocRegExcept(r53)
						r55 := ctx.AllocRegExcept(r53, r54)
						ctx.EmitMovRegMem64(r53, fieldAddr)
						ctx.EmitMovRegMem64(r54, fieldAddr+8)
						ctx.EmitMovRegMem64(r55, fieldAddr+16)
						d117 = JITValueDesc{Loc: LocRegTriple, Reg: r53, Reg2: r54, Reg3: r55}
						ctx.BindReg(r53, &d117)
						ctx.BindReg(r54, &d117)
						ctx.BindReg(r55, &d117)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r56 := ctx.AllocRegExcept(baseReg)
						r57 := ctx.AllocRegExcept(baseReg, r56)
						r58 := ctx.AllocRegExcept(baseReg, r56, r57)
						ctx.EmitMovRegMem(r56, baseReg, off)
						ctx.EmitMovRegMem(r57, baseReg, off+8)
						ctx.EmitMovRegMem(r58, baseReg, off+16)
						d117 = JITValueDesc{Loc: LocRegTriple, Reg: r56, Reg2: r57, Reg3: r58}
						ctx.BindReg(r56, &d117)
						ctx.BindReg(r57, &d117)
						ctx.BindReg(r58, &d117)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					var d118 JITValueDesc
					if d73.Loc == LocImm {
						d118 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d73.Reg)
						ctx.EmitMovRegReg(scratch, d73.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d118 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d118)
					}
					if d118.Loc == LocReg && d73.Loc == LocReg && d118.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d119 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r59 := ctx.AllocReg()
						r60 := ctx.AllocRegExcept(r59)
						r61 := ctx.AllocRegExcept(r59, r60)
						ctx.EmitMovRegMem64(r59, fieldAddr)
						ctx.EmitMovRegMem64(r60, fieldAddr+8)
						ctx.EmitMovRegMem64(r61, fieldAddr+16)
						d119 = JITValueDesc{Loc: LocRegTriple, Reg: r59, Reg2: r60, Reg3: r61}
						ctx.BindReg(r59, &d119)
						ctx.BindReg(r60, &d119)
						ctx.BindReg(r61, &d119)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r62 := ctx.AllocRegExcept(baseReg)
						r63 := ctx.AllocRegExcept(baseReg, r62)
						r64 := ctx.AllocRegExcept(baseReg, r62, r63)
						ctx.EmitMovRegMem(r62, baseReg, off)
						ctx.EmitMovRegMem(r63, baseReg, off+8)
						ctx.EmitMovRegMem(r64, baseReg, off+16)
						d119 = JITValueDesc{Loc: LocRegTriple, Reg: r62, Reg2: r63, Reg3: r64}
						ctx.BindReg(r62, &d119)
						ctx.BindReg(r63, &d119)
						ctx.BindReg(r64, &d119)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EnsureDesc(&d73)
					var d120 JITValueDesc
					if d73.Loc == LocImm {
						d120 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d73.Reg)
						ctx.EmitMovRegReg(scratch, d73.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d120 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d120)
					}
					if d120.Loc == LocReg && d73.Loc == LocReg && d120.Reg == d73.Reg {
						ctx.TransferReg(d73.Reg)
						d73.Loc = LocNone
					}
					ctx.FreeDesc(&d73)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d120)
					ctx.ReclaimUntrackedRegs()
					d122 = ctx.EmitSliceElementAddress(&d119, &d120, 16)
					ctx.EnsureDesc(&d122)
					r65 := ctx.AllocRegExcept(d122.Reg)
					ctx.EmitMovRegMem(r65, d122.Reg, 8)
					ctx.EmitMovRegMem(d122.Reg, d122.Reg, 0)
					d121 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d122.Reg, Reg2: r65}
					ctx.BindReg(d122.Reg, &d121)
					ctx.BindReg(r65, &d121)
					ctx.FreeDesc(&d120)
					ctx.ReclaimUntrackedRegs()
					stackArray123 = ctx.AllocStack(int32(32))
					_ = stackArray123
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d121)
					ctx.EnsureDesc(&d121)
					ctx.EmitStoreScmerToStack(d121, int32(stackArray123)+int32(0))
					ctx.FreeDesc(&d121)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EmitStoreScmerToStack(d65, int32(stackArray123)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d124 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d124
					ctx.ReclaimUntrackedRegs()
					callbackArgs126 := make([]JITValueDesc, 2)
					callbackArgs126[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray123) + 0}
					callbackArgs126[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray123) + 16}
					var d125 JITValueDesc
					callbackResultOff127 = ctx.AllocStack(16)
					ctx.FreeDesc(&d124)
					if d67.Loc == LocLambdaTemplate && d67.Lambda != nil {
						stableCallbackArgs128 := ctx.StabilizeCallbackArgs(callbackArgs126)
						ctx.ReclaimUntrackedRegs()
						outerRegs129 := ctx.PreserveOuterRegs()
						d125 = JITEmitProcInlineWithOuter(ctx, &d67.Lambda.Proc, d67.Lambda.Outer, stableCallbackArgs128, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff127), ID: 0})
						ctx.RestoreOuterRegs(outerRegs129)
						ctx.ReclaimUntrackedRegs()
					} else {
						d130, knownBuiltin131 := jitEmitKnownDeclaration(ctx, d67, callbackArgs126, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff127), ID: 0})
						if knownBuiltin131 {
							d125 = d130
						} else {
							d132 := jitCopyScmerToPair(ctx, d67)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d132)
							callbackCallArgs = append(callbackCallArgs, callbackArgs126...)
							d125 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d125, int32(callbackResultOff127))
							ctx.FreeDesc(&d125)
							d125 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff127), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d118)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d125)
					d133 = ctx.EmitSliceElementAddress(&d117, &d118, int32(16))
					ctx.EmitStoreScmerAt(&d133, &d125)
					ctx.FreeDesc(&d133)
					ctx.FreeDesc(&d118)
					ctx.FreeDesc(&d125)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl12)
					if r6 {
						ctx.UnprotectReg(r7)
					}
					if r8 {
						ctx.UnprotectReg(r9)
					}
					if r10 {
						ctx.UnprotectReg(r11)
					}
					ctx.FreeDesc(&d55)
					ctx.FreeDesc(&d51)
					ctx.FreeDesc(&d63)
					if ps.General {
					}
					ps134 := PhiState{General: ps.General}
					ps134.OverlayValues = make([]JITValueDesc, 134)
					ps134.OverlayValues[1] = d1
					ps134.OverlayValues[2] = d2
					ps134.OverlayValues[3] = d3
					ps134.OverlayValues[4] = d4
					ps134.OverlayValues[5] = d5
					ps134.OverlayValues[7] = d7
					ps134.OverlayValues[8] = d8
					ps134.OverlayValues[10] = d10
					ps134.OverlayValues[11] = d11
					ps134.OverlayValues[12] = d12
					ps134.OverlayValues[13] = d13
					ps134.OverlayValues[14] = d14
					ps134.OverlayValues[15] = d15
					ps134.OverlayValues[16] = d16
					ps134.OverlayValues[17] = d17
					ps134.OverlayValues[19] = d19
					ps134.OverlayValues[20] = d20
					ps134.OverlayValues[21] = d21
					ps134.OverlayValues[22] = d22
					ps134.OverlayValues[23] = d23
					ps134.OverlayValues[26] = d26
					ps134.OverlayValues[51] = d51
					ps134.OverlayValues[52] = d52
					ps134.OverlayValues[54] = d54
					ps134.OverlayValues[55] = d55
					ps134.OverlayValues[60] = d60
					ps134.OverlayValues[62] = d62
					ps134.OverlayValues[63] = d63
					ps134.OverlayValues[64] = d64
					ps134.OverlayValues[65] = d65
					ps134.OverlayValues[66] = d66
					ps134.OverlayValues[67] = d67
					ps134.OverlayValues[68] = d68
					ps134.OverlayValues[69] = d69
					ps134.OverlayValues[70] = d70
					ps134.OverlayValues[71] = d71
					ps134.OverlayValues[73] = d73
					ps134.OverlayValues[74] = d74
					ps134.OverlayValues[75] = d75
					ps134.OverlayValues[77] = d77
					ps134.OverlayValues[78] = d78
					ps134.OverlayValues[83] = d83
					ps134.OverlayValues[85] = d85
					ps134.OverlayValues[86] = d86
					ps134.OverlayValues[87] = d87
					ps134.OverlayValues[88] = d88
					ps134.OverlayValues[89] = d89
					ps134.OverlayValues[90] = d90
					ps134.OverlayValues[91] = d91
					ps134.OverlayValues[93] = d93
					ps134.OverlayValues[94] = d94
					ps134.OverlayValues[96] = d96
					ps134.OverlayValues[97] = d97
					ps134.OverlayValues[99] = d99
					ps134.OverlayValues[100] = d100
					ps134.OverlayValues[101] = d101
					ps134.OverlayValues[102] = d102
					ps134.OverlayValues[103] = d103
					ps134.OverlayValues[104] = d104
					ps134.OverlayValues[105] = d105
					ps134.OverlayValues[106] = d106
					ps134.OverlayValues[107] = d107
					ps134.OverlayValues[108] = d108
					ps134.OverlayValues[109] = d109
					ps134.OverlayValues[110] = d110
					ps134.OverlayValues[112] = d112
					ps134.OverlayValues[114] = d114
					ps134.OverlayValues[115] = d115
					ps134.OverlayValues[116] = d116
					ps134.OverlayValues[117] = d117
					ps134.OverlayValues[118] = d118
					ps134.OverlayValues[119] = d119
					ps134.OverlayValues[120] = d120
					ps134.OverlayValues[121] = d121
					ps134.OverlayValues[122] = d122
					ps134.OverlayValues[124] = d124
					ps134.OverlayValues[125] = d125
					ps134.OverlayValues[130] = d130
					ps134.OverlayValues[132] = d132
					ps134.OverlayValues[133] = d133
					ps134.PhiValues = make([]JITValueDesc, 1)
					if ps134.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps134)
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
					if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
						d70 = ps.OverlayValues[70]
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
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
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
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
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
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
						d122 = ps.OverlayValues[122]
					}
					if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != LocNone {
						d124 = ps.OverlayValues[124]
					}
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					ctx.ReclaimUntrackedRegs()
					var d135 JITValueDesc
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r66 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r66, makeAux(tagFastDict, 0))
						d135 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r66}
						ctx.BindReg(d16.Reg, &d135)
						ctx.BindReg(r66, &d135)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d135)
						ctx.BindReg(r66, &d135)
						d16.Loc = LocNone
					}
					ctx.FreeDesc(&d16)
					ctx.EnsureDesc(&d135)
					if d135.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d135, &result)
						result.Type = d135.Type
					} else {
						switch d135.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d135)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d135)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d135)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d135, &result)
							result.Type = d135.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps136 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps136)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITInlineCallbacks: true,
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
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_append"].Fn, args, result)
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
				var d75 JITValueDesc
				_ = d75
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var stackArray84 int32
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
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
				var stackArray95 int32
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				var d102 JITValueDesc
				_ = d102
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
				var d109 JITValueDesc
				_ = d109
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d113 JITValueDesc
				_ = d113
				var d115 JITValueDesc
				_ = d115
				var d117 JITValueDesc
				_ = d117
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var stackArray127 int32
				var d128 JITValueDesc
				_ = d128
				var d129 JITValueDesc
				_ = d129
				var d131 JITValueDesc
				_ = d131
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d135 JITValueDesc
				_ = d135
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
						d5 = ctx.RequestOptimizedCallback(1)
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
						d8 = ctx.RequestOptimizedCallback(2)
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
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
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
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitJmp(lbl6)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitCmpRegImm32(d13.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl6)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d12)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
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
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d21)
					ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
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
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl11)
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
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray53)+int32(0))
					d54 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d54
					callbackArgs56 := make([]JITValueDesc, 1)
					callbackArgs56[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray53) + 0}
					var d55 JITValueDesc
					callbackResultOff57 = ctx.AllocStack(16)
					ctx.FreeDesc(&d54)
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
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d63)
					ctx.EmitStoreScmerToStack(d63, int32(stackArray64)+int32(0))
					ctx.FreeDesc(&d63)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray64)+int32(16))
					ctx.FreeDesc(&d51)
					d65 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d65
					callbackArgs67 := make([]JITValueDesc, 2)
					callbackArgs67[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray64) + 0}
					callbackArgs67[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray64) + 16}
					var d66 JITValueDesc
					callbackResultOff68 = ctx.AllocStack(16)
					ctx.FreeDesc(&d65)
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
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d66)
					d74 = d55
					_ = d74
					ctx.StabilizeDescForControlFlow(&d74)
					d75 = d66
					_ = d75
					ctx.StabilizeDescForControlFlow(&d75)
					r6 := d16.Loc == LocReg || d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r7 := d16.Reg
					if r6 {
						ctx.ProtectReg(r7)
					}
					r8 := d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r9 := d16.Reg2
					if r8 {
						ctx.ProtectReg(r9)
					}
					r10 := d16.Loc == LocRegTriple
					r11 := d16.Reg3
					if r10 {
						ctx.ProtectReg(r11)
					}
					lbl12 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					bbpos_2_4 := int32(-1)
					_ = bbpos_2_4
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d76 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r12 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r12, fieldAddr)
						d76 = JITValueDesc{Loc: LocReg, Reg: r12}
						ctx.BindReg(r12, &d76)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r13 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r13, baseReg, off)
						d76 = JITValueDesc{Loc: LocReg, Reg: r13}
						ctx.BindReg(r13, &d76)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d76)
					var d77 JITValueDesc
					if d76.Loc == LocImm {
						d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d76.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d76)
						if d76.Loc != LocReg && d76.Loc != LocRegPair && d76.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r14 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d76.Reg, 0)
						ctx.EmitSetcc(r14, CondEqual)
						d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d77)
					}
					ctx.FreeDesc(&d76)
					ctx.ReclaimUntrackedRegs()
					d78 = d77
					ctx.EnsureDesc(&d78)
					if d78.Loc != LocImm && d78.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d78.Loc == LocImm {
						if d78.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						}
					} else {
						ctx.EmitCmpRegImm32(d78.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d77)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d74.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d74.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d74)
						} else if d74.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d74)
						} else if d74.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d74)
						} else if d74.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d74.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d74 = tmpPair
					} else if d74.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d74.Type, Reg: ctx.AllocRegExcept(d74.Reg), Reg2: ctx.AllocRegExcept(d74.Reg)}
						switch d74.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d74)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d74)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d74)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d74)
						d74 = tmpPair
					}
					if d74.Loc != LocRegPair && d74.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (HashKey arg0)")
					}
					ctx.SyncDesc(&d74)
					d79 = ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d74}, 1)
					ctx.BindReg(d79.Reg, &d79)
					ctx.StabilizeDescForControlFlow(&d79)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d74)
					if d74.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d74.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d74.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d74)
						} else if d74.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d74)
						} else if d74.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d74)
						} else if d74.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d74.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d74 = tmpPair
					} else if d74.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d74.Type, Reg: ctx.AllocRegExcept(d74.Reg), Reg2: ctx.AllocRegExcept(d74.Reg)}
						switch d74.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d74)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d74)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d74)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d74)
						d74 = tmpPair
					}
					if d74.Loc != LocRegPair && d74.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*FastDict).findPos arg1)")
					}
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d79)
					if d79.Loc == LocRegPair || d79.Loc == LocStackPair || d79.Loc == LocRegTriple || d79.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d74)
					ctx.SyncDesc(&d79)
					callResults80 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d16, d74, d79}, []uint8{1, 1}, []uint8{0, 0})
					d81 = callResults80[0]
					_ = d81
					d82 = callResults80[1]
					_ = d82
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d81)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d83 = d82
					ctx.EnsureDesc(&d83)
					if d83.Loc != LocImm && d83.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d83.Loc == LocImm {
						if d83.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitCmpRegImm32(d83.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d82)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					stackArray84 = ctx.AllocStack(int32(16))
					_ = stackArray84
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray84)+int32(0))
					ctx.ReclaimUntrackedRegs()
					d85 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d85
					ctx.ReclaimUntrackedRegs()
					r15 := ctx.AllocReg()
					r16 := ctx.AllocRegExcept(r15)
					r17 := ctx.AllocRegExcept(r15, r16)
					d86 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r15, Reg2: r16, Reg3: r17}
					ctx.BindReg(r15, &d86)
					ctx.BindReg(r16, &d86)
					ctx.BindReg(r17, &d86)
					ctx.BindReg(r15, &d86)
					ctx.BindReg(r16, &d86)
					ctx.BindReg(r17, &d86)
					ctx.EmitLeaRegMem(d86.Reg, ctx.StackReg, int32(stackArray84))
					ctx.EmitMovRegImm64(d86.Reg2, uint64(1))
					ctx.EmitMovRegImm64(d86.Reg3, uint64(1))
					callResults87 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d86}, []uint8{2}, []uint8{1})
					d88 = callResults87[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d74)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d79)
					d89 = d74
					_ = d89
					ctx.StabilizeDescForControlFlow(&d89)
					d90 = d88
					_ = d90
					ctx.StabilizeDescForControlFlow(&d90)
					d91 = d79
					_ = d91
					ctx.StabilizeDescForControlFlow(&d91)
					r18 := d16.Loc == LocReg || d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r19 := d16.Reg
					if r18 {
						ctx.ProtectReg(r19)
					}
					r20 := d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r21 := d16.Reg2
					if r20 {
						ctx.ProtectReg(r21)
					}
					r22 := d16.Loc == LocRegTriple
					r23 := d16.Reg3
					if r22 {
						ctx.ProtectReg(r23)
					}
					r24 := d74.Loc == LocReg || d74.Loc == LocRegPair || d74.Loc == LocRegTriple
					r25 := d74.Reg
					if r24 {
						ctx.ProtectReg(r25)
					}
					r26 := d74.Loc == LocRegPair || d74.Loc == LocRegTriple
					r27 := d74.Reg2
					if r26 {
						ctx.ProtectReg(r27)
					}
					r28 := d74.Loc == LocRegTriple
					r29 := d74.Reg3
					if r28 {
						ctx.ProtectReg(r29)
					}
					lbl21 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					bbpos_3_3 := int32(-1)
					_ = bbpos_3_3
					bbpos_3_4 := int32(-1)
					_ = bbpos_3_4
					bbpos_3_5 := int32(-1)
					_ = bbpos_3_5
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d92 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r30 := ctx.AllocReg()
						r31 := ctx.AllocRegExcept(r30)
						r32 := ctx.AllocRegExcept(r30, r31)
						ctx.EmitMovRegMem64(r30, fieldAddr)
						ctx.EmitMovRegMem64(r31, fieldAddr+8)
						ctx.EmitMovRegMem64(r32, fieldAddr+16)
						d92 = JITValueDesc{Loc: LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
						ctx.BindReg(r30, &d92)
						ctx.BindReg(r31, &d92)
						ctx.BindReg(r32, &d92)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r33 := ctx.AllocRegExcept(baseReg)
						r34 := ctx.AllocRegExcept(baseReg, r33)
						r35 := ctx.AllocRegExcept(baseReg, r33, r34)
						ctx.EmitMovRegMem(r33, baseReg, off)
						ctx.EmitMovRegMem(r34, baseReg, off+8)
						ctx.EmitMovRegMem(r35, baseReg, off+16)
						d92 = JITValueDesc{Loc: LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
						ctx.BindReg(r33, &d92)
						ctx.BindReg(r34, &d92)
						ctx.BindReg(r35, &d92)
					}
					ctx.ReclaimUntrackedRegs()
					var d93 JITValueDesc
					if d92.SliceSizeKnown {
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d92.KnownSliceLen))}
					} else if d92.Loc == LocImm {
						d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d92.StackOff))}
					} else if d92.Loc == LocStackTriple {
						d93 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d92.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d92)
						if d92.Loc == LocRegPair || d92.Loc == LocRegTriple {
							d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d92.Reg2, ID: 0}
						} else if d92.Loc == LocReg {
							d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d92.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d93)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d94 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r36 := ctx.AllocReg()
						r37 := ctx.AllocRegExcept(r36)
						r38 := ctx.AllocRegExcept(r36, r37)
						ctx.EmitMovRegMem64(r36, fieldAddr)
						ctx.EmitMovRegMem64(r37, fieldAddr+8)
						ctx.EmitMovRegMem64(r38, fieldAddr+16)
						d94 = JITValueDesc{Loc: LocRegTriple, Reg: r36, Reg2: r37, Reg3: r38}
						ctx.BindReg(r36, &d94)
						ctx.BindReg(r37, &d94)
						ctx.BindReg(r38, &d94)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r39 := ctx.AllocRegExcept(baseReg)
						r40 := ctx.AllocRegExcept(baseReg, r39)
						r41 := ctx.AllocRegExcept(baseReg, r39, r40)
						ctx.EmitMovRegMem(r39, baseReg, off)
						ctx.EmitMovRegMem(r40, baseReg, off+8)
						ctx.EmitMovRegMem(r41, baseReg, off+16)
						d94 = JITValueDesc{Loc: LocRegTriple, Reg: r39, Reg2: r40, Reg3: r41}
						ctx.BindReg(r39, &d94)
						ctx.BindReg(r40, &d94)
						ctx.BindReg(r41, &d94)
					}
					ctx.ReclaimUntrackedRegs()
					stackArray95 = ctx.AllocStack(int32(32))
					_ = stackArray95
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d89)
					ctx.EmitStoreScmerToStack(d89, int32(stackArray95)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d90)
					ctx.EmitStoreScmerToStack(d90, int32(stackArray95)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d96 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d96
					ctx.ReclaimUntrackedRegs()
					r42 := ctx.AllocReg()
					r43 := ctx.AllocRegExcept(r42)
					r44 := ctx.AllocRegExcept(r42, r43)
					d97 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r42, Reg2: r43, Reg3: r44}
					ctx.BindReg(r42, &d97)
					ctx.BindReg(r43, &d97)
					ctx.BindReg(r44, &d97)
					ctx.BindReg(r42, &d97)
					ctx.BindReg(r43, &d97)
					ctx.BindReg(r44, &d97)
					ctx.EmitLeaRegMem(d97.Reg, ctx.StackReg, int32(stackArray95))
					ctx.EmitMovRegImm64(d97.Reg2, uint64(2))
					ctx.EmitMovRegImm64(d97.Reg3, uint64(2))
					callResults98 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d94, d97}, []uint8{3}, []uint8{1})
					d99 = callResults98[0]
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d99)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d99)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value []Scmer) { base.Pairs = value }), []JITValueDesc{d16, d99})
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d100 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r45 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r45, fieldAddr)
						d100 = JITValueDesc{Loc: LocReg, Reg: r45}
						ctx.BindReg(r45, &d100)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r46 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r46, baseReg, off)
						d100 = JITValueDesc{Loc: LocReg, Reg: r46}
						ctx.BindReg(r46, &d100)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d100)
					ctx.EnsureDesc(&d91)
					lookupResults101 := JITEmitGoCallResults(ctx, GoFuncAddr(func(m map[uint64]int, k uint64) (int, bool) { value, ok := m[k]; return value, ok }), []JITValueDesc{d100, d91}, []uint8{1, 1}, []uint8{0, 0})
					d102 = lookupResults101[0]
					d103 = lookupResults101[1]
					ctx.EmitAndRegImm32(d103.Reg, 1)
					d103.Type = tagBool
					ctx.FreeDesc(&d100)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d104 = d103
					ctx.EnsureDesc(&d104)
					if d104.Loc != LocImm && d104.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d104.Loc == LocImm {
						if d104.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl22)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitCmpRegImm32(d104.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
					ctx.FreeDesc(&d103)
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d105 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r47 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r47, fieldAddr)
						d105 = JITValueDesc{Loc: LocReg, Reg: r47}
						ctx.BindReg(r47, &d105)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r48 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r48, baseReg, off)
						d105 = JITValueDesc{Loc: LocReg, Reg: r48}
						ctx.BindReg(r48, &d105)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d91)
					ctx.EnsureDesc(&d93)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64]int, key uint64, value int) { m[key] = value }), []JITValueDesc{d105, d91, d93})
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl21)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d106 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r49 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r49, fieldAddr)
						d106 = JITValueDesc{Loc: LocReg, Reg: r49}
						ctx.BindReg(r49, &d106)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r50 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r50, baseReg, off)
						d106 = JITValueDesc{Loc: LocReg, Reg: r50}
						ctx.BindReg(r50, &d106)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					var d107 JITValueDesc
					if d106.Loc == LocImm {
						d107 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d106.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d106)
						if d106.Loc != LocReg && d106.Loc != LocRegPair && d106.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r51 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d106.Reg, 0)
						ctx.EmitSetcc(r51, CondEqual)
						d107 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r51}
						ctx.BindReg(r51, &d107)
					}
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					d108 = d107
					ctx.EnsureDesc(&d108)
					if d108.Loc != LocImm && d108.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					if d108.Loc == LocImm {
						if d108.Imm.Bool() {
							ctx.MarkLabel(lbl28)
							ctx.EmitJmp(lbl26)
						} else {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl27)
						}
					} else {
						ctx.EmitCmpRegImm32(d108.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl28)
						ctx.EmitJmp(lbl29)
						ctx.MarkLabel(lbl28)
						ctx.EmitJmp(lbl26)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
					}
					ctx.FreeDesc(&d107)
					bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d109 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r52 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r52, fieldAddr)
						d109 = JITValueDesc{Loc: LocReg, Reg: r52}
						ctx.BindReg(r52, &d109)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r53 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r53, baseReg, off)
						d109 = JITValueDesc{Loc: LocReg, Reg: r53}
						ctx.BindReg(r53, &d109)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d110 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r54 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r54, fieldAddr)
						d110 = JITValueDesc{Loc: LocReg, Reg: r54}
						ctx.BindReg(r54, &d110)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r55 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r55, baseReg, off)
						d110 = JITValueDesc{Loc: LocReg, Reg: r55}
						ctx.BindReg(r55, &d110)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					ctx.EnsureDesc(&d91)
					d111 = ctx.EmitGoCallScalar(GoFuncAddr(func(m map[uint64][]int, k uint64) []int { return m[k] }), []JITValueDesc{d110, d91}, 3)
					ctx.FreeDesc(&d110)
					ctx.ReclaimUntrackedRegs()
					d112 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]int { return new([1]int) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d113 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d93)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]int, index int, value int) { dst[index] = value }), []JITValueDesc{d112, d113, d93})
					ctx.FreeDesc(&d93)
					ctx.ReclaimUntrackedRegs()
					sliceResults114 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]int) []int { return value[0:1:1] }), []JITValueDesc{d112}, []uint8{3}, []uint8{1})
					d115 = sliceResults114[0]
					ctx.ReclaimUntrackedRegs()
					callResults116 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []int) []int { return append(dst, src...) }), []JITValueDesc{d111, d115}, []uint8{3}, []uint8{1})
					d117 = callResults116[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d109)
					ctx.EnsureDesc(&d91)
					ctx.EnsureDesc(&d117)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64][]int, key uint64, value []int) { m[key] = value }), []JITValueDesc{d109, d91, d117})
					ctx.FreeDesc(&d109)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_3_2)
					bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d118 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64][]int { return make(map[uint64][]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d118)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d118)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64][]int) { base.collisions = value }), []JITValueDesc{d16, d118})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl21)
					if r18 {
						ctx.UnprotectReg(r19)
					}
					if r20 {
						ctx.UnprotectReg(r21)
					}
					if r22 {
						ctx.UnprotectReg(r23)
					}
					if r24 {
						ctx.UnprotectReg(r25)
					}
					if r26 {
						ctx.UnprotectReg(r27)
					}
					if r28 {
						ctx.UnprotectReg(r29)
					}
					ctx.FreeDesc(&d88)
					ctx.FreeDesc(&d79)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d119 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d119)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d119)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d16, d119})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d120 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r56 := ctx.AllocReg()
						r57 := ctx.AllocRegExcept(r56)
						r58 := ctx.AllocRegExcept(r56, r57)
						ctx.EmitMovRegMem64(r56, fieldAddr)
						ctx.EmitMovRegMem64(r57, fieldAddr+8)
						ctx.EmitMovRegMem64(r58, fieldAddr+16)
						d120 = JITValueDesc{Loc: LocRegTriple, Reg: r56, Reg2: r57, Reg3: r58}
						ctx.BindReg(r56, &d120)
						ctx.BindReg(r57, &d120)
						ctx.BindReg(r58, &d120)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r59 := ctx.AllocRegExcept(baseReg)
						r60 := ctx.AllocRegExcept(baseReg, r59)
						r61 := ctx.AllocRegExcept(baseReg, r59, r60)
						ctx.EmitMovRegMem(r59, baseReg, off)
						ctx.EmitMovRegMem(r60, baseReg, off+8)
						ctx.EmitMovRegMem(r61, baseReg, off+16)
						d120 = JITValueDesc{Loc: LocRegTriple, Reg: r59, Reg2: r60, Reg3: r61}
						ctx.BindReg(r59, &d120)
						ctx.BindReg(r60, &d120)
						ctx.BindReg(r61, &d120)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d81)
					ctx.EnsureDesc(&d81)
					var d121 JITValueDesc
					if d81.Loc == LocImm {
						d121 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d81.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d81.Reg)
						ctx.EmitMovRegReg(scratch, d81.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d121)
					}
					if d121.Loc == LocReg && d81.Loc == LocReg && d121.Reg == d81.Reg {
						ctx.TransferReg(d81.Reg)
						d81.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d121)
					ctx.ReclaimUntrackedRegs()
					d123 = ctx.EmitSliceElementAddress(&d120, &d121, 16)
					ctx.EnsureDesc(&d123)
					r62 := ctx.AllocRegExcept(d123.Reg)
					ctx.EmitMovRegMem(r62, d123.Reg, 8)
					ctx.EmitMovRegMem(d123.Reg, d123.Reg, 0)
					d122 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d123.Reg, Reg2: r62}
					ctx.BindReg(d123.Reg, &d122)
					ctx.BindReg(r62, &d122)
					ctx.FreeDesc(&d121)
					ctx.ReclaimUntrackedRegs()
					var d124 JITValueDesc
					if d122.Type == tagSlice {
						d124 = jitKnownSliceHeader(ctx, &d122)
					} else {
						d124 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d122}, 3)
					}
					ctx.BindReg(d124.Reg, &d124)
					ctx.BindReg(d124.Reg2, &d124)
					ctx.BindReg(d124.Reg3, &d124)
					ctx.FreeDesc(&d122)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d125 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r63 := ctx.AllocReg()
						r64 := ctx.AllocRegExcept(r63)
						r65 := ctx.AllocRegExcept(r63, r64)
						ctx.EmitMovRegMem64(r63, fieldAddr)
						ctx.EmitMovRegMem64(r64, fieldAddr+8)
						ctx.EmitMovRegMem64(r65, fieldAddr+16)
						d125 = JITValueDesc{Loc: LocRegTriple, Reg: r63, Reg2: r64, Reg3: r65}
						ctx.BindReg(r63, &d125)
						ctx.BindReg(r64, &d125)
						ctx.BindReg(r65, &d125)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r66 := ctx.AllocRegExcept(baseReg)
						r67 := ctx.AllocRegExcept(baseReg, r66)
						r68 := ctx.AllocRegExcept(baseReg, r66, r67)
						ctx.EmitMovRegMem(r66, baseReg, off)
						ctx.EmitMovRegMem(r67, baseReg, off+8)
						ctx.EmitMovRegMem(r68, baseReg, off+16)
						d125 = JITValueDesc{Loc: LocRegTriple, Reg: r66, Reg2: r67, Reg3: r68}
						ctx.BindReg(r66, &d125)
						ctx.BindReg(r67, &d125)
						ctx.BindReg(r68, &d125)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d81)
					ctx.EnsureDesc(&d81)
					var d126 JITValueDesc
					if d81.Loc == LocImm {
						d126 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d81.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d81.Reg)
						ctx.EmitMovRegReg(scratch, d81.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d126 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d126)
					}
					if d126.Loc == LocReg && d81.Loc == LocReg && d126.Reg == d81.Reg {
						ctx.TransferReg(d81.Reg)
						d81.Loc = LocNone
					}
					ctx.FreeDesc(&d81)
					ctx.ReclaimUntrackedRegs()
					stackArray127 = ctx.AllocStack(int32(16))
					_ = stackArray127
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray127)+int32(0))
					ctx.ReclaimUntrackedRegs()
					d128 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d128
					ctx.ReclaimUntrackedRegs()
					r69 := ctx.AllocReg()
					r70 := ctx.AllocRegExcept(r69)
					r71 := ctx.AllocRegExcept(r69, r70)
					d129 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r69, Reg2: r70, Reg3: r71}
					ctx.BindReg(r69, &d129)
					ctx.BindReg(r70, &d129)
					ctx.BindReg(r71, &d129)
					ctx.BindReg(r69, &d129)
					ctx.BindReg(r70, &d129)
					ctx.BindReg(r71, &d129)
					ctx.EmitLeaRegMem(d129.Reg, ctx.StackReg, int32(stackArray127))
					ctx.EmitMovRegImm64(d129.Reg2, uint64(1))
					ctx.EmitMovRegImm64(d129.Reg3, uint64(1))
					callResults130 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d124, d129}, []uint8{3}, []uint8{1})
					d131 = callResults130[0]
					ctx.ReclaimUntrackedRegs()
					d132 = ctx.EmitNewSliceFromGoSlice(&d131)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d126)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d132)
					d133 = ctx.EmitSliceElementAddress(&d125, &d126, int32(16))
					ctx.EmitStoreScmerAt(&d133, &d132)
					ctx.FreeDesc(&d133)
					ctx.FreeDesc(&d126)
					ctx.FreeDesc(&d132)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl12)
					if r6 {
						ctx.UnprotectReg(r7)
					}
					if r8 {
						ctx.UnprotectReg(r9)
					}
					if r10 {
						ctx.UnprotectReg(r11)
					}
					ctx.FreeDesc(&d55)
					ctx.FreeDesc(&d66)
					if ps.General {
					}
					ps134 := PhiState{General: ps.General}
					ps134.OverlayValues = make([]JITValueDesc, 134)
					ps134.OverlayValues[1] = d1
					ps134.OverlayValues[2] = d2
					ps134.OverlayValues[3] = d3
					ps134.OverlayValues[4] = d4
					ps134.OverlayValues[5] = d5
					ps134.OverlayValues[7] = d7
					ps134.OverlayValues[8] = d8
					ps134.OverlayValues[10] = d10
					ps134.OverlayValues[11] = d11
					ps134.OverlayValues[12] = d12
					ps134.OverlayValues[13] = d13
					ps134.OverlayValues[14] = d14
					ps134.OverlayValues[15] = d15
					ps134.OverlayValues[16] = d16
					ps134.OverlayValues[17] = d17
					ps134.OverlayValues[19] = d19
					ps134.OverlayValues[20] = d20
					ps134.OverlayValues[21] = d21
					ps134.OverlayValues[22] = d22
					ps134.OverlayValues[23] = d23
					ps134.OverlayValues[26] = d26
					ps134.OverlayValues[51] = d51
					ps134.OverlayValues[52] = d52
					ps134.OverlayValues[54] = d54
					ps134.OverlayValues[55] = d55
					ps134.OverlayValues[60] = d60
					ps134.OverlayValues[62] = d62
					ps134.OverlayValues[63] = d63
					ps134.OverlayValues[65] = d65
					ps134.OverlayValues[66] = d66
					ps134.OverlayValues[71] = d71
					ps134.OverlayValues[73] = d73
					ps134.OverlayValues[74] = d74
					ps134.OverlayValues[75] = d75
					ps134.OverlayValues[76] = d76
					ps134.OverlayValues[77] = d77
					ps134.OverlayValues[78] = d78
					ps134.OverlayValues[79] = d79
					ps134.OverlayValues[81] = d81
					ps134.OverlayValues[82] = d82
					ps134.OverlayValues[83] = d83
					ps134.OverlayValues[85] = d85
					ps134.OverlayValues[86] = d86
					ps134.OverlayValues[88] = d88
					ps134.OverlayValues[89] = d89
					ps134.OverlayValues[90] = d90
					ps134.OverlayValues[91] = d91
					ps134.OverlayValues[92] = d92
					ps134.OverlayValues[93] = d93
					ps134.OverlayValues[94] = d94
					ps134.OverlayValues[96] = d96
					ps134.OverlayValues[97] = d97
					ps134.OverlayValues[99] = d99
					ps134.OverlayValues[100] = d100
					ps134.OverlayValues[102] = d102
					ps134.OverlayValues[103] = d103
					ps134.OverlayValues[104] = d104
					ps134.OverlayValues[105] = d105
					ps134.OverlayValues[106] = d106
					ps134.OverlayValues[107] = d107
					ps134.OverlayValues[108] = d108
					ps134.OverlayValues[109] = d109
					ps134.OverlayValues[110] = d110
					ps134.OverlayValues[111] = d111
					ps134.OverlayValues[112] = d112
					ps134.OverlayValues[113] = d113
					ps134.OverlayValues[115] = d115
					ps134.OverlayValues[117] = d117
					ps134.OverlayValues[118] = d118
					ps134.OverlayValues[119] = d119
					ps134.OverlayValues[120] = d120
					ps134.OverlayValues[121] = d121
					ps134.OverlayValues[122] = d122
					ps134.OverlayValues[123] = d123
					ps134.OverlayValues[124] = d124
					ps134.OverlayValues[125] = d125
					ps134.OverlayValues[126] = d126
					ps134.OverlayValues[128] = d128
					ps134.OverlayValues[129] = d129
					ps134.OverlayValues[131] = d131
					ps134.OverlayValues[132] = d132
					ps134.OverlayValues[133] = d133
					ps134.PhiValues = make([]JITValueDesc, 1)
					if ps134.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps134)
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
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
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
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
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
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
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
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
						d117 = ps.OverlayValues[117]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
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
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
						d128 = ps.OverlayValues[128]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					ctx.ReclaimUntrackedRegs()
					var d135 JITValueDesc
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r72 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r72, makeAux(tagFastDict, 0))
						d135 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r72}
						ctx.BindReg(d16.Reg, &d135)
						ctx.BindReg(r72, &d135)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d135)
						ctx.BindReg(r72, &d135)
						d16.Loc = LocNone
					}
					ctx.FreeDesc(&d16)
					ctx.EnsureDesc(&d135)
					if d135.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d135, &result)
						result.Type = d135.Type
					} else {
						switch d135.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d135)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d135)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d135)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d135, &result)
							result.Type = d135.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps136 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps136)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITInlineCallbacks: true,
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
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_append_reduce"].Fn, args, result)
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
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var stackArray85 int32
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d89 JITValueDesc
				_ = d89
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
				var stackArray96 int32
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
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
				var d109 JITValueDesc
				_ = d109
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
				var d116 JITValueDesc
				_ = d116
				var d118 JITValueDesc
				_ = d118
				var d119 JITValueDesc
				_ = d119
				var d120 JITValueDesc
				_ = d120
				var d121 JITValueDesc
				_ = d121
				var d122 JITValueDesc
				_ = d122
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d127 JITValueDesc
				_ = d127
				var stackArray128 int32
				var d129 JITValueDesc
				_ = d129
				var d130 JITValueDesc
				_ = d130
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d136 JITValueDesc
				_ = d136
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
						d5 = ctx.RequestOptimizedCallback(1)
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
						d8 = ctx.RequestOptimizedCallback(2)
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
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
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
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d13.Loc == LocImm {
						if d13.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitJmp(lbl6)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitCmpRegImm32(d13.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl6)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d12)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
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
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d21)
					ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d21)
					ctx.EnsureDesc(&d17)
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
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d23.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl11)
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
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray54)+int32(0))
					ctx.FreeDesc(&d53)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray54)+int32(16))
					d55 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d55
					callbackArgs57 := make([]JITValueDesc, 2)
					callbackArgs57[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray54) + 0}
					callbackArgs57[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray54) + 16}
					var d56 JITValueDesc
					callbackResultOff58 = ctx.AllocStack(16)
					ctx.FreeDesc(&d55)
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
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					ctx.EmitStoreScmerToStack(d64, int32(stackArray65)+int32(0))
					ctx.FreeDesc(&d64)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray65)+int32(16))
					ctx.FreeDesc(&d51)
					d66 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d66
					callbackArgs68 := make([]JITValueDesc, 2)
					callbackArgs68[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray65) + 0}
					callbackArgs68[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray65) + 16}
					var d67 JITValueDesc
					callbackResultOff69 = ctx.AllocStack(16)
					ctx.FreeDesc(&d66)
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
					ctx.EnsureDesc(&d56)
					ctx.EnsureDesc(&d67)
					d75 = d56
					_ = d75
					ctx.StabilizeDescForControlFlow(&d75)
					d76 = d67
					_ = d76
					ctx.StabilizeDescForControlFlow(&d76)
					r6 := d16.Loc == LocReg || d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r7 := d16.Reg
					if r6 {
						ctx.ProtectReg(r7)
					}
					r8 := d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r9 := d16.Reg2
					if r8 {
						ctx.ProtectReg(r9)
					}
					r10 := d16.Loc == LocRegTriple
					r11 := d16.Reg3
					if r10 {
						ctx.ProtectReg(r11)
					}
					lbl12 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					bbpos_2_4 := int32(-1)
					_ = bbpos_2_4
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d77 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r12 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r12, fieldAddr)
						d77 = JITValueDesc{Loc: LocReg, Reg: r12}
						ctx.BindReg(r12, &d77)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r13 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r13, baseReg, off)
						d77 = JITValueDesc{Loc: LocReg, Reg: r13}
						ctx.BindReg(r13, &d77)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d77)
					var d78 JITValueDesc
					if d77.Loc == LocImm {
						d78 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d77.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d77)
						if d77.Loc != LocReg && d77.Loc != LocRegPair && d77.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r14 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d77.Reg, 0)
						ctx.EmitSetcc(r14, CondEqual)
						d78 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d78)
					}
					ctx.FreeDesc(&d77)
					ctx.ReclaimUntrackedRegs()
					d79 = d78
					ctx.EnsureDesc(&d79)
					if d79.Loc != LocImm && d79.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d79.Loc == LocImm {
						if d79.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						}
					} else {
						ctx.EmitCmpRegImm32(d79.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d78)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d75.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d75.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d75)
						} else if d75.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d75)
						} else if d75.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d75)
						} else if d75.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d75.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d75 = tmpPair
					} else if d75.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d75.Type, Reg: ctx.AllocRegExcept(d75.Reg), Reg2: ctx.AllocRegExcept(d75.Reg)}
						switch d75.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d75)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d75)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d75)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d75)
						d75 = tmpPair
					}
					if d75.Loc != LocRegPair && d75.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (HashKey arg0)")
					}
					ctx.SyncDesc(&d75)
					d80 = ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d75}, 1)
					ctx.BindReg(d80.Reg, &d80)
					ctx.StabilizeDescForControlFlow(&d80)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair || d16.Loc == LocStackPair || d16.Loc == LocRegTriple || d16.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d75.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d75.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d75)
						} else if d75.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d75)
						} else if d75.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d75)
						} else if d75.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d75.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d75 = tmpPair
					} else if d75.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d75.Type, Reg: ctx.AllocRegExcept(d75.Reg), Reg2: ctx.AllocRegExcept(d75.Reg)}
						switch d75.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d75)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d75)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d75)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d75)
						d75 = tmpPair
					}
					if d75.Loc != LocRegPair && d75.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*FastDict).findPos arg1)")
					}
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d80)
					if d80.Loc == LocRegPair || d80.Loc == LocStackPair || d80.Loc == LocRegTriple || d80.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d16)
					ctx.SyncDesc(&d75)
					ctx.SyncDesc(&d80)
					callResults81 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d16, d75, d80}, []uint8{1, 1}, []uint8{0, 0})
					d82 = callResults81[0]
					_ = d82
					d83 = callResults81[1]
					_ = d83
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d82)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d84 = d83
					ctx.EnsureDesc(&d84)
					if d84.Loc != LocImm && d84.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d84.Loc == LocImm {
						if d84.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitCmpRegImm32(d84.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d83)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					stackArray85 = ctx.AllocStack(int32(16))
					_ = stackArray85
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d76)
					ctx.EnsureDesc(&d76)
					ctx.EmitStoreScmerToStack(d76, int32(stackArray85)+int32(0))
					ctx.ReclaimUntrackedRegs()
					d86 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d86
					ctx.ReclaimUntrackedRegs()
					r15 := ctx.AllocReg()
					r16 := ctx.AllocRegExcept(r15)
					r17 := ctx.AllocRegExcept(r15, r16)
					d87 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r15, Reg2: r16, Reg3: r17}
					ctx.BindReg(r15, &d87)
					ctx.BindReg(r16, &d87)
					ctx.BindReg(r17, &d87)
					ctx.BindReg(r15, &d87)
					ctx.BindReg(r16, &d87)
					ctx.BindReg(r17, &d87)
					ctx.EmitLeaRegMem(d87.Reg, ctx.StackReg, int32(stackArray85))
					ctx.EmitMovRegImm64(d87.Reg2, uint64(1))
					ctx.EmitMovRegImm64(d87.Reg3, uint64(1))
					callResults88 := JITEmitGoCallResults(ctx, GoFuncAddr(JITNewSliceCopy), []JITValueDesc{d87}, []uint8{2}, []uint8{1})
					d89 = callResults88[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d75)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d80)
					d90 = d75
					_ = d90
					ctx.StabilizeDescForControlFlow(&d90)
					d91 = d89
					_ = d91
					ctx.StabilizeDescForControlFlow(&d91)
					d92 = d80
					_ = d92
					ctx.StabilizeDescForControlFlow(&d92)
					r18 := d16.Loc == LocReg || d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r19 := d16.Reg
					if r18 {
						ctx.ProtectReg(r19)
					}
					r20 := d16.Loc == LocRegPair || d16.Loc == LocRegTriple
					r21 := d16.Reg2
					if r20 {
						ctx.ProtectReg(r21)
					}
					r22 := d16.Loc == LocRegTriple
					r23 := d16.Reg3
					if r22 {
						ctx.ProtectReg(r23)
					}
					r24 := d75.Loc == LocReg || d75.Loc == LocRegPair || d75.Loc == LocRegTriple
					r25 := d75.Reg
					if r24 {
						ctx.ProtectReg(r25)
					}
					r26 := d75.Loc == LocRegPair || d75.Loc == LocRegTriple
					r27 := d75.Reg2
					if r26 {
						ctx.ProtectReg(r27)
					}
					r28 := d75.Loc == LocRegTriple
					r29 := d75.Reg3
					if r28 {
						ctx.ProtectReg(r29)
					}
					lbl21 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					bbpos_3_3 := int32(-1)
					_ = bbpos_3_3
					bbpos_3_4 := int32(-1)
					_ = bbpos_3_4
					bbpos_3_5 := int32(-1)
					_ = bbpos_3_5
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d93 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r30 := ctx.AllocReg()
						r31 := ctx.AllocRegExcept(r30)
						r32 := ctx.AllocRegExcept(r30, r31)
						ctx.EmitMovRegMem64(r30, fieldAddr)
						ctx.EmitMovRegMem64(r31, fieldAddr+8)
						ctx.EmitMovRegMem64(r32, fieldAddr+16)
						d93 = JITValueDesc{Loc: LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
						ctx.BindReg(r30, &d93)
						ctx.BindReg(r31, &d93)
						ctx.BindReg(r32, &d93)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r33 := ctx.AllocRegExcept(baseReg)
						r34 := ctx.AllocRegExcept(baseReg, r33)
						r35 := ctx.AllocRegExcept(baseReg, r33, r34)
						ctx.EmitMovRegMem(r33, baseReg, off)
						ctx.EmitMovRegMem(r34, baseReg, off+8)
						ctx.EmitMovRegMem(r35, baseReg, off+16)
						d93 = JITValueDesc{Loc: LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
						ctx.BindReg(r33, &d93)
						ctx.BindReg(r34, &d93)
						ctx.BindReg(r35, &d93)
					}
					ctx.ReclaimUntrackedRegs()
					var d94 JITValueDesc
					if d93.SliceSizeKnown {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d93.KnownSliceLen))}
					} else if d93.Loc == LocImm {
						d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d93.StackOff))}
					} else if d93.Loc == LocStackTriple {
						d94 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d93.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d93)
						if d93.Loc == LocRegPair || d93.Loc == LocRegTriple {
							d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d93.Reg2, ID: 0}
						} else if d93.Loc == LocReg {
							d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d93.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d94)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d95 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r36 := ctx.AllocReg()
						r37 := ctx.AllocRegExcept(r36)
						r38 := ctx.AllocRegExcept(r36, r37)
						ctx.EmitMovRegMem64(r36, fieldAddr)
						ctx.EmitMovRegMem64(r37, fieldAddr+8)
						ctx.EmitMovRegMem64(r38, fieldAddr+16)
						d95 = JITValueDesc{Loc: LocRegTriple, Reg: r36, Reg2: r37, Reg3: r38}
						ctx.BindReg(r36, &d95)
						ctx.BindReg(r37, &d95)
						ctx.BindReg(r38, &d95)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r39 := ctx.AllocRegExcept(baseReg)
						r40 := ctx.AllocRegExcept(baseReg, r39)
						r41 := ctx.AllocRegExcept(baseReg, r39, r40)
						ctx.EmitMovRegMem(r39, baseReg, off)
						ctx.EmitMovRegMem(r40, baseReg, off+8)
						ctx.EmitMovRegMem(r41, baseReg, off+16)
						d95 = JITValueDesc{Loc: LocRegTriple, Reg: r39, Reg2: r40, Reg3: r41}
						ctx.BindReg(r39, &d95)
						ctx.BindReg(r40, &d95)
						ctx.BindReg(r41, &d95)
					}
					ctx.ReclaimUntrackedRegs()
					stackArray96 = ctx.AllocStack(int32(32))
					_ = stackArray96
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d90)
					ctx.EmitStoreScmerToStack(d90, int32(stackArray96)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d91)
					ctx.EnsureDesc(&d91)
					ctx.EmitStoreScmerToStack(d91, int32(stackArray96)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d97 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d97
					ctx.ReclaimUntrackedRegs()
					r42 := ctx.AllocReg()
					r43 := ctx.AllocRegExcept(r42)
					r44 := ctx.AllocRegExcept(r42, r43)
					d98 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r42, Reg2: r43, Reg3: r44}
					ctx.BindReg(r42, &d98)
					ctx.BindReg(r43, &d98)
					ctx.BindReg(r44, &d98)
					ctx.BindReg(r42, &d98)
					ctx.BindReg(r43, &d98)
					ctx.BindReg(r44, &d98)
					ctx.EmitLeaRegMem(d98.Reg, ctx.StackReg, int32(stackArray96))
					ctx.EmitMovRegImm64(d98.Reg2, uint64(2))
					ctx.EmitMovRegImm64(d98.Reg3, uint64(2))
					callResults99 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d95, d98}, []uint8{3}, []uint8{1})
					d100 = callResults99[0]
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d100)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d100)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value []Scmer) { base.Pairs = value }), []JITValueDesc{d16, d100})
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d101 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r45 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r45, fieldAddr)
						d101 = JITValueDesc{Loc: LocReg, Reg: r45}
						ctx.BindReg(r45, &d101)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r46 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r46, baseReg, off)
						d101 = JITValueDesc{Loc: LocReg, Reg: r46}
						ctx.BindReg(r46, &d101)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d101)
					ctx.EnsureDesc(&d92)
					lookupResults102 := JITEmitGoCallResults(ctx, GoFuncAddr(func(m map[uint64]int, k uint64) (int, bool) { value, ok := m[k]; return value, ok }), []JITValueDesc{d101, d92}, []uint8{1, 1}, []uint8{0, 0})
					d103 = lookupResults102[0]
					d104 = lookupResults102[1]
					ctx.EmitAndRegImm32(d104.Reg, 1)
					d104.Type = tagBool
					ctx.FreeDesc(&d101)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d105 = d104
					ctx.EnsureDesc(&d105)
					if d105.Loc != LocImm && d105.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d105.Loc == LocImm {
						if d105.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl22)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitCmpRegImm32(d105.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
					ctx.FreeDesc(&d104)
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d106 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 24
						r47 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r47, fieldAddr)
						d106 = JITValueDesc{Loc: LocReg, Reg: r47}
						ctx.BindReg(r47, &d106)
					} else {
						off := int32(24)
						baseReg := d16.Reg
						r48 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r48, baseReg, off)
						d106 = JITValueDesc{Loc: LocReg, Reg: r48}
						ctx.BindReg(r48, &d106)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d92)
					ctx.EnsureDesc(&d94)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64]int, key uint64, value int) { m[key] = value }), []JITValueDesc{d106, d92, d94})
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl21)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d107 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r49 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r49, fieldAddr)
						d107 = JITValueDesc{Loc: LocReg, Reg: r49}
						ctx.BindReg(r49, &d107)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r50 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r50, baseReg, off)
						d107 = JITValueDesc{Loc: LocReg, Reg: r50}
						ctx.BindReg(r50, &d107)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d107)
					var d108 JITValueDesc
					if d107.Loc == LocImm {
						d108 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d107.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d107)
						if d107.Loc != LocReg && d107.Loc != LocRegPair && d107.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r51 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d107.Reg, 0)
						ctx.EmitSetcc(r51, CondEqual)
						d108 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r51}
						ctx.BindReg(r51, &d108)
					}
					ctx.FreeDesc(&d107)
					ctx.ReclaimUntrackedRegs()
					d109 = d108
					ctx.EnsureDesc(&d109)
					if d109.Loc != LocImm && d109.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					if d109.Loc == LocImm {
						if d109.Imm.Bool() {
							ctx.MarkLabel(lbl28)
							ctx.EmitJmp(lbl26)
						} else {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl27)
						}
					} else {
						ctx.EmitCmpRegImm32(d109.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl28)
						ctx.EmitJmp(lbl29)
						ctx.MarkLabel(lbl28)
						ctx.EmitJmp(lbl26)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
					}
					ctx.FreeDesc(&d108)
					bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d110 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r52 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r52, fieldAddr)
						d110 = JITValueDesc{Loc: LocReg, Reg: r52}
						ctx.BindReg(r52, &d110)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r53 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r53, baseReg, off)
						d110 = JITValueDesc{Loc: LocReg, Reg: r53}
						ctx.BindReg(r53, &d110)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d111 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 32
						r54 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r54, fieldAddr)
						d111 = JITValueDesc{Loc: LocReg, Reg: r54}
						ctx.BindReg(r54, &d111)
					} else {
						off := int32(32)
						baseReg := d16.Reg
						r55 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r55, baseReg, off)
						d111 = JITValueDesc{Loc: LocReg, Reg: r55}
						ctx.BindReg(r55, &d111)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d111)
					ctx.EnsureDesc(&d92)
					d112 = ctx.EmitGoCallScalar(GoFuncAddr(func(m map[uint64][]int, k uint64) []int { return m[k] }), []JITValueDesc{d111, d92}, 3)
					ctx.FreeDesc(&d111)
					ctx.ReclaimUntrackedRegs()
					d113 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]int { return new([1]int) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d114 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d94)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]int, index int, value int) { dst[index] = value }), []JITValueDesc{d113, d114, d94})
					ctx.FreeDesc(&d94)
					ctx.ReclaimUntrackedRegs()
					sliceResults115 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]int) []int { return value[0:1:1] }), []JITValueDesc{d113}, []uint8{3}, []uint8{1})
					d116 = sliceResults115[0]
					ctx.ReclaimUntrackedRegs()
					callResults117 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []int) []int { return append(dst, src...) }), []JITValueDesc{d112, d116}, []uint8{3}, []uint8{1})
					d118 = callResults117[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					ctx.EnsureDesc(&d92)
					ctx.EnsureDesc(&d118)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64][]int, key uint64, value []int) { m[key] = value }), []JITValueDesc{d110, d92, d118})
					ctx.FreeDesc(&d110)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_3_2)
					bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d119 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64][]int { return make(map[uint64][]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d119)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d119)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64][]int) { base.collisions = value }), []JITValueDesc{d16, d119})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl21)
					if r18 {
						ctx.UnprotectReg(r19)
					}
					if r20 {
						ctx.UnprotectReg(r21)
					}
					if r22 {
						ctx.UnprotectReg(r23)
					}
					if r24 {
						ctx.UnprotectReg(r25)
					}
					if r26 {
						ctx.UnprotectReg(r27)
					}
					if r28 {
						ctx.UnprotectReg(r29)
					}
					ctx.FreeDesc(&d89)
					ctx.FreeDesc(&d80)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d120 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d120)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d120)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d16, d120})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d121 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r56 := ctx.AllocReg()
						r57 := ctx.AllocRegExcept(r56)
						r58 := ctx.AllocRegExcept(r56, r57)
						ctx.EmitMovRegMem64(r56, fieldAddr)
						ctx.EmitMovRegMem64(r57, fieldAddr+8)
						ctx.EmitMovRegMem64(r58, fieldAddr+16)
						d121 = JITValueDesc{Loc: LocRegTriple, Reg: r56, Reg2: r57, Reg3: r58}
						ctx.BindReg(r56, &d121)
						ctx.BindReg(r57, &d121)
						ctx.BindReg(r58, &d121)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r59 := ctx.AllocRegExcept(baseReg)
						r60 := ctx.AllocRegExcept(baseReg, r59)
						r61 := ctx.AllocRegExcept(baseReg, r59, r60)
						ctx.EmitMovRegMem(r59, baseReg, off)
						ctx.EmitMovRegMem(r60, baseReg, off+8)
						ctx.EmitMovRegMem(r61, baseReg, off+16)
						d121 = JITValueDesc{Loc: LocRegTriple, Reg: r59, Reg2: r60, Reg3: r61}
						ctx.BindReg(r59, &d121)
						ctx.BindReg(r60, &d121)
						ctx.BindReg(r61, &d121)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d82)
					ctx.EnsureDesc(&d82)
					var d122 JITValueDesc
					if d82.Loc == LocImm {
						d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d82.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d82.Reg)
						ctx.EmitMovRegReg(scratch, d82.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d122)
					}
					if d122.Loc == LocReg && d82.Loc == LocReg && d122.Reg == d82.Reg {
						ctx.TransferReg(d82.Reg)
						d82.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d122)
					ctx.ReclaimUntrackedRegs()
					d124 = ctx.EmitSliceElementAddress(&d121, &d122, 16)
					ctx.EnsureDesc(&d124)
					r62 := ctx.AllocRegExcept(d124.Reg)
					ctx.EmitMovRegMem(r62, d124.Reg, 8)
					ctx.EmitMovRegMem(d124.Reg, d124.Reg, 0)
					d123 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d124.Reg, Reg2: r62}
					ctx.BindReg(d124.Reg, &d123)
					ctx.BindReg(r62, &d123)
					ctx.FreeDesc(&d122)
					ctx.ReclaimUntrackedRegs()
					var d125 JITValueDesc
					if d123.Type == tagSlice {
						d125 = jitKnownSliceHeader(ctx, &d123)
					} else {
						d125 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d123}, 3)
					}
					ctx.BindReg(d125.Reg, &d125)
					ctx.BindReg(d125.Reg2, &d125)
					ctx.BindReg(d125.Reg3, &d125)
					ctx.FreeDesc(&d123)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d126 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						fieldAddr := uintptr(d16.Imm.Int()) + 0
						r63 := ctx.AllocReg()
						r64 := ctx.AllocRegExcept(r63)
						r65 := ctx.AllocRegExcept(r63, r64)
						ctx.EmitMovRegMem64(r63, fieldAddr)
						ctx.EmitMovRegMem64(r64, fieldAddr+8)
						ctx.EmitMovRegMem64(r65, fieldAddr+16)
						d126 = JITValueDesc{Loc: LocRegTriple, Reg: r63, Reg2: r64, Reg3: r65}
						ctx.BindReg(r63, &d126)
						ctx.BindReg(r64, &d126)
						ctx.BindReg(r65, &d126)
					} else {
						off := int32(0)
						baseReg := d16.Reg
						r66 := ctx.AllocRegExcept(baseReg)
						r67 := ctx.AllocRegExcept(baseReg, r66)
						r68 := ctx.AllocRegExcept(baseReg, r66, r67)
						ctx.EmitMovRegMem(r66, baseReg, off)
						ctx.EmitMovRegMem(r67, baseReg, off+8)
						ctx.EmitMovRegMem(r68, baseReg, off+16)
						d126 = JITValueDesc{Loc: LocRegTriple, Reg: r66, Reg2: r67, Reg3: r68}
						ctx.BindReg(r66, &d126)
						ctx.BindReg(r67, &d126)
						ctx.BindReg(r68, &d126)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d82)
					ctx.EnsureDesc(&d82)
					var d127 JITValueDesc
					if d82.Loc == LocImm {
						d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d82.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d82.Reg)
						ctx.EmitMovRegReg(scratch, d82.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d127)
					}
					if d127.Loc == LocReg && d82.Loc == LocReg && d127.Reg == d82.Reg {
						ctx.TransferReg(d82.Reg)
						d82.Loc = LocNone
					}
					ctx.FreeDesc(&d82)
					ctx.ReclaimUntrackedRegs()
					stackArray128 = ctx.AllocStack(int32(16))
					_ = stackArray128
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d76)
					ctx.EnsureDesc(&d76)
					ctx.EmitStoreScmerToStack(d76, int32(stackArray128)+int32(0))
					ctx.ReclaimUntrackedRegs()
					d129 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d129
					ctx.ReclaimUntrackedRegs()
					r69 := ctx.AllocReg()
					r70 := ctx.AllocRegExcept(r69)
					r71 := ctx.AllocRegExcept(r69, r70)
					d130 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r69, Reg2: r70, Reg3: r71}
					ctx.BindReg(r69, &d130)
					ctx.BindReg(r70, &d130)
					ctx.BindReg(r71, &d130)
					ctx.BindReg(r69, &d130)
					ctx.BindReg(r70, &d130)
					ctx.BindReg(r71, &d130)
					ctx.EmitLeaRegMem(d130.Reg, ctx.StackReg, int32(stackArray128))
					ctx.EmitMovRegImm64(d130.Reg2, uint64(1))
					ctx.EmitMovRegImm64(d130.Reg3, uint64(1))
					callResults131 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d125, d130}, []uint8{3}, []uint8{1})
					d132 = callResults131[0]
					ctx.ReclaimUntrackedRegs()
					d133 = ctx.EmitNewSliceFromGoSlice(&d132)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d127)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d133)
					d134 = ctx.EmitSliceElementAddress(&d126, &d127, int32(16))
					ctx.EmitStoreScmerAt(&d134, &d133)
					ctx.FreeDesc(&d134)
					ctx.FreeDesc(&d127)
					ctx.FreeDesc(&d133)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl12)
					if r6 {
						ctx.UnprotectReg(r7)
					}
					if r8 {
						ctx.UnprotectReg(r9)
					}
					if r10 {
						ctx.UnprotectReg(r11)
					}
					ctx.FreeDesc(&d56)
					ctx.FreeDesc(&d67)
					if ps.General {
					}
					ps135 := PhiState{General: ps.General}
					ps135.OverlayValues = make([]JITValueDesc, 135)
					ps135.OverlayValues[1] = d1
					ps135.OverlayValues[2] = d2
					ps135.OverlayValues[3] = d3
					ps135.OverlayValues[4] = d4
					ps135.OverlayValues[5] = d5
					ps135.OverlayValues[7] = d7
					ps135.OverlayValues[8] = d8
					ps135.OverlayValues[10] = d10
					ps135.OverlayValues[11] = d11
					ps135.OverlayValues[12] = d12
					ps135.OverlayValues[13] = d13
					ps135.OverlayValues[14] = d14
					ps135.OverlayValues[15] = d15
					ps135.OverlayValues[16] = d16
					ps135.OverlayValues[17] = d17
					ps135.OverlayValues[19] = d19
					ps135.OverlayValues[20] = d20
					ps135.OverlayValues[21] = d21
					ps135.OverlayValues[22] = d22
					ps135.OverlayValues[23] = d23
					ps135.OverlayValues[26] = d26
					ps135.OverlayValues[51] = d51
					ps135.OverlayValues[52] = d52
					ps135.OverlayValues[53] = d53
					ps135.OverlayValues[55] = d55
					ps135.OverlayValues[56] = d56
					ps135.OverlayValues[61] = d61
					ps135.OverlayValues[63] = d63
					ps135.OverlayValues[64] = d64
					ps135.OverlayValues[66] = d66
					ps135.OverlayValues[67] = d67
					ps135.OverlayValues[72] = d72
					ps135.OverlayValues[74] = d74
					ps135.OverlayValues[75] = d75
					ps135.OverlayValues[76] = d76
					ps135.OverlayValues[77] = d77
					ps135.OverlayValues[78] = d78
					ps135.OverlayValues[79] = d79
					ps135.OverlayValues[80] = d80
					ps135.OverlayValues[82] = d82
					ps135.OverlayValues[83] = d83
					ps135.OverlayValues[84] = d84
					ps135.OverlayValues[86] = d86
					ps135.OverlayValues[87] = d87
					ps135.OverlayValues[89] = d89
					ps135.OverlayValues[90] = d90
					ps135.OverlayValues[91] = d91
					ps135.OverlayValues[92] = d92
					ps135.OverlayValues[93] = d93
					ps135.OverlayValues[94] = d94
					ps135.OverlayValues[95] = d95
					ps135.OverlayValues[97] = d97
					ps135.OverlayValues[98] = d98
					ps135.OverlayValues[100] = d100
					ps135.OverlayValues[101] = d101
					ps135.OverlayValues[103] = d103
					ps135.OverlayValues[104] = d104
					ps135.OverlayValues[105] = d105
					ps135.OverlayValues[106] = d106
					ps135.OverlayValues[107] = d107
					ps135.OverlayValues[108] = d108
					ps135.OverlayValues[109] = d109
					ps135.OverlayValues[110] = d110
					ps135.OverlayValues[111] = d111
					ps135.OverlayValues[112] = d112
					ps135.OverlayValues[113] = d113
					ps135.OverlayValues[114] = d114
					ps135.OverlayValues[116] = d116
					ps135.OverlayValues[118] = d118
					ps135.OverlayValues[119] = d119
					ps135.OverlayValues[120] = d120
					ps135.OverlayValues[121] = d121
					ps135.OverlayValues[122] = d122
					ps135.OverlayValues[123] = d123
					ps135.OverlayValues[124] = d124
					ps135.OverlayValues[125] = d125
					ps135.OverlayValues[126] = d126
					ps135.OverlayValues[127] = d127
					ps135.OverlayValues[129] = d129
					ps135.OverlayValues[130] = d130
					ps135.OverlayValues[132] = d132
					ps135.OverlayValues[133] = d133
					ps135.OverlayValues[134] = d134
					ps135.PhiValues = make([]JITValueDesc, 1)
					if ps135.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps135)
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
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
					}
					if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != LocNone {
						d77 = ps.OverlayValues[77]
					}
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
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
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
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
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
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
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
						d114 = ps.OverlayValues[114]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
						d118 = ps.OverlayValues[118]
					}
					if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
						d119 = ps.OverlayValues[119]
					}
					if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != LocNone {
						d120 = ps.OverlayValues[120]
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
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
						d127 = ps.OverlayValues[127]
					}
					if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
						d129 = ps.OverlayValues[129]
					}
					if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
						d130 = ps.OverlayValues[130]
					}
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					ctx.ReclaimUntrackedRegs()
					var d136 JITValueDesc
					if d16.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r72 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r72, makeAux(tagFastDict, 0))
						d136 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d16.Reg, Reg2: r72}
						ctx.BindReg(d16.Reg, &d136)
						ctx.BindReg(r72, &d136)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d136)
						ctx.BindReg(r72, &d136)
						d16.Loc = LocNone
					}
					ctx.FreeDesc(&d16)
					ctx.EnsureDesc(&d136)
					if d136.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d136, &result)
						result.Type = d136.Type
					} else {
						switch d136.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d136)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d136)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d136)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d136, &result)
							result.Type = d136.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps137 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps137)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITInlineCallbacks: true,
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
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_count"].Fn, args, result)
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
				var d59 JITValueDesc
				_ = d59
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
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
				var d70 JITValueDesc
				_ = d70
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var stackArray74 int32
				var d75 JITValueDesc
				_ = d75
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
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d94 JITValueDesc
				_ = d94
				var d96 JITValueDesc
				_ = d96
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
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d109 JITValueDesc
				_ = d109
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
						d5 = ctx.RequestOptimizedCallback(1)
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
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
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
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitJmp(lbl6)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitCmpRegImm32(d10.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl6)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d9)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
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
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d18)
					ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
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
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl11)
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
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d46)
					ctx.EmitStoreScmerToStack(d46, int32(stackArray48)+int32(0))
					ctx.FreeDesc(&d46)
					d49 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d49
					callbackArgs51 := make([]JITValueDesc, 1)
					callbackArgs51[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray48) + 0}
					var d50 JITValueDesc
					callbackResultOff52 = ctx.AllocStack(16)
					ctx.FreeDesc(&d49)
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
					ctx.EnsureDesc(&d50)
					d58 = d50
					_ = d58
					ctx.StabilizeDescForControlFlow(&d58)
					r6 := d13.Loc == LocReg || d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r7 := d13.Reg
					if r6 {
						ctx.ProtectReg(r7)
					}
					r8 := d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r9 := d13.Reg2
					if r8 {
						ctx.ProtectReg(r9)
					}
					r10 := d13.Loc == LocRegTriple
					r11 := d13.Reg3
					if r10 {
						ctx.ProtectReg(r11)
					}
					lbl12 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					bbpos_2_4 := int32(-1)
					_ = bbpos_2_4
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d59 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 24
						r12 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r12, fieldAddr)
						d59 = JITValueDesc{Loc: LocReg, Reg: r12}
						ctx.BindReg(r12, &d59)
					} else {
						off := int32(24)
						baseReg := d13.Reg
						r13 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r13, baseReg, off)
						d59 = JITValueDesc{Loc: LocReg, Reg: r13}
						ctx.BindReg(r13, &d59)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d59)
					var d60 JITValueDesc
					if d59.Loc == LocImm {
						d60 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d59.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d59)
						if d59.Loc != LocReg && d59.Loc != LocRegPair && d59.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r14 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d59.Reg, 0)
						ctx.EmitSetcc(r14, CondEqual)
						d60 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d60)
					}
					ctx.FreeDesc(&d59)
					ctx.ReclaimUntrackedRegs()
					d61 = d60
					ctx.EnsureDesc(&d61)
					if d61.Loc != LocImm && d61.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d61.Loc == LocImm {
						if d61.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						}
					} else {
						ctx.EmitCmpRegImm32(d61.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d60)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d58)
					if d58.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d58.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d58.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d58)
						} else if d58.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d58)
						} else if d58.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d58)
						} else if d58.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d58.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
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
					if d58.Loc != LocRegPair && d58.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (HashKey arg0)")
					}
					ctx.SyncDesc(&d58)
					d62 = ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d58}, 1)
					ctx.BindReg(d62.Reg, &d62)
					ctx.StabilizeDescForControlFlow(&d62)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d58)
					if d58.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d58.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d58.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d58)
						} else if d58.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d58)
						} else if d58.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d58)
						} else if d58.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d58.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
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
					if d58.Loc != LocRegPair && d58.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*FastDict).findPos arg1)")
					}
					ctx.EnsureDesc(&d62)
					ctx.EnsureDesc(&d62)
					if d62.Loc == LocRegPair || d62.Loc == LocStackPair || d62.Loc == LocRegTriple || d62.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d58)
					ctx.SyncDesc(&d62)
					callResults63 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d13, d58, d62}, []uint8{1, 1}, []uint8{0, 0})
					d64 = callResults63[0]
					_ = d64
					d65 = callResults63[1]
					_ = d65
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d64)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d66 = d65
					ctx.EnsureDesc(&d66)
					if d66.Loc != LocImm && d66.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d66.Loc == LocImm {
						if d66.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitCmpRegImm32(d66.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d65)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d67 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d67)
					ctx.EnsureDesc(&d62)
					d68 = d58
					_ = d68
					ctx.StabilizeDescForControlFlow(&d68)
					d69 = d67
					_ = d69
					ctx.StabilizeDescForControlFlow(&d69)
					d70 = d62
					_ = d70
					ctx.StabilizeDescForControlFlow(&d70)
					r15 := d13.Loc == LocReg || d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r16 := d13.Reg
					if r15 {
						ctx.ProtectReg(r16)
					}
					r17 := d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r18 := d13.Reg2
					if r17 {
						ctx.ProtectReg(r18)
					}
					r19 := d13.Loc == LocRegTriple
					r20 := d13.Reg3
					if r19 {
						ctx.ProtectReg(r20)
					}
					r21 := d58.Loc == LocReg || d58.Loc == LocRegPair || d58.Loc == LocRegTriple
					r22 := d58.Reg
					if r21 {
						ctx.ProtectReg(r22)
					}
					r23 := d58.Loc == LocRegPair || d58.Loc == LocRegTriple
					r24 := d58.Reg2
					if r23 {
						ctx.ProtectReg(r24)
					}
					r25 := d58.Loc == LocRegTriple
					r26 := d58.Reg3
					if r25 {
						ctx.ProtectReg(r26)
					}
					lbl21 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					bbpos_3_3 := int32(-1)
					_ = bbpos_3_3
					bbpos_3_4 := int32(-1)
					_ = bbpos_3_4
					bbpos_3_5 := int32(-1)
					_ = bbpos_3_5
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d71 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r27 := ctx.AllocReg()
						r28 := ctx.AllocRegExcept(r27)
						r29 := ctx.AllocRegExcept(r27, r28)
						ctx.EmitMovRegMem64(r27, fieldAddr)
						ctx.EmitMovRegMem64(r28, fieldAddr+8)
						ctx.EmitMovRegMem64(r29, fieldAddr+16)
						d71 = JITValueDesc{Loc: LocRegTriple, Reg: r27, Reg2: r28, Reg3: r29}
						ctx.BindReg(r27, &d71)
						ctx.BindReg(r28, &d71)
						ctx.BindReg(r29, &d71)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r30 := ctx.AllocRegExcept(baseReg)
						r31 := ctx.AllocRegExcept(baseReg, r30)
						r32 := ctx.AllocRegExcept(baseReg, r30, r31)
						ctx.EmitMovRegMem(r30, baseReg, off)
						ctx.EmitMovRegMem(r31, baseReg, off+8)
						ctx.EmitMovRegMem(r32, baseReg, off+16)
						d71 = JITValueDesc{Loc: LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
						ctx.BindReg(r30, &d71)
						ctx.BindReg(r31, &d71)
						ctx.BindReg(r32, &d71)
					}
					ctx.ReclaimUntrackedRegs()
					var d72 JITValueDesc
					if d71.SliceSizeKnown {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d71.KnownSliceLen))}
					} else if d71.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d71.StackOff))}
					} else if d71.Loc == LocStackTriple {
						d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d71.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d71)
						if d71.Loc == LocRegPair || d71.Loc == LocRegTriple {
							d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg2, ID: 0}
						} else if d71.Loc == LocReg {
							d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d72)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d73 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r33 := ctx.AllocReg()
						r34 := ctx.AllocRegExcept(r33)
						r35 := ctx.AllocRegExcept(r33, r34)
						ctx.EmitMovRegMem64(r33, fieldAddr)
						ctx.EmitMovRegMem64(r34, fieldAddr+8)
						ctx.EmitMovRegMem64(r35, fieldAddr+16)
						d73 = JITValueDesc{Loc: LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
						ctx.BindReg(r33, &d73)
						ctx.BindReg(r34, &d73)
						ctx.BindReg(r35, &d73)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r36 := ctx.AllocRegExcept(baseReg)
						r37 := ctx.AllocRegExcept(baseReg, r36)
						r38 := ctx.AllocRegExcept(baseReg, r36, r37)
						ctx.EmitMovRegMem(r36, baseReg, off)
						ctx.EmitMovRegMem(r37, baseReg, off+8)
						ctx.EmitMovRegMem(r38, baseReg, off+16)
						d73 = JITValueDesc{Loc: LocRegTriple, Reg: r36, Reg2: r37, Reg3: r38}
						ctx.BindReg(r36, &d73)
						ctx.BindReg(r37, &d73)
						ctx.BindReg(r38, &d73)
					}
					ctx.ReclaimUntrackedRegs()
					stackArray74 = ctx.AllocStack(int32(32))
					_ = stackArray74
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d68)
					ctx.EnsureDesc(&d68)
					ctx.EmitStoreScmerToStack(d68, int32(stackArray74)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					ctx.EnsureDesc(&d69)
					ctx.EmitStoreTypedScmerToStack(d69, tagInt, int32(stackArray74)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d75 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d75
					ctx.ReclaimUntrackedRegs()
					r39 := ctx.AllocReg()
					r40 := ctx.AllocRegExcept(r39)
					r41 := ctx.AllocRegExcept(r39, r40)
					d76 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r39, Reg2: r40, Reg3: r41}
					ctx.BindReg(r39, &d76)
					ctx.BindReg(r40, &d76)
					ctx.BindReg(r41, &d76)
					ctx.BindReg(r39, &d76)
					ctx.BindReg(r40, &d76)
					ctx.BindReg(r41, &d76)
					ctx.EmitLeaRegMem(d76.Reg, ctx.StackReg, int32(stackArray74))
					ctx.EmitMovRegImm64(d76.Reg2, uint64(2))
					ctx.EmitMovRegImm64(d76.Reg3, uint64(2))
					callResults77 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d73, d76}, []uint8{3}, []uint8{1})
					d78 = callResults77[0]
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d78)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d78)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value []Scmer) { base.Pairs = value }), []JITValueDesc{d13, d78})
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d79 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 24
						r42 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r42, fieldAddr)
						d79 = JITValueDesc{Loc: LocReg, Reg: r42}
						ctx.BindReg(r42, &d79)
					} else {
						off := int32(24)
						baseReg := d13.Reg
						r43 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r43, baseReg, off)
						d79 = JITValueDesc{Loc: LocReg, Reg: r43}
						ctx.BindReg(r43, &d79)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d70)
					lookupResults80 := JITEmitGoCallResults(ctx, GoFuncAddr(func(m map[uint64]int, k uint64) (int, bool) { value, ok := m[k]; return value, ok }), []JITValueDesc{d79, d70}, []uint8{1, 1}, []uint8{0, 0})
					d81 = lookupResults80[0]
					d82 = lookupResults80[1]
					ctx.EmitAndRegImm32(d82.Reg, 1)
					d82.Type = tagBool
					ctx.FreeDesc(&d79)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d83 = d82
					ctx.EnsureDesc(&d83)
					if d83.Loc != LocImm && d83.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d83.Loc == LocImm {
						if d83.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl22)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitCmpRegImm32(d83.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
					ctx.FreeDesc(&d82)
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d84 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 24
						r44 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r44, fieldAddr)
						d84 = JITValueDesc{Loc: LocReg, Reg: r44}
						ctx.BindReg(r44, &d84)
					} else {
						off := int32(24)
						baseReg := d13.Reg
						r45 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r45, baseReg, off)
						d84 = JITValueDesc{Loc: LocReg, Reg: r45}
						ctx.BindReg(r45, &d84)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d84)
					ctx.EnsureDesc(&d70)
					ctx.EnsureDesc(&d72)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64]int, key uint64, value int) { m[key] = value }), []JITValueDesc{d84, d70, d72})
					ctx.FreeDesc(&d84)
					ctx.ReclaimUntrackedRegs()
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl21)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d85 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 32
						r46 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r46, fieldAddr)
						d85 = JITValueDesc{Loc: LocReg, Reg: r46}
						ctx.BindReg(r46, &d85)
					} else {
						off := int32(32)
						baseReg := d13.Reg
						r47 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r47, baseReg, off)
						d85 = JITValueDesc{Loc: LocReg, Reg: r47}
						ctx.BindReg(r47, &d85)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d85)
					var d86 JITValueDesc
					if d85.Loc == LocImm {
						d86 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d85.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d85)
						if d85.Loc != LocReg && d85.Loc != LocRegPair && d85.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r48 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d85.Reg, 0)
						ctx.EmitSetcc(r48, CondEqual)
						d86 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r48}
						ctx.BindReg(r48, &d86)
					}
					ctx.FreeDesc(&d85)
					ctx.ReclaimUntrackedRegs()
					d87 = d86
					ctx.EnsureDesc(&d87)
					if d87.Loc != LocImm && d87.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					if d87.Loc == LocImm {
						if d87.Imm.Bool() {
							ctx.MarkLabel(lbl28)
							ctx.EmitJmp(lbl26)
						} else {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl27)
						}
					} else {
						ctx.EmitCmpRegImm32(d87.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl28)
						ctx.EmitJmp(lbl29)
						ctx.MarkLabel(lbl28)
						ctx.EmitJmp(lbl26)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
					}
					ctx.FreeDesc(&d86)
					bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d88 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 32
						r49 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r49, fieldAddr)
						d88 = JITValueDesc{Loc: LocReg, Reg: r49}
						ctx.BindReg(r49, &d88)
					} else {
						off := int32(32)
						baseReg := d13.Reg
						r50 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r50, baseReg, off)
						d88 = JITValueDesc{Loc: LocReg, Reg: r50}
						ctx.BindReg(r50, &d88)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d89 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 32
						r51 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r51, fieldAddr)
						d89 = JITValueDesc{Loc: LocReg, Reg: r51}
						ctx.BindReg(r51, &d89)
					} else {
						off := int32(32)
						baseReg := d13.Reg
						r52 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r52, baseReg, off)
						d89 = JITValueDesc{Loc: LocReg, Reg: r52}
						ctx.BindReg(r52, &d89)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d70)
					d90 = ctx.EmitGoCallScalar(GoFuncAddr(func(m map[uint64][]int, k uint64) []int { return m[k] }), []JITValueDesc{d89, d70}, 3)
					ctx.FreeDesc(&d89)
					ctx.ReclaimUntrackedRegs()
					d91 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]int { return new([1]int) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d72)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]int, index int, value int) { dst[index] = value }), []JITValueDesc{d91, d92, d72})
					ctx.FreeDesc(&d72)
					ctx.ReclaimUntrackedRegs()
					sliceResults93 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]int) []int { return value[0:1:1] }), []JITValueDesc{d91}, []uint8{3}, []uint8{1})
					d94 = sliceResults93[0]
					ctx.ReclaimUntrackedRegs()
					callResults95 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []int) []int { return append(dst, src...) }), []JITValueDesc{d90, d94}, []uint8{3}, []uint8{1})
					d96 = callResults95[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d70)
					ctx.EnsureDesc(&d96)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64][]int, key uint64, value []int) { m[key] = value }), []JITValueDesc{d88, d70, d96})
					ctx.FreeDesc(&d88)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_3_2)
					bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d97 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64][]int { return make(map[uint64][]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d97)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64][]int) { base.collisions = value }), []JITValueDesc{d13, d97})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl21)
					if r15 {
						ctx.UnprotectReg(r16)
					}
					if r17 {
						ctx.UnprotectReg(r18)
					}
					if r19 {
						ctx.UnprotectReg(r20)
					}
					if r21 {
						ctx.UnprotectReg(r22)
					}
					if r23 {
						ctx.UnprotectReg(r24)
					}
					if r25 {
						ctx.UnprotectReg(r26)
					}
					ctx.FreeDesc(&d62)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d98 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d98)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d98)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d13, d98})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d99 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r53 := ctx.AllocReg()
						r54 := ctx.AllocRegExcept(r53)
						r55 := ctx.AllocRegExcept(r53, r54)
						ctx.EmitMovRegMem64(r53, fieldAddr)
						ctx.EmitMovRegMem64(r54, fieldAddr+8)
						ctx.EmitMovRegMem64(r55, fieldAddr+16)
						d99 = JITValueDesc{Loc: LocRegTriple, Reg: r53, Reg2: r54, Reg3: r55}
						ctx.BindReg(r53, &d99)
						ctx.BindReg(r54, &d99)
						ctx.BindReg(r55, &d99)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r56 := ctx.AllocRegExcept(baseReg)
						r57 := ctx.AllocRegExcept(baseReg, r56)
						r58 := ctx.AllocRegExcept(baseReg, r56, r57)
						ctx.EmitMovRegMem(r56, baseReg, off)
						ctx.EmitMovRegMem(r57, baseReg, off+8)
						ctx.EmitMovRegMem(r58, baseReg, off+16)
						d99 = JITValueDesc{Loc: LocRegTriple, Reg: r56, Reg2: r57, Reg3: r58}
						ctx.BindReg(r56, &d99)
						ctx.BindReg(r57, &d99)
						ctx.BindReg(r58, &d99)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					var d100 JITValueDesc
					if d64.Loc == LocImm {
						d100 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d64.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d64.Reg)
						ctx.EmitMovRegReg(scratch, d64.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d100 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d100)
					}
					if d100.Loc == LocReg && d64.Loc == LocReg && d100.Reg == d64.Reg {
						ctx.TransferReg(d64.Reg)
						d64.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d101 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r59 := ctx.AllocReg()
						r60 := ctx.AllocRegExcept(r59)
						r61 := ctx.AllocRegExcept(r59, r60)
						ctx.EmitMovRegMem64(r59, fieldAddr)
						ctx.EmitMovRegMem64(r60, fieldAddr+8)
						ctx.EmitMovRegMem64(r61, fieldAddr+16)
						d101 = JITValueDesc{Loc: LocRegTriple, Reg: r59, Reg2: r60, Reg3: r61}
						ctx.BindReg(r59, &d101)
						ctx.BindReg(r60, &d101)
						ctx.BindReg(r61, &d101)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r62 := ctx.AllocRegExcept(baseReg)
						r63 := ctx.AllocRegExcept(baseReg, r62)
						r64 := ctx.AllocRegExcept(baseReg, r62, r63)
						ctx.EmitMovRegMem(r62, baseReg, off)
						ctx.EmitMovRegMem(r63, baseReg, off+8)
						ctx.EmitMovRegMem(r64, baseReg, off+16)
						d101 = JITValueDesc{Loc: LocRegTriple, Reg: r62, Reg2: r63, Reg3: r64}
						ctx.BindReg(r62, &d101)
						ctx.BindReg(r63, &d101)
						ctx.BindReg(r64, &d101)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					var d102 JITValueDesc
					if d64.Loc == LocImm {
						d102 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d64.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d64.Reg)
						ctx.EmitMovRegReg(scratch, d64.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d102 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d102)
					}
					if d102.Loc == LocReg && d64.Loc == LocReg && d102.Reg == d64.Reg {
						ctx.TransferReg(d64.Reg)
						d64.Loc = LocNone
					}
					ctx.FreeDesc(&d64)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d102)
					ctx.ReclaimUntrackedRegs()
					d104 = ctx.EmitSliceElementAddress(&d101, &d102, 16)
					ctx.EnsureDesc(&d104)
					r65 := ctx.AllocRegExcept(d104.Reg)
					ctx.EmitMovRegMem(r65, d104.Reg, 8)
					ctx.EmitMovRegMem(d104.Reg, d104.Reg, 0)
					d103 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d104.Reg, Reg2: r65}
					ctx.BindReg(d104.Reg, &d103)
					ctx.BindReg(r65, &d103)
					ctx.FreeDesc(&d102)
					ctx.ReclaimUntrackedRegs()
					var d105 JITValueDesc
					if d103.Loc == LocImm {
						d105 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d103.Imm.Int())}
					} else if d103.Type == tagInt && d103.Loc == LocRegPair {
						ctx.FreeReg(d103.Reg)
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d103.Reg2}
						ctx.BindReg(d103.Reg2, &d105)
						ctx.BindReg(d103.Reg2, &d105)
					} else if d103.Type == tagInt && d103.Loc == LocReg {
						d105 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d103.Reg}
						ctx.BindReg(d103.Reg, &d105)
						ctx.BindReg(d103.Reg, &d105)
					} else {
						d105 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d103}, 1)
						d105.Type = tagInt
						ctx.BindReg(d105.Reg, &d105)
					}
					ctx.FreeDesc(&d103)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d105)
					ctx.EnsureDesc(&d105)
					var d106 JITValueDesc
					if d105.Loc == LocImm {
						d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d105.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d105.Reg)
						ctx.EmitMovRegReg(scratch, d105.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d106)
					}
					if d106.Loc == LocReg && d105.Loc == LocReg && d106.Reg == d105.Reg {
						ctx.TransferReg(d105.Reg)
						d105.Loc = LocNone
					}
					ctx.FreeDesc(&d105)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d100)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					d107 = ctx.EmitSliceElementAddress(&d99, &d100, int32(16))
					ctx.EmitStoreScmerAt(&d107, &d106)
					ctx.FreeDesc(&d107)
					ctx.FreeDesc(&d100)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl12)
					if r6 {
						ctx.UnprotectReg(r7)
					}
					if r8 {
						ctx.UnprotectReg(r9)
					}
					if r10 {
						ctx.UnprotectReg(r11)
					}
					ctx.FreeDesc(&d50)
					if ps.General {
					}
					ps108 := PhiState{General: ps.General}
					ps108.OverlayValues = make([]JITValueDesc, 108)
					ps108.OverlayValues[1] = d1
					ps108.OverlayValues[2] = d2
					ps108.OverlayValues[3] = d3
					ps108.OverlayValues[4] = d4
					ps108.OverlayValues[5] = d5
					ps108.OverlayValues[7] = d7
					ps108.OverlayValues[8] = d8
					ps108.OverlayValues[9] = d9
					ps108.OverlayValues[10] = d10
					ps108.OverlayValues[11] = d11
					ps108.OverlayValues[12] = d12
					ps108.OverlayValues[13] = d13
					ps108.OverlayValues[14] = d14
					ps108.OverlayValues[16] = d16
					ps108.OverlayValues[17] = d17
					ps108.OverlayValues[18] = d18
					ps108.OverlayValues[19] = d19
					ps108.OverlayValues[20] = d20
					ps108.OverlayValues[23] = d23
					ps108.OverlayValues[46] = d46
					ps108.OverlayValues[47] = d47
					ps108.OverlayValues[49] = d49
					ps108.OverlayValues[50] = d50
					ps108.OverlayValues[55] = d55
					ps108.OverlayValues[57] = d57
					ps108.OverlayValues[58] = d58
					ps108.OverlayValues[59] = d59
					ps108.OverlayValues[60] = d60
					ps108.OverlayValues[61] = d61
					ps108.OverlayValues[62] = d62
					ps108.OverlayValues[64] = d64
					ps108.OverlayValues[65] = d65
					ps108.OverlayValues[66] = d66
					ps108.OverlayValues[67] = d67
					ps108.OverlayValues[68] = d68
					ps108.OverlayValues[69] = d69
					ps108.OverlayValues[70] = d70
					ps108.OverlayValues[71] = d71
					ps108.OverlayValues[72] = d72
					ps108.OverlayValues[73] = d73
					ps108.OverlayValues[75] = d75
					ps108.OverlayValues[76] = d76
					ps108.OverlayValues[78] = d78
					ps108.OverlayValues[79] = d79
					ps108.OverlayValues[81] = d81
					ps108.OverlayValues[82] = d82
					ps108.OverlayValues[83] = d83
					ps108.OverlayValues[84] = d84
					ps108.OverlayValues[85] = d85
					ps108.OverlayValues[86] = d86
					ps108.OverlayValues[87] = d87
					ps108.OverlayValues[88] = d88
					ps108.OverlayValues[89] = d89
					ps108.OverlayValues[90] = d90
					ps108.OverlayValues[91] = d91
					ps108.OverlayValues[92] = d92
					ps108.OverlayValues[94] = d94
					ps108.OverlayValues[96] = d96
					ps108.OverlayValues[97] = d97
					ps108.OverlayValues[98] = d98
					ps108.OverlayValues[99] = d99
					ps108.OverlayValues[100] = d100
					ps108.OverlayValues[101] = d101
					ps108.OverlayValues[102] = d102
					ps108.OverlayValues[103] = d103
					ps108.OverlayValues[104] = d104
					ps108.OverlayValues[105] = d105
					ps108.OverlayValues[106] = d106
					ps108.OverlayValues[107] = d107
					ps108.PhiValues = make([]JITValueDesc, 1)
					if ps108.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps108)
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
					if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != LocNone {
						d81 = ps.OverlayValues[81]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
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
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
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
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
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
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 107 && ps.OverlayValues[107].Loc != LocNone {
						d107 = ps.OverlayValues[107]
					}
					ctx.ReclaimUntrackedRegs()
					var d109 JITValueDesc
					if d13.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r66 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r66, makeAux(tagFastDict, 0))
						d109 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d13.Reg, Reg2: r66}
						ctx.BindReg(d13.Reg, &d109)
						ctx.BindReg(r66, &d109)
						ctx.TransferReg(d13.Reg)
						ctx.BindReg(d13.Reg, &d109)
						ctx.BindReg(r66, &d109)
						d13.Loc = LocNone
					}
					ctx.FreeDesc(&d13)
					ctx.EnsureDesc(&d109)
					if d109.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d109, &result)
						result.Type = d109.Type
					} else {
						switch d109.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d109)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d109)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d109)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d109, &result)
							result.Type = d109.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps110 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps110)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITInlineCallbacks: true,
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
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["group_assoc_count_reduce"].Fn, args, result)
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
				var d60 JITValueDesc
				_ = d60
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
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
				var stackArray75 int32
				var d76 JITValueDesc
				_ = d76
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
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
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
				var d106 JITValueDesc
				_ = d106
				var d107 JITValueDesc
				_ = d107
				var d108 JITValueDesc
				_ = d108
				var d110 JITValueDesc
				_ = d110
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
						d5 = ctx.RequestOptimizedCallback(1)
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
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
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
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					if d10.Loc == LocImm {
						if d10.Imm.Bool() {
							ctx.MarkLabel(lbl8)
							ctx.EmitJmp(lbl6)
						} else {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						}
					} else {
						ctx.EmitCmpRegImm32(d10.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl8)
						ctx.EmitJmp(lbl9)
						ctx.MarkLabel(lbl8)
						ctx.EmitJmp(lbl6)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
					}
					ctx.FreeDesc(&d9)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
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
					ctx.MarkLabel(lbl6)
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
					ctx.EnsureDesc(&d18)
					ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
					ctx.EnsureDesc(&d18)
					ctx.EnsureDesc(&d14)
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
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl10)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl11)
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
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					ctx.EmitStoreScmerToStack(d48, int32(stackArray49)+int32(0))
					ctx.FreeDesc(&d48)
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d46)
					ctx.EmitStoreScmerToStack(d46, int32(stackArray49)+int32(16))
					ctx.FreeDesc(&d46)
					d50 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d50
					callbackArgs52 := make([]JITValueDesc, 2)
					callbackArgs52[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray49) + 0}
					callbackArgs52[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray49) + 16}
					var d51 JITValueDesc
					callbackResultOff53 = ctx.AllocStack(16)
					ctx.FreeDesc(&d50)
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
					ctx.EnsureDesc(&d51)
					d59 = d51
					_ = d59
					ctx.StabilizeDescForControlFlow(&d59)
					r6 := d13.Loc == LocReg || d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r7 := d13.Reg
					if r6 {
						ctx.ProtectReg(r7)
					}
					r8 := d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r9 := d13.Reg2
					if r8 {
						ctx.ProtectReg(r9)
					}
					r10 := d13.Loc == LocRegTriple
					r11 := d13.Reg3
					if r10 {
						ctx.ProtectReg(r11)
					}
					lbl12 := ctx.ReserveLabel()
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_1 := int32(-1)
					_ = bbpos_2_1
					bbpos_2_2 := int32(-1)
					_ = bbpos_2_2
					bbpos_2_3 := int32(-1)
					_ = bbpos_2_3
					bbpos_2_4 := int32(-1)
					_ = bbpos_2_4
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d60 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 24
						r12 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r12, fieldAddr)
						d60 = JITValueDesc{Loc: LocReg, Reg: r12}
						ctx.BindReg(r12, &d60)
					} else {
						off := int32(24)
						baseReg := d13.Reg
						r13 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r13, baseReg, off)
						d60 = JITValueDesc{Loc: LocReg, Reg: r13}
						ctx.BindReg(r13, &d60)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d60)
					var d61 JITValueDesc
					if d60.Loc == LocImm {
						d61 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d60.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d60)
						if d60.Loc != LocReg && d60.Loc != LocRegPair && d60.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r14 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d60.Reg, 0)
						ctx.EmitSetcc(r14, CondEqual)
						d61 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d61)
					}
					ctx.FreeDesc(&d60)
					ctx.ReclaimUntrackedRegs()
					d62 = d61
					ctx.EnsureDesc(&d62)
					if d62.Loc != LocImm && d62.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					if d62.Loc == LocImm {
						if d62.Imm.Bool() {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						}
					} else {
						ctx.EmitCmpRegImm32(d62.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl15)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
					}
					ctx.FreeDesc(&d61)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					if d59.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d59.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d59.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d59)
						} else if d59.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d59)
						} else if d59.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d59)
						} else if d59.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d59.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d59 = tmpPair
					} else if d59.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d59.Type, Reg: ctx.AllocRegExcept(d59.Reg), Reg2: ctx.AllocRegExcept(d59.Reg)}
						switch d59.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d59)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d59)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d59)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d59)
						d59 = tmpPair
					}
					if d59.Loc != LocRegPair && d59.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (HashKey arg0)")
					}
					ctx.SyncDesc(&d59)
					d63 = ctx.EmitGoCallScalar(GoFuncAddr(HashKey), []JITValueDesc{d59}, 1)
					ctx.BindReg(d63.Reg, &d63)
					ctx.StabilizeDescForControlFlow(&d63)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair || d13.Loc == LocStackPair || d13.Loc == LocRegTriple || d13.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					if d59.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d59.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d59.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d59)
						} else if d59.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d59)
						} else if d59.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d59)
						} else if d59.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d59.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d59 = tmpPair
					} else if d59.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d59.Type, Reg: ctx.AllocRegExcept(d59.Reg), Reg2: ctx.AllocRegExcept(d59.Reg)}
						switch d59.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d59)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d59)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d59)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d59)
						d59 = tmpPair
					}
					if d59.Loc != LocRegPair && d59.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((*FastDict).findPos arg1)")
					}
					ctx.EnsureDesc(&d63)
					ctx.EnsureDesc(&d63)
					if d63.Loc == LocRegPair || d63.Loc == LocStackPair || d63.Loc == LocRegTriple || d63.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d59)
					ctx.SyncDesc(&d63)
					callResults64 := JITEmitGoCallResults(ctx, GoFuncAddr((*FastDict).findPos), []JITValueDesc{d13, d59, d63}, []uint8{1, 1}, []uint8{0, 0})
					d65 = callResults64[0]
					_ = d65
					d66 = callResults64[1]
					_ = d66
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d65)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d67 = d66
					ctx.EnsureDesc(&d67)
					if d67.Loc != LocImm && d67.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					if d67.Loc == LocImm {
						if d67.Imm.Bool() {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						} else {
							ctx.MarkLabel(lbl20)
							ctx.EmitJmp(lbl18)
						}
					} else {
						ctx.EmitCmpRegImm32(d67.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl19)
						ctx.EmitJmp(lbl20)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl18)
					}
					ctx.FreeDesc(&d66)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl18)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d68)
					ctx.EnsureDesc(&d63)
					d69 = d59
					_ = d69
					ctx.StabilizeDescForControlFlow(&d69)
					d70 = d68
					_ = d70
					ctx.StabilizeDescForControlFlow(&d70)
					d71 = d63
					_ = d71
					ctx.StabilizeDescForControlFlow(&d71)
					r15 := d13.Loc == LocReg || d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r16 := d13.Reg
					if r15 {
						ctx.ProtectReg(r16)
					}
					r17 := d13.Loc == LocRegPair || d13.Loc == LocRegTriple
					r18 := d13.Reg2
					if r17 {
						ctx.ProtectReg(r18)
					}
					r19 := d13.Loc == LocRegTriple
					r20 := d13.Reg3
					if r19 {
						ctx.ProtectReg(r20)
					}
					r21 := d59.Loc == LocReg || d59.Loc == LocRegPair || d59.Loc == LocRegTriple
					r22 := d59.Reg
					if r21 {
						ctx.ProtectReg(r22)
					}
					r23 := d59.Loc == LocRegPair || d59.Loc == LocRegTriple
					r24 := d59.Reg2
					if r23 {
						ctx.ProtectReg(r24)
					}
					r25 := d59.Loc == LocRegTriple
					r26 := d59.Reg3
					if r25 {
						ctx.ProtectReg(r26)
					}
					lbl21 := ctx.ReserveLabel()
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_1 := int32(-1)
					_ = bbpos_3_1
					bbpos_3_2 := int32(-1)
					_ = bbpos_3_2
					bbpos_3_3 := int32(-1)
					_ = bbpos_3_3
					bbpos_3_4 := int32(-1)
					_ = bbpos_3_4
					bbpos_3_5 := int32(-1)
					_ = bbpos_3_5
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d72 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r27 := ctx.AllocReg()
						r28 := ctx.AllocRegExcept(r27)
						r29 := ctx.AllocRegExcept(r27, r28)
						ctx.EmitMovRegMem64(r27, fieldAddr)
						ctx.EmitMovRegMem64(r28, fieldAddr+8)
						ctx.EmitMovRegMem64(r29, fieldAddr+16)
						d72 = JITValueDesc{Loc: LocRegTriple, Reg: r27, Reg2: r28, Reg3: r29}
						ctx.BindReg(r27, &d72)
						ctx.BindReg(r28, &d72)
						ctx.BindReg(r29, &d72)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r30 := ctx.AllocRegExcept(baseReg)
						r31 := ctx.AllocRegExcept(baseReg, r30)
						r32 := ctx.AllocRegExcept(baseReg, r30, r31)
						ctx.EmitMovRegMem(r30, baseReg, off)
						ctx.EmitMovRegMem(r31, baseReg, off+8)
						ctx.EmitMovRegMem(r32, baseReg, off+16)
						d72 = JITValueDesc{Loc: LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
						ctx.BindReg(r30, &d72)
						ctx.BindReg(r31, &d72)
						ctx.BindReg(r32, &d72)
					}
					ctx.ReclaimUntrackedRegs()
					var d73 JITValueDesc
					if d72.SliceSizeKnown {
						d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d72.KnownSliceLen))}
					} else if d72.Loc == LocImm {
						d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d72.StackOff))}
					} else if d72.Loc == LocStackTriple {
						d73 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d72.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d72)
						if d72.Loc == LocRegPair || d72.Loc == LocRegTriple {
							d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg2, ID: 0}
						} else if d72.Loc == LocReg {
							d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d72.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d73)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d74 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r33 := ctx.AllocReg()
						r34 := ctx.AllocRegExcept(r33)
						r35 := ctx.AllocRegExcept(r33, r34)
						ctx.EmitMovRegMem64(r33, fieldAddr)
						ctx.EmitMovRegMem64(r34, fieldAddr+8)
						ctx.EmitMovRegMem64(r35, fieldAddr+16)
						d74 = JITValueDesc{Loc: LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
						ctx.BindReg(r33, &d74)
						ctx.BindReg(r34, &d74)
						ctx.BindReg(r35, &d74)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r36 := ctx.AllocRegExcept(baseReg)
						r37 := ctx.AllocRegExcept(baseReg, r36)
						r38 := ctx.AllocRegExcept(baseReg, r36, r37)
						ctx.EmitMovRegMem(r36, baseReg, off)
						ctx.EmitMovRegMem(r37, baseReg, off+8)
						ctx.EmitMovRegMem(r38, baseReg, off+16)
						d74 = JITValueDesc{Loc: LocRegTriple, Reg: r36, Reg2: r37, Reg3: r38}
						ctx.BindReg(r36, &d74)
						ctx.BindReg(r37, &d74)
						ctx.BindReg(r38, &d74)
					}
					ctx.ReclaimUntrackedRegs()
					stackArray75 = ctx.AllocStack(int32(32))
					_ = stackArray75
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d69)
					ctx.EnsureDesc(&d69)
					ctx.EmitStoreScmerToStack(d69, int32(stackArray75)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d70)
					ctx.EnsureDesc(&d70)
					ctx.EmitStoreTypedScmerToStack(d70, tagInt, int32(stackArray75)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d76 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d76
					ctx.ReclaimUntrackedRegs()
					r39 := ctx.AllocReg()
					r40 := ctx.AllocRegExcept(r39)
					r41 := ctx.AllocRegExcept(r39, r40)
					d77 = JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r39, Reg2: r40, Reg3: r41}
					ctx.BindReg(r39, &d77)
					ctx.BindReg(r40, &d77)
					ctx.BindReg(r41, &d77)
					ctx.BindReg(r39, &d77)
					ctx.BindReg(r40, &d77)
					ctx.BindReg(r41, &d77)
					ctx.EmitLeaRegMem(d77.Reg, ctx.StackReg, int32(stackArray75))
					ctx.EmitMovRegImm64(d77.Reg2, uint64(2))
					ctx.EmitMovRegImm64(d77.Reg3, uint64(2))
					callResults78 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d74, d77}, []uint8{3}, []uint8{1})
					d79 = callResults78[0]
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d79)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d79)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value []Scmer) { base.Pairs = value }), []JITValueDesc{d13, d79})
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d80 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 24
						r42 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r42, fieldAddr)
						d80 = JITValueDesc{Loc: LocReg, Reg: r42}
						ctx.BindReg(r42, &d80)
					} else {
						off := int32(24)
						baseReg := d13.Reg
						r43 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r43, baseReg, off)
						d80 = JITValueDesc{Loc: LocReg, Reg: r43}
						ctx.BindReg(r43, &d80)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d80)
					ctx.EnsureDesc(&d71)
					lookupResults81 := JITEmitGoCallResults(ctx, GoFuncAddr(func(m map[uint64]int, k uint64) (int, bool) { value, ok := m[k]; return value, ok }), []JITValueDesc{d80, d71}, []uint8{1, 1}, []uint8{0, 0})
					d82 = lookupResults81[0]
					d83 = lookupResults81[1]
					ctx.EmitAndRegImm32(d83.Reg, 1)
					d83.Type = tagBool
					ctx.FreeDesc(&d80)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d84 = d83
					ctx.EnsureDesc(&d84)
					if d84.Loc != LocImm && d84.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					if d84.Loc == LocImm {
						if d84.Imm.Bool() {
							ctx.MarkLabel(lbl24)
							ctx.EmitJmp(lbl22)
						} else {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						}
					} else {
						ctx.EmitCmpRegImm32(d84.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl24)
						ctx.EmitJmp(lbl25)
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl22)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
					}
					ctx.FreeDesc(&d83)
					bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d85 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 24
						r44 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r44, fieldAddr)
						d85 = JITValueDesc{Loc: LocReg, Reg: r44}
						ctx.BindReg(r44, &d85)
					} else {
						off := int32(24)
						baseReg := d13.Reg
						r45 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r45, baseReg, off)
						d85 = JITValueDesc{Loc: LocReg, Reg: r45}
						ctx.BindReg(r45, &d85)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d85)
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d73)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64]int, key uint64, value int) { m[key] = value }), []JITValueDesc{d85, d71, d73})
					ctx.FreeDesc(&d85)
					ctx.ReclaimUntrackedRegs()
					bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl21)
					bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl22)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d86 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 32
						r46 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r46, fieldAddr)
						d86 = JITValueDesc{Loc: LocReg, Reg: r46}
						ctx.BindReg(r46, &d86)
					} else {
						off := int32(32)
						baseReg := d13.Reg
						r47 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r47, baseReg, off)
						d86 = JITValueDesc{Loc: LocReg, Reg: r47}
						ctx.BindReg(r47, &d86)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d86)
					var d87 JITValueDesc
					if d86.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d86.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d86)
						if d86.Loc != LocReg && d86.Loc != LocRegPair && d86.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r48 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d86.Reg, 0)
						ctx.EmitSetcc(r48, CondEqual)
						d87 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r48}
						ctx.BindReg(r48, &d87)
					}
					ctx.FreeDesc(&d86)
					ctx.ReclaimUntrackedRegs()
					d88 = d87
					ctx.EnsureDesc(&d88)
					if d88.Loc != LocImm && d88.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl26 := ctx.ReserveLabel()
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					if d88.Loc == LocImm {
						if d88.Imm.Bool() {
							ctx.MarkLabel(lbl28)
							ctx.EmitJmp(lbl26)
						} else {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl27)
						}
					} else {
						ctx.EmitCmpRegImm32(d88.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl28)
						ctx.EmitJmp(lbl29)
						ctx.MarkLabel(lbl28)
						ctx.EmitJmp(lbl26)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
					}
					ctx.FreeDesc(&d87)
					bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d89 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 32
						r49 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r49, fieldAddr)
						d89 = JITValueDesc{Loc: LocReg, Reg: r49}
						ctx.BindReg(r49, &d89)
					} else {
						off := int32(32)
						baseReg := d13.Reg
						r50 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r50, baseReg, off)
						d89 = JITValueDesc{Loc: LocReg, Reg: r50}
						ctx.BindReg(r50, &d89)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d90 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 32
						r51 := ctx.AllocReg()
						ctx.EmitMovRegMem64(r51, fieldAddr)
						d90 = JITValueDesc{Loc: LocReg, Reg: r51}
						ctx.BindReg(r51, &d90)
					} else {
						off := int32(32)
						baseReg := d13.Reg
						r52 := ctx.AllocRegExcept(baseReg)
						ctx.EmitMovRegMem(r52, baseReg, off)
						d90 = JITValueDesc{Loc: LocReg, Reg: r52}
						ctx.BindReg(r52, &d90)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d90)
					ctx.EnsureDesc(&d71)
					d91 = ctx.EmitGoCallScalar(GoFuncAddr(func(m map[uint64][]int, k uint64) []int { return m[k] }), []JITValueDesc{d90, d71}, 3)
					ctx.FreeDesc(&d90)
					ctx.ReclaimUntrackedRegs()
					d92 = ctx.EmitGoCallScalar(GoFuncAddr(func() *[1]int { return new([1]int) }), nil, 1)
					ctx.ReclaimUntrackedRegs()
					d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d73)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *[1]int, index int, value int) { dst[index] = value }), []JITValueDesc{d92, d93, d73})
					ctx.FreeDesc(&d73)
					ctx.ReclaimUntrackedRegs()
					sliceResults94 := JITEmitGoCallResults(ctx, GoFuncAddr(func(value *[1]int) []int { return value[0:1:1] }), []JITValueDesc{d92}, []uint8{3}, []uint8{1})
					d95 = sliceResults94[0]
					ctx.ReclaimUntrackedRegs()
					callResults96 := JITEmitGoCallResults(ctx, GoFuncAddr(func(dst, src []int) []int { return append(dst, src...) }), []JITValueDesc{d91, d95}, []uint8{3}, []uint8{1})
					d97 = callResults96[0]
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d97)
					ctx.EmitGoCallVoid(GoFuncAddr(func(m map[uint64][]int, key uint64, value []int) { m[key] = value }), []JITValueDesc{d89, d71, d97})
					ctx.FreeDesc(&d89)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_3_2)
					bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl26)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d98 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64][]int { return make(map[uint64][]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d98)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d98)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64][]int) { base.collisions = value }), []JITValueDesc{d13, d98})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl27)
					ctx.MarkLabel(lbl21)
					if r15 {
						ctx.UnprotectReg(r16)
					}
					if r17 {
						ctx.UnprotectReg(r18)
					}
					if r19 {
						ctx.UnprotectReg(r20)
					}
					if r21 {
						ctx.UnprotectReg(r22)
					}
					if r23 {
						ctx.UnprotectReg(r24)
					}
					if r25 {
						ctx.UnprotectReg(r26)
					}
					ctx.FreeDesc(&d63)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d99 = ctx.EmitGoCallScalar(GoFuncAddr(func(size int) map[uint64]int { return make(map[uint64]int, size) }), []JITValueDesc{JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0), NoHeapPointer: true}}, 1)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d99)
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d99)
					ctx.EmitGoCallVoid(GoFuncAddr(func(base *FastDict, value map[uint64]int) { base.index = value }), []JITValueDesc{d13, d99})
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl14)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d100 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r53 := ctx.AllocReg()
						r54 := ctx.AllocRegExcept(r53)
						r55 := ctx.AllocRegExcept(r53, r54)
						ctx.EmitMovRegMem64(r53, fieldAddr)
						ctx.EmitMovRegMem64(r54, fieldAddr+8)
						ctx.EmitMovRegMem64(r55, fieldAddr+16)
						d100 = JITValueDesc{Loc: LocRegTriple, Reg: r53, Reg2: r54, Reg3: r55}
						ctx.BindReg(r53, &d100)
						ctx.BindReg(r54, &d100)
						ctx.BindReg(r55, &d100)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r56 := ctx.AllocRegExcept(baseReg)
						r57 := ctx.AllocRegExcept(baseReg, r56)
						r58 := ctx.AllocRegExcept(baseReg, r56, r57)
						ctx.EmitMovRegMem(r56, baseReg, off)
						ctx.EmitMovRegMem(r57, baseReg, off+8)
						ctx.EmitMovRegMem(r58, baseReg, off+16)
						d100 = JITValueDesc{Loc: LocRegTriple, Reg: r56, Reg2: r57, Reg3: r58}
						ctx.BindReg(r56, &d100)
						ctx.BindReg(r57, &d100)
						ctx.BindReg(r58, &d100)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					var d101 JITValueDesc
					if d65.Loc == LocImm {
						d101 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d65.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegReg(scratch, d65.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d101 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d101)
					}
					if d101.Loc == LocReg && d65.Loc == LocReg && d101.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d102 JITValueDesc
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						fieldAddr := uintptr(d13.Imm.Int()) + 0
						r59 := ctx.AllocReg()
						r60 := ctx.AllocRegExcept(r59)
						r61 := ctx.AllocRegExcept(r59, r60)
						ctx.EmitMovRegMem64(r59, fieldAddr)
						ctx.EmitMovRegMem64(r60, fieldAddr+8)
						ctx.EmitMovRegMem64(r61, fieldAddr+16)
						d102 = JITValueDesc{Loc: LocRegTriple, Reg: r59, Reg2: r60, Reg3: r61}
						ctx.BindReg(r59, &d102)
						ctx.BindReg(r60, &d102)
						ctx.BindReg(r61, &d102)
					} else {
						off := int32(0)
						baseReg := d13.Reg
						r62 := ctx.AllocRegExcept(baseReg)
						r63 := ctx.AllocRegExcept(baseReg, r62)
						r64 := ctx.AllocRegExcept(baseReg, r62, r63)
						ctx.EmitMovRegMem(r62, baseReg, off)
						ctx.EmitMovRegMem(r63, baseReg, off+8)
						ctx.EmitMovRegMem(r64, baseReg, off+16)
						d102 = JITValueDesc{Loc: LocRegTriple, Reg: r62, Reg2: r63, Reg3: r64}
						ctx.BindReg(r62, &d102)
						ctx.BindReg(r63, &d102)
						ctx.BindReg(r64, &d102)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					var d103 JITValueDesc
					if d65.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d65.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d65.Reg)
						ctx.EmitMovRegReg(scratch, d65.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d103)
					}
					if d103.Loc == LocReg && d65.Loc == LocReg && d103.Reg == d65.Reg {
						ctx.TransferReg(d65.Reg)
						d65.Loc = LocNone
					}
					ctx.FreeDesc(&d65)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d103)
					ctx.ReclaimUntrackedRegs()
					d105 = ctx.EmitSliceElementAddress(&d102, &d103, 16)
					ctx.EnsureDesc(&d105)
					r65 := ctx.AllocRegExcept(d105.Reg)
					ctx.EmitMovRegMem(r65, d105.Reg, 8)
					ctx.EmitMovRegMem(d105.Reg, d105.Reg, 0)
					d104 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d105.Reg, Reg2: r65}
					ctx.BindReg(d105.Reg, &d104)
					ctx.BindReg(r65, &d104)
					ctx.FreeDesc(&d103)
					ctx.ReclaimUntrackedRegs()
					var d106 JITValueDesc
					if d104.Loc == LocImm {
						d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d104.Imm.Int())}
					} else if d104.Type == tagInt && d104.Loc == LocRegPair {
						ctx.FreeReg(d104.Reg)
						d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d104.Reg2}
						ctx.BindReg(d104.Reg2, &d106)
						ctx.BindReg(d104.Reg2, &d106)
					} else if d104.Type == tagInt && d104.Loc == LocReg {
						d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d104.Reg}
						ctx.BindReg(d104.Reg, &d106)
						ctx.BindReg(d104.Reg, &d106)
					} else {
						d106 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d104}, 1)
						d106.Type = tagInt
						ctx.BindReg(d106.Reg, &d106)
					}
					ctx.FreeDesc(&d104)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d106)
					var d107 JITValueDesc
					if d106.Loc == LocImm {
						d107 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d106.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d106.Reg)
						ctx.EmitMovRegReg(scratch, d106.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d107 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d107)
					}
					if d107.Loc == LocReg && d106.Loc == LocReg && d107.Reg == d106.Reg {
						ctx.TransferReg(d106.Reg)
						d106.Loc = LocNone
					}
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d107)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d101)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d107)
					d108 = ctx.EmitSliceElementAddress(&d100, &d101, int32(16))
					ctx.EmitStoreScmerAt(&d108, &d107)
					ctx.FreeDesc(&d108)
					ctx.FreeDesc(&d101)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl12)
					if r6 {
						ctx.UnprotectReg(r7)
					}
					if r8 {
						ctx.UnprotectReg(r9)
					}
					if r10 {
						ctx.UnprotectReg(r11)
					}
					ctx.FreeDesc(&d51)
					if ps.General {
					}
					ps109 := PhiState{General: ps.General}
					ps109.OverlayValues = make([]JITValueDesc, 109)
					ps109.OverlayValues[1] = d1
					ps109.OverlayValues[2] = d2
					ps109.OverlayValues[3] = d3
					ps109.OverlayValues[4] = d4
					ps109.OverlayValues[5] = d5
					ps109.OverlayValues[7] = d7
					ps109.OverlayValues[8] = d8
					ps109.OverlayValues[9] = d9
					ps109.OverlayValues[10] = d10
					ps109.OverlayValues[11] = d11
					ps109.OverlayValues[12] = d12
					ps109.OverlayValues[13] = d13
					ps109.OverlayValues[14] = d14
					ps109.OverlayValues[16] = d16
					ps109.OverlayValues[17] = d17
					ps109.OverlayValues[18] = d18
					ps109.OverlayValues[19] = d19
					ps109.OverlayValues[20] = d20
					ps109.OverlayValues[23] = d23
					ps109.OverlayValues[46] = d46
					ps109.OverlayValues[47] = d47
					ps109.OverlayValues[48] = d48
					ps109.OverlayValues[50] = d50
					ps109.OverlayValues[51] = d51
					ps109.OverlayValues[56] = d56
					ps109.OverlayValues[58] = d58
					ps109.OverlayValues[59] = d59
					ps109.OverlayValues[60] = d60
					ps109.OverlayValues[61] = d61
					ps109.OverlayValues[62] = d62
					ps109.OverlayValues[63] = d63
					ps109.OverlayValues[65] = d65
					ps109.OverlayValues[66] = d66
					ps109.OverlayValues[67] = d67
					ps109.OverlayValues[68] = d68
					ps109.OverlayValues[69] = d69
					ps109.OverlayValues[70] = d70
					ps109.OverlayValues[71] = d71
					ps109.OverlayValues[72] = d72
					ps109.OverlayValues[73] = d73
					ps109.OverlayValues[74] = d74
					ps109.OverlayValues[76] = d76
					ps109.OverlayValues[77] = d77
					ps109.OverlayValues[79] = d79
					ps109.OverlayValues[80] = d80
					ps109.OverlayValues[82] = d82
					ps109.OverlayValues[83] = d83
					ps109.OverlayValues[84] = d84
					ps109.OverlayValues[85] = d85
					ps109.OverlayValues[86] = d86
					ps109.OverlayValues[87] = d87
					ps109.OverlayValues[88] = d88
					ps109.OverlayValues[89] = d89
					ps109.OverlayValues[90] = d90
					ps109.OverlayValues[91] = d91
					ps109.OverlayValues[92] = d92
					ps109.OverlayValues[93] = d93
					ps109.OverlayValues[95] = d95
					ps109.OverlayValues[97] = d97
					ps109.OverlayValues[98] = d98
					ps109.OverlayValues[99] = d99
					ps109.OverlayValues[100] = d100
					ps109.OverlayValues[101] = d101
					ps109.OverlayValues[102] = d102
					ps109.OverlayValues[103] = d103
					ps109.OverlayValues[104] = d104
					ps109.OverlayValues[105] = d105
					ps109.OverlayValues[106] = d106
					ps109.OverlayValues[107] = d107
					ps109.OverlayValues[108] = d108
					ps109.PhiValues = make([]JITValueDesc, 1)
					if ps109.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps109)
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
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
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
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
						d89 = ps.OverlayValues[89]
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
					var d110 JITValueDesc
					if d13.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r66 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r66, makeAux(tagFastDict, 0))
						d110 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d13.Reg, Reg2: r66}
						ctx.BindReg(d13.Reg, &d110)
						ctx.BindReg(r66, &d110)
						ctx.TransferReg(d13.Reg)
						ctx.BindReg(d13.Reg, &d110)
						ctx.BindReg(r66, &d110)
						d13.Loc = LocNone
					}
					ctx.FreeDesc(&d13)
					ctx.EnsureDesc(&d110)
					if d110.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d110, &result)
						result.Type = d110.Type
					} else {
						switch d110.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d110)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d110)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d110)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d110, &result)
							result.Type = d110.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps111 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps111)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITInlineCallbacks: true,
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
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["mapkey_assoc"].Fn, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
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
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var phiBase39 int32
				_ = phiBase39
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
				var d45 JITValueDesc
				_ = d45
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
				var d53 JITValueDesc
				_ = d53
				var d54 JITValueDesc
				_ = d54
				var d55 JITValueDesc
				_ = d55
				var d56 JITValueDesc
				_ = d56
				var stackArray57 int32
				var d58 JITValueDesc
				_ = d58
				var d59 JITValueDesc
				_ = d59
				var callbackResultOff61 int32
				var d64 JITValueDesc
				_ = d64
				var d66 JITValueDesc
				_ = d66
				var stackArray67 int32
				var d68 JITValueDesc
				_ = d68
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
				var d79 JITValueDesc
				_ = d79
				var d80 JITValueDesc
				_ = d80
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d88 JITValueDesc
				_ = d88
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
				var stackArray149 int32
				var d150 JITValueDesc
				_ = d150
				var d151 JITValueDesc
				_ = d151
				var callbackResultOff153 int32
				var d156 JITValueDesc
				_ = d156
				var d158 JITValueDesc
				_ = d158
				var d159 JITValueDesc
				_ = d159
				var d160 JITValueDesc
				_ = d160
				var d161 JITValueDesc
				_ = d161
				var stackArray162 int32
				var d163 JITValueDesc
				_ = d163
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var callbackResultOff167 int32
				var d170 JITValueDesc
				_ = d170
				var d172 JITValueDesc
				_ = d172
				var d173 JITValueDesc
				_ = d173
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [6]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
				bbs[4].PhiCount = uint16(1)
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
					d2 = args[1]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Loc == LocLambdaTemplate {
						d3 = d2
					} else if d2.Loc == LocImm {
						optimizedCallback4 := NewFunc(OptimizeProcToSerialFunction(d2.Imm))
						ctx.TrackImm(optimizedCallback4)
						d3 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback4, Rooted: true}
					} else {
						d3 = ctx.RequestOptimizedCallback(1)
					}
					ctx.FreeDesc(&d2)
					ctx.SyncDesc(&d3)
					globalLookup5 := Globalenv.Vars[Symbol("set_assoc")]
					ctx.TrackImm(globalLookup5)
					d6 = JITValueDesc{Loc: LocImm, Type: globalLookup5.GetTag(), Imm: globalLookup5, Rooted: true}
					optimizedCallback8 := NewFunc(OptimizeProcToSerialFunction(d6.Imm))
					ctx.TrackImm(optimizedCallback8)
					d7 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback8, Rooted: true}
					ctx.SyncDesc(&d7)
					r0 := ctx.AllocReg()
					r1 := ctx.AllocRegExcept(r0)
					ctx.EmitMovRegImm64(r0, 0)
					ctx.EmitMovRegImm64(r1, 0)
					d9 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r0, Reg2: r1}
					ctx.BindReg(r0, &d9)
					ctx.BindReg(r1, &d9)
					ctx.StabilizeDescForControlFlow(&d9)
					d10 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: nil}
					ctx.SyncDesc(&d10)
					d11 = args[0]
					d11.ID = 0
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d11.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d11)
						} else if d11.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d11)
						} else if d11.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d11)
						} else if d11.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d11.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d11 = tmpPair
					} else if d11.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocRegExcept(d11.Reg), Reg2: ctx.AllocRegExcept(d11.Reg)}
						switch d11.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d11)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d11)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d11)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d11)
						d11 = tmpPair
					}
					if d11.Loc != LocRegPair && d11.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (asAssoc arg0)")
					}
					d12 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("mapkey_assoc")}
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d12.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d12.Imm)
						ptrWord, _ := d12.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d12.Imm.String())))
						d12 = tmpPair
					} else if d12.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d12.Type, Reg: ctx.AllocRegExcept(d12.Reg), Reg2: ctx.AllocRegExcept(d12.Reg)}
						switch d12.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d12)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d12)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d12)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d12)
						d12 = tmpPair
					}
					if d12.Loc != LocRegPair && d12.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (asAssoc arg1)")
					}
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d12)
					callResults13 := JITEmitGoCallResults(ctx, GoFuncAddr(asAssoc), []JITValueDesc{d11, d12}, []uint8{3, 1}, []uint8{1, 1})
					ctx.FreeDesc(&d12)
					d14 = callResults13[0]
					_ = d14
					d15 = callResults13[1]
					_ = d15
					ctx.FreeDesc(&d11)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.EnsureDesc(&d15)
					var d16 JITValueDesc
					if d15.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d15.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d15)
						if d15.Loc != LocReg && d15.Loc != LocRegPair && d15.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r2 := ctx.AllocRegExcept(d15.Reg)
						ctx.EmitCmpRegImm32(d15.Reg, 0)
						ctx.EmitSetcc(r2, CondEqual)
						d16 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
						ctx.BindReg(r2, &d16)
					}
					d17 = d16
					ctx.EnsureDesc(&d17)
					if d17.Loc != LocImm && d17.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d17.Loc == LocImm {
						if d17.Imm.Bool() {
							if ps.General {
							}
							ps18 := PhiState{General: ps.General}
							ps18.OverlayValues = make([]JITValueDesc, 18)
							ps18.OverlayValues[1] = d1
							ps18.OverlayValues[2] = d2
							ps18.OverlayValues[3] = d3
							ps18.OverlayValues[6] = d6
							ps18.OverlayValues[7] = d7
							ps18.OverlayValues[9] = d9
							ps18.OverlayValues[10] = d10
							ps18.OverlayValues[11] = d11
							ps18.OverlayValues[12] = d12
							ps18.OverlayValues[14] = d14
							ps18.OverlayValues[15] = d15
							ps18.OverlayValues[16] = d16
							ps18.OverlayValues[17] = d17
							return bbs[1].RenderPS(ps18)
						}
						if ps.General {
						}
						ps19 := PhiState{General: ps.General}
						ps19.OverlayValues = make([]JITValueDesc, 18)
						ps19.OverlayValues[1] = d1
						ps19.OverlayValues[2] = d2
						ps19.OverlayValues[3] = d3
						ps19.OverlayValues[6] = d6
						ps19.OverlayValues[7] = d7
						ps19.OverlayValues[9] = d9
						ps19.OverlayValues[10] = d10
						ps19.OverlayValues[11] = d11
						ps19.OverlayValues[12] = d12
						ps19.OverlayValues[14] = d14
						ps19.OverlayValues[15] = d15
						ps19.OverlayValues[16] = d16
						ps19.OverlayValues[17] = d17
						return bbs[3].RenderPS(ps19)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d17.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 18)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[6] = d6
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[9] = d9
					ps20.OverlayValues[10] = d10
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[12] = d12
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[16] = d16
					ps20.OverlayValues[17] = d17
					ps21 := PhiState{General: true}
					ps21.OverlayValues = make([]JITValueDesc, 18)
					ps21.OverlayValues[1] = d1
					ps21.OverlayValues[2] = d2
					ps21.OverlayValues[3] = d3
					ps21.OverlayValues[6] = d6
					ps21.OverlayValues[7] = d7
					ps21.OverlayValues[9] = d9
					ps21.OverlayValues[10] = d10
					ps21.OverlayValues[11] = d11
					ps21.OverlayValues[12] = d12
					ps21.OverlayValues[14] = d14
					ps21.OverlayValues[15] = d15
					ps21.OverlayValues[16] = d16
					ps21.OverlayValues[17] = d17
					snap22 := d1
					snap23 := d2
					snap24 := d3
					snap25 := d6
					snap26 := d7
					snap27 := d9
					snap28 := d10
					snap29 := d11
					snap30 := d12
					snap31 := d14
					snap32 := d15
					snap33 := d16
					snap34 := d17
					alloc35 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps21)
					}
					ctx.RestoreAllocState(alloc35)
					d1 = snap22
					d2 = snap23
					d3 = snap24
					d6 = snap25
					d7 = snap26
					d9 = snap27
					d10 = snap28
					d11 = snap29
					d12 = snap30
					d14 = snap31
					d15 = snap32
					d16 = snap33
					d17 = snap34
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps20)
					}
					return result
					ctx.FreeDesc(&d16)
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
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					}
					ps36 := PhiState{General: ps.General}
					ps36.OverlayValues = make([]JITValueDesc, 18)
					ps36.OverlayValues[1] = d1
					ps36.OverlayValues[2] = d2
					ps36.OverlayValues[3] = d3
					ps36.OverlayValues[6] = d6
					ps36.OverlayValues[7] = d7
					ps36.OverlayValues[9] = d9
					ps36.OverlayValues[10] = d10
					ps36.OverlayValues[11] = d11
					ps36.OverlayValues[12] = d12
					ps36.OverlayValues[14] = d14
					ps36.OverlayValues[15] = d15
					ps36.OverlayValues[16] = d16
					ps36.OverlayValues[17] = d17
					ps36.PhiValues = make([]JITValueDesc, 1)
					d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps36.PhiValues[0] = d37
					if ps36.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps36)
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
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					ctx.ReclaimUntrackedRegs()
					d38 = d10
					_ = d38
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d38, &result)
						result.Type = d38.Type
					} else {
						switch d38.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d38)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d38)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d38)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d38, &result)
							result.Type = d38.Type
						}
					}
					ctx.EmitJmp(lbl0)
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
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d15)
					phiBase39 = ctx.AllocStack(int32(16))
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase39) + int32(0)}
					_ = d40
					lbl9 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					bbpos_1_4 := int32(-1)
					_ = bbpos_1_4
					bbpos_1_5 := int32(-1)
					_ = bbpos_1_5
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase39)+int32(0))
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d40)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d41 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						fieldAddr := uintptr(d15.Imm.Int()) + 0
						r3 := ctx.AllocReg()
						r4 := ctx.AllocRegExcept(r3)
						r5 := ctx.AllocRegExcept(r3, r4)
						ctx.EmitMovRegMem64(r3, fieldAddr)
						ctx.EmitMovRegMem64(r4, fieldAddr+8)
						ctx.EmitMovRegMem64(r5, fieldAddr+16)
						d41 = JITValueDesc{Loc: LocRegTriple, Reg: r3, Reg2: r4, Reg3: r5}
						ctx.BindReg(r3, &d41)
						ctx.BindReg(r4, &d41)
						ctx.BindReg(r5, &d41)
					} else {
						off := int32(0)
						baseReg := d15.Reg
						r6 := ctx.AllocRegExcept(baseReg)
						r7 := ctx.AllocRegExcept(baseReg, r6)
						r8 := ctx.AllocRegExcept(baseReg, r6, r7)
						ctx.EmitMovRegMem(r6, baseReg, off)
						ctx.EmitMovRegMem(r7, baseReg, off+8)
						ctx.EmitMovRegMem(r8, baseReg, off+16)
						d41 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
						ctx.BindReg(r6, &d41)
						ctx.BindReg(r7, &d41)
						ctx.BindReg(r8, &d41)
					}
					ctx.ReclaimUntrackedRegs()
					var d42 JITValueDesc
					if d41.SliceSizeKnown {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.KnownSliceLen))}
					} else if d41.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d41.StackOff))}
					} else if d41.Loc == LocStackTriple {
						d42 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d41.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d41)
						if d41.Loc == LocRegPair || d41.Loc == LocRegTriple {
							d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg2, ID: 0}
						} else if d41.Loc == LocReg {
							d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d41.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d42)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d42)
					var d43 JITValueDesc
					if d40.Loc == LocImm && d42.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d40.Imm.Int() < d42.Imm.Int())}
					} else if d42.Loc == LocImm {
						r9 := ctx.AllocRegExcept(d40.Reg)
						if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d40.Reg, int32(d42.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d42.Imm.Int()))
							ctx.EmitCmpInt64(d40.Reg, RegR11)
						}
						ctx.EmitSetcc(r9, CondSignedLess)
						d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d43)
					} else if d40.Loc == LocImm {
						r10 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d40.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d42.Reg)
						ctx.EmitSetcc(r10, CondSignedLess)
						d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
						ctx.BindReg(r10, &d43)
					} else {
						r11 := ctx.AllocRegExcept(d40.Reg)
						ctx.EmitCmpInt64(d40.Reg, d42.Reg)
						ctx.EmitSetcc(r11, CondSignedLess)
						d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
						ctx.BindReg(r11, &d43)
					}
					ctx.FreeDesc(&d42)
					ctx.ReclaimUntrackedRegs()
					d44 = d43
					ctx.EnsureDesc(&d44)
					if d44.Loc != LocImm && d44.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d44.Loc == LocImm {
						if d44.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitCmpRegImm32(d44.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					ctx.FreeDesc(&d43)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d45 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						fieldAddr := uintptr(d15.Imm.Int()) + 0
						r12 := ctx.AllocReg()
						r13 := ctx.AllocRegExcept(r12)
						r14 := ctx.AllocRegExcept(r12, r13)
						ctx.EmitMovRegMem64(r12, fieldAddr)
						ctx.EmitMovRegMem64(r13, fieldAddr+8)
						ctx.EmitMovRegMem64(r14, fieldAddr+16)
						d45 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
						ctx.BindReg(r12, &d45)
						ctx.BindReg(r13, &d45)
						ctx.BindReg(r14, &d45)
					} else {
						off := int32(0)
						baseReg := d15.Reg
						r15 := ctx.AllocRegExcept(baseReg)
						r16 := ctx.AllocRegExcept(baseReg, r15)
						r17 := ctx.AllocRegExcept(baseReg, r15, r16)
						ctx.EmitMovRegMem(r15, baseReg, off)
						ctx.EmitMovRegMem(r16, baseReg, off+8)
						ctx.EmitMovRegMem(r17, baseReg, off+16)
						d45 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
						ctx.BindReg(r15, &d45)
						ctx.BindReg(r16, &d45)
						ctx.BindReg(r17, &d45)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d40)
					ctx.ReclaimUntrackedRegs()
					d47 = ctx.EmitSliceElementAddress(&d45, &d40, 16)
					ctx.EnsureDesc(&d47)
					r18 := ctx.AllocRegExcept(d47.Reg)
					ctx.EmitMovRegMem(r18, d47.Reg, 8)
					ctx.EmitMovRegMem(d47.Reg, d47.Reg, 0)
					d46 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d47.Reg, Reg2: r18}
					ctx.BindReg(d47.Reg, &d46)
					ctx.BindReg(r18, &d46)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d48 JITValueDesc
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						fieldAddr := uintptr(d15.Imm.Int()) + 0
						r19 := ctx.AllocReg()
						r20 := ctx.AllocRegExcept(r19)
						r21 := ctx.AllocRegExcept(r19, r20)
						ctx.EmitMovRegMem64(r19, fieldAddr)
						ctx.EmitMovRegMem64(r20, fieldAddr+8)
						ctx.EmitMovRegMem64(r21, fieldAddr+16)
						d48 = JITValueDesc{Loc: LocRegTriple, Reg: r19, Reg2: r20, Reg3: r21}
						ctx.BindReg(r19, &d48)
						ctx.BindReg(r20, &d48)
						ctx.BindReg(r21, &d48)
					} else {
						off := int32(0)
						baseReg := d15.Reg
						r22 := ctx.AllocRegExcept(baseReg)
						r23 := ctx.AllocRegExcept(baseReg, r22)
						r24 := ctx.AllocRegExcept(baseReg, r22, r23)
						ctx.EmitMovRegMem(r22, baseReg, off)
						ctx.EmitMovRegMem(r23, baseReg, off+8)
						ctx.EmitMovRegMem(r24, baseReg, off+16)
						d48 = JITValueDesc{Loc: LocRegTriple, Reg: r22, Reg2: r23, Reg3: r24}
						ctx.BindReg(r22, &d48)
						ctx.BindReg(r23, &d48)
						ctx.BindReg(r24, &d48)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					var d49 JITValueDesc
					if d40.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d40.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d40.Reg)
						ctx.EmitMovRegReg(scratch, d40.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d49)
					}
					if d49.Loc == LocReg && d40.Loc == LocReg && d49.Reg == d40.Reg {
						ctx.TransferReg(d40.Reg)
						d40.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d49)
					ctx.ReclaimUntrackedRegs()
					d51 = ctx.EmitSliceElementAddress(&d48, &d49, 16)
					ctx.EnsureDesc(&d51)
					r25 := ctx.AllocRegExcept(d51.Reg)
					ctx.EmitMovRegMem(r25, d51.Reg, 8)
					ctx.EmitMovRegMem(d51.Reg, d51.Reg, 0)
					d50 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d51.Reg, Reg2: r25}
					ctx.BindReg(d51.Reg, &d50)
					ctx.BindReg(r25, &d50)
					ctx.FreeDesc(&d49)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d46)
					ctx.EnsureDesc(&d50)
					d52 = d46
					_ = d52
					ctx.StabilizeDescForControlFlow(&d52)
					d53 = d50
					_ = d53
					ctx.StabilizeDescForControlFlow(&d53)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d54 = d7
					_ = d54
					ctx.ReclaimUntrackedRegs()
					d55 = d10
					_ = d55
					ctx.ReclaimUntrackedRegs()
					d56 = d3
					_ = d56
					ctx.ReclaimUntrackedRegs()
					stackArray57 = ctx.AllocStack(int32(32))
					_ = stackArray57
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d52)
					ctx.EmitStoreScmerToStack(d52, int32(stackArray57)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray57)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d58 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d58
					ctx.ReclaimUntrackedRegs()
					callbackArgs60 := make([]JITValueDesc, 2)
					callbackArgs60[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray57) + 0}
					callbackArgs60[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray57) + 16}
					var d59 JITValueDesc
					callbackResultOff61 = ctx.AllocStack(16)
					ctx.FreeDesc(&d58)
					if d56.Loc == LocLambdaTemplate && d56.Lambda != nil {
						stableCallbackArgs62 := ctx.StabilizeCallbackArgs(callbackArgs60)
						ctx.ReclaimUntrackedRegs()
						outerRegs63 := ctx.PreserveOuterRegs()
						d59 = JITEmitProcInlineWithOuter(ctx, &d56.Lambda.Proc, d56.Lambda.Outer, stableCallbackArgs62, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff61), ID: 0})
						ctx.RestoreOuterRegs(outerRegs63)
						ctx.ReclaimUntrackedRegs()
					} else {
						d64, knownBuiltin65 := jitEmitKnownDeclaration(ctx, d56, callbackArgs60, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff61), ID: 0})
						if knownBuiltin65 {
							d59 = d64
						} else {
							d66 := jitCopyScmerToPair(ctx, d56)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d66)
							callbackCallArgs = append(callbackCallArgs, callbackArgs60...)
							d59 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d59, int32(callbackResultOff61))
							ctx.FreeDesc(&d59)
							d59 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff61), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					stackArray67 = ctx.AllocStack(int32(48))
					_ = stackArray67
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d55)
					d68 = jitMaterializeVirtualSlice(ctx, d55, JITValueDesc{Loc: LocAny})
					ctx.EmitStoreScmerToStack(d68, int32(stackArray67)+int32(0))
					ctx.FreeDesc(&d68)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d59)
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreScmerToStack(d59, int32(stackArray67)+int32(16))
					ctx.FreeDesc(&d59)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d53)
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(stackArray67)+int32(32))
					ctx.ReclaimUntrackedRegs()
					d69 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d69
					ctx.ReclaimUntrackedRegs()
					callbackArgs71 := make([]JITValueDesc, 3)
					callbackArgs71[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray67) + 0}
					callbackArgs71[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray67) + 16}
					callbackArgs71[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray67) + 32}
					var d70 JITValueDesc
					callbackResultOff72 = ctx.AllocStack(16)
					ctx.FreeDesc(&d69)
					if d54.Loc == LocLambdaTemplate && d54.Lambda != nil {
						stableCallbackArgs73 := ctx.StabilizeCallbackArgs(callbackArgs71)
						ctx.ReclaimUntrackedRegs()
						outerRegs74 := ctx.PreserveOuterRegs()
						d70 = JITEmitProcInlineWithOuter(ctx, &d54.Lambda.Proc, d54.Lambda.Outer, stableCallbackArgs73, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff72), ID: 0})
						ctx.RestoreOuterRegs(outerRegs74)
						ctx.ReclaimUntrackedRegs()
					} else {
						d75, knownBuiltin76 := jitEmitKnownDeclaration(ctx, d54, callbackArgs71, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff72), ID: 0})
						if knownBuiltin76 {
							d70 = d75
						} else {
							d77 := jitCopyScmerToPair(ctx, d54)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d77)
							callbackCallArgs = append(callbackCallArgs, callbackArgs71...)
							d70 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d70, int32(callbackResultOff72))
							ctx.FreeDesc(&d70)
							d70 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff72), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d70)
					ctx.FreeDesc(&d70)
					ctx.FreeDesc(&d70)
					ctx.ReclaimUntrackedRegs()
					d78 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.FreeDesc(&d46)
					ctx.FreeDesc(&d50)
					ctx.ReclaimUntrackedRegs()
					d79 = d78
					ctx.EnsureDesc(&d79)
					if d79.Loc != LocImm && d79.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					if d79.Loc == LocImm {
						if d79.Imm.Bool() {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl15)
						}
					} else {
						ctx.EmitCmpRegImm32(d79.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl16)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
					}
					ctx.FreeDesc(&d78)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase39) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d40)
					ctx.EnsureDesc(&d40)
					var d80 JITValueDesc
					if d40.Loc == LocImm {
						d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d40.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d40.Reg)
						ctx.EmitMovRegReg(scratch, d40.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d80)
					}
					if d80.Loc == LocReg && d40.Loc == LocReg && d80.Reg == d40.Reg {
						ctx.TransferReg(d40.Reg)
						d40.Loc = LocNone
					}
					ctx.EnsureDesc(&d80)
					ctx.EmitStoreToStack(d80, int32(phiBase39)+int32(0))
					ctx.StabilizeDescForControlFlow(&d80)
					ctx.FreeDesc(&d40)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_1_1)
					ctx.MarkLabel(lbl9)
					ctx.FreeDesc(&d15)
					if ps.General {
					}
					ps81 := PhiState{General: ps.General}
					ps81.OverlayValues = make([]JITValueDesc, 81)
					ps81.OverlayValues[1] = d1
					ps81.OverlayValues[2] = d2
					ps81.OverlayValues[3] = d3
					ps81.OverlayValues[6] = d6
					ps81.OverlayValues[7] = d7
					ps81.OverlayValues[9] = d9
					ps81.OverlayValues[10] = d10
					ps81.OverlayValues[11] = d11
					ps81.OverlayValues[12] = d12
					ps81.OverlayValues[14] = d14
					ps81.OverlayValues[15] = d15
					ps81.OverlayValues[16] = d16
					ps81.OverlayValues[17] = d17
					ps81.OverlayValues[37] = d37
					ps81.OverlayValues[38] = d38
					ps81.OverlayValues[40] = d40
					ps81.OverlayValues[41] = d41
					ps81.OverlayValues[42] = d42
					ps81.OverlayValues[43] = d43
					ps81.OverlayValues[44] = d44
					ps81.OverlayValues[45] = d45
					ps81.OverlayValues[46] = d46
					ps81.OverlayValues[47] = d47
					ps81.OverlayValues[48] = d48
					ps81.OverlayValues[49] = d49
					ps81.OverlayValues[50] = d50
					ps81.OverlayValues[51] = d51
					ps81.OverlayValues[52] = d52
					ps81.OverlayValues[53] = d53
					ps81.OverlayValues[54] = d54
					ps81.OverlayValues[55] = d55
					ps81.OverlayValues[56] = d56
					ps81.OverlayValues[58] = d58
					ps81.OverlayValues[59] = d59
					ps81.OverlayValues[64] = d64
					ps81.OverlayValues[66] = d66
					ps81.OverlayValues[68] = d68
					ps81.OverlayValues[69] = d69
					ps81.OverlayValues[70] = d70
					ps81.OverlayValues[75] = d75
					ps81.OverlayValues[77] = d77
					ps81.OverlayValues[78] = d78
					ps81.OverlayValues[79] = d79
					ps81.OverlayValues[80] = d80
					if ps81.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps81)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d82 := ps.PhiValues[0]
							ctx.EnsureDesc(&d82)
							ctx.EmitStoreToStack(d82, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
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
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					var d83 JITValueDesc
					if d14.SliceSizeKnown {
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d14.KnownSliceLen))}
					} else if d14.Loc == LocImm {
						d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d14.StackOff))}
					} else if d14.Loc == LocStackTriple {
						d83 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d14.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d14)
						if d14.Loc == LocRegPair || d14.Loc == LocRegTriple {
							d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg2, ID: 0}
						} else if d14.Loc == LocReg {
							d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d83)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d83)
					var d84 JITValueDesc
					if d1.Loc == LocImm && d83.Loc == LocImm {
						d84 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d83.Imm.Int())}
					} else if d83.Loc == LocImm {
						r26 := ctx.AllocRegExcept(d1.Reg)
						if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d83.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d83.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r26, CondSignedLess)
						d84 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d84)
					} else if d1.Loc == LocImm {
						r27 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d83.Reg)
						ctx.EmitSetcc(r27, CondSignedLess)
						d84 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r27}
						ctx.BindReg(r27, &d84)
					} else {
						r28 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d83.Reg)
						ctx.EmitSetcc(r28, CondSignedLess)
						d84 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r28}
						ctx.BindReg(r28, &d84)
					}
					ctx.FreeDesc(&d83)
					d85 = d84
					ctx.EnsureDesc(&d85)
					if d85.Loc != LocImm && d85.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d85.Loc == LocImm {
						if d85.Imm.Bool() {
							if ps.General {
							}
							ps86 := PhiState{General: ps.General}
							ps86.OverlayValues = make([]JITValueDesc, 86)
							ps86.OverlayValues[1] = d1
							ps86.OverlayValues[2] = d2
							ps86.OverlayValues[3] = d3
							ps86.OverlayValues[6] = d6
							ps86.OverlayValues[7] = d7
							ps86.OverlayValues[9] = d9
							ps86.OverlayValues[10] = d10
							ps86.OverlayValues[11] = d11
							ps86.OverlayValues[12] = d12
							ps86.OverlayValues[14] = d14
							ps86.OverlayValues[15] = d15
							ps86.OverlayValues[16] = d16
							ps86.OverlayValues[17] = d17
							ps86.OverlayValues[37] = d37
							ps86.OverlayValues[38] = d38
							ps86.OverlayValues[40] = d40
							ps86.OverlayValues[41] = d41
							ps86.OverlayValues[42] = d42
							ps86.OverlayValues[43] = d43
							ps86.OverlayValues[44] = d44
							ps86.OverlayValues[45] = d45
							ps86.OverlayValues[46] = d46
							ps86.OverlayValues[47] = d47
							ps86.OverlayValues[48] = d48
							ps86.OverlayValues[49] = d49
							ps86.OverlayValues[50] = d50
							ps86.OverlayValues[51] = d51
							ps86.OverlayValues[52] = d52
							ps86.OverlayValues[53] = d53
							ps86.OverlayValues[54] = d54
							ps86.OverlayValues[55] = d55
							ps86.OverlayValues[56] = d56
							ps86.OverlayValues[58] = d58
							ps86.OverlayValues[59] = d59
							ps86.OverlayValues[64] = d64
							ps86.OverlayValues[66] = d66
							ps86.OverlayValues[68] = d68
							ps86.OverlayValues[69] = d69
							ps86.OverlayValues[70] = d70
							ps86.OverlayValues[75] = d75
							ps86.OverlayValues[77] = d77
							ps86.OverlayValues[78] = d78
							ps86.OverlayValues[79] = d79
							ps86.OverlayValues[80] = d80
							ps86.OverlayValues[82] = d82
							ps86.OverlayValues[83] = d83
							ps86.OverlayValues[84] = d84
							ps86.OverlayValues[85] = d85
							return bbs[5].RenderPS(ps86)
						}
						if ps.General {
						}
						ps87 := PhiState{General: ps.General}
						ps87.OverlayValues = make([]JITValueDesc, 86)
						ps87.OverlayValues[1] = d1
						ps87.OverlayValues[2] = d2
						ps87.OverlayValues[3] = d3
						ps87.OverlayValues[6] = d6
						ps87.OverlayValues[7] = d7
						ps87.OverlayValues[9] = d9
						ps87.OverlayValues[10] = d10
						ps87.OverlayValues[11] = d11
						ps87.OverlayValues[12] = d12
						ps87.OverlayValues[14] = d14
						ps87.OverlayValues[15] = d15
						ps87.OverlayValues[16] = d16
						ps87.OverlayValues[17] = d17
						ps87.OverlayValues[37] = d37
						ps87.OverlayValues[38] = d38
						ps87.OverlayValues[40] = d40
						ps87.OverlayValues[41] = d41
						ps87.OverlayValues[42] = d42
						ps87.OverlayValues[43] = d43
						ps87.OverlayValues[44] = d44
						ps87.OverlayValues[45] = d45
						ps87.OverlayValues[46] = d46
						ps87.OverlayValues[47] = d47
						ps87.OverlayValues[48] = d48
						ps87.OverlayValues[49] = d49
						ps87.OverlayValues[50] = d50
						ps87.OverlayValues[51] = d51
						ps87.OverlayValues[52] = d52
						ps87.OverlayValues[53] = d53
						ps87.OverlayValues[54] = d54
						ps87.OverlayValues[55] = d55
						ps87.OverlayValues[56] = d56
						ps87.OverlayValues[58] = d58
						ps87.OverlayValues[59] = d59
						ps87.OverlayValues[64] = d64
						ps87.OverlayValues[66] = d66
						ps87.OverlayValues[68] = d68
						ps87.OverlayValues[69] = d69
						ps87.OverlayValues[70] = d70
						ps87.OverlayValues[75] = d75
						ps87.OverlayValues[77] = d77
						ps87.OverlayValues[78] = d78
						ps87.OverlayValues[79] = d79
						ps87.OverlayValues[80] = d80
						ps87.OverlayValues[82] = d82
						ps87.OverlayValues[83] = d83
						ps87.OverlayValues[84] = d84
						ps87.OverlayValues[85] = d85
						return bbs[2].RenderPS(ps87)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d88 := ps.PhiValues[0]
							ctx.EnsureDesc(&d88)
							ctx.EmitStoreToStack(d88, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d85.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl3)
					ps89 := PhiState{General: true}
					ps89.OverlayValues = make([]JITValueDesc, 89)
					ps89.OverlayValues[1] = d1
					ps89.OverlayValues[2] = d2
					ps89.OverlayValues[3] = d3
					ps89.OverlayValues[6] = d6
					ps89.OverlayValues[7] = d7
					ps89.OverlayValues[9] = d9
					ps89.OverlayValues[10] = d10
					ps89.OverlayValues[11] = d11
					ps89.OverlayValues[12] = d12
					ps89.OverlayValues[14] = d14
					ps89.OverlayValues[15] = d15
					ps89.OverlayValues[16] = d16
					ps89.OverlayValues[17] = d17
					ps89.OverlayValues[37] = d37
					ps89.OverlayValues[38] = d38
					ps89.OverlayValues[40] = d40
					ps89.OverlayValues[41] = d41
					ps89.OverlayValues[42] = d42
					ps89.OverlayValues[43] = d43
					ps89.OverlayValues[44] = d44
					ps89.OverlayValues[45] = d45
					ps89.OverlayValues[46] = d46
					ps89.OverlayValues[47] = d47
					ps89.OverlayValues[48] = d48
					ps89.OverlayValues[49] = d49
					ps89.OverlayValues[50] = d50
					ps89.OverlayValues[51] = d51
					ps89.OverlayValues[52] = d52
					ps89.OverlayValues[53] = d53
					ps89.OverlayValues[54] = d54
					ps89.OverlayValues[55] = d55
					ps89.OverlayValues[56] = d56
					ps89.OverlayValues[58] = d58
					ps89.OverlayValues[59] = d59
					ps89.OverlayValues[64] = d64
					ps89.OverlayValues[66] = d66
					ps89.OverlayValues[68] = d68
					ps89.OverlayValues[69] = d69
					ps89.OverlayValues[70] = d70
					ps89.OverlayValues[75] = d75
					ps89.OverlayValues[77] = d77
					ps89.OverlayValues[78] = d78
					ps89.OverlayValues[79] = d79
					ps89.OverlayValues[80] = d80
					ps89.OverlayValues[82] = d82
					ps89.OverlayValues[83] = d83
					ps89.OverlayValues[84] = d84
					ps89.OverlayValues[85] = d85
					ps89.OverlayValues[88] = d88
					ps90 := PhiState{General: true}
					ps90.OverlayValues = make([]JITValueDesc, 89)
					ps90.OverlayValues[1] = d1
					ps90.OverlayValues[2] = d2
					ps90.OverlayValues[3] = d3
					ps90.OverlayValues[6] = d6
					ps90.OverlayValues[7] = d7
					ps90.OverlayValues[9] = d9
					ps90.OverlayValues[10] = d10
					ps90.OverlayValues[11] = d11
					ps90.OverlayValues[12] = d12
					ps90.OverlayValues[14] = d14
					ps90.OverlayValues[15] = d15
					ps90.OverlayValues[16] = d16
					ps90.OverlayValues[17] = d17
					ps90.OverlayValues[37] = d37
					ps90.OverlayValues[38] = d38
					ps90.OverlayValues[40] = d40
					ps90.OverlayValues[41] = d41
					ps90.OverlayValues[42] = d42
					ps90.OverlayValues[43] = d43
					ps90.OverlayValues[44] = d44
					ps90.OverlayValues[45] = d45
					ps90.OverlayValues[46] = d46
					ps90.OverlayValues[47] = d47
					ps90.OverlayValues[48] = d48
					ps90.OverlayValues[49] = d49
					ps90.OverlayValues[50] = d50
					ps90.OverlayValues[51] = d51
					ps90.OverlayValues[52] = d52
					ps90.OverlayValues[53] = d53
					ps90.OverlayValues[54] = d54
					ps90.OverlayValues[55] = d55
					ps90.OverlayValues[56] = d56
					ps90.OverlayValues[58] = d58
					ps90.OverlayValues[59] = d59
					ps90.OverlayValues[64] = d64
					ps90.OverlayValues[66] = d66
					ps90.OverlayValues[68] = d68
					ps90.OverlayValues[69] = d69
					ps90.OverlayValues[70] = d70
					ps90.OverlayValues[75] = d75
					ps90.OverlayValues[77] = d77
					ps90.OverlayValues[78] = d78
					ps90.OverlayValues[79] = d79
					ps90.OverlayValues[80] = d80
					ps90.OverlayValues[82] = d82
					ps90.OverlayValues[83] = d83
					ps90.OverlayValues[84] = d84
					ps90.OverlayValues[85] = d85
					ps90.OverlayValues[88] = d88
					snap91 := d1
					snap92 := d2
					snap93 := d3
					snap94 := d6
					snap95 := d7
					snap96 := d9
					snap97 := d10
					snap98 := d11
					snap99 := d12
					snap100 := d14
					snap101 := d15
					snap102 := d16
					snap103 := d17
					snap104 := d37
					snap105 := d38
					snap106 := d40
					snap107 := d41
					snap108 := d42
					snap109 := d43
					snap110 := d44
					snap111 := d45
					snap112 := d46
					snap113 := d47
					snap114 := d48
					snap115 := d49
					snap116 := d50
					snap117 := d51
					snap118 := d52
					snap119 := d53
					snap120 := d54
					snap121 := d55
					snap122 := d56
					snap123 := d58
					snap124 := d59
					snap125 := d64
					snap126 := d66
					snap127 := d68
					snap128 := d69
					snap129 := d70
					snap130 := d75
					snap131 := d77
					snap132 := d78
					snap133 := d79
					snap134 := d80
					snap135 := d82
					snap136 := d83
					snap137 := d84
					snap138 := d85
					snap139 := d88
					alloc140 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps90)
					}
					ctx.RestoreAllocState(alloc140)
					d1 = snap91
					d2 = snap92
					d3 = snap93
					d6 = snap94
					d7 = snap95
					d9 = snap96
					d10 = snap97
					d11 = snap98
					d12 = snap99
					d14 = snap100
					d15 = snap101
					d16 = snap102
					d17 = snap103
					d37 = snap104
					d38 = snap105
					d40 = snap106
					d41 = snap107
					d42 = snap108
					d43 = snap109
					d44 = snap110
					d45 = snap111
					d46 = snap112
					d47 = snap113
					d48 = snap114
					d49 = snap115
					d50 = snap116
					d51 = snap117
					d52 = snap118
					d53 = snap119
					d54 = snap120
					d55 = snap121
					d56 = snap122
					d58 = snap123
					d59 = snap124
					d64 = snap125
					d66 = snap126
					d68 = snap127
					d69 = snap128
					d70 = snap129
					d75 = snap130
					d77 = snap131
					d78 = snap132
					d79 = snap133
					d80 = snap134
					d82 = snap135
					d83 = snap136
					d84 = snap137
					d85 = snap138
					d88 = snap139
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps89)
					}
					return result
					ctx.FreeDesc(&d84)
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
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
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
					if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
						d58 = ps.OverlayValues[58]
					}
					if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
						d59 = ps.OverlayValues[59]
					}
					if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
						d64 = ps.OverlayValues[64]
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
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
					}
					if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
						d80 = ps.OverlayValues[80]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != LocNone {
						d83 = ps.OverlayValues[83]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					ctx.ReclaimUntrackedRegs()
					d141 = d7
					_ = d141
					d142 = d10
					_ = d142
					d143 = d3
					_ = d143
					ctx.EnsureDesc(&d1)
					d145 = ctx.EmitSliceElementAddress(&d14, &d1, 16)
					ctx.EnsureDesc(&d145)
					r29 := ctx.AllocRegExcept(d145.Reg)
					ctx.EmitMovRegMem(r29, d145.Reg, 8)
					ctx.EmitMovRegMem(d145.Reg, d145.Reg, 0)
					d144 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d145.Reg, Reg2: r29}
					ctx.BindReg(d145.Reg, &d144)
					ctx.BindReg(r29, &d144)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d146 JITValueDesc
					if d1.Loc == LocImm {
						d146 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d146 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d146)
					}
					if d146.Loc == LocReg && d1.Loc == LocReg && d146.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d146)
					d148 = ctx.EmitSliceElementAddress(&d14, &d146, 16)
					ctx.EnsureDesc(&d148)
					r30 := ctx.AllocRegExcept(d148.Reg)
					ctx.EmitMovRegMem(r30, d148.Reg, 8)
					ctx.EmitMovRegMem(d148.Reg, d148.Reg, 0)
					d147 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d148.Reg, Reg2: r30}
					ctx.BindReg(d148.Reg, &d147)
					ctx.BindReg(r30, &d147)
					ctx.FreeDesc(&d146)
					stackArray149 = ctx.AllocStack(int32(32))
					_ = stackArray149
					ctx.EnsureDesc(&d144)
					ctx.EnsureDesc(&d144)
					ctx.EmitStoreScmerToStack(d144, int32(stackArray149)+int32(0))
					ctx.FreeDesc(&d144)
					ctx.EnsureDesc(&d147)
					ctx.EnsureDesc(&d147)
					ctx.EmitStoreScmerToStack(d147, int32(stackArray149)+int32(16))
					ctx.FreeDesc(&d147)
					d150 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d150
					callbackArgs152 := make([]JITValueDesc, 2)
					callbackArgs152[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray149) + 0}
					callbackArgs152[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray149) + 16}
					var d151 JITValueDesc
					callbackResultOff153 = ctx.AllocStack(16)
					ctx.FreeDesc(&d150)
					if d143.Loc == LocLambdaTemplate && d143.Lambda != nil {
						stableCallbackArgs154 := ctx.StabilizeCallbackArgs(callbackArgs152)
						ctx.ReclaimUntrackedRegs()
						outerRegs155 := ctx.PreserveOuterRegs()
						d151 = JITEmitProcInlineWithOuter(ctx, &d143.Lambda.Proc, d143.Lambda.Outer, stableCallbackArgs154, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff153), ID: 0})
						ctx.RestoreOuterRegs(outerRegs155)
						ctx.ReclaimUntrackedRegs()
					} else {
						d156, knownBuiltin157 := jitEmitKnownDeclaration(ctx, d143, callbackArgs152, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff153), ID: 0})
						if knownBuiltin157 {
							d151 = d156
						} else {
							d158 := jitCopyScmerToPair(ctx, d143)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d158)
							callbackCallArgs = append(callbackCallArgs, callbackArgs152...)
							d151 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d151, int32(callbackResultOff153))
							ctx.FreeDesc(&d151)
							d151 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff153), ID: 0}
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d159 JITValueDesc
					if d1.Loc == LocImm {
						d159 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d159 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d159)
					}
					if d159.Loc == LocReg && d1.Loc == LocReg && d159.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d159)
					d161 = ctx.EmitSliceElementAddress(&d14, &d159, 16)
					ctx.EnsureDesc(&d161)
					r31 := ctx.AllocRegExcept(d161.Reg)
					ctx.EmitMovRegMem(r31, d161.Reg, 8)
					ctx.EmitMovRegMem(d161.Reg, d161.Reg, 0)
					d160 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d161.Reg, Reg2: r31}
					ctx.BindReg(d161.Reg, &d160)
					ctx.BindReg(r31, &d160)
					ctx.FreeDesc(&d159)
					stackArray162 = ctx.AllocStack(int32(48))
					_ = stackArray162
					ctx.EnsureDesc(&d142)
					ctx.EnsureDesc(&d142)
					d163 = jitMaterializeVirtualSlice(ctx, d142, JITValueDesc{Loc: LocAny})
					ctx.EmitStoreScmerToStack(d163, int32(stackArray162)+int32(0))
					ctx.FreeDesc(&d163)
					ctx.EnsureDesc(&d151)
					ctx.EnsureDesc(&d151)
					ctx.EmitStoreScmerToStack(d151, int32(stackArray162)+int32(16))
					ctx.FreeDesc(&d151)
					ctx.EnsureDesc(&d160)
					ctx.EnsureDesc(&d160)
					ctx.EmitStoreScmerToStack(d160, int32(stackArray162)+int32(32))
					ctx.FreeDesc(&d160)
					d164 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d164
					callbackArgs166 := make([]JITValueDesc, 3)
					callbackArgs166[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray162) + 0}
					callbackArgs166[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray162) + 16}
					callbackArgs166[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray162) + 32}
					var d165 JITValueDesc
					callbackResultOff167 = ctx.AllocStack(16)
					ctx.FreeDesc(&d164)
					if d141.Loc == LocLambdaTemplate && d141.Lambda != nil {
						stableCallbackArgs168 := ctx.StabilizeCallbackArgs(callbackArgs166)
						ctx.ReclaimUntrackedRegs()
						outerRegs169 := ctx.PreserveOuterRegs()
						d165 = JITEmitProcInlineWithOuter(ctx, &d141.Lambda.Proc, d141.Lambda.Outer, stableCallbackArgs168, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff167), ID: 0})
						ctx.RestoreOuterRegs(outerRegs169)
						ctx.ReclaimUntrackedRegs()
					} else {
						d170, knownBuiltin171 := jitEmitKnownDeclaration(ctx, d141, callbackArgs166, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff167), ID: 0})
						if knownBuiltin171 {
							d165 = d170
						} else {
							d172 := jitCopyScmerToPair(ctx, d141)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d172)
							callbackCallArgs = append(callbackCallArgs, callbackArgs166...)
							d165 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d165, int32(callbackResultOff167))
							ctx.FreeDesc(&d165)
							d165 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff167), ID: 0}
						}
					}
					ctx.SyncDesc(&d165)
					ctx.FreeDesc(&d165)
					ctx.FreeDesc(&d165)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d173 JITValueDesc
					if d1.Loc == LocImm {
						d173 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d173 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d173)
					}
					if d173.Loc == LocReg && d1.Loc == LocReg && d173.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d173)
					ctx.EmitStoreToStack(d173, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d173)
					ctx.FreeDesc(&d1)
					if ps.General {
					}
					ps174 := PhiState{General: ps.General}
					ps174.OverlayValues = make([]JITValueDesc, 174)
					ps174.OverlayValues[1] = d1
					ps174.OverlayValues[2] = d2
					ps174.OverlayValues[3] = d3
					ps174.OverlayValues[6] = d6
					ps174.OverlayValues[7] = d7
					ps174.OverlayValues[9] = d9
					ps174.OverlayValues[10] = d10
					ps174.OverlayValues[11] = d11
					ps174.OverlayValues[12] = d12
					ps174.OverlayValues[14] = d14
					ps174.OverlayValues[15] = d15
					ps174.OverlayValues[16] = d16
					ps174.OverlayValues[17] = d17
					ps174.OverlayValues[37] = d37
					ps174.OverlayValues[38] = d38
					ps174.OverlayValues[40] = d40
					ps174.OverlayValues[41] = d41
					ps174.OverlayValues[42] = d42
					ps174.OverlayValues[43] = d43
					ps174.OverlayValues[44] = d44
					ps174.OverlayValues[45] = d45
					ps174.OverlayValues[46] = d46
					ps174.OverlayValues[47] = d47
					ps174.OverlayValues[48] = d48
					ps174.OverlayValues[49] = d49
					ps174.OverlayValues[50] = d50
					ps174.OverlayValues[51] = d51
					ps174.OverlayValues[52] = d52
					ps174.OverlayValues[53] = d53
					ps174.OverlayValues[54] = d54
					ps174.OverlayValues[55] = d55
					ps174.OverlayValues[56] = d56
					ps174.OverlayValues[58] = d58
					ps174.OverlayValues[59] = d59
					ps174.OverlayValues[64] = d64
					ps174.OverlayValues[66] = d66
					ps174.OverlayValues[68] = d68
					ps174.OverlayValues[69] = d69
					ps174.OverlayValues[70] = d70
					ps174.OverlayValues[75] = d75
					ps174.OverlayValues[77] = d77
					ps174.OverlayValues[78] = d78
					ps174.OverlayValues[79] = d79
					ps174.OverlayValues[80] = d80
					ps174.OverlayValues[82] = d82
					ps174.OverlayValues[83] = d83
					ps174.OverlayValues[84] = d84
					ps174.OverlayValues[85] = d85
					ps174.OverlayValues[88] = d88
					ps174.OverlayValues[141] = d141
					ps174.OverlayValues[142] = d142
					ps174.OverlayValues[143] = d143
					ps174.OverlayValues[144] = d144
					ps174.OverlayValues[145] = d145
					ps174.OverlayValues[146] = d146
					ps174.OverlayValues[147] = d147
					ps174.OverlayValues[148] = d148
					ps174.OverlayValues[150] = d150
					ps174.OverlayValues[151] = d151
					ps174.OverlayValues[156] = d156
					ps174.OverlayValues[158] = d158
					ps174.OverlayValues[159] = d159
					ps174.OverlayValues[160] = d160
					ps174.OverlayValues[161] = d161
					ps174.OverlayValues[163] = d163
					ps174.OverlayValues[164] = d164
					ps174.OverlayValues[165] = d165
					ps174.OverlayValues[170] = d170
					ps174.OverlayValues[172] = d172
					ps174.OverlayValues[173] = d173
					ps174.PhiValues = make([]JITValueDesc, 1)
					if ps174.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps174)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps175 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps175)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
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
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["mapkey_assoc_mut"].Fn, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d13 JITValueDesc
				_ = d13
				var d14 JITValueDesc
				_ = d14
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var stackArray34 int32
				var d35 JITValueDesc
				_ = d35
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
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
				var d45 JITValueDesc
				_ = d45
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var phiBase51 int32
				_ = phiBase51
				var d52 JITValueDesc
				_ = d52
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
				var d65 JITValueDesc
				_ = d65
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
				var stackArray79 int32
				var d80 JITValueDesc
				_ = d80
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var callbackResultOff84 int32
				var d87 JITValueDesc
				_ = d87
				var d89 JITValueDesc
				_ = d89
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
				var d99 JITValueDesc
				_ = d99
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
				var d104 JITValueDesc
				_ = d104
				var d105 JITValueDesc
				_ = d105
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
				var stackArray182 int32
				var d183 JITValueDesc
				_ = d183
				var d184 JITValueDesc
				_ = d184
				var callbackResultOff186 int32
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
				var stackArray195 int32
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d201 JITValueDesc
				_ = d201
				var d203 JITValueDesc
				_ = d203
				var d204 JITValueDesc
				_ = d204
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				bbs[3].PhiCount = uint16(2)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					d3 = args[1]
					d3.ID = 0
					var d4 JITValueDesc
					if d3.Loc == LocLambdaTemplate {
						d4 = d3
					} else if d3.Loc == LocImm {
						optimizedCallback5 := NewFunc(OptimizeProcToSerialFunction(d3.Imm))
						ctx.TrackImm(optimizedCallback5)
						d4 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback5, Rooted: true}
					} else {
						d4 = ctx.RequestOptimizedCallback(1)
					}
					ctx.FreeDesc(&d3)
					ctx.SyncDesc(&d4)
					globalLookup6 := Globalenv.Vars[Symbol("set_assoc_mut")]
					ctx.TrackImm(globalLookup6)
					d7 = JITValueDesc{Loc: LocImm, Type: globalLookup6.GetTag(), Imm: globalLookup6, Rooted: true}
					optimizedCallback9 := NewFunc(OptimizeProcToSerialFunction(d7.Imm))
					ctx.TrackImm(optimizedCallback9)
					d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					ctx.SyncDesc(&d8)
					d10 = args[0]
					d10.ID = 0
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					ctx.EnsureDesc(&d10)
					if d10.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d10.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d10)
						} else if d10.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d10)
						} else if d10.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d10)
						} else if d10.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d10.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d10 = tmpPair
					} else if d10.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d10.Type, Reg: ctx.AllocRegExcept(d10.Reg), Reg2: ctx.AllocRegExcept(d10.Reg)}
						switch d10.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d10)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d10)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d10)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d10)
						d10 = tmpPair
					}
					if d10.Loc != LocRegPair && d10.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (asAssoc arg0)")
					}
					d11 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("mapkey_assoc_mut")}
					ctx.EnsureDesc(&d11)
					if d11.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d11.Imm)
						ptrWord, _ := d11.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d11.Imm.String())))
						d11 = tmpPair
					} else if d11.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d11.Type, Reg: ctx.AllocRegExcept(d11.Reg), Reg2: ctx.AllocRegExcept(d11.Reg)}
						switch d11.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d11)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d11)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d11)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d11)
						d11 = tmpPair
					}
					if d11.Loc != LocRegPair && d11.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (asAssoc arg1)")
					}
					ctx.SyncDesc(&d10)
					ctx.SyncDesc(&d11)
					callResults12 := JITEmitGoCallResults(ctx, GoFuncAddr(asAssoc), []JITValueDesc{d10, d11}, []uint8{3, 1}, []uint8{1, 1})
					ctx.FreeDesc(&d11)
					d13 = callResults12[0]
					_ = d13
					d14 = callResults12[1]
					_ = d14
					ctx.FreeDesc(&d10)
					ctx.StabilizeDescForControlFlow(&d13)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.EnsureDesc(&d14)
					var d15 JITValueDesc
					if d14.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d14)
						if d14.Loc != LocReg && d14.Loc != LocRegPair && d14.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d14.Reg)
						ctx.EmitCmpRegImm32(d14.Reg, 0)
						ctx.EmitSetcc(r0, CondEqual)
						d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d15)
					}
					d16 = d15
					ctx.EnsureDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							if ps.General {
							}
							ps17 := PhiState{General: ps.General}
							ps17.OverlayValues = make([]JITValueDesc, 17)
							ps17.OverlayValues[1] = d1
							ps17.OverlayValues[2] = d2
							ps17.OverlayValues[3] = d3
							ps17.OverlayValues[4] = d4
							ps17.OverlayValues[7] = d7
							ps17.OverlayValues[8] = d8
							ps17.OverlayValues[10] = d10
							ps17.OverlayValues[11] = d11
							ps17.OverlayValues[13] = d13
							ps17.OverlayValues[14] = d14
							ps17.OverlayValues[15] = d15
							ps17.OverlayValues[16] = d16
							return bbs[1].RenderPS(ps17)
						}
						if ps.General {
						}
						ps18 := PhiState{General: ps.General}
						ps18.OverlayValues = make([]JITValueDesc, 17)
						ps18.OverlayValues[1] = d1
						ps18.OverlayValues[2] = d2
						ps18.OverlayValues[3] = d3
						ps18.OverlayValues[4] = d4
						ps18.OverlayValues[7] = d7
						ps18.OverlayValues[8] = d8
						ps18.OverlayValues[10] = d10
						ps18.OverlayValues[11] = d11
						ps18.OverlayValues[13] = d13
						ps18.OverlayValues[14] = d14
						ps18.OverlayValues[15] = d15
						ps18.OverlayValues[16] = d16
						return bbs[2].RenderPS(ps18)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl7 := ctx.ReserveLabel()
					lbl8 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d16.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl7)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl3)
					ps19 := PhiState{General: true}
					ps19.OverlayValues = make([]JITValueDesc, 17)
					ps19.OverlayValues[1] = d1
					ps19.OverlayValues[2] = d2
					ps19.OverlayValues[3] = d3
					ps19.OverlayValues[4] = d4
					ps19.OverlayValues[7] = d7
					ps19.OverlayValues[8] = d8
					ps19.OverlayValues[10] = d10
					ps19.OverlayValues[11] = d11
					ps19.OverlayValues[13] = d13
					ps19.OverlayValues[14] = d14
					ps19.OverlayValues[15] = d15
					ps19.OverlayValues[16] = d16
					ps20 := PhiState{General: true}
					ps20.OverlayValues = make([]JITValueDesc, 17)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[8] = d8
					ps20.OverlayValues[10] = d10
					ps20.OverlayValues[11] = d11
					ps20.OverlayValues[13] = d13
					ps20.OverlayValues[14] = d14
					ps20.OverlayValues[15] = d15
					ps20.OverlayValues[16] = d16
					snap21 := d1
					snap22 := d2
					snap23 := d3
					snap24 := d4
					snap25 := d7
					snap26 := d8
					snap27 := d10
					snap28 := d11
					snap29 := d13
					snap30 := d14
					snap31 := d15
					snap32 := d16
					alloc33 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps20)
					}
					ctx.RestoreAllocState(alloc33)
					d1 = snap21
					d2 = snap22
					d3 = snap23
					d4 = snap24
					d7 = snap25
					d8 = snap26
					d10 = snap27
					d11 = snap28
					d13 = snap29
					d14 = snap30
					d15 = snap31
					d16 = snap32
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps19)
					}
					return result
					ctx.FreeDesc(&d15)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
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
					ctx.ReclaimUntrackedRegs()
					stackArray34 = ctx.AllocStack(int32(0))
					_ = stackArray34
					d35 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d35
					callResults36 := JITEmitGoCallResults(ctx, GoFuncAddr(JITCloneScmerSlice), []JITValueDesc{d13}, []uint8{3}, []uint8{1})
					d37 = callResults36[0]
					ctx.StabilizeDescForControlFlow(&d37)
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d13)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d39)
					var d41 JITValueDesc
					if d39.Loc == LocImm && d38.Loc == LocImm {
						d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d39.Imm.Int() - d38.Imm.Int())}
					} else {
						r1 := ctx.AllocReg()
						if d39.Loc == LocImm {
							ctx.EmitMovRegImm64(r1, uint64(d39.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r1, d39.Reg)
						}
						if d38.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d38.Imm.Int()))
							ctx.EmitSubInt64(r1, RegR11)
						} else {
							ctx.EmitSubInt64(r1, d38.Reg)
						}
						d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d41)
					}
					var d42 JITValueDesc
					if d13.Loc == LocImm && d38.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d13.Imm.Int() + d38.Imm.Int()*16)}
					} else {
						r2 := ctx.AllocReg()
						if d13.Loc == LocImm {
							ctx.EmitMovRegImm64(r2, uint64(d13.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r2, d13.Reg)
						}
						if d38.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d38.Imm.Int()*16))
							ctx.EmitAddInt64(r2, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r2, d38.Reg)
							ctx.EmitMovRegReg(offsetReg, d38.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r2, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d42)
					}
					var d43 JITValueDesc
					var r3 Reg
					var r4 Reg
					ctx.SyncDesc(&d42)
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						r3 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r3, uint64(d42.Imm.Int()))
					} else {
						r3 = d42.Reg
					}
					ctx.ProtectReg(r3)
					ctx.SyncDesc(&d41)
					ctx.EnsureDesc(&d41)
					if d41.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d41.Imm.Int()))
					} else {
						r4 = d41.Reg
					}
					ctx.ProtectReg(r4)
					r5 := ctx.EmitSliceCapAfterLow(&d13, &d38, r3, r4)
					ctx.UnprotectReg(r4)
					ctx.UnprotectReg(r3)
					d43 = JITValueDesc{Loc: LocRegTriple, Reg: r3, Reg2: r4, Reg3: r5}
					ctx.BindReg(r3, &d43)
					ctx.BindReg(r4, &d43)
					ctx.BindReg(r5, &d43)
					ctx.BindReg(r3, &d43)
					ctx.BindReg(r4, &d43)
					ctx.BindReg(r5, &d43)
					d44 = ctx.EmitNewSliceFromGoSlice(&d43)
					ctx.StabilizeDescForControlFlow(&d44)
					if ps.General {
						ctx.SyncDesc(&d44)
						if d44.Loc == LocReg {
							ctx.ProtectReg(d44.Reg)
						} else if d44.Loc == LocRegPair {
							ctx.ProtectReg(d44.Reg)
							ctx.ProtectReg(d44.Reg2)
						}
						d45 = d44
						if d45.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d45)
						if d45.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d45, int32(bbs[3].PhiBase)+int32(0), 2)
						} else if d45.Loc == LocInputPair {
							ctx.EnsureDesc(&d45)
							ctx.EmitStoreScmerToStack(d45, int32(bbs[3].PhiBase)+int32(0))
						} else if d45.Loc == LocRegPair || d45.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d45, int32(bbs[3].PhiBase)+int32(0))
						} else {
							ctx.EnsureDesc(&d45)
							ctx.EmitStoreToStack(d45, int32(bbs[3].PhiBase)+int32(0))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[3].PhiBase)+int32(0))+8)
						}
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(16))
						if d44.Loc == LocReg {
							ctx.UnprotectReg(d44.Reg)
						} else if d44.Loc == LocRegPair {
							ctx.UnprotectReg(d44.Reg)
							ctx.UnprotectReg(d44.Reg2)
						}
					}
					ps46 := PhiState{General: ps.General}
					ps46.OverlayValues = make([]JITValueDesc, 46)
					ps46.OverlayValues[1] = d1
					ps46.OverlayValues[2] = d2
					ps46.OverlayValues[3] = d3
					ps46.OverlayValues[4] = d4
					ps46.OverlayValues[7] = d7
					ps46.OverlayValues[8] = d8
					ps46.OverlayValues[10] = d10
					ps46.OverlayValues[11] = d11
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[15] = d15
					ps46.OverlayValues[16] = d16
					ps46.OverlayValues[35] = d35
					ps46.OverlayValues[37] = d37
					ps46.OverlayValues[38] = d38
					ps46.OverlayValues[39] = d39
					ps46.OverlayValues[40] = d40
					ps46.OverlayValues[41] = d41
					ps46.OverlayValues[42] = d42
					ps46.OverlayValues[43] = d43
					ps46.OverlayValues[44] = d44
					ps46.OverlayValues[45] = d45
					ps46.PhiValues = make([]JITValueDesc, 2)
					d47 = d44
					ps46.PhiValues[0] = d47
					d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps46.PhiValues[1] = d48
					if ps46.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps46)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
						d47 = ps.OverlayValues[47]
					}
					if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
						d48 = ps.OverlayValues[48]
					}
					ctx.ReclaimUntrackedRegs()
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					ctx.EmitMovRegImm64(r6, 0)
					ctx.EmitMovRegImm64(r7, 0)
					d49 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d49)
					ctx.BindReg(r7, &d49)
					d50 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: nil}
					ctx.SyncDesc(&d50)
					ctx.EnsureDesc(&d14)
					phiBase51 = ctx.AllocStack(int32(16))
					d52 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase51) + int32(0)}
					_ = d52
					lbl9 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					bbpos_1_4 := int32(-1)
					_ = bbpos_1_4
					bbpos_1_5 := int32(-1)
					_ = bbpos_1_5
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d52 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase51) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase51)+int32(0))
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d52 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase51) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d52)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d53 JITValueDesc
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						fieldAddr := uintptr(d14.Imm.Int()) + 0
						r8 := ctx.AllocReg()
						r9 := ctx.AllocRegExcept(r8)
						r10 := ctx.AllocRegExcept(r8, r9)
						ctx.EmitMovRegMem64(r8, fieldAddr)
						ctx.EmitMovRegMem64(r9, fieldAddr+8)
						ctx.EmitMovRegMem64(r10, fieldAddr+16)
						d53 = JITValueDesc{Loc: LocRegTriple, Reg: r8, Reg2: r9, Reg3: r10}
						ctx.BindReg(r8, &d53)
						ctx.BindReg(r9, &d53)
						ctx.BindReg(r10, &d53)
					} else {
						off := int32(0)
						baseReg := d14.Reg
						r11 := ctx.AllocRegExcept(baseReg)
						r12 := ctx.AllocRegExcept(baseReg, r11)
						r13 := ctx.AllocRegExcept(baseReg, r11, r12)
						ctx.EmitMovRegMem(r11, baseReg, off)
						ctx.EmitMovRegMem(r12, baseReg, off+8)
						ctx.EmitMovRegMem(r13, baseReg, off+16)
						d53 = JITValueDesc{Loc: LocRegTriple, Reg: r11, Reg2: r12, Reg3: r13}
						ctx.BindReg(r11, &d53)
						ctx.BindReg(r12, &d53)
						ctx.BindReg(r13, &d53)
					}
					ctx.ReclaimUntrackedRegs()
					var d54 JITValueDesc
					if d53.SliceSizeKnown {
						d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.KnownSliceLen))}
					} else if d53.Loc == LocImm {
						d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d53.StackOff))}
					} else if d53.Loc == LocStackTriple {
						d54 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d53.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d53)
						if d53.Loc == LocRegPair || d53.Loc == LocRegTriple {
							d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg2, ID: 0}
						} else if d53.Loc == LocReg {
							d54 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d53.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d54)
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d54)
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d54)
					var d55 JITValueDesc
					if d52.Loc == LocImm && d54.Loc == LocImm {
						d55 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d52.Imm.Int() < d54.Imm.Int())}
					} else if d54.Loc == LocImm {
						r14 := ctx.AllocRegExcept(d52.Reg)
						if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d52.Reg, int32(d54.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d54.Imm.Int()))
							ctx.EmitCmpInt64(d52.Reg, RegR11)
						}
						ctx.EmitSetcc(r14, CondSignedLess)
						d55 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d55)
					} else if d52.Loc == LocImm {
						r15 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d52.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d54.Reg)
						ctx.EmitSetcc(r15, CondSignedLess)
						d55 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
						ctx.BindReg(r15, &d55)
					} else {
						r16 := ctx.AllocRegExcept(d52.Reg)
						ctx.EmitCmpInt64(d52.Reg, d54.Reg)
						ctx.EmitSetcc(r16, CondSignedLess)
						d55 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
						ctx.BindReg(r16, &d55)
					}
					ctx.FreeDesc(&d54)
					ctx.ReclaimUntrackedRegs()
					d56 = d55
					ctx.EnsureDesc(&d56)
					if d56.Loc != LocImm && d56.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					if d56.Loc == LocImm {
						if d56.Imm.Bool() {
							ctx.MarkLabel(lbl12)
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl13)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitCmpRegImm32(d56.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl12)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl12)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl13)
						ctx.EmitJmp(lbl11)
					}
					ctx.FreeDesc(&d55)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d52 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase51) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d52 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase51) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d57 JITValueDesc
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						fieldAddr := uintptr(d14.Imm.Int()) + 0
						r17 := ctx.AllocReg()
						r18 := ctx.AllocRegExcept(r17)
						r19 := ctx.AllocRegExcept(r17, r18)
						ctx.EmitMovRegMem64(r17, fieldAddr)
						ctx.EmitMovRegMem64(r18, fieldAddr+8)
						ctx.EmitMovRegMem64(r19, fieldAddr+16)
						d57 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
						ctx.BindReg(r17, &d57)
						ctx.BindReg(r18, &d57)
						ctx.BindReg(r19, &d57)
					} else {
						off := int32(0)
						baseReg := d14.Reg
						r20 := ctx.AllocRegExcept(baseReg)
						r21 := ctx.AllocRegExcept(baseReg, r20)
						r22 := ctx.AllocRegExcept(baseReg, r20, r21)
						ctx.EmitMovRegMem(r20, baseReg, off)
						ctx.EmitMovRegMem(r21, baseReg, off+8)
						ctx.EmitMovRegMem(r22, baseReg, off+16)
						d57 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
						ctx.BindReg(r20, &d57)
						ctx.BindReg(r21, &d57)
						ctx.BindReg(r22, &d57)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d52)
					ctx.ReclaimUntrackedRegs()
					d59 = ctx.EmitSliceElementAddress(&d57, &d52, 16)
					ctx.EnsureDesc(&d59)
					r23 := ctx.AllocRegExcept(d59.Reg)
					ctx.EmitMovRegMem(r23, d59.Reg, 8)
					ctx.EmitMovRegMem(d59.Reg, d59.Reg, 0)
					d58 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d59.Reg, Reg2: r23}
					ctx.BindReg(d59.Reg, &d58)
					ctx.BindReg(r23, &d58)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d60 JITValueDesc
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						fieldAddr := uintptr(d14.Imm.Int()) + 0
						r24 := ctx.AllocReg()
						r25 := ctx.AllocRegExcept(r24)
						r26 := ctx.AllocRegExcept(r24, r25)
						ctx.EmitMovRegMem64(r24, fieldAddr)
						ctx.EmitMovRegMem64(r25, fieldAddr+8)
						ctx.EmitMovRegMem64(r26, fieldAddr+16)
						d60 = JITValueDesc{Loc: LocRegTriple, Reg: r24, Reg2: r25, Reg3: r26}
						ctx.BindReg(r24, &d60)
						ctx.BindReg(r25, &d60)
						ctx.BindReg(r26, &d60)
					} else {
						off := int32(0)
						baseReg := d14.Reg
						r27 := ctx.AllocRegExcept(baseReg)
						r28 := ctx.AllocRegExcept(baseReg, r27)
						r29 := ctx.AllocRegExcept(baseReg, r27, r28)
						ctx.EmitMovRegMem(r27, baseReg, off)
						ctx.EmitMovRegMem(r28, baseReg, off+8)
						ctx.EmitMovRegMem(r29, baseReg, off+16)
						d60 = JITValueDesc{Loc: LocRegTriple, Reg: r27, Reg2: r28, Reg3: r29}
						ctx.BindReg(r27, &d60)
						ctx.BindReg(r28, &d60)
						ctx.BindReg(r29, &d60)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d52)
					var d61 JITValueDesc
					if d52.Loc == LocImm {
						d61 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d52.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d52.Reg)
						ctx.EmitMovRegReg(scratch, d52.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d61 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d61)
					}
					if d61.Loc == LocReg && d52.Loc == LocReg && d61.Reg == d52.Reg {
						ctx.TransferReg(d52.Reg)
						d52.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d61)
					ctx.ReclaimUntrackedRegs()
					d63 = ctx.EmitSliceElementAddress(&d60, &d61, 16)
					ctx.EnsureDesc(&d63)
					r30 := ctx.AllocRegExcept(d63.Reg)
					ctx.EmitMovRegMem(r30, d63.Reg, 8)
					ctx.EmitMovRegMem(d63.Reg, d63.Reg, 0)
					d62 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d63.Reg, Reg2: r30}
					ctx.BindReg(d63.Reg, &d62)
					ctx.BindReg(r30, &d62)
					ctx.FreeDesc(&d61)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d58)
					ctx.EnsureDesc(&d62)
					d64 = d58
					_ = d64
					ctx.StabilizeDescForControlFlow(&d64)
					d65 = d62
					_ = d65
					ctx.StabilizeDescForControlFlow(&d65)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d66 = d8
					_ = d66
					ctx.ReclaimUntrackedRegs()
					d67 = d50
					_ = d67
					ctx.ReclaimUntrackedRegs()
					d68 = d4
					_ = d68
					ctx.ReclaimUntrackedRegs()
					stackArray69 = ctx.AllocStack(int32(32))
					_ = stackArray69
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d64)
					ctx.EnsureDesc(&d64)
					ctx.EmitStoreScmerToStack(d64, int32(stackArray69)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EmitStoreScmerToStack(d65, int32(stackArray69)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d70 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d70
					ctx.ReclaimUntrackedRegs()
					callbackArgs72 := make([]JITValueDesc, 2)
					callbackArgs72[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray69) + 0}
					callbackArgs72[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray69) + 16}
					var d71 JITValueDesc
					callbackResultOff73 = ctx.AllocStack(16)
					ctx.FreeDesc(&d70)
					if d68.Loc == LocLambdaTemplate && d68.Lambda != nil {
						stableCallbackArgs74 := ctx.StabilizeCallbackArgs(callbackArgs72)
						ctx.ReclaimUntrackedRegs()
						outerRegs75 := ctx.PreserveOuterRegs()
						d71 = JITEmitProcInlineWithOuter(ctx, &d68.Lambda.Proc, d68.Lambda.Outer, stableCallbackArgs74, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff73), ID: 0})
						ctx.RestoreOuterRegs(outerRegs75)
						ctx.ReclaimUntrackedRegs()
					} else {
						d76, knownBuiltin77 := jitEmitKnownDeclaration(ctx, d68, callbackArgs72, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff73), ID: 0})
						if knownBuiltin77 {
							d71 = d76
						} else {
							d78 := jitCopyScmerToPair(ctx, d68)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d78)
							callbackCallArgs = append(callbackCallArgs, callbackArgs72...)
							d71 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d71, int32(callbackResultOff73))
							ctx.FreeDesc(&d71)
							d71 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff73), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					stackArray79 = ctx.AllocStack(int32(48))
					_ = stackArray79
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d67)
					ctx.EnsureDesc(&d67)
					d80 = jitMaterializeVirtualSlice(ctx, d67, JITValueDesc{Loc: LocAny})
					ctx.EmitStoreScmerToStack(d80, int32(stackArray79)+int32(0))
					ctx.FreeDesc(&d80)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d71)
					ctx.EnsureDesc(&d71)
					ctx.EmitStoreScmerToStack(d71, int32(stackArray79)+int32(16))
					ctx.FreeDesc(&d71)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d65)
					ctx.EnsureDesc(&d65)
					ctx.EmitStoreScmerToStack(d65, int32(stackArray79)+int32(32))
					ctx.ReclaimUntrackedRegs()
					d81 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d81
					ctx.ReclaimUntrackedRegs()
					callbackArgs83 := make([]JITValueDesc, 3)
					callbackArgs83[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray79) + 0}
					callbackArgs83[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray79) + 16}
					callbackArgs83[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray79) + 32}
					var d82 JITValueDesc
					callbackResultOff84 = ctx.AllocStack(16)
					ctx.FreeDesc(&d81)
					if d66.Loc == LocLambdaTemplate && d66.Lambda != nil {
						stableCallbackArgs85 := ctx.StabilizeCallbackArgs(callbackArgs83)
						ctx.ReclaimUntrackedRegs()
						outerRegs86 := ctx.PreserveOuterRegs()
						d82 = JITEmitProcInlineWithOuter(ctx, &d66.Lambda.Proc, d66.Lambda.Outer, stableCallbackArgs85, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff84), ID: 0})
						ctx.RestoreOuterRegs(outerRegs86)
						ctx.ReclaimUntrackedRegs()
					} else {
						d87, knownBuiltin88 := jitEmitKnownDeclaration(ctx, d66, callbackArgs83, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff84), ID: 0})
						if knownBuiltin88 {
							d82 = d87
						} else {
							d89 := jitCopyScmerToPair(ctx, d66)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d89)
							callbackCallArgs = append(callbackCallArgs, callbackArgs83...)
							d82 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d82, int32(callbackResultOff84))
							ctx.FreeDesc(&d82)
							d82 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff84), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d82)
					ctx.FreeDesc(&d82)
					ctx.FreeDesc(&d82)
					ctx.ReclaimUntrackedRegs()
					d90 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.FreeDesc(&d58)
					ctx.FreeDesc(&d62)
					ctx.ReclaimUntrackedRegs()
					d91 = d90
					ctx.EnsureDesc(&d91)
					if d91.Loc != LocImm && d91.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					if d91.Loc == LocImm {
						if d91.Imm.Bool() {
							ctx.MarkLabel(lbl16)
							ctx.EmitJmp(lbl14)
						} else {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl15)
						}
					} else {
						ctx.EmitCmpRegImm32(d91.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl16)
						ctx.EmitJmp(lbl17)
						ctx.MarkLabel(lbl16)
						ctx.EmitJmp(lbl14)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl15)
					}
					ctx.FreeDesc(&d90)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl15)
					ctx.ResolveFixups()
					d52 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase51) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl14)
					ctx.ResolveFixups()
					d52 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase51) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d52)
					ctx.EnsureDesc(&d52)
					var d92 JITValueDesc
					if d52.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d52.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d52.Reg)
						ctx.EmitMovRegReg(scratch, d52.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d92)
					}
					if d92.Loc == LocReg && d52.Loc == LocReg && d92.Reg == d52.Reg {
						ctx.TransferReg(d52.Reg)
						d52.Loc = LocNone
					}
					ctx.EnsureDesc(&d92)
					ctx.EmitStoreToStack(d92, int32(phiBase51)+int32(0))
					ctx.StabilizeDescForControlFlow(&d92)
					ctx.FreeDesc(&d52)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_1_1)
					ctx.MarkLabel(lbl9)
					ctx.FreeDesc(&d14)
					d93 = d50
					_ = d93
					ctx.EnsureDesc(&d93)
					if d93.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d93, &result)
						result.Type = d93.Type
					} else {
						switch d93.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d93)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d93)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d93)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d93, &result)
							result.Type = d93.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d94 := ps.PhiValues[0]
							ctx.EnsureDesc(&d94)
							ctx.EmitStoreScmerToStack(d94, int32(bbs[3].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d95 := ps.PhiValues[1]
							ctx.EnsureDesc(&d95)
							ctx.EmitStoreToStack(d95, int32(bbs[3].PhiBase)+int32(16))
						}
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
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
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
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
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d2 = ps.PhiValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs96 := make([]Reg, 0, 3)
					seenBlockPinnedRegs97 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs97
					for _, r := range []Reg{d37.Reg, d37.Reg2, d37.Reg3} {
						live := d37.Loc == LocRegTriple && (r == d37.Reg || r == d37.Reg2 || r == d37.Reg3)
						if live && !seenBlockPinnedRegs97[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs97[r] = true
							blockPinnedRegs96 = append(blockPinnedRegs96, r)
						}
					}
					unpinBlockRegs98 := func() {
						for _, r := range blockPinnedRegs96 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs98()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					var d99 JITValueDesc
					if d37.SliceSizeKnown {
						d99 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.KnownSliceLen))}
					} else if d37.Loc == LocImm {
						d99 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d37.StackOff))}
					} else if d37.Loc == LocStackTriple {
						d99 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d37.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d37)
						if d37.Loc == LocRegPair || d37.Loc == LocRegTriple {
							d99 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg2, ID: 0}
						} else if d37.Loc == LocReg {
							d99 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d37.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d99)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d99)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d99)
					var d100 JITValueDesc
					if d2.Loc == LocImm && d99.Loc == LocImm {
						d100 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d99.Imm.Int())}
					} else if d99.Loc == LocImm {
						r31 := ctx.AllocRegExcept(d2.Reg)
						if d99.Imm.Int() >= -2147483648 && d99.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d99.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d99.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, RegR11)
						}
						ctx.EmitSetcc(r31, CondSignedLess)
						d100 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r31}
						ctx.BindReg(r31, &d100)
					} else if d2.Loc == LocImm {
						r32 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d99.Reg)
						ctx.EmitSetcc(r32, CondSignedLess)
						d100 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r32}
						ctx.BindReg(r32, &d100)
					} else {
						r33 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d99.Reg)
						ctx.EmitSetcc(r33, CondSignedLess)
						d100 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r33}
						ctx.BindReg(r33, &d100)
					}
					ctx.FreeDesc(&d99)
					d101 = d100
					ctx.EnsureDesc(&d101)
					if d101.Loc != LocImm && d101.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d101.Loc == LocImm {
						if d101.Imm.Bool() {
							if ps.General {
							}
							ps102 := PhiState{General: ps.General}
							ps102.OverlayValues = make([]JITValueDesc, 102)
							ps102.OverlayValues[1] = d1
							ps102.OverlayValues[2] = d2
							ps102.OverlayValues[3] = d3
							ps102.OverlayValues[4] = d4
							ps102.OverlayValues[7] = d7
							ps102.OverlayValues[8] = d8
							ps102.OverlayValues[10] = d10
							ps102.OverlayValues[11] = d11
							ps102.OverlayValues[13] = d13
							ps102.OverlayValues[14] = d14
							ps102.OverlayValues[15] = d15
							ps102.OverlayValues[16] = d16
							ps102.OverlayValues[35] = d35
							ps102.OverlayValues[37] = d37
							ps102.OverlayValues[38] = d38
							ps102.OverlayValues[39] = d39
							ps102.OverlayValues[40] = d40
							ps102.OverlayValues[41] = d41
							ps102.OverlayValues[42] = d42
							ps102.OverlayValues[43] = d43
							ps102.OverlayValues[44] = d44
							ps102.OverlayValues[45] = d45
							ps102.OverlayValues[47] = d47
							ps102.OverlayValues[48] = d48
							ps102.OverlayValues[49] = d49
							ps102.OverlayValues[50] = d50
							ps102.OverlayValues[52] = d52
							ps102.OverlayValues[53] = d53
							ps102.OverlayValues[54] = d54
							ps102.OverlayValues[55] = d55
							ps102.OverlayValues[56] = d56
							ps102.OverlayValues[57] = d57
							ps102.OverlayValues[58] = d58
							ps102.OverlayValues[59] = d59
							ps102.OverlayValues[60] = d60
							ps102.OverlayValues[61] = d61
							ps102.OverlayValues[62] = d62
							ps102.OverlayValues[63] = d63
							ps102.OverlayValues[64] = d64
							ps102.OverlayValues[65] = d65
							ps102.OverlayValues[66] = d66
							ps102.OverlayValues[67] = d67
							ps102.OverlayValues[68] = d68
							ps102.OverlayValues[70] = d70
							ps102.OverlayValues[71] = d71
							ps102.OverlayValues[76] = d76
							ps102.OverlayValues[78] = d78
							ps102.OverlayValues[80] = d80
							ps102.OverlayValues[81] = d81
							ps102.OverlayValues[82] = d82
							ps102.OverlayValues[87] = d87
							ps102.OverlayValues[89] = d89
							ps102.OverlayValues[90] = d90
							ps102.OverlayValues[91] = d91
							ps102.OverlayValues[92] = d92
							ps102.OverlayValues[93] = d93
							ps102.OverlayValues[94] = d94
							ps102.OverlayValues[95] = d95
							ps102.OverlayValues[99] = d99
							ps102.OverlayValues[100] = d100
							ps102.OverlayValues[101] = d101
							return bbs[4].RenderPS(ps102)
						}
						if ps.General {
						}
						ps103 := PhiState{General: ps.General}
						ps103.OverlayValues = make([]JITValueDesc, 102)
						ps103.OverlayValues[1] = d1
						ps103.OverlayValues[2] = d2
						ps103.OverlayValues[3] = d3
						ps103.OverlayValues[4] = d4
						ps103.OverlayValues[7] = d7
						ps103.OverlayValues[8] = d8
						ps103.OverlayValues[10] = d10
						ps103.OverlayValues[11] = d11
						ps103.OverlayValues[13] = d13
						ps103.OverlayValues[14] = d14
						ps103.OverlayValues[15] = d15
						ps103.OverlayValues[16] = d16
						ps103.OverlayValues[35] = d35
						ps103.OverlayValues[37] = d37
						ps103.OverlayValues[38] = d38
						ps103.OverlayValues[39] = d39
						ps103.OverlayValues[40] = d40
						ps103.OverlayValues[41] = d41
						ps103.OverlayValues[42] = d42
						ps103.OverlayValues[43] = d43
						ps103.OverlayValues[44] = d44
						ps103.OverlayValues[45] = d45
						ps103.OverlayValues[47] = d47
						ps103.OverlayValues[48] = d48
						ps103.OverlayValues[49] = d49
						ps103.OverlayValues[50] = d50
						ps103.OverlayValues[52] = d52
						ps103.OverlayValues[53] = d53
						ps103.OverlayValues[54] = d54
						ps103.OverlayValues[55] = d55
						ps103.OverlayValues[56] = d56
						ps103.OverlayValues[57] = d57
						ps103.OverlayValues[58] = d58
						ps103.OverlayValues[59] = d59
						ps103.OverlayValues[60] = d60
						ps103.OverlayValues[61] = d61
						ps103.OverlayValues[62] = d62
						ps103.OverlayValues[63] = d63
						ps103.OverlayValues[64] = d64
						ps103.OverlayValues[65] = d65
						ps103.OverlayValues[66] = d66
						ps103.OverlayValues[67] = d67
						ps103.OverlayValues[68] = d68
						ps103.OverlayValues[70] = d70
						ps103.OverlayValues[71] = d71
						ps103.OverlayValues[76] = d76
						ps103.OverlayValues[78] = d78
						ps103.OverlayValues[80] = d80
						ps103.OverlayValues[81] = d81
						ps103.OverlayValues[82] = d82
						ps103.OverlayValues[87] = d87
						ps103.OverlayValues[89] = d89
						ps103.OverlayValues[90] = d90
						ps103.OverlayValues[91] = d91
						ps103.OverlayValues[92] = d92
						ps103.OverlayValues[93] = d93
						ps103.OverlayValues[94] = d94
						ps103.OverlayValues[95] = d95
						ps103.OverlayValues[99] = d99
						ps103.OverlayValues[100] = d100
						ps103.OverlayValues[101] = d101
						return bbs[5].RenderPS(ps103)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d104 := ps.PhiValues[0]
							ctx.EnsureDesc(&d104)
							ctx.EmitStoreScmerToStack(d104, int32(bbs[3].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d105 := ps.PhiValues[1]
							ctx.EnsureDesc(&d105)
							ctx.EmitStoreToStack(d105, int32(bbs[3].PhiBase)+int32(16))
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d101.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl6)
					ps106 := PhiState{General: true}
					ps106.OverlayValues = make([]JITValueDesc, 106)
					ps106.OverlayValues[1] = d1
					ps106.OverlayValues[2] = d2
					ps106.OverlayValues[3] = d3
					ps106.OverlayValues[4] = d4
					ps106.OverlayValues[7] = d7
					ps106.OverlayValues[8] = d8
					ps106.OverlayValues[10] = d10
					ps106.OverlayValues[11] = d11
					ps106.OverlayValues[13] = d13
					ps106.OverlayValues[14] = d14
					ps106.OverlayValues[15] = d15
					ps106.OverlayValues[16] = d16
					ps106.OverlayValues[35] = d35
					ps106.OverlayValues[37] = d37
					ps106.OverlayValues[38] = d38
					ps106.OverlayValues[39] = d39
					ps106.OverlayValues[40] = d40
					ps106.OverlayValues[41] = d41
					ps106.OverlayValues[42] = d42
					ps106.OverlayValues[43] = d43
					ps106.OverlayValues[44] = d44
					ps106.OverlayValues[45] = d45
					ps106.OverlayValues[47] = d47
					ps106.OverlayValues[48] = d48
					ps106.OverlayValues[49] = d49
					ps106.OverlayValues[50] = d50
					ps106.OverlayValues[52] = d52
					ps106.OverlayValues[53] = d53
					ps106.OverlayValues[54] = d54
					ps106.OverlayValues[55] = d55
					ps106.OverlayValues[56] = d56
					ps106.OverlayValues[57] = d57
					ps106.OverlayValues[58] = d58
					ps106.OverlayValues[59] = d59
					ps106.OverlayValues[60] = d60
					ps106.OverlayValues[61] = d61
					ps106.OverlayValues[62] = d62
					ps106.OverlayValues[63] = d63
					ps106.OverlayValues[64] = d64
					ps106.OverlayValues[65] = d65
					ps106.OverlayValues[66] = d66
					ps106.OverlayValues[67] = d67
					ps106.OverlayValues[68] = d68
					ps106.OverlayValues[70] = d70
					ps106.OverlayValues[71] = d71
					ps106.OverlayValues[76] = d76
					ps106.OverlayValues[78] = d78
					ps106.OverlayValues[80] = d80
					ps106.OverlayValues[81] = d81
					ps106.OverlayValues[82] = d82
					ps106.OverlayValues[87] = d87
					ps106.OverlayValues[89] = d89
					ps106.OverlayValues[90] = d90
					ps106.OverlayValues[91] = d91
					ps106.OverlayValues[92] = d92
					ps106.OverlayValues[93] = d93
					ps106.OverlayValues[94] = d94
					ps106.OverlayValues[95] = d95
					ps106.OverlayValues[99] = d99
					ps106.OverlayValues[100] = d100
					ps106.OverlayValues[101] = d101
					ps106.OverlayValues[104] = d104
					ps106.OverlayValues[105] = d105
					ps107 := PhiState{General: true}
					ps107.OverlayValues = make([]JITValueDesc, 106)
					ps107.OverlayValues[1] = d1
					ps107.OverlayValues[2] = d2
					ps107.OverlayValues[3] = d3
					ps107.OverlayValues[4] = d4
					ps107.OverlayValues[7] = d7
					ps107.OverlayValues[8] = d8
					ps107.OverlayValues[10] = d10
					ps107.OverlayValues[11] = d11
					ps107.OverlayValues[13] = d13
					ps107.OverlayValues[14] = d14
					ps107.OverlayValues[15] = d15
					ps107.OverlayValues[16] = d16
					ps107.OverlayValues[35] = d35
					ps107.OverlayValues[37] = d37
					ps107.OverlayValues[38] = d38
					ps107.OverlayValues[39] = d39
					ps107.OverlayValues[40] = d40
					ps107.OverlayValues[41] = d41
					ps107.OverlayValues[42] = d42
					ps107.OverlayValues[43] = d43
					ps107.OverlayValues[44] = d44
					ps107.OverlayValues[45] = d45
					ps107.OverlayValues[47] = d47
					ps107.OverlayValues[48] = d48
					ps107.OverlayValues[49] = d49
					ps107.OverlayValues[50] = d50
					ps107.OverlayValues[52] = d52
					ps107.OverlayValues[53] = d53
					ps107.OverlayValues[54] = d54
					ps107.OverlayValues[55] = d55
					ps107.OverlayValues[56] = d56
					ps107.OverlayValues[57] = d57
					ps107.OverlayValues[58] = d58
					ps107.OverlayValues[59] = d59
					ps107.OverlayValues[60] = d60
					ps107.OverlayValues[61] = d61
					ps107.OverlayValues[62] = d62
					ps107.OverlayValues[63] = d63
					ps107.OverlayValues[64] = d64
					ps107.OverlayValues[65] = d65
					ps107.OverlayValues[66] = d66
					ps107.OverlayValues[67] = d67
					ps107.OverlayValues[68] = d68
					ps107.OverlayValues[70] = d70
					ps107.OverlayValues[71] = d71
					ps107.OverlayValues[76] = d76
					ps107.OverlayValues[78] = d78
					ps107.OverlayValues[80] = d80
					ps107.OverlayValues[81] = d81
					ps107.OverlayValues[82] = d82
					ps107.OverlayValues[87] = d87
					ps107.OverlayValues[89] = d89
					ps107.OverlayValues[90] = d90
					ps107.OverlayValues[91] = d91
					ps107.OverlayValues[92] = d92
					ps107.OverlayValues[93] = d93
					ps107.OverlayValues[94] = d94
					ps107.OverlayValues[95] = d95
					ps107.OverlayValues[99] = d99
					ps107.OverlayValues[100] = d100
					ps107.OverlayValues[101] = d101
					ps107.OverlayValues[104] = d104
					ps107.OverlayValues[105] = d105
					snap108 := d1
					snap109 := d2
					snap110 := d3
					snap111 := d4
					snap112 := d7
					snap113 := d8
					snap114 := d10
					snap115 := d11
					snap116 := d13
					snap117 := d14
					snap118 := d15
					snap119 := d16
					snap120 := d35
					snap121 := d37
					snap122 := d38
					snap123 := d39
					snap124 := d40
					snap125 := d41
					snap126 := d42
					snap127 := d43
					snap128 := d44
					snap129 := d45
					snap130 := d47
					snap131 := d48
					snap132 := d49
					snap133 := d50
					snap134 := d52
					snap135 := d53
					snap136 := d54
					snap137 := d55
					snap138 := d56
					snap139 := d57
					snap140 := d58
					snap141 := d59
					snap142 := d60
					snap143 := d61
					snap144 := d62
					snap145 := d63
					snap146 := d64
					snap147 := d65
					snap148 := d66
					snap149 := d67
					snap150 := d68
					snap151 := d70
					snap152 := d71
					snap153 := d76
					snap154 := d78
					snap155 := d80
					snap156 := d81
					snap157 := d82
					snap158 := d87
					snap159 := d89
					snap160 := d90
					snap161 := d91
					snap162 := d92
					snap163 := d93
					snap164 := d94
					snap165 := d95
					snap166 := d99
					snap167 := d100
					snap168 := d101
					snap169 := d104
					snap170 := d105
					alloc171 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps107)
					}
					ctx.RestoreAllocState(alloc171)
					d1 = snap108
					d2 = snap109
					d3 = snap110
					d4 = snap111
					d7 = snap112
					d8 = snap113
					d10 = snap114
					d11 = snap115
					d13 = snap116
					d14 = snap117
					d15 = snap118
					d16 = snap119
					d35 = snap120
					d37 = snap121
					d38 = snap122
					d39 = snap123
					d40 = snap124
					d41 = snap125
					d42 = snap126
					d43 = snap127
					d44 = snap128
					d45 = snap129
					d47 = snap130
					d48 = snap131
					d49 = snap132
					d50 = snap133
					d52 = snap134
					d53 = snap135
					d54 = snap136
					d55 = snap137
					d56 = snap138
					d57 = snap139
					d58 = snap140
					d59 = snap141
					d60 = snap142
					d61 = snap143
					d62 = snap144
					d63 = snap145
					d64 = snap146
					d65 = snap147
					d66 = snap148
					d67 = snap149
					d68 = snap150
					d70 = snap151
					d71 = snap152
					d76 = snap153
					d78 = snap154
					d80 = snap155
					d81 = snap156
					d82 = snap157
					d87 = snap158
					d89 = snap159
					d90 = snap160
					d91 = snap161
					d92 = snap162
					d93 = snap163
					d94 = snap164
					d95 = snap165
					d99 = snap166
					d100 = snap167
					d101 = snap168
					d104 = snap169
					d105 = snap170
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps106)
					}
					return result
					ctx.FreeDesc(&d100)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
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
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
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
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs172 := make([]Reg, 0, 3)
					seenBlockPinnedRegs173 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs173
					for _, r := range []Reg{d37.Reg, d37.Reg2, d37.Reg3} {
						live := d37.Loc == LocRegTriple && (r == d37.Reg || r == d37.Reg2 || r == d37.Reg3)
						if live && !seenBlockPinnedRegs173[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs173[r] = true
							blockPinnedRegs172 = append(blockPinnedRegs172, r)
						}
					}
					unpinBlockRegs174 := func() {
						for _, r := range blockPinnedRegs172 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs174()
					d175 = d8
					_ = d175
					d176 = d4
					_ = d176
					ctx.EnsureDesc(&d2)
					d178 = ctx.EmitSliceElementAddress(&d37, &d2, 16)
					ctx.EnsureDesc(&d178)
					r34 := ctx.AllocRegExcept(d178.Reg)
					ctx.EmitMovRegMem(r34, d178.Reg, 8)
					ctx.EmitMovRegMem(d178.Reg, d178.Reg, 0)
					d177 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d178.Reg, Reg2: r34}
					ctx.BindReg(d178.Reg, &d177)
					ctx.BindReg(r34, &d177)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d179 JITValueDesc
					if d2.Loc == LocImm {
						d179 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d179 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d179)
					}
					if d179.Loc == LocReg && d2.Loc == LocReg && d179.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d179)
					d181 = ctx.EmitSliceElementAddress(&d37, &d179, 16)
					ctx.EnsureDesc(&d181)
					r35 := ctx.AllocRegExcept(d181.Reg)
					ctx.EmitMovRegMem(r35, d181.Reg, 8)
					ctx.EmitMovRegMem(d181.Reg, d181.Reg, 0)
					d180 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d181.Reg, Reg2: r35}
					ctx.BindReg(d181.Reg, &d180)
					ctx.BindReg(r35, &d180)
					ctx.FreeDesc(&d179)
					stackArray182 = ctx.AllocStack(int32(32))
					_ = stackArray182
					ctx.EnsureDesc(&d177)
					ctx.EnsureDesc(&d177)
					ctx.EmitStoreScmerToStack(d177, int32(stackArray182)+int32(0))
					ctx.FreeDesc(&d177)
					ctx.EnsureDesc(&d180)
					ctx.EnsureDesc(&d180)
					ctx.EmitStoreScmerToStack(d180, int32(stackArray182)+int32(16))
					ctx.FreeDesc(&d180)
					d183 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d183
					callbackArgs185 := make([]JITValueDesc, 2)
					callbackArgs185[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray182) + 0}
					callbackArgs185[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray182) + 16}
					var d184 JITValueDesc
					callbackResultOff186 = ctx.AllocStack(16)
					ctx.FreeDesc(&d183)
					if d176.Loc == LocLambdaTemplate && d176.Lambda != nil {
						stableCallbackArgs187 := ctx.StabilizeCallbackArgs(callbackArgs185)
						ctx.ReclaimUntrackedRegs()
						outerRegs188 := ctx.PreserveOuterRegs()
						d184 = JITEmitProcInlineWithOuter(ctx, &d176.Lambda.Proc, d176.Lambda.Outer, stableCallbackArgs187, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff186), ID: 0})
						ctx.RestoreOuterRegs(outerRegs188)
						ctx.ReclaimUntrackedRegs()
					} else {
						d189, knownBuiltin190 := jitEmitKnownDeclaration(ctx, d176, callbackArgs185, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff186), ID: 0})
						if knownBuiltin190 {
							d184 = d189
						} else {
							d191 := jitCopyScmerToPair(ctx, d176)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d191)
							callbackCallArgs = append(callbackCallArgs, callbackArgs185...)
							d184 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d184, int32(callbackResultOff186))
							ctx.FreeDesc(&d184)
							d184 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff186), ID: 0}
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d192 JITValueDesc
					if d2.Loc == LocImm {
						d192 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d192 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d192)
					}
					if d192.Loc == LocReg && d2.Loc == LocReg && d192.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d192)
					d194 = ctx.EmitSliceElementAddress(&d37, &d192, 16)
					ctx.EnsureDesc(&d194)
					r36 := ctx.AllocRegExcept(d194.Reg)
					ctx.EmitMovRegMem(r36, d194.Reg, 8)
					ctx.EmitMovRegMem(d194.Reg, d194.Reg, 0)
					d193 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d194.Reg, Reg2: r36}
					ctx.BindReg(d194.Reg, &d193)
					ctx.BindReg(r36, &d193)
					ctx.FreeDesc(&d192)
					stackArray195 = ctx.AllocStack(int32(48))
					_ = stackArray195
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EmitStoreScmerToStack(d1, int32(stackArray195)+int32(0))
					ctx.EnsureDesc(&d184)
					ctx.EnsureDesc(&d184)
					ctx.EmitStoreScmerToStack(d184, int32(stackArray195)+int32(16))
					ctx.FreeDesc(&d184)
					ctx.EnsureDesc(&d193)
					ctx.EnsureDesc(&d193)
					ctx.EmitStoreScmerToStack(d193, int32(stackArray195)+int32(32))
					ctx.FreeDesc(&d193)
					d196 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d196
					callbackArgs198 := make([]JITValueDesc, 3)
					callbackArgs198[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray195) + 0}
					callbackArgs198[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray195) + 16}
					callbackArgs198[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray195) + 32}
					var d197 JITValueDesc
					ctx.FreeDesc(&d196)
					if d175.Loc == LocLambdaTemplate && d175.Lambda != nil {
						stableCallbackArgs199 := ctx.StabilizeCallbackArgs(callbackArgs198)
						ctx.ReclaimUntrackedRegs()
						outerRegs200 := ctx.PreserveOuterRegs()
						d197 = JITEmitProcInlineWithOuter(ctx, &d175.Lambda.Proc, d175.Lambda.Outer, stableCallbackArgs199, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0})
						ctx.RestoreOuterRegs(outerRegs200)
						ctx.ReclaimUntrackedRegs()
					} else {
						d201, knownBuiltin202 := jitEmitKnownDeclaration(ctx, d175, callbackArgs198, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0})
						if knownBuiltin202 {
							d197 = d201
						} else {
							d203 := jitCopyScmerToPair(ctx, d175)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d203)
							callbackCallArgs = append(callbackCallArgs, callbackArgs198...)
							d197 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d197, int32(bbs[3].PhiBase)+int32(0))
							ctx.FreeDesc(&d197)
							d197 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0}
						}
					}
					ctx.StabilizeDescForControlFlow(&d197)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d204 JITValueDesc
					if d2.Loc == LocImm {
						d204 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d204 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d204)
					}
					if d204.Loc == LocReg && d2.Loc == LocReg && d204.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d204)
					ctx.EmitStoreToStack(d204, int32(bbs[3].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d204)
					ctx.FreeDesc(&d2)
					if ps.General {
					}
					ps205 := PhiState{General: ps.General}
					ps205.OverlayValues = make([]JITValueDesc, 205)
					ps205.OverlayValues[1] = d1
					ps205.OverlayValues[2] = d2
					ps205.OverlayValues[3] = d3
					ps205.OverlayValues[4] = d4
					ps205.OverlayValues[7] = d7
					ps205.OverlayValues[8] = d8
					ps205.OverlayValues[10] = d10
					ps205.OverlayValues[11] = d11
					ps205.OverlayValues[13] = d13
					ps205.OverlayValues[14] = d14
					ps205.OverlayValues[15] = d15
					ps205.OverlayValues[16] = d16
					ps205.OverlayValues[35] = d35
					ps205.OverlayValues[37] = d37
					ps205.OverlayValues[38] = d38
					ps205.OverlayValues[39] = d39
					ps205.OverlayValues[40] = d40
					ps205.OverlayValues[41] = d41
					ps205.OverlayValues[42] = d42
					ps205.OverlayValues[43] = d43
					ps205.OverlayValues[44] = d44
					ps205.OverlayValues[45] = d45
					ps205.OverlayValues[47] = d47
					ps205.OverlayValues[48] = d48
					ps205.OverlayValues[49] = d49
					ps205.OverlayValues[50] = d50
					ps205.OverlayValues[52] = d52
					ps205.OverlayValues[53] = d53
					ps205.OverlayValues[54] = d54
					ps205.OverlayValues[55] = d55
					ps205.OverlayValues[56] = d56
					ps205.OverlayValues[57] = d57
					ps205.OverlayValues[58] = d58
					ps205.OverlayValues[59] = d59
					ps205.OverlayValues[60] = d60
					ps205.OverlayValues[61] = d61
					ps205.OverlayValues[62] = d62
					ps205.OverlayValues[63] = d63
					ps205.OverlayValues[64] = d64
					ps205.OverlayValues[65] = d65
					ps205.OverlayValues[66] = d66
					ps205.OverlayValues[67] = d67
					ps205.OverlayValues[68] = d68
					ps205.OverlayValues[70] = d70
					ps205.OverlayValues[71] = d71
					ps205.OverlayValues[76] = d76
					ps205.OverlayValues[78] = d78
					ps205.OverlayValues[80] = d80
					ps205.OverlayValues[81] = d81
					ps205.OverlayValues[82] = d82
					ps205.OverlayValues[87] = d87
					ps205.OverlayValues[89] = d89
					ps205.OverlayValues[90] = d90
					ps205.OverlayValues[91] = d91
					ps205.OverlayValues[92] = d92
					ps205.OverlayValues[93] = d93
					ps205.OverlayValues[94] = d94
					ps205.OverlayValues[95] = d95
					ps205.OverlayValues[99] = d99
					ps205.OverlayValues[100] = d100
					ps205.OverlayValues[101] = d101
					ps205.OverlayValues[104] = d104
					ps205.OverlayValues[105] = d105
					ps205.OverlayValues[175] = d175
					ps205.OverlayValues[176] = d176
					ps205.OverlayValues[177] = d177
					ps205.OverlayValues[178] = d178
					ps205.OverlayValues[179] = d179
					ps205.OverlayValues[180] = d180
					ps205.OverlayValues[181] = d181
					ps205.OverlayValues[183] = d183
					ps205.OverlayValues[184] = d184
					ps205.OverlayValues[189] = d189
					ps205.OverlayValues[191] = d191
					ps205.OverlayValues[192] = d192
					ps205.OverlayValues[193] = d193
					ps205.OverlayValues[194] = d194
					ps205.OverlayValues[196] = d196
					ps205.OverlayValues[197] = d197
					ps205.OverlayValues[201] = d201
					ps205.OverlayValues[203] = d203
					ps205.OverlayValues[204] = d204
					ps205.PhiValues = make([]JITValueDesc, 2)
					if ps205.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps205)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
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
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
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
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
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
					if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
						d76 = ps.OverlayValues[76]
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
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
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
					if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
						d99 = ps.OverlayValues[99]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
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
					if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != LocNone {
						d183 = ps.OverlayValues[183]
					}
					if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != LocNone {
						d184 = ps.OverlayValues[184]
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
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
						d204 = ps.OverlayValues[204]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d1, &result)
						result.Type = d1.Type
					} else {
						switch d1.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d1)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d1)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d1)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d1, &result)
							result.Type = d1.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps206 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps206)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(32))
				return result
			},
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
		},
	})
}
