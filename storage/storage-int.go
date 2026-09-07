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
	storageJITFunctions
	chunk   []uint64 `jit:"immutable-after-finish"`
	bitsize uint8    `jit:"immutable-after-finish"`
	offset  int64    `jit:"immutable-after-finish"`
	max     int64    // only of statistic use
	count   uint64   `jit:"immutable-after-finish"` // only stored for serialization purposes
	hasNull bool     `jit:"immutable-after-finish"`
	null    uint64   `jit:"immutable-after-finish"` // which value is null
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

func (s *StorageInt) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	var d0 scm.JITValueDesc
	_ = d0
	var d1 scm.JITValueDesc
	_ = d1
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
	var d19 scm.JITValueDesc
	_ = d19
	var d20 scm.JITValueDesc
	_ = d20
	var d21 scm.JITValueDesc
	_ = d21
	var d22 scm.JITValueDesc
	_ = d22
	var d23 scm.JITValueDesc
	_ = d23
	var d24 scm.JITValueDesc
	_ = d24
	var d25 scm.JITValueDesc
	_ = d25
	var d26 scm.JITValueDesc
	_ = d26
	var d27 scm.JITValueDesc
	_ = d27
	var d28 scm.JITValueDesc
	_ = d28
	var d29 scm.JITValueDesc
	_ = d29
	var d30 scm.JITValueDesc
	_ = d30
	var d31 scm.JITValueDesc
	_ = d31
	var d32 scm.JITValueDesc
	_ = d32
	var d33 scm.JITValueDesc
	_ = d33
	var d35 scm.JITValueDesc
	_ = d35
	var d36 scm.JITValueDesc
	_ = d36
	var d37 scm.JITValueDesc
	_ = d37
	var d38 scm.JITValueDesc
	_ = d38
	var d39 scm.JITValueDesc
	_ = d39
	var d40 scm.JITValueDesc
	_ = d40
	var d41 scm.JITValueDesc
	_ = d41
	var d42 scm.JITValueDesc
	_ = d42
	var d44 scm.JITValueDesc
	_ = d44
	var d45 scm.JITValueDesc
	_ = d45
	var d46 scm.JITValueDesc
	_ = d46
	var d47 scm.JITValueDesc
	_ = d47
	var d48 scm.JITValueDesc
	_ = d48
	var d49 scm.JITValueDesc
	_ = d49
	var d50 scm.JITValueDesc
	_ = d50
	var d51 scm.JITValueDesc
	_ = d51
	var d52 scm.JITValueDesc
	_ = d52
	var d53 scm.JITValueDesc
	_ = d53
	var d54 scm.JITValueDesc
	_ = d54
	var d55 scm.JITValueDesc
	_ = d55
	var d56 scm.JITValueDesc
	_ = d56
	var d57 scm.JITValueDesc
	_ = d57
	var d58 scm.JITValueDesc
	_ = d58
	var d59 scm.JITValueDesc
	_ = d59
	var d160 scm.JITValueDesc
	_ = d160
	var d161 scm.JITValueDesc
	_ = d161
	var d162 scm.JITValueDesc
	_ = d162
	var d163 scm.JITValueDesc
	_ = d163
	var d165 scm.JITValueDesc
	_ = d165
	var d166 scm.JITValueDesc
	_ = d166
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	ctx.TrackPointer(unsafe.Pointer(s))
	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	standaloneFrame := ctx.BeginStandaloneFrame()
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
	idxPinned := idxInt.Loc == scm.LocReg
	idxPinnedReg := idxInt.Reg
	if idxPinned {
		ctx.ProtectReg(idxPinnedReg)
		defer ctx.UnprotectReg(idxPinnedReg)
	}
	var bbs [5]scm.BBDescriptor
	if result.Loc == scm.LocAny {
		result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.BindReg(result.Reg, &result)
		ctx.BindReg(result.Reg2, &result)
	}
	resultRegsProtected := result.Loc == scm.LocRegPair
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
			ctx.FlushRegisterMoves()
			bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_0 = bbs[0].Address
			ctx.MarkLabel(lbl1)
			ctx.ResolveFixups()
		}
		ctx.ReclaimUntrackedRegs()
		var d0 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
			r0 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r0, thisptr.Reg, off)
			d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
			ctx.BindReg(r0, &d0)
		}
		d1 = d0
		ctx.EnsureDesc(&d1)
		if d1.Loc != scm.LocImm && d1.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d1.Loc == scm.LocImm {
			if d1.Imm.Bool() {
				if ps.General {
				}
				ps2 := scm.PhiState{General: ps.General}
				ps2.OverlayValues = make([]scm.JITValueDesc, 2)
				ps2.OverlayValues[0] = d0
				ps2.OverlayValues[1] = d1
				return bbs[2].RenderPS(ps2)
			}
			if ps.General {
			}
			ps3 := scm.PhiState{General: ps.General}
			ps3.OverlayValues = make([]scm.JITValueDesc, 2)
			ps3.OverlayValues[0] = d0
			ps3.OverlayValues[1] = d1
			return bbs[1].RenderPS(ps3)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d1.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl3)
		if bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
		}
		snap4 := d0
		snap5 := d1
		alloc6 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc6)
		d0 = snap4
		d1 = snap5
		ctx.RestoreAllocState(alloc6)
		d0 = snap4
		d1 = snap5
		ps7 := scm.PhiState{General: true}
		ps7.OverlayValues = make([]scm.JITValueDesc, 2)
		ps7.OverlayValues[0] = d0
		ps7.OverlayValues[1] = d1
		ps8 := scm.PhiState{General: true}
		ps8.OverlayValues = make([]scm.JITValueDesc, 2)
		ps8.OverlayValues[0] = d0
		ps8.OverlayValues[1] = d1
		snap9 := d0
		snap10 := d1
		alloc11 := ctx.SnapshotAllocState()
		if !bbs[1].Rendered {
			bbs[1].RenderPS(ps8)
		}
		ctx.RestoreAllocState(alloc11)
		d0 = snap9
		d1 = snap10
		if !bbs[2].Rendered {
			return bbs[2].RenderPS(ps7)
		}
		return result
		ctx.FreeDesc(&d0)
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
			ctx.FlushRegisterMoves()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&idxInt)
		d12 = idxInt
		_ = d12
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl6 := ctx.ReserveLabel()
		_ = lbl6
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl6)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d13 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
			r1 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r1, thisptr.Reg, off)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
			ctx.BindReg(r1, &d13)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d13)
		ctx.EnsureDesc(&d13)
		var d14 scm.JITValueDesc
		if d13.Loc == scm.LocImm {
			d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d13.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, d13.Reg)
			ctx.EmitShlRegImm8(r2, 56)
			ctx.EmitShrRegImm8(r2, 56)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d14)
		}
		ctx.FreeDesc(&d13)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d12)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d14)
		ctx.EnsureDescsTogether(&d12, &d14)
		var d16 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d14.Loc == scm.LocImm {
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() * d14.Imm.Int())}
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitImulInt64(scratch, d14.Reg)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d16)
		} else if d14.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d14.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d16)
		} else {
			r3 := ctx.AllocRegExcept(d12.Reg, d14.Reg)
			ctx.EmitMovRegReg(r3, d12.Reg)
			ctx.EmitImulInt64(r3, d14.Reg)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d16)
		}
		if d16.Loc == scm.LocReg && d12.Loc == scm.LocReg && d16.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d16)
		var d17 scm.JITValueDesc
		if d16.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() / 64)}
		} else {
			r4 := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(r4, d16.Reg)
			ctx.EmitShrRegImm8(r4, 6)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d17)
		}
		if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
			ctx.TransferReg(d16.Reg)
			d16.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d16)
		resultTarget18 := false
		_ = resultTarget18
		var d19 scm.JITValueDesc
		if d16.Loc == scm.LocImm {
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() % 64)}
		} else {
			r5 := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(r5, d16.Reg)
			ctx.EmitAndRegImm32(r5, 63)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d19)
		}
		if d19.Loc == scm.LocReg && d16.Loc == scm.LocReg && d19.Reg == d16.Reg {
			ctx.TransferReg(d16.Reg)
			d16.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d16)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d20 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d20 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r6 := ctx.AllocReg()
			r7 := ctx.AllocRegExcept(r6)
			r8 := ctx.AllocRegExcept(r6, r7)
			off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
			ctx.EmitMovRegMem(r6, thisptr.Reg, off)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off+16)
			d20 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r6, Reg2: r7, Reg3: r8}
			ctx.BindReg(r6, &d20)
			ctx.BindReg(r7, &d20)
			ctx.BindReg(r8, &d20)
			ctx.BindReg(r6, &d20)
			ctx.BindReg(r7, &d20)
			ctx.BindReg(r8, &d20)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.ReclaimUntrackedRegs()
		d21 = ctx.EmitLoadScalarSliceElement(&d20, &d17, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d19)
		ctx.EnsureDescsTogether(&d21, &d19)
		var d22 scm.JITValueDesc
		if d21.Loc == scm.LocImm && d19.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d21.Imm.Int()) << uint64(d19.Imm.Int())))}
		} else if d19.Loc == scm.LocImm {
			r9 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitMovRegReg(r9, d21.Reg)
			ctx.EmitShlRegImm8(r9, uint8(d19.Imm.Int()))
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d22)
		} else {
			{
				shiftSrc := d21.Reg
				r10 := ctx.AllocRegExcept(d21.Reg, d19.Reg)
				ctx.EmitMovRegReg(r10, d21.Reg)
				shiftSrc = r10
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d19.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d19.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d19.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d22)
			}
		}
		if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
			ctx.TransferReg(d21.Reg)
			d21.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d21)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d17)
		var d23 scm.JITValueDesc
		if d17.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(scratch, d17.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d23)
		}
		if d23.Loc == scm.LocReg && d17.Loc == scm.LocReg && d23.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		d24 = ctx.EmitLoadScalarSliceElement(&d20, &d23, 8, scm.TagInt)
		ctx.FreeDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d19)
		ctx.EnsureDescsTogether(&d25, &d19)
		var d26 scm.JITValueDesc
		if d25.Loc == scm.LocImm && d19.Loc == scm.LocImm {
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() - d19.Imm.Int())}
		} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
			r11 := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegReg(r11, d25.Reg)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d26)
		} else if d25.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
			ctx.EmitSubInt64(scratch, d19.Reg)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d26)
		} else if d19.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegReg(scratch, d25.Reg)
			if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d19.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d26)
		} else {
			r12 := ctx.AllocRegExcept(d25.Reg, d19.Reg)
			ctx.EmitMovRegReg(r12, d25.Reg)
			ctx.EmitSubInt64(r12, d19.Reg)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d26)
		}
		if d26.Loc == scm.LocReg && d25.Loc == scm.LocReg && d26.Reg == d25.Reg {
			ctx.TransferReg(d25.Reg)
			d25.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d19)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d26)
		ctx.EnsureDescsTogether(&d24, &d26)
		var d27 scm.JITValueDesc
		if d24.Loc == scm.LocImm && d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d26.Imm.Int())))}
		} else if d26.Loc == scm.LocImm {
			r13 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r13, d24.Reg)
			ctx.EmitShrRegImm8(r13, uint8(d26.Imm.Int()))
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d27)
		} else {
			{
				shiftSrc := d24.Reg
				r14 := ctx.AllocRegExcept(d24.Reg, d26.Reg)
				ctx.EmitMovRegReg(r14, d24.Reg)
				shiftSrc = r14
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d26.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d26.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d26.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d27)
			}
		}
		if d27.Loc == scm.LocReg && d24.Loc == scm.LocReg && d27.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.FreeDesc(&d26)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d27)
		var d28 scm.JITValueDesc
		if d22.Loc == scm.LocImm && d27.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() | d27.Imm.Int())}
		} else if d22.Loc == scm.LocImm && d22.Imm.Int() == 0 {
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg}
			ctx.BindReg(d27.Reg, &d28)
		} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
			r15 := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(r15, d22.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d28)
		} else if d22.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d27.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
			ctx.EmitOrInt64(scratch, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d28)
		} else if d27.Loc == scm.LocImm {
			r16 := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(r16, d22.Reg)
			if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r16, int32(d27.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
				ctx.EmitOrInt64(r16, scm.RegR11)
			}
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d28)
		} else {
			r17 := ctx.AllocRegExcept(d22.Reg, d27.Reg)
			ctx.EmitMovRegReg(r17, d22.Reg)
			ctx.EmitOrInt64(r17, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d28)
		}
		if d28.Loc == scm.LocReg && d22.Loc == scm.LocReg && d28.Reg == d22.Reg {
			ctx.TransferReg(d22.Reg)
			d22.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d22)
		ctx.FreeDesc(&d27)
		ctx.ReclaimUntrackedRegs()
		d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d14)
		ctx.EnsureDescsTogether(&d29, &d14)
		var d30 scm.JITValueDesc
		if d29.Loc == scm.LocImm && d14.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() - d14.Imm.Int())}
		} else if d14.Loc == scm.LocImm && d14.Imm.Int() == 0 {
			r18 := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(r18, d29.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d30)
		} else if d29.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
			ctx.EmitSubInt64(scratch, d14.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		} else if d14.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(scratch, d29.Reg)
			if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d14.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		} else {
			r19 := ctx.AllocRegExcept(d29.Reg, d14.Reg)
			ctx.EmitMovRegReg(r19, d29.Reg)
			ctx.EmitSubInt64(r19, d14.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d30)
		}
		if d30.Loc == scm.LocReg && d29.Loc == scm.LocReg && d30.Reg == d29.Reg {
			ctx.TransferReg(d29.Reg)
			d29.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d14)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d30)
		ctx.EnsureDescsTogether(&d28, &d30)
		var d31 scm.JITValueDesc
		if d28.Loc == scm.LocImm && d30.Loc == scm.LocImm {
			d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d28.Imm.Int()) >> uint64(d30.Imm.Int())))}
		} else if d30.Loc == scm.LocImm {
			r20 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r20, d28.Reg)
			ctx.EmitShrRegImm8(r20, uint8(d30.Imm.Int()))
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d31)
		} else {
			{
				shiftSrc := d28.Reg
				r21 := ctx.AllocRegExcept(d28.Reg, d30.Reg)
				ctx.EmitMovRegReg(r21, d28.Reg)
				shiftSrc = r21
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d30.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d30.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d30.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d31)
			}
		}
		if d31.Loc == scm.LocReg && d28.Loc == scm.LocReg && d31.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		var d32 scm.JITValueDesc
		if d31.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d31.Imm.Int()))))}
		} else {
			r22 := ctx.AllocReg()
			ctx.EmitMovRegReg(r22, d31.Reg)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d32)
		}
		ctx.FreeDesc(&d31)
		var d33 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r23 := ctx.AllocReg()
			ctx.EmitMovRegMem(r23, thisptr.Reg, off)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d33)
		}
		ctx.EnsureDesc(&d32)
		resultTarget34 := false
		_ = resultTarget34
		ctx.EnsureDesc(&d33)
		ctx.EnsureDescsTogether(&d32, &d33)
		var d35 scm.JITValueDesc
		if d32.Loc == scm.LocImm && d33.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() + d33.Imm.Int())}
		} else if d33.Loc == scm.LocImm && d33.Imm.Int() == 0 {
			var r24 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d32.Reg {
				r24 = result.Reg2
				resultTarget34 = true
			} else {
				r24 = ctx.AllocRegExcept(d32.Reg)
			}
			ctx.EmitMovRegReg(r24, d32.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d35)
		} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d33.Reg}
			ctx.BindReg(d33.Reg, &d35)
		} else if d32.Loc == scm.LocImm {
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d33.Reg {
				scratch = result.Reg2
				resultTarget34 = true
			} else {
				scratch = ctx.AllocRegExcept(d33.Reg)
			}
			ctx.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
			ctx.EmitAddInt64(scratch, d33.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d35)
		} else if d33.Loc == scm.LocImm {
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d32.Reg {
				scratch = result.Reg2
				resultTarget34 = true
			} else {
				scratch = ctx.AllocRegExcept(d32.Reg)
			}
			ctx.EmitMovRegReg(scratch, d32.Reg)
			if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d33.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d33.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d35)
		} else {
			var r25 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d32.Reg && result.Reg2 != d33.Reg {
				r25 = result.Reg2
				resultTarget34 = true
			} else {
				r25 = ctx.AllocRegExcept(d32.Reg, d33.Reg)
			}
			ctx.EmitMovRegReg(r25, d32.Reg)
			ctx.EmitAddInt64(r25, d33.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d35)
		}
		if d35.Loc == scm.LocReg && d32.Loc == scm.LocReg && d35.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		if resultTarget34 && d35.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d32)
		ctx.FreeDesc(&d33)
		ctx.EnsureDesc(&d35)
		d36 = result
		ctx.EnsureDesc(&d35)
		ctx.EmitMakeInt(d36, d35)
		if d35.Loc == scm.LocReg {
			ctx.FreeReg(d35.Reg)
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
			d19 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
			d20 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
			d21 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
			d30 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&idxInt)
		d37 = idxInt
		_ = d37
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl7 := ctx.ReserveLabel()
		_ = lbl7
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl7)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d38 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
			r26 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r26, thisptr.Reg, off)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d38)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d38)
		var d39 scm.JITValueDesc
		if d38.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d38.Imm.Int()))))}
		} else {
			r27 := ctx.AllocReg()
			ctx.EmitMovRegReg(r27, d38.Reg)
			ctx.EmitShlRegImm8(r27, 56)
			ctx.EmitShrRegImm8(r27, 56)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d39)
		}
		ctx.FreeDesc(&d38)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d39)
		ctx.EnsureDescsTogether(&d37, &d39)
		var d41 scm.JITValueDesc
		if d37.Loc == scm.LocImm && d39.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() * d39.Imm.Int())}
		} else if d37.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
			ctx.EmitImulInt64(scratch, d39.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else if d39.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(scratch, d37.Reg)
			if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d39.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else {
			r28 := ctx.AllocRegExcept(d37.Reg, d39.Reg)
			ctx.EmitMovRegReg(r28, d37.Reg)
			ctx.EmitImulInt64(r28, d39.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d41)
		}
		if d41.Loc == scm.LocReg && d37.Loc == scm.LocReg && d41.Reg == d37.Reg {
			ctx.TransferReg(d37.Reg)
			d37.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		var d42 scm.JITValueDesc
		if d41.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() / 64)}
		} else {
			r29 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r29, d41.Reg)
			ctx.EmitShrRegImm8(r29, 6)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d42)
		}
		if d42.Loc == scm.LocReg && d41.Loc == scm.LocReg && d42.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		resultTarget43 := false
		_ = resultTarget43
		var d44 scm.JITValueDesc
		if d41.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() % 64)}
		} else {
			r30 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r30, d41.Reg)
			ctx.EmitAndRegImm32(r30, 63)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d44)
		}
		if d44.Loc == scm.LocReg && d41.Loc == scm.LocReg && d44.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d45 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d45 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r31 := ctx.AllocReg()
			r32 := ctx.AllocRegExcept(r31)
			r33 := ctx.AllocRegExcept(r31, r32)
			off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
			ctx.EmitMovRegMem(r31, thisptr.Reg, off)
			ctx.EmitMovRegMem(r32, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r33, thisptr.Reg, off+16)
			d45 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r31, Reg2: r32, Reg3: r33}
			ctx.BindReg(r31, &d45)
			ctx.BindReg(r32, &d45)
			ctx.BindReg(r33, &d45)
			ctx.BindReg(r31, &d45)
			ctx.BindReg(r32, &d45)
			ctx.BindReg(r33, &d45)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		d46 = ctx.EmitLoadScalarSliceElement(&d45, &d42, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d46, &d44)
		var d47 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) << uint64(d44.Imm.Int())))}
		} else if d44.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r34, d46.Reg)
			ctx.EmitShlRegImm8(r34, uint8(d44.Imm.Int()))
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d47)
		} else {
			{
				shiftSrc := d46.Reg
				r35 := ctx.AllocRegExcept(d46.Reg, d44.Reg)
				ctx.EmitMovRegReg(r35, d46.Reg)
				shiftSrc = r35
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d44.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d44.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d44.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d47)
			}
		}
		if d47.Loc == scm.LocReg && d46.Loc == scm.LocReg && d47.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d42)
		ctx.EnsureDesc(&d42)
		var d48 scm.JITValueDesc
		if d42.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegReg(scratch, d42.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d48)
		}
		if d48.Loc == scm.LocReg && d42.Loc == scm.LocReg && d48.Reg == d42.Reg {
			ctx.TransferReg(d42.Reg)
			d42.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d48)
		ctx.ReclaimUntrackedRegs()
		d49 = ctx.EmitLoadScalarSliceElement(&d45, &d48, 8, scm.TagInt)
		ctx.FreeDesc(&d48)
		ctx.ReclaimUntrackedRegs()
		d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d50, &d44)
		var d51 scm.JITValueDesc
		if d50.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() - d44.Imm.Int())}
		} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
			r36 := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegReg(r36, d50.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d51)
		} else if d50.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d50.Imm.Int()))
			ctx.EmitSubInt64(scratch, d44.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d51)
		} else if d44.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegReg(scratch, d50.Reg)
			if d44.Imm.Int() >= -2147483648 && d44.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d44.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d51)
		} else {
			r37 := ctx.AllocRegExcept(d50.Reg, d44.Reg)
			ctx.EmitMovRegReg(r37, d50.Reg)
			ctx.EmitSubInt64(r37, d44.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d51)
		}
		if d51.Loc == scm.LocReg && d50.Loc == scm.LocReg && d51.Reg == d50.Reg {
			ctx.TransferReg(d50.Reg)
			d50.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d49)
		ctx.EnsureDesc(&d51)
		ctx.EnsureDescsTogether(&d49, &d51)
		var d52 scm.JITValueDesc
		if d49.Loc == scm.LocImm && d51.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d49.Imm.Int()) >> uint64(d51.Imm.Int())))}
		} else if d51.Loc == scm.LocImm {
			r38 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegReg(r38, d49.Reg)
			ctx.EmitShrRegImm8(r38, uint8(d51.Imm.Int()))
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d52)
		} else {
			{
				shiftSrc := d49.Reg
				r39 := ctx.AllocRegExcept(d49.Reg, d51.Reg)
				ctx.EmitMovRegReg(r39, d49.Reg)
				shiftSrc = r39
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d51.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d51.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d51.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d52)
			}
		}
		if d52.Loc == scm.LocReg && d49.Loc == scm.LocReg && d52.Reg == d49.Reg {
			ctx.TransferReg(d49.Reg)
			d49.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d49)
		ctx.FreeDesc(&d51)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.EnsureDesc(&d52)
		var d53 scm.JITValueDesc
		if d47.Loc == scm.LocImm && d52.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() | d52.Imm.Int())}
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			ctx.BindReg(d52.Reg, &d53)
		} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
			r40 := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(r40, d47.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d53)
		} else if d47.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
			ctx.EmitOrInt64(scratch, d52.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d53)
		} else if d52.Loc == scm.LocImm {
			r41 := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(r41, d47.Reg)
			if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r41, int32(d52.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
				ctx.EmitOrInt64(r41, scm.RegR11)
			}
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d53)
		} else {
			r42 := ctx.AllocRegExcept(d47.Reg, d52.Reg)
			ctx.EmitMovRegReg(r42, d47.Reg)
			ctx.EmitOrInt64(r42, d52.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d53)
		}
		if d53.Loc == scm.LocReg && d47.Loc == scm.LocReg && d53.Reg == d47.Reg {
			ctx.TransferReg(d47.Reg)
			d47.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d47)
		ctx.FreeDesc(&d52)
		ctx.ReclaimUntrackedRegs()
		d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d39)
		ctx.EnsureDescsTogether(&d54, &d39)
		var d55 scm.JITValueDesc
		if d54.Loc == scm.LocImm && d39.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() - d39.Imm.Int())}
		} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
			r43 := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(r43, d54.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d55)
		} else if d54.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
			ctx.EmitSubInt64(scratch, d39.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d55)
		} else if d39.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(scratch, d54.Reg)
			if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d39.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d55)
		} else {
			r44 := ctx.AllocRegExcept(d54.Reg, d39.Reg)
			ctx.EmitMovRegReg(r44, d54.Reg)
			ctx.EmitSubInt64(r44, d39.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d55)
		}
		if d55.Loc == scm.LocReg && d54.Loc == scm.LocReg && d55.Reg == d54.Reg {
			ctx.TransferReg(d54.Reg)
			d54.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDescsTogether(&d53, &d55)
		var d56 scm.JITValueDesc
		if d53.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) >> uint64(d55.Imm.Int())))}
		} else if d55.Loc == scm.LocImm {
			r45 := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegReg(r45, d53.Reg)
			ctx.EmitShrRegImm8(r45, uint8(d55.Imm.Int()))
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d56)
		} else {
			{
				shiftSrc := d53.Reg
				r46 := ctx.AllocRegExcept(d53.Reg, d55.Reg)
				ctx.EmitMovRegReg(r46, d53.Reg)
				shiftSrc = r46
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d55.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d55.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d55.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d56)
			}
		}
		if d56.Loc == scm.LocReg && d53.Loc == scm.LocReg && d56.Reg == d53.Reg {
			ctx.TransferReg(d53.Reg)
			d53.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d53)
		ctx.FreeDesc(&d55)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d56)
		ctx.FreeDesc(&idxInt)
		var d57 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
			r47 := ctx.AllocReg()
			ctx.EmitMovRegMem(r47, thisptr.Reg, off)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			ctx.BindReg(r47, &d57)
		}
		ctx.EnsureDesc(&d56)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d56, &d57)
		var d58 scm.JITValueDesc
		if d56.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d56.Imm.Int()) == uint64(d57.Imm.Int()))}
		} else if d57.Loc == scm.LocImm {
			r48 := ctx.AllocRegExcept(d56.Reg)
			if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d56.Reg, int32(d57.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
				ctx.EmitCmpInt64(d56.Reg, scm.RegR11)
			}
			d58 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r48, Condition: scm.CondEqual}
			ctx.BindReg(r48, &d58)
		} else if d56.Loc == scm.LocImm {
			r49 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d57.Reg)
			d58 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r49, Condition: scm.CondEqual}
			ctx.BindReg(r49, &d58)
		} else {
			r50 := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitCmpInt64(d56.Reg, d57.Reg)
			d58 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r50, Condition: scm.CondEqual}
			ctx.BindReg(r50, &d58)
		}
		ctx.FreeDesc(&d57)
		d59 = d58
		ctx.EnsureDesc(&d59)
		if d59.Loc != scm.LocImm && d59.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d59.Loc == scm.LocImm {
			if d59.Imm.Bool() {
				if ps.General {
				}
				ps60 := scm.PhiState{General: ps.General}
				ps60.OverlayValues = make([]scm.JITValueDesc, 60)
				ps60.OverlayValues[0] = d0
				ps60.OverlayValues[1] = d1
				ps60.OverlayValues[12] = d12
				ps60.OverlayValues[13] = d13
				ps60.OverlayValues[14] = d14
				ps60.OverlayValues[15] = d15
				ps60.OverlayValues[16] = d16
				ps60.OverlayValues[17] = d17
				ps60.OverlayValues[19] = d19
				ps60.OverlayValues[20] = d20
				ps60.OverlayValues[21] = d21
				ps60.OverlayValues[22] = d22
				ps60.OverlayValues[23] = d23
				ps60.OverlayValues[24] = d24
				ps60.OverlayValues[25] = d25
				ps60.OverlayValues[26] = d26
				ps60.OverlayValues[27] = d27
				ps60.OverlayValues[28] = d28
				ps60.OverlayValues[29] = d29
				ps60.OverlayValues[30] = d30
				ps60.OverlayValues[31] = d31
				ps60.OverlayValues[32] = d32
				ps60.OverlayValues[33] = d33
				ps60.OverlayValues[35] = d35
				ps60.OverlayValues[36] = d36
				ps60.OverlayValues[37] = d37
				ps60.OverlayValues[38] = d38
				ps60.OverlayValues[39] = d39
				ps60.OverlayValues[40] = d40
				ps60.OverlayValues[41] = d41
				ps60.OverlayValues[42] = d42
				ps60.OverlayValues[44] = d44
				ps60.OverlayValues[45] = d45
				ps60.OverlayValues[46] = d46
				ps60.OverlayValues[47] = d47
				ps60.OverlayValues[48] = d48
				ps60.OverlayValues[49] = d49
				ps60.OverlayValues[50] = d50
				ps60.OverlayValues[51] = d51
				ps60.OverlayValues[52] = d52
				ps60.OverlayValues[53] = d53
				ps60.OverlayValues[54] = d54
				ps60.OverlayValues[55] = d55
				ps60.OverlayValues[56] = d56
				ps60.OverlayValues[57] = d57
				ps60.OverlayValues[58] = d58
				ps60.OverlayValues[59] = d59
				return bbs[3].RenderPS(ps60)
			}
			if ps.General {
			}
			ps61 := scm.PhiState{General: ps.General}
			ps61.OverlayValues = make([]scm.JITValueDesc, 60)
			ps61.OverlayValues[0] = d0
			ps61.OverlayValues[1] = d1
			ps61.OverlayValues[12] = d12
			ps61.OverlayValues[13] = d13
			ps61.OverlayValues[14] = d14
			ps61.OverlayValues[15] = d15
			ps61.OverlayValues[16] = d16
			ps61.OverlayValues[17] = d17
			ps61.OverlayValues[19] = d19
			ps61.OverlayValues[20] = d20
			ps61.OverlayValues[21] = d21
			ps61.OverlayValues[22] = d22
			ps61.OverlayValues[23] = d23
			ps61.OverlayValues[24] = d24
			ps61.OverlayValues[25] = d25
			ps61.OverlayValues[26] = d26
			ps61.OverlayValues[27] = d27
			ps61.OverlayValues[28] = d28
			ps61.OverlayValues[29] = d29
			ps61.OverlayValues[30] = d30
			ps61.OverlayValues[31] = d31
			ps61.OverlayValues[32] = d32
			ps61.OverlayValues[33] = d33
			ps61.OverlayValues[35] = d35
			ps61.OverlayValues[36] = d36
			ps61.OverlayValues[37] = d37
			ps61.OverlayValues[38] = d38
			ps61.OverlayValues[39] = d39
			ps61.OverlayValues[40] = d40
			ps61.OverlayValues[41] = d41
			ps61.OverlayValues[42] = d42
			ps61.OverlayValues[44] = d44
			ps61.OverlayValues[45] = d45
			ps61.OverlayValues[46] = d46
			ps61.OverlayValues[47] = d47
			ps61.OverlayValues[48] = d48
			ps61.OverlayValues[49] = d49
			ps61.OverlayValues[50] = d50
			ps61.OverlayValues[51] = d51
			ps61.OverlayValues[52] = d52
			ps61.OverlayValues[53] = d53
			ps61.OverlayValues[54] = d54
			ps61.OverlayValues[55] = d55
			ps61.OverlayValues[56] = d56
			ps61.OverlayValues[57] = d57
			ps61.OverlayValues[58] = d58
			ps61.OverlayValues[59] = d59
			return bbs[4].RenderPS(ps61)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitJump(d59.Condition, lbl4)
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
		}
		ctx.FreeDesc(&d58)
		snap62 := d0
		snap63 := d1
		snap64 := d12
		snap65 := d13
		snap66 := d14
		snap67 := d15
		snap68 := d16
		snap69 := d17
		snap70 := d19
		snap71 := d20
		snap72 := d21
		snap73 := d22
		snap74 := d23
		snap75 := d24
		snap76 := d25
		snap77 := d26
		snap78 := d27
		snap79 := d28
		snap80 := d29
		snap81 := d30
		snap82 := d31
		snap83 := d32
		snap84 := d33
		snap85 := d35
		snap86 := d36
		snap87 := d37
		snap88 := d38
		snap89 := d39
		snap90 := d40
		snap91 := d41
		snap92 := d42
		snap93 := d44
		snap94 := d45
		snap95 := d46
		snap96 := d47
		snap97 := d48
		snap98 := d49
		snap99 := d50
		snap100 := d51
		snap101 := d52
		snap102 := d53
		snap103 := d54
		snap104 := d55
		snap105 := d56
		snap106 := d57
		snap107 := d58
		snap108 := d59
		alloc109 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc109)
		d0 = snap62
		d1 = snap63
		d12 = snap64
		d13 = snap65
		d14 = snap66
		d15 = snap67
		d16 = snap68
		d17 = snap69
		d19 = snap70
		d20 = snap71
		d21 = snap72
		d22 = snap73
		d23 = snap74
		d24 = snap75
		d25 = snap76
		d26 = snap77
		d27 = snap78
		d28 = snap79
		d29 = snap80
		d30 = snap81
		d31 = snap82
		d32 = snap83
		d33 = snap84
		d35 = snap85
		d36 = snap86
		d37 = snap87
		d38 = snap88
		d39 = snap89
		d40 = snap90
		d41 = snap91
		d42 = snap92
		d44 = snap93
		d45 = snap94
		d46 = snap95
		d47 = snap96
		d48 = snap97
		d49 = snap98
		d50 = snap99
		d51 = snap100
		d52 = snap101
		d53 = snap102
		d54 = snap103
		d55 = snap104
		d56 = snap105
		d57 = snap106
		d58 = snap107
		d59 = snap108
		ctx.RestoreAllocState(alloc109)
		d0 = snap62
		d1 = snap63
		d12 = snap64
		d13 = snap65
		d14 = snap66
		d15 = snap67
		d16 = snap68
		d17 = snap69
		d19 = snap70
		d20 = snap71
		d21 = snap72
		d22 = snap73
		d23 = snap74
		d24 = snap75
		d25 = snap76
		d26 = snap77
		d27 = snap78
		d28 = snap79
		d29 = snap80
		d30 = snap81
		d31 = snap82
		d32 = snap83
		d33 = snap84
		d35 = snap85
		d36 = snap86
		d37 = snap87
		d38 = snap88
		d39 = snap89
		d40 = snap90
		d41 = snap91
		d42 = snap92
		d44 = snap93
		d45 = snap94
		d46 = snap95
		d47 = snap96
		d48 = snap97
		d49 = snap98
		d50 = snap99
		d51 = snap100
		d52 = snap101
		d53 = snap102
		d54 = snap103
		d55 = snap104
		d56 = snap105
		d57 = snap106
		d58 = snap107
		d59 = snap108
		ps110 := scm.PhiState{General: true}
		ps110.OverlayValues = make([]scm.JITValueDesc, 60)
		ps110.OverlayValues[0] = d0
		ps110.OverlayValues[1] = d1
		ps110.OverlayValues[12] = d12
		ps110.OverlayValues[13] = d13
		ps110.OverlayValues[14] = d14
		ps110.OverlayValues[15] = d15
		ps110.OverlayValues[16] = d16
		ps110.OverlayValues[17] = d17
		ps110.OverlayValues[19] = d19
		ps110.OverlayValues[20] = d20
		ps110.OverlayValues[21] = d21
		ps110.OverlayValues[22] = d22
		ps110.OverlayValues[23] = d23
		ps110.OverlayValues[24] = d24
		ps110.OverlayValues[25] = d25
		ps110.OverlayValues[26] = d26
		ps110.OverlayValues[27] = d27
		ps110.OverlayValues[28] = d28
		ps110.OverlayValues[29] = d29
		ps110.OverlayValues[30] = d30
		ps110.OverlayValues[31] = d31
		ps110.OverlayValues[32] = d32
		ps110.OverlayValues[33] = d33
		ps110.OverlayValues[35] = d35
		ps110.OverlayValues[36] = d36
		ps110.OverlayValues[37] = d37
		ps110.OverlayValues[38] = d38
		ps110.OverlayValues[39] = d39
		ps110.OverlayValues[40] = d40
		ps110.OverlayValues[41] = d41
		ps110.OverlayValues[42] = d42
		ps110.OverlayValues[44] = d44
		ps110.OverlayValues[45] = d45
		ps110.OverlayValues[46] = d46
		ps110.OverlayValues[47] = d47
		ps110.OverlayValues[48] = d48
		ps110.OverlayValues[49] = d49
		ps110.OverlayValues[50] = d50
		ps110.OverlayValues[51] = d51
		ps110.OverlayValues[52] = d52
		ps110.OverlayValues[53] = d53
		ps110.OverlayValues[54] = d54
		ps110.OverlayValues[55] = d55
		ps110.OverlayValues[56] = d56
		ps110.OverlayValues[57] = d57
		ps110.OverlayValues[58] = d58
		ps110.OverlayValues[59] = d59
		ps111 := scm.PhiState{General: true}
		ps111.OverlayValues = make([]scm.JITValueDesc, 60)
		ps111.OverlayValues[0] = d0
		ps111.OverlayValues[1] = d1
		ps111.OverlayValues[12] = d12
		ps111.OverlayValues[13] = d13
		ps111.OverlayValues[14] = d14
		ps111.OverlayValues[15] = d15
		ps111.OverlayValues[16] = d16
		ps111.OverlayValues[17] = d17
		ps111.OverlayValues[19] = d19
		ps111.OverlayValues[20] = d20
		ps111.OverlayValues[21] = d21
		ps111.OverlayValues[22] = d22
		ps111.OverlayValues[23] = d23
		ps111.OverlayValues[24] = d24
		ps111.OverlayValues[25] = d25
		ps111.OverlayValues[26] = d26
		ps111.OverlayValues[27] = d27
		ps111.OverlayValues[28] = d28
		ps111.OverlayValues[29] = d29
		ps111.OverlayValues[30] = d30
		ps111.OverlayValues[31] = d31
		ps111.OverlayValues[32] = d32
		ps111.OverlayValues[33] = d33
		ps111.OverlayValues[35] = d35
		ps111.OverlayValues[36] = d36
		ps111.OverlayValues[37] = d37
		ps111.OverlayValues[38] = d38
		ps111.OverlayValues[39] = d39
		ps111.OverlayValues[40] = d40
		ps111.OverlayValues[41] = d41
		ps111.OverlayValues[42] = d42
		ps111.OverlayValues[44] = d44
		ps111.OverlayValues[45] = d45
		ps111.OverlayValues[46] = d46
		ps111.OverlayValues[47] = d47
		ps111.OverlayValues[48] = d48
		ps111.OverlayValues[49] = d49
		ps111.OverlayValues[50] = d50
		ps111.OverlayValues[51] = d51
		ps111.OverlayValues[52] = d52
		ps111.OverlayValues[53] = d53
		ps111.OverlayValues[54] = d54
		ps111.OverlayValues[55] = d55
		ps111.OverlayValues[56] = d56
		ps111.OverlayValues[57] = d57
		ps111.OverlayValues[58] = d58
		ps111.OverlayValues[59] = d59
		snap112 := d0
		snap113 := d1
		snap114 := d12
		snap115 := d13
		snap116 := d14
		snap117 := d15
		snap118 := d16
		snap119 := d17
		snap120 := d19
		snap121 := d20
		snap122 := d21
		snap123 := d22
		snap124 := d23
		snap125 := d24
		snap126 := d25
		snap127 := d26
		snap128 := d27
		snap129 := d28
		snap130 := d29
		snap131 := d30
		snap132 := d31
		snap133 := d32
		snap134 := d33
		snap135 := d35
		snap136 := d36
		snap137 := d37
		snap138 := d38
		snap139 := d39
		snap140 := d40
		snap141 := d41
		snap142 := d42
		snap143 := d44
		snap144 := d45
		snap145 := d46
		snap146 := d47
		snap147 := d48
		snap148 := d49
		snap149 := d50
		snap150 := d51
		snap151 := d52
		snap152 := d53
		snap153 := d54
		snap154 := d55
		snap155 := d56
		snap156 := d57
		snap157 := d58
		snap158 := d59
		alloc159 := ctx.SnapshotAllocState()
		if !bbs[4].Rendered {
			bbs[4].RenderPS(ps111)
		}
		ctx.RestoreAllocState(alloc159)
		d0 = snap112
		d1 = snap113
		d12 = snap114
		d13 = snap115
		d14 = snap116
		d15 = snap117
		d16 = snap118
		d17 = snap119
		d19 = snap120
		d20 = snap121
		d21 = snap122
		d22 = snap123
		d23 = snap124
		d24 = snap125
		d25 = snap126
		d26 = snap127
		d27 = snap128
		d28 = snap129
		d29 = snap130
		d30 = snap131
		d31 = snap132
		d32 = snap133
		d33 = snap134
		d35 = snap135
		d36 = snap136
		d37 = snap137
		d38 = snap138
		d39 = snap139
		d40 = snap140
		d41 = snap141
		d42 = snap142
		d44 = snap143
		d45 = snap144
		d46 = snap145
		d47 = snap146
		d48 = snap147
		d49 = snap148
		d50 = snap149
		d51 = snap150
		d52 = snap151
		d53 = snap152
		d54 = snap153
		d55 = snap154
		d56 = snap155
		d57 = snap156
		d58 = snap157
		d59 = snap158
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps110)
		}
		return result
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
			ctx.FlushRegisterMoves()
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
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
			d19 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
			d20 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
			d21 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
			d30 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
			d45 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
			d46 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
			d47 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
			d48 = ps.OverlayValues[48]
		}
		if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
			d49 = ps.OverlayValues[49]
		}
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
			d50 = ps.OverlayValues[50]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
			d56 = ps.OverlayValues[56]
		}
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		ctx.ReclaimUntrackedRegs()
		d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d161 = result
		ctx.EnsureDesc(&d160)
		if d160.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d160, &d161)
		} else {
			switch d160.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d161, d160)
			case scm.TagInt:
				ctx.EmitMakeInt(d161, d160)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d161, d160)
			case scm.TagNil:
				ctx.EmitMakeNil(d161)
			default:
				ctx.EmitMovPairToResult(&d160, &d161)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			ctx.FlushRegisterMoves()
			bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_4 = bbs[4].Address
			ctx.MarkLabel(lbl5)
			ctx.ResolveFixups()
		}
		if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
			d0 = ps.OverlayValues[0]
		}
		if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
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
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
			d19 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
			d20 = ps.OverlayValues[20]
		}
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
			d21 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
			d23 = ps.OverlayValues[23]
		}
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
			d25 = ps.OverlayValues[25]
		}
		if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
			d26 = ps.OverlayValues[26]
		}
		if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
			d27 = ps.OverlayValues[27]
		}
		if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
			d28 = ps.OverlayValues[28]
		}
		if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
			d29 = ps.OverlayValues[29]
		}
		if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
			d30 = ps.OverlayValues[30]
		}
		if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
			d31 = ps.OverlayValues[31]
		}
		if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
			d32 = ps.OverlayValues[32]
		}
		if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
			d33 = ps.OverlayValues[33]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
			d45 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
			d46 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
			d47 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
			d48 = ps.OverlayValues[48]
		}
		if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
			d49 = ps.OverlayValues[49]
		}
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
			d50 = ps.OverlayValues[50]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
			d56 = ps.OverlayValues[56]
		}
		if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
			d57 = ps.OverlayValues[57]
		}
		if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
			d58 = ps.OverlayValues[58]
		}
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 160 && ps.OverlayValues[160].Loc != scm.LocNone {
			d160 = ps.OverlayValues[160]
		}
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d56)
		ctx.EnsureDesc(&d56)
		var d162 scm.JITValueDesc
		if d56.Loc == scm.LocImm {
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d56.Imm.Int()))))}
		} else {
			r51 := ctx.AllocReg()
			ctx.EmitMovRegReg(r51, d56.Reg)
			d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d162)
		}
		var d163 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r52 := ctx.AllocReg()
			ctx.EmitMovRegMem(r52, thisptr.Reg, off)
			d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
			ctx.BindReg(r52, &d163)
		}
		ctx.EnsureDesc(&d162)
		resultTarget164 := false
		_ = resultTarget164
		ctx.EnsureDesc(&d163)
		ctx.EnsureDescsTogether(&d162, &d163)
		var d165 scm.JITValueDesc
		if d162.Loc == scm.LocImm && d163.Loc == scm.LocImm {
			d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() + d163.Imm.Int())}
		} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
			var r53 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d162.Reg {
				r53 = result.Reg2
				resultTarget164 = true
			} else {
				r53 = ctx.AllocRegExcept(d162.Reg)
			}
			ctx.EmitMovRegReg(r53, d162.Reg)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d165)
		} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d163.Reg}
			ctx.BindReg(d163.Reg, &d165)
		} else if d162.Loc == scm.LocImm {
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d163.Reg {
				scratch = result.Reg2
				resultTarget164 = true
			} else {
				scratch = ctx.AllocRegExcept(d163.Reg)
			}
			ctx.EmitMovRegImm64(scratch, uint64(d162.Imm.Int()))
			ctx.EmitAddInt64(scratch, d163.Reg)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d165)
		} else if d163.Loc == scm.LocImm {
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d162.Reg {
				scratch = result.Reg2
				resultTarget164 = true
			} else {
				scratch = ctx.AllocRegExcept(d162.Reg)
			}
			ctx.EmitMovRegReg(scratch, d162.Reg)
			if d163.Imm.Int() >= -2147483648 && d163.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d163.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d165)
		} else {
			var r54 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d162.Reg && result.Reg2 != d163.Reg {
				r54 = result.Reg2
				resultTarget164 = true
			} else {
				r54 = ctx.AllocRegExcept(d162.Reg, d163.Reg)
			}
			ctx.EmitMovRegReg(r54, d162.Reg)
			ctx.EmitAddInt64(r54, d163.Reg)
			d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d165)
		}
		if d165.Loc == scm.LocReg && d162.Loc == scm.LocReg && d165.Reg == d162.Reg {
			ctx.TransferReg(d162.Reg)
			d162.Loc = scm.LocNone
		}
		if resultTarget164 && d165.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d162)
		ctx.FreeDesc(&d163)
		ctx.EnsureDesc(&d165)
		d166 = result
		ctx.EnsureDesc(&d165)
		ctx.EmitMakeInt(d166, d165)
		if d165.Loc == scm.LocReg {
			ctx.FreeReg(d165.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps167 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps167)
	ctx.MarkLabel(lbl0)
	ctx.ResolveFixups()
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
	}
	ctx.EndStandaloneFrame(standaloneFrame)
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

