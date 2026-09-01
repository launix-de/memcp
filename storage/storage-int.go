/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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
package storage

import "io"
import "fmt"
import "unsafe"
import "math/bits"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

type StorageInt struct {
	chunk   []uint64
	bitsize uint8
	offset  int64
	max     int64  // only of statistic use
	count   uint64 // only stored for serialization purposes
	hasNull bool
	null    uint64 // which value is null
}

// storageIntVersion is the current binary format version for StorageInt.
// Increment this constant and add a new deserializeIntV* helper whenever the
// layout after the magic byte changes.  Never delete old helpers.
const storageIntVersion = 0

// StorageInt binary layout (magic byte 10 consumed by shard loader):
//
//	[version uint8]        ← first byte read by Deserialize; was padding in v0
//	[bitsize uint8]
//	[hasNull uint8]
//	[pad uint32]
//	[chunkcount uint64]
//	[count uint64]
//	[offset int64]
//	[null uint64]
//	[chunk data: chunkcount × 8 bytes]
//
// Version history:
//
//	0 (current): layout as above; the version byte was previously a uint8(0)
//	             padding byte, so all pre-versioning data reads correctly as v0.

func (s *StorageInt) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	var d0 scm.JITValueDesc
	_ = d0
	var d1 scm.JITValueDesc
	_ = d1
	var d2 scm.JITValueDesc
	_ = d2
	var d11 scm.JITValueDesc
	_ = d11
	var d12 scm.JITValueDesc
	_ = d12
	var d13 scm.JITValueDesc
	_ = d13
	var d14 scm.JITValueDesc
	_ = d14
	var d15 scm.JITValueDesc
	_ = d15
	var d16 scm.JITValueDesc
	_ = d16
	var d17 scm.JITValueDesc
	_ = d17
	var d18 scm.JITValueDesc
	_ = d18
	var d19 scm.JITValueDesc
	_ = d19
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	var idxInt scm.JITValueDesc
	if idx.Loc == scm.LocImm {
		idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
	} else if idx.Loc == scm.LocRegPair {
		ctx.FreeReg(idx.Reg)
		idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
		ctx.BindReg(idx.Reg2, &idxInt)
	} else {
		idxInt = idx
	}
	if idxInt.Loc == scm.LocImm {
		idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.EnsureDesc(&idxInt)
		if idxInt.Loc != scm.LocReg {
			panic("jit: idxInt not in register")
		}
		ctx.EmitShlRegImm8(idxInt.Reg, 32)
		ctx.EmitShrRegImm8(idxInt.Reg, 32)
		ctx.BindReg(idxInt.Reg, &idxInt)
	}
	idxPinned := idxInt.Loc == scm.LocReg
	idxPinnedReg := idxInt.Reg
	if idxPinned {
		ctx.ProtectReg(idxPinnedReg)
	}
	var bbs [4]scm.BBDescriptor
	if result.Loc == scm.LocAny {
		result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.BindReg(result.Reg, &result)
		ctx.BindReg(result.Reg2, &result)
	}
	r0 := ctx.AllocReg()
	r1 := ctx.AllocRegExcept(r0)
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
	bbs[0].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&thisptr)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		if idxInt.Loc == scm.LocRegPair || idxInt.Loc == scm.LocStackPair || idxInt.Loc == scm.LocRegTriple || idxInt.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&idxInt)
		d0 = ctx.EmitGoCallScalar(scm.GoFuncAddr((*StorageInt).GetValueUInt), []scm.JITValueDesc{thisptr, idxInt}, 1)
		d0.NoHeapPointer = true
		ctx.BindReg(d0.Reg, &d0)
		ctx.StabilizeDescForControlFlow(&d0)
		ctx.FreeDesc(&idxInt)
		var d1 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
			r2 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r2, fieldAddr)
			d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			ctx.BindReg(r2, &d1)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r3, thisptr.Reg, off)
			d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d1)
		}
		d2 = d1
		ctx.EnsureDesc(&d2)
		if d2.Loc != scm.LocImm && d2.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d2.Loc == scm.LocImm {
			if d2.Imm.Bool() {
				if ps.General {
				}
				ps3 := scm.PhiState{General: ps.General}
				ps3.OverlayValues = make([]scm.JITValueDesc, 3)
				ps3.OverlayValues[0] = d0
				ps3.OverlayValues[1] = d1
				ps3.OverlayValues[2] = d2
				return bbs[3].RenderPS(ps3)
			}
			if ps.General {
			}
			ps4 := scm.PhiState{General: ps.General}
			ps4.OverlayValues = make([]scm.JITValueDesc, 3)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			return bbs[2].RenderPS(ps4)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl5 := ctx.ReserveLabel()
		lbl6 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d2.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl5)
		ctx.EmitJmp(lbl6)
		ctx.MarkLabel(lbl5)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl6)
		ctx.EmitJmp(lbl3)
		ps5 := scm.PhiState{General: true}
		ps5.OverlayValues = make([]scm.JITValueDesc, 3)
		ps5.OverlayValues[0] = d0
		ps5.OverlayValues[1] = d1
		ps5.OverlayValues[2] = d2
		ps6 := scm.PhiState{General: true}
		ps6.OverlayValues = make([]scm.JITValueDesc, 3)
		ps6.OverlayValues[0] = d0
		ps6.OverlayValues[1] = d1
		ps6.OverlayValues[2] = d2
		snap7 := d0
		snap8 := d1
		snap9 := d2
		alloc10 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps6)
		}
		ctx.RestoreAllocState(alloc10)
		d0 = snap7
		d1 = snap8
		d2 = snap9
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps5)
		}
		return result
		return result
	}
	bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
		if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
			d0 = ps.OverlayValues[0]
		}
		if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		ctx.ReclaimUntrackedRegs()
		d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d12 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d12)
		ctx.BindReg(r1, &d12)
		ctx.EnsureDesc(&d11)
		if d11.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d11, &d12)
		} else {
			switch d11.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d12, d11)
			case scm.TagInt:
				ctx.EmitMakeInt(d12, d11)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d12, d11)
			case scm.TagNil:
				ctx.EmitMakeNil(d12)
			default:
				ctx.EmitMovPairToResult(&d11, &d12)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
		if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
			d0 = ps.OverlayValues[0]
		}
		if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d0)
		var d13 scm.JITValueDesc
		if d0.Loc == scm.LocImm {
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d0.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d0.Reg)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d13)
		}
		var d14 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			r5 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r5, fieldAddr)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
			ctx.BindReg(r5, &d14)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r6 := ctx.AllocReg()
			ctx.EmitMovRegMem(r6, thisptr.Reg, off)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
			ctx.BindReg(r6, &d14)
		}
		ctx.EnsureDesc(&d13)
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d13)
		ctx.ProtectReg(d13.Reg)
		ctx.EnsureDesc(&d14)
		ctx.UnprotectReg(d13.Reg)
		var d15 scm.JITValueDesc
		if d13.Loc == scm.LocImm && d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() + d14.Imm.Int())}
		} else if d14.Loc == scm.LocImm && d14.Imm.Int() == 0 {
			r7 := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegReg(r7, d13.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d15)
		} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			ctx.BindReg(d14.Reg, &d15)
		} else if d13.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
			ctx.EmitAddInt64(scratch, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else if d14.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegReg(scratch, d13.Reg)
			if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d14.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else {
			r8 := ctx.AllocRegExcept(d13.Reg, d14.Reg)
			ctx.EmitMovRegReg(r8, d13.Reg)
			ctx.EmitAddInt64(r8, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d15)
		}
		if d15.Loc == scm.LocReg && d13.Loc == scm.LocReg && d15.Reg == d13.Reg {
			ctx.TransferReg(d13.Reg)
			d13.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d13)
		ctx.EnsureDesc(&d15)
		d16 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d16)
		ctx.BindReg(r1, &d16)
		ctx.EnsureDesc(&d15)
		ctx.EmitMakeInt(d16, d15)
		if d15.Loc == scm.LocReg {
			ctx.FreeReg(d15.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[3].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
		if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
			d0 = ps.OverlayValues[0]
		}
		if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
			d13 = ps.OverlayValues[13]
		}
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		ctx.ReclaimUntrackedRegs()
		var d17 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
			r9 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r9, fieldAddr)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
			ctx.BindReg(r9, &d17)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
			r10 := ctx.AllocReg()
			ctx.EmitMovRegMem(r10, thisptr.Reg, off)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			ctx.BindReg(r10, &d17)
		}
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d17)
		var d18 scm.JITValueDesc
		if d0.Loc == scm.LocImm && d17.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d0.Imm.Int()) == uint64(d17.Imm.Int()))}
		} else if d17.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d0.Reg)
			if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d0.Reg, int32(d17.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d17.Imm.Int()))
				ctx.EmitCmpInt64(d0.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r11, scm.CondEqual)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r11}
			ctx.BindReg(r11, &d18)
		} else if d0.Loc == scm.LocImm {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d17.Reg)
			ctx.EmitSetcc(r12, scm.CondEqual)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r12}
			ctx.BindReg(r12, &d18)
		} else {
			r13 := ctx.AllocRegExcept(d0.Reg)
			ctx.EmitCmpInt64(d0.Reg, d17.Reg)
			ctx.EmitSetcc(r13, scm.CondEqual)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r13}
			ctx.BindReg(r13, &d18)
		}
		ctx.FreeDesc(&d0)
		d19 = d18
		ctx.EnsureDesc(&d19)
		if d19.Loc != scm.LocImm && d19.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d19.Loc == scm.LocImm {
			if d19.Imm.Bool() {
				if ps.General {
				}
				ps20 := scm.PhiState{General: ps.General}
				ps20.OverlayValues = make([]scm.JITValueDesc, 20)
				ps20.OverlayValues[0] = d0
				ps20.OverlayValues[1] = d1
				ps20.OverlayValues[2] = d2
				ps20.OverlayValues[11] = d11
				ps20.OverlayValues[12] = d12
				ps20.OverlayValues[13] = d13
				ps20.OverlayValues[14] = d14
				ps20.OverlayValues[15] = d15
				ps20.OverlayValues[16] = d16
				ps20.OverlayValues[17] = d17
				ps20.OverlayValues[18] = d18
				ps20.OverlayValues[19] = d19
				return bbs[1].RenderPS(ps20)
			}
			if ps.General {
			}
			ps21 := scm.PhiState{General: ps.General}
			ps21.OverlayValues = make([]scm.JITValueDesc, 20)
			ps21.OverlayValues[0] = d0
			ps21.OverlayValues[1] = d1
			ps21.OverlayValues[2] = d2
			ps21.OverlayValues[11] = d11
			ps21.OverlayValues[12] = d12
			ps21.OverlayValues[13] = d13
			ps21.OverlayValues[14] = d14
			ps21.OverlayValues[15] = d15
			ps21.OverlayValues[16] = d16
			ps21.OverlayValues[17] = d17
			ps21.OverlayValues[18] = d18
			ps21.OverlayValues[19] = d19
			return bbs[2].RenderPS(ps21)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl7 := ctx.ReserveLabel()
		lbl8 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d19.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl7)
		ctx.EmitJmp(lbl8)
		ctx.MarkLabel(lbl7)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl8)
		ctx.EmitJmp(lbl3)
		ps22 := scm.PhiState{General: true}
		ps22.OverlayValues = make([]scm.JITValueDesc, 20)
		ps22.OverlayValues[0] = d0
		ps22.OverlayValues[1] = d1
		ps22.OverlayValues[2] = d2
		ps22.OverlayValues[11] = d11
		ps22.OverlayValues[12] = d12
		ps22.OverlayValues[13] = d13
		ps22.OverlayValues[14] = d14
		ps22.OverlayValues[15] = d15
		ps22.OverlayValues[16] = d16
		ps22.OverlayValues[17] = d17
		ps22.OverlayValues[18] = d18
		ps22.OverlayValues[19] = d19
		ps23 := scm.PhiState{General: true}
		ps23.OverlayValues = make([]scm.JITValueDesc, 20)
		ps23.OverlayValues[0] = d0
		ps23.OverlayValues[1] = d1
		ps23.OverlayValues[2] = d2
		ps23.OverlayValues[11] = d11
		ps23.OverlayValues[12] = d12
		ps23.OverlayValues[13] = d13
		ps23.OverlayValues[14] = d14
		ps23.OverlayValues[15] = d15
		ps23.OverlayValues[16] = d16
		ps23.OverlayValues[17] = d17
		ps23.OverlayValues[18] = d18
		ps23.OverlayValues[19] = d19
		snap24 := d0
		snap25 := d1
		snap26 := d2
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
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps23)
		}
		ctx.RestoreAllocState(alloc36)
		d0 = snap24
		d1 = snap25
		d2 = snap26
		d11 = snap27
		d12 = snap28
		d13 = snap29
		d14 = snap30
		d15 = snap31
		d16 = snap32
		d17 = snap33
		d18 = snap34
		d19 = snap35
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps22)
		}
		return result
		ctx.FreeDesc(&d18)
		return result
	}
	ps37 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps37)
	ctx.MarkLabel(lbl0)
	d38 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d38)
	ctx.BindReg(r1, &d38)
	ctx.EmitMovPairToResult(&d38, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
	return result
}

