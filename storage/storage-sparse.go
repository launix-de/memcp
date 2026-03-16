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
import "bufio"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"
import "unsafe"

type StorageSparse struct {
	i, count uint64
	recids   StorageInt
	values   []scm.Scmer // TODO: embed other formats as values (ColumnStorage with a proposeCompression loop)
}

func (s *StorageSparse) ComputeSize() uint {
	var sz uint = 16 + 8 + 24 + s.recids.ComputeSize() + 8*uint(len(s.values))
	for _, v := range s.values {
		sz += scm.ComputeSize(v)
	}
	return sz
}

func (s *StorageSparse) String() string {
	return "SCMER-sparse"
}

// StorageSparse binary layout (magic byte 2 consumed by shard loader):
//
//	[count uint64]         ← total row count (including NULL rows)
//	[l2 uint64]            ← number of non-NULL (sparse) entries
//	[entries: l2 pairs of JSON lines: recid\nvalue\n]
//
// Version history:
//
//	v0 (original, no version byte): layout as above.  This type had no padding
//	byte in v0.1.0, so there is no safe location for a version byte without
//	breaking existing data.  If the format must change, register a NEW magic
//	byte in storages[] (storage.go) for the new layout and keep magic 2 for
//	reading legacy data forever.

func (s *StorageSparse) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var d2 scm.JITValueDesc
			_ = d2
			var d3 scm.JITValueDesc
			_ = d3
			var d4 scm.JITValueDesc
			_ = d4
			var d5 scm.JITValueDesc
			_ = d5
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
			var d15 scm.JITValueDesc
			_ = d15
			var d16 scm.JITValueDesc
			_ = d16
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
			var d62 scm.JITValueDesc
			_ = d62
			var d63 scm.JITValueDesc
			_ = d63
			var d64 scm.JITValueDesc
			_ = d64
			var d65 scm.JITValueDesc
			_ = d65
			var d66 scm.JITValueDesc
			_ = d66
			var d67 scm.JITValueDesc
			_ = d67
			var d68 scm.JITValueDesc
			_ = d68
			var d69 scm.JITValueDesc
			_ = d69
			var d70 scm.JITValueDesc
			_ = d70
			var d71 scm.JITValueDesc
			_ = d71
			var d72 scm.JITValueDesc
			_ = d72
			var d73 scm.JITValueDesc
			_ = d73
			var d74 scm.JITValueDesc
			_ = d74
			var d135 scm.JITValueDesc
			_ = d135
			var d136 scm.JITValueDesc
			_ = d136
			var d137 scm.JITValueDesc
			_ = d137
			var d138 scm.JITValueDesc
			_ = d138
			var d139 scm.JITValueDesc
			_ = d139
			var d140 scm.JITValueDesc
			_ = d140
			var d207 scm.JITValueDesc
			_ = d207
			var d208 scm.JITValueDesc
			_ = d208
			var d209 scm.JITValueDesc
			_ = d209
			var d210 scm.JITValueDesc
			_ = d210
			var d211 scm.JITValueDesc
			_ = d211
			var d213 scm.JITValueDesc
			_ = d213
			var d214 scm.JITValueDesc
			_ = d214
			var d215 scm.JITValueDesc
			_ = d215
			var d216 scm.JITValueDesc
			_ = d216
			var d217 scm.JITValueDesc
			_ = d217
			var d218 scm.JITValueDesc
			_ = d218
			var d220 scm.JITValueDesc
			_ = d220
			var d221 scm.JITValueDesc
			_ = d221
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
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			if thisptr.MemPtr == 0 && (thisptr.Loc == scm.LocStack || thisptr.Loc == scm.LocStackPair) {
				thisptr.StackOff += int32(32)
			}
			if idxInt.MemPtr == 0 && (idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair) {
				idxInt.StackOff += int32(32)
			}
			if result.MemPtr == 0 && (result.Loc == scm.LocStack || result.Loc == scm.LocStackPair) {
				result.StackOff += int32(32)
			}
			d0 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			var bbs [8]scm.BBDescriptor
			bbs[1].PhiBase = int32(0)
			bbs[1].PhiCount = uint16(2)
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
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
			bbs[0].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).i)
				r3 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r3, fieldAddr)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
				ctx.BindReg(r3, &d2)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
				r4 := ctx.AllocReg()
				ctx.EmitMovRegMem(r4, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
				ctx.BindReg(r4, &d2)
			}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d2.Imm.Int()))))}
			} else {
				r5 := ctx.AllocReg()
				ctx.EmitMovRegReg(r5, d2.Reg)
				ctx.EmitShlRegImm8(r5, 32)
				ctx.EmitShrRegImm8(r5, 32)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
				ctx.BindReg(r5, &d3)
			}
			ctx.EnsureDesc(&d3)
			if d3.Loc == scm.LocReg {
				ctx.ProtectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.ProtectReg(d3.Reg)
				ctx.ProtectReg(d3.Reg2)
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
			d4 = d3
			if d4.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d4)
			d5 = d4
			if d5.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: d5.Type, Imm: scm.NewInt(int64(uint64(d5.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d5.Reg, 32)
				ctx.EmitShrRegImm8(d5.Reg, 32)
			}
			ctx.EmitStoreToStack(d5, int32(bbs[1].PhiBase)+int32(16))
			if d3.Loc == scm.LocReg {
				ctx.UnprotectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d3.Reg)
				ctx.UnprotectReg(d3.Reg2)
			}
			ps6 := scm.PhiState{General: ps.General}
			ps6.OverlayValues = make([]scm.JITValueDesc, 6)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps6.OverlayValues[4] = d4
			ps6.OverlayValues[5] = d5
			ps6.PhiValues = make([]scm.JITValueDesc, 2)
			d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps6.PhiValues[0] = d7
			d8 = d3
			ps6.PhiValues[1] = d8
			if ps6.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps6)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d9 := ps.PhiValues[0]
					ctx.EnsureDesc(&d9)
					ctx.EmitStoreToStack(d9, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d10 := ps.PhiValues[1]
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(bbs[1].PhiBase)+int32(16))
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d0 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d1 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d11 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d0.Imm.Int()) == uint64(d1.Imm.Int()))}
			} else if d1.Loc == scm.LocImm {
				r6 := ctx.AllocRegExcept(d0.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d0.Reg, int32(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
					ctx.EmitCmpInt64(d0.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r6, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d11)
			} else if d0.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d1.Reg)
				ctx.EmitSetcc(r7, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d11)
			} else {
				r8 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitCmpInt64(d0.Reg, d1.Reg)
				ctx.EmitSetcc(r8, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d11)
			}
			d12 = d11
			ctx.EnsureDesc(&d12)
			if d12.Loc != scm.LocImm && d12.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d12.Loc == scm.LocImm {
				if d12.Imm.Bool() {
			ps13 := scm.PhiState{General: ps.General}
			ps13.OverlayValues = make([]scm.JITValueDesc, 13)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[3] = d3
			ps13.OverlayValues[4] = d4
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[7] = d7
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
			ps13.OverlayValues[12] = d12
					return bbs[2].RenderPS(ps13)
				}
			ps14 := scm.PhiState{General: ps.General}
			ps14.OverlayValues = make([]scm.JITValueDesc, 13)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[7] = d7
			ps14.OverlayValues[8] = d8
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
				return bbs[3].RenderPS(ps14)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d15 := ps.PhiValues[0]
					ctx.EnsureDesc(&d15)
					ctx.EmitStoreToStack(d15, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d16 := ps.PhiValues[1]
					ctx.EnsureDesc(&d16)
					ctx.EmitStoreToStack(d16, int32(bbs[1].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d12.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl9)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl4)
			ps17 := scm.PhiState{General: true}
			ps17.OverlayValues = make([]scm.JITValueDesc, 17)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
			ps17.OverlayValues[4] = d4
			ps17.OverlayValues[5] = d5
			ps17.OverlayValues[7] = d7
			ps17.OverlayValues[8] = d8
			ps17.OverlayValues[9] = d9
			ps17.OverlayValues[10] = d10
			ps17.OverlayValues[11] = d11
			ps17.OverlayValues[12] = d12
			ps17.OverlayValues[15] = d15
			ps17.OverlayValues[16] = d16
			ps18 := scm.PhiState{General: true}
			ps18.OverlayValues = make([]scm.JITValueDesc, 17)
			ps18.OverlayValues[0] = d0
			ps18.OverlayValues[1] = d1
			ps18.OverlayValues[2] = d2
			ps18.OverlayValues[3] = d3
			ps18.OverlayValues[4] = d4
			ps18.OverlayValues[5] = d5
			ps18.OverlayValues[7] = d7
			ps18.OverlayValues[8] = d8
			ps18.OverlayValues[9] = d9
			ps18.OverlayValues[10] = d10
			ps18.OverlayValues[11] = d11
			ps18.OverlayValues[12] = d12
			ps18.OverlayValues[15] = d15
			ps18.OverlayValues[16] = d16
			snap19 := d0
			snap20 := d1
			snap21 := d2
			snap22 := d3
			snap23 := d4
			snap24 := d5
			snap25 := d7
			snap26 := d8
			snap27 := d9
			snap28 := d10
			snap29 := d11
			snap30 := d12
			snap31 := d15
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
			d7 = snap25
			d8 = snap26
			d9 = snap27
			d10 = snap28
			d11 = snap29
			d12 = snap30
			d15 = snap31
			d16 = snap32
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps17)
			}
			return result
			ctx.FreeDesc(&d11)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			ctx.ReclaimUntrackedRegs()
			d34 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d34)
			ctx.BindReg(r2, &d34)
			ctx.EmitMakeNil(d34)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[3].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d35 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() + d1.Imm.Int())}
			} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(r9, d0.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d35)
			} else if d0.Loc == scm.LocImm && d0.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d35)
			} else if d0.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(scratch, d1.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else {
				r10 := ctx.AllocRegExcept(d0.Reg, d1.Reg)
				ctx.EmitMovRegReg(r10, d0.Reg)
				ctx.EmitAddInt64(r10, d1.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d35)
			}
			if d35.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: d35.Type, Imm: scm.NewInt(int64(uint64(d35.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d35.Reg, 32)
				ctx.EmitShrRegImm8(d35.Reg, 32)
			}
			if d35.Loc == scm.LocReg && d0.Loc == scm.LocReg && d35.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() / 2)}
			} else {
				r11 := ctx.AllocRegExcept(d35.Reg)
				ctx.EmitMovRegReg(r11, d35.Reg)
				ctx.EmitShrRegImm8(r11, 1)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d36)
			}
			if d36.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: d36.Type, Imm: scm.NewInt(int64(uint64(d36.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d36.Reg, 32)
				ctx.EmitShrRegImm8(d36.Reg, 32)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.EnsureDesc(&d36)
			d37 = d36
			_ = d37
			r12 := d36.Loc == scm.LocReg
			r13 := d36.Reg
			if r12 { ctx.ProtectReg(r13) }
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			lbl11 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d39 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d37.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.EmitMovRegReg(r14, d37.Reg)
				ctx.EmitShlRegImm8(r14, 32)
				ctx.EmitShrRegImm8(r14, 32)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d39)
			}
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				r15 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r15, fieldAddr)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d40)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r16 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r16, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r16}
				ctx.BindReg(r16, &d40)
			}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d40.Imm.Int()))))}
			} else {
				r17 := ctx.AllocReg()
				ctx.EmitMovRegReg(r17, d40.Reg)
				ctx.EmitShlRegImm8(r17, 56)
				ctx.EmitShrRegImm8(r17, 56)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d41)
			}
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() * d41.Imm.Int())}
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.EmitImulInt64(scratch, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.EmitMovRegReg(scratch, d39.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d41.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else {
				r18 := ctx.AllocRegExcept(d39.Reg, d41.Reg)
				ctx.EmitMovRegReg(r18, d39.Reg)
				ctx.EmitImulInt64(r18, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d42)
			}
			if d42.Loc == scm.LocReg && d39.Loc == scm.LocReg && d42.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				r19 := ctx.AllocReg()
				r20 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r19, fieldAddr)
				ctx.EmitMovRegMem64(r20, fieldAddr+8)
				d43 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r19, Reg2: r20}
				ctx.BindReg(r19, &d43)
				ctx.BindReg(r20, &d43)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				r21 := ctx.AllocReg()
				r22 := ctx.AllocReg()
				ctx.EmitMovRegMem(r21, thisptr.Reg, off)
				ctx.EmitMovRegMem(r22, thisptr.Reg, off+8)
				d43 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r21, Reg2: r22}
				ctx.BindReg(r21, &d43)
				ctx.BindReg(r22, &d43)
			}
			ctx.EnsureDesc(&d42)
			var d44 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() / 64)}
			} else {
				r23 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r23, d42.Reg)
				ctx.EmitShrRegImm8(r23, 6)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d44)
			}
			if d44.Loc == scm.LocReg && d42.Loc == scm.LocReg && d44.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d44)
			r24 := ctx.AllocReg()
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d43)
			if d44.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r24, uint64(d44.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r24, d44.Reg)
				ctx.EmitShlRegImm8(r24, 3)
			}
			if d43.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
				ctx.EmitAddInt64(r24, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r24, d43.Reg)
			}
			r25 := ctx.AllocRegExcept(r24)
			ctx.EmitMovRegMem(r25, r24, 0)
			ctx.FreeReg(r24)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			ctx.BindReg(r25, &d45)
			ctx.FreeDesc(&d44)
			ctx.EnsureDesc(&d42)
			var d46 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() % 64)}
			} else {
				r26 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r26, d42.Reg)
				ctx.EmitAndRegImm32(r26, 63)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d46)
			}
			if d46.Loc == scm.LocReg && d42.Loc == scm.LocReg && d46.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d45.Imm.Int()) << uint64(d46.Imm.Int())))}
			} else if d46.Loc == scm.LocImm {
				r27 := ctx.AllocRegExcept(d45.Reg)
				ctx.EmitMovRegReg(r27, d45.Reg)
				ctx.EmitShlRegImm8(r27, uint8(d46.Imm.Int()))
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d47)
			} else {
				{
					shiftSrc := d45.Reg
					r28 := ctx.AllocRegExcept(d45.Reg)
					ctx.EmitMovRegReg(r28, d45.Reg)
					shiftSrc = r28
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d46.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d46.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d46.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d47)
				}
			}
			if d47.Loc == scm.LocReg && d45.Loc == scm.LocReg && d47.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d45)
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&d42)
			var d48 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() % 64)}
			} else {
				r29 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r29, d42.Reg)
				ctx.EmitAndRegImm32(r29, 63)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d48)
			}
			if d48.Loc == scm.LocReg && d42.Loc == scm.LocReg && d48.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d49 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d40.Imm.Int()))))}
			} else {
				r30 := ctx.AllocReg()
				ctx.EmitMovRegReg(r30, d40.Reg)
				ctx.EmitShlRegImm8(r30, 56)
				ctx.EmitShrRegImm8(r30, 56)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d49)
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() + d49.Imm.Int())}
			} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
				r31 := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(r31, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d50)
			} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
				ctx.BindReg(d49.Reg, &d50)
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d48.Imm.Int()))
				ctx.EmitAddInt64(scratch, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(scratch, d48.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d49.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else {
				r32 := ctx.AllocRegExcept(d48.Reg, d49.Reg)
				ctx.EmitMovRegReg(r32, d48.Reg)
				ctx.EmitAddInt64(r32, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d50)
			}
			if d50.Loc == scm.LocReg && d48.Loc == scm.LocReg && d50.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d50)
			var d51 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d50.Imm.Int()) > uint64(64))}
			} else {
				r33 := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitCmpRegImm32(d50.Reg, 64)
				ctx.EmitSetcc(r33, scm.CcA)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r33}
				ctx.BindReg(r33, &d51)
			}
			ctx.FreeDesc(&d50)
			d52 = d51
			ctx.EnsureDesc(&d52)
			if d52.Loc != scm.LocImm && d52.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			lbl15 := ctx.ReserveLabel()
			if d52.Loc == scm.LocImm {
				if d52.Imm.Bool() {
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl12)
				} else {
					ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d47)
			if d47.Loc == scm.LocReg {
				ctx.ProtectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.ProtectReg(d47.Reg)
				ctx.ProtectReg(d47.Reg2)
			}
			d53 = d47
			if d53.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d53)
			ctx.EmitStoreToStack(d53, int32(bbs[2].PhiBase)+int32(0))
			if d47.Loc == scm.LocReg {
				ctx.UnprotectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d47.Reg)
				ctx.UnprotectReg(d47.Reg2)
			}
					ctx.EmitJmp(lbl13)
				}
			} else {
				ctx.EmitCmpRegImm32(d52.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl14)
				ctx.EmitJmp(lbl15)
				ctx.MarkLabel(lbl14)
				ctx.EmitJmp(lbl12)
				ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d47)
			if d47.Loc == scm.LocReg {
				ctx.ProtectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.ProtectReg(d47.Reg)
				ctx.ProtectReg(d47.Reg2)
			}
			d54 = d47
			if d54.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d54)
			ctx.EmitStoreToStack(d54, int32(bbs[2].PhiBase)+int32(0))
			if d47.Loc == scm.LocReg {
				ctx.UnprotectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d47.Reg)
				ctx.UnprotectReg(d47.Reg2)
			}
				ctx.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d51)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl13)
			ctx.ResolveFixups()
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d55 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d40.Imm.Int()))))}
			} else {
				r34 := ctx.AllocReg()
				ctx.EmitMovRegReg(r34, d40.Reg)
				ctx.EmitShlRegImm8(r34, 56)
				ctx.EmitShrRegImm8(r34, 56)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d55)
			}
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d55)
			var d57 scm.JITValueDesc
			if d56.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() - d55.Imm.Int())}
			} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d56.Reg)
				ctx.EmitMovRegReg(r35, d56.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d57)
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d55.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d56.Imm.Int()))
				ctx.EmitSubInt64(scratch, d55.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else if d55.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d56.Reg)
				ctx.EmitMovRegReg(scratch, d56.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d55.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else {
				r36 := ctx.AllocRegExcept(d56.Reg, d55.Reg)
				ctx.EmitMovRegReg(r36, d56.Reg)
				ctx.EmitSubInt64(r36, d55.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d57)
			}
			if d57.Loc == scm.LocReg && d56.Loc == scm.LocReg && d57.Reg == d56.Reg {
				ctx.TransferReg(d56.Reg)
				d56.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d57)
			var d58 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d38.Imm.Int()) >> uint64(d57.Imm.Int())))}
			} else if d57.Loc == scm.LocImm {
				r37 := ctx.AllocRegExcept(d38.Reg)
				ctx.EmitMovRegReg(r37, d38.Reg)
				ctx.EmitShrRegImm8(r37, uint8(d57.Imm.Int()))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d58)
			} else {
				{
					shiftSrc := d38.Reg
					r38 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegReg(r38, d38.Reg)
					shiftSrc = r38
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d57.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d57.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d57.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d58)
				}
			}
			if d58.Loc == scm.LocReg && d38.Loc == scm.LocReg && d58.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d57)
			r39 := ctx.AllocReg()
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d58)
			if d58.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r39, d58)
			}
			ctx.EmitJmp(lbl11)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl12)
			ctx.ResolveFixups()
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d42)
			var d59 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() / 64)}
			} else {
				r40 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r40, d42.Reg)
				ctx.EmitShrRegImm8(r40, 6)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d59)
			}
			if d59.Loc == scm.LocReg && d42.Loc == scm.LocReg && d59.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.EmitMovRegReg(scratch, d59.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			}
			if d60.Loc == scm.LocReg && d59.Loc == scm.LocReg && d60.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.EnsureDesc(&d60)
			r41 := ctx.AllocReg()
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d43)
			if d60.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r41, uint64(d60.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r41, d60.Reg)
				ctx.EmitShlRegImm8(r41, 3)
			}
			if d43.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
				ctx.EmitAddInt64(r41, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r41, d43.Reg)
			}
			r42 := ctx.AllocRegExcept(r41)
			ctx.EmitMovRegMem(r42, r41, 0)
			ctx.FreeReg(r41)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
			ctx.BindReg(r42, &d61)
			ctx.FreeDesc(&d60)
			ctx.EnsureDesc(&d42)
			var d62 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() % 64)}
			} else {
				r43 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r43, d42.Reg)
				ctx.EmitAndRegImm32(r43, 63)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d62)
			}
			if d62.Loc == scm.LocReg && d42.Loc == scm.LocReg && d62.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d42)
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d62)
			var d64 scm.JITValueDesc
			if d63.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d63.Imm.Int() - d62.Imm.Int())}
			} else if d62.Loc == scm.LocImm && d62.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d63.Reg)
				ctx.EmitMovRegReg(r44, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d64)
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d62.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
				ctx.EmitSubInt64(scratch, d62.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else if d62.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.EmitMovRegReg(scratch, d63.Reg)
				if d62.Imm.Int() >= -2147483648 && d62.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d62.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d62.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else {
				r45 := ctx.AllocRegExcept(d63.Reg, d62.Reg)
				ctx.EmitMovRegReg(r45, d63.Reg)
				ctx.EmitSubInt64(r45, d62.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d64)
			}
			if d64.Loc == scm.LocReg && d63.Loc == scm.LocReg && d64.Reg == d63.Reg {
				ctx.TransferReg(d63.Reg)
				d63.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d62)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d64)
			var d65 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d64.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d61.Imm.Int()) >> uint64(d64.Imm.Int())))}
			} else if d64.Loc == scm.LocImm {
				r46 := ctx.AllocRegExcept(d61.Reg)
				ctx.EmitMovRegReg(r46, d61.Reg)
				ctx.EmitShrRegImm8(r46, uint8(d64.Imm.Int()))
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d65)
			} else {
				{
					shiftSrc := d61.Reg
					r47 := ctx.AllocRegExcept(d61.Reg)
					ctx.EmitMovRegReg(r47, d61.Reg)
					shiftSrc = r47
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d64.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d64.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d64.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d65)
				}
			}
			if d65.Loc == scm.LocReg && d61.Loc == scm.LocReg && d65.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d61)
			ctx.FreeDesc(&d64)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d65)
			var d66 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() | d65.Imm.Int())}
			} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
				ctx.BindReg(d65.Reg, &d66)
			} else if d65.Loc == scm.LocImm && d65.Imm.Int() == 0 {
				r48 := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(r48, d47.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d66)
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d65.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
				ctx.EmitOrInt64(scratch, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d66)
			} else if d65.Loc == scm.LocImm {
				r49 := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(r49, d47.Reg)
				if d65.Imm.Int() >= -2147483648 && d65.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r49, int32(d65.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d65.Imm.Int()))
					ctx.EmitOrInt64(r49, scm.RegR11)
				}
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d66)
			} else {
				r50 := ctx.AllocRegExcept(d47.Reg, d65.Reg)
				ctx.EmitMovRegReg(r50, d47.Reg)
				ctx.EmitOrInt64(r50, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d66)
			}
			if d66.Loc == scm.LocReg && d47.Loc == scm.LocReg && d66.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d65)
			ctx.EnsureDesc(&d66)
			if d66.Loc == scm.LocReg {
				ctx.ProtectReg(d66.Reg)
			} else if d66.Loc == scm.LocRegPair {
				ctx.ProtectReg(d66.Reg)
				ctx.ProtectReg(d66.Reg2)
			}
			d67 = d66
			if d67.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d67)
			ctx.EmitStoreToStack(d67, int32(bbs[2].PhiBase)+int32(0))
			if d66.Loc == scm.LocReg {
				ctx.UnprotectReg(d66.Reg)
			} else if d66.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d66.Reg)
				ctx.UnprotectReg(d66.Reg2)
			}
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl11)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.BindReg(r39, &d68)
			ctx.BindReg(r39, &d68)
			if r12 { ctx.UnprotectReg(r13) }
			var d69 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				r51 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r51, fieldAddr)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d69)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r52 := ctx.AllocReg()
				ctx.EmitMovRegMem(r52, thisptr.Reg, off)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
				ctx.BindReg(r52, &d69)
			}
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d69)
			var d70 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d69.Imm.Int()))))}
			} else {
				r53 := ctx.AllocReg()
				ctx.EmitMovRegReg(r53, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d70)
			}
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d70)
			var d71 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d68.Imm.Int() + d70.Imm.Int())}
			} else if d70.Loc == scm.LocImm && d70.Imm.Int() == 0 {
				r54 := ctx.AllocRegExcept(d68.Reg)
				ctx.EmitMovRegReg(r54, d68.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d71)
			} else if d68.Loc == scm.LocImm && d68.Imm.Int() == 0 {
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d70.Reg}
				ctx.BindReg(d70.Reg, &d71)
			} else if d68.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d70.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
				ctx.EmitAddInt64(scratch, d70.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			} else if d70.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d68.Reg)
				ctx.EmitMovRegReg(scratch, d68.Reg)
				if d70.Imm.Int() >= -2147483648 && d70.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d70.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d70.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			} else {
				r55 := ctx.AllocRegExcept(d68.Reg, d70.Reg)
				ctx.EmitMovRegReg(r55, d68.Reg)
				ctx.EmitAddInt64(r55, d70.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d71)
			}
			if d71.Loc == scm.LocReg && d68.Loc == scm.LocReg && d71.Reg == d68.Reg {
				ctx.TransferReg(d68.Reg)
				d68.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d70)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d72 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r56 := ctx.AllocReg()
				ctx.EmitMovRegReg(r56, idxInt.Reg)
				ctx.EmitShlRegImm8(r56, 32)
				ctx.EmitShrRegImm8(r56, 32)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
				ctx.BindReg(r56, &d72)
			}
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d72)
			var d73 scm.JITValueDesc
			if d71.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d71.Imm.Int()) == uint64(d72.Imm.Int()))}
			} else if d72.Loc == scm.LocImm {
				r57 := ctx.AllocRegExcept(d71.Reg)
				if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d71.Reg, int32(d72.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
					ctx.EmitCmpInt64(d71.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r57, scm.CcE)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d73)
			} else if d71.Loc == scm.LocImm {
				r58 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d71.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d72.Reg)
				ctx.EmitSetcc(r58, scm.CcE)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d73)
			} else {
				r59 := ctx.AllocRegExcept(d71.Reg)
				ctx.EmitCmpInt64(d71.Reg, d72.Reg)
				ctx.EmitSetcc(r59, scm.CcE)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d73)
			}
			ctx.FreeDesc(&d72)
			d74 = d73
			ctx.EnsureDesc(&d74)
			if d74.Loc != scm.LocImm && d74.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d74.Loc == scm.LocImm {
				if d74.Imm.Bool() {
			ps75 := scm.PhiState{General: ps.General}
			ps75.OverlayValues = make([]scm.JITValueDesc, 75)
			ps75.OverlayValues[0] = d0
			ps75.OverlayValues[1] = d1
			ps75.OverlayValues[2] = d2
			ps75.OverlayValues[3] = d3
			ps75.OverlayValues[4] = d4
			ps75.OverlayValues[5] = d5
			ps75.OverlayValues[7] = d7
			ps75.OverlayValues[8] = d8
			ps75.OverlayValues[9] = d9
			ps75.OverlayValues[10] = d10
			ps75.OverlayValues[11] = d11
			ps75.OverlayValues[12] = d12
			ps75.OverlayValues[15] = d15
			ps75.OverlayValues[16] = d16
			ps75.OverlayValues[34] = d34
			ps75.OverlayValues[35] = d35
			ps75.OverlayValues[36] = d36
			ps75.OverlayValues[37] = d37
			ps75.OverlayValues[38] = d38
			ps75.OverlayValues[39] = d39
			ps75.OverlayValues[40] = d40
			ps75.OverlayValues[41] = d41
			ps75.OverlayValues[42] = d42
			ps75.OverlayValues[43] = d43
			ps75.OverlayValues[44] = d44
			ps75.OverlayValues[45] = d45
			ps75.OverlayValues[46] = d46
			ps75.OverlayValues[47] = d47
			ps75.OverlayValues[48] = d48
			ps75.OverlayValues[49] = d49
			ps75.OverlayValues[50] = d50
			ps75.OverlayValues[51] = d51
			ps75.OverlayValues[52] = d52
			ps75.OverlayValues[53] = d53
			ps75.OverlayValues[54] = d54
			ps75.OverlayValues[55] = d55
			ps75.OverlayValues[56] = d56
			ps75.OverlayValues[57] = d57
			ps75.OverlayValues[58] = d58
			ps75.OverlayValues[59] = d59
			ps75.OverlayValues[60] = d60
			ps75.OverlayValues[61] = d61
			ps75.OverlayValues[62] = d62
			ps75.OverlayValues[63] = d63
			ps75.OverlayValues[64] = d64
			ps75.OverlayValues[65] = d65
			ps75.OverlayValues[66] = d66
			ps75.OverlayValues[67] = d67
			ps75.OverlayValues[68] = d68
			ps75.OverlayValues[69] = d69
			ps75.OverlayValues[70] = d70
			ps75.OverlayValues[71] = d71
			ps75.OverlayValues[72] = d72
			ps75.OverlayValues[73] = d73
			ps75.OverlayValues[74] = d74
					return bbs[4].RenderPS(ps75)
				}
			ps76 := scm.PhiState{General: ps.General}
			ps76.OverlayValues = make([]scm.JITValueDesc, 75)
			ps76.OverlayValues[0] = d0
			ps76.OverlayValues[1] = d1
			ps76.OverlayValues[2] = d2
			ps76.OverlayValues[3] = d3
			ps76.OverlayValues[4] = d4
			ps76.OverlayValues[5] = d5
			ps76.OverlayValues[7] = d7
			ps76.OverlayValues[8] = d8
			ps76.OverlayValues[9] = d9
			ps76.OverlayValues[10] = d10
			ps76.OverlayValues[11] = d11
			ps76.OverlayValues[12] = d12
			ps76.OverlayValues[15] = d15
			ps76.OverlayValues[16] = d16
			ps76.OverlayValues[34] = d34
			ps76.OverlayValues[35] = d35
			ps76.OverlayValues[36] = d36
			ps76.OverlayValues[37] = d37
			ps76.OverlayValues[38] = d38
			ps76.OverlayValues[39] = d39
			ps76.OverlayValues[40] = d40
			ps76.OverlayValues[41] = d41
			ps76.OverlayValues[42] = d42
			ps76.OverlayValues[43] = d43
			ps76.OverlayValues[44] = d44
			ps76.OverlayValues[45] = d45
			ps76.OverlayValues[46] = d46
			ps76.OverlayValues[47] = d47
			ps76.OverlayValues[48] = d48
			ps76.OverlayValues[49] = d49
			ps76.OverlayValues[50] = d50
			ps76.OverlayValues[51] = d51
			ps76.OverlayValues[52] = d52
			ps76.OverlayValues[53] = d53
			ps76.OverlayValues[54] = d54
			ps76.OverlayValues[55] = d55
			ps76.OverlayValues[56] = d56
			ps76.OverlayValues[57] = d57
			ps76.OverlayValues[58] = d58
			ps76.OverlayValues[59] = d59
			ps76.OverlayValues[60] = d60
			ps76.OverlayValues[61] = d61
			ps76.OverlayValues[62] = d62
			ps76.OverlayValues[63] = d63
			ps76.OverlayValues[64] = d64
			ps76.OverlayValues[65] = d65
			ps76.OverlayValues[66] = d66
			ps76.OverlayValues[67] = d67
			ps76.OverlayValues[68] = d68
			ps76.OverlayValues[69] = d69
			ps76.OverlayValues[70] = d70
			ps76.OverlayValues[71] = d71
			ps76.OverlayValues[72] = d72
			ps76.OverlayValues[73] = d73
			ps76.OverlayValues[74] = d74
				return bbs[5].RenderPS(ps76)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d74.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl16)
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl6)
			ps77 := scm.PhiState{General: true}
			ps77.OverlayValues = make([]scm.JITValueDesc, 75)
			ps77.OverlayValues[0] = d0
			ps77.OverlayValues[1] = d1
			ps77.OverlayValues[2] = d2
			ps77.OverlayValues[3] = d3
			ps77.OverlayValues[4] = d4
			ps77.OverlayValues[5] = d5
			ps77.OverlayValues[7] = d7
			ps77.OverlayValues[8] = d8
			ps77.OverlayValues[9] = d9
			ps77.OverlayValues[10] = d10
			ps77.OverlayValues[11] = d11
			ps77.OverlayValues[12] = d12
			ps77.OverlayValues[15] = d15
			ps77.OverlayValues[16] = d16
			ps77.OverlayValues[34] = d34
			ps77.OverlayValues[35] = d35
			ps77.OverlayValues[36] = d36
			ps77.OverlayValues[37] = d37
			ps77.OverlayValues[38] = d38
			ps77.OverlayValues[39] = d39
			ps77.OverlayValues[40] = d40
			ps77.OverlayValues[41] = d41
			ps77.OverlayValues[42] = d42
			ps77.OverlayValues[43] = d43
			ps77.OverlayValues[44] = d44
			ps77.OverlayValues[45] = d45
			ps77.OverlayValues[46] = d46
			ps77.OverlayValues[47] = d47
			ps77.OverlayValues[48] = d48
			ps77.OverlayValues[49] = d49
			ps77.OverlayValues[50] = d50
			ps77.OverlayValues[51] = d51
			ps77.OverlayValues[52] = d52
			ps77.OverlayValues[53] = d53
			ps77.OverlayValues[54] = d54
			ps77.OverlayValues[55] = d55
			ps77.OverlayValues[56] = d56
			ps77.OverlayValues[57] = d57
			ps77.OverlayValues[58] = d58
			ps77.OverlayValues[59] = d59
			ps77.OverlayValues[60] = d60
			ps77.OverlayValues[61] = d61
			ps77.OverlayValues[62] = d62
			ps77.OverlayValues[63] = d63
			ps77.OverlayValues[64] = d64
			ps77.OverlayValues[65] = d65
			ps77.OverlayValues[66] = d66
			ps77.OverlayValues[67] = d67
			ps77.OverlayValues[68] = d68
			ps77.OverlayValues[69] = d69
			ps77.OverlayValues[70] = d70
			ps77.OverlayValues[71] = d71
			ps77.OverlayValues[72] = d72
			ps77.OverlayValues[73] = d73
			ps77.OverlayValues[74] = d74
			ps78 := scm.PhiState{General: true}
			ps78.OverlayValues = make([]scm.JITValueDesc, 75)
			ps78.OverlayValues[0] = d0
			ps78.OverlayValues[1] = d1
			ps78.OverlayValues[2] = d2
			ps78.OverlayValues[3] = d3
			ps78.OverlayValues[4] = d4
			ps78.OverlayValues[5] = d5
			ps78.OverlayValues[7] = d7
			ps78.OverlayValues[8] = d8
			ps78.OverlayValues[9] = d9
			ps78.OverlayValues[10] = d10
			ps78.OverlayValues[11] = d11
			ps78.OverlayValues[12] = d12
			ps78.OverlayValues[15] = d15
			ps78.OverlayValues[16] = d16
			ps78.OverlayValues[34] = d34
			ps78.OverlayValues[35] = d35
			ps78.OverlayValues[36] = d36
			ps78.OverlayValues[37] = d37
			ps78.OverlayValues[38] = d38
			ps78.OverlayValues[39] = d39
			ps78.OverlayValues[40] = d40
			ps78.OverlayValues[41] = d41
			ps78.OverlayValues[42] = d42
			ps78.OverlayValues[43] = d43
			ps78.OverlayValues[44] = d44
			ps78.OverlayValues[45] = d45
			ps78.OverlayValues[46] = d46
			ps78.OverlayValues[47] = d47
			ps78.OverlayValues[48] = d48
			ps78.OverlayValues[49] = d49
			ps78.OverlayValues[50] = d50
			ps78.OverlayValues[51] = d51
			ps78.OverlayValues[52] = d52
			ps78.OverlayValues[53] = d53
			ps78.OverlayValues[54] = d54
			ps78.OverlayValues[55] = d55
			ps78.OverlayValues[56] = d56
			ps78.OverlayValues[57] = d57
			ps78.OverlayValues[58] = d58
			ps78.OverlayValues[59] = d59
			ps78.OverlayValues[60] = d60
			ps78.OverlayValues[61] = d61
			ps78.OverlayValues[62] = d62
			ps78.OverlayValues[63] = d63
			ps78.OverlayValues[64] = d64
			ps78.OverlayValues[65] = d65
			ps78.OverlayValues[66] = d66
			ps78.OverlayValues[67] = d67
			ps78.OverlayValues[68] = d68
			ps78.OverlayValues[69] = d69
			ps78.OverlayValues[70] = d70
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
			snap85 := d7
			snap86 := d8
			snap87 := d9
			snap88 := d10
			snap89 := d11
			snap90 := d12
			snap91 := d15
			snap92 := d16
			snap93 := d34
			snap94 := d35
			snap95 := d36
			snap96 := d37
			snap97 := d38
			snap98 := d39
			snap99 := d40
			snap100 := d41
			snap101 := d42
			snap102 := d43
			snap103 := d44
			snap104 := d45
			snap105 := d46
			snap106 := d47
			snap107 := d48
			snap108 := d49
			snap109 := d50
			snap110 := d51
			snap111 := d52
			snap112 := d53
			snap113 := d54
			snap114 := d55
			snap115 := d56
			snap116 := d57
			snap117 := d58
			snap118 := d59
			snap119 := d60
			snap120 := d61
			snap121 := d62
			snap122 := d63
			snap123 := d64
			snap124 := d65
			snap125 := d66
			snap126 := d67
			snap127 := d68
			snap128 := d69
			snap129 := d70
			snap130 := d71
			snap131 := d72
			snap132 := d73
			snap133 := d74
			alloc134 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps78)
			}
			ctx.RestoreAllocState(alloc134)
			d0 = snap79
			d1 = snap80
			d2 = snap81
			d3 = snap82
			d4 = snap83
			d5 = snap84
			d7 = snap85
			d8 = snap86
			d9 = snap87
			d10 = snap88
			d11 = snap89
			d12 = snap90
			d15 = snap91
			d16 = snap92
			d34 = snap93
			d35 = snap94
			d36 = snap95
			d37 = snap96
			d38 = snap97
			d39 = snap98
			d40 = snap99
			d41 = snap100
			d42 = snap101
			d43 = snap102
			d44 = snap103
			d45 = snap104
			d46 = snap105
			d47 = snap106
			d48 = snap107
			d49 = snap108
			d50 = snap109
			d51 = snap110
			d52 = snap111
			d53 = snap112
			d54 = snap113
			d55 = snap114
			d56 = snap115
			d57 = snap116
			d58 = snap117
			d59 = snap118
			d60 = snap119
			d61 = snap120
			d62 = snap121
			d63 = snap122
			d64 = snap123
			d65 = snap124
			d66 = snap125
			d67 = snap126
			d68 = snap127
			d69 = snap128
			d70 = snap129
			d71 = snap130
			d72 = snap131
			d73 = snap132
			d74 = snap133
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps77)
			}
			return result
			ctx.FreeDesc(&d73)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			ctx.ReclaimUntrackedRegs()
			var d135 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r60 := ctx.AllocReg()
				r61 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r60, fieldAddr)
				ctx.EmitMovRegMem64(r61, fieldAddr+8)
				d135 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r60, Reg2: r61}
				ctx.BindReg(r60, &d135)
				ctx.BindReg(r61, &d135)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r62 := ctx.AllocReg()
				r63 := ctx.AllocReg()
				ctx.EmitMovRegMem(r62, thisptr.Reg, off)
				ctx.EmitMovRegMem(r63, thisptr.Reg, off+8)
				d135 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r62, Reg2: r63}
				ctx.BindReg(r62, &d135)
				ctx.BindReg(r63, &d135)
			}
			ctx.EnsureDesc(&d36)
			r64 := ctx.AllocReg()
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d135)
			if d36.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r64, uint64(d36.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r64, d36.Reg)
				ctx.EmitShlRegImm8(r64, 4)
			}
			if d135.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
				ctx.EmitAddInt64(r64, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r64, d135.Reg)
			}
			r65 := ctx.AllocRegExcept(r64)
			r66 := ctx.AllocRegExcept(r64, r65)
			ctx.EmitMovRegMem(r65, r64, 0)
			ctx.EmitMovRegMem(r66, r64, 8)
			ctx.FreeReg(r64)
			d136 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r65, Reg2: r66}
			ctx.BindReg(r65, &d136)
			ctx.BindReg(r66, &d136)
			d137 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d137)
			ctx.BindReg(r2, &d137)
			ctx.EnsureDesc(&d136)
			if d136.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d136, &d137)
			} else {
				switch d136.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(d137, d136)
				case scm.TagInt:
					ctx.EmitMakeInt(d137, d136)
				case scm.TagFloat:
					ctx.EmitMakeFloat(d137, d136)
				case scm.TagNil:
					ctx.EmitMakeNil(d137)
				default:
					ctx.EmitMovPairToResult(&d136, &d137)
				}
			}
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[5].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d138 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r67 := ctx.AllocReg()
				ctx.EmitMovRegReg(r67, idxInt.Reg)
				ctx.EmitShlRegImm8(r67, 32)
				ctx.EmitShrRegImm8(r67, 32)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d138)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d138)
			var d139 scm.JITValueDesc
			if d71.Loc == scm.LocImm && d138.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d71.Imm.Int()) < uint64(d138.Imm.Int()))}
			} else if d138.Loc == scm.LocImm {
				r68 := ctx.AllocRegExcept(d71.Reg)
				if d138.Imm.Int() >= -2147483648 && d138.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d71.Reg, int32(d138.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
					ctx.EmitCmpInt64(d71.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r68, scm.CcB)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r68}
				ctx.BindReg(r68, &d139)
			} else if d71.Loc == scm.LocImm {
				r69 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d71.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d138.Reg)
				ctx.EmitSetcc(r69, scm.CcB)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r69}
				ctx.BindReg(r69, &d139)
			} else {
				r70 := ctx.AllocRegExcept(d71.Reg)
				ctx.EmitCmpInt64(d71.Reg, d138.Reg)
				ctx.EmitSetcc(r70, scm.CcB)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r70}
				ctx.BindReg(r70, &d139)
			}
			ctx.FreeDesc(&d71)
			ctx.FreeDesc(&d138)
			d140 = d139
			ctx.EnsureDesc(&d140)
			if d140.Loc != scm.LocImm && d140.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d140.Loc == scm.LocImm {
				if d140.Imm.Bool() {
			ps141 := scm.PhiState{General: ps.General}
			ps141.OverlayValues = make([]scm.JITValueDesc, 141)
			ps141.OverlayValues[0] = d0
			ps141.OverlayValues[1] = d1
			ps141.OverlayValues[2] = d2
			ps141.OverlayValues[3] = d3
			ps141.OverlayValues[4] = d4
			ps141.OverlayValues[5] = d5
			ps141.OverlayValues[7] = d7
			ps141.OverlayValues[8] = d8
			ps141.OverlayValues[9] = d9
			ps141.OverlayValues[10] = d10
			ps141.OverlayValues[11] = d11
			ps141.OverlayValues[12] = d12
			ps141.OverlayValues[15] = d15
			ps141.OverlayValues[16] = d16
			ps141.OverlayValues[34] = d34
			ps141.OverlayValues[35] = d35
			ps141.OverlayValues[36] = d36
			ps141.OverlayValues[37] = d37
			ps141.OverlayValues[38] = d38
			ps141.OverlayValues[39] = d39
			ps141.OverlayValues[40] = d40
			ps141.OverlayValues[41] = d41
			ps141.OverlayValues[42] = d42
			ps141.OverlayValues[43] = d43
			ps141.OverlayValues[44] = d44
			ps141.OverlayValues[45] = d45
			ps141.OverlayValues[46] = d46
			ps141.OverlayValues[47] = d47
			ps141.OverlayValues[48] = d48
			ps141.OverlayValues[49] = d49
			ps141.OverlayValues[50] = d50
			ps141.OverlayValues[51] = d51
			ps141.OverlayValues[52] = d52
			ps141.OverlayValues[53] = d53
			ps141.OverlayValues[54] = d54
			ps141.OverlayValues[55] = d55
			ps141.OverlayValues[56] = d56
			ps141.OverlayValues[57] = d57
			ps141.OverlayValues[58] = d58
			ps141.OverlayValues[59] = d59
			ps141.OverlayValues[60] = d60
			ps141.OverlayValues[61] = d61
			ps141.OverlayValues[62] = d62
			ps141.OverlayValues[63] = d63
			ps141.OverlayValues[64] = d64
			ps141.OverlayValues[65] = d65
			ps141.OverlayValues[66] = d66
			ps141.OverlayValues[67] = d67
			ps141.OverlayValues[68] = d68
			ps141.OverlayValues[69] = d69
			ps141.OverlayValues[70] = d70
			ps141.OverlayValues[71] = d71
			ps141.OverlayValues[72] = d72
			ps141.OverlayValues[73] = d73
			ps141.OverlayValues[74] = d74
			ps141.OverlayValues[135] = d135
			ps141.OverlayValues[136] = d136
			ps141.OverlayValues[137] = d137
			ps141.OverlayValues[138] = d138
			ps141.OverlayValues[139] = d139
			ps141.OverlayValues[140] = d140
					return bbs[6].RenderPS(ps141)
				}
			ps142 := scm.PhiState{General: ps.General}
			ps142.OverlayValues = make([]scm.JITValueDesc, 141)
			ps142.OverlayValues[0] = d0
			ps142.OverlayValues[1] = d1
			ps142.OverlayValues[2] = d2
			ps142.OverlayValues[3] = d3
			ps142.OverlayValues[4] = d4
			ps142.OverlayValues[5] = d5
			ps142.OverlayValues[7] = d7
			ps142.OverlayValues[8] = d8
			ps142.OverlayValues[9] = d9
			ps142.OverlayValues[10] = d10
			ps142.OverlayValues[11] = d11
			ps142.OverlayValues[12] = d12
			ps142.OverlayValues[15] = d15
			ps142.OverlayValues[16] = d16
			ps142.OverlayValues[34] = d34
			ps142.OverlayValues[35] = d35
			ps142.OverlayValues[36] = d36
			ps142.OverlayValues[37] = d37
			ps142.OverlayValues[38] = d38
			ps142.OverlayValues[39] = d39
			ps142.OverlayValues[40] = d40
			ps142.OverlayValues[41] = d41
			ps142.OverlayValues[42] = d42
			ps142.OverlayValues[43] = d43
			ps142.OverlayValues[44] = d44
			ps142.OverlayValues[45] = d45
			ps142.OverlayValues[46] = d46
			ps142.OverlayValues[47] = d47
			ps142.OverlayValues[48] = d48
			ps142.OverlayValues[49] = d49
			ps142.OverlayValues[50] = d50
			ps142.OverlayValues[51] = d51
			ps142.OverlayValues[52] = d52
			ps142.OverlayValues[53] = d53
			ps142.OverlayValues[54] = d54
			ps142.OverlayValues[55] = d55
			ps142.OverlayValues[56] = d56
			ps142.OverlayValues[57] = d57
			ps142.OverlayValues[58] = d58
			ps142.OverlayValues[59] = d59
			ps142.OverlayValues[60] = d60
			ps142.OverlayValues[61] = d61
			ps142.OverlayValues[62] = d62
			ps142.OverlayValues[63] = d63
			ps142.OverlayValues[64] = d64
			ps142.OverlayValues[65] = d65
			ps142.OverlayValues[66] = d66
			ps142.OverlayValues[67] = d67
			ps142.OverlayValues[68] = d68
			ps142.OverlayValues[69] = d69
			ps142.OverlayValues[70] = d70
			ps142.OverlayValues[71] = d71
			ps142.OverlayValues[72] = d72
			ps142.OverlayValues[73] = d73
			ps142.OverlayValues[74] = d74
			ps142.OverlayValues[135] = d135
			ps142.OverlayValues[136] = d136
			ps142.OverlayValues[137] = d137
			ps142.OverlayValues[138] = d138
			ps142.OverlayValues[139] = d139
			ps142.OverlayValues[140] = d140
				return bbs[7].RenderPS(ps142)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl18 := ctx.ReserveLabel()
			lbl19 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d140.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl18)
			ctx.EmitJmp(lbl19)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl8)
			ps143 := scm.PhiState{General: true}
			ps143.OverlayValues = make([]scm.JITValueDesc, 141)
			ps143.OverlayValues[0] = d0
			ps143.OverlayValues[1] = d1
			ps143.OverlayValues[2] = d2
			ps143.OverlayValues[3] = d3
			ps143.OverlayValues[4] = d4
			ps143.OverlayValues[5] = d5
			ps143.OverlayValues[7] = d7
			ps143.OverlayValues[8] = d8
			ps143.OverlayValues[9] = d9
			ps143.OverlayValues[10] = d10
			ps143.OverlayValues[11] = d11
			ps143.OverlayValues[12] = d12
			ps143.OverlayValues[15] = d15
			ps143.OverlayValues[16] = d16
			ps143.OverlayValues[34] = d34
			ps143.OverlayValues[35] = d35
			ps143.OverlayValues[36] = d36
			ps143.OverlayValues[37] = d37
			ps143.OverlayValues[38] = d38
			ps143.OverlayValues[39] = d39
			ps143.OverlayValues[40] = d40
			ps143.OverlayValues[41] = d41
			ps143.OverlayValues[42] = d42
			ps143.OverlayValues[43] = d43
			ps143.OverlayValues[44] = d44
			ps143.OverlayValues[45] = d45
			ps143.OverlayValues[46] = d46
			ps143.OverlayValues[47] = d47
			ps143.OverlayValues[48] = d48
			ps143.OverlayValues[49] = d49
			ps143.OverlayValues[50] = d50
			ps143.OverlayValues[51] = d51
			ps143.OverlayValues[52] = d52
			ps143.OverlayValues[53] = d53
			ps143.OverlayValues[54] = d54
			ps143.OverlayValues[55] = d55
			ps143.OverlayValues[56] = d56
			ps143.OverlayValues[57] = d57
			ps143.OverlayValues[58] = d58
			ps143.OverlayValues[59] = d59
			ps143.OverlayValues[60] = d60
			ps143.OverlayValues[61] = d61
			ps143.OverlayValues[62] = d62
			ps143.OverlayValues[63] = d63
			ps143.OverlayValues[64] = d64
			ps143.OverlayValues[65] = d65
			ps143.OverlayValues[66] = d66
			ps143.OverlayValues[67] = d67
			ps143.OverlayValues[68] = d68
			ps143.OverlayValues[69] = d69
			ps143.OverlayValues[70] = d70
			ps143.OverlayValues[71] = d71
			ps143.OverlayValues[72] = d72
			ps143.OverlayValues[73] = d73
			ps143.OverlayValues[74] = d74
			ps143.OverlayValues[135] = d135
			ps143.OverlayValues[136] = d136
			ps143.OverlayValues[137] = d137
			ps143.OverlayValues[138] = d138
			ps143.OverlayValues[139] = d139
			ps143.OverlayValues[140] = d140
			ps144 := scm.PhiState{General: true}
			ps144.OverlayValues = make([]scm.JITValueDesc, 141)
			ps144.OverlayValues[0] = d0
			ps144.OverlayValues[1] = d1
			ps144.OverlayValues[2] = d2
			ps144.OverlayValues[3] = d3
			ps144.OverlayValues[4] = d4
			ps144.OverlayValues[5] = d5
			ps144.OverlayValues[7] = d7
			ps144.OverlayValues[8] = d8
			ps144.OverlayValues[9] = d9
			ps144.OverlayValues[10] = d10
			ps144.OverlayValues[11] = d11
			ps144.OverlayValues[12] = d12
			ps144.OverlayValues[15] = d15
			ps144.OverlayValues[16] = d16
			ps144.OverlayValues[34] = d34
			ps144.OverlayValues[35] = d35
			ps144.OverlayValues[36] = d36
			ps144.OverlayValues[37] = d37
			ps144.OverlayValues[38] = d38
			ps144.OverlayValues[39] = d39
			ps144.OverlayValues[40] = d40
			ps144.OverlayValues[41] = d41
			ps144.OverlayValues[42] = d42
			ps144.OverlayValues[43] = d43
			ps144.OverlayValues[44] = d44
			ps144.OverlayValues[45] = d45
			ps144.OverlayValues[46] = d46
			ps144.OverlayValues[47] = d47
			ps144.OverlayValues[48] = d48
			ps144.OverlayValues[49] = d49
			ps144.OverlayValues[50] = d50
			ps144.OverlayValues[51] = d51
			ps144.OverlayValues[52] = d52
			ps144.OverlayValues[53] = d53
			ps144.OverlayValues[54] = d54
			ps144.OverlayValues[55] = d55
			ps144.OverlayValues[56] = d56
			ps144.OverlayValues[57] = d57
			ps144.OverlayValues[58] = d58
			ps144.OverlayValues[59] = d59
			ps144.OverlayValues[60] = d60
			ps144.OverlayValues[61] = d61
			ps144.OverlayValues[62] = d62
			ps144.OverlayValues[63] = d63
			ps144.OverlayValues[64] = d64
			ps144.OverlayValues[65] = d65
			ps144.OverlayValues[66] = d66
			ps144.OverlayValues[67] = d67
			ps144.OverlayValues[68] = d68
			ps144.OverlayValues[69] = d69
			ps144.OverlayValues[70] = d70
			ps144.OverlayValues[71] = d71
			ps144.OverlayValues[72] = d72
			ps144.OverlayValues[73] = d73
			ps144.OverlayValues[74] = d74
			ps144.OverlayValues[135] = d135
			ps144.OverlayValues[136] = d136
			ps144.OverlayValues[137] = d137
			ps144.OverlayValues[138] = d138
			ps144.OverlayValues[139] = d139
			ps144.OverlayValues[140] = d140
			snap145 := d0
			snap146 := d1
			snap147 := d2
			snap148 := d3
			snap149 := d4
			snap150 := d5
			snap151 := d7
			snap152 := d8
			snap153 := d9
			snap154 := d10
			snap155 := d11
			snap156 := d12
			snap157 := d15
			snap158 := d16
			snap159 := d34
			snap160 := d35
			snap161 := d36
			snap162 := d37
			snap163 := d38
			snap164 := d39
			snap165 := d40
			snap166 := d41
			snap167 := d42
			snap168 := d43
			snap169 := d44
			snap170 := d45
			snap171 := d46
			snap172 := d47
			snap173 := d48
			snap174 := d49
			snap175 := d50
			snap176 := d51
			snap177 := d52
			snap178 := d53
			snap179 := d54
			snap180 := d55
			snap181 := d56
			snap182 := d57
			snap183 := d58
			snap184 := d59
			snap185 := d60
			snap186 := d61
			snap187 := d62
			snap188 := d63
			snap189 := d64
			snap190 := d65
			snap191 := d66
			snap192 := d67
			snap193 := d68
			snap194 := d69
			snap195 := d70
			snap196 := d71
			snap197 := d72
			snap198 := d73
			snap199 := d74
			snap200 := d135
			snap201 := d136
			snap202 := d137
			snap203 := d138
			snap204 := d139
			snap205 := d140
			alloc206 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps144)
			}
			ctx.RestoreAllocState(alloc206)
			d0 = snap145
			d1 = snap146
			d2 = snap147
			d3 = snap148
			d4 = snap149
			d5 = snap150
			d7 = snap151
			d8 = snap152
			d9 = snap153
			d10 = snap154
			d11 = snap155
			d12 = snap156
			d15 = snap157
			d16 = snap158
			d34 = snap159
			d35 = snap160
			d36 = snap161
			d37 = snap162
			d38 = snap163
			d39 = snap164
			d40 = snap165
			d41 = snap166
			d42 = snap167
			d43 = snap168
			d44 = snap169
			d45 = snap170
			d46 = snap171
			d47 = snap172
			d48 = snap173
			d49 = snap174
			d50 = snap175
			d51 = snap176
			d52 = snap177
			d53 = snap178
			d54 = snap179
			d55 = snap180
			d56 = snap181
			d57 = snap182
			d58 = snap183
			d59 = snap184
			d60 = snap185
			d61 = snap186
			d62 = snap187
			d63 = snap188
			d64 = snap189
			d65 = snap190
			d66 = snap191
			d67 = snap192
			d68 = snap193
			d69 = snap194
			d70 = snap195
			d71 = snap196
			d72 = snap197
			d73 = snap198
			d74 = snap199
			d135 = snap200
			d136 = snap201
			d137 = snap202
			d138 = snap203
			d139 = snap204
			d140 = snap205
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps143)
			}
			return result
			ctx.FreeDesc(&d139)
			return result
			}
			bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d36)
			var d207 scm.JITValueDesc
			if d36.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.EmitMovRegReg(scratch, d36.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			}
			if d207.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: d207.Type, Imm: scm.NewInt(int64(uint64(d207.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d207.Reg, 32)
				ctx.EmitShrRegImm8(d207.Reg, 32)
			}
			if d207.Loc == scm.LocReg && d36.Loc == scm.LocReg && d207.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d1)
			if d1.Loc == scm.LocReg {
				ctx.ProtectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1.Reg)
				ctx.ProtectReg(d1.Reg2)
			}
			ctx.EnsureDesc(&d207)
			if d207.Loc == scm.LocReg {
				ctx.ProtectReg(d207.Reg)
			} else if d207.Loc == scm.LocRegPair {
				ctx.ProtectReg(d207.Reg)
				ctx.ProtectReg(d207.Reg2)
			}
			d208 = d207
			if d208.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d208)
			d209 = d208
			if d209.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: d209.Type, Imm: scm.NewInt(int64(uint64(d209.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d209.Reg, 32)
				ctx.EmitShrRegImm8(d209.Reg, 32)
			}
			ctx.EmitStoreToStack(d209, int32(bbs[1].PhiBase)+int32(0))
			d210 = d1
			if d210.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			d211 = d210
			if d211.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: d211.Type, Imm: scm.NewInt(int64(uint64(d211.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d211.Reg, 32)
				ctx.EmitShrRegImm8(d211.Reg, 32)
			}
			ctx.EmitStoreToStack(d211, int32(bbs[1].PhiBase)+int32(16))
			if d1.Loc == scm.LocReg {
				ctx.UnprotectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1.Reg)
				ctx.UnprotectReg(d1.Reg2)
			}
			if d207.Loc == scm.LocReg {
				ctx.UnprotectReg(d207.Reg)
			} else if d207.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d207.Reg)
				ctx.UnprotectReg(d207.Reg2)
			}
			ps212 := scm.PhiState{General: ps.General}
			ps212.OverlayValues = make([]scm.JITValueDesc, 212)
			ps212.OverlayValues[0] = d0
			ps212.OverlayValues[1] = d1
			ps212.OverlayValues[2] = d2
			ps212.OverlayValues[3] = d3
			ps212.OverlayValues[4] = d4
			ps212.OverlayValues[5] = d5
			ps212.OverlayValues[7] = d7
			ps212.OverlayValues[8] = d8
			ps212.OverlayValues[9] = d9
			ps212.OverlayValues[10] = d10
			ps212.OverlayValues[11] = d11
			ps212.OverlayValues[12] = d12
			ps212.OverlayValues[15] = d15
			ps212.OverlayValues[16] = d16
			ps212.OverlayValues[34] = d34
			ps212.OverlayValues[35] = d35
			ps212.OverlayValues[36] = d36
			ps212.OverlayValues[37] = d37
			ps212.OverlayValues[38] = d38
			ps212.OverlayValues[39] = d39
			ps212.OverlayValues[40] = d40
			ps212.OverlayValues[41] = d41
			ps212.OverlayValues[42] = d42
			ps212.OverlayValues[43] = d43
			ps212.OverlayValues[44] = d44
			ps212.OverlayValues[45] = d45
			ps212.OverlayValues[46] = d46
			ps212.OverlayValues[47] = d47
			ps212.OverlayValues[48] = d48
			ps212.OverlayValues[49] = d49
			ps212.OverlayValues[50] = d50
			ps212.OverlayValues[51] = d51
			ps212.OverlayValues[52] = d52
			ps212.OverlayValues[53] = d53
			ps212.OverlayValues[54] = d54
			ps212.OverlayValues[55] = d55
			ps212.OverlayValues[56] = d56
			ps212.OverlayValues[57] = d57
			ps212.OverlayValues[58] = d58
			ps212.OverlayValues[59] = d59
			ps212.OverlayValues[60] = d60
			ps212.OverlayValues[61] = d61
			ps212.OverlayValues[62] = d62
			ps212.OverlayValues[63] = d63
			ps212.OverlayValues[64] = d64
			ps212.OverlayValues[65] = d65
			ps212.OverlayValues[66] = d66
			ps212.OverlayValues[67] = d67
			ps212.OverlayValues[68] = d68
			ps212.OverlayValues[69] = d69
			ps212.OverlayValues[70] = d70
			ps212.OverlayValues[71] = d71
			ps212.OverlayValues[72] = d72
			ps212.OverlayValues[73] = d73
			ps212.OverlayValues[74] = d74
			ps212.OverlayValues[135] = d135
			ps212.OverlayValues[136] = d136
			ps212.OverlayValues[137] = d137
			ps212.OverlayValues[138] = d138
			ps212.OverlayValues[139] = d139
			ps212.OverlayValues[140] = d140
			ps212.OverlayValues[207] = d207
			ps212.OverlayValues[208] = d208
			ps212.OverlayValues[209] = d209
			ps212.OverlayValues[210] = d210
			ps212.OverlayValues[211] = d211
			ps212.PhiValues = make([]scm.JITValueDesc, 2)
			d213 = d207
			ps212.PhiValues[0] = d213
			d214 = d1
			ps212.PhiValues[1] = d214
			if ps212.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps212)
			return result
			}
			bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
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
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 135 && ps.OverlayValues[135].Loc != scm.LocNone {
				d135 = ps.OverlayValues[135]
			}
			if len(ps.OverlayValues) > 136 && ps.OverlayValues[136].Loc != scm.LocNone {
				d136 = ps.OverlayValues[136]
			}
			if len(ps.OverlayValues) > 137 && ps.OverlayValues[137].Loc != scm.LocNone {
				d137 = ps.OverlayValues[137]
			}
			if len(ps.OverlayValues) > 138 && ps.OverlayValues[138].Loc != scm.LocNone {
				d138 = ps.OverlayValues[138]
			}
			if len(ps.OverlayValues) > 139 && ps.OverlayValues[139].Loc != scm.LocNone {
				d139 = ps.OverlayValues[139]
			}
			if len(ps.OverlayValues) > 140 && ps.OverlayValues[140].Loc != scm.LocNone {
				d140 = ps.OverlayValues[140]
			}
			if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != scm.LocNone {
				d207 = ps.OverlayValues[207]
			}
			if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != scm.LocNone {
				d208 = ps.OverlayValues[208]
			}
			if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != scm.LocNone {
				d209 = ps.OverlayValues[209]
			}
			if len(ps.OverlayValues) > 210 && ps.OverlayValues[210].Loc != scm.LocNone {
				d210 = ps.OverlayValues[210]
			}
			if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
				d211 = ps.OverlayValues[211]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
				d214 = ps.OverlayValues[214]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			if d0.Loc == scm.LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == scm.LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d36)
			if d36.Loc == scm.LocReg {
				ctx.ProtectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.ProtectReg(d36.Reg)
				ctx.ProtectReg(d36.Reg2)
			}
			d215 = d0
			if d215.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d215)
			d216 = d215
			if d216.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: d216.Type, Imm: scm.NewInt(int64(uint64(d216.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d216.Reg, 32)
				ctx.EmitShrRegImm8(d216.Reg, 32)
			}
			ctx.EmitStoreToStack(d216, int32(bbs[1].PhiBase)+int32(0))
			d217 = d36
			if d217.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d217)
			d218 = d217
			if d218.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: d218.Type, Imm: scm.NewInt(int64(uint64(d218.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d218.Reg, 32)
				ctx.EmitShrRegImm8(d218.Reg, 32)
			}
			ctx.EmitStoreToStack(d218, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == scm.LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d36.Loc == scm.LocReg {
				ctx.UnprotectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d36.Reg)
				ctx.UnprotectReg(d36.Reg2)
			}
			ps219 := scm.PhiState{General: ps.General}
			ps219.OverlayValues = make([]scm.JITValueDesc, 219)
			ps219.OverlayValues[0] = d0
			ps219.OverlayValues[1] = d1
			ps219.OverlayValues[2] = d2
			ps219.OverlayValues[3] = d3
			ps219.OverlayValues[4] = d4
			ps219.OverlayValues[5] = d5
			ps219.OverlayValues[7] = d7
			ps219.OverlayValues[8] = d8
			ps219.OverlayValues[9] = d9
			ps219.OverlayValues[10] = d10
			ps219.OverlayValues[11] = d11
			ps219.OverlayValues[12] = d12
			ps219.OverlayValues[15] = d15
			ps219.OverlayValues[16] = d16
			ps219.OverlayValues[34] = d34
			ps219.OverlayValues[35] = d35
			ps219.OverlayValues[36] = d36
			ps219.OverlayValues[37] = d37
			ps219.OverlayValues[38] = d38
			ps219.OverlayValues[39] = d39
			ps219.OverlayValues[40] = d40
			ps219.OverlayValues[41] = d41
			ps219.OverlayValues[42] = d42
			ps219.OverlayValues[43] = d43
			ps219.OverlayValues[44] = d44
			ps219.OverlayValues[45] = d45
			ps219.OverlayValues[46] = d46
			ps219.OverlayValues[47] = d47
			ps219.OverlayValues[48] = d48
			ps219.OverlayValues[49] = d49
			ps219.OverlayValues[50] = d50
			ps219.OverlayValues[51] = d51
			ps219.OverlayValues[52] = d52
			ps219.OverlayValues[53] = d53
			ps219.OverlayValues[54] = d54
			ps219.OverlayValues[55] = d55
			ps219.OverlayValues[56] = d56
			ps219.OverlayValues[57] = d57
			ps219.OverlayValues[58] = d58
			ps219.OverlayValues[59] = d59
			ps219.OverlayValues[60] = d60
			ps219.OverlayValues[61] = d61
			ps219.OverlayValues[62] = d62
			ps219.OverlayValues[63] = d63
			ps219.OverlayValues[64] = d64
			ps219.OverlayValues[65] = d65
			ps219.OverlayValues[66] = d66
			ps219.OverlayValues[67] = d67
			ps219.OverlayValues[68] = d68
			ps219.OverlayValues[69] = d69
			ps219.OverlayValues[70] = d70
			ps219.OverlayValues[71] = d71
			ps219.OverlayValues[72] = d72
			ps219.OverlayValues[73] = d73
			ps219.OverlayValues[74] = d74
			ps219.OverlayValues[135] = d135
			ps219.OverlayValues[136] = d136
			ps219.OverlayValues[137] = d137
			ps219.OverlayValues[138] = d138
			ps219.OverlayValues[139] = d139
			ps219.OverlayValues[140] = d140
			ps219.OverlayValues[207] = d207
			ps219.OverlayValues[208] = d208
			ps219.OverlayValues[209] = d209
			ps219.OverlayValues[210] = d210
			ps219.OverlayValues[211] = d211
			ps219.OverlayValues[213] = d213
			ps219.OverlayValues[214] = d214
			ps219.OverlayValues[215] = d215
			ps219.OverlayValues[216] = d216
			ps219.OverlayValues[217] = d217
			ps219.OverlayValues[218] = d218
			ps219.PhiValues = make([]scm.JITValueDesc, 2)
			d220 = d0
			ps219.PhiValues[0] = d220
			d221 = d36
			ps219.PhiValues[1] = d221
			if ps219.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps219)
			return result
			}
			ps222 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps222)
			ctx.MarkLabel(lbl0)
			d223 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d223)
			ctx.BindReg(r2, &d223)
			ctx.EmitMovPairToResult(&d223, &result)
			ctx.FreeReg(r1)
			ctx.FreeReg(r2)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r0, int32(48))
			ctx.EmitAddRSP32(int32(48))
			return result
}

