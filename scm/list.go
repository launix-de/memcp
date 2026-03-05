/*
Copyright (C) 2023-2026  Carl-Philip Hänsch
Copyright (C) 2013  Pieter Kelchtermans (originally licensed unter WTFPL 2.0)

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
/*
 * A minimal Scheme interpreter, as seen in lis.py and SICP
 * http://norvig.com/lispy.html
 * http://mitpress.mit.edu/sicp/full-text/sicp/book/node77.html
 *
 * Pieter Kelchtermans 2013
 * LICENSE: WTFPL 2.0
 */
package scm

import "fmt"

// optimizeMap is the optimizer hook for `map`. It applies default optimization
// (including FirstParameterMutable swap to map_mut), then fuses
// (map (produceN N) fn) → (produceN N fn) to eliminate the intermediate list.
func optimizeMap(v []Scmer, oc *OptimizerContext, useResult bool) (Scmer, *TypeDescriptor) {
	// Run default optimization first (handles map → map_mut swap etc.)
	result, td := oc.applyDefaultOptimization(v, useResult, "map_mut")
	// Check if the optimized result is still a call to map/map_mut
	if result.IsSlice() {
		rv := result.Slice()
		if len(rv) == 3 {
			if sym, ok := scmerSymbol(rv[0]); ok && (sym == "map" || sym == "map_mut") {
				// Check if arg 1 is a (produceN N) call
				if rv[1].IsSlice() {
					inner := rv[1].Slice()
					if len(inner) == 2 {
						if isym, ok := scmerSymbol(inner[0]); ok && isym == "produceN" {
							// Fuse: (map (produceN N) fn) → (produceN N fn)
							return NewSlice([]Scmer{inner[0], inner[1], rv[2]}), td
						}
					}
				}
			}
		}
	}
	return result, td
}

func asSlice(v Scmer, ctx string) []Scmer {
	// Treat nil as empty list so higher-level code can be concise
	if v.IsNil() {
		return []Scmer{}
	}
	if v.IsSlice() {
		return v.Slice()
	}
	panic(fmt.Sprintf("%s expects a list, got %s", ctx, v.String()))
}

func asAssoc(v Scmer, ctx string) ([]Scmer, *FastDict) {
	// Treat nil as empty dictionary (assoc list)
	if v.IsNil() {
		return []Scmer{}, nil
	}
	if v.IsSlice() {
		return v.Slice(), nil
	}
	if v.IsFastDict() {
		return nil, v.FastDict()
	}
	panic(fmt.Sprintf("%s expects a dictionary", ctx))
}

