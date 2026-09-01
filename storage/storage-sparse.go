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
	var d39 scm.JITValueDesc
	_ = d39
	var phiBase40 int32
	_ = phiBase40
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
	var d143 scm.JITValueDesc
	_ = d143
	var d211 scm.JITValueDesc
	_ = d211
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
	phiBase0 := ctx.AllocStack(int32(32))
	d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
	_ = d1
	d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
	_ = d2
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
		ctx.StabilizeDescForControlFlow(&d4)
		if ps.General {
			ctx.SyncDesc(&d4)
			if d4.Loc == scm.LocReg {
				ctx.ProtectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.ProtectReg(d4.Reg)
				ctx.ProtectReg(d4.Reg2)
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
			d5 = d4
			if d5.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
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
		ctx.StabilizeDescForControlFlow(&d1)
		ctx.StabilizeDescForControlFlow(&d2)
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
			ctx.EmitSetcc(r5, scm.CondEqual)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r5}
			ctx.BindReg(r5, &d12)
		} else if d1.Loc == scm.LocImm {
			r6 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d2.Reg)
			ctx.EmitSetcc(r6, scm.CondEqual)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
			ctx.BindReg(r6, &d12)
		} else {
			r7 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpInt64(d1.Reg, d2.Reg)
			ctx.EmitSetcc(r7, scm.CondEqual)
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
				if ps.General {
				}
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
			if ps.General {
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
		ctx.EmitJump(scm.CondNotEqual, lbl9)
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
		d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d36 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d36)
		ctx.BindReg(r1, &d36)
		ctx.EnsureDesc(&d35)
		if d35.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d35, &d36)
		} else {
			switch d35.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d36, d35)
			case scm.TagInt:
				ctx.EmitMakeInt(d36, d35)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d36, d35)
			case scm.TagNil:
				ctx.EmitMakeNil(d36)
			default:
				ctx.EmitMovPairToResult(&d35, &d36)
			}
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d1)
		ctx.ProtectReg(d1.Reg)
		ctx.EnsureDesc(&d2)
		ctx.UnprotectReg(d1.Reg)
		var d37 scm.JITValueDesc
		if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + d2.Imm.Int())}
		} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
			r8 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(r8, d1.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d37)
		} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d2.Reg}
			ctx.BindReg(d2.Reg, &d37)
		} else if d1.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
			ctx.EmitAddInt64(scratch, d2.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else if d2.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else {
			r9 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
			ctx.EmitMovRegReg(r9, d1.Reg)
			ctx.EmitAddInt64(r9, d2.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d37)
		}
		if d37.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: d37.Type, Imm: scm.NewInt(int64(uint64(d37.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d37.Reg, 32)
			ctx.EmitShrRegImm8(d37.Reg, 32)
		}
		if d37.Loc == scm.LocReg && d1.Loc == scm.LocReg && d37.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d37)
		var d38 scm.JITValueDesc
		if d37.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() / 2)}
		} else {
			r10 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(r10, d37.Reg)
			ctx.EmitShrRegImm8(r10, 1)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d38)
		}
		if d38.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: d38.Type, Imm: scm.NewInt(int64(uint64(d38.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d38.Reg, 32)
			ctx.EmitShrRegImm8(d38.Reg, 32)
		}
		if d38.Loc == scm.LocReg && d37.Loc == scm.LocReg && d38.Reg == d37.Reg {
			ctx.TransferReg(d37.Reg)
			d37.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d38)
		ctx.EmitStoreToStack(d38, int32(bbs[1].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d38)
		ctx.FreeDesc(&d37)
		ctx.EnsureDesc(&d38)
		d39 = d38
		_ = d39
		ctx.StabilizeDescForControlFlow(&d39)
		r11 := d38.Loc == scm.LocReg || d38.Loc == scm.LocRegPair || d38.Loc == scm.LocRegTriple
		r12 := d38.Reg
		if r11 {
			ctx.ProtectReg(r12)
		}
		r13 := d38.Loc == scm.LocRegPair || d38.Loc == scm.LocRegTriple
		r14 := d38.Reg2
		if r13 {
			ctx.ProtectReg(r14)
		}
		r15 := d38.Loc == scm.LocRegTriple
		r16 := d38.Reg3
		if r15 {
			ctx.ProtectReg(r16)
		}
		phiBase40 = ctx.AllocStack(int32(16))
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase40) + int32(0)}
		_ = d41
		lbl11 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d39)
		var d42 scm.JITValueDesc
		if d39.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d39.Imm.Int()))))}
		} else {
			r17 := ctx.AllocReg()
			ctx.EmitMovRegReg(r17, d39.Reg)
			ctx.EmitShlRegImm8(r17, 32)
			ctx.EmitShrRegImm8(r17, 32)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d42)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d43 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
			r18 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r18, fieldAddr)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
			ctx.BindReg(r18, &d43)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
			r19 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r19, thisptr.Reg, off)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
			ctx.BindReg(r19, &d43)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d43.Imm.Int()))))}
		} else {
			r20 := ctx.AllocReg()
			ctx.EmitMovRegReg(r20, d43.Reg)
			ctx.EmitShlRegImm8(r20, 56)
			ctx.EmitShrRegImm8(r20, 56)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d44)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d42)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d42)
		ctx.ProtectReg(d42.Reg)
		ctx.EnsureDesc(&d44)
		ctx.UnprotectReg(d42.Reg)
		var d45 scm.JITValueDesc
		if d42.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() * d44.Imm.Int())}
		} else if d42.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d42.Imm.Int()))
			ctx.EmitImulInt64(scratch, d44.Reg)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d45)
		} else if d44.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegReg(scratch, d42.Reg)
			if d44.Imm.Int() >= -2147483648 && d44.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d44.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d45)
		} else {
			r21 := ctx.AllocRegExcept(d42.Reg, d44.Reg)
			ctx.EmitMovRegReg(r21, d42.Reg)
			ctx.EmitImulInt64(r21, d44.Reg)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d45)
		}
		if d45.Loc == scm.LocReg && d42.Loc == scm.LocReg && d45.Reg == d42.Reg {
			ctx.TransferReg(d42.Reg)
			d42.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d45)
		ctx.FreeDesc(&d42)
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d46 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
			r22 := ctx.AllocReg()
			r23 := ctx.AllocRegExcept(r22)
			r24 := ctx.AllocRegExcept(r22, r23)
			ctx.EmitMovRegMem64(r22, fieldAddr)
			ctx.EmitMovRegMem64(r23, fieldAddr+8)
			ctx.EmitMovRegMem64(r24, fieldAddr+16)
			d46 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r22, Reg2: r23, Reg3: r24}
			ctx.BindReg(r22, &d46)
			ctx.BindReg(r23, &d46)
			ctx.BindReg(r24, &d46)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
			r25 := ctx.AllocReg()
			r26 := ctx.AllocRegExcept(r25)
			r27 := ctx.AllocRegExcept(r25, r26)
			ctx.EmitMovRegMem(r25, thisptr.Reg, off)
			ctx.EmitMovRegMem(r26, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r27, thisptr.Reg, off+16)
			d46 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r25, Reg2: r26, Reg3: r27}
			ctx.BindReg(r25, &d46)
			ctx.BindReg(r26, &d46)
			ctx.BindReg(r27, &d46)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d47 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() / 64)}
		} else {
			r28 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r28, d45.Reg)
			ctx.EmitShrRegImm8(r28, 6)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d47)
		}
		if d47.Loc == scm.LocReg && d45.Loc == scm.LocReg && d47.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		d49 = ctx.EmitSliceElementAddress(&d46, &d47, 8)
		ctx.EnsureDesc(&d49)
		ctx.EmitMovRegMem(d49.Reg, d49.Reg, 0)
		d48 = d49
		ctx.FreeDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d50 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() % 64)}
		} else {
			r29 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r29, d45.Reg)
			ctx.EmitAndRegImm32(r29, 63)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d50)
		}
		if d50.Loc == scm.LocReg && d45.Loc == scm.LocReg && d50.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d48)
		ctx.EnsureDesc(&d50)
		var d51 scm.JITValueDesc
		if d48.Loc == scm.LocImm && d50.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) << uint64(d50.Imm.Int())))}
		} else if d50.Loc == scm.LocImm {
			r30 := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegReg(r30, d48.Reg)
			ctx.EmitShlRegImm8(r30, uint8(d50.Imm.Int()))
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d51)
		} else {
			{
				shiftSrc := d48.Reg
				r31 := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(r31, d48.Reg)
				shiftSrc = r31
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d50.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d50.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d50.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d51)
			}
		}
		if d51.Loc == scm.LocReg && d48.Loc == scm.LocReg && d51.Reg == d48.Reg {
			ctx.TransferReg(d48.Reg)
			d48.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d51)
		ctx.EmitStoreToStack(d51, int32(phiBase40)+int32(0))
		ctx.StabilizeDescForControlFlow(&d51)
		ctx.FreeDesc(&d48)
		ctx.FreeDesc(&d50)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d52 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() % 64)}
		} else {
			r32 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r32, d45.Reg)
			ctx.EmitAndRegImm32(r32, 63)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d52)
		}
		if d52.Loc == scm.LocReg && d45.Loc == scm.LocReg && d52.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		var d53 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d43.Imm.Int()))))}
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitMovRegReg(r33, d43.Reg)
			ctx.EmitShlRegImm8(r33, 56)
			ctx.EmitShrRegImm8(r33, 56)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d53)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d52)
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d52)
		ctx.ProtectReg(d52.Reg)
		ctx.EnsureDesc(&d53)
		ctx.UnprotectReg(d52.Reg)
		var d54 scm.JITValueDesc
		if d52.Loc == scm.LocImm && d53.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d53.Imm.Int())}
		} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegReg(r34, d52.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d54)
		} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d53.Reg}
			ctx.BindReg(d53.Reg, &d54)
		} else if d52.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
			ctx.EmitAddInt64(scratch, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d54)
		} else if d53.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegReg(scratch, d52.Reg)
			if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d53.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d54)
		} else {
			r35 := ctx.AllocRegExcept(d52.Reg, d53.Reg)
			ctx.EmitMovRegReg(r35, d52.Reg)
			ctx.EmitAddInt64(r35, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d54)
		}
		if d54.Loc == scm.LocReg && d52.Loc == scm.LocReg && d54.Reg == d52.Reg {
			ctx.TransferReg(d52.Reg)
			d52.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d52)
		ctx.FreeDesc(&d53)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d54)
		var d55 scm.JITValueDesc
		if d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d54.Imm.Int()) > uint64(0x40))}
		} else {
			r36 := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitCmpRegImm32(d54.Reg, 64)
			ctx.EmitSetcc(r36, scm.CondUnsignedAbove)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r36}
			ctx.BindReg(r36, &d55)
		}
		ctx.FreeDesc(&d54)
		ctx.ReclaimUntrackedRegs()
		d56 = d55
		ctx.EnsureDesc(&d56)
		if d56.Loc != scm.LocImm && d56.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		if d56.Loc == scm.LocImm {
			if d56.Imm.Bool() {
				ctx.MarkLabel(lbl14)
				ctx.EmitJmp(lbl12)
			} else {
				ctx.MarkLabel(lbl15)
				ctx.EmitJmp(lbl13)
			}
		} else {
			ctx.EmitCmpRegImm32(d56.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl14)
			ctx.EmitJmp(lbl15)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl12)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d55)
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl13)
		ctx.ResolveFixups()
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		var d57 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d43.Imm.Int()))))}
		} else {
			r37 := ctx.AllocReg()
			ctx.EmitMovRegReg(r37, d43.Reg)
			ctx.EmitShlRegImm8(r37, 56)
			ctx.EmitShrRegImm8(r37, 56)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d57)
		}
		ctx.ReclaimUntrackedRegs()
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
			r38 := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegReg(r38, d58.Reg)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d59)
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
			r39 := ctx.AllocRegExcept(d58.Reg, d57.Reg)
			ctx.EmitMovRegReg(r39, d58.Reg)
			ctx.EmitSubInt64(r39, d57.Reg)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d59)
		}
		if d59.Loc == scm.LocReg && d58.Loc == scm.LocReg && d59.Reg == d58.Reg {
			ctx.TransferReg(d58.Reg)
			d58.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d57)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d59)
		var d60 scm.JITValueDesc
		if d41.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d41.Imm.Int()) >> uint64(d59.Imm.Int())))}
		} else if d59.Loc == scm.LocImm {
			r40 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r40, d41.Reg)
			ctx.EmitShrRegImm8(r40, uint8(d59.Imm.Int()))
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d60)
		} else {
			{
				shiftSrc := d41.Reg
				r41 := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(r41, d41.Reg)
				shiftSrc = r41
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d59.Reg != scm.RegRCX
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
		if d60.Loc == scm.LocReg && d41.Loc == scm.LocReg && d60.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d41)
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		r42 := ctx.AllocReg()
		ctx.EnsureDesc(&d60)
		ctx.EnsureDesc(&d60)
		if d60.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r42, d60)
		}
		ctx.EmitJmp(lbl11)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl12)
		ctx.ResolveFixups()
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d61 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() / 64)}
		} else {
			r43 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r43, d45.Reg)
			ctx.EmitShrRegImm8(r43, 6)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d61)
		}
		if d61.Loc == scm.LocReg && d45.Loc == scm.LocReg && d61.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d62)
		ctx.ReclaimUntrackedRegs()
		d64 = ctx.EmitSliceElementAddress(&d46, &d62, 8)
		ctx.EnsureDesc(&d64)
		ctx.EmitMovRegMem(d64.Reg, d64.Reg, 0)
		d63 = d64
		ctx.FreeDesc(&d62)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d65 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() % 64)}
		} else {
			r44 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r44, d45.Reg)
			ctx.EmitAndRegImm32(r44, 63)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d65)
		}
		if d65.Loc == scm.LocReg && d45.Loc == scm.LocReg && d65.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d65)
		ctx.EnsureDesc(&d66)
		ctx.ProtectReg(d66.Reg)
		ctx.EnsureDesc(&d65)
		ctx.UnprotectReg(d66.Reg)
		var d67 scm.JITValueDesc
		if d66.Loc == scm.LocImm && d65.Loc == scm.LocImm {
			d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() - d65.Imm.Int())}
		} else if d65.Loc == scm.LocImm && d65.Imm.Int() == 0 {
			r45 := ctx.AllocRegExcept(d66.Reg)
			ctx.EmitMovRegReg(r45, d66.Reg)
			d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d67)
		} else if d66.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d65.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d66.Imm.Int()))
			ctx.EmitSubInt64(scratch, d65.Reg)
			d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d67)
		} else if d65.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d66.Reg)
			ctx.EmitMovRegReg(scratch, d66.Reg)
			if d65.Imm.Int() >= -2147483648 && d65.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d65.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d65.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d67)
		} else {
			r46 := ctx.AllocRegExcept(d66.Reg, d65.Reg)
			ctx.EmitMovRegReg(r46, d66.Reg)
			ctx.EmitSubInt64(r46, d65.Reg)
			d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d67)
		}
		if d67.Loc == scm.LocReg && d66.Loc == scm.LocReg && d67.Reg == d66.Reg {
			ctx.TransferReg(d66.Reg)
			d66.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d65)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d67)
		var d68 scm.JITValueDesc
		if d63.Loc == scm.LocImm && d67.Loc == scm.LocImm {
			d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d63.Imm.Int()) >> uint64(d67.Imm.Int())))}
		} else if d67.Loc == scm.LocImm {
			r47 := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitMovRegReg(r47, d63.Reg)
			ctx.EmitShrRegImm8(r47, uint8(d67.Imm.Int()))
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d68)
		} else {
			{
				shiftSrc := d63.Reg
				r48 := ctx.AllocRegExcept(d63.Reg)
				ctx.EmitMovRegReg(r48, d63.Reg)
				shiftSrc = r48
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d67.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d67.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d67.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d68)
			}
		}
		if d68.Loc == scm.LocReg && d63.Loc == scm.LocReg && d68.Reg == d63.Reg {
			ctx.TransferReg(d63.Reg)
			d63.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d63)
		ctx.FreeDesc(&d67)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d68)
		var d69 scm.JITValueDesc
		if d51.Loc == scm.LocImm && d68.Loc == scm.LocImm {
			d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() | d68.Imm.Int())}
		} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d68.Reg}
			ctx.BindReg(d68.Reg, &d69)
		} else if d68.Loc == scm.LocImm && d68.Imm.Int() == 0 {
			r49 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r49, d51.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d69)
		} else if d51.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d68.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d51.Imm.Int()))
			ctx.EmitOrInt64(scratch, d68.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d69)
		} else if d68.Loc == scm.LocImm {
			r50 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r50, d51.Reg)
			if d68.Imm.Int() >= -2147483648 && d68.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r50, int32(d68.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d68.Imm.Int()))
				ctx.EmitOrInt64(r50, scm.RegR11)
			}
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d69)
		} else {
			r51 := ctx.AllocRegExcept(d51.Reg, d68.Reg)
			ctx.EmitMovRegReg(r51, d51.Reg)
			ctx.EmitOrInt64(r51, d68.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d69)
		}
		if d69.Loc == scm.LocReg && d51.Loc == scm.LocReg && d69.Reg == d51.Reg {
			ctx.TransferReg(d51.Reg)
			d51.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d69)
		ctx.EmitStoreToStack(d69, int32(phiBase40)+int32(0))
		ctx.StabilizeDescForControlFlow(&d69)
		ctx.FreeDesc(&d68)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl13)
		ctx.MarkLabel(lbl11)
		d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
		ctx.BindReg(r42, &d70)
		ctx.BindReg(r42, &d70)
		if r11 {
			ctx.UnprotectReg(r12)
		}
		if r13 {
			ctx.UnprotectReg(r14)
		}
		if r15 {
			ctx.UnprotectReg(r16)
		}
		var d71 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
			r52 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r52, fieldAddr)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
			ctx.BindReg(r52, &d71)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
			r53 := ctx.AllocReg()
			ctx.EmitMovRegMem(r53, thisptr.Reg, off)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
			ctx.BindReg(r53, &d71)
		}
		ctx.EnsureDesc(&d71)
		ctx.EnsureDesc(&d71)
		var d72 scm.JITValueDesc
		if d71.Loc == scm.LocImm {
			d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d71.Imm.Int()))))}
		} else {
			r54 := ctx.AllocReg()
			ctx.EmitMovRegReg(r54, d71.Reg)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d72)
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
			r55 := ctx.AllocRegExcept(d70.Reg)
			ctx.EmitMovRegReg(r55, d70.Reg)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d73)
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
			r56 := ctx.AllocRegExcept(d70.Reg, d72.Reg)
			ctx.EmitMovRegReg(r56, d70.Reg)
			ctx.EmitAddInt64(r56, d72.Reg)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d73)
		}
		if d73.Loc == scm.LocReg && d70.Loc == scm.LocReg && d73.Reg == d70.Reg {
			ctx.TransferReg(d70.Reg)
			d70.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d73)
		ctx.FreeDesc(&d70)
		ctx.FreeDesc(&d72)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d74 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r57 := ctx.AllocReg()
			ctx.EmitMovRegReg(r57, idxInt.Reg)
			ctx.EmitShlRegImm8(r57, 32)
			ctx.EmitShrRegImm8(r57, 32)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d74)
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
			r58 := ctx.AllocRegExcept(d73.Reg)
			if d74.Imm.Int() >= -2147483648 && d74.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d73.Reg, int32(d74.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
				ctx.EmitCmpInt64(d73.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r58, scm.CondEqual)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
			ctx.BindReg(r58, &d75)
		} else if d73.Loc == scm.LocImm {
			r59 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d74.Reg)
			ctx.EmitSetcc(r59, scm.CondEqual)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
			ctx.BindReg(r59, &d75)
		} else {
			r60 := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitCmpInt64(d73.Reg, d74.Reg)
			ctx.EmitSetcc(r60, scm.CondEqual)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
			ctx.BindReg(r60, &d75)
		}
		ctx.FreeDesc(&d74)
		d76 = d75
		ctx.EnsureDesc(&d76)
		if d76.Loc != scm.LocImm && d76.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d76.Loc == scm.LocImm {
			if d76.Imm.Bool() {
				if ps.General {
				}
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
				ps77.OverlayValues[39] = d39
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
			if ps.General {
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
			ps78.OverlayValues[39] = d39
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
		ctx.EmitJump(scm.CondNotEqual, lbl16)
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
		ps79.OverlayValues[39] = d39
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
		ps80.OverlayValues[39] = d39
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
		snap99 := d39
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
		d39 = snap99
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
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
			r61 := ctx.AllocReg()
			r62 := ctx.AllocRegExcept(r61)
			r63 := ctx.AllocRegExcept(r61, r62)
			ctx.EmitMovRegMem64(r61, fieldAddr)
			ctx.EmitMovRegMem64(r62, fieldAddr+8)
			ctx.EmitMovRegMem64(r63, fieldAddr+16)
			d137 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r61, Reg2: r62, Reg3: r63}
			ctx.BindReg(r61, &d137)
			ctx.BindReg(r62, &d137)
			ctx.BindReg(r63, &d137)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
			r64 := ctx.AllocReg()
			r65 := ctx.AllocRegExcept(r64)
			r66 := ctx.AllocRegExcept(r64, r65)
			ctx.EmitMovRegMem(r64, thisptr.Reg, off)
			ctx.EmitMovRegMem(r65, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r66, thisptr.Reg, off+16)
			d137 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r64, Reg2: r65, Reg3: r66}
			ctx.BindReg(r64, &d137)
			ctx.BindReg(r65, &d137)
			ctx.BindReg(r66, &d137)
		}
		ctx.EnsureDesc(&d38)
		d139 = ctx.EmitSliceElementAddress(&d137, &d38, 16)
		ctx.EnsureDesc(&d139)
		r67 := ctx.AllocRegExcept(d139.Reg)
		ctx.EmitMovRegMem(r67, d139.Reg, 8)
		ctx.EmitMovRegMem(d139.Reg, d139.Reg, 0)
		d138 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d139.Reg, Reg2: r67}
		ctx.BindReg(d139.Reg, &d138)
		ctx.BindReg(r67, &d138)
		d140 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d140)
		ctx.BindReg(r1, &d140)
		ctx.EnsureDesc(&d138)
		if d138.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d138, &d140)
		} else {
			switch d138.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d140, d138)
			case scm.TagInt:
				ctx.EmitMakeInt(d140, d138)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d140, d138)
			case scm.TagNil:
				ctx.EmitMakeNil(d140)
			default:
				ctx.EmitMovPairToResult(&d138, &d140)
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d141 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r68 := ctx.AllocReg()
			ctx.EmitMovRegReg(r68, idxInt.Reg)
			ctx.EmitShlRegImm8(r68, 32)
			ctx.EmitShrRegImm8(r68, 32)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d141)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d73)
		ctx.EnsureDesc(&d141)
		ctx.EnsureDesc(&d73)
		ctx.EnsureDesc(&d141)
		ctx.EnsureDesc(&d73)
		ctx.EnsureDesc(&d141)
		var d142 scm.JITValueDesc
		if d73.Loc == scm.LocImm && d141.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d73.Imm.Int()) < uint64(d141.Imm.Int()))}
		} else if d141.Loc == scm.LocImm {
			r69 := ctx.AllocRegExcept(d73.Reg)
			if d141.Imm.Int() >= -2147483648 && d141.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d73.Reg, int32(d141.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d141.Imm.Int()))
				ctx.EmitCmpInt64(d73.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r69, scm.CondUnsignedBelow)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r69}
			ctx.BindReg(r69, &d142)
		} else if d73.Loc == scm.LocImm {
			r70 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d141.Reg)
			ctx.EmitSetcc(r70, scm.CondUnsignedBelow)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r70}
			ctx.BindReg(r70, &d142)
		} else {
			r71 := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitCmpInt64(d73.Reg, d141.Reg)
			ctx.EmitSetcc(r71, scm.CondUnsignedBelow)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r71}
			ctx.BindReg(r71, &d142)
		}
		ctx.FreeDesc(&d73)
		ctx.FreeDesc(&d141)
		d143 = d142
		ctx.EnsureDesc(&d143)
		if d143.Loc != scm.LocImm && d143.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d143.Loc == scm.LocImm {
			if d143.Imm.Bool() {
				if ps.General {
				}
				ps144 := scm.PhiState{General: ps.General}
				ps144.OverlayValues = make([]scm.JITValueDesc, 144)
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
				ps144.OverlayValues[39] = d39
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
				ps144.OverlayValues[143] = d143
				return bbs[6].RenderPS(ps144)
			}
			if ps.General {
			}
			ps145 := scm.PhiState{General: ps.General}
			ps145.OverlayValues = make([]scm.JITValueDesc, 144)
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
			ps145.OverlayValues[39] = d39
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
			ps145.OverlayValues[143] = d143
			return bbs[7].RenderPS(ps145)
		}
		if !ps.General {
			ps.General = true
			return bbs[5].RenderPS(ps)
		}
		lbl18 := ctx.ReserveLabel()
		lbl19 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d143.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl18)
		ctx.EmitJmp(lbl19)
		ctx.MarkLabel(lbl18)
		ctx.EmitJmp(lbl7)
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl8)
		ps146 := scm.PhiState{General: true}
		ps146.OverlayValues = make([]scm.JITValueDesc, 144)
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
		ps146.OverlayValues[39] = d39
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
		ps146.OverlayValues[143] = d143
		ps147 := scm.PhiState{General: true}
		ps147.OverlayValues = make([]scm.JITValueDesc, 144)
		ps147.OverlayValues[1] = d1
		ps147.OverlayValues[2] = d2
		ps147.OverlayValues[3] = d3
		ps147.OverlayValues[4] = d4
		ps147.OverlayValues[5] = d5
		ps147.OverlayValues[6] = d6
		ps147.OverlayValues[8] = d8
		ps147.OverlayValues[9] = d9
		ps147.OverlayValues[10] = d10
		ps147.OverlayValues[11] = d11
		ps147.OverlayValues[12] = d12
		ps147.OverlayValues[13] = d13
		ps147.OverlayValues[16] = d16
		ps147.OverlayValues[17] = d17
		ps147.OverlayValues[35] = d35
		ps147.OverlayValues[36] = d36
		ps147.OverlayValues[37] = d37
		ps147.OverlayValues[38] = d38
		ps147.OverlayValues[39] = d39
		ps147.OverlayValues[41] = d41
		ps147.OverlayValues[42] = d42
		ps147.OverlayValues[43] = d43
		ps147.OverlayValues[44] = d44
		ps147.OverlayValues[45] = d45
		ps147.OverlayValues[46] = d46
		ps147.OverlayValues[47] = d47
		ps147.OverlayValues[48] = d48
		ps147.OverlayValues[49] = d49
		ps147.OverlayValues[50] = d50
		ps147.OverlayValues[51] = d51
		ps147.OverlayValues[52] = d52
		ps147.OverlayValues[53] = d53
		ps147.OverlayValues[54] = d54
		ps147.OverlayValues[55] = d55
		ps147.OverlayValues[56] = d56
		ps147.OverlayValues[57] = d57
		ps147.OverlayValues[58] = d58
		ps147.OverlayValues[59] = d59
		ps147.OverlayValues[60] = d60
		ps147.OverlayValues[61] = d61
		ps147.OverlayValues[62] = d62
		ps147.OverlayValues[63] = d63
		ps147.OverlayValues[64] = d64
		ps147.OverlayValues[65] = d65
		ps147.OverlayValues[66] = d66
		ps147.OverlayValues[67] = d67
		ps147.OverlayValues[68] = d68
		ps147.OverlayValues[69] = d69
		ps147.OverlayValues[70] = d70
		ps147.OverlayValues[71] = d71
		ps147.OverlayValues[72] = d72
		ps147.OverlayValues[73] = d73
		ps147.OverlayValues[74] = d74
		ps147.OverlayValues[75] = d75
		ps147.OverlayValues[76] = d76
		ps147.OverlayValues[137] = d137
		ps147.OverlayValues[138] = d138
		ps147.OverlayValues[139] = d139
		ps147.OverlayValues[140] = d140
		ps147.OverlayValues[141] = d141
		ps147.OverlayValues[142] = d142
		ps147.OverlayValues[143] = d143
		snap148 := d1
		snap149 := d2
		snap150 := d3
		snap151 := d4
		snap152 := d5
		snap153 := d6
		snap154 := d8
		snap155 := d9
		snap156 := d10
		snap157 := d11
		snap158 := d12
		snap159 := d13
		snap160 := d16
		snap161 := d17
		snap162 := d35
		snap163 := d36
		snap164 := d37
		snap165 := d38
		snap166 := d39
		snap167 := d41
		snap168 := d42
		snap169 := d43
		snap170 := d44
		snap171 := d45
		snap172 := d46
		snap173 := d47
		snap174 := d48
		snap175 := d49
		snap176 := d50
		snap177 := d51
		snap178 := d52
		snap179 := d53
		snap180 := d54
		snap181 := d55
		snap182 := d56
		snap183 := d57
		snap184 := d58
		snap185 := d59
		snap186 := d60
		snap187 := d61
		snap188 := d62
		snap189 := d63
		snap190 := d64
		snap191 := d65
		snap192 := d66
		snap193 := d67
		snap194 := d68
		snap195 := d69
		snap196 := d70
		snap197 := d71
		snap198 := d72
		snap199 := d73
		snap200 := d74
		snap201 := d75
		snap202 := d76
		snap203 := d137
		snap204 := d138
		snap205 := d139
		snap206 := d140
		snap207 := d141
		snap208 := d142
		snap209 := d143
		alloc210 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps147)
		}
		ctx.RestoreAllocState(alloc210)
		d1 = snap148
		d2 = snap149
		d3 = snap150
		d4 = snap151
		d5 = snap152
		d6 = snap153
		d8 = snap154
		d9 = snap155
		d10 = snap156
		d11 = snap157
		d12 = snap158
		d13 = snap159
		d16 = snap160
		d17 = snap161
		d35 = snap162
		d36 = snap163
		d37 = snap164
		d38 = snap165
		d39 = snap166
		d41 = snap167
		d42 = snap168
		d43 = snap169
		d44 = snap170
		d45 = snap171
		d46 = snap172
		d47 = snap173
		d48 = snap174
		d49 = snap175
		d50 = snap176
		d51 = snap177
		d52 = snap178
		d53 = snap179
		d54 = snap180
		d55 = snap181
		d56 = snap182
		d57 = snap183
		d58 = snap184
		d59 = snap185
		d60 = snap186
		d61 = snap187
		d62 = snap188
		d63 = snap189
		d64 = snap190
		d65 = snap191
		d66 = snap192
		d67 = snap193
		d68 = snap194
		d69 = snap195
		d70 = snap196
		d71 = snap197
		d72 = snap198
		d73 = snap199
		d74 = snap200
		d75 = snap201
		d76 = snap202
		d137 = snap203
		d138 = snap204
		d139 = snap205
		d140 = snap206
		d141 = snap207
		d142 = snap208
		d143 = snap209
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps146)
		}
		return result
		ctx.FreeDesc(&d142)
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d38)
		var d211 scm.JITValueDesc
		if d38.Loc == scm.LocImm {
			d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(scratch, d38.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d211)
		}
		if d211.Loc == scm.LocImm {
			d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: d211.Type, Imm: scm.NewInt(int64(uint64(d211.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d211.Reg, 32)
			ctx.EmitShrRegImm8(d211.Reg, 32)
		}
		if d211.Loc == scm.LocReg && d38.Loc == scm.LocReg && d211.Reg == d38.Reg {
			ctx.TransferReg(d38.Reg)
			d38.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d211)
		ctx.EmitStoreToStack(d211, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d211)
		if ps.General {
		}
		ps212 := scm.PhiState{General: ps.General}
		ps212.OverlayValues = make([]scm.JITValueDesc, 212)
		ps212.OverlayValues[1] = d1
		ps212.OverlayValues[2] = d2
		ps212.OverlayValues[3] = d3
		ps212.OverlayValues[4] = d4
		ps212.OverlayValues[5] = d5
		ps212.OverlayValues[6] = d6
		ps212.OverlayValues[8] = d8
		ps212.OverlayValues[9] = d9
		ps212.OverlayValues[10] = d10
		ps212.OverlayValues[11] = d11
		ps212.OverlayValues[12] = d12
		ps212.OverlayValues[13] = d13
		ps212.OverlayValues[16] = d16
		ps212.OverlayValues[17] = d17
		ps212.OverlayValues[35] = d35
		ps212.OverlayValues[36] = d36
		ps212.OverlayValues[37] = d37
		ps212.OverlayValues[38] = d38
		ps212.OverlayValues[39] = d39
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
		ps212.OverlayValues[75] = d75
		ps212.OverlayValues[76] = d76
		ps212.OverlayValues[137] = d137
		ps212.OverlayValues[138] = d138
		ps212.OverlayValues[139] = d139
		ps212.OverlayValues[140] = d140
		ps212.OverlayValues[141] = d141
		ps212.OverlayValues[142] = d142
		ps212.OverlayValues[143] = d143
		ps212.OverlayValues[211] = d211
		ps212.PhiValues = make([]scm.JITValueDesc, 2)
		if ps212.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps212)
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
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 211 && ps.OverlayValues[211].Loc != scm.LocNone {
			d211 = ps.OverlayValues[211]
		}
		ctx.ReclaimUntrackedRegs()
		if ps.General {
		}
		ps213 := scm.PhiState{General: ps.General}
		ps213.OverlayValues = make([]scm.JITValueDesc, 212)
		ps213.OverlayValues[1] = d1
		ps213.OverlayValues[2] = d2
		ps213.OverlayValues[3] = d3
		ps213.OverlayValues[4] = d4
		ps213.OverlayValues[5] = d5
		ps213.OverlayValues[6] = d6
		ps213.OverlayValues[8] = d8
		ps213.OverlayValues[9] = d9
		ps213.OverlayValues[10] = d10
		ps213.OverlayValues[11] = d11
		ps213.OverlayValues[12] = d12
		ps213.OverlayValues[13] = d13
		ps213.OverlayValues[16] = d16
		ps213.OverlayValues[17] = d17
		ps213.OverlayValues[35] = d35
		ps213.OverlayValues[36] = d36
		ps213.OverlayValues[37] = d37
		ps213.OverlayValues[38] = d38
		ps213.OverlayValues[39] = d39
		ps213.OverlayValues[41] = d41
		ps213.OverlayValues[42] = d42
		ps213.OverlayValues[43] = d43
		ps213.OverlayValues[44] = d44
		ps213.OverlayValues[45] = d45
		ps213.OverlayValues[46] = d46
		ps213.OverlayValues[47] = d47
		ps213.OverlayValues[48] = d48
		ps213.OverlayValues[49] = d49
		ps213.OverlayValues[50] = d50
		ps213.OverlayValues[51] = d51
		ps213.OverlayValues[52] = d52
		ps213.OverlayValues[53] = d53
		ps213.OverlayValues[54] = d54
		ps213.OverlayValues[55] = d55
		ps213.OverlayValues[56] = d56
		ps213.OverlayValues[57] = d57
		ps213.OverlayValues[58] = d58
		ps213.OverlayValues[59] = d59
		ps213.OverlayValues[60] = d60
		ps213.OverlayValues[61] = d61
		ps213.OverlayValues[62] = d62
		ps213.OverlayValues[63] = d63
		ps213.OverlayValues[64] = d64
		ps213.OverlayValues[65] = d65
		ps213.OverlayValues[66] = d66
		ps213.OverlayValues[67] = d67
		ps213.OverlayValues[68] = d68
		ps213.OverlayValues[69] = d69
		ps213.OverlayValues[70] = d70
		ps213.OverlayValues[71] = d71
		ps213.OverlayValues[72] = d72
		ps213.OverlayValues[73] = d73
		ps213.OverlayValues[74] = d74
		ps213.OverlayValues[75] = d75
		ps213.OverlayValues[76] = d76
		ps213.OverlayValues[137] = d137
		ps213.OverlayValues[138] = d138
		ps213.OverlayValues[139] = d139
		ps213.OverlayValues[140] = d140
		ps213.OverlayValues[141] = d141
		ps213.OverlayValues[142] = d142
		ps213.OverlayValues[143] = d143
		ps213.OverlayValues[211] = d211
		ps213.PhiValues = make([]scm.JITValueDesc, 2)
		if ps213.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps213)
		return result
	}
	ps214 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps214)
	ctx.MarkLabel(lbl0)
	d215 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d215)
	ctx.BindReg(r1, &d215)
	ctx.EmitMovPairToResult(&d215, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
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
