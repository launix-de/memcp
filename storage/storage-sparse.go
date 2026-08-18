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
			var d3 scm.JITValueDesc
			_ = d3
			var d4 scm.JITValueDesc
			_ = d4
			var d5 scm.JITValueDesc
			_ = d5
			var d6 scm.JITValueDesc
			_ = d6
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
			var d16 scm.JITValueDesc
			_ = d16
			var d17 scm.JITValueDesc
			_ = d17
			var d35 scm.JITValueDesc
			_ = d35
			var d36 scm.JITValueDesc
			_ = d36
			var d37 scm.JITValueDesc
			_ = d37
			var d38 scm.JITValueDesc
			_ = d38
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
			var d75 scm.JITValueDesc
			_ = d75
			var d76 scm.JITValueDesc
			_ = d76
			var d137 scm.JITValueDesc
			_ = d137
			var d138 scm.JITValueDesc
			_ = d138
			var d139 scm.JITValueDesc
			_ = d139
			var d140 scm.JITValueDesc
			_ = d140
			var d141 scm.JITValueDesc
			_ = d141
			var d142 scm.JITValueDesc
			_ = d142
			var d209 scm.JITValueDesc
			_ = d209
			var d210 scm.JITValueDesc
			_ = d210
			var d211 scm.JITValueDesc
			_ = d211
			var d212 scm.JITValueDesc
			_ = d212
			var d213 scm.JITValueDesc
			_ = d213
			var d215 scm.JITValueDesc
			_ = d215
			var d216 scm.JITValueDesc
			_ = d216
			var d217 scm.JITValueDesc
			_ = d217
			var d218 scm.JITValueDesc
			_ = d218
			var d219 scm.JITValueDesc
			_ = d219
			var d220 scm.JITValueDesc
			_ = d220
			var d222 scm.JITValueDesc
			_ = d222
			var d223 scm.JITValueDesc
			_ = d223
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
			phiBase0 := ctx.AllocStack(int32(32))
			d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(0)}
			d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0)+int32(16)}
			var bbs [8]scm.BBDescriptor
			bbs[1].PhiBase = int32(phiBase0) + int32(0)
			bbs[1].PhiCount = uint16(2)
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			ctx.ReclaimUntrackedRegs()
			var d3 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).i)
				r2 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r2, fieldAddr)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
				ctx.BindReg(r2, &d3)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
				r3 := ctx.AllocReg()
				ctx.EmitMovRegMem(r3, thisptr.Reg, off)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
				ctx.BindReg(r3, &d3)
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d4 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d3.Imm.Int()))))}
			} else {
				r4 := ctx.AllocReg()
				ctx.EmitMovRegReg(r4, d3.Reg)
				ctx.EmitShlRegImm8(r4, 32)
				ctx.EmitShrRegImm8(r4, 32)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
				ctx.BindReg(r4, &d4)
			}
			ctx.EnsureDesc(&d4)
			if d4.Loc == scm.LocReg {
				ctx.ProtectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.ProtectReg(d4.Reg)
				ctx.ProtectReg(d4.Reg2)
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
			d5 = d4
			if d5.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d5)
			d6 = d5
			if d6.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: d6.Type, Imm: scm.NewInt(int64(uint64(d6.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d6.Reg, 32)
				ctx.EmitShrRegImm8(d6.Reg, 32)
			}
			ctx.EmitStoreToStack(d6, int32(bbs[1].PhiBase)+int32(16))
			if d4.Loc == scm.LocReg {
				ctx.UnprotectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d4.Reg)
				ctx.UnprotectReg(d4.Reg2)
			}
			ps7 := scm.PhiState{General: ps.General}
			ps7.OverlayValues = make([]scm.JITValueDesc, 7)
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			ps7.OverlayValues[4] = d4
			ps7.OverlayValues[5] = d5
			ps7.OverlayValues[6] = d6
			ps7.PhiValues = make([]scm.JITValueDesc, 2)
			d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps7.PhiValues[0] = d8
			d9 = d4
			ps7.PhiValues[1] = d9
			if ps7.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps7)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d10 := ps.PhiValues[0]
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d11 := ps.PhiValues[1]
					ctx.EnsureDesc(&d11)
					ctx.EmitStoreToStack(d11, int32(bbs[1].PhiBase)+int32(16))
				}
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
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
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d1 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d2 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d12 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d1.Imm.Int()) == uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r5 := ctx.AllocRegExcept(d1.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.EmitCmpInt64(d1.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r5, scm.CcE)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r5}
				ctx.BindReg(r5, &d12)
			} else if d1.Loc == scm.LocImm {
				r6 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.EmitSetcc(r6, scm.CcE)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d12)
			} else {
				r7 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitCmpInt64(d1.Reg, d2.Reg)
				ctx.EmitSetcc(r7, scm.CcE)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d12)
			}
			d13 = d12
			ctx.EnsureDesc(&d13)
			if d13.Loc != scm.LocImm && d13.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d13.Loc == scm.LocImm {
				if d13.Imm.Bool() {
			ps14 := scm.PhiState{General: ps.General}
			ps14.OverlayValues = make([]scm.JITValueDesc, 14)
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
			ps14.OverlayValues[13] = d13
					return bbs[2].RenderPS(ps14)
				}
			ps15 := scm.PhiState{General: ps.General}
			ps15.OverlayValues = make([]scm.JITValueDesc, 14)
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
			ps15.OverlayValues[13] = d13
				return bbs[3].RenderPS(ps15)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d16 := ps.PhiValues[0]
					ctx.EnsureDesc(&d16)
					ctx.EmitStoreToStack(d16, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d17 := ps.PhiValues[1]
					ctx.EnsureDesc(&d17)
					ctx.EmitStoreToStack(d17, int32(bbs[1].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d13.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl9)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl4)
			ps18 := scm.PhiState{General: true}
			ps18.OverlayValues = make([]scm.JITValueDesc, 18)
			ps18.OverlayValues[1] = d1
			ps18.OverlayValues[2] = d2
			ps18.OverlayValues[3] = d3
			ps18.OverlayValues[4] = d4
			ps18.OverlayValues[5] = d5
			ps18.OverlayValues[6] = d6
			ps18.OverlayValues[8] = d8
			ps18.OverlayValues[9] = d9
			ps18.OverlayValues[10] = d10
			ps18.OverlayValues[11] = d11
			ps18.OverlayValues[12] = d12
			ps18.OverlayValues[13] = d13
			ps18.OverlayValues[16] = d16
			ps18.OverlayValues[17] = d17
			ps19 := scm.PhiState{General: true}
			ps19.OverlayValues = make([]scm.JITValueDesc, 18)
			ps19.OverlayValues[1] = d1
			ps19.OverlayValues[2] = d2
			ps19.OverlayValues[3] = d3
			ps19.OverlayValues[4] = d4
			ps19.OverlayValues[5] = d5
			ps19.OverlayValues[6] = d6
			ps19.OverlayValues[8] = d8
			ps19.OverlayValues[9] = d9
			ps19.OverlayValues[10] = d10
			ps19.OverlayValues[11] = d11
			ps19.OverlayValues[12] = d12
			ps19.OverlayValues[13] = d13
			ps19.OverlayValues[16] = d16
			ps19.OverlayValues[17] = d17
			snap20 := d1
			snap21 := d2
			snap22 := d3
			snap23 := d4
			snap24 := d5
			snap25 := d6
			snap26 := d8
			snap27 := d9
			snap28 := d10
			snap29 := d11
			snap30 := d12
			snap31 := d13
			snap32 := d16
			snap33 := d17
			alloc34 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps19)
			}
			ctx.RestoreAllocState(alloc34)
			d1 = snap20
			d2 = snap21
			d3 = snap22
			d4 = snap23
			d5 = snap24
			d6 = snap25
			d8 = snap26
			d9 = snap27
			d10 = snap28
			d11 = snap29
			d12 = snap30
			d13 = snap31
			d16 = snap32
			d17 = snap33
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps18)
			}
			return result
			ctx.FreeDesc(&d12)
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			ctx.ReclaimUntrackedRegs()
			d35 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d35)
			ctx.BindReg(r1, &d35)
			ctx.EmitMakeNil(d35)
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.ProtectReg(d1.Reg)
			ctx.EnsureDesc(&d2)
			ctx.UnprotectReg(d1.Reg)
			var d36 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + d2.Imm.Int())}
			} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(r8, d1.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d36)
			} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d2.Reg}
				ctx.BindReg(d2.Reg, &d36)
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.EmitAddInt64(scratch, d2.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(scratch, d1.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d2.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else {
				r9 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
				ctx.EmitMovRegReg(r9, d1.Reg)
				ctx.EmitAddInt64(r9, d2.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d36)
			}
			if d36.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: d36.Type, Imm: scm.NewInt(int64(uint64(d36.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d36.Reg, 32)
				ctx.EmitShrRegImm8(d36.Reg, 32)
			}
			if d36.Loc == scm.LocReg && d1.Loc == scm.LocReg && d36.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() / 2)}
			} else {
				r10 := ctx.AllocRegExcept(d36.Reg)
				ctx.EmitMovRegReg(r10, d36.Reg)
				ctx.EmitShrRegImm8(r10, 1)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d37)
			}
			if d37.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: d37.Type, Imm: scm.NewInt(int64(uint64(d37.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d37.Reg, 32)
				ctx.EmitShrRegImm8(d37.Reg, 32)
			}
			if d37.Loc == scm.LocReg && d36.Loc == scm.LocReg && d37.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d37)
			d38 = d37
			_ = d38
			r11 := d37.Loc == scm.LocReg
			r12 := d37.Reg
			if r11 { ctx.ProtectReg(r12) }
			phiBase39 := ctx.AllocStack(int32(16))
			d40 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase39)+int32(32)}
			lbl11 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d40 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d38)
			var d41 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d38.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.EmitMovRegReg(r13, d38.Reg)
				ctx.EmitShlRegImm8(r13, 32)
				ctx.EmitShrRegImm8(r13, 32)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d41)
			}
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				r14 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r14, fieldAddr)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
				ctx.BindReg(r14, &d42)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r15 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r15, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d42)
			}
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d42.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.EmitMovRegReg(r16, d42.Reg)
				ctx.EmitShlRegImm8(r16, 56)
				ctx.EmitShrRegImm8(r16, 56)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d43)
			}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d41)
			ctx.ProtectReg(d41.Reg)
			ctx.EnsureDesc(&d43)
			ctx.UnprotectReg(d41.Reg)
			var d44 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() * d43.Imm.Int())}
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.EmitImulInt64(scratch, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(scratch, d41.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d43.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else {
				r17 := ctx.AllocRegExcept(d41.Reg, d43.Reg)
				ctx.EmitMovRegReg(r17, d41.Reg)
				ctx.EmitImulInt64(r17, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d44)
			}
			if d44.Loc == scm.LocReg && d41.Loc == scm.LocReg && d44.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d43)
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				r18 := ctx.AllocReg()
				r19 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r18, fieldAddr)
				ctx.EmitMovRegMem64(r19, fieldAddr+8)
				d45 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r18, Reg2: r19}
				ctx.BindReg(r18, &d45)
				ctx.BindReg(r19, &d45)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				r20 := ctx.AllocReg()
				r21 := ctx.AllocReg()
				ctx.EmitMovRegMem(r20, thisptr.Reg, off)
				ctx.EmitMovRegMem(r21, thisptr.Reg, off+8)
				d45 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r20, Reg2: r21}
				ctx.BindReg(r20, &d45)
				ctx.BindReg(r21, &d45)
			}
			ctx.EnsureDesc(&d44)
			var d46 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() / 64)}
			} else {
				r22 := ctx.AllocRegExcept(d44.Reg)
				ctx.EmitMovRegReg(r22, d44.Reg)
				ctx.EmitShrRegImm8(r22, 6)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d46)
			}
			if d46.Loc == scm.LocReg && d44.Loc == scm.LocReg && d46.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d46)
			r23 := ctx.AllocReg()
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d45)
			if d46.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r23, uint64(d46.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r23, d46.Reg)
				ctx.EmitShlRegImm8(r23, 3)
			}
			if d45.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitAddInt64(r23, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r23, d45.Reg)
			}
			r24 := ctx.AllocRegExcept(r23)
			ctx.EmitMovRegMem(r24, r23, 0)
			ctx.FreeReg(r23)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d47)
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&d44)
			var d48 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r25 := ctx.AllocRegExcept(d44.Reg)
				ctx.EmitMovRegReg(r25, d44.Reg)
				ctx.EmitAndRegImm32(r25, 63)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d48)
			}
			if d48.Loc == scm.LocReg && d44.Loc == scm.LocReg && d48.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d47.Imm.Int()) << uint64(d48.Imm.Int())))}
			} else if d48.Loc == scm.LocImm {
				r26 := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(r26, d47.Reg)
				ctx.EmitShlRegImm8(r26, uint8(d48.Imm.Int()))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d49)
			} else {
				{
					shiftSrc := d47.Reg
					r27 := ctx.AllocRegExcept(d47.Reg)
					ctx.EmitMovRegReg(r27, d47.Reg)
					shiftSrc = r27
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d48.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d48.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d48.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d49)
				}
			}
			if d49.Loc == scm.LocReg && d47.Loc == scm.LocReg && d49.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d44)
			var d50 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r28 := ctx.AllocRegExcept(d44.Reg)
				ctx.EmitMovRegReg(r28, d44.Reg)
				ctx.EmitAndRegImm32(r28, 63)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d50)
			}
			if d50.Loc == scm.LocReg && d44.Loc == scm.LocReg && d50.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d51 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d42.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.EmitMovRegReg(r29, d42.Reg)
				ctx.EmitShlRegImm8(r29, 56)
				ctx.EmitShrRegImm8(r29, 56)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d51)
			}
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d50)
			ctx.ProtectReg(d50.Reg)
			ctx.EnsureDesc(&d51)
			ctx.UnprotectReg(d50.Reg)
			var d52 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() + d51.Imm.Int())}
			} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(r30, d50.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d52)
			} else if d50.Loc == scm.LocImm && d50.Imm.Int() == 0 {
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d51.Reg}
				ctx.BindReg(d51.Reg, &d52)
			} else if d50.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d50.Imm.Int()))
				ctx.EmitAddInt64(scratch, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(scratch, d50.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d51.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else {
				r31 := ctx.AllocRegExcept(d50.Reg, d51.Reg)
				ctx.EmitMovRegReg(r31, d50.Reg)
				ctx.EmitAddInt64(r31, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d52)
			}
			if d52.Loc == scm.LocReg && d50.Loc == scm.LocReg && d52.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d50)
			ctx.FreeDesc(&d51)
			ctx.EnsureDesc(&d52)
			var d53 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d52.Imm.Int()) > uint64(64))}
			} else {
				r32 := ctx.AllocRegExcept(d52.Reg)
				ctx.EmitCmpRegImm32(d52.Reg, 64)
				ctx.EmitSetcc(r32, scm.CcA)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
				ctx.BindReg(r32, &d53)
			}
			ctx.FreeDesc(&d52)
			d54 = d53
			ctx.EnsureDesc(&d54)
			if d54.Loc != scm.LocImm && d54.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			lbl15 := ctx.ReserveLabel()
			if d54.Loc == scm.LocImm {
				if d54.Imm.Bool() {
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl12)
				} else {
					ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d49)
			if d49.Loc == scm.LocReg {
				ctx.ProtectReg(d49.Reg)
			} else if d49.Loc == scm.LocRegPair {
				ctx.ProtectReg(d49.Reg)
				ctx.ProtectReg(d49.Reg2)
			}
			d55 = d49
			if d55.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d55)
			ctx.EmitStoreToStack(d55, int32(bbs[2].PhiBase)+int32(0))
			if d49.Loc == scm.LocReg {
				ctx.UnprotectReg(d49.Reg)
			} else if d49.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d49.Reg)
				ctx.UnprotectReg(d49.Reg2)
			}
					ctx.EmitJmp(lbl13)
				}
			} else {
				ctx.EmitCmpRegImm32(d54.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl14)
				ctx.EmitJmp(lbl15)
				ctx.MarkLabel(lbl14)
				ctx.EmitJmp(lbl12)
				ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d49)
			if d49.Loc == scm.LocReg {
				ctx.ProtectReg(d49.Reg)
			} else if d49.Loc == scm.LocRegPair {
				ctx.ProtectReg(d49.Reg)
				ctx.ProtectReg(d49.Reg2)
			}
			d56 = d49
			if d56.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d56)
			ctx.EmitStoreToStack(d56, int32(bbs[2].PhiBase)+int32(0))
			if d49.Loc == scm.LocReg {
				ctx.UnprotectReg(d49.Reg)
			} else if d49.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d49.Reg)
				ctx.UnprotectReg(d49.Reg2)
			}
				ctx.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d53)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl13)
			ctx.ResolveFixups()
			d40 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d57 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d42.Imm.Int()))))}
			} else {
				r33 := ctx.AllocReg()
				ctx.EmitMovRegReg(r33, d42.Reg)
				ctx.EmitShlRegImm8(r33, 56)
				ctx.EmitShrRegImm8(r33, 56)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d57)
			}
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d58)
			ctx.ProtectReg(d58.Reg)
			ctx.EnsureDesc(&d57)
			ctx.UnprotectReg(d58.Reg)
			var d59 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() - d57.Imm.Int())}
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				r34 := ctx.AllocRegExcept(d58.Reg)
				ctx.EmitMovRegReg(r34, d58.Reg)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d59)
			} else if d58.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d58.Imm.Int()))
				ctx.EmitSubInt64(scratch, d57.Reg)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d59)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d58.Reg)
				ctx.EmitMovRegReg(scratch, d58.Reg)
				if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d57.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d59)
			} else {
				r35 := ctx.AllocRegExcept(d58.Reg, d57.Reg)
				ctx.EmitMovRegReg(r35, d58.Reg)
				ctx.EmitSubInt64(r35, d57.Reg)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d59)
			}
			if d59.Loc == scm.LocReg && d58.Loc == scm.LocReg && d59.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d40.Imm.Int()) >> uint64(d59.Imm.Int())))}
			} else if d59.Loc == scm.LocImm {
				r36 := ctx.AllocRegExcept(d40.Reg)
				ctx.EmitMovRegReg(r36, d40.Reg)
				ctx.EmitShrRegImm8(r36, uint8(d59.Imm.Int()))
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d60)
			} else {
				{
					shiftSrc := d40.Reg
					r37 := ctx.AllocRegExcept(d40.Reg)
					ctx.EmitMovRegReg(r37, d40.Reg)
					shiftSrc = r37
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d59.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d59.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d59.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d60)
				}
			}
			if d60.Loc == scm.LocReg && d40.Loc == scm.LocReg && d60.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d59)
			r38 := ctx.AllocReg()
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d60)
			if d60.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r38, d60)
			}
			ctx.EmitJmp(lbl11)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl12)
			ctx.ResolveFixups()
			d40 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d44)
			var d61 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() / 64)}
			} else {
				r39 := ctx.AllocRegExcept(d44.Reg)
				ctx.EmitMovRegReg(r39, d44.Reg)
				ctx.EmitShrRegImm8(r39, 6)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d61)
			}
			if d61.Loc == scm.LocReg && d44.Loc == scm.LocReg && d61.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d61.Reg)
				ctx.EmitMovRegReg(scratch, d61.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			}
			if d62.Loc == scm.LocReg && d61.Loc == scm.LocReg && d62.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d61)
			ctx.EnsureDesc(&d62)
			r40 := ctx.AllocReg()
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d45)
			if d62.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r40, uint64(d62.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r40, d62.Reg)
				ctx.EmitShlRegImm8(r40, 3)
			}
			if d45.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitAddInt64(r40, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r40, d45.Reg)
			}
			r41 := ctx.AllocRegExcept(r40)
			ctx.EmitMovRegMem(r41, r40, 0)
			ctx.FreeReg(r40)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d63)
			ctx.FreeDesc(&d62)
			ctx.EnsureDesc(&d44)
			var d64 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r42 := ctx.AllocRegExcept(d44.Reg)
				ctx.EmitMovRegReg(r42, d44.Reg)
				ctx.EmitAndRegImm32(r42, 63)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d64)
			}
			if d64.Loc == scm.LocReg && d44.Loc == scm.LocReg && d64.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d65)
			ctx.ProtectReg(d65.Reg)
			ctx.EnsureDesc(&d64)
			ctx.UnprotectReg(d65.Reg)
			var d66 scm.JITValueDesc
			if d65.Loc == scm.LocImm && d64.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d65.Imm.Int() - d64.Imm.Int())}
			} else if d64.Loc == scm.LocImm && d64.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d65.Reg)
				ctx.EmitMovRegReg(r43, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d66)
			} else if d65.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d64.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d65.Imm.Int()))
				ctx.EmitSubInt64(scratch, d64.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d66)
			} else if d64.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d65.Reg)
				ctx.EmitMovRegReg(scratch, d65.Reg)
				if d64.Imm.Int() >= -2147483648 && d64.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d64.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d64.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d66)
			} else {
				r44 := ctx.AllocRegExcept(d65.Reg, d64.Reg)
				ctx.EmitMovRegReg(r44, d65.Reg)
				ctx.EmitSubInt64(r44, d64.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d66)
			}
			if d66.Loc == scm.LocReg && d65.Loc == scm.LocReg && d66.Reg == d65.Reg {
				ctx.TransferReg(d65.Reg)
				d65.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d64)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d66)
			var d67 scm.JITValueDesc
			if d63.Loc == scm.LocImm && d66.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d63.Imm.Int()) >> uint64(d66.Imm.Int())))}
			} else if d66.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d63.Reg)
				ctx.EmitMovRegReg(r45, d63.Reg)
				ctx.EmitShrRegImm8(r45, uint8(d66.Imm.Int()))
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d67)
			} else {
				{
					shiftSrc := d63.Reg
					r46 := ctx.AllocRegExcept(d63.Reg)
					ctx.EmitMovRegReg(r46, d63.Reg)
					shiftSrc = r46
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d66.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d66.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d66.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d67)
				}
			}
			if d67.Loc == scm.LocReg && d63.Loc == scm.LocReg && d67.Reg == d63.Reg {
				ctx.TransferReg(d63.Reg)
				d63.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d63)
			ctx.FreeDesc(&d66)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d67)
			var d68 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() | d67.Imm.Int())}
			} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d67.Reg}
				ctx.BindReg(d67.Reg, &d68)
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				r47 := ctx.AllocRegExcept(d49.Reg)
				ctx.EmitMovRegReg(r47, d49.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d68)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d67.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.EmitOrInt64(scratch, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d68)
			} else if d67.Loc == scm.LocImm {
				r48 := ctx.AllocRegExcept(d49.Reg)
				ctx.EmitMovRegReg(r48, d49.Reg)
				if d67.Imm.Int() >= -2147483648 && d67.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r48, int32(d67.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d67.Imm.Int()))
					ctx.EmitOrInt64(r48, scm.RegR11)
				}
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d68)
			} else {
				r49 := ctx.AllocRegExcept(d49.Reg, d67.Reg)
				ctx.EmitMovRegReg(r49, d49.Reg)
				ctx.EmitOrInt64(r49, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d68)
			}
			if d68.Loc == scm.LocReg && d49.Loc == scm.LocReg && d68.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d67)
			ctx.EnsureDesc(&d68)
			if d68.Loc == scm.LocReg {
				ctx.ProtectReg(d68.Reg)
			} else if d68.Loc == scm.LocRegPair {
				ctx.ProtectReg(d68.Reg)
				ctx.ProtectReg(d68.Reg2)
			}
			d69 = d68
			if d69.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d69)
			ctx.EmitStoreToStack(d69, int32(bbs[2].PhiBase)+int32(0))
			if d68.Loc == scm.LocReg {
				ctx.UnprotectReg(d68.Reg)
			} else if d68.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d68.Reg)
				ctx.UnprotectReg(d68.Reg2)
			}
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl11)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d70)
			ctx.BindReg(r38, &d70)
			if r11 { ctx.UnprotectReg(r12) }
			var d71 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				r50 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r50, fieldAddr)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d71)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r51 := ctx.AllocReg()
				ctx.EmitMovRegMem(r51, thisptr.Reg, off)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d71)
			}
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d71)
			var d72 scm.JITValueDesc
			if d71.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d71.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.EmitMovRegReg(r52, d71.Reg)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d72)
			}
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d70)
			ctx.ProtectReg(d70.Reg)
			ctx.EnsureDesc(&d72)
			ctx.UnprotectReg(d70.Reg)
			var d73 scm.JITValueDesc
			if d70.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d70.Imm.Int() + d72.Imm.Int())}
			} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
				r53 := ctx.AllocRegExcept(d70.Reg)
				ctx.EmitMovRegReg(r53, d70.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d73)
			} else if d70.Loc == scm.LocImm && d70.Imm.Int() == 0 {
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d72.Reg}
				ctx.BindReg(d72.Reg, &d73)
			} else if d70.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d70.Imm.Int()))
				ctx.EmitAddInt64(scratch, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else if d72.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d70.Reg)
				ctx.EmitMovRegReg(scratch, d70.Reg)
				if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d72.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else {
				r54 := ctx.AllocRegExcept(d70.Reg, d72.Reg)
				ctx.EmitMovRegReg(r54, d70.Reg)
				ctx.EmitAddInt64(r54, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d73)
			}
			if d73.Loc == scm.LocReg && d70.Loc == scm.LocReg && d73.Reg == d70.Reg {
				ctx.TransferReg(d70.Reg)
				d70.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d70)
			ctx.FreeDesc(&d72)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d74 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r55 := ctx.AllocReg()
				ctx.EmitMovRegReg(r55, idxInt.Reg)
				ctx.EmitShlRegImm8(r55, 32)
				ctx.EmitShrRegImm8(r55, 32)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d74)
			}
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d74)
			var d75 scm.JITValueDesc
			if d73.Loc == scm.LocImm && d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d73.Imm.Int()) == uint64(d74.Imm.Int()))}
			} else if d74.Loc == scm.LocImm {
				r56 := ctx.AllocRegExcept(d73.Reg)
				if d74.Imm.Int() >= -2147483648 && d74.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d73.Reg, int32(d74.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
					ctx.EmitCmpInt64(d73.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r56, scm.CcE)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d75)
			} else if d73.Loc == scm.LocImm {
				r57 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d74.Reg)
				ctx.EmitSetcc(r57, scm.CcE)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d75)
			} else {
				r58 := ctx.AllocRegExcept(d73.Reg)
				ctx.EmitCmpInt64(d73.Reg, d74.Reg)
				ctx.EmitSetcc(r58, scm.CcE)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d75)
			}
			ctx.FreeDesc(&d74)
			d76 = d75
			ctx.EnsureDesc(&d76)
			if d76.Loc != scm.LocImm && d76.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d76.Loc == scm.LocImm {
				if d76.Imm.Bool() {
			ps77 := scm.PhiState{General: ps.General}
			ps77.OverlayValues = make([]scm.JITValueDesc, 77)
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
			ps77.OverlayValues[13] = d13
			ps77.OverlayValues[16] = d16
			ps77.OverlayValues[17] = d17
			ps77.OverlayValues[35] = d35
			ps77.OverlayValues[36] = d36
			ps77.OverlayValues[37] = d37
			ps77.OverlayValues[38] = d38
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
			ps77.OverlayValues[75] = d75
			ps77.OverlayValues[76] = d76
					return bbs[4].RenderPS(ps77)
				}
			ps78 := scm.PhiState{General: ps.General}
			ps78.OverlayValues = make([]scm.JITValueDesc, 77)
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
			ps78.OverlayValues[13] = d13
			ps78.OverlayValues[16] = d16
			ps78.OverlayValues[17] = d17
			ps78.OverlayValues[35] = d35
			ps78.OverlayValues[36] = d36
			ps78.OverlayValues[37] = d37
			ps78.OverlayValues[38] = d38
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
			ps78.OverlayValues[75] = d75
			ps78.OverlayValues[76] = d76
				return bbs[5].RenderPS(ps78)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d76.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl16)
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl6)
			ps79 := scm.PhiState{General: true}
			ps79.OverlayValues = make([]scm.JITValueDesc, 77)
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
			ps79.OverlayValues[13] = d13
			ps79.OverlayValues[16] = d16
			ps79.OverlayValues[17] = d17
			ps79.OverlayValues[35] = d35
			ps79.OverlayValues[36] = d36
			ps79.OverlayValues[37] = d37
			ps79.OverlayValues[38] = d38
			ps79.OverlayValues[40] = d40
			ps79.OverlayValues[41] = d41
			ps79.OverlayValues[42] = d42
			ps79.OverlayValues[43] = d43
			ps79.OverlayValues[44] = d44
			ps79.OverlayValues[45] = d45
			ps79.OverlayValues[46] = d46
			ps79.OverlayValues[47] = d47
			ps79.OverlayValues[48] = d48
			ps79.OverlayValues[49] = d49
			ps79.OverlayValues[50] = d50
			ps79.OverlayValues[51] = d51
			ps79.OverlayValues[52] = d52
			ps79.OverlayValues[53] = d53
			ps79.OverlayValues[54] = d54
			ps79.OverlayValues[55] = d55
			ps79.OverlayValues[56] = d56
			ps79.OverlayValues[57] = d57
			ps79.OverlayValues[58] = d58
			ps79.OverlayValues[59] = d59
			ps79.OverlayValues[60] = d60
			ps79.OverlayValues[61] = d61
			ps79.OverlayValues[62] = d62
			ps79.OverlayValues[63] = d63
			ps79.OverlayValues[64] = d64
			ps79.OverlayValues[65] = d65
			ps79.OverlayValues[66] = d66
			ps79.OverlayValues[67] = d67
			ps79.OverlayValues[68] = d68
			ps79.OverlayValues[69] = d69
			ps79.OverlayValues[70] = d70
			ps79.OverlayValues[71] = d71
			ps79.OverlayValues[72] = d72
			ps79.OverlayValues[73] = d73
			ps79.OverlayValues[74] = d74
			ps79.OverlayValues[75] = d75
			ps79.OverlayValues[76] = d76
			ps80 := scm.PhiState{General: true}
			ps80.OverlayValues = make([]scm.JITValueDesc, 77)
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
			ps80.OverlayValues[13] = d13
			ps80.OverlayValues[16] = d16
			ps80.OverlayValues[17] = d17
			ps80.OverlayValues[35] = d35
			ps80.OverlayValues[36] = d36
			ps80.OverlayValues[37] = d37
			ps80.OverlayValues[38] = d38
			ps80.OverlayValues[40] = d40
			ps80.OverlayValues[41] = d41
			ps80.OverlayValues[42] = d42
			ps80.OverlayValues[43] = d43
			ps80.OverlayValues[44] = d44
			ps80.OverlayValues[45] = d45
			ps80.OverlayValues[46] = d46
			ps80.OverlayValues[47] = d47
			ps80.OverlayValues[48] = d48
			ps80.OverlayValues[49] = d49
			ps80.OverlayValues[50] = d50
			ps80.OverlayValues[51] = d51
			ps80.OverlayValues[52] = d52
			ps80.OverlayValues[53] = d53
			ps80.OverlayValues[54] = d54
			ps80.OverlayValues[55] = d55
			ps80.OverlayValues[56] = d56
			ps80.OverlayValues[57] = d57
			ps80.OverlayValues[58] = d58
			ps80.OverlayValues[59] = d59
			ps80.OverlayValues[60] = d60
			ps80.OverlayValues[61] = d61
			ps80.OverlayValues[62] = d62
			ps80.OverlayValues[63] = d63
			ps80.OverlayValues[64] = d64
			ps80.OverlayValues[65] = d65
			ps80.OverlayValues[66] = d66
			ps80.OverlayValues[67] = d67
			ps80.OverlayValues[68] = d68
			ps80.OverlayValues[69] = d69
			ps80.OverlayValues[70] = d70
			ps80.OverlayValues[71] = d71
			ps80.OverlayValues[72] = d72
			ps80.OverlayValues[73] = d73
			ps80.OverlayValues[74] = d74
			ps80.OverlayValues[75] = d75
			ps80.OverlayValues[76] = d76
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
			snap92 := d13
			snap93 := d16
			snap94 := d17
			snap95 := d35
			snap96 := d36
			snap97 := d37
			snap98 := d38
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
			snap134 := d75
			snap135 := d76
			alloc136 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps80)
			}
			ctx.RestoreAllocState(alloc136)
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
			d13 = snap92
			d16 = snap93
			d17 = snap94
			d35 = snap95
			d36 = snap96
			d37 = snap97
			d38 = snap98
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
			d75 = snap134
			d76 = snap135
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps79)
			}
			return result
			ctx.FreeDesc(&d75)
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			ctx.ReclaimUntrackedRegs()
			var d137 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r59 := ctx.AllocReg()
				r60 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r59, fieldAddr)
				ctx.EmitMovRegMem64(r60, fieldAddr+8)
				d137 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r59, Reg2: r60}
				ctx.BindReg(r59, &d137)
				ctx.BindReg(r60, &d137)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r61 := ctx.AllocReg()
				r62 := ctx.AllocReg()
				ctx.EmitMovRegMem(r61, thisptr.Reg, off)
				ctx.EmitMovRegMem(r62, thisptr.Reg, off+8)
				d137 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r61, Reg2: r62}
				ctx.BindReg(r61, &d137)
				ctx.BindReg(r62, &d137)
			}
			ctx.EnsureDesc(&d37)
			r63 := ctx.AllocReg()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d137)
			if d37.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r63, uint64(d37.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r63, d37.Reg)
				ctx.EmitShlRegImm8(r63, 4)
			}
			if d137.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
				ctx.EmitAddInt64(r63, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r63, d137.Reg)
			}
			r64 := ctx.AllocRegExcept(r63)
			r65 := ctx.AllocRegExcept(r63, r64)
			ctx.EmitMovRegMem(r64, r63, 0)
			ctx.EmitMovRegMem(r65, r63, 8)
			ctx.FreeReg(r63)
			d138 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r64, Reg2: r65}
			ctx.BindReg(r64, &d138)
			ctx.BindReg(r65, &d138)
			d139 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d139)
			ctx.BindReg(r1, &d139)
			ctx.EnsureDesc(&d138)
			if d138.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d138, &d139)
			} else {
				switch d138.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(d139, d138)
				case scm.TagInt:
					ctx.EmitMakeInt(d139, d138)
				case scm.TagFloat:
					ctx.EmitMakeFloat(d139, d138)
				case scm.TagNil:
					ctx.EmitMakeNil(d139)
				default:
					ctx.EmitMovPairToResult(&d138, &d139)
				}
			}
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[5].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[5].VisitCount >= 0 {
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d140 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r66 := ctx.AllocReg()
				ctx.EmitMovRegReg(r66, idxInt.Reg)
				ctx.EmitShlRegImm8(r66, 32)
				ctx.EmitShrRegImm8(r66, 32)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d140)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d140)
			var d141 scm.JITValueDesc
			if d73.Loc == scm.LocImm && d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d73.Imm.Int()) < uint64(d140.Imm.Int()))}
			} else if d140.Loc == scm.LocImm {
				r67 := ctx.AllocRegExcept(d73.Reg)
				if d140.Imm.Int() >= -2147483648 && d140.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d73.Reg, int32(d140.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
					ctx.EmitCmpInt64(d73.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r67, scm.CcB)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r67}
				ctx.BindReg(r67, &d141)
			} else if d73.Loc == scm.LocImm {
				r68 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d140.Reg)
				ctx.EmitSetcc(r68, scm.CcB)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r68}
				ctx.BindReg(r68, &d141)
			} else {
				r69 := ctx.AllocRegExcept(d73.Reg)
				ctx.EmitCmpInt64(d73.Reg, d140.Reg)
				ctx.EmitSetcc(r69, scm.CcB)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r69}
				ctx.BindReg(r69, &d141)
			}
			ctx.FreeDesc(&d73)
			ctx.FreeDesc(&d140)
			d142 = d141
			ctx.EnsureDesc(&d142)
			if d142.Loc != scm.LocImm && d142.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d142.Loc == scm.LocImm {
				if d142.Imm.Bool() {
			ps143 := scm.PhiState{General: ps.General}
			ps143.OverlayValues = make([]scm.JITValueDesc, 143)
			ps143.OverlayValues[1] = d1
			ps143.OverlayValues[2] = d2
			ps143.OverlayValues[3] = d3
			ps143.OverlayValues[4] = d4
			ps143.OverlayValues[5] = d5
			ps143.OverlayValues[6] = d6
			ps143.OverlayValues[8] = d8
			ps143.OverlayValues[9] = d9
			ps143.OverlayValues[10] = d10
			ps143.OverlayValues[11] = d11
			ps143.OverlayValues[12] = d12
			ps143.OverlayValues[13] = d13
			ps143.OverlayValues[16] = d16
			ps143.OverlayValues[17] = d17
			ps143.OverlayValues[35] = d35
			ps143.OverlayValues[36] = d36
			ps143.OverlayValues[37] = d37
			ps143.OverlayValues[38] = d38
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
			ps143.OverlayValues[75] = d75
			ps143.OverlayValues[76] = d76
			ps143.OverlayValues[137] = d137
			ps143.OverlayValues[138] = d138
			ps143.OverlayValues[139] = d139
			ps143.OverlayValues[140] = d140
			ps143.OverlayValues[141] = d141
			ps143.OverlayValues[142] = d142
					return bbs[6].RenderPS(ps143)
				}
			ps144 := scm.PhiState{General: ps.General}
			ps144.OverlayValues = make([]scm.JITValueDesc, 143)
			ps144.OverlayValues[1] = d1
			ps144.OverlayValues[2] = d2
			ps144.OverlayValues[3] = d3
			ps144.OverlayValues[4] = d4
			ps144.OverlayValues[5] = d5
			ps144.OverlayValues[6] = d6
			ps144.OverlayValues[8] = d8
			ps144.OverlayValues[9] = d9
			ps144.OverlayValues[10] = d10
			ps144.OverlayValues[11] = d11
			ps144.OverlayValues[12] = d12
			ps144.OverlayValues[13] = d13
			ps144.OverlayValues[16] = d16
			ps144.OverlayValues[17] = d17
			ps144.OverlayValues[35] = d35
			ps144.OverlayValues[36] = d36
			ps144.OverlayValues[37] = d37
			ps144.OverlayValues[38] = d38
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
			ps144.OverlayValues[75] = d75
			ps144.OverlayValues[76] = d76
			ps144.OverlayValues[137] = d137
			ps144.OverlayValues[138] = d138
			ps144.OverlayValues[139] = d139
			ps144.OverlayValues[140] = d140
			ps144.OverlayValues[141] = d141
			ps144.OverlayValues[142] = d142
				return bbs[7].RenderPS(ps144)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl18 := ctx.ReserveLabel()
			lbl19 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d142.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl18)
			ctx.EmitJmp(lbl19)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl8)
			ps145 := scm.PhiState{General: true}
			ps145.OverlayValues = make([]scm.JITValueDesc, 143)
			ps145.OverlayValues[1] = d1
			ps145.OverlayValues[2] = d2
			ps145.OverlayValues[3] = d3
			ps145.OverlayValues[4] = d4
			ps145.OverlayValues[5] = d5
			ps145.OverlayValues[6] = d6
			ps145.OverlayValues[8] = d8
			ps145.OverlayValues[9] = d9
			ps145.OverlayValues[10] = d10
			ps145.OverlayValues[11] = d11
			ps145.OverlayValues[12] = d12
			ps145.OverlayValues[13] = d13
			ps145.OverlayValues[16] = d16
			ps145.OverlayValues[17] = d17
			ps145.OverlayValues[35] = d35
			ps145.OverlayValues[36] = d36
			ps145.OverlayValues[37] = d37
			ps145.OverlayValues[38] = d38
			ps145.OverlayValues[40] = d40
			ps145.OverlayValues[41] = d41
			ps145.OverlayValues[42] = d42
			ps145.OverlayValues[43] = d43
			ps145.OverlayValues[44] = d44
			ps145.OverlayValues[45] = d45
			ps145.OverlayValues[46] = d46
			ps145.OverlayValues[47] = d47
			ps145.OverlayValues[48] = d48
			ps145.OverlayValues[49] = d49
			ps145.OverlayValues[50] = d50
			ps145.OverlayValues[51] = d51
			ps145.OverlayValues[52] = d52
			ps145.OverlayValues[53] = d53
			ps145.OverlayValues[54] = d54
			ps145.OverlayValues[55] = d55
			ps145.OverlayValues[56] = d56
			ps145.OverlayValues[57] = d57
			ps145.OverlayValues[58] = d58
			ps145.OverlayValues[59] = d59
			ps145.OverlayValues[60] = d60
			ps145.OverlayValues[61] = d61
			ps145.OverlayValues[62] = d62
			ps145.OverlayValues[63] = d63
			ps145.OverlayValues[64] = d64
			ps145.OverlayValues[65] = d65
			ps145.OverlayValues[66] = d66
			ps145.OverlayValues[67] = d67
			ps145.OverlayValues[68] = d68
			ps145.OverlayValues[69] = d69
			ps145.OverlayValues[70] = d70
			ps145.OverlayValues[71] = d71
			ps145.OverlayValues[72] = d72
			ps145.OverlayValues[73] = d73
			ps145.OverlayValues[74] = d74
			ps145.OverlayValues[75] = d75
			ps145.OverlayValues[76] = d76
			ps145.OverlayValues[137] = d137
			ps145.OverlayValues[138] = d138
			ps145.OverlayValues[139] = d139
			ps145.OverlayValues[140] = d140
			ps145.OverlayValues[141] = d141
			ps145.OverlayValues[142] = d142
			ps146 := scm.PhiState{General: true}
			ps146.OverlayValues = make([]scm.JITValueDesc, 143)
			ps146.OverlayValues[1] = d1
			ps146.OverlayValues[2] = d2
			ps146.OverlayValues[3] = d3
			ps146.OverlayValues[4] = d4
			ps146.OverlayValues[5] = d5
			ps146.OverlayValues[6] = d6
			ps146.OverlayValues[8] = d8
			ps146.OverlayValues[9] = d9
			ps146.OverlayValues[10] = d10
			ps146.OverlayValues[11] = d11
			ps146.OverlayValues[12] = d12
			ps146.OverlayValues[13] = d13
			ps146.OverlayValues[16] = d16
			ps146.OverlayValues[17] = d17
			ps146.OverlayValues[35] = d35
			ps146.OverlayValues[36] = d36
			ps146.OverlayValues[37] = d37
			ps146.OverlayValues[38] = d38
			ps146.OverlayValues[40] = d40
			ps146.OverlayValues[41] = d41
			ps146.OverlayValues[42] = d42
			ps146.OverlayValues[43] = d43
			ps146.OverlayValues[44] = d44
			ps146.OverlayValues[45] = d45
			ps146.OverlayValues[46] = d46
			ps146.OverlayValues[47] = d47
			ps146.OverlayValues[48] = d48
			ps146.OverlayValues[49] = d49
			ps146.OverlayValues[50] = d50
			ps146.OverlayValues[51] = d51
			ps146.OverlayValues[52] = d52
			ps146.OverlayValues[53] = d53
			ps146.OverlayValues[54] = d54
			ps146.OverlayValues[55] = d55
			ps146.OverlayValues[56] = d56
			ps146.OverlayValues[57] = d57
			ps146.OverlayValues[58] = d58
			ps146.OverlayValues[59] = d59
			ps146.OverlayValues[60] = d60
			ps146.OverlayValues[61] = d61
			ps146.OverlayValues[62] = d62
			ps146.OverlayValues[63] = d63
			ps146.OverlayValues[64] = d64
			ps146.OverlayValues[65] = d65
			ps146.OverlayValues[66] = d66
			ps146.OverlayValues[67] = d67
			ps146.OverlayValues[68] = d68
			ps146.OverlayValues[69] = d69
			ps146.OverlayValues[70] = d70
			ps146.OverlayValues[71] = d71
			ps146.OverlayValues[72] = d72
			ps146.OverlayValues[73] = d73
			ps146.OverlayValues[74] = d74
			ps146.OverlayValues[75] = d75
			ps146.OverlayValues[76] = d76
			ps146.OverlayValues[137] = d137
			ps146.OverlayValues[138] = d138
			ps146.OverlayValues[139] = d139
			ps146.OverlayValues[140] = d140
			ps146.OverlayValues[141] = d141
			ps146.OverlayValues[142] = d142
			snap147 := d1
			snap148 := d2
			snap149 := d3
			snap150 := d4
			snap151 := d5
			snap152 := d6
			snap153 := d8
			snap154 := d9
			snap155 := d10
			snap156 := d11
			snap157 := d12
			snap158 := d13
			snap159 := d16
			snap160 := d17
			snap161 := d35
			snap162 := d36
			snap163 := d37
			snap164 := d38
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
			snap200 := d75
			snap201 := d76
			snap202 := d137
			snap203 := d138
			snap204 := d139
			snap205 := d140
			snap206 := d141
			snap207 := d142
			alloc208 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps146)
			}
			ctx.RestoreAllocState(alloc208)
			d1 = snap147
			d2 = snap148
			d3 = snap149
			d4 = snap150
			d5 = snap151
			d6 = snap152
			d8 = snap153
			d9 = snap154
			d10 = snap155
			d11 = snap156
			d12 = snap157
			d13 = snap158
			d16 = snap159
			d17 = snap160
			d35 = snap161
			d36 = snap162
			d37 = snap163
			d38 = snap164
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
			d75 = snap200
			d76 = snap201
			d137 = snap202
			d138 = snap203
			d139 = snap204
			d140 = snap205
			d141 = snap206
			d142 = snap207
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps145)
			}
			return result
			ctx.FreeDesc(&d141)
			return result
			}
			bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[6].VisitCount >= 0 {
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
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
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d209 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitMovRegReg(scratch, d37.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			}
			if d209.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: d209.Type, Imm: scm.NewInt(int64(uint64(d209.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d209.Reg, 32)
				ctx.EmitShrRegImm8(d209.Reg, 32)
			}
			if d209.Loc == scm.LocReg && d37.Loc == scm.LocReg && d209.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			ctx.EnsureDesc(&d209)
			if d209.Loc == scm.LocReg {
				ctx.ProtectReg(d209.Reg)
			} else if d209.Loc == scm.LocRegPair {
				ctx.ProtectReg(d209.Reg)
				ctx.ProtectReg(d209.Reg2)
			}
			d210 = d209
			if d210.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			d211 = d210
			if d211.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: d211.Type, Imm: scm.NewInt(int64(uint64(d211.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d211.Reg, 32)
				ctx.EmitShrRegImm8(d211.Reg, 32)
			}
			ctx.EmitStoreToStack(d211, int32(bbs[1].PhiBase)+int32(0))
			d212 = d2
			if d212.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d212)
			d213 = d212
			if d213.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: d213.Type, Imm: scm.NewInt(int64(uint64(d213.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d213.Reg, 32)
				ctx.EmitShrRegImm8(d213.Reg, 32)
			}
			ctx.EmitStoreToStack(d213, int32(bbs[1].PhiBase)+int32(16))
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
			if d209.Loc == scm.LocReg {
				ctx.UnprotectReg(d209.Reg)
			} else if d209.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d209.Reg)
				ctx.UnprotectReg(d209.Reg2)
			}
			ps214 := scm.PhiState{General: ps.General}
			ps214.OverlayValues = make([]scm.JITValueDesc, 214)
			ps214.OverlayValues[1] = d1
			ps214.OverlayValues[2] = d2
			ps214.OverlayValues[3] = d3
			ps214.OverlayValues[4] = d4
			ps214.OverlayValues[5] = d5
			ps214.OverlayValues[6] = d6
			ps214.OverlayValues[8] = d8
			ps214.OverlayValues[9] = d9
			ps214.OverlayValues[10] = d10
			ps214.OverlayValues[11] = d11
			ps214.OverlayValues[12] = d12
			ps214.OverlayValues[13] = d13
			ps214.OverlayValues[16] = d16
			ps214.OverlayValues[17] = d17
			ps214.OverlayValues[35] = d35
			ps214.OverlayValues[36] = d36
			ps214.OverlayValues[37] = d37
			ps214.OverlayValues[38] = d38
			ps214.OverlayValues[40] = d40
			ps214.OverlayValues[41] = d41
			ps214.OverlayValues[42] = d42
			ps214.OverlayValues[43] = d43
			ps214.OverlayValues[44] = d44
			ps214.OverlayValues[45] = d45
			ps214.OverlayValues[46] = d46
			ps214.OverlayValues[47] = d47
			ps214.OverlayValues[48] = d48
			ps214.OverlayValues[49] = d49
			ps214.OverlayValues[50] = d50
			ps214.OverlayValues[51] = d51
			ps214.OverlayValues[52] = d52
			ps214.OverlayValues[53] = d53
			ps214.OverlayValues[54] = d54
			ps214.OverlayValues[55] = d55
			ps214.OverlayValues[56] = d56
			ps214.OverlayValues[57] = d57
			ps214.OverlayValues[58] = d58
			ps214.OverlayValues[59] = d59
			ps214.OverlayValues[60] = d60
			ps214.OverlayValues[61] = d61
			ps214.OverlayValues[62] = d62
			ps214.OverlayValues[63] = d63
			ps214.OverlayValues[64] = d64
			ps214.OverlayValues[65] = d65
			ps214.OverlayValues[66] = d66
			ps214.OverlayValues[67] = d67
			ps214.OverlayValues[68] = d68
			ps214.OverlayValues[69] = d69
			ps214.OverlayValues[70] = d70
			ps214.OverlayValues[71] = d71
			ps214.OverlayValues[72] = d72
			ps214.OverlayValues[73] = d73
			ps214.OverlayValues[74] = d74
			ps214.OverlayValues[75] = d75
			ps214.OverlayValues[76] = d76
			ps214.OverlayValues[137] = d137
			ps214.OverlayValues[138] = d138
			ps214.OverlayValues[139] = d139
			ps214.OverlayValues[140] = d140
			ps214.OverlayValues[141] = d141
			ps214.OverlayValues[142] = d142
			ps214.OverlayValues[209] = d209
			ps214.OverlayValues[210] = d210
			ps214.OverlayValues[211] = d211
			ps214.OverlayValues[212] = d212
			ps214.OverlayValues[213] = d213
			ps214.PhiValues = make([]scm.JITValueDesc, 2)
			d215 = d209
			ps214.PhiValues[0] = d215
			d216 = d2
			ps214.PhiValues[1] = d216
			if ps214.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps214)
			return result
			}
			bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[7].VisitCount >= 0 {
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
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
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
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
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
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
			if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
				d141 = ps.OverlayValues[141]
			}
			if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
				d142 = ps.OverlayValues[142]
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
			if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != scm.LocNone {
				d212 = ps.OverlayValues[212]
			}
			if len(ps.OverlayValues) > 213 && ps.OverlayValues[213].Loc != scm.LocNone {
				d213 = ps.OverlayValues[213]
			}
			if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
				d215 = ps.OverlayValues[215]
			}
			if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
				d216 = ps.OverlayValues[216]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d1)
			if d1.Loc == scm.LocReg {
				ctx.ProtectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1.Reg)
				ctx.ProtectReg(d1.Reg2)
			}
			ctx.EnsureDesc(&d37)
			if d37.Loc == scm.LocReg {
				ctx.ProtectReg(d37.Reg)
			} else if d37.Loc == scm.LocRegPair {
				ctx.ProtectReg(d37.Reg)
				ctx.ProtectReg(d37.Reg2)
			}
			d217 = d1
			if d217.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d217)
			d218 = d217
			if d218.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: d218.Type, Imm: scm.NewInt(int64(uint64(d218.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d218.Reg, 32)
				ctx.EmitShrRegImm8(d218.Reg, 32)
			}
			ctx.EmitStoreToStack(d218, int32(bbs[1].PhiBase)+int32(0))
			d219 = d37
			if d219.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d219)
			d220 = d219
			if d220.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: d220.Type, Imm: scm.NewInt(int64(uint64(d220.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d220.Reg, 32)
				ctx.EmitShrRegImm8(d220.Reg, 32)
			}
			ctx.EmitStoreToStack(d220, int32(bbs[1].PhiBase)+int32(16))
			if d1.Loc == scm.LocReg {
				ctx.UnprotectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1.Reg)
				ctx.UnprotectReg(d1.Reg2)
			}
			if d37.Loc == scm.LocReg {
				ctx.UnprotectReg(d37.Reg)
			} else if d37.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d37.Reg)
				ctx.UnprotectReg(d37.Reg2)
			}
			ps221 := scm.PhiState{General: ps.General}
			ps221.OverlayValues = make([]scm.JITValueDesc, 221)
			ps221.OverlayValues[1] = d1
			ps221.OverlayValues[2] = d2
			ps221.OverlayValues[3] = d3
			ps221.OverlayValues[4] = d4
			ps221.OverlayValues[5] = d5
			ps221.OverlayValues[6] = d6
			ps221.OverlayValues[8] = d8
			ps221.OverlayValues[9] = d9
			ps221.OverlayValues[10] = d10
			ps221.OverlayValues[11] = d11
			ps221.OverlayValues[12] = d12
			ps221.OverlayValues[13] = d13
			ps221.OverlayValues[16] = d16
			ps221.OverlayValues[17] = d17
			ps221.OverlayValues[35] = d35
			ps221.OverlayValues[36] = d36
			ps221.OverlayValues[37] = d37
			ps221.OverlayValues[38] = d38
			ps221.OverlayValues[40] = d40
			ps221.OverlayValues[41] = d41
			ps221.OverlayValues[42] = d42
			ps221.OverlayValues[43] = d43
			ps221.OverlayValues[44] = d44
			ps221.OverlayValues[45] = d45
			ps221.OverlayValues[46] = d46
			ps221.OverlayValues[47] = d47
			ps221.OverlayValues[48] = d48
			ps221.OverlayValues[49] = d49
			ps221.OverlayValues[50] = d50
			ps221.OverlayValues[51] = d51
			ps221.OverlayValues[52] = d52
			ps221.OverlayValues[53] = d53
			ps221.OverlayValues[54] = d54
			ps221.OverlayValues[55] = d55
			ps221.OverlayValues[56] = d56
			ps221.OverlayValues[57] = d57
			ps221.OverlayValues[58] = d58
			ps221.OverlayValues[59] = d59
			ps221.OverlayValues[60] = d60
			ps221.OverlayValues[61] = d61
			ps221.OverlayValues[62] = d62
			ps221.OverlayValues[63] = d63
			ps221.OverlayValues[64] = d64
			ps221.OverlayValues[65] = d65
			ps221.OverlayValues[66] = d66
			ps221.OverlayValues[67] = d67
			ps221.OverlayValues[68] = d68
			ps221.OverlayValues[69] = d69
			ps221.OverlayValues[70] = d70
			ps221.OverlayValues[71] = d71
			ps221.OverlayValues[72] = d72
			ps221.OverlayValues[73] = d73
			ps221.OverlayValues[74] = d74
			ps221.OverlayValues[75] = d75
			ps221.OverlayValues[76] = d76
			ps221.OverlayValues[137] = d137
			ps221.OverlayValues[138] = d138
			ps221.OverlayValues[139] = d139
			ps221.OverlayValues[140] = d140
			ps221.OverlayValues[141] = d141
			ps221.OverlayValues[142] = d142
			ps221.OverlayValues[209] = d209
			ps221.OverlayValues[210] = d210
			ps221.OverlayValues[211] = d211
			ps221.OverlayValues[212] = d212
			ps221.OverlayValues[213] = d213
			ps221.OverlayValues[215] = d215
			ps221.OverlayValues[216] = d216
			ps221.OverlayValues[217] = d217
			ps221.OverlayValues[218] = d218
			ps221.OverlayValues[219] = d219
			ps221.OverlayValues[220] = d220
			ps221.PhiValues = make([]scm.JITValueDesc, 2)
			d222 = d1
			ps221.PhiValues[0] = d222
			d223 = d37
			ps221.PhiValues[1] = d223
			if ps221.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps221)
			return result
			}
			ps224 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps224)
			ctx.MarkLabel(lbl0)
			d225 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d225)
			ctx.BindReg(r1, &d225)
			ctx.EmitMovPairToResult(&d225, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.FreeStack(int32(32))
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

// sparseSeek returns the smallest pivot in [0,s.i) whose recid is >= want,
// via binary search. Used once to seed the forward merge-scan below instead
// of a fresh binary search per requested row.
func (s *StorageSparse) sparseSeek(want uint32) uint32 {
	var lower, upper uint32 = 0, uint32(s.i)
	for lower < upper {
		pivot := (lower + upper) / 2
		recid := uint32(s.recids.GetValueUInt(pivot)) + uint32(s.recids.offset)
		if recid < want {
			lower = pivot + 1
		} else {
			upper = pivot
		}
	}
	return lower
}

// GetValueRange and GetValueMulti (ascending case) do a single binary search
// to seed a pointer into the sparse recids array, then merge-scan it forward
// against the requested rows in one pass — O(touched sparse entries + n)
// instead of a binary search per requested row.
func (s *StorageSparse) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	n := uint32(s.i)
	sp := s.sparseSeek(recid)
	var curRecid uint32
	haveCur := false
	idx := 0
	for k := uint32(0); k < count; k++ {
		want := recid + k
		for {
			if !haveCur {
				if sp >= n {
					break
				}
				curRecid = uint32(s.recids.GetValueUInt(sp)) + uint32(s.recids.offset)
				haveCur = true
			}
			if curRecid < want {
				sp++
				haveCur = false
				continue
			}
			break
		}
		if haveCur && curRecid == want {
			target[idx] = s.values[sp]
		} else {
			target[idx] = scm.NewNil()
		}
		idx += stride
	}
}

func (s *StorageSparse) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	n := len(recids)
	if n == 0 {
		return
	}
	ascending := true
	for k := 1; k < n; k++ {
		if recids[k] < recids[k-1] {
			ascending = false
			break
		}
	}
	if !ascending {
		idx := 0
		for _, want := range recids {
			target[idx] = s.GetValue(want)
			idx += stride
		}
		return
	}

	total := uint32(s.i)
	sp := s.sparseSeek(recids[0])
	var curRecid uint32
	haveCur := false
	idx := 0
	for _, want := range recids {
		for {
			if !haveCur {
				if sp >= total {
					break
				}
				curRecid = uint32(s.recids.GetValueUInt(sp)) + uint32(s.recids.offset)
				haveCur = true
			}
			if curRecid < want {
				sp++
				haveCur = false
				continue
			}
			break
		}
		if haveCur && curRecid == want {
			target[idx] = s.values[sp]
		} else {
			target[idx] = scm.NewNil()
		}
		idx += stride
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

func (s *StorageSparse) DistinctCount() uint {
	return uint(len(s.values)) + 1 // +1 for nil values
}