func (s *StorageInt) Serialize(f io.Writer) {
	var hasNull uint8
	if s.hasNull {
		hasNull = 1
	}
	binary.Write(f, binary.LittleEndian, uint8(10))                // 10 = StorageInt
	binary.Write(f, binary.LittleEndian, uint8(s.bitsize))         // len=2
	binary.Write(f, binary.LittleEndian, uint8(hasNull))           // len=3
	binary.Write(f, binary.LittleEndian, uint8(storageIntVersion)) // len=4  ← version byte (was uint8(0) pad)
	binary.Write(f, binary.LittleEndian, uint32(0))                // len=8  pad
	binary.Write(f, binary.LittleEndian, uint64(len(s.chunk)))     // chunk size so we know how many data is left
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(s.offset))
	binary.Write(f, binary.LittleEndian, uint64(s.null))
	if len(s.chunk) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.chunk[0])), 8*len(s.chunk)))
	}
}
func (s *StorageInt) Deserialize(f io.Reader) uint {
	return s.DeserializeEx(f, false)
}

func (s *StorageInt) DeserializeEx(f io.Reader, readMagicbyte bool) uint {
	var dummy8 uint8
	var dummy32 uint32
	if readMagicbyte {
		binary.Read(f, binary.LittleEndian, &dummy8)
		if dummy8 != 10 {
			panic(fmt.Sprintf("Tried to deserialize StorageInt(10) from file but found %d", dummy8))
		}
	}
	binary.Read(f, binary.LittleEndian, &s.bitsize)
	var hasNull uint8
	binary.Read(f, binary.LittleEndian, &hasNull)
	s.hasNull = hasNull != 0
	var version uint8
	binary.Read(f, binary.LittleEndian, &version) // was uint8(0) pad; now version byte
	binary.Read(f, binary.LittleEndian, &dummy32)
	switch version {
	case 0:
		return s.deserializeIntV0(f)
	default:
		panic(fmt.Sprintf("StorageInt: unknown version %d", version))
	}
}

