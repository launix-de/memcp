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
	var d166 scm.JITValueDesc
	_ = d166
	var d167 scm.JITValueDesc
	_ = d167
	var d168 scm.JITValueDesc
	_ = d168
	var d169 scm.JITValueDesc
	_ = d169
	var d170 scm.JITValueDesc
	_ = d170
	var d171 scm.JITValueDesc
	_ = d171
	var d172 scm.JITValueDesc
	_ = d172
	var d277 scm.JITValueDesc
	_ = d277
	var d279 scm.JITValueDesc
	_ = d279
	var d280 scm.JITValueDesc
	_ = d280
	var d282 scm.JITValueDesc
	_ = d282
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
	phiBase0 := ctx.AllocStack(int32(32))
	var bbs [8]scm.BBDescriptor
	bbs[1].PhiBase = int32(phiBase0) + int32(0)
	bbs[1].PhiCount = uint16(2)
	d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
	_ = d1
	d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
	_ = d2
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
			ctx.EmitStoreToStack(d5, int32(bbs[1].PhiBase)+int32(16))
			if d4.Loc == scm.LocReg {
				ctx.UnprotectReg(d4.Reg)
			} else if d4.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d4.Reg)
				ctx.UnprotectReg(d4.Reg2)
			}
		}
		ps6 := scm.PhiState{General: ps.General}
		ps6.OverlayValues = make([]scm.JITValueDesc, 6)
		ps6.OverlayValues[1] = d1
		ps6.OverlayValues[2] = d2
		ps6.OverlayValues[3] = d3
		ps6.OverlayValues[4] = d4
		ps6.OverlayValues[5] = d5
		ps6.PhiValues = make([]scm.JITValueDesc, 2)
		d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps6.PhiValues[0] = d7
		d8 = d4
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
		var d11 scm.JITValueDesc
		if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d1.Imm.Int()) == uint64(d2.Imm.Int()))}
		} else if d2.Loc == scm.LocImm {
			r4 := ctx.AllocRegExcept(d1.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(d1.Reg, scm.RegR11)
			}
			d11 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r4, Condition: scm.CondEqual}
			ctx.BindReg(r4, &d11)
		} else if d1.Loc == scm.LocImm {
			r5 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d2.Reg)
			d11 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r5, Condition: scm.CondEqual}
			ctx.BindReg(r5, &d11)
		} else {
			r6 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpInt64(d1.Reg, d2.Reg)
			d11 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r6, Condition: scm.CondEqual}
			ctx.BindReg(r6, &d11)
		}
		d12 = d11
		ctx.EnsureDesc(&d12)
		if d12.Loc != scm.LocImm && d12.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d12.Loc == scm.LocImm {
			if d12.Imm.Bool() {
				if ps.General {
				}
				ps13 := scm.PhiState{General: ps.General}
				ps13.OverlayValues = make([]scm.JITValueDesc, 13)
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
			if ps.General {
			}
			ps14 := scm.PhiState{General: ps.General}
			ps14.OverlayValues = make([]scm.JITValueDesc, 13)
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
		ctx.EmitJump(d12.Condition, lbl3)
		ctx.FreeDesc(&d11)
		snap17 := d1
		snap18 := d2
		snap19 := d3
		snap20 := d4
		snap21 := d5
		snap22 := d7
		snap23 := d8
		snap24 := d9
		snap25 := d10
		snap26 := d11
		snap27 := d12
		snap28 := d15
		snap29 := d16
		alloc30 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc30)
		d1 = snap17
		d2 = snap18
		d3 = snap19
		d4 = snap20
		d5 = snap21
		d7 = snap22
		d8 = snap23
		d9 = snap24
		d10 = snap25
		d11 = snap26
		d12 = snap27
		d15 = snap28
		d16 = snap29
		ctx.RestoreAllocState(alloc30)
		d1 = snap17
		d2 = snap18
		d3 = snap19
		d4 = snap20
		d5 = snap21
		d7 = snap22
		d8 = snap23
		d9 = snap24
		d10 = snap25
		d11 = snap26
		d12 = snap27
		d15 = snap28
		d16 = snap29
		ps31 := scm.PhiState{General: true}
		ps31.OverlayValues = make([]scm.JITValueDesc, 17)
		ps31.OverlayValues[1] = d1
		ps31.OverlayValues[2] = d2
		ps31.OverlayValues[3] = d3
		ps31.OverlayValues[4] = d4
		ps31.OverlayValues[5] = d5
		ps31.OverlayValues[7] = d7
		ps31.OverlayValues[8] = d8
		ps31.OverlayValues[9] = d9
		ps31.OverlayValues[10] = d10
		ps31.OverlayValues[11] = d11
		ps31.OverlayValues[12] = d12
		ps31.OverlayValues[15] = d15
		ps31.OverlayValues[16] = d16
		ps32 := scm.PhiState{General: true}
		ps32.OverlayValues = make([]scm.JITValueDesc, 17)
		ps32.OverlayValues[1] = d1
		ps32.OverlayValues[2] = d2
		ps32.OverlayValues[3] = d3
		ps32.OverlayValues[4] = d4
		ps32.OverlayValues[5] = d5
		ps32.OverlayValues[7] = d7
		ps32.OverlayValues[8] = d8
		ps32.OverlayValues[9] = d9
		ps32.OverlayValues[10] = d10
		ps32.OverlayValues[11] = d11
		ps32.OverlayValues[12] = d12
		ps32.OverlayValues[15] = d15
		ps32.OverlayValues[16] = d16
		snap33 := d1
		snap34 := d2
		snap35 := d3
		snap36 := d4
		snap37 := d5
		snap38 := d7
		snap39 := d8
		snap40 := d9
		snap41 := d10
		snap42 := d11
		snap43 := d12
		snap44 := d15
		snap45 := d16
		alloc46 := ctx.SnapshotAllocState()
		if !bbs[3].Rendered {
			bbs[3].RenderPS(ps32)
		}
		ctx.RestoreAllocState(alloc46)
		d1 = snap33
		d2 = snap34
		d3 = snap35
		d4 = snap36
		d5 = snap37
		d7 = snap38
		d8 = snap39
		d9 = snap40
		d10 = snap41
		d11 = snap42
		d12 = snap43
		d15 = snap44
		d16 = snap45
		if !bbs[2].Rendered {
			return bbs[2].RenderPS(ps31)
		}
		return result
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
		d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d48 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d48)
		ctx.BindReg(r1, &d48)
		ctx.EnsureDesc(&d47)
		if d47.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d47, &d48)
		} else {
			switch d47.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d48, d47)
			case scm.TagInt:
				ctx.EmitMakeInt(d48, d47)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d48, d47)
			case scm.TagNil:
				ctx.EmitMakeNil(d48)
			default:
				ctx.EmitMovPairToResult(&d47, &d48)
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
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
			d47 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
			d48 = ps.OverlayValues[48]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&d1, &d2)
		var d49 scm.JITValueDesc
		if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + d2.Imm.Int())}
		} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
			r7 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(r7, d1.Reg)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d49)
		} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d2.Reg}
			ctx.BindReg(d2.Reg, &d49)
		} else if d1.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
			ctx.EmitAddInt32(scratch, d2.Reg)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d49)
		} else if d2.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32Low(scratch, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.EmitAddInt32(scratch, scm.RegR11)
			}
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d49)
		} else {
			r8 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
			ctx.EmitMovRegReg(r8, d1.Reg)
			ctx.EmitAddInt32(r8, d2.Reg)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d49)
		}
		if d49.Loc == scm.LocReg && d1.Loc == scm.LocReg && d49.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d49)
		var d50 scm.JITValueDesc
		if d49.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() / 2)}
		} else {
			r9 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitMovRegReg(r9, d49.Reg)
			ctx.EmitShrRegImm8(r9, 1)
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d50)
		}
		if d50.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: d50.Type, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d50.Reg, 32)
			ctx.EmitShrRegImm8(d50.Reg, 32)
		}
		if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
			ctx.TransferReg(d49.Reg)
			d49.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d50)
		ctx.FreeDesc(&d49)
		ctx.EnsureDesc(&d50)
		d51 = d50
		_ = d51
		ctx.StabilizeDescForControlFlow(&d50)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl9 := ctx.ReserveLabel()
		_ = lbl9
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl9)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d52 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 48)
			r10 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r10, thisptr.Reg, off)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			ctx.BindReg(r10, &d52)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d52)
		ctx.EnsureDesc(&d52)
		var d53 scm.JITValueDesc
		if d52.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d52.Imm.Int()))))}
		} else {
			r11 := ctx.AllocReg()
			ctx.EmitMovRegReg(r11, d52.Reg)
			ctx.EmitShlRegImm8(r11, 56)
			ctx.EmitShrRegImm8(r11, 56)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d53)
		}
		ctx.FreeDesc(&d52)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d51)
		var d54 scm.JITValueDesc
		if d51.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d51.Imm.Int()))))}
		} else {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegReg(r12, d51.Reg)
			ctx.EmitShlRegImm8(r12, 32)
			ctx.EmitShrRegImm8(r12, 32)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d54)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d54)
		ctx.EnsureDesc(&d53)
		ctx.EnsureDescsTogether(&d54, &d53)
		var d55 scm.JITValueDesc
		if d54.Loc == scm.LocImm && d53.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() * d53.Imm.Int())}
		} else if d54.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
			ctx.EmitImulInt64(scratch, d53.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d55)
		} else if d53.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(scratch, d54.Reg)
			if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d53.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d55)
		} else {
			r13 := ctx.AllocRegExcept(d54.Reg, d53.Reg)
			ctx.EmitMovRegReg(r13, d54.Reg)
			ctx.EmitImulInt64(r13, d53.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d55)
		}
		if d55.Loc == scm.LocReg && d54.Loc == scm.LocReg && d55.Reg == d54.Reg {
			ctx.TransferReg(d54.Reg)
			d54.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d54)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d55)
		var d56 scm.JITValueDesc
		if d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d55.Imm.Int() / 64)}
		} else {
			r14 := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegReg(r14, d55.Reg)
			ctx.EmitShrRegImm8(r14, 6)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d56)
		}
		if d56.Loc == scm.LocReg && d55.Loc == scm.LocReg && d56.Reg == d55.Reg {
			ctx.TransferReg(d55.Reg)
			d55.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d55)
		var d57 scm.JITValueDesc
		if d55.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d55.Imm.Int() % 64)}
		} else {
			r15 := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegReg(r15, d55.Reg)
			ctx.EmitAndRegImm32(r15, 63)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d57)
		}
		if d57.Loc == scm.LocReg && d55.Loc == scm.LocReg && d57.Reg == d55.Reg {
			ctx.TransferReg(d55.Reg)
			d55.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d55)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d58 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d58 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r16 := ctx.AllocReg()
			r17 := ctx.AllocRegExcept(r16)
			r18 := ctx.AllocRegExcept(r16, r17)
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off)
			ctx.EmitMovRegMem(r17, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r18, thisptr.Reg, off+16)
			d58 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r16, Reg2: r17, Reg3: r18}
			ctx.BindReg(r16, &d58)
			ctx.BindReg(r17, &d58)
			ctx.BindReg(r18, &d58)
			ctx.BindReg(r16, &d58)
			ctx.BindReg(r17, &d58)
			ctx.BindReg(r18, &d58)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d56)
		ctx.ReclaimUntrackedRegs()
		d59 = ctx.EmitLoadScalarSliceElement(&d58, &d56, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d59)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d59, &d57)
		var d60 scm.JITValueDesc
		if d59.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d59.Imm.Int()) << uint64(d57.Imm.Int())))}
		} else if d57.Loc == scm.LocImm {
			r19 := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegReg(r19, d59.Reg)
			ctx.EmitShlRegImm8(r19, uint8(d57.Imm.Int()))
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d60)
		} else {
			{
				shiftSrc := d59.Reg
				r20 := ctx.AllocRegExcept(d59.Reg, d57.Reg)
				ctx.EmitMovRegReg(r20, d59.Reg)
				shiftSrc = r20
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d57.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d57.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d57.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d60)
			}
		}
		if d60.Loc == scm.LocReg && d59.Loc == scm.LocReg && d60.Reg == d59.Reg {
			ctx.TransferReg(d59.Reg)
			d59.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d56)
		ctx.EnsureDesc(&d56)
		var d61 scm.JITValueDesc
		if d56.Loc == scm.LocImm {
			d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegReg(scratch, d56.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d61)
		}
		if d61.Loc == scm.LocReg && d56.Loc == scm.LocReg && d61.Reg == d56.Reg {
			ctx.TransferReg(d56.Reg)
			d56.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d56)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d61)
		ctx.ReclaimUntrackedRegs()
		d62 = ctx.EmitLoadScalarSliceElement(&d58, &d61, 8, scm.TagInt)
		ctx.FreeDesc(&d61)
		ctx.ReclaimUntrackedRegs()
		d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d63, &d57)
		var d64 scm.JITValueDesc
		if d63.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d63.Imm.Int() - d57.Imm.Int())}
		} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
			r21 := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitMovRegReg(r21, d63.Reg)
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d64)
		} else if d63.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
			ctx.EmitSubInt64(scratch, d57.Reg)
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d64)
		} else if d57.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitMovRegReg(scratch, d63.Reg)
			if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d57.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d64)
		} else {
			r22 := ctx.AllocRegExcept(d63.Reg, d57.Reg)
			ctx.EmitMovRegReg(r22, d63.Reg)
			ctx.EmitSubInt64(r22, d57.Reg)
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d64)
		}
		if d64.Loc == scm.LocReg && d63.Loc == scm.LocReg && d64.Reg == d63.Reg {
			ctx.TransferReg(d63.Reg)
			d63.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d57)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d62)
		ctx.EnsureDesc(&d64)
		ctx.EnsureDescsTogether(&d62, &d64)
		var d65 scm.JITValueDesc
		if d62.Loc == scm.LocImm && d64.Loc == scm.LocImm {
			d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d62.Imm.Int()) >> uint64(d64.Imm.Int())))}
		} else if d64.Loc == scm.LocImm {
			r23 := ctx.AllocRegExcept(d62.Reg)
			ctx.EmitMovRegReg(r23, d62.Reg)
			ctx.EmitShrRegImm8(r23, uint8(d64.Imm.Int()))
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d65)
		} else {
			{
				shiftSrc := d62.Reg
				r24 := ctx.AllocRegExcept(d62.Reg, d64.Reg)
				ctx.EmitMovRegReg(r24, d62.Reg)
				shiftSrc = r24
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d64.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d64.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d64.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d65)
			}
		}
		if d65.Loc == scm.LocReg && d62.Loc == scm.LocReg && d65.Reg == d62.Reg {
			ctx.TransferReg(d62.Reg)
			d62.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d62)
		ctx.FreeDesc(&d64)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d60)
		ctx.EnsureDesc(&d65)
		var d66 scm.JITValueDesc
		if d60.Loc == scm.LocImm && d65.Loc == scm.LocImm {
			d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d60.Imm.Int() | d65.Imm.Int())}
		} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			ctx.BindReg(d65.Reg, &d66)
		} else if d65.Loc == scm.LocImm && d65.Imm.Int() == 0 {
			r25 := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegReg(r25, d60.Reg)
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d66)
		} else if d60.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d65.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d60.Imm.Int()))
			ctx.EmitOrInt64(scratch, d65.Reg)
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d66)
		} else if d65.Loc == scm.LocImm {
			r26 := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegReg(r26, d60.Reg)
			if d65.Imm.Int() >= -2147483648 && d65.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r26, int32(d65.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d65.Imm.Int()))
				ctx.EmitOrInt64(r26, scm.RegR11)
			}
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d66)
		} else {
			r27 := ctx.AllocRegExcept(d60.Reg, d65.Reg)
			ctx.EmitMovRegReg(r27, d60.Reg)
			ctx.EmitOrInt64(r27, d65.Reg)
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d66)
		}
		if d66.Loc == scm.LocReg && d60.Loc == scm.LocReg && d66.Reg == d60.Reg {
			ctx.TransferReg(d60.Reg)
			d60.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d60)
		ctx.FreeDesc(&d65)
		ctx.ReclaimUntrackedRegs()
		d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d53)
		ctx.EnsureDescsTogether(&d67, &d53)
		var d68 scm.JITValueDesc
		if d67.Loc == scm.LocImm && d53.Loc == scm.LocImm {
			d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d67.Imm.Int() - d53.Imm.Int())}
		} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
			r28 := ctx.AllocRegExcept(d67.Reg)
			ctx.EmitMovRegReg(r28, d67.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d68)
		} else if d67.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d67.Imm.Int()))
			ctx.EmitSubInt64(scratch, d53.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d68)
		} else if d53.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d67.Reg)
			ctx.EmitMovRegReg(scratch, d67.Reg)
			if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d53.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d68)
		} else {
			r29 := ctx.AllocRegExcept(d67.Reg, d53.Reg)
			ctx.EmitMovRegReg(r29, d67.Reg)
			ctx.EmitSubInt64(r29, d53.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d68)
		}
		if d68.Loc == scm.LocReg && d67.Loc == scm.LocReg && d68.Reg == d67.Reg {
			ctx.TransferReg(d67.Reg)
			d67.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d53)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d66)
		ctx.EnsureDesc(&d68)
		ctx.EnsureDescsTogether(&d66, &d68)
		var d69 scm.JITValueDesc
		if d66.Loc == scm.LocImm && d68.Loc == scm.LocImm {
			d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d66.Imm.Int()) >> uint64(d68.Imm.Int())))}
		} else if d68.Loc == scm.LocImm {
			r30 := ctx.AllocRegExcept(d66.Reg)
			ctx.EmitMovRegReg(r30, d66.Reg)
			ctx.EmitShrRegImm8(r30, uint8(d68.Imm.Int()))
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d69)
		} else {
			{
				shiftSrc := d66.Reg
				r31 := ctx.AllocRegExcept(d66.Reg, d68.Reg)
				ctx.EmitMovRegReg(r31, d66.Reg)
				shiftSrc = r31
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d68.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d68.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d68.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d69)
			}
		}
		if d69.Loc == scm.LocReg && d66.Loc == scm.LocReg && d69.Reg == d66.Reg {
			ctx.TransferReg(d66.Reg)
			d66.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d66)
		ctx.FreeDesc(&d68)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d69)
		var d70 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 56)
			r32 := ctx.AllocReg()
			ctx.EmitMovRegMem(r32, thisptr.Reg, off)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d70)
		}
		ctx.EnsureDesc(&d70)
		ctx.EnsureDesc(&d70)
		var d71 scm.JITValueDesc
		if d70.Loc == scm.LocImm {
			d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d70.Imm.Int()))))}
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitMovRegReg(r33, d70.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d71)
		}
		ctx.FreeDesc(&d70)
		ctx.EnsureDesc(&d69)
		ctx.EnsureDesc(&d71)
		ctx.EnsureDescsTogether(&d69, &d71)
		var d72 scm.JITValueDesc
		if d69.Loc == scm.LocImm && d71.Loc == scm.LocImm {
			d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() + d71.Imm.Int())}
		} else if d71.Loc == scm.LocImm && d71.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d69.Reg)
			ctx.EmitMovRegReg(r34, d69.Reg)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d72)
		} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
			ctx.BindReg(d71.Reg, &d72)
		} else if d69.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d71.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
			ctx.EmitAddInt64(scratch, d71.Reg)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d72)
		} else if d71.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d69.Reg)
			ctx.EmitMovRegReg(scratch, d69.Reg)
			if d71.Imm.Int() >= -2147483648 && d71.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d71.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d71.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d72)
		} else {
			r35 := ctx.AllocRegExcept(d69.Reg, d71.Reg)
			ctx.EmitMovRegReg(r35, d69.Reg)
			ctx.EmitAddInt64(r35, d71.Reg)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d72)
		}
		if d72.Loc == scm.LocReg && d69.Loc == scm.LocReg && d72.Reg == d69.Reg {
			ctx.TransferReg(d69.Reg)
			d69.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d72)
		ctx.FreeDesc(&d69)
		ctx.FreeDesc(&d71)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d72)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDescsTogether(&d72, &idxInt)
		var d74 scm.JITValueDesc
		if d72.Loc == scm.LocImm && idxInt.Loc == scm.LocImm {
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d72.Imm.Int()) == uint64(idxInt.Imm.Int()))}
		} else if idxInt.Loc == scm.LocImm {
			r36 := ctx.AllocRegExcept(d72.Reg)
			if idxInt.Imm.Int() >= -2147483648 && idxInt.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d72.Reg, int32(idxInt.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(d72.Reg, scm.RegR11)
			}
			d74 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r36, Condition: scm.CondEqual}
			ctx.BindReg(r36, &d74)
		} else if d72.Loc == scm.LocImm {
			r37 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, idxInt.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r37, Condition: scm.CondEqual}
			ctx.BindReg(r37, &d74)
		} else {
			r38 := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitCmpInt64(d72.Reg, idxInt.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r38, Condition: scm.CondEqual}
			ctx.BindReg(r38, &d74)
		}
		d75 = d74
		ctx.EnsureDesc(&d75)
		if d75.Loc != scm.LocImm && d75.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d75.Loc == scm.LocImm {
			if d75.Imm.Bool() {
				if ps.General {
				}
				ps76 := scm.PhiState{General: ps.General}
				ps76.OverlayValues = make([]scm.JITValueDesc, 76)
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
				ps76.OverlayValues[75] = d75
				return bbs[4].RenderPS(ps76)
			}
			if ps.General {
			}
			ps77 := scm.PhiState{General: ps.General}
			ps77.OverlayValues = make([]scm.JITValueDesc, 76)
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
			return bbs[5].RenderPS(ps77)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		ctx.EmitJump(d75.Condition, lbl5)
		ctx.FreeDesc(&d74)
		snap78 := d1
		snap79 := d2
		snap80 := d3
		snap81 := d4
		snap82 := d5
		snap83 := d7
		snap84 := d8
		snap85 := d9
		snap86 := d10
		snap87 := d11
		snap88 := d12
		snap89 := d15
		snap90 := d16
		snap91 := d47
		snap92 := d48
		snap93 := d49
		snap94 := d50
		snap95 := d51
		snap96 := d52
		snap97 := d53
		snap98 := d54
		snap99 := d55
		snap100 := d56
		snap101 := d57
		snap102 := d58
		snap103 := d59
		snap104 := d60
		snap105 := d61
		snap106 := d62
		snap107 := d63
		snap108 := d64
		snap109 := d65
		snap110 := d66
		snap111 := d67
		snap112 := d68
		snap113 := d69
		snap114 := d70
		snap115 := d71
		snap116 := d72
		snap117 := d73
		snap118 := d74
		snap119 := d75
		alloc120 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc120)
		d1 = snap78
		d2 = snap79
		d3 = snap80
		d4 = snap81
		d5 = snap82
		d7 = snap83
		d8 = snap84
		d9 = snap85
		d10 = snap86
		d11 = snap87
		d12 = snap88
		d15 = snap89
		d16 = snap90
		d47 = snap91
		d48 = snap92
		d49 = snap93
		d50 = snap94
		d51 = snap95
		d52 = snap96
		d53 = snap97
		d54 = snap98
		d55 = snap99
		d56 = snap100
		d57 = snap101
		d58 = snap102
		d59 = snap103
		d60 = snap104
		d61 = snap105
		d62 = snap106
		d63 = snap107
		d64 = snap108
		d65 = snap109
		d66 = snap110
		d67 = snap111
		d68 = snap112
		d69 = snap113
		d70 = snap114
		d71 = snap115
		d72 = snap116
		d73 = snap117
		d74 = snap118
		d75 = snap119
		ctx.RestoreAllocState(alloc120)
		d1 = snap78
		d2 = snap79
		d3 = snap80
		d4 = snap81
		d5 = snap82
		d7 = snap83
		d8 = snap84
		d9 = snap85
		d10 = snap86
		d11 = snap87
		d12 = snap88
		d15 = snap89
		d16 = snap90
		d47 = snap91
		d48 = snap92
		d49 = snap93
		d50 = snap94
		d51 = snap95
		d52 = snap96
		d53 = snap97
		d54 = snap98
		d55 = snap99
		d56 = snap100
		d57 = snap101
		d58 = snap102
		d59 = snap103
		d60 = snap104
		d61 = snap105
		d62 = snap106
		d63 = snap107
		d64 = snap108
		d65 = snap109
		d66 = snap110
		d67 = snap111
		d68 = snap112
		d69 = snap113
		d70 = snap114
		d71 = snap115
		d72 = snap116
		d73 = snap117
		d74 = snap118
		d75 = snap119
		ps121 := scm.PhiState{General: true}
		ps121.OverlayValues = make([]scm.JITValueDesc, 76)
		ps121.OverlayValues[1] = d1
		ps121.OverlayValues[2] = d2
		ps121.OverlayValues[3] = d3
		ps121.OverlayValues[4] = d4
		ps121.OverlayValues[5] = d5
		ps121.OverlayValues[7] = d7
		ps121.OverlayValues[8] = d8
		ps121.OverlayValues[9] = d9
		ps121.OverlayValues[10] = d10
		ps121.OverlayValues[11] = d11
		ps121.OverlayValues[12] = d12
		ps121.OverlayValues[15] = d15
		ps121.OverlayValues[16] = d16
		ps121.OverlayValues[47] = d47
		ps121.OverlayValues[48] = d48
		ps121.OverlayValues[49] = d49
		ps121.OverlayValues[50] = d50
		ps121.OverlayValues[51] = d51
		ps121.OverlayValues[52] = d52
		ps121.OverlayValues[53] = d53
		ps121.OverlayValues[54] = d54
		ps121.OverlayValues[55] = d55
		ps121.OverlayValues[56] = d56
		ps121.OverlayValues[57] = d57
		ps121.OverlayValues[58] = d58
		ps121.OverlayValues[59] = d59
		ps121.OverlayValues[60] = d60
		ps121.OverlayValues[61] = d61
		ps121.OverlayValues[62] = d62
		ps121.OverlayValues[63] = d63
		ps121.OverlayValues[64] = d64
		ps121.OverlayValues[65] = d65
		ps121.OverlayValues[66] = d66
		ps121.OverlayValues[67] = d67
		ps121.OverlayValues[68] = d68
		ps121.OverlayValues[69] = d69
		ps121.OverlayValues[70] = d70
		ps121.OverlayValues[71] = d71
		ps121.OverlayValues[72] = d72
		ps121.OverlayValues[73] = d73
		ps121.OverlayValues[74] = d74
		ps121.OverlayValues[75] = d75
		ps122 := scm.PhiState{General: true}
		ps122.OverlayValues = make([]scm.JITValueDesc, 76)
		ps122.OverlayValues[1] = d1
		ps122.OverlayValues[2] = d2
		ps122.OverlayValues[3] = d3
		ps122.OverlayValues[4] = d4
		ps122.OverlayValues[5] = d5
		ps122.OverlayValues[7] = d7
		ps122.OverlayValues[8] = d8
		ps122.OverlayValues[9] = d9
		ps122.OverlayValues[10] = d10
		ps122.OverlayValues[11] = d11
		ps122.OverlayValues[12] = d12
		ps122.OverlayValues[15] = d15
		ps122.OverlayValues[16] = d16
		ps122.OverlayValues[47] = d47
		ps122.OverlayValues[48] = d48
		ps122.OverlayValues[49] = d49
		ps122.OverlayValues[50] = d50
		ps122.OverlayValues[51] = d51
		ps122.OverlayValues[52] = d52
		ps122.OverlayValues[53] = d53
		ps122.OverlayValues[54] = d54
		ps122.OverlayValues[55] = d55
		ps122.OverlayValues[56] = d56
		ps122.OverlayValues[57] = d57
		ps122.OverlayValues[58] = d58
		ps122.OverlayValues[59] = d59
		ps122.OverlayValues[60] = d60
		ps122.OverlayValues[61] = d61
		ps122.OverlayValues[62] = d62
		ps122.OverlayValues[63] = d63
		ps122.OverlayValues[64] = d64
		ps122.OverlayValues[65] = d65
		ps122.OverlayValues[66] = d66
		ps122.OverlayValues[67] = d67
		ps122.OverlayValues[68] = d68
		ps122.OverlayValues[69] = d69
		ps122.OverlayValues[70] = d70
		ps122.OverlayValues[71] = d71
		ps122.OverlayValues[72] = d72
		ps122.OverlayValues[73] = d73
		ps122.OverlayValues[74] = d74
		ps122.OverlayValues[75] = d75
		snap123 := d1
		snap124 := d2
		snap125 := d3
		snap126 := d4
		snap127 := d5
		snap128 := d7
		snap129 := d8
		snap130 := d9
		snap131 := d10
		snap132 := d11
		snap133 := d12
		snap134 := d15
		snap135 := d16
		snap136 := d47
		snap137 := d48
		snap138 := d49
		snap139 := d50
		snap140 := d51
		snap141 := d52
		snap142 := d53
		snap143 := d54
		snap144 := d55
		snap145 := d56
		snap146 := d57
		snap147 := d58
		snap148 := d59
		snap149 := d60
		snap150 := d61
		snap151 := d62
		snap152 := d63
		snap153 := d64
		snap154 := d65
		snap155 := d66
		snap156 := d67
		snap157 := d68
		snap158 := d69
		snap159 := d70
		snap160 := d71
		snap161 := d72
		snap162 := d73
		snap163 := d74
		snap164 := d75
		alloc165 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps122)
		}
		ctx.RestoreAllocState(alloc165)
		d1 = snap123
		d2 = snap124
		d3 = snap125
		d4 = snap126
		d5 = snap127
		d7 = snap128
		d8 = snap129
		d9 = snap130
		d10 = snap131
		d11 = snap132
		d12 = snap133
		d15 = snap134
		d16 = snap135
		d47 = snap136
		d48 = snap137
		d49 = snap138
		d50 = snap139
		d51 = snap140
		d52 = snap141
		d53 = snap142
		d54 = snap143
		d55 = snap144
		d56 = snap145
		d57 = snap146
		d58 = snap147
		d59 = snap148
		d60 = snap149
		d61 = snap150
		d62 = snap151
		d63 = snap152
		d64 = snap153
		d65 = snap154
		d66 = snap155
		d67 = snap156
		d68 = snap157
		d69 = snap158
		d70 = snap159
		d71 = snap160
		d72 = snap161
		d73 = snap162
		d74 = snap163
		d75 = snap164
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps121)
		}
		return result
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
		ctx.ReclaimUntrackedRegs()
		var d166 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d166 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r39 := ctx.AllocReg()
			r40 := ctx.AllocRegExcept(r39)
			r41 := ctx.AllocRegExcept(r39, r40)
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
			ctx.EmitMovRegMem(r39, thisptr.Reg, off)
			ctx.EmitMovRegMem(r40, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r41, thisptr.Reg, off+16)
			d166 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r39, Reg2: r40, Reg3: r41}
			ctx.BindReg(r39, &d166)
			ctx.BindReg(r40, &d166)
			ctx.BindReg(r41, &d166)
			ctx.BindReg(r39, &d166)
			ctx.BindReg(r40, &d166)
			ctx.BindReg(r41, &d166)
		}
		ctx.EnsureDesc(&d50)
		d168 = ctx.EmitSliceElementAddress(&d166, &d50, 16)
		ctx.EnsureDesc(&d168)
		r42 := ctx.AllocRegExcept(d168.Reg)
		ctx.EmitMovRegMem(r42, d168.Reg, 8)
		ctx.EmitMovRegMem(d168.Reg, d168.Reg, 0)
		d167 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d168.Reg, Reg2: r42}
		ctx.BindReg(d168.Reg, &d167)
		ctx.BindReg(r42, &d167)
		d169 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d169)
		ctx.BindReg(r1, &d169)
		ctx.EnsureDesc(&d167)
		if d167.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d167, &d169)
		} else {
			switch d167.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d169, d167)
			case scm.TagInt:
				ctx.EmitMakeInt(d169, d167)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d169, d167)
			case scm.TagNil:
				ctx.EmitMakeNil(d169)
			default:
				ctx.EmitMovPairToResult(&d167, &d169)
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
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
			d167 = ps.OverlayValues[167]
		}
		if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
			d168 = ps.OverlayValues[168]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d72)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDescsTogether(&d72, &idxInt)
		var d171 scm.JITValueDesc
		if d72.Loc == scm.LocImm && idxInt.Loc == scm.LocImm {
			d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d72.Imm.Int()) < uint64(idxInt.Imm.Int()))}
		} else if idxInt.Loc == scm.LocImm {
			r43 := ctx.AllocRegExcept(d72.Reg)
			if idxInt.Imm.Int() >= -2147483648 && idxInt.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d72.Reg, int32(idxInt.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(d72.Reg, scm.RegR11)
			}
			d171 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r43, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r43, &d171)
		} else if d72.Loc == scm.LocImm {
			r44 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, idxInt.Reg)
			d171 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r44, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r44, &d171)
		} else {
			r45 := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitCmpInt64(d72.Reg, idxInt.Reg)
			d171 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r45, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r45, &d171)
		}
		ctx.FreeDesc(&idxInt)
		d172 = d171
		ctx.EnsureDesc(&d172)
		if d172.Loc != scm.LocImm && d172.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d172.Loc == scm.LocImm {
			if d172.Imm.Bool() {
				if ps.General {
				}
				ps173 := scm.PhiState{General: ps.General}
				ps173.OverlayValues = make([]scm.JITValueDesc, 173)
				ps173.OverlayValues[1] = d1
				ps173.OverlayValues[2] = d2
				ps173.OverlayValues[3] = d3
				ps173.OverlayValues[4] = d4
				ps173.OverlayValues[5] = d5
				ps173.OverlayValues[7] = d7
				ps173.OverlayValues[8] = d8
				ps173.OverlayValues[9] = d9
				ps173.OverlayValues[10] = d10
				ps173.OverlayValues[11] = d11
				ps173.OverlayValues[12] = d12
				ps173.OverlayValues[15] = d15
				ps173.OverlayValues[16] = d16
				ps173.OverlayValues[47] = d47
				ps173.OverlayValues[48] = d48
				ps173.OverlayValues[49] = d49
				ps173.OverlayValues[50] = d50
				ps173.OverlayValues[51] = d51
				ps173.OverlayValues[52] = d52
				ps173.OverlayValues[53] = d53
				ps173.OverlayValues[54] = d54
				ps173.OverlayValues[55] = d55
				ps173.OverlayValues[56] = d56
				ps173.OverlayValues[57] = d57
				ps173.OverlayValues[58] = d58
				ps173.OverlayValues[59] = d59
				ps173.OverlayValues[60] = d60
				ps173.OverlayValues[61] = d61
				ps173.OverlayValues[62] = d62
				ps173.OverlayValues[63] = d63
				ps173.OverlayValues[64] = d64
				ps173.OverlayValues[65] = d65
				ps173.OverlayValues[66] = d66
				ps173.OverlayValues[67] = d67
				ps173.OverlayValues[68] = d68
				ps173.OverlayValues[69] = d69
				ps173.OverlayValues[70] = d70
				ps173.OverlayValues[71] = d71
				ps173.OverlayValues[72] = d72
				ps173.OverlayValues[73] = d73
				ps173.OverlayValues[74] = d74
				ps173.OverlayValues[75] = d75
				ps173.OverlayValues[166] = d166
				ps173.OverlayValues[167] = d167
				ps173.OverlayValues[168] = d168
				ps173.OverlayValues[169] = d169
				ps173.OverlayValues[170] = d170
				ps173.OverlayValues[171] = d171
				ps173.OverlayValues[172] = d172
				return bbs[6].RenderPS(ps173)
			}
			if ps.General {
			}
			ps174 := scm.PhiState{General: ps.General}
			ps174.OverlayValues = make([]scm.JITValueDesc, 173)
			ps174.OverlayValues[1] = d1
			ps174.OverlayValues[2] = d2
			ps174.OverlayValues[3] = d3
			ps174.OverlayValues[4] = d4
			ps174.OverlayValues[5] = d5
			ps174.OverlayValues[7] = d7
			ps174.OverlayValues[8] = d8
			ps174.OverlayValues[9] = d9
			ps174.OverlayValues[10] = d10
			ps174.OverlayValues[11] = d11
			ps174.OverlayValues[12] = d12
			ps174.OverlayValues[15] = d15
			ps174.OverlayValues[16] = d16
			ps174.OverlayValues[47] = d47
			ps174.OverlayValues[48] = d48
			ps174.OverlayValues[49] = d49
			ps174.OverlayValues[50] = d50
			ps174.OverlayValues[51] = d51
			ps174.OverlayValues[52] = d52
			ps174.OverlayValues[53] = d53
			ps174.OverlayValues[54] = d54
			ps174.OverlayValues[55] = d55
			ps174.OverlayValues[56] = d56
			ps174.OverlayValues[57] = d57
			ps174.OverlayValues[58] = d58
			ps174.OverlayValues[59] = d59
			ps174.OverlayValues[60] = d60
			ps174.OverlayValues[61] = d61
			ps174.OverlayValues[62] = d62
			ps174.OverlayValues[63] = d63
			ps174.OverlayValues[64] = d64
			ps174.OverlayValues[65] = d65
			ps174.OverlayValues[66] = d66
			ps174.OverlayValues[67] = d67
			ps174.OverlayValues[68] = d68
			ps174.OverlayValues[69] = d69
			ps174.OverlayValues[70] = d70
			ps174.OverlayValues[71] = d71
			ps174.OverlayValues[72] = d72
			ps174.OverlayValues[73] = d73
			ps174.OverlayValues[74] = d74
			ps174.OverlayValues[75] = d75
			ps174.OverlayValues[166] = d166
			ps174.OverlayValues[167] = d167
			ps174.OverlayValues[168] = d168
			ps174.OverlayValues[169] = d169
			ps174.OverlayValues[170] = d170
			ps174.OverlayValues[171] = d171
			ps174.OverlayValues[172] = d172
			return bbs[7].RenderPS(ps174)
		}
		if !ps.General {
			ps.General = true
			return bbs[5].RenderPS(ps)
		}
		ctx.EmitJump(d172.Condition, lbl7)
		ctx.FreeDesc(&d171)
		snap175 := d1
		snap176 := d2
		snap177 := d3
		snap178 := d4
		snap179 := d5
		snap180 := d7
		snap181 := d8
		snap182 := d9
		snap183 := d10
		snap184 := d11
		snap185 := d12
		snap186 := d15
		snap187 := d16
		snap188 := d47
		snap189 := d48
		snap190 := d49
		snap191 := d50
		snap192 := d51
		snap193 := d52
		snap194 := d53
		snap195 := d54
		snap196 := d55
		snap197 := d56
		snap198 := d57
		snap199 := d58
		snap200 := d59
		snap201 := d60
		snap202 := d61
		snap203 := d62
		snap204 := d63
		snap205 := d64
		snap206 := d65
		snap207 := d66
		snap208 := d67
		snap209 := d68
		snap210 := d69
		snap211 := d70
		snap212 := d71
		snap213 := d72
		snap214 := d73
		snap215 := d74
		snap216 := d75
		snap217 := d166
		snap218 := d167
		snap219 := d168
		snap220 := d169
		snap221 := d170
		snap222 := d171
		snap223 := d172
		alloc224 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc224)
		d1 = snap175
		d2 = snap176
		d3 = snap177
		d4 = snap178
		d5 = snap179
		d7 = snap180
		d8 = snap181
		d9 = snap182
		d10 = snap183
		d11 = snap184
		d12 = snap185
		d15 = snap186
		d16 = snap187
		d47 = snap188
		d48 = snap189
		d49 = snap190
		d50 = snap191
		d51 = snap192
		d52 = snap193
		d53 = snap194
		d54 = snap195
		d55 = snap196
		d56 = snap197
		d57 = snap198
		d58 = snap199
		d59 = snap200
		d60 = snap201
		d61 = snap202
		d62 = snap203
		d63 = snap204
		d64 = snap205
		d65 = snap206
		d66 = snap207
		d67 = snap208
		d68 = snap209
		d69 = snap210
		d70 = snap211
		d71 = snap212
		d72 = snap213
		d73 = snap214
		d74 = snap215
		d75 = snap216
		d166 = snap217
		d167 = snap218
		d168 = snap219
		d169 = snap220
		d170 = snap221
		d171 = snap222
		d172 = snap223
		ctx.RestoreAllocState(alloc224)
		d1 = snap175
		d2 = snap176
		d3 = snap177
		d4 = snap178
		d5 = snap179
		d7 = snap180
		d8 = snap181
		d9 = snap182
		d10 = snap183
		d11 = snap184
		d12 = snap185
		d15 = snap186
		d16 = snap187
		d47 = snap188
		d48 = snap189
		d49 = snap190
		d50 = snap191
		d51 = snap192
		d52 = snap193
		d53 = snap194
		d54 = snap195
		d55 = snap196
		d56 = snap197
		d57 = snap198
		d58 = snap199
		d59 = snap200
		d60 = snap201
		d61 = snap202
		d62 = snap203
		d63 = snap204
		d64 = snap205
		d65 = snap206
		d66 = snap207
		d67 = snap208
		d68 = snap209
		d69 = snap210
		d70 = snap211
		d71 = snap212
		d72 = snap213
		d73 = snap214
		d74 = snap215
		d75 = snap216
		d166 = snap217
		d167 = snap218
		d168 = snap219
		d169 = snap220
		d170 = snap221
		d171 = snap222
		d172 = snap223
		ps225 := scm.PhiState{General: true}
		ps225.OverlayValues = make([]scm.JITValueDesc, 173)
		ps225.OverlayValues[1] = d1
		ps225.OverlayValues[2] = d2
		ps225.OverlayValues[3] = d3
		ps225.OverlayValues[4] = d4
		ps225.OverlayValues[5] = d5
		ps225.OverlayValues[7] = d7
		ps225.OverlayValues[8] = d8
		ps225.OverlayValues[9] = d9
		ps225.OverlayValues[10] = d10
		ps225.OverlayValues[11] = d11
		ps225.OverlayValues[12] = d12
		ps225.OverlayValues[15] = d15
		ps225.OverlayValues[16] = d16
		ps225.OverlayValues[47] = d47
		ps225.OverlayValues[48] = d48
		ps225.OverlayValues[49] = d49
		ps225.OverlayValues[50] = d50
		ps225.OverlayValues[51] = d51
		ps225.OverlayValues[52] = d52
		ps225.OverlayValues[53] = d53
		ps225.OverlayValues[54] = d54
		ps225.OverlayValues[55] = d55
		ps225.OverlayValues[56] = d56
		ps225.OverlayValues[57] = d57
		ps225.OverlayValues[58] = d58
		ps225.OverlayValues[59] = d59
		ps225.OverlayValues[60] = d60
		ps225.OverlayValues[61] = d61
		ps225.OverlayValues[62] = d62
		ps225.OverlayValues[63] = d63
		ps225.OverlayValues[64] = d64
		ps225.OverlayValues[65] = d65
		ps225.OverlayValues[66] = d66
		ps225.OverlayValues[67] = d67
		ps225.OverlayValues[68] = d68
		ps225.OverlayValues[69] = d69
		ps225.OverlayValues[70] = d70
		ps225.OverlayValues[71] = d71
		ps225.OverlayValues[72] = d72
		ps225.OverlayValues[73] = d73
		ps225.OverlayValues[74] = d74
		ps225.OverlayValues[75] = d75
		ps225.OverlayValues[166] = d166
		ps225.OverlayValues[167] = d167
		ps225.OverlayValues[168] = d168
		ps225.OverlayValues[169] = d169
		ps225.OverlayValues[170] = d170
		ps225.OverlayValues[171] = d171
		ps225.OverlayValues[172] = d172
		ps226 := scm.PhiState{General: true}
		ps226.OverlayValues = make([]scm.JITValueDesc, 173)
		ps226.OverlayValues[1] = d1
		ps226.OverlayValues[2] = d2
		ps226.OverlayValues[3] = d3
		ps226.OverlayValues[4] = d4
		ps226.OverlayValues[5] = d5
		ps226.OverlayValues[7] = d7
		ps226.OverlayValues[8] = d8
		ps226.OverlayValues[9] = d9
		ps226.OverlayValues[10] = d10
		ps226.OverlayValues[11] = d11
		ps226.OverlayValues[12] = d12
		ps226.OverlayValues[15] = d15
		ps226.OverlayValues[16] = d16
		ps226.OverlayValues[47] = d47
		ps226.OverlayValues[48] = d48
		ps226.OverlayValues[49] = d49
		ps226.OverlayValues[50] = d50
		ps226.OverlayValues[51] = d51
		ps226.OverlayValues[52] = d52
		ps226.OverlayValues[53] = d53
		ps226.OverlayValues[54] = d54
		ps226.OverlayValues[55] = d55
		ps226.OverlayValues[56] = d56
		ps226.OverlayValues[57] = d57
		ps226.OverlayValues[58] = d58
		ps226.OverlayValues[59] = d59
		ps226.OverlayValues[60] = d60
		ps226.OverlayValues[61] = d61
		ps226.OverlayValues[62] = d62
		ps226.OverlayValues[63] = d63
		ps226.OverlayValues[64] = d64
		ps226.OverlayValues[65] = d65
		ps226.OverlayValues[66] = d66
		ps226.OverlayValues[67] = d67
		ps226.OverlayValues[68] = d68
		ps226.OverlayValues[69] = d69
		ps226.OverlayValues[70] = d70
		ps226.OverlayValues[71] = d71
		ps226.OverlayValues[72] = d72
		ps226.OverlayValues[73] = d73
		ps226.OverlayValues[74] = d74
		ps226.OverlayValues[75] = d75
		ps226.OverlayValues[166] = d166
		ps226.OverlayValues[167] = d167
		ps226.OverlayValues[168] = d168
		ps226.OverlayValues[169] = d169
		ps226.OverlayValues[170] = d170
		ps226.OverlayValues[171] = d171
		ps226.OverlayValues[172] = d172
		snap227 := d1
		snap228 := d2
		snap229 := d3
		snap230 := d4
		snap231 := d5
		snap232 := d7
		snap233 := d8
		snap234 := d9
		snap235 := d10
		snap236 := d11
		snap237 := d12
		snap238 := d15
		snap239 := d16
		snap240 := d47
		snap241 := d48
		snap242 := d49
		snap243 := d50
		snap244 := d51
		snap245 := d52
		snap246 := d53
		snap247 := d54
		snap248 := d55
		snap249 := d56
		snap250 := d57
		snap251 := d58
		snap252 := d59
		snap253 := d60
		snap254 := d61
		snap255 := d62
		snap256 := d63
		snap257 := d64
		snap258 := d65
		snap259 := d66
		snap260 := d67
		snap261 := d68
		snap262 := d69
		snap263 := d70
		snap264 := d71
		snap265 := d72
		snap266 := d73
		snap267 := d74
		snap268 := d75
		snap269 := d166
		snap270 := d167
		snap271 := d168
		snap272 := d169
		snap273 := d170
		snap274 := d171
		snap275 := d172
		alloc276 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps226)
		}
		ctx.RestoreAllocState(alloc276)
		d1 = snap227
		d2 = snap228
		d3 = snap229
		d4 = snap230
		d5 = snap231
		d7 = snap232
		d8 = snap233
		d9 = snap234
		d10 = snap235
		d11 = snap236
		d12 = snap237
		d15 = snap238
		d16 = snap239
		d47 = snap240
		d48 = snap241
		d49 = snap242
		d50 = snap243
		d51 = snap244
		d52 = snap245
		d53 = snap246
		d54 = snap247
		d55 = snap248
		d56 = snap249
		d57 = snap250
		d58 = snap251
		d59 = snap252
		d60 = snap253
		d61 = snap254
		d62 = snap255
		d63 = snap256
		d64 = snap257
		d65 = snap258
		d66 = snap259
		d67 = snap260
		d68 = snap261
		d69 = snap262
		d70 = snap263
		d71 = snap264
		d72 = snap265
		d73 = snap266
		d74 = snap267
		d75 = snap268
		d166 = snap269
		d167 = snap270
		d168 = snap271
		d169 = snap272
		d170 = snap273
		d171 = snap274
		d172 = snap275
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps225)
		}
		return result
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
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
			d167 = ps.OverlayValues[167]
		}
		if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
			d168 = ps.OverlayValues[168]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
			d170 = ps.OverlayValues[170]
		}
		if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
			d171 = ps.OverlayValues[171]
		}
		if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
			d172 = ps.OverlayValues[172]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d50)
		ctx.EnsureDesc(&d50)
		var d277 scm.JITValueDesc
		if d50.Loc == scm.LocImm {
			d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegReg(scratch, d50.Reg)
			ctx.EmitAddRegImm32Low(scratch, int32(1))
			d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d277)
		}
		if d277.Loc == scm.LocReg && d50.Loc == scm.LocReg && d277.Reg == d50.Reg {
			ctx.TransferReg(d50.Reg)
			d50.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d277)
		ctx.EmitStoreToStack(d277, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d277)
		if ps.General {
		}
		ps278 := scm.PhiState{General: ps.General}
		ps278.OverlayValues = make([]scm.JITValueDesc, 278)
		ps278.OverlayValues[1] = d1
		ps278.OverlayValues[2] = d2
		ps278.OverlayValues[3] = d3
		ps278.OverlayValues[4] = d4
		ps278.OverlayValues[5] = d5
		ps278.OverlayValues[7] = d7
		ps278.OverlayValues[8] = d8
		ps278.OverlayValues[9] = d9
		ps278.OverlayValues[10] = d10
		ps278.OverlayValues[11] = d11
		ps278.OverlayValues[12] = d12
		ps278.OverlayValues[15] = d15
		ps278.OverlayValues[16] = d16
		ps278.OverlayValues[47] = d47
		ps278.OverlayValues[48] = d48
		ps278.OverlayValues[49] = d49
		ps278.OverlayValues[50] = d50
		ps278.OverlayValues[51] = d51
		ps278.OverlayValues[52] = d52
		ps278.OverlayValues[53] = d53
		ps278.OverlayValues[54] = d54
		ps278.OverlayValues[55] = d55
		ps278.OverlayValues[56] = d56
		ps278.OverlayValues[57] = d57
		ps278.OverlayValues[58] = d58
		ps278.OverlayValues[59] = d59
		ps278.OverlayValues[60] = d60
		ps278.OverlayValues[61] = d61
		ps278.OverlayValues[62] = d62
		ps278.OverlayValues[63] = d63
		ps278.OverlayValues[64] = d64
		ps278.OverlayValues[65] = d65
		ps278.OverlayValues[66] = d66
		ps278.OverlayValues[67] = d67
		ps278.OverlayValues[68] = d68
		ps278.OverlayValues[69] = d69
		ps278.OverlayValues[70] = d70
		ps278.OverlayValues[71] = d71
		ps278.OverlayValues[72] = d72
		ps278.OverlayValues[73] = d73
		ps278.OverlayValues[74] = d74
		ps278.OverlayValues[75] = d75
		ps278.OverlayValues[166] = d166
		ps278.OverlayValues[167] = d167
		ps278.OverlayValues[168] = d168
		ps278.OverlayValues[169] = d169
		ps278.OverlayValues[170] = d170
		ps278.OverlayValues[171] = d171
		ps278.OverlayValues[172] = d172
		ps278.OverlayValues[277] = d277
		ps278.PhiValues = make([]scm.JITValueDesc, 2)
		if ps278.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps278)
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
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
			d167 = ps.OverlayValues[167]
		}
		if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
			d168 = ps.OverlayValues[168]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
			d170 = ps.OverlayValues[170]
		}
		if len(ps.OverlayValues) > 171 && ps.OverlayValues[171].Loc != scm.LocNone {
			d171 = ps.OverlayValues[171]
		}
		if len(ps.OverlayValues) > 172 && ps.OverlayValues[172].Loc != scm.LocNone {
			d172 = ps.OverlayValues[172]
		}
		if len(ps.OverlayValues) > 277 && ps.OverlayValues[277].Loc != scm.LocNone {
			d277 = ps.OverlayValues[277]
		}
		ctx.ReclaimUntrackedRegs()
		if ps.General {
			ctx.SyncDesc(&d50)
			if d50.Loc == scm.LocReg {
				ctx.ProtectReg(d50.Reg)
			} else if d50.Loc == scm.LocRegPair {
				ctx.ProtectReg(d50.Reg)
				ctx.ProtectReg(d50.Reg2)
			}
			d279 = d50
			if d279.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d279)
			d280 = d279
			if d280.Loc == scm.LocImm {
				d280 = scm.JITValueDesc{Loc: scm.LocImm, Type: d280.Type, Imm: scm.NewInt(int64(uint64(d280.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d280.Reg, 32)
				ctx.EmitShrRegImm8(d280.Reg, 32)
			}
			ctx.EmitStoreToStack(d280, int32(bbs[1].PhiBase)+int32(16))
			if d50.Loc == scm.LocReg {
				ctx.UnprotectReg(d50.Reg)
			} else if d50.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d50.Reg)
				ctx.UnprotectReg(d50.Reg2)
			}
		}
		ps281 := scm.PhiState{General: ps.General}
		ps281.OverlayValues = make([]scm.JITValueDesc, 281)
		ps281.OverlayValues[1] = d1
		ps281.OverlayValues[2] = d2
		ps281.OverlayValues[3] = d3
		ps281.OverlayValues[4] = d4
		ps281.OverlayValues[5] = d5
		ps281.OverlayValues[7] = d7
		ps281.OverlayValues[8] = d8
		ps281.OverlayValues[9] = d9
		ps281.OverlayValues[10] = d10
		ps281.OverlayValues[11] = d11
		ps281.OverlayValues[12] = d12
		ps281.OverlayValues[15] = d15
		ps281.OverlayValues[16] = d16
		ps281.OverlayValues[47] = d47
		ps281.OverlayValues[48] = d48
		ps281.OverlayValues[49] = d49
		ps281.OverlayValues[50] = d50
		ps281.OverlayValues[51] = d51
		ps281.OverlayValues[52] = d52
		ps281.OverlayValues[53] = d53
		ps281.OverlayValues[54] = d54
		ps281.OverlayValues[55] = d55
		ps281.OverlayValues[56] = d56
		ps281.OverlayValues[57] = d57
		ps281.OverlayValues[58] = d58
		ps281.OverlayValues[59] = d59
		ps281.OverlayValues[60] = d60
		ps281.OverlayValues[61] = d61
		ps281.OverlayValues[62] = d62
		ps281.OverlayValues[63] = d63
		ps281.OverlayValues[64] = d64
		ps281.OverlayValues[65] = d65
		ps281.OverlayValues[66] = d66
		ps281.OverlayValues[67] = d67
		ps281.OverlayValues[68] = d68
		ps281.OverlayValues[69] = d69
		ps281.OverlayValues[70] = d70
		ps281.OverlayValues[71] = d71
		ps281.OverlayValues[72] = d72
		ps281.OverlayValues[73] = d73
		ps281.OverlayValues[74] = d74
		ps281.OverlayValues[75] = d75
		ps281.OverlayValues[166] = d166
		ps281.OverlayValues[167] = d167
		ps281.OverlayValues[168] = d168
		ps281.OverlayValues[169] = d169
		ps281.OverlayValues[170] = d170
		ps281.OverlayValues[171] = d171
		ps281.OverlayValues[172] = d172
		ps281.OverlayValues[277] = d277
		ps281.OverlayValues[279] = d279
		ps281.OverlayValues[280] = d280
		ps281.PhiValues = make([]scm.JITValueDesc, 2)
		d282 = d50
		ps281.PhiValues[1] = d282
		if ps281.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps281)
		return result
	}
	ps283 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps283)
	ctx.MarkLabel(lbl0)
	d284 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d284)
	ctx.BindReg(r1, &d284)
	ctx.EmitMovPairToResult(&d284, &result)
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
