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
	var d77 scm.JITValueDesc
	_ = d77
	var d78 scm.JITValueDesc
	_ = d78
	var d141 scm.JITValueDesc
	_ = d141
	var d142 scm.JITValueDesc
	_ = d142
	var d143 scm.JITValueDesc
	_ = d143
	var d144 scm.JITValueDesc
	_ = d144
	var d145 scm.JITValueDesc
	_ = d145
	var d146 scm.JITValueDesc
	_ = d146
	var d147 scm.JITValueDesc
	_ = d147
	var d217 scm.JITValueDesc
	_ = d217
	var d219 scm.JITValueDesc
	_ = d219
	var d220 scm.JITValueDesc
	_ = d220
	var d222 scm.JITValueDesc
	_ = d222
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
	bbpos_0_5 := int32(-1)
	_ = bbpos_0_5
	lbl6 := ctx.ReserveLabel()
	_ = lbl6
	bbpos_0_6 := int32(-1)
	_ = bbpos_0_6
	lbl7 := ctx.ReserveLabel()
	_ = lbl7
	bbpos_0_7 := int32(-1)
	_ = bbpos_0_7
	lbl8 := ctx.ReserveLabel()
	_ = lbl8
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
		ctx.EnsureDescsTogether(&d1, &d2)
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
		ctx.EnsureDescsTogether(&d1, &d2)
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
		ctx.StabilizeDescForControlFlow(&d38)
		ctx.FreeDesc(&d37)
		ctx.EnsureDesc(&d38)
		d39 = d38
		_ = d39
		ctx.StabilizeDescForControlFlow(&d39)
		ctx.StabilizeDescForControlFlow(&d38)
		phiBase40 = ctx.AllocStack(int32(16))
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase40) + int32(0)}
		_ = d41
		lbl11 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl12 := ctx.ReserveLabel()
		_ = lbl12
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		lbl13 := ctx.ReserveLabel()
		_ = lbl13
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		lbl14 := ctx.ReserveLabel()
		_ = lbl14
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl12)
		ctx.ResolveFixups()
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d39)
		var d42 scm.JITValueDesc
		if d39.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d39.Imm.Int()))))}
		} else {
			r11 := ctx.AllocReg()
			ctx.EmitMovRegReg(r11, d39.Reg)
			ctx.EmitShlRegImm8(r11, 32)
			ctx.EmitShrRegImm8(r11, 32)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d42)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d43 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
			r12 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r12, fieldAddr)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			ctx.BindReg(r12, &d43)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
			r13 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r13, thisptr.Reg, off)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d43)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d43.Imm.Int()))))}
		} else {
			r14 := ctx.AllocReg()
			ctx.EmitMovRegReg(r14, d43.Reg)
			ctx.EmitShlRegImm8(r14, 56)
			ctx.EmitShrRegImm8(r14, 56)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d44)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d42)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDescsTogether(&d42, &d44)
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
			r15 := ctx.AllocRegExcept(d42.Reg, d44.Reg)
			ctx.EmitMovRegReg(r15, d42.Reg)
			ctx.EmitImulInt64(r15, d44.Reg)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d45)
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
			r16 := ctx.AllocReg()
			r17 := ctx.AllocRegExcept(r16)
			r18 := ctx.AllocRegExcept(r16, r17)
			ctx.EmitMovRegMem64(r16, fieldAddr)
			ctx.EmitMovRegMem64(r17, fieldAddr+8)
			ctx.EmitMovRegMem64(r18, fieldAddr+16)
			d46 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r16, Reg2: r17, Reg3: r18}
			ctx.BindReg(r16, &d46)
			ctx.BindReg(r17, &d46)
			ctx.BindReg(r18, &d46)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
			r19 := ctx.AllocReg()
			r20 := ctx.AllocRegExcept(r19)
			r21 := ctx.AllocRegExcept(r19, r20)
			ctx.EmitMovRegMem(r19, thisptr.Reg, off)
			ctx.EmitMovRegMem(r20, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r21, thisptr.Reg, off+16)
			d46 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r19, Reg2: r20, Reg3: r21}
			ctx.BindReg(r19, &d46)
			ctx.BindReg(r20, &d46)
			ctx.BindReg(r21, &d46)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d47 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() / 64)}
		} else {
			r22 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r22, d45.Reg)
			ctx.EmitShrRegImm8(r22, 6)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d47)
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
			r23 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r23, d45.Reg)
			ctx.EmitAndRegImm32(r23, 63)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d50)
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
			r24 := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegReg(r24, d48.Reg)
			ctx.EmitShlRegImm8(r24, uint8(d50.Imm.Int()))
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d51)
		} else {
			{
				shiftSrc := d48.Reg
				r25 := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegReg(r25, d48.Reg)
				shiftSrc = r25
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
		ctx.StabilizeDescForControlFlow(&d51)
		ctx.FreeDesc(&d48)
		ctx.FreeDesc(&d50)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d52 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() % 64)}
		} else {
			r26 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r26, d45.Reg)
			ctx.EmitAndRegImm32(r26, 63)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d52)
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
			r27 := ctx.AllocReg()
			ctx.EmitMovRegReg(r27, d43.Reg)
			ctx.EmitShlRegImm8(r27, 56)
			ctx.EmitShrRegImm8(r27, 56)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d53)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d52)
		ctx.EnsureDesc(&d53)
		ctx.EnsureDescsTogether(&d52, &d53)
		var d54 scm.JITValueDesc
		if d52.Loc == scm.LocImm && d53.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d53.Imm.Int())}
		} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
			r28 := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegReg(r28, d52.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d54)
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
			r29 := ctx.AllocRegExcept(d52.Reg, d53.Reg)
			ctx.EmitMovRegReg(r29, d52.Reg)
			ctx.EmitAddInt64(r29, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d54)
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
			r30 := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitCmpRegImm32(d54.Reg, 64)
			ctx.EmitSetcc(r30, scm.CondUnsignedAbove)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
			ctx.BindReg(r30, &d55)
		}
		ctx.FreeDesc(&d54)
		ctx.ReclaimUntrackedRegs()
		d56 = d55
		ctx.EnsureDesc(&d56)
		if d56.Loc != scm.LocImm && d56.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl15 := ctx.ReserveLabel()
		lbl16 := ctx.ReserveLabel()
		if d56.Loc == scm.LocImm {
			if d56.Imm.Bool() {
				ctx.MarkLabel(lbl15)
				ctx.EmitJmp(lbl13)
			} else {
				ctx.MarkLabel(lbl16)
				ctx.SyncDesc(&d51)
				if d51.Loc == scm.LocReg {
					ctx.ProtectReg(d51.Reg)
				} else if d51.Loc == scm.LocRegPair {
					ctx.ProtectReg(d51.Reg)
					ctx.ProtectReg(d51.Reg2)
				}
				d57 = d51
				if d57.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d57)
				ctx.EmitStoreToStack(d57, int32(phiBase40)+int32(0))
				if d51.Loc == scm.LocReg {
					ctx.UnprotectReg(d51.Reg)
				} else if d51.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d51.Reg)
					ctx.UnprotectReg(d51.Reg2)
				}
				ctx.EmitJmp(lbl14)
			}
		} else {
			ctx.EmitCmpRegImm32(d56.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl16)
			ctx.SyncDesc(&d51)
			if d51.Loc == scm.LocReg {
				ctx.ProtectReg(d51.Reg)
			} else if d51.Loc == scm.LocRegPair {
				ctx.ProtectReg(d51.Reg)
				ctx.ProtectReg(d51.Reg2)
			}
			d58 = d51
			if d58.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d58)
			ctx.EmitStoreToStack(d58, int32(phiBase40)+int32(0))
			if d51.Loc == scm.LocReg {
				ctx.UnprotectReg(d51.Reg)
			} else if d51.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d51.Reg)
				ctx.UnprotectReg(d51.Reg2)
			}
			ctx.EmitJmp(lbl14)
		}
		ctx.FreeDesc(&d55)
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl14)
		ctx.ResolveFixups()
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d43)
		var d59 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d43.Imm.Int()))))}
		} else {
			r31 := ctx.AllocReg()
			ctx.EmitMovRegReg(r31, d43.Reg)
			ctx.EmitShlRegImm8(r31, 56)
			ctx.EmitShrRegImm8(r31, 56)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d59)
		}
		ctx.ReclaimUntrackedRegs()
		d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&d60, &d59)
		var d61 scm.JITValueDesc
		if d60.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d60.Imm.Int() - d59.Imm.Int())}
		} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
			r32 := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegReg(r32, d60.Reg)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d61)
		} else if d60.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d60.Imm.Int()))
			ctx.EmitSubInt64(scratch, d59.Reg)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d61)
		} else if d59.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegReg(scratch, d60.Reg)
			if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d59.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d61)
		} else {
			r33 := ctx.AllocRegExcept(d60.Reg, d59.Reg)
			ctx.EmitMovRegReg(r33, d60.Reg)
			ctx.EmitSubInt64(r33, d59.Reg)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d61)
		}
		if d61.Loc == scm.LocReg && d60.Loc == scm.LocReg && d61.Reg == d60.Reg {
			ctx.TransferReg(d60.Reg)
			d60.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d61)
		var d62 scm.JITValueDesc
		if d41.Loc == scm.LocImm && d61.Loc == scm.LocImm {
			d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d41.Imm.Int()) >> uint64(d61.Imm.Int())))}
		} else if d61.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r34, d41.Reg)
			ctx.EmitShrRegImm8(r34, uint8(d61.Imm.Int()))
			d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d62)
		} else {
			{
				shiftSrc := d41.Reg
				r35 := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(r35, d41.Reg)
				shiftSrc = r35
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d61.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d61.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d61.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d62)
			}
		}
		if d62.Loc == scm.LocReg && d41.Loc == scm.LocReg && d62.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d41)
		ctx.FreeDesc(&d61)
		ctx.ReclaimUntrackedRegs()
		r36 := ctx.AllocReg()
		ctx.EnsureDesc(&d62)
		ctx.EnsureDesc(&d62)
		if d62.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r36, d62)
		}
		ctx.EmitJmp(lbl11)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl13)
		ctx.ResolveFixups()
		d41 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d63 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() / 64)}
		} else {
			r37 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r37, d45.Reg)
			ctx.EmitShrRegImm8(r37, 6)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d63)
		}
		if d63.Loc == scm.LocReg && d45.Loc == scm.LocReg && d63.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d63)
		var d64 scm.JITValueDesc
		if d63.Loc == scm.LocImm {
			d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d63.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitMovRegReg(scratch, d63.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d64)
		}
		if d64.Loc == scm.LocReg && d63.Loc == scm.LocReg && d64.Reg == d63.Reg {
			ctx.TransferReg(d63.Reg)
			d63.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d63)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d64)
		ctx.ReclaimUntrackedRegs()
		d66 = ctx.EmitSliceElementAddress(&d46, &d64, 8)
		ctx.EnsureDesc(&d66)
		ctx.EmitMovRegMem(d66.Reg, d66.Reg, 0)
		d65 = d66
		ctx.FreeDesc(&d64)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d45)
		var d67 scm.JITValueDesc
		if d45.Loc == scm.LocImm {
			d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() % 64)}
		} else {
			r38 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r38, d45.Reg)
			ctx.EmitAndRegImm32(r38, 63)
			d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d67)
		}
		if d67.Loc == scm.LocReg && d45.Loc == scm.LocReg && d67.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d67)
		ctx.EnsureDescsTogether(&d68, &d67)
		var d69 scm.JITValueDesc
		if d68.Loc == scm.LocImm && d67.Loc == scm.LocImm {
			d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d68.Imm.Int() - d67.Imm.Int())}
		} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
			r39 := ctx.AllocRegExcept(d68.Reg)
			ctx.EmitMovRegReg(r39, d68.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d69)
		} else if d68.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d67.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
			ctx.EmitSubInt64(scratch, d67.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d69)
		} else if d67.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d68.Reg)
			ctx.EmitMovRegReg(scratch, d68.Reg)
			if d67.Imm.Int() >= -2147483648 && d67.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d67.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d67.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d69)
		} else {
			r40 := ctx.AllocRegExcept(d68.Reg, d67.Reg)
			ctx.EmitMovRegReg(r40, d68.Reg)
			ctx.EmitSubInt64(r40, d67.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d69)
		}
		if d69.Loc == scm.LocReg && d68.Loc == scm.LocReg && d69.Reg == d68.Reg {
			ctx.TransferReg(d68.Reg)
			d68.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d67)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d65)
		ctx.EnsureDesc(&d69)
		var d70 scm.JITValueDesc
		if d65.Loc == scm.LocImm && d69.Loc == scm.LocImm {
			d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) >> uint64(d69.Imm.Int())))}
		} else if d69.Loc == scm.LocImm {
			r41 := ctx.AllocRegExcept(d65.Reg)
			ctx.EmitMovRegReg(r41, d65.Reg)
			ctx.EmitShrRegImm8(r41, uint8(d69.Imm.Int()))
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d70)
		} else {
			{
				shiftSrc := d65.Reg
				r42 := ctx.AllocRegExcept(d65.Reg)
				ctx.EmitMovRegReg(r42, d65.Reg)
				shiftSrc = r42
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d69.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d69.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d69.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d70)
			}
		}
		if d70.Loc == scm.LocReg && d65.Loc == scm.LocReg && d70.Reg == d65.Reg {
			ctx.TransferReg(d65.Reg)
			d65.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d65)
		ctx.FreeDesc(&d69)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d70)
		var d71 scm.JITValueDesc
		if d51.Loc == scm.LocImm && d70.Loc == scm.LocImm {
			d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() | d70.Imm.Int())}
		} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d70.Reg}
			ctx.BindReg(d70.Reg, &d71)
		} else if d70.Loc == scm.LocImm && d70.Imm.Int() == 0 {
			r43 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r43, d51.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d71)
		} else if d51.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d70.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d51.Imm.Int()))
			ctx.EmitOrInt64(scratch, d70.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d71)
		} else if d70.Loc == scm.LocImm {
			r44 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r44, d51.Reg)
			if d70.Imm.Int() >= -2147483648 && d70.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r44, int32(d70.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d70.Imm.Int()))
				ctx.EmitOrInt64(r44, scm.RegR11)
			}
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d71)
		} else {
			r45 := ctx.AllocRegExcept(d51.Reg, d70.Reg)
			ctx.EmitMovRegReg(r45, d51.Reg)
			ctx.EmitOrInt64(r45, d70.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d71)
		}
		if d71.Loc == scm.LocReg && d51.Loc == scm.LocReg && d71.Reg == d51.Reg {
			ctx.TransferReg(d51.Reg)
			d51.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d71)
		ctx.EmitStoreToStack(d71, int32(phiBase40)+int32(0))
		ctx.StabilizeDescForControlFlow(&d71)
		ctx.FreeDesc(&d70)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl11)
		d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r36}
		ctx.BindReg(r36, &d72)
		ctx.BindReg(r36, &d72)
		var d73 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
			r46 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r46, fieldAddr)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
			ctx.BindReg(r46, &d73)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
			r47 := ctx.AllocReg()
			ctx.EmitMovRegMem(r47, thisptr.Reg, off)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			ctx.BindReg(r47, &d73)
		}
		ctx.EnsureDesc(&d73)
		ctx.EnsureDesc(&d73)
		var d74 scm.JITValueDesc
		if d73.Loc == scm.LocImm {
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d73.Imm.Int()))))}
		} else {
			r48 := ctx.AllocReg()
			ctx.EmitMovRegReg(r48, d73.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d74)
		}
		ctx.EnsureDesc(&d72)
		ctx.EnsureDesc(&d74)
		ctx.EnsureDescsTogether(&d72, &d74)
		var d75 scm.JITValueDesc
		if d72.Loc == scm.LocImm && d74.Loc == scm.LocImm {
			d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d72.Imm.Int() + d74.Imm.Int())}
		} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
			r49 := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitMovRegReg(r49, d72.Reg)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d75)
		} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
			ctx.BindReg(d74.Reg, &d75)
		} else if d72.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d72.Imm.Int()))
			ctx.EmitAddInt64(scratch, d74.Reg)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d75)
		} else if d74.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitMovRegReg(scratch, d72.Reg)
			if d74.Imm.Int() >= -2147483648 && d74.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d74.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d75)
		} else {
			r50 := ctx.AllocRegExcept(d72.Reg, d74.Reg)
			ctx.EmitMovRegReg(r50, d72.Reg)
			ctx.EmitAddInt64(r50, d74.Reg)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d75)
		}
		if d75.Loc == scm.LocReg && d72.Loc == scm.LocReg && d75.Reg == d72.Reg {
			ctx.TransferReg(d72.Reg)
			d72.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d75)
		ctx.FreeDesc(&d72)
		ctx.FreeDesc(&d74)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d76 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r51 := ctx.AllocReg()
			ctx.EmitMovRegReg(r51, idxInt.Reg)
			ctx.EmitShlRegImm8(r51, 32)
			ctx.EmitShrRegImm8(r51, 32)
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d76)
		}
		ctx.EnsureDesc(&d75)
		ctx.EnsureDesc(&d76)
		ctx.EnsureDescsTogether(&d75, &d76)
		var d77 scm.JITValueDesc
		if d75.Loc == scm.LocImm && d76.Loc == scm.LocImm {
			d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d75.Imm.Int()) == uint64(d76.Imm.Int()))}
		} else if d76.Loc == scm.LocImm {
			r52 := ctx.AllocRegExcept(d75.Reg)
			if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d75.Reg, int32(d76.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
				ctx.EmitCmpInt64(d75.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r52, scm.CondEqual)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
			ctx.BindReg(r52, &d77)
		} else if d75.Loc == scm.LocImm {
			r53 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d76.Reg)
			ctx.EmitSetcc(r53, scm.CondEqual)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
			ctx.BindReg(r53, &d77)
		} else {
			r54 := ctx.AllocRegExcept(d75.Reg)
			ctx.EmitCmpInt64(d75.Reg, d76.Reg)
			ctx.EmitSetcc(r54, scm.CondEqual)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
			ctx.BindReg(r54, &d77)
		}
		ctx.FreeDesc(&d76)
		d78 = d77
		ctx.EnsureDesc(&d78)
		if d78.Loc != scm.LocImm && d78.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d78.Loc == scm.LocImm {
			if d78.Imm.Bool() {
				if ps.General {
				}
				ps79 := scm.PhiState{General: ps.General}
				ps79.OverlayValues = make([]scm.JITValueDesc, 79)
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
				ps79.OverlayValues[77] = d77
				ps79.OverlayValues[78] = d78
				return bbs[4].RenderPS(ps79)
			}
			if ps.General {
			}
			ps80 := scm.PhiState{General: ps.General}
			ps80.OverlayValues = make([]scm.JITValueDesc, 79)
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
			ps80.OverlayValues[77] = d77
			ps80.OverlayValues[78] = d78
			return bbs[5].RenderPS(ps80)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl17 := ctx.ReserveLabel()
		lbl18 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d78.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl17)
		ctx.EmitJmp(lbl18)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl5)
		ctx.MarkLabel(lbl18)
		ctx.EmitJmp(lbl6)
		ps81 := scm.PhiState{General: true}
		ps81.OverlayValues = make([]scm.JITValueDesc, 79)
		ps81.OverlayValues[1] = d1
		ps81.OverlayValues[2] = d2
		ps81.OverlayValues[3] = d3
		ps81.OverlayValues[4] = d4
		ps81.OverlayValues[5] = d5
		ps81.OverlayValues[6] = d6
		ps81.OverlayValues[8] = d8
		ps81.OverlayValues[9] = d9
		ps81.OverlayValues[10] = d10
		ps81.OverlayValues[11] = d11
		ps81.OverlayValues[12] = d12
		ps81.OverlayValues[13] = d13
		ps81.OverlayValues[16] = d16
		ps81.OverlayValues[17] = d17
		ps81.OverlayValues[35] = d35
		ps81.OverlayValues[36] = d36
		ps81.OverlayValues[37] = d37
		ps81.OverlayValues[38] = d38
		ps81.OverlayValues[39] = d39
		ps81.OverlayValues[41] = d41
		ps81.OverlayValues[42] = d42
		ps81.OverlayValues[43] = d43
		ps81.OverlayValues[44] = d44
		ps81.OverlayValues[45] = d45
		ps81.OverlayValues[46] = d46
		ps81.OverlayValues[47] = d47
		ps81.OverlayValues[48] = d48
		ps81.OverlayValues[49] = d49
		ps81.OverlayValues[50] = d50
		ps81.OverlayValues[51] = d51
		ps81.OverlayValues[52] = d52
		ps81.OverlayValues[53] = d53
		ps81.OverlayValues[54] = d54
		ps81.OverlayValues[55] = d55
		ps81.OverlayValues[56] = d56
		ps81.OverlayValues[57] = d57
		ps81.OverlayValues[58] = d58
		ps81.OverlayValues[59] = d59
		ps81.OverlayValues[60] = d60
		ps81.OverlayValues[61] = d61
		ps81.OverlayValues[62] = d62
		ps81.OverlayValues[63] = d63
		ps81.OverlayValues[64] = d64
		ps81.OverlayValues[65] = d65
		ps81.OverlayValues[66] = d66
		ps81.OverlayValues[67] = d67
		ps81.OverlayValues[68] = d68
		ps81.OverlayValues[69] = d69
		ps81.OverlayValues[70] = d70
		ps81.OverlayValues[71] = d71
		ps81.OverlayValues[72] = d72
		ps81.OverlayValues[73] = d73
		ps81.OverlayValues[74] = d74
		ps81.OverlayValues[75] = d75
		ps81.OverlayValues[76] = d76
		ps81.OverlayValues[77] = d77
		ps81.OverlayValues[78] = d78
		ps82 := scm.PhiState{General: true}
		ps82.OverlayValues = make([]scm.JITValueDesc, 79)
		ps82.OverlayValues[1] = d1
		ps82.OverlayValues[2] = d2
		ps82.OverlayValues[3] = d3
		ps82.OverlayValues[4] = d4
		ps82.OverlayValues[5] = d5
		ps82.OverlayValues[6] = d6
		ps82.OverlayValues[8] = d8
		ps82.OverlayValues[9] = d9
		ps82.OverlayValues[10] = d10
		ps82.OverlayValues[11] = d11
		ps82.OverlayValues[12] = d12
		ps82.OverlayValues[13] = d13
		ps82.OverlayValues[16] = d16
		ps82.OverlayValues[17] = d17
		ps82.OverlayValues[35] = d35
		ps82.OverlayValues[36] = d36
		ps82.OverlayValues[37] = d37
		ps82.OverlayValues[38] = d38
		ps82.OverlayValues[39] = d39
		ps82.OverlayValues[41] = d41
		ps82.OverlayValues[42] = d42
		ps82.OverlayValues[43] = d43
		ps82.OverlayValues[44] = d44
		ps82.OverlayValues[45] = d45
		ps82.OverlayValues[46] = d46
		ps82.OverlayValues[47] = d47
		ps82.OverlayValues[48] = d48
		ps82.OverlayValues[49] = d49
		ps82.OverlayValues[50] = d50
		ps82.OverlayValues[51] = d51
		ps82.OverlayValues[52] = d52
		ps82.OverlayValues[53] = d53
		ps82.OverlayValues[54] = d54
		ps82.OverlayValues[55] = d55
		ps82.OverlayValues[56] = d56
		ps82.OverlayValues[57] = d57
		ps82.OverlayValues[58] = d58
		ps82.OverlayValues[59] = d59
		ps82.OverlayValues[60] = d60
		ps82.OverlayValues[61] = d61
		ps82.OverlayValues[62] = d62
		ps82.OverlayValues[63] = d63
		ps82.OverlayValues[64] = d64
		ps82.OverlayValues[65] = d65
		ps82.OverlayValues[66] = d66
		ps82.OverlayValues[67] = d67
		ps82.OverlayValues[68] = d68
		ps82.OverlayValues[69] = d69
		ps82.OverlayValues[70] = d70
		ps82.OverlayValues[71] = d71
		ps82.OverlayValues[72] = d72
		ps82.OverlayValues[73] = d73
		ps82.OverlayValues[74] = d74
		ps82.OverlayValues[75] = d75
		ps82.OverlayValues[76] = d76
		ps82.OverlayValues[77] = d77
		ps82.OverlayValues[78] = d78
		snap83 := d1
		snap84 := d2
		snap85 := d3
		snap86 := d4
		snap87 := d5
		snap88 := d6
		snap89 := d8
		snap90 := d9
		snap91 := d10
		snap92 := d11
		snap93 := d12
		snap94 := d13
		snap95 := d16
		snap96 := d17
		snap97 := d35
		snap98 := d36
		snap99 := d37
		snap100 := d38
		snap101 := d39
		snap102 := d41
		snap103 := d42
		snap104 := d43
		snap105 := d44
		snap106 := d45
		snap107 := d46
		snap108 := d47
		snap109 := d48
		snap110 := d49
		snap111 := d50
		snap112 := d51
		snap113 := d52
		snap114 := d53
		snap115 := d54
		snap116 := d55
		snap117 := d56
		snap118 := d57
		snap119 := d58
		snap120 := d59
		snap121 := d60
		snap122 := d61
		snap123 := d62
		snap124 := d63
		snap125 := d64
		snap126 := d65
		snap127 := d66
		snap128 := d67
		snap129 := d68
		snap130 := d69
		snap131 := d70
		snap132 := d71
		snap133 := d72
		snap134 := d73
		snap135 := d74
		snap136 := d75
		snap137 := d76
		snap138 := d77
		snap139 := d78
		alloc140 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps82)
		}
		ctx.RestoreAllocState(alloc140)
		d1 = snap83
		d2 = snap84
		d3 = snap85
		d4 = snap86
		d5 = snap87
		d6 = snap88
		d8 = snap89
		d9 = snap90
		d10 = snap91
		d11 = snap92
		d12 = snap93
		d13 = snap94
		d16 = snap95
		d17 = snap96
		d35 = snap97
		d36 = snap98
		d37 = snap99
		d38 = snap100
		d39 = snap101
		d41 = snap102
		d42 = snap103
		d43 = snap104
		d44 = snap105
		d45 = snap106
		d46 = snap107
		d47 = snap108
		d48 = snap109
		d49 = snap110
		d50 = snap111
		d51 = snap112
		d52 = snap113
		d53 = snap114
		d54 = snap115
		d55 = snap116
		d56 = snap117
		d57 = snap118
		d58 = snap119
		d59 = snap120
		d60 = snap121
		d61 = snap122
		d62 = snap123
		d63 = snap124
		d64 = snap125
		d65 = snap126
		d66 = snap127
		d67 = snap128
		d68 = snap129
		d69 = snap130
		d70 = snap131
		d71 = snap132
		d72 = snap133
		d73 = snap134
		d74 = snap135
		d75 = snap136
		d76 = snap137
		d77 = snap138
		d78 = snap139
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps81)
		}
		return result
		ctx.FreeDesc(&d77)
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
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		ctx.ReclaimUntrackedRegs()
		var d141 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
			r55 := ctx.AllocReg()
			r56 := ctx.AllocRegExcept(r55)
			r57 := ctx.AllocRegExcept(r55, r56)
			ctx.EmitMovRegMem64(r55, fieldAddr)
			ctx.EmitMovRegMem64(r56, fieldAddr+8)
			ctx.EmitMovRegMem64(r57, fieldAddr+16)
			d141 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r55, Reg2: r56, Reg3: r57}
			ctx.BindReg(r55, &d141)
			ctx.BindReg(r56, &d141)
			ctx.BindReg(r57, &d141)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
			r58 := ctx.AllocReg()
			r59 := ctx.AllocRegExcept(r58)
			r60 := ctx.AllocRegExcept(r58, r59)
			ctx.EmitMovRegMem(r58, thisptr.Reg, off)
			ctx.EmitMovRegMem(r59, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r60, thisptr.Reg, off+16)
			d141 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r58, Reg2: r59, Reg3: r60}
			ctx.BindReg(r58, &d141)
			ctx.BindReg(r59, &d141)
			ctx.BindReg(r60, &d141)
		}
		ctx.EnsureDesc(&d38)
		d143 = ctx.EmitSliceElementAddress(&d141, &d38, 16)
		ctx.EnsureDesc(&d143)
		r61 := ctx.AllocRegExcept(d143.Reg)
		ctx.EmitMovRegMem(r61, d143.Reg, 8)
		ctx.EmitMovRegMem(d143.Reg, d143.Reg, 0)
		d142 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d143.Reg, Reg2: r61}
		ctx.BindReg(d143.Reg, &d142)
		ctx.BindReg(r61, &d142)
		d144 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d144)
		ctx.BindReg(r1, &d144)
		ctx.EnsureDesc(&d142)
		if d142.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d142, &d144)
		} else {
			switch d142.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d144, d142)
			case scm.TagInt:
				ctx.EmitMakeInt(d144, d142)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d144, d142)
			case scm.TagNil:
				ctx.EmitMakeNil(d144)
			default:
				ctx.EmitMovPairToResult(&d142, &d144)
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
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
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
		if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
			d144 = ps.OverlayValues[144]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d145 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r62 := ctx.AllocReg()
			ctx.EmitMovRegReg(r62, idxInt.Reg)
			ctx.EmitShlRegImm8(r62, 32)
			ctx.EmitShrRegImm8(r62, 32)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d145)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d75)
		ctx.EnsureDesc(&d145)
		ctx.EnsureDescsTogether(&d75, &d145)
		var d146 scm.JITValueDesc
		if d75.Loc == scm.LocImm && d145.Loc == scm.LocImm {
			d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d75.Imm.Int()) < uint64(d145.Imm.Int()))}
		} else if d145.Loc == scm.LocImm {
			r63 := ctx.AllocRegExcept(d75.Reg)
			if d145.Imm.Int() >= -2147483648 && d145.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d75.Reg, int32(d145.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
				ctx.EmitCmpInt64(d75.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r63, scm.CondUnsignedBelow)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r63}
			ctx.BindReg(r63, &d146)
		} else if d75.Loc == scm.LocImm {
			r64 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d145.Reg)
			ctx.EmitSetcc(r64, scm.CondUnsignedBelow)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r64}
			ctx.BindReg(r64, &d146)
		} else {
			r65 := ctx.AllocRegExcept(d75.Reg)
			ctx.EmitCmpInt64(d75.Reg, d145.Reg)
			ctx.EmitSetcc(r65, scm.CondUnsignedBelow)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r65}
			ctx.BindReg(r65, &d146)
		}
		ctx.FreeDesc(&d75)
		ctx.FreeDesc(&d145)
		d147 = d146
		ctx.EnsureDesc(&d147)
		if d147.Loc != scm.LocImm && d147.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d147.Loc == scm.LocImm {
			if d147.Imm.Bool() {
				if ps.General {
				}
				ps148 := scm.PhiState{General: ps.General}
				ps148.OverlayValues = make([]scm.JITValueDesc, 148)
				ps148.OverlayValues[1] = d1
				ps148.OverlayValues[2] = d2
				ps148.OverlayValues[3] = d3
				ps148.OverlayValues[4] = d4
				ps148.OverlayValues[5] = d5
				ps148.OverlayValues[6] = d6
				ps148.OverlayValues[8] = d8
				ps148.OverlayValues[9] = d9
				ps148.OverlayValues[10] = d10
				ps148.OverlayValues[11] = d11
				ps148.OverlayValues[12] = d12
				ps148.OverlayValues[13] = d13
				ps148.OverlayValues[16] = d16
				ps148.OverlayValues[17] = d17
				ps148.OverlayValues[35] = d35
				ps148.OverlayValues[36] = d36
				ps148.OverlayValues[37] = d37
				ps148.OverlayValues[38] = d38
				ps148.OverlayValues[39] = d39
				ps148.OverlayValues[41] = d41
				ps148.OverlayValues[42] = d42
				ps148.OverlayValues[43] = d43
				ps148.OverlayValues[44] = d44
				ps148.OverlayValues[45] = d45
				ps148.OverlayValues[46] = d46
				ps148.OverlayValues[47] = d47
				ps148.OverlayValues[48] = d48
				ps148.OverlayValues[49] = d49
				ps148.OverlayValues[50] = d50
				ps148.OverlayValues[51] = d51
				ps148.OverlayValues[52] = d52
				ps148.OverlayValues[53] = d53
				ps148.OverlayValues[54] = d54
				ps148.OverlayValues[55] = d55
				ps148.OverlayValues[56] = d56
				ps148.OverlayValues[57] = d57
				ps148.OverlayValues[58] = d58
				ps148.OverlayValues[59] = d59
				ps148.OverlayValues[60] = d60
				ps148.OverlayValues[61] = d61
				ps148.OverlayValues[62] = d62
				ps148.OverlayValues[63] = d63
				ps148.OverlayValues[64] = d64
				ps148.OverlayValues[65] = d65
				ps148.OverlayValues[66] = d66
				ps148.OverlayValues[67] = d67
				ps148.OverlayValues[68] = d68
				ps148.OverlayValues[69] = d69
				ps148.OverlayValues[70] = d70
				ps148.OverlayValues[71] = d71
				ps148.OverlayValues[72] = d72
				ps148.OverlayValues[73] = d73
				ps148.OverlayValues[74] = d74
				ps148.OverlayValues[75] = d75
				ps148.OverlayValues[76] = d76
				ps148.OverlayValues[77] = d77
				ps148.OverlayValues[78] = d78
				ps148.OverlayValues[141] = d141
				ps148.OverlayValues[142] = d142
				ps148.OverlayValues[143] = d143
				ps148.OverlayValues[144] = d144
				ps148.OverlayValues[145] = d145
				ps148.OverlayValues[146] = d146
				ps148.OverlayValues[147] = d147
				return bbs[6].RenderPS(ps148)
			}
			if ps.General {
			}
			ps149 := scm.PhiState{General: ps.General}
			ps149.OverlayValues = make([]scm.JITValueDesc, 148)
			ps149.OverlayValues[1] = d1
			ps149.OverlayValues[2] = d2
			ps149.OverlayValues[3] = d3
			ps149.OverlayValues[4] = d4
			ps149.OverlayValues[5] = d5
			ps149.OverlayValues[6] = d6
			ps149.OverlayValues[8] = d8
			ps149.OverlayValues[9] = d9
			ps149.OverlayValues[10] = d10
			ps149.OverlayValues[11] = d11
			ps149.OverlayValues[12] = d12
			ps149.OverlayValues[13] = d13
			ps149.OverlayValues[16] = d16
			ps149.OverlayValues[17] = d17
			ps149.OverlayValues[35] = d35
			ps149.OverlayValues[36] = d36
			ps149.OverlayValues[37] = d37
			ps149.OverlayValues[38] = d38
			ps149.OverlayValues[39] = d39
			ps149.OverlayValues[41] = d41
			ps149.OverlayValues[42] = d42
			ps149.OverlayValues[43] = d43
			ps149.OverlayValues[44] = d44
			ps149.OverlayValues[45] = d45
			ps149.OverlayValues[46] = d46
			ps149.OverlayValues[47] = d47
			ps149.OverlayValues[48] = d48
			ps149.OverlayValues[49] = d49
			ps149.OverlayValues[50] = d50
			ps149.OverlayValues[51] = d51
			ps149.OverlayValues[52] = d52
			ps149.OverlayValues[53] = d53
			ps149.OverlayValues[54] = d54
			ps149.OverlayValues[55] = d55
			ps149.OverlayValues[56] = d56
			ps149.OverlayValues[57] = d57
			ps149.OverlayValues[58] = d58
			ps149.OverlayValues[59] = d59
			ps149.OverlayValues[60] = d60
			ps149.OverlayValues[61] = d61
			ps149.OverlayValues[62] = d62
			ps149.OverlayValues[63] = d63
			ps149.OverlayValues[64] = d64
			ps149.OverlayValues[65] = d65
			ps149.OverlayValues[66] = d66
			ps149.OverlayValues[67] = d67
			ps149.OverlayValues[68] = d68
			ps149.OverlayValues[69] = d69
			ps149.OverlayValues[70] = d70
			ps149.OverlayValues[71] = d71
			ps149.OverlayValues[72] = d72
			ps149.OverlayValues[73] = d73
			ps149.OverlayValues[74] = d74
			ps149.OverlayValues[75] = d75
			ps149.OverlayValues[76] = d76
			ps149.OverlayValues[77] = d77
			ps149.OverlayValues[78] = d78
			ps149.OverlayValues[141] = d141
			ps149.OverlayValues[142] = d142
			ps149.OverlayValues[143] = d143
			ps149.OverlayValues[144] = d144
			ps149.OverlayValues[145] = d145
			ps149.OverlayValues[146] = d146
			ps149.OverlayValues[147] = d147
			return bbs[7].RenderPS(ps149)
		}
		if !ps.General {
			ps.General = true
			return bbs[5].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d147.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl19)
		ctx.EmitJmp(lbl20)
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl7)
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl8)
		ps150 := scm.PhiState{General: true}
		ps150.OverlayValues = make([]scm.JITValueDesc, 148)
		ps150.OverlayValues[1] = d1
		ps150.OverlayValues[2] = d2
		ps150.OverlayValues[3] = d3
		ps150.OverlayValues[4] = d4
		ps150.OverlayValues[5] = d5
		ps150.OverlayValues[6] = d6
		ps150.OverlayValues[8] = d8
		ps150.OverlayValues[9] = d9
		ps150.OverlayValues[10] = d10
		ps150.OverlayValues[11] = d11
		ps150.OverlayValues[12] = d12
		ps150.OverlayValues[13] = d13
		ps150.OverlayValues[16] = d16
		ps150.OverlayValues[17] = d17
		ps150.OverlayValues[35] = d35
		ps150.OverlayValues[36] = d36
		ps150.OverlayValues[37] = d37
		ps150.OverlayValues[38] = d38
		ps150.OverlayValues[39] = d39
		ps150.OverlayValues[41] = d41
		ps150.OverlayValues[42] = d42
		ps150.OverlayValues[43] = d43
		ps150.OverlayValues[44] = d44
		ps150.OverlayValues[45] = d45
		ps150.OverlayValues[46] = d46
		ps150.OverlayValues[47] = d47
		ps150.OverlayValues[48] = d48
		ps150.OverlayValues[49] = d49
		ps150.OverlayValues[50] = d50
		ps150.OverlayValues[51] = d51
		ps150.OverlayValues[52] = d52
		ps150.OverlayValues[53] = d53
		ps150.OverlayValues[54] = d54
		ps150.OverlayValues[55] = d55
		ps150.OverlayValues[56] = d56
		ps150.OverlayValues[57] = d57
		ps150.OverlayValues[58] = d58
		ps150.OverlayValues[59] = d59
		ps150.OverlayValues[60] = d60
		ps150.OverlayValues[61] = d61
		ps150.OverlayValues[62] = d62
		ps150.OverlayValues[63] = d63
		ps150.OverlayValues[64] = d64
		ps150.OverlayValues[65] = d65
		ps150.OverlayValues[66] = d66
		ps150.OverlayValues[67] = d67
		ps150.OverlayValues[68] = d68
		ps150.OverlayValues[69] = d69
		ps150.OverlayValues[70] = d70
		ps150.OverlayValues[71] = d71
		ps150.OverlayValues[72] = d72
		ps150.OverlayValues[73] = d73
		ps150.OverlayValues[74] = d74
		ps150.OverlayValues[75] = d75
		ps150.OverlayValues[76] = d76
		ps150.OverlayValues[77] = d77
		ps150.OverlayValues[78] = d78
		ps150.OverlayValues[141] = d141
		ps150.OverlayValues[142] = d142
		ps150.OverlayValues[143] = d143
		ps150.OverlayValues[144] = d144
		ps150.OverlayValues[145] = d145
		ps150.OverlayValues[146] = d146
		ps150.OverlayValues[147] = d147
		ps151 := scm.PhiState{General: true}
		ps151.OverlayValues = make([]scm.JITValueDesc, 148)
		ps151.OverlayValues[1] = d1
		ps151.OverlayValues[2] = d2
		ps151.OverlayValues[3] = d3
		ps151.OverlayValues[4] = d4
		ps151.OverlayValues[5] = d5
		ps151.OverlayValues[6] = d6
		ps151.OverlayValues[8] = d8
		ps151.OverlayValues[9] = d9
		ps151.OverlayValues[10] = d10
		ps151.OverlayValues[11] = d11
		ps151.OverlayValues[12] = d12
		ps151.OverlayValues[13] = d13
		ps151.OverlayValues[16] = d16
		ps151.OverlayValues[17] = d17
		ps151.OverlayValues[35] = d35
		ps151.OverlayValues[36] = d36
		ps151.OverlayValues[37] = d37
		ps151.OverlayValues[38] = d38
		ps151.OverlayValues[39] = d39
		ps151.OverlayValues[41] = d41
		ps151.OverlayValues[42] = d42
		ps151.OverlayValues[43] = d43
		ps151.OverlayValues[44] = d44
		ps151.OverlayValues[45] = d45
		ps151.OverlayValues[46] = d46
		ps151.OverlayValues[47] = d47
		ps151.OverlayValues[48] = d48
		ps151.OverlayValues[49] = d49
		ps151.OverlayValues[50] = d50
		ps151.OverlayValues[51] = d51
		ps151.OverlayValues[52] = d52
		ps151.OverlayValues[53] = d53
		ps151.OverlayValues[54] = d54
		ps151.OverlayValues[55] = d55
		ps151.OverlayValues[56] = d56
		ps151.OverlayValues[57] = d57
		ps151.OverlayValues[58] = d58
		ps151.OverlayValues[59] = d59
		ps151.OverlayValues[60] = d60
		ps151.OverlayValues[61] = d61
		ps151.OverlayValues[62] = d62
		ps151.OverlayValues[63] = d63
		ps151.OverlayValues[64] = d64
		ps151.OverlayValues[65] = d65
		ps151.OverlayValues[66] = d66
		ps151.OverlayValues[67] = d67
		ps151.OverlayValues[68] = d68
		ps151.OverlayValues[69] = d69
		ps151.OverlayValues[70] = d70
		ps151.OverlayValues[71] = d71
		ps151.OverlayValues[72] = d72
		ps151.OverlayValues[73] = d73
		ps151.OverlayValues[74] = d74
		ps151.OverlayValues[75] = d75
		ps151.OverlayValues[76] = d76
		ps151.OverlayValues[77] = d77
		ps151.OverlayValues[78] = d78
		ps151.OverlayValues[141] = d141
		ps151.OverlayValues[142] = d142
		ps151.OverlayValues[143] = d143
		ps151.OverlayValues[144] = d144
		ps151.OverlayValues[145] = d145
		ps151.OverlayValues[146] = d146
		ps151.OverlayValues[147] = d147
		snap152 := d1
		snap153 := d2
		snap154 := d3
		snap155 := d4
		snap156 := d5
		snap157 := d6
		snap158 := d8
		snap159 := d9
		snap160 := d10
		snap161 := d11
		snap162 := d12
		snap163 := d13
		snap164 := d16
		snap165 := d17
		snap166 := d35
		snap167 := d36
		snap168 := d37
		snap169 := d38
		snap170 := d39
		snap171 := d41
		snap172 := d42
		snap173 := d43
		snap174 := d44
		snap175 := d45
		snap176 := d46
		snap177 := d47
		snap178 := d48
		snap179 := d49
		snap180 := d50
		snap181 := d51
		snap182 := d52
		snap183 := d53
		snap184 := d54
		snap185 := d55
		snap186 := d56
		snap187 := d57
		snap188 := d58
		snap189 := d59
		snap190 := d60
		snap191 := d61
		snap192 := d62
		snap193 := d63
		snap194 := d64
		snap195 := d65
		snap196 := d66
		snap197 := d67
		snap198 := d68
		snap199 := d69
		snap200 := d70
		snap201 := d71
		snap202 := d72
		snap203 := d73
		snap204 := d74
		snap205 := d75
		snap206 := d76
		snap207 := d77
		snap208 := d78
		snap209 := d141
		snap210 := d142
		snap211 := d143
		snap212 := d144
		snap213 := d145
		snap214 := d146
		snap215 := d147
		alloc216 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps151)
		}
		ctx.RestoreAllocState(alloc216)
		d1 = snap152
		d2 = snap153
		d3 = snap154
		d4 = snap155
		d5 = snap156
		d6 = snap157
		d8 = snap158
		d9 = snap159
		d10 = snap160
		d11 = snap161
		d12 = snap162
		d13 = snap163
		d16 = snap164
		d17 = snap165
		d35 = snap166
		d36 = snap167
		d37 = snap168
		d38 = snap169
		d39 = snap170
		d41 = snap171
		d42 = snap172
		d43 = snap173
		d44 = snap174
		d45 = snap175
		d46 = snap176
		d47 = snap177
		d48 = snap178
		d49 = snap179
		d50 = snap180
		d51 = snap181
		d52 = snap182
		d53 = snap183
		d54 = snap184
		d55 = snap185
		d56 = snap186
		d57 = snap187
		d58 = snap188
		d59 = snap189
		d60 = snap190
		d61 = snap191
		d62 = snap192
		d63 = snap193
		d64 = snap194
		d65 = snap195
		d66 = snap196
		d67 = snap197
		d68 = snap198
		d69 = snap199
		d70 = snap200
		d71 = snap201
		d72 = snap202
		d73 = snap203
		d74 = snap204
		d75 = snap205
		d76 = snap206
		d77 = snap207
		d78 = snap208
		d141 = snap209
		d142 = snap210
		d143 = snap211
		d144 = snap212
		d145 = snap213
		d146 = snap214
		d147 = snap215
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps150)
		}
		return result
		ctx.FreeDesc(&d146)
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
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
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
		if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
			d144 = ps.OverlayValues[144]
		}
		if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
			d145 = ps.OverlayValues[145]
		}
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d38)
		var d217 scm.JITValueDesc
		if d38.Loc == scm.LocImm {
			d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(scratch, d38.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d217)
		}
		if d217.Loc == scm.LocImm {
			d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: d217.Type, Imm: scm.NewInt(int64(uint64(d217.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d217.Reg, 32)
			ctx.EmitShrRegImm8(d217.Reg, 32)
		}
		if d217.Loc == scm.LocReg && d38.Loc == scm.LocReg && d217.Reg == d38.Reg {
			ctx.TransferReg(d38.Reg)
			d38.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d217)
		ctx.EmitStoreToStack(d217, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d217)
		if ps.General {
		}
		ps218 := scm.PhiState{General: ps.General}
		ps218.OverlayValues = make([]scm.JITValueDesc, 218)
		ps218.OverlayValues[1] = d1
		ps218.OverlayValues[2] = d2
		ps218.OverlayValues[3] = d3
		ps218.OverlayValues[4] = d4
		ps218.OverlayValues[5] = d5
		ps218.OverlayValues[6] = d6
		ps218.OverlayValues[8] = d8
		ps218.OverlayValues[9] = d9
		ps218.OverlayValues[10] = d10
		ps218.OverlayValues[11] = d11
		ps218.OverlayValues[12] = d12
		ps218.OverlayValues[13] = d13
		ps218.OverlayValues[16] = d16
		ps218.OverlayValues[17] = d17
		ps218.OverlayValues[35] = d35
		ps218.OverlayValues[36] = d36
		ps218.OverlayValues[37] = d37
		ps218.OverlayValues[38] = d38
		ps218.OverlayValues[39] = d39
		ps218.OverlayValues[41] = d41
		ps218.OverlayValues[42] = d42
		ps218.OverlayValues[43] = d43
		ps218.OverlayValues[44] = d44
		ps218.OverlayValues[45] = d45
		ps218.OverlayValues[46] = d46
		ps218.OverlayValues[47] = d47
		ps218.OverlayValues[48] = d48
		ps218.OverlayValues[49] = d49
		ps218.OverlayValues[50] = d50
		ps218.OverlayValues[51] = d51
		ps218.OverlayValues[52] = d52
		ps218.OverlayValues[53] = d53
		ps218.OverlayValues[54] = d54
		ps218.OverlayValues[55] = d55
		ps218.OverlayValues[56] = d56
		ps218.OverlayValues[57] = d57
		ps218.OverlayValues[58] = d58
		ps218.OverlayValues[59] = d59
		ps218.OverlayValues[60] = d60
		ps218.OverlayValues[61] = d61
		ps218.OverlayValues[62] = d62
		ps218.OverlayValues[63] = d63
		ps218.OverlayValues[64] = d64
		ps218.OverlayValues[65] = d65
		ps218.OverlayValues[66] = d66
		ps218.OverlayValues[67] = d67
		ps218.OverlayValues[68] = d68
		ps218.OverlayValues[69] = d69
		ps218.OverlayValues[70] = d70
		ps218.OverlayValues[71] = d71
		ps218.OverlayValues[72] = d72
		ps218.OverlayValues[73] = d73
		ps218.OverlayValues[74] = d74
		ps218.OverlayValues[75] = d75
		ps218.OverlayValues[76] = d76
		ps218.OverlayValues[77] = d77
		ps218.OverlayValues[78] = d78
		ps218.OverlayValues[141] = d141
		ps218.OverlayValues[142] = d142
		ps218.OverlayValues[143] = d143
		ps218.OverlayValues[144] = d144
		ps218.OverlayValues[145] = d145
		ps218.OverlayValues[146] = d146
		ps218.OverlayValues[147] = d147
		ps218.OverlayValues[217] = d217
		ps218.PhiValues = make([]scm.JITValueDesc, 2)
		if ps218.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps218)
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
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
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
		if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
			d144 = ps.OverlayValues[144]
		}
		if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
			d145 = ps.OverlayValues[145]
		}
		if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
			d146 = ps.OverlayValues[146]
		}
		if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
			d147 = ps.OverlayValues[147]
		}
		if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
			d217 = ps.OverlayValues[217]
		}
		ctx.ReclaimUntrackedRegs()
		if ps.General {
			ctx.SyncDesc(&d38)
			if d38.Loc == scm.LocReg {
				ctx.ProtectReg(d38.Reg)
			} else if d38.Loc == scm.LocRegPair {
				ctx.ProtectReg(d38.Reg)
				ctx.ProtectReg(d38.Reg2)
			}
			d219 = d38
			if d219.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d219)
			d220 = d219
			if d220.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: d220.Type, Imm: scm.NewInt(int64(uint64(d220.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d220.Reg, 32)
				ctx.EmitShrRegImm8(d220.Reg, 32)
			}
			ctx.EmitStoreToStack(d220, int32(bbs[1].PhiBase)+int32(16))
			if d38.Loc == scm.LocReg {
				ctx.UnprotectReg(d38.Reg)
			} else if d38.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d38.Reg)
				ctx.UnprotectReg(d38.Reg2)
			}
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
		ps221.OverlayValues[39] = d39
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
		ps221.OverlayValues[77] = d77
		ps221.OverlayValues[78] = d78
		ps221.OverlayValues[141] = d141
		ps221.OverlayValues[142] = d142
		ps221.OverlayValues[143] = d143
		ps221.OverlayValues[144] = d144
		ps221.OverlayValues[145] = d145
		ps221.OverlayValues[146] = d146
		ps221.OverlayValues[147] = d147
		ps221.OverlayValues[217] = d217
		ps221.OverlayValues[219] = d219
		ps221.OverlayValues[220] = d220
		ps221.PhiValues = make([]scm.JITValueDesc, 2)
		d222 = d38
		ps221.PhiValues[1] = d222
		if ps221.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps221)
		return result
	}
	ps223 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps223)
	ctx.MarkLabel(lbl0)
	d224 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d224)
	ctx.BindReg(r1, &d224)
	ctx.EmitMovPairToResult(&d224, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
	}
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
		var v scm.Scmer
		if err := json.Unmarshal(scanner.Bytes(), &v); err != nil {
			panic(err)
		}
		s.recids.build(uint32(i), scm.NewInt(int64(k)))
		s.values[i] = v
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
