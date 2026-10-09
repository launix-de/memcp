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
	sz := uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.recids)) + s.recids.ComputeSize() + uint(cap(s.values)-len(s.values))*uint(unsafe.Sizeof(scm.Scmer{}))
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
	var d7 scm.JITValueDesc
	_ = d7
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
	var d82 scm.JITValueDesc
	_ = d82
	var d83 scm.JITValueDesc
	_ = d83
	var d84 scm.JITValueDesc
	_ = d84
	var d85 scm.JITValueDesc
	_ = d85
	var d86 scm.JITValueDesc
	_ = d86
	var d87 scm.JITValueDesc
	_ = d87
	var d88 scm.JITValueDesc
	_ = d88
	var d133 scm.JITValueDesc
	_ = d133
	var d134 scm.JITValueDesc
	_ = d134
	var d135 scm.JITValueDesc
	_ = d135
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
	bbs[0].Render = func() scm.JITValueDesc {
		if bbs[0].Rendered {
			ctx.EmitJmp(lbl1)
			return result
		}
		bbs[0].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl1)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).i)
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
			r0 := ctx.AllocReg()
			ctx.EmitMovRegMem(r0, thisptr.Reg, off)
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
			ctx.BindReg(r0, &d3)
		}
		ctx.EnsureDesc(&d3)
		ctx.EnsureDesc(&d3)
		if d3.Loc == scm.LocImm {
			d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d3.Imm.Int()))))}
		} else {
			r1 := ctx.AllocReg()
			ctx.EmitMovRegReg(r1, d3.Reg)
			ctx.EmitShlRegImm8(r1, 32)
			ctx.EmitShrRegImm8(r1, 32)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1}
			ctx.BindReg(r1, &d4)
		}
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.FreeDesc(&d3)
		ctx.SyncDesc(&d4)
		if d4.Loc == scm.LocReg || d4.Loc == scm.LocFPReg {
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
		if d4.Loc == scm.LocReg || d4.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d4.Reg)
		} else if d4.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d4.Reg)
			ctx.UnprotectReg(d4.Reg2)
		}
		return bbs[1].Render()
		return result
	}
	bbs[1].Render = func() scm.JITValueDesc {
		if bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		bbs[1].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl2)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&d1, &d2)
		if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d1.Imm.Int()) == uint64(d2.Imm.Int()))}
		} else if d2.Loc == scm.LocImm {
			r2 := ctx.AllocRegExcept(d1.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d2.Imm.Int()))
				ctx.EmitCmpInt64(d1.Reg, ctx.ScratchReg)
			}
			d6 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r2, Condition: scm.CondEqual}
			ctx.BindReg(r2, &d6)
		} else if d1.Loc == scm.LocImm {
			r3 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d1.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d2.Reg)
			d6 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r3, Condition: scm.CondEqual}
			ctx.BindReg(r3, &d6)
		} else {
			r4 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitCmpInt64(d1.Reg, d2.Reg)
			d6 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r4, Condition: scm.CondEqual}
			ctx.BindReg(r4, &d6)
		}
		d7 = d6
		ctx.EnsureDesc(&d7)
		if d7.Loc != scm.LocImm && d7.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d7.Loc == scm.LocImm {
			if d7.Imm.Bool() {
				return bbs[2].Render()
			}
			return bbs[3].Render()
		}
		ctx.EmitJump(d7.Condition, lbl3)
		if bbs[3].Rendered {
			ctx.EmitJmp(lbl4)
		}
		ctx.FreeDesc(&d6)
		ctx.FlushRegisterMoves()
		if !bbs[3].Rendered {
			snap8 := d1
			snap9 := d2
			snap10 := d3
			snap11 := d4
			snap12 := d5
			snap13 := d6
			snap14 := d7
			alloc15 := ctx.SnapshotAllocState()
			bbs[3].Render()
			ctx.RestoreAllocState(alloc15)
			d1 = snap8
			d2 = snap9
			d3 = snap10
			d4 = snap11
			d5 = snap12
			d6 = snap13
			d7 = snap14
		}
		if !bbs[2].Rendered {
			return bbs[2].Render()
		}
		return result
		return result
	}
	bbs[2].Render = func() scm.JITValueDesc {
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
			return result
		}
		bbs[2].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl3)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d17 = result
		ctx.EnsureDesc(&d16)
		if d16.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d16, &d17)
		} else {
			switch d16.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d17, d16)
			case scm.TagInt:
				ctx.EmitMakeInt(d17, d16)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d17, d16)
			case scm.TagNil:
				ctx.EmitMakeNil(d17)
			default:
				ctx.EmitMovPairToResult(&d16, &d17)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[3].Render = func() scm.JITValueDesc {
		if bbs[3].Rendered {
			ctx.EmitJmp(lbl4)
			return result
		}
		bbs[3].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl4)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&d1, &d2)
		if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + d2.Imm.Int())}
		} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
			ctx.EnsureDesc(&d1)
			r5 := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(r5, d1.Reg)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d18)
		} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
			ctx.EnsureDesc(&d2)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d2.Reg}
			ctx.BindReg(d2.Reg, &d18)
		} else if d1.Loc == scm.LocImm {
			ctx.EnsureDesc(&d2)
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegReg(scratch, d2.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 32, scratch, d1.Imm.Int())
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d18)
		} else if d2.Loc == scm.LocImm {
			ctx.EnsureDesc(&d1)
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 32, scratch, d2.Imm.Int())
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d18)
		} else {
			ctx.EnsureDesc(&d1)
			ctx.SyncDesc(&d2)
			r6 := ctx.AllocRegExcept(d1.Reg, d2.Reg)
			ctx.EmitMovRegReg(r6, d1.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 32, r6, &d2)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d18)
		}
		if d18.Loc == scm.LocReg && d1.Loc == scm.LocReg && d18.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d18)
		if d18.Loc == scm.LocImm {
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() / 2)}
		} else {
			r7 := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitMovRegReg(r7, d18.Reg)
			ctx.EmitShrRegImm8(r7, 1)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d19)
		}
		if d19.Loc == scm.LocImm {
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: d19.Type, Imm: scm.NewInt(int64(uint64(d19.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d19.Reg, 32)
			ctx.EmitShrRegImm8(d19.Reg, 32)
		}
		if d19.Loc == scm.LocReg && d18.Loc == scm.LocReg && d19.Reg == d18.Reg {
			ctx.TransferReg(d18.Reg)
			d18.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d19)
		ctx.FreeDesc(&d18)
		ctx.EnsureDesc(&d19)
		d20 = d19
		_ = d20
		ctx.StabilizeDescForControlFlow(&d19)
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
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 48)
			r8 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r8, thisptr.Reg, off)
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			ctx.BindReg(r8, &d21)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d21)
		if d21.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d21.Imm.Int()))))}
		} else {
			r9 := ctx.AllocReg()
			ctx.EmitMovRegReg(r9, d21.Reg)
			ctx.EmitShlRegImm8(r9, 56)
			ctx.EmitShrRegImm8(r9, 56)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d22)
		}
		ctx.FreeDesc(&d21)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d20)
		ctx.EnsureDesc(&d20)
		if d20.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d20.Imm.Int()))))}
		} else {
			r10 := ctx.AllocReg()
			ctx.EmitMovRegReg(r10, d20.Reg)
			ctx.EmitShlRegImm8(r10, 32)
			ctx.EmitShrRegImm8(r10, 32)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d23)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d23)
		ctx.EnsureDesc(&d22)
		ctx.EnsureDescsTogether(&d23, &d22)
		if d23.Loc == scm.LocImm && d22.Loc == scm.LocImm {
			d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() * d22.Imm.Int())}
		} else if d23.Loc == scm.LocImm {
			ctx.EnsureDesc(&d22)
			scratch := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(scratch, d22.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d23.Imm.Int())
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d24)
		} else if d22.Loc == scm.LocImm {
			ctx.EnsureDesc(&d23)
			scratch := ctx.AllocRegExcept(d23.Reg)
			ctx.EmitMovRegReg(scratch, d23.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d22.Imm.Int())
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d24)
		} else {
			ctx.EnsureDesc(&d23)
			ctx.SyncDesc(&d22)
			r11 := ctx.AllocRegExcept(d23.Reg, d22.Reg)
			ctx.EmitMovRegReg(r11, d23.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r11, &d22)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d24)
		}
		if d24.Loc == scm.LocReg && d23.Loc == scm.LocReg && d24.Reg == d23.Reg {
			ctx.TransferReg(d23.Reg)
			d23.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		if d24.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() / 64)}
		} else {
			r12 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r12, d24.Reg)
			ctx.EmitShrRegImm8(r12, 6)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d25)
		}
		if d25.Loc == scm.LocReg && d24.Loc == scm.LocReg && d25.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		if d24.Loc == scm.LocImm {
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() % 64)}
		} else {
			r13 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r13, d24.Reg)
			ctx.EmitAndRegImm32(r13, 63)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d26)
		}
		if d26.Loc == scm.LocReg && d24.Loc == scm.LocReg && d26.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d27 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r14 := ctx.AllocReg()
			r15 := ctx.AllocRegExcept(r14)
			r16 := ctx.AllocRegExcept(r14, r15)
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off)
			ctx.EmitMovRegMem(r15, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r16, thisptr.Reg, off+16)
			d27 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r14, Reg2: r15, Reg3: r16}
			ctx.BindReg(r14, &d27)
			ctx.BindReg(r15, &d27)
			ctx.BindReg(r16, &d27)
			ctx.BindReg(r14, &d27)
			ctx.BindReg(r15, &d27)
			ctx.BindReg(r16, &d27)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d25)
		ctx.ReclaimUntrackedRegs()
		d28 = ctx.EmitLoadScalarSliceElement(&d27, &d25, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d26)
		ctx.EnsureDescsTogether(&d28, &d26)
		if d28.Loc == scm.LocImm && d26.Loc == scm.LocImm {
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d28.Imm.Int()) << uint64(d26.Imm.Int())))}
		} else if d26.Loc == scm.LocImm {
			r17 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r17, d28.Reg)
			ctx.EmitShlRegImm8(r17, uint8(d26.Imm.Int()))
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d29)
		} else {
			shiftSrc := d28.Reg
			r18 := ctx.AllocRegExcept(d28.Reg, d26.Reg)
			ctx.EmitMovRegReg(r18, d28.Reg)
			shiftSrc = r18
			ctx.EmitShiftLeft(shiftSrc, d26.Reg, true)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d29)
		}
		if d29.Loc == scm.LocReg && d28.Loc == scm.LocReg && d29.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d25)
		ctx.EnsureDesc(&d25)
		if d25.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegReg(scratch, d25.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		}
		if d30.Loc == scm.LocReg && d25.Loc == scm.LocReg && d30.Reg == d25.Reg {
			ctx.TransferReg(d25.Reg)
			d25.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d25)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		d31 = ctx.EmitLoadScalarSliceElement(&d27, &d30, 8, scm.TagInt)
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d26)
		ctx.EnsureDescsTogether(&d32, &d26)
		if d32.Loc == scm.LocImm && d26.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() - d26.Imm.Int())}
		} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
			ctx.EnsureDesc(&d32)
			r19 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r19, d32.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d33)
		} else if d32.Loc == scm.LocImm {
			ctx.EnsureDesc(&d26)
			scratch := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d26)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d33)
		} else if d26.Loc == scm.LocImm {
			ctx.EnsureDesc(&d32)
			scratch := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(scratch, d32.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d26.Imm.Int())
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d33)
		} else {
			ctx.EnsureDesc(&d32)
			ctx.SyncDesc(&d26)
			r20 := ctx.AllocRegExcept(d32.Reg, d26.Reg)
			ctx.EmitMovRegReg(r20, d32.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r20, &d26)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d33)
		}
		if d33.Loc == scm.LocReg && d32.Loc == scm.LocReg && d33.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d26)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d33)
		ctx.EnsureDescsTogether(&d31, &d33)
		if d31.Loc == scm.LocImm && d33.Loc == scm.LocImm {
			d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d31.Imm.Int()) >> uint64(d33.Imm.Int())))}
		} else if d33.Loc == scm.LocImm {
			r21 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r21, d31.Reg)
			ctx.EmitShrRegImm8(r21, uint8(d33.Imm.Int()))
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d34)
		} else {
			shiftSrc := d31.Reg
			r22 := ctx.AllocRegExcept(d31.Reg, d33.Reg)
			ctx.EmitMovRegReg(r22, d31.Reg)
			shiftSrc = r22
			ctx.EmitShiftRight(shiftSrc, d33.Reg, false)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d34)
		}
		if d34.Loc == scm.LocReg && d31.Loc == scm.LocReg && d34.Reg == d31.Reg {
			ctx.TransferReg(d31.Reg)
			d31.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d31)
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d29)
		ctx.EnsureDesc(&d34)
		if d29.Loc == scm.LocImm && d34.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() | d34.Imm.Int())}
		} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
			ctx.BindReg(d34.Reg, &d35)
		} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
			r23 := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(r23, d29.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d35)
		} else if d29.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
			ctx.EmitOrInt64(scratch, d34.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d35)
		} else if d34.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(r24, d29.Reg)
			if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r24, int32(d34.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d34.Imm.Int()))
				ctx.EmitOrInt64(r24, ctx.ScratchReg)
			}
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d35)
		} else {
			r25 := ctx.AllocRegExcept(d29.Reg, d34.Reg)
			ctx.EmitMovRegReg(r25, d29.Reg)
			ctx.EmitOrInt64(r25, d34.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d35)
		}
		if d35.Loc == scm.LocReg && d29.Loc == scm.LocReg && d35.Reg == d29.Reg {
			ctx.TransferReg(d29.Reg)
			d29.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d29)
		ctx.FreeDesc(&d34)
		ctx.ReclaimUntrackedRegs()
		d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d22)
		ctx.EnsureDescsTogether(&d36, &d22)
		if d36.Loc == scm.LocImm && d22.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() - d22.Imm.Int())}
		} else if d22.Loc == scm.LocImm && d22.Imm.Int() == 0 {
			ctx.EnsureDesc(&d36)
			r26 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r26, d36.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d37)
		} else if d36.Loc == scm.LocImm {
			ctx.EnsureDesc(&d22)
			scratch := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d22)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else if d22.Loc == scm.LocImm {
			ctx.EnsureDesc(&d36)
			scratch := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(scratch, d36.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d22.Imm.Int())
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else {
			ctx.EnsureDesc(&d36)
			ctx.SyncDesc(&d22)
			r27 := ctx.AllocRegExcept(d36.Reg, d22.Reg)
			ctx.EmitMovRegReg(r27, d36.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r27, &d22)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d37)
		}
		if d37.Loc == scm.LocReg && d36.Loc == scm.LocReg && d37.Reg == d36.Reg {
			ctx.TransferReg(d36.Reg)
			d36.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d22)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d37)
		ctx.EnsureDescsTogether(&d35, &d37)
		if d35.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d35.Imm.Int()) >> uint64(d37.Imm.Int())))}
		} else if d37.Loc == scm.LocImm {
			r28 := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(r28, d35.Reg)
			ctx.EmitShrRegImm8(r28, uint8(d37.Imm.Int()))
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d38)
		} else {
			shiftSrc := d35.Reg
			r29 := ctx.AllocRegExcept(d35.Reg, d37.Reg)
			ctx.EmitMovRegReg(r29, d35.Reg)
			shiftSrc = r29
			ctx.EmitShiftRight(shiftSrc, d37.Reg, false)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d38)
		}
		if d38.Loc == scm.LocReg && d35.Loc == scm.LocReg && d38.Reg == d35.Reg {
			ctx.TransferReg(d35.Reg)
			d35.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d35)
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 56)
			r30 := ctx.AllocReg()
			ctx.EmitMovRegMem(r30, thisptr.Reg, off)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			ctx.BindReg(r30, &d39)
		}
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d39)
		if d39.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d39.Imm.Int()))))}
		} else {
			r31 := ctx.AllocReg()
			ctx.EmitMovRegReg(r31, d39.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d40)
		}
		ctx.FreeDesc(&d39)
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d40)
		ctx.EnsureDescsTogether(&d38, &d40)
		if d38.Loc == scm.LocImm && d40.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + d40.Imm.Int())}
		} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
			ctx.EnsureDesc(&d38)
			r32 := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(r32, d38.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d41)
		} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
			ctx.EnsureDesc(&d40)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
			ctx.BindReg(d40.Reg, &d41)
		} else if d38.Loc == scm.LocImm {
			ctx.EnsureDesc(&d40)
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(scratch, d40.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d38.Imm.Int())
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else if d40.Loc == scm.LocImm {
			ctx.EnsureDesc(&d38)
			scratch := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegReg(scratch, d38.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d40.Imm.Int())
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else {
			ctx.EnsureDesc(&d38)
			ctx.SyncDesc(&d40)
			r33 := ctx.AllocRegExcept(d38.Reg, d40.Reg)
			ctx.EmitMovRegReg(r33, d38.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r33, &d40)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d41)
		}
		if d41.Loc == scm.LocReg && d38.Loc == scm.LocReg && d41.Reg == d38.Reg {
			ctx.TransferReg(d38.Reg)
			d38.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d38)
		ctx.FreeDesc(&d40)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDescsTogether(&d41, &idxInt)
		if d41.Loc == scm.LocImm && idxInt.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d41.Imm.Int()) == uint64(idxInt.Imm.Int()))}
		} else if idxInt.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(d41.Reg)
			if idxInt.Imm.Int() >= -2147483648 && idxInt.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d41.Reg, int32(idxInt.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(d41.Reg, ctx.ScratchReg)
			}
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r34, Condition: scm.CondEqual}
			ctx.BindReg(r34, &d43)
		} else if d41.Loc == scm.LocImm {
			r35 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d41.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, idxInt.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r35, Condition: scm.CondEqual}
			ctx.BindReg(r35, &d43)
		} else {
			r36 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitCmpInt64(d41.Reg, idxInt.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r36, Condition: scm.CondEqual}
			ctx.BindReg(r36, &d43)
		}
		d44 = d43
		ctx.EnsureDesc(&d44)
		if d44.Loc != scm.LocImm && d44.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d44.Loc == scm.LocImm {
			if d44.Imm.Bool() {
				return bbs[4].Render()
			}
			return bbs[5].Render()
		}
		ctx.EmitJump(d44.Condition, lbl5)
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
		}
		ctx.FreeDesc(&d43)
		ctx.FlushRegisterMoves()
		if !bbs[5].Rendered {
			snap45 := d1
			snap46 := d2
			snap47 := d3
			snap48 := d4
			snap49 := d5
			snap50 := d6
			snap51 := d7
			snap52 := d16
			snap53 := d17
			snap54 := d18
			snap55 := d19
			snap56 := d20
			snap57 := d21
			snap58 := d22
			snap59 := d23
			snap60 := d24
			snap61 := d25
			snap62 := d26
			snap63 := d27
			snap64 := d28
			snap65 := d29
			snap66 := d30
			snap67 := d31
			snap68 := d32
			snap69 := d33
			snap70 := d34
			snap71 := d35
			snap72 := d36
			snap73 := d37
			snap74 := d38
			snap75 := d39
			snap76 := d40
			snap77 := d41
			snap78 := d42
			snap79 := d43
			snap80 := d44
			alloc81 := ctx.SnapshotAllocState()
			bbs[5].Render()
			ctx.RestoreAllocState(alloc81)
			d1 = snap45
			d2 = snap46
			d3 = snap47
			d4 = snap48
			d5 = snap49
			d6 = snap50
			d7 = snap51
			d16 = snap52
			d17 = snap53
			d18 = snap54
			d19 = snap55
			d20 = snap56
			d21 = snap57
			d22 = snap58
			d23 = snap59
			d24 = snap60
			d25 = snap61
			d26 = snap62
			d27 = snap63
			d28 = snap64
			d29 = snap65
			d30 = snap66
			d31 = snap67
			d32 = snap68
			d33 = snap69
			d34 = snap70
			d35 = snap71
			d36 = snap72
			d37 = snap73
			d38 = snap74
			d39 = snap75
			d40 = snap76
			d41 = snap77
			d42 = snap78
			d43 = snap79
			d44 = snap80
		}
		if !bbs[4].Rendered {
			return bbs[4].Render()
		}
		return result
		return result
	}
	bbs[4].Render = func() scm.JITValueDesc {
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		bbs[4].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl5)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d82 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r37 := ctx.AllocReg()
			r38 := ctx.AllocRegExcept(r37)
			r39 := ctx.AllocRegExcept(r37, r38)
			off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
			ctx.EmitMovRegMem(r37, thisptr.Reg, off)
			ctx.EmitMovRegMem(r38, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r39, thisptr.Reg, off+16)
			d82 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r37, Reg2: r38, Reg3: r39}
			ctx.BindReg(r37, &d82)
			ctx.BindReg(r38, &d82)
			ctx.BindReg(r39, &d82)
			ctx.BindReg(r37, &d82)
			ctx.BindReg(r38, &d82)
			ctx.BindReg(r39, &d82)
		}
		ctx.EnsureDesc(&d19)
		d84 = ctx.EmitSliceElementAddress(&d82, &d19, 16)
		ctx.EnsureDesc(&d84)
		r40 := ctx.AllocRegExcept(d84.Reg)
		ctx.EmitMovRegMem(r40, d84.Reg, 8)
		ctx.EmitMovRegMem(d84.Reg, d84.Reg, 0)
		d83 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d84.Reg, Reg2: r40}
		ctx.BindReg(d84.Reg, &d83)
		ctx.BindReg(r40, &d83)
		d85 = result
		ctx.EnsureDesc(&d83)
		if d83.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d83, &d85)
		} else {
			switch d83.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d85, d83)
			case scm.TagInt:
				ctx.EmitMakeInt(d85, d83)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d85, d83)
			case scm.TagNil:
				ctx.EmitMakeNil(d85)
			default:
				ctx.EmitMovPairToResult(&d83, &d85)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[5].Render = func() scm.JITValueDesc {
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
			return result
		}
		bbs[5].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl6)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDescsTogether(&d41, &idxInt)
		if d41.Loc == scm.LocImm && idxInt.Loc == scm.LocImm {
			d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d41.Imm.Int()) < uint64(idxInt.Imm.Int()))}
		} else if idxInt.Loc == scm.LocImm {
			r41 := ctx.AllocRegExcept(d41.Reg)
			if idxInt.Imm.Int() >= -2147483648 && idxInt.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d41.Reg, int32(idxInt.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(idxInt.Imm.Int()))
				ctx.EmitCmpInt64(d41.Reg, ctx.ScratchReg)
			}
			d87 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r41, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r41, &d87)
		} else if d41.Loc == scm.LocImm {
			r42 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d41.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, idxInt.Reg)
			d87 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r42, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r42, &d87)
		} else {
			r43 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitCmpInt64(d41.Reg, idxInt.Reg)
			d87 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r43, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r43, &d87)
		}
		ctx.FreeDesc(&idxInt)
		d88 = d87
		ctx.EnsureDesc(&d88)
		if d88.Loc != scm.LocImm && d88.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d88.Loc == scm.LocImm {
			if d88.Imm.Bool() {
				return bbs[6].Render()
			}
			return bbs[7].Render()
		}
		ctx.EmitJump(d88.Condition, lbl7)
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
		}
		ctx.FreeDesc(&d87)
		ctx.FlushRegisterMoves()
		if !bbs[7].Rendered {
			snap89 := d1
			snap90 := d2
			snap91 := d3
			snap92 := d4
			snap93 := d5
			snap94 := d6
			snap95 := d7
			snap96 := d16
			snap97 := d17
			snap98 := d18
			snap99 := d19
			snap100 := d20
			snap101 := d21
			snap102 := d22
			snap103 := d23
			snap104 := d24
			snap105 := d25
			snap106 := d26
			snap107 := d27
			snap108 := d28
			snap109 := d29
			snap110 := d30
			snap111 := d31
			snap112 := d32
			snap113 := d33
			snap114 := d34
			snap115 := d35
			snap116 := d36
			snap117 := d37
			snap118 := d38
			snap119 := d39
			snap120 := d40
			snap121 := d41
			snap122 := d42
			snap123 := d43
			snap124 := d44
			snap125 := d82
			snap126 := d83
			snap127 := d84
			snap128 := d85
			snap129 := d86
			snap130 := d87
			snap131 := d88
			alloc132 := ctx.SnapshotAllocState()
			bbs[7].Render()
			ctx.RestoreAllocState(alloc132)
			d1 = snap89
			d2 = snap90
			d3 = snap91
			d4 = snap92
			d5 = snap93
			d6 = snap94
			d7 = snap95
			d16 = snap96
			d17 = snap97
			d18 = snap98
			d19 = snap99
			d20 = snap100
			d21 = snap101
			d22 = snap102
			d23 = snap103
			d24 = snap104
			d25 = snap105
			d26 = snap106
			d27 = snap107
			d28 = snap108
			d29 = snap109
			d30 = snap110
			d31 = snap111
			d32 = snap112
			d33 = snap113
			d34 = snap114
			d35 = snap115
			d36 = snap116
			d37 = snap117
			d38 = snap118
			d39 = snap119
			d40 = snap120
			d41 = snap121
			d42 = snap122
			d43 = snap123
			d44 = snap124
			d82 = snap125
			d83 = snap126
			d84 = snap127
			d85 = snap128
			d86 = snap129
			d87 = snap130
			d88 = snap131
		}
		if !bbs[6].Rendered {
			return bbs[6].Render()
		}
		return result
		return result
	}
	bbs[6].Render = func() scm.JITValueDesc {
		if bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
			return result
		}
		bbs[6].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl7)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d19)
		ctx.EnsureDesc(&d19)
		if d19.Loc == scm.LocImm {
			d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegReg(scratch, d19.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 32, scratch, 1)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d133)
		}
		if d133.Loc == scm.LocReg && d19.Loc == scm.LocReg && d133.Reg == d19.Reg {
			ctx.TransferReg(d19.Reg)
			d19.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d133)
		ctx.EmitStoreToStack(d133, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d133)
		return bbs[1].Render()
		return result
	}
	bbs[7].Render = func() scm.JITValueDesc {
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		bbs[7].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl8)
		ctx.ResolveFixups()
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		ctx.ReclaimUntrackedRegs()
		ctx.SyncDesc(&d19)
		if d19.Loc == scm.LocReg || d19.Loc == scm.LocFPReg {
			ctx.ProtectReg(d19.Reg)
		} else if d19.Loc == scm.LocRegPair {
			ctx.ProtectReg(d19.Reg)
			ctx.ProtectReg(d19.Reg2)
		}
		d134 = d19
		if d134.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d134)
		d135 = d134
		if d135.Loc == scm.LocImm {
			d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: d135.Type, Imm: scm.NewInt(int64(uint64(d135.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d135.Reg, 32)
			ctx.EmitShrRegImm8(d135.Reg, 32)
		}
		ctx.EmitStoreToStack(d135, int32(bbs[1].PhiBase)+int32(16))
		if d19.Loc == scm.LocReg || d19.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d19.Reg)
		} else if d19.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d19.Reg)
			ctx.UnprotectReg(d19.Reg2)
		}
		return bbs[1].Render()
		return result
	}
	_ = bbs[0].Render()
	ctx.MarkLabel(lbl0)
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
//
//jitgen:control-flow-stable recid count target/1 stride
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

//jitgen:control-flow-stable recids/2 target/1 stride
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
