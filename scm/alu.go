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

import (
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"strings"
)

func init_alu() {
	// string functions
	DeclareTitle("Arithmetic / Logic")

	Declare(&Globalenv, &Declaration{
		"int?", "tells if the value is a integer",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].GetTag() == tagInt)
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
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(4))}
			} else {
				r0 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d1.Reg, 4)
				ctx.W.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps3 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps3)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"number?", "tells if the value is a number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			tag := a[0].GetTag()
			return NewBool(tag == tagFloat || tag == tagInt || tag == tagDate)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
			var d6 JITValueDesc
			_ = d6
			var d10 JITValueDesc
			_ = d10
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
			var d21 JITValueDesc
			_ = d21
			var d25 JITValueDesc
			_ = d25
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			var bbs [4]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d1 = args[0]
			d1.ID = 0
			d2 = ctx.EmitGetTagDesc(&d1, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(3))}
			} else {
				r1 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpRegImm32(d2.Reg, 3)
				ctx.W.EmitSetcc(r1, CcE)
				d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d3)
			}
			d4 = d3
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocImm && d4.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 5)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
			ps5.OverlayValues[4] = d4
			ps5.PhiValues = make([]JITValueDesc, 1)
			d6 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			ps5.PhiValues[0] = d6
					return bbs[2].RenderPS(ps5)
				}
			ps7 := PhiState{General: ps.General}
			ps7.OverlayValues = make([]JITValueDesc, 7)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			ps7.OverlayValues[4] = d4
			ps7.OverlayValues[6] = d6
				return bbs[3].RenderPS(ps7)
			}
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d4.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl5)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl5)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl6)
			ctx.W.EmitJmp(lbl4)
			ps8 := PhiState{General: true}
			ps8.OverlayValues = make([]JITValueDesc, 7)
			ps8.OverlayValues[0] = d0
			ps8.OverlayValues[1] = d1
			ps8.OverlayValues[2] = d2
			ps8.OverlayValues[3] = d3
			ps8.OverlayValues[4] = d4
			ps8.OverlayValues[6] = d6
			ps8.PhiValues = make([]JITValueDesc, 1)
			d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			ps8.PhiValues[0] = d10
			ps9 := PhiState{General: true}
			ps9.OverlayValues = make([]JITValueDesc, 11)
			ps9.OverlayValues[0] = d0
			ps9.OverlayValues[1] = d1
			ps9.OverlayValues[2] = d2
			ps9.OverlayValues[3] = d3
			ps9.OverlayValues[4] = d4
			ps9.OverlayValues[6] = d6
			ps9.OverlayValues[10] = d10
			snap11 := d2
			alloc12 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps8)
			}
			ctx.RestoreAllocState(alloc12)
			d2 = snap11
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps9)
			}
			return result
			ctx.FreeDesc(&d3)
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
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d13 JITValueDesc
			if d2.Loc == LocImm {
				d13 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(15))}
			} else {
				r2 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpRegImm32(d2.Reg, 15)
				ctx.W.EmitSetcc(r2, CcE)
				d13 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d13)
			}
			d14 = d13
			if d14.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 0)
			ps15 := PhiState{General: ps.General}
			ps15.OverlayValues = make([]JITValueDesc, 15)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[13] = d13
			ps15.OverlayValues[14] = d14
			ps15.PhiValues = make([]JITValueDesc, 1)
			d16 = d13
			ps15.PhiValues[0] = d16
			if ps15.General && bbs[2].Rendered {
				ctx.W.EmitJmp(lbl3)
				return result
			}
			return bbs[2].RenderPS(ps15)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d17 := ps.PhiValues[0]
						ctx.EnsureDesc(&d17)
						ctx.EmitStoreToStack(d17, 0)
					}
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
			}
			bbs[2].VisitCount++
			if ps.General {
				if bbs[2].Rendered {
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.W.EmitMakeBool(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagBool
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d18 JITValueDesc
			if d2.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(4))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d2.Reg, 4)
				ctx.W.EmitSetcc(r3, CcE)
				d18 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d18)
			}
			ctx.FreeDesc(&d2)
			d19 = d18
			ctx.EnsureDesc(&d19)
			if d19.Loc != LocImm && d19.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d19.Loc == LocImm {
				if d19.Imm.Bool() {
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
			ps20 := PhiState{General: ps.General}
			ps20.OverlayValues = make([]JITValueDesc, 20)
			ps20.OverlayValues[0] = d0
			ps20.OverlayValues[1] = d1
			ps20.OverlayValues[2] = d2
			ps20.OverlayValues[3] = d3
			ps20.OverlayValues[4] = d4
			ps20.OverlayValues[6] = d6
			ps20.OverlayValues[10] = d10
			ps20.OverlayValues[13] = d13
			ps20.OverlayValues[14] = d14
			ps20.OverlayValues[16] = d16
			ps20.OverlayValues[17] = d17
			ps20.OverlayValues[18] = d18
			ps20.OverlayValues[19] = d19
			ps20.PhiValues = make([]JITValueDesc, 1)
			d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			ps20.PhiValues[0] = d21
					return bbs[2].RenderPS(ps20)
				}
			ps22 := PhiState{General: ps.General}
			ps22.OverlayValues = make([]JITValueDesc, 22)
			ps22.OverlayValues[0] = d0
			ps22.OverlayValues[1] = d1
			ps22.OverlayValues[2] = d2
			ps22.OverlayValues[3] = d3
			ps22.OverlayValues[4] = d4
			ps22.OverlayValues[6] = d6
			ps22.OverlayValues[10] = d10
			ps22.OverlayValues[13] = d13
			ps22.OverlayValues[14] = d14
			ps22.OverlayValues[16] = d16
			ps22.OverlayValues[17] = d17
			ps22.OverlayValues[18] = d18
			ps22.OverlayValues[19] = d19
			ps22.OverlayValues[21] = d21
				return bbs[1].RenderPS(ps22)
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d19.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl7)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl7)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 0)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl8)
			ctx.W.EmitJmp(lbl2)
			ps23 := PhiState{General: true}
			ps23.OverlayValues = make([]JITValueDesc, 22)
			ps23.OverlayValues[0] = d0
			ps23.OverlayValues[1] = d1
			ps23.OverlayValues[2] = d2
			ps23.OverlayValues[3] = d3
			ps23.OverlayValues[4] = d4
			ps23.OverlayValues[6] = d6
			ps23.OverlayValues[10] = d10
			ps23.OverlayValues[13] = d13
			ps23.OverlayValues[14] = d14
			ps23.OverlayValues[16] = d16
			ps23.OverlayValues[17] = d17
			ps23.OverlayValues[18] = d18
			ps23.OverlayValues[19] = d19
			ps23.OverlayValues[21] = d21
			ps23.PhiValues = make([]JITValueDesc, 1)
			d25 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			ps23.PhiValues[0] = d25
			ps24 := PhiState{General: true}
			ps24.OverlayValues = make([]JITValueDesc, 26)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[3] = d3
			ps24.OverlayValues[4] = d4
			ps24.OverlayValues[6] = d6
			ps24.OverlayValues[10] = d10
			ps24.OverlayValues[13] = d13
			ps24.OverlayValues[14] = d14
			ps24.OverlayValues[16] = d16
			ps24.OverlayValues[17] = d17
			ps24.OverlayValues[18] = d18
			ps24.OverlayValues[19] = d19
			ps24.OverlayValues[21] = d21
			ps24.OverlayValues[25] = d25
			snap26 := d2
			alloc27 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps23)
			}
			ctx.RestoreAllocState(alloc27)
			d2 = snap26
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps24)
			}
			return result
			ctx.FreeDesc(&d18)
			return result
			}
			ps28 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps28)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(8))
			ctx.W.EmitAddRSP32(int32(8))
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"+", "adds two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values to add", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Fast path: accumulate ints until first non-int, then promote to float if needed
			var sumInt int64
			i := 0
			for i < len(a) {
				v := a[i]
				if v.IsInt() {
					sumInt += v.Int()
					i++
					continue
				}
				break
			}
			if i == len(a) {
				return NewInt(sumInt)
			}
			// Promote to float and continue
			sumFloat := float64(sumInt)
			for ; i < len(a); i++ {
				v := a[i]
				if v.IsNil() {
					return NewNil()
				}
				sumFloat += v.Float()
			}
			return NewFloat(sumFloat)
		},
		true, false, &TypeDescriptor{Optimize: optimizeAssociative},
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d5 JITValueDesc
			_ = d5
			var d6 JITValueDesc
			_ = d6
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
			var d20 JITValueDesc
			_ = d20
			var d21 JITValueDesc
			_ = d21
			var d22 JITValueDesc
			_ = d22
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
			var d58 JITValueDesc
			_ = d58
			var d59 JITValueDesc
			_ = d59
			var d60 JITValueDesc
			_ = d60
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
			var d85 JITValueDesc
			_ = d85
			var d86 JITValueDesc
			_ = d86
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			var bbs [12]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			lbl9 := ctx.W.ReserveLabel()
			bbpos_0_9 := int32(-1)
			_ = bbpos_0_9
			lbl10 := ctx.W.ReserveLabel()
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.W.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 8)
			ps4 := PhiState{General: ps.General}
			ps4.OverlayValues = make([]JITValueDesc, 4)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			ps4.OverlayValues[3] = d3
			ps4.PhiValues = make([]JITValueDesc, 2)
			d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps4.PhiValues[0] = d5
			d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps4.PhiValues[1] = d6
			if ps4.General && bbs[3].Rendered {
				ctx.W.EmitJmp(lbl4)
				return result
			}
			return bbs[3].RenderPS(ps4)
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
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			var d7 JITValueDesc
			if d1.Loc == LocImm {
				idx := int(d1.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d7 = args[idx]
				d7.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d1.Reg] {
					ctx.ProtectReg(d1.Reg)
					seen[d1.Reg] = true
					protected = append(protected, d1.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r1 := ctx.AllocReg()
				r2 := ctx.AllocRegExcept(r1)
				lbl13 := ctx.W.ReserveLabel()
				lbl14 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d1.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl14)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r1, ai.Reg)
						ctx.W.EmitMovRegReg(r2, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r1, tmp.Reg)
						ctx.W.EmitMovRegReg(r2, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
						ctx.BindReg(r1, &pair)
						ctx.BindReg(r2, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r1, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r2, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl13)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl14)
				d8 := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d8)
				ctx.BindReg(r2, &d8)
				ctx.BindReg(r1, &d8)
				ctx.BindReg(r2, &d8)
				ctx.W.EmitMakeNil(d8)
				ctx.W.MarkLabel(lbl13)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d7 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d7)
				ctx.BindReg(r2, &d7)
			}
			d10 = d7
			d10.ID = 0
			d9 = ctx.EmitTagEqualsBorrowed(&d10, tagInt, JITValueDesc{Loc: LocAny})
			d11 = d9
			ctx.EnsureDesc(&d11)
			if d11.Loc != LocImm && d11.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d11.Loc == LocImm {
				if d11.Imm.Bool() {
			ps12 := PhiState{General: ps.General}
			ps12.OverlayValues = make([]JITValueDesc, 12)
			ps12.OverlayValues[0] = d0
			ps12.OverlayValues[1] = d1
			ps12.OverlayValues[2] = d2
			ps12.OverlayValues[3] = d3
			ps12.OverlayValues[5] = d5
			ps12.OverlayValues[6] = d6
			ps12.OverlayValues[7] = d7
			ps12.OverlayValues[8] = d8
			ps12.OverlayValues[9] = d9
			ps12.OverlayValues[10] = d10
			ps12.OverlayValues[11] = d11
					return bbs[4].RenderPS(ps12)
				}
			ps13 := PhiState{General: ps.General}
			ps13.OverlayValues = make([]JITValueDesc, 12)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[3] = d3
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[6] = d6
			ps13.OverlayValues[7] = d7
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
				return bbs[2].RenderPS(ps13)
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d11.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl15)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl15)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl16)
			ctx.W.EmitJmp(lbl3)
			ps14 := PhiState{General: true}
			ps14.OverlayValues = make([]JITValueDesc, 12)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[6] = d6
			ps14.OverlayValues[7] = d7
			ps14.OverlayValues[8] = d8
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps15 := PhiState{General: true}
			ps15.OverlayValues = make([]JITValueDesc, 12)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[7] = d7
			ps15.OverlayValues[8] = d8
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			snap16 := d0
			snap17 := d1
			snap18 := d7
			alloc19 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps15)
			}
			ctx.RestoreAllocState(alloc19)
			d0 = snap16
			d1 = snap17
			d7 = snap18
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps14)
			}
			return result
			ctx.FreeDesc(&d9)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			ctx.ReclaimUntrackedRegs()
			d20 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d20)
			var d21 JITValueDesc
			if d1.Loc == LocImm && d20.Loc == LocImm {
				d21 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == d20.Imm.Int())}
			} else if d20.Loc == LocImm {
				r3 := ctx.AllocRegExcept(d1.Reg)
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d20.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r3, CcE)
				d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d21)
			} else if d1.Loc == LocImm {
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d20.Reg)
				ctx.W.EmitSetcc(r4, CcE)
				d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d21)
			} else {
				r5 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d20.Reg)
				ctx.W.EmitSetcc(r5, CcE)
				d21 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d21)
			}
			ctx.FreeDesc(&d20)
			d22 = d21
			ctx.EnsureDesc(&d22)
			if d22.Loc != LocImm && d22.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d22.Loc == LocImm {
				if d22.Imm.Bool() {
			ps23 := PhiState{General: ps.General}
			ps23.OverlayValues = make([]JITValueDesc, 23)
			ps23.OverlayValues[0] = d0
			ps23.OverlayValues[1] = d1
			ps23.OverlayValues[2] = d2
			ps23.OverlayValues[3] = d3
			ps23.OverlayValues[5] = d5
			ps23.OverlayValues[6] = d6
			ps23.OverlayValues[7] = d7
			ps23.OverlayValues[8] = d8
			ps23.OverlayValues[9] = d9
			ps23.OverlayValues[10] = d10
			ps23.OverlayValues[11] = d11
			ps23.OverlayValues[20] = d20
			ps23.OverlayValues[21] = d21
			ps23.OverlayValues[22] = d22
					return bbs[5].RenderPS(ps23)
				}
			ps24 := PhiState{General: ps.General}
			ps24.OverlayValues = make([]JITValueDesc, 23)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[3] = d3
			ps24.OverlayValues[5] = d5
			ps24.OverlayValues[6] = d6
			ps24.OverlayValues[7] = d7
			ps24.OverlayValues[8] = d8
			ps24.OverlayValues[9] = d9
			ps24.OverlayValues[10] = d10
			ps24.OverlayValues[11] = d11
			ps24.OverlayValues[20] = d20
			ps24.OverlayValues[21] = d21
			ps24.OverlayValues[22] = d22
				return bbs[6].RenderPS(ps24)
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d22.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl17)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl17)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl18)
			ctx.W.EmitJmp(lbl7)
			ps25 := PhiState{General: true}
			ps25.OverlayValues = make([]JITValueDesc, 23)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[6] = d6
			ps25.OverlayValues[7] = d7
			ps25.OverlayValues[8] = d8
			ps25.OverlayValues[9] = d9
			ps25.OverlayValues[10] = d10
			ps25.OverlayValues[11] = d11
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps26 := PhiState{General: true}
			ps26.OverlayValues = make([]JITValueDesc, 23)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[6] = d6
			ps26.OverlayValues[7] = d7
			ps26.OverlayValues[8] = d8
			ps26.OverlayValues[9] = d9
			ps26.OverlayValues[10] = d10
			ps26.OverlayValues[11] = d11
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[22] = d22
			snap27 := d0
			alloc28 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps26)
			}
			ctx.RestoreAllocState(alloc28)
			d0 = snap27
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps25)
			}
			return result
			ctx.FreeDesc(&d21)
			return result
			}
			bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d29 := ps.PhiValues[0]
						ctx.EnsureDesc(&d29)
						ctx.EmitStoreToStack(d29, 0)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d30 := ps.PhiValues[1]
						ctx.EnsureDesc(&d30)
						ctx.EmitStoreToStack(d30, 8)
					}
					ps.General = true
					return bbs[3].RenderPS(ps)
				}
			}
			bbs[3].VisitCount++
			if ps.General {
				if bbs[3].Rendered {
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d1 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d31)
			var d32 JITValueDesc
			if d1.Loc == LocImm && d31.Loc == LocImm {
				d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d31.Imm.Int())}
			} else if d31.Loc == LocImm {
				r6 := ctx.AllocRegExcept(d1.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d31.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d31.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r6, CcL)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d32)
			} else if d1.Loc == LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d31.Reg)
				ctx.W.EmitSetcc(r7, CcL)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d32)
			} else {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d31.Reg)
				ctx.W.EmitSetcc(r8, CcL)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d32)
			}
			ctx.FreeDesc(&d31)
			d33 = d32
			ctx.EnsureDesc(&d33)
			if d33.Loc != LocImm && d33.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d33.Loc == LocImm {
				if d33.Imm.Bool() {
			ps34 := PhiState{General: ps.General}
			ps34.OverlayValues = make([]JITValueDesc, 34)
			ps34.OverlayValues[0] = d0
			ps34.OverlayValues[1] = d1
			ps34.OverlayValues[2] = d2
			ps34.OverlayValues[3] = d3
			ps34.OverlayValues[5] = d5
			ps34.OverlayValues[6] = d6
			ps34.OverlayValues[7] = d7
			ps34.OverlayValues[8] = d8
			ps34.OverlayValues[9] = d9
			ps34.OverlayValues[10] = d10
			ps34.OverlayValues[11] = d11
			ps34.OverlayValues[20] = d20
			ps34.OverlayValues[21] = d21
			ps34.OverlayValues[22] = d22
			ps34.OverlayValues[29] = d29
			ps34.OverlayValues[30] = d30
			ps34.OverlayValues[31] = d31
			ps34.OverlayValues[32] = d32
			ps34.OverlayValues[33] = d33
					return bbs[1].RenderPS(ps34)
				}
			ps35 := PhiState{General: ps.General}
			ps35.OverlayValues = make([]JITValueDesc, 34)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[1] = d1
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[3] = d3
			ps35.OverlayValues[5] = d5
			ps35.OverlayValues[6] = d6
			ps35.OverlayValues[7] = d7
			ps35.OverlayValues[8] = d8
			ps35.OverlayValues[9] = d9
			ps35.OverlayValues[10] = d10
			ps35.OverlayValues[11] = d11
			ps35.OverlayValues[20] = d20
			ps35.OverlayValues[21] = d21
			ps35.OverlayValues[22] = d22
			ps35.OverlayValues[29] = d29
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[32] = d32
			ps35.OverlayValues[33] = d33
				return bbs[2].RenderPS(ps35)
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d33.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl19)
			ctx.W.EmitJmp(lbl20)
			ctx.W.MarkLabel(lbl19)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl20)
			ctx.W.EmitJmp(lbl3)
			ps36 := PhiState{General: true}
			ps36.OverlayValues = make([]JITValueDesc, 34)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[1] = d1
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[3] = d3
			ps36.OverlayValues[5] = d5
			ps36.OverlayValues[6] = d6
			ps36.OverlayValues[7] = d7
			ps36.OverlayValues[8] = d8
			ps36.OverlayValues[9] = d9
			ps36.OverlayValues[10] = d10
			ps36.OverlayValues[11] = d11
			ps36.OverlayValues[20] = d20
			ps36.OverlayValues[21] = d21
			ps36.OverlayValues[22] = d22
			ps36.OverlayValues[29] = d29
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[32] = d32
			ps36.OverlayValues[33] = d33
			ps37 := PhiState{General: true}
			ps37.OverlayValues = make([]JITValueDesc, 34)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[3] = d3
			ps37.OverlayValues[5] = d5
			ps37.OverlayValues[6] = d6
			ps37.OverlayValues[7] = d7
			ps37.OverlayValues[8] = d8
			ps37.OverlayValues[9] = d9
			ps37.OverlayValues[10] = d10
			ps37.OverlayValues[11] = d11
			ps37.OverlayValues[20] = d20
			ps37.OverlayValues[21] = d21
			ps37.OverlayValues[22] = d22
			ps37.OverlayValues[29] = d29
			ps37.OverlayValues[30] = d30
			ps37.OverlayValues[31] = d31
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			snap38 := d1
			snap39 := d7
			snap40 := d9
			alloc41 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps37)
			}
			ctx.RestoreAllocState(alloc41)
			d1 = snap38
			d7 = snap39
			d9 = snap40
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps36)
			}
			return result
			ctx.FreeDesc(&d32)
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
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			ctx.ReclaimUntrackedRegs()
			var d42 JITValueDesc
			if d7.Loc == LocImm {
				d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d7.Imm.Int())}
			} else if d7.Type == tagInt && d7.Loc == LocRegPair {
				ctx.FreeReg(d7.Reg)
				d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg2}
				ctx.BindReg(d7.Reg2, &d42)
				ctx.BindReg(d7.Reg2, &d42)
			} else if d7.Type == tagInt && d7.Loc == LocReg {
				d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg}
				ctx.BindReg(d7.Reg, &d42)
				ctx.BindReg(d7.Reg, &d42)
			} else {
				d42 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d7}, 1)
				d42.Type = tagInt
				ctx.BindReg(d42.Reg, &d42)
			}
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d42)
			var d43 JITValueDesc
			if d0.Loc == LocImm && d42.Loc == LocImm {
				d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d42.Imm.Int())}
			} else if d42.Loc == LocImm && d42.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r9, d0.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
				ctx.BindReg(r9, &d43)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d42.Reg}
				ctx.BindReg(d42.Reg, &d43)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d42.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, RegR11)
				}
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r10 := ctx.AllocRegExcept(d0.Reg, d42.Reg)
				ctx.W.EmitMovRegReg(r10, d0.Reg)
				ctx.W.EmitAddInt64(r10, d42.Reg)
				d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
				ctx.BindReg(r10, &d43)
			}
			if d43.Loc == LocReg && d0.Loc == LocReg && d43.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d44 JITValueDesc
			if d1.Loc == LocImm {
				d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			}
			if d44.Loc == LocReg && d1.Loc == LocReg && d44.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			d45 = d43
			if d45.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d45)
			ctx.EmitStoreToStack(d45, 0)
			d46 = d44
			if d46.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d46)
			ctx.EmitStoreToStack(d46, 8)
			ps47 := PhiState{General: ps.General}
			ps47.OverlayValues = make([]JITValueDesc, 47)
			ps47.OverlayValues[0] = d0
			ps47.OverlayValues[1] = d1
			ps47.OverlayValues[2] = d2
			ps47.OverlayValues[3] = d3
			ps47.OverlayValues[5] = d5
			ps47.OverlayValues[6] = d6
			ps47.OverlayValues[7] = d7
			ps47.OverlayValues[8] = d8
			ps47.OverlayValues[9] = d9
			ps47.OverlayValues[10] = d10
			ps47.OverlayValues[11] = d11
			ps47.OverlayValues[20] = d20
			ps47.OverlayValues[21] = d21
			ps47.OverlayValues[22] = d22
			ps47.OverlayValues[29] = d29
			ps47.OverlayValues[30] = d30
			ps47.OverlayValues[31] = d31
			ps47.OverlayValues[32] = d32
			ps47.OverlayValues[33] = d33
			ps47.OverlayValues[42] = d42
			ps47.OverlayValues[43] = d43
			ps47.OverlayValues[44] = d44
			ps47.OverlayValues[45] = d45
			ps47.OverlayValues[46] = d46
			ps47.PhiValues = make([]JITValueDesc, 2)
			d48 = d43
			ps47.PhiValues[0] = d48
			d49 = d44
			ps47.PhiValues[1] = d49
			if ps47.General && bbs[3].Rendered {
				ctx.W.EmitJmp(lbl4)
				return result
			}
			return bbs[3].RenderPS(ps47)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.W.EmitMakeInt(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d50 JITValueDesc
			if d0.Loc == LocImm {
				d50 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d0.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d0.Reg)
				d50 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d50)
			}
			d51 = d1
			if d51.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d51)
			ctx.EmitStoreToStack(d51, 16)
			d52 = d50
			if d52.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d52)
			ctx.EmitStoreToStack(d52, 24)
			ps53 := PhiState{General: ps.General}
			ps53.OverlayValues = make([]JITValueDesc, 53)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
			ps53.OverlayValues[2] = d2
			ps53.OverlayValues[3] = d3
			ps53.OverlayValues[5] = d5
			ps53.OverlayValues[6] = d6
			ps53.OverlayValues[7] = d7
			ps53.OverlayValues[8] = d8
			ps53.OverlayValues[9] = d9
			ps53.OverlayValues[10] = d10
			ps53.OverlayValues[11] = d11
			ps53.OverlayValues[20] = d20
			ps53.OverlayValues[21] = d21
			ps53.OverlayValues[22] = d22
			ps53.OverlayValues[29] = d29
			ps53.OverlayValues[30] = d30
			ps53.OverlayValues[31] = d31
			ps53.OverlayValues[32] = d32
			ps53.OverlayValues[33] = d33
			ps53.OverlayValues[42] = d42
			ps53.OverlayValues[43] = d43
			ps53.OverlayValues[44] = d44
			ps53.OverlayValues[45] = d45
			ps53.OverlayValues[46] = d46
			ps53.OverlayValues[48] = d48
			ps53.OverlayValues[49] = d49
			ps53.OverlayValues[50] = d50
			ps53.OverlayValues[51] = d51
			ps53.OverlayValues[52] = d52
			ps53.PhiValues = make([]JITValueDesc, 2)
			d54 = d1
			ps53.PhiValues[0] = d54
			d55 = d50
			ps53.PhiValues[1] = d55
			if ps53.General && bbs[9].Rendered {
				ctx.W.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps53)
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
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d56 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d56 = args[idx]
				d56.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d2.Reg] {
					ctx.ProtectReg(d2.Reg)
					seen[d2.Reg] = true
					protected = append(protected, d2.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r11 := ctx.AllocReg()
				r12 := ctx.AllocRegExcept(r11)
				lbl21 := ctx.W.ReserveLabel()
				lbl22 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl22)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r11, ai.Reg)
						ctx.W.EmitMovRegReg(r12, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r11, tmp.Reg)
						ctx.W.EmitMovRegReg(r12, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r11, Reg2: r12}
						ctx.BindReg(r11, &pair)
						ctx.BindReg(r12, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r11, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r12, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl21)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl22)
				d57 := JITValueDesc{Loc: LocRegPair, Reg: r11, Reg2: r12}
				ctx.BindReg(r11, &d57)
				ctx.BindReg(r12, &d57)
				ctx.BindReg(r11, &d57)
				ctx.BindReg(r12, &d57)
				ctx.W.EmitMakeNil(d57)
				ctx.W.MarkLabel(lbl21)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d56 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r11, Reg2: r12}
				ctx.BindReg(r11, &d56)
				ctx.BindReg(r12, &d56)
			}
			d59 = d56
			d59.ID = 0
			d58 = ctx.EmitTagEqualsBorrowed(&d59, tagNil, JITValueDesc{Loc: LocAny})
			d60 = d58
			ctx.EnsureDesc(&d60)
			if d60.Loc != LocImm && d60.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d60.Loc == LocImm {
				if d60.Imm.Bool() {
			ps61 := PhiState{General: ps.General}
			ps61.OverlayValues = make([]JITValueDesc, 61)
			ps61.OverlayValues[0] = d0
			ps61.OverlayValues[1] = d1
			ps61.OverlayValues[2] = d2
			ps61.OverlayValues[3] = d3
			ps61.OverlayValues[5] = d5
			ps61.OverlayValues[6] = d6
			ps61.OverlayValues[7] = d7
			ps61.OverlayValues[8] = d8
			ps61.OverlayValues[9] = d9
			ps61.OverlayValues[10] = d10
			ps61.OverlayValues[11] = d11
			ps61.OverlayValues[20] = d20
			ps61.OverlayValues[21] = d21
			ps61.OverlayValues[22] = d22
			ps61.OverlayValues[29] = d29
			ps61.OverlayValues[30] = d30
			ps61.OverlayValues[31] = d31
			ps61.OverlayValues[32] = d32
			ps61.OverlayValues[33] = d33
			ps61.OverlayValues[42] = d42
			ps61.OverlayValues[43] = d43
			ps61.OverlayValues[44] = d44
			ps61.OverlayValues[45] = d45
			ps61.OverlayValues[46] = d46
			ps61.OverlayValues[48] = d48
			ps61.OverlayValues[49] = d49
			ps61.OverlayValues[50] = d50
			ps61.OverlayValues[51] = d51
			ps61.OverlayValues[52] = d52
			ps61.OverlayValues[54] = d54
			ps61.OverlayValues[55] = d55
			ps61.OverlayValues[56] = d56
			ps61.OverlayValues[57] = d57
			ps61.OverlayValues[58] = d58
			ps61.OverlayValues[59] = d59
			ps61.OverlayValues[60] = d60
					return bbs[10].RenderPS(ps61)
				}
			ps62 := PhiState{General: ps.General}
			ps62.OverlayValues = make([]JITValueDesc, 61)
			ps62.OverlayValues[0] = d0
			ps62.OverlayValues[1] = d1
			ps62.OverlayValues[2] = d2
			ps62.OverlayValues[3] = d3
			ps62.OverlayValues[5] = d5
			ps62.OverlayValues[6] = d6
			ps62.OverlayValues[7] = d7
			ps62.OverlayValues[8] = d8
			ps62.OverlayValues[9] = d9
			ps62.OverlayValues[10] = d10
			ps62.OverlayValues[11] = d11
			ps62.OverlayValues[20] = d20
			ps62.OverlayValues[21] = d21
			ps62.OverlayValues[22] = d22
			ps62.OverlayValues[29] = d29
			ps62.OverlayValues[30] = d30
			ps62.OverlayValues[31] = d31
			ps62.OverlayValues[32] = d32
			ps62.OverlayValues[33] = d33
			ps62.OverlayValues[42] = d42
			ps62.OverlayValues[43] = d43
			ps62.OverlayValues[44] = d44
			ps62.OverlayValues[45] = d45
			ps62.OverlayValues[46] = d46
			ps62.OverlayValues[48] = d48
			ps62.OverlayValues[49] = d49
			ps62.OverlayValues[50] = d50
			ps62.OverlayValues[51] = d51
			ps62.OverlayValues[52] = d52
			ps62.OverlayValues[54] = d54
			ps62.OverlayValues[55] = d55
			ps62.OverlayValues[56] = d56
			ps62.OverlayValues[57] = d57
			ps62.OverlayValues[58] = d58
			ps62.OverlayValues[59] = d59
			ps62.OverlayValues[60] = d60
				return bbs[11].RenderPS(ps62)
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d60.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl23)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl23)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl24)
			ctx.W.EmitJmp(lbl12)
			ps63 := PhiState{General: true}
			ps63.OverlayValues = make([]JITValueDesc, 61)
			ps63.OverlayValues[0] = d0
			ps63.OverlayValues[1] = d1
			ps63.OverlayValues[2] = d2
			ps63.OverlayValues[3] = d3
			ps63.OverlayValues[5] = d5
			ps63.OverlayValues[6] = d6
			ps63.OverlayValues[7] = d7
			ps63.OverlayValues[8] = d8
			ps63.OverlayValues[9] = d9
			ps63.OverlayValues[10] = d10
			ps63.OverlayValues[11] = d11
			ps63.OverlayValues[20] = d20
			ps63.OverlayValues[21] = d21
			ps63.OverlayValues[22] = d22
			ps63.OverlayValues[29] = d29
			ps63.OverlayValues[30] = d30
			ps63.OverlayValues[31] = d31
			ps63.OverlayValues[32] = d32
			ps63.OverlayValues[33] = d33
			ps63.OverlayValues[42] = d42
			ps63.OverlayValues[43] = d43
			ps63.OverlayValues[44] = d44
			ps63.OverlayValues[45] = d45
			ps63.OverlayValues[46] = d46
			ps63.OverlayValues[48] = d48
			ps63.OverlayValues[49] = d49
			ps63.OverlayValues[50] = d50
			ps63.OverlayValues[51] = d51
			ps63.OverlayValues[52] = d52
			ps63.OverlayValues[54] = d54
			ps63.OverlayValues[55] = d55
			ps63.OverlayValues[56] = d56
			ps63.OverlayValues[57] = d57
			ps63.OverlayValues[58] = d58
			ps63.OverlayValues[59] = d59
			ps63.OverlayValues[60] = d60
			ps64 := PhiState{General: true}
			ps64.OverlayValues = make([]JITValueDesc, 61)
			ps64.OverlayValues[0] = d0
			ps64.OverlayValues[1] = d1
			ps64.OverlayValues[2] = d2
			ps64.OverlayValues[3] = d3
			ps64.OverlayValues[5] = d5
			ps64.OverlayValues[6] = d6
			ps64.OverlayValues[7] = d7
			ps64.OverlayValues[8] = d8
			ps64.OverlayValues[9] = d9
			ps64.OverlayValues[10] = d10
			ps64.OverlayValues[11] = d11
			ps64.OverlayValues[20] = d20
			ps64.OverlayValues[21] = d21
			ps64.OverlayValues[22] = d22
			ps64.OverlayValues[29] = d29
			ps64.OverlayValues[30] = d30
			ps64.OverlayValues[31] = d31
			ps64.OverlayValues[32] = d32
			ps64.OverlayValues[33] = d33
			ps64.OverlayValues[42] = d42
			ps64.OverlayValues[43] = d43
			ps64.OverlayValues[44] = d44
			ps64.OverlayValues[45] = d45
			ps64.OverlayValues[46] = d46
			ps64.OverlayValues[48] = d48
			ps64.OverlayValues[49] = d49
			ps64.OverlayValues[50] = d50
			ps64.OverlayValues[51] = d51
			ps64.OverlayValues[52] = d52
			ps64.OverlayValues[54] = d54
			ps64.OverlayValues[55] = d55
			ps64.OverlayValues[56] = d56
			ps64.OverlayValues[57] = d57
			ps64.OverlayValues[58] = d58
			ps64.OverlayValues[59] = d59
			ps64.OverlayValues[60] = d60
			alloc65 := ctx.SnapshotAllocState()
			if !bbs[11].Rendered {
				bbs[11].RenderPS(ps64)
			}
			ctx.RestoreAllocState(alloc65)
			if !bbs[10].Rendered {
				return bbs[10].RenderPS(ps63)
			}
			return result
			ctx.FreeDesc(&d58)
			return result
			}
			bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[8].VisitCount >= 2 {
					ps.General = true
					return bbs[8].RenderPS(ps)
				}
			}
			bbs[8].VisitCount++
			if ps.General {
				if bbs[8].Rendered {
					ctx.W.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.W.MarkLabel(lbl9)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
				d60 = ps.OverlayValues[60]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			ctx.W.EmitMakeFloat(result, d3)
			if d3.Loc == LocReg { ctx.FreeReg(d3.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[9].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d66 := ps.PhiValues[0]
						ctx.EnsureDesc(&d66)
						ctx.EmitStoreToStack(d66, 16)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d67 := ps.PhiValues[1]
						ctx.EnsureDesc(&d67)
						ctx.EmitStoreToStack(d67, 24)
					}
					ps.General = true
					return bbs[9].RenderPS(ps)
				}
			}
			bbs[9].VisitCount++
			if ps.General {
				if bbs[9].Rendered {
					ctx.W.EmitJmp(lbl10)
					return result
				}
				bbs[9].Rendered = true
				bbs[9].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_9 = bbs[9].Address
				ctx.W.MarkLabel(lbl10)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != LocNone {
				d67 = ps.OverlayValues[67]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d2 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d3 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d68 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d68)
			var d69 JITValueDesc
			if d2.Loc == LocImm && d68.Loc == LocImm {
				d69 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d68.Imm.Int())}
			} else if d68.Loc == LocImm {
				r13 := ctx.AllocRegExcept(d2.Reg)
				if d68.Imm.Int() >= -2147483648 && d68.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d68.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d68.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r13, CcL)
				d69 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
				ctx.BindReg(r13, &d69)
			} else if d2.Loc == LocImm {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d68.Reg)
				ctx.W.EmitSetcc(r14, CcL)
				d69 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
				ctx.BindReg(r14, &d69)
			} else {
				r15 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d68.Reg)
				ctx.W.EmitSetcc(r15, CcL)
				d69 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d69)
			}
			ctx.FreeDesc(&d68)
			d70 = d69
			ctx.EnsureDesc(&d70)
			if d70.Loc != LocImm && d70.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d70.Loc == LocImm {
				if d70.Imm.Bool() {
			ps71 := PhiState{General: ps.General}
			ps71.OverlayValues = make([]JITValueDesc, 71)
			ps71.OverlayValues[0] = d0
			ps71.OverlayValues[1] = d1
			ps71.OverlayValues[2] = d2
			ps71.OverlayValues[3] = d3
			ps71.OverlayValues[5] = d5
			ps71.OverlayValues[6] = d6
			ps71.OverlayValues[7] = d7
			ps71.OverlayValues[8] = d8
			ps71.OverlayValues[9] = d9
			ps71.OverlayValues[10] = d10
			ps71.OverlayValues[11] = d11
			ps71.OverlayValues[20] = d20
			ps71.OverlayValues[21] = d21
			ps71.OverlayValues[22] = d22
			ps71.OverlayValues[29] = d29
			ps71.OverlayValues[30] = d30
			ps71.OverlayValues[31] = d31
			ps71.OverlayValues[32] = d32
			ps71.OverlayValues[33] = d33
			ps71.OverlayValues[42] = d42
			ps71.OverlayValues[43] = d43
			ps71.OverlayValues[44] = d44
			ps71.OverlayValues[45] = d45
			ps71.OverlayValues[46] = d46
			ps71.OverlayValues[48] = d48
			ps71.OverlayValues[49] = d49
			ps71.OverlayValues[50] = d50
			ps71.OverlayValues[51] = d51
			ps71.OverlayValues[52] = d52
			ps71.OverlayValues[54] = d54
			ps71.OverlayValues[55] = d55
			ps71.OverlayValues[56] = d56
			ps71.OverlayValues[57] = d57
			ps71.OverlayValues[58] = d58
			ps71.OverlayValues[59] = d59
			ps71.OverlayValues[60] = d60
			ps71.OverlayValues[66] = d66
			ps71.OverlayValues[67] = d67
			ps71.OverlayValues[68] = d68
			ps71.OverlayValues[69] = d69
			ps71.OverlayValues[70] = d70
					return bbs[7].RenderPS(ps71)
				}
			ps72 := PhiState{General: ps.General}
			ps72.OverlayValues = make([]JITValueDesc, 71)
			ps72.OverlayValues[0] = d0
			ps72.OverlayValues[1] = d1
			ps72.OverlayValues[2] = d2
			ps72.OverlayValues[3] = d3
			ps72.OverlayValues[5] = d5
			ps72.OverlayValues[6] = d6
			ps72.OverlayValues[7] = d7
			ps72.OverlayValues[8] = d8
			ps72.OverlayValues[9] = d9
			ps72.OverlayValues[10] = d10
			ps72.OverlayValues[11] = d11
			ps72.OverlayValues[20] = d20
			ps72.OverlayValues[21] = d21
			ps72.OverlayValues[22] = d22
			ps72.OverlayValues[29] = d29
			ps72.OverlayValues[30] = d30
			ps72.OverlayValues[31] = d31
			ps72.OverlayValues[32] = d32
			ps72.OverlayValues[33] = d33
			ps72.OverlayValues[42] = d42
			ps72.OverlayValues[43] = d43
			ps72.OverlayValues[44] = d44
			ps72.OverlayValues[45] = d45
			ps72.OverlayValues[46] = d46
			ps72.OverlayValues[48] = d48
			ps72.OverlayValues[49] = d49
			ps72.OverlayValues[50] = d50
			ps72.OverlayValues[51] = d51
			ps72.OverlayValues[52] = d52
			ps72.OverlayValues[54] = d54
			ps72.OverlayValues[55] = d55
			ps72.OverlayValues[56] = d56
			ps72.OverlayValues[57] = d57
			ps72.OverlayValues[58] = d58
			ps72.OverlayValues[59] = d59
			ps72.OverlayValues[60] = d60
			ps72.OverlayValues[66] = d66
			ps72.OverlayValues[67] = d67
			ps72.OverlayValues[68] = d68
			ps72.OverlayValues[69] = d69
			ps72.OverlayValues[70] = d70
				return bbs[8].RenderPS(ps72)
			}
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d70.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl25)
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl25)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl26)
			ctx.W.EmitJmp(lbl9)
			ps73 := PhiState{General: true}
			ps73.OverlayValues = make([]JITValueDesc, 71)
			ps73.OverlayValues[0] = d0
			ps73.OverlayValues[1] = d1
			ps73.OverlayValues[2] = d2
			ps73.OverlayValues[3] = d3
			ps73.OverlayValues[5] = d5
			ps73.OverlayValues[6] = d6
			ps73.OverlayValues[7] = d7
			ps73.OverlayValues[8] = d8
			ps73.OverlayValues[9] = d9
			ps73.OverlayValues[10] = d10
			ps73.OverlayValues[11] = d11
			ps73.OverlayValues[20] = d20
			ps73.OverlayValues[21] = d21
			ps73.OverlayValues[22] = d22
			ps73.OverlayValues[29] = d29
			ps73.OverlayValues[30] = d30
			ps73.OverlayValues[31] = d31
			ps73.OverlayValues[32] = d32
			ps73.OverlayValues[33] = d33
			ps73.OverlayValues[42] = d42
			ps73.OverlayValues[43] = d43
			ps73.OverlayValues[44] = d44
			ps73.OverlayValues[45] = d45
			ps73.OverlayValues[46] = d46
			ps73.OverlayValues[48] = d48
			ps73.OverlayValues[49] = d49
			ps73.OverlayValues[50] = d50
			ps73.OverlayValues[51] = d51
			ps73.OverlayValues[52] = d52
			ps73.OverlayValues[54] = d54
			ps73.OverlayValues[55] = d55
			ps73.OverlayValues[56] = d56
			ps73.OverlayValues[57] = d57
			ps73.OverlayValues[58] = d58
			ps73.OverlayValues[59] = d59
			ps73.OverlayValues[60] = d60
			ps73.OverlayValues[66] = d66
			ps73.OverlayValues[67] = d67
			ps73.OverlayValues[68] = d68
			ps73.OverlayValues[69] = d69
			ps73.OverlayValues[70] = d70
			ps74 := PhiState{General: true}
			ps74.OverlayValues = make([]JITValueDesc, 71)
			ps74.OverlayValues[0] = d0
			ps74.OverlayValues[1] = d1
			ps74.OverlayValues[2] = d2
			ps74.OverlayValues[3] = d3
			ps74.OverlayValues[5] = d5
			ps74.OverlayValues[6] = d6
			ps74.OverlayValues[7] = d7
			ps74.OverlayValues[8] = d8
			ps74.OverlayValues[9] = d9
			ps74.OverlayValues[10] = d10
			ps74.OverlayValues[11] = d11
			ps74.OverlayValues[20] = d20
			ps74.OverlayValues[21] = d21
			ps74.OverlayValues[22] = d22
			ps74.OverlayValues[29] = d29
			ps74.OverlayValues[30] = d30
			ps74.OverlayValues[31] = d31
			ps74.OverlayValues[32] = d32
			ps74.OverlayValues[33] = d33
			ps74.OverlayValues[42] = d42
			ps74.OverlayValues[43] = d43
			ps74.OverlayValues[44] = d44
			ps74.OverlayValues[45] = d45
			ps74.OverlayValues[46] = d46
			ps74.OverlayValues[48] = d48
			ps74.OverlayValues[49] = d49
			ps74.OverlayValues[50] = d50
			ps74.OverlayValues[51] = d51
			ps74.OverlayValues[52] = d52
			ps74.OverlayValues[54] = d54
			ps74.OverlayValues[55] = d55
			ps74.OverlayValues[56] = d56
			ps74.OverlayValues[57] = d57
			ps74.OverlayValues[58] = d58
			ps74.OverlayValues[59] = d59
			ps74.OverlayValues[60] = d60
			ps74.OverlayValues[66] = d66
			ps74.OverlayValues[67] = d67
			ps74.OverlayValues[68] = d68
			ps74.OverlayValues[69] = d69
			ps74.OverlayValues[70] = d70
			snap75 := d2
			snap76 := d56
			snap77 := d58
			alloc78 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps74)
			}
			ctx.RestoreAllocState(alloc78)
			d2 = snap75
			d56 = snap76
			d58 = snap77
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps73)
			}
			return result
			ctx.FreeDesc(&d69)
			return result
			}
			bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[10].VisitCount >= 2 {
					ps.General = true
					return bbs[10].RenderPS(ps)
				}
			}
			bbs[10].VisitCount++
			if ps.General {
				if bbs[10].Rendered {
					ctx.W.EmitJmp(lbl11)
					return result
				}
				bbs[10].Rendered = true
				bbs[10].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_10 = bbs[10].Address
				ctx.W.MarkLabel(lbl11)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
				d60 = ps.OverlayValues[60]
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
			ctx.ReclaimUntrackedRegs()
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[11].VisitCount >= 2 {
					ps.General = true
					return bbs[11].RenderPS(ps)
				}
			}
			bbs[11].VisitCount++
			if ps.General {
				if bbs[11].Rendered {
					ctx.W.EmitJmp(lbl12)
					return result
				}
				bbs[11].Rendered = true
				bbs[11].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_11 = bbs[11].Address
				ctx.W.MarkLabel(lbl12)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
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
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != LocNone {
				d60 = ps.OverlayValues[60]
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
			ctx.ReclaimUntrackedRegs()
			var d79 JITValueDesc
			if d56.Loc == LocImm {
				d79 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d56.Imm.Float())}
			} else if d56.Type == tagFloat && d56.Loc == LocReg {
				d79 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d56.Reg}
				ctx.BindReg(d56.Reg, &d79)
				ctx.BindReg(d56.Reg, &d79)
			} else if d56.Type == tagFloat && d56.Loc == LocRegPair {
				ctx.FreeReg(d56.Reg)
				d79 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d56.Reg2}
				ctx.BindReg(d56.Reg2, &d79)
				ctx.BindReg(d56.Reg2, &d79)
			} else {
				d79 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d56}, 1)
				d79.Type = tagFloat
				ctx.BindReg(d79.Reg, &d79)
			}
			ctx.FreeDesc(&d56)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d79)
			var d80 JITValueDesc
			if d3.Loc == LocImm && d79.Loc == LocImm {
				d80 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float() + d79.Imm.Float())}
			} else if d3.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				_, xBits := d3.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitAddFloat64(scratch, d79.Reg)
				d80 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else if d79.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(scratch, d3.Reg)
				_, yBits := d79.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitAddFloat64(scratch, RegR11)
				d80 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else {
				r16 := ctx.AllocRegExcept(d3.Reg, d79.Reg)
				ctx.W.EmitMovRegReg(r16, d3.Reg)
				ctx.W.EmitAddFloat64(r16, d79.Reg)
				d80 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
				ctx.BindReg(r16, &d80)
			}
			if d80.Loc == LocReg && d3.Loc == LocReg && d80.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = LocNone
			}
			ctx.FreeDesc(&d79)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d81 JITValueDesc
			if d2.Loc == LocImm {
				d81 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d81 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d81)
			}
			if d81.Loc == LocReg && d2.Loc == LocReg && d81.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			d82 = d81
			if d82.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d82)
			ctx.EmitStoreToStack(d82, 16)
			d83 = d80
			if d83.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d83)
			ctx.EmitStoreToStack(d83, 24)
			ps84 := PhiState{General: ps.General}
			ps84.OverlayValues = make([]JITValueDesc, 84)
			ps84.OverlayValues[0] = d0
			ps84.OverlayValues[1] = d1
			ps84.OverlayValues[2] = d2
			ps84.OverlayValues[3] = d3
			ps84.OverlayValues[5] = d5
			ps84.OverlayValues[6] = d6
			ps84.OverlayValues[7] = d7
			ps84.OverlayValues[8] = d8
			ps84.OverlayValues[9] = d9
			ps84.OverlayValues[10] = d10
			ps84.OverlayValues[11] = d11
			ps84.OverlayValues[20] = d20
			ps84.OverlayValues[21] = d21
			ps84.OverlayValues[22] = d22
			ps84.OverlayValues[29] = d29
			ps84.OverlayValues[30] = d30
			ps84.OverlayValues[31] = d31
			ps84.OverlayValues[32] = d32
			ps84.OverlayValues[33] = d33
			ps84.OverlayValues[42] = d42
			ps84.OverlayValues[43] = d43
			ps84.OverlayValues[44] = d44
			ps84.OverlayValues[45] = d45
			ps84.OverlayValues[46] = d46
			ps84.OverlayValues[48] = d48
			ps84.OverlayValues[49] = d49
			ps84.OverlayValues[50] = d50
			ps84.OverlayValues[51] = d51
			ps84.OverlayValues[52] = d52
			ps84.OverlayValues[54] = d54
			ps84.OverlayValues[55] = d55
			ps84.OverlayValues[56] = d56
			ps84.OverlayValues[57] = d57
			ps84.OverlayValues[58] = d58
			ps84.OverlayValues[59] = d59
			ps84.OverlayValues[60] = d60
			ps84.OverlayValues[66] = d66
			ps84.OverlayValues[67] = d67
			ps84.OverlayValues[68] = d68
			ps84.OverlayValues[69] = d69
			ps84.OverlayValues[70] = d70
			ps84.OverlayValues[79] = d79
			ps84.OverlayValues[80] = d80
			ps84.OverlayValues[81] = d81
			ps84.OverlayValues[82] = d82
			ps84.OverlayValues[83] = d83
			ps84.PhiValues = make([]JITValueDesc, 2)
			d85 = d81
			ps84.PhiValues[0] = d85
			d86 = d80
			ps84.PhiValues[1] = d86
			if ps84.General && bbs[9].Rendered {
				ctx.W.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps84)
			return result
			}
			ps87 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps87)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(32))
			ctx.W.EmitAddRSP32(int32(32))
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"-", "subtracts two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			// Int-first, then promote to float if needed
			if a[0].IsInt() {
				diffInt := a[0].Int()
				i := 1
				for i < len(a) && a[i].IsInt() {
					diffInt -= a[i].Int()
					i++
				}
				if i == len(a) {
					return NewInt(diffInt)
				}
				diffFloat := float64(diffInt)
				for ; i < len(a); i++ {
					diffFloat -= a[i].Float()
				}
				return NewFloat(diffFloat)
			}
			// Float mode from the start
			diffFloat := a[0].Float()
			for i := 1; i < len(a); i++ {
				diffFloat -= a[i].Float()
			}
			return NewFloat(diffFloat)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
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
			var d26 JITValueDesc
			_ = d26
			var d28 JITValueDesc
			_ = d28
			var d29 JITValueDesc
			_ = d29
			var d32 JITValueDesc
			_ = d32
			var d34 JITValueDesc
			_ = d34
			var d35 JITValueDesc
			_ = d35
			var d36 JITValueDesc
			_ = d36
			var d37 JITValueDesc
			_ = d37
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
			var d51 JITValueDesc
			_ = d51
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
			var d99 JITValueDesc
			_ = d99
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
			var d113 JITValueDesc
			_ = d113
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
			var d129 JITValueDesc
			_ = d129
			var d130 JITValueDesc
			_ = d130
			var d131 JITValueDesc
			_ = d131
			var d132 JITValueDesc
			_ = d132
			var d133 JITValueDesc
			_ = d133
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
			var d149 JITValueDesc
			_ = d149
			var d150 JITValueDesc
			_ = d150
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			var bbs [19]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			lbl9 := ctx.W.ReserveLabel()
			bbpos_0_9 := int32(-1)
			_ = bbpos_0_9
			lbl10 := ctx.W.ReserveLabel()
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.W.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.W.ReserveLabel()
			bbpos_0_12 := int32(-1)
			_ = bbpos_0_12
			lbl13 := ctx.W.ReserveLabel()
			bbpos_0_13 := int32(-1)
			_ = bbpos_0_13
			lbl14 := ctx.W.ReserveLabel()
			bbpos_0_14 := int32(-1)
			_ = bbpos_0_14
			lbl15 := ctx.W.ReserveLabel()
			bbpos_0_15 := int32(-1)
			_ = bbpos_0_15
			lbl16 := ctx.W.ReserveLabel()
			bbpos_0_16 := int32(-1)
			_ = bbpos_0_16
			lbl17 := ctx.W.ReserveLabel()
			bbpos_0_17 := int32(-1)
			_ = bbpos_0_17
			lbl18 := ctx.W.ReserveLabel()
			bbpos_0_18 := int32(-1)
			_ = bbpos_0_18
			lbl19 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			ctx.ReclaimUntrackedRegs()
			d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 0)
			ps8 := PhiState{General: ps.General}
			ps8.OverlayValues = make([]JITValueDesc, 8)
			ps8.OverlayValues[0] = d0
			ps8.OverlayValues[1] = d1
			ps8.OverlayValues[2] = d2
			ps8.OverlayValues[3] = d3
			ps8.OverlayValues[4] = d4
			ps8.OverlayValues[5] = d5
			ps8.OverlayValues[6] = d6
			ps8.OverlayValues[7] = d7
			ps8.PhiValues = make([]JITValueDesc, 1)
			d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
			ps8.PhiValues[0] = d9
			if ps8.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps8)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d10 := ps.PhiValues[0]
						ctx.EnsureDesc(&d10)
						ctx.EmitStoreToStack(d10, 0)
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d11 JITValueDesc
			if d0.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d11 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d11)
			}
			if d11.Loc == LocReg && d0.Loc == LocReg && d11.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d7)
			var d12 JITValueDesc
			if d11.Loc == LocImm && d7.Loc == LocImm {
				d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Int() < d7.Imm.Int())}
			} else if d7.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d11.Reg)
				if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d11.Reg, int32(d7.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
					ctx.W.EmitCmpInt64(d11.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d12)
			} else if d11.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d7.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d12)
			} else {
				r3 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitCmpInt64(d11.Reg, d7.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d12)
			}
			ctx.FreeDesc(&d7)
			d13 = d12
			ctx.EnsureDesc(&d13)
			if d13.Loc != LocImm && d13.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d13.Loc == LocImm {
				if d13.Imm.Bool() {
			ps14 := PhiState{General: ps.General}
			ps14.OverlayValues = make([]JITValueDesc, 14)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[6] = d6
			ps14.OverlayValues[7] = d7
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
			ps14.OverlayValues[13] = d13
					return bbs[2].RenderPS(ps14)
				}
			ps15 := PhiState{General: ps.General}
			ps15.OverlayValues = make([]JITValueDesc, 14)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[7] = d7
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			ps15.OverlayValues[12] = d12
			ps15.OverlayValues[13] = d13
				return bbs[3].RenderPS(ps15)
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d13.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl20)
			ctx.W.EmitJmp(lbl21)
			ctx.W.MarkLabel(lbl20)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl21)
			ctx.W.EmitJmp(lbl4)
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 14)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[3] = d3
			ps16.OverlayValues[4] = d4
			ps16.OverlayValues[5] = d5
			ps16.OverlayValues[6] = d6
			ps16.OverlayValues[7] = d7
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[11] = d11
			ps16.OverlayValues[12] = d12
			ps16.OverlayValues[13] = d13
			ps17 := PhiState{General: true}
			ps17.OverlayValues = make([]JITValueDesc, 14)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
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
			snap18 := d11
			alloc19 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps17)
			}
			ctx.RestoreAllocState(alloc19)
			d11 = snap18
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps16)
			}
			return result
			ctx.FreeDesc(&d12)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d11)
			var d20 JITValueDesc
			if d11.Loc == LocImm {
				idx := int(d11.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d20 = args[idx]
				d20.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d11.Reg] {
					ctx.ProtectReg(d11.Reg)
					seen[d11.Reg] = true
					protected = append(protected, d11.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl22 := ctx.W.ReserveLabel()
				lbl23 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d11.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl23)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d11.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl22)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl23)
				d21 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d21)
				ctx.BindReg(r5, &d21)
				ctx.BindReg(r4, &d21)
				ctx.BindReg(r5, &d21)
				ctx.W.EmitMakeNil(d21)
				ctx.W.MarkLabel(lbl22)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d20 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d20)
				ctx.BindReg(r5, &d20)
			}
			d23 = d20
			d23.ID = 0
			d22 = ctx.EmitTagEqualsBorrowed(&d23, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d20)
			d24 = d22
			ctx.EnsureDesc(&d24)
			if d24.Loc != LocImm && d24.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d24.Loc == LocImm {
				if d24.Imm.Bool() {
			ps25 := PhiState{General: ps.General}
			ps25.OverlayValues = make([]JITValueDesc, 25)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[6] = d6
			ps25.OverlayValues[7] = d7
			ps25.OverlayValues[9] = d9
			ps25.OverlayValues[10] = d10
			ps25.OverlayValues[11] = d11
			ps25.OverlayValues[12] = d12
			ps25.OverlayValues[13] = d13
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps25.OverlayValues[23] = d23
			ps25.OverlayValues[24] = d24
					return bbs[4].RenderPS(ps25)
				}
			d26 = d11
			if d26.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, 0)
			ps27 := PhiState{General: ps.General}
			ps27.OverlayValues = make([]JITValueDesc, 27)
			ps27.OverlayValues[0] = d0
			ps27.OverlayValues[1] = d1
			ps27.OverlayValues[2] = d2
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
			ps27.OverlayValues[20] = d20
			ps27.OverlayValues[21] = d21
			ps27.OverlayValues[22] = d22
			ps27.OverlayValues[23] = d23
			ps27.OverlayValues[24] = d24
			ps27.OverlayValues[26] = d26
			ps27.PhiValues = make([]JITValueDesc, 1)
			d28 = d11
			ps27.PhiValues[0] = d28
				return bbs[1].RenderPS(ps27)
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d24.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl24)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl24)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl25)
			d29 = d11
			if d29.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 0)
			ctx.W.EmitJmp(lbl2)
			ps30 := PhiState{General: true}
			ps30.OverlayValues = make([]JITValueDesc, 30)
			ps30.OverlayValues[0] = d0
			ps30.OverlayValues[1] = d1
			ps30.OverlayValues[2] = d2
			ps30.OverlayValues[3] = d3
			ps30.OverlayValues[4] = d4
			ps30.OverlayValues[5] = d5
			ps30.OverlayValues[6] = d6
			ps30.OverlayValues[7] = d7
			ps30.OverlayValues[9] = d9
			ps30.OverlayValues[10] = d10
			ps30.OverlayValues[11] = d11
			ps30.OverlayValues[12] = d12
			ps30.OverlayValues[13] = d13
			ps30.OverlayValues[20] = d20
			ps30.OverlayValues[21] = d21
			ps30.OverlayValues[22] = d22
			ps30.OverlayValues[23] = d23
			ps30.OverlayValues[24] = d24
			ps30.OverlayValues[26] = d26
			ps30.OverlayValues[28] = d28
			ps30.OverlayValues[29] = d29
			ps31 := PhiState{General: true}
			ps31.OverlayValues = make([]JITValueDesc, 30)
			ps31.OverlayValues[0] = d0
			ps31.OverlayValues[1] = d1
			ps31.OverlayValues[2] = d2
			ps31.OverlayValues[3] = d3
			ps31.OverlayValues[4] = d4
			ps31.OverlayValues[5] = d5
			ps31.OverlayValues[6] = d6
			ps31.OverlayValues[7] = d7
			ps31.OverlayValues[9] = d9
			ps31.OverlayValues[10] = d10
			ps31.OverlayValues[11] = d11
			ps31.OverlayValues[12] = d12
			ps31.OverlayValues[13] = d13
			ps31.OverlayValues[20] = d20
			ps31.OverlayValues[21] = d21
			ps31.OverlayValues[22] = d22
			ps31.OverlayValues[23] = d23
			ps31.OverlayValues[24] = d24
			ps31.OverlayValues[26] = d26
			ps31.OverlayValues[28] = d28
			ps31.OverlayValues[29] = d29
			ps31.PhiValues = make([]JITValueDesc, 1)
			d32 = d11
			ps31.PhiValues[0] = d32
			alloc33 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps31)
			}
			ctx.RestoreAllocState(alloc33)
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps30)
			}
			return result
			ctx.FreeDesc(&d22)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
			}
			ctx.ReclaimUntrackedRegs()
			d34 = args[0]
			d34.ID = 0
			d36 = d34
			d36.ID = 0
			d35 = ctx.EmitTagEqualsBorrowed(&d36, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d34)
			d37 = d35
			ctx.EnsureDesc(&d37)
			if d37.Loc != LocImm && d37.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d37.Loc == LocImm {
				if d37.Imm.Bool() {
			ps38 := PhiState{General: ps.General}
			ps38.OverlayValues = make([]JITValueDesc, 38)
			ps38.OverlayValues[0] = d0
			ps38.OverlayValues[1] = d1
			ps38.OverlayValues[2] = d2
			ps38.OverlayValues[3] = d3
			ps38.OverlayValues[4] = d4
			ps38.OverlayValues[5] = d5
			ps38.OverlayValues[6] = d6
			ps38.OverlayValues[7] = d7
			ps38.OverlayValues[9] = d9
			ps38.OverlayValues[10] = d10
			ps38.OverlayValues[11] = d11
			ps38.OverlayValues[12] = d12
			ps38.OverlayValues[13] = d13
			ps38.OverlayValues[20] = d20
			ps38.OverlayValues[21] = d21
			ps38.OverlayValues[22] = d22
			ps38.OverlayValues[23] = d23
			ps38.OverlayValues[24] = d24
			ps38.OverlayValues[26] = d26
			ps38.OverlayValues[28] = d28
			ps38.OverlayValues[29] = d29
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[34] = d34
			ps38.OverlayValues[35] = d35
			ps38.OverlayValues[36] = d36
			ps38.OverlayValues[37] = d37
					return bbs[5].RenderPS(ps38)
				}
			ps39 := PhiState{General: ps.General}
			ps39.OverlayValues = make([]JITValueDesc, 38)
			ps39.OverlayValues[0] = d0
			ps39.OverlayValues[1] = d1
			ps39.OverlayValues[2] = d2
			ps39.OverlayValues[3] = d3
			ps39.OverlayValues[4] = d4
			ps39.OverlayValues[5] = d5
			ps39.OverlayValues[6] = d6
			ps39.OverlayValues[7] = d7
			ps39.OverlayValues[9] = d9
			ps39.OverlayValues[10] = d10
			ps39.OverlayValues[11] = d11
			ps39.OverlayValues[12] = d12
			ps39.OverlayValues[13] = d13
			ps39.OverlayValues[20] = d20
			ps39.OverlayValues[21] = d21
			ps39.OverlayValues[22] = d22
			ps39.OverlayValues[23] = d23
			ps39.OverlayValues[24] = d24
			ps39.OverlayValues[26] = d26
			ps39.OverlayValues[28] = d28
			ps39.OverlayValues[29] = d29
			ps39.OverlayValues[32] = d32
			ps39.OverlayValues[34] = d34
			ps39.OverlayValues[35] = d35
			ps39.OverlayValues[36] = d36
			ps39.OverlayValues[37] = d37
				return bbs[6].RenderPS(ps39)
			}
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d37.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl26)
			ctx.W.EmitJmp(lbl27)
			ctx.W.MarkLabel(lbl26)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl27)
			ctx.W.EmitJmp(lbl7)
			ps40 := PhiState{General: true}
			ps40.OverlayValues = make([]JITValueDesc, 38)
			ps40.OverlayValues[0] = d0
			ps40.OverlayValues[1] = d1
			ps40.OverlayValues[2] = d2
			ps40.OverlayValues[3] = d3
			ps40.OverlayValues[4] = d4
			ps40.OverlayValues[5] = d5
			ps40.OverlayValues[6] = d6
			ps40.OverlayValues[7] = d7
			ps40.OverlayValues[9] = d9
			ps40.OverlayValues[10] = d10
			ps40.OverlayValues[11] = d11
			ps40.OverlayValues[12] = d12
			ps40.OverlayValues[13] = d13
			ps40.OverlayValues[20] = d20
			ps40.OverlayValues[21] = d21
			ps40.OverlayValues[22] = d22
			ps40.OverlayValues[23] = d23
			ps40.OverlayValues[24] = d24
			ps40.OverlayValues[26] = d26
			ps40.OverlayValues[28] = d28
			ps40.OverlayValues[29] = d29
			ps40.OverlayValues[32] = d32
			ps40.OverlayValues[34] = d34
			ps40.OverlayValues[35] = d35
			ps40.OverlayValues[36] = d36
			ps40.OverlayValues[37] = d37
			ps41 := PhiState{General: true}
			ps41.OverlayValues = make([]JITValueDesc, 38)
			ps41.OverlayValues[0] = d0
			ps41.OverlayValues[1] = d1
			ps41.OverlayValues[2] = d2
			ps41.OverlayValues[3] = d3
			ps41.OverlayValues[4] = d4
			ps41.OverlayValues[5] = d5
			ps41.OverlayValues[6] = d6
			ps41.OverlayValues[7] = d7
			ps41.OverlayValues[9] = d9
			ps41.OverlayValues[10] = d10
			ps41.OverlayValues[11] = d11
			ps41.OverlayValues[12] = d12
			ps41.OverlayValues[13] = d13
			ps41.OverlayValues[20] = d20
			ps41.OverlayValues[21] = d21
			ps41.OverlayValues[22] = d22
			ps41.OverlayValues[23] = d23
			ps41.OverlayValues[24] = d24
			ps41.OverlayValues[26] = d26
			ps41.OverlayValues[28] = d28
			ps41.OverlayValues[29] = d29
			ps41.OverlayValues[32] = d32
			ps41.OverlayValues[34] = d34
			ps41.OverlayValues[35] = d35
			ps41.OverlayValues[36] = d36
			ps41.OverlayValues[37] = d37
			alloc42 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps41)
			}
			ctx.RestoreAllocState(alloc42)
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps40)
			}
			return result
			ctx.FreeDesc(&d35)
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
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			d43 = args[0]
			d43.ID = 0
			var d44 JITValueDesc
			if d43.Loc == LocImm {
				d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d43.Imm.Int())}
			} else if d43.Type == tagInt && d43.Loc == LocRegPair {
				ctx.FreeReg(d43.Reg)
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d43.Reg2}
				ctx.BindReg(d43.Reg2, &d44)
				ctx.BindReg(d43.Reg2, &d44)
			} else if d43.Type == tagInt && d43.Loc == LocReg {
				d44 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d43.Reg}
				ctx.BindReg(d43.Reg, &d44)
				ctx.BindReg(d43.Reg, &d44)
			} else {
				d44 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d43}, 1)
				d44.Type = tagInt
				ctx.BindReg(d44.Reg, &d44)
			}
			ctx.FreeDesc(&d43)
			d45 = d44
			if d45.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d45)
			ctx.EmitStoreToStack(d45, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 16)
			ps46 := PhiState{General: ps.General}
			ps46.OverlayValues = make([]JITValueDesc, 46)
			ps46.OverlayValues[0] = d0
			ps46.OverlayValues[1] = d1
			ps46.OverlayValues[2] = d2
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
			ps46.OverlayValues[20] = d20
			ps46.OverlayValues[21] = d21
			ps46.OverlayValues[22] = d22
			ps46.OverlayValues[23] = d23
			ps46.OverlayValues[24] = d24
			ps46.OverlayValues[26] = d26
			ps46.OverlayValues[28] = d28
			ps46.OverlayValues[29] = d29
			ps46.OverlayValues[32] = d32
			ps46.OverlayValues[34] = d34
			ps46.OverlayValues[35] = d35
			ps46.OverlayValues[36] = d36
			ps46.OverlayValues[37] = d37
			ps46.OverlayValues[43] = d43
			ps46.OverlayValues[44] = d44
			ps46.OverlayValues[45] = d45
			ps46.PhiValues = make([]JITValueDesc, 2)
			d47 = d44
			ps46.PhiValues[0] = d47
			d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
			ps46.PhiValues[1] = d48
			if ps46.General && bbs[9].Rendered {
				ctx.W.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps46)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			d49 = args[0]
			d49.ID = 0
			var d50 JITValueDesc
			if d49.Loc == LocImm {
				d50 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d49.Imm.Float())}
			} else if d49.Type == tagFloat && d49.Loc == LocReg {
				d50 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d49.Reg}
				ctx.BindReg(d49.Reg, &d50)
				ctx.BindReg(d49.Reg, &d50)
			} else if d49.Type == tagFloat && d49.Loc == LocRegPair {
				ctx.FreeReg(d49.Reg)
				d50 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d49.Reg2}
				ctx.BindReg(d49.Reg2, &d50)
				ctx.BindReg(d49.Reg2, &d50)
			} else {
				d50 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d49}, 1)
				d50.Type = tagFloat
				ctx.BindReg(d50.Reg, &d50)
			}
			ctx.FreeDesc(&d49)
			d51 = d50
			if d51.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d51)
			ctx.EmitStoreToStack(d51, 40)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 48)
			ps52 := PhiState{General: ps.General}
			ps52.OverlayValues = make([]JITValueDesc, 52)
			ps52.OverlayValues[0] = d0
			ps52.OverlayValues[1] = d1
			ps52.OverlayValues[2] = d2
			ps52.OverlayValues[3] = d3
			ps52.OverlayValues[4] = d4
			ps52.OverlayValues[5] = d5
			ps52.OverlayValues[6] = d6
			ps52.OverlayValues[7] = d7
			ps52.OverlayValues[9] = d9
			ps52.OverlayValues[10] = d10
			ps52.OverlayValues[11] = d11
			ps52.OverlayValues[12] = d12
			ps52.OverlayValues[13] = d13
			ps52.OverlayValues[20] = d20
			ps52.OverlayValues[21] = d21
			ps52.OverlayValues[22] = d22
			ps52.OverlayValues[23] = d23
			ps52.OverlayValues[24] = d24
			ps52.OverlayValues[26] = d26
			ps52.OverlayValues[28] = d28
			ps52.OverlayValues[29] = d29
			ps52.OverlayValues[32] = d32
			ps52.OverlayValues[34] = d34
			ps52.OverlayValues[35] = d35
			ps52.OverlayValues[36] = d36
			ps52.OverlayValues[37] = d37
			ps52.OverlayValues[43] = d43
			ps52.OverlayValues[44] = d44
			ps52.OverlayValues[45] = d45
			ps52.OverlayValues[47] = d47
			ps52.OverlayValues[48] = d48
			ps52.OverlayValues[49] = d49
			ps52.OverlayValues[50] = d50
			ps52.OverlayValues[51] = d51
			ps52.PhiValues = make([]JITValueDesc, 2)
			d53 = d50
			ps52.PhiValues[0] = d53
			d54 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
			ps52.PhiValues[1] = d54
			if ps52.General && bbs[16].Rendered {
				ctx.W.EmitJmp(lbl17)
				return result
			}
			return bbs[16].RenderPS(ps52)
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
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != LocNone {
				d54 = ps.OverlayValues[54]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d55 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d55 = args[idx]
				d55.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d2.Reg] {
					ctx.ProtectReg(d2.Reg)
					seen[d2.Reg] = true
					protected = append(protected, d2.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r6 := ctx.AllocReg()
				r7 := ctx.AllocRegExcept(r6)
				lbl28 := ctx.W.ReserveLabel()
				lbl29 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl29)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r6, ai.Reg)
						ctx.W.EmitMovRegReg(r7, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r6, tmp.Reg)
						ctx.W.EmitMovRegReg(r7, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
						ctx.BindReg(r6, &pair)
						ctx.BindReg(r7, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r6, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r7, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl28)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl29)
				d56 := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d56)
				ctx.BindReg(r7, &d56)
				ctx.BindReg(r6, &d56)
				ctx.BindReg(r7, &d56)
				ctx.W.EmitMakeNil(d56)
				ctx.W.MarkLabel(lbl28)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d55 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d55)
				ctx.BindReg(r7, &d55)
			}
			var d57 JITValueDesc
			if d55.Loc == LocImm {
				d57 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d55.Imm.Int())}
			} else if d55.Type == tagInt && d55.Loc == LocRegPair {
				ctx.FreeReg(d55.Reg)
				d57 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg2}
				ctx.BindReg(d55.Reg2, &d57)
				ctx.BindReg(d55.Reg2, &d57)
			} else if d55.Type == tagInt && d55.Loc == LocReg {
				d57 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d55.Reg}
				ctx.BindReg(d55.Reg, &d57)
				ctx.BindReg(d55.Reg, &d57)
			} else {
				d57 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d55}, 1)
				d57.Type = tagInt
				ctx.BindReg(d57.Reg, &d57)
			}
			ctx.FreeDesc(&d55)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d57)
			var d58 JITValueDesc
			if d1.Loc == LocImm && d57.Loc == LocImm {
				d58 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() - d57.Imm.Int())}
			} else if d57.Loc == LocImm && d57.Imm.Int() == 0 {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(r8, d1.Reg)
				d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d58)
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d57.Reg)
				d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d58)
			} else if d57.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d57.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d57.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, RegR11)
				}
				d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d58)
			} else {
				r9 := ctx.AllocRegExcept(d1.Reg, d57.Reg)
				ctx.W.EmitMovRegReg(r9, d1.Reg)
				ctx.W.EmitSubInt64(r9, d57.Reg)
				d58 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
				ctx.BindReg(r9, &d58)
			}
			if d58.Loc == LocReg && d1.Loc == LocReg && d58.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d57)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d59 JITValueDesc
			if d2.Loc == LocImm {
				d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d59 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d59)
			}
			if d59.Loc == LocReg && d2.Loc == LocReg && d59.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			d60 = d58
			if d60.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d60)
			ctx.EmitStoreToStack(d60, 8)
			d61 = d59
			if d61.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d61)
			ctx.EmitStoreToStack(d61, 16)
			ps62 := PhiState{General: ps.General}
			ps62.OverlayValues = make([]JITValueDesc, 62)
			ps62.OverlayValues[0] = d0
			ps62.OverlayValues[1] = d1
			ps62.OverlayValues[2] = d2
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
			ps62.OverlayValues[20] = d20
			ps62.OverlayValues[21] = d21
			ps62.OverlayValues[22] = d22
			ps62.OverlayValues[23] = d23
			ps62.OverlayValues[24] = d24
			ps62.OverlayValues[26] = d26
			ps62.OverlayValues[28] = d28
			ps62.OverlayValues[29] = d29
			ps62.OverlayValues[32] = d32
			ps62.OverlayValues[34] = d34
			ps62.OverlayValues[35] = d35
			ps62.OverlayValues[36] = d36
			ps62.OverlayValues[37] = d37
			ps62.OverlayValues[43] = d43
			ps62.OverlayValues[44] = d44
			ps62.OverlayValues[45] = d45
			ps62.OverlayValues[47] = d47
			ps62.OverlayValues[48] = d48
			ps62.OverlayValues[49] = d49
			ps62.OverlayValues[50] = d50
			ps62.OverlayValues[51] = d51
			ps62.OverlayValues[53] = d53
			ps62.OverlayValues[54] = d54
			ps62.OverlayValues[55] = d55
			ps62.OverlayValues[56] = d56
			ps62.OverlayValues[57] = d57
			ps62.OverlayValues[58] = d58
			ps62.OverlayValues[59] = d59
			ps62.OverlayValues[60] = d60
			ps62.OverlayValues[61] = d61
			ps62.PhiValues = make([]JITValueDesc, 2)
			d63 = d58
			ps62.PhiValues[0] = d63
			d64 = d59
			ps62.PhiValues[1] = d64
			if ps62.General && bbs[9].Rendered {
				ctx.W.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps62)
			return result
			}
			bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[8].VisitCount >= 2 {
					ps.General = true
					return bbs[8].RenderPS(ps)
				}
			}
			bbs[8].VisitCount++
			if ps.General {
				if bbs[8].Rendered {
					ctx.W.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.W.MarkLabel(lbl9)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != LocNone {
				d64 = ps.OverlayValues[64]
			}
			ctx.ReclaimUntrackedRegs()
			d65 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d65)
			var d66 JITValueDesc
			if d2.Loc == LocImm && d65.Loc == LocImm {
				d66 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == d65.Imm.Int())}
			} else if d65.Loc == LocImm {
				r10 := ctx.AllocRegExcept(d2.Reg)
				if d65.Imm.Int() >= -2147483648 && d65.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d65.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d65.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r10, CcE)
				d66 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d66)
			} else if d2.Loc == LocImm {
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d65.Reg)
				ctx.W.EmitSetcc(r11, CcE)
				d66 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d66)
			} else {
				r12 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d65.Reg)
				ctx.W.EmitSetcc(r12, CcE)
				d66 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
				ctx.BindReg(r12, &d66)
			}
			ctx.FreeDesc(&d65)
			d67 = d66
			ctx.EnsureDesc(&d67)
			if d67.Loc != LocImm && d67.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d67.Loc == LocImm {
				if d67.Imm.Bool() {
			ps68 := PhiState{General: ps.General}
			ps68.OverlayValues = make([]JITValueDesc, 68)
			ps68.OverlayValues[0] = d0
			ps68.OverlayValues[1] = d1
			ps68.OverlayValues[2] = d2
			ps68.OverlayValues[3] = d3
			ps68.OverlayValues[4] = d4
			ps68.OverlayValues[5] = d5
			ps68.OverlayValues[6] = d6
			ps68.OverlayValues[7] = d7
			ps68.OverlayValues[9] = d9
			ps68.OverlayValues[10] = d10
			ps68.OverlayValues[11] = d11
			ps68.OverlayValues[12] = d12
			ps68.OverlayValues[13] = d13
			ps68.OverlayValues[20] = d20
			ps68.OverlayValues[21] = d21
			ps68.OverlayValues[22] = d22
			ps68.OverlayValues[23] = d23
			ps68.OverlayValues[24] = d24
			ps68.OverlayValues[26] = d26
			ps68.OverlayValues[28] = d28
			ps68.OverlayValues[29] = d29
			ps68.OverlayValues[32] = d32
			ps68.OverlayValues[34] = d34
			ps68.OverlayValues[35] = d35
			ps68.OverlayValues[36] = d36
			ps68.OverlayValues[37] = d37
			ps68.OverlayValues[43] = d43
			ps68.OverlayValues[44] = d44
			ps68.OverlayValues[45] = d45
			ps68.OverlayValues[47] = d47
			ps68.OverlayValues[48] = d48
			ps68.OverlayValues[49] = d49
			ps68.OverlayValues[50] = d50
			ps68.OverlayValues[51] = d51
			ps68.OverlayValues[53] = d53
			ps68.OverlayValues[54] = d54
			ps68.OverlayValues[55] = d55
			ps68.OverlayValues[56] = d56
			ps68.OverlayValues[57] = d57
			ps68.OverlayValues[58] = d58
			ps68.OverlayValues[59] = d59
			ps68.OverlayValues[60] = d60
			ps68.OverlayValues[61] = d61
			ps68.OverlayValues[63] = d63
			ps68.OverlayValues[64] = d64
			ps68.OverlayValues[65] = d65
			ps68.OverlayValues[66] = d66
			ps68.OverlayValues[67] = d67
					return bbs[11].RenderPS(ps68)
				}
			ps69 := PhiState{General: ps.General}
			ps69.OverlayValues = make([]JITValueDesc, 68)
			ps69.OverlayValues[0] = d0
			ps69.OverlayValues[1] = d1
			ps69.OverlayValues[2] = d2
			ps69.OverlayValues[3] = d3
			ps69.OverlayValues[4] = d4
			ps69.OverlayValues[5] = d5
			ps69.OverlayValues[6] = d6
			ps69.OverlayValues[7] = d7
			ps69.OverlayValues[9] = d9
			ps69.OverlayValues[10] = d10
			ps69.OverlayValues[11] = d11
			ps69.OverlayValues[12] = d12
			ps69.OverlayValues[13] = d13
			ps69.OverlayValues[20] = d20
			ps69.OverlayValues[21] = d21
			ps69.OverlayValues[22] = d22
			ps69.OverlayValues[23] = d23
			ps69.OverlayValues[24] = d24
			ps69.OverlayValues[26] = d26
			ps69.OverlayValues[28] = d28
			ps69.OverlayValues[29] = d29
			ps69.OverlayValues[32] = d32
			ps69.OverlayValues[34] = d34
			ps69.OverlayValues[35] = d35
			ps69.OverlayValues[36] = d36
			ps69.OverlayValues[37] = d37
			ps69.OverlayValues[43] = d43
			ps69.OverlayValues[44] = d44
			ps69.OverlayValues[45] = d45
			ps69.OverlayValues[47] = d47
			ps69.OverlayValues[48] = d48
			ps69.OverlayValues[49] = d49
			ps69.OverlayValues[50] = d50
			ps69.OverlayValues[51] = d51
			ps69.OverlayValues[53] = d53
			ps69.OverlayValues[54] = d54
			ps69.OverlayValues[55] = d55
			ps69.OverlayValues[56] = d56
			ps69.OverlayValues[57] = d57
			ps69.OverlayValues[58] = d58
			ps69.OverlayValues[59] = d59
			ps69.OverlayValues[60] = d60
			ps69.OverlayValues[61] = d61
			ps69.OverlayValues[63] = d63
			ps69.OverlayValues[64] = d64
			ps69.OverlayValues[65] = d65
			ps69.OverlayValues[66] = d66
			ps69.OverlayValues[67] = d67
				return bbs[12].RenderPS(ps69)
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d67.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl30)
			ctx.W.EmitJmp(lbl31)
			ctx.W.MarkLabel(lbl30)
			ctx.W.EmitJmp(lbl12)
			ctx.W.MarkLabel(lbl31)
			ctx.W.EmitJmp(lbl13)
			ps70 := PhiState{General: true}
			ps70.OverlayValues = make([]JITValueDesc, 68)
			ps70.OverlayValues[0] = d0
			ps70.OverlayValues[1] = d1
			ps70.OverlayValues[2] = d2
			ps70.OverlayValues[3] = d3
			ps70.OverlayValues[4] = d4
			ps70.OverlayValues[5] = d5
			ps70.OverlayValues[6] = d6
			ps70.OverlayValues[7] = d7
			ps70.OverlayValues[9] = d9
			ps70.OverlayValues[10] = d10
			ps70.OverlayValues[11] = d11
			ps70.OverlayValues[12] = d12
			ps70.OverlayValues[13] = d13
			ps70.OverlayValues[20] = d20
			ps70.OverlayValues[21] = d21
			ps70.OverlayValues[22] = d22
			ps70.OverlayValues[23] = d23
			ps70.OverlayValues[24] = d24
			ps70.OverlayValues[26] = d26
			ps70.OverlayValues[28] = d28
			ps70.OverlayValues[29] = d29
			ps70.OverlayValues[32] = d32
			ps70.OverlayValues[34] = d34
			ps70.OverlayValues[35] = d35
			ps70.OverlayValues[36] = d36
			ps70.OverlayValues[37] = d37
			ps70.OverlayValues[43] = d43
			ps70.OverlayValues[44] = d44
			ps70.OverlayValues[45] = d45
			ps70.OverlayValues[47] = d47
			ps70.OverlayValues[48] = d48
			ps70.OverlayValues[49] = d49
			ps70.OverlayValues[50] = d50
			ps70.OverlayValues[51] = d51
			ps70.OverlayValues[53] = d53
			ps70.OverlayValues[54] = d54
			ps70.OverlayValues[55] = d55
			ps70.OverlayValues[56] = d56
			ps70.OverlayValues[57] = d57
			ps70.OverlayValues[58] = d58
			ps70.OverlayValues[59] = d59
			ps70.OverlayValues[60] = d60
			ps70.OverlayValues[61] = d61
			ps70.OverlayValues[63] = d63
			ps70.OverlayValues[64] = d64
			ps70.OverlayValues[65] = d65
			ps70.OverlayValues[66] = d66
			ps70.OverlayValues[67] = d67
			ps71 := PhiState{General: true}
			ps71.OverlayValues = make([]JITValueDesc, 68)
			ps71.OverlayValues[0] = d0
			ps71.OverlayValues[1] = d1
			ps71.OverlayValues[2] = d2
			ps71.OverlayValues[3] = d3
			ps71.OverlayValues[4] = d4
			ps71.OverlayValues[5] = d5
			ps71.OverlayValues[6] = d6
			ps71.OverlayValues[7] = d7
			ps71.OverlayValues[9] = d9
			ps71.OverlayValues[10] = d10
			ps71.OverlayValues[11] = d11
			ps71.OverlayValues[12] = d12
			ps71.OverlayValues[13] = d13
			ps71.OverlayValues[20] = d20
			ps71.OverlayValues[21] = d21
			ps71.OverlayValues[22] = d22
			ps71.OverlayValues[23] = d23
			ps71.OverlayValues[24] = d24
			ps71.OverlayValues[26] = d26
			ps71.OverlayValues[28] = d28
			ps71.OverlayValues[29] = d29
			ps71.OverlayValues[32] = d32
			ps71.OverlayValues[34] = d34
			ps71.OverlayValues[35] = d35
			ps71.OverlayValues[36] = d36
			ps71.OverlayValues[37] = d37
			ps71.OverlayValues[43] = d43
			ps71.OverlayValues[44] = d44
			ps71.OverlayValues[45] = d45
			ps71.OverlayValues[47] = d47
			ps71.OverlayValues[48] = d48
			ps71.OverlayValues[49] = d49
			ps71.OverlayValues[50] = d50
			ps71.OverlayValues[51] = d51
			ps71.OverlayValues[53] = d53
			ps71.OverlayValues[54] = d54
			ps71.OverlayValues[55] = d55
			ps71.OverlayValues[56] = d56
			ps71.OverlayValues[57] = d57
			ps71.OverlayValues[58] = d58
			ps71.OverlayValues[59] = d59
			ps71.OverlayValues[60] = d60
			ps71.OverlayValues[61] = d61
			ps71.OverlayValues[63] = d63
			ps71.OverlayValues[64] = d64
			ps71.OverlayValues[65] = d65
			ps71.OverlayValues[66] = d66
			ps71.OverlayValues[67] = d67
			snap72 := d1
			alloc73 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps71)
			}
			ctx.RestoreAllocState(alloc73)
			d1 = snap72
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps70)
			}
			return result
			ctx.FreeDesc(&d66)
			return result
			}
			bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[9].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d74 := ps.PhiValues[0]
						ctx.EnsureDesc(&d74)
						ctx.EmitStoreToStack(d74, 8)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d75 := ps.PhiValues[1]
						ctx.EnsureDesc(&d75)
						ctx.EmitStoreToStack(d75, 16)
					}
					ps.General = true
					return bbs[9].RenderPS(ps)
				}
			}
			bbs[9].VisitCount++
			if ps.General {
				if bbs[9].Rendered {
					ctx.W.EmitJmp(lbl10)
					return result
				}
				bbs[9].Rendered = true
				bbs[9].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_9 = bbs[9].Address
				ctx.W.MarkLabel(lbl10)
				ctx.W.ResolveFixups()
			}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d2 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d76 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d76)
			var d77 JITValueDesc
			if d2.Loc == LocImm && d76.Loc == LocImm {
				d77 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d76.Imm.Int())}
			} else if d76.Loc == LocImm {
				r13 := ctx.AllocRegExcept(d2.Reg)
				if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d76.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d76.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r13, CcL)
				d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
				ctx.BindReg(r13, &d77)
			} else if d2.Loc == LocImm {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d76.Reg)
				ctx.W.EmitSetcc(r14, CcL)
				d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
				ctx.BindReg(r14, &d77)
			} else {
				r15 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d76.Reg)
				ctx.W.EmitSetcc(r15, CcL)
				d77 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d77)
			}
			ctx.FreeDesc(&d76)
			d78 = d77
			ctx.EnsureDesc(&d78)
			if d78.Loc != LocImm && d78.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d78.Loc == LocImm {
				if d78.Imm.Bool() {
			ps79 := PhiState{General: ps.General}
			ps79.OverlayValues = make([]JITValueDesc, 79)
			ps79.OverlayValues[0] = d0
			ps79.OverlayValues[1] = d1
			ps79.OverlayValues[2] = d2
			ps79.OverlayValues[3] = d3
			ps79.OverlayValues[4] = d4
			ps79.OverlayValues[5] = d5
			ps79.OverlayValues[6] = d6
			ps79.OverlayValues[7] = d7
			ps79.OverlayValues[9] = d9
			ps79.OverlayValues[10] = d10
			ps79.OverlayValues[11] = d11
			ps79.OverlayValues[12] = d12
			ps79.OverlayValues[13] = d13
			ps79.OverlayValues[20] = d20
			ps79.OverlayValues[21] = d21
			ps79.OverlayValues[22] = d22
			ps79.OverlayValues[23] = d23
			ps79.OverlayValues[24] = d24
			ps79.OverlayValues[26] = d26
			ps79.OverlayValues[28] = d28
			ps79.OverlayValues[29] = d29
			ps79.OverlayValues[32] = d32
			ps79.OverlayValues[34] = d34
			ps79.OverlayValues[35] = d35
			ps79.OverlayValues[36] = d36
			ps79.OverlayValues[37] = d37
			ps79.OverlayValues[43] = d43
			ps79.OverlayValues[44] = d44
			ps79.OverlayValues[45] = d45
			ps79.OverlayValues[47] = d47
			ps79.OverlayValues[48] = d48
			ps79.OverlayValues[49] = d49
			ps79.OverlayValues[50] = d50
			ps79.OverlayValues[51] = d51
			ps79.OverlayValues[53] = d53
			ps79.OverlayValues[54] = d54
			ps79.OverlayValues[55] = d55
			ps79.OverlayValues[56] = d56
			ps79.OverlayValues[57] = d57
			ps79.OverlayValues[58] = d58
			ps79.OverlayValues[59] = d59
			ps79.OverlayValues[60] = d60
			ps79.OverlayValues[61] = d61
			ps79.OverlayValues[63] = d63
			ps79.OverlayValues[64] = d64
			ps79.OverlayValues[65] = d65
			ps79.OverlayValues[66] = d66
			ps79.OverlayValues[67] = d67
			ps79.OverlayValues[74] = d74
			ps79.OverlayValues[75] = d75
			ps79.OverlayValues[76] = d76
			ps79.OverlayValues[77] = d77
			ps79.OverlayValues[78] = d78
					return bbs[10].RenderPS(ps79)
				}
			ps80 := PhiState{General: ps.General}
			ps80.OverlayValues = make([]JITValueDesc, 79)
			ps80.OverlayValues[0] = d0
			ps80.OverlayValues[1] = d1
			ps80.OverlayValues[2] = d2
			ps80.OverlayValues[3] = d3
			ps80.OverlayValues[4] = d4
			ps80.OverlayValues[5] = d5
			ps80.OverlayValues[6] = d6
			ps80.OverlayValues[7] = d7
			ps80.OverlayValues[9] = d9
			ps80.OverlayValues[10] = d10
			ps80.OverlayValues[11] = d11
			ps80.OverlayValues[12] = d12
			ps80.OverlayValues[13] = d13
			ps80.OverlayValues[20] = d20
			ps80.OverlayValues[21] = d21
			ps80.OverlayValues[22] = d22
			ps80.OverlayValues[23] = d23
			ps80.OverlayValues[24] = d24
			ps80.OverlayValues[26] = d26
			ps80.OverlayValues[28] = d28
			ps80.OverlayValues[29] = d29
			ps80.OverlayValues[32] = d32
			ps80.OverlayValues[34] = d34
			ps80.OverlayValues[35] = d35
			ps80.OverlayValues[36] = d36
			ps80.OverlayValues[37] = d37
			ps80.OverlayValues[43] = d43
			ps80.OverlayValues[44] = d44
			ps80.OverlayValues[45] = d45
			ps80.OverlayValues[47] = d47
			ps80.OverlayValues[48] = d48
			ps80.OverlayValues[49] = d49
			ps80.OverlayValues[50] = d50
			ps80.OverlayValues[51] = d51
			ps80.OverlayValues[53] = d53
			ps80.OverlayValues[54] = d54
			ps80.OverlayValues[55] = d55
			ps80.OverlayValues[56] = d56
			ps80.OverlayValues[57] = d57
			ps80.OverlayValues[58] = d58
			ps80.OverlayValues[59] = d59
			ps80.OverlayValues[60] = d60
			ps80.OverlayValues[61] = d61
			ps80.OverlayValues[63] = d63
			ps80.OverlayValues[64] = d64
			ps80.OverlayValues[65] = d65
			ps80.OverlayValues[66] = d66
			ps80.OverlayValues[67] = d67
			ps80.OverlayValues[74] = d74
			ps80.OverlayValues[75] = d75
			ps80.OverlayValues[76] = d76
			ps80.OverlayValues[77] = d77
			ps80.OverlayValues[78] = d78
				return bbs[8].RenderPS(ps80)
			}
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d78.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl32)
			ctx.W.EmitJmp(lbl33)
			ctx.W.MarkLabel(lbl32)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl33)
			ctx.W.EmitJmp(lbl9)
			ps81 := PhiState{General: true}
			ps81.OverlayValues = make([]JITValueDesc, 79)
			ps81.OverlayValues[0] = d0
			ps81.OverlayValues[1] = d1
			ps81.OverlayValues[2] = d2
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
			ps81.OverlayValues[20] = d20
			ps81.OverlayValues[21] = d21
			ps81.OverlayValues[22] = d22
			ps81.OverlayValues[23] = d23
			ps81.OverlayValues[24] = d24
			ps81.OverlayValues[26] = d26
			ps81.OverlayValues[28] = d28
			ps81.OverlayValues[29] = d29
			ps81.OverlayValues[32] = d32
			ps81.OverlayValues[34] = d34
			ps81.OverlayValues[35] = d35
			ps81.OverlayValues[36] = d36
			ps81.OverlayValues[37] = d37
			ps81.OverlayValues[43] = d43
			ps81.OverlayValues[44] = d44
			ps81.OverlayValues[45] = d45
			ps81.OverlayValues[47] = d47
			ps81.OverlayValues[48] = d48
			ps81.OverlayValues[49] = d49
			ps81.OverlayValues[50] = d50
			ps81.OverlayValues[51] = d51
			ps81.OverlayValues[53] = d53
			ps81.OverlayValues[54] = d54
			ps81.OverlayValues[55] = d55
			ps81.OverlayValues[56] = d56
			ps81.OverlayValues[57] = d57
			ps81.OverlayValues[58] = d58
			ps81.OverlayValues[59] = d59
			ps81.OverlayValues[60] = d60
			ps81.OverlayValues[61] = d61
			ps81.OverlayValues[63] = d63
			ps81.OverlayValues[64] = d64
			ps81.OverlayValues[65] = d65
			ps81.OverlayValues[66] = d66
			ps81.OverlayValues[67] = d67
			ps81.OverlayValues[74] = d74
			ps81.OverlayValues[75] = d75
			ps81.OverlayValues[76] = d76
			ps81.OverlayValues[77] = d77
			ps81.OverlayValues[78] = d78
			ps82 := PhiState{General: true}
			ps82.OverlayValues = make([]JITValueDesc, 79)
			ps82.OverlayValues[0] = d0
			ps82.OverlayValues[1] = d1
			ps82.OverlayValues[2] = d2
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
			ps82.OverlayValues[20] = d20
			ps82.OverlayValues[21] = d21
			ps82.OverlayValues[22] = d22
			ps82.OverlayValues[23] = d23
			ps82.OverlayValues[24] = d24
			ps82.OverlayValues[26] = d26
			ps82.OverlayValues[28] = d28
			ps82.OverlayValues[29] = d29
			ps82.OverlayValues[32] = d32
			ps82.OverlayValues[34] = d34
			ps82.OverlayValues[35] = d35
			ps82.OverlayValues[36] = d36
			ps82.OverlayValues[37] = d37
			ps82.OverlayValues[43] = d43
			ps82.OverlayValues[44] = d44
			ps82.OverlayValues[45] = d45
			ps82.OverlayValues[47] = d47
			ps82.OverlayValues[48] = d48
			ps82.OverlayValues[49] = d49
			ps82.OverlayValues[50] = d50
			ps82.OverlayValues[51] = d51
			ps82.OverlayValues[53] = d53
			ps82.OverlayValues[54] = d54
			ps82.OverlayValues[55] = d55
			ps82.OverlayValues[56] = d56
			ps82.OverlayValues[57] = d57
			ps82.OverlayValues[58] = d58
			ps82.OverlayValues[59] = d59
			ps82.OverlayValues[60] = d60
			ps82.OverlayValues[61] = d61
			ps82.OverlayValues[63] = d63
			ps82.OverlayValues[64] = d64
			ps82.OverlayValues[65] = d65
			ps82.OverlayValues[66] = d66
			ps82.OverlayValues[67] = d67
			ps82.OverlayValues[74] = d74
			ps82.OverlayValues[75] = d75
			ps82.OverlayValues[76] = d76
			ps82.OverlayValues[77] = d77
			ps82.OverlayValues[78] = d78
			snap83 := d2
			alloc84 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps82)
			}
			ctx.RestoreAllocState(alloc84)
			d2 = snap83
			if !bbs[10].Rendered {
				return bbs[10].RenderPS(ps81)
			}
			return result
			ctx.FreeDesc(&d77)
			return result
			}
			bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[10].VisitCount >= 2 {
					ps.General = true
					return bbs[10].RenderPS(ps)
				}
			}
			bbs[10].VisitCount++
			if ps.General {
				if bbs[10].Rendered {
					ctx.W.EmitJmp(lbl11)
					return result
				}
				bbs[10].Rendered = true
				bbs[10].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_10 = bbs[10].Address
				ctx.W.MarkLabel(lbl11)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d85 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d85 = args[idx]
				d85.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d2.Reg] {
					ctx.ProtectReg(d2.Reg)
					seen[d2.Reg] = true
					protected = append(protected, d2.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r16 := ctx.AllocReg()
				r17 := ctx.AllocRegExcept(r16)
				lbl34 := ctx.W.ReserveLabel()
				lbl35 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl35)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r16, ai.Reg)
						ctx.W.EmitMovRegReg(r17, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r16, tmp.Reg)
						ctx.W.EmitMovRegReg(r17, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r16, Reg2: r17}
						ctx.BindReg(r16, &pair)
						ctx.BindReg(r17, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r16, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r17, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl34)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl35)
				d86 := JITValueDesc{Loc: LocRegPair, Reg: r16, Reg2: r17}
				ctx.BindReg(r16, &d86)
				ctx.BindReg(r17, &d86)
				ctx.BindReg(r16, &d86)
				ctx.BindReg(r17, &d86)
				ctx.W.EmitMakeNil(d86)
				ctx.W.MarkLabel(lbl34)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d85 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r16, Reg2: r17}
				ctx.BindReg(r16, &d85)
				ctx.BindReg(r17, &d85)
			}
			d88 = d85
			d88.ID = 0
			d87 = ctx.EmitTagEqualsBorrowed(&d88, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d85)
			d89 = d87
			ctx.EnsureDesc(&d89)
			if d89.Loc != LocImm && d89.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d89.Loc == LocImm {
				if d89.Imm.Bool() {
			ps90 := PhiState{General: ps.General}
			ps90.OverlayValues = make([]JITValueDesc, 90)
			ps90.OverlayValues[0] = d0
			ps90.OverlayValues[1] = d1
			ps90.OverlayValues[2] = d2
			ps90.OverlayValues[3] = d3
			ps90.OverlayValues[4] = d4
			ps90.OverlayValues[5] = d5
			ps90.OverlayValues[6] = d6
			ps90.OverlayValues[7] = d7
			ps90.OverlayValues[9] = d9
			ps90.OverlayValues[10] = d10
			ps90.OverlayValues[11] = d11
			ps90.OverlayValues[12] = d12
			ps90.OverlayValues[13] = d13
			ps90.OverlayValues[20] = d20
			ps90.OverlayValues[21] = d21
			ps90.OverlayValues[22] = d22
			ps90.OverlayValues[23] = d23
			ps90.OverlayValues[24] = d24
			ps90.OverlayValues[26] = d26
			ps90.OverlayValues[28] = d28
			ps90.OverlayValues[29] = d29
			ps90.OverlayValues[32] = d32
			ps90.OverlayValues[34] = d34
			ps90.OverlayValues[35] = d35
			ps90.OverlayValues[36] = d36
			ps90.OverlayValues[37] = d37
			ps90.OverlayValues[43] = d43
			ps90.OverlayValues[44] = d44
			ps90.OverlayValues[45] = d45
			ps90.OverlayValues[47] = d47
			ps90.OverlayValues[48] = d48
			ps90.OverlayValues[49] = d49
			ps90.OverlayValues[50] = d50
			ps90.OverlayValues[51] = d51
			ps90.OverlayValues[53] = d53
			ps90.OverlayValues[54] = d54
			ps90.OverlayValues[55] = d55
			ps90.OverlayValues[56] = d56
			ps90.OverlayValues[57] = d57
			ps90.OverlayValues[58] = d58
			ps90.OverlayValues[59] = d59
			ps90.OverlayValues[60] = d60
			ps90.OverlayValues[61] = d61
			ps90.OverlayValues[63] = d63
			ps90.OverlayValues[64] = d64
			ps90.OverlayValues[65] = d65
			ps90.OverlayValues[66] = d66
			ps90.OverlayValues[67] = d67
			ps90.OverlayValues[74] = d74
			ps90.OverlayValues[75] = d75
			ps90.OverlayValues[76] = d76
			ps90.OverlayValues[77] = d77
			ps90.OverlayValues[78] = d78
			ps90.OverlayValues[85] = d85
			ps90.OverlayValues[86] = d86
			ps90.OverlayValues[87] = d87
			ps90.OverlayValues[88] = d88
			ps90.OverlayValues[89] = d89
					return bbs[7].RenderPS(ps90)
				}
			ps91 := PhiState{General: ps.General}
			ps91.OverlayValues = make([]JITValueDesc, 90)
			ps91.OverlayValues[0] = d0
			ps91.OverlayValues[1] = d1
			ps91.OverlayValues[2] = d2
			ps91.OverlayValues[3] = d3
			ps91.OverlayValues[4] = d4
			ps91.OverlayValues[5] = d5
			ps91.OverlayValues[6] = d6
			ps91.OverlayValues[7] = d7
			ps91.OverlayValues[9] = d9
			ps91.OverlayValues[10] = d10
			ps91.OverlayValues[11] = d11
			ps91.OverlayValues[12] = d12
			ps91.OverlayValues[13] = d13
			ps91.OverlayValues[20] = d20
			ps91.OverlayValues[21] = d21
			ps91.OverlayValues[22] = d22
			ps91.OverlayValues[23] = d23
			ps91.OverlayValues[24] = d24
			ps91.OverlayValues[26] = d26
			ps91.OverlayValues[28] = d28
			ps91.OverlayValues[29] = d29
			ps91.OverlayValues[32] = d32
			ps91.OverlayValues[34] = d34
			ps91.OverlayValues[35] = d35
			ps91.OverlayValues[36] = d36
			ps91.OverlayValues[37] = d37
			ps91.OverlayValues[43] = d43
			ps91.OverlayValues[44] = d44
			ps91.OverlayValues[45] = d45
			ps91.OverlayValues[47] = d47
			ps91.OverlayValues[48] = d48
			ps91.OverlayValues[49] = d49
			ps91.OverlayValues[50] = d50
			ps91.OverlayValues[51] = d51
			ps91.OverlayValues[53] = d53
			ps91.OverlayValues[54] = d54
			ps91.OverlayValues[55] = d55
			ps91.OverlayValues[56] = d56
			ps91.OverlayValues[57] = d57
			ps91.OverlayValues[58] = d58
			ps91.OverlayValues[59] = d59
			ps91.OverlayValues[60] = d60
			ps91.OverlayValues[61] = d61
			ps91.OverlayValues[63] = d63
			ps91.OverlayValues[64] = d64
			ps91.OverlayValues[65] = d65
			ps91.OverlayValues[66] = d66
			ps91.OverlayValues[67] = d67
			ps91.OverlayValues[74] = d74
			ps91.OverlayValues[75] = d75
			ps91.OverlayValues[76] = d76
			ps91.OverlayValues[77] = d77
			ps91.OverlayValues[78] = d78
			ps91.OverlayValues[85] = d85
			ps91.OverlayValues[86] = d86
			ps91.OverlayValues[87] = d87
			ps91.OverlayValues[88] = d88
			ps91.OverlayValues[89] = d89
				return bbs[8].RenderPS(ps91)
			}
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d89.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl36)
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl36)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl37)
			ctx.W.EmitJmp(lbl9)
			ps92 := PhiState{General: true}
			ps92.OverlayValues = make([]JITValueDesc, 90)
			ps92.OverlayValues[0] = d0
			ps92.OverlayValues[1] = d1
			ps92.OverlayValues[2] = d2
			ps92.OverlayValues[3] = d3
			ps92.OverlayValues[4] = d4
			ps92.OverlayValues[5] = d5
			ps92.OverlayValues[6] = d6
			ps92.OverlayValues[7] = d7
			ps92.OverlayValues[9] = d9
			ps92.OverlayValues[10] = d10
			ps92.OverlayValues[11] = d11
			ps92.OverlayValues[12] = d12
			ps92.OverlayValues[13] = d13
			ps92.OverlayValues[20] = d20
			ps92.OverlayValues[21] = d21
			ps92.OverlayValues[22] = d22
			ps92.OverlayValues[23] = d23
			ps92.OverlayValues[24] = d24
			ps92.OverlayValues[26] = d26
			ps92.OverlayValues[28] = d28
			ps92.OverlayValues[29] = d29
			ps92.OverlayValues[32] = d32
			ps92.OverlayValues[34] = d34
			ps92.OverlayValues[35] = d35
			ps92.OverlayValues[36] = d36
			ps92.OverlayValues[37] = d37
			ps92.OverlayValues[43] = d43
			ps92.OverlayValues[44] = d44
			ps92.OverlayValues[45] = d45
			ps92.OverlayValues[47] = d47
			ps92.OverlayValues[48] = d48
			ps92.OverlayValues[49] = d49
			ps92.OverlayValues[50] = d50
			ps92.OverlayValues[51] = d51
			ps92.OverlayValues[53] = d53
			ps92.OverlayValues[54] = d54
			ps92.OverlayValues[55] = d55
			ps92.OverlayValues[56] = d56
			ps92.OverlayValues[57] = d57
			ps92.OverlayValues[58] = d58
			ps92.OverlayValues[59] = d59
			ps92.OverlayValues[60] = d60
			ps92.OverlayValues[61] = d61
			ps92.OverlayValues[63] = d63
			ps92.OverlayValues[64] = d64
			ps92.OverlayValues[65] = d65
			ps92.OverlayValues[66] = d66
			ps92.OverlayValues[67] = d67
			ps92.OverlayValues[74] = d74
			ps92.OverlayValues[75] = d75
			ps92.OverlayValues[76] = d76
			ps92.OverlayValues[77] = d77
			ps92.OverlayValues[78] = d78
			ps92.OverlayValues[85] = d85
			ps92.OverlayValues[86] = d86
			ps92.OverlayValues[87] = d87
			ps92.OverlayValues[88] = d88
			ps92.OverlayValues[89] = d89
			ps93 := PhiState{General: true}
			ps93.OverlayValues = make([]JITValueDesc, 90)
			ps93.OverlayValues[0] = d0
			ps93.OverlayValues[1] = d1
			ps93.OverlayValues[2] = d2
			ps93.OverlayValues[3] = d3
			ps93.OverlayValues[4] = d4
			ps93.OverlayValues[5] = d5
			ps93.OverlayValues[6] = d6
			ps93.OverlayValues[7] = d7
			ps93.OverlayValues[9] = d9
			ps93.OverlayValues[10] = d10
			ps93.OverlayValues[11] = d11
			ps93.OverlayValues[12] = d12
			ps93.OverlayValues[13] = d13
			ps93.OverlayValues[20] = d20
			ps93.OverlayValues[21] = d21
			ps93.OverlayValues[22] = d22
			ps93.OverlayValues[23] = d23
			ps93.OverlayValues[24] = d24
			ps93.OverlayValues[26] = d26
			ps93.OverlayValues[28] = d28
			ps93.OverlayValues[29] = d29
			ps93.OverlayValues[32] = d32
			ps93.OverlayValues[34] = d34
			ps93.OverlayValues[35] = d35
			ps93.OverlayValues[36] = d36
			ps93.OverlayValues[37] = d37
			ps93.OverlayValues[43] = d43
			ps93.OverlayValues[44] = d44
			ps93.OverlayValues[45] = d45
			ps93.OverlayValues[47] = d47
			ps93.OverlayValues[48] = d48
			ps93.OverlayValues[49] = d49
			ps93.OverlayValues[50] = d50
			ps93.OverlayValues[51] = d51
			ps93.OverlayValues[53] = d53
			ps93.OverlayValues[54] = d54
			ps93.OverlayValues[55] = d55
			ps93.OverlayValues[56] = d56
			ps93.OverlayValues[57] = d57
			ps93.OverlayValues[58] = d58
			ps93.OverlayValues[59] = d59
			ps93.OverlayValues[60] = d60
			ps93.OverlayValues[61] = d61
			ps93.OverlayValues[63] = d63
			ps93.OverlayValues[64] = d64
			ps93.OverlayValues[65] = d65
			ps93.OverlayValues[66] = d66
			ps93.OverlayValues[67] = d67
			ps93.OverlayValues[74] = d74
			ps93.OverlayValues[75] = d75
			ps93.OverlayValues[76] = d76
			ps93.OverlayValues[77] = d77
			ps93.OverlayValues[78] = d78
			ps93.OverlayValues[85] = d85
			ps93.OverlayValues[86] = d86
			ps93.OverlayValues[87] = d87
			ps93.OverlayValues[88] = d88
			ps93.OverlayValues[89] = d89
			snap94 := d1
			snap95 := d2
			snap96 := d55
			snap97 := d57
			alloc98 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps93)
			}
			ctx.RestoreAllocState(alloc98)
			d1 = snap94
			d2 = snap95
			d55 = snap96
			d57 = snap97
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps92)
			}
			return result
			ctx.FreeDesc(&d87)
			return result
			}
			bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[11].VisitCount >= 2 {
					ps.General = true
					return bbs[11].RenderPS(ps)
				}
			}
			bbs[11].VisitCount++
			if ps.General {
				if bbs[11].Rendered {
					ctx.W.EmitJmp(lbl12)
					return result
				}
				bbs[11].Rendered = true
				bbs[11].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_11 = bbs[11].Address
				ctx.W.MarkLabel(lbl12)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			ctx.W.EmitMakeInt(result, d1)
			if d1.Loc == LocReg { ctx.FreeReg(d1.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[12].VisitCount >= 2 {
					ps.General = true
					return bbs[12].RenderPS(ps)
				}
			}
			bbs[12].VisitCount++
			if ps.General {
				if bbs[12].Rendered {
					ctx.W.EmitJmp(lbl13)
					return result
				}
				bbs[12].Rendered = true
				bbs[12].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_12 = bbs[12].Address
				ctx.W.MarkLabel(lbl13)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d99 JITValueDesc
			if d1.Loc == LocImm {
				d99 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d1.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d1.Reg)
				d99 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d99)
			}
			d100 = d2
			if d100.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d100)
			ctx.EmitStoreToStack(d100, 24)
			d101 = d99
			if d101.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d101)
			ctx.EmitStoreToStack(d101, 32)
			ps102 := PhiState{General: ps.General}
			ps102.OverlayValues = make([]JITValueDesc, 102)
			ps102.OverlayValues[0] = d0
			ps102.OverlayValues[1] = d1
			ps102.OverlayValues[2] = d2
			ps102.OverlayValues[3] = d3
			ps102.OverlayValues[4] = d4
			ps102.OverlayValues[5] = d5
			ps102.OverlayValues[6] = d6
			ps102.OverlayValues[7] = d7
			ps102.OverlayValues[9] = d9
			ps102.OverlayValues[10] = d10
			ps102.OverlayValues[11] = d11
			ps102.OverlayValues[12] = d12
			ps102.OverlayValues[13] = d13
			ps102.OverlayValues[20] = d20
			ps102.OverlayValues[21] = d21
			ps102.OverlayValues[22] = d22
			ps102.OverlayValues[23] = d23
			ps102.OverlayValues[24] = d24
			ps102.OverlayValues[26] = d26
			ps102.OverlayValues[28] = d28
			ps102.OverlayValues[29] = d29
			ps102.OverlayValues[32] = d32
			ps102.OverlayValues[34] = d34
			ps102.OverlayValues[35] = d35
			ps102.OverlayValues[36] = d36
			ps102.OverlayValues[37] = d37
			ps102.OverlayValues[43] = d43
			ps102.OverlayValues[44] = d44
			ps102.OverlayValues[45] = d45
			ps102.OverlayValues[47] = d47
			ps102.OverlayValues[48] = d48
			ps102.OverlayValues[49] = d49
			ps102.OverlayValues[50] = d50
			ps102.OverlayValues[51] = d51
			ps102.OverlayValues[53] = d53
			ps102.OverlayValues[54] = d54
			ps102.OverlayValues[55] = d55
			ps102.OverlayValues[56] = d56
			ps102.OverlayValues[57] = d57
			ps102.OverlayValues[58] = d58
			ps102.OverlayValues[59] = d59
			ps102.OverlayValues[60] = d60
			ps102.OverlayValues[61] = d61
			ps102.OverlayValues[63] = d63
			ps102.OverlayValues[64] = d64
			ps102.OverlayValues[65] = d65
			ps102.OverlayValues[66] = d66
			ps102.OverlayValues[67] = d67
			ps102.OverlayValues[74] = d74
			ps102.OverlayValues[75] = d75
			ps102.OverlayValues[76] = d76
			ps102.OverlayValues[77] = d77
			ps102.OverlayValues[78] = d78
			ps102.OverlayValues[85] = d85
			ps102.OverlayValues[86] = d86
			ps102.OverlayValues[87] = d87
			ps102.OverlayValues[88] = d88
			ps102.OverlayValues[89] = d89
			ps102.OverlayValues[99] = d99
			ps102.OverlayValues[100] = d100
			ps102.OverlayValues[101] = d101
			ps102.PhiValues = make([]JITValueDesc, 2)
			d103 = d2
			ps102.PhiValues[0] = d103
			d104 = d99
			ps102.PhiValues[1] = d104
			if ps102.General && bbs[15].Rendered {
				ctx.W.EmitJmp(lbl16)
				return result
			}
			return bbs[15].RenderPS(ps102)
			return result
			}
			bbs[13].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[13].VisitCount >= 2 {
					ps.General = true
					return bbs[13].RenderPS(ps)
				}
			}
			bbs[13].VisitCount++
			if ps.General {
				if bbs[13].Rendered {
					ctx.W.EmitJmp(lbl14)
					return result
				}
				bbs[13].Rendered = true
				bbs[13].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_13 = bbs[13].Address
				ctx.W.MarkLabel(lbl14)
				ctx.W.ResolveFixups()
			}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
				d99 = ps.OverlayValues[99]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d3)
			var d105 JITValueDesc
			if d3.Loc == LocImm {
				idx := int(d3.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d105 = args[idx]
				d105.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d3.Reg] {
					ctx.ProtectReg(d3.Reg)
					seen[d3.Reg] = true
					protected = append(protected, d3.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r18 := ctx.AllocReg()
				r19 := ctx.AllocRegExcept(r18)
				lbl38 := ctx.W.ReserveLabel()
				lbl39 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d3.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl39)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r18, ai.Reg)
						ctx.W.EmitMovRegReg(r19, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r18, tmp.Reg)
						ctx.W.EmitMovRegReg(r19, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r18, Reg2: r19}
						ctx.BindReg(r18, &pair)
						ctx.BindReg(r19, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r18, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r19, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl38)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl39)
				d106 := JITValueDesc{Loc: LocRegPair, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d106)
				ctx.BindReg(r19, &d106)
				ctx.BindReg(r18, &d106)
				ctx.BindReg(r19, &d106)
				ctx.W.EmitMakeNil(d106)
				ctx.W.MarkLabel(lbl38)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d105 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d105)
				ctx.BindReg(r19, &d105)
			}
			var d107 JITValueDesc
			if d105.Loc == LocImm {
				d107 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d105.Imm.Float())}
			} else if d105.Type == tagFloat && d105.Loc == LocReg {
				d107 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d105.Reg}
				ctx.BindReg(d105.Reg, &d107)
				ctx.BindReg(d105.Reg, &d107)
			} else if d105.Type == tagFloat && d105.Loc == LocRegPair {
				ctx.FreeReg(d105.Reg)
				d107 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d105.Reg2}
				ctx.BindReg(d105.Reg2, &d107)
				ctx.BindReg(d105.Reg2, &d107)
			} else {
				d107 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d105}, 1)
				d107.Type = tagFloat
				ctx.BindReg(d107.Reg, &d107)
			}
			ctx.FreeDesc(&d105)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d107)
			var d108 JITValueDesc
			if d4.Loc == LocImm && d107.Loc == LocImm {
				d108 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float() - d107.Imm.Float())}
			} else if d4.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d107.Reg)
				_, xBits := d4.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d107.Reg)
				d108 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d108)
			} else if d107.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				_, yBits := d107.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d108 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d108)
			} else {
				r20 := ctx.AllocRegExcept(d4.Reg, d107.Reg)
				ctx.W.EmitMovRegReg(r20, d4.Reg)
				ctx.W.EmitSubFloat64(r20, d107.Reg)
				d108 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r20}
				ctx.BindReg(r20, &d108)
			}
			if d108.Loc == LocReg && d4.Loc == LocReg && d108.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.FreeDesc(&d107)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d109 JITValueDesc
			if d3.Loc == LocImm {
				d109 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(scratch, d3.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d109 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			}
			if d109.Loc == LocReg && d3.Loc == LocReg && d109.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = LocNone
			}
			d110 = d109
			if d110.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d110)
			ctx.EmitStoreToStack(d110, 24)
			d111 = d108
			if d111.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d111)
			ctx.EmitStoreToStack(d111, 32)
			ps112 := PhiState{General: ps.General}
			ps112.OverlayValues = make([]JITValueDesc, 112)
			ps112.OverlayValues[0] = d0
			ps112.OverlayValues[1] = d1
			ps112.OverlayValues[2] = d2
			ps112.OverlayValues[3] = d3
			ps112.OverlayValues[4] = d4
			ps112.OverlayValues[5] = d5
			ps112.OverlayValues[6] = d6
			ps112.OverlayValues[7] = d7
			ps112.OverlayValues[9] = d9
			ps112.OverlayValues[10] = d10
			ps112.OverlayValues[11] = d11
			ps112.OverlayValues[12] = d12
			ps112.OverlayValues[13] = d13
			ps112.OverlayValues[20] = d20
			ps112.OverlayValues[21] = d21
			ps112.OverlayValues[22] = d22
			ps112.OverlayValues[23] = d23
			ps112.OverlayValues[24] = d24
			ps112.OverlayValues[26] = d26
			ps112.OverlayValues[28] = d28
			ps112.OverlayValues[29] = d29
			ps112.OverlayValues[32] = d32
			ps112.OverlayValues[34] = d34
			ps112.OverlayValues[35] = d35
			ps112.OverlayValues[36] = d36
			ps112.OverlayValues[37] = d37
			ps112.OverlayValues[43] = d43
			ps112.OverlayValues[44] = d44
			ps112.OverlayValues[45] = d45
			ps112.OverlayValues[47] = d47
			ps112.OverlayValues[48] = d48
			ps112.OverlayValues[49] = d49
			ps112.OverlayValues[50] = d50
			ps112.OverlayValues[51] = d51
			ps112.OverlayValues[53] = d53
			ps112.OverlayValues[54] = d54
			ps112.OverlayValues[55] = d55
			ps112.OverlayValues[56] = d56
			ps112.OverlayValues[57] = d57
			ps112.OverlayValues[58] = d58
			ps112.OverlayValues[59] = d59
			ps112.OverlayValues[60] = d60
			ps112.OverlayValues[61] = d61
			ps112.OverlayValues[63] = d63
			ps112.OverlayValues[64] = d64
			ps112.OverlayValues[65] = d65
			ps112.OverlayValues[66] = d66
			ps112.OverlayValues[67] = d67
			ps112.OverlayValues[74] = d74
			ps112.OverlayValues[75] = d75
			ps112.OverlayValues[76] = d76
			ps112.OverlayValues[77] = d77
			ps112.OverlayValues[78] = d78
			ps112.OverlayValues[85] = d85
			ps112.OverlayValues[86] = d86
			ps112.OverlayValues[87] = d87
			ps112.OverlayValues[88] = d88
			ps112.OverlayValues[89] = d89
			ps112.OverlayValues[99] = d99
			ps112.OverlayValues[100] = d100
			ps112.OverlayValues[101] = d101
			ps112.OverlayValues[103] = d103
			ps112.OverlayValues[104] = d104
			ps112.OverlayValues[105] = d105
			ps112.OverlayValues[106] = d106
			ps112.OverlayValues[107] = d107
			ps112.OverlayValues[108] = d108
			ps112.OverlayValues[109] = d109
			ps112.OverlayValues[110] = d110
			ps112.OverlayValues[111] = d111
			ps112.PhiValues = make([]JITValueDesc, 2)
			d113 = d109
			ps112.PhiValues[0] = d113
			d114 = d108
			ps112.PhiValues[1] = d114
			if ps112.General && bbs[15].Rendered {
				ctx.W.EmitJmp(lbl16)
				return result
			}
			return bbs[15].RenderPS(ps112)
			return result
			}
			bbs[14].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[14].VisitCount >= 2 {
					ps.General = true
					return bbs[14].RenderPS(ps)
				}
			}
			bbs[14].VisitCount++
			if ps.General {
				if bbs[14].Rendered {
					ctx.W.EmitJmp(lbl15)
					return result
				}
				bbs[14].Rendered = true
				bbs[14].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_14 = bbs[14].Address
				ctx.W.MarkLabel(lbl15)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
				d99 = ps.OverlayValues[99]
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
			if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != LocNone {
				d113 = ps.OverlayValues[113]
			}
			if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != LocNone {
				d114 = ps.OverlayValues[114]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			ctx.W.EmitMakeFloat(result, d4)
			if d4.Loc == LocReg { ctx.FreeReg(d4.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[15].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[15].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d115 := ps.PhiValues[0]
						ctx.EnsureDesc(&d115)
						ctx.EmitStoreToStack(d115, 24)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d116 := ps.PhiValues[1]
						ctx.EnsureDesc(&d116)
						ctx.EmitStoreToStack(d116, 32)
					}
					ps.General = true
					return bbs[15].RenderPS(ps)
				}
			}
			bbs[15].VisitCount++
			if ps.General {
				if bbs[15].Rendered {
					ctx.W.EmitJmp(lbl16)
					return result
				}
				bbs[15].Rendered = true
				bbs[15].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_15 = bbs[15].Address
				ctx.W.MarkLabel(lbl16)
				ctx.W.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
				d99 = ps.OverlayValues[99]
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d3 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d4 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d117 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d117)
			var d118 JITValueDesc
			if d3.Loc == LocImm && d117.Loc == LocImm {
				d118 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d117.Imm.Int())}
			} else if d117.Loc == LocImm {
				r21 := ctx.AllocRegExcept(d3.Reg)
				if d117.Imm.Int() >= -2147483648 && d117.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(d117.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d117.Imm.Int()))
					ctx.W.EmitCmpInt64(d3.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r21, CcL)
				d118 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
				ctx.BindReg(r21, &d118)
			} else if d3.Loc == LocImm {
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d117.Reg)
				ctx.W.EmitSetcc(r22, CcL)
				d118 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d118)
			} else {
				r23 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitCmpInt64(d3.Reg, d117.Reg)
				ctx.W.EmitSetcc(r23, CcL)
				d118 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d118)
			}
			ctx.FreeDesc(&d117)
			d119 = d118
			ctx.EnsureDesc(&d119)
			if d119.Loc != LocImm && d119.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d119.Loc == LocImm {
				if d119.Imm.Bool() {
			ps120 := PhiState{General: ps.General}
			ps120.OverlayValues = make([]JITValueDesc, 120)
			ps120.OverlayValues[0] = d0
			ps120.OverlayValues[1] = d1
			ps120.OverlayValues[2] = d2
			ps120.OverlayValues[3] = d3
			ps120.OverlayValues[4] = d4
			ps120.OverlayValues[5] = d5
			ps120.OverlayValues[6] = d6
			ps120.OverlayValues[7] = d7
			ps120.OverlayValues[9] = d9
			ps120.OverlayValues[10] = d10
			ps120.OverlayValues[11] = d11
			ps120.OverlayValues[12] = d12
			ps120.OverlayValues[13] = d13
			ps120.OverlayValues[20] = d20
			ps120.OverlayValues[21] = d21
			ps120.OverlayValues[22] = d22
			ps120.OverlayValues[23] = d23
			ps120.OverlayValues[24] = d24
			ps120.OverlayValues[26] = d26
			ps120.OverlayValues[28] = d28
			ps120.OverlayValues[29] = d29
			ps120.OverlayValues[32] = d32
			ps120.OverlayValues[34] = d34
			ps120.OverlayValues[35] = d35
			ps120.OverlayValues[36] = d36
			ps120.OverlayValues[37] = d37
			ps120.OverlayValues[43] = d43
			ps120.OverlayValues[44] = d44
			ps120.OverlayValues[45] = d45
			ps120.OverlayValues[47] = d47
			ps120.OverlayValues[48] = d48
			ps120.OverlayValues[49] = d49
			ps120.OverlayValues[50] = d50
			ps120.OverlayValues[51] = d51
			ps120.OverlayValues[53] = d53
			ps120.OverlayValues[54] = d54
			ps120.OverlayValues[55] = d55
			ps120.OverlayValues[56] = d56
			ps120.OverlayValues[57] = d57
			ps120.OverlayValues[58] = d58
			ps120.OverlayValues[59] = d59
			ps120.OverlayValues[60] = d60
			ps120.OverlayValues[61] = d61
			ps120.OverlayValues[63] = d63
			ps120.OverlayValues[64] = d64
			ps120.OverlayValues[65] = d65
			ps120.OverlayValues[66] = d66
			ps120.OverlayValues[67] = d67
			ps120.OverlayValues[74] = d74
			ps120.OverlayValues[75] = d75
			ps120.OverlayValues[76] = d76
			ps120.OverlayValues[77] = d77
			ps120.OverlayValues[78] = d78
			ps120.OverlayValues[85] = d85
			ps120.OverlayValues[86] = d86
			ps120.OverlayValues[87] = d87
			ps120.OverlayValues[88] = d88
			ps120.OverlayValues[89] = d89
			ps120.OverlayValues[99] = d99
			ps120.OverlayValues[100] = d100
			ps120.OverlayValues[101] = d101
			ps120.OverlayValues[103] = d103
			ps120.OverlayValues[104] = d104
			ps120.OverlayValues[105] = d105
			ps120.OverlayValues[106] = d106
			ps120.OverlayValues[107] = d107
			ps120.OverlayValues[108] = d108
			ps120.OverlayValues[109] = d109
			ps120.OverlayValues[110] = d110
			ps120.OverlayValues[111] = d111
			ps120.OverlayValues[113] = d113
			ps120.OverlayValues[114] = d114
			ps120.OverlayValues[115] = d115
			ps120.OverlayValues[116] = d116
			ps120.OverlayValues[117] = d117
			ps120.OverlayValues[118] = d118
			ps120.OverlayValues[119] = d119
					return bbs[13].RenderPS(ps120)
				}
			ps121 := PhiState{General: ps.General}
			ps121.OverlayValues = make([]JITValueDesc, 120)
			ps121.OverlayValues[0] = d0
			ps121.OverlayValues[1] = d1
			ps121.OverlayValues[2] = d2
			ps121.OverlayValues[3] = d3
			ps121.OverlayValues[4] = d4
			ps121.OverlayValues[5] = d5
			ps121.OverlayValues[6] = d6
			ps121.OverlayValues[7] = d7
			ps121.OverlayValues[9] = d9
			ps121.OverlayValues[10] = d10
			ps121.OverlayValues[11] = d11
			ps121.OverlayValues[12] = d12
			ps121.OverlayValues[13] = d13
			ps121.OverlayValues[20] = d20
			ps121.OverlayValues[21] = d21
			ps121.OverlayValues[22] = d22
			ps121.OverlayValues[23] = d23
			ps121.OverlayValues[24] = d24
			ps121.OverlayValues[26] = d26
			ps121.OverlayValues[28] = d28
			ps121.OverlayValues[29] = d29
			ps121.OverlayValues[32] = d32
			ps121.OverlayValues[34] = d34
			ps121.OverlayValues[35] = d35
			ps121.OverlayValues[36] = d36
			ps121.OverlayValues[37] = d37
			ps121.OverlayValues[43] = d43
			ps121.OverlayValues[44] = d44
			ps121.OverlayValues[45] = d45
			ps121.OverlayValues[47] = d47
			ps121.OverlayValues[48] = d48
			ps121.OverlayValues[49] = d49
			ps121.OverlayValues[50] = d50
			ps121.OverlayValues[51] = d51
			ps121.OverlayValues[53] = d53
			ps121.OverlayValues[54] = d54
			ps121.OverlayValues[55] = d55
			ps121.OverlayValues[56] = d56
			ps121.OverlayValues[57] = d57
			ps121.OverlayValues[58] = d58
			ps121.OverlayValues[59] = d59
			ps121.OverlayValues[60] = d60
			ps121.OverlayValues[61] = d61
			ps121.OverlayValues[63] = d63
			ps121.OverlayValues[64] = d64
			ps121.OverlayValues[65] = d65
			ps121.OverlayValues[66] = d66
			ps121.OverlayValues[67] = d67
			ps121.OverlayValues[74] = d74
			ps121.OverlayValues[75] = d75
			ps121.OverlayValues[76] = d76
			ps121.OverlayValues[77] = d77
			ps121.OverlayValues[78] = d78
			ps121.OverlayValues[85] = d85
			ps121.OverlayValues[86] = d86
			ps121.OverlayValues[87] = d87
			ps121.OverlayValues[88] = d88
			ps121.OverlayValues[89] = d89
			ps121.OverlayValues[99] = d99
			ps121.OverlayValues[100] = d100
			ps121.OverlayValues[101] = d101
			ps121.OverlayValues[103] = d103
			ps121.OverlayValues[104] = d104
			ps121.OverlayValues[105] = d105
			ps121.OverlayValues[106] = d106
			ps121.OverlayValues[107] = d107
			ps121.OverlayValues[108] = d108
			ps121.OverlayValues[109] = d109
			ps121.OverlayValues[110] = d110
			ps121.OverlayValues[111] = d111
			ps121.OverlayValues[113] = d113
			ps121.OverlayValues[114] = d114
			ps121.OverlayValues[115] = d115
			ps121.OverlayValues[116] = d116
			ps121.OverlayValues[117] = d117
			ps121.OverlayValues[118] = d118
			ps121.OverlayValues[119] = d119
				return bbs[14].RenderPS(ps121)
			}
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d119.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl40)
			ctx.W.EmitJmp(lbl41)
			ctx.W.MarkLabel(lbl40)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl41)
			ctx.W.EmitJmp(lbl15)
			ps122 := PhiState{General: true}
			ps122.OverlayValues = make([]JITValueDesc, 120)
			ps122.OverlayValues[0] = d0
			ps122.OverlayValues[1] = d1
			ps122.OverlayValues[2] = d2
			ps122.OverlayValues[3] = d3
			ps122.OverlayValues[4] = d4
			ps122.OverlayValues[5] = d5
			ps122.OverlayValues[6] = d6
			ps122.OverlayValues[7] = d7
			ps122.OverlayValues[9] = d9
			ps122.OverlayValues[10] = d10
			ps122.OverlayValues[11] = d11
			ps122.OverlayValues[12] = d12
			ps122.OverlayValues[13] = d13
			ps122.OverlayValues[20] = d20
			ps122.OverlayValues[21] = d21
			ps122.OverlayValues[22] = d22
			ps122.OverlayValues[23] = d23
			ps122.OverlayValues[24] = d24
			ps122.OverlayValues[26] = d26
			ps122.OverlayValues[28] = d28
			ps122.OverlayValues[29] = d29
			ps122.OverlayValues[32] = d32
			ps122.OverlayValues[34] = d34
			ps122.OverlayValues[35] = d35
			ps122.OverlayValues[36] = d36
			ps122.OverlayValues[37] = d37
			ps122.OverlayValues[43] = d43
			ps122.OverlayValues[44] = d44
			ps122.OverlayValues[45] = d45
			ps122.OverlayValues[47] = d47
			ps122.OverlayValues[48] = d48
			ps122.OverlayValues[49] = d49
			ps122.OverlayValues[50] = d50
			ps122.OverlayValues[51] = d51
			ps122.OverlayValues[53] = d53
			ps122.OverlayValues[54] = d54
			ps122.OverlayValues[55] = d55
			ps122.OverlayValues[56] = d56
			ps122.OverlayValues[57] = d57
			ps122.OverlayValues[58] = d58
			ps122.OverlayValues[59] = d59
			ps122.OverlayValues[60] = d60
			ps122.OverlayValues[61] = d61
			ps122.OverlayValues[63] = d63
			ps122.OverlayValues[64] = d64
			ps122.OverlayValues[65] = d65
			ps122.OverlayValues[66] = d66
			ps122.OverlayValues[67] = d67
			ps122.OverlayValues[74] = d74
			ps122.OverlayValues[75] = d75
			ps122.OverlayValues[76] = d76
			ps122.OverlayValues[77] = d77
			ps122.OverlayValues[78] = d78
			ps122.OverlayValues[85] = d85
			ps122.OverlayValues[86] = d86
			ps122.OverlayValues[87] = d87
			ps122.OverlayValues[88] = d88
			ps122.OverlayValues[89] = d89
			ps122.OverlayValues[99] = d99
			ps122.OverlayValues[100] = d100
			ps122.OverlayValues[101] = d101
			ps122.OverlayValues[103] = d103
			ps122.OverlayValues[104] = d104
			ps122.OverlayValues[105] = d105
			ps122.OverlayValues[106] = d106
			ps122.OverlayValues[107] = d107
			ps122.OverlayValues[108] = d108
			ps122.OverlayValues[109] = d109
			ps122.OverlayValues[110] = d110
			ps122.OverlayValues[111] = d111
			ps122.OverlayValues[113] = d113
			ps122.OverlayValues[114] = d114
			ps122.OverlayValues[115] = d115
			ps122.OverlayValues[116] = d116
			ps122.OverlayValues[117] = d117
			ps122.OverlayValues[118] = d118
			ps122.OverlayValues[119] = d119
			ps123 := PhiState{General: true}
			ps123.OverlayValues = make([]JITValueDesc, 120)
			ps123.OverlayValues[0] = d0
			ps123.OverlayValues[1] = d1
			ps123.OverlayValues[2] = d2
			ps123.OverlayValues[3] = d3
			ps123.OverlayValues[4] = d4
			ps123.OverlayValues[5] = d5
			ps123.OverlayValues[6] = d6
			ps123.OverlayValues[7] = d7
			ps123.OverlayValues[9] = d9
			ps123.OverlayValues[10] = d10
			ps123.OverlayValues[11] = d11
			ps123.OverlayValues[12] = d12
			ps123.OverlayValues[13] = d13
			ps123.OverlayValues[20] = d20
			ps123.OverlayValues[21] = d21
			ps123.OverlayValues[22] = d22
			ps123.OverlayValues[23] = d23
			ps123.OverlayValues[24] = d24
			ps123.OverlayValues[26] = d26
			ps123.OverlayValues[28] = d28
			ps123.OverlayValues[29] = d29
			ps123.OverlayValues[32] = d32
			ps123.OverlayValues[34] = d34
			ps123.OverlayValues[35] = d35
			ps123.OverlayValues[36] = d36
			ps123.OverlayValues[37] = d37
			ps123.OverlayValues[43] = d43
			ps123.OverlayValues[44] = d44
			ps123.OverlayValues[45] = d45
			ps123.OverlayValues[47] = d47
			ps123.OverlayValues[48] = d48
			ps123.OverlayValues[49] = d49
			ps123.OverlayValues[50] = d50
			ps123.OverlayValues[51] = d51
			ps123.OverlayValues[53] = d53
			ps123.OverlayValues[54] = d54
			ps123.OverlayValues[55] = d55
			ps123.OverlayValues[56] = d56
			ps123.OverlayValues[57] = d57
			ps123.OverlayValues[58] = d58
			ps123.OverlayValues[59] = d59
			ps123.OverlayValues[60] = d60
			ps123.OverlayValues[61] = d61
			ps123.OverlayValues[63] = d63
			ps123.OverlayValues[64] = d64
			ps123.OverlayValues[65] = d65
			ps123.OverlayValues[66] = d66
			ps123.OverlayValues[67] = d67
			ps123.OverlayValues[74] = d74
			ps123.OverlayValues[75] = d75
			ps123.OverlayValues[76] = d76
			ps123.OverlayValues[77] = d77
			ps123.OverlayValues[78] = d78
			ps123.OverlayValues[85] = d85
			ps123.OverlayValues[86] = d86
			ps123.OverlayValues[87] = d87
			ps123.OverlayValues[88] = d88
			ps123.OverlayValues[89] = d89
			ps123.OverlayValues[99] = d99
			ps123.OverlayValues[100] = d100
			ps123.OverlayValues[101] = d101
			ps123.OverlayValues[103] = d103
			ps123.OverlayValues[104] = d104
			ps123.OverlayValues[105] = d105
			ps123.OverlayValues[106] = d106
			ps123.OverlayValues[107] = d107
			ps123.OverlayValues[108] = d108
			ps123.OverlayValues[109] = d109
			ps123.OverlayValues[110] = d110
			ps123.OverlayValues[111] = d111
			ps123.OverlayValues[113] = d113
			ps123.OverlayValues[114] = d114
			ps123.OverlayValues[115] = d115
			ps123.OverlayValues[116] = d116
			ps123.OverlayValues[117] = d117
			ps123.OverlayValues[118] = d118
			ps123.OverlayValues[119] = d119
			snap124 := d3
			snap125 := d4
			snap126 := d105
			snap127 := d107
			alloc128 := ctx.SnapshotAllocState()
			if !bbs[14].Rendered {
				bbs[14].RenderPS(ps123)
			}
			ctx.RestoreAllocState(alloc128)
			d3 = snap124
			d4 = snap125
			d105 = snap126
			d107 = snap127
			if !bbs[13].Rendered {
				return bbs[13].RenderPS(ps122)
			}
			return result
			ctx.FreeDesc(&d118)
			return result
			}
			bbs[16].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[16].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d129 := ps.PhiValues[0]
						ctx.EnsureDesc(&d129)
						ctx.EmitStoreToStack(d129, 40)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d130 := ps.PhiValues[1]
						ctx.EnsureDesc(&d130)
						ctx.EmitStoreToStack(d130, 48)
					}
					ps.General = true
					return bbs[16].RenderPS(ps)
				}
			}
			bbs[16].VisitCount++
			if ps.General {
				if bbs[16].Rendered {
					ctx.W.EmitJmp(lbl17)
					return result
				}
				bbs[16].Rendered = true
				bbs[16].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_16 = bbs[16].Address
				ctx.W.MarkLabel(lbl17)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
				d99 = ps.OverlayValues[99]
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
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
				d130 = ps.OverlayValues[130]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d5 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d6 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d131 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d131)
			var d132 JITValueDesc
			if d6.Loc == LocImm && d131.Loc == LocImm {
				d132 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() < d131.Imm.Int())}
			} else if d131.Loc == LocImm {
				r24 := ctx.AllocRegExcept(d6.Reg)
				if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d6.Reg, int32(d131.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d131.Imm.Int()))
					ctx.W.EmitCmpInt64(d6.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r24, CcL)
				d132 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
				ctx.BindReg(r24, &d132)
			} else if d6.Loc == LocImm {
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d131.Reg)
				ctx.W.EmitSetcc(r25, CcL)
				d132 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
				ctx.BindReg(r25, &d132)
			} else {
				r26 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitCmpInt64(d6.Reg, d131.Reg)
				ctx.W.EmitSetcc(r26, CcL)
				d132 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
				ctx.BindReg(r26, &d132)
			}
			ctx.FreeDesc(&d131)
			d133 = d132
			ctx.EnsureDesc(&d133)
			if d133.Loc != LocImm && d133.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d133.Loc == LocImm {
				if d133.Imm.Bool() {
			ps134 := PhiState{General: ps.General}
			ps134.OverlayValues = make([]JITValueDesc, 134)
			ps134.OverlayValues[0] = d0
			ps134.OverlayValues[1] = d1
			ps134.OverlayValues[2] = d2
			ps134.OverlayValues[3] = d3
			ps134.OverlayValues[4] = d4
			ps134.OverlayValues[5] = d5
			ps134.OverlayValues[6] = d6
			ps134.OverlayValues[7] = d7
			ps134.OverlayValues[9] = d9
			ps134.OverlayValues[10] = d10
			ps134.OverlayValues[11] = d11
			ps134.OverlayValues[12] = d12
			ps134.OverlayValues[13] = d13
			ps134.OverlayValues[20] = d20
			ps134.OverlayValues[21] = d21
			ps134.OverlayValues[22] = d22
			ps134.OverlayValues[23] = d23
			ps134.OverlayValues[24] = d24
			ps134.OverlayValues[26] = d26
			ps134.OverlayValues[28] = d28
			ps134.OverlayValues[29] = d29
			ps134.OverlayValues[32] = d32
			ps134.OverlayValues[34] = d34
			ps134.OverlayValues[35] = d35
			ps134.OverlayValues[36] = d36
			ps134.OverlayValues[37] = d37
			ps134.OverlayValues[43] = d43
			ps134.OverlayValues[44] = d44
			ps134.OverlayValues[45] = d45
			ps134.OverlayValues[47] = d47
			ps134.OverlayValues[48] = d48
			ps134.OverlayValues[49] = d49
			ps134.OverlayValues[50] = d50
			ps134.OverlayValues[51] = d51
			ps134.OverlayValues[53] = d53
			ps134.OverlayValues[54] = d54
			ps134.OverlayValues[55] = d55
			ps134.OverlayValues[56] = d56
			ps134.OverlayValues[57] = d57
			ps134.OverlayValues[58] = d58
			ps134.OverlayValues[59] = d59
			ps134.OverlayValues[60] = d60
			ps134.OverlayValues[61] = d61
			ps134.OverlayValues[63] = d63
			ps134.OverlayValues[64] = d64
			ps134.OverlayValues[65] = d65
			ps134.OverlayValues[66] = d66
			ps134.OverlayValues[67] = d67
			ps134.OverlayValues[74] = d74
			ps134.OverlayValues[75] = d75
			ps134.OverlayValues[76] = d76
			ps134.OverlayValues[77] = d77
			ps134.OverlayValues[78] = d78
			ps134.OverlayValues[85] = d85
			ps134.OverlayValues[86] = d86
			ps134.OverlayValues[87] = d87
			ps134.OverlayValues[88] = d88
			ps134.OverlayValues[89] = d89
			ps134.OverlayValues[99] = d99
			ps134.OverlayValues[100] = d100
			ps134.OverlayValues[101] = d101
			ps134.OverlayValues[103] = d103
			ps134.OverlayValues[104] = d104
			ps134.OverlayValues[105] = d105
			ps134.OverlayValues[106] = d106
			ps134.OverlayValues[107] = d107
			ps134.OverlayValues[108] = d108
			ps134.OverlayValues[109] = d109
			ps134.OverlayValues[110] = d110
			ps134.OverlayValues[111] = d111
			ps134.OverlayValues[113] = d113
			ps134.OverlayValues[114] = d114
			ps134.OverlayValues[115] = d115
			ps134.OverlayValues[116] = d116
			ps134.OverlayValues[117] = d117
			ps134.OverlayValues[118] = d118
			ps134.OverlayValues[119] = d119
			ps134.OverlayValues[129] = d129
			ps134.OverlayValues[130] = d130
			ps134.OverlayValues[131] = d131
			ps134.OverlayValues[132] = d132
			ps134.OverlayValues[133] = d133
					return bbs[17].RenderPS(ps134)
				}
			ps135 := PhiState{General: ps.General}
			ps135.OverlayValues = make([]JITValueDesc, 134)
			ps135.OverlayValues[0] = d0
			ps135.OverlayValues[1] = d1
			ps135.OverlayValues[2] = d2
			ps135.OverlayValues[3] = d3
			ps135.OverlayValues[4] = d4
			ps135.OverlayValues[5] = d5
			ps135.OverlayValues[6] = d6
			ps135.OverlayValues[7] = d7
			ps135.OverlayValues[9] = d9
			ps135.OverlayValues[10] = d10
			ps135.OverlayValues[11] = d11
			ps135.OverlayValues[12] = d12
			ps135.OverlayValues[13] = d13
			ps135.OverlayValues[20] = d20
			ps135.OverlayValues[21] = d21
			ps135.OverlayValues[22] = d22
			ps135.OverlayValues[23] = d23
			ps135.OverlayValues[24] = d24
			ps135.OverlayValues[26] = d26
			ps135.OverlayValues[28] = d28
			ps135.OverlayValues[29] = d29
			ps135.OverlayValues[32] = d32
			ps135.OverlayValues[34] = d34
			ps135.OverlayValues[35] = d35
			ps135.OverlayValues[36] = d36
			ps135.OverlayValues[37] = d37
			ps135.OverlayValues[43] = d43
			ps135.OverlayValues[44] = d44
			ps135.OverlayValues[45] = d45
			ps135.OverlayValues[47] = d47
			ps135.OverlayValues[48] = d48
			ps135.OverlayValues[49] = d49
			ps135.OverlayValues[50] = d50
			ps135.OverlayValues[51] = d51
			ps135.OverlayValues[53] = d53
			ps135.OverlayValues[54] = d54
			ps135.OverlayValues[55] = d55
			ps135.OverlayValues[56] = d56
			ps135.OverlayValues[57] = d57
			ps135.OverlayValues[58] = d58
			ps135.OverlayValues[59] = d59
			ps135.OverlayValues[60] = d60
			ps135.OverlayValues[61] = d61
			ps135.OverlayValues[63] = d63
			ps135.OverlayValues[64] = d64
			ps135.OverlayValues[65] = d65
			ps135.OverlayValues[66] = d66
			ps135.OverlayValues[67] = d67
			ps135.OverlayValues[74] = d74
			ps135.OverlayValues[75] = d75
			ps135.OverlayValues[76] = d76
			ps135.OverlayValues[77] = d77
			ps135.OverlayValues[78] = d78
			ps135.OverlayValues[85] = d85
			ps135.OverlayValues[86] = d86
			ps135.OverlayValues[87] = d87
			ps135.OverlayValues[88] = d88
			ps135.OverlayValues[89] = d89
			ps135.OverlayValues[99] = d99
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
			ps135.OverlayValues[113] = d113
			ps135.OverlayValues[114] = d114
			ps135.OverlayValues[115] = d115
			ps135.OverlayValues[116] = d116
			ps135.OverlayValues[117] = d117
			ps135.OverlayValues[118] = d118
			ps135.OverlayValues[119] = d119
			ps135.OverlayValues[129] = d129
			ps135.OverlayValues[130] = d130
			ps135.OverlayValues[131] = d131
			ps135.OverlayValues[132] = d132
			ps135.OverlayValues[133] = d133
				return bbs[18].RenderPS(ps135)
			}
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d133.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl42)
			ctx.W.EmitJmp(lbl43)
			ctx.W.MarkLabel(lbl42)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl43)
			ctx.W.EmitJmp(lbl19)
			ps136 := PhiState{General: true}
			ps136.OverlayValues = make([]JITValueDesc, 134)
			ps136.OverlayValues[0] = d0
			ps136.OverlayValues[1] = d1
			ps136.OverlayValues[2] = d2
			ps136.OverlayValues[3] = d3
			ps136.OverlayValues[4] = d4
			ps136.OverlayValues[5] = d5
			ps136.OverlayValues[6] = d6
			ps136.OverlayValues[7] = d7
			ps136.OverlayValues[9] = d9
			ps136.OverlayValues[10] = d10
			ps136.OverlayValues[11] = d11
			ps136.OverlayValues[12] = d12
			ps136.OverlayValues[13] = d13
			ps136.OverlayValues[20] = d20
			ps136.OverlayValues[21] = d21
			ps136.OverlayValues[22] = d22
			ps136.OverlayValues[23] = d23
			ps136.OverlayValues[24] = d24
			ps136.OverlayValues[26] = d26
			ps136.OverlayValues[28] = d28
			ps136.OverlayValues[29] = d29
			ps136.OverlayValues[32] = d32
			ps136.OverlayValues[34] = d34
			ps136.OverlayValues[35] = d35
			ps136.OverlayValues[36] = d36
			ps136.OverlayValues[37] = d37
			ps136.OverlayValues[43] = d43
			ps136.OverlayValues[44] = d44
			ps136.OverlayValues[45] = d45
			ps136.OverlayValues[47] = d47
			ps136.OverlayValues[48] = d48
			ps136.OverlayValues[49] = d49
			ps136.OverlayValues[50] = d50
			ps136.OverlayValues[51] = d51
			ps136.OverlayValues[53] = d53
			ps136.OverlayValues[54] = d54
			ps136.OverlayValues[55] = d55
			ps136.OverlayValues[56] = d56
			ps136.OverlayValues[57] = d57
			ps136.OverlayValues[58] = d58
			ps136.OverlayValues[59] = d59
			ps136.OverlayValues[60] = d60
			ps136.OverlayValues[61] = d61
			ps136.OverlayValues[63] = d63
			ps136.OverlayValues[64] = d64
			ps136.OverlayValues[65] = d65
			ps136.OverlayValues[66] = d66
			ps136.OverlayValues[67] = d67
			ps136.OverlayValues[74] = d74
			ps136.OverlayValues[75] = d75
			ps136.OverlayValues[76] = d76
			ps136.OverlayValues[77] = d77
			ps136.OverlayValues[78] = d78
			ps136.OverlayValues[85] = d85
			ps136.OverlayValues[86] = d86
			ps136.OverlayValues[87] = d87
			ps136.OverlayValues[88] = d88
			ps136.OverlayValues[89] = d89
			ps136.OverlayValues[99] = d99
			ps136.OverlayValues[100] = d100
			ps136.OverlayValues[101] = d101
			ps136.OverlayValues[103] = d103
			ps136.OverlayValues[104] = d104
			ps136.OverlayValues[105] = d105
			ps136.OverlayValues[106] = d106
			ps136.OverlayValues[107] = d107
			ps136.OverlayValues[108] = d108
			ps136.OverlayValues[109] = d109
			ps136.OverlayValues[110] = d110
			ps136.OverlayValues[111] = d111
			ps136.OverlayValues[113] = d113
			ps136.OverlayValues[114] = d114
			ps136.OverlayValues[115] = d115
			ps136.OverlayValues[116] = d116
			ps136.OverlayValues[117] = d117
			ps136.OverlayValues[118] = d118
			ps136.OverlayValues[119] = d119
			ps136.OverlayValues[129] = d129
			ps136.OverlayValues[130] = d130
			ps136.OverlayValues[131] = d131
			ps136.OverlayValues[132] = d132
			ps136.OverlayValues[133] = d133
			ps137 := PhiState{General: true}
			ps137.OverlayValues = make([]JITValueDesc, 134)
			ps137.OverlayValues[0] = d0
			ps137.OverlayValues[1] = d1
			ps137.OverlayValues[2] = d2
			ps137.OverlayValues[3] = d3
			ps137.OverlayValues[4] = d4
			ps137.OverlayValues[5] = d5
			ps137.OverlayValues[6] = d6
			ps137.OverlayValues[7] = d7
			ps137.OverlayValues[9] = d9
			ps137.OverlayValues[10] = d10
			ps137.OverlayValues[11] = d11
			ps137.OverlayValues[12] = d12
			ps137.OverlayValues[13] = d13
			ps137.OverlayValues[20] = d20
			ps137.OverlayValues[21] = d21
			ps137.OverlayValues[22] = d22
			ps137.OverlayValues[23] = d23
			ps137.OverlayValues[24] = d24
			ps137.OverlayValues[26] = d26
			ps137.OverlayValues[28] = d28
			ps137.OverlayValues[29] = d29
			ps137.OverlayValues[32] = d32
			ps137.OverlayValues[34] = d34
			ps137.OverlayValues[35] = d35
			ps137.OverlayValues[36] = d36
			ps137.OverlayValues[37] = d37
			ps137.OverlayValues[43] = d43
			ps137.OverlayValues[44] = d44
			ps137.OverlayValues[45] = d45
			ps137.OverlayValues[47] = d47
			ps137.OverlayValues[48] = d48
			ps137.OverlayValues[49] = d49
			ps137.OverlayValues[50] = d50
			ps137.OverlayValues[51] = d51
			ps137.OverlayValues[53] = d53
			ps137.OverlayValues[54] = d54
			ps137.OverlayValues[55] = d55
			ps137.OverlayValues[56] = d56
			ps137.OverlayValues[57] = d57
			ps137.OverlayValues[58] = d58
			ps137.OverlayValues[59] = d59
			ps137.OverlayValues[60] = d60
			ps137.OverlayValues[61] = d61
			ps137.OverlayValues[63] = d63
			ps137.OverlayValues[64] = d64
			ps137.OverlayValues[65] = d65
			ps137.OverlayValues[66] = d66
			ps137.OverlayValues[67] = d67
			ps137.OverlayValues[74] = d74
			ps137.OverlayValues[75] = d75
			ps137.OverlayValues[76] = d76
			ps137.OverlayValues[77] = d77
			ps137.OverlayValues[78] = d78
			ps137.OverlayValues[85] = d85
			ps137.OverlayValues[86] = d86
			ps137.OverlayValues[87] = d87
			ps137.OverlayValues[88] = d88
			ps137.OverlayValues[89] = d89
			ps137.OverlayValues[99] = d99
			ps137.OverlayValues[100] = d100
			ps137.OverlayValues[101] = d101
			ps137.OverlayValues[103] = d103
			ps137.OverlayValues[104] = d104
			ps137.OverlayValues[105] = d105
			ps137.OverlayValues[106] = d106
			ps137.OverlayValues[107] = d107
			ps137.OverlayValues[108] = d108
			ps137.OverlayValues[109] = d109
			ps137.OverlayValues[110] = d110
			ps137.OverlayValues[111] = d111
			ps137.OverlayValues[113] = d113
			ps137.OverlayValues[114] = d114
			ps137.OverlayValues[115] = d115
			ps137.OverlayValues[116] = d116
			ps137.OverlayValues[117] = d117
			ps137.OverlayValues[118] = d118
			ps137.OverlayValues[119] = d119
			ps137.OverlayValues[129] = d129
			ps137.OverlayValues[130] = d130
			ps137.OverlayValues[131] = d131
			ps137.OverlayValues[132] = d132
			ps137.OverlayValues[133] = d133
			snap138 := d5
			snap139 := d6
			alloc140 := ctx.SnapshotAllocState()
			if !bbs[18].Rendered {
				bbs[18].RenderPS(ps137)
			}
			ctx.RestoreAllocState(alloc140)
			d5 = snap138
			d6 = snap139
			if !bbs[17].Rendered {
				return bbs[17].RenderPS(ps136)
			}
			return result
			ctx.FreeDesc(&d132)
			return result
			}
			bbs[17].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[17].VisitCount >= 2 {
					ps.General = true
					return bbs[17].RenderPS(ps)
				}
			}
			bbs[17].VisitCount++
			if ps.General {
				if bbs[17].Rendered {
					ctx.W.EmitJmp(lbl18)
					return result
				}
				bbs[17].Rendered = true
				bbs[17].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_17 = bbs[17].Address
				ctx.W.MarkLabel(lbl18)
				ctx.W.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
				d99 = ps.OverlayValues[99]
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
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
				d130 = ps.OverlayValues[130]
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
			ctx.EnsureDesc(&d6)
			var d141 JITValueDesc
			if d6.Loc == LocImm {
				idx := int(d6.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d141 = args[idx]
				d141.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d6.Reg] {
					ctx.ProtectReg(d6.Reg)
					seen[d6.Reg] = true
					protected = append(protected, d6.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r27 := ctx.AllocReg()
				r28 := ctx.AllocRegExcept(r27)
				lbl44 := ctx.W.ReserveLabel()
				lbl45 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d6.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl45)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d6.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r27, ai.Reg)
						ctx.W.EmitMovRegReg(r28, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r27, tmp.Reg)
						ctx.W.EmitMovRegReg(r28, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r27, Reg2: r28}
						ctx.BindReg(r27, &pair)
						ctx.BindReg(r28, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r27, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r28, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl44)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl45)
				d142 := JITValueDesc{Loc: LocRegPair, Reg: r27, Reg2: r28}
				ctx.BindReg(r27, &d142)
				ctx.BindReg(r28, &d142)
				ctx.BindReg(r27, &d142)
				ctx.BindReg(r28, &d142)
				ctx.W.EmitMakeNil(d142)
				ctx.W.MarkLabel(lbl44)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d141 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r27, Reg2: r28}
				ctx.BindReg(r27, &d141)
				ctx.BindReg(r28, &d141)
			}
			var d143 JITValueDesc
			if d141.Loc == LocImm {
				d143 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d141.Imm.Float())}
			} else if d141.Type == tagFloat && d141.Loc == LocReg {
				d143 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d141.Reg}
				ctx.BindReg(d141.Reg, &d143)
				ctx.BindReg(d141.Reg, &d143)
			} else if d141.Type == tagFloat && d141.Loc == LocRegPair {
				ctx.FreeReg(d141.Reg)
				d143 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d141.Reg2}
				ctx.BindReg(d141.Reg2, &d143)
				ctx.BindReg(d141.Reg2, &d143)
			} else {
				d143 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d141}, 1)
				d143.Type = tagFloat
				ctx.BindReg(d143.Reg, &d143)
			}
			ctx.FreeDesc(&d141)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d143)
			var d144 JITValueDesc
			if d5.Loc == LocImm && d143.Loc == LocImm {
				d144 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d5.Imm.Float() - d143.Imm.Float())}
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d143.Reg)
				_, xBits := d5.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitSubFloat64(scratch, d143.Reg)
				d144 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else if d143.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				_, yBits := d143.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitSubFloat64(scratch, RegR11)
				d144 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else {
				r29 := ctx.AllocRegExcept(d5.Reg, d143.Reg)
				ctx.W.EmitMovRegReg(r29, d5.Reg)
				ctx.W.EmitSubFloat64(r29, d143.Reg)
				d144 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r29}
				ctx.BindReg(r29, &d144)
			}
			if d144.Loc == LocReg && d5.Loc == LocReg && d144.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d143)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			var d145 JITValueDesc
			if d6.Loc == LocImm {
				d145 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d145 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d145)
			}
			if d145.Loc == LocReg && d6.Loc == LocReg && d145.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = LocNone
			}
			d146 = d144
			if d146.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d146)
			ctx.EmitStoreToStack(d146, 40)
			d147 = d145
			if d147.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d147)
			ctx.EmitStoreToStack(d147, 48)
			ps148 := PhiState{General: ps.General}
			ps148.OverlayValues = make([]JITValueDesc, 148)
			ps148.OverlayValues[0] = d0
			ps148.OverlayValues[1] = d1
			ps148.OverlayValues[2] = d2
			ps148.OverlayValues[3] = d3
			ps148.OverlayValues[4] = d4
			ps148.OverlayValues[5] = d5
			ps148.OverlayValues[6] = d6
			ps148.OverlayValues[7] = d7
			ps148.OverlayValues[9] = d9
			ps148.OverlayValues[10] = d10
			ps148.OverlayValues[11] = d11
			ps148.OverlayValues[12] = d12
			ps148.OverlayValues[13] = d13
			ps148.OverlayValues[20] = d20
			ps148.OverlayValues[21] = d21
			ps148.OverlayValues[22] = d22
			ps148.OverlayValues[23] = d23
			ps148.OverlayValues[24] = d24
			ps148.OverlayValues[26] = d26
			ps148.OverlayValues[28] = d28
			ps148.OverlayValues[29] = d29
			ps148.OverlayValues[32] = d32
			ps148.OverlayValues[34] = d34
			ps148.OverlayValues[35] = d35
			ps148.OverlayValues[36] = d36
			ps148.OverlayValues[37] = d37
			ps148.OverlayValues[43] = d43
			ps148.OverlayValues[44] = d44
			ps148.OverlayValues[45] = d45
			ps148.OverlayValues[47] = d47
			ps148.OverlayValues[48] = d48
			ps148.OverlayValues[49] = d49
			ps148.OverlayValues[50] = d50
			ps148.OverlayValues[51] = d51
			ps148.OverlayValues[53] = d53
			ps148.OverlayValues[54] = d54
			ps148.OverlayValues[55] = d55
			ps148.OverlayValues[56] = d56
			ps148.OverlayValues[57] = d57
			ps148.OverlayValues[58] = d58
			ps148.OverlayValues[59] = d59
			ps148.OverlayValues[60] = d60
			ps148.OverlayValues[61] = d61
			ps148.OverlayValues[63] = d63
			ps148.OverlayValues[64] = d64
			ps148.OverlayValues[65] = d65
			ps148.OverlayValues[66] = d66
			ps148.OverlayValues[67] = d67
			ps148.OverlayValues[74] = d74
			ps148.OverlayValues[75] = d75
			ps148.OverlayValues[76] = d76
			ps148.OverlayValues[77] = d77
			ps148.OverlayValues[78] = d78
			ps148.OverlayValues[85] = d85
			ps148.OverlayValues[86] = d86
			ps148.OverlayValues[87] = d87
			ps148.OverlayValues[88] = d88
			ps148.OverlayValues[89] = d89
			ps148.OverlayValues[99] = d99
			ps148.OverlayValues[100] = d100
			ps148.OverlayValues[101] = d101
			ps148.OverlayValues[103] = d103
			ps148.OverlayValues[104] = d104
			ps148.OverlayValues[105] = d105
			ps148.OverlayValues[106] = d106
			ps148.OverlayValues[107] = d107
			ps148.OverlayValues[108] = d108
			ps148.OverlayValues[109] = d109
			ps148.OverlayValues[110] = d110
			ps148.OverlayValues[111] = d111
			ps148.OverlayValues[113] = d113
			ps148.OverlayValues[114] = d114
			ps148.OverlayValues[115] = d115
			ps148.OverlayValues[116] = d116
			ps148.OverlayValues[117] = d117
			ps148.OverlayValues[118] = d118
			ps148.OverlayValues[119] = d119
			ps148.OverlayValues[129] = d129
			ps148.OverlayValues[130] = d130
			ps148.OverlayValues[131] = d131
			ps148.OverlayValues[132] = d132
			ps148.OverlayValues[133] = d133
			ps148.OverlayValues[141] = d141
			ps148.OverlayValues[142] = d142
			ps148.OverlayValues[143] = d143
			ps148.OverlayValues[144] = d144
			ps148.OverlayValues[145] = d145
			ps148.OverlayValues[146] = d146
			ps148.OverlayValues[147] = d147
			ps148.PhiValues = make([]JITValueDesc, 2)
			d149 = d144
			ps148.PhiValues[0] = d149
			d150 = d145
			ps148.PhiValues[1] = d150
			if ps148.General && bbs[16].Rendered {
				ctx.W.EmitJmp(lbl17)
				return result
			}
			return bbs[16].RenderPS(ps148)
			return result
			}
			bbs[18].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[18].VisitCount >= 2 {
					ps.General = true
					return bbs[18].RenderPS(ps)
				}
			}
			bbs[18].VisitCount++
			if ps.General {
				if bbs[18].Rendered {
					ctx.W.EmitJmp(lbl19)
					return result
				}
				bbs[18].Rendered = true
				bbs[18].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_18 = bbs[18].Address
				ctx.W.MarkLabel(lbl19)
				ctx.W.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d6 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
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
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
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
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
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
			if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != LocNone {
				d99 = ps.OverlayValues[99]
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
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
				d118 = ps.OverlayValues[118]
			}
			if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != LocNone {
				d119 = ps.OverlayValues[119]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != LocNone {
				d129 = ps.OverlayValues[129]
			}
			if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != LocNone {
				d130 = ps.OverlayValues[130]
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
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != LocNone {
				d142 = ps.OverlayValues[142]
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
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			ctx.W.EmitMakeFloat(result, d5)
			if d5.Loc == LocReg { ctx.FreeReg(d5.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			return result
			}
			ps151 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps151)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(56))
			ctx.W.EmitAddRSP32(int32(56))
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"*", "multiplies two or more numbers",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit (SQL-style): if any arg is nil, result is nil
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			// Try integer mode: treat float operands with zero fractional part as integers
			prodInt := int64(1)
			i := 0
			for ; i < len(a); i++ {
				v := a[i]
				if v.IsInt() {
					prodInt *= v.Int()
					continue
				}
				if v.IsFloat() {
					f := v.Float()
					if f == math.Trunc(f) {
						prodInt *= int64(f)
						continue
					}
				}
				break // non-integer number encountered -> switch to float mode
			}
			if i == len(a) {
				return NewInt(prodInt)
			}
			// Float mode: include any prior integer product and continue in float
			prodFloat := float64(prodInt)
			for ; i < len(a); i++ {
				prodFloat *= a[i].Float()
			}
			return NewFloat(prodFloat)
		},
		true, false, &TypeDescriptor{Optimize: optimizeAssociative},
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d6 JITValueDesc
			_ = d6
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
			var d25 JITValueDesc
			_ = d25
			var d27 JITValueDesc
			_ = d27
			var d28 JITValueDesc
			_ = d28
			var d31 JITValueDesc
			_ = d31
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
			var d48 JITValueDesc
			_ = d48
			var d49 JITValueDesc
			_ = d49
			var d50 JITValueDesc
			_ = d50
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
			var d70 JITValueDesc
			_ = d70
			var d71 JITValueDesc
			_ = d71
			var d72 JITValueDesc
			_ = d72
			var d73 JITValueDesc
			_ = d73
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
			var d84 JITValueDesc
			_ = d84
			var d91 JITValueDesc
			_ = d91
			var d92 JITValueDesc
			_ = d92
			var d93 JITValueDesc
			_ = d93
			var d94 JITValueDesc
			_ = d94
			var d102 JITValueDesc
			_ = d102
			var d103 JITValueDesc
			_ = d103
			var d104 JITValueDesc
			_ = d104
			var d106 JITValueDesc
			_ = d106
			var d107 JITValueDesc
			_ = d107
			var d108 JITValueDesc
			_ = d108
			var d109 JITValueDesc
			_ = d109
			var d111 JITValueDesc
			_ = d111
			var d112 JITValueDesc
			_ = d112
			var d113 JITValueDesc
			_ = d113
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			var bbs [18]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			lbl9 := ctx.W.ReserveLabel()
			bbpos_0_9 := int32(-1)
			_ = bbpos_0_9
			lbl10 := ctx.W.ReserveLabel()
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.W.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.W.ReserveLabel()
			bbpos_0_12 := int32(-1)
			_ = bbpos_0_12
			lbl13 := ctx.W.ReserveLabel()
			bbpos_0_13 := int32(-1)
			_ = bbpos_0_13
			lbl14 := ctx.W.ReserveLabel()
			bbpos_0_14 := int32(-1)
			_ = bbpos_0_14
			lbl15 := ctx.W.ReserveLabel()
			bbpos_0_15 := int32(-1)
			_ = bbpos_0_15
			lbl16 := ctx.W.ReserveLabel()
			bbpos_0_16 := int32(-1)
			_ = bbpos_0_16
			lbl17 := ctx.W.ReserveLabel()
			bbpos_0_17 := int32(-1)
			_ = bbpos_0_17
			lbl18 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			ctx.ReclaimUntrackedRegs()
			d6 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 0)
			ps7 := PhiState{General: ps.General}
			ps7.OverlayValues = make([]JITValueDesc, 7)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			ps7.OverlayValues[4] = d4
			ps7.OverlayValues[5] = d5
			ps7.OverlayValues[6] = d6
			ps7.PhiValues = make([]JITValueDesc, 1)
			d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
			ps7.PhiValues[0] = d8
			if ps7.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps7)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d9 := ps.PhiValues[0]
						ctx.EnsureDesc(&d9)
						ctx.EmitStoreToStack(d9, 0)
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d10 JITValueDesc
			if d0.Loc == LocImm {
				d10 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d10 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d10)
			}
			if d10.Loc == LocReg && d0.Loc == LocReg && d10.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d6)
			var d11 JITValueDesc
			if d10.Loc == LocImm && d6.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Int() < d6.Imm.Int())}
			} else if d6.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d10.Reg)
				if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d10.Reg, int32(d6.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
					ctx.W.EmitCmpInt64(d10.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d11)
			} else if d10.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d6.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d11)
			} else {
				r3 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitCmpInt64(d10.Reg, d6.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d11)
			}
			ctx.FreeDesc(&d6)
			d12 = d11
			ctx.EnsureDesc(&d12)
			if d12.Loc != LocImm && d12.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d12.Loc == LocImm {
				if d12.Imm.Bool() {
			ps13 := PhiState{General: ps.General}
			ps13.OverlayValues = make([]JITValueDesc, 13)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[3] = d3
			ps13.OverlayValues[4] = d4
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[6] = d6
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
			ps13.OverlayValues[12] = d12
					return bbs[2].RenderPS(ps13)
				}
			ps14 := PhiState{General: ps.General}
			ps14.OverlayValues = make([]JITValueDesc, 13)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[6] = d6
			ps14.OverlayValues[8] = d8
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
				return bbs[3].RenderPS(ps14)
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d12.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl19)
			ctx.W.EmitJmp(lbl20)
			ctx.W.MarkLabel(lbl19)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl20)
			ctx.W.EmitJmp(lbl4)
			ps15 := PhiState{General: true}
			ps15.OverlayValues = make([]JITValueDesc, 13)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[8] = d8
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			ps15.OverlayValues[12] = d12
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 13)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[3] = d3
			ps16.OverlayValues[4] = d4
			ps16.OverlayValues[5] = d5
			ps16.OverlayValues[6] = d6
			ps16.OverlayValues[8] = d8
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[11] = d11
			ps16.OverlayValues[12] = d12
			snap17 := d10
			alloc18 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc18)
			d10 = snap17
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps15)
			}
			return result
			ctx.FreeDesc(&d11)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d10)
			var d19 JITValueDesc
			if d10.Loc == LocImm {
				idx := int(d10.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d19 = args[idx]
				d19.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d10.Reg] {
					ctx.ProtectReg(d10.Reg)
					seen[d10.Reg] = true
					protected = append(protected, d10.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl21 := ctx.W.ReserveLabel()
				lbl22 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d10.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl22)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d10.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl21)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl22)
				d20 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d20)
				ctx.BindReg(r5, &d20)
				ctx.BindReg(r4, &d20)
				ctx.BindReg(r5, &d20)
				ctx.W.EmitMakeNil(d20)
				ctx.W.MarkLabel(lbl21)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d19 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d19)
				ctx.BindReg(r5, &d19)
			}
			d22 = d19
			d22.ID = 0
			d21 = ctx.EmitTagEqualsBorrowed(&d22, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d19)
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
			ps24.OverlayValues[6] = d6
			ps24.OverlayValues[8] = d8
			ps24.OverlayValues[9] = d9
			ps24.OverlayValues[10] = d10
			ps24.OverlayValues[11] = d11
			ps24.OverlayValues[12] = d12
			ps24.OverlayValues[19] = d19
			ps24.OverlayValues[20] = d20
			ps24.OverlayValues[21] = d21
			ps24.OverlayValues[22] = d22
			ps24.OverlayValues[23] = d23
					return bbs[4].RenderPS(ps24)
				}
			d25 = d10
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			ctx.EmitStoreToStack(d25, 0)
			ps26 := PhiState{General: ps.General}
			ps26.OverlayValues = make([]JITValueDesc, 26)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[6] = d6
			ps26.OverlayValues[8] = d8
			ps26.OverlayValues[9] = d9
			ps26.OverlayValues[10] = d10
			ps26.OverlayValues[11] = d11
			ps26.OverlayValues[12] = d12
			ps26.OverlayValues[19] = d19
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[22] = d22
			ps26.OverlayValues[23] = d23
			ps26.OverlayValues[25] = d25
			ps26.PhiValues = make([]JITValueDesc, 1)
			d27 = d10
			ps26.PhiValues[0] = d27
				return bbs[1].RenderPS(ps26)
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d23.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl23)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl23)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl24)
			d28 = d10
			if d28.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			ctx.EmitStoreToStack(d28, 0)
			ctx.W.EmitJmp(lbl2)
			ps29 := PhiState{General: true}
			ps29.OverlayValues = make([]JITValueDesc, 29)
			ps29.OverlayValues[0] = d0
			ps29.OverlayValues[1] = d1
			ps29.OverlayValues[2] = d2
			ps29.OverlayValues[3] = d3
			ps29.OverlayValues[4] = d4
			ps29.OverlayValues[5] = d5
			ps29.OverlayValues[6] = d6
			ps29.OverlayValues[8] = d8
			ps29.OverlayValues[9] = d9
			ps29.OverlayValues[10] = d10
			ps29.OverlayValues[11] = d11
			ps29.OverlayValues[12] = d12
			ps29.OverlayValues[19] = d19
			ps29.OverlayValues[20] = d20
			ps29.OverlayValues[21] = d21
			ps29.OverlayValues[22] = d22
			ps29.OverlayValues[23] = d23
			ps29.OverlayValues[25] = d25
			ps29.OverlayValues[27] = d27
			ps29.OverlayValues[28] = d28
			ps30 := PhiState{General: true}
			ps30.OverlayValues = make([]JITValueDesc, 29)
			ps30.OverlayValues[0] = d0
			ps30.OverlayValues[1] = d1
			ps30.OverlayValues[2] = d2
			ps30.OverlayValues[3] = d3
			ps30.OverlayValues[4] = d4
			ps30.OverlayValues[5] = d5
			ps30.OverlayValues[6] = d6
			ps30.OverlayValues[8] = d8
			ps30.OverlayValues[9] = d9
			ps30.OverlayValues[10] = d10
			ps30.OverlayValues[11] = d11
			ps30.OverlayValues[12] = d12
			ps30.OverlayValues[19] = d19
			ps30.OverlayValues[20] = d20
			ps30.OverlayValues[21] = d21
			ps30.OverlayValues[22] = d22
			ps30.OverlayValues[23] = d23
			ps30.OverlayValues[25] = d25
			ps30.OverlayValues[27] = d27
			ps30.OverlayValues[28] = d28
			ps30.PhiValues = make([]JITValueDesc, 1)
			d31 = d10
			ps30.PhiValues[0] = d31
			alloc32 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps30)
			}
			ctx.RestoreAllocState(alloc32)
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps29)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(1)}, 8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 16)
			ps33 := PhiState{General: ps.General}
			ps33.OverlayValues = make([]JITValueDesc, 32)
			ps33.OverlayValues[0] = d0
			ps33.OverlayValues[1] = d1
			ps33.OverlayValues[2] = d2
			ps33.OverlayValues[3] = d3
			ps33.OverlayValues[4] = d4
			ps33.OverlayValues[5] = d5
			ps33.OverlayValues[6] = d6
			ps33.OverlayValues[8] = d8
			ps33.OverlayValues[9] = d9
			ps33.OverlayValues[10] = d10
			ps33.OverlayValues[11] = d11
			ps33.OverlayValues[12] = d12
			ps33.OverlayValues[19] = d19
			ps33.OverlayValues[20] = d20
			ps33.OverlayValues[21] = d21
			ps33.OverlayValues[22] = d22
			ps33.OverlayValues[23] = d23
			ps33.OverlayValues[25] = d25
			ps33.OverlayValues[27] = d27
			ps33.OverlayValues[28] = d28
			ps33.OverlayValues[31] = d31
			ps33.PhiValues = make([]JITValueDesc, 2)
			d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
			ps33.PhiValues[0] = d34
			d35 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps33.PhiValues[1] = d35
			if ps33.General && bbs[7].Rendered {
				ctx.W.EmitJmp(lbl8)
				return result
			}
			return bbs[7].RenderPS(ps33)
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
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
				d35 = ps.OverlayValues[35]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
				d35 = ps.OverlayValues[35]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d36 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d36 = args[idx]
				d36.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d2.Reg] {
					ctx.ProtectReg(d2.Reg)
					seen[d2.Reg] = true
					protected = append(protected, d2.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r6 := ctx.AllocReg()
				r7 := ctx.AllocRegExcept(r6)
				lbl25 := ctx.W.ReserveLabel()
				lbl26 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl26)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r6, ai.Reg)
						ctx.W.EmitMovRegReg(r7, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r6, tmp.Reg)
						ctx.W.EmitMovRegReg(r7, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
						ctx.BindReg(r6, &pair)
						ctx.BindReg(r7, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r6, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r7, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl25)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl26)
				d37 := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d37)
				ctx.BindReg(r7, &d37)
				ctx.BindReg(r6, &d37)
				ctx.BindReg(r7, &d37)
				ctx.W.EmitMakeNil(d37)
				ctx.W.MarkLabel(lbl25)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d36 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d36)
				ctx.BindReg(r7, &d36)
			}
			d39 = d36
			d39.ID = 0
			d38 = ctx.EmitTagEqualsBorrowed(&d39, tagInt, JITValueDesc{Loc: LocAny})
			d40 = d38
			ctx.EnsureDesc(&d40)
			if d40.Loc != LocImm && d40.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d40.Loc == LocImm {
				if d40.Imm.Bool() {
			ps41 := PhiState{General: ps.General}
			ps41.OverlayValues = make([]JITValueDesc, 41)
			ps41.OverlayValues[0] = d0
			ps41.OverlayValues[1] = d1
			ps41.OverlayValues[2] = d2
			ps41.OverlayValues[3] = d3
			ps41.OverlayValues[4] = d4
			ps41.OverlayValues[5] = d5
			ps41.OverlayValues[6] = d6
			ps41.OverlayValues[8] = d8
			ps41.OverlayValues[9] = d9
			ps41.OverlayValues[10] = d10
			ps41.OverlayValues[11] = d11
			ps41.OverlayValues[12] = d12
			ps41.OverlayValues[19] = d19
			ps41.OverlayValues[20] = d20
			ps41.OverlayValues[21] = d21
			ps41.OverlayValues[22] = d22
			ps41.OverlayValues[23] = d23
			ps41.OverlayValues[25] = d25
			ps41.OverlayValues[27] = d27
			ps41.OverlayValues[28] = d28
			ps41.OverlayValues[31] = d31
			ps41.OverlayValues[34] = d34
			ps41.OverlayValues[35] = d35
			ps41.OverlayValues[36] = d36
			ps41.OverlayValues[37] = d37
			ps41.OverlayValues[38] = d38
			ps41.OverlayValues[39] = d39
			ps41.OverlayValues[40] = d40
					return bbs[9].RenderPS(ps41)
				}
			ps42 := PhiState{General: ps.General}
			ps42.OverlayValues = make([]JITValueDesc, 41)
			ps42.OverlayValues[0] = d0
			ps42.OverlayValues[1] = d1
			ps42.OverlayValues[2] = d2
			ps42.OverlayValues[3] = d3
			ps42.OverlayValues[4] = d4
			ps42.OverlayValues[5] = d5
			ps42.OverlayValues[6] = d6
			ps42.OverlayValues[8] = d8
			ps42.OverlayValues[9] = d9
			ps42.OverlayValues[10] = d10
			ps42.OverlayValues[11] = d11
			ps42.OverlayValues[12] = d12
			ps42.OverlayValues[19] = d19
			ps42.OverlayValues[20] = d20
			ps42.OverlayValues[21] = d21
			ps42.OverlayValues[22] = d22
			ps42.OverlayValues[23] = d23
			ps42.OverlayValues[25] = d25
			ps42.OverlayValues[27] = d27
			ps42.OverlayValues[28] = d28
			ps42.OverlayValues[31] = d31
			ps42.OverlayValues[34] = d34
			ps42.OverlayValues[35] = d35
			ps42.OverlayValues[36] = d36
			ps42.OverlayValues[37] = d37
			ps42.OverlayValues[38] = d38
			ps42.OverlayValues[39] = d39
			ps42.OverlayValues[40] = d40
				return bbs[10].RenderPS(ps42)
			}
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d40.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl27)
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl27)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl28)
			ctx.W.EmitJmp(lbl11)
			ps43 := PhiState{General: true}
			ps43.OverlayValues = make([]JITValueDesc, 41)
			ps43.OverlayValues[0] = d0
			ps43.OverlayValues[1] = d1
			ps43.OverlayValues[2] = d2
			ps43.OverlayValues[3] = d3
			ps43.OverlayValues[4] = d4
			ps43.OverlayValues[5] = d5
			ps43.OverlayValues[6] = d6
			ps43.OverlayValues[8] = d8
			ps43.OverlayValues[9] = d9
			ps43.OverlayValues[10] = d10
			ps43.OverlayValues[11] = d11
			ps43.OverlayValues[12] = d12
			ps43.OverlayValues[19] = d19
			ps43.OverlayValues[20] = d20
			ps43.OverlayValues[21] = d21
			ps43.OverlayValues[22] = d22
			ps43.OverlayValues[23] = d23
			ps43.OverlayValues[25] = d25
			ps43.OverlayValues[27] = d27
			ps43.OverlayValues[28] = d28
			ps43.OverlayValues[31] = d31
			ps43.OverlayValues[34] = d34
			ps43.OverlayValues[35] = d35
			ps43.OverlayValues[36] = d36
			ps43.OverlayValues[37] = d37
			ps43.OverlayValues[38] = d38
			ps43.OverlayValues[39] = d39
			ps43.OverlayValues[40] = d40
			ps44 := PhiState{General: true}
			ps44.OverlayValues = make([]JITValueDesc, 41)
			ps44.OverlayValues[0] = d0
			ps44.OverlayValues[1] = d1
			ps44.OverlayValues[2] = d2
			ps44.OverlayValues[3] = d3
			ps44.OverlayValues[4] = d4
			ps44.OverlayValues[5] = d5
			ps44.OverlayValues[6] = d6
			ps44.OverlayValues[8] = d8
			ps44.OverlayValues[9] = d9
			ps44.OverlayValues[10] = d10
			ps44.OverlayValues[11] = d11
			ps44.OverlayValues[12] = d12
			ps44.OverlayValues[19] = d19
			ps44.OverlayValues[20] = d20
			ps44.OverlayValues[21] = d21
			ps44.OverlayValues[22] = d22
			ps44.OverlayValues[23] = d23
			ps44.OverlayValues[25] = d25
			ps44.OverlayValues[27] = d27
			ps44.OverlayValues[28] = d28
			ps44.OverlayValues[31] = d31
			ps44.OverlayValues[34] = d34
			ps44.OverlayValues[35] = d35
			ps44.OverlayValues[36] = d36
			ps44.OverlayValues[37] = d37
			ps44.OverlayValues[38] = d38
			ps44.OverlayValues[39] = d39
			ps44.OverlayValues[40] = d40
			snap45 := d1
			snap46 := d36
			alloc47 := ctx.SnapshotAllocState()
			if !bbs[10].Rendered {
				bbs[10].RenderPS(ps44)
			}
			ctx.RestoreAllocState(alloc47)
			d1 = snap45
			d36 = snap46
			if !bbs[9].Rendered {
				return bbs[9].RenderPS(ps43)
			}
			return result
			ctx.FreeDesc(&d38)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			ctx.ReclaimUntrackedRegs()
			d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d48)
			var d49 JITValueDesc
			if d2.Loc == LocImm && d48.Loc == LocImm {
				d49 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == d48.Imm.Int())}
			} else if d48.Loc == LocImm {
				r8 := ctx.AllocRegExcept(d2.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d48.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r8, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d49)
			} else if d2.Loc == LocImm {
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d48.Reg)
				ctx.W.EmitSetcc(r9, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d49)
			} else {
				r10 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d48.Reg)
				ctx.W.EmitSetcc(r10, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d49)
			}
			ctx.FreeDesc(&d48)
			d50 = d49
			ctx.EnsureDesc(&d50)
			if d50.Loc != LocImm && d50.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d50.Loc == LocImm {
				if d50.Imm.Bool() {
			ps51 := PhiState{General: ps.General}
			ps51.OverlayValues = make([]JITValueDesc, 51)
			ps51.OverlayValues[0] = d0
			ps51.OverlayValues[1] = d1
			ps51.OverlayValues[2] = d2
			ps51.OverlayValues[3] = d3
			ps51.OverlayValues[4] = d4
			ps51.OverlayValues[5] = d5
			ps51.OverlayValues[6] = d6
			ps51.OverlayValues[8] = d8
			ps51.OverlayValues[9] = d9
			ps51.OverlayValues[10] = d10
			ps51.OverlayValues[11] = d11
			ps51.OverlayValues[12] = d12
			ps51.OverlayValues[19] = d19
			ps51.OverlayValues[20] = d20
			ps51.OverlayValues[21] = d21
			ps51.OverlayValues[22] = d22
			ps51.OverlayValues[23] = d23
			ps51.OverlayValues[25] = d25
			ps51.OverlayValues[27] = d27
			ps51.OverlayValues[28] = d28
			ps51.OverlayValues[31] = d31
			ps51.OverlayValues[34] = d34
			ps51.OverlayValues[35] = d35
			ps51.OverlayValues[36] = d36
			ps51.OverlayValues[37] = d37
			ps51.OverlayValues[38] = d38
			ps51.OverlayValues[39] = d39
			ps51.OverlayValues[40] = d40
			ps51.OverlayValues[48] = d48
			ps51.OverlayValues[49] = d49
			ps51.OverlayValues[50] = d50
					return bbs[13].RenderPS(ps51)
				}
			ps52 := PhiState{General: ps.General}
			ps52.OverlayValues = make([]JITValueDesc, 51)
			ps52.OverlayValues[0] = d0
			ps52.OverlayValues[1] = d1
			ps52.OverlayValues[2] = d2
			ps52.OverlayValues[3] = d3
			ps52.OverlayValues[4] = d4
			ps52.OverlayValues[5] = d5
			ps52.OverlayValues[6] = d6
			ps52.OverlayValues[8] = d8
			ps52.OverlayValues[9] = d9
			ps52.OverlayValues[10] = d10
			ps52.OverlayValues[11] = d11
			ps52.OverlayValues[12] = d12
			ps52.OverlayValues[19] = d19
			ps52.OverlayValues[20] = d20
			ps52.OverlayValues[21] = d21
			ps52.OverlayValues[22] = d22
			ps52.OverlayValues[23] = d23
			ps52.OverlayValues[25] = d25
			ps52.OverlayValues[27] = d27
			ps52.OverlayValues[28] = d28
			ps52.OverlayValues[31] = d31
			ps52.OverlayValues[34] = d34
			ps52.OverlayValues[35] = d35
			ps52.OverlayValues[36] = d36
			ps52.OverlayValues[37] = d37
			ps52.OverlayValues[38] = d38
			ps52.OverlayValues[39] = d39
			ps52.OverlayValues[40] = d40
			ps52.OverlayValues[48] = d48
			ps52.OverlayValues[49] = d49
			ps52.OverlayValues[50] = d50
				return bbs[14].RenderPS(ps52)
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d50.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl29)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl29)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl30)
			ctx.W.EmitJmp(lbl15)
			ps53 := PhiState{General: true}
			ps53.OverlayValues = make([]JITValueDesc, 51)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
			ps53.OverlayValues[2] = d2
			ps53.OverlayValues[3] = d3
			ps53.OverlayValues[4] = d4
			ps53.OverlayValues[5] = d5
			ps53.OverlayValues[6] = d6
			ps53.OverlayValues[8] = d8
			ps53.OverlayValues[9] = d9
			ps53.OverlayValues[10] = d10
			ps53.OverlayValues[11] = d11
			ps53.OverlayValues[12] = d12
			ps53.OverlayValues[19] = d19
			ps53.OverlayValues[20] = d20
			ps53.OverlayValues[21] = d21
			ps53.OverlayValues[22] = d22
			ps53.OverlayValues[23] = d23
			ps53.OverlayValues[25] = d25
			ps53.OverlayValues[27] = d27
			ps53.OverlayValues[28] = d28
			ps53.OverlayValues[31] = d31
			ps53.OverlayValues[34] = d34
			ps53.OverlayValues[35] = d35
			ps53.OverlayValues[36] = d36
			ps53.OverlayValues[37] = d37
			ps53.OverlayValues[38] = d38
			ps53.OverlayValues[39] = d39
			ps53.OverlayValues[40] = d40
			ps53.OverlayValues[48] = d48
			ps53.OverlayValues[49] = d49
			ps53.OverlayValues[50] = d50
			ps54 := PhiState{General: true}
			ps54.OverlayValues = make([]JITValueDesc, 51)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
			ps54.OverlayValues[2] = d2
			ps54.OverlayValues[3] = d3
			ps54.OverlayValues[4] = d4
			ps54.OverlayValues[5] = d5
			ps54.OverlayValues[6] = d6
			ps54.OverlayValues[8] = d8
			ps54.OverlayValues[9] = d9
			ps54.OverlayValues[10] = d10
			ps54.OverlayValues[11] = d11
			ps54.OverlayValues[12] = d12
			ps54.OverlayValues[19] = d19
			ps54.OverlayValues[20] = d20
			ps54.OverlayValues[21] = d21
			ps54.OverlayValues[22] = d22
			ps54.OverlayValues[23] = d23
			ps54.OverlayValues[25] = d25
			ps54.OverlayValues[27] = d27
			ps54.OverlayValues[28] = d28
			ps54.OverlayValues[31] = d31
			ps54.OverlayValues[34] = d34
			ps54.OverlayValues[35] = d35
			ps54.OverlayValues[36] = d36
			ps54.OverlayValues[37] = d37
			ps54.OverlayValues[38] = d38
			ps54.OverlayValues[39] = d39
			ps54.OverlayValues[40] = d40
			ps54.OverlayValues[48] = d48
			ps54.OverlayValues[49] = d49
			ps54.OverlayValues[50] = d50
			snap55 := d1
			alloc56 := ctx.SnapshotAllocState()
			if !bbs[14].Rendered {
				bbs[14].RenderPS(ps54)
			}
			ctx.RestoreAllocState(alloc56)
			d1 = snap55
			if !bbs[13].Rendered {
				return bbs[13].RenderPS(ps53)
			}
			return result
			ctx.FreeDesc(&d49)
			return result
			}
			bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[7].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d57 := ps.PhiValues[0]
						ctx.EnsureDesc(&d57)
						ctx.EmitStoreToStack(d57, 8)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d58 := ps.PhiValues[1]
						ctx.EnsureDesc(&d58)
						ctx.EmitStoreToStack(d58, 16)
					}
					ps.General = true
					return bbs[7].RenderPS(ps)
				}
			}
			bbs[7].VisitCount++
			if ps.General {
				if bbs[7].Rendered {
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d2 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d59 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d59)
			var d60 JITValueDesc
			if d2.Loc == LocImm && d59.Loc == LocImm {
				d60 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d59.Imm.Int())}
			} else if d59.Loc == LocImm {
				r11 := ctx.AllocRegExcept(d2.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d59.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d59.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r11, CcL)
				d60 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d60)
			} else if d2.Loc == LocImm {
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d59.Reg)
				ctx.W.EmitSetcc(r12, CcL)
				d60 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
				ctx.BindReg(r12, &d60)
			} else {
				r13 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d59.Reg)
				ctx.W.EmitSetcc(r13, CcL)
				d60 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
				ctx.BindReg(r13, &d60)
			}
			ctx.FreeDesc(&d59)
			d61 = d60
			ctx.EnsureDesc(&d61)
			if d61.Loc != LocImm && d61.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d61.Loc == LocImm {
				if d61.Imm.Bool() {
			ps62 := PhiState{General: ps.General}
			ps62.OverlayValues = make([]JITValueDesc, 62)
			ps62.OverlayValues[0] = d0
			ps62.OverlayValues[1] = d1
			ps62.OverlayValues[2] = d2
			ps62.OverlayValues[3] = d3
			ps62.OverlayValues[4] = d4
			ps62.OverlayValues[5] = d5
			ps62.OverlayValues[6] = d6
			ps62.OverlayValues[8] = d8
			ps62.OverlayValues[9] = d9
			ps62.OverlayValues[10] = d10
			ps62.OverlayValues[11] = d11
			ps62.OverlayValues[12] = d12
			ps62.OverlayValues[19] = d19
			ps62.OverlayValues[20] = d20
			ps62.OverlayValues[21] = d21
			ps62.OverlayValues[22] = d22
			ps62.OverlayValues[23] = d23
			ps62.OverlayValues[25] = d25
			ps62.OverlayValues[27] = d27
			ps62.OverlayValues[28] = d28
			ps62.OverlayValues[31] = d31
			ps62.OverlayValues[34] = d34
			ps62.OverlayValues[35] = d35
			ps62.OverlayValues[36] = d36
			ps62.OverlayValues[37] = d37
			ps62.OverlayValues[38] = d38
			ps62.OverlayValues[39] = d39
			ps62.OverlayValues[40] = d40
			ps62.OverlayValues[48] = d48
			ps62.OverlayValues[49] = d49
			ps62.OverlayValues[50] = d50
			ps62.OverlayValues[57] = d57
			ps62.OverlayValues[58] = d58
			ps62.OverlayValues[59] = d59
			ps62.OverlayValues[60] = d60
			ps62.OverlayValues[61] = d61
					return bbs[5].RenderPS(ps62)
				}
			ps63 := PhiState{General: ps.General}
			ps63.OverlayValues = make([]JITValueDesc, 62)
			ps63.OverlayValues[0] = d0
			ps63.OverlayValues[1] = d1
			ps63.OverlayValues[2] = d2
			ps63.OverlayValues[3] = d3
			ps63.OverlayValues[4] = d4
			ps63.OverlayValues[5] = d5
			ps63.OverlayValues[6] = d6
			ps63.OverlayValues[8] = d8
			ps63.OverlayValues[9] = d9
			ps63.OverlayValues[10] = d10
			ps63.OverlayValues[11] = d11
			ps63.OverlayValues[12] = d12
			ps63.OverlayValues[19] = d19
			ps63.OverlayValues[20] = d20
			ps63.OverlayValues[21] = d21
			ps63.OverlayValues[22] = d22
			ps63.OverlayValues[23] = d23
			ps63.OverlayValues[25] = d25
			ps63.OverlayValues[27] = d27
			ps63.OverlayValues[28] = d28
			ps63.OverlayValues[31] = d31
			ps63.OverlayValues[34] = d34
			ps63.OverlayValues[35] = d35
			ps63.OverlayValues[36] = d36
			ps63.OverlayValues[37] = d37
			ps63.OverlayValues[38] = d38
			ps63.OverlayValues[39] = d39
			ps63.OverlayValues[40] = d40
			ps63.OverlayValues[48] = d48
			ps63.OverlayValues[49] = d49
			ps63.OverlayValues[50] = d50
			ps63.OverlayValues[57] = d57
			ps63.OverlayValues[58] = d58
			ps63.OverlayValues[59] = d59
			ps63.OverlayValues[60] = d60
			ps63.OverlayValues[61] = d61
				return bbs[6].RenderPS(ps63)
			}
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d61.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl31)
			ctx.W.EmitJmp(lbl32)
			ctx.W.MarkLabel(lbl31)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl32)
			ctx.W.EmitJmp(lbl7)
			ps64 := PhiState{General: true}
			ps64.OverlayValues = make([]JITValueDesc, 62)
			ps64.OverlayValues[0] = d0
			ps64.OverlayValues[1] = d1
			ps64.OverlayValues[2] = d2
			ps64.OverlayValues[3] = d3
			ps64.OverlayValues[4] = d4
			ps64.OverlayValues[5] = d5
			ps64.OverlayValues[6] = d6
			ps64.OverlayValues[8] = d8
			ps64.OverlayValues[9] = d9
			ps64.OverlayValues[10] = d10
			ps64.OverlayValues[11] = d11
			ps64.OverlayValues[12] = d12
			ps64.OverlayValues[19] = d19
			ps64.OverlayValues[20] = d20
			ps64.OverlayValues[21] = d21
			ps64.OverlayValues[22] = d22
			ps64.OverlayValues[23] = d23
			ps64.OverlayValues[25] = d25
			ps64.OverlayValues[27] = d27
			ps64.OverlayValues[28] = d28
			ps64.OverlayValues[31] = d31
			ps64.OverlayValues[34] = d34
			ps64.OverlayValues[35] = d35
			ps64.OverlayValues[36] = d36
			ps64.OverlayValues[37] = d37
			ps64.OverlayValues[38] = d38
			ps64.OverlayValues[39] = d39
			ps64.OverlayValues[40] = d40
			ps64.OverlayValues[48] = d48
			ps64.OverlayValues[49] = d49
			ps64.OverlayValues[50] = d50
			ps64.OverlayValues[57] = d57
			ps64.OverlayValues[58] = d58
			ps64.OverlayValues[59] = d59
			ps64.OverlayValues[60] = d60
			ps64.OverlayValues[61] = d61
			ps65 := PhiState{General: true}
			ps65.OverlayValues = make([]JITValueDesc, 62)
			ps65.OverlayValues[0] = d0
			ps65.OverlayValues[1] = d1
			ps65.OverlayValues[2] = d2
			ps65.OverlayValues[3] = d3
			ps65.OverlayValues[4] = d4
			ps65.OverlayValues[5] = d5
			ps65.OverlayValues[6] = d6
			ps65.OverlayValues[8] = d8
			ps65.OverlayValues[9] = d9
			ps65.OverlayValues[10] = d10
			ps65.OverlayValues[11] = d11
			ps65.OverlayValues[12] = d12
			ps65.OverlayValues[19] = d19
			ps65.OverlayValues[20] = d20
			ps65.OverlayValues[21] = d21
			ps65.OverlayValues[22] = d22
			ps65.OverlayValues[23] = d23
			ps65.OverlayValues[25] = d25
			ps65.OverlayValues[27] = d27
			ps65.OverlayValues[28] = d28
			ps65.OverlayValues[31] = d31
			ps65.OverlayValues[34] = d34
			ps65.OverlayValues[35] = d35
			ps65.OverlayValues[36] = d36
			ps65.OverlayValues[37] = d37
			ps65.OverlayValues[38] = d38
			ps65.OverlayValues[39] = d39
			ps65.OverlayValues[40] = d40
			ps65.OverlayValues[48] = d48
			ps65.OverlayValues[49] = d49
			ps65.OverlayValues[50] = d50
			ps65.OverlayValues[57] = d57
			ps65.OverlayValues[58] = d58
			ps65.OverlayValues[59] = d59
			ps65.OverlayValues[60] = d60
			ps65.OverlayValues[61] = d61
			snap66 := d2
			snap67 := d36
			snap68 := d38
			alloc69 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps65)
			}
			ctx.RestoreAllocState(alloc69)
			d2 = snap66
			d36 = snap67
			d38 = snap68
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps64)
			}
			return result
			ctx.FreeDesc(&d60)
			return result
			}
			bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[8].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d70 := ps.PhiValues[0]
						ctx.EnsureDesc(&d70)
						ctx.EmitStoreToStack(d70, 24)
					}
					ps.General = true
					return bbs[8].RenderPS(ps)
				}
			}
			bbs[8].VisitCount++
			if ps.General {
				if bbs[8].Rendered {
					ctx.W.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.W.MarkLabel(lbl9)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
				d70 = ps.OverlayValues[70]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d3 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d71 JITValueDesc
			if d2.Loc == LocImm {
				d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d71 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			}
			if d71.Loc == LocReg && d2.Loc == LocReg && d71.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			d72 = d3
			if d72.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d72)
			ctx.EmitStoreToStack(d72, 8)
			d73 = d71
			if d73.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, 16)
			ps74 := PhiState{General: ps.General}
			ps74.OverlayValues = make([]JITValueDesc, 74)
			ps74.OverlayValues[0] = d0
			ps74.OverlayValues[1] = d1
			ps74.OverlayValues[2] = d2
			ps74.OverlayValues[3] = d3
			ps74.OverlayValues[4] = d4
			ps74.OverlayValues[5] = d5
			ps74.OverlayValues[6] = d6
			ps74.OverlayValues[8] = d8
			ps74.OverlayValues[9] = d9
			ps74.OverlayValues[10] = d10
			ps74.OverlayValues[11] = d11
			ps74.OverlayValues[12] = d12
			ps74.OverlayValues[19] = d19
			ps74.OverlayValues[20] = d20
			ps74.OverlayValues[21] = d21
			ps74.OverlayValues[22] = d22
			ps74.OverlayValues[23] = d23
			ps74.OverlayValues[25] = d25
			ps74.OverlayValues[27] = d27
			ps74.OverlayValues[28] = d28
			ps74.OverlayValues[31] = d31
			ps74.OverlayValues[34] = d34
			ps74.OverlayValues[35] = d35
			ps74.OverlayValues[36] = d36
			ps74.OverlayValues[37] = d37
			ps74.OverlayValues[38] = d38
			ps74.OverlayValues[39] = d39
			ps74.OverlayValues[40] = d40
			ps74.OverlayValues[48] = d48
			ps74.OverlayValues[49] = d49
			ps74.OverlayValues[50] = d50
			ps74.OverlayValues[57] = d57
			ps74.OverlayValues[58] = d58
			ps74.OverlayValues[59] = d59
			ps74.OverlayValues[60] = d60
			ps74.OverlayValues[61] = d61
			ps74.OverlayValues[70] = d70
			ps74.OverlayValues[71] = d71
			ps74.OverlayValues[72] = d72
			ps74.OverlayValues[73] = d73
			ps74.PhiValues = make([]JITValueDesc, 2)
			d75 = d3
			ps74.PhiValues[0] = d75
			d76 = d71
			ps74.PhiValues[1] = d76
			if ps74.General && bbs[7].Rendered {
				ctx.W.EmitJmp(lbl8)
				return result
			}
			return bbs[7].RenderPS(ps74)
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
					ctx.W.EmitJmp(lbl10)
					return result
				}
				bbs[9].Rendered = true
				bbs[9].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_9 = bbs[9].Address
				ctx.W.MarkLabel(lbl10)
				ctx.W.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			ctx.ReclaimUntrackedRegs()
			var d77 JITValueDesc
			if d36.Loc == LocImm {
				d77 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d36.Imm.Int())}
			} else if d36.Type == tagInt && d36.Loc == LocRegPair {
				ctx.FreeReg(d36.Reg)
				d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg2}
				ctx.BindReg(d36.Reg2, &d77)
				ctx.BindReg(d36.Reg2, &d77)
			} else if d36.Type == tagInt && d36.Loc == LocReg {
				d77 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d36.Reg}
				ctx.BindReg(d36.Reg, &d77)
				ctx.BindReg(d36.Reg, &d77)
			} else {
				d77 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d36}, 1)
				d77.Type = tagInt
				ctx.BindReg(d77.Reg, &d77)
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d77)
			var d78 JITValueDesc
			if d1.Loc == LocImm && d77.Loc == LocImm {
				d78 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() * d77.Imm.Int())}
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d77.Reg)
				d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else if d77.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d77.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d77.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, RegR11)
				}
				d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else {
				r14 := ctx.AllocRegExcept(d1.Reg, d77.Reg)
				ctx.W.EmitMovRegReg(r14, d1.Reg)
				ctx.W.EmitImulInt64(r14, d77.Reg)
				d78 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
				ctx.BindReg(r14, &d78)
			}
			if d78.Loc == LocReg && d1.Loc == LocReg && d78.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d77)
			d79 = d78
			if d79.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d79)
			ctx.EmitStoreToStack(d79, 24)
			ps80 := PhiState{General: ps.General}
			ps80.OverlayValues = make([]JITValueDesc, 80)
			ps80.OverlayValues[0] = d0
			ps80.OverlayValues[1] = d1
			ps80.OverlayValues[2] = d2
			ps80.OverlayValues[3] = d3
			ps80.OverlayValues[4] = d4
			ps80.OverlayValues[5] = d5
			ps80.OverlayValues[6] = d6
			ps80.OverlayValues[8] = d8
			ps80.OverlayValues[9] = d9
			ps80.OverlayValues[10] = d10
			ps80.OverlayValues[11] = d11
			ps80.OverlayValues[12] = d12
			ps80.OverlayValues[19] = d19
			ps80.OverlayValues[20] = d20
			ps80.OverlayValues[21] = d21
			ps80.OverlayValues[22] = d22
			ps80.OverlayValues[23] = d23
			ps80.OverlayValues[25] = d25
			ps80.OverlayValues[27] = d27
			ps80.OverlayValues[28] = d28
			ps80.OverlayValues[31] = d31
			ps80.OverlayValues[34] = d34
			ps80.OverlayValues[35] = d35
			ps80.OverlayValues[36] = d36
			ps80.OverlayValues[37] = d37
			ps80.OverlayValues[38] = d38
			ps80.OverlayValues[39] = d39
			ps80.OverlayValues[40] = d40
			ps80.OverlayValues[48] = d48
			ps80.OverlayValues[49] = d49
			ps80.OverlayValues[50] = d50
			ps80.OverlayValues[57] = d57
			ps80.OverlayValues[58] = d58
			ps80.OverlayValues[59] = d59
			ps80.OverlayValues[60] = d60
			ps80.OverlayValues[61] = d61
			ps80.OverlayValues[70] = d70
			ps80.OverlayValues[71] = d71
			ps80.OverlayValues[72] = d72
			ps80.OverlayValues[73] = d73
			ps80.OverlayValues[75] = d75
			ps80.OverlayValues[76] = d76
			ps80.OverlayValues[77] = d77
			ps80.OverlayValues[78] = d78
			ps80.OverlayValues[79] = d79
			ps80.PhiValues = make([]JITValueDesc, 1)
			d81 = d78
			ps80.PhiValues[0] = d81
			if ps80.General && bbs[8].Rendered {
				ctx.W.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps80)
			return result
			}
			bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[10].VisitCount >= 2 {
					ps.General = true
					return bbs[10].RenderPS(ps)
				}
			}
			bbs[10].VisitCount++
			if ps.General {
				if bbs[10].Rendered {
					ctx.W.EmitJmp(lbl11)
					return result
				}
				bbs[10].Rendered = true
				bbs[10].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_10 = bbs[10].Address
				ctx.W.MarkLabel(lbl11)
				ctx.W.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			ctx.ReclaimUntrackedRegs()
			d83 = d36
			d83.ID = 0
			d82 = ctx.EmitTagEqualsBorrowed(&d83, tagFloat, JITValueDesc{Loc: LocAny})
			d84 = d82
			ctx.EnsureDesc(&d84)
			if d84.Loc != LocImm && d84.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d84.Loc == LocImm {
				if d84.Imm.Bool() {
			ps85 := PhiState{General: ps.General}
			ps85.OverlayValues = make([]JITValueDesc, 85)
			ps85.OverlayValues[0] = d0
			ps85.OverlayValues[1] = d1
			ps85.OverlayValues[2] = d2
			ps85.OverlayValues[3] = d3
			ps85.OverlayValues[4] = d4
			ps85.OverlayValues[5] = d5
			ps85.OverlayValues[6] = d6
			ps85.OverlayValues[8] = d8
			ps85.OverlayValues[9] = d9
			ps85.OverlayValues[10] = d10
			ps85.OverlayValues[11] = d11
			ps85.OverlayValues[12] = d12
			ps85.OverlayValues[19] = d19
			ps85.OverlayValues[20] = d20
			ps85.OverlayValues[21] = d21
			ps85.OverlayValues[22] = d22
			ps85.OverlayValues[23] = d23
			ps85.OverlayValues[25] = d25
			ps85.OverlayValues[27] = d27
			ps85.OverlayValues[28] = d28
			ps85.OverlayValues[31] = d31
			ps85.OverlayValues[34] = d34
			ps85.OverlayValues[35] = d35
			ps85.OverlayValues[36] = d36
			ps85.OverlayValues[37] = d37
			ps85.OverlayValues[38] = d38
			ps85.OverlayValues[39] = d39
			ps85.OverlayValues[40] = d40
			ps85.OverlayValues[48] = d48
			ps85.OverlayValues[49] = d49
			ps85.OverlayValues[50] = d50
			ps85.OverlayValues[57] = d57
			ps85.OverlayValues[58] = d58
			ps85.OverlayValues[59] = d59
			ps85.OverlayValues[60] = d60
			ps85.OverlayValues[61] = d61
			ps85.OverlayValues[70] = d70
			ps85.OverlayValues[71] = d71
			ps85.OverlayValues[72] = d72
			ps85.OverlayValues[73] = d73
			ps85.OverlayValues[75] = d75
			ps85.OverlayValues[76] = d76
			ps85.OverlayValues[77] = d77
			ps85.OverlayValues[78] = d78
			ps85.OverlayValues[79] = d79
			ps85.OverlayValues[81] = d81
			ps85.OverlayValues[82] = d82
			ps85.OverlayValues[83] = d83
			ps85.OverlayValues[84] = d84
					return bbs[11].RenderPS(ps85)
				}
			ps86 := PhiState{General: ps.General}
			ps86.OverlayValues = make([]JITValueDesc, 85)
			ps86.OverlayValues[0] = d0
			ps86.OverlayValues[1] = d1
			ps86.OverlayValues[2] = d2
			ps86.OverlayValues[3] = d3
			ps86.OverlayValues[4] = d4
			ps86.OverlayValues[5] = d5
			ps86.OverlayValues[6] = d6
			ps86.OverlayValues[8] = d8
			ps86.OverlayValues[9] = d9
			ps86.OverlayValues[10] = d10
			ps86.OverlayValues[11] = d11
			ps86.OverlayValues[12] = d12
			ps86.OverlayValues[19] = d19
			ps86.OverlayValues[20] = d20
			ps86.OverlayValues[21] = d21
			ps86.OverlayValues[22] = d22
			ps86.OverlayValues[23] = d23
			ps86.OverlayValues[25] = d25
			ps86.OverlayValues[27] = d27
			ps86.OverlayValues[28] = d28
			ps86.OverlayValues[31] = d31
			ps86.OverlayValues[34] = d34
			ps86.OverlayValues[35] = d35
			ps86.OverlayValues[36] = d36
			ps86.OverlayValues[37] = d37
			ps86.OverlayValues[38] = d38
			ps86.OverlayValues[39] = d39
			ps86.OverlayValues[40] = d40
			ps86.OverlayValues[48] = d48
			ps86.OverlayValues[49] = d49
			ps86.OverlayValues[50] = d50
			ps86.OverlayValues[57] = d57
			ps86.OverlayValues[58] = d58
			ps86.OverlayValues[59] = d59
			ps86.OverlayValues[60] = d60
			ps86.OverlayValues[61] = d61
			ps86.OverlayValues[70] = d70
			ps86.OverlayValues[71] = d71
			ps86.OverlayValues[72] = d72
			ps86.OverlayValues[73] = d73
			ps86.OverlayValues[75] = d75
			ps86.OverlayValues[76] = d76
			ps86.OverlayValues[77] = d77
			ps86.OverlayValues[78] = d78
			ps86.OverlayValues[79] = d79
			ps86.OverlayValues[81] = d81
			ps86.OverlayValues[82] = d82
			ps86.OverlayValues[83] = d83
			ps86.OverlayValues[84] = d84
				return bbs[6].RenderPS(ps86)
			}
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d84.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl33)
			ctx.W.EmitJmp(lbl34)
			ctx.W.MarkLabel(lbl33)
			ctx.W.EmitJmp(lbl12)
			ctx.W.MarkLabel(lbl34)
			ctx.W.EmitJmp(lbl7)
			ps87 := PhiState{General: true}
			ps87.OverlayValues = make([]JITValueDesc, 85)
			ps87.OverlayValues[0] = d0
			ps87.OverlayValues[1] = d1
			ps87.OverlayValues[2] = d2
			ps87.OverlayValues[3] = d3
			ps87.OverlayValues[4] = d4
			ps87.OverlayValues[5] = d5
			ps87.OverlayValues[6] = d6
			ps87.OverlayValues[8] = d8
			ps87.OverlayValues[9] = d9
			ps87.OverlayValues[10] = d10
			ps87.OverlayValues[11] = d11
			ps87.OverlayValues[12] = d12
			ps87.OverlayValues[19] = d19
			ps87.OverlayValues[20] = d20
			ps87.OverlayValues[21] = d21
			ps87.OverlayValues[22] = d22
			ps87.OverlayValues[23] = d23
			ps87.OverlayValues[25] = d25
			ps87.OverlayValues[27] = d27
			ps87.OverlayValues[28] = d28
			ps87.OverlayValues[31] = d31
			ps87.OverlayValues[34] = d34
			ps87.OverlayValues[35] = d35
			ps87.OverlayValues[36] = d36
			ps87.OverlayValues[37] = d37
			ps87.OverlayValues[38] = d38
			ps87.OverlayValues[39] = d39
			ps87.OverlayValues[40] = d40
			ps87.OverlayValues[48] = d48
			ps87.OverlayValues[49] = d49
			ps87.OverlayValues[50] = d50
			ps87.OverlayValues[57] = d57
			ps87.OverlayValues[58] = d58
			ps87.OverlayValues[59] = d59
			ps87.OverlayValues[60] = d60
			ps87.OverlayValues[61] = d61
			ps87.OverlayValues[70] = d70
			ps87.OverlayValues[71] = d71
			ps87.OverlayValues[72] = d72
			ps87.OverlayValues[73] = d73
			ps87.OverlayValues[75] = d75
			ps87.OverlayValues[76] = d76
			ps87.OverlayValues[77] = d77
			ps87.OverlayValues[78] = d78
			ps87.OverlayValues[79] = d79
			ps87.OverlayValues[81] = d81
			ps87.OverlayValues[82] = d82
			ps87.OverlayValues[83] = d83
			ps87.OverlayValues[84] = d84
			ps88 := PhiState{General: true}
			ps88.OverlayValues = make([]JITValueDesc, 85)
			ps88.OverlayValues[0] = d0
			ps88.OverlayValues[1] = d1
			ps88.OverlayValues[2] = d2
			ps88.OverlayValues[3] = d3
			ps88.OverlayValues[4] = d4
			ps88.OverlayValues[5] = d5
			ps88.OverlayValues[6] = d6
			ps88.OverlayValues[8] = d8
			ps88.OverlayValues[9] = d9
			ps88.OverlayValues[10] = d10
			ps88.OverlayValues[11] = d11
			ps88.OverlayValues[12] = d12
			ps88.OverlayValues[19] = d19
			ps88.OverlayValues[20] = d20
			ps88.OverlayValues[21] = d21
			ps88.OverlayValues[22] = d22
			ps88.OverlayValues[23] = d23
			ps88.OverlayValues[25] = d25
			ps88.OverlayValues[27] = d27
			ps88.OverlayValues[28] = d28
			ps88.OverlayValues[31] = d31
			ps88.OverlayValues[34] = d34
			ps88.OverlayValues[35] = d35
			ps88.OverlayValues[36] = d36
			ps88.OverlayValues[37] = d37
			ps88.OverlayValues[38] = d38
			ps88.OverlayValues[39] = d39
			ps88.OverlayValues[40] = d40
			ps88.OverlayValues[48] = d48
			ps88.OverlayValues[49] = d49
			ps88.OverlayValues[50] = d50
			ps88.OverlayValues[57] = d57
			ps88.OverlayValues[58] = d58
			ps88.OverlayValues[59] = d59
			ps88.OverlayValues[60] = d60
			ps88.OverlayValues[61] = d61
			ps88.OverlayValues[70] = d70
			ps88.OverlayValues[71] = d71
			ps88.OverlayValues[72] = d72
			ps88.OverlayValues[73] = d73
			ps88.OverlayValues[75] = d75
			ps88.OverlayValues[76] = d76
			ps88.OverlayValues[77] = d77
			ps88.OverlayValues[78] = d78
			ps88.OverlayValues[79] = d79
			ps88.OverlayValues[81] = d81
			ps88.OverlayValues[82] = d82
			ps88.OverlayValues[83] = d83
			ps88.OverlayValues[84] = d84
			snap89 := d36
			alloc90 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps88)
			}
			ctx.RestoreAllocState(alloc90)
			d36 = snap89
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps87)
			}
			return result
			ctx.FreeDesc(&d82)
			return result
			}
			bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[11].VisitCount >= 2 {
					ps.General = true
					return bbs[11].RenderPS(ps)
				}
			}
			bbs[11].VisitCount++
			if ps.General {
				if bbs[11].Rendered {
					ctx.W.EmitJmp(lbl12)
					return result
				}
				bbs[11].Rendered = true
				bbs[11].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_11 = bbs[11].Address
				ctx.W.MarkLabel(lbl12)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
			}
			ctx.ReclaimUntrackedRegs()
			var d91 JITValueDesc
			if d36.Loc == LocImm {
				d91 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d36.Imm.Float())}
			} else if d36.Type == tagFloat && d36.Loc == LocReg {
				d91 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d36.Reg}
				ctx.BindReg(d36.Reg, &d91)
				ctx.BindReg(d36.Reg, &d91)
			} else if d36.Type == tagFloat && d36.Loc == LocRegPair {
				ctx.FreeReg(d36.Reg)
				d91 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d36.Reg2}
				ctx.BindReg(d36.Reg2, &d91)
				ctx.BindReg(d36.Reg2, &d91)
			} else {
				d91 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d36}, 1)
				d91.Type = tagFloat
				ctx.BindReg(d91.Reg, &d91)
			}
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d91)
			var d92 JITValueDesc
			if d91.Loc == LocImm {
				d92 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d91.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d91)
				var truncSrc Reg
				if d91.Loc == LocRegPair {
					ctx.FreeReg(d91.Reg)
					truncSrc = d91.Reg2
				} else {
					truncSrc = d91.Reg
				}
				truncInt := ctx.AllocRegExcept(truncSrc)
				ctx.W.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
				ctx.W.EmitCvtInt64ToFloat64(RegX0, truncInt)
				d92 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
				ctx.BindReg(truncInt, &d92)
				ctx.BindReg(truncInt, &d92)
			}
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d92)
			var d93 JITValueDesc
			if d91.Loc == LocImm && d92.Loc == LocImm {
				d93 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d91.Imm.Float() == d92.Imm.Float())}
			} else if d92.Loc == LocImm {
				r15 := ctx.AllocRegExcept(d91.Reg)
				_, yBits := d92.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitCmpFloat64Setcc(r15, d91.Reg, RegR11, CcE)
				d93 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d93)
			} else if d91.Loc == LocImm {
				r16 := ctx.AllocRegExcept(d92.Reg)
				_, xBits := d91.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, xBits)
				ctx.W.EmitCmpFloat64Setcc(r16, RegR11, d92.Reg, CcE)
				d93 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d93)
			} else {
				r17 := ctx.AllocRegExcept(d91.Reg, d92.Reg)
				ctx.W.EmitCmpFloat64Setcc(r17, d91.Reg, d92.Reg, CcE)
				d93 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d93)
			}
			ctx.FreeDesc(&d92)
			d94 = d93
			ctx.EnsureDesc(&d94)
			if d94.Loc != LocImm && d94.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d94.Loc == LocImm {
				if d94.Imm.Bool() {
			ps95 := PhiState{General: ps.General}
			ps95.OverlayValues = make([]JITValueDesc, 95)
			ps95.OverlayValues[0] = d0
			ps95.OverlayValues[1] = d1
			ps95.OverlayValues[2] = d2
			ps95.OverlayValues[3] = d3
			ps95.OverlayValues[4] = d4
			ps95.OverlayValues[5] = d5
			ps95.OverlayValues[6] = d6
			ps95.OverlayValues[8] = d8
			ps95.OverlayValues[9] = d9
			ps95.OverlayValues[10] = d10
			ps95.OverlayValues[11] = d11
			ps95.OverlayValues[12] = d12
			ps95.OverlayValues[19] = d19
			ps95.OverlayValues[20] = d20
			ps95.OverlayValues[21] = d21
			ps95.OverlayValues[22] = d22
			ps95.OverlayValues[23] = d23
			ps95.OverlayValues[25] = d25
			ps95.OverlayValues[27] = d27
			ps95.OverlayValues[28] = d28
			ps95.OverlayValues[31] = d31
			ps95.OverlayValues[34] = d34
			ps95.OverlayValues[35] = d35
			ps95.OverlayValues[36] = d36
			ps95.OverlayValues[37] = d37
			ps95.OverlayValues[38] = d38
			ps95.OverlayValues[39] = d39
			ps95.OverlayValues[40] = d40
			ps95.OverlayValues[48] = d48
			ps95.OverlayValues[49] = d49
			ps95.OverlayValues[50] = d50
			ps95.OverlayValues[57] = d57
			ps95.OverlayValues[58] = d58
			ps95.OverlayValues[59] = d59
			ps95.OverlayValues[60] = d60
			ps95.OverlayValues[61] = d61
			ps95.OverlayValues[70] = d70
			ps95.OverlayValues[71] = d71
			ps95.OverlayValues[72] = d72
			ps95.OverlayValues[73] = d73
			ps95.OverlayValues[75] = d75
			ps95.OverlayValues[76] = d76
			ps95.OverlayValues[77] = d77
			ps95.OverlayValues[78] = d78
			ps95.OverlayValues[79] = d79
			ps95.OverlayValues[81] = d81
			ps95.OverlayValues[82] = d82
			ps95.OverlayValues[83] = d83
			ps95.OverlayValues[84] = d84
			ps95.OverlayValues[91] = d91
			ps95.OverlayValues[92] = d92
			ps95.OverlayValues[93] = d93
			ps95.OverlayValues[94] = d94
					return bbs[12].RenderPS(ps95)
				}
			ps96 := PhiState{General: ps.General}
			ps96.OverlayValues = make([]JITValueDesc, 95)
			ps96.OverlayValues[0] = d0
			ps96.OverlayValues[1] = d1
			ps96.OverlayValues[2] = d2
			ps96.OverlayValues[3] = d3
			ps96.OverlayValues[4] = d4
			ps96.OverlayValues[5] = d5
			ps96.OverlayValues[6] = d6
			ps96.OverlayValues[8] = d8
			ps96.OverlayValues[9] = d9
			ps96.OverlayValues[10] = d10
			ps96.OverlayValues[11] = d11
			ps96.OverlayValues[12] = d12
			ps96.OverlayValues[19] = d19
			ps96.OverlayValues[20] = d20
			ps96.OverlayValues[21] = d21
			ps96.OverlayValues[22] = d22
			ps96.OverlayValues[23] = d23
			ps96.OverlayValues[25] = d25
			ps96.OverlayValues[27] = d27
			ps96.OverlayValues[28] = d28
			ps96.OverlayValues[31] = d31
			ps96.OverlayValues[34] = d34
			ps96.OverlayValues[35] = d35
			ps96.OverlayValues[36] = d36
			ps96.OverlayValues[37] = d37
			ps96.OverlayValues[38] = d38
			ps96.OverlayValues[39] = d39
			ps96.OverlayValues[40] = d40
			ps96.OverlayValues[48] = d48
			ps96.OverlayValues[49] = d49
			ps96.OverlayValues[50] = d50
			ps96.OverlayValues[57] = d57
			ps96.OverlayValues[58] = d58
			ps96.OverlayValues[59] = d59
			ps96.OverlayValues[60] = d60
			ps96.OverlayValues[61] = d61
			ps96.OverlayValues[70] = d70
			ps96.OverlayValues[71] = d71
			ps96.OverlayValues[72] = d72
			ps96.OverlayValues[73] = d73
			ps96.OverlayValues[75] = d75
			ps96.OverlayValues[76] = d76
			ps96.OverlayValues[77] = d77
			ps96.OverlayValues[78] = d78
			ps96.OverlayValues[79] = d79
			ps96.OverlayValues[81] = d81
			ps96.OverlayValues[82] = d82
			ps96.OverlayValues[83] = d83
			ps96.OverlayValues[84] = d84
			ps96.OverlayValues[91] = d91
			ps96.OverlayValues[92] = d92
			ps96.OverlayValues[93] = d93
			ps96.OverlayValues[94] = d94
				return bbs[6].RenderPS(ps96)
			}
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d94.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl35)
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl35)
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl36)
			ctx.W.EmitJmp(lbl7)
			ps97 := PhiState{General: true}
			ps97.OverlayValues = make([]JITValueDesc, 95)
			ps97.OverlayValues[0] = d0
			ps97.OverlayValues[1] = d1
			ps97.OverlayValues[2] = d2
			ps97.OverlayValues[3] = d3
			ps97.OverlayValues[4] = d4
			ps97.OverlayValues[5] = d5
			ps97.OverlayValues[6] = d6
			ps97.OverlayValues[8] = d8
			ps97.OverlayValues[9] = d9
			ps97.OverlayValues[10] = d10
			ps97.OverlayValues[11] = d11
			ps97.OverlayValues[12] = d12
			ps97.OverlayValues[19] = d19
			ps97.OverlayValues[20] = d20
			ps97.OverlayValues[21] = d21
			ps97.OverlayValues[22] = d22
			ps97.OverlayValues[23] = d23
			ps97.OverlayValues[25] = d25
			ps97.OverlayValues[27] = d27
			ps97.OverlayValues[28] = d28
			ps97.OverlayValues[31] = d31
			ps97.OverlayValues[34] = d34
			ps97.OverlayValues[35] = d35
			ps97.OverlayValues[36] = d36
			ps97.OverlayValues[37] = d37
			ps97.OverlayValues[38] = d38
			ps97.OverlayValues[39] = d39
			ps97.OverlayValues[40] = d40
			ps97.OverlayValues[48] = d48
			ps97.OverlayValues[49] = d49
			ps97.OverlayValues[50] = d50
			ps97.OverlayValues[57] = d57
			ps97.OverlayValues[58] = d58
			ps97.OverlayValues[59] = d59
			ps97.OverlayValues[60] = d60
			ps97.OverlayValues[61] = d61
			ps97.OverlayValues[70] = d70
			ps97.OverlayValues[71] = d71
			ps97.OverlayValues[72] = d72
			ps97.OverlayValues[73] = d73
			ps97.OverlayValues[75] = d75
			ps97.OverlayValues[76] = d76
			ps97.OverlayValues[77] = d77
			ps97.OverlayValues[78] = d78
			ps97.OverlayValues[79] = d79
			ps97.OverlayValues[81] = d81
			ps97.OverlayValues[82] = d82
			ps97.OverlayValues[83] = d83
			ps97.OverlayValues[84] = d84
			ps97.OverlayValues[91] = d91
			ps97.OverlayValues[92] = d92
			ps97.OverlayValues[93] = d93
			ps97.OverlayValues[94] = d94
			ps98 := PhiState{General: true}
			ps98.OverlayValues = make([]JITValueDesc, 95)
			ps98.OverlayValues[0] = d0
			ps98.OverlayValues[1] = d1
			ps98.OverlayValues[2] = d2
			ps98.OverlayValues[3] = d3
			ps98.OverlayValues[4] = d4
			ps98.OverlayValues[5] = d5
			ps98.OverlayValues[6] = d6
			ps98.OverlayValues[8] = d8
			ps98.OverlayValues[9] = d9
			ps98.OverlayValues[10] = d10
			ps98.OverlayValues[11] = d11
			ps98.OverlayValues[12] = d12
			ps98.OverlayValues[19] = d19
			ps98.OverlayValues[20] = d20
			ps98.OverlayValues[21] = d21
			ps98.OverlayValues[22] = d22
			ps98.OverlayValues[23] = d23
			ps98.OverlayValues[25] = d25
			ps98.OverlayValues[27] = d27
			ps98.OverlayValues[28] = d28
			ps98.OverlayValues[31] = d31
			ps98.OverlayValues[34] = d34
			ps98.OverlayValues[35] = d35
			ps98.OverlayValues[36] = d36
			ps98.OverlayValues[37] = d37
			ps98.OverlayValues[38] = d38
			ps98.OverlayValues[39] = d39
			ps98.OverlayValues[40] = d40
			ps98.OverlayValues[48] = d48
			ps98.OverlayValues[49] = d49
			ps98.OverlayValues[50] = d50
			ps98.OverlayValues[57] = d57
			ps98.OverlayValues[58] = d58
			ps98.OverlayValues[59] = d59
			ps98.OverlayValues[60] = d60
			ps98.OverlayValues[61] = d61
			ps98.OverlayValues[70] = d70
			ps98.OverlayValues[71] = d71
			ps98.OverlayValues[72] = d72
			ps98.OverlayValues[73] = d73
			ps98.OverlayValues[75] = d75
			ps98.OverlayValues[76] = d76
			ps98.OverlayValues[77] = d77
			ps98.OverlayValues[78] = d78
			ps98.OverlayValues[79] = d79
			ps98.OverlayValues[81] = d81
			ps98.OverlayValues[82] = d82
			ps98.OverlayValues[83] = d83
			ps98.OverlayValues[84] = d84
			ps98.OverlayValues[91] = d91
			ps98.OverlayValues[92] = d92
			ps98.OverlayValues[93] = d93
			ps98.OverlayValues[94] = d94
			snap99 := d1
			snap100 := d91
			alloc101 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps98)
			}
			ctx.RestoreAllocState(alloc101)
			d1 = snap99
			d91 = snap100
			if !bbs[12].Rendered {
				return bbs[12].RenderPS(ps97)
			}
			return result
			ctx.FreeDesc(&d93)
			return result
			}
			bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[12].VisitCount >= 2 {
					ps.General = true
					return bbs[12].RenderPS(ps)
				}
			}
			bbs[12].VisitCount++
			if ps.General {
				if bbs[12].Rendered {
					ctx.W.EmitJmp(lbl13)
					return result
				}
				bbs[12].Rendered = true
				bbs[12].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_12 = bbs[12].Address
				ctx.W.MarkLabel(lbl13)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d91)
			var d102 JITValueDesc
			if d91.Loc == LocImm {
				d102 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d91.Imm.Float()))}
			} else {
				r18 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r18, d91.Reg)
				d102 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
				ctx.BindReg(r18, &d102)
			}
			ctx.FreeDesc(&d91)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d102)
			var d103 JITValueDesc
			if d1.Loc == LocImm && d102.Loc == LocImm {
				d103 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() * d102.Imm.Int())}
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d102.Reg)
				d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else if d102.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d102.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d102.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, RegR11)
				}
				d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else {
				r19 := ctx.AllocRegExcept(d1.Reg, d102.Reg)
				ctx.W.EmitMovRegReg(r19, d1.Reg)
				ctx.W.EmitImulInt64(r19, d102.Reg)
				d103 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
				ctx.BindReg(r19, &d103)
			}
			if d103.Loc == LocReg && d1.Loc == LocReg && d103.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d102)
			d104 = d103
			if d104.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d104)
			ctx.EmitStoreToStack(d104, 24)
			ps105 := PhiState{General: ps.General}
			ps105.OverlayValues = make([]JITValueDesc, 105)
			ps105.OverlayValues[0] = d0
			ps105.OverlayValues[1] = d1
			ps105.OverlayValues[2] = d2
			ps105.OverlayValues[3] = d3
			ps105.OverlayValues[4] = d4
			ps105.OverlayValues[5] = d5
			ps105.OverlayValues[6] = d6
			ps105.OverlayValues[8] = d8
			ps105.OverlayValues[9] = d9
			ps105.OverlayValues[10] = d10
			ps105.OverlayValues[11] = d11
			ps105.OverlayValues[12] = d12
			ps105.OverlayValues[19] = d19
			ps105.OverlayValues[20] = d20
			ps105.OverlayValues[21] = d21
			ps105.OverlayValues[22] = d22
			ps105.OverlayValues[23] = d23
			ps105.OverlayValues[25] = d25
			ps105.OverlayValues[27] = d27
			ps105.OverlayValues[28] = d28
			ps105.OverlayValues[31] = d31
			ps105.OverlayValues[34] = d34
			ps105.OverlayValues[35] = d35
			ps105.OverlayValues[36] = d36
			ps105.OverlayValues[37] = d37
			ps105.OverlayValues[38] = d38
			ps105.OverlayValues[39] = d39
			ps105.OverlayValues[40] = d40
			ps105.OverlayValues[48] = d48
			ps105.OverlayValues[49] = d49
			ps105.OverlayValues[50] = d50
			ps105.OverlayValues[57] = d57
			ps105.OverlayValues[58] = d58
			ps105.OverlayValues[59] = d59
			ps105.OverlayValues[60] = d60
			ps105.OverlayValues[61] = d61
			ps105.OverlayValues[70] = d70
			ps105.OverlayValues[71] = d71
			ps105.OverlayValues[72] = d72
			ps105.OverlayValues[73] = d73
			ps105.OverlayValues[75] = d75
			ps105.OverlayValues[76] = d76
			ps105.OverlayValues[77] = d77
			ps105.OverlayValues[78] = d78
			ps105.OverlayValues[79] = d79
			ps105.OverlayValues[81] = d81
			ps105.OverlayValues[82] = d82
			ps105.OverlayValues[83] = d83
			ps105.OverlayValues[84] = d84
			ps105.OverlayValues[91] = d91
			ps105.OverlayValues[92] = d92
			ps105.OverlayValues[93] = d93
			ps105.OverlayValues[94] = d94
			ps105.OverlayValues[102] = d102
			ps105.OverlayValues[103] = d103
			ps105.OverlayValues[104] = d104
			ps105.PhiValues = make([]JITValueDesc, 1)
			d106 = d103
			ps105.PhiValues[0] = d106
			if ps105.General && bbs[8].Rendered {
				ctx.W.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps105)
			return result
			}
			bbs[13].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[13].VisitCount >= 2 {
					ps.General = true
					return bbs[13].RenderPS(ps)
				}
			}
			bbs[13].VisitCount++
			if ps.General {
				if bbs[13].Rendered {
					ctx.W.EmitJmp(lbl14)
					return result
				}
				bbs[13].Rendered = true
				bbs[13].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_13 = bbs[13].Address
				ctx.W.MarkLabel(lbl14)
				ctx.W.ResolveFixups()
			}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
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
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
				d106 = ps.OverlayValues[106]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			ctx.W.EmitMakeInt(result, d1)
			if d1.Loc == LocReg { ctx.FreeReg(d1.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[14].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[14].VisitCount >= 2 {
					ps.General = true
					return bbs[14].RenderPS(ps)
				}
			}
			bbs[14].VisitCount++
			if ps.General {
				if bbs[14].Rendered {
					ctx.W.EmitJmp(lbl15)
					return result
				}
				bbs[14].Rendered = true
				bbs[14].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_14 = bbs[14].Address
				ctx.W.MarkLabel(lbl15)
				ctx.W.ResolveFixups()
			}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
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
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
				d104 = ps.OverlayValues[104]
			}
			if len(ps.OverlayValues) > 106 && ps.OverlayValues[106].Loc != LocNone {
				d106 = ps.OverlayValues[106]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d107 JITValueDesc
			if d1.Loc == LocImm {
				d107 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d1.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(RegX0, d1.Reg)
				d107 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d107)
			}
			d108 = d2
			if d108.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d108)
			ctx.EmitStoreToStack(d108, 32)
			d109 = d107
			if d109.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d109)
			ctx.EmitStoreToStack(d109, 40)
			ps110 := PhiState{General: ps.General}
			ps110.OverlayValues = make([]JITValueDesc, 110)
			ps110.OverlayValues[0] = d0
			ps110.OverlayValues[1] = d1
			ps110.OverlayValues[2] = d2
			ps110.OverlayValues[3] = d3
			ps110.OverlayValues[4] = d4
			ps110.OverlayValues[5] = d5
			ps110.OverlayValues[6] = d6
			ps110.OverlayValues[8] = d8
			ps110.OverlayValues[9] = d9
			ps110.OverlayValues[10] = d10
			ps110.OverlayValues[11] = d11
			ps110.OverlayValues[12] = d12
			ps110.OverlayValues[19] = d19
			ps110.OverlayValues[20] = d20
			ps110.OverlayValues[21] = d21
			ps110.OverlayValues[22] = d22
			ps110.OverlayValues[23] = d23
			ps110.OverlayValues[25] = d25
			ps110.OverlayValues[27] = d27
			ps110.OverlayValues[28] = d28
			ps110.OverlayValues[31] = d31
			ps110.OverlayValues[34] = d34
			ps110.OverlayValues[35] = d35
			ps110.OverlayValues[36] = d36
			ps110.OverlayValues[37] = d37
			ps110.OverlayValues[38] = d38
			ps110.OverlayValues[39] = d39
			ps110.OverlayValues[40] = d40
			ps110.OverlayValues[48] = d48
			ps110.OverlayValues[49] = d49
			ps110.OverlayValues[50] = d50
			ps110.OverlayValues[57] = d57
			ps110.OverlayValues[58] = d58
			ps110.OverlayValues[59] = d59
			ps110.OverlayValues[60] = d60
			ps110.OverlayValues[61] = d61
			ps110.OverlayValues[70] = d70
			ps110.OverlayValues[71] = d71
			ps110.OverlayValues[72] = d72
			ps110.OverlayValues[73] = d73
			ps110.OverlayValues[75] = d75
			ps110.OverlayValues[76] = d76
			ps110.OverlayValues[77] = d77
			ps110.OverlayValues[78] = d78
			ps110.OverlayValues[79] = d79
			ps110.OverlayValues[81] = d81
			ps110.OverlayValues[82] = d82
			ps110.OverlayValues[83] = d83
			ps110.OverlayValues[84] = d84
			ps110.OverlayValues[91] = d91
			ps110.OverlayValues[92] = d92
			ps110.OverlayValues[93] = d93
			ps110.OverlayValues[94] = d94
			ps110.OverlayValues[102] = d102
			ps110.OverlayValues[103] = d103
			ps110.OverlayValues[104] = d104
			ps110.OverlayValues[106] = d106
			ps110.OverlayValues[107] = d107
			ps110.OverlayValues[108] = d108
			ps110.OverlayValues[109] = d109
			ps110.PhiValues = make([]JITValueDesc, 2)
			d111 = d2
			ps110.PhiValues[0] = d111
			d112 = d107
			ps110.PhiValues[1] = d112
			if ps110.General && bbs[17].Rendered {
				ctx.W.EmitJmp(lbl18)
				return result
			}
			return bbs[17].RenderPS(ps110)
			return result
			}
			bbs[15].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[15].VisitCount >= 2 {
					ps.General = true
					return bbs[15].RenderPS(ps)
				}
			}
			bbs[15].VisitCount++
			if ps.General {
				if bbs[15].Rendered {
					ctx.W.EmitJmp(lbl16)
					return result
				}
				bbs[15].Rendered = true
				bbs[15].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_15 = bbs[15].Address
				ctx.W.MarkLabel(lbl16)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
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
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
				d104 = ps.OverlayValues[104]
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
			if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
				d111 = ps.OverlayValues[111]
			}
			if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
				d112 = ps.OverlayValues[112]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			var d113 JITValueDesc
			if d4.Loc == LocImm {
				idx := int(d4.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d113 = args[idx]
				d113.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d4.Reg] {
					ctx.ProtectReg(d4.Reg)
					seen[d4.Reg] = true
					protected = append(protected, d4.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r20 := ctx.AllocReg()
				r21 := ctx.AllocRegExcept(r20)
				lbl37 := ctx.W.ReserveLabel()
				lbl38 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d4.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl38)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d4.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r20, ai.Reg)
						ctx.W.EmitMovRegReg(r21, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r20, tmp.Reg)
						ctx.W.EmitMovRegReg(r21, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r20, Reg2: r21}
						ctx.BindReg(r20, &pair)
						ctx.BindReg(r21, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r20, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r21, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl37)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl38)
				d114 := JITValueDesc{Loc: LocRegPair, Reg: r20, Reg2: r21}
				ctx.BindReg(r20, &d114)
				ctx.BindReg(r21, &d114)
				ctx.BindReg(r20, &d114)
				ctx.BindReg(r21, &d114)
				ctx.W.EmitMakeNil(d114)
				ctx.W.MarkLabel(lbl37)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d113 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r20, Reg2: r21}
				ctx.BindReg(r20, &d113)
				ctx.BindReg(r21, &d113)
			}
			var d115 JITValueDesc
			if d113.Loc == LocImm {
				d115 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d113.Imm.Float())}
			} else if d113.Type == tagFloat && d113.Loc == LocReg {
				d115 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d113.Reg}
				ctx.BindReg(d113.Reg, &d115)
				ctx.BindReg(d113.Reg, &d115)
			} else if d113.Type == tagFloat && d113.Loc == LocRegPair {
				ctx.FreeReg(d113.Reg)
				d115 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d113.Reg2}
				ctx.BindReg(d113.Reg2, &d115)
				ctx.BindReg(d113.Reg2, &d115)
			} else {
				d115 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d113}, 1)
				d115.Type = tagFloat
				ctx.BindReg(d115.Reg, &d115)
			}
			ctx.FreeDesc(&d113)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d115)
			var d116 JITValueDesc
			if d5.Loc == LocImm && d115.Loc == LocImm {
				d116 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d5.Imm.Float() * d115.Imm.Float())}
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d115.Reg)
				_, xBits := d5.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitMulFloat64(scratch, d115.Reg)
				d116 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d116)
			} else if d115.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(scratch, d5.Reg)
				_, yBits := d115.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitMulFloat64(scratch, RegR11)
				d116 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d116)
			} else {
				r22 := ctx.AllocRegExcept(d5.Reg, d115.Reg)
				ctx.W.EmitMovRegReg(r22, d5.Reg)
				ctx.W.EmitMulFloat64(r22, d115.Reg)
				d116 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r22}
				ctx.BindReg(r22, &d116)
			}
			if d116.Loc == LocReg && d5.Loc == LocReg && d116.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d115)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d117 JITValueDesc
			if d4.Loc == LocImm {
				d117 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d117 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d117)
			}
			if d117.Loc == LocReg && d4.Loc == LocReg && d117.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			d118 = d117
			if d118.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d118)
			ctx.EmitStoreToStack(d118, 32)
			d119 = d116
			if d119.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d119)
			ctx.EmitStoreToStack(d119, 40)
			ps120 := PhiState{General: ps.General}
			ps120.OverlayValues = make([]JITValueDesc, 120)
			ps120.OverlayValues[0] = d0
			ps120.OverlayValues[1] = d1
			ps120.OverlayValues[2] = d2
			ps120.OverlayValues[3] = d3
			ps120.OverlayValues[4] = d4
			ps120.OverlayValues[5] = d5
			ps120.OverlayValues[6] = d6
			ps120.OverlayValues[8] = d8
			ps120.OverlayValues[9] = d9
			ps120.OverlayValues[10] = d10
			ps120.OverlayValues[11] = d11
			ps120.OverlayValues[12] = d12
			ps120.OverlayValues[19] = d19
			ps120.OverlayValues[20] = d20
			ps120.OverlayValues[21] = d21
			ps120.OverlayValues[22] = d22
			ps120.OverlayValues[23] = d23
			ps120.OverlayValues[25] = d25
			ps120.OverlayValues[27] = d27
			ps120.OverlayValues[28] = d28
			ps120.OverlayValues[31] = d31
			ps120.OverlayValues[34] = d34
			ps120.OverlayValues[35] = d35
			ps120.OverlayValues[36] = d36
			ps120.OverlayValues[37] = d37
			ps120.OverlayValues[38] = d38
			ps120.OverlayValues[39] = d39
			ps120.OverlayValues[40] = d40
			ps120.OverlayValues[48] = d48
			ps120.OverlayValues[49] = d49
			ps120.OverlayValues[50] = d50
			ps120.OverlayValues[57] = d57
			ps120.OverlayValues[58] = d58
			ps120.OverlayValues[59] = d59
			ps120.OverlayValues[60] = d60
			ps120.OverlayValues[61] = d61
			ps120.OverlayValues[70] = d70
			ps120.OverlayValues[71] = d71
			ps120.OverlayValues[72] = d72
			ps120.OverlayValues[73] = d73
			ps120.OverlayValues[75] = d75
			ps120.OverlayValues[76] = d76
			ps120.OverlayValues[77] = d77
			ps120.OverlayValues[78] = d78
			ps120.OverlayValues[79] = d79
			ps120.OverlayValues[81] = d81
			ps120.OverlayValues[82] = d82
			ps120.OverlayValues[83] = d83
			ps120.OverlayValues[84] = d84
			ps120.OverlayValues[91] = d91
			ps120.OverlayValues[92] = d92
			ps120.OverlayValues[93] = d93
			ps120.OverlayValues[94] = d94
			ps120.OverlayValues[102] = d102
			ps120.OverlayValues[103] = d103
			ps120.OverlayValues[104] = d104
			ps120.OverlayValues[106] = d106
			ps120.OverlayValues[107] = d107
			ps120.OverlayValues[108] = d108
			ps120.OverlayValues[109] = d109
			ps120.OverlayValues[111] = d111
			ps120.OverlayValues[112] = d112
			ps120.OverlayValues[113] = d113
			ps120.OverlayValues[114] = d114
			ps120.OverlayValues[115] = d115
			ps120.OverlayValues[116] = d116
			ps120.OverlayValues[117] = d117
			ps120.OverlayValues[118] = d118
			ps120.OverlayValues[119] = d119
			ps120.PhiValues = make([]JITValueDesc, 2)
			d121 = d117
			ps120.PhiValues[0] = d121
			d122 = d116
			ps120.PhiValues[1] = d122
			if ps120.General && bbs[17].Rendered {
				ctx.W.EmitJmp(lbl18)
				return result
			}
			return bbs[17].RenderPS(ps120)
			return result
			}
			bbs[16].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[16].VisitCount >= 2 {
					ps.General = true
					return bbs[16].RenderPS(ps)
				}
			}
			bbs[16].VisitCount++
			if ps.General {
				if bbs[16].Rendered {
					ctx.W.EmitJmp(lbl17)
					return result
				}
				bbs[16].Rendered = true
				bbs[16].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_16 = bbs[16].Address
				ctx.W.MarkLabel(lbl17)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
			}
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
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
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
				d104 = ps.OverlayValues[104]
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
			if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != LocNone {
				d121 = ps.OverlayValues[121]
			}
			if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != LocNone {
				d122 = ps.OverlayValues[122]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			ctx.W.EmitMakeFloat(result, d5)
			if d5.Loc == LocReg { ctx.FreeReg(d5.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			return result
			}
			bbs[17].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[17].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d123 := ps.PhiValues[0]
						ctx.EnsureDesc(&d123)
						ctx.EmitStoreToStack(d123, 32)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d124 := ps.PhiValues[1]
						ctx.EnsureDesc(&d124)
						ctx.EmitStoreToStack(d124, 40)
					}
					ps.General = true
					return bbs[17].RenderPS(ps)
				}
			}
			bbs[17].VisitCount++
			if ps.General {
				if bbs[17].Rendered {
					ctx.W.EmitJmp(lbl18)
					return result
				}
				bbs[17].Rendered = true
				bbs[17].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_17 = bbs[17].Address
				ctx.W.MarkLabel(lbl18)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(8)}
			d2 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d3 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(24)}
			d4 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(32)}
			d5 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(40)}
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
			if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
				d3 = ps.OverlayValues[3]
			}
			if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
				d6 = ps.OverlayValues[6]
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
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
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
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
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
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
				d84 = ps.OverlayValues[84]
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
			if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
				d102 = ps.OverlayValues[102]
			}
			if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
				d103 = ps.OverlayValues[103]
			}
			if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
				d104 = ps.OverlayValues[104]
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d4 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d5 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d125 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d125)
			var d126 JITValueDesc
			if d4.Loc == LocImm && d125.Loc == LocImm {
				d126 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d125.Imm.Int())}
			} else if d125.Loc == LocImm {
				r23 := ctx.AllocRegExcept(d4.Reg)
				if d125.Imm.Int() >= -2147483648 && d125.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d4.Reg, int32(d125.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d125.Imm.Int()))
					ctx.W.EmitCmpInt64(d4.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r23, CcL)
				d126 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d126)
			} else if d4.Loc == LocImm {
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d125.Reg)
				ctx.W.EmitSetcc(r24, CcL)
				d126 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
				ctx.BindReg(r24, &d126)
			} else {
				r25 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitCmpInt64(d4.Reg, d125.Reg)
				ctx.W.EmitSetcc(r25, CcL)
				d126 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
				ctx.BindReg(r25, &d126)
			}
			ctx.FreeDesc(&d125)
			d127 = d126
			ctx.EnsureDesc(&d127)
			if d127.Loc != LocImm && d127.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d127.Loc == LocImm {
				if d127.Imm.Bool() {
			ps128 := PhiState{General: ps.General}
			ps128.OverlayValues = make([]JITValueDesc, 128)
			ps128.OverlayValues[0] = d0
			ps128.OverlayValues[1] = d1
			ps128.OverlayValues[2] = d2
			ps128.OverlayValues[3] = d3
			ps128.OverlayValues[4] = d4
			ps128.OverlayValues[5] = d5
			ps128.OverlayValues[6] = d6
			ps128.OverlayValues[8] = d8
			ps128.OverlayValues[9] = d9
			ps128.OverlayValues[10] = d10
			ps128.OverlayValues[11] = d11
			ps128.OverlayValues[12] = d12
			ps128.OverlayValues[19] = d19
			ps128.OverlayValues[20] = d20
			ps128.OverlayValues[21] = d21
			ps128.OverlayValues[22] = d22
			ps128.OverlayValues[23] = d23
			ps128.OverlayValues[25] = d25
			ps128.OverlayValues[27] = d27
			ps128.OverlayValues[28] = d28
			ps128.OverlayValues[31] = d31
			ps128.OverlayValues[34] = d34
			ps128.OverlayValues[35] = d35
			ps128.OverlayValues[36] = d36
			ps128.OverlayValues[37] = d37
			ps128.OverlayValues[38] = d38
			ps128.OverlayValues[39] = d39
			ps128.OverlayValues[40] = d40
			ps128.OverlayValues[48] = d48
			ps128.OverlayValues[49] = d49
			ps128.OverlayValues[50] = d50
			ps128.OverlayValues[57] = d57
			ps128.OverlayValues[58] = d58
			ps128.OverlayValues[59] = d59
			ps128.OverlayValues[60] = d60
			ps128.OverlayValues[61] = d61
			ps128.OverlayValues[70] = d70
			ps128.OverlayValues[71] = d71
			ps128.OverlayValues[72] = d72
			ps128.OverlayValues[73] = d73
			ps128.OverlayValues[75] = d75
			ps128.OverlayValues[76] = d76
			ps128.OverlayValues[77] = d77
			ps128.OverlayValues[78] = d78
			ps128.OverlayValues[79] = d79
			ps128.OverlayValues[81] = d81
			ps128.OverlayValues[82] = d82
			ps128.OverlayValues[83] = d83
			ps128.OverlayValues[84] = d84
			ps128.OverlayValues[91] = d91
			ps128.OverlayValues[92] = d92
			ps128.OverlayValues[93] = d93
			ps128.OverlayValues[94] = d94
			ps128.OverlayValues[102] = d102
			ps128.OverlayValues[103] = d103
			ps128.OverlayValues[104] = d104
			ps128.OverlayValues[106] = d106
			ps128.OverlayValues[107] = d107
			ps128.OverlayValues[108] = d108
			ps128.OverlayValues[109] = d109
			ps128.OverlayValues[111] = d111
			ps128.OverlayValues[112] = d112
			ps128.OverlayValues[113] = d113
			ps128.OverlayValues[114] = d114
			ps128.OverlayValues[115] = d115
			ps128.OverlayValues[116] = d116
			ps128.OverlayValues[117] = d117
			ps128.OverlayValues[118] = d118
			ps128.OverlayValues[119] = d119
			ps128.OverlayValues[121] = d121
			ps128.OverlayValues[122] = d122
			ps128.OverlayValues[123] = d123
			ps128.OverlayValues[124] = d124
			ps128.OverlayValues[125] = d125
			ps128.OverlayValues[126] = d126
			ps128.OverlayValues[127] = d127
					return bbs[15].RenderPS(ps128)
				}
			ps129 := PhiState{General: ps.General}
			ps129.OverlayValues = make([]JITValueDesc, 128)
			ps129.OverlayValues[0] = d0
			ps129.OverlayValues[1] = d1
			ps129.OverlayValues[2] = d2
			ps129.OverlayValues[3] = d3
			ps129.OverlayValues[4] = d4
			ps129.OverlayValues[5] = d5
			ps129.OverlayValues[6] = d6
			ps129.OverlayValues[8] = d8
			ps129.OverlayValues[9] = d9
			ps129.OverlayValues[10] = d10
			ps129.OverlayValues[11] = d11
			ps129.OverlayValues[12] = d12
			ps129.OverlayValues[19] = d19
			ps129.OverlayValues[20] = d20
			ps129.OverlayValues[21] = d21
			ps129.OverlayValues[22] = d22
			ps129.OverlayValues[23] = d23
			ps129.OverlayValues[25] = d25
			ps129.OverlayValues[27] = d27
			ps129.OverlayValues[28] = d28
			ps129.OverlayValues[31] = d31
			ps129.OverlayValues[34] = d34
			ps129.OverlayValues[35] = d35
			ps129.OverlayValues[36] = d36
			ps129.OverlayValues[37] = d37
			ps129.OverlayValues[38] = d38
			ps129.OverlayValues[39] = d39
			ps129.OverlayValues[40] = d40
			ps129.OverlayValues[48] = d48
			ps129.OverlayValues[49] = d49
			ps129.OverlayValues[50] = d50
			ps129.OverlayValues[57] = d57
			ps129.OverlayValues[58] = d58
			ps129.OverlayValues[59] = d59
			ps129.OverlayValues[60] = d60
			ps129.OverlayValues[61] = d61
			ps129.OverlayValues[70] = d70
			ps129.OverlayValues[71] = d71
			ps129.OverlayValues[72] = d72
			ps129.OverlayValues[73] = d73
			ps129.OverlayValues[75] = d75
			ps129.OverlayValues[76] = d76
			ps129.OverlayValues[77] = d77
			ps129.OverlayValues[78] = d78
			ps129.OverlayValues[79] = d79
			ps129.OverlayValues[81] = d81
			ps129.OverlayValues[82] = d82
			ps129.OverlayValues[83] = d83
			ps129.OverlayValues[84] = d84
			ps129.OverlayValues[91] = d91
			ps129.OverlayValues[92] = d92
			ps129.OverlayValues[93] = d93
			ps129.OverlayValues[94] = d94
			ps129.OverlayValues[102] = d102
			ps129.OverlayValues[103] = d103
			ps129.OverlayValues[104] = d104
			ps129.OverlayValues[106] = d106
			ps129.OverlayValues[107] = d107
			ps129.OverlayValues[108] = d108
			ps129.OverlayValues[109] = d109
			ps129.OverlayValues[111] = d111
			ps129.OverlayValues[112] = d112
			ps129.OverlayValues[113] = d113
			ps129.OverlayValues[114] = d114
			ps129.OverlayValues[115] = d115
			ps129.OverlayValues[116] = d116
			ps129.OverlayValues[117] = d117
			ps129.OverlayValues[118] = d118
			ps129.OverlayValues[119] = d119
			ps129.OverlayValues[121] = d121
			ps129.OverlayValues[122] = d122
			ps129.OverlayValues[123] = d123
			ps129.OverlayValues[124] = d124
			ps129.OverlayValues[125] = d125
			ps129.OverlayValues[126] = d126
			ps129.OverlayValues[127] = d127
				return bbs[16].RenderPS(ps129)
			}
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d127.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl39)
			ctx.W.EmitJmp(lbl40)
			ctx.W.MarkLabel(lbl39)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl40)
			ctx.W.EmitJmp(lbl17)
			ps130 := PhiState{General: true}
			ps130.OverlayValues = make([]JITValueDesc, 128)
			ps130.OverlayValues[0] = d0
			ps130.OverlayValues[1] = d1
			ps130.OverlayValues[2] = d2
			ps130.OverlayValues[3] = d3
			ps130.OverlayValues[4] = d4
			ps130.OverlayValues[5] = d5
			ps130.OverlayValues[6] = d6
			ps130.OverlayValues[8] = d8
			ps130.OverlayValues[9] = d9
			ps130.OverlayValues[10] = d10
			ps130.OverlayValues[11] = d11
			ps130.OverlayValues[12] = d12
			ps130.OverlayValues[19] = d19
			ps130.OverlayValues[20] = d20
			ps130.OverlayValues[21] = d21
			ps130.OverlayValues[22] = d22
			ps130.OverlayValues[23] = d23
			ps130.OverlayValues[25] = d25
			ps130.OverlayValues[27] = d27
			ps130.OverlayValues[28] = d28
			ps130.OverlayValues[31] = d31
			ps130.OverlayValues[34] = d34
			ps130.OverlayValues[35] = d35
			ps130.OverlayValues[36] = d36
			ps130.OverlayValues[37] = d37
			ps130.OverlayValues[38] = d38
			ps130.OverlayValues[39] = d39
			ps130.OverlayValues[40] = d40
			ps130.OverlayValues[48] = d48
			ps130.OverlayValues[49] = d49
			ps130.OverlayValues[50] = d50
			ps130.OverlayValues[57] = d57
			ps130.OverlayValues[58] = d58
			ps130.OverlayValues[59] = d59
			ps130.OverlayValues[60] = d60
			ps130.OverlayValues[61] = d61
			ps130.OverlayValues[70] = d70
			ps130.OverlayValues[71] = d71
			ps130.OverlayValues[72] = d72
			ps130.OverlayValues[73] = d73
			ps130.OverlayValues[75] = d75
			ps130.OverlayValues[76] = d76
			ps130.OverlayValues[77] = d77
			ps130.OverlayValues[78] = d78
			ps130.OverlayValues[79] = d79
			ps130.OverlayValues[81] = d81
			ps130.OverlayValues[82] = d82
			ps130.OverlayValues[83] = d83
			ps130.OverlayValues[84] = d84
			ps130.OverlayValues[91] = d91
			ps130.OverlayValues[92] = d92
			ps130.OverlayValues[93] = d93
			ps130.OverlayValues[94] = d94
			ps130.OverlayValues[102] = d102
			ps130.OverlayValues[103] = d103
			ps130.OverlayValues[104] = d104
			ps130.OverlayValues[106] = d106
			ps130.OverlayValues[107] = d107
			ps130.OverlayValues[108] = d108
			ps130.OverlayValues[109] = d109
			ps130.OverlayValues[111] = d111
			ps130.OverlayValues[112] = d112
			ps130.OverlayValues[113] = d113
			ps130.OverlayValues[114] = d114
			ps130.OverlayValues[115] = d115
			ps130.OverlayValues[116] = d116
			ps130.OverlayValues[117] = d117
			ps130.OverlayValues[118] = d118
			ps130.OverlayValues[119] = d119
			ps130.OverlayValues[121] = d121
			ps130.OverlayValues[122] = d122
			ps130.OverlayValues[123] = d123
			ps130.OverlayValues[124] = d124
			ps130.OverlayValues[125] = d125
			ps130.OverlayValues[126] = d126
			ps130.OverlayValues[127] = d127
			ps131 := PhiState{General: true}
			ps131.OverlayValues = make([]JITValueDesc, 128)
			ps131.OverlayValues[0] = d0
			ps131.OverlayValues[1] = d1
			ps131.OverlayValues[2] = d2
			ps131.OverlayValues[3] = d3
			ps131.OverlayValues[4] = d4
			ps131.OverlayValues[5] = d5
			ps131.OverlayValues[6] = d6
			ps131.OverlayValues[8] = d8
			ps131.OverlayValues[9] = d9
			ps131.OverlayValues[10] = d10
			ps131.OverlayValues[11] = d11
			ps131.OverlayValues[12] = d12
			ps131.OverlayValues[19] = d19
			ps131.OverlayValues[20] = d20
			ps131.OverlayValues[21] = d21
			ps131.OverlayValues[22] = d22
			ps131.OverlayValues[23] = d23
			ps131.OverlayValues[25] = d25
			ps131.OverlayValues[27] = d27
			ps131.OverlayValues[28] = d28
			ps131.OverlayValues[31] = d31
			ps131.OverlayValues[34] = d34
			ps131.OverlayValues[35] = d35
			ps131.OverlayValues[36] = d36
			ps131.OverlayValues[37] = d37
			ps131.OverlayValues[38] = d38
			ps131.OverlayValues[39] = d39
			ps131.OverlayValues[40] = d40
			ps131.OverlayValues[48] = d48
			ps131.OverlayValues[49] = d49
			ps131.OverlayValues[50] = d50
			ps131.OverlayValues[57] = d57
			ps131.OverlayValues[58] = d58
			ps131.OverlayValues[59] = d59
			ps131.OverlayValues[60] = d60
			ps131.OverlayValues[61] = d61
			ps131.OverlayValues[70] = d70
			ps131.OverlayValues[71] = d71
			ps131.OverlayValues[72] = d72
			ps131.OverlayValues[73] = d73
			ps131.OverlayValues[75] = d75
			ps131.OverlayValues[76] = d76
			ps131.OverlayValues[77] = d77
			ps131.OverlayValues[78] = d78
			ps131.OverlayValues[79] = d79
			ps131.OverlayValues[81] = d81
			ps131.OverlayValues[82] = d82
			ps131.OverlayValues[83] = d83
			ps131.OverlayValues[84] = d84
			ps131.OverlayValues[91] = d91
			ps131.OverlayValues[92] = d92
			ps131.OverlayValues[93] = d93
			ps131.OverlayValues[94] = d94
			ps131.OverlayValues[102] = d102
			ps131.OverlayValues[103] = d103
			ps131.OverlayValues[104] = d104
			ps131.OverlayValues[106] = d106
			ps131.OverlayValues[107] = d107
			ps131.OverlayValues[108] = d108
			ps131.OverlayValues[109] = d109
			ps131.OverlayValues[111] = d111
			ps131.OverlayValues[112] = d112
			ps131.OverlayValues[113] = d113
			ps131.OverlayValues[114] = d114
			ps131.OverlayValues[115] = d115
			ps131.OverlayValues[116] = d116
			ps131.OverlayValues[117] = d117
			ps131.OverlayValues[118] = d118
			ps131.OverlayValues[119] = d119
			ps131.OverlayValues[121] = d121
			ps131.OverlayValues[122] = d122
			ps131.OverlayValues[123] = d123
			ps131.OverlayValues[124] = d124
			ps131.OverlayValues[125] = d125
			ps131.OverlayValues[126] = d126
			ps131.OverlayValues[127] = d127
			snap132 := d4
			snap133 := d5
			snap134 := d113
			snap135 := d115
			alloc136 := ctx.SnapshotAllocState()
			if !bbs[16].Rendered {
				bbs[16].RenderPS(ps131)
			}
			ctx.RestoreAllocState(alloc136)
			d4 = snap132
			d5 = snap133
			d113 = snap134
			d115 = snap135
			if !bbs[15].Rendered {
				return bbs[15].RenderPS(ps130)
			}
			return result
			ctx.FreeDesc(&d126)
			return result
			}
			ps137 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps137)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(48))
			ctx.W.EmitAddRSP32(int32(48))
			return result
		}, /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */ /* TODO: unsupported call: archTrunc(x) */
	})
	Declare(&Globalenv, &Declaration{
		"/", "divides two or more numbers from the first one",
		2, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number", "values", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			// Nil short-circuit
			for _, v := range a {
				if v.IsNil() {
					return NewNil()
				}
			}
			v := a[0].Float()
			for _, i := range a[1:] {
				v /= i.Float()
			}
			return NewFloat(v)
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice a[1:int:] */, /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */
	})
	Declare(&Globalenv, &Declaration{
		"<=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!Less(a[1], a[0]))
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[1]
			d0.ID = 0
			d1 = args[0]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d2.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d2.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d2.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d2.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps4 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps4)
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
	})
	Declare(&Globalenv, &Declaration{
		"<", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Less(a[0], a[1]))
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
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps3 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps3)
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
	})
	Declare(&Globalenv, &Declaration{
		">", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Less(a[1], a[0]))
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
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[1]
			d0.ID = 0
			d1 = args[0]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps3 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps3)
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
	})
	Declare(&Globalenv, &Declaration{
		">=", "compares two numbers or strings",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!Less(a[0], a[1]))
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d2 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d3 JITValueDesc
			if d2.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d2.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d2.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d2.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d2.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps4 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps4)
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
	})
	Declare(&Globalenv, &Declaration{
		"equal?", "compares two values of the same type, (equal? nil nil) is true",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(Equal(a[0], a[1]))
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
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Equal arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Equal arg1)")
			}
			d2 = ctx.EmitGoCallScalar(GoFuncAddr(Equal), []JITValueDesc{d0, d1}, 1)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d2)
			} else {
				ctx.W.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps3 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps3)
			return result
		}, /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */ /* TODO: unresolved SSA value: github.com/launix-de/memcp/scm.scmerIntSentinel */
	})
	Declare(&Globalenv, &Declaration{
		"equal??", "performs a SQL compliant sloppy equality check on primitive values (number, int, string, bool. nil), strings are compared case insensitive, (equal? nil nil) is nil",
		2, 2,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "any", "values", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return EqualSQL(a[0], a[1])
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
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (EqualSQL arg0)")
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d1.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d1)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d1)
				d1 = tmpPair
			}
			if d1.Loc != LocRegPair && d1.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (EqualSQL arg1)")
			}
			d2 = ctx.EmitGoCallScalar(GoFuncAddr(EqualSQL), []JITValueDesc{d0, d1}, 2)
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d1)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d2, &result)
				result.Type = d2.Type
			} else {
				switch d2.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d2)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d2)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d2)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					panic("jit: single-block scalar return with unknown type")
				}
			}
			return result
			return result
			}
			ps3 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps3)
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
	})
	Declare(&Globalenv, &Declaration{
		"equal_collate", "performs SQL equality with a specified collation (e.g. *_ci case-insensitive, *_bin case-sensitive); returns nil if either arg is nil",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"a", "any", "left side", nil},
			DeclarationParameter{"b", "any", "right side", nil},
			DeclarationParameter{"collation", "string", "collation name", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			coll := strings.ToLower(String(a[2]))
			ta := a[0].GetTag()
			tb := a[1].GetTag()
			if (ta == tagString || ta == tagSymbol) && (tb == tagString || tb == tagSymbol) {
				as := a[0].String()
				bs := a[1].String()
				if strings.Contains(coll, "_ci") {
					return NewBool(strings.EqualFold(as, bs))
				}
				return NewBool(as == bs)
			}
			return EqualSQL(a[0], a[1])
		},
		true, false, nil,
		nil /* TODO: unsupported compare const kind: (Scmer).String(t22) */, /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
	})
	Declare(&Globalenv, &Declaration{
		"notequal_collate", "performs SQL inequality with a specified collation; returns nil if either arg is nil",
		3, 3,
		[]DeclarationParameter{
			DeclarationParameter{"a", "any", "left side", nil},
			DeclarationParameter{"b", "any", "right side", nil},
			DeclarationParameter{"collation", "string", "collation name", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			r := Globalenv.Vars["equal_collate"].Func()(a[0], a[1], a[2])
			if r.IsNil() {
				return r
			}
			return NewBool(!r.Bool())
		},
		true, false, nil,
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
	})
	Declare(&Globalenv, &Declaration{
		"!", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			d2.ID = 0
			d1 = ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d1.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d1.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d1.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps4 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps4)
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
	})
	Declare(&Globalenv, &Declaration{
		"not", "negates the boolean value",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "bool", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(!a[0].Bool())
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			d2.ID = 0
			d1 = ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.W.EmitMovRegReg(negReg, d1.Reg2)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d1.Loc == LocReg {
					ctx.W.EmitMovRegReg(negReg, d1.Reg)
					ctx.W.EmitAndRegImm32(negReg, 1)
					ctx.W.EmitCmpRegImm32(negReg, 0)
					ctx.W.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d3)
			} else {
				ctx.W.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps4 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps4)
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
	})
	Declare(&Globalenv, &Declaration{
		"nil?", "returns true if value is nil",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "any", "value", nil},
		}, "bool",
		func(a ...Scmer) Scmer {
			return NewBool(a[0].IsNil())
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
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			d2.ID = 0
			d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d1.Loc == LocImm {
				ctx.W.EmitMakeBool(result, d1)
			} else {
				ctx.W.EmitMakeBool(result, d1)
				ctx.FreeReg(d1.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			ps3 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps3)
			return result
		},
	})
	Declare(&Globalenv, &Declaration{
		"min", "returns the smallest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value", nil},
		}, "number|string",
		func(a ...Scmer) Scmer {
			var result Scmer
			for _, v := range a {
				if result.IsNil() {
					result = v
				} else if !v.IsNil() && Less(v, result) {
					result = v
				}
			}
			return result
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d2 JITValueDesc
			_ = d2
			var d4 JITValueDesc
			_ = d4
			var d5 JITValueDesc
			_ = d5
			var d6 JITValueDesc
			_ = d6
			var d7 JITValueDesc
			_ = d7
			var d8 JITValueDesc
			_ = d8
			var d9 JITValueDesc
			_ = d9
			var d10 JITValueDesc
			_ = d10
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
			var d28 JITValueDesc
			_ = d28
			var d29 JITValueDesc
			_ = d29
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
			var d36 JITValueDesc
			_ = d36
			var d37 JITValueDesc
			_ = d37
			var d39 JITValueDesc
			_ = d39
			var d40 JITValueDesc
			_ = d40
			var d42 JITValueDesc
			_ = d42
			var d43 JITValueDesc
			_ = d43
			var d46 JITValueDesc
			_ = d46
			var d47 JITValueDesc
			_ = d47
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
			var d62 JITValueDesc
			_ = d62
			var d63 JITValueDesc
			_ = d63
			var d64 JITValueDesc
			_ = d64
			var d65 JITValueDesc
			_ = d65
			var d68 JITValueDesc
			_ = d68
			var d69 JITValueDesc
			_ = d69
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			var bbs [8]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 16)
			ps3 := PhiState{General: ps.General}
			ps3.OverlayValues = make([]JITValueDesc, 3)
			ps3.OverlayValues[0] = d0
			ps3.OverlayValues[1] = d1
			ps3.OverlayValues[2] = d2
			ps3.PhiValues = make([]JITValueDesc, 2)
			d4 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			ps3.PhiValues[0] = d4
			d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
			ps3.PhiValues[1] = d5
			if ps3.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps3)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d6 := ps.PhiValues[0]
						ctx.EnsureDesc(&d6)
						ctx.EmitStoreScmerToStack(d6, 0)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d7 := ps.PhiValues[1]
						ctx.EnsureDesc(&d7)
						ctx.EmitStoreToStack(d7, 16)
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d1 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d8 JITValueDesc
			if d1.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			}
			if d8.Loc == LocReg && d1.Loc == LocReg && d8.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d2)
			var d9 JITValueDesc
			if d8.Loc == LocImm && d2.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < d2.Imm.Int())}
			} else if d2.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d8.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d8.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d8.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d9)
			} else if d8.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d2.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d9)
			} else {
				r3 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitCmpInt64(d8.Reg, d2.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d9)
			}
			ctx.FreeDesc(&d2)
			d10 = d9
			ctx.EnsureDesc(&d10)
			if d10.Loc != LocImm && d10.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d10.Loc == LocImm {
				if d10.Imm.Bool() {
			ps11 := PhiState{General: ps.General}
			ps11.OverlayValues = make([]JITValueDesc, 11)
			ps11.OverlayValues[0] = d0
			ps11.OverlayValues[1] = d1
			ps11.OverlayValues[2] = d2
			ps11.OverlayValues[4] = d4
			ps11.OverlayValues[5] = d5
			ps11.OverlayValues[6] = d6
			ps11.OverlayValues[7] = d7
			ps11.OverlayValues[8] = d8
			ps11.OverlayValues[9] = d9
			ps11.OverlayValues[10] = d10
					return bbs[2].RenderPS(ps11)
				}
			ps12 := PhiState{General: ps.General}
			ps12.OverlayValues = make([]JITValueDesc, 11)
			ps12.OverlayValues[0] = d0
			ps12.OverlayValues[1] = d1
			ps12.OverlayValues[2] = d2
			ps12.OverlayValues[4] = d4
			ps12.OverlayValues[5] = d5
			ps12.OverlayValues[6] = d6
			ps12.OverlayValues[7] = d7
			ps12.OverlayValues[8] = d8
			ps12.OverlayValues[9] = d9
			ps12.OverlayValues[10] = d10
				return bbs[3].RenderPS(ps12)
			}
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d10.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl9)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl9)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitJmp(lbl4)
			ps13 := PhiState{General: true}
			ps13.OverlayValues = make([]JITValueDesc, 11)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[4] = d4
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[6] = d6
			ps13.OverlayValues[7] = d7
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps14 := PhiState{General: true}
			ps14.OverlayValues = make([]JITValueDesc, 11)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[6] = d6
			ps14.OverlayValues[7] = d7
			ps14.OverlayValues[8] = d8
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			snap15 := d0
			snap16 := d8
			alloc17 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps14)
			}
			ctx.RestoreAllocState(alloc17)
			d0 = snap15
			d8 = snap16
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps13)
			}
			return result
			ctx.FreeDesc(&d9)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			var d18 JITValueDesc
			if d8.Loc == LocImm {
				idx := int(d8.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d18 = args[idx]
				d18.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d8.Reg] {
					ctx.ProtectReg(d8.Reg)
					seen[d8.Reg] = true
					protected = append(protected, d8.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl11 := ctx.W.ReserveLabel()
				lbl12 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d8.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl12)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d8.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl11)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl12)
				d19 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d19)
				ctx.BindReg(r5, &d19)
				ctx.BindReg(r4, &d19)
				ctx.BindReg(r5, &d19)
				ctx.W.EmitMakeNil(d19)
				ctx.W.MarkLabel(lbl11)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d18 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d18)
				ctx.BindReg(r5, &d18)
			}
			d21 = d0
			d21.ID = 0
			d20 = ctx.EmitTagEqualsBorrowed(&d21, tagNil, JITValueDesc{Loc: LocAny})
			d22 = d20
			ctx.EnsureDesc(&d22)
			if d22.Loc != LocImm && d22.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d22.Loc == LocImm {
				if d22.Imm.Bool() {
			ps23 := PhiState{General: ps.General}
			ps23.OverlayValues = make([]JITValueDesc, 23)
			ps23.OverlayValues[0] = d0
			ps23.OverlayValues[1] = d1
			ps23.OverlayValues[2] = d2
			ps23.OverlayValues[4] = d4
			ps23.OverlayValues[5] = d5
			ps23.OverlayValues[6] = d6
			ps23.OverlayValues[7] = d7
			ps23.OverlayValues[8] = d8
			ps23.OverlayValues[9] = d9
			ps23.OverlayValues[10] = d10
			ps23.OverlayValues[18] = d18
			ps23.OverlayValues[19] = d19
			ps23.OverlayValues[20] = d20
			ps23.OverlayValues[21] = d21
			ps23.OverlayValues[22] = d22
					return bbs[4].RenderPS(ps23)
				}
			ps24 := PhiState{General: ps.General}
			ps24.OverlayValues = make([]JITValueDesc, 23)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[4] = d4
			ps24.OverlayValues[5] = d5
			ps24.OverlayValues[6] = d6
			ps24.OverlayValues[7] = d7
			ps24.OverlayValues[8] = d8
			ps24.OverlayValues[9] = d9
			ps24.OverlayValues[10] = d10
			ps24.OverlayValues[18] = d18
			ps24.OverlayValues[19] = d19
			ps24.OverlayValues[20] = d20
			ps24.OverlayValues[21] = d21
			ps24.OverlayValues[22] = d22
				return bbs[5].RenderPS(ps24)
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d22.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl13)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl14)
			ctx.W.EmitJmp(lbl6)
			ps25 := PhiState{General: true}
			ps25.OverlayValues = make([]JITValueDesc, 23)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[6] = d6
			ps25.OverlayValues[7] = d7
			ps25.OverlayValues[8] = d8
			ps25.OverlayValues[9] = d9
			ps25.OverlayValues[10] = d10
			ps25.OverlayValues[18] = d18
			ps25.OverlayValues[19] = d19
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps26 := PhiState{General: true}
			ps26.OverlayValues = make([]JITValueDesc, 23)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[6] = d6
			ps26.OverlayValues[7] = d7
			ps26.OverlayValues[8] = d8
			ps26.OverlayValues[9] = d9
			ps26.OverlayValues[10] = d10
			ps26.OverlayValues[18] = d18
			ps26.OverlayValues[19] = d19
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[22] = d22
			alloc27 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps26)
			}
			ctx.RestoreAllocState(alloc27)
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps25)
			}
			return result
			ctx.FreeDesc(&d20)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d0, &result)
				result.Type = d0.Type
			} else {
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d0)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d0)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d0)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d0, &result)
					result.Type = d0.Type
				}
			}
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			ctx.ReclaimUntrackedRegs()
			d28 = d18
			if d28.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			if d28.Loc == LocRegPair || d28.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d28, 0)
			} else {
				ctx.EmitStoreToStack(d28, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d29 = d8
			if d29.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 16)
			ps30 := PhiState{General: ps.General}
			ps30.OverlayValues = make([]JITValueDesc, 30)
			ps30.OverlayValues[0] = d0
			ps30.OverlayValues[1] = d1
			ps30.OverlayValues[2] = d2
			ps30.OverlayValues[4] = d4
			ps30.OverlayValues[5] = d5
			ps30.OverlayValues[6] = d6
			ps30.OverlayValues[7] = d7
			ps30.OverlayValues[8] = d8
			ps30.OverlayValues[9] = d9
			ps30.OverlayValues[10] = d10
			ps30.OverlayValues[18] = d18
			ps30.OverlayValues[19] = d19
			ps30.OverlayValues[20] = d20
			ps30.OverlayValues[21] = d21
			ps30.OverlayValues[22] = d22
			ps30.OverlayValues[28] = d28
			ps30.OverlayValues[29] = d29
			ps30.PhiValues = make([]JITValueDesc, 2)
			d31 = d18
			ps30.PhiValues[0] = d31
			d32 = d8
			ps30.PhiValues[1] = d32
			if ps30.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps30)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
			}
			ctx.ReclaimUntrackedRegs()
			d34 = d18
			d34.ID = 0
			d33 = ctx.EmitTagEqualsBorrowed(&d34, tagNil, JITValueDesc{Loc: LocAny})
			d35 = d33
			ctx.EnsureDesc(&d35)
			if d35.Loc != LocImm && d35.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d35.Loc == LocImm {
				if d35.Imm.Bool() {
			d36 = d0
			if d36.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			if d36.Loc == LocRegPair || d36.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d36, 0)
			} else {
				ctx.EmitStoreToStack(d36, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d37 = d8
			if d37.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d37)
			ctx.EmitStoreToStack(d37, 16)
			ps38 := PhiState{General: ps.General}
			ps38.OverlayValues = make([]JITValueDesc, 38)
			ps38.OverlayValues[0] = d0
			ps38.OverlayValues[1] = d1
			ps38.OverlayValues[2] = d2
			ps38.OverlayValues[4] = d4
			ps38.OverlayValues[5] = d5
			ps38.OverlayValues[6] = d6
			ps38.OverlayValues[7] = d7
			ps38.OverlayValues[8] = d8
			ps38.OverlayValues[9] = d9
			ps38.OverlayValues[10] = d10
			ps38.OverlayValues[18] = d18
			ps38.OverlayValues[19] = d19
			ps38.OverlayValues[20] = d20
			ps38.OverlayValues[21] = d21
			ps38.OverlayValues[22] = d22
			ps38.OverlayValues[28] = d28
			ps38.OverlayValues[29] = d29
			ps38.OverlayValues[31] = d31
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[33] = d33
			ps38.OverlayValues[34] = d34
			ps38.OverlayValues[35] = d35
			ps38.OverlayValues[36] = d36
			ps38.OverlayValues[37] = d37
			ps38.PhiValues = make([]JITValueDesc, 2)
			d39 = d0
			ps38.PhiValues[0] = d39
			d40 = d8
			ps38.PhiValues[1] = d40
					return bbs[1].RenderPS(ps38)
				}
			ps41 := PhiState{General: ps.General}
			ps41.OverlayValues = make([]JITValueDesc, 41)
			ps41.OverlayValues[0] = d0
			ps41.OverlayValues[1] = d1
			ps41.OverlayValues[2] = d2
			ps41.OverlayValues[4] = d4
			ps41.OverlayValues[5] = d5
			ps41.OverlayValues[6] = d6
			ps41.OverlayValues[7] = d7
			ps41.OverlayValues[8] = d8
			ps41.OverlayValues[9] = d9
			ps41.OverlayValues[10] = d10
			ps41.OverlayValues[18] = d18
			ps41.OverlayValues[19] = d19
			ps41.OverlayValues[20] = d20
			ps41.OverlayValues[21] = d21
			ps41.OverlayValues[22] = d22
			ps41.OverlayValues[28] = d28
			ps41.OverlayValues[29] = d29
			ps41.OverlayValues[31] = d31
			ps41.OverlayValues[32] = d32
			ps41.OverlayValues[33] = d33
			ps41.OverlayValues[34] = d34
			ps41.OverlayValues[35] = d35
			ps41.OverlayValues[36] = d36
			ps41.OverlayValues[37] = d37
			ps41.OverlayValues[39] = d39
			ps41.OverlayValues[40] = d40
				return bbs[7].RenderPS(ps41)
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d35.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl15)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl15)
			d42 = d0
			if d42.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d42)
			if d42.Loc == LocRegPair || d42.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d42, 0)
			} else {
				ctx.EmitStoreToStack(d42, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d43 = d8
			if d43.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			ctx.EmitStoreToStack(d43, 16)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl16)
			ctx.W.EmitJmp(lbl8)
			ps44 := PhiState{General: true}
			ps44.OverlayValues = make([]JITValueDesc, 44)
			ps44.OverlayValues[0] = d0
			ps44.OverlayValues[1] = d1
			ps44.OverlayValues[2] = d2
			ps44.OverlayValues[4] = d4
			ps44.OverlayValues[5] = d5
			ps44.OverlayValues[6] = d6
			ps44.OverlayValues[7] = d7
			ps44.OverlayValues[8] = d8
			ps44.OverlayValues[9] = d9
			ps44.OverlayValues[10] = d10
			ps44.OverlayValues[18] = d18
			ps44.OverlayValues[19] = d19
			ps44.OverlayValues[20] = d20
			ps44.OverlayValues[21] = d21
			ps44.OverlayValues[22] = d22
			ps44.OverlayValues[28] = d28
			ps44.OverlayValues[29] = d29
			ps44.OverlayValues[31] = d31
			ps44.OverlayValues[32] = d32
			ps44.OverlayValues[33] = d33
			ps44.OverlayValues[34] = d34
			ps44.OverlayValues[35] = d35
			ps44.OverlayValues[36] = d36
			ps44.OverlayValues[37] = d37
			ps44.OverlayValues[39] = d39
			ps44.OverlayValues[40] = d40
			ps44.OverlayValues[42] = d42
			ps44.OverlayValues[43] = d43
			ps44.PhiValues = make([]JITValueDesc, 2)
			d46 = d0
			ps44.PhiValues[0] = d46
			d47 = d8
			ps44.PhiValues[1] = d47
			ps45 := PhiState{General: true}
			ps45.OverlayValues = make([]JITValueDesc, 48)
			ps45.OverlayValues[0] = d0
			ps45.OverlayValues[1] = d1
			ps45.OverlayValues[2] = d2
			ps45.OverlayValues[4] = d4
			ps45.OverlayValues[5] = d5
			ps45.OverlayValues[6] = d6
			ps45.OverlayValues[7] = d7
			ps45.OverlayValues[8] = d8
			ps45.OverlayValues[9] = d9
			ps45.OverlayValues[10] = d10
			ps45.OverlayValues[18] = d18
			ps45.OverlayValues[19] = d19
			ps45.OverlayValues[20] = d20
			ps45.OverlayValues[21] = d21
			ps45.OverlayValues[22] = d22
			ps45.OverlayValues[28] = d28
			ps45.OverlayValues[29] = d29
			ps45.OverlayValues[31] = d31
			ps45.OverlayValues[32] = d32
			ps45.OverlayValues[33] = d33
			ps45.OverlayValues[34] = d34
			ps45.OverlayValues[35] = d35
			ps45.OverlayValues[36] = d36
			ps45.OverlayValues[37] = d37
			ps45.OverlayValues[39] = d39
			ps45.OverlayValues[40] = d40
			ps45.OverlayValues[42] = d42
			ps45.OverlayValues[43] = d43
			ps45.OverlayValues[46] = d46
			ps45.OverlayValues[47] = d47
			snap48 := d0
			snap49 := d18
			alloc50 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps44)
			}
			ctx.RestoreAllocState(alloc50)
			d0 = snap48
			d18 = snap49
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps45)
			}
			return result
			ctx.FreeDesc(&d33)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
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
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
				d47 = ps.OverlayValues[47]
			}
			ctx.ReclaimUntrackedRegs()
			d51 = d18
			if d51.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d51)
			if d51.Loc == LocRegPair || d51.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d51, 0)
			} else {
				ctx.EmitStoreToStack(d51, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d52 = d8
			if d52.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d52)
			ctx.EmitStoreToStack(d52, 16)
			ps53 := PhiState{General: ps.General}
			ps53.OverlayValues = make([]JITValueDesc, 53)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
			ps53.OverlayValues[2] = d2
			ps53.OverlayValues[4] = d4
			ps53.OverlayValues[5] = d5
			ps53.OverlayValues[6] = d6
			ps53.OverlayValues[7] = d7
			ps53.OverlayValues[8] = d8
			ps53.OverlayValues[9] = d9
			ps53.OverlayValues[10] = d10
			ps53.OverlayValues[18] = d18
			ps53.OverlayValues[19] = d19
			ps53.OverlayValues[20] = d20
			ps53.OverlayValues[21] = d21
			ps53.OverlayValues[22] = d22
			ps53.OverlayValues[28] = d28
			ps53.OverlayValues[29] = d29
			ps53.OverlayValues[31] = d31
			ps53.OverlayValues[32] = d32
			ps53.OverlayValues[33] = d33
			ps53.OverlayValues[34] = d34
			ps53.OverlayValues[35] = d35
			ps53.OverlayValues[36] = d36
			ps53.OverlayValues[37] = d37
			ps53.OverlayValues[39] = d39
			ps53.OverlayValues[40] = d40
			ps53.OverlayValues[42] = d42
			ps53.OverlayValues[43] = d43
			ps53.OverlayValues[46] = d46
			ps53.OverlayValues[47] = d47
			ps53.OverlayValues[51] = d51
			ps53.OverlayValues[52] = d52
			ps53.PhiValues = make([]JITValueDesc, 2)
			d54 = d18
			ps53.PhiValues[0] = d54
			d55 = d8
			ps53.PhiValues[1] = d55
			if ps53.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps53)
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
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
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
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
				d47 = ps.OverlayValues[47]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			if d18.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d18.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d18.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d18)
				} else if d18.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d18)
				} else if d18.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d18)
				} else if d18.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d18.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d18 = tmpPair
			} else if d18.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d18.Type, Reg: ctx.AllocRegExcept(d18.Reg), Reg2: ctx.AllocRegExcept(d18.Reg)}
				switch d18.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d18)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d18)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d18)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d18)
				d18 = tmpPair
			}
			if d18.Loc != LocRegPair && d18.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d56 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d18, d0}, 1)
			d57 = d56
			ctx.EnsureDesc(&d57)
			if d57.Loc != LocImm && d57.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d57.Loc == LocImm {
				if d57.Imm.Bool() {
			ps58 := PhiState{General: ps.General}
			ps58.OverlayValues = make([]JITValueDesc, 58)
			ps58.OverlayValues[0] = d0
			ps58.OverlayValues[1] = d1
			ps58.OverlayValues[2] = d2
			ps58.OverlayValues[4] = d4
			ps58.OverlayValues[5] = d5
			ps58.OverlayValues[6] = d6
			ps58.OverlayValues[7] = d7
			ps58.OverlayValues[8] = d8
			ps58.OverlayValues[9] = d9
			ps58.OverlayValues[10] = d10
			ps58.OverlayValues[18] = d18
			ps58.OverlayValues[19] = d19
			ps58.OverlayValues[20] = d20
			ps58.OverlayValues[21] = d21
			ps58.OverlayValues[22] = d22
			ps58.OverlayValues[28] = d28
			ps58.OverlayValues[29] = d29
			ps58.OverlayValues[31] = d31
			ps58.OverlayValues[32] = d32
			ps58.OverlayValues[33] = d33
			ps58.OverlayValues[34] = d34
			ps58.OverlayValues[35] = d35
			ps58.OverlayValues[36] = d36
			ps58.OverlayValues[37] = d37
			ps58.OverlayValues[39] = d39
			ps58.OverlayValues[40] = d40
			ps58.OverlayValues[42] = d42
			ps58.OverlayValues[43] = d43
			ps58.OverlayValues[46] = d46
			ps58.OverlayValues[47] = d47
			ps58.OverlayValues[51] = d51
			ps58.OverlayValues[52] = d52
			ps58.OverlayValues[54] = d54
			ps58.OverlayValues[55] = d55
			ps58.OverlayValues[56] = d56
			ps58.OverlayValues[57] = d57
					return bbs[6].RenderPS(ps58)
				}
			d59 = d0
			if d59.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			if d59.Loc == LocRegPair || d59.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d59, 0)
			} else {
				ctx.EmitStoreToStack(d59, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d60 = d8
			if d60.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d60)
			ctx.EmitStoreToStack(d60, 16)
			ps61 := PhiState{General: ps.General}
			ps61.OverlayValues = make([]JITValueDesc, 61)
			ps61.OverlayValues[0] = d0
			ps61.OverlayValues[1] = d1
			ps61.OverlayValues[2] = d2
			ps61.OverlayValues[4] = d4
			ps61.OverlayValues[5] = d5
			ps61.OverlayValues[6] = d6
			ps61.OverlayValues[7] = d7
			ps61.OverlayValues[8] = d8
			ps61.OverlayValues[9] = d9
			ps61.OverlayValues[10] = d10
			ps61.OverlayValues[18] = d18
			ps61.OverlayValues[19] = d19
			ps61.OverlayValues[20] = d20
			ps61.OverlayValues[21] = d21
			ps61.OverlayValues[22] = d22
			ps61.OverlayValues[28] = d28
			ps61.OverlayValues[29] = d29
			ps61.OverlayValues[31] = d31
			ps61.OverlayValues[32] = d32
			ps61.OverlayValues[33] = d33
			ps61.OverlayValues[34] = d34
			ps61.OverlayValues[35] = d35
			ps61.OverlayValues[36] = d36
			ps61.OverlayValues[37] = d37
			ps61.OverlayValues[39] = d39
			ps61.OverlayValues[40] = d40
			ps61.OverlayValues[42] = d42
			ps61.OverlayValues[43] = d43
			ps61.OverlayValues[46] = d46
			ps61.OverlayValues[47] = d47
			ps61.OverlayValues[51] = d51
			ps61.OverlayValues[52] = d52
			ps61.OverlayValues[54] = d54
			ps61.OverlayValues[55] = d55
			ps61.OverlayValues[56] = d56
			ps61.OverlayValues[57] = d57
			ps61.OverlayValues[59] = d59
			ps61.OverlayValues[60] = d60
			ps61.PhiValues = make([]JITValueDesc, 2)
			d62 = d0
			ps61.PhiValues[0] = d62
			d63 = d8
			ps61.PhiValues[1] = d63
				return bbs[1].RenderPS(ps61)
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d57.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl17)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl17)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl18)
			d64 = d0
			if d64.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d64)
			if d64.Loc == LocRegPair || d64.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d64, 0)
			} else {
				ctx.EmitStoreToStack(d64, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d65 = d8
			if d65.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d65)
			ctx.EmitStoreToStack(d65, 16)
			ctx.W.EmitJmp(lbl2)
			ps66 := PhiState{General: true}
			ps66.OverlayValues = make([]JITValueDesc, 66)
			ps66.OverlayValues[0] = d0
			ps66.OverlayValues[1] = d1
			ps66.OverlayValues[2] = d2
			ps66.OverlayValues[4] = d4
			ps66.OverlayValues[5] = d5
			ps66.OverlayValues[6] = d6
			ps66.OverlayValues[7] = d7
			ps66.OverlayValues[8] = d8
			ps66.OverlayValues[9] = d9
			ps66.OverlayValues[10] = d10
			ps66.OverlayValues[18] = d18
			ps66.OverlayValues[19] = d19
			ps66.OverlayValues[20] = d20
			ps66.OverlayValues[21] = d21
			ps66.OverlayValues[22] = d22
			ps66.OverlayValues[28] = d28
			ps66.OverlayValues[29] = d29
			ps66.OverlayValues[31] = d31
			ps66.OverlayValues[32] = d32
			ps66.OverlayValues[33] = d33
			ps66.OverlayValues[34] = d34
			ps66.OverlayValues[35] = d35
			ps66.OverlayValues[36] = d36
			ps66.OverlayValues[37] = d37
			ps66.OverlayValues[39] = d39
			ps66.OverlayValues[40] = d40
			ps66.OverlayValues[42] = d42
			ps66.OverlayValues[43] = d43
			ps66.OverlayValues[46] = d46
			ps66.OverlayValues[47] = d47
			ps66.OverlayValues[51] = d51
			ps66.OverlayValues[52] = d52
			ps66.OverlayValues[54] = d54
			ps66.OverlayValues[55] = d55
			ps66.OverlayValues[56] = d56
			ps66.OverlayValues[57] = d57
			ps66.OverlayValues[59] = d59
			ps66.OverlayValues[60] = d60
			ps66.OverlayValues[62] = d62
			ps66.OverlayValues[63] = d63
			ps66.OverlayValues[64] = d64
			ps66.OverlayValues[65] = d65
			ps67 := PhiState{General: true}
			ps67.OverlayValues = make([]JITValueDesc, 66)
			ps67.OverlayValues[0] = d0
			ps67.OverlayValues[1] = d1
			ps67.OverlayValues[2] = d2
			ps67.OverlayValues[4] = d4
			ps67.OverlayValues[5] = d5
			ps67.OverlayValues[6] = d6
			ps67.OverlayValues[7] = d7
			ps67.OverlayValues[8] = d8
			ps67.OverlayValues[9] = d9
			ps67.OverlayValues[10] = d10
			ps67.OverlayValues[18] = d18
			ps67.OverlayValues[19] = d19
			ps67.OverlayValues[20] = d20
			ps67.OverlayValues[21] = d21
			ps67.OverlayValues[22] = d22
			ps67.OverlayValues[28] = d28
			ps67.OverlayValues[29] = d29
			ps67.OverlayValues[31] = d31
			ps67.OverlayValues[32] = d32
			ps67.OverlayValues[33] = d33
			ps67.OverlayValues[34] = d34
			ps67.OverlayValues[35] = d35
			ps67.OverlayValues[36] = d36
			ps67.OverlayValues[37] = d37
			ps67.OverlayValues[39] = d39
			ps67.OverlayValues[40] = d40
			ps67.OverlayValues[42] = d42
			ps67.OverlayValues[43] = d43
			ps67.OverlayValues[46] = d46
			ps67.OverlayValues[47] = d47
			ps67.OverlayValues[51] = d51
			ps67.OverlayValues[52] = d52
			ps67.OverlayValues[54] = d54
			ps67.OverlayValues[55] = d55
			ps67.OverlayValues[56] = d56
			ps67.OverlayValues[57] = d57
			ps67.OverlayValues[59] = d59
			ps67.OverlayValues[60] = d60
			ps67.OverlayValues[62] = d62
			ps67.OverlayValues[63] = d63
			ps67.OverlayValues[64] = d64
			ps67.OverlayValues[65] = d65
			ps67.PhiValues = make([]JITValueDesc, 2)
			d68 = d0
			ps67.PhiValues[0] = d68
			d69 = d8
			ps67.PhiValues[1] = d69
			alloc70 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps67)
			}
			ctx.RestoreAllocState(alloc70)
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps66)
			}
			return result
			ctx.FreeDesc(&d56)
			return result
			}
			ps71 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps71)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(24))
			ctx.W.EmitAddRSP32(int32(24))
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
	})
	Declare(&Globalenv, &Declaration{
		"max", "returns the highest value",
		1, 1000,
		[]DeclarationParameter{
			DeclarationParameter{"value...", "number|string", "value", nil},
		}, "number|string",
		func(a ...Scmer) Scmer {
			var result Scmer
			for _, v := range a {
				if result.IsNil() {
					result = v
				} else if !v.IsNil() && Less(result, v) {
					result = v
				}
			}
			return result
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d2 JITValueDesc
			_ = d2
			var d4 JITValueDesc
			_ = d4
			var d5 JITValueDesc
			_ = d5
			var d6 JITValueDesc
			_ = d6
			var d7 JITValueDesc
			_ = d7
			var d8 JITValueDesc
			_ = d8
			var d9 JITValueDesc
			_ = d9
			var d10 JITValueDesc
			_ = d10
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
			var d28 JITValueDesc
			_ = d28
			var d29 JITValueDesc
			_ = d29
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
			var d36 JITValueDesc
			_ = d36
			var d37 JITValueDesc
			_ = d37
			var d39 JITValueDesc
			_ = d39
			var d40 JITValueDesc
			_ = d40
			var d42 JITValueDesc
			_ = d42
			var d43 JITValueDesc
			_ = d43
			var d46 JITValueDesc
			_ = d46
			var d47 JITValueDesc
			_ = d47
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
			var d62 JITValueDesc
			_ = d62
			var d63 JITValueDesc
			_ = d63
			var d64 JITValueDesc
			_ = d64
			var d65 JITValueDesc
			_ = d65
			var d68 JITValueDesc
			_ = d68
			var d69 JITValueDesc
			_ = d69
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			var bbs [8]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, 0)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(-1)}, 16)
			ps3 := PhiState{General: ps.General}
			ps3.OverlayValues = make([]JITValueDesc, 3)
			ps3.OverlayValues[0] = d0
			ps3.OverlayValues[1] = d1
			ps3.OverlayValues[2] = d2
			ps3.PhiValues = make([]JITValueDesc, 2)
			d4 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
			ps3.PhiValues[0] = d4
			d5 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
			ps3.PhiValues[1] = d5
			if ps3.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps3)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d6 := ps.PhiValues[0]
						ctx.EnsureDesc(&d6)
						ctx.EmitStoreScmerToStack(d6, 0)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
						d7 := ps.PhiValues[1]
						ctx.EnsureDesc(&d7)
						ctx.EmitStoreToStack(d7, 16)
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d1 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d8 JITValueDesc
			if d1.Loc == LocImm {
				d8 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d8 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			}
			if d8.Loc == LocReg && d1.Loc == LocReg && d8.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d2)
			var d9 JITValueDesc
			if d8.Loc == LocImm && d2.Loc == LocImm {
				d9 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d8.Imm.Int() < d2.Imm.Int())}
			} else if d2.Loc == LocImm {
				r1 := ctx.AllocRegExcept(d8.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d8.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d8.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r1, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d9)
			} else if d8.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d2.Reg)
				ctx.W.EmitSetcc(r2, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d9)
			} else {
				r3 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitCmpInt64(d8.Reg, d2.Reg)
				ctx.W.EmitSetcc(r3, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d9)
			}
			ctx.FreeDesc(&d2)
			d10 = d9
			ctx.EnsureDesc(&d10)
			if d10.Loc != LocImm && d10.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d10.Loc == LocImm {
				if d10.Imm.Bool() {
			ps11 := PhiState{General: ps.General}
			ps11.OverlayValues = make([]JITValueDesc, 11)
			ps11.OverlayValues[0] = d0
			ps11.OverlayValues[1] = d1
			ps11.OverlayValues[2] = d2
			ps11.OverlayValues[4] = d4
			ps11.OverlayValues[5] = d5
			ps11.OverlayValues[6] = d6
			ps11.OverlayValues[7] = d7
			ps11.OverlayValues[8] = d8
			ps11.OverlayValues[9] = d9
			ps11.OverlayValues[10] = d10
					return bbs[2].RenderPS(ps11)
				}
			ps12 := PhiState{General: ps.General}
			ps12.OverlayValues = make([]JITValueDesc, 11)
			ps12.OverlayValues[0] = d0
			ps12.OverlayValues[1] = d1
			ps12.OverlayValues[2] = d2
			ps12.OverlayValues[4] = d4
			ps12.OverlayValues[5] = d5
			ps12.OverlayValues[6] = d6
			ps12.OverlayValues[7] = d7
			ps12.OverlayValues[8] = d8
			ps12.OverlayValues[9] = d9
			ps12.OverlayValues[10] = d10
				return bbs[3].RenderPS(ps12)
			}
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d10.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl9)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl9)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitJmp(lbl4)
			ps13 := PhiState{General: true}
			ps13.OverlayValues = make([]JITValueDesc, 11)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[4] = d4
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[6] = d6
			ps13.OverlayValues[7] = d7
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps14 := PhiState{General: true}
			ps14.OverlayValues = make([]JITValueDesc, 11)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[6] = d6
			ps14.OverlayValues[7] = d7
			ps14.OverlayValues[8] = d8
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			snap15 := d0
			snap16 := d8
			alloc17 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps14)
			}
			ctx.RestoreAllocState(alloc17)
			d0 = snap15
			d8 = snap16
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps13)
			}
			return result
			ctx.FreeDesc(&d9)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			var d18 JITValueDesc
			if d8.Loc == LocImm {
				idx := int(d8.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d18 = args[idx]
				d18.ID = 0
			} else {
				protected := make([]Reg, 0, len(args)*2+1)
				seen := make(map[Reg]bool)
				if !seen[d8.Reg] {
					ctx.ProtectReg(d8.Reg)
					seen[d8.Reg] = true
					protected = append(protected, d8.Reg)
				}
				for _, ai := range args {
					if ai.Loc == LocReg {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
					} else if ai.Loc == LocRegPair {
						if !seen[ai.Reg] {
							ctx.ProtectReg(ai.Reg)
							seen[ai.Reg] = true
							protected = append(protected, ai.Reg)
						}
						if !seen[ai.Reg2] {
							ctx.ProtectReg(ai.Reg2)
							seen[ai.Reg2] = true
							protected = append(protected, ai.Reg2)
						}
					} else if ai.Loc == LocStackPair {
						// no direct registers to protect
					}
				}
				r4 := ctx.AllocReg()
				r5 := ctx.AllocRegExcept(r4)
				lbl11 := ctx.W.ReserveLabel()
				lbl12 := ctx.W.ReserveLabel()
				ctx.W.EmitCmpRegImm32(d8.Reg, int32(len(args)))
				ctx.W.EmitJcc(CcAE, lbl12)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.W.ReserveLabel()
					ctx.W.EmitCmpRegImm32(d8.Reg, int32(i))
					ctx.W.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.W.EmitMovRegReg(r4, ai.Reg)
						ctx.W.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.W.EmitMovRegReg(r4, tmp.Reg)
						ctx.W.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.W.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.W.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.W.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.W.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.W.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.W.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.W.EmitJmp(lbl11)
					ctx.W.MarkLabel(nextLbl)
				}
				ctx.W.MarkLabel(lbl12)
				d19 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d19)
				ctx.BindReg(r5, &d19)
				ctx.BindReg(r4, &d19)
				ctx.BindReg(r5, &d19)
				ctx.W.EmitMakeNil(d19)
				ctx.W.MarkLabel(lbl11)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d18 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d18)
				ctx.BindReg(r5, &d18)
			}
			d21 = d0
			d21.ID = 0
			d20 = ctx.EmitTagEqualsBorrowed(&d21, tagNil, JITValueDesc{Loc: LocAny})
			d22 = d20
			ctx.EnsureDesc(&d22)
			if d22.Loc != LocImm && d22.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d22.Loc == LocImm {
				if d22.Imm.Bool() {
			ps23 := PhiState{General: ps.General}
			ps23.OverlayValues = make([]JITValueDesc, 23)
			ps23.OverlayValues[0] = d0
			ps23.OverlayValues[1] = d1
			ps23.OverlayValues[2] = d2
			ps23.OverlayValues[4] = d4
			ps23.OverlayValues[5] = d5
			ps23.OverlayValues[6] = d6
			ps23.OverlayValues[7] = d7
			ps23.OverlayValues[8] = d8
			ps23.OverlayValues[9] = d9
			ps23.OverlayValues[10] = d10
			ps23.OverlayValues[18] = d18
			ps23.OverlayValues[19] = d19
			ps23.OverlayValues[20] = d20
			ps23.OverlayValues[21] = d21
			ps23.OverlayValues[22] = d22
					return bbs[4].RenderPS(ps23)
				}
			ps24 := PhiState{General: ps.General}
			ps24.OverlayValues = make([]JITValueDesc, 23)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[4] = d4
			ps24.OverlayValues[5] = d5
			ps24.OverlayValues[6] = d6
			ps24.OverlayValues[7] = d7
			ps24.OverlayValues[8] = d8
			ps24.OverlayValues[9] = d9
			ps24.OverlayValues[10] = d10
			ps24.OverlayValues[18] = d18
			ps24.OverlayValues[19] = d19
			ps24.OverlayValues[20] = d20
			ps24.OverlayValues[21] = d21
			ps24.OverlayValues[22] = d22
				return bbs[5].RenderPS(ps24)
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d22.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl13)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl14)
			ctx.W.EmitJmp(lbl6)
			ps25 := PhiState{General: true}
			ps25.OverlayValues = make([]JITValueDesc, 23)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[6] = d6
			ps25.OverlayValues[7] = d7
			ps25.OverlayValues[8] = d8
			ps25.OverlayValues[9] = d9
			ps25.OverlayValues[10] = d10
			ps25.OverlayValues[18] = d18
			ps25.OverlayValues[19] = d19
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps26 := PhiState{General: true}
			ps26.OverlayValues = make([]JITValueDesc, 23)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[5] = d5
			ps26.OverlayValues[6] = d6
			ps26.OverlayValues[7] = d7
			ps26.OverlayValues[8] = d8
			ps26.OverlayValues[9] = d9
			ps26.OverlayValues[10] = d10
			ps26.OverlayValues[18] = d18
			ps26.OverlayValues[19] = d19
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[22] = d22
			alloc27 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps26)
			}
			ctx.RestoreAllocState(alloc27)
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps25)
			}
			return result
			ctx.FreeDesc(&d20)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d0, &result)
				result.Type = d0.Type
			} else {
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(result, d0)
					result.Type = tagBool
				case tagInt:
					ctx.W.EmitMakeInt(result, d0)
					result.Type = tagInt
				case tagFloat:
					ctx.W.EmitMakeFloat(result, d0)
					result.Type = tagFloat
				case tagNil:
					ctx.W.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d0, &result)
					result.Type = d0.Type
				}
			}
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			ctx.ReclaimUntrackedRegs()
			d28 = d18
			if d28.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			if d28.Loc == LocRegPair || d28.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d28, 0)
			} else {
				ctx.EmitStoreToStack(d28, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d29 = d8
			if d29.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 16)
			ps30 := PhiState{General: ps.General}
			ps30.OverlayValues = make([]JITValueDesc, 30)
			ps30.OverlayValues[0] = d0
			ps30.OverlayValues[1] = d1
			ps30.OverlayValues[2] = d2
			ps30.OverlayValues[4] = d4
			ps30.OverlayValues[5] = d5
			ps30.OverlayValues[6] = d6
			ps30.OverlayValues[7] = d7
			ps30.OverlayValues[8] = d8
			ps30.OverlayValues[9] = d9
			ps30.OverlayValues[10] = d10
			ps30.OverlayValues[18] = d18
			ps30.OverlayValues[19] = d19
			ps30.OverlayValues[20] = d20
			ps30.OverlayValues[21] = d21
			ps30.OverlayValues[22] = d22
			ps30.OverlayValues[28] = d28
			ps30.OverlayValues[29] = d29
			ps30.PhiValues = make([]JITValueDesc, 2)
			d31 = d18
			ps30.PhiValues[0] = d31
			d32 = d8
			ps30.PhiValues[1] = d32
			if ps30.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps30)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
			}
			ctx.ReclaimUntrackedRegs()
			d34 = d18
			d34.ID = 0
			d33 = ctx.EmitTagEqualsBorrowed(&d34, tagNil, JITValueDesc{Loc: LocAny})
			d35 = d33
			ctx.EnsureDesc(&d35)
			if d35.Loc != LocImm && d35.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d35.Loc == LocImm {
				if d35.Imm.Bool() {
			d36 = d0
			if d36.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			if d36.Loc == LocRegPair || d36.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d36, 0)
			} else {
				ctx.EmitStoreToStack(d36, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d37 = d8
			if d37.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d37)
			ctx.EmitStoreToStack(d37, 16)
			ps38 := PhiState{General: ps.General}
			ps38.OverlayValues = make([]JITValueDesc, 38)
			ps38.OverlayValues[0] = d0
			ps38.OverlayValues[1] = d1
			ps38.OverlayValues[2] = d2
			ps38.OverlayValues[4] = d4
			ps38.OverlayValues[5] = d5
			ps38.OverlayValues[6] = d6
			ps38.OverlayValues[7] = d7
			ps38.OverlayValues[8] = d8
			ps38.OverlayValues[9] = d9
			ps38.OverlayValues[10] = d10
			ps38.OverlayValues[18] = d18
			ps38.OverlayValues[19] = d19
			ps38.OverlayValues[20] = d20
			ps38.OverlayValues[21] = d21
			ps38.OverlayValues[22] = d22
			ps38.OverlayValues[28] = d28
			ps38.OverlayValues[29] = d29
			ps38.OverlayValues[31] = d31
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[33] = d33
			ps38.OverlayValues[34] = d34
			ps38.OverlayValues[35] = d35
			ps38.OverlayValues[36] = d36
			ps38.OverlayValues[37] = d37
			ps38.PhiValues = make([]JITValueDesc, 2)
			d39 = d0
			ps38.PhiValues[0] = d39
			d40 = d8
			ps38.PhiValues[1] = d40
					return bbs[1].RenderPS(ps38)
				}
			ps41 := PhiState{General: ps.General}
			ps41.OverlayValues = make([]JITValueDesc, 41)
			ps41.OverlayValues[0] = d0
			ps41.OverlayValues[1] = d1
			ps41.OverlayValues[2] = d2
			ps41.OverlayValues[4] = d4
			ps41.OverlayValues[5] = d5
			ps41.OverlayValues[6] = d6
			ps41.OverlayValues[7] = d7
			ps41.OverlayValues[8] = d8
			ps41.OverlayValues[9] = d9
			ps41.OverlayValues[10] = d10
			ps41.OverlayValues[18] = d18
			ps41.OverlayValues[19] = d19
			ps41.OverlayValues[20] = d20
			ps41.OverlayValues[21] = d21
			ps41.OverlayValues[22] = d22
			ps41.OverlayValues[28] = d28
			ps41.OverlayValues[29] = d29
			ps41.OverlayValues[31] = d31
			ps41.OverlayValues[32] = d32
			ps41.OverlayValues[33] = d33
			ps41.OverlayValues[34] = d34
			ps41.OverlayValues[35] = d35
			ps41.OverlayValues[36] = d36
			ps41.OverlayValues[37] = d37
			ps41.OverlayValues[39] = d39
			ps41.OverlayValues[40] = d40
				return bbs[7].RenderPS(ps41)
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d35.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl15)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl15)
			d42 = d0
			if d42.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d42)
			if d42.Loc == LocRegPair || d42.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d42, 0)
			} else {
				ctx.EmitStoreToStack(d42, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d43 = d8
			if d43.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			ctx.EmitStoreToStack(d43, 16)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl16)
			ctx.W.EmitJmp(lbl8)
			ps44 := PhiState{General: true}
			ps44.OverlayValues = make([]JITValueDesc, 44)
			ps44.OverlayValues[0] = d0
			ps44.OverlayValues[1] = d1
			ps44.OverlayValues[2] = d2
			ps44.OverlayValues[4] = d4
			ps44.OverlayValues[5] = d5
			ps44.OverlayValues[6] = d6
			ps44.OverlayValues[7] = d7
			ps44.OverlayValues[8] = d8
			ps44.OverlayValues[9] = d9
			ps44.OverlayValues[10] = d10
			ps44.OverlayValues[18] = d18
			ps44.OverlayValues[19] = d19
			ps44.OverlayValues[20] = d20
			ps44.OverlayValues[21] = d21
			ps44.OverlayValues[22] = d22
			ps44.OverlayValues[28] = d28
			ps44.OverlayValues[29] = d29
			ps44.OverlayValues[31] = d31
			ps44.OverlayValues[32] = d32
			ps44.OverlayValues[33] = d33
			ps44.OverlayValues[34] = d34
			ps44.OverlayValues[35] = d35
			ps44.OverlayValues[36] = d36
			ps44.OverlayValues[37] = d37
			ps44.OverlayValues[39] = d39
			ps44.OverlayValues[40] = d40
			ps44.OverlayValues[42] = d42
			ps44.OverlayValues[43] = d43
			ps44.PhiValues = make([]JITValueDesc, 2)
			d46 = d0
			ps44.PhiValues[0] = d46
			d47 = d8
			ps44.PhiValues[1] = d47
			ps45 := PhiState{General: true}
			ps45.OverlayValues = make([]JITValueDesc, 48)
			ps45.OverlayValues[0] = d0
			ps45.OverlayValues[1] = d1
			ps45.OverlayValues[2] = d2
			ps45.OverlayValues[4] = d4
			ps45.OverlayValues[5] = d5
			ps45.OverlayValues[6] = d6
			ps45.OverlayValues[7] = d7
			ps45.OverlayValues[8] = d8
			ps45.OverlayValues[9] = d9
			ps45.OverlayValues[10] = d10
			ps45.OverlayValues[18] = d18
			ps45.OverlayValues[19] = d19
			ps45.OverlayValues[20] = d20
			ps45.OverlayValues[21] = d21
			ps45.OverlayValues[22] = d22
			ps45.OverlayValues[28] = d28
			ps45.OverlayValues[29] = d29
			ps45.OverlayValues[31] = d31
			ps45.OverlayValues[32] = d32
			ps45.OverlayValues[33] = d33
			ps45.OverlayValues[34] = d34
			ps45.OverlayValues[35] = d35
			ps45.OverlayValues[36] = d36
			ps45.OverlayValues[37] = d37
			ps45.OverlayValues[39] = d39
			ps45.OverlayValues[40] = d40
			ps45.OverlayValues[42] = d42
			ps45.OverlayValues[43] = d43
			ps45.OverlayValues[46] = d46
			ps45.OverlayValues[47] = d47
			snap48 := d0
			snap49 := d18
			alloc50 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps44)
			}
			ctx.RestoreAllocState(alloc50)
			d0 = snap48
			d18 = snap49
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps45)
			}
			return result
			ctx.FreeDesc(&d33)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
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
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
				d47 = ps.OverlayValues[47]
			}
			ctx.ReclaimUntrackedRegs()
			d51 = d18
			if d51.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d51)
			if d51.Loc == LocRegPair || d51.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d51, 0)
			} else {
				ctx.EmitStoreToStack(d51, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d52 = d8
			if d52.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d52)
			ctx.EmitStoreToStack(d52, 16)
			ps53 := PhiState{General: ps.General}
			ps53.OverlayValues = make([]JITValueDesc, 53)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
			ps53.OverlayValues[2] = d2
			ps53.OverlayValues[4] = d4
			ps53.OverlayValues[5] = d5
			ps53.OverlayValues[6] = d6
			ps53.OverlayValues[7] = d7
			ps53.OverlayValues[8] = d8
			ps53.OverlayValues[9] = d9
			ps53.OverlayValues[10] = d10
			ps53.OverlayValues[18] = d18
			ps53.OverlayValues[19] = d19
			ps53.OverlayValues[20] = d20
			ps53.OverlayValues[21] = d21
			ps53.OverlayValues[22] = d22
			ps53.OverlayValues[28] = d28
			ps53.OverlayValues[29] = d29
			ps53.OverlayValues[31] = d31
			ps53.OverlayValues[32] = d32
			ps53.OverlayValues[33] = d33
			ps53.OverlayValues[34] = d34
			ps53.OverlayValues[35] = d35
			ps53.OverlayValues[36] = d36
			ps53.OverlayValues[37] = d37
			ps53.OverlayValues[39] = d39
			ps53.OverlayValues[40] = d40
			ps53.OverlayValues[42] = d42
			ps53.OverlayValues[43] = d43
			ps53.OverlayValues[46] = d46
			ps53.OverlayValues[47] = d47
			ps53.OverlayValues[51] = d51
			ps53.OverlayValues[52] = d52
			ps53.PhiValues = make([]JITValueDesc, 2)
			d54 = d18
			ps53.PhiValues[0] = d54
			d55 = d8
			ps53.PhiValues[1] = d55
			if ps53.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps53)
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
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
				d2 = ps.OverlayValues[2]
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
				d10 = ps.OverlayValues[10]
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
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
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != LocNone {
				d47 = ps.OverlayValues[47]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			if d18.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d18.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d18.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d18)
				} else if d18.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d18)
				} else if d18.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d18)
				} else if d18.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d18.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d18 = tmpPair
			} else if d18.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d18.Type, Reg: ctx.AllocRegExcept(d18.Reg), Reg2: ctx.AllocRegExcept(d18.Reg)}
				switch d18.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d18)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d18)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d18)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d18)
				d18 = tmpPair
			}
			if d18.Loc != LocRegPair && d18.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d56 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d18}, 1)
			d57 = d56
			ctx.EnsureDesc(&d57)
			if d57.Loc != LocImm && d57.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d57.Loc == LocImm {
				if d57.Imm.Bool() {
			ps58 := PhiState{General: ps.General}
			ps58.OverlayValues = make([]JITValueDesc, 58)
			ps58.OverlayValues[0] = d0
			ps58.OverlayValues[1] = d1
			ps58.OverlayValues[2] = d2
			ps58.OverlayValues[4] = d4
			ps58.OverlayValues[5] = d5
			ps58.OverlayValues[6] = d6
			ps58.OverlayValues[7] = d7
			ps58.OverlayValues[8] = d8
			ps58.OverlayValues[9] = d9
			ps58.OverlayValues[10] = d10
			ps58.OverlayValues[18] = d18
			ps58.OverlayValues[19] = d19
			ps58.OverlayValues[20] = d20
			ps58.OverlayValues[21] = d21
			ps58.OverlayValues[22] = d22
			ps58.OverlayValues[28] = d28
			ps58.OverlayValues[29] = d29
			ps58.OverlayValues[31] = d31
			ps58.OverlayValues[32] = d32
			ps58.OverlayValues[33] = d33
			ps58.OverlayValues[34] = d34
			ps58.OverlayValues[35] = d35
			ps58.OverlayValues[36] = d36
			ps58.OverlayValues[37] = d37
			ps58.OverlayValues[39] = d39
			ps58.OverlayValues[40] = d40
			ps58.OverlayValues[42] = d42
			ps58.OverlayValues[43] = d43
			ps58.OverlayValues[46] = d46
			ps58.OverlayValues[47] = d47
			ps58.OverlayValues[51] = d51
			ps58.OverlayValues[52] = d52
			ps58.OverlayValues[54] = d54
			ps58.OverlayValues[55] = d55
			ps58.OverlayValues[56] = d56
			ps58.OverlayValues[57] = d57
					return bbs[6].RenderPS(ps58)
				}
			d59 = d0
			if d59.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			if d59.Loc == LocRegPair || d59.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d59, 0)
			} else {
				ctx.EmitStoreToStack(d59, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d60 = d8
			if d60.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d60)
			ctx.EmitStoreToStack(d60, 16)
			ps61 := PhiState{General: ps.General}
			ps61.OverlayValues = make([]JITValueDesc, 61)
			ps61.OverlayValues[0] = d0
			ps61.OverlayValues[1] = d1
			ps61.OverlayValues[2] = d2
			ps61.OverlayValues[4] = d4
			ps61.OverlayValues[5] = d5
			ps61.OverlayValues[6] = d6
			ps61.OverlayValues[7] = d7
			ps61.OverlayValues[8] = d8
			ps61.OverlayValues[9] = d9
			ps61.OverlayValues[10] = d10
			ps61.OverlayValues[18] = d18
			ps61.OverlayValues[19] = d19
			ps61.OverlayValues[20] = d20
			ps61.OverlayValues[21] = d21
			ps61.OverlayValues[22] = d22
			ps61.OverlayValues[28] = d28
			ps61.OverlayValues[29] = d29
			ps61.OverlayValues[31] = d31
			ps61.OverlayValues[32] = d32
			ps61.OverlayValues[33] = d33
			ps61.OverlayValues[34] = d34
			ps61.OverlayValues[35] = d35
			ps61.OverlayValues[36] = d36
			ps61.OverlayValues[37] = d37
			ps61.OverlayValues[39] = d39
			ps61.OverlayValues[40] = d40
			ps61.OverlayValues[42] = d42
			ps61.OverlayValues[43] = d43
			ps61.OverlayValues[46] = d46
			ps61.OverlayValues[47] = d47
			ps61.OverlayValues[51] = d51
			ps61.OverlayValues[52] = d52
			ps61.OverlayValues[54] = d54
			ps61.OverlayValues[55] = d55
			ps61.OverlayValues[56] = d56
			ps61.OverlayValues[57] = d57
			ps61.OverlayValues[59] = d59
			ps61.OverlayValues[60] = d60
			ps61.PhiValues = make([]JITValueDesc, 2)
			d62 = d0
			ps61.PhiValues[0] = d62
			d63 = d8
			ps61.PhiValues[1] = d63
				return bbs[1].RenderPS(ps61)
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d57.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl17)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl17)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl18)
			d64 = d0
			if d64.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d64)
			if d64.Loc == LocRegPair || d64.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d64, 0)
			} else {
				ctx.EmitStoreToStack(d64, 0)
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (0)+8)
			}
			d65 = d8
			if d65.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d65)
			ctx.EmitStoreToStack(d65, 16)
			ctx.W.EmitJmp(lbl2)
			ps66 := PhiState{General: true}
			ps66.OverlayValues = make([]JITValueDesc, 66)
			ps66.OverlayValues[0] = d0
			ps66.OverlayValues[1] = d1
			ps66.OverlayValues[2] = d2
			ps66.OverlayValues[4] = d4
			ps66.OverlayValues[5] = d5
			ps66.OverlayValues[6] = d6
			ps66.OverlayValues[7] = d7
			ps66.OverlayValues[8] = d8
			ps66.OverlayValues[9] = d9
			ps66.OverlayValues[10] = d10
			ps66.OverlayValues[18] = d18
			ps66.OverlayValues[19] = d19
			ps66.OverlayValues[20] = d20
			ps66.OverlayValues[21] = d21
			ps66.OverlayValues[22] = d22
			ps66.OverlayValues[28] = d28
			ps66.OverlayValues[29] = d29
			ps66.OverlayValues[31] = d31
			ps66.OverlayValues[32] = d32
			ps66.OverlayValues[33] = d33
			ps66.OverlayValues[34] = d34
			ps66.OverlayValues[35] = d35
			ps66.OverlayValues[36] = d36
			ps66.OverlayValues[37] = d37
			ps66.OverlayValues[39] = d39
			ps66.OverlayValues[40] = d40
			ps66.OverlayValues[42] = d42
			ps66.OverlayValues[43] = d43
			ps66.OverlayValues[46] = d46
			ps66.OverlayValues[47] = d47
			ps66.OverlayValues[51] = d51
			ps66.OverlayValues[52] = d52
			ps66.OverlayValues[54] = d54
			ps66.OverlayValues[55] = d55
			ps66.OverlayValues[56] = d56
			ps66.OverlayValues[57] = d57
			ps66.OverlayValues[59] = d59
			ps66.OverlayValues[60] = d60
			ps66.OverlayValues[62] = d62
			ps66.OverlayValues[63] = d63
			ps66.OverlayValues[64] = d64
			ps66.OverlayValues[65] = d65
			ps67 := PhiState{General: true}
			ps67.OverlayValues = make([]JITValueDesc, 66)
			ps67.OverlayValues[0] = d0
			ps67.OverlayValues[1] = d1
			ps67.OverlayValues[2] = d2
			ps67.OverlayValues[4] = d4
			ps67.OverlayValues[5] = d5
			ps67.OverlayValues[6] = d6
			ps67.OverlayValues[7] = d7
			ps67.OverlayValues[8] = d8
			ps67.OverlayValues[9] = d9
			ps67.OverlayValues[10] = d10
			ps67.OverlayValues[18] = d18
			ps67.OverlayValues[19] = d19
			ps67.OverlayValues[20] = d20
			ps67.OverlayValues[21] = d21
			ps67.OverlayValues[22] = d22
			ps67.OverlayValues[28] = d28
			ps67.OverlayValues[29] = d29
			ps67.OverlayValues[31] = d31
			ps67.OverlayValues[32] = d32
			ps67.OverlayValues[33] = d33
			ps67.OverlayValues[34] = d34
			ps67.OverlayValues[35] = d35
			ps67.OverlayValues[36] = d36
			ps67.OverlayValues[37] = d37
			ps67.OverlayValues[39] = d39
			ps67.OverlayValues[40] = d40
			ps67.OverlayValues[42] = d42
			ps67.OverlayValues[43] = d43
			ps67.OverlayValues[46] = d46
			ps67.OverlayValues[47] = d47
			ps67.OverlayValues[51] = d51
			ps67.OverlayValues[52] = d52
			ps67.OverlayValues[54] = d54
			ps67.OverlayValues[55] = d55
			ps67.OverlayValues[56] = d56
			ps67.OverlayValues[57] = d57
			ps67.OverlayValues[59] = d59
			ps67.OverlayValues[60] = d60
			ps67.OverlayValues[62] = d62
			ps67.OverlayValues[63] = d63
			ps67.OverlayValues[64] = d64
			ps67.OverlayValues[65] = d65
			ps67.PhiValues = make([]JITValueDesc, 2)
			d68 = d0
			ps67.PhiValues[0] = d68
			d69 = d8
			ps67.PhiValues[1] = d69
			alloc70 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps67)
			}
			ctx.RestoreAllocState(alloc70)
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps66)
			}
			return result
			ctx.FreeDesc(&d56)
			return result
			}
			ps71 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps71)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(24))
			ctx.W.EmitAddRSP32(int32(24))
			return result
		}, /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */ /* TODO: IndexAddr on non-parameter: &t0[t3] */
	})
	Declare(&Globalenv, &Declaration{
		"floor", "rounds the number down",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Floor(a[0].Float()))
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d0.Imm.Float())}
			} else if d0.Type == tagFloat && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else if d0.Type == tagFloat && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d0}, 1)
				d1.Type = tagFloat
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Floor(d1.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d1)
				var d3 JITValueDesc
				if d1.Loc == LocRegPair {
					ctx.FreeReg(d1.Reg)
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg2}
					ctx.BindReg(d1.Reg2, &d3)
					ctx.BindReg(d1.Reg2, &d3)
				} else {
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
					ctx.BindReg(d1.Reg, &d3)
					ctx.BindReg(d1.Reg, &d3)
				}
				d2 = ctx.EmitGoCallScalar(GoFuncAddr(JITFloorBits), []JITValueDesc{d3}, 1)
				d2.Type = tagFloat
				ctx.BindReg(d2.Reg, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			return result
			}
			ps4 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps4)
			return result
		}, /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */ /* TODO: unsupported call: archFloor(x) */
	})
	Declare(&Globalenv, &Declaration{
		"ceil", "rounds the number up",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Ceil(a[0].Float()))
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [1]BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d0.Imm.Float())}
			} else if d0.Type == tagFloat && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else if d0.Type == tagFloat && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d0}, 1)
				d1.Type = tagFloat
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Ceil(d1.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d1)
				var d3 JITValueDesc
				if d1.Loc == LocRegPair {
					ctx.FreeReg(d1.Reg)
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg2}
					ctx.BindReg(d1.Reg2, &d3)
					ctx.BindReg(d1.Reg2, &d3)
				} else {
					d3 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
					ctx.BindReg(d1.Reg, &d3)
					ctx.BindReg(d1.Reg, &d3)
				}
				d2 = ctx.EmitGoCallScalar(GoFuncAddr(JITCeilBits), []JITValueDesc{d3}, 1)
				d2.Type = tagFloat
				ctx.BindReg(d2.Reg, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			return result
			}
			ps4 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps4)
			return result
		}, /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */ /* TODO: unsupported call: archCeil(x) */
	})
	Declare(&Globalenv, &Declaration{
		"round", "rounds the number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			return NewFloat(math.Round(a[0].Float()))
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
			lbl0 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl0)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			var d1 JITValueDesc
			if d0.Loc == LocImm {
				d1 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d0.Imm.Float())}
			} else if d0.Type == tagFloat && d0.Loc == LocReg {
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d1)
				ctx.BindReg(d0.Reg, &d1)
			} else if d0.Type == tagFloat && d0.Loc == LocRegPair {
				ctx.FreeReg(d0.Reg)
				d1 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg2}
				ctx.BindReg(d0.Reg2, &d1)
				ctx.BindReg(d0.Reg2, &d1)
			} else {
				d1 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d0}, 1)
				d1.Type = tagFloat
				ctx.BindReg(d1.Reg, &d1)
			}
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocRegPair || d1.Loc == LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d2 = ctx.EmitGoCallScalar(GoFuncAddr(math.Round), []JITValueDesc{d1}, 1)
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.W.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.W.EmitMakeFloat(result, d2)
			} else {
				ctx.W.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			return result
			}
			ps3 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps3)
			return result
		}, /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */ /* TODO: unsupported BinOp &^ */
	})
	Declare(&Globalenv, &Declaration{
		"sql_abs", "SQL ABS(): returns absolute value, NULL-safe",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			v := a[0].Float()
			if v < 0 {
				v = -v
			}
			// preserve int type
			if ToInt(a[0]) == int(v) && a[0].Float() == v {
				return NewInt(int64(v))
			}
			return NewFloat(v)
		},
		true, false, nil,
		func(ctx *JITContext, args []JITValueDesc, result JITValueDesc) JITValueDesc {
			var d1 JITValueDesc
			_ = d1
			var d2 JITValueDesc
			_ = d2
			var d3 JITValueDesc
			_ = d3
			var d4 JITValueDesc
			_ = d4
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
			var d21 JITValueDesc
			_ = d21
			var d24 JITValueDesc
			_ = d24
			var d25 JITValueDesc
			_ = d25
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
			var d32 JITValueDesc
			_ = d32
			var d33 JITValueDesc
			_ = d33
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
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			var bbs [8]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d1 = args[0]
			d1.ID = 0
			d3 = d1
			d3.ID = 0
			d2 = ctx.EmitTagEqualsBorrowed(&d3, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d1)
			d4 = d2
			ctx.EnsureDesc(&d4)
			if d4.Loc != LocImm && d4.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d4.Loc == LocImm {
				if d4.Imm.Bool() {
			ps5 := PhiState{General: ps.General}
			ps5.OverlayValues = make([]JITValueDesc, 5)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
			ps5.OverlayValues[4] = d4
					return bbs[1].RenderPS(ps5)
				}
			ps6 := PhiState{General: ps.General}
			ps6.OverlayValues = make([]JITValueDesc, 5)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps6.OverlayValues[4] = d4
				return bbs[2].RenderPS(ps6)
			}
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d4.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl9)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl9)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitJmp(lbl3)
			ps7 := PhiState{General: true}
			ps7.OverlayValues = make([]JITValueDesc, 5)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			ps7.OverlayValues[4] = d4
			ps8 := PhiState{General: true}
			ps8.OverlayValues = make([]JITValueDesc, 5)
			ps8.OverlayValues[0] = d0
			ps8.OverlayValues[1] = d1
			ps8.OverlayValues[2] = d2
			ps8.OverlayValues[3] = d3
			ps8.OverlayValues[4] = d4
			alloc9 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps8)
			}
			ctx.RestoreAllocState(alloc9)
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps7)
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
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
			}
			ctx.ReclaimUntrackedRegs()
			d10 = args[0]
			d10.ID = 0
			var d11 JITValueDesc
			if d10.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d10.Imm.Float())}
			} else if d10.Type == tagFloat && d10.Loc == LocReg {
				d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg}
				ctx.BindReg(d10.Reg, &d11)
				ctx.BindReg(d10.Reg, &d11)
			} else if d10.Type == tagFloat && d10.Loc == LocRegPair {
				ctx.FreeReg(d10.Reg)
				d11 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg2}
				ctx.BindReg(d10.Reg2, &d11)
				ctx.BindReg(d10.Reg2, &d11)
			} else {
				d11 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d10}, 1)
				d11.Type = tagFloat
				ctx.BindReg(d11.Reg, &d11)
			}
			ctx.FreeDesc(&d10)
			ctx.EnsureDesc(&d11)
			var d12 JITValueDesc
			if d11.Loc == LocImm {
				d12 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d11.Imm.Float() < 0)}
			} else {
				r1 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegImm64(RegR11, uint64(0))
				ctx.W.EmitCmpFloat64Setcc(r1, d11.Reg, RegR11, CcL)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d12)
			}
			d13 = d12
			ctx.EnsureDesc(&d13)
			if d13.Loc != LocImm && d13.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d13.Loc == LocImm {
				if d13.Imm.Bool() {
			ps14 := PhiState{General: ps.General}
			ps14.OverlayValues = make([]JITValueDesc, 14)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
			ps14.OverlayValues[13] = d13
					return bbs[3].RenderPS(ps14)
				}
			d15 = d11
			if d15.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
			ps16 := PhiState{General: ps.General}
			ps16.OverlayValues = make([]JITValueDesc, 16)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[3] = d3
			ps16.OverlayValues[4] = d4
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[11] = d11
			ps16.OverlayValues[12] = d12
			ps16.OverlayValues[13] = d13
			ps16.OverlayValues[15] = d15
			ps16.PhiValues = make([]JITValueDesc, 1)
			d17 = d11
			ps16.PhiValues[0] = d17
				return bbs[4].RenderPS(ps16)
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d13.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl11)
			ctx.W.EmitJmp(lbl12)
			ctx.W.MarkLabel(lbl11)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl12)
			d18 = d11
			if d18.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d18)
			ctx.EmitStoreToStack(d18, 0)
			ctx.W.EmitJmp(lbl5)
			ps19 := PhiState{General: true}
			ps19.OverlayValues = make([]JITValueDesc, 19)
			ps19.OverlayValues[0] = d0
			ps19.OverlayValues[1] = d1
			ps19.OverlayValues[2] = d2
			ps19.OverlayValues[3] = d3
			ps19.OverlayValues[4] = d4
			ps19.OverlayValues[10] = d10
			ps19.OverlayValues[11] = d11
			ps19.OverlayValues[12] = d12
			ps19.OverlayValues[13] = d13
			ps19.OverlayValues[15] = d15
			ps19.OverlayValues[17] = d17
			ps19.OverlayValues[18] = d18
			ps20 := PhiState{General: true}
			ps20.OverlayValues = make([]JITValueDesc, 19)
			ps20.OverlayValues[0] = d0
			ps20.OverlayValues[1] = d1
			ps20.OverlayValues[2] = d2
			ps20.OverlayValues[3] = d3
			ps20.OverlayValues[4] = d4
			ps20.OverlayValues[10] = d10
			ps20.OverlayValues[11] = d11
			ps20.OverlayValues[12] = d12
			ps20.OverlayValues[13] = d13
			ps20.OverlayValues[15] = d15
			ps20.OverlayValues[17] = d17
			ps20.OverlayValues[18] = d18
			ps20.PhiValues = make([]JITValueDesc, 1)
			d21 = d11
			ps20.PhiValues[0] = d21
			snap22 := d11
			alloc23 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps20)
			}
			ctx.RestoreAllocState(alloc23)
			d11 = snap22
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps19)
			}
			return result
			ctx.FreeDesc(&d12)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
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
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d11)
			var d24 JITValueDesc
			if d11.Loc == LocImm {
				if d11.Type == tagFloat {
					d24 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d11.Imm.Float())}
				} else {
					d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d11.Imm.Int())}
				}
			} else {
				if d11.Type == tagFloat {
					r2 := ctx.AllocRegExcept(d11.Reg)
					ctx.W.EmitMovRegImm64(r2, 0)
					ctx.W.EmitSubFloat64(r2, d11.Reg)
					d24 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
					ctx.BindReg(r2, &d24)
				} else {
					r3 := ctx.AllocRegExcept(d11.Reg)
					ctx.W.EmitMovRegImm64(r3, 0)
					ctx.W.EmitSubInt64(r3, d11.Reg)
					d24 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
					ctx.BindReg(r3, &d24)
				}
			}
			d25 = d24
			if d25.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			ctx.EmitStoreToStack(d25, 0)
			ps26 := PhiState{General: ps.General}
			ps26.OverlayValues = make([]JITValueDesc, 26)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[10] = d10
			ps26.OverlayValues[11] = d11
			ps26.OverlayValues[12] = d12
			ps26.OverlayValues[13] = d13
			ps26.OverlayValues[15] = d15
			ps26.OverlayValues[17] = d17
			ps26.OverlayValues[18] = d18
			ps26.OverlayValues[21] = d21
			ps26.OverlayValues[24] = d24
			ps26.OverlayValues[25] = d25
			ps26.PhiValues = make([]JITValueDesc, 1)
			d27 = d24
			ps26.PhiValues[0] = d27
			if ps26.General && bbs[4].Rendered {
				ctx.W.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps26)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d28 := ps.PhiValues[0]
						ctx.EnsureDesc(&d28)
						ctx.EmitStoreToStack(d28, 0)
					}
					ps.General = true
					return bbs[4].RenderPS(ps)
				}
			}
			bbs[4].VisitCount++
			if ps.General {
				if bbs[4].Rendered {
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
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
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d29 = args[0]
			d29.ID = 0
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			if d29.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d29.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d29.Imm.GetTag() == tagBool {
					ctx.W.EmitMakeBool(tmpPair, d29)
				} else if d29.Imm.GetTag() == tagInt {
					ctx.W.EmitMakeInt(tmpPair, d29)
				} else if d29.Imm.GetTag() == tagFloat {
					ctx.W.EmitMakeFloat(tmpPair, d29)
				} else if d29.Imm.GetTag() == tagNil {
					ctx.W.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d29.Imm.RawWords()
					ctx.W.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.W.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d29 = tmpPair
			} else if d29.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d29.Type, Reg: ctx.AllocRegExcept(d29.Reg), Reg2: ctx.AllocRegExcept(d29.Reg)}
				switch d29.Type {
				case tagBool:
					ctx.W.EmitMakeBool(tmpPair, d29)
				case tagInt:
					ctx.W.EmitMakeInt(tmpPair, d29)
				case tagFloat:
					ctx.W.EmitMakeFloat(tmpPair, d29)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d29)
				d29 = tmpPair
			}
			if d29.Loc != LocRegPair && d29.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d30 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d29}, 1)
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d31 JITValueDesc
			if d0.Loc == LocImm {
				d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d0.Imm.Float()))}
			} else {
				r4 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r4, d0.Reg)
				d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
				ctx.BindReg(r4, &d31)
			}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d31)
			var d32 JITValueDesc
			if d30.Loc == LocImm && d31.Loc == LocImm {
				d32 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d30.Imm.Int() == d31.Imm.Int())}
			} else if d31.Loc == LocImm {
				r5 := ctx.AllocReg()
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d30.Reg, int32(d31.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(RegR11, uint64(d31.Imm.Int()))
					ctx.W.EmitCmpInt64(d30.Reg, RegR11)
				}
				ctx.W.EmitSetcc(r5, CcE)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d32)
			} else if d30.Loc == LocImm {
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(RegR11, uint64(d30.Imm.Int()))
				ctx.W.EmitCmpInt64(RegR11, d31.Reg)
				ctx.W.EmitSetcc(r6, CcE)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d32)
			} else {
				r7 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d30.Reg, d31.Reg)
				ctx.W.EmitSetcc(r7, CcE)
				d32 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d32)
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d31)
			d33 = d32
			ctx.EnsureDesc(&d33)
			if d33.Loc != LocImm && d33.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d33.Loc == LocImm {
				if d33.Imm.Bool() {
			ps34 := PhiState{General: ps.General}
			ps34.OverlayValues = make([]JITValueDesc, 34)
			ps34.OverlayValues[0] = d0
			ps34.OverlayValues[1] = d1
			ps34.OverlayValues[2] = d2
			ps34.OverlayValues[3] = d3
			ps34.OverlayValues[4] = d4
			ps34.OverlayValues[10] = d10
			ps34.OverlayValues[11] = d11
			ps34.OverlayValues[12] = d12
			ps34.OverlayValues[13] = d13
			ps34.OverlayValues[15] = d15
			ps34.OverlayValues[17] = d17
			ps34.OverlayValues[18] = d18
			ps34.OverlayValues[21] = d21
			ps34.OverlayValues[24] = d24
			ps34.OverlayValues[25] = d25
			ps34.OverlayValues[27] = d27
			ps34.OverlayValues[28] = d28
			ps34.OverlayValues[29] = d29
			ps34.OverlayValues[30] = d30
			ps34.OverlayValues[31] = d31
			ps34.OverlayValues[32] = d32
			ps34.OverlayValues[33] = d33
					return bbs[7].RenderPS(ps34)
				}
			ps35 := PhiState{General: ps.General}
			ps35.OverlayValues = make([]JITValueDesc, 34)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[1] = d1
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[3] = d3
			ps35.OverlayValues[4] = d4
			ps35.OverlayValues[10] = d10
			ps35.OverlayValues[11] = d11
			ps35.OverlayValues[12] = d12
			ps35.OverlayValues[13] = d13
			ps35.OverlayValues[15] = d15
			ps35.OverlayValues[17] = d17
			ps35.OverlayValues[18] = d18
			ps35.OverlayValues[21] = d21
			ps35.OverlayValues[24] = d24
			ps35.OverlayValues[25] = d25
			ps35.OverlayValues[27] = d27
			ps35.OverlayValues[28] = d28
			ps35.OverlayValues[29] = d29
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[32] = d32
			ps35.OverlayValues[33] = d33
				return bbs[6].RenderPS(ps35)
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d33.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl13)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl14)
			ctx.W.EmitJmp(lbl7)
			ps36 := PhiState{General: true}
			ps36.OverlayValues = make([]JITValueDesc, 34)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[1] = d1
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[3] = d3
			ps36.OverlayValues[4] = d4
			ps36.OverlayValues[10] = d10
			ps36.OverlayValues[11] = d11
			ps36.OverlayValues[12] = d12
			ps36.OverlayValues[13] = d13
			ps36.OverlayValues[15] = d15
			ps36.OverlayValues[17] = d17
			ps36.OverlayValues[18] = d18
			ps36.OverlayValues[21] = d21
			ps36.OverlayValues[24] = d24
			ps36.OverlayValues[25] = d25
			ps36.OverlayValues[27] = d27
			ps36.OverlayValues[28] = d28
			ps36.OverlayValues[29] = d29
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[32] = d32
			ps36.OverlayValues[33] = d33
			ps37 := PhiState{General: true}
			ps37.OverlayValues = make([]JITValueDesc, 34)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[3] = d3
			ps37.OverlayValues[4] = d4
			ps37.OverlayValues[10] = d10
			ps37.OverlayValues[11] = d11
			ps37.OverlayValues[12] = d12
			ps37.OverlayValues[13] = d13
			ps37.OverlayValues[15] = d15
			ps37.OverlayValues[17] = d17
			ps37.OverlayValues[18] = d18
			ps37.OverlayValues[21] = d21
			ps37.OverlayValues[24] = d24
			ps37.OverlayValues[25] = d25
			ps37.OverlayValues[27] = d27
			ps37.OverlayValues[28] = d28
			ps37.OverlayValues[29] = d29
			ps37.OverlayValues[30] = d30
			ps37.OverlayValues[31] = d31
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			snap38 := d0
			alloc39 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps37)
			}
			ctx.RestoreAllocState(alloc39)
			d0 = snap38
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps36)
			}
			return result
			ctx.FreeDesc(&d32)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
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
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
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
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d40 JITValueDesc
			if d0.Loc == LocImm {
				d40 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d0.Imm.Float()))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitCvtFloatBitsToInt64(r8, d0.Reg)
				d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d40)
			}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			ctx.W.EmitMakeInt(result, d40)
			if d40.Loc == LocReg { ctx.FreeReg(d40.Reg) }
			result.Type = tagInt
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
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
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
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
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.W.EmitMakeFloat(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: JITTypeUnknown, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
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
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
				d4 = ps.OverlayValues[4]
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
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
				d25 = ps.OverlayValues[25]
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
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			ctx.ReclaimUntrackedRegs()
			d41 = args[0]
			d41.ID = 0
			var d42 JITValueDesc
			if d41.Loc == LocImm {
				d42 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d41.Imm.Float())}
			} else if d41.Type == tagFloat && d41.Loc == LocReg {
				d42 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d41.Reg}
				ctx.BindReg(d41.Reg, &d42)
				ctx.BindReg(d41.Reg, &d42)
			} else if d41.Type == tagFloat && d41.Loc == LocRegPair {
				ctx.FreeReg(d41.Reg)
				d42 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d41.Reg2}
				ctx.BindReg(d41.Reg2, &d42)
				ctx.BindReg(d41.Reg2, &d42)
			} else {
				d42 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d41}, 1)
				d42.Type = tagFloat
				ctx.BindReg(d42.Reg, &d42)
			}
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d0)
			var d43 JITValueDesc
			if d42.Loc == LocImm && d0.Loc == LocImm {
				d43 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d42.Imm.Float() == d0.Imm.Float())}
			} else if d0.Loc == LocImm {
				r9 := ctx.AllocReg()
				_, yBits := d0.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, yBits)
				ctx.W.EmitCmpFloat64Setcc(r9, d42.Reg, RegR11, CcE)
				d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d43)
			} else if d42.Loc == LocImm {
				r10 := ctx.AllocRegExcept(d0.Reg)
				_, xBits := d42.Imm.RawWords()
				ctx.W.EmitMovRegImm64(RegR11, xBits)
				ctx.W.EmitCmpFloat64Setcc(r10, RegR11, d0.Reg, CcE)
				d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d43)
			} else {
				r11 := ctx.AllocRegExcept(d42.Reg, d0.Reg)
				ctx.W.EmitCmpFloat64Setcc(r11, d42.Reg, d0.Reg, CcE)
				d43 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d43)
			}
			ctx.FreeDesc(&d42)
			d44 = d43
			ctx.EnsureDesc(&d44)
			if d44.Loc != LocImm && d44.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d44.Loc == LocImm {
				if d44.Imm.Bool() {
			ps45 := PhiState{General: ps.General}
			ps45.OverlayValues = make([]JITValueDesc, 45)
			ps45.OverlayValues[0] = d0
			ps45.OverlayValues[1] = d1
			ps45.OverlayValues[2] = d2
			ps45.OverlayValues[3] = d3
			ps45.OverlayValues[4] = d4
			ps45.OverlayValues[10] = d10
			ps45.OverlayValues[11] = d11
			ps45.OverlayValues[12] = d12
			ps45.OverlayValues[13] = d13
			ps45.OverlayValues[15] = d15
			ps45.OverlayValues[17] = d17
			ps45.OverlayValues[18] = d18
			ps45.OverlayValues[21] = d21
			ps45.OverlayValues[24] = d24
			ps45.OverlayValues[25] = d25
			ps45.OverlayValues[27] = d27
			ps45.OverlayValues[28] = d28
			ps45.OverlayValues[29] = d29
			ps45.OverlayValues[30] = d30
			ps45.OverlayValues[31] = d31
			ps45.OverlayValues[32] = d32
			ps45.OverlayValues[33] = d33
			ps45.OverlayValues[40] = d40
			ps45.OverlayValues[41] = d41
			ps45.OverlayValues[42] = d42
			ps45.OverlayValues[43] = d43
			ps45.OverlayValues[44] = d44
					return bbs[5].RenderPS(ps45)
				}
			ps46 := PhiState{General: ps.General}
			ps46.OverlayValues = make([]JITValueDesc, 45)
			ps46.OverlayValues[0] = d0
			ps46.OverlayValues[1] = d1
			ps46.OverlayValues[2] = d2
			ps46.OverlayValues[3] = d3
			ps46.OverlayValues[4] = d4
			ps46.OverlayValues[10] = d10
			ps46.OverlayValues[11] = d11
			ps46.OverlayValues[12] = d12
			ps46.OverlayValues[13] = d13
			ps46.OverlayValues[15] = d15
			ps46.OverlayValues[17] = d17
			ps46.OverlayValues[18] = d18
			ps46.OverlayValues[21] = d21
			ps46.OverlayValues[24] = d24
			ps46.OverlayValues[25] = d25
			ps46.OverlayValues[27] = d27
			ps46.OverlayValues[28] = d28
			ps46.OverlayValues[29] = d29
			ps46.OverlayValues[30] = d30
			ps46.OverlayValues[31] = d31
			ps46.OverlayValues[32] = d32
			ps46.OverlayValues[33] = d33
			ps46.OverlayValues[40] = d40
			ps46.OverlayValues[41] = d41
			ps46.OverlayValues[42] = d42
			ps46.OverlayValues[43] = d43
			ps46.OverlayValues[44] = d44
				return bbs[6].RenderPS(ps46)
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d44.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl15)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl15)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl16)
			ctx.W.EmitJmp(lbl7)
			ps47 := PhiState{General: true}
			ps47.OverlayValues = make([]JITValueDesc, 45)
			ps47.OverlayValues[0] = d0
			ps47.OverlayValues[1] = d1
			ps47.OverlayValues[2] = d2
			ps47.OverlayValues[3] = d3
			ps47.OverlayValues[4] = d4
			ps47.OverlayValues[10] = d10
			ps47.OverlayValues[11] = d11
			ps47.OverlayValues[12] = d12
			ps47.OverlayValues[13] = d13
			ps47.OverlayValues[15] = d15
			ps47.OverlayValues[17] = d17
			ps47.OverlayValues[18] = d18
			ps47.OverlayValues[21] = d21
			ps47.OverlayValues[24] = d24
			ps47.OverlayValues[25] = d25
			ps47.OverlayValues[27] = d27
			ps47.OverlayValues[28] = d28
			ps47.OverlayValues[29] = d29
			ps47.OverlayValues[30] = d30
			ps47.OverlayValues[31] = d31
			ps47.OverlayValues[32] = d32
			ps47.OverlayValues[33] = d33
			ps47.OverlayValues[40] = d40
			ps47.OverlayValues[41] = d41
			ps47.OverlayValues[42] = d42
			ps47.OverlayValues[43] = d43
			ps47.OverlayValues[44] = d44
			ps48 := PhiState{General: true}
			ps48.OverlayValues = make([]JITValueDesc, 45)
			ps48.OverlayValues[0] = d0
			ps48.OverlayValues[1] = d1
			ps48.OverlayValues[2] = d2
			ps48.OverlayValues[3] = d3
			ps48.OverlayValues[4] = d4
			ps48.OverlayValues[10] = d10
			ps48.OverlayValues[11] = d11
			ps48.OverlayValues[12] = d12
			ps48.OverlayValues[13] = d13
			ps48.OverlayValues[15] = d15
			ps48.OverlayValues[17] = d17
			ps48.OverlayValues[18] = d18
			ps48.OverlayValues[21] = d21
			ps48.OverlayValues[24] = d24
			ps48.OverlayValues[25] = d25
			ps48.OverlayValues[27] = d27
			ps48.OverlayValues[28] = d28
			ps48.OverlayValues[29] = d29
			ps48.OverlayValues[30] = d30
			ps48.OverlayValues[31] = d31
			ps48.OverlayValues[32] = d32
			ps48.OverlayValues[33] = d33
			ps48.OverlayValues[40] = d40
			ps48.OverlayValues[41] = d41
			ps48.OverlayValues[42] = d42
			ps48.OverlayValues[43] = d43
			ps48.OverlayValues[44] = d44
			snap49 := d0
			snap50 := d40
			alloc51 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps48)
			}
			ctx.RestoreAllocState(alloc51)
			d0 = snap49
			d40 = snap50
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps47)
			}
			return result
			ctx.FreeDesc(&d43)
			return result
			}
			ps52 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps52)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(8))
			ctx.W.EmitAddRSP32(int32(8))
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
	})
	Declare(&Globalenv, &Declaration{
		"sqrt", "returns the square root of a number",
		1, 1,
		[]DeclarationParameter{
			DeclarationParameter{"value", "number", "value", nil},
		}, "number",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			v := a[0].Float()
			if v < 0 {
				return NewNil()
			}
			return NewFloat(math.Sqrt(v))
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
			var d18 JITValueDesc
			_ = d18
			var d19 JITValueDesc
			_ = d19
		/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			var bbs [5]BBDescriptor
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			d0 = args[0]
			d0.ID = 0
			d2 = d0
			d2.ID = 0
			d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			d3 = d1
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
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d3.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl6)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl6)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl7)
			ctx.W.EmitJmp(lbl3)
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
			ctx.FreeDesc(&d1)
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
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
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
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
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
				d10 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d9.Imm.Float())}
			} else if d9.Type == tagFloat && d9.Loc == LocReg {
				d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d9.Reg}
				ctx.BindReg(d9.Reg, &d10)
				ctx.BindReg(d9.Reg, &d10)
			} else if d9.Type == tagFloat && d9.Loc == LocRegPair {
				ctx.FreeReg(d9.Reg)
				d10 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d9.Reg2}
				ctx.BindReg(d9.Reg2, &d10)
				ctx.BindReg(d9.Reg2, &d10)
			} else {
				d10 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d9}, 1)
				d10.Type = tagFloat
				ctx.BindReg(d10.Reg, &d10)
			}
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d10)
			var d11 JITValueDesc
			if d10.Loc == LocImm {
				d11 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d10.Imm.Float() < 0)}
			} else {
				r0 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegImm64(RegR11, uint64(0))
				ctx.W.EmitCmpFloat64Setcc(r0, d10.Reg, RegR11, CcL)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d11)
			}
			d12 = d11
			ctx.EnsureDesc(&d12)
			if d12.Loc != LocImm && d12.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d12.Loc == LocImm {
				if d12.Imm.Bool() {
			ps13 := PhiState{General: ps.General}
			ps13.OverlayValues = make([]JITValueDesc, 13)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[3] = d3
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
			ps13.OverlayValues[12] = d12
					return bbs[3].RenderPS(ps13)
				}
			ps14 := PhiState{General: ps.General}
			ps14.OverlayValues = make([]JITValueDesc, 13)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
				return bbs[4].RenderPS(ps14)
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d12.Reg, 0)
			ctx.W.EmitJcc(CcNE, lbl8)
			ctx.W.EmitJmp(lbl9)
			ctx.W.MarkLabel(lbl8)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl9)
			ctx.W.EmitJmp(lbl5)
			ps15 := PhiState{General: true}
			ps15.OverlayValues = make([]JITValueDesc, 13)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			ps15.OverlayValues[12] = d12
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 13)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[3] = d3
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[11] = d11
			ps16.OverlayValues[12] = d12
			alloc17 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc17)
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps15)
			}
			return result
			ctx.FreeDesc(&d11)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
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
			ctx.W.EmitMakeNil(result)
			result.Type = tagNil
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
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
			ctx.EnsureDesc(&d10)
			var d18 JITValueDesc
			if d10.Loc == LocImm {
				d18 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d10.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d10)
				var d19 JITValueDesc
				if d10.Loc == LocRegPair {
					ctx.FreeReg(d10.Reg)
					d19 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg2}
					ctx.BindReg(d10.Reg2, &d19)
					ctx.BindReg(d10.Reg2, &d19)
				} else {
					d19 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d10.Reg}
					ctx.BindReg(d10.Reg, &d19)
					ctx.BindReg(d10.Reg, &d19)
				}
				d18 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d19}, 1)
				d18.Type = tagFloat
				ctx.BindReg(d18.Reg, &d18)
			}
			ctx.FreeDesc(&d10)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			ctx.W.EmitMakeFloat(result, d18)
			if d18.Loc == LocReg { ctx.FreeReg(d18.Reg) }
			result.Type = tagFloat
			ctx.W.EmitJmp(lbl0)
			return result
			}
			ps20 := PhiState{General: false}
			_ = bbs[0].RenderPS(ps20)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			return result
		}, /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */ /* TODO: unsupported compare const kind: 0:float64 */
	})
	Declare(&Globalenv, &Declaration{
		"sql_rand", "SQL RAND(): returns a random float in [0,1)",
		0, 0,
		[]DeclarationParameter{}, "number",
		func(a ...Scmer) Scmer {
			var buf [8]byte
			if _, err := crand.Read(buf[:]); err != nil {
				panic("sql_rand: " + err.Error())
			}
			// 53 random bits map exactly into float64 mantissa range.
			u := binary.LittleEndian.Uint64(buf[:]) >> 11
			return NewFloat(float64(u) / (1 << 53))
		},
		true, false, nil,
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
}
