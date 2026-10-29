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
				ctx.EnsureDesc(&d3)
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
					boxed := JITPrepareScmerGoArg(ctx, d3)
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
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
				inlineHome11 := d10
				inlineHome11.ID = 0
				d12 := d7
				_ = d12
				ctx.StabilizeDescForControlFlow(&d12)
				inlineHome13 := d12
				inlineHome13.ID = 0
				d14 := d8
				_ = d14
				ctx.StabilizeDescForControlFlow(&d14)
				inlineHome15 := d14
				inlineHome15.ID = 0
				phiBase16 := ctx.AllocStack(int32(120))
				d17 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				ctx.PreparePointerStackTarget(int32(phiBase16)+int32(0), 3)
				_ = d17
				d18 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				ctx.PreparePointerStackTarget(int32(phiBase16)+int32(24), 3)
				_ = d18
				d19 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				_ = d19
				d20 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				_ = d20
				d21 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				ctx.PreparePointerStackTarget(int32(phiBase16)+int32(80), 3)
				_ = d21
				d22 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				_ = d22
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
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d17 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				var d23 JITValueDesc
				if d14.Loc == LocImm {
					d23 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.Int() < 0)}
				} else {
					r0 := ctx.AllocRegExcept(d14.Reg)
					ctx.EmitCmpRegImm32(d14.Reg, 0)
					d23 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
					ctx.BindReg(r0, &d23)
				}
				ctx.ReclaimUntrackedRegs()
				d24 := d23
				ctx.SyncDesc(&d24)
				if d24.Loc != LocImm && d24.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl22 := ctx.ReserveLabel()
				lbl23 := ctx.ReserveLabel()
				if d24.Loc == LocImm {
					if d24.Imm.Bool() {
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl4)
					} else {
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl5)
					}
				} else {
					ctx.EmitJump(d24.Condition, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.FreeDesc(&d23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl5)
				}
				bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d17 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				stackArray25 := ctx.AllocStack(int32(1024))
				_ = stackArray25
				ctx.ReclaimUntrackedRegs()
				d26 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(64), SliceSizeKnown: true}
				_ = d26
				ctx.StabilizeDescForControlFlow(&d26)
				inlineHome27 := d26
				inlineHome27.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d10)
				ctx.EmitStoreScmerToStack(d10, int32(stackArray25)+int32(0))
				ctx.ReclaimUntrackedRegs()
				stackArray28 := ctx.AllocStack(int32(128))
				_ = stackArray28
				ctx.ReclaimUntrackedRegs()
				d29 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(8), SliceSizeKnown: true}
				_ = d29
				ctx.StabilizeDescForControlFlow(&d29)
				inlineHome30 := d29
				inlineHome30.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d26)
				if d26.Loc == LocReg || d26.Loc == LocFPReg {
					ctx.ProtectReg(d26.Reg)
				} else if d26.Loc == LocRegPair {
					ctx.ProtectReg(d26.Reg)
					ctx.ProtectReg(d26.Reg2)
				}
				ctx.SyncDesc(&d29)
				if d29.Loc == LocReg || d29.Loc == LocFPReg {
					ctx.ProtectReg(d29.Reg)
				} else if d29.Loc == LocRegPair {
					ctx.ProtectReg(d29.Reg)
					ctx.ProtectReg(d29.Reg2)
				}
				d31 := d26
				if d31.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d31)
				if d31.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d31, int32(phiBase16)+int32(0), 3)
				} else {
					if d31.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d31.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
					ctx.EmitStoreRegMem(d31.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
					ctx.EmitStoreRegMem(d31.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
				}
				d32 := d29
				if d32.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d32)
				if d32.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d32, int32(phiBase16)+int32(24), 3)
				} else {
					if d32.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d32.Reg, ctx.StackReg, int32(phiBase16)+int32(24))
					ctx.EmitStoreRegMem(d32.Reg2, ctx.StackReg, int32(phiBase16)+int32(24)+8)
					ctx.EmitStoreRegMem(d32.Reg3, ctx.StackReg, int32(phiBase16)+int32(24)+16)
				}
				if d26.Loc == LocReg || d26.Loc == LocFPReg {
					ctx.UnprotectReg(d26.Reg)
				} else if d26.Loc == LocRegPair {
					ctx.UnprotectReg(d26.Reg)
					ctx.UnprotectReg(d26.Reg2)
				}
				if d29.Loc == LocReg || d29.Loc == LocFPReg {
					ctx.UnprotectReg(d29.Reg)
				} else if d29.Loc == LocRegPair {
					ctx.UnprotectReg(d29.Reg)
					ctx.UnprotectReg(d29.Reg2)
				}
				bbpos_3_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d17 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d18 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d33 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d33)
				ctx.ReclaimUntrackedRegs()
				d34 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				ctx.StabilizeDescForControlFlow(&d34)
				ctx.ReclaimUntrackedRegs()
				var d35 JITValueDesc
				if d33.SliceSizeKnown {
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d33.KnownSliceLen))}
				} else if d33.Loc == LocImm {
					d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d33.StackOff))}
				} else if d33.Loc == LocStackTriple {
					d35 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d33.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocRegPair || d33.Loc == LocRegTriple {
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg2, ID: 0}
					} else if d33.Loc == LocReg {
						d35 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d35
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d35)
				var d36 JITValueDesc
				if d35.Loc == LocImm {
					d36 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d35.Imm.Int() != 0)}
				} else {
					r1 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d35.Reg, 0)
					d36 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r1, Condition: CondNotEqual}
					ctx.BindReg(r1, &d36)
				}
				ctx.FreeDesc(&d35)
				ctx.ReclaimUntrackedRegs()
				d37 := d36
				ctx.SyncDesc(&d37)
				if d37.Loc != LocImm && d37.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl24 := ctx.ReserveLabel()
				lbl25 := ctx.ReserveLabel()
				if d37.Loc == LocImm {
					if d37.Imm.Bool() {
						ctx.MarkLabel(lbl24)
						ctx.EmitJmp(lbl6)
					} else {
						ctx.MarkLabel(lbl25)
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d37.Condition, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.FreeDesc(&d36)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl6)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl7)
				}
				bbpos_3_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d38 JITValueDesc
				if d34.SliceSizeKnown {
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.KnownSliceLen))}
				} else if d34.Loc == LocImm {
					d38 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.StackOff))}
				} else if d34.Loc == LocStackTriple {
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d34.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocRegPair || d34.Loc == LocRegTriple {
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg2, ID: 0}
					} else if d34.Loc == LocReg {
						d38 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d38
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d38)
				ctx.EnsureDesc(&d38)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d38)
				ctx.EnsureDesc(&d38)
				callResults39 := JITEmitGoCallResults(ctx, GoFuncAddr(jitMakeScmerSlice), []JITValueDesc{d38, d38}, []uint8{3}, []uint8{1})
				d40 := callResults39[0]
				d40.Type = tagSlice
				ctx.FreeDesc(&d38)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d40)
				ctx.EnsureDesc(&d34)
				callResults41 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d40, d34}, []uint8{1}, []uint8{0})
				d42 := callResults41[0]
				d42.Type = tagInt
				ctx.ReclaimUntrackedRegs()
				d43 := ctx.EmitNewSliceFromGoSlice(&d40)
				ctx.ReclaimUntrackedRegs()
				r2 := ctx.AllocReg()
				r3 := ctx.AllocRegExcept(r2)
				d44 := JITValueDesc{Loc: LocRegPair, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d44)
				ctx.BindReg(r3, &d44)
				ctx.EmitMovPairToResult(&d43, &d44)
				ctx.EmitJmp(lbl2)
				bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d45 JITValueDesc
				if d33.SliceSizeKnown {
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d33.KnownSliceLen))}
				} else if d33.Loc == LocImm {
					d45 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d33.StackOff))}
				} else if d33.Loc == LocStackTriple {
					d45 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d33.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d33)
					if d33.Loc == LocRegPair || d33.Loc == LocRegTriple {
						d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg2, ID: 0}
					} else if d33.Loc == LocReg {
						d45 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d33.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d45
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d45)
				ctx.EnsureDesc(&d45)
				var d46 JITValueDesc
				if d45.Loc == LocImm {
					d46 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d45.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d45.Reg)
					ctx.EmitMovRegReg(scratch, d45.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d46 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d46)
				}
				if d46.Loc == LocReg && d45.Loc == LocReg && d46.Reg == d45.Reg {
					ctx.TransferReg(d45.Reg)
					d45.Loc = LocNone
				}
				ctx.FreeDesc(&d45)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d46)
				ctx.ReclaimUntrackedRegs()
				d48 := ctx.EmitSliceElementAddress(&d33, &d46, 16)
				ctx.EnsureDesc(&d48)
				r4 := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegMem(r4, d48.Reg, 8)
				ctx.EmitMovRegMem(d48.Reg, d48.Reg, 0)
				d47 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d48.Reg, Reg2: r4}
				ctx.BindReg(d48.Reg, &d47)
				ctx.BindReg(r4, &d47)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d47)
				d49 := d47
				_ = d49
				ctx.StabilizeDescForControlFlow(&d49)
				inlineHome50 := d49
				inlineHome50.ID = 0
				phiBase51 := ctx.AllocStack(int32(16))
				d52 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase51) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase51) + int32(0))
				_ = d52
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
				d49 = inlineHome50
				d52 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase51) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
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
				if d53.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d53, int32(phiBase51)+int32(0), 2)
				} else if d53.Loc == LocInputPair {
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreScmerToStack(d53, int32(phiBase51)+int32(0))
				} else if d53.Loc == LocRegPair || d53.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d53, int32(phiBase51)+int32(0))
				} else {
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreToStack(d53, int32(phiBase51)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase51)+int32(0))+8)
				}
				if d49.Loc == LocReg || d49.Loc == LocFPReg {
					ctx.UnprotectReg(d49.Reg)
				} else if d49.Loc == LocRegPair {
					ctx.UnprotectReg(d49.Reg)
					ctx.UnprotectReg(d49.Reg2)
				}
				bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl30)
				ctx.ResolveFixups()
				d49 = inlineHome50
				d52 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase51) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d54 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase51) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d54)
				ctx.ReclaimUntrackedRegs()
				d54 = JITPrepareScmerGoArg(ctx, d54)
				ctx.SyncDesc(&d54)
				d55 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d54}, 1)
				d55.NoHeapPointer = true
				ctx.EmitAndRegImm32(d55.Reg, 1)
				d55.Type = tagBool
				ctx.BindReg(d55.Reg, &d55)
				ctx.ReclaimUntrackedRegs()
				d56 := d55
				ctx.EnsureDesc(&d56)
				if d56.Loc != LocImm && d56.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl31 := ctx.ReserveLabel()
				lbl32 := ctx.ReserveLabel()
				if d56.Loc == LocImm {
					if d56.Imm.Bool() {
						ctx.MarkLabel(lbl31)
						ctx.EmitJmp(lbl28)
					} else {
						ctx.MarkLabel(lbl32)
						ctx.EmitJmp(lbl29)
					}
				} else {
					ctx.EmitCmpRegImm32(d56.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl31)
					ctx.EmitJmp(lbl32)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl28)
					ctx.MarkLabel(lbl32)
					ctx.EmitJmp(lbl29)
				}
				ctx.FreeDesc(&d55)
				bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl29)
				ctx.ResolveFixups()
				d49 = inlineHome50
				d54 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase51) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r5 := ctx.AllocReg()
				r6 := ctx.AllocRegExcept(r5)
				d57 := JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d57)
				ctx.BindReg(r6, &d57)
				ctx.EmitMovPairToResult(&d54, &d57)
				ctx.EmitJmp(lbl26)
				bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl28)
				ctx.ResolveFixups()
				d49 = inlineHome50
				d54 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase51) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d54 = JITPrepareScmerGoArg(ctx, d54)
				ctx.SyncDesc(&d54)
				d58 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d54}, 1)
				d58.NoHeapPointer = false
				ctx.BindReg(d58.Reg, &d58)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d59 JITValueDesc
				ctx.EnsureDesc(&d58)
				if d58.Loc == LocImm {
					fieldAddr := uintptr(d58.Imm.Int()) + 32
					r7 := ctx.AllocReg()
					r8 := ctx.AllocRegExcept(r7)
					ctx.EmitMovRegMem64(r7, fieldAddr)
					ctx.EmitMovRegMem64(r8, fieldAddr+8)
					d59 = JITValueDesc{Loc: LocRegPair, Reg: r7, Reg2: r8}
					ctx.BindReg(r7, &d59)
					ctx.BindReg(r8, &d59)
				} else {
					off := int32(32)
					baseReg := d58.Reg
					r9 := ctx.AllocRegExcept(baseReg)
					r10 := ctx.AllocRegExcept(baseReg, r9)
					ctx.EmitMovRegMem(r9, baseReg, off)
					ctx.EmitMovRegMem(r10, baseReg, off+8)
					d59 = JITValueDesc{Loc: LocRegPair, Reg: r9, Reg2: r10}
					ctx.BindReg(r9, &d59)
					ctx.BindReg(r10, &d59)
				}
				ctx.FreeDesc(&d58)
				ctx.StabilizeDescForControlFlow(&d59)
				inlineHome60 := d59
				inlineHome60.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d59)
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.ProtectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.ProtectReg(d59.Reg)
					ctx.ProtectReg(d59.Reg2)
				}
				d61 := d59
				if d61.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d61)
				if d61.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d61, int32(phiBase51)+int32(0), 2)
				} else if d61.Loc == LocInputPair {
					ctx.EnsureDesc(&d61)
					ctx.EmitStoreScmerToStack(d61, int32(phiBase51)+int32(0))
				} else if d61.Loc == LocRegPair || d61.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d61, int32(phiBase51)+int32(0))
				} else {
					ctx.EnsureDesc(&d61)
					ctx.EmitStoreToStack(d61, int32(phiBase51)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase51)+int32(0))+8)
				}
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.UnprotectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.UnprotectReg(d59.Reg)
					ctx.UnprotectReg(d59.Reg2)
				}
				ctx.EmitJmp(lbl30)
				ctx.MarkLabel(lbl26)
				d62 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d62)
				ctx.BindReg(r6, &d62)
				ctx.BindReg(r5, &d62)
				ctx.BindReg(r6, &d62)
				ctx.StabilizeDescForControlFlow(&d62)
				inlineHome63 := d62
				inlineHome63.ID = 0
				ctx.FreeDesc(&d47)
				ctx.ReclaimUntrackedRegs()
				d64 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d46)
				ctx.EnsureDesc(&d33)
				ctx.EnsureDesc(&d64)
				ctx.EnsureDesc(&d46)
				var d66 JITValueDesc
				if d46.Loc == LocImm && d64.Loc == LocImm {
					d66 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d46.Imm.Int() - d64.Imm.Int())}
				} else {
					r11 := ctx.AllocReg()
					if d46.Loc == LocImm {
						ctx.EmitMovRegImm64(r11, uint64(d46.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r11, d46.Reg)
					}
					if d64.Loc == LocImm {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d64.Imm.Int()))
						ctx.EmitSubInt64(r11, ctx.ScratchReg)
					} else {
						ctx.EmitSubInt64(r11, d64.Reg)
					}
					d66 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
					ctx.BindReg(r11, &d66)
				}
				var d67 JITValueDesc
				r12 := ctx.EmitSliceDataAfterLow(&d33, &d64, 16)
				d67 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r12}
				ctx.BindReg(r12, &d67)
				ctx.BindReg(r12, &d67)
				var d68 JITValueDesc
				var r13 Reg
				var r14 Reg
				ctx.SyncDesc(&d67)
				ctx.EnsureDesc(&d67)
				if d67.Loc == LocImm {
					r13 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r13, uint64(d67.Imm.Int()))
				} else {
					r13 = d67.Reg
				}
				ctx.ProtectReg(r13)
				ctx.SyncDesc(&d66)
				ctx.EnsureDesc(&d66)
				if d66.Loc == LocImm {
					r14 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r14, uint64(d66.Imm.Int()))
				} else {
					r14 = d66.Reg
				}
				ctx.ProtectReg(r14)
				r15 := ctx.EmitSliceCapAfterLow(&d33, &d64, r13, r14)
				ctx.UnprotectReg(r14)
				ctx.UnprotectReg(r13)
				d68 = JITValueDesc{Loc: LocRegTriple, Reg: r13, Reg2: r14, Reg3: r15}
				ctx.BindReg(r13, &d68)
				ctx.BindReg(r14, &d68)
				ctx.BindReg(r15, &d68)
				ctx.BindReg(r13, &d68)
				ctx.BindReg(r14, &d68)
				ctx.BindReg(r15, &d68)
				ctx.StabilizeDescForControlFlow(&d68)
				inlineHome69 := d68
				inlineHome69.ID = 0
				ctx.FreeDesc(&d46)
				ctx.ReclaimUntrackedRegs()
				d71 := d62
				d71.ID = 0
				d70 := ctx.EmitTagEqualsBorrowed(&d71, tagSlice, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				d72 := d70
				ctx.EnsureDesc(&d72)
				if d72.Loc != LocImm && d72.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl33 := ctx.ReserveLabel()
				lbl34 := ctx.ReserveLabel()
				if d72.Loc == LocImm {
					if d72.Imm.Bool() {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl34)
						ctx.SyncDesc(&d68)
						if d68.Loc == LocReg || d68.Loc == LocFPReg {
							ctx.ProtectReg(d68.Reg)
						} else if d68.Loc == LocRegPair {
							ctx.ProtectReg(d68.Reg)
							ctx.ProtectReg(d68.Reg2)
						}
						d73 := d68
						if d73.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d73)
						if d73.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d73, int32(phiBase16)+int32(0), 3)
						} else {
							if d73.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d73.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
							ctx.EmitStoreRegMem(d73.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
							ctx.EmitStoreRegMem(d73.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
						}
						if d68.Loc == LocReg || d68.Loc == LocFPReg {
							ctx.UnprotectReg(d68.Reg)
						} else if d68.Loc == LocRegPair {
							ctx.UnprotectReg(d68.Reg)
							ctx.UnprotectReg(d68.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitCmpRegImm32(d72.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl33)
					ctx.EmitJmp(lbl34)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl34)
					ctx.SyncDesc(&d68)
					if d68.Loc == LocReg || d68.Loc == LocFPReg {
						ctx.ProtectReg(d68.Reg)
					} else if d68.Loc == LocRegPair {
						ctx.ProtectReg(d68.Reg)
						ctx.ProtectReg(d68.Reg2)
					}
					d74 := d68
					if d74.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d74)
					if d74.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d74, int32(phiBase16)+int32(0), 3)
					} else {
						if d74.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d74.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
						ctx.EmitStoreRegMem(d74.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
						ctx.EmitStoreRegMem(d74.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
					}
					if d68.Loc == LocReg || d68.Loc == LocFPReg {
						ctx.UnprotectReg(d68.Reg)
					} else if d68.Loc == LocRegPair {
						ctx.UnprotectReg(d68.Reg)
						ctx.UnprotectReg(d68.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				ctx.FreeDesc(&d70)
				bbpos_3_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d75 := jitKnownSliceHeader(ctx, &d62)
				ctx.StabilizeDescForControlFlow(&d75)
				inlineHome76 := d75
				inlineHome76.ID = 0
				ctx.ReclaimUntrackedRegs()
				var d77 JITValueDesc
				if d75.SliceSizeKnown {
					d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.KnownSliceLen))}
				} else if d75.Loc == LocImm {
					d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.StackOff))}
				} else if d75.Loc == LocStackTriple {
					d77 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d75.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocRegPair || d75.Loc == LocRegTriple {
						d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg2, ID: 0}
					} else if d75.Loc == LocReg {
						d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d77
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d77)
				var d78 JITValueDesc
				if d77.Loc == LocImm {
					d78 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d77.Imm.Int() != 0)}
				} else {
					r16 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d77.Reg, 0)
					d78 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondNotEqual}
					ctx.BindReg(r16, &d78)
				}
				ctx.FreeDesc(&d77)
				ctx.ReclaimUntrackedRegs()
				d79 := d78
				ctx.SyncDesc(&d79)
				if d79.Loc != LocImm && d79.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl35 := ctx.ReserveLabel()
				lbl36 := ctx.ReserveLabel()
				if d79.Loc == LocImm {
					if d79.Imm.Bool() {
						ctx.MarkLabel(lbl35)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl36)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitJump(d79.Condition, lbl35)
					ctx.EmitJmp(lbl36)
					ctx.FreeDesc(&d78)
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl36)
					ctx.EmitJmp(lbl11)
				}
				bbpos_3_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				bbpos_3_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d80 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d80
				ctx.ReclaimUntrackedRegs()
				var d81 JITValueDesc
				if d75.SliceSizeKnown {
					d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.KnownSliceLen))}
				} else if d75.Loc == LocImm {
					d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.StackOff))}
				} else if d75.Loc == LocStackTriple {
					d81 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d75.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocRegPair || d75.Loc == LocRegTriple {
						d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg2, ID: 0}
					} else if d75.Loc == LocReg {
						d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d81
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d81)
				ctx.EnsureDesc(&d81)
				var d82 JITValueDesc
				if d81.Loc == LocImm {
					d82 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d81.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d81.Reg)
					ctx.EmitMovRegReg(scratch, d81.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d82 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d82)
				}
				if d82.Loc == LocReg && d81.Loc == LocReg && d82.Reg == d81.Reg {
					ctx.TransferReg(d81.Reg)
					d81.Loc = LocNone
				}
				ctx.EnsureDesc(&d82)
				ctx.EmitStoreToStack(d82, int32(phiBase16)+int32(104))
				ctx.StabilizeDescForControlFlow(&d82)
				inlineHome83 := d82
				inlineHome83.ID = 0
				ctx.FreeDesc(&d81)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d68)
				if d68.Loc == LocReg || d68.Loc == LocFPReg {
					ctx.ProtectReg(d68.Reg)
				} else if d68.Loc == LocRegPair {
					ctx.ProtectReg(d68.Reg)
					ctx.ProtectReg(d68.Reg2)
				}
				d84 := d68
				if d84.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d84)
				if d84.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d84, int32(phiBase16)+int32(80), 3)
				} else {
					if d84.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d84.Reg, ctx.StackReg, int32(phiBase16)+int32(80))
					ctx.EmitStoreRegMem(d84.Reg2, ctx.StackReg, int32(phiBase16)+int32(80)+8)
					ctx.EmitStoreRegMem(d84.Reg3, ctx.StackReg, int32(phiBase16)+int32(80)+16)
				}
				if d68.Loc == LocReg || d68.Loc == LocFPReg {
					ctx.UnprotectReg(d68.Reg)
				} else if d68.Loc == LocRegPair {
					ctx.UnprotectReg(d68.Reg)
					ctx.UnprotectReg(d68.Reg2)
				}
				bbpos_3_18 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl20)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d21 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d85 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				ctx.StabilizeDescForControlFlow(&d85)
				ctx.ReclaimUntrackedRegs()
				d86 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(104)}
				ctx.StabilizeDescForControlFlow(&d86)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d86)
				d88 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d88
				ctx.EnsureDescsTogether(&d86, &d88)
				var d87 JITValueDesc
				if d86.Loc == LocImm && d88.Loc == LocImm {
					d87 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d86.Imm.Int() >= d88.Imm.Int())}
				} else if d88.Loc == LocImm {
					r17 := ctx.AllocRegExcept(d86.Reg)
					if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d86.Reg, int32(d88.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d88.Imm.Int()))
						ctx.EmitCmpInt64(d86.Reg, ctx.ScratchReg)
					}
					d87 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r17, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r17, &d87)
				} else if d86.Loc == LocImm {
					r18 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d86.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d88.Reg)
					d87 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r18, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r18, &d87)
				} else {
					r19 := ctx.AllocRegExcept(d86.Reg)
					ctx.EmitCmpInt64(d86.Reg, d88.Reg)
					d87 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r19, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r19, &d87)
				}
				ctx.ReclaimUntrackedRegs()
				d89 := d87
				ctx.SyncDesc(&d89)
				if d89.Loc != LocImm && d89.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl37 := ctx.ReserveLabel()
				lbl38 := ctx.ReserveLabel()
				if d89.Loc == LocImm {
					if d89.Imm.Bool() {
						ctx.MarkLabel(lbl37)
						ctx.EmitJmp(lbl21)
					} else {
						ctx.MarkLabel(lbl38)
						ctx.SyncDesc(&d85)
						if d85.Loc == LocReg || d85.Loc == LocFPReg {
							ctx.ProtectReg(d85.Reg)
						} else if d85.Loc == LocRegPair {
							ctx.ProtectReg(d85.Reg)
							ctx.ProtectReg(d85.Reg2)
						}
						d90 := d85
						if d90.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d90)
						if d90.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d90, int32(phiBase16)+int32(0), 3)
						} else {
							if d90.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d90.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
							ctx.EmitStoreRegMem(d90.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
							ctx.EmitStoreRegMem(d90.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
						}
						if d85.Loc == LocReg || d85.Loc == LocFPReg {
							ctx.UnprotectReg(d85.Reg)
						} else if d85.Loc == LocRegPair {
							ctx.UnprotectReg(d85.Reg)
							ctx.UnprotectReg(d85.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitJump(d89.Condition, lbl37)
					ctx.EmitJmp(lbl38)
					ctx.FreeDesc(&d87)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl38)
					ctx.SyncDesc(&d85)
					if d85.Loc == LocReg || d85.Loc == LocFPReg {
						ctx.ProtectReg(d85.Reg)
					} else if d85.Loc == LocRegPair {
						ctx.ProtectReg(d85.Reg)
						ctx.ProtectReg(d85.Reg2)
					}
					d91 := d85
					if d91.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d91)
					if d91.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d91, int32(phiBase16)+int32(0), 3)
					} else {
						if d91.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d91.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
						ctx.EmitStoreRegMem(d91.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
						ctx.EmitStoreRegMem(d91.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
					}
					if d85.Loc == LocReg || d85.Loc == LocFPReg {
						ctx.UnprotectReg(d85.Reg)
					} else if d85.Loc == LocRegPair {
						ctx.UnprotectReg(d85.Reg)
						ctx.UnprotectReg(d85.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				bbpos_3_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d92 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d92
				ctx.ReclaimUntrackedRegs()
				d94 := ctx.EmitSliceElementAddress(&d75, &d92, 16)
				ctx.EnsureDesc(&d94)
				r20 := ctx.AllocRegExcept(d94.Reg)
				ctx.EmitMovRegMem(r20, d94.Reg, 8)
				ctx.EmitMovRegMem(d94.Reg, d94.Reg, 0)
				d93 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d94.Reg, Reg2: r20}
				ctx.BindReg(d94.Reg, &d93)
				ctx.BindReg(r20, &d93)
				ctx.ReclaimUntrackedRegs()
				d93 = JITPrepareScmerGoArg(ctx, d93)
				d12 = JITPrepareScmerGoArg(ctx, d12)
				ctx.SyncDesc(&d93)
				ctx.SyncDesc(&d12)
				d95 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d93, d12}, 1)
				d95.NoHeapPointer = true
				ctx.EmitAndRegImm32(d95.Reg, 1)
				d95.Type = tagBool
				ctx.BindReg(d95.Reg, &d95)
				ctx.FreeDesc(&d93)
				ctx.ReclaimUntrackedRegs()
				d96 := d95
				ctx.EnsureDesc(&d96)
				if d96.Loc != LocImm && d96.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl39 := ctx.ReserveLabel()
				lbl40 := ctx.ReserveLabel()
				if d96.Loc == LocImm {
					if d96.Imm.Bool() {
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl10)
					} else {
						ctx.MarkLabel(lbl40)
						ctx.EmitJmp(lbl11)
					}
				} else {
					ctx.EmitCmpRegImm32(d96.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl39)
					ctx.EmitJmp(lbl40)
					ctx.MarkLabel(lbl39)
					ctx.EmitJmp(lbl10)
					ctx.MarkLabel(lbl40)
					ctx.EmitJmp(lbl11)
				}
				ctx.FreeDesc(&d95)
				bbpos_3_19 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl21)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d86)
				ctx.ReclaimUntrackedRegs()
				d98 := ctx.EmitSliceElementAddress(&d75, &d86, 16)
				ctx.EnsureDesc(&d98)
				r21 := ctx.AllocRegExcept(d98.Reg)
				ctx.EmitMovRegMem(r21, d98.Reg, 8)
				ctx.EmitMovRegMem(d98.Reg, d98.Reg, 0)
				d97 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d98.Reg, Reg2: r21}
				ctx.BindReg(d98.Reg, &d97)
				ctx.BindReg(r21, &d97)
				ctx.ReclaimUntrackedRegs()
				stackArray99 := ctx.AllocStack(int32(16))
				_ = stackArray99
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d97)
				ctx.EmitStoreScmerToStack(d97, int32(stackArray99)+int32(0))
				ctx.FreeDesc(&d97)
				ctx.ReclaimUntrackedRegs()
				d100 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d100
				ctx.ReclaimUntrackedRegs()
				r22 := ctx.AllocReg()
				r23 := ctx.AllocRegExcept(r22)
				r24 := ctx.AllocRegExcept(r22, r23)
				d101 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r22, Reg2: r23, Reg3: r24}
				ctx.BindReg(r22, &d101)
				ctx.BindReg(r23, &d101)
				ctx.BindReg(r24, &d101)
				ctx.BindReg(r22, &d101)
				ctx.BindReg(r23, &d101)
				ctx.BindReg(r24, &d101)
				ctx.EmitLeaRegMem(d101.Reg, ctx.StackReg, int32(stackArray99))
				ctx.EmitMovRegImm64(d101.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d101.Reg3, uint64(1))
				callResults102 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d85, d101}, []uint8{3}, []uint8{1})
				d103 := callResults102[0]
				d104 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase16) + int32(80)}
				ctx.EmitCopyDescWords(&d104, &d103, 3)
				ctx.FreeDesc(&d103)
				d103 = d104
				ctx.StabilizeDescForControlFlow(&d103)
				inlineHome105 := d103
				inlineHome105.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d86)
				ctx.EnsureDesc(&d86)
				var d106 JITValueDesc
				if d86.Loc == LocImm {
					d106 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d86.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d86.Reg)
					ctx.EmitMovRegReg(scratch, d86.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d106 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d106)
				}
				if d106.Loc == LocReg && d86.Loc == LocReg && d106.Reg == d86.Reg {
					ctx.TransferReg(d86.Reg)
					d86.Loc = LocNone
				}
				ctx.EnsureDesc(&d106)
				ctx.EmitStoreToStack(d106, int32(phiBase16)+int32(104))
				ctx.StabilizeDescForControlFlow(&d106)
				inlineHome107 := d106
				inlineHome107.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl20)
				bbpos_3_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d103 = inlineHome105
				d106 = inlineHome107
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d108 JITValueDesc
				if d75.SliceSizeKnown {
					d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.KnownSliceLen))}
				} else if d75.Loc == LocImm {
					d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d75.StackOff))}
				} else if d75.Loc == LocStackTriple {
					d108 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d75.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d75)
					if d75.Loc == LocRegPair || d75.Loc == LocRegTriple {
						d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg2, ID: 0}
					} else if d75.Loc == LocReg {
						d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d75.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d108
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d108)
				ctx.EnsureDesc(&d14)
				ctx.EnsureDescsTogether(&d108, &d14)
				var d109 JITValueDesc
				if d108.Loc == LocImm && d14.Loc == LocImm {
					d109 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d108.Imm.Int() > d14.Imm.Int())}
				} else if d14.Loc == LocImm {
					r25 := ctx.AllocReg()
					if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d108.Reg, int32(d14.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d14.Imm.Int()))
						ctx.EmitCmpInt64(d108.Reg, ctx.ScratchReg)
					}
					d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r25, Condition: CondSignedGreater}
					ctx.BindReg(r25, &d109)
				} else if d108.Loc == LocImm {
					r26 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d108.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d14.Reg)
					d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedGreater}
					ctx.BindReg(r26, &d109)
				} else {
					r27 := ctx.AllocReg()
					ctx.EmitCmpInt64(d108.Reg, d14.Reg)
					d109 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedGreater}
					ctx.BindReg(r27, &d109)
				}
				ctx.FreeDesc(&d108)
				ctx.ReclaimUntrackedRegs()
				d110 := d109
				ctx.SyncDesc(&d110)
				if d110.Loc != LocImm && d110.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl41 := ctx.ReserveLabel()
				lbl42 := ctx.ReserveLabel()
				if d110.Loc == LocImm {
					if d110.Imm.Bool() {
						ctx.MarkLabel(lbl41)
						ctx.EmitJmp(lbl13)
					} else {
						ctx.MarkLabel(lbl42)
						ctx.SyncDesc(&d68)
						if d68.Loc == LocReg || d68.Loc == LocFPReg {
							ctx.ProtectReg(d68.Reg)
						} else if d68.Loc == LocRegPair {
							ctx.ProtectReg(d68.Reg)
							ctx.ProtectReg(d68.Reg2)
						}
						d111 := d68
						if d111.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d111)
						if d111.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d111, int32(phiBase16)+int32(0), 3)
						} else {
							if d111.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d111.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
							ctx.EmitStoreRegMem(d111.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
							ctx.EmitStoreRegMem(d111.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
						}
						if d68.Loc == LocReg || d68.Loc == LocFPReg {
							ctx.UnprotectReg(d68.Reg)
						} else if d68.Loc == LocRegPair {
							ctx.UnprotectReg(d68.Reg)
							ctx.UnprotectReg(d68.Reg2)
						}
						ctx.EmitJmp(lbl8)
					}
				} else {
					ctx.EmitJump(d110.Condition, lbl41)
					ctx.EmitJmp(lbl42)
					ctx.FreeDesc(&d109)
					ctx.MarkLabel(lbl41)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl42)
					ctx.SyncDesc(&d68)
					if d68.Loc == LocReg || d68.Loc == LocFPReg {
						ctx.ProtectReg(d68.Reg)
					} else if d68.Loc == LocRegPair {
						ctx.ProtectReg(d68.Reg)
						ctx.ProtectReg(d68.Reg2)
					}
					d112 := d68
					if d112.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d112)
					if d112.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d112, int32(phiBase16)+int32(0), 3)
					} else {
						if d112.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d112.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
						ctx.EmitStoreRegMem(d112.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
						ctx.EmitStoreRegMem(d112.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
					}
					if d68.Loc == LocReg || d68.Loc == LocFPReg {
						ctx.UnprotectReg(d68.Reg)
					} else if d68.Loc == LocRegPair {
						ctx.UnprotectReg(d68.Reg)
						ctx.UnprotectReg(d68.Reg2)
					}
					ctx.EmitJmp(lbl8)
				}
				bbpos_3_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d103 = inlineHome105
				d106 = inlineHome107
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.ReclaimUntrackedRegs()
				d114 := ctx.EmitSliceElementAddress(&d75, &d14, 16)
				ctx.EnsureDesc(&d114)
				r28 := ctx.AllocRegExcept(d114.Reg)
				ctx.EmitMovRegMem(r28, d114.Reg, 8)
				ctx.EmitMovRegMem(d114.Reg, d114.Reg, 0)
				d113 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d114.Reg, Reg2: r28}
				ctx.BindReg(d114.Reg, &d113)
				ctx.BindReg(r28, &d113)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d113)
				d115 := d113
				_ = d115
				ctx.StabilizeDescForControlFlow(&d115)
				inlineHome116 := d115
				inlineHome116.ID = 0
				phiBase117 := ctx.AllocStack(int32(16))
				d118 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase117) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase117) + int32(0))
				_ = d118
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
				d115 = inlineHome116
				d118 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase117) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d115)
				if d115.Loc == LocReg || d115.Loc == LocFPReg {
					ctx.ProtectReg(d115.Reg)
				} else if d115.Loc == LocRegPair {
					ctx.ProtectReg(d115.Reg)
					ctx.ProtectReg(d115.Reg2)
				}
				d119 := d115
				if d119.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d119)
				if d119.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d119, int32(phiBase117)+int32(0), 2)
				} else if d119.Loc == LocInputPair {
					ctx.EnsureDesc(&d119)
					ctx.EmitStoreScmerToStack(d119, int32(phiBase117)+int32(0))
				} else if d119.Loc == LocRegPair || d119.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d119, int32(phiBase117)+int32(0))
				} else {
					ctx.EnsureDesc(&d119)
					ctx.EmitStoreToStack(d119, int32(phiBase117)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase117)+int32(0))+8)
				}
				if d115.Loc == LocReg || d115.Loc == LocFPReg {
					ctx.UnprotectReg(d115.Reg)
				} else if d115.Loc == LocRegPair {
					ctx.UnprotectReg(d115.Reg)
					ctx.UnprotectReg(d115.Reg2)
				}
				bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl47)
				ctx.ResolveFixups()
				d115 = inlineHome116
				d118 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase117) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d120 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase117) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d120)
				ctx.ReclaimUntrackedRegs()
				d120 = JITPrepareScmerGoArg(ctx, d120)
				ctx.SyncDesc(&d120)
				d121 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d120}, 1)
				d121.NoHeapPointer = true
				ctx.EmitAndRegImm32(d121.Reg, 1)
				d121.Type = tagBool
				ctx.BindReg(d121.Reg, &d121)
				ctx.ReclaimUntrackedRegs()
				d122 := d121
				ctx.EnsureDesc(&d122)
				if d122.Loc != LocImm && d122.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl48 := ctx.ReserveLabel()
				lbl49 := ctx.ReserveLabel()
				if d122.Loc == LocImm {
					if d122.Imm.Bool() {
						ctx.MarkLabel(lbl48)
						ctx.EmitJmp(lbl45)
					} else {
						ctx.MarkLabel(lbl49)
						ctx.EmitJmp(lbl46)
					}
				} else {
					ctx.EmitCmpRegImm32(d122.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl48)
					ctx.EmitJmp(lbl49)
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl45)
					ctx.MarkLabel(lbl49)
					ctx.EmitJmp(lbl46)
				}
				ctx.FreeDesc(&d121)
				bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl46)
				ctx.ResolveFixups()
				d115 = inlineHome116
				d120 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase117) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r29 := ctx.AllocReg()
				r30 := ctx.AllocRegExcept(r29)
				d123 := JITValueDesc{Loc: LocRegPair, Reg: r29, Reg2: r30}
				ctx.BindReg(r29, &d123)
				ctx.BindReg(r30, &d123)
				ctx.EmitMovPairToResult(&d120, &d123)
				ctx.EmitJmp(lbl43)
				bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl45)
				ctx.ResolveFixups()
				d115 = inlineHome116
				d120 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase117) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d120 = JITPrepareScmerGoArg(ctx, d120)
				ctx.SyncDesc(&d120)
				d124 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d120}, 1)
				d124.NoHeapPointer = false
				ctx.BindReg(d124.Reg, &d124)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d125 JITValueDesc
				ctx.EnsureDesc(&d124)
				if d124.Loc == LocImm {
					fieldAddr := uintptr(d124.Imm.Int()) + 32
					r31 := ctx.AllocReg()
					r32 := ctx.AllocRegExcept(r31)
					ctx.EmitMovRegMem64(r31, fieldAddr)
					ctx.EmitMovRegMem64(r32, fieldAddr+8)
					d125 = JITValueDesc{Loc: LocRegPair, Reg: r31, Reg2: r32}
					ctx.BindReg(r31, &d125)
					ctx.BindReg(r32, &d125)
				} else {
					off := int32(32)
					baseReg := d124.Reg
					r33 := ctx.AllocRegExcept(baseReg)
					r34 := ctx.AllocRegExcept(baseReg, r33)
					ctx.EmitMovRegMem(r33, baseReg, off)
					ctx.EmitMovRegMem(r34, baseReg, off+8)
					d125 = JITValueDesc{Loc: LocRegPair, Reg: r33, Reg2: r34}
					ctx.BindReg(r33, &d125)
					ctx.BindReg(r34, &d125)
				}
				ctx.FreeDesc(&d124)
				ctx.StabilizeDescForControlFlow(&d125)
				inlineHome126 := d125
				inlineHome126.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d125)
				if d125.Loc == LocReg || d125.Loc == LocFPReg {
					ctx.ProtectReg(d125.Reg)
				} else if d125.Loc == LocRegPair {
					ctx.ProtectReg(d125.Reg)
					ctx.ProtectReg(d125.Reg2)
				}
				d127 := d125
				if d127.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d127)
				if d127.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d127, int32(phiBase117)+int32(0), 2)
				} else if d127.Loc == LocInputPair {
					ctx.EnsureDesc(&d127)
					ctx.EmitStoreScmerToStack(d127, int32(phiBase117)+int32(0))
				} else if d127.Loc == LocRegPair || d127.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d127, int32(phiBase117)+int32(0))
				} else {
					ctx.EnsureDesc(&d127)
					ctx.EmitStoreToStack(d127, int32(phiBase117)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase117)+int32(0))+8)
				}
				if d125.Loc == LocReg || d125.Loc == LocFPReg {
					ctx.UnprotectReg(d125.Reg)
				} else if d125.Loc == LocRegPair {
					ctx.UnprotectReg(d125.Reg)
					ctx.UnprotectReg(d125.Reg2)
				}
				ctx.EmitJmp(lbl47)
				ctx.MarkLabel(lbl43)
				d128 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r29, Reg2: r30}
				ctx.BindReg(r29, &d128)
				ctx.BindReg(r30, &d128)
				ctx.BindReg(r29, &d128)
				ctx.BindReg(r30, &d128)
				ctx.StabilizeDescForControlFlow(&d128)
				inlineHome129 := d128
				inlineHome129.ID = 0
				ctx.FreeDesc(&d113)
				ctx.ReclaimUntrackedRegs()
				var d130 JITValueDesc
				if d34.SliceSizeKnown {
					d130 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.KnownSliceLen))}
				} else if d34.Loc == LocImm {
					d130 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d34.StackOff))}
				} else if d34.Loc == LocStackTriple {
					d130 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d34.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d34)
					if d34.Loc == LocRegPair || d34.Loc == LocRegTriple {
						d130 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg2, ID: 0}
					} else if d34.Loc == LocReg {
						d130 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d34.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d130
				ctx.StabilizeDescForControlFlow(&d130)
				inlineHome131 := d130
				inlineHome131.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase16)+int32(48))
				bbpos_3_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d103 = inlineHome105
				d106 = inlineHome107
				d128 = inlineHome129
				d130 = inlineHome131
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d19 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d132 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(48)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d132)
				ctx.EnsureDesc(&d132)
				var d133 JITValueDesc
				if d132.Loc == LocImm {
					d133 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d132.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d132.Reg)
					ctx.EmitMovRegReg(scratch, d132.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d133 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d133)
				}
				if d133.Loc == LocReg && d132.Loc == LocReg && d133.Reg == d132.Reg {
					ctx.TransferReg(d132.Reg)
					d132.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d133)
				inlineHome134 := d133
				inlineHome134.ID = 0
				ctx.FreeDesc(&d132)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d133)
				ctx.EnsureDesc(&d130)
				ctx.EnsureDescsTogether(&d133, &d130)
				var d135 JITValueDesc
				if d133.Loc == LocImm && d130.Loc == LocImm {
					d135 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d133.Imm.Int() < d130.Imm.Int())}
				} else if d130.Loc == LocImm {
					r35 := ctx.AllocRegExcept(d133.Reg)
					if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d133.Reg, int32(d130.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d130.Imm.Int()))
						ctx.EmitCmpInt64(d133.Reg, ctx.ScratchReg)
					}
					d135 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r35, Condition: CondSignedLess}
					ctx.BindReg(r35, &d135)
				} else if d133.Loc == LocImm {
					r36 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d133.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d130.Reg)
					d135 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r36, Condition: CondSignedLess}
					ctx.BindReg(r36, &d135)
				} else {
					r37 := ctx.AllocRegExcept(d133.Reg)
					ctx.EmitCmpInt64(d133.Reg, d130.Reg)
					d135 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r37, Condition: CondSignedLess}
					ctx.BindReg(r37, &d135)
				}
				ctx.ReclaimUntrackedRegs()
				d136 := d135
				ctx.SyncDesc(&d136)
				if d136.Loc != LocImm && d136.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl50 := ctx.ReserveLabel()
				lbl51 := ctx.ReserveLabel()
				if d136.Loc == LocImm {
					if d136.Imm.Bool() {
						ctx.MarkLabel(lbl50)
						ctx.EmitJmp(lbl15)
					} else {
						ctx.MarkLabel(lbl51)
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase16)+int32(64))
						ctx.EmitJmp(lbl16)
					}
				} else {
					ctx.EmitJump(d136.Condition, lbl50)
					ctx.EmitJmp(lbl51)
					ctx.FreeDesc(&d135)
					ctx.MarkLabel(lbl50)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl51)
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(0)}, int32(phiBase16)+int32(64))
					ctx.EmitJmp(lbl16)
				}
				bbpos_3_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d103 = inlineHome105
				d106 = inlineHome107
				d128 = inlineHome129
				d130 = inlineHome131
				d133 = inlineHome134
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d132 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d20 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d137 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(64)}
				ctx.ReclaimUntrackedRegs()
				d138 := d137
				ctx.EnsureDesc(&d138)
				if d138.Loc != LocImm && d138.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl52 := ctx.ReserveLabel()
				lbl53 := ctx.ReserveLabel()
				if d138.Loc == LocImm {
					if d138.Imm.Bool() {
						ctx.MarkLabel(lbl52)
						ctx.SyncDesc(&d68)
						if d68.Loc == LocReg || d68.Loc == LocFPReg {
							ctx.ProtectReg(d68.Reg)
						} else if d68.Loc == LocRegPair {
							ctx.ProtectReg(d68.Reg)
							ctx.ProtectReg(d68.Reg2)
						}
						d139 := d68
						if d139.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d139)
						if d139.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d139, int32(phiBase16)+int32(0), 3)
						} else {
							if d139.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d139.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
							ctx.EmitStoreRegMem(d139.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
							ctx.EmitStoreRegMem(d139.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
						}
						if d68.Loc == LocReg || d68.Loc == LocFPReg {
							ctx.UnprotectReg(d68.Reg)
						} else if d68.Loc == LocRegPair {
							ctx.UnprotectReg(d68.Reg)
							ctx.UnprotectReg(d68.Reg2)
						}
						ctx.EmitJmp(lbl8)
					} else {
						ctx.MarkLabel(lbl53)
						ctx.EmitJmp(lbl18)
					}
				} else {
					ctx.EmitCmpRegImm32(d138.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl52)
					ctx.EmitJmp(lbl53)
					ctx.MarkLabel(lbl52)
					ctx.SyncDesc(&d68)
					if d68.Loc == LocReg || d68.Loc == LocFPReg {
						ctx.ProtectReg(d68.Reg)
					} else if d68.Loc == LocRegPair {
						ctx.ProtectReg(d68.Reg)
						ctx.ProtectReg(d68.Reg2)
					}
					d140 := d68
					if d140.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d140)
					if d140.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d140, int32(phiBase16)+int32(0), 3)
					} else {
						if d140.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d140.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
						ctx.EmitStoreRegMem(d140.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
						ctx.EmitStoreRegMem(d140.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
					}
					if d68.Loc == LocReg || d68.Loc == LocFPReg {
						ctx.UnprotectReg(d68.Reg)
					} else if d68.Loc == LocRegPair {
						ctx.UnprotectReg(d68.Reg)
						ctx.UnprotectReg(d68.Reg2)
					}
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl53)
					ctx.EmitJmp(lbl18)
				}
				ctx.FreeDesc(&d137)
				bbpos_3_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d103 = inlineHome105
				d106 = inlineHome107
				d128 = inlineHome129
				d130 = inlineHome131
				d133 = inlineHome134
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d132 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d137 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d133)
				ctx.ReclaimUntrackedRegs()
				d142 := ctx.EmitSliceElementAddress(&d34, &d133, 16)
				ctx.EnsureDesc(&d142)
				r38 := ctx.AllocRegExcept(d142.Reg)
				ctx.EmitMovRegMem(r38, d142.Reg, 8)
				ctx.EmitMovRegMem(d142.Reg, d142.Reg, 0)
				d141 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d142.Reg, Reg2: r38}
				ctx.BindReg(d142.Reg, &d141)
				ctx.BindReg(r38, &d141)
				ctx.ReclaimUntrackedRegs()
				d141 = JITPrepareScmerGoArg(ctx, d141)
				d128 = JITPrepareScmerGoArg(ctx, d128)
				ctx.SyncDesc(&d141)
				ctx.SyncDesc(&d128)
				d143 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d141, d128}, 1)
				d143.NoHeapPointer = true
				ctx.EmitAndRegImm32(d143.Reg, 1)
				d143.Type = tagBool
				ctx.BindReg(d143.Reg, &d143)
				ctx.FreeDesc(&d141)
				ctx.ReclaimUntrackedRegs()
				d144 := d143
				ctx.EnsureDesc(&d144)
				if d144.Loc != LocImm && d144.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl54 := ctx.ReserveLabel()
				lbl55 := ctx.ReserveLabel()
				if d144.Loc == LocImm {
					if d144.Imm.Bool() {
						ctx.MarkLabel(lbl54)
						ctx.EmitJmp(lbl17)
					} else {
						ctx.MarkLabel(lbl55)
						ctx.SyncDesc(&d133)
						if d133.Loc == LocReg || d133.Loc == LocFPReg {
							ctx.ProtectReg(d133.Reg)
						} else if d133.Loc == LocRegPair {
							ctx.ProtectReg(d133.Reg)
							ctx.ProtectReg(d133.Reg2)
						}
						d145 := d133
						if d145.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d145)
						ctx.EmitStoreToStack(d145, int32(phiBase16)+int32(48))
						if d133.Loc == LocReg || d133.Loc == LocFPReg {
							ctx.UnprotectReg(d133.Reg)
						} else if d133.Loc == LocRegPair {
							ctx.UnprotectReg(d133.Reg)
							ctx.UnprotectReg(d133.Reg2)
						}
						ctx.EmitJmp(lbl14)
					}
				} else {
					ctx.EmitCmpRegImm32(d144.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl54)
					ctx.EmitJmp(lbl55)
					ctx.MarkLabel(lbl54)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl55)
					ctx.SyncDesc(&d133)
					if d133.Loc == LocReg || d133.Loc == LocFPReg {
						ctx.ProtectReg(d133.Reg)
					} else if d133.Loc == LocRegPair {
						ctx.ProtectReg(d133.Reg)
						ctx.ProtectReg(d133.Reg2)
					}
					d146 := d133
					if d146.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d146)
					ctx.EmitStoreToStack(d146, int32(phiBase16)+int32(48))
					if d133.Loc == LocReg || d133.Loc == LocFPReg {
						ctx.UnprotectReg(d133.Reg)
					} else if d133.Loc == LocRegPair {
						ctx.UnprotectReg(d133.Reg)
						ctx.UnprotectReg(d133.Reg2)
					}
					ctx.EmitJmp(lbl14)
				}
				ctx.FreeDesc(&d143)
				bbpos_3_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d103 = inlineHome105
				d106 = inlineHome107
				d128 = inlineHome129
				d130 = inlineHome131
				d133 = inlineHome134
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d132 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d137 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				stackArray147 := ctx.AllocStack(int32(16))
				_ = stackArray147
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d128)
				ctx.EmitStoreScmerToStack(d128, int32(stackArray147)+int32(0))
				ctx.ReclaimUntrackedRegs()
				d148 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d148
				ctx.ReclaimUntrackedRegs()
				r39 := ctx.AllocReg()
				r40 := ctx.AllocRegExcept(r39)
				r41 := ctx.AllocRegExcept(r39, r40)
				d149 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r39, Reg2: r40, Reg3: r41}
				ctx.BindReg(r39, &d149)
				ctx.BindReg(r40, &d149)
				ctx.BindReg(r41, &d149)
				ctx.BindReg(r39, &d149)
				ctx.BindReg(r40, &d149)
				ctx.BindReg(r41, &d149)
				ctx.EmitLeaRegMem(d149.Reg, ctx.StackReg, int32(stackArray147))
				ctx.EmitMovRegImm64(d149.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d149.Reg3, uint64(1))
				callResults150 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d34, d149}, []uint8{3}, []uint8{1})
				d151 := callResults150[0]
				d152 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase16) + int32(24)}
				ctx.EmitCopyDescWords(&d152, &d151, 3)
				ctx.FreeDesc(&d151)
				d151 = d152
				ctx.StabilizeDescForControlFlow(&d151)
				inlineHome153 := d151
				inlineHome153.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d68)
				if d68.Loc == LocReg || d68.Loc == LocFPReg {
					ctx.ProtectReg(d68.Reg)
				} else if d68.Loc == LocRegPair {
					ctx.ProtectReg(d68.Reg)
					ctx.ProtectReg(d68.Reg2)
				}
				d154 := d68
				if d154.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d154)
				if d154.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d154, int32(phiBase16)+int32(0), 3)
				} else {
					if d154.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d154.Reg, ctx.StackReg, int32(phiBase16)+int32(0))
					ctx.EmitStoreRegMem(d154.Reg2, ctx.StackReg, int32(phiBase16)+int32(0)+8)
					ctx.EmitStoreRegMem(d154.Reg3, ctx.StackReg, int32(phiBase16)+int32(0)+16)
				}
				if d68.Loc == LocReg || d68.Loc == LocFPReg {
					ctx.UnprotectReg(d68.Reg)
				} else if d68.Loc == LocRegPair {
					ctx.UnprotectReg(d68.Reg)
					ctx.UnprotectReg(d68.Reg2)
				}
				ctx.EmitJmp(lbl8)
				bbpos_3_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
				d10 = inlineHome11
				d12 = inlineHome13
				d14 = inlineHome15
				d26 = inlineHome27
				d29 = inlineHome30
				d62 = inlineHome63
				d68 = inlineHome69
				d75 = inlineHome76
				d82 = inlineHome83
				d103 = inlineHome105
				d106 = inlineHome107
				d128 = inlineHome129
				d130 = inlineHome131
				d133 = inlineHome134
				d151 = inlineHome153
				d33 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(0)}
				d34 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(24)}
				d132 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(48)}
				d137 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(phiBase16) + int32(64)}
				d85 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase16) + int32(80)}
				d86 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase16) + int32(104)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(phiBase16)+int32(64))
				ctx.EmitJmp(lbl16)
				ctx.MarkLabel(lbl2)
				d155 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d155)
				ctx.BindReg(r3, &d155)
				ctx.BindReg(r2, &d155)
				ctx.BindReg(r3, &d155)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d155)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				ctx.FreeDesc(&d4)
				if d155.Loc == LocImm {
					if result.Loc == LocAny {
						return d155
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d155)
				if d155.Loc == LocRegPair || d155.Loc == LocStackPair || d155.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d155, &result)
					result.Type = d155.Type
				} else {
					switch d155.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d155)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d155)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d155)
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
				ctx.EnsureDesc(&d3)
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
					boxed := JITPrepareScmerGoArg(ctx, d3)
					d4 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{boxed}, 1)
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
				inlineHome10 := d9
				inlineHome10.ID = 0
				d11 := d1
				_ = d11
				ctx.StabilizeDescForControlFlow(&d11)
				inlineHome12 := d11
				inlineHome12.ID = 0
				d13 := d4
				_ = d13
				ctx.StabilizeDescForControlFlow(&d13)
				inlineHome14 := d13
				inlineHome14.ID = 0
				d15 := d6
				_ = d15
				ctx.StabilizeDescForControlFlow(&d15)
				inlineHome16 := d15
				inlineHome16.ID = 0
				d17 := d8
				_ = d17
				ctx.StabilizeDescForControlFlow(&d17)
				inlineHome18 := d17
				inlineHome18.ID = 0
				phiBase19 := ctx.AllocStack(int32(96))
				d20 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				ctx.PreparePointerStackTarget(int32(phiBase19)+int32(0), 3)
				_ = d20
				d21 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				ctx.PrepareScmerStackTarget(int32(phiBase19) + int32(24))
				_ = d21
				d22 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				_ = d22
				d23 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				ctx.PreparePointerStackTarget(int32(phiBase19)+int32(56), 3)
				_ = d23
				d24 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				_ = d24
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
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				var d25 JITValueDesc
				if d13.Loc == LocImm {
					d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < 0)}
				} else {
					r0 := ctx.AllocRegExcept(d13.Reg)
					ctx.EmitCmpRegImm32(d13.Reg, 0)
					d25 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
					ctx.BindReg(r0, &d25)
				}
				ctx.ReclaimUntrackedRegs()
				d26 := d25
				ctx.SyncDesc(&d26)
				if d26.Loc != LocImm && d26.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl20 := ctx.ReserveLabel()
				lbl21 := ctx.ReserveLabel()
				if d26.Loc == LocImm {
					if d26.Imm.Bool() {
						ctx.MarkLabel(lbl20)
						ctx.EmitJmp(lbl3)
					} else {
						ctx.MarkLabel(lbl21)
						ctx.EmitJmp(lbl4)
					}
				} else {
					ctx.EmitJump(d26.Condition, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.FreeDesc(&d25)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl4)
				}
				bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				stackArray27 := ctx.AllocStack(int32(1024))
				_ = stackArray27
				ctx.ReclaimUntrackedRegs()
				d28 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(64), SliceSizeKnown: true}
				_ = d28
				ctx.StabilizeDescForControlFlow(&d28)
				inlineHome29 := d28
				inlineHome29.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d9)
				ctx.EmitStoreScmerToStack(d9, int32(stackArray27)+int32(0))
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d28)
				if d28.Loc == LocReg || d28.Loc == LocFPReg {
					ctx.ProtectReg(d28.Reg)
				} else if d28.Loc == LocRegPair {
					ctx.ProtectReg(d28.Reg)
					ctx.ProtectReg(d28.Reg2)
				}
				d30 := d28
				if d30.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d30)
				if d30.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d30, int32(phiBase19)+int32(0), 3)
				} else {
					if d30.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d30.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
					ctx.EmitStoreRegMem(d30.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
					ctx.EmitStoreRegMem(d30.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
				}
				if d28.Loc == LocReg || d28.Loc == LocFPReg {
					ctx.UnprotectReg(d28.Reg)
				} else if d28.Loc == LocRegPair {
					ctx.UnprotectReg(d28.Reg)
					ctx.UnprotectReg(d28.Reg2)
				}
				bbpos_2_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d20 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d31 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d31)
				ctx.ReclaimUntrackedRegs()
				var d32 JITValueDesc
				if d31.SliceSizeKnown {
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.KnownSliceLen))}
				} else if d31.Loc == LocImm {
					d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.StackOff))}
				} else if d31.Loc == LocStackTriple {
					d32 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d31.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocRegTriple {
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2, ID: 0}
					} else if d31.Loc == LocReg {
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d32
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
				ctx.SyncDesc(&d34)
				if d34.Loc != LocImm && d34.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl22 := ctx.ReserveLabel()
				lbl23 := ctx.ReserveLabel()
				if d34.Loc == LocImm {
					if d34.Imm.Bool() {
						ctx.MarkLabel(lbl22)
						ctx.EmitJmp(lbl5)
					} else {
						ctx.MarkLabel(lbl23)
						ctx.EmitJmp(lbl6)
					}
				} else {
					ctx.EmitJump(d34.Condition, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.FreeDesc(&d33)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl5)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl6)
				}
				bbpos_2_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r2 := ctx.AllocReg()
				d35 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(false)}
				_ = d35
				ctx.EnsureDesc(&d35)
				if d35.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r2, d35)
				}
				ctx.EmitJmp(lbl1)
				bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.EmitGoPanic("jit: invalid arguments for inlined Go helper")
				bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d36 JITValueDesc
				if d31.SliceSizeKnown {
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.KnownSliceLen))}
				} else if d31.Loc == LocImm {
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d31.StackOff))}
				} else if d31.Loc == LocStackTriple {
					d36 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d31.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d31)
					if d31.Loc == LocRegPair || d31.Loc == LocRegTriple {
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg2, ID: 0}
					} else if d31.Loc == LocReg {
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d31.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d36
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d36)
				ctx.EnsureDesc(&d36)
				var d37 JITValueDesc
				if d36.Loc == LocImm {
					d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d36.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d36.Reg)
					ctx.EmitMovRegReg(scratch, d36.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d37)
				}
				if d37.Loc == LocReg && d36.Loc == LocReg && d37.Reg == d36.Reg {
					ctx.TransferReg(d36.Reg)
					d36.Loc = LocNone
				}
				ctx.FreeDesc(&d36)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d37)
				ctx.ReclaimUntrackedRegs()
				d39 := ctx.EmitSliceElementAddress(&d31, &d37, 16)
				ctx.EnsureDesc(&d39)
				r3 := ctx.AllocRegExcept(d39.Reg)
				ctx.EmitMovRegMem(r3, d39.Reg, 8)
				ctx.EmitMovRegMem(d39.Reg, d39.Reg, 0)
				d38 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d39.Reg, Reg2: r3}
				ctx.BindReg(d39.Reg, &d38)
				ctx.BindReg(r3, &d38)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d38)
				d40 := d38
				_ = d40
				ctx.StabilizeDescForControlFlow(&d40)
				inlineHome41 := d40
				inlineHome41.ID = 0
				phiBase42 := ctx.AllocStack(int32(16))
				d43 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase42) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase42) + int32(0))
				_ = d43
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
				d40 = inlineHome41
				d43 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase42) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d40)
				if d40.Loc == LocReg || d40.Loc == LocFPReg {
					ctx.ProtectReg(d40.Reg)
				} else if d40.Loc == LocRegPair {
					ctx.ProtectReg(d40.Reg)
					ctx.ProtectReg(d40.Reg2)
				}
				d44 := d40
				if d44.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d44)
				if d44.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d44, int32(phiBase42)+int32(0), 2)
				} else if d44.Loc == LocInputPair {
					ctx.EnsureDesc(&d44)
					ctx.EmitStoreScmerToStack(d44, int32(phiBase42)+int32(0))
				} else if d44.Loc == LocRegPair || d44.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d44, int32(phiBase42)+int32(0))
				} else {
					ctx.EnsureDesc(&d44)
					ctx.EmitStoreToStack(d44, int32(phiBase42)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase42)+int32(0))+8)
				}
				if d40.Loc == LocReg || d40.Loc == LocFPReg {
					ctx.UnprotectReg(d40.Reg)
				} else if d40.Loc == LocRegPair {
					ctx.UnprotectReg(d40.Reg)
					ctx.UnprotectReg(d40.Reg2)
				}
				bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl28)
				ctx.ResolveFixups()
				d40 = inlineHome41
				d43 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase42) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d45 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase42) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d45)
				ctx.ReclaimUntrackedRegs()
				d45 = JITPrepareScmerGoArg(ctx, d45)
				ctx.SyncDesc(&d45)
				d46 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d45}, 1)
				d46.NoHeapPointer = true
				ctx.EmitAndRegImm32(d46.Reg, 1)
				d46.Type = tagBool
				ctx.BindReg(d46.Reg, &d46)
				ctx.ReclaimUntrackedRegs()
				d47 := d46
				ctx.EnsureDesc(&d47)
				if d47.Loc != LocImm && d47.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl29 := ctx.ReserveLabel()
				lbl30 := ctx.ReserveLabel()
				if d47.Loc == LocImm {
					if d47.Imm.Bool() {
						ctx.MarkLabel(lbl29)
						ctx.EmitJmp(lbl26)
					} else {
						ctx.MarkLabel(lbl30)
						ctx.EmitJmp(lbl27)
					}
				} else {
					ctx.EmitCmpRegImm32(d47.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl29)
					ctx.EmitJmp(lbl30)
					ctx.MarkLabel(lbl29)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl30)
					ctx.EmitJmp(lbl27)
				}
				ctx.FreeDesc(&d46)
				bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl27)
				ctx.ResolveFixups()
				d40 = inlineHome41
				d45 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase42) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				d48 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d48)
				ctx.BindReg(r5, &d48)
				ctx.EmitMovPairToResult(&d45, &d48)
				ctx.EmitJmp(lbl24)
				bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl26)
				ctx.ResolveFixups()
				d40 = inlineHome41
				d45 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase42) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d45 = JITPrepareScmerGoArg(ctx, d45)
				ctx.SyncDesc(&d45)
				d49 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d45}, 1)
				d49.NoHeapPointer = false
				ctx.BindReg(d49.Reg, &d49)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d50 JITValueDesc
				ctx.EnsureDesc(&d49)
				if d49.Loc == LocImm {
					fieldAddr := uintptr(d49.Imm.Int()) + 32
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					ctx.EmitMovRegMem64(r6, fieldAddr)
					ctx.EmitMovRegMem64(r7, fieldAddr+8)
					d50 = JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d50)
					ctx.BindReg(r7, &d50)
				} else {
					off := int32(32)
					baseReg := d49.Reg
					r8 := ctx.AllocRegExcept(baseReg)
					r9 := ctx.AllocRegExcept(baseReg, r8)
					ctx.EmitMovRegMem(r8, baseReg, off)
					ctx.EmitMovRegMem(r9, baseReg, off+8)
					d50 = JITValueDesc{Loc: LocRegPair, Reg: r8, Reg2: r9}
					ctx.BindReg(r8, &d50)
					ctx.BindReg(r9, &d50)
				}
				ctx.FreeDesc(&d49)
				ctx.StabilizeDescForControlFlow(&d50)
				inlineHome51 := d50
				inlineHome51.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d50)
				if d50.Loc == LocReg || d50.Loc == LocFPReg {
					ctx.ProtectReg(d50.Reg)
				} else if d50.Loc == LocRegPair {
					ctx.ProtectReg(d50.Reg)
					ctx.ProtectReg(d50.Reg2)
				}
				d52 := d50
				if d52.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d52)
				if d52.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d52, int32(phiBase42)+int32(0), 2)
				} else if d52.Loc == LocInputPair {
					ctx.EnsureDesc(&d52)
					ctx.EmitStoreScmerToStack(d52, int32(phiBase42)+int32(0))
				} else if d52.Loc == LocRegPair || d52.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d52, int32(phiBase42)+int32(0))
				} else {
					ctx.EnsureDesc(&d52)
					ctx.EmitStoreToStack(d52, int32(phiBase42)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase42)+int32(0))+8)
				}
				if d50.Loc == LocReg || d50.Loc == LocFPReg {
					ctx.UnprotectReg(d50.Reg)
				} else if d50.Loc == LocRegPair {
					ctx.UnprotectReg(d50.Reg)
					ctx.UnprotectReg(d50.Reg2)
				}
				ctx.EmitJmp(lbl28)
				ctx.MarkLabel(lbl24)
				d53 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d53)
				ctx.BindReg(r5, &d53)
				ctx.BindReg(r4, &d53)
				ctx.BindReg(r5, &d53)
				ctx.StabilizeDescForControlFlow(&d53)
				inlineHome54 := d53
				inlineHome54.ID = 0
				ctx.FreeDesc(&d38)
				ctx.ReclaimUntrackedRegs()
				d55 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				ctx.EnsureDesc(&d37)
				ctx.EnsureDesc(&d31)
				ctx.EnsureDesc(&d55)
				ctx.EnsureDesc(&d37)
				var d57 JITValueDesc
				if d37.Loc == LocImm && d55.Loc == LocImm {
					d57 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() - d55.Imm.Int())}
				} else {
					r10 := ctx.AllocReg()
					if d37.Loc == LocImm {
						ctx.EmitMovRegImm64(r10, uint64(d37.Imm.Int()))
					} else {
						ctx.EmitMovRegReg(r10, d37.Reg)
					}
					if d55.Loc == LocImm {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d55.Imm.Int()))
						ctx.EmitSubInt64(r10, ctx.ScratchReg)
					} else {
						ctx.EmitSubInt64(r10, d55.Reg)
					}
					d57 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
					ctx.BindReg(r10, &d57)
				}
				var d58 JITValueDesc
				r11 := ctx.EmitSliceDataAfterLow(&d31, &d55, 16)
				d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
				ctx.BindReg(r11, &d58)
				ctx.BindReg(r11, &d58)
				var d59 JITValueDesc
				var r12 Reg
				var r13 Reg
				ctx.SyncDesc(&d58)
				ctx.EnsureDesc(&d58)
				if d58.Loc == LocImm {
					r12 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r12, uint64(d58.Imm.Int()))
				} else {
					r12 = d58.Reg
				}
				ctx.ProtectReg(r12)
				ctx.SyncDesc(&d57)
				ctx.EnsureDesc(&d57)
				if d57.Loc == LocImm {
					r13 = ctx.AllocReg()
					ctx.EmitMovRegImm64(r13, uint64(d57.Imm.Int()))
				} else {
					r13 = d57.Reg
				}
				ctx.ProtectReg(r13)
				r14 := ctx.EmitSliceCapAfterLow(&d31, &d55, r12, r13)
				ctx.UnprotectReg(r13)
				ctx.UnprotectReg(r12)
				d59 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
				ctx.BindReg(r12, &d59)
				ctx.BindReg(r13, &d59)
				ctx.BindReg(r14, &d59)
				ctx.BindReg(r12, &d59)
				ctx.BindReg(r13, &d59)
				ctx.BindReg(r14, &d59)
				ctx.StabilizeDescForControlFlow(&d59)
				inlineHome60 := d59
				inlineHome60.ID = 0
				ctx.FreeDesc(&d37)
				ctx.ReclaimUntrackedRegs()
				d62 := d53
				d62.ID = 0
				d61 := ctx.EmitTagEqualsBorrowed(&d62, tagSlice, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				d63 := d61
				ctx.EnsureDesc(&d63)
				if d63.Loc != LocImm && d63.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl31 := ctx.ReserveLabel()
				lbl32 := ctx.ReserveLabel()
				if d63.Loc == LocImm {
					if d63.Imm.Bool() {
						ctx.MarkLabel(lbl31)
						ctx.EmitJmp(lbl8)
					} else {
						ctx.MarkLabel(lbl32)
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
							ctx.EmitCopyStackWords(d64, int32(phiBase19)+int32(0), 3)
						} else {
							if d64.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d64.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
							ctx.EmitStoreRegMem(d64.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
							ctx.EmitStoreRegMem(d64.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
						}
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.UnprotectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.UnprotectReg(d59.Reg)
							ctx.UnprotectReg(d59.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitCmpRegImm32(d63.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl31)
					ctx.EmitJmp(lbl32)
					ctx.MarkLabel(lbl31)
					ctx.EmitJmp(lbl8)
					ctx.MarkLabel(lbl32)
					ctx.SyncDesc(&d59)
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.ProtectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.ProtectReg(d59.Reg)
						ctx.ProtectReg(d59.Reg2)
					}
					d65 := d59
					if d65.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d65)
					if d65.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d65, int32(phiBase19)+int32(0), 3)
					} else {
						if d65.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d65.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
						ctx.EmitStoreRegMem(d65.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
						ctx.EmitStoreRegMem(d65.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
					}
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.UnprotectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.UnprotectReg(d59.Reg)
						ctx.UnprotectReg(d59.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				ctx.FreeDesc(&d61)
				bbpos_2_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d66 := jitKnownSliceHeader(ctx, &d53)
				ctx.StabilizeDescForControlFlow(&d66)
				inlineHome67 := d66
				inlineHome67.ID = 0
				ctx.ReclaimUntrackedRegs()
				var d68 JITValueDesc
				if d66.SliceSizeKnown {
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d66.KnownSliceLen))}
				} else if d66.Loc == LocImm {
					d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d66.StackOff))}
				} else if d66.Loc == LocStackTriple {
					d68 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d66.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d66)
					if d66.Loc == LocRegPair || d66.Loc == LocRegTriple {
						d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg2, ID: 0}
					} else if d66.Loc == LocReg {
						d68 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d68
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d68)
				var d69 JITValueDesc
				if d68.Loc == LocImm {
					d69 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d68.Imm.Int() != 0)}
				} else {
					r15 := ctx.AllocReg()
					ctx.EmitCmpRegImm32(d68.Reg, 0)
					d69 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondNotEqual}
					ctx.BindReg(r15, &d69)
				}
				ctx.FreeDesc(&d68)
				ctx.ReclaimUntrackedRegs()
				d70 := d69
				ctx.SyncDesc(&d70)
				if d70.Loc != LocImm && d70.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl33 := ctx.ReserveLabel()
				lbl34 := ctx.ReserveLabel()
				if d70.Loc == LocImm {
					if d70.Imm.Bool() {
						ctx.MarkLabel(lbl33)
						ctx.EmitJmp(lbl11)
					} else {
						ctx.MarkLabel(lbl34)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitJump(d70.Condition, lbl33)
					ctx.EmitJmp(lbl34)
					ctx.FreeDesc(&d69)
					ctx.MarkLabel(lbl33)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl34)
					ctx.EmitJmp(lbl10)
				}
				bbpos_2_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d71 JITValueDesc
				if d66.SliceSizeKnown {
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d66.KnownSliceLen))}
				} else if d66.Loc == LocImm {
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d66.StackOff))}
				} else if d66.Loc == LocStackTriple {
					d71 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d66.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d66)
					if d66.Loc == LocRegPair || d66.Loc == LocRegTriple {
						d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg2, ID: 0}
					} else if d66.Loc == LocReg {
						d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d71
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d71)
				ctx.EnsureDesc(&d71)
				var d72 JITValueDesc
				if d71.Loc == LocImm {
					d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d71.Reg)
					ctx.EmitMovRegReg(scratch, d71.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d72)
				}
				if d72.Loc == LocReg && d71.Loc == LocReg && d72.Reg == d71.Reg {
					ctx.TransferReg(d71.Reg)
					d71.Loc = LocNone
				}
				ctx.EnsureDesc(&d72)
				ctx.EmitStoreToStack(d72, int32(phiBase19)+int32(80))
				ctx.StabilizeDescForControlFlow(&d72)
				inlineHome73 := d72
				inlineHome73.ID = 0
				ctx.FreeDesc(&d71)
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d59)
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.ProtectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.ProtectReg(d59.Reg)
					ctx.ProtectReg(d59.Reg2)
				}
				d74 := d59
				if d74.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d74)
				if d74.Loc == LocStackTriple {
					ctx.EmitCopyStackWords(d74, int32(phiBase19)+int32(56), 3)
				} else {
					if d74.Loc != LocRegTriple {
						panic("jit: slice phi source is not a triple")
					}
					ctx.EmitStoreRegMem(d74.Reg, ctx.StackReg, int32(phiBase19)+int32(56))
					ctx.EmitStoreRegMem(d74.Reg2, ctx.StackReg, int32(phiBase19)+int32(56)+8)
					ctx.EmitStoreRegMem(d74.Reg3, ctx.StackReg, int32(phiBase19)+int32(56)+16)
				}
				if d59.Loc == LocReg || d59.Loc == LocFPReg {
					ctx.UnprotectReg(d59.Reg)
				} else if d59.Loc == LocRegPair {
					ctx.UnprotectReg(d59.Reg)
					ctx.UnprotectReg(d59.Reg2)
				}
				bbpos_2_16 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d23 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d24 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d75 := JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				ctx.StabilizeDescForControlFlow(&d75)
				ctx.ReclaimUntrackedRegs()
				d76 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(80)}
				ctx.StabilizeDescForControlFlow(&d76)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d76)
				var d77 JITValueDesc
				if d76.Loc == LocImm {
					d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d76.Imm.Int() >= 1)}
				} else {
					r16 := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitCmpRegImm32(d76.Reg, 1)
					d77 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedGreaterOrEqual}
					ctx.BindReg(r16, &d77)
				}
				ctx.ReclaimUntrackedRegs()
				d78 := d77
				ctx.SyncDesc(&d78)
				if d78.Loc != LocImm && d78.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl35 := ctx.ReserveLabel()
				lbl36 := ctx.ReserveLabel()
				if d78.Loc == LocImm {
					if d78.Imm.Bool() {
						ctx.MarkLabel(lbl35)
						ctx.EmitJmp(lbl19)
					} else {
						ctx.MarkLabel(lbl36)
						ctx.SyncDesc(&d75)
						if d75.Loc == LocReg || d75.Loc == LocFPReg {
							ctx.ProtectReg(d75.Reg)
						} else if d75.Loc == LocRegPair {
							ctx.ProtectReg(d75.Reg)
							ctx.ProtectReg(d75.Reg2)
						}
						d79 := d75
						if d79.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d79)
						if d79.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d79, int32(phiBase19)+int32(0), 3)
						} else {
							if d79.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d79.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
							ctx.EmitStoreRegMem(d79.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
							ctx.EmitStoreRegMem(d79.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
						}
						if d75.Loc == LocReg || d75.Loc == LocFPReg {
							ctx.UnprotectReg(d75.Reg)
						} else if d75.Loc == LocRegPair {
							ctx.UnprotectReg(d75.Reg)
							ctx.UnprotectReg(d75.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d78.Condition, lbl35)
					ctx.EmitJmp(lbl36)
					ctx.FreeDesc(&d77)
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl36)
					ctx.SyncDesc(&d75)
					if d75.Loc == LocReg || d75.Loc == LocFPReg {
						ctx.ProtectReg(d75.Reg)
					} else if d75.Loc == LocRegPair {
						ctx.ProtectReg(d75.Reg)
						ctx.ProtectReg(d75.Reg2)
					}
					d80 := d75
					if d80.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d80)
					if d80.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d80, int32(phiBase19)+int32(0), 3)
					} else {
						if d80.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d80.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
						ctx.EmitStoreRegMem(d80.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
						ctx.EmitStoreRegMem(d80.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
					}
					if d75.Loc == LocReg || d75.Loc == LocFPReg {
						ctx.UnprotectReg(d75.Reg)
					} else if d75.Loc == LocRegPair {
						ctx.UnprotectReg(d75.Reg)
						ctx.UnprotectReg(d75.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				bbpos_2_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d81 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				_ = d81
				ctx.ReclaimUntrackedRegs()
				d83 := ctx.EmitSliceElementAddress(&d66, &d81, 16)
				ctx.EnsureDesc(&d83)
				r17 := ctx.AllocRegExcept(d83.Reg)
				ctx.EmitMovRegMem(r17, d83.Reg, 8)
				ctx.EmitMovRegMem(d83.Reg, d83.Reg, 0)
				d82 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d83.Reg, Reg2: r17}
				ctx.BindReg(d83.Reg, &d82)
				ctx.BindReg(r17, &d82)
				ctx.ReclaimUntrackedRegs()
				d82 = JITPrepareScmerGoArg(ctx, d82)
				d11 = JITPrepareScmerGoArg(ctx, d11)
				ctx.SyncDesc(&d82)
				ctx.SyncDesc(&d11)
				d84 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d82, d11}, 1)
				d84.NoHeapPointer = true
				ctx.EmitAndRegImm32(d84.Reg, 1)
				d84.Type = tagBool
				ctx.BindReg(d84.Reg, &d84)
				ctx.FreeDesc(&d82)
				ctx.ReclaimUntrackedRegs()
				d85 := d84
				ctx.EnsureDesc(&d85)
				if d85.Loc != LocImm && d85.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl37 := ctx.ReserveLabel()
				lbl38 := ctx.ReserveLabel()
				if d85.Loc == LocImm {
					if d85.Imm.Bool() {
						ctx.MarkLabel(lbl37)
						ctx.EmitJmp(lbl9)
					} else {
						ctx.MarkLabel(lbl38)
						ctx.EmitJmp(lbl10)
					}
				} else {
					ctx.EmitCmpRegImm32(d85.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl37)
					ctx.EmitJmp(lbl38)
					ctx.MarkLabel(lbl37)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl38)
					ctx.EmitJmp(lbl10)
				}
				ctx.FreeDesc(&d84)
				bbpos_2_17 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d76)
				ctx.ReclaimUntrackedRegs()
				d87 := ctx.EmitSliceElementAddress(&d66, &d76, 16)
				ctx.EnsureDesc(&d87)
				r18 := ctx.AllocRegExcept(d87.Reg)
				ctx.EmitMovRegMem(r18, d87.Reg, 8)
				ctx.EmitMovRegMem(d87.Reg, d87.Reg, 0)
				d86 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d87.Reg, Reg2: r18}
				ctx.BindReg(d87.Reg, &d86)
				ctx.BindReg(r18, &d86)
				ctx.ReclaimUntrackedRegs()
				stackArray88 := ctx.AllocStack(int32(16))
				_ = stackArray88
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d86)
				ctx.EmitStoreScmerToStack(d86, int32(stackArray88)+int32(0))
				ctx.FreeDesc(&d86)
				ctx.ReclaimUntrackedRegs()
				d89 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d89
				ctx.ReclaimUntrackedRegs()
				r19 := ctx.AllocReg()
				r20 := ctx.AllocRegExcept(r19)
				r21 := ctx.AllocRegExcept(r19, r20)
				d90 := JITValueDesc{Loc: LocRegTriple, Type: JITTypeUnknown, Reg: r19, Reg2: r20, Reg3: r21}
				ctx.BindReg(r19, &d90)
				ctx.BindReg(r20, &d90)
				ctx.BindReg(r21, &d90)
				ctx.BindReg(r19, &d90)
				ctx.BindReg(r20, &d90)
				ctx.BindReg(r21, &d90)
				ctx.EmitLeaRegMem(d90.Reg, ctx.StackReg, int32(stackArray88))
				ctx.EmitMovRegImm64(d90.Reg2, uint64(1))
				ctx.EmitMovRegImm64(d90.Reg3, uint64(1))
				callResults91 := JITEmitGoCallResults(ctx, GoFuncAddr(JITAppendScmerSlice), []JITValueDesc{d75, d90}, []uint8{3}, []uint8{1})
				d92 := callResults91[0]
				d93 := JITValueDesc{Loc: LocStackTriple, Type: tagSlice, StackOff: int32(phiBase19) + int32(56)}
				ctx.EmitCopyDescWords(&d93, &d92, 3)
				ctx.FreeDesc(&d92)
				d92 = d93
				ctx.StabilizeDescForControlFlow(&d92)
				inlineHome94 := d92
				inlineHome94.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d76)
				ctx.EnsureDesc(&d76)
				var d95 JITValueDesc
				if d76.Loc == LocImm {
					d95 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d76.Imm.Int() - 1)}
				} else {
					scratch := ctx.AllocRegExcept(d76.Reg)
					ctx.EmitMovRegReg(scratch, d76.Reg)
					ctx.EmitIntBinaryImm(JITIntSub, 64, scratch, 1)
					d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d95)
				}
				if d95.Loc == LocReg && d76.Loc == LocReg && d95.Reg == d76.Reg {
					ctx.TransferReg(d76.Reg)
					d76.Loc = LocNone
				}
				ctx.EnsureDesc(&d95)
				ctx.EmitStoreToStack(d95, int32(phiBase19)+int32(80))
				ctx.StabilizeDescForControlFlow(&d95)
				inlineHome96 := d95
				inlineHome96.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EmitJmp(lbl18)
				bbpos_2_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d92 = inlineHome94
				d95 = inlineHome96
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d97 JITValueDesc
				if d66.SliceSizeKnown {
					d97 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d66.KnownSliceLen))}
				} else if d66.Loc == LocImm {
					d97 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d66.StackOff))}
				} else if d66.Loc == LocStackTriple {
					d97 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d66.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d66)
					if d66.Loc == LocRegPair || d66.Loc == LocRegTriple {
						d97 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg2, ID: 0}
					} else if d66.Loc == LocReg {
						d97 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d66.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d97
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d97)
				ctx.EnsureDesc(&d13)
				ctx.EnsureDescsTogether(&d97, &d13)
				var d98 JITValueDesc
				if d97.Loc == LocImm && d13.Loc == LocImm {
					d98 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d97.Imm.Int() > d13.Imm.Int())}
				} else if d13.Loc == LocImm {
					r22 := ctx.AllocReg()
					if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d97.Reg, int32(d13.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d13.Imm.Int()))
						ctx.EmitCmpInt64(d97.Reg, ctx.ScratchReg)
					}
					d98 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r22, Condition: CondSignedGreater}
					ctx.BindReg(r22, &d98)
				} else if d97.Loc == LocImm {
					r23 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d97.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d13.Reg)
					d98 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r23, Condition: CondSignedGreater}
					ctx.BindReg(r23, &d98)
				} else {
					r24 := ctx.AllocReg()
					ctx.EmitCmpInt64(d97.Reg, d13.Reg)
					d98 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r24, Condition: CondSignedGreater}
					ctx.BindReg(r24, &d98)
				}
				ctx.FreeDesc(&d97)
				ctx.ReclaimUntrackedRegs()
				d99 := d98
				ctx.SyncDesc(&d99)
				if d99.Loc != LocImm && d99.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl39 := ctx.ReserveLabel()
				lbl40 := ctx.ReserveLabel()
				if d99.Loc == LocImm {
					if d99.Imm.Bool() {
						ctx.MarkLabel(lbl39)
						ctx.EmitJmp(lbl12)
					} else {
						ctx.MarkLabel(lbl40)
						ctx.SyncDesc(&d59)
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.ProtectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.ProtectReg(d59.Reg)
							ctx.ProtectReg(d59.Reg2)
						}
						d100 := d59
						if d100.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d100)
						if d100.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d100, int32(phiBase19)+int32(0), 3)
						} else {
							if d100.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d100.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
							ctx.EmitStoreRegMem(d100.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
							ctx.EmitStoreRegMem(d100.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
						}
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.UnprotectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.UnprotectReg(d59.Reg)
							ctx.UnprotectReg(d59.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d99.Condition, lbl39)
					ctx.EmitJmp(lbl40)
					ctx.FreeDesc(&d98)
					ctx.MarkLabel(lbl39)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl40)
					ctx.SyncDesc(&d59)
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.ProtectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.ProtectReg(d59.Reg)
						ctx.ProtectReg(d59.Reg2)
					}
					d101 := d59
					if d101.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d101)
					if d101.Loc == LocStackTriple {
						ctx.EmitCopyStackWords(d101, int32(phiBase19)+int32(0), 3)
					} else {
						if d101.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d101.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
						ctx.EmitStoreRegMem(d101.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
						ctx.EmitStoreRegMem(d101.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
					}
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.UnprotectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.UnprotectReg(d59.Reg)
						ctx.UnprotectReg(d59.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				bbpos_2_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d92 = inlineHome94
				d95 = inlineHome96
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				ctx.ReclaimUntrackedRegs()
				d103 := ctx.EmitSliceElementAddress(&d66, &d13, 16)
				ctx.EnsureDesc(&d103)
				r25 := ctx.AllocRegExcept(d103.Reg)
				ctx.EmitMovRegMem(r25, d103.Reg, 8)
				ctx.EmitMovRegMem(d103.Reg, d103.Reg, 0)
				d102 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d103.Reg, Reg2: r25}
				ctx.BindReg(d103.Reg, &d102)
				ctx.BindReg(r25, &d102)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d102)
				d104 := d102
				_ = d104
				ctx.StabilizeDescForControlFlow(&d104)
				inlineHome105 := d104
				inlineHome105.ID = 0
				phiBase106 := ctx.AllocStack(int32(16))
				d107 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase106) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase106) + int32(0))
				_ = d107
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
				d104 = inlineHome105
				d107 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase106) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d104)
				if d104.Loc == LocReg || d104.Loc == LocFPReg {
					ctx.ProtectReg(d104.Reg)
				} else if d104.Loc == LocRegPair {
					ctx.ProtectReg(d104.Reg)
					ctx.ProtectReg(d104.Reg2)
				}
				d108 := d104
				if d108.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d108)
				if d108.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d108, int32(phiBase106)+int32(0), 2)
				} else if d108.Loc == LocInputPair {
					ctx.EnsureDesc(&d108)
					ctx.EmitStoreScmerToStack(d108, int32(phiBase106)+int32(0))
				} else if d108.Loc == LocRegPair || d108.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d108, int32(phiBase106)+int32(0))
				} else {
					ctx.EnsureDesc(&d108)
					ctx.EmitStoreToStack(d108, int32(phiBase106)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase106)+int32(0))+8)
				}
				if d104.Loc == LocReg || d104.Loc == LocFPReg {
					ctx.UnprotectReg(d104.Reg)
				} else if d104.Loc == LocRegPair {
					ctx.UnprotectReg(d104.Reg)
					ctx.UnprotectReg(d104.Reg2)
				}
				bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl45)
				ctx.ResolveFixups()
				d104 = inlineHome105
				d107 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase106) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d109 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase106) + int32(0)}
				ctx.StabilizeDescForControlFlow(&d109)
				ctx.ReclaimUntrackedRegs()
				d109 = JITPrepareScmerGoArg(ctx, d109)
				ctx.SyncDesc(&d109)
				d110 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).IsSourceInfo), []JITValueDesc{d109}, 1)
				d110.NoHeapPointer = true
				ctx.EmitAndRegImm32(d110.Reg, 1)
				d110.Type = tagBool
				ctx.BindReg(d110.Reg, &d110)
				ctx.ReclaimUntrackedRegs()
				d111 := d110
				ctx.EnsureDesc(&d111)
				if d111.Loc != LocImm && d111.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl46 := ctx.ReserveLabel()
				lbl47 := ctx.ReserveLabel()
				if d111.Loc == LocImm {
					if d111.Imm.Bool() {
						ctx.MarkLabel(lbl46)
						ctx.EmitJmp(lbl43)
					} else {
						ctx.MarkLabel(lbl47)
						ctx.EmitJmp(lbl44)
					}
				} else {
					ctx.EmitCmpRegImm32(d111.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl46)
					ctx.EmitJmp(lbl47)
					ctx.MarkLabel(lbl46)
					ctx.EmitJmp(lbl43)
					ctx.MarkLabel(lbl47)
					ctx.EmitJmp(lbl44)
				}
				ctx.FreeDesc(&d110)
				bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl44)
				ctx.ResolveFixups()
				d104 = inlineHome105
				d109 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase106) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				r26 := ctx.AllocReg()
				r27 := ctx.AllocRegExcept(r26)
				d112 := JITValueDesc{Loc: LocRegPair, Reg: r26, Reg2: r27}
				ctx.BindReg(r26, &d112)
				ctx.BindReg(r27, &d112)
				ctx.EmitMovPairToResult(&d109, &d112)
				ctx.EmitJmp(lbl41)
				bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl43)
				ctx.ResolveFixups()
				d104 = inlineHome105
				d109 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase106) + int32(0)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d109 = JITPrepareScmerGoArg(ctx, d109)
				ctx.SyncDesc(&d109)
				d113 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).SourceInfo), []JITValueDesc{d109}, 1)
				d113.NoHeapPointer = false
				ctx.BindReg(d113.Reg, &d113)
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				var d114 JITValueDesc
				ctx.EnsureDesc(&d113)
				if d113.Loc == LocImm {
					fieldAddr := uintptr(d113.Imm.Int()) + 32
					r28 := ctx.AllocReg()
					r29 := ctx.AllocRegExcept(r28)
					ctx.EmitMovRegMem64(r28, fieldAddr)
					ctx.EmitMovRegMem64(r29, fieldAddr+8)
					d114 = JITValueDesc{Loc: LocRegPair, Reg: r28, Reg2: r29}
					ctx.BindReg(r28, &d114)
					ctx.BindReg(r29, &d114)
				} else {
					off := int32(32)
					baseReg := d113.Reg
					r30 := ctx.AllocRegExcept(baseReg)
					r31 := ctx.AllocRegExcept(baseReg, r30)
					ctx.EmitMovRegMem(r30, baseReg, off)
					ctx.EmitMovRegMem(r31, baseReg, off+8)
					d114 = JITValueDesc{Loc: LocRegPair, Reg: r30, Reg2: r31}
					ctx.BindReg(r30, &d114)
					ctx.BindReg(r31, &d114)
				}
				ctx.FreeDesc(&d113)
				ctx.StabilizeDescForControlFlow(&d114)
				inlineHome115 := d114
				inlineHome115.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d114)
				if d114.Loc == LocReg || d114.Loc == LocFPReg {
					ctx.ProtectReg(d114.Reg)
				} else if d114.Loc == LocRegPair {
					ctx.ProtectReg(d114.Reg)
					ctx.ProtectReg(d114.Reg2)
				}
				d116 := d114
				if d116.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d116)
				if d116.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d116, int32(phiBase106)+int32(0), 2)
				} else if d116.Loc == LocInputPair {
					ctx.EnsureDesc(&d116)
					ctx.EmitStoreScmerToStack(d116, int32(phiBase106)+int32(0))
				} else if d116.Loc == LocRegPair || d116.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d116, int32(phiBase106)+int32(0))
				} else {
					ctx.EnsureDesc(&d116)
					ctx.EmitStoreToStack(d116, int32(phiBase106)+int32(0))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase106)+int32(0))+8)
				}
				if d114.Loc == LocReg || d114.Loc == LocFPReg {
					ctx.UnprotectReg(d114.Reg)
				} else if d114.Loc == LocRegPair {
					ctx.UnprotectReg(d114.Reg)
					ctx.UnprotectReg(d114.Reg2)
				}
				ctx.EmitJmp(lbl45)
				ctx.MarkLabel(lbl41)
				d117 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r26, Reg2: r27}
				ctx.BindReg(r26, &d117)
				ctx.BindReg(r27, &d117)
				ctx.BindReg(r26, &d117)
				ctx.BindReg(r27, &d117)
				ctx.StabilizeDescForControlFlow(&d117)
				inlineHome118 := d117
				inlineHome118.ID = 0
				ctx.FreeDesc(&d102)
				ctx.ReclaimUntrackedRegs()
				d120 := d117
				d120.ID = 0
				d119 := ctx.EmitTagEqualsBorrowed(&d120, tagNil, JITValueDesc{Loc: LocAny})
				ctx.ReclaimUntrackedRegs()
				d121 := d119
				ctx.EnsureDesc(&d121)
				if d121.Loc != LocImm && d121.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl48 := ctx.ReserveLabel()
				lbl49 := ctx.ReserveLabel()
				if d121.Loc == LocImm {
					if d121.Imm.Bool() {
						ctx.MarkLabel(lbl48)
						ctx.EmitJmp(lbl13)
					} else {
						ctx.MarkLabel(lbl49)
						ctx.SyncDesc(&d117)
						if d117.Loc == LocReg || d117.Loc == LocFPReg {
							ctx.ProtectReg(d117.Reg)
						} else if d117.Loc == LocRegPair {
							ctx.ProtectReg(d117.Reg)
							ctx.ProtectReg(d117.Reg2)
						}
						d122 := d117
						if d122.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d122)
						if d122.Loc == LocStackPair {
							ctx.EmitCopyStackWords(d122, int32(phiBase19)+int32(24), 2)
						} else if d122.Loc == LocInputPair {
							ctx.EnsureDesc(&d122)
							ctx.EmitStoreScmerToStack(d122, int32(phiBase19)+int32(24))
						} else if d122.Loc == LocRegPair || d122.Loc == LocImm {
							ctx.EmitStoreScmerToStack(d122, int32(phiBase19)+int32(24))
						} else {
							ctx.EnsureDesc(&d122)
							ctx.EmitStoreToStack(d122, int32(phiBase19)+int32(24))
							ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase19)+int32(24))+8)
						}
						if d117.Loc == LocReg || d117.Loc == LocFPReg {
							ctx.UnprotectReg(d117.Reg)
						} else if d117.Loc == LocRegPair {
							ctx.UnprotectReg(d117.Reg)
							ctx.UnprotectReg(d117.Reg2)
						}
						ctx.EmitJmp(lbl14)
					}
				} else {
					ctx.EmitCmpRegImm32(d121.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl48)
					ctx.EmitJmp(lbl49)
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl49)
					ctx.SyncDesc(&d117)
					if d117.Loc == LocReg || d117.Loc == LocFPReg {
						ctx.ProtectReg(d117.Reg)
					} else if d117.Loc == LocRegPair {
						ctx.ProtectReg(d117.Reg)
						ctx.ProtectReg(d117.Reg2)
					}
					d123 := d117
					if d123.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d123)
					if d123.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d123, int32(phiBase19)+int32(24), 2)
					} else if d123.Loc == LocInputPair {
						ctx.EnsureDesc(&d123)
						ctx.EmitStoreScmerToStack(d123, int32(phiBase19)+int32(24))
					} else if d123.Loc == LocRegPair || d123.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d123, int32(phiBase19)+int32(24))
					} else {
						ctx.EnsureDesc(&d123)
						ctx.EmitStoreToStack(d123, int32(phiBase19)+int32(24))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase19)+int32(24))+8)
					}
					if d117.Loc == LocReg || d117.Loc == LocFPReg {
						ctx.UnprotectReg(d117.Reg)
					} else if d117.Loc == LocRegPair {
						ctx.UnprotectReg(d117.Reg)
						ctx.UnprotectReg(d117.Reg2)
					}
					ctx.EmitJmp(lbl14)
				}
				ctx.FreeDesc(&d119)
				bbpos_2_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d92 = inlineHome94
				d95 = inlineHome96
				d117 = inlineHome118
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d21 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d124 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				ctx.StabilizeDescForControlFlow(&d124)
				ctx.ReclaimUntrackedRegs()
				var d125 JITValueDesc
				if d17.SliceSizeKnown {
					d125 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.KnownSliceLen))}
				} else if d17.Loc == LocImm {
					d125 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d17.StackOff))}
				} else if d17.Loc == LocStackTriple {
					d125 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d17.StackOff + 8, NoHeapPointer: true}
				} else {
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocRegPair || d17.Loc == LocRegTriple {
						d125 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg2, ID: 0}
					} else if d17.Loc == LocReg {
						d125 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d17.Reg, ID: 0}
					} else {
						panic("len on unsupported descriptor location")
					}
				}
				_ = d125
				ctx.StabilizeDescForControlFlow(&d125)
				inlineHome126 := d125
				inlineHome126.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(phiBase19)+int32(40))
				bbpos_2_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d92 = inlineHome94
				d95 = inlineHome96
				d117 = inlineHome118
				d125 = inlineHome126
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d124 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d22 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d127 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(40)}
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d127)
				ctx.EnsureDesc(&d127)
				var d128 JITValueDesc
				if d127.Loc == LocImm {
					d128 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d127.Imm.Int() + 1)}
				} else {
					scratch := ctx.AllocRegExcept(d127.Reg)
					ctx.EmitMovRegReg(scratch, d127.Reg)
					ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
					d128 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d128)
				}
				if d128.Loc == LocReg && d127.Loc == LocReg && d128.Reg == d127.Reg {
					ctx.TransferReg(d127.Reg)
					d127.Loc = LocNone
				}
				ctx.StabilizeDescForControlFlow(&d128)
				inlineHome129 := d128
				inlineHome129.ID = 0
				ctx.FreeDesc(&d127)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d128)
				ctx.EnsureDesc(&d125)
				ctx.EnsureDescsTogether(&d128, &d125)
				var d130 JITValueDesc
				if d128.Loc == LocImm && d125.Loc == LocImm {
					d130 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d128.Imm.Int() < d125.Imm.Int())}
				} else if d125.Loc == LocImm {
					r32 := ctx.AllocRegExcept(d128.Reg)
					if d125.Imm.Int() >= -2147483648 && d125.Imm.Int() <= 2147483647 {
						ctx.EmitCmpRegImm32(d128.Reg, int32(d125.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d125.Imm.Int()))
						ctx.EmitCmpInt64(d128.Reg, ctx.ScratchReg)
					}
					d130 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r32, Condition: CondSignedLess}
					ctx.BindReg(r32, &d130)
				} else if d128.Loc == LocImm {
					r33 := ctx.AllocReg()
					ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d128.Imm.Int()))
					ctx.EmitCmpInt64(ctx.ScratchReg, d125.Reg)
					d130 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r33, Condition: CondSignedLess}
					ctx.BindReg(r33, &d130)
				} else {
					r34 := ctx.AllocRegExcept(d128.Reg)
					ctx.EmitCmpInt64(d128.Reg, d125.Reg)
					d130 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r34, Condition: CondSignedLess}
					ctx.BindReg(r34, &d130)
				}
				ctx.ReclaimUntrackedRegs()
				d131 := d130
				ctx.SyncDesc(&d131)
				if d131.Loc != LocImm && d131.Loc != LocFlags {
					panic("jit: fused If condition is neither LocImm nor LocFlags")
				}
				lbl50 := ctx.ReserveLabel()
				lbl51 := ctx.ReserveLabel()
				if d131.Loc == LocImm {
					if d131.Imm.Bool() {
						ctx.MarkLabel(lbl50)
						ctx.EmitJmp(lbl16)
					} else {
						ctx.MarkLabel(lbl51)
						ctx.SyncDesc(&d59)
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.ProtectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.ProtectReg(d59.Reg)
							ctx.ProtectReg(d59.Reg2)
						}
						d132 := d59
						if d132.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.SyncDesc(&d132)
						if d132.Loc == LocStackTriple {
							ctx.EmitCopyStackWords(d132, int32(phiBase19)+int32(0), 3)
						} else {
							if d132.Loc != LocRegTriple {
								panic("jit: slice phi source is not a triple")
							}
							ctx.EmitStoreRegMem(d132.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
							ctx.EmitStoreRegMem(d132.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
							ctx.EmitStoreRegMem(d132.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
						}
						if d59.Loc == LocReg || d59.Loc == LocFPReg {
							ctx.UnprotectReg(d59.Reg)
						} else if d59.Loc == LocRegPair {
							ctx.UnprotectReg(d59.Reg)
							ctx.UnprotectReg(d59.Reg2)
						}
						ctx.EmitJmp(lbl7)
					}
				} else {
					ctx.EmitJump(d131.Condition, lbl50)
					ctx.EmitJmp(lbl51)
					ctx.FreeDesc(&d130)
					ctx.MarkLabel(lbl50)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl51)
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
						ctx.EmitCopyStackWords(d133, int32(phiBase19)+int32(0), 3)
					} else {
						if d133.Loc != LocRegTriple {
							panic("jit: slice phi source is not a triple")
						}
						ctx.EmitStoreRegMem(d133.Reg, ctx.StackReg, int32(phiBase19)+int32(0))
						ctx.EmitStoreRegMem(d133.Reg2, ctx.StackReg, int32(phiBase19)+int32(0)+8)
						ctx.EmitStoreRegMem(d133.Reg3, ctx.StackReg, int32(phiBase19)+int32(0)+16)
					}
					if d59.Loc == LocReg || d59.Loc == LocFPReg {
						ctx.UnprotectReg(d59.Reg)
					} else if d59.Loc == LocRegPair {
						ctx.UnprotectReg(d59.Reg)
						ctx.UnprotectReg(d59.Reg2)
					}
					ctx.EmitJmp(lbl7)
				}
				bbpos_2_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d92 = inlineHome94
				d95 = inlineHome96
				d117 = inlineHome118
				d125 = inlineHome126
				d128 = inlineHome129
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d124 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d127 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.SyncDesc(&d15)
				if d15.Loc == LocReg || d15.Loc == LocFPReg {
					ctx.ProtectReg(d15.Reg)
				} else if d15.Loc == LocRegPair {
					ctx.ProtectReg(d15.Reg)
					ctx.ProtectReg(d15.Reg2)
				}
				d134 := d15
				if d134.Loc == LocNone {
					panic("jit: phi source has no location")
				}
				ctx.SyncDesc(&d134)
				if d134.Loc == LocStackPair {
					ctx.EmitCopyStackWords(d134, int32(phiBase19)+int32(24), 2)
				} else if d134.Loc == LocInputPair {
					ctx.EnsureDesc(&d134)
					ctx.EmitStoreScmerToStack(d134, int32(phiBase19)+int32(24))
				} else if d134.Loc == LocRegPair || d134.Loc == LocImm {
					ctx.EmitStoreScmerToStack(d134, int32(phiBase19)+int32(24))
				} else {
					ctx.EnsureDesc(&d134)
					ctx.EmitStoreToStack(d134, int32(phiBase19)+int32(24))
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(phiBase19)+int32(24))+8)
				}
				if d15.Loc == LocReg || d15.Loc == LocFPReg {
					ctx.UnprotectReg(d15.Reg)
				} else if d15.Loc == LocRegPair {
					ctx.UnprotectReg(d15.Reg)
					ctx.UnprotectReg(d15.Reg2)
				}
				ctx.EmitJmp(lbl14)
				bbpos_2_14 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d92 = inlineHome94
				d95 = inlineHome96
				d117 = inlineHome118
				d125 = inlineHome126
				d128 = inlineHome129
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d124 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d127 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d128)
				ctx.ReclaimUntrackedRegs()
				d136 := ctx.EmitSliceElementAddress(&d17, &d128, 16)
				ctx.EnsureDesc(&d136)
				r35 := ctx.AllocRegExcept(d136.Reg)
				ctx.EmitMovRegMem(r35, d136.Reg, 8)
				ctx.EmitMovRegMem(d136.Reg, d136.Reg, 0)
				d135 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d136.Reg, Reg2: r35}
				ctx.BindReg(d136.Reg, &d135)
				ctx.BindReg(r35, &d135)
				ctx.ReclaimUntrackedRegs()
				d124 = JITPrepareScmerGoArg(ctx, d124)
				d135 = JITPrepareScmerGoArg(ctx, d135)
				ctx.SyncDesc(&d124)
				ctx.SyncDesc(&d135)
				d137 := ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d124, d135}, 1)
				d137.NoHeapPointer = true
				ctx.EmitAndRegImm32(d137.Reg, 1)
				d137.Type = tagBool
				ctx.BindReg(d137.Reg, &d137)
				ctx.FreeDesc(&d135)
				ctx.ReclaimUntrackedRegs()
				d138 := d137
				ctx.EnsureDesc(&d138)
				if d138.Loc != LocImm && d138.Loc != LocReg {
					panic("jit: If condition is neither LocImm nor LocReg")
				}
				lbl52 := ctx.ReserveLabel()
				lbl53 := ctx.ReserveLabel()
				if d138.Loc == LocImm {
					if d138.Imm.Bool() {
						ctx.MarkLabel(lbl52)
						ctx.EmitJmp(lbl17)
					} else {
						ctx.MarkLabel(lbl53)
						ctx.SyncDesc(&d128)
						if d128.Loc == LocReg || d128.Loc == LocFPReg {
							ctx.ProtectReg(d128.Reg)
						} else if d128.Loc == LocRegPair {
							ctx.ProtectReg(d128.Reg)
							ctx.ProtectReg(d128.Reg2)
						}
						d139 := d128
						if d139.Loc == LocNone {
							panic("jit: phi source has no location")
						}
						ctx.EnsureDesc(&d139)
						ctx.EmitStoreToStack(d139, int32(phiBase19)+int32(40))
						if d128.Loc == LocReg || d128.Loc == LocFPReg {
							ctx.UnprotectReg(d128.Reg)
						} else if d128.Loc == LocRegPair {
							ctx.UnprotectReg(d128.Reg)
							ctx.UnprotectReg(d128.Reg2)
						}
						ctx.EmitJmp(lbl15)
					}
				} else {
					ctx.EmitCmpRegImm32(d138.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl52)
					ctx.EmitJmp(lbl53)
					ctx.MarkLabel(lbl52)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl53)
					ctx.SyncDesc(&d128)
					if d128.Loc == LocReg || d128.Loc == LocFPReg {
						ctx.ProtectReg(d128.Reg)
					} else if d128.Loc == LocRegPair {
						ctx.ProtectReg(d128.Reg)
						ctx.ProtectReg(d128.Reg2)
					}
					d140 := d128
					if d140.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d140)
					ctx.EmitStoreToStack(d140, int32(phiBase19)+int32(40))
					if d128.Loc == LocReg || d128.Loc == LocFPReg {
						ctx.UnprotectReg(d128.Reg)
					} else if d128.Loc == LocRegPair {
						ctx.UnprotectReg(d128.Reg)
						ctx.UnprotectReg(d128.Reg2)
					}
					ctx.EmitJmp(lbl15)
				}
				ctx.FreeDesc(&d137)
				bbpos_2_15 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
				d9 = inlineHome10
				d11 = inlineHome12
				d13 = inlineHome14
				d15 = inlineHome16
				d17 = inlineHome18
				d28 = inlineHome29
				d53 = inlineHome54
				d59 = inlineHome60
				d66 = inlineHome67
				d72 = inlineHome73
				d92 = inlineHome94
				d95 = inlineHome96
				d117 = inlineHome118
				d125 = inlineHome126
				d128 = inlineHome129
				d31 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(0)}
				d124 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(24)}
				d127 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(40)}
				d75 = JITValueDesc{Loc: LocStackTriple, Type: JITTypeUnknown, StackOff: int32(phiBase19) + int32(56)}
				d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase19) + int32(80)}
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d141 := JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
				_ = d141
				ctx.EnsureDesc(&d141)
				if d141.Loc == LocRegPair {
					panic("jit: scalar inline return has LocRegPair")
				} else {
					ctx.EmitMovToReg(r2, d141)
				}
				ctx.EmitJmp(lbl1)
				ctx.MarkLabel(lbl1)
				d142 := JITValueDesc{Loc: LocReg, Reg: r2}
				ctx.BindReg(r2, &d142)
				ctx.BindReg(r2, &d142)
				ctx.FreeDesc(&d0)
				ctx.FreeDesc(&d1)
				ctx.FreeDesc(&d4)
				ctx.FreeDesc(&d6)
				ctx.SyncDesc(&d142)
				if ctx.hasBooleanFlags(d142) {
					return d142
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				if d142.Loc == LocImm {
					ctx.EmitMakeBool(result, d142)
				} else {
					ctx.EmitMakeBool(result, d142)
					ctx.FreeReg(d142.Reg)
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
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
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
					_ = d15
					ctx.EnsureDesc(&d15)
					d16 = d15
					_ = d16
					ctx.StabilizeDescForControlFlow(&d16)
					inlineHome17 := d16
					inlineHome17.ID = 0
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
					d16 = inlineHome17
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 32)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d18)
					}
					ctx.ReclaimUntrackedRegs()
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d19.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d18)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d16 = inlineHome17
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d20)
					if d20.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d20)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d16 = inlineHome17
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
					d21 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d21)
					ctx.BindReg(r1, &d21)
					ctx.FreeDesc(&d15)
					ctx.EnsureDesc(&d21)
					d22 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d21}, 1)
					ctx.StabilizeDescForControlFlow(&d22)
					ctx.FreeDesc(&d21)
					if d3.SliceSizeKnown {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d23 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d23 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d23 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d23
					ctx.StabilizeDescForControlFlow(&d23)
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
						d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d24)
					}
					if d24.Loc == LocReg && d1.Loc == LocReg && d24.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d24)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d24)
					ctx.EnsureDesc(&d23)
					ctx.EnsureDescsTogether(&d24, &d23)
					if d24.Loc == LocImm && d23.Loc == LocImm {
						d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d24.Imm.Int() < d23.Imm.Int())}
					} else if d23.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d24.Reg)
						if d23.Imm.Int() >= -2147483648 && d23.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d24.Reg, int32(d23.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d23.Imm.Int()))
							ctx.EmitCmpInt64(d24.Reg, ctx.ScratchReg)
						}
						d25 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d25)
					} else if d24.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d24.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d23.Reg)
						d25 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d25)
					} else {
						r4 := ctx.AllocRegExcept(d24.Reg)
						ctx.EmitCmpInt64(d24.Reg, d23.Reg)
						d25 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d25)
					}
					d26 = d25
					ctx.SyncDesc(&d26)
					if d26.Loc != LocImm && d26.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d26.Loc == LocImm {
						if d26.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d26.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d25)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap27 := d1
						snap28 := d2
						snap29 := d3
						snap30 := d4
						snap31 := d5
						snap32 := d7
						snap33 := d8
						snap34 := d10
						snap35 := d11
						snap36 := d15
						snap37 := d16
						snap38 := d18
						snap39 := d19
						snap40 := d20
						snap41 := d21
						snap42 := d22
						snap43 := d23
						snap44 := d24
						snap45 := d25
						snap46 := d26
						alloc47 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc47)
						d1 = snap27
						d2 = snap28
						d3 = snap29
						d4 = snap30
						d5 = snap31
						d7 = snap32
						d8 = snap33
						d10 = snap34
						d11 = snap35
						d15 = snap36
						d16 = snap37
						d18 = snap38
						d19 = snap39
						d20 = snap40
						d21 = snap41
						d22 = snap42
						d23 = snap43
						d24 = snap44
						d25 = snap45
						d26 = snap46
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
					ctx.EnsureDesc(&d24)
					d49 = ctx.EmitSliceElementAddress(&d3, &d24, 16)
					ctx.EnsureDesc(&d49)
					r5 := ctx.AllocRegExcept(d49.Reg)
					ctx.EmitMovRegMem(r5, d49.Reg, 8)
					ctx.EmitMovRegMem(d49.Reg, d49.Reg, 0)
					d48 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d49.Reg, Reg2: r5}
					ctx.BindReg(d49.Reg, &d48)
					ctx.BindReg(r5, &d48)
					ctx.SyncDesc(&d48)
					ctx.EmitStoreScmerToStack(d48, int32(stackArray13)+int32(0))
					d50 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d50
					callbackArgs52 := make([]JITValueDesc, 1)
					callbackArgs52[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray13) + 0}
					_ = d51
					callbackResultOff53 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff53))
					ctx.FreeDesc(&d50)
					ctx.StabilizeDescAcrossNestedCall(&d24)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs54 := ctx.StabilizeCallbackArgs(callbackArgs52)
						ctx.ReclaimUntrackedRegs()
						outerRegs55 := ctx.PreserveOuterRegs()
						d51 = JITEmitProcInlineWithOuter(ctx, d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs54, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff53), ID: 0})
						ctx.RestoreOuterRegs(outerRegs55)
						ctx.ReclaimUntrackedRegs()
					} else {
						d56, knownBuiltin57 := jitEmitKnownDeclaration(ctx, d5, callbackArgs52, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff53), ID: 0})
						if knownBuiltin57 {
							d51 = d56
						} else {
							ctx.Coverage.DynamicCalls++
							d58 = jitCopyScmerToPair(ctx, d5)
							d51 = jitEmitDynamicCallableAt(ctx, d58, callbackArgs52, int32(stackArray13), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff53), ID: 0})
						}
					}
					d59 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d59
					if d22.Loc == LocRegPair || d22.Loc == LocStackPair || d22.Loc == LocRegTriple || d22.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d51 = JITPrepareScmerGoArg(ctx, d51)
					d48 = JITPrepareScmerGoArg(ctx, d48)
					if d11.Loc == LocRegPair || d11.Loc == LocStackPair || d11.Loc == LocRegTriple || d11.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					if d8.Loc == LocRegPair || d8.Loc == LocStackPair || d8.Loc == LocRegTriple || d8.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d59 = JITPrepareGoSliceArg(ctx, d59)
					if d59.Loc != LocRegTriple && d59.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice ((*FastDict).ReduceValue arg5)")
					}
					ctx.SyncDesc(&d22)
					ctx.SyncDesc(&d51)
					ctx.SyncDesc(&d48)
					ctx.SyncDesc(&d11)
					ctx.SyncDesc(&d8)
					ctx.SyncDesc(&d59)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).ReduceValue), []JITValueDesc{d22, d51, d48, d11, d8, d59})
					ctx.FreeDesc(&d51)
					ctx.FreeDesc(&d48)
					ctx.SyncDesc(&d24)
					if d24.Loc == LocReg || d24.Loc == LocFPReg {
						ctx.ProtectReg(d24.Reg)
					} else if d24.Loc == LocRegPair {
						ctx.ProtectReg(d24.Reg)
						ctx.ProtectReg(d24.Reg2)
					}
					d60 = d24
					if d60.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d60)
					ctx.EmitStoreToStack(d60, int32(bbs[1].PhiBase)+int32(0))
					if d24.Loc == LocReg || d24.Loc == LocFPReg {
						ctx.UnprotectReg(d24.Reg)
					} else if d24.Loc == LocRegPair {
						ctx.UnprotectReg(d24.Reg)
						ctx.UnprotectReg(d24.Reg2)
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
					ctx.EnsureDesc(&d22)
					if d22.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d61 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d22.Reg, Reg2: r6}
						ctx.BindReg(d22.Reg, &d61)
						ctx.BindReg(r6, &d61)
						ctx.TransferReg(d22.Reg)
						ctx.BindReg(d22.Reg, &d61)
						ctx.BindReg(r6, &d61)
						d22.Loc = LocNone
					}
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
					_ = d12
					ctx.EnsureDesc(&d12)
					d13 = d12
					_ = d13
					ctx.StabilizeDescForControlFlow(&d13)
					inlineHome14 := d13
					inlineHome14.ID = 0
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
					d13 = inlineHome14
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d15 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d15)
					}
					ctx.ReclaimUntrackedRegs()
					d16 = d15
					ctx.SyncDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d16.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d15)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = inlineHome14
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d17)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d13 = inlineHome14
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
					d18 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d18)
					ctx.BindReg(r1, &d18)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d18)
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d18}, 1)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d18)
					if d3.SliceSizeKnown {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d20
					ctx.StabilizeDescForControlFlow(&d20)
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
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
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
					ctx.EnsureDesc(&d20)
					ctx.EnsureDescsTogether(&d21, &d20)
					if d21.Loc == LocImm && d20.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() < d20.Imm.Int())}
					} else if d20.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d21.Reg)
						if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d21.Reg, int32(d20.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d20.Imm.Int()))
							ctx.EmitCmpInt64(d21.Reg, ctx.ScratchReg)
						}
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d20.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d20.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d22)
					}
					d23 = d22
					ctx.SyncDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d23.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d22)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap24 := d1
						snap25 := d2
						snap26 := d3
						snap27 := d4
						snap28 := d5
						snap29 := d7
						snap30 := d8
						snap31 := d12
						snap32 := d13
						snap33 := d15
						snap34 := d16
						snap35 := d17
						snap36 := d18
						snap37 := d19
						snap38 := d20
						snap39 := d21
						snap40 := d22
						snap41 := d23
						alloc42 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc42)
						d1 = snap24
						d2 = snap25
						d3 = snap26
						d4 = snap27
						d5 = snap28
						d7 = snap29
						d8 = snap30
						d12 = snap31
						d13 = snap32
						d15 = snap33
						d16 = snap34
						d17 = snap35
						d18 = snap36
						d19 = snap37
						d20 = snap38
						d21 = snap39
						d22 = snap40
						d23 = snap41
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
					ctx.EnsureDesc(&d21)
					d44 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d44)
					r5 := ctx.AllocRegExcept(d44.Reg)
					ctx.EmitMovRegMem(r5, d44.Reg, 8)
					ctx.EmitMovRegMem(d44.Reg, d44.Reg, 0)
					d43 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d44.Reg, Reg2: r5}
					ctx.BindReg(d44.Reg, &d43)
					ctx.BindReg(r5, &d43)
					ctx.SyncDesc(&d43)
					ctx.EmitStoreScmerToStack(d43, int32(stackArray10)+int32(0))
					d45 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d45)
					ctx.EmitStoreScmerToStack(d45, int32(stackArray11)+int32(0))
					ctx.FreeDesc(&d45)
					ctx.SyncDesc(&d43)
					ctx.EmitStoreScmerToStack(d43, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d43)
					d46 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d46
					callbackArgs48 := make([]JITValueDesc, 1)
					callbackArgs48[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					_ = d47
					callbackResultOff49 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff49))
					ctx.FreeDesc(&d46)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs50 := ctx.StabilizeCallbackArgs(callbackArgs48)
						ctx.ReclaimUntrackedRegs()
						outerRegs51 := ctx.PreserveOuterRegs()
						d47 = JITEmitProcInlineWithOuter(ctx, d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs50, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff49), ID: 0})
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
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs59 := ctx.StabilizeCallbackArgs(callbackArgs57)
						ctx.ReclaimUntrackedRegs()
						outerRegs60 := ctx.PreserveOuterRegs()
						d56 = JITEmitProcInlineWithOuter(ctx, d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs59, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff58), ID: 0})
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
					if d19.Loc == LocRegPair || d19.Loc == LocStackPair || d19.Loc == LocRegTriple || d19.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d47 = JITPrepareScmerGoArg(ctx, d47)
					d56 = JITPrepareScmerGoArg(ctx, d56)
					ctx.SyncDesc(&d19)
					ctx.SyncDesc(&d47)
					ctx.SyncDesc(&d56)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d19, d47, d56})
					ctx.FreeDesc(&d47)
					ctx.FreeDesc(&d56)
					ctx.SyncDesc(&d21)
					if d21.Loc == LocReg || d21.Loc == LocFPReg {
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
					if d21.Loc == LocReg || d21.Loc == LocFPReg {
						ctx.UnprotectReg(d21.Reg)
					} else if d21.Loc == LocRegPair {
						ctx.UnprotectReg(d21.Reg)
						ctx.UnprotectReg(d21.Reg2)
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
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d65 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d19.Reg, Reg2: r6}
						ctx.BindReg(d19.Reg, &d65)
						ctx.BindReg(r6, &d65)
						ctx.TransferReg(d19.Reg)
						ctx.BindReg(d19.Reg, &d65)
						ctx.BindReg(r6, &d65)
						d19.Loc = LocNone
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
				var callbackResultOff50 int32
				var d53 JITValueDesc
				_ = d53
				var d55 JITValueDesc
				_ = d55
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
					_ = d12
					ctx.EnsureDesc(&d12)
					d13 = d12
					_ = d13
					ctx.StabilizeDescForControlFlow(&d13)
					inlineHome14 := d13
					inlineHome14.ID = 0
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
					d13 = inlineHome14
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d13.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d13.Reg)
						ctx.EmitCmpRegImm32(d13.Reg, 32)
						d15 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d15)
					}
					ctx.ReclaimUntrackedRegs()
					d16 = d15
					ctx.SyncDesc(&d16)
					if d16.Loc != LocImm && d16.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl9 := ctx.ReserveLabel()
					lbl10 := ctx.ReserveLabel()
					if d16.Loc == LocImm {
						if d16.Imm.Bool() {
							ctx.MarkLabel(lbl9)
							ctx.EmitJmp(lbl7)
						} else {
							ctx.MarkLabel(lbl10)
							ctx.EmitJmp(lbl8)
						}
					} else {
						ctx.EmitJump(d16.Condition, lbl9)
						ctx.EmitJmp(lbl10)
						ctx.FreeDesc(&d15)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d13 = inlineHome14
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d17)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d13 = inlineHome14
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
					d18 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d18)
					ctx.BindReg(r1, &d18)
					ctx.FreeDesc(&d12)
					ctx.EnsureDesc(&d18)
					d19 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d18}, 1)
					ctx.StabilizeDescForControlFlow(&d19)
					ctx.FreeDesc(&d18)
					if d3.SliceSizeKnown {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d20 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d20 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d20
					ctx.StabilizeDescForControlFlow(&d20)
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
						d21 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
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
					ctx.EnsureDesc(&d20)
					ctx.EnsureDescsTogether(&d21, &d20)
					if d21.Loc == LocImm && d20.Loc == LocImm {
						d22 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d21.Imm.Int() < d20.Imm.Int())}
					} else if d20.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d21.Reg)
						if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d21.Reg, int32(d20.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d20.Imm.Int()))
							ctx.EmitCmpInt64(d21.Reg, ctx.ScratchReg)
						}
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d22)
					} else if d21.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d21.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d20.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d22)
					} else {
						r4 := ctx.AllocRegExcept(d21.Reg)
						ctx.EmitCmpInt64(d21.Reg, d20.Reg)
						d22 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d22)
					}
					d23 = d22
					ctx.SyncDesc(&d23)
					if d23.Loc != LocImm && d23.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d23.Loc == LocImm {
						if d23.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d23.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d22)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap24 := d1
						snap25 := d2
						snap26 := d3
						snap27 := d4
						snap28 := d5
						snap29 := d7
						snap30 := d8
						snap31 := d12
						snap32 := d13
						snap33 := d15
						snap34 := d16
						snap35 := d17
						snap36 := d18
						snap37 := d19
						snap38 := d20
						snap39 := d21
						snap40 := d22
						snap41 := d23
						alloc42 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc42)
						d1 = snap24
						d2 = snap25
						d3 = snap26
						d4 = snap27
						d5 = snap28
						d7 = snap29
						d8 = snap30
						d12 = snap31
						d13 = snap32
						d15 = snap33
						d16 = snap34
						d17 = snap35
						d18 = snap36
						d19 = snap37
						d20 = snap38
						d21 = snap39
						d22 = snap40
						d23 = snap41
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
					ctx.EnsureDesc(&d21)
					d44 = ctx.EmitSliceElementAddress(&d3, &d21, 16)
					ctx.EnsureDesc(&d44)
					r5 := ctx.AllocRegExcept(d44.Reg)
					ctx.EmitMovRegMem(r5, d44.Reg, 8)
					ctx.EmitMovRegMem(d44.Reg, d44.Reg, 0)
					d43 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d44.Reg, Reg2: r5}
					ctx.BindReg(d44.Reg, &d43)
					ctx.BindReg(r5, &d43)
					d45 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d45)
					ctx.EmitStoreScmerToStack(d45, int32(stackArray10)+int32(0))
					ctx.FreeDesc(&d45)
					ctx.SyncDesc(&d43)
					ctx.EmitStoreScmerToStack(d43, int32(stackArray10)+int32(16))
					d46 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d46)
					ctx.EmitStoreScmerToStack(d46, int32(stackArray11)+int32(0))
					ctx.FreeDesc(&d46)
					ctx.SyncDesc(&d43)
					ctx.EmitStoreScmerToStack(d43, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d43)
					d47 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d47
					callbackArgs49 := make([]JITValueDesc, 2)
					callbackArgs49[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					callbackArgs49[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 16}
					_ = d48
					callbackResultOff50 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff50))
					ctx.FreeDesc(&d47)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs51 := ctx.StabilizeCallbackArgs(callbackArgs49)
						ctx.ReclaimUntrackedRegs()
						outerRegs52 := ctx.PreserveOuterRegs()
						d48 = JITEmitProcInlineWithOuter(ctx, d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs51, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff50), ID: 0})
						ctx.RestoreOuterRegs(outerRegs52)
						ctx.ReclaimUntrackedRegs()
					} else {
						d53, knownBuiltin54 := jitEmitKnownDeclaration(ctx, d5, callbackArgs49, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff50), ID: 0})
						if knownBuiltin54 {
							d48 = d53
						} else {
							ctx.Coverage.DynamicCalls++
							d55 = jitCopyScmerToPair(ctx, d5)
							d48 = jitEmitDynamicCallableAt(ctx, d55, callbackArgs49, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff50), ID: 0})
						}
					}
					d56 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d56
					callbackArgs58 := make([]JITValueDesc, 2)
					callbackArgs58[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 0}
					callbackArgs58[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 16}
					_ = d57
					callbackResultOff59 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff59))
					ctx.FreeDesc(&d56)
					ctx.StabilizeDescAcrossNestedCall(&d21)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs60 := ctx.StabilizeCallbackArgs(callbackArgs58)
						ctx.ReclaimUntrackedRegs()
						outerRegs61 := ctx.PreserveOuterRegs()
						d57 = JITEmitProcInlineWithOuter(ctx, d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs60, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						ctx.RestoreOuterRegs(outerRegs61)
						ctx.ReclaimUntrackedRegs()
					} else {
						d62, knownBuiltin63 := jitEmitKnownDeclaration(ctx, d8, callbackArgs58, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						if knownBuiltin63 {
							d57 = d62
						} else {
							ctx.Coverage.DynamicCalls++
							d64 = jitCopyScmerToPair(ctx, d8)
							d57 = jitEmitDynamicCallableAt(ctx, d64, callbackArgs58, int32(stackArray11), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff59), ID: 0})
						}
					}
					if d19.Loc == LocRegPair || d19.Loc == LocStackPair || d19.Loc == LocRegTriple || d19.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d48 = JITPrepareScmerGoArg(ctx, d48)
					d57 = JITPrepareScmerGoArg(ctx, d57)
					ctx.SyncDesc(&d19)
					ctx.SyncDesc(&d48)
					ctx.SyncDesc(&d57)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).AppendValue), []JITValueDesc{d19, d48, d57})
					ctx.FreeDesc(&d48)
					ctx.FreeDesc(&d57)
					ctx.SyncDesc(&d21)
					if d21.Loc == LocReg || d21.Loc == LocFPReg {
						ctx.ProtectReg(d21.Reg)
					} else if d21.Loc == LocRegPair {
						ctx.ProtectReg(d21.Reg)
						ctx.ProtectReg(d21.Reg2)
					}
					d65 = d21
					if d65.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d65)
					ctx.EmitStoreToStack(d65, int32(bbs[1].PhiBase)+int32(0))
					if d21.Loc == LocReg || d21.Loc == LocFPReg {
						ctx.UnprotectReg(d21.Reg)
					} else if d21.Loc == LocRegPair {
						ctx.UnprotectReg(d21.Reg)
						ctx.UnprotectReg(d21.Reg2)
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
					ctx.EnsureDesc(&d19)
					if d19.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d66 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d19.Reg, Reg2: r6}
						ctx.BindReg(d19.Reg, &d66)
						ctx.BindReg(r6, &d66)
						ctx.TransferReg(d19.Reg)
						ctx.BindReg(d19.Reg, &d66)
						ctx.BindReg(r6, &d66)
						d19.Loc = LocNone
					}
					ctx.SyncDesc(&d66)
					if d66.Loc == LocRegPair || d66.Loc == LocStackPair || d66.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d66, &result)
						result.Type = d66.Type
					} else {
						switch d66.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d66)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d66)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d66)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d66, &result)
							result.Type = d66.Type
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
				var d19 JITValueDesc
				_ = d19
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
					_ = d8
					ctx.EnsureDesc(&d8)
					d9 = d8
					_ = d9
					ctx.StabilizeDescForControlFlow(&d9)
					inlineHome10 := d9
					inlineHome10.ID = 0
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
					d9 = inlineHome10
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d9.Reg)
						ctx.EmitCmpRegImm32(d9.Reg, 32)
						d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d11)
					}
					ctx.ReclaimUntrackedRegs()
					d12 = d11
					ctx.SyncDesc(&d12)
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
						ctx.FreeDesc(&d11)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d9 = inlineHome10
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d9 = inlineHome10
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
					d14 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r1, &d14)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d14}, 1)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.FreeDesc(&d14)
					if d3.SliceSizeKnown {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d16
					ctx.StabilizeDescForControlFlow(&d16)
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
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d17)
					}
					if d17.Loc == LocReg && d1.Loc == LocReg && d17.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDescsTogether(&d17, &d16)
					if d17.Loc == LocImm && d16.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() < d16.Imm.Int())}
					} else if d16.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d17.Reg)
						if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d17.Reg, int32(d16.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d16.Imm.Int()))
							ctx.EmitCmpInt64(d17.Reg, ctx.ScratchReg)
						}
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d18)
					} else if d17.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d17.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d16.Reg)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d18)
					} else {
						r4 := ctx.AllocRegExcept(d17.Reg)
						ctx.EmitCmpInt64(d17.Reg, d16.Reg)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d18)
					}
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d19.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d18)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d4
						snap24 := d5
						snap25 := d8
						snap26 := d9
						snap27 := d11
						snap28 := d12
						snap29 := d13
						snap30 := d14
						snap31 := d15
						snap32 := d16
						snap33 := d17
						snap34 := d18
						snap35 := d19
						alloc36 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc36)
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d4 = snap23
						d5 = snap24
						d8 = snap25
						d9 = snap26
						d11 = snap27
						d12 = snap28
						d13 = snap29
						d14 = snap30
						d15 = snap31
						d16 = snap32
						d17 = snap33
						d18 = snap34
						d19 = snap35
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
					ctx.EnsureDesc(&d17)
					d38 = ctx.EmitSliceElementAddress(&d3, &d17, 16)
					ctx.EnsureDesc(&d38)
					r5 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegMem(r5, d38.Reg, 8)
					ctx.EmitMovRegMem(d38.Reg, d38.Reg, 0)
					d37 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d38.Reg, Reg2: r5}
					ctx.BindReg(d38.Reg, &d37)
					ctx.BindReg(r5, &d37)
					ctx.SyncDesc(&d37)
					ctx.EmitStoreScmerToStack(d37, int32(stackArray7)+int32(0))
					ctx.FreeDesc(&d37)
					d39 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
					_ = d39
					callbackArgs41 := make([]JITValueDesc, 1)
					callbackArgs41[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 0}
					_ = d40
					callbackResultOff42 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff42))
					ctx.FreeDesc(&d39)
					ctx.StabilizeDescAcrossNestedCall(&d17)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs43 := ctx.StabilizeCallbackArgs(callbackArgs41)
						ctx.ReclaimUntrackedRegs()
						outerRegs44 := ctx.PreserveOuterRegs()
						d40 = JITEmitProcInlineWithOuter(ctx, d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs43, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff42), ID: 0})
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
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d40 = JITPrepareScmerGoArg(ctx, d40)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d40)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d15, d40})
					ctx.FreeDesc(&d40)
					ctx.SyncDesc(&d17)
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.ProtectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.ProtectReg(d17.Reg)
						ctx.ProtectReg(d17.Reg2)
					}
					d48 = d17
					if d48.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d48)
					ctx.EmitStoreToStack(d48, int32(bbs[1].PhiBase)+int32(0))
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.UnprotectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.UnprotectReg(d17.Reg)
						ctx.UnprotectReg(d17.Reg2)
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
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d49 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d15.Reg, Reg2: r6}
						ctx.BindReg(d15.Reg, &d49)
						ctx.BindReg(r6, &d49)
						ctx.TransferReg(d15.Reg)
						ctx.BindReg(d15.Reg, &d49)
						ctx.BindReg(r6, &d49)
						d15.Loc = LocNone
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
				var d19 JITValueDesc
				_ = d19
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
				var callbackResultOff43 int32
				var d46 JITValueDesc
				_ = d46
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
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
					_ = d8
					ctx.EnsureDesc(&d8)
					d9 = d8
					_ = d9
					ctx.StabilizeDescForControlFlow(&d9)
					inlineHome10 := d9
					inlineHome10.ID = 0
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
					d9 = inlineHome10
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d9)
					if d9.Loc == LocImm {
						d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() < 32)}
					} else {
						r0 := ctx.AllocRegExcept(d9.Reg)
						ctx.EmitCmpRegImm32(d9.Reg, 32)
						d11 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondSignedLess}
						ctx.BindReg(r0, &d11)
					}
					ctx.ReclaimUntrackedRegs()
					d12 = d11
					ctx.SyncDesc(&d12)
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
						ctx.FreeDesc(&d11)
						ctx.MarkLabel(lbl9)
						ctx.EmitJmp(lbl7)
						ctx.MarkLabel(lbl10)
						ctx.EmitJmp(lbl8)
					}
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d9 = inlineHome10
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					d13 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(32)}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocRegPair {
						panic("jit: scalar inline return has LocRegPair")
					} else {
						ctx.EmitMovToReg(r1, d13)
					}
					ctx.EmitJmp(lbl5)
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl7)
					ctx.ResolveFixups()
					d9 = inlineHome10
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
					d14 = JITValueDesc{Loc: LocReg, Reg: r1}
					ctx.BindReg(r1, &d14)
					ctx.BindReg(r1, &d14)
					ctx.FreeDesc(&d8)
					ctx.EnsureDesc(&d14)
					d15 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d14}, 1)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.FreeDesc(&d14)
					if d3.SliceSizeKnown {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d16 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d16
					ctx.StabilizeDescForControlFlow(&d16)
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
						d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d17)
					}
					if d17.Loc == LocReg && d1.Loc == LocReg && d17.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d17)
					ctx.EnsureDesc(&d16)
					ctx.EnsureDescsTogether(&d17, &d16)
					if d17.Loc == LocImm && d16.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.Int() < d16.Imm.Int())}
					} else if d16.Loc == LocImm {
						r2 := ctx.AllocRegExcept(d17.Reg)
						if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d17.Reg, int32(d16.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d16.Imm.Int()))
							ctx.EmitCmpInt64(d17.Reg, ctx.ScratchReg)
						}
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondSignedLess}
						ctx.BindReg(r2, &d18)
					} else if d17.Loc == LocImm {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d17.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d16.Reg)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r3, Condition: CondSignedLess}
						ctx.BindReg(r3, &d18)
					} else {
						r4 := ctx.AllocRegExcept(d17.Reg)
						ctx.EmitCmpInt64(d17.Reg, d16.Reg)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r4, Condition: CondSignedLess}
						ctx.BindReg(r4, &d18)
					}
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[2].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d19.Condition, lbl3)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d18)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d4
						snap24 := d5
						snap25 := d8
						snap26 := d9
						snap27 := d11
						snap28 := d12
						snap29 := d13
						snap30 := d14
						snap31 := d15
						snap32 := d16
						snap33 := d17
						snap34 := d18
						snap35 := d19
						alloc36 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc36)
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d4 = snap23
						d5 = snap24
						d8 = snap25
						d9 = snap26
						d11 = snap27
						d12 = snap28
						d13 = snap29
						d14 = snap30
						d15 = snap31
						d16 = snap32
						d17 = snap33
						d18 = snap34
						d19 = snap35
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
					ctx.EnsureDesc(&d17)
					d38 = ctx.EmitSliceElementAddress(&d3, &d17, 16)
					ctx.EnsureDesc(&d38)
					r5 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegMem(r5, d38.Reg, 8)
					ctx.EmitMovRegMem(d38.Reg, d38.Reg, 0)
					d37 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d38.Reg, Reg2: r5}
					ctx.BindReg(d38.Reg, &d37)
					ctx.BindReg(r5, &d37)
					d39 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.SyncDesc(&d39)
					ctx.EmitStoreScmerToStack(d39, int32(stackArray7)+int32(0))
					ctx.FreeDesc(&d39)
					ctx.SyncDesc(&d37)
					ctx.EmitStoreScmerToStack(d37, int32(stackArray7)+int32(16))
					ctx.FreeDesc(&d37)
					d40 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d40
					callbackArgs42 := make([]JITValueDesc, 2)
					callbackArgs42[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 0}
					callbackArgs42[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray7) + 16}
					_ = d41
					callbackResultOff43 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff43))
					ctx.FreeDesc(&d40)
					ctx.StabilizeDescAcrossNestedCall(&d17)
					if d5.Loc == LocLambdaTemplate && d5.Lambda != nil {
						stableCallbackArgs44 := ctx.StabilizeCallbackArgs(callbackArgs42)
						ctx.ReclaimUntrackedRegs()
						outerRegs45 := ctx.PreserveOuterRegs()
						d41 = JITEmitProcInlineWithOuter(ctx, d5.Lambda.Proc, d5.Lambda.Outer, stableCallbackArgs44, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff43), ID: 0})
						ctx.RestoreOuterRegs(outerRegs45)
						ctx.ReclaimUntrackedRegs()
					} else {
						d46, knownBuiltin47 := jitEmitKnownDeclaration(ctx, d5, callbackArgs42, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff43), ID: 0})
						if knownBuiltin47 {
							d41 = d46
						} else {
							ctx.Coverage.DynamicCalls++
							d48 = jitCopyScmerToPair(ctx, d5)
							d41 = jitEmitDynamicCallableAt(ctx, d48, callbackArgs42, int32(stackArray7), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff43), ID: 0})
						}
					}
					if d15.Loc == LocRegPair || d15.Loc == LocStackPair || d15.Loc == LocRegTriple || d15.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					d41 = JITPrepareScmerGoArg(ctx, d41)
					ctx.SyncDesc(&d15)
					ctx.SyncDesc(&d41)
					ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).IncrementCount), []JITValueDesc{d15, d41})
					ctx.FreeDesc(&d41)
					ctx.SyncDesc(&d17)
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.ProtectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.ProtectReg(d17.Reg)
						ctx.ProtectReg(d17.Reg2)
					}
					d49 = d17
					if d49.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d49)
					ctx.EmitStoreToStack(d49, int32(bbs[1].PhiBase)+int32(0))
					if d17.Loc == LocReg || d17.Loc == LocFPReg {
						ctx.UnprotectReg(d17.Reg)
					} else if d17.Loc == LocRegPair {
						ctx.UnprotectReg(d17.Reg)
						ctx.UnprotectReg(d17.Reg2)
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
					ctx.EnsureDesc(&d15)
					if d15.Loc == LocImm {
						panic("NewFastDict: LocImm not expected at JIT compile time")
					} else {
						r6 := ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, makeAux(tagFastDict, 0))
						d50 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d15.Reg, Reg2: r6}
						ctx.BindReg(d15.Reg, &d50)
						ctx.BindReg(r6, &d50)
						ctx.TransferReg(d15.Reg)
						ctx.BindReg(d15.Reg, &d50)
						ctx.BindReg(r6, &d50)
						d15.Loc = LocNone
					}
					ctx.SyncDesc(&d50)
					if d50.Loc == LocRegPair || d50.Loc == LocStackPair || d50.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d50, &result)
						result.Type = d50.Type
					} else {
						switch d50.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d50)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d50)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d50)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d50, &result)
							result.Type = d50.Type
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
				declaration := declarations["mapkey_assoc"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var stackArray5 int32
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var stackArray10 int32
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
				var d34 JITValueDesc
				_ = d34
				var d35 JITValueDesc
				_ = d35
				var phiBase37 int32
				_ = phiBase37
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
				var callbackResultOff56 int32
				var d59 JITValueDesc
				_ = d59
				var d61 JITValueDesc
				_ = d61
				var d62 JITValueDesc
				_ = d62
				var d63 JITValueDesc
				_ = d63
				var d64 JITValueDesc
				_ = d64
				var callbackResultOff66 int32
				var d69 JITValueDesc
				_ = d69
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d76 JITValueDesc
				_ = d76
				var d77 JITValueDesc
				_ = d77
				var d78 JITValueDesc
				_ = d78
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
				var callbackResultOff134 int32
				var d137 JITValueDesc
				_ = d137
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
				var callbackResultOff147 int32
				var d150 JITValueDesc
				_ = d150
				var d152 JITValueDesc
				_ = d152
				var d153 JITValueDesc
				_ = d153
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
				var bbs [6]BBDescriptor
				bbs[4].PhiBase = int32(phiBase0) + int32(0)
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
				bbpos_0_4 := int32(-1)
				_ = bbpos_0_4
				lbl5 := ctx.ReserveLabel()
				_ = lbl5
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				_ = lbl6
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
					d2 = args[1]
					d2.ID = 0
					if d2.Loc == LocLambdaTemplate {
						d3 = d2
					} else if d2.Loc == LocImm {
						optimizedCallback4 := jitPrepareCallback(d2.Imm)
						ctx.TrackImm(optimizedCallback4)
						d3 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback4, Rooted: true}
					} else {
						if d2.Loc == LocInputPair && int(d2.StackOff) < ctx.InputArgCount {
							d3 = ctx.RequestOptimizedCallback(int(d2.StackOff))
						} else {
							d3 = jitCopyScmerToPair(ctx, d2)
						}
					}
					ctx.FreeDesc(&d2)
					ctx.EnsureDesc(&d3)
					ctx.StabilizeDescForControlFlow(&d3)
					stackArray5 = ctx.AllocStack(int32(32))
					_ = stackArray5
					globalLookup6 := Globalenv.Vars[Symbol("set_assoc")]
					ctx.TrackImm(globalLookup6)
					d7 = JITValueDesc{Loc: LocImm, Type: globalLookup6.GetTag(), Imm: globalLookup6, Rooted: true}
					optimizedCallback9 := jitPrepareCallback(d7.Imm)
					ctx.TrackImm(optimizedCallback9)
					d8 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback9, Rooted: true}
					ctx.EnsureDesc(&d8)
					ctx.StabilizeDescForControlFlow(&d8)
					stackArray10 = ctx.AllocStack(int32(48))
					_ = stackArray10
					r0 := ctx.AllocReg()
					r1 := ctx.AllocRegExcept(r0)
					ctx.EmitMovRegImm64(r0, 0)
					ctx.EmitMovRegImm64(r1, 0)
					d11 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r0, Reg2: r1}
					ctx.BindReg(r0, &d11)
					ctx.BindReg(r1, &d11)
					ctx.StabilizeDescForControlFlow(&d11)
					d12 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: nil}
					ctx.SyncDesc(&d12)
					d13 = args[0]
					d13.ID = 0
					d13 = JITPrepareScmerGoArg(ctx, d13)
					d14 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("mapkey_assoc")}
					ctx.EnsureDesc(&d14)
					if d14.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d14.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d14.Imm)
						ptrWord, _ := d14.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d14.Imm.String())))
						d14 = tmpPair
					} else if d14.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d14.Type, Reg: ctx.AllocRegExcept(d14.Reg), Reg2: ctx.AllocRegExcept(d14.Reg)}
						switch d14.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d14)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d14)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d14)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d14)
						d14 = tmpPair
					}
					if d14.Loc != LocRegPair && d14.Loc != LocStackPair && d14.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (asAssoc arg1)")
					}
					ctx.SyncDesc(&d13)
					ctx.SyncDesc(&d14)
					callResults15 := JITEmitGoCallResults(ctx, GoFuncAddr(asAssoc), []JITValueDesc{d13, d14}, []uint8{3, 1}, []uint8{1, 1})
					ctx.FreeDesc(&d14)
					d16 = callResults15[0]
					_ = d16
					d17 = callResults15[1]
					_ = d17
					ctx.FreeDesc(&d13)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.EnsureDesc(&d17)
					if d17.Loc == LocImm {
						d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d17.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d17)
						if d17.Loc != LocReg && d17.Loc != LocRegPair && d17.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r2 := ctx.AllocRegExcept(d17.Reg)
						ctx.EmitCmpRegImm32(d17.Reg, 0)
						d18 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r2, Condition: CondEqual}
						ctx.BindReg(r2, &d18)
					}
					d19 = d18
					ctx.SyncDesc(&d19)
					if d19.Loc != LocImm && d19.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d19.Loc == LocImm {
						if d19.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[3].Render()
					}
					ctx.EmitJump(d19.Condition, lbl2)
					if bbs[3].Rendered {
						ctx.EmitJmp(lbl4)
					}
					ctx.FreeDesc(&d18)
					ctx.FlushRegisterMoves()
					if !bbs[3].Rendered {
						snap20 := d1
						snap21 := d2
						snap22 := d3
						snap23 := d7
						snap24 := d8
						snap25 := d11
						snap26 := d12
						snap27 := d13
						snap28 := d14
						snap29 := d16
						snap30 := d17
						snap31 := d18
						snap32 := d19
						alloc33 := ctx.SnapshotAllocState()
						bbs[3].Render()
						ctx.RestoreAllocState(alloc33)
						d1 = snap20
						d2 = snap21
						d3 = snap22
						d7 = snap23
						d8 = snap24
						d11 = snap25
						d12 = snap26
						d13 = snap27
						d14 = snap28
						d16 = snap29
						d17 = snap30
						d18 = snap31
						d19 = snap32
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[4].PhiBase)+int32(0))
					return bbs[4].Render()
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
					d34 = d12
					_ = d34
					ctx.SyncDesc(&d34)
					if d34.Loc == LocRegPair || d34.Loc == LocStackPair || d34.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d34, &result)
						result.Type = d34.Type
					} else {
						switch d34.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d34)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d34)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d34)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d34, &result)
							result.Type = d34.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d17)
					d35 = d17
					_ = d35
					ctx.StabilizeDescForControlFlow(&d35)
					inlineHome36 := d35
					inlineHome36.ID = 0
					phiBase37 = ctx.AllocStack(int32(16))
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase37) + int32(0)}
					_ = d38
					lbl7 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_1_4 := int32(-1)
					_ = bbpos_1_4
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_1_5 := int32(-1)
					_ = bbpos_1_5
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d35 = inlineHome36
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase37) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase37)+int32(0))
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d35 = inlineHome36
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase37) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						fieldAddr := uintptr(d35.Imm.Int()) + 0
						r3 := ctx.AllocReg()
						r4 := ctx.AllocRegExcept(r3)
						r5 := ctx.AllocRegExcept(r3, r4)
						ctx.EmitMovRegMem64(r3, fieldAddr)
						ctx.EmitMovRegMem64(r4, fieldAddr+8)
						ctx.EmitMovRegMem64(r5, fieldAddr+16)
						d39 = JITValueDesc{Loc: LocRegTriple, Reg: r3, Reg2: r4, Reg3: r5}
						ctx.BindReg(r3, &d39)
						ctx.BindReg(r4, &d39)
						ctx.BindReg(r5, &d39)
					} else {
						off := int32(0)
						baseReg := d35.Reg
						r6 := ctx.AllocRegExcept(baseReg)
						r7 := ctx.AllocRegExcept(baseReg, r6)
						r8 := ctx.AllocRegExcept(baseReg, r6, r7)
						ctx.EmitMovRegMem(r6, baseReg, off)
						ctx.EmitMovRegMem(r7, baseReg, off+8)
						ctx.EmitMovRegMem(r8, baseReg, off+16)
						d39 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
						ctx.BindReg(r6, &d39)
						ctx.BindReg(r7, &d39)
						ctx.BindReg(r8, &d39)
					}
					ctx.ReclaimUntrackedRegs()
					if d39.SliceSizeKnown {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.KnownSliceLen))}
					} else if d39.Loc == LocImm {
						d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d39.StackOff))}
					} else if d39.Loc == LocStackTriple {
						d40 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d39.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d39)
						if d39.Loc == LocRegPair || d39.Loc == LocRegTriple {
							d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg2, ID: 0}
						} else if d39.Loc == LocReg {
							d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d39.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d40
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d40)
					ctx.EnsureDescsTogether(&d38, &d40)
					if d38.Loc == LocImm && d40.Loc == LocImm {
						d41 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d38.Imm.Int() < d40.Imm.Int())}
					} else if d40.Loc == LocImm {
						r9 := ctx.AllocRegExcept(d38.Reg)
						if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d38.Reg, int32(d40.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d40.Imm.Int()))
							ctx.EmitCmpInt64(d38.Reg, ctx.ScratchReg)
						}
						d41 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r9, Condition: CondSignedLess}
						ctx.BindReg(r9, &d41)
					} else if d38.Loc == LocImm {
						r10 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d38.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d40.Reg)
						d41 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r10, Condition: CondSignedLess}
						ctx.BindReg(r10, &d41)
					} else {
						r11 := ctx.AllocRegExcept(d38.Reg)
						ctx.EmitCmpInt64(d38.Reg, d40.Reg)
						d41 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r11, Condition: CondSignedLess}
						ctx.BindReg(r11, &d41)
					}
					ctx.FreeDesc(&d40)
					ctx.ReclaimUntrackedRegs()
					d42 = d41
					ctx.SyncDesc(&d42)
					if d42.Loc != LocImm && d42.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d42.Loc == LocImm {
						if d42.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d42.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d41)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl11)
					}
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d35 = inlineHome36
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase37) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d35 = inlineHome36
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase37) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						fieldAddr := uintptr(d35.Imm.Int()) + 0
						r12 := ctx.AllocReg()
						r13 := ctx.AllocRegExcept(r12)
						r14 := ctx.AllocRegExcept(r12, r13)
						ctx.EmitMovRegMem64(r12, fieldAddr)
						ctx.EmitMovRegMem64(r13, fieldAddr+8)
						ctx.EmitMovRegMem64(r14, fieldAddr+16)
						d43 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
						ctx.BindReg(r12, &d43)
						ctx.BindReg(r13, &d43)
						ctx.BindReg(r14, &d43)
					} else {
						off := int32(0)
						baseReg := d35.Reg
						r15 := ctx.AllocRegExcept(baseReg)
						r16 := ctx.AllocRegExcept(baseReg, r15)
						r17 := ctx.AllocRegExcept(baseReg, r15, r16)
						ctx.EmitMovRegMem(r15, baseReg, off)
						ctx.EmitMovRegMem(r16, baseReg, off+8)
						ctx.EmitMovRegMem(r17, baseReg, off+16)
						d43 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
						ctx.BindReg(r15, &d43)
						ctx.BindReg(r16, &d43)
						ctx.BindReg(r17, &d43)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d38)
					ctx.ReclaimUntrackedRegs()
					d45 = ctx.EmitSliceElementAddress(&d43, &d38, 16)
					ctx.EnsureDesc(&d45)
					r18 := ctx.AllocRegExcept(d45.Reg)
					ctx.EmitMovRegMem(r18, d45.Reg, 8)
					ctx.EmitMovRegMem(d45.Reg, d45.Reg, 0)
					d44 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d45.Reg, Reg2: r18}
					ctx.BindReg(d45.Reg, &d44)
					ctx.BindReg(r18, &d44)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d35)
					if d35.Loc == LocImm {
						fieldAddr := uintptr(d35.Imm.Int()) + 0
						r19 := ctx.AllocReg()
						r20 := ctx.AllocRegExcept(r19)
						r21 := ctx.AllocRegExcept(r19, r20)
						ctx.EmitMovRegMem64(r19, fieldAddr)
						ctx.EmitMovRegMem64(r20, fieldAddr+8)
						ctx.EmitMovRegMem64(r21, fieldAddr+16)
						d46 = JITValueDesc{Loc: LocRegTriple, Reg: r19, Reg2: r20, Reg3: r21}
						ctx.BindReg(r19, &d46)
						ctx.BindReg(r20, &d46)
						ctx.BindReg(r21, &d46)
					} else {
						off := int32(0)
						baseReg := d35.Reg
						r22 := ctx.AllocRegExcept(baseReg)
						r23 := ctx.AllocRegExcept(baseReg, r22)
						r24 := ctx.AllocRegExcept(baseReg, r22, r23)
						ctx.EmitMovRegMem(r22, baseReg, off)
						ctx.EmitMovRegMem(r23, baseReg, off+8)
						ctx.EmitMovRegMem(r24, baseReg, off+16)
						d46 = JITValueDesc{Loc: LocRegTriple, Reg: r22, Reg2: r23, Reg3: r24}
						ctx.BindReg(r22, &d46)
						ctx.BindReg(r23, &d46)
						ctx.BindReg(r24, &d46)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d38.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d38.Reg)
						ctx.EmitMovRegReg(scratch, d38.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d47)
					}
					if d47.Loc == LocReg && d38.Loc == LocReg && d47.Reg == d38.Reg {
						ctx.TransferReg(d38.Reg)
						d38.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d47)
					ctx.ReclaimUntrackedRegs()
					d49 = ctx.EmitSliceElementAddress(&d46, &d47, 16)
					ctx.EnsureDesc(&d49)
					r25 := ctx.AllocRegExcept(d49.Reg)
					ctx.EmitMovRegMem(r25, d49.Reg, 8)
					ctx.EmitMovRegMem(d49.Reg, d49.Reg, 0)
					d48 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d49.Reg, Reg2: r25}
					ctx.BindReg(d49.Reg, &d48)
					ctx.BindReg(r25, &d48)
					ctx.FreeDesc(&d47)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d48)
					d50 = d44
					_ = d50
					d51 = d48
					_ = d51
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d50)
					ctx.EmitStoreScmerToStack(d50, int32(stackArray5)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray5)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d52 = d12
					_ = d52
					ctx.ReclaimUntrackedRegs()
					d53 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d53
					ctx.ReclaimUntrackedRegs()
					callbackArgs55 := make([]JITValueDesc, 2)
					callbackArgs55[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray5) + 0}
					callbackArgs55[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray5) + 16}
					_ = d54
					callbackResultOff56 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff56))
					ctx.FreeDesc(&d53)
					if d3.Loc == LocLambdaTemplate && d3.Lambda != nil {
						stableCallbackArgs57 := ctx.StabilizeCallbackArgs(callbackArgs55)
						ctx.ReclaimUntrackedRegs()
						outerRegs58 := ctx.PreserveOuterRegs()
						d54 = JITEmitProcInlineWithOuter(ctx, d3.Lambda.Proc, d3.Lambda.Outer, stableCallbackArgs57, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff56), ID: 0})
						ctx.RestoreOuterRegs(outerRegs58)
						ctx.ReclaimUntrackedRegs()
					} else {
						d59, knownBuiltin60 := jitEmitKnownDeclaration(ctx, d3, callbackArgs55, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff56), ID: 0})
						if knownBuiltin60 {
							d54 = d59
						} else {
							ctx.Coverage.DynamicCalls++
							d61 = jitCopyScmerToPair(ctx, d3)
							d54 = jitEmitDynamicCallableAt(ctx, d61, callbackArgs55, int32(stackArray5), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff56), ID: 0})
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d52)
					d62 = jitMaterializeVirtualSlice(ctx, d52, JITValueDesc{Loc: LocAny})
					ctx.EmitStoreScmerToStack(d62, int32(stackArray10)+int32(0))
					ctx.FreeDesc(&d62)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d54)
					ctx.EmitStoreScmerToStack(d54, int32(stackArray10)+int32(16))
					ctx.FreeDesc(&d54)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d51)
					ctx.EmitStoreScmerToStack(d51, int32(stackArray10)+int32(32))
					ctx.ReclaimUntrackedRegs()
					d63 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d63
					ctx.ReclaimUntrackedRegs()
					callbackArgs65 := make([]JITValueDesc, 3)
					callbackArgs65[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					callbackArgs65[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 16}
					callbackArgs65[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 32}
					_ = d64
					callbackResultOff66 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff66))
					ctx.FreeDesc(&d63)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs67 := ctx.StabilizeCallbackArgs(callbackArgs65)
						ctx.ReclaimUntrackedRegs()
						outerRegs68 := ctx.PreserveOuterRegs()
						d64 = JITEmitProcInlineWithOuter(ctx, d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs67, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff66), ID: 0})
						ctx.RestoreOuterRegs(outerRegs68)
						ctx.ReclaimUntrackedRegs()
					} else {
						d69, knownBuiltin70 := jitEmitKnownDeclaration(ctx, d8, callbackArgs65, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff66), ID: 0})
						if knownBuiltin70 {
							d64 = d69
						} else {
							ctx.Coverage.DynamicCalls++
							d71 = jitCopyScmerToPair(ctx, d8)
							d64 = jitEmitDynamicCallableAt(ctx, d71, callbackArgs65, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff66), ID: 0})
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d64)
					ctx.FreeDesc(&d64)
					ctx.FreeDesc(&d64)
					ctx.ReclaimUntrackedRegs()
					d72 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.FreeDesc(&d44)
					ctx.FreeDesc(&d48)
					ctx.ReclaimUntrackedRegs()
					d73 = d72
					ctx.EnsureDesc(&d73)
					if d73.Loc != LocImm && d73.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					if d73.Loc == LocImm {
						if d73.Imm.Bool() {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl18)
							ctx.EmitJmp(lbl12)
						}
					} else {
						ctx.EmitCmpRegImm32(d73.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl17)
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl18)
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d72)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d35 = inlineHome36
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase37) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d35 = inlineHome36
					d38 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase37) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocImm {
						d74 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d38.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d38.Reg)
						ctx.EmitMovRegReg(scratch, d38.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 2)
						d74 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d74)
					}
					if d74.Loc == LocReg && d38.Loc == LocReg && d74.Reg == d38.Reg {
						ctx.TransferReg(d38.Reg)
						d38.Loc = LocNone
					}
					ctx.EnsureDesc(&d74)
					ctx.EmitStoreToStack(d74, int32(phiBase37)+int32(0))
					ctx.StabilizeDescForControlFlow(&d74)
					inlineHome75 := d74
					inlineHome75.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl7)
					return bbs[2].Render()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					if d16.SliceSizeKnown {
						d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d16.KnownSliceLen))}
					} else if d16.Loc == LocImm {
						d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d16.StackOff))}
					} else if d16.Loc == LocStackTriple {
						d76 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d16.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d16)
						if d16.Loc == LocRegPair || d16.Loc == LocRegTriple {
							d76 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg2, ID: 0}
						} else if d16.Loc == LocReg {
							d76 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d76
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d76)
					ctx.EnsureDescsTogether(&d1, &d76)
					if d1.Loc == LocImm && d76.Loc == LocImm {
						d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d76.Imm.Int())}
					} else if d76.Loc == LocImm {
						r26 := ctx.AllocRegExcept(d1.Reg)
						if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d76.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d76.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
						}
						d77 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r26, Condition: CondSignedLess}
						ctx.BindReg(r26, &d77)
					} else if d1.Loc == LocImm {
						r27 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d76.Reg)
						d77 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r27, Condition: CondSignedLess}
						ctx.BindReg(r27, &d77)
					} else {
						r28 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d76.Reg)
						d77 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r28, Condition: CondSignedLess}
						ctx.BindReg(r28, &d77)
					}
					ctx.FreeDesc(&d76)
					d78 = d77
					ctx.SyncDesc(&d78)
					if d78.Loc != LocImm && d78.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d78.Loc == LocImm {
						if d78.Imm.Bool() {
							return bbs[5].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d78.Condition, lbl6)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d77)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap79 := d1
						snap80 := d2
						snap81 := d3
						snap82 := d7
						snap83 := d8
						snap84 := d11
						snap85 := d12
						snap86 := d13
						snap87 := d14
						snap88 := d16
						snap89 := d17
						snap90 := d18
						snap91 := d19
						snap92 := d34
						snap93 := d35
						snap94 := d38
						snap95 := d39
						snap96 := d40
						snap97 := d41
						snap98 := d42
						snap99 := d43
						snap100 := d44
						snap101 := d45
						snap102 := d46
						snap103 := d47
						snap104 := d48
						snap105 := d49
						snap106 := d50
						snap107 := d51
						snap108 := d52
						snap109 := d53
						snap110 := d54
						snap111 := d59
						snap112 := d61
						snap113 := d62
						snap114 := d63
						snap115 := d64
						snap116 := d69
						snap117 := d71
						snap118 := d72
						snap119 := d73
						snap120 := d74
						snap121 := d76
						snap122 := d77
						snap123 := d78
						alloc124 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc124)
						d1 = snap79
						d2 = snap80
						d3 = snap81
						d7 = snap82
						d8 = snap83
						d11 = snap84
						d12 = snap85
						d13 = snap86
						d14 = snap87
						d16 = snap88
						d17 = snap89
						d18 = snap90
						d19 = snap91
						d34 = snap92
						d35 = snap93
						d38 = snap94
						d39 = snap95
						d40 = snap96
						d41 = snap97
						d42 = snap98
						d43 = snap99
						d44 = snap100
						d45 = snap101
						d46 = snap102
						d47 = snap103
						d48 = snap104
						d49 = snap105
						d50 = snap106
						d51 = snap107
						d52 = snap108
						d53 = snap109
						d54 = snap110
						d59 = snap111
						d61 = snap112
						d62 = snap113
						d63 = snap114
						d64 = snap115
						d69 = snap116
						d71 = snap117
						d72 = snap118
						d73 = snap119
						d74 = snap120
						d76 = snap121
						d77 = snap122
						d78 = snap123
					}
					if !bbs[5].Rendered {
						return bbs[5].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.StabilizeDescForControlFlow(&d8)
					ctx.EnsureDesc(&d1)
					d126 = ctx.EmitSliceElementAddress(&d16, &d1, 16)
					ctx.EnsureDesc(&d126)
					r29 := ctx.AllocRegExcept(d126.Reg)
					ctx.EmitMovRegMem(r29, d126.Reg, 8)
					ctx.EmitMovRegMem(d126.Reg, d126.Reg, 0)
					d125 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d126.Reg, Reg2: r29}
					ctx.BindReg(d126.Reg, &d125)
					ctx.BindReg(r29, &d125)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d127 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d127 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d127)
					}
					if d127.Loc == LocReg && d1.Loc == LocReg && d127.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d127)
					d129 = ctx.EmitSliceElementAddress(&d16, &d127, 16)
					ctx.EnsureDesc(&d129)
					r30 := ctx.AllocRegExcept(d129.Reg)
					ctx.EmitMovRegMem(r30, d129.Reg, 8)
					ctx.EmitMovRegMem(d129.Reg, d129.Reg, 0)
					d128 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d129.Reg, Reg2: r30}
					ctx.BindReg(d129.Reg, &d128)
					ctx.BindReg(r30, &d128)
					ctx.FreeDesc(&d127)
					ctx.SyncDesc(&d125)
					ctx.EmitStoreScmerToStack(d125, int32(stackArray5)+int32(0))
					ctx.FreeDesc(&d125)
					ctx.SyncDesc(&d128)
					ctx.EmitStoreScmerToStack(d128, int32(stackArray5)+int32(16))
					ctx.FreeDesc(&d128)
					d130 = d12
					_ = d130
					d131 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d131
					callbackArgs133 := make([]JITValueDesc, 2)
					callbackArgs133[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray5) + 0}
					callbackArgs133[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray5) + 16}
					_ = d132
					callbackResultOff134 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff134))
					ctx.FreeDesc(&d131)
					ctx.StabilizeDescAcrossNestedCall(&d1)
					if d3.Loc == LocLambdaTemplate && d3.Lambda != nil {
						stableCallbackArgs135 := ctx.StabilizeCallbackArgs(callbackArgs133)
						ctx.ReclaimUntrackedRegs()
						outerRegs136 := ctx.PreserveOuterRegs()
						d132 = JITEmitProcInlineWithOuter(ctx, d3.Lambda.Proc, d3.Lambda.Outer, stableCallbackArgs135, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff134), ID: 0})
						ctx.RestoreOuterRegs(outerRegs136)
						ctx.ReclaimUntrackedRegs()
					} else {
						d137, knownBuiltin138 := jitEmitKnownDeclaration(ctx, d3, callbackArgs133, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff134), ID: 0})
						if knownBuiltin138 {
							d132 = d137
						} else {
							ctx.Coverage.DynamicCalls++
							d139 = jitCopyScmerToPair(ctx, d3)
							d132 = jitEmitDynamicCallableAt(ctx, d139, callbackArgs133, int32(stackArray5), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff134), ID: 0})
						}
					}
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d140 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d140 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d140)
					}
					if d140.Loc == LocReg && d1.Loc == LocReg && d140.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d140)
					d142 = ctx.EmitSliceElementAddress(&d16, &d140, 16)
					ctx.EnsureDesc(&d142)
					r31 := ctx.AllocRegExcept(d142.Reg)
					ctx.EmitMovRegMem(r31, d142.Reg, 8)
					ctx.EmitMovRegMem(d142.Reg, d142.Reg, 0)
					d141 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d142.Reg, Reg2: r31}
					ctx.BindReg(d142.Reg, &d141)
					ctx.BindReg(r31, &d141)
					ctx.FreeDesc(&d140)
					ctx.SyncDesc(&d130)
					d143 = jitMaterializeVirtualSlice(ctx, d130, JITValueDesc{Loc: LocAny})
					ctx.EmitStoreScmerToStack(d143, int32(stackArray10)+int32(0))
					ctx.FreeDesc(&d143)
					ctx.SyncDesc(&d132)
					ctx.EmitStoreScmerToStack(d132, int32(stackArray10)+int32(16))
					ctx.FreeDesc(&d132)
					ctx.SyncDesc(&d141)
					ctx.EmitStoreScmerToStack(d141, int32(stackArray10)+int32(32))
					ctx.FreeDesc(&d141)
					d144 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d144
					callbackArgs146 := make([]JITValueDesc, 3)
					callbackArgs146[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 0}
					callbackArgs146[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 16}
					callbackArgs146[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray10) + 32}
					_ = d145
					callbackResultOff147 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff147))
					ctx.FreeDesc(&d144)
					ctx.StabilizeDescAcrossNestedCall(&d1)
					if d8.Loc == LocLambdaTemplate && d8.Lambda != nil {
						stableCallbackArgs148 := ctx.StabilizeCallbackArgs(callbackArgs146)
						ctx.ReclaimUntrackedRegs()
						outerRegs149 := ctx.PreserveOuterRegs()
						d145 = JITEmitProcInlineWithOuter(ctx, d8.Lambda.Proc, d8.Lambda.Outer, stableCallbackArgs148, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff147), ID: 0})
						ctx.RestoreOuterRegs(outerRegs149)
						ctx.ReclaimUntrackedRegs()
					} else {
						d150, knownBuiltin151 := jitEmitKnownDeclaration(ctx, d8, callbackArgs146, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff147), ID: 0})
						if knownBuiltin151 {
							d145 = d150
						} else {
							ctx.Coverage.DynamicCalls++
							d152 = jitCopyScmerToPair(ctx, d8)
							d145 = jitEmitDynamicCallableAt(ctx, d152, callbackArgs146, int32(stackArray10), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff147), ID: 0})
						}
					}
					ctx.SyncDesc(&d145)
					ctx.FreeDesc(&d145)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					if d1.Loc == LocImm {
						d153 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 2)
						d153 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d153)
					}
					if d153.Loc == LocReg && d1.Loc == LocReg && d153.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d153)
					ctx.EmitStoreToStack(d153, int32(bbs[4].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d153)
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
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      98,
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
				declaration := declarations["mapkey_assoc_mut"]
				if !jitGeneratedEmitterInline(ctx, declaration, args) {
					ctx.Coverage.NativeCalls++
					return jitEmitGeneratedCallBoundary(ctx, declaration, sourceArgs, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var stackArray6 int32
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var stackArray11 int32
				var d12 JITValueDesc
				_ = d12
				var d13 JITValueDesc
				_ = d13
				var d15 JITValueDesc
				_ = d15
				var d16 JITValueDesc
				_ = d16
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var stackArray32 int32
				var d33 JITValueDesc
				_ = d33
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
				var phiBase48 int32
				_ = phiBase48
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
				var callbackResultOff67 int32
				var d70 JITValueDesc
				_ = d70
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var callbackResultOff77 int32
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
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d89 JITValueDesc
				_ = d89
				var d90 JITValueDesc
				_ = d90
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
				var d154 JITValueDesc
				_ = d154
				var callbackResultOff156 int32
				var d159 JITValueDesc
				_ = d159
				var d161 JITValueDesc
				_ = d161
				var d162 JITValueDesc
				_ = d162
				var d163 JITValueDesc
				_ = d163
				var d164 JITValueDesc
				_ = d164
				var d165 JITValueDesc
				_ = d165
				var d166 JITValueDesc
				_ = d166
				var d170 JITValueDesc
				_ = d170
				var d172 JITValueDesc
				_ = d172
				var d173 JITValueDesc
				_ = d173
				var d174 JITValueDesc
				_ = d174
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
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
				var bbs [6]BBDescriptor
				bbs[3].PhiBase = int32(phiBase0) + int32(0)
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d1 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
				ctx.PrepareScmerStackTarget(int32(phiBase0) + int32(0))
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					d3 = args[1]
					d3.ID = 0
					if d3.Loc == LocLambdaTemplate {
						d4 = d3
					} else if d3.Loc == LocImm {
						optimizedCallback5 := jitPrepareCallback(d3.Imm)
						ctx.TrackImm(optimizedCallback5)
						d4 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback5, Rooted: true}
					} else {
						if d3.Loc == LocInputPair && int(d3.StackOff) < ctx.InputArgCount {
							d4 = ctx.RequestOptimizedCallback(int(d3.StackOff))
						} else {
							d4 = jitCopyScmerToPair(ctx, d3)
						}
					}
					ctx.FreeDesc(&d3)
					ctx.EnsureDesc(&d4)
					ctx.StabilizeDescForControlFlow(&d4)
					stackArray6 = ctx.AllocStack(int32(32))
					_ = stackArray6
					globalLookup7 := Globalenv.Vars[Symbol("set_assoc_mut")]
					ctx.TrackImm(globalLookup7)
					d8 = JITValueDesc{Loc: LocImm, Type: globalLookup7.GetTag(), Imm: globalLookup7, Rooted: true}
					optimizedCallback10 := jitPrepareCallback(d8.Imm)
					ctx.TrackImm(optimizedCallback10)
					d9 = JITValueDesc{Loc: LocImm, Type: tagFunc, Imm: optimizedCallback10, Rooted: true}
					ctx.EnsureDesc(&d9)
					ctx.StabilizeDescForControlFlow(&d9)
					stackArray11 = ctx.AllocStack(int32(48))
					_ = stackArray11
					d12 = args[0]
					d12.ID = 0
					d12 = JITPrepareScmerGoArg(ctx, d12)
					d13 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("mapkey_assoc_mut")}
					ctx.EnsureDesc(&d13)
					if d13.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d13.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						ctx.TrackImm(d13.Imm)
						ptrWord, _ := d13.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, uint64(len(d13.Imm.String())))
						d13 = tmpPair
					} else if d13.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d13.Type, Reg: ctx.AllocRegExcept(d13.Reg), Reg2: ctx.AllocRegExcept(d13.Reg)}
						switch d13.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d13)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d13)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d13)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d13)
						d13 = tmpPair
					}
					if d13.Loc != LocRegPair && d13.Loc != LocStackPair && d13.Loc != LocInputPair {
						panic("jit: generic call arg expects 2-word value (asAssoc arg1)")
					}
					ctx.SyncDesc(&d12)
					ctx.SyncDesc(&d13)
					callResults14 := JITEmitGoCallResults(ctx, GoFuncAddr(asAssoc), []JITValueDesc{d12, d13}, []uint8{3, 1}, []uint8{1, 1})
					ctx.FreeDesc(&d13)
					d15 = callResults14[0]
					_ = d15
					d16 = callResults14[1]
					_ = d16
					ctx.FreeDesc(&d12)
					ctx.StabilizeDescForControlFlow(&d15)
					ctx.StabilizeDescForControlFlow(&d16)
					ctx.EnsureDesc(&d16)
					if d16.Loc == LocImm {
						d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.IsNil() == true)}
					} else {
						ctx.EnsureDesc(&d16)
						if d16.Loc != LocReg && d16.Loc != LocRegPair && d16.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d16.Reg)
						ctx.EmitCmpRegImm32(d16.Reg, 0)
						d17 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r0, Condition: CondEqual}
						ctx.BindReg(r0, &d17)
					}
					d18 = d17
					ctx.SyncDesc(&d18)
					if d18.Loc != LocImm && d18.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d18.Loc == LocImm {
						if d18.Imm.Bool() {
							return bbs[1].Render()
						}
						return bbs[2].Render()
					}
					ctx.EmitJump(d18.Condition, lbl2)
					if bbs[2].Rendered {
						ctx.EmitJmp(lbl3)
					}
					ctx.FreeDesc(&d17)
					ctx.FlushRegisterMoves()
					if !bbs[2].Rendered {
						snap19 := d1
						snap20 := d2
						snap21 := d3
						snap22 := d4
						snap23 := d8
						snap24 := d9
						snap25 := d12
						snap26 := d13
						snap27 := d15
						snap28 := d16
						snap29 := d17
						snap30 := d18
						alloc31 := ctx.SnapshotAllocState()
						bbs[2].Render()
						ctx.RestoreAllocState(alloc31)
						d1 = snap19
						d2 = snap20
						d3 = snap21
						d4 = snap22
						d8 = snap23
						d9 = snap24
						d12 = snap25
						d13 = snap26
						d15 = snap27
						d16 = snap28
						d17 = snap29
						d18 = snap30
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					stackArray32 = ctx.AllocStack(int32(0))
					_ = stackArray32
					d33 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(0), KnownSliceCap: int32(0), SliceSizeKnown: true}
					_ = d33
					callResults34 := JITEmitGoCallResults(ctx, GoFuncAddr(JITCloneScmerSlice), []JITValueDesc{d15}, []uint8{3}, []uint8{1})
					d35 = callResults34[0]
					ctx.StabilizeDescForControlFlow(&d35)
					d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d15)
					ctx.EnsureDesc(&d36)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm && d36.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d37.Imm.Int() - d36.Imm.Int())}
					} else {
						r1 := ctx.AllocReg()
						if d37.Loc == LocImm {
							ctx.EmitMovRegImm64(r1, uint64(d37.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r1, d37.Reg)
						}
						if d36.Loc == LocImm {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d36.Imm.Int()))
							ctx.EmitSubInt64(r1, ctx.ScratchReg)
						} else {
							ctx.EmitSubInt64(r1, d36.Reg)
						}
						d39 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r1}
						ctx.BindReg(r1, &d39)
					}
					r2 := ctx.EmitSliceDataAfterLow(&d15, &d36, 16)
					d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d40)
					ctx.BindReg(r2, &d40)
					var r3 Reg
					var r4 Reg
					ctx.SyncDesc(&d40)
					ctx.EnsureDesc(&d40)
					if d40.Loc == LocImm {
						r3 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r3, uint64(d40.Imm.Int()))
					} else {
						r3 = d40.Reg
					}
					ctx.ProtectReg(r3)
					ctx.SyncDesc(&d39)
					ctx.EnsureDesc(&d39)
					if d39.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d39.Imm.Int()))
					} else {
						r4 = d39.Reg
					}
					ctx.ProtectReg(r4)
					r5 := ctx.EmitSliceCapAfterLow(&d15, &d36, r3, r4)
					ctx.UnprotectReg(r4)
					ctx.UnprotectReg(r3)
					d41 = JITValueDesc{Loc: LocRegTriple, Reg: r3, Reg2: r4, Reg3: r5}
					ctx.BindReg(r3, &d41)
					ctx.BindReg(r4, &d41)
					ctx.BindReg(r5, &d41)
					ctx.BindReg(r3, &d41)
					ctx.BindReg(r4, &d41)
					ctx.BindReg(r5, &d41)
					d42 = ctx.EmitNewSliceFromGoSlice(&d41)
					ctx.StabilizeDescForControlFlow(&d42)
					ctx.SyncDesc(&d42)
					if d42.Loc == LocReg || d42.Loc == LocFPReg {
						ctx.ProtectReg(d42.Reg)
					} else if d42.Loc == LocRegPair {
						ctx.ProtectReg(d42.Reg)
						ctx.ProtectReg(d42.Reg2)
					}
					d43 = d42
					if d43.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d43)
					if d43.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d43, int32(bbs[3].PhiBase)+int32(0), 2)
					} else if d43.Loc == LocInputPair {
						ctx.EnsureDesc(&d43)
						ctx.EmitStoreScmerToStack(d43, int32(bbs[3].PhiBase)+int32(0))
					} else if d43.Loc == LocRegPair || d43.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d43, int32(bbs[3].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d43)
						ctx.EmitStoreToStack(d43, int32(bbs[3].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[3].PhiBase)+int32(0))+8)
					}
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(16))
					if d42.Loc == LocReg || d42.Loc == LocFPReg {
						ctx.UnprotectReg(d42.Reg)
					} else if d42.Loc == LocRegPair {
						ctx.UnprotectReg(d42.Reg)
						ctx.UnprotectReg(d42.Reg2)
					}
					return bbs[3].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.StabilizeDescForControlFlow(&d9)
					r6 := ctx.AllocReg()
					r7 := ctx.AllocRegExcept(r6)
					ctx.EmitMovRegImm64(r6, 0)
					ctx.EmitMovRegImm64(r7, 0)
					d44 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
					ctx.BindReg(r6, &d44)
					ctx.BindReg(r7, &d44)
					d45 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, Virtual: nil}
					ctx.SyncDesc(&d45)
					ctx.EnsureDesc(&d16)
					d46 = d16
					_ = d46
					ctx.StabilizeDescForControlFlow(&d46)
					inlineHome47 := d46
					inlineHome47.ID = 0
					phiBase48 = ctx.AllocStack(int32(16))
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase48) + int32(0)}
					_ = d49
					lbl7 := ctx.ReserveLabel()
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					lbl8 := ctx.ReserveLabel()
					_ = lbl8
					bbpos_1_1 := int32(-1)
					_ = bbpos_1_1
					lbl9 := ctx.ReserveLabel()
					_ = lbl9
					bbpos_1_2 := int32(-1)
					_ = bbpos_1_2
					lbl10 := ctx.ReserveLabel()
					_ = lbl10
					bbpos_1_3 := int32(-1)
					_ = bbpos_1_3
					lbl11 := ctx.ReserveLabel()
					_ = lbl11
					bbpos_1_4 := int32(-1)
					_ = bbpos_1_4
					lbl12 := ctx.ReserveLabel()
					_ = lbl12
					bbpos_1_5 := int32(-1)
					_ = bbpos_1_5
					lbl13 := ctx.ReserveLabel()
					_ = lbl13
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl8)
					ctx.ResolveFixups()
					d46 = inlineHome47
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase48) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(phiBase48)+int32(0))
					bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl9)
					ctx.ResolveFixups()
					d46 = inlineHome47
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase48) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d49)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d46)
					if d46.Loc == LocImm {
						fieldAddr := uintptr(d46.Imm.Int()) + 0
						r8 := ctx.AllocReg()
						r9 := ctx.AllocRegExcept(r8)
						r10 := ctx.AllocRegExcept(r8, r9)
						ctx.EmitMovRegMem64(r8, fieldAddr)
						ctx.EmitMovRegMem64(r9, fieldAddr+8)
						ctx.EmitMovRegMem64(r10, fieldAddr+16)
						d50 = JITValueDesc{Loc: LocRegTriple, Reg: r8, Reg2: r9, Reg3: r10}
						ctx.BindReg(r8, &d50)
						ctx.BindReg(r9, &d50)
						ctx.BindReg(r10, &d50)
					} else {
						off := int32(0)
						baseReg := d46.Reg
						r11 := ctx.AllocRegExcept(baseReg)
						r12 := ctx.AllocRegExcept(baseReg, r11)
						r13 := ctx.AllocRegExcept(baseReg, r11, r12)
						ctx.EmitMovRegMem(r11, baseReg, off)
						ctx.EmitMovRegMem(r12, baseReg, off+8)
						ctx.EmitMovRegMem(r13, baseReg, off+16)
						d50 = JITValueDesc{Loc: LocRegTriple, Reg: r11, Reg2: r12, Reg3: r13}
						ctx.BindReg(r11, &d50)
						ctx.BindReg(r12, &d50)
						ctx.BindReg(r13, &d50)
					}
					ctx.ReclaimUntrackedRegs()
					if d50.SliceSizeKnown {
						d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.KnownSliceLen))}
					} else if d50.Loc == LocImm {
						d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d50.StackOff))}
					} else if d50.Loc == LocStackTriple {
						d51 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d50.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d50)
						if d50.Loc == LocRegPair || d50.Loc == LocRegTriple {
							d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg2, ID: 0}
						} else if d50.Loc == LocReg {
							d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d50.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d51
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d51)
					ctx.EnsureDescsTogether(&d49, &d51)
					if d49.Loc == LocImm && d51.Loc == LocImm {
						d52 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d49.Imm.Int() < d51.Imm.Int())}
					} else if d51.Loc == LocImm {
						r14 := ctx.AllocRegExcept(d49.Reg)
						if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d49.Reg, int32(d51.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d51.Imm.Int()))
							ctx.EmitCmpInt64(d49.Reg, ctx.ScratchReg)
						}
						d52 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r14, Condition: CondSignedLess}
						ctx.BindReg(r14, &d52)
					} else if d49.Loc == LocImm {
						r15 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d49.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d51.Reg)
						d52 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r15, Condition: CondSignedLess}
						ctx.BindReg(r15, &d52)
					} else {
						r16 := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitCmpInt64(d49.Reg, d51.Reg)
						d52 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r16, Condition: CondSignedLess}
						ctx.BindReg(r16, &d52)
					}
					ctx.FreeDesc(&d51)
					ctx.ReclaimUntrackedRegs()
					d53 = d52
					ctx.SyncDesc(&d53)
					if d53.Loc != LocImm && d53.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					if d53.Loc == LocImm {
						if d53.Imm.Bool() {
							ctx.MarkLabel(lbl14)
							ctx.EmitJmp(lbl10)
						} else {
							ctx.MarkLabel(lbl15)
							ctx.EmitJmp(lbl11)
						}
					} else {
						ctx.EmitJump(d53.Condition, lbl14)
						ctx.EmitJmp(lbl15)
						ctx.FreeDesc(&d52)
						ctx.MarkLabel(lbl14)
						ctx.EmitJmp(lbl10)
						ctx.MarkLabel(lbl15)
						ctx.EmitJmp(lbl11)
					}
					bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl11)
					ctx.ResolveFixups()
					d46 = inlineHome47
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase48) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl10)
					ctx.ResolveFixups()
					d46 = inlineHome47
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase48) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d46)
					if d46.Loc == LocImm {
						fieldAddr := uintptr(d46.Imm.Int()) + 0
						r17 := ctx.AllocReg()
						r18 := ctx.AllocRegExcept(r17)
						r19 := ctx.AllocRegExcept(r17, r18)
						ctx.EmitMovRegMem64(r17, fieldAddr)
						ctx.EmitMovRegMem64(r18, fieldAddr+8)
						ctx.EmitMovRegMem64(r19, fieldAddr+16)
						d54 = JITValueDesc{Loc: LocRegTriple, Reg: r17, Reg2: r18, Reg3: r19}
						ctx.BindReg(r17, &d54)
						ctx.BindReg(r18, &d54)
						ctx.BindReg(r19, &d54)
					} else {
						off := int32(0)
						baseReg := d46.Reg
						r20 := ctx.AllocRegExcept(baseReg)
						r21 := ctx.AllocRegExcept(baseReg, r20)
						r22 := ctx.AllocRegExcept(baseReg, r20, r21)
						ctx.EmitMovRegMem(r20, baseReg, off)
						ctx.EmitMovRegMem(r21, baseReg, off+8)
						ctx.EmitMovRegMem(r22, baseReg, off+16)
						d54 = JITValueDesc{Loc: LocRegTriple, Reg: r20, Reg2: r21, Reg3: r22}
						ctx.BindReg(r20, &d54)
						ctx.BindReg(r21, &d54)
						ctx.BindReg(r22, &d54)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d49)
					ctx.ReclaimUntrackedRegs()
					d56 = ctx.EmitSliceElementAddress(&d54, &d49, 16)
					ctx.EnsureDesc(&d56)
					r23 := ctx.AllocRegExcept(d56.Reg)
					ctx.EmitMovRegMem(r23, d56.Reg, 8)
					ctx.EmitMovRegMem(d56.Reg, d56.Reg, 0)
					d55 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d56.Reg, Reg2: r23}
					ctx.BindReg(d56.Reg, &d55)
					ctx.BindReg(r23, &d55)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d46)
					if d46.Loc == LocImm {
						fieldAddr := uintptr(d46.Imm.Int()) + 0
						r24 := ctx.AllocReg()
						r25 := ctx.AllocRegExcept(r24)
						r26 := ctx.AllocRegExcept(r24, r25)
						ctx.EmitMovRegMem64(r24, fieldAddr)
						ctx.EmitMovRegMem64(r25, fieldAddr+8)
						ctx.EmitMovRegMem64(r26, fieldAddr+16)
						d57 = JITValueDesc{Loc: LocRegTriple, Reg: r24, Reg2: r25, Reg3: r26}
						ctx.BindReg(r24, &d57)
						ctx.BindReg(r25, &d57)
						ctx.BindReg(r26, &d57)
					} else {
						off := int32(0)
						baseReg := d46.Reg
						r27 := ctx.AllocRegExcept(baseReg)
						r28 := ctx.AllocRegExcept(baseReg, r27)
						r29 := ctx.AllocRegExcept(baseReg, r27, r28)
						ctx.EmitMovRegMem(r27, baseReg, off)
						ctx.EmitMovRegMem(r28, baseReg, off+8)
						ctx.EmitMovRegMem(r29, baseReg, off+16)
						d57 = JITValueDesc{Loc: LocRegTriple, Reg: r27, Reg2: r28, Reg3: r29}
						ctx.BindReg(r27, &d57)
						ctx.BindReg(r28, &d57)
						ctx.BindReg(r29, &d57)
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d49)
					if d49.Loc == LocImm {
						d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d49.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitMovRegReg(scratch, d49.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d58)
					}
					if d58.Loc == LocReg && d49.Loc == LocReg && d58.Reg == d49.Reg {
						ctx.TransferReg(d49.Reg)
						d49.Loc = LocNone
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d58)
					ctx.ReclaimUntrackedRegs()
					d60 = ctx.EmitSliceElementAddress(&d57, &d58, 16)
					ctx.EnsureDesc(&d60)
					r30 := ctx.AllocRegExcept(d60.Reg)
					ctx.EmitMovRegMem(r30, d60.Reg, 8)
					ctx.EmitMovRegMem(d60.Reg, d60.Reg, 0)
					d59 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d60.Reg, Reg2: r30}
					ctx.BindReg(d60.Reg, &d59)
					ctx.BindReg(r30, &d59)
					ctx.FreeDesc(&d58)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d55)
					ctx.EnsureDesc(&d59)
					d61 = d55
					_ = d61
					d62 = d59
					_ = d62
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					lbl16 := ctx.ReserveLabel()
					_ = lbl16
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl16)
					ctx.ResolveFixups()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d61)
					ctx.EmitStoreScmerToStack(d61, int32(stackArray6)+int32(0))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d62)
					ctx.EmitStoreScmerToStack(d62, int32(stackArray6)+int32(16))
					ctx.ReclaimUntrackedRegs()
					d63 = d45
					_ = d63
					ctx.ReclaimUntrackedRegs()
					d64 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d64
					ctx.ReclaimUntrackedRegs()
					callbackArgs66 := make([]JITValueDesc, 2)
					callbackArgs66[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray6) + 0}
					callbackArgs66[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray6) + 16}
					_ = d65
					callbackResultOff67 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff67))
					ctx.FreeDesc(&d64)
					if d4.Loc == LocLambdaTemplate && d4.Lambda != nil {
						stableCallbackArgs68 := ctx.StabilizeCallbackArgs(callbackArgs66)
						ctx.ReclaimUntrackedRegs()
						outerRegs69 := ctx.PreserveOuterRegs()
						d65 = JITEmitProcInlineWithOuter(ctx, d4.Lambda.Proc, d4.Lambda.Outer, stableCallbackArgs68, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff67), ID: 0})
						ctx.RestoreOuterRegs(outerRegs69)
						ctx.ReclaimUntrackedRegs()
					} else {
						d70, knownBuiltin71 := jitEmitKnownDeclaration(ctx, d4, callbackArgs66, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff67), ID: 0})
						if knownBuiltin71 {
							d65 = d70
						} else {
							ctx.Coverage.DynamicCalls++
							d72 = jitCopyScmerToPair(ctx, d4)
							d65 = jitEmitDynamicCallableAt(ctx, d72, callbackArgs66, int32(stackArray6), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff67), ID: 0})
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d63)
					d73 = jitMaterializeVirtualSlice(ctx, d63, JITValueDesc{Loc: LocAny})
					ctx.EmitStoreScmerToStack(d73, int32(stackArray11)+int32(0))
					ctx.FreeDesc(&d73)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d65)
					ctx.EmitStoreScmerToStack(d65, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d65)
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d62)
					ctx.EmitStoreScmerToStack(d62, int32(stackArray11)+int32(32))
					ctx.ReclaimUntrackedRegs()
					d74 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d74
					ctx.ReclaimUntrackedRegs()
					callbackArgs76 := make([]JITValueDesc, 3)
					callbackArgs76[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 0}
					callbackArgs76[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 16}
					callbackArgs76[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 32}
					_ = d75
					callbackResultOff77 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff77))
					ctx.FreeDesc(&d74)
					if d9.Loc == LocLambdaTemplate && d9.Lambda != nil {
						stableCallbackArgs78 := ctx.StabilizeCallbackArgs(callbackArgs76)
						ctx.ReclaimUntrackedRegs()
						outerRegs79 := ctx.PreserveOuterRegs()
						d75 = JITEmitProcInlineWithOuter(ctx, d9.Lambda.Proc, d9.Lambda.Outer, stableCallbackArgs78, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff77), ID: 0})
						ctx.RestoreOuterRegs(outerRegs79)
						ctx.ReclaimUntrackedRegs()
					} else {
						d80, knownBuiltin81 := jitEmitKnownDeclaration(ctx, d9, callbackArgs76, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff77), ID: 0})
						if knownBuiltin81 {
							d75 = d80
						} else {
							ctx.Coverage.DynamicCalls++
							d82 = jitCopyScmerToPair(ctx, d9)
							d75 = jitEmitDynamicCallableAt(ctx, d82, callbackArgs76, int32(stackArray11), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff77), ID: 0})
						}
					}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d75)
					ctx.FreeDesc(&d75)
					ctx.FreeDesc(&d75)
					ctx.ReclaimUntrackedRegs()
					d83 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
					ctx.FreeDesc(&d55)
					ctx.FreeDesc(&d59)
					ctx.ReclaimUntrackedRegs()
					d84 = d83
					ctx.EnsureDesc(&d84)
					if d84.Loc != LocImm && d84.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					if d84.Loc == LocImm {
						if d84.Imm.Bool() {
							ctx.MarkLabel(lbl17)
							ctx.EmitJmp(lbl13)
						} else {
							ctx.MarkLabel(lbl18)
							ctx.EmitJmp(lbl12)
						}
					} else {
						ctx.EmitCmpRegImm32(d84.Reg, 0)
						ctx.EmitJump(CondNotEqual, lbl17)
						ctx.EmitJmp(lbl18)
						ctx.MarkLabel(lbl17)
						ctx.EmitJmp(lbl13)
						ctx.MarkLabel(lbl18)
						ctx.EmitJmp(lbl12)
					}
					ctx.FreeDesc(&d83)
					bbpos_1_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl12)
					ctx.ResolveFixups()
					d46 = inlineHome47
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase48) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl7)
					bbpos_1_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.MarkLabel(lbl13)
					ctx.ResolveFixups()
					d46 = inlineHome47
					d49 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase48) + int32(0)}
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d49)
					ctx.EnsureDesc(&d49)
					if d49.Loc == LocImm {
						d85 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d49.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitMovRegReg(scratch, d49.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 2)
						d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d85)
					}
					if d85.Loc == LocReg && d49.Loc == LocReg && d85.Reg == d49.Reg {
						ctx.TransferReg(d49.Reg)
						d49.Loc = LocNone
					}
					ctx.EnsureDesc(&d85)
					ctx.EmitStoreToStack(d85, int32(phiBase48)+int32(0))
					ctx.StabilizeDescForControlFlow(&d85)
					inlineHome86 := d85
					inlineHome86.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl7)
					d87 = d45
					_ = d87
					ctx.SyncDesc(&d87)
					if d87.Loc == LocRegPair || d87.Loc == LocStackPair || d87.Loc == LocInputPair {
						ctx.EmitMovPairToResult(&d87, &result)
						result.Type = d87.Type
					} else {
						switch d87.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d87)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d87)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d87)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d87, &result)
							result.Type = d87.Type
						}
					}
					mergeReturnType(result.Type)
					ctx.EmitJmp(lbl0)
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.StabilizeDescForControlFlow(&d2)
					if d35.SliceSizeKnown {
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d35.KnownSliceLen))}
					} else if d35.Loc == LocImm {
						d88 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d35.StackOff))}
					} else if d35.Loc == LocStackTriple {
						d88 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d35.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d35)
						if d35.Loc == LocRegPair || d35.Loc == LocRegTriple {
							d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg2, ID: 0}
						} else if d35.Loc == LocReg {
							d88 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					_ = d88
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDescsTogether(&d2, &d88)
					if d2.Loc == LocImm && d88.Loc == LocImm {
						d89 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d88.Imm.Int())}
					} else if d88.Loc == LocImm {
						r31 := ctx.AllocRegExcept(d2.Reg)
						if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d88.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d88.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, ctx.ScratchReg)
						}
						d89 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r31, Condition: CondSignedLess}
						ctx.BindReg(r31, &d89)
					} else if d2.Loc == LocImm {
						r32 := ctx.AllocReg()
						ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(ctx.ScratchReg, d88.Reg)
						d89 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r32, Condition: CondSignedLess}
						ctx.BindReg(r32, &d89)
					} else {
						r33 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d88.Reg)
						d89 = JITValueDesc{Loc: LocFlags, Type: tagBool, Reg: r33, Condition: CondSignedLess}
						ctx.BindReg(r33, &d89)
					}
					ctx.FreeDesc(&d88)
					d90 = d89
					ctx.SyncDesc(&d90)
					if d90.Loc != LocImm && d90.Loc != LocFlags {
						panic("jit: fused If condition is neither LocImm nor LocFlags")
					}
					if d90.Loc == LocImm {
						if d90.Imm.Bool() {
							return bbs[4].Render()
						}
						return bbs[5].Render()
					}
					ctx.EmitJump(d90.Condition, lbl5)
					if bbs[5].Rendered {
						ctx.EmitJmp(lbl6)
					}
					ctx.FreeDesc(&d89)
					ctx.FlushRegisterMoves()
					if !bbs[5].Rendered {
						snap91 := d1
						snap92 := d2
						snap93 := d3
						snap94 := d4
						snap95 := d8
						snap96 := d9
						snap97 := d12
						snap98 := d13
						snap99 := d15
						snap100 := d16
						snap101 := d17
						snap102 := d18
						snap103 := d33
						snap104 := d35
						snap105 := d36
						snap106 := d37
						snap107 := d38
						snap108 := d39
						snap109 := d40
						snap110 := d41
						snap111 := d42
						snap112 := d43
						snap113 := d44
						snap114 := d45
						snap115 := d46
						snap116 := d49
						snap117 := d50
						snap118 := d51
						snap119 := d52
						snap120 := d53
						snap121 := d54
						snap122 := d55
						snap123 := d56
						snap124 := d57
						snap125 := d58
						snap126 := d59
						snap127 := d60
						snap128 := d61
						snap129 := d62
						snap130 := d63
						snap131 := d64
						snap132 := d65
						snap133 := d70
						snap134 := d72
						snap135 := d73
						snap136 := d74
						snap137 := d75
						snap138 := d80
						snap139 := d82
						snap140 := d83
						snap141 := d84
						snap142 := d85
						snap143 := d87
						snap144 := d88
						snap145 := d89
						snap146 := d90
						alloc147 := ctx.SnapshotAllocState()
						bbs[5].Render()
						ctx.RestoreAllocState(alloc147)
						d1 = snap91
						d2 = snap92
						d3 = snap93
						d4 = snap94
						d8 = snap95
						d9 = snap96
						d12 = snap97
						d13 = snap98
						d15 = snap99
						d16 = snap100
						d17 = snap101
						d18 = snap102
						d33 = snap103
						d35 = snap104
						d36 = snap105
						d37 = snap106
						d38 = snap107
						d39 = snap108
						d40 = snap109
						d41 = snap110
						d42 = snap111
						d43 = snap112
						d44 = snap113
						d45 = snap114
						d46 = snap115
						d49 = snap116
						d50 = snap117
						d51 = snap118
						d52 = snap119
						d53 = snap120
						d54 = snap121
						d55 = snap122
						d56 = snap123
						d57 = snap124
						d58 = snap125
						d59 = snap126
						d60 = snap127
						d61 = snap128
						d62 = snap129
						d63 = snap130
						d64 = snap131
						d65 = snap132
						d70 = snap133
						d72 = snap134
						d73 = snap135
						d74 = snap136
						d75 = snap137
						d80 = snap138
						d82 = snap139
						d83 = snap140
						d84 = snap141
						d85 = snap142
						d87 = snap143
						d88 = snap144
						d89 = snap145
						d90 = snap146
					}
					if !bbs[4].Rendered {
						return bbs[4].Render()
					}
					return result
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d35)
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.StabilizeDescForControlFlow(&d9)
					ctx.EnsureDesc(&d2)
					d149 = ctx.EmitSliceElementAddress(&d35, &d2, 16)
					ctx.EnsureDesc(&d149)
					r34 := ctx.AllocRegExcept(d149.Reg)
					ctx.EmitMovRegMem(r34, d149.Reg, 8)
					ctx.EmitMovRegMem(d149.Reg, d149.Reg, 0)
					d148 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d149.Reg, Reg2: r34}
					ctx.BindReg(d149.Reg, &d148)
					ctx.BindReg(r34, &d148)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d150 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d150 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d150)
					}
					if d150.Loc == LocReg && d2.Loc == LocReg && d150.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d150)
					d152 = ctx.EmitSliceElementAddress(&d35, &d150, 16)
					ctx.EnsureDesc(&d152)
					r35 := ctx.AllocRegExcept(d152.Reg)
					ctx.EmitMovRegMem(r35, d152.Reg, 8)
					ctx.EmitMovRegMem(d152.Reg, d152.Reg, 0)
					d151 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d152.Reg, Reg2: r35}
					ctx.BindReg(d152.Reg, &d151)
					ctx.BindReg(r35, &d151)
					ctx.FreeDesc(&d150)
					ctx.SyncDesc(&d148)
					ctx.EmitStoreScmerToStack(d148, int32(stackArray6)+int32(0))
					ctx.FreeDesc(&d148)
					ctx.SyncDesc(&d151)
					ctx.EmitStoreScmerToStack(d151, int32(stackArray6)+int32(16))
					ctx.FreeDesc(&d151)
					d153 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(2), KnownSliceCap: int32(2), SliceSizeKnown: true}
					_ = d153
					callbackArgs155 := make([]JITValueDesc, 2)
					callbackArgs155[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray6) + 0}
					callbackArgs155[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray6) + 16}
					_ = d154
					callbackResultOff156 = ctx.AllocStack(16)
					ctx.PrepareScmerStackTarget(int32(callbackResultOff156))
					ctx.FreeDesc(&d153)
					ctx.StabilizeDescAcrossNestedCall(&d2)
					if d4.Loc == LocLambdaTemplate && d4.Lambda != nil {
						stableCallbackArgs157 := ctx.StabilizeCallbackArgs(callbackArgs155)
						ctx.ReclaimUntrackedRegs()
						outerRegs158 := ctx.PreserveOuterRegs()
						d154 = JITEmitProcInlineWithOuter(ctx, d4.Lambda.Proc, d4.Lambda.Outer, stableCallbackArgs157, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff156), ID: 0})
						ctx.RestoreOuterRegs(outerRegs158)
						ctx.ReclaimUntrackedRegs()
					} else {
						d159, knownBuiltin160 := jitEmitKnownDeclaration(ctx, d4, callbackArgs155, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff156), ID: 0})
						if knownBuiltin160 {
							d154 = d159
						} else {
							ctx.Coverage.DynamicCalls++
							d161 = jitCopyScmerToPair(ctx, d4)
							d154 = jitEmitDynamicCallableAt(ctx, d161, callbackArgs155, int32(stackArray6), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(callbackResultOff156), ID: 0})
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d162 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 1)
						d162 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d162)
					}
					if d162.Loc == LocReg && d2.Loc == LocReg && d162.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d162)
					d164 = ctx.EmitSliceElementAddress(&d35, &d162, 16)
					ctx.EnsureDesc(&d164)
					r36 := ctx.AllocRegExcept(d164.Reg)
					ctx.EmitMovRegMem(r36, d164.Reg, 8)
					ctx.EmitMovRegMem(d164.Reg, d164.Reg, 0)
					d163 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d164.Reg, Reg2: r36}
					ctx.BindReg(d164.Reg, &d163)
					ctx.BindReg(r36, &d163)
					ctx.FreeDesc(&d162)
					ctx.SyncDesc(&d1)
					ctx.EmitStoreScmerToStack(d1, int32(stackArray11)+int32(0))
					ctx.SyncDesc(&d154)
					ctx.EmitStoreScmerToStack(d154, int32(stackArray11)+int32(16))
					ctx.FreeDesc(&d154)
					ctx.SyncDesc(&d163)
					ctx.EmitStoreScmerToStack(d163, int32(stackArray11)+int32(32))
					ctx.FreeDesc(&d163)
					d165 = JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(3), KnownSliceCap: int32(3), SliceSizeKnown: true}
					_ = d165
					callbackArgs167 := make([]JITValueDesc, 3)
					callbackArgs167[0] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 0}
					callbackArgs167[1] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 16}
					callbackArgs167[2] = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(stackArray11) + 32}
					_ = d166
					ctx.FreeDesc(&d165)
					ctx.StabilizeDescAcrossNestedCall(&d2)
					if d9.Loc == LocLambdaTemplate && d9.Lambda != nil {
						stableCallbackArgs168 := ctx.StabilizeCallbackArgs(callbackArgs167)
						ctx.ReclaimUntrackedRegs()
						outerRegs169 := ctx.PreserveOuterRegs()
						d166 = JITEmitProcInlineWithOuter(ctx, d9.Lambda.Proc, d9.Lambda.Outer, stableCallbackArgs168, ctx.SliceBase, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0})
						ctx.RestoreOuterRegs(outerRegs169)
						ctx.ReclaimUntrackedRegs()
					} else {
						d170, knownBuiltin171 := jitEmitKnownDeclaration(ctx, d9, callbackArgs167, JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0})
						if knownBuiltin171 {
							d166 = d170
						} else {
							ctx.Coverage.DynamicCalls++
							d172 = jitCopyScmerToPair(ctx, d9)
							d166 = jitEmitDynamicCallableAt(ctx, d172, callbackArgs167, int32(stackArray11), JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(bbs[3].PhiBase) + int32(0), ID: 0})
						}
					}
					ctx.StabilizeDescForControlFlow(&d166)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					if d2.Loc == LocImm {
						d173 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 2)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitIntBinaryImm(JITIntAdd, 64, scratch, 2)
						d173 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d173)
					}
					if d173.Loc == LocReg && d2.Loc == LocReg && d173.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d173)
					ctx.EmitStoreToStack(d173, int32(bbs[3].PhiBase)+int32(16))
					ctx.StabilizeDescForControlFlow(&d173)
					ctx.SyncDesc(&d166)
					if d166.Loc == LocReg || d166.Loc == LocFPReg {
						ctx.ProtectReg(d166.Reg)
					} else if d166.Loc == LocRegPair {
						ctx.ProtectReg(d166.Reg)
						ctx.ProtectReg(d166.Reg2)
					}
					d174 = d166
					if d174.Loc == LocNone {
						panic("jit: phi source has no location")
					}
					ctx.SyncDesc(&d174)
					if d174.Loc == LocStackPair {
						ctx.EmitCopyStackWords(d174, int32(bbs[3].PhiBase)+int32(0), 2)
					} else if d174.Loc == LocInputPair {
						ctx.EnsureDesc(&d174)
						ctx.EmitStoreScmerToStack(d174, int32(bbs[3].PhiBase)+int32(0))
					} else if d174.Loc == LocRegPair || d174.Loc == LocImm {
						ctx.EmitStoreScmerToStack(d174, int32(bbs[3].PhiBase)+int32(0))
					} else {
						ctx.EnsureDesc(&d174)
						ctx.EmitStoreToStack(d174, int32(bbs[3].PhiBase)+int32(0))
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[3].PhiBase)+int32(0))+8)
					}
					if d166.Loc == LocReg || d166.Loc == LocFPReg {
						ctx.UnprotectReg(d166.Reg)
					} else if d166.Loc == LocRegPair {
						ctx.UnprotectReg(d166.Reg)
						ctx.UnprotectReg(d166.Reg2)
					}
					return bbs[3].Render()
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
					d1 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					ctx.ReclaimUntrackedRegs()
					ctx.SyncDesc(&d1)
					if d1.Loc == LocRegPair || d1.Loc == LocStackPair || d1.Loc == LocInputPair {
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
			JITVirtualArgs:     true,
			JITInlineCallbacks: true,
			JITInlineCost:      102,
		},
	})
}