func (s *StorageInt) deserializeIntV0(f io.Reader) uint {
	var chunkcount uint64
	binary.Read(f, binary.LittleEndian, &chunkcount)
	binary.Read(f, binary.LittleEndian, &s.count)
	binary.Read(f, binary.LittleEndian, &s.offset)
	binary.Read(f, binary.LittleEndian, &s.null)
	if chunkcount > 0 {
		rawdata := make([]byte, chunkcount*8)
		f.Read(rawdata)
		s.chunk = unsafe.Slice((*uint64)(unsafe.Pointer(&rawdata[0])), chunkcount)
	}
	return uint(s.count)
}

func (s *StorageInt) ComputeSize() uint {
	return 8*uint(len(s.chunk)) + 64 // management overhead
}

func (s *StorageInt) String() string {
	if s.hasNull {
		return fmt.Sprintf("int[%d]NULL", s.bitsize)
	} else {
		return fmt.Sprintf("int[%d]", s.bitsize)
	}
}

func (s *StorageInt) GetCachedReader() ColumnReader { return s }

func (s *StorageInt) GetValue(i uint32) scm.Scmer {
	v := s.GetValueUInt(i)
	if s.hasNull && v == s.null {
		return scm.NewNil()
	}
	return scm.NewInt(int64(v) + s.offset)
}