func (s *StorageSparse) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(2)) // 2 = StorageSparse
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	for k, v := range s.values {
		vbytes, err := json.Marshal(uint64(s.recids.GetValueUInt(uint32(k)) + uint64(s.recids.offset)))
		if err != nil {
			panic(err)
		}
		f.Write(vbytes)
		f.Write([]byte("\n")) // endline so the serialized file becomes a jsonl file
		vbytes, err = json.Marshal(v)
		if err != nil {
			panic(err)
		}
		f.Write(vbytes)
		f.Write([]byte("\n")) // endline so the serialized file becomes a jsonl file
	}
}
func (s *StorageSparse) Deserialize(f io.Reader) uint {
	// No version byte: this type had no padding byte in v0.1.0.
	// Count is read directly.  Format changes require a new magic byte.
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.count = l
	var l2 uint64
	binary.Read(f, binary.LittleEndian, &l2)
	s.values = make([]scm.Scmer, l2)
	s.i = l2
	scanner := bufio.NewScanner(f)
	s.recids.prepare()
	s.recids.scan(0, scm.NewInt(0))
	s.recids.scan(uint32(l2-1), scm.NewInt(int64(l-1)))
	s.recids.init(uint32(l2))
	i := 0
	for {
		var k uint64
		if !scanner.Scan() {
			break
		}
		json.Unmarshal(scanner.Bytes(), &k)
		if !scanner.Scan() {
			break
		}
		var v any
		json.Unmarshal(scanner.Bytes(), &v)
		s.recids.build(uint32(i), scm.NewInt(int64(k)))
		s.values[i] = scm.TransformFromJSON(v)
		i++
	}
	s.recids.finish()
	return uint(l)
}

