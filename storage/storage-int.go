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
	var d3 scm.JITValueDesc
	_ = d3
	var d4 scm.JITValueDesc
	_ = d4
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
	var d20 scm.JITValueDesc
	_ = d20
	var d21 scm.JITValueDesc
	_ = d21
	var d22 scm.JITValueDesc
	_ = d22
	var d23 scm.JITValueDesc
	_ = d23
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
	var d60 scm.JITValueDesc
	_ = d60
	var d61 scm.JITValueDesc
	_ = d61
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	thisptrPinned := thisptr.Loc == scm.LocReg
	thisptrPinnedReg := thisptr.Reg
	if thisptrPinned {
		ctx.ProtectReg(thisptrPinnedReg)
	}
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
		ctx.EnsureDesc(&idxInt)
		d0 = idxInt
		_ = d0
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl5 := ctx.ReserveLabel()
		_ = lbl5
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl5)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d0)
		var d1 scm.JITValueDesc
		if d0.Loc == scm.LocImm {
			d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, d0.Reg)
			ctx.EmitShlRegImm8(r2, 32)
			ctx.EmitShrRegImm8(r2, 32)
			d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d1)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d2 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r3, fieldAddr)
			d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d2)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
			r4 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r4, thisptr.Reg, off)
			d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			ctx.BindReg(r4, &d2)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d2)
		var d3 scm.JITValueDesc
		if d2.Loc == scm.LocImm {
			d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
		} else {
			r5 := ctx.AllocReg()
			ctx.EmitMovRegReg(r5, d2.Reg)
			ctx.EmitShlRegImm8(r5, 56)
			ctx.EmitShrRegImm8(r5, 56)
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d3)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d3)
		ctx.EnsureDescsTogether(&d1, &d3)
		var d4 scm.JITValueDesc
		if d1.Loc == scm.LocImm && d3.Loc == scm.LocImm {
			d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() * d3.Imm.Int())}
		} else if d1.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
			ctx.EmitImulInt64(scratch, d3.Reg)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else if d3.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d3.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d3.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else {
			r6 := ctx.AllocRegExcept(d1.Reg, d3.Reg)
			ctx.EmitMovRegReg(r6, d1.Reg)
			ctx.EmitImulInt64(r6, d3.Reg)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d4)
		}
		if d4.Loc == scm.LocReg && d1.Loc == scm.LocReg && d4.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1)
		ctx.FreeDesc(&d3)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		var d5 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
		} else {
			r7 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r7, d4.Reg)
			ctx.EmitShrRegImm8(r7, 6)
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d5)
		}
		if d5.Loc == scm.LocReg && d4.Loc == scm.LocReg && d5.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		var d6 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
		} else {
			r8 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r8, d4.Reg)
			ctx.EmitAndRegImm32(r8, 63)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d6)
		}
		if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d4)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d7 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
			r9 := ctx.AllocReg()
			r10 := ctx.AllocRegExcept(r9)
			r11 := ctx.AllocRegExcept(r9, r10)
			ctx.EmitMovRegMem64(r9, fieldAddr)
			ctx.EmitMovRegMem64(r10, fieldAddr+8)
			ctx.EmitMovRegMem64(r11, fieldAddr+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11}
			ctx.BindReg(r9, &d7)
			ctx.BindReg(r10, &d7)
			ctx.BindReg(r11, &d7)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
			r12 := ctx.AllocReg()
			r13 := ctx.AllocRegExcept(r12)
			r14 := ctx.AllocRegExcept(r12, r13)
			ctx.EmitMovRegMem(r12, thisptr.Reg, off)
			ctx.EmitMovRegMem(r13, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
			ctx.BindReg(r12, &d7)
			ctx.BindReg(r13, &d7)
			ctx.BindReg(r14, &d7)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		d9 = ctx.EmitSliceElementAddress(&d7, &d5, 8)
		ctx.EnsureDesc(&d9)
		ctx.EmitMovRegMem(d9.Reg, d9.Reg, 0)
		d8 = d9
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d6)
		var d10 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d6.Imm.Int())))}
		} else if d6.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r15, d8.Reg)
			ctx.EmitShlRegImm8(r15, uint8(d6.Imm.Int()))
			d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d10)
		} else {
			{
				shiftSrc := d8.Reg
				r16 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r16, d8.Reg)
				shiftSrc = r16
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d6.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d6.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d6.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d10)
			}
		}
		if d10.Loc == scm.LocReg && d8.Loc == scm.LocReg && d10.Reg == d8.Reg {
			ctx.TransferReg(d8.Reg)
			d8.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d8)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d11 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d11)
		}
		if d11.Loc == scm.LocReg && d5.Loc == scm.LocReg && d11.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d11)
		ctx.ReclaimUntrackedRegs()
		d13 = ctx.EmitSliceElementAddress(&d7, &d11, 8)
		ctx.EnsureDesc(&d13)
		ctx.EmitMovRegMem(d13.Reg, d13.Reg, 0)
		d12 = d13
		ctx.FreeDesc(&d11)
		ctx.ReclaimUntrackedRegs()
		d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d14, &d6)
		var d15 scm.JITValueDesc
		if d14.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d6.Imm.Int())}
		} else if d6.Loc == scm.LocImm && d6.Imm.Int() == 0 {
			r17 := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegReg(r17, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d15)
		} else if d14.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
			ctx.EmitSubInt64(scratch, d6.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else if d6.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegReg(scratch, d14.Reg)
			if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d6.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else {
			r18 := ctx.AllocRegExcept(d14.Reg, d6.Reg)
			ctx.EmitMovRegReg(r18, d14.Reg)
			ctx.EmitSubInt64(r18, d6.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d15)
		}
		if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
			ctx.TransferReg(d14.Reg)
			d14.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d6)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d15)
		var d16 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d15.Loc == scm.LocImm {
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d12.Imm.Int()) >> uint64(d15.Imm.Int())))}
		} else if d15.Loc == scm.LocImm {
			r19 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r19, d12.Reg)
			ctx.EmitShrRegImm8(r19, uint8(d15.Imm.Int()))
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d16)
		} else {
			{
				shiftSrc := d12.Reg
				r20 := ctx.AllocRegExcept(d12.Reg)
				ctx.EmitMovRegReg(r20, d12.Reg)
				shiftSrc = r20
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d15.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d15.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d15.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d16)
			}
		}
		if d16.Loc == scm.LocReg && d12.Loc == scm.LocReg && d16.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d12)
		ctx.FreeDesc(&d15)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d16)
		var d17 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d16.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() | d16.Imm.Int())}
		} else if d10.Loc == scm.LocImm && d10.Imm.Int() == 0 {
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			ctx.BindReg(d16.Reg, &d17)
		} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
			r21 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(r21, d10.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d17)
		} else if d10.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
			ctx.EmitOrInt64(scratch, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else if d16.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(r22, d10.Reg)
			if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r22, int32(d16.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d16.Imm.Int()))
				ctx.EmitOrInt64(r22, scm.RegR11)
			}
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d17)
		} else {
			r23 := ctx.AllocRegExcept(d10.Reg, d16.Reg)
			ctx.EmitMovRegReg(r23, d10.Reg)
			ctx.EmitOrInt64(r23, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d17)
		}
		if d17.Loc == scm.LocReg && d10.Loc == scm.LocReg && d17.Reg == d10.Reg {
			ctx.TransferReg(d10.Reg)
			d10.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d10)
		ctx.FreeDesc(&d16)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d2)
		var d18 scm.JITValueDesc
		if d2.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
		} else {
			r24 := ctx.AllocReg()
			ctx.EmitMovRegReg(r24, d2.Reg)
			ctx.EmitShlRegImm8(r24, 56)
			ctx.EmitShrRegImm8(r24, 56)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d18)
		}
		ctx.ReclaimUntrackedRegs()
		d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d18)
		ctx.EnsureDescsTogether(&d19, &d18)
		var d20 scm.JITValueDesc
		if d19.Loc == scm.LocImm && d18.Loc == scm.LocImm {
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() - d18.Imm.Int())}
		} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
			r25 := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegReg(r25, d19.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d20)
		} else if d19.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
			ctx.EmitSubInt64(scratch, d18.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d20)
		} else if d18.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegReg(scratch, d19.Reg)
			if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d18.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d20)
		} else {
			r26 := ctx.AllocRegExcept(d19.Reg, d18.Reg)
			ctx.EmitMovRegReg(r26, d19.Reg)
			ctx.EmitSubInt64(r26, d18.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d20)
		}
		if d20.Loc == scm.LocReg && d19.Loc == scm.LocReg && d20.Reg == d19.Reg {
			ctx.TransferReg(d19.Reg)
			d19.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d18)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d20)
		var d21 scm.JITValueDesc
		if d17.Loc == scm.LocImm && d20.Loc == scm.LocImm {
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) >> uint64(d20.Imm.Int())))}
		} else if d20.Loc == scm.LocImm {
			r27 := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(r27, d17.Reg)
			ctx.EmitShrRegImm8(r27, uint8(d20.Imm.Int()))
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d21)
		} else {
			{
				shiftSrc := d17.Reg
				r28 := ctx.AllocRegExcept(d17.Reg)
				ctx.EmitMovRegReg(r28, d17.Reg)
				shiftSrc = r28
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d20.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d20.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d20.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d21)
			}
		}
		if d21.Loc == scm.LocReg && d17.Loc == scm.LocReg && d21.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.FreeDesc(&d20)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.StabilizeDescForControlFlow(&d21)
		ctx.FreeDesc(&idxInt)
		var d22 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
			r29 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r29, fieldAddr)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d22)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
			r30 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r30, thisptr.Reg, off)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			ctx.BindReg(r30, &d22)
		}
		d23 = d22
		ctx.EnsureDesc(&d23)
		if d23.Loc != scm.LocImm && d23.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d23.Loc == scm.LocImm {
			if d23.Imm.Bool() {
				if ps.General {
				}
				ps24 := scm.PhiState{General: ps.General}
				ps24.OverlayValues = make([]scm.JITValueDesc, 24)
				ps24.OverlayValues[0] = d0
				ps24.OverlayValues[1] = d1
				ps24.OverlayValues[2] = d2
				ps24.OverlayValues[3] = d3
				ps24.OverlayValues[4] = d4
				ps24.OverlayValues[5] = d5
				ps24.OverlayValues[6] = d6
				ps24.OverlayValues[7] = d7
				ps24.OverlayValues[8] = d8
				ps24.OverlayValues[9] = d9
				ps24.OverlayValues[10] = d10
				ps24.OverlayValues[11] = d11
				ps24.OverlayValues[12] = d12
				ps24.OverlayValues[13] = d13
				ps24.OverlayValues[14] = d14
				ps24.OverlayValues[15] = d15
				ps24.OverlayValues[16] = d16
				ps24.OverlayValues[17] = d17
				ps24.OverlayValues[18] = d18
				ps24.OverlayValues[19] = d19
				ps24.OverlayValues[20] = d20
				ps24.OverlayValues[21] = d21
				ps24.OverlayValues[22] = d22
				ps24.OverlayValues[23] = d23
				return bbs[3].RenderPS(ps24)
			}
			if ps.General {
			}
			ps25 := scm.PhiState{General: ps.General}
			ps25.OverlayValues = make([]scm.JITValueDesc, 24)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[6] = d6
			ps25.OverlayValues[7] = d7
			ps25.OverlayValues[8] = d8
			ps25.OverlayValues[9] = d9
			ps25.OverlayValues[10] = d10
			ps25.OverlayValues[11] = d11
			ps25.OverlayValues[12] = d12
			ps25.OverlayValues[13] = d13
			ps25.OverlayValues[14] = d14
			ps25.OverlayValues[15] = d15
			ps25.OverlayValues[16] = d16
			ps25.OverlayValues[17] = d17
			ps25.OverlayValues[18] = d18
			ps25.OverlayValues[19] = d19
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps25.OverlayValues[23] = d23
			return bbs[2].RenderPS(ps25)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl6 := ctx.ReserveLabel()
		lbl7 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d23.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl6)
		ctx.EmitJmp(lbl7)
		ctx.MarkLabel(lbl6)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl7)
		ctx.EmitJmp(lbl3)
		ps26 := scm.PhiState{General: true}
		ps26.OverlayValues = make([]scm.JITValueDesc, 24)
		ps26.OverlayValues[0] = d0
		ps26.OverlayValues[1] = d1
		ps26.OverlayValues[2] = d2
		ps26.OverlayValues[3] = d3
		ps26.OverlayValues[4] = d4
		ps26.OverlayValues[5] = d5
		ps26.OverlayValues[6] = d6
		ps26.OverlayValues[7] = d7
		ps26.OverlayValues[8] = d8
		ps26.OverlayValues[9] = d9
		ps26.OverlayValues[10] = d10
		ps26.OverlayValues[11] = d11
		ps26.OverlayValues[12] = d12
		ps26.OverlayValues[13] = d13
		ps26.OverlayValues[14] = d14
		ps26.OverlayValues[15] = d15
		ps26.OverlayValues[16] = d16
		ps26.OverlayValues[17] = d17
		ps26.OverlayValues[18] = d18
		ps26.OverlayValues[19] = d19
		ps26.OverlayValues[20] = d20
		ps26.OverlayValues[21] = d21
		ps26.OverlayValues[22] = d22
		ps26.OverlayValues[23] = d23
		ps27 := scm.PhiState{General: true}
		ps27.OverlayValues = make([]scm.JITValueDesc, 24)
		ps27.OverlayValues[0] = d0
		ps27.OverlayValues[1] = d1
		ps27.OverlayValues[2] = d2
		ps27.OverlayValues[3] = d3
		ps27.OverlayValues[4] = d4
		ps27.OverlayValues[5] = d5
		ps27.OverlayValues[6] = d6
		ps27.OverlayValues[7] = d7
		ps27.OverlayValues[8] = d8
		ps27.OverlayValues[9] = d9
		ps27.OverlayValues[10] = d10
		ps27.OverlayValues[11] = d11
		ps27.OverlayValues[12] = d12
		ps27.OverlayValues[13] = d13
		ps27.OverlayValues[14] = d14
		ps27.OverlayValues[15] = d15
		ps27.OverlayValues[16] = d16
		ps27.OverlayValues[17] = d17
		ps27.OverlayValues[18] = d18
		ps27.OverlayValues[19] = d19
		ps27.OverlayValues[20] = d20
		ps27.OverlayValues[21] = d21
		ps27.OverlayValues[22] = d22
		ps27.OverlayValues[23] = d23
		snap28 := d0
		snap29 := d1
		snap30 := d2
		snap31 := d3
		snap32 := d4
		snap33 := d5
		snap34 := d6
		snap35 := d7
		snap36 := d8
		snap37 := d9
		snap38 := d10
		snap39 := d11
		snap40 := d12
		snap41 := d13
		snap42 := d14
		snap43 := d15
		snap44 := d16
		snap45 := d17
		snap46 := d18
		snap47 := d19
		snap48 := d20
		snap49 := d21
		snap50 := d22
		snap51 := d23
		alloc52 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps27)
		}
		ctx.RestoreAllocState(alloc52)
		d0 = snap28
		d1 = snap29
		d2 = snap30
		d3 = snap31
		d4 = snap32
		d5 = snap33
		d6 = snap34
		d7 = snap35
		d8 = snap36
		d9 = snap37
		d10 = snap38
		d11 = snap39
		d12 = snap40
		d13 = snap41
		d14 = snap42
		d15 = snap43
		d16 = snap44
		d17 = snap45
		d18 = snap46
		d19 = snap47
		d20 = snap48
		d21 = snap49
		d22 = snap50
		d23 = snap51
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps26)
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
		if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
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
		ctx.ReclaimUntrackedRegs()
		d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d54 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d54)
		ctx.BindReg(r1, &d54)
		ctx.EnsureDesc(&d53)
		if d53.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d53, &d54)
		} else {
			switch d53.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d54, d53)
			case scm.TagInt:
				ctx.EmitMakeInt(d54, d53)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d54, d53)
			case scm.TagNil:
				ctx.EmitMakeNil(d54)
			default:
				ctx.EmitMovPairToResult(&d53, &d54)
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
		if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d21)
		var d55 scm.JITValueDesc
		if d21.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d21.Imm.Int()))))}
		} else {
			r31 := ctx.AllocReg()
			ctx.EmitMovRegReg(r31, d21.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d55)
		}
		var d56 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
			r32 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r32, fieldAddr)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d56)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
			r33 := ctx.AllocReg()
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d56)
		}
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d56)
		ctx.EnsureDescsTogether(&d55, &d56)
		var d57 scm.JITValueDesc
		if d55.Loc == scm.LocImm && d56.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d55.Imm.Int() + d56.Imm.Int())}
		} else if d56.Loc == scm.LocImm && d56.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegReg(r34, d55.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d57)
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d56.Reg}
			ctx.BindReg(d56.Reg, &d57)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d55.Imm.Int()))
			ctx.EmitAddInt64(scratch, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else if d56.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegReg(scratch, d55.Reg)
			if d56.Imm.Int() >= -2147483648 && d56.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d56.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else {
			r35 := ctx.AllocRegExcept(d55.Reg, d56.Reg)
			ctx.EmitMovRegReg(r35, d55.Reg)
			ctx.EmitAddInt64(r35, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d57)
		}
		if d57.Loc == scm.LocReg && d55.Loc == scm.LocReg && d57.Reg == d55.Reg {
			ctx.TransferReg(d55.Reg)
			d55.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d55)
		ctx.EnsureDesc(&d57)
		d58 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d58)
		ctx.BindReg(r1, &d58)
		ctx.EnsureDesc(&d57)
		ctx.EmitMakeInt(d58, d57)
		if d57.Loc == scm.LocReg {
			ctx.FreeReg(d57.Reg)
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
		if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
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
		ctx.ReclaimUntrackedRegs()
		var d59 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
			r36 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r36, fieldAddr)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r36}
			ctx.BindReg(r36, &d59)
		} else {
			off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMem(r37, thisptr.Reg, off)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d59)
		}
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&d21, &d59)
		var d60 scm.JITValueDesc
		if d21.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d21.Imm.Int()) == uint64(d59.Imm.Int()))}
		} else if d59.Loc == scm.LocImm {
			r38 := ctx.AllocRegExcept(d21.Reg)
			if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d21.Reg, int32(d59.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
				ctx.EmitCmpInt64(d21.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r38, scm.CondEqual)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
			ctx.BindReg(r38, &d60)
		} else if d21.Loc == scm.LocImm {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d21.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d59.Reg)
			ctx.EmitSetcc(r39, scm.CondEqual)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r39}
			ctx.BindReg(r39, &d60)
		} else {
			r40 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitCmpInt64(d21.Reg, d59.Reg)
			ctx.EmitSetcc(r40, scm.CondEqual)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r40}
			ctx.BindReg(r40, &d60)
		}
		ctx.FreeDesc(&d21)
		d61 = d60
		ctx.EnsureDesc(&d61)
		if d61.Loc != scm.LocImm && d61.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d61.Loc == scm.LocImm {
			if d61.Imm.Bool() {
				if ps.General {
				}
				ps62 := scm.PhiState{General: ps.General}
				ps62.OverlayValues = make([]scm.JITValueDesc, 62)
				ps62.OverlayValues[0] = d0
				ps62.OverlayValues[1] = d1
				ps62.OverlayValues[2] = d2
				ps62.OverlayValues[3] = d3
				ps62.OverlayValues[4] = d4
				ps62.OverlayValues[5] = d5
				ps62.OverlayValues[6] = d6
				ps62.OverlayValues[7] = d7
				ps62.OverlayValues[8] = d8
				ps62.OverlayValues[9] = d9
				ps62.OverlayValues[10] = d10
				ps62.OverlayValues[11] = d11
				ps62.OverlayValues[12] = d12
				ps62.OverlayValues[13] = d13
				ps62.OverlayValues[14] = d14
				ps62.OverlayValues[15] = d15
				ps62.OverlayValues[16] = d16
				ps62.OverlayValues[17] = d17
				ps62.OverlayValues[18] = d18
				ps62.OverlayValues[19] = d19
				ps62.OverlayValues[20] = d20
				ps62.OverlayValues[21] = d21
				ps62.OverlayValues[22] = d22
				ps62.OverlayValues[23] = d23
				ps62.OverlayValues[53] = d53
				ps62.OverlayValues[54] = d54
				ps62.OverlayValues[55] = d55
				ps62.OverlayValues[56] = d56
				ps62.OverlayValues[57] = d57
				ps62.OverlayValues[58] = d58
				ps62.OverlayValues[59] = d59
				ps62.OverlayValues[60] = d60
				ps62.OverlayValues[61] = d61
				return bbs[1].RenderPS(ps62)
			}
			if ps.General {
			}
			ps63 := scm.PhiState{General: ps.General}
			ps63.OverlayValues = make([]scm.JITValueDesc, 62)
			ps63.OverlayValues[0] = d0
			ps63.OverlayValues[1] = d1
			ps63.OverlayValues[2] = d2
			ps63.OverlayValues[3] = d3
			ps63.OverlayValues[4] = d4
			ps63.OverlayValues[5] = d5
			ps63.OverlayValues[6] = d6
			ps63.OverlayValues[7] = d7
			ps63.OverlayValues[8] = d8
			ps63.OverlayValues[9] = d9
			ps63.OverlayValues[10] = d10
			ps63.OverlayValues[11] = d11
			ps63.OverlayValues[12] = d12
			ps63.OverlayValues[13] = d13
			ps63.OverlayValues[14] = d14
			ps63.OverlayValues[15] = d15
			ps63.OverlayValues[16] = d16
			ps63.OverlayValues[17] = d17
			ps63.OverlayValues[18] = d18
			ps63.OverlayValues[19] = d19
			ps63.OverlayValues[20] = d20
			ps63.OverlayValues[21] = d21
			ps63.OverlayValues[22] = d22
			ps63.OverlayValues[23] = d23
			ps63.OverlayValues[53] = d53
			ps63.OverlayValues[54] = d54
			ps63.OverlayValues[55] = d55
			ps63.OverlayValues[56] = d56
			ps63.OverlayValues[57] = d57
			ps63.OverlayValues[58] = d58
			ps63.OverlayValues[59] = d59
			ps63.OverlayValues[60] = d60
			ps63.OverlayValues[61] = d61
			return bbs[2].RenderPS(ps63)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl8 := ctx.ReserveLabel()
		lbl9 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d61.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl8)
		ctx.EmitJmp(lbl9)
		ctx.MarkLabel(lbl8)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl9)
		ctx.EmitJmp(lbl3)
		ps64 := scm.PhiState{General: true}
		ps64.OverlayValues = make([]scm.JITValueDesc, 62)
		ps64.OverlayValues[0] = d0
		ps64.OverlayValues[1] = d1
		ps64.OverlayValues[2] = d2
		ps64.OverlayValues[3] = d3
		ps64.OverlayValues[4] = d4
		ps64.OverlayValues[5] = d5
		ps64.OverlayValues[6] = d6
		ps64.OverlayValues[7] = d7
		ps64.OverlayValues[8] = d8
		ps64.OverlayValues[9] = d9
		ps64.OverlayValues[10] = d10
		ps64.OverlayValues[11] = d11
		ps64.OverlayValues[12] = d12
		ps64.OverlayValues[13] = d13
		ps64.OverlayValues[14] = d14
		ps64.OverlayValues[15] = d15
		ps64.OverlayValues[16] = d16
		ps64.OverlayValues[17] = d17
		ps64.OverlayValues[18] = d18
		ps64.OverlayValues[19] = d19
		ps64.OverlayValues[20] = d20
		ps64.OverlayValues[21] = d21
		ps64.OverlayValues[22] = d22
		ps64.OverlayValues[23] = d23
		ps64.OverlayValues[53] = d53
		ps64.OverlayValues[54] = d54
		ps64.OverlayValues[55] = d55
		ps64.OverlayValues[56] = d56
		ps64.OverlayValues[57] = d57
		ps64.OverlayValues[58] = d58
		ps64.OverlayValues[59] = d59
		ps64.OverlayValues[60] = d60
		ps64.OverlayValues[61] = d61
		ps65 := scm.PhiState{General: true}
		ps65.OverlayValues = make([]scm.JITValueDesc, 62)
		ps65.OverlayValues[0] = d0
		ps65.OverlayValues[1] = d1
		ps65.OverlayValues[2] = d2
		ps65.OverlayValues[3] = d3
		ps65.OverlayValues[4] = d4
		ps65.OverlayValues[5] = d5
		ps65.OverlayValues[6] = d6
		ps65.OverlayValues[7] = d7
		ps65.OverlayValues[8] = d8
		ps65.OverlayValues[9] = d9
		ps65.OverlayValues[10] = d10
		ps65.OverlayValues[11] = d11
		ps65.OverlayValues[12] = d12
		ps65.OverlayValues[13] = d13
		ps65.OverlayValues[14] = d14
		ps65.OverlayValues[15] = d15
		ps65.OverlayValues[16] = d16
		ps65.OverlayValues[17] = d17
		ps65.OverlayValues[18] = d18
		ps65.OverlayValues[19] = d19
		ps65.OverlayValues[20] = d20
		ps65.OverlayValues[21] = d21
		ps65.OverlayValues[22] = d22
		ps65.OverlayValues[23] = d23
		ps65.OverlayValues[53] = d53
		ps65.OverlayValues[54] = d54
		ps65.OverlayValues[55] = d55
		ps65.OverlayValues[56] = d56
		ps65.OverlayValues[57] = d57
		ps65.OverlayValues[58] = d58
		ps65.OverlayValues[59] = d59
		ps65.OverlayValues[60] = d60
		ps65.OverlayValues[61] = d61
		snap66 := d0
		snap67 := d1
		snap68 := d2
		snap69 := d3
		snap70 := d4
		snap71 := d5
		snap72 := d6
		snap73 := d7
		snap74 := d8
		snap75 := d9
		snap76 := d10
		snap77 := d11
		snap78 := d12
		snap79 := d13
		snap80 := d14
		snap81 := d15
		snap82 := d16
		snap83 := d17
		snap84 := d18
		snap85 := d19
		snap86 := d20
		snap87 := d21
		snap88 := d22
		snap89 := d23
		snap90 := d53
		snap91 := d54
		snap92 := d55
		snap93 := d56
		snap94 := d57
		snap95 := d58
		snap96 := d59
		snap97 := d60
		snap98 := d61
		alloc99 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps65)
		}
		ctx.RestoreAllocState(alloc99)
		d0 = snap66
		d1 = snap67
		d2 = snap68
		d3 = snap69
		d4 = snap70
		d5 = snap71
		d6 = snap72
		d7 = snap73
		d8 = snap74
		d9 = snap75
		d10 = snap76
		d11 = snap77
		d12 = snap78
		d13 = snap79
		d14 = snap80
		d15 = snap81
		d16 = snap82
		d17 = snap83
		d18 = snap84
		d19 = snap85
		d20 = snap86
		d21 = snap87
		d22 = snap88
		d23 = snap89
		d53 = snap90
		d54 = snap91
		d55 = snap92
		d56 = snap93
		d57 = snap94
		d58 = snap95
		d59 = snap96
		d60 = snap97
		d61 = snap98
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps64)
		}
		return result
		ctx.FreeDesc(&d60)
		return result
	}
	ps100 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps100)
	ctx.MarkLabel(lbl0)
	d101 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d101)
	ctx.BindReg(r1, &d101)
	ctx.EmitMovPairToResult(&d101, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
	if thisptrPinned {
		ctx.UnprotectReg(thisptrPinnedReg)
	}
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
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
	chunk := bitpos / 64
	offset := bitpos % 64
	// StorageInt keeps one trailing sentinel chunk. Variable shifts by 64
	// produce zero, so reading both words removes the only decode branch while
	// preserving the aligned case where offset is zero.
	v := s.chunk[chunk]<<offset | s.chunk[chunk+1]>>(64-offset)
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