// SetValue overwrites a single element in the bit-packed array.
// The new value must fit within the existing [offset, offset+2^bitsize) range.
// Caller must hold the shard write lock.
func (s *StorageInt) SetValue(i uint32, value scm.Scmer) {
	var vi int64
	if value.IsNil() {
		vi = int64(s.null)
	} else {
		vi = value.Int() - s.offset
	}
	bitpos := uint(i) * uint(s.bitsize)
	mask := uint64((1<<uint(s.bitsize))-1) << (64 - uint(s.bitsize)) // bitsize ones at MSB
	v := uint64(vi) << (64 - uint(s.bitsize))
	// clear old bits then set new bits in first chunk
	shifted := mask >> (bitpos % 64)
	s.chunk[bitpos/64] = (s.chunk[bitpos/64] & ^shifted) | (v >> (bitpos % 64))
	if bitpos%64+uint(s.bitsize) > 64 {
		// spans two chunks
		shifted2 := mask << (64 - bitpos%64)
		s.chunk[bitpos/64+1] = (s.chunk[bitpos/64+1] & ^shifted2) | (v << (64 - bitpos%64))
	}
}

func (s *StorageInt) GetValueUInt(i uint32) uint64 {
	bitpos := uint(i) * uint(s.bitsize)

	v := s.chunk[bitpos/64] << (bitpos % 64) // align to leftmost position
	if bitpos%64+uint(s.bitsize) > 64 {
		v = v | s.chunk[bitpos/64+1]>>(64-bitpos%64)
	}

	return uint64(v) >> (64 - uint(s.bitsize)) // shift right without sign
}

