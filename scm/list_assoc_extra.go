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

func treeCollectTaggedNthUnique(root, tag Scmer, position int) Scmer {
	return collectTaggedNthUnique(root, tag, position, true)
}

func collectTaggedNthUnique(root, tag Scmer, position int, traverseHeads bool) Scmer {
	if position < 0 {
		panic("tagged nth collector expects a non-negative position")
	}

	// Compiler trees are normally shallow and aliases are few. Keep traversal
	// state on the stack for those common cases; append retains correct behavior
	// for unusually deep trees without imposing a fixed planner limit.
	var pendingStorage [64]Scmer
	pending := pendingStorage[:1]
	pending[0] = root
	var uniqueStorage [8]Scmer
	unique := uniqueStorage[:0]

	for len(pending) != 0 {
		last := len(pending) - 1
		current := pending[last].WithoutSourceInfo()
		pending = pending[:last]
		if !current.IsSlice() {
			continue
		}
		items := current.Slice()
		if len(items) != 0 && Equal(items[0], tag) {
			if len(items) > position {
				candidate := items[position].WithoutSourceInfo()
				seen := false
				for _, existing := range unique {
					if Equal(existing, candidate) {
						seen = true
						break
					}
				}
				if !seen {
					unique = append(unique, candidate)
				}
			}
			// A tagged node is one logical leaf for this operation. Its payload is
			// data, not another expression to inspect for the same tag.
			continue
		}
		firstChild := 0
		if !traverseHeads {
			firstChild = 1
		}
		for index := len(items) - 1; index >= firstChild; index-- {
			pending = append(pending, items[index])
		}
	}

	result := make([]Scmer, len(unique))
	copy(result, unique)
	return NewSlice(result)
}

func exprTaggedNthMatchesAny(root, tag Scmer, position int, nilReplacement Scmer, candidates []Scmer) bool {
	if position < 0 {
		panic("tagged nth matcher expects a non-negative position")
	}

	var pendingStorage [64]Scmer
	pending := pendingStorage[:1]
	pending[0] = root
	for len(pending) != 0 {
		last := len(pending) - 1
		current := pending[last].WithoutSourceInfo()
		pending = pending[:last]
		if !current.IsSlice() {
			continue
		}
		items := current.Slice()
		if len(items) != 0 && Equal(items[0], tag) {
			if len(items) > position {
				candidate := items[position].WithoutSourceInfo()
				if candidate.IsNil() {
					candidate = nilReplacement
				}
				for _, expected := range candidates {
					if Equal(candidate, expected) {
						return true
					}
				}
			}
			continue
		}
		// Expression heads are operators, not operands in the caller's scope.
		for index := len(items) - 1; index >= 1; index-- {
			pending = append(pending, items[index])
		}
	}
	return false
}

func groupAssocCapacity(inputLength int) int {
	const initialGroups = 32
	if inputLength < initialGroups {
		return inputLength
	}
	return initialGroups
}