func init_list() {
	// list functions
	DeclareTitle("Lists")

	// list is already in Globalenv.Vars (scm.go init); register it
	// in declarations so serialization can resolve the function pointer.
	Declare(&Globalenv, &Declaration{
		"list", "constructs a list from its arguments",
		0, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"items", "any", "items to put into the list", nil},
		}, "list",
		List,
		true, false, nil,
		nil,
	})

	Declare(&Globalenv, &Declaration{
		"count", "counts the number of elements in the list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", NoEscape},
		}, "int",
		func(a ...Scmer) Scmer {
			if a[0].GetTag() == tagSlice {
				return NewInt(int64(len(a[0].Slice())))
			}
			if a[0].GetTag() == tagFastDict {
				return NewInt(int64(len(a[0].FastDict().Pairs)))
			}
			panic("count expects a list")
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [5]BBDescriptor
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
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(6))}
			} else {
				r0 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d1.Reg, 6)
				ctx.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			d3 = d2
			ctx.EnsureDesc(&d3)
			if d3.Loc != LocImm && d3.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d3.Loc == LocImm {
				if d3.Imm.Bool() {
			ps4 := PhiState{General: ps.General}
			ps4.OverlayValues = make([]JITValueDesc, 4)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			ps4.OverlayValues[3] = d3
					return bbs[1].RenderPS(ps4)
				}
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 4)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
				return bbs[2].RenderPS(ps5)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl6 := ctx.ReserveLabel()
			lbl7 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			ctx.EmitJcc(CcNE, lbl6)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl6)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl7)
			ctx.EmitJmp(lbl3)
			ps6 := PhiState{General: true}
			ps6.OverlayValues = make([]JITValueDesc, 4)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps7 := PhiState{General: true}
			ps7.OverlayValues = make([]JITValueDesc, 4)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			snap8 := d0
			snap9 := d1
			snap10 := d2
			snap11 := d3
			alloc12 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
			}
			ctx.RestoreAllocState(alloc12)
			d0 = snap8
			d1 = snap9
			d2 = snap10
			d3 = snap11
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps6)
			}
			return result
			ctx.FreeDesc(&d2)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			ctx.ReclaimUntrackedRegs()
			d13 = args[0]
			d13.ID = 0
			var d14 JITValueDesc
			if d13.Loc == LocImm {
				slice := d13.Imm.Slice()
				d14 = JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: NewInt(int64(len(slice)))}
			} else {
				r1 := ctx.AllocReg()
				ctx.EmitMovRegReg(r1, d13.Reg2)
				ctx.EmitShrRegImm8(r1, 8)
				ctx.FreeReg(d13.Reg2)
				d14 = JITValueDesc{Loc: LocRegPair, Reg: d13.Reg, Reg2: r1}
				ctx.BindReg(d13.Reg, &d14)
				ctx.BindReg(r1, &d14)
			}
			ctx.FreeDesc(&d13)
			var d15 JITValueDesc
			if d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d14.StackOff))}
			} else {
				ctx.EnsureDesc(&d14)
				if d14.Loc == LocRegPair {
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg2}
					ctx.BindReg(d14.Reg2, &d15)
					ctx.BindReg(d14.Reg2, &d15)
				} else if d14.Loc == LocReg {
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg}
					ctx.BindReg(d14.Reg, &d15)
					ctx.BindReg(d14.Reg, &d15)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			ctx.EmitMakeInt(result, d15)
			if d15.Loc == LocReg { ctx.FreeReg(d15.Reg) }
			result.Type = tagInt
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			d17 = args[0]
			d17.ID = 0
			d18 = ctx.EmitGetTagDesc(&d17, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d17)
			ctx.EnsureDesc(&d18)
			var d19 JITValueDesc
			if d18.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d18.Imm.Int()) == uint64(15))}
			} else {
				r2 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d18.Reg, 15)
				ctx.EmitSetcc(r2, CcE)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d19)
			}
			ctx.FreeDesc(&d18)
			d20 = d19
			ctx.EnsureDesc(&d20)
			if d20.Loc != LocImm && d20.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d20.Loc == LocImm {
				if d20.Imm.Bool() {
			ps21 := PhiState{General: ps.General}
			ps21.OverlayValues = make([]JITValueDesc, 21)
			ps21.OverlayValues[0] = d0
			ps21.OverlayValues[1] = d1
			ps21.OverlayValues[2] = d2
			ps21.OverlayValues[3] = d3
			ps21.OverlayValues[13] = d13
			ps21.OverlayValues[14] = d14
			ps21.OverlayValues[15] = d15
			ps21.OverlayValues[16] = d16
			ps21.OverlayValues[17] = d17
			ps21.OverlayValues[18] = d18
			ps21.OverlayValues[19] = d19
			ps21.OverlayValues[20] = d20
					return bbs[3].RenderPS(ps21)
				}
			ps22 := PhiState{General: ps.General}
			ps22.OverlayValues = make([]JITValueDesc, 21)
			ps22.OverlayValues[0] = d0
			ps22.OverlayValues[1] = d1
			ps22.OverlayValues[2] = d2
			ps22.OverlayValues[3] = d3
			ps22.OverlayValues[13] = d13
			ps22.OverlayValues[14] = d14
			ps22.OverlayValues[15] = d15
			ps22.OverlayValues[16] = d16
			ps22.OverlayValues[17] = d17
			ps22.OverlayValues[18] = d18
			ps22.OverlayValues[19] = d19
			ps22.OverlayValues[20] = d20
				return bbs[4].RenderPS(ps22)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d20.Reg, 0)
			ctx.EmitJcc(CcNE, lbl8)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl8)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl5)
			ps23 := PhiState{General: true}
			ps23.OverlayValues = make([]JITValueDesc, 21)
			ps23.OverlayValues[0] = d0
			ps23.OverlayValues[1] = d1
			ps23.OverlayValues[2] = d2
			ps23.OverlayValues[3] = d3
			ps23.OverlayValues[13] = d13
			ps23.OverlayValues[14] = d14
			ps23.OverlayValues[15] = d15
			ps23.OverlayValues[16] = d16
			ps23.OverlayValues[17] = d17
			ps23.OverlayValues[18] = d18
			ps23.OverlayValues[19] = d19
			ps23.OverlayValues[20] = d20
			ps24 := PhiState{General: true}
			ps24.OverlayValues = make([]JITValueDesc, 21)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[3] = d3
			ps24.OverlayValues[13] = d13
			ps24.OverlayValues[14] = d14
			ps24.OverlayValues[15] = d15
			ps24.OverlayValues[16] = d16
			ps24.OverlayValues[17] = d17
			ps24.OverlayValues[18] = d18
			ps24.OverlayValues[19] = d19
			ps24.OverlayValues[20] = d20
			snap25 := d0
			snap26 := d1
			snap27 := d2
			snap28 := d3
			snap29 := d13
			snap30 := d14
			snap31 := d15
			snap32 := d16
			snap33 := d17
			snap34 := d18
			snap35 := d19
			snap36 := d20
			alloc37 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps24)
			}
			ctx.RestoreAllocState(alloc37)
			d0 = snap25
			d1 = snap26
			d2 = snap27
			d3 = snap28
			d13 = snap29
			d14 = snap30
			d15 = snap31
			d16 = snap32
			d17 = snap33
			d18 = snap34
			d19 = snap35
			d20 = snap36
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps23)
			}
			return result
			ctx.FreeDesc(&d19)
			return result
			}
			bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			ctx.ReclaimUntrackedRegs()
			d38 = args[0]
			d38.ID = 0
			var d39 JITValueDesc
			if d38.Loc == LocImm {
				panic("FastDict: LocImm not expected at JIT compile time")
			} else {
				ctx.FreeReg(d38.Reg2)
				d39 = JITValueDesc{Loc: LocReg, Reg: d38.Reg}
				ctx.BindReg(d38.Reg, &d39)
			}
			ctx.FreeDesc(&d38)
			var d40 JITValueDesc
			ctx.EnsureDesc(&d39)
			if d39.Loc == LocImm {
				fieldAddr := uintptr(d39.Imm.Int()) + 0
				r3 := ctx.AllocReg()
				r4 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r3, fieldAddr)
				ctx.EmitMovRegMem64(r4, fieldAddr+8)
				d40 = JITValueDesc{Loc: LocRegPair, Reg: r3, Reg2: r4}
				ctx.BindReg(r3, &d40)
				ctx.BindReg(r4, &d40)
			} else {
				off := int32(0)
				baseReg := d39.Reg
				r5 := ctx.AllocRegExcept(baseReg)
				r6 := ctx.AllocRegExcept(baseReg, r5)
				ctx.EmitMovRegMem(r5, baseReg, off)
				ctx.EmitMovRegMem(r6, baseReg, off+8)
				d40 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d40)
				ctx.BindReg(r6, &d40)
			}
			ctx.FreeDesc(&d39)
			var d41 JITValueDesc
			if d40.Loc == LocImm {
				d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d40.StackOff))}
			} else {
				ctx.EnsureDesc(&d40)
				if d40.Loc == LocRegPair {
					d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg2}
					ctx.BindReg(d40.Reg2, &d41)
					ctx.BindReg(d40.Reg2, &d41)
				} else if d40.Loc == LocReg {
					d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d40.Reg}
					ctx.BindReg(d40.Reg, &d41)
					ctx.BindReg(d40.Reg, &d41)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d41)
			ctx.EmitMakeInt(result, d41)
			if d41.Loc == LocReg { ctx.FreeReg(d41.Reg) }
			result.Type = tagInt
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
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
			ctx.ReclaimUntrackedRegs()
			d43 = JITValueDesc{Loc: LocImm, Type: tagString, Imm: NewString("count expects a list")}
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d43)
			if d43.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d43.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d43)
				} else if d43.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d43)
				} else if d43.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d43)
				} else if d43.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d43.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d43 = tmpPair
			} else if d43.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d43.Type, Reg: ctx.AllocRegExcept(d43.Reg), Reg2: ctx.AllocRegExcept(d43.Reg)}
				switch d43.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d43)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d43)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d43)
				default:
					panic("jit: panic arg scalar type unknown for Scmer pair")
				}
				ctx.FreeDesc(&d43)
				d43 = tmpPair
			}
			if d43.Loc != LocRegPair && d43.Loc != LocStackPair {
				panic("jit: panic arg expects Scmer pair")
			}
			ctx.EmitGoCallVoid(GoFuncAddr(jitPanic), []JITValueDesc{d43})
			ctx.FreeDesc(&d43)
			return result
			}
			argPinned44 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned44 = append(argPinned44, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned44 = append(argPinned44, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned44 = append(argPinned44, ai.Reg2)
					}
				}
			}
			ps45 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps45)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			for _, r := range argPinned44 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */ /* TODO: unsupported compare const kind: nil:*github.com/launix-de/memcp/scm.FastDict */
	})
	Declare(&Globalenv, &Declaration{
		"nth", "get the nth item of a list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", NoEscape},
			DeclarationParameter{"index", "number", "index beginning from 0", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "nth")
			idx := int(a[1].Int())
			if idx < 0 || idx >= len(list) {
				panic("nth index out of range")
			}
			return list[idx]
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"slice", "extract a sublist from start (inclusive) to end (exclusive).\n(slice list start end) returns elements list[start..end).",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", NoEscape},
			DeclarationParameter{"start", "number", "start index (inclusive)", nil},
			DeclarationParameter{"end", "number", "end index (exclusive)", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "slice")
			start := int(a[1].Int())
			end := int(a[2].Int())
			if start < 0 {
				start = 0
			}
			if end > len(list) {
				end = len(list)
			}
			if start >= end {
				return NewSlice([]Scmer{})
			}
			result := make([]Scmer, end-start)
			copy(result, list[start:end])
			return NewSlice(result)
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"append", "appends items to a list and return the extended list.\nThe original list stays unharmed.",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", nil},
			DeclarationParameter{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			base := append([]Scmer{}, asSlice(a[0], "append")...)
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("append_mut")},
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
	Declare(&Globalenv, &Declaration{
		"append_unique", "appends items to a list but only if they are new.\nThe original list stays unharmed.",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "base list", nil},
			DeclarationParameter{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := append([]Scmer{}, asSlice(a[0], "append_unique")...)
			for _, el := range a[1:] {
				for _, el2 := range list {
					if Equal(el, el2) {
						// ignore duplicates
						goto skipItem
					}
				}
				list = append(list, el)
			skipItem:
			}
			return NewSlice(list)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("append_unique_mut")},
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
	Declare(&Globalenv, &Declaration{
		"cons", "constructs a list from a head and a tail list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"car", "any", "new head element", nil},
			DeclarationParameter{"cdr", "list", "tail that is appended after car", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
			car := a[0]
			if a[1].GetTag() == tagSlice {
				return NewSlice(append([]Scmer{car}, a[1].Slice()...))
			}
			return NewSlice([]Scmer{car, a[1]})
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"car", "extracts the head of a list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list", NoEscape},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "car")
			if len(list) == 0 {
				panic("car on empty list")
			}
			return list[0]
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"cdr", "extracts the tail of a list\nThe tail of a list is a list with all items except the head.",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list", NoEscape},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cdr")
			if len(list) == 0 {
				return NewSlice([]Scmer{})
			}
			return NewSlice(list[1:])
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"cadr", "extracts the second element of a list.\nEquivalent to (car (cdr x)).",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list", NoEscape},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "cadr")
			if len(list) < 2 {
				panic("cadr on list with fewer than 2 elements")
			}
			return list[1]
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"zip", "swaps the dimension of a list of lists. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as the components that will be zipped into the sub list",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "any", "list of lists of items", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "zip")
			}
			if len(lists) == 0 {
				return NewSlice([]Scmer{})
			}
			first := asSlice(lists[0], "zip element")
			size := len(first)
			result := make([]Scmer, size)
			for i := 0; i < size; i++ {
				subresult := make([]Scmer, len(lists))
				for j, v := range lists {
					current := asSlice(v, "zip item")
					if i >= len(current) {
						panic("zip expects lists of equal length")
					}
					subresult[j] = current[i]
				}
				result[i] = NewSlice(subresult)
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
	})
	Declare(&Globalenv, &Declaration{
		"merge", "flattens a list of lists into a list containing all the subitems. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as lists that will be merged into one",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "any", "list of lists of items", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "merge")
			}
			size := 0
			for _, v := range lists {
				size += len(asSlice(v, "merge item"))
			}
			result := make([]Scmer, 0, size)
			for _, v := range lists {
				result = append(result, asSlice(v, "merge item")...)
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
	})
	Declare(&Globalenv, &Declaration{
		"merge_unique", "flattens a list of lists into a list containing all the subitems. Duplicates are filtered out.",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list of lists of items", NoEscape},
		}, "list",
		func(a ...Scmer) Scmer {
			lists := a
			if len(a) == 1 {
				lists = asSlice(a[0], "merge_unique")
			}
			size := 0
			for _, v := range lists {
				size += len(asSlice(v, "merge_unique item"))
			}
			result := make([]Scmer, 0, size)
			for _, v := range lists {
				for _, el := range asSlice(v, "merge_unique item") {
					duplicate := false
					for _, existing := range result {
						if Equal(el, existing) {
							duplicate = true
							break
						}
					}
					if !duplicate {
						result = append(result, el)
					}
				}
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
	})
	Declare(&Globalenv, &Declaration{
		"has?", "checks if a list has a certain item (equal?)",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"haystack", "list", "list to search in", NoEscape},
			DeclarationParameter{"needle", "any", "item to search for", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "has?")
			for _, v := range list {
				if Equal(a[1], v) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"filter", "returns a list that only contains elements that pass the filter function",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be filtered", NoEscape},
			DeclarationParameter{"condition", "func", "filter condition func(any)->bool", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			input := asSlice(a[0], "filter")
			result := make([]Scmer, 0, len(input))
			fn := OptimizeProcToSerialFunction(a[1])
			for _, v := range input {
				if fn(v).Bool() {
					result = append(result, v)
				}
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("filter_mut")},
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"map", "returns a list that contains the results of a map function that is applied to the list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be mapped", NoEscape},
			DeclarationParameter{"map", "func", "map function func(any)->any that is applied to each item", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "map")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(v)
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: optimizeMap},
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"mapIndex", "returns a list that contains the results of a map function that is applied to the list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be mapped", NoEscape},
			DeclarationParameter{"map", "func", "map function func(i, any)->any that is applied to each item", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "mapIndex")
			result := make([]Scmer, len(list))
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				result[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("mapIndex_mut")},
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"reduce", "returns a list that contains the result of a map function",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list that has to be reduced", NoEscape},
			DeclarationParameter{"reduce", "func", "reduce function func(any any)->any where the first parameter is the accumulator, the second is a list item", &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Transfer: true}, nil}}},
			DeclarationParameter{"neutral", "any", "(optional) initial value of the accumulator, defaults to nil", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "reduce")
			fn := OptimizeProcToSerialFunction(a[1])
			result := NewNil()
			i := 0
			if len(a) > 2 {
				result = a[2]
			} else if len(list) > 0 {
				result = list[0]
				i = 1
			}
			for i < len(list) {
				result = fn(result, list[i])
				i++
			}
			return result
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"produce", "returns a list that contains produced items - it works like for(state = startstate, condition(state), state = iterator(state)) {yield state}",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"startstate", "any", "start state to begin with", nil},
			DeclarationParameter{"condition", "func", "func that returns true whether the state will be inserted into the result or the loop is stopped", nil},
			DeclarationParameter{"iterator", "func", "func that produces the next state", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			result := make([]Scmer, 0)
			state := a[0]
			condition := OptimizeProcToSerialFunction(a[1])
			iterator := OptimizeProcToSerialFunction(a[2])
			for condition(state).Bool() {
				result = append(result, state)
				state = iterator(state)
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: Slice on non-desc: slice t0[:0:int] */, /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"produceN", "returns a list with numbers from 0..n-1, optionally mapped through a function",
		1, 2,
		[]DeclarationParameter{
			DeclarationParameter{"n", "number", "number of elements to produce", nil},
			DeclarationParameter{"fn", "func", "(optional) map function applied to each index", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			n := int(a[0].Int())
			if n < 0 {
				n = 0
			}
			result := make([]Scmer, n)
			if len(a) > 1 && !a[1].IsNil() {
				// fused produceN+map: generate and transform in one pass
				fn := OptimizeProcToSerialFunction(a[1])
				for i := 0; i < n; i++ {
					result[i] = fn(NewInt(int64(i)))
				}
			} else {
				for i := 0; i < n; i++ {
					result[i] = NewInt(int64(i))
				}
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: MakeSlice: make []Scmer t5 t5 */, /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */
	})
	Declare(&Globalenv, &Declaration{
		"list?", "checks if a value is a list",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value to check", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].IsSlice())
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d0 JITValueDesc
			_ = d0
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			d2.ID = 0
			d1 = ctx.EmitTagEqualsBorrowed(&d2, tagSlice, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d1.Loc == LocImm {
				ctx.EmitMakeBool(result, d1)
			} else {
				ctx.EmitMakeBool(result, d1)
				ctx.FreeReg(d1.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			argPinned3 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned3 = append(argPinned3, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned3 = append(argPinned3, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned3 = append(argPinned3, ai.Reg2)
					}
				}
			}
			ps4 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps4)
			for _, r := range argPinned3 {
				ctx.UnprotectReg(r)
			}
			return result
		}, /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */ /* TODO: unresolved SSA value: false:bool */
	})
	Declare(&Globalenv, &Declaration{
		"contains?", "checks if a value is in a list; uses the equal?? operator",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"list", "list", "list to check", NoEscape},
			DeclarationParameter{"value", "any", "value to check", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			arr := asSlice(a[0], "contains?")
			for _, v := range arr {
				if Equal(v, a[1]) {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	// dictionary functions
	DeclareTitle("Associative Lists / Dictionaries")

	Declare(&Globalenv, &Declaration{
		"filter_assoc", "returns a filtered dictionary according to a filter function",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be filtered", NoEscape},
			DeclarationParameter{"condition", "func", "filter function func(string any)->bool where the first parameter is the key, the second is the value", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			result := make([]Scmer, 0)
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "filter_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						result = append(result, slice[i], slice[i+1])
					}
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = append(result, k, v)
					}
					return true
				})
			}
			return NewSlice(result)
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("filter_assoc_mut")},
		nil /* TODO: Slice on non-desc: slice t1[:0:int] */, /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */
	})
	Declare(&Globalenv, &Declaration{
		"map_assoc", "returns a mapped dictionary according to a map function\nKeys will stay the same but values are mapped.",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be mapped", NoEscape},
			DeclarationParameter{"map", "func", "map function func(string any)->any where the first parameter is the key, the second is the value. It must return the new value.", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "map_assoc"); fd == nil {
				result := make([]Scmer, len(slice))
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						result[i] = v
						key = v
					} else {
						result[i] = fn(key, v)
					}
				}
				return NewSlice(result)
			} else {
				result := make([]Scmer, 0, len(fd.Pairs))
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, k, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("map_assoc_mut")},
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"reduce_assoc", "reduces a dictionary according to a reduce function",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be reduced", NoEscape},
			DeclarationParameter{"reduce", "func", "reduce function func(any string any)->any where the first parameter is the accumulator, second is key, third is value. It must return the new accumulator.", &TypeDescriptor{Kind: "func", Params: []*TypeDescriptor{{Transfer: true}, nil, nil}}},
			DeclarationParameter{"neutral", "any", "initial value for the accumulator", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			result := a[2]
			reduce := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "reduce_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					result = reduce(result, slice[i], slice[i+1])
				}
			} else {
				fd.Iterate(func(k, v Scmer) bool { result = reduce(result, k, v); return true })
			}
			return result
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"has_assoc?", "checks if a dictionary has a key present",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be checked", NoEscape},
			DeclarationParameter{"key", "string", "key to test", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			if slice, fd := asAssoc(a[0], "has_assoc?"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if Equal(slice[i], a[1]) {
						return NewBool(true)
					}
				}
			} else {
				if _, ok := fd.Get(a[1]); ok {
					return NewBool(true)
				}
			}
			return NewBool(false)
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"get_assoc", "gets a value from a dictionary by key, returns nil if not found",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary to look up", NoEscape},
			DeclarationParameter{"key", "any", "key to look up", nil},
			DeclarationParameter{"default", "any", "optional default value if key not found", nil},
		}, "any",
		func(a ...Scmer) Scmer {
			if slice, fd := asAssoc(a[0], "get_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if Equal(slice[i], a[1]) {
						return slice[i+1]
					}
				}
			} else {
				if v, ok := fd.Get(a[1]); ok {
					return v
				}
			}
			// Return default value if provided, otherwise nil
			if len(a) >= 3 {
				return a[2]
			}
			return NewNil()
		},
		true, false, nil,
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"extract_assoc", "applies a function (key value) on the dictionary and returns the results as a flat list",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "dictionary that has to be checked", NoEscape},
			DeclarationParameter{"map", "func", "func(string any)->any that flattens down each element", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "extract_assoc"); fd == nil {
				result := make([]Scmer, len(slice)/2)
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						key = v
					} else {
						result[i/2] = fn(key, v)
					}
				}
				return NewSlice(result)
			} else {
				result := make([]Scmer, 0, len(fd.Pairs)/2)
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("extract_assoc_mut")},
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"set_assoc", "returns a new dictionary where a single value has been changed.\nThe original dictionary is not modified.",
		3, 4,
		[]DeclarationParameter{
			DeclarationParameter{"dict", "list", "input dictionary", nil},
			DeclarationParameter{"key", "string", "key that has to be set", nil},
			DeclarationParameter{"value", "any", "new value to set", nil},
			DeclarationParameter{"merge", "func", "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value. It must return the merged value that shall be physically stored in the new dictionary.", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			var mergeFn func(Scmer, Scmer) Scmer
			if len(a) > 3 {
				mfn := OptimizeProcToSerialFunction(a[3])
				mergeFn = func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
			}
			slice, fd := asAssoc(a[0], "set_assoc")
			if fd == nil {
				// defensive copy — set_assoc must not mutate the original
				list := append([]Scmer{}, slice...)
				for i := 0; i < len(list); i += 2 {
					if Equal(list[i], a[1]) {
						if mergeFn != nil {
							list[i+1] = mergeFn(list[i+1], a[2])
						} else {
							list[i+1] = a[2]
						}
						return NewSlice(list)
					}
				}
				list = append(list, a[1], a[2])
				if len(list) >= 10 {
					fd := NewFastDictValue(len(list)/2 + 4)
					for i := 0; i < len(list); i += 2 {
						fd.Set(list[i], list[i+1], nil)
					}
					return NewFastDict(fd)
				}
				return NewSlice(list)
			} else {
				fd = fd.Copy()
				fd.Set(a[1], a[2], mergeFn)
				return NewFastDict(fd)
			}
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("set_assoc_mut")},
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})
	Declare(&Globalenv, &Declaration{
		"merge_assoc", "returns a dictionary where all keys from dict1 and all keys from dict2 are present.\nIf a key is present in both inputs, the second one will be dominant so the first value will be overwritten unless you provide a merge function",
		2, 3,
		[]DeclarationParameter{
			DeclarationParameter{"dict1", "list", "first input dictionary that has to be changed. You must not use this value again.", nil},
			DeclarationParameter{"dict2", "list", "input dictionary that contains the new values that have to be added", nil},
			DeclarationParameter{"merge", "func", "(optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value from dict2. It must return the merged value that shall be pysically stored in the new dictionary.", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc"])
			dst := a[0]
			if slice, fd := asAssoc(a[1], "merge_assoc"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if len(a) > 2 {
						dst = setAssoc(dst, slice[i], slice[i+1], a[2])
					} else {
						dst = setAssoc(dst, slice[i], slice[i+1])
					}
				}
			} else {
				if len(a) > 2 {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v, a[2]); return true })
				} else {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v); return true })
				}
			}
			return dst
		},
		true, false, &TypeDescriptor{Return: FreshAlloc, Optimize: FirstParameterMutable("merge_assoc_mut")},
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
	})

	// _mut variants: optimizer-only, forbidden from .scm code
	// Tier 1: same-length, zero-copy

	Declare(&Globalenv, &Declaration{
		"map_mut", "in-place map (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"list", "list", "owned list to map in-place", nil},
			{"map", "func", "map function", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(v)
			}
			return NewSlice(list)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
	})

	Declare(&Globalenv, &Declaration{
		"mapIndex_mut", "in-place mapIndex (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"list", "list", "owned list to map in-place", nil},
			{"map", "func", "map function func(i, any)->any", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			for i, v := range list {
				list[i] = fn(NewInt(int64(i)), v)
			}
			return NewSlice(list)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
	})

	Declare(&Globalenv, &Declaration{
		"map_assoc_mut", "in-place map_assoc (optimizer-only, slice path only)",
		2, 2,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to map in-place", nil},
			{"map", "func", "map function func(key, value)->value", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "map_assoc_mut"); fd == nil {
				var key Scmer
				for i, v := range slice {
					if i%2 == 0 {
						key = v
					} else {
						slice[i] = fn(key, v)
					}
				}
				return NewSlice(slice)
			} else {
				// FastDict path: cannot mutate in-place, fall back to allocating
				result := make([]Scmer, 0, len(fd.Pairs))
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, k, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	// Tier 2: shrinking, write-cursor

	Declare(&Globalenv, &Declaration{
		"filter_mut", "in-place filter (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"list", "list", "owned list to filter in-place", nil},
			{"condition", "func", "filter condition func(any)->bool", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			input := a[0].Slice()
			fn := OptimizeProcToSerialFunction(a[1])
			w := 0
			for _, v := range input {
				if fn(v).Bool() {
					input[w] = v
					w++
				}
			}
			return NewSlice(input[:w])
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
	})

	Declare(&Globalenv, &Declaration{
		"filter_assoc_mut", "in-place filter_assoc (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to filter in-place", nil},
			{"condition", "func", "filter function func(key, value)->bool", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "filter_assoc_mut"); fd == nil {
				w := 0
				for i := 0; i < len(slice); i += 2 {
					if fn(slice[i], slice[i+1]).Bool() {
						slice[w] = slice[i]
						slice[w+1] = slice[i+1]
						w += 2
					}
				}
				return NewSlice(slice[:w])
			} else {
				result := make([]Scmer, 0)
				fd.Iterate(func(k, v Scmer) bool {
					if fn(k, v).Bool() {
						result = append(result, k, v)
					}
					return true
				})
				return NewSlice(result)
			}
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"extract_assoc_mut", "in-place extract_assoc (optimizer-only)",
		2, 2,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to extract from in-place", nil},
			{"map", "func", "func(key, value)->any that extracts each element", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			fn := OptimizeProcToSerialFunction(a[1])
			if slice, fd := asAssoc(a[0], "extract_assoc_mut"); fd == nil {
				w := 0
				for i := 0; i < len(slice); i += 2 {
					slice[w] = fn(slice[i], slice[i+1])
					w++
				}
				return NewSlice(slice[:w])
			} else {
				result := make([]Scmer, 0, len(fd.Pairs)/2)
				fd.Iterate(func(k, v Scmer) bool {
					result = append(result, fn(k, v))
					return true
				})
				return NewSlice(result)
			}
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"set_assoc_mut", "in-place set_assoc (optimizer-only, mutates input directly)",
		3, 4,
		[]DeclarationParameter{
			{"dict", "list", "owned dictionary to mutate", nil},
			{"key", "string", "key to set", nil},
			{"value", "any", "new value", nil},
			{"merge", "func", "(optional) merge function", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			var mergeFn func(Scmer, Scmer) Scmer
			if len(a) > 3 {
				mfn := OptimizeProcToSerialFunction(a[3])
				mergeFn = func(oldV, newV Scmer) Scmer { return mfn(oldV, newV) }
			}
			// Always operate on FastDict; promote slice/nil inputs
			var fd *FastDict
			if a[0].IsFastDict() {
				fd = a[0].FastDict()
			} else if a[0].IsSlice() {
				list := a[0].Slice()
				fd = NewFastDictValue(len(list)/2 + 4)
				for i := 0; i+1 < len(list); i += 2 {
					fd.Set(list[i], list[i+1], nil)
				}
			} else {
				fd = NewFastDictValue(8)
			}
			fd.Set(a[1], a[2], mergeFn)
			return NewFastDict(fd)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			var d5 JITValueDesc
			_ = d5
			var d8 JITValueDesc
			_ = d8
			var d11 JITValueDesc
			_ = d11
			var d21 JITValueDesc
			_ = d21
			var d22 JITValueDesc
			_ = d22
			var d23 JITValueDesc
			_ = d23
			var d24 JITValueDesc
			_ = d24
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
			var d31 JITValueDesc
			_ = d31
			var d34 JITValueDesc
			_ = d34
			var d57 JITValueDesc
			_ = d57
			var d58 JITValueDesc
			_ = d58
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
			var d113 JITValueDesc
			_ = d113
			var d114 JITValueDesc
			_ = d114
			var d115 JITValueDesc
			_ = d115
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
			var d125 JITValueDesc
			_ = d125
			var d127 JITValueDesc
			_ = d127
			var d128 JITValueDesc
			_ = d128
			var d129 JITValueDesc
			_ = d129
			var d132 JITValueDesc
			_ = d132
			var d186 JITValueDesc
			_ = d186
			var d187 JITValueDesc
			_ = d187
			var d188 JITValueDesc
			_ = d188
			var d189 JITValueDesc
			_ = d189
			var d190 JITValueDesc
			_ = d190
			var d191 JITValueDesc
			_ = d191
			var d193 JITValueDesc
			_ = d193
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(48)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(48)
			}
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			var bbs [10]BBDescriptor
			bbs[2].PhiBase = int32(0)
			bbs[2].PhiCount = uint16(1)
			bbs[4].PhiBase = int32(16)
			bbs[4].PhiCount = uint16(1)
			bbs[8].PhiBase = int32(32)
			bbs[8].PhiCount = uint16(1)
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
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.ReserveLabel()
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			lbl9 := ctx.ReserveLabel()
			bbpos_0_9 := int32(-1)
			_ = bbpos_0_9
			lbl10 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			ctx.ReclaimUntrackedRegs()
			d3 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d3)
			var d4 JITValueDesc
			if d3.Loc == LocImm {
				d4 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() > 3)}
			} else {
				r1 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d3.Reg, 3)
				ctx.EmitSetcc(r1, CcG)
				d4 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d4)
			}
			ctx.FreeDesc(&d3)
			d5 = d4
			ctx.EnsureDesc(&d5)
			if d5.Loc != LocImm && d5.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d5.Loc == LocImm {
				if d5.Imm.Bool() {
			ps6 := PhiState{General: ps.General}
			ps6.OverlayValues = make([]JITValueDesc, 6)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps6.OverlayValues[4] = d4
			ps6.OverlayValues[5] = d5
					return bbs[1].RenderPS(ps6)
				}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
			ps7 := PhiState{General: ps.General}
			ps7.OverlayValues = make([]JITValueDesc, 6)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			ps7.OverlayValues[4] = d4
			ps7.OverlayValues[5] = d5
			ps7.PhiValues = make([]JITValueDesc, 1)
			d8 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			ps7.PhiValues[0] = d8
				return bbs[2].RenderPS(ps7)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl11 := ctx.ReserveLabel()
			lbl12 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d5.Reg, 0)
			ctx.EmitJcc(CcNE, lbl11)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl11)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl12)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[2].PhiBase)+int32(0))
			ctx.EmitJmp(lbl3)
			ps9 := PhiState{General: true}
			ps9.OverlayValues = make([]JITValueDesc, 9)
			ps9.OverlayValues[0] = d0
			ps9.OverlayValues[1] = d1
			ps9.OverlayValues[2] = d2
			ps9.OverlayValues[3] = d3
			ps9.OverlayValues[4] = d4
			ps9.OverlayValues[5] = d5
			ps9.OverlayValues[8] = d8
			ps10 := PhiState{General: true}
			ps10.OverlayValues = make([]JITValueDesc, 9)
			ps10.OverlayValues[0] = d0
			ps10.OverlayValues[1] = d1
			ps10.OverlayValues[2] = d2
			ps10.OverlayValues[3] = d3
			ps10.OverlayValues[4] = d4
			ps10.OverlayValues[5] = d5
			ps10.OverlayValues[8] = d8
			ps10.PhiValues = make([]JITValueDesc, 1)
			d11 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			ps10.PhiValues[0] = d11
			snap12 := d0
			snap13 := d1
			snap14 := d2
			snap15 := d3
			snap16 := d4
			snap17 := d5
			snap18 := d8
			snap19 := d11
			alloc20 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps10)
			}
			ctx.RestoreAllocState(alloc20)
			d0 = snap12
			d1 = snap13
			d2 = snap14
			d3 = snap15
			d4 = snap16
			d5 = snap17
			d8 = snap18
			d11 = snap19
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps9)
			}
			return result
			ctx.FreeDesc(&d4)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
			}
			ctx.ReclaimUntrackedRegs()
			d21 = args[3]
			d21.ID = 0
			d22 = ctx.EmitGoCallScalar(GoFuncAddr(OptimizeProcToSerialFunction), []JITValueDesc{d21}, 1)
			ctx.FreeDesc(&d21)
			ctx.FreeDesc(&d22)
			d23 = ctx.EmitGoCallScalar(GoFuncAddr(JITBuildMergeClosure), []JITValueDesc{d22}, 1)
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocReg {
				ctx.ProtectReg(d23.Reg)
			} else if d23.Loc == LocRegPair {
				ctx.ProtectReg(d23.Reg)
				ctx.ProtectReg(d23.Reg2)
			}
			d24 = d23
			if d24.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d24)
			ctx.EmitStoreToStack(d24, int32(bbs[2].PhiBase)+int32(0))
			if d23.Loc == LocReg {
				ctx.UnprotectReg(d23.Reg)
			} else if d23.Loc == LocRegPair {
				ctx.UnprotectReg(d23.Reg)
				ctx.UnprotectReg(d23.Reg2)
			}
			ps25 := PhiState{General: ps.General}
			ps25.OverlayValues = make([]JITValueDesc, 25)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[8] = d8
			ps25.OverlayValues[11] = d11
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps25.OverlayValues[23] = d23
			ps25.OverlayValues[24] = d24
			ps25.PhiValues = make([]JITValueDesc, 1)
			d26 = d23
			ps25.PhiValues[0] = d26
			if ps25.General && bbs[2].Rendered {
				ctx.EmitJmp(lbl3)
				return result
			}
			return bbs[2].RenderPS(ps25)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d27 := ps.PhiValues[0]
					ctx.EnsureDesc(&d27)
					ctx.EmitStoreToStack(d27, int32(bbs[2].PhiBase)+int32(0))
				}
				if bbs[2].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d28 = args[0]
			d28.ID = 0
			d30 = d28
			d30.ID = 0
			d29 = ctx.EmitTagEqualsBorrowed(&d30, tagFastDict, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d28)
			d31 = d29
			ctx.EnsureDesc(&d31)
			if d31.Loc != LocImm && d31.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d31.Loc == LocImm {
				if d31.Imm.Bool() {
			ps32 := PhiState{General: ps.General}
			ps32.OverlayValues = make([]JITValueDesc, 32)
			ps32.OverlayValues[0] = d0
			ps32.OverlayValues[1] = d1
			ps32.OverlayValues[2] = d2
			ps32.OverlayValues[3] = d3
			ps32.OverlayValues[4] = d4
			ps32.OverlayValues[5] = d5
			ps32.OverlayValues[8] = d8
			ps32.OverlayValues[11] = d11
			ps32.OverlayValues[21] = d21
			ps32.OverlayValues[22] = d22
			ps32.OverlayValues[23] = d23
			ps32.OverlayValues[24] = d24
			ps32.OverlayValues[26] = d26
			ps32.OverlayValues[27] = d27
			ps32.OverlayValues[28] = d28
			ps32.OverlayValues[29] = d29
			ps32.OverlayValues[30] = d30
			ps32.OverlayValues[31] = d31
					return bbs[3].RenderPS(ps32)
				}
			ps33 := PhiState{General: ps.General}
			ps33.OverlayValues = make([]JITValueDesc, 32)
			ps33.OverlayValues[0] = d0
			ps33.OverlayValues[1] = d1
			ps33.OverlayValues[2] = d2
			ps33.OverlayValues[3] = d3
			ps33.OverlayValues[4] = d4
			ps33.OverlayValues[5] = d5
			ps33.OverlayValues[8] = d8
			ps33.OverlayValues[11] = d11
			ps33.OverlayValues[21] = d21
			ps33.OverlayValues[22] = d22
			ps33.OverlayValues[23] = d23
			ps33.OverlayValues[24] = d24
			ps33.OverlayValues[26] = d26
			ps33.OverlayValues[27] = d27
			ps33.OverlayValues[28] = d28
			ps33.OverlayValues[29] = d29
			ps33.OverlayValues[30] = d30
			ps33.OverlayValues[31] = d31
				return bbs[5].RenderPS(ps33)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d34 := ps.PhiValues[0]
					ctx.EnsureDesc(&d34)
					ctx.EmitStoreToStack(d34, int32(bbs[2].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d31.Reg, 0)
			ctx.EmitJcc(CcNE, lbl13)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl6)
			ps35 := PhiState{General: true}
			ps35.OverlayValues = make([]JITValueDesc, 35)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[1] = d1
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[3] = d3
			ps35.OverlayValues[4] = d4
			ps35.OverlayValues[5] = d5
			ps35.OverlayValues[8] = d8
			ps35.OverlayValues[11] = d11
			ps35.OverlayValues[21] = d21
			ps35.OverlayValues[22] = d22
			ps35.OverlayValues[23] = d23
			ps35.OverlayValues[24] = d24
			ps35.OverlayValues[26] = d26
			ps35.OverlayValues[27] = d27
			ps35.OverlayValues[28] = d28
			ps35.OverlayValues[29] = d29
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[34] = d34
			ps36 := PhiState{General: true}
			ps36.OverlayValues = make([]JITValueDesc, 35)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[1] = d1
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[3] = d3
			ps36.OverlayValues[4] = d4
			ps36.OverlayValues[5] = d5
			ps36.OverlayValues[8] = d8
			ps36.OverlayValues[11] = d11
			ps36.OverlayValues[21] = d21
			ps36.OverlayValues[22] = d22
			ps36.OverlayValues[23] = d23
			ps36.OverlayValues[24] = d24
			ps36.OverlayValues[26] = d26
			ps36.OverlayValues[27] = d27
			ps36.OverlayValues[28] = d28
			ps36.OverlayValues[29] = d29
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[34] = d34
			snap37 := d0
			snap38 := d1
			snap39 := d2
			snap40 := d3
			snap41 := d4
			snap42 := d5
			snap43 := d8
			snap44 := d11
			snap45 := d21
			snap46 := d22
			snap47 := d23
			snap48 := d24
			snap49 := d26
			snap50 := d27
			snap51 := d28
			snap52 := d29
			snap53 := d30
			snap54 := d31
			snap55 := d34
			alloc56 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps36)
			}
			ctx.RestoreAllocState(alloc56)
			d0 = snap37
			d1 = snap38
			d2 = snap39
			d3 = snap40
			d4 = snap41
			d5 = snap42
			d8 = snap43
			d11 = snap44
			d21 = snap45
			d22 = snap46
			d23 = snap47
			d24 = snap48
			d26 = snap49
			d27 = snap50
			d28 = snap51
			d29 = snap52
			d30 = snap53
			d31 = snap54
			d34 = snap55
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps35)
			}
			return result
			ctx.FreeDesc(&d29)
			return result
			}
			bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
			}
			ctx.ReclaimUntrackedRegs()
			d57 = args[0]
			d57.ID = 0
			var d58 JITValueDesc
			if d57.Loc == LocImm {
				panic("FastDict: LocImm not expected at JIT compile time")
			} else {
				ctx.FreeReg(d57.Reg2)
				d58 = JITValueDesc{Loc: LocReg, Reg: d57.Reg}
				ctx.BindReg(d57.Reg, &d58)
			}
			ctx.FreeDesc(&d57)
			ctx.EnsureDesc(&d58)
			if d58.Loc == LocReg {
				ctx.ProtectReg(d58.Reg)
			} else if d58.Loc == LocRegPair {
				ctx.ProtectReg(d58.Reg)
				ctx.ProtectReg(d58.Reg2)
			}
			d59 = d58
			if d59.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			ctx.EmitStoreToStack(d59, int32(bbs[4].PhiBase)+int32(0))
			if d58.Loc == LocReg {
				ctx.UnprotectReg(d58.Reg)
			} else if d58.Loc == LocRegPair {
				ctx.UnprotectReg(d58.Reg)
				ctx.UnprotectReg(d58.Reg2)
			}
			ps60 := PhiState{General: ps.General}
			ps60.OverlayValues = make([]JITValueDesc, 60)
			ps60.OverlayValues[0] = d0
			ps60.OverlayValues[1] = d1
			ps60.OverlayValues[2] = d2
			ps60.OverlayValues[3] = d3
			ps60.OverlayValues[4] = d4
			ps60.OverlayValues[5] = d5
			ps60.OverlayValues[8] = d8
			ps60.OverlayValues[11] = d11
			ps60.OverlayValues[21] = d21
			ps60.OverlayValues[22] = d22
			ps60.OverlayValues[23] = d23
			ps60.OverlayValues[24] = d24
			ps60.OverlayValues[26] = d26
			ps60.OverlayValues[27] = d27
			ps60.OverlayValues[28] = d28
			ps60.OverlayValues[29] = d29
			ps60.OverlayValues[30] = d30
			ps60.OverlayValues[31] = d31
			ps60.OverlayValues[34] = d34
			ps60.OverlayValues[57] = d57
			ps60.OverlayValues[58] = d58
			ps60.OverlayValues[59] = d59
			ps60.PhiValues = make([]JITValueDesc, 1)
			d61 = d58
			ps60.PhiValues[0] = d61
			if ps60.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps60)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d62 := ps.PhiValues[0]
					ctx.EnsureDesc(&d62)
					ctx.EmitStoreToStack(d62, int32(bbs[4].PhiBase)+int32(0))
				}
				if bbs[4].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
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
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != LocNone {
				d62 = ps.OverlayValues[62]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d63 = args[1]
			d63.ID = 0
			d64 = args[2]
			d64.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d1, d63, d64, d0})
			ctx.FreeDesc(&d63)
			ctx.FreeDesc(&d64)
			ctx.FreeDesc(&d0)
			var d65 JITValueDesc
			if d1.Loc == LocImm {
				panic("NewFastDict: LocImm not expected at JIT compile time")
			} else {
				r2 := ctx.AllocReg()
				ctx.EmitMovRegImm64(r2, makeAux(tagFastDict, 0))
				d65 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d1.Reg, Reg2: r2}
				ctx.BindReg(d1.Reg, &d65)
				ctx.BindReg(r2, &d65)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d65)
			if d65.Loc == LocRegPair {
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
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[5].VisitCount >= 2 {
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
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
			ctx.ReclaimUntrackedRegs()
			d66 = args[0]
			d66.ID = 0
			d68 = d66
			d68.ID = 0
			d67 = ctx.EmitTagEqualsBorrowed(&d68, tagSlice, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d66)
			d69 = d67
			ctx.EnsureDesc(&d69)
			if d69.Loc != LocImm && d69.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d69.Loc == LocImm {
				if d69.Imm.Bool() {
			ps70 := PhiState{General: ps.General}
			ps70.OverlayValues = make([]JITValueDesc, 70)
			ps70.OverlayValues[0] = d0
			ps70.OverlayValues[1] = d1
			ps70.OverlayValues[2] = d2
			ps70.OverlayValues[3] = d3
			ps70.OverlayValues[4] = d4
			ps70.OverlayValues[5] = d5
			ps70.OverlayValues[8] = d8
			ps70.OverlayValues[11] = d11
			ps70.OverlayValues[21] = d21
			ps70.OverlayValues[22] = d22
			ps70.OverlayValues[23] = d23
			ps70.OverlayValues[24] = d24
			ps70.OverlayValues[26] = d26
			ps70.OverlayValues[27] = d27
			ps70.OverlayValues[28] = d28
			ps70.OverlayValues[29] = d29
			ps70.OverlayValues[30] = d30
			ps70.OverlayValues[31] = d31
			ps70.OverlayValues[34] = d34
			ps70.OverlayValues[57] = d57
			ps70.OverlayValues[58] = d58
			ps70.OverlayValues[59] = d59
			ps70.OverlayValues[61] = d61
			ps70.OverlayValues[62] = d62
			ps70.OverlayValues[63] = d63
			ps70.OverlayValues[64] = d64
			ps70.OverlayValues[65] = d65
			ps70.OverlayValues[66] = d66
			ps70.OverlayValues[67] = d67
			ps70.OverlayValues[68] = d68
			ps70.OverlayValues[69] = d69
					return bbs[6].RenderPS(ps70)
				}
			ps71 := PhiState{General: ps.General}
			ps71.OverlayValues = make([]JITValueDesc, 70)
			ps71.OverlayValues[0] = d0
			ps71.OverlayValues[1] = d1
			ps71.OverlayValues[2] = d2
			ps71.OverlayValues[3] = d3
			ps71.OverlayValues[4] = d4
			ps71.OverlayValues[5] = d5
			ps71.OverlayValues[8] = d8
			ps71.OverlayValues[11] = d11
			ps71.OverlayValues[21] = d21
			ps71.OverlayValues[22] = d22
			ps71.OverlayValues[23] = d23
			ps71.OverlayValues[24] = d24
			ps71.OverlayValues[26] = d26
			ps71.OverlayValues[27] = d27
			ps71.OverlayValues[28] = d28
			ps71.OverlayValues[29] = d29
			ps71.OverlayValues[30] = d30
			ps71.OverlayValues[31] = d31
			ps71.OverlayValues[34] = d34
			ps71.OverlayValues[57] = d57
			ps71.OverlayValues[58] = d58
			ps71.OverlayValues[59] = d59
			ps71.OverlayValues[61] = d61
			ps71.OverlayValues[62] = d62
			ps71.OverlayValues[63] = d63
			ps71.OverlayValues[64] = d64
			ps71.OverlayValues[65] = d65
			ps71.OverlayValues[66] = d66
			ps71.OverlayValues[67] = d67
			ps71.OverlayValues[68] = d68
			ps71.OverlayValues[69] = d69
				return bbs[7].RenderPS(ps71)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d69.Reg, 0)
			ctx.EmitJcc(CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl8)
			ps72 := PhiState{General: true}
			ps72.OverlayValues = make([]JITValueDesc, 70)
			ps72.OverlayValues[0] = d0
			ps72.OverlayValues[1] = d1
			ps72.OverlayValues[2] = d2
			ps72.OverlayValues[3] = d3
			ps72.OverlayValues[4] = d4
			ps72.OverlayValues[5] = d5
			ps72.OverlayValues[8] = d8
			ps72.OverlayValues[11] = d11
			ps72.OverlayValues[21] = d21
			ps72.OverlayValues[22] = d22
			ps72.OverlayValues[23] = d23
			ps72.OverlayValues[24] = d24
			ps72.OverlayValues[26] = d26
			ps72.OverlayValues[27] = d27
			ps72.OverlayValues[28] = d28
			ps72.OverlayValues[29] = d29
			ps72.OverlayValues[30] = d30
			ps72.OverlayValues[31] = d31
			ps72.OverlayValues[34] = d34
			ps72.OverlayValues[57] = d57
			ps72.OverlayValues[58] = d58
			ps72.OverlayValues[59] = d59
			ps72.OverlayValues[61] = d61
			ps72.OverlayValues[62] = d62
			ps72.OverlayValues[63] = d63
			ps72.OverlayValues[64] = d64
			ps72.OverlayValues[65] = d65
			ps72.OverlayValues[66] = d66
			ps72.OverlayValues[67] = d67
			ps72.OverlayValues[68] = d68
			ps72.OverlayValues[69] = d69
			ps73 := PhiState{General: true}
			ps73.OverlayValues = make([]JITValueDesc, 70)
			ps73.OverlayValues[0] = d0
			ps73.OverlayValues[1] = d1
			ps73.OverlayValues[2] = d2
			ps73.OverlayValues[3] = d3
			ps73.OverlayValues[4] = d4
			ps73.OverlayValues[5] = d5
			ps73.OverlayValues[8] = d8
			ps73.OverlayValues[11] = d11
			ps73.OverlayValues[21] = d21
			ps73.OverlayValues[22] = d22
			ps73.OverlayValues[23] = d23
			ps73.OverlayValues[24] = d24
			ps73.OverlayValues[26] = d26
			ps73.OverlayValues[27] = d27
			ps73.OverlayValues[28] = d28
			ps73.OverlayValues[29] = d29
			ps73.OverlayValues[30] = d30
			ps73.OverlayValues[31] = d31
			ps73.OverlayValues[34] = d34
			ps73.OverlayValues[57] = d57
			ps73.OverlayValues[58] = d58
			ps73.OverlayValues[59] = d59
			ps73.OverlayValues[61] = d61
			ps73.OverlayValues[62] = d62
			ps73.OverlayValues[63] = d63
			ps73.OverlayValues[64] = d64
			ps73.OverlayValues[65] = d65
			ps73.OverlayValues[66] = d66
			ps73.OverlayValues[67] = d67
			ps73.OverlayValues[68] = d68
			ps73.OverlayValues[69] = d69
			snap74 := d0
			snap75 := d1
			snap76 := d2
			snap77 := d3
			snap78 := d4
			snap79 := d5
			snap80 := d8
			snap81 := d11
			snap82 := d21
			snap83 := d22
			snap84 := d23
			snap85 := d24
			snap86 := d26
			snap87 := d27
			snap88 := d28
			snap89 := d29
			snap90 := d30
			snap91 := d31
			snap92 := d34
			snap93 := d57
			snap94 := d58
			snap95 := d59
			snap96 := d61
			snap97 := d62
			snap98 := d63
			snap99 := d64
			snap100 := d65
			snap101 := d66
			snap102 := d67
			snap103 := d68
			snap104 := d69
			alloc105 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps73)
			}
			ctx.RestoreAllocState(alloc105)
			d0 = snap74
			d1 = snap75
			d2 = snap76
			d3 = snap77
			d4 = snap78
			d5 = snap79
			d8 = snap80
			d11 = snap81
			d21 = snap82
			d22 = snap83
			d23 = snap84
			d24 = snap85
			d26 = snap86
			d27 = snap87
			d28 = snap88
			d29 = snap89
			d30 = snap90
			d31 = snap91
			d34 = snap92
			d57 = snap93
			d58 = snap94
			d59 = snap95
			d61 = snap96
			d62 = snap97
			d63 = snap98
			d64 = snap99
			d65 = snap100
			d66 = snap101
			d67 = snap102
			d68 = snap103
			d69 = snap104
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps72)
			}
			return result
			ctx.FreeDesc(&d67)
			return result
			}
			bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[6].VisitCount >= 2 {
					ps.General = true
					return bbs[6].RenderPS(ps)
				}
			}
			bbs[6].VisitCount++
			if ps.General {
				if bbs[6].Rendered {
					ctx.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			ctx.ReclaimUntrackedRegs()
			d106 = args[0]
			d106.ID = 0
			var d107 JITValueDesc
			if d106.Loc == LocImm {
				slice := d106.Imm.Slice()
				d107 = JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: NewInt(int64(len(slice)))}
			} else {
				r3 := ctx.AllocReg()
				ctx.EmitMovRegReg(r3, d106.Reg2)
				ctx.EmitShrRegImm8(r3, 8)
				ctx.FreeReg(d106.Reg2)
				d107 = JITValueDesc{Loc: LocRegPair, Reg: d106.Reg, Reg2: r3}
				ctx.BindReg(d106.Reg, &d107)
				ctx.BindReg(r3, &d107)
			}
			ctx.FreeDesc(&d106)
			var d108 JITValueDesc
			if d107.Loc == LocImm {
				d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d107.StackOff))}
			} else {
				ctx.EnsureDesc(&d107)
				if d107.Loc == LocRegPair {
					d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg2}
					ctx.BindReg(d107.Reg2, &d108)
					ctx.BindReg(d107.Reg2, &d108)
				} else if d107.Loc == LocReg {
					d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg}
					ctx.BindReg(d107.Reg, &d108)
					ctx.BindReg(d107.Reg, &d108)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d108)
			var d109 JITValueDesc
			if d108.Loc == LocImm {
				d109 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d108.Imm.Int() / 2)}
			} else {
				ctx.EmitShrRegImm8(d108.Reg, 1)
				d109 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d108.Reg}
				ctx.BindReg(d108.Reg, &d109)
			}
			if d109.Loc == LocReg && d108.Loc == LocReg && d109.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = LocNone
			}
			ctx.FreeDesc(&d108)
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d109)
			var d110 JITValueDesc
			if d109.Loc == LocImm {
				d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d109.Imm.Int() + 4)}
			} else {
				scratch := ctx.AllocRegExcept(d109.Reg)
				ctx.EmitMovRegReg(scratch, d109.Reg)
				ctx.EmitAddRegImm32(scratch, int32(4))
				d110 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d110)
			}
			if d110.Loc == LocReg && d109.Loc == LocReg && d110.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = LocNone
			}
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d110)
			d111 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d110}, 1)
			ctx.FreeDesc(&d110)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[8].PhiBase)+int32(0))
			ps112 := PhiState{General: ps.General}
			ps112.OverlayValues = make([]JITValueDesc, 112)
			ps112.OverlayValues[0] = d0
			ps112.OverlayValues[1] = d1
			ps112.OverlayValues[2] = d2
			ps112.OverlayValues[3] = d3
			ps112.OverlayValues[4] = d4
			ps112.OverlayValues[5] = d5
			ps112.OverlayValues[8] = d8
			ps112.OverlayValues[11] = d11
			ps112.OverlayValues[21] = d21
			ps112.OverlayValues[22] = d22
			ps112.OverlayValues[23] = d23
			ps112.OverlayValues[24] = d24
			ps112.OverlayValues[26] = d26
			ps112.OverlayValues[27] = d27
			ps112.OverlayValues[28] = d28
			ps112.OverlayValues[29] = d29
			ps112.OverlayValues[30] = d30
			ps112.OverlayValues[31] = d31
			ps112.OverlayValues[34] = d34
			ps112.OverlayValues[57] = d57
			ps112.OverlayValues[58] = d58
			ps112.OverlayValues[59] = d59
			ps112.OverlayValues[61] = d61
			ps112.OverlayValues[62] = d62
			ps112.OverlayValues[63] = d63
			ps112.OverlayValues[64] = d64
			ps112.OverlayValues[65] = d65
			ps112.OverlayValues[66] = d66
			ps112.OverlayValues[67] = d67
			ps112.OverlayValues[68] = d68
			ps112.OverlayValues[69] = d69
			ps112.OverlayValues[106] = d106
			ps112.OverlayValues[107] = d107
			ps112.OverlayValues[108] = d108
			ps112.OverlayValues[109] = d109
			ps112.OverlayValues[110] = d110
			ps112.OverlayValues[111] = d111
			ps112.PhiValues = make([]JITValueDesc, 1)
			d113 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps112.PhiValues[0] = d113
			if ps112.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps112)
			return result
			}
			bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[7].VisitCount >= 2 {
					ps.General = true
					return bbs[7].RenderPS(ps)
				}
			}
			bbs[7].VisitCount++
			if ps.General {
				if bbs[7].Rendered {
					ctx.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
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
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
				d113 = ps.OverlayValues[113]
			}
			ctx.ReclaimUntrackedRegs()
			d114 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(8)}
			d115 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d114}, 1)
			ctx.EnsureDesc(&d115)
			if d115.Loc == LocReg {
				ctx.ProtectReg(d115.Reg)
			} else if d115.Loc == LocRegPair {
				ctx.ProtectReg(d115.Reg)
				ctx.ProtectReg(d115.Reg2)
			}
			d116 = d115
			if d116.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d116)
			ctx.EmitStoreToStack(d116, int32(bbs[4].PhiBase)+int32(0))
			if d115.Loc == LocReg {
				ctx.UnprotectReg(d115.Reg)
			} else if d115.Loc == LocRegPair {
				ctx.UnprotectReg(d115.Reg)
				ctx.UnprotectReg(d115.Reg2)
			}
			ps117 := PhiState{General: ps.General}
			ps117.OverlayValues = make([]JITValueDesc, 117)
			ps117.OverlayValues[0] = d0
			ps117.OverlayValues[1] = d1
			ps117.OverlayValues[2] = d2
			ps117.OverlayValues[3] = d3
			ps117.OverlayValues[4] = d4
			ps117.OverlayValues[5] = d5
			ps117.OverlayValues[8] = d8
			ps117.OverlayValues[11] = d11
			ps117.OverlayValues[21] = d21
			ps117.OverlayValues[22] = d22
			ps117.OverlayValues[23] = d23
			ps117.OverlayValues[24] = d24
			ps117.OverlayValues[26] = d26
			ps117.OverlayValues[27] = d27
			ps117.OverlayValues[28] = d28
			ps117.OverlayValues[29] = d29
			ps117.OverlayValues[30] = d30
			ps117.OverlayValues[31] = d31
			ps117.OverlayValues[34] = d34
			ps117.OverlayValues[57] = d57
			ps117.OverlayValues[58] = d58
			ps117.OverlayValues[59] = d59
			ps117.OverlayValues[61] = d61
			ps117.OverlayValues[62] = d62
			ps117.OverlayValues[63] = d63
			ps117.OverlayValues[64] = d64
			ps117.OverlayValues[65] = d65
			ps117.OverlayValues[66] = d66
			ps117.OverlayValues[67] = d67
			ps117.OverlayValues[68] = d68
			ps117.OverlayValues[69] = d69
			ps117.OverlayValues[106] = d106
			ps117.OverlayValues[107] = d107
			ps117.OverlayValues[108] = d108
			ps117.OverlayValues[109] = d109
			ps117.OverlayValues[110] = d110
			ps117.OverlayValues[111] = d111
			ps117.OverlayValues[113] = d113
			ps117.OverlayValues[114] = d114
			ps117.OverlayValues[115] = d115
			ps117.OverlayValues[116] = d116
			ps117.PhiValues = make([]JITValueDesc, 1)
			d118 = d115
			ps117.PhiValues[0] = d118
			if ps117.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps117)
			return result
			}
			bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d119 := ps.PhiValues[0]
					ctx.EnsureDesc(&d119)
					ctx.EmitStoreToStack(d119, int32(bbs[8].PhiBase)+int32(0))
				}
				if bbs[8].VisitCount >= 2 {
					ps.General = true
					return bbs[8].RenderPS(ps)
				}
			}
			bbs[8].VisitCount++
			if ps.General {
				if bbs[8].Rendered {
					ctx.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
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
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
				d113 = ps.OverlayValues[113]
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
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
				d119 = ps.OverlayValues[119]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d2 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d120 JITValueDesc
			if d2.Loc == LocImm {
				d120 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d120 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d120)
			}
			if d120.Loc == LocReg && d2.Loc == LocReg && d120.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			var d121 JITValueDesc
			if d107.Loc == LocImm {
				d121 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d107.StackOff))}
			} else {
				ctx.EnsureDesc(&d107)
				if d107.Loc == LocRegPair {
					d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg2}
					ctx.BindReg(d107.Reg2, &d121)
					ctx.BindReg(d107.Reg2, &d121)
				} else if d107.Loc == LocReg {
					d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg}
					ctx.BindReg(d107.Reg, &d121)
					ctx.BindReg(d107.Reg, &d121)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d121)
			var d122 JITValueDesc
			if d120.Loc == LocImm && d121.Loc == LocImm {
				d122 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d120.Imm.Int() < d121.Imm.Int())}
			} else if d121.Loc == LocImm {
				r4 := ctx.AllocReg()
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d120.Reg, int32(d121.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d121.Imm.Int()))
					ctx.EmitCmpInt64(d120.Reg, RegR11)
				}
				ctx.EmitSetcc(r4, CcL)
				d122 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d122)
			} else if d120.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d120.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d121.Reg)
				ctx.EmitSetcc(r5, CcL)
				d122 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d122)
			} else {
				r6 := ctx.AllocReg()
				ctx.EmitCmpInt64(d120.Reg, d121.Reg)
				ctx.EmitSetcc(r6, CcL)
				d122 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d122)
			}
			ctx.FreeDesc(&d120)
			ctx.FreeDesc(&d121)
			d123 = d122
			ctx.EnsureDesc(&d123)
			if d123.Loc != LocImm && d123.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d123.Loc == LocImm {
				if d123.Imm.Bool() {
			ps124 := PhiState{General: ps.General}
			ps124.OverlayValues = make([]JITValueDesc, 124)
			ps124.OverlayValues[0] = d0
			ps124.OverlayValues[1] = d1
			ps124.OverlayValues[2] = d2
			ps124.OverlayValues[3] = d3
			ps124.OverlayValues[4] = d4
			ps124.OverlayValues[5] = d5
			ps124.OverlayValues[8] = d8
			ps124.OverlayValues[11] = d11
			ps124.OverlayValues[21] = d21
			ps124.OverlayValues[22] = d22
			ps124.OverlayValues[23] = d23
			ps124.OverlayValues[24] = d24
			ps124.OverlayValues[26] = d26
			ps124.OverlayValues[27] = d27
			ps124.OverlayValues[28] = d28
			ps124.OverlayValues[29] = d29
			ps124.OverlayValues[30] = d30
			ps124.OverlayValues[31] = d31
			ps124.OverlayValues[34] = d34
			ps124.OverlayValues[57] = d57
			ps124.OverlayValues[58] = d58
			ps124.OverlayValues[59] = d59
			ps124.OverlayValues[61] = d61
			ps124.OverlayValues[62] = d62
			ps124.OverlayValues[63] = d63
			ps124.OverlayValues[64] = d64
			ps124.OverlayValues[65] = d65
			ps124.OverlayValues[66] = d66
			ps124.OverlayValues[67] = d67
			ps124.OverlayValues[68] = d68
			ps124.OverlayValues[69] = d69
			ps124.OverlayValues[106] = d106
			ps124.OverlayValues[107] = d107
			ps124.OverlayValues[108] = d108
			ps124.OverlayValues[109] = d109
			ps124.OverlayValues[110] = d110
			ps124.OverlayValues[111] = d111
			ps124.OverlayValues[113] = d113
			ps124.OverlayValues[114] = d114
			ps124.OverlayValues[115] = d115
			ps124.OverlayValues[116] = d116
			ps124.OverlayValues[118] = d118
			ps124.OverlayValues[119] = d119
			ps124.OverlayValues[120] = d120
			ps124.OverlayValues[121] = d121
			ps124.OverlayValues[122] = d122
			ps124.OverlayValues[123] = d123
					return bbs[9].RenderPS(ps124)
				}
			ctx.EnsureDesc(&d111)
			if d111.Loc == LocReg {
				ctx.ProtectReg(d111.Reg)
			} else if d111.Loc == LocRegPair {
				ctx.ProtectReg(d111.Reg)
				ctx.ProtectReg(d111.Reg2)
			}
			d125 = d111
			if d125.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d125)
			ctx.EmitStoreToStack(d125, int32(bbs[4].PhiBase)+int32(0))
			if d111.Loc == LocReg {
				ctx.UnprotectReg(d111.Reg)
			} else if d111.Loc == LocRegPair {
				ctx.UnprotectReg(d111.Reg)
				ctx.UnprotectReg(d111.Reg2)
			}
			ps126 := PhiState{General: ps.General}
			ps126.OverlayValues = make([]JITValueDesc, 126)
			ps126.OverlayValues[0] = d0
			ps126.OverlayValues[1] = d1
			ps126.OverlayValues[2] = d2
			ps126.OverlayValues[3] = d3
			ps126.OverlayValues[4] = d4
			ps126.OverlayValues[5] = d5
			ps126.OverlayValues[8] = d8
			ps126.OverlayValues[11] = d11
			ps126.OverlayValues[21] = d21
			ps126.OverlayValues[22] = d22
			ps126.OverlayValues[23] = d23
			ps126.OverlayValues[24] = d24
			ps126.OverlayValues[26] = d26
			ps126.OverlayValues[27] = d27
			ps126.OverlayValues[28] = d28
			ps126.OverlayValues[29] = d29
			ps126.OverlayValues[30] = d30
			ps126.OverlayValues[31] = d31
			ps126.OverlayValues[34] = d34
			ps126.OverlayValues[57] = d57
			ps126.OverlayValues[58] = d58
			ps126.OverlayValues[59] = d59
			ps126.OverlayValues[61] = d61
			ps126.OverlayValues[62] = d62
			ps126.OverlayValues[63] = d63
			ps126.OverlayValues[64] = d64
			ps126.OverlayValues[65] = d65
			ps126.OverlayValues[66] = d66
			ps126.OverlayValues[67] = d67
			ps126.OverlayValues[68] = d68
			ps126.OverlayValues[69] = d69
			ps126.OverlayValues[106] = d106
			ps126.OverlayValues[107] = d107
			ps126.OverlayValues[108] = d108
			ps126.OverlayValues[109] = d109
			ps126.OverlayValues[110] = d110
			ps126.OverlayValues[111] = d111
			ps126.OverlayValues[113] = d113
			ps126.OverlayValues[114] = d114
			ps126.OverlayValues[115] = d115
			ps126.OverlayValues[116] = d116
			ps126.OverlayValues[118] = d118
			ps126.OverlayValues[119] = d119
			ps126.OverlayValues[120] = d120
			ps126.OverlayValues[121] = d121
			ps126.OverlayValues[122] = d122
			ps126.OverlayValues[123] = d123
			ps126.OverlayValues[125] = d125
			ps126.PhiValues = make([]JITValueDesc, 1)
			d127 = d111
			ps126.PhiValues[0] = d127
				return bbs[4].RenderPS(ps126)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d128 := ps.PhiValues[0]
					ctx.EnsureDesc(&d128)
					ctx.EmitStoreToStack(d128, int32(bbs[8].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d123.Reg, 0)
			ctx.EmitJcc(CcNE, lbl17)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl18)
			ctx.EnsureDesc(&d111)
			if d111.Loc == LocReg {
				ctx.ProtectReg(d111.Reg)
			} else if d111.Loc == LocRegPair {
				ctx.ProtectReg(d111.Reg)
				ctx.ProtectReg(d111.Reg2)
			}
			d129 = d111
			if d129.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d129)
			ctx.EmitStoreToStack(d129, int32(bbs[4].PhiBase)+int32(0))
			if d111.Loc == LocReg {
				ctx.UnprotectReg(d111.Reg)
			} else if d111.Loc == LocRegPair {
				ctx.UnprotectReg(d111.Reg)
				ctx.UnprotectReg(d111.Reg2)
			}
			ctx.EmitJmp(lbl5)
			ps130 := PhiState{General: true}
			ps130.OverlayValues = make([]JITValueDesc, 130)
			ps130.OverlayValues[0] = d0
			ps130.OverlayValues[1] = d1
			ps130.OverlayValues[2] = d2
			ps130.OverlayValues[3] = d3
			ps130.OverlayValues[4] = d4
			ps130.OverlayValues[5] = d5
			ps130.OverlayValues[8] = d8
			ps130.OverlayValues[11] = d11
			ps130.OverlayValues[21] = d21
			ps130.OverlayValues[22] = d22
			ps130.OverlayValues[23] = d23
			ps130.OverlayValues[24] = d24
			ps130.OverlayValues[26] = d26
			ps130.OverlayValues[27] = d27
			ps130.OverlayValues[28] = d28
			ps130.OverlayValues[29] = d29
			ps130.OverlayValues[30] = d30
			ps130.OverlayValues[31] = d31
			ps130.OverlayValues[34] = d34
			ps130.OverlayValues[57] = d57
			ps130.OverlayValues[58] = d58
			ps130.OverlayValues[59] = d59
			ps130.OverlayValues[61] = d61
			ps130.OverlayValues[62] = d62
			ps130.OverlayValues[63] = d63
			ps130.OverlayValues[64] = d64
			ps130.OverlayValues[65] = d65
			ps130.OverlayValues[66] = d66
			ps130.OverlayValues[67] = d67
			ps130.OverlayValues[68] = d68
			ps130.OverlayValues[69] = d69
			ps130.OverlayValues[106] = d106
			ps130.OverlayValues[107] = d107
			ps130.OverlayValues[108] = d108
			ps130.OverlayValues[109] = d109
			ps130.OverlayValues[110] = d110
			ps130.OverlayValues[111] = d111
			ps130.OverlayValues[113] = d113
			ps130.OverlayValues[114] = d114
			ps130.OverlayValues[115] = d115
			ps130.OverlayValues[116] = d116
			ps130.OverlayValues[118] = d118
			ps130.OverlayValues[119] = d119
			ps130.OverlayValues[120] = d120
			ps130.OverlayValues[121] = d121
			ps130.OverlayValues[122] = d122
			ps130.OverlayValues[123] = d123
			ps130.OverlayValues[125] = d125
			ps130.OverlayValues[127] = d127
			ps130.OverlayValues[128] = d128
			ps130.OverlayValues[129] = d129
			ps131 := PhiState{General: true}
			ps131.OverlayValues = make([]JITValueDesc, 130)
			ps131.OverlayValues[0] = d0
			ps131.OverlayValues[1] = d1
			ps131.OverlayValues[2] = d2
			ps131.OverlayValues[3] = d3
			ps131.OverlayValues[4] = d4
			ps131.OverlayValues[5] = d5
			ps131.OverlayValues[8] = d8
			ps131.OverlayValues[11] = d11
			ps131.OverlayValues[21] = d21
			ps131.OverlayValues[22] = d22
			ps131.OverlayValues[23] = d23
			ps131.OverlayValues[24] = d24
			ps131.OverlayValues[26] = d26
			ps131.OverlayValues[27] = d27
			ps131.OverlayValues[28] = d28
			ps131.OverlayValues[29] = d29
			ps131.OverlayValues[30] = d30
			ps131.OverlayValues[31] = d31
			ps131.OverlayValues[34] = d34
			ps131.OverlayValues[57] = d57
			ps131.OverlayValues[58] = d58
			ps131.OverlayValues[59] = d59
			ps131.OverlayValues[61] = d61
			ps131.OverlayValues[62] = d62
			ps131.OverlayValues[63] = d63
			ps131.OverlayValues[64] = d64
			ps131.OverlayValues[65] = d65
			ps131.OverlayValues[66] = d66
			ps131.OverlayValues[67] = d67
			ps131.OverlayValues[68] = d68
			ps131.OverlayValues[69] = d69
			ps131.OverlayValues[106] = d106
			ps131.OverlayValues[107] = d107
			ps131.OverlayValues[108] = d108
			ps131.OverlayValues[109] = d109
			ps131.OverlayValues[110] = d110
			ps131.OverlayValues[111] = d111
			ps131.OverlayValues[113] = d113
			ps131.OverlayValues[114] = d114
			ps131.OverlayValues[115] = d115
			ps131.OverlayValues[116] = d116
			ps131.OverlayValues[118] = d118
			ps131.OverlayValues[119] = d119
			ps131.OverlayValues[120] = d120
			ps131.OverlayValues[121] = d121
			ps131.OverlayValues[122] = d122
			ps131.OverlayValues[123] = d123
			ps131.OverlayValues[125] = d125
			ps131.OverlayValues[127] = d127
			ps131.OverlayValues[128] = d128
			ps131.OverlayValues[129] = d129
			ps131.PhiValues = make([]JITValueDesc, 1)
			d132 = d111
			ps131.PhiValues[0] = d132
			snap133 := d0
			snap134 := d1
			snap135 := d2
			snap136 := d3
			snap137 := d4
			snap138 := d5
			snap139 := d8
			snap140 := d11
			snap141 := d21
			snap142 := d22
			snap143 := d23
			snap144 := d24
			snap145 := d26
			snap146 := d27
			snap147 := d28
			snap148 := d29
			snap149 := d30
			snap150 := d31
			snap151 := d34
			snap152 := d57
			snap153 := d58
			snap154 := d59
			snap155 := d61
			snap156 := d62
			snap157 := d63
			snap158 := d64
			snap159 := d65
			snap160 := d66
			snap161 := d67
			snap162 := d68
			snap163 := d69
			snap164 := d106
			snap165 := d107
			snap166 := d108
			snap167 := d109
			snap168 := d110
			snap169 := d111
			snap170 := d113
			snap171 := d114
			snap172 := d115
			snap173 := d116
			snap174 := d118
			snap175 := d119
			snap176 := d120
			snap177 := d121
			snap178 := d122
			snap179 := d123
			snap180 := d125
			snap181 := d127
			snap182 := d128
			snap183 := d129
			snap184 := d132
			alloc185 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps131)
			}
			ctx.RestoreAllocState(alloc185)
			d0 = snap133
			d1 = snap134
			d2 = snap135
			d3 = snap136
			d4 = snap137
			d5 = snap138
			d8 = snap139
			d11 = snap140
			d21 = snap141
			d22 = snap142
			d23 = snap143
			d24 = snap144
			d26 = snap145
			d27 = snap146
			d28 = snap147
			d29 = snap148
			d30 = snap149
			d31 = snap150
			d34 = snap151
			d57 = snap152
			d58 = snap153
			d59 = snap154
			d61 = snap155
			d62 = snap156
			d63 = snap157
			d64 = snap158
			d65 = snap159
			d66 = snap160
			d67 = snap161
			d68 = snap162
			d69 = snap163
			d106 = snap164
			d107 = snap165
			d108 = snap166
			d109 = snap167
			d110 = snap168
			d111 = snap169
			d113 = snap170
			d114 = snap171
			d115 = snap172
			d116 = snap173
			d118 = snap174
			d119 = snap175
			d120 = snap176
			d121 = snap177
			d122 = snap178
			d123 = snap179
			d125 = snap180
			d127 = snap181
			d128 = snap182
			d129 = snap183
			d132 = snap184
			if !bbs[9].Rendered {
				return bbs[9].RenderPS(ps130)
			}
			return result
			ctx.FreeDesc(&d122)
			return result
			}
			bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[9].VisitCount >= 2 {
					ps.General = true
					return bbs[9].RenderPS(ps)
				}
			}
			bbs[9].VisitCount++
			if ps.General {
				if bbs[9].Rendered {
					ctx.EmitJmp(lbl10)
					return result
				}
				bbs[9].Rendered = true
				bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_9 = bbs[9].Address
				ctx.MarkLabel(lbl10)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
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
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
				d11 = ps.OverlayValues[11]
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
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
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
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
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
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
				d113 = ps.OverlayValues[113]
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
			if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != LocNone {
				d125 = ps.OverlayValues[125]
			}
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != LocNone {
				d132 = ps.OverlayValues[132]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			r7 := ctx.AllocReg()
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d107)
			if d2.Loc == LocImm {
				ctx.EmitMovRegImm64(r7, uint64(d2.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r7, d2.Reg)
				ctx.EmitShlRegImm8(r7, 4)
			}
			if d107.Loc == LocImm {
				ctx.EmitMovRegImm64(RegR11, uint64(d107.Imm.Int()))
				ctx.EmitAddInt64(r7, RegR11)
			} else {
				ctx.EmitAddInt64(r7, d107.Reg)
			}
			r8 := ctx.AllocRegExcept(r7)
			r9 := ctx.AllocRegExcept(r7, r8)
			ctx.EmitMovRegMem(r8, r7, 0)
			ctx.EmitMovRegMem(r9, r7, 8)
			ctx.FreeReg(r7)
			d186 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r8, Reg2: r9}
			ctx.BindReg(r8, &d186)
			ctx.BindReg(r9, &d186)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d187 JITValueDesc
			if d2.Loc == LocImm {
				d187 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d187 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d187)
			}
			if d187.Loc == LocReg && d2.Loc == LocReg && d187.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.EnsureDesc(&d187)
			r10 := ctx.AllocReg()
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d107)
			if d187.Loc == LocImm {
				ctx.EmitMovRegImm64(r10, uint64(d187.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r10, d187.Reg)
				ctx.EmitShlRegImm8(r10, 4)
			}
			if d107.Loc == LocImm {
				ctx.EmitMovRegImm64(RegR11, uint64(d107.Imm.Int()))
				ctx.EmitAddInt64(r10, RegR11)
			} else {
				ctx.EmitAddInt64(r10, d107.Reg)
			}
			r11 := ctx.AllocRegExcept(r10)
			r12 := ctx.AllocRegExcept(r10, r11)
			ctx.EmitMovRegMem(r11, r10, 0)
			ctx.EmitMovRegMem(r12, r10, 8)
			ctx.FreeReg(r10)
			d188 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r11, Reg2: r12}
			ctx.BindReg(r11, &d188)
			ctx.BindReg(r12, &d188)
			ctx.FreeDesc(&d187)
			d189 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d111, d186, d188, d189})
			ctx.FreeDesc(&d186)
			ctx.FreeDesc(&d188)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d190 JITValueDesc
			if d2.Loc == LocImm {
				d190 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 2)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(2))
				d190 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d190)
			}
			if d190.Loc == LocReg && d2.Loc == LocReg && d190.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d190)
			if d190.Loc == LocReg {
				ctx.ProtectReg(d190.Reg)
			} else if d190.Loc == LocRegPair {
				ctx.ProtectReg(d190.Reg)
				ctx.ProtectReg(d190.Reg2)
			}
			d191 = d190
			if d191.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d191)
			ctx.EmitStoreToStack(d191, int32(bbs[8].PhiBase)+int32(0))
			if d190.Loc == LocReg {
				ctx.UnprotectReg(d190.Reg)
			} else if d190.Loc == LocRegPair {
				ctx.UnprotectReg(d190.Reg)
				ctx.UnprotectReg(d190.Reg2)
			}
			ps192 := PhiState{General: ps.General}
			ps192.OverlayValues = make([]JITValueDesc, 192)
			ps192.OverlayValues[0] = d0
			ps192.OverlayValues[1] = d1
			ps192.OverlayValues[2] = d2
			ps192.OverlayValues[3] = d3
			ps192.OverlayValues[4] = d4
			ps192.OverlayValues[5] = d5
			ps192.OverlayValues[8] = d8
			ps192.OverlayValues[11] = d11
			ps192.OverlayValues[21] = d21
			ps192.OverlayValues[22] = d22
			ps192.OverlayValues[23] = d23
			ps192.OverlayValues[24] = d24
			ps192.OverlayValues[26] = d26
			ps192.OverlayValues[27] = d27
			ps192.OverlayValues[28] = d28
			ps192.OverlayValues[29] = d29
			ps192.OverlayValues[30] = d30
			ps192.OverlayValues[31] = d31
			ps192.OverlayValues[34] = d34
			ps192.OverlayValues[57] = d57
			ps192.OverlayValues[58] = d58
			ps192.OverlayValues[59] = d59
			ps192.OverlayValues[61] = d61
			ps192.OverlayValues[62] = d62
			ps192.OverlayValues[63] = d63
			ps192.OverlayValues[64] = d64
			ps192.OverlayValues[65] = d65
			ps192.OverlayValues[66] = d66
			ps192.OverlayValues[67] = d67
			ps192.OverlayValues[68] = d68
			ps192.OverlayValues[69] = d69
			ps192.OverlayValues[106] = d106
			ps192.OverlayValues[107] = d107
			ps192.OverlayValues[108] = d108
			ps192.OverlayValues[109] = d109
			ps192.OverlayValues[110] = d110
			ps192.OverlayValues[111] = d111
			ps192.OverlayValues[113] = d113
			ps192.OverlayValues[114] = d114
			ps192.OverlayValues[115] = d115
			ps192.OverlayValues[116] = d116
			ps192.OverlayValues[118] = d118
			ps192.OverlayValues[119] = d119
			ps192.OverlayValues[120] = d120
			ps192.OverlayValues[121] = d121
			ps192.OverlayValues[122] = d122
			ps192.OverlayValues[123] = d123
			ps192.OverlayValues[125] = d125
			ps192.OverlayValues[127] = d127
			ps192.OverlayValues[128] = d128
			ps192.OverlayValues[129] = d129
			ps192.OverlayValues[132] = d132
			ps192.OverlayValues[186] = d186
			ps192.OverlayValues[187] = d187
			ps192.OverlayValues[188] = d188
			ps192.OverlayValues[189] = d189
			ps192.OverlayValues[190] = d190
			ps192.OverlayValues[191] = d191
			ps192.PhiValues = make([]JITValueDesc, 1)
			d193 = d190
			ps192.PhiValues[0] = d193
			if ps192.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps192)
			return result
			}
			argPinned194 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned194 = append(argPinned194, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned194 = append(argPinned194, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned194 = append(argPinned194, ai.Reg2)
					}
				}
			}
			ps195 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps195)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(48))
			ctx.EmitAddRSP32(int32(48))
			for _, r := range argPinned194 {
				ctx.UnprotectReg(r)
			}
			return result
		},
	})

	// Tier 3: append/grow

	Declare(&Globalenv, &Declaration{
		"append_mut", "in-place append (optimizer-only)",
		2, 1000,
		[]DeclarationParameter{
			{"list", "list", "owned base list", nil},
			{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			base := asSlice(a[0], "append_mut")
			base = append(base, a[1:]...)
			return NewSlice(base)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"append_unique_mut", "in-place append_unique (optimizer-only)",
		2, 1000,
		[]DeclarationParameter{
			{"list", "list", "owned base list", nil},
			{"item...", "any", "items to add", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			list := asSlice(a[0], "append_unique_mut")
			for _, el := range a[1:] {
				for _, el2 := range list {
					if Equal(el, el2) {
						goto skipItem
					}
				}
				list = append(list, el)
			skipItem:
			}
			return NewSlice(list)
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
	})

	Declare(&Globalenv, &Declaration{
		"merge_assoc_mut", "in-place merge_assoc (optimizer-only)",
		2, 3,
		[]DeclarationParameter{
			{"dict1", "list", "owned first dictionary", nil},
			{"dict2", "list", "dictionary with new values", nil},
			{"merge", "func", "(optional) merge function", nil},
		}, "list",
		func(a ...Scmer) Scmer {
			setAssoc := OptimizeProcToSerialFunction(Globalenv.Vars["set_assoc_mut"])
			dst := a[0]
			if slice, fd := asAssoc(a[1], "merge_assoc_mut"); fd == nil {
				for i := 0; i < len(slice); i += 2 {
					if len(a) > 2 {
						dst = setAssoc(dst, slice[i], slice[i+1], a[2])
					} else {
						dst = setAssoc(dst, slice[i], slice[i+1])
					}
				}
			} else {
				if len(a) > 2 {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v, a[2]); return true })
				} else {
					fd.Iterate(func(k, v Scmer) bool { dst = setAssoc(dst, k, v); return true })
				}
			}
			return dst
		},
		true, true, &TypeDescriptor{Return: FreshAlloc},
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
	})
}
