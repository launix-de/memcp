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
	var d118 scm.JITValueDesc
	_ = d118
	var d119 scm.JITValueDesc
	_ = d119
	var d120 scm.JITValueDesc
	_ = d120
	var d121 scm.JITValueDesc
	_ = d121
	var d122 scm.JITValueDesc
	_ = d122
	var d123 scm.JITValueDesc
	_ = d123
	var d124 scm.JITValueDesc
	_ = d124
	var d183 scm.JITValueDesc
	_ = d183
	var d185 scm.JITValueDesc
	_ = d185
	var d186 scm.JITValueDesc
	_ = d186
	var d188 scm.JITValueDesc
	_ = d188
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	ctx.TrackPointer(unsafe.Pointer(s))
	thisptrPinned := thisptr.Loc == scm.LocReg
	thisptrPinnedReg := thisptr.Reg
	if thisptrPinned {
		ctx.ProtectReg(thisptrPinnedReg)
		defer ctx.UnprotectReg(thisptrPinnedReg)
	}
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
		defer ctx.UnprotectReg(idxPinnedReg)
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
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl11 := ctx.ReserveLabel()
		_ = lbl11
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl11)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d39)
		var d40 scm.JITValueDesc
		if d39.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d39.Imm.Int()))))}
		} else {
			r11 := ctx.AllocReg()
			ctx.EmitMovRegReg(r11, d39.Reg)
			ctx.EmitShlRegImm8(r11, 32)
			ctx.EmitShrRegImm8(r11, 32)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d40)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d41 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
			r12 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r12, fieldAddr)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			ctx.BindReg(r12, &d41)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
			r13 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r13, thisptr.Reg, off)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d41)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d41)
		var d42 scm.JITValueDesc
		if d41.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d41.Imm.Int()))))}
		} else {
			r14 := ctx.AllocReg()
			ctx.EmitMovRegReg(r14, d41.Reg)
			ctx.EmitShlRegImm8(r14, 56)
			ctx.EmitShrRegImm8(r14, 56)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d42)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		ctx.EnsureDesc(&d42)
		ctx.EnsureDescsTogether(&d40, &d42)
		var d43 scm.JITValueDesc
		if d40.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() * d42.Imm.Int())}
		} else if d40.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
			ctx.EmitImulInt64(scratch, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d43)
		} else if d42.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(scratch, d40.Reg)
			if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d42.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d43)
		} else {
			r15 := ctx.AllocRegExcept(d40.Reg, d42.Reg)
			ctx.EmitMovRegReg(r15, d40.Reg)
			ctx.EmitImulInt64(r15, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d43)
		}
		if d43.Loc == scm.LocReg && d40.Loc == scm.LocReg && d43.Reg == d40.Reg {
			ctx.TransferReg(d40.Reg)
			d40.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d40)
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() / 64)}
		} else {
			r16 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r16, d43.Reg)
			ctx.EmitShrRegImm8(r16, 6)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d44)
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		var d45 scm.JITValueDesc
		if d43.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() % 64)}
		} else {
			r17 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r17, d43.Reg)
			ctx.EmitAndRegImm32(r17, 63)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d45)
		}
		if d45.Loc == scm.LocReg && d43.Loc == scm.LocReg && d45.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d43)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d46 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
			r18 := ctx.AllocReg()
			r19 := ctx.AllocRegExcept(r18)
			r20 := ctx.AllocRegExcept(r18, r19)
			ctx.EmitMovRegMem64(r18, fieldAddr)
			ctx.EmitMovRegMem64(r19, fieldAddr+8)
			ctx.EmitMovRegMem64(r20, fieldAddr+16)
			d46 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r18, Reg2: r19, Reg3: r20}
			ctx.BindReg(r18, &d46)
			ctx.BindReg(r19, &d46)
			ctx.BindReg(r20, &d46)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
			r21 := ctx.AllocReg()
			r22 := ctx.AllocRegExcept(r21)
			r23 := ctx.AllocRegExcept(r21, r22)
			ctx.EmitMovRegMem(r21, thisptr.Reg, off)
			ctx.EmitMovRegMem(r22, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r23, thisptr.Reg, off+16)
			d46 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r21, Reg2: r22, Reg3: r23}
			ctx.BindReg(r21, &d46)
			ctx.BindReg(r22, &d46)
			ctx.BindReg(r23, &d46)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		d48 = ctx.EmitSliceElementAddress(&d46, &d44, 8)
		ctx.EnsureDesc(&d48)
		ctx.EmitMovRegMem(d48.Reg, d48.Reg, 0)
		d47 = d48
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.EnsureDesc(&d45)
		var d49 scm.JITValueDesc
		if d47.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d47.Imm.Int()) << uint64(d45.Imm.Int())))}
		} else if d45.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(r24, d47.Reg)
			ctx.EmitShlRegImm8(r24, uint8(d45.Imm.Int()))
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d49)
		} else {
			{
				shiftSrc := d47.Reg
				r25 := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(r25, d47.Reg)
				shiftSrc = r25
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d45.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d45.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d45.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
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
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d44)
		var d50 scm.JITValueDesc
		if d44.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(scratch, d44.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d50)
		}
		if d50.Loc == scm.LocReg && d44.Loc == scm.LocReg && d50.Reg == d44.Reg {
			ctx.TransferReg(d44.Reg)
			d44.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d50)
		ctx.ReclaimUntrackedRegs()
		d52 = ctx.EmitSliceElementAddress(&d46, &d50, 8)
		ctx.EnsureDesc(&d52)
		ctx.EmitMovRegMem(d52.Reg, d52.Reg, 0)
		d51 = d52
		ctx.FreeDesc(&d50)
		ctx.ReclaimUntrackedRegs()
		d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d45)
		ctx.EnsureDescsTogether(&d53, &d45)
		var d54 scm.JITValueDesc
		if d53.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() - d45.Imm.Int())}
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			r26 := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegReg(r26, d53.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d54)
		} else if d53.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
			ctx.EmitSubInt64(scratch, d45.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d54)
		} else if d45.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegReg(scratch, d53.Reg)
			if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d45.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d54)
		} else {
			r27 := ctx.AllocRegExcept(d53.Reg, d45.Reg)
			ctx.EmitMovRegReg(r27, d53.Reg)
			ctx.EmitSubInt64(r27, d45.Reg)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d54)
		}
		if d54.Loc == scm.LocReg && d53.Loc == scm.LocReg && d54.Reg == d53.Reg {
			ctx.TransferReg(d53.Reg)
			d53.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d54)
		var d55 scm.JITValueDesc
		if d51.Loc == scm.LocImm && d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d51.Imm.Int()) >> uint64(d54.Imm.Int())))}
		} else if d54.Loc == scm.LocImm {
			r28 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r28, d51.Reg)
			ctx.EmitShrRegImm8(r28, uint8(d54.Imm.Int()))
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d55)
		} else {
			{
				shiftSrc := d51.Reg
				r29 := ctx.AllocRegExcept(d51.Reg)
				ctx.EmitMovRegReg(r29, d51.Reg)
				shiftSrc = r29
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d54.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d54.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d54.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d55)
			}
		}
		if d55.Loc == scm.LocReg && d51.Loc == scm.LocReg && d55.Reg == d51.Reg {
			ctx.TransferReg(d51.Reg)
			d51.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d51)
		ctx.FreeDesc(&d54)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d49)
		ctx.EnsureDesc(&d55)
		var d56 scm.JITValueDesc
		if d49.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() | d55.Imm.Int())}
		} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d55.Reg}
			ctx.BindReg(d55.Reg, &d56)
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			r30 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegReg(r30, d49.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d56)
		} else if d49.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
			ctx.EmitOrInt64(scratch, d55.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d56)
		} else if d55.Loc == scm.LocImm {
			r31 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegReg(r31, d49.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r31, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitOrInt64(r31, scm.RegR11)
			}
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d56)
		} else {
			r32 := ctx.AllocRegExcept(d49.Reg, d55.Reg)
			ctx.EmitMovRegReg(r32, d49.Reg)
			ctx.EmitOrInt64(r32, d55.Reg)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d56)
		}
		if d56.Loc == scm.LocReg && d49.Loc == scm.LocReg && d56.Reg == d49.Reg {
			ctx.TransferReg(d49.Reg)
			d49.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d49)
		ctx.FreeDesc(&d55)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d41)
		var d57 scm.JITValueDesc
		if d41.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d41.Imm.Int()))))}
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitMovRegReg(r33, d41.Reg)
			ctx.EmitShlRegImm8(r33, 56)
			ctx.EmitShrRegImm8(r33, 56)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d57)
		}
		ctx.ReclaimUntrackedRegs()
		d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d58, &d57)
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d56)
		ctx.EnsureDesc(&d59)
		var d60 scm.JITValueDesc
		if d56.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d56.Imm.Int()) >> uint64(d59.Imm.Int())))}
		} else if d59.Loc == scm.LocImm {
			r36 := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegReg(r36, d56.Reg)
			ctx.EmitShrRegImm8(r36, uint8(d59.Imm.Int()))
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d60)
		} else {
			{
				shiftSrc := d56.Reg
				r37 := ctx.AllocRegExcept(d56.Reg)
				ctx.EmitMovRegReg(r37, d56.Reg)
				shiftSrc = r37
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d59.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d59.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d59.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d60)
			}
		}
		if d60.Loc == scm.LocReg && d56.Loc == scm.LocReg && d60.Reg == d56.Reg {
			ctx.TransferReg(d56.Reg)
			d56.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d56)
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d60)
		var d61 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
			r38 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r38, fieldAddr)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d61)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
			r39 := ctx.AllocReg()
			ctx.EmitMovRegMem(r39, thisptr.Reg, off)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.BindReg(r39, &d61)
		}
		ctx.EnsureDesc(&d61)
		ctx.EnsureDesc(&d61)
		var d62 scm.JITValueDesc
		if d61.Loc == scm.LocImm {
			d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d61.Imm.Int()))))}
		} else {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegReg(r40, d61.Reg)
			d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d62)
		}
		ctx.EnsureDesc(&d60)
		ctx.EnsureDesc(&d62)
		ctx.EnsureDescsTogether(&d60, &d62)
		var d63 scm.JITValueDesc
		if d60.Loc == scm.LocImm && d62.Loc == scm.LocImm {
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d60.Imm.Int() + d62.Imm.Int())}
		} else if d62.Loc == scm.LocImm && d62.Imm.Int() == 0 {
			r41 := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegReg(r41, d60.Reg)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d63)
		} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d62.Reg}
			ctx.BindReg(d62.Reg, &d63)
		} else if d60.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d62.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d60.Imm.Int()))
			ctx.EmitAddInt64(scratch, d62.Reg)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d63)
		} else if d62.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegReg(scratch, d60.Reg)
			if d62.Imm.Int() >= -2147483648 && d62.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d62.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d62.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d63)
		} else {
			r42 := ctx.AllocRegExcept(d60.Reg, d62.Reg)
			ctx.EmitMovRegReg(r42, d60.Reg)
			ctx.EmitAddInt64(r42, d62.Reg)
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d63)
		}
		if d63.Loc == scm.LocReg && d60.Loc == scm.LocReg && d63.Reg == d60.Reg {
			ctx.TransferReg(d60.Reg)
			d60.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d63)
		ctx.FreeDesc(&d60)
		ctx.FreeDesc(&d62)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d64 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r43 := ctx.AllocReg()
			ctx.EmitMovRegReg(r43, idxInt.Reg)
			ctx.EmitShlRegImm8(r43, 32)
			ctx.EmitShrRegImm8(r43, 32)
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d64)
		}
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d64)
		ctx.EnsureDescsTogether(&d63, &d64)
		var d65 scm.JITValueDesc
		if d63.Loc == scm.LocImm && d64.Loc == scm.LocImm {
			d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d63.Imm.Int()) == uint64(d64.Imm.Int()))}
		} else if d64.Loc == scm.LocImm {
			r44 := ctx.AllocRegExcept(d63.Reg)
			if d64.Imm.Int() >= -2147483648 && d64.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d63.Reg, int32(d64.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d64.Imm.Int()))
				ctx.EmitCmpInt64(d63.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r44, scm.CondEqual)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r44}
			ctx.BindReg(r44, &d65)
		} else if d63.Loc == scm.LocImm {
			r45 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d64.Reg)
			ctx.EmitSetcc(r45, scm.CondEqual)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r45}
			ctx.BindReg(r45, &d65)
		} else {
			r46 := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitCmpInt64(d63.Reg, d64.Reg)
			ctx.EmitSetcc(r46, scm.CondEqual)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
			ctx.BindReg(r46, &d65)
		}
		ctx.FreeDesc(&d64)
		d66 = d65
		ctx.EnsureDesc(&d66)
		if d66.Loc != scm.LocImm && d66.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d66.Loc == scm.LocImm {
			if d66.Imm.Bool() {
				if ps.General {
				}
				ps67 := scm.PhiState{General: ps.General}
				ps67.OverlayValues = make([]scm.JITValueDesc, 67)
				ps67.OverlayValues[1] = d1
				ps67.OverlayValues[2] = d2
				ps67.OverlayValues[3] = d3
				ps67.OverlayValues[4] = d4
				ps67.OverlayValues[5] = d5
				ps67.OverlayValues[6] = d6
				ps67.OverlayValues[8] = d8
				ps67.OverlayValues[9] = d9
				ps67.OverlayValues[10] = d10
				ps67.OverlayValues[11] = d11
				ps67.OverlayValues[12] = d12
				ps67.OverlayValues[13] = d13
				ps67.OverlayValues[16] = d16
				ps67.OverlayValues[17] = d17
				ps67.OverlayValues[35] = d35
				ps67.OverlayValues[36] = d36
				ps67.OverlayValues[37] = d37
				ps67.OverlayValues[38] = d38
				ps67.OverlayValues[39] = d39
				ps67.OverlayValues[40] = d40
				ps67.OverlayValues[41] = d41
				ps67.OverlayValues[42] = d42
				ps67.OverlayValues[43] = d43
				ps67.OverlayValues[44] = d44
				ps67.OverlayValues[45] = d45
				ps67.OverlayValues[46] = d46
				ps67.OverlayValues[47] = d47
				ps67.OverlayValues[48] = d48
				ps67.OverlayValues[49] = d49
				ps67.OverlayValues[50] = d50
				ps67.OverlayValues[51] = d51
				ps67.OverlayValues[52] = d52
				ps67.OverlayValues[53] = d53
				ps67.OverlayValues[54] = d54
				ps67.OverlayValues[55] = d55
				ps67.OverlayValues[56] = d56
				ps67.OverlayValues[57] = d57
				ps67.OverlayValues[58] = d58
				ps67.OverlayValues[59] = d59
				ps67.OverlayValues[60] = d60
				ps67.OverlayValues[61] = d61
				ps67.OverlayValues[62] = d62
				ps67.OverlayValues[63] = d63
				ps67.OverlayValues[64] = d64
				ps67.OverlayValues[65] = d65
				ps67.OverlayValues[66] = d66
				return bbs[4].RenderPS(ps67)
			}
			if ps.General {
			}
			ps68 := scm.PhiState{General: ps.General}
			ps68.OverlayValues = make([]scm.JITValueDesc, 67)
			ps68.OverlayValues[1] = d1
			ps68.OverlayValues[2] = d2
			ps68.OverlayValues[3] = d3
			ps68.OverlayValues[4] = d4
			ps68.OverlayValues[5] = d5
			ps68.OverlayValues[6] = d6
			ps68.OverlayValues[8] = d8
			ps68.OverlayValues[9] = d9
			ps68.OverlayValues[10] = d10
			ps68.OverlayValues[11] = d11
			ps68.OverlayValues[12] = d12
			ps68.OverlayValues[13] = d13
			ps68.OverlayValues[16] = d16
			ps68.OverlayValues[17] = d17
			ps68.OverlayValues[35] = d35
			ps68.OverlayValues[36] = d36
			ps68.OverlayValues[37] = d37
			ps68.OverlayValues[38] = d38
			ps68.OverlayValues[39] = d39
			ps68.OverlayValues[40] = d40
			ps68.OverlayValues[41] = d41
			ps68.OverlayValues[42] = d42
			ps68.OverlayValues[43] = d43
			ps68.OverlayValues[44] = d44
			ps68.OverlayValues[45] = d45
			ps68.OverlayValues[46] = d46
			ps68.OverlayValues[47] = d47
			ps68.OverlayValues[48] = d48
			ps68.OverlayValues[49] = d49
			ps68.OverlayValues[50] = d50
			ps68.OverlayValues[51] = d51
			ps68.OverlayValues[52] = d52
			ps68.OverlayValues[53] = d53
			ps68.OverlayValues[54] = d54
			ps68.OverlayValues[55] = d55
			ps68.OverlayValues[56] = d56
			ps68.OverlayValues[57] = d57
			ps68.OverlayValues[58] = d58
			ps68.OverlayValues[59] = d59
			ps68.OverlayValues[60] = d60
			ps68.OverlayValues[61] = d61
			ps68.OverlayValues[62] = d62
			ps68.OverlayValues[63] = d63
			ps68.OverlayValues[64] = d64
			ps68.OverlayValues[65] = d65
			ps68.OverlayValues[66] = d66
			return bbs[5].RenderPS(ps68)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d66.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl12)
		ctx.EmitJmp(lbl13)
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl5)
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl6)
		ps69 := scm.PhiState{General: true}
		ps69.OverlayValues = make([]scm.JITValueDesc, 67)
		ps69.OverlayValues[1] = d1
		ps69.OverlayValues[2] = d2
		ps69.OverlayValues[3] = d3
		ps69.OverlayValues[4] = d4
		ps69.OverlayValues[5] = d5
		ps69.OverlayValues[6] = d6
		ps69.OverlayValues[8] = d8
		ps69.OverlayValues[9] = d9
		ps69.OverlayValues[10] = d10
		ps69.OverlayValues[11] = d11
		ps69.OverlayValues[12] = d12
		ps69.OverlayValues[13] = d13
		ps69.OverlayValues[16] = d16
		ps69.OverlayValues[17] = d17
		ps69.OverlayValues[35] = d35
		ps69.OverlayValues[36] = d36
		ps69.OverlayValues[37] = d37
		ps69.OverlayValues[38] = d38
		ps69.OverlayValues[39] = d39
		ps69.OverlayValues[40] = d40
		ps69.OverlayValues[41] = d41
		ps69.OverlayValues[42] = d42
		ps69.OverlayValues[43] = d43
		ps69.OverlayValues[44] = d44
		ps69.OverlayValues[45] = d45
		ps69.OverlayValues[46] = d46
		ps69.OverlayValues[47] = d47
		ps69.OverlayValues[48] = d48
		ps69.OverlayValues[49] = d49
		ps69.OverlayValues[50] = d50
		ps69.OverlayValues[51] = d51
		ps69.OverlayValues[52] = d52
		ps69.OverlayValues[53] = d53
		ps69.OverlayValues[54] = d54
		ps69.OverlayValues[55] = d55
		ps69.OverlayValues[56] = d56
		ps69.OverlayValues[57] = d57
		ps69.OverlayValues[58] = d58
		ps69.OverlayValues[59] = d59
		ps69.OverlayValues[60] = d60
		ps69.OverlayValues[61] = d61
		ps69.OverlayValues[62] = d62
		ps69.OverlayValues[63] = d63
		ps69.OverlayValues[64] = d64
		ps69.OverlayValues[65] = d65
		ps69.OverlayValues[66] = d66
		ps70 := scm.PhiState{General: true}
		ps70.OverlayValues = make([]scm.JITValueDesc, 67)
		ps70.OverlayValues[1] = d1
		ps70.OverlayValues[2] = d2
		ps70.OverlayValues[3] = d3
		ps70.OverlayValues[4] = d4
		ps70.OverlayValues[5] = d5
		ps70.OverlayValues[6] = d6
		ps70.OverlayValues[8] = d8
		ps70.OverlayValues[9] = d9
		ps70.OverlayValues[10] = d10
		ps70.OverlayValues[11] = d11
		ps70.OverlayValues[12] = d12
		ps70.OverlayValues[13] = d13
		ps70.OverlayValues[16] = d16
		ps70.OverlayValues[17] = d17
		ps70.OverlayValues[35] = d35
		ps70.OverlayValues[36] = d36
		ps70.OverlayValues[37] = d37
		ps70.OverlayValues[38] = d38
		ps70.OverlayValues[39] = d39
		ps70.OverlayValues[40] = d40
		ps70.OverlayValues[41] = d41
		ps70.OverlayValues[42] = d42
		ps70.OverlayValues[43] = d43
		ps70.OverlayValues[44] = d44
		ps70.OverlayValues[45] = d45
		ps70.OverlayValues[46] = d46
		ps70.OverlayValues[47] = d47
		ps70.OverlayValues[48] = d48
		ps70.OverlayValues[49] = d49
		ps70.OverlayValues[50] = d50
		ps70.OverlayValues[51] = d51
		ps70.OverlayValues[52] = d52
		ps70.OverlayValues[53] = d53
		ps70.OverlayValues[54] = d54
		ps70.OverlayValues[55] = d55
		ps70.OverlayValues[56] = d56
		ps70.OverlayValues[57] = d57
		ps70.OverlayValues[58] = d58
		ps70.OverlayValues[59] = d59
		ps70.OverlayValues[60] = d60
		ps70.OverlayValues[61] = d61
		ps70.OverlayValues[62] = d62
		ps70.OverlayValues[63] = d63
		ps70.OverlayValues[64] = d64
		ps70.OverlayValues[65] = d65
		ps70.OverlayValues[66] = d66
		snap71 := d1
		snap72 := d2
		snap73 := d3
		snap74 := d4
		snap75 := d5
		snap76 := d6
		snap77 := d8
		snap78 := d9
		snap79 := d10
		snap80 := d11
		snap81 := d12
		snap82 := d13
		snap83 := d16
		snap84 := d17
		snap85 := d35
		snap86 := d36
		snap87 := d37
		snap88 := d38
		snap89 := d39
		snap90 := d40
		snap91 := d41
		snap92 := d42
		snap93 := d43
		snap94 := d44
		snap95 := d45
		snap96 := d46
		snap97 := d47
		snap98 := d48
		snap99 := d49
		snap100 := d50
		snap101 := d51
		snap102 := d52
		snap103 := d53
		snap104 := d54
		snap105 := d55
		snap106 := d56
		snap107 := d57
		snap108 := d58
		snap109 := d59
		snap110 := d60
		snap111 := d61
		snap112 := d62
		snap113 := d63
		snap114 := d64
		snap115 := d65
		snap116 := d66
		alloc117 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps70)
		}
		ctx.RestoreAllocState(alloc117)
		d1 = snap71
		d2 = snap72
		d3 = snap73
		d4 = snap74
		d5 = snap75
		d6 = snap76
		d8 = snap77
		d9 = snap78
		d10 = snap79
		d11 = snap80
		d12 = snap81
		d13 = snap82
		d16 = snap83
		d17 = snap84
		d35 = snap85
		d36 = snap86
		d37 = snap87
		d38 = snap88
		d39 = snap89
		d40 = snap90
		d41 = snap91
		d42 = snap92
		d43 = snap93
		d44 = snap94
		d45 = snap95
		d46 = snap96
		d47 = snap97
		d48 = snap98
		d49 = snap99
		d50 = snap100
		d51 = snap101
		d52 = snap102
		d53 = snap103
		d54 = snap104
		d55 = snap105
		d56 = snap106
		d57 = snap107
		d58 = snap108
		d59 = snap109
		d60 = snap110
		d61 = snap111
		d62 = snap112
		d63 = snap113
		d64 = snap114
		d65 = snap115
		d66 = snap116
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps69)
		}
		return result
		ctx.FreeDesc(&d65)
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
		ctx.ReclaimUntrackedRegs()
		var d118 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
			r47 := ctx.AllocReg()
			r48 := ctx.AllocRegExcept(r47)
			r49 := ctx.AllocRegExcept(r47, r48)
			ctx.EmitMovRegMem64(r47, fieldAddr)
			ctx.EmitMovRegMem64(r48, fieldAddr+8)
			ctx.EmitMovRegMem64(r49, fieldAddr+16)
			d118 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r47, Reg2: r48, Reg3: r49}
			ctx.BindReg(r47, &d118)
			ctx.BindReg(r48, &d118)
			ctx.BindReg(r49, &d118)
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
			r50 := ctx.AllocReg()
			r51 := ctx.AllocRegExcept(r50)
			r52 := ctx.AllocRegExcept(r50, r51)
			ctx.EmitMovRegMem(r50, thisptr.Reg, off)
			ctx.EmitMovRegMem(r51, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r52, thisptr.Reg, off+16)
			d118 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r50, Reg2: r51, Reg3: r52}
			ctx.BindReg(r50, &d118)
			ctx.BindReg(r51, &d118)
			ctx.BindReg(r52, &d118)
		}
		ctx.EnsureDesc(&d38)
		d120 = ctx.EmitSliceElementAddress(&d118, &d38, 16)
		ctx.EnsureDesc(&d120)
		r53 := ctx.AllocRegExcept(d120.Reg)
		ctx.EmitMovRegMem(r53, d120.Reg, 8)
		ctx.EmitMovRegMem(d120.Reg, d120.Reg, 0)
		d119 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d120.Reg, Reg2: r53}
		ctx.BindReg(d120.Reg, &d119)
		ctx.BindReg(r53, &d119)
		d121 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d121)
		ctx.BindReg(r1, &d121)
		ctx.EnsureDesc(&d119)
		if d119.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d119, &d121)
		} else {
			switch d119.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d121, d119)
			case scm.TagInt:
				ctx.EmitMakeInt(d121, d119)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d121, d119)
			case scm.TagNil:
				ctx.EmitMakeNil(d121)
			default:
				ctx.EmitMovPairToResult(&d119, &d121)
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
			d121 = ps.OverlayValues[121]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d122 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r54 := ctx.AllocReg()
			ctx.EmitMovRegReg(r54, idxInt.Reg)
			ctx.EmitShlRegImm8(r54, 32)
			ctx.EmitShrRegImm8(r54, 32)
			d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d122)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d122)
		ctx.EnsureDescsTogether(&d63, &d122)
		var d123 scm.JITValueDesc
		if d63.Loc == scm.LocImm && d122.Loc == scm.LocImm {
			d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d63.Imm.Int()) < uint64(d122.Imm.Int()))}
		} else if d122.Loc == scm.LocImm {
			r55 := ctx.AllocRegExcept(d63.Reg)
			if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d63.Reg, int32(d122.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
				ctx.EmitCmpInt64(d63.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r55, scm.CondUnsignedBelow)
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
			ctx.BindReg(r55, &d123)
		} else if d63.Loc == scm.LocImm {
			r56 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d122.Reg)
			ctx.EmitSetcc(r56, scm.CondUnsignedBelow)
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
			ctx.BindReg(r56, &d123)
		} else {
			r57 := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitCmpInt64(d63.Reg, d122.Reg)
			ctx.EmitSetcc(r57, scm.CondUnsignedBelow)
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
			ctx.BindReg(r57, &d123)
		}
		ctx.FreeDesc(&d63)
		ctx.FreeDesc(&d122)
		d124 = d123
		ctx.EnsureDesc(&d124)
		if d124.Loc != scm.LocImm && d124.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d124.Loc == scm.LocImm {
			if d124.Imm.Bool() {
				if ps.General {
				}
				ps125 := scm.PhiState{General: ps.General}
				ps125.OverlayValues = make([]scm.JITValueDesc, 125)
				ps125.OverlayValues[1] = d1
				ps125.OverlayValues[2] = d2
				ps125.OverlayValues[3] = d3
				ps125.OverlayValues[4] = d4
				ps125.OverlayValues[5] = d5
				ps125.OverlayValues[6] = d6
				ps125.OverlayValues[8] = d8
				ps125.OverlayValues[9] = d9
				ps125.OverlayValues[10] = d10
				ps125.OverlayValues[11] = d11
				ps125.OverlayValues[12] = d12
				ps125.OverlayValues[13] = d13
				ps125.OverlayValues[16] = d16
				ps125.OverlayValues[17] = d17
				ps125.OverlayValues[35] = d35
				ps125.OverlayValues[36] = d36
				ps125.OverlayValues[37] = d37
				ps125.OverlayValues[38] = d38
				ps125.OverlayValues[39] = d39
				ps125.OverlayValues[40] = d40
				ps125.OverlayValues[41] = d41
				ps125.OverlayValues[42] = d42
				ps125.OverlayValues[43] = d43
				ps125.OverlayValues[44] = d44
				ps125.OverlayValues[45] = d45
				ps125.OverlayValues[46] = d46
				ps125.OverlayValues[47] = d47
				ps125.OverlayValues[48] = d48
				ps125.OverlayValues[49] = d49
				ps125.OverlayValues[50] = d50
				ps125.OverlayValues[51] = d51
				ps125.OverlayValues[52] = d52
				ps125.OverlayValues[53] = d53
				ps125.OverlayValues[54] = d54
				ps125.OverlayValues[55] = d55
				ps125.OverlayValues[56] = d56
				ps125.OverlayValues[57] = d57
				ps125.OverlayValues[58] = d58
				ps125.OverlayValues[59] = d59
				ps125.OverlayValues[60] = d60
				ps125.OverlayValues[61] = d61
				ps125.OverlayValues[62] = d62
				ps125.OverlayValues[63] = d63
				ps125.OverlayValues[64] = d64
				ps125.OverlayValues[65] = d65
				ps125.OverlayValues[66] = d66
				ps125.OverlayValues[118] = d118
				ps125.OverlayValues[119] = d119
				ps125.OverlayValues[120] = d120
				ps125.OverlayValues[121] = d121
				ps125.OverlayValues[122] = d122
				ps125.OverlayValues[123] = d123
				ps125.OverlayValues[124] = d124
				return bbs[6].RenderPS(ps125)
			}
			if ps.General {
			}
			ps126 := scm.PhiState{General: ps.General}
			ps126.OverlayValues = make([]scm.JITValueDesc, 125)
			ps126.OverlayValues[1] = d1
			ps126.OverlayValues[2] = d2
			ps126.OverlayValues[3] = d3
			ps126.OverlayValues[4] = d4
			ps126.OverlayValues[5] = d5
			ps126.OverlayValues[6] = d6
			ps126.OverlayValues[8] = d8
			ps126.OverlayValues[9] = d9
			ps126.OverlayValues[10] = d10
			ps126.OverlayValues[11] = d11
			ps126.OverlayValues[12] = d12
			ps126.OverlayValues[13] = d13
			ps126.OverlayValues[16] = d16
			ps126.OverlayValues[17] = d17
			ps126.OverlayValues[35] = d35
			ps126.OverlayValues[36] = d36
			ps126.OverlayValues[37] = d37
			ps126.OverlayValues[38] = d38
			ps126.OverlayValues[39] = d39
			ps126.OverlayValues[40] = d40
			ps126.OverlayValues[41] = d41
			ps126.OverlayValues[42] = d42
			ps126.OverlayValues[43] = d43
			ps126.OverlayValues[44] = d44
			ps126.OverlayValues[45] = d45
			ps126.OverlayValues[46] = d46
			ps126.OverlayValues[47] = d47
			ps126.OverlayValues[48] = d48
			ps126.OverlayValues[49] = d49
			ps126.OverlayValues[50] = d50
			ps126.OverlayValues[51] = d51
			ps126.OverlayValues[52] = d52
			ps126.OverlayValues[53] = d53
			ps126.OverlayValues[54] = d54
			ps126.OverlayValues[55] = d55
			ps126.OverlayValues[56] = d56
			ps126.OverlayValues[57] = d57
			ps126.OverlayValues[58] = d58
			ps126.OverlayValues[59] = d59
			ps126.OverlayValues[60] = d60
			ps126.OverlayValues[61] = d61
			ps126.OverlayValues[62] = d62
			ps126.OverlayValues[63] = d63
			ps126.OverlayValues[64] = d64
			ps126.OverlayValues[65] = d65
			ps126.OverlayValues[66] = d66
			ps126.OverlayValues[118] = d118
			ps126.OverlayValues[119] = d119
			ps126.OverlayValues[120] = d120
			ps126.OverlayValues[121] = d121
			ps126.OverlayValues[122] = d122
			ps126.OverlayValues[123] = d123
			ps126.OverlayValues[124] = d124
			return bbs[7].RenderPS(ps126)
		}
		if !ps.General {
			ps.General = true
			return bbs[5].RenderPS(ps)
		}
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d124.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		ctx.EmitJmp(lbl15)
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl7)
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl8)
		ps127 := scm.PhiState{General: true}
		ps127.OverlayValues = make([]scm.JITValueDesc, 125)
		ps127.OverlayValues[1] = d1
		ps127.OverlayValues[2] = d2
		ps127.OverlayValues[3] = d3
		ps127.OverlayValues[4] = d4
		ps127.OverlayValues[5] = d5
		ps127.OverlayValues[6] = d6
		ps127.OverlayValues[8] = d8
		ps127.OverlayValues[9] = d9
		ps127.OverlayValues[10] = d10
		ps127.OverlayValues[11] = d11
		ps127.OverlayValues[12] = d12
		ps127.OverlayValues[13] = d13
		ps127.OverlayValues[16] = d16
		ps127.OverlayValues[17] = d17
		ps127.OverlayValues[35] = d35
		ps127.OverlayValues[36] = d36
		ps127.OverlayValues[37] = d37
		ps127.OverlayValues[38] = d38
		ps127.OverlayValues[39] = d39
		ps127.OverlayValues[40] = d40
		ps127.OverlayValues[41] = d41
		ps127.OverlayValues[42] = d42
		ps127.OverlayValues[43] = d43
		ps127.OverlayValues[44] = d44
		ps127.OverlayValues[45] = d45
		ps127.OverlayValues[46] = d46
		ps127.OverlayValues[47] = d47
		ps127.OverlayValues[48] = d48
		ps127.OverlayValues[49] = d49
		ps127.OverlayValues[50] = d50
		ps127.OverlayValues[51] = d51
		ps127.OverlayValues[52] = d52
		ps127.OverlayValues[53] = d53
		ps127.OverlayValues[54] = d54
		ps127.OverlayValues[55] = d55
		ps127.OverlayValues[56] = d56
		ps127.OverlayValues[57] = d57
		ps127.OverlayValues[58] = d58
		ps127.OverlayValues[59] = d59
		ps127.OverlayValues[60] = d60
		ps127.OverlayValues[61] = d61
		ps127.OverlayValues[62] = d62
		ps127.OverlayValues[63] = d63
		ps127.OverlayValues[64] = d64
		ps127.OverlayValues[65] = d65
		ps127.OverlayValues[66] = d66
		ps127.OverlayValues[118] = d118
		ps127.OverlayValues[119] = d119
		ps127.OverlayValues[120] = d120
		ps127.OverlayValues[121] = d121
		ps127.OverlayValues[122] = d122
		ps127.OverlayValues[123] = d123
		ps127.OverlayValues[124] = d124
		ps128 := scm.PhiState{General: true}
		ps128.OverlayValues = make([]scm.JITValueDesc, 125)
		ps128.OverlayValues[1] = d1
		ps128.OverlayValues[2] = d2
		ps128.OverlayValues[3] = d3
		ps128.OverlayValues[4] = d4
		ps128.OverlayValues[5] = d5
		ps128.OverlayValues[6] = d6
		ps128.OverlayValues[8] = d8
		ps128.OverlayValues[9] = d9
		ps128.OverlayValues[10] = d10
		ps128.OverlayValues[11] = d11
		ps128.OverlayValues[12] = d12
		ps128.OverlayValues[13] = d13
		ps128.OverlayValues[16] = d16
		ps128.OverlayValues[17] = d17
		ps128.OverlayValues[35] = d35
		ps128.OverlayValues[36] = d36
		ps128.OverlayValues[37] = d37
		ps128.OverlayValues[38] = d38
		ps128.OverlayValues[39] = d39
		ps128.OverlayValues[40] = d40
		ps128.OverlayValues[41] = d41
		ps128.OverlayValues[42] = d42
		ps128.OverlayValues[43] = d43
		ps128.OverlayValues[44] = d44
		ps128.OverlayValues[45] = d45
		ps128.OverlayValues[46] = d46
		ps128.OverlayValues[47] = d47
		ps128.OverlayValues[48] = d48
		ps128.OverlayValues[49] = d49
		ps128.OverlayValues[50] = d50
		ps128.OverlayValues[51] = d51
		ps128.OverlayValues[52] = d52
		ps128.OverlayValues[53] = d53
		ps128.OverlayValues[54] = d54
		ps128.OverlayValues[55] = d55
		ps128.OverlayValues[56] = d56
		ps128.OverlayValues[57] = d57
		ps128.OverlayValues[58] = d58
		ps128.OverlayValues[59] = d59
		ps128.OverlayValues[60] = d60
		ps128.OverlayValues[61] = d61
		ps128.OverlayValues[62] = d62
		ps128.OverlayValues[63] = d63
		ps128.OverlayValues[64] = d64
		ps128.OverlayValues[65] = d65
		ps128.OverlayValues[66] = d66
		ps128.OverlayValues[118] = d118
		ps128.OverlayValues[119] = d119
		ps128.OverlayValues[120] = d120
		ps128.OverlayValues[121] = d121
		ps128.OverlayValues[122] = d122
		ps128.OverlayValues[123] = d123
		ps128.OverlayValues[124] = d124
		snap129 := d1
		snap130 := d2
		snap131 := d3
		snap132 := d4
		snap133 := d5
		snap134 := d6
		snap135 := d8
		snap136 := d9
		snap137 := d10
		snap138 := d11
		snap139 := d12
		snap140 := d13
		snap141 := d16
		snap142 := d17
		snap143 := d35
		snap144 := d36
		snap145 := d37
		snap146 := d38
		snap147 := d39
		snap148 := d40
		snap149 := d41
		snap150 := d42
		snap151 := d43
		snap152 := d44
		snap153 := d45
		snap154 := d46
		snap155 := d47
		snap156 := d48
		snap157 := d49
		snap158 := d50
		snap159 := d51
		snap160 := d52
		snap161 := d53
		snap162 := d54
		snap163 := d55
		snap164 := d56
		snap165 := d57
		snap166 := d58
		snap167 := d59
		snap168 := d60
		snap169 := d61
		snap170 := d62
		snap171 := d63
		snap172 := d64
		snap173 := d65
		snap174 := d66
		snap175 := d118
		snap176 := d119
		snap177 := d120
		snap178 := d121
		snap179 := d122
		snap180 := d123
		snap181 := d124
		alloc182 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps128)
		}
		ctx.RestoreAllocState(alloc182)
		d1 = snap129
		d2 = snap130
		d3 = snap131
		d4 = snap132
		d5 = snap133
		d6 = snap134
		d8 = snap135
		d9 = snap136
		d10 = snap137
		d11 = snap138
		d12 = snap139
		d13 = snap140
		d16 = snap141
		d17 = snap142
		d35 = snap143
		d36 = snap144
		d37 = snap145
		d38 = snap146
		d39 = snap147
		d40 = snap148
		d41 = snap149
		d42 = snap150
		d43 = snap151
		d44 = snap152
		d45 = snap153
		d46 = snap154
		d47 = snap155
		d48 = snap156
		d49 = snap157
		d50 = snap158
		d51 = snap159
		d52 = snap160
		d53 = snap161
		d54 = snap162
		d55 = snap163
		d56 = snap164
		d57 = snap165
		d58 = snap166
		d59 = snap167
		d60 = snap168
		d61 = snap169
		d62 = snap170
		d63 = snap171
		d64 = snap172
		d65 = snap173
		d66 = snap174
		d118 = snap175
		d119 = snap176
		d120 = snap177
		d121 = snap178
		d122 = snap179
		d123 = snap180
		d124 = snap181
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps127)
		}
		return result
		ctx.FreeDesc(&d123)
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
			d121 = ps.OverlayValues[121]
		}
		if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
			d122 = ps.OverlayValues[122]
		}
		if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
			d123 = ps.OverlayValues[123]
		}
		if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
			d124 = ps.OverlayValues[124]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d38)
		var d183 scm.JITValueDesc
		if d38.Loc == scm.LocImm {
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(scratch, d38.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d183)
		}
		if d183.Loc == scm.LocImm {
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: d183.Type, Imm: scm.NewInt(int64(uint64(d183.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d183.Reg, 32)
			ctx.EmitShrRegImm8(d183.Reg, 32)
		}
		if d183.Loc == scm.LocReg && d38.Loc == scm.LocReg && d183.Reg == d38.Reg {
			ctx.TransferReg(d38.Reg)
			d38.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d183)
		ctx.EmitStoreToStack(d183, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d183)
		if ps.General {
		}
		ps184 := scm.PhiState{General: ps.General}
		ps184.OverlayValues = make([]scm.JITValueDesc, 184)
		ps184.OverlayValues[1] = d1
		ps184.OverlayValues[2] = d2
		ps184.OverlayValues[3] = d3
		ps184.OverlayValues[4] = d4
		ps184.OverlayValues[5] = d5
		ps184.OverlayValues[6] = d6
		ps184.OverlayValues[8] = d8
		ps184.OverlayValues[9] = d9
		ps184.OverlayValues[10] = d10
		ps184.OverlayValues[11] = d11
		ps184.OverlayValues[12] = d12
		ps184.OverlayValues[13] = d13
		ps184.OverlayValues[16] = d16
		ps184.OverlayValues[17] = d17
		ps184.OverlayValues[35] = d35
		ps184.OverlayValues[36] = d36
		ps184.OverlayValues[37] = d37
		ps184.OverlayValues[38] = d38
		ps184.OverlayValues[39] = d39
		ps184.OverlayValues[40] = d40
		ps184.OverlayValues[41] = d41
		ps184.OverlayValues[42] = d42
		ps184.OverlayValues[43] = d43
		ps184.OverlayValues[44] = d44
		ps184.OverlayValues[45] = d45
		ps184.OverlayValues[46] = d46
		ps184.OverlayValues[47] = d47
		ps184.OverlayValues[48] = d48
		ps184.OverlayValues[49] = d49
		ps184.OverlayValues[50] = d50
		ps184.OverlayValues[51] = d51
		ps184.OverlayValues[52] = d52
		ps184.OverlayValues[53] = d53
		ps184.OverlayValues[54] = d54
		ps184.OverlayValues[55] = d55
		ps184.OverlayValues[56] = d56
		ps184.OverlayValues[57] = d57
		ps184.OverlayValues[58] = d58
		ps184.OverlayValues[59] = d59
		ps184.OverlayValues[60] = d60
		ps184.OverlayValues[61] = d61
		ps184.OverlayValues[62] = d62
		ps184.OverlayValues[63] = d63
		ps184.OverlayValues[64] = d64
		ps184.OverlayValues[65] = d65
		ps184.OverlayValues[66] = d66
		ps184.OverlayValues[118] = d118
		ps184.OverlayValues[119] = d119
		ps184.OverlayValues[120] = d120
		ps184.OverlayValues[121] = d121
		ps184.OverlayValues[122] = d122
		ps184.OverlayValues[123] = d123
		ps184.OverlayValues[124] = d124
		ps184.OverlayValues[183] = d183
		ps184.PhiValues = make([]scm.JITValueDesc, 2)
		if ps184.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps184)
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
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
			d121 = ps.OverlayValues[121]
		}
		if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
			d122 = ps.OverlayValues[122]
		}
		if len(ps.OverlayValues) > 123 && ps.OverlayValues[123].Loc != scm.LocNone {
			d123 = ps.OverlayValues[123]
		}
		if len(ps.OverlayValues) > 124 && ps.OverlayValues[124].Loc != scm.LocNone {
			d124 = ps.OverlayValues[124]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
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
			d185 = d38
			if d185.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d185)
			d186 = d185
			if d186.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: d186.Type, Imm: scm.NewInt(int64(uint64(d186.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d186.Reg, 32)
				ctx.EmitShrRegImm8(d186.Reg, 32)
			}
			ctx.EmitStoreToStack(d186, int32(bbs[1].PhiBase)+int32(16))
			if d38.Loc == scm.LocReg {
				ctx.UnprotectReg(d38.Reg)
			} else if d38.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d38.Reg)
				ctx.UnprotectReg(d38.Reg2)
			}
		}
		ps187 := scm.PhiState{General: ps.General}
		ps187.OverlayValues = make([]scm.JITValueDesc, 187)
		ps187.OverlayValues[1] = d1
		ps187.OverlayValues[2] = d2
		ps187.OverlayValues[3] = d3
		ps187.OverlayValues[4] = d4
		ps187.OverlayValues[5] = d5
		ps187.OverlayValues[6] = d6
		ps187.OverlayValues[8] = d8
		ps187.OverlayValues[9] = d9
		ps187.OverlayValues[10] = d10
		ps187.OverlayValues[11] = d11
		ps187.OverlayValues[12] = d12
		ps187.OverlayValues[13] = d13
		ps187.OverlayValues[16] = d16
		ps187.OverlayValues[17] = d17
		ps187.OverlayValues[35] = d35
		ps187.OverlayValues[36] = d36
		ps187.OverlayValues[37] = d37
		ps187.OverlayValues[38] = d38
		ps187.OverlayValues[39] = d39
		ps187.OverlayValues[40] = d40
		ps187.OverlayValues[41] = d41
		ps187.OverlayValues[42] = d42
		ps187.OverlayValues[43] = d43
		ps187.OverlayValues[44] = d44
		ps187.OverlayValues[45] = d45
		ps187.OverlayValues[46] = d46
		ps187.OverlayValues[47] = d47
		ps187.OverlayValues[48] = d48
		ps187.OverlayValues[49] = d49
		ps187.OverlayValues[50] = d50
		ps187.OverlayValues[51] = d51
		ps187.OverlayValues[52] = d52
		ps187.OverlayValues[53] = d53
		ps187.OverlayValues[54] = d54
		ps187.OverlayValues[55] = d55
		ps187.OverlayValues[56] = d56
		ps187.OverlayValues[57] = d57
		ps187.OverlayValues[58] = d58
		ps187.OverlayValues[59] = d59
		ps187.OverlayValues[60] = d60
		ps187.OverlayValues[61] = d61
		ps187.OverlayValues[62] = d62
		ps187.OverlayValues[63] = d63
		ps187.OverlayValues[64] = d64
		ps187.OverlayValues[65] = d65
		ps187.OverlayValues[66] = d66
		ps187.OverlayValues[118] = d118
		ps187.OverlayValues[119] = d119
		ps187.OverlayValues[120] = d120
		ps187.OverlayValues[121] = d121
		ps187.OverlayValues[122] = d122
		ps187.OverlayValues[123] = d123
		ps187.OverlayValues[124] = d124
		ps187.OverlayValues[183] = d183
		ps187.OverlayValues[185] = d185
		ps187.OverlayValues[186] = d186
		ps187.PhiValues = make([]scm.JITValueDesc, 2)
		d188 = d38
		ps187.PhiValues[1] = d188
		if ps187.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps187)
		return result
	}
	ps189 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps189)
	ctx.MarkLabel(lbl0)
	d190 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d190)
	ctx.BindReg(r1, &d190)
	ctx.EmitMovPairToResult(&d190, &result)
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