func init_list_assoc_extra() {
	Declare(&Globalenv, &Declaration{
		Name:          "optimizer_tree_collect_tagged_nth_unique",
		OptimizerOnly: true,
		Fn: func(a ...Scmer) Scmer {
			return treeCollectTaggedNthUnique(a[0], a[1], int(ToInt(a[2])))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "collects the unique zero-based nth values of tagged nodes in a nested list tree",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "tree", NoEscape: true},
				{Kind: "any", Label: "tag", NoEscape: true},
				{Kind: "number", Label: "position"},
			},
			Return:        FreshAlloc,
			Const:         true,
			JITInlineCost: 113,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["optimizer_tree_collect_tagged_nth_unique"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				d2 := args[2]
				d2.ID = 0
				ctx.EnsureDesc(&d2)
				d3 := d2
				_ = d3
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d4 JITValueDesc
				if d3.Loc == LocImm {
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int())}
				} else if d3.Type == tagInt && d3.Loc == LocRegPair {
					ctx.FreeReg(d3.Reg)
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2}
					ctx.BindReg(d3.Reg2, &d4)
					ctx.BindReg(d3.Reg2, &d4)
				} else if d3.Type == tagInt && d3.Loc == LocReg {
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg}
					ctx.BindReg(d3.Reg, &d4)
					ctx.BindReg(d3.Reg, &d4)
				} else {
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d3}, 1)
					d4.Type = tagInt
					ctx.BindReg(d4.Reg, &d4)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.FreeDesc(&d2)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d4)
				d6 := d0
				_ = d6
				d7 := d1
				_ = d7
				d8 := d4
				_ = d8
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl1 := ctx.ReserveLabel()
				_ = lbl1
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d7)
				ctx.EnsureDesc(&d8)
				d9 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				d10 := d6
				_ = d10
				ctx.StabilizeDescForControlFlow(&d10)
				d11 := d7
				_ = d11
				ctx.StabilizeDescForControlFlow(&d11)
				d12 := d8
				_ = d12
				ctx.StabilizeDescForControlFlow(&d12)
				d13 := d9
				_ = d13
				ctx.StabilizeDescForControlFlow(&d13)
				phiBase14 := ctx.AllocStack(int32(136))
				d15 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				ctx.PreparePointerStackTarget(int32(phiBase14)+int32(0), 3)
				_ = d15
				d16 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				ctx.PreparePointerStackTarget(int32(phiBase14)+int32(24), 3)
				_ = d16
				d17 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				_ = d17
				d18 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				_ = d18
				d19 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				_ = d19
				d20 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				ctx.PreparePointerStackTarget(int32(phiBase14)+int32(96), 3)
				_ = d20
				d21 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				_ = d21
				lbl2 := ctx.ReserveLabel()
				bbpos_3_0 := int32(-1)
				_ = bbpos_3_0
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_3_1 := int32(-1)
				_ = bbpos_3_1
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_3_2 := int32(-1)
				_ = bbpos_3_2
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_3_3 := int32(-1)
				_ = bbpos_3_3
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_3_4 := int32(-1)
				_ = bbpos_3_4
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_3_5 := int32(-1)
				_ = bbpos_3_5
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_3_6 := int32(-1)
				_ = bbpos_3_6
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_3_7 := int32(-1)
				_ = bbpos_3_7
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_3_8 := int32(-1)
				_ = bbpos_3_8
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_3_9 := int32(-1)
				_ = bbpos_3_9
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_3_10 := int32(-1)
				_ = bbpos_3_10
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_3_11 := int32(-1)
				_ = bbpos_3_11
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_3_12 := int32(-1)
				_ = bbpos_3_12
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_3_13 := int32(-1)
				_ = bbpos_3_13
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_3_14 := int32(-1)
				_ = bbpos_3_14
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbpos_3_15 := int32(-1)
				_ = bbpos_3_15
				lbl18 := ctx.ReserveLabel()
				_ = lbl18
				bbpos_3_16 := int32(-1)
				_ = bbpos_3_16
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbpos_3_17 := int32(-1)
				_ = bbpos_3_17
				lbl20 := ctx.ReserveLabel()
				_ = lbl20
				bbpos_3_18 := int32(-1)
				_ = bbpos_3_18
				lbl21 := ctx.ReserveLabel()
				_ = lbl21
				bbpos_3_19 := int32(-1)
				_ = bbpos_3_19
				lbl22 := ctx.ReserveLabel()
				_ = lbl22
				bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d22 JITValueDesc
				if d12.Loc == LocImm {
					d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < 0)}
				} else {
					r0 := ctx.AllocRegExcept(d12.Reg)
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
					ctx.BindReg(r0, &d22)
				}
				ctx.ReclaimUntrackedRegs()
				d23 := d22
				ctx.EnsureDesc(&d23)
				if d23.Loc != LocImm && d23.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl23 := ctx.ReserveLabel()
				lbl24 := ctx.ReserveLabel()
				if d23.Loc == LocImm {
					if d23.Imm.Bool() {
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl4)
					} else {
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitJump(d23.Condition, lbl23)
					ctx.EmitJmp(lbl24)
					ctx.FreeDesc(&d22)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl5)
				}
				bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				stackArray24 := ctx.AllocStack(int32(1024))
				_ = stackArray24
				ctx.ReclaimUntrackedRegs()
				d25 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(64), SliceSizeKnown: true}
				_ = d25
				ctx.StabilizeDescForControlFlow(&d25)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d10)
				ctx.EmitStoreScmerToStack(d10, int32(stackArray24)+int32(0))
				ctx.ReclaimUntrackedRegs()
				stackArray26 := ctx.AllocStack(int32(128))
				_ = stackArray26
				ctx.ReclaimUntrackedRegs()
				d27 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(8), SliceSizeKnown: true}
				_ = d27
				ctx.StabilizeDescForControlFlow(&d27)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d25)
				if d25.Loc == LocReg || d25.Loc == LocFPReg {
					ctx.ProtectReg(d25.Reg)
				} else if d25.Loc == LocRegPair {
					ctx.ProtectReg(d25.Reg)
					ctx.ProtectReg(d25.Reg2)
				}
				ctx.SyncDesc(&d27)
				if d27.Loc == LocReg || d27.Loc == LocFPReg {
					ctx.ProtectReg(d27.Reg)
				} else if d27.Loc == LocRegPair {
					ctx.ProtectReg(d27.Reg)
					ctx.ProtectReg(d27.Reg2)
				}
				d28 := d25
				if d28.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d28)
				if d28.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d28, int32(phiBase14)+int32(0), 3)
				} else {
					if d28.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d28.Reg, RegRSP, int32(phiBase14)+int32(0))
					ctx.EmitStoreRegMem(d28.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
					ctx.EmitStoreRegMem(d28.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
				}
				d29 := d27
				if d29.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d29)
				if d29.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d29, int32(phiBase14)+int32(24), 3)
				} else {
					if d29.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d29.Reg, RegRSP, int32(phiBase14)+int32(24))
					ctx.EmitStoreRegMem(d29.Reg2, RegRSP, int32(phiBase14)+int32(24)+8)
					ctx.EmitStoreRegMem(d29.Reg3, RegRSP, int32(phiBase14)+int32(24)+16)
				}
				if d25.Loc == LocReg || d25.Loc == LocFPReg {
					ctx.UnprotectReg(d25.Reg)
				} else if d25.Loc == LocRegPair {
					ctx.UnprotectReg(d25.Reg)
					ctx.UnprotectReg(d25.Reg2)
				}
				if d27.Loc == LocReg || d27.Loc == LocFPReg {
					ctx.UnprotectReg(d27.Reg)
				} else if d27.Loc == LocRegPair {
					ctx.UnprotectReg(d27.Reg)
					ctx.UnprotectReg(d27.Reg2)
				}
				bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d30 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d30)
				ctx.ReclaimUntrackedRegs()
				d31 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				ctx.StabilizeDescForControlFlow(&d31)
				ctx.ReclaimUntrackedRegs()
				var d32 JITValueDesc
				if d30.SliceSizeKnown {
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.KnownSliceLen))}
				} else if d30.Loc == LocImm {
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.StackOff))}
				} else if d30.Loc == LocStackTriple {
					d32 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d30.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocRegPair || d30.Loc == LocRegTriple {
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg2, ID: 0}
					} else if d30.Loc == LocReg {
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d32)
				var d33 JITValueDesc
				if d32.Loc == LocImm {
					d33 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d32.Imm.Int() != 0)}
				} else {
					r1 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d32.Reg, 0)
					d33 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondNotEqual}
					ctx.BindReg(r1, &d33)
				}
				ctx.FreeDesc(&d32)
				ctx.ReclaimUntrackedRegs()
				d34 := d33
				ctx.EnsureDesc(&d34)
				if d34.Loc != LocImm && d34.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl25 := ctx.ReserveLabel()
				lbl26 := ctx.ReserveLabel()
				if d34.Loc == LocImm {
					if d34.Imm.Bool() {
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl6)
					} else {
						ctx.MarkLabel(lbl26)
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d34.Condition, lbl25)
					ctx.EmitJmp(lbl26)
					ctx.FreeDesc(&d33)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl7)
				}
				bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d35 JITValueDesc
				if d31.SliceSizeKnown {
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.KnownSliceLen))}
				} else if d31.Loc == LocImm {
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.StackOff))}
				} else if d31.Loc == LocStackTriple {
					d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d31.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocRegTriple {
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2, ID: 0}
					} else if d31.Loc == LocReg {
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d35)
				ctx.EnsureDesc(&d35)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d35)
				ctx.EnsureDesc(&d35)
				callResults36 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d35, d35}, []uint8{3}, []uint8{1})
				d37 := callResults36[0]
				d37.Type = tagSlice
				ctx.FreeDesc(&d35)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d37)
				ctx.EnsureDesc(&d31)
				callResults38 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d37, d31}, []uint8{1}, []uint8{0})
				d39 := callResults38[0]
				d39.Type = tagInt
				ctx.ReclaimUntrackedRegs()
				d40 := ctx.EmitNewSliceFromGoSlice(&d37)
				ctx.ReclaimUntrackedRegs()
				r2 := ctx.AllocReg()
				r3 := ctx.AllocRegExcept(r2)
				d41 := JITValueDesc{Loc: LocRegPair, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d41)
				ctx.BindReg(r3, &d41)
				ctx.EmitMovPairToResult(&d40, &d41)
				ctx.EmitJmp(lbl2)
				bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d42 JITValueDesc
				if d30.SliceSizeKnown {
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.KnownSliceLen))}
				} else if d30.Loc == LocImm {
					d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d30.StackOff))}
				} else if d30.Loc == LocStackTriple {
					d42 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d30.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d30)
					if d30.Loc == LocRegPair || d30.Loc == LocRegTriple {
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg2, ID: 0}
					} else if d30.Loc == LocReg {
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d42)
				ctx.EnsureDesc(&d42)
				var d43 JITValueDesc
				if d42.Loc == LocImm {
					d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d42.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d42.Reg)
					ctx.EmitMovRegReg(scratch, d42.Reg)
					ctx.EmitSubRegImm32(scratch, int32(1))
					d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d43)
				}
				if d43.Loc == LocReg && d42.Loc == LocReg && d43.Reg == d42.Reg {
					ctx.TransferReg(d42.Reg)
					d42.Loc = LocNone
				}
				ctx.FreeDesc(&d42)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d43)
				ctx.ReclaimUntrackedRegs()
				d45 := ctx.EmitSliceElementAddress(&d30, &d43, 16)
				ctx.EnsureDesc(&d45)
				r4 := ctx.AllocRegExcept(d45.Reg)
				ctx.EmitMovRegMem(r4, d45.Reg, 8)
				ctx.EmitMovRegMem(d45.Reg, d45.Reg, 0)
				d44 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d45.Reg, Reg2: r4}
				ctx.BindReg(d45.Reg, &d44)
				ctx.BindReg(r4, &d44)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d44)
				phiBase46 := ctx.AllocStack(int32(16))
				d47 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase46) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase46) + int32(0))
				_ = d47
				lbl27 := ctx.ReserveLabel()
				bbpos_4_0 := int32(-1)
				_ = bbpos_4_0
				lbl28 := ctx.ReserveLabel()
				_ = lbl28
				bbpos_4_1 := int32(-1)
				_ = bbpos_4_1
				lbl29 := ctx.ReserveLabel()
				_ = lbl29
				bbpos_4_2 := int32(-1)
				_ = bbpos_4_2
				lbl30 := ctx.ReserveLabel()
				_ = lbl30
				bbpos_4_3 := int32(-1)
				_ = bbpos_4_3
				lbl31 := ctx.ReserveLabel()
				_ = lbl31
				bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl28)
				ctx.ResolveFixups()
				d47 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase46) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d44)
				if d44.Loc == LocReg || d44.Loc == LocFPReg {
					ctx.ProtectReg(d44.Reg)
				} else if d44.Loc == LocRegPair {
					ctx.ProtectReg(d44.Reg)
					ctx.ProtectReg(d44.Reg2)
				}
				d48 := d44
				if d48.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d48)
				if d48.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d48, int32(phiBase46)+int32(0), 2)
				} else if d48.Loc == LocInputPair {
					ctx.EnsureDesc(&d48)
					ctx.EmitStoreScmerToStack(d48, int32(phiBase46)+int32(0))
				} else if d48.Loc == LocRegPair || d48.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d48, int32(phiBase46)+int32(0))
				} else {
					ctx.EnsureDesc(&d48)
					ctx.EmitStoreToStack(d48, int32(phiBase46)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase46)+int32(0))+8)
				}
				if d44.Loc == LocReg || d44.Loc == LocFPReg {
					ctx.UnprotectReg(d44.Reg)
				} else if d44.Loc == LocRegPair {
					ctx.UnprotectReg(d44.Reg)
					ctx.UnprotectReg(d44.Reg2)
				}
				bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl31)
				ctx.ResolveFixups()
				d47 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase46) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d49 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase46) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d49)
				ctx.ReclaimUntrackedRegs()
				d49 = JITPrepareScmerGoArg(ctx, d49)
				ctx.SyncDesc(&d49)
				d50 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d49}, 1)
				d50.NoHeapPointer = true
				ctx.EmitAndRegImm32(d50.Reg, 1)
				d50.Type = tagBool
				ctx.BindReg(d50.Reg, &d50)
				ctx.ReclaimUntrackedRegs()
				d51 := d50
				ctx.EnsureDesc(&d51)
				if d51.Loc != LocImm && d51.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl32 := ctx.ReserveLabel()
				lbl33 := ctx.ReserveLabel()
				if d51.Loc == LocImm {
					if d51.Imm.Bool() {
						ctx.MarkLabel(lbl32)
						ctx.EmitJmp(lbl29)
					} else {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl30)
					}
				} else {
					ctx.EmitCmpRegImm32(d51.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl32)
					ctx.EmitJmp(lbl33)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl29)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl30)
				}
				ctx.FreeDesc(&d50)
				bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl30)
				ctx.ResolveFixups()
				d49 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase46) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r5 := ctx.AllocReg()
				r6 := ctx.AllocRegExcept(r5)
				d52 := JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d52)
				ctx.BindReg(r6, &d52)
				ctx.EmitMovPairToResult(&d49, &d52)
				ctx.EmitJmp(lbl27)
				bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl29)
				ctx.ResolveFixups()
				d49 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase46) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d49 = JITPrepareScmerGoArg(ctx, d49)
				ctx.SyncDesc(&d49)
				d53 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d49}, 1)
				d53.NoHeapPointer = false
				ctx.BindReg(d53.Reg, &d53)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d54 JITValueDesc
				ctx.EnsureDesc(&d53)
				if d53.Loc == LocImm {
					fieldAddr := uintptr(d53.Imm.Int()) + 32
					r7 := ctx.AllocReg()
					r8 := ctx.AllocRegExcept(r7)
					ctx.EmitMovRegMem64(r7, fieldAddr)
					ctx.EmitMovRegMem64(r8, fieldAddr+8)
					d54 = JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
					ctx.BindReg(r7, &d54)
					ctx.BindReg(r8, &d54)
				} else {
					off := int32(32)
					baseReg := d53.Reg
					r9 := ctx.AllocRegExcept(baseReg)
					r10 := ctx.AllocRegExcept(baseReg, r9)
					ctx.EmitMovRegMem(r9, baseReg, off)
					ctx.EmitMovRegMem(r10, baseReg, off+8)
					d54 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
					ctx.BindReg(r9, &d54)
					ctx.BindReg(r10, &d54)
				}
				ctx.FreeDesc(&d53)
				ctx.StabilizeDescForControlFlow(&d54)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d54)
				if d54.Loc == LocReg || d54.Loc == LocFPReg {
					ctx.ProtectReg(d54.Reg)
				} else if d54.Loc == LocRegPair {
					ctx.ProtectReg(d54.Reg)
					ctx.ProtectReg(d54.Reg2)
				}
				d55 := d54
				if d55.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d55)
				if d55.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d55, int32(phiBase46)+int32(0), 2)
				} else if d55.Loc == LocInputPair {
					ctx.EnsureDesc(&d55)
					ctx.EmitStoreScmerToStack(d55, int32(phiBase46)+int32(0))
				} else if d55.Loc == LocRegPair || d55.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d55, int32(phiBase46)+int32(0))
				} else {
					ctx.EnsureDesc(&d55)
					ctx.EmitStoreToStack(d55, int32(phiBase46)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase46)+int32(0))+8)
				}
				if d54.Loc == LocReg || d54.Loc == LocFPReg {
					ctx.UnprotectReg(d54.Reg)
				} else if d54.Loc == LocRegPair {
					ctx.UnprotectReg(d54.Reg)
					ctx.UnprotectReg(d54.Reg2)
				}
				ctx.EmitJmp(lbl31)
				ctx.MarkLabel(lbl27)
				d56 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d56)
				ctx.BindReg(r6, &d56)
				ctx.BindReg(r5, &d56)
				ctx.BindReg(r6, &d56)
				ctx.StabilizeDescForControlFlow(&d56)
				ctx.FreeDesc(&d44)
				ctx.ReclaimUntrackedRegs()
				d57 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d43)
				ctx.EnsureDesc(&d30)
				ctx.EnsureDesc(&d57)
				ctx.EnsureDesc(&d43)
				var d59 JITValueDesc
				if d43.Loc == LocImm && d57.Loc == LocImm {
					d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d43.Imm.Int() - d57.Imm.Int())}
				} else {
					r11 := ctx.AllocReg()
					if d43.Loc == LocImm {
						ctx.EmitMovRegImm64(r11, uint64(d43.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r11, d43.Reg)
					}
					if d57.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d57.Imm.Int()))
						ctx.EmitSubInt64(r11, RegR11)
					} else {
						ctx.EmitSubInt64(r11, d57.Reg)
					}
					d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d59)
				}
				var d60 JITValueDesc
				r12 := ctx.EmitSliceDataAfterLow(&d30, &d57, 16)
				d60 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d60)
				ctx.BindReg(r12, &d60)
				var d61 JITValueDesc
				var r13 Reg
				var r14 Reg
				ctx.SyncDesc(&d60)
				ctx.EnsureDesc(&d60)
				if d60.Loc == LocImm {
					r13 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r13, uint64(d60.Imm.Int()))
				} else {
					r13 = d60.Reg
				}
				ctx.ProtectReg(r13)
				ctx.SyncDesc(&d59)
				ctx.EnsureDesc(&d59)
				if d59.Loc == LocImm {
					r14 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r14, uint64(d59.Imm.Int()))
				} else {
					r14 = d59.Reg
				}
				ctx.ProtectReg(r14)
				r15 := ctx.EmitSliceCapAfterLow(&d30, &d57, r13, r14)
				ctx.UnprotectReg(r14)
				ctx.UnprotectReg(r13)
				d61 = JITValueDesc{Loc: LocRegTriple, Reg: r13, Reg2: r14, Reg3: r15}
				ctx.BindReg(r13, &d61)
				ctx.BindReg(r14, &d61)
				ctx.BindReg(r15, &d61)
				ctx.BindReg(r13, &d61)
				ctx.BindReg(r14, &d61)
				ctx.BindReg(r15, &d61)
				ctx.StabilizeDescForControlFlow(&d61)
				ctx.FreeDesc(&d43)
				ctx.ReclaimUntrackedRegs()
				d63 := d56
				d63.ID = 0
				d62 := ctx.EmitTagEqualsBorrowed(&d63, tagSlice, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				d64 := d62
				ctx.EnsureDesc(&d64)
				if d64.Loc != LocImm && d64.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl34 := ctx.ReserveLabel()
				lbl35 := ctx.ReserveLabel()
				if d64.Loc == LocImm {
					if d64.Imm.Bool() {
						ctx.MarkLabel(lbl34)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl35)
						ctx.SyncDesc(&d61)
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.ProtectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.ProtectReg(d61.Reg)
							ctx.ProtectReg(d61.Reg2)
						}
						d65 := d61
						if d65.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d65)
						if d65.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d65, int32(phiBase14)+int32(0), 3)
						} else {
							if d65.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d65.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d65.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d65.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.UnprotectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.UnprotectReg(d61.Reg)
							ctx.UnprotectReg(d61.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitCmpRegImm32(d64.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl34)
					ctx.EmitJmp(lbl35)
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl35)
					ctx.SyncDesc(&d61)
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.ProtectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.ProtectReg(d61.Reg)
						ctx.ProtectReg(d61.Reg2)
					}
					d66 := d61
					if d66.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d66)
					if d66.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d66, int32(phiBase14)+int32(0), 3)
					} else {
						if d66.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d66.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d66.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d66.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.UnprotectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.UnprotectReg(d61.Reg)
						ctx.UnprotectReg(d61.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				ctx.FreeDesc(&d62)
				bbpos_3_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d67 := jitKnownSliceHeader(ctx, &d56)
				ctx.StabilizeDescForControlFlow(&d67)
				ctx.ReclaimUntrackedRegs()
				var d68 JITValueDesc
				if d67.SliceSizeKnown {
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d67.KnownSliceLen))}
				} else if d67.Loc == LocImm {
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d67.StackOff))}
				} else if d67.Loc == LocStackTriple {
					d68 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d67.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d67)
					if d67.Loc == LocRegPair || d67.Loc == LocRegTriple {
						d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d67.Reg2, ID: 0}
					} else if d67.Loc == LocReg {
						d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d67.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d68)
				var d69 JITValueDesc
				if d68.Loc == LocImm {
					d69 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d68.Imm.Int() != 0)}
				} else {
					r16 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d68.Reg, 0)
					d69 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondNotEqual}
					ctx.BindReg(r16, &d69)
				}
				ctx.FreeDesc(&d68)
				ctx.ReclaimUntrackedRegs()
				d70 := d69
				ctx.EnsureDesc(&d70)
				if d70.Loc != LocImm && d70.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl36 := ctx.ReserveLabel()
				lbl37 := ctx.ReserveLabel()
				if d70.Loc == LocImm {
					if d70.Imm.Bool() {
						ctx.MarkLabel(lbl36)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl37)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitJump(d70.Condition, lbl36)
					ctx.EmitJmp(lbl37)
					ctx.FreeDesc(&d69)
					ctx.MarkLabel(lbl36)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl11)
				}
				bbpos_3_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d71 := d13
				ctx.EnsureDesc(&d71)
				if d71.Loc != LocImm && d71.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl38 := ctx.ReserveLabel()
				lbl39 := ctx.ReserveLabel()
				if d71.Loc == LocImm {
					if d71.Imm.Bool() {
						ctx.MarkLabel(lbl38)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase14)+int32(80))
						ctx.EmitJmp(lbl20)
					} else {
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl19)
					}
				} else {
					ctx.EmitCmpRegImm32(d71.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl38)
					ctx.EmitJmp(lbl39)
					ctx.MarkLabel(lbl38)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase14)+int32(80))
					ctx.EmitJmp(lbl20)
					ctx.MarkLabel(lbl39)
					ctx.EmitJmp(lbl19)
				}
				ctx.FreeDesc(&d13)
				bbpos_3_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl20)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d72 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(80)}
				ctx.StabilizeDescForControlFlow(&d72)
				ctx.ReclaimUntrackedRegs()
				var d73 JITValueDesc
				if d67.SliceSizeKnown {
					d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d67.KnownSliceLen))}
				} else if d67.Loc == LocImm {
					d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d67.StackOff))}
				} else if d67.Loc == LocStackTriple {
					d73 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d67.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d67)
					if d67.Loc == LocRegPair || d67.Loc == LocRegTriple {
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d67.Reg2, ID: 0}
					} else if d67.Loc == LocReg {
						d73 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d67.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d73)
				ctx.EnsureDesc(&d73)
				var d74 JITValueDesc
				if d73.Loc == LocImm {
					d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d73.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d73.Reg)
					ctx.EmitMovRegReg(scratch, d73.Reg)
					ctx.EmitSubRegImm32(scratch, int32(1))
					d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d74)
				}
				if d74.Loc == LocReg && d73.Loc == LocReg && d74.Reg == d73.Reg {
					ctx.TransferReg(d73.Reg)
					d73.Loc = LocNone
				}
				ctx.EnsureDesc(&d74)
				ctx.EmitStoreToStack(d74, int32(phiBase14)+int32(120))
				ctx.StabilizeDescForControlFlow(&d74)
				ctx.FreeDesc(&d73)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d61)
				if d61.Loc == LocReg || d61.Loc == LocFPReg {
					ctx.ProtectReg(d61.Reg)
				} else if d61.Loc == LocRegPair {
					ctx.ProtectReg(d61.Reg)
					ctx.ProtectReg(d61.Reg2)
				}
				d75 := d61
				if d75.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d75)
				if d75.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d75, int32(phiBase14)+int32(96), 3)
				} else {
					if d75.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d75.Reg, RegRSP, int32(phiBase14)+int32(96))
					ctx.EmitStoreRegMem(d75.Reg2, RegRSP, int32(phiBase14)+int32(96)+8)
					ctx.EmitStoreRegMem(d75.Reg3, RegRSP, int32(phiBase14)+int32(96)+16)
				}
				if d61.Loc == LocReg || d61.Loc == LocFPReg {
					ctx.UnprotectReg(d61.Reg)
				} else if d61.Loc == LocRegPair {
					ctx.UnprotectReg(d61.Reg)
					ctx.UnprotectReg(d61.Reg2)
				}
				bbpos_3_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl21)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d21 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d76 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				ctx.StabilizeDescForControlFlow(&d76)
				ctx.ReclaimUntrackedRegs()
				d77 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(120)}
				ctx.StabilizeDescForControlFlow(&d77)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d77)
				ctx.EnsureDesc(&d72)
				ctx.EnsureDescsTogether(&d77, &d72)
				var d78 JITValueDesc
				if d77.Loc == LocImm && d72.Loc == LocImm {
					d78 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d77.Imm.Int() >= d72.Imm.Int())}
				} else if d72.Loc == LocImm {
					r17 := ctx.AllocRegExcept(d77.Reg)
					if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d77.Reg, int32(d72.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d72.Imm.Int()))
						ctx.EmitCmpInt64(d77.Reg, RegR11)
					}
					d78 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r17, &d78)
				} else if d77.Loc == LocImm {
					r18 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d77.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d72.Reg)
					d78 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r18, &d78)
				} else {
					r19 := ctx.AllocRegExcept(d77.Reg)
					ctx.EmitCmpInt64(d77.Reg, d72.Reg)
					d78 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r19, &d78)
				}
				ctx.ReclaimUntrackedRegs()
				d79 := d78
				ctx.EnsureDesc(&d79)
				if d79.Loc != LocImm && d79.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl40 := ctx.ReserveLabel()
				lbl41 := ctx.ReserveLabel()
				if d79.Loc == LocImm {
					if d79.Imm.Bool() {
						ctx.MarkLabel(lbl40)
						ctx.EmitJmp(lbl22)
					} else {
						ctx.MarkLabel(lbl41)
						ctx.SyncDesc(&d76)
						if d76.Loc == LocReg || d76.Loc == LocFPReg {
							ctx.ProtectReg(d76.Reg)
						} else if d76.Loc == LocRegPair {
							ctx.ProtectReg(d76.Reg)
							ctx.ProtectReg(d76.Reg2)
						}
						d80 := d76
						if d80.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d80)
						if d80.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d80, int32(phiBase14)+int32(0), 3)
						} else {
							if d80.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d80.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d80.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d80.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d76.Loc == LocReg || d76.Loc == LocFPReg {
							ctx.UnprotectReg(d76.Reg)
						} else if d76.Loc == LocRegPair {
							ctx.UnprotectReg(d76.Reg)
							ctx.UnprotectReg(d76.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitJump(d79.Condition, lbl40)
					ctx.EmitJmp(lbl41)
					ctx.FreeDesc(&d78)
					ctx.MarkLabel(lbl40)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl41)
					ctx.SyncDesc(&d76)
					if d76.Loc == LocReg || d76.Loc == LocFPReg {
						ctx.ProtectReg(d76.Reg)
					} else if d76.Loc == LocRegPair {
						ctx.ProtectReg(d76.Reg)
						ctx.ProtectReg(d76.Reg2)
					}
					d81 := d76
					if d81.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d81)
					if d81.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d81, int32(phiBase14)+int32(0), 3)
					} else {
						if d81.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d81.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d81.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d81.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d76.Loc == LocReg || d76.Loc == LocFPReg {
						ctx.UnprotectReg(d76.Reg)
					} else if d76.Loc == LocRegPair {
						ctx.UnprotectReg(d76.Reg)
						ctx.UnprotectReg(d76.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				bbpos_3_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d82 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.ReclaimUntrackedRegs()
				d84 := ctx.EmitSliceElementAddress(&d67, &d82, 16)
				ctx.EnsureDesc(&d84)
				r20 := ctx.AllocRegExcept(d84.Reg)
				ctx.EmitMovRegMem(r20, d84.Reg, 8)
				ctx.EmitMovRegMem(d84.Reg, d84.Reg, 0)
				d83 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d84.Reg, Reg2: r20}
				ctx.BindReg(d84.Reg, &d83)
				ctx.BindReg(r20, &d83)
				ctx.ReclaimUntrackedRegs()
				d83 = JITPrepareScmerGoArg(ctx, d83)
				d11 = JITPrepareScmerGoArg(ctx, d11)
				ctx.SyncDesc(&d83)
				ctx.SyncDesc(&d11)
				d85 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d83, d11}, 1)
				d85.NoHeapPointer = true
				ctx.EmitAndRegImm32(d85.Reg, 1)
				d85.Type = tagBool
				ctx.BindReg(d85.Reg, &d85)
				ctx.FreeDesc(&d83)
				ctx.ReclaimUntrackedRegs()
				d86 := d85
				ctx.EnsureDesc(&d86)
				if d86.Loc != LocImm && d86.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl42 := ctx.ReserveLabel()
				lbl43 := ctx.ReserveLabel()
				if d86.Loc == LocImm {
					if d86.Imm.Bool() {
						ctx.MarkLabel(lbl42)
						ctx.EmitJmp(lbl10)
					} else {
						ctx.MarkLabel(lbl43)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitCmpRegImm32(d86.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl42)
					ctx.EmitJmp(lbl43)
					ctx.MarkLabel(lbl42)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl43)
					ctx.EmitJmp(lbl11)
				}
				ctx.FreeDesc(&d85)
				bbpos_3_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(phiBase14)+int32(80))
				ctx.EmitJmp(lbl20)
				bbpos_3_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl22)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d77)
				ctx.ReclaimUntrackedRegs()
				d88 := ctx.EmitSliceElementAddress(&d67, &d77, 16)
				ctx.EnsureDesc(&d88)
				r21 := ctx.AllocRegExcept(d88.Reg)
				ctx.EmitMovRegMem(r21, d88.Reg, 8)
				ctx.EmitMovRegMem(d88.Reg, d88.Reg, 0)
				d87 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d88.Reg, Reg2: r21}
				ctx.BindReg(d88.Reg, &d87)
				ctx.BindReg(r21, &d87)
				ctx.ReclaimUntrackedRegs()
				stackArray89 := ctx.AllocStack(int32(16))
				_ = stackArray89
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d87)
				ctx.EmitStoreScmerToStack(d87, int32(stackArray89)+int32(0))
				ctx.FreeDesc(&d87)
				ctx.ReclaimUntrackedRegs()
				d90 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d90
				ctx.ReclaimUntrackedRegs()
				r22 := ctx.AllocReg()
				r23 := ctx.AllocRegExcept(r22)
				r24 := ctx.AllocRegExcept(r22, r23)
				d91 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r22, Reg2: r23, Reg3: r24}
				ctx.BindReg(r22, &d91)
				ctx.BindReg(r23, &d91)
				ctx.BindReg(r24, &d91)
				ctx.BindReg(r22, &d91)
				ctx.BindReg(r23, &d91)
				ctx.BindReg(r24, &d91)
				ctx.EmitLeaRegMem(d91.Reg, ctx.StackReg, int32(stackArray89))
				ctx.EmitMovRegImm64(d91.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d91.Reg3, uint64(1))
				callResults92 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d76, d91}, []uint8{3}, []uint8{1})
				d93 := callResults92[0]
				d94 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase14) + int32(96)}
				ctx.EmitCopyDescWords(&d94, &d93, 3)
				ctx.FreeDesc(&d93)
				d93 = d94
				ctx.StabilizeDescForControlFlow(&d93)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d77)
				ctx.EnsureDesc(&d77)
				var d95 JITValueDesc
				if d77.Loc == LocImm {
					d95 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d77.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d77.Reg)
					ctx.EmitMovRegReg(scratch, d77.Reg)
					ctx.EmitSubRegImm32(scratch, int32(1))
					d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d95)
				}
				if d95.Loc == LocReg && d77.Loc == LocReg && d95.Reg == d77.Reg {
					ctx.TransferReg(d77.Reg)
					d77.Loc = LocNone
				}
				ctx.EnsureDesc(&d95)
				ctx.EmitStoreToStack(d95, int32(phiBase14)+int32(120))
				ctx.StabilizeDescForControlFlow(&d95)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl21)
				bbpos_3_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d96 JITValueDesc
				if d67.SliceSizeKnown {
					d96 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d67.KnownSliceLen))}
				} else if d67.Loc == LocImm {
					d96 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d67.StackOff))}
				} else if d67.Loc == LocStackTriple {
					d96 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d67.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d67)
					if d67.Loc == LocRegPair || d67.Loc == LocRegTriple {
						d96 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d67.Reg2, ID: 0}
					} else if d67.Loc == LocReg {
						d96 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d67.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d96)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDescsTogether(&d96, &d12)
				var d97 JITValueDesc
				if d96.Loc == LocImm && d12.Loc == LocImm {
					d97 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d96.Imm.Int() > d12.Imm.Int())}
				} else if d12.Loc == LocImm {
					r25 := ctx.AllocReg()
					if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d96.Reg, int32(d12.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d12.Imm.Int()))
						ctx.EmitCmpInt64(d96.Reg, RegR11)
					}
					d97 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedGreater}
					ctx.BindReg(r25, &d97)
				} else if d96.Loc == LocImm {
					r26 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d96.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d12.Reg)
					d97 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedGreater}
					ctx.BindReg(r26, &d97)
				} else {
					r27 := ctx.AllocReg()
					ctx.EmitCmpInt64(d96.Reg, d12.Reg)
					d97 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedGreater}
					ctx.BindReg(r27, &d97)
				}
				ctx.FreeDesc(&d96)
				ctx.ReclaimUntrackedRegs()
				d98 := d97
				ctx.EnsureDesc(&d98)
				if d98.Loc != LocImm && d98.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl44 := ctx.ReserveLabel()
				lbl45 := ctx.ReserveLabel()
				if d98.Loc == LocImm {
					if d98.Imm.Bool() {
						ctx.MarkLabel(lbl44)
						ctx.EmitJmp(lbl13)
					} else {
						ctx.MarkLabel(lbl45)
						ctx.SyncDesc(&d61)
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.ProtectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.ProtectReg(d61.Reg)
							ctx.ProtectReg(d61.Reg2)
						}
						d99 := d61
						if d99.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d99)
						if d99.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d99, int32(phiBase14)+int32(0), 3)
						} else {
							if d99.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d99.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d99.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d99.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.UnprotectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.UnprotectReg(d61.Reg)
							ctx.UnprotectReg(d61.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitJump(d98.Condition, lbl44)
					ctx.EmitJmp(lbl45)
					ctx.FreeDesc(&d97)
					ctx.MarkLabel(lbl44)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl45)
					ctx.SyncDesc(&d61)
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.ProtectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.ProtectReg(d61.Reg)
						ctx.ProtectReg(d61.Reg2)
					}
					d100 := d61
					if d100.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d100)
					if d100.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d100, int32(phiBase14)+int32(0), 3)
					} else {
						if d100.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d100.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d100.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d100.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.UnprotectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.UnprotectReg(d61.Reg)
						ctx.UnprotectReg(d61.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				bbpos_3_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				ctx.ReclaimUntrackedRegs()
				d102 := ctx.EmitSliceElementAddress(&d67, &d12, 16)
				ctx.EnsureDesc(&d102)
				r28 := ctx.AllocRegExcept(d102.Reg)
				ctx.EmitMovRegMem(r28, d102.Reg, 8)
				ctx.EmitMovRegMem(d102.Reg, d102.Reg, 0)
				d101 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d102.Reg, Reg2: r28}
				ctx.BindReg(d102.Reg, &d101)
				ctx.BindReg(r28, &d101)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d101)
				phiBase103 := ctx.AllocStack(int32(16))
				d104 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase103) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase103) + int32(0))
				_ = d104
				lbl46 := ctx.ReserveLabel()
				bbpos_5_0 := int32(-1)
				_ = bbpos_5_0
				lbl47 := ctx.ReserveLabel()
				_ = lbl47
				bbpos_5_1 := int32(-1)
				_ = bbpos_5_1
				lbl48 := ctx.ReserveLabel()
				_ = lbl48
				bbpos_5_2 := int32(-1)
				_ = bbpos_5_2
				lbl49 := ctx.ReserveLabel()
				_ = lbl49
				bbpos_5_3 := int32(-1)
				_ = bbpos_5_3
				lbl50 := ctx.ReserveLabel()
				_ = lbl50
				bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl47)
				ctx.ResolveFixups()
				d104 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase103) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d101)
				if d101.Loc == LocReg || d101.Loc == LocFPReg {
					ctx.ProtectReg(d101.Reg)
				} else if d101.Loc == LocRegPair {
					ctx.ProtectReg(d101.Reg)
					ctx.ProtectReg(d101.Reg2)
				}
				d105 := d101
				if d105.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d105)
				if d105.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d105, int32(phiBase103)+int32(0), 2)
				} else if d105.Loc == LocInputPair {
					ctx.EnsureDesc(&d105)
					ctx.EmitStoreScmerToStack(d105, int32(phiBase103)+int32(0))
				} else if d105.Loc == LocRegPair || d105.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d105, int32(phiBase103)+int32(0))
				} else {
					ctx.EnsureDesc(&d105)
					ctx.EmitStoreToStack(d105, int32(phiBase103)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase103)+int32(0))+8)
				}
				if d101.Loc == LocReg || d101.Loc == LocFPReg {
					ctx.UnprotectReg(d101.Reg)
				} else if d101.Loc == LocRegPair {
					ctx.UnprotectReg(d101.Reg)
					ctx.UnprotectReg(d101.Reg2)
				}
				bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl50)
				ctx.ResolveFixups()
				d104 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase103) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d106 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase103) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d106)
				ctx.ReclaimUntrackedRegs()
				d106 = JITPrepareScmerGoArg(ctx, d106)
				ctx.SyncDesc(&d106)
				d107 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d106}, 1)
				d107.NoHeapPointer = true
				ctx.EmitAndRegImm32(d107.Reg, 1)
				d107.Type = tagBool
				ctx.BindReg(d107.Reg, &d107)
				ctx.ReclaimUntrackedRegs()
				d108 := d107
				ctx.EnsureDesc(&d108)
				if d108.Loc != LocImm && d108.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl51 := ctx.ReserveLabel()
				lbl52 := ctx.ReserveLabel()
				if d108.Loc == LocImm {
					if d108.Imm.Bool() {
						ctx.MarkLabel(lbl51)
						ctx.EmitJmp(lbl48)
					} else {
						ctx.MarkLabel(lbl52)
						ctx.EmitJmp(lbl49)
					}
				} else {
					ctx.EmitCmpRegImm32(d108.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl51)
					ctx.EmitJmp(lbl52)
					ctx.MarkLabel(lbl51)
					ctx.EmitJmp(lbl48)
					ctx.MarkLabel(lbl52)
					ctx.EmitJmp(lbl49)
				}
				ctx.FreeDesc(&d107)
				bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl49)
				ctx.ResolveFixups()
				d106 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase103) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r29 := ctx.AllocReg()
				r30 := ctx.AllocRegExcept(r29)
				d109 := JITValueDesc{Loc: LocRegPair, Reg: r29, Reg2: r30}
				ctx.BindReg(r29, &d109)
				ctx.BindReg(r30, &d109)
				ctx.EmitMovPairToResult(&d106, &d109)
				ctx.EmitJmp(lbl46)
				bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl48)
				ctx.ResolveFixups()
				d106 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase103) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d106 = JITPrepareScmerGoArg(ctx, d106)
				ctx.SyncDesc(&d106)
				d110 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d106}, 1)
				d110.NoHeapPointer = false
				ctx.BindReg(d110.Reg, &d110)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d111 JITValueDesc
				ctx.EnsureDesc(&d110)
				if d110.Loc == LocImm {
					fieldAddr := uintptr(d110.Imm.Int()) + 32
					r31 := ctx.AllocReg()
					r32 := ctx.AllocRegExcept(r31)
					ctx.EmitMovRegMem64(r31, fieldAddr)
					ctx.EmitMovRegMem64(r32, fieldAddr+8)
					d111 = JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
					ctx.BindReg(r31, &d111)
					ctx.BindReg(r32, &d111)
				} else {
					off := int32(32)
					baseReg := d110.Reg
					r33 := ctx.AllocRegExcept(baseReg)
					r34 := ctx.AllocRegExcept(baseReg, r33)
					ctx.EmitMovRegMem(r33, baseReg, off)
					ctx.EmitMovRegMem(r34, baseReg, off+8)
					d111 = JITValueDesc{Loc: LocRegPair, Reg: r33, Reg2: r34}
					ctx.BindReg(r33, &d111)
					ctx.BindReg(r34, &d111)
				}
				ctx.FreeDesc(&d110)
				ctx.StabilizeDescForControlFlow(&d111)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d111)
				if d111.Loc == LocReg || d111.Loc == LocFPReg {
					ctx.ProtectReg(d111.Reg)
				} else if d111.Loc == LocRegPair {
					ctx.ProtectReg(d111.Reg)
					ctx.ProtectReg(d111.Reg2)
				}
				d112 := d111
				if d112.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d112)
				if d112.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d112, int32(phiBase103)+int32(0), 2)
				} else if d112.Loc == LocInputPair {
					ctx.EnsureDesc(&d112)
					ctx.EmitStoreScmerToStack(d112, int32(phiBase103)+int32(0))
				} else if d112.Loc == LocRegPair || d112.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d112, int32(phiBase103)+int32(0))
				} else {
					ctx.EnsureDesc(&d112)
					ctx.EmitStoreToStack(d112, int32(phiBase103)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase103)+int32(0))+8)
				}
				if d111.Loc == LocReg || d111.Loc == LocFPReg {
					ctx.UnprotectReg(d111.Reg)
				} else if d111.Loc == LocRegPair {
					ctx.UnprotectReg(d111.Reg)
					ctx.UnprotectReg(d111.Reg2)
				}
				ctx.EmitJmp(lbl50)
				ctx.MarkLabel(lbl46)
				d113 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r29, Reg2: r30}
				ctx.BindReg(r29, &d113)
				ctx.BindReg(r30, &d113)
				ctx.BindReg(r29, &d113)
				ctx.BindReg(r30, &d113)
				ctx.StabilizeDescForControlFlow(&d113)
				ctx.FreeDesc(&d101)
				ctx.ReclaimUntrackedRegs()
				var d114 JITValueDesc
				if d31.SliceSizeKnown {
					d114 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.KnownSliceLen))}
				} else if d31.Loc == LocImm {
					d114 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.StackOff))}
				} else if d31.Loc == LocStackTriple {
					d114 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d31.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocRegTriple {
						d114 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2, ID: 0}
					} else if d31.Loc == LocReg {
						d114 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.StabilizeDescForControlFlow(&d114)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase14)+int32(48))
				bbpos_3_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d115 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d115)
				ctx.EnsureDesc(&d115)
				var d116 JITValueDesc
				if d115.Loc == LocImm {
					d116 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d115.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d115.Reg)
					ctx.EmitMovRegReg(scratch, d115.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d116 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d116)
				}
				if d116.Loc == LocReg && d115.Loc == LocReg && d116.Reg == d115.Reg {
					ctx.TransferReg(d115.Reg)
					d115.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d116)
				ctx.FreeDesc(&d115)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d116)
				ctx.EnsureDesc(&d114)
				ctx.EnsureDescsTogether(&d116, &d114)
				var d117 JITValueDesc
				if d116.Loc == LocImm && d114.Loc == LocImm {
					d117 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d116.Imm.Int() < d114.Imm.Int())}
				} else if d114.Loc == LocImm {
					r35 := ctx.AllocRegExcept(d116.Reg)
					if d114.Imm.Int() >= -2147483648 && d114.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d116.Reg, int32(d114.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d114.Imm.Int()))
						ctx.EmitCmpInt64(d116.Reg, RegR11)
					}
					d117 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r35, Condition: CondSignedLess}
					ctx.BindReg(r35, &d117)
				} else if d116.Loc == LocImm {
					r36 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d116.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d114.Reg)
					d117 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r36, Condition: CondSignedLess}
					ctx.BindReg(r36, &d117)
				} else {
					r37 := ctx.AllocRegExcept(d116.Reg)
					ctx.EmitCmpInt64(d116.Reg, d114.Reg)
					d117 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r37, Condition: CondSignedLess}
					ctx.BindReg(r37, &d117)
				}
				ctx.ReclaimUntrackedRegs()
				d118 := d117
				ctx.EnsureDesc(&d118)
				if d118.Loc != LocImm && d118.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl53 := ctx.ReserveLabel()
				lbl54 := ctx.ReserveLabel()
				if d118.Loc == LocImm {
					if d118.Imm.Bool() {
						ctx.MarkLabel(lbl53)
						ctx.EmitJmp(lbl15)
					} else {
						ctx.MarkLabel(lbl54)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase14)+int32(64))
						ctx.EmitJmp(lbl16)
					}
				} else {
					ctx.EmitJump(d118.Condition, lbl53)
					ctx.EmitJmp(lbl54)
					ctx.FreeDesc(&d117)
					ctx.MarkLabel(lbl53)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl54)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase14)+int32(64))
					ctx.EmitJmp(lbl16)
				}
				bbpos_3_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d115 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d18 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d119 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(64)}
				ctx.ReclaimUntrackedRegs()
				d120 := d119
				ctx.EnsureDesc(&d120)
				if d120.Loc != LocImm && d120.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl55 := ctx.ReserveLabel()
				lbl56 := ctx.ReserveLabel()
				if d120.Loc == LocImm {
					if d120.Imm.Bool() {
						ctx.MarkLabel(lbl55)
						ctx.SyncDesc(&d61)
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.ProtectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.ProtectReg(d61.Reg)
							ctx.ProtectReg(d61.Reg2)
						}
						d121 := d61
						if d121.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d121)
						if d121.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d121, int32(phiBase14)+int32(0), 3)
						} else {
							if d121.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d121.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d121.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d121.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d61.Loc == LocReg || d61.Loc == LocFPReg {
							ctx.UnprotectReg(d61.Reg)
						} else if d61.Loc == LocRegPair {
							ctx.UnprotectReg(d61.Reg)
							ctx.UnprotectReg(d61.Reg2)
						}
						ctx.EmitJmp(lbl8)
					} else {
						ctx.MarkLabel(lbl56)
						ctx.EmitJmp(lbl18)
					}
				} else {
					ctx.EmitCmpRegImm32(d120.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl55)
					ctx.EmitJmp(lbl56)
					ctx.MarkLabel(lbl55)
					ctx.SyncDesc(&d61)
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.ProtectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.ProtectReg(d61.Reg)
						ctx.ProtectReg(d61.Reg2)
					}
					d122 := d61
					if d122.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d122)
					if d122.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d122, int32(phiBase14)+int32(0), 3)
					} else {
						if d122.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d122.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d122.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d122.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d61.Loc == LocReg || d61.Loc == LocFPReg {
						ctx.UnprotectReg(d61.Reg)
					} else if d61.Loc == LocRegPair {
						ctx.UnprotectReg(d61.Reg)
						ctx.UnprotectReg(d61.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl56)
					ctx.EmitJmp(lbl18)
				}
				ctx.FreeDesc(&d119)
				bbpos_3_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d115 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d119 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d116)
				ctx.ReclaimUntrackedRegs()
				d124 := ctx.EmitSliceElementAddress(&d31, &d116, 16)
				ctx.EnsureDesc(&d124)
				r38 := ctx.AllocRegExcept(d124.Reg)
				ctx.EmitMovRegMem(r38, d124.Reg, 8)
				ctx.EmitMovRegMem(d124.Reg, d124.Reg, 0)
				d123 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d124.Reg, Reg2: r38}
				ctx.BindReg(d124.Reg, &d123)
				ctx.BindReg(r38, &d123)
				ctx.ReclaimUntrackedRegs()
				d123 = JITPrepareScmerGoArg(ctx, d123)
				d113 = JITPrepareScmerGoArg(ctx, d113)
				ctx.SyncDesc(&d123)
				ctx.SyncDesc(&d113)
				d125 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d123, d113}, 1)
				d125.NoHeapPointer = true
				ctx.EmitAndRegImm32(d125.Reg, 1)
				d125.Type = tagBool
				ctx.BindReg(d125.Reg, &d125)
				ctx.FreeDesc(&d123)
				ctx.ReclaimUntrackedRegs()
				d126 := d125
				ctx.EnsureDesc(&d126)
				if d126.Loc != LocImm && d126.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl57 := ctx.ReserveLabel()
				lbl58 := ctx.ReserveLabel()
				if d126.Loc == LocImm {
					if d126.Imm.Bool() {
						ctx.MarkLabel(lbl57)
						ctx.EmitJmp(lbl17)
					} else {
						ctx.MarkLabel(lbl58)
						ctx.SyncDesc(&d116)
						if d116.Loc == LocReg || d116.Loc == LocFPReg {
							ctx.ProtectReg(d116.Reg)
						} else if d116.Loc == LocRegPair {
							ctx.ProtectReg(d116.Reg)
							ctx.ProtectReg(d116.Reg2)
						}
						d127 := d116
						if d127.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d127)
						ctx.EmitStoreToStack(d127, int32(phiBase14)+int32(48))
						if d116.Loc == LocReg || d116.Loc == LocFPReg {
							ctx.UnprotectReg(d116.Reg)
						} else if d116.Loc == LocRegPair {
							ctx.UnprotectReg(d116.Reg)
							ctx.UnprotectReg(d116.Reg2)
						}
						ctx.EmitJmp(lbl14)
					}
				} else {
					ctx.EmitCmpRegImm32(d126.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl57)
					ctx.EmitJmp(lbl58)
					ctx.MarkLabel(lbl57)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl58)
					ctx.SyncDesc(&d116)
					if d116.Loc == LocReg || d116.Loc == LocFPReg {
						ctx.ProtectReg(d116.Reg)
					} else if d116.Loc == LocRegPair {
						ctx.ProtectReg(d116.Reg)
						ctx.ProtectReg(d116.Reg2)
					}
					d128 := d116
					if d128.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d128)
					ctx.EmitStoreToStack(d128, int32(phiBase14)+int32(48))
					if d116.Loc == LocReg || d116.Loc == LocFPReg {
						ctx.UnprotectReg(d116.Reg)
					} else if d116.Loc == LocRegPair {
						ctx.UnprotectReg(d116.Reg)
						ctx.UnprotectReg(d116.Reg2)
					}
					ctx.EmitJmp(lbl14)
				}
				ctx.FreeDesc(&d125)
				bbpos_3_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d115 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d119 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				stackArray129 := ctx.AllocStack(int32(16))
				_ = stackArray129
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d113)
				ctx.EmitStoreScmerToStack(d113, int32(stackArray129)+int32(0))
				ctx.ReclaimUntrackedRegs()
				d130 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d130
				ctx.ReclaimUntrackedRegs()
				r39 := ctx.AllocReg()
				r40 := ctx.AllocRegExcept(r39)
				r41 := ctx.AllocRegExcept(r39, r40)
				d131 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r39, Reg2: r40, Reg3: r41}
				ctx.BindReg(r39, &d131)
				ctx.BindReg(r40, &d131)
				ctx.BindReg(r41, &d131)
				ctx.BindReg(r39, &d131)
				ctx.BindReg(r40, &d131)
				ctx.BindReg(r41, &d131)
				ctx.EmitLeaRegMem(d131.Reg, ctx.StackReg, int32(stackArray129))
				ctx.EmitMovRegImm64(d131.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d131.Reg3, uint64(1))
				callResults132 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d31, d131}, []uint8{3}, []uint8{1})
				d133 := callResults132[0]
				d134 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase14) + int32(24)}
				ctx.EmitCopyDescWords(&d134, &d133, 3)
				ctx.FreeDesc(&d133)
				d133 = d134
				ctx.StabilizeDescForControlFlow(&d133)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d61)
				if d61.Loc == LocReg || d61.Loc == LocFPReg {
					ctx.ProtectReg(d61.Reg)
				} else if d61.Loc == LocRegPair {
					ctx.ProtectReg(d61.Reg)
					ctx.ProtectReg(d61.Reg2)
				}
				d135 := d61
				if d135.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d135)
				if d135.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d135, int32(phiBase14)+int32(0), 3)
				} else {
					if d135.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d135.Reg, RegRSP, int32(phiBase14)+int32(0))
					ctx.EmitStoreRegMem(d135.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
					ctx.EmitStoreRegMem(d135.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
				}
				if d61.Loc == LocReg || d61.Loc == LocFPReg {
					ctx.UnprotectReg(d61.Reg)
				} else if d61.Loc == LocRegPair {
					ctx.UnprotectReg(d61.Reg)
					ctx.UnprotectReg(d61.Reg2)
				}
				ctx.EmitJmp(lbl8)
				bbpos_3_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
				d30 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d115 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(48)}
				d119 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase14) + int32(64)}
				d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				d76 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(96)}
				d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(120)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase14)+int32(64))
				ctx.EmitJmp(lbl16)
				ctx.MarkLabel(lbl2)
				d136 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d136)
				ctx.BindReg(r3, &d136)
				ctx.BindReg(r2, &d136)
				ctx.BindReg(r3, &d136)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d136)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				ctx.FreeDesc(&d4)
				if d136.Loc == LocImm {
					if result.Loc == LocAny {
						return d136
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d136)
				if d136.Loc == LocRegPair || d136.Loc == LocStackPair || d136.Loc == LocInputPair {
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
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
		},
	})
	Declare(&Globalenv, &Declaration{
		Name:          "optimizer_expr_tagged_nth_matches_any",
		OptimizerOnly: true,
		Fn: func(a ...Scmer) Scmer {
			return NewBool(exprTaggedNthMatchesAny(a[0], a[1], int(ToInt(a[2])), a[3], asSlice(a[4], "optimizer_expr_tagged_nth_matches_any")))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "tests whether a tagged operand has any expected nth value, replacing nil with a supplied default",
			Params: []*TypeDescriptor{
				{Kind: "any", Label: "expression", NoEscape: true},
				{Kind: "any", Label: "tag", NoEscape: true},
				{Kind: "number", Label: "position"},
				{Kind: "any", Label: "nil replacement", NoEscape: true},
				{Kind: "list", Label: "expected values", NoEscape: true},
			},
			Return: &TypeDescriptor{Kind: "bool"},
			Const:  true,
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				declaration := declarations["optimizer_expr_tagged_nth_matches_any"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				d2 := args[2]
				d2.ID = 0
				ctx.EnsureDesc(&d2)
				d3 := d2
				_ = d3
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				lbl0 := ctx.ReserveLabel()
				_ = lbl0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d4 JITValueDesc
				if d3.Loc == LocImm {
					d4 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int())}
				} else if d3.Type == tagInt && d3.Loc == LocRegPair {
					ctx.FreeReg(d3.Reg)
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2}
					ctx.BindReg(d3.Reg2, &d4)
					ctx.BindReg(d3.Reg2, &d4)
				} else if d3.Type == tagInt && d3.Loc == LocReg {
					d4 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg}
					ctx.BindReg(d3.Reg, &d4)
					ctx.BindReg(d3.Reg, &d4)
				} else {
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d3}, 1)
					d4.Type = tagInt
					ctx.BindReg(d4.Reg, &d4)
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.FreeDesc(&d2)
				d6 := args[3]
				d6.ID = 0
				d7 := args[4]
				d7.ID = 0
				var d8 JITValueDesc
				if d7.Type == tagSlice {
					d8 = jitKnownSliceHeader(ctx, &d7)
				} else {
					d8 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d7}, 3)
				}
				ctx.BindReg(d8.Reg, &d8)
				ctx.BindReg(d8.Reg2, &d8)
				ctx.BindReg(d8.Reg3, &d8)
				ctx.FreeDesc(&d7)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d6)
				ctx.EnsureDesc(&d8)
				d9 := d0
				_ = d9
				ctx.StabilizeDescForControlFlow(&d9)
				d10 := d1
				_ = d10
				ctx.StabilizeDescForControlFlow(&d10)
				d11 := d4
				_ = d11
				ctx.StabilizeDescForControlFlow(&d11)
				d12 := d6
				_ = d12
				ctx.StabilizeDescForControlFlow(&d12)
				d13 := d8
				_ = d13
				ctx.StabilizeDescForControlFlow(&d13)
				phiBase14 := ctx.AllocStack(int32(96))
				d15 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				ctx.PreparePointerStackTarget(int32(phiBase14)+int32(0), 3)
				_ = d15
				d16 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				ctx.PrepareScmerStackTarget(int32(phiBase14) + int32(24))
				_ = d16
				d17 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				_ = d17
				d18 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				ctx.PreparePointerStackTarget(int32(phiBase14)+int32(56), 3)
				_ = d18
				d19 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				_ = d19
				lbl1 := ctx.ReserveLabel()
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				lbl2 := ctx.ReserveLabel()
				_ = lbl2
				bbpos_2_1 := int32(-1)
				_ = bbpos_2_1
				lbl3 := ctx.ReserveLabel()
				_ = lbl3
				bbpos_2_2 := int32(-1)
				_ = bbpos_2_2
				lbl4 := ctx.ReserveLabel()
				_ = lbl4
				bbpos_2_3 := int32(-1)
				_ = bbpos_2_3
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_2_4 := int32(-1)
				_ = bbpos_2_4
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
				bbpos_2_5 := int32(-1)
				_ = bbpos_2_5
				lbl7 := ctx.ReserveLabel()
				_ = lbl7
				bbpos_2_6 := int32(-1)
				_ = bbpos_2_6
				lbl8 := ctx.ReserveLabel()
				_ = lbl8
				bbpos_2_7 := int32(-1)
				_ = bbpos_2_7
				lbl9 := ctx.ReserveLabel()
				_ = lbl9
				bbpos_2_8 := int32(-1)
				_ = bbpos_2_8
				lbl10 := ctx.ReserveLabel()
				_ = lbl10
				bbpos_2_9 := int32(-1)
				_ = bbpos_2_9
				lbl11 := ctx.ReserveLabel()
				_ = lbl11
				bbpos_2_10 := int32(-1)
				_ = bbpos_2_10
				lbl12 := ctx.ReserveLabel()
				_ = lbl12
				bbpos_2_11 := int32(-1)
				_ = bbpos_2_11
				lbl13 := ctx.ReserveLabel()
				_ = lbl13
				bbpos_2_12 := int32(-1)
				_ = bbpos_2_12
				lbl14 := ctx.ReserveLabel()
				_ = lbl14
				bbpos_2_13 := int32(-1)
				_ = bbpos_2_13
				lbl15 := ctx.ReserveLabel()
				_ = lbl15
				bbpos_2_14 := int32(-1)
				_ = bbpos_2_14
				lbl16 := ctx.ReserveLabel()
				_ = lbl16
				bbpos_2_15 := int32(-1)
				_ = bbpos_2_15
				lbl17 := ctx.ReserveLabel()
				_ = lbl17
				bbpos_2_16 := int32(-1)
				_ = bbpos_2_16
				lbl18 := ctx.ReserveLabel()
				_ = lbl18
				bbpos_2_17 := int32(-1)
				_ = bbpos_2_17
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d11)
				var d20 JITValueDesc
				if d11.Loc == LocImm {
					d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Int() < 0)}
				} else {
					r0 := ctx.AllocRegExcept(d11.Reg)
					ctx.EmitCmpRegImm32(d11.Reg, 0)
					d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
					ctx.BindReg(r0, &d20)
				}
				ctx.ReclaimUntrackedRegs()
				d21 := d20
				ctx.EnsureDesc(&d21)
				if d21.Loc != LocImm && d21.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl20 := ctx.ReserveLabel()
				lbl21 := ctx.ReserveLabel()
				if d21.Loc == LocImm {
					if d21.Imm.Bool() {
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl3)
					} else {
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl4)
					}
				} else {
					ctx.EmitJump(d21.Condition, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.FreeDesc(&d20)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl4)
				}
				bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				stackArray22 := ctx.AllocStack(int32(1024))
				_ = stackArray22
				ctx.ReclaimUntrackedRegs()
				d23 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(64), SliceSizeKnown: true}
				_ = d23
				ctx.StabilizeDescForControlFlow(&d23)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d9)
				ctx.EmitStoreScmerToStack(d9, int32(stackArray22)+int32(0))
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d23)
				if d23.Loc == LocReg || d23.Loc == LocFPReg {
					ctx.ProtectReg(d23.Reg)
				} else if d23.Loc == LocRegPair {
					ctx.ProtectReg(d23.Reg)
					ctx.ProtectReg(d23.Reg2)
				}
				d24 := d23
				if d24.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d24)
				if d24.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d24, int32(phiBase14)+int32(0), 3)
				} else {
					if d24.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d24.Reg, RegRSP, int32(phiBase14)+int32(0))
					ctx.EmitStoreRegMem(d24.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
					ctx.EmitStoreRegMem(d24.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
				}
				if d23.Loc == LocReg || d23.Loc == LocFPReg {
					ctx.UnprotectReg(d23.Reg)
				} else if d23.Loc == LocRegPair {
					ctx.UnprotectReg(d23.Reg)
					ctx.UnprotectReg(d23.Reg2)
				}
				bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d25 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d25)
				ctx.ReclaimUntrackedRegs()
				var d26 JITValueDesc
				if d25.SliceSizeKnown {
					d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.KnownSliceLen))}
				} else if d25.Loc == LocImm {
					d26 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.StackOff))}
				} else if d25.Loc == LocStackTriple {
					d26 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d25.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocRegPair || d25.Loc == LocRegTriple {
						d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg2, ID: 0}
					} else if d25.Loc == LocReg {
						d26 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d26)
				var d27 JITValueDesc
				if d26.Loc == LocImm {
					d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d26.Imm.Int() != 0)}
				} else {
					r1 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d26.Reg, 0)
					d27 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondNotEqual}
					ctx.BindReg(r1, &d27)
				}
				ctx.FreeDesc(&d26)
				ctx.ReclaimUntrackedRegs()
				d28 := d27
				ctx.EnsureDesc(&d28)
				if d28.Loc != LocImm && d28.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl22 := ctx.ReserveLabel()
				lbl23 := ctx.ReserveLabel()
				if d28.Loc == LocImm {
					if d28.Imm.Bool() {
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl5)
					} else {
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl6)
					}
				} else {
					ctx.EmitJump(d28.Condition, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.FreeDesc(&d27)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl6)
				}
				bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r2 := ctx.AllocReg()
				d29 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				ctx.EnsureDesc(&d29)
				if d29.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r2, d29)
				}
				ctx.EmitJmp(lbl1)
				bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d30 JITValueDesc
				if d25.SliceSizeKnown {
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.KnownSliceLen))}
				} else if d25.Loc == LocImm {
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d25.StackOff))}
				} else if d25.Loc == LocStackTriple {
					d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d25.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d25)
					if d25.Loc == LocRegPair || d25.Loc == LocRegTriple {
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg2, ID: 0}
					} else if d25.Loc == LocReg {
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d30)
				ctx.EnsureDesc(&d30)
				var d31 JITValueDesc
				if d30.Loc == LocImm {
					d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d30.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d30.Reg)
					ctx.EmitMovRegReg(scratch, d30.Reg)
					ctx.EmitSubRegImm32(scratch, int32(1))
					d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d31)
				}
				if d31.Loc == LocReg && d30.Loc == LocReg && d31.Reg == d30.Reg {
					ctx.TransferReg(d30.Reg)
					d30.Loc = LocNone
				}
				ctx.FreeDesc(&d30)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d31)
				ctx.ReclaimUntrackedRegs()
				d33 := ctx.EmitSliceElementAddress(&d25, &d31, 16)
				ctx.EnsureDesc(&d33)
				r3 := ctx.AllocRegExcept(d33.Reg)
				ctx.EmitMovRegMem(r3, d33.Reg, 8)
				ctx.EmitMovRegMem(d33.Reg, d33.Reg, 0)
				d32 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d33.Reg, Reg2: r3}
				ctx.BindReg(d33.Reg, &d32)
				ctx.BindReg(r3, &d32)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d32)
				phiBase34 := ctx.AllocStack(int32(16))
				d35 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase34) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase34) + int32(0))
				_ = d35
				lbl24 := ctx.ReserveLabel()
				bbpos_3_0 := int32(-1)
				_ = bbpos_3_0
				lbl25 := ctx.ReserveLabel()
				_ = lbl25
				bbpos_3_1 := int32(-1)
				_ = bbpos_3_1
				lbl26 := ctx.ReserveLabel()
				_ = lbl26
				bbpos_3_2 := int32(-1)
				_ = bbpos_3_2
				lbl27 := ctx.ReserveLabel()
				_ = lbl27
				bbpos_3_3 := int32(-1)
				_ = bbpos_3_3
				lbl28 := ctx.ReserveLabel()
				_ = lbl28
				bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl25)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase34) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d32)
				if d32.Loc == LocReg || d32.Loc == LocFPReg {
					ctx.ProtectReg(d32.Reg)
				} else if d32.Loc == LocRegPair {
					ctx.ProtectReg(d32.Reg)
					ctx.ProtectReg(d32.Reg2)
				}
				d36 := d32
				if d36.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d36)
				if d36.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d36, int32(phiBase34)+int32(0), 2)
				} else if d36.Loc == LocInputPair {
					ctx.EnsureDesc(&d36)
					ctx.EmitStoreScmerToStack(d36, int32(phiBase34)+int32(0))
				} else if d36.Loc == LocRegPair || d36.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d36, int32(phiBase34)+int32(0))
				} else {
					ctx.EnsureDesc(&d36)
					ctx.EmitStoreToStack(d36, int32(phiBase34)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase34)+int32(0))+8)
				}
				if d32.Loc == LocReg || d32.Loc == LocFPReg {
					ctx.UnprotectReg(d32.Reg)
				} else if d32.Loc == LocRegPair {
					ctx.UnprotectReg(d32.Reg)
					ctx.UnprotectReg(d32.Reg2)
				}
				bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl28)
				ctx.ResolveFixups()
				d35 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase34) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d37 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase34) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d37)
				ctx.ReclaimUntrackedRegs()
				d37 = JITPrepareScmerGoArg(ctx, d37)
				ctx.SyncDesc(&d37)
				d38 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d37}, 1)
				d38.NoHeapPointer = true
				ctx.EmitAndRegImm32(d38.Reg, 1)
				d38.Type = tagBool
				ctx.BindReg(d38.Reg, &d38)
				ctx.ReclaimUntrackedRegs()
				d39 := d38
				ctx.EnsureDesc(&d39)
				if d39.Loc != LocImm && d39.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl29 := ctx.ReserveLabel()
				lbl30 := ctx.ReserveLabel()
				if d39.Loc == LocImm {
					if d39.Imm.Bool() {
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl26)
					} else {
						ctx.MarkLabel(lbl30)
						ctx.EmitJmp(lbl27)
					}
				} else {
					ctx.EmitCmpRegImm32(d39.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl29)
					ctx.EmitJmp(lbl30)
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl30)
					ctx.EmitJmp(lbl27)
				}
				ctx.FreeDesc(&d38)
				bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl27)
				ctx.ResolveFixups()
				d37 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase34) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				d40 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d40)
				ctx.BindReg(r5, &d40)
				ctx.EmitMovPairToResult(&d37, &d40)
				ctx.EmitJmp(lbl24)
				bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl26)
				ctx.ResolveFixups()
				d37 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase34) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d37 = JITPrepareScmerGoArg(ctx, d37)
				ctx.SyncDesc(&d37)
				d41 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d37}, 1)
				d41.NoHeapPointer = false
				ctx.BindReg(d41.Reg, &d41)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d42 JITValueDesc
				ctx.EnsureDesc(&d41)
				if d41.Loc == LocImm {
					fieldAddr := uintptr(d41.Imm.Int()) + 32
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					ctx.EmitMovRegMem64(r6, fieldAddr)
					ctx.EmitMovRegMem64(r7, fieldAddr+8)
					d42 = JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d42)
					ctx.BindReg(r7, &d42)
				} else {
					off := int32(32)
					baseReg := d41.Reg
					r8 := ctx.AllocRegExcept(baseReg)
					r9 := ctx.AllocRegExcept(baseReg, r8)
					ctx.EmitMovRegMem(r8, baseReg, off)
					ctx.EmitMovRegMem(r9, baseReg, off+8)
					d42 = JITValueDesc{Loc: LocRegPair, Reg: r8, Reg2: r9}
					ctx.BindReg(r8, &d42)
					ctx.BindReg(r9, &d42)
				}
				ctx.FreeDesc(&d41)
				ctx.StabilizeDescForControlFlow(&d42)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d42)
				if d42.Loc == LocReg || d42.Loc == LocFPReg {
					ctx.ProtectReg(d42.Reg)
				} else if d42.Loc == LocRegPair {
					ctx.ProtectReg(d42.Reg)
					ctx.ProtectReg(d42.Reg2)
				}
				d43 := d42
				if d43.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d43)
				if d43.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d43, int32(phiBase34)+int32(0), 2)
				} else if d43.Loc == LocInputPair {
					ctx.EnsureDesc(&d43)
					ctx.EmitStoreScmerToStack(d43, int32(phiBase34)+int32(0))
				} else if d43.Loc == LocRegPair || d43.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d43, int32(phiBase34)+int32(0))
				} else {
					ctx.EnsureDesc(&d43)
					ctx.EmitStoreToStack(d43, int32(phiBase34)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase34)+int32(0))+8)
				}
				if d42.Loc == LocReg || d42.Loc == LocFPReg {
					ctx.UnprotectReg(d42.Reg)
				} else if d42.Loc == LocRegPair {
					ctx.UnprotectReg(d42.Reg)
					ctx.UnprotectReg(d42.Reg2)
				}
				ctx.EmitJmp(lbl28)
				ctx.MarkLabel(lbl24)
				d44 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d44)
				ctx.BindReg(r5, &d44)
				ctx.BindReg(r4, &d44)
				ctx.BindReg(r5, &d44)
				ctx.StabilizeDescForControlFlow(&d44)
				ctx.FreeDesc(&d32)
				ctx.ReclaimUntrackedRegs()
				d45 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d31)
				ctx.EnsureDesc(&d25)
				ctx.EnsureDesc(&d45)
				ctx.EnsureDesc(&d31)
				var d47 JITValueDesc
				if d31.Loc == LocImm && d45.Loc == LocImm {
					d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d31.Imm.Int() - d45.Imm.Int())}
				} else {
					r10 := ctx.AllocReg()
					if d31.Loc == LocImm {
						ctx.EmitMovRegImm64(r10, uint64(d31.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r10, d31.Reg)
					}
					if d45.Loc == LocImm {
						ctx.EmitMovRegImm64(RegR11, uint64(d45.Imm.Int()))
						ctx.EmitSubInt64(r10, RegR11)
					} else {
						ctx.EmitSubInt64(r10, d45.Reg)
					}
					d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
					ctx.BindReg(r10, &d47)
				}
				var d48 JITValueDesc
				r11 := ctx.EmitSliceDataAfterLow(&d25, &d45, 16)
				d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d48)
				ctx.BindReg(r11, &d48)
				var d49 JITValueDesc
				var r12 Reg
				var r13 Reg
				ctx.SyncDesc(&d48)
				ctx.EnsureDesc(&d48)
				if d48.Loc == LocImm {
					r12 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r12, uint64(d48.Imm.Int()))
				} else {
					r12 = d48.Reg
				}
				ctx.ProtectReg(r12)
				ctx.SyncDesc(&d47)
				ctx.EnsureDesc(&d47)
				if d47.Loc == LocImm {
					r13 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r13, uint64(d47.Imm.Int()))
				} else {
					r13 = d47.Reg
				}
				ctx.ProtectReg(r13)
				r14 := ctx.EmitSliceCapAfterLow(&d25, &d45, r12, r13)
				ctx.UnprotectReg(r13)
				ctx.UnprotectReg(r12)
				d49 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
				ctx.BindReg(r12, &d49)
				ctx.BindReg(r13, &d49)
				ctx.BindReg(r14, &d49)
				ctx.BindReg(r12, &d49)
				ctx.BindReg(r13, &d49)
				ctx.BindReg(r14, &d49)
				ctx.StabilizeDescForControlFlow(&d49)
				ctx.FreeDesc(&d31)
				ctx.ReclaimUntrackedRegs()
				d51 := d44
				d51.ID = 0
				d50 := ctx.EmitTagEqualsBorrowed(&d51, tagSlice, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				d52 := d50
				ctx.EnsureDesc(&d52)
				if d52.Loc != LocImm && d52.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl31 := ctx.ReserveLabel()
				lbl32 := ctx.ReserveLabel()
				if d52.Loc == LocImm {
					if d52.Imm.Bool() {
						ctx.MarkLabel(lbl31)
						ctx.EmitJmp(lbl8)
					} else {
						ctx.MarkLabel(lbl32)
						ctx.SyncDesc(&d49)
						if d49.Loc == LocReg || d49.Loc == LocFPReg {
							ctx.ProtectReg(d49.Reg)
						} else if d49.Loc == LocRegPair {
							ctx.ProtectReg(d49.Reg)
							ctx.ProtectReg(d49.Reg2)
						}
						d53 := d49
						if d53.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d53)
						if d53.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d53, int32(phiBase14)+int32(0), 3)
						} else {
							if d53.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d53.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d53.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d53.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d49.Loc == LocReg || d49.Loc == LocFPReg {
							ctx.UnprotectReg(d49.Reg)
						} else if d49.Loc == LocRegPair {
							ctx.UnprotectReg(d49.Reg)
							ctx.UnprotectReg(d49.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitCmpRegImm32(d52.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl31)
					ctx.EmitJmp(lbl32)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl32)
					ctx.SyncDesc(&d49)
					if d49.Loc == LocReg || d49.Loc == LocFPReg {
						ctx.ProtectReg(d49.Reg)
					} else if d49.Loc == LocRegPair {
						ctx.ProtectReg(d49.Reg)
						ctx.ProtectReg(d49.Reg2)
					}
					d54 := d49
					if d54.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d54)
					if d54.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d54, int32(phiBase14)+int32(0), 3)
					} else {
						if d54.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d54.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d54.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d54.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d49.Loc == LocReg || d49.Loc == LocFPReg {
						ctx.UnprotectReg(d49.Reg)
					} else if d49.Loc == LocRegPair {
						ctx.UnprotectReg(d49.Reg)
						ctx.UnprotectReg(d49.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				ctx.FreeDesc(&d50)
				bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d55 := jitKnownSliceHeader(ctx, &d44)
				ctx.StabilizeDescForControlFlow(&d55)
				ctx.ReclaimUntrackedRegs()
				var d56 JITValueDesc
				if d55.SliceSizeKnown {
					d56 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d55.KnownSliceLen))}
				} else if d55.Loc == LocImm {
					d56 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d55.StackOff))}
				} else if d55.Loc == LocStackTriple {
					d56 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d55.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocRegPair || d55.Loc == LocRegTriple {
						d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg2, ID: 0}
					} else if d55.Loc == LocReg {
						d56 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d56)
				var d57 JITValueDesc
				if d56.Loc == LocImm {
					d57 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d56.Imm.Int() != 0)}
				} else {
					r15 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d56.Reg, 0)
					d57 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondNotEqual}
					ctx.BindReg(r15, &d57)
				}
				ctx.FreeDesc(&d56)
				ctx.ReclaimUntrackedRegs()
				d58 := d57
				ctx.EnsureDesc(&d58)
				if d58.Loc != LocImm && d58.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl33 := ctx.ReserveLabel()
				lbl34 := ctx.ReserveLabel()
				if d58.Loc == LocImm {
					if d58.Imm.Bool() {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl11)
					} else {
						ctx.MarkLabel(lbl34)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitJump(d58.Condition, lbl33)
					ctx.EmitJmp(lbl34)
					ctx.FreeDesc(&d57)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl10)
				}
				bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d59 JITValueDesc
				if d55.SliceSizeKnown {
					d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d55.KnownSliceLen))}
				} else if d55.Loc == LocImm {
					d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d55.StackOff))}
				} else if d55.Loc == LocStackTriple {
					d59 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d55.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocRegPair || d55.Loc == LocRegTriple {
						d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg2, ID: 0}
					} else if d55.Loc == LocReg {
						d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d59)
				ctx.EnsureDesc(&d59)
				var d60 JITValueDesc
				if d59.Loc == LocImm {
					d60 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d59.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d59.Reg)
					ctx.EmitMovRegReg(scratch, d59.Reg)
					ctx.EmitSubRegImm32(scratch, int32(1))
					d60 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d60)
				}
				if d60.Loc == LocReg && d59.Loc == LocReg && d60.Reg == d59.Reg {
					ctx.TransferReg(d59.Reg)
					d59.Loc = LocNone
				}
				ctx.EnsureDesc(&d60)
				ctx.EmitStoreToStack(d60, int32(phiBase14)+int32(80))
				ctx.StabilizeDescForControlFlow(&d60)
				ctx.FreeDesc(&d59)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d49)
				if d49.Loc == LocReg || d49.Loc == LocFPReg {
					ctx.ProtectReg(d49.Reg)
				} else if d49.Loc == LocRegPair {
					ctx.ProtectReg(d49.Reg)
					ctx.ProtectReg(d49.Reg2)
				}
				d61 := d49
				if d61.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d61)
				if d61.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d61, int32(phiBase14)+int32(56), 3)
				} else {
					if d61.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d61.Reg, RegRSP, int32(phiBase14)+int32(56))
					ctx.EmitStoreRegMem(d61.Reg2, RegRSP, int32(phiBase14)+int32(56)+8)
					ctx.EmitStoreRegMem(d61.Reg3, RegRSP, int32(phiBase14)+int32(56)+16)
				}
				if d49.Loc == LocReg || d49.Loc == LocFPReg {
					ctx.UnprotectReg(d49.Reg)
				} else if d49.Loc == LocRegPair {
					ctx.UnprotectReg(d49.Reg)
					ctx.UnprotectReg(d49.Reg2)
				}
				bbpos_2_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d62 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				ctx.StabilizeDescForControlFlow(&d62)
				ctx.ReclaimUntrackedRegs()
				d63 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(80)}
				ctx.StabilizeDescForControlFlow(&d63)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d63)
				var d64 JITValueDesc
				if d63.Loc == LocImm {
					d64 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d63.Imm.Int() >= 1)}
				} else {
					r16 := ctx.AllocRegExcept(d63.Reg)
					ctx.EmitCmpRegImm32(d63.Reg, 1)
					d64 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r16, &d64)
				}
				ctx.ReclaimUntrackedRegs()
				d65 := d64
				ctx.EnsureDesc(&d65)
				if d65.Loc != LocImm && d65.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl35 := ctx.ReserveLabel()
				lbl36 := ctx.ReserveLabel()
				if d65.Loc == LocImm {
					if d65.Imm.Bool() {
						ctx.MarkLabel(lbl35)
						ctx.EmitJmp(lbl19)
					} else {
						ctx.MarkLabel(lbl36)
						ctx.SyncDesc(&d62)
						if d62.Loc == LocReg || d62.Loc == LocFPReg {
							ctx.ProtectReg(d62.Reg)
						} else if d62.Loc == LocRegPair {
							ctx.ProtectReg(d62.Reg)
							ctx.ProtectReg(d62.Reg2)
						}
						d66 := d62
						if d66.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d66)
						if d66.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d66, int32(phiBase14)+int32(0), 3)
						} else {
							if d66.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d66.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d66.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d66.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d62.Loc == LocReg || d62.Loc == LocFPReg {
							ctx.UnprotectReg(d62.Reg)
						} else if d62.Loc == LocRegPair {
							ctx.UnprotectReg(d62.Reg)
							ctx.UnprotectReg(d62.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d65.Condition, lbl35)
					ctx.EmitJmp(lbl36)
					ctx.FreeDesc(&d64)
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl36)
					ctx.SyncDesc(&d62)
					if d62.Loc == LocReg || d62.Loc == LocFPReg {
						ctx.ProtectReg(d62.Reg)
					} else if d62.Loc == LocRegPair {
						ctx.ProtectReg(d62.Reg)
						ctx.ProtectReg(d62.Reg2)
					}
					d67 := d62
					if d67.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d67)
					if d67.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d67, int32(phiBase14)+int32(0), 3)
					} else {
						if d67.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d67.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d67.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d67.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d62.Loc == LocReg || d62.Loc == LocFPReg {
						ctx.UnprotectReg(d62.Reg)
					} else if d62.Loc == LocRegPair {
						ctx.UnprotectReg(d62.Reg)
						ctx.UnprotectReg(d62.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				bbpos_2_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d68 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.ReclaimUntrackedRegs()
				d70 := ctx.EmitSliceElementAddress(&d55, &d68, 16)
				ctx.EnsureDesc(&d70)
				r17 := ctx.AllocRegExcept(d70.Reg)
				ctx.EmitMovRegMem(r17, d70.Reg, 8)
				ctx.EmitMovRegMem(d70.Reg, d70.Reg, 0)
				d69 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d70.Reg, Reg2: r17}
				ctx.BindReg(d70.Reg, &d69)
				ctx.BindReg(r17, &d69)
				ctx.ReclaimUntrackedRegs()
				d69 = JITPrepareScmerGoArg(ctx, d69)
				d10 = JITPrepareScmerGoArg(ctx, d10)
				ctx.SyncDesc(&d69)
				ctx.SyncDesc(&d10)
				d71 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d69, d10}, 1)
				d71.NoHeapPointer = true
				ctx.EmitAndRegImm32(d71.Reg, 1)
				d71.Type = tagBool
				ctx.BindReg(d71.Reg, &d71)
				ctx.FreeDesc(&d69)
				ctx.ReclaimUntrackedRegs()
				d72 := d71
				ctx.EnsureDesc(&d72)
				if d72.Loc != LocImm && d72.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl37 := ctx.ReserveLabel()
				lbl38 := ctx.ReserveLabel()
				if d72.Loc == LocImm {
					if d72.Imm.Bool() {
						ctx.MarkLabel(lbl37)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl38)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitCmpRegImm32(d72.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl37)
					ctx.EmitJmp(lbl38)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl38)
					ctx.EmitJmp(lbl10)
				}
				ctx.FreeDesc(&d71)
				bbpos_2_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d63)
				ctx.ReclaimUntrackedRegs()
				d74 := ctx.EmitSliceElementAddress(&d55, &d63, 16)
				ctx.EnsureDesc(&d74)
				r18 := ctx.AllocRegExcept(d74.Reg)
				ctx.EmitMovRegMem(r18, d74.Reg, 8)
				ctx.EmitMovRegMem(d74.Reg, d74.Reg, 0)
				d73 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d74.Reg, Reg2: r18}
				ctx.BindReg(d74.Reg, &d73)
				ctx.BindReg(r18, &d73)
				ctx.ReclaimUntrackedRegs()
				stackArray75 := ctx.AllocStack(int32(16))
				_ = stackArray75
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d73)
				ctx.EmitStoreScmerToStack(d73, int32(stackArray75)+int32(0))
				ctx.FreeDesc(&d73)
				ctx.ReclaimUntrackedRegs()
				d76 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d76
				ctx.ReclaimUntrackedRegs()
				r19 := ctx.AllocReg()
				r20 := ctx.AllocRegExcept(r19)
				r21 := ctx.AllocRegExcept(r19, r20)
				d77 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r19, Reg2: r20, Reg3: r21}
				ctx.BindReg(r19, &d77)
				ctx.BindReg(r20, &d77)
				ctx.BindReg(r21, &d77)
				ctx.BindReg(r19, &d77)
				ctx.BindReg(r20, &d77)
				ctx.BindReg(r21, &d77)
				ctx.EmitLeaRegMem(d77.Reg, ctx.StackReg, int32(stackArray75))
				ctx.EmitMovRegImm64(d77.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d77.Reg3, uint64(1))
				callResults78 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d62, d77}, []uint8{3}, []uint8{1})
				d79 := callResults78[0]
				d80 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase14) + int32(56)}
				ctx.EmitCopyDescWords(&d80, &d79, 3)
				ctx.FreeDesc(&d79)
				d79 = d80
				ctx.StabilizeDescForControlFlow(&d79)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d63)
				ctx.EnsureDesc(&d63)
				var d81 JITValueDesc
				if d63.Loc == LocImm {
					d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d63.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d63.Reg)
					ctx.EmitMovRegReg(scratch, d63.Reg)
					ctx.EmitSubRegImm32(scratch, int32(1))
					d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d81)
				}
				if d81.Loc == LocReg && d63.Loc == LocReg && d81.Reg == d63.Reg {
					ctx.TransferReg(d63.Reg)
					d63.Loc = LocNone
				}
				ctx.EnsureDesc(&d81)
				ctx.EmitStoreToStack(d81, int32(phiBase14)+int32(80))
				ctx.StabilizeDescForControlFlow(&d81)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl18)
				bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d82 JITValueDesc
				if d55.SliceSizeKnown {
					d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d55.KnownSliceLen))}
				} else if d55.Loc == LocImm {
					d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d55.StackOff))}
				} else if d55.Loc == LocStackTriple {
					d82 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d55.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d55)
					if d55.Loc == LocRegPair || d55.Loc == LocRegTriple {
						d82 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg2, ID: 0}
					} else if d55.Loc == LocReg {
						d82 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d82)
				ctx.EnsureDesc(&d11)
				ctx.EnsureDescsTogether(&d82, &d11)
				var d83 JITValueDesc
				if d82.Loc == LocImm && d11.Loc == LocImm {
					d83 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d82.Imm.Int() > d11.Imm.Int())}
				} else if d11.Loc == LocImm {
					r22 := ctx.AllocReg()
					if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d82.Reg, int32(d11.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d11.Imm.Int()))
						ctx.EmitCmpInt64(d82.Reg, RegR11)
					}
					d83 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedGreater}
					ctx.BindReg(r22, &d83)
				} else if d82.Loc == LocImm {
					r23 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d82.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d11.Reg)
					d83 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondSignedGreater}
					ctx.BindReg(r23, &d83)
				} else {
					r24 := ctx.AllocReg()
					ctx.EmitCmpInt64(d82.Reg, d11.Reg)
					d83 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedGreater}
					ctx.BindReg(r24, &d83)
				}
				ctx.FreeDesc(&d82)
				ctx.ReclaimUntrackedRegs()
				d84 := d83
				ctx.EnsureDesc(&d84)
				if d84.Loc != LocImm && d84.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl39 := ctx.ReserveLabel()
				lbl40 := ctx.ReserveLabel()
				if d84.Loc == LocImm {
					if d84.Imm.Bool() {
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl40)
						ctx.SyncDesc(&d49)
						if d49.Loc == LocReg || d49.Loc == LocFPReg {
							ctx.ProtectReg(d49.Reg)
						} else if d49.Loc == LocRegPair {
							ctx.ProtectReg(d49.Reg)
							ctx.ProtectReg(d49.Reg2)
						}
						d85 := d49
						if d85.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d85)
						if d85.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d85, int32(phiBase14)+int32(0), 3)
						} else {
							if d85.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d85.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d85.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d85.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d49.Loc == LocReg || d49.Loc == LocFPReg {
							ctx.UnprotectReg(d49.Reg)
						} else if d49.Loc == LocRegPair {
							ctx.UnprotectReg(d49.Reg)
							ctx.UnprotectReg(d49.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d84.Condition, lbl39)
					ctx.EmitJmp(lbl40)
					ctx.FreeDesc(&d83)
					ctx.MarkLabel(lbl39)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl40)
					ctx.SyncDesc(&d49)
					if d49.Loc == LocReg || d49.Loc == LocFPReg {
						ctx.ProtectReg(d49.Reg)
					} else if d49.Loc == LocRegPair {
						ctx.ProtectReg(d49.Reg)
						ctx.ProtectReg(d49.Reg2)
					}
					d86 := d49
					if d86.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d86)
					if d86.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d86, int32(phiBase14)+int32(0), 3)
					} else {
						if d86.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d86.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d86.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d86.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d49.Loc == LocReg || d49.Loc == LocFPReg {
						ctx.UnprotectReg(d49.Reg)
					} else if d49.Loc == LocRegPair {
						ctx.UnprotectReg(d49.Reg)
						ctx.UnprotectReg(d49.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				bbpos_2_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d11)
				ctx.ReclaimUntrackedRegs()
				d88 := ctx.EmitSliceElementAddress(&d55, &d11, 16)
				ctx.EnsureDesc(&d88)
				r25 := ctx.AllocRegExcept(d88.Reg)
				ctx.EmitMovRegMem(r25, d88.Reg, 8)
				ctx.EmitMovRegMem(d88.Reg, d88.Reg, 0)
				d87 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d88.Reg, Reg2: r25}
				ctx.BindReg(d88.Reg, &d87)
				ctx.BindReg(r25, &d87)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d87)
				phiBase89 := ctx.AllocStack(int32(16))
				d90 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase89) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase89) + int32(0))
				_ = d90
				lbl41 := ctx.ReserveLabel()
				bbpos_4_0 := int32(-1)
				_ = bbpos_4_0
				lbl42 := ctx.ReserveLabel()
				_ = lbl42
				bbpos_4_1 := int32(-1)
				_ = bbpos_4_1
				lbl43 := ctx.ReserveLabel()
				_ = lbl43
				bbpos_4_2 := int32(-1)
				_ = bbpos_4_2
				lbl44 := ctx.ReserveLabel()
				_ = lbl44
				bbpos_4_3 := int32(-1)
				_ = bbpos_4_3
				lbl45 := ctx.ReserveLabel()
				_ = lbl45
				bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl42)
				ctx.ResolveFixups()
				d90 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase89) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d87)
				if d87.Loc == LocReg || d87.Loc == LocFPReg {
					ctx.ProtectReg(d87.Reg)
				} else if d87.Loc == LocRegPair {
					ctx.ProtectReg(d87.Reg)
					ctx.ProtectReg(d87.Reg2)
				}
				d91 := d87
				if d91.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d91)
				if d91.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d91, int32(phiBase89)+int32(0), 2)
				} else if d91.Loc == LocInputPair {
					ctx.EnsureDesc(&d91)
					ctx.EmitStoreScmerToStack(d91, int32(phiBase89)+int32(0))
				} else if d91.Loc == LocRegPair || d91.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d91, int32(phiBase89)+int32(0))
				} else {
					ctx.EnsureDesc(&d91)
					ctx.EmitStoreToStack(d91, int32(phiBase89)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase89)+int32(0))+8)
				}
				if d87.Loc == LocReg || d87.Loc == LocFPReg {
					ctx.UnprotectReg(d87.Reg)
				} else if d87.Loc == LocRegPair {
					ctx.UnprotectReg(d87.Reg)
					ctx.UnprotectReg(d87.Reg2)
				}
				bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl45)
				ctx.ResolveFixups()
				d90 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase89) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d92 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase89) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d92)
				ctx.ReclaimUntrackedRegs()
				d92 = JITPrepareScmerGoArg(ctx, d92)
				ctx.SyncDesc(&d92)
				d93 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d92}, 1)
				d93.NoHeapPointer = true
				ctx.EmitAndRegImm32(d93.Reg, 1)
				d93.Type = tagBool
				ctx.BindReg(d93.Reg, &d93)
				ctx.ReclaimUntrackedRegs()
				d94 := d93
				ctx.EnsureDesc(&d94)
				if d94.Loc != LocImm && d94.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl46 := ctx.ReserveLabel()
				lbl47 := ctx.ReserveLabel()
				if d94.Loc == LocImm {
					if d94.Imm.Bool() {
						ctx.MarkLabel(lbl46)
						ctx.EmitJmp(lbl43)
					} else {
						ctx.MarkLabel(lbl47)
						ctx.EmitJmp(lbl44)
					}
				} else {
					ctx.EmitCmpRegImm32(d94.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl46)
					ctx.EmitJmp(lbl47)
					ctx.MarkLabel(lbl46)
					ctx.EmitJmp(lbl43)
					ctx.MarkLabel(lbl47)
					ctx.EmitJmp(lbl44)
				}
				ctx.FreeDesc(&d93)
				bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl44)
				ctx.ResolveFixups()
				d92 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase89) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r26 := ctx.AllocReg()
				r27 := ctx.AllocRegExcept(r26)
				d95 := JITValueDesc{Loc: LocRegPair, Reg: r26, Reg2: r27}
				ctx.BindReg(r26, &d95)
				ctx.BindReg(r27, &d95)
				ctx.EmitMovPairToResult(&d92, &d95)
				ctx.EmitJmp(lbl41)
				bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl43)
				ctx.ResolveFixups()
				d92 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase89) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d92 = JITPrepareScmerGoArg(ctx, d92)
				ctx.SyncDesc(&d92)
				d96 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d92}, 1)
				d96.NoHeapPointer = false
				ctx.BindReg(d96.Reg, &d96)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d97 JITValueDesc
				ctx.EnsureDesc(&d96)
				if d96.Loc == LocImm {
					fieldAddr := uintptr(d96.Imm.Int()) + 32
					r28 := ctx.AllocReg()
					r29 := ctx.AllocRegExcept(r28)
					ctx.EmitMovRegMem64(r28, fieldAddr)
					ctx.EmitMovRegMem64(r29, fieldAddr+8)
					d97 = JITValueDesc{Loc: LocRegPair, Reg: r28, Reg2: r29}
					ctx.BindReg(r28, &d97)
					ctx.BindReg(r29, &d97)
				} else {
					off := int32(32)
					baseReg := d96.Reg
					r30 := ctx.AllocRegExcept(baseReg)
					r31 := ctx.AllocRegExcept(baseReg, r30)
					ctx.EmitMovRegMem(r30, baseReg, off)
					ctx.EmitMovRegMem(r31, baseReg, off+8)
					d97 = JITValueDesc{Loc: LocRegPair, Reg: r30, Reg2: r31}
					ctx.BindReg(r30, &d97)
					ctx.BindReg(r31, &d97)
				}
				ctx.FreeDesc(&d96)
				ctx.StabilizeDescForControlFlow(&d97)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d97)
				if d97.Loc == LocReg || d97.Loc == LocFPReg {
					ctx.ProtectReg(d97.Reg)
				} else if d97.Loc == LocRegPair {
					ctx.ProtectReg(d97.Reg)
					ctx.ProtectReg(d97.Reg2)
				}
				d98 := d97
				if d98.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d98)
				if d98.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d98, int32(phiBase89)+int32(0), 2)
				} else if d98.Loc == LocInputPair {
					ctx.EnsureDesc(&d98)
					ctx.EmitStoreScmerToStack(d98, int32(phiBase89)+int32(0))
				} else if d98.Loc == LocRegPair || d98.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d98, int32(phiBase89)+int32(0))
				} else {
					ctx.EnsureDesc(&d98)
					ctx.EmitStoreToStack(d98, int32(phiBase89)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase89)+int32(0))+8)
				}
				if d97.Loc == LocReg || d97.Loc == LocFPReg {
					ctx.UnprotectReg(d97.Reg)
				} else if d97.Loc == LocRegPair {
					ctx.UnprotectReg(d97.Reg)
					ctx.UnprotectReg(d97.Reg2)
				}
				ctx.EmitJmp(lbl45)
				ctx.MarkLabel(lbl41)
				d99 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r26, Reg2: r27}
				ctx.BindReg(r26, &d99)
				ctx.BindReg(r27, &d99)
				ctx.BindReg(r26, &d99)
				ctx.BindReg(r27, &d99)
				ctx.StabilizeDescForControlFlow(&d99)
				ctx.FreeDesc(&d87)
				ctx.ReclaimUntrackedRegs()
				d101 := d99
				d101.ID = 0
				d100 := ctx.EmitTagEqualsBorrowed(&d101, tagNil, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				d102 := d100
				ctx.EnsureDesc(&d102)
				if d102.Loc != LocImm && d102.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl48 := ctx.ReserveLabel()
				lbl49 := ctx.ReserveLabel()
				if d102.Loc == LocImm {
					if d102.Imm.Bool() {
						ctx.MarkLabel(lbl48)
						ctx.EmitJmp(lbl13)
					} else {
						ctx.MarkLabel(lbl49)
						ctx.SyncDesc(&d99)
						if d99.Loc == LocReg || d99.Loc == LocFPReg {
							ctx.ProtectReg(d99.Reg)
						} else if d99.Loc == LocRegPair {
							ctx.ProtectReg(d99.Reg)
							ctx.ProtectReg(d99.Reg2)
						}
						d103 := d99
						if d103.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d103)
						if d103.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d103, int32(phiBase14)+int32(24), 2)
						} else if d103.Loc == LocInputPair {
							ctx.EnsureDesc(&d103)
							ctx.EmitStoreScmerToStack(d103, int32(phiBase14)+int32(24))
						} else if d103.Loc == LocRegPair || d103.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d103, int32(phiBase14)+int32(24))
						} else {
							ctx.EnsureDesc(&d103)
							ctx.EmitStoreToStack(d103, int32(phiBase14)+int32(24))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase14)+int32(24))+8)
						}
						if d99.Loc == LocReg || d99.Loc == LocFPReg {
							ctx.UnprotectReg(d99.Reg)
						} else if d99.Loc == LocRegPair {
							ctx.UnprotectReg(d99.Reg)
							ctx.UnprotectReg(d99.Reg2)
						}
						ctx.EmitJmp(lbl14)
					}
				} else {
					ctx.EmitCmpRegImm32(d102.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl48)
					ctx.EmitJmp(lbl49)
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl49)
					ctx.SyncDesc(&d99)
					if d99.Loc == LocReg || d99.Loc == LocFPReg {
						ctx.ProtectReg(d99.Reg)
					} else if d99.Loc == LocRegPair {
						ctx.ProtectReg(d99.Reg)
						ctx.ProtectReg(d99.Reg2)
					}
					d104 := d99
					if d104.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d104)
					if d104.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d104, int32(phiBase14)+int32(24), 2)
					} else if d104.Loc == LocInputPair {
						ctx.EnsureDesc(&d104)
						ctx.EmitStoreScmerToStack(d104, int32(phiBase14)+int32(24))
					} else if d104.Loc == LocRegPair || d104.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d104, int32(phiBase14)+int32(24))
					} else {
						ctx.EnsureDesc(&d104)
						ctx.EmitStoreToStack(d104, int32(phiBase14)+int32(24))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase14)+int32(24))+8)
					}
					if d99.Loc == LocReg || d99.Loc == LocFPReg {
						ctx.UnprotectReg(d99.Reg)
					} else if d99.Loc == LocRegPair {
						ctx.UnprotectReg(d99.Reg)
						ctx.UnprotectReg(d99.Reg2)
					}
					ctx.EmitJmp(lbl14)
				}
				ctx.FreeDesc(&d100)
				bbpos_2_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d16 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d105 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				ctx.StabilizeDescForControlFlow(&d105)
				ctx.ReclaimUntrackedRegs()
				var d106 JITValueDesc
				if d13.SliceSizeKnown {
					d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d13.KnownSliceLen))}
				} else if d13.Loc == LocImm {
					d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d13.StackOff))}
				} else if d13.Loc == LocStackTriple {
					d106 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d13.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair || d13.Loc == LocRegTriple {
						d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg2, ID: 0}
					} else if d13.Loc == LocReg {
						d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.StabilizeDescForControlFlow(&d106)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase14)+int32(40))
				bbpos_2_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d105 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d107 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(40)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d107)
				ctx.EnsureDesc(&d107)
				var d108 JITValueDesc
				if d107.Loc == LocImm {
					d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d107.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d107.Reg)
					ctx.EmitMovRegReg(scratch, d107.Reg)
					ctx.EmitAddRegImm32(scratch, int32(1))
					d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d108)
				}
				if d108.Loc == LocReg && d107.Loc == LocReg && d108.Reg == d107.Reg {
					ctx.TransferReg(d107.Reg)
					d107.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d108)
				ctx.FreeDesc(&d107)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d108)
				ctx.EnsureDesc(&d106)
				ctx.EnsureDescsTogether(&d108, &d106)
				var d109 JITValueDesc
				if d108.Loc == LocImm && d106.Loc == LocImm {
					d109 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d108.Imm.Int() < d106.Imm.Int())}
				} else if d106.Loc == LocImm {
					r32 := ctx.AllocRegExcept(d108.Reg)
					if d106.Imm.Int() >= -2147483648 && d106.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d108.Reg, int32(d106.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d106.Imm.Int()))
						ctx.EmitCmpInt64(d108.Reg, RegR11)
					}
					d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r32, Condition: CondSignedLess}
					ctx.BindReg(r32, &d109)
				} else if d108.Loc == LocImm {
					r33 := ctx.AllocReg()
					ctx.EmitMovRegImm64(RegR11, uint64(d108.Imm.Int()))
					ctx.EmitCmpInt64(RegR11, d106.Reg)
					d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r33, Condition: CondSignedLess}
					ctx.BindReg(r33, &d109)
				} else {
					r34 := ctx.AllocRegExcept(d108.Reg)
					ctx.EmitCmpInt64(d108.Reg, d106.Reg)
					d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r34, Condition: CondSignedLess}
					ctx.BindReg(r34, &d109)
				}
				ctx.ReclaimUntrackedRegs()
				d110 := d109
				ctx.EnsureDesc(&d110)
				if d110.Loc != LocImm && d110.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl50 := ctx.ReserveLabel()
				lbl51 := ctx.ReserveLabel()
				if d110.Loc == LocImm {
					if d110.Imm.Bool() {
						ctx.MarkLabel(lbl50)
						ctx.EmitJmp(lbl16)
					} else {
						ctx.MarkLabel(lbl51)
						ctx.SyncDesc(&d49)
						if d49.Loc == LocReg || d49.Loc == LocFPReg {
							ctx.ProtectReg(d49.Reg)
						} else if d49.Loc == LocRegPair {
							ctx.ProtectReg(d49.Reg)
							ctx.ProtectReg(d49.Reg2)
						}
						d111 := d49
						if d111.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d111)
						if d111.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d111, int32(phiBase14)+int32(0), 3)
						} else {
							if d111.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d111.Reg, RegRSP, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d111.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d111.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
						}
						if d49.Loc == LocReg || d49.Loc == LocFPReg {
							ctx.UnprotectReg(d49.Reg)
						} else if d49.Loc == LocRegPair {
							ctx.UnprotectReg(d49.Reg)
							ctx.UnprotectReg(d49.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d110.Condition, lbl50)
					ctx.EmitJmp(lbl51)
					ctx.FreeDesc(&d109)
					ctx.MarkLabel(lbl50)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl51)
					ctx.SyncDesc(&d49)
					if d49.Loc == LocReg || d49.Loc == LocFPReg {
						ctx.ProtectReg(d49.Reg)
					} else if d49.Loc == LocRegPair {
						ctx.ProtectReg(d49.Reg)
						ctx.ProtectReg(d49.Reg2)
					}
					d112 := d49
					if d112.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d112)
					if d112.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d112, int32(phiBase14)+int32(0), 3)
					} else {
						if d112.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d112.Reg, RegRSP, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d112.Reg2, RegRSP, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d112.Reg3, RegRSP, int32(phiBase14)+int32(0)+16)
					}
					if d49.Loc == LocReg || d49.Loc == LocFPReg {
						ctx.UnprotectReg(d49.Reg)
					} else if d49.Loc == LocRegPair {
						ctx.UnprotectReg(d49.Reg)
						ctx.UnprotectReg(d49.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				bbpos_2_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d105 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d107 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d12)
				if d12.Loc == LocReg || d12.Loc == LocFPReg {
					ctx.ProtectReg(d12.Reg)
				} else if d12.Loc == LocRegPair {
					ctx.ProtectReg(d12.Reg)
					ctx.ProtectReg(d12.Reg2)
				}
				d113 := d12
				if d113.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d113)
				if d113.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d113, int32(phiBase14)+int32(24), 2)
				} else if d113.Loc == LocInputPair {
					ctx.EnsureDesc(&d113)
					ctx.EmitStoreScmerToStack(d113, int32(phiBase14)+int32(24))
				} else if d113.Loc == LocRegPair || d113.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d113, int32(phiBase14)+int32(24))
				} else {
					ctx.EnsureDesc(&d113)
					ctx.EmitStoreToStack(d113, int32(phiBase14)+int32(24))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase14)+int32(24))+8)
				}
				if d12.Loc == LocReg || d12.Loc == LocFPReg {
					ctx.UnprotectReg(d12.Reg)
				} else if d12.Loc == LocRegPair {
					ctx.UnprotectReg(d12.Reg)
					ctx.UnprotectReg(d12.Reg2)
				}
				ctx.EmitJmp(lbl14)
				bbpos_2_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d105 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d107 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d108)
				ctx.ReclaimUntrackedRegs()
				d115 := ctx.EmitSliceElementAddress(&d13, &d108, 16)
				ctx.EnsureDesc(&d115)
				r35 := ctx.AllocRegExcept(d115.Reg)
				ctx.EmitMovRegMem(r35, d115.Reg, 8)
				ctx.EmitMovRegMem(d115.Reg, d115.Reg, 0)
				d114 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d115.Reg, Reg2: r35}
				ctx.BindReg(d115.Reg, &d114)
				ctx.BindReg(r35, &d114)
				ctx.ReclaimUntrackedRegs()
				d105 = JITPrepareScmerGoArg(ctx, d105)
				d114 = JITPrepareScmerGoArg(ctx, d114)
				ctx.SyncDesc(&d105)
				ctx.SyncDesc(&d114)
				d116 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d105, d114}, 1)
				d116.NoHeapPointer = true
				ctx.EmitAndRegImm32(d116.Reg, 1)
				d116.Type = tagBool
				ctx.BindReg(d116.Reg, &d116)
				ctx.FreeDesc(&d114)
				ctx.ReclaimUntrackedRegs()
				d117 := d116
				ctx.EnsureDesc(&d117)
				if d117.Loc != LocImm && d117.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl52 := ctx.ReserveLabel()
				lbl53 := ctx.ReserveLabel()
				if d117.Loc == LocImm {
					if d117.Imm.Bool() {
						ctx.MarkLabel(lbl52)
						ctx.EmitJmp(lbl17)
					} else {
						ctx.MarkLabel(lbl53)
						ctx.SyncDesc(&d108)
						if d108.Loc == LocReg || d108.Loc == LocFPReg {
							ctx.ProtectReg(d108.Reg)
						} else if d108.Loc == LocRegPair {
							ctx.ProtectReg(d108.Reg)
							ctx.ProtectReg(d108.Reg2)
						}
						d118 := d108
						if d118.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d118)
						ctx.EmitStoreToStack(d118, int32(phiBase14)+int32(40))
						if d108.Loc == LocReg || d108.Loc == LocFPReg {
							ctx.UnprotectReg(d108.Reg)
						} else if d108.Loc == LocRegPair {
							ctx.UnprotectReg(d108.Reg)
							ctx.UnprotectReg(d108.Reg2)
						}
						ctx.EmitJmp(lbl15)
					}
				} else {
					ctx.EmitCmpRegImm32(d117.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl52)
					ctx.EmitJmp(lbl53)
					ctx.MarkLabel(lbl52)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl53)
					ctx.SyncDesc(&d108)
					if d108.Loc == LocReg || d108.Loc == LocFPReg {
						ctx.ProtectReg(d108.Reg)
					} else if d108.Loc == LocRegPair {
						ctx.ProtectReg(d108.Reg)
						ctx.ProtectReg(d108.Reg2)
					}
					d119 := d108
					if d119.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d119)
					ctx.EmitStoreToStack(d119, int32(phiBase14)+int32(40))
					if d108.Loc == LocReg || d108.Loc == LocFPReg {
						ctx.UnprotectReg(d108.Reg)
					} else if d108.Loc == LocRegPair {
						ctx.UnprotectReg(d108.Reg)
						ctx.UnprotectReg(d108.Reg2)
					}
					ctx.EmitJmp(lbl15)
				}
				ctx.FreeDesc(&d116)
				bbpos_2_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
				d25 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(0)}
				d105 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(24)}
				d107 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(40)}
				d62 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase14) + int32(56)}
				d63 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase14) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d120 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				ctx.EnsureDesc(&d120)
				if d120.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r2, d120)
				}
				ctx.EmitJmp(lbl1)
				ctx.MarkLabel(lbl1)
				d121 := JITValueDesc{Loc: LocReg, Reg: r2}
				ctx.BindReg(r2, &d121)
				ctx.BindReg(r2, &d121)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				ctx.FreeDesc(&d4)
				ctx.FreeDesc(&d6)
				ctx.EnsureDesc(&d121)
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d121.Loc == LocImm {
					ctx.EmitMakeBool(result, d121)
				} else {
					ctx.EmitMakeBool(result, d121)
					ctx.FreeReg(d121.Reg)
				}
				result.Type = tagBool
				return result
				return result
			},
			JITInlineCost: 103,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc")
			key := PrepareSerialProc(a[1])
			reduce := PrepareSerialProc(a[2])
			var keyArgs [1]Scmer
			var reduceArgs [2]Scmer
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				keyArgs[0] = item
				result.ReduceValue(key.Call(keyArgs[:]), item, a[3], &reduce, reduceArgs[:])
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
				var stackArray10 int32
				var stackArray11 int32
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
						ctx.FlushRegisterMoves()
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
						optimizedCallback6 := jitPrepareCallback(d4.Imm)
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.StabilizeDescForControlFlow(&d5)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := jitPrepareCallback(d7.Imm)
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						if d7.Loc == LocInputPair && int(d7.StackOff) < ctx.InputArgCount {
							d8 = ctx.RequestOptimizedCallback(int(d7.StackOff))
						} else {
							d8 = jitCopyScmerToPair(ctx, d7)
						}
					}
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d8)
					ctx.StabilizeDescForControlFlow(&d8)
					stackArray10 = ctx.AllocStack(int32(16))
					_ = stackArray10
					stackArray11 = ctx.AllocStack(int32(32))
					_ = stackArray11
					var d12 JITValueDesc
					if d3.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d12 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
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
						r0 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d14)
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
						ctx.FreeDesc(&d14)
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
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d16)
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
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d17 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d17)
					ctx.BindReg(r1, &d17)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d17}, 1)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					var d19 JITValueDesc
					if d3.SliceSizeKnown {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d19)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps20 := PhiState{General: ps.General}
					ps20.OverlayValues = make([]JITValueDesc, 20)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[8] = d8
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
							ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(0))
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
						ctx.FlushRegisterMoves()
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
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d23 JITValueDesc
					if d1.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d23)
					}
					if d23.Loc == LocReg && d1.Loc == LocReg && d23.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d23, &d19)
					var d24 JITValueDesc
					if d23.Loc == LocImm && d19.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d23.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, RegR11)
						}
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d24)
					} else if d23.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d24)
					} else {
						r4 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d24)
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
							ps26.OverlayValues[1] = d1
							ps26.OverlayValues[2] = d2
							ps26.OverlayValues[3] = d3
							ps26.OverlayValues[4] = d4
							ps26.OverlayValues[5] = d5
							ps26.OverlayValues[7] = d7
							ps26.OverlayValues[8] = d8
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
						ps27.OverlayValues[1] = d1
						ps27.OverlayValues[2] = d2
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[7] = d7
						ps27.OverlayValues[8] = d8
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
							ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d25.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d24)
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d7
					snap35 := d8
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
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
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
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
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
					ps51.OverlayValues[1] = d1
					ps51.OverlayValues[2] = d2
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[7] = d7
					ps51.OverlayValues[8] = d8
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
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[8] = d8
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
					snap53 := d1
					snap54 := d2
					snap55 := d3
					snap56 := d4
					snap57 := d5
					snap58 := d7
					snap59 := d8
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
					d1 = snap53
					d2 = snap54
					d3 = snap55
					d4 = snap56
					d5 = snap57
					d7 = snap58
					d8 = snap59
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
						ctx.FlushRegisterMoves()
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
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d23)
					d76 = ctx.EmitSliceElementAddress(&d3, &d23, 16)
					ctx.EnsureDesc(&d76)
					r5 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMem(r5, d76.Reg, 8)
					ctx.EmitMovRegMem(d76.Reg, d76.Reg, 0)
					d75 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d76.Reg, Reg2: r5}
					ctx.BindReg(d76.Reg, &d75)
					ctx.BindReg(r5, &d75)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray10)+int32(0))
					d77 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d77
					callbackArgs79 := make([]JITValueDesc, 1)
					callbackArgs79[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					var d78 JITValueDesc
					callbackResultOff80 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff80))
					ctx.FreeDesc(&d77)
					ctx.StabilizeDescAcrossNestedCall(&d23)
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
							d78 = jitEmitDynamicCallableAt(ctx, d85, callbackArgs79, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff80), ID: 0})
						}
					}
					d86 = args[3]
					d86.ID = 0
					d87 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d87
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d78 = JITPrepareScmerGoArg(ctx, d78)
					d75 = JITPrepareScmerGoArg(ctx, d75)
					d86 = JITPrepareScmerGoArg(ctx, d86)
					if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d87 = JITPrepareGoSliceArg(ctx, d87)
					if d87.Loc != LocRegTriple && d87.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((*FastDict).ReduceValue arg5)")
					}
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d78)
					ctx.SyncDesc(&d75)
					ctx.SyncDesc(&d86)
					ctx.SyncDesc(&d8)
					ctx.SyncDesc(&d87)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).ReduceValue), []JITValueDesc{d18, d78, d75, d86, d8, d87})
					ctx.FreeDesc(&d78)
					ctx.FreeDesc(&d75)
					ctx.FreeDesc(&d86)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
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
						ctx.EmitStoreToStack(d88, int32(bbs[1].PhiBase)+int32(0))
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
						}
					}
					ps89 := PhiState{General: ps.General}
					ps89.OverlayValues = make([]JITValueDesc, 89)
					ps89.OverlayValues[1] = d1
					ps89.OverlayValues[2] = d2
					ps89.OverlayValues[3] = d3
					ps89.OverlayValues[4] = d4
					ps89.OverlayValues[5] = d5
					ps89.OverlayValues[7] = d7
					ps89.OverlayValues[8] = d8
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
					ps89.OverlayValues[77] = d77
					ps89.OverlayValues[78] = d78
					ps89.OverlayValues[83] = d83
					ps89.OverlayValues[85] = d85
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					ctx.ReclaimUntrackedRegs()
					var d91 JITValueDesc
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d91 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r6}
						ctx.BindReg(d18.Reg, &d91)
						ctx.BindReg(r6, &d91)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d91)
						ctx.BindReg(r6, &d91)
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
			JITInlineCost:      41,
			JITVirtualArgs:     true,
		},
		Optimize: optimizeGroupAssoc,
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_append",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_append")
			key := PrepareSerialProc(a[1])
			value := PrepareSerialProc(a[2])
			var keyArgs [1]Scmer
			var valueArgs [2]Scmer
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				keyArgs[0] = item
				valueArgs[0], valueArgs[1] = NewNil(), item
				result.AppendValue(key.Call(keyArgs[:]), value.Call(valueArgs[:]))
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
				var stackArray10 int32
				var stackArray11 int32
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
						ctx.FlushRegisterMoves()
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
						optimizedCallback6 := jitPrepareCallback(d4.Imm)
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.StabilizeDescForControlFlow(&d5)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := jitPrepareCallback(d7.Imm)
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						if d7.Loc == LocInputPair && int(d7.StackOff) < ctx.InputArgCount {
							d8 = ctx.RequestOptimizedCallback(int(d7.StackOff))
						} else {
							d8 = jitCopyScmerToPair(ctx, d7)
						}
					}
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d8)
					ctx.StabilizeDescForControlFlow(&d8)
					stackArray10 = ctx.AllocStack(int32(16))
					_ = stackArray10
					stackArray11 = ctx.AllocStack(int32(32))
					_ = stackArray11
					var d12 JITValueDesc
					if d3.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d12 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
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
						r0 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d14)
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
						ctx.FreeDesc(&d14)
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
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d16)
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
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d17 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d17)
					ctx.BindReg(r1, &d17)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d17}, 1)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					var d19 JITValueDesc
					if d3.SliceSizeKnown {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d19)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps20 := PhiState{General: ps.General}
					ps20.OverlayValues = make([]JITValueDesc, 20)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[8] = d8
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
							ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(0))
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
						ctx.FlushRegisterMoves()
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
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d23 JITValueDesc
					if d1.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d23)
					}
					if d23.Loc == LocReg && d1.Loc == LocReg && d23.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d23, &d19)
					var d24 JITValueDesc
					if d23.Loc == LocImm && d19.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d23.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, RegR11)
						}
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d24)
					} else if d23.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d24)
					} else {
						r4 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d24)
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
							ps26.OverlayValues[1] = d1
							ps26.OverlayValues[2] = d2
							ps26.OverlayValues[3] = d3
							ps26.OverlayValues[4] = d4
							ps26.OverlayValues[5] = d5
							ps26.OverlayValues[7] = d7
							ps26.OverlayValues[8] = d8
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
						ps27.OverlayValues[1] = d1
						ps27.OverlayValues[2] = d2
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[7] = d7
						ps27.OverlayValues[8] = d8
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
							ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d25.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d24)
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d7
					snap35 := d8
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
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
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
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
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
					ps51.OverlayValues[1] = d1
					ps51.OverlayValues[2] = d2
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[7] = d7
					ps51.OverlayValues[8] = d8
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
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[8] = d8
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
					snap53 := d1
					snap54 := d2
					snap55 := d3
					snap56 := d4
					snap57 := d5
					snap58 := d7
					snap59 := d8
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
					d1 = snap53
					d2 = snap54
					d3 = snap55
					d4 = snap56
					d5 = snap57
					d7 = snap58
					d8 = snap59
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
						ctx.FlushRegisterMoves()
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
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d23)
					d76 = ctx.EmitSliceElementAddress(&d3, &d23, 16)
					ctx.EnsureDesc(&d76)
					r5 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMem(r5, d76.Reg, 8)
					ctx.EmitMovRegMem(d76.Reg, d76.Reg, 0)
					d75 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d76.Reg, Reg2: r5}
					ctx.BindReg(d76.Reg, &d75)
					ctx.BindReg(r5, &d75)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray10)+int32(0))
					d77 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d77)
					ctx.EmitStoreScmerToStack(d77, int32(stackArray11)+int32(0))
					ctx.FreeDesc(&d77)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d75)
					d78 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d78
					callbackArgs80 := make([]JITValueDesc, 1)
					callbackArgs80[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					var d79 JITValueDesc
					callbackResultOff81 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff81))
					ctx.FreeDesc(&d78)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs82 := ctx.StabilizeCallbackArgs(callbackArgs80)
						ctx.ReclaimUntrackedRegs()
						outerRegs83 := ctx.PreserveOuterRegs()
						d79 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs82, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						ctx.RestoreOuterRegs(outerRegs83)
						ctx.ReclaimUntrackedRegs()
					} else {
						d84, knownBuiltin85 := jitEmitKnownDeclaration(ctx, d5, callbackArgs80, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						if knownBuiltin85 {
							d79 = d84
						} else {
							ctx.Coverage.DynamicCalls++
							d86 := jitCopyScmerToPair(ctx, d5)
							d79 = jitEmitDynamicCallableAt(ctx, d86, callbackArgs80, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff81), ID: 0})
						}
					}
					d87 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d87
					callbackArgs89 := make([]JITValueDesc, 2)
					callbackArgs89[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 0}
					callbackArgs89[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 16}
					var d88 JITValueDesc
					callbackResultOff90 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff90))
					ctx.FreeDesc(&d87)
					ctx.StabilizeDescAcrossNestedCall(&d23)
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
							d88 = jitEmitDynamicCallableAt(ctx, d95, callbackArgs89, int32(stackArray11), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff90), ID: 0})
						}
					}
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d79 = JITPrepareScmerGoArg(ctx, d79)
					d88 = JITPrepareScmerGoArg(ctx, d88)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d79)
					ctx.SyncDesc(&d88)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d79, d88})
					ctx.FreeDesc(&d79)
					ctx.FreeDesc(&d88)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d96 = d23
						if d96.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d96)
						ctx.EmitStoreToStack(d96, int32(bbs[1].PhiBase)+int32(0))
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
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
					ps97.OverlayValues[12] = d12
					ps97.OverlayValues[13] = d13
					ps97.OverlayValues[14] = d14
					ps97.OverlayValues[15] = d15
					ps97.OverlayValues[16] = d16
					ps97.OverlayValues[17] = d17
					ps97.OverlayValues[18] = d18
					ps97.OverlayValues[19] = d19
					ps97.OverlayValues[21] = d21
					ps97.OverlayValues[22] = d22
					ps97.OverlayValues[23] = d23
					ps97.OverlayValues[24] = d24
					ps97.OverlayValues[25] = d25
					ps97.OverlayValues[28] = d28
					ps97.OverlayValues[75] = d75
					ps97.OverlayValues[76] = d76
					ps97.OverlayValues[77] = d77
					ps97.OverlayValues[78] = d78
					ps97.OverlayValues[79] = d79
					ps97.OverlayValues[84] = d84
					ps97.OverlayValues[86] = d86
					ps97.OverlayValues[87] = d87
					ps97.OverlayValues[88] = d88
					ps97.OverlayValues[93] = d93
					ps97.OverlayValues[95] = d95
					ps97.OverlayValues[96] = d96
					ps97.PhiValues = make([]JITValueDesc, 1)
					d98 = d23
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
						ctx.FlushRegisterMoves()
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
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d99 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r6}
						ctx.BindReg(d18.Reg, &d99)
						ctx.BindReg(r6, &d99)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d99)
						ctx.BindReg(r6, &d99)
						d18.Loc = LocNone
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
			JITInlineCost:      45,
			JITVirtualArgs:     true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_append_reduce",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_append_reduce")
			key := PrepareSerialProc(a[1])
			value := PrepareSerialProc(a[2])
			var keyArgs, valueArgs [2]Scmer
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				keyArgs[0], keyArgs[1] = NewNil(), item
				valueArgs[0], valueArgs[1] = NewNil(), item
				result.AppendValue(key.Call(keyArgs[:]), value.Call(valueArgs[:]))
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
				var stackArray10 int32
				var stackArray11 int32
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
				var d78 JITValueDesc
				_ = d78
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
						ctx.FlushRegisterMoves()
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
						optimizedCallback6 := jitPrepareCallback(d4.Imm)
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.StabilizeDescForControlFlow(&d5)
					d7 = args[2]
					d7.ID = 0
					var d8 JITValueDesc
					if d7.Loc == LocLambdaTemplate {
						d8 = d7
					} else if d7.Loc == LocImm {
						optimizedCallback9 := jitPrepareCallback(d7.Imm)
						ctx.TrackImm(optimizedCallback9)
						d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					} else {
						if d7.Loc == LocInputPair && int(d7.StackOff) < ctx.InputArgCount {
							d8 = ctx.RequestOptimizedCallback(int(d7.StackOff))
						} else {
							d8 = jitCopyScmerToPair(ctx, d7)
						}
					}
					ctx.FreeDesc(&d7)
					ctx.EnsureDesc(&d8)
					ctx.StabilizeDescForControlFlow(&d8)
					stackArray10 = ctx.AllocStack(int32(32))
					_ = stackArray10
					stackArray11 = ctx.AllocStack(int32(32))
					_ = stackArray11
					var d12 JITValueDesc
					if d3.SliceSizeKnown {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d12 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d12 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
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
						r0 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d14 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d14)
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
						ctx.FreeDesc(&d14)
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
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d16)
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
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d17 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d17)
					ctx.BindReg(r1, &d17)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d17)
					d18 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d17}, 1)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.FreeDesc(&d17)
					var d19 JITValueDesc
					if d3.SliceSizeKnown {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d19)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps20 := PhiState{General: ps.General}
					ps20.OverlayValues = make([]JITValueDesc, 20)
					ps20.OverlayValues[1] = d1
					ps20.OverlayValues[2] = d2
					ps20.OverlayValues[3] = d3
					ps20.OverlayValues[4] = d4
					ps20.OverlayValues[5] = d5
					ps20.OverlayValues[7] = d7
					ps20.OverlayValues[8] = d8
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
							ctx.EmitStoreToStack(d22, int32(bbs[1].PhiBase)+int32(0))
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
						ctx.FlushRegisterMoves()
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
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d23 JITValueDesc
					if d1.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d23)
					}
					if d23.Loc == LocReg && d1.Loc == LocReg && d23.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d23)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d23, &d19)
					var d24 JITValueDesc
					if d23.Loc == LocImm && d19.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d23.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, RegR11)
						}
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d24)
					} else if d23.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d24)
					} else {
						r4 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d19.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d24)
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
							ps26.OverlayValues[1] = d1
							ps26.OverlayValues[2] = d2
							ps26.OverlayValues[3] = d3
							ps26.OverlayValues[4] = d4
							ps26.OverlayValues[5] = d5
							ps26.OverlayValues[7] = d7
							ps26.OverlayValues[8] = d8
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
						ps27.OverlayValues[1] = d1
						ps27.OverlayValues[2] = d2
						ps27.OverlayValues[3] = d3
						ps27.OverlayValues[4] = d4
						ps27.OverlayValues[5] = d5
						ps27.OverlayValues[7] = d7
						ps27.OverlayValues[8] = d8
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
							ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d25.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d24)
					snap29 := d1
					snap30 := d2
					snap31 := d3
					snap32 := d4
					snap33 := d5
					snap34 := d7
					snap35 := d8
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
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
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
					d1 = snap29
					d2 = snap30
					d3 = snap31
					d4 = snap32
					d5 = snap33
					d7 = snap34
					d8 = snap35
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
					ps51.OverlayValues[1] = d1
					ps51.OverlayValues[2] = d2
					ps51.OverlayValues[3] = d3
					ps51.OverlayValues[4] = d4
					ps51.OverlayValues[5] = d5
					ps51.OverlayValues[7] = d7
					ps51.OverlayValues[8] = d8
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
					ps52.OverlayValues[1] = d1
					ps52.OverlayValues[2] = d2
					ps52.OverlayValues[3] = d3
					ps52.OverlayValues[4] = d4
					ps52.OverlayValues[5] = d5
					ps52.OverlayValues[7] = d7
					ps52.OverlayValues[8] = d8
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
					snap53 := d1
					snap54 := d2
					snap55 := d3
					snap56 := d4
					snap57 := d5
					snap58 := d7
					snap59 := d8
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
					d1 = snap53
					d2 = snap54
					d3 = snap55
					d4 = snap56
					d5 = snap57
					d7 = snap58
					d8 = snap59
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
						ctx.FlushRegisterMoves()
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
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d23)
					d76 = ctx.EmitSliceElementAddress(&d3, &d23, 16)
					ctx.EnsureDesc(&d76)
					r5 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegMem(r5, d76.Reg, 8)
					ctx.EmitMovRegMem(d76.Reg, d76.Reg, 0)
					d75 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d76.Reg, Reg2: r5}
					ctx.BindReg(d76.Reg, &d75)
					ctx.BindReg(r5, &d75)
					d77 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d77)
					ctx.EmitStoreScmerToStack(d77, int32(stackArray10)+int32(0))
					ctx.FreeDesc(&d77)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray10)+int32(16))
					d78 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d78)
					ctx.EmitStoreScmerToStack(d78, int32(stackArray11)+int32(0))
					ctx.FreeDesc(&d78)
					ctx.SyncDesc(&d75)
					ctx.EmitStoreScmerToStack(d75, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d75)
					d79 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d79
					callbackArgs81 := make([]JITValueDesc, 2)
					callbackArgs81[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					callbackArgs81[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 16}
					var d80 JITValueDesc
					callbackResultOff82 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff82))
					ctx.FreeDesc(&d79)
					ctx.StabilizeDescAcrossNestedCall(&d23)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs83 := ctx.StabilizeCallbackArgs(callbackArgs81)
						ctx.ReclaimUntrackedRegs()
						outerRegs84 := ctx.PreserveOuterRegs()
						d80 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs83, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff82), ID: 0})
						ctx.RestoreOuterRegs(outerRegs84)
						ctx.ReclaimUntrackedRegs()
					} else {
						d85, knownBuiltin86 := jitEmitKnownDeclaration(ctx, d5, callbackArgs81, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff82), ID: 0})
						if knownBuiltin86 {
							d80 = d85
						} else {
							ctx.Coverage.DynamicCalls++
							d87 := jitCopyScmerToPair(ctx, d5)
							d80 = jitEmitDynamicCallableAt(ctx, d87, callbackArgs81, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff82), ID: 0})
						}
					}
					d88 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d88
					callbackArgs90 := make([]JITValueDesc, 2)
					callbackArgs90[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 0}
					callbackArgs90[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 16}
					var d89 JITValueDesc
					callbackResultOff91 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff91))
					ctx.FreeDesc(&d88)
					ctx.StabilizeDescAcrossNestedCall(&d23)
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
							d89 = jitEmitDynamicCallableAt(ctx, d96, callbackArgs90, int32(stackArray11), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff91), ID: 0})
						}
					}
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d80 = JITPrepareScmerGoArg(ctx, d80)
					d89 = JITPrepareScmerGoArg(ctx, d89)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d80)
					ctx.SyncDesc(&d89)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d80, d89})
					ctx.FreeDesc(&d80)
					ctx.FreeDesc(&d89)
					if ps.General {
						ctx.SyncDesc(&d23)
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
							ctx.ProtectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.ProtectReg(d23.Reg)
							ctx.ProtectReg(d23.Reg2)
						}
						d97 = d23
						if d97.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d97)
						ctx.EmitStoreToStack(d97, int32(bbs[1].PhiBase)+int32(0))
						if d23.Loc == LocReg || d23.Loc == LocFPReg {
							ctx.UnprotectReg(d23.Reg)
						} else if d23.Loc == LocRegPair {
							ctx.UnprotectReg(d23.Reg)
							ctx.UnprotectReg(d23.Reg2)
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
					ps98.OverlayValues[12] = d12
					ps98.OverlayValues[13] = d13
					ps98.OverlayValues[14] = d14
					ps98.OverlayValues[15] = d15
					ps98.OverlayValues[16] = d16
					ps98.OverlayValues[17] = d17
					ps98.OverlayValues[18] = d18
					ps98.OverlayValues[19] = d19
					ps98.OverlayValues[21] = d21
					ps98.OverlayValues[22] = d22
					ps98.OverlayValues[23] = d23
					ps98.OverlayValues[24] = d24
					ps98.OverlayValues[25] = d25
					ps98.OverlayValues[28] = d28
					ps98.OverlayValues[75] = d75
					ps98.OverlayValues[76] = d76
					ps98.OverlayValues[77] = d77
					ps98.OverlayValues[78] = d78
					ps98.OverlayValues[79] = d79
					ps98.OverlayValues[80] = d80
					ps98.OverlayValues[85] = d85
					ps98.OverlayValues[87] = d87
					ps98.OverlayValues[88] = d88
					ps98.OverlayValues[89] = d89
					ps98.OverlayValues[94] = d94
					ps98.OverlayValues[96] = d96
					ps98.OverlayValues[97] = d97
					ps98.PhiValues = make([]JITValueDesc, 1)
					d99 = d23
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != LocNone {
						d78 = ps.OverlayValues[78]
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
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d100 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r6}
						ctx.BindReg(d18.Reg, &d100)
						ctx.BindReg(r6, &d100)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d100)
						ctx.BindReg(r6, &d100)
						d18.Loc = LocNone
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
			JITInlineCost:      48,
			JITVirtualArgs:     true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_multi_append_reduce",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_multi_append_reduce")
			legs := (len(a) - 1) / 2
			keys := make([]SerialProc, legs)
			values := make([]SerialProc, legs)
			for i := 0; i < legs; i++ {
				keys[i] = PrepareSerialProc(a[1+2*i])
				values[i] = PrepareSerialProc(a[2+2*i])
			}
			result := NewFastDictValue(groupAssocCapacity(len(input) * legs))
			var args [2]Scmer
			for _, item := range input {
				args[0], args[1] = NewNil(), item
				for i := 0; i < legs; i++ {
					result.AppendValue(keys[i].Call(args[:]), values[i].Call(args[:]))
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
			keys := make([]SerialProc, legs)
			for i := 0; i < legs; i++ {
				keys[i] = PrepareSerialProc(a[1+i])
			}
			result := NewFastDictValue(groupAssocCapacity(len(input) * legs))
			var args [2]Scmer
			for _, item := range input {
				args[0], args[1] = NewNil(), item
				for i := 0; i < legs; i++ {
					result.IncrementCount(keys[i].Call(args[:]))
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
			key := PrepareSerialProc(a[1])
			var args [1]Scmer
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				args[0] = item
				result.IncrementCount(key.Call(args[:]))
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
				var stackArray7 int32
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
				var d24 JITValueDesc
				_ = d24
				var d67 JITValueDesc
				_ = d67
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
						ctx.FlushRegisterMoves()
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
						optimizedCallback6 := jitPrepareCallback(d4.Imm)
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.StabilizeDescForControlFlow(&d5)
					stackArray7 = ctx.AllocStack(int32(16))
					_ = stackArray7
					var d8 JITValueDesc
					if d3.SliceSizeKnown {
						d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d8)
					d9 = d8
					_ = d9
					ctx.StabilizeDescForControlFlow(&d9)
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
					ctx.EnsureDesc(&d9)
					var d10 JITValueDesc
					if d9.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d9.Reg)
						ctx.EmitCmpRegImm32(d9.Reg, 32)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.EnsureDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d10)
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
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d12)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d9)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d13 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d13)
					ctx.BindReg(r1, &d13)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d13)
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d13}, 1)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.FreeDesc(&d13)
					var d15 JITValueDesc
					if d3.SliceSizeKnown {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d15)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps16 := PhiState{General: ps.General}
					ps16.OverlayValues = make([]JITValueDesc, 16)
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[2] = d2
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[5] = d5
					ps16.OverlayValues[8] = d8
					ps16.OverlayValues[9] = d9
					ps16.OverlayValues[10] = d10
					ps16.OverlayValues[11] = d11
					ps16.OverlayValues[12] = d12
					ps16.OverlayValues[13] = d13
					ps16.OverlayValues[14] = d14
					ps16.OverlayValues[15] = d15
					ps16.PhiValues = make([]JITValueDesc, 1)
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps16.PhiValues[0] = d17
					if ps16.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps16)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d18 := ps.PhiValues[0]
							ctx.EnsureDesc(&d18)
							ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d19 JITValueDesc
					if d1.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d19)
					}
					if d19.Loc == LocReg && d1.Loc == LocReg && d19.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDescsTogether(&d19, &d15)
					var d20 JITValueDesc
					if d19.Loc == LocImm && d15.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.Int() < d15.Imm.Int())}
					} else if d15.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d19.Reg)
						if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d19.Reg, int32(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
							ctx.EmitCmpInt64(d19.Reg, RegR11)
						}
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d20)
					} else if d19.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d15.Reg)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d20)
					} else {
						r4 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpInt64(d19.Reg, d15.Reg)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d20)
					}
					d21 = d20
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							if ps.General {
							}
							ps22 := PhiState{General: ps.General}
							ps22.OverlayValues = make([]JITValueDesc, 22)
							ps22.OverlayValues[1] = d1
							ps22.OverlayValues[2] = d2
							ps22.OverlayValues[3] = d3
							ps22.OverlayValues[4] = d4
							ps22.OverlayValues[5] = d5
							ps22.OverlayValues[8] = d8
							ps22.OverlayValues[9] = d9
							ps22.OverlayValues[10] = d10
							ps22.OverlayValues[11] = d11
							ps22.OverlayValues[12] = d12
							ps22.OverlayValues[13] = d13
							ps22.OverlayValues[14] = d14
							ps22.OverlayValues[15] = d15
							ps22.OverlayValues[17] = d17
							ps22.OverlayValues[18] = d18
							ps22.OverlayValues[19] = d19
							ps22.OverlayValues[20] = d20
							ps22.OverlayValues[21] = d21
							return bbs[2].RenderPS(ps22)
						}
						if ps.General {
						}
						ps23 := PhiState{General: ps.General}
						ps23.OverlayValues = make([]JITValueDesc, 22)
						ps23.OverlayValues[1] = d1
						ps23.OverlayValues[2] = d2
						ps23.OverlayValues[3] = d3
						ps23.OverlayValues[4] = d4
						ps23.OverlayValues[5] = d5
						ps23.OverlayValues[8] = d8
						ps23.OverlayValues[9] = d9
						ps23.OverlayValues[10] = d10
						ps23.OverlayValues[11] = d11
						ps23.OverlayValues[12] = d12
						ps23.OverlayValues[13] = d13
						ps23.OverlayValues[14] = d14
						ps23.OverlayValues[15] = d15
						ps23.OverlayValues[17] = d17
						ps23.OverlayValues[18] = d18
						ps23.OverlayValues[19] = d19
						ps23.OverlayValues[20] = d20
						ps23.OverlayValues[21] = d21
						return bbs[3].RenderPS(ps23)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d24 := ps.PhiValues[0]
							ctx.EnsureDesc(&d24)
							ctx.EmitStoreToStack(d24, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d21.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d20)
					snap25 := d1
					snap26 := d2
					snap27 := d3
					snap28 := d4
					snap29 := d5
					snap30 := d8
					snap31 := d9
					snap32 := d10
					snap33 := d11
					snap34 := d12
					snap35 := d13
					snap36 := d14
					snap37 := d15
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d21
					snap43 := d24
					alloc44 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc44)
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d24 = snap43
					ctx.RestoreAllocState(alloc44)
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d24 = snap43
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 25)
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[8] = d8
					ps45.OverlayValues[9] = d9
					ps45.OverlayValues[10] = d10
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[12] = d12
					ps45.OverlayValues[13] = d13
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[15] = d15
					ps45.OverlayValues[17] = d17
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[19] = d19
					ps45.OverlayValues[20] = d20
					ps45.OverlayValues[21] = d21
					ps45.OverlayValues[24] = d24
					ps46 := PhiState{General: true}
					ps46.OverlayValues = make([]JITValueDesc, 25)
					ps46.OverlayValues[1] = d1
					ps46.OverlayValues[2] = d2
					ps46.OverlayValues[3] = d3
					ps46.OverlayValues[4] = d4
					ps46.OverlayValues[5] = d5
					ps46.OverlayValues[8] = d8
					ps46.OverlayValues[9] = d9
					ps46.OverlayValues[10] = d10
					ps46.OverlayValues[11] = d11
					ps46.OverlayValues[12] = d12
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[15] = d15
					ps46.OverlayValues[17] = d17
					ps46.OverlayValues[18] = d18
					ps46.OverlayValues[19] = d19
					ps46.OverlayValues[20] = d20
					ps46.OverlayValues[21] = d21
					ps46.OverlayValues[24] = d24
					snap47 := d1
					snap48 := d2
					snap49 := d3
					snap50 := d4
					snap51 := d5
					snap52 := d8
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d12
					snap57 := d13
					snap58 := d14
					snap59 := d15
					snap60 := d17
					snap61 := d18
					snap62 := d19
					snap63 := d20
					snap64 := d21
					snap65 := d24
					alloc66 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps46)
					}
					ctx.RestoreAllocState(alloc66)
					d1 = snap47
					d2 = snap48
					d3 = snap49
					d4 = snap50
					d5 = snap51
					d8 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d12 = snap56
					d13 = snap57
					d14 = snap58
					d15 = snap59
					d17 = snap60
					d18 = snap61
					d19 = snap62
					d20 = snap63
					d21 = snap64
					d24 = snap65
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps45)
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.EnsureDesc(&d19)
					d68 = ctx.EmitSliceElementAddress(&d3, &d19, 16)
					ctx.EnsureDesc(&d68)
					r5 := ctx.AllocRegExcept(d68.Reg)
					ctx.EmitMovRegMem(r5, d68.Reg, 8)
					ctx.EmitMovRegMem(d68.Reg, d68.Reg, 0)
					d67 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d68.Reg, Reg2: r5}
					ctx.BindReg(d68.Reg, &d67)
					ctx.BindReg(r5, &d67)
					ctx.SyncDesc(&d67)
					ctx.EmitStoreScmerToStack(d67, int32(stackArray7)+int32(0))
					ctx.FreeDesc(&d67)
					d69 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d69
					callbackArgs71 := make([]JITValueDesc, 1)
					callbackArgs71[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 0}
					var d70 JITValueDesc
					callbackResultOff72 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff72))
					ctx.FreeDesc(&d69)
					ctx.StabilizeDescAcrossNestedCall(&d19)
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
							d70 = jitEmitDynamicCallableAt(ctx, d77, callbackArgs71, int32(stackArray7), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff72), ID: 0})
						}
					}
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d70 = JITPrepareScmerGoArg(ctx, d70)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d70)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d14, d70})
					ctx.FreeDesc(&d70)
					if ps.General {
						ctx.SyncDesc(&d19)
						if d19.Loc == LocReg || d19.Loc == LocFPReg {
							ctx.ProtectReg(d19.Reg)
						} else if d19.Loc == LocRegPair {
							ctx.ProtectReg(d19.Reg)
							ctx.ProtectReg(d19.Reg2)
						}
						d78 = d19
						if d78.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d78)
						ctx.EmitStoreToStack(d78, int32(bbs[1].PhiBase)+int32(0))
						if d19.Loc == LocReg || d19.Loc == LocFPReg {
							ctx.UnprotectReg(d19.Reg)
						} else if d19.Loc == LocRegPair {
							ctx.UnprotectReg(d19.Reg)
							ctx.UnprotectReg(d19.Reg2)
						}
					}
					ps79 := PhiState{General: ps.General}
					ps79.OverlayValues = make([]JITValueDesc, 79)
					ps79.OverlayValues[1] = d1
					ps79.OverlayValues[2] = d2
					ps79.OverlayValues[3] = d3
					ps79.OverlayValues[4] = d4
					ps79.OverlayValues[5] = d5
					ps79.OverlayValues[8] = d8
					ps79.OverlayValues[9] = d9
					ps79.OverlayValues[10] = d10
					ps79.OverlayValues[11] = d11
					ps79.OverlayValues[12] = d12
					ps79.OverlayValues[13] = d13
					ps79.OverlayValues[14] = d14
					ps79.OverlayValues[15] = d15
					ps79.OverlayValues[17] = d17
					ps79.OverlayValues[18] = d18
					ps79.OverlayValues[19] = d19
					ps79.OverlayValues[20] = d20
					ps79.OverlayValues[21] = d21
					ps79.OverlayValues[24] = d24
					ps79.OverlayValues[67] = d67
					ps79.OverlayValues[68] = d68
					ps79.OverlayValues[69] = d69
					ps79.OverlayValues[70] = d70
					ps79.OverlayValues[75] = d75
					ps79.OverlayValues[77] = d77
					ps79.OverlayValues[78] = d78
					ps79.PhiValues = make([]JITValueDesc, 1)
					d80 = d19
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
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
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d81 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d14.Reg, Reg2: r6}
						ctx.BindReg(d14.Reg, &d81)
						ctx.BindReg(r6, &d81)
						ctx.TransferReg(d14.Reg)
						ctx.BindReg(d14.Reg, &d81)
						ctx.BindReg(r6, &d81)
						d14.Loc = LocNone
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
			JITInlineCost:      32,
			JITVirtualArgs:     true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "group_assoc_count_reduce",
		Fn: func(a ...Scmer) Scmer {
			input := asSlice(a[0], "group_assoc_count_reduce")
			key := PrepareSerialProc(a[1])
			var args [2]Scmer
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				args[0], args[1] = NewNil(), item
				result.IncrementCount(key.Call(args[:]))
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
				var stackArray7 int32
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
				var d24 JITValueDesc
				_ = d24
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
						ctx.FlushRegisterMoves()
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
						optimizedCallback6 := jitPrepareCallback(d4.Imm)
						ctx.TrackImm(optimizedCallback6)
						d5 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback6, Rooted: true}
					} else {
						if d4.Loc == LocInputPair && int(d4.StackOff) < ctx.InputArgCount {
							d5 = ctx.RequestOptimizedCallback(int(d4.StackOff))
						} else {
							d5 = jitCopyScmerToPair(ctx, d4)
						}
					}
					ctx.FreeDesc(&d4)
					ctx.EnsureDesc(&d5)
					ctx.StabilizeDescForControlFlow(&d5)
					stackArray7 = ctx.AllocStack(int32(32))
					_ = stackArray7
					var d8 JITValueDesc
					if d3.SliceSizeKnown {
						d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d8 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d8)
					d9 = d8
					_ = d9
					ctx.StabilizeDescForControlFlow(&d9)
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
					ctx.EnsureDesc(&d9)
					var d10 JITValueDesc
					if d9.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d9.Reg)
						ctx.EmitCmpRegImm32(d9.Reg, 32)
						d10 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d10)
					}
					ctx.ReclaimUntrackedRegs()
					d11 = d10
					ctx.EnsureDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d11.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d10)
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
					d12 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d12)
					if d12.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d12)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d9)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d13 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d13)
					ctx.BindReg(r1, &d13)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d13)
					d14 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d13}, 1)
					ctx.StabilizeDescForControlFlow(&d14)
					ctx.FreeDesc(&d13)
					var d15 JITValueDesc
					if d3.SliceSizeKnown {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d15 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d15)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					}
					ps16 := PhiState{General: ps.General}
					ps16.OverlayValues = make([]JITValueDesc, 16)
					ps16.OverlayValues[1] = d1
					ps16.OverlayValues[2] = d2
					ps16.OverlayValues[3] = d3
					ps16.OverlayValues[4] = d4
					ps16.OverlayValues[5] = d5
					ps16.OverlayValues[8] = d8
					ps16.OverlayValues[9] = d9
					ps16.OverlayValues[10] = d10
					ps16.OverlayValues[11] = d11
					ps16.OverlayValues[12] = d12
					ps16.OverlayValues[13] = d13
					ps16.OverlayValues[14] = d14
					ps16.OverlayValues[15] = d15
					ps16.PhiValues = make([]JITValueDesc, 1)
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps16.PhiValues[0] = d17
					if ps16.General && bbs[1].Rendered {
						ctx.EmitJmp(lbl2)
						return result
					}
					return bbs[1].RenderPS(ps16)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d18 := ps.PhiValues[0]
							ctx.EnsureDesc(&d18)
							ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
						d15 = ps.OverlayValues[15]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d19 JITValueDesc
					if d1.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d19 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d19)
					}
					if d19.Loc == LocReg && d1.Loc == LocReg && d19.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDescsTogether(&d19, &d15)
					var d20 JITValueDesc
					if d19.Loc == LocImm && d15.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d19.Imm.Int() < d15.Imm.Int())}
					} else if d15.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d19.Reg)
						if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d19.Reg, int32(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d15.Imm.Int()))
							ctx.EmitCmpInt64(d19.Reg, RegR11)
						}
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d20)
					} else if d19.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d19.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d15.Reg)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d20)
					} else {
						r4 := ctx.AllocRegExcept(d19.Reg)
						ctx.EmitCmpInt64(d19.Reg, d15.Reg)
						d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d20)
					}
					d21 = d20
					ctx.EnsureDesc(&d21)
					if d21.Loc != LocImm && d21.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d21.Loc == LocImm {
						if d21.Imm.Bool() {
							if ps.General {
							}
							ps22 := PhiState{General: ps.General}
							ps22.OverlayValues = make([]JITValueDesc, 22)
							ps22.OverlayValues[1] = d1
							ps22.OverlayValues[2] = d2
							ps22.OverlayValues[3] = d3
							ps22.OverlayValues[4] = d4
							ps22.OverlayValues[5] = d5
							ps22.OverlayValues[8] = d8
							ps22.OverlayValues[9] = d9
							ps22.OverlayValues[10] = d10
							ps22.OverlayValues[11] = d11
							ps22.OverlayValues[12] = d12
							ps22.OverlayValues[13] = d13
							ps22.OverlayValues[14] = d14
							ps22.OverlayValues[15] = d15
							ps22.OverlayValues[17] = d17
							ps22.OverlayValues[18] = d18
							ps22.OverlayValues[19] = d19
							ps22.OverlayValues[20] = d20
							ps22.OverlayValues[21] = d21
							return bbs[2].RenderPS(ps22)
						}
						if ps.General {
						}
						ps23 := PhiState{General: ps.General}
						ps23.OverlayValues = make([]JITValueDesc, 22)
						ps23.OverlayValues[1] = d1
						ps23.OverlayValues[2] = d2
						ps23.OverlayValues[3] = d3
						ps23.OverlayValues[4] = d4
						ps23.OverlayValues[5] = d5
						ps23.OverlayValues[8] = d8
						ps23.OverlayValues[9] = d9
						ps23.OverlayValues[10] = d10
						ps23.OverlayValues[11] = d11
						ps23.OverlayValues[12] = d12
						ps23.OverlayValues[13] = d13
						ps23.OverlayValues[14] = d14
						ps23.OverlayValues[15] = d15
						ps23.OverlayValues[17] = d17
						ps23.OverlayValues[18] = d18
						ps23.OverlayValues[19] = d19
						ps23.OverlayValues[20] = d20
						ps23.OverlayValues[21] = d21
						return bbs[3].RenderPS(ps23)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d24 := ps.PhiValues[0]
							ctx.EnsureDesc(&d24)
							ctx.EmitStoreToStack(d24, int32(bbs[1].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[1].RenderPS(ps)
					}
					ctx.EmitJump(d21.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d20)
					snap25 := d1
					snap26 := d2
					snap27 := d3
					snap28 := d4
					snap29 := d5
					snap30 := d8
					snap31 := d9
					snap32 := d10
					snap33 := d11
					snap34 := d12
					snap35 := d13
					snap36 := d14
					snap37 := d15
					snap38 := d17
					snap39 := d18
					snap40 := d19
					snap41 := d20
					snap42 := d21
					snap43 := d24
					alloc44 := ctx.SnapshotAllocState()
					ctx.RestoreAllocState(alloc44)
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d24 = snap43
					ctx.RestoreAllocState(alloc44)
					d1 = snap25
					d2 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d8 = snap30
					d9 = snap31
					d10 = snap32
					d11 = snap33
					d12 = snap34
					d13 = snap35
					d14 = snap36
					d15 = snap37
					d17 = snap38
					d18 = snap39
					d19 = snap40
					d20 = snap41
					d21 = snap42
					d24 = snap43
					ps45 := PhiState{General: true}
					ps45.OverlayValues = make([]JITValueDesc, 25)
					ps45.OverlayValues[1] = d1
					ps45.OverlayValues[2] = d2
					ps45.OverlayValues[3] = d3
					ps45.OverlayValues[4] = d4
					ps45.OverlayValues[5] = d5
					ps45.OverlayValues[8] = d8
					ps45.OverlayValues[9] = d9
					ps45.OverlayValues[10] = d10
					ps45.OverlayValues[11] = d11
					ps45.OverlayValues[12] = d12
					ps45.OverlayValues[13] = d13
					ps45.OverlayValues[14] = d14
					ps45.OverlayValues[15] = d15
					ps45.OverlayValues[17] = d17
					ps45.OverlayValues[18] = d18
					ps45.OverlayValues[19] = d19
					ps45.OverlayValues[20] = d20
					ps45.OverlayValues[21] = d21
					ps45.OverlayValues[24] = d24
					ps46 := PhiState{General: true}
					ps46.OverlayValues = make([]JITValueDesc, 25)
					ps46.OverlayValues[1] = d1
					ps46.OverlayValues[2] = d2
					ps46.OverlayValues[3] = d3
					ps46.OverlayValues[4] = d4
					ps46.OverlayValues[5] = d5
					ps46.OverlayValues[8] = d8
					ps46.OverlayValues[9] = d9
					ps46.OverlayValues[10] = d10
					ps46.OverlayValues[11] = d11
					ps46.OverlayValues[12] = d12
					ps46.OverlayValues[13] = d13
					ps46.OverlayValues[14] = d14
					ps46.OverlayValues[15] = d15
					ps46.OverlayValues[17] = d17
					ps46.OverlayValues[18] = d18
					ps46.OverlayValues[19] = d19
					ps46.OverlayValues[20] = d20
					ps46.OverlayValues[21] = d21
					ps46.OverlayValues[24] = d24
					snap47 := d1
					snap48 := d2
					snap49 := d3
					snap50 := d4
					snap51 := d5
					snap52 := d8
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d12
					snap57 := d13
					snap58 := d14
					snap59 := d15
					snap60 := d17
					snap61 := d18
					snap62 := d19
					snap63 := d20
					snap64 := d21
					snap65 := d24
					alloc66 := ctx.SnapshotAllocState()
					if !bbs[3].Rendered {
						bbs[3].RenderPS(ps46)
					}
					ctx.RestoreAllocState(alloc66)
					d1 = snap47
					d2 = snap48
					d3 = snap49
					d4 = snap50
					d5 = snap51
					d8 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d12 = snap56
					d13 = snap57
					d14 = snap58
					d15 = snap59
					d17 = snap60
					d18 = snap61
					d19 = snap62
					d20 = snap63
					d21 = snap64
					d24 = snap65
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps45)
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.EnsureDesc(&d19)
					d68 = ctx.EmitSliceElementAddress(&d3, &d19, 16)
					ctx.EnsureDesc(&d68)
					r5 := ctx.AllocRegExcept(d68.Reg)
					ctx.EmitMovRegMem(r5, d68.Reg, 8)
					ctx.EmitMovRegMem(d68.Reg, d68.Reg, 0)
					d67 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d68.Reg, Reg2: r5}
					ctx.BindReg(d68.Reg, &d67)
					ctx.BindReg(r5, &d67)
					d69 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d69)
					ctx.EmitStoreScmerToStack(d69, int32(stackArray7)+int32(0))
					ctx.FreeDesc(&d69)
					ctx.SyncDesc(&d67)
					ctx.EmitStoreScmerToStack(d67, int32(stackArray7)+int32(16))
					ctx.FreeDesc(&d67)
					d70 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d70
					callbackArgs72 := make([]JITValueDesc, 2)
					callbackArgs72[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 0}
					callbackArgs72[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 16}
					var d71 JITValueDesc
					callbackResultOff73 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff73))
					ctx.FreeDesc(&d70)
					ctx.StabilizeDescAcrossNestedCall(&d19)
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
							d71 = jitEmitDynamicCallableAt(ctx, d78, callbackArgs72, int32(stackArray7), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff73), ID: 0})
						}
					}
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d71 = JITPrepareScmerGoArg(ctx, d71)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d71)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d14, d71})
					ctx.FreeDesc(&d71)
					if ps.General {
						ctx.SyncDesc(&d19)
						if d19.Loc == LocReg || d19.Loc == LocFPReg {
							ctx.ProtectReg(d19.Reg)
						} else if d19.Loc == LocRegPair {
							ctx.ProtectReg(d19.Reg)
							ctx.ProtectReg(d19.Reg2)
						}
						d79 = d19
						if d79.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d79)
						ctx.EmitStoreToStack(d79, int32(bbs[1].PhiBase)+int32(0))
						if d19.Loc == LocReg || d19.Loc == LocFPReg {
							ctx.UnprotectReg(d19.Reg)
						} else if d19.Loc == LocRegPair {
							ctx.UnprotectReg(d19.Reg)
							ctx.UnprotectReg(d19.Reg2)
						}
					}
					ps80 := PhiState{General: ps.General}
					ps80.OverlayValues = make([]JITValueDesc, 80)
					ps80.OverlayValues[1] = d1
					ps80.OverlayValues[2] = d2
					ps80.OverlayValues[3] = d3
					ps80.OverlayValues[4] = d4
					ps80.OverlayValues[5] = d5
					ps80.OverlayValues[8] = d8
					ps80.OverlayValues[9] = d9
					ps80.OverlayValues[10] = d10
					ps80.OverlayValues[11] = d11
					ps80.OverlayValues[12] = d12
					ps80.OverlayValues[13] = d13
					ps80.OverlayValues[14] = d14
					ps80.OverlayValues[15] = d15
					ps80.OverlayValues[17] = d17
					ps80.OverlayValues[18] = d18
					ps80.OverlayValues[19] = d19
					ps80.OverlayValues[20] = d20
					ps80.OverlayValues[21] = d21
					ps80.OverlayValues[24] = d24
					ps80.OverlayValues[67] = d67
					ps80.OverlayValues[68] = d68
					ps80.OverlayValues[69] = d69
					ps80.OverlayValues[70] = d70
					ps80.OverlayValues[71] = d71
					ps80.OverlayValues[76] = d76
					ps80.OverlayValues[78] = d78
					ps80.OverlayValues[79] = d79
					ps80.PhiValues = make([]JITValueDesc, 1)
					d81 = d19
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
						ctx.FlushRegisterMoves()
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
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
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
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d82 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d14.Reg, Reg2: r6}
						ctx.BindReg(d14.Reg, &d82)
						ctx.BindReg(r6, &d82)
						ctx.TransferReg(d14.Reg)
						ctx.BindReg(d14.Reg, &d82)
						ctx.BindReg(r6, &d82)
						d14.Loc = LocNone
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
			JITInlineCost:      35,
			JITVirtualArgs:     true,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "mapkey_assoc",

		Fn: func(a ...Scmer) Scmer {
			fn := PrepareSerialProc(a[1])
			var fnArgs [2]Scmer
			setAssoc := PrepareSerialProc(Globalenv.Vars["set_assoc"])
			var setAssocArgs [3]Scmer
			result := NewSlice(nil)
			if slice, fd := asAssoc(a[0], "mapkey_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					fnArgs[0], fnArgs[1] = slice[i], slice[i+1]
					setAssocArgs[0], setAssocArgs[1], setAssocArgs[2] = result, fn.Call(fnArgs[:2]), slice[i+1]
					result = setAssoc.Call(setAssocArgs[:3])
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool {
					fnArgs[0], fnArgs[1] = k, v
					setAssocArgs[0], setAssocArgs[1], setAssocArgs[2] = result, fn.Call(fnArgs[:2]), v
					result = setAssoc.Call(setAssocArgs[:3])
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
			fn := PrepareSerialProc(a[1])
			var fnArgs [2]Scmer
			setAssoc := PrepareSerialProc(Globalenv.Vars["set_assoc_mut"])
			var setAssocArgs [3]Scmer
			slice, fd := asAssoc(a[0], "mapkey_assoc_mut")
			if fd == nil {
				orig := append([]Scmer{}, slice...)
				result := NewSlice(slice[:0])
				for i := 0; i < len(orig); i += 2 {
					fnArgs[0], fnArgs[1] = orig[i], orig[i+1]
					setAssocArgs[0], setAssocArgs[1], setAssocArgs[2] = result, fn.Call(fnArgs[:2]), orig[i+1]
					result = setAssoc.Call(setAssocArgs[:3])
				}
				return result
			}
			result := NewSlice(nil)
			fd.Iterate(func(k, v Scmer) bool {
				fnArgs[0], fnArgs[1] = k, v
				setAssocArgs[0], setAssocArgs[1], setAssocArgs[2] = result, fn.Call(fnArgs[:2]), v
				result = setAssoc.Call(setAssocArgs[:3])
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
