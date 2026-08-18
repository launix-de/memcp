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
			var d74 scm.JITValueDesc
			_ = d74
			var d75 scm.JITValueDesc
			_ = d75
			var d76 scm.JITValueDesc
			_ = d76
			var d77 scm.JITValueDesc
			_ = d77
			var d78 scm.JITValueDesc
			_ = d78
			var d79 scm.JITValueDesc
			_ = d79
			var d80 scm.JITValueDesc
			_ = d80
			var d81 scm.JITValueDesc
			_ = d81
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
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
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
			ctx.EnsureDesc(&idxInt)
			d0 = idxInt
			_ = d0
			r2 := thisptr.Loc == scm.LocReg
			r3 := thisptr.Reg
			if r2 { ctx.ProtectReg(r3) }
			r4 := idxInt.Loc == scm.LocReg
			r5 := idxInt.Reg
			if r4 { ctx.ProtectReg(r5) }
			phiBase1 := ctx.AllocStack(int32(16))
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase1)+int32(0)}
			lbl5 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d3 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
			} else {
				r6 := ctx.AllocReg()
				ctx.EmitMovRegReg(r6, d0.Reg)
				ctx.EmitShlRegImm8(r6, 32)
				ctx.EmitShrRegImm8(r6, 32)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d3)
			}
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				r7 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r7, fieldAddr)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
				ctx.BindReg(r7, &d4)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r8 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r8, thisptr.Reg, off)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
				ctx.BindReg(r8, &d4)
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d5 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r9 := ctx.AllocReg()
				ctx.EmitMovRegReg(r9, d4.Reg)
				ctx.EmitShlRegImm8(r9, 56)
				ctx.EmitShrRegImm8(r9, 56)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d5)
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d3)
			ctx.ProtectReg(d3.Reg)
			ctx.EnsureDesc(&d5)
			ctx.UnprotectReg(d3.Reg)
			var d6 scm.JITValueDesc
			if d3.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() * d5.Imm.Int())}
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.EmitImulInt64(scratch, d5.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else if d5.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.EmitMovRegReg(scratch, d3.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d5.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else {
				r10 := ctx.AllocRegExcept(d3.Reg, d5.Reg)
				ctx.EmitMovRegReg(r10, d3.Reg)
				ctx.EmitImulInt64(r10, d5.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d6)
			}
			if d6.Loc == scm.LocReg && d3.Loc == scm.LocReg && d6.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			ctx.FreeDesc(&d5)
			var d7 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
				r11 := ctx.AllocReg()
				r12 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r11, fieldAddr)
				ctx.EmitMovRegMem64(r12, fieldAddr+8)
				d7 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r11, Reg2: r12}
				ctx.BindReg(r11, &d7)
				ctx.BindReg(r12, &d7)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
				r13 := ctx.AllocReg()
				r14 := ctx.AllocReg()
				ctx.EmitMovRegMem(r13, thisptr.Reg, off)
				ctx.EmitMovRegMem(r14, thisptr.Reg, off+8)
				d7 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r13, Reg2: r14}
				ctx.BindReg(r13, &d7)
				ctx.BindReg(r14, &d7)
			}
			ctx.EnsureDesc(&d6)
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r15 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r15, d6.Reg)
				ctx.EmitShrRegImm8(r15, 6)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d8)
			}
			if d8.Loc == scm.LocReg && d6.Loc == scm.LocReg && d8.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d8)
			r16 := ctx.AllocReg()
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d7)
			if d8.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r16, uint64(d8.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r16, d8.Reg)
				ctx.EmitShlRegImm8(r16, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitAddInt64(r16, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r16, d7.Reg)
			}
			r17 := ctx.AllocRegExcept(r16)
			ctx.EmitMovRegMem(r17, r16, 0)
			ctx.FreeReg(r16)
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
			ctx.BindReg(r17, &d9)
			ctx.FreeDesc(&d8)
			ctx.EnsureDesc(&d6)
			var d10 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r18 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r18, d6.Reg)
				ctx.EmitAndRegImm32(r18, 63)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d10)
			}
			if d10.Loc == scm.LocReg && d6.Loc == scm.LocReg && d10.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d10)
			var d11 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d9.Imm.Int()) << uint64(d10.Imm.Int())))}
			} else if d10.Loc == scm.LocImm {
				r19 := ctx.AllocRegExcept(d9.Reg)
				ctx.EmitMovRegReg(r19, d9.Reg)
				ctx.EmitShlRegImm8(r19, uint8(d10.Imm.Int()))
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d11)
			} else {
				{
					shiftSrc := d9.Reg
					r20 := ctx.AllocRegExcept(d9.Reg)
					ctx.EmitMovRegReg(r20, d9.Reg)
					shiftSrc = r20
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d10.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d10.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d10.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d11)
				}
			}
			if d11.Loc == scm.LocReg && d9.Loc == scm.LocReg && d11.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d9)
			ctx.FreeDesc(&d10)
			ctx.EnsureDesc(&d6)
			var d12 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r21 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r21, d6.Reg)
				ctx.EmitAndRegImm32(r21, 63)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d12)
			}
			if d12.Loc == scm.LocReg && d6.Loc == scm.LocReg && d12.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d13 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r22 := ctx.AllocReg()
				ctx.EmitMovRegReg(r22, d4.Reg)
				ctx.EmitShlRegImm8(r22, 56)
				ctx.EmitShrRegImm8(r22, 56)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d13)
			}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d12)
			ctx.ProtectReg(d12.Reg)
			ctx.EnsureDesc(&d13)
			ctx.UnprotectReg(d12.Reg)
			var d14 scm.JITValueDesc
			if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() + d13.Imm.Int())}
			} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
				r23 := ctx.AllocRegExcept(d12.Reg)
				ctx.EmitMovRegReg(r23, d12.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d14)
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
				ctx.BindReg(d13.Reg, &d14)
			} else if d12.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.EmitAddInt64(scratch, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.EmitMovRegReg(scratch, d12.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d13.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else {
				r24 := ctx.AllocRegExcept(d12.Reg, d13.Reg)
				ctx.EmitMovRegReg(r24, d12.Reg)
				ctx.EmitAddInt64(r24, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d14)
			}
			if d14.Loc == scm.LocReg && d12.Loc == scm.LocReg && d14.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d14.Imm.Int()) > uint64(64))}
			} else {
				r25 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitCmpRegImm32(d14.Reg, 64)
				ctx.EmitSetcc(r25, scm.CcA)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r25}
				ctx.BindReg(r25, &d15)
			}
			ctx.FreeDesc(&d14)
			d16 = d15
			ctx.EnsureDesc(&d16)
			if d16.Loc != scm.LocImm && d16.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl6 := ctx.ReserveLabel()
			lbl7 := ctx.ReserveLabel()
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			if d16.Loc == scm.LocImm {
				if d16.Imm.Bool() {
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl6)
				} else {
					ctx.MarkLabel(lbl9)
			ctx.EnsureDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d17 = d11
			if d17.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			ctx.EmitStoreToStack(d17, int32(bbs[2].PhiBase)+int32(0))
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
					ctx.EmitJmp(lbl7)
				}
			} else {
				ctx.EmitCmpRegImm32(d16.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl8)
				ctx.EmitJmp(lbl9)
				ctx.MarkLabel(lbl8)
				ctx.EmitJmp(lbl6)
				ctx.MarkLabel(lbl9)
			ctx.EnsureDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d18 = d11
			if d18.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d18)
			ctx.EmitStoreToStack(d18, int32(bbs[2].PhiBase)+int32(0))
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
				ctx.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d15)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl7)
			ctx.ResolveFixups()
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d19 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.EmitMovRegReg(r26, d4.Reg)
				ctx.EmitShlRegImm8(r26, 56)
				ctx.EmitShrRegImm8(r26, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d19)
			}
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.ProtectReg(d20.Reg)
			ctx.EnsureDesc(&d19)
			ctx.UnprotectReg(d20.Reg)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(r27, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d21)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.EmitSubInt64(scratch, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(scratch, d20.Reg)
				if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d19.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r28 := ctx.AllocRegExcept(d20.Reg, d19.Reg)
				ctx.EmitMovRegReg(r28, d20.Reg)
				ctx.EmitSubInt64(r28, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d21)
			}
			if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d2.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d2.Imm.Int()) >> uint64(d21.Imm.Int())))}
			} else if d21.Loc == scm.LocImm {
				r29 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(r29, d2.Reg)
				ctx.EmitShrRegImm8(r29, uint8(d21.Imm.Int()))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d22)
			} else {
				{
					shiftSrc := d2.Reg
					r30 := ctx.AllocRegExcept(d2.Reg)
					ctx.EmitMovRegReg(r30, d2.Reg)
					shiftSrc = r30
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d21.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d21.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d21.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d22)
				}
			}
			if d22.Loc == scm.LocReg && d2.Loc == scm.LocReg && d22.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.FreeDesc(&d21)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			if d22.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r31, d22)
			}
			ctx.EmitJmp(lbl5)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl6)
			ctx.ResolveFixups()
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			var d23 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r32 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r32, d6.Reg)
				ctx.EmitShrRegImm8(r32, 6)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d23)
			}
			if d23.Loc == scm.LocReg && d6.Loc == scm.LocReg && d23.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.EmitMovRegReg(scratch, d23.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			}
			if d24.Loc == scm.LocReg && d23.Loc == scm.LocReg && d24.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.EnsureDesc(&d24)
			r33 := ctx.AllocReg()
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d7)
			if d24.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r33, uint64(d24.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r33, d24.Reg)
				ctx.EmitShlRegImm8(r33, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitAddInt64(r33, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r33, d7.Reg)
			}
			r34 := ctx.AllocRegExcept(r33)
			ctx.EmitMovRegMem(r34, r33, 0)
			ctx.FreeReg(r33)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d25)
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d6)
			var d26 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r35 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r35, d6.Reg)
				ctx.EmitAndRegImm32(r35, 63)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d26)
			}
			if d26.Loc == scm.LocReg && d6.Loc == scm.LocReg && d26.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d27)
			ctx.ProtectReg(d27.Reg)
			ctx.EnsureDesc(&d26)
			ctx.UnprotectReg(d27.Reg)
			var d28 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d27.Imm.Int() - d26.Imm.Int())}
			} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
				r36 := ctx.AllocRegExcept(d27.Reg)
				ctx.EmitMovRegReg(r36, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d28)
			} else if d27.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d27.Imm.Int()))
				ctx.EmitSubInt64(scratch, d26.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.EmitMovRegReg(scratch, d27.Reg)
				if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d26.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else {
				r37 := ctx.AllocRegExcept(d27.Reg, d26.Reg)
				ctx.EmitMovRegReg(r37, d27.Reg)
				ctx.EmitSubInt64(r37, d26.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d28)
			}
			if d28.Loc == scm.LocReg && d27.Loc == scm.LocReg && d28.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d25.Imm.Int()) >> uint64(d28.Imm.Int())))}
			} else if d28.Loc == scm.LocImm {
				r38 := ctx.AllocRegExcept(d25.Reg)
				ctx.EmitMovRegReg(r38, d25.Reg)
				ctx.EmitShrRegImm8(r38, uint8(d28.Imm.Int()))
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d29)
			} else {
				{
					shiftSrc := d25.Reg
					r39 := ctx.AllocRegExcept(d25.Reg)
					ctx.EmitMovRegReg(r39, d25.Reg)
					shiftSrc = r39
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d28.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d28.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d28.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d29)
				}
			}
			if d29.Loc == scm.LocReg && d25.Loc == scm.LocReg && d29.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d29)
			var d30 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() | d29.Imm.Int())}
			} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d29.Reg}
				ctx.BindReg(d29.Reg, &d30)
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				r40 := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(r40, d11.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d30)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.EmitOrInt64(scratch, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else if d29.Loc == scm.LocImm {
				r41 := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(r41, d11.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r41, int32(d29.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
					ctx.EmitOrInt64(r41, scm.RegR11)
				}
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d30)
			} else {
				r42 := ctx.AllocRegExcept(d11.Reg, d29.Reg)
				ctx.EmitMovRegReg(r42, d11.Reg)
				ctx.EmitOrInt64(r42, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d30)
			}
			if d30.Loc == scm.LocReg && d11.Loc == scm.LocReg && d30.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d30)
			if d30.Loc == scm.LocReg {
				ctx.ProtectReg(d30.Reg)
			} else if d30.Loc == scm.LocRegPair {
				ctx.ProtectReg(d30.Reg)
				ctx.ProtectReg(d30.Reg2)
			}
			d31 = d30
			if d31.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d31)
			ctx.EmitStoreToStack(d31, int32(bbs[2].PhiBase)+int32(0))
			if d30.Loc == scm.LocReg {
				ctx.UnprotectReg(d30.Reg)
			} else if d30.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d30.Reg)
				ctx.UnprotectReg(d30.Reg2)
			}
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl5)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d32)
			ctx.BindReg(r31, &d32)
			if r2 { ctx.UnprotectReg(r3) }
			if r4 { ctx.UnprotectReg(r5) }
			ctx.FreeDesc(&idxInt)
			var d33 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
				r43 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r43, fieldAddr)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d33)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
				r44 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r44, thisptr.Reg, off)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
				ctx.BindReg(r44, &d33)
			}
			d34 = d33
			ctx.EnsureDesc(&d34)
			if d34.Loc != scm.LocImm && d34.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d34.Loc == scm.LocImm {
				if d34.Imm.Bool() {
			ps35 := scm.PhiState{General: ps.General}
			ps35.OverlayValues = make([]scm.JITValueDesc, 35)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[3] = d3
			ps35.OverlayValues[4] = d4
			ps35.OverlayValues[5] = d5
			ps35.OverlayValues[6] = d6
			ps35.OverlayValues[7] = d7
			ps35.OverlayValues[8] = d8
			ps35.OverlayValues[9] = d9
			ps35.OverlayValues[10] = d10
			ps35.OverlayValues[11] = d11
			ps35.OverlayValues[12] = d12
			ps35.OverlayValues[13] = d13
			ps35.OverlayValues[14] = d14
			ps35.OverlayValues[15] = d15
			ps35.OverlayValues[16] = d16
			ps35.OverlayValues[17] = d17
			ps35.OverlayValues[18] = d18
			ps35.OverlayValues[19] = d19
			ps35.OverlayValues[20] = d20
			ps35.OverlayValues[21] = d21
			ps35.OverlayValues[22] = d22
			ps35.OverlayValues[23] = d23
			ps35.OverlayValues[24] = d24
			ps35.OverlayValues[25] = d25
			ps35.OverlayValues[26] = d26
			ps35.OverlayValues[27] = d27
			ps35.OverlayValues[28] = d28
			ps35.OverlayValues[29] = d29
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[32] = d32
			ps35.OverlayValues[33] = d33
			ps35.OverlayValues[34] = d34
					return bbs[3].RenderPS(ps35)
				}
			ps36 := scm.PhiState{General: ps.General}
			ps36.OverlayValues = make([]scm.JITValueDesc, 35)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[3] = d3
			ps36.OverlayValues[4] = d4
			ps36.OverlayValues[5] = d5
			ps36.OverlayValues[6] = d6
			ps36.OverlayValues[7] = d7
			ps36.OverlayValues[8] = d8
			ps36.OverlayValues[9] = d9
			ps36.OverlayValues[10] = d10
			ps36.OverlayValues[11] = d11
			ps36.OverlayValues[12] = d12
			ps36.OverlayValues[13] = d13
			ps36.OverlayValues[14] = d14
			ps36.OverlayValues[15] = d15
			ps36.OverlayValues[16] = d16
			ps36.OverlayValues[17] = d17
			ps36.OverlayValues[18] = d18
			ps36.OverlayValues[19] = d19
			ps36.OverlayValues[20] = d20
			ps36.OverlayValues[21] = d21
			ps36.OverlayValues[22] = d22
			ps36.OverlayValues[23] = d23
			ps36.OverlayValues[24] = d24
			ps36.OverlayValues[25] = d25
			ps36.OverlayValues[26] = d26
			ps36.OverlayValues[27] = d27
			ps36.OverlayValues[28] = d28
			ps36.OverlayValues[29] = d29
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[32] = d32
			ps36.OverlayValues[33] = d33
			ps36.OverlayValues[34] = d34
				return bbs[2].RenderPS(ps36)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl10 := ctx.ReserveLabel()
			lbl11 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d34.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl10)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl11)
			ctx.EmitJmp(lbl3)
			ps37 := scm.PhiState{General: true}
			ps37.OverlayValues = make([]scm.JITValueDesc, 35)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[2] = d2
			ps37.OverlayValues[3] = d3
			ps37.OverlayValues[4] = d4
			ps37.OverlayValues[5] = d5
			ps37.OverlayValues[6] = d6
			ps37.OverlayValues[7] = d7
			ps37.OverlayValues[8] = d8
			ps37.OverlayValues[9] = d9
			ps37.OverlayValues[10] = d10
			ps37.OverlayValues[11] = d11
			ps37.OverlayValues[12] = d12
			ps37.OverlayValues[13] = d13
			ps37.OverlayValues[14] = d14
			ps37.OverlayValues[15] = d15
			ps37.OverlayValues[16] = d16
			ps37.OverlayValues[17] = d17
			ps37.OverlayValues[18] = d18
			ps37.OverlayValues[19] = d19
			ps37.OverlayValues[20] = d20
			ps37.OverlayValues[21] = d21
			ps37.OverlayValues[22] = d22
			ps37.OverlayValues[23] = d23
			ps37.OverlayValues[24] = d24
			ps37.OverlayValues[25] = d25
			ps37.OverlayValues[26] = d26
			ps37.OverlayValues[27] = d27
			ps37.OverlayValues[28] = d28
			ps37.OverlayValues[29] = d29
			ps37.OverlayValues[30] = d30
			ps37.OverlayValues[31] = d31
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			ps37.OverlayValues[34] = d34
			ps38 := scm.PhiState{General: true}
			ps38.OverlayValues = make([]scm.JITValueDesc, 35)
			ps38.OverlayValues[0] = d0
			ps38.OverlayValues[2] = d2
			ps38.OverlayValues[3] = d3
			ps38.OverlayValues[4] = d4
			ps38.OverlayValues[5] = d5
			ps38.OverlayValues[6] = d6
			ps38.OverlayValues[7] = d7
			ps38.OverlayValues[8] = d8
			ps38.OverlayValues[9] = d9
			ps38.OverlayValues[10] = d10
			ps38.OverlayValues[11] = d11
			ps38.OverlayValues[12] = d12
			ps38.OverlayValues[13] = d13
			ps38.OverlayValues[14] = d14
			ps38.OverlayValues[15] = d15
			ps38.OverlayValues[16] = d16
			ps38.OverlayValues[17] = d17
			ps38.OverlayValues[18] = d18
			ps38.OverlayValues[19] = d19
			ps38.OverlayValues[20] = d20
			ps38.OverlayValues[21] = d21
			ps38.OverlayValues[22] = d22
			ps38.OverlayValues[23] = d23
			ps38.OverlayValues[24] = d24
			ps38.OverlayValues[25] = d25
			ps38.OverlayValues[26] = d26
			ps38.OverlayValues[27] = d27
			ps38.OverlayValues[28] = d28
			ps38.OverlayValues[29] = d29
			ps38.OverlayValues[30] = d30
			ps38.OverlayValues[31] = d31
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[33] = d33
			ps38.OverlayValues[34] = d34
			snap39 := d0
			snap40 := d2
			snap41 := d3
			snap42 := d4
			snap43 := d5
			snap44 := d6
			snap45 := d7
			snap46 := d8
			snap47 := d9
			snap48 := d10
			snap49 := d11
			snap50 := d12
			snap51 := d13
			snap52 := d14
			snap53 := d15
			snap54 := d16
			snap55 := d17
			snap56 := d18
			snap57 := d19
			snap58 := d20
			snap59 := d21
			snap60 := d22
			snap61 := d23
			snap62 := d24
			snap63 := d25
			snap64 := d26
			snap65 := d27
			snap66 := d28
			snap67 := d29
			snap68 := d30
			snap69 := d31
			snap70 := d32
			snap71 := d33
			snap72 := d34
			alloc73 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps38)
			}
			ctx.RestoreAllocState(alloc73)
			d0 = snap39
			d2 = snap40
			d3 = snap41
			d4 = snap42
			d5 = snap43
			d6 = snap44
			d7 = snap45
			d8 = snap46
			d9 = snap47
			d10 = snap48
			d11 = snap49
			d12 = snap50
			d13 = snap51
			d14 = snap52
			d15 = snap53
			d16 = snap54
			d17 = snap55
			d18 = snap56
			d19 = snap57
			d20 = snap58
			d21 = snap59
			d22 = snap60
			d23 = snap61
			d24 = snap62
			d25 = snap63
			d26 = snap64
			d27 = snap65
			d28 = snap66
			d29 = snap67
			d30 = snap68
			d31 = snap69
			d32 = snap70
			d33 = snap71
			d34 = snap72
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps37)
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
			d74 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d74)
			ctx.BindReg(r1, &d74)
			ctx.EmitMakeNil(d74)
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
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d32)
			var d75 scm.JITValueDesc
			if d32.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d32.Imm.Int()))))}
			} else {
				r45 := ctx.AllocReg()
				ctx.EmitMovRegReg(r45, d32.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d75)
			}
			var d76 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
				r46 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r46, fieldAddr)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
				ctx.BindReg(r46, &d76)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
				r47 := ctx.AllocReg()
				ctx.EmitMovRegMem(r47, thisptr.Reg, off)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d76)
			}
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d75)
			ctx.ProtectReg(d75.Reg)
			ctx.EnsureDesc(&d76)
			ctx.UnprotectReg(d75.Reg)
			var d77 scm.JITValueDesc
			if d75.Loc == scm.LocImm && d76.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() + d76.Imm.Int())}
			} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
				r48 := ctx.AllocRegExcept(d75.Reg)
				ctx.EmitMovRegReg(r48, d75.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d77)
			} else if d75.Loc == scm.LocImm && d75.Imm.Int() == 0 {
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d76.Reg}
				ctx.BindReg(d76.Reg, &d77)
			} else if d75.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d76.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d75.Imm.Int()))
				ctx.EmitAddInt64(scratch, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			} else if d76.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.EmitMovRegReg(scratch, d75.Reg)
				if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d76.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			} else {
				r49 := ctx.AllocRegExcept(d75.Reg, d76.Reg)
				ctx.EmitMovRegReg(r49, d75.Reg)
				ctx.EmitAddInt64(r49, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d77)
			}
			if d77.Loc == scm.LocReg && d75.Loc == scm.LocReg && d77.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d77)
			d78 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d78)
			ctx.BindReg(r1, &d78)
			ctx.EnsureDesc(&d77)
			ctx.EmitMakeInt(d78, d77)
			if d77.Loc == scm.LocReg { ctx.FreeReg(d77.Reg) }
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
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			ctx.ReclaimUntrackedRegs()
			var d79 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
				r50 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r50, fieldAddr)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d79)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
				r51 := ctx.AllocReg()
				ctx.EmitMovRegMem(r51, thisptr.Reg, off)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d79)
			}
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d79)
			var d80 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d79.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d32.Imm.Int()) == uint64(d79.Imm.Int()))}
			} else if d79.Loc == scm.LocImm {
				r52 := ctx.AllocRegExcept(d32.Reg)
				if d79.Imm.Int() >= -2147483648 && d79.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d32.Reg, int32(d79.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
					ctx.EmitCmpInt64(d32.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r52, scm.CcE)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
				ctx.BindReg(r52, &d80)
			} else if d32.Loc == scm.LocImm {
				r53 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d79.Reg)
				ctx.EmitSetcc(r53, scm.CcE)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d80)
			} else {
				r54 := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitCmpInt64(d32.Reg, d79.Reg)
				ctx.EmitSetcc(r54, scm.CcE)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d80)
			}
			ctx.FreeDesc(&d32)
			d81 = d80
			ctx.EnsureDesc(&d81)
			if d81.Loc != scm.LocImm && d81.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d81.Loc == scm.LocImm {
				if d81.Imm.Bool() {
			ps82 := scm.PhiState{General: ps.General}
			ps82.OverlayValues = make([]scm.JITValueDesc, 82)
			ps82.OverlayValues[0] = d0
			ps82.OverlayValues[2] = d2
			ps82.OverlayValues[3] = d3
			ps82.OverlayValues[4] = d4
			ps82.OverlayValues[5] = d5
			ps82.OverlayValues[6] = d6
			ps82.OverlayValues[7] = d7
			ps82.OverlayValues[8] = d8
			ps82.OverlayValues[9] = d9
			ps82.OverlayValues[10] = d10
			ps82.OverlayValues[11] = d11
			ps82.OverlayValues[12] = d12
			ps82.OverlayValues[13] = d13
			ps82.OverlayValues[14] = d14
			ps82.OverlayValues[15] = d15
			ps82.OverlayValues[16] = d16
			ps82.OverlayValues[17] = d17
			ps82.OverlayValues[18] = d18
			ps82.OverlayValues[19] = d19
			ps82.OverlayValues[20] = d20
			ps82.OverlayValues[21] = d21
			ps82.OverlayValues[22] = d22
			ps82.OverlayValues[23] = d23
			ps82.OverlayValues[24] = d24
			ps82.OverlayValues[25] = d25
			ps82.OverlayValues[26] = d26
			ps82.OverlayValues[27] = d27
			ps82.OverlayValues[28] = d28
			ps82.OverlayValues[29] = d29
			ps82.OverlayValues[30] = d30
			ps82.OverlayValues[31] = d31
			ps82.OverlayValues[32] = d32
			ps82.OverlayValues[33] = d33
			ps82.OverlayValues[34] = d34
			ps82.OverlayValues[74] = d74
			ps82.OverlayValues[75] = d75
			ps82.OverlayValues[76] = d76
			ps82.OverlayValues[77] = d77
			ps82.OverlayValues[78] = d78
			ps82.OverlayValues[79] = d79
			ps82.OverlayValues[80] = d80
			ps82.OverlayValues[81] = d81
					return bbs[1].RenderPS(ps82)
				}
			ps83 := scm.PhiState{General: ps.General}
			ps83.OverlayValues = make([]scm.JITValueDesc, 82)
			ps83.OverlayValues[0] = d0
			ps83.OverlayValues[2] = d2
			ps83.OverlayValues[3] = d3
			ps83.OverlayValues[4] = d4
			ps83.OverlayValues[5] = d5
			ps83.OverlayValues[6] = d6
			ps83.OverlayValues[7] = d7
			ps83.OverlayValues[8] = d8
			ps83.OverlayValues[9] = d9
			ps83.OverlayValues[10] = d10
			ps83.OverlayValues[11] = d11
			ps83.OverlayValues[12] = d12
			ps83.OverlayValues[13] = d13
			ps83.OverlayValues[14] = d14
			ps83.OverlayValues[15] = d15
			ps83.OverlayValues[16] = d16
			ps83.OverlayValues[17] = d17
			ps83.OverlayValues[18] = d18
			ps83.OverlayValues[19] = d19
			ps83.OverlayValues[20] = d20
			ps83.OverlayValues[21] = d21
			ps83.OverlayValues[22] = d22
			ps83.OverlayValues[23] = d23
			ps83.OverlayValues[24] = d24
			ps83.OverlayValues[25] = d25
			ps83.OverlayValues[26] = d26
			ps83.OverlayValues[27] = d27
			ps83.OverlayValues[28] = d28
			ps83.OverlayValues[29] = d29
			ps83.OverlayValues[30] = d30
			ps83.OverlayValues[31] = d31
			ps83.OverlayValues[32] = d32
			ps83.OverlayValues[33] = d33
			ps83.OverlayValues[34] = d34
			ps83.OverlayValues[74] = d74
			ps83.OverlayValues[75] = d75
			ps83.OverlayValues[76] = d76
			ps83.OverlayValues[77] = d77
			ps83.OverlayValues[78] = d78
			ps83.OverlayValues[79] = d79
			ps83.OverlayValues[80] = d80
			ps83.OverlayValues[81] = d81
				return bbs[2].RenderPS(ps83)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d81.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl12)
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl12)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl3)
			ps84 := scm.PhiState{General: true}
			ps84.OverlayValues = make([]scm.JITValueDesc, 82)
			ps84.OverlayValues[0] = d0
			ps84.OverlayValues[2] = d2
			ps84.OverlayValues[3] = d3
			ps84.OverlayValues[4] = d4
			ps84.OverlayValues[5] = d5
			ps84.OverlayValues[6] = d6
			ps84.OverlayValues[7] = d7
			ps84.OverlayValues[8] = d8
			ps84.OverlayValues[9] = d9
			ps84.OverlayValues[10] = d10
			ps84.OverlayValues[11] = d11
			ps84.OverlayValues[12] = d12
			ps84.OverlayValues[13] = d13
			ps84.OverlayValues[14] = d14
			ps84.OverlayValues[15] = d15
			ps84.OverlayValues[16] = d16
			ps84.OverlayValues[17] = d17
			ps84.OverlayValues[18] = d18
			ps84.OverlayValues[19] = d19
			ps84.OverlayValues[20] = d20
			ps84.OverlayValues[21] = d21
			ps84.OverlayValues[22] = d22
			ps84.OverlayValues[23] = d23
			ps84.OverlayValues[24] = d24
			ps84.OverlayValues[25] = d25
			ps84.OverlayValues[26] = d26
			ps84.OverlayValues[27] = d27
			ps84.OverlayValues[28] = d28
			ps84.OverlayValues[29] = d29
			ps84.OverlayValues[30] = d30
			ps84.OverlayValues[31] = d31
			ps84.OverlayValues[32] = d32
			ps84.OverlayValues[33] = d33
			ps84.OverlayValues[34] = d34
			ps84.OverlayValues[74] = d74
			ps84.OverlayValues[75] = d75
			ps84.OverlayValues[76] = d76
			ps84.OverlayValues[77] = d77
			ps84.OverlayValues[78] = d78
			ps84.OverlayValues[79] = d79
			ps84.OverlayValues[80] = d80
			ps84.OverlayValues[81] = d81
			ps85 := scm.PhiState{General: true}
			ps85.OverlayValues = make([]scm.JITValueDesc, 82)
			ps85.OverlayValues[0] = d0
			ps85.OverlayValues[2] = d2
			ps85.OverlayValues[3] = d3
			ps85.OverlayValues[4] = d4
			ps85.OverlayValues[5] = d5
			ps85.OverlayValues[6] = d6
			ps85.OverlayValues[7] = d7
			ps85.OverlayValues[8] = d8
			ps85.OverlayValues[9] = d9
			ps85.OverlayValues[10] = d10
			ps85.OverlayValues[11] = d11
			ps85.OverlayValues[12] = d12
			ps85.OverlayValues[13] = d13
			ps85.OverlayValues[14] = d14
			ps85.OverlayValues[15] = d15
			ps85.OverlayValues[16] = d16
			ps85.OverlayValues[17] = d17
			ps85.OverlayValues[18] = d18
			ps85.OverlayValues[19] = d19
			ps85.OverlayValues[20] = d20
			ps85.OverlayValues[21] = d21
			ps85.OverlayValues[22] = d22
			ps85.OverlayValues[23] = d23
			ps85.OverlayValues[24] = d24
			ps85.OverlayValues[25] = d25
			ps85.OverlayValues[26] = d26
			ps85.OverlayValues[27] = d27
			ps85.OverlayValues[28] = d28
			ps85.OverlayValues[29] = d29
			ps85.OverlayValues[30] = d30
			ps85.OverlayValues[31] = d31
			ps85.OverlayValues[32] = d32
			ps85.OverlayValues[33] = d33
			ps85.OverlayValues[34] = d34
			ps85.OverlayValues[74] = d74
			ps85.OverlayValues[75] = d75
			ps85.OverlayValues[76] = d76
			ps85.OverlayValues[77] = d77
			ps85.OverlayValues[78] = d78
			ps85.OverlayValues[79] = d79
			ps85.OverlayValues[80] = d80
			ps85.OverlayValues[81] = d81
			snap86 := d0
			snap87 := d2
			snap88 := d3
			snap89 := d4
			snap90 := d5
			snap91 := d6
			snap92 := d7
			snap93 := d8
			snap94 := d9
			snap95 := d10
			snap96 := d11
			snap97 := d12
			snap98 := d13
			snap99 := d14
			snap100 := d15
			snap101 := d16
			snap102 := d17
			snap103 := d18
			snap104 := d19
			snap105 := d20
			snap106 := d21
			snap107 := d22
			snap108 := d23
			snap109 := d24
			snap110 := d25
			snap111 := d26
			snap112 := d27
			snap113 := d28
			snap114 := d29
			snap115 := d30
			snap116 := d31
			snap117 := d32
			snap118 := d33
			snap119 := d34
			snap120 := d74
			snap121 := d75
			snap122 := d76
			snap123 := d77
			snap124 := d78
			snap125 := d79
			snap126 := d80
			snap127 := d81
			alloc128 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps85)
			}
			ctx.RestoreAllocState(alloc128)
			d0 = snap86
			d2 = snap87
			d3 = snap88
			d4 = snap89
			d5 = snap90
			d6 = snap91
			d7 = snap92
			d8 = snap93
			d9 = snap94
			d10 = snap95
			d11 = snap96
			d12 = snap97
			d13 = snap98
			d14 = snap99
			d15 = snap100
			d16 = snap101
			d17 = snap102
			d18 = snap103
			d19 = snap104
			d20 = snap105
			d21 = snap106
			d22 = snap107
			d23 = snap108
			d24 = snap109
			d25 = snap110
			d26 = snap111
			d27 = snap112
			d28 = snap113
			d29 = snap114
			d30 = snap115
			d31 = snap116
			d32 = snap117
			d33 = snap118
			d34 = snap119
			d74 = snap120
			d75 = snap121
			d76 = snap122
			d77 = snap123
			d78 = snap124
			d79 = snap125
			d80 = snap126
			d81 = snap127
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps84)
			}
			return result
			ctx.FreeDesc(&d80)
			return result
			}
			ps129 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps129)
			ctx.MarkLabel(lbl0)
			d130 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d130)
			ctx.BindReg(r1, &d130)
			ctx.EmitMovPairToResult(&d130, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
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

func (s *StorageInt) prepare() {
	// set up scan
	s.bitsize = 0
	s.offset = int64(1<<63 - 1)
	s.max = -s.offset - 1
	s.hasNull = false
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
