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
	storageJITFunctions
	i, count uint64      `jit:"immutable-after-finish"`
	recids   StorageInt  `jit:"immutable-after-finish"`
	values   []scm.Scmer `jit:"immutable-after-finish"` // TODO: embed other formats as values (ColumnStorage with a proposeCompression loop)
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

func (s *StorageSparse) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
	var d79 scm.JITValueDesc
	_ = d79
	var d80 scm.JITValueDesc
	_ = d80
	var d81 scm.JITValueDesc
	_ = d81
	var d82 scm.JITValueDesc
	_ = d82
	var d183 scm.JITValueDesc
	_ = d183
	var d184 scm.JITValueDesc
	_ = d184
	var d185 scm.JITValueDesc
	_ = d185
	var d186 scm.JITValueDesc
	_ = d186
	var d187 scm.JITValueDesc
	_ = d187
	var d188 scm.JITValueDesc
	_ = d188
	var d189 scm.JITValueDesc
	_ = d189
	var d304 scm.JITValueDesc
	_ = d304
	var d306 scm.JITValueDesc
	_ = d306
	var d307 scm.JITValueDesc
	_ = d307
	var d309 scm.JITValueDesc
	_ = d309
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
			r2 := ctx.AllocReg()
			ctx.EmitMovRegMem(r2, thisptr.Reg, off)
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			ctx.BindReg(r2, &d3)
		}
		ctx.EnsureDesc(&d3)
		ctx.EnsureDesc(&d3)
		var d4 scm.JITValueDesc
		if d3.Loc == scm.LocImm {
			d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d3.Imm.Int()))))}
		} else {
			r3 := ctx.AllocReg()
			ctx.EmitMovRegReg(r3, d3.Reg)
			ctx.EmitShlRegImm8(r3, 32)
			ctx.EmitShrRegImm8(r3, 32)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d4)
		}
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.FreeDesc(&d3)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
			r4 := ctx.AllocRegExcept(d1.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(d1.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r4, scm.CondEqual)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r4}
			ctx.BindReg(r4, &d12)
		} else if d1.Loc == scm.LocImm {
			r5 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d2.Reg)
			ctx.EmitSetcc(r5, scm.CondEqual)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r5}
			ctx.BindReg(r5, &d12)
		} else {
			r6 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpInt64(d1.Reg, d2.Reg)
			ctx.EmitSetcc(r6, scm.CondEqual)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
			ctx.BindReg(r6, &d12)
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
		snap18 := d1
		snap19 := d2
		snap20 := d3
		snap21 := d4
		snap22 := d5
		snap23 := d6
		snap24 := d8
		snap25 := d9
		snap26 := d10
		snap27 := d11
		snap28 := d12
		snap29 := d13
		snap30 := d16
		snap31 := d17
		alloc32 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl9)
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc32)
		d1 = snap18
		d2 = snap19
		d3 = snap20
		d4 = snap21
		d5 = snap22
		d6 = snap23
		d8 = snap24
		d9 = snap25
		d10 = snap26
		d11 = snap27
		d12 = snap28
		d13 = snap29
		d16 = snap30
		d17 = snap31
		ctx.MarkLabel(lbl10)
		ctx.EmitJmp(lbl4)
		ctx.RestoreAllocState(alloc32)
		d1 = snap18
		d2 = snap19
		d3 = snap20
		d4 = snap21
		d5 = snap22
		d6 = snap23
		d8 = snap24
		d9 = snap25
		d10 = snap26
		d11 = snap27
		d12 = snap28
		d13 = snap29
		d16 = snap30
		d17 = snap31
		ps33 := scm.PhiState{General: true}
		ps33.OverlayValues = make([]scm.JITValueDesc, 18)
		ps33.OverlayValues[1] = d1
		ps33.OverlayValues[2] = d2
		ps33.OverlayValues[3] = d3
		ps33.OverlayValues[4] = d4
		ps33.OverlayValues[5] = d5
		ps33.OverlayValues[6] = d6
		ps33.OverlayValues[8] = d8
		ps33.OverlayValues[9] = d9
		ps33.OverlayValues[10] = d10
		ps33.OverlayValues[11] = d11
		ps33.OverlayValues[12] = d12
		ps33.OverlayValues[13] = d13
		ps33.OverlayValues[16] = d16
		ps33.OverlayValues[17] = d17
		ps34 := scm.PhiState{General: true}
		ps34.OverlayValues = make([]scm.JITValueDesc, 18)
		ps34.OverlayValues[1] = d1
		ps34.OverlayValues[2] = d2
		ps34.OverlayValues[3] = d3
		ps34.OverlayValues[4] = d4
		ps34.OverlayValues[5] = d5
		ps34.OverlayValues[6] = d6
		ps34.OverlayValues[8] = d8
		ps34.OverlayValues[9] = d9
		ps34.OverlayValues[10] = d10
		ps34.OverlayValues[11] = d11
		ps34.OverlayValues[12] = d12
		ps34.OverlayValues[13] = d13
		ps34.OverlayValues[16] = d16
		ps34.OverlayValues[17] = d17
		snap35 := d1
		snap36 := d2
		snap37 := d3
		snap38 := d4
		snap39 := d5
		snap40 := d6
		snap41 := d8
		snap42 := d9
		snap43 := d10
		snap44 := d11
		snap45 := d12
		snap46 := d13
		snap47 := d16
		snap48 := d17
		alloc49 := ctx.SnapshotAllocState()
		if !bbs[3].Rendered {
			bbs[3].RenderPS(ps34)
		}
		ctx.RestoreAllocState(alloc49)
		d1 = snap35
		d2 = snap36
		d3 = snap37
		d4 = snap38
		d5 = snap39
		d6 = snap40
		d8 = snap41
		d9 = snap42
		d10 = snap43
		d11 = snap44
		d12 = snap45
		d13 = snap46
		d16 = snap47
		d17 = snap48
		if !bbs[2].Rendered {
			return bbs[2].RenderPS(ps33)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
		d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d51 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d51)
		ctx.BindReg(r1, &d51)
		ctx.EnsureDesc(&d50)
		if d50.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d50, &d51)
		} else {
			switch d50.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d51, d50)
			case scm.TagInt:
				ctx.EmitMakeInt(d51, d50)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d51, d50)
			case scm.TagNil:
				ctx.EmitMakeNil(d51)
			default:
				ctx.EmitMovPairToResult(&d50, &d51)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
			d50 = ps.OverlayValues[50]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&d1, &d2)
		var d52 scm.JITValueDesc
		if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + d2.Imm.Int())}
		} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
			r7 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(r7, d1.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d52)
		} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d2.Reg}
			ctx.BindReg(d2.Reg, &d52)
		} else if d1.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
			ctx.EmitAddInt64(scratch, d2.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		} else if d2.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d52)
		} else {
			r8 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
			ctx.EmitMovRegReg(r8, d1.Reg)
			ctx.EmitAddInt64(r8, d2.Reg)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d52)
		}
		if d52.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: d52.Type, Imm: scm.NewInt(int64(uint64(d52.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d52.Reg, 32)
			ctx.EmitShrRegImm8(d52.Reg, 32)
		}
		if d52.Loc == scm.LocReg && d1.Loc == scm.LocReg && d52.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d52)
		var d53 scm.JITValueDesc
		if d52.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() / 2)}
		} else {
			r9 := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegReg(r9, d52.Reg)
			ctx.EmitShrRegImm8(r9, 1)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d53)
		}
		if d53.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: d53.Type, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d53.Reg, 32)
			ctx.EmitShrRegImm8(d53.Reg, 32)
		}
		if d53.Loc == scm.LocReg && d52.Loc == scm.LocReg && d53.Reg == d52.Reg {
			ctx.TransferReg(d52.Reg)
			d52.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d53)
		ctx.FreeDesc(&d52)
		ctx.EnsureDesc(&d53)
		d54 = d53
		_ = d54
		ctx.StabilizeDescForControlFlow(&d54)
		ctx.StabilizeDescForControlFlow(&d53)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl11 := ctx.ReserveLabel()
		_ = lbl11
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl11)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d54)
		ctx.EnsureDesc(&d54)
		var d55 scm.JITValueDesc
		if d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d54.Imm.Int()))))}
		} else {
			r10 := ctx.AllocReg()
			ctx.EmitMovRegReg(r10, d54.Reg)
			ctx.EmitShlRegImm8(r10, 32)
			ctx.EmitShrRegImm8(r10, 32)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d55)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d56 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 48)
			r11 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r11, thisptr.Reg, off)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
			ctx.BindReg(r11, &d56)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d56)
		ctx.EnsureDesc(&d56)
		var d57 scm.JITValueDesc
		if d56.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d56.Imm.Int()))))}
		} else {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegReg(r12, d56.Reg)
			ctx.EmitShlRegImm8(r12, 56)
			ctx.EmitShrRegImm8(r12, 56)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d57)
		}
		ctx.FreeDesc(&d56)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d55, &d57)
		var d58 scm.JITValueDesc
		if d55.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d55.Imm.Int() * d57.Imm.Int())}
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d55.Imm.Int()))
			ctx.EmitImulInt64(scratch, d57.Reg)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d58)
		} else if d57.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegReg(scratch, d55.Reg)
			if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d57.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d58)
		} else {
			r13 := ctx.AllocRegExcept(d55.Reg, d57.Reg)
			ctx.EmitMovRegReg(r13, d55.Reg)
			ctx.EmitImulInt64(r13, d57.Reg)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d58)
		}
		if d58.Loc == scm.LocReg && d55.Loc == scm.LocReg && d58.Reg == d55.Reg {
			ctx.TransferReg(d55.Reg)
			d55.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d55)
		ctx.FreeDesc(&d57)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		var d59 scm.JITValueDesc
		if d58.Loc == scm.LocImm {
			d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() / 64)}
		} else {
			r14 := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegReg(r14, d58.Reg)
			ctx.EmitShrRegImm8(r14, 6)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d59)
		}
		if d59.Loc == scm.LocReg && d58.Loc == scm.LocReg && d59.Reg == d58.Reg {
			ctx.TransferReg(d58.Reg)
			d58.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		var d60 scm.JITValueDesc
		if d58.Loc == scm.LocImm {
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() % 64)}
		} else {
			r15 := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegReg(r15, d58.Reg)
			ctx.EmitAndRegImm32(r15, 63)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d60)
		}
		if d60.Loc == scm.LocReg && d58.Loc == scm.LocReg && d60.Reg == d58.Reg {
			ctx.TransferReg(d58.Reg)
			d58.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d58)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d61 scm.JITValueDesc
		r16 := ctx.AllocReg()
		r17 := ctx.AllocRegExcept(r16)
		r18 := ctx.AllocRegExcept(r16, r17)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r16, uint64(dataPtr))
			ctx.EmitMovRegImm64(r17, uint64(sliceLen))
			ctx.EmitMovRegImm64(r18, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off)
			ctx.EmitMovRegMem(r17, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r18, thisptr.Reg, off+16)
		}
		d61 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r16, Reg2: r17, Reg3: r18}
		ctx.BindReg(r16, &d61)
		ctx.BindReg(r17, &d61)
		ctx.BindReg(r18, &d61)
		ctx.BindReg(r16, &d61)
		ctx.BindReg(r17, &d61)
		ctx.BindReg(r18, &d61)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		d63 = ctx.EmitSliceElementAddress(&d61, &d59, 8)
		ctx.EnsureDesc(&d63)
		ctx.EmitMovRegMem(d63.Reg, d63.Reg, 0)
		d62 = d63
		d62.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d62)
		ctx.EnsureDesc(&d60)
		ctx.EnsureDescsTogether(&d62, &d60)
		var d64 scm.JITValueDesc
		if d62.Loc == scm.LocImm && d60.Loc == scm.LocImm {
			d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d62.Imm.Int()) << uint64(d60.Imm.Int())))}
		} else if d60.Loc == scm.LocImm {
			r19 := ctx.AllocRegExcept(d62.Reg)
			ctx.EmitMovRegReg(r19, d62.Reg)
			ctx.EmitShlRegImm8(r19, uint8(d60.Imm.Int()))
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d64)
		} else {
			{
				shiftSrc := d62.Reg
				r20 := ctx.AllocRegExcept(d62.Reg, d60.Reg)
				ctx.EmitMovRegReg(r20, d62.Reg)
				shiftSrc = r20
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d60.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d60.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d60.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d64)
			}
		}
		if d64.Loc == scm.LocReg && d62.Loc == scm.LocReg && d64.Reg == d62.Reg {
			ctx.TransferReg(d62.Reg)
			d62.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d62)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d59)
		ctx.EnsureDesc(&d59)
		var d65 scm.JITValueDesc
		if d59.Loc == scm.LocImm {
			d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegReg(scratch, d59.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d65)
		}
		if d65.Loc == scm.LocReg && d59.Loc == scm.LocReg && d65.Reg == d59.Reg {
			ctx.TransferReg(d59.Reg)
			d59.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d65)
		ctx.ReclaimUntrackedRegs()
		d67 = ctx.EmitSliceElementAddress(&d61, &d65, 8)
		ctx.EnsureDesc(&d67)
		ctx.EmitMovRegMem(d67.Reg, d67.Reg, 0)
		d66 = d67
		d66.Type = scm.TagInt
		ctx.FreeDesc(&d65)
		ctx.ReclaimUntrackedRegs()
		d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d60)
		ctx.EnsureDescsTogether(&d68, &d60)
		var d69 scm.JITValueDesc
		if d68.Loc == scm.LocImm && d60.Loc == scm.LocImm {
			d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d68.Imm.Int() - d60.Imm.Int())}
		} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
			r21 := ctx.AllocRegExcept(d68.Reg)
			ctx.EmitMovRegReg(r21, d68.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d69)
		} else if d68.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
			ctx.EmitSubInt64(scratch, d60.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d69)
		} else if d60.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d68.Reg)
			ctx.EmitMovRegReg(scratch, d68.Reg)
			if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d60.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d69)
		} else {
			r22 := ctx.AllocRegExcept(d68.Reg, d60.Reg)
			ctx.EmitMovRegReg(r22, d68.Reg)
			ctx.EmitSubInt64(r22, d60.Reg)
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d69)
		}
		if d69.Loc == scm.LocReg && d68.Loc == scm.LocReg && d69.Reg == d68.Reg {
			ctx.TransferReg(d68.Reg)
			d68.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d60)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d66)
		ctx.EnsureDesc(&d69)
		ctx.EnsureDescsTogether(&d66, &d69)
		var d70 scm.JITValueDesc
		if d66.Loc == scm.LocImm && d69.Loc == scm.LocImm {
			d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d66.Imm.Int()) >> uint64(d69.Imm.Int())))}
		} else if d69.Loc == scm.LocImm {
			r23 := ctx.AllocRegExcept(d66.Reg)
			ctx.EmitMovRegReg(r23, d66.Reg)
			ctx.EmitShrRegImm8(r23, uint8(d69.Imm.Int()))
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d70)
		} else {
			{
				shiftSrc := d66.Reg
				r24 := ctx.AllocRegExcept(d66.Reg, d69.Reg)
				ctx.EmitMovRegReg(r24, d66.Reg)
				shiftSrc = r24
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d69.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d69.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d69.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d70)
			}
		}
		if d70.Loc == scm.LocReg && d66.Loc == scm.LocReg && d70.Reg == d66.Reg {
			ctx.TransferReg(d66.Reg)
			d66.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d66)
		ctx.FreeDesc(&d69)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d64)
		ctx.EnsureDesc(&d70)
		var d71 scm.JITValueDesc
		if d64.Loc == scm.LocImm && d70.Loc == scm.LocImm {
			d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d64.Imm.Int() | d70.Imm.Int())}
		} else if d64.Loc == scm.LocImm && d64.Imm.Int() == 0 {
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d70.Reg}
			ctx.BindReg(d70.Reg, &d71)
		} else if d70.Loc == scm.LocImm && d70.Imm.Int() == 0 {
			r25 := ctx.AllocRegExcept(d64.Reg)
			ctx.EmitMovRegReg(r25, d64.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d71)
		} else if d64.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d70.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
			ctx.EmitOrInt64(scratch, d70.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d71)
		} else if d70.Loc == scm.LocImm {
			r26 := ctx.AllocRegExcept(d64.Reg)
			ctx.EmitMovRegReg(r26, d64.Reg)
			if d70.Imm.Int() >= -2147483648 && d70.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r26, int32(d70.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d70.Imm.Int()))
				ctx.EmitOrInt64(r26, scm.RegR11)
			}
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d71)
		} else {
			r27 := ctx.AllocRegExcept(d64.Reg, d70.Reg)
			ctx.EmitMovRegReg(r27, d64.Reg)
			ctx.EmitOrInt64(r27, d70.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d71)
		}
		if d71.Loc == scm.LocReg && d64.Loc == scm.LocReg && d71.Reg == d64.Reg {
			ctx.TransferReg(d64.Reg)
			d64.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d64)
		ctx.FreeDesc(&d70)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d72 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 48)
			r28 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r28, thisptr.Reg, off)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.BindReg(r28, &d72)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d72)
		ctx.EnsureDesc(&d72)
		var d73 scm.JITValueDesc
		if d72.Loc == scm.LocImm {
			d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d72.Imm.Int()))))}
		} else {
			r29 := ctx.AllocReg()
			ctx.EmitMovRegReg(r29, d72.Reg)
			ctx.EmitShlRegImm8(r29, 56)
			ctx.EmitShrRegImm8(r29, 56)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d73)
		}
		ctx.FreeDesc(&d72)
		ctx.ReclaimUntrackedRegs()
		d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d73)
		ctx.EnsureDescsTogether(&d74, &d73)
		var d75 scm.JITValueDesc
		if d74.Loc == scm.LocImm && d73.Loc == scm.LocImm {
			d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() - d73.Imm.Int())}
		} else if d73.Loc == scm.LocImm && d73.Imm.Int() == 0 {
			r30 := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitMovRegReg(r30, d74.Reg)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d75)
		} else if d74.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
			ctx.EmitSubInt64(scratch, d73.Reg)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d75)
		} else if d73.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitMovRegReg(scratch, d74.Reg)
			if d73.Imm.Int() >= -2147483648 && d73.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d73.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d75)
		} else {
			r31 := ctx.AllocRegExcept(d74.Reg, d73.Reg)
			ctx.EmitMovRegReg(r31, d74.Reg)
			ctx.EmitSubInt64(r31, d73.Reg)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d75)
		}
		if d75.Loc == scm.LocReg && d74.Loc == scm.LocReg && d75.Reg == d74.Reg {
			ctx.TransferReg(d74.Reg)
			d74.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d73)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d71)
		ctx.EnsureDesc(&d75)
		ctx.EnsureDescsTogether(&d71, &d75)
		var d76 scm.JITValueDesc
		if d71.Loc == scm.LocImm && d75.Loc == scm.LocImm {
			d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d71.Imm.Int()) >> uint64(d75.Imm.Int())))}
		} else if d75.Loc == scm.LocImm {
			r32 := ctx.AllocRegExcept(d71.Reg)
			ctx.EmitMovRegReg(r32, d71.Reg)
			ctx.EmitShrRegImm8(r32, uint8(d75.Imm.Int()))
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d76)
		} else {
			{
				shiftSrc := d71.Reg
				r33 := ctx.AllocRegExcept(d71.Reg, d75.Reg)
				ctx.EmitMovRegReg(r33, d71.Reg)
				shiftSrc = r33
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d75.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d75.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d75.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d76)
			}
		}
		if d76.Loc == scm.LocReg && d71.Loc == scm.LocReg && d76.Reg == d71.Reg {
			ctx.TransferReg(d71.Reg)
			d71.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d71)
		ctx.FreeDesc(&d75)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d76)
		var d77 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 56)
			r34 := ctx.AllocReg()
			ctx.EmitMovRegMem(r34, thisptr.Reg, off)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d77)
		}
		ctx.EnsureDesc(&d77)
		ctx.EnsureDesc(&d77)
		var d78 scm.JITValueDesc
		if d77.Loc == scm.LocImm {
			d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d77.Imm.Int()))))}
		} else {
			r35 := ctx.AllocReg()
			ctx.EmitMovRegReg(r35, d77.Reg)
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d78)
		}
		ctx.FreeDesc(&d77)
		ctx.EnsureDesc(&d76)
		ctx.EnsureDesc(&d78)
		ctx.EnsureDescsTogether(&d76, &d78)
		var d79 scm.JITValueDesc
		if d76.Loc == scm.LocImm && d78.Loc == scm.LocImm {
			d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d76.Imm.Int() + d78.Imm.Int())}
		} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
			r36 := ctx.AllocRegExcept(d76.Reg)
			ctx.EmitMovRegReg(r36, d76.Reg)
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d79)
		} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d78.Reg}
			ctx.BindReg(d78.Reg, &d79)
		} else if d76.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d78.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d76.Imm.Int()))
			ctx.EmitAddInt64(scratch, d78.Reg)
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d79)
		} else if d78.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d76.Reg)
			ctx.EmitMovRegReg(scratch, d76.Reg)
			if d78.Imm.Int() >= -2147483648 && d78.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d78.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d79)
		} else {
			r37 := ctx.AllocRegExcept(d76.Reg, d78.Reg)
			ctx.EmitMovRegReg(r37, d76.Reg)
			ctx.EmitAddInt64(r37, d78.Reg)
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d79)
		}
		if d79.Loc == scm.LocReg && d76.Loc == scm.LocReg && d79.Reg == d76.Reg {
			ctx.TransferReg(d76.Reg)
			d76.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d79)
		ctx.FreeDesc(&d76)
		ctx.FreeDesc(&d78)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d80 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegReg(r38, idxInt.Reg)
			ctx.EmitShlRegImm8(r38, 32)
			ctx.EmitShrRegImm8(r38, 32)
			d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d80)
		}
		ctx.EnsureDesc(&d79)
		ctx.EnsureDesc(&d80)
		ctx.EnsureDescsTogether(&d79, &d80)
		var d81 scm.JITValueDesc
		if d79.Loc == scm.LocImm && d80.Loc == scm.LocImm {
			d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d79.Imm.Int()) == uint64(d80.Imm.Int()))}
		} else if d80.Loc == scm.LocImm {
			r39 := ctx.AllocRegExcept(d79.Reg)
			if d80.Imm.Int() >= -2147483648 && d80.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d79.Reg, int32(d80.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d80.Imm.Int()))
				ctx.EmitCmpInt64(d79.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r39, scm.CondEqual)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r39}
			ctx.BindReg(r39, &d81)
		} else if d79.Loc == scm.LocImm {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d80.Reg)
			ctx.EmitSetcc(r40, scm.CondEqual)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r40}
			ctx.BindReg(r40, &d81)
		} else {
			r41 := ctx.AllocRegExcept(d79.Reg)
			ctx.EmitCmpInt64(d79.Reg, d80.Reg)
			ctx.EmitSetcc(r41, scm.CondEqual)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r41}
			ctx.BindReg(r41, &d81)
		}
		ctx.FreeDesc(&d80)
		d82 = d81
		ctx.EnsureDesc(&d82)
		if d82.Loc != scm.LocImm && d82.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d82.Loc == scm.LocImm {
			if d82.Imm.Bool() {
				if ps.General {
				}
				ps83 := scm.PhiState{General: ps.General}
				ps83.OverlayValues = make([]scm.JITValueDesc, 83)
				ps83.OverlayValues[1] = d1
				ps83.OverlayValues[2] = d2
				ps83.OverlayValues[3] = d3
				ps83.OverlayValues[4] = d4
				ps83.OverlayValues[5] = d5
				ps83.OverlayValues[6] = d6
				ps83.OverlayValues[8] = d8
				ps83.OverlayValues[9] = d9
				ps83.OverlayValues[10] = d10
				ps83.OverlayValues[11] = d11
				ps83.OverlayValues[12] = d12
				ps83.OverlayValues[13] = d13
				ps83.OverlayValues[16] = d16
				ps83.OverlayValues[17] = d17
				ps83.OverlayValues[50] = d50
				ps83.OverlayValues[51] = d51
				ps83.OverlayValues[52] = d52
				ps83.OverlayValues[53] = d53
				ps83.OverlayValues[54] = d54
				ps83.OverlayValues[55] = d55
				ps83.OverlayValues[56] = d56
				ps83.OverlayValues[57] = d57
				ps83.OverlayValues[58] = d58
				ps83.OverlayValues[59] = d59
				ps83.OverlayValues[60] = d60
				ps83.OverlayValues[61] = d61
				ps83.OverlayValues[62] = d62
				ps83.OverlayValues[63] = d63
				ps83.OverlayValues[64] = d64
				ps83.OverlayValues[65] = d65
				ps83.OverlayValues[66] = d66
				ps83.OverlayValues[67] = d67
				ps83.OverlayValues[68] = d68
				ps83.OverlayValues[69] = d69
				ps83.OverlayValues[70] = d70
				ps83.OverlayValues[71] = d71
				ps83.OverlayValues[72] = d72
				ps83.OverlayValues[73] = d73
				ps83.OverlayValues[74] = d74
				ps83.OverlayValues[75] = d75
				ps83.OverlayValues[76] = d76
				ps83.OverlayValues[77] = d77
				ps83.OverlayValues[78] = d78
				ps83.OverlayValues[79] = d79
				ps83.OverlayValues[80] = d80
				ps83.OverlayValues[81] = d81
				ps83.OverlayValues[82] = d82
				return bbs[4].RenderPS(ps83)
			}
			if ps.General {
			}
			ps84 := scm.PhiState{General: ps.General}
			ps84.OverlayValues = make([]scm.JITValueDesc, 83)
			ps84.OverlayValues[1] = d1
			ps84.OverlayValues[2] = d2
			ps84.OverlayValues[3] = d3
			ps84.OverlayValues[4] = d4
			ps84.OverlayValues[5] = d5
			ps84.OverlayValues[6] = d6
			ps84.OverlayValues[8] = d8
			ps84.OverlayValues[9] = d9
			ps84.OverlayValues[10] = d10
			ps84.OverlayValues[11] = d11
			ps84.OverlayValues[12] = d12
			ps84.OverlayValues[13] = d13
			ps84.OverlayValues[16] = d16
			ps84.OverlayValues[17] = d17
			ps84.OverlayValues[50] = d50
			ps84.OverlayValues[51] = d51
			ps84.OverlayValues[52] = d52
			ps84.OverlayValues[53] = d53
			ps84.OverlayValues[54] = d54
			ps84.OverlayValues[55] = d55
			ps84.OverlayValues[56] = d56
			ps84.OverlayValues[57] = d57
			ps84.OverlayValues[58] = d58
			ps84.OverlayValues[59] = d59
			ps84.OverlayValues[60] = d60
			ps84.OverlayValues[61] = d61
			ps84.OverlayValues[62] = d62
			ps84.OverlayValues[63] = d63
			ps84.OverlayValues[64] = d64
			ps84.OverlayValues[65] = d65
			ps84.OverlayValues[66] = d66
			ps84.OverlayValues[67] = d67
			ps84.OverlayValues[68] = d68
			ps84.OverlayValues[69] = d69
			ps84.OverlayValues[70] = d70
			ps84.OverlayValues[71] = d71
			ps84.OverlayValues[72] = d72
			ps84.OverlayValues[73] = d73
			ps84.OverlayValues[74] = d74
			ps84.OverlayValues[75] = d75
			ps84.OverlayValues[76] = d76
			ps84.OverlayValues[77] = d77
			ps84.OverlayValues[78] = d78
			ps84.OverlayValues[79] = d79
			ps84.OverlayValues[80] = d80
			ps84.OverlayValues[81] = d81
			ps84.OverlayValues[82] = d82
			return bbs[5].RenderPS(ps84)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d82.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl12)
		ctx.EmitJmp(lbl13)
		snap85 := d1
		snap86 := d2
		snap87 := d3
		snap88 := d4
		snap89 := d5
		snap90 := d6
		snap91 := d8
		snap92 := d9
		snap93 := d10
		snap94 := d11
		snap95 := d12
		snap96 := d13
		snap97 := d16
		snap98 := d17
		snap99 := d50
		snap100 := d51
		snap101 := d52
		snap102 := d53
		snap103 := d54
		snap104 := d55
		snap105 := d56
		snap106 := d57
		snap107 := d58
		snap108 := d59
		snap109 := d60
		snap110 := d61
		snap111 := d62
		snap112 := d63
		snap113 := d64
		snap114 := d65
		snap115 := d66
		snap116 := d67
		snap117 := d68
		snap118 := d69
		snap119 := d70
		snap120 := d71
		snap121 := d72
		snap122 := d73
		snap123 := d74
		snap124 := d75
		snap125 := d76
		snap126 := d77
		snap127 := d78
		snap128 := d79
		snap129 := d80
		snap130 := d81
		snap131 := d82
		alloc132 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl5)
		ctx.RestoreAllocState(alloc132)
		d1 = snap85
		d2 = snap86
		d3 = snap87
		d4 = snap88
		d5 = snap89
		d6 = snap90
		d8 = snap91
		d9 = snap92
		d10 = snap93
		d11 = snap94
		d12 = snap95
		d13 = snap96
		d16 = snap97
		d17 = snap98
		d50 = snap99
		d51 = snap100
		d52 = snap101
		d53 = snap102
		d54 = snap103
		d55 = snap104
		d56 = snap105
		d57 = snap106
		d58 = snap107
		d59 = snap108
		d60 = snap109
		d61 = snap110
		d62 = snap111
		d63 = snap112
		d64 = snap113
		d65 = snap114
		d66 = snap115
		d67 = snap116
		d68 = snap117
		d69 = snap118
		d70 = snap119
		d71 = snap120
		d72 = snap121
		d73 = snap122
		d74 = snap123
		d75 = snap124
		d76 = snap125
		d77 = snap126
		d78 = snap127
		d79 = snap128
		d80 = snap129
		d81 = snap130
		d82 = snap131
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl6)
		ctx.RestoreAllocState(alloc132)
		d1 = snap85
		d2 = snap86
		d3 = snap87
		d4 = snap88
		d5 = snap89
		d6 = snap90
		d8 = snap91
		d9 = snap92
		d10 = snap93
		d11 = snap94
		d12 = snap95
		d13 = snap96
		d16 = snap97
		d17 = snap98
		d50 = snap99
		d51 = snap100
		d52 = snap101
		d53 = snap102
		d54 = snap103
		d55 = snap104
		d56 = snap105
		d57 = snap106
		d58 = snap107
		d59 = snap108
		d60 = snap109
		d61 = snap110
		d62 = snap111
		d63 = snap112
		d64 = snap113
		d65 = snap114
		d66 = snap115
		d67 = snap116
		d68 = snap117
		d69 = snap118
		d70 = snap119
		d71 = snap120
		d72 = snap121
		d73 = snap122
		d74 = snap123
		d75 = snap124
		d76 = snap125
		d77 = snap126
		d78 = snap127
		d79 = snap128
		d80 = snap129
		d81 = snap130
		d82 = snap131
		ps133 := scm.PhiState{General: true}
		ps133.OverlayValues = make([]scm.JITValueDesc, 83)
		ps133.OverlayValues[1] = d1
		ps133.OverlayValues[2] = d2
		ps133.OverlayValues[3] = d3
		ps133.OverlayValues[4] = d4
		ps133.OverlayValues[5] = d5
		ps133.OverlayValues[6] = d6
		ps133.OverlayValues[8] = d8
		ps133.OverlayValues[9] = d9
		ps133.OverlayValues[10] = d10
		ps133.OverlayValues[11] = d11
		ps133.OverlayValues[12] = d12
		ps133.OverlayValues[13] = d13
		ps133.OverlayValues[16] = d16
		ps133.OverlayValues[17] = d17
		ps133.OverlayValues[50] = d50
		ps133.OverlayValues[51] = d51
		ps133.OverlayValues[52] = d52
		ps133.OverlayValues[53] = d53
		ps133.OverlayValues[54] = d54
		ps133.OverlayValues[55] = d55
		ps133.OverlayValues[56] = d56
		ps133.OverlayValues[57] = d57
		ps133.OverlayValues[58] = d58
		ps133.OverlayValues[59] = d59
		ps133.OverlayValues[60] = d60
		ps133.OverlayValues[61] = d61
		ps133.OverlayValues[62] = d62
		ps133.OverlayValues[63] = d63
		ps133.OverlayValues[64] = d64
		ps133.OverlayValues[65] = d65
		ps133.OverlayValues[66] = d66
		ps133.OverlayValues[67] = d67
		ps133.OverlayValues[68] = d68
		ps133.OverlayValues[69] = d69
		ps133.OverlayValues[70] = d70
		ps133.OverlayValues[71] = d71
		ps133.OverlayValues[72] = d72
		ps133.OverlayValues[73] = d73
		ps133.OverlayValues[74] = d74
		ps133.OverlayValues[75] = d75
		ps133.OverlayValues[76] = d76
		ps133.OverlayValues[77] = d77
		ps133.OverlayValues[78] = d78
		ps133.OverlayValues[79] = d79
		ps133.OverlayValues[80] = d80
		ps133.OverlayValues[81] = d81
		ps133.OverlayValues[82] = d82
		ps134 := scm.PhiState{General: true}
		ps134.OverlayValues = make([]scm.JITValueDesc, 83)
		ps134.OverlayValues[1] = d1
		ps134.OverlayValues[2] = d2
		ps134.OverlayValues[3] = d3
		ps134.OverlayValues[4] = d4
		ps134.OverlayValues[5] = d5
		ps134.OverlayValues[6] = d6
		ps134.OverlayValues[8] = d8
		ps134.OverlayValues[9] = d9
		ps134.OverlayValues[10] = d10
		ps134.OverlayValues[11] = d11
		ps134.OverlayValues[12] = d12
		ps134.OverlayValues[13] = d13
		ps134.OverlayValues[16] = d16
		ps134.OverlayValues[17] = d17
		ps134.OverlayValues[50] = d50
		ps134.OverlayValues[51] = d51
		ps134.OverlayValues[52] = d52
		ps134.OverlayValues[53] = d53
		ps134.OverlayValues[54] = d54
		ps134.OverlayValues[55] = d55
		ps134.OverlayValues[56] = d56
		ps134.OverlayValues[57] = d57
		ps134.OverlayValues[58] = d58
		ps134.OverlayValues[59] = d59
		ps134.OverlayValues[60] = d60
		ps134.OverlayValues[61] = d61
		ps134.OverlayValues[62] = d62
		ps134.OverlayValues[63] = d63
		ps134.OverlayValues[64] = d64
		ps134.OverlayValues[65] = d65
		ps134.OverlayValues[66] = d66
		ps134.OverlayValues[67] = d67
		ps134.OverlayValues[68] = d68
		ps134.OverlayValues[69] = d69
		ps134.OverlayValues[70] = d70
		ps134.OverlayValues[71] = d71
		ps134.OverlayValues[72] = d72
		ps134.OverlayValues[73] = d73
		ps134.OverlayValues[74] = d74
		ps134.OverlayValues[75] = d75
		ps134.OverlayValues[76] = d76
		ps134.OverlayValues[77] = d77
		ps134.OverlayValues[78] = d78
		ps134.OverlayValues[79] = d79
		ps134.OverlayValues[80] = d80
		ps134.OverlayValues[81] = d81
		ps134.OverlayValues[82] = d82
		snap135 := d1
		snap136 := d2
		snap137 := d3
		snap138 := d4
		snap139 := d5
		snap140 := d6
		snap141 := d8
		snap142 := d9
		snap143 := d10
		snap144 := d11
		snap145 := d12
		snap146 := d13
		snap147 := d16
		snap148 := d17
		snap149 := d50
		snap150 := d51
		snap151 := d52
		snap152 := d53
		snap153 := d54
		snap154 := d55
		snap155 := d56
		snap156 := d57
		snap157 := d58
		snap158 := d59
		snap159 := d60
		snap160 := d61
		snap161 := d62
		snap162 := d63
		snap163 := d64
		snap164 := d65
		snap165 := d66
		snap166 := d67
		snap167 := d68
		snap168 := d69
		snap169 := d70
		snap170 := d71
		snap171 := d72
		snap172 := d73
		snap173 := d74
		snap174 := d75
		snap175 := d76
		snap176 := d77
		snap177 := d78
		snap178 := d79
		snap179 := d80
		snap180 := d81
		snap181 := d82
		alloc182 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps134)
		}
		ctx.RestoreAllocState(alloc182)
		d1 = snap135
		d2 = snap136
		d3 = snap137
		d4 = snap138
		d5 = snap139
		d6 = snap140
		d8 = snap141
		d9 = snap142
		d10 = snap143
		d11 = snap144
		d12 = snap145
		d13 = snap146
		d16 = snap147
		d17 = snap148
		d50 = snap149
		d51 = snap150
		d52 = snap151
		d53 = snap152
		d54 = snap153
		d55 = snap154
		d56 = snap155
		d57 = snap156
		d58 = snap157
		d59 = snap158
		d60 = snap159
		d61 = snap160
		d62 = snap161
		d63 = snap162
		d64 = snap163
		d65 = snap164
		d66 = snap165
		d67 = snap166
		d68 = snap167
		d69 = snap168
		d70 = snap169
		d71 = snap170
		d72 = snap171
		d73 = snap172
		d74 = snap173
		d75 = snap174
		d76 = snap175
		d77 = snap176
		d78 = snap177
		d79 = snap178
		d80 = snap179
		d81 = snap180
		d82 = snap181
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps133)
		}
		return result
		ctx.FreeDesc(&d81)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		ctx.ReclaimUntrackedRegs()
		var d183 scm.JITValueDesc
		r42 := ctx.AllocReg()
		r43 := ctx.AllocRegExcept(r42)
		r44 := ctx.AllocRegExcept(r42, r43)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r42, uint64(dataPtr))
			ctx.EmitMovRegImm64(r43, uint64(sliceLen))
			ctx.EmitMovRegImm64(r44, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
			ctx.EmitMovRegMem(r42, thisptr.Reg, off)
			ctx.EmitMovRegMem(r43, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r44, thisptr.Reg, off+16)
		}
		d183 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r42, Reg2: r43, Reg3: r44}
		ctx.BindReg(r42, &d183)
		ctx.BindReg(r43, &d183)
		ctx.BindReg(r44, &d183)
		ctx.BindReg(r42, &d183)
		ctx.BindReg(r43, &d183)
		ctx.BindReg(r44, &d183)
		ctx.EnsureDesc(&d53)
		d185 = ctx.EmitSliceElementAddress(&d183, &d53, 16)
		ctx.EnsureDesc(&d185)
		r45 := ctx.AllocRegExcept(d185.Reg)
		ctx.EmitMovRegMem(r45, d185.Reg, 8)
		ctx.EmitMovRegMem(d185.Reg, d185.Reg, 0)
		d184 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d185.Reg, Reg2: r45}
		ctx.BindReg(d185.Reg, &d184)
		ctx.BindReg(r45, &d184)
		d186 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d186)
		ctx.BindReg(r1, &d186)
		ctx.EnsureDesc(&d184)
		if d184.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d184, &d186)
		} else {
			switch d184.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d186, d184)
			case scm.TagInt:
				ctx.EmitMakeInt(d186, d184)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d186, d184)
			case scm.TagNil:
				ctx.EmitMakeNil(d186)
			default:
				ctx.EmitMovPairToResult(&d184, &d186)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d187 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
		} else {
			r46 := ctx.AllocReg()
			ctx.EmitMovRegReg(r46, idxInt.Reg)
			ctx.EmitShlRegImm8(r46, 32)
			ctx.EmitShrRegImm8(r46, 32)
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d187)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d79)
		ctx.EnsureDesc(&d187)
		ctx.EnsureDescsTogether(&d79, &d187)
		var d188 scm.JITValueDesc
		if d79.Loc == scm.LocImm && d187.Loc == scm.LocImm {
			d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d79.Imm.Int()) < uint64(d187.Imm.Int()))}
		} else if d187.Loc == scm.LocImm {
			r47 := ctx.AllocRegExcept(d79.Reg)
			if d187.Imm.Int() >= -2147483648 && d187.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d79.Reg, int32(d187.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
				ctx.EmitCmpInt64(d79.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r47, scm.CondUnsignedBelow)
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r47}
			ctx.BindReg(r47, &d188)
		} else if d79.Loc == scm.LocImm {
			r48 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d187.Reg)
			ctx.EmitSetcc(r48, scm.CondUnsignedBelow)
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
			ctx.BindReg(r48, &d188)
		} else {
			r49 := ctx.AllocRegExcept(d79.Reg)
			ctx.EmitCmpInt64(d79.Reg, d187.Reg)
			ctx.EmitSetcc(r49, scm.CondUnsignedBelow)
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
			ctx.BindReg(r49, &d188)
		}
		ctx.FreeDesc(&d187)
		d189 = d188
		ctx.EnsureDesc(&d189)
		if d189.Loc != scm.LocImm && d189.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d189.Loc == scm.LocImm {
			if d189.Imm.Bool() {
				if ps.General {
				}
				ps190 := scm.PhiState{General: ps.General}
				ps190.OverlayValues = make([]scm.JITValueDesc, 190)
				ps190.OverlayValues[1] = d1
				ps190.OverlayValues[2] = d2
				ps190.OverlayValues[3] = d3
				ps190.OverlayValues[4] = d4
				ps190.OverlayValues[5] = d5
				ps190.OverlayValues[6] = d6
				ps190.OverlayValues[8] = d8
				ps190.OverlayValues[9] = d9
				ps190.OverlayValues[10] = d10
				ps190.OverlayValues[11] = d11
				ps190.OverlayValues[12] = d12
				ps190.OverlayValues[13] = d13
				ps190.OverlayValues[16] = d16
				ps190.OverlayValues[17] = d17
				ps190.OverlayValues[50] = d50
				ps190.OverlayValues[51] = d51
				ps190.OverlayValues[52] = d52
				ps190.OverlayValues[53] = d53
				ps190.OverlayValues[54] = d54
				ps190.OverlayValues[55] = d55
				ps190.OverlayValues[56] = d56
				ps190.OverlayValues[57] = d57
				ps190.OverlayValues[58] = d58
				ps190.OverlayValues[59] = d59
				ps190.OverlayValues[60] = d60
				ps190.OverlayValues[61] = d61
				ps190.OverlayValues[62] = d62
				ps190.OverlayValues[63] = d63
				ps190.OverlayValues[64] = d64
				ps190.OverlayValues[65] = d65
				ps190.OverlayValues[66] = d66
				ps190.OverlayValues[67] = d67
				ps190.OverlayValues[68] = d68
				ps190.OverlayValues[69] = d69
				ps190.OverlayValues[70] = d70
				ps190.OverlayValues[71] = d71
				ps190.OverlayValues[72] = d72
				ps190.OverlayValues[73] = d73
				ps190.OverlayValues[74] = d74
				ps190.OverlayValues[75] = d75
				ps190.OverlayValues[76] = d76
				ps190.OverlayValues[77] = d77
				ps190.OverlayValues[78] = d78
				ps190.OverlayValues[79] = d79
				ps190.OverlayValues[80] = d80
				ps190.OverlayValues[81] = d81
				ps190.OverlayValues[82] = d82
				ps190.OverlayValues[183] = d183
				ps190.OverlayValues[184] = d184
				ps190.OverlayValues[185] = d185
				ps190.OverlayValues[186] = d186
				ps190.OverlayValues[187] = d187
				ps190.OverlayValues[188] = d188
				ps190.OverlayValues[189] = d189
				return bbs[6].RenderPS(ps190)
			}
			if ps.General {
			}
			ps191 := scm.PhiState{General: ps.General}
			ps191.OverlayValues = make([]scm.JITValueDesc, 190)
			ps191.OverlayValues[1] = d1
			ps191.OverlayValues[2] = d2
			ps191.OverlayValues[3] = d3
			ps191.OverlayValues[4] = d4
			ps191.OverlayValues[5] = d5
			ps191.OverlayValues[6] = d6
			ps191.OverlayValues[8] = d8
			ps191.OverlayValues[9] = d9
			ps191.OverlayValues[10] = d10
			ps191.OverlayValues[11] = d11
			ps191.OverlayValues[12] = d12
			ps191.OverlayValues[13] = d13
			ps191.OverlayValues[16] = d16
			ps191.OverlayValues[17] = d17
			ps191.OverlayValues[50] = d50
			ps191.OverlayValues[51] = d51
			ps191.OverlayValues[52] = d52
			ps191.OverlayValues[53] = d53
			ps191.OverlayValues[54] = d54
			ps191.OverlayValues[55] = d55
			ps191.OverlayValues[56] = d56
			ps191.OverlayValues[57] = d57
			ps191.OverlayValues[58] = d58
			ps191.OverlayValues[59] = d59
			ps191.OverlayValues[60] = d60
			ps191.OverlayValues[61] = d61
			ps191.OverlayValues[62] = d62
			ps191.OverlayValues[63] = d63
			ps191.OverlayValues[64] = d64
			ps191.OverlayValues[65] = d65
			ps191.OverlayValues[66] = d66
			ps191.OverlayValues[67] = d67
			ps191.OverlayValues[68] = d68
			ps191.OverlayValues[69] = d69
			ps191.OverlayValues[70] = d70
			ps191.OverlayValues[71] = d71
			ps191.OverlayValues[72] = d72
			ps191.OverlayValues[73] = d73
			ps191.OverlayValues[74] = d74
			ps191.OverlayValues[75] = d75
			ps191.OverlayValues[76] = d76
			ps191.OverlayValues[77] = d77
			ps191.OverlayValues[78] = d78
			ps191.OverlayValues[79] = d79
			ps191.OverlayValues[80] = d80
			ps191.OverlayValues[81] = d81
			ps191.OverlayValues[82] = d82
			ps191.OverlayValues[183] = d183
			ps191.OverlayValues[184] = d184
			ps191.OverlayValues[185] = d185
			ps191.OverlayValues[186] = d186
			ps191.OverlayValues[187] = d187
			ps191.OverlayValues[188] = d188
			ps191.OverlayValues[189] = d189
			return bbs[7].RenderPS(ps191)
		}
		if !ps.General {
			ps.General = true
			return bbs[5].RenderPS(ps)
		}
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d189.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		ctx.EmitJmp(lbl15)
		snap192 := d1
		snap193 := d2
		snap194 := d3
		snap195 := d4
		snap196 := d5
		snap197 := d6
		snap198 := d8
		snap199 := d9
		snap200 := d10
		snap201 := d11
		snap202 := d12
		snap203 := d13
		snap204 := d16
		snap205 := d17
		snap206 := d50
		snap207 := d51
		snap208 := d52
		snap209 := d53
		snap210 := d54
		snap211 := d55
		snap212 := d56
		snap213 := d57
		snap214 := d58
		snap215 := d59
		snap216 := d60
		snap217 := d61
		snap218 := d62
		snap219 := d63
		snap220 := d64
		snap221 := d65
		snap222 := d66
		snap223 := d67
		snap224 := d68
		snap225 := d69
		snap226 := d70
		snap227 := d71
		snap228 := d72
		snap229 := d73
		snap230 := d74
		snap231 := d75
		snap232 := d76
		snap233 := d77
		snap234 := d78
		snap235 := d79
		snap236 := d80
		snap237 := d81
		snap238 := d82
		snap239 := d183
		snap240 := d184
		snap241 := d185
		snap242 := d186
		snap243 := d187
		snap244 := d188
		snap245 := d189
		alloc246 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc246)
		d1 = snap192
		d2 = snap193
		d3 = snap194
		d4 = snap195
		d5 = snap196
		d6 = snap197
		d8 = snap198
		d9 = snap199
		d10 = snap200
		d11 = snap201
		d12 = snap202
		d13 = snap203
		d16 = snap204
		d17 = snap205
		d50 = snap206
		d51 = snap207
		d52 = snap208
		d53 = snap209
		d54 = snap210
		d55 = snap211
		d56 = snap212
		d57 = snap213
		d58 = snap214
		d59 = snap215
		d60 = snap216
		d61 = snap217
		d62 = snap218
		d63 = snap219
		d64 = snap220
		d65 = snap221
		d66 = snap222
		d67 = snap223
		d68 = snap224
		d69 = snap225
		d70 = snap226
		d71 = snap227
		d72 = snap228
		d73 = snap229
		d74 = snap230
		d75 = snap231
		d76 = snap232
		d77 = snap233
		d78 = snap234
		d79 = snap235
		d80 = snap236
		d81 = snap237
		d82 = snap238
		d183 = snap239
		d184 = snap240
		d185 = snap241
		d186 = snap242
		d187 = snap243
		d188 = snap244
		d189 = snap245
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl8)
		ctx.RestoreAllocState(alloc246)
		d1 = snap192
		d2 = snap193
		d3 = snap194
		d4 = snap195
		d5 = snap196
		d6 = snap197
		d8 = snap198
		d9 = snap199
		d10 = snap200
		d11 = snap201
		d12 = snap202
		d13 = snap203
		d16 = snap204
		d17 = snap205
		d50 = snap206
		d51 = snap207
		d52 = snap208
		d53 = snap209
		d54 = snap210
		d55 = snap211
		d56 = snap212
		d57 = snap213
		d58 = snap214
		d59 = snap215
		d60 = snap216
		d61 = snap217
		d62 = snap218
		d63 = snap219
		d64 = snap220
		d65 = snap221
		d66 = snap222
		d67 = snap223
		d68 = snap224
		d69 = snap225
		d70 = snap226
		d71 = snap227
		d72 = snap228
		d73 = snap229
		d74 = snap230
		d75 = snap231
		d76 = snap232
		d77 = snap233
		d78 = snap234
		d79 = snap235
		d80 = snap236
		d81 = snap237
		d82 = snap238
		d183 = snap239
		d184 = snap240
		d185 = snap241
		d186 = snap242
		d187 = snap243
		d188 = snap244
		d189 = snap245
		ps247 := scm.PhiState{General: true}
		ps247.OverlayValues = make([]scm.JITValueDesc, 190)
		ps247.OverlayValues[1] = d1
		ps247.OverlayValues[2] = d2
		ps247.OverlayValues[3] = d3
		ps247.OverlayValues[4] = d4
		ps247.OverlayValues[5] = d5
		ps247.OverlayValues[6] = d6
		ps247.OverlayValues[8] = d8
		ps247.OverlayValues[9] = d9
		ps247.OverlayValues[10] = d10
		ps247.OverlayValues[11] = d11
		ps247.OverlayValues[12] = d12
		ps247.OverlayValues[13] = d13
		ps247.OverlayValues[16] = d16
		ps247.OverlayValues[17] = d17
		ps247.OverlayValues[50] = d50
		ps247.OverlayValues[51] = d51
		ps247.OverlayValues[52] = d52
		ps247.OverlayValues[53] = d53
		ps247.OverlayValues[54] = d54
		ps247.OverlayValues[55] = d55
		ps247.OverlayValues[56] = d56
		ps247.OverlayValues[57] = d57
		ps247.OverlayValues[58] = d58
		ps247.OverlayValues[59] = d59
		ps247.OverlayValues[60] = d60
		ps247.OverlayValues[61] = d61
		ps247.OverlayValues[62] = d62
		ps247.OverlayValues[63] = d63
		ps247.OverlayValues[64] = d64
		ps247.OverlayValues[65] = d65
		ps247.OverlayValues[66] = d66
		ps247.OverlayValues[67] = d67
		ps247.OverlayValues[68] = d68
		ps247.OverlayValues[69] = d69
		ps247.OverlayValues[70] = d70
		ps247.OverlayValues[71] = d71
		ps247.OverlayValues[72] = d72
		ps247.OverlayValues[73] = d73
		ps247.OverlayValues[74] = d74
		ps247.OverlayValues[75] = d75
		ps247.OverlayValues[76] = d76
		ps247.OverlayValues[77] = d77
		ps247.OverlayValues[78] = d78
		ps247.OverlayValues[79] = d79
		ps247.OverlayValues[80] = d80
		ps247.OverlayValues[81] = d81
		ps247.OverlayValues[82] = d82
		ps247.OverlayValues[183] = d183
		ps247.OverlayValues[184] = d184
		ps247.OverlayValues[185] = d185
		ps247.OverlayValues[186] = d186
		ps247.OverlayValues[187] = d187
		ps247.OverlayValues[188] = d188
		ps247.OverlayValues[189] = d189
		ps248 := scm.PhiState{General: true}
		ps248.OverlayValues = make([]scm.JITValueDesc, 190)
		ps248.OverlayValues[1] = d1
		ps248.OverlayValues[2] = d2
		ps248.OverlayValues[3] = d3
		ps248.OverlayValues[4] = d4
		ps248.OverlayValues[5] = d5
		ps248.OverlayValues[6] = d6
		ps248.OverlayValues[8] = d8
		ps248.OverlayValues[9] = d9
		ps248.OverlayValues[10] = d10
		ps248.OverlayValues[11] = d11
		ps248.OverlayValues[12] = d12
		ps248.OverlayValues[13] = d13
		ps248.OverlayValues[16] = d16
		ps248.OverlayValues[17] = d17
		ps248.OverlayValues[50] = d50
		ps248.OverlayValues[51] = d51
		ps248.OverlayValues[52] = d52
		ps248.OverlayValues[53] = d53
		ps248.OverlayValues[54] = d54
		ps248.OverlayValues[55] = d55
		ps248.OverlayValues[56] = d56
		ps248.OverlayValues[57] = d57
		ps248.OverlayValues[58] = d58
		ps248.OverlayValues[59] = d59
		ps248.OverlayValues[60] = d60
		ps248.OverlayValues[61] = d61
		ps248.OverlayValues[62] = d62
		ps248.OverlayValues[63] = d63
		ps248.OverlayValues[64] = d64
		ps248.OverlayValues[65] = d65
		ps248.OverlayValues[66] = d66
		ps248.OverlayValues[67] = d67
		ps248.OverlayValues[68] = d68
		ps248.OverlayValues[69] = d69
		ps248.OverlayValues[70] = d70
		ps248.OverlayValues[71] = d71
		ps248.OverlayValues[72] = d72
		ps248.OverlayValues[73] = d73
		ps248.OverlayValues[74] = d74
		ps248.OverlayValues[75] = d75
		ps248.OverlayValues[76] = d76
		ps248.OverlayValues[77] = d77
		ps248.OverlayValues[78] = d78
		ps248.OverlayValues[79] = d79
		ps248.OverlayValues[80] = d80
		ps248.OverlayValues[81] = d81
		ps248.OverlayValues[82] = d82
		ps248.OverlayValues[183] = d183
		ps248.OverlayValues[184] = d184
		ps248.OverlayValues[185] = d185
		ps248.OverlayValues[186] = d186
		ps248.OverlayValues[187] = d187
		ps248.OverlayValues[188] = d188
		ps248.OverlayValues[189] = d189
		snap249 := d1
		snap250 := d2
		snap251 := d3
		snap252 := d4
		snap253 := d5
		snap254 := d6
		snap255 := d8
		snap256 := d9
		snap257 := d10
		snap258 := d11
		snap259 := d12
		snap260 := d13
		snap261 := d16
		snap262 := d17
		snap263 := d50
		snap264 := d51
		snap265 := d52
		snap266 := d53
		snap267 := d54
		snap268 := d55
		snap269 := d56
		snap270 := d57
		snap271 := d58
		snap272 := d59
		snap273 := d60
		snap274 := d61
		snap275 := d62
		snap276 := d63
		snap277 := d64
		snap278 := d65
		snap279 := d66
		snap280 := d67
		snap281 := d68
		snap282 := d69
		snap283 := d70
		snap284 := d71
		snap285 := d72
		snap286 := d73
		snap287 := d74
		snap288 := d75
		snap289 := d76
		snap290 := d77
		snap291 := d78
		snap292 := d79
		snap293 := d80
		snap294 := d81
		snap295 := d82
		snap296 := d183
		snap297 := d184
		snap298 := d185
		snap299 := d186
		snap300 := d187
		snap301 := d188
		snap302 := d189
		alloc303 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps248)
		}
		ctx.RestoreAllocState(alloc303)
		d1 = snap249
		d2 = snap250
		d3 = snap251
		d4 = snap252
		d5 = snap253
		d6 = snap254
		d8 = snap255
		d9 = snap256
		d10 = snap257
		d11 = snap258
		d12 = snap259
		d13 = snap260
		d16 = snap261
		d17 = snap262
		d50 = snap263
		d51 = snap264
		d52 = snap265
		d53 = snap266
		d54 = snap267
		d55 = snap268
		d56 = snap269
		d57 = snap270
		d58 = snap271
		d59 = snap272
		d60 = snap273
		d61 = snap274
		d62 = snap275
		d63 = snap276
		d64 = snap277
		d65 = snap278
		d66 = snap279
		d67 = snap280
		d68 = snap281
		d69 = snap282
		d70 = snap283
		d71 = snap284
		d72 = snap285
		d73 = snap286
		d74 = snap287
		d75 = snap288
		d76 = snap289
		d77 = snap290
		d78 = snap291
		d79 = snap292
		d80 = snap293
		d81 = snap294
		d82 = snap295
		d183 = snap296
		d184 = snap297
		d185 = snap298
		d186 = snap299
		d187 = snap300
		d188 = snap301
		d189 = snap302
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps247)
		}
		return result
		ctx.FreeDesc(&d188)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d53)
		var d304 scm.JITValueDesc
		if d53.Loc == scm.LocImm {
			d304 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegReg(scratch, d53.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d304)
		}
		if d304.Loc == scm.LocImm {
			d304 = scm.JITValueDesc{Loc: scm.LocImm, Type: d304.Type, Imm: scm.NewInt(int64(uint64(d304.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d304.Reg, 32)
			ctx.EmitShrRegImm8(d304.Reg, 32)
		}
		if d304.Loc == scm.LocReg && d53.Loc == scm.LocReg && d304.Reg == d53.Reg {
			ctx.TransferReg(d53.Reg)
			d53.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d304)
		ctx.EmitStoreToStack(d304, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d304)
		if ps.General {
		}
		ps305 := scm.PhiState{General: ps.General}
		ps305.OverlayValues = make([]scm.JITValueDesc, 305)
		ps305.OverlayValues[1] = d1
		ps305.OverlayValues[2] = d2
		ps305.OverlayValues[3] = d3
		ps305.OverlayValues[4] = d4
		ps305.OverlayValues[5] = d5
		ps305.OverlayValues[6] = d6
		ps305.OverlayValues[8] = d8
		ps305.OverlayValues[9] = d9
		ps305.OverlayValues[10] = d10
		ps305.OverlayValues[11] = d11
		ps305.OverlayValues[12] = d12
		ps305.OverlayValues[13] = d13
		ps305.OverlayValues[16] = d16
		ps305.OverlayValues[17] = d17
		ps305.OverlayValues[50] = d50
		ps305.OverlayValues[51] = d51
		ps305.OverlayValues[52] = d52
		ps305.OverlayValues[53] = d53
		ps305.OverlayValues[54] = d54
		ps305.OverlayValues[55] = d55
		ps305.OverlayValues[56] = d56
		ps305.OverlayValues[57] = d57
		ps305.OverlayValues[58] = d58
		ps305.OverlayValues[59] = d59
		ps305.OverlayValues[60] = d60
		ps305.OverlayValues[61] = d61
		ps305.OverlayValues[62] = d62
		ps305.OverlayValues[63] = d63
		ps305.OverlayValues[64] = d64
		ps305.OverlayValues[65] = d65
		ps305.OverlayValues[66] = d66
		ps305.OverlayValues[67] = d67
		ps305.OverlayValues[68] = d68
		ps305.OverlayValues[69] = d69
		ps305.OverlayValues[70] = d70
		ps305.OverlayValues[71] = d71
		ps305.OverlayValues[72] = d72
		ps305.OverlayValues[73] = d73
		ps305.OverlayValues[74] = d74
		ps305.OverlayValues[75] = d75
		ps305.OverlayValues[76] = d76
		ps305.OverlayValues[77] = d77
		ps305.OverlayValues[78] = d78
		ps305.OverlayValues[79] = d79
		ps305.OverlayValues[80] = d80
		ps305.OverlayValues[81] = d81
		ps305.OverlayValues[82] = d82
		ps305.OverlayValues[183] = d183
		ps305.OverlayValues[184] = d184
		ps305.OverlayValues[185] = d185
		ps305.OverlayValues[186] = d186
		ps305.OverlayValues[187] = d187
		ps305.OverlayValues[188] = d188
		ps305.OverlayValues[189] = d189
		ps305.OverlayValues[304] = d304
		ps305.PhiValues = make([]scm.JITValueDesc, 2)
		if ps305.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps305)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
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
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
			d80 = ps.OverlayValues[80]
		}
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 183 && ps.OverlayValues[183].Loc != scm.LocNone {
			d183 = ps.OverlayValues[183]
		}
		if len(ps.OverlayValues) > 184 && ps.OverlayValues[184].Loc != scm.LocNone {
			d184 = ps.OverlayValues[184]
		}
		if len(ps.OverlayValues) > 185 && ps.OverlayValues[185].Loc != scm.LocNone {
			d185 = ps.OverlayValues[185]
		}
		if len(ps.OverlayValues) > 186 && ps.OverlayValues[186].Loc != scm.LocNone {
			d186 = ps.OverlayValues[186]
		}
		if len(ps.OverlayValues) > 187 && ps.OverlayValues[187].Loc != scm.LocNone {
			d187 = ps.OverlayValues[187]
		}
		if len(ps.OverlayValues) > 188 && ps.OverlayValues[188].Loc != scm.LocNone {
			d188 = ps.OverlayValues[188]
		}
		if len(ps.OverlayValues) > 189 && ps.OverlayValues[189].Loc != scm.LocNone {
			d189 = ps.OverlayValues[189]
		}
		if len(ps.OverlayValues) > 304 && ps.OverlayValues[304].Loc != scm.LocNone {
			d304 = ps.OverlayValues[304]
		}
		ctx.ReclaimUntrackedRegs()
		if ps.General {
			ctx.SyncDesc(&d53)
			if d53.Loc == scm.LocReg {
				ctx.ProtectReg(d53.Reg)
			} else if d53.Loc == scm.LocRegPair {
				ctx.ProtectReg(d53.Reg)
				ctx.ProtectReg(d53.Reg2)
			}
			d306 = d53
			if d306.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d306)
			d307 = d306
			if d307.Loc == scm.LocImm {
				d307 = scm.JITValueDesc{Loc: scm.LocImm, Type: d307.Type, Imm: scm.NewInt(int64(uint64(d307.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d307.Reg, 32)
				ctx.EmitShrRegImm8(d307.Reg, 32)
			}
			ctx.EmitStoreToStack(d307, int32(bbs[1].PhiBase)+int32(16))
			if d53.Loc == scm.LocReg {
				ctx.UnprotectReg(d53.Reg)
			} else if d53.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d53.Reg)
				ctx.UnprotectReg(d53.Reg2)
			}
		}
		ps308 := scm.PhiState{General: ps.General}
		ps308.OverlayValues = make([]scm.JITValueDesc, 308)
		ps308.OverlayValues[1] = d1
		ps308.OverlayValues[2] = d2
		ps308.OverlayValues[3] = d3
		ps308.OverlayValues[4] = d4
		ps308.OverlayValues[5] = d5
		ps308.OverlayValues[6] = d6
		ps308.OverlayValues[8] = d8
		ps308.OverlayValues[9] = d9
		ps308.OverlayValues[10] = d10
		ps308.OverlayValues[11] = d11
		ps308.OverlayValues[12] = d12
		ps308.OverlayValues[13] = d13
		ps308.OverlayValues[16] = d16
		ps308.OverlayValues[17] = d17
		ps308.OverlayValues[50] = d50
		ps308.OverlayValues[51] = d51
		ps308.OverlayValues[52] = d52
		ps308.OverlayValues[53] = d53
		ps308.OverlayValues[54] = d54
		ps308.OverlayValues[55] = d55
		ps308.OverlayValues[56] = d56
		ps308.OverlayValues[57] = d57
		ps308.OverlayValues[58] = d58
		ps308.OverlayValues[59] = d59
		ps308.OverlayValues[60] = d60
		ps308.OverlayValues[61] = d61
		ps308.OverlayValues[62] = d62
		ps308.OverlayValues[63] = d63
		ps308.OverlayValues[64] = d64
		ps308.OverlayValues[65] = d65
		ps308.OverlayValues[66] = d66
		ps308.OverlayValues[67] = d67
		ps308.OverlayValues[68] = d68
		ps308.OverlayValues[69] = d69
		ps308.OverlayValues[70] = d70
		ps308.OverlayValues[71] = d71
		ps308.OverlayValues[72] = d72
		ps308.OverlayValues[73] = d73
		ps308.OverlayValues[74] = d74
		ps308.OverlayValues[75] = d75
		ps308.OverlayValues[76] = d76
		ps308.OverlayValues[77] = d77
		ps308.OverlayValues[78] = d78
		ps308.OverlayValues[79] = d79
		ps308.OverlayValues[80] = d80
		ps308.OverlayValues[81] = d81
		ps308.OverlayValues[82] = d82
		ps308.OverlayValues[183] = d183
		ps308.OverlayValues[184] = d184
		ps308.OverlayValues[185] = d185
		ps308.OverlayValues[186] = d186
		ps308.OverlayValues[187] = d187
		ps308.OverlayValues[188] = d188
		ps308.OverlayValues[189] = d189
		ps308.OverlayValues[304] = d304
		ps308.OverlayValues[306] = d306
		ps308.OverlayValues[307] = d307
		ps308.PhiValues = make([]scm.JITValueDesc, 2)
		d309 = d53
		ps308.PhiValues[1] = d309
		if ps308.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps308)
		return result
	}
	ps310 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps310)
	ctx.MarkLabel(lbl0)
	d311 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d311)
	ctx.BindReg(r1, &d311)
	ctx.EmitMovPairToResult(&d311, &result)
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

func (s *StorageSparse) GetCachedReader() ColumnReader { return s.storageJITFunctions.reader(s) }

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
	s.storageJITFunctions.finish(s)
}

// soley to StorageSparse
func (s *StorageSparse) proposeCompression(i uint32) ColumnStorage {
	return nil
}

func (s *StorageSparse) DistinctCount() uint {
	return uint(len(s.values)) + 1 // +1 for nil values
}