// GetValuesUInt32Range decodes consecutive, non-NULL integers without Scmer
// boxing. Composite storage types use this for internal uint32 vectors. The
// caller guarantees that all decoded values fit uint32.
func (s *StorageInt) GetValuesUInt32Range(recid uint32, count uint32, target []uint32, stride int) {
	if count == 0 {
		return
	}
	if s.hasNull {
		panic("StorageInt: UInt32 extraction does not support NULL")
	}
	if stride <= 0 {
		stride = 1
	}
	bitsize := uint(s.bitsize)
	bitpos := uint(recid) * bitsize
	chunkIdx := bitpos / 64
	bitOff := bitpos % 64
	targetIndex := 0
	for range count {
		value := s.chunk[chunkIdx] << bitOff
		if bitOff+bitsize > 64 {
			value |= s.chunk[chunkIdx+1] >> (64 - bitOff)
		}
		target[targetIndex] = uint32(int64(value>>(64-bitsize)) + s.offset)
		targetIndex += stride
		bitOff += bitsize
		if bitOff >= 64 {
			bitOff -= 64
			chunkIdx++
		}
	}
}

// GetValuesUInt32Multi is the arbitrary-position counterpart to
// GetValuesUInt32Range. Adjacent IDs retain the rolling bit cursor.
func (s *StorageInt) GetValuesUInt32Multi(recids []uint32, target []uint32, stride int) {
	if len(recids) == 0 {
		return
	}
	if s.hasNull {
		panic("StorageInt: UInt32 extraction does not support NULL")
	}
	if stride <= 0 {
		stride = 1
	}
	bitsize := uint(s.bitsize)
	var chunkIdx, bitOff uint
	var previous uint32
	havePosition := false
	targetIndex := 0
	for _, recid := range recids {
		if !havePosition || recid != previous+1 {
			bitpos := uint(recid) * bitsize
			chunkIdx = bitpos / 64
			bitOff = bitpos % 64
		}
		value := s.chunk[chunkIdx] << bitOff
		if bitOff+bitsize > 64 {
			value |= s.chunk[chunkIdx+1] >> (64 - bitOff)
		}
		target[targetIndex] = uint32(int64(value>>(64-bitsize)) + s.offset)
		targetIndex += stride
		previous, havePosition = recid, true
		bitOff += bitsize
		if bitOff >= 64 {
			bitOff -= 64
			chunkIdx++
		}
	}
}