func (s *StorageSparse) GetCachedReader() ColumnReader { return s }

func (s *StorageSparse) GetValue(i uint32) scm.Scmer {
	var lower uint32 = 0
	var upper uint32 = uint32(s.i)
	for {
		if lower == upper {
			return scm.NewNil() // sparse value
		}
		pivot := (lower + upper) / 2
		recid := s.recids.GetValueUInt(pivot) + uint64(s.recids.offset)
		if recid == uint64(i) {
			return s.values[pivot] // found the value
		}
		if recid < uint64(i) {
			lower = pivot + 1
		} else {
			upper = pivot
		}

	}
}

func (s *StorageSparse) scan(i uint32, value scm.Scmer) {
	if !value.IsNil() {
		s.recids.scan(uint32(s.i), scm.NewInt(int64(i)))
		s.i++
	}
}
func (s *StorageSparse) prepare() {
	s.i = 0
}
func (s *StorageSparse) init(i uint32) {
	s.values = make([]scm.Scmer, s.i)
	s.count = uint64(i)
	s.recids.init(uint32(s.i))
	s.i = 0
}
func (s *StorageSparse) build(i uint32, value scm.Scmer) {
	// store
	if !value.IsNil() {
		s.recids.build(uint32(s.i), scm.NewInt(int64(i)))
		s.values[s.i] = value
		s.i++
	}
}
func (s *StorageSparse) finish() {
	s.recids.finish()
}

// soley to StorageSparse
func (s *StorageSparse) proposeCompression(i uint32) ColumnStorage {
	return nil
}
