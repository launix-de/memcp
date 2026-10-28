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
			JITInlineCost: 112,
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
				_ = d9
				d10 := d6
				_ = d10
				ctx.StabilizeDescForControlFlow(&d10)
				d11 := d7
				_ = d11
				ctx.StabilizeDescForControlFlow(&d11)
				d12 := d8
				_ = d12
				ctx.StabilizeDescForControlFlow(&d12)
				phiBase13 := ctx.AllocStack(int32(120))
				d14 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				ctx.PreparePointerStackTarget(int32(phiBase13)+int32(0), 3)
				_ = d14
				d15 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				ctx.PreparePointerStackTarget(int32(phiBase13)+int32(24), 3)
				_ = d15
				d16 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				_ = d16
				d17 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				_ = d17
				d18 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				ctx.PreparePointerStackTarget(int32(phiBase13)+int32(80), 3)
				_ = d18
				d19 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				_ = d19
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
				bbpos_3_17 := int32(-1)
				_ = bbpos_3_17
				lbl19 := ctx.ReserveLabel()
				_ = lbl19
				bbpos_3_18 := int32(-1)
				_ = bbpos_3_18
				lbl20 := ctx.ReserveLabel()
				_ = lbl20
				bbpos_3_19 := int32(-1)
				_ = bbpos_3_19
				lbl21 := ctx.ReserveLabel()
				_ = lbl21
				bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d14 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d20 JITValueDesc
				if d12.Loc == LocImm {
					d20 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d12.Imm.Int() < 0)}
				} else {
					r0 := ctx.AllocRegExcept(d12.Reg)
					ctx.EmitCmpRegImm32(d12.Reg, 0)
					d20 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
					ctx.BindReg(r0, &d20)
				}
				ctx.ReclaimUntrackedRegs()
				d21 := d20
				ctx.SyncDesc(&d21)
				if d21.Loc != LocImm && d21.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl22 := ctx.ReserveLabel()
				lbl23 := ctx.ReserveLabel()
				if d21.Loc == LocImm {
					if d21.Imm.Bool() {
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl4)
					} else {
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitJump(d21.Condition, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.FreeDesc(&d20)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl5)
				}
				bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d14 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
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
				ctx.SyncDesc(&d10)
				ctx.EmitStoreScmerToStack(d10, int32(stackArray22)+int32(0))
				ctx.ReclaimUntrackedRegs()
				stackArray24 := ctx.AllocStack(int32(128))
				_ = stackArray24
				ctx.ReclaimUntrackedRegs()
				d25 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(8), SliceSizeKnown: true}
				_ = d25
				ctx.StabilizeDescForControlFlow(&d25)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d23)
				if d23.Loc == LocReg || d23.Loc == LocFPReg {
					ctx.ProtectReg(d23.Reg)
				} else if d23.Loc == LocRegPair {
					ctx.ProtectReg(d23.Reg)
					ctx.ProtectReg(d23.Reg2)
				}
				ctx.SyncDesc(&d25)
				if d25.Loc == LocReg || d25.Loc == LocFPReg {
					ctx.ProtectReg(d25.Reg)
				} else if d25.Loc == LocRegPair {
					ctx.ProtectReg(d25.Reg)
					ctx.ProtectReg(d25.Reg2)
				}
				d26 := d23
				if d26.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d26)
				if d26.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d26, int32(phiBase13)+int32(0), 3)
				} else {
					if d26.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d26.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
					ctx.EmitStoreRegMem(d26.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
					ctx.EmitStoreRegMem(d26.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
				}
				d27 := d25
				if d27.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d27)
				if d27.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d27, int32(phiBase13)+int32(24), 3)
				} else {
					if d27.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d27.Reg, ctx.StackReg, int32(phiBase13)+int32(24))
					ctx.EmitStoreRegMem(d27.Reg2, ctx.StackReg, int32(phiBase13)+int32(24)+8)
					ctx.EmitStoreRegMem(d27.Reg3, ctx.StackReg, int32(phiBase13)+int32(24)+16)
				}
				if d23.Loc == LocReg || d23.Loc == LocFPReg {
					ctx.UnprotectReg(d23.Reg)
				} else if d23.Loc == LocRegPair {
					ctx.UnprotectReg(d23.Reg)
					ctx.UnprotectReg(d23.Reg2)
				}
				if d25.Loc == LocReg || d25.Loc == LocFPReg {
					ctx.UnprotectReg(d25.Reg)
				} else if d25.Loc == LocRegPair {
					ctx.UnprotectReg(d25.Reg)
					ctx.UnprotectReg(d25.Reg2)
				}
				bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d14 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d15 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d28 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d28)
				ctx.ReclaimUntrackedRegs()
				d29 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				ctx.StabilizeDescForControlFlow(&d29)
				ctx.ReclaimUntrackedRegs()
				var d30 JITValueDesc
				if d28.SliceSizeKnown {
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.KnownSliceLen))}
				} else if d28.Loc == LocImm {
					d30 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.StackOff))}
				} else if d28.Loc == LocStackTriple {
					d30 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocRegTriple {
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg2, ID: 0}
					} else if d28.Loc == LocReg {
						d30 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d30)
				var d31 JITValueDesc
				if d30.Loc == LocImm {
					d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d30.Imm.Int() != 0)}
				} else {
					r1 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d30.Reg, 0)
					d31 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondNotEqual}
					ctx.BindReg(r1, &d31)
				}
				ctx.FreeDesc(&d30)
				ctx.ReclaimUntrackedRegs()
				d32 := d31
				ctx.SyncDesc(&d32)
				if d32.Loc != LocImm && d32.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl24 := ctx.ReserveLabel()
				lbl25 := ctx.ReserveLabel()
				if d32.Loc == LocImm {
					if d32.Imm.Bool() {
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl6)
					} else {
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d32.Condition, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.FreeDesc(&d31)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl7)
				}
				bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d33 JITValueDesc
				if d29.SliceSizeKnown {
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.KnownSliceLen))}
				} else if d29.Loc == LocImm {
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.StackOff))}
				} else if d29.Loc == LocStackTriple {
					d33 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d29.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d29)
					if d29.Loc == LocRegPair || d29.Loc == LocRegTriple {
						d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg2, ID: 0}
					} else if d29.Loc == LocReg {
						d33 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d33)
				ctx.EnsureDesc(&d33)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d33)
				ctx.EnsureDesc(&d33)
				callResults34 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d33, d33}, []uint8{3}, []uint8{1})
				d35 := callResults34[0]
				d35.Type = tagSlice
				ctx.FreeDesc(&d33)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d35)
				ctx.EnsureDesc(&d29)
				callResults36 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d35, d29}, []uint8{1}, []uint8{0})
				d37 := callResults36[0]
				d37.Type = tagInt
				ctx.ReclaimUntrackedRegs()
				d38 := ctx.EmitNewSliceFromGoSlice(&d35)
				ctx.ReclaimUntrackedRegs()
				r2 := ctx.AllocReg()
				r3 := ctx.AllocRegExcept(r2)
				d39 := JITValueDesc{Loc: LocRegPair, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d39)
				ctx.BindReg(r3, &d39)
				ctx.EmitMovPairToResult(&d38, &d39)
				ctx.EmitJmp(lbl2)
				bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d40 JITValueDesc
				if d28.SliceSizeKnown {
					d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.KnownSliceLen))}
				} else if d28.Loc == LocImm {
					d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d28.StackOff))}
				} else if d28.Loc == LocStackTriple {
					d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d28.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d28)
					if d28.Loc == LocRegPair || d28.Loc == LocRegTriple {
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg2, ID: 0}
					} else if d28.Loc == LocReg {
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d28.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d40)
				ctx.EnsureDesc(&d40)
				var d41 JITValueDesc
				if d40.Loc == LocImm {
					d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d40.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d40.Reg)
					ctx.EmitMovRegReg(scratch, d40.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d41)
				}
				if d41.Loc == LocReg && d40.Loc == LocReg && d41.Reg == d40.Reg {
					ctx.TransferReg(d40.Reg)
					d40.Loc = LocNone
				}
				ctx.FreeDesc(&d40)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d41)
				ctx.ReclaimUntrackedRegs()
				d43 := ctx.EmitSliceElementAddress(&d28, &d41, 16)
				ctx.EnsureDesc(&d43)
				r4 := ctx.AllocRegExcept(d43.Reg)
				ctx.EmitMovRegMem(r4, d43.Reg, 8)
				ctx.EmitMovRegMem(d43.Reg, d43.Reg, 0)
				d42 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d43.Reg, Reg2: r4}
				ctx.BindReg(d43.Reg, &d42)
				ctx.BindReg(r4, &d42)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d42)
				phiBase44 := ctx.AllocStack(int32(16))
				d45 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase44) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase44) + int32(0))
				_ = d45
				lbl26 := ctx.ReserveLabel()
				bbpos_4_0 := int32(-1)
				_ = bbpos_4_0
				lbl27 := ctx.ReserveLabel()
				_ = lbl27
				bbpos_4_1 := int32(-1)
				_ = bbpos_4_1
				lbl28 := ctx.ReserveLabel()
				_ = lbl28
				bbpos_4_2 := int32(-1)
				_ = bbpos_4_2
				lbl29 := ctx.ReserveLabel()
				_ = lbl29
				bbpos_4_3 := int32(-1)
				_ = bbpos_4_3
				lbl30 := ctx.ReserveLabel()
				_ = lbl30
				bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl27)
				ctx.ResolveFixups()
				d45 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase44) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d42)
				if d42.Loc == LocReg || d42.Loc == LocFPReg {
					ctx.ProtectReg(d42.Reg)
				} else if d42.Loc == LocRegPair {
					ctx.ProtectReg(d42.Reg)
					ctx.ProtectReg(d42.Reg2)
				}
				d46 := d42
				if d46.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d46)
				if d46.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d46, int32(phiBase44)+int32(0), 2)
				} else if d46.Loc == LocInputPair {
					ctx.EnsureDesc(&d46)
					ctx.EmitStoreScmerToStack(d46, int32(phiBase44)+int32(0))
				} else if d46.Loc == LocRegPair || d46.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d46, int32(phiBase44)+int32(0))
				} else {
					ctx.EnsureDesc(&d46)
					ctx.EmitStoreToStack(d46, int32(phiBase44)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase44)+int32(0))+8)
				}
				if d42.Loc == LocReg || d42.Loc == LocFPReg {
					ctx.UnprotectReg(d42.Reg)
				} else if d42.Loc == LocRegPair {
					ctx.UnprotectReg(d42.Reg)
					ctx.UnprotectReg(d42.Reg2)
				}
				bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl30)
				ctx.ResolveFixups()
				d45 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase44) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d47 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase44) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d47)
				ctx.ReclaimUntrackedRegs()
				d47 = JITPrepareScmerGoArg(ctx, d47)
				ctx.SyncDesc(&d47)
				d48 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d47}, 1)
				d48.NoHeapPointer = true
				ctx.EmitAndRegImm32(d48.Reg, 1)
				d48.Type = tagBool
				ctx.BindReg(d48.Reg, &d48)
				ctx.ReclaimUntrackedRegs()
				d49 := d48
				ctx.EnsureDesc(&d49)
				if d49.Loc != LocImm && d49.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl31 := ctx.ReserveLabel()
				lbl32 := ctx.ReserveLabel()
				if d49.Loc == LocImm {
					if d49.Imm.Bool() {
						ctx.MarkLabel(lbl31)
						ctx.EmitJmp(lbl28)
					} else {
						ctx.MarkLabel(lbl32)
						ctx.EmitJmp(lbl29)
					}
				} else {
					ctx.EmitCmpRegImm32(d49.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl31)
					ctx.EmitJmp(lbl32)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl28)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl29)
				}
				ctx.FreeDesc(&d48)
				bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl29)
				ctx.ResolveFixups()
				d47 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase44) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r5 := ctx.AllocReg()
				r6 := ctx.AllocRegExcept(r5)
				d50 := JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d50)
				ctx.BindReg(r6, &d50)
				ctx.EmitMovPairToResult(&d47, &d50)
				ctx.EmitJmp(lbl26)
				bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl28)
				ctx.ResolveFixups()
				d47 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase44) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d47 = JITPrepareScmerGoArg(ctx, d47)
				ctx.SyncDesc(&d47)
				d51 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d47}, 1)
				d51.NoHeapPointer = false
				ctx.BindReg(d51.Reg, &d51)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d52 JITValueDesc
				ctx.EnsureDesc(&d51)
				if d51.Loc == LocImm {
					fieldAddr := uintptr(d51.Imm.Int()) + 32
					r7 := ctx.AllocReg()
					r8 := ctx.AllocRegExcept(r7)
					ctx.EmitMovRegMem64(r7, fieldAddr)
					ctx.EmitMovRegMem64(r8, fieldAddr+8)
					d52 = JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
					ctx.BindReg(r7, &d52)
					ctx.BindReg(r8, &d52)
				} else {
					off := int32(32)
					baseReg := d51.Reg
					r9 := ctx.AllocRegExcept(baseReg)
					r10 := ctx.AllocRegExcept(baseReg, r9)
					ctx.EmitMovRegMem(r9, baseReg, off)
					ctx.EmitMovRegMem(r10, baseReg, off+8)
					d52 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
					ctx.BindReg(r9, &d52)
					ctx.BindReg(r10, &d52)
				}
				ctx.FreeDesc(&d51)
				ctx.StabilizeDescForControlFlow(&d52)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d52)
				if d52.Loc == LocReg || d52.Loc == LocFPReg {
					ctx.ProtectReg(d52.Reg)
				} else if d52.Loc == LocRegPair {
					ctx.ProtectReg(d52.Reg)
					ctx.ProtectReg(d52.Reg2)
				}
				d53 := d52
				if d53.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d53)
				if d53.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d53, int32(phiBase44)+int32(0), 2)
				} else if d53.Loc == LocInputPair {
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(phiBase44)+int32(0))
				} else if d53.Loc == LocRegPair || d53.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d53, int32(phiBase44)+int32(0))
				} else {
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreToStack(d53, int32(phiBase44)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase44)+int32(0))+8)
				}
				if d52.Loc == LocReg || d52.Loc == LocFPReg {
					ctx.UnprotectReg(d52.Reg)
				} else if d52.Loc == LocRegPair {
					ctx.UnprotectReg(d52.Reg)
					ctx.UnprotectReg(d52.Reg2)
				}
				ctx.EmitJmp(lbl30)
				ctx.MarkLabel(lbl26)
				d54 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d54)
				ctx.BindReg(r6, &d54)
				ctx.BindReg(r5, &d54)
				ctx.BindReg(r6, &d54)
				ctx.StabilizeDescForControlFlow(&d54)
				ctx.FreeDesc(&d42)
				ctx.ReclaimUntrackedRegs()
				d55 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d41)
				ctx.EnsureDesc(&d28)
				ctx.EnsureDesc(&d55)
				ctx.EnsureDesc(&d41)
				var d57 JITValueDesc
				if d41.Loc == LocImm && d55.Loc == LocImm {
					d57 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d41.Imm.Int() - d55.Imm.Int())}
				} else {
					r11 := ctx.AllocReg()
					if d41.Loc == LocImm {
						ctx.EmitMovRegImm64(r11, uint64(d41.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r11, d41.Reg)
					}
					if d55.Loc == LocImm {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d55.Imm.Int()))
						ctx.EmitSubInt64(r11, ctx.ScratchReg)
					} else {
						ctx.EmitSubInt64(r11, d55.Reg)
					}
					d57 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d57)
				}
				var d58 JITValueDesc
				r12 := ctx.EmitSliceDataAfterLow(&d28, &d55, 16)
				d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d58)
				ctx.BindReg(r12, &d58)
				var d59 JITValueDesc
				var r13 Reg
				var r14 Reg
				ctx.SyncDesc(&d58)
				ctx.EnsureDesc(&d58)
				if d58.Loc == LocImm {
					r13 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r13, uint64(d58.Imm.Int()))
				} else {
					r13 = d58.Reg
				}
				ctx.ProtectReg(r13)
				ctx.SyncDesc(&d57)
				ctx.EnsureDesc(&d57)
				if d57.Loc == LocImm {
					r14 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r14, uint64(d57.Imm.Int()))
				} else {
					r14 = d57.Reg
				}
				ctx.ProtectReg(r14)
				r15 := ctx.EmitSliceCapAfterLow(&d28, &d55, r13, r14)
				ctx.UnprotectReg(r14)
				ctx.UnprotectReg(r13)
				d59 = JITValueDesc{Loc: LocRegTriple, Reg: r13, Reg2: r14, Reg3: r15}
				ctx.BindReg(r13, &d59)
				ctx.BindReg(r14, &d59)
				ctx.BindReg(r15, &d59)
				ctx.BindReg(r13, &d59)
				ctx.BindReg(r14, &d59)
				ctx.BindReg(r15, &d59)
				ctx.StabilizeDescForControlFlow(&d59)
				ctx.FreeDesc(&d41)
				ctx.ReclaimUntrackedRegs()
				d61 := d54
				d61.ID = 0
				d60 := ctx.EmitTagEqualsBorrowed(&d61, tagSlice, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				d62 := d60
				ctx.EnsureDesc(&d62)
				if d62.Loc != LocImm && d62.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl33 := ctx.ReserveLabel()
				lbl34 := ctx.ReserveLabel()
				if d62.Loc == LocImm {
					if d62.Imm.Bool() {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl34)
						ctx.SyncDesc(&d59)
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.ProtectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.ProtectReg(d59.Reg)
							ctx.ProtectReg(d59.Reg2)
						}
						d63 := d59
						if d63.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d63)
						if d63.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d63, int32(phiBase13)+int32(0), 3)
						} else {
							if d63.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d63.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
							ctx.EmitStoreRegMem(d63.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
							ctx.EmitStoreRegMem(d63.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
						}
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.UnprotectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.UnprotectReg(d59.Reg)
							ctx.UnprotectReg(d59.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitCmpRegImm32(d62.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl33)
					ctx.EmitJmp(lbl34)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl34)
					ctx.SyncDesc(&d59)
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.ProtectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.ProtectReg(d59.Reg)
						ctx.ProtectReg(d59.Reg2)
					}
					d64 := d59
					if d64.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d64)
					if d64.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d64, int32(phiBase13)+int32(0), 3)
					} else {
						if d64.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d64.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
						ctx.EmitStoreRegMem(d64.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
						ctx.EmitStoreRegMem(d64.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
					}
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.UnprotectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.UnprotectReg(d59.Reg)
						ctx.UnprotectReg(d59.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				ctx.FreeDesc(&d60)
				bbpos_3_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d65 := jitKnownSliceHeader(ctx, &d54)
				ctx.StabilizeDescForControlFlow(&d65)
				ctx.ReclaimUntrackedRegs()
				var d66 JITValueDesc
				if d65.SliceSizeKnown {
					d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d65.KnownSliceLen))}
				} else if d65.Loc == LocImm {
					d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d65.StackOff))}
				} else if d65.Loc == LocStackTriple {
					d66 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d65.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocRegPair || d65.Loc == LocRegTriple {
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg2, ID: 0}
					} else if d65.Loc == LocReg {
						d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d66)
				var d67 JITValueDesc
				if d66.Loc == LocImm {
					d67 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d66.Imm.Int() != 0)}
				} else {
					r16 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d66.Reg, 0)
					d67 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondNotEqual}
					ctx.BindReg(r16, &d67)
				}
				ctx.FreeDesc(&d66)
				ctx.ReclaimUntrackedRegs()
				d68 := d67
				ctx.SyncDesc(&d68)
				if d68.Loc != LocImm && d68.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl35 := ctx.ReserveLabel()
				lbl36 := ctx.ReserveLabel()
				if d68.Loc == LocImm {
					if d68.Imm.Bool() {
						ctx.MarkLabel(lbl35)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl36)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitJump(d68.Condition, lbl35)
					ctx.EmitJmp(lbl36)
					ctx.FreeDesc(&d67)
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl36)
					ctx.EmitJmp(lbl11)
				}
				bbpos_3_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				bbpos_3_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d69 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d69
				ctx.ReclaimUntrackedRegs()
				var d70 JITValueDesc
				if d65.SliceSizeKnown {
					d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d65.KnownSliceLen))}
				} else if d65.Loc == LocImm {
					d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d65.StackOff))}
				} else if d65.Loc == LocStackTriple {
					d70 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d65.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocRegPair || d65.Loc == LocRegTriple {
						d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg2, ID: 0}
					} else if d65.Loc == LocReg {
						d70 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d70)
				ctx.EnsureDesc(&d70)
				var d71 JITValueDesc
				if d70.Loc == LocImm {
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d70.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d70.Reg)
					ctx.EmitMovRegReg(scratch, d70.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d71)
				}
				if d71.Loc == LocReg && d70.Loc == LocReg && d71.Reg == d70.Reg {
					ctx.TransferReg(d70.Reg)
					d70.Loc = LocNone
				}
				ctx.EnsureDesc(&d71)
				ctx.EmitStoreToStack(d71, int32(phiBase13)+int32(104))
				ctx.StabilizeDescForControlFlow(&d71)
				ctx.FreeDesc(&d70)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d59)
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.ProtectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.ProtectReg(d59.Reg)
					ctx.ProtectReg(d59.Reg2)
				}
				d72 := d59
				if d72.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d72)
				if d72.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d72, int32(phiBase13)+int32(80), 3)
				} else {
					if d72.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d72.Reg, ctx.StackReg, int32(phiBase13)+int32(80))
					ctx.EmitStoreRegMem(d72.Reg2, ctx.StackReg, int32(phiBase13)+int32(80)+8)
					ctx.EmitStoreRegMem(d72.Reg3, ctx.StackReg, int32(phiBase13)+int32(80)+16)
				}
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.UnprotectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.UnprotectReg(d59.Reg)
					ctx.UnprotectReg(d59.Reg2)
				}
				bbpos_3_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl20)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d73 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				ctx.StabilizeDescForControlFlow(&d73)
				ctx.ReclaimUntrackedRegs()
				d74 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(104)}
				ctx.StabilizeDescForControlFlow(&d74)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d74)
				d76 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d76
				ctx.EnsureDescsTogether(&d74, &d76)
				var d75 JITValueDesc
				if d74.Loc == LocImm && d76.Loc == LocImm {
					d75 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d74.Imm.Int() >= d76.Imm.Int())}
				} else if d76.Loc == LocImm {
					r17 := ctx.AllocRegExcept(d74.Reg)
					if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d74.Reg, int32(d76.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d76.Imm.Int()))
						ctx.EmitCmpInt64(d74.Reg, ctx.ScratchReg)
					}
					d75 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r17, &d75)
				} else if d74.Loc == LocImm {
					r18 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d74.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d76.Reg)
					d75 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r18, &d75)
				} else {
					r19 := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitCmpInt64(d74.Reg, d76.Reg)
					d75 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r19, &d75)
				}
				ctx.ReclaimUntrackedRegs()
				d77 := d75
				ctx.SyncDesc(&d77)
				if d77.Loc != LocImm && d77.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl37 := ctx.ReserveLabel()
				lbl38 := ctx.ReserveLabel()
				if d77.Loc == LocImm {
					if d77.Imm.Bool() {
						ctx.MarkLabel(lbl37)
						ctx.EmitJmp(lbl21)
					} else {
						ctx.MarkLabel(lbl38)
						ctx.SyncDesc(&d73)
						if d73.Loc == LocReg || d73.Loc == LocFPReg {
							ctx.ProtectReg(d73.Reg)
						} else if d73.Loc == LocRegPair {
							ctx.ProtectReg(d73.Reg)
							ctx.ProtectReg(d73.Reg2)
						}
						d78 := d73
						if d78.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d78)
						if d78.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d78, int32(phiBase13)+int32(0), 3)
						} else {
							if d78.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d78.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
							ctx.EmitStoreRegMem(d78.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
							ctx.EmitStoreRegMem(d78.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
						}
						if d73.Loc == LocReg || d73.Loc == LocFPReg {
							ctx.UnprotectReg(d73.Reg)
						} else if d73.Loc == LocRegPair {
							ctx.UnprotectReg(d73.Reg)
							ctx.UnprotectReg(d73.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitJump(d77.Condition, lbl37)
					ctx.EmitJmp(lbl38)
					ctx.FreeDesc(&d75)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl38)
					ctx.SyncDesc(&d73)
					if d73.Loc == LocReg || d73.Loc == LocFPReg {
						ctx.ProtectReg(d73.Reg)
					} else if d73.Loc == LocRegPair {
						ctx.ProtectReg(d73.Reg)
						ctx.ProtectReg(d73.Reg2)
					}
					d79 := d73
					if d79.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d79)
					if d79.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d79, int32(phiBase13)+int32(0), 3)
					} else {
						if d79.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d79.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
						ctx.EmitStoreRegMem(d79.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
						ctx.EmitStoreRegMem(d79.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
					}
					if d73.Loc == LocReg || d73.Loc == LocFPReg {
						ctx.UnprotectReg(d73.Reg)
					} else if d73.Loc == LocRegPair {
						ctx.UnprotectReg(d73.Reg)
						ctx.UnprotectReg(d73.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				bbpos_3_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d80 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d80
				ctx.ReclaimUntrackedRegs()
				d82 := ctx.EmitSliceElementAddress(&d65, &d80, 16)
				ctx.EnsureDesc(&d82)
				r20 := ctx.AllocRegExcept(d82.Reg)
				ctx.EmitMovRegMem(r20, d82.Reg, 8)
				ctx.EmitMovRegMem(d82.Reg, d82.Reg, 0)
				d81 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d82.Reg, Reg2: r20}
				ctx.BindReg(d82.Reg, &d81)
				ctx.BindReg(r20, &d81)
				ctx.ReclaimUntrackedRegs()
				d81 = JITPrepareScmerGoArg(ctx, d81)
				d11 = JITPrepareScmerGoArg(ctx, d11)
				ctx.SyncDesc(&d81)
				ctx.SyncDesc(&d11)
				d83 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d81, d11}, 1)
				d83.NoHeapPointer = true
				ctx.EmitAndRegImm32(d83.Reg, 1)
				d83.Type = tagBool
				ctx.BindReg(d83.Reg, &d83)
				ctx.FreeDesc(&d81)
				ctx.ReclaimUntrackedRegs()
				d84 := d83
				ctx.EnsureDesc(&d84)
				if d84.Loc != LocImm && d84.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl39 := ctx.ReserveLabel()
				lbl40 := ctx.ReserveLabel()
				if d84.Loc == LocImm {
					if d84.Imm.Bool() {
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl10)
					} else {
						ctx.MarkLabel(lbl40)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitCmpRegImm32(d84.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl39)
					ctx.EmitJmp(lbl40)
					ctx.MarkLabel(lbl39)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl40)
					ctx.EmitJmp(lbl11)
				}
				ctx.FreeDesc(&d83)
				bbpos_3_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl21)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d74)
				ctx.ReclaimUntrackedRegs()
				d86 := ctx.EmitSliceElementAddress(&d65, &d74, 16)
				ctx.EnsureDesc(&d86)
				r21 := ctx.AllocRegExcept(d86.Reg)
				ctx.EmitMovRegMem(r21, d86.Reg, 8)
				ctx.EmitMovRegMem(d86.Reg, d86.Reg, 0)
				d85 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d86.Reg, Reg2: r21}
				ctx.BindReg(d86.Reg, &d85)
				ctx.BindReg(r21, &d85)
				ctx.ReclaimUntrackedRegs()
				stackArray87 := ctx.AllocStack(int32(16))
				_ = stackArray87
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d85)
				ctx.EmitStoreScmerToStack(d85, int32(stackArray87)+int32(0))
				ctx.FreeDesc(&d85)
				ctx.ReclaimUntrackedRegs()
				d88 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d88
				ctx.ReclaimUntrackedRegs()
				r22 := ctx.AllocReg()
				r23 := ctx.AllocRegExcept(r22)
				r24 := ctx.AllocRegExcept(r22, r23)
				d89 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r22, Reg2: r23, Reg3: r24}
				ctx.BindReg(r22, &d89)
				ctx.BindReg(r23, &d89)
				ctx.BindReg(r24, &d89)
				ctx.BindReg(r22, &d89)
				ctx.BindReg(r23, &d89)
				ctx.BindReg(r24, &d89)
				ctx.EmitLeaRegMem(d89.Reg, ctx.StackReg, int32(stackArray87))
				ctx.EmitMovRegImm64(d89.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d89.Reg3, uint64(1))
				callResults90 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d73, d89}, []uint8{3}, []uint8{1})
				d91 := callResults90[0]
				d92 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase13) + int32(80)}
				ctx.EmitCopyDescWords(&d92, &d91, 3)
				ctx.FreeDesc(&d91)
				d91 = d92
				ctx.StabilizeDescForControlFlow(&d91)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d74)
				ctx.EnsureDesc(&d74)
				var d93 JITValueDesc
				if d74.Loc == LocImm {
					d93 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d74.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d74.Reg)
					ctx.EmitMovRegReg(scratch, d74.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d93)
				}
				if d93.Loc == LocReg && d74.Loc == LocReg && d93.Reg == d74.Reg {
					ctx.TransferReg(d74.Reg)
					d74.Loc = LocNone
				}
				ctx.EnsureDesc(&d93)
				ctx.EmitStoreToStack(d93, int32(phiBase13)+int32(104))
				ctx.StabilizeDescForControlFlow(&d93)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl20)
				bbpos_3_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d94 JITValueDesc
				if d65.SliceSizeKnown {
					d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d65.KnownSliceLen))}
				} else if d65.Loc == LocImm {
					d94 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d65.StackOff))}
				} else if d65.Loc == LocStackTriple {
					d94 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d65.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d65)
					if d65.Loc == LocRegPair || d65.Loc == LocRegTriple {
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg2, ID: 0}
					} else if d65.Loc == LocReg {
						d94 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d65.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d94)
				ctx.EnsureDesc(&d12)
				ctx.EnsureDescsTogether(&d94, &d12)
				var d95 JITValueDesc
				if d94.Loc == LocImm && d12.Loc == LocImm {
					d95 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d94.Imm.Int() > d12.Imm.Int())}
				} else if d12.Loc == LocImm {
					r25 := ctx.AllocReg()
					if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d94.Reg, int32(d12.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d12.Imm.Int()))
						ctx.EmitCmpInt64(d94.Reg, ctx.ScratchReg)
					}
					d95 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedGreater}
					ctx.BindReg(r25, &d95)
				} else if d94.Loc == LocImm {
					r26 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d94.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d12.Reg)
					d95 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedGreater}
					ctx.BindReg(r26, &d95)
				} else {
					r27 := ctx.AllocReg()
					ctx.EmitCmpInt64(d94.Reg, d12.Reg)
					d95 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedGreater}
					ctx.BindReg(r27, &d95)
				}
				ctx.FreeDesc(&d94)
				ctx.ReclaimUntrackedRegs()
				d96 := d95
				ctx.SyncDesc(&d96)
				if d96.Loc != LocImm && d96.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl41 := ctx.ReserveLabel()
				lbl42 := ctx.ReserveLabel()
				if d96.Loc == LocImm {
					if d96.Imm.Bool() {
						ctx.MarkLabel(lbl41)
						ctx.EmitJmp(lbl13)
					} else {
						ctx.MarkLabel(lbl42)
						ctx.SyncDesc(&d59)
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.ProtectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.ProtectReg(d59.Reg)
							ctx.ProtectReg(d59.Reg2)
						}
						d97 := d59
						if d97.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d97)
						if d97.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d97, int32(phiBase13)+int32(0), 3)
						} else {
							if d97.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d97.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
							ctx.EmitStoreRegMem(d97.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
							ctx.EmitStoreRegMem(d97.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
						}
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.UnprotectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.UnprotectReg(d59.Reg)
							ctx.UnprotectReg(d59.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitJump(d96.Condition, lbl41)
					ctx.EmitJmp(lbl42)
					ctx.FreeDesc(&d95)
					ctx.MarkLabel(lbl41)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl42)
					ctx.SyncDesc(&d59)
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.ProtectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.ProtectReg(d59.Reg)
						ctx.ProtectReg(d59.Reg2)
					}
					d98 := d59
					if d98.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d98)
					if d98.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d98, int32(phiBase13)+int32(0), 3)
					} else {
						if d98.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d98.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
						ctx.EmitStoreRegMem(d98.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
						ctx.EmitStoreRegMem(d98.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
					}
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.UnprotectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.UnprotectReg(d59.Reg)
						ctx.UnprotectReg(d59.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				bbpos_3_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				ctx.ReclaimUntrackedRegs()
				d100 := ctx.EmitSliceElementAddress(&d65, &d12, 16)
				ctx.EnsureDesc(&d100)
				r28 := ctx.AllocRegExcept(d100.Reg)
				ctx.EmitMovRegMem(r28, d100.Reg, 8)
				ctx.EmitMovRegMem(d100.Reg, d100.Reg, 0)
				d99 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d100.Reg, Reg2: r28}
				ctx.BindReg(d100.Reg, &d99)
				ctx.BindReg(r28, &d99)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d99)
				phiBase101 := ctx.AllocStack(int32(16))
				d102 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase101) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase101) + int32(0))
				_ = d102
				lbl43 := ctx.ReserveLabel()
				bbpos_5_0 := int32(-1)
				_ = bbpos_5_0
				lbl44 := ctx.ReserveLabel()
				_ = lbl44
				bbpos_5_1 := int32(-1)
				_ = bbpos_5_1
				lbl45 := ctx.ReserveLabel()
				_ = lbl45
				bbpos_5_2 := int32(-1)
				_ = bbpos_5_2
				lbl46 := ctx.ReserveLabel()
				_ = lbl46
				bbpos_5_3 := int32(-1)
				_ = bbpos_5_3
				lbl47 := ctx.ReserveLabel()
				_ = lbl47
				bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl44)
				ctx.ResolveFixups()
				d102 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase101) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
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
					ctx.EmitCopyStackWords(d103, int32(phiBase101)+int32(0), 2)
				} else if d103.Loc == LocInputPair {
					ctx.EnsureDesc(&d103)
					ctx.EmitStoreScmerToStack(d103, int32(phiBase101)+int32(0))
				} else if d103.Loc == LocRegPair || d103.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d103, int32(phiBase101)+int32(0))
				} else {
					ctx.EnsureDesc(&d103)
					ctx.EmitStoreToStack(d103, int32(phiBase101)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase101)+int32(0))+8)
				}
				if d99.Loc == LocReg || d99.Loc == LocFPReg {
					ctx.UnprotectReg(d99.Reg)
				} else if d99.Loc == LocRegPair {
					ctx.UnprotectReg(d99.Reg)
					ctx.UnprotectReg(d99.Reg2)
				}
				bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl47)
				ctx.ResolveFixups()
				d102 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase101) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d104 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase101) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d104)
				ctx.ReclaimUntrackedRegs()
				d104 = JITPrepareScmerGoArg(ctx, d104)
				ctx.SyncDesc(&d104)
				d105 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d104}, 1)
				d105.NoHeapPointer = true
				ctx.EmitAndRegImm32(d105.Reg, 1)
				d105.Type = tagBool
				ctx.BindReg(d105.Reg, &d105)
				ctx.ReclaimUntrackedRegs()
				d106 := d105
				ctx.EnsureDesc(&d106)
				if d106.Loc != LocImm && d106.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl48 := ctx.ReserveLabel()
				lbl49 := ctx.ReserveLabel()
				if d106.Loc == LocImm {
					if d106.Imm.Bool() {
						ctx.MarkLabel(lbl48)
						ctx.EmitJmp(lbl45)
					} else {
						ctx.MarkLabel(lbl49)
						ctx.EmitJmp(lbl46)
					}
				} else {
					ctx.EmitCmpRegImm32(d106.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl48)
					ctx.EmitJmp(lbl49)
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl45)
					ctx.MarkLabel(lbl49)
					ctx.EmitJmp(lbl46)
				}
				ctx.FreeDesc(&d105)
				bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl46)
				ctx.ResolveFixups()
				d104 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase101) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r29 := ctx.AllocReg()
				r30 := ctx.AllocRegExcept(r29)
				d107 := JITValueDesc{Loc: LocRegPair, Reg: r29, Reg2: r30}
				ctx.BindReg(r29, &d107)
				ctx.BindReg(r30, &d107)
				ctx.EmitMovPairToResult(&d104, &d107)
				ctx.EmitJmp(lbl43)
				bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl45)
				ctx.ResolveFixups()
				d104 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase101) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d104 = JITPrepareScmerGoArg(ctx, d104)
				ctx.SyncDesc(&d104)
				d108 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d104}, 1)
				d108.NoHeapPointer = false
				ctx.BindReg(d108.Reg, &d108)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d109 JITValueDesc
				ctx.EnsureDesc(&d108)
				if d108.Loc == LocImm {
					fieldAddr := uintptr(d108.Imm.Int()) + 32
					r31 := ctx.AllocReg()
					r32 := ctx.AllocRegExcept(r31)
					ctx.EmitMovRegMem64(r31, fieldAddr)
					ctx.EmitMovRegMem64(r32, fieldAddr+8)
					d109 = JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
					ctx.BindReg(r31, &d109)
					ctx.BindReg(r32, &d109)
				} else {
					off := int32(32)
					baseReg := d108.Reg
					r33 := ctx.AllocRegExcept(baseReg)
					r34 := ctx.AllocRegExcept(baseReg, r33)
					ctx.EmitMovRegMem(r33, baseReg, off)
					ctx.EmitMovRegMem(r34, baseReg, off+8)
					d109 = JITValueDesc{Loc: LocRegPair, Reg: r33, Reg2: r34}
					ctx.BindReg(r33, &d109)
					ctx.BindReg(r34, &d109)
				}
				ctx.FreeDesc(&d108)
				ctx.StabilizeDescForControlFlow(&d109)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d109)
				if d109.Loc == LocReg || d109.Loc == LocFPReg {
					ctx.ProtectReg(d109.Reg)
				} else if d109.Loc == LocRegPair {
					ctx.ProtectReg(d109.Reg)
					ctx.ProtectReg(d109.Reg2)
				}
				d110 := d109
				if d110.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d110)
				if d110.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d110, int32(phiBase101)+int32(0), 2)
				} else if d110.Loc == LocInputPair {
					ctx.EnsureDesc(&d110)
					ctx.EmitStoreScmerToStack(d110, int32(phiBase101)+int32(0))
				} else if d110.Loc == LocRegPair || d110.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d110, int32(phiBase101)+int32(0))
				} else {
					ctx.EnsureDesc(&d110)
					ctx.EmitStoreToStack(d110, int32(phiBase101)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase101)+int32(0))+8)
				}
				if d109.Loc == LocReg || d109.Loc == LocFPReg {
					ctx.UnprotectReg(d109.Reg)
				} else if d109.Loc == LocRegPair {
					ctx.UnprotectReg(d109.Reg)
					ctx.UnprotectReg(d109.Reg2)
				}
				ctx.EmitJmp(lbl47)
				ctx.MarkLabel(lbl43)
				d111 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r29, Reg2: r30}
				ctx.BindReg(r29, &d111)
				ctx.BindReg(r30, &d111)
				ctx.BindReg(r29, &d111)
				ctx.BindReg(r30, &d111)
				ctx.StabilizeDescForControlFlow(&d111)
				ctx.FreeDesc(&d99)
				ctx.ReclaimUntrackedRegs()
				var d112 JITValueDesc
				if d29.SliceSizeKnown {
					d112 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.KnownSliceLen))}
				} else if d29.Loc == LocImm {
					d112 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d29.StackOff))}
				} else if d29.Loc == LocStackTriple {
					d112 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d29.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d29)
					if d29.Loc == LocRegPair || d29.Loc == LocRegTriple {
						d112 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg2, ID: 0}
					} else if d29.Loc == LocReg {
						d112 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				ctx.StabilizeDescForControlFlow(&d112)
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase13)+int32(48))
				bbpos_3_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d113 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d113)
				ctx.EnsureDesc(&d113)
				var d114 JITValueDesc
				if d113.Loc == LocImm {
					d114 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d113.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d113.Reg)
					ctx.EmitMovRegReg(scratch, d113.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d114 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d114)
				}
				if d114.Loc == LocReg && d113.Loc == LocReg && d114.Reg == d113.Reg {
					ctx.TransferReg(d113.Reg)
					d113.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d114)
				ctx.FreeDesc(&d113)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d114)
				ctx.EnsureDesc(&d112)
				ctx.EnsureDescsTogether(&d114, &d112)
				var d115 JITValueDesc
				if d114.Loc == LocImm && d112.Loc == LocImm {
					d115 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d114.Imm.Int() < d112.Imm.Int())}
				} else if d112.Loc == LocImm {
					r35 := ctx.AllocRegExcept(d114.Reg)
					if d112.Imm.Int() >= -2147483648 && d112.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d114.Reg, int32(d112.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d112.Imm.Int()))
						ctx.EmitCmpInt64(d114.Reg, ctx.ScratchReg)
					}
					d115 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r35, Condition: CondSignedLess}
					ctx.BindReg(r35, &d115)
				} else if d114.Loc == LocImm {
					r36 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d114.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d112.Reg)
					d115 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r36, Condition: CondSignedLess}
					ctx.BindReg(r36, &d115)
				} else {
					r37 := ctx.AllocRegExcept(d114.Reg)
					ctx.EmitCmpInt64(d114.Reg, d112.Reg)
					d115 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r37, Condition: CondSignedLess}
					ctx.BindReg(r37, &d115)
				}
				ctx.ReclaimUntrackedRegs()
				d116 := d115
				ctx.SyncDesc(&d116)
				if d116.Loc != LocImm && d116.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl50 := ctx.ReserveLabel()
				lbl51 := ctx.ReserveLabel()
				if d116.Loc == LocImm {
					if d116.Imm.Bool() {
						ctx.MarkLabel(lbl50)
						ctx.EmitJmp(lbl15)
					} else {
						ctx.MarkLabel(lbl51)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase13)+int32(64))
						ctx.EmitJmp(lbl16)
					}
				} else {
					ctx.EmitJump(d116.Condition, lbl50)
					ctx.EmitJmp(lbl51)
					ctx.FreeDesc(&d115)
					ctx.MarkLabel(lbl50)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl51)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase13)+int32(64))
					ctx.EmitJmp(lbl16)
				}
				bbpos_3_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d113 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d17 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d117 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(64)}
				ctx.ReclaimUntrackedRegs()
				d118 := d117
				ctx.EnsureDesc(&d118)
				if d118.Loc != LocImm && d118.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl52 := ctx.ReserveLabel()
				lbl53 := ctx.ReserveLabel()
				if d118.Loc == LocImm {
					if d118.Imm.Bool() {
						ctx.MarkLabel(lbl52)
						ctx.SyncDesc(&d59)
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.ProtectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.ProtectReg(d59.Reg)
							ctx.ProtectReg(d59.Reg2)
						}
						d119 := d59
						if d119.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d119)
						if d119.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d119, int32(phiBase13)+int32(0), 3)
						} else {
							if d119.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d119.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
							ctx.EmitStoreRegMem(d119.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
							ctx.EmitStoreRegMem(d119.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
						}
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.UnprotectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.UnprotectReg(d59.Reg)
							ctx.UnprotectReg(d59.Reg2)
						}
						ctx.EmitJmp(lbl8)
					} else {
						ctx.MarkLabel(lbl53)
						ctx.EmitJmp(lbl18)
					}
				} else {
					ctx.EmitCmpRegImm32(d118.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl52)
					ctx.EmitJmp(lbl53)
					ctx.MarkLabel(lbl52)
					ctx.SyncDesc(&d59)
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.ProtectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.ProtectReg(d59.Reg)
						ctx.ProtectReg(d59.Reg2)
					}
					d120 := d59
					if d120.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d120)
					if d120.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d120, int32(phiBase13)+int32(0), 3)
					} else {
						if d120.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d120.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
						ctx.EmitStoreRegMem(d120.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
						ctx.EmitStoreRegMem(d120.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
					}
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.UnprotectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.UnprotectReg(d59.Reg)
						ctx.UnprotectReg(d59.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl53)
					ctx.EmitJmp(lbl18)
				}
				ctx.FreeDesc(&d117)
				bbpos_3_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d113 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d117 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d114)
				ctx.ReclaimUntrackedRegs()
				d122 := ctx.EmitSliceElementAddress(&d29, &d114, 16)
				ctx.EnsureDesc(&d122)
				r38 := ctx.AllocRegExcept(d122.Reg)
				ctx.EmitMovRegMem(r38, d122.Reg, 8)
				ctx.EmitMovRegMem(d122.Reg, d122.Reg, 0)
				d121 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d122.Reg, Reg2: r38}
				ctx.BindReg(d122.Reg, &d121)
				ctx.BindReg(r38, &d121)
				ctx.ReclaimUntrackedRegs()
				d121 = JITPrepareScmerGoArg(ctx, d121)
				d111 = JITPrepareScmerGoArg(ctx, d111)
				ctx.SyncDesc(&d121)
				ctx.SyncDesc(&d111)
				d123 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d121, d111}, 1)
				d123.NoHeapPointer = true
				ctx.EmitAndRegImm32(d123.Reg, 1)
				d123.Type = tagBool
				ctx.BindReg(d123.Reg, &d123)
				ctx.FreeDesc(&d121)
				ctx.ReclaimUntrackedRegs()
				d124 := d123
				ctx.EnsureDesc(&d124)
				if d124.Loc != LocImm && d124.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl54 := ctx.ReserveLabel()
				lbl55 := ctx.ReserveLabel()
				if d124.Loc == LocImm {
					if d124.Imm.Bool() {
						ctx.MarkLabel(lbl54)
						ctx.EmitJmp(lbl17)
					} else {
						ctx.MarkLabel(lbl55)
						ctx.SyncDesc(&d114)
						if d114.Loc == LocReg || d114.Loc == LocFPReg {
							ctx.ProtectReg(d114.Reg)
						} else if d114.Loc == LocRegPair {
							ctx.ProtectReg(d114.Reg)
							ctx.ProtectReg(d114.Reg2)
						}
						d125 := d114
						if d125.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d125)
						ctx.EmitStoreToStack(d125, int32(phiBase13)+int32(48))
						if d114.Loc == LocReg || d114.Loc == LocFPReg {
							ctx.UnprotectReg(d114.Reg)
						} else if d114.Loc == LocRegPair {
							ctx.UnprotectReg(d114.Reg)
							ctx.UnprotectReg(d114.Reg2)
						}
						ctx.EmitJmp(lbl14)
					}
				} else {
					ctx.EmitCmpRegImm32(d124.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl54)
					ctx.EmitJmp(lbl55)
					ctx.MarkLabel(lbl54)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl55)
					ctx.SyncDesc(&d114)
					if d114.Loc == LocReg || d114.Loc == LocFPReg {
						ctx.ProtectReg(d114.Reg)
					} else if d114.Loc == LocRegPair {
						ctx.ProtectReg(d114.Reg)
						ctx.ProtectReg(d114.Reg2)
					}
					d126 := d114
					if d126.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d126)
					ctx.EmitStoreToStack(d126, int32(phiBase13)+int32(48))
					if d114.Loc == LocReg || d114.Loc == LocFPReg {
						ctx.UnprotectReg(d114.Reg)
					} else if d114.Loc == LocRegPair {
						ctx.UnprotectReg(d114.Reg)
						ctx.UnprotectReg(d114.Reg2)
					}
					ctx.EmitJmp(lbl14)
				}
				ctx.FreeDesc(&d123)
				bbpos_3_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d113 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d117 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				stackArray127 := ctx.AllocStack(int32(16))
				_ = stackArray127
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d111)
				ctx.EmitStoreScmerToStack(d111, int32(stackArray127)+int32(0))
				ctx.ReclaimUntrackedRegs()
				d128 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d128
				ctx.ReclaimUntrackedRegs()
				r39 := ctx.AllocReg()
				r40 := ctx.AllocRegExcept(r39)
				r41 := ctx.AllocRegExcept(r39, r40)
				d129 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r39, Reg2: r40, Reg3: r41}
				ctx.BindReg(r39, &d129)
				ctx.BindReg(r40, &d129)
				ctx.BindReg(r41, &d129)
				ctx.BindReg(r39, &d129)
				ctx.BindReg(r40, &d129)
				ctx.BindReg(r41, &d129)
				ctx.EmitLeaRegMem(d129.Reg, ctx.StackReg, int32(stackArray127))
				ctx.EmitMovRegImm64(d129.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d129.Reg3, uint64(1))
				callResults130 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d29, d129}, []uint8{3}, []uint8{1})
				d131 := callResults130[0]
				d132 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase13) + int32(24)}
				ctx.EmitCopyDescWords(&d132, &d131, 3)
				ctx.FreeDesc(&d131)
				d131 = d132
				ctx.StabilizeDescForControlFlow(&d131)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d59)
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.ProtectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.ProtectReg(d59.Reg)
					ctx.ProtectReg(d59.Reg2)
				}
				d133 := d59
				if d133.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d133)
				if d133.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d133, int32(phiBase13)+int32(0), 3)
				} else {
					if d133.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d133.Reg, ctx.StackReg, int32(phiBase13)+int32(0))
					ctx.EmitStoreRegMem(d133.Reg2, ctx.StackReg, int32(phiBase13)+int32(0)+8)
					ctx.EmitStoreRegMem(d133.Reg3, ctx.StackReg, int32(phiBase13)+int32(0)+16)
				}
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.UnprotectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.UnprotectReg(d59.Reg)
					ctx.UnprotectReg(d59.Reg2)
				}
				ctx.EmitJmp(lbl8)
				bbpos_3_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
				d28 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(0)}
				d29 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(24)}
				d113 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(48)}
				d117 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase13) + int32(64)}
				d73 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase13) + int32(80)}
				d74 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase13) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase13)+int32(64))
				ctx.EmitJmp(lbl16)
				ctx.MarkLabel(lbl2)
				d134 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d134)
				ctx.BindReg(r3, &d134)
				ctx.BindReg(r2, &d134)
				ctx.BindReg(r3, &d134)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d134)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				ctx.FreeDesc(&d4)
				if d134.Loc == LocImm {
					if result.Loc == LocAny {
						return d134
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d134)
				if d134.Loc == LocRegPair || d134.Loc == LocStackPair || d134.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d134, &result)
					result.Type = d134.Type
				} else {
					switch d134.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d134)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d134)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d134)
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
				if d8.Loc == LocRegTriple {
					ctx.BindReg(d8.Reg, &d8)
					ctx.BindReg(d8.Reg2, &d8)
					ctx.BindReg(d8.Reg3, &d8)
				}
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
				ctx.SyncDesc(&d21)
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
					ctx.EmitStoreRegMem(d24.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
					ctx.EmitStoreRegMem(d24.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
					ctx.EmitStoreRegMem(d24.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
				ctx.SyncDesc(&d28)
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
				_ = d29
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
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
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
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d45.Imm.Int()))
						ctx.EmitSubInt64(r10, ctx.ScratchReg)
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
							ctx.EmitStoreRegMem(d53.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d53.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d53.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
						ctx.EmitStoreRegMem(d54.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d54.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d54.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
				ctx.SyncDesc(&d58)
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
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
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
					ctx.EmitStoreRegMem(d61.Reg, ctx.StackReg, int32(phiBase14)+int32(56))
					ctx.EmitStoreRegMem(d61.Reg2, ctx.StackReg, int32(phiBase14)+int32(56)+8)
					ctx.EmitStoreRegMem(d61.Reg3, ctx.StackReg, int32(phiBase14)+int32(56)+16)
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
				ctx.SyncDesc(&d65)
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
							ctx.EmitStoreRegMem(d66.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d66.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d66.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
						ctx.EmitStoreRegMem(d67.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d67.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d67.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
				_ = d68
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
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
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
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d11.Imm.Int()))
						ctx.EmitCmpInt64(d82.Reg, ctx.ScratchReg)
					}
					d83 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedGreater}
					ctx.BindReg(r22, &d83)
				} else if d82.Loc == LocImm {
					r23 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d82.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d11.Reg)
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
				ctx.SyncDesc(&d84)
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
							ctx.EmitStoreRegMem(d85.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d85.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d85.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
						ctx.EmitStoreRegMem(d86.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d86.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d86.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
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
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d106.Imm.Int()))
						ctx.EmitCmpInt64(d108.Reg, ctx.ScratchReg)
					}
					d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r32, Condition: CondSignedLess}
					ctx.BindReg(r32, &d109)
				} else if d108.Loc == LocImm {
					r33 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d108.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d106.Reg)
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
				ctx.SyncDesc(&d110)
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
							ctx.EmitStoreRegMem(d111.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
							ctx.EmitStoreRegMem(d111.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
							ctx.EmitStoreRegMem(d111.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
						ctx.EmitStoreRegMem(d112.Reg, ctx.StackReg, int32(phiBase14)+int32(0))
						ctx.EmitStoreRegMem(d112.Reg2, ctx.StackReg, int32(phiBase14)+int32(0)+8)
						ctx.EmitStoreRegMem(d112.Reg3, ctx.StackReg, int32(phiBase14)+int32(0)+16)
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
				_ = d120
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
				ctx.SyncDesc(&d121)
				if ctx.hasBooleanFlags(d121) {
					return d121
				}
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
			initializer := PrepareSerialProc(a[3])
			var keyArgs [1]Scmer
			var reduceArgs [2]Scmer
			result := NewFastDictValue(groupAssocCapacity(len(input)))
			for _, item := range input {
				keyArgs[0] = item
				result.ReduceValue(key.Call(keyArgs[:]), item, &initializer, &reduce, reduceArgs[:])
			}
			return NewFastDict(result)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "groups list elements by key and reduces every group from a neutral value",
			Params: []*TypeDescriptor{
				{Kind: "list", Label: "list", NoEscape: true},
				{Kind: "func", Label: "key", Params: []*TypeDescriptor{{Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "func", Label: "reducer", Params: []*TypeDescriptor{{Kind: "any", Label: "current"}, {Kind: "any", Label: "item"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "neutral", Description: "initial value or zero-argument factory invoked once per new group"},
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
				var stackArray13 int32
				var stackArray14 int32
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
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
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
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
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
					d10 = args[3]
					d10.ID = 0
					if d10.Loc == LocLambdaTemplate {
						d11 = d10
					} else if d10.Loc == LocImm {
						optimizedCallback12 := jitPrepareCallback(d10.Imm)
						ctx.TrackImm(optimizedCallback12)
						d11 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback12, Rooted: true}
					} else {
						if d10.Loc == LocInputPair && int(d10.StackOff) < ctx.InputArgCount {
							d11 = ctx.RequestOptimizedCallback(int(d10.StackOff))
						} else {
							d11 = jitCopyScmerToPair(ctx, d10)
						}
					}
					ctx.FreeDesc(&d10)
					ctx.EnsureDesc(&d11)
					ctx.StabilizeDescForControlFlow(&d11)
					stackArray13 = ctx.AllocStack(int32(16))
					_ = stackArray13
					stackArray14 = ctx.AllocStack(int32(32))
					_ = stackArray14
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
					ctx.EnsureDesc(&d15)
					d16 = d15
					_ = d16
					ctx.StabilizeDescForControlFlow(&d16)
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
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 32)
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d17)
					}
					ctx.ReclaimUntrackedRegs()
					d18 = d17
					ctx.SyncDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d18.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d17)
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
					d19 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d19)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d16)
					}
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl5)
					d20 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d20)
					ctx.BindReg(r1, &d20)
					ctx.FreeDesc(&d15)
					ctx.EnsureDesc(&d20)
					d21 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d20}, 1)
					ctx.StabilizeDescForControlFlow(&d21)
					ctx.FreeDesc(&d20)
					if d3.SliceSizeKnown {
						d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d22 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d22)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
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
					ctx.EnsureDesc(&d22)
					ctx.EnsureDescsTogether(&d23, &d22)
					if d23.Loc == LocImm && d22.Loc == LocImm {
						d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d23.Imm.Int() < d22.Imm.Int())}
					} else if d22.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d23.Reg)
						if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d23.Reg, int32(d22.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d22.Imm.Int()))
							ctx.EmitCmpInt64(d23.Reg, ctx.ScratchReg)
						}
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d24)
					} else if d23.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d23.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d22.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d24)
					} else {
						r4 := ctx.AllocRegExcept(d23.Reg)
						ctx.EmitCmpInt64(d23.Reg, d22.Reg)
						d24 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d24)
					}
					d25 = d24
					ctx.SyncDesc(&d25)
					if d25.Loc != LocImm && d25.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d25.Loc == LocImm {
						if d25.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d25.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d24)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap26 := d1
						snap27 := d2
						snap28 := d3
						snap29 := d4
						snap30 := d5
						snap31 := d7
						snap32 := d8
						snap33 := d10
						snap34 := d11
						snap35 := d15
						snap36 := d16
						snap37 := d17
						snap38 := d18
						snap39 := d19
						snap40 := d20
						snap41 := d21
						snap42 := d22
						snap43 := d23
						snap44 := d24
						snap45 := d25
						alloc46 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc46)
						d1 = snap26
						d2 = snap27
						d3 = snap28
						d4 = snap29
						d5 = snap30
						d7 = snap31
						d8 = snap32
						d10 = snap33
						d11 = snap34
						d15 = snap35
						d16 = snap36
						d17 = snap37
						d18 = snap38
						d19 = snap39
						d20 = snap40
						d21 = snap41
						d22 = snap42
						d23 = snap43
						d24 = snap44
						d25 = snap45
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d11)
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d23)
					d48 = ctx.EmitSliceElementAddress(&d3, &d23, 16)
					ctx.EnsureDesc(&d48)
					r5 := ctx.AllocRegExcept(d48.Reg)
					ctx.EmitMovRegMem(r5, d48.Reg, 8)
					ctx.EmitMovRegMem(d48.Reg, d48.Reg, 0)
					d47 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d48.Reg, Reg2: r5}
					ctx.BindReg(d48.Reg, &d47)
					ctx.BindReg(r5, &d47)
					ctx.SyncDesc(&d47)
					ctx.EmitStoreScmerToStack(d47, int32(stackArray13)+int32(0))
					d49 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d49
					callbackArgs51 := make([]JITValueDesc, 1)
					callbackArgs51[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray13) + 0}
					_ = d50
					callbackResultOff52 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff52))
					ctx.FreeDesc(&d49)
					ctx.StabilizeDescAcrossNestedCall(&d23)
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
							d57 = jitCopyScmerToPair(ctx, d5)
							d50 = jitEmitDynamicCallableAt(ctx, d57, callbackArgs51, int32(stackArray13), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff52), ID: 0})
						}
					}
					d58 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d58
					if d21.Loc == LocRegPair || d21.Loc == LocStackPair || d21.Loc == LocRegTriple || d21.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d50 = JITPrepareScmerGoArg(ctx, d50)
					d47 = JITPrepareScmerGoArg(ctx, d47)
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d58 = JITPrepareGoSliceArg(ctx, d58)
					if d58.Loc != LocRegTriple && d58.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((*FastDict).ReduceValue arg5)")
					}
					ctx.SyncDesc(&d21)
					ctx.SyncDesc(&d50)
					ctx.SyncDesc(&d47)
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d8)
					ctx.SyncDesc(&d58)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).ReduceValue), []JITValueDesc{d21, d50, d47, d11, d8, d58})
					ctx.FreeDesc(&d50)
					ctx.FreeDesc(&d47)
					ctx.SyncDesc(&d23)
					if d23.Loc == LocReg || d23.Loc == LocFPReg {
						ctx.ProtectReg(d23.Reg)
					} else if d23.Loc == LocRegPair {
						ctx.ProtectReg(d23.Reg)
						ctx.ProtectReg(d23.Reg2)
					}
					d59 = d23
					if d59.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d59)
					ctx.EmitStoreToStack(d59, int32(bbs[1].PhiBase)+int32(0))
					if d23.Loc == LocReg || d23.Loc == LocFPReg {
						ctx.UnprotectReg(d23.Reg)
					} else if d23.Loc == LocRegPair {
						ctx.UnprotectReg(d23.Reg)
						ctx.UnprotectReg(d23.Reg2)
					}
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d21)
					if d21.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d60 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d21.Reg, Reg2: r6}
						ctx.BindReg(d21.Reg, &d60)
						ctx.BindReg(r6, &d60)
						ctx.TransferReg(d21.Reg)
						ctx.BindReg(d21.Reg, &d60)
						ctx.BindReg(r6, &d60)
						d21.Loc = LocNone
					}
					ctx.SyncDesc(&d60)
					if d60.Loc == LocRegPair || d60.Loc == LocStackPair || d60.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d60, &result)
						result.Type = d60.Type
					} else {
						switch d60.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d60)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d60)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d60)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d60, &result)
							result.Type = d60.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
			JITInlineCallbacks: true,
			JITInlineCost:      44,
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
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
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
				var callbackResultOff48 int32
				var d51 JITValueDesc
				_ = d51
				var d53 JITValueDesc
				_ = d53
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
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
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
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
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
					ctx.SyncDesc(&d15)
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
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d20)
					}
					if d20.Loc == LocReg && d1.Loc == LocReg && d20.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d20, &d19)
					if d20.Loc == LocImm && d19.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d20.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d20.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d20.Reg, ctx.ScratchReg)
						}
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d21)
					} else if d20.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d20.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d19.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d21)
					} else {
						r4 := ctx.AllocRegExcept(d20.Reg)
						ctx.EmitCmpInt64(d20.Reg, d19.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d21)
					}
					d22 = d21
					ctx.SyncDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d22.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d21)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap23 := d1
						snap24 := d2
						snap25 := d3
						snap26 := d4
						snap27 := d5
						snap28 := d7
						snap29 := d8
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
						alloc41 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc41)
						d1 = snap23
						d2 = snap24
						d3 = snap25
						d4 = snap26
						d5 = snap27
						d7 = snap28
						d8 = snap29
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
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d20)
					d43 = ctx.EmitSliceElementAddress(&d3, &d20, 16)
					ctx.EnsureDesc(&d43)
					r5 := ctx.AllocRegExcept(d43.Reg)
					ctx.EmitMovRegMem(r5, d43.Reg, 8)
					ctx.EmitMovRegMem(d43.Reg, d43.Reg, 0)
					d42 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d43.Reg, Reg2: r5}
					ctx.BindReg(d43.Reg, &d42)
					ctx.BindReg(r5, &d42)
					ctx.SyncDesc(&d42)
					ctx.EmitStoreScmerToStack(d42, int32(stackArray10)+int32(0))
					d44 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d44)
					ctx.EmitStoreScmerToStack(d44, int32(stackArray11)+int32(0))
					ctx.FreeDesc(&d44)
					ctx.SyncDesc(&d42)
					ctx.EmitStoreScmerToStack(d42, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d42)
					d45 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d45
					callbackArgs47 := make([]JITValueDesc, 1)
					callbackArgs47[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					_ = d46
					callbackResultOff48 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff48))
					ctx.FreeDesc(&d45)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs49 := ctx.StabilizeCallbackArgs(callbackArgs47)
						ctx.ReclaimUntrackedRegs()
						outerRegs50 := ctx.PreserveOuterRegs()
						d46 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs49, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff48), ID: 0})
						ctx.RestoreOuterRegs(outerRegs50)
						ctx.ReclaimUntrackedRegs()
					} else {
						d51, knownBuiltin52 := jitEmitKnownDeclaration(ctx, d5, callbackArgs47, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff48), ID: 0})
						if knownBuiltin52 {
							d46 = d51
						} else {
							ctx.Coverage.DynamicCalls++
							d53 = jitCopyScmerToPair(ctx, d5)
							d46 = jitEmitDynamicCallableAt(ctx, d53, callbackArgs47, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff48), ID: 0})
						}
					}
					d54 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d54
					callbackArgs56 := make([]JITValueDesc, 2)
					callbackArgs56[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 0}
					callbackArgs56[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 16}
					_ = d55
					callbackResultOff57 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff57))
					ctx.FreeDesc(&d54)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs58 := ctx.StabilizeCallbackArgs(callbackArgs56)
						ctx.ReclaimUntrackedRegs()
						outerRegs59 := ctx.PreserveOuterRegs()
						d55 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs58, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0})
						ctx.RestoreOuterRegs(outerRegs59)
						ctx.ReclaimUntrackedRegs()
					} else {
						d60, knownBuiltin61 := jitEmitKnownDeclaration(ctx, d8, callbackArgs56, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0})
						if knownBuiltin61 {
							d55 = d60
						} else {
							ctx.Coverage.DynamicCalls++
							d62 = jitCopyScmerToPair(ctx, d8)
							d55 = jitEmitDynamicCallableAt(ctx, d62, callbackArgs56, int32(stackArray11), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff57), ID: 0})
						}
					}
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d46 = JITPrepareScmerGoArg(ctx, d46)
					d55 = JITPrepareScmerGoArg(ctx, d55)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d46)
					ctx.SyncDesc(&d55)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d46, d55})
					ctx.FreeDesc(&d46)
					ctx.FreeDesc(&d55)
					ctx.SyncDesc(&d20)
					if d20.Loc == LocReg || d20.Loc == LocFPReg {
						ctx.ProtectReg(d20.Reg)
					} else if d20.Loc == LocRegPair {
						ctx.ProtectReg(d20.Reg)
						ctx.ProtectReg(d20.Reg2)
					}
					d63 = d20
					if d63.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d63)
					ctx.EmitStoreToStack(d63, int32(bbs[1].PhiBase)+int32(0))
					if d20.Loc == LocReg || d20.Loc == LocFPReg {
						ctx.UnprotectReg(d20.Reg)
					} else if d20.Loc == LocRegPair {
						ctx.UnprotectReg(d20.Reg)
						ctx.UnprotectReg(d20.Reg2)
					}
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d64 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r6}
						ctx.BindReg(d18.Reg, &d64)
						ctx.BindReg(r6, &d64)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d64)
						ctx.BindReg(r6, &d64)
						d18.Loc = LocNone
					}
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
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
				var d20 JITValueDesc
				_ = d20
				var d21 JITValueDesc
				_ = d21
				var d22 JITValueDesc
				_ = d22
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
				var callbackResultOff49 int32
				var d52 JITValueDesc
				_ = d52
				var d54 JITValueDesc
				_ = d54
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
				var d65 JITValueDesc
				_ = d65
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
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
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
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
					ctx.SyncDesc(&d15)
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
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d20)
					}
					if d20.Loc == LocReg && d1.Loc == LocReg && d20.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d20)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d20)
					ctx.EnsureDesc(&d19)
					ctx.EnsureDescsTogether(&d20, &d19)
					if d20.Loc == LocImm && d19.Loc == LocImm {
						d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d20.Imm.Int() < d19.Imm.Int())}
					} else if d19.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d20.Reg)
						if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d20.Reg, int32(d19.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d19.Imm.Int()))
							ctx.EmitCmpInt64(d20.Reg, ctx.ScratchReg)
						}
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d21)
					} else if d20.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d20.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d19.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d21)
					} else {
						r4 := ctx.AllocRegExcept(d20.Reg)
						ctx.EmitCmpInt64(d20.Reg, d19.Reg)
						d21 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d21)
					}
					d22 = d21
					ctx.SyncDesc(&d22)
					if d22.Loc != LocImm && d22.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d22.Loc == LocImm {
						if d22.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d22.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d21)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap23 := d1
						snap24 := d2
						snap25 := d3
						snap26 := d4
						snap27 := d5
						snap28 := d7
						snap29 := d8
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
						alloc41 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc41)
						d1 = snap23
						d2 = snap24
						d3 = snap25
						d4 = snap26
						d5 = snap27
						d7 = snap28
						d8 = snap29
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
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d20)
					d43 = ctx.EmitSliceElementAddress(&d3, &d20, 16)
					ctx.EnsureDesc(&d43)
					r5 := ctx.AllocRegExcept(d43.Reg)
					ctx.EmitMovRegMem(r5, d43.Reg, 8)
					ctx.EmitMovRegMem(d43.Reg, d43.Reg, 0)
					d42 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d43.Reg, Reg2: r5}
					ctx.BindReg(d43.Reg, &d42)
					ctx.BindReg(r5, &d42)
					d44 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d44)
					ctx.EmitStoreScmerToStack(d44, int32(stackArray10)+int32(0))
					ctx.FreeDesc(&d44)
					ctx.SyncDesc(&d42)
					ctx.EmitStoreScmerToStack(d42, int32(stackArray10)+int32(16))
					d45 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d45)
					ctx.EmitStoreScmerToStack(d45, int32(stackArray11)+int32(0))
					ctx.FreeDesc(&d45)
					ctx.SyncDesc(&d42)
					ctx.EmitStoreScmerToStack(d42, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d42)
					d46 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d46
					callbackArgs48 := make([]JITValueDesc, 2)
					callbackArgs48[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					callbackArgs48[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 16}
					_ = d47
					callbackResultOff49 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff49))
					ctx.FreeDesc(&d46)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs50 := ctx.StabilizeCallbackArgs(callbackArgs48)
						ctx.ReclaimUntrackedRegs()
						outerRegs51 := ctx.PreserveOuterRegs()
						d47 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs50, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff49), ID: 0})
						ctx.RestoreOuterRegs(outerRegs51)
						ctx.ReclaimUntrackedRegs()
					} else {
						d52, knownBuiltin53 := jitEmitKnownDeclaration(ctx, d5, callbackArgs48, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff49), ID: 0})
						if knownBuiltin53 {
							d47 = d52
						} else {
							ctx.Coverage.DynamicCalls++
							d54 = jitCopyScmerToPair(ctx, d5)
							d47 = jitEmitDynamicCallableAt(ctx, d54, callbackArgs48, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff49), ID: 0})
						}
					}
					d55 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d55
					callbackArgs57 := make([]JITValueDesc, 2)
					callbackArgs57[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 0}
					callbackArgs57[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 16}
					_ = d56
					callbackResultOff58 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff58))
					ctx.FreeDesc(&d55)
					ctx.StabilizeDescAcrossNestedCall(&d20)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs59 := ctx.StabilizeCallbackArgs(callbackArgs57)
						ctx.ReclaimUntrackedRegs()
						outerRegs60 := ctx.PreserveOuterRegs()
						d56 = JITEmitProcInlineWithOuter(ctx, &d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs59, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff58), ID: 0})
						ctx.RestoreOuterRegs(outerRegs60)
						ctx.ReclaimUntrackedRegs()
					} else {
						d61, knownBuiltin62 := jitEmitKnownDeclaration(ctx, d8, callbackArgs57, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff58), ID: 0})
						if knownBuiltin62 {
							d56 = d61
						} else {
							ctx.Coverage.DynamicCalls++
							d63 = jitCopyScmerToPair(ctx, d8)
							d56 = jitEmitDynamicCallableAt(ctx, d63, callbackArgs57, int32(stackArray11), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff58), ID: 0})
						}
					}
					if d18.Loc == LocRegPair || d18.Loc == LocStackPair || d18.Loc == LocRegTriple || d18.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d47 = JITPrepareScmerGoArg(ctx, d47)
					d56 = JITPrepareScmerGoArg(ctx, d56)
					ctx.SyncDesc(&d18)
					ctx.SyncDesc(&d47)
					ctx.SyncDesc(&d56)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d18, d47, d56})
					ctx.FreeDesc(&d47)
					ctx.FreeDesc(&d56)
					ctx.SyncDesc(&d20)
					if d20.Loc == LocReg || d20.Loc == LocFPReg {
						ctx.ProtectReg(d20.Reg)
					} else if d20.Loc == LocRegPair {
						ctx.ProtectReg(d20.Reg)
						ctx.ProtectReg(d20.Reg2)
					}
					d64 = d20
					if d64.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d64)
					ctx.EmitStoreToStack(d64, int32(bbs[1].PhiBase)+int32(0))
					if d20.Loc == LocReg || d20.Loc == LocFPReg {
						ctx.UnprotectReg(d20.Reg)
					} else if d20.Loc == LocRegPair {
						ctx.UnprotectReg(d20.Reg)
						ctx.UnprotectReg(d20.Reg2)
					}
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d18)
					if d18.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d65 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d18.Reg, Reg2: r6}
						ctx.BindReg(d18.Reg, &d65)
						ctx.BindReg(r6, &d65)
						ctx.TransferReg(d18.Reg)
						ctx.BindReg(d18.Reg, &d65)
						ctx.BindReg(r6, &d65)
						d18.Loc = LocNone
					}
					ctx.SyncDesc(&d65)
					if d65.Loc == LocRegPair || d65.Loc == LocStackPair || d65.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d65, &result)
						result.Type = d65.Type
					} else {
						switch d65.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d65)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d65)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d65)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d65, &result)
							result.Type = d65.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var callbackResultOff41 int32
				var d44 JITValueDesc
				_ = d44
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
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
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
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
					ctx.SyncDesc(&d11)
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
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d16)
					}
					if d16.Loc == LocReg && d1.Loc == LocReg && d16.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDescsTogether(&d16, &d15)
					if d16.Loc == LocImm && d15.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() < d15.Imm.Int())}
					} else if d15.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d16.Reg)
						if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d16.Reg, int32(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d15.Imm.Int()))
							ctx.EmitCmpInt64(d16.Reg, ctx.ScratchReg)
						}
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d17)
					} else if d16.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d16.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d15.Reg)
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d17)
					} else {
						r4 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpInt64(d16.Reg, d15.Reg)
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d17)
					}
					d18 = d17
					ctx.SyncDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d18.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d17)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap19 := d1
						snap20 := d2
						snap21 := d3
						snap22 := d4
						snap23 := d5
						snap24 := d8
						snap25 := d9
						snap26 := d10
						snap27 := d11
						snap28 := d12
						snap29 := d13
						snap30 := d14
						snap31 := d15
						snap32 := d16
						snap33 := d17
						snap34 := d18
						alloc35 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc35)
						d1 = snap19
						d2 = snap20
						d3 = snap21
						d4 = snap22
						d5 = snap23
						d8 = snap24
						d9 = snap25
						d10 = snap26
						d11 = snap27
						d12 = snap28
						d13 = snap29
						d14 = snap30
						d15 = snap31
						d16 = snap32
						d17 = snap33
						d18 = snap34
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.EnsureDesc(&d16)
					d37 = ctx.EmitSliceElementAddress(&d3, &d16, 16)
					ctx.EnsureDesc(&d37)
					r5 := ctx.AllocRegExcept(d37.Reg)
					ctx.EmitMovRegMem(r5, d37.Reg, 8)
					ctx.EmitMovRegMem(d37.Reg, d37.Reg, 0)
					d36 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d37.Reg, Reg2: r5}
					ctx.BindReg(d37.Reg, &d36)
					ctx.BindReg(r5, &d36)
					ctx.SyncDesc(&d36)
					ctx.EmitStoreScmerToStack(d36, int32(stackArray7)+int32(0))
					ctx.FreeDesc(&d36)
					d38 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d38
					callbackArgs40 := make([]JITValueDesc, 1)
					callbackArgs40[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 0}
					_ = d39
					callbackResultOff41 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff41))
					ctx.FreeDesc(&d38)
					ctx.StabilizeDescAcrossNestedCall(&d16)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs42 := ctx.StabilizeCallbackArgs(callbackArgs40)
						ctx.ReclaimUntrackedRegs()
						outerRegs43 := ctx.PreserveOuterRegs()
						d39 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs42, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff41), ID: 0})
						ctx.RestoreOuterRegs(outerRegs43)
						ctx.ReclaimUntrackedRegs()
					} else {
						d44, knownBuiltin45 := jitEmitKnownDeclaration(ctx, d5, callbackArgs40, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff41), ID: 0})
						if knownBuiltin45 {
							d39 = d44
						} else {
							ctx.Coverage.DynamicCalls++
							d46 = jitCopyScmerToPair(ctx, d5)
							d39 = jitEmitDynamicCallableAt(ctx, d46, callbackArgs40, int32(stackArray7), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff41), ID: 0})
						}
					}
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d39 = JITPrepareScmerGoArg(ctx, d39)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d39)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d14, d39})
					ctx.FreeDesc(&d39)
					ctx.SyncDesc(&d16)
					if d16.Loc == LocReg || d16.Loc == LocFPReg {
						ctx.ProtectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.ProtectReg(d16.Reg)
						ctx.ProtectReg(d16.Reg2)
					}
					d47 = d16
					if d47.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d47)
					ctx.EmitStoreToStack(d47, int32(bbs[1].PhiBase)+int32(0))
					if d16.Loc == LocReg || d16.Loc == LocFPReg {
						ctx.UnprotectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.UnprotectReg(d16.Reg)
						ctx.UnprotectReg(d16.Reg2)
					}
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d48 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d14.Reg, Reg2: r6}
						ctx.BindReg(d14.Reg, &d48)
						ctx.BindReg(r6, &d48)
						ctx.TransferReg(d14.Reg)
						ctx.BindReg(d14.Reg, &d48)
						ctx.BindReg(r6, &d48)
						d14.Loc = LocNone
					}
					ctx.SyncDesc(&d48)
					if d48.Loc == LocRegPair || d48.Loc == LocStackPair || d48.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d48, &result)
						result.Type = d48.Type
					} else {
						switch d48.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d48)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d48)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d48)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d48, &result)
							result.Type = d48.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var callbackResultOff42 int32
				var d45 JITValueDesc
				_ = d45
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
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
				var bbs [4]BBDescriptor
				bbs[1].PhiBase = int32(phiBase0) + int32(0)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					if d3.Loc == LocRegTriple {
						ctx.BindReg(d3.Reg, &d3)
						ctx.BindReg(d3.Reg2, &d3)
						ctx.BindReg(d3.Reg3, &d3)
					}
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
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
					ctx.SyncDesc(&d11)
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
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d16)
					}
					if d16.Loc == LocReg && d1.Loc == LocReg && d16.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDesc(&d15)
					ctx.EnsureDescsTogether(&d16, &d15)
					if d16.Loc == LocImm && d15.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() < d15.Imm.Int())}
					} else if d15.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d16.Reg)
						if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d16.Reg, int32(d15.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d15.Imm.Int()))
							ctx.EmitCmpInt64(d16.Reg, ctx.ScratchReg)
						}
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d17)
					} else if d16.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d16.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d15.Reg)
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d17)
					} else {
						r4 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpInt64(d16.Reg, d15.Reg)
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d17)
					}
					d18 = d17
					ctx.SyncDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d18.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d17)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap19 := d1
						snap20 := d2
						snap21 := d3
						snap22 := d4
						snap23 := d5
						snap24 := d8
						snap25 := d9
						snap26 := d10
						snap27 := d11
						snap28 := d12
						snap29 := d13
						snap30 := d14
						snap31 := d15
						snap32 := d16
						snap33 := d17
						snap34 := d18
						alloc35 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc35)
						d1 = snap19
						d2 = snap20
						d3 = snap21
						d4 = snap22
						d5 = snap23
						d8 = snap24
						d9 = snap25
						d10 = snap26
						d11 = snap27
						d12 = snap28
						d13 = snap29
						d14 = snap30
						d15 = snap31
						d16 = snap32
						d17 = snap33
						d18 = snap34
					}
					if !bbs[2].Rendered {
						return bbs[2].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d5)
					ctx.EnsureDesc(&d16)
					d37 = ctx.EmitSliceElementAddress(&d3, &d16, 16)
					ctx.EnsureDesc(&d37)
					r5 := ctx.AllocRegExcept(d37.Reg)
					ctx.EmitMovRegMem(r5, d37.Reg, 8)
					ctx.EmitMovRegMem(d37.Reg, d37.Reg, 0)
					d36 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d37.Reg, Reg2: r5}
					ctx.BindReg(d37.Reg, &d36)
					ctx.BindReg(r5, &d36)
					d38 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d38)
					ctx.EmitStoreScmerToStack(d38, int32(stackArray7)+int32(0))
					ctx.FreeDesc(&d38)
					ctx.SyncDesc(&d36)
					ctx.EmitStoreScmerToStack(d36, int32(stackArray7)+int32(16))
					ctx.FreeDesc(&d36)
					d39 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d39
					callbackArgs41 := make([]JITValueDesc, 2)
					callbackArgs41[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 0}
					callbackArgs41[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 16}
					_ = d40
					callbackResultOff42 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff42))
					ctx.FreeDesc(&d39)
					ctx.StabilizeDescAcrossNestedCall(&d16)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs43 := ctx.StabilizeCallbackArgs(callbackArgs41)
						ctx.ReclaimUntrackedRegs()
						outerRegs44 := ctx.PreserveOuterRegs()
						d40 = JITEmitProcInlineWithOuter(ctx, &d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs43, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff42), ID: 0})
						ctx.RestoreOuterRegs(outerRegs44)
						ctx.ReclaimUntrackedRegs()
					} else {
						d45, knownBuiltin46 := jitEmitKnownDeclaration(ctx, d5, callbackArgs41, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff42), ID: 0})
						if knownBuiltin46 {
							d40 = d45
						} else {
							ctx.Coverage.DynamicCalls++
							d47 = jitCopyScmerToPair(ctx, d5)
							d40 = jitEmitDynamicCallableAt(ctx, d47, callbackArgs41, int32(stackArray7), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff42), ID: 0})
						}
					}
					if d14.Loc == LocRegPair || d14.Loc == LocStackPair || d14.Loc == LocRegTriple || d14.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d40 = JITPrepareScmerGoArg(ctx, d40)
					ctx.SyncDesc(&d14)
					ctx.SyncDesc(&d40)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d14, d40})
					ctx.FreeDesc(&d40)
					ctx.SyncDesc(&d16)
					if d16.Loc == LocReg || d16.Loc == LocFPReg {
						ctx.ProtectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.ProtectReg(d16.Reg)
						ctx.ProtectReg(d16.Reg2)
					}
					d48 = d16
					if d48.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d48)
					ctx.EmitStoreToStack(d48, int32(bbs[1].PhiBase)+int32(0))
					if d16.Loc == LocReg || d16.Loc == LocFPReg {
						ctx.UnprotectReg(d16.Reg)
					} else if d16.Loc == LocRegPair {
						ctx.UnprotectReg(d16.Reg)
						ctx.UnprotectReg(d16.Reg2)
					}
					return bbs[1].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d49 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d14.Reg, Reg2: r6}
						ctx.BindReg(d14.Reg, &d49)
						ctx.BindReg(r6, &d49)
						ctx.TransferReg(d14.Reg)
						ctx.BindReg(d14.Reg, &d49)
						ctx.BindReg(r6, &d49)
						d14.Loc = LocNone
					}
					ctx.SyncDesc(&d49)
					if d49.Loc == LocRegPair || d49.Loc == LocStackPair || d49.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d49, &result)
						result.Type = d49.Type
					} else {
						switch d49.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d49)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d49)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d49)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d49, &result)
							result.Type = d49.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
