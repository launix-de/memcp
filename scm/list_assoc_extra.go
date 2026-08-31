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
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d78 JITValueDesc
				_ = d78
				var d82 JITValueDesc
				_ = d82
				var phiBase86 int32
				_ = phiBase86
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
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
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
				var d105 JITValueDesc
				_ = d105
				var d106 JITValueDesc
				_ = d106
				var d111 JITValueDesc
				_ = d111
				var d113 JITValueDesc
				_ = d113
				var d115 JITValueDesc
				_ = d115
				var d116 JITValueDesc
				_ = d116
				var d121 JITValueDesc
				_ = d121
				var d123 JITValueDesc
				_ = d123
				var d124 JITValueDesc
				_ = d124
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d128 JITValueDesc
				_ = d128
				var d132 JITValueDesc
				_ = d132
				var d133 JITValueDesc
				_ = d133
				var d134 JITValueDesc
				_ = d134
				var d137 JITValueDesc
				_ = d137
				var d211 JITValueDesc
				_ = d211
				var d212 JITValueDesc
				_ = d212
				var d213 JITValueDesc
				_ = d213
				var d214 JITValueDesc
				_ = d214
				var d215 JITValueDesc
				_ = d215
				var d216 JITValueDesc
				_ = d216
				var d217 JITValueDesc
				_ = d217
				var d218 JITValueDesc
				_ = d218
				var d220 JITValueDesc
				_ = d220
				var d221 JITValueDesc
				_ = d221
				var d226 JITValueDesc
				_ = d226
				var d228 JITValueDesc
				_ = d228
				var d229 JITValueDesc
				_ = d229
				var d230 JITValueDesc
				_ = d230
				var d231 JITValueDesc
				_ = d231
				var d233 JITValueDesc
				_ = d233
				var d234 JITValueDesc
				_ = d234
				var d239 JITValueDesc
				_ = d239
				var d241 JITValueDesc
				_ = d241
				var d242 JITValueDesc
				_ = d242
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
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
					d9 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: nil}
					ctx.SyncDesc(&d9)
					d10 = jitMaterializeVirtualSlice(ctx, d9, JITValueDesc{Loc: LocAny})
					d11 = jitCopyScmerToPair(ctx, d10)
					ctx.FreeDesc(&d10)
					scmerCellOff12 := ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d11, int32(scmerCellOff12))
					d13 = JITValueDesc{Loc: LocStackPair, Type: d11.Type, StackOff: int32(scmerCellOff12)}
					ctx.FreeDesc(&d11)
					d14 = args[0]
					d14.ID = 0
					ctx.EnsureDesc(&d14)
					d15 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("mapkey_assoc")}
					d16 = d14
					_ = d16
					ctx.StabilizeDescForControlFlow(&d16)
					d17 = d15
					_ = d17
					ctx.StabilizeDescForControlFlow(&d17)
					inlineResultOff18 := ctx.AllocStack(int32(24))
					d19 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: inlineResultOff18}
					inlineResultOff20 := ctx.AllocStack(int32(8))
					d21 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: inlineResultOff20}
					lbl7 := ctx.ReserveLabel()
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
					bbpos_1_6 := int32(-1)
					_ = bbpos_1_6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d23 = d16
					d23.ID = 0
					d22 = ctx.EmitTagEqualsBorrowed(&d23, tagNil, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					d24 = d22
					ctx.EnsureDesc(&d24)
					if d24.Loc != LocImm && d24.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					if d24.Loc == LocImm {
						if d24.Imm.Bool() {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						} else {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitCmpRegImm32(d24.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl10)
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d22)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d26 = d16
					d26.ID = 0
					d25 = ctx.EmitTagEqualsBorrowed(&d26, tagSlice, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					d27 = d25
					ctx.EnsureDesc(&d27)
					if d27.Loc != LocImm && d27.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d27.Loc == LocImm {
						if d27.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitCmpRegImm32(d27.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d25)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d29 = d16
					d29.ID = 0
					d28 = ctx.EmitTagEqualsBorrowed(&d29, tagFastDict, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					d30 = d28
					ctx.EnsureDesc(&d30)
					if d30.Loc != LocImm && d30.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					if d30.Loc == LocImm {
						if d30.Imm.Bool() {
							ctx.MarkLabel(lbl18)
							ctx.EmitJmp(lbl16)
						} else {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						}
					} else {
						ctx.EmitCmpRegImm32(d30.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl18)
						ctx.EmitJmp(lbl19)
						ctx.MarkLabel(lbl18)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
					}
					ctx.FreeDesc(&d28)
					bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					stackArray31 := ctx.AllocStack(int32(0))
					_ = stackArray31
					ctx.ReclaimUntrackedRegs()
					d32 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EmitCopyDescWords(&d19, &d32, 3)
					d33 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EmitZeroDescWords(&d21, 1)
					ctx.EmitJmp(lbl7)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d34 = jitKnownSliceHeader(ctx, &d16)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d34)
					ctx.EmitCopyDescWords(&d19, &d34, 3)
					d35 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EmitZeroDescWords(&d21, 1)
					ctx.EmitJmp(lbl7)
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d36 JITValueDesc
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						panic("FastDict: LocImm not expected at JIT compile time")
					} else if d16.Loc != LocRegPair {
						panic("FastDict: expected Scmer register pair")
					} else {
						ctx.FreeReg(d16.Reg2)
						d36 = JITValueDesc{Loc: LocReg, Reg: d16.Reg}
						ctx.BindReg(d16.Reg, &d36)
						ctx.TransferReg(d16.Reg)
						ctx.BindReg(d16.Reg, &d36)
						d16.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					d37 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EmitZeroDescWords(&d19, 3)
					ctx.EnsureDesc(&d36)
					ctx.EmitCopyDescWords(&d21, &d36, 1)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl7)
					ctx.FreeDesc(&d14)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.EnsureDesc(&d21)
					var d38 JITValueDesc
					if d21.Loc == LocImm {
						d38 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d21)
						if d21.Loc != LocReg && d21.Loc != LocRegPair && d21.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpRegImm32(d21.Reg, 0)
						ctx.EmitSetcc(r0, CondEqual)
						d38 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d38)
					}
					d39 = d38
					ctx.EnsureDesc(&d39)
					if d39.Loc != LocImm && d39.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d39.Loc == LocImm {
						if d39.Imm.Bool() {
							ps40 := PhiState{General: ps.General}
							ps40.OverlayValues = make([]JITValueDesc, 40)
							ps40.OverlayValues[1] = d1
							ps40.OverlayValues[2] = d2
							ps40.OverlayValues[3] = d3
							ps40.OverlayValues[6] = d6
							ps40.OverlayValues[7] = d7
							ps40.OverlayValues[9] = d9
							ps40.OverlayValues[10] = d10
							ps40.OverlayValues[11] = d11
							ps40.OverlayValues[13] = d13
							ps40.OverlayValues[14] = d14
							ps40.OverlayValues[15] = d15
							ps40.OverlayValues[16] = d16
							ps40.OverlayValues[17] = d17
							ps40.OverlayValues[19] = d19
							ps40.OverlayValues[21] = d21
							ps40.OverlayValues[22] = d22
							ps40.OverlayValues[23] = d23
							ps40.OverlayValues[24] = d24
							ps40.OverlayValues[25] = d25
							ps40.OverlayValues[26] = d26
							ps40.OverlayValues[27] = d27
							ps40.OverlayValues[28] = d28
							ps40.OverlayValues[29] = d29
							ps40.OverlayValues[30] = d30
							ps40.OverlayValues[32] = d32
							ps40.OverlayValues[33] = d33
							ps40.OverlayValues[34] = d34
							ps40.OverlayValues[35] = d35
							ps40.OverlayValues[36] = d36
							ps40.OverlayValues[37] = d37
							ps40.OverlayValues[38] = d38
							ps40.OverlayValues[39] = d39
							return bbs[1].RenderPS(ps40)
						}
						ps41 := PhiState{General: ps.General}
						ps41.OverlayValues = make([]JITValueDesc, 40)
						ps41.OverlayValues[1] = d1
						ps41.OverlayValues[2] = d2
						ps41.OverlayValues[3] = d3
						ps41.OverlayValues[6] = d6
						ps41.OverlayValues[7] = d7
						ps41.OverlayValues[9] = d9
						ps41.OverlayValues[10] = d10
						ps41.OverlayValues[11] = d11
						ps41.OverlayValues[13] = d13
						ps41.OverlayValues[14] = d14
						ps41.OverlayValues[15] = d15
						ps41.OverlayValues[16] = d16
						ps41.OverlayValues[17] = d17
						ps41.OverlayValues[19] = d19
						ps41.OverlayValues[21] = d21
						ps41.OverlayValues[22] = d22
						ps41.OverlayValues[23] = d23
						ps41.OverlayValues[24] = d24
						ps41.OverlayValues[25] = d25
						ps41.OverlayValues[26] = d26
						ps41.OverlayValues[27] = d27
						ps41.OverlayValues[28] = d28
						ps41.OverlayValues[29] = d29
						ps41.OverlayValues[30] = d30
						ps41.OverlayValues[32] = d32
						ps41.OverlayValues[33] = d33
						ps41.OverlayValues[34] = d34
						ps41.OverlayValues[35] = d35
						ps41.OverlayValues[36] = d36
						ps41.OverlayValues[37] = d37
						ps41.OverlayValues[38] = d38
						ps41.OverlayValues[39] = d39
						return bbs[3].RenderPS(ps41)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d39.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl4)
					ps42 := PhiState{General: true}
					ps42.OverlayValues = make([]JITValueDesc, 40)
					ps42.OverlayValues[1] = d1
					ps42.OverlayValues[2] = d2
					ps42.OverlayValues[3] = d3
					ps42.OverlayValues[6] = d6
					ps42.OverlayValues[7] = d7
					ps42.OverlayValues[9] = d9
					ps42.OverlayValues[10] = d10
					ps42.OverlayValues[11] = d11
					ps42.OverlayValues[13] = d13
					ps42.OverlayValues[14] = d14
					ps42.OverlayValues[15] = d15
					ps42.OverlayValues[16] = d16
					ps42.OverlayValues[17] = d17
					ps42.OverlayValues[19] = d19
					ps42.OverlayValues[21] = d21
					ps42.OverlayValues[22] = d22
					ps42.OverlayValues[23] = d23
					ps42.OverlayValues[24] = d24
					ps42.OverlayValues[25] = d25
					ps42.OverlayValues[26] = d26
					ps42.OverlayValues[27] = d27
					ps42.OverlayValues[28] = d28
					ps42.OverlayValues[29] = d29
					ps42.OverlayValues[30] = d30
					ps42.OverlayValues[32] = d32
					ps42.OverlayValues[33] = d33
					ps42.OverlayValues[34] = d34
					ps42.OverlayValues[35] = d35
					ps42.OverlayValues[36] = d36
					ps42.OverlayValues[37] = d37
					ps42.OverlayValues[38] = d38
					ps42.OverlayValues[39] = d39
					ps43 := PhiState{General: true}
					ps43.OverlayValues = make([]JITValueDesc, 40)
					ps43.OverlayValues[1] = d1
					ps43.OverlayValues[2] = d2
					ps43.OverlayValues[3] = d3
					ps43.OverlayValues[6] = d6
					ps43.OverlayValues[7] = d7
					ps43.OverlayValues[9] = d9
					ps43.OverlayValues[10] = d10
					ps43.OverlayValues[11] = d11
					ps43.OverlayValues[13] = d13
					ps43.OverlayValues[14] = d14
					ps43.OverlayValues[15] = d15
					ps43.OverlayValues[16] = d16
					ps43.OverlayValues[17] = d17
					ps43.OverlayValues[19] = d19
					ps43.OverlayValues[21] = d21
					ps43.OverlayValues[22] = d22
					ps43.OverlayValues[23] = d23
					ps43.OverlayValues[24] = d24
					ps43.OverlayValues[25] = d25
					ps43.OverlayValues[26] = d26
					ps43.OverlayValues[27] = d27
					ps43.OverlayValues[28] = d28
					ps43.OverlayValues[29] = d29
					ps43.OverlayValues[30] = d30
					ps43.OverlayValues[32] = d32
					ps43.OverlayValues[33] = d33
					ps43.OverlayValues[34] = d34
					ps43.OverlayValues[35] = d35
					ps43.OverlayValues[36] = d36
					ps43.OverlayValues[37] = d37
					ps43.OverlayValues[38] = d38
					ps43.OverlayValues[39] = d39
					snap44 := d1
					snap45 := d2
					snap46 := d3
					snap47 := d6
					snap48 := d7
					snap49 := d9
					snap50 := d10
					snap51 := d11
					snap52 := d13
					snap53 := d14
					snap54 := d15
					snap55 := d16
					snap56 := d17
					snap57 := d19
					snap58 := d21
					snap59 := d22
					snap60 := d23
					snap61 := d24
					snap62 := d25
					snap63 := d26
					snap64 := d27
					snap65 := d28
					snap66 := d29
					snap67 := d30
					snap68 := d32
					snap69 := d33
					snap70 := d34
					snap71 := d35
					snap72 := d36
					snap73 := d37
					snap74 := d38
					snap75 := d39
					alloc76 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps43)
					}
					ctx.RestoreAllocState(alloc76)
					d1 = snap44
					d2 = snap45
					d3 = snap46
					d6 = snap47
					d7 = snap48
					d9 = snap49
					d10 = snap50
					d11 = snap51
					d13 = snap52
					d14 = snap53
					d15 = snap54
					d16 = snap55
					d17 = snap56
					d19 = snap57
					d21 = snap58
					d22 = snap59
					d23 = snap60
					d24 = snap61
					d25 = snap62
					d26 = snap63
					d27 = snap64
					d28 = snap65
					d29 = snap66
					d30 = snap67
					d32 = snap68
					d33 = snap69
					d34 = snap70
					d35 = snap71
					d36 = snap72
					d37 = snap73
					d38 = snap74
					d39 = snap75
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps42)
					}
					return result
					ctx.FreeDesc(&d38)
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
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
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
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					ps77 := PhiState{General: ps.General}
					ps77.OverlayValues = make([]JITValueDesc, 40)
					ps77.OverlayValues[1] = d1
					ps77.OverlayValues[2] = d2
					ps77.OverlayValues[3] = d3
					ps77.OverlayValues[6] = d6
					ps77.OverlayValues[7] = d7
					ps77.OverlayValues[9] = d9
					ps77.OverlayValues[10] = d10
					ps77.OverlayValues[11] = d11
					ps77.OverlayValues[13] = d13
					ps77.OverlayValues[14] = d14
					ps77.OverlayValues[15] = d15
					ps77.OverlayValues[16] = d16
					ps77.OverlayValues[17] = d17
					ps77.OverlayValues[19] = d19
					ps77.OverlayValues[21] = d21
					ps77.OverlayValues[22] = d22
					ps77.OverlayValues[23] = d23
					ps77.OverlayValues[24] = d24
					ps77.OverlayValues[25] = d25
					ps77.OverlayValues[26] = d26
					ps77.OverlayValues[27] = d27
					ps77.OverlayValues[28] = d28
					ps77.OverlayValues[29] = d29
					ps77.OverlayValues[30] = d30
					ps77.OverlayValues[32] = d32
					ps77.OverlayValues[33] = d33
					ps77.OverlayValues[34] = d34
					ps77.OverlayValues[35] = d35
					ps77.OverlayValues[36] = d36
					ps77.OverlayValues[37] = d37
					ps77.OverlayValues[38] = d38
					ps77.OverlayValues[39] = d39
					ps77.PhiValues = make([]JITValueDesc, 1)
					d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps77.PhiValues[0] = d78
					if ps77.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps77)
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
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs79 := make([]Reg, 0, 3)
					seenBlockPinnedRegs80 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs80
					for _, r := range []Reg{d13.Reg, d13.Reg2, d13.Reg3} {
						live := d13.Loc == LocRegTriple && (r == d13.Reg || r == d13.Reg2 || r == d13.Reg3)
						if live && !seenBlockPinnedRegs80[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs80[r] = true
							blockPinnedRegs79 = append(blockPinnedRegs79, r)
						}
					}
					unpinBlockRegs81 := func() {
						for _, r := range blockPinnedRegs79 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs81()
					d82 = d13
					_ = d82
					ctx.EnsureDesc(&d82)
					if d82.Loc == LocRegPair {
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
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs83 := make([]Reg, 0, 6)
					seenBlockPinnedRegs84 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs84
					for _, r := range []Reg{d13.Reg, d13.Reg2, d13.Reg3} {
						live := d13.Loc == LocRegTriple && (r == d13.Reg || r == d13.Reg2 || r == d13.Reg3)
						if live && !seenBlockPinnedRegs84[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs84[r] = true
							blockPinnedRegs83 = append(blockPinnedRegs83, r)
						}
					}
					for _, r := range []Reg{d21.Reg, d21.Reg2, d21.Reg3} {
						live := d21.Loc == LocRegTriple && (r == d21.Reg || r == d21.Reg2 || r == d21.Reg3)
						if live && !seenBlockPinnedRegs84[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs84[r] = true
							blockPinnedRegs83 = append(blockPinnedRegs83, r)
						}
					}
					unpinBlockRegs85 := func() {
						for _, r := range blockPinnedRegs83 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs85()
					ctx.EnsureDesc(&d21)
					phiBase86 = ctx.AllocStack(int32(16))
					d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase86) + int32(0)}
					lbl22 := ctx.ReserveLabel()
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
					bbpos_2_5 := int32(-1)
					_ = bbpos_2_5
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase86) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase86)+int32(0))
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase86) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d87)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d88 JITValueDesc
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						fieldAddr := uintptr(d21.Imm.Int()) + 0
						r1 := ctx.AllocReg()
						r2 := ctx.AllocRegExcept(r1)
						r3 := ctx.AllocRegExcept(r1, r2)
						ctx.EmitMovRegMem64(r1, fieldAddr)
						ctx.EmitMovRegMem64(r2, fieldAddr+8)
						ctx.EmitMovRegMem64(r3, fieldAddr+16)
						d88 = JITValueDesc{Loc: LocRegTriple, Reg: r1, Reg2: r2, Reg3: r3}
						ctx.BindReg(r1, &d88)
						ctx.BindReg(r2, &d88)
						ctx.BindReg(r3, &d88)
					} else {
						off := int32(0)
						baseReg := d21.Reg
						r4 := ctx.AllocRegExcept(baseReg)
						r5 := ctx.AllocRegExcept(baseReg, r4)
						r6 := ctx.AllocRegExcept(baseReg, r4, r5)
						ctx.EmitMovRegMem(r4, baseReg, off)
						ctx.EmitMovRegMem(r5, baseReg, off+8)
						ctx.EmitMovRegMem(r6, baseReg, off+16)
						d88 = JITValueDesc{Loc: LocRegTriple, Reg: r4, Reg2: r5, Reg3: r6}
						ctx.BindReg(r4, &d88)
						ctx.BindReg(r5, &d88)
						ctx.BindReg(r6, &d88)
					}
					ctx.ReclaimUntrackedRegs()
					var d89 JITValueDesc
					if d88.SliceSizeKnown {
						d89 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d88.KnownSliceLen))}
					} else if d88.Loc == LocImm {
						d89 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d88.StackOff))}
					} else if d88.Loc == LocStackTriple {
						d89 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d88.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d88)
						if d88.Loc == LocRegPair || d88.Loc == LocRegTriple {
							d89 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d88.Reg2, ID: 0}
						} else if d88.Loc == LocReg {
							d89 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d88.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d89)
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d89)
					var d90 JITValueDesc
					if d87.Loc == LocImm && d89.Loc == LocImm {
						d90 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d87.Imm.Int() < d89.Imm.Int())}
					} else if d89.Loc == LocImm {
						r7 := ctx.AllocRegExcept(d87.Reg)
						if d89.Imm.Int() >= -2147483648 && d89.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d87.Reg, int32(d89.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d89.Imm.Int()))
							ctx.EmitCmpInt64(d87.Reg, RegR11)
						}
						ctx.EmitSetcc(r7, CondSignedLess)
						d90 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
						ctx.BindReg(r7, &d90)
					} else if d87.Loc == LocImm {
						r8 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d87.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d89.Reg)
						ctx.EmitSetcc(r8, CondSignedLess)
						d90 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d90)
					} else {
						r9 := ctx.AllocRegExcept(d87.Reg)
						ctx.EmitCmpInt64(d87.Reg, d89.Reg)
						ctx.EmitSetcc(r9, CondSignedLess)
						d90 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d90)
					}
					ctx.FreeDesc(&d89)
					ctx.ReclaimUntrackedRegs()
					d91 = d90
					ctx.EnsureDesc(&d91)
					if d91.Loc != LocImm && d91.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					lbl26 := ctx.ReserveLabel()
					if d91.Loc == LocImm {
						if d91.Imm.Bool() {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						} else {
							ctx.MarkLabel(lbl26)
							ctx.EmitJmp(lbl24)
						}
					} else {
						ctx.EmitCmpRegImm32(d91.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl25)
						ctx.EmitJmp(lbl26)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl24)
					}
					ctx.FreeDesc(&d90)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl24)
					ctx.ResolveFixups()
					d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase86) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl22)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase86) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d92 JITValueDesc
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						fieldAddr := uintptr(d21.Imm.Int()) + 0
						r10 := ctx.AllocReg()
						r11 := ctx.AllocRegExcept(r10)
						r12 := ctx.AllocRegExcept(r10, r11)
						ctx.EmitMovRegMem64(r10, fieldAddr)
						ctx.EmitMovRegMem64(r11, fieldAddr+8)
						ctx.EmitMovRegMem64(r12, fieldAddr+16)
						d92 = JITValueDesc{Loc: LocRegTriple, Reg: r10, Reg2: r11, Reg3: r12}
						ctx.BindReg(r10, &d92)
						ctx.BindReg(r11, &d92)
						ctx.BindReg(r12, &d92)
					} else {
						off := int32(0)
						baseReg := d21.Reg
						r13 := ctx.AllocRegExcept(baseReg)
						r14 := ctx.AllocRegExcept(baseReg, r13)
						r15 := ctx.AllocRegExcept(baseReg, r13, r14)
						ctx.EmitMovRegMem(r13, baseReg, off)
						ctx.EmitMovRegMem(r14, baseReg, off+8)
						ctx.EmitMovRegMem(r15, baseReg, off+16)
						d92 = JITValueDesc{Loc: LocRegTriple, Reg: r13, Reg2: r14, Reg3: r15}
						ctx.BindReg(r13, &d92)
						ctx.BindReg(r14, &d92)
						ctx.BindReg(r15, &d92)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d87)
					ctx.ReclaimUntrackedRegs()
					d94 = ctx.EmitSliceElementAddress(&d92, &d87, 16)
					ctx.EnsureDesc(&d94)
					r16 := ctx.AllocRegExcept(d94.Reg)
					ctx.EmitMovRegMem(r16, d94.Reg, 8)
					ctx.EmitMovRegMem(d94.Reg, d94.Reg, 0)
					d93 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d94.Reg, Reg2: r16}
					ctx.BindReg(d94.Reg, &d93)
					ctx.BindReg(r16, &d93)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d95 JITValueDesc
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						fieldAddr := uintptr(d21.Imm.Int()) + 0
						r17 := ctx.AllocReg()
						r18 := ctx.AllocRegExcept(r17)
						r19 := ctx.AllocRegExcept(r17, r18)
						ctx.EmitMovRegMem64(r17, fieldAddr)
						ctx.EmitMovRegMem64(r18, fieldAddr+8)
						ctx.EmitMovRegMem64(r19, fieldAddr+16)
						d95 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
						ctx.BindReg(r17, &d95)
						ctx.BindReg(r18, &d95)
						ctx.BindReg(r19, &d95)
					} else {
						off := int32(0)
						baseReg := d21.Reg
						r20 := ctx.AllocRegExcept(baseReg)
						r21 := ctx.AllocRegExcept(baseReg, r20)
						r22 := ctx.AllocRegExcept(baseReg, r20, r21)
						ctx.EmitMovRegMem(r20, baseReg, off)
						ctx.EmitMovRegMem(r21, baseReg, off+8)
						ctx.EmitMovRegMem(r22, baseReg, off+16)
						d95 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
						ctx.BindReg(r20, &d95)
						ctx.BindReg(r21, &d95)
						ctx.BindReg(r22, &d95)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d87)
					var d96 JITValueDesc
					if d87.Loc == LocImm {
						d96 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d87.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d87.Reg)
						ctx.EmitMovRegReg(scratch, d87.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d96 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d96)
					}
					if d96.Loc == LocReg && d87.Loc == LocReg && d96.Reg == d87.Reg {
						ctx.TransferReg(d87.Reg)
						d87.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d96)
					ctx.ReclaimUntrackedRegs()
					d98 = ctx.EmitSliceElementAddress(&d95, &d96, 16)
					ctx.EnsureDesc(&d98)
					r23 := ctx.AllocRegExcept(d98.Reg)
					ctx.EmitMovRegMem(r23, d98.Reg, 8)
					ctx.EmitMovRegMem(d98.Reg, d98.Reg, 0)
					d97 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d98.Reg, Reg2: r23}
					ctx.BindReg(d98.Reg, &d97)
					ctx.BindReg(r23, &d97)
					ctx.FreeDesc(&d96)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d93)
					ctx.EnsureDesc(&d97)
					d99 = d93
					_ = d99
					ctx.StabilizeDescForControlFlow(&d99)
					d100 = d97
					_ = d100
					ctx.StabilizeDescForControlFlow(&d100)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d101 = d7
					_ = d101
					ctx.ReclaimUntrackedRegs()
					d102 = d13
					_ = d102
					ctx.ReclaimUntrackedRegs()
					d103 = d3
					_ = d103
					ctx.ReclaimUntrackedRegs()
					stackArray104 := ctx.AllocStack(int32(32))
					_ = stackArray104
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d99)
					ctx.EnsureDesc(&d99)
					ctx.EmitStoreScmerToStack(d99, int32(stackArray104)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d100)
					ctx.EnsureDesc(&d100)
					ctx.EmitStoreScmerToStack(d100, int32(stackArray104)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d105 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					ctx.ReclaimUntrackedRegs()
					callbackArgs107 := make([]JITValueDesc, 2)
					callbackArgs107[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray104) + 0}
					callbackArgs107[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray104) + 16}
					var d106 JITValueDesc
					callbackResultOff108 := ctx.AllocStack(16)
					ctx.FreeDesc(&d105)
					if d103.Loc == LocLambdaTemplate && d103.Lambda != nil {
						stableCallbackArgs109 := ctx.StabilizeCallbackArgs(callbackArgs107)
						ctx.ReclaimUntrackedRegs()
						outerRegs110 := ctx.PreserveOuterRegs()
						d106 = JITEmitProcInlineWithOuter(ctx, &d103.Lambda.Proc, d103.Lambda.Outer, stableCallbackArgs109, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff108), ID: 0})
						ctx.RestoreOuterRegs(outerRegs110)
						ctx.ReclaimUntrackedRegs()
					} else {
						d111, knownBuiltin112 := jitEmitKnownDeclaration(ctx, d103, callbackArgs107, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff108), ID: 0})
						if knownBuiltin112 {
							d106 = d111
						} else {
							d113 := jitCopyScmerToPair(ctx, d103)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d113)
							callbackCallArgs = append(callbackCallArgs, callbackArgs107...)
							d106 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d106, int32(callbackResultOff108))
							ctx.FreeDesc(&d106)
							d106 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff108), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					stackArray114 := ctx.AllocStack(int32(48))
					_ = stackArray114
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d102)
					ctx.EnsureDesc(&d102)
					ctx.EmitStoreScmerToStack(d102, int32(stackArray114)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					ctx.EnsureDesc(&d106)
					ctx.EmitStoreScmerToStack(d106, int32(stackArray114)+int32(16))
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d100)
					ctx.EnsureDesc(&d100)
					ctx.EmitStoreScmerToStack(d100, int32(stackArray114)+int32(32))
					ctx.ReclaimUntrackedRegs()
					d115 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					ctx.ReclaimUntrackedRegs()
					callbackArgs117 := make([]JITValueDesc, 3)
					callbackArgs117[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray114) + 0}
					callbackArgs117[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray114) + 16}
					callbackArgs117[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray114) + 32}
					var d116 JITValueDesc
					callbackResultOff118 := ctx.AllocStack(16)
					ctx.FreeDesc(&d115)
					if d101.Loc == LocLambdaTemplate && d101.Lambda != nil {
						stableCallbackArgs119 := ctx.StabilizeCallbackArgs(callbackArgs117)
						ctx.ReclaimUntrackedRegs()
						outerRegs120 := ctx.PreserveOuterRegs()
						d116 = JITEmitProcInlineWithOuter(ctx, &d101.Lambda.Proc, d101.Lambda.Outer, stableCallbackArgs119, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff118), ID: 0})
						ctx.RestoreOuterRegs(outerRegs120)
						ctx.ReclaimUntrackedRegs()
					} else {
						d121, knownBuiltin122 := jitEmitKnownDeclaration(ctx, d101, callbackArgs117, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff118), ID: 0})
						if knownBuiltin122 {
							d116 = d121
						} else {
							d123 := jitCopyScmerToPair(ctx, d101)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d123)
							callbackCallArgs = append(callbackCallArgs, callbackArgs117...)
							d116 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d116, int32(callbackResultOff118))
							ctx.FreeDesc(&d116)
							d116 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff118), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d116)
					ctx.EmitCopyScmerToDesc(&d13, &d116)
					ctx.FreeDesc(&d116)
					ctx.ReclaimUntrackedRegs()
					d124 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.FreeDesc(&d93)
					ctx.FreeDesc(&d97)
					ctx.ReclaimUntrackedRegs()
					d125 = d124
					ctx.EnsureDesc(&d125)
					if d125.Loc != LocImm && d125.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					lbl30 := ctx.ReserveLabel()
					if d125.Loc == LocImm {
						if d125.Imm.Bool() {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl27)
						} else {
							ctx.MarkLabel(lbl30)
							ctx.EmitJmp(lbl28)
						}
					} else {
						ctx.EmitCmpRegImm32(d125.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl29)
						ctx.EmitJmp(lbl30)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
						ctx.MarkLabel(lbl30)
						ctx.EmitJmp(lbl28)
					}
					ctx.FreeDesc(&d124)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl28)
					ctx.ResolveFixups()
					d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase86) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl22)
					bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					d87 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase86) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d87)
					ctx.EnsureDesc(&d87)
					var d126 JITValueDesc
					if d87.Loc == LocImm {
						d126 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d87.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d87.Reg)
						ctx.EmitMovRegReg(scratch, d87.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d126 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d126)
					}
					if d126.Loc == LocReg && d87.Loc == LocReg && d126.Reg == d87.Reg {
						ctx.TransferReg(d87.Reg)
						d87.Loc = LocNone
					}
					ctx.EnsureDesc(&d126)
					ctx.EmitStoreToStack(d126, int32(phiBase86)+int32(0))
					ctx.StabilizeDescForControlFlow(&d126)
					ctx.FreeDesc(&d87)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_2_1)
					ctx.MarkLabel(lbl22)
					ctx.FreeDesc(&d21)
					ps127 := PhiState{General: ps.General}
					ps127.OverlayValues = make([]JITValueDesc, 127)
					ps127.OverlayValues[1] = d1
					ps127.OverlayValues[2] = d2
					ps127.OverlayValues[3] = d3
					ps127.OverlayValues[6] = d6
					ps127.OverlayValues[7] = d7
					ps127.OverlayValues[9] = d9
					ps127.OverlayValues[10] = d10
					ps127.OverlayValues[11] = d11
					ps127.OverlayValues[13] = d13
					ps127.OverlayValues[14] = d14
					ps127.OverlayValues[15] = d15
					ps127.OverlayValues[16] = d16
					ps127.OverlayValues[17] = d17
					ps127.OverlayValues[19] = d19
					ps127.OverlayValues[21] = d21
					ps127.OverlayValues[22] = d22
					ps127.OverlayValues[23] = d23
					ps127.OverlayValues[24] = d24
					ps127.OverlayValues[25] = d25
					ps127.OverlayValues[26] = d26
					ps127.OverlayValues[27] = d27
					ps127.OverlayValues[28] = d28
					ps127.OverlayValues[29] = d29
					ps127.OverlayValues[30] = d30
					ps127.OverlayValues[32] = d32
					ps127.OverlayValues[33] = d33
					ps127.OverlayValues[34] = d34
					ps127.OverlayValues[35] = d35
					ps127.OverlayValues[36] = d36
					ps127.OverlayValues[37] = d37
					ps127.OverlayValues[38] = d38
					ps127.OverlayValues[39] = d39
					ps127.OverlayValues[78] = d78
					ps127.OverlayValues[82] = d82
					ps127.OverlayValues[87] = d87
					ps127.OverlayValues[88] = d88
					ps127.OverlayValues[89] = d89
					ps127.OverlayValues[90] = d90
					ps127.OverlayValues[91] = d91
					ps127.OverlayValues[92] = d92
					ps127.OverlayValues[93] = d93
					ps127.OverlayValues[94] = d94
					ps127.OverlayValues[95] = d95
					ps127.OverlayValues[96] = d96
					ps127.OverlayValues[97] = d97
					ps127.OverlayValues[98] = d98
					ps127.OverlayValues[99] = d99
					ps127.OverlayValues[100] = d100
					ps127.OverlayValues[101] = d101
					ps127.OverlayValues[102] = d102
					ps127.OverlayValues[103] = d103
					ps127.OverlayValues[105] = d105
					ps127.OverlayValues[106] = d106
					ps127.OverlayValues[111] = d111
					ps127.OverlayValues[113] = d113
					ps127.OverlayValues[115] = d115
					ps127.OverlayValues[116] = d116
					ps127.OverlayValues[121] = d121
					ps127.OverlayValues[123] = d123
					ps127.OverlayValues[124] = d124
					ps127.OverlayValues[125] = d125
					ps127.OverlayValues[126] = d126
					if ps127.General && bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
						return result
					}
					return bbs[2].RenderPS(ps127)
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d128 := ps.PhiValues[0]
							ctx.EnsureDesc(&d128)
							ctx.EmitStoreToStack(d128, int32(bbs[4].PhiBase)+int32(0))
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
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
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
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
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
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs129 := make([]Reg, 0, 3)
					seenBlockPinnedRegs130 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs130
					for _, r := range []Reg{d19.Reg, d19.Reg2, d19.Reg3} {
						live := d19.Loc == LocRegTriple && (r == d19.Reg || r == d19.Reg2 || r == d19.Reg3)
						if live && !seenBlockPinnedRegs130[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs130[r] = true
							blockPinnedRegs129 = append(blockPinnedRegs129, r)
						}
					}
					unpinBlockRegs131 := func() {
						for _, r := range blockPinnedRegs129 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs131()
					ctx.StabilizeDescForControlFlow(&d1)
					var d132 JITValueDesc
					if d19.SliceSizeKnown {
						d132 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.KnownSliceLen))}
					} else if d19.Loc == LocImm {
						d132 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d19.StackOff))}
					} else if d19.Loc == LocStackTriple {
						d132 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d19.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d19)
						if d19.Loc == LocRegPair || d19.Loc == LocRegTriple {
							d132 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg2, ID: 0}
						} else if d19.Loc == LocReg {
							d132 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d19.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d132)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d132)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d132)
					var d133 JITValueDesc
					if d1.Loc == LocImm && d132.Loc == LocImm {
						d133 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d132.Imm.Int())}
					} else if d132.Loc == LocImm {
						r24 := ctx.AllocRegExcept(d1.Reg)
						if d132.Imm.Int() >= -2147483648 && d132.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d132.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d132.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r24, CondSignedLess)
						d133 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
						ctx.BindReg(r24, &d133)
					} else if d1.Loc == LocImm {
						r25 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d132.Reg)
						ctx.EmitSetcc(r25, CondSignedLess)
						d133 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
						ctx.BindReg(r25, &d133)
					} else {
						r26 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d132.Reg)
						ctx.EmitSetcc(r26, CondSignedLess)
						d133 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d133)
					}
					ctx.FreeDesc(&d132)
					d134 = d133
					ctx.EnsureDesc(&d134)
					if d134.Loc != LocImm && d134.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d134.Loc == LocImm {
						if d134.Imm.Bool() {
							ps135 := PhiState{General: ps.General}
							ps135.OverlayValues = make([]JITValueDesc, 135)
							ps135.OverlayValues[1] = d1
							ps135.OverlayValues[2] = d2
							ps135.OverlayValues[3] = d3
							ps135.OverlayValues[6] = d6
							ps135.OverlayValues[7] = d7
							ps135.OverlayValues[9] = d9
							ps135.OverlayValues[10] = d10
							ps135.OverlayValues[11] = d11
							ps135.OverlayValues[13] = d13
							ps135.OverlayValues[14] = d14
							ps135.OverlayValues[15] = d15
							ps135.OverlayValues[16] = d16
							ps135.OverlayValues[17] = d17
							ps135.OverlayValues[19] = d19
							ps135.OverlayValues[21] = d21
							ps135.OverlayValues[22] = d22
							ps135.OverlayValues[23] = d23
							ps135.OverlayValues[24] = d24
							ps135.OverlayValues[25] = d25
							ps135.OverlayValues[26] = d26
							ps135.OverlayValues[27] = d27
							ps135.OverlayValues[28] = d28
							ps135.OverlayValues[29] = d29
							ps135.OverlayValues[30] = d30
							ps135.OverlayValues[32] = d32
							ps135.OverlayValues[33] = d33
							ps135.OverlayValues[34] = d34
							ps135.OverlayValues[35] = d35
							ps135.OverlayValues[36] = d36
							ps135.OverlayValues[37] = d37
							ps135.OverlayValues[38] = d38
							ps135.OverlayValues[39] = d39
							ps135.OverlayValues[78] = d78
							ps135.OverlayValues[82] = d82
							ps135.OverlayValues[87] = d87
							ps135.OverlayValues[88] = d88
							ps135.OverlayValues[89] = d89
							ps135.OverlayValues[90] = d90
							ps135.OverlayValues[91] = d91
							ps135.OverlayValues[92] = d92
							ps135.OverlayValues[93] = d93
							ps135.OverlayValues[94] = d94
							ps135.OverlayValues[95] = d95
							ps135.OverlayValues[96] = d96
							ps135.OverlayValues[97] = d97
							ps135.OverlayValues[98] = d98
							ps135.OverlayValues[99] = d99
							ps135.OverlayValues[100] = d100
							ps135.OverlayValues[101] = d101
							ps135.OverlayValues[102] = d102
							ps135.OverlayValues[103] = d103
							ps135.OverlayValues[105] = d105
							ps135.OverlayValues[106] = d106
							ps135.OverlayValues[111] = d111
							ps135.OverlayValues[113] = d113
							ps135.OverlayValues[115] = d115
							ps135.OverlayValues[116] = d116
							ps135.OverlayValues[121] = d121
							ps135.OverlayValues[123] = d123
							ps135.OverlayValues[124] = d124
							ps135.OverlayValues[125] = d125
							ps135.OverlayValues[126] = d126
							ps135.OverlayValues[128] = d128
							ps135.OverlayValues[132] = d132
							ps135.OverlayValues[133] = d133
							ps135.OverlayValues[134] = d134
							return bbs[5].RenderPS(ps135)
						}
						ps136 := PhiState{General: ps.General}
						ps136.OverlayValues = make([]JITValueDesc, 135)
						ps136.OverlayValues[1] = d1
						ps136.OverlayValues[2] = d2
						ps136.OverlayValues[3] = d3
						ps136.OverlayValues[6] = d6
						ps136.OverlayValues[7] = d7
						ps136.OverlayValues[9] = d9
						ps136.OverlayValues[10] = d10
						ps136.OverlayValues[11] = d11
						ps136.OverlayValues[13] = d13
						ps136.OverlayValues[14] = d14
						ps136.OverlayValues[15] = d15
						ps136.OverlayValues[16] = d16
						ps136.OverlayValues[17] = d17
						ps136.OverlayValues[19] = d19
						ps136.OverlayValues[21] = d21
						ps136.OverlayValues[22] = d22
						ps136.OverlayValues[23] = d23
						ps136.OverlayValues[24] = d24
						ps136.OverlayValues[25] = d25
						ps136.OverlayValues[26] = d26
						ps136.OverlayValues[27] = d27
						ps136.OverlayValues[28] = d28
						ps136.OverlayValues[29] = d29
						ps136.OverlayValues[30] = d30
						ps136.OverlayValues[32] = d32
						ps136.OverlayValues[33] = d33
						ps136.OverlayValues[34] = d34
						ps136.OverlayValues[35] = d35
						ps136.OverlayValues[36] = d36
						ps136.OverlayValues[37] = d37
						ps136.OverlayValues[38] = d38
						ps136.OverlayValues[39] = d39
						ps136.OverlayValues[78] = d78
						ps136.OverlayValues[82] = d82
						ps136.OverlayValues[87] = d87
						ps136.OverlayValues[88] = d88
						ps136.OverlayValues[89] = d89
						ps136.OverlayValues[90] = d90
						ps136.OverlayValues[91] = d91
						ps136.OverlayValues[92] = d92
						ps136.OverlayValues[93] = d93
						ps136.OverlayValues[94] = d94
						ps136.OverlayValues[95] = d95
						ps136.OverlayValues[96] = d96
						ps136.OverlayValues[97] = d97
						ps136.OverlayValues[98] = d98
						ps136.OverlayValues[99] = d99
						ps136.OverlayValues[100] = d100
						ps136.OverlayValues[101] = d101
						ps136.OverlayValues[102] = d102
						ps136.OverlayValues[103] = d103
						ps136.OverlayValues[105] = d105
						ps136.OverlayValues[106] = d106
						ps136.OverlayValues[111] = d111
						ps136.OverlayValues[113] = d113
						ps136.OverlayValues[115] = d115
						ps136.OverlayValues[116] = d116
						ps136.OverlayValues[121] = d121
						ps136.OverlayValues[123] = d123
						ps136.OverlayValues[124] = d124
						ps136.OverlayValues[125] = d125
						ps136.OverlayValues[126] = d126
						ps136.OverlayValues[128] = d128
						ps136.OverlayValues[132] = d132
						ps136.OverlayValues[133] = d133
						ps136.OverlayValues[134] = d134
						return bbs[2].RenderPS(ps136)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d137 := ps.PhiValues[0]
							ctx.EnsureDesc(&d137)
							ctx.EmitStoreToStack(d137, int32(bbs[4].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[4].RenderPS(ps)
					}
					lbl31 := ctx.ReserveLabel()
					lbl32 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d134.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl31)
					ctx.EmitJmp(lbl32)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl3)
					ps138 := PhiState{General: true}
					ps138.OverlayValues = make([]JITValueDesc, 138)
					ps138.OverlayValues[1] = d1
					ps138.OverlayValues[2] = d2
					ps138.OverlayValues[3] = d3
					ps138.OverlayValues[6] = d6
					ps138.OverlayValues[7] = d7
					ps138.OverlayValues[9] = d9
					ps138.OverlayValues[10] = d10
					ps138.OverlayValues[11] = d11
					ps138.OverlayValues[13] = d13
					ps138.OverlayValues[14] = d14
					ps138.OverlayValues[15] = d15
					ps138.OverlayValues[16] = d16
					ps138.OverlayValues[17] = d17
					ps138.OverlayValues[19] = d19
					ps138.OverlayValues[21] = d21
					ps138.OverlayValues[22] = d22
					ps138.OverlayValues[23] = d23
					ps138.OverlayValues[24] = d24
					ps138.OverlayValues[25] = d25
					ps138.OverlayValues[26] = d26
					ps138.OverlayValues[27] = d27
					ps138.OverlayValues[28] = d28
					ps138.OverlayValues[29] = d29
					ps138.OverlayValues[30] = d30
					ps138.OverlayValues[32] = d32
					ps138.OverlayValues[33] = d33
					ps138.OverlayValues[34] = d34
					ps138.OverlayValues[35] = d35
					ps138.OverlayValues[36] = d36
					ps138.OverlayValues[37] = d37
					ps138.OverlayValues[38] = d38
					ps138.OverlayValues[39] = d39
					ps138.OverlayValues[78] = d78
					ps138.OverlayValues[82] = d82
					ps138.OverlayValues[87] = d87
					ps138.OverlayValues[88] = d88
					ps138.OverlayValues[89] = d89
					ps138.OverlayValues[90] = d90
					ps138.OverlayValues[91] = d91
					ps138.OverlayValues[92] = d92
					ps138.OverlayValues[93] = d93
					ps138.OverlayValues[94] = d94
					ps138.OverlayValues[95] = d95
					ps138.OverlayValues[96] = d96
					ps138.OverlayValues[97] = d97
					ps138.OverlayValues[98] = d98
					ps138.OverlayValues[99] = d99
					ps138.OverlayValues[100] = d100
					ps138.OverlayValues[101] = d101
					ps138.OverlayValues[102] = d102
					ps138.OverlayValues[103] = d103
					ps138.OverlayValues[105] = d105
					ps138.OverlayValues[106] = d106
					ps138.OverlayValues[111] = d111
					ps138.OverlayValues[113] = d113
					ps138.OverlayValues[115] = d115
					ps138.OverlayValues[116] = d116
					ps138.OverlayValues[121] = d121
					ps138.OverlayValues[123] = d123
					ps138.OverlayValues[124] = d124
					ps138.OverlayValues[125] = d125
					ps138.OverlayValues[126] = d126
					ps138.OverlayValues[128] = d128
					ps138.OverlayValues[132] = d132
					ps138.OverlayValues[133] = d133
					ps138.OverlayValues[134] = d134
					ps138.OverlayValues[137] = d137
					ps139 := PhiState{General: true}
					ps139.OverlayValues = make([]JITValueDesc, 138)
					ps139.OverlayValues[1] = d1
					ps139.OverlayValues[2] = d2
					ps139.OverlayValues[3] = d3
					ps139.OverlayValues[6] = d6
					ps139.OverlayValues[7] = d7
					ps139.OverlayValues[9] = d9
					ps139.OverlayValues[10] = d10
					ps139.OverlayValues[11] = d11
					ps139.OverlayValues[13] = d13
					ps139.OverlayValues[14] = d14
					ps139.OverlayValues[15] = d15
					ps139.OverlayValues[16] = d16
					ps139.OverlayValues[17] = d17
					ps139.OverlayValues[19] = d19
					ps139.OverlayValues[21] = d21
					ps139.OverlayValues[22] = d22
					ps139.OverlayValues[23] = d23
					ps139.OverlayValues[24] = d24
					ps139.OverlayValues[25] = d25
					ps139.OverlayValues[26] = d26
					ps139.OverlayValues[27] = d27
					ps139.OverlayValues[28] = d28
					ps139.OverlayValues[29] = d29
					ps139.OverlayValues[30] = d30
					ps139.OverlayValues[32] = d32
					ps139.OverlayValues[33] = d33
					ps139.OverlayValues[34] = d34
					ps139.OverlayValues[35] = d35
					ps139.OverlayValues[36] = d36
					ps139.OverlayValues[37] = d37
					ps139.OverlayValues[38] = d38
					ps139.OverlayValues[39] = d39
					ps139.OverlayValues[78] = d78
					ps139.OverlayValues[82] = d82
					ps139.OverlayValues[87] = d87
					ps139.OverlayValues[88] = d88
					ps139.OverlayValues[89] = d89
					ps139.OverlayValues[90] = d90
					ps139.OverlayValues[91] = d91
					ps139.OverlayValues[92] = d92
					ps139.OverlayValues[93] = d93
					ps139.OverlayValues[94] = d94
					ps139.OverlayValues[95] = d95
					ps139.OverlayValues[96] = d96
					ps139.OverlayValues[97] = d97
					ps139.OverlayValues[98] = d98
					ps139.OverlayValues[99] = d99
					ps139.OverlayValues[100] = d100
					ps139.OverlayValues[101] = d101
					ps139.OverlayValues[102] = d102
					ps139.OverlayValues[103] = d103
					ps139.OverlayValues[105] = d105
					ps139.OverlayValues[106] = d106
					ps139.OverlayValues[111] = d111
					ps139.OverlayValues[113] = d113
					ps139.OverlayValues[115] = d115
					ps139.OverlayValues[116] = d116
					ps139.OverlayValues[121] = d121
					ps139.OverlayValues[123] = d123
					ps139.OverlayValues[124] = d124
					ps139.OverlayValues[125] = d125
					ps139.OverlayValues[126] = d126
					ps139.OverlayValues[128] = d128
					ps139.OverlayValues[132] = d132
					ps139.OverlayValues[133] = d133
					ps139.OverlayValues[134] = d134
					ps139.OverlayValues[137] = d137
					snap140 := d1
					snap141 := d2
					snap142 := d3
					snap143 := d6
					snap144 := d7
					snap145 := d9
					snap146 := d10
					snap147 := d11
					snap148 := d13
					snap149 := d14
					snap150 := d15
					snap151 := d16
					snap152 := d17
					snap153 := d19
					snap154 := d21
					snap155 := d22
					snap156 := d23
					snap157 := d24
					snap158 := d25
					snap159 := d26
					snap160 := d27
					snap161 := d28
					snap162 := d29
					snap163 := d30
					snap164 := d32
					snap165 := d33
					snap166 := d34
					snap167 := d35
					snap168 := d36
					snap169 := d37
					snap170 := d38
					snap171 := d39
					snap172 := d78
					snap173 := d82
					snap174 := d87
					snap175 := d88
					snap176 := d89
					snap177 := d90
					snap178 := d91
					snap179 := d92
					snap180 := d93
					snap181 := d94
					snap182 := d95
					snap183 := d96
					snap184 := d97
					snap185 := d98
					snap186 := d99
					snap187 := d100
					snap188 := d101
					snap189 := d102
					snap190 := d103
					snap191 := d105
					snap192 := d106
					snap193 := d111
					snap194 := d113
					snap195 := d115
					snap196 := d116
					snap197 := d121
					snap198 := d123
					snap199 := d124
					snap200 := d125
					snap201 := d126
					snap202 := d128
					snap203 := d132
					snap204 := d133
					snap205 := d134
					snap206 := d137
					alloc207 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps139)
					}
					ctx.RestoreAllocState(alloc207)
					d1 = snap140
					d2 = snap141
					d3 = snap142
					d6 = snap143
					d7 = snap144
					d9 = snap145
					d10 = snap146
					d11 = snap147
					d13 = snap148
					d14 = snap149
					d15 = snap150
					d16 = snap151
					d17 = snap152
					d19 = snap153
					d21 = snap154
					d22 = snap155
					d23 = snap156
					d24 = snap157
					d25 = snap158
					d26 = snap159
					d27 = snap160
					d28 = snap161
					d29 = snap162
					d30 = snap163
					d32 = snap164
					d33 = snap165
					d34 = snap166
					d35 = snap167
					d36 = snap168
					d37 = snap169
					d38 = snap170
					d39 = snap171
					d78 = snap172
					d82 = snap173
					d87 = snap174
					d88 = snap175
					d89 = snap176
					d90 = snap177
					d91 = snap178
					d92 = snap179
					d93 = snap180
					d94 = snap181
					d95 = snap182
					d96 = snap183
					d97 = snap184
					d98 = snap185
					d99 = snap186
					d100 = snap187
					d101 = snap188
					d102 = snap189
					d103 = snap190
					d105 = snap191
					d106 = snap192
					d111 = snap193
					d113 = snap194
					d115 = snap195
					d116 = snap196
					d121 = snap197
					d123 = snap198
					d124 = snap199
					d125 = snap200
					d126 = snap201
					d128 = snap202
					d132 = snap203
					d133 = snap204
					d134 = snap205
					d137 = snap206
					if !bbs[5].Rendered {
						return bbs[5].RenderPS(ps138)
					}
					return result
					ctx.FreeDesc(&d133)
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
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != LocNone {
						d82 = ps.OverlayValues[82]
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
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
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
					if len(ps.OverlayValues) > 105 && ps.OverlayValues[105].Loc != LocNone {
						d105 = ps.OverlayValues[105]
					}
					if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
						d106 = ps.OverlayValues[106]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
						d113 = ps.OverlayValues[113]
					}
					if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != LocNone {
						d115 = ps.OverlayValues[115]
					}
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
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
					if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
						d132 = ps.OverlayValues[132]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs208 := make([]Reg, 0, 6)
					seenBlockPinnedRegs209 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs209
					for _, r := range []Reg{d13.Reg, d13.Reg2, d13.Reg3} {
						live := d13.Loc == LocRegTriple && (r == d13.Reg || r == d13.Reg2 || r == d13.Reg3)
						if live && !seenBlockPinnedRegs209[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs209[r] = true
							blockPinnedRegs208 = append(blockPinnedRegs208, r)
						}
					}
					for _, r := range []Reg{d19.Reg, d19.Reg2, d19.Reg3} {
						live := d19.Loc == LocRegTriple && (r == d19.Reg || r == d19.Reg2 || r == d19.Reg3)
						if live && !seenBlockPinnedRegs209[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs209[r] = true
							blockPinnedRegs208 = append(blockPinnedRegs208, r)
						}
					}
					unpinBlockRegs210 := func() {
						for _, r := range blockPinnedRegs208 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs210()
					d211 = d7
					_ = d211
					d212 = d13
					_ = d212
					d213 = d3
					_ = d213
					ctx.EnsureDesc(&d1)
					d215 = ctx.EmitSliceElementAddress(&d19, &d1, 16)
					ctx.EnsureDesc(&d215)
					r27 := ctx.AllocRegExcept(d215.Reg)
					ctx.EmitMovRegMem(r27, d215.Reg, 8)
					ctx.EmitMovRegMem(d215.Reg, d215.Reg, 0)
					d214 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d215.Reg, Reg2: r27}
					ctx.BindReg(d215.Reg, &d214)
					ctx.BindReg(r27, &d214)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d216 JITValueDesc
					if d1.Loc == LocImm {
						d216 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d216 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d216)
					}
					if d216.Loc == LocReg && d1.Loc == LocReg && d216.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d216)
					d218 = ctx.EmitSliceElementAddress(&d19, &d216, 16)
					ctx.EnsureDesc(&d218)
					r28 := ctx.AllocRegExcept(d218.Reg)
					ctx.EmitMovRegMem(r28, d218.Reg, 8)
					ctx.EmitMovRegMem(d218.Reg, d218.Reg, 0)
					d217 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d218.Reg, Reg2: r28}
					ctx.BindReg(d218.Reg, &d217)
					ctx.BindReg(r28, &d217)
					ctx.FreeDesc(&d216)
					stackArray219 := ctx.AllocStack(int32(32))
					_ = stackArray219
					ctx.EnsureDesc(&d214)
					ctx.EnsureDesc(&d214)
					ctx.EmitStoreScmerToStack(d214, int32(stackArray219)+int32(0))
					ctx.FreeDesc(&d214)
					ctx.EnsureDesc(&d217)
					ctx.EnsureDesc(&d217)
					ctx.EmitStoreScmerToStack(d217, int32(stackArray219)+int32(16))
					ctx.FreeDesc(&d217)
					d220 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					callbackArgs222 := make([]JITValueDesc, 2)
					callbackArgs222[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray219) + 0}
					callbackArgs222[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray219) + 16}
					var d221 JITValueDesc
					callbackResultOff223 := ctx.AllocStack(16)
					ctx.FreeDesc(&d220)
					if d213.Loc == LocLambdaTemplate && d213.Lambda != nil {
						stableCallbackArgs224 := ctx.StabilizeCallbackArgs(callbackArgs222)
						ctx.ReclaimUntrackedRegs()
						outerRegs225 := ctx.PreserveOuterRegs()
						d221 = JITEmitProcInlineWithOuter(ctx, &d213.Lambda.Proc, d213.Lambda.Outer, stableCallbackArgs224, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff223), ID: 0})
						ctx.RestoreOuterRegs(outerRegs225)
						ctx.ReclaimUntrackedRegs()
					} else {
						d226, knownBuiltin227 := jitEmitKnownDeclaration(ctx, d213, callbackArgs222, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff223), ID: 0})
						if knownBuiltin227 {
							d221 = d226
						} else {
							d228 := jitCopyScmerToPair(ctx, d213)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d228)
							callbackCallArgs = append(callbackCallArgs, callbackArgs222...)
							d221 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d221, int32(callbackResultOff223))
							ctx.FreeDesc(&d221)
							d221 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff223), ID: 0}
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d229 JITValueDesc
					if d1.Loc == LocImm {
						d229 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d229 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d229)
					}
					if d229.Loc == LocReg && d1.Loc == LocReg && d229.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d229)
					d231 = ctx.EmitSliceElementAddress(&d19, &d229, 16)
					ctx.EnsureDesc(&d231)
					r29 := ctx.AllocRegExcept(d231.Reg)
					ctx.EmitMovRegMem(r29, d231.Reg, 8)
					ctx.EmitMovRegMem(d231.Reg, d231.Reg, 0)
					d230 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d231.Reg, Reg2: r29}
					ctx.BindReg(d231.Reg, &d230)
					ctx.BindReg(r29, &d230)
					ctx.FreeDesc(&d229)
					stackArray232 := ctx.AllocStack(int32(48))
					_ = stackArray232
					ctx.EnsureDesc(&d212)
					ctx.EnsureDesc(&d212)
					ctx.EmitStoreScmerToStack(d212, int32(stackArray232)+int32(0))
					ctx.EnsureDesc(&d221)
					ctx.EnsureDesc(&d221)
					ctx.EmitStoreScmerToStack(d221, int32(stackArray232)+int32(16))
					ctx.FreeDesc(&d221)
					ctx.EnsureDesc(&d230)
					ctx.EnsureDesc(&d230)
					ctx.EmitStoreScmerToStack(d230, int32(stackArray232)+int32(32))
					ctx.FreeDesc(&d230)
					d233 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					callbackArgs235 := make([]JITValueDesc, 3)
					callbackArgs235[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray232) + 0}
					callbackArgs235[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray232) + 16}
					callbackArgs235[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray232) + 32}
					var d234 JITValueDesc
					callbackResultOff236 := ctx.AllocStack(16)
					ctx.FreeDesc(&d233)
					if d211.Loc == LocLambdaTemplate && d211.Lambda != nil {
						stableCallbackArgs237 := ctx.StabilizeCallbackArgs(callbackArgs235)
						ctx.ReclaimUntrackedRegs()
						outerRegs238 := ctx.PreserveOuterRegs()
						d234 = JITEmitProcInlineWithOuter(ctx, &d211.Lambda.Proc, d211.Lambda.Outer, stableCallbackArgs237, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff236), ID: 0})
						ctx.RestoreOuterRegs(outerRegs238)
						ctx.ReclaimUntrackedRegs()
					} else {
						d239, knownBuiltin240 := jitEmitKnownDeclaration(ctx, d211, callbackArgs235, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff236), ID: 0})
						if knownBuiltin240 {
							d234 = d239
						} else {
							d241 := jitCopyScmerToPair(ctx, d211)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d241)
							callbackCallArgs = append(callbackCallArgs, callbackArgs235...)
							d234 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d234, int32(callbackResultOff236))
							ctx.FreeDesc(&d234)
							d234 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff236), ID: 0}
						}
					}
					ctx.SyncDesc(&d234)
					ctx.EmitCopyScmerToDesc(&d13, &d234)
					ctx.FreeDesc(&d234)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d242 JITValueDesc
					if d1.Loc == LocImm {
						d242 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d242 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d242)
					}
					if d242.Loc == LocReg && d1.Loc == LocReg && d242.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d242)
					ctx.EmitStoreToStack(d242, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d242)
					ctx.FreeDesc(&d1)
					ps243 := PhiState{General: ps.General}
					ps243.OverlayValues = make([]JITValueDesc, 243)
					ps243.OverlayValues[1] = d1
					ps243.OverlayValues[2] = d2
					ps243.OverlayValues[3] = d3
					ps243.OverlayValues[6] = d6
					ps243.OverlayValues[7] = d7
					ps243.OverlayValues[9] = d9
					ps243.OverlayValues[10] = d10
					ps243.OverlayValues[11] = d11
					ps243.OverlayValues[13] = d13
					ps243.OverlayValues[14] = d14
					ps243.OverlayValues[15] = d15
					ps243.OverlayValues[16] = d16
					ps243.OverlayValues[17] = d17
					ps243.OverlayValues[19] = d19
					ps243.OverlayValues[21] = d21
					ps243.OverlayValues[22] = d22
					ps243.OverlayValues[23] = d23
					ps243.OverlayValues[24] = d24
					ps243.OverlayValues[25] = d25
					ps243.OverlayValues[26] = d26
					ps243.OverlayValues[27] = d27
					ps243.OverlayValues[28] = d28
					ps243.OverlayValues[29] = d29
					ps243.OverlayValues[30] = d30
					ps243.OverlayValues[32] = d32
					ps243.OverlayValues[33] = d33
					ps243.OverlayValues[34] = d34
					ps243.OverlayValues[35] = d35
					ps243.OverlayValues[36] = d36
					ps243.OverlayValues[37] = d37
					ps243.OverlayValues[38] = d38
					ps243.OverlayValues[39] = d39
					ps243.OverlayValues[78] = d78
					ps243.OverlayValues[82] = d82
					ps243.OverlayValues[87] = d87
					ps243.OverlayValues[88] = d88
					ps243.OverlayValues[89] = d89
					ps243.OverlayValues[90] = d90
					ps243.OverlayValues[91] = d91
					ps243.OverlayValues[92] = d92
					ps243.OverlayValues[93] = d93
					ps243.OverlayValues[94] = d94
					ps243.OverlayValues[95] = d95
					ps243.OverlayValues[96] = d96
					ps243.OverlayValues[97] = d97
					ps243.OverlayValues[98] = d98
					ps243.OverlayValues[99] = d99
					ps243.OverlayValues[100] = d100
					ps243.OverlayValues[101] = d101
					ps243.OverlayValues[102] = d102
					ps243.OverlayValues[103] = d103
					ps243.OverlayValues[105] = d105
					ps243.OverlayValues[106] = d106
					ps243.OverlayValues[111] = d111
					ps243.OverlayValues[113] = d113
					ps243.OverlayValues[115] = d115
					ps243.OverlayValues[116] = d116
					ps243.OverlayValues[121] = d121
					ps243.OverlayValues[123] = d123
					ps243.OverlayValues[124] = d124
					ps243.OverlayValues[125] = d125
					ps243.OverlayValues[126] = d126
					ps243.OverlayValues[128] = d128
					ps243.OverlayValues[132] = d132
					ps243.OverlayValues[133] = d133
					ps243.OverlayValues[134] = d134
					ps243.OverlayValues[137] = d137
					ps243.OverlayValues[211] = d211
					ps243.OverlayValues[212] = d212
					ps243.OverlayValues[213] = d213
					ps243.OverlayValues[214] = d214
					ps243.OverlayValues[215] = d215
					ps243.OverlayValues[216] = d216
					ps243.OverlayValues[217] = d217
					ps243.OverlayValues[218] = d218
					ps243.OverlayValues[220] = d220
					ps243.OverlayValues[221] = d221
					ps243.OverlayValues[226] = d226
					ps243.OverlayValues[228] = d228
					ps243.OverlayValues[229] = d229
					ps243.OverlayValues[230] = d230
					ps243.OverlayValues[231] = d231
					ps243.OverlayValues[233] = d233
					ps243.OverlayValues[234] = d234
					ps243.OverlayValues[239] = d239
					ps243.OverlayValues[241] = d241
					ps243.OverlayValues[242] = d242
					ps243.PhiValues = make([]JITValueDesc, 1)
					if ps243.General && bbs[4].Rendered {
						ctx.EmitJmp(lbl5)
						return result
					}
					return bbs[4].RenderPS(ps243)
					return result
				}
				argPinned244 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned244 = append(argPinned244, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned244 = append(argPinned244, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned244 = append(argPinned244, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned244 = append(argPinned244, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned244 {
						ctx.UnprotectReg(r)
					}
				}()
				ps245 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps245)
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
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
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
				var d23 JITValueDesc
				_ = d23
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var d74 JITValueDesc
				_ = d74
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
				var d81 JITValueDesc
				_ = d81
				var d82 JITValueDesc
				_ = d82
				var d83 JITValueDesc
				_ = d83
				var d84 JITValueDesc
				_ = d84
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d95 JITValueDesc
				_ = d95
				var phiBase96 int32
				_ = phiBase96
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
				var d116 JITValueDesc
				_ = d116
				var d121 JITValueDesc
				_ = d121
				var d123 JITValueDesc
				_ = d123
				var d125 JITValueDesc
				_ = d125
				var d126 JITValueDesc
				_ = d126
				var d131 JITValueDesc
				_ = d131
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
				var d143 JITValueDesc
				_ = d143
				var d144 JITValueDesc
				_ = d144
				var d145 JITValueDesc
				_ = d145
				var d148 JITValueDesc
				_ = d148
				var d149 JITValueDesc
				_ = d149
				var d237 JITValueDesc
				_ = d237
				var d238 JITValueDesc
				_ = d238
				var d239 JITValueDesc
				_ = d239
				var d240 JITValueDesc
				_ = d240
				var d241 JITValueDesc
				_ = d241
				var d242 JITValueDesc
				_ = d242
				var d243 JITValueDesc
				_ = d243
				var d245 JITValueDesc
				_ = d245
				var d246 JITValueDesc
				_ = d246
				var d251 JITValueDesc
				_ = d251
				var d253 JITValueDesc
				_ = d253
				var d254 JITValueDesc
				_ = d254
				var d255 JITValueDesc
				_ = d255
				var d256 JITValueDesc
				_ = d256
				var d258 JITValueDesc
				_ = d258
				var d259 JITValueDesc
				_ = d259
				var d263 JITValueDesc
				_ = d263
				var d265 JITValueDesc
				_ = d265
				var d266 JITValueDesc
				_ = d266
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
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
					d11 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("mapkey_assoc_mut")}
					d12 = d10
					_ = d12
					ctx.StabilizeDescForControlFlow(&d12)
					d13 = d11
					_ = d13
					ctx.StabilizeDescForControlFlow(&d13)
					inlineResultOff14 := ctx.AllocStack(int32(24))
					d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: inlineResultOff14}
					inlineResultOff16 := ctx.AllocStack(int32(8))
					d17 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: inlineResultOff16}
					lbl7 := ctx.ReserveLabel()
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
					bbpos_1_6 := int32(-1)
					_ = bbpos_1_6
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d19 = d12
					d19.ID = 0
					d18 = ctx.EmitTagEqualsBorrowed(&d19, tagNil, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					d20 = d18
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					lbl11 := ctx.ReserveLabel()
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						} else {
							ctx.MarkLabel(lbl11)
							ctx.EmitJmp(lbl9)
						}
					} else {
						ctx.EmitCmpRegImm32(d20.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl10)
						ctx.EmitJmp(lbl11)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
						ctx.MarkLabel(lbl11)
						ctx.EmitJmp(lbl9)
					}
					ctx.FreeDesc(&d18)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d22 = d12
					d22.ID = 0
					d21 = ctx.EmitTagEqualsBorrowed(&d22, tagSlice, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					d23 = d21
					ctx.EnsureDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl12 := ctx.ReserveLabel()
					lbl13 := ctx.ReserveLabel()
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl12)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl13)
						}
					} else {
						ctx.EmitCmpRegImm32(d23.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl12)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl13)
					}
					ctx.FreeDesc(&d21)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d25 = d12
					d25.ID = 0
					d24 = ctx.EmitTagEqualsBorrowed(&d25, tagFastDict, JITValueDesc{Loc: LocAny})
					ctx.ReclaimUntrackedRegs()
					d26 = d24
					ctx.EnsureDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
							ctx.MarkLabel(lbl18)
							ctx.EmitJmp(lbl16)
						} else {
							ctx.MarkLabel(lbl19)
							ctx.EmitJmp(lbl17)
						}
					} else {
						ctx.EmitCmpRegImm32(d26.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl18)
						ctx.EmitJmp(lbl19)
						ctx.MarkLabel(lbl18)
						ctx.EmitJmp(lbl16)
						ctx.MarkLabel(lbl19)
						ctx.EmitJmp(lbl17)
					}
					ctx.FreeDesc(&d24)
					bbpos_1_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl17)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					stackArray27 := ctx.AllocStack(int32(0))
					_ = stackArray27
					ctx.ReclaimUntrackedRegs()
					d28 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d28)
					ctx.EmitCopyDescWords(&d15, &d28, 3)
					d29 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EmitZeroDescWords(&d17, 1)
					ctx.EmitJmp(lbl7)
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d30 = jitKnownSliceHeader(ctx, &d12)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d30)
					ctx.EmitCopyDescWords(&d15, &d30, 3)
					d31 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EmitZeroDescWords(&d17, 1)
					ctx.EmitJmp(lbl7)
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d32 JITValueDesc
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocImm {
						panic("FastDict: LocImm not expected at JIT compile time")
					} else if d12.Loc != LocRegPair {
						panic("FastDict: expected Scmer register pair")
					} else {
						ctx.FreeReg(d12.Reg2)
						d32 = JITValueDesc{Loc: LocReg, Reg: d12.Reg}
						ctx.BindReg(d12.Reg, &d32)
						ctx.TransferReg(d12.Reg)
						ctx.BindReg(d12.Reg, &d32)
						d12.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					d33 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EmitZeroDescWords(&d15, 3)
					ctx.EnsureDesc(&d32)
					ctx.EmitCopyDescWords(&d17, &d32, 1)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl7)
					ctx.FreeDesc(&d10)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.EnsureDesc(&d17)
					var d34 JITValueDesc
					if d17.Loc == LocImm {
						d34 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc != LocReg && d17.Loc != LocRegPair && d17.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d17.Reg)
						ctx.EmitCmpRegImm32(d17.Reg, 0)
						ctx.EmitSetcc(r0, CondEqual)
						d34 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d34)
					}
					d35 = d34
					ctx.EnsureDesc(&d35)
					if d35.Loc != LocImm && d35.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d35.Loc == LocImm {
						if d35.Imm.Bool() {
							ps36 := PhiState{General: ps.General}
							ps36.OverlayValues = make([]JITValueDesc, 36)
							ps36.OverlayValues[1] = d1
							ps36.OverlayValues[2] = d2
							ps36.OverlayValues[3] = d3
							ps36.OverlayValues[4] = d4
							ps36.OverlayValues[7] = d7
							ps36.OverlayValues[8] = d8
							ps36.OverlayValues[10] = d10
							ps36.OverlayValues[11] = d11
							ps36.OverlayValues[12] = d12
							ps36.OverlayValues[13] = d13
							ps36.OverlayValues[15] = d15
							ps36.OverlayValues[17] = d17
							ps36.OverlayValues[18] = d18
							ps36.OverlayValues[19] = d19
							ps36.OverlayValues[20] = d20
							ps36.OverlayValues[21] = d21
							ps36.OverlayValues[22] = d22
							ps36.OverlayValues[23] = d23
							ps36.OverlayValues[24] = d24
							ps36.OverlayValues[25] = d25
							ps36.OverlayValues[26] = d26
							ps36.OverlayValues[28] = d28
							ps36.OverlayValues[29] = d29
							ps36.OverlayValues[30] = d30
							ps36.OverlayValues[31] = d31
							ps36.OverlayValues[32] = d32
							ps36.OverlayValues[33] = d33
							ps36.OverlayValues[34] = d34
							ps36.OverlayValues[35] = d35
							return bbs[1].RenderPS(ps36)
						}
						ps37 := PhiState{General: ps.General}
						ps37.OverlayValues = make([]JITValueDesc, 36)
						ps37.OverlayValues[1] = d1
						ps37.OverlayValues[2] = d2
						ps37.OverlayValues[3] = d3
						ps37.OverlayValues[4] = d4
						ps37.OverlayValues[7] = d7
						ps37.OverlayValues[8] = d8
						ps37.OverlayValues[10] = d10
						ps37.OverlayValues[11] = d11
						ps37.OverlayValues[12] = d12
						ps37.OverlayValues[13] = d13
						ps37.OverlayValues[15] = d15
						ps37.OverlayValues[17] = d17
						ps37.OverlayValues[18] = d18
						ps37.OverlayValues[19] = d19
						ps37.OverlayValues[20] = d20
						ps37.OverlayValues[21] = d21
						ps37.OverlayValues[22] = d22
						ps37.OverlayValues[23] = d23
						ps37.OverlayValues[24] = d24
						ps37.OverlayValues[25] = d25
						ps37.OverlayValues[26] = d26
						ps37.OverlayValues[28] = d28
						ps37.OverlayValues[29] = d29
						ps37.OverlayValues[30] = d30
						ps37.OverlayValues[31] = d31
						ps37.OverlayValues[32] = d32
						ps37.OverlayValues[33] = d33
						ps37.OverlayValues[34] = d34
						ps37.OverlayValues[35] = d35
						return bbs[2].RenderPS(ps37)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d35.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl3)
					ps38 := PhiState{General: true}
					ps38.OverlayValues = make([]JITValueDesc, 36)
					ps38.OverlayValues[1] = d1
					ps38.OverlayValues[2] = d2
					ps38.OverlayValues[3] = d3
					ps38.OverlayValues[4] = d4
					ps38.OverlayValues[7] = d7
					ps38.OverlayValues[8] = d8
					ps38.OverlayValues[10] = d10
					ps38.OverlayValues[11] = d11
					ps38.OverlayValues[12] = d12
					ps38.OverlayValues[13] = d13
					ps38.OverlayValues[15] = d15
					ps38.OverlayValues[17] = d17
					ps38.OverlayValues[18] = d18
					ps38.OverlayValues[19] = d19
					ps38.OverlayValues[20] = d20
					ps38.OverlayValues[21] = d21
					ps38.OverlayValues[22] = d22
					ps38.OverlayValues[23] = d23
					ps38.OverlayValues[24] = d24
					ps38.OverlayValues[25] = d25
					ps38.OverlayValues[26] = d26
					ps38.OverlayValues[28] = d28
					ps38.OverlayValues[29] = d29
					ps38.OverlayValues[30] = d30
					ps38.OverlayValues[31] = d31
					ps38.OverlayValues[32] = d32
					ps38.OverlayValues[33] = d33
					ps38.OverlayValues[34] = d34
					ps38.OverlayValues[35] = d35
					ps39 := PhiState{General: true}
					ps39.OverlayValues = make([]JITValueDesc, 36)
					ps39.OverlayValues[1] = d1
					ps39.OverlayValues[2] = d2
					ps39.OverlayValues[3] = d3
					ps39.OverlayValues[4] = d4
					ps39.OverlayValues[7] = d7
					ps39.OverlayValues[8] = d8
					ps39.OverlayValues[10] = d10
					ps39.OverlayValues[11] = d11
					ps39.OverlayValues[12] = d12
					ps39.OverlayValues[13] = d13
					ps39.OverlayValues[15] = d15
					ps39.OverlayValues[17] = d17
					ps39.OverlayValues[18] = d18
					ps39.OverlayValues[19] = d19
					ps39.OverlayValues[20] = d20
					ps39.OverlayValues[21] = d21
					ps39.OverlayValues[22] = d22
					ps39.OverlayValues[23] = d23
					ps39.OverlayValues[24] = d24
					ps39.OverlayValues[25] = d25
					ps39.OverlayValues[26] = d26
					ps39.OverlayValues[28] = d28
					ps39.OverlayValues[29] = d29
					ps39.OverlayValues[30] = d30
					ps39.OverlayValues[31] = d31
					ps39.OverlayValues[32] = d32
					ps39.OverlayValues[33] = d33
					ps39.OverlayValues[34] = d34
					ps39.OverlayValues[35] = d35
					snap40 := d1
					snap41 := d2
					snap42 := d3
					snap43 := d4
					snap44 := d7
					snap45 := d8
					snap46 := d10
					snap47 := d11
					snap48 := d12
					snap49 := d13
					snap50 := d15
					snap51 := d17
					snap52 := d18
					snap53 := d19
					snap54 := d20
					snap55 := d21
					snap56 := d22
					snap57 := d23
					snap58 := d24
					snap59 := d25
					snap60 := d26
					snap61 := d28
					snap62 := d29
					snap63 := d30
					snap64 := d31
					snap65 := d32
					snap66 := d33
					snap67 := d34
					snap68 := d35
					alloc69 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps39)
					}
					ctx.RestoreAllocState(alloc69)
					d1 = snap40
					d2 = snap41
					d3 = snap42
					d4 = snap43
					d7 = snap44
					d8 = snap45
					d10 = snap46
					d11 = snap47
					d12 = snap48
					d13 = snap49
					d15 = snap50
					d17 = snap51
					d18 = snap52
					d19 = snap53
					d20 = snap54
					d21 = snap55
					d22 = snap56
					d23 = snap57
					d24 = snap58
					d25 = snap59
					d26 = snap60
					d28 = snap61
					d29 = snap62
					d30 = snap63
					d31 = snap64
					d32 = snap65
					d33 = snap66
					d34 = snap67
					d35 = snap68
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps38)
					}
					return result
					ctx.FreeDesc(&d34)
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs70 := make([]Reg, 0, 3)
					seenBlockPinnedRegs71 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs71
					for _, r := range []Reg{d15.Reg, d15.Reg2, d15.Reg3} {
						live := d15.Loc == LocRegTriple && (r == d15.Reg || r == d15.Reg2 || r == d15.Reg3)
						if live && !seenBlockPinnedRegs71[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs71[r] = true
							blockPinnedRegs70 = append(blockPinnedRegs70, r)
						}
					}
					unpinBlockRegs72 := func() {
						for _, r := range blockPinnedRegs70 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs72()
					stackArray73 := ctx.AllocStack(int32(0))
					_ = stackArray73
					d74 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					callResults75 := JITEmitGoCallResults(ctx, GoFuncAddr(JITCloneScmerSlice), []JITValueDesc{d15}, []uint8{3}, []uint8{1})
					d76 = callResults75[0]
					ctx.StabilizeDescForControlFlow(&d76)
					d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d77)
					ctx.EnsureDesc(&d78)
					var d80 JITValueDesc
					if d78.Loc == LocImm && d77.Loc == LocImm {
						d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d78.Imm.Int() - d77.Imm.Int())}
					} else {
						r1 := ctx.AllocReg()
						if d78.Loc == LocImm {
							ctx.EmitMovRegImm64(r1, uint64(d78.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r1, d78.Reg)
						}
						if d77.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d77.Imm.Int()))
							ctx.EmitSubInt64(r1, RegR11)
						} else {
							ctx.EmitSubInt64(r1, d77.Reg)
						}
						d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d80)
					}
					var d81 JITValueDesc
					if d15.Loc == LocImm && d77.Loc == LocImm {
						d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d15.Imm.Int() + d77.Imm.Int()*16)}
					} else {
						r2 := ctx.AllocReg()
						if d15.Loc == LocImm {
							ctx.EmitMovRegImm64(r2, uint64(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r2, d15.Reg)
						}
						if d77.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d77.Imm.Int()*16))
							ctx.EmitAddInt64(r2, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r2, d77.Reg)
							ctx.EmitMovRegReg(offsetReg, d77.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r2, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d81)
					}
					var d82 JITValueDesc
					var r3 Reg
					var r4 Reg
					ctx.SyncDesc(&d81)
					ctx.EnsureDesc(&d81)
					if d81.Loc == LocImm {
						r3 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r3, uint64(d81.Imm.Int()))
					} else {
						r3 = d81.Reg
					}
					ctx.ProtectReg(r3)
					ctx.SyncDesc(&d80)
					ctx.EnsureDesc(&d80)
					if d80.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d80.Imm.Int()))
					} else {
						r4 = d80.Reg
					}
					ctx.ProtectReg(r4)
					r5 := ctx.EmitSliceCapAfterLow(&d15, &d77, r3, r4)
					ctx.UnprotectReg(r4)
					ctx.UnprotectReg(r3)
					d82 = JITValueDesc{Loc: LocRegTriple, Reg: r3, Reg2: r4, Reg3: r5}
					ctx.BindReg(r3, &d82)
					ctx.BindReg(r4, &d82)
					ctx.BindReg(r5, &d82)
					ctx.BindReg(r3, &d82)
					ctx.BindReg(r4, &d82)
					ctx.BindReg(r5, &d82)
					d83 = ctx.EmitNewSliceFromGoSlice(&d82)
					ctx.StabilizeDescForControlFlow(&d83)
					ctx.SyncDesc(&d83)
					if d83.Loc == LocReg {
						ctx.ProtectReg(d83.Reg)
					} else if d83.Loc == LocRegPair {
						ctx.ProtectReg(d83.Reg)
						ctx.ProtectReg(d83.Reg2)
					}
					d84 = d83
					if d84.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d84)
					if d84.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d84, int32(bbs[3].PhiBase)+int32(0), 2)
					} else if d84.Loc == LocRegPair || d84.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d84, int32(bbs[3].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d84)
						ctx.EmitStoreToStack(d84, int32(bbs[3].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[3].PhiBase)+int32(0))+8)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(16))
					if d83.Loc == LocReg {
						ctx.UnprotectReg(d83.Reg)
					} else if d83.Loc == LocRegPair {
						ctx.UnprotectReg(d83.Reg)
						ctx.UnprotectReg(d83.Reg2)
					}
					ps85 := PhiState{General: ps.General}
					ps85.OverlayValues = make([]JITValueDesc, 85)
					ps85.OverlayValues[1] = d1
					ps85.OverlayValues[2] = d2
					ps85.OverlayValues[3] = d3
					ps85.OverlayValues[4] = d4
					ps85.OverlayValues[7] = d7
					ps85.OverlayValues[8] = d8
					ps85.OverlayValues[10] = d10
					ps85.OverlayValues[11] = d11
					ps85.OverlayValues[12] = d12
					ps85.OverlayValues[13] = d13
					ps85.OverlayValues[15] = d15
					ps85.OverlayValues[17] = d17
					ps85.OverlayValues[18] = d18
					ps85.OverlayValues[19] = d19
					ps85.OverlayValues[20] = d20
					ps85.OverlayValues[21] = d21
					ps85.OverlayValues[22] = d22
					ps85.OverlayValues[23] = d23
					ps85.OverlayValues[24] = d24
					ps85.OverlayValues[25] = d25
					ps85.OverlayValues[26] = d26
					ps85.OverlayValues[28] = d28
					ps85.OverlayValues[29] = d29
					ps85.OverlayValues[30] = d30
					ps85.OverlayValues[31] = d31
					ps85.OverlayValues[32] = d32
					ps85.OverlayValues[33] = d33
					ps85.OverlayValues[34] = d34
					ps85.OverlayValues[35] = d35
					ps85.OverlayValues[74] = d74
					ps85.OverlayValues[76] = d76
					ps85.OverlayValues[77] = d77
					ps85.OverlayValues[78] = d78
					ps85.OverlayValues[79] = d79
					ps85.OverlayValues[80] = d80
					ps85.OverlayValues[81] = d81
					ps85.OverlayValues[82] = d82
					ps85.OverlayValues[83] = d83
					ps85.OverlayValues[84] = d84
					ps85.PhiValues = make([]JITValueDesc, 2)
					d86 = d83
					ps85.PhiValues[0] = d86
					d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps85.PhiValues[1] = d87
					if ps85.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps85)
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
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
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs88 := make([]Reg, 0, 3)
					seenBlockPinnedRegs89 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs89
					for _, r := range []Reg{d17.Reg, d17.Reg2, d17.Reg3} {
						live := d17.Loc == LocRegTriple && (r == d17.Reg || r == d17.Reg2 || r == d17.Reg3)
						if live && !seenBlockPinnedRegs89[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs89[r] = true
							blockPinnedRegs88 = append(blockPinnedRegs88, r)
						}
					}
					unpinBlockRegs90 := func() {
						for _, r := range blockPinnedRegs88 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs90()
					d91 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: nil}
					ctx.SyncDesc(&d91)
					d92 = jitMaterializeVirtualSlice(ctx, d91, JITValueDesc{Loc: LocAny})
					d93 = jitCopyScmerToPair(ctx, d92)
					ctx.FreeDesc(&d92)
					scmerCellOff94 := ctx.AllocStack(16)
					ctx.EmitStoreScmerToStack(d93, int32(scmerCellOff94))
					d95 = JITValueDesc{Loc: LocStackPair, Type: d93.Type, StackOff: int32(scmerCellOff94)}
					ctx.FreeDesc(&d93)
					ctx.EnsureDesc(&d17)
					phiBase96 = ctx.AllocStack(int32(16))
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase96) + int32(0)}
					lbl22 := ctx.ReserveLabel()
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
					bbpos_2_5 := int32(-1)
					_ = bbpos_2_5
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase96) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase96)+int32(0))
					bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase96) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d97)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d98 JITValueDesc
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						fieldAddr := uintptr(d17.Imm.Int()) + 0
						r6 := ctx.AllocReg()
						r7 := ctx.AllocRegExcept(r6)
						r8 := ctx.AllocRegExcept(r6, r7)
						ctx.EmitMovRegMem64(r6, fieldAddr)
						ctx.EmitMovRegMem64(r7, fieldAddr+8)
						ctx.EmitMovRegMem64(r8, fieldAddr+16)
						d98 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
						ctx.BindReg(r6, &d98)
						ctx.BindReg(r7, &d98)
						ctx.BindReg(r8, &d98)
					} else {
						off := int32(0)
						baseReg := d17.Reg
						r9 := ctx.AllocRegExcept(baseReg)
						r10 := ctx.AllocRegExcept(baseReg, r9)
						r11 := ctx.AllocRegExcept(baseReg, r9, r10)
						ctx.EmitMovRegMem(r9, baseReg, off)
						ctx.EmitMovRegMem(r10, baseReg, off+8)
						ctx.EmitMovRegMem(r11, baseReg, off+16)
						d98 = JITValueDesc{Loc: LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11}
						ctx.BindReg(r9, &d98)
						ctx.BindReg(r10, &d98)
						ctx.BindReg(r11, &d98)
					}
					ctx.ReclaimUntrackedRegs()
					var d99 JITValueDesc
					if d98.SliceSizeKnown {
						d99 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d98.KnownSliceLen))}
					} else if d98.Loc == LocImm {
						d99 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d98.StackOff))}
					} else if d98.Loc == LocStackTriple {
						d99 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d98.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d98)
						if d98.Loc == LocRegPair || d98.Loc == LocRegTriple {
							d99 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d98.Reg2, ID: 0}
						} else if d98.Loc == LocReg {
							d99 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d98.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d99)
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d99)
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d99)
					var d100 JITValueDesc
					if d97.Loc == LocImm && d99.Loc == LocImm {
						d100 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d97.Imm.Int() < d99.Imm.Int())}
					} else if d99.Loc == LocImm {
						r12 := ctx.AllocRegExcept(d97.Reg)
						if d99.Imm.Int() >= -2147483648 && d99.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d97.Reg, int32(d99.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d99.Imm.Int()))
							ctx.EmitCmpInt64(d97.Reg, RegR11)
						}
						ctx.EmitSetcc(r12, CondSignedLess)
						d100 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
						ctx.BindReg(r12, &d100)
					} else if d97.Loc == LocImm {
						r13 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d97.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d99.Reg)
						ctx.EmitSetcc(r13, CondSignedLess)
						d100 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
						ctx.BindReg(r13, &d100)
					} else {
						r14 := ctx.AllocRegExcept(d97.Reg)
						ctx.EmitCmpInt64(d97.Reg, d99.Reg)
						ctx.EmitSetcc(r14, CondSignedLess)
						d100 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
						ctx.BindReg(r14, &d100)
					}
					ctx.FreeDesc(&d99)
					ctx.ReclaimUntrackedRegs()
					d101 = d100
					ctx.EnsureDesc(&d101)
					if d101.Loc != LocImm && d101.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					lbl26 := ctx.ReserveLabel()
					if d101.Loc == LocImm {
						if d101.Imm.Bool() {
							ctx.MarkLabel(lbl25)
							ctx.EmitJmp(lbl23)
						} else {
							ctx.MarkLabel(lbl26)
							ctx.EmitJmp(lbl24)
						}
					} else {
						ctx.EmitCmpRegImm32(d101.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl25)
						ctx.EmitJmp(lbl26)
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl23)
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl24)
					}
					ctx.FreeDesc(&d100)
					bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl24)
					ctx.ResolveFixups()
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase96) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl22)
					bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl23)
					ctx.ResolveFixups()
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase96) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d102 JITValueDesc
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						fieldAddr := uintptr(d17.Imm.Int()) + 0
						r15 := ctx.AllocReg()
						r16 := ctx.AllocRegExcept(r15)
						r17 := ctx.AllocRegExcept(r15, r16)
						ctx.EmitMovRegMem64(r15, fieldAddr)
						ctx.EmitMovRegMem64(r16, fieldAddr+8)
						ctx.EmitMovRegMem64(r17, fieldAddr+16)
						d102 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
						ctx.BindReg(r15, &d102)
						ctx.BindReg(r16, &d102)
						ctx.BindReg(r17, &d102)
					} else {
						off := int32(0)
						baseReg := d17.Reg
						r18 := ctx.AllocRegExcept(baseReg)
						r19 := ctx.AllocRegExcept(baseReg, r18)
						r20 := ctx.AllocRegExcept(baseReg, r18, r19)
						ctx.EmitMovRegMem(r18, baseReg, off)
						ctx.EmitMovRegMem(r19, baseReg, off+8)
						ctx.EmitMovRegMem(r20, baseReg, off+16)
						d102 = JITValueDesc{Loc: LocRegTriple, Reg: r18, Reg2: r19, Reg3: r20}
						ctx.BindReg(r18, &d102)
						ctx.BindReg(r19, &d102)
						ctx.BindReg(r20, &d102)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					ctx.ReclaimUntrackedRegs()
					d104 = ctx.EmitSliceElementAddress(&d102, &d97, 16)
					ctx.EnsureDesc(&d104)
					r21 := ctx.AllocRegExcept(d104.Reg)
					ctx.EmitMovRegMem(r21, d104.Reg, 8)
					ctx.EmitMovRegMem(d104.Reg, d104.Reg, 0)
					d103 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d104.Reg, Reg2: r21}
					ctx.BindReg(d104.Reg, &d103)
					ctx.BindReg(r21, &d103)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					var d105 JITValueDesc
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						fieldAddr := uintptr(d17.Imm.Int()) + 0
						r22 := ctx.AllocReg()
						r23 := ctx.AllocRegExcept(r22)
						r24 := ctx.AllocRegExcept(r22, r23)
						ctx.EmitMovRegMem64(r22, fieldAddr)
						ctx.EmitMovRegMem64(r23, fieldAddr+8)
						ctx.EmitMovRegMem64(r24, fieldAddr+16)
						d105 = JITValueDesc{Loc: LocRegTriple, Reg: r22, Reg2: r23, Reg3: r24}
						ctx.BindReg(r22, &d105)
						ctx.BindReg(r23, &d105)
						ctx.BindReg(r24, &d105)
					} else {
						off := int32(0)
						baseReg := d17.Reg
						r25 := ctx.AllocRegExcept(baseReg)
						r26 := ctx.AllocRegExcept(baseReg, r25)
						r27 := ctx.AllocRegExcept(baseReg, r25, r26)
						ctx.EmitMovRegMem(r25, baseReg, off)
						ctx.EmitMovRegMem(r26, baseReg, off+8)
						ctx.EmitMovRegMem(r27, baseReg, off+16)
						d105 = JITValueDesc{Loc: LocRegTriple, Reg: r25, Reg2: r26, Reg3: r27}
						ctx.BindReg(r25, &d105)
						ctx.BindReg(r26, &d105)
						ctx.BindReg(r27, &d105)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d97)
					var d106 JITValueDesc
					if d97.Loc == LocImm {
						d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d97.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d97.Reg)
						ctx.EmitMovRegReg(scratch, d97.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d106)
					}
					if d106.Loc == LocReg && d97.Loc == LocReg && d106.Reg == d97.Reg {
						ctx.TransferReg(d97.Reg)
						d97.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					d108 = ctx.EmitSliceElementAddress(&d105, &d106, 16)
					ctx.EnsureDesc(&d108)
					r28 := ctx.AllocRegExcept(d108.Reg)
					ctx.EmitMovRegMem(r28, d108.Reg, 8)
					ctx.EmitMovRegMem(d108.Reg, d108.Reg, 0)
					d107 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d108.Reg, Reg2: r28}
					ctx.BindReg(d108.Reg, &d107)
					ctx.BindReg(r28, &d107)
					ctx.FreeDesc(&d106)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d103)
					ctx.EnsureDesc(&d107)
					d109 = d103
					_ = d109
					ctx.StabilizeDescForControlFlow(&d109)
					d110 = d107
					_ = d110
					ctx.StabilizeDescForControlFlow(&d110)
					bbpos_3_0 := int32(-1)
					_ = bbpos_3_0
					bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d111 = d8
					_ = d111
					ctx.ReclaimUntrackedRegs()
					d112 = d95
					_ = d112
					ctx.ReclaimUntrackedRegs()
					d113 = d4
					_ = d113
					ctx.ReclaimUntrackedRegs()
					stackArray114 := ctx.AllocStack(int32(32))
					_ = stackArray114
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d109)
					ctx.EnsureDesc(&d109)
					ctx.EmitStoreScmerToStack(d109, int32(stackArray114)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					ctx.EnsureDesc(&d110)
					ctx.EmitStoreScmerToStack(d110, int32(stackArray114)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d115 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					ctx.ReclaimUntrackedRegs()
					callbackArgs117 := make([]JITValueDesc, 2)
					callbackArgs117[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray114) + 0}
					callbackArgs117[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray114) + 16}
					var d116 JITValueDesc
					callbackResultOff118 := ctx.AllocStack(16)
					ctx.FreeDesc(&d115)
					if d113.Loc == LocLambdaTemplate && d113.Lambda != nil {
						stableCallbackArgs119 := ctx.StabilizeCallbackArgs(callbackArgs117)
						ctx.ReclaimUntrackedRegs()
						outerRegs120 := ctx.PreserveOuterRegs()
						d116 = JITEmitProcInlineWithOuter(ctx, &d113.Lambda.Proc, d113.Lambda.Outer, stableCallbackArgs119, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff118), ID: 0})
						ctx.RestoreOuterRegs(outerRegs120)
						ctx.ReclaimUntrackedRegs()
					} else {
						d121, knownBuiltin122 := jitEmitKnownDeclaration(ctx, d113, callbackArgs117, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff118), ID: 0})
						if knownBuiltin122 {
							d116 = d121
						} else {
							d123 := jitCopyScmerToPair(ctx, d113)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d123)
							callbackCallArgs = append(callbackCallArgs, callbackArgs117...)
							d116 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d116, int32(callbackResultOff118))
							ctx.FreeDesc(&d116)
							d116 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff118), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					stackArray124 := ctx.AllocStack(int32(48))
					_ = stackArray124
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d112)
					ctx.EnsureDesc(&d112)
					ctx.EmitStoreScmerToStack(d112, int32(stackArray124)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d116)
					ctx.EnsureDesc(&d116)
					ctx.EmitStoreScmerToStack(d116, int32(stackArray124)+int32(16))
					ctx.FreeDesc(&d116)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d110)
					ctx.EnsureDesc(&d110)
					ctx.EmitStoreScmerToStack(d110, int32(stackArray124)+int32(32))
					ctx.ReclaimUntrackedRegs()
					d125 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					ctx.ReclaimUntrackedRegs()
					callbackArgs127 := make([]JITValueDesc, 3)
					callbackArgs127[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray124) + 0}
					callbackArgs127[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray124) + 16}
					callbackArgs127[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray124) + 32}
					var d126 JITValueDesc
					callbackResultOff128 := ctx.AllocStack(16)
					ctx.FreeDesc(&d125)
					if d111.Loc == LocLambdaTemplate && d111.Lambda != nil {
						stableCallbackArgs129 := ctx.StabilizeCallbackArgs(callbackArgs127)
						ctx.ReclaimUntrackedRegs()
						outerRegs130 := ctx.PreserveOuterRegs()
						d126 = JITEmitProcInlineWithOuter(ctx, &d111.Lambda.Proc, d111.Lambda.Outer, stableCallbackArgs129, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff128), ID: 0})
						ctx.RestoreOuterRegs(outerRegs130)
						ctx.ReclaimUntrackedRegs()
					} else {
						d131, knownBuiltin132 := jitEmitKnownDeclaration(ctx, d111, callbackArgs127, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff128), ID: 0})
						if knownBuiltin132 {
							d126 = d131
						} else {
							d133 := jitCopyScmerToPair(ctx, d111)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d133)
							callbackCallArgs = append(callbackCallArgs, callbackArgs127...)
							d126 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d126, int32(callbackResultOff128))
							ctx.FreeDesc(&d126)
							d126 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff128), ID: 0}
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d126)
					ctx.EmitCopyScmerToDesc(&d95, &d126)
					ctx.FreeDesc(&d126)
					ctx.ReclaimUntrackedRegs()
					d134 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.FreeDesc(&d103)
					ctx.FreeDesc(&d107)
					ctx.ReclaimUntrackedRegs()
					d135 = d134
					ctx.EnsureDesc(&d135)
					if d135.Loc != LocImm && d135.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					lbl29 := ctx.ReserveLabel()
					lbl30 := ctx.ReserveLabel()
					if d135.Loc == LocImm {
						if d135.Imm.Bool() {
							ctx.MarkLabel(lbl29)
							ctx.EmitJmp(lbl27)
						} else {
							ctx.MarkLabel(lbl30)
							ctx.EmitJmp(lbl28)
						}
					} else {
						ctx.EmitCmpRegImm32(d135.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl29)
						ctx.EmitJmp(lbl30)
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl27)
						ctx.MarkLabel(lbl30)
						ctx.EmitJmp(lbl28)
					}
					ctx.FreeDesc(&d134)
					bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl28)
					ctx.ResolveFixups()
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase96) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl22)
					bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl27)
					ctx.ResolveFixups()
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase96) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d97)
					ctx.EnsureDesc(&d97)
					var d136 JITValueDesc
					if d97.Loc == LocImm {
						d136 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d97.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d97.Reg)
						ctx.EmitMovRegReg(scratch, d97.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d136 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d136)
					}
					if d136.Loc == LocReg && d97.Loc == LocReg && d136.Reg == d97.Reg {
						ctx.TransferReg(d97.Reg)
						d97.Loc = LocNone
					}
					ctx.EnsureDesc(&d136)
					ctx.EmitStoreToStack(d136, int32(phiBase96)+int32(0))
					ctx.StabilizeDescForControlFlow(&d136)
					ctx.FreeDesc(&d97)
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmpToPos(bbpos_2_1)
					ctx.MarkLabel(lbl22)
					ctx.FreeDesc(&d17)
					d137 = d95
					_ = d137
					ctx.EnsureDesc(&d137)
					if d137.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d137, &result)
						result.Type = d137.Type
					} else {
						switch d137.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d137)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d137)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d137)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d137, &result)
							result.Type = d137.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d138 := ps.PhiValues[0]
							ctx.EnsureDesc(&d138)
							ctx.EmitStoreScmerToStack(d138, int32(bbs[3].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d139 := ps.PhiValues[1]
							ctx.EnsureDesc(&d139)
							ctx.EmitStoreToStack(d139, int32(bbs[3].PhiBase)+int32(16))
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
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
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d2 = ps.PhiValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs140 := make([]Reg, 0, 3)
					seenBlockPinnedRegs141 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs141
					for _, r := range []Reg{d76.Reg, d76.Reg2, d76.Reg3} {
						live := d76.Loc == LocRegTriple && (r == d76.Reg || r == d76.Reg2 || r == d76.Reg3)
						if live && !seenBlockPinnedRegs141[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs141[r] = true
							blockPinnedRegs140 = append(blockPinnedRegs140, r)
						}
					}
					unpinBlockRegs142 := func() {
						for _, r := range blockPinnedRegs140 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs142()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					var d143 JITValueDesc
					if d76.SliceSizeKnown {
						d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d76.KnownSliceLen))}
					} else if d76.Loc == LocImm {
						d143 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d76.StackOff))}
					} else if d76.Loc == LocStackTriple {
						d143 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d76.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d76)
						if d76.Loc == LocRegPair || d76.Loc == LocRegTriple {
							d143 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d76.Reg2, ID: 0}
						} else if d76.Loc == LocReg {
							d143 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d76.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d143)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d143)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d143)
					var d144 JITValueDesc
					if d2.Loc == LocImm && d143.Loc == LocImm {
						d144 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d143.Imm.Int())}
					} else if d143.Loc == LocImm {
						r29 := ctx.AllocRegExcept(d2.Reg)
						if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d143.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d143.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, RegR11)
						}
						ctx.EmitSetcc(r29, CondSignedLess)
						d144 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r29}
						ctx.BindReg(r29, &d144)
					} else if d2.Loc == LocImm {
						r30 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d143.Reg)
						ctx.EmitSetcc(r30, CondSignedLess)
						d144 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r30}
						ctx.BindReg(r30, &d144)
					} else {
						r31 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d143.Reg)
						ctx.EmitSetcc(r31, CondSignedLess)
						d144 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r31}
						ctx.BindReg(r31, &d144)
					}
					ctx.FreeDesc(&d143)
					d145 = d144
					ctx.EnsureDesc(&d145)
					if d145.Loc != LocImm && d145.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d145.Loc == LocImm {
						if d145.Imm.Bool() {
							ps146 := PhiState{General: ps.General}
							ps146.OverlayValues = make([]JITValueDesc, 146)
							ps146.OverlayValues[1] = d1
							ps146.OverlayValues[2] = d2
							ps146.OverlayValues[3] = d3
							ps146.OverlayValues[4] = d4
							ps146.OverlayValues[7] = d7
							ps146.OverlayValues[8] = d8
							ps146.OverlayValues[10] = d10
							ps146.OverlayValues[11] = d11
							ps146.OverlayValues[12] = d12
							ps146.OverlayValues[13] = d13
							ps146.OverlayValues[15] = d15
							ps146.OverlayValues[17] = d17
							ps146.OverlayValues[18] = d18
							ps146.OverlayValues[19] = d19
							ps146.OverlayValues[20] = d20
							ps146.OverlayValues[21] = d21
							ps146.OverlayValues[22] = d22
							ps146.OverlayValues[23] = d23
							ps146.OverlayValues[24] = d24
							ps146.OverlayValues[25] = d25
							ps146.OverlayValues[26] = d26
							ps146.OverlayValues[28] = d28
							ps146.OverlayValues[29] = d29
							ps146.OverlayValues[30] = d30
							ps146.OverlayValues[31] = d31
							ps146.OverlayValues[32] = d32
							ps146.OverlayValues[33] = d33
							ps146.OverlayValues[34] = d34
							ps146.OverlayValues[35] = d35
							ps146.OverlayValues[74] = d74
							ps146.OverlayValues[76] = d76
							ps146.OverlayValues[77] = d77
							ps146.OverlayValues[78] = d78
							ps146.OverlayValues[79] = d79
							ps146.OverlayValues[80] = d80
							ps146.OverlayValues[81] = d81
							ps146.OverlayValues[82] = d82
							ps146.OverlayValues[83] = d83
							ps146.OverlayValues[84] = d84
							ps146.OverlayValues[86] = d86
							ps146.OverlayValues[87] = d87
							ps146.OverlayValues[91] = d91
							ps146.OverlayValues[92] = d92
							ps146.OverlayValues[93] = d93
							ps146.OverlayValues[95] = d95
							ps146.OverlayValues[97] = d97
							ps146.OverlayValues[98] = d98
							ps146.OverlayValues[99] = d99
							ps146.OverlayValues[100] = d100
							ps146.OverlayValues[101] = d101
							ps146.OverlayValues[102] = d102
							ps146.OverlayValues[103] = d103
							ps146.OverlayValues[104] = d104
							ps146.OverlayValues[105] = d105
							ps146.OverlayValues[106] = d106
							ps146.OverlayValues[107] = d107
							ps146.OverlayValues[108] = d108
							ps146.OverlayValues[109] = d109
							ps146.OverlayValues[110] = d110
							ps146.OverlayValues[111] = d111
							ps146.OverlayValues[112] = d112
							ps146.OverlayValues[113] = d113
							ps146.OverlayValues[115] = d115
							ps146.OverlayValues[116] = d116
							ps146.OverlayValues[121] = d121
							ps146.OverlayValues[123] = d123
							ps146.OverlayValues[125] = d125
							ps146.OverlayValues[126] = d126
							ps146.OverlayValues[131] = d131
							ps146.OverlayValues[133] = d133
							ps146.OverlayValues[134] = d134
							ps146.OverlayValues[135] = d135
							ps146.OverlayValues[136] = d136
							ps146.OverlayValues[137] = d137
							ps146.OverlayValues[138] = d138
							ps146.OverlayValues[139] = d139
							ps146.OverlayValues[143] = d143
							ps146.OverlayValues[144] = d144
							ps146.OverlayValues[145] = d145
							return bbs[4].RenderPS(ps146)
						}
						ps147 := PhiState{General: ps.General}
						ps147.OverlayValues = make([]JITValueDesc, 146)
						ps147.OverlayValues[1] = d1
						ps147.OverlayValues[2] = d2
						ps147.OverlayValues[3] = d3
						ps147.OverlayValues[4] = d4
						ps147.OverlayValues[7] = d7
						ps147.OverlayValues[8] = d8
						ps147.OverlayValues[10] = d10
						ps147.OverlayValues[11] = d11
						ps147.OverlayValues[12] = d12
						ps147.OverlayValues[13] = d13
						ps147.OverlayValues[15] = d15
						ps147.OverlayValues[17] = d17
						ps147.OverlayValues[18] = d18
						ps147.OverlayValues[19] = d19
						ps147.OverlayValues[20] = d20
						ps147.OverlayValues[21] = d21
						ps147.OverlayValues[22] = d22
						ps147.OverlayValues[23] = d23
						ps147.OverlayValues[24] = d24
						ps147.OverlayValues[25] = d25
						ps147.OverlayValues[26] = d26
						ps147.OverlayValues[28] = d28
						ps147.OverlayValues[29] = d29
						ps147.OverlayValues[30] = d30
						ps147.OverlayValues[31] = d31
						ps147.OverlayValues[32] = d32
						ps147.OverlayValues[33] = d33
						ps147.OverlayValues[34] = d34
						ps147.OverlayValues[35] = d35
						ps147.OverlayValues[74] = d74
						ps147.OverlayValues[76] = d76
						ps147.OverlayValues[77] = d77
						ps147.OverlayValues[78] = d78
						ps147.OverlayValues[79] = d79
						ps147.OverlayValues[80] = d80
						ps147.OverlayValues[81] = d81
						ps147.OverlayValues[82] = d82
						ps147.OverlayValues[83] = d83
						ps147.OverlayValues[84] = d84
						ps147.OverlayValues[86] = d86
						ps147.OverlayValues[87] = d87
						ps147.OverlayValues[91] = d91
						ps147.OverlayValues[92] = d92
						ps147.OverlayValues[93] = d93
						ps147.OverlayValues[95] = d95
						ps147.OverlayValues[97] = d97
						ps147.OverlayValues[98] = d98
						ps147.OverlayValues[99] = d99
						ps147.OverlayValues[100] = d100
						ps147.OverlayValues[101] = d101
						ps147.OverlayValues[102] = d102
						ps147.OverlayValues[103] = d103
						ps147.OverlayValues[104] = d104
						ps147.OverlayValues[105] = d105
						ps147.OverlayValues[106] = d106
						ps147.OverlayValues[107] = d107
						ps147.OverlayValues[108] = d108
						ps147.OverlayValues[109] = d109
						ps147.OverlayValues[110] = d110
						ps147.OverlayValues[111] = d111
						ps147.OverlayValues[112] = d112
						ps147.OverlayValues[113] = d113
						ps147.OverlayValues[115] = d115
						ps147.OverlayValues[116] = d116
						ps147.OverlayValues[121] = d121
						ps147.OverlayValues[123] = d123
						ps147.OverlayValues[125] = d125
						ps147.OverlayValues[126] = d126
						ps147.OverlayValues[131] = d131
						ps147.OverlayValues[133] = d133
						ps147.OverlayValues[134] = d134
						ps147.OverlayValues[135] = d135
						ps147.OverlayValues[136] = d136
						ps147.OverlayValues[137] = d137
						ps147.OverlayValues[138] = d138
						ps147.OverlayValues[139] = d139
						ps147.OverlayValues[143] = d143
						ps147.OverlayValues[144] = d144
						ps147.OverlayValues[145] = d145
						return bbs[5].RenderPS(ps147)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d148 := ps.PhiValues[0]
							ctx.EnsureDesc(&d148)
							ctx.EmitStoreScmerToStack(d148, int32(bbs[3].PhiBase)+int32(0))
						}
						if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
							d149 := ps.PhiValues[1]
							ctx.EnsureDesc(&d149)
							ctx.EmitStoreToStack(d149, int32(bbs[3].PhiBase)+int32(16))
						}
						ps.General = true
						return bbs[3].RenderPS(ps)
					}
					lbl31 := ctx.ReserveLabel()
					lbl32 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d145.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl31)
					ctx.EmitJmp(lbl32)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl6)
					ps150 := PhiState{General: true}
					ps150.OverlayValues = make([]JITValueDesc, 150)
					ps150.OverlayValues[1] = d1
					ps150.OverlayValues[2] = d2
					ps150.OverlayValues[3] = d3
					ps150.OverlayValues[4] = d4
					ps150.OverlayValues[7] = d7
					ps150.OverlayValues[8] = d8
					ps150.OverlayValues[10] = d10
					ps150.OverlayValues[11] = d11
					ps150.OverlayValues[12] = d12
					ps150.OverlayValues[13] = d13
					ps150.OverlayValues[15] = d15
					ps150.OverlayValues[17] = d17
					ps150.OverlayValues[18] = d18
					ps150.OverlayValues[19] = d19
					ps150.OverlayValues[20] = d20
					ps150.OverlayValues[21] = d21
					ps150.OverlayValues[22] = d22
					ps150.OverlayValues[23] = d23
					ps150.OverlayValues[24] = d24
					ps150.OverlayValues[25] = d25
					ps150.OverlayValues[26] = d26
					ps150.OverlayValues[28] = d28
					ps150.OverlayValues[29] = d29
					ps150.OverlayValues[30] = d30
					ps150.OverlayValues[31] = d31
					ps150.OverlayValues[32] = d32
					ps150.OverlayValues[33] = d33
					ps150.OverlayValues[34] = d34
					ps150.OverlayValues[35] = d35
					ps150.OverlayValues[74] = d74
					ps150.OverlayValues[76] = d76
					ps150.OverlayValues[77] = d77
					ps150.OverlayValues[78] = d78
					ps150.OverlayValues[79] = d79
					ps150.OverlayValues[80] = d80
					ps150.OverlayValues[81] = d81
					ps150.OverlayValues[82] = d82
					ps150.OverlayValues[83] = d83
					ps150.OverlayValues[84] = d84
					ps150.OverlayValues[86] = d86
					ps150.OverlayValues[87] = d87
					ps150.OverlayValues[91] = d91
					ps150.OverlayValues[92] = d92
					ps150.OverlayValues[93] = d93
					ps150.OverlayValues[95] = d95
					ps150.OverlayValues[97] = d97
					ps150.OverlayValues[98] = d98
					ps150.OverlayValues[99] = d99
					ps150.OverlayValues[100] = d100
					ps150.OverlayValues[101] = d101
					ps150.OverlayValues[102] = d102
					ps150.OverlayValues[103] = d103
					ps150.OverlayValues[104] = d104
					ps150.OverlayValues[105] = d105
					ps150.OverlayValues[106] = d106
					ps150.OverlayValues[107] = d107
					ps150.OverlayValues[108] = d108
					ps150.OverlayValues[109] = d109
					ps150.OverlayValues[110] = d110
					ps150.OverlayValues[111] = d111
					ps150.OverlayValues[112] = d112
					ps150.OverlayValues[113] = d113
					ps150.OverlayValues[115] = d115
					ps150.OverlayValues[116] = d116
					ps150.OverlayValues[121] = d121
					ps150.OverlayValues[123] = d123
					ps150.OverlayValues[125] = d125
					ps150.OverlayValues[126] = d126
					ps150.OverlayValues[131] = d131
					ps150.OverlayValues[133] = d133
					ps150.OverlayValues[134] = d134
					ps150.OverlayValues[135] = d135
					ps150.OverlayValues[136] = d136
					ps150.OverlayValues[137] = d137
					ps150.OverlayValues[138] = d138
					ps150.OverlayValues[139] = d139
					ps150.OverlayValues[143] = d143
					ps150.OverlayValues[144] = d144
					ps150.OverlayValues[145] = d145
					ps150.OverlayValues[148] = d148
					ps150.OverlayValues[149] = d149
					ps151 := PhiState{General: true}
					ps151.OverlayValues = make([]JITValueDesc, 150)
					ps151.OverlayValues[1] = d1
					ps151.OverlayValues[2] = d2
					ps151.OverlayValues[3] = d3
					ps151.OverlayValues[4] = d4
					ps151.OverlayValues[7] = d7
					ps151.OverlayValues[8] = d8
					ps151.OverlayValues[10] = d10
					ps151.OverlayValues[11] = d11
					ps151.OverlayValues[12] = d12
					ps151.OverlayValues[13] = d13
					ps151.OverlayValues[15] = d15
					ps151.OverlayValues[17] = d17
					ps151.OverlayValues[18] = d18
					ps151.OverlayValues[19] = d19
					ps151.OverlayValues[20] = d20
					ps151.OverlayValues[21] = d21
					ps151.OverlayValues[22] = d22
					ps151.OverlayValues[23] = d23
					ps151.OverlayValues[24] = d24
					ps151.OverlayValues[25] = d25
					ps151.OverlayValues[26] = d26
					ps151.OverlayValues[28] = d28
					ps151.OverlayValues[29] = d29
					ps151.OverlayValues[30] = d30
					ps151.OverlayValues[31] = d31
					ps151.OverlayValues[32] = d32
					ps151.OverlayValues[33] = d33
					ps151.OverlayValues[34] = d34
					ps151.OverlayValues[35] = d35
					ps151.OverlayValues[74] = d74
					ps151.OverlayValues[76] = d76
					ps151.OverlayValues[77] = d77
					ps151.OverlayValues[78] = d78
					ps151.OverlayValues[79] = d79
					ps151.OverlayValues[80] = d80
					ps151.OverlayValues[81] = d81
					ps151.OverlayValues[82] = d82
					ps151.OverlayValues[83] = d83
					ps151.OverlayValues[84] = d84
					ps151.OverlayValues[86] = d86
					ps151.OverlayValues[87] = d87
					ps151.OverlayValues[91] = d91
					ps151.OverlayValues[92] = d92
					ps151.OverlayValues[93] = d93
					ps151.OverlayValues[95] = d95
					ps151.OverlayValues[97] = d97
					ps151.OverlayValues[98] = d98
					ps151.OverlayValues[99] = d99
					ps151.OverlayValues[100] = d100
					ps151.OverlayValues[101] = d101
					ps151.OverlayValues[102] = d102
					ps151.OverlayValues[103] = d103
					ps151.OverlayValues[104] = d104
					ps151.OverlayValues[105] = d105
					ps151.OverlayValues[106] = d106
					ps151.OverlayValues[107] = d107
					ps151.OverlayValues[108] = d108
					ps151.OverlayValues[109] = d109
					ps151.OverlayValues[110] = d110
					ps151.OverlayValues[111] = d111
					ps151.OverlayValues[112] = d112
					ps151.OverlayValues[113] = d113
					ps151.OverlayValues[115] = d115
					ps151.OverlayValues[116] = d116
					ps151.OverlayValues[121] = d121
					ps151.OverlayValues[123] = d123
					ps151.OverlayValues[125] = d125
					ps151.OverlayValues[126] = d126
					ps151.OverlayValues[131] = d131
					ps151.OverlayValues[133] = d133
					ps151.OverlayValues[134] = d134
					ps151.OverlayValues[135] = d135
					ps151.OverlayValues[136] = d136
					ps151.OverlayValues[137] = d137
					ps151.OverlayValues[138] = d138
					ps151.OverlayValues[139] = d139
					ps151.OverlayValues[143] = d143
					ps151.OverlayValues[144] = d144
					ps151.OverlayValues[145] = d145
					ps151.OverlayValues[148] = d148
					ps151.OverlayValues[149] = d149
					snap152 := d1
					snap153 := d2
					snap154 := d3
					snap155 := d4
					snap156 := d7
					snap157 := d8
					snap158 := d10
					snap159 := d11
					snap160 := d12
					snap161 := d13
					snap162 := d15
					snap163 := d17
					snap164 := d18
					snap165 := d19
					snap166 := d20
					snap167 := d21
					snap168 := d22
					snap169 := d23
					snap170 := d24
					snap171 := d25
					snap172 := d26
					snap173 := d28
					snap174 := d29
					snap175 := d30
					snap176 := d31
					snap177 := d32
					snap178 := d33
					snap179 := d34
					snap180 := d35
					snap181 := d74
					snap182 := d76
					snap183 := d77
					snap184 := d78
					snap185 := d79
					snap186 := d80
					snap187 := d81
					snap188 := d82
					snap189 := d83
					snap190 := d84
					snap191 := d86
					snap192 := d87
					snap193 := d91
					snap194 := d92
					snap195 := d93
					snap196 := d95
					snap197 := d97
					snap198 := d98
					snap199 := d99
					snap200 := d100
					snap201 := d101
					snap202 := d102
					snap203 := d103
					snap204 := d104
					snap205 := d105
					snap206 := d106
					snap207 := d107
					snap208 := d108
					snap209 := d109
					snap210 := d110
					snap211 := d111
					snap212 := d112
					snap213 := d113
					snap214 := d115
					snap215 := d116
					snap216 := d121
					snap217 := d123
					snap218 := d125
					snap219 := d126
					snap220 := d131
					snap221 := d133
					snap222 := d134
					snap223 := d135
					snap224 := d136
					snap225 := d137
					snap226 := d138
					snap227 := d139
					snap228 := d143
					snap229 := d144
					snap230 := d145
					snap231 := d148
					snap232 := d149
					alloc233 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps151)
					}
					ctx.RestoreAllocState(alloc233)
					d1 = snap152
					d2 = snap153
					d3 = snap154
					d4 = snap155
					d7 = snap156
					d8 = snap157
					d10 = snap158
					d11 = snap159
					d12 = snap160
					d13 = snap161
					d15 = snap162
					d17 = snap163
					d18 = snap164
					d19 = snap165
					d20 = snap166
					d21 = snap167
					d22 = snap168
					d23 = snap169
					d24 = snap170
					d25 = snap171
					d26 = snap172
					d28 = snap173
					d29 = snap174
					d30 = snap175
					d31 = snap176
					d32 = snap177
					d33 = snap178
					d34 = snap179
					d35 = snap180
					d74 = snap181
					d76 = snap182
					d77 = snap183
					d78 = snap184
					d79 = snap185
					d80 = snap186
					d81 = snap187
					d82 = snap188
					d83 = snap189
					d84 = snap190
					d86 = snap191
					d87 = snap192
					d91 = snap193
					d92 = snap194
					d93 = snap195
					d95 = snap196
					d97 = snap197
					d98 = snap198
					d99 = snap199
					d100 = snap200
					d101 = snap201
					d102 = snap202
					d103 = snap203
					d104 = snap204
					d105 = snap205
					d106 = snap206
					d107 = snap207
					d108 = snap208
					d109 = snap209
					d110 = snap210
					d111 = snap211
					d112 = snap212
					d113 = snap213
					d115 = snap214
					d116 = snap215
					d121 = snap216
					d123 = snap217
					d125 = snap218
					d126 = snap219
					d131 = snap220
					d133 = snap221
					d134 = snap222
					d135 = snap223
					d136 = snap224
					d137 = snap225
					d138 = snap226
					d139 = snap227
					d143 = snap228
					d144 = snap229
					d145 = snap230
					d148 = snap231
					d149 = snap232
					if !bbs[4].Rendered {
						return bbs[4].RenderPS(ps150)
					}
					return result
					ctx.FreeDesc(&d144)
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
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
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					ctx.ReclaimUntrackedRegs()
					blockPinnedRegs234 := make([]Reg, 0, 3)
					seenBlockPinnedRegs235 := make(map[Reg]bool)
					_ = seenBlockPinnedRegs235
					for _, r := range []Reg{d76.Reg, d76.Reg2, d76.Reg3} {
						live := d76.Loc == LocRegTriple && (r == d76.Reg || r == d76.Reg2 || r == d76.Reg3)
						if live && !seenBlockPinnedRegs235[r] {
							ctx.ProtectReg(r)
							seenBlockPinnedRegs235[r] = true
							blockPinnedRegs234 = append(blockPinnedRegs234, r)
						}
					}
					unpinBlockRegs236 := func() {
						for _, r := range blockPinnedRegs234 {
							ctx.UnprotectReg(r)
						}
					}
					defer unpinBlockRegs236()
					d237 = d8
					_ = d237
					d238 = d4
					_ = d238
					ctx.EnsureDesc(&d2)
					d240 = ctx.EmitSliceElementAddress(&d76, &d2, 16)
					ctx.EnsureDesc(&d240)
					r32 := ctx.AllocRegExcept(d240.Reg)
					ctx.EmitMovRegMem(r32, d240.Reg, 8)
					ctx.EmitMovRegMem(d240.Reg, d240.Reg, 0)
					d239 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d240.Reg, Reg2: r32}
					ctx.BindReg(d240.Reg, &d239)
					ctx.BindReg(r32, &d239)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d241 JITValueDesc
					if d2.Loc == LocImm {
						d241 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d241 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d241)
					}
					if d241.Loc == LocReg && d2.Loc == LocReg && d241.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d241)
					d243 = ctx.EmitSliceElementAddress(&d76, &d241, 16)
					ctx.EnsureDesc(&d243)
					r33 := ctx.AllocRegExcept(d243.Reg)
					ctx.EmitMovRegMem(r33, d243.Reg, 8)
					ctx.EmitMovRegMem(d243.Reg, d243.Reg, 0)
					d242 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d243.Reg, Reg2: r33}
					ctx.BindReg(d243.Reg, &d242)
					ctx.BindReg(r33, &d242)
					ctx.FreeDesc(&d241)
					stackArray244 := ctx.AllocStack(int32(32))
					_ = stackArray244
					ctx.EnsureDesc(&d239)
					ctx.EnsureDesc(&d239)
					ctx.EmitStoreScmerToStack(d239, int32(stackArray244)+int32(0))
					ctx.FreeDesc(&d239)
					ctx.EnsureDesc(&d242)
					ctx.EnsureDesc(&d242)
					ctx.EmitStoreScmerToStack(d242, int32(stackArray244)+int32(16))
					ctx.FreeDesc(&d242)
					d245 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					callbackArgs247 := make([]JITValueDesc, 2)
					callbackArgs247[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray244) + 0}
					callbackArgs247[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray244) + 16}
					var d246 JITValueDesc
					callbackResultOff248 := ctx.AllocStack(16)
					ctx.FreeDesc(&d245)
					if d238.Loc == LocLambdaTemplate && d238.Lambda != nil {
						stableCallbackArgs249 := ctx.StabilizeCallbackArgs(callbackArgs247)
						ctx.ReclaimUntrackedRegs()
						outerRegs250 := ctx.PreserveOuterRegs()
						d246 = JITEmitProcInlineWithOuter(ctx, &d238.Lambda.Proc, d238.Lambda.Outer, stableCallbackArgs249, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff248), ID: 0})
						ctx.RestoreOuterRegs(outerRegs250)
						ctx.ReclaimUntrackedRegs()
					} else {
						d251, knownBuiltin252 := jitEmitKnownDeclaration(ctx, d238, callbackArgs247, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff248), ID: 0})
						if knownBuiltin252 {
							d246 = d251
						} else {
							d253 := jitCopyScmerToPair(ctx, d238)
							callbackCallArgs := make([]JITValueDesc, 0, 3)
							callbackCallArgs = append(callbackCallArgs, d253)
							callbackCallArgs = append(callbackCallArgs, callbackArgs247...)
							d246 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback2), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d246, int32(callbackResultOff248))
							ctx.FreeDesc(&d246)
							d246 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff248), ID: 0}
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d254 JITValueDesc
					if d2.Loc == LocImm {
						d254 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d254 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d254)
					}
					if d254.Loc == LocReg && d2.Loc == LocReg && d254.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d254)
					d256 = ctx.EmitSliceElementAddress(&d76, &d254, 16)
					ctx.EnsureDesc(&d256)
					r34 := ctx.AllocRegExcept(d256.Reg)
					ctx.EmitMovRegMem(r34, d256.Reg, 8)
					ctx.EmitMovRegMem(d256.Reg, d256.Reg, 0)
					d255 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d256.Reg, Reg2: r34}
					ctx.BindReg(d256.Reg, &d255)
					ctx.BindReg(r34, &d255)
					ctx.FreeDesc(&d254)
					stackArray257 := ctx.AllocStack(int32(48))
					_ = stackArray257
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EmitStoreScmerToStack(d1, int32(stackArray257)+int32(0))
					ctx.EnsureDesc(&d246)
					ctx.EnsureDesc(&d246)
					ctx.EmitStoreScmerToStack(d246, int32(stackArray257)+int32(16))
					ctx.FreeDesc(&d246)
					ctx.EnsureDesc(&d255)
					ctx.EnsureDesc(&d255)
					ctx.EmitStoreScmerToStack(d255, int32(stackArray257)+int32(32))
					ctx.FreeDesc(&d255)
					d258 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					callbackArgs260 := make([]JITValueDesc, 3)
					callbackArgs260[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray257) + 0}
					callbackArgs260[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray257) + 16}
					callbackArgs260[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray257) + 32}
					var d259 JITValueDesc
					ctx.FreeDesc(&d258)
					if d237.Loc == LocLambdaTemplate && d237.Lambda != nil {
						stableCallbackArgs261 := ctx.StabilizeCallbackArgs(callbackArgs260)
						ctx.ReclaimUntrackedRegs()
						outerRegs262 := ctx.PreserveOuterRegs()
						d259 = JITEmitProcInlineWithOuter(ctx, &d237.Lambda.Proc, d237.Lambda.Outer, stableCallbackArgs261, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0})
						ctx.RestoreOuterRegs(outerRegs262)
						ctx.ReclaimUntrackedRegs()
					} else {
						d263, knownBuiltin264 := jitEmitKnownDeclaration(ctx, d237, callbackArgs260, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0})
						if knownBuiltin264 {
							d259 = d263
						} else {
							d265 := jitCopyScmerToPair(ctx, d237)
							callbackCallArgs := make([]JITValueDesc, 0, 4)
							callbackCallArgs = append(callbackCallArgs, d265)
							callbackCallArgs = append(callbackCallArgs, callbackArgs260...)
							d259 = ctx.EmitGoCallScalarInto(GoFuncAddr(jitInvokeCallback3), callbackCallArgs, JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: RegRAX, Reg2: RegRBX, ID: 0})
							ctx.EmitStoreScmerToStack(d259, int32(bbs[3].PhiBase)+int32(0))
							ctx.FreeDesc(&d259)
							d259 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0}
						}
					}
					ctx.StabilizeDescForControlFlow(&d259)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d266 JITValueDesc
					if d2.Loc == LocImm {
						d266 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(2))
						d266 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d266)
					}
					if d266.Loc == LocReg && d2.Loc == LocReg && d266.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d266)
					ctx.EmitStoreToStack(d266, int32(bbs[3].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d266)
					ctx.FreeDesc(&d2)
					ps267 := PhiState{General: ps.General}
					ps267.OverlayValues = make([]JITValueDesc, 267)
					ps267.OverlayValues[1] = d1
					ps267.OverlayValues[2] = d2
					ps267.OverlayValues[3] = d3
					ps267.OverlayValues[4] = d4
					ps267.OverlayValues[7] = d7
					ps267.OverlayValues[8] = d8
					ps267.OverlayValues[10] = d10
					ps267.OverlayValues[11] = d11
					ps267.OverlayValues[12] = d12
					ps267.OverlayValues[13] = d13
					ps267.OverlayValues[15] = d15
					ps267.OverlayValues[17] = d17
					ps267.OverlayValues[18] = d18
					ps267.OverlayValues[19] = d19
					ps267.OverlayValues[20] = d20
					ps267.OverlayValues[21] = d21
					ps267.OverlayValues[22] = d22
					ps267.OverlayValues[23] = d23
					ps267.OverlayValues[24] = d24
					ps267.OverlayValues[25] = d25
					ps267.OverlayValues[26] = d26
					ps267.OverlayValues[28] = d28
					ps267.OverlayValues[29] = d29
					ps267.OverlayValues[30] = d30
					ps267.OverlayValues[31] = d31
					ps267.OverlayValues[32] = d32
					ps267.OverlayValues[33] = d33
					ps267.OverlayValues[34] = d34
					ps267.OverlayValues[35] = d35
					ps267.OverlayValues[74] = d74
					ps267.OverlayValues[76] = d76
					ps267.OverlayValues[77] = d77
					ps267.OverlayValues[78] = d78
					ps267.OverlayValues[79] = d79
					ps267.OverlayValues[80] = d80
					ps267.OverlayValues[81] = d81
					ps267.OverlayValues[82] = d82
					ps267.OverlayValues[83] = d83
					ps267.OverlayValues[84] = d84
					ps267.OverlayValues[86] = d86
					ps267.OverlayValues[87] = d87
					ps267.OverlayValues[91] = d91
					ps267.OverlayValues[92] = d92
					ps267.OverlayValues[93] = d93
					ps267.OverlayValues[95] = d95
					ps267.OverlayValues[97] = d97
					ps267.OverlayValues[98] = d98
					ps267.OverlayValues[99] = d99
					ps267.OverlayValues[100] = d100
					ps267.OverlayValues[101] = d101
					ps267.OverlayValues[102] = d102
					ps267.OverlayValues[103] = d103
					ps267.OverlayValues[104] = d104
					ps267.OverlayValues[105] = d105
					ps267.OverlayValues[106] = d106
					ps267.OverlayValues[107] = d107
					ps267.OverlayValues[108] = d108
					ps267.OverlayValues[109] = d109
					ps267.OverlayValues[110] = d110
					ps267.OverlayValues[111] = d111
					ps267.OverlayValues[112] = d112
					ps267.OverlayValues[113] = d113
					ps267.OverlayValues[115] = d115
					ps267.OverlayValues[116] = d116
					ps267.OverlayValues[121] = d121
					ps267.OverlayValues[123] = d123
					ps267.OverlayValues[125] = d125
					ps267.OverlayValues[126] = d126
					ps267.OverlayValues[131] = d131
					ps267.OverlayValues[133] = d133
					ps267.OverlayValues[134] = d134
					ps267.OverlayValues[135] = d135
					ps267.OverlayValues[136] = d136
					ps267.OverlayValues[137] = d137
					ps267.OverlayValues[138] = d138
					ps267.OverlayValues[139] = d139
					ps267.OverlayValues[143] = d143
					ps267.OverlayValues[144] = d144
					ps267.OverlayValues[145] = d145
					ps267.OverlayValues[148] = d148
					ps267.OverlayValues[149] = d149
					ps267.OverlayValues[237] = d237
					ps267.OverlayValues[238] = d238
					ps267.OverlayValues[239] = d239
					ps267.OverlayValues[240] = d240
					ps267.OverlayValues[241] = d241
					ps267.OverlayValues[242] = d242
					ps267.OverlayValues[243] = d243
					ps267.OverlayValues[245] = d245
					ps267.OverlayValues[246] = d246
					ps267.OverlayValues[251] = d251
					ps267.OverlayValues[253] = d253
					ps267.OverlayValues[254] = d254
					ps267.OverlayValues[255] = d255
					ps267.OverlayValues[256] = d256
					ps267.OverlayValues[258] = d258
					ps267.OverlayValues[259] = d259
					ps267.OverlayValues[263] = d263
					ps267.OverlayValues[265] = d265
					ps267.OverlayValues[266] = d266
					ps267.PhiValues = make([]JITValueDesc, 2)
					if ps267.General && bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
						return result
					}
					return bbs[3].RenderPS(ps267)
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
					if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != LocNone {
						d12 = ps.OverlayValues[12]
					}
					if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
						d13 = ps.OverlayValues[13]
					}
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
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
					if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
						d23 = ps.OverlayValues[23]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
					}
					if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != LocNone {
						d79 = ps.OverlayValues[79]
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
					if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != LocNone {
						d116 = ps.OverlayValues[116]
					}
					if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
						d121 = ps.OverlayValues[121]
					}
					if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != LocNone {
						d123 = ps.OverlayValues[123]
					}
					if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
						d125 = ps.OverlayValues[125]
					}
					if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != LocNone {
						d126 = ps.OverlayValues[126]
					}
					if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != LocNone {
						d131 = ps.OverlayValues[131]
					}
					if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != LocNone {
						d133 = ps.OverlayValues[133]
					}
					if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != LocNone {
						d134 = ps.OverlayValues[134]
					}
					if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != LocNone {
						d135 = ps.OverlayValues[135]
					}
					if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != LocNone {
						d136 = ps.OverlayValues[136]
					}
					if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != LocNone {
						d137 = ps.OverlayValues[137]
					}
					if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != LocNone {
						d138 = ps.OverlayValues[138]
					}
					if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != LocNone {
						d139 = ps.OverlayValues[139]
					}
					if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != LocNone {
						d143 = ps.OverlayValues[143]
					}
					if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != LocNone {
						d144 = ps.OverlayValues[144]
					}
					if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
						d145 = ps.OverlayValues[145]
					}
					if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
						d148 = ps.OverlayValues[148]
					}
					if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
						d149 = ps.OverlayValues[149]
					}
					if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != LocNone {
						d237 = ps.OverlayValues[237]
					}
					if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != LocNone {
						d238 = ps.OverlayValues[238]
					}
					if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != LocNone {
						d239 = ps.OverlayValues[239]
					}
					if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != LocNone {
						d240 = ps.OverlayValues[240]
					}
					if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != LocNone {
						d241 = ps.OverlayValues[241]
					}
					if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != LocNone {
						d242 = ps.OverlayValues[242]
					}
					if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != LocNone {
						d243 = ps.OverlayValues[243]
					}
					if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != LocNone {
						d245 = ps.OverlayValues[245]
					}
					if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != LocNone {
						d246 = ps.OverlayValues[246]
					}
					if len(ps.OverlayValues) > 251 && ps.OverlayValues[251].Loc != LocNone {
						d251 = ps.OverlayValues[251]
					}
					if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != LocNone {
						d253 = ps.OverlayValues[253]
					}
					if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != LocNone {
						d254 = ps.OverlayValues[254]
					}
					if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != LocNone {
						d255 = ps.OverlayValues[255]
					}
					if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != LocNone {
						d256 = ps.OverlayValues[256]
					}
					if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
						d258 = ps.OverlayValues[258]
					}
					if len(ps.OverlayValues) > 259 && ps.OverlayValues[259].Loc != LocNone {
						d259 = ps.OverlayValues[259]
					}
					if len(ps.OverlayValues) > 263 && ps.OverlayValues[263].Loc != LocNone {
						d263 = ps.OverlayValues[263]
					}
					if len(ps.OverlayValues) > 265 && ps.OverlayValues[265].Loc != LocNone {
						d265 = ps.OverlayValues[265]
					}
					if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != LocNone {
						d266 = ps.OverlayValues[266]
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
				argPinned268 := make([]Reg, 0, len(args)*3)
				seenArgRegs := make(map[Reg]bool)
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned268 = append(argPinned268, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seenArgRegs[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seenArgRegs[ai.Reg] = true
							argPinned268 = append(argPinned268, ai.Reg)
						}
						if !seenArgRegs[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seenArgRegs[ai.Reg2] = true
							argPinned268 = append(argPinned268, ai.Reg2)
						}
					} else if ai.Loc == LocRegTriple {
						for _, r := range [...]Reg{ai.Reg, ai.Reg2, ai.Reg3} {
							if !seenArgRegs[r] {
								ctx.ProtectReg(r)
								seenArgRegs[r] = true
								argPinned268 = append(argPinned268, r)
							}
						}
					}
				}
				defer func() {
					for _, r := range argPinned268 {
						ctx.UnprotectReg(r)
					}
				}()
				ps269 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps269)
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
