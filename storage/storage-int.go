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
	var d18 scm.JITValueDesc
	_ = d18
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
	var d34 scm.JITValueDesc
	_ = d34
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
	var d43 scm.JITValueDesc
	_ = d43
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
	var d157 scm.JITValueDesc
	_ = d157
	var d158 scm.JITValueDesc
	_ = d158
	var d159 scm.JITValueDesc
	_ = d159
	var d160 scm.JITValueDesc
	_ = d160
	var d161 scm.JITValueDesc
	_ = d161
	var d162 scm.JITValueDesc
	_ = d162
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
	r0 := ctx.AllocReg()
	r1 := ctx.AllocRegExcept(r0)
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
			r2 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r2, thisptr.Reg, off)
			d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			ctx.BindReg(r2, &d0)
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
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r3, thisptr.Reg, off)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d13)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d13)
		ctx.EnsureDesc(&d13)
		var d14 scm.JITValueDesc
		if d13.Loc == scm.LocImm {
			d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d13.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d13.Reg)
			ctx.EmitShlRegImm8(r4, 56)
			ctx.EmitShrRegImm8(r4, 56)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d14)
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
			r5 := ctx.AllocRegExcept(d12.Reg, d14.Reg)
			ctx.EmitMovRegReg(r5, d12.Reg)
			ctx.EmitImulInt64(r5, d14.Reg)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d16)
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
			r6 := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(r6, d16.Reg)
			ctx.EmitShrRegImm8(r6, 6)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d17)
		}
		if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
			ctx.TransferReg(d16.Reg)
			d16.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d16)
		var d18 scm.JITValueDesc
		if d16.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() % 64)}
		} else {
			r7 := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(r7, d16.Reg)
			ctx.EmitAndRegImm32(r7, 63)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d18)
		}
		if d18.Loc == scm.LocReg && d16.Loc == scm.LocReg && d18.Reg == d16.Reg {
			ctx.TransferReg(d16.Reg)
			d16.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d16)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d19 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d19 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r8 := ctx.AllocReg()
			r9 := ctx.AllocRegExcept(r8)
			r10 := ctx.AllocRegExcept(r8, r9)
			off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
			ctx.EmitMovRegMem(r8, thisptr.Reg, off)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r10, thisptr.Reg, off+16)
			d19 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r8, Reg2: r9, Reg3: r10}
			ctx.BindReg(r8, &d19)
			ctx.BindReg(r9, &d19)
			ctx.BindReg(r10, &d19)
			ctx.BindReg(r8, &d19)
			ctx.BindReg(r9, &d19)
			ctx.BindReg(r10, &d19)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.ReclaimUntrackedRegs()
		d20 = ctx.EmitLoadScalarSliceElement(&d19, &d17, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d20)
		ctx.EnsureDesc(&d18)
		ctx.EnsureDescsTogether(&d20, &d18)
		var d21 scm.JITValueDesc
		if d20.Loc == scm.LocImm && d18.Loc == scm.LocImm {
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d20.Imm.Int()) << uint64(d18.Imm.Int())))}
		} else if d18.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d20.Reg)
			ctx.EmitMovRegReg(r11, d20.Reg)
			ctx.EmitShlRegImm8(r11, uint8(d18.Imm.Int()))
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d21)
		} else {
			{
				shiftSrc := d20.Reg
				r12 := ctx.AllocRegExcept(d20.Reg, d18.Reg)
				ctx.EmitMovRegReg(r12, d20.Reg)
				shiftSrc = r12
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d18.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d18.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d21)
			}
		}
		if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
			ctx.TransferReg(d20.Reg)
			d20.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d20)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d17)
		var d22 scm.JITValueDesc
		if d17.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(scratch, d17.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d22)
		}
		if d22.Loc == scm.LocReg && d17.Loc == scm.LocReg && d22.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.ReclaimUntrackedRegs()
		d23 = ctx.EmitLoadScalarSliceElement(&d19, &d22, 8, scm.TagInt)
		ctx.FreeDesc(&d22)
		ctx.ReclaimUntrackedRegs()
		d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d18)
		ctx.EnsureDescsTogether(&d24, &d18)
		var d25 scm.JITValueDesc
		if d24.Loc == scm.LocImm && d18.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() - d18.Imm.Int())}
		} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
			r13 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r13, d24.Reg)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d25)
		} else if d24.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
			ctx.EmitSubInt64(scratch, d18.Reg)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d25)
		} else if d18.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(scratch, d24.Reg)
			if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d18.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d25)
		} else {
			r14 := ctx.AllocRegExcept(d24.Reg, d18.Reg)
			ctx.EmitMovRegReg(r14, d24.Reg)
			ctx.EmitSubInt64(r14, d18.Reg)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d25)
		}
		if d25.Loc == scm.LocReg && d24.Loc == scm.LocReg && d25.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d18)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d23)
		ctx.EnsureDesc(&d25)
		ctx.EnsureDescsTogether(&d23, &d25)
		var d26 scm.JITValueDesc
		if d23.Loc == scm.LocImm && d25.Loc == scm.LocImm {
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d23.Imm.Int()) >> uint64(d25.Imm.Int())))}
		} else if d25.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d23.Reg)
			ctx.EmitMovRegReg(r15, d23.Reg)
			ctx.EmitShrRegImm8(r15, uint8(d25.Imm.Int()))
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d26)
		} else {
			{
				shiftSrc := d23.Reg
				r16 := ctx.AllocRegExcept(d23.Reg, d25.Reg)
				ctx.EmitMovRegReg(r16, d23.Reg)
				shiftSrc = r16
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d25.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d25.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d25.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d26)
			}
		}
		if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
			ctx.TransferReg(d23.Reg)
			d23.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d23)
		ctx.FreeDesc(&d25)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d26)
		var d27 scm.JITValueDesc
		if d21.Loc == scm.LocImm && d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() | d26.Imm.Int())}
		} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			ctx.BindReg(d26.Reg, &d27)
		} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
			r17 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitMovRegReg(r17, d21.Reg)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d27)
		} else if d21.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
			ctx.EmitOrInt64(scratch, d26.Reg)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d27)
		} else if d26.Loc == scm.LocImm {
			r18 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitMovRegReg(r18, d21.Reg)
			if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r18, int32(d26.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
				ctx.EmitOrInt64(r18, scm.RegR11)
			}
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d27)
		} else {
			r19 := ctx.AllocRegExcept(d21.Reg, d26.Reg)
			ctx.EmitMovRegReg(r19, d21.Reg)
			ctx.EmitOrInt64(r19, d26.Reg)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d27)
		}
		if d27.Loc == scm.LocReg && d21.Loc == scm.LocReg && d27.Reg == d21.Reg {
			ctx.TransferReg(d21.Reg)
			d21.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d21)
		ctx.FreeDesc(&d26)
		ctx.ReclaimUntrackedRegs()
		d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d14)
		ctx.EnsureDescsTogether(&d28, &d14)
		var d29 scm.JITValueDesc
		if d28.Loc == scm.LocImm && d14.Loc == scm.LocImm {
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() - d14.Imm.Int())}
		} else if d14.Loc == scm.LocImm && d14.Imm.Int() == 0 {
			r20 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r20, d28.Reg)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d29)
		} else if d28.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d28.Imm.Int()))
			ctx.EmitSubInt64(scratch, d14.Reg)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d29)
		} else if d14.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(scratch, d28.Reg)
			if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d14.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d29)
		} else {
			r21 := ctx.AllocRegExcept(d28.Reg, d14.Reg)
			ctx.EmitMovRegReg(r21, d28.Reg)
			ctx.EmitSubInt64(r21, d14.Reg)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d29)
		}
		if d29.Loc == scm.LocReg && d28.Loc == scm.LocReg && d29.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d14)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d27)
		ctx.EnsureDesc(&d29)
		ctx.EnsureDescsTogether(&d27, &d29)
		var d30 scm.JITValueDesc
		if d27.Loc == scm.LocImm && d29.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d27.Imm.Int()) >> uint64(d29.Imm.Int())))}
		} else if d29.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d27.Reg)
			ctx.EmitMovRegReg(r22, d27.Reg)
			ctx.EmitShrRegImm8(r22, uint8(d29.Imm.Int()))
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d30)
		} else {
			{
				shiftSrc := d27.Reg
				r23 := ctx.AllocRegExcept(d27.Reg, d29.Reg)
				ctx.EmitMovRegReg(r23, d27.Reg)
				shiftSrc = r23
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d29.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d29.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d29.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d30)
			}
		}
		if d30.Loc == scm.LocReg && d27.Loc == scm.LocReg && d30.Reg == d27.Reg {
			ctx.TransferReg(d27.Reg)
			d27.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d27)
		ctx.FreeDesc(&d29)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d30)
		var d31 scm.JITValueDesc
		if d30.Loc == scm.LocImm {
			d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d30.Imm.Int()))))}
		} else {
			r24 := ctx.AllocReg()
			ctx.EmitMovRegReg(r24, d30.Reg)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d31)
		}
		ctx.FreeDesc(&d30)
		var d32 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r25 := ctx.AllocReg()
			ctx.EmitMovRegMem(r25, thisptr.Reg, off)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			ctx.BindReg(r25, &d32)
		}
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d32)
		ctx.EnsureDescsTogether(&d31, &d32)
		var d33 scm.JITValueDesc
		if d31.Loc == scm.LocImm && d32.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + d32.Imm.Int())}
		} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
			r26 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r26, d31.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d33)
		} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			ctx.BindReg(d32.Reg, &d33)
		} else if d31.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
			ctx.EmitAddInt64(scratch, d32.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d33)
		} else if d32.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(scratch, d31.Reg)
			if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d32.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d33)
		} else {
			r27 := ctx.AllocRegExcept(d31.Reg, d32.Reg)
			ctx.EmitMovRegReg(r27, d31.Reg)
			ctx.EmitAddInt64(r27, d32.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d33)
		}
		if d33.Loc == scm.LocReg && d31.Loc == scm.LocReg && d33.Reg == d31.Reg {
			ctx.TransferReg(d31.Reg)
			d31.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d31)
		ctx.FreeDesc(&d32)
		ctx.EnsureDesc(&d33)
		d34 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d34)
		ctx.BindReg(r1, &d34)
		ctx.EnsureDesc(&d33)
		ctx.EmitMakeInt(d34, d33)
		if d33.Loc == scm.LocReg {
			ctx.FreeReg(d33.Reg)
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
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
			d18 = ps.OverlayValues[18]
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
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&idxInt)
		d35 = idxInt
		_ = d35
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
		var d36 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
			r28 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r28, thisptr.Reg, off)
			d36 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.BindReg(r28, &d36)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d36)
		ctx.EnsureDesc(&d36)
		var d37 scm.JITValueDesc
		if d36.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d36.Imm.Int()))))}
		} else {
			r29 := ctx.AllocReg()
			ctx.EmitMovRegReg(r29, d36.Reg)
			ctx.EmitShlRegImm8(r29, 56)
			ctx.EmitShrRegImm8(r29, 56)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d37)
		}
		ctx.FreeDesc(&d36)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d37)
		ctx.EnsureDescsTogether(&d35, &d37)
		var d39 scm.JITValueDesc
		if d35.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() * d37.Imm.Int())}
		} else if d35.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
			ctx.EmitImulInt64(scratch, d37.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d39)
		} else if d37.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(scratch, d35.Reg)
			if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d37.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d39)
		} else {
			r30 := ctx.AllocRegExcept(d35.Reg, d37.Reg)
			ctx.EmitMovRegReg(r30, d35.Reg)
			ctx.EmitImulInt64(r30, d37.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d39)
		}
		if d39.Loc == scm.LocReg && d35.Loc == scm.LocReg && d39.Reg == d35.Reg {
			ctx.TransferReg(d35.Reg)
			d35.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		var d40 scm.JITValueDesc
		if d39.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
		} else {
			r31 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r31, d39.Reg)
			ctx.EmitShrRegImm8(r31, 6)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d40)
		}
		if d40.Loc == scm.LocReg && d39.Loc == scm.LocReg && d40.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		var d41 scm.JITValueDesc
		if d39.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
		} else {
			r32 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r32, d39.Reg)
			ctx.EmitAndRegImm32(r32, 63)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d41)
		}
		if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d42 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d42 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r33 := ctx.AllocReg()
			r34 := ctx.AllocRegExcept(r33)
			r35 := ctx.AllocRegExcept(r33, r34)
			off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			ctx.EmitMovRegMem(r34, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r35, thisptr.Reg, off+16)
			d42 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r33, Reg2: r34, Reg3: r35}
			ctx.BindReg(r33, &d42)
			ctx.BindReg(r34, &d42)
			ctx.BindReg(r35, &d42)
			ctx.BindReg(r33, &d42)
			ctx.BindReg(r34, &d42)
			ctx.BindReg(r35, &d42)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		d43 = ctx.EmitLoadScalarSliceElement(&d42, &d40, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d41)
		ctx.EnsureDescsTogether(&d43, &d41)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm && d41.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d43.Imm.Int()) << uint64(d41.Imm.Int())))}
		} else if d41.Loc == scm.LocImm {
			r36 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r36, d43.Reg)
			ctx.EmitShlRegImm8(r36, uint8(d41.Imm.Int()))
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d44)
		} else {
			{
				shiftSrc := d43.Reg
				r37 := ctx.AllocRegExcept(d43.Reg, d41.Reg)
				ctx.EmitMovRegReg(r37, d43.Reg)
				shiftSrc = r37
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d41.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d41.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d41.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d44)
			}
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d43)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		ctx.EnsureDesc(&d40)
		var d45 scm.JITValueDesc
		if d40.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(scratch, d40.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d45)
		}
		if d45.Loc == scm.LocReg && d40.Loc == scm.LocReg && d45.Reg == d40.Reg {
			ctx.TransferReg(d40.Reg)
			d40.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		d46 = ctx.EmitLoadScalarSliceElement(&d42, &d45, 8, scm.TagInt)
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d41)
		ctx.EnsureDescsTogether(&d47, &d41)
		var d48 scm.JITValueDesc
		if d47.Loc == scm.LocImm && d41.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() - d41.Imm.Int())}
		} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
			r38 := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(r38, d47.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d48)
		} else if d47.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
			ctx.EmitSubInt64(scratch, d41.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d48)
		} else if d41.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(scratch, d47.Reg)
			if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d41.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d48)
		} else {
			r39 := ctx.AllocRegExcept(d47.Reg, d41.Reg)
			ctx.EmitMovRegReg(r39, d47.Reg)
			ctx.EmitSubInt64(r39, d41.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d48)
		}
		if d48.Loc == scm.LocReg && d47.Loc == scm.LocReg && d48.Reg == d47.Reg {
			ctx.TransferReg(d47.Reg)
			d47.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d48)
		ctx.EnsureDescsTogether(&d46, &d48)
		var d49 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d48.Imm.Int())))}
		} else if d48.Loc == scm.LocImm {
			r40 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r40, d46.Reg)
			ctx.EmitShrRegImm8(r40, uint8(d48.Imm.Int()))
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d49)
		} else {
			{
				shiftSrc := d46.Reg
				r41 := ctx.AllocRegExcept(d46.Reg, d48.Reg)
				ctx.EmitMovRegReg(r41, d46.Reg)
				shiftSrc = r41
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d48.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d48.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d48.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d49)
			}
		}
		if d49.Loc == scm.LocReg && d46.Loc == scm.LocReg && d49.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.FreeDesc(&d48)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d49)
		var d50 scm.JITValueDesc
		if d44.Loc == scm.LocImm && d49.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() | d49.Imm.Int())}
		} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			ctx.BindReg(d49.Reg, &d50)
		} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
			r42 := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(r42, d44.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d50)
		} else if d44.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
			ctx.EmitOrInt64(scratch, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d50)
		} else if d49.Loc == scm.LocImm {
			r43 := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(r43, d44.Reg)
			if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r43, int32(d49.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
				ctx.EmitOrInt64(r43, scm.RegR11)
			}
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d50)
		} else {
			r44 := ctx.AllocRegExcept(d44.Reg, d49.Reg)
			ctx.EmitMovRegReg(r44, d44.Reg)
			ctx.EmitOrInt64(r44, d49.Reg)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d50)
		}
		if d50.Loc == scm.LocReg && d44.Loc == scm.LocReg && d50.Reg == d44.Reg {
			ctx.TransferReg(d44.Reg)
			d44.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d44)
		ctx.FreeDesc(&d49)
		ctx.ReclaimUntrackedRegs()
		d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d37)
		ctx.EnsureDescsTogether(&d51, &d37)
		var d52 scm.JITValueDesc
		if d51.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() - d37.Imm.Int())}
		} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
			r45 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r45, d51.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d52)
		} else if d51.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d51.Imm.Int()))
			ctx.EmitSubInt64(scratch, d37.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		} else if d37.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(scratch, d51.Reg)
			if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d37.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		} else {
			r46 := ctx.AllocRegExcept(d51.Reg, d37.Reg)
			ctx.EmitMovRegReg(r46, d51.Reg)
			ctx.EmitSubInt64(r46, d37.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d52)
		}
		if d52.Loc == scm.LocReg && d51.Loc == scm.LocReg && d52.Reg == d51.Reg {
			ctx.TransferReg(d51.Reg)
			d51.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d50)
		ctx.EnsureDesc(&d52)
		ctx.EnsureDescsTogether(&d50, &d52)
		var d53 scm.JITValueDesc
		if d50.Loc == scm.LocImm && d52.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) >> uint64(d52.Imm.Int())))}
		} else if d52.Loc == scm.LocImm {
			r47 := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegReg(r47, d50.Reg)
			ctx.EmitShrRegImm8(r47, uint8(d52.Imm.Int()))
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d53)
		} else {
			{
				shiftSrc := d50.Reg
				r48 := ctx.AllocRegExcept(d50.Reg, d52.Reg)
				ctx.EmitMovRegReg(r48, d50.Reg)
				shiftSrc = r48
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d52.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d52.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d52.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d53)
			}
		}
		if d53.Loc == scm.LocReg && d50.Loc == scm.LocReg && d53.Reg == d50.Reg {
			ctx.TransferReg(d50.Reg)
			d50.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d50)
		ctx.FreeDesc(&d52)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.StabilizeDescForControlFlow(&d53)
		ctx.FreeDesc(&idxInt)
		var d54 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
			r49 := ctx.AllocReg()
			ctx.EmitMovRegMem(r49, thisptr.Reg, off)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
			ctx.BindReg(r49, &d54)
		}
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d54)
		ctx.EnsureDescsTogether(&d53, &d54)
		var d55 scm.JITValueDesc
		if d53.Loc == scm.LocImm && d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d53.Imm.Int()) == uint64(d54.Imm.Int()))}
		} else if d54.Loc == scm.LocImm {
			r50 := ctx.AllocRegExcept(d53.Reg)
			if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d53.Reg, int32(d54.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d54.Imm.Int()))
				ctx.EmitCmpInt64(d53.Reg, scm.RegR11)
			}
			d55 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r50, Condition: scm.CondEqual}
			ctx.BindReg(r50, &d55)
		} else if d53.Loc == scm.LocImm {
			r51 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d54.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r51, Condition: scm.CondEqual}
			ctx.BindReg(r51, &d55)
		} else {
			r52 := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitCmpInt64(d53.Reg, d54.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r52, Condition: scm.CondEqual}
			ctx.BindReg(r52, &d55)
		}
		ctx.FreeDesc(&d54)
		d56 = d55
		ctx.EnsureDesc(&d56)
		if d56.Loc != scm.LocImm && d56.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d56.Loc == scm.LocImm {
			if d56.Imm.Bool() {
				if ps.General {
				}
				ps57 := scm.PhiState{General: ps.General}
				ps57.OverlayValues = make([]scm.JITValueDesc, 57)
				ps57.OverlayValues[0] = d0
				ps57.OverlayValues[1] = d1
				ps57.OverlayValues[12] = d12
				ps57.OverlayValues[13] = d13
				ps57.OverlayValues[14] = d14
				ps57.OverlayValues[15] = d15
				ps57.OverlayValues[16] = d16
				ps57.OverlayValues[17] = d17
				ps57.OverlayValues[18] = d18
				ps57.OverlayValues[19] = d19
				ps57.OverlayValues[20] = d20
				ps57.OverlayValues[21] = d21
				ps57.OverlayValues[22] = d22
				ps57.OverlayValues[23] = d23
				ps57.OverlayValues[24] = d24
				ps57.OverlayValues[25] = d25
				ps57.OverlayValues[26] = d26
				ps57.OverlayValues[27] = d27
				ps57.OverlayValues[28] = d28
				ps57.OverlayValues[29] = d29
				ps57.OverlayValues[30] = d30
				ps57.OverlayValues[31] = d31
				ps57.OverlayValues[32] = d32
				ps57.OverlayValues[33] = d33
				ps57.OverlayValues[34] = d34
				ps57.OverlayValues[35] = d35
				ps57.OverlayValues[36] = d36
				ps57.OverlayValues[37] = d37
				ps57.OverlayValues[38] = d38
				ps57.OverlayValues[39] = d39
				ps57.OverlayValues[40] = d40
				ps57.OverlayValues[41] = d41
				ps57.OverlayValues[42] = d42
				ps57.OverlayValues[43] = d43
				ps57.OverlayValues[44] = d44
				ps57.OverlayValues[45] = d45
				ps57.OverlayValues[46] = d46
				ps57.OverlayValues[47] = d47
				ps57.OverlayValues[48] = d48
				ps57.OverlayValues[49] = d49
				ps57.OverlayValues[50] = d50
				ps57.OverlayValues[51] = d51
				ps57.OverlayValues[52] = d52
				ps57.OverlayValues[53] = d53
				ps57.OverlayValues[54] = d54
				ps57.OverlayValues[55] = d55
				ps57.OverlayValues[56] = d56
				return bbs[3].RenderPS(ps57)
			}
			if ps.General {
			}
			ps58 := scm.PhiState{General: ps.General}
			ps58.OverlayValues = make([]scm.JITValueDesc, 57)
			ps58.OverlayValues[0] = d0
			ps58.OverlayValues[1] = d1
			ps58.OverlayValues[12] = d12
			ps58.OverlayValues[13] = d13
			ps58.OverlayValues[14] = d14
			ps58.OverlayValues[15] = d15
			ps58.OverlayValues[16] = d16
			ps58.OverlayValues[17] = d17
			ps58.OverlayValues[18] = d18
			ps58.OverlayValues[19] = d19
			ps58.OverlayValues[20] = d20
			ps58.OverlayValues[21] = d21
			ps58.OverlayValues[22] = d22
			ps58.OverlayValues[23] = d23
			ps58.OverlayValues[24] = d24
			ps58.OverlayValues[25] = d25
			ps58.OverlayValues[26] = d26
			ps58.OverlayValues[27] = d27
			ps58.OverlayValues[28] = d28
			ps58.OverlayValues[29] = d29
			ps58.OverlayValues[30] = d30
			ps58.OverlayValues[31] = d31
			ps58.OverlayValues[32] = d32
			ps58.OverlayValues[33] = d33
			ps58.OverlayValues[34] = d34
			ps58.OverlayValues[35] = d35
			ps58.OverlayValues[36] = d36
			ps58.OverlayValues[37] = d37
			ps58.OverlayValues[38] = d38
			ps58.OverlayValues[39] = d39
			ps58.OverlayValues[40] = d40
			ps58.OverlayValues[41] = d41
			ps58.OverlayValues[42] = d42
			ps58.OverlayValues[43] = d43
			ps58.OverlayValues[44] = d44
			ps58.OverlayValues[45] = d45
			ps58.OverlayValues[46] = d46
			ps58.OverlayValues[47] = d47
			ps58.OverlayValues[48] = d48
			ps58.OverlayValues[49] = d49
			ps58.OverlayValues[50] = d50
			ps58.OverlayValues[51] = d51
			ps58.OverlayValues[52] = d52
			ps58.OverlayValues[53] = d53
			ps58.OverlayValues[54] = d54
			ps58.OverlayValues[55] = d55
			ps58.OverlayValues[56] = d56
			return bbs[4].RenderPS(ps58)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitJump(d56.Condition, lbl4)
		snap59 := d0
		snap60 := d1
		snap61 := d12
		snap62 := d13
		snap63 := d14
		snap64 := d15
		snap65 := d16
		snap66 := d17
		snap67 := d18
		snap68 := d19
		snap69 := d20
		snap70 := d21
		snap71 := d22
		snap72 := d23
		snap73 := d24
		snap74 := d25
		snap75 := d26
		snap76 := d27
		snap77 := d28
		snap78 := d29
		snap79 := d30
		snap80 := d31
		snap81 := d32
		snap82 := d33
		snap83 := d34
		snap84 := d35
		snap85 := d36
		snap86 := d37
		snap87 := d38
		snap88 := d39
		snap89 := d40
		snap90 := d41
		snap91 := d42
		snap92 := d43
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
		alloc106 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc106)
		d0 = snap59
		d1 = snap60
		d12 = snap61
		d13 = snap62
		d14 = snap63
		d15 = snap64
		d16 = snap65
		d17 = snap66
		d18 = snap67
		d19 = snap68
		d20 = snap69
		d21 = snap70
		d22 = snap71
		d23 = snap72
		d24 = snap73
		d25 = snap74
		d26 = snap75
		d27 = snap76
		d28 = snap77
		d29 = snap78
		d30 = snap79
		d31 = snap80
		d32 = snap81
		d33 = snap82
		d34 = snap83
		d35 = snap84
		d36 = snap85
		d37 = snap86
		d38 = snap87
		d39 = snap88
		d40 = snap89
		d41 = snap90
		d42 = snap91
		d43 = snap92
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
		ctx.RestoreAllocState(alloc106)
		d0 = snap59
		d1 = snap60
		d12 = snap61
		d13 = snap62
		d14 = snap63
		d15 = snap64
		d16 = snap65
		d17 = snap66
		d18 = snap67
		d19 = snap68
		d20 = snap69
		d21 = snap70
		d22 = snap71
		d23 = snap72
		d24 = snap73
		d25 = snap74
		d26 = snap75
		d27 = snap76
		d28 = snap77
		d29 = snap78
		d30 = snap79
		d31 = snap80
		d32 = snap81
		d33 = snap82
		d34 = snap83
		d35 = snap84
		d36 = snap85
		d37 = snap86
		d38 = snap87
		d39 = snap88
		d40 = snap89
		d41 = snap90
		d42 = snap91
		d43 = snap92
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
		ps107 := scm.PhiState{General: true}
		ps107.OverlayValues = make([]scm.JITValueDesc, 57)
		ps107.OverlayValues[0] = d0
		ps107.OverlayValues[1] = d1
		ps107.OverlayValues[12] = d12
		ps107.OverlayValues[13] = d13
		ps107.OverlayValues[14] = d14
		ps107.OverlayValues[15] = d15
		ps107.OverlayValues[16] = d16
		ps107.OverlayValues[17] = d17
		ps107.OverlayValues[18] = d18
		ps107.OverlayValues[19] = d19
		ps107.OverlayValues[20] = d20
		ps107.OverlayValues[21] = d21
		ps107.OverlayValues[22] = d22
		ps107.OverlayValues[23] = d23
		ps107.OverlayValues[24] = d24
		ps107.OverlayValues[25] = d25
		ps107.OverlayValues[26] = d26
		ps107.OverlayValues[27] = d27
		ps107.OverlayValues[28] = d28
		ps107.OverlayValues[29] = d29
		ps107.OverlayValues[30] = d30
		ps107.OverlayValues[31] = d31
		ps107.OverlayValues[32] = d32
		ps107.OverlayValues[33] = d33
		ps107.OverlayValues[34] = d34
		ps107.OverlayValues[35] = d35
		ps107.OverlayValues[36] = d36
		ps107.OverlayValues[37] = d37
		ps107.OverlayValues[38] = d38
		ps107.OverlayValues[39] = d39
		ps107.OverlayValues[40] = d40
		ps107.OverlayValues[41] = d41
		ps107.OverlayValues[42] = d42
		ps107.OverlayValues[43] = d43
		ps107.OverlayValues[44] = d44
		ps107.OverlayValues[45] = d45
		ps107.OverlayValues[46] = d46
		ps107.OverlayValues[47] = d47
		ps107.OverlayValues[48] = d48
		ps107.OverlayValues[49] = d49
		ps107.OverlayValues[50] = d50
		ps107.OverlayValues[51] = d51
		ps107.OverlayValues[52] = d52
		ps107.OverlayValues[53] = d53
		ps107.OverlayValues[54] = d54
		ps107.OverlayValues[55] = d55
		ps107.OverlayValues[56] = d56
		ps108 := scm.PhiState{General: true}
		ps108.OverlayValues = make([]scm.JITValueDesc, 57)
		ps108.OverlayValues[0] = d0
		ps108.OverlayValues[1] = d1
		ps108.OverlayValues[12] = d12
		ps108.OverlayValues[13] = d13
		ps108.OverlayValues[14] = d14
		ps108.OverlayValues[15] = d15
		ps108.OverlayValues[16] = d16
		ps108.OverlayValues[17] = d17
		ps108.OverlayValues[18] = d18
		ps108.OverlayValues[19] = d19
		ps108.OverlayValues[20] = d20
		ps108.OverlayValues[21] = d21
		ps108.OverlayValues[22] = d22
		ps108.OverlayValues[23] = d23
		ps108.OverlayValues[24] = d24
		ps108.OverlayValues[25] = d25
		ps108.OverlayValues[26] = d26
		ps108.OverlayValues[27] = d27
		ps108.OverlayValues[28] = d28
		ps108.OverlayValues[29] = d29
		ps108.OverlayValues[30] = d30
		ps108.OverlayValues[31] = d31
		ps108.OverlayValues[32] = d32
		ps108.OverlayValues[33] = d33
		ps108.OverlayValues[34] = d34
		ps108.OverlayValues[35] = d35
		ps108.OverlayValues[36] = d36
		ps108.OverlayValues[37] = d37
		ps108.OverlayValues[38] = d38
		ps108.OverlayValues[39] = d39
		ps108.OverlayValues[40] = d40
		ps108.OverlayValues[41] = d41
		ps108.OverlayValues[42] = d42
		ps108.OverlayValues[43] = d43
		ps108.OverlayValues[44] = d44
		ps108.OverlayValues[45] = d45
		ps108.OverlayValues[46] = d46
		ps108.OverlayValues[47] = d47
		ps108.OverlayValues[48] = d48
		ps108.OverlayValues[49] = d49
		ps108.OverlayValues[50] = d50
		ps108.OverlayValues[51] = d51
		ps108.OverlayValues[52] = d52
		ps108.OverlayValues[53] = d53
		ps108.OverlayValues[54] = d54
		ps108.OverlayValues[55] = d55
		ps108.OverlayValues[56] = d56
		snap109 := d0
		snap110 := d1
		snap111 := d12
		snap112 := d13
		snap113 := d14
		snap114 := d15
		snap115 := d16
		snap116 := d17
		snap117 := d18
		snap118 := d19
		snap119 := d20
		snap120 := d21
		snap121 := d22
		snap122 := d23
		snap123 := d24
		snap124 := d25
		snap125 := d26
		snap126 := d27
		snap127 := d28
		snap128 := d29
		snap129 := d30
		snap130 := d31
		snap131 := d32
		snap132 := d33
		snap133 := d34
		snap134 := d35
		snap135 := d36
		snap136 := d37
		snap137 := d38
		snap138 := d39
		snap139 := d40
		snap140 := d41
		snap141 := d42
		snap142 := d43
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
		alloc156 := ctx.SnapshotAllocState()
		if !bbs[4].Rendered {
			bbs[4].RenderPS(ps108)
		}
		ctx.RestoreAllocState(alloc156)
		d0 = snap109
		d1 = snap110
		d12 = snap111
		d13 = snap112
		d14 = snap113
		d15 = snap114
		d16 = snap115
		d17 = snap116
		d18 = snap117
		d19 = snap118
		d20 = snap119
		d21 = snap120
		d22 = snap121
		d23 = snap122
		d24 = snap123
		d25 = snap124
		d26 = snap125
		d27 = snap126
		d28 = snap127
		d29 = snap128
		d30 = snap129
		d31 = snap130
		d32 = snap131
		d33 = snap132
		d34 = snap133
		d35 = snap134
		d36 = snap135
		d37 = snap136
		d38 = snap137
		d39 = snap138
		d40 = snap139
		d41 = snap140
		d42 = snap141
		d43 = snap142
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
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps107)
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
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
			d18 = ps.OverlayValues[18]
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
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
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
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
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
		ctx.ReclaimUntrackedRegs()
		d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d158 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d158)
		ctx.BindReg(r1, &d158)
		ctx.EnsureDesc(&d157)
		if d157.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d157, &d158)
		} else {
			switch d157.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d158, d157)
			case scm.TagInt:
				ctx.EmitMakeInt(d158, d157)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d158, d157)
			case scm.TagNil:
				ctx.EmitMakeNil(d158)
			default:
				ctx.EmitMovPairToResult(&d157, &d158)
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
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
			d18 = ps.OverlayValues[18]
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
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
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
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
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
		if len(ps.OverlayValues) > 157 && ps.OverlayValues[157].Loc != scm.LocNone {
			d157 = ps.OverlayValues[157]
		}
		if len(ps.OverlayValues) > 158 && ps.OverlayValues[158].Loc != scm.LocNone {
			d158 = ps.OverlayValues[158]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d53)
		var d159 scm.JITValueDesc
		if d53.Loc == scm.LocImm {
			d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d53.Imm.Int()))))}
		} else {
			r53 := ctx.AllocReg()
			ctx.EmitMovRegReg(r53, d53.Reg)
			d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d159)
		}
		var d160 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r54 := ctx.AllocReg()
			ctx.EmitMovRegMem(r54, thisptr.Reg, off)
			d160 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
			ctx.BindReg(r54, &d160)
		}
		ctx.EnsureDesc(&d159)
		ctx.EnsureDesc(&d160)
		ctx.EnsureDescsTogether(&d159, &d160)
		var d161 scm.JITValueDesc
		if d159.Loc == scm.LocImm && d160.Loc == scm.LocImm {
			d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() + d160.Imm.Int())}
		} else if d160.Loc == scm.LocImm && d160.Imm.Int() == 0 {
			r55 := ctx.AllocRegExcept(d159.Reg)
			ctx.EmitMovRegReg(r55, d159.Reg)
			d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d161)
		} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
			d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			ctx.BindReg(d160.Reg, &d161)
		} else if d159.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d160.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d159.Imm.Int()))
			ctx.EmitAddInt64(scratch, d160.Reg)
			d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d161)
		} else if d160.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d159.Reg)
			ctx.EmitMovRegReg(scratch, d159.Reg)
			if d160.Imm.Int() >= -2147483648 && d160.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d160.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d161)
		} else {
			r56 := ctx.AllocRegExcept(d159.Reg, d160.Reg)
			ctx.EmitMovRegReg(r56, d159.Reg)
			ctx.EmitAddInt64(r56, d160.Reg)
			d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d161)
		}
		if d161.Loc == scm.LocReg && d159.Loc == scm.LocReg && d161.Reg == d159.Reg {
			ctx.TransferReg(d159.Reg)
			d159.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d159)
		ctx.FreeDesc(&d160)
		ctx.EnsureDesc(&d161)
		d162 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d162)
		ctx.BindReg(r1, &d162)
		ctx.EnsureDesc(&d161)
		ctx.EmitMakeInt(d162, d161)
		if d161.Loc == scm.LocReg {
			ctx.FreeReg(d161.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps163 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps163)
	ctx.MarkLabel(lbl0)
	d164 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d164)
	ctx.BindReg(r1, &d164)
	ctx.EmitMovPairToResult(&d164, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
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
	s.storageJITFunctions.finish(s)
}
func (s *StorageInt) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

func (s *StorageInt) DistinctCount() uint { return uint(s.count) }