// GetValueRange decodes count consecutive bit-packed values starting at
// recid. Unlike calling GetValueUInt in a loop, it keeps a running
// chunk/bit-offset cursor and advances it by bitsize each step instead of
// recomputing bitpos/64 and bitpos%64 (a division+modulo) from scratch for
// every element, since consecutive rows are exactly bitsize bits apart.
func (s *StorageInt) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	bitsize := uint(s.bitsize)
	hasNull := s.hasNull
	null := s.null
	offset := s.offset
	chunk := s.chunk

	bitpos := uint(recid) * bitsize
	chunkIdx := bitpos / 64
	bitOff := bitpos % 64
	idx := 0
	for k := uint32(0); k < count; k++ {
		v := chunk[chunkIdx] << bitOff
		if bitOff+bitsize > 64 {
			v |= chunk[chunkIdx+1] >> (64 - bitOff)
		}
		raw := v >> (64 - bitsize)
		if hasNull && raw == null {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(int64(raw) + offset)
		}
		idx += stride
		bitOff += bitsize
		if bitOff >= 64 {
			bitOff -= 64
			chunkIdx++
		}
	}
}

// GetValueMulti gathers values at arbitrary recids. Consecutive recids that
// happen to be adjacent (recids[k] == recids[k-1]+1, the common case for a
// batch drawn from a contiguous index range) reuse the rolling cursor from
// GetValueRange; any jump falls back to a direct bitpos computation for that
// element only.
func (s *StorageInt) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if len(recids) == 0 {
		return
	}
	bitsize := uint(s.bitsize)
	hasNull := s.hasNull
	null := s.null
	offset := s.offset
	chunk := s.chunk

	var chunkIdx, bitOff uint
	havePos := false
	var prevRecid uint32
	idx := 0
	for _, recid := range recids {
		if !havePos || recid != prevRecid+1 {
			bitpos := uint(recid) * bitsize
			chunkIdx = bitpos / 64
			bitOff = bitpos % 64
		}
		v := chunk[chunkIdx] << bitOff
		if bitOff+bitsize > 64 {
			v |= chunk[chunkIdx+1] >> (64 - bitOff)
		}
		raw := v >> (64 - bitsize)
		if hasNull && raw == null {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(int64(raw) + offset)
		}
		idx += stride
		prevRecid = recid
		havePos = true
		bitOff += bitsize
		if bitOff >= 64 {
			bitOff -= 64
			chunkIdx++
		}
	}
}

// getUIntMultiRaw decodes raw bit-packed values (no offset/null
// interpretation, no Scmer boxing) at recids into dst. Package-internal use
// only: wrapper formats that need these bits purely for an indirect lookup
// (e.g. StorageString's dictionary-entry indirection) can skip the
// Scmer-boxing round trip GetValueMulti pays for on every element, since
// those boxed values would just be unwrapped again immediately. Duplicates
// GetValueMulti's rolling-cursor loop rather than building on it, so the
// public GetValueMulti keeps writing straight into its caller's target with
// no extra intermediate allocation.
func (s *StorageInt) getUIntMultiRaw(recids []uint32, dst []uint64) {
	if len(recids) == 0 {
		return
	}
	bitsize := uint(s.bitsize)
	chunk := s.chunk

	var chunkIdx, bitOff uint
	havePos := false
	var prevRecid uint32
	for i, recid := range recids {
		if !havePos || recid != prevRecid+1 {
			bitpos := uint(recid) * bitsize
			chunkIdx = bitpos / 64
			bitOff = bitpos % 64
		}
		v := chunk[chunkIdx] << bitOff
		if bitOff+bitsize > 64 {
			v |= chunk[chunkIdx+1] >> (64 - bitOff)
		}
		dst[i] = v >> (64 - bitsize)
		prevRecid = recid
		havePos = true
		bitOff += bitsize
		if bitOff >= 64 {
			bitOff -= 64
			chunkIdx++
		}
	}
}

