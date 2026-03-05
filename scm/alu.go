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
			d1 = ctx.EmitGetTagDesc(&d0, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d2 JITValueDesc
			if d1.Loc == LocImm {
				d2 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d1.Imm.Int()) == uint64(4))}
			} else {
				r0 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d1.Reg, 4)
				ctx.EmitSetcc(r0, CcE)
				d2 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
				ctx.BindReg(r0, &d2)
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeBool(result, d2)
			} else {
				ctx.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
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
			var d19 JITValueDesc
			_ = d19
			var d20 JITValueDesc
			_ = d20
			var d22 JITValueDesc
			_ = d22
			var d23 JITValueDesc
			_ = d23
			var d24 JITValueDesc
			_ = d24
			var d25 JITValueDesc
			_ = d25
			var d27 JITValueDesc
			_ = d27
			var d31 JITValueDesc
			_ = d31
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(16)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(16)
			}
			d0 := JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(0)}
			var bbs [4]BBDescriptor
			bbs[2].PhiBase = int32(0)
			bbs[2].PhiCount = uint16(1)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(0)}
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
				ctx.EmitCmpRegImm32(d2.Reg, 3)
				ctx.EmitSetcc(r1, CcE)
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
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
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl5 := ctx.ReserveLabel()
			lbl6 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d4.Reg, 0)
			ctx.EmitJcc(CcNE, lbl5)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl5)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl6)
			ctx.EmitJmp(lbl4)
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
			snap11 := d0
			snap12 := d1
			snap13 := d2
			snap14 := d3
			snap15 := d4
			snap16 := d6
			snap17 := d10
			alloc18 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps8)
			}
			ctx.RestoreAllocState(alloc18)
			d0 = snap11
			d1 = snap12
			d2 = snap13
			d3 = snap14
			d4 = snap15
			d6 = snap16
			d10 = snap17
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
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(0)}
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
			var d19 JITValueDesc
			if d2.Loc == LocImm {
				d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(16))}
			} else {
				r2 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitCmpRegImm32(d2.Reg, 16)
				ctx.EmitSetcc(r2, CcE)
				d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d19)
			}
			ctx.EnsureDesc(&d19)
			if d19.Loc == LocReg {
				ctx.ProtectReg(d19.Reg)
			} else if d19.Loc == LocRegPair {
				ctx.ProtectReg(d19.Reg)
				ctx.ProtectReg(d19.Reg2)
			}
			d20 = d19
			if d20.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, int32(bbs[2].PhiBase)+int32(0))
			if d19.Loc == LocReg {
				ctx.UnprotectReg(d19.Reg)
			} else if d19.Loc == LocRegPair {
				ctx.UnprotectReg(d19.Reg)
				ctx.UnprotectReg(d19.Reg2)
			}
			ps21 := PhiState{General: ps.General}
			ps21.OverlayValues = make([]JITValueDesc, 21)
			ps21.OverlayValues[0] = d0
			ps21.OverlayValues[1] = d1
			ps21.OverlayValues[2] = d2
			ps21.OverlayValues[3] = d3
			ps21.OverlayValues[4] = d4
			ps21.OverlayValues[6] = d6
			ps21.OverlayValues[10] = d10
			ps21.OverlayValues[19] = d19
			ps21.OverlayValues[20] = d20
			ps21.PhiValues = make([]JITValueDesc, 1)
			d22 = d19
			ps21.PhiValues[0] = d22
			if ps21.General && bbs[2].Rendered {
				ctx.EmitJmp(lbl3)
				return result
			}
			return bbs[2].RenderPS(ps21)
			return result
			}
			bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d23 := ps.PhiValues[0]
					ctx.EnsureDesc(&d23)
					ctx.EmitStoreToStack(d23, int32(bbs[2].PhiBase)+int32(0))
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.EmitMakeBool(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagBool
			ctx.EmitJmp(lbl0)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagBool, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != LocNone {
				d23 = ps.OverlayValues[23]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d24 JITValueDesc
			if d2.Loc == LocImm {
				d24 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(uint64(d2.Imm.Int()) == uint64(4))}
			} else {
				r3 := ctx.AllocReg()
				ctx.EmitCmpRegImm32(d2.Reg, 4)
				ctx.EmitSetcc(r3, CcE)
				d24 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d24)
			}
			ctx.FreeDesc(&d2)
			d25 = d24
			ctx.EnsureDesc(&d25)
			if d25.Loc != LocImm && d25.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d25.Loc == LocImm {
				if d25.Imm.Bool() {
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
			ps26 := PhiState{General: ps.General}
			ps26.OverlayValues = make([]JITValueDesc, 26)
			ps26.OverlayValues[0] = d0
			ps26.OverlayValues[1] = d1
			ps26.OverlayValues[2] = d2
			ps26.OverlayValues[3] = d3
			ps26.OverlayValues[4] = d4
			ps26.OverlayValues[6] = d6
			ps26.OverlayValues[10] = d10
			ps26.OverlayValues[19] = d19
			ps26.OverlayValues[20] = d20
			ps26.OverlayValues[22] = d22
			ps26.OverlayValues[23] = d23
			ps26.OverlayValues[24] = d24
			ps26.OverlayValues[25] = d25
			ps26.PhiValues = make([]JITValueDesc, 1)
			d27 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			ps26.PhiValues[0] = d27
					return bbs[2].RenderPS(ps26)
				}
			ps28 := PhiState{General: ps.General}
			ps28.OverlayValues = make([]JITValueDesc, 28)
			ps28.OverlayValues[0] = d0
			ps28.OverlayValues[1] = d1
			ps28.OverlayValues[2] = d2
			ps28.OverlayValues[3] = d3
			ps28.OverlayValues[4] = d4
			ps28.OverlayValues[6] = d6
			ps28.OverlayValues[10] = d10
			ps28.OverlayValues[19] = d19
			ps28.OverlayValues[20] = d20
			ps28.OverlayValues[22] = d22
			ps28.OverlayValues[23] = d23
			ps28.OverlayValues[24] = d24
			ps28.OverlayValues[25] = d25
			ps28.OverlayValues[27] = d27
				return bbs[1].RenderPS(ps28)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl7 := ctx.ReserveLabel()
			lbl8 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d25.Reg, 0)
			ctx.EmitJcc(CcNE, lbl7)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl7)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewInt(1)}, int32(bbs[2].PhiBase)+int32(0))
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl8)
			ctx.EmitJmp(lbl2)
			ps29 := PhiState{General: true}
			ps29.OverlayValues = make([]JITValueDesc, 28)
			ps29.OverlayValues[0] = d0
			ps29.OverlayValues[1] = d1
			ps29.OverlayValues[2] = d2
			ps29.OverlayValues[3] = d3
			ps29.OverlayValues[4] = d4
			ps29.OverlayValues[6] = d6
			ps29.OverlayValues[10] = d10
			ps29.OverlayValues[19] = d19
			ps29.OverlayValues[20] = d20
			ps29.OverlayValues[22] = d22
			ps29.OverlayValues[23] = d23
			ps29.OverlayValues[24] = d24
			ps29.OverlayValues[25] = d25
			ps29.OverlayValues[27] = d27
			ps29.PhiValues = make([]JITValueDesc, 1)
			d31 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(true)}
			ps29.PhiValues[0] = d31
			ps30 := PhiState{General: true}
			ps30.OverlayValues = make([]JITValueDesc, 32)
			ps30.OverlayValues[0] = d0
			ps30.OverlayValues[1] = d1
			ps30.OverlayValues[2] = d2
			ps30.OverlayValues[3] = d3
			ps30.OverlayValues[4] = d4
			ps30.OverlayValues[6] = d6
			ps30.OverlayValues[10] = d10
			ps30.OverlayValues[19] = d19
			ps30.OverlayValues[20] = d20
			ps30.OverlayValues[22] = d22
			ps30.OverlayValues[23] = d23
			ps30.OverlayValues[24] = d24
			ps30.OverlayValues[25] = d25
			ps30.OverlayValues[27] = d27
			ps30.OverlayValues[31] = d31
			snap32 := d0
			snap33 := d1
			snap34 := d2
			snap35 := d3
			snap36 := d4
			snap37 := d6
			snap38 := d10
			snap39 := d19
			snap40 := d20
			snap41 := d22
			snap42 := d23
			snap43 := d24
			snap44 := d25
			snap45 := d27
			snap46 := d31
			alloc47 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps29)
			}
			ctx.RestoreAllocState(alloc47)
			d0 = snap32
			d1 = snap33
			d2 = snap34
			d3 = snap35
			d4 = snap36
			d6 = snap37
			d10 = snap38
			d19 = snap39
			d20 = snap40
			d22 = snap41
			d23 = snap42
			d24 = snap43
			d25 = snap44
			d27 = snap45
			d31 = snap46
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps30)
			}
			return result
			ctx.FreeDesc(&d24)
			return result
			}
			argPinned48 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned48 = append(argPinned48, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned48 = append(argPinned48, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned48 = append(argPinned48, ai.Reg2)
					}
				}
			}
			ps49 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps49)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(16))
			ctx.EmitAddRSP32(int32(16))
			for _, r := range argPinned48 {
				ctx.UnprotectReg(r)
			}
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
			var d28 JITValueDesc
			_ = d28
			var d29 JITValueDesc
			_ = d29
			var d30 JITValueDesc
			_ = d30
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
			var d57 JITValueDesc
			_ = d57
			var d58 JITValueDesc
			_ = d58
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
			var d145 JITValueDesc
			_ = d145
			var d146 JITValueDesc
			_ = d146
			var d147 JITValueDesc
			_ = d147
			var d148 JITValueDesc
			_ = d148
			var d149 JITValueDesc
			_ = d149
			var d152 JITValueDesc
			_ = d152
			var d153 JITValueDesc
			_ = d153
			var d202 JITValueDesc
			_ = d202
			var d203 JITValueDesc
			_ = d203
			var d204 JITValueDesc
			_ = d204
			var d205 JITValueDesc
			_ = d205
			var d206 JITValueDesc
			_ = d206
			var d208 JITValueDesc
			_ = d208
			var d209 JITValueDesc
			_ = d209
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(64)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(64)
			}
			d0 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
			var bbs [12]BBDescriptor
			bbs[3].PhiBase = int32(0)
			bbs[3].PhiCount = uint16(2)
			bbs[9].PhiBase = int32(32)
			bbs[9].PhiCount = uint16(2)
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
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.ReserveLabel()
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(0))
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[3].PhiBase)+int32(16))
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
				ctx.EmitJmp(lbl4)
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
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
				lbl13 := ctx.ReserveLabel()
				lbl14 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d1.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl14)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d1.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r1, ai.Reg)
						ctx.EmitMovRegReg(r2, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r1, tmp.Reg)
						ctx.EmitMovRegReg(r2, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
						ctx.BindReg(r1, &pair)
						ctx.BindReg(r2, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r1, uint64(ptrWord))
							ctx.EmitMovRegImm64(r2, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl14)
				d8 := JITValueDesc{Loc: LocRegPair, Reg: r1, Reg2: r2}
				ctx.BindReg(r1, &d8)
				ctx.BindReg(r2, &d8)
				ctx.BindReg(r1, &d8)
				ctx.BindReg(r2, &d8)
				ctx.EmitMakeNil(d8)
				ctx.MarkLabel(lbl13)
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
			if !ps.General {
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d11.Reg, 0)
			ctx.EmitJcc(CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl3)
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
			snap18 := d2
			snap19 := d3
			snap20 := d5
			snap21 := d6
			snap22 := d7
			snap23 := d8
			snap24 := d9
			snap25 := d10
			snap26 := d11
			alloc27 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps15)
			}
			ctx.RestoreAllocState(alloc27)
			d0 = snap16
			d1 = snap17
			d2 = snap18
			d3 = snap19
			d5 = snap20
			d6 = snap21
			d7 = snap22
			d8 = snap23
			d9 = snap24
			d10 = snap25
			d11 = snap26
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
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
			d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d28)
			var d29 JITValueDesc
			if d1.Loc == LocImm && d28.Loc == LocImm {
				d29 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() == d28.Imm.Int())}
			} else if d28.Loc == LocImm {
				r3 := ctx.AllocRegExcept(d1.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d1.Reg, int32(d28.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d28.Imm.Int()))
					ctx.EmitCmpInt64(d1.Reg, RegR11)
				}
				ctx.EmitSetcc(r3, CcE)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r3}
				ctx.BindReg(r3, &d29)
			} else if d1.Loc == LocImm {
				r4 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d28.Reg)
				ctx.EmitSetcc(r4, CcE)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r4}
				ctx.BindReg(r4, &d29)
			} else {
				r5 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitCmpInt64(d1.Reg, d28.Reg)
				ctx.EmitSetcc(r5, CcE)
				d29 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d29)
			}
			ctx.FreeDesc(&d28)
			d30 = d29
			ctx.EnsureDesc(&d30)
			if d30.Loc != LocImm && d30.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d30.Loc == LocImm {
				if d30.Imm.Bool() {
			ps31 := PhiState{General: ps.General}
			ps31.OverlayValues = make([]JITValueDesc, 31)
			ps31.OverlayValues[0] = d0
			ps31.OverlayValues[1] = d1
			ps31.OverlayValues[2] = d2
			ps31.OverlayValues[3] = d3
			ps31.OverlayValues[5] = d5
			ps31.OverlayValues[6] = d6
			ps31.OverlayValues[7] = d7
			ps31.OverlayValues[8] = d8
			ps31.OverlayValues[9] = d9
			ps31.OverlayValues[10] = d10
			ps31.OverlayValues[11] = d11
			ps31.OverlayValues[28] = d28
			ps31.OverlayValues[29] = d29
			ps31.OverlayValues[30] = d30
					return bbs[5].RenderPS(ps31)
				}
			ps32 := PhiState{General: ps.General}
			ps32.OverlayValues = make([]JITValueDesc, 31)
			ps32.OverlayValues[0] = d0
			ps32.OverlayValues[1] = d1
			ps32.OverlayValues[2] = d2
			ps32.OverlayValues[3] = d3
			ps32.OverlayValues[5] = d5
			ps32.OverlayValues[6] = d6
			ps32.OverlayValues[7] = d7
			ps32.OverlayValues[8] = d8
			ps32.OverlayValues[9] = d9
			ps32.OverlayValues[10] = d10
			ps32.OverlayValues[11] = d11
			ps32.OverlayValues[28] = d28
			ps32.OverlayValues[29] = d29
			ps32.OverlayValues[30] = d30
				return bbs[6].RenderPS(ps32)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d30.Reg, 0)
			ctx.EmitJcc(CcNE, lbl17)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl7)
			ps33 := PhiState{General: true}
			ps33.OverlayValues = make([]JITValueDesc, 31)
			ps33.OverlayValues[0] = d0
			ps33.OverlayValues[1] = d1
			ps33.OverlayValues[2] = d2
			ps33.OverlayValues[3] = d3
			ps33.OverlayValues[5] = d5
			ps33.OverlayValues[6] = d6
			ps33.OverlayValues[7] = d7
			ps33.OverlayValues[8] = d8
			ps33.OverlayValues[9] = d9
			ps33.OverlayValues[10] = d10
			ps33.OverlayValues[11] = d11
			ps33.OverlayValues[28] = d28
			ps33.OverlayValues[29] = d29
			ps33.OverlayValues[30] = d30
			ps34 := PhiState{General: true}
			ps34.OverlayValues = make([]JITValueDesc, 31)
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
			ps34.OverlayValues[28] = d28
			ps34.OverlayValues[29] = d29
			ps34.OverlayValues[30] = d30
			snap35 := d0
			snap36 := d1
			snap37 := d2
			snap38 := d3
			snap39 := d5
			snap40 := d6
			snap41 := d7
			snap42 := d8
			snap43 := d9
			snap44 := d10
			snap45 := d11
			snap46 := d28
			snap47 := d29
			snap48 := d30
			alloc49 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps34)
			}
			ctx.RestoreAllocState(alloc49)
			d0 = snap35
			d1 = snap36
			d2 = snap37
			d3 = snap38
			d5 = snap39
			d6 = snap40
			d7 = snap41
			d8 = snap42
			d9 = snap43
			d10 = snap44
			d11 = snap45
			d28 = snap46
			d29 = snap47
			d30 = snap48
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps33)
			}
			return result
			ctx.FreeDesc(&d29)
			return result
			}
			bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d50 := ps.PhiValues[0]
					ctx.EnsureDesc(&d50)
					ctx.EmitStoreToStack(d50, int32(bbs[3].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d51 := ps.PhiValues[1]
					ctx.EnsureDesc(&d51)
					ctx.EmitStoreToStack(d51, int32(bbs[3].PhiBase)+int32(16))
				}
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
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != LocNone {
				d51 = ps.OverlayValues[51]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d1 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d52 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d52)
			var d53 JITValueDesc
			if d1.Loc == LocImm && d52.Loc == LocImm {
				d53 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d52.Imm.Int())}
			} else if d52.Loc == LocImm {
				r6 := ctx.AllocRegExcept(d1.Reg)
				if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d1.Reg, int32(d52.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d52.Imm.Int()))
					ctx.EmitCmpInt64(d1.Reg, RegR11)
				}
				ctx.EmitSetcc(r6, CcL)
				d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d53)
			} else if d1.Loc == LocImm {
				r7 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d52.Reg)
				ctx.EmitSetcc(r7, CcL)
				d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d53)
			} else {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitCmpInt64(d1.Reg, d52.Reg)
				ctx.EmitSetcc(r8, CcL)
				d53 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d53)
			}
			ctx.FreeDesc(&d52)
			d54 = d53
			ctx.EnsureDesc(&d54)
			if d54.Loc != LocImm && d54.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d54.Loc == LocImm {
				if d54.Imm.Bool() {
			ps55 := PhiState{General: ps.General}
			ps55.OverlayValues = make([]JITValueDesc, 55)
			ps55.OverlayValues[0] = d0
			ps55.OverlayValues[1] = d1
			ps55.OverlayValues[2] = d2
			ps55.OverlayValues[3] = d3
			ps55.OverlayValues[5] = d5
			ps55.OverlayValues[6] = d6
			ps55.OverlayValues[7] = d7
			ps55.OverlayValues[8] = d8
			ps55.OverlayValues[9] = d9
			ps55.OverlayValues[10] = d10
			ps55.OverlayValues[11] = d11
			ps55.OverlayValues[28] = d28
			ps55.OverlayValues[29] = d29
			ps55.OverlayValues[30] = d30
			ps55.OverlayValues[50] = d50
			ps55.OverlayValues[51] = d51
			ps55.OverlayValues[52] = d52
			ps55.OverlayValues[53] = d53
			ps55.OverlayValues[54] = d54
					return bbs[1].RenderPS(ps55)
				}
			ps56 := PhiState{General: ps.General}
			ps56.OverlayValues = make([]JITValueDesc, 55)
			ps56.OverlayValues[0] = d0
			ps56.OverlayValues[1] = d1
			ps56.OverlayValues[2] = d2
			ps56.OverlayValues[3] = d3
			ps56.OverlayValues[5] = d5
			ps56.OverlayValues[6] = d6
			ps56.OverlayValues[7] = d7
			ps56.OverlayValues[8] = d8
			ps56.OverlayValues[9] = d9
			ps56.OverlayValues[10] = d10
			ps56.OverlayValues[11] = d11
			ps56.OverlayValues[28] = d28
			ps56.OverlayValues[29] = d29
			ps56.OverlayValues[30] = d30
			ps56.OverlayValues[50] = d50
			ps56.OverlayValues[51] = d51
			ps56.OverlayValues[52] = d52
			ps56.OverlayValues[53] = d53
			ps56.OverlayValues[54] = d54
				return bbs[2].RenderPS(ps56)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d57 := ps.PhiValues[0]
					ctx.EnsureDesc(&d57)
					ctx.EmitStoreToStack(d57, int32(bbs[3].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d58 := ps.PhiValues[1]
					ctx.EnsureDesc(&d58)
					ctx.EmitStoreToStack(d58, int32(bbs[3].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl19 := ctx.ReserveLabel()
			lbl20 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d54.Reg, 0)
			ctx.EmitJcc(CcNE, lbl19)
			ctx.EmitJmp(lbl20)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl3)
			ps59 := PhiState{General: true}
			ps59.OverlayValues = make([]JITValueDesc, 59)
			ps59.OverlayValues[0] = d0
			ps59.OverlayValues[1] = d1
			ps59.OverlayValues[2] = d2
			ps59.OverlayValues[3] = d3
			ps59.OverlayValues[5] = d5
			ps59.OverlayValues[6] = d6
			ps59.OverlayValues[7] = d7
			ps59.OverlayValues[8] = d8
			ps59.OverlayValues[9] = d9
			ps59.OverlayValues[10] = d10
			ps59.OverlayValues[11] = d11
			ps59.OverlayValues[28] = d28
			ps59.OverlayValues[29] = d29
			ps59.OverlayValues[30] = d30
			ps59.OverlayValues[50] = d50
			ps59.OverlayValues[51] = d51
			ps59.OverlayValues[52] = d52
			ps59.OverlayValues[53] = d53
			ps59.OverlayValues[54] = d54
			ps59.OverlayValues[57] = d57
			ps59.OverlayValues[58] = d58
			ps60 := PhiState{General: true}
			ps60.OverlayValues = make([]JITValueDesc, 59)
			ps60.OverlayValues[0] = d0
			ps60.OverlayValues[1] = d1
			ps60.OverlayValues[2] = d2
			ps60.OverlayValues[3] = d3
			ps60.OverlayValues[5] = d5
			ps60.OverlayValues[6] = d6
			ps60.OverlayValues[7] = d7
			ps60.OverlayValues[8] = d8
			ps60.OverlayValues[9] = d9
			ps60.OverlayValues[10] = d10
			ps60.OverlayValues[11] = d11
			ps60.OverlayValues[28] = d28
			ps60.OverlayValues[29] = d29
			ps60.OverlayValues[30] = d30
			ps60.OverlayValues[50] = d50
			ps60.OverlayValues[51] = d51
			ps60.OverlayValues[52] = d52
			ps60.OverlayValues[53] = d53
			ps60.OverlayValues[54] = d54
			ps60.OverlayValues[57] = d57
			ps60.OverlayValues[58] = d58
			snap61 := d0
			snap62 := d1
			snap63 := d2
			snap64 := d3
			snap65 := d5
			snap66 := d6
			snap67 := d7
			snap68 := d8
			snap69 := d9
			snap70 := d10
			snap71 := d11
			snap72 := d28
			snap73 := d29
			snap74 := d30
			snap75 := d50
			snap76 := d51
			snap77 := d52
			snap78 := d53
			snap79 := d54
			snap80 := d57
			snap81 := d58
			alloc82 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps60)
			}
			ctx.RestoreAllocState(alloc82)
			d0 = snap61
			d1 = snap62
			d2 = snap63
			d3 = snap64
			d5 = snap65
			d6 = snap66
			d7 = snap67
			d8 = snap68
			d9 = snap69
			d10 = snap70
			d11 = snap71
			d28 = snap72
			d29 = snap73
			d30 = snap74
			d50 = snap75
			d51 = snap76
			d52 = snap77
			d53 = snap78
			d54 = snap79
			d57 = snap80
			d58 = snap81
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps59)
			}
			return result
			ctx.FreeDesc(&d53)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
			}
			ctx.ReclaimUntrackedRegs()
			var d83 JITValueDesc
			if d7.Loc == LocImm {
				d83 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d7.Imm.Int())}
			} else if d7.Type == tagInt && d7.Loc == LocRegPair {
				ctx.FreeReg(d7.Reg)
				d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg2}
				ctx.BindReg(d7.Reg2, &d83)
				ctx.BindReg(d7.Reg2, &d83)
			} else if d7.Type == tagInt && d7.Loc == LocReg {
				d83 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d7.Reg}
				ctx.BindReg(d7.Reg, &d83)
				ctx.BindReg(d7.Reg, &d83)
			} else {
				d83 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d7}, 1)
				d83.Type = tagInt
				ctx.BindReg(d83.Reg, &d83)
			}
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d83)
			var d84 JITValueDesc
			if d0.Loc == LocImm && d83.Loc == LocImm {
				d84 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d0.Imm.Int() + d83.Imm.Int())}
			} else if d83.Loc == LocImm && d83.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(r9, d0.Reg)
				d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
				ctx.BindReg(r9, &d84)
			} else if d0.Loc == LocImm && d0.Imm.Int() == 0 {
				d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d83.Reg}
				ctx.BindReg(d83.Reg, &d84)
			} else if d0.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(scratch, d83.Reg)
				d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d84)
			} else if d83.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d83.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d83.Imm.Int()))
					ctx.EmitAddInt64(scratch, RegR11)
				}
				d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d84)
			} else {
				r10 := ctx.AllocRegExcept(d0.Reg, d83.Reg)
				ctx.EmitMovRegReg(r10, d0.Reg)
				ctx.EmitAddInt64(r10, d83.Reg)
				d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
				ctx.BindReg(r10, &d84)
			}
			if d84.Loc == LocReg && d0.Loc == LocReg && d84.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = LocNone
			}
			ctx.FreeDesc(&d83)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d85 JITValueDesc
			if d1.Loc == LocImm {
				d85 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d85 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			}
			if d85.Loc == LocReg && d1.Loc == LocReg && d85.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.EnsureDesc(&d84)
			if d84.Loc == LocReg {
				ctx.ProtectReg(d84.Reg)
			} else if d84.Loc == LocRegPair {
				ctx.ProtectReg(d84.Reg)
				ctx.ProtectReg(d84.Reg2)
			}
			ctx.EnsureDesc(&d85)
			if d85.Loc == LocReg {
				ctx.ProtectReg(d85.Reg)
			} else if d85.Loc == LocRegPair {
				ctx.ProtectReg(d85.Reg)
				ctx.ProtectReg(d85.Reg2)
			}
			d86 = d84
			if d86.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d86)
			ctx.EmitStoreToStack(d86, int32(bbs[3].PhiBase)+int32(0))
			d87 = d85
			if d87.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d87)
			ctx.EmitStoreToStack(d87, int32(bbs[3].PhiBase)+int32(16))
			if d84.Loc == LocReg {
				ctx.UnprotectReg(d84.Reg)
			} else if d84.Loc == LocRegPair {
				ctx.UnprotectReg(d84.Reg)
				ctx.UnprotectReg(d84.Reg2)
			}
			if d85.Loc == LocReg {
				ctx.UnprotectReg(d85.Reg)
			} else if d85.Loc == LocRegPair {
				ctx.UnprotectReg(d85.Reg)
				ctx.UnprotectReg(d85.Reg2)
			}
			ps88 := PhiState{General: ps.General}
			ps88.OverlayValues = make([]JITValueDesc, 88)
			ps88.OverlayValues[0] = d0
			ps88.OverlayValues[1] = d1
			ps88.OverlayValues[2] = d2
			ps88.OverlayValues[3] = d3
			ps88.OverlayValues[5] = d5
			ps88.OverlayValues[6] = d6
			ps88.OverlayValues[7] = d7
			ps88.OverlayValues[8] = d8
			ps88.OverlayValues[9] = d9
			ps88.OverlayValues[10] = d10
			ps88.OverlayValues[11] = d11
			ps88.OverlayValues[28] = d28
			ps88.OverlayValues[29] = d29
			ps88.OverlayValues[30] = d30
			ps88.OverlayValues[50] = d50
			ps88.OverlayValues[51] = d51
			ps88.OverlayValues[52] = d52
			ps88.OverlayValues[53] = d53
			ps88.OverlayValues[54] = d54
			ps88.OverlayValues[57] = d57
			ps88.OverlayValues[58] = d58
			ps88.OverlayValues[83] = d83
			ps88.OverlayValues[84] = d84
			ps88.OverlayValues[85] = d85
			ps88.OverlayValues[86] = d86
			ps88.OverlayValues[87] = d87
			ps88.PhiValues = make([]JITValueDesc, 2)
			d89 = d84
			ps88.PhiValues[0] = d89
			d90 = d85
			ps88.PhiValues[1] = d90
			if ps88.General && bbs[3].Rendered {
				ctx.EmitJmp(lbl4)
				return result
			}
			return bbs[3].RenderPS(ps88)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
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
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.EmitMakeInt(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagInt
			ctx.EmitJmp(lbl0)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
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
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
				d90 = ps.OverlayValues[90]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d91 JITValueDesc
			if d0.Loc == LocImm {
				d91 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d0.Imm.Int()))}
			} else {
				ctx.EmitCvtInt64ToFloat64(RegX0, d0.Reg)
				d91 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d0.Reg}
				ctx.BindReg(d0.Reg, &d91)
			}
			ctx.EnsureDesc(&d1)
			if d1.Loc == LocReg {
				ctx.ProtectReg(d1.Reg)
			} else if d1.Loc == LocRegPair {
				ctx.ProtectReg(d1.Reg)
				ctx.ProtectReg(d1.Reg2)
			}
			ctx.EnsureDesc(&d91)
			if d91.Loc == LocReg {
				ctx.ProtectReg(d91.Reg)
			} else if d91.Loc == LocRegPair {
				ctx.ProtectReg(d91.Reg)
				ctx.ProtectReg(d91.Reg2)
			}
			d92 = d1
			if d92.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d92)
			ctx.EmitStoreToStack(d92, int32(bbs[9].PhiBase)+int32(0))
			d93 = d91
			if d93.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d93)
			ctx.EmitStoreToStack(d93, int32(bbs[9].PhiBase)+int32(16))
			if d1.Loc == LocReg {
				ctx.UnprotectReg(d1.Reg)
			} else if d1.Loc == LocRegPair {
				ctx.UnprotectReg(d1.Reg)
				ctx.UnprotectReg(d1.Reg2)
			}
			if d91.Loc == LocReg {
				ctx.UnprotectReg(d91.Reg)
			} else if d91.Loc == LocRegPair {
				ctx.UnprotectReg(d91.Reg)
				ctx.UnprotectReg(d91.Reg2)
			}
			ps94 := PhiState{General: ps.General}
			ps94.OverlayValues = make([]JITValueDesc, 94)
			ps94.OverlayValues[0] = d0
			ps94.OverlayValues[1] = d1
			ps94.OverlayValues[2] = d2
			ps94.OverlayValues[3] = d3
			ps94.OverlayValues[5] = d5
			ps94.OverlayValues[6] = d6
			ps94.OverlayValues[7] = d7
			ps94.OverlayValues[8] = d8
			ps94.OverlayValues[9] = d9
			ps94.OverlayValues[10] = d10
			ps94.OverlayValues[11] = d11
			ps94.OverlayValues[28] = d28
			ps94.OverlayValues[29] = d29
			ps94.OverlayValues[30] = d30
			ps94.OverlayValues[50] = d50
			ps94.OverlayValues[51] = d51
			ps94.OverlayValues[52] = d52
			ps94.OverlayValues[53] = d53
			ps94.OverlayValues[54] = d54
			ps94.OverlayValues[57] = d57
			ps94.OverlayValues[58] = d58
			ps94.OverlayValues[83] = d83
			ps94.OverlayValues[84] = d84
			ps94.OverlayValues[85] = d85
			ps94.OverlayValues[86] = d86
			ps94.OverlayValues[87] = d87
			ps94.OverlayValues[89] = d89
			ps94.OverlayValues[90] = d90
			ps94.OverlayValues[91] = d91
			ps94.OverlayValues[92] = d92
			ps94.OverlayValues[93] = d93
			ps94.PhiValues = make([]JITValueDesc, 2)
			d95 = d1
			ps94.PhiValues[0] = d95
			d96 = d91
			ps94.PhiValues[1] = d96
			if ps94.General && bbs[9].Rendered {
				ctx.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps94)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
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
			if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
				d96 = ps.OverlayValues[96]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d97 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d97 = args[idx]
				d97.ID = 0
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
				lbl21 := ctx.ReserveLabel()
				lbl22 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl22)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r11, ai.Reg)
						ctx.EmitMovRegReg(r12, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r11, tmp.Reg)
						ctx.EmitMovRegReg(r12, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r11, Reg2: r12}
						ctx.BindReg(r11, &pair)
						ctx.BindReg(r12, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r11, uint64(ptrWord))
							ctx.EmitMovRegImm64(r12, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl22)
				d98 := JITValueDesc{Loc: LocRegPair, Reg: r11, Reg2: r12}
				ctx.BindReg(r11, &d98)
				ctx.BindReg(r12, &d98)
				ctx.BindReg(r11, &d98)
				ctx.BindReg(r12, &d98)
				ctx.EmitMakeNil(d98)
				ctx.MarkLabel(lbl21)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d97 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r11, Reg2: r12}
				ctx.BindReg(r11, &d97)
				ctx.BindReg(r12, &d97)
			}
			d100 = d97
			d100.ID = 0
			d99 = ctx.EmitTagEqualsBorrowed(&d100, tagNil, JITValueDesc{Loc: LocAny})
			d101 = d99
			ctx.EnsureDesc(&d101)
			if d101.Loc != LocImm && d101.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d101.Loc == LocImm {
				if d101.Imm.Bool() {
			ps102 := PhiState{General: ps.General}
			ps102.OverlayValues = make([]JITValueDesc, 102)
			ps102.OverlayValues[0] = d0
			ps102.OverlayValues[1] = d1
			ps102.OverlayValues[2] = d2
			ps102.OverlayValues[3] = d3
			ps102.OverlayValues[5] = d5
			ps102.OverlayValues[6] = d6
			ps102.OverlayValues[7] = d7
			ps102.OverlayValues[8] = d8
			ps102.OverlayValues[9] = d9
			ps102.OverlayValues[10] = d10
			ps102.OverlayValues[11] = d11
			ps102.OverlayValues[28] = d28
			ps102.OverlayValues[29] = d29
			ps102.OverlayValues[30] = d30
			ps102.OverlayValues[50] = d50
			ps102.OverlayValues[51] = d51
			ps102.OverlayValues[52] = d52
			ps102.OverlayValues[53] = d53
			ps102.OverlayValues[54] = d54
			ps102.OverlayValues[57] = d57
			ps102.OverlayValues[58] = d58
			ps102.OverlayValues[83] = d83
			ps102.OverlayValues[84] = d84
			ps102.OverlayValues[85] = d85
			ps102.OverlayValues[86] = d86
			ps102.OverlayValues[87] = d87
			ps102.OverlayValues[89] = d89
			ps102.OverlayValues[90] = d90
			ps102.OverlayValues[91] = d91
			ps102.OverlayValues[92] = d92
			ps102.OverlayValues[93] = d93
			ps102.OverlayValues[95] = d95
			ps102.OverlayValues[96] = d96
			ps102.OverlayValues[97] = d97
			ps102.OverlayValues[98] = d98
			ps102.OverlayValues[99] = d99
			ps102.OverlayValues[100] = d100
			ps102.OverlayValues[101] = d101
					return bbs[10].RenderPS(ps102)
				}
			ps103 := PhiState{General: ps.General}
			ps103.OverlayValues = make([]JITValueDesc, 102)
			ps103.OverlayValues[0] = d0
			ps103.OverlayValues[1] = d1
			ps103.OverlayValues[2] = d2
			ps103.OverlayValues[3] = d3
			ps103.OverlayValues[5] = d5
			ps103.OverlayValues[6] = d6
			ps103.OverlayValues[7] = d7
			ps103.OverlayValues[8] = d8
			ps103.OverlayValues[9] = d9
			ps103.OverlayValues[10] = d10
			ps103.OverlayValues[11] = d11
			ps103.OverlayValues[28] = d28
			ps103.OverlayValues[29] = d29
			ps103.OverlayValues[30] = d30
			ps103.OverlayValues[50] = d50
			ps103.OverlayValues[51] = d51
			ps103.OverlayValues[52] = d52
			ps103.OverlayValues[53] = d53
			ps103.OverlayValues[54] = d54
			ps103.OverlayValues[57] = d57
			ps103.OverlayValues[58] = d58
			ps103.OverlayValues[83] = d83
			ps103.OverlayValues[84] = d84
			ps103.OverlayValues[85] = d85
			ps103.OverlayValues[86] = d86
			ps103.OverlayValues[87] = d87
			ps103.OverlayValues[89] = d89
			ps103.OverlayValues[90] = d90
			ps103.OverlayValues[91] = d91
			ps103.OverlayValues[92] = d92
			ps103.OverlayValues[93] = d93
			ps103.OverlayValues[95] = d95
			ps103.OverlayValues[96] = d96
			ps103.OverlayValues[97] = d97
			ps103.OverlayValues[98] = d98
			ps103.OverlayValues[99] = d99
			ps103.OverlayValues[100] = d100
			ps103.OverlayValues[101] = d101
				return bbs[11].RenderPS(ps103)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl23 := ctx.ReserveLabel()
			lbl24 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d101.Reg, 0)
			ctx.EmitJcc(CcNE, lbl23)
			ctx.EmitJmp(lbl24)
			ctx.MarkLabel(lbl23)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl24)
			ctx.EmitJmp(lbl12)
			ps104 := PhiState{General: true}
			ps104.OverlayValues = make([]JITValueDesc, 102)
			ps104.OverlayValues[0] = d0
			ps104.OverlayValues[1] = d1
			ps104.OverlayValues[2] = d2
			ps104.OverlayValues[3] = d3
			ps104.OverlayValues[5] = d5
			ps104.OverlayValues[6] = d6
			ps104.OverlayValues[7] = d7
			ps104.OverlayValues[8] = d8
			ps104.OverlayValues[9] = d9
			ps104.OverlayValues[10] = d10
			ps104.OverlayValues[11] = d11
			ps104.OverlayValues[28] = d28
			ps104.OverlayValues[29] = d29
			ps104.OverlayValues[30] = d30
			ps104.OverlayValues[50] = d50
			ps104.OverlayValues[51] = d51
			ps104.OverlayValues[52] = d52
			ps104.OverlayValues[53] = d53
			ps104.OverlayValues[54] = d54
			ps104.OverlayValues[57] = d57
			ps104.OverlayValues[58] = d58
			ps104.OverlayValues[83] = d83
			ps104.OverlayValues[84] = d84
			ps104.OverlayValues[85] = d85
			ps104.OverlayValues[86] = d86
			ps104.OverlayValues[87] = d87
			ps104.OverlayValues[89] = d89
			ps104.OverlayValues[90] = d90
			ps104.OverlayValues[91] = d91
			ps104.OverlayValues[92] = d92
			ps104.OverlayValues[93] = d93
			ps104.OverlayValues[95] = d95
			ps104.OverlayValues[96] = d96
			ps104.OverlayValues[97] = d97
			ps104.OverlayValues[98] = d98
			ps104.OverlayValues[99] = d99
			ps104.OverlayValues[100] = d100
			ps104.OverlayValues[101] = d101
			ps105 := PhiState{General: true}
			ps105.OverlayValues = make([]JITValueDesc, 102)
			ps105.OverlayValues[0] = d0
			ps105.OverlayValues[1] = d1
			ps105.OverlayValues[2] = d2
			ps105.OverlayValues[3] = d3
			ps105.OverlayValues[5] = d5
			ps105.OverlayValues[6] = d6
			ps105.OverlayValues[7] = d7
			ps105.OverlayValues[8] = d8
			ps105.OverlayValues[9] = d9
			ps105.OverlayValues[10] = d10
			ps105.OverlayValues[11] = d11
			ps105.OverlayValues[28] = d28
			ps105.OverlayValues[29] = d29
			ps105.OverlayValues[30] = d30
			ps105.OverlayValues[50] = d50
			ps105.OverlayValues[51] = d51
			ps105.OverlayValues[52] = d52
			ps105.OverlayValues[53] = d53
			ps105.OverlayValues[54] = d54
			ps105.OverlayValues[57] = d57
			ps105.OverlayValues[58] = d58
			ps105.OverlayValues[83] = d83
			ps105.OverlayValues[84] = d84
			ps105.OverlayValues[85] = d85
			ps105.OverlayValues[86] = d86
			ps105.OverlayValues[87] = d87
			ps105.OverlayValues[89] = d89
			ps105.OverlayValues[90] = d90
			ps105.OverlayValues[91] = d91
			ps105.OverlayValues[92] = d92
			ps105.OverlayValues[93] = d93
			ps105.OverlayValues[95] = d95
			ps105.OverlayValues[96] = d96
			ps105.OverlayValues[97] = d97
			ps105.OverlayValues[98] = d98
			ps105.OverlayValues[99] = d99
			ps105.OverlayValues[100] = d100
			ps105.OverlayValues[101] = d101
			snap106 := d0
			snap107 := d1
			snap108 := d2
			snap109 := d3
			snap110 := d5
			snap111 := d6
			snap112 := d7
			snap113 := d8
			snap114 := d9
			snap115 := d10
			snap116 := d11
			snap117 := d28
			snap118 := d29
			snap119 := d30
			snap120 := d50
			snap121 := d51
			snap122 := d52
			snap123 := d53
			snap124 := d54
			snap125 := d57
			snap126 := d58
			snap127 := d83
			snap128 := d84
			snap129 := d85
			snap130 := d86
			snap131 := d87
			snap132 := d89
			snap133 := d90
			snap134 := d91
			snap135 := d92
			snap136 := d93
			snap137 := d95
			snap138 := d96
			snap139 := d97
			snap140 := d98
			snap141 := d99
			snap142 := d100
			snap143 := d101
			alloc144 := ctx.SnapshotAllocState()
			if !bbs[11].Rendered {
				bbs[11].RenderPS(ps105)
			}
			ctx.RestoreAllocState(alloc144)
			d0 = snap106
			d1 = snap107
			d2 = snap108
			d3 = snap109
			d5 = snap110
			d6 = snap111
			d7 = snap112
			d8 = snap113
			d9 = snap114
			d10 = snap115
			d11 = snap116
			d28 = snap117
			d29 = snap118
			d30 = snap119
			d50 = snap120
			d51 = snap121
			d52 = snap122
			d53 = snap123
			d54 = snap124
			d57 = snap125
			d58 = snap126
			d83 = snap127
			d84 = snap128
			d85 = snap129
			d86 = snap130
			d87 = snap131
			d89 = snap132
			d90 = snap133
			d91 = snap134
			d92 = snap135
			d93 = snap136
			d95 = snap137
			d96 = snap138
			d97 = snap139
			d98 = snap140
			d99 = snap141
			d100 = snap142
			d101 = snap143
			if !bbs[10].Rendered {
				return bbs[10].RenderPS(ps104)
			}
			return result
			ctx.FreeDesc(&d99)
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
					ctx.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			ctx.EmitMakeFloat(result, d3)
			if d3.Loc == LocReg { ctx.FreeReg(d3.Reg) }
			result.Type = tagFloat
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d145 := ps.PhiValues[0]
					ctx.EnsureDesc(&d145)
					ctx.EmitStoreToStack(d145, int32(bbs[9].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d146 := ps.PhiValues[1]
					ctx.EnsureDesc(&d146)
					ctx.EmitStoreToStack(d146, int32(bbs[9].PhiBase)+int32(16))
				}
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
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
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
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
				d146 = ps.OverlayValues[146]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d2 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d3 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d147 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d147)
			var d148 JITValueDesc
			if d2.Loc == LocImm && d147.Loc == LocImm {
				d148 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d147.Imm.Int())}
			} else if d147.Loc == LocImm {
				r13 := ctx.AllocRegExcept(d2.Reg)
				if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d2.Reg, int32(d147.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d147.Imm.Int()))
					ctx.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.EmitSetcc(r13, CcL)
				d148 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
				ctx.BindReg(r13, &d148)
			} else if d2.Loc == LocImm {
				r14 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d147.Reg)
				ctx.EmitSetcc(r14, CcL)
				d148 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
				ctx.BindReg(r14, &d148)
			} else {
				r15 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitCmpInt64(d2.Reg, d147.Reg)
				ctx.EmitSetcc(r15, CcL)
				d148 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d148)
			}
			ctx.FreeDesc(&d147)
			d149 = d148
			ctx.EnsureDesc(&d149)
			if d149.Loc != LocImm && d149.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d149.Loc == LocImm {
				if d149.Imm.Bool() {
			ps150 := PhiState{General: ps.General}
			ps150.OverlayValues = make([]JITValueDesc, 150)
			ps150.OverlayValues[0] = d0
			ps150.OverlayValues[1] = d1
			ps150.OverlayValues[2] = d2
			ps150.OverlayValues[3] = d3
			ps150.OverlayValues[5] = d5
			ps150.OverlayValues[6] = d6
			ps150.OverlayValues[7] = d7
			ps150.OverlayValues[8] = d8
			ps150.OverlayValues[9] = d9
			ps150.OverlayValues[10] = d10
			ps150.OverlayValues[11] = d11
			ps150.OverlayValues[28] = d28
			ps150.OverlayValues[29] = d29
			ps150.OverlayValues[30] = d30
			ps150.OverlayValues[50] = d50
			ps150.OverlayValues[51] = d51
			ps150.OverlayValues[52] = d52
			ps150.OverlayValues[53] = d53
			ps150.OverlayValues[54] = d54
			ps150.OverlayValues[57] = d57
			ps150.OverlayValues[58] = d58
			ps150.OverlayValues[83] = d83
			ps150.OverlayValues[84] = d84
			ps150.OverlayValues[85] = d85
			ps150.OverlayValues[86] = d86
			ps150.OverlayValues[87] = d87
			ps150.OverlayValues[89] = d89
			ps150.OverlayValues[90] = d90
			ps150.OverlayValues[91] = d91
			ps150.OverlayValues[92] = d92
			ps150.OverlayValues[93] = d93
			ps150.OverlayValues[95] = d95
			ps150.OverlayValues[96] = d96
			ps150.OverlayValues[97] = d97
			ps150.OverlayValues[98] = d98
			ps150.OverlayValues[99] = d99
			ps150.OverlayValues[100] = d100
			ps150.OverlayValues[101] = d101
			ps150.OverlayValues[145] = d145
			ps150.OverlayValues[146] = d146
			ps150.OverlayValues[147] = d147
			ps150.OverlayValues[148] = d148
			ps150.OverlayValues[149] = d149
					return bbs[7].RenderPS(ps150)
				}
			ps151 := PhiState{General: ps.General}
			ps151.OverlayValues = make([]JITValueDesc, 150)
			ps151.OverlayValues[0] = d0
			ps151.OverlayValues[1] = d1
			ps151.OverlayValues[2] = d2
			ps151.OverlayValues[3] = d3
			ps151.OverlayValues[5] = d5
			ps151.OverlayValues[6] = d6
			ps151.OverlayValues[7] = d7
			ps151.OverlayValues[8] = d8
			ps151.OverlayValues[9] = d9
			ps151.OverlayValues[10] = d10
			ps151.OverlayValues[11] = d11
			ps151.OverlayValues[28] = d28
			ps151.OverlayValues[29] = d29
			ps151.OverlayValues[30] = d30
			ps151.OverlayValues[50] = d50
			ps151.OverlayValues[51] = d51
			ps151.OverlayValues[52] = d52
			ps151.OverlayValues[53] = d53
			ps151.OverlayValues[54] = d54
			ps151.OverlayValues[57] = d57
			ps151.OverlayValues[58] = d58
			ps151.OverlayValues[83] = d83
			ps151.OverlayValues[84] = d84
			ps151.OverlayValues[85] = d85
			ps151.OverlayValues[86] = d86
			ps151.OverlayValues[87] = d87
			ps151.OverlayValues[89] = d89
			ps151.OverlayValues[90] = d90
			ps151.OverlayValues[91] = d91
			ps151.OverlayValues[92] = d92
			ps151.OverlayValues[93] = d93
			ps151.OverlayValues[95] = d95
			ps151.OverlayValues[96] = d96
			ps151.OverlayValues[97] = d97
			ps151.OverlayValues[98] = d98
			ps151.OverlayValues[99] = d99
			ps151.OverlayValues[100] = d100
			ps151.OverlayValues[101] = d101
			ps151.OverlayValues[145] = d145
			ps151.OverlayValues[146] = d146
			ps151.OverlayValues[147] = d147
			ps151.OverlayValues[148] = d148
			ps151.OverlayValues[149] = d149
				return bbs[8].RenderPS(ps151)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d152 := ps.PhiValues[0]
					ctx.EnsureDesc(&d152)
					ctx.EmitStoreToStack(d152, int32(bbs[9].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d153 := ps.PhiValues[1]
					ctx.EnsureDesc(&d153)
					ctx.EmitStoreToStack(d153, int32(bbs[9].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[9].RenderPS(ps)
			}
			lbl25 := ctx.ReserveLabel()
			lbl26 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d149.Reg, 0)
			ctx.EmitJcc(CcNE, lbl25)
			ctx.EmitJmp(lbl26)
			ctx.MarkLabel(lbl25)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl26)
			ctx.EmitJmp(lbl9)
			ps154 := PhiState{General: true}
			ps154.OverlayValues = make([]JITValueDesc, 154)
			ps154.OverlayValues[0] = d0
			ps154.OverlayValues[1] = d1
			ps154.OverlayValues[2] = d2
			ps154.OverlayValues[3] = d3
			ps154.OverlayValues[5] = d5
			ps154.OverlayValues[6] = d6
			ps154.OverlayValues[7] = d7
			ps154.OverlayValues[8] = d8
			ps154.OverlayValues[9] = d9
			ps154.OverlayValues[10] = d10
			ps154.OverlayValues[11] = d11
			ps154.OverlayValues[28] = d28
			ps154.OverlayValues[29] = d29
			ps154.OverlayValues[30] = d30
			ps154.OverlayValues[50] = d50
			ps154.OverlayValues[51] = d51
			ps154.OverlayValues[52] = d52
			ps154.OverlayValues[53] = d53
			ps154.OverlayValues[54] = d54
			ps154.OverlayValues[57] = d57
			ps154.OverlayValues[58] = d58
			ps154.OverlayValues[83] = d83
			ps154.OverlayValues[84] = d84
			ps154.OverlayValues[85] = d85
			ps154.OverlayValues[86] = d86
			ps154.OverlayValues[87] = d87
			ps154.OverlayValues[89] = d89
			ps154.OverlayValues[90] = d90
			ps154.OverlayValues[91] = d91
			ps154.OverlayValues[92] = d92
			ps154.OverlayValues[93] = d93
			ps154.OverlayValues[95] = d95
			ps154.OverlayValues[96] = d96
			ps154.OverlayValues[97] = d97
			ps154.OverlayValues[98] = d98
			ps154.OverlayValues[99] = d99
			ps154.OverlayValues[100] = d100
			ps154.OverlayValues[101] = d101
			ps154.OverlayValues[145] = d145
			ps154.OverlayValues[146] = d146
			ps154.OverlayValues[147] = d147
			ps154.OverlayValues[148] = d148
			ps154.OverlayValues[149] = d149
			ps154.OverlayValues[152] = d152
			ps154.OverlayValues[153] = d153
			ps155 := PhiState{General: true}
			ps155.OverlayValues = make([]JITValueDesc, 154)
			ps155.OverlayValues[0] = d0
			ps155.OverlayValues[1] = d1
			ps155.OverlayValues[2] = d2
			ps155.OverlayValues[3] = d3
			ps155.OverlayValues[5] = d5
			ps155.OverlayValues[6] = d6
			ps155.OverlayValues[7] = d7
			ps155.OverlayValues[8] = d8
			ps155.OverlayValues[9] = d9
			ps155.OverlayValues[10] = d10
			ps155.OverlayValues[11] = d11
			ps155.OverlayValues[28] = d28
			ps155.OverlayValues[29] = d29
			ps155.OverlayValues[30] = d30
			ps155.OverlayValues[50] = d50
			ps155.OverlayValues[51] = d51
			ps155.OverlayValues[52] = d52
			ps155.OverlayValues[53] = d53
			ps155.OverlayValues[54] = d54
			ps155.OverlayValues[57] = d57
			ps155.OverlayValues[58] = d58
			ps155.OverlayValues[83] = d83
			ps155.OverlayValues[84] = d84
			ps155.OverlayValues[85] = d85
			ps155.OverlayValues[86] = d86
			ps155.OverlayValues[87] = d87
			ps155.OverlayValues[89] = d89
			ps155.OverlayValues[90] = d90
			ps155.OverlayValues[91] = d91
			ps155.OverlayValues[92] = d92
			ps155.OverlayValues[93] = d93
			ps155.OverlayValues[95] = d95
			ps155.OverlayValues[96] = d96
			ps155.OverlayValues[97] = d97
			ps155.OverlayValues[98] = d98
			ps155.OverlayValues[99] = d99
			ps155.OverlayValues[100] = d100
			ps155.OverlayValues[101] = d101
			ps155.OverlayValues[145] = d145
			ps155.OverlayValues[146] = d146
			ps155.OverlayValues[147] = d147
			ps155.OverlayValues[148] = d148
			ps155.OverlayValues[149] = d149
			ps155.OverlayValues[152] = d152
			ps155.OverlayValues[153] = d153
			snap156 := d0
			snap157 := d1
			snap158 := d2
			snap159 := d3
			snap160 := d5
			snap161 := d6
			snap162 := d7
			snap163 := d8
			snap164 := d9
			snap165 := d10
			snap166 := d11
			snap167 := d28
			snap168 := d29
			snap169 := d30
			snap170 := d50
			snap171 := d51
			snap172 := d52
			snap173 := d53
			snap174 := d54
			snap175 := d57
			snap176 := d58
			snap177 := d83
			snap178 := d84
			snap179 := d85
			snap180 := d86
			snap181 := d87
			snap182 := d89
			snap183 := d90
			snap184 := d91
			snap185 := d92
			snap186 := d93
			snap187 := d95
			snap188 := d96
			snap189 := d97
			snap190 := d98
			snap191 := d99
			snap192 := d100
			snap193 := d101
			snap194 := d145
			snap195 := d146
			snap196 := d147
			snap197 := d148
			snap198 := d149
			snap199 := d152
			snap200 := d153
			alloc201 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps155)
			}
			ctx.RestoreAllocState(alloc201)
			d0 = snap156
			d1 = snap157
			d2 = snap158
			d3 = snap159
			d5 = snap160
			d6 = snap161
			d7 = snap162
			d8 = snap163
			d9 = snap164
			d10 = snap165
			d11 = snap166
			d28 = snap167
			d29 = snap168
			d30 = snap169
			d50 = snap170
			d51 = snap171
			d52 = snap172
			d53 = snap173
			d54 = snap174
			d57 = snap175
			d58 = snap176
			d83 = snap177
			d84 = snap178
			d85 = snap179
			d86 = snap180
			d87 = snap181
			d89 = snap182
			d90 = snap183
			d91 = snap184
			d92 = snap185
			d93 = snap186
			d95 = snap187
			d96 = snap188
			d97 = snap189
			d98 = snap190
			d99 = snap191
			d100 = snap192
			d101 = snap193
			d145 = snap194
			d146 = snap195
			d147 = snap196
			d148 = snap197
			d149 = snap198
			d152 = snap199
			d153 = snap200
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps154)
			}
			return result
			ctx.FreeDesc(&d148)
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
					ctx.EmitJmp(lbl11)
					return result
				}
				bbs[10].Rendered = true
				bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_10 = bbs[10].Address
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
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
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitMakeNil(result)
			result.Type = tagNil
			ctx.EmitJmp(lbl0)
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
					ctx.EmitJmp(lbl12)
					return result
				}
				bbs[11].Rendered = true
				bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_11 = bbs[11].Address
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
				d30 = ps.OverlayValues[30]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != LocNone {
				d58 = ps.OverlayValues[58]
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
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != LocNone {
				d146 = ps.OverlayValues[146]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			ctx.ReclaimUntrackedRegs()
			var d202 JITValueDesc
			if d97.Loc == LocImm {
				d202 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d97.Imm.Float())}
			} else if d97.Type == tagFloat && d97.Loc == LocReg {
				d202 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d97.Reg}
				ctx.BindReg(d97.Reg, &d202)
				ctx.BindReg(d97.Reg, &d202)
			} else if d97.Type == tagFloat && d97.Loc == LocRegPair {
				ctx.FreeReg(d97.Reg)
				d202 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d97.Reg2}
				ctx.BindReg(d97.Reg2, &d202)
				ctx.BindReg(d97.Reg2, &d202)
			} else {
				d202 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d97}, 1)
				d202.Type = tagFloat
				ctx.BindReg(d202.Reg, &d202)
			}
			ctx.FreeDesc(&d97)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d202)
			var d203 JITValueDesc
			if d3.Loc == LocImm && d202.Loc == LocImm {
				d203 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d3.Imm.Float() + d202.Imm.Float())}
			} else if d3.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				_, xBits := d3.Imm.RawWords()
				ctx.EmitMovRegImm64(scratch, xBits)
				ctx.EmitAddFloat64(scratch, d202.Reg)
				d203 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else if d202.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.EmitMovRegReg(scratch, d3.Reg)
				_, yBits := d202.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, yBits)
				ctx.EmitAddFloat64(scratch, RegR11)
				d203 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else {
				r16 := ctx.AllocRegExcept(d3.Reg, d202.Reg)
				ctx.EmitMovRegReg(r16, d3.Reg)
				ctx.EmitAddFloat64(r16, d202.Reg)
				d203 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r16}
				ctx.BindReg(r16, &d203)
			}
			if d203.Loc == LocReg && d3.Loc == LocReg && d203.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = LocNone
			}
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d204 JITValueDesc
			if d2.Loc == LocImm {
				d204 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d204 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d204)
			}
			if d204.Loc == LocReg && d2.Loc == LocReg && d204.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.EnsureDesc(&d203)
			if d203.Loc == LocReg {
				ctx.ProtectReg(d203.Reg)
			} else if d203.Loc == LocRegPair {
				ctx.ProtectReg(d203.Reg)
				ctx.ProtectReg(d203.Reg2)
			}
			ctx.EnsureDesc(&d204)
			if d204.Loc == LocReg {
				ctx.ProtectReg(d204.Reg)
			} else if d204.Loc == LocRegPair {
				ctx.ProtectReg(d204.Reg)
				ctx.ProtectReg(d204.Reg2)
			}
			d205 = d204
			if d205.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d205)
			ctx.EmitStoreToStack(d205, int32(bbs[9].PhiBase)+int32(0))
			d206 = d203
			if d206.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d206)
			ctx.EmitStoreToStack(d206, int32(bbs[9].PhiBase)+int32(16))
			if d203.Loc == LocReg {
				ctx.UnprotectReg(d203.Reg)
			} else if d203.Loc == LocRegPair {
				ctx.UnprotectReg(d203.Reg)
				ctx.UnprotectReg(d203.Reg2)
			}
			if d204.Loc == LocReg {
				ctx.UnprotectReg(d204.Reg)
			} else if d204.Loc == LocRegPair {
				ctx.UnprotectReg(d204.Reg)
				ctx.UnprotectReg(d204.Reg2)
			}
			ps207 := PhiState{General: ps.General}
			ps207.OverlayValues = make([]JITValueDesc, 207)
			ps207.OverlayValues[0] = d0
			ps207.OverlayValues[1] = d1
			ps207.OverlayValues[2] = d2
			ps207.OverlayValues[3] = d3
			ps207.OverlayValues[5] = d5
			ps207.OverlayValues[6] = d6
			ps207.OverlayValues[7] = d7
			ps207.OverlayValues[8] = d8
			ps207.OverlayValues[9] = d9
			ps207.OverlayValues[10] = d10
			ps207.OverlayValues[11] = d11
			ps207.OverlayValues[28] = d28
			ps207.OverlayValues[29] = d29
			ps207.OverlayValues[30] = d30
			ps207.OverlayValues[50] = d50
			ps207.OverlayValues[51] = d51
			ps207.OverlayValues[52] = d52
			ps207.OverlayValues[53] = d53
			ps207.OverlayValues[54] = d54
			ps207.OverlayValues[57] = d57
			ps207.OverlayValues[58] = d58
			ps207.OverlayValues[83] = d83
			ps207.OverlayValues[84] = d84
			ps207.OverlayValues[85] = d85
			ps207.OverlayValues[86] = d86
			ps207.OverlayValues[87] = d87
			ps207.OverlayValues[89] = d89
			ps207.OverlayValues[90] = d90
			ps207.OverlayValues[91] = d91
			ps207.OverlayValues[92] = d92
			ps207.OverlayValues[93] = d93
			ps207.OverlayValues[95] = d95
			ps207.OverlayValues[96] = d96
			ps207.OverlayValues[97] = d97
			ps207.OverlayValues[98] = d98
			ps207.OverlayValues[99] = d99
			ps207.OverlayValues[100] = d100
			ps207.OverlayValues[101] = d101
			ps207.OverlayValues[145] = d145
			ps207.OverlayValues[146] = d146
			ps207.OverlayValues[147] = d147
			ps207.OverlayValues[148] = d148
			ps207.OverlayValues[149] = d149
			ps207.OverlayValues[152] = d152
			ps207.OverlayValues[153] = d153
			ps207.OverlayValues[202] = d202
			ps207.OverlayValues[203] = d203
			ps207.OverlayValues[204] = d204
			ps207.OverlayValues[205] = d205
			ps207.OverlayValues[206] = d206
			ps207.PhiValues = make([]JITValueDesc, 2)
			d208 = d204
			ps207.PhiValues[0] = d208
			d209 = d203
			ps207.PhiValues[1] = d209
			if ps207.General && bbs[9].Rendered {
				ctx.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps207)
			return result
			}
			argPinned210 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned210 = append(argPinned210, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned210 = append(argPinned210, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned210 = append(argPinned210, ai.Reg2)
					}
				}
			}
			ps211 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps211)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(64))
			ctx.EmitAddRSP32(int32(64))
			for _, r := range argPinned210 {
				ctx.UnprotectReg(r)
			}
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
			var d16 JITValueDesc
			_ = d16
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
			var d40 JITValueDesc
			_ = d40
			var d42 JITValueDesc
			_ = d42
			var d43 JITValueDesc
			_ = d43
			var d46 JITValueDesc
			_ = d46
			var d71 JITValueDesc
			_ = d71
			var d72 JITValueDesc
			_ = d72
			var d73 JITValueDesc
			_ = d73
			var d74 JITValueDesc
			_ = d74
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
			var d193 JITValueDesc
			_ = d193
			var d194 JITValueDesc
			_ = d194
			var d254 JITValueDesc
			_ = d254
			var d255 JITValueDesc
			_ = d255
			var d256 JITValueDesc
			_ = d256
			var d257 JITValueDesc
			_ = d257
			var d258 JITValueDesc
			_ = d258
			var d325 JITValueDesc
			_ = d325
			var d326 JITValueDesc
			_ = d326
			var d327 JITValueDesc
			_ = d327
			var d329 JITValueDesc
			_ = d329
			var d330 JITValueDesc
			_ = d330
			var d331 JITValueDesc
			_ = d331
			var d332 JITValueDesc
			_ = d332
			var d333 JITValueDesc
			_ = d333
			var d334 JITValueDesc
			_ = d334
			var d335 JITValueDesc
			_ = d335
			var d336 JITValueDesc
			_ = d336
			var d337 JITValueDesc
			_ = d337
			var d339 JITValueDesc
			_ = d339
			var d340 JITValueDesc
			_ = d340
			var d341 JITValueDesc
			_ = d341
			var d342 JITValueDesc
			_ = d342
			var d343 JITValueDesc
			_ = d343
			var d344 JITValueDesc
			_ = d344
			var d345 JITValueDesc
			_ = d345
			var d348 JITValueDesc
			_ = d348
			var d349 JITValueDesc
			_ = d349
			var d435 JITValueDesc
			_ = d435
			var d436 JITValueDesc
			_ = d436
			var d437 JITValueDesc
			_ = d437
			var d438 JITValueDesc
			_ = d438
			var d439 JITValueDesc
			_ = d439
			var d442 JITValueDesc
			_ = d442
			var d443 JITValueDesc
			_ = d443
			var d536 JITValueDesc
			_ = d536
			var d537 JITValueDesc
			_ = d537
			var d538 JITValueDesc
			_ = d538
			var d539 JITValueDesc
			_ = d539
			var d540 JITValueDesc
			_ = d540
			var d541 JITValueDesc
			_ = d541
			var d542 JITValueDesc
			_ = d542
			var d544 JITValueDesc
			_ = d544
			var d545 JITValueDesc
			_ = d545
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(112)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(112)
			}
			d0 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			var bbs [19]BBDescriptor
			bbs[1].PhiBase = int32(0)
			bbs[1].PhiCount = uint16(1)
			bbs[9].PhiBase = int32(16)
			bbs[9].PhiCount = uint16(2)
			bbs[15].PhiBase = int32(48)
			bbs[15].PhiCount = uint16(2)
			bbs[16].PhiBase = int32(80)
			bbs[16].PhiCount = uint16(2)
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
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.ReserveLabel()
			bbpos_0_12 := int32(-1)
			_ = bbpos_0_12
			lbl13 := ctx.ReserveLabel()
			bbpos_0_13 := int32(-1)
			_ = bbpos_0_13
			lbl14 := ctx.ReserveLabel()
			bbpos_0_14 := int32(-1)
			_ = bbpos_0_14
			lbl15 := ctx.ReserveLabel()
			bbpos_0_15 := int32(-1)
			_ = bbpos_0_15
			lbl16 := ctx.ReserveLabel()
			bbpos_0_16 := int32(-1)
			_ = bbpos_0_16
			lbl17 := ctx.ReserveLabel()
			bbpos_0_17 := int32(-1)
			_ = bbpos_0_17
			lbl18 := ctx.ReserveLabel()
			bbpos_0_18 := int32(-1)
			_ = bbpos_0_18
			lbl19 := ctx.ReserveLabel()
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
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
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps8)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d10 := ps.PhiValues[0]
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(bbs[1].PhiBase)+int32(0))
				}
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
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
				ctx.EmitMovRegReg(scratch, d0.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
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
					ctx.EmitCmpRegImm32(d11.Reg, int32(d7.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
					ctx.EmitCmpInt64(d11.Reg, RegR11)
				}
				ctx.EmitSetcc(r1, CcL)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d12)
			} else if d11.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d11.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d7.Reg)
				ctx.EmitSetcc(r2, CcL)
				d12 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d12)
			} else {
				r3 := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitCmpInt64(d11.Reg, d7.Reg)
				ctx.EmitSetcc(r3, CcL)
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
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d16 := ps.PhiValues[0]
					ctx.EnsureDesc(&d16)
					ctx.EmitStoreToStack(d16, int32(bbs[1].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl20 := ctx.ReserveLabel()
			lbl21 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d13.Reg, 0)
			ctx.EmitJcc(CcNE, lbl20)
			ctx.EmitJmp(lbl21)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl21)
			ctx.EmitJmp(lbl4)
			ps17 := PhiState{General: true}
			ps17.OverlayValues = make([]JITValueDesc, 17)
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
			ps17.OverlayValues[16] = d16
			ps18 := PhiState{General: true}
			ps18.OverlayValues = make([]JITValueDesc, 17)
			ps18.OverlayValues[0] = d0
			ps18.OverlayValues[1] = d1
			ps18.OverlayValues[2] = d2
			ps18.OverlayValues[3] = d3
			ps18.OverlayValues[4] = d4
			ps18.OverlayValues[5] = d5
			ps18.OverlayValues[6] = d6
			ps18.OverlayValues[7] = d7
			ps18.OverlayValues[9] = d9
			ps18.OverlayValues[10] = d10
			ps18.OverlayValues[11] = d11
			ps18.OverlayValues[12] = d12
			ps18.OverlayValues[13] = d13
			ps18.OverlayValues[16] = d16
			snap19 := d0
			snap20 := d1
			snap21 := d2
			snap22 := d3
			snap23 := d4
			snap24 := d5
			snap25 := d6
			snap26 := d7
			snap27 := d9
			snap28 := d10
			snap29 := d11
			snap30 := d12
			snap31 := d13
			snap32 := d16
			alloc33 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps18)
			}
			ctx.RestoreAllocState(alloc33)
			d0 = snap19
			d1 = snap20
			d2 = snap21
			d3 = snap22
			d4 = snap23
			d5 = snap24
			d6 = snap25
			d7 = snap26
			d9 = snap27
			d10 = snap28
			d11 = snap29
			d12 = snap30
			d13 = snap31
			d16 = snap32
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps17)
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
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d11)
			var d34 JITValueDesc
			if d11.Loc == LocImm {
				idx := int(d11.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d34 = args[idx]
				d34.ID = 0
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
				lbl22 := ctx.ReserveLabel()
				lbl23 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d11.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl23)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d11.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r4, ai.Reg)
						ctx.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r4, tmp.Reg)
						ctx.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl23)
				d35 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d35)
				ctx.BindReg(r5, &d35)
				ctx.BindReg(r4, &d35)
				ctx.BindReg(r5, &d35)
				ctx.EmitMakeNil(d35)
				ctx.MarkLabel(lbl22)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d34 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d34)
				ctx.BindReg(r5, &d34)
			}
			d37 = d34
			d37.ID = 0
			d36 = ctx.EmitTagEqualsBorrowed(&d37, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d34)
			d38 = d36
			ctx.EnsureDesc(&d38)
			if d38.Loc != LocImm && d38.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d38.Loc == LocImm {
				if d38.Imm.Bool() {
			ps39 := PhiState{General: ps.General}
			ps39.OverlayValues = make([]JITValueDesc, 39)
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
			ps39.OverlayValues[16] = d16
			ps39.OverlayValues[34] = d34
			ps39.OverlayValues[35] = d35
			ps39.OverlayValues[36] = d36
			ps39.OverlayValues[37] = d37
			ps39.OverlayValues[38] = d38
					return bbs[4].RenderPS(ps39)
				}
			ctx.EnsureDesc(&d11)
			if d11.Loc == LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d40 = d11
			if d40.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d40)
			ctx.EmitStoreToStack(d40, int32(bbs[1].PhiBase)+int32(0))
			if d11.Loc == LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
			ps41 := PhiState{General: ps.General}
			ps41.OverlayValues = make([]JITValueDesc, 41)
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
			ps41.OverlayValues[16] = d16
			ps41.OverlayValues[34] = d34
			ps41.OverlayValues[35] = d35
			ps41.OverlayValues[36] = d36
			ps41.OverlayValues[37] = d37
			ps41.OverlayValues[38] = d38
			ps41.OverlayValues[40] = d40
			ps41.PhiValues = make([]JITValueDesc, 1)
			d42 = d11
			ps41.PhiValues[0] = d42
				return bbs[1].RenderPS(ps41)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl24 := ctx.ReserveLabel()
			lbl25 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d38.Reg, 0)
			ctx.EmitJcc(CcNE, lbl24)
			ctx.EmitJmp(lbl25)
			ctx.MarkLabel(lbl24)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl25)
			ctx.EnsureDesc(&d11)
			if d11.Loc == LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d43 = d11
			if d43.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			ctx.EmitStoreToStack(d43, int32(bbs[1].PhiBase)+int32(0))
			if d11.Loc == LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
			ctx.EmitJmp(lbl2)
			ps44 := PhiState{General: true}
			ps44.OverlayValues = make([]JITValueDesc, 44)
			ps44.OverlayValues[0] = d0
			ps44.OverlayValues[1] = d1
			ps44.OverlayValues[2] = d2
			ps44.OverlayValues[3] = d3
			ps44.OverlayValues[4] = d4
			ps44.OverlayValues[5] = d5
			ps44.OverlayValues[6] = d6
			ps44.OverlayValues[7] = d7
			ps44.OverlayValues[9] = d9
			ps44.OverlayValues[10] = d10
			ps44.OverlayValues[11] = d11
			ps44.OverlayValues[12] = d12
			ps44.OverlayValues[13] = d13
			ps44.OverlayValues[16] = d16
			ps44.OverlayValues[34] = d34
			ps44.OverlayValues[35] = d35
			ps44.OverlayValues[36] = d36
			ps44.OverlayValues[37] = d37
			ps44.OverlayValues[38] = d38
			ps44.OverlayValues[40] = d40
			ps44.OverlayValues[42] = d42
			ps44.OverlayValues[43] = d43
			ps45 := PhiState{General: true}
			ps45.OverlayValues = make([]JITValueDesc, 44)
			ps45.OverlayValues[0] = d0
			ps45.OverlayValues[1] = d1
			ps45.OverlayValues[2] = d2
			ps45.OverlayValues[3] = d3
			ps45.OverlayValues[4] = d4
			ps45.OverlayValues[5] = d5
			ps45.OverlayValues[6] = d6
			ps45.OverlayValues[7] = d7
			ps45.OverlayValues[9] = d9
			ps45.OverlayValues[10] = d10
			ps45.OverlayValues[11] = d11
			ps45.OverlayValues[12] = d12
			ps45.OverlayValues[13] = d13
			ps45.OverlayValues[16] = d16
			ps45.OverlayValues[34] = d34
			ps45.OverlayValues[35] = d35
			ps45.OverlayValues[36] = d36
			ps45.OverlayValues[37] = d37
			ps45.OverlayValues[38] = d38
			ps45.OverlayValues[40] = d40
			ps45.OverlayValues[42] = d42
			ps45.OverlayValues[43] = d43
			ps45.PhiValues = make([]JITValueDesc, 1)
			d46 = d11
			ps45.PhiValues[0] = d46
			snap47 := d0
			snap48 := d1
			snap49 := d2
			snap50 := d3
			snap51 := d4
			snap52 := d5
			snap53 := d6
			snap54 := d7
			snap55 := d9
			snap56 := d10
			snap57 := d11
			snap58 := d12
			snap59 := d13
			snap60 := d16
			snap61 := d34
			snap62 := d35
			snap63 := d36
			snap64 := d37
			snap65 := d38
			snap66 := d40
			snap67 := d42
			snap68 := d43
			snap69 := d46
			alloc70 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps45)
			}
			ctx.RestoreAllocState(alloc70)
			d0 = snap47
			d1 = snap48
			d2 = snap49
			d3 = snap50
			d4 = snap51
			d5 = snap52
			d6 = snap53
			d7 = snap54
			d9 = snap55
			d10 = snap56
			d11 = snap57
			d12 = snap58
			d13 = snap59
			d16 = snap60
			d34 = snap61
			d35 = snap62
			d36 = snap63
			d37 = snap64
			d38 = snap65
			d40 = snap66
			d42 = snap67
			d43 = snap68
			d46 = snap69
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps44)
			}
			return result
			ctx.FreeDesc(&d36)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			ctx.ReclaimUntrackedRegs()
			d71 = args[0]
			d71.ID = 0
			d73 = d71
			d73.ID = 0
			d72 = ctx.EmitTagEqualsBorrowed(&d73, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d71)
			d74 = d72
			ctx.EnsureDesc(&d74)
			if d74.Loc != LocImm && d74.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d74.Loc == LocImm {
				if d74.Imm.Bool() {
			ps75 := PhiState{General: ps.General}
			ps75.OverlayValues = make([]JITValueDesc, 75)
			ps75.OverlayValues[0] = d0
			ps75.OverlayValues[1] = d1
			ps75.OverlayValues[2] = d2
			ps75.OverlayValues[3] = d3
			ps75.OverlayValues[4] = d4
			ps75.OverlayValues[5] = d5
			ps75.OverlayValues[6] = d6
			ps75.OverlayValues[7] = d7
			ps75.OverlayValues[9] = d9
			ps75.OverlayValues[10] = d10
			ps75.OverlayValues[11] = d11
			ps75.OverlayValues[12] = d12
			ps75.OverlayValues[13] = d13
			ps75.OverlayValues[16] = d16
			ps75.OverlayValues[34] = d34
			ps75.OverlayValues[35] = d35
			ps75.OverlayValues[36] = d36
			ps75.OverlayValues[37] = d37
			ps75.OverlayValues[38] = d38
			ps75.OverlayValues[40] = d40
			ps75.OverlayValues[42] = d42
			ps75.OverlayValues[43] = d43
			ps75.OverlayValues[46] = d46
			ps75.OverlayValues[71] = d71
			ps75.OverlayValues[72] = d72
			ps75.OverlayValues[73] = d73
			ps75.OverlayValues[74] = d74
					return bbs[5].RenderPS(ps75)
				}
			ps76 := PhiState{General: ps.General}
			ps76.OverlayValues = make([]JITValueDesc, 75)
			ps76.OverlayValues[0] = d0
			ps76.OverlayValues[1] = d1
			ps76.OverlayValues[2] = d2
			ps76.OverlayValues[3] = d3
			ps76.OverlayValues[4] = d4
			ps76.OverlayValues[5] = d5
			ps76.OverlayValues[6] = d6
			ps76.OverlayValues[7] = d7
			ps76.OverlayValues[9] = d9
			ps76.OverlayValues[10] = d10
			ps76.OverlayValues[11] = d11
			ps76.OverlayValues[12] = d12
			ps76.OverlayValues[13] = d13
			ps76.OverlayValues[16] = d16
			ps76.OverlayValues[34] = d34
			ps76.OverlayValues[35] = d35
			ps76.OverlayValues[36] = d36
			ps76.OverlayValues[37] = d37
			ps76.OverlayValues[38] = d38
			ps76.OverlayValues[40] = d40
			ps76.OverlayValues[42] = d42
			ps76.OverlayValues[43] = d43
			ps76.OverlayValues[46] = d46
			ps76.OverlayValues[71] = d71
			ps76.OverlayValues[72] = d72
			ps76.OverlayValues[73] = d73
			ps76.OverlayValues[74] = d74
				return bbs[6].RenderPS(ps76)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl26 := ctx.ReserveLabel()
			lbl27 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d74.Reg, 0)
			ctx.EmitJcc(CcNE, lbl26)
			ctx.EmitJmp(lbl27)
			ctx.MarkLabel(lbl26)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl27)
			ctx.EmitJmp(lbl7)
			ps77 := PhiState{General: true}
			ps77.OverlayValues = make([]JITValueDesc, 75)
			ps77.OverlayValues[0] = d0
			ps77.OverlayValues[1] = d1
			ps77.OverlayValues[2] = d2
			ps77.OverlayValues[3] = d3
			ps77.OverlayValues[4] = d4
			ps77.OverlayValues[5] = d5
			ps77.OverlayValues[6] = d6
			ps77.OverlayValues[7] = d7
			ps77.OverlayValues[9] = d9
			ps77.OverlayValues[10] = d10
			ps77.OverlayValues[11] = d11
			ps77.OverlayValues[12] = d12
			ps77.OverlayValues[13] = d13
			ps77.OverlayValues[16] = d16
			ps77.OverlayValues[34] = d34
			ps77.OverlayValues[35] = d35
			ps77.OverlayValues[36] = d36
			ps77.OverlayValues[37] = d37
			ps77.OverlayValues[38] = d38
			ps77.OverlayValues[40] = d40
			ps77.OverlayValues[42] = d42
			ps77.OverlayValues[43] = d43
			ps77.OverlayValues[46] = d46
			ps77.OverlayValues[71] = d71
			ps77.OverlayValues[72] = d72
			ps77.OverlayValues[73] = d73
			ps77.OverlayValues[74] = d74
			ps78 := PhiState{General: true}
			ps78.OverlayValues = make([]JITValueDesc, 75)
			ps78.OverlayValues[0] = d0
			ps78.OverlayValues[1] = d1
			ps78.OverlayValues[2] = d2
			ps78.OverlayValues[3] = d3
			ps78.OverlayValues[4] = d4
			ps78.OverlayValues[5] = d5
			ps78.OverlayValues[6] = d6
			ps78.OverlayValues[7] = d7
			ps78.OverlayValues[9] = d9
			ps78.OverlayValues[10] = d10
			ps78.OverlayValues[11] = d11
			ps78.OverlayValues[12] = d12
			ps78.OverlayValues[13] = d13
			ps78.OverlayValues[16] = d16
			ps78.OverlayValues[34] = d34
			ps78.OverlayValues[35] = d35
			ps78.OverlayValues[36] = d36
			ps78.OverlayValues[37] = d37
			ps78.OverlayValues[38] = d38
			ps78.OverlayValues[40] = d40
			ps78.OverlayValues[42] = d42
			ps78.OverlayValues[43] = d43
			ps78.OverlayValues[46] = d46
			ps78.OverlayValues[71] = d71
			ps78.OverlayValues[72] = d72
			ps78.OverlayValues[73] = d73
			ps78.OverlayValues[74] = d74
			snap79 := d0
			snap80 := d1
			snap81 := d2
			snap82 := d3
			snap83 := d4
			snap84 := d5
			snap85 := d6
			snap86 := d7
			snap87 := d9
			snap88 := d10
			snap89 := d11
			snap90 := d12
			snap91 := d13
			snap92 := d16
			snap93 := d34
			snap94 := d35
			snap95 := d36
			snap96 := d37
			snap97 := d38
			snap98 := d40
			snap99 := d42
			snap100 := d43
			snap101 := d46
			snap102 := d71
			snap103 := d72
			snap104 := d73
			snap105 := d74
			alloc106 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps78)
			}
			ctx.RestoreAllocState(alloc106)
			d0 = snap79
			d1 = snap80
			d2 = snap81
			d3 = snap82
			d4 = snap83
			d5 = snap84
			d6 = snap85
			d7 = snap86
			d9 = snap87
			d10 = snap88
			d11 = snap89
			d12 = snap90
			d13 = snap91
			d16 = snap92
			d34 = snap93
			d35 = snap94
			d36 = snap95
			d37 = snap96
			d38 = snap97
			d40 = snap98
			d42 = snap99
			d43 = snap100
			d46 = snap101
			d71 = snap102
			d72 = snap103
			d73 = snap104
			d74 = snap105
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps77)
			}
			return result
			ctx.FreeDesc(&d72)
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
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EmitMakeNil(result)
			result.Type = tagNil
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
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			ctx.ReclaimUntrackedRegs()
			d107 = args[0]
			d107.ID = 0
			var d108 JITValueDesc
			if d107.Loc == LocImm {
				d108 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d107.Imm.Int())}
			} else if d107.Type == tagInt && d107.Loc == LocRegPair {
				ctx.FreeReg(d107.Reg)
				d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg2}
				ctx.BindReg(d107.Reg2, &d108)
				ctx.BindReg(d107.Reg2, &d108)
			} else if d107.Type == tagInt && d107.Loc == LocReg {
				d108 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d107.Reg}
				ctx.BindReg(d107.Reg, &d108)
				ctx.BindReg(d107.Reg, &d108)
			} else {
				d108 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d107}, 1)
				d108.Type = tagInt
				ctx.BindReg(d108.Reg, &d108)
			}
			ctx.FreeDesc(&d107)
			ctx.EnsureDesc(&d108)
			if d108.Loc == LocReg {
				ctx.ProtectReg(d108.Reg)
			} else if d108.Loc == LocRegPair {
				ctx.ProtectReg(d108.Reg)
				ctx.ProtectReg(d108.Reg2)
			}
			d109 = d108
			if d109.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d109)
			ctx.EmitStoreToStack(d109, int32(bbs[9].PhiBase)+int32(0))
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(bbs[9].PhiBase)+int32(16))
			if d108.Loc == LocReg {
				ctx.UnprotectReg(d108.Reg)
			} else if d108.Loc == LocRegPair {
				ctx.UnprotectReg(d108.Reg)
				ctx.UnprotectReg(d108.Reg2)
			}
			ps110 := PhiState{General: ps.General}
			ps110.OverlayValues = make([]JITValueDesc, 110)
			ps110.OverlayValues[0] = d0
			ps110.OverlayValues[1] = d1
			ps110.OverlayValues[2] = d2
			ps110.OverlayValues[3] = d3
			ps110.OverlayValues[4] = d4
			ps110.OverlayValues[5] = d5
			ps110.OverlayValues[6] = d6
			ps110.OverlayValues[7] = d7
			ps110.OverlayValues[9] = d9
			ps110.OverlayValues[10] = d10
			ps110.OverlayValues[11] = d11
			ps110.OverlayValues[12] = d12
			ps110.OverlayValues[13] = d13
			ps110.OverlayValues[16] = d16
			ps110.OverlayValues[34] = d34
			ps110.OverlayValues[35] = d35
			ps110.OverlayValues[36] = d36
			ps110.OverlayValues[37] = d37
			ps110.OverlayValues[38] = d38
			ps110.OverlayValues[40] = d40
			ps110.OverlayValues[42] = d42
			ps110.OverlayValues[43] = d43
			ps110.OverlayValues[46] = d46
			ps110.OverlayValues[71] = d71
			ps110.OverlayValues[72] = d72
			ps110.OverlayValues[73] = d73
			ps110.OverlayValues[74] = d74
			ps110.OverlayValues[107] = d107
			ps110.OverlayValues[108] = d108
			ps110.OverlayValues[109] = d109
			ps110.PhiValues = make([]JITValueDesc, 2)
			d111 = d108
			ps110.PhiValues[0] = d111
			d112 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
			ps110.PhiValues[1] = d112
			if ps110.General && bbs[9].Rendered {
				ctx.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps110)
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
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			d113 = args[0]
			d113.ID = 0
			var d114 JITValueDesc
			if d113.Loc == LocImm {
				d114 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d113.Imm.Float())}
			} else if d113.Type == tagFloat && d113.Loc == LocReg {
				d114 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d113.Reg}
				ctx.BindReg(d113.Reg, &d114)
				ctx.BindReg(d113.Reg, &d114)
			} else if d113.Type == tagFloat && d113.Loc == LocRegPair {
				ctx.FreeReg(d113.Reg)
				d114 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d113.Reg2}
				ctx.BindReg(d113.Reg2, &d114)
				ctx.BindReg(d113.Reg2, &d114)
			} else {
				d114 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d113}, 1)
				d114.Type = tagFloat
				ctx.BindReg(d114.Reg, &d114)
			}
			ctx.FreeDesc(&d113)
			ctx.EnsureDesc(&d114)
			if d114.Loc == LocReg {
				ctx.ProtectReg(d114.Reg)
			} else if d114.Loc == LocRegPair {
				ctx.ProtectReg(d114.Reg)
				ctx.ProtectReg(d114.Reg2)
			}
			d115 = d114
			if d115.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d115)
			ctx.EmitStoreToStack(d115, int32(bbs[16].PhiBase)+int32(0))
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(bbs[16].PhiBase)+int32(16))
			if d114.Loc == LocReg {
				ctx.UnprotectReg(d114.Reg)
			} else if d114.Loc == LocRegPair {
				ctx.UnprotectReg(d114.Reg)
				ctx.UnprotectReg(d114.Reg2)
			}
			ps116 := PhiState{General: ps.General}
			ps116.OverlayValues = make([]JITValueDesc, 116)
			ps116.OverlayValues[0] = d0
			ps116.OverlayValues[1] = d1
			ps116.OverlayValues[2] = d2
			ps116.OverlayValues[3] = d3
			ps116.OverlayValues[4] = d4
			ps116.OverlayValues[5] = d5
			ps116.OverlayValues[6] = d6
			ps116.OverlayValues[7] = d7
			ps116.OverlayValues[9] = d9
			ps116.OverlayValues[10] = d10
			ps116.OverlayValues[11] = d11
			ps116.OverlayValues[12] = d12
			ps116.OverlayValues[13] = d13
			ps116.OverlayValues[16] = d16
			ps116.OverlayValues[34] = d34
			ps116.OverlayValues[35] = d35
			ps116.OverlayValues[36] = d36
			ps116.OverlayValues[37] = d37
			ps116.OverlayValues[38] = d38
			ps116.OverlayValues[40] = d40
			ps116.OverlayValues[42] = d42
			ps116.OverlayValues[43] = d43
			ps116.OverlayValues[46] = d46
			ps116.OverlayValues[71] = d71
			ps116.OverlayValues[72] = d72
			ps116.OverlayValues[73] = d73
			ps116.OverlayValues[74] = d74
			ps116.OverlayValues[107] = d107
			ps116.OverlayValues[108] = d108
			ps116.OverlayValues[109] = d109
			ps116.OverlayValues[111] = d111
			ps116.OverlayValues[112] = d112
			ps116.OverlayValues[113] = d113
			ps116.OverlayValues[114] = d114
			ps116.OverlayValues[115] = d115
			ps116.PhiValues = make([]JITValueDesc, 2)
			d117 = d114
			ps116.PhiValues[0] = d117
			d118 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
			ps116.PhiValues[1] = d118
			if ps116.General && bbs[16].Rendered {
				ctx.EmitJmp(lbl17)
				return result
			}
			return bbs[16].RenderPS(ps116)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != LocNone {
				d117 = ps.OverlayValues[117]
			}
			if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != LocNone {
				d118 = ps.OverlayValues[118]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d119 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d119 = args[idx]
				d119.ID = 0
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
				lbl28 := ctx.ReserveLabel()
				lbl29 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl29)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r6, ai.Reg)
						ctx.EmitMovRegReg(r7, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r6, tmp.Reg)
						ctx.EmitMovRegReg(r7, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
						ctx.BindReg(r6, &pair)
						ctx.BindReg(r7, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r6, uint64(ptrWord))
							ctx.EmitMovRegImm64(r7, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl28)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl29)
				d120 := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d120)
				ctx.BindReg(r7, &d120)
				ctx.BindReg(r6, &d120)
				ctx.BindReg(r7, &d120)
				ctx.EmitMakeNil(d120)
				ctx.MarkLabel(lbl28)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d119 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d119)
				ctx.BindReg(r7, &d119)
			}
			var d121 JITValueDesc
			if d119.Loc == LocImm {
				d121 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d119.Imm.Int())}
			} else if d119.Type == tagInt && d119.Loc == LocRegPair {
				ctx.FreeReg(d119.Reg)
				d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d119.Reg2}
				ctx.BindReg(d119.Reg2, &d121)
				ctx.BindReg(d119.Reg2, &d121)
			} else if d119.Type == tagInt && d119.Loc == LocReg {
				d121 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d119.Reg}
				ctx.BindReg(d119.Reg, &d121)
				ctx.BindReg(d119.Reg, &d121)
			} else {
				d121 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d119}, 1)
				d121.Type = tagInt
				ctx.BindReg(d121.Reg, &d121)
			}
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d121)
			var d122 JITValueDesc
			if d1.Loc == LocImm && d121.Loc == LocImm {
				d122 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() - d121.Imm.Int())}
			} else if d121.Loc == LocImm && d121.Imm.Int() == 0 {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(r8, d1.Reg)
				d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d122)
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.EmitSubInt64(scratch, d121.Reg)
				d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d121.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d121.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d121.Imm.Int()))
					ctx.EmitSubInt64(scratch, RegR11)
				}
				d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r9 := ctx.AllocRegExcept(d1.Reg, d121.Reg)
				ctx.EmitMovRegReg(r9, d1.Reg)
				ctx.EmitSubInt64(r9, d121.Reg)
				d122 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r9}
				ctx.BindReg(r9, &d122)
			}
			if d122.Loc == LocReg && d1.Loc == LocReg && d122.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d121)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d123 JITValueDesc
			if d2.Loc == LocImm {
				d123 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d123 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			}
			if d123.Loc == LocReg && d2.Loc == LocReg && d123.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.EnsureDesc(&d122)
			if d122.Loc == LocReg {
				ctx.ProtectReg(d122.Reg)
			} else if d122.Loc == LocRegPair {
				ctx.ProtectReg(d122.Reg)
				ctx.ProtectReg(d122.Reg2)
			}
			ctx.EnsureDesc(&d123)
			if d123.Loc == LocReg {
				ctx.ProtectReg(d123.Reg)
			} else if d123.Loc == LocRegPair {
				ctx.ProtectReg(d123.Reg)
				ctx.ProtectReg(d123.Reg2)
			}
			d124 = d122
			if d124.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, int32(bbs[9].PhiBase)+int32(0))
			d125 = d123
			if d125.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d125)
			ctx.EmitStoreToStack(d125, int32(bbs[9].PhiBase)+int32(16))
			if d122.Loc == LocReg {
				ctx.UnprotectReg(d122.Reg)
			} else if d122.Loc == LocRegPair {
				ctx.UnprotectReg(d122.Reg)
				ctx.UnprotectReg(d122.Reg2)
			}
			if d123.Loc == LocReg {
				ctx.UnprotectReg(d123.Reg)
			} else if d123.Loc == LocRegPair {
				ctx.UnprotectReg(d123.Reg)
				ctx.UnprotectReg(d123.Reg2)
			}
			ps126 := PhiState{General: ps.General}
			ps126.OverlayValues = make([]JITValueDesc, 126)
			ps126.OverlayValues[0] = d0
			ps126.OverlayValues[1] = d1
			ps126.OverlayValues[2] = d2
			ps126.OverlayValues[3] = d3
			ps126.OverlayValues[4] = d4
			ps126.OverlayValues[5] = d5
			ps126.OverlayValues[6] = d6
			ps126.OverlayValues[7] = d7
			ps126.OverlayValues[9] = d9
			ps126.OverlayValues[10] = d10
			ps126.OverlayValues[11] = d11
			ps126.OverlayValues[12] = d12
			ps126.OverlayValues[13] = d13
			ps126.OverlayValues[16] = d16
			ps126.OverlayValues[34] = d34
			ps126.OverlayValues[35] = d35
			ps126.OverlayValues[36] = d36
			ps126.OverlayValues[37] = d37
			ps126.OverlayValues[38] = d38
			ps126.OverlayValues[40] = d40
			ps126.OverlayValues[42] = d42
			ps126.OverlayValues[43] = d43
			ps126.OverlayValues[46] = d46
			ps126.OverlayValues[71] = d71
			ps126.OverlayValues[72] = d72
			ps126.OverlayValues[73] = d73
			ps126.OverlayValues[74] = d74
			ps126.OverlayValues[107] = d107
			ps126.OverlayValues[108] = d108
			ps126.OverlayValues[109] = d109
			ps126.OverlayValues[111] = d111
			ps126.OverlayValues[112] = d112
			ps126.OverlayValues[113] = d113
			ps126.OverlayValues[114] = d114
			ps126.OverlayValues[115] = d115
			ps126.OverlayValues[117] = d117
			ps126.OverlayValues[118] = d118
			ps126.OverlayValues[119] = d119
			ps126.OverlayValues[120] = d120
			ps126.OverlayValues[121] = d121
			ps126.OverlayValues[122] = d122
			ps126.OverlayValues[123] = d123
			ps126.OverlayValues[124] = d124
			ps126.OverlayValues[125] = d125
			ps126.PhiValues = make([]JITValueDesc, 2)
			d127 = d122
			ps126.PhiValues[0] = d127
			d128 = d123
			ps126.PhiValues[1] = d128
			if ps126.General && bbs[9].Rendered {
				ctx.EmitJmp(lbl10)
				return result
			}
			return bbs[9].RenderPS(ps126)
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
					ctx.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
			}
			ctx.ReclaimUntrackedRegs()
			d129 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d129)
			var d130 JITValueDesc
			if d2.Loc == LocImm && d129.Loc == LocImm {
				d130 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == d129.Imm.Int())}
			} else if d129.Loc == LocImm {
				r10 := ctx.AllocRegExcept(d2.Reg)
				if d129.Imm.Int() >= -2147483648 && d129.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d2.Reg, int32(d129.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d129.Imm.Int()))
					ctx.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.EmitSetcc(r10, CcE)
				d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d130)
			} else if d2.Loc == LocImm {
				r11 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d129.Reg)
				ctx.EmitSetcc(r11, CcE)
				d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d130)
			} else {
				r12 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitCmpInt64(d2.Reg, d129.Reg)
				ctx.EmitSetcc(r12, CcE)
				d130 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
				ctx.BindReg(r12, &d130)
			}
			ctx.FreeDesc(&d129)
			d131 = d130
			ctx.EnsureDesc(&d131)
			if d131.Loc != LocImm && d131.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d131.Loc == LocImm {
				if d131.Imm.Bool() {
			ps132 := PhiState{General: ps.General}
			ps132.OverlayValues = make([]JITValueDesc, 132)
			ps132.OverlayValues[0] = d0
			ps132.OverlayValues[1] = d1
			ps132.OverlayValues[2] = d2
			ps132.OverlayValues[3] = d3
			ps132.OverlayValues[4] = d4
			ps132.OverlayValues[5] = d5
			ps132.OverlayValues[6] = d6
			ps132.OverlayValues[7] = d7
			ps132.OverlayValues[9] = d9
			ps132.OverlayValues[10] = d10
			ps132.OverlayValues[11] = d11
			ps132.OverlayValues[12] = d12
			ps132.OverlayValues[13] = d13
			ps132.OverlayValues[16] = d16
			ps132.OverlayValues[34] = d34
			ps132.OverlayValues[35] = d35
			ps132.OverlayValues[36] = d36
			ps132.OverlayValues[37] = d37
			ps132.OverlayValues[38] = d38
			ps132.OverlayValues[40] = d40
			ps132.OverlayValues[42] = d42
			ps132.OverlayValues[43] = d43
			ps132.OverlayValues[46] = d46
			ps132.OverlayValues[71] = d71
			ps132.OverlayValues[72] = d72
			ps132.OverlayValues[73] = d73
			ps132.OverlayValues[74] = d74
			ps132.OverlayValues[107] = d107
			ps132.OverlayValues[108] = d108
			ps132.OverlayValues[109] = d109
			ps132.OverlayValues[111] = d111
			ps132.OverlayValues[112] = d112
			ps132.OverlayValues[113] = d113
			ps132.OverlayValues[114] = d114
			ps132.OverlayValues[115] = d115
			ps132.OverlayValues[117] = d117
			ps132.OverlayValues[118] = d118
			ps132.OverlayValues[119] = d119
			ps132.OverlayValues[120] = d120
			ps132.OverlayValues[121] = d121
			ps132.OverlayValues[122] = d122
			ps132.OverlayValues[123] = d123
			ps132.OverlayValues[124] = d124
			ps132.OverlayValues[125] = d125
			ps132.OverlayValues[127] = d127
			ps132.OverlayValues[128] = d128
			ps132.OverlayValues[129] = d129
			ps132.OverlayValues[130] = d130
			ps132.OverlayValues[131] = d131
					return bbs[11].RenderPS(ps132)
				}
			ps133 := PhiState{General: ps.General}
			ps133.OverlayValues = make([]JITValueDesc, 132)
			ps133.OverlayValues[0] = d0
			ps133.OverlayValues[1] = d1
			ps133.OverlayValues[2] = d2
			ps133.OverlayValues[3] = d3
			ps133.OverlayValues[4] = d4
			ps133.OverlayValues[5] = d5
			ps133.OverlayValues[6] = d6
			ps133.OverlayValues[7] = d7
			ps133.OverlayValues[9] = d9
			ps133.OverlayValues[10] = d10
			ps133.OverlayValues[11] = d11
			ps133.OverlayValues[12] = d12
			ps133.OverlayValues[13] = d13
			ps133.OverlayValues[16] = d16
			ps133.OverlayValues[34] = d34
			ps133.OverlayValues[35] = d35
			ps133.OverlayValues[36] = d36
			ps133.OverlayValues[37] = d37
			ps133.OverlayValues[38] = d38
			ps133.OverlayValues[40] = d40
			ps133.OverlayValues[42] = d42
			ps133.OverlayValues[43] = d43
			ps133.OverlayValues[46] = d46
			ps133.OverlayValues[71] = d71
			ps133.OverlayValues[72] = d72
			ps133.OverlayValues[73] = d73
			ps133.OverlayValues[74] = d74
			ps133.OverlayValues[107] = d107
			ps133.OverlayValues[108] = d108
			ps133.OverlayValues[109] = d109
			ps133.OverlayValues[111] = d111
			ps133.OverlayValues[112] = d112
			ps133.OverlayValues[113] = d113
			ps133.OverlayValues[114] = d114
			ps133.OverlayValues[115] = d115
			ps133.OverlayValues[117] = d117
			ps133.OverlayValues[118] = d118
			ps133.OverlayValues[119] = d119
			ps133.OverlayValues[120] = d120
			ps133.OverlayValues[121] = d121
			ps133.OverlayValues[122] = d122
			ps133.OverlayValues[123] = d123
			ps133.OverlayValues[124] = d124
			ps133.OverlayValues[125] = d125
			ps133.OverlayValues[127] = d127
			ps133.OverlayValues[128] = d128
			ps133.OverlayValues[129] = d129
			ps133.OverlayValues[130] = d130
			ps133.OverlayValues[131] = d131
				return bbs[12].RenderPS(ps133)
			}
			if !ps.General {
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
			lbl30 := ctx.ReserveLabel()
			lbl31 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d131.Reg, 0)
			ctx.EmitJcc(CcNE, lbl30)
			ctx.EmitJmp(lbl31)
			ctx.MarkLabel(lbl30)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl31)
			ctx.EmitJmp(lbl13)
			ps134 := PhiState{General: true}
			ps134.OverlayValues = make([]JITValueDesc, 132)
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
			ps134.OverlayValues[16] = d16
			ps134.OverlayValues[34] = d34
			ps134.OverlayValues[35] = d35
			ps134.OverlayValues[36] = d36
			ps134.OverlayValues[37] = d37
			ps134.OverlayValues[38] = d38
			ps134.OverlayValues[40] = d40
			ps134.OverlayValues[42] = d42
			ps134.OverlayValues[43] = d43
			ps134.OverlayValues[46] = d46
			ps134.OverlayValues[71] = d71
			ps134.OverlayValues[72] = d72
			ps134.OverlayValues[73] = d73
			ps134.OverlayValues[74] = d74
			ps134.OverlayValues[107] = d107
			ps134.OverlayValues[108] = d108
			ps134.OverlayValues[109] = d109
			ps134.OverlayValues[111] = d111
			ps134.OverlayValues[112] = d112
			ps134.OverlayValues[113] = d113
			ps134.OverlayValues[114] = d114
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
			ps134.OverlayValues[127] = d127
			ps134.OverlayValues[128] = d128
			ps134.OverlayValues[129] = d129
			ps134.OverlayValues[130] = d130
			ps134.OverlayValues[131] = d131
			ps135 := PhiState{General: true}
			ps135.OverlayValues = make([]JITValueDesc, 132)
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
			ps135.OverlayValues[16] = d16
			ps135.OverlayValues[34] = d34
			ps135.OverlayValues[35] = d35
			ps135.OverlayValues[36] = d36
			ps135.OverlayValues[37] = d37
			ps135.OverlayValues[38] = d38
			ps135.OverlayValues[40] = d40
			ps135.OverlayValues[42] = d42
			ps135.OverlayValues[43] = d43
			ps135.OverlayValues[46] = d46
			ps135.OverlayValues[71] = d71
			ps135.OverlayValues[72] = d72
			ps135.OverlayValues[73] = d73
			ps135.OverlayValues[74] = d74
			ps135.OverlayValues[107] = d107
			ps135.OverlayValues[108] = d108
			ps135.OverlayValues[109] = d109
			ps135.OverlayValues[111] = d111
			ps135.OverlayValues[112] = d112
			ps135.OverlayValues[113] = d113
			ps135.OverlayValues[114] = d114
			ps135.OverlayValues[115] = d115
			ps135.OverlayValues[117] = d117
			ps135.OverlayValues[118] = d118
			ps135.OverlayValues[119] = d119
			ps135.OverlayValues[120] = d120
			ps135.OverlayValues[121] = d121
			ps135.OverlayValues[122] = d122
			ps135.OverlayValues[123] = d123
			ps135.OverlayValues[124] = d124
			ps135.OverlayValues[125] = d125
			ps135.OverlayValues[127] = d127
			ps135.OverlayValues[128] = d128
			ps135.OverlayValues[129] = d129
			ps135.OverlayValues[130] = d130
			ps135.OverlayValues[131] = d131
			snap136 := d0
			snap137 := d1
			snap138 := d2
			snap139 := d3
			snap140 := d4
			snap141 := d5
			snap142 := d6
			snap143 := d7
			snap144 := d9
			snap145 := d10
			snap146 := d11
			snap147 := d12
			snap148 := d13
			snap149 := d16
			snap150 := d34
			snap151 := d35
			snap152 := d36
			snap153 := d37
			snap154 := d38
			snap155 := d40
			snap156 := d42
			snap157 := d43
			snap158 := d46
			snap159 := d71
			snap160 := d72
			snap161 := d73
			snap162 := d74
			snap163 := d107
			snap164 := d108
			snap165 := d109
			snap166 := d111
			snap167 := d112
			snap168 := d113
			snap169 := d114
			snap170 := d115
			snap171 := d117
			snap172 := d118
			snap173 := d119
			snap174 := d120
			snap175 := d121
			snap176 := d122
			snap177 := d123
			snap178 := d124
			snap179 := d125
			snap180 := d127
			snap181 := d128
			snap182 := d129
			snap183 := d130
			snap184 := d131
			alloc185 := ctx.SnapshotAllocState()
			if !bbs[12].Rendered {
				bbs[12].RenderPS(ps135)
			}
			ctx.RestoreAllocState(alloc185)
			d0 = snap136
			d1 = snap137
			d2 = snap138
			d3 = snap139
			d4 = snap140
			d5 = snap141
			d6 = snap142
			d7 = snap143
			d9 = snap144
			d10 = snap145
			d11 = snap146
			d12 = snap147
			d13 = snap148
			d16 = snap149
			d34 = snap150
			d35 = snap151
			d36 = snap152
			d37 = snap153
			d38 = snap154
			d40 = snap155
			d42 = snap156
			d43 = snap157
			d46 = snap158
			d71 = snap159
			d72 = snap160
			d73 = snap161
			d74 = snap162
			d107 = snap163
			d108 = snap164
			d109 = snap165
			d111 = snap166
			d112 = snap167
			d113 = snap168
			d114 = snap169
			d115 = snap170
			d117 = snap171
			d118 = snap172
			d119 = snap173
			d120 = snap174
			d121 = snap175
			d122 = snap176
			d123 = snap177
			d124 = snap178
			d125 = snap179
			d127 = snap180
			d128 = snap181
			d129 = snap182
			d130 = snap183
			d131 = snap184
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps134)
			}
			return result
			ctx.FreeDesc(&d130)
			return result
			}
			bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d186 := ps.PhiValues[0]
					ctx.EnsureDesc(&d186)
					ctx.EmitStoreToStack(d186, int32(bbs[9].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d187 := ps.PhiValues[1]
					ctx.EnsureDesc(&d187)
					ctx.EmitStoreToStack(d187, int32(bbs[9].PhiBase)+int32(16))
				}
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
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d2 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d188 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d188)
			var d189 JITValueDesc
			if d2.Loc == LocImm && d188.Loc == LocImm {
				d189 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d188.Imm.Int())}
			} else if d188.Loc == LocImm {
				r13 := ctx.AllocRegExcept(d2.Reg)
				if d188.Imm.Int() >= -2147483648 && d188.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d2.Reg, int32(d188.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d188.Imm.Int()))
					ctx.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.EmitSetcc(r13, CcL)
				d189 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
				ctx.BindReg(r13, &d189)
			} else if d2.Loc == LocImm {
				r14 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d188.Reg)
				ctx.EmitSetcc(r14, CcL)
				d189 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r14}
				ctx.BindReg(r14, &d189)
			} else {
				r15 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitCmpInt64(d2.Reg, d188.Reg)
				ctx.EmitSetcc(r15, CcL)
				d189 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d189)
			}
			ctx.FreeDesc(&d188)
			d190 = d189
			ctx.EnsureDesc(&d190)
			if d190.Loc != LocImm && d190.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d190.Loc == LocImm {
				if d190.Imm.Bool() {
			ps191 := PhiState{General: ps.General}
			ps191.OverlayValues = make([]JITValueDesc, 191)
			ps191.OverlayValues[0] = d0
			ps191.OverlayValues[1] = d1
			ps191.OverlayValues[2] = d2
			ps191.OverlayValues[3] = d3
			ps191.OverlayValues[4] = d4
			ps191.OverlayValues[5] = d5
			ps191.OverlayValues[6] = d6
			ps191.OverlayValues[7] = d7
			ps191.OverlayValues[9] = d9
			ps191.OverlayValues[10] = d10
			ps191.OverlayValues[11] = d11
			ps191.OverlayValues[12] = d12
			ps191.OverlayValues[13] = d13
			ps191.OverlayValues[16] = d16
			ps191.OverlayValues[34] = d34
			ps191.OverlayValues[35] = d35
			ps191.OverlayValues[36] = d36
			ps191.OverlayValues[37] = d37
			ps191.OverlayValues[38] = d38
			ps191.OverlayValues[40] = d40
			ps191.OverlayValues[42] = d42
			ps191.OverlayValues[43] = d43
			ps191.OverlayValues[46] = d46
			ps191.OverlayValues[71] = d71
			ps191.OverlayValues[72] = d72
			ps191.OverlayValues[73] = d73
			ps191.OverlayValues[74] = d74
			ps191.OverlayValues[107] = d107
			ps191.OverlayValues[108] = d108
			ps191.OverlayValues[109] = d109
			ps191.OverlayValues[111] = d111
			ps191.OverlayValues[112] = d112
			ps191.OverlayValues[113] = d113
			ps191.OverlayValues[114] = d114
			ps191.OverlayValues[115] = d115
			ps191.OverlayValues[117] = d117
			ps191.OverlayValues[118] = d118
			ps191.OverlayValues[119] = d119
			ps191.OverlayValues[120] = d120
			ps191.OverlayValues[121] = d121
			ps191.OverlayValues[122] = d122
			ps191.OverlayValues[123] = d123
			ps191.OverlayValues[124] = d124
			ps191.OverlayValues[125] = d125
			ps191.OverlayValues[127] = d127
			ps191.OverlayValues[128] = d128
			ps191.OverlayValues[129] = d129
			ps191.OverlayValues[130] = d130
			ps191.OverlayValues[131] = d131
			ps191.OverlayValues[186] = d186
			ps191.OverlayValues[187] = d187
			ps191.OverlayValues[188] = d188
			ps191.OverlayValues[189] = d189
			ps191.OverlayValues[190] = d190
					return bbs[10].RenderPS(ps191)
				}
			ps192 := PhiState{General: ps.General}
			ps192.OverlayValues = make([]JITValueDesc, 191)
			ps192.OverlayValues[0] = d0
			ps192.OverlayValues[1] = d1
			ps192.OverlayValues[2] = d2
			ps192.OverlayValues[3] = d3
			ps192.OverlayValues[4] = d4
			ps192.OverlayValues[5] = d5
			ps192.OverlayValues[6] = d6
			ps192.OverlayValues[7] = d7
			ps192.OverlayValues[9] = d9
			ps192.OverlayValues[10] = d10
			ps192.OverlayValues[11] = d11
			ps192.OverlayValues[12] = d12
			ps192.OverlayValues[13] = d13
			ps192.OverlayValues[16] = d16
			ps192.OverlayValues[34] = d34
			ps192.OverlayValues[35] = d35
			ps192.OverlayValues[36] = d36
			ps192.OverlayValues[37] = d37
			ps192.OverlayValues[38] = d38
			ps192.OverlayValues[40] = d40
			ps192.OverlayValues[42] = d42
			ps192.OverlayValues[43] = d43
			ps192.OverlayValues[46] = d46
			ps192.OverlayValues[71] = d71
			ps192.OverlayValues[72] = d72
			ps192.OverlayValues[73] = d73
			ps192.OverlayValues[74] = d74
			ps192.OverlayValues[107] = d107
			ps192.OverlayValues[108] = d108
			ps192.OverlayValues[109] = d109
			ps192.OverlayValues[111] = d111
			ps192.OverlayValues[112] = d112
			ps192.OverlayValues[113] = d113
			ps192.OverlayValues[114] = d114
			ps192.OverlayValues[115] = d115
			ps192.OverlayValues[117] = d117
			ps192.OverlayValues[118] = d118
			ps192.OverlayValues[119] = d119
			ps192.OverlayValues[120] = d120
			ps192.OverlayValues[121] = d121
			ps192.OverlayValues[122] = d122
			ps192.OverlayValues[123] = d123
			ps192.OverlayValues[124] = d124
			ps192.OverlayValues[125] = d125
			ps192.OverlayValues[127] = d127
			ps192.OverlayValues[128] = d128
			ps192.OverlayValues[129] = d129
			ps192.OverlayValues[130] = d130
			ps192.OverlayValues[131] = d131
			ps192.OverlayValues[186] = d186
			ps192.OverlayValues[187] = d187
			ps192.OverlayValues[188] = d188
			ps192.OverlayValues[189] = d189
			ps192.OverlayValues[190] = d190
				return bbs[8].RenderPS(ps192)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d193 := ps.PhiValues[0]
					ctx.EnsureDesc(&d193)
					ctx.EmitStoreToStack(d193, int32(bbs[9].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d194 := ps.PhiValues[1]
					ctx.EnsureDesc(&d194)
					ctx.EmitStoreToStack(d194, int32(bbs[9].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[9].RenderPS(ps)
			}
			lbl32 := ctx.ReserveLabel()
			lbl33 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d190.Reg, 0)
			ctx.EmitJcc(CcNE, lbl32)
			ctx.EmitJmp(lbl33)
			ctx.MarkLabel(lbl32)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl33)
			ctx.EmitJmp(lbl9)
			ps195 := PhiState{General: true}
			ps195.OverlayValues = make([]JITValueDesc, 195)
			ps195.OverlayValues[0] = d0
			ps195.OverlayValues[1] = d1
			ps195.OverlayValues[2] = d2
			ps195.OverlayValues[3] = d3
			ps195.OverlayValues[4] = d4
			ps195.OverlayValues[5] = d5
			ps195.OverlayValues[6] = d6
			ps195.OverlayValues[7] = d7
			ps195.OverlayValues[9] = d9
			ps195.OverlayValues[10] = d10
			ps195.OverlayValues[11] = d11
			ps195.OverlayValues[12] = d12
			ps195.OverlayValues[13] = d13
			ps195.OverlayValues[16] = d16
			ps195.OverlayValues[34] = d34
			ps195.OverlayValues[35] = d35
			ps195.OverlayValues[36] = d36
			ps195.OverlayValues[37] = d37
			ps195.OverlayValues[38] = d38
			ps195.OverlayValues[40] = d40
			ps195.OverlayValues[42] = d42
			ps195.OverlayValues[43] = d43
			ps195.OverlayValues[46] = d46
			ps195.OverlayValues[71] = d71
			ps195.OverlayValues[72] = d72
			ps195.OverlayValues[73] = d73
			ps195.OverlayValues[74] = d74
			ps195.OverlayValues[107] = d107
			ps195.OverlayValues[108] = d108
			ps195.OverlayValues[109] = d109
			ps195.OverlayValues[111] = d111
			ps195.OverlayValues[112] = d112
			ps195.OverlayValues[113] = d113
			ps195.OverlayValues[114] = d114
			ps195.OverlayValues[115] = d115
			ps195.OverlayValues[117] = d117
			ps195.OverlayValues[118] = d118
			ps195.OverlayValues[119] = d119
			ps195.OverlayValues[120] = d120
			ps195.OverlayValues[121] = d121
			ps195.OverlayValues[122] = d122
			ps195.OverlayValues[123] = d123
			ps195.OverlayValues[124] = d124
			ps195.OverlayValues[125] = d125
			ps195.OverlayValues[127] = d127
			ps195.OverlayValues[128] = d128
			ps195.OverlayValues[129] = d129
			ps195.OverlayValues[130] = d130
			ps195.OverlayValues[131] = d131
			ps195.OverlayValues[186] = d186
			ps195.OverlayValues[187] = d187
			ps195.OverlayValues[188] = d188
			ps195.OverlayValues[189] = d189
			ps195.OverlayValues[190] = d190
			ps195.OverlayValues[193] = d193
			ps195.OverlayValues[194] = d194
			ps196 := PhiState{General: true}
			ps196.OverlayValues = make([]JITValueDesc, 195)
			ps196.OverlayValues[0] = d0
			ps196.OverlayValues[1] = d1
			ps196.OverlayValues[2] = d2
			ps196.OverlayValues[3] = d3
			ps196.OverlayValues[4] = d4
			ps196.OverlayValues[5] = d5
			ps196.OverlayValues[6] = d6
			ps196.OverlayValues[7] = d7
			ps196.OverlayValues[9] = d9
			ps196.OverlayValues[10] = d10
			ps196.OverlayValues[11] = d11
			ps196.OverlayValues[12] = d12
			ps196.OverlayValues[13] = d13
			ps196.OverlayValues[16] = d16
			ps196.OverlayValues[34] = d34
			ps196.OverlayValues[35] = d35
			ps196.OverlayValues[36] = d36
			ps196.OverlayValues[37] = d37
			ps196.OverlayValues[38] = d38
			ps196.OverlayValues[40] = d40
			ps196.OverlayValues[42] = d42
			ps196.OverlayValues[43] = d43
			ps196.OverlayValues[46] = d46
			ps196.OverlayValues[71] = d71
			ps196.OverlayValues[72] = d72
			ps196.OverlayValues[73] = d73
			ps196.OverlayValues[74] = d74
			ps196.OverlayValues[107] = d107
			ps196.OverlayValues[108] = d108
			ps196.OverlayValues[109] = d109
			ps196.OverlayValues[111] = d111
			ps196.OverlayValues[112] = d112
			ps196.OverlayValues[113] = d113
			ps196.OverlayValues[114] = d114
			ps196.OverlayValues[115] = d115
			ps196.OverlayValues[117] = d117
			ps196.OverlayValues[118] = d118
			ps196.OverlayValues[119] = d119
			ps196.OverlayValues[120] = d120
			ps196.OverlayValues[121] = d121
			ps196.OverlayValues[122] = d122
			ps196.OverlayValues[123] = d123
			ps196.OverlayValues[124] = d124
			ps196.OverlayValues[125] = d125
			ps196.OverlayValues[127] = d127
			ps196.OverlayValues[128] = d128
			ps196.OverlayValues[129] = d129
			ps196.OverlayValues[130] = d130
			ps196.OverlayValues[131] = d131
			ps196.OverlayValues[186] = d186
			ps196.OverlayValues[187] = d187
			ps196.OverlayValues[188] = d188
			ps196.OverlayValues[189] = d189
			ps196.OverlayValues[190] = d190
			ps196.OverlayValues[193] = d193
			ps196.OverlayValues[194] = d194
			snap197 := d0
			snap198 := d1
			snap199 := d2
			snap200 := d3
			snap201 := d4
			snap202 := d5
			snap203 := d6
			snap204 := d7
			snap205 := d9
			snap206 := d10
			snap207 := d11
			snap208 := d12
			snap209 := d13
			snap210 := d16
			snap211 := d34
			snap212 := d35
			snap213 := d36
			snap214 := d37
			snap215 := d38
			snap216 := d40
			snap217 := d42
			snap218 := d43
			snap219 := d46
			snap220 := d71
			snap221 := d72
			snap222 := d73
			snap223 := d74
			snap224 := d107
			snap225 := d108
			snap226 := d109
			snap227 := d111
			snap228 := d112
			snap229 := d113
			snap230 := d114
			snap231 := d115
			snap232 := d117
			snap233 := d118
			snap234 := d119
			snap235 := d120
			snap236 := d121
			snap237 := d122
			snap238 := d123
			snap239 := d124
			snap240 := d125
			snap241 := d127
			snap242 := d128
			snap243 := d129
			snap244 := d130
			snap245 := d131
			snap246 := d186
			snap247 := d187
			snap248 := d188
			snap249 := d189
			snap250 := d190
			snap251 := d193
			snap252 := d194
			alloc253 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps196)
			}
			ctx.RestoreAllocState(alloc253)
			d0 = snap197
			d1 = snap198
			d2 = snap199
			d3 = snap200
			d4 = snap201
			d5 = snap202
			d6 = snap203
			d7 = snap204
			d9 = snap205
			d10 = snap206
			d11 = snap207
			d12 = snap208
			d13 = snap209
			d16 = snap210
			d34 = snap211
			d35 = snap212
			d36 = snap213
			d37 = snap214
			d38 = snap215
			d40 = snap216
			d42 = snap217
			d43 = snap218
			d46 = snap219
			d71 = snap220
			d72 = snap221
			d73 = snap222
			d74 = snap223
			d107 = snap224
			d108 = snap225
			d109 = snap226
			d111 = snap227
			d112 = snap228
			d113 = snap229
			d114 = snap230
			d115 = snap231
			d117 = snap232
			d118 = snap233
			d119 = snap234
			d120 = snap235
			d121 = snap236
			d122 = snap237
			d123 = snap238
			d124 = snap239
			d125 = snap240
			d127 = snap241
			d128 = snap242
			d129 = snap243
			d130 = snap244
			d131 = snap245
			d186 = snap246
			d187 = snap247
			d188 = snap248
			d189 = snap249
			d190 = snap250
			d193 = snap251
			d194 = snap252
			if !bbs[10].Rendered {
				return bbs[10].RenderPS(ps195)
			}
			return result
			ctx.FreeDesc(&d189)
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
					ctx.EmitJmp(lbl11)
					return result
				}
				bbs[10].Rendered = true
				bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_10 = bbs[10].Address
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d254 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d254 = args[idx]
				d254.ID = 0
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
				lbl34 := ctx.ReserveLabel()
				lbl35 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl35)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r16, ai.Reg)
						ctx.EmitMovRegReg(r17, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r16, tmp.Reg)
						ctx.EmitMovRegReg(r17, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r16, Reg2: r17}
						ctx.BindReg(r16, &pair)
						ctx.BindReg(r17, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r16, uint64(ptrWord))
							ctx.EmitMovRegImm64(r17, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl34)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl35)
				d255 := JITValueDesc{Loc: LocRegPair, Reg: r16, Reg2: r17}
				ctx.BindReg(r16, &d255)
				ctx.BindReg(r17, &d255)
				ctx.BindReg(r16, &d255)
				ctx.BindReg(r17, &d255)
				ctx.EmitMakeNil(d255)
				ctx.MarkLabel(lbl34)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d254 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r16, Reg2: r17}
				ctx.BindReg(r16, &d254)
				ctx.BindReg(r17, &d254)
			}
			d257 = d254
			d257.ID = 0
			d256 = ctx.EmitTagEqualsBorrowed(&d257, tagInt, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d254)
			d258 = d256
			ctx.EnsureDesc(&d258)
			if d258.Loc != LocImm && d258.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d258.Loc == LocImm {
				if d258.Imm.Bool() {
			ps259 := PhiState{General: ps.General}
			ps259.OverlayValues = make([]JITValueDesc, 259)
			ps259.OverlayValues[0] = d0
			ps259.OverlayValues[1] = d1
			ps259.OverlayValues[2] = d2
			ps259.OverlayValues[3] = d3
			ps259.OverlayValues[4] = d4
			ps259.OverlayValues[5] = d5
			ps259.OverlayValues[6] = d6
			ps259.OverlayValues[7] = d7
			ps259.OverlayValues[9] = d9
			ps259.OverlayValues[10] = d10
			ps259.OverlayValues[11] = d11
			ps259.OverlayValues[12] = d12
			ps259.OverlayValues[13] = d13
			ps259.OverlayValues[16] = d16
			ps259.OverlayValues[34] = d34
			ps259.OverlayValues[35] = d35
			ps259.OverlayValues[36] = d36
			ps259.OverlayValues[37] = d37
			ps259.OverlayValues[38] = d38
			ps259.OverlayValues[40] = d40
			ps259.OverlayValues[42] = d42
			ps259.OverlayValues[43] = d43
			ps259.OverlayValues[46] = d46
			ps259.OverlayValues[71] = d71
			ps259.OverlayValues[72] = d72
			ps259.OverlayValues[73] = d73
			ps259.OverlayValues[74] = d74
			ps259.OverlayValues[107] = d107
			ps259.OverlayValues[108] = d108
			ps259.OverlayValues[109] = d109
			ps259.OverlayValues[111] = d111
			ps259.OverlayValues[112] = d112
			ps259.OverlayValues[113] = d113
			ps259.OverlayValues[114] = d114
			ps259.OverlayValues[115] = d115
			ps259.OverlayValues[117] = d117
			ps259.OverlayValues[118] = d118
			ps259.OverlayValues[119] = d119
			ps259.OverlayValues[120] = d120
			ps259.OverlayValues[121] = d121
			ps259.OverlayValues[122] = d122
			ps259.OverlayValues[123] = d123
			ps259.OverlayValues[124] = d124
			ps259.OverlayValues[125] = d125
			ps259.OverlayValues[127] = d127
			ps259.OverlayValues[128] = d128
			ps259.OverlayValues[129] = d129
			ps259.OverlayValues[130] = d130
			ps259.OverlayValues[131] = d131
			ps259.OverlayValues[186] = d186
			ps259.OverlayValues[187] = d187
			ps259.OverlayValues[188] = d188
			ps259.OverlayValues[189] = d189
			ps259.OverlayValues[190] = d190
			ps259.OverlayValues[193] = d193
			ps259.OverlayValues[194] = d194
			ps259.OverlayValues[254] = d254
			ps259.OverlayValues[255] = d255
			ps259.OverlayValues[256] = d256
			ps259.OverlayValues[257] = d257
			ps259.OverlayValues[258] = d258
					return bbs[7].RenderPS(ps259)
				}
			ps260 := PhiState{General: ps.General}
			ps260.OverlayValues = make([]JITValueDesc, 259)
			ps260.OverlayValues[0] = d0
			ps260.OverlayValues[1] = d1
			ps260.OverlayValues[2] = d2
			ps260.OverlayValues[3] = d3
			ps260.OverlayValues[4] = d4
			ps260.OverlayValues[5] = d5
			ps260.OverlayValues[6] = d6
			ps260.OverlayValues[7] = d7
			ps260.OverlayValues[9] = d9
			ps260.OverlayValues[10] = d10
			ps260.OverlayValues[11] = d11
			ps260.OverlayValues[12] = d12
			ps260.OverlayValues[13] = d13
			ps260.OverlayValues[16] = d16
			ps260.OverlayValues[34] = d34
			ps260.OverlayValues[35] = d35
			ps260.OverlayValues[36] = d36
			ps260.OverlayValues[37] = d37
			ps260.OverlayValues[38] = d38
			ps260.OverlayValues[40] = d40
			ps260.OverlayValues[42] = d42
			ps260.OverlayValues[43] = d43
			ps260.OverlayValues[46] = d46
			ps260.OverlayValues[71] = d71
			ps260.OverlayValues[72] = d72
			ps260.OverlayValues[73] = d73
			ps260.OverlayValues[74] = d74
			ps260.OverlayValues[107] = d107
			ps260.OverlayValues[108] = d108
			ps260.OverlayValues[109] = d109
			ps260.OverlayValues[111] = d111
			ps260.OverlayValues[112] = d112
			ps260.OverlayValues[113] = d113
			ps260.OverlayValues[114] = d114
			ps260.OverlayValues[115] = d115
			ps260.OverlayValues[117] = d117
			ps260.OverlayValues[118] = d118
			ps260.OverlayValues[119] = d119
			ps260.OverlayValues[120] = d120
			ps260.OverlayValues[121] = d121
			ps260.OverlayValues[122] = d122
			ps260.OverlayValues[123] = d123
			ps260.OverlayValues[124] = d124
			ps260.OverlayValues[125] = d125
			ps260.OverlayValues[127] = d127
			ps260.OverlayValues[128] = d128
			ps260.OverlayValues[129] = d129
			ps260.OverlayValues[130] = d130
			ps260.OverlayValues[131] = d131
			ps260.OverlayValues[186] = d186
			ps260.OverlayValues[187] = d187
			ps260.OverlayValues[188] = d188
			ps260.OverlayValues[189] = d189
			ps260.OverlayValues[190] = d190
			ps260.OverlayValues[193] = d193
			ps260.OverlayValues[194] = d194
			ps260.OverlayValues[254] = d254
			ps260.OverlayValues[255] = d255
			ps260.OverlayValues[256] = d256
			ps260.OverlayValues[257] = d257
			ps260.OverlayValues[258] = d258
				return bbs[8].RenderPS(ps260)
			}
			if !ps.General {
				ps.General = true
				return bbs[10].RenderPS(ps)
			}
			lbl36 := ctx.ReserveLabel()
			lbl37 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d258.Reg, 0)
			ctx.EmitJcc(CcNE, lbl36)
			ctx.EmitJmp(lbl37)
			ctx.MarkLabel(lbl36)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl37)
			ctx.EmitJmp(lbl9)
			ps261 := PhiState{General: true}
			ps261.OverlayValues = make([]JITValueDesc, 259)
			ps261.OverlayValues[0] = d0
			ps261.OverlayValues[1] = d1
			ps261.OverlayValues[2] = d2
			ps261.OverlayValues[3] = d3
			ps261.OverlayValues[4] = d4
			ps261.OverlayValues[5] = d5
			ps261.OverlayValues[6] = d6
			ps261.OverlayValues[7] = d7
			ps261.OverlayValues[9] = d9
			ps261.OverlayValues[10] = d10
			ps261.OverlayValues[11] = d11
			ps261.OverlayValues[12] = d12
			ps261.OverlayValues[13] = d13
			ps261.OverlayValues[16] = d16
			ps261.OverlayValues[34] = d34
			ps261.OverlayValues[35] = d35
			ps261.OverlayValues[36] = d36
			ps261.OverlayValues[37] = d37
			ps261.OverlayValues[38] = d38
			ps261.OverlayValues[40] = d40
			ps261.OverlayValues[42] = d42
			ps261.OverlayValues[43] = d43
			ps261.OverlayValues[46] = d46
			ps261.OverlayValues[71] = d71
			ps261.OverlayValues[72] = d72
			ps261.OverlayValues[73] = d73
			ps261.OverlayValues[74] = d74
			ps261.OverlayValues[107] = d107
			ps261.OverlayValues[108] = d108
			ps261.OverlayValues[109] = d109
			ps261.OverlayValues[111] = d111
			ps261.OverlayValues[112] = d112
			ps261.OverlayValues[113] = d113
			ps261.OverlayValues[114] = d114
			ps261.OverlayValues[115] = d115
			ps261.OverlayValues[117] = d117
			ps261.OverlayValues[118] = d118
			ps261.OverlayValues[119] = d119
			ps261.OverlayValues[120] = d120
			ps261.OverlayValues[121] = d121
			ps261.OverlayValues[122] = d122
			ps261.OverlayValues[123] = d123
			ps261.OverlayValues[124] = d124
			ps261.OverlayValues[125] = d125
			ps261.OverlayValues[127] = d127
			ps261.OverlayValues[128] = d128
			ps261.OverlayValues[129] = d129
			ps261.OverlayValues[130] = d130
			ps261.OverlayValues[131] = d131
			ps261.OverlayValues[186] = d186
			ps261.OverlayValues[187] = d187
			ps261.OverlayValues[188] = d188
			ps261.OverlayValues[189] = d189
			ps261.OverlayValues[190] = d190
			ps261.OverlayValues[193] = d193
			ps261.OverlayValues[194] = d194
			ps261.OverlayValues[254] = d254
			ps261.OverlayValues[255] = d255
			ps261.OverlayValues[256] = d256
			ps261.OverlayValues[257] = d257
			ps261.OverlayValues[258] = d258
			ps262 := PhiState{General: true}
			ps262.OverlayValues = make([]JITValueDesc, 259)
			ps262.OverlayValues[0] = d0
			ps262.OverlayValues[1] = d1
			ps262.OverlayValues[2] = d2
			ps262.OverlayValues[3] = d3
			ps262.OverlayValues[4] = d4
			ps262.OverlayValues[5] = d5
			ps262.OverlayValues[6] = d6
			ps262.OverlayValues[7] = d7
			ps262.OverlayValues[9] = d9
			ps262.OverlayValues[10] = d10
			ps262.OverlayValues[11] = d11
			ps262.OverlayValues[12] = d12
			ps262.OverlayValues[13] = d13
			ps262.OverlayValues[16] = d16
			ps262.OverlayValues[34] = d34
			ps262.OverlayValues[35] = d35
			ps262.OverlayValues[36] = d36
			ps262.OverlayValues[37] = d37
			ps262.OverlayValues[38] = d38
			ps262.OverlayValues[40] = d40
			ps262.OverlayValues[42] = d42
			ps262.OverlayValues[43] = d43
			ps262.OverlayValues[46] = d46
			ps262.OverlayValues[71] = d71
			ps262.OverlayValues[72] = d72
			ps262.OverlayValues[73] = d73
			ps262.OverlayValues[74] = d74
			ps262.OverlayValues[107] = d107
			ps262.OverlayValues[108] = d108
			ps262.OverlayValues[109] = d109
			ps262.OverlayValues[111] = d111
			ps262.OverlayValues[112] = d112
			ps262.OverlayValues[113] = d113
			ps262.OverlayValues[114] = d114
			ps262.OverlayValues[115] = d115
			ps262.OverlayValues[117] = d117
			ps262.OverlayValues[118] = d118
			ps262.OverlayValues[119] = d119
			ps262.OverlayValues[120] = d120
			ps262.OverlayValues[121] = d121
			ps262.OverlayValues[122] = d122
			ps262.OverlayValues[123] = d123
			ps262.OverlayValues[124] = d124
			ps262.OverlayValues[125] = d125
			ps262.OverlayValues[127] = d127
			ps262.OverlayValues[128] = d128
			ps262.OverlayValues[129] = d129
			ps262.OverlayValues[130] = d130
			ps262.OverlayValues[131] = d131
			ps262.OverlayValues[186] = d186
			ps262.OverlayValues[187] = d187
			ps262.OverlayValues[188] = d188
			ps262.OverlayValues[189] = d189
			ps262.OverlayValues[190] = d190
			ps262.OverlayValues[193] = d193
			ps262.OverlayValues[194] = d194
			ps262.OverlayValues[254] = d254
			ps262.OverlayValues[255] = d255
			ps262.OverlayValues[256] = d256
			ps262.OverlayValues[257] = d257
			ps262.OverlayValues[258] = d258
			snap263 := d0
			snap264 := d1
			snap265 := d2
			snap266 := d3
			snap267 := d4
			snap268 := d5
			snap269 := d6
			snap270 := d7
			snap271 := d9
			snap272 := d10
			snap273 := d11
			snap274 := d12
			snap275 := d13
			snap276 := d16
			snap277 := d34
			snap278 := d35
			snap279 := d36
			snap280 := d37
			snap281 := d38
			snap282 := d40
			snap283 := d42
			snap284 := d43
			snap285 := d46
			snap286 := d71
			snap287 := d72
			snap288 := d73
			snap289 := d74
			snap290 := d107
			snap291 := d108
			snap292 := d109
			snap293 := d111
			snap294 := d112
			snap295 := d113
			snap296 := d114
			snap297 := d115
			snap298 := d117
			snap299 := d118
			snap300 := d119
			snap301 := d120
			snap302 := d121
			snap303 := d122
			snap304 := d123
			snap305 := d124
			snap306 := d125
			snap307 := d127
			snap308 := d128
			snap309 := d129
			snap310 := d130
			snap311 := d131
			snap312 := d186
			snap313 := d187
			snap314 := d188
			snap315 := d189
			snap316 := d190
			snap317 := d193
			snap318 := d194
			snap319 := d254
			snap320 := d255
			snap321 := d256
			snap322 := d257
			snap323 := d258
			alloc324 := ctx.SnapshotAllocState()
			if !bbs[8].Rendered {
				bbs[8].RenderPS(ps262)
			}
			ctx.RestoreAllocState(alloc324)
			d0 = snap263
			d1 = snap264
			d2 = snap265
			d3 = snap266
			d4 = snap267
			d5 = snap268
			d6 = snap269
			d7 = snap270
			d9 = snap271
			d10 = snap272
			d11 = snap273
			d12 = snap274
			d13 = snap275
			d16 = snap276
			d34 = snap277
			d35 = snap278
			d36 = snap279
			d37 = snap280
			d38 = snap281
			d40 = snap282
			d42 = snap283
			d43 = snap284
			d46 = snap285
			d71 = snap286
			d72 = snap287
			d73 = snap288
			d74 = snap289
			d107 = snap290
			d108 = snap291
			d109 = snap292
			d111 = snap293
			d112 = snap294
			d113 = snap295
			d114 = snap296
			d115 = snap297
			d117 = snap298
			d118 = snap299
			d119 = snap300
			d120 = snap301
			d121 = snap302
			d122 = snap303
			d123 = snap304
			d124 = snap305
			d125 = snap306
			d127 = snap307
			d128 = snap308
			d129 = snap309
			d130 = snap310
			d131 = snap311
			d186 = snap312
			d187 = snap313
			d188 = snap314
			d189 = snap315
			d190 = snap316
			d193 = snap317
			d194 = snap318
			d254 = snap319
			d255 = snap320
			d256 = snap321
			d257 = snap322
			d258 = snap323
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps261)
			}
			return result
			ctx.FreeDesc(&d256)
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
					ctx.EmitJmp(lbl12)
					return result
				}
				bbs[11].Rendered = true
				bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_11 = bbs[11].Address
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			ctx.EmitMakeInt(result, d1)
			if d1.Loc == LocReg { ctx.FreeReg(d1.Reg) }
			result.Type = tagInt
			ctx.EmitJmp(lbl0)
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
					ctx.EmitJmp(lbl13)
					return result
				}
				bbs[12].Rendered = true
				bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_12 = bbs[12].Address
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d325 JITValueDesc
			if d1.Loc == LocImm {
				d325 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d1.Imm.Int()))}
			} else {
				ctx.EmitCvtInt64ToFloat64(RegX0, d1.Reg)
				d325 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d325)
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			ctx.EnsureDesc(&d325)
			if d325.Loc == LocReg {
				ctx.ProtectReg(d325.Reg)
			} else if d325.Loc == LocRegPair {
				ctx.ProtectReg(d325.Reg)
				ctx.ProtectReg(d325.Reg2)
			}
			d326 = d2
			if d326.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d326)
			ctx.EmitStoreToStack(d326, int32(bbs[15].PhiBase)+int32(0))
			d327 = d325
			if d327.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d327)
			ctx.EmitStoreToStack(d327, int32(bbs[15].PhiBase)+int32(16))
			if d2.Loc == LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
			if d325.Loc == LocReg {
				ctx.UnprotectReg(d325.Reg)
			} else if d325.Loc == LocRegPair {
				ctx.UnprotectReg(d325.Reg)
				ctx.UnprotectReg(d325.Reg2)
			}
			ps328 := PhiState{General: ps.General}
			ps328.OverlayValues = make([]JITValueDesc, 328)
			ps328.OverlayValues[0] = d0
			ps328.OverlayValues[1] = d1
			ps328.OverlayValues[2] = d2
			ps328.OverlayValues[3] = d3
			ps328.OverlayValues[4] = d4
			ps328.OverlayValues[5] = d5
			ps328.OverlayValues[6] = d6
			ps328.OverlayValues[7] = d7
			ps328.OverlayValues[9] = d9
			ps328.OverlayValues[10] = d10
			ps328.OverlayValues[11] = d11
			ps328.OverlayValues[12] = d12
			ps328.OverlayValues[13] = d13
			ps328.OverlayValues[16] = d16
			ps328.OverlayValues[34] = d34
			ps328.OverlayValues[35] = d35
			ps328.OverlayValues[36] = d36
			ps328.OverlayValues[37] = d37
			ps328.OverlayValues[38] = d38
			ps328.OverlayValues[40] = d40
			ps328.OverlayValues[42] = d42
			ps328.OverlayValues[43] = d43
			ps328.OverlayValues[46] = d46
			ps328.OverlayValues[71] = d71
			ps328.OverlayValues[72] = d72
			ps328.OverlayValues[73] = d73
			ps328.OverlayValues[74] = d74
			ps328.OverlayValues[107] = d107
			ps328.OverlayValues[108] = d108
			ps328.OverlayValues[109] = d109
			ps328.OverlayValues[111] = d111
			ps328.OverlayValues[112] = d112
			ps328.OverlayValues[113] = d113
			ps328.OverlayValues[114] = d114
			ps328.OverlayValues[115] = d115
			ps328.OverlayValues[117] = d117
			ps328.OverlayValues[118] = d118
			ps328.OverlayValues[119] = d119
			ps328.OverlayValues[120] = d120
			ps328.OverlayValues[121] = d121
			ps328.OverlayValues[122] = d122
			ps328.OverlayValues[123] = d123
			ps328.OverlayValues[124] = d124
			ps328.OverlayValues[125] = d125
			ps328.OverlayValues[127] = d127
			ps328.OverlayValues[128] = d128
			ps328.OverlayValues[129] = d129
			ps328.OverlayValues[130] = d130
			ps328.OverlayValues[131] = d131
			ps328.OverlayValues[186] = d186
			ps328.OverlayValues[187] = d187
			ps328.OverlayValues[188] = d188
			ps328.OverlayValues[189] = d189
			ps328.OverlayValues[190] = d190
			ps328.OverlayValues[193] = d193
			ps328.OverlayValues[194] = d194
			ps328.OverlayValues[254] = d254
			ps328.OverlayValues[255] = d255
			ps328.OverlayValues[256] = d256
			ps328.OverlayValues[257] = d257
			ps328.OverlayValues[258] = d258
			ps328.OverlayValues[325] = d325
			ps328.OverlayValues[326] = d326
			ps328.OverlayValues[327] = d327
			ps328.PhiValues = make([]JITValueDesc, 2)
			d329 = d2
			ps328.PhiValues[0] = d329
			d330 = d325
			ps328.PhiValues[1] = d330
			if ps328.General && bbs[15].Rendered {
				ctx.EmitJmp(lbl16)
				return result
			}
			return bbs[15].RenderPS(ps328)
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
					ctx.EmitJmp(lbl14)
					return result
				}
				bbs[13].Rendered = true
				bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_13 = bbs[13].Address
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
			}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
				d327 = ps.OverlayValues[327]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
				d330 = ps.OverlayValues[330]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d3)
			var d331 JITValueDesc
			if d3.Loc == LocImm {
				idx := int(d3.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d331 = args[idx]
				d331.ID = 0
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
				lbl38 := ctx.ReserveLabel()
				lbl39 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d3.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl39)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d3.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r18, ai.Reg)
						ctx.EmitMovRegReg(r19, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r18, tmp.Reg)
						ctx.EmitMovRegReg(r19, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r18, Reg2: r19}
						ctx.BindReg(r18, &pair)
						ctx.BindReg(r19, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r18, uint64(ptrWord))
							ctx.EmitMovRegImm64(r19, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl38)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl39)
				d332 := JITValueDesc{Loc: LocRegPair, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d332)
				ctx.BindReg(r19, &d332)
				ctx.BindReg(r18, &d332)
				ctx.BindReg(r19, &d332)
				ctx.EmitMakeNil(d332)
				ctx.MarkLabel(lbl38)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d331 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d331)
				ctx.BindReg(r19, &d331)
			}
			var d333 JITValueDesc
			if d331.Loc == LocImm {
				d333 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d331.Imm.Float())}
			} else if d331.Type == tagFloat && d331.Loc == LocReg {
				d333 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d331.Reg}
				ctx.BindReg(d331.Reg, &d333)
				ctx.BindReg(d331.Reg, &d333)
			} else if d331.Type == tagFloat && d331.Loc == LocRegPair {
				ctx.FreeReg(d331.Reg)
				d333 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d331.Reg2}
				ctx.BindReg(d331.Reg2, &d333)
				ctx.BindReg(d331.Reg2, &d333)
			} else {
				d333 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d331}, 1)
				d333.Type = tagFloat
				ctx.BindReg(d333.Reg, &d333)
			}
			ctx.FreeDesc(&d331)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d333)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d333)
			var d334 JITValueDesc
			if d4.Loc == LocImm && d333.Loc == LocImm {
				d334 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d4.Imm.Float() - d333.Imm.Float())}
			} else if d4.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d333.Reg)
				_, xBits := d4.Imm.RawWords()
				ctx.EmitMovRegImm64(scratch, xBits)
				ctx.EmitSubFloat64(scratch, d333.Reg)
				d334 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d334)
			} else if d333.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(scratch, d4.Reg)
				_, yBits := d333.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, yBits)
				ctx.EmitSubFloat64(scratch, RegR11)
				d334 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d334)
			} else {
				r20 := ctx.AllocRegExcept(d4.Reg, d333.Reg)
				ctx.EmitMovRegReg(r20, d4.Reg)
				ctx.EmitSubFloat64(r20, d333.Reg)
				d334 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r20}
				ctx.BindReg(r20, &d334)
			}
			if d334.Loc == LocReg && d4.Loc == LocReg && d334.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.FreeDesc(&d333)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d335 JITValueDesc
			if d3.Loc == LocImm {
				d335 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.EmitMovRegReg(scratch, d3.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d335 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d335)
			}
			if d335.Loc == LocReg && d3.Loc == LocReg && d335.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = LocNone
			}
			ctx.EnsureDesc(&d334)
			if d334.Loc == LocReg {
				ctx.ProtectReg(d334.Reg)
			} else if d334.Loc == LocRegPair {
				ctx.ProtectReg(d334.Reg)
				ctx.ProtectReg(d334.Reg2)
			}
			ctx.EnsureDesc(&d335)
			if d335.Loc == LocReg {
				ctx.ProtectReg(d335.Reg)
			} else if d335.Loc == LocRegPair {
				ctx.ProtectReg(d335.Reg)
				ctx.ProtectReg(d335.Reg2)
			}
			d336 = d335
			if d336.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d336)
			ctx.EmitStoreToStack(d336, int32(bbs[15].PhiBase)+int32(0))
			d337 = d334
			if d337.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d337)
			ctx.EmitStoreToStack(d337, int32(bbs[15].PhiBase)+int32(16))
			if d334.Loc == LocReg {
				ctx.UnprotectReg(d334.Reg)
			} else if d334.Loc == LocRegPair {
				ctx.UnprotectReg(d334.Reg)
				ctx.UnprotectReg(d334.Reg2)
			}
			if d335.Loc == LocReg {
				ctx.UnprotectReg(d335.Reg)
			} else if d335.Loc == LocRegPair {
				ctx.UnprotectReg(d335.Reg)
				ctx.UnprotectReg(d335.Reg2)
			}
			ps338 := PhiState{General: ps.General}
			ps338.OverlayValues = make([]JITValueDesc, 338)
			ps338.OverlayValues[0] = d0
			ps338.OverlayValues[1] = d1
			ps338.OverlayValues[2] = d2
			ps338.OverlayValues[3] = d3
			ps338.OverlayValues[4] = d4
			ps338.OverlayValues[5] = d5
			ps338.OverlayValues[6] = d6
			ps338.OverlayValues[7] = d7
			ps338.OverlayValues[9] = d9
			ps338.OverlayValues[10] = d10
			ps338.OverlayValues[11] = d11
			ps338.OverlayValues[12] = d12
			ps338.OverlayValues[13] = d13
			ps338.OverlayValues[16] = d16
			ps338.OverlayValues[34] = d34
			ps338.OverlayValues[35] = d35
			ps338.OverlayValues[36] = d36
			ps338.OverlayValues[37] = d37
			ps338.OverlayValues[38] = d38
			ps338.OverlayValues[40] = d40
			ps338.OverlayValues[42] = d42
			ps338.OverlayValues[43] = d43
			ps338.OverlayValues[46] = d46
			ps338.OverlayValues[71] = d71
			ps338.OverlayValues[72] = d72
			ps338.OverlayValues[73] = d73
			ps338.OverlayValues[74] = d74
			ps338.OverlayValues[107] = d107
			ps338.OverlayValues[108] = d108
			ps338.OverlayValues[109] = d109
			ps338.OverlayValues[111] = d111
			ps338.OverlayValues[112] = d112
			ps338.OverlayValues[113] = d113
			ps338.OverlayValues[114] = d114
			ps338.OverlayValues[115] = d115
			ps338.OverlayValues[117] = d117
			ps338.OverlayValues[118] = d118
			ps338.OverlayValues[119] = d119
			ps338.OverlayValues[120] = d120
			ps338.OverlayValues[121] = d121
			ps338.OverlayValues[122] = d122
			ps338.OverlayValues[123] = d123
			ps338.OverlayValues[124] = d124
			ps338.OverlayValues[125] = d125
			ps338.OverlayValues[127] = d127
			ps338.OverlayValues[128] = d128
			ps338.OverlayValues[129] = d129
			ps338.OverlayValues[130] = d130
			ps338.OverlayValues[131] = d131
			ps338.OverlayValues[186] = d186
			ps338.OverlayValues[187] = d187
			ps338.OverlayValues[188] = d188
			ps338.OverlayValues[189] = d189
			ps338.OverlayValues[190] = d190
			ps338.OverlayValues[193] = d193
			ps338.OverlayValues[194] = d194
			ps338.OverlayValues[254] = d254
			ps338.OverlayValues[255] = d255
			ps338.OverlayValues[256] = d256
			ps338.OverlayValues[257] = d257
			ps338.OverlayValues[258] = d258
			ps338.OverlayValues[325] = d325
			ps338.OverlayValues[326] = d326
			ps338.OverlayValues[327] = d327
			ps338.OverlayValues[329] = d329
			ps338.OverlayValues[330] = d330
			ps338.OverlayValues[331] = d331
			ps338.OverlayValues[332] = d332
			ps338.OverlayValues[333] = d333
			ps338.OverlayValues[334] = d334
			ps338.OverlayValues[335] = d335
			ps338.OverlayValues[336] = d336
			ps338.OverlayValues[337] = d337
			ps338.PhiValues = make([]JITValueDesc, 2)
			d339 = d335
			ps338.PhiValues[0] = d339
			d340 = d334
			ps338.PhiValues[1] = d340
			if ps338.General && bbs[15].Rendered {
				ctx.EmitJmp(lbl16)
				return result
			}
			return bbs[15].RenderPS(ps338)
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
					ctx.EmitJmp(lbl15)
					return result
				}
				bbs[14].Rendered = true
				bbs[14].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_14 = bbs[14].Address
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
				d327 = ps.OverlayValues[327]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
				d330 = ps.OverlayValues[330]
			}
			if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
				d331 = ps.OverlayValues[331]
			}
			if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
				d332 = ps.OverlayValues[332]
			}
			if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
				d333 = ps.OverlayValues[333]
			}
			if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
				d334 = ps.OverlayValues[334]
			}
			if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
				d335 = ps.OverlayValues[335]
			}
			if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
				d336 = ps.OverlayValues[336]
			}
			if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
				d337 = ps.OverlayValues[337]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			ctx.EmitMakeFloat(result, d4)
			if d4.Loc == LocReg { ctx.FreeReg(d4.Reg) }
			result.Type = tagFloat
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[15].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d341 := ps.PhiValues[0]
					ctx.EnsureDesc(&d341)
					ctx.EmitStoreToStack(d341, int32(bbs[15].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d342 := ps.PhiValues[1]
					ctx.EnsureDesc(&d342)
					ctx.EmitStoreToStack(d342, int32(bbs[15].PhiBase)+int32(16))
				}
				if bbs[15].VisitCount >= 2 {
					ps.General = true
					return bbs[15].RenderPS(ps)
				}
			}
			bbs[15].VisitCount++
			if ps.General {
				if bbs[15].Rendered {
					ctx.EmitJmp(lbl16)
					return result
				}
				bbs[15].Rendered = true
				bbs[15].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_15 = bbs[15].Address
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
			}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
				d327 = ps.OverlayValues[327]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
				d330 = ps.OverlayValues[330]
			}
			if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
				d331 = ps.OverlayValues[331]
			}
			if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
				d332 = ps.OverlayValues[332]
			}
			if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
				d333 = ps.OverlayValues[333]
			}
			if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
				d334 = ps.OverlayValues[334]
			}
			if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
				d335 = ps.OverlayValues[335]
			}
			if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
				d336 = ps.OverlayValues[336]
			}
			if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
				d337 = ps.OverlayValues[337]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
				d341 = ps.OverlayValues[341]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d3 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d4 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d343 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d343)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d343)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d343)
			var d344 JITValueDesc
			if d3.Loc == LocImm && d343.Loc == LocImm {
				d344 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d3.Imm.Int() < d343.Imm.Int())}
			} else if d343.Loc == LocImm {
				r21 := ctx.AllocRegExcept(d3.Reg)
				if d343.Imm.Int() >= -2147483648 && d343.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d3.Reg, int32(d343.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d343.Imm.Int()))
					ctx.EmitCmpInt64(d3.Reg, RegR11)
				}
				ctx.EmitSetcc(r21, CcL)
				d344 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
				ctx.BindReg(r21, &d344)
			} else if d3.Loc == LocImm {
				r22 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d3.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d343.Reg)
				ctx.EmitSetcc(r22, CcL)
				d344 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
				ctx.BindReg(r22, &d344)
			} else {
				r23 := ctx.AllocRegExcept(d3.Reg)
				ctx.EmitCmpInt64(d3.Reg, d343.Reg)
				ctx.EmitSetcc(r23, CcL)
				d344 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d344)
			}
			ctx.FreeDesc(&d343)
			d345 = d344
			ctx.EnsureDesc(&d345)
			if d345.Loc != LocImm && d345.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d345.Loc == LocImm {
				if d345.Imm.Bool() {
			ps346 := PhiState{General: ps.General}
			ps346.OverlayValues = make([]JITValueDesc, 346)
			ps346.OverlayValues[0] = d0
			ps346.OverlayValues[1] = d1
			ps346.OverlayValues[2] = d2
			ps346.OverlayValues[3] = d3
			ps346.OverlayValues[4] = d4
			ps346.OverlayValues[5] = d5
			ps346.OverlayValues[6] = d6
			ps346.OverlayValues[7] = d7
			ps346.OverlayValues[9] = d9
			ps346.OverlayValues[10] = d10
			ps346.OverlayValues[11] = d11
			ps346.OverlayValues[12] = d12
			ps346.OverlayValues[13] = d13
			ps346.OverlayValues[16] = d16
			ps346.OverlayValues[34] = d34
			ps346.OverlayValues[35] = d35
			ps346.OverlayValues[36] = d36
			ps346.OverlayValues[37] = d37
			ps346.OverlayValues[38] = d38
			ps346.OverlayValues[40] = d40
			ps346.OverlayValues[42] = d42
			ps346.OverlayValues[43] = d43
			ps346.OverlayValues[46] = d46
			ps346.OverlayValues[71] = d71
			ps346.OverlayValues[72] = d72
			ps346.OverlayValues[73] = d73
			ps346.OverlayValues[74] = d74
			ps346.OverlayValues[107] = d107
			ps346.OverlayValues[108] = d108
			ps346.OverlayValues[109] = d109
			ps346.OverlayValues[111] = d111
			ps346.OverlayValues[112] = d112
			ps346.OverlayValues[113] = d113
			ps346.OverlayValues[114] = d114
			ps346.OverlayValues[115] = d115
			ps346.OverlayValues[117] = d117
			ps346.OverlayValues[118] = d118
			ps346.OverlayValues[119] = d119
			ps346.OverlayValues[120] = d120
			ps346.OverlayValues[121] = d121
			ps346.OverlayValues[122] = d122
			ps346.OverlayValues[123] = d123
			ps346.OverlayValues[124] = d124
			ps346.OverlayValues[125] = d125
			ps346.OverlayValues[127] = d127
			ps346.OverlayValues[128] = d128
			ps346.OverlayValues[129] = d129
			ps346.OverlayValues[130] = d130
			ps346.OverlayValues[131] = d131
			ps346.OverlayValues[186] = d186
			ps346.OverlayValues[187] = d187
			ps346.OverlayValues[188] = d188
			ps346.OverlayValues[189] = d189
			ps346.OverlayValues[190] = d190
			ps346.OverlayValues[193] = d193
			ps346.OverlayValues[194] = d194
			ps346.OverlayValues[254] = d254
			ps346.OverlayValues[255] = d255
			ps346.OverlayValues[256] = d256
			ps346.OverlayValues[257] = d257
			ps346.OverlayValues[258] = d258
			ps346.OverlayValues[325] = d325
			ps346.OverlayValues[326] = d326
			ps346.OverlayValues[327] = d327
			ps346.OverlayValues[329] = d329
			ps346.OverlayValues[330] = d330
			ps346.OverlayValues[331] = d331
			ps346.OverlayValues[332] = d332
			ps346.OverlayValues[333] = d333
			ps346.OverlayValues[334] = d334
			ps346.OverlayValues[335] = d335
			ps346.OverlayValues[336] = d336
			ps346.OverlayValues[337] = d337
			ps346.OverlayValues[339] = d339
			ps346.OverlayValues[340] = d340
			ps346.OverlayValues[341] = d341
			ps346.OverlayValues[342] = d342
			ps346.OverlayValues[343] = d343
			ps346.OverlayValues[344] = d344
			ps346.OverlayValues[345] = d345
					return bbs[13].RenderPS(ps346)
				}
			ps347 := PhiState{General: ps.General}
			ps347.OverlayValues = make([]JITValueDesc, 346)
			ps347.OverlayValues[0] = d0
			ps347.OverlayValues[1] = d1
			ps347.OverlayValues[2] = d2
			ps347.OverlayValues[3] = d3
			ps347.OverlayValues[4] = d4
			ps347.OverlayValues[5] = d5
			ps347.OverlayValues[6] = d6
			ps347.OverlayValues[7] = d7
			ps347.OverlayValues[9] = d9
			ps347.OverlayValues[10] = d10
			ps347.OverlayValues[11] = d11
			ps347.OverlayValues[12] = d12
			ps347.OverlayValues[13] = d13
			ps347.OverlayValues[16] = d16
			ps347.OverlayValues[34] = d34
			ps347.OverlayValues[35] = d35
			ps347.OverlayValues[36] = d36
			ps347.OverlayValues[37] = d37
			ps347.OverlayValues[38] = d38
			ps347.OverlayValues[40] = d40
			ps347.OverlayValues[42] = d42
			ps347.OverlayValues[43] = d43
			ps347.OverlayValues[46] = d46
			ps347.OverlayValues[71] = d71
			ps347.OverlayValues[72] = d72
			ps347.OverlayValues[73] = d73
			ps347.OverlayValues[74] = d74
			ps347.OverlayValues[107] = d107
			ps347.OverlayValues[108] = d108
			ps347.OverlayValues[109] = d109
			ps347.OverlayValues[111] = d111
			ps347.OverlayValues[112] = d112
			ps347.OverlayValues[113] = d113
			ps347.OverlayValues[114] = d114
			ps347.OverlayValues[115] = d115
			ps347.OverlayValues[117] = d117
			ps347.OverlayValues[118] = d118
			ps347.OverlayValues[119] = d119
			ps347.OverlayValues[120] = d120
			ps347.OverlayValues[121] = d121
			ps347.OverlayValues[122] = d122
			ps347.OverlayValues[123] = d123
			ps347.OverlayValues[124] = d124
			ps347.OverlayValues[125] = d125
			ps347.OverlayValues[127] = d127
			ps347.OverlayValues[128] = d128
			ps347.OverlayValues[129] = d129
			ps347.OverlayValues[130] = d130
			ps347.OverlayValues[131] = d131
			ps347.OverlayValues[186] = d186
			ps347.OverlayValues[187] = d187
			ps347.OverlayValues[188] = d188
			ps347.OverlayValues[189] = d189
			ps347.OverlayValues[190] = d190
			ps347.OverlayValues[193] = d193
			ps347.OverlayValues[194] = d194
			ps347.OverlayValues[254] = d254
			ps347.OverlayValues[255] = d255
			ps347.OverlayValues[256] = d256
			ps347.OverlayValues[257] = d257
			ps347.OverlayValues[258] = d258
			ps347.OverlayValues[325] = d325
			ps347.OverlayValues[326] = d326
			ps347.OverlayValues[327] = d327
			ps347.OverlayValues[329] = d329
			ps347.OverlayValues[330] = d330
			ps347.OverlayValues[331] = d331
			ps347.OverlayValues[332] = d332
			ps347.OverlayValues[333] = d333
			ps347.OverlayValues[334] = d334
			ps347.OverlayValues[335] = d335
			ps347.OverlayValues[336] = d336
			ps347.OverlayValues[337] = d337
			ps347.OverlayValues[339] = d339
			ps347.OverlayValues[340] = d340
			ps347.OverlayValues[341] = d341
			ps347.OverlayValues[342] = d342
			ps347.OverlayValues[343] = d343
			ps347.OverlayValues[344] = d344
			ps347.OverlayValues[345] = d345
				return bbs[14].RenderPS(ps347)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d348 := ps.PhiValues[0]
					ctx.EnsureDesc(&d348)
					ctx.EmitStoreToStack(d348, int32(bbs[15].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d349 := ps.PhiValues[1]
					ctx.EnsureDesc(&d349)
					ctx.EmitStoreToStack(d349, int32(bbs[15].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[15].RenderPS(ps)
			}
			lbl40 := ctx.ReserveLabel()
			lbl41 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d345.Reg, 0)
			ctx.EmitJcc(CcNE, lbl40)
			ctx.EmitJmp(lbl41)
			ctx.MarkLabel(lbl40)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl41)
			ctx.EmitJmp(lbl15)
			ps350 := PhiState{General: true}
			ps350.OverlayValues = make([]JITValueDesc, 350)
			ps350.OverlayValues[0] = d0
			ps350.OverlayValues[1] = d1
			ps350.OverlayValues[2] = d2
			ps350.OverlayValues[3] = d3
			ps350.OverlayValues[4] = d4
			ps350.OverlayValues[5] = d5
			ps350.OverlayValues[6] = d6
			ps350.OverlayValues[7] = d7
			ps350.OverlayValues[9] = d9
			ps350.OverlayValues[10] = d10
			ps350.OverlayValues[11] = d11
			ps350.OverlayValues[12] = d12
			ps350.OverlayValues[13] = d13
			ps350.OverlayValues[16] = d16
			ps350.OverlayValues[34] = d34
			ps350.OverlayValues[35] = d35
			ps350.OverlayValues[36] = d36
			ps350.OverlayValues[37] = d37
			ps350.OverlayValues[38] = d38
			ps350.OverlayValues[40] = d40
			ps350.OverlayValues[42] = d42
			ps350.OverlayValues[43] = d43
			ps350.OverlayValues[46] = d46
			ps350.OverlayValues[71] = d71
			ps350.OverlayValues[72] = d72
			ps350.OverlayValues[73] = d73
			ps350.OverlayValues[74] = d74
			ps350.OverlayValues[107] = d107
			ps350.OverlayValues[108] = d108
			ps350.OverlayValues[109] = d109
			ps350.OverlayValues[111] = d111
			ps350.OverlayValues[112] = d112
			ps350.OverlayValues[113] = d113
			ps350.OverlayValues[114] = d114
			ps350.OverlayValues[115] = d115
			ps350.OverlayValues[117] = d117
			ps350.OverlayValues[118] = d118
			ps350.OverlayValues[119] = d119
			ps350.OverlayValues[120] = d120
			ps350.OverlayValues[121] = d121
			ps350.OverlayValues[122] = d122
			ps350.OverlayValues[123] = d123
			ps350.OverlayValues[124] = d124
			ps350.OverlayValues[125] = d125
			ps350.OverlayValues[127] = d127
			ps350.OverlayValues[128] = d128
			ps350.OverlayValues[129] = d129
			ps350.OverlayValues[130] = d130
			ps350.OverlayValues[131] = d131
			ps350.OverlayValues[186] = d186
			ps350.OverlayValues[187] = d187
			ps350.OverlayValues[188] = d188
			ps350.OverlayValues[189] = d189
			ps350.OverlayValues[190] = d190
			ps350.OverlayValues[193] = d193
			ps350.OverlayValues[194] = d194
			ps350.OverlayValues[254] = d254
			ps350.OverlayValues[255] = d255
			ps350.OverlayValues[256] = d256
			ps350.OverlayValues[257] = d257
			ps350.OverlayValues[258] = d258
			ps350.OverlayValues[325] = d325
			ps350.OverlayValues[326] = d326
			ps350.OverlayValues[327] = d327
			ps350.OverlayValues[329] = d329
			ps350.OverlayValues[330] = d330
			ps350.OverlayValues[331] = d331
			ps350.OverlayValues[332] = d332
			ps350.OverlayValues[333] = d333
			ps350.OverlayValues[334] = d334
			ps350.OverlayValues[335] = d335
			ps350.OverlayValues[336] = d336
			ps350.OverlayValues[337] = d337
			ps350.OverlayValues[339] = d339
			ps350.OverlayValues[340] = d340
			ps350.OverlayValues[341] = d341
			ps350.OverlayValues[342] = d342
			ps350.OverlayValues[343] = d343
			ps350.OverlayValues[344] = d344
			ps350.OverlayValues[345] = d345
			ps350.OverlayValues[348] = d348
			ps350.OverlayValues[349] = d349
			ps351 := PhiState{General: true}
			ps351.OverlayValues = make([]JITValueDesc, 350)
			ps351.OverlayValues[0] = d0
			ps351.OverlayValues[1] = d1
			ps351.OverlayValues[2] = d2
			ps351.OverlayValues[3] = d3
			ps351.OverlayValues[4] = d4
			ps351.OverlayValues[5] = d5
			ps351.OverlayValues[6] = d6
			ps351.OverlayValues[7] = d7
			ps351.OverlayValues[9] = d9
			ps351.OverlayValues[10] = d10
			ps351.OverlayValues[11] = d11
			ps351.OverlayValues[12] = d12
			ps351.OverlayValues[13] = d13
			ps351.OverlayValues[16] = d16
			ps351.OverlayValues[34] = d34
			ps351.OverlayValues[35] = d35
			ps351.OverlayValues[36] = d36
			ps351.OverlayValues[37] = d37
			ps351.OverlayValues[38] = d38
			ps351.OverlayValues[40] = d40
			ps351.OverlayValues[42] = d42
			ps351.OverlayValues[43] = d43
			ps351.OverlayValues[46] = d46
			ps351.OverlayValues[71] = d71
			ps351.OverlayValues[72] = d72
			ps351.OverlayValues[73] = d73
			ps351.OverlayValues[74] = d74
			ps351.OverlayValues[107] = d107
			ps351.OverlayValues[108] = d108
			ps351.OverlayValues[109] = d109
			ps351.OverlayValues[111] = d111
			ps351.OverlayValues[112] = d112
			ps351.OverlayValues[113] = d113
			ps351.OverlayValues[114] = d114
			ps351.OverlayValues[115] = d115
			ps351.OverlayValues[117] = d117
			ps351.OverlayValues[118] = d118
			ps351.OverlayValues[119] = d119
			ps351.OverlayValues[120] = d120
			ps351.OverlayValues[121] = d121
			ps351.OverlayValues[122] = d122
			ps351.OverlayValues[123] = d123
			ps351.OverlayValues[124] = d124
			ps351.OverlayValues[125] = d125
			ps351.OverlayValues[127] = d127
			ps351.OverlayValues[128] = d128
			ps351.OverlayValues[129] = d129
			ps351.OverlayValues[130] = d130
			ps351.OverlayValues[131] = d131
			ps351.OverlayValues[186] = d186
			ps351.OverlayValues[187] = d187
			ps351.OverlayValues[188] = d188
			ps351.OverlayValues[189] = d189
			ps351.OverlayValues[190] = d190
			ps351.OverlayValues[193] = d193
			ps351.OverlayValues[194] = d194
			ps351.OverlayValues[254] = d254
			ps351.OverlayValues[255] = d255
			ps351.OverlayValues[256] = d256
			ps351.OverlayValues[257] = d257
			ps351.OverlayValues[258] = d258
			ps351.OverlayValues[325] = d325
			ps351.OverlayValues[326] = d326
			ps351.OverlayValues[327] = d327
			ps351.OverlayValues[329] = d329
			ps351.OverlayValues[330] = d330
			ps351.OverlayValues[331] = d331
			ps351.OverlayValues[332] = d332
			ps351.OverlayValues[333] = d333
			ps351.OverlayValues[334] = d334
			ps351.OverlayValues[335] = d335
			ps351.OverlayValues[336] = d336
			ps351.OverlayValues[337] = d337
			ps351.OverlayValues[339] = d339
			ps351.OverlayValues[340] = d340
			ps351.OverlayValues[341] = d341
			ps351.OverlayValues[342] = d342
			ps351.OverlayValues[343] = d343
			ps351.OverlayValues[344] = d344
			ps351.OverlayValues[345] = d345
			ps351.OverlayValues[348] = d348
			ps351.OverlayValues[349] = d349
			snap352 := d0
			snap353 := d1
			snap354 := d2
			snap355 := d3
			snap356 := d4
			snap357 := d5
			snap358 := d6
			snap359 := d7
			snap360 := d9
			snap361 := d10
			snap362 := d11
			snap363 := d12
			snap364 := d13
			snap365 := d16
			snap366 := d34
			snap367 := d35
			snap368 := d36
			snap369 := d37
			snap370 := d38
			snap371 := d40
			snap372 := d42
			snap373 := d43
			snap374 := d46
			snap375 := d71
			snap376 := d72
			snap377 := d73
			snap378 := d74
			snap379 := d107
			snap380 := d108
			snap381 := d109
			snap382 := d111
			snap383 := d112
			snap384 := d113
			snap385 := d114
			snap386 := d115
			snap387 := d117
			snap388 := d118
			snap389 := d119
			snap390 := d120
			snap391 := d121
			snap392 := d122
			snap393 := d123
			snap394 := d124
			snap395 := d125
			snap396 := d127
			snap397 := d128
			snap398 := d129
			snap399 := d130
			snap400 := d131
			snap401 := d186
			snap402 := d187
			snap403 := d188
			snap404 := d189
			snap405 := d190
			snap406 := d193
			snap407 := d194
			snap408 := d254
			snap409 := d255
			snap410 := d256
			snap411 := d257
			snap412 := d258
			snap413 := d325
			snap414 := d326
			snap415 := d327
			snap416 := d329
			snap417 := d330
			snap418 := d331
			snap419 := d332
			snap420 := d333
			snap421 := d334
			snap422 := d335
			snap423 := d336
			snap424 := d337
			snap425 := d339
			snap426 := d340
			snap427 := d341
			snap428 := d342
			snap429 := d343
			snap430 := d344
			snap431 := d345
			snap432 := d348
			snap433 := d349
			alloc434 := ctx.SnapshotAllocState()
			if !bbs[14].Rendered {
				bbs[14].RenderPS(ps351)
			}
			ctx.RestoreAllocState(alloc434)
			d0 = snap352
			d1 = snap353
			d2 = snap354
			d3 = snap355
			d4 = snap356
			d5 = snap357
			d6 = snap358
			d7 = snap359
			d9 = snap360
			d10 = snap361
			d11 = snap362
			d12 = snap363
			d13 = snap364
			d16 = snap365
			d34 = snap366
			d35 = snap367
			d36 = snap368
			d37 = snap369
			d38 = snap370
			d40 = snap371
			d42 = snap372
			d43 = snap373
			d46 = snap374
			d71 = snap375
			d72 = snap376
			d73 = snap377
			d74 = snap378
			d107 = snap379
			d108 = snap380
			d109 = snap381
			d111 = snap382
			d112 = snap383
			d113 = snap384
			d114 = snap385
			d115 = snap386
			d117 = snap387
			d118 = snap388
			d119 = snap389
			d120 = snap390
			d121 = snap391
			d122 = snap392
			d123 = snap393
			d124 = snap394
			d125 = snap395
			d127 = snap396
			d128 = snap397
			d129 = snap398
			d130 = snap399
			d131 = snap400
			d186 = snap401
			d187 = snap402
			d188 = snap403
			d189 = snap404
			d190 = snap405
			d193 = snap406
			d194 = snap407
			d254 = snap408
			d255 = snap409
			d256 = snap410
			d257 = snap411
			d258 = snap412
			d325 = snap413
			d326 = snap414
			d327 = snap415
			d329 = snap416
			d330 = snap417
			d331 = snap418
			d332 = snap419
			d333 = snap420
			d334 = snap421
			d335 = snap422
			d336 = snap423
			d337 = snap424
			d339 = snap425
			d340 = snap426
			d341 = snap427
			d342 = snap428
			d343 = snap429
			d344 = snap430
			d345 = snap431
			d348 = snap432
			d349 = snap433
			if !bbs[13].Rendered {
				return bbs[13].RenderPS(ps350)
			}
			return result
			ctx.FreeDesc(&d344)
			return result
			}
			bbs[16].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d435 := ps.PhiValues[0]
					ctx.EnsureDesc(&d435)
					ctx.EmitStoreToStack(d435, int32(bbs[16].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d436 := ps.PhiValues[1]
					ctx.EnsureDesc(&d436)
					ctx.EmitStoreToStack(d436, int32(bbs[16].PhiBase)+int32(16))
				}
				if bbs[16].VisitCount >= 2 {
					ps.General = true
					return bbs[16].RenderPS(ps)
				}
			}
			bbs[16].VisitCount++
			if ps.General {
				if bbs[16].Rendered {
					ctx.EmitJmp(lbl17)
					return result
				}
				bbs[16].Rendered = true
				bbs[16].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_16 = bbs[16].Address
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
				d327 = ps.OverlayValues[327]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
				d330 = ps.OverlayValues[330]
			}
			if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
				d331 = ps.OverlayValues[331]
			}
			if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
				d332 = ps.OverlayValues[332]
			}
			if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
				d333 = ps.OverlayValues[333]
			}
			if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
				d334 = ps.OverlayValues[334]
			}
			if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
				d335 = ps.OverlayValues[335]
			}
			if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
				d336 = ps.OverlayValues[336]
			}
			if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
				d337 = ps.OverlayValues[337]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
				d341 = ps.OverlayValues[341]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
				d343 = ps.OverlayValues[343]
			}
			if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
				d344 = ps.OverlayValues[344]
			}
			if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
				d345 = ps.OverlayValues[345]
			}
			if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
				d348 = ps.OverlayValues[348]
			}
			if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
				d349 = ps.OverlayValues[349]
			}
			if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
				d435 = ps.OverlayValues[435]
			}
			if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != LocNone {
				d436 = ps.OverlayValues[436]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d5 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d6 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d437 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d437)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d437)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d437)
			var d438 JITValueDesc
			if d6.Loc == LocImm && d437.Loc == LocImm {
				d438 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d6.Imm.Int() < d437.Imm.Int())}
			} else if d437.Loc == LocImm {
				r24 := ctx.AllocRegExcept(d6.Reg)
				if d437.Imm.Int() >= -2147483648 && d437.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d6.Reg, int32(d437.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d437.Imm.Int()))
					ctx.EmitCmpInt64(d6.Reg, RegR11)
				}
				ctx.EmitSetcc(r24, CcL)
				d438 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
				ctx.BindReg(r24, &d438)
			} else if d6.Loc == LocImm {
				r25 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d437.Reg)
				ctx.EmitSetcc(r25, CcL)
				d438 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
				ctx.BindReg(r25, &d438)
			} else {
				r26 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitCmpInt64(d6.Reg, d437.Reg)
				ctx.EmitSetcc(r26, CcL)
				d438 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
				ctx.BindReg(r26, &d438)
			}
			ctx.FreeDesc(&d437)
			d439 = d438
			ctx.EnsureDesc(&d439)
			if d439.Loc != LocImm && d439.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d439.Loc == LocImm {
				if d439.Imm.Bool() {
			ps440 := PhiState{General: ps.General}
			ps440.OverlayValues = make([]JITValueDesc, 440)
			ps440.OverlayValues[0] = d0
			ps440.OverlayValues[1] = d1
			ps440.OverlayValues[2] = d2
			ps440.OverlayValues[3] = d3
			ps440.OverlayValues[4] = d4
			ps440.OverlayValues[5] = d5
			ps440.OverlayValues[6] = d6
			ps440.OverlayValues[7] = d7
			ps440.OverlayValues[9] = d9
			ps440.OverlayValues[10] = d10
			ps440.OverlayValues[11] = d11
			ps440.OverlayValues[12] = d12
			ps440.OverlayValues[13] = d13
			ps440.OverlayValues[16] = d16
			ps440.OverlayValues[34] = d34
			ps440.OverlayValues[35] = d35
			ps440.OverlayValues[36] = d36
			ps440.OverlayValues[37] = d37
			ps440.OverlayValues[38] = d38
			ps440.OverlayValues[40] = d40
			ps440.OverlayValues[42] = d42
			ps440.OverlayValues[43] = d43
			ps440.OverlayValues[46] = d46
			ps440.OverlayValues[71] = d71
			ps440.OverlayValues[72] = d72
			ps440.OverlayValues[73] = d73
			ps440.OverlayValues[74] = d74
			ps440.OverlayValues[107] = d107
			ps440.OverlayValues[108] = d108
			ps440.OverlayValues[109] = d109
			ps440.OverlayValues[111] = d111
			ps440.OverlayValues[112] = d112
			ps440.OverlayValues[113] = d113
			ps440.OverlayValues[114] = d114
			ps440.OverlayValues[115] = d115
			ps440.OverlayValues[117] = d117
			ps440.OverlayValues[118] = d118
			ps440.OverlayValues[119] = d119
			ps440.OverlayValues[120] = d120
			ps440.OverlayValues[121] = d121
			ps440.OverlayValues[122] = d122
			ps440.OverlayValues[123] = d123
			ps440.OverlayValues[124] = d124
			ps440.OverlayValues[125] = d125
			ps440.OverlayValues[127] = d127
			ps440.OverlayValues[128] = d128
			ps440.OverlayValues[129] = d129
			ps440.OverlayValues[130] = d130
			ps440.OverlayValues[131] = d131
			ps440.OverlayValues[186] = d186
			ps440.OverlayValues[187] = d187
			ps440.OverlayValues[188] = d188
			ps440.OverlayValues[189] = d189
			ps440.OverlayValues[190] = d190
			ps440.OverlayValues[193] = d193
			ps440.OverlayValues[194] = d194
			ps440.OverlayValues[254] = d254
			ps440.OverlayValues[255] = d255
			ps440.OverlayValues[256] = d256
			ps440.OverlayValues[257] = d257
			ps440.OverlayValues[258] = d258
			ps440.OverlayValues[325] = d325
			ps440.OverlayValues[326] = d326
			ps440.OverlayValues[327] = d327
			ps440.OverlayValues[329] = d329
			ps440.OverlayValues[330] = d330
			ps440.OverlayValues[331] = d331
			ps440.OverlayValues[332] = d332
			ps440.OverlayValues[333] = d333
			ps440.OverlayValues[334] = d334
			ps440.OverlayValues[335] = d335
			ps440.OverlayValues[336] = d336
			ps440.OverlayValues[337] = d337
			ps440.OverlayValues[339] = d339
			ps440.OverlayValues[340] = d340
			ps440.OverlayValues[341] = d341
			ps440.OverlayValues[342] = d342
			ps440.OverlayValues[343] = d343
			ps440.OverlayValues[344] = d344
			ps440.OverlayValues[345] = d345
			ps440.OverlayValues[348] = d348
			ps440.OverlayValues[349] = d349
			ps440.OverlayValues[435] = d435
			ps440.OverlayValues[436] = d436
			ps440.OverlayValues[437] = d437
			ps440.OverlayValues[438] = d438
			ps440.OverlayValues[439] = d439
					return bbs[17].RenderPS(ps440)
				}
			ps441 := PhiState{General: ps.General}
			ps441.OverlayValues = make([]JITValueDesc, 440)
			ps441.OverlayValues[0] = d0
			ps441.OverlayValues[1] = d1
			ps441.OverlayValues[2] = d2
			ps441.OverlayValues[3] = d3
			ps441.OverlayValues[4] = d4
			ps441.OverlayValues[5] = d5
			ps441.OverlayValues[6] = d6
			ps441.OverlayValues[7] = d7
			ps441.OverlayValues[9] = d9
			ps441.OverlayValues[10] = d10
			ps441.OverlayValues[11] = d11
			ps441.OverlayValues[12] = d12
			ps441.OverlayValues[13] = d13
			ps441.OverlayValues[16] = d16
			ps441.OverlayValues[34] = d34
			ps441.OverlayValues[35] = d35
			ps441.OverlayValues[36] = d36
			ps441.OverlayValues[37] = d37
			ps441.OverlayValues[38] = d38
			ps441.OverlayValues[40] = d40
			ps441.OverlayValues[42] = d42
			ps441.OverlayValues[43] = d43
			ps441.OverlayValues[46] = d46
			ps441.OverlayValues[71] = d71
			ps441.OverlayValues[72] = d72
			ps441.OverlayValues[73] = d73
			ps441.OverlayValues[74] = d74
			ps441.OverlayValues[107] = d107
			ps441.OverlayValues[108] = d108
			ps441.OverlayValues[109] = d109
			ps441.OverlayValues[111] = d111
			ps441.OverlayValues[112] = d112
			ps441.OverlayValues[113] = d113
			ps441.OverlayValues[114] = d114
			ps441.OverlayValues[115] = d115
			ps441.OverlayValues[117] = d117
			ps441.OverlayValues[118] = d118
			ps441.OverlayValues[119] = d119
			ps441.OverlayValues[120] = d120
			ps441.OverlayValues[121] = d121
			ps441.OverlayValues[122] = d122
			ps441.OverlayValues[123] = d123
			ps441.OverlayValues[124] = d124
			ps441.OverlayValues[125] = d125
			ps441.OverlayValues[127] = d127
			ps441.OverlayValues[128] = d128
			ps441.OverlayValues[129] = d129
			ps441.OverlayValues[130] = d130
			ps441.OverlayValues[131] = d131
			ps441.OverlayValues[186] = d186
			ps441.OverlayValues[187] = d187
			ps441.OverlayValues[188] = d188
			ps441.OverlayValues[189] = d189
			ps441.OverlayValues[190] = d190
			ps441.OverlayValues[193] = d193
			ps441.OverlayValues[194] = d194
			ps441.OverlayValues[254] = d254
			ps441.OverlayValues[255] = d255
			ps441.OverlayValues[256] = d256
			ps441.OverlayValues[257] = d257
			ps441.OverlayValues[258] = d258
			ps441.OverlayValues[325] = d325
			ps441.OverlayValues[326] = d326
			ps441.OverlayValues[327] = d327
			ps441.OverlayValues[329] = d329
			ps441.OverlayValues[330] = d330
			ps441.OverlayValues[331] = d331
			ps441.OverlayValues[332] = d332
			ps441.OverlayValues[333] = d333
			ps441.OverlayValues[334] = d334
			ps441.OverlayValues[335] = d335
			ps441.OverlayValues[336] = d336
			ps441.OverlayValues[337] = d337
			ps441.OverlayValues[339] = d339
			ps441.OverlayValues[340] = d340
			ps441.OverlayValues[341] = d341
			ps441.OverlayValues[342] = d342
			ps441.OverlayValues[343] = d343
			ps441.OverlayValues[344] = d344
			ps441.OverlayValues[345] = d345
			ps441.OverlayValues[348] = d348
			ps441.OverlayValues[349] = d349
			ps441.OverlayValues[435] = d435
			ps441.OverlayValues[436] = d436
			ps441.OverlayValues[437] = d437
			ps441.OverlayValues[438] = d438
			ps441.OverlayValues[439] = d439
				return bbs[18].RenderPS(ps441)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d442 := ps.PhiValues[0]
					ctx.EnsureDesc(&d442)
					ctx.EmitStoreToStack(d442, int32(bbs[16].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d443 := ps.PhiValues[1]
					ctx.EnsureDesc(&d443)
					ctx.EmitStoreToStack(d443, int32(bbs[16].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[16].RenderPS(ps)
			}
			lbl42 := ctx.ReserveLabel()
			lbl43 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d439.Reg, 0)
			ctx.EmitJcc(CcNE, lbl42)
			ctx.EmitJmp(lbl43)
			ctx.MarkLabel(lbl42)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl43)
			ctx.EmitJmp(lbl19)
			ps444 := PhiState{General: true}
			ps444.OverlayValues = make([]JITValueDesc, 444)
			ps444.OverlayValues[0] = d0
			ps444.OverlayValues[1] = d1
			ps444.OverlayValues[2] = d2
			ps444.OverlayValues[3] = d3
			ps444.OverlayValues[4] = d4
			ps444.OverlayValues[5] = d5
			ps444.OverlayValues[6] = d6
			ps444.OverlayValues[7] = d7
			ps444.OverlayValues[9] = d9
			ps444.OverlayValues[10] = d10
			ps444.OverlayValues[11] = d11
			ps444.OverlayValues[12] = d12
			ps444.OverlayValues[13] = d13
			ps444.OverlayValues[16] = d16
			ps444.OverlayValues[34] = d34
			ps444.OverlayValues[35] = d35
			ps444.OverlayValues[36] = d36
			ps444.OverlayValues[37] = d37
			ps444.OverlayValues[38] = d38
			ps444.OverlayValues[40] = d40
			ps444.OverlayValues[42] = d42
			ps444.OverlayValues[43] = d43
			ps444.OverlayValues[46] = d46
			ps444.OverlayValues[71] = d71
			ps444.OverlayValues[72] = d72
			ps444.OverlayValues[73] = d73
			ps444.OverlayValues[74] = d74
			ps444.OverlayValues[107] = d107
			ps444.OverlayValues[108] = d108
			ps444.OverlayValues[109] = d109
			ps444.OverlayValues[111] = d111
			ps444.OverlayValues[112] = d112
			ps444.OverlayValues[113] = d113
			ps444.OverlayValues[114] = d114
			ps444.OverlayValues[115] = d115
			ps444.OverlayValues[117] = d117
			ps444.OverlayValues[118] = d118
			ps444.OverlayValues[119] = d119
			ps444.OverlayValues[120] = d120
			ps444.OverlayValues[121] = d121
			ps444.OverlayValues[122] = d122
			ps444.OverlayValues[123] = d123
			ps444.OverlayValues[124] = d124
			ps444.OverlayValues[125] = d125
			ps444.OverlayValues[127] = d127
			ps444.OverlayValues[128] = d128
			ps444.OverlayValues[129] = d129
			ps444.OverlayValues[130] = d130
			ps444.OverlayValues[131] = d131
			ps444.OverlayValues[186] = d186
			ps444.OverlayValues[187] = d187
			ps444.OverlayValues[188] = d188
			ps444.OverlayValues[189] = d189
			ps444.OverlayValues[190] = d190
			ps444.OverlayValues[193] = d193
			ps444.OverlayValues[194] = d194
			ps444.OverlayValues[254] = d254
			ps444.OverlayValues[255] = d255
			ps444.OverlayValues[256] = d256
			ps444.OverlayValues[257] = d257
			ps444.OverlayValues[258] = d258
			ps444.OverlayValues[325] = d325
			ps444.OverlayValues[326] = d326
			ps444.OverlayValues[327] = d327
			ps444.OverlayValues[329] = d329
			ps444.OverlayValues[330] = d330
			ps444.OverlayValues[331] = d331
			ps444.OverlayValues[332] = d332
			ps444.OverlayValues[333] = d333
			ps444.OverlayValues[334] = d334
			ps444.OverlayValues[335] = d335
			ps444.OverlayValues[336] = d336
			ps444.OverlayValues[337] = d337
			ps444.OverlayValues[339] = d339
			ps444.OverlayValues[340] = d340
			ps444.OverlayValues[341] = d341
			ps444.OverlayValues[342] = d342
			ps444.OverlayValues[343] = d343
			ps444.OverlayValues[344] = d344
			ps444.OverlayValues[345] = d345
			ps444.OverlayValues[348] = d348
			ps444.OverlayValues[349] = d349
			ps444.OverlayValues[435] = d435
			ps444.OverlayValues[436] = d436
			ps444.OverlayValues[437] = d437
			ps444.OverlayValues[438] = d438
			ps444.OverlayValues[439] = d439
			ps444.OverlayValues[442] = d442
			ps444.OverlayValues[443] = d443
			ps445 := PhiState{General: true}
			ps445.OverlayValues = make([]JITValueDesc, 444)
			ps445.OverlayValues[0] = d0
			ps445.OverlayValues[1] = d1
			ps445.OverlayValues[2] = d2
			ps445.OverlayValues[3] = d3
			ps445.OverlayValues[4] = d4
			ps445.OverlayValues[5] = d5
			ps445.OverlayValues[6] = d6
			ps445.OverlayValues[7] = d7
			ps445.OverlayValues[9] = d9
			ps445.OverlayValues[10] = d10
			ps445.OverlayValues[11] = d11
			ps445.OverlayValues[12] = d12
			ps445.OverlayValues[13] = d13
			ps445.OverlayValues[16] = d16
			ps445.OverlayValues[34] = d34
			ps445.OverlayValues[35] = d35
			ps445.OverlayValues[36] = d36
			ps445.OverlayValues[37] = d37
			ps445.OverlayValues[38] = d38
			ps445.OverlayValues[40] = d40
			ps445.OverlayValues[42] = d42
			ps445.OverlayValues[43] = d43
			ps445.OverlayValues[46] = d46
			ps445.OverlayValues[71] = d71
			ps445.OverlayValues[72] = d72
			ps445.OverlayValues[73] = d73
			ps445.OverlayValues[74] = d74
			ps445.OverlayValues[107] = d107
			ps445.OverlayValues[108] = d108
			ps445.OverlayValues[109] = d109
			ps445.OverlayValues[111] = d111
			ps445.OverlayValues[112] = d112
			ps445.OverlayValues[113] = d113
			ps445.OverlayValues[114] = d114
			ps445.OverlayValues[115] = d115
			ps445.OverlayValues[117] = d117
			ps445.OverlayValues[118] = d118
			ps445.OverlayValues[119] = d119
			ps445.OverlayValues[120] = d120
			ps445.OverlayValues[121] = d121
			ps445.OverlayValues[122] = d122
			ps445.OverlayValues[123] = d123
			ps445.OverlayValues[124] = d124
			ps445.OverlayValues[125] = d125
			ps445.OverlayValues[127] = d127
			ps445.OverlayValues[128] = d128
			ps445.OverlayValues[129] = d129
			ps445.OverlayValues[130] = d130
			ps445.OverlayValues[131] = d131
			ps445.OverlayValues[186] = d186
			ps445.OverlayValues[187] = d187
			ps445.OverlayValues[188] = d188
			ps445.OverlayValues[189] = d189
			ps445.OverlayValues[190] = d190
			ps445.OverlayValues[193] = d193
			ps445.OverlayValues[194] = d194
			ps445.OverlayValues[254] = d254
			ps445.OverlayValues[255] = d255
			ps445.OverlayValues[256] = d256
			ps445.OverlayValues[257] = d257
			ps445.OverlayValues[258] = d258
			ps445.OverlayValues[325] = d325
			ps445.OverlayValues[326] = d326
			ps445.OverlayValues[327] = d327
			ps445.OverlayValues[329] = d329
			ps445.OverlayValues[330] = d330
			ps445.OverlayValues[331] = d331
			ps445.OverlayValues[332] = d332
			ps445.OverlayValues[333] = d333
			ps445.OverlayValues[334] = d334
			ps445.OverlayValues[335] = d335
			ps445.OverlayValues[336] = d336
			ps445.OverlayValues[337] = d337
			ps445.OverlayValues[339] = d339
			ps445.OverlayValues[340] = d340
			ps445.OverlayValues[341] = d341
			ps445.OverlayValues[342] = d342
			ps445.OverlayValues[343] = d343
			ps445.OverlayValues[344] = d344
			ps445.OverlayValues[345] = d345
			ps445.OverlayValues[348] = d348
			ps445.OverlayValues[349] = d349
			ps445.OverlayValues[435] = d435
			ps445.OverlayValues[436] = d436
			ps445.OverlayValues[437] = d437
			ps445.OverlayValues[438] = d438
			ps445.OverlayValues[439] = d439
			ps445.OverlayValues[442] = d442
			ps445.OverlayValues[443] = d443
			snap446 := d0
			snap447 := d1
			snap448 := d2
			snap449 := d3
			snap450 := d4
			snap451 := d5
			snap452 := d6
			snap453 := d7
			snap454 := d9
			snap455 := d10
			snap456 := d11
			snap457 := d12
			snap458 := d13
			snap459 := d16
			snap460 := d34
			snap461 := d35
			snap462 := d36
			snap463 := d37
			snap464 := d38
			snap465 := d40
			snap466 := d42
			snap467 := d43
			snap468 := d46
			snap469 := d71
			snap470 := d72
			snap471 := d73
			snap472 := d74
			snap473 := d107
			snap474 := d108
			snap475 := d109
			snap476 := d111
			snap477 := d112
			snap478 := d113
			snap479 := d114
			snap480 := d115
			snap481 := d117
			snap482 := d118
			snap483 := d119
			snap484 := d120
			snap485 := d121
			snap486 := d122
			snap487 := d123
			snap488 := d124
			snap489 := d125
			snap490 := d127
			snap491 := d128
			snap492 := d129
			snap493 := d130
			snap494 := d131
			snap495 := d186
			snap496 := d187
			snap497 := d188
			snap498 := d189
			snap499 := d190
			snap500 := d193
			snap501 := d194
			snap502 := d254
			snap503 := d255
			snap504 := d256
			snap505 := d257
			snap506 := d258
			snap507 := d325
			snap508 := d326
			snap509 := d327
			snap510 := d329
			snap511 := d330
			snap512 := d331
			snap513 := d332
			snap514 := d333
			snap515 := d334
			snap516 := d335
			snap517 := d336
			snap518 := d337
			snap519 := d339
			snap520 := d340
			snap521 := d341
			snap522 := d342
			snap523 := d343
			snap524 := d344
			snap525 := d345
			snap526 := d348
			snap527 := d349
			snap528 := d435
			snap529 := d436
			snap530 := d437
			snap531 := d438
			snap532 := d439
			snap533 := d442
			snap534 := d443
			alloc535 := ctx.SnapshotAllocState()
			if !bbs[18].Rendered {
				bbs[18].RenderPS(ps445)
			}
			ctx.RestoreAllocState(alloc535)
			d0 = snap446
			d1 = snap447
			d2 = snap448
			d3 = snap449
			d4 = snap450
			d5 = snap451
			d6 = snap452
			d7 = snap453
			d9 = snap454
			d10 = snap455
			d11 = snap456
			d12 = snap457
			d13 = snap458
			d16 = snap459
			d34 = snap460
			d35 = snap461
			d36 = snap462
			d37 = snap463
			d38 = snap464
			d40 = snap465
			d42 = snap466
			d43 = snap467
			d46 = snap468
			d71 = snap469
			d72 = snap470
			d73 = snap471
			d74 = snap472
			d107 = snap473
			d108 = snap474
			d109 = snap475
			d111 = snap476
			d112 = snap477
			d113 = snap478
			d114 = snap479
			d115 = snap480
			d117 = snap481
			d118 = snap482
			d119 = snap483
			d120 = snap484
			d121 = snap485
			d122 = snap486
			d123 = snap487
			d124 = snap488
			d125 = snap489
			d127 = snap490
			d128 = snap491
			d129 = snap492
			d130 = snap493
			d131 = snap494
			d186 = snap495
			d187 = snap496
			d188 = snap497
			d189 = snap498
			d190 = snap499
			d193 = snap500
			d194 = snap501
			d254 = snap502
			d255 = snap503
			d256 = snap504
			d257 = snap505
			d258 = snap506
			d325 = snap507
			d326 = snap508
			d327 = snap509
			d329 = snap510
			d330 = snap511
			d331 = snap512
			d332 = snap513
			d333 = snap514
			d334 = snap515
			d335 = snap516
			d336 = snap517
			d337 = snap518
			d339 = snap519
			d340 = snap520
			d341 = snap521
			d342 = snap522
			d343 = snap523
			d344 = snap524
			d345 = snap525
			d348 = snap526
			d349 = snap527
			d435 = snap528
			d436 = snap529
			d437 = snap530
			d438 = snap531
			d439 = snap532
			d442 = snap533
			d443 = snap534
			if !bbs[17].Rendered {
				return bbs[17].RenderPS(ps444)
			}
			return result
			ctx.FreeDesc(&d438)
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
					ctx.EmitJmp(lbl18)
					return result
				}
				bbs[17].Rendered = true
				bbs[17].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_17 = bbs[17].Address
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
				d327 = ps.OverlayValues[327]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
				d330 = ps.OverlayValues[330]
			}
			if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
				d331 = ps.OverlayValues[331]
			}
			if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
				d332 = ps.OverlayValues[332]
			}
			if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
				d333 = ps.OverlayValues[333]
			}
			if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
				d334 = ps.OverlayValues[334]
			}
			if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
				d335 = ps.OverlayValues[335]
			}
			if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
				d336 = ps.OverlayValues[336]
			}
			if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
				d337 = ps.OverlayValues[337]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
				d341 = ps.OverlayValues[341]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
				d343 = ps.OverlayValues[343]
			}
			if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
				d344 = ps.OverlayValues[344]
			}
			if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
				d345 = ps.OverlayValues[345]
			}
			if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
				d348 = ps.OverlayValues[348]
			}
			if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
				d349 = ps.OverlayValues[349]
			}
			if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
				d435 = ps.OverlayValues[435]
			}
			if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != LocNone {
				d436 = ps.OverlayValues[436]
			}
			if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != LocNone {
				d437 = ps.OverlayValues[437]
			}
			if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != LocNone {
				d438 = ps.OverlayValues[438]
			}
			if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != LocNone {
				d439 = ps.OverlayValues[439]
			}
			if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
				d442 = ps.OverlayValues[442]
			}
			if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != LocNone {
				d443 = ps.OverlayValues[443]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d6)
			var d536 JITValueDesc
			if d6.Loc == LocImm {
				idx := int(d6.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d536 = args[idx]
				d536.ID = 0
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
				lbl44 := ctx.ReserveLabel()
				lbl45 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d6.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl45)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d6.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r27, ai.Reg)
						ctx.EmitMovRegReg(r28, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r27, tmp.Reg)
						ctx.EmitMovRegReg(r28, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r27, Reg2: r28}
						ctx.BindReg(r27, &pair)
						ctx.BindReg(r28, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r27, uint64(ptrWord))
							ctx.EmitMovRegImm64(r28, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl44)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl45)
				d537 := JITValueDesc{Loc: LocRegPair, Reg: r27, Reg2: r28}
				ctx.BindReg(r27, &d537)
				ctx.BindReg(r28, &d537)
				ctx.BindReg(r27, &d537)
				ctx.BindReg(r28, &d537)
				ctx.EmitMakeNil(d537)
				ctx.MarkLabel(lbl44)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d536 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r27, Reg2: r28}
				ctx.BindReg(r27, &d536)
				ctx.BindReg(r28, &d536)
			}
			var d538 JITValueDesc
			if d536.Loc == LocImm {
				d538 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d536.Imm.Float())}
			} else if d536.Type == tagFloat && d536.Loc == LocReg {
				d538 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d536.Reg}
				ctx.BindReg(d536.Reg, &d538)
				ctx.BindReg(d536.Reg, &d538)
			} else if d536.Type == tagFloat && d536.Loc == LocRegPair {
				ctx.FreeReg(d536.Reg)
				d538 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d536.Reg2}
				ctx.BindReg(d536.Reg2, &d538)
				ctx.BindReg(d536.Reg2, &d538)
			} else {
				d538 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d536}, 1)
				d538.Type = tagFloat
				ctx.BindReg(d538.Reg, &d538)
			}
			ctx.FreeDesc(&d536)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d538)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d538)
			var d539 JITValueDesc
			if d5.Loc == LocImm && d538.Loc == LocImm {
				d539 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d5.Imm.Float() - d538.Imm.Float())}
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d538.Reg)
				_, xBits := d5.Imm.RawWords()
				ctx.EmitMovRegImm64(scratch, xBits)
				ctx.EmitSubFloat64(scratch, d538.Reg)
				d539 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d539)
			} else if d538.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(scratch, d5.Reg)
				_, yBits := d538.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, yBits)
				ctx.EmitSubFloat64(scratch, RegR11)
				d539 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d539)
			} else {
				r29 := ctx.AllocRegExcept(d5.Reg, d538.Reg)
				ctx.EmitMovRegReg(r29, d5.Reg)
				ctx.EmitSubFloat64(r29, d538.Reg)
				d539 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r29}
				ctx.BindReg(r29, &d539)
			}
			if d539.Loc == LocReg && d5.Loc == LocReg && d539.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d538)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d6)
			var d540 JITValueDesc
			if d6.Loc == LocImm {
				d540 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(scratch, d6.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d540 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d540)
			}
			if d540.Loc == LocReg && d6.Loc == LocReg && d540.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = LocNone
			}
			ctx.EnsureDesc(&d539)
			if d539.Loc == LocReg {
				ctx.ProtectReg(d539.Reg)
			} else if d539.Loc == LocRegPair {
				ctx.ProtectReg(d539.Reg)
				ctx.ProtectReg(d539.Reg2)
			}
			ctx.EnsureDesc(&d540)
			if d540.Loc == LocReg {
				ctx.ProtectReg(d540.Reg)
			} else if d540.Loc == LocRegPair {
				ctx.ProtectReg(d540.Reg)
				ctx.ProtectReg(d540.Reg2)
			}
			d541 = d539
			if d541.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d541)
			ctx.EmitStoreToStack(d541, int32(bbs[16].PhiBase)+int32(0))
			d542 = d540
			if d542.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d542)
			ctx.EmitStoreToStack(d542, int32(bbs[16].PhiBase)+int32(16))
			if d539.Loc == LocReg {
				ctx.UnprotectReg(d539.Reg)
			} else if d539.Loc == LocRegPair {
				ctx.UnprotectReg(d539.Reg)
				ctx.UnprotectReg(d539.Reg2)
			}
			if d540.Loc == LocReg {
				ctx.UnprotectReg(d540.Reg)
			} else if d540.Loc == LocRegPair {
				ctx.UnprotectReg(d540.Reg)
				ctx.UnprotectReg(d540.Reg2)
			}
			ps543 := PhiState{General: ps.General}
			ps543.OverlayValues = make([]JITValueDesc, 543)
			ps543.OverlayValues[0] = d0
			ps543.OverlayValues[1] = d1
			ps543.OverlayValues[2] = d2
			ps543.OverlayValues[3] = d3
			ps543.OverlayValues[4] = d4
			ps543.OverlayValues[5] = d5
			ps543.OverlayValues[6] = d6
			ps543.OverlayValues[7] = d7
			ps543.OverlayValues[9] = d9
			ps543.OverlayValues[10] = d10
			ps543.OverlayValues[11] = d11
			ps543.OverlayValues[12] = d12
			ps543.OverlayValues[13] = d13
			ps543.OverlayValues[16] = d16
			ps543.OverlayValues[34] = d34
			ps543.OverlayValues[35] = d35
			ps543.OverlayValues[36] = d36
			ps543.OverlayValues[37] = d37
			ps543.OverlayValues[38] = d38
			ps543.OverlayValues[40] = d40
			ps543.OverlayValues[42] = d42
			ps543.OverlayValues[43] = d43
			ps543.OverlayValues[46] = d46
			ps543.OverlayValues[71] = d71
			ps543.OverlayValues[72] = d72
			ps543.OverlayValues[73] = d73
			ps543.OverlayValues[74] = d74
			ps543.OverlayValues[107] = d107
			ps543.OverlayValues[108] = d108
			ps543.OverlayValues[109] = d109
			ps543.OverlayValues[111] = d111
			ps543.OverlayValues[112] = d112
			ps543.OverlayValues[113] = d113
			ps543.OverlayValues[114] = d114
			ps543.OverlayValues[115] = d115
			ps543.OverlayValues[117] = d117
			ps543.OverlayValues[118] = d118
			ps543.OverlayValues[119] = d119
			ps543.OverlayValues[120] = d120
			ps543.OverlayValues[121] = d121
			ps543.OverlayValues[122] = d122
			ps543.OverlayValues[123] = d123
			ps543.OverlayValues[124] = d124
			ps543.OverlayValues[125] = d125
			ps543.OverlayValues[127] = d127
			ps543.OverlayValues[128] = d128
			ps543.OverlayValues[129] = d129
			ps543.OverlayValues[130] = d130
			ps543.OverlayValues[131] = d131
			ps543.OverlayValues[186] = d186
			ps543.OverlayValues[187] = d187
			ps543.OverlayValues[188] = d188
			ps543.OverlayValues[189] = d189
			ps543.OverlayValues[190] = d190
			ps543.OverlayValues[193] = d193
			ps543.OverlayValues[194] = d194
			ps543.OverlayValues[254] = d254
			ps543.OverlayValues[255] = d255
			ps543.OverlayValues[256] = d256
			ps543.OverlayValues[257] = d257
			ps543.OverlayValues[258] = d258
			ps543.OverlayValues[325] = d325
			ps543.OverlayValues[326] = d326
			ps543.OverlayValues[327] = d327
			ps543.OverlayValues[329] = d329
			ps543.OverlayValues[330] = d330
			ps543.OverlayValues[331] = d331
			ps543.OverlayValues[332] = d332
			ps543.OverlayValues[333] = d333
			ps543.OverlayValues[334] = d334
			ps543.OverlayValues[335] = d335
			ps543.OverlayValues[336] = d336
			ps543.OverlayValues[337] = d337
			ps543.OverlayValues[339] = d339
			ps543.OverlayValues[340] = d340
			ps543.OverlayValues[341] = d341
			ps543.OverlayValues[342] = d342
			ps543.OverlayValues[343] = d343
			ps543.OverlayValues[344] = d344
			ps543.OverlayValues[345] = d345
			ps543.OverlayValues[348] = d348
			ps543.OverlayValues[349] = d349
			ps543.OverlayValues[435] = d435
			ps543.OverlayValues[436] = d436
			ps543.OverlayValues[437] = d437
			ps543.OverlayValues[438] = d438
			ps543.OverlayValues[439] = d439
			ps543.OverlayValues[442] = d442
			ps543.OverlayValues[443] = d443
			ps543.OverlayValues[536] = d536
			ps543.OverlayValues[537] = d537
			ps543.OverlayValues[538] = d538
			ps543.OverlayValues[539] = d539
			ps543.OverlayValues[540] = d540
			ps543.OverlayValues[541] = d541
			ps543.OverlayValues[542] = d542
			ps543.PhiValues = make([]JITValueDesc, 2)
			d544 = d539
			ps543.PhiValues[0] = d544
			d545 = d540
			ps543.PhiValues[1] = d545
			if ps543.General && bbs[16].Rendered {
				ctx.EmitJmp(lbl17)
				return result
			}
			return bbs[16].RenderPS(ps543)
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
					ctx.EmitJmp(lbl19)
					return result
				}
				bbs[18].Rendered = true
				bbs[18].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_18 = bbs[18].Address
				ctx.MarkLabel(lbl19)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d6 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(96)}
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != LocNone {
				d128 = ps.OverlayValues[128]
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
			if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != LocNone {
				d186 = ps.OverlayValues[186]
			}
			if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != LocNone {
				d187 = ps.OverlayValues[187]
			}
			if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != LocNone {
				d188 = ps.OverlayValues[188]
			}
			if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != LocNone {
				d189 = ps.OverlayValues[189]
			}
			if len(ps.OverlayValues) > 190 && ps.OverlayValues[190].Loc != LocNone {
				d190 = ps.OverlayValues[190]
			}
			if len(ps.OverlayValues) > 193 && ps.OverlayValues[193].Loc != LocNone {
				d193 = ps.OverlayValues[193]
			}
			if len(ps.OverlayValues) > 194 && ps.OverlayValues[194].Loc != LocNone {
				d194 = ps.OverlayValues[194]
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
			if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != LocNone {
				d257 = ps.OverlayValues[257]
			}
			if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != LocNone {
				d258 = ps.OverlayValues[258]
			}
			if len(ps.OverlayValues) > 325 && ps.OverlayValues[325].Loc != LocNone {
				d325 = ps.OverlayValues[325]
			}
			if len(ps.OverlayValues) > 326 && ps.OverlayValues[326].Loc != LocNone {
				d326 = ps.OverlayValues[326]
			}
			if len(ps.OverlayValues) > 327 && ps.OverlayValues[327].Loc != LocNone {
				d327 = ps.OverlayValues[327]
			}
			if len(ps.OverlayValues) > 329 && ps.OverlayValues[329].Loc != LocNone {
				d329 = ps.OverlayValues[329]
			}
			if len(ps.OverlayValues) > 330 && ps.OverlayValues[330].Loc != LocNone {
				d330 = ps.OverlayValues[330]
			}
			if len(ps.OverlayValues) > 331 && ps.OverlayValues[331].Loc != LocNone {
				d331 = ps.OverlayValues[331]
			}
			if len(ps.OverlayValues) > 332 && ps.OverlayValues[332].Loc != LocNone {
				d332 = ps.OverlayValues[332]
			}
			if len(ps.OverlayValues) > 333 && ps.OverlayValues[333].Loc != LocNone {
				d333 = ps.OverlayValues[333]
			}
			if len(ps.OverlayValues) > 334 && ps.OverlayValues[334].Loc != LocNone {
				d334 = ps.OverlayValues[334]
			}
			if len(ps.OverlayValues) > 335 && ps.OverlayValues[335].Loc != LocNone {
				d335 = ps.OverlayValues[335]
			}
			if len(ps.OverlayValues) > 336 && ps.OverlayValues[336].Loc != LocNone {
				d336 = ps.OverlayValues[336]
			}
			if len(ps.OverlayValues) > 337 && ps.OverlayValues[337].Loc != LocNone {
				d337 = ps.OverlayValues[337]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 341 && ps.OverlayValues[341].Loc != LocNone {
				d341 = ps.OverlayValues[341]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
				d343 = ps.OverlayValues[343]
			}
			if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
				d344 = ps.OverlayValues[344]
			}
			if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
				d345 = ps.OverlayValues[345]
			}
			if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
				d348 = ps.OverlayValues[348]
			}
			if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
				d349 = ps.OverlayValues[349]
			}
			if len(ps.OverlayValues) > 435 && ps.OverlayValues[435].Loc != LocNone {
				d435 = ps.OverlayValues[435]
			}
			if len(ps.OverlayValues) > 436 && ps.OverlayValues[436].Loc != LocNone {
				d436 = ps.OverlayValues[436]
			}
			if len(ps.OverlayValues) > 437 && ps.OverlayValues[437].Loc != LocNone {
				d437 = ps.OverlayValues[437]
			}
			if len(ps.OverlayValues) > 438 && ps.OverlayValues[438].Loc != LocNone {
				d438 = ps.OverlayValues[438]
			}
			if len(ps.OverlayValues) > 439 && ps.OverlayValues[439].Loc != LocNone {
				d439 = ps.OverlayValues[439]
			}
			if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
				d442 = ps.OverlayValues[442]
			}
			if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != LocNone {
				d443 = ps.OverlayValues[443]
			}
			if len(ps.OverlayValues) > 536 && ps.OverlayValues[536].Loc != LocNone {
				d536 = ps.OverlayValues[536]
			}
			if len(ps.OverlayValues) > 537 && ps.OverlayValues[537].Loc != LocNone {
				d537 = ps.OverlayValues[537]
			}
			if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != LocNone {
				d538 = ps.OverlayValues[538]
			}
			if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != LocNone {
				d539 = ps.OverlayValues[539]
			}
			if len(ps.OverlayValues) > 540 && ps.OverlayValues[540].Loc != LocNone {
				d540 = ps.OverlayValues[540]
			}
			if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != LocNone {
				d541 = ps.OverlayValues[541]
			}
			if len(ps.OverlayValues) > 542 && ps.OverlayValues[542].Loc != LocNone {
				d542 = ps.OverlayValues[542]
			}
			if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != LocNone {
				d544 = ps.OverlayValues[544]
			}
			if len(ps.OverlayValues) > 545 && ps.OverlayValues[545].Loc != LocNone {
				d545 = ps.OverlayValues[545]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			ctx.EmitMakeFloat(result, d5)
			if d5.Loc == LocReg { ctx.FreeReg(d5.Reg) }
			result.Type = tagFloat
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned546 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned546 = append(argPinned546, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned546 = append(argPinned546, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned546 = append(argPinned546, ai.Reg2)
					}
				}
			}
			ps547 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps547)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(112))
			ctx.EmitAddRSP32(int32(112))
			for _, r := range argPinned546 {
				ctx.UnprotectReg(r)
			}
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
			var d15 JITValueDesc
			_ = d15
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
			var d38 JITValueDesc
			_ = d38
			var d40 JITValueDesc
			_ = d40
			var d41 JITValueDesc
			_ = d41
			var d44 JITValueDesc
			_ = d44
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
			var d75 JITValueDesc
			_ = d75
			var d110 JITValueDesc
			_ = d110
			var d111 JITValueDesc
			_ = d111
			var d112 JITValueDesc
			_ = d112
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
			var d157 JITValueDesc
			_ = d157
			var d158 JITValueDesc
			_ = d158
			var d201 JITValueDesc
			_ = d201
			var d202 JITValueDesc
			_ = d202
			var d203 JITValueDesc
			_ = d203
			var d204 JITValueDesc
			_ = d204
			var d206 JITValueDesc
			_ = d206
			var d207 JITValueDesc
			_ = d207
			var d208 JITValueDesc
			_ = d208
			var d209 JITValueDesc
			_ = d209
			var d210 JITValueDesc
			_ = d210
			var d212 JITValueDesc
			_ = d212
			var d213 JITValueDesc
			_ = d213
			var d214 JITValueDesc
			_ = d214
			var d215 JITValueDesc
			_ = d215
			var d273 JITValueDesc
			_ = d273
			var d274 JITValueDesc
			_ = d274
			var d275 JITValueDesc
			_ = d275
			var d276 JITValueDesc
			_ = d276
			var d338 JITValueDesc
			_ = d338
			var d339 JITValueDesc
			_ = d339
			var d340 JITValueDesc
			_ = d340
			var d342 JITValueDesc
			_ = d342
			var d343 JITValueDesc
			_ = d343
			var d344 JITValueDesc
			_ = d344
			var d345 JITValueDesc
			_ = d345
			var d347 JITValueDesc
			_ = d347
			var d348 JITValueDesc
			_ = d348
			var d349 JITValueDesc
			_ = d349
			var d350 JITValueDesc
			_ = d350
			var d351 JITValueDesc
			_ = d351
			var d352 JITValueDesc
			_ = d352
			var d353 JITValueDesc
			_ = d353
			var d354 JITValueDesc
			_ = d354
			var d355 JITValueDesc
			_ = d355
			var d357 JITValueDesc
			_ = d357
			var d358 JITValueDesc
			_ = d358
			var d359 JITValueDesc
			_ = d359
			var d360 JITValueDesc
			_ = d360
			var d361 JITValueDesc
			_ = d361
			var d362 JITValueDesc
			_ = d362
			var d363 JITValueDesc
			_ = d363
			var d366 JITValueDesc
			_ = d366
			var d367 JITValueDesc
			_ = d367
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(96)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(96)
			}
			d0 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			var bbs [18]BBDescriptor
			bbs[1].PhiBase = int32(0)
			bbs[1].PhiCount = uint16(1)
			bbs[7].PhiBase = int32(16)
			bbs[7].PhiCount = uint16(2)
			bbs[8].PhiBase = int32(48)
			bbs[8].PhiCount = uint16(1)
			bbs[17].PhiBase = int32(64)
			bbs[17].PhiCount = uint16(2)
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
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			lbl11 := ctx.ReserveLabel()
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			lbl12 := ctx.ReserveLabel()
			bbpos_0_12 := int32(-1)
			_ = bbpos_0_12
			lbl13 := ctx.ReserveLabel()
			bbpos_0_13 := int32(-1)
			_ = bbpos_0_13
			lbl14 := ctx.ReserveLabel()
			bbpos_0_14 := int32(-1)
			_ = bbpos_0_14
			lbl15 := ctx.ReserveLabel()
			bbpos_0_15 := int32(-1)
			_ = bbpos_0_15
			lbl16 := ctx.ReserveLabel()
			bbpos_0_16 := int32(-1)
			_ = bbpos_0_16
			lbl17 := ctx.ReserveLabel()
			bbpos_0_17 := int32(-1)
			_ = bbpos_0_17
			lbl18 := ctx.ReserveLabel()
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
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
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
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(0))
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
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps7)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d9 := ps.PhiValues[0]
					ctx.EnsureDesc(&d9)
					ctx.EmitStoreToStack(d9, int32(bbs[1].PhiBase)+int32(0))
				}
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
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
				ctx.EmitMovRegReg(scratch, d0.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
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
					ctx.EmitCmpRegImm32(d10.Reg, int32(d6.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d6.Imm.Int()))
					ctx.EmitCmpInt64(d10.Reg, RegR11)
				}
				ctx.EmitSetcc(r1, CcL)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d11)
			} else if d10.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d10.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d6.Reg)
				ctx.EmitSetcc(r2, CcL)
				d11 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d11)
			} else {
				r3 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitCmpInt64(d10.Reg, d6.Reg)
				ctx.EmitSetcc(r3, CcL)
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
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d15 := ps.PhiValues[0]
					ctx.EnsureDesc(&d15)
					ctx.EmitStoreToStack(d15, int32(bbs[1].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl19 := ctx.ReserveLabel()
			lbl20 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d12.Reg, 0)
			ctx.EmitJcc(CcNE, lbl19)
			ctx.EmitJmp(lbl20)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl4)
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 16)
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
			ps16.OverlayValues[15] = d15
			ps17 := PhiState{General: true}
			ps17.OverlayValues = make([]JITValueDesc, 16)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
			ps17.OverlayValues[4] = d4
			ps17.OverlayValues[5] = d5
			ps17.OverlayValues[6] = d6
			ps17.OverlayValues[8] = d8
			ps17.OverlayValues[9] = d9
			ps17.OverlayValues[10] = d10
			ps17.OverlayValues[11] = d11
			ps17.OverlayValues[12] = d12
			ps17.OverlayValues[15] = d15
			snap18 := d0
			snap19 := d1
			snap20 := d2
			snap21 := d3
			snap22 := d4
			snap23 := d5
			snap24 := d6
			snap25 := d8
			snap26 := d9
			snap27 := d10
			snap28 := d11
			snap29 := d12
			snap30 := d15
			alloc31 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps17)
			}
			ctx.RestoreAllocState(alloc31)
			d0 = snap18
			d1 = snap19
			d2 = snap20
			d3 = snap21
			d4 = snap22
			d5 = snap23
			d6 = snap24
			d8 = snap25
			d9 = snap26
			d10 = snap27
			d11 = snap28
			d12 = snap29
			d15 = snap30
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps16)
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
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d10)
			var d32 JITValueDesc
			if d10.Loc == LocImm {
				idx := int(d10.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d32 = args[idx]
				d32.ID = 0
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
				lbl21 := ctx.ReserveLabel()
				lbl22 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d10.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl22)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d10.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r4, ai.Reg)
						ctx.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r4, tmp.Reg)
						ctx.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl22)
				d33 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d33)
				ctx.BindReg(r5, &d33)
				ctx.BindReg(r4, &d33)
				ctx.BindReg(r5, &d33)
				ctx.EmitMakeNil(d33)
				ctx.MarkLabel(lbl21)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d32 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d32)
				ctx.BindReg(r5, &d32)
			}
			d35 = d32
			d35.ID = 0
			d34 = ctx.EmitTagEqualsBorrowed(&d35, tagNil, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d32)
			d36 = d34
			ctx.EnsureDesc(&d36)
			if d36.Loc != LocImm && d36.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d36.Loc == LocImm {
				if d36.Imm.Bool() {
			ps37 := PhiState{General: ps.General}
			ps37.OverlayValues = make([]JITValueDesc, 37)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[3] = d3
			ps37.OverlayValues[4] = d4
			ps37.OverlayValues[5] = d5
			ps37.OverlayValues[6] = d6
			ps37.OverlayValues[8] = d8
			ps37.OverlayValues[9] = d9
			ps37.OverlayValues[10] = d10
			ps37.OverlayValues[11] = d11
			ps37.OverlayValues[12] = d12
			ps37.OverlayValues[15] = d15
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			ps37.OverlayValues[34] = d34
			ps37.OverlayValues[35] = d35
			ps37.OverlayValues[36] = d36
					return bbs[4].RenderPS(ps37)
				}
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d38 = d10
			if d38.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d38)
			ctx.EmitStoreToStack(d38, int32(bbs[1].PhiBase)+int32(0))
			if d10.Loc == LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
			ps39 := PhiState{General: ps.General}
			ps39.OverlayValues = make([]JITValueDesc, 39)
			ps39.OverlayValues[0] = d0
			ps39.OverlayValues[1] = d1
			ps39.OverlayValues[2] = d2
			ps39.OverlayValues[3] = d3
			ps39.OverlayValues[4] = d4
			ps39.OverlayValues[5] = d5
			ps39.OverlayValues[6] = d6
			ps39.OverlayValues[8] = d8
			ps39.OverlayValues[9] = d9
			ps39.OverlayValues[10] = d10
			ps39.OverlayValues[11] = d11
			ps39.OverlayValues[12] = d12
			ps39.OverlayValues[15] = d15
			ps39.OverlayValues[32] = d32
			ps39.OverlayValues[33] = d33
			ps39.OverlayValues[34] = d34
			ps39.OverlayValues[35] = d35
			ps39.OverlayValues[36] = d36
			ps39.OverlayValues[38] = d38
			ps39.PhiValues = make([]JITValueDesc, 1)
			d40 = d10
			ps39.PhiValues[0] = d40
				return bbs[1].RenderPS(ps39)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl23 := ctx.ReserveLabel()
			lbl24 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d36.Reg, 0)
			ctx.EmitJcc(CcNE, lbl23)
			ctx.EmitJmp(lbl24)
			ctx.MarkLabel(lbl23)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl24)
			ctx.EnsureDesc(&d10)
			if d10.Loc == LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d41 = d10
			if d41.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d41)
			ctx.EmitStoreToStack(d41, int32(bbs[1].PhiBase)+int32(0))
			if d10.Loc == LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
			ctx.EmitJmp(lbl2)
			ps42 := PhiState{General: true}
			ps42.OverlayValues = make([]JITValueDesc, 42)
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
			ps42.OverlayValues[15] = d15
			ps42.OverlayValues[32] = d32
			ps42.OverlayValues[33] = d33
			ps42.OverlayValues[34] = d34
			ps42.OverlayValues[35] = d35
			ps42.OverlayValues[36] = d36
			ps42.OverlayValues[38] = d38
			ps42.OverlayValues[40] = d40
			ps42.OverlayValues[41] = d41
			ps43 := PhiState{General: true}
			ps43.OverlayValues = make([]JITValueDesc, 42)
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
			ps43.OverlayValues[15] = d15
			ps43.OverlayValues[32] = d32
			ps43.OverlayValues[33] = d33
			ps43.OverlayValues[34] = d34
			ps43.OverlayValues[35] = d35
			ps43.OverlayValues[36] = d36
			ps43.OverlayValues[38] = d38
			ps43.OverlayValues[40] = d40
			ps43.OverlayValues[41] = d41
			ps43.PhiValues = make([]JITValueDesc, 1)
			d44 = d10
			ps43.PhiValues[0] = d44
			snap45 := d0
			snap46 := d1
			snap47 := d2
			snap48 := d3
			snap49 := d4
			snap50 := d5
			snap51 := d6
			snap52 := d8
			snap53 := d9
			snap54 := d10
			snap55 := d11
			snap56 := d12
			snap57 := d15
			snap58 := d32
			snap59 := d33
			snap60 := d34
			snap61 := d35
			snap62 := d36
			snap63 := d38
			snap64 := d40
			snap65 := d41
			snap66 := d44
			alloc67 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps43)
			}
			ctx.RestoreAllocState(alloc67)
			d0 = snap45
			d1 = snap46
			d2 = snap47
			d3 = snap48
			d4 = snap49
			d5 = snap50
			d6 = snap51
			d8 = snap52
			d9 = snap53
			d10 = snap54
			d11 = snap55
			d12 = snap56
			d15 = snap57
			d32 = snap58
			d33 = snap59
			d34 = snap60
			d35 = snap61
			d36 = snap62
			d38 = snap63
			d40 = snap64
			d41 = snap65
			d44 = snap66
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps42)
			}
			return result
			ctx.FreeDesc(&d34)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}, int32(bbs[7].PhiBase)+int32(0))
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(16))
			ps68 := PhiState{General: ps.General}
			ps68.OverlayValues = make([]JITValueDesc, 45)
			ps68.OverlayValues[0] = d0
			ps68.OverlayValues[1] = d1
			ps68.OverlayValues[2] = d2
			ps68.OverlayValues[3] = d3
			ps68.OverlayValues[4] = d4
			ps68.OverlayValues[5] = d5
			ps68.OverlayValues[6] = d6
			ps68.OverlayValues[8] = d8
			ps68.OverlayValues[9] = d9
			ps68.OverlayValues[10] = d10
			ps68.OverlayValues[11] = d11
			ps68.OverlayValues[12] = d12
			ps68.OverlayValues[15] = d15
			ps68.OverlayValues[32] = d32
			ps68.OverlayValues[33] = d33
			ps68.OverlayValues[34] = d34
			ps68.OverlayValues[35] = d35
			ps68.OverlayValues[36] = d36
			ps68.OverlayValues[38] = d38
			ps68.OverlayValues[40] = d40
			ps68.OverlayValues[41] = d41
			ps68.OverlayValues[44] = d44
			ps68.PhiValues = make([]JITValueDesc, 2)
			d69 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
			ps68.PhiValues[0] = d69
			d70 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
			ps68.PhiValues[1] = d70
			if ps68.General && bbs[7].Rendered {
				ctx.EmitJmp(lbl8)
				return result
			}
			return bbs[7].RenderPS(ps68)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
				d70 = ps.OverlayValues[70]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EmitMakeNil(result)
			result.Type = tagNil
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != LocNone {
				d70 = ps.OverlayValues[70]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			var d71 JITValueDesc
			if d2.Loc == LocImm {
				idx := int(d2.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d71 = args[idx]
				d71.ID = 0
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
				lbl25 := ctx.ReserveLabel()
				lbl26 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d2.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl26)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d2.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r6, ai.Reg)
						ctx.EmitMovRegReg(r7, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r6, tmp.Reg)
						ctx.EmitMovRegReg(r7, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
						ctx.BindReg(r6, &pair)
						ctx.BindReg(r7, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r6, uint64(ptrWord))
							ctx.EmitMovRegImm64(r7, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl26)
				d72 := JITValueDesc{Loc: LocRegPair, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d72)
				ctx.BindReg(r7, &d72)
				ctx.BindReg(r6, &d72)
				ctx.BindReg(r7, &d72)
				ctx.EmitMakeNil(d72)
				ctx.MarkLabel(lbl25)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d71 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r6, Reg2: r7}
				ctx.BindReg(r6, &d71)
				ctx.BindReg(r7, &d71)
			}
			d74 = d71
			d74.ID = 0
			d73 = ctx.EmitTagEqualsBorrowed(&d74, tagInt, JITValueDesc{Loc: LocAny})
			d75 = d73
			ctx.EnsureDesc(&d75)
			if d75.Loc != LocImm && d75.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d75.Loc == LocImm {
				if d75.Imm.Bool() {
			ps76 := PhiState{General: ps.General}
			ps76.OverlayValues = make([]JITValueDesc, 76)
			ps76.OverlayValues[0] = d0
			ps76.OverlayValues[1] = d1
			ps76.OverlayValues[2] = d2
			ps76.OverlayValues[3] = d3
			ps76.OverlayValues[4] = d4
			ps76.OverlayValues[5] = d5
			ps76.OverlayValues[6] = d6
			ps76.OverlayValues[8] = d8
			ps76.OverlayValues[9] = d9
			ps76.OverlayValues[10] = d10
			ps76.OverlayValues[11] = d11
			ps76.OverlayValues[12] = d12
			ps76.OverlayValues[15] = d15
			ps76.OverlayValues[32] = d32
			ps76.OverlayValues[33] = d33
			ps76.OverlayValues[34] = d34
			ps76.OverlayValues[35] = d35
			ps76.OverlayValues[36] = d36
			ps76.OverlayValues[38] = d38
			ps76.OverlayValues[40] = d40
			ps76.OverlayValues[41] = d41
			ps76.OverlayValues[44] = d44
			ps76.OverlayValues[69] = d69
			ps76.OverlayValues[70] = d70
			ps76.OverlayValues[71] = d71
			ps76.OverlayValues[72] = d72
			ps76.OverlayValues[73] = d73
			ps76.OverlayValues[74] = d74
			ps76.OverlayValues[75] = d75
					return bbs[9].RenderPS(ps76)
				}
			ps77 := PhiState{General: ps.General}
			ps77.OverlayValues = make([]JITValueDesc, 76)
			ps77.OverlayValues[0] = d0
			ps77.OverlayValues[1] = d1
			ps77.OverlayValues[2] = d2
			ps77.OverlayValues[3] = d3
			ps77.OverlayValues[4] = d4
			ps77.OverlayValues[5] = d5
			ps77.OverlayValues[6] = d6
			ps77.OverlayValues[8] = d8
			ps77.OverlayValues[9] = d9
			ps77.OverlayValues[10] = d10
			ps77.OverlayValues[11] = d11
			ps77.OverlayValues[12] = d12
			ps77.OverlayValues[15] = d15
			ps77.OverlayValues[32] = d32
			ps77.OverlayValues[33] = d33
			ps77.OverlayValues[34] = d34
			ps77.OverlayValues[35] = d35
			ps77.OverlayValues[36] = d36
			ps77.OverlayValues[38] = d38
			ps77.OverlayValues[40] = d40
			ps77.OverlayValues[41] = d41
			ps77.OverlayValues[44] = d44
			ps77.OverlayValues[69] = d69
			ps77.OverlayValues[70] = d70
			ps77.OverlayValues[71] = d71
			ps77.OverlayValues[72] = d72
			ps77.OverlayValues[73] = d73
			ps77.OverlayValues[74] = d74
			ps77.OverlayValues[75] = d75
				return bbs[10].RenderPS(ps77)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl27 := ctx.ReserveLabel()
			lbl28 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d75.Reg, 0)
			ctx.EmitJcc(CcNE, lbl27)
			ctx.EmitJmp(lbl28)
			ctx.MarkLabel(lbl27)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl28)
			ctx.EmitJmp(lbl11)
			ps78 := PhiState{General: true}
			ps78.OverlayValues = make([]JITValueDesc, 76)
			ps78.OverlayValues[0] = d0
			ps78.OverlayValues[1] = d1
			ps78.OverlayValues[2] = d2
			ps78.OverlayValues[3] = d3
			ps78.OverlayValues[4] = d4
			ps78.OverlayValues[5] = d5
			ps78.OverlayValues[6] = d6
			ps78.OverlayValues[8] = d8
			ps78.OverlayValues[9] = d9
			ps78.OverlayValues[10] = d10
			ps78.OverlayValues[11] = d11
			ps78.OverlayValues[12] = d12
			ps78.OverlayValues[15] = d15
			ps78.OverlayValues[32] = d32
			ps78.OverlayValues[33] = d33
			ps78.OverlayValues[34] = d34
			ps78.OverlayValues[35] = d35
			ps78.OverlayValues[36] = d36
			ps78.OverlayValues[38] = d38
			ps78.OverlayValues[40] = d40
			ps78.OverlayValues[41] = d41
			ps78.OverlayValues[44] = d44
			ps78.OverlayValues[69] = d69
			ps78.OverlayValues[70] = d70
			ps78.OverlayValues[71] = d71
			ps78.OverlayValues[72] = d72
			ps78.OverlayValues[73] = d73
			ps78.OverlayValues[74] = d74
			ps78.OverlayValues[75] = d75
			ps79 := PhiState{General: true}
			ps79.OverlayValues = make([]JITValueDesc, 76)
			ps79.OverlayValues[0] = d0
			ps79.OverlayValues[1] = d1
			ps79.OverlayValues[2] = d2
			ps79.OverlayValues[3] = d3
			ps79.OverlayValues[4] = d4
			ps79.OverlayValues[5] = d5
			ps79.OverlayValues[6] = d6
			ps79.OverlayValues[8] = d8
			ps79.OverlayValues[9] = d9
			ps79.OverlayValues[10] = d10
			ps79.OverlayValues[11] = d11
			ps79.OverlayValues[12] = d12
			ps79.OverlayValues[15] = d15
			ps79.OverlayValues[32] = d32
			ps79.OverlayValues[33] = d33
			ps79.OverlayValues[34] = d34
			ps79.OverlayValues[35] = d35
			ps79.OverlayValues[36] = d36
			ps79.OverlayValues[38] = d38
			ps79.OverlayValues[40] = d40
			ps79.OverlayValues[41] = d41
			ps79.OverlayValues[44] = d44
			ps79.OverlayValues[69] = d69
			ps79.OverlayValues[70] = d70
			ps79.OverlayValues[71] = d71
			ps79.OverlayValues[72] = d72
			ps79.OverlayValues[73] = d73
			ps79.OverlayValues[74] = d74
			ps79.OverlayValues[75] = d75
			snap80 := d0
			snap81 := d1
			snap82 := d2
			snap83 := d3
			snap84 := d4
			snap85 := d5
			snap86 := d6
			snap87 := d8
			snap88 := d9
			snap89 := d10
			snap90 := d11
			snap91 := d12
			snap92 := d15
			snap93 := d32
			snap94 := d33
			snap95 := d34
			snap96 := d35
			snap97 := d36
			snap98 := d38
			snap99 := d40
			snap100 := d41
			snap101 := d44
			snap102 := d69
			snap103 := d70
			snap104 := d71
			snap105 := d72
			snap106 := d73
			snap107 := d74
			snap108 := d75
			alloc109 := ctx.SnapshotAllocState()
			if !bbs[10].Rendered {
				bbs[10].RenderPS(ps79)
			}
			ctx.RestoreAllocState(alloc109)
			d0 = snap80
			d1 = snap81
			d2 = snap82
			d3 = snap83
			d4 = snap84
			d5 = snap85
			d6 = snap86
			d8 = snap87
			d9 = snap88
			d10 = snap89
			d11 = snap90
			d12 = snap91
			d15 = snap92
			d32 = snap93
			d33 = snap94
			d34 = snap95
			d35 = snap96
			d36 = snap97
			d38 = snap98
			d40 = snap99
			d41 = snap100
			d44 = snap101
			d69 = snap102
			d70 = snap103
			d71 = snap104
			d72 = snap105
			d73 = snap106
			d74 = snap107
			d75 = snap108
			if !bbs[9].Rendered {
				return bbs[9].RenderPS(ps78)
			}
			return result
			ctx.FreeDesc(&d73)
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
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			ctx.ReclaimUntrackedRegs()
			d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d110)
			var d111 JITValueDesc
			if d2.Loc == LocImm && d110.Loc == LocImm {
				d111 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() == d110.Imm.Int())}
			} else if d110.Loc == LocImm {
				r8 := ctx.AllocRegExcept(d2.Reg)
				if d110.Imm.Int() >= -2147483648 && d110.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d2.Reg, int32(d110.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d110.Imm.Int()))
					ctx.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.EmitSetcc(r8, CcE)
				d111 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
				ctx.BindReg(r8, &d111)
			} else if d2.Loc == LocImm {
				r9 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d110.Reg)
				ctx.EmitSetcc(r9, CcE)
				d111 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d111)
			} else {
				r10 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitCmpInt64(d2.Reg, d110.Reg)
				ctx.EmitSetcc(r10, CcE)
				d111 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d111)
			}
			ctx.FreeDesc(&d110)
			d112 = d111
			ctx.EnsureDesc(&d112)
			if d112.Loc != LocImm && d112.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d112.Loc == LocImm {
				if d112.Imm.Bool() {
			ps113 := PhiState{General: ps.General}
			ps113.OverlayValues = make([]JITValueDesc, 113)
			ps113.OverlayValues[0] = d0
			ps113.OverlayValues[1] = d1
			ps113.OverlayValues[2] = d2
			ps113.OverlayValues[3] = d3
			ps113.OverlayValues[4] = d4
			ps113.OverlayValues[5] = d5
			ps113.OverlayValues[6] = d6
			ps113.OverlayValues[8] = d8
			ps113.OverlayValues[9] = d9
			ps113.OverlayValues[10] = d10
			ps113.OverlayValues[11] = d11
			ps113.OverlayValues[12] = d12
			ps113.OverlayValues[15] = d15
			ps113.OverlayValues[32] = d32
			ps113.OverlayValues[33] = d33
			ps113.OverlayValues[34] = d34
			ps113.OverlayValues[35] = d35
			ps113.OverlayValues[36] = d36
			ps113.OverlayValues[38] = d38
			ps113.OverlayValues[40] = d40
			ps113.OverlayValues[41] = d41
			ps113.OverlayValues[44] = d44
			ps113.OverlayValues[69] = d69
			ps113.OverlayValues[70] = d70
			ps113.OverlayValues[71] = d71
			ps113.OverlayValues[72] = d72
			ps113.OverlayValues[73] = d73
			ps113.OverlayValues[74] = d74
			ps113.OverlayValues[75] = d75
			ps113.OverlayValues[110] = d110
			ps113.OverlayValues[111] = d111
			ps113.OverlayValues[112] = d112
					return bbs[13].RenderPS(ps113)
				}
			ps114 := PhiState{General: ps.General}
			ps114.OverlayValues = make([]JITValueDesc, 113)
			ps114.OverlayValues[0] = d0
			ps114.OverlayValues[1] = d1
			ps114.OverlayValues[2] = d2
			ps114.OverlayValues[3] = d3
			ps114.OverlayValues[4] = d4
			ps114.OverlayValues[5] = d5
			ps114.OverlayValues[6] = d6
			ps114.OverlayValues[8] = d8
			ps114.OverlayValues[9] = d9
			ps114.OverlayValues[10] = d10
			ps114.OverlayValues[11] = d11
			ps114.OverlayValues[12] = d12
			ps114.OverlayValues[15] = d15
			ps114.OverlayValues[32] = d32
			ps114.OverlayValues[33] = d33
			ps114.OverlayValues[34] = d34
			ps114.OverlayValues[35] = d35
			ps114.OverlayValues[36] = d36
			ps114.OverlayValues[38] = d38
			ps114.OverlayValues[40] = d40
			ps114.OverlayValues[41] = d41
			ps114.OverlayValues[44] = d44
			ps114.OverlayValues[69] = d69
			ps114.OverlayValues[70] = d70
			ps114.OverlayValues[71] = d71
			ps114.OverlayValues[72] = d72
			ps114.OverlayValues[73] = d73
			ps114.OverlayValues[74] = d74
			ps114.OverlayValues[75] = d75
			ps114.OverlayValues[110] = d110
			ps114.OverlayValues[111] = d111
			ps114.OverlayValues[112] = d112
				return bbs[14].RenderPS(ps114)
			}
			if !ps.General {
				ps.General = true
				return bbs[6].RenderPS(ps)
			}
			lbl29 := ctx.ReserveLabel()
			lbl30 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d112.Reg, 0)
			ctx.EmitJcc(CcNE, lbl29)
			ctx.EmitJmp(lbl30)
			ctx.MarkLabel(lbl29)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl30)
			ctx.EmitJmp(lbl15)
			ps115 := PhiState{General: true}
			ps115.OverlayValues = make([]JITValueDesc, 113)
			ps115.OverlayValues[0] = d0
			ps115.OverlayValues[1] = d1
			ps115.OverlayValues[2] = d2
			ps115.OverlayValues[3] = d3
			ps115.OverlayValues[4] = d4
			ps115.OverlayValues[5] = d5
			ps115.OverlayValues[6] = d6
			ps115.OverlayValues[8] = d8
			ps115.OverlayValues[9] = d9
			ps115.OverlayValues[10] = d10
			ps115.OverlayValues[11] = d11
			ps115.OverlayValues[12] = d12
			ps115.OverlayValues[15] = d15
			ps115.OverlayValues[32] = d32
			ps115.OverlayValues[33] = d33
			ps115.OverlayValues[34] = d34
			ps115.OverlayValues[35] = d35
			ps115.OverlayValues[36] = d36
			ps115.OverlayValues[38] = d38
			ps115.OverlayValues[40] = d40
			ps115.OverlayValues[41] = d41
			ps115.OverlayValues[44] = d44
			ps115.OverlayValues[69] = d69
			ps115.OverlayValues[70] = d70
			ps115.OverlayValues[71] = d71
			ps115.OverlayValues[72] = d72
			ps115.OverlayValues[73] = d73
			ps115.OverlayValues[74] = d74
			ps115.OverlayValues[75] = d75
			ps115.OverlayValues[110] = d110
			ps115.OverlayValues[111] = d111
			ps115.OverlayValues[112] = d112
			ps116 := PhiState{General: true}
			ps116.OverlayValues = make([]JITValueDesc, 113)
			ps116.OverlayValues[0] = d0
			ps116.OverlayValues[1] = d1
			ps116.OverlayValues[2] = d2
			ps116.OverlayValues[3] = d3
			ps116.OverlayValues[4] = d4
			ps116.OverlayValues[5] = d5
			ps116.OverlayValues[6] = d6
			ps116.OverlayValues[8] = d8
			ps116.OverlayValues[9] = d9
			ps116.OverlayValues[10] = d10
			ps116.OverlayValues[11] = d11
			ps116.OverlayValues[12] = d12
			ps116.OverlayValues[15] = d15
			ps116.OverlayValues[32] = d32
			ps116.OverlayValues[33] = d33
			ps116.OverlayValues[34] = d34
			ps116.OverlayValues[35] = d35
			ps116.OverlayValues[36] = d36
			ps116.OverlayValues[38] = d38
			ps116.OverlayValues[40] = d40
			ps116.OverlayValues[41] = d41
			ps116.OverlayValues[44] = d44
			ps116.OverlayValues[69] = d69
			ps116.OverlayValues[70] = d70
			ps116.OverlayValues[71] = d71
			ps116.OverlayValues[72] = d72
			ps116.OverlayValues[73] = d73
			ps116.OverlayValues[74] = d74
			ps116.OverlayValues[75] = d75
			ps116.OverlayValues[110] = d110
			ps116.OverlayValues[111] = d111
			ps116.OverlayValues[112] = d112
			snap117 := d0
			snap118 := d1
			snap119 := d2
			snap120 := d3
			snap121 := d4
			snap122 := d5
			snap123 := d6
			snap124 := d8
			snap125 := d9
			snap126 := d10
			snap127 := d11
			snap128 := d12
			snap129 := d15
			snap130 := d32
			snap131 := d33
			snap132 := d34
			snap133 := d35
			snap134 := d36
			snap135 := d38
			snap136 := d40
			snap137 := d41
			snap138 := d44
			snap139 := d69
			snap140 := d70
			snap141 := d71
			snap142 := d72
			snap143 := d73
			snap144 := d74
			snap145 := d75
			snap146 := d110
			snap147 := d111
			snap148 := d112
			alloc149 := ctx.SnapshotAllocState()
			if !bbs[14].Rendered {
				bbs[14].RenderPS(ps116)
			}
			ctx.RestoreAllocState(alloc149)
			d0 = snap117
			d1 = snap118
			d2 = snap119
			d3 = snap120
			d4 = snap121
			d5 = snap122
			d6 = snap123
			d8 = snap124
			d9 = snap125
			d10 = snap126
			d11 = snap127
			d12 = snap128
			d15 = snap129
			d32 = snap130
			d33 = snap131
			d34 = snap132
			d35 = snap133
			d36 = snap134
			d38 = snap135
			d40 = snap136
			d41 = snap137
			d44 = snap138
			d69 = snap139
			d70 = snap140
			d71 = snap141
			d72 = snap142
			d73 = snap143
			d74 = snap144
			d75 = snap145
			d110 = snap146
			d111 = snap147
			d112 = snap148
			if !bbs[13].Rendered {
				return bbs[13].RenderPS(ps115)
			}
			return result
			ctx.FreeDesc(&d111)
			return result
			}
			bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d150 := ps.PhiValues[0]
					ctx.EnsureDesc(&d150)
					ctx.EmitStoreToStack(d150, int32(bbs[7].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d151 := ps.PhiValues[1]
					ctx.EnsureDesc(&d151)
					ctx.EmitStoreToStack(d151, int32(bbs[7].PhiBase)+int32(16))
				}
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
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d1 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d2 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d152 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d152)
			var d153 JITValueDesc
			if d2.Loc == LocImm && d152.Loc == LocImm {
				d153 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d152.Imm.Int())}
			} else if d152.Loc == LocImm {
				r11 := ctx.AllocRegExcept(d2.Reg)
				if d152.Imm.Int() >= -2147483648 && d152.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d2.Reg, int32(d152.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d152.Imm.Int()))
					ctx.EmitCmpInt64(d2.Reg, RegR11)
				}
				ctx.EmitSetcc(r11, CcL)
				d153 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d153)
			} else if d2.Loc == LocImm {
				r12 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d152.Reg)
				ctx.EmitSetcc(r12, CcL)
				d153 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
				ctx.BindReg(r12, &d153)
			} else {
				r13 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitCmpInt64(d2.Reg, d152.Reg)
				ctx.EmitSetcc(r13, CcL)
				d153 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r13}
				ctx.BindReg(r13, &d153)
			}
			ctx.FreeDesc(&d152)
			d154 = d153
			ctx.EnsureDesc(&d154)
			if d154.Loc != LocImm && d154.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d154.Loc == LocImm {
				if d154.Imm.Bool() {
			ps155 := PhiState{General: ps.General}
			ps155.OverlayValues = make([]JITValueDesc, 155)
			ps155.OverlayValues[0] = d0
			ps155.OverlayValues[1] = d1
			ps155.OverlayValues[2] = d2
			ps155.OverlayValues[3] = d3
			ps155.OverlayValues[4] = d4
			ps155.OverlayValues[5] = d5
			ps155.OverlayValues[6] = d6
			ps155.OverlayValues[8] = d8
			ps155.OverlayValues[9] = d9
			ps155.OverlayValues[10] = d10
			ps155.OverlayValues[11] = d11
			ps155.OverlayValues[12] = d12
			ps155.OverlayValues[15] = d15
			ps155.OverlayValues[32] = d32
			ps155.OverlayValues[33] = d33
			ps155.OverlayValues[34] = d34
			ps155.OverlayValues[35] = d35
			ps155.OverlayValues[36] = d36
			ps155.OverlayValues[38] = d38
			ps155.OverlayValues[40] = d40
			ps155.OverlayValues[41] = d41
			ps155.OverlayValues[44] = d44
			ps155.OverlayValues[69] = d69
			ps155.OverlayValues[70] = d70
			ps155.OverlayValues[71] = d71
			ps155.OverlayValues[72] = d72
			ps155.OverlayValues[73] = d73
			ps155.OverlayValues[74] = d74
			ps155.OverlayValues[75] = d75
			ps155.OverlayValues[110] = d110
			ps155.OverlayValues[111] = d111
			ps155.OverlayValues[112] = d112
			ps155.OverlayValues[150] = d150
			ps155.OverlayValues[151] = d151
			ps155.OverlayValues[152] = d152
			ps155.OverlayValues[153] = d153
			ps155.OverlayValues[154] = d154
					return bbs[5].RenderPS(ps155)
				}
			ps156 := PhiState{General: ps.General}
			ps156.OverlayValues = make([]JITValueDesc, 155)
			ps156.OverlayValues[0] = d0
			ps156.OverlayValues[1] = d1
			ps156.OverlayValues[2] = d2
			ps156.OverlayValues[3] = d3
			ps156.OverlayValues[4] = d4
			ps156.OverlayValues[5] = d5
			ps156.OverlayValues[6] = d6
			ps156.OverlayValues[8] = d8
			ps156.OverlayValues[9] = d9
			ps156.OverlayValues[10] = d10
			ps156.OverlayValues[11] = d11
			ps156.OverlayValues[12] = d12
			ps156.OverlayValues[15] = d15
			ps156.OverlayValues[32] = d32
			ps156.OverlayValues[33] = d33
			ps156.OverlayValues[34] = d34
			ps156.OverlayValues[35] = d35
			ps156.OverlayValues[36] = d36
			ps156.OverlayValues[38] = d38
			ps156.OverlayValues[40] = d40
			ps156.OverlayValues[41] = d41
			ps156.OverlayValues[44] = d44
			ps156.OverlayValues[69] = d69
			ps156.OverlayValues[70] = d70
			ps156.OverlayValues[71] = d71
			ps156.OverlayValues[72] = d72
			ps156.OverlayValues[73] = d73
			ps156.OverlayValues[74] = d74
			ps156.OverlayValues[75] = d75
			ps156.OverlayValues[110] = d110
			ps156.OverlayValues[111] = d111
			ps156.OverlayValues[112] = d112
			ps156.OverlayValues[150] = d150
			ps156.OverlayValues[151] = d151
			ps156.OverlayValues[152] = d152
			ps156.OverlayValues[153] = d153
			ps156.OverlayValues[154] = d154
				return bbs[6].RenderPS(ps156)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d157 := ps.PhiValues[0]
					ctx.EnsureDesc(&d157)
					ctx.EmitStoreToStack(d157, int32(bbs[7].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d158 := ps.PhiValues[1]
					ctx.EnsureDesc(&d158)
					ctx.EmitStoreToStack(d158, int32(bbs[7].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl31 := ctx.ReserveLabel()
			lbl32 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d154.Reg, 0)
			ctx.EmitJcc(CcNE, lbl31)
			ctx.EmitJmp(lbl32)
			ctx.MarkLabel(lbl31)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl32)
			ctx.EmitJmp(lbl7)
			ps159 := PhiState{General: true}
			ps159.OverlayValues = make([]JITValueDesc, 159)
			ps159.OverlayValues[0] = d0
			ps159.OverlayValues[1] = d1
			ps159.OverlayValues[2] = d2
			ps159.OverlayValues[3] = d3
			ps159.OverlayValues[4] = d4
			ps159.OverlayValues[5] = d5
			ps159.OverlayValues[6] = d6
			ps159.OverlayValues[8] = d8
			ps159.OverlayValues[9] = d9
			ps159.OverlayValues[10] = d10
			ps159.OverlayValues[11] = d11
			ps159.OverlayValues[12] = d12
			ps159.OverlayValues[15] = d15
			ps159.OverlayValues[32] = d32
			ps159.OverlayValues[33] = d33
			ps159.OverlayValues[34] = d34
			ps159.OverlayValues[35] = d35
			ps159.OverlayValues[36] = d36
			ps159.OverlayValues[38] = d38
			ps159.OverlayValues[40] = d40
			ps159.OverlayValues[41] = d41
			ps159.OverlayValues[44] = d44
			ps159.OverlayValues[69] = d69
			ps159.OverlayValues[70] = d70
			ps159.OverlayValues[71] = d71
			ps159.OverlayValues[72] = d72
			ps159.OverlayValues[73] = d73
			ps159.OverlayValues[74] = d74
			ps159.OverlayValues[75] = d75
			ps159.OverlayValues[110] = d110
			ps159.OverlayValues[111] = d111
			ps159.OverlayValues[112] = d112
			ps159.OverlayValues[150] = d150
			ps159.OverlayValues[151] = d151
			ps159.OverlayValues[152] = d152
			ps159.OverlayValues[153] = d153
			ps159.OverlayValues[154] = d154
			ps159.OverlayValues[157] = d157
			ps159.OverlayValues[158] = d158
			ps160 := PhiState{General: true}
			ps160.OverlayValues = make([]JITValueDesc, 159)
			ps160.OverlayValues[0] = d0
			ps160.OverlayValues[1] = d1
			ps160.OverlayValues[2] = d2
			ps160.OverlayValues[3] = d3
			ps160.OverlayValues[4] = d4
			ps160.OverlayValues[5] = d5
			ps160.OverlayValues[6] = d6
			ps160.OverlayValues[8] = d8
			ps160.OverlayValues[9] = d9
			ps160.OverlayValues[10] = d10
			ps160.OverlayValues[11] = d11
			ps160.OverlayValues[12] = d12
			ps160.OverlayValues[15] = d15
			ps160.OverlayValues[32] = d32
			ps160.OverlayValues[33] = d33
			ps160.OverlayValues[34] = d34
			ps160.OverlayValues[35] = d35
			ps160.OverlayValues[36] = d36
			ps160.OverlayValues[38] = d38
			ps160.OverlayValues[40] = d40
			ps160.OverlayValues[41] = d41
			ps160.OverlayValues[44] = d44
			ps160.OverlayValues[69] = d69
			ps160.OverlayValues[70] = d70
			ps160.OverlayValues[71] = d71
			ps160.OverlayValues[72] = d72
			ps160.OverlayValues[73] = d73
			ps160.OverlayValues[74] = d74
			ps160.OverlayValues[75] = d75
			ps160.OverlayValues[110] = d110
			ps160.OverlayValues[111] = d111
			ps160.OverlayValues[112] = d112
			ps160.OverlayValues[150] = d150
			ps160.OverlayValues[151] = d151
			ps160.OverlayValues[152] = d152
			ps160.OverlayValues[153] = d153
			ps160.OverlayValues[154] = d154
			ps160.OverlayValues[157] = d157
			ps160.OverlayValues[158] = d158
			snap161 := d0
			snap162 := d1
			snap163 := d2
			snap164 := d3
			snap165 := d4
			snap166 := d5
			snap167 := d6
			snap168 := d8
			snap169 := d9
			snap170 := d10
			snap171 := d11
			snap172 := d12
			snap173 := d15
			snap174 := d32
			snap175 := d33
			snap176 := d34
			snap177 := d35
			snap178 := d36
			snap179 := d38
			snap180 := d40
			snap181 := d41
			snap182 := d44
			snap183 := d69
			snap184 := d70
			snap185 := d71
			snap186 := d72
			snap187 := d73
			snap188 := d74
			snap189 := d75
			snap190 := d110
			snap191 := d111
			snap192 := d112
			snap193 := d150
			snap194 := d151
			snap195 := d152
			snap196 := d153
			snap197 := d154
			snap198 := d157
			snap199 := d158
			alloc200 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps160)
			}
			ctx.RestoreAllocState(alloc200)
			d0 = snap161
			d1 = snap162
			d2 = snap163
			d3 = snap164
			d4 = snap165
			d5 = snap166
			d6 = snap167
			d8 = snap168
			d9 = snap169
			d10 = snap170
			d11 = snap171
			d12 = snap172
			d15 = snap173
			d32 = snap174
			d33 = snap175
			d34 = snap176
			d35 = snap177
			d36 = snap178
			d38 = snap179
			d40 = snap180
			d41 = snap181
			d44 = snap182
			d69 = snap183
			d70 = snap184
			d71 = snap185
			d72 = snap186
			d73 = snap187
			d74 = snap188
			d75 = snap189
			d110 = snap190
			d111 = snap191
			d112 = snap192
			d150 = snap193
			d151 = snap194
			d152 = snap195
			d153 = snap196
			d154 = snap197
			d157 = snap198
			d158 = snap199
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps159)
			}
			return result
			ctx.FreeDesc(&d153)
			return result
			}
			bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d201 := ps.PhiValues[0]
					ctx.EnsureDesc(&d201)
					ctx.EmitStoreToStack(d201, int32(bbs[8].PhiBase)+int32(0))
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d3 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d202 JITValueDesc
			if d2.Loc == LocImm {
				d202 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d202 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			}
			if d202.Loc == LocReg && d2.Loc == LocReg && d202.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = LocNone
			}
			ctx.EnsureDesc(&d3)
			if d3.Loc == LocReg {
				ctx.ProtectReg(d3.Reg)
			} else if d3.Loc == LocRegPair {
				ctx.ProtectReg(d3.Reg)
				ctx.ProtectReg(d3.Reg2)
			}
			ctx.EnsureDesc(&d202)
			if d202.Loc == LocReg {
				ctx.ProtectReg(d202.Reg)
			} else if d202.Loc == LocRegPair {
				ctx.ProtectReg(d202.Reg)
				ctx.ProtectReg(d202.Reg2)
			}
			d203 = d3
			if d203.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d203)
			ctx.EmitStoreToStack(d203, int32(bbs[7].PhiBase)+int32(0))
			d204 = d202
			if d204.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d204)
			ctx.EmitStoreToStack(d204, int32(bbs[7].PhiBase)+int32(16))
			if d3.Loc == LocReg {
				ctx.UnprotectReg(d3.Reg)
			} else if d3.Loc == LocRegPair {
				ctx.UnprotectReg(d3.Reg)
				ctx.UnprotectReg(d3.Reg2)
			}
			if d202.Loc == LocReg {
				ctx.UnprotectReg(d202.Reg)
			} else if d202.Loc == LocRegPair {
				ctx.UnprotectReg(d202.Reg)
				ctx.UnprotectReg(d202.Reg2)
			}
			ps205 := PhiState{General: ps.General}
			ps205.OverlayValues = make([]JITValueDesc, 205)
			ps205.OverlayValues[0] = d0
			ps205.OverlayValues[1] = d1
			ps205.OverlayValues[2] = d2
			ps205.OverlayValues[3] = d3
			ps205.OverlayValues[4] = d4
			ps205.OverlayValues[5] = d5
			ps205.OverlayValues[6] = d6
			ps205.OverlayValues[8] = d8
			ps205.OverlayValues[9] = d9
			ps205.OverlayValues[10] = d10
			ps205.OverlayValues[11] = d11
			ps205.OverlayValues[12] = d12
			ps205.OverlayValues[15] = d15
			ps205.OverlayValues[32] = d32
			ps205.OverlayValues[33] = d33
			ps205.OverlayValues[34] = d34
			ps205.OverlayValues[35] = d35
			ps205.OverlayValues[36] = d36
			ps205.OverlayValues[38] = d38
			ps205.OverlayValues[40] = d40
			ps205.OverlayValues[41] = d41
			ps205.OverlayValues[44] = d44
			ps205.OverlayValues[69] = d69
			ps205.OverlayValues[70] = d70
			ps205.OverlayValues[71] = d71
			ps205.OverlayValues[72] = d72
			ps205.OverlayValues[73] = d73
			ps205.OverlayValues[74] = d74
			ps205.OverlayValues[75] = d75
			ps205.OverlayValues[110] = d110
			ps205.OverlayValues[111] = d111
			ps205.OverlayValues[112] = d112
			ps205.OverlayValues[150] = d150
			ps205.OverlayValues[151] = d151
			ps205.OverlayValues[152] = d152
			ps205.OverlayValues[153] = d153
			ps205.OverlayValues[154] = d154
			ps205.OverlayValues[157] = d157
			ps205.OverlayValues[158] = d158
			ps205.OverlayValues[201] = d201
			ps205.OverlayValues[202] = d202
			ps205.OverlayValues[203] = d203
			ps205.OverlayValues[204] = d204
			ps205.PhiValues = make([]JITValueDesc, 2)
			d206 = d3
			ps205.PhiValues[0] = d206
			d207 = d202
			ps205.PhiValues[1] = d207
			if ps205.General && bbs[7].Rendered {
				ctx.EmitJmp(lbl8)
				return result
			}
			return bbs[7].RenderPS(ps205)
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
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			ctx.ReclaimUntrackedRegs()
			var d208 JITValueDesc
			if d71.Loc == LocImm {
				d208 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d71.Imm.Int())}
			} else if d71.Type == tagInt && d71.Loc == LocRegPair {
				ctx.FreeReg(d71.Reg)
				d208 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg2}
				ctx.BindReg(d71.Reg2, &d208)
				ctx.BindReg(d71.Reg2, &d208)
			} else if d71.Type == tagInt && d71.Loc == LocReg {
				d208 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d71.Reg}
				ctx.BindReg(d71.Reg, &d208)
				ctx.BindReg(d71.Reg, &d208)
			} else {
				d208 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d71}, 1)
				d208.Type = tagInt
				ctx.BindReg(d208.Reg, &d208)
			}
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d208)
			var d209 JITValueDesc
			if d1.Loc == LocImm && d208.Loc == LocImm {
				d209 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() * d208.Imm.Int())}
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.EmitImulInt64(scratch, d208.Reg)
				d209 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else if d208.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d208.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d208.Imm.Int()))
					ctx.EmitImulInt64(scratch, RegR11)
				}
				d209 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else {
				r14 := ctx.AllocRegExcept(d1.Reg, d208.Reg)
				ctx.EmitMovRegReg(r14, d1.Reg)
				ctx.EmitImulInt64(r14, d208.Reg)
				d209 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
				ctx.BindReg(r14, &d209)
			}
			if d209.Loc == LocReg && d1.Loc == LocReg && d209.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d209)
			if d209.Loc == LocReg {
				ctx.ProtectReg(d209.Reg)
			} else if d209.Loc == LocRegPair {
				ctx.ProtectReg(d209.Reg)
				ctx.ProtectReg(d209.Reg2)
			}
			d210 = d209
			if d210.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			ctx.EmitStoreToStack(d210, int32(bbs[8].PhiBase)+int32(0))
			if d209.Loc == LocReg {
				ctx.UnprotectReg(d209.Reg)
			} else if d209.Loc == LocRegPair {
				ctx.UnprotectReg(d209.Reg)
				ctx.UnprotectReg(d209.Reg2)
			}
			ps211 := PhiState{General: ps.General}
			ps211.OverlayValues = make([]JITValueDesc, 211)
			ps211.OverlayValues[0] = d0
			ps211.OverlayValues[1] = d1
			ps211.OverlayValues[2] = d2
			ps211.OverlayValues[3] = d3
			ps211.OverlayValues[4] = d4
			ps211.OverlayValues[5] = d5
			ps211.OverlayValues[6] = d6
			ps211.OverlayValues[8] = d8
			ps211.OverlayValues[9] = d9
			ps211.OverlayValues[10] = d10
			ps211.OverlayValues[11] = d11
			ps211.OverlayValues[12] = d12
			ps211.OverlayValues[15] = d15
			ps211.OverlayValues[32] = d32
			ps211.OverlayValues[33] = d33
			ps211.OverlayValues[34] = d34
			ps211.OverlayValues[35] = d35
			ps211.OverlayValues[36] = d36
			ps211.OverlayValues[38] = d38
			ps211.OverlayValues[40] = d40
			ps211.OverlayValues[41] = d41
			ps211.OverlayValues[44] = d44
			ps211.OverlayValues[69] = d69
			ps211.OverlayValues[70] = d70
			ps211.OverlayValues[71] = d71
			ps211.OverlayValues[72] = d72
			ps211.OverlayValues[73] = d73
			ps211.OverlayValues[74] = d74
			ps211.OverlayValues[75] = d75
			ps211.OverlayValues[110] = d110
			ps211.OverlayValues[111] = d111
			ps211.OverlayValues[112] = d112
			ps211.OverlayValues[150] = d150
			ps211.OverlayValues[151] = d151
			ps211.OverlayValues[152] = d152
			ps211.OverlayValues[153] = d153
			ps211.OverlayValues[154] = d154
			ps211.OverlayValues[157] = d157
			ps211.OverlayValues[158] = d158
			ps211.OverlayValues[201] = d201
			ps211.OverlayValues[202] = d202
			ps211.OverlayValues[203] = d203
			ps211.OverlayValues[204] = d204
			ps211.OverlayValues[206] = d206
			ps211.OverlayValues[207] = d207
			ps211.OverlayValues[208] = d208
			ps211.OverlayValues[209] = d209
			ps211.OverlayValues[210] = d210
			ps211.PhiValues = make([]JITValueDesc, 1)
			d212 = d209
			ps211.PhiValues[0] = d212
			if ps211.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps211)
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
					ctx.EmitJmp(lbl11)
					return result
				}
				bbs[10].Rendered = true
				bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_10 = bbs[10].Address
				ctx.MarkLabel(lbl11)
				ctx.ResolveFixups()
			}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			ctx.ReclaimUntrackedRegs()
			d214 = d71
			d214.ID = 0
			d213 = ctx.EmitTagEqualsBorrowed(&d214, tagFloat, JITValueDesc{Loc: LocAny})
			d215 = d213
			ctx.EnsureDesc(&d215)
			if d215.Loc != LocImm && d215.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d215.Loc == LocImm {
				if d215.Imm.Bool() {
			ps216 := PhiState{General: ps.General}
			ps216.OverlayValues = make([]JITValueDesc, 216)
			ps216.OverlayValues[0] = d0
			ps216.OverlayValues[1] = d1
			ps216.OverlayValues[2] = d2
			ps216.OverlayValues[3] = d3
			ps216.OverlayValues[4] = d4
			ps216.OverlayValues[5] = d5
			ps216.OverlayValues[6] = d6
			ps216.OverlayValues[8] = d8
			ps216.OverlayValues[9] = d9
			ps216.OverlayValues[10] = d10
			ps216.OverlayValues[11] = d11
			ps216.OverlayValues[12] = d12
			ps216.OverlayValues[15] = d15
			ps216.OverlayValues[32] = d32
			ps216.OverlayValues[33] = d33
			ps216.OverlayValues[34] = d34
			ps216.OverlayValues[35] = d35
			ps216.OverlayValues[36] = d36
			ps216.OverlayValues[38] = d38
			ps216.OverlayValues[40] = d40
			ps216.OverlayValues[41] = d41
			ps216.OverlayValues[44] = d44
			ps216.OverlayValues[69] = d69
			ps216.OverlayValues[70] = d70
			ps216.OverlayValues[71] = d71
			ps216.OverlayValues[72] = d72
			ps216.OverlayValues[73] = d73
			ps216.OverlayValues[74] = d74
			ps216.OverlayValues[75] = d75
			ps216.OverlayValues[110] = d110
			ps216.OverlayValues[111] = d111
			ps216.OverlayValues[112] = d112
			ps216.OverlayValues[150] = d150
			ps216.OverlayValues[151] = d151
			ps216.OverlayValues[152] = d152
			ps216.OverlayValues[153] = d153
			ps216.OverlayValues[154] = d154
			ps216.OverlayValues[157] = d157
			ps216.OverlayValues[158] = d158
			ps216.OverlayValues[201] = d201
			ps216.OverlayValues[202] = d202
			ps216.OverlayValues[203] = d203
			ps216.OverlayValues[204] = d204
			ps216.OverlayValues[206] = d206
			ps216.OverlayValues[207] = d207
			ps216.OverlayValues[208] = d208
			ps216.OverlayValues[209] = d209
			ps216.OverlayValues[210] = d210
			ps216.OverlayValues[212] = d212
			ps216.OverlayValues[213] = d213
			ps216.OverlayValues[214] = d214
			ps216.OverlayValues[215] = d215
					return bbs[11].RenderPS(ps216)
				}
			ps217 := PhiState{General: ps.General}
			ps217.OverlayValues = make([]JITValueDesc, 216)
			ps217.OverlayValues[0] = d0
			ps217.OverlayValues[1] = d1
			ps217.OverlayValues[2] = d2
			ps217.OverlayValues[3] = d3
			ps217.OverlayValues[4] = d4
			ps217.OverlayValues[5] = d5
			ps217.OverlayValues[6] = d6
			ps217.OverlayValues[8] = d8
			ps217.OverlayValues[9] = d9
			ps217.OverlayValues[10] = d10
			ps217.OverlayValues[11] = d11
			ps217.OverlayValues[12] = d12
			ps217.OverlayValues[15] = d15
			ps217.OverlayValues[32] = d32
			ps217.OverlayValues[33] = d33
			ps217.OverlayValues[34] = d34
			ps217.OverlayValues[35] = d35
			ps217.OverlayValues[36] = d36
			ps217.OverlayValues[38] = d38
			ps217.OverlayValues[40] = d40
			ps217.OverlayValues[41] = d41
			ps217.OverlayValues[44] = d44
			ps217.OverlayValues[69] = d69
			ps217.OverlayValues[70] = d70
			ps217.OverlayValues[71] = d71
			ps217.OverlayValues[72] = d72
			ps217.OverlayValues[73] = d73
			ps217.OverlayValues[74] = d74
			ps217.OverlayValues[75] = d75
			ps217.OverlayValues[110] = d110
			ps217.OverlayValues[111] = d111
			ps217.OverlayValues[112] = d112
			ps217.OverlayValues[150] = d150
			ps217.OverlayValues[151] = d151
			ps217.OverlayValues[152] = d152
			ps217.OverlayValues[153] = d153
			ps217.OverlayValues[154] = d154
			ps217.OverlayValues[157] = d157
			ps217.OverlayValues[158] = d158
			ps217.OverlayValues[201] = d201
			ps217.OverlayValues[202] = d202
			ps217.OverlayValues[203] = d203
			ps217.OverlayValues[204] = d204
			ps217.OverlayValues[206] = d206
			ps217.OverlayValues[207] = d207
			ps217.OverlayValues[208] = d208
			ps217.OverlayValues[209] = d209
			ps217.OverlayValues[210] = d210
			ps217.OverlayValues[212] = d212
			ps217.OverlayValues[213] = d213
			ps217.OverlayValues[214] = d214
			ps217.OverlayValues[215] = d215
				return bbs[6].RenderPS(ps217)
			}
			if !ps.General {
				ps.General = true
				return bbs[10].RenderPS(ps)
			}
			lbl33 := ctx.ReserveLabel()
			lbl34 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d215.Reg, 0)
			ctx.EmitJcc(CcNE, lbl33)
			ctx.EmitJmp(lbl34)
			ctx.MarkLabel(lbl33)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl34)
			ctx.EmitJmp(lbl7)
			ps218 := PhiState{General: true}
			ps218.OverlayValues = make([]JITValueDesc, 216)
			ps218.OverlayValues[0] = d0
			ps218.OverlayValues[1] = d1
			ps218.OverlayValues[2] = d2
			ps218.OverlayValues[3] = d3
			ps218.OverlayValues[4] = d4
			ps218.OverlayValues[5] = d5
			ps218.OverlayValues[6] = d6
			ps218.OverlayValues[8] = d8
			ps218.OverlayValues[9] = d9
			ps218.OverlayValues[10] = d10
			ps218.OverlayValues[11] = d11
			ps218.OverlayValues[12] = d12
			ps218.OverlayValues[15] = d15
			ps218.OverlayValues[32] = d32
			ps218.OverlayValues[33] = d33
			ps218.OverlayValues[34] = d34
			ps218.OverlayValues[35] = d35
			ps218.OverlayValues[36] = d36
			ps218.OverlayValues[38] = d38
			ps218.OverlayValues[40] = d40
			ps218.OverlayValues[41] = d41
			ps218.OverlayValues[44] = d44
			ps218.OverlayValues[69] = d69
			ps218.OverlayValues[70] = d70
			ps218.OverlayValues[71] = d71
			ps218.OverlayValues[72] = d72
			ps218.OverlayValues[73] = d73
			ps218.OverlayValues[74] = d74
			ps218.OverlayValues[75] = d75
			ps218.OverlayValues[110] = d110
			ps218.OverlayValues[111] = d111
			ps218.OverlayValues[112] = d112
			ps218.OverlayValues[150] = d150
			ps218.OverlayValues[151] = d151
			ps218.OverlayValues[152] = d152
			ps218.OverlayValues[153] = d153
			ps218.OverlayValues[154] = d154
			ps218.OverlayValues[157] = d157
			ps218.OverlayValues[158] = d158
			ps218.OverlayValues[201] = d201
			ps218.OverlayValues[202] = d202
			ps218.OverlayValues[203] = d203
			ps218.OverlayValues[204] = d204
			ps218.OverlayValues[206] = d206
			ps218.OverlayValues[207] = d207
			ps218.OverlayValues[208] = d208
			ps218.OverlayValues[209] = d209
			ps218.OverlayValues[210] = d210
			ps218.OverlayValues[212] = d212
			ps218.OverlayValues[213] = d213
			ps218.OverlayValues[214] = d214
			ps218.OverlayValues[215] = d215
			ps219 := PhiState{General: true}
			ps219.OverlayValues = make([]JITValueDesc, 216)
			ps219.OverlayValues[0] = d0
			ps219.OverlayValues[1] = d1
			ps219.OverlayValues[2] = d2
			ps219.OverlayValues[3] = d3
			ps219.OverlayValues[4] = d4
			ps219.OverlayValues[5] = d5
			ps219.OverlayValues[6] = d6
			ps219.OverlayValues[8] = d8
			ps219.OverlayValues[9] = d9
			ps219.OverlayValues[10] = d10
			ps219.OverlayValues[11] = d11
			ps219.OverlayValues[12] = d12
			ps219.OverlayValues[15] = d15
			ps219.OverlayValues[32] = d32
			ps219.OverlayValues[33] = d33
			ps219.OverlayValues[34] = d34
			ps219.OverlayValues[35] = d35
			ps219.OverlayValues[36] = d36
			ps219.OverlayValues[38] = d38
			ps219.OverlayValues[40] = d40
			ps219.OverlayValues[41] = d41
			ps219.OverlayValues[44] = d44
			ps219.OverlayValues[69] = d69
			ps219.OverlayValues[70] = d70
			ps219.OverlayValues[71] = d71
			ps219.OverlayValues[72] = d72
			ps219.OverlayValues[73] = d73
			ps219.OverlayValues[74] = d74
			ps219.OverlayValues[75] = d75
			ps219.OverlayValues[110] = d110
			ps219.OverlayValues[111] = d111
			ps219.OverlayValues[112] = d112
			ps219.OverlayValues[150] = d150
			ps219.OverlayValues[151] = d151
			ps219.OverlayValues[152] = d152
			ps219.OverlayValues[153] = d153
			ps219.OverlayValues[154] = d154
			ps219.OverlayValues[157] = d157
			ps219.OverlayValues[158] = d158
			ps219.OverlayValues[201] = d201
			ps219.OverlayValues[202] = d202
			ps219.OverlayValues[203] = d203
			ps219.OverlayValues[204] = d204
			ps219.OverlayValues[206] = d206
			ps219.OverlayValues[207] = d207
			ps219.OverlayValues[208] = d208
			ps219.OverlayValues[209] = d209
			ps219.OverlayValues[210] = d210
			ps219.OverlayValues[212] = d212
			ps219.OverlayValues[213] = d213
			ps219.OverlayValues[214] = d214
			ps219.OverlayValues[215] = d215
			snap220 := d0
			snap221 := d1
			snap222 := d2
			snap223 := d3
			snap224 := d4
			snap225 := d5
			snap226 := d6
			snap227 := d8
			snap228 := d9
			snap229 := d10
			snap230 := d11
			snap231 := d12
			snap232 := d15
			snap233 := d32
			snap234 := d33
			snap235 := d34
			snap236 := d35
			snap237 := d36
			snap238 := d38
			snap239 := d40
			snap240 := d41
			snap241 := d44
			snap242 := d69
			snap243 := d70
			snap244 := d71
			snap245 := d72
			snap246 := d73
			snap247 := d74
			snap248 := d75
			snap249 := d110
			snap250 := d111
			snap251 := d112
			snap252 := d150
			snap253 := d151
			snap254 := d152
			snap255 := d153
			snap256 := d154
			snap257 := d157
			snap258 := d158
			snap259 := d201
			snap260 := d202
			snap261 := d203
			snap262 := d204
			snap263 := d206
			snap264 := d207
			snap265 := d208
			snap266 := d209
			snap267 := d210
			snap268 := d212
			snap269 := d213
			snap270 := d214
			snap271 := d215
			alloc272 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps219)
			}
			ctx.RestoreAllocState(alloc272)
			d0 = snap220
			d1 = snap221
			d2 = snap222
			d3 = snap223
			d4 = snap224
			d5 = snap225
			d6 = snap226
			d8 = snap227
			d9 = snap228
			d10 = snap229
			d11 = snap230
			d12 = snap231
			d15 = snap232
			d32 = snap233
			d33 = snap234
			d34 = snap235
			d35 = snap236
			d36 = snap237
			d38 = snap238
			d40 = snap239
			d41 = snap240
			d44 = snap241
			d69 = snap242
			d70 = snap243
			d71 = snap244
			d72 = snap245
			d73 = snap246
			d74 = snap247
			d75 = snap248
			d110 = snap249
			d111 = snap250
			d112 = snap251
			d150 = snap252
			d151 = snap253
			d152 = snap254
			d153 = snap255
			d154 = snap256
			d157 = snap257
			d158 = snap258
			d201 = snap259
			d202 = snap260
			d203 = snap261
			d204 = snap262
			d206 = snap263
			d207 = snap264
			d208 = snap265
			d209 = snap266
			d210 = snap267
			d212 = snap268
			d213 = snap269
			d214 = snap270
			d215 = snap271
			if !bbs[11].Rendered {
				return bbs[11].RenderPS(ps218)
			}
			return result
			ctx.FreeDesc(&d213)
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
					ctx.EmitJmp(lbl12)
					return result
				}
				bbs[11].Rendered = true
				bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_11 = bbs[11].Address
				ctx.MarkLabel(lbl12)
				ctx.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
				d215 = ps.OverlayValues[215]
			}
			ctx.ReclaimUntrackedRegs()
			var d273 JITValueDesc
			if d71.Loc == LocImm {
				d273 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d71.Imm.Float())}
			} else if d71.Type == tagFloat && d71.Loc == LocReg {
				d273 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d71.Reg}
				ctx.BindReg(d71.Reg, &d273)
				ctx.BindReg(d71.Reg, &d273)
			} else if d71.Type == tagFloat && d71.Loc == LocRegPair {
				ctx.FreeReg(d71.Reg)
				d273 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d71.Reg2}
				ctx.BindReg(d71.Reg2, &d273)
				ctx.BindReg(d71.Reg2, &d273)
			} else {
				d273 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d71}, 1)
				d273.Type = tagFloat
				ctx.BindReg(d273.Reg, &d273)
			}
			ctx.FreeDesc(&d71)
			ctx.EnsureDesc(&d273)
			var d274 JITValueDesc
			if d273.Loc == LocImm {
				d274 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Trunc(d273.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d273)
				var truncSrc Reg
				if d273.Loc == LocRegPair {
					ctx.FreeReg(d273.Reg)
					truncSrc = d273.Reg2
				} else {
					truncSrc = d273.Reg
				}
				truncInt := ctx.AllocRegExcept(truncSrc)
				ctx.EmitCvtFloatBitsToInt64(truncInt, truncSrc)
				ctx.EmitCvtInt64ToFloat64(RegX0, truncInt)
				d274 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: truncInt}
				ctx.BindReg(truncInt, &d274)
				ctx.BindReg(truncInt, &d274)
			}
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d274)
			var d275 JITValueDesc
			if d273.Loc == LocImm && d274.Loc == LocImm {
				d275 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d273.Imm.Float() == d274.Imm.Float())}
			} else if d274.Loc == LocImm {
				r15 := ctx.AllocRegExcept(d273.Reg)
				_, yBits := d274.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, yBits)
				ctx.EmitCmpFloat64Setcc(r15, d273.Reg, RegR11, CcE)
				d275 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r15}
				ctx.BindReg(r15, &d275)
			} else if d273.Loc == LocImm {
				r16 := ctx.AllocRegExcept(d274.Reg)
				_, xBits := d273.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, xBits)
				ctx.EmitCmpFloat64Setcc(r16, RegR11, d274.Reg, CcE)
				d275 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r16}
				ctx.BindReg(r16, &d275)
			} else {
				r17 := ctx.AllocRegExcept(d273.Reg, d274.Reg)
				ctx.EmitCmpFloat64Setcc(r17, d273.Reg, d274.Reg, CcE)
				d275 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r17}
				ctx.BindReg(r17, &d275)
			}
			ctx.FreeDesc(&d274)
			d276 = d275
			ctx.EnsureDesc(&d276)
			if d276.Loc != LocImm && d276.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d276.Loc == LocImm {
				if d276.Imm.Bool() {
			ps277 := PhiState{General: ps.General}
			ps277.OverlayValues = make([]JITValueDesc, 277)
			ps277.OverlayValues[0] = d0
			ps277.OverlayValues[1] = d1
			ps277.OverlayValues[2] = d2
			ps277.OverlayValues[3] = d3
			ps277.OverlayValues[4] = d4
			ps277.OverlayValues[5] = d5
			ps277.OverlayValues[6] = d6
			ps277.OverlayValues[8] = d8
			ps277.OverlayValues[9] = d9
			ps277.OverlayValues[10] = d10
			ps277.OverlayValues[11] = d11
			ps277.OverlayValues[12] = d12
			ps277.OverlayValues[15] = d15
			ps277.OverlayValues[32] = d32
			ps277.OverlayValues[33] = d33
			ps277.OverlayValues[34] = d34
			ps277.OverlayValues[35] = d35
			ps277.OverlayValues[36] = d36
			ps277.OverlayValues[38] = d38
			ps277.OverlayValues[40] = d40
			ps277.OverlayValues[41] = d41
			ps277.OverlayValues[44] = d44
			ps277.OverlayValues[69] = d69
			ps277.OverlayValues[70] = d70
			ps277.OverlayValues[71] = d71
			ps277.OverlayValues[72] = d72
			ps277.OverlayValues[73] = d73
			ps277.OverlayValues[74] = d74
			ps277.OverlayValues[75] = d75
			ps277.OverlayValues[110] = d110
			ps277.OverlayValues[111] = d111
			ps277.OverlayValues[112] = d112
			ps277.OverlayValues[150] = d150
			ps277.OverlayValues[151] = d151
			ps277.OverlayValues[152] = d152
			ps277.OverlayValues[153] = d153
			ps277.OverlayValues[154] = d154
			ps277.OverlayValues[157] = d157
			ps277.OverlayValues[158] = d158
			ps277.OverlayValues[201] = d201
			ps277.OverlayValues[202] = d202
			ps277.OverlayValues[203] = d203
			ps277.OverlayValues[204] = d204
			ps277.OverlayValues[206] = d206
			ps277.OverlayValues[207] = d207
			ps277.OverlayValues[208] = d208
			ps277.OverlayValues[209] = d209
			ps277.OverlayValues[210] = d210
			ps277.OverlayValues[212] = d212
			ps277.OverlayValues[213] = d213
			ps277.OverlayValues[214] = d214
			ps277.OverlayValues[215] = d215
			ps277.OverlayValues[273] = d273
			ps277.OverlayValues[274] = d274
			ps277.OverlayValues[275] = d275
			ps277.OverlayValues[276] = d276
					return bbs[12].RenderPS(ps277)
				}
			ps278 := PhiState{General: ps.General}
			ps278.OverlayValues = make([]JITValueDesc, 277)
			ps278.OverlayValues[0] = d0
			ps278.OverlayValues[1] = d1
			ps278.OverlayValues[2] = d2
			ps278.OverlayValues[3] = d3
			ps278.OverlayValues[4] = d4
			ps278.OverlayValues[5] = d5
			ps278.OverlayValues[6] = d6
			ps278.OverlayValues[8] = d8
			ps278.OverlayValues[9] = d9
			ps278.OverlayValues[10] = d10
			ps278.OverlayValues[11] = d11
			ps278.OverlayValues[12] = d12
			ps278.OverlayValues[15] = d15
			ps278.OverlayValues[32] = d32
			ps278.OverlayValues[33] = d33
			ps278.OverlayValues[34] = d34
			ps278.OverlayValues[35] = d35
			ps278.OverlayValues[36] = d36
			ps278.OverlayValues[38] = d38
			ps278.OverlayValues[40] = d40
			ps278.OverlayValues[41] = d41
			ps278.OverlayValues[44] = d44
			ps278.OverlayValues[69] = d69
			ps278.OverlayValues[70] = d70
			ps278.OverlayValues[71] = d71
			ps278.OverlayValues[72] = d72
			ps278.OverlayValues[73] = d73
			ps278.OverlayValues[74] = d74
			ps278.OverlayValues[75] = d75
			ps278.OverlayValues[110] = d110
			ps278.OverlayValues[111] = d111
			ps278.OverlayValues[112] = d112
			ps278.OverlayValues[150] = d150
			ps278.OverlayValues[151] = d151
			ps278.OverlayValues[152] = d152
			ps278.OverlayValues[153] = d153
			ps278.OverlayValues[154] = d154
			ps278.OverlayValues[157] = d157
			ps278.OverlayValues[158] = d158
			ps278.OverlayValues[201] = d201
			ps278.OverlayValues[202] = d202
			ps278.OverlayValues[203] = d203
			ps278.OverlayValues[204] = d204
			ps278.OverlayValues[206] = d206
			ps278.OverlayValues[207] = d207
			ps278.OverlayValues[208] = d208
			ps278.OverlayValues[209] = d209
			ps278.OverlayValues[210] = d210
			ps278.OverlayValues[212] = d212
			ps278.OverlayValues[213] = d213
			ps278.OverlayValues[214] = d214
			ps278.OverlayValues[215] = d215
			ps278.OverlayValues[273] = d273
			ps278.OverlayValues[274] = d274
			ps278.OverlayValues[275] = d275
			ps278.OverlayValues[276] = d276
				return bbs[6].RenderPS(ps278)
			}
			if !ps.General {
				ps.General = true
				return bbs[11].RenderPS(ps)
			}
			lbl35 := ctx.ReserveLabel()
			lbl36 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d276.Reg, 0)
			ctx.EmitJcc(CcNE, lbl35)
			ctx.EmitJmp(lbl36)
			ctx.MarkLabel(lbl35)
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl36)
			ctx.EmitJmp(lbl7)
			ps279 := PhiState{General: true}
			ps279.OverlayValues = make([]JITValueDesc, 277)
			ps279.OverlayValues[0] = d0
			ps279.OverlayValues[1] = d1
			ps279.OverlayValues[2] = d2
			ps279.OverlayValues[3] = d3
			ps279.OverlayValues[4] = d4
			ps279.OverlayValues[5] = d5
			ps279.OverlayValues[6] = d6
			ps279.OverlayValues[8] = d8
			ps279.OverlayValues[9] = d9
			ps279.OverlayValues[10] = d10
			ps279.OverlayValues[11] = d11
			ps279.OverlayValues[12] = d12
			ps279.OverlayValues[15] = d15
			ps279.OverlayValues[32] = d32
			ps279.OverlayValues[33] = d33
			ps279.OverlayValues[34] = d34
			ps279.OverlayValues[35] = d35
			ps279.OverlayValues[36] = d36
			ps279.OverlayValues[38] = d38
			ps279.OverlayValues[40] = d40
			ps279.OverlayValues[41] = d41
			ps279.OverlayValues[44] = d44
			ps279.OverlayValues[69] = d69
			ps279.OverlayValues[70] = d70
			ps279.OverlayValues[71] = d71
			ps279.OverlayValues[72] = d72
			ps279.OverlayValues[73] = d73
			ps279.OverlayValues[74] = d74
			ps279.OverlayValues[75] = d75
			ps279.OverlayValues[110] = d110
			ps279.OverlayValues[111] = d111
			ps279.OverlayValues[112] = d112
			ps279.OverlayValues[150] = d150
			ps279.OverlayValues[151] = d151
			ps279.OverlayValues[152] = d152
			ps279.OverlayValues[153] = d153
			ps279.OverlayValues[154] = d154
			ps279.OverlayValues[157] = d157
			ps279.OverlayValues[158] = d158
			ps279.OverlayValues[201] = d201
			ps279.OverlayValues[202] = d202
			ps279.OverlayValues[203] = d203
			ps279.OverlayValues[204] = d204
			ps279.OverlayValues[206] = d206
			ps279.OverlayValues[207] = d207
			ps279.OverlayValues[208] = d208
			ps279.OverlayValues[209] = d209
			ps279.OverlayValues[210] = d210
			ps279.OverlayValues[212] = d212
			ps279.OverlayValues[213] = d213
			ps279.OverlayValues[214] = d214
			ps279.OverlayValues[215] = d215
			ps279.OverlayValues[273] = d273
			ps279.OverlayValues[274] = d274
			ps279.OverlayValues[275] = d275
			ps279.OverlayValues[276] = d276
			ps280 := PhiState{General: true}
			ps280.OverlayValues = make([]JITValueDesc, 277)
			ps280.OverlayValues[0] = d0
			ps280.OverlayValues[1] = d1
			ps280.OverlayValues[2] = d2
			ps280.OverlayValues[3] = d3
			ps280.OverlayValues[4] = d4
			ps280.OverlayValues[5] = d5
			ps280.OverlayValues[6] = d6
			ps280.OverlayValues[8] = d8
			ps280.OverlayValues[9] = d9
			ps280.OverlayValues[10] = d10
			ps280.OverlayValues[11] = d11
			ps280.OverlayValues[12] = d12
			ps280.OverlayValues[15] = d15
			ps280.OverlayValues[32] = d32
			ps280.OverlayValues[33] = d33
			ps280.OverlayValues[34] = d34
			ps280.OverlayValues[35] = d35
			ps280.OverlayValues[36] = d36
			ps280.OverlayValues[38] = d38
			ps280.OverlayValues[40] = d40
			ps280.OverlayValues[41] = d41
			ps280.OverlayValues[44] = d44
			ps280.OverlayValues[69] = d69
			ps280.OverlayValues[70] = d70
			ps280.OverlayValues[71] = d71
			ps280.OverlayValues[72] = d72
			ps280.OverlayValues[73] = d73
			ps280.OverlayValues[74] = d74
			ps280.OverlayValues[75] = d75
			ps280.OverlayValues[110] = d110
			ps280.OverlayValues[111] = d111
			ps280.OverlayValues[112] = d112
			ps280.OverlayValues[150] = d150
			ps280.OverlayValues[151] = d151
			ps280.OverlayValues[152] = d152
			ps280.OverlayValues[153] = d153
			ps280.OverlayValues[154] = d154
			ps280.OverlayValues[157] = d157
			ps280.OverlayValues[158] = d158
			ps280.OverlayValues[201] = d201
			ps280.OverlayValues[202] = d202
			ps280.OverlayValues[203] = d203
			ps280.OverlayValues[204] = d204
			ps280.OverlayValues[206] = d206
			ps280.OverlayValues[207] = d207
			ps280.OverlayValues[208] = d208
			ps280.OverlayValues[209] = d209
			ps280.OverlayValues[210] = d210
			ps280.OverlayValues[212] = d212
			ps280.OverlayValues[213] = d213
			ps280.OverlayValues[214] = d214
			ps280.OverlayValues[215] = d215
			ps280.OverlayValues[273] = d273
			ps280.OverlayValues[274] = d274
			ps280.OverlayValues[275] = d275
			ps280.OverlayValues[276] = d276
			snap281 := d0
			snap282 := d1
			snap283 := d2
			snap284 := d3
			snap285 := d4
			snap286 := d5
			snap287 := d6
			snap288 := d8
			snap289 := d9
			snap290 := d10
			snap291 := d11
			snap292 := d12
			snap293 := d15
			snap294 := d32
			snap295 := d33
			snap296 := d34
			snap297 := d35
			snap298 := d36
			snap299 := d38
			snap300 := d40
			snap301 := d41
			snap302 := d44
			snap303 := d69
			snap304 := d70
			snap305 := d71
			snap306 := d72
			snap307 := d73
			snap308 := d74
			snap309 := d75
			snap310 := d110
			snap311 := d111
			snap312 := d112
			snap313 := d150
			snap314 := d151
			snap315 := d152
			snap316 := d153
			snap317 := d154
			snap318 := d157
			snap319 := d158
			snap320 := d201
			snap321 := d202
			snap322 := d203
			snap323 := d204
			snap324 := d206
			snap325 := d207
			snap326 := d208
			snap327 := d209
			snap328 := d210
			snap329 := d212
			snap330 := d213
			snap331 := d214
			snap332 := d215
			snap333 := d273
			snap334 := d274
			snap335 := d275
			snap336 := d276
			alloc337 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps280)
			}
			ctx.RestoreAllocState(alloc337)
			d0 = snap281
			d1 = snap282
			d2 = snap283
			d3 = snap284
			d4 = snap285
			d5 = snap286
			d6 = snap287
			d8 = snap288
			d9 = snap289
			d10 = snap290
			d11 = snap291
			d12 = snap292
			d15 = snap293
			d32 = snap294
			d33 = snap295
			d34 = snap296
			d35 = snap297
			d36 = snap298
			d38 = snap299
			d40 = snap300
			d41 = snap301
			d44 = snap302
			d69 = snap303
			d70 = snap304
			d71 = snap305
			d72 = snap306
			d73 = snap307
			d74 = snap308
			d75 = snap309
			d110 = snap310
			d111 = snap311
			d112 = snap312
			d150 = snap313
			d151 = snap314
			d152 = snap315
			d153 = snap316
			d154 = snap317
			d157 = snap318
			d158 = snap319
			d201 = snap320
			d202 = snap321
			d203 = snap322
			d204 = snap323
			d206 = snap324
			d207 = snap325
			d208 = snap326
			d209 = snap327
			d210 = snap328
			d212 = snap329
			d213 = snap330
			d214 = snap331
			d215 = snap332
			d273 = snap333
			d274 = snap334
			d275 = snap335
			d276 = snap336
			if !bbs[12].Rendered {
				return bbs[12].RenderPS(ps279)
			}
			return result
			ctx.FreeDesc(&d275)
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
					ctx.EmitJmp(lbl13)
					return result
				}
				bbs[12].Rendered = true
				bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_12 = bbs[12].Address
				ctx.MarkLabel(lbl13)
				ctx.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
				d273 = ps.OverlayValues[273]
			}
			if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
				d274 = ps.OverlayValues[274]
			}
			if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
				d275 = ps.OverlayValues[275]
			}
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
				d276 = ps.OverlayValues[276]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d273)
			var d338 JITValueDesc
			if d273.Loc == LocImm {
				d338 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d273.Imm.Float()))}
			} else {
				r18 := ctx.AllocReg()
				ctx.EmitCvtFloatBitsToInt64(r18, d273.Reg)
				d338 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
				ctx.BindReg(r18, &d338)
			}
			ctx.FreeDesc(&d273)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d338)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d338)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d338)
			var d339 JITValueDesc
			if d1.Loc == LocImm && d338.Loc == LocImm {
				d339 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() * d338.Imm.Int())}
			} else if d1.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d338.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.EmitImulInt64(scratch, d338.Reg)
				d339 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d339)
			} else if d338.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				if d338.Imm.Int() >= -2147483648 && d338.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d338.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d338.Imm.Int()))
					ctx.EmitImulInt64(scratch, RegR11)
				}
				d339 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d339)
			} else {
				r19 := ctx.AllocRegExcept(d1.Reg, d338.Reg)
				ctx.EmitMovRegReg(r19, d1.Reg)
				ctx.EmitImulInt64(r19, d338.Reg)
				d339 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
				ctx.BindReg(r19, &d339)
			}
			if d339.Loc == LocReg && d1.Loc == LocReg && d339.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = LocNone
			}
			ctx.FreeDesc(&d338)
			ctx.EnsureDesc(&d339)
			if d339.Loc == LocReg {
				ctx.ProtectReg(d339.Reg)
			} else if d339.Loc == LocRegPair {
				ctx.ProtectReg(d339.Reg)
				ctx.ProtectReg(d339.Reg2)
			}
			d340 = d339
			if d340.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d340)
			ctx.EmitStoreToStack(d340, int32(bbs[8].PhiBase)+int32(0))
			if d339.Loc == LocReg {
				ctx.UnprotectReg(d339.Reg)
			} else if d339.Loc == LocRegPair {
				ctx.UnprotectReg(d339.Reg)
				ctx.UnprotectReg(d339.Reg2)
			}
			ps341 := PhiState{General: ps.General}
			ps341.OverlayValues = make([]JITValueDesc, 341)
			ps341.OverlayValues[0] = d0
			ps341.OverlayValues[1] = d1
			ps341.OverlayValues[2] = d2
			ps341.OverlayValues[3] = d3
			ps341.OverlayValues[4] = d4
			ps341.OverlayValues[5] = d5
			ps341.OverlayValues[6] = d6
			ps341.OverlayValues[8] = d8
			ps341.OverlayValues[9] = d9
			ps341.OverlayValues[10] = d10
			ps341.OverlayValues[11] = d11
			ps341.OverlayValues[12] = d12
			ps341.OverlayValues[15] = d15
			ps341.OverlayValues[32] = d32
			ps341.OverlayValues[33] = d33
			ps341.OverlayValues[34] = d34
			ps341.OverlayValues[35] = d35
			ps341.OverlayValues[36] = d36
			ps341.OverlayValues[38] = d38
			ps341.OverlayValues[40] = d40
			ps341.OverlayValues[41] = d41
			ps341.OverlayValues[44] = d44
			ps341.OverlayValues[69] = d69
			ps341.OverlayValues[70] = d70
			ps341.OverlayValues[71] = d71
			ps341.OverlayValues[72] = d72
			ps341.OverlayValues[73] = d73
			ps341.OverlayValues[74] = d74
			ps341.OverlayValues[75] = d75
			ps341.OverlayValues[110] = d110
			ps341.OverlayValues[111] = d111
			ps341.OverlayValues[112] = d112
			ps341.OverlayValues[150] = d150
			ps341.OverlayValues[151] = d151
			ps341.OverlayValues[152] = d152
			ps341.OverlayValues[153] = d153
			ps341.OverlayValues[154] = d154
			ps341.OverlayValues[157] = d157
			ps341.OverlayValues[158] = d158
			ps341.OverlayValues[201] = d201
			ps341.OverlayValues[202] = d202
			ps341.OverlayValues[203] = d203
			ps341.OverlayValues[204] = d204
			ps341.OverlayValues[206] = d206
			ps341.OverlayValues[207] = d207
			ps341.OverlayValues[208] = d208
			ps341.OverlayValues[209] = d209
			ps341.OverlayValues[210] = d210
			ps341.OverlayValues[212] = d212
			ps341.OverlayValues[213] = d213
			ps341.OverlayValues[214] = d214
			ps341.OverlayValues[215] = d215
			ps341.OverlayValues[273] = d273
			ps341.OverlayValues[274] = d274
			ps341.OverlayValues[275] = d275
			ps341.OverlayValues[276] = d276
			ps341.OverlayValues[338] = d338
			ps341.OverlayValues[339] = d339
			ps341.OverlayValues[340] = d340
			ps341.PhiValues = make([]JITValueDesc, 1)
			d342 = d339
			ps341.PhiValues[0] = d342
			if ps341.General && bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			return bbs[8].RenderPS(ps341)
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
					ctx.EmitJmp(lbl14)
					return result
				}
				bbs[13].Rendered = true
				bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_13 = bbs[13].Address
				ctx.MarkLabel(lbl14)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
				d273 = ps.OverlayValues[273]
			}
			if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
				d274 = ps.OverlayValues[274]
			}
			if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
				d275 = ps.OverlayValues[275]
			}
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
				d276 = ps.OverlayValues[276]
			}
			if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
				d338 = ps.OverlayValues[338]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			ctx.EmitMakeInt(result, d1)
			if d1.Loc == LocReg { ctx.FreeReg(d1.Reg) }
			result.Type = tagInt
			ctx.EmitJmp(lbl0)
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
					ctx.EmitJmp(lbl15)
					return result
				}
				bbs[14].Rendered = true
				bbs[14].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_14 = bbs[14].Address
				ctx.MarkLabel(lbl15)
				ctx.ResolveFixups()
			}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
				d273 = ps.OverlayValues[273]
			}
			if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
				d274 = ps.OverlayValues[274]
			}
			if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
				d275 = ps.OverlayValues[275]
			}
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
				d276 = ps.OverlayValues[276]
			}
			if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
				d338 = ps.OverlayValues[338]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d343 JITValueDesc
			if d1.Loc == LocImm {
				d343 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(float64(d1.Imm.Int()))}
			} else {
				ctx.EmitCvtInt64ToFloat64(RegX0, d1.Reg)
				d343 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d343)
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			ctx.EnsureDesc(&d343)
			if d343.Loc == LocReg {
				ctx.ProtectReg(d343.Reg)
			} else if d343.Loc == LocRegPair {
				ctx.ProtectReg(d343.Reg)
				ctx.ProtectReg(d343.Reg2)
			}
			d344 = d2
			if d344.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d344)
			ctx.EmitStoreToStack(d344, int32(bbs[17].PhiBase)+int32(0))
			d345 = d343
			if d345.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d345)
			ctx.EmitStoreToStack(d345, int32(bbs[17].PhiBase)+int32(16))
			if d2.Loc == LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
			if d343.Loc == LocReg {
				ctx.UnprotectReg(d343.Reg)
			} else if d343.Loc == LocRegPair {
				ctx.UnprotectReg(d343.Reg)
				ctx.UnprotectReg(d343.Reg2)
			}
			ps346 := PhiState{General: ps.General}
			ps346.OverlayValues = make([]JITValueDesc, 346)
			ps346.OverlayValues[0] = d0
			ps346.OverlayValues[1] = d1
			ps346.OverlayValues[2] = d2
			ps346.OverlayValues[3] = d3
			ps346.OverlayValues[4] = d4
			ps346.OverlayValues[5] = d5
			ps346.OverlayValues[6] = d6
			ps346.OverlayValues[8] = d8
			ps346.OverlayValues[9] = d9
			ps346.OverlayValues[10] = d10
			ps346.OverlayValues[11] = d11
			ps346.OverlayValues[12] = d12
			ps346.OverlayValues[15] = d15
			ps346.OverlayValues[32] = d32
			ps346.OverlayValues[33] = d33
			ps346.OverlayValues[34] = d34
			ps346.OverlayValues[35] = d35
			ps346.OverlayValues[36] = d36
			ps346.OverlayValues[38] = d38
			ps346.OverlayValues[40] = d40
			ps346.OverlayValues[41] = d41
			ps346.OverlayValues[44] = d44
			ps346.OverlayValues[69] = d69
			ps346.OverlayValues[70] = d70
			ps346.OverlayValues[71] = d71
			ps346.OverlayValues[72] = d72
			ps346.OverlayValues[73] = d73
			ps346.OverlayValues[74] = d74
			ps346.OverlayValues[75] = d75
			ps346.OverlayValues[110] = d110
			ps346.OverlayValues[111] = d111
			ps346.OverlayValues[112] = d112
			ps346.OverlayValues[150] = d150
			ps346.OverlayValues[151] = d151
			ps346.OverlayValues[152] = d152
			ps346.OverlayValues[153] = d153
			ps346.OverlayValues[154] = d154
			ps346.OverlayValues[157] = d157
			ps346.OverlayValues[158] = d158
			ps346.OverlayValues[201] = d201
			ps346.OverlayValues[202] = d202
			ps346.OverlayValues[203] = d203
			ps346.OverlayValues[204] = d204
			ps346.OverlayValues[206] = d206
			ps346.OverlayValues[207] = d207
			ps346.OverlayValues[208] = d208
			ps346.OverlayValues[209] = d209
			ps346.OverlayValues[210] = d210
			ps346.OverlayValues[212] = d212
			ps346.OverlayValues[213] = d213
			ps346.OverlayValues[214] = d214
			ps346.OverlayValues[215] = d215
			ps346.OverlayValues[273] = d273
			ps346.OverlayValues[274] = d274
			ps346.OverlayValues[275] = d275
			ps346.OverlayValues[276] = d276
			ps346.OverlayValues[338] = d338
			ps346.OverlayValues[339] = d339
			ps346.OverlayValues[340] = d340
			ps346.OverlayValues[342] = d342
			ps346.OverlayValues[343] = d343
			ps346.OverlayValues[344] = d344
			ps346.OverlayValues[345] = d345
			ps346.PhiValues = make([]JITValueDesc, 2)
			d347 = d2
			ps346.PhiValues[0] = d347
			d348 = d343
			ps346.PhiValues[1] = d348
			if ps346.General && bbs[17].Rendered {
				ctx.EmitJmp(lbl18)
				return result
			}
			return bbs[17].RenderPS(ps346)
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
					ctx.EmitJmp(lbl16)
					return result
				}
				bbs[15].Rendered = true
				bbs[15].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_15 = bbs[15].Address
				ctx.MarkLabel(lbl16)
				ctx.ResolveFixups()
			}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
				d273 = ps.OverlayValues[273]
			}
			if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
				d274 = ps.OverlayValues[274]
			}
			if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
				d275 = ps.OverlayValues[275]
			}
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
				d276 = ps.OverlayValues[276]
			}
			if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
				d338 = ps.OverlayValues[338]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
				d343 = ps.OverlayValues[343]
			}
			if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
				d344 = ps.OverlayValues[344]
			}
			if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
				d345 = ps.OverlayValues[345]
			}
			if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
				d347 = ps.OverlayValues[347]
			}
			if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
				d348 = ps.OverlayValues[348]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d4)
			var d349 JITValueDesc
			if d4.Loc == LocImm {
				idx := int(d4.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d349 = args[idx]
				d349.ID = 0
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
				lbl37 := ctx.ReserveLabel()
				lbl38 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d4.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl38)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d4.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r20, ai.Reg)
						ctx.EmitMovRegReg(r21, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r20, tmp.Reg)
						ctx.EmitMovRegReg(r21, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r20, Reg2: r21}
						ctx.BindReg(r20, &pair)
						ctx.BindReg(r21, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r20, uint64(ptrWord))
							ctx.EmitMovRegImm64(r21, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl37)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl38)
				d350 := JITValueDesc{Loc: LocRegPair, Reg: r20, Reg2: r21}
				ctx.BindReg(r20, &d350)
				ctx.BindReg(r21, &d350)
				ctx.BindReg(r20, &d350)
				ctx.BindReg(r21, &d350)
				ctx.EmitMakeNil(d350)
				ctx.MarkLabel(lbl37)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d349 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r20, Reg2: r21}
				ctx.BindReg(r20, &d349)
				ctx.BindReg(r21, &d349)
			}
			var d351 JITValueDesc
			if d349.Loc == LocImm {
				d351 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d349.Imm.Float())}
			} else if d349.Type == tagFloat && d349.Loc == LocReg {
				d351 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d349.Reg}
				ctx.BindReg(d349.Reg, &d351)
				ctx.BindReg(d349.Reg, &d351)
			} else if d349.Type == tagFloat && d349.Loc == LocRegPair {
				ctx.FreeReg(d349.Reg)
				d351 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d349.Reg2}
				ctx.BindReg(d349.Reg2, &d351)
				ctx.BindReg(d349.Reg2, &d351)
			} else {
				d351 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d349}, 1)
				d351.Type = tagFloat
				ctx.BindReg(d351.Reg, &d351)
			}
			ctx.FreeDesc(&d349)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d351)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d351)
			var d352 JITValueDesc
			if d5.Loc == LocImm && d351.Loc == LocImm {
				d352 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d5.Imm.Float() * d351.Imm.Float())}
			} else if d5.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d351.Reg)
				_, xBits := d5.Imm.RawWords()
				ctx.EmitMovRegImm64(scratch, xBits)
				ctx.EmitMulFloat64(scratch, d351.Reg)
				d352 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d352)
			} else if d351.Loc == LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(scratch, d5.Reg)
				_, yBits := d351.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, yBits)
				ctx.EmitMulFloat64(scratch, RegR11)
				d352 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d352)
			} else {
				r22 := ctx.AllocRegExcept(d5.Reg, d351.Reg)
				ctx.EmitMovRegReg(r22, d5.Reg)
				ctx.EmitMulFloat64(r22, d351.Reg)
				d352 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r22}
				ctx.BindReg(r22, &d352)
			}
			if d352.Loc == LocReg && d5.Loc == LocReg && d352.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = LocNone
			}
			ctx.FreeDesc(&d351)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d353 JITValueDesc
			if d4.Loc == LocImm {
				d353 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegReg(scratch, d4.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d353 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
				ctx.BindReg(scratch, &d353)
			}
			if d353.Loc == LocReg && d4.Loc == LocReg && d353.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = LocNone
			}
			ctx.EnsureDesc(&d352)
			if d352.Loc == LocReg {
				ctx.ProtectReg(d352.Reg)
			} else if d352.Loc == LocRegPair {
				ctx.ProtectReg(d352.Reg)
				ctx.ProtectReg(d352.Reg2)
			}
			ctx.EnsureDesc(&d353)
			if d353.Loc == LocReg {
				ctx.ProtectReg(d353.Reg)
			} else if d353.Loc == LocRegPair {
				ctx.ProtectReg(d353.Reg)
				ctx.ProtectReg(d353.Reg2)
			}
			d354 = d353
			if d354.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d354)
			ctx.EmitStoreToStack(d354, int32(bbs[17].PhiBase)+int32(0))
			d355 = d352
			if d355.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d355)
			ctx.EmitStoreToStack(d355, int32(bbs[17].PhiBase)+int32(16))
			if d352.Loc == LocReg {
				ctx.UnprotectReg(d352.Reg)
			} else if d352.Loc == LocRegPair {
				ctx.UnprotectReg(d352.Reg)
				ctx.UnprotectReg(d352.Reg2)
			}
			if d353.Loc == LocReg {
				ctx.UnprotectReg(d353.Reg)
			} else if d353.Loc == LocRegPair {
				ctx.UnprotectReg(d353.Reg)
				ctx.UnprotectReg(d353.Reg2)
			}
			ps356 := PhiState{General: ps.General}
			ps356.OverlayValues = make([]JITValueDesc, 356)
			ps356.OverlayValues[0] = d0
			ps356.OverlayValues[1] = d1
			ps356.OverlayValues[2] = d2
			ps356.OverlayValues[3] = d3
			ps356.OverlayValues[4] = d4
			ps356.OverlayValues[5] = d5
			ps356.OverlayValues[6] = d6
			ps356.OverlayValues[8] = d8
			ps356.OverlayValues[9] = d9
			ps356.OverlayValues[10] = d10
			ps356.OverlayValues[11] = d11
			ps356.OverlayValues[12] = d12
			ps356.OverlayValues[15] = d15
			ps356.OverlayValues[32] = d32
			ps356.OverlayValues[33] = d33
			ps356.OverlayValues[34] = d34
			ps356.OverlayValues[35] = d35
			ps356.OverlayValues[36] = d36
			ps356.OverlayValues[38] = d38
			ps356.OverlayValues[40] = d40
			ps356.OverlayValues[41] = d41
			ps356.OverlayValues[44] = d44
			ps356.OverlayValues[69] = d69
			ps356.OverlayValues[70] = d70
			ps356.OverlayValues[71] = d71
			ps356.OverlayValues[72] = d72
			ps356.OverlayValues[73] = d73
			ps356.OverlayValues[74] = d74
			ps356.OverlayValues[75] = d75
			ps356.OverlayValues[110] = d110
			ps356.OverlayValues[111] = d111
			ps356.OverlayValues[112] = d112
			ps356.OverlayValues[150] = d150
			ps356.OverlayValues[151] = d151
			ps356.OverlayValues[152] = d152
			ps356.OverlayValues[153] = d153
			ps356.OverlayValues[154] = d154
			ps356.OverlayValues[157] = d157
			ps356.OverlayValues[158] = d158
			ps356.OverlayValues[201] = d201
			ps356.OverlayValues[202] = d202
			ps356.OverlayValues[203] = d203
			ps356.OverlayValues[204] = d204
			ps356.OverlayValues[206] = d206
			ps356.OverlayValues[207] = d207
			ps356.OverlayValues[208] = d208
			ps356.OverlayValues[209] = d209
			ps356.OverlayValues[210] = d210
			ps356.OverlayValues[212] = d212
			ps356.OverlayValues[213] = d213
			ps356.OverlayValues[214] = d214
			ps356.OverlayValues[215] = d215
			ps356.OverlayValues[273] = d273
			ps356.OverlayValues[274] = d274
			ps356.OverlayValues[275] = d275
			ps356.OverlayValues[276] = d276
			ps356.OverlayValues[338] = d338
			ps356.OverlayValues[339] = d339
			ps356.OverlayValues[340] = d340
			ps356.OverlayValues[342] = d342
			ps356.OverlayValues[343] = d343
			ps356.OverlayValues[344] = d344
			ps356.OverlayValues[345] = d345
			ps356.OverlayValues[347] = d347
			ps356.OverlayValues[348] = d348
			ps356.OverlayValues[349] = d349
			ps356.OverlayValues[350] = d350
			ps356.OverlayValues[351] = d351
			ps356.OverlayValues[352] = d352
			ps356.OverlayValues[353] = d353
			ps356.OverlayValues[354] = d354
			ps356.OverlayValues[355] = d355
			ps356.PhiValues = make([]JITValueDesc, 2)
			d357 = d353
			ps356.PhiValues[0] = d357
			d358 = d352
			ps356.PhiValues[1] = d358
			if ps356.General && bbs[17].Rendered {
				ctx.EmitJmp(lbl18)
				return result
			}
			return bbs[17].RenderPS(ps356)
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
					ctx.EmitJmp(lbl17)
					return result
				}
				bbs[16].Rendered = true
				bbs[16].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_16 = bbs[16].Address
				ctx.MarkLabel(lbl17)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(32)}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
				d273 = ps.OverlayValues[273]
			}
			if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
				d274 = ps.OverlayValues[274]
			}
			if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
				d275 = ps.OverlayValues[275]
			}
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
				d276 = ps.OverlayValues[276]
			}
			if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
				d338 = ps.OverlayValues[338]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
				d343 = ps.OverlayValues[343]
			}
			if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
				d344 = ps.OverlayValues[344]
			}
			if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
				d345 = ps.OverlayValues[345]
			}
			if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
				d347 = ps.OverlayValues[347]
			}
			if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
				d348 = ps.OverlayValues[348]
			}
			if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
				d349 = ps.OverlayValues[349]
			}
			if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
				d350 = ps.OverlayValues[350]
			}
			if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
				d351 = ps.OverlayValues[351]
			}
			if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
				d352 = ps.OverlayValues[352]
			}
			if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
				d353 = ps.OverlayValues[353]
			}
			if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
				d354 = ps.OverlayValues[354]
			}
			if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
				d355 = ps.OverlayValues[355]
			}
			if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
				d357 = ps.OverlayValues[357]
			}
			if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
				d358 = ps.OverlayValues[358]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			ctx.EmitMakeFloat(result, d5)
			if d5.Loc == LocReg { ctx.FreeReg(d5.Reg) }
			result.Type = tagFloat
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[17].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d359 := ps.PhiValues[0]
					ctx.EnsureDesc(&d359)
					ctx.EmitStoreToStack(d359, int32(bbs[17].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d360 := ps.PhiValues[1]
					ctx.EnsureDesc(&d360)
					ctx.EmitStoreToStack(d360, int32(bbs[17].PhiBase)+int32(16))
				}
				if bbs[17].VisitCount >= 2 {
					ps.General = true
					return bbs[17].RenderPS(ps)
				}
			}
			bbs[17].VisitCount++
			if ps.General {
				if bbs[17].Rendered {
					ctx.EmitJmp(lbl18)
					return result
				}
				bbs[17].Rendered = true
				bbs[17].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_17 = bbs[17].Address
				ctx.MarkLabel(lbl18)
				ctx.ResolveFixups()
			}
			d3 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(48)}
			d4 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(64)}
			d5 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(80)}
			d0 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != LocNone {
				d15 = ps.OverlayValues[15]
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
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
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
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 153 && ps.OverlayValues[153].Loc != LocNone {
				d153 = ps.OverlayValues[153]
			}
			if len(ps.OverlayValues) > 154 && ps.OverlayValues[154].Loc != LocNone {
				d154 = ps.OverlayValues[154]
			}
			if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != LocNone {
				d157 = ps.OverlayValues[157]
			}
			if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != LocNone {
				d158 = ps.OverlayValues[158]
			}
			if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
				d201 = ps.OverlayValues[201]
			}
			if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
				d202 = ps.OverlayValues[202]
			}
			if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
				d203 = ps.OverlayValues[203]
			}
			if len(ps.OverlayValues) > 204 && ps.OverlayValues[204].Loc != LocNone {
				d204 = ps.OverlayValues[204]
			}
			if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
				d206 = ps.OverlayValues[206]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != LocNone {
				d214 = ps.OverlayValues[214]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 273 && ps.OverlayValues[273].Loc != LocNone {
				d273 = ps.OverlayValues[273]
			}
			if len(ps.OverlayValues) > 274 && ps.OverlayValues[274].Loc != LocNone {
				d274 = ps.OverlayValues[274]
			}
			if len(ps.OverlayValues) > 275 && ps.OverlayValues[275].Loc != LocNone {
				d275 = ps.OverlayValues[275]
			}
			if len(ps.OverlayValues) > 276 && ps.OverlayValues[276].Loc != LocNone {
				d276 = ps.OverlayValues[276]
			}
			if len(ps.OverlayValues) > 338 && ps.OverlayValues[338].Loc != LocNone {
				d338 = ps.OverlayValues[338]
			}
			if len(ps.OverlayValues) > 339 && ps.OverlayValues[339].Loc != LocNone {
				d339 = ps.OverlayValues[339]
			}
			if len(ps.OverlayValues) > 340 && ps.OverlayValues[340].Loc != LocNone {
				d340 = ps.OverlayValues[340]
			}
			if len(ps.OverlayValues) > 342 && ps.OverlayValues[342].Loc != LocNone {
				d342 = ps.OverlayValues[342]
			}
			if len(ps.OverlayValues) > 343 && ps.OverlayValues[343].Loc != LocNone {
				d343 = ps.OverlayValues[343]
			}
			if len(ps.OverlayValues) > 344 && ps.OverlayValues[344].Loc != LocNone {
				d344 = ps.OverlayValues[344]
			}
			if len(ps.OverlayValues) > 345 && ps.OverlayValues[345].Loc != LocNone {
				d345 = ps.OverlayValues[345]
			}
			if len(ps.OverlayValues) > 347 && ps.OverlayValues[347].Loc != LocNone {
				d347 = ps.OverlayValues[347]
			}
			if len(ps.OverlayValues) > 348 && ps.OverlayValues[348].Loc != LocNone {
				d348 = ps.OverlayValues[348]
			}
			if len(ps.OverlayValues) > 349 && ps.OverlayValues[349].Loc != LocNone {
				d349 = ps.OverlayValues[349]
			}
			if len(ps.OverlayValues) > 350 && ps.OverlayValues[350].Loc != LocNone {
				d350 = ps.OverlayValues[350]
			}
			if len(ps.OverlayValues) > 351 && ps.OverlayValues[351].Loc != LocNone {
				d351 = ps.OverlayValues[351]
			}
			if len(ps.OverlayValues) > 352 && ps.OverlayValues[352].Loc != LocNone {
				d352 = ps.OverlayValues[352]
			}
			if len(ps.OverlayValues) > 353 && ps.OverlayValues[353].Loc != LocNone {
				d353 = ps.OverlayValues[353]
			}
			if len(ps.OverlayValues) > 354 && ps.OverlayValues[354].Loc != LocNone {
				d354 = ps.OverlayValues[354]
			}
			if len(ps.OverlayValues) > 355 && ps.OverlayValues[355].Loc != LocNone {
				d355 = ps.OverlayValues[355]
			}
			if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != LocNone {
				d357 = ps.OverlayValues[357]
			}
			if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != LocNone {
				d358 = ps.OverlayValues[358]
			}
			if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != LocNone {
				d359 = ps.OverlayValues[359]
			}
			if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != LocNone {
				d360 = ps.OverlayValues[360]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d4 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
				d5 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d361 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d361)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d361)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d361)
			var d362 JITValueDesc
			if d4.Loc == LocImm && d361.Loc == LocImm {
				d362 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d4.Imm.Int() < d361.Imm.Int())}
			} else if d361.Loc == LocImm {
				r23 := ctx.AllocRegExcept(d4.Reg)
				if d361.Imm.Int() >= -2147483648 && d361.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d4.Reg, int32(d361.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d361.Imm.Int()))
					ctx.EmitCmpInt64(d4.Reg, RegR11)
				}
				ctx.EmitSetcc(r23, CcL)
				d362 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
				ctx.BindReg(r23, &d362)
			} else if d4.Loc == LocImm {
				r24 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d4.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d361.Reg)
				ctx.EmitSetcc(r24, CcL)
				d362 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
				ctx.BindReg(r24, &d362)
			} else {
				r25 := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitCmpInt64(d4.Reg, d361.Reg)
				ctx.EmitSetcc(r25, CcL)
				d362 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
				ctx.BindReg(r25, &d362)
			}
			ctx.FreeDesc(&d361)
			d363 = d362
			ctx.EnsureDesc(&d363)
			if d363.Loc != LocImm && d363.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d363.Loc == LocImm {
				if d363.Imm.Bool() {
			ps364 := PhiState{General: ps.General}
			ps364.OverlayValues = make([]JITValueDesc, 364)
			ps364.OverlayValues[0] = d0
			ps364.OverlayValues[1] = d1
			ps364.OverlayValues[2] = d2
			ps364.OverlayValues[3] = d3
			ps364.OverlayValues[4] = d4
			ps364.OverlayValues[5] = d5
			ps364.OverlayValues[6] = d6
			ps364.OverlayValues[8] = d8
			ps364.OverlayValues[9] = d9
			ps364.OverlayValues[10] = d10
			ps364.OverlayValues[11] = d11
			ps364.OverlayValues[12] = d12
			ps364.OverlayValues[15] = d15
			ps364.OverlayValues[32] = d32
			ps364.OverlayValues[33] = d33
			ps364.OverlayValues[34] = d34
			ps364.OverlayValues[35] = d35
			ps364.OverlayValues[36] = d36
			ps364.OverlayValues[38] = d38
			ps364.OverlayValues[40] = d40
			ps364.OverlayValues[41] = d41
			ps364.OverlayValues[44] = d44
			ps364.OverlayValues[69] = d69
			ps364.OverlayValues[70] = d70
			ps364.OverlayValues[71] = d71
			ps364.OverlayValues[72] = d72
			ps364.OverlayValues[73] = d73
			ps364.OverlayValues[74] = d74
			ps364.OverlayValues[75] = d75
			ps364.OverlayValues[110] = d110
			ps364.OverlayValues[111] = d111
			ps364.OverlayValues[112] = d112
			ps364.OverlayValues[150] = d150
			ps364.OverlayValues[151] = d151
			ps364.OverlayValues[152] = d152
			ps364.OverlayValues[153] = d153
			ps364.OverlayValues[154] = d154
			ps364.OverlayValues[157] = d157
			ps364.OverlayValues[158] = d158
			ps364.OverlayValues[201] = d201
			ps364.OverlayValues[202] = d202
			ps364.OverlayValues[203] = d203
			ps364.OverlayValues[204] = d204
			ps364.OverlayValues[206] = d206
			ps364.OverlayValues[207] = d207
			ps364.OverlayValues[208] = d208
			ps364.OverlayValues[209] = d209
			ps364.OverlayValues[210] = d210
			ps364.OverlayValues[212] = d212
			ps364.OverlayValues[213] = d213
			ps364.OverlayValues[214] = d214
			ps364.OverlayValues[215] = d215
			ps364.OverlayValues[273] = d273
			ps364.OverlayValues[274] = d274
			ps364.OverlayValues[275] = d275
			ps364.OverlayValues[276] = d276
			ps364.OverlayValues[338] = d338
			ps364.OverlayValues[339] = d339
			ps364.OverlayValues[340] = d340
			ps364.OverlayValues[342] = d342
			ps364.OverlayValues[343] = d343
			ps364.OverlayValues[344] = d344
			ps364.OverlayValues[345] = d345
			ps364.OverlayValues[347] = d347
			ps364.OverlayValues[348] = d348
			ps364.OverlayValues[349] = d349
			ps364.OverlayValues[350] = d350
			ps364.OverlayValues[351] = d351
			ps364.OverlayValues[352] = d352
			ps364.OverlayValues[353] = d353
			ps364.OverlayValues[354] = d354
			ps364.OverlayValues[355] = d355
			ps364.OverlayValues[357] = d357
			ps364.OverlayValues[358] = d358
			ps364.OverlayValues[359] = d359
			ps364.OverlayValues[360] = d360
			ps364.OverlayValues[361] = d361
			ps364.OverlayValues[362] = d362
			ps364.OverlayValues[363] = d363
					return bbs[15].RenderPS(ps364)
				}
			ps365 := PhiState{General: ps.General}
			ps365.OverlayValues = make([]JITValueDesc, 364)
			ps365.OverlayValues[0] = d0
			ps365.OverlayValues[1] = d1
			ps365.OverlayValues[2] = d2
			ps365.OverlayValues[3] = d3
			ps365.OverlayValues[4] = d4
			ps365.OverlayValues[5] = d5
			ps365.OverlayValues[6] = d6
			ps365.OverlayValues[8] = d8
			ps365.OverlayValues[9] = d9
			ps365.OverlayValues[10] = d10
			ps365.OverlayValues[11] = d11
			ps365.OverlayValues[12] = d12
			ps365.OverlayValues[15] = d15
			ps365.OverlayValues[32] = d32
			ps365.OverlayValues[33] = d33
			ps365.OverlayValues[34] = d34
			ps365.OverlayValues[35] = d35
			ps365.OverlayValues[36] = d36
			ps365.OverlayValues[38] = d38
			ps365.OverlayValues[40] = d40
			ps365.OverlayValues[41] = d41
			ps365.OverlayValues[44] = d44
			ps365.OverlayValues[69] = d69
			ps365.OverlayValues[70] = d70
			ps365.OverlayValues[71] = d71
			ps365.OverlayValues[72] = d72
			ps365.OverlayValues[73] = d73
			ps365.OverlayValues[74] = d74
			ps365.OverlayValues[75] = d75
			ps365.OverlayValues[110] = d110
			ps365.OverlayValues[111] = d111
			ps365.OverlayValues[112] = d112
			ps365.OverlayValues[150] = d150
			ps365.OverlayValues[151] = d151
			ps365.OverlayValues[152] = d152
			ps365.OverlayValues[153] = d153
			ps365.OverlayValues[154] = d154
			ps365.OverlayValues[157] = d157
			ps365.OverlayValues[158] = d158
			ps365.OverlayValues[201] = d201
			ps365.OverlayValues[202] = d202
			ps365.OverlayValues[203] = d203
			ps365.OverlayValues[204] = d204
			ps365.OverlayValues[206] = d206
			ps365.OverlayValues[207] = d207
			ps365.OverlayValues[208] = d208
			ps365.OverlayValues[209] = d209
			ps365.OverlayValues[210] = d210
			ps365.OverlayValues[212] = d212
			ps365.OverlayValues[213] = d213
			ps365.OverlayValues[214] = d214
			ps365.OverlayValues[215] = d215
			ps365.OverlayValues[273] = d273
			ps365.OverlayValues[274] = d274
			ps365.OverlayValues[275] = d275
			ps365.OverlayValues[276] = d276
			ps365.OverlayValues[338] = d338
			ps365.OverlayValues[339] = d339
			ps365.OverlayValues[340] = d340
			ps365.OverlayValues[342] = d342
			ps365.OverlayValues[343] = d343
			ps365.OverlayValues[344] = d344
			ps365.OverlayValues[345] = d345
			ps365.OverlayValues[347] = d347
			ps365.OverlayValues[348] = d348
			ps365.OverlayValues[349] = d349
			ps365.OverlayValues[350] = d350
			ps365.OverlayValues[351] = d351
			ps365.OverlayValues[352] = d352
			ps365.OverlayValues[353] = d353
			ps365.OverlayValues[354] = d354
			ps365.OverlayValues[355] = d355
			ps365.OverlayValues[357] = d357
			ps365.OverlayValues[358] = d358
			ps365.OverlayValues[359] = d359
			ps365.OverlayValues[360] = d360
			ps365.OverlayValues[361] = d361
			ps365.OverlayValues[362] = d362
			ps365.OverlayValues[363] = d363
				return bbs[16].RenderPS(ps365)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d366 := ps.PhiValues[0]
					ctx.EnsureDesc(&d366)
					ctx.EmitStoreToStack(d366, int32(bbs[17].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d367 := ps.PhiValues[1]
					ctx.EnsureDesc(&d367)
					ctx.EmitStoreToStack(d367, int32(bbs[17].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[17].RenderPS(ps)
			}
			lbl39 := ctx.ReserveLabel()
			lbl40 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d363.Reg, 0)
			ctx.EmitJcc(CcNE, lbl39)
			ctx.EmitJmp(lbl40)
			ctx.MarkLabel(lbl39)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl40)
			ctx.EmitJmp(lbl17)
			ps368 := PhiState{General: true}
			ps368.OverlayValues = make([]JITValueDesc, 368)
			ps368.OverlayValues[0] = d0
			ps368.OverlayValues[1] = d1
			ps368.OverlayValues[2] = d2
			ps368.OverlayValues[3] = d3
			ps368.OverlayValues[4] = d4
			ps368.OverlayValues[5] = d5
			ps368.OverlayValues[6] = d6
			ps368.OverlayValues[8] = d8
			ps368.OverlayValues[9] = d9
			ps368.OverlayValues[10] = d10
			ps368.OverlayValues[11] = d11
			ps368.OverlayValues[12] = d12
			ps368.OverlayValues[15] = d15
			ps368.OverlayValues[32] = d32
			ps368.OverlayValues[33] = d33
			ps368.OverlayValues[34] = d34
			ps368.OverlayValues[35] = d35
			ps368.OverlayValues[36] = d36
			ps368.OverlayValues[38] = d38
			ps368.OverlayValues[40] = d40
			ps368.OverlayValues[41] = d41
			ps368.OverlayValues[44] = d44
			ps368.OverlayValues[69] = d69
			ps368.OverlayValues[70] = d70
			ps368.OverlayValues[71] = d71
			ps368.OverlayValues[72] = d72
			ps368.OverlayValues[73] = d73
			ps368.OverlayValues[74] = d74
			ps368.OverlayValues[75] = d75
			ps368.OverlayValues[110] = d110
			ps368.OverlayValues[111] = d111
			ps368.OverlayValues[112] = d112
			ps368.OverlayValues[150] = d150
			ps368.OverlayValues[151] = d151
			ps368.OverlayValues[152] = d152
			ps368.OverlayValues[153] = d153
			ps368.OverlayValues[154] = d154
			ps368.OverlayValues[157] = d157
			ps368.OverlayValues[158] = d158
			ps368.OverlayValues[201] = d201
			ps368.OverlayValues[202] = d202
			ps368.OverlayValues[203] = d203
			ps368.OverlayValues[204] = d204
			ps368.OverlayValues[206] = d206
			ps368.OverlayValues[207] = d207
			ps368.OverlayValues[208] = d208
			ps368.OverlayValues[209] = d209
			ps368.OverlayValues[210] = d210
			ps368.OverlayValues[212] = d212
			ps368.OverlayValues[213] = d213
			ps368.OverlayValues[214] = d214
			ps368.OverlayValues[215] = d215
			ps368.OverlayValues[273] = d273
			ps368.OverlayValues[274] = d274
			ps368.OverlayValues[275] = d275
			ps368.OverlayValues[276] = d276
			ps368.OverlayValues[338] = d338
			ps368.OverlayValues[339] = d339
			ps368.OverlayValues[340] = d340
			ps368.OverlayValues[342] = d342
			ps368.OverlayValues[343] = d343
			ps368.OverlayValues[344] = d344
			ps368.OverlayValues[345] = d345
			ps368.OverlayValues[347] = d347
			ps368.OverlayValues[348] = d348
			ps368.OverlayValues[349] = d349
			ps368.OverlayValues[350] = d350
			ps368.OverlayValues[351] = d351
			ps368.OverlayValues[352] = d352
			ps368.OverlayValues[353] = d353
			ps368.OverlayValues[354] = d354
			ps368.OverlayValues[355] = d355
			ps368.OverlayValues[357] = d357
			ps368.OverlayValues[358] = d358
			ps368.OverlayValues[359] = d359
			ps368.OverlayValues[360] = d360
			ps368.OverlayValues[361] = d361
			ps368.OverlayValues[362] = d362
			ps368.OverlayValues[363] = d363
			ps368.OverlayValues[366] = d366
			ps368.OverlayValues[367] = d367
			ps369 := PhiState{General: true}
			ps369.OverlayValues = make([]JITValueDesc, 368)
			ps369.OverlayValues[0] = d0
			ps369.OverlayValues[1] = d1
			ps369.OverlayValues[2] = d2
			ps369.OverlayValues[3] = d3
			ps369.OverlayValues[4] = d4
			ps369.OverlayValues[5] = d5
			ps369.OverlayValues[6] = d6
			ps369.OverlayValues[8] = d8
			ps369.OverlayValues[9] = d9
			ps369.OverlayValues[10] = d10
			ps369.OverlayValues[11] = d11
			ps369.OverlayValues[12] = d12
			ps369.OverlayValues[15] = d15
			ps369.OverlayValues[32] = d32
			ps369.OverlayValues[33] = d33
			ps369.OverlayValues[34] = d34
			ps369.OverlayValues[35] = d35
			ps369.OverlayValues[36] = d36
			ps369.OverlayValues[38] = d38
			ps369.OverlayValues[40] = d40
			ps369.OverlayValues[41] = d41
			ps369.OverlayValues[44] = d44
			ps369.OverlayValues[69] = d69
			ps369.OverlayValues[70] = d70
			ps369.OverlayValues[71] = d71
			ps369.OverlayValues[72] = d72
			ps369.OverlayValues[73] = d73
			ps369.OverlayValues[74] = d74
			ps369.OverlayValues[75] = d75
			ps369.OverlayValues[110] = d110
			ps369.OverlayValues[111] = d111
			ps369.OverlayValues[112] = d112
			ps369.OverlayValues[150] = d150
			ps369.OverlayValues[151] = d151
			ps369.OverlayValues[152] = d152
			ps369.OverlayValues[153] = d153
			ps369.OverlayValues[154] = d154
			ps369.OverlayValues[157] = d157
			ps369.OverlayValues[158] = d158
			ps369.OverlayValues[201] = d201
			ps369.OverlayValues[202] = d202
			ps369.OverlayValues[203] = d203
			ps369.OverlayValues[204] = d204
			ps369.OverlayValues[206] = d206
			ps369.OverlayValues[207] = d207
			ps369.OverlayValues[208] = d208
			ps369.OverlayValues[209] = d209
			ps369.OverlayValues[210] = d210
			ps369.OverlayValues[212] = d212
			ps369.OverlayValues[213] = d213
			ps369.OverlayValues[214] = d214
			ps369.OverlayValues[215] = d215
			ps369.OverlayValues[273] = d273
			ps369.OverlayValues[274] = d274
			ps369.OverlayValues[275] = d275
			ps369.OverlayValues[276] = d276
			ps369.OverlayValues[338] = d338
			ps369.OverlayValues[339] = d339
			ps369.OverlayValues[340] = d340
			ps369.OverlayValues[342] = d342
			ps369.OverlayValues[343] = d343
			ps369.OverlayValues[344] = d344
			ps369.OverlayValues[345] = d345
			ps369.OverlayValues[347] = d347
			ps369.OverlayValues[348] = d348
			ps369.OverlayValues[349] = d349
			ps369.OverlayValues[350] = d350
			ps369.OverlayValues[351] = d351
			ps369.OverlayValues[352] = d352
			ps369.OverlayValues[353] = d353
			ps369.OverlayValues[354] = d354
			ps369.OverlayValues[355] = d355
			ps369.OverlayValues[357] = d357
			ps369.OverlayValues[358] = d358
			ps369.OverlayValues[359] = d359
			ps369.OverlayValues[360] = d360
			ps369.OverlayValues[361] = d361
			ps369.OverlayValues[362] = d362
			ps369.OverlayValues[363] = d363
			ps369.OverlayValues[366] = d366
			ps369.OverlayValues[367] = d367
			snap370 := d0
			snap371 := d1
			snap372 := d2
			snap373 := d3
			snap374 := d4
			snap375 := d5
			snap376 := d6
			snap377 := d8
			snap378 := d9
			snap379 := d10
			snap380 := d11
			snap381 := d12
			snap382 := d15
			snap383 := d32
			snap384 := d33
			snap385 := d34
			snap386 := d35
			snap387 := d36
			snap388 := d38
			snap389 := d40
			snap390 := d41
			snap391 := d44
			snap392 := d69
			snap393 := d70
			snap394 := d71
			snap395 := d72
			snap396 := d73
			snap397 := d74
			snap398 := d75
			snap399 := d110
			snap400 := d111
			snap401 := d112
			snap402 := d150
			snap403 := d151
			snap404 := d152
			snap405 := d153
			snap406 := d154
			snap407 := d157
			snap408 := d158
			snap409 := d201
			snap410 := d202
			snap411 := d203
			snap412 := d204
			snap413 := d206
			snap414 := d207
			snap415 := d208
			snap416 := d209
			snap417 := d210
			snap418 := d212
			snap419 := d213
			snap420 := d214
			snap421 := d215
			snap422 := d273
			snap423 := d274
			snap424 := d275
			snap425 := d276
			snap426 := d338
			snap427 := d339
			snap428 := d340
			snap429 := d342
			snap430 := d343
			snap431 := d344
			snap432 := d345
			snap433 := d347
			snap434 := d348
			snap435 := d349
			snap436 := d350
			snap437 := d351
			snap438 := d352
			snap439 := d353
			snap440 := d354
			snap441 := d355
			snap442 := d357
			snap443 := d358
			snap444 := d359
			snap445 := d360
			snap446 := d361
			snap447 := d362
			snap448 := d363
			snap449 := d366
			snap450 := d367
			alloc451 := ctx.SnapshotAllocState()
			if !bbs[16].Rendered {
				bbs[16].RenderPS(ps369)
			}
			ctx.RestoreAllocState(alloc451)
			d0 = snap370
			d1 = snap371
			d2 = snap372
			d3 = snap373
			d4 = snap374
			d5 = snap375
			d6 = snap376
			d8 = snap377
			d9 = snap378
			d10 = snap379
			d11 = snap380
			d12 = snap381
			d15 = snap382
			d32 = snap383
			d33 = snap384
			d34 = snap385
			d35 = snap386
			d36 = snap387
			d38 = snap388
			d40 = snap389
			d41 = snap390
			d44 = snap391
			d69 = snap392
			d70 = snap393
			d71 = snap394
			d72 = snap395
			d73 = snap396
			d74 = snap397
			d75 = snap398
			d110 = snap399
			d111 = snap400
			d112 = snap401
			d150 = snap402
			d151 = snap403
			d152 = snap404
			d153 = snap405
			d154 = snap406
			d157 = snap407
			d158 = snap408
			d201 = snap409
			d202 = snap410
			d203 = snap411
			d204 = snap412
			d206 = snap413
			d207 = snap414
			d208 = snap415
			d209 = snap416
			d210 = snap417
			d212 = snap418
			d213 = snap419
			d214 = snap420
			d215 = snap421
			d273 = snap422
			d274 = snap423
			d275 = snap424
			d276 = snap425
			d338 = snap426
			d339 = snap427
			d340 = snap428
			d342 = snap429
			d343 = snap430
			d344 = snap431
			d345 = snap432
			d347 = snap433
			d348 = snap434
			d349 = snap435
			d350 = snap436
			d351 = snap437
			d352 = snap438
			d353 = snap439
			d354 = snap440
			d355 = snap441
			d357 = snap442
			d358 = snap443
			d359 = snap444
			d360 = snap445
			d361 = snap446
			d362 = snap447
			d363 = snap448
			d366 = snap449
			d367 = snap450
			if !bbs[15].Rendered {
				return bbs[15].RenderPS(ps368)
			}
			return result
			ctx.FreeDesc(&d362)
			return result
			}
			argPinned452 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned452 = append(argPinned452, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned452 = append(argPinned452, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned452 = append(argPinned452, ai.Reg2)
					}
				}
			}
			ps453 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps453)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(96))
			ctx.EmitAddRSP32(int32(96))
			for _, r := range argPinned452 {
				ctx.UnprotectReg(r)
			}
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
		nil /* TODO: Slice on non-desc: slice a[1:int:] */, /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */ /* TODO: Slice on non-desc: slice a[1:int:] */
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
			d0 = args[1]
			d0.ID = 0
			d1 = args[0]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
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
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
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
					ctx.EmitMovRegReg(negReg, d2.Reg2)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.EmitMovRegReg(negReg, d2.Reg)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.EmitMakeBool(result, d3)
			} else {
				ctx.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			argPinned4 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned4 = append(argPinned4, ai.Reg2)
					}
				}
			}
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
			for _, r := range argPinned4 {
				ctx.UnprotectReg(r)
			}
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
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
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
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
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
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeBool(result, d2)
			} else {
				ctx.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
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
			d0 = args[1]
			d0.ID = 0
			d1 = args[0]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
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
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
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
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeBool(result, d2)
			} else {
				ctx.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
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
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
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
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
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
					ctx.EmitMovRegReg(negReg, d2.Reg2)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d2.Loc == LocReg {
					ctx.EmitMovRegReg(negReg, d2.Reg)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d3)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.EmitMakeBool(result, d3)
			} else {
				ctx.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			argPinned4 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned4 = append(argPinned4, ai.Reg2)
					}
				}
			}
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
			for _, r := range argPinned4 {
				ctx.UnprotectReg(r)
			}
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
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
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
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
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
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeBool(result, d2)
			} else {
				ctx.EmitMakeBool(result, d2)
				ctx.FreeReg(d2.Reg)
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
			d1 = args[1]
			d1.ID = 0
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
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
					ctx.EmitMakeBool(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d1)
				} else if d1.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d1.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d1 = tmpPair
			} else if d1.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d1.Type, Reg: ctx.AllocRegExcept(d1.Reg), Reg2: ctx.AllocRegExcept(d1.Reg)}
				switch d1.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d1)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d1)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d1)
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
			ctx.ResolveFixups()
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
					ctx.EmitMakeBool(result, d2)
					result.Type = tagBool
				case tagInt:
					ctx.EmitMakeInt(result, d2)
					result.Type = tagInt
				case tagFloat:
					ctx.EmitMakeFloat(result, d2)
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
		nil /* TODO: unsupported compare const kind: (Scmer).String(t22) */, /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: unsupported compare const kind: (Scmer).String(t22) */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */ /* TODO: Index: s[t1] */
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
		nil /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */, /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */ /* TODO: FieldAddr on non-receiver: &Globalenv.Vars [#0] */
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
			d1 = ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.EmitMovRegReg(negReg, d1.Reg2)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d1.Loc == LocReg {
					ctx.EmitMovRegReg(negReg, d1.Reg)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.EmitMakeBool(result, d3)
			} else {
				ctx.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			argPinned4 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned4 = append(argPinned4, ai.Reg2)
					}
				}
			}
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
			for _, r := range argPinned4 {
				ctx.UnprotectReg(r)
			}
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
			d1 = ctx.EmitBoolDesc(&d2, JITValueDesc{Loc: LocAny})
			ctx.FreeDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d3 JITValueDesc
			if d1.Loc == LocImm {
				d3 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(!d1.Imm.Bool())}
			} else {
				negReg := ctx.AllocReg()
				if d1.Loc == LocRegPair {
					ctx.EmitMovRegReg(negReg, d1.Reg2)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else if d1.Loc == LocReg {
					ctx.EmitMovRegReg(negReg, d1.Reg)
					ctx.EmitAndRegImm32(negReg, 1)
					ctx.EmitCmpRegImm32(negReg, 0)
					ctx.EmitSetcc(negReg, CcE)
					d3 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: negReg}
					ctx.BindReg(negReg, &d3)
				} else {
					panic("UnOp ! unsupported source location")
				}
			}
			ctx.FreeDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d3.Loc == LocImm {
				ctx.EmitMakeBool(result, d3)
			} else {
				ctx.EmitMakeBool(result, d3)
				ctx.FreeReg(d3.Reg)
			}
			result.Type = tagBool
			return result
			return result
			}
			argPinned4 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned4 = append(argPinned4, ai.Reg2)
					}
				}
			}
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
			for _, r := range argPinned4 {
				ctx.UnprotectReg(r)
			}
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
			d1 = ctx.EmitTagEqualsBorrowed(&d2, tagNil, JITValueDesc{Loc: LocAny})
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
			var d13 JITValueDesc
			_ = d13
			var d14 JITValueDesc
			_ = d14
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
			var d57 JITValueDesc
			_ = d57
			var d58 JITValueDesc
			_ = d58
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
			var d68 JITValueDesc
			_ = d68
			var d69 JITValueDesc
			_ = d69
			var d71 JITValueDesc
			_ = d71
			var d72 JITValueDesc
			_ = d72
			var d75 JITValueDesc
			_ = d75
			var d76 JITValueDesc
			_ = d76
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
			var d121 JITValueDesc
			_ = d121
			var d122 JITValueDesc
			_ = d122
			var d123 JITValueDesc
			_ = d123
			var d124 JITValueDesc
			_ = d124
			var d127 JITValueDesc
			_ = d127
			var d128 JITValueDesc
			_ = d128
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(32)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(32)
			}
			d0 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			var bbs [8]BBDescriptor
			bbs[1].PhiBase = int32(0)
			bbs[1].PhiCount = uint16(2)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(16))
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
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps3)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d6 := ps.PhiValues[0]
					ctx.EnsureDesc(&d6)
					ctx.EmitStoreScmerToStack(d6, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d7 := ps.PhiValues[1]
					ctx.EnsureDesc(&d7)
					ctx.EmitStoreToStack(d7, int32(bbs[1].PhiBase)+int32(16))
				}
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
				ctx.EmitMovRegReg(scratch, d1.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
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
					ctx.EmitCmpRegImm32(d8.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
					ctx.EmitCmpInt64(d8.Reg, RegR11)
				}
				ctx.EmitSetcc(r1, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d9)
			} else if d8.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d2.Reg)
				ctx.EmitSetcc(r2, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d9)
			} else {
				r3 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitCmpInt64(d8.Reg, d2.Reg)
				ctx.EmitSetcc(r3, CcL)
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
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d13 := ps.PhiValues[0]
					ctx.EnsureDesc(&d13)
					ctx.EmitStoreScmerToStack(d13, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d14 := ps.PhiValues[1]
					ctx.EnsureDesc(&d14)
					ctx.EmitStoreToStack(d14, int32(bbs[1].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d10.Reg, 0)
			ctx.EmitJcc(CcNE, lbl9)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl4)
			ps15 := PhiState{General: true}
			ps15.OverlayValues = make([]JITValueDesc, 15)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[7] = d7
			ps15.OverlayValues[8] = d8
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[13] = d13
			ps15.OverlayValues[14] = d14
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 15)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[4] = d4
			ps16.OverlayValues[5] = d5
			ps16.OverlayValues[6] = d6
			ps16.OverlayValues[7] = d7
			ps16.OverlayValues[8] = d8
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[13] = d13
			ps16.OverlayValues[14] = d14
			snap17 := d0
			snap18 := d1
			snap19 := d2
			snap20 := d4
			snap21 := d5
			snap22 := d6
			snap23 := d7
			snap24 := d8
			snap25 := d9
			snap26 := d10
			snap27 := d13
			snap28 := d14
			alloc29 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc29)
			d0 = snap17
			d1 = snap18
			d2 = snap19
			d4 = snap20
			d5 = snap21
			d6 = snap22
			d7 = snap23
			d8 = snap24
			d9 = snap25
			d10 = snap26
			d13 = snap27
			d14 = snap28
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps15)
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
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			var d30 JITValueDesc
			if d8.Loc == LocImm {
				idx := int(d8.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d30 = args[idx]
				d30.ID = 0
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
				lbl11 := ctx.ReserveLabel()
				lbl12 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d8.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl12)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d8.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r4, ai.Reg)
						ctx.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r4, tmp.Reg)
						ctx.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl12)
				d31 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d31)
				ctx.BindReg(r5, &d31)
				ctx.BindReg(r4, &d31)
				ctx.BindReg(r5, &d31)
				ctx.EmitMakeNil(d31)
				ctx.MarkLabel(lbl11)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d30 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d30)
				ctx.BindReg(r5, &d30)
			}
			d33 = d0
			d33.ID = 0
			d32 = ctx.EmitTagEqualsBorrowed(&d33, tagNil, JITValueDesc{Loc: LocAny})
			d34 = d32
			ctx.EnsureDesc(&d34)
			if d34.Loc != LocImm && d34.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d34.Loc == LocImm {
				if d34.Imm.Bool() {
			ps35 := PhiState{General: ps.General}
			ps35.OverlayValues = make([]JITValueDesc, 35)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[1] = d1
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[4] = d4
			ps35.OverlayValues[5] = d5
			ps35.OverlayValues[6] = d6
			ps35.OverlayValues[7] = d7
			ps35.OverlayValues[8] = d8
			ps35.OverlayValues[9] = d9
			ps35.OverlayValues[10] = d10
			ps35.OverlayValues[13] = d13
			ps35.OverlayValues[14] = d14
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[32] = d32
			ps35.OverlayValues[33] = d33
			ps35.OverlayValues[34] = d34
					return bbs[4].RenderPS(ps35)
				}
			ps36 := PhiState{General: ps.General}
			ps36.OverlayValues = make([]JITValueDesc, 35)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[1] = d1
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[4] = d4
			ps36.OverlayValues[5] = d5
			ps36.OverlayValues[6] = d6
			ps36.OverlayValues[7] = d7
			ps36.OverlayValues[8] = d8
			ps36.OverlayValues[9] = d9
			ps36.OverlayValues[10] = d10
			ps36.OverlayValues[13] = d13
			ps36.OverlayValues[14] = d14
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[32] = d32
			ps36.OverlayValues[33] = d33
			ps36.OverlayValues[34] = d34
				return bbs[5].RenderPS(ps36)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d34.Reg, 0)
			ctx.EmitJcc(CcNE, lbl13)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl6)
			ps37 := PhiState{General: true}
			ps37.OverlayValues = make([]JITValueDesc, 35)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[4] = d4
			ps37.OverlayValues[5] = d5
			ps37.OverlayValues[6] = d6
			ps37.OverlayValues[7] = d7
			ps37.OverlayValues[8] = d8
			ps37.OverlayValues[9] = d9
			ps37.OverlayValues[10] = d10
			ps37.OverlayValues[13] = d13
			ps37.OverlayValues[14] = d14
			ps37.OverlayValues[30] = d30
			ps37.OverlayValues[31] = d31
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			ps37.OverlayValues[34] = d34
			ps38 := PhiState{General: true}
			ps38.OverlayValues = make([]JITValueDesc, 35)
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
			ps38.OverlayValues[13] = d13
			ps38.OverlayValues[14] = d14
			ps38.OverlayValues[30] = d30
			ps38.OverlayValues[31] = d31
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[33] = d33
			ps38.OverlayValues[34] = d34
			snap39 := d0
			snap40 := d1
			snap41 := d2
			snap42 := d4
			snap43 := d5
			snap44 := d6
			snap45 := d7
			snap46 := d8
			snap47 := d9
			snap48 := d10
			snap49 := d13
			snap50 := d14
			snap51 := d30
			snap52 := d31
			snap53 := d32
			snap54 := d33
			snap55 := d34
			alloc56 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps38)
			}
			ctx.RestoreAllocState(alloc56)
			d0 = snap39
			d1 = snap40
			d2 = snap41
			d4 = snap42
			d5 = snap43
			d6 = snap44
			d7 = snap45
			d8 = snap46
			d9 = snap47
			d10 = snap48
			d13 = snap49
			d14 = snap50
			d30 = snap51
			d31 = snap52
			d32 = snap53
			d33 = snap54
			d34 = snap55
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps37)
			}
			return result
			ctx.FreeDesc(&d32)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d0, &result)
				result.Type = d0.Type
			} else {
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(result, d0)
					result.Type = tagBool
				case tagInt:
					ctx.EmitMakeInt(result, d0)
					result.Type = tagInt
				case tagFloat:
					ctx.EmitMakeFloat(result, d0)
					result.Type = tagFloat
				case tagNil:
					ctx.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d0, &result)
					result.Type = d0.Type
				}
			}
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			ctx.EnsureDesc(&d30)
			if d30.Loc == LocReg {
				ctx.ProtectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.ProtectReg(d30.Reg)
				ctx.ProtectReg(d30.Reg2)
			}
			d57 = d30
			if d57.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			if d57.Loc == LocRegPair || d57.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d57, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d57, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d58 = d8
			if d58.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d58)
			ctx.EmitStoreToStack(d58, int32(bbs[1].PhiBase)+int32(16))
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			if d30.Loc == LocReg {
				ctx.UnprotectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.UnprotectReg(d30.Reg)
				ctx.UnprotectReg(d30.Reg2)
			}
			ps59 := PhiState{General: ps.General}
			ps59.OverlayValues = make([]JITValueDesc, 59)
			ps59.OverlayValues[0] = d0
			ps59.OverlayValues[1] = d1
			ps59.OverlayValues[2] = d2
			ps59.OverlayValues[4] = d4
			ps59.OverlayValues[5] = d5
			ps59.OverlayValues[6] = d6
			ps59.OverlayValues[7] = d7
			ps59.OverlayValues[8] = d8
			ps59.OverlayValues[9] = d9
			ps59.OverlayValues[10] = d10
			ps59.OverlayValues[13] = d13
			ps59.OverlayValues[14] = d14
			ps59.OverlayValues[30] = d30
			ps59.OverlayValues[31] = d31
			ps59.OverlayValues[32] = d32
			ps59.OverlayValues[33] = d33
			ps59.OverlayValues[34] = d34
			ps59.OverlayValues[57] = d57
			ps59.OverlayValues[58] = d58
			ps59.PhiValues = make([]JITValueDesc, 2)
			d60 = d30
			ps59.PhiValues[0] = d60
			d61 = d8
			ps59.PhiValues[1] = d61
			if ps59.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps59)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
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
			ctx.ReclaimUntrackedRegs()
			d63 = d30
			d63.ID = 0
			d62 = ctx.EmitTagEqualsBorrowed(&d63, tagNil, JITValueDesc{Loc: LocAny})
			d64 = d62
			ctx.EnsureDesc(&d64)
			if d64.Loc != LocImm && d64.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d64.Loc == LocImm {
				if d64.Imm.Bool() {
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d65 = d0
			if d65.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d65)
			if d65.Loc == LocRegPair || d65.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d65, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d65, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d66 = d8
			if d66.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			ctx.EmitStoreToStack(d66, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ps67 := PhiState{General: ps.General}
			ps67.OverlayValues = make([]JITValueDesc, 67)
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
			ps67.OverlayValues[13] = d13
			ps67.OverlayValues[14] = d14
			ps67.OverlayValues[30] = d30
			ps67.OverlayValues[31] = d31
			ps67.OverlayValues[32] = d32
			ps67.OverlayValues[33] = d33
			ps67.OverlayValues[34] = d34
			ps67.OverlayValues[57] = d57
			ps67.OverlayValues[58] = d58
			ps67.OverlayValues[60] = d60
			ps67.OverlayValues[61] = d61
			ps67.OverlayValues[62] = d62
			ps67.OverlayValues[63] = d63
			ps67.OverlayValues[64] = d64
			ps67.OverlayValues[65] = d65
			ps67.OverlayValues[66] = d66
			ps67.PhiValues = make([]JITValueDesc, 2)
			d68 = d0
			ps67.PhiValues[0] = d68
			d69 = d8
			ps67.PhiValues[1] = d69
					return bbs[1].RenderPS(ps67)
				}
			ps70 := PhiState{General: ps.General}
			ps70.OverlayValues = make([]JITValueDesc, 70)
			ps70.OverlayValues[0] = d0
			ps70.OverlayValues[1] = d1
			ps70.OverlayValues[2] = d2
			ps70.OverlayValues[4] = d4
			ps70.OverlayValues[5] = d5
			ps70.OverlayValues[6] = d6
			ps70.OverlayValues[7] = d7
			ps70.OverlayValues[8] = d8
			ps70.OverlayValues[9] = d9
			ps70.OverlayValues[10] = d10
			ps70.OverlayValues[13] = d13
			ps70.OverlayValues[14] = d14
			ps70.OverlayValues[30] = d30
			ps70.OverlayValues[31] = d31
			ps70.OverlayValues[32] = d32
			ps70.OverlayValues[33] = d33
			ps70.OverlayValues[34] = d34
			ps70.OverlayValues[57] = d57
			ps70.OverlayValues[58] = d58
			ps70.OverlayValues[60] = d60
			ps70.OverlayValues[61] = d61
			ps70.OverlayValues[62] = d62
			ps70.OverlayValues[63] = d63
			ps70.OverlayValues[64] = d64
			ps70.OverlayValues[65] = d65
			ps70.OverlayValues[66] = d66
			ps70.OverlayValues[68] = d68
			ps70.OverlayValues[69] = d69
				return bbs[7].RenderPS(ps70)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d64.Reg, 0)
			ctx.EmitJcc(CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d71 = d0
			if d71.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d71)
			if d71.Loc == LocRegPair || d71.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d71, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d71, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d72 = d8
			if d72.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d72)
			ctx.EmitStoreToStack(d72, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl8)
			ps73 := PhiState{General: true}
			ps73.OverlayValues = make([]JITValueDesc, 73)
			ps73.OverlayValues[0] = d0
			ps73.OverlayValues[1] = d1
			ps73.OverlayValues[2] = d2
			ps73.OverlayValues[4] = d4
			ps73.OverlayValues[5] = d5
			ps73.OverlayValues[6] = d6
			ps73.OverlayValues[7] = d7
			ps73.OverlayValues[8] = d8
			ps73.OverlayValues[9] = d9
			ps73.OverlayValues[10] = d10
			ps73.OverlayValues[13] = d13
			ps73.OverlayValues[14] = d14
			ps73.OverlayValues[30] = d30
			ps73.OverlayValues[31] = d31
			ps73.OverlayValues[32] = d32
			ps73.OverlayValues[33] = d33
			ps73.OverlayValues[34] = d34
			ps73.OverlayValues[57] = d57
			ps73.OverlayValues[58] = d58
			ps73.OverlayValues[60] = d60
			ps73.OverlayValues[61] = d61
			ps73.OverlayValues[62] = d62
			ps73.OverlayValues[63] = d63
			ps73.OverlayValues[64] = d64
			ps73.OverlayValues[65] = d65
			ps73.OverlayValues[66] = d66
			ps73.OverlayValues[68] = d68
			ps73.OverlayValues[69] = d69
			ps73.OverlayValues[71] = d71
			ps73.OverlayValues[72] = d72
			ps73.PhiValues = make([]JITValueDesc, 2)
			d75 = d0
			ps73.PhiValues[0] = d75
			d76 = d8
			ps73.PhiValues[1] = d76
			ps74 := PhiState{General: true}
			ps74.OverlayValues = make([]JITValueDesc, 77)
			ps74.OverlayValues[0] = d0
			ps74.OverlayValues[1] = d1
			ps74.OverlayValues[2] = d2
			ps74.OverlayValues[4] = d4
			ps74.OverlayValues[5] = d5
			ps74.OverlayValues[6] = d6
			ps74.OverlayValues[7] = d7
			ps74.OverlayValues[8] = d8
			ps74.OverlayValues[9] = d9
			ps74.OverlayValues[10] = d10
			ps74.OverlayValues[13] = d13
			ps74.OverlayValues[14] = d14
			ps74.OverlayValues[30] = d30
			ps74.OverlayValues[31] = d31
			ps74.OverlayValues[32] = d32
			ps74.OverlayValues[33] = d33
			ps74.OverlayValues[34] = d34
			ps74.OverlayValues[57] = d57
			ps74.OverlayValues[58] = d58
			ps74.OverlayValues[60] = d60
			ps74.OverlayValues[61] = d61
			ps74.OverlayValues[62] = d62
			ps74.OverlayValues[63] = d63
			ps74.OverlayValues[64] = d64
			ps74.OverlayValues[65] = d65
			ps74.OverlayValues[66] = d66
			ps74.OverlayValues[68] = d68
			ps74.OverlayValues[69] = d69
			ps74.OverlayValues[71] = d71
			ps74.OverlayValues[72] = d72
			ps74.OverlayValues[75] = d75
			ps74.OverlayValues[76] = d76
			snap77 := d0
			snap78 := d1
			snap79 := d2
			snap80 := d4
			snap81 := d5
			snap82 := d6
			snap83 := d7
			snap84 := d8
			snap85 := d9
			snap86 := d10
			snap87 := d13
			snap88 := d14
			snap89 := d30
			snap90 := d31
			snap91 := d32
			snap92 := d33
			snap93 := d34
			snap94 := d57
			snap95 := d58
			snap96 := d60
			snap97 := d61
			snap98 := d62
			snap99 := d63
			snap100 := d64
			snap101 := d65
			snap102 := d66
			snap103 := d68
			snap104 := d69
			snap105 := d71
			snap106 := d72
			snap107 := d75
			snap108 := d76
			alloc109 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps73)
			}
			ctx.RestoreAllocState(alloc109)
			d0 = snap77
			d1 = snap78
			d2 = snap79
			d4 = snap80
			d5 = snap81
			d6 = snap82
			d7 = snap83
			d8 = snap84
			d9 = snap85
			d10 = snap86
			d13 = snap87
			d14 = snap88
			d30 = snap89
			d31 = snap90
			d32 = snap91
			d33 = snap92
			d34 = snap93
			d57 = snap94
			d58 = snap95
			d60 = snap96
			d61 = snap97
			d62 = snap98
			d63 = snap99
			d64 = snap100
			d65 = snap101
			d66 = snap102
			d68 = snap103
			d69 = snap104
			d71 = snap105
			d72 = snap106
			d75 = snap107
			d76 = snap108
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps74)
			}
			return result
			ctx.FreeDesc(&d62)
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
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
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
				d76 = ps.OverlayValues[76]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			ctx.EnsureDesc(&d30)
			if d30.Loc == LocReg {
				ctx.ProtectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.ProtectReg(d30.Reg)
				ctx.ProtectReg(d30.Reg2)
			}
			d110 = d30
			if d110.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d110)
			if d110.Loc == LocRegPair || d110.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d110, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d110, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d111 = d8
			if d111.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d111)
			ctx.EmitStoreToStack(d111, int32(bbs[1].PhiBase)+int32(16))
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			if d30.Loc == LocReg {
				ctx.UnprotectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.UnprotectReg(d30.Reg)
				ctx.UnprotectReg(d30.Reg2)
			}
			ps112 := PhiState{General: ps.General}
			ps112.OverlayValues = make([]JITValueDesc, 112)
			ps112.OverlayValues[0] = d0
			ps112.OverlayValues[1] = d1
			ps112.OverlayValues[2] = d2
			ps112.OverlayValues[4] = d4
			ps112.OverlayValues[5] = d5
			ps112.OverlayValues[6] = d6
			ps112.OverlayValues[7] = d7
			ps112.OverlayValues[8] = d8
			ps112.OverlayValues[9] = d9
			ps112.OverlayValues[10] = d10
			ps112.OverlayValues[13] = d13
			ps112.OverlayValues[14] = d14
			ps112.OverlayValues[30] = d30
			ps112.OverlayValues[31] = d31
			ps112.OverlayValues[32] = d32
			ps112.OverlayValues[33] = d33
			ps112.OverlayValues[34] = d34
			ps112.OverlayValues[57] = d57
			ps112.OverlayValues[58] = d58
			ps112.OverlayValues[60] = d60
			ps112.OverlayValues[61] = d61
			ps112.OverlayValues[62] = d62
			ps112.OverlayValues[63] = d63
			ps112.OverlayValues[64] = d64
			ps112.OverlayValues[65] = d65
			ps112.OverlayValues[66] = d66
			ps112.OverlayValues[68] = d68
			ps112.OverlayValues[69] = d69
			ps112.OverlayValues[71] = d71
			ps112.OverlayValues[72] = d72
			ps112.OverlayValues[75] = d75
			ps112.OverlayValues[76] = d76
			ps112.OverlayValues[110] = d110
			ps112.OverlayValues[111] = d111
			ps112.PhiValues = make([]JITValueDesc, 2)
			d113 = d30
			ps112.PhiValues[0] = d113
			d114 = d8
			ps112.PhiValues[1] = d114
			if ps112.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps112)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
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
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
				d76 = ps.OverlayValues[76]
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
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			if d30.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d30.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d30)
				} else if d30.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d30)
				} else if d30.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d30)
				} else if d30.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d30.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d30 = tmpPair
			} else if d30.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocRegExcept(d30.Reg), Reg2: ctx.AllocRegExcept(d30.Reg)}
				switch d30.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d30)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d30)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d30)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d30)
				d30 = tmpPair
			}
			if d30.Loc != LocRegPair && d30.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d115 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d30, d0}, 1)
			d116 = d115
			ctx.EnsureDesc(&d116)
			if d116.Loc != LocImm && d116.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d116.Loc == LocImm {
				if d116.Imm.Bool() {
			ps117 := PhiState{General: ps.General}
			ps117.OverlayValues = make([]JITValueDesc, 117)
			ps117.OverlayValues[0] = d0
			ps117.OverlayValues[1] = d1
			ps117.OverlayValues[2] = d2
			ps117.OverlayValues[4] = d4
			ps117.OverlayValues[5] = d5
			ps117.OverlayValues[6] = d6
			ps117.OverlayValues[7] = d7
			ps117.OverlayValues[8] = d8
			ps117.OverlayValues[9] = d9
			ps117.OverlayValues[10] = d10
			ps117.OverlayValues[13] = d13
			ps117.OverlayValues[14] = d14
			ps117.OverlayValues[30] = d30
			ps117.OverlayValues[31] = d31
			ps117.OverlayValues[32] = d32
			ps117.OverlayValues[33] = d33
			ps117.OverlayValues[34] = d34
			ps117.OverlayValues[57] = d57
			ps117.OverlayValues[58] = d58
			ps117.OverlayValues[60] = d60
			ps117.OverlayValues[61] = d61
			ps117.OverlayValues[62] = d62
			ps117.OverlayValues[63] = d63
			ps117.OverlayValues[64] = d64
			ps117.OverlayValues[65] = d65
			ps117.OverlayValues[66] = d66
			ps117.OverlayValues[68] = d68
			ps117.OverlayValues[69] = d69
			ps117.OverlayValues[71] = d71
			ps117.OverlayValues[72] = d72
			ps117.OverlayValues[75] = d75
			ps117.OverlayValues[76] = d76
			ps117.OverlayValues[110] = d110
			ps117.OverlayValues[111] = d111
			ps117.OverlayValues[113] = d113
			ps117.OverlayValues[114] = d114
			ps117.OverlayValues[115] = d115
			ps117.OverlayValues[116] = d116
					return bbs[6].RenderPS(ps117)
				}
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d118 = d0
			if d118.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d118)
			if d118.Loc == LocRegPair || d118.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d118, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d118, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d119 = d8
			if d119.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d119)
			ctx.EmitStoreToStack(d119, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ps120 := PhiState{General: ps.General}
			ps120.OverlayValues = make([]JITValueDesc, 120)
			ps120.OverlayValues[0] = d0
			ps120.OverlayValues[1] = d1
			ps120.OverlayValues[2] = d2
			ps120.OverlayValues[4] = d4
			ps120.OverlayValues[5] = d5
			ps120.OverlayValues[6] = d6
			ps120.OverlayValues[7] = d7
			ps120.OverlayValues[8] = d8
			ps120.OverlayValues[9] = d9
			ps120.OverlayValues[10] = d10
			ps120.OverlayValues[13] = d13
			ps120.OverlayValues[14] = d14
			ps120.OverlayValues[30] = d30
			ps120.OverlayValues[31] = d31
			ps120.OverlayValues[32] = d32
			ps120.OverlayValues[33] = d33
			ps120.OverlayValues[34] = d34
			ps120.OverlayValues[57] = d57
			ps120.OverlayValues[58] = d58
			ps120.OverlayValues[60] = d60
			ps120.OverlayValues[61] = d61
			ps120.OverlayValues[62] = d62
			ps120.OverlayValues[63] = d63
			ps120.OverlayValues[64] = d64
			ps120.OverlayValues[65] = d65
			ps120.OverlayValues[66] = d66
			ps120.OverlayValues[68] = d68
			ps120.OverlayValues[69] = d69
			ps120.OverlayValues[71] = d71
			ps120.OverlayValues[72] = d72
			ps120.OverlayValues[75] = d75
			ps120.OverlayValues[76] = d76
			ps120.OverlayValues[110] = d110
			ps120.OverlayValues[111] = d111
			ps120.OverlayValues[113] = d113
			ps120.OverlayValues[114] = d114
			ps120.OverlayValues[115] = d115
			ps120.OverlayValues[116] = d116
			ps120.OverlayValues[118] = d118
			ps120.OverlayValues[119] = d119
			ps120.PhiValues = make([]JITValueDesc, 2)
			d121 = d0
			ps120.PhiValues[0] = d121
			d122 = d8
			ps120.PhiValues[1] = d122
				return bbs[1].RenderPS(ps120)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d116.Reg, 0)
			ctx.EmitJcc(CcNE, lbl17)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl18)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d123 = d0
			if d123.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d123)
			if d123.Loc == LocRegPair || d123.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d123, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d123, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d124 = d8
			if d124.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ctx.EmitJmp(lbl2)
			ps125 := PhiState{General: true}
			ps125.OverlayValues = make([]JITValueDesc, 125)
			ps125.OverlayValues[0] = d0
			ps125.OverlayValues[1] = d1
			ps125.OverlayValues[2] = d2
			ps125.OverlayValues[4] = d4
			ps125.OverlayValues[5] = d5
			ps125.OverlayValues[6] = d6
			ps125.OverlayValues[7] = d7
			ps125.OverlayValues[8] = d8
			ps125.OverlayValues[9] = d9
			ps125.OverlayValues[10] = d10
			ps125.OverlayValues[13] = d13
			ps125.OverlayValues[14] = d14
			ps125.OverlayValues[30] = d30
			ps125.OverlayValues[31] = d31
			ps125.OverlayValues[32] = d32
			ps125.OverlayValues[33] = d33
			ps125.OverlayValues[34] = d34
			ps125.OverlayValues[57] = d57
			ps125.OverlayValues[58] = d58
			ps125.OverlayValues[60] = d60
			ps125.OverlayValues[61] = d61
			ps125.OverlayValues[62] = d62
			ps125.OverlayValues[63] = d63
			ps125.OverlayValues[64] = d64
			ps125.OverlayValues[65] = d65
			ps125.OverlayValues[66] = d66
			ps125.OverlayValues[68] = d68
			ps125.OverlayValues[69] = d69
			ps125.OverlayValues[71] = d71
			ps125.OverlayValues[72] = d72
			ps125.OverlayValues[75] = d75
			ps125.OverlayValues[76] = d76
			ps125.OverlayValues[110] = d110
			ps125.OverlayValues[111] = d111
			ps125.OverlayValues[113] = d113
			ps125.OverlayValues[114] = d114
			ps125.OverlayValues[115] = d115
			ps125.OverlayValues[116] = d116
			ps125.OverlayValues[118] = d118
			ps125.OverlayValues[119] = d119
			ps125.OverlayValues[121] = d121
			ps125.OverlayValues[122] = d122
			ps125.OverlayValues[123] = d123
			ps125.OverlayValues[124] = d124
			ps126 := PhiState{General: true}
			ps126.OverlayValues = make([]JITValueDesc, 125)
			ps126.OverlayValues[0] = d0
			ps126.OverlayValues[1] = d1
			ps126.OverlayValues[2] = d2
			ps126.OverlayValues[4] = d4
			ps126.OverlayValues[5] = d5
			ps126.OverlayValues[6] = d6
			ps126.OverlayValues[7] = d7
			ps126.OverlayValues[8] = d8
			ps126.OverlayValues[9] = d9
			ps126.OverlayValues[10] = d10
			ps126.OverlayValues[13] = d13
			ps126.OverlayValues[14] = d14
			ps126.OverlayValues[30] = d30
			ps126.OverlayValues[31] = d31
			ps126.OverlayValues[32] = d32
			ps126.OverlayValues[33] = d33
			ps126.OverlayValues[34] = d34
			ps126.OverlayValues[57] = d57
			ps126.OverlayValues[58] = d58
			ps126.OverlayValues[60] = d60
			ps126.OverlayValues[61] = d61
			ps126.OverlayValues[62] = d62
			ps126.OverlayValues[63] = d63
			ps126.OverlayValues[64] = d64
			ps126.OverlayValues[65] = d65
			ps126.OverlayValues[66] = d66
			ps126.OverlayValues[68] = d68
			ps126.OverlayValues[69] = d69
			ps126.OverlayValues[71] = d71
			ps126.OverlayValues[72] = d72
			ps126.OverlayValues[75] = d75
			ps126.OverlayValues[76] = d76
			ps126.OverlayValues[110] = d110
			ps126.OverlayValues[111] = d111
			ps126.OverlayValues[113] = d113
			ps126.OverlayValues[114] = d114
			ps126.OverlayValues[115] = d115
			ps126.OverlayValues[116] = d116
			ps126.OverlayValues[118] = d118
			ps126.OverlayValues[119] = d119
			ps126.OverlayValues[121] = d121
			ps126.OverlayValues[122] = d122
			ps126.OverlayValues[123] = d123
			ps126.OverlayValues[124] = d124
			ps126.PhiValues = make([]JITValueDesc, 2)
			d127 = d0
			ps126.PhiValues[0] = d127
			d128 = d8
			ps126.PhiValues[1] = d128
			snap129 := d0
			snap130 := d1
			snap131 := d2
			snap132 := d4
			snap133 := d5
			snap134 := d6
			snap135 := d7
			snap136 := d8
			snap137 := d9
			snap138 := d10
			snap139 := d13
			snap140 := d14
			snap141 := d30
			snap142 := d31
			snap143 := d32
			snap144 := d33
			snap145 := d34
			snap146 := d57
			snap147 := d58
			snap148 := d60
			snap149 := d61
			snap150 := d62
			snap151 := d63
			snap152 := d64
			snap153 := d65
			snap154 := d66
			snap155 := d68
			snap156 := d69
			snap157 := d71
			snap158 := d72
			snap159 := d75
			snap160 := d76
			snap161 := d110
			snap162 := d111
			snap163 := d113
			snap164 := d114
			snap165 := d115
			snap166 := d116
			snap167 := d118
			snap168 := d119
			snap169 := d121
			snap170 := d122
			snap171 := d123
			snap172 := d124
			snap173 := d127
			snap174 := d128
			alloc175 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps126)
			}
			ctx.RestoreAllocState(alloc175)
			d0 = snap129
			d1 = snap130
			d2 = snap131
			d4 = snap132
			d5 = snap133
			d6 = snap134
			d7 = snap135
			d8 = snap136
			d9 = snap137
			d10 = snap138
			d13 = snap139
			d14 = snap140
			d30 = snap141
			d31 = snap142
			d32 = snap143
			d33 = snap144
			d34 = snap145
			d57 = snap146
			d58 = snap147
			d60 = snap148
			d61 = snap149
			d62 = snap150
			d63 = snap151
			d64 = snap152
			d65 = snap153
			d66 = snap154
			d68 = snap155
			d69 = snap156
			d71 = snap157
			d72 = snap158
			d75 = snap159
			d76 = snap160
			d110 = snap161
			d111 = snap162
			d113 = snap163
			d114 = snap164
			d115 = snap165
			d116 = snap166
			d118 = snap167
			d119 = snap168
			d121 = snap169
			d122 = snap170
			d123 = snap171
			d124 = snap172
			d127 = snap173
			d128 = snap174
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps125)
			}
			return result
			ctx.FreeDesc(&d115)
			return result
			}
			argPinned176 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned176 = append(argPinned176, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned176 = append(argPinned176, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned176 = append(argPinned176, ai.Reg2)
					}
				}
			}
			ps177 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps177)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(32))
			ctx.EmitAddRSP32(int32(32))
			for _, r := range argPinned176 {
				ctx.UnprotectReg(r)
			}
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
			var d13 JITValueDesc
			_ = d13
			var d14 JITValueDesc
			_ = d14
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
			var d57 JITValueDesc
			_ = d57
			var d58 JITValueDesc
			_ = d58
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
			var d68 JITValueDesc
			_ = d68
			var d69 JITValueDesc
			_ = d69
			var d71 JITValueDesc
			_ = d71
			var d72 JITValueDesc
			_ = d72
			var d75 JITValueDesc
			_ = d75
			var d76 JITValueDesc
			_ = d76
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
			var d121 JITValueDesc
			_ = d121
			var d122 JITValueDesc
			_ = d122
			var d123 JITValueDesc
			_ = d123
			var d124 JITValueDesc
			_ = d124
			var d127 JITValueDesc
			_ = d127
			var d128 JITValueDesc
			_ = d128
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(32)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(32)
			}
			d0 := JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			var bbs [8]BBDescriptor
			bbs[1].PhiBase = int32(0)
			bbs[1].PhiCount = uint16(2)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			d2 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(len(args)))}
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[1].PhiBase)+int32(16))
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
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps3)
			return result
			}
			bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d6 := ps.PhiValues[0]
					ctx.EnsureDesc(&d6)
					ctx.EmitStoreScmerToStack(d6, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d7 := ps.PhiValues[1]
					ctx.EnsureDesc(&d7)
					ctx.EmitStoreToStack(d7, int32(bbs[1].PhiBase)+int32(16))
				}
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
				ctx.EmitMovRegReg(scratch, d1.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
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
					ctx.EmitCmpRegImm32(d8.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
					ctx.EmitCmpInt64(d8.Reg, RegR11)
				}
				ctx.EmitSetcc(r1, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d9)
			} else if d8.Loc == LocImm {
				r2 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d8.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d2.Reg)
				ctx.EmitSetcc(r2, CcL)
				d9 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r2}
				ctx.BindReg(r2, &d9)
			} else {
				r3 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitCmpInt64(d8.Reg, d2.Reg)
				ctx.EmitSetcc(r3, CcL)
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
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d13 := ps.PhiValues[0]
					ctx.EnsureDesc(&d13)
					ctx.EmitStoreScmerToStack(d13, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != LocNone {
					d14 := ps.PhiValues[1]
					ctx.EnsureDesc(&d14)
					ctx.EmitStoreToStack(d14, int32(bbs[1].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d10.Reg, 0)
			ctx.EmitJcc(CcNE, lbl9)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl4)
			ps15 := PhiState{General: true}
			ps15.OverlayValues = make([]JITValueDesc, 15)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[6] = d6
			ps15.OverlayValues[7] = d7
			ps15.OverlayValues[8] = d8
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[13] = d13
			ps15.OverlayValues[14] = d14
			ps16 := PhiState{General: true}
			ps16.OverlayValues = make([]JITValueDesc, 15)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[4] = d4
			ps16.OverlayValues[5] = d5
			ps16.OverlayValues[6] = d6
			ps16.OverlayValues[7] = d7
			ps16.OverlayValues[8] = d8
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[13] = d13
			ps16.OverlayValues[14] = d14
			snap17 := d0
			snap18 := d1
			snap19 := d2
			snap20 := d4
			snap21 := d5
			snap22 := d6
			snap23 := d7
			snap24 := d8
			snap25 := d9
			snap26 := d10
			snap27 := d13
			snap28 := d14
			alloc29 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc29)
			d0 = snap17
			d1 = snap18
			d2 = snap19
			d4 = snap20
			d5 = snap21
			d6 = snap22
			d7 = snap23
			d8 = snap24
			d9 = snap25
			d10 = snap26
			d13 = snap27
			d14 = snap28
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps15)
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
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			var d30 JITValueDesc
			if d8.Loc == LocImm {
				idx := int(d8.Imm.Int())
				if idx < 0 || idx >= len(args) {
					panic("jitgen: dynamic args index out of range")
				}
				d30 = args[idx]
				d30.ID = 0
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
				lbl11 := ctx.ReserveLabel()
				lbl12 := ctx.ReserveLabel()
				ctx.EmitCmpRegImm32(d8.Reg, int32(len(args)))
				ctx.EmitJcc(CcAE, lbl12)
				for i := 0; i < len(args); i++ {
					nextLbl := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d8.Reg, int32(i))
					ctx.EmitJcc(CcNE, nextLbl)
					ai := args[i]
					ai.ID = 0
					switch ai.Loc {
					case LocRegPair:
						ctx.EmitMovRegReg(r4, ai.Reg)
						ctx.EmitMovRegReg(r5, ai.Reg2)
					case LocStackPair:
						tmp := ai
						ctx.EnsureDesc(&tmp)
						if tmp.Loc != LocRegPair {
							panic("jitgen: emitter args index expected Scmer pair")
						}
						ctx.EmitMovRegReg(r4, tmp.Reg)
						ctx.EmitMovRegReg(r5, tmp.Reg2)
						ctx.FreeDesc(&tmp)
					case LocImm:
						pair := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
						ctx.BindReg(r4, &pair)
						ctx.BindReg(r5, &pair)
						if ai.Imm.GetTag() == tagInt {
							src := ai
							src.Type = tagInt
							src.Imm = NewInt(ai.Imm.Int())
							ctx.EmitMakeInt(pair, src)
						} else if ai.Imm.GetTag() == tagFloat {
							src := ai
							src.Type = tagFloat
							src.Imm = NewFloat(ai.Imm.Float())
							ctx.EmitMakeFloat(pair, src)
						} else if ai.Imm.GetTag() == tagBool {
							src := ai
							src.Type = tagBool
							src.Imm = NewBool(ai.Imm.Bool())
							ctx.EmitMakeBool(pair, src)
						} else if ai.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(pair)
						} else {
							ptrWord, auxWord := ai.Imm.RawWords()
							ctx.EmitMovRegImm64(r4, uint64(ptrWord))
							ctx.EmitMovRegImm64(r5, auxWord)
						}
					default:
						panic("jitgen: emitter args index expected Scmer pair")
					}
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(nextLbl)
				}
				ctx.MarkLabel(lbl12)
				d31 := JITValueDesc{Loc: LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d31)
				ctx.BindReg(r5, &d31)
				ctx.BindReg(r4, &d31)
				ctx.BindReg(r5, &d31)
				ctx.EmitMakeNil(d31)
				ctx.MarkLabel(lbl11)
				for _, r := range protected {
					ctx.UnprotectReg(r)
				}
				d30 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d30)
				ctx.BindReg(r5, &d30)
			}
			d33 = d0
			d33.ID = 0
			d32 = ctx.EmitTagEqualsBorrowed(&d33, tagNil, JITValueDesc{Loc: LocAny})
			d34 = d32
			ctx.EnsureDesc(&d34)
			if d34.Loc != LocImm && d34.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d34.Loc == LocImm {
				if d34.Imm.Bool() {
			ps35 := PhiState{General: ps.General}
			ps35.OverlayValues = make([]JITValueDesc, 35)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[1] = d1
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[4] = d4
			ps35.OverlayValues[5] = d5
			ps35.OverlayValues[6] = d6
			ps35.OverlayValues[7] = d7
			ps35.OverlayValues[8] = d8
			ps35.OverlayValues[9] = d9
			ps35.OverlayValues[10] = d10
			ps35.OverlayValues[13] = d13
			ps35.OverlayValues[14] = d14
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[32] = d32
			ps35.OverlayValues[33] = d33
			ps35.OverlayValues[34] = d34
					return bbs[4].RenderPS(ps35)
				}
			ps36 := PhiState{General: ps.General}
			ps36.OverlayValues = make([]JITValueDesc, 35)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[1] = d1
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[4] = d4
			ps36.OverlayValues[5] = d5
			ps36.OverlayValues[6] = d6
			ps36.OverlayValues[7] = d7
			ps36.OverlayValues[8] = d8
			ps36.OverlayValues[9] = d9
			ps36.OverlayValues[10] = d10
			ps36.OverlayValues[13] = d13
			ps36.OverlayValues[14] = d14
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[32] = d32
			ps36.OverlayValues[33] = d33
			ps36.OverlayValues[34] = d34
				return bbs[5].RenderPS(ps36)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d34.Reg, 0)
			ctx.EmitJcc(CcNE, lbl13)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl6)
			ps37 := PhiState{General: true}
			ps37.OverlayValues = make([]JITValueDesc, 35)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[4] = d4
			ps37.OverlayValues[5] = d5
			ps37.OverlayValues[6] = d6
			ps37.OverlayValues[7] = d7
			ps37.OverlayValues[8] = d8
			ps37.OverlayValues[9] = d9
			ps37.OverlayValues[10] = d10
			ps37.OverlayValues[13] = d13
			ps37.OverlayValues[14] = d14
			ps37.OverlayValues[30] = d30
			ps37.OverlayValues[31] = d31
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			ps37.OverlayValues[34] = d34
			ps38 := PhiState{General: true}
			ps38.OverlayValues = make([]JITValueDesc, 35)
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
			ps38.OverlayValues[13] = d13
			ps38.OverlayValues[14] = d14
			ps38.OverlayValues[30] = d30
			ps38.OverlayValues[31] = d31
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[33] = d33
			ps38.OverlayValues[34] = d34
			snap39 := d0
			snap40 := d1
			snap41 := d2
			snap42 := d4
			snap43 := d5
			snap44 := d6
			snap45 := d7
			snap46 := d8
			snap47 := d9
			snap48 := d10
			snap49 := d13
			snap50 := d14
			snap51 := d30
			snap52 := d31
			snap53 := d32
			snap54 := d33
			snap55 := d34
			alloc56 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps38)
			}
			ctx.RestoreAllocState(alloc56)
			d0 = snap39
			d1 = snap40
			d2 = snap41
			d4 = snap42
			d5 = snap43
			d6 = snap44
			d7 = snap45
			d8 = snap46
			d9 = snap47
			d10 = snap48
			d13 = snap49
			d14 = snap50
			d30 = snap51
			d31 = snap52
			d32 = snap53
			d33 = snap54
			d34 = snap55
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps37)
			}
			return result
			ctx.FreeDesc(&d32)
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocRegPair {
				ctx.EmitMovPairToResult(&d0, &result)
				result.Type = d0.Type
			} else {
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(result, d0)
					result.Type = tagBool
				case tagInt:
					ctx.EmitMakeInt(result, d0)
					result.Type = tagInt
				case tagFloat:
					ctx.EmitMakeFloat(result, d0)
					result.Type = tagFloat
				case tagNil:
					ctx.EmitMakeNil(result)
					result.Type = tagNil
				default:
					ctx.EmitMovPairToResult(&d0, &result)
					result.Type = d0.Type
				}
			}
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			ctx.EnsureDesc(&d30)
			if d30.Loc == LocReg {
				ctx.ProtectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.ProtectReg(d30.Reg)
				ctx.ProtectReg(d30.Reg2)
			}
			d57 = d30
			if d57.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			if d57.Loc == LocRegPair || d57.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d57, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d57, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d58 = d8
			if d58.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d58)
			ctx.EmitStoreToStack(d58, int32(bbs[1].PhiBase)+int32(16))
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			if d30.Loc == LocReg {
				ctx.UnprotectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.UnprotectReg(d30.Reg)
				ctx.UnprotectReg(d30.Reg2)
			}
			ps59 := PhiState{General: ps.General}
			ps59.OverlayValues = make([]JITValueDesc, 59)
			ps59.OverlayValues[0] = d0
			ps59.OverlayValues[1] = d1
			ps59.OverlayValues[2] = d2
			ps59.OverlayValues[4] = d4
			ps59.OverlayValues[5] = d5
			ps59.OverlayValues[6] = d6
			ps59.OverlayValues[7] = d7
			ps59.OverlayValues[8] = d8
			ps59.OverlayValues[9] = d9
			ps59.OverlayValues[10] = d10
			ps59.OverlayValues[13] = d13
			ps59.OverlayValues[14] = d14
			ps59.OverlayValues[30] = d30
			ps59.OverlayValues[31] = d31
			ps59.OverlayValues[32] = d32
			ps59.OverlayValues[33] = d33
			ps59.OverlayValues[34] = d34
			ps59.OverlayValues[57] = d57
			ps59.OverlayValues[58] = d58
			ps59.PhiValues = make([]JITValueDesc, 2)
			d60 = d30
			ps59.PhiValues[0] = d60
			d61 = d8
			ps59.PhiValues[1] = d61
			if ps59.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps59)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
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
			ctx.ReclaimUntrackedRegs()
			d63 = d30
			d63.ID = 0
			d62 = ctx.EmitTagEqualsBorrowed(&d63, tagNil, JITValueDesc{Loc: LocAny})
			d64 = d62
			ctx.EnsureDesc(&d64)
			if d64.Loc != LocImm && d64.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d64.Loc == LocImm {
				if d64.Imm.Bool() {
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d65 = d0
			if d65.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d65)
			if d65.Loc == LocRegPair || d65.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d65, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d65, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d66 = d8
			if d66.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			ctx.EmitStoreToStack(d66, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ps67 := PhiState{General: ps.General}
			ps67.OverlayValues = make([]JITValueDesc, 67)
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
			ps67.OverlayValues[13] = d13
			ps67.OverlayValues[14] = d14
			ps67.OverlayValues[30] = d30
			ps67.OverlayValues[31] = d31
			ps67.OverlayValues[32] = d32
			ps67.OverlayValues[33] = d33
			ps67.OverlayValues[34] = d34
			ps67.OverlayValues[57] = d57
			ps67.OverlayValues[58] = d58
			ps67.OverlayValues[60] = d60
			ps67.OverlayValues[61] = d61
			ps67.OverlayValues[62] = d62
			ps67.OverlayValues[63] = d63
			ps67.OverlayValues[64] = d64
			ps67.OverlayValues[65] = d65
			ps67.OverlayValues[66] = d66
			ps67.PhiValues = make([]JITValueDesc, 2)
			d68 = d0
			ps67.PhiValues[0] = d68
			d69 = d8
			ps67.PhiValues[1] = d69
					return bbs[1].RenderPS(ps67)
				}
			ps70 := PhiState{General: ps.General}
			ps70.OverlayValues = make([]JITValueDesc, 70)
			ps70.OverlayValues[0] = d0
			ps70.OverlayValues[1] = d1
			ps70.OverlayValues[2] = d2
			ps70.OverlayValues[4] = d4
			ps70.OverlayValues[5] = d5
			ps70.OverlayValues[6] = d6
			ps70.OverlayValues[7] = d7
			ps70.OverlayValues[8] = d8
			ps70.OverlayValues[9] = d9
			ps70.OverlayValues[10] = d10
			ps70.OverlayValues[13] = d13
			ps70.OverlayValues[14] = d14
			ps70.OverlayValues[30] = d30
			ps70.OverlayValues[31] = d31
			ps70.OverlayValues[32] = d32
			ps70.OverlayValues[33] = d33
			ps70.OverlayValues[34] = d34
			ps70.OverlayValues[57] = d57
			ps70.OverlayValues[58] = d58
			ps70.OverlayValues[60] = d60
			ps70.OverlayValues[61] = d61
			ps70.OverlayValues[62] = d62
			ps70.OverlayValues[63] = d63
			ps70.OverlayValues[64] = d64
			ps70.OverlayValues[65] = d65
			ps70.OverlayValues[66] = d66
			ps70.OverlayValues[68] = d68
			ps70.OverlayValues[69] = d69
				return bbs[7].RenderPS(ps70)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d64.Reg, 0)
			ctx.EmitJcc(CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d71 = d0
			if d71.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d71)
			if d71.Loc == LocRegPair || d71.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d71, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d71, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d72 = d8
			if d72.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d72)
			ctx.EmitStoreToStack(d72, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl8)
			ps73 := PhiState{General: true}
			ps73.OverlayValues = make([]JITValueDesc, 73)
			ps73.OverlayValues[0] = d0
			ps73.OverlayValues[1] = d1
			ps73.OverlayValues[2] = d2
			ps73.OverlayValues[4] = d4
			ps73.OverlayValues[5] = d5
			ps73.OverlayValues[6] = d6
			ps73.OverlayValues[7] = d7
			ps73.OverlayValues[8] = d8
			ps73.OverlayValues[9] = d9
			ps73.OverlayValues[10] = d10
			ps73.OverlayValues[13] = d13
			ps73.OverlayValues[14] = d14
			ps73.OverlayValues[30] = d30
			ps73.OverlayValues[31] = d31
			ps73.OverlayValues[32] = d32
			ps73.OverlayValues[33] = d33
			ps73.OverlayValues[34] = d34
			ps73.OverlayValues[57] = d57
			ps73.OverlayValues[58] = d58
			ps73.OverlayValues[60] = d60
			ps73.OverlayValues[61] = d61
			ps73.OverlayValues[62] = d62
			ps73.OverlayValues[63] = d63
			ps73.OverlayValues[64] = d64
			ps73.OverlayValues[65] = d65
			ps73.OverlayValues[66] = d66
			ps73.OverlayValues[68] = d68
			ps73.OverlayValues[69] = d69
			ps73.OverlayValues[71] = d71
			ps73.OverlayValues[72] = d72
			ps73.PhiValues = make([]JITValueDesc, 2)
			d75 = d0
			ps73.PhiValues[0] = d75
			d76 = d8
			ps73.PhiValues[1] = d76
			ps74 := PhiState{General: true}
			ps74.OverlayValues = make([]JITValueDesc, 77)
			ps74.OverlayValues[0] = d0
			ps74.OverlayValues[1] = d1
			ps74.OverlayValues[2] = d2
			ps74.OverlayValues[4] = d4
			ps74.OverlayValues[5] = d5
			ps74.OverlayValues[6] = d6
			ps74.OverlayValues[7] = d7
			ps74.OverlayValues[8] = d8
			ps74.OverlayValues[9] = d9
			ps74.OverlayValues[10] = d10
			ps74.OverlayValues[13] = d13
			ps74.OverlayValues[14] = d14
			ps74.OverlayValues[30] = d30
			ps74.OverlayValues[31] = d31
			ps74.OverlayValues[32] = d32
			ps74.OverlayValues[33] = d33
			ps74.OverlayValues[34] = d34
			ps74.OverlayValues[57] = d57
			ps74.OverlayValues[58] = d58
			ps74.OverlayValues[60] = d60
			ps74.OverlayValues[61] = d61
			ps74.OverlayValues[62] = d62
			ps74.OverlayValues[63] = d63
			ps74.OverlayValues[64] = d64
			ps74.OverlayValues[65] = d65
			ps74.OverlayValues[66] = d66
			ps74.OverlayValues[68] = d68
			ps74.OverlayValues[69] = d69
			ps74.OverlayValues[71] = d71
			ps74.OverlayValues[72] = d72
			ps74.OverlayValues[75] = d75
			ps74.OverlayValues[76] = d76
			snap77 := d0
			snap78 := d1
			snap79 := d2
			snap80 := d4
			snap81 := d5
			snap82 := d6
			snap83 := d7
			snap84 := d8
			snap85 := d9
			snap86 := d10
			snap87 := d13
			snap88 := d14
			snap89 := d30
			snap90 := d31
			snap91 := d32
			snap92 := d33
			snap93 := d34
			snap94 := d57
			snap95 := d58
			snap96 := d60
			snap97 := d61
			snap98 := d62
			snap99 := d63
			snap100 := d64
			snap101 := d65
			snap102 := d66
			snap103 := d68
			snap104 := d69
			snap105 := d71
			snap106 := d72
			snap107 := d75
			snap108 := d76
			alloc109 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps73)
			}
			ctx.RestoreAllocState(alloc109)
			d0 = snap77
			d1 = snap78
			d2 = snap79
			d4 = snap80
			d5 = snap81
			d6 = snap82
			d7 = snap83
			d8 = snap84
			d9 = snap85
			d10 = snap86
			d13 = snap87
			d14 = snap88
			d30 = snap89
			d31 = snap90
			d32 = snap91
			d33 = snap92
			d34 = snap93
			d57 = snap94
			d58 = snap95
			d60 = snap96
			d61 = snap97
			d62 = snap98
			d63 = snap99
			d64 = snap100
			d65 = snap101
			d66 = snap102
			d68 = snap103
			d69 = snap104
			d71 = snap105
			d72 = snap106
			d75 = snap107
			d76 = snap108
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps74)
			}
			return result
			ctx.FreeDesc(&d62)
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
			d0 = JITValueDesc{Loc: LocStackPair, Type: JITTypeUnknown, StackOff: int32(0)}
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != LocNone {
				d14 = ps.OverlayValues[14]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
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
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
				d76 = ps.OverlayValues[76]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			ctx.EnsureDesc(&d30)
			if d30.Loc == LocReg {
				ctx.ProtectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.ProtectReg(d30.Reg)
				ctx.ProtectReg(d30.Reg2)
			}
			d110 = d30
			if d110.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d110)
			if d110.Loc == LocRegPair || d110.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d110, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d110, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d111 = d8
			if d111.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d111)
			ctx.EmitStoreToStack(d111, int32(bbs[1].PhiBase)+int32(16))
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			if d30.Loc == LocReg {
				ctx.UnprotectReg(d30.Reg)
			} else if d30.Loc == LocRegPair {
				ctx.UnprotectReg(d30.Reg)
				ctx.UnprotectReg(d30.Reg2)
			}
			ps112 := PhiState{General: ps.General}
			ps112.OverlayValues = make([]JITValueDesc, 112)
			ps112.OverlayValues[0] = d0
			ps112.OverlayValues[1] = d1
			ps112.OverlayValues[2] = d2
			ps112.OverlayValues[4] = d4
			ps112.OverlayValues[5] = d5
			ps112.OverlayValues[6] = d6
			ps112.OverlayValues[7] = d7
			ps112.OverlayValues[8] = d8
			ps112.OverlayValues[9] = d9
			ps112.OverlayValues[10] = d10
			ps112.OverlayValues[13] = d13
			ps112.OverlayValues[14] = d14
			ps112.OverlayValues[30] = d30
			ps112.OverlayValues[31] = d31
			ps112.OverlayValues[32] = d32
			ps112.OverlayValues[33] = d33
			ps112.OverlayValues[34] = d34
			ps112.OverlayValues[57] = d57
			ps112.OverlayValues[58] = d58
			ps112.OverlayValues[60] = d60
			ps112.OverlayValues[61] = d61
			ps112.OverlayValues[62] = d62
			ps112.OverlayValues[63] = d63
			ps112.OverlayValues[64] = d64
			ps112.OverlayValues[65] = d65
			ps112.OverlayValues[66] = d66
			ps112.OverlayValues[68] = d68
			ps112.OverlayValues[69] = d69
			ps112.OverlayValues[71] = d71
			ps112.OverlayValues[72] = d72
			ps112.OverlayValues[75] = d75
			ps112.OverlayValues[76] = d76
			ps112.OverlayValues[110] = d110
			ps112.OverlayValues[111] = d111
			ps112.PhiValues = make([]JITValueDesc, 2)
			d113 = d30
			ps112.PhiValues[0] = d113
			d114 = d8
			ps112.PhiValues[1] = d114
			if ps112.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps112)
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
			d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(16)}
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
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
				d9 = ps.OverlayValues[9]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != LocNone {
				d57 = ps.OverlayValues[57]
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
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != LocNone {
				d76 = ps.OverlayValues[76]
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
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d0.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d0)
				} else if d0.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d0.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d0 = tmpPair
			} else if d0.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
				switch d0.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d0)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d0)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d0)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d0)
				d0 = tmpPair
			}
			if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg0)")
			}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			if d30.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d30.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d30)
				} else if d30.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d30)
				} else if d30.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d30)
				} else if d30.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d30.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d30 = tmpPair
			} else if d30.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d30.Type, Reg: ctx.AllocRegExcept(d30.Reg), Reg2: ctx.AllocRegExcept(d30.Reg)}
				switch d30.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d30)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d30)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d30)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d30)
				d30 = tmpPair
			}
			if d30.Loc != LocRegPair && d30.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (Less arg1)")
			}
			d115 = ctx.EmitGoCallScalar(GoFuncAddr(Less), []JITValueDesc{d0, d30}, 1)
			d116 = d115
			ctx.EnsureDesc(&d116)
			if d116.Loc != LocImm && d116.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d116.Loc == LocImm {
				if d116.Imm.Bool() {
			ps117 := PhiState{General: ps.General}
			ps117.OverlayValues = make([]JITValueDesc, 117)
			ps117.OverlayValues[0] = d0
			ps117.OverlayValues[1] = d1
			ps117.OverlayValues[2] = d2
			ps117.OverlayValues[4] = d4
			ps117.OverlayValues[5] = d5
			ps117.OverlayValues[6] = d6
			ps117.OverlayValues[7] = d7
			ps117.OverlayValues[8] = d8
			ps117.OverlayValues[9] = d9
			ps117.OverlayValues[10] = d10
			ps117.OverlayValues[13] = d13
			ps117.OverlayValues[14] = d14
			ps117.OverlayValues[30] = d30
			ps117.OverlayValues[31] = d31
			ps117.OverlayValues[32] = d32
			ps117.OverlayValues[33] = d33
			ps117.OverlayValues[34] = d34
			ps117.OverlayValues[57] = d57
			ps117.OverlayValues[58] = d58
			ps117.OverlayValues[60] = d60
			ps117.OverlayValues[61] = d61
			ps117.OverlayValues[62] = d62
			ps117.OverlayValues[63] = d63
			ps117.OverlayValues[64] = d64
			ps117.OverlayValues[65] = d65
			ps117.OverlayValues[66] = d66
			ps117.OverlayValues[68] = d68
			ps117.OverlayValues[69] = d69
			ps117.OverlayValues[71] = d71
			ps117.OverlayValues[72] = d72
			ps117.OverlayValues[75] = d75
			ps117.OverlayValues[76] = d76
			ps117.OverlayValues[110] = d110
			ps117.OverlayValues[111] = d111
			ps117.OverlayValues[113] = d113
			ps117.OverlayValues[114] = d114
			ps117.OverlayValues[115] = d115
			ps117.OverlayValues[116] = d116
					return bbs[6].RenderPS(ps117)
				}
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d118 = d0
			if d118.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d118)
			if d118.Loc == LocRegPair || d118.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d118, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d118, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d119 = d8
			if d119.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d119)
			ctx.EmitStoreToStack(d119, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ps120 := PhiState{General: ps.General}
			ps120.OverlayValues = make([]JITValueDesc, 120)
			ps120.OverlayValues[0] = d0
			ps120.OverlayValues[1] = d1
			ps120.OverlayValues[2] = d2
			ps120.OverlayValues[4] = d4
			ps120.OverlayValues[5] = d5
			ps120.OverlayValues[6] = d6
			ps120.OverlayValues[7] = d7
			ps120.OverlayValues[8] = d8
			ps120.OverlayValues[9] = d9
			ps120.OverlayValues[10] = d10
			ps120.OverlayValues[13] = d13
			ps120.OverlayValues[14] = d14
			ps120.OverlayValues[30] = d30
			ps120.OverlayValues[31] = d31
			ps120.OverlayValues[32] = d32
			ps120.OverlayValues[33] = d33
			ps120.OverlayValues[34] = d34
			ps120.OverlayValues[57] = d57
			ps120.OverlayValues[58] = d58
			ps120.OverlayValues[60] = d60
			ps120.OverlayValues[61] = d61
			ps120.OverlayValues[62] = d62
			ps120.OverlayValues[63] = d63
			ps120.OverlayValues[64] = d64
			ps120.OverlayValues[65] = d65
			ps120.OverlayValues[66] = d66
			ps120.OverlayValues[68] = d68
			ps120.OverlayValues[69] = d69
			ps120.OverlayValues[71] = d71
			ps120.OverlayValues[72] = d72
			ps120.OverlayValues[75] = d75
			ps120.OverlayValues[76] = d76
			ps120.OverlayValues[110] = d110
			ps120.OverlayValues[111] = d111
			ps120.OverlayValues[113] = d113
			ps120.OverlayValues[114] = d114
			ps120.OverlayValues[115] = d115
			ps120.OverlayValues[116] = d116
			ps120.OverlayValues[118] = d118
			ps120.OverlayValues[119] = d119
			ps120.PhiValues = make([]JITValueDesc, 2)
			d121 = d0
			ps120.PhiValues[0] = d121
			d122 = d8
			ps120.PhiValues[1] = d122
				return bbs[1].RenderPS(ps120)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d116.Reg, 0)
			ctx.EmitJcc(CcNE, lbl17)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl18)
			ctx.EnsureDesc(&d0)
			if d0.Loc == LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d8)
			if d8.Loc == LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			d123 = d0
			if d123.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d123)
			if d123.Loc == LocRegPair || d123.Loc == LocImm {
				ctx.EmitStoreScmerToStack(d123, int32(bbs[1].PhiBase)+int32(0))
			} else {
				ctx.EmitStoreToStack(d123, int32(bbs[1].PhiBase)+int32(0))
				ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Imm: NewInt(0)}, (int32(bbs[1].PhiBase)+int32(0))+8)
			}
			d124 = d8
			if d124.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d8.Loc == LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			ctx.EmitJmp(lbl2)
			ps125 := PhiState{General: true}
			ps125.OverlayValues = make([]JITValueDesc, 125)
			ps125.OverlayValues[0] = d0
			ps125.OverlayValues[1] = d1
			ps125.OverlayValues[2] = d2
			ps125.OverlayValues[4] = d4
			ps125.OverlayValues[5] = d5
			ps125.OverlayValues[6] = d6
			ps125.OverlayValues[7] = d7
			ps125.OverlayValues[8] = d8
			ps125.OverlayValues[9] = d9
			ps125.OverlayValues[10] = d10
			ps125.OverlayValues[13] = d13
			ps125.OverlayValues[14] = d14
			ps125.OverlayValues[30] = d30
			ps125.OverlayValues[31] = d31
			ps125.OverlayValues[32] = d32
			ps125.OverlayValues[33] = d33
			ps125.OverlayValues[34] = d34
			ps125.OverlayValues[57] = d57
			ps125.OverlayValues[58] = d58
			ps125.OverlayValues[60] = d60
			ps125.OverlayValues[61] = d61
			ps125.OverlayValues[62] = d62
			ps125.OverlayValues[63] = d63
			ps125.OverlayValues[64] = d64
			ps125.OverlayValues[65] = d65
			ps125.OverlayValues[66] = d66
			ps125.OverlayValues[68] = d68
			ps125.OverlayValues[69] = d69
			ps125.OverlayValues[71] = d71
			ps125.OverlayValues[72] = d72
			ps125.OverlayValues[75] = d75
			ps125.OverlayValues[76] = d76
			ps125.OverlayValues[110] = d110
			ps125.OverlayValues[111] = d111
			ps125.OverlayValues[113] = d113
			ps125.OverlayValues[114] = d114
			ps125.OverlayValues[115] = d115
			ps125.OverlayValues[116] = d116
			ps125.OverlayValues[118] = d118
			ps125.OverlayValues[119] = d119
			ps125.OverlayValues[121] = d121
			ps125.OverlayValues[122] = d122
			ps125.OverlayValues[123] = d123
			ps125.OverlayValues[124] = d124
			ps126 := PhiState{General: true}
			ps126.OverlayValues = make([]JITValueDesc, 125)
			ps126.OverlayValues[0] = d0
			ps126.OverlayValues[1] = d1
			ps126.OverlayValues[2] = d2
			ps126.OverlayValues[4] = d4
			ps126.OverlayValues[5] = d5
			ps126.OverlayValues[6] = d6
			ps126.OverlayValues[7] = d7
			ps126.OverlayValues[8] = d8
			ps126.OverlayValues[9] = d9
			ps126.OverlayValues[10] = d10
			ps126.OverlayValues[13] = d13
			ps126.OverlayValues[14] = d14
			ps126.OverlayValues[30] = d30
			ps126.OverlayValues[31] = d31
			ps126.OverlayValues[32] = d32
			ps126.OverlayValues[33] = d33
			ps126.OverlayValues[34] = d34
			ps126.OverlayValues[57] = d57
			ps126.OverlayValues[58] = d58
			ps126.OverlayValues[60] = d60
			ps126.OverlayValues[61] = d61
			ps126.OverlayValues[62] = d62
			ps126.OverlayValues[63] = d63
			ps126.OverlayValues[64] = d64
			ps126.OverlayValues[65] = d65
			ps126.OverlayValues[66] = d66
			ps126.OverlayValues[68] = d68
			ps126.OverlayValues[69] = d69
			ps126.OverlayValues[71] = d71
			ps126.OverlayValues[72] = d72
			ps126.OverlayValues[75] = d75
			ps126.OverlayValues[76] = d76
			ps126.OverlayValues[110] = d110
			ps126.OverlayValues[111] = d111
			ps126.OverlayValues[113] = d113
			ps126.OverlayValues[114] = d114
			ps126.OverlayValues[115] = d115
			ps126.OverlayValues[116] = d116
			ps126.OverlayValues[118] = d118
			ps126.OverlayValues[119] = d119
			ps126.OverlayValues[121] = d121
			ps126.OverlayValues[122] = d122
			ps126.OverlayValues[123] = d123
			ps126.OverlayValues[124] = d124
			ps126.PhiValues = make([]JITValueDesc, 2)
			d127 = d0
			ps126.PhiValues[0] = d127
			d128 = d8
			ps126.PhiValues[1] = d128
			snap129 := d0
			snap130 := d1
			snap131 := d2
			snap132 := d4
			snap133 := d5
			snap134 := d6
			snap135 := d7
			snap136 := d8
			snap137 := d9
			snap138 := d10
			snap139 := d13
			snap140 := d14
			snap141 := d30
			snap142 := d31
			snap143 := d32
			snap144 := d33
			snap145 := d34
			snap146 := d57
			snap147 := d58
			snap148 := d60
			snap149 := d61
			snap150 := d62
			snap151 := d63
			snap152 := d64
			snap153 := d65
			snap154 := d66
			snap155 := d68
			snap156 := d69
			snap157 := d71
			snap158 := d72
			snap159 := d75
			snap160 := d76
			snap161 := d110
			snap162 := d111
			snap163 := d113
			snap164 := d114
			snap165 := d115
			snap166 := d116
			snap167 := d118
			snap168 := d119
			snap169 := d121
			snap170 := d122
			snap171 := d123
			snap172 := d124
			snap173 := d127
			snap174 := d128
			alloc175 := ctx.SnapshotAllocState()
			if !bbs[1].Rendered {
				bbs[1].RenderPS(ps126)
			}
			ctx.RestoreAllocState(alloc175)
			d0 = snap129
			d1 = snap130
			d2 = snap131
			d4 = snap132
			d5 = snap133
			d6 = snap134
			d7 = snap135
			d8 = snap136
			d9 = snap137
			d10 = snap138
			d13 = snap139
			d14 = snap140
			d30 = snap141
			d31 = snap142
			d32 = snap143
			d33 = snap144
			d34 = snap145
			d57 = snap146
			d58 = snap147
			d60 = snap148
			d61 = snap149
			d62 = snap150
			d63 = snap151
			d64 = snap152
			d65 = snap153
			d66 = snap154
			d68 = snap155
			d69 = snap156
			d71 = snap157
			d72 = snap158
			d75 = snap159
			d76 = snap160
			d110 = snap161
			d111 = snap162
			d113 = snap163
			d114 = snap164
			d115 = snap165
			d116 = snap166
			d118 = snap167
			d119 = snap168
			d121 = snap169
			d122 = snap170
			d123 = snap171
			d124 = snap172
			d127 = snap173
			d128 = snap174
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps125)
			}
			return result
			ctx.FreeDesc(&d115)
			return result
			}
			argPinned176 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned176 = append(argPinned176, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned176 = append(argPinned176, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned176 = append(argPinned176, ai.Reg2)
					}
				}
			}
			ps177 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps177)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(32))
			ctx.EmitAddRSP32(int32(32))
			for _, r := range argPinned176 {
				ctx.UnprotectReg(r)
			}
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
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeFloat(result, d2)
			} else {
				ctx.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			return result
			}
			argPinned4 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned4 = append(argPinned4, ai.Reg2)
					}
				}
			}
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
			for _, r := range argPinned4 {
				ctx.UnprotectReg(r)
			}
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
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeFloat(result, d2)
			} else {
				ctx.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
			return result
			return result
			}
			argPinned4 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned4 = append(argPinned4, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned4 = append(argPinned4, ai.Reg2)
					}
				}
			}
			ps5 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps5)
			for _, r := range argPinned4 {
				ctx.UnprotectReg(r)
			}
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
			ctx.ResolveFixups()
			if result.Loc == LocAny {
				result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			if d2.Loc == LocImm {
				ctx.EmitMakeFloat(result, d2)
			} else {
				ctx.EmitMakeFloat(result, d2)
				ctx.FreeReg(d2.Reg)
			}
			result.Type = tagFloat
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
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
			var d17 JITValueDesc
			_ = d17
			var d18 JITValueDesc
			_ = d18
			var d20 JITValueDesc
			_ = d20
			var d22 JITValueDesc
			_ = d22
			var d23 JITValueDesc
			_ = d23
			var d26 JITValueDesc
			_ = d26
			var d41 JITValueDesc
			_ = d41
			var d42 JITValueDesc
			_ = d42
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
			var d53 JITValueDesc
			_ = d53
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
			/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			for i := range args {
				if args[i].MemPtr == 0 && (args[i].Loc == LocStack || args[i].Loc == LocStackPair) {
					args[i].StackOff += int32(16)
				}
			}
			if result.MemPtr == 0 && (result.Loc == LocStack || result.Loc == LocStackPair) {
				result.StackOff += int32(16)
			}
			d0 := JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
			var bbs [8]BBDescriptor
			bbs[4].PhiBase = int32(0)
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
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.ReserveLabel()
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d4.Reg, 0)
			ctx.EmitJcc(CcNE, lbl9)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl3)
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
			snap9 := d0
			snap10 := d1
			snap11 := d2
			snap12 := d3
			snap13 := d4
			alloc14 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps8)
			}
			ctx.RestoreAllocState(alloc14)
			d0 = snap9
			d1 = snap10
			d2 = snap11
			d3 = snap12
			d4 = snap13
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
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			d15 = args[0]
			d15.ID = 0
			var d16 JITValueDesc
			if d15.Loc == LocImm {
				d16 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d15.Imm.Float())}
			} else if d15.Type == tagFloat && d15.Loc == LocReg {
				d16 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d15.Reg}
				ctx.BindReg(d15.Reg, &d16)
				ctx.BindReg(d15.Reg, &d16)
			} else if d15.Type == tagFloat && d15.Loc == LocRegPair {
				ctx.FreeReg(d15.Reg)
				d16 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d15.Reg2}
				ctx.BindReg(d15.Reg2, &d16)
				ctx.BindReg(d15.Reg2, &d16)
			} else {
				d16 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d15}, 1)
				d16.Type = tagFloat
				ctx.BindReg(d16.Reg, &d16)
			}
			ctx.FreeDesc(&d15)
			ctx.EnsureDesc(&d16)
			var d17 JITValueDesc
			if d16.Loc == LocImm {
				d17 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d16.Imm.Float() < 0)}
			} else {
				r1 := ctx.AllocRegExcept(d16.Reg)
				ctx.EmitMovRegImm64(RegR11, uint64(0))
				ctx.EmitCmpFloat64Setcc(r1, d16.Reg, RegR11, CcL)
				d17 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r1}
				ctx.BindReg(r1, &d17)
			}
			d18 = d17
			ctx.EnsureDesc(&d18)
			if d18.Loc != LocImm && d18.Loc != LocReg {
				panic("jit: If condition is neither LocImm nor LocReg")
			}
			if d18.Loc == LocImm {
				if d18.Imm.Bool() {
			ps19 := PhiState{General: ps.General}
			ps19.OverlayValues = make([]JITValueDesc, 19)
			ps19.OverlayValues[0] = d0
			ps19.OverlayValues[1] = d1
			ps19.OverlayValues[2] = d2
			ps19.OverlayValues[3] = d3
			ps19.OverlayValues[4] = d4
			ps19.OverlayValues[15] = d15
			ps19.OverlayValues[16] = d16
			ps19.OverlayValues[17] = d17
			ps19.OverlayValues[18] = d18
					return bbs[3].RenderPS(ps19)
				}
			ctx.EnsureDesc(&d16)
			if d16.Loc == LocReg {
				ctx.ProtectReg(d16.Reg)
			} else if d16.Loc == LocRegPair {
				ctx.ProtectReg(d16.Reg)
				ctx.ProtectReg(d16.Reg2)
			}
			d20 = d16
			if d20.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, int32(bbs[4].PhiBase)+int32(0))
			if d16.Loc == LocReg {
				ctx.UnprotectReg(d16.Reg)
			} else if d16.Loc == LocRegPair {
				ctx.UnprotectReg(d16.Reg)
				ctx.UnprotectReg(d16.Reg2)
			}
			ps21 := PhiState{General: ps.General}
			ps21.OverlayValues = make([]JITValueDesc, 21)
			ps21.OverlayValues[0] = d0
			ps21.OverlayValues[1] = d1
			ps21.OverlayValues[2] = d2
			ps21.OverlayValues[3] = d3
			ps21.OverlayValues[4] = d4
			ps21.OverlayValues[15] = d15
			ps21.OverlayValues[16] = d16
			ps21.OverlayValues[17] = d17
			ps21.OverlayValues[18] = d18
			ps21.OverlayValues[20] = d20
			ps21.PhiValues = make([]JITValueDesc, 1)
			d22 = d16
			ps21.PhiValues[0] = d22
				return bbs[4].RenderPS(ps21)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl11 := ctx.ReserveLabel()
			lbl12 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d18.Reg, 0)
			ctx.EmitJcc(CcNE, lbl11)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl11)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl12)
			ctx.EnsureDesc(&d16)
			if d16.Loc == LocReg {
				ctx.ProtectReg(d16.Reg)
			} else if d16.Loc == LocRegPair {
				ctx.ProtectReg(d16.Reg)
				ctx.ProtectReg(d16.Reg2)
			}
			d23 = d16
			if d23.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			ctx.EmitStoreToStack(d23, int32(bbs[4].PhiBase)+int32(0))
			if d16.Loc == LocReg {
				ctx.UnprotectReg(d16.Reg)
			} else if d16.Loc == LocRegPair {
				ctx.UnprotectReg(d16.Reg)
				ctx.UnprotectReg(d16.Reg2)
			}
			ctx.EmitJmp(lbl5)
			ps24 := PhiState{General: true}
			ps24.OverlayValues = make([]JITValueDesc, 24)
			ps24.OverlayValues[0] = d0
			ps24.OverlayValues[1] = d1
			ps24.OverlayValues[2] = d2
			ps24.OverlayValues[3] = d3
			ps24.OverlayValues[4] = d4
			ps24.OverlayValues[15] = d15
			ps24.OverlayValues[16] = d16
			ps24.OverlayValues[17] = d17
			ps24.OverlayValues[18] = d18
			ps24.OverlayValues[20] = d20
			ps24.OverlayValues[22] = d22
			ps24.OverlayValues[23] = d23
			ps25 := PhiState{General: true}
			ps25.OverlayValues = make([]JITValueDesc, 24)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[15] = d15
			ps25.OverlayValues[16] = d16
			ps25.OverlayValues[17] = d17
			ps25.OverlayValues[18] = d18
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[22] = d22
			ps25.OverlayValues[23] = d23
			ps25.PhiValues = make([]JITValueDesc, 1)
			d26 = d16
			ps25.PhiValues[0] = d26
			snap27 := d0
			snap28 := d1
			snap29 := d2
			snap30 := d3
			snap31 := d4
			snap32 := d15
			snap33 := d16
			snap34 := d17
			snap35 := d18
			snap36 := d20
			snap37 := d22
			snap38 := d23
			snap39 := d26
			alloc40 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps25)
			}
			ctx.RestoreAllocState(alloc40)
			d0 = snap27
			d1 = snap28
			d2 = snap29
			d3 = snap30
			d4 = snap31
			d15 = snap32
			d16 = snap33
			d17 = snap34
			d18 = snap35
			d20 = snap36
			d22 = snap37
			d23 = snap38
			d26 = snap39
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps24)
			}
			return result
			ctx.FreeDesc(&d17)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
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
			ctx.EnsureDesc(&d16)
			var d41 JITValueDesc
			if d16.Loc == LocImm {
				if d16.Type == tagFloat {
					d41 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(-d16.Imm.Float())}
				} else {
					d41 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-d16.Imm.Int())}
				}
			} else {
				if d16.Type == tagFloat {
					r2 := ctx.AllocRegExcept(d16.Reg)
					ctx.EmitMovRegImm64(r2, 0)
					ctx.EmitSubFloat64(r2, d16.Reg)
					d41 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: r2}
					ctx.BindReg(r2, &d41)
				} else {
					r3 := ctx.AllocRegExcept(d16.Reg)
					ctx.EmitMovRegImm64(r3, 0)
					ctx.EmitSubInt64(r3, d16.Reg)
					d41 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
					ctx.BindReg(r3, &d41)
				}
			}
			ctx.EnsureDesc(&d41)
			if d41.Loc == LocReg {
				ctx.ProtectReg(d41.Reg)
			} else if d41.Loc == LocRegPair {
				ctx.ProtectReg(d41.Reg)
				ctx.ProtectReg(d41.Reg2)
			}
			d42 = d41
			if d42.Loc == LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d42)
			ctx.EmitStoreToStack(d42, int32(bbs[4].PhiBase)+int32(0))
			if d41.Loc == LocReg {
				ctx.UnprotectReg(d41.Reg)
			} else if d41.Loc == LocRegPair {
				ctx.UnprotectReg(d41.Reg)
				ctx.UnprotectReg(d41.Reg2)
			}
			ps43 := PhiState{General: ps.General}
			ps43.OverlayValues = make([]JITValueDesc, 43)
			ps43.OverlayValues[0] = d0
			ps43.OverlayValues[1] = d1
			ps43.OverlayValues[2] = d2
			ps43.OverlayValues[3] = d3
			ps43.OverlayValues[4] = d4
			ps43.OverlayValues[15] = d15
			ps43.OverlayValues[16] = d16
			ps43.OverlayValues[17] = d17
			ps43.OverlayValues[18] = d18
			ps43.OverlayValues[20] = d20
			ps43.OverlayValues[22] = d22
			ps43.OverlayValues[23] = d23
			ps43.OverlayValues[26] = d26
			ps43.OverlayValues[41] = d41
			ps43.OverlayValues[42] = d42
			ps43.PhiValues = make([]JITValueDesc, 1)
			d44 = d41
			ps43.PhiValues[0] = d44
			if ps43.General && bbs[4].Rendered {
				ctx.EmitJmp(lbl5)
				return result
			}
			return bbs[4].RenderPS(ps43)
			return result
			}
			bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d45 := ps.PhiValues[0]
					ctx.EnsureDesc(&d45)
					ctx.EmitStoreToStack(d45, int32(bbs[4].PhiBase)+int32(0))
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
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
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
				d45 = ps.OverlayValues[45]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
				d0 = ps.PhiValues[0]
			}
			ctx.ReclaimUntrackedRegs()
			d46 = args[0]
			d46.ID = 0
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d46)
			if d46.Loc == LocImm {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d46.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				if d46.Imm.GetTag() == tagBool {
					ctx.EmitMakeBool(tmpPair, d46)
				} else if d46.Imm.GetTag() == tagInt {
					ctx.EmitMakeInt(tmpPair, d46)
				} else if d46.Imm.GetTag() == tagFloat {
					ctx.EmitMakeFloat(tmpPair, d46)
				} else if d46.Imm.GetTag() == tagNil {
					ctx.EmitMakeNil(tmpPair)
				} else {
					ptrWord, auxWord := d46.Imm.RawWords()
					ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
					ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
				}
				d46 = tmpPair
			} else if d46.Loc == LocReg {
				tmpPair := JITValueDesc{Loc: LocRegPair, Type: d46.Type, Reg: ctx.AllocRegExcept(d46.Reg), Reg2: ctx.AllocRegExcept(d46.Reg)}
				switch d46.Type {
				case tagBool:
					ctx.EmitMakeBool(tmpPair, d46)
				case tagInt:
					ctx.EmitMakeInt(tmpPair, d46)
				case tagFloat:
					ctx.EmitMakeFloat(tmpPair, d46)
				default:
					panic("jit: generic call arg scalar type unknown for 2-word value")
				}
				ctx.FreeDesc(&d46)
				d46 = tmpPair
			}
			if d46.Loc != LocRegPair && d46.Loc != LocStackPair {
				panic("jit: generic call arg expects 2-word value (ToInt arg0)")
			}
			d47 = ctx.EmitGoCallScalar(GoFuncAddr(ToInt), []JITValueDesc{d46}, 1)
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d48 JITValueDesc
			if d0.Loc == LocImm {
				d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d0.Imm.Float()))}
			} else {
				r4 := ctx.AllocReg()
				ctx.EmitCvtFloatBitsToInt64(r4, d0.Reg)
				d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
				ctx.BindReg(r4, &d48)
			}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			var d49 JITValueDesc
			if d47.Loc == LocImm && d48.Loc == LocImm {
				d49 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d47.Imm.Int() == d48.Imm.Int())}
			} else if d48.Loc == LocImm {
				r5 := ctx.AllocReg()
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d47.Reg, int32(d48.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(RegR11, uint64(d48.Imm.Int()))
					ctx.EmitCmpInt64(d47.Reg, RegR11)
				}
				ctx.EmitSetcc(r5, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r5}
				ctx.BindReg(r5, &d49)
			} else if d47.Loc == LocImm {
				r6 := ctx.AllocReg()
				ctx.EmitMovRegImm64(RegR11, uint64(d47.Imm.Int()))
				ctx.EmitCmpInt64(RegR11, d48.Reg)
				ctx.EmitSetcc(r6, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r6}
				ctx.BindReg(r6, &d49)
			} else {
				r7 := ctx.AllocReg()
				ctx.EmitCmpInt64(d47.Reg, d48.Reg)
				ctx.EmitSetcc(r7, CcE)
				d49 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
				ctx.BindReg(r7, &d49)
			}
			ctx.FreeDesc(&d47)
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
			ps51.OverlayValues[15] = d15
			ps51.OverlayValues[16] = d16
			ps51.OverlayValues[17] = d17
			ps51.OverlayValues[18] = d18
			ps51.OverlayValues[20] = d20
			ps51.OverlayValues[22] = d22
			ps51.OverlayValues[23] = d23
			ps51.OverlayValues[26] = d26
			ps51.OverlayValues[41] = d41
			ps51.OverlayValues[42] = d42
			ps51.OverlayValues[44] = d44
			ps51.OverlayValues[45] = d45
			ps51.OverlayValues[46] = d46
			ps51.OverlayValues[47] = d47
			ps51.OverlayValues[48] = d48
			ps51.OverlayValues[49] = d49
			ps51.OverlayValues[50] = d50
					return bbs[7].RenderPS(ps51)
				}
			ps52 := PhiState{General: ps.General}
			ps52.OverlayValues = make([]JITValueDesc, 51)
			ps52.OverlayValues[0] = d0
			ps52.OverlayValues[1] = d1
			ps52.OverlayValues[2] = d2
			ps52.OverlayValues[3] = d3
			ps52.OverlayValues[4] = d4
			ps52.OverlayValues[15] = d15
			ps52.OverlayValues[16] = d16
			ps52.OverlayValues[17] = d17
			ps52.OverlayValues[18] = d18
			ps52.OverlayValues[20] = d20
			ps52.OverlayValues[22] = d22
			ps52.OverlayValues[23] = d23
			ps52.OverlayValues[26] = d26
			ps52.OverlayValues[41] = d41
			ps52.OverlayValues[42] = d42
			ps52.OverlayValues[44] = d44
			ps52.OverlayValues[45] = d45
			ps52.OverlayValues[46] = d46
			ps52.OverlayValues[47] = d47
			ps52.OverlayValues[48] = d48
			ps52.OverlayValues[49] = d49
			ps52.OverlayValues[50] = d50
				return bbs[6].RenderPS(ps52)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
					d53 := ps.PhiValues[0]
					ctx.EnsureDesc(&d53)
					ctx.EmitStoreToStack(d53, int32(bbs[4].PhiBase)+int32(0))
				}
				ps.General = true
				return bbs[4].RenderPS(ps)
			}
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d50.Reg, 0)
			ctx.EmitJcc(CcNE, lbl13)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl7)
			ps54 := PhiState{General: true}
			ps54.OverlayValues = make([]JITValueDesc, 54)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
			ps54.OverlayValues[2] = d2
			ps54.OverlayValues[3] = d3
			ps54.OverlayValues[4] = d4
			ps54.OverlayValues[15] = d15
			ps54.OverlayValues[16] = d16
			ps54.OverlayValues[17] = d17
			ps54.OverlayValues[18] = d18
			ps54.OverlayValues[20] = d20
			ps54.OverlayValues[22] = d22
			ps54.OverlayValues[23] = d23
			ps54.OverlayValues[26] = d26
			ps54.OverlayValues[41] = d41
			ps54.OverlayValues[42] = d42
			ps54.OverlayValues[44] = d44
			ps54.OverlayValues[45] = d45
			ps54.OverlayValues[46] = d46
			ps54.OverlayValues[47] = d47
			ps54.OverlayValues[48] = d48
			ps54.OverlayValues[49] = d49
			ps54.OverlayValues[50] = d50
			ps54.OverlayValues[53] = d53
			ps55 := PhiState{General: true}
			ps55.OverlayValues = make([]JITValueDesc, 54)
			ps55.OverlayValues[0] = d0
			ps55.OverlayValues[1] = d1
			ps55.OverlayValues[2] = d2
			ps55.OverlayValues[3] = d3
			ps55.OverlayValues[4] = d4
			ps55.OverlayValues[15] = d15
			ps55.OverlayValues[16] = d16
			ps55.OverlayValues[17] = d17
			ps55.OverlayValues[18] = d18
			ps55.OverlayValues[20] = d20
			ps55.OverlayValues[22] = d22
			ps55.OverlayValues[23] = d23
			ps55.OverlayValues[26] = d26
			ps55.OverlayValues[41] = d41
			ps55.OverlayValues[42] = d42
			ps55.OverlayValues[44] = d44
			ps55.OverlayValues[45] = d45
			ps55.OverlayValues[46] = d46
			ps55.OverlayValues[47] = d47
			ps55.OverlayValues[48] = d48
			ps55.OverlayValues[49] = d49
			ps55.OverlayValues[50] = d50
			ps55.OverlayValues[53] = d53
			snap56 := d0
			snap57 := d1
			snap58 := d2
			snap59 := d3
			snap60 := d4
			snap61 := d15
			snap62 := d16
			snap63 := d17
			snap64 := d18
			snap65 := d20
			snap66 := d22
			snap67 := d23
			snap68 := d26
			snap69 := d41
			snap70 := d42
			snap71 := d44
			snap72 := d45
			snap73 := d46
			snap74 := d47
			snap75 := d48
			snap76 := d49
			snap77 := d50
			snap78 := d53
			alloc79 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps55)
			}
			ctx.RestoreAllocState(alloc79)
			d0 = snap56
			d1 = snap57
			d2 = snap58
			d3 = snap59
			d4 = snap60
			d15 = snap61
			d16 = snap62
			d17 = snap63
			d18 = snap64
			d20 = snap65
			d22 = snap66
			d23 = snap67
			d26 = snap68
			d41 = snap69
			d42 = snap70
			d44 = snap71
			d45 = snap72
			d46 = snap73
			d47 = snap74
			d48 = snap75
			d49 = snap76
			d50 = snap77
			d53 = snap78
			if !bbs[7].Rendered {
				return bbs[7].RenderPS(ps54)
			}
			return result
			ctx.FreeDesc(&d49)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
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
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
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
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
				d53 = ps.OverlayValues[53]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d80 JITValueDesc
			if d0.Loc == LocImm {
				d80 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d0.Imm.Float()))}
			} else {
				r8 := ctx.AllocReg()
				ctx.EmitCvtFloatBitsToInt64(r8, d0.Reg)
				d80 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r8}
				ctx.BindReg(r8, &d80)
			}
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d80)
			ctx.EmitMakeInt(result, d80)
			if d80.Loc == LocReg { ctx.FreeReg(d80.Reg) }
			result.Type = tagInt
			ctx.EmitJmp(lbl0)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
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
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
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
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
				d80 = ps.OverlayValues[80]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			ctx.EmitMakeFloat(result, d0)
			if d0.Loc == LocReg { ctx.FreeReg(d0.Reg) }
			result.Type = tagFloat
			ctx.EmitJmp(lbl0)
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
			d0 = JITValueDesc{Loc: LocStack, Type: tagFloat, StackOff: int32(0)}
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
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
				d20 = ps.OverlayValues[20]
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
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
				d42 = ps.OverlayValues[42]
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
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != LocNone {
				d80 = ps.OverlayValues[80]
			}
			ctx.ReclaimUntrackedRegs()
			d81 = args[0]
			d81.ID = 0
			var d82 JITValueDesc
			if d81.Loc == LocImm {
				d82 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d81.Imm.Float())}
			} else if d81.Type == tagFloat && d81.Loc == LocReg {
				d82 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d81.Reg}
				ctx.BindReg(d81.Reg, &d82)
				ctx.BindReg(d81.Reg, &d82)
			} else if d81.Type == tagFloat && d81.Loc == LocRegPair {
				ctx.FreeReg(d81.Reg)
				d82 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d81.Reg2}
				ctx.BindReg(d81.Reg2, &d82)
				ctx.BindReg(d81.Reg2, &d82)
			} else {
				d82 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d81}, 1)
				d82.Type = tagFloat
				ctx.BindReg(d82.Reg, &d82)
			}
			ctx.FreeDesc(&d81)
			ctx.EnsureDesc(&d82)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d82)
			ctx.EnsureDesc(&d0)
			var d83 JITValueDesc
			if d82.Loc == LocImm && d0.Loc == LocImm {
				d83 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d82.Imm.Float() == d0.Imm.Float())}
			} else if d0.Loc == LocImm {
				r9 := ctx.AllocReg()
				_, yBits := d0.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, yBits)
				ctx.EmitCmpFloat64Setcc(r9, d82.Reg, RegR11, CcE)
				d83 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
				ctx.BindReg(r9, &d83)
			} else if d82.Loc == LocImm {
				r10 := ctx.AllocRegExcept(d0.Reg)
				_, xBits := d82.Imm.RawWords()
				ctx.EmitMovRegImm64(RegR11, xBits)
				ctx.EmitCmpFloat64Setcc(r10, RegR11, d0.Reg, CcE)
				d83 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
				ctx.BindReg(r10, &d83)
			} else {
				r11 := ctx.AllocRegExcept(d82.Reg, d0.Reg)
				ctx.EmitCmpFloat64Setcc(r11, d82.Reg, d0.Reg, CcE)
				d83 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
				ctx.BindReg(r11, &d83)
			}
			ctx.FreeDesc(&d82)
			d84 = d83
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
			ps85.OverlayValues[15] = d15
			ps85.OverlayValues[16] = d16
			ps85.OverlayValues[17] = d17
			ps85.OverlayValues[18] = d18
			ps85.OverlayValues[20] = d20
			ps85.OverlayValues[22] = d22
			ps85.OverlayValues[23] = d23
			ps85.OverlayValues[26] = d26
			ps85.OverlayValues[41] = d41
			ps85.OverlayValues[42] = d42
			ps85.OverlayValues[44] = d44
			ps85.OverlayValues[45] = d45
			ps85.OverlayValues[46] = d46
			ps85.OverlayValues[47] = d47
			ps85.OverlayValues[48] = d48
			ps85.OverlayValues[49] = d49
			ps85.OverlayValues[50] = d50
			ps85.OverlayValues[53] = d53
			ps85.OverlayValues[80] = d80
			ps85.OverlayValues[81] = d81
			ps85.OverlayValues[82] = d82
			ps85.OverlayValues[83] = d83
			ps85.OverlayValues[84] = d84
					return bbs[5].RenderPS(ps85)
				}
			ps86 := PhiState{General: ps.General}
			ps86.OverlayValues = make([]JITValueDesc, 85)
			ps86.OverlayValues[0] = d0
			ps86.OverlayValues[1] = d1
			ps86.OverlayValues[2] = d2
			ps86.OverlayValues[3] = d3
			ps86.OverlayValues[4] = d4
			ps86.OverlayValues[15] = d15
			ps86.OverlayValues[16] = d16
			ps86.OverlayValues[17] = d17
			ps86.OverlayValues[18] = d18
			ps86.OverlayValues[20] = d20
			ps86.OverlayValues[22] = d22
			ps86.OverlayValues[23] = d23
			ps86.OverlayValues[26] = d26
			ps86.OverlayValues[41] = d41
			ps86.OverlayValues[42] = d42
			ps86.OverlayValues[44] = d44
			ps86.OverlayValues[45] = d45
			ps86.OverlayValues[46] = d46
			ps86.OverlayValues[47] = d47
			ps86.OverlayValues[48] = d48
			ps86.OverlayValues[49] = d49
			ps86.OverlayValues[50] = d50
			ps86.OverlayValues[53] = d53
			ps86.OverlayValues[80] = d80
			ps86.OverlayValues[81] = d81
			ps86.OverlayValues[82] = d82
			ps86.OverlayValues[83] = d83
			ps86.OverlayValues[84] = d84
				return bbs[6].RenderPS(ps86)
			}
			if !ps.General {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d84.Reg, 0)
			ctx.EmitJcc(CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl7)
			ps87 := PhiState{General: true}
			ps87.OverlayValues = make([]JITValueDesc, 85)
			ps87.OverlayValues[0] = d0
			ps87.OverlayValues[1] = d1
			ps87.OverlayValues[2] = d2
			ps87.OverlayValues[3] = d3
			ps87.OverlayValues[4] = d4
			ps87.OverlayValues[15] = d15
			ps87.OverlayValues[16] = d16
			ps87.OverlayValues[17] = d17
			ps87.OverlayValues[18] = d18
			ps87.OverlayValues[20] = d20
			ps87.OverlayValues[22] = d22
			ps87.OverlayValues[23] = d23
			ps87.OverlayValues[26] = d26
			ps87.OverlayValues[41] = d41
			ps87.OverlayValues[42] = d42
			ps87.OverlayValues[44] = d44
			ps87.OverlayValues[45] = d45
			ps87.OverlayValues[46] = d46
			ps87.OverlayValues[47] = d47
			ps87.OverlayValues[48] = d48
			ps87.OverlayValues[49] = d49
			ps87.OverlayValues[50] = d50
			ps87.OverlayValues[53] = d53
			ps87.OverlayValues[80] = d80
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
			ps88.OverlayValues[15] = d15
			ps88.OverlayValues[16] = d16
			ps88.OverlayValues[17] = d17
			ps88.OverlayValues[18] = d18
			ps88.OverlayValues[20] = d20
			ps88.OverlayValues[22] = d22
			ps88.OverlayValues[23] = d23
			ps88.OverlayValues[26] = d26
			ps88.OverlayValues[41] = d41
			ps88.OverlayValues[42] = d42
			ps88.OverlayValues[44] = d44
			ps88.OverlayValues[45] = d45
			ps88.OverlayValues[46] = d46
			ps88.OverlayValues[47] = d47
			ps88.OverlayValues[48] = d48
			ps88.OverlayValues[49] = d49
			ps88.OverlayValues[50] = d50
			ps88.OverlayValues[53] = d53
			ps88.OverlayValues[80] = d80
			ps88.OverlayValues[81] = d81
			ps88.OverlayValues[82] = d82
			ps88.OverlayValues[83] = d83
			ps88.OverlayValues[84] = d84
			snap89 := d0
			snap90 := d1
			snap91 := d2
			snap92 := d3
			snap93 := d4
			snap94 := d15
			snap95 := d16
			snap96 := d17
			snap97 := d18
			snap98 := d20
			snap99 := d22
			snap100 := d23
			snap101 := d26
			snap102 := d41
			snap103 := d42
			snap104 := d44
			snap105 := d45
			snap106 := d46
			snap107 := d47
			snap108 := d48
			snap109 := d49
			snap110 := d50
			snap111 := d53
			snap112 := d80
			snap113 := d81
			snap114 := d82
			snap115 := d83
			snap116 := d84
			alloc117 := ctx.SnapshotAllocState()
			if !bbs[6].Rendered {
				bbs[6].RenderPS(ps88)
			}
			ctx.RestoreAllocState(alloc117)
			d0 = snap89
			d1 = snap90
			d2 = snap91
			d3 = snap92
			d4 = snap93
			d15 = snap94
			d16 = snap95
			d17 = snap96
			d18 = snap97
			d20 = snap98
			d22 = snap99
			d23 = snap100
			d26 = snap101
			d41 = snap102
			d42 = snap103
			d44 = snap104
			d45 = snap105
			d46 = snap106
			d47 = snap107
			d48 = snap108
			d49 = snap109
			d50 = snap110
			d53 = snap111
			d80 = snap112
			d81 = snap113
			d82 = snap114
			d83 = snap115
			d84 = snap116
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps87)
			}
			return result
			ctx.FreeDesc(&d83)
			return result
			}
			argPinned118 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned118 = append(argPinned118, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned118 = append(argPinned118, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned118 = append(argPinned118, ai.Reg2)
					}
				}
			}
			ps119 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps119)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			ctx.PatchInt32(r0, int32(16))
			ctx.EmitAddRSP32(int32(16))
			for _, r := range argPinned118 {
				ctx.UnprotectReg(r)
			}
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
			var d13 JITValueDesc
			_ = d13
			var d14 JITValueDesc
			_ = d14
			var d15 JITValueDesc
			_ = d15
			var d16 JITValueDesc
			_ = d16
			var d30 JITValueDesc
			_ = d30
			var d31 JITValueDesc
			_ = d31
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
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
			ctx.ReclaimUntrackedRegs()
			d13 = args[0]
			d13.ID = 0
			var d14 JITValueDesc
			if d13.Loc == LocImm {
				d14 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(d13.Imm.Float())}
			} else if d13.Type == tagFloat && d13.Loc == LocReg {
				d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d13.Reg}
				ctx.BindReg(d13.Reg, &d14)
				ctx.BindReg(d13.Reg, &d14)
			} else if d13.Type == tagFloat && d13.Loc == LocRegPair {
				ctx.FreeReg(d13.Reg)
				d14 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d13.Reg2}
				ctx.BindReg(d13.Reg2, &d14)
				ctx.BindReg(d13.Reg2, &d14)
			} else {
				d14 = ctx.EmitGoCallScalar(GoFuncAddr(JITScmerToFloatBits), []JITValueDesc{d13}, 1)
				d14.Type = tagFloat
				ctx.BindReg(d14.Reg, &d14)
			}
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			var d15 JITValueDesc
			if d14.Loc == LocImm {
				d15 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d14.Imm.Float() < 0)}
			} else {
				r0 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegImm64(RegR11, uint64(0))
				ctx.EmitCmpFloat64Setcc(r0, d14.Reg, RegR11, CcL)
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
			ps17 := PhiState{General: ps.General}
			ps17.OverlayValues = make([]JITValueDesc, 17)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
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
			ps20.OverlayValues[13] = d13
			ps20.OverlayValues[14] = d14
			ps20.OverlayValues[15] = d15
			ps20.OverlayValues[16] = d16
			snap21 := d0
			snap22 := d1
			snap23 := d2
			snap24 := d3
			snap25 := d13
			snap26 := d14
			snap27 := d15
			snap28 := d16
			alloc29 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps20)
			}
			ctx.RestoreAllocState(alloc29)
			d0 = snap21
			d1 = snap22
			d2 = snap23
			d3 = snap24
			d13 = snap25
			d14 = snap26
			d15 = snap27
			d16 = snap28
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
			ctx.EmitMakeNil(result)
			result.Type = tagNil
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d14)
			var d30 JITValueDesc
			if d14.Loc == LocImm {
				d30 = JITValueDesc{Loc: LocImm, Type: tagFloat, Imm: NewFloat(math.Sqrt(d14.Imm.Float()))}
			} else {
				ctx.EnsureDesc(&d14)
				var d31 JITValueDesc
				if d14.Loc == LocRegPair {
					ctx.FreeReg(d14.Reg)
					d31 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg2}
					ctx.BindReg(d14.Reg2, &d31)
					ctx.BindReg(d14.Reg2, &d31)
				} else {
					d31 = JITValueDesc{Loc: LocReg, Type: tagFloat, Reg: d14.Reg}
					ctx.BindReg(d14.Reg, &d31)
					ctx.BindReg(d14.Reg, &d31)
				}
				d30 = ctx.EmitGoCallScalar(GoFuncAddr(JITSqrtBits), []JITValueDesc{d31}, 1)
				d30.Type = tagFloat
				ctx.BindReg(d30.Reg, &d30)
			}
			ctx.FreeDesc(&d14)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			ctx.EmitMakeFloat(result, d30)
			if d30.Loc == LocReg { ctx.FreeReg(d30.Reg) }
			result.Type = tagFloat
			ctx.EmitJmp(lbl0)
			return result
			}
			argPinned32 := make([]Reg, 0, len(args)*2)
			seenArgRegs := make(map[Reg]bool)
			for _, ai := range args {
				if ai.Loc == LocReg {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned32 = append(argPinned32, ai.Reg)
					}
				} else if ai.Loc == LocRegPair {
					if !seenArgRegs[ai.Reg] {
						ctx.ProtectReg(ai.Reg)
						seenArgRegs[ai.Reg] = true
						argPinned32 = append(argPinned32, ai.Reg)
					}
					if !seenArgRegs[ai.Reg2] {
						ctx.ProtectReg(ai.Reg2)
						seenArgRegs[ai.Reg2] = true
						argPinned32 = append(argPinned32, ai.Reg2)
					}
				}
			}
			ps33 := PhiState{General: true}
			_ = bbs[0].RenderPS(ps33)
			ctx.MarkLabel(lbl0)
			ctx.ResolveFixups()
			for _, r := range argPinned32 {
				ctx.UnprotectReg(r)
			}
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
		nil /* TODO: Slice on non-desc: slice t0[:] */, /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */ /* TODO: Slice on non-desc: slice t0[:] */
	})
}
