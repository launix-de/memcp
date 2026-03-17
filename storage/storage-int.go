/*
Copyright (C) 2023  Carl-Philip Hänsch

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
			var r6 unsafe.Pointer
			_ = r6
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
			var d73 scm.JITValueDesc
			_ = d73
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
			r6 = ctx.EmitSubRSP32Fixup()
			_ = r6
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			lbl5 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d2 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
			} else {
				r7 := ctx.AllocReg()
				ctx.EmitMovRegReg(r7, d0.Reg)
				ctx.EmitShlRegImm8(r7, 32)
				ctx.EmitShrRegImm8(r7, 32)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d2)
			}
			var d3 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				r8 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r8, fieldAddr)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
				ctx.BindReg(r8, &d3)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r9 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r9, thisptr.Reg, off)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d3)
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d4 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d3.Imm.Int()))))}
			} else {
				r10 := ctx.AllocReg()
				ctx.EmitMovRegReg(r10, d3.Reg)
				ctx.EmitShlRegImm8(r10, 56)
				ctx.EmitShrRegImm8(r10, 56)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d4)
			}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			ctx.ProtectReg(d2.Reg)
			ctx.EnsureDesc(&d4)
			ctx.UnprotectReg(d2.Reg)
			var d5 scm.JITValueDesc
			if d2.Loc == scm.LocImm && d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() * d4.Imm.Int())}
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.EmitImulInt64(scratch, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d4.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else {
				r11 := ctx.AllocRegExcept(d2.Reg, d4.Reg)
				ctx.EmitMovRegReg(r11, d2.Reg)
				ctx.EmitImulInt64(r11, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d5)
			}
			if d5.Loc == scm.LocReg && d2.Loc == scm.LocReg && d5.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.FreeDesc(&d4)
			var d6 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
				r12 := ctx.AllocReg()
				r13 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r12, fieldAddr)
				ctx.EmitMovRegMem64(r13, fieldAddr+8)
				d6 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r12, Reg2: r13}
				ctx.BindReg(r12, &d6)
				ctx.BindReg(r13, &d6)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
				r14 := ctx.AllocReg()
				r15 := ctx.AllocReg()
				ctx.EmitMovRegMem(r14, thisptr.Reg, off)
				ctx.EmitMovRegMem(r15, thisptr.Reg, off+8)
				d6 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r14, Reg2: r15}
				ctx.BindReg(r14, &d6)
				ctx.BindReg(r15, &d6)
			}
			ctx.EnsureDesc(&d5)
			var d7 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r16 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r16, d5.Reg)
				ctx.EmitShrRegImm8(r16, 6)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d7)
			}
			if d7.Loc == scm.LocReg && d5.Loc == scm.LocReg && d7.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d7)
			r17 := ctx.AllocReg()
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			if d7.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r17, uint64(d7.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r17, d7.Reg)
				ctx.EmitShlRegImm8(r17, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitAddInt64(r17, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r17, d6.Reg)
			}
			r18 := ctx.AllocRegExcept(r17)
			ctx.EmitMovRegMem(r18, r17, 0)
			ctx.FreeReg(r17)
			d8 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
			ctx.BindReg(r18, &d8)
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d5)
			var d9 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r19 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r19, d5.Reg)
				ctx.EmitAndRegImm32(r19, 63)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d9)
			}
			if d9.Loc == scm.LocReg && d5.Loc == scm.LocReg && d9.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			var d10 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d9.Imm.Int())))}
			} else if d9.Loc == scm.LocImm {
				r20 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r20, d8.Reg)
				ctx.EmitShlRegImm8(r20, uint8(d9.Imm.Int()))
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d10)
			} else {
				{
					shiftSrc := d8.Reg
					r21 := ctx.AllocRegExcept(d8.Reg)
					ctx.EmitMovRegReg(r21, d8.Reg)
					shiftSrc = r21
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d9.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d9.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d9.Reg)
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
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d5)
			var d11 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r22 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r22, d5.Reg)
				ctx.EmitAndRegImm32(r22, 63)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d11)
			}
			if d11.Loc == scm.LocReg && d5.Loc == scm.LocReg && d11.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d12 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d3.Imm.Int()))))}
			} else {
				r23 := ctx.AllocReg()
				ctx.EmitMovRegReg(r23, d3.Reg)
				ctx.EmitShlRegImm8(r23, 56)
				ctx.EmitShrRegImm8(r23, 56)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d12)
			}
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d11)
			ctx.ProtectReg(d11.Reg)
			ctx.EnsureDesc(&d12)
			ctx.UnprotectReg(d11.Reg)
			var d13 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() + d12.Imm.Int())}
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				r24 := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(r24, d11.Reg)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d13)
			} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d12.Reg}
				ctx.BindReg(d12.Reg, &d13)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.EmitAddInt64(scratch, d12.Reg)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d13)
			} else if d12.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(scratch, d11.Reg)
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d12.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d13)
			} else {
				r25 := ctx.AllocRegExcept(d11.Reg, d12.Reg)
				ctx.EmitMovRegReg(r25, d11.Reg)
				ctx.EmitAddInt64(r25, d12.Reg)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d13)
			}
			if d13.Loc == scm.LocReg && d11.Loc == scm.LocReg && d13.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d12)
			ctx.EnsureDesc(&d13)
			var d14 scm.JITValueDesc
			if d13.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d13.Imm.Int()) > uint64(64))}
			} else {
				r26 := ctx.AllocRegExcept(d13.Reg)
				ctx.EmitCmpRegImm32(d13.Reg, 64)
				ctx.EmitSetcc(r26, scm.CcA)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r26}
				ctx.BindReg(r26, &d14)
			}
			ctx.FreeDesc(&d13)
			d15 = d14
			ctx.EnsureDesc(&d15)
			if d15.Loc != scm.LocImm && d15.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl6 := ctx.ReserveLabel()
			lbl7 := ctx.ReserveLabel()
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			if d15.Loc == scm.LocImm {
				if d15.Imm.Bool() {
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl6)
				} else {
					ctx.MarkLabel(lbl9)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d16 = d10
			if d16.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
					ctx.EmitJmp(lbl7)
				}
			} else {
				ctx.EmitCmpRegImm32(d15.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl8)
				ctx.EmitJmp(lbl9)
				ctx.MarkLabel(lbl8)
				ctx.EmitJmp(lbl6)
				ctx.MarkLabel(lbl9)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d17 = d10
			if d17.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			ctx.EmitStoreToStack(d17, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
				ctx.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d14)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl7)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d18 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d3.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.EmitMovRegReg(r27, d3.Reg)
				ctx.EmitShlRegImm8(r27, 56)
				ctx.EmitShrRegImm8(r27, 56)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d18)
			}
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d19)
			ctx.ProtectReg(d19.Reg)
			ctx.EnsureDesc(&d18)
			ctx.UnprotectReg(d19.Reg)
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() - d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d19.Reg)
				ctx.EmitMovRegReg(r28, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d20)
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
				r29 := ctx.AllocRegExcept(d19.Reg, d18.Reg)
				ctx.EmitMovRegReg(r29, d19.Reg)
				ctx.EmitSubInt64(r29, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d20)
			}
			if d20.Loc == scm.LocReg && d19.Loc == scm.LocReg && d20.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d20)
			var d21 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1.Imm.Int()) >> uint64(d20.Imm.Int())))}
			} else if d20.Loc == scm.LocImm {
				r30 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(r30, d1.Reg)
				ctx.EmitShrRegImm8(r30, uint8(d20.Imm.Int()))
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d21)
			} else {
				{
					shiftSrc := d1.Reg
					r31 := ctx.AllocRegExcept(d1.Reg)
					ctx.EmitMovRegReg(r31, d1.Reg)
					shiftSrc = r31
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d20.Reg != scm.RegRCX
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
			if d21.Loc == scm.LocReg && d1.Loc == scm.LocReg && d21.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d20)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			if d21.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r32, d21)
			}
			ctx.EmitJmp(lbl5)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl6)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d22 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r33 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r33, d5.Reg)
				ctx.EmitShrRegImm8(r33, 6)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d22)
			}
			if d22.Loc == scm.LocReg && d5.Loc == scm.LocReg && d22.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.EmitMovRegReg(scratch, d22.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			ctx.EnsureDesc(&d23)
			r34 := ctx.AllocReg()
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d6)
			if d23.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r34, uint64(d23.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r34, d23.Reg)
				ctx.EmitShlRegImm8(r34, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitAddInt64(r34, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r34, d6.Reg)
			}
			r35 := ctx.AllocRegExcept(r34)
			ctx.EmitMovRegMem(r35, r34, 0)
			ctx.FreeReg(r34)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d24)
			ctx.FreeDesc(&d23)
			ctx.EnsureDesc(&d5)
			var d25 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r36 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r36, d5.Reg)
				ctx.EmitAndRegImm32(r36, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d25)
			}
			if d25.Loc == scm.LocReg && d5.Loc == scm.LocReg && d25.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d5)
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d26)
			ctx.ProtectReg(d26.Reg)
			ctx.EnsureDesc(&d25)
			ctx.UnprotectReg(d26.Reg)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegReg(r37, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d27)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.EmitSubInt64(scratch, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegReg(scratch, d26.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else {
				r38 := ctx.AllocRegExcept(d26.Reg, d25.Reg)
				ctx.EmitMovRegReg(r38, d26.Reg)
				ctx.EmitSubInt64(r38, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d27)
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d27)
			var d28 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d27.Imm.Int())))}
			} else if d27.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d24.Reg)
				ctx.EmitMovRegReg(r39, d24.Reg)
				ctx.EmitShrRegImm8(r39, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d28)
			} else {
				{
					shiftSrc := d24.Reg
					r40 := ctx.AllocRegExcept(d24.Reg)
					ctx.EmitMovRegReg(r40, d24.Reg)
					shiftSrc = r40
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d27.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d27.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d27.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d28)
				}
			}
			if d28.Loc == scm.LocReg && d24.Loc == scm.LocReg && d28.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d27)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() | d28.Imm.Int())}
			} else if d10.Loc == scm.LocImm && d10.Imm.Int() == 0 {
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d28.Reg}
				ctx.BindReg(d28.Reg, &d29)
			} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
				r41 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitMovRegReg(r41, d10.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d29)
			} else if d10.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
				ctx.EmitOrInt64(scratch, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			} else if d28.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitMovRegReg(r42, d10.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r42, int32(d28.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d28.Imm.Int()))
					ctx.EmitOrInt64(r42, scm.RegR11)
				}
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d29)
			} else {
				r43 := ctx.AllocRegExcept(d10.Reg, d28.Reg)
				ctx.EmitMovRegReg(r43, d10.Reg)
				ctx.EmitOrInt64(r43, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d29)
			}
			if d29.Loc == scm.LocReg && d10.Loc == scm.LocReg && d29.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d29)
			if d29.Loc == scm.LocReg {
				ctx.ProtectReg(d29.Reg)
			} else if d29.Loc == scm.LocRegPair {
				ctx.ProtectReg(d29.Reg)
				ctx.ProtectReg(d29.Reg2)
			}
			d30 = d29
			if d30.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d30)
			ctx.EmitStoreToStack(d30, int32(bbs[2].PhiBase)+int32(0))
			if d29.Loc == scm.LocReg {
				ctx.UnprotectReg(d29.Reg)
			} else if d29.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d29.Reg)
				ctx.UnprotectReg(d29.Reg2)
			}
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl5)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d31)
			ctx.BindReg(r32, &d31)
			if r2 { ctx.UnprotectReg(r3) }
			if r4 { ctx.UnprotectReg(r5) }
			ctx.FreeDesc(&idxInt)
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
				r44 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r44, fieldAddr)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
				ctx.BindReg(r44, &d32)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
				r45 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r45, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d32)
			}
			d33 = d32
			ctx.EnsureDesc(&d33)
			if d33.Loc != scm.LocImm && d33.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d33.Loc == scm.LocImm {
				if d33.Imm.Bool() {
			ps34 := scm.PhiState{General: ps.General}
			ps34.OverlayValues = make([]scm.JITValueDesc, 34)
			ps34.OverlayValues[0] = d0
			ps34.OverlayValues[1] = d1
			ps34.OverlayValues[2] = d2
			ps34.OverlayValues[3] = d3
			ps34.OverlayValues[4] = d4
			ps34.OverlayValues[5] = d5
			ps34.OverlayValues[6] = d6
			ps34.OverlayValues[7] = d7
			ps34.OverlayValues[8] = d8
			ps34.OverlayValues[9] = d9
			ps34.OverlayValues[10] = d10
			ps34.OverlayValues[11] = d11
			ps34.OverlayValues[12] = d12
			ps34.OverlayValues[13] = d13
			ps34.OverlayValues[14] = d14
			ps34.OverlayValues[15] = d15
			ps34.OverlayValues[16] = d16
			ps34.OverlayValues[17] = d17
			ps34.OverlayValues[18] = d18
			ps34.OverlayValues[19] = d19
			ps34.OverlayValues[20] = d20
			ps34.OverlayValues[21] = d21
			ps34.OverlayValues[22] = d22
			ps34.OverlayValues[23] = d23
			ps34.OverlayValues[24] = d24
			ps34.OverlayValues[25] = d25
			ps34.OverlayValues[26] = d26
			ps34.OverlayValues[27] = d27
			ps34.OverlayValues[28] = d28
			ps34.OverlayValues[29] = d29
			ps34.OverlayValues[30] = d30
			ps34.OverlayValues[31] = d31
			ps34.OverlayValues[32] = d32
			ps34.OverlayValues[33] = d33
					return bbs[3].RenderPS(ps34)
				}
			ps35 := scm.PhiState{General: ps.General}
			ps35.OverlayValues = make([]scm.JITValueDesc, 34)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[1] = d1
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
				return bbs[2].RenderPS(ps35)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl10 := ctx.ReserveLabel()
			lbl11 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d33.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl10)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl11)
			ctx.EmitJmp(lbl3)
			ps36 := scm.PhiState{General: true}
			ps36.OverlayValues = make([]scm.JITValueDesc, 34)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[1] = d1
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
			ps37 := scm.PhiState{General: true}
			ps37.OverlayValues = make([]scm.JITValueDesc, 34)
			ps37.OverlayValues[0] = d0
			ps37.OverlayValues[1] = d1
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
			snap38 := d0
			snap39 := d1
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
			alloc72 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps37)
			}
			ctx.RestoreAllocState(alloc72)
			d0 = snap38
			d1 = snap39
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
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps36)
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
			ctx.ReclaimUntrackedRegs()
			d73 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d73)
			ctx.BindReg(r1, &d73)
			ctx.EmitMakeNil(d73)
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
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d31)
			var d74 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d31.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.EmitMovRegReg(r46, d31.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d74)
			}
			var d75 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
				r47 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r47, fieldAddr)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d75)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
				r48 := ctx.AllocReg()
				ctx.EmitMovRegMem(r48, thisptr.Reg, off)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d75)
			}
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d74)
			ctx.ProtectReg(d74.Reg)
			ctx.EnsureDesc(&d75)
			ctx.UnprotectReg(d74.Reg)
			var d76 scm.JITValueDesc
			if d74.Loc == scm.LocImm && d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() + d75.Imm.Int())}
			} else if d75.Loc == scm.LocImm && d75.Imm.Int() == 0 {
				r49 := ctx.AllocRegExcept(d74.Reg)
				ctx.EmitMovRegReg(r49, d74.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d76)
			} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d75.Reg}
				ctx.BindReg(d75.Reg, &d76)
			} else if d74.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
				ctx.EmitAddInt64(scratch, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			} else if d75.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.EmitMovRegReg(scratch, d74.Reg)
				if d75.Imm.Int() >= -2147483648 && d75.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d75.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			} else {
				r50 := ctx.AllocRegExcept(d74.Reg, d75.Reg)
				ctx.EmitMovRegReg(r50, d74.Reg)
				ctx.EmitAddInt64(r50, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d76)
			}
			if d76.Loc == scm.LocReg && d74.Loc == scm.LocReg && d76.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			ctx.EnsureDesc(&d76)
			d77 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d77)
			ctx.BindReg(r1, &d77)
			ctx.EnsureDesc(&d76)
			ctx.EmitMakeInt(d77, d76)
			if d76.Loc == scm.LocReg { ctx.FreeReg(d76.Reg) }
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
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
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
			ctx.ReclaimUntrackedRegs()
			var d78 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
				r51 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r51, fieldAddr)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d78)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
				r52 := ctx.AllocReg()
				ctx.EmitMovRegMem(r52, thisptr.Reg, off)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
				ctx.BindReg(r52, &d78)
			}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d78)
			var d79 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d31.Imm.Int()) == uint64(d78.Imm.Int()))}
			} else if d78.Loc == scm.LocImm {
				r53 := ctx.AllocRegExcept(d31.Reg)
				if d78.Imm.Int() >= -2147483648 && d78.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d31.Reg, int32(d78.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
					ctx.EmitCmpInt64(d31.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r53, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d79)
			} else if d31.Loc == scm.LocImm {
				r54 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d78.Reg)
				ctx.EmitSetcc(r54, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d79)
			} else {
				r55 := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitCmpInt64(d31.Reg, d78.Reg)
				ctx.EmitSetcc(r55, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d79)
			}
			ctx.FreeDesc(&d31)
			d80 = d79
			ctx.EnsureDesc(&d80)
			if d80.Loc != scm.LocImm && d80.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d80.Loc == scm.LocImm {
				if d80.Imm.Bool() {
			ps81 := scm.PhiState{General: ps.General}
			ps81.OverlayValues = make([]scm.JITValueDesc, 81)
			ps81.OverlayValues[0] = d0
			ps81.OverlayValues[1] = d1
			ps81.OverlayValues[2] = d2
			ps81.OverlayValues[3] = d3
			ps81.OverlayValues[4] = d4
			ps81.OverlayValues[5] = d5
			ps81.OverlayValues[6] = d6
			ps81.OverlayValues[7] = d7
			ps81.OverlayValues[8] = d8
			ps81.OverlayValues[9] = d9
			ps81.OverlayValues[10] = d10
			ps81.OverlayValues[11] = d11
			ps81.OverlayValues[12] = d12
			ps81.OverlayValues[13] = d13
			ps81.OverlayValues[14] = d14
			ps81.OverlayValues[15] = d15
			ps81.OverlayValues[16] = d16
			ps81.OverlayValues[17] = d17
			ps81.OverlayValues[18] = d18
			ps81.OverlayValues[19] = d19
			ps81.OverlayValues[20] = d20
			ps81.OverlayValues[21] = d21
			ps81.OverlayValues[22] = d22
			ps81.OverlayValues[23] = d23
			ps81.OverlayValues[24] = d24
			ps81.OverlayValues[25] = d25
			ps81.OverlayValues[26] = d26
			ps81.OverlayValues[27] = d27
			ps81.OverlayValues[28] = d28
			ps81.OverlayValues[29] = d29
			ps81.OverlayValues[30] = d30
			ps81.OverlayValues[31] = d31
			ps81.OverlayValues[32] = d32
			ps81.OverlayValues[33] = d33
			ps81.OverlayValues[73] = d73
			ps81.OverlayValues[74] = d74
			ps81.OverlayValues[75] = d75
			ps81.OverlayValues[76] = d76
			ps81.OverlayValues[77] = d77
			ps81.OverlayValues[78] = d78
			ps81.OverlayValues[79] = d79
			ps81.OverlayValues[80] = d80
					return bbs[1].RenderPS(ps81)
				}
			ps82 := scm.PhiState{General: ps.General}
			ps82.OverlayValues = make([]scm.JITValueDesc, 81)
			ps82.OverlayValues[0] = d0
			ps82.OverlayValues[1] = d1
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
			ps82.OverlayValues[73] = d73
			ps82.OverlayValues[74] = d74
			ps82.OverlayValues[75] = d75
			ps82.OverlayValues[76] = d76
			ps82.OverlayValues[77] = d77
			ps82.OverlayValues[78] = d78
			ps82.OverlayValues[79] = d79
			ps82.OverlayValues[80] = d80
				return bbs[2].RenderPS(ps82)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d80.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl12)
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl12)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl3)
			ps83 := scm.PhiState{General: true}
			ps83.OverlayValues = make([]scm.JITValueDesc, 81)
			ps83.OverlayValues[0] = d0
			ps83.OverlayValues[1] = d1
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
			ps83.OverlayValues[73] = d73
			ps83.OverlayValues[74] = d74
			ps83.OverlayValues[75] = d75
			ps83.OverlayValues[76] = d76
			ps83.OverlayValues[77] = d77
			ps83.OverlayValues[78] = d78
			ps83.OverlayValues[79] = d79
			ps83.OverlayValues[80] = d80
			ps84 := scm.PhiState{General: true}
			ps84.OverlayValues = make([]scm.JITValueDesc, 81)
			ps84.OverlayValues[0] = d0
			ps84.OverlayValues[1] = d1
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
			ps84.OverlayValues[73] = d73
			ps84.OverlayValues[74] = d74
			ps84.OverlayValues[75] = d75
			ps84.OverlayValues[76] = d76
			ps84.OverlayValues[77] = d77
			ps84.OverlayValues[78] = d78
			ps84.OverlayValues[79] = d79
			ps84.OverlayValues[80] = d80
			snap85 := d0
			snap86 := d1
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
			snap119 := d73
			snap120 := d74
			snap121 := d75
			snap122 := d76
			snap123 := d77
			snap124 := d78
			snap125 := d79
			snap126 := d80
			alloc127 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps84)
			}
			ctx.RestoreAllocState(alloc127)
			d0 = snap85
			d1 = snap86
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
			d73 = snap119
			d74 = snap120
			d75 = snap121
			d76 = snap122
			d77 = snap123
			d78 = snap124
			d79 = snap125
			d80 = snap126
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps83)
			}
			return result
			ctx.FreeDesc(&d79)
			return result
			}
			ps128 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps128)
			ctx.MarkLabel(lbl0)
			d129 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d129)
			ctx.BindReg(r1, &d129)
			ctx.EmitMovPairToResult(&d129, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r6, int32(16))
			ctx.EmitAddRSP32(int32(16))
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