func (s *StorageInt) prepare() {
	// set up scan
	s.bitsize = 0
	s.offset = int64(1<<63 - 1)
	s.max = -s.offset - 1
	s.hasNull = false
}

// initValuesUInt32 initializes the normal StorageInt bit layout when a
// composite storage already knows its exact non-NULL uint32 range.
func (s *StorageInt) initValuesUInt32(count uint32, minimum, maximum uint32) {
	*s = StorageInt{offset: int64(minimum), max: int64(maximum), count: uint64(count)}
	s.bitsize = uint8(bits.Len32(maximum - minimum))
	if s.bitsize == 0 {
		s.bitsize = 1
	}
	if count > 0 {
		s.chunk = make([]uint64, ((uint(count)-1)*uint(s.bitsize)+65)/64+1)
	}
}

// buildValueUInt32 stores one value in an initValuesUInt32 backing without
// constructing a Scmer.
func (s *StorageInt) buildValueUInt32(i uint32, value uint32) {
	if i >= uint32(s.count) || int64(value) < s.offset || int64(value) > s.max {
		panic("StorageInt: uint32 value outside initialized range")
	}
	bitpos := uint(i) * uint(s.bitsize)
	packed := uint64(int64(value)-s.offset) << (64 - uint(s.bitsize))
	s.chunk[bitpos/64] |= packed >> (bitpos % 64)
	if bitpos%64+uint(s.bitsize) > 64 {
		s.chunk[bitpos/64+1] |= packed << (64 - bitpos%64)
	}
}
func (s *StorageInt) scan(i uint32, value scm.Scmer) {
	// storage is so simple, dont need scan
	if value.IsNil() {
		s.hasNull = true
		return
	}
	v := value.Int()
	if v < s.offset {
		s.offset = v
	}
	if v > s.max {
		s.max = v
	}
}
func (s *StorageInt) init(i uint32) {
	v := s.max - s.offset
	if s.hasNull {
		// store the value
		v = v + 1
		s.null = uint64(v)
	}
	if v == -1 {
		// no values at all
		v = 0
		s.offset = 0
		s.null = 0
	}
	s.bitsize = uint8(bits.Len64(uint64(v)))
	if s.bitsize == 0 {
		s.bitsize = 1
	}
	// allocate
	s.chunk = make([]uint64, ((uint(i)-1)*uint(s.bitsize)+65)/64+1)
	s.count = uint64(i)
	// fmt.Println("storing bitsize", s.bitsize,"null",s.null,"offset",s.offset)
}
func (s *StorageInt) build(i uint32, value scm.Scmer) {
	if i >= uint32(s.count) {
		panic("tried to build StorageInt outside of range")
	}
	// store
	vi := value.Int()
	if value.IsNil() {
		// null value
		vi = int64(s.null)
	} else {
		vi = vi - s.offset
	}
	bitpos := uint(i) * uint(s.bitsize)
	v := uint64(vi) << (64 - uint(s.bitsize))                      // shift value to the leftmost position of 64bit int
	s.chunk[bitpos/64] = s.chunk[bitpos/64] | (v >> (bitpos % 64)) // first chunk
	if bitpos%64+uint(s.bitsize) > 64 {
		s.chunk[bitpos/64+1] = s.chunk[bitpos/64+1] | v<<(64-bitpos%64) // second chunk
	}
}
func (s *StorageInt) finish() {
}
func (s *StorageInt) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

func (s *StorageInt) DistinctCount() uint { return uint(s.count) }
