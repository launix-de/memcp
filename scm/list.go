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
			alloc8 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
			}
			ctx.RestoreAllocState(alloc8)
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
			d9 = args[0]
			d9.ID = 0
			var d10 JITValueDesc
			if d9.Loc == LocImm {
				slice := d9.Imm.Slice()
				d10 = JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: NewInt(int64(len(slice)))}
			} else {
				r1 := ctx.AllocReg()
				ctx.EmitMovRegReg(r1, d9.Reg2)
				ctx.EmitShrRegImm8(r1, 8)
				ctx.FreeReg(d9.Reg2)
				d10 = JITValueDesc{Loc: LocRegPair, Reg: d9.Reg, Reg2: r1}
				ctx.BindReg(d9.Reg, &d10)
				ctx.BindReg(r1, &d10)
			}
			ctx.FreeDesc(&d9)
			var d11 JITValueDesc
			if d10.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d10.StackOff))}
			} else {
				ctx.EnsureDesc(&d10)
				if d10.Loc == LocRegPair {
					d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg2}
					ctx.BindReg(d10.Reg2, &d11)
					ctx.BindReg(d10.Reg2, &d11)
				} else if d10.Loc == LocReg {
					d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d10.Reg}
					ctx.BindReg(d10.Reg, &d11)
					ctx.BindReg(d10.Reg, &d11)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d11)
			ctx.EmitMakeInt(result, d11)
			if d11.Loc == LocReg { ctx.FreeReg(d11.Reg) }
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
			ctx.ReclaimUntrackedRegs()
			d13 = args[0]
			d13.ID = 0
			d14 = ctx.EmitGetTagDesc(&d13, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			var d15 JITValueDesc
			if d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d14.Imm.Int()) == uint64(15))}
			} else {
				r2 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d14.Reg, 15)
				ctx.EmitSetcc(r2, CcE)
				d15 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d15)
			}
			ctx.FreeDesc(&d14)
			d16 = d15
			ctx.EnsureDesc(&d16)
			if d16.Loc != LocImm && d16.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d16.Loc == LocImm {
				if d16.Imm.Bool() {
			ps17 := PhiState{General: ps.General}
			ps17.OverlayValues = make([]JITValueDesc, 17)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
			ps17.OverlayValues[9] = d9
			ps17.OverlayValues[10] = d10
			ps17.OverlayValues[11] = d11
			ps17.OverlayValues[12] = d12
			ps17.OverlayValues[13] = d13
			ps17.OverlayValues[14] = d14
			ps17.OverlayValues[15] = d15
			ps17.OverlayValues[16] = d16
					return bbs[3].RenderPS(ps17)
				}
			ps18 := PhiState{General: ps.General}
			ps18.OverlayValues = make([]JITValueDesc, 17)
			ps18.OverlayValues[0] = d0
			ps18.OverlayValues[1] = d1
			ps18.OverlayValues[2] = d2
			ps18.OverlayValues[3] = d3
			ps18.OverlayValues[9] = d9
			ps18.OverlayValues[10] = d10
			ps18.OverlayValues[11] = d11
			ps18.OverlayValues[12] = d12
			ps18.OverlayValues[13] = d13
			ps18.OverlayValues[14] = d14
			ps18.OverlayValues[15] = d15
			ps18.OverlayValues[16] = d16
				return bbs[4].RenderPS(ps18)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d16.Reg, 0)
			ctx.EmitJcc(CcNE, lbl8)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl8)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl5)
			ps19 := PhiState{General: true}
			ps19.OverlayValues = make([]JITValueDesc, 17)
			ps19.OverlayValues[0] = d0
			ps19.OverlayValues[1] = d1
			ps19.OverlayValues[2] = d2
			ps19.OverlayValues[3] = d3
			ps19.OverlayValues[9] = d9
			ps19.OverlayValues[10] = d10
			ps19.OverlayValues[11] = d11
			ps19.OverlayValues[12] = d12
			ps19.OverlayValues[13] = d13
			ps19.OverlayValues[14] = d14
			ps19.OverlayValues[15] = d15
			ps19.OverlayValues[16] = d16
			ps20 := PhiState{General: true}
			ps20.OverlayValues = make([]JITValueDesc, 17)
			ps20.OverlayValues[0] = d0
			ps20.OverlayValues[1] = d1
			ps20.OverlayValues[2] = d2
			ps20.OverlayValues[3] = d3
			ps20.OverlayValues[9] = d9
			ps20.OverlayValues[10] = d10
			ps20.OverlayValues[11] = d11
			ps20.OverlayValues[12] = d12
			ps20.OverlayValues[13] = d13
			ps20.OverlayValues[14] = d14
			ps20.OverlayValues[15] = d15
			ps20.OverlayValues[16] = d16
			alloc21 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps20)
			}
			ctx.RestoreAllocState(alloc21)
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps19)
			}
			return result
			ctx.FreeDesc(&d15)
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
			ctx.ReclaimUntrackedRegs()
			d22 = args[0]
			d22.ID = 0
			var d23 JITValueDesc
			if d22.Loc == LocImm {
				panic("FastDict: LocImm not expected at JIT compile time")
			} else {
				ctx.FreeReg(d22.Reg2)
				d23 = JITValueDesc{Loc: LocReg, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d23)
			}
			ctx.FreeDesc(&d22)
			var d24 JITValueDesc
			ctx.EnsureDesc(&d23)
			if d23.Loc == LocImm {
				fieldAddr := uintptr(d23.Imm.Int()) + 0
				r3 := ctx.AllocReg()
				r4 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r3, fieldAddr)
				ctx.EmitMovRegMem64(r4, fieldAddr+8)
				d24 = JITValueDesc{Loc: LocRegPair, Reg: r3, Reg2: r4}
				ctx.BindReg(r3, &d24)
				ctx.BindReg(r4, &d24)
			} else {
				off := int32(0)
				baseReg := d23.Reg
				r5 := ctx.AllocRegExcept(baseReg)
				r6 := ctx.AllocRegExcept(baseReg, r5)
				ctx.EmitMovRegMem(r5, baseReg, off)
				ctx.EmitMovRegMem(r6, baseReg, off+8)
				d24 = JITValueDesc{Loc: LocRegPair, Reg: r5, Reg2: r6}
				ctx.BindReg(r5, &d24)
				ctx.BindReg(r6, &d24)
			}
			ctx.FreeDesc(&d23)
			var d25 JITValueDesc
			if d24.Loc == LocImm {
				d25 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d24.StackOff))}
			} else {
				ctx.EnsureDesc(&d24)
				if d24.Loc == LocRegPair {
					d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg2}
					ctx.BindReg(d24.Reg2, &d25)
					ctx.BindReg(d24.Reg2, &d25)
				} else if d24.Loc == LocReg {
					d25 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d24.Reg}
					ctx.BindReg(d24.Reg, &d25)
					ctx.BindReg(d24.Reg, &d25)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d25)
			ctx.EmitMakeInt(result, d25)
			if d25.Loc == LocReg { ctx.FreeReg(d25.Reg) }
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
			ctx.ReclaimUntrackedRegs()
			ctx.EmitByte(0xCC)
			return result
			}
			ps27 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps27)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
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
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */ /* TODO: IndexAddr on non-parameter: &t14[0:int] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
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
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
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
		nil /* TODO: phi edge references unknown value: parameter a : []Scmer */, /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */ /* TODO: phi edge references unknown value: parameter a : []Scmer */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: Slice on non-desc: slice t0[:0:int] */, /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */ /* TODO: Slice on non-desc: slice t0[:0:int] */
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
		nil /* TODO: MakeSlice: make []Scmer t5 t5 */, /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */ /* TODO: MakeSlice: make []Scmer t5 t5 */
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
			ps3 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps3)
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: Slice on non-desc: slice t1[:0:int] */, /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */ /* TODO: Slice on non-desc: slice t1[:0:int] */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
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
		nil /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t12[0:int] (x=t12 marker="_alloc" isDesc=false goVar=) */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
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
		nil /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t14[0:int] (x=t14 marker="_alloc" isDesc=false goVar=) */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t13[0:int] (x=t13 marker="_alloc" isDesc=false goVar=) */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */ /* TODO: unsupported builtin: SliceData */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t7[0:int] (x=t7 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
			var d23 JITValueDesc
			_ = d23
			var d29 JITValueDesc
			_ = d29
			var d30 JITValueDesc
			_ = d30
			var d31 JITValueDesc
			_ = d31
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
			var d40 JITValueDesc
			_ = d40
			var d41 JITValueDesc
			_ = d41
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
			var d54 JITValueDesc
			_ = d54
			var d55 JITValueDesc
			_ = d55
			var d56 JITValueDesc
			_ = d56
			var d57 JITValueDesc
			_ = d57
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
			var d66 JITValueDesc
			_ = d66
			var d68 JITValueDesc
			_ = d68
			var d69 JITValueDesc
			_ = d69
			var d72 JITValueDesc
			_ = d72
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
			var d84 JITValueDesc
			_ = d84
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			var bbs [10]BBDescriptor
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
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
			alloc12 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps10)
			}
			ctx.RestoreAllocState(alloc12)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			d13 = args[3]
			d13.ID = 0
			d14 = ctx.EmitGoCallScalar(GoFuncAddr(OptimizeProcToSerialFunction), []JITValueDesc{d13}, 1)
			ctx.FreeDesc(&d13)
			ctx.FreeDesc(&d14)
			d15 = ctx.EmitGoCallScalar(GoFuncAddr(JITBuildMergeClosure), []JITValueDesc{d14}, 1)
			d16 = d15
			if d16.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, 0)
			ps17 := PhiState{General: ps.General}
			ps17.OverlayValues = make([]JITValueDesc, 17)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
			ps17.OverlayValues[4] = d4
			ps17.OverlayValues[5] = d5
			ps17.OverlayValues[8] = d8
			ps17.OverlayValues[11] = d11
			ps17.OverlayValues[13] = d13
			ps17.OverlayValues[14] = d14
			ps17.OverlayValues[15] = d15
			ps17.OverlayValues[16] = d16
			ps17.PhiValues = make([]JITValueDesc, 1)
			d18 = d15
			ps17.PhiValues[0] = d18
			if ps17.General && bbs[2].Rendered {
				ctx.EmitJmp(lbl3)
				return result
			}
			return bbs[2].RenderPS(ps17)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d19 := ps.PhiValues[0]
						ctx.EnsureDesc(&d19)
						ctx.EmitStoreToStack(d19, 0)
					}
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d20 = args[0]
			d20.ID = 0
			d22 = d20
			d22.ID = 0
			d21 = ctx.EmitTagEqualsBorrowed(&d22, tagFastDict, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d20)
			d23 = d21
			ctx.EnsureDesc(&d23)
			if d23.Loc != LocImm && d23.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d23.Loc == LocImm {
				if d23.Imm.Bool() {
			ps24 := PhiState{General: ps.General}
			ps24.OverlayValues = make([]JITValueDesc, 24)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[3] = d3
			ps24.OverlayValues[4] = d4
			ps24.OverlayValues[5] = d5
			ps24.OverlayValues[8] = d8
			ps24.OverlayValues[11] = d11
			ps24.OverlayValues[13] = d13
			ps24.OverlayValues[14] = d14
			ps24.OverlayValues[15] = d15
			ps24.OverlayValues[16] = d16
			ps24.OverlayValues[18] = d18
			ps24.OverlayValues[19] = d19
			ps24.OverlayValues[20] = d20
			ps24.OverlayValues[21] = d21
			ps24.OverlayValues[22] = d22
			ps24.OverlayValues[23] = d23
					return bbs[3].RenderPS(ps24)
				}
			ps25 := PhiState{General: ps.General}
			ps25.OverlayValues = make([]JITValueDesc, 24)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[8] = d8
			ps25.OverlayValues[11] = d11
			ps25.OverlayValues[13] = d13
			ps25.OverlayValues[14] = d14
			ps25.OverlayValues[15] = d15
			ps25.OverlayValues[16] = d16
			ps25.OverlayValues[18] = d18
			ps25.OverlayValues[19] = d19
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps25.OverlayValues[23] = d23
				return bbs[5].RenderPS(ps25)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d23.Reg, 0)
			ctx.EmitJcc(CcNE, lbl13)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl6)
			ps26 := PhiState{General: true}
			ps26.OverlayValues = make([]JITValueDesc, 24)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[8] = d8
			ps26.OverlayValues[11] = d11
			ps26.OverlayValues[13] = d13
			ps26.OverlayValues[14] = d14
			ps26.OverlayValues[15] = d15
			ps26.OverlayValues[16] = d16
			ps26.OverlayValues[18] = d18
			ps26.OverlayValues[19] = d19
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[22] = d22
			ps26.OverlayValues[23] = d23
			ps27 := PhiState{General: true}
			ps27.OverlayValues = make([]JITValueDesc, 24)
			ps27.OverlayValues[0] = d0
			ps27.OverlayValues[1] = d1
			ps27.OverlayValues[2] = d2
			ps27.OverlayValues[3] = d3
			ps27.OverlayValues[4] = d4
			ps27.OverlayValues[5] = d5
			ps27.OverlayValues[8] = d8
			ps27.OverlayValues[11] = d11
			ps27.OverlayValues[13] = d13
			ps27.OverlayValues[14] = d14
			ps27.OverlayValues[15] = d15
			ps27.OverlayValues[16] = d16
			ps27.OverlayValues[18] = d18
			ps27.OverlayValues[19] = d19
			ps27.OverlayValues[20] = d20
			ps27.OverlayValues[21] = d21
			ps27.OverlayValues[22] = d22
			ps27.OverlayValues[23] = d23
			alloc28 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps27)
			}
			ctx.RestoreAllocState(alloc28)
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps26)
			}
			return result
			ctx.FreeDesc(&d21)
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
			}
			ctx.ReclaimUntrackedRegs()
			d29 = args[0]
			d29.ID = 0
			var d30 JITValueDesc
			if d29.Loc == LocImm {
				panic("FastDict: LocImm not expected at JIT compile time")
			} else {
				ctx.FreeReg(d29.Reg2)
				d30 = JITValueDesc{Loc: LocReg, Reg: d29.Reg}
				ctx.BindReg(d29.Reg, &d30)
			}
			ctx.FreeDesc(&d29)
			d31 = d30
			if d31.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d31)
			ctx.EmitStoreToStack(d31, 8)
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
			ps32.OverlayValues[13] = d13
			ps32.OverlayValues[14] = d14
			ps32.OverlayValues[15] = d15
			ps32.OverlayValues[16] = d16
			ps32.OverlayValues[18] = d18
			ps32.OverlayValues[19] = d19
			ps32.OverlayValues[20] = d20
			ps32.OverlayValues[21] = d21
			ps32.OverlayValues[22] = d22
			ps32.OverlayValues[23] = d23
			ps32.OverlayValues[29] = d29
			ps32.OverlayValues[30] = d30
			ps32.OverlayValues[31] = d31
			ps32.PhiValues = make([]JITValueDesc, 1)
			d33 = d30
			ps32.PhiValues[0] = d33
			if ps32.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps32)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d34 := ps.PhiValues[0]
						ctx.EnsureDesc(&d34)
						ctx.EmitStoreToStack(d34, 8)
					}
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
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
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d35 = args[1]
			d35.ID = 0
			d36 = args[2]
			d36.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d1, d35, d36, d0})
			ctx.FreeDesc(&d35)
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d0)
			var d37 JITValueDesc
			if d1.Loc == LocImm {
				panic("NewFastDict: LocImm not expected at JIT compile time")
			} else {
				r2 := ctx.AllocReg()
				ctx.EmitMovRegImm64(r2, makeAux(tagFastDict, 0))
				d37 = JITValueDesc{Loc: LocRegPair, Type: tagFastDict, Reg: d1.Reg, Reg2: r2}
				ctx.BindReg(d1.Reg, &d37)
				ctx.BindReg(r2, &d37)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d37)
			if d37.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d37, &result)
				result.Type = d37.Type
			} else {
				switch d37.Type {
				case tagBool:
					ctx.EmitMakeBool(result, d37)
					result.Type = tagBool
				case tagInt:
					ctx.EmitMakeInt(result, d37)
					result.Type = tagInt
				case tagFloat:
					ctx.EmitMakeFloat(result, d37)
					result.Type = tagFloat
				case tagNil:
					ctx.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d37, &result)
					result.Type = d37.Type
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
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
			ctx.ReclaimUntrackedRegs()
			d38 = args[0]
			d38.ID = 0
			d40 = d38
			d40.ID = 0
			d39 = ctx.EmitTagEqualsBorrowed(&d40, tagSlice, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d38)
			d41 = d39
			ctx.EnsureDesc(&d41)
			if d41.Loc != LocImm && d41.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d41.Loc == LocImm {
				if d41.Imm.Bool() {
			ps42 := PhiState{General: ps.General}
			ps42.OverlayValues = make([]JITValueDesc, 42)
			ps42.OverlayValues[0] = d0
			ps42.OverlayValues[1] = d1
			ps42.OverlayValues[2] = d2
			ps42.OverlayValues[3] = d3
			ps42.OverlayValues[4] = d4
			ps42.OverlayValues[5] = d5
			ps42.OverlayValues[8] = d8
			ps42.OverlayValues[11] = d11
			ps42.OverlayValues[13] = d13
			ps42.OverlayValues[14] = d14
			ps42.OverlayValues[15] = d15
			ps42.OverlayValues[16] = d16
			ps42.OverlayValues[18] = d18
			ps42.OverlayValues[19] = d19
			ps42.OverlayValues[20] = d20
			ps42.OverlayValues[21] = d21
			ps42.OverlayValues[22] = d22
			ps42.OverlayValues[23] = d23
			ps42.OverlayValues[29] = d29
			ps42.OverlayValues[30] = d30
			ps42.OverlayValues[31] = d31
			ps42.OverlayValues[33] = d33
			ps42.OverlayValues[34] = d34
			ps42.OverlayValues[35] = d35
			ps42.OverlayValues[36] = d36
			ps42.OverlayValues[37] = d37
			ps42.OverlayValues[38] = d38
			ps42.OverlayValues[39] = d39
			ps42.OverlayValues[40] = d40
			ps42.OverlayValues[41] = d41
					return bbs[6].RenderPS(ps42)
				}
			ps43 := PhiState{General: ps.General}
			ps43.OverlayValues = make([]JITValueDesc, 42)
			ps43.OverlayValues[0] = d0
			ps43.OverlayValues[1] = d1
			ps43.OverlayValues[2] = d2
			ps43.OverlayValues[3] = d3
			ps43.OverlayValues[4] = d4
			ps43.OverlayValues[5] = d5
			ps43.OverlayValues[8] = d8
			ps43.OverlayValues[11] = d11
			ps43.OverlayValues[13] = d13
			ps43.OverlayValues[14] = d14
			ps43.OverlayValues[15] = d15
			ps43.OverlayValues[16] = d16
			ps43.OverlayValues[18] = d18
			ps43.OverlayValues[19] = d19
			ps43.OverlayValues[20] = d20
			ps43.OverlayValues[21] = d21
			ps43.OverlayValues[22] = d22
			ps43.OverlayValues[23] = d23
			ps43.OverlayValues[29] = d29
			ps43.OverlayValues[30] = d30
			ps43.OverlayValues[31] = d31
			ps43.OverlayValues[33] = d33
			ps43.OverlayValues[34] = d34
			ps43.OverlayValues[35] = d35
			ps43.OverlayValues[36] = d36
			ps43.OverlayValues[37] = d37
			ps43.OverlayValues[38] = d38
			ps43.OverlayValues[39] = d39
			ps43.OverlayValues[40] = d40
			ps43.OverlayValues[41] = d41
				return bbs[7].RenderPS(ps43)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d41.Reg, 0)
			ctx.EmitJcc(CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl8)
			ps44 := PhiState{General: true}
			ps44.OverlayValues = make([]JITValueDesc, 42)
			ps44.OverlayValues[0] = d0
			ps44.OverlayValues[1] = d1
			ps44.OverlayValues[2] = d2
			ps44.OverlayValues[3] = d3
			ps44.OverlayValues[4] = d4
			ps44.OverlayValues[5] = d5
			ps44.OverlayValues[8] = d8
			ps44.OverlayValues[11] = d11
			ps44.OverlayValues[13] = d13
			ps44.OverlayValues[14] = d14
			ps44.OverlayValues[15] = d15
			ps44.OverlayValues[16] = d16
			ps44.OverlayValues[18] = d18
			ps44.OverlayValues[19] = d19
			ps44.OverlayValues[20] = d20
			ps44.OverlayValues[21] = d21
			ps44.OverlayValues[22] = d22
			ps44.OverlayValues[23] = d23
			ps44.OverlayValues[29] = d29
			ps44.OverlayValues[30] = d30
			ps44.OverlayValues[31] = d31
			ps44.OverlayValues[33] = d33
			ps44.OverlayValues[34] = d34
			ps44.OverlayValues[35] = d35
			ps44.OverlayValues[36] = d36
			ps44.OverlayValues[37] = d37
			ps44.OverlayValues[38] = d38
			ps44.OverlayValues[39] = d39
			ps44.OverlayValues[40] = d40
			ps44.OverlayValues[41] = d41
			ps45 := PhiState{General: true}
			ps45.OverlayValues = make([]JITValueDesc, 42)
			ps45.OverlayValues[0] = d0
			ps45.OverlayValues[1] = d1
			ps45.OverlayValues[2] = d2
			ps45.OverlayValues[3] = d3
			ps45.OverlayValues[4] = d4
			ps45.OverlayValues[5] = d5
			ps45.OverlayValues[8] = d8
			ps45.OverlayValues[11] = d11
			ps45.OverlayValues[13] = d13
			ps45.OverlayValues[14] = d14
			ps45.OverlayValues[15] = d15
			ps45.OverlayValues[16] = d16
			ps45.OverlayValues[18] = d18
			ps45.OverlayValues[19] = d19
			ps45.OverlayValues[20] = d20
			ps45.OverlayValues[21] = d21
			ps45.OverlayValues[22] = d22
			ps45.OverlayValues[23] = d23
			ps45.OverlayValues[29] = d29
			ps45.OverlayValues[30] = d30
			ps45.OverlayValues[31] = d31
			ps45.OverlayValues[33] = d33
			ps45.OverlayValues[34] = d34
			ps45.OverlayValues[35] = d35
			ps45.OverlayValues[36] = d36
			ps45.OverlayValues[37] = d37
			ps45.OverlayValues[38] = d38
			ps45.OverlayValues[39] = d39
			ps45.OverlayValues[40] = d40
			ps45.OverlayValues[41] = d41
			alloc46 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps45)
			}
			ctx.RestoreAllocState(alloc46)
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps44)
			}
			return result
			ctx.FreeDesc(&d39)
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
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
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
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			ctx.ReclaimUntrackedRegs()
			d47 = args[0]
			d47.ID = 0
			var d48 JITValueDesc
			if d47.Loc == LocImm {
				slice := d47.Imm.Slice()
				d48 = JITValueDesc{Loc: LocImm, Type: tagSlice, Imm: NewInt(int64(len(slice)))}
			} else {
				r3 := ctx.AllocReg()
				ctx.EmitMovRegReg(r3, d47.Reg2)
				ctx.EmitShrRegImm8(r3, 8)
				ctx.FreeReg(d47.Reg2)
				d48 = JITValueDesc{Loc: LocRegPair, Reg: d47.Reg, Reg2: r3}
				ctx.BindReg(d47.Reg, &d48)
				ctx.BindReg(r3, &d48)
			}
			ctx.FreeDesc(&d47)
			var d49 JITValueDesc
			if d48.Loc == LocImm {
				d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d48.StackOff))}
			} else {
				ctx.EnsureDesc(&d48)
				if d48.Loc == LocRegPair {
					d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg2}
					ctx.BindReg(d48.Reg2, &d49)
					ctx.BindReg(d48.Reg2, &d49)
				} else if d48.Loc == LocReg {
					d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg}
					ctx.BindReg(d48.Reg, &d49)
					ctx.BindReg(d48.Reg, &d49)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d49)
			var d50 JITValueDesc
			if d49.Loc == LocImm {
				d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d49.Imm.Int() / 2)}
			} else {
				ctx.EmitShrRegImm8(d49.Reg, 1)
				d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d49.Reg}
				ctx.BindReg(d49.Reg, &d50)
			}
			if d50.Loc == LocReg && d49.Loc == LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d50)
			var d51 JITValueDesc
			if d50.Loc == LocImm {
				d51 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d50.Imm.Int() + 4)}
			} else {
				scratch := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(scratch, d50.Reg)
				ctx.EmitAddRegImm32(scratch, int32(4))
				d51 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d51)
			}
			if d51.Loc == LocReg && d50.Loc == LocReg && d51.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = LocNone
			}
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d51)
			d52 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d51}, 1)
			ctx.FreeDesc(&d51)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 16)
			ps53 := PhiState{General: ps.General}
			ps53.OverlayValues = make([]JITValueDesc, 53)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
			ps53.OverlayValues[2] = d2
			ps53.OverlayValues[3] = d3
			ps53.OverlayValues[4] = d4
			ps53.OverlayValues[5] = d5
			ps53.OverlayValues[8] = d8
			ps53.OverlayValues[11] = d11
			ps53.OverlayValues[13] = d13
			ps53.OverlayValues[14] = d14
			ps53.OverlayValues[15] = d15
			ps53.OverlayValues[16] = d16
			ps53.OverlayValues[18] = d18
			ps53.OverlayValues[19] = d19
			ps53.OverlayValues[20] = d20
			ps53.OverlayValues[21] = d21
			ps53.OverlayValues[22] = d22
			ps53.OverlayValues[23] = d23
			ps53.OverlayValues[29] = d29
			ps53.OverlayValues[30] = d30
			ps53.OverlayValues[31] = d31
			ps53.OverlayValues[33] = d33
			ps53.OverlayValues[34] = d34
			ps53.OverlayValues[35] = d35
			ps53.OverlayValues[36] = d36
			ps53.OverlayValues[37] = d37
			ps53.OverlayValues[38] = d38
			ps53.OverlayValues[39] = d39
			ps53.OverlayValues[40] = d40
			ps53.OverlayValues[41] = d41
			ps53.OverlayValues[47] = d47
			ps53.OverlayValues[48] = d48
			ps53.OverlayValues[49] = d49
			ps53.OverlayValues[50] = d50
			ps53.OverlayValues[51] = d51
			ps53.OverlayValues[52] = d52
			ps53.PhiValues = make([]JITValueDesc, 1)
			d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps53.PhiValues[0] = d54
			if ps53.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps53)
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
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
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
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
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
				d54 = ps.OverlayValues[54]
			}
			ctx.ReclaimUntrackedRegs()
			d55 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(8)}
			d56 = ctx.EmitGoCallScalar(GoFuncAddr(NewFastDictValue), []JITValueDesc{d55}, 1)
			d57 = d56
			if d57.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			ctx.EmitStoreToStack(d57, 8)
			ps58 := PhiState{General: ps.General}
			ps58.OverlayValues = make([]JITValueDesc, 58)
			ps58.OverlayValues[0] = d0
			ps58.OverlayValues[1] = d1
			ps58.OverlayValues[2] = d2
			ps58.OverlayValues[3] = d3
			ps58.OverlayValues[4] = d4
			ps58.OverlayValues[5] = d5
			ps58.OverlayValues[8] = d8
			ps58.OverlayValues[11] = d11
			ps58.OverlayValues[13] = d13
			ps58.OverlayValues[14] = d14
			ps58.OverlayValues[15] = d15
			ps58.OverlayValues[16] = d16
			ps58.OverlayValues[18] = d18
			ps58.OverlayValues[19] = d19
			ps58.OverlayValues[20] = d20
			ps58.OverlayValues[21] = d21
			ps58.OverlayValues[22] = d22
			ps58.OverlayValues[23] = d23
			ps58.OverlayValues[29] = d29
			ps58.OverlayValues[30] = d30
			ps58.OverlayValues[31] = d31
			ps58.OverlayValues[33] = d33
			ps58.OverlayValues[34] = d34
			ps58.OverlayValues[35] = d35
			ps58.OverlayValues[36] = d36
			ps58.OverlayValues[37] = d37
			ps58.OverlayValues[38] = d38
			ps58.OverlayValues[39] = d39
			ps58.OverlayValues[40] = d40
			ps58.OverlayValues[41] = d41
			ps58.OverlayValues[47] = d47
			ps58.OverlayValues[48] = d48
			ps58.OverlayValues[49] = d49
			ps58.OverlayValues[50] = d50
			ps58.OverlayValues[51] = d51
			ps58.OverlayValues[52] = d52
			ps58.OverlayValues[54] = d54
			ps58.OverlayValues[55] = d55
			ps58.OverlayValues[56] = d56
			ps58.OverlayValues[57] = d57
			ps58.PhiValues = make([]JITValueDesc, 1)
			d59 = d56
			ps58.PhiValues[0] = d59
			if ps58.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps58)
			return result
			}
			bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[8].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d60 := ps.PhiValues[0]
						ctx.EnsureDesc(&d60)
						ctx.EmitStoreToStack(d60, 16)
					}
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
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
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
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
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
				d60 = ps.OverlayValues[60]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d2 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d61 JITValueDesc
			if d2.Loc == LocImm {
				d61 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d61 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			}
			if d61.Loc == LocReg && d2.Loc == LocReg && d61.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			var d62 JITValueDesc
			if d48.Loc == LocImm {
				d62 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d48.StackOff))}
			} else {
				ctx.EnsureDesc(&d48)
				if d48.Loc == LocRegPair {
					d62 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg2}
					ctx.BindReg(d48.Reg2, &d62)
					ctx.BindReg(d48.Reg2, &d62)
				} else if d48.Loc == LocReg {
					d62 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d48.Reg}
					ctx.BindReg(d48.Reg, &d62)
					ctx.BindReg(d48.Reg, &d62)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d62)
			var d63 JITValueDesc
			if d61.Loc == LocImm && d62.Loc == LocImm {
				d63 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d61.Imm.Int() < d62.Imm.Int())}
			} else if d62.Loc == LocImm {
				r4 := ctx.AllocReg()
				if d62.Imm.Int() >= -2147483648 && d62.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d61.Reg, int32(d62.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d62.Imm.Int()))
					ctx.EmitCmpInt64(d61.Reg, RegR11)
				}
				ctx.EmitSetcc(r4, CcL)
				d63 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d63)
			} else if d61.Loc == LocImm {
				r5 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d61.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d62.Reg)
				ctx.EmitSetcc(r5, CcL)
				d63 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d63)
			} else {
				r6 := ctx.AllocReg()
				ctx.EmitCmpInt64(d61.Reg, d62.Reg)
				ctx.EmitSetcc(r6, CcL)
				d63 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d63)
			}
			ctx.FreeDesc(&d61)
			ctx.FreeDesc(&d62)
			d64 = d63
			ctx.EnsureDesc(&d64)
			if d64.Loc != LocImm && d64.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d64.Loc == LocImm {
				if d64.Imm.Bool() {
			ps65 := PhiState{General: ps.General}
			ps65.OverlayValues = make([]JITValueDesc, 65)
			ps65.OverlayValues[0] = d0
			ps65.OverlayValues[1] = d1
			ps65.OverlayValues[2] = d2
			ps65.OverlayValues[3] = d3
			ps65.OverlayValues[4] = d4
			ps65.OverlayValues[5] = d5
			ps65.OverlayValues[8] = d8
			ps65.OverlayValues[11] = d11
			ps65.OverlayValues[13] = d13
			ps65.OverlayValues[14] = d14
			ps65.OverlayValues[15] = d15
			ps65.OverlayValues[16] = d16
			ps65.OverlayValues[18] = d18
			ps65.OverlayValues[19] = d19
			ps65.OverlayValues[20] = d20
			ps65.OverlayValues[21] = d21
			ps65.OverlayValues[22] = d22
			ps65.OverlayValues[23] = d23
			ps65.OverlayValues[29] = d29
			ps65.OverlayValues[30] = d30
			ps65.OverlayValues[31] = d31
			ps65.OverlayValues[33] = d33
			ps65.OverlayValues[34] = d34
			ps65.OverlayValues[35] = d35
			ps65.OverlayValues[36] = d36
			ps65.OverlayValues[37] = d37
			ps65.OverlayValues[38] = d38
			ps65.OverlayValues[39] = d39
			ps65.OverlayValues[40] = d40
			ps65.OverlayValues[41] = d41
			ps65.OverlayValues[47] = d47
			ps65.OverlayValues[48] = d48
			ps65.OverlayValues[49] = d49
			ps65.OverlayValues[50] = d50
			ps65.OverlayValues[51] = d51
			ps65.OverlayValues[52] = d52
			ps65.OverlayValues[54] = d54
			ps65.OverlayValues[55] = d55
			ps65.OverlayValues[56] = d56
			ps65.OverlayValues[57] = d57
			ps65.OverlayValues[59] = d59
			ps65.OverlayValues[60] = d60
			ps65.OverlayValues[61] = d61
			ps65.OverlayValues[62] = d62
			ps65.OverlayValues[63] = d63
			ps65.OverlayValues[64] = d64
					return bbs[9].RenderPS(ps65)
				}
			d66 = d52
			if d66.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			ctx.EmitStoreToStack(d66, 8)
			ps67 := PhiState{General: ps.General}
			ps67.OverlayValues = make([]JITValueDesc, 67)
			ps67.OverlayValues[0] = d0
			ps67.OverlayValues[1] = d1
			ps67.OverlayValues[2] = d2
			ps67.OverlayValues[3] = d3
			ps67.OverlayValues[4] = d4
			ps67.OverlayValues[5] = d5
			ps67.OverlayValues[8] = d8
			ps67.OverlayValues[11] = d11
			ps67.OverlayValues[13] = d13
			ps67.OverlayValues[14] = d14
			ps67.OverlayValues[15] = d15
			ps67.OverlayValues[16] = d16
			ps67.OverlayValues[18] = d18
			ps67.OverlayValues[19] = d19
			ps67.OverlayValues[20] = d20
			ps67.OverlayValues[21] = d21
			ps67.OverlayValues[22] = d22
			ps67.OverlayValues[23] = d23
			ps67.OverlayValues[29] = d29
			ps67.OverlayValues[30] = d30
			ps67.OverlayValues[31] = d31
			ps67.OverlayValues[33] = d33
			ps67.OverlayValues[34] = d34
			ps67.OverlayValues[35] = d35
			ps67.OverlayValues[36] = d36
			ps67.OverlayValues[37] = d37
			ps67.OverlayValues[38] = d38
			ps67.OverlayValues[39] = d39
			ps67.OverlayValues[40] = d40
			ps67.OverlayValues[41] = d41
			ps67.OverlayValues[47] = d47
			ps67.OverlayValues[48] = d48
			ps67.OverlayValues[49] = d49
			ps67.OverlayValues[50] = d50
			ps67.OverlayValues[51] = d51
			ps67.OverlayValues[52] = d52
			ps67.OverlayValues[54] = d54
			ps67.OverlayValues[55] = d55
			ps67.OverlayValues[56] = d56
			ps67.OverlayValues[57] = d57
			ps67.OverlayValues[59] = d59
			ps67.OverlayValues[60] = d60
			ps67.OverlayValues[61] = d61
			ps67.OverlayValues[62] = d62
			ps67.OverlayValues[63] = d63
			ps67.OverlayValues[64] = d64
			ps67.OverlayValues[66] = d66
			ps67.PhiValues = make([]JITValueDesc, 1)
			d68 = d52
			ps67.PhiValues[0] = d68
				return bbs[4].RenderPS(ps67)
			}
			if !ps.General {
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d64.Reg, 0)
			ctx.EmitJcc(CcNE, lbl17)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl18)
			d69 = d52
			if d69.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d69)
			ctx.EmitStoreToStack(d69, 8)
			ctx.EmitJmp(lbl5)
			ps70 := PhiState{General: true}
			ps70.OverlayValues = make([]JITValueDesc, 70)
			ps70.OverlayValues[0] = d0
			ps70.OverlayValues[1] = d1
			ps70.OverlayValues[2] = d2
			ps70.OverlayValues[3] = d3
			ps70.OverlayValues[4] = d4
			ps70.OverlayValues[5] = d5
			ps70.OverlayValues[8] = d8
			ps70.OverlayValues[11] = d11
			ps70.OverlayValues[13] = d13
			ps70.OverlayValues[14] = d14
			ps70.OverlayValues[15] = d15
			ps70.OverlayValues[16] = d16
			ps70.OverlayValues[18] = d18
			ps70.OverlayValues[19] = d19
			ps70.OverlayValues[20] = d20
			ps70.OverlayValues[21] = d21
			ps70.OverlayValues[22] = d22
			ps70.OverlayValues[23] = d23
			ps70.OverlayValues[29] = d29
			ps70.OverlayValues[30] = d30
			ps70.OverlayValues[31] = d31
			ps70.OverlayValues[33] = d33
			ps70.OverlayValues[34] = d34
			ps70.OverlayValues[35] = d35
			ps70.OverlayValues[36] = d36
			ps70.OverlayValues[37] = d37
			ps70.OverlayValues[38] = d38
			ps70.OverlayValues[39] = d39
			ps70.OverlayValues[40] = d40
			ps70.OverlayValues[41] = d41
			ps70.OverlayValues[47] = d47
			ps70.OverlayValues[48] = d48
			ps70.OverlayValues[49] = d49
			ps70.OverlayValues[50] = d50
			ps70.OverlayValues[51] = d51
			ps70.OverlayValues[52] = d52
			ps70.OverlayValues[54] = d54
			ps70.OverlayValues[55] = d55
			ps70.OverlayValues[56] = d56
			ps70.OverlayValues[57] = d57
			ps70.OverlayValues[59] = d59
			ps70.OverlayValues[60] = d60
			ps70.OverlayValues[61] = d61
			ps70.OverlayValues[62] = d62
			ps70.OverlayValues[63] = d63
			ps70.OverlayValues[64] = d64
			ps70.OverlayValues[66] = d66
			ps70.OverlayValues[68] = d68
			ps70.OverlayValues[69] = d69
			ps71 := PhiState{General: true}
			ps71.OverlayValues = make([]JITValueDesc, 70)
			ps71.OverlayValues[0] = d0
			ps71.OverlayValues[1] = d1
			ps71.OverlayValues[2] = d2
			ps71.OverlayValues[3] = d3
			ps71.OverlayValues[4] = d4
			ps71.OverlayValues[5] = d5
			ps71.OverlayValues[8] = d8
			ps71.OverlayValues[11] = d11
			ps71.OverlayValues[13] = d13
			ps71.OverlayValues[14] = d14
			ps71.OverlayValues[15] = d15
			ps71.OverlayValues[16] = d16
			ps71.OverlayValues[18] = d18
			ps71.OverlayValues[19] = d19
			ps71.OverlayValues[20] = d20
			ps71.OverlayValues[21] = d21
			ps71.OverlayValues[22] = d22
			ps71.OverlayValues[23] = d23
			ps71.OverlayValues[29] = d29
			ps71.OverlayValues[30] = d30
			ps71.OverlayValues[31] = d31
			ps71.OverlayValues[33] = d33
			ps71.OverlayValues[34] = d34
			ps71.OverlayValues[35] = d35
			ps71.OverlayValues[36] = d36
			ps71.OverlayValues[37] = d37
			ps71.OverlayValues[38] = d38
			ps71.OverlayValues[39] = d39
			ps71.OverlayValues[40] = d40
			ps71.OverlayValues[41] = d41
			ps71.OverlayValues[47] = d47
			ps71.OverlayValues[48] = d48
			ps71.OverlayValues[49] = d49
			ps71.OverlayValues[50] = d50
			ps71.OverlayValues[51] = d51
			ps71.OverlayValues[52] = d52
			ps71.OverlayValues[54] = d54
			ps71.OverlayValues[55] = d55
			ps71.OverlayValues[56] = d56
			ps71.OverlayValues[57] = d57
			ps71.OverlayValues[59] = d59
			ps71.OverlayValues[60] = d60
			ps71.OverlayValues[61] = d61
			ps71.OverlayValues[62] = d62
			ps71.OverlayValues[63] = d63
			ps71.OverlayValues[64] = d64
			ps71.OverlayValues[66] = d66
			ps71.OverlayValues[68] = d68
			ps71.OverlayValues[69] = d69
			ps71.PhiValues = make([]JITValueDesc, 1)
			d72 = d52
			ps71.PhiValues[0] = d72
			snap73 := d2
			snap74 := d48
			snap75 := d52
			alloc76 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps71)
			}
			ctx.RestoreAllocState(alloc76)
			d2 = snap73
			d48 = snap74
			d52 = snap75
			if !bbs[9].Rendered {
				return bbs[9].RenderPS(ps70)
			}
			return result
			ctx.FreeDesc(&d63)
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
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
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
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
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
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
				d72 = ps.OverlayValues[72]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			r7 := ctx.AllocReg()
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d48)
			if d2.Loc == LocImm {
				ctx.EmitMovRegImm64(r7, uint64(d2.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r7, d2.Reg)
				ctx.EmitShlRegImm8(r7, 4)
			}
			if d48.Loc == LocImm {
				ctx.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()))
				ctx.EmitAddInt64(r7, RegR11)
			} else {
				ctx.EmitAddInt64(r7, d48.Reg)
			}
			r8 := ctx.AllocRegExcept(r7)
			r9 := ctx.AllocRegExcept(r7, r8)
			ctx.EmitMovRegMem(r8, r7, 0)
			ctx.EmitMovRegMem(r9, r7, 8)
			ctx.FreeReg(r7)
			d77 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r8, Reg2: r9}
			ctx.BindReg(r8, &d77)
			ctx.BindReg(r9, &d77)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d78 JITValueDesc
			if d2.Loc == LocImm {
				d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			}
			if d78.Loc == LocReg && d2.Loc == LocReg && d78.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.EnsureDesc(&d78)
			r10 := ctx.AllocReg()
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d48)
			if d78.Loc == LocImm {
				ctx.EmitMovRegImm64(r10, uint64(d78.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r10, d78.Reg)
				ctx.EmitShlRegImm8(r10, 4)
			}
			if d48.Loc == LocImm {
				ctx.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()))
				ctx.EmitAddInt64(r10, RegR11)
			} else {
				ctx.EmitAddInt64(r10, d48.Reg)
			}
			r11 := ctx.AllocRegExcept(r10)
			r12 := ctx.AllocRegExcept(r10, r11)
			ctx.EmitMovRegMem(r11, r10, 0)
			ctx.EmitMovRegMem(r12, r10, 8)
			ctx.FreeReg(r10)
			d79 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r11, Reg2: r12}
			ctx.BindReg(r11, &d79)
			ctx.BindReg(r12, &d79)
			ctx.FreeDesc(&d78)
			d80 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			ctx.EmitGoCallVoid(GoFuncAddr((*FastDict).Set), []JITValueDesc{d52, d77, d79, d80})
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d79)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d81 JITValueDesc
			if d2.Loc == LocImm {
				d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 2)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(2))
				d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d81)
			}
			if d81.Loc == LocReg && d2.Loc == LocReg && d81.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.FreeDesc(&d2)
			d82 = d81
			if d82.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d82)
			ctx.EmitStoreToStack(d82, 16)
			ps83 := PhiState{General: ps.General}
			ps83.OverlayValues = make([]JITValueDesc, 83)
			ps83.OverlayValues[0] = d0
			ps83.OverlayValues[1] = d1
			ps83.OverlayValues[2] = d2
			ps83.OverlayValues[3] = d3
			ps83.OverlayValues[4] = d4
			ps83.OverlayValues[5] = d5
			ps83.OverlayValues[8] = d8
			ps83.OverlayValues[11] = d11
			ps83.OverlayValues[13] = d13
			ps83.OverlayValues[14] = d14
			ps83.OverlayValues[15] = d15
			ps83.OverlayValues[16] = d16
			ps83.OverlayValues[18] = d18
			ps83.OverlayValues[19] = d19
			ps83.OverlayValues[20] = d20
			ps83.OverlayValues[21] = d21
			ps83.OverlayValues[22] = d22
			ps83.OverlayValues[23] = d23
			ps83.OverlayValues[29] = d29
			ps83.OverlayValues[30] = d30
			ps83.OverlayValues[31] = d31
			ps83.OverlayValues[33] = d33
			ps83.OverlayValues[34] = d34
			ps83.OverlayValues[35] = d35
			ps83.OverlayValues[36] = d36
			ps83.OverlayValues[37] = d37
			ps83.OverlayValues[38] = d38
			ps83.OverlayValues[39] = d39
			ps83.OverlayValues[40] = d40
			ps83.OverlayValues[41] = d41
			ps83.OverlayValues[47] = d47
			ps83.OverlayValues[48] = d48
			ps83.OverlayValues[49] = d49
			ps83.OverlayValues[50] = d50
			ps83.OverlayValues[51] = d51
			ps83.OverlayValues[52] = d52
			ps83.OverlayValues[54] = d54
			ps83.OverlayValues[55] = d55
			ps83.OverlayValues[56] = d56
			ps83.OverlayValues[57] = d57
			ps83.OverlayValues[59] = d59
			ps83.OverlayValues[60] = d60
			ps83.OverlayValues[61] = d61
			ps83.OverlayValues[62] = d62
			ps83.OverlayValues[63] = d63
			ps83.OverlayValues[64] = d64
			ps83.OverlayValues[66] = d66
			ps83.OverlayValues[68] = d68
			ps83.OverlayValues[69] = d69
			ps83.OverlayValues[72] = d72
			ps83.OverlayValues[77] = d77
			ps83.OverlayValues[78] = d78
			ps83.OverlayValues[79] = d79
			ps83.OverlayValues[80] = d80
			ps83.OverlayValues[81] = d81
			ps83.OverlayValues[82] = d82
			ps83.PhiValues = make([]JITValueDesc, 1)
			d84 = d81
			ps83.PhiValues[0] = d84
			if ps83.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps83)
			return result
			}
			ps85 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps85)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(24))
			ctx.EmitAddRSP32(int32(24))
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */, /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: IndexAddr on non-parameter: &t6[0:int] (x=t6 marker="_alloc" isDesc=false goVar=) */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */ /* TODO: Slice on non-desc: slice t1[:] */
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
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
	})
}