func (s *StorageInt) GetCachedReader() ColumnReader { return s.storageJITFunctions.reader(s) }

func (s *StorageInt) GetValue(i uint32) scm.Scmer {
	if !s.hasNull {
		return scm.NewInt(int64(s.GetValueUInt(i)) + s.offset)
	}
	v := s.GetValueUInt(i)
	if v == s.null {
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
	bitsize := uint(s.bitsize)
	bitpos := uint(i) * bitsize
	chunk := bitpos / 64
	offset := bitpos % 64
	// StorageInt keeps one trailing sentinel chunk. Variable shifts by 64
	// produce zero, so reading both words removes the only decode branch while
	// preserving the aligned case where offset is zero.
	v := s.chunk[chunk]<<offset | s.chunk[chunk+1]>>(64-offset)
	return uint64(v) >> (64 - bitsize) // shift right without sign
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
// recid. It keeps one monotonically increasing bit position, which minimizes
// loop-carried state while constant division and modulo by 64 lower to shifts
// and masks in both Go and the generated reader.
//
//jitgen:control-flow-stable recid count target/1 stride
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
	if bitsize == 8 || bitsize == 16 || bitsize == 32 || bitsize == 64 {
		// Aligned encodings never straddle a chunk. Keeping this source-level
		// shape separate lets jitgen fold bitsize and discard the generic
		// overflow edge, while the ordinary Go implementation benefits from the
		// same single-word decode.
		idx := 0
		for k := uint32(0); k < count; k++ {
			bitpos := uint(recid+k) * bitsize
			raw := chunk[bitpos/64] << (bitpos % 64) >> (64 - bitsize)
			if hasNull && raw == null {
				target[idx] = scm.NewNil()
			} else {
				target[idx] = scm.NewInt(int64(raw) + offset)
			}
			idx += stride
		}
		return
	}

	bitpos := uint(recid) * bitsize
	endBit := bitpos + uint(count)*bitsize
	idx := 0
	for bitpos < endBit {
		chunkIdx := bitpos / 64
		bitOff := bitpos % 64
		// The trailing sentinel chunk makes the second load valid even when the
		// value stays within one word. Removing the data-dependent straddle
		// branch is cheaper than conditionally avoiding that sequential load.
		v := chunk[chunkIdx]<<bitOff | chunk[chunkIdx+1]>>(64-bitOff)
		raw := v >> (64 - bitsize)
		if hasNull && raw == null {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(int64(raw) + offset)
		}
		idx += stride
		bitpos += bitsize
	}
}

// GetValueMulti gathers values at arbitrary recids. An explicit index loop is
// intentional: unlike a Go range loop, its SSA induction variable starts at
// zero and needs no synthetic -1/+1 state. That smaller live set materially
// improves the generated one-pass JIT loop without changing the Go result.
//
//jitgen:control-flow-stable recids/2 target/1 stride
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
	if bitsize == 8 || bitsize == 16 || bitsize == 32 || bitsize == 64 {
		idx := 0
		for _, recid := range recids {
			bitpos := uint(recid) * bitsize
			raw := chunk[bitpos/64] << (bitpos % 64) >> (64 - bitsize)
			if hasNull && raw == null {
				target[idx] = scm.NewNil()
			} else {
				target[idx] = scm.NewInt(int64(raw) + offset)
			}
			idx += stride
		}
		return
	}

	idx := 0
	for recidIndex := 0; recidIndex < len(recids); recidIndex++ {
		recid := recids[recidIndex]
		bitpos := uint(recid) * bitsize
		chunkIdx := bitpos / 64
		bitOff := bitpos % 64
		v := chunk[chunkIdx]<<bitOff | chunk[chunkIdx+1]>>(64-bitOff)
		raw := v >> (64 - bitsize)
		if hasNull && raw == null {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(int64(raw) + offset)
		}
		idx += stride
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
	s.storageJITFunctions.finish(s)
}
func (s *StorageInt) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

func (s *StorageInt) DistinctCount() uint { return uint(s.count) }
