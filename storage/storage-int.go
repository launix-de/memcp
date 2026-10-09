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
	var d5 scm.JITValueDesc
	_ = d5
	var d6 scm.JITValueDesc
	_ = d6
	var d7 scm.JITValueDesc
	_ = d7
	var d8 scm.JITValueDesc
	_ = d8
	var d9 scm.JITValueDesc
	_ = d9
	var d10 scm.JITValueDesc
	_ = d10
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
	var d101 scm.JITValueDesc
	_ = d101
	var d102 scm.JITValueDesc
	_ = d102
	var d103 scm.JITValueDesc
	_ = d103
	var d104 scm.JITValueDesc
	_ = d104
	var d106 scm.JITValueDesc
	_ = d106
	var d107 scm.JITValueDesc
	_ = d107
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
	bbs[0].Render = func() scm.JITValueDesc {
		if bbs[0].Rendered {
			ctx.EmitJmp(lbl1)
			return result
		}
		bbs[0].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl1)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
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
				return bbs[2].Render()
			}
			return bbs[1].Render()
		}
		ctx.EmitCmpRegImm32(d1.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl3)
		if bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
		}
		ctx.FlushRegisterMoves()
		if !bbs[1].Rendered {
			snap2 := d0
			snap3 := d1
			alloc4 := ctx.SnapshotAllocState()
			bbs[1].Render()
			ctx.RestoreAllocState(alloc4)
			d0 = snap2
			d1 = snap3
		}
		if !bbs[2].Rendered {
			return bbs[2].Render()
		}
		return result
		ctx.FreeDesc(&d0)
		return result
	}
	bbs[1].Render = func() scm.JITValueDesc {
		if bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		bbs[1].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl2)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&idxInt)
		d5 = idxInt
		_ = d5
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
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
			r1 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r1, thisptr.Reg, off)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
			ctx.BindReg(r1, &d6)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d6)
		ctx.EnsureDesc(&d6)
		if d6.Loc == scm.LocImm {
			d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d6.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, d6.Reg)
			ctx.EmitShlRegImm8(r2, 56)
			ctx.EmitShrRegImm8(r2, 56)
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d7)
		}
		ctx.FreeDesc(&d6)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDescsTogether(&d5, &d7)
		if d5.Loc == scm.LocImm && d7.Loc == scm.LocImm {
			d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() * d7.Imm.Int())}
		} else if d5.Loc == scm.LocImm {
			ctx.EnsureDesc(&d7)
			scratch := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitMovRegReg(scratch, d7.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d5.Imm.Int())
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d9)
		} else if d7.Loc == scm.LocImm {
			ctx.EnsureDesc(&d5)
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d7.Imm.Int())
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d9)
		} else {
			ctx.EnsureDesc(&d5)
			ctx.SyncDesc(&d7)
			r3 := ctx.AllocRegExcept(d5.Reg, d7.Reg)
			ctx.EmitMovRegReg(r3, d5.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r3, &d7)
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d9)
		}
		if d9.Loc == scm.LocReg && d5.Loc == scm.LocReg && d9.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		if d9.Loc == scm.LocImm {
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() / 64)}
		} else {
			r4 := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(r4, d9.Reg)
			ctx.EmitShrRegImm8(r4, 6)
			d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d10)
		}
		if d10.Loc == scm.LocReg && d9.Loc == scm.LocReg && d10.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		resultTarget11 := false
		_ = resultTarget11
		if d9.Loc == scm.LocImm {
			d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() % 64)}
		} else {
			r5 := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(r5, d9.Reg)
			ctx.EmitAndRegImm32(r5, 63)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d12)
		}
		if d12.Loc == scm.LocReg && d9.Loc == scm.LocReg && d12.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d9)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d13 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r6 := ctx.AllocReg()
			r7 := ctx.AllocRegExcept(r6)
			r8 := ctx.AllocRegExcept(r6, r7)
			off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
			ctx.EmitMovRegMem(r6, thisptr.Reg, off)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off+16)
			d13 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r6, Reg2: r7, Reg3: r8}
			ctx.BindReg(r6, &d13)
			ctx.BindReg(r7, &d13)
			ctx.BindReg(r8, &d13)
			ctx.BindReg(r6, &d13)
			ctx.BindReg(r7, &d13)
			ctx.BindReg(r8, &d13)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		ctx.ReclaimUntrackedRegs()
		d14 = ctx.EmitLoadScalarSliceElement(&d13, &d10, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d12)
		ctx.EnsureDescsTogether(&d14, &d12)
		if d14.Loc == scm.LocImm && d12.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) << uint64(d12.Imm.Int())))}
		} else if d12.Loc == scm.LocImm {
			r9 := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegReg(r9, d14.Reg)
			ctx.EmitShlRegImm8(r9, uint8(d12.Imm.Int()))
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d15)
		} else {
			shiftSrc := d14.Reg
			r10 := ctx.AllocRegExcept(d14.Reg, d12.Reg)
			ctx.EmitMovRegReg(r10, d14.Reg)
			shiftSrc = r10
			ctx.EmitShiftLeft(shiftSrc, d12.Reg, true)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d15)
		}
		if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
			ctx.TransferReg(d14.Reg)
			d14.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d14)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d10)
		if d10.Loc == scm.LocImm {
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(scratch, d10.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d16)
		}
		if d16.Loc == scm.LocReg && d10.Loc == scm.LocReg && d16.Reg == d10.Reg {
			ctx.TransferReg(d10.Reg)
			d10.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d10)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d16)
		ctx.ReclaimUntrackedRegs()
		d17 = ctx.EmitLoadScalarSliceElement(&d13, &d16, 8, scm.TagInt)
		ctx.FreeDesc(&d16)
		ctx.ReclaimUntrackedRegs()
		d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d12)
		ctx.EnsureDescsTogether(&d18, &d12)
		if d18.Loc == scm.LocImm && d12.Loc == scm.LocImm {
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() - d12.Imm.Int())}
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			ctx.EnsureDesc(&d18)
			r11 := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitMovRegReg(r11, d18.Reg)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d19)
		} else if d18.Loc == scm.LocImm {
			ctx.EnsureDesc(&d12)
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d12)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d19)
		} else if d12.Loc == scm.LocImm {
			ctx.EnsureDesc(&d18)
			scratch := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitMovRegReg(scratch, d18.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d12.Imm.Int())
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d19)
		} else {
			ctx.EnsureDesc(&d18)
			ctx.SyncDesc(&d12)
			r12 := ctx.AllocRegExcept(d18.Reg, d12.Reg)
			ctx.EmitMovRegReg(r12, d18.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r12, &d12)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d19)
		}
		if d19.Loc == scm.LocReg && d18.Loc == scm.LocReg && d19.Reg == d18.Reg {
			ctx.TransferReg(d18.Reg)
			d18.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d12)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d19)
		ctx.EnsureDescsTogether(&d17, &d19)
		if d17.Loc == scm.LocImm && d19.Loc == scm.LocImm {
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) >> uint64(d19.Imm.Int())))}
		} else if d19.Loc == scm.LocImm {
			r13 := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(r13, d17.Reg)
			ctx.EmitShrRegImm8(r13, uint8(d19.Imm.Int()))
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d20)
		} else {
			shiftSrc := d17.Reg
			r14 := ctx.AllocRegExcept(d17.Reg, d19.Reg)
			ctx.EmitMovRegReg(r14, d17.Reg)
			shiftSrc = r14
			ctx.EmitShiftRight(shiftSrc, d19.Reg, false)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d20)
		}
		if d20.Loc == scm.LocReg && d17.Loc == scm.LocReg && d20.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.FreeDesc(&d19)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d15)
		ctx.EnsureDesc(&d20)
		if d15.Loc == scm.LocImm && d20.Loc == scm.LocImm {
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() | d20.Imm.Int())}
		} else if d15.Loc == scm.LocImm && d15.Imm.Int() == 0 {
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			ctx.BindReg(d20.Reg, &d21)
		} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
			r15 := ctx.AllocRegExcept(d15.Reg)
			ctx.EmitMovRegReg(r15, d15.Reg)
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d21)
		} else if d15.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d20.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d15.Imm.Int()))
			ctx.EmitOrInt64(scratch, d20.Reg)
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d21)
		} else if d20.Loc == scm.LocImm {
			r16 := ctx.AllocRegExcept(d15.Reg)
			ctx.EmitMovRegReg(r16, d15.Reg)
			if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r16, int32(d20.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d20.Imm.Int()))
				ctx.EmitOrInt64(r16, ctx.ScratchReg)
			}
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d21)
		} else {
			r17 := ctx.AllocRegExcept(d15.Reg, d20.Reg)
			ctx.EmitMovRegReg(r17, d15.Reg)
			ctx.EmitOrInt64(r17, d20.Reg)
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d21)
		}
		if d21.Loc == scm.LocReg && d15.Loc == scm.LocReg && d21.Reg == d15.Reg {
			ctx.TransferReg(d15.Reg)
			d15.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d15)
		ctx.FreeDesc(&d20)
		ctx.ReclaimUntrackedRegs()
		d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d7)
		ctx.EnsureDescsTogether(&d22, &d7)
		if d22.Loc == scm.LocImm && d7.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() - d7.Imm.Int())}
		} else if d7.Loc == scm.LocImm && d7.Imm.Int() == 0 {
			ctx.EnsureDesc(&d22)
			r18 := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(r18, d22.Reg)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d23)
		} else if d22.Loc == scm.LocImm {
			ctx.EnsureDesc(&d7)
			scratch := ctx.AllocRegExcept(d7.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d7)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d23)
		} else if d7.Loc == scm.LocImm {
			ctx.EnsureDesc(&d22)
			scratch := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(scratch, d22.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d7.Imm.Int())
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d23)
		} else {
			ctx.EnsureDesc(&d22)
			ctx.SyncDesc(&d7)
			r19 := ctx.AllocRegExcept(d22.Reg, d7.Reg)
			ctx.EmitMovRegReg(r19, d22.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r19, &d7)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d23)
		}
		if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
			ctx.TransferReg(d22.Reg)
			d22.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d7)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d23)
		ctx.EnsureDescsTogether(&d21, &d23)
		if d21.Loc == scm.LocImm && d23.Loc == scm.LocImm {
			d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d21.Imm.Int()) >> uint64(d23.Imm.Int())))}
		} else if d23.Loc == scm.LocImm {
			r20 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitMovRegReg(r20, d21.Reg)
			ctx.EmitShrRegImm8(r20, uint8(d23.Imm.Int()))
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d24)
		} else {
			shiftSrc := d21.Reg
			r21 := ctx.AllocRegExcept(d21.Reg, d23.Reg)
			ctx.EmitMovRegReg(r21, d21.Reg)
			shiftSrc = r21
			ctx.EmitShiftRight(shiftSrc, d23.Reg, false)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d24)
		}
		if d24.Loc == scm.LocReg && d21.Loc == scm.LocReg && d24.Reg == d21.Reg {
			ctx.TransferReg(d21.Reg)
			d21.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d21)
		ctx.FreeDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		if d24.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d24.Imm.Int()))))}
		} else {
			r22 := ctx.AllocReg()
			ctx.EmitMovRegReg(r22, d24.Reg)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d25)
		}
		ctx.FreeDesc(&d24)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r23 := ctx.AllocReg()
			ctx.EmitMovRegMem(r23, thisptr.Reg, off)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d26)
		}
		ctx.EnsureDesc(&d25)
		resultTarget27 := false
		_ = resultTarget27
		ctx.EnsureDesc(&d26)
		ctx.EnsureDescsTogether(&d25, &d26)
		if d25.Loc == scm.LocImm && d26.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() + d26.Imm.Int())}
		} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
			ctx.EnsureDesc(&d25)
			var r24 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d25.Reg {
				r24 = result.Reg2
				resultTarget27 = true
			} else {
				r24 = ctx.AllocRegExcept(d25.Reg)
			}
			ctx.EmitMovRegReg(r24, d25.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d28)
		} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
			ctx.EnsureDesc(&d26)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			ctx.BindReg(d26.Reg, &d28)
		} else if d25.Loc == scm.LocImm {
			ctx.EnsureDesc(&d26)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d26.Reg {
				scratch = result.Reg2
				resultTarget27 = true
			} else {
				scratch = ctx.AllocRegExcept(d26.Reg)
			}
			ctx.EmitMovRegReg(scratch, d26.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d25.Imm.Int())
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d28)
		} else if d26.Loc == scm.LocImm {
			ctx.EnsureDesc(&d25)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d25.Reg {
				scratch = result.Reg2
				resultTarget27 = true
			} else {
				scratch = ctx.AllocRegExcept(d25.Reg)
			}
			ctx.EmitMovRegReg(scratch, d25.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d26.Imm.Int())
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d28)
		} else {
			ctx.EnsureDesc(&d25)
			ctx.SyncDesc(&d26)
			var r25 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d25.Reg && result.Reg2 != d26.Reg {
				r25 = result.Reg2
				resultTarget27 = true
			} else {
				r25 = ctx.AllocRegExcept(d25.Reg, d26.Reg)
			}
			ctx.EmitMovRegReg(r25, d25.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r25, &d26)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d28)
		}
		if d28.Loc == scm.LocReg && d25.Loc == scm.LocReg && d28.Reg == d25.Reg {
			ctx.TransferReg(d25.Reg)
			d25.Loc = scm.LocNone
		}
		if resultTarget27 && d28.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d25)
		ctx.FreeDesc(&d26)
		ctx.EnsureDesc(&d28)
		d29 = result
		ctx.EnsureDesc(&d28)
		ctx.EmitMakeInt(d29, d28)
		if d28.Loc == scm.LocReg {
			ctx.FreeReg(d28.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[2].Render = func() scm.JITValueDesc {
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
			return result
		}
		bbs[2].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl3)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&thisptr)
		ctx.EnsureDesc(&idxInt)
		d30 = idxInt
		_ = d30
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
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
			r26 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r26, thisptr.Reg, off)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d31)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		if d31.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d31.Imm.Int()))))}
		} else {
			r27 := ctx.AllocReg()
			ctx.EmitMovRegReg(r27, d31.Reg)
			ctx.EmitShlRegImm8(r27, 56)
			ctx.EmitShrRegImm8(r27, 56)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d32)
		}
		ctx.FreeDesc(&d31)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d32)
		ctx.EnsureDescsTogether(&d30, &d32)
		if d30.Loc == scm.LocImm && d32.Loc == scm.LocImm {
			d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() * d32.Imm.Int())}
		} else if d30.Loc == scm.LocImm {
			ctx.EnsureDesc(&d32)
			scratch := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(scratch, d32.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d30.Imm.Int())
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d34)
		} else if d32.Loc == scm.LocImm {
			ctx.EnsureDesc(&d30)
			scratch := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegReg(scratch, d30.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d32.Imm.Int())
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d34)
		} else {
			ctx.EnsureDesc(&d30)
			ctx.SyncDesc(&d32)
			r28 := ctx.AllocRegExcept(d30.Reg, d32.Reg)
			ctx.EmitMovRegReg(r28, d30.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r28, &d32)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d34)
		}
		if d34.Loc == scm.LocReg && d30.Loc == scm.LocReg && d34.Reg == d30.Reg {
			ctx.TransferReg(d30.Reg)
			d30.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		if d34.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() / 64)}
		} else {
			r29 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r29, d34.Reg)
			ctx.EmitShrRegImm8(r29, 6)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d35)
		}
		if d35.Loc == scm.LocReg && d34.Loc == scm.LocReg && d35.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		resultTarget36 := false
		_ = resultTarget36
		if d34.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() % 64)}
		} else {
			r30 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r30, d34.Reg)
			ctx.EmitAndRegImm32(r30, 63)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d37)
		}
		if d37.Loc == scm.LocReg && d34.Loc == scm.LocReg && d37.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d34)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d38 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r31 := ctx.AllocReg()
			r32 := ctx.AllocRegExcept(r31)
			r33 := ctx.AllocRegExcept(r31, r32)
			off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
			ctx.EmitMovRegMem(r31, thisptr.Reg, off)
			ctx.EmitMovRegMem(r32, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r33, thisptr.Reg, off+16)
			d38 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r31, Reg2: r32, Reg3: r33}
			ctx.BindReg(r31, &d38)
			ctx.BindReg(r32, &d38)
			ctx.BindReg(r33, &d38)
			ctx.BindReg(r31, &d38)
			ctx.BindReg(r32, &d38)
			ctx.BindReg(r33, &d38)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		d39 = ctx.EmitLoadScalarSliceElement(&d38, &d35, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d37)
		ctx.EnsureDescsTogether(&d39, &d37)
		if d39.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d39.Imm.Int()) << uint64(d37.Imm.Int())))}
		} else if d37.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r34, d39.Reg)
			ctx.EmitShlRegImm8(r34, uint8(d37.Imm.Int()))
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d40)
		} else {
			shiftSrc := d39.Reg
			r35 := ctx.AllocRegExcept(d39.Reg, d37.Reg)
			ctx.EmitMovRegReg(r35, d39.Reg)
			shiftSrc = r35
			ctx.EmitShiftLeft(shiftSrc, d37.Reg, true)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d40)
		}
		if d40.Loc == scm.LocReg && d39.Loc == scm.LocReg && d40.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d35)
		if d35.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(scratch, d35.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		}
		if d41.Loc == scm.LocReg && d35.Loc == scm.LocReg && d41.Reg == d35.Reg {
			ctx.TransferReg(d35.Reg)
			d35.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		d42 = ctx.EmitLoadScalarSliceElement(&d38, &d41, 8, scm.TagInt)
		ctx.FreeDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d37)
		ctx.EnsureDescsTogether(&d43, &d37)
		if d43.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() - d37.Imm.Int())}
		} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
			ctx.EnsureDesc(&d43)
			r36 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r36, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d44)
		} else if d43.Loc == scm.LocImm {
			ctx.EnsureDesc(&d37)
			scratch := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d37)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else if d37.Loc == scm.LocImm {
			ctx.EnsureDesc(&d43)
			scratch := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(scratch, d43.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d37.Imm.Int())
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else {
			ctx.EnsureDesc(&d43)
			ctx.SyncDesc(&d37)
			r37 := ctx.AllocRegExcept(d43.Reg, d37.Reg)
			ctx.EmitMovRegReg(r37, d43.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r37, &d37)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d44)
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d42)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d42, &d44)
		if d42.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d42.Imm.Int()) >> uint64(d44.Imm.Int())))}
		} else if d44.Loc == scm.LocImm {
			r38 := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegReg(r38, d42.Reg)
			ctx.EmitShrRegImm8(r38, uint8(d44.Imm.Int()))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d45)
		} else {
			shiftSrc := d42.Reg
			r39 := ctx.AllocRegExcept(d42.Reg, d44.Reg)
			ctx.EmitMovRegReg(r39, d42.Reg)
			shiftSrc = r39
			ctx.EmitShiftRight(shiftSrc, d44.Reg, false)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d45)
		}
		if d45.Loc == scm.LocReg && d42.Loc == scm.LocReg && d45.Reg == d42.Reg {
			ctx.TransferReg(d42.Reg)
			d42.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d42)
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		ctx.EnsureDesc(&d45)
		if d40.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() | d45.Imm.Int())}
		} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d45.Reg}
			ctx.BindReg(d45.Reg, &d46)
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			r40 := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(r40, d40.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d46)
		} else if d40.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
			ctx.EmitOrInt64(scratch, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d46)
		} else if d45.Loc == scm.LocImm {
			r41 := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(r41, d40.Reg)
			if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r41, int32(d45.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d45.Imm.Int()))
				ctx.EmitOrInt64(r41, ctx.ScratchReg)
			}
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d46)
		} else {
			r42 := ctx.AllocRegExcept(d40.Reg, d45.Reg)
			ctx.EmitMovRegReg(r42, d40.Reg)
			ctx.EmitOrInt64(r42, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d46)
		}
		if d46.Loc == scm.LocReg && d40.Loc == scm.LocReg && d46.Reg == d40.Reg {
			ctx.TransferReg(d40.Reg)
			d40.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d40)
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d32)
		ctx.EnsureDescsTogether(&d47, &d32)
		if d47.Loc == scm.LocImm && d32.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() - d32.Imm.Int())}
		} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
			ctx.EnsureDesc(&d47)
			r43 := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(r43, d47.Reg)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d48)
		} else if d47.Loc == scm.LocImm {
			ctx.EnsureDesc(&d32)
			scratch := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d32)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d48)
		} else if d32.Loc == scm.LocImm {
			ctx.EnsureDesc(&d47)
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(scratch, d47.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d32.Imm.Int())
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d48)
		} else {
			ctx.EnsureDesc(&d47)
			ctx.SyncDesc(&d32)
			r44 := ctx.AllocRegExcept(d47.Reg, d32.Reg)
			ctx.EmitMovRegReg(r44, d47.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r44, &d32)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d48)
		}
		if d48.Loc == scm.LocReg && d47.Loc == scm.LocReg && d48.Reg == d47.Reg {
			ctx.TransferReg(d47.Reg)
			d47.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d32)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d48)
		ctx.EnsureDescsTogether(&d46, &d48)
		if d46.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d48.Imm.Int())))}
		} else if d48.Loc == scm.LocImm {
			r45 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r45, d46.Reg)
			ctx.EmitShrRegImm8(r45, uint8(d48.Imm.Int()))
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d49)
		} else {
			shiftSrc := d46.Reg
			r46 := ctx.AllocRegExcept(d46.Reg, d48.Reg)
			ctx.EmitMovRegReg(r46, d46.Reg)
			shiftSrc = r46
			ctx.EmitShiftRight(shiftSrc, d48.Reg, false)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d49)
		}
		if d49.Loc == scm.LocReg && d46.Loc == scm.LocReg && d49.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.FreeDesc(&d48)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d49)
		ctx.FreeDesc(&idxInt)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
			r47 := ctx.AllocReg()
			ctx.EmitMovRegMem(r47, thisptr.Reg, off)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			ctx.BindReg(r47, &d50)
		}
		ctx.EnsureDesc(&d49)
		ctx.EnsureDesc(&d50)
		ctx.EnsureDescsTogether(&d49, &d50)
		if d49.Loc == scm.LocImm && d50.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d49.Imm.Int()) == uint64(d50.Imm.Int()))}
		} else if d50.Loc == scm.LocImm {
			r48 := ctx.AllocRegExcept(d49.Reg)
			if d50.Imm.Int() >= -2147483648 && d50.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d49.Reg, int32(d50.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d50.Imm.Int()))
				ctx.EmitCmpInt64(d49.Reg, ctx.ScratchReg)
			}
			d51 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r48, Condition: scm.CondEqual}
			ctx.BindReg(r48, &d51)
		} else if d49.Loc == scm.LocImm {
			r49 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d49.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d50.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r49, Condition: scm.CondEqual}
			ctx.BindReg(r49, &d51)
		} else {
			r50 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitCmpInt64(d49.Reg, d50.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r50, Condition: scm.CondEqual}
			ctx.BindReg(r50, &d51)
		}
		ctx.FreeDesc(&d50)
		d52 = d51
		ctx.EnsureDesc(&d52)
		if d52.Loc != scm.LocImm && d52.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d52.Loc == scm.LocImm {
			if d52.Imm.Bool() {
				return bbs[3].Render()
			}
			return bbs[4].Render()
		}
		ctx.EmitJump(d52.Condition, lbl4)
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
		}
		ctx.FreeDesc(&d51)
		ctx.FlushRegisterMoves()
		if !bbs[4].Rendered {
			snap53 := d0
			snap54 := d1
			snap55 := d5
			snap56 := d6
			snap57 := d7
			snap58 := d8
			snap59 := d9
			snap60 := d10
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
			snap76 := d28
			snap77 := d29
			snap78 := d30
			snap79 := d31
			snap80 := d32
			snap81 := d33
			snap82 := d34
			snap83 := d35
			snap84 := d37
			snap85 := d38
			snap86 := d39
			snap87 := d40
			snap88 := d41
			snap89 := d42
			snap90 := d43
			snap91 := d44
			snap92 := d45
			snap93 := d46
			snap94 := d47
			snap95 := d48
			snap96 := d49
			snap97 := d50
			snap98 := d51
			snap99 := d52
			alloc100 := ctx.SnapshotAllocState()
			bbs[4].Render()
			ctx.RestoreAllocState(alloc100)
			d0 = snap53
			d1 = snap54
			d5 = snap55
			d6 = snap56
			d7 = snap57
			d8 = snap58
			d9 = snap59
			d10 = snap60
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
			d28 = snap76
			d29 = snap77
			d30 = snap78
			d31 = snap79
			d32 = snap80
			d33 = snap81
			d34 = snap82
			d35 = snap83
			d37 = snap84
			d38 = snap85
			d39 = snap86
			d40 = snap87
			d41 = snap88
			d42 = snap89
			d43 = snap90
			d44 = snap91
			d45 = snap92
			d46 = snap93
			d47 = snap94
			d48 = snap95
			d49 = snap96
			d50 = snap97
			d51 = snap98
			d52 = snap99
		}
		if !bbs[3].Rendered {
			return bbs[3].Render()
		}
		return result
		return result
	}
	bbs[3].Render = func() scm.JITValueDesc {
		if bbs[3].Rendered {
			ctx.EmitJmp(lbl4)
			return result
		}
		bbs[3].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl4)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d102 = result
		ctx.EnsureDesc(&d101)
		if d101.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d101, &d102)
		} else {
			switch d101.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d102, d101)
			case scm.TagInt:
				ctx.EmitMakeInt(d102, d101)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d102, d101)
			case scm.TagNil:
				ctx.EmitMakeNil(d102)
			default:
				ctx.EmitMovPairToResult(&d101, &d102)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[4].Render = func() scm.JITValueDesc {
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		bbs[4].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl5)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d49)
		ctx.EnsureDesc(&d49)
		if d49.Loc == scm.LocImm {
			d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d49.Imm.Int()))))}
		} else {
			r51 := ctx.AllocReg()
			ctx.EmitMovRegReg(r51, d49.Reg)
			d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d103)
		}
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r52 := ctx.AllocReg()
			ctx.EmitMovRegMem(r52, thisptr.Reg, off)
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
			ctx.BindReg(r52, &d104)
		}
		ctx.EnsureDesc(&d103)
		resultTarget105 := false
		_ = resultTarget105
		ctx.EnsureDesc(&d104)
		ctx.EnsureDescsTogether(&d103, &d104)
		if d103.Loc == scm.LocImm && d104.Loc == scm.LocImm {
			d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() + d104.Imm.Int())}
		} else if d104.Loc == scm.LocImm && d104.Imm.Int() == 0 {
			ctx.EnsureDesc(&d103)
			var r53 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d103.Reg {
				r53 = result.Reg2
				resultTarget105 = true
			} else {
				r53 = ctx.AllocRegExcept(d103.Reg)
			}
			ctx.EmitMovRegReg(r53, d103.Reg)
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d106)
		} else if d103.Loc == scm.LocImm && d103.Imm.Int() == 0 {
			ctx.EnsureDesc(&d104)
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d104.Reg}
			ctx.BindReg(d104.Reg, &d106)
		} else if d103.Loc == scm.LocImm {
			ctx.EnsureDesc(&d104)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d104.Reg {
				scratch = result.Reg2
				resultTarget105 = true
			} else {
				scratch = ctx.AllocRegExcept(d104.Reg)
			}
			ctx.EmitMovRegReg(scratch, d104.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d103.Imm.Int())
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d106)
		} else if d104.Loc == scm.LocImm {
			ctx.EnsureDesc(&d103)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d103.Reg {
				scratch = result.Reg2
				resultTarget105 = true
			} else {
				scratch = ctx.AllocRegExcept(d103.Reg)
			}
			ctx.EmitMovRegReg(scratch, d103.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d104.Imm.Int())
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d106)
		} else {
			ctx.EnsureDesc(&d103)
			ctx.SyncDesc(&d104)
			var r54 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d103.Reg && result.Reg2 != d104.Reg {
				r54 = result.Reg2
				resultTarget105 = true
			} else {
				r54 = ctx.AllocRegExcept(d103.Reg, d104.Reg)
			}
			ctx.EmitMovRegReg(r54, d103.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r54, &d104)
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d106)
		}
		if d106.Loc == scm.LocReg && d103.Loc == scm.LocReg && d106.Reg == d103.Reg {
			ctx.TransferReg(d103.Reg)
			d103.Loc = scm.LocNone
		}
		if resultTarget105 && d106.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d103)
		ctx.FreeDesc(&d104)
		ctx.EnsureDesc(&d106)
		d107 = result
		ctx.EnsureDesc(&d106)
		ctx.EmitMakeInt(d107, d106)
		if d106.Loc == scm.LocReg {
			ctx.FreeReg(d106.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	_ = bbs[0].Render()
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
	return uint(unsafe.Sizeof(*s)) + 8*uint(cap(s.chunk))
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
