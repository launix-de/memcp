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
import "encoding/binary"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "unsafe"

type StorageSeq struct {
	storageJITFunctions
	// data
	recordId,
	start,
	stride StorageInt `jit:"immutable-after-finish"`
	count    uint   `jit:"immutable-after-finish"` // number of values
	seqCount uint32 `jit:"immutable-after-finish"` // number of sequences

	// analysis (lastValue also used as atomic pivot cache for concurrent GetValue)
	lastValue      atomic.Int64
	lastStride     int64
	lastValueNil   bool
	lastValueFirst bool
}

func (s *StorageSeq) ComputeSize() uint {
	return uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.recordId)-unsafe.Sizeof(s.start)-unsafe.Sizeof(s.stride)) + s.recordId.ComputeSize() + s.start.ComputeSize() + s.stride.ComputeSize()
}

func (s *StorageSeq) String() string {
	return fmt.Sprintf("seq[%dx %s/%s]", s.seqCount, s.start.String(), s.stride.String())
}

// storageSeqVersion is the current binary format version for StorageSeq.
// Increment this constant and add a new deserializeSeqV* helper whenever the
// layout after the magic byte changes.  Never delete old helpers.
const storageSeqVersion = 0

// StorageSeq binary layout (magic byte 11 consumed by shard loader):
//
//	[version uint8]    ← first byte read by Deserialize
//	[pad 7 bytes]      ← alignment padding
//	[count uint64]
//	[seqCount uint64]
//	[recordId StorageInt] (with its own magic byte)
//	[start StorageInt]    (with its own magic byte)
//	[stride StorageInt]   (with its own magic byte)
//
// Version history:
//
//	0 (current): layout as above; the version byte was previously the first byte
//	             of a 7-byte ASCII dummy "1234567" (byte value '1'=49).
//	             Legacy detection: if version byte == '1' (49), treat as v0 legacy.

func (s *StorageSeq) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
	var d86 scm.JITValueDesc
	_ = d86
	var d87 scm.JITValueDesc
	_ = d87
	var d88 scm.JITValueDesc
	_ = d88
	var d89 scm.JITValueDesc
	_ = d89
	var d90 scm.JITValueDesc
	_ = d90
	var d91 scm.JITValueDesc
	_ = d91
	var d92 scm.JITValueDesc
	_ = d92
	var d93 scm.JITValueDesc
	_ = d93
	var d94 scm.JITValueDesc
	_ = d94
	var d95 scm.JITValueDesc
	_ = d95
	var d96 scm.JITValueDesc
	_ = d96
	var d97 scm.JITValueDesc
	_ = d97
	var d98 scm.JITValueDesc
	_ = d98
	var d99 scm.JITValueDesc
	_ = d99
	var d100 scm.JITValueDesc
	_ = d100
	var d101 scm.JITValueDesc
	_ = d101
	var d102 scm.JITValueDesc
	_ = d102
	var d103 scm.JITValueDesc
	_ = d103
	var d104 scm.JITValueDesc
	_ = d104
	var d105 scm.JITValueDesc
	_ = d105
	var d106 scm.JITValueDesc
	_ = d106
	var d107 scm.JITValueDesc
	_ = d107
	var d171 scm.JITValueDesc
	_ = d171
	var d172 scm.JITValueDesc
	_ = d172
	var d173 scm.JITValueDesc
	_ = d173
	var d174 scm.JITValueDesc
	_ = d174
	var d175 scm.JITValueDesc
	_ = d175
	var d176 scm.JITValueDesc
	_ = d176
	var d177 scm.JITValueDesc
	_ = d177
	var d178 scm.JITValueDesc
	_ = d178
	var d250 scm.JITValueDesc
	_ = d250
	var d251 scm.JITValueDesc
	_ = d251
	var d325 scm.JITValueDesc
	_ = d325
	var d326 scm.JITValueDesc
	_ = d326
	var d327 scm.JITValueDesc
	_ = d327
	var d328 scm.JITValueDesc
	_ = d328
	var d329 scm.JITValueDesc
	_ = d329
	var d330 scm.JITValueDesc
	_ = d330
	var d331 scm.JITValueDesc
	_ = d331
	var d332 scm.JITValueDesc
	_ = d332
	var d333 scm.JITValueDesc
	_ = d333
	var d334 scm.JITValueDesc
	_ = d334
	var d335 scm.JITValueDesc
	_ = d335
	var d336 scm.JITValueDesc
	_ = d336
	var d337 scm.JITValueDesc
	_ = d337
	var d338 scm.JITValueDesc
	_ = d338
	var d339 scm.JITValueDesc
	_ = d339
	var d340 scm.JITValueDesc
	_ = d340
	var d341 scm.JITValueDesc
	_ = d341
	var d342 scm.JITValueDesc
	_ = d342
	var d343 scm.JITValueDesc
	_ = d343
	var d344 scm.JITValueDesc
	_ = d344
	var d345 scm.JITValueDesc
	_ = d345
	var d346 scm.JITValueDesc
	_ = d346
	var d347 scm.JITValueDesc
	_ = d347
	var d348 scm.JITValueDesc
	_ = d348
	var d349 scm.JITValueDesc
	_ = d349
	var d350 scm.JITValueDesc
	_ = d350
	var d351 scm.JITValueDesc
	_ = d351
	var d352 scm.JITValueDesc
	_ = d352
	var d353 scm.JITValueDesc
	_ = d353
	var d354 scm.JITValueDesc
	_ = d354
	var d458 scm.JITValueDesc
	_ = d458
	var d459 scm.JITValueDesc
	_ = d459
	var d460 scm.JITValueDesc
	_ = d460
	var d461 scm.JITValueDesc
	_ = d461
	var d462 scm.JITValueDesc
	_ = d462
	var d463 scm.JITValueDesc
	_ = d463
	var d464 scm.JITValueDesc
	_ = d464
	var d575 scm.JITValueDesc
	_ = d575
	var d576 scm.JITValueDesc
	_ = d576
	var d689 scm.JITValueDesc
	_ = d689
	var d690 scm.JITValueDesc
	_ = d690
	var d691 scm.JITValueDesc
	_ = d691
	var d692 scm.JITValueDesc
	_ = d692
	var d693 scm.JITValueDesc
	_ = d693
	var d694 scm.JITValueDesc
	_ = d694
	var d695 scm.JITValueDesc
	_ = d695
	var d696 scm.JITValueDesc
	_ = d696
	var d697 scm.JITValueDesc
	_ = d697
	var d698 scm.JITValueDesc
	_ = d698
	var d699 scm.JITValueDesc
	_ = d699
	var d700 scm.JITValueDesc
	_ = d700
	var d701 scm.JITValueDesc
	_ = d701
	var d702 scm.JITValueDesc
	_ = d702
	var d703 scm.JITValueDesc
	_ = d703
	var d704 scm.JITValueDesc
	_ = d704
	var d705 scm.JITValueDesc
	_ = d705
	var d706 scm.JITValueDesc
	_ = d706
	var d707 scm.JITValueDesc
	_ = d707
	var d708 scm.JITValueDesc
	_ = d708
	var d709 scm.JITValueDesc
	_ = d709
	var d710 scm.JITValueDesc
	_ = d710
	var d711 scm.JITValueDesc
	_ = d711
	var d712 scm.JITValueDesc
	_ = d712
	var d713 scm.JITValueDesc
	_ = d713
	var d714 scm.JITValueDesc
	_ = d714
	var d715 scm.JITValueDesc
	_ = d715
	var d716 scm.JITValueDesc
	_ = d716
	var d717 scm.JITValueDesc
	_ = d717
	var d718 scm.JITValueDesc
	_ = d718
	var d719 scm.JITValueDesc
	_ = d719
	var d720 scm.JITValueDesc
	_ = d720
	var d721 scm.JITValueDesc
	_ = d721
	var d722 scm.JITValueDesc
	_ = d722
	var d723 scm.JITValueDesc
	_ = d723
	var d724 scm.JITValueDesc
	_ = d724
	var d725 scm.JITValueDesc
	_ = d725
	var d726 scm.JITValueDesc
	_ = d726
	var d727 scm.JITValueDesc
	_ = d727
	var d728 scm.JITValueDesc
	_ = d728
	var d729 scm.JITValueDesc
	_ = d729
	var d730 scm.JITValueDesc
	_ = d730
	var d731 scm.JITValueDesc
	_ = d731
	var d732 scm.JITValueDesc
	_ = d732
	var d733 scm.JITValueDesc
	_ = d733
	var d734 scm.JITValueDesc
	_ = d734
	var d735 scm.JITValueDesc
	_ = d735
	var d736 scm.JITValueDesc
	_ = d736
	var d737 scm.JITValueDesc
	_ = d737
	var d738 scm.JITValueDesc
	_ = d738
	var d739 scm.JITValueDesc
	_ = d739
	var d740 scm.JITValueDesc
	_ = d740
	var d741 scm.JITValueDesc
	_ = d741
	var d742 scm.JITValueDesc
	_ = d742
	var d743 scm.JITValueDesc
	_ = d743
	var d744 scm.JITValueDesc
	_ = d744
	var d745 scm.JITValueDesc
	_ = d745
	var d746 scm.JITValueDesc
	_ = d746
	var d747 scm.JITValueDesc
	_ = d747
	var d748 scm.JITValueDesc
	_ = d748
	var d749 scm.JITValueDesc
	_ = d749
	var d750 scm.JITValueDesc
	_ = d750
	var d751 scm.JITValueDesc
	_ = d751
	var d752 scm.JITValueDesc
	_ = d752
	var d754 scm.JITValueDesc
	_ = d754
	var d755 scm.JITValueDesc
	_ = d755
	var d756 scm.JITValueDesc
	_ = d756
	var d757 scm.JITValueDesc
	_ = d757
	var d758 scm.JITValueDesc
	_ = d758
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
	phiBase0 := ctx.AllocStack(int32(144))
	var bbs [14]scm.BBDescriptor
	bbs[1].PhiBase = int32(phiBase0) + int32(0)
	bbs[2].PhiBase = int32(phiBase0) + int32(48)
	bbs[4].PhiBase = int32(phiBase0) + int32(64)
	bbs[8].PhiBase = int32(phiBase0) + int32(112)
	registerHomes1 := ctx.AllocRegisterHomes(scm.JITRegisterPlan{Slots: [16]scm.JITRegisterSlot{{Color: 0, Width: 1, Cost: 32}, {Color: 1, Width: 1, Cost: 12}, {Color: 2, Width: 1, Cost: 12}}, Count: 3})
	defer ctx.ReleaseRegisterHomes(registerHomes1)
	var r0 scm.Reg
	phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
	if phiHomeOK2 {
		r0 = registerHomes1.Registers[0]
	}
	var r1 scm.Reg
	phiHomeOK3 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
	if phiHomeOK3 {
		r1 = registerHomes1.Registers[1]
	}
	var r2 scm.Reg
	phiHomeOK4 := registerHomes1.Available&(uint16(1)<<2) == uint16(1)<<2
	if phiHomeOK4 {
		r2 = registerHomes1.Registers[2]
	}
	var d5 scm.JITValueDesc
	if phiHomeOK2 {
		d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
	} else {
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
	}
	_ = d5
	var d6 scm.JITValueDesc
	if phiHomeOK3 {
		d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
	} else {
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
	}
	_ = d6
	var d7 scm.JITValueDesc
	if phiHomeOK4 {
		d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
	} else {
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
	}
	_ = d7
	d8 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
	_ = d8
	d9 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
	_ = d9
	d10 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
	_ = d10
	d11 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
	_ = d11
	d12 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
	_ = d12
	d13 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
	_ = d13
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
	bbpos_0_8 := int32(-1)
	_ = bbpos_0_8
	lbl9 := ctx.ReserveLabel()
	_ = lbl9
	bbpos_0_9 := int32(-1)
	_ = bbpos_0_9
	lbl10 := ctx.ReserveLabel()
	_ = lbl10
	bbpos_0_10 := int32(-1)
	_ = bbpos_0_10
	lbl11 := ctx.ReserveLabel()
	_ = lbl11
	bbpos_0_11 := int32(-1)
	_ = bbpos_0_11
	lbl12 := ctx.ReserveLabel()
	_ = lbl12
	bbpos_0_12 := int32(-1)
	_ = bbpos_0_12
	lbl13 := ctx.ReserveLabel()
	_ = lbl13
	bbpos_0_13 := int32(-1)
	_ = bbpos_0_13
	lbl14 := ctx.ReserveLabel()
	_ = lbl14
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		r3 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(fieldAddr))
			ctx.EmitAtomicLoad64(r3, ctx.ScratchReg, 0)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			ctx.EmitAtomicLoad64(r3, thisptr.Reg, off)
		}
		d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
		ctx.BindReg(r3, &d14)
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d14)
		if d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d14.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d14.Reg)
			ctx.EmitShlRegImm8(r4, 32)
			ctx.EmitShrRegImm8(r4, 32)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d15)
		}
		ctx.StabilizeDescForControlFlow(&d15)
		ctx.FreeDesc(&d14)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
			val := *(*uint32)(unsafe.Pointer(fieldAddr))
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
			r5 := ctx.AllocReg()
			ctx.EmitMovRegMemL(r5, thisptr.Reg, off)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
			ctx.BindReg(r5, &d16)
		}
		ctx.EnsureDesc(&d16)
		ctx.EnsureDesc(&d16)
		if d16.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - 1)}
		} else {
			var scratch scm.Reg
			if phiHomeOK4 && r2 != d16.Reg {
				scratch = r2
			} else {
				scratch = ctx.AllocRegExcept(d16.Reg)
			}
			ctx.EmitMovRegReg(scratch, d16.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 32, scratch, 1)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		}
		if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
			ctx.TransferReg(d16.Reg)
			d16.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d16)
		ctx.SyncDesc(&d15)
		if d15.Loc == scm.LocReg || d15.Loc == scm.LocFPReg {
			ctx.ProtectReg(d15.Reg)
		} else if d15.Loc == scm.LocRegPair {
			ctx.ProtectReg(d15.Reg)
			ctx.ProtectReg(d15.Reg2)
		}
		ctx.SyncDesc(&d17)
		if d17.Loc == scm.LocReg || d17.Loc == scm.LocFPReg {
			ctx.ProtectReg(d17.Reg)
		} else if d17.Loc == scm.LocRegPair {
			ctx.ProtectReg(d17.Reg)
			ctx.ProtectReg(d17.Reg2)
		}
		d18 = d15
		if d18.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d18)
		if phiHomeOK2 {
			ctx.EmitMovToReg(r0, d18)
		} else {
			ctx.EmitStoreToStack(d18, int32(bbs[1].PhiBase)+int32(0))
		}
		if phiHomeOK3 {
			ctx.EmitMovToReg(r1, scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)})
		} else {
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
		}
		d19 = d17
		if d19.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d19)
		if phiHomeOK4 {
			ctx.EmitMovToReg(r2, d19)
		} else {
			ctx.EmitStoreToStack(d19, int32(bbs[1].PhiBase)+int32(32))
		}
		if d15.Loc == scm.LocReg || d15.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d15.Reg)
		} else if d15.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d15.Reg)
			ctx.UnprotectReg(d15.Reg2)
		}
		if d17.Loc == scm.LocReg || d17.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d17.Reg)
		} else if d17.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d17.Reg)
			ctx.UnprotectReg(d17.Reg2)
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if phiHomeOK2 && d5.Loc == scm.LocReg {
			ctx.BindReg(r0, &d5)
		}
		if phiHomeOK3 && d6.Loc == scm.LocReg {
			ctx.BindReg(r1, &d6)
		}
		if phiHomeOK4 && d7.Loc == scm.LocReg {
			ctx.BindReg(r2, &d7)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d20 = d5
		_ = d20
		ctx.StabilizeDescForControlFlow(&d5)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl15 := ctx.ReserveLabel()
		_ = lbl15
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl15)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r6 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r6, thisptr.Reg, off)
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
			ctx.BindReg(r6, &d21)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d21)
		if d21.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d21.Imm.Int()))))}
		} else {
			r7 := ctx.AllocReg()
			ctx.EmitMovRegReg(r7, d21.Reg)
			ctx.EmitShlRegImm8(r7, 56)
			ctx.EmitShrRegImm8(r7, 56)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d22)
		}
		ctx.FreeDesc(&d21)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d20)
		ctx.EnsureDesc(&d20)
		if d20.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d20.Imm.Int()))))}
		} else {
			r8 := ctx.AllocReg()
			ctx.EmitMovRegReg(r8, d20.Reg)
			ctx.EmitShlRegImm8(r8, 32)
			ctx.EmitShrRegImm8(r8, 32)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d23)
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
			r9 := ctx.AllocRegExcept(d23.Reg, d22.Reg)
			ctx.EmitMovRegReg(r9, d23.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r9, &d22)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d24)
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
			r10 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r10, d24.Reg)
			ctx.EmitShrRegImm8(r10, 6)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d25)
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
			r11 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r11, d24.Reg)
			ctx.EmitAndRegImm32(r11, 63)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d26)
		}
		if d26.Loc == scm.LocReg && d24.Loc == scm.LocReg && d26.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d27 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r12 := ctx.AllocReg()
			r13 := ctx.AllocRegExcept(r12)
			r14 := ctx.AllocRegExcept(r12, r13)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r12, thisptr.Reg, off)
			ctx.EmitMovRegMem(r13, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off+16)
			d27 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r12, Reg2: r13, Reg3: r14}
			ctx.BindReg(r12, &d27)
			ctx.BindReg(r13, &d27)
			ctx.BindReg(r14, &d27)
			ctx.BindReg(r12, &d27)
			ctx.BindReg(r13, &d27)
			ctx.BindReg(r14, &d27)
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
			r15 := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(r15, d28.Reg)
			ctx.EmitShlRegImm8(r15, uint8(d26.Imm.Int()))
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d29)
		} else {
			shiftSrc := d28.Reg
			r16 := ctx.AllocRegExcept(d28.Reg, d26.Reg)
			ctx.EmitMovRegReg(r16, d28.Reg)
			shiftSrc = r16
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
			r17 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r17, d32.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d33)
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
			r18 := ctx.AllocRegExcept(d32.Reg, d26.Reg)
			ctx.EmitMovRegReg(r18, d32.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r18, &d26)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d33)
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
			r19 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegReg(r19, d31.Reg)
			ctx.EmitShrRegImm8(r19, uint8(d33.Imm.Int()))
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d34)
		} else {
			shiftSrc := d31.Reg
			r20 := ctx.AllocRegExcept(d31.Reg, d33.Reg)
			ctx.EmitMovRegReg(r20, d31.Reg)
			shiftSrc = r20
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
			r21 := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(r21, d29.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d35)
		} else if d29.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
			ctx.EmitOrInt64(scratch, d34.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d35)
		} else if d34.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(r22, d29.Reg)
			if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r22, int32(d34.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d34.Imm.Int()))
				ctx.EmitOrInt64(r22, ctx.ScratchReg)
			}
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d35)
		} else {
			r23 := ctx.AllocRegExcept(d29.Reg, d34.Reg)
			ctx.EmitMovRegReg(r23, d29.Reg)
			ctx.EmitOrInt64(r23, d34.Reg)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d35)
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
			r24 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r24, d36.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d37)
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
			r25 := ctx.AllocRegExcept(d36.Reg, d22.Reg)
			ctx.EmitMovRegReg(r25, d36.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r25, &d22)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d37)
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
			r26 := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegReg(r26, d35.Reg)
			ctx.EmitShrRegImm8(r26, uint8(d37.Imm.Int()))
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d38)
		} else {
			shiftSrc := d35.Reg
			r27 := ctx.AllocRegExcept(d35.Reg, d37.Reg)
			ctx.EmitMovRegReg(r27, d35.Reg)
			shiftSrc = r27
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
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d38)
		if d38.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d38.Imm.Int()))))}
		} else {
			r28 := ctx.AllocReg()
			ctx.EmitMovRegReg(r28, d38.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d39)
		}
		ctx.FreeDesc(&d38)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r29 := ctx.AllocReg()
			ctx.EmitMovRegMem(r29, thisptr.Reg, off)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d40)
		}
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d40)
		ctx.EnsureDescsTogether(&d39, &d40)
		if d39.Loc == scm.LocImm && d40.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() + d40.Imm.Int())}
		} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
			ctx.EnsureDesc(&d39)
			r30 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r30, d39.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d41)
		} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
			ctx.EnsureDesc(&d40)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
			ctx.BindReg(d40.Reg, &d41)
		} else if d39.Loc == scm.LocImm {
			ctx.EnsureDesc(&d40)
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(scratch, d40.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d39.Imm.Int())
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else if d40.Loc == scm.LocImm {
			ctx.EnsureDesc(&d39)
			scratch := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(scratch, d39.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d40.Imm.Int())
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else {
			ctx.EnsureDesc(&d39)
			ctx.SyncDesc(&d40)
			r31 := ctx.AllocRegExcept(d39.Reg, d40.Reg)
			ctx.EmitMovRegReg(r31, d39.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r31, &d40)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d41)
		}
		if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.FreeDesc(&d40)
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d41)
		if d41.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d41.Imm.Int()))))}
		} else {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegReg(r32, d41.Reg)
			ctx.EmitShlRegImm8(r32, 32)
			ctx.EmitShrRegImm8(r32, 32)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d42)
		}
		ctx.FreeDesc(&d41)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d42)
		ctx.EnsureDescsTogether(&idxInt, &d42)
		if idxInt.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d42.Imm.Int()))}
		} else if d42.Loc == scm.LocImm {
			r33 := ctx.AllocRegExcept(idxInt.Reg)
			if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d42.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d42.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, ctx.ScratchReg)
			}
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r33, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r33, &d43)
		} else if idxInt.Loc == scm.LocImm {
			r34 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r34, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r34, &d43)
		} else {
			r35 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r35, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r35, &d43)
		}
		ctx.FreeDesc(&d42)
		d44 = d43
		ctx.EnsureDesc(&d44)
		if d44.Loc != scm.LocImm && d44.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d44.Loc == scm.LocImm {
			if d44.Imm.Bool() {
				return bbs[3].Render()
			}
			return bbs[5].Render()
		}
		ctx.EmitJump(d44.Condition, lbl4)
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
		}
		ctx.FreeDesc(&d43)
		ctx.FlushRegisterMoves()
		if !bbs[5].Rendered {
			snap45 := d5
			snap46 := d6
			snap47 := d7
			snap48 := d8
			snap49 := d9
			snap50 := d10
			snap51 := d11
			snap52 := d12
			snap53 := d13
			snap54 := d14
			snap55 := d15
			snap56 := d16
			snap57 := d17
			snap58 := d18
			snap59 := d19
			snap60 := d20
			snap61 := d21
			snap62 := d22
			snap63 := d23
			snap64 := d24
			snap65 := d25
			snap66 := d26
			snap67 := d27
			snap68 := d28
			snap69 := d29
			snap70 := d30
			snap71 := d31
			snap72 := d32
			snap73 := d33
			snap74 := d34
			snap75 := d35
			snap76 := d36
			snap77 := d37
			snap78 := d38
			snap79 := d39
			snap80 := d40
			snap81 := d41
			snap82 := d42
			snap83 := d43
			snap84 := d44
			alloc85 := ctx.SnapshotAllocState()
			bbs[5].Render()
			ctx.RestoreAllocState(alloc85)
			d5 = snap45
			d6 = snap46
			d7 = snap47
			d8 = snap48
			d9 = snap49
			d10 = snap50
			d11 = snap51
			d12 = snap52
			d13 = snap53
			d14 = snap54
			d15 = snap55
			d16 = snap56
			d17 = snap57
			d18 = snap58
			d19 = snap59
			d20 = snap60
			d21 = snap61
			d22 = snap62
			d23 = snap63
			d24 = snap64
			d25 = snap65
			d26 = snap66
			d27 = snap67
			d28 = snap68
			d29 = snap69
			d30 = snap70
			d31 = snap71
			d32 = snap72
			d33 = snap73
			d34 = snap74
			d35 = snap75
			d36 = snap76
			d37 = snap77
			d38 = snap78
			d39 = snap79
			d40 = snap80
			d41 = snap81
			d42 = snap82
			d43 = snap83
			d44 = snap84
		}
		if !bbs[3].Rendered {
			return bbs[3].Render()
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d8)
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d8)
		if d8.Loc == scm.LocImm {
			d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d8.Imm.Int()))))}
		} else {
			r36 := ctx.AllocReg()
			ctx.EmitMovRegReg(r36, d8.Reg)
			ctx.EmitShlRegImm8(r36, 32)
			ctx.EmitShrRegImm8(r36, 32)
			d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d86)
		}
		ctx.EnsureDesc(&d86)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d86.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d86.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d86.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d86.Imm.Int()))
				ctx.EmitAtomicStore64(ctx.ScratchReg, baseReg, 0)
			} else {
				ctx.EmitAtomicStore64(d86.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d86.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d86.Imm.Int()))
				ctx.EmitAtomicStore64(ctx.ScratchReg, thisptr.Reg, off)
			} else {
				ctx.EmitAtomicStore64(d86.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d86)
		ctx.EnsureDesc(&d8)
		d87 = d8
		_ = d87
		ctx.StabilizeDescForControlFlow(&d8)
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl16 := ctx.ReserveLabel()
		_ = lbl16
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl16)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r37, thisptr.Reg, off)
			d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d88)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d88)
		ctx.EnsureDesc(&d88)
		if d88.Loc == scm.LocImm {
			d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d88.Imm.Int()))))}
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegReg(r38, d88.Reg)
			ctx.EmitShlRegImm8(r38, 56)
			ctx.EmitShrRegImm8(r38, 56)
			d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d89)
		}
		ctx.FreeDesc(&d88)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d87)
		ctx.EnsureDesc(&d87)
		if d87.Loc == scm.LocImm {
			d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d87.Imm.Int()))))}
		} else {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegReg(r39, d87.Reg)
			ctx.EmitShlRegImm8(r39, 32)
			ctx.EmitShrRegImm8(r39, 32)
			d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d90)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d90)
		ctx.EnsureDesc(&d89)
		ctx.EnsureDescsTogether(&d90, &d89)
		if d90.Loc == scm.LocImm && d89.Loc == scm.LocImm {
			d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() * d89.Imm.Int())}
		} else if d90.Loc == scm.LocImm {
			ctx.EnsureDesc(&d89)
			scratch := ctx.AllocRegExcept(d89.Reg)
			ctx.EmitMovRegReg(scratch, d89.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d90.Imm.Int())
			d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d91)
		} else if d89.Loc == scm.LocImm {
			ctx.EnsureDesc(&d90)
			scratch := ctx.AllocRegExcept(d90.Reg)
			ctx.EmitMovRegReg(scratch, d90.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d89.Imm.Int())
			d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d91)
		} else {
			ctx.EnsureDesc(&d90)
			ctx.SyncDesc(&d89)
			r40 := ctx.AllocRegExcept(d90.Reg, d89.Reg)
			ctx.EmitMovRegReg(r40, d90.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r40, &d89)
			d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d91)
		}
		if d91.Loc == scm.LocReg && d90.Loc == scm.LocReg && d91.Reg == d90.Reg {
			ctx.TransferReg(d90.Reg)
			d90.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d90)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d91)
		if d91.Loc == scm.LocImm {
			d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() / 64)}
		} else {
			r41 := ctx.AllocRegExcept(d91.Reg)
			ctx.EmitMovRegReg(r41, d91.Reg)
			ctx.EmitShrRegImm8(r41, 6)
			d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d92)
		}
		if d92.Loc == scm.LocReg && d91.Loc == scm.LocReg && d92.Reg == d91.Reg {
			ctx.TransferReg(d91.Reg)
			d91.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d91)
		if d91.Loc == scm.LocImm {
			d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() % 64)}
		} else {
			r42 := ctx.AllocRegExcept(d91.Reg)
			ctx.EmitMovRegReg(r42, d91.Reg)
			ctx.EmitAndRegImm32(r42, 63)
			d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d93)
		}
		if d93.Loc == scm.LocReg && d91.Loc == scm.LocReg && d93.Reg == d91.Reg {
			ctx.TransferReg(d91.Reg)
			d91.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d91)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d94 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r43 := ctx.AllocReg()
			r44 := ctx.AllocRegExcept(r43)
			r45 := ctx.AllocRegExcept(r43, r44)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			ctx.EmitMovRegMem(r43, thisptr.Reg, off)
			ctx.EmitMovRegMem(r44, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r45, thisptr.Reg, off+16)
			d94 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r43, Reg2: r44, Reg3: r45}
			ctx.BindReg(r43, &d94)
			ctx.BindReg(r44, &d94)
			ctx.BindReg(r45, &d94)
			ctx.BindReg(r43, &d94)
			ctx.BindReg(r44, &d94)
			ctx.BindReg(r45, &d94)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d92)
		ctx.ReclaimUntrackedRegs()
		d95 = ctx.EmitLoadScalarSliceElement(&d94, &d92, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d95)
		ctx.EnsureDesc(&d93)
		ctx.EnsureDescsTogether(&d95, &d93)
		if d95.Loc == scm.LocImm && d93.Loc == scm.LocImm {
			d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d95.Imm.Int()) << uint64(d93.Imm.Int())))}
		} else if d93.Loc == scm.LocImm {
			r46 := ctx.AllocRegExcept(d95.Reg)
			ctx.EmitMovRegReg(r46, d95.Reg)
			ctx.EmitShlRegImm8(r46, uint8(d93.Imm.Int()))
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d96)
		} else {
			shiftSrc := d95.Reg
			r47 := ctx.AllocRegExcept(d95.Reg, d93.Reg)
			ctx.EmitMovRegReg(r47, d95.Reg)
			shiftSrc = r47
			ctx.EmitShiftLeft(shiftSrc, d93.Reg, true)
			d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d96)
		}
		if d96.Loc == scm.LocReg && d95.Loc == scm.LocReg && d96.Reg == d95.Reg {
			ctx.TransferReg(d95.Reg)
			d95.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d95)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d92)
		ctx.EnsureDesc(&d92)
		if d92.Loc == scm.LocImm {
			d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d92.Reg)
			ctx.EmitMovRegReg(scratch, d92.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d97)
		}
		if d97.Loc == scm.LocReg && d92.Loc == scm.LocReg && d97.Reg == d92.Reg {
			ctx.TransferReg(d92.Reg)
			d92.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d92)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d97)
		ctx.ReclaimUntrackedRegs()
		d98 = ctx.EmitLoadScalarSliceElement(&d94, &d97, 8, scm.TagInt)
		ctx.FreeDesc(&d97)
		ctx.ReclaimUntrackedRegs()
		d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d93)
		ctx.EnsureDescsTogether(&d99, &d93)
		if d99.Loc == scm.LocImm && d93.Loc == scm.LocImm {
			d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() - d93.Imm.Int())}
		} else if d93.Loc == scm.LocImm && d93.Imm.Int() == 0 {
			ctx.EnsureDesc(&d99)
			r48 := ctx.AllocRegExcept(d99.Reg)
			ctx.EmitMovRegReg(r48, d99.Reg)
			d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d100)
		} else if d99.Loc == scm.LocImm {
			ctx.EnsureDesc(&d93)
			scratch := ctx.AllocRegExcept(d93.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d93)
			d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d100)
		} else if d93.Loc == scm.LocImm {
			ctx.EnsureDesc(&d99)
			scratch := ctx.AllocRegExcept(d99.Reg)
			ctx.EmitMovRegReg(scratch, d99.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d93.Imm.Int())
			d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d100)
		} else {
			ctx.EnsureDesc(&d99)
			ctx.SyncDesc(&d93)
			r49 := ctx.AllocRegExcept(d99.Reg, d93.Reg)
			ctx.EmitMovRegReg(r49, d99.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r49, &d93)
			d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d100)
		}
		if d100.Loc == scm.LocReg && d99.Loc == scm.LocReg && d100.Reg == d99.Reg {
			ctx.TransferReg(d99.Reg)
			d99.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d93)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d98)
		ctx.EnsureDesc(&d100)
		ctx.EnsureDescsTogether(&d98, &d100)
		if d98.Loc == scm.LocImm && d100.Loc == scm.LocImm {
			d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d98.Imm.Int()) >> uint64(d100.Imm.Int())))}
		} else if d100.Loc == scm.LocImm {
			r50 := ctx.AllocRegExcept(d98.Reg)
			ctx.EmitMovRegReg(r50, d98.Reg)
			ctx.EmitShrRegImm8(r50, uint8(d100.Imm.Int()))
			d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d101)
		} else {
			shiftSrc := d98.Reg
			r51 := ctx.AllocRegExcept(d98.Reg, d100.Reg)
			ctx.EmitMovRegReg(r51, d98.Reg)
			shiftSrc = r51
			ctx.EmitShiftRight(shiftSrc, d100.Reg, false)
			d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d101)
		}
		if d101.Loc == scm.LocReg && d98.Loc == scm.LocReg && d101.Reg == d98.Reg {
			ctx.TransferReg(d98.Reg)
			d98.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d98)
		ctx.FreeDesc(&d100)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d96)
		ctx.EnsureDesc(&d101)
		if d96.Loc == scm.LocImm && d101.Loc == scm.LocImm {
			d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() | d101.Imm.Int())}
		} else if d96.Loc == scm.LocImm && d96.Imm.Int() == 0 {
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			ctx.BindReg(d101.Reg, &d102)
		} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
			r52 := ctx.AllocRegExcept(d96.Reg)
			ctx.EmitMovRegReg(r52, d96.Reg)
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
			ctx.BindReg(r52, &d102)
		} else if d96.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d101.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d96.Imm.Int()))
			ctx.EmitOrInt64(scratch, d101.Reg)
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d102)
		} else if d101.Loc == scm.LocImm {
			r53 := ctx.AllocRegExcept(d96.Reg)
			ctx.EmitMovRegReg(r53, d96.Reg)
			if d101.Imm.Int() >= -2147483648 && d101.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r53, int32(d101.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d101.Imm.Int()))
				ctx.EmitOrInt64(r53, ctx.ScratchReg)
			}
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d102)
		} else {
			r54 := ctx.AllocRegExcept(d96.Reg, d101.Reg)
			ctx.EmitMovRegReg(r54, d96.Reg)
			ctx.EmitOrInt64(r54, d101.Reg)
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d102)
		}
		if d102.Loc == scm.LocReg && d96.Loc == scm.LocReg && d102.Reg == d96.Reg {
			ctx.TransferReg(d96.Reg)
			d96.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d96)
		ctx.FreeDesc(&d101)
		ctx.ReclaimUntrackedRegs()
		d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d89)
		ctx.EnsureDescsTogether(&d103, &d89)
		if d103.Loc == scm.LocImm && d89.Loc == scm.LocImm {
			d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() - d89.Imm.Int())}
		} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
			ctx.EnsureDesc(&d103)
			r55 := ctx.AllocRegExcept(d103.Reg)
			ctx.EmitMovRegReg(r55, d103.Reg)
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d104)
		} else if d103.Loc == scm.LocImm {
			ctx.EnsureDesc(&d89)
			scratch := ctx.AllocRegExcept(d89.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d89)
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d104)
		} else if d89.Loc == scm.LocImm {
			ctx.EnsureDesc(&d103)
			scratch := ctx.AllocRegExcept(d103.Reg)
			ctx.EmitMovRegReg(scratch, d103.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d89.Imm.Int())
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d104)
		} else {
			ctx.EnsureDesc(&d103)
			ctx.SyncDesc(&d89)
			r56 := ctx.AllocRegExcept(d103.Reg, d89.Reg)
			ctx.EmitMovRegReg(r56, d103.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r56, &d89)
			d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d104)
		}
		if d104.Loc == scm.LocReg && d103.Loc == scm.LocReg && d104.Reg == d103.Reg {
			ctx.TransferReg(d103.Reg)
			d103.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d89)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d102)
		ctx.EnsureDesc(&d104)
		ctx.EnsureDescsTogether(&d102, &d104)
		if d102.Loc == scm.LocImm && d104.Loc == scm.LocImm {
			d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d102.Imm.Int()) >> uint64(d104.Imm.Int())))}
		} else if d104.Loc == scm.LocImm {
			r57 := ctx.AllocRegExcept(d102.Reg)
			ctx.EmitMovRegReg(r57, d102.Reg)
			ctx.EmitShrRegImm8(r57, uint8(d104.Imm.Int()))
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d105)
		} else {
			shiftSrc := d102.Reg
			r58 := ctx.AllocRegExcept(d102.Reg, d104.Reg)
			ctx.EmitMovRegReg(r58, d102.Reg)
			shiftSrc = r58
			ctx.EmitShiftRight(shiftSrc, d104.Reg, false)
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d105)
		}
		if d105.Loc == scm.LocReg && d102.Loc == scm.LocReg && d105.Reg == d102.Reg {
			ctx.TransferReg(d102.Reg)
			d102.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d102)
		ctx.FreeDesc(&d104)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d105)
		ctx.StabilizeDescForControlFlow(&d105)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 80)
			r59 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r59, thisptr.Reg, off)
			d106 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d106)
		}
		d107 = d106
		ctx.EnsureDesc(&d107)
		if d107.Loc != scm.LocImm && d107.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d107.Loc == scm.LocImm {
			if d107.Imm.Bool() {
				return bbs[13].Render()
			}
			return bbs[12].Render()
		}
		ctx.EmitCmpRegImm32(d107.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FlushRegisterMoves()
		if !bbs[12].Rendered {
			snap108 := d5
			snap109 := d6
			snap110 := d7
			snap111 := d8
			snap112 := d9
			snap113 := d10
			snap114 := d11
			snap115 := d12
			snap116 := d13
			snap117 := d14
			snap118 := d15
			snap119 := d16
			snap120 := d17
			snap121 := d18
			snap122 := d19
			snap123 := d20
			snap124 := d21
			snap125 := d22
			snap126 := d23
			snap127 := d24
			snap128 := d25
			snap129 := d26
			snap130 := d27
			snap131 := d28
			snap132 := d29
			snap133 := d30
			snap134 := d31
			snap135 := d32
			snap136 := d33
			snap137 := d34
			snap138 := d35
			snap139 := d36
			snap140 := d37
			snap141 := d38
			snap142 := d39
			snap143 := d40
			snap144 := d41
			snap145 := d42
			snap146 := d43
			snap147 := d44
			snap148 := d86
			snap149 := d87
			snap150 := d88
			snap151 := d89
			snap152 := d90
			snap153 := d91
			snap154 := d92
			snap155 := d93
			snap156 := d94
			snap157 := d95
			snap158 := d96
			snap159 := d97
			snap160 := d98
			snap161 := d99
			snap162 := d100
			snap163 := d101
			snap164 := d102
			snap165 := d103
			snap166 := d104
			snap167 := d105
			snap168 := d106
			snap169 := d107
			alloc170 := ctx.SnapshotAllocState()
			bbs[12].Render()
			ctx.RestoreAllocState(alloc170)
			d5 = snap108
			d6 = snap109
			d7 = snap110
			d8 = snap111
			d9 = snap112
			d10 = snap113
			d11 = snap114
			d12 = snap115
			d13 = snap116
			d14 = snap117
			d15 = snap118
			d16 = snap119
			d17 = snap120
			d18 = snap121
			d19 = snap122
			d20 = snap123
			d21 = snap124
			d22 = snap125
			d23 = snap126
			d24 = snap127
			d25 = snap128
			d26 = snap129
			d27 = snap130
			d28 = snap131
			d29 = snap132
			d30 = snap133
			d31 = snap134
			d32 = snap135
			d33 = snap136
			d34 = snap137
			d35 = snap138
			d36 = snap139
			d37 = snap140
			d38 = snap141
			d39 = snap142
			d40 = snap143
			d41 = snap144
			d42 = snap145
			d43 = snap146
			d44 = snap147
			d86 = snap148
			d87 = snap149
			d88 = snap150
			d89 = snap151
			d90 = snap152
			d91 = snap153
			d92 = snap154
			d93 = snap155
			d94 = snap156
			d95 = snap157
			d96 = snap158
			d97 = snap159
			d98 = snap160
			d99 = snap161
			d100 = snap162
			d101 = snap163
			d102 = snap164
			d103 = snap165
			d104 = snap166
			d105 = snap167
			d106 = snap168
			d107 = snap169
		}
		if !bbs[13].Rendered {
			return bbs[13].Render()
		}
		return result
		ctx.FreeDesc(&d106)
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		if d5.Loc == scm.LocImm {
			d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 32, scratch, 1)
			d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d171)
		}
		if d171.Loc == scm.LocReg && d5.Loc == scm.LocReg && d171.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d171)
		ctx.EmitStoreToStack(d171, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d171)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		if d5.Loc == scm.LocImm {
			d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 32, scratch, 1)
			d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d172)
		}
		if d172.Loc == scm.LocReg && d5.Loc == scm.LocReg && d172.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d172)
		ctx.EmitStoreToStack(d172, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d172)
		ctx.SyncDesc(&d6)
		if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
			ctx.ProtectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.ProtectReg(d6.Reg)
			ctx.ProtectReg(d6.Reg2)
		}
		d173 = d6
		if d173.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d173)
		d174 = d173
		if d174.Loc == scm.LocImm {
			d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: d174.Type, Imm: scm.NewInt(int64(uint64(d174.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d174.Reg, 32)
			ctx.EmitShrRegImm8(d174.Reg, 32)
		}
		ctx.EmitStoreToStack(d174, int32(bbs[4].PhiBase)+int32(16))
		if d6.Loc == scm.LocReg || d6.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		return bbs[4].Render()
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d9)
		ctx.StabilizeDescForControlFlow(&d10)
		ctx.StabilizeDescForControlFlow(&d11)
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d11)
		ctx.EnsureDescsTogether(&d10, &d11)
		if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d10.Imm.Int()) == uint64(d11.Imm.Int()))}
		} else if d11.Loc == scm.LocImm {
			r60 := ctx.AllocRegExcept(d10.Reg)
			if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d10.Reg, int32(d11.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d11.Imm.Int()))
				ctx.EmitCmpInt64(d10.Reg, ctx.ScratchReg)
			}
			d175 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r60, Condition: scm.CondEqual}
			ctx.BindReg(r60, &d175)
		} else if d10.Loc == scm.LocImm {
			r61 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d10.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d11.Reg)
			d175 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r61, Condition: scm.CondEqual}
			ctx.BindReg(r61, &d175)
		} else {
			r62 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpInt64(d10.Reg, d11.Reg)
			d175 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r62, Condition: scm.CondEqual}
			ctx.BindReg(r62, &d175)
		}
		d176 = d175
		ctx.EnsureDesc(&d176)
		if d176.Loc != scm.LocImm && d176.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d176.Loc == scm.LocImm {
			if d176.Imm.Bool() {
				ctx.SyncDesc(&d10)
				if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
					ctx.ProtectReg(d10.Reg)
				} else if d10.Loc == scm.LocRegPair {
					ctx.ProtectReg(d10.Reg)
					ctx.ProtectReg(d10.Reg2)
				}
				d177 = d10
				if d177.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d177)
				d178 = d177
				if d178.Loc == scm.LocImm {
					d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: d178.Type, Imm: scm.NewInt(int64(uint64(d178.Imm.Int()) & 0xffffffff))}
				} else {
					ctx.EmitShlRegImm8(d178.Reg, 32)
					ctx.EmitShrRegImm8(d178.Reg, 32)
				}
				ctx.EmitStoreToStack(d178, int32(bbs[2].PhiBase)+int32(0))
				if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
					ctx.UnprotectReg(d10.Reg)
				} else if d10.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d10.Reg)
					ctx.UnprotectReg(d10.Reg2)
				}
				return bbs[2].Render()
			}
			return bbs[6].Render()
		}
		lbl17 := ctx.ReserveLabel()
		ctx.EmitJump(d176.Condition, lbl17)
		ctx.EmitJmp(lbl7)
		ctx.FreeDesc(&d175)
		snap179 := d5
		snap180 := d6
		snap181 := d7
		snap182 := d8
		snap183 := d9
		snap184 := d10
		snap185 := d11
		snap186 := d12
		snap187 := d13
		snap188 := d14
		snap189 := d15
		snap190 := d16
		snap191 := d17
		snap192 := d18
		snap193 := d19
		snap194 := d20
		snap195 := d21
		snap196 := d22
		snap197 := d23
		snap198 := d24
		snap199 := d25
		snap200 := d26
		snap201 := d27
		snap202 := d28
		snap203 := d29
		snap204 := d30
		snap205 := d31
		snap206 := d32
		snap207 := d33
		snap208 := d34
		snap209 := d35
		snap210 := d36
		snap211 := d37
		snap212 := d38
		snap213 := d39
		snap214 := d40
		snap215 := d41
		snap216 := d42
		snap217 := d43
		snap218 := d44
		snap219 := d86
		snap220 := d87
		snap221 := d88
		snap222 := d89
		snap223 := d90
		snap224 := d91
		snap225 := d92
		snap226 := d93
		snap227 := d94
		snap228 := d95
		snap229 := d96
		snap230 := d97
		snap231 := d98
		snap232 := d99
		snap233 := d100
		snap234 := d101
		snap235 := d102
		snap236 := d103
		snap237 := d104
		snap238 := d105
		snap239 := d106
		snap240 := d107
		snap241 := d171
		snap242 := d172
		snap243 := d173
		snap244 := d174
		snap245 := d175
		snap246 := d176
		snap247 := d177
		snap248 := d178
		alloc249 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl17)
		ctx.SyncDesc(&d10)
		if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
			ctx.ProtectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.ProtectReg(d10.Reg)
			ctx.ProtectReg(d10.Reg2)
		}
		d250 = d10
		if d250.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d250)
		d251 = d250
		if d251.Loc == scm.LocImm {
			d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: d251.Type, Imm: scm.NewInt(int64(uint64(d251.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d251.Reg, 32)
			ctx.EmitShrRegImm8(d251.Reg, 32)
		}
		ctx.EmitStoreToStack(d251, int32(bbs[2].PhiBase)+int32(0))
		if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d10.Reg)
			ctx.UnprotectReg(d10.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc249)
		d5 = snap179
		d6 = snap180
		d7 = snap181
		d8 = snap182
		d9 = snap183
		d10 = snap184
		d11 = snap185
		d12 = snap186
		d13 = snap187
		d14 = snap188
		d15 = snap189
		d16 = snap190
		d17 = snap191
		d18 = snap192
		d19 = snap193
		d20 = snap194
		d21 = snap195
		d22 = snap196
		d23 = snap197
		d24 = snap198
		d25 = snap199
		d26 = snap200
		d27 = snap201
		d28 = snap202
		d29 = snap203
		d30 = snap204
		d31 = snap205
		d32 = snap206
		d33 = snap207
		d34 = snap208
		d35 = snap209
		d36 = snap210
		d37 = snap211
		d38 = snap212
		d39 = snap213
		d40 = snap214
		d41 = snap215
		d42 = snap216
		d43 = snap217
		d44 = snap218
		d86 = snap219
		d87 = snap220
		d88 = snap221
		d89 = snap222
		d90 = snap223
		d91 = snap224
		d92 = snap225
		d93 = snap226
		d94 = snap227
		d95 = snap228
		d96 = snap229
		d97 = snap230
		d98 = snap231
		d99 = snap232
		d100 = snap233
		d101 = snap234
		d102 = snap235
		d103 = snap236
		d104 = snap237
		d105 = snap238
		d106 = snap239
		d107 = snap240
		d171 = snap241
		d172 = snap242
		d173 = snap243
		d174 = snap244
		d175 = snap245
		d176 = snap246
		d177 = snap247
		d178 = snap248
		if !bbs[2].Rendered {
			snap252 := d5
			snap253 := d6
			snap254 := d7
			snap255 := d8
			snap256 := d9
			snap257 := d10
			snap258 := d11
			snap259 := d12
			snap260 := d13
			snap261 := d14
			snap262 := d15
			snap263 := d16
			snap264 := d17
			snap265 := d18
			snap266 := d19
			snap267 := d20
			snap268 := d21
			snap269 := d22
			snap270 := d23
			snap271 := d24
			snap272 := d25
			snap273 := d26
			snap274 := d27
			snap275 := d28
			snap276 := d29
			snap277 := d30
			snap278 := d31
			snap279 := d32
			snap280 := d33
			snap281 := d34
			snap282 := d35
			snap283 := d36
			snap284 := d37
			snap285 := d38
			snap286 := d39
			snap287 := d40
			snap288 := d41
			snap289 := d42
			snap290 := d43
			snap291 := d44
			snap292 := d86
			snap293 := d87
			snap294 := d88
			snap295 := d89
			snap296 := d90
			snap297 := d91
			snap298 := d92
			snap299 := d93
			snap300 := d94
			snap301 := d95
			snap302 := d96
			snap303 := d97
			snap304 := d98
			snap305 := d99
			snap306 := d100
			snap307 := d101
			snap308 := d102
			snap309 := d103
			snap310 := d104
			snap311 := d105
			snap312 := d106
			snap313 := d107
			snap314 := d171
			snap315 := d172
			snap316 := d173
			snap317 := d174
			snap318 := d175
			snap319 := d176
			snap320 := d177
			snap321 := d178
			snap322 := d250
			snap323 := d251
			alloc324 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc324)
			d5 = snap252
			d6 = snap253
			d7 = snap254
			d8 = snap255
			d9 = snap256
			d10 = snap257
			d11 = snap258
			d12 = snap259
			d13 = snap260
			d14 = snap261
			d15 = snap262
			d16 = snap263
			d17 = snap264
			d18 = snap265
			d19 = snap266
			d20 = snap267
			d21 = snap268
			d22 = snap269
			d23 = snap270
			d24 = snap271
			d25 = snap272
			d26 = snap273
			d27 = snap274
			d28 = snap275
			d29 = snap276
			d30 = snap277
			d31 = snap278
			d32 = snap279
			d33 = snap280
			d34 = snap281
			d35 = snap282
			d36 = snap283
			d37 = snap284
			d38 = snap285
			d39 = snap286
			d40 = snap287
			d41 = snap288
			d42 = snap289
			d43 = snap290
			d44 = snap291
			d86 = snap292
			d87 = snap293
			d88 = snap294
			d89 = snap295
			d90 = snap296
			d91 = snap297
			d92 = snap298
			d93 = snap299
			d94 = snap300
			d95 = snap301
			d96 = snap302
			d97 = snap303
			d98 = snap304
			d99 = snap305
			d100 = snap306
			d101 = snap307
			d102 = snap308
			d103 = snap309
			d104 = snap310
			d105 = snap311
			d106 = snap312
			d107 = snap313
			d171 = snap314
			d172 = snap315
			d173 = snap316
			d174 = snap317
			d175 = snap318
			d176 = snap319
			d177 = snap320
			d178 = snap321
			d250 = snap322
			d251 = snap323
		}
		if !bbs[6].Rendered {
			return bbs[6].Render()
		}
		return result
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		if d5.Loc == scm.LocImm {
			d325 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 32, scratch, 1)
			d325 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d325)
		}
		if d325.Loc == scm.LocReg && d5.Loc == scm.LocReg && d325.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d325)
		ctx.EmitStoreToStack(d325, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d325)
		ctx.SyncDesc(&d5)
		if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
			ctx.ProtectReg(d5.Reg)
		} else if d5.Loc == scm.LocRegPair {
			ctx.ProtectReg(d5.Reg)
			ctx.ProtectReg(d5.Reg2)
		}
		ctx.SyncDesc(&d7)
		if d7.Loc == scm.LocReg || d7.Loc == scm.LocFPReg {
			ctx.ProtectReg(d7.Reg)
		} else if d7.Loc == scm.LocRegPair {
			ctx.ProtectReg(d7.Reg)
			ctx.ProtectReg(d7.Reg2)
		}
		d326 = d5
		if d326.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d326)
		d327 = d326
		if d327.Loc == scm.LocImm {
			d327 = scm.JITValueDesc{Loc: scm.LocImm, Type: d327.Type, Imm: scm.NewInt(int64(uint64(d327.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d327.Reg, 32)
			ctx.EmitShrRegImm8(d327.Reg, 32)
		}
		ctx.EmitStoreToStack(d327, int32(bbs[4].PhiBase)+int32(16))
		d328 = d7
		if d328.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d328)
		d329 = d328
		if d329.Loc == scm.LocImm {
			d329 = scm.JITValueDesc{Loc: scm.LocImm, Type: d329.Type, Imm: scm.NewInt(int64(uint64(d329.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d329.Reg, 32)
			ctx.EmitShrRegImm8(d329.Reg, 32)
		}
		ctx.EmitStoreToStack(d329, int32(bbs[4].PhiBase)+int32(32))
		if d5.Loc == scm.LocReg || d5.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d5.Reg)
		} else if d5.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d5.Reg)
			ctx.UnprotectReg(d5.Reg2)
		}
		if d7.Loc == scm.LocReg || d7.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d7.Reg)
		} else if d7.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d7.Reg)
			ctx.UnprotectReg(d7.Reg2)
		}
		return bbs[4].Render()
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		d330 = d9
		_ = d330
		ctx.StabilizeDescForControlFlow(&d9)
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl18 := ctx.ReserveLabel()
		_ = lbl18
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl18)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d331 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r63 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r63, thisptr.Reg, off)
			d331 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
			ctx.BindReg(r63, &d331)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d331)
		ctx.EnsureDesc(&d331)
		if d331.Loc == scm.LocImm {
			d332 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d331.Imm.Int()))))}
		} else {
			r64 := ctx.AllocReg()
			ctx.EmitMovRegReg(r64, d331.Reg)
			ctx.EmitShlRegImm8(r64, 56)
			ctx.EmitShrRegImm8(r64, 56)
			d332 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d332)
		}
		ctx.FreeDesc(&d331)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d330)
		ctx.EnsureDesc(&d330)
		if d330.Loc == scm.LocImm {
			d333 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d330.Imm.Int()))))}
		} else {
			r65 := ctx.AllocReg()
			ctx.EmitMovRegReg(r65, d330.Reg)
			ctx.EmitShlRegImm8(r65, 32)
			ctx.EmitShrRegImm8(r65, 32)
			d333 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d333)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d333)
		ctx.EnsureDesc(&d332)
		ctx.EnsureDescsTogether(&d333, &d332)
		if d333.Loc == scm.LocImm && d332.Loc == scm.LocImm {
			d334 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d333.Imm.Int() * d332.Imm.Int())}
		} else if d333.Loc == scm.LocImm {
			ctx.EnsureDesc(&d332)
			scratch := ctx.AllocRegExcept(d332.Reg)
			ctx.EmitMovRegReg(scratch, d332.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d333.Imm.Int())
			d334 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d334)
		} else if d332.Loc == scm.LocImm {
			ctx.EnsureDesc(&d333)
			scratch := ctx.AllocRegExcept(d333.Reg)
			ctx.EmitMovRegReg(scratch, d333.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d332.Imm.Int())
			d334 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d334)
		} else {
			ctx.EnsureDesc(&d333)
			ctx.SyncDesc(&d332)
			r66 := ctx.AllocRegExcept(d333.Reg, d332.Reg)
			ctx.EmitMovRegReg(r66, d333.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r66, &d332)
			d334 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			ctx.BindReg(r66, &d334)
		}
		if d334.Loc == scm.LocReg && d333.Loc == scm.LocReg && d334.Reg == d333.Reg {
			ctx.TransferReg(d333.Reg)
			d333.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d333)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d334)
		if d334.Loc == scm.LocImm {
			d335 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d334.Imm.Int() / 64)}
		} else {
			r67 := ctx.AllocRegExcept(d334.Reg)
			ctx.EmitMovRegReg(r67, d334.Reg)
			ctx.EmitShrRegImm8(r67, 6)
			d335 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d335)
		}
		if d335.Loc == scm.LocReg && d334.Loc == scm.LocReg && d335.Reg == d334.Reg {
			ctx.TransferReg(d334.Reg)
			d334.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d334)
		if d334.Loc == scm.LocImm {
			d336 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d334.Imm.Int() % 64)}
		} else {
			r68 := ctx.AllocRegExcept(d334.Reg)
			ctx.EmitMovRegReg(r68, d334.Reg)
			ctx.EmitAndRegImm32(r68, 63)
			d336 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d336)
		}
		if d336.Loc == scm.LocReg && d334.Loc == scm.LocReg && d336.Reg == d334.Reg {
			ctx.TransferReg(d334.Reg)
			d334.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d334)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d337 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r69 := ctx.AllocReg()
			r70 := ctx.AllocRegExcept(r69)
			r71 := ctx.AllocRegExcept(r69, r70)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r69, thisptr.Reg, off)
			ctx.EmitMovRegMem(r70, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r71, thisptr.Reg, off+16)
			d337 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r69, Reg2: r70, Reg3: r71}
			ctx.BindReg(r69, &d337)
			ctx.BindReg(r70, &d337)
			ctx.BindReg(r71, &d337)
			ctx.BindReg(r69, &d337)
			ctx.BindReg(r70, &d337)
			ctx.BindReg(r71, &d337)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d335)
		ctx.ReclaimUntrackedRegs()
		d338 = ctx.EmitLoadScalarSliceElement(&d337, &d335, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d338)
		ctx.EnsureDesc(&d336)
		ctx.EnsureDescsTogether(&d338, &d336)
		if d338.Loc == scm.LocImm && d336.Loc == scm.LocImm {
			d339 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d338.Imm.Int()) << uint64(d336.Imm.Int())))}
		} else if d336.Loc == scm.LocImm {
			r72 := ctx.AllocRegExcept(d338.Reg)
			ctx.EmitMovRegReg(r72, d338.Reg)
			ctx.EmitShlRegImm8(r72, uint8(d336.Imm.Int()))
			d339 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			ctx.BindReg(r72, &d339)
		} else {
			shiftSrc := d338.Reg
			r73 := ctx.AllocRegExcept(d338.Reg, d336.Reg)
			ctx.EmitMovRegReg(r73, d338.Reg)
			shiftSrc = r73
			ctx.EmitShiftLeft(shiftSrc, d336.Reg, true)
			d339 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d339)
		}
		if d339.Loc == scm.LocReg && d338.Loc == scm.LocReg && d339.Reg == d338.Reg {
			ctx.TransferReg(d338.Reg)
			d338.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d338)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d335)
		ctx.EnsureDesc(&d335)
		if d335.Loc == scm.LocImm {
			d340 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d335.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d335.Reg)
			ctx.EmitMovRegReg(scratch, d335.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d340 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d340)
		}
		if d340.Loc == scm.LocReg && d335.Loc == scm.LocReg && d340.Reg == d335.Reg {
			ctx.TransferReg(d335.Reg)
			d335.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d335)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d340)
		ctx.ReclaimUntrackedRegs()
		d341 = ctx.EmitLoadScalarSliceElement(&d337, &d340, 8, scm.TagInt)
		ctx.FreeDesc(&d340)
		ctx.ReclaimUntrackedRegs()
		d342 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d336)
		ctx.EnsureDescsTogether(&d342, &d336)
		if d342.Loc == scm.LocImm && d336.Loc == scm.LocImm {
			d343 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d342.Imm.Int() - d336.Imm.Int())}
		} else if d336.Loc == scm.LocImm && d336.Imm.Int() == 0 {
			ctx.EnsureDesc(&d342)
			r74 := ctx.AllocRegExcept(d342.Reg)
			ctx.EmitMovRegReg(r74, d342.Reg)
			d343 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d343)
		} else if d342.Loc == scm.LocImm {
			ctx.EnsureDesc(&d336)
			scratch := ctx.AllocRegExcept(d336.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d342.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d336)
			d343 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d343)
		} else if d336.Loc == scm.LocImm {
			ctx.EnsureDesc(&d342)
			scratch := ctx.AllocRegExcept(d342.Reg)
			ctx.EmitMovRegReg(scratch, d342.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d336.Imm.Int())
			d343 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d343)
		} else {
			ctx.EnsureDesc(&d342)
			ctx.SyncDesc(&d336)
			r75 := ctx.AllocRegExcept(d342.Reg, d336.Reg)
			ctx.EmitMovRegReg(r75, d342.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r75, &d336)
			d343 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			ctx.BindReg(r75, &d343)
		}
		if d343.Loc == scm.LocReg && d342.Loc == scm.LocReg && d343.Reg == d342.Reg {
			ctx.TransferReg(d342.Reg)
			d342.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d336)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d341)
		ctx.EnsureDesc(&d343)
		ctx.EnsureDescsTogether(&d341, &d343)
		if d341.Loc == scm.LocImm && d343.Loc == scm.LocImm {
			d344 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d341.Imm.Int()) >> uint64(d343.Imm.Int())))}
		} else if d343.Loc == scm.LocImm {
			r76 := ctx.AllocRegExcept(d341.Reg)
			ctx.EmitMovRegReg(r76, d341.Reg)
			ctx.EmitShrRegImm8(r76, uint8(d343.Imm.Int()))
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			ctx.BindReg(r76, &d344)
		} else {
			shiftSrc := d341.Reg
			r77 := ctx.AllocRegExcept(d341.Reg, d343.Reg)
			ctx.EmitMovRegReg(r77, d341.Reg)
			shiftSrc = r77
			ctx.EmitShiftRight(shiftSrc, d343.Reg, false)
			d344 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d344)
		}
		if d344.Loc == scm.LocReg && d341.Loc == scm.LocReg && d344.Reg == d341.Reg {
			ctx.TransferReg(d341.Reg)
			d341.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d341)
		ctx.FreeDesc(&d343)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d339)
		ctx.EnsureDesc(&d344)
		if d339.Loc == scm.LocImm && d344.Loc == scm.LocImm {
			d345 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d339.Imm.Int() | d344.Imm.Int())}
		} else if d339.Loc == scm.LocImm && d339.Imm.Int() == 0 {
			d345 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d344.Reg}
			ctx.BindReg(d344.Reg, &d345)
		} else if d344.Loc == scm.LocImm && d344.Imm.Int() == 0 {
			r78 := ctx.AllocRegExcept(d339.Reg)
			ctx.EmitMovRegReg(r78, d339.Reg)
			d345 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			ctx.BindReg(r78, &d345)
		} else if d339.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d344.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d339.Imm.Int()))
			ctx.EmitOrInt64(scratch, d344.Reg)
			d345 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d345)
		} else if d344.Loc == scm.LocImm {
			r79 := ctx.AllocRegExcept(d339.Reg)
			ctx.EmitMovRegReg(r79, d339.Reg)
			if d344.Imm.Int() >= -2147483648 && d344.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r79, int32(d344.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d344.Imm.Int()))
				ctx.EmitOrInt64(r79, ctx.ScratchReg)
			}
			d345 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			ctx.BindReg(r79, &d345)
		} else {
			r80 := ctx.AllocRegExcept(d339.Reg, d344.Reg)
			ctx.EmitMovRegReg(r80, d339.Reg)
			ctx.EmitOrInt64(r80, d344.Reg)
			d345 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			ctx.BindReg(r80, &d345)
		}
		if d345.Loc == scm.LocReg && d339.Loc == scm.LocReg && d345.Reg == d339.Reg {
			ctx.TransferReg(d339.Reg)
			d339.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d339)
		ctx.FreeDesc(&d344)
		ctx.ReclaimUntrackedRegs()
		d346 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d332)
		ctx.EnsureDescsTogether(&d346, &d332)
		if d346.Loc == scm.LocImm && d332.Loc == scm.LocImm {
			d347 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d346.Imm.Int() - d332.Imm.Int())}
		} else if d332.Loc == scm.LocImm && d332.Imm.Int() == 0 {
			ctx.EnsureDesc(&d346)
			r81 := ctx.AllocRegExcept(d346.Reg)
			ctx.EmitMovRegReg(r81, d346.Reg)
			d347 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d347)
		} else if d346.Loc == scm.LocImm {
			ctx.EnsureDesc(&d332)
			scratch := ctx.AllocRegExcept(d332.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d346.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d332)
			d347 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d347)
		} else if d332.Loc == scm.LocImm {
			ctx.EnsureDesc(&d346)
			scratch := ctx.AllocRegExcept(d346.Reg)
			ctx.EmitMovRegReg(scratch, d346.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d332.Imm.Int())
			d347 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d347)
		} else {
			ctx.EnsureDesc(&d346)
			ctx.SyncDesc(&d332)
			r82 := ctx.AllocRegExcept(d346.Reg, d332.Reg)
			ctx.EmitMovRegReg(r82, d346.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r82, &d332)
			d347 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			ctx.BindReg(r82, &d347)
		}
		if d347.Loc == scm.LocReg && d346.Loc == scm.LocReg && d347.Reg == d346.Reg {
			ctx.TransferReg(d346.Reg)
			d346.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d332)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d345)
		ctx.EnsureDesc(&d347)
		ctx.EnsureDescsTogether(&d345, &d347)
		if d345.Loc == scm.LocImm && d347.Loc == scm.LocImm {
			d348 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d345.Imm.Int()) >> uint64(d347.Imm.Int())))}
		} else if d347.Loc == scm.LocImm {
			r83 := ctx.AllocRegExcept(d345.Reg)
			ctx.EmitMovRegReg(r83, d345.Reg)
			ctx.EmitShrRegImm8(r83, uint8(d347.Imm.Int()))
			d348 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d348)
		} else {
			shiftSrc := d345.Reg
			r84 := ctx.AllocRegExcept(d345.Reg, d347.Reg)
			ctx.EmitMovRegReg(r84, d345.Reg)
			shiftSrc = r84
			ctx.EmitShiftRight(shiftSrc, d347.Reg, false)
			d348 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d348)
		}
		if d348.Loc == scm.LocReg && d345.Loc == scm.LocReg && d348.Reg == d345.Reg {
			ctx.TransferReg(d345.Reg)
			d345.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d345)
		ctx.FreeDesc(&d347)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d348)
		ctx.EnsureDesc(&d348)
		ctx.EnsureDesc(&d348)
		if d348.Loc == scm.LocImm {
			d349 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d348.Imm.Int()))))}
		} else {
			r85 := ctx.AllocReg()
			ctx.EmitMovRegReg(r85, d348.Reg)
			d349 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d349)
		}
		ctx.FreeDesc(&d348)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d350 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r86 := ctx.AllocReg()
			ctx.EmitMovRegMem(r86, thisptr.Reg, off)
			d350 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r86}
			ctx.BindReg(r86, &d350)
		}
		ctx.EnsureDesc(&d349)
		ctx.EnsureDesc(&d350)
		ctx.EnsureDescsTogether(&d349, &d350)
		if d349.Loc == scm.LocImm && d350.Loc == scm.LocImm {
			d351 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d349.Imm.Int() + d350.Imm.Int())}
		} else if d350.Loc == scm.LocImm && d350.Imm.Int() == 0 {
			ctx.EnsureDesc(&d349)
			r87 := ctx.AllocRegExcept(d349.Reg)
			ctx.EmitMovRegReg(r87, d349.Reg)
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d351)
		} else if d349.Loc == scm.LocImm && d349.Imm.Int() == 0 {
			ctx.EnsureDesc(&d350)
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d350.Reg}
			ctx.BindReg(d350.Reg, &d351)
		} else if d349.Loc == scm.LocImm {
			ctx.EnsureDesc(&d350)
			scratch := ctx.AllocRegExcept(d350.Reg)
			ctx.EmitMovRegReg(scratch, d350.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d349.Imm.Int())
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d351)
		} else if d350.Loc == scm.LocImm {
			ctx.EnsureDesc(&d349)
			scratch := ctx.AllocRegExcept(d349.Reg)
			ctx.EmitMovRegReg(scratch, d349.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d350.Imm.Int())
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d351)
		} else {
			ctx.EnsureDesc(&d349)
			ctx.SyncDesc(&d350)
			r88 := ctx.AllocRegExcept(d349.Reg, d350.Reg)
			ctx.EmitMovRegReg(r88, d349.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r88, &d350)
			d351 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
			ctx.BindReg(r88, &d351)
		}
		if d351.Loc == scm.LocReg && d349.Loc == scm.LocReg && d351.Reg == d349.Reg {
			ctx.TransferReg(d349.Reg)
			d349.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d349)
		ctx.FreeDesc(&d350)
		ctx.EnsureDesc(&d351)
		ctx.EnsureDesc(&d351)
		if d351.Loc == scm.LocImm {
			d352 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d351.Imm.Int()))))}
		} else {
			r89 := ctx.AllocReg()
			ctx.EmitMovRegReg(r89, d351.Reg)
			ctx.EmitShlRegImm8(r89, 32)
			ctx.EmitShrRegImm8(r89, 32)
			d352 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d352)
		}
		ctx.FreeDesc(&d351)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d352)
		ctx.EnsureDescsTogether(&idxInt, &d352)
		if idxInt.Loc == scm.LocImm && d352.Loc == scm.LocImm {
			d353 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d352.Imm.Int()))}
		} else if d352.Loc == scm.LocImm {
			r90 := ctx.AllocRegExcept(idxInt.Reg)
			if d352.Imm.Int() >= -2147483648 && d352.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d352.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d352.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, ctx.ScratchReg)
			}
			d353 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r90, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r90, &d353)
		} else if idxInt.Loc == scm.LocImm {
			r91 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d352.Reg)
			d353 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r91, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r91, &d353)
		} else {
			r92 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d352.Reg)
			d353 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r92, Condition: scm.CondUnsignedBelow}
			ctx.BindReg(r92, &d353)
		}
		ctx.FreeDesc(&d352)
		d354 = d353
		ctx.EnsureDesc(&d354)
		if d354.Loc != scm.LocImm && d354.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d354.Loc == scm.LocImm {
			if d354.Imm.Bool() {
				return bbs[7].Render()
			}
			return bbs[9].Render()
		}
		ctx.EmitJump(d354.Condition, lbl8)
		if bbs[9].Rendered {
			ctx.EmitJmp(lbl10)
		}
		ctx.FreeDesc(&d353)
		ctx.FlushRegisterMoves()
		if !bbs[9].Rendered {
			snap355 := d5
			snap356 := d6
			snap357 := d7
			snap358 := d8
			snap359 := d9
			snap360 := d10
			snap361 := d11
			snap362 := d12
			snap363 := d13
			snap364 := d14
			snap365 := d15
			snap366 := d16
			snap367 := d17
			snap368 := d18
			snap369 := d19
			snap370 := d20
			snap371 := d21
			snap372 := d22
			snap373 := d23
			snap374 := d24
			snap375 := d25
			snap376 := d26
			snap377 := d27
			snap378 := d28
			snap379 := d29
			snap380 := d30
			snap381 := d31
			snap382 := d32
			snap383 := d33
			snap384 := d34
			snap385 := d35
			snap386 := d36
			snap387 := d37
			snap388 := d38
			snap389 := d39
			snap390 := d40
			snap391 := d41
			snap392 := d42
			snap393 := d43
			snap394 := d44
			snap395 := d86
			snap396 := d87
			snap397 := d88
			snap398 := d89
			snap399 := d90
			snap400 := d91
			snap401 := d92
			snap402 := d93
			snap403 := d94
			snap404 := d95
			snap405 := d96
			snap406 := d97
			snap407 := d98
			snap408 := d99
			snap409 := d100
			snap410 := d101
			snap411 := d102
			snap412 := d103
			snap413 := d104
			snap414 := d105
			snap415 := d106
			snap416 := d107
			snap417 := d171
			snap418 := d172
			snap419 := d173
			snap420 := d174
			snap421 := d175
			snap422 := d176
			snap423 := d177
			snap424 := d178
			snap425 := d250
			snap426 := d251
			snap427 := d325
			snap428 := d326
			snap429 := d327
			snap430 := d328
			snap431 := d329
			snap432 := d330
			snap433 := d331
			snap434 := d332
			snap435 := d333
			snap436 := d334
			snap437 := d335
			snap438 := d336
			snap439 := d337
			snap440 := d338
			snap441 := d339
			snap442 := d340
			snap443 := d341
			snap444 := d342
			snap445 := d343
			snap446 := d344
			snap447 := d345
			snap448 := d346
			snap449 := d347
			snap450 := d348
			snap451 := d349
			snap452 := d350
			snap453 := d351
			snap454 := d352
			snap455 := d353
			snap456 := d354
			alloc457 := ctx.SnapshotAllocState()
			bbs[9].Render()
			ctx.RestoreAllocState(alloc457)
			d5 = snap355
			d6 = snap356
			d7 = snap357
			d8 = snap358
			d9 = snap359
			d10 = snap360
			d11 = snap361
			d12 = snap362
			d13 = snap363
			d14 = snap364
			d15 = snap365
			d16 = snap366
			d17 = snap367
			d18 = snap368
			d19 = snap369
			d20 = snap370
			d21 = snap371
			d22 = snap372
			d23 = snap373
			d24 = snap374
			d25 = snap375
			d26 = snap376
			d27 = snap377
			d28 = snap378
			d29 = snap379
			d30 = snap380
			d31 = snap381
			d32 = snap382
			d33 = snap383
			d34 = snap384
			d35 = snap385
			d36 = snap386
			d37 = snap387
			d38 = snap388
			d39 = snap389
			d40 = snap390
			d41 = snap391
			d42 = snap392
			d43 = snap393
			d44 = snap394
			d86 = snap395
			d87 = snap396
			d88 = snap397
			d89 = snap398
			d90 = snap399
			d91 = snap400
			d92 = snap401
			d93 = snap402
			d94 = snap403
			d95 = snap404
			d96 = snap405
			d97 = snap406
			d98 = snap407
			d99 = snap408
			d100 = snap409
			d101 = snap410
			d102 = snap411
			d103 = snap412
			d104 = snap413
			d105 = snap414
			d106 = snap415
			d107 = snap416
			d171 = snap417
			d172 = snap418
			d173 = snap419
			d174 = snap420
			d175 = snap421
			d176 = snap422
			d177 = snap423
			d178 = snap424
			d250 = snap425
			d251 = snap426
			d325 = snap427
			d326 = snap428
			d327 = snap429
			d328 = snap430
			d329 = snap431
			d330 = snap432
			d331 = snap433
			d332 = snap434
			d333 = snap435
			d334 = snap436
			d335 = snap437
			d336 = snap438
			d337 = snap439
			d338 = snap440
			d339 = snap441
			d340 = snap442
			d341 = snap443
			d342 = snap444
			d343 = snap445
			d344 = snap446
			d345 = snap447
			d346 = snap448
			d347 = snap449
			d348 = snap450
			d349 = snap451
			d350 = snap452
			d351 = snap453
			d352 = snap454
			d353 = snap455
			d354 = snap456
		}
		if !bbs[7].Rendered {
			return bbs[7].Render()
		}
		return result
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d9)
		if d9.Loc == scm.LocImm {
			d458 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(scratch, d9.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 32, scratch, 1)
			d458 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d458)
		}
		if d458.Loc == scm.LocReg && d9.Loc == scm.LocReg && d458.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d458)
		ctx.EmitStoreToStack(d458, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d458)
		ctx.SyncDesc(&d10)
		if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
			ctx.ProtectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.ProtectReg(d10.Reg)
			ctx.ProtectReg(d10.Reg2)
		}
		d459 = d10
		if d459.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d459)
		d460 = d459
		if d460.Loc == scm.LocImm {
			d460 = scm.JITValueDesc{Loc: scm.LocImm, Type: d460.Type, Imm: scm.NewInt(int64(uint64(d460.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d460.Reg, 32)
			ctx.EmitShrRegImm8(d460.Reg, 32)
		}
		ctx.EmitStoreToStack(d460, int32(bbs[8].PhiBase)+int32(0))
		if d10.Loc == scm.LocReg || d10.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d10.Reg)
			ctx.UnprotectReg(d10.Reg2)
		}
		return bbs[8].Render()
		return result
	}
	bbs[8].Render = func() scm.JITValueDesc {
		if bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		bbs[8].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_8 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl9)
		ctx.ResolveFixups()
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d12)
		ctx.StabilizeDescForControlFlow(&d13)
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d12, &d13)
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d461 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d12.Imm.Int()) == uint64(d13.Imm.Int()))}
		} else if d13.Loc == scm.LocImm {
			r93 := ctx.AllocRegExcept(d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d12.Reg, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d13.Imm.Int()))
				ctx.EmitCmpInt64(d12.Reg, ctx.ScratchReg)
			}
			d461 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r93, Condition: scm.CondEqual}
			ctx.BindReg(r93, &d461)
		} else if d12.Loc == scm.LocImm {
			r94 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d12.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d13.Reg)
			d461 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r94, Condition: scm.CondEqual}
			ctx.BindReg(r94, &d461)
		} else {
			r95 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitCmpInt64(d12.Reg, d13.Reg)
			d461 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r95, Condition: scm.CondEqual}
			ctx.BindReg(r95, &d461)
		}
		d462 = d461
		ctx.EnsureDesc(&d462)
		if d462.Loc != scm.LocImm && d462.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d462.Loc == scm.LocImm {
			if d462.Imm.Bool() {
				ctx.SyncDesc(&d12)
				if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
					ctx.ProtectReg(d12.Reg)
				} else if d12.Loc == scm.LocRegPair {
					ctx.ProtectReg(d12.Reg)
					ctx.ProtectReg(d12.Reg2)
				}
				d463 = d12
				if d463.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d463)
				d464 = d463
				if d464.Loc == scm.LocImm {
					d464 = scm.JITValueDesc{Loc: scm.LocImm, Type: d464.Type, Imm: scm.NewInt(int64(uint64(d464.Imm.Int()) & 0xffffffff))}
				} else {
					ctx.EmitShlRegImm8(d464.Reg, 32)
					ctx.EmitShrRegImm8(d464.Reg, 32)
				}
				ctx.EmitStoreToStack(d464, int32(bbs[2].PhiBase)+int32(0))
				if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
					ctx.UnprotectReg(d12.Reg)
				} else if d12.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d12.Reg)
					ctx.UnprotectReg(d12.Reg2)
				}
				return bbs[2].Render()
			}
			return bbs[10].Render()
		}
		lbl19 := ctx.ReserveLabel()
		ctx.EmitJump(d462.Condition, lbl19)
		ctx.EmitJmp(lbl11)
		ctx.FreeDesc(&d461)
		snap465 := d5
		snap466 := d6
		snap467 := d7
		snap468 := d8
		snap469 := d9
		snap470 := d10
		snap471 := d11
		snap472 := d12
		snap473 := d13
		snap474 := d14
		snap475 := d15
		snap476 := d16
		snap477 := d17
		snap478 := d18
		snap479 := d19
		snap480 := d20
		snap481 := d21
		snap482 := d22
		snap483 := d23
		snap484 := d24
		snap485 := d25
		snap486 := d26
		snap487 := d27
		snap488 := d28
		snap489 := d29
		snap490 := d30
		snap491 := d31
		snap492 := d32
		snap493 := d33
		snap494 := d34
		snap495 := d35
		snap496 := d36
		snap497 := d37
		snap498 := d38
		snap499 := d39
		snap500 := d40
		snap501 := d41
		snap502 := d42
		snap503 := d43
		snap504 := d44
		snap505 := d86
		snap506 := d87
		snap507 := d88
		snap508 := d89
		snap509 := d90
		snap510 := d91
		snap511 := d92
		snap512 := d93
		snap513 := d94
		snap514 := d95
		snap515 := d96
		snap516 := d97
		snap517 := d98
		snap518 := d99
		snap519 := d100
		snap520 := d101
		snap521 := d102
		snap522 := d103
		snap523 := d104
		snap524 := d105
		snap525 := d106
		snap526 := d107
		snap527 := d171
		snap528 := d172
		snap529 := d173
		snap530 := d174
		snap531 := d175
		snap532 := d176
		snap533 := d177
		snap534 := d178
		snap535 := d250
		snap536 := d251
		snap537 := d325
		snap538 := d326
		snap539 := d327
		snap540 := d328
		snap541 := d329
		snap542 := d330
		snap543 := d331
		snap544 := d332
		snap545 := d333
		snap546 := d334
		snap547 := d335
		snap548 := d336
		snap549 := d337
		snap550 := d338
		snap551 := d339
		snap552 := d340
		snap553 := d341
		snap554 := d342
		snap555 := d343
		snap556 := d344
		snap557 := d345
		snap558 := d346
		snap559 := d347
		snap560 := d348
		snap561 := d349
		snap562 := d350
		snap563 := d351
		snap564 := d352
		snap565 := d353
		snap566 := d354
		snap567 := d458
		snap568 := d459
		snap569 := d460
		snap570 := d461
		snap571 := d462
		snap572 := d463
		snap573 := d464
		alloc574 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl19)
		ctx.SyncDesc(&d12)
		if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
			ctx.ProtectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.ProtectReg(d12.Reg)
			ctx.ProtectReg(d12.Reg2)
		}
		d575 = d12
		if d575.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d575)
		d576 = d575
		if d576.Loc == scm.LocImm {
			d576 = scm.JITValueDesc{Loc: scm.LocImm, Type: d576.Type, Imm: scm.NewInt(int64(uint64(d576.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d576.Reg, 32)
			ctx.EmitShrRegImm8(d576.Reg, 32)
		}
		ctx.EmitStoreToStack(d576, int32(bbs[2].PhiBase)+int32(0))
		if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d12.Reg)
			ctx.UnprotectReg(d12.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc574)
		d5 = snap465
		d6 = snap466
		d7 = snap467
		d8 = snap468
		d9 = snap469
		d10 = snap470
		d11 = snap471
		d12 = snap472
		d13 = snap473
		d14 = snap474
		d15 = snap475
		d16 = snap476
		d17 = snap477
		d18 = snap478
		d19 = snap479
		d20 = snap480
		d21 = snap481
		d22 = snap482
		d23 = snap483
		d24 = snap484
		d25 = snap485
		d26 = snap486
		d27 = snap487
		d28 = snap488
		d29 = snap489
		d30 = snap490
		d31 = snap491
		d32 = snap492
		d33 = snap493
		d34 = snap494
		d35 = snap495
		d36 = snap496
		d37 = snap497
		d38 = snap498
		d39 = snap499
		d40 = snap500
		d41 = snap501
		d42 = snap502
		d43 = snap503
		d44 = snap504
		d86 = snap505
		d87 = snap506
		d88 = snap507
		d89 = snap508
		d90 = snap509
		d91 = snap510
		d92 = snap511
		d93 = snap512
		d94 = snap513
		d95 = snap514
		d96 = snap515
		d97 = snap516
		d98 = snap517
		d99 = snap518
		d100 = snap519
		d101 = snap520
		d102 = snap521
		d103 = snap522
		d104 = snap523
		d105 = snap524
		d106 = snap525
		d107 = snap526
		d171 = snap527
		d172 = snap528
		d173 = snap529
		d174 = snap530
		d175 = snap531
		d176 = snap532
		d177 = snap533
		d178 = snap534
		d250 = snap535
		d251 = snap536
		d325 = snap537
		d326 = snap538
		d327 = snap539
		d328 = snap540
		d329 = snap541
		d330 = snap542
		d331 = snap543
		d332 = snap544
		d333 = snap545
		d334 = snap546
		d335 = snap547
		d336 = snap548
		d337 = snap549
		d338 = snap550
		d339 = snap551
		d340 = snap552
		d341 = snap553
		d342 = snap554
		d343 = snap555
		d344 = snap556
		d345 = snap557
		d346 = snap558
		d347 = snap559
		d348 = snap560
		d349 = snap561
		d350 = snap562
		d351 = snap563
		d352 = snap564
		d353 = snap565
		d354 = snap566
		d458 = snap567
		d459 = snap568
		d460 = snap569
		d461 = snap570
		d462 = snap571
		d463 = snap572
		d464 = snap573
		if !bbs[2].Rendered {
			snap577 := d5
			snap578 := d6
			snap579 := d7
			snap580 := d8
			snap581 := d9
			snap582 := d10
			snap583 := d11
			snap584 := d12
			snap585 := d13
			snap586 := d14
			snap587 := d15
			snap588 := d16
			snap589 := d17
			snap590 := d18
			snap591 := d19
			snap592 := d20
			snap593 := d21
			snap594 := d22
			snap595 := d23
			snap596 := d24
			snap597 := d25
			snap598 := d26
			snap599 := d27
			snap600 := d28
			snap601 := d29
			snap602 := d30
			snap603 := d31
			snap604 := d32
			snap605 := d33
			snap606 := d34
			snap607 := d35
			snap608 := d36
			snap609 := d37
			snap610 := d38
			snap611 := d39
			snap612 := d40
			snap613 := d41
			snap614 := d42
			snap615 := d43
			snap616 := d44
			snap617 := d86
			snap618 := d87
			snap619 := d88
			snap620 := d89
			snap621 := d90
			snap622 := d91
			snap623 := d92
			snap624 := d93
			snap625 := d94
			snap626 := d95
			snap627 := d96
			snap628 := d97
			snap629 := d98
			snap630 := d99
			snap631 := d100
			snap632 := d101
			snap633 := d102
			snap634 := d103
			snap635 := d104
			snap636 := d105
			snap637 := d106
			snap638 := d107
			snap639 := d171
			snap640 := d172
			snap641 := d173
			snap642 := d174
			snap643 := d175
			snap644 := d176
			snap645 := d177
			snap646 := d178
			snap647 := d250
			snap648 := d251
			snap649 := d325
			snap650 := d326
			snap651 := d327
			snap652 := d328
			snap653 := d329
			snap654 := d330
			snap655 := d331
			snap656 := d332
			snap657 := d333
			snap658 := d334
			snap659 := d335
			snap660 := d336
			snap661 := d337
			snap662 := d338
			snap663 := d339
			snap664 := d340
			snap665 := d341
			snap666 := d342
			snap667 := d343
			snap668 := d344
			snap669 := d345
			snap670 := d346
			snap671 := d347
			snap672 := d348
			snap673 := d349
			snap674 := d350
			snap675 := d351
			snap676 := d352
			snap677 := d353
			snap678 := d354
			snap679 := d458
			snap680 := d459
			snap681 := d460
			snap682 := d461
			snap683 := d462
			snap684 := d463
			snap685 := d464
			snap686 := d575
			snap687 := d576
			alloc688 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc688)
			d5 = snap577
			d6 = snap578
			d7 = snap579
			d8 = snap580
			d9 = snap581
			d10 = snap582
			d11 = snap583
			d12 = snap584
			d13 = snap585
			d14 = snap586
			d15 = snap587
			d16 = snap588
			d17 = snap589
			d18 = snap590
			d19 = snap591
			d20 = snap592
			d21 = snap593
			d22 = snap594
			d23 = snap595
			d24 = snap596
			d25 = snap597
			d26 = snap598
			d27 = snap599
			d28 = snap600
			d29 = snap601
			d30 = snap602
			d31 = snap603
			d32 = snap604
			d33 = snap605
			d34 = snap606
			d35 = snap607
			d36 = snap608
			d37 = snap609
			d38 = snap610
			d39 = snap611
			d40 = snap612
			d41 = snap613
			d42 = snap614
			d43 = snap615
			d44 = snap616
			d86 = snap617
			d87 = snap618
			d88 = snap619
			d89 = snap620
			d90 = snap621
			d91 = snap622
			d92 = snap623
			d93 = snap624
			d94 = snap625
			d95 = snap626
			d96 = snap627
			d97 = snap628
			d98 = snap629
			d99 = snap630
			d100 = snap631
			d101 = snap632
			d102 = snap633
			d103 = snap634
			d104 = snap635
			d105 = snap636
			d106 = snap637
			d107 = snap638
			d171 = snap639
			d172 = snap640
			d173 = snap641
			d174 = snap642
			d175 = snap643
			d176 = snap644
			d177 = snap645
			d178 = snap646
			d250 = snap647
			d251 = snap648
			d325 = snap649
			d326 = snap650
			d327 = snap651
			d328 = snap652
			d329 = snap653
			d330 = snap654
			d331 = snap655
			d332 = snap656
			d333 = snap657
			d334 = snap658
			d335 = snap659
			d336 = snap660
			d337 = snap661
			d338 = snap662
			d339 = snap663
			d340 = snap664
			d341 = snap665
			d342 = snap666
			d343 = snap667
			d344 = snap668
			d345 = snap669
			d346 = snap670
			d347 = snap671
			d348 = snap672
			d349 = snap673
			d350 = snap674
			d351 = snap675
			d352 = snap676
			d353 = snap677
			d354 = snap678
			d458 = snap679
			d459 = snap680
			d460 = snap681
			d461 = snap682
			d462 = snap683
			d463 = snap684
			d464 = snap685
			d575 = snap686
			d576 = snap687
		}
		if !bbs[10].Rendered {
			return bbs[10].Render()
		}
		return result
		return result
	}
	bbs[9].Render = func() scm.JITValueDesc {
		if bbs[9].Rendered {
			ctx.EmitJmp(lbl10)
			return result
		}
		bbs[9].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_9 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl10)
		ctx.ResolveFixups()
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.SyncDesc(&d9)
		if d9.Loc == scm.LocReg || d9.Loc == scm.LocFPReg {
			ctx.ProtectReg(d9.Reg)
		} else if d9.Loc == scm.LocRegPair {
			ctx.ProtectReg(d9.Reg)
			ctx.ProtectReg(d9.Reg2)
		}
		ctx.SyncDesc(&d11)
		if d11.Loc == scm.LocReg || d11.Loc == scm.LocFPReg {
			ctx.ProtectReg(d11.Reg)
		} else if d11.Loc == scm.LocRegPair {
			ctx.ProtectReg(d11.Reg)
			ctx.ProtectReg(d11.Reg2)
		}
		d689 = d9
		if d689.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d689)
		d690 = d689
		if d690.Loc == scm.LocImm {
			d690 = scm.JITValueDesc{Loc: scm.LocImm, Type: d690.Type, Imm: scm.NewInt(int64(uint64(d690.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d690.Reg, 32)
			ctx.EmitShrRegImm8(d690.Reg, 32)
		}
		ctx.EmitStoreToStack(d690, int32(bbs[8].PhiBase)+int32(0))
		d691 = d11
		if d691.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d691)
		d692 = d691
		if d692.Loc == scm.LocImm {
			d692 = scm.JITValueDesc{Loc: scm.LocImm, Type: d692.Type, Imm: scm.NewInt(int64(uint64(d692.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d692.Reg, 32)
			ctx.EmitShrRegImm8(d692.Reg, 32)
		}
		ctx.EmitStoreToStack(d692, int32(bbs[8].PhiBase)+int32(16))
		if d9.Loc == scm.LocReg || d9.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d9.Reg)
		} else if d9.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d9.Reg)
			ctx.UnprotectReg(d9.Reg2)
		}
		if d11.Loc == scm.LocReg || d11.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d11.Reg)
		} else if d11.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d11.Reg)
			ctx.UnprotectReg(d11.Reg2)
		}
		return bbs[8].Render()
		return result
	}
	bbs[10].Render = func() scm.JITValueDesc {
		if bbs[10].Rendered {
			ctx.EmitJmp(lbl11)
			return result
		}
		bbs[10].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_10 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl11)
		ctx.ResolveFixups()
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d12, &d13)
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d693 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() + d13.Imm.Int())}
		} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
			ctx.EnsureDesc(&d12)
			r96 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r96, d12.Reg)
			d693 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d693)
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			ctx.EnsureDesc(&d13)
			d693 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			ctx.BindReg(d13.Reg, &d693)
		} else if d12.Loc == scm.LocImm {
			ctx.EnsureDesc(&d13)
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegReg(scratch, d13.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 32, scratch, d12.Imm.Int())
			d693 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d693)
		} else if d13.Loc == scm.LocImm {
			ctx.EnsureDesc(&d12)
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 32, scratch, d13.Imm.Int())
			d693 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d693)
		} else {
			ctx.EnsureDesc(&d12)
			ctx.SyncDesc(&d13)
			r97 := ctx.AllocRegExcept(d12.Reg, d13.Reg)
			ctx.EmitMovRegReg(r97, d12.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 32, r97, &d13)
			d693 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d693)
		}
		if d693.Loc == scm.LocReg && d12.Loc == scm.LocReg && d693.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d693)
		if d693.Loc == scm.LocImm {
			d694 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d693.Imm.Int() / 2)}
		} else {
			r98 := ctx.AllocRegExcept(d693.Reg)
			ctx.EmitMovRegReg(r98, d693.Reg)
			ctx.EmitShrRegImm8(r98, 1)
			d694 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			ctx.BindReg(r98, &d694)
		}
		if d694.Loc == scm.LocImm {
			d694 = scm.JITValueDesc{Loc: scm.LocImm, Type: d694.Type, Imm: scm.NewInt(int64(uint64(d694.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d694.Reg, 32)
			ctx.EmitShrRegImm8(d694.Reg, 32)
		}
		if d694.Loc == scm.LocReg && d693.Loc == scm.LocReg && d694.Reg == d693.Reg {
			ctx.TransferReg(d693.Reg)
			d693.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d693)
		ctx.SyncDesc(&d12)
		if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
			ctx.ProtectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.ProtectReg(d12.Reg)
			ctx.ProtectReg(d12.Reg2)
		}
		ctx.SyncDesc(&d13)
		if d13.Loc == scm.LocReg || d13.Loc == scm.LocFPReg {
			ctx.ProtectReg(d13.Reg)
		} else if d13.Loc == scm.LocRegPair {
			ctx.ProtectReg(d13.Reg)
			ctx.ProtectReg(d13.Reg2)
		}
		ctx.SyncDesc(&d694)
		if d694.Loc == scm.LocReg || d694.Loc == scm.LocFPReg {
			ctx.ProtectReg(d694.Reg)
		} else if d694.Loc == scm.LocRegPair {
			ctx.ProtectReg(d694.Reg)
			ctx.ProtectReg(d694.Reg2)
		}
		d695 = d694
		if d695.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d695)
		d696 = d695
		if d696.Loc == scm.LocImm {
			d696 = scm.JITValueDesc{Loc: scm.LocImm, Type: d696.Type, Imm: scm.NewInt(int64(uint64(d696.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d696.Reg, 32)
			ctx.EmitShrRegImm8(d696.Reg, 32)
		}
		if phiHomeOK2 {
			ctx.EmitMovToReg(r0, d696)
		} else {
			ctx.EmitStoreToStack(d696, int32(bbs[1].PhiBase)+int32(0))
		}
		d697 = d12
		if d697.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d697)
		d698 = d697
		if d698.Loc == scm.LocImm {
			d698 = scm.JITValueDesc{Loc: scm.LocImm, Type: d698.Type, Imm: scm.NewInt(int64(uint64(d698.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d698.Reg, 32)
			ctx.EmitShrRegImm8(d698.Reg, 32)
		}
		if phiHomeOK3 {
			ctx.EmitMovToReg(r1, d698)
		} else {
			ctx.EmitStoreToStack(d698, int32(bbs[1].PhiBase)+int32(16))
		}
		d699 = d13
		if d699.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d699)
		d700 = d699
		if d700.Loc == scm.LocImm {
			d700 = scm.JITValueDesc{Loc: scm.LocImm, Type: d700.Type, Imm: scm.NewInt(int64(uint64(d700.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d700.Reg, 32)
			ctx.EmitShrRegImm8(d700.Reg, 32)
		}
		if phiHomeOK4 {
			ctx.EmitMovToReg(r2, d700)
		} else {
			ctx.EmitStoreToStack(d700, int32(bbs[1].PhiBase)+int32(32))
		}
		if d12.Loc == scm.LocReg || d12.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d12.Reg)
			ctx.UnprotectReg(d12.Reg2)
		}
		if d13.Loc == scm.LocReg || d13.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d13.Reg)
		} else if d13.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d13.Reg)
			ctx.UnprotectReg(d13.Reg2)
		}
		if d694.Loc == scm.LocReg || d694.Loc == scm.LocFPReg {
			ctx.UnprotectReg(d694.Reg)
		} else if d694.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d694.Reg)
			ctx.UnprotectReg(d694.Reg2)
		}
		return bbs[1].Render()
		return result
	}
	bbs[11].Render = func() scm.JITValueDesc {
		if bbs[11].Rendered {
			ctx.EmitJmp(lbl12)
			return result
		}
		bbs[11].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_11 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl12)
		ctx.ResolveFixups()
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		d701 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d702 = result
		ctx.EnsureDesc(&d701)
		if d701.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d701, &d702)
		} else {
			switch d701.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d702, d701)
			case scm.TagInt:
				ctx.EmitMakeInt(d702, d701)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d702, d701)
			case scm.TagNil:
				ctx.EmitMakeNil(d702)
			default:
				ctx.EmitMovPairToResult(&d701, &d702)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[12].Render = func() scm.JITValueDesc {
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
			return result
		}
		bbs[12].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_12 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl13)
		ctx.ResolveFixups()
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d105)
		ctx.EnsureDesc(&d105)
		if d105.Loc == scm.LocImm {
			d703 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d105.Imm.Int()))))}
		} else {
			r99 := ctx.AllocReg()
			ctx.EmitMovRegReg(r99, d105.Reg)
			d703 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
			ctx.BindReg(r99, &d703)
		}
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d704 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r100 := ctx.AllocReg()
			ctx.EmitMovRegMem(r100, thisptr.Reg, off)
			d704 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
			ctx.BindReg(r100, &d704)
		}
		ctx.EnsureDesc(&d703)
		ctx.EnsureDesc(&d704)
		ctx.EnsureDescsTogether(&d703, &d704)
		if d703.Loc == scm.LocImm && d704.Loc == scm.LocImm {
			d705 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d703.Imm.Int() + d704.Imm.Int())}
		} else if d704.Loc == scm.LocImm && d704.Imm.Int() == 0 {
			ctx.EnsureDesc(&d703)
			r101 := ctx.AllocRegExcept(d703.Reg)
			ctx.EmitMovRegReg(r101, d703.Reg)
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			ctx.BindReg(r101, &d705)
		} else if d703.Loc == scm.LocImm && d703.Imm.Int() == 0 {
			ctx.EnsureDesc(&d704)
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d704.Reg}
			ctx.BindReg(d704.Reg, &d705)
		} else if d703.Loc == scm.LocImm {
			ctx.EnsureDesc(&d704)
			scratch := ctx.AllocRegExcept(d704.Reg)
			ctx.EmitMovRegReg(scratch, d704.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d703.Imm.Int())
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d705)
		} else if d704.Loc == scm.LocImm {
			ctx.EnsureDesc(&d703)
			scratch := ctx.AllocRegExcept(d703.Reg)
			ctx.EmitMovRegReg(scratch, d703.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d704.Imm.Int())
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d705)
		} else {
			ctx.EnsureDesc(&d703)
			ctx.SyncDesc(&d704)
			r102 := ctx.AllocRegExcept(d703.Reg, d704.Reg)
			ctx.EmitMovRegReg(r102, d703.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r102, &d704)
			d705 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d705)
		}
		if d705.Loc == scm.LocReg && d703.Loc == scm.LocReg && d705.Reg == d703.Reg {
			ctx.TransferReg(d703.Reg)
			d703.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d703)
		ctx.FreeDesc(&d704)
		ctx.EnsureDesc(&d8)
		d706 = d8
		_ = d706
		ctx.StabilizeDescForControlFlow(&d8)
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl20 := ctx.ReserveLabel()
		_ = lbl20
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl20)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d707 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r103 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r103, thisptr.Reg, off)
			d707 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			ctx.BindReg(r103, &d707)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d707)
		ctx.EnsureDesc(&d707)
		if d707.Loc == scm.LocImm {
			d708 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d707.Imm.Int()))))}
		} else {
			r104 := ctx.AllocReg()
			ctx.EmitMovRegReg(r104, d707.Reg)
			ctx.EmitShlRegImm8(r104, 56)
			ctx.EmitShrRegImm8(r104, 56)
			d708 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			ctx.BindReg(r104, &d708)
		}
		ctx.FreeDesc(&d707)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d706)
		ctx.EnsureDesc(&d706)
		if d706.Loc == scm.LocImm {
			d709 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d706.Imm.Int()))))}
		} else {
			r105 := ctx.AllocReg()
			ctx.EmitMovRegReg(r105, d706.Reg)
			ctx.EmitShlRegImm8(r105, 32)
			ctx.EmitShrRegImm8(r105, 32)
			d709 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			ctx.BindReg(r105, &d709)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d709)
		ctx.EnsureDesc(&d708)
		ctx.EnsureDescsTogether(&d709, &d708)
		if d709.Loc == scm.LocImm && d708.Loc == scm.LocImm {
			d710 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d709.Imm.Int() * d708.Imm.Int())}
		} else if d709.Loc == scm.LocImm {
			ctx.EnsureDesc(&d708)
			scratch := ctx.AllocRegExcept(d708.Reg)
			ctx.EmitMovRegReg(scratch, d708.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d709.Imm.Int())
			d710 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d710)
		} else if d708.Loc == scm.LocImm {
			ctx.EnsureDesc(&d709)
			scratch := ctx.AllocRegExcept(d709.Reg)
			ctx.EmitMovRegReg(scratch, d709.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d708.Imm.Int())
			d710 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d710)
		} else {
			ctx.EnsureDesc(&d709)
			ctx.SyncDesc(&d708)
			r106 := ctx.AllocRegExcept(d709.Reg, d708.Reg)
			ctx.EmitMovRegReg(r106, d709.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r106, &d708)
			d710 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			ctx.BindReg(r106, &d710)
		}
		if d710.Loc == scm.LocReg && d709.Loc == scm.LocReg && d710.Reg == d709.Reg {
			ctx.TransferReg(d709.Reg)
			d709.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d709)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d710)
		if d710.Loc == scm.LocImm {
			d711 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d710.Imm.Int() / 64)}
		} else {
			r107 := ctx.AllocRegExcept(d710.Reg)
			ctx.EmitMovRegReg(r107, d710.Reg)
			ctx.EmitShrRegImm8(r107, 6)
			d711 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d711)
		}
		if d711.Loc == scm.LocReg && d710.Loc == scm.LocReg && d711.Reg == d710.Reg {
			ctx.TransferReg(d710.Reg)
			d710.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d710)
		if d710.Loc == scm.LocImm {
			d712 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d710.Imm.Int() % 64)}
		} else {
			r108 := ctx.AllocRegExcept(d710.Reg)
			ctx.EmitMovRegReg(r108, d710.Reg)
			ctx.EmitAndRegImm32(r108, 63)
			d712 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d712)
		}
		if d712.Loc == scm.LocReg && d710.Loc == scm.LocReg && d712.Reg == d710.Reg {
			ctx.TransferReg(d710.Reg)
			d710.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d710)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d713 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r109 := ctx.AllocReg()
			r110 := ctx.AllocRegExcept(r109)
			r111 := ctx.AllocRegExcept(r109, r110)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			ctx.EmitMovRegMem(r109, thisptr.Reg, off)
			ctx.EmitMovRegMem(r110, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r111, thisptr.Reg, off+16)
			d713 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r109, Reg2: r110, Reg3: r111}
			ctx.BindReg(r109, &d713)
			ctx.BindReg(r110, &d713)
			ctx.BindReg(r111, &d713)
			ctx.BindReg(r109, &d713)
			ctx.BindReg(r110, &d713)
			ctx.BindReg(r111, &d713)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d711)
		ctx.ReclaimUntrackedRegs()
		d714 = ctx.EmitLoadScalarSliceElement(&d713, &d711, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d714)
		ctx.EnsureDesc(&d712)
		ctx.EnsureDescsTogether(&d714, &d712)
		if d714.Loc == scm.LocImm && d712.Loc == scm.LocImm {
			d715 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d714.Imm.Int()) << uint64(d712.Imm.Int())))}
		} else if d712.Loc == scm.LocImm {
			r112 := ctx.AllocRegExcept(d714.Reg)
			ctx.EmitMovRegReg(r112, d714.Reg)
			ctx.EmitShlRegImm8(r112, uint8(d712.Imm.Int()))
			d715 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			ctx.BindReg(r112, &d715)
		} else {
			shiftSrc := d714.Reg
			r113 := ctx.AllocRegExcept(d714.Reg, d712.Reg)
			ctx.EmitMovRegReg(r113, d714.Reg)
			shiftSrc = r113
			ctx.EmitShiftLeft(shiftSrc, d712.Reg, true)
			d715 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d715)
		}
		if d715.Loc == scm.LocReg && d714.Loc == scm.LocReg && d715.Reg == d714.Reg {
			ctx.TransferReg(d714.Reg)
			d714.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d714)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d711)
		ctx.EnsureDesc(&d711)
		if d711.Loc == scm.LocImm {
			d716 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d711.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d711.Reg)
			ctx.EmitMovRegReg(scratch, d711.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d716 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d716)
		}
		if d716.Loc == scm.LocReg && d711.Loc == scm.LocReg && d716.Reg == d711.Reg {
			ctx.TransferReg(d711.Reg)
			d711.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d711)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d716)
		ctx.ReclaimUntrackedRegs()
		d717 = ctx.EmitLoadScalarSliceElement(&d713, &d716, 8, scm.TagInt)
		ctx.FreeDesc(&d716)
		ctx.ReclaimUntrackedRegs()
		d718 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d712)
		ctx.EnsureDescsTogether(&d718, &d712)
		if d718.Loc == scm.LocImm && d712.Loc == scm.LocImm {
			d719 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d718.Imm.Int() - d712.Imm.Int())}
		} else if d712.Loc == scm.LocImm && d712.Imm.Int() == 0 {
			ctx.EnsureDesc(&d718)
			r114 := ctx.AllocRegExcept(d718.Reg)
			ctx.EmitMovRegReg(r114, d718.Reg)
			d719 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			ctx.BindReg(r114, &d719)
		} else if d718.Loc == scm.LocImm {
			ctx.EnsureDesc(&d712)
			scratch := ctx.AllocRegExcept(d712.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d718.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d712)
			d719 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d719)
		} else if d712.Loc == scm.LocImm {
			ctx.EnsureDesc(&d718)
			scratch := ctx.AllocRegExcept(d718.Reg)
			ctx.EmitMovRegReg(scratch, d718.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d712.Imm.Int())
			d719 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d719)
		} else {
			ctx.EnsureDesc(&d718)
			ctx.SyncDesc(&d712)
			r115 := ctx.AllocRegExcept(d718.Reg, d712.Reg)
			ctx.EmitMovRegReg(r115, d718.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r115, &d712)
			d719 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
			ctx.BindReg(r115, &d719)
		}
		if d719.Loc == scm.LocReg && d718.Loc == scm.LocReg && d719.Reg == d718.Reg {
			ctx.TransferReg(d718.Reg)
			d718.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d712)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d717)
		ctx.EnsureDesc(&d719)
		ctx.EnsureDescsTogether(&d717, &d719)
		if d717.Loc == scm.LocImm && d719.Loc == scm.LocImm {
			d720 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d717.Imm.Int()) >> uint64(d719.Imm.Int())))}
		} else if d719.Loc == scm.LocImm {
			r116 := ctx.AllocRegExcept(d717.Reg)
			ctx.EmitMovRegReg(r116, d717.Reg)
			ctx.EmitShrRegImm8(r116, uint8(d719.Imm.Int()))
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
			ctx.BindReg(r116, &d720)
		} else {
			shiftSrc := d717.Reg
			r117 := ctx.AllocRegExcept(d717.Reg, d719.Reg)
			ctx.EmitMovRegReg(r117, d717.Reg)
			shiftSrc = r117
			ctx.EmitShiftRight(shiftSrc, d719.Reg, false)
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d720)
		}
		if d720.Loc == scm.LocReg && d717.Loc == scm.LocReg && d720.Reg == d717.Reg {
			ctx.TransferReg(d717.Reg)
			d717.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d717)
		ctx.FreeDesc(&d719)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d715)
		ctx.EnsureDesc(&d720)
		if d715.Loc == scm.LocImm && d720.Loc == scm.LocImm {
			d721 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d715.Imm.Int() | d720.Imm.Int())}
		} else if d715.Loc == scm.LocImm && d715.Imm.Int() == 0 {
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d720.Reg}
			ctx.BindReg(d720.Reg, &d721)
		} else if d720.Loc == scm.LocImm && d720.Imm.Int() == 0 {
			r118 := ctx.AllocRegExcept(d715.Reg)
			ctx.EmitMovRegReg(r118, d715.Reg)
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
			ctx.BindReg(r118, &d721)
		} else if d715.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d720.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d715.Imm.Int()))
			ctx.EmitOrInt64(scratch, d720.Reg)
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d721)
		} else if d720.Loc == scm.LocImm {
			r119 := ctx.AllocRegExcept(d715.Reg)
			ctx.EmitMovRegReg(r119, d715.Reg)
			if d720.Imm.Int() >= -2147483648 && d720.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r119, int32(d720.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d720.Imm.Int()))
				ctx.EmitOrInt64(r119, ctx.ScratchReg)
			}
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			ctx.BindReg(r119, &d721)
		} else {
			r120 := ctx.AllocRegExcept(d715.Reg, d720.Reg)
			ctx.EmitMovRegReg(r120, d715.Reg)
			ctx.EmitOrInt64(r120, d720.Reg)
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d721)
		}
		if d721.Loc == scm.LocReg && d715.Loc == scm.LocReg && d721.Reg == d715.Reg {
			ctx.TransferReg(d715.Reg)
			d715.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d715)
		ctx.FreeDesc(&d720)
		ctx.ReclaimUntrackedRegs()
		d722 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d708)
		ctx.EnsureDescsTogether(&d722, &d708)
		if d722.Loc == scm.LocImm && d708.Loc == scm.LocImm {
			d723 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d722.Imm.Int() - d708.Imm.Int())}
		} else if d708.Loc == scm.LocImm && d708.Imm.Int() == 0 {
			ctx.EnsureDesc(&d722)
			r121 := ctx.AllocRegExcept(d722.Reg)
			ctx.EmitMovRegReg(r121, d722.Reg)
			d723 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
			ctx.BindReg(r121, &d723)
		} else if d722.Loc == scm.LocImm {
			ctx.EnsureDesc(&d708)
			scratch := ctx.AllocRegExcept(d708.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d722.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d708)
			d723 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d723)
		} else if d708.Loc == scm.LocImm {
			ctx.EnsureDesc(&d722)
			scratch := ctx.AllocRegExcept(d722.Reg)
			ctx.EmitMovRegReg(scratch, d722.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d708.Imm.Int())
			d723 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d723)
		} else {
			ctx.EnsureDesc(&d722)
			ctx.SyncDesc(&d708)
			r122 := ctx.AllocRegExcept(d722.Reg, d708.Reg)
			ctx.EmitMovRegReg(r122, d722.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r122, &d708)
			d723 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			ctx.BindReg(r122, &d723)
		}
		if d723.Loc == scm.LocReg && d722.Loc == scm.LocReg && d723.Reg == d722.Reg {
			ctx.TransferReg(d722.Reg)
			d722.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d708)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d721)
		ctx.EnsureDesc(&d723)
		ctx.EnsureDescsTogether(&d721, &d723)
		if d721.Loc == scm.LocImm && d723.Loc == scm.LocImm {
			d724 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d721.Imm.Int()) >> uint64(d723.Imm.Int())))}
		} else if d723.Loc == scm.LocImm {
			r123 := ctx.AllocRegExcept(d721.Reg)
			ctx.EmitMovRegReg(r123, d721.Reg)
			ctx.EmitShrRegImm8(r123, uint8(d723.Imm.Int()))
			d724 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d724)
		} else {
			shiftSrc := d721.Reg
			r124 := ctx.AllocRegExcept(d721.Reg, d723.Reg)
			ctx.EmitMovRegReg(r124, d721.Reg)
			shiftSrc = r124
			ctx.EmitShiftRight(shiftSrc, d723.Reg, false)
			d724 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d724)
		}
		if d724.Loc == scm.LocReg && d721.Loc == scm.LocReg && d724.Reg == d721.Reg {
			ctx.TransferReg(d721.Reg)
			d721.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d721)
		ctx.FreeDesc(&d723)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d724)
		ctx.EnsureDesc(&d724)
		ctx.EnsureDesc(&d724)
		if d724.Loc == scm.LocImm {
			d725 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d724.Imm.Int()))))}
		} else {
			r125 := ctx.AllocReg()
			ctx.EmitMovRegReg(r125, d724.Reg)
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d725)
		}
		ctx.FreeDesc(&d724)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d726 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 56)
			r126 := ctx.AllocReg()
			ctx.EmitMovRegMem(r126, thisptr.Reg, off)
			d726 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.BindReg(r126, &d726)
		}
		ctx.EnsureDesc(&d725)
		ctx.EnsureDesc(&d726)
		ctx.EnsureDescsTogether(&d725, &d726)
		if d725.Loc == scm.LocImm && d726.Loc == scm.LocImm {
			d727 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d725.Imm.Int() + d726.Imm.Int())}
		} else if d726.Loc == scm.LocImm && d726.Imm.Int() == 0 {
			ctx.EnsureDesc(&d725)
			r127 := ctx.AllocRegExcept(d725.Reg)
			ctx.EmitMovRegReg(r127, d725.Reg)
			d727 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d727)
		} else if d725.Loc == scm.LocImm && d725.Imm.Int() == 0 {
			ctx.EnsureDesc(&d726)
			d727 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d726.Reg}
			ctx.BindReg(d726.Reg, &d727)
		} else if d725.Loc == scm.LocImm {
			ctx.EnsureDesc(&d726)
			scratch := ctx.AllocRegExcept(d726.Reg)
			ctx.EmitMovRegReg(scratch, d726.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d725.Imm.Int())
			d727 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d727)
		} else if d726.Loc == scm.LocImm {
			ctx.EnsureDesc(&d725)
			scratch := ctx.AllocRegExcept(d725.Reg)
			ctx.EmitMovRegReg(scratch, d725.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d726.Imm.Int())
			d727 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d727)
		} else {
			ctx.EnsureDesc(&d725)
			ctx.SyncDesc(&d726)
			r128 := ctx.AllocRegExcept(d725.Reg, d726.Reg)
			ctx.EmitMovRegReg(r128, d725.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r128, &d726)
			d727 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d727)
		}
		if d727.Loc == scm.LocReg && d725.Loc == scm.LocReg && d727.Reg == d725.Reg {
			ctx.TransferReg(d725.Reg)
			d725.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d725)
		ctx.FreeDesc(&d726)
		ctx.EnsureDesc(&d8)
		d728 = d8
		_ = d728
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl21 := ctx.ReserveLabel()
		_ = lbl21
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl21)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d729 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r129 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r129, thisptr.Reg, off)
			d729 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
			ctx.BindReg(r129, &d729)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d729)
		ctx.EnsureDesc(&d729)
		if d729.Loc == scm.LocImm {
			d730 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d729.Imm.Int()))))}
		} else {
			r130 := ctx.AllocReg()
			ctx.EmitMovRegReg(r130, d729.Reg)
			ctx.EmitShlRegImm8(r130, 56)
			ctx.EmitShrRegImm8(r130, 56)
			d730 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d730)
		}
		ctx.FreeDesc(&d729)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d728)
		ctx.EnsureDesc(&d728)
		if d728.Loc == scm.LocImm {
			d731 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d728.Imm.Int()))))}
		} else {
			r131 := ctx.AllocReg()
			ctx.EmitMovRegReg(r131, d728.Reg)
			ctx.EmitShlRegImm8(r131, 32)
			ctx.EmitShrRegImm8(r131, 32)
			d731 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
			ctx.BindReg(r131, &d731)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d731)
		ctx.EnsureDesc(&d730)
		ctx.EnsureDescsTogether(&d731, &d730)
		if d731.Loc == scm.LocImm && d730.Loc == scm.LocImm {
			d732 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d731.Imm.Int() * d730.Imm.Int())}
		} else if d731.Loc == scm.LocImm {
			ctx.EnsureDesc(&d730)
			scratch := ctx.AllocRegExcept(d730.Reg)
			ctx.EmitMovRegReg(scratch, d730.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d731.Imm.Int())
			d732 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d732)
		} else if d730.Loc == scm.LocImm {
			ctx.EnsureDesc(&d731)
			scratch := ctx.AllocRegExcept(d731.Reg)
			ctx.EmitMovRegReg(scratch, d731.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d730.Imm.Int())
			d732 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d732)
		} else {
			ctx.EnsureDesc(&d731)
			ctx.SyncDesc(&d730)
			r132 := ctx.AllocRegExcept(d731.Reg, d730.Reg)
			ctx.EmitMovRegReg(r132, d731.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r132, &d730)
			d732 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d732)
		}
		if d732.Loc == scm.LocReg && d731.Loc == scm.LocReg && d732.Reg == d731.Reg {
			ctx.TransferReg(d731.Reg)
			d731.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d731)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d732)
		if d732.Loc == scm.LocImm {
			d733 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d732.Imm.Int() / 64)}
		} else {
			r133 := ctx.AllocRegExcept(d732.Reg)
			ctx.EmitMovRegReg(r133, d732.Reg)
			ctx.EmitShrRegImm8(r133, 6)
			d733 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			ctx.BindReg(r133, &d733)
		}
		if d733.Loc == scm.LocReg && d732.Loc == scm.LocReg && d733.Reg == d732.Reg {
			ctx.TransferReg(d732.Reg)
			d732.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d732)
		if d732.Loc == scm.LocImm {
			d734 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d732.Imm.Int() % 64)}
		} else {
			r134 := ctx.AllocRegExcept(d732.Reg)
			ctx.EmitMovRegReg(r134, d732.Reg)
			ctx.EmitAndRegImm32(r134, 63)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d734)
		}
		if d734.Loc == scm.LocReg && d732.Loc == scm.LocReg && d734.Reg == d732.Reg {
			ctx.TransferReg(d732.Reg)
			d732.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d732)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d735 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r135 := ctx.AllocReg()
			r136 := ctx.AllocRegExcept(r135)
			r137 := ctx.AllocRegExcept(r135, r136)
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r135, thisptr.Reg, off)
			ctx.EmitMovRegMem(r136, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r137, thisptr.Reg, off+16)
			d735 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r135, Reg2: r136, Reg3: r137}
			ctx.BindReg(r135, &d735)
			ctx.BindReg(r136, &d735)
			ctx.BindReg(r137, &d735)
			ctx.BindReg(r135, &d735)
			ctx.BindReg(r136, &d735)
			ctx.BindReg(r137, &d735)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d733)
		ctx.ReclaimUntrackedRegs()
		d736 = ctx.EmitLoadScalarSliceElement(&d735, &d733, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d736)
		ctx.EnsureDesc(&d734)
		ctx.EnsureDescsTogether(&d736, &d734)
		if d736.Loc == scm.LocImm && d734.Loc == scm.LocImm {
			d737 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d736.Imm.Int()) << uint64(d734.Imm.Int())))}
		} else if d734.Loc == scm.LocImm {
			r138 := ctx.AllocRegExcept(d736.Reg)
			ctx.EmitMovRegReg(r138, d736.Reg)
			ctx.EmitShlRegImm8(r138, uint8(d734.Imm.Int()))
			d737 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d737)
		} else {
			shiftSrc := d736.Reg
			r139 := ctx.AllocRegExcept(d736.Reg, d734.Reg)
			ctx.EmitMovRegReg(r139, d736.Reg)
			shiftSrc = r139
			ctx.EmitShiftLeft(shiftSrc, d734.Reg, true)
			d737 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d737)
		}
		if d737.Loc == scm.LocReg && d736.Loc == scm.LocReg && d737.Reg == d736.Reg {
			ctx.TransferReg(d736.Reg)
			d736.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d736)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d733)
		ctx.EnsureDesc(&d733)
		if d733.Loc == scm.LocImm {
			d738 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d733.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d733.Reg)
			ctx.EmitMovRegReg(scratch, d733.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d738 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d738)
		}
		if d738.Loc == scm.LocReg && d733.Loc == scm.LocReg && d738.Reg == d733.Reg {
			ctx.TransferReg(d733.Reg)
			d733.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d733)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d738)
		ctx.ReclaimUntrackedRegs()
		d739 = ctx.EmitLoadScalarSliceElement(&d735, &d738, 8, scm.TagInt)
		ctx.FreeDesc(&d738)
		ctx.ReclaimUntrackedRegs()
		d740 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d734)
		ctx.EnsureDescsTogether(&d740, &d734)
		if d740.Loc == scm.LocImm && d734.Loc == scm.LocImm {
			d741 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d740.Imm.Int() - d734.Imm.Int())}
		} else if d734.Loc == scm.LocImm && d734.Imm.Int() == 0 {
			ctx.EnsureDesc(&d740)
			r140 := ctx.AllocRegExcept(d740.Reg)
			ctx.EmitMovRegReg(r140, d740.Reg)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d741)
		} else if d740.Loc == scm.LocImm {
			ctx.EnsureDesc(&d734)
			scratch := ctx.AllocRegExcept(d734.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d740.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d734)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d741)
		} else if d734.Loc == scm.LocImm {
			ctx.EnsureDesc(&d740)
			scratch := ctx.AllocRegExcept(d740.Reg)
			ctx.EmitMovRegReg(scratch, d740.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d734.Imm.Int())
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d741)
		} else {
			ctx.EnsureDesc(&d740)
			ctx.SyncDesc(&d734)
			r141 := ctx.AllocRegExcept(d740.Reg, d734.Reg)
			ctx.EmitMovRegReg(r141, d740.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r141, &d734)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d741)
		}
		if d741.Loc == scm.LocReg && d740.Loc == scm.LocReg && d741.Reg == d740.Reg {
			ctx.TransferReg(d740.Reg)
			d740.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d734)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d739)
		ctx.EnsureDesc(&d741)
		ctx.EnsureDescsTogether(&d739, &d741)
		if d739.Loc == scm.LocImm && d741.Loc == scm.LocImm {
			d742 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d739.Imm.Int()) >> uint64(d741.Imm.Int())))}
		} else if d741.Loc == scm.LocImm {
			r142 := ctx.AllocRegExcept(d739.Reg)
			ctx.EmitMovRegReg(r142, d739.Reg)
			ctx.EmitShrRegImm8(r142, uint8(d741.Imm.Int()))
			d742 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d742)
		} else {
			shiftSrc := d739.Reg
			r143 := ctx.AllocRegExcept(d739.Reg, d741.Reg)
			ctx.EmitMovRegReg(r143, d739.Reg)
			shiftSrc = r143
			ctx.EmitShiftRight(shiftSrc, d741.Reg, false)
			d742 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d742)
		}
		if d742.Loc == scm.LocReg && d739.Loc == scm.LocReg && d742.Reg == d739.Reg {
			ctx.TransferReg(d739.Reg)
			d739.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d739)
		ctx.FreeDesc(&d741)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d737)
		ctx.EnsureDesc(&d742)
		if d737.Loc == scm.LocImm && d742.Loc == scm.LocImm {
			d743 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d737.Imm.Int() | d742.Imm.Int())}
		} else if d737.Loc == scm.LocImm && d737.Imm.Int() == 0 {
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d742.Reg}
			ctx.BindReg(d742.Reg, &d743)
		} else if d742.Loc == scm.LocImm && d742.Imm.Int() == 0 {
			r144 := ctx.AllocRegExcept(d737.Reg)
			ctx.EmitMovRegReg(r144, d737.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d743)
		} else if d737.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d742.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d737.Imm.Int()))
			ctx.EmitOrInt64(scratch, d742.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d743)
		} else if d742.Loc == scm.LocImm {
			r145 := ctx.AllocRegExcept(d737.Reg)
			ctx.EmitMovRegReg(r145, d737.Reg)
			if d742.Imm.Int() >= -2147483648 && d742.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r145, int32(d742.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d742.Imm.Int()))
				ctx.EmitOrInt64(r145, ctx.ScratchReg)
			}
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d743)
		} else {
			r146 := ctx.AllocRegExcept(d737.Reg, d742.Reg)
			ctx.EmitMovRegReg(r146, d737.Reg)
			ctx.EmitOrInt64(r146, d742.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			ctx.BindReg(r146, &d743)
		}
		if d743.Loc == scm.LocReg && d737.Loc == scm.LocReg && d743.Reg == d737.Reg {
			ctx.TransferReg(d737.Reg)
			d737.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d737)
		ctx.FreeDesc(&d742)
		ctx.ReclaimUntrackedRegs()
		d744 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d730)
		ctx.EnsureDescsTogether(&d744, &d730)
		if d744.Loc == scm.LocImm && d730.Loc == scm.LocImm {
			d745 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d744.Imm.Int() - d730.Imm.Int())}
		} else if d730.Loc == scm.LocImm && d730.Imm.Int() == 0 {
			ctx.EnsureDesc(&d744)
			r147 := ctx.AllocRegExcept(d744.Reg)
			ctx.EmitMovRegReg(r147, d744.Reg)
			d745 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d745)
		} else if d744.Loc == scm.LocImm {
			ctx.EnsureDesc(&d730)
			scratch := ctx.AllocRegExcept(d730.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d744.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d730)
			d745 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d745)
		} else if d730.Loc == scm.LocImm {
			ctx.EnsureDesc(&d744)
			scratch := ctx.AllocRegExcept(d744.Reg)
			ctx.EmitMovRegReg(scratch, d744.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d730.Imm.Int())
			d745 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d745)
		} else {
			ctx.EnsureDesc(&d744)
			ctx.SyncDesc(&d730)
			r148 := ctx.AllocRegExcept(d744.Reg, d730.Reg)
			ctx.EmitMovRegReg(r148, d744.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r148, &d730)
			d745 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d745)
		}
		if d745.Loc == scm.LocReg && d744.Loc == scm.LocReg && d745.Reg == d744.Reg {
			ctx.TransferReg(d744.Reg)
			d744.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d730)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d743)
		ctx.EnsureDesc(&d745)
		ctx.EnsureDescsTogether(&d743, &d745)
		if d743.Loc == scm.LocImm && d745.Loc == scm.LocImm {
			d746 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d743.Imm.Int()) >> uint64(d745.Imm.Int())))}
		} else if d745.Loc == scm.LocImm {
			r149 := ctx.AllocRegExcept(d743.Reg)
			ctx.EmitMovRegReg(r149, d743.Reg)
			ctx.EmitShrRegImm8(r149, uint8(d745.Imm.Int()))
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			ctx.BindReg(r149, &d746)
		} else {
			shiftSrc := d743.Reg
			r150 := ctx.AllocRegExcept(d743.Reg, d745.Reg)
			ctx.EmitMovRegReg(r150, d743.Reg)
			shiftSrc = r150
			ctx.EmitShiftRight(shiftSrc, d745.Reg, false)
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d746)
		}
		if d746.Loc == scm.LocReg && d743.Loc == scm.LocReg && d746.Reg == d743.Reg {
			ctx.TransferReg(d743.Reg)
			d743.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d743)
		ctx.FreeDesc(&d745)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d746)
		ctx.EnsureDesc(&d746)
		ctx.EnsureDesc(&d746)
		if d746.Loc == scm.LocImm {
			d747 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d746.Imm.Int()))))}
		} else {
			r151 := ctx.AllocReg()
			ctx.EmitMovRegReg(r151, d746.Reg)
			d747 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d747)
		}
		ctx.FreeDesc(&d746)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d748 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r152 := ctx.AllocReg()
			ctx.EmitMovRegMem(r152, thisptr.Reg, off)
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
			ctx.BindReg(r152, &d748)
		}
		ctx.EnsureDesc(&d747)
		ctx.EnsureDesc(&d748)
		ctx.EnsureDescsTogether(&d747, &d748)
		if d747.Loc == scm.LocImm && d748.Loc == scm.LocImm {
			d749 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d747.Imm.Int() + d748.Imm.Int())}
		} else if d748.Loc == scm.LocImm && d748.Imm.Int() == 0 {
			ctx.EnsureDesc(&d747)
			r153 := ctx.AllocRegExcept(d747.Reg)
			ctx.EmitMovRegReg(r153, d747.Reg)
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d749)
		} else if d747.Loc == scm.LocImm && d747.Imm.Int() == 0 {
			ctx.EnsureDesc(&d748)
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d748.Reg}
			ctx.BindReg(d748.Reg, &d749)
		} else if d747.Loc == scm.LocImm {
			ctx.EnsureDesc(&d748)
			scratch := ctx.AllocRegExcept(d748.Reg)
			ctx.EmitMovRegReg(scratch, d748.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d747.Imm.Int())
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d749)
		} else if d748.Loc == scm.LocImm {
			ctx.EnsureDesc(&d747)
			scratch := ctx.AllocRegExcept(d747.Reg)
			ctx.EmitMovRegReg(scratch, d747.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d748.Imm.Int())
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d749)
		} else {
			ctx.EnsureDesc(&d747)
			ctx.SyncDesc(&d748)
			r154 := ctx.AllocRegExcept(d747.Reg, d748.Reg)
			ctx.EmitMovRegReg(r154, d747.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r154, &d748)
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			ctx.BindReg(r154, &d749)
		}
		if d749.Loc == scm.LocReg && d747.Loc == scm.LocReg && d749.Reg == d747.Reg {
			ctx.TransferReg(d747.Reg)
			d747.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d747)
		ctx.FreeDesc(&d748)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d749)
		ctx.EnsureDescsTogether(&idxInt, &d749)
		if idxInt.Loc == scm.LocImm && d749.Loc == scm.LocImm {
			d751 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d749.Imm.Int())}
		} else if d749.Loc == scm.LocImm && d749.Imm.Int() == 0 {
			ctx.EnsureDesc(&idxInt)
			r155 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(r155, idxInt.Reg)
			d751 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d751)
		} else if idxInt.Loc == scm.LocImm {
			ctx.EnsureDesc(&d749)
			scratch := ctx.AllocRegExcept(d749.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d749)
			d751 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d751)
		} else if d749.Loc == scm.LocImm {
			ctx.EnsureDesc(&idxInt)
			scratch := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(scratch, idxInt.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d749.Imm.Int())
			d751 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d751)
		} else {
			ctx.EnsureDesc(&idxInt)
			ctx.SyncDesc(&d749)
			r156 := ctx.AllocRegExcept(idxInt.Reg, d749.Reg)
			ctx.EmitMovRegReg(r156, idxInt.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r156, &d749)
			d751 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d751)
		}
		if d751.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d751.Reg == idxInt.Reg {
			ctx.TransferReg(idxInt.Reg)
			idxInt.Loc = scm.LocNone
		}
		ctx.FreeDesc(&idxInt)
		ctx.FreeDesc(&d749)
		ctx.EnsureDesc(&d751)
		ctx.EnsureDesc(&d727)
		ctx.EnsureDescsTogether(&d751, &d727)
		if d751.Loc == scm.LocImm && d727.Loc == scm.LocImm {
			d752 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d751.Imm.Int() * d727.Imm.Int())}
		} else if d751.Loc == scm.LocImm {
			ctx.EnsureDesc(&d727)
			scratch := ctx.AllocRegExcept(d727.Reg)
			ctx.EmitMovRegReg(scratch, d727.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d751.Imm.Int())
			d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d752)
		} else if d727.Loc == scm.LocImm {
			ctx.EnsureDesc(&d751)
			scratch := ctx.AllocRegExcept(d751.Reg)
			ctx.EmitMovRegReg(scratch, d751.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d727.Imm.Int())
			d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d752)
		} else {
			ctx.EnsureDesc(&d751)
			ctx.SyncDesc(&d727)
			r157 := ctx.AllocRegExcept(d751.Reg, d727.Reg)
			ctx.EmitMovRegReg(r157, d751.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r157, &d727)
			d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d752)
		}
		if d752.Loc == scm.LocReg && d751.Loc == scm.LocReg && d752.Reg == d751.Reg {
			ctx.TransferReg(d751.Reg)
			d751.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d751)
		ctx.FreeDesc(&d727)
		ctx.EnsureDesc(&d705)
		resultTarget753 := false
		_ = resultTarget753
		ctx.EnsureDesc(&d752)
		ctx.EnsureDescsTogether(&d705, &d752)
		if d705.Loc == scm.LocImm && d752.Loc == scm.LocImm {
			d754 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d705.Imm.Int() + d752.Imm.Int())}
		} else if d752.Loc == scm.LocImm && d752.Imm.Int() == 0 {
			ctx.EnsureDesc(&d705)
			var r158 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d705.Reg {
				r158 = result.Reg2
				resultTarget753 = true
			} else {
				r158 = ctx.AllocRegExcept(d705.Reg)
			}
			ctx.EmitMovRegReg(r158, d705.Reg)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d754)
		} else if d705.Loc == scm.LocImm && d705.Imm.Int() == 0 {
			ctx.EnsureDesc(&d752)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d752.Reg}
			ctx.BindReg(d752.Reg, &d754)
		} else if d705.Loc == scm.LocImm {
			ctx.EnsureDesc(&d752)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d752.Reg {
				scratch = result.Reg2
				resultTarget753 = true
			} else {
				scratch = ctx.AllocRegExcept(d752.Reg)
			}
			ctx.EmitMovRegReg(scratch, d752.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d705.Imm.Int())
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d754)
		} else if d752.Loc == scm.LocImm {
			ctx.EnsureDesc(&d705)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d705.Reg {
				scratch = result.Reg2
				resultTarget753 = true
			} else {
				scratch = ctx.AllocRegExcept(d705.Reg)
			}
			ctx.EmitMovRegReg(scratch, d705.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d752.Imm.Int())
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d754)
		} else {
			ctx.EnsureDesc(&d705)
			ctx.SyncDesc(&d752)
			var r159 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d705.Reg && result.Reg2 != d752.Reg {
				r159 = result.Reg2
				resultTarget753 = true
			} else {
				r159 = ctx.AllocRegExcept(d705.Reg, d752.Reg)
			}
			ctx.EmitMovRegReg(r159, d705.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r159, &d752)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d754)
		}
		if d754.Loc == scm.LocReg && d705.Loc == scm.LocReg && d754.Reg == d705.Reg {
			ctx.TransferReg(d705.Reg)
			d705.Loc = scm.LocNone
		}
		if resultTarget753 && d754.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d705)
		ctx.FreeDesc(&d752)
		ctx.EnsureDesc(&d754)
		d755 = result
		ctx.EnsureDesc(&d754)
		ctx.EmitMakeInt(d755, d754)
		if d754.Loc == scm.LocReg {
			ctx.FreeReg(d754.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[13].Render = func() scm.JITValueDesc {
		if bbs[13].Rendered {
			ctx.EmitJmp(lbl14)
			return result
		}
		bbs[13].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_13 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl14)
		ctx.ResolveFixups()
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d756 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 88)
			r160 := ctx.AllocReg()
			ctx.EmitMovRegMem(r160, thisptr.Reg, off)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
			ctx.BindReg(r160, &d756)
		}
		ctx.EnsureDesc(&d105)
		ctx.EnsureDesc(&d756)
		ctx.EnsureDescsTogether(&d105, &d756)
		if d105.Loc == scm.LocImm && d756.Loc == scm.LocImm {
			d757 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d105.Imm.Int()) == uint64(d756.Imm.Int()))}
		} else if d756.Loc == scm.LocImm {
			r161 := ctx.AllocRegExcept(d105.Reg)
			if d756.Imm.Int() >= -2147483648 && d756.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d105.Reg, int32(d756.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d756.Imm.Int()))
				ctx.EmitCmpInt64(d105.Reg, ctx.ScratchReg)
			}
			d757 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r161, Condition: scm.CondEqual}
			ctx.BindReg(r161, &d757)
		} else if d105.Loc == scm.LocImm {
			r162 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d105.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d756.Reg)
			d757 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r162, Condition: scm.CondEqual}
			ctx.BindReg(r162, &d757)
		} else {
			r163 := ctx.AllocRegExcept(d105.Reg)
			ctx.EmitCmpInt64(d105.Reg, d756.Reg)
			d757 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r163, Condition: scm.CondEqual}
			ctx.BindReg(r163, &d757)
		}
		ctx.FreeDesc(&d756)
		d758 = d757
		ctx.EnsureDesc(&d758)
		if d758.Loc != scm.LocImm && d758.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d758.Loc == scm.LocImm {
			if d758.Imm.Bool() {
				return bbs[11].Render()
			}
			return bbs[12].Render()
		}
		ctx.EmitJump(d758.Condition, lbl12)
		if bbs[12].Rendered {
			ctx.EmitJmp(lbl13)
		}
		ctx.FreeDesc(&d757)
		ctx.FlushRegisterMoves()
		if !bbs[12].Rendered {
			snap759 := d5
			snap760 := d6
			snap761 := d7
			snap762 := d8
			snap763 := d9
			snap764 := d10
			snap765 := d11
			snap766 := d12
			snap767 := d13
			snap768 := d14
			snap769 := d15
			snap770 := d16
			snap771 := d17
			snap772 := d18
			snap773 := d19
			snap774 := d20
			snap775 := d21
			snap776 := d22
			snap777 := d23
			snap778 := d24
			snap779 := d25
			snap780 := d26
			snap781 := d27
			snap782 := d28
			snap783 := d29
			snap784 := d30
			snap785 := d31
			snap786 := d32
			snap787 := d33
			snap788 := d34
			snap789 := d35
			snap790 := d36
			snap791 := d37
			snap792 := d38
			snap793 := d39
			snap794 := d40
			snap795 := d41
			snap796 := d42
			snap797 := d43
			snap798 := d44
			snap799 := d86
			snap800 := d87
			snap801 := d88
			snap802 := d89
			snap803 := d90
			snap804 := d91
			snap805 := d92
			snap806 := d93
			snap807 := d94
			snap808 := d95
			snap809 := d96
			snap810 := d97
			snap811 := d98
			snap812 := d99
			snap813 := d100
			snap814 := d101
			snap815 := d102
			snap816 := d103
			snap817 := d104
			snap818 := d105
			snap819 := d106
			snap820 := d107
			snap821 := d171
			snap822 := d172
			snap823 := d173
			snap824 := d174
			snap825 := d175
			snap826 := d176
			snap827 := d177
			snap828 := d178
			snap829 := d250
			snap830 := d251
			snap831 := d325
			snap832 := d326
			snap833 := d327
			snap834 := d328
			snap835 := d329
			snap836 := d330
			snap837 := d331
			snap838 := d332
			snap839 := d333
			snap840 := d334
			snap841 := d335
			snap842 := d336
			snap843 := d337
			snap844 := d338
			snap845 := d339
			snap846 := d340
			snap847 := d341
			snap848 := d342
			snap849 := d343
			snap850 := d344
			snap851 := d345
			snap852 := d346
			snap853 := d347
			snap854 := d348
			snap855 := d349
			snap856 := d350
			snap857 := d351
			snap858 := d352
			snap859 := d353
			snap860 := d354
			snap861 := d458
			snap862 := d459
			snap863 := d460
			snap864 := d461
			snap865 := d462
			snap866 := d463
			snap867 := d464
			snap868 := d575
			snap869 := d576
			snap870 := d689
			snap871 := d690
			snap872 := d691
			snap873 := d692
			snap874 := d693
			snap875 := d694
			snap876 := d695
			snap877 := d696
			snap878 := d697
			snap879 := d698
			snap880 := d699
			snap881 := d700
			snap882 := d701
			snap883 := d702
			snap884 := d703
			snap885 := d704
			snap886 := d705
			snap887 := d706
			snap888 := d707
			snap889 := d708
			snap890 := d709
			snap891 := d710
			snap892 := d711
			snap893 := d712
			snap894 := d713
			snap895 := d714
			snap896 := d715
			snap897 := d716
			snap898 := d717
			snap899 := d718
			snap900 := d719
			snap901 := d720
			snap902 := d721
			snap903 := d722
			snap904 := d723
			snap905 := d724
			snap906 := d725
			snap907 := d726
			snap908 := d727
			snap909 := d728
			snap910 := d729
			snap911 := d730
			snap912 := d731
			snap913 := d732
			snap914 := d733
			snap915 := d734
			snap916 := d735
			snap917 := d736
			snap918 := d737
			snap919 := d738
			snap920 := d739
			snap921 := d740
			snap922 := d741
			snap923 := d742
			snap924 := d743
			snap925 := d744
			snap926 := d745
			snap927 := d746
			snap928 := d747
			snap929 := d748
			snap930 := d749
			snap931 := d750
			snap932 := d751
			snap933 := d752
			snap934 := d754
			snap935 := d755
			snap936 := d756
			snap937 := d757
			snap938 := d758
			alloc939 := ctx.SnapshotAllocState()
			bbs[12].Render()
			ctx.RestoreAllocState(alloc939)
			d5 = snap759
			d6 = snap760
			d7 = snap761
			d8 = snap762
			d9 = snap763
			d10 = snap764
			d11 = snap765
			d12 = snap766
			d13 = snap767
			d14 = snap768
			d15 = snap769
			d16 = snap770
			d17 = snap771
			d18 = snap772
			d19 = snap773
			d20 = snap774
			d21 = snap775
			d22 = snap776
			d23 = snap777
			d24 = snap778
			d25 = snap779
			d26 = snap780
			d27 = snap781
			d28 = snap782
			d29 = snap783
			d30 = snap784
			d31 = snap785
			d32 = snap786
			d33 = snap787
			d34 = snap788
			d35 = snap789
			d36 = snap790
			d37 = snap791
			d38 = snap792
			d39 = snap793
			d40 = snap794
			d41 = snap795
			d42 = snap796
			d43 = snap797
			d44 = snap798
			d86 = snap799
			d87 = snap800
			d88 = snap801
			d89 = snap802
			d90 = snap803
			d91 = snap804
			d92 = snap805
			d93 = snap806
			d94 = snap807
			d95 = snap808
			d96 = snap809
			d97 = snap810
			d98 = snap811
			d99 = snap812
			d100 = snap813
			d101 = snap814
			d102 = snap815
			d103 = snap816
			d104 = snap817
			d105 = snap818
			d106 = snap819
			d107 = snap820
			d171 = snap821
			d172 = snap822
			d173 = snap823
			d174 = snap824
			d175 = snap825
			d176 = snap826
			d177 = snap827
			d178 = snap828
			d250 = snap829
			d251 = snap830
			d325 = snap831
			d326 = snap832
			d327 = snap833
			d328 = snap834
			d329 = snap835
			d330 = snap836
			d331 = snap837
			d332 = snap838
			d333 = snap839
			d334 = snap840
			d335 = snap841
			d336 = snap842
			d337 = snap843
			d338 = snap844
			d339 = snap845
			d340 = snap846
			d341 = snap847
			d342 = snap848
			d343 = snap849
			d344 = snap850
			d345 = snap851
			d346 = snap852
			d347 = snap853
			d348 = snap854
			d349 = snap855
			d350 = snap856
			d351 = snap857
			d352 = snap858
			d353 = snap859
			d354 = snap860
			d458 = snap861
			d459 = snap862
			d460 = snap863
			d461 = snap864
			d462 = snap865
			d463 = snap866
			d464 = snap867
			d575 = snap868
			d576 = snap869
			d689 = snap870
			d690 = snap871
			d691 = snap872
			d692 = snap873
			d693 = snap874
			d694 = snap875
			d695 = snap876
			d696 = snap877
			d697 = snap878
			d698 = snap879
			d699 = snap880
			d700 = snap881
			d701 = snap882
			d702 = snap883
			d703 = snap884
			d704 = snap885
			d705 = snap886
			d706 = snap887
			d707 = snap888
			d708 = snap889
			d709 = snap890
			d710 = snap891
			d711 = snap892
			d712 = snap893
			d713 = snap894
			d714 = snap895
			d715 = snap896
			d716 = snap897
			d717 = snap898
			d718 = snap899
			d719 = snap900
			d720 = snap901
			d721 = snap902
			d722 = snap903
			d723 = snap904
			d724 = snap905
			d725 = snap906
			d726 = snap907
			d727 = snap908
			d728 = snap909
			d729 = snap910
			d730 = snap911
			d731 = snap912
			d732 = snap913
			d733 = snap914
			d734 = snap915
			d735 = snap916
			d736 = snap917
			d737 = snap918
			d738 = snap919
			d739 = snap920
			d740 = snap921
			d741 = snap922
			d742 = snap923
			d743 = snap924
			d744 = snap925
			d745 = snap926
			d746 = snap927
			d747 = snap928
			d748 = snap929
			d749 = snap930
			d750 = snap931
			d751 = snap932
			d752 = snap933
			d754 = snap934
			d755 = snap935
			d756 = snap936
			d757 = snap937
			d758 = snap938
		}
		if !bbs[11].Rendered {
			return bbs[11].Render()
		}
		return result
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

func (s *StorageSeq) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(11))                // 11 = StorageSeq
	binary.Write(f, binary.LittleEndian, uint8(storageSeqVersion)) // version byte (was '1' in legacy)
	var pad [6]byte
	f.Write(pad[:]) // remaining alignment padding (was "234567")
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(s.seqCount))
	s.recordId.Serialize(f)
	s.start.Serialize(f)
	s.stride.Serialize(f)
}

func (s *StorageSeq) Deserialize(f io.Reader) uint {
	var version uint8
	binary.Read(f, binary.LittleEndian, &version)
	var pad [6]byte
	f.Read(pad[:])
	switch version {
	case 0, '1': // '1'=49: legacy pre-versioning dummy byte; treat as v0
		return s.deserializeSeqV0(f)
	default:
		panic(fmt.Sprintf("StorageSeq: unknown version %d", version))
	}
}

func (s *StorageSeq) deserializeSeqV0(f io.Reader) uint {
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.count = uint(l)
	var sc uint64
	binary.Read(f, binary.LittleEndian, &sc)
	s.seqCount = uint32(sc)
	s.recordId.DeserializeEx(f, true)
	s.start.DeserializeEx(f, true)
	s.stride.DeserializeEx(f, true)
	return uint(l)
}

func (s *StorageSeq) GetCachedReader() ColumnReader { return s.storageJITFunctions.reader(s) }

func (s *StorageSeq) GetValue(i uint32) scm.Scmer {
	// bisect to the correct index where to find (lowest idx to find our sequence)
	pivot := uint32(s.lastValue.Load()) // atomic pivot cache for concurrent access
	min := uint32(0)
	max := s.seqCount - 1
	for {
		recid := int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
			max = pivot - 1
			pivot--
		} else {
			min = pivot
			pivot++
		}
		if min == max {
			break // we found the sequence for i
		}

		// also read the next neighbour (we are in the cache line anyway and we achieve O(1) in case the same sequence is read again!)
		recid = int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
			max = pivot - 1
		} else {
			min = pivot
		}
		if min == max {
			break // we found the sequence for i
		}
		pivot = (min + max) / 2
	}

	// remember match for next time
	s.lastValue.Store(int64(min))

	var value, stride int64
	startRaw := s.start.GetValueUInt(min)
	if s.start.hasNull && startRaw == s.start.null {
		return scm.NewNil()
	}
	value = int64(startRaw) + s.start.offset
	stride = int64(s.stride.GetValueUInt(min)) + s.stride.offset
	recid := int64(s.recordId.GetValueUInt(min)) + s.recordId.offset
	return scm.NewInt(value + (int64(i)-recid)*stride)

}

// findSegment does the same bisection as GetValue but as a plain local
// search that never touches the shared s.lastValue atomic pivot cache.
// GetValue's cache is a single field on the struct, so concurrent goroutines
// doing bulk sequential reads over the same column would otherwise thrash
// each other's cached pivot; the bulk paths below seed their own local walk
// once and then advance it purely with local state.
func (s *StorageSeq) findSegment(i uint32) uint32 {
	var min, max uint32 = 0, s.seqCount - 1
	for min < max {
		pivot := (min + max + 1) / 2
		recid := int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if uint32(recid) <= i {
			min = pivot
		} else {
			max = pivot - 1
		}
	}
	return min
}

// segmentAt reads the (recordId, isNil, start, stride) tuple for segment
// seg. Called once per segment touched, not once per row.
func (s *StorageSeq) segmentAt(seg uint32) (recordId int64, isNil bool, start int64, stride int64) {
	recordId = int64(s.recordId.GetValueUInt(seg)) + s.recordId.offset
	startRaw := s.start.GetValueUInt(seg)
	if s.start.hasNull && startRaw == s.start.null {
		isNil = true
		return
	}
	start = int64(startRaw) + s.start.offset
	stride = int64(s.stride.GetValueUInt(seg)) + s.stride.offset
	return
}

func (s *StorageSeq) segmentEnd(seg uint32) int64 {
	if seg+1 < s.seqCount {
		return int64(s.recordId.GetValueUInt(seg+1)) + s.recordId.offset
	}
	return int64(s.count)
}

// GetValueRange reads count consecutive rows starting at recid. It seeds the
// segment cursor with one local binary search and then walks forward: each
// arithmetic-sequence segment is read as start+delta*stride incrementally
// (a running add, no per-row multiply or search), and a nil segment fills
// its whole span directly.
//
//jitgen:control-flow-stable recid count target/1 stride
func (s *StorageSeq) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	seg := s.findSegment(recid)
	segRecordId, isNil, segStart, segStride := s.segmentAt(seg)
	nextRecordId := s.segmentEnd(seg)
	curVal := segStart + (int64(recid)-segRecordId)*segStride

	idx := 0
	for k := uint32(0); k < count; k++ {
		i := int64(recid) + int64(k)
		if i >= nextRecordId {
			seg++
			segRecordId, isNil, segStart, segStride = s.segmentAt(seg)
			nextRecordId = s.segmentEnd(seg)
			curVal = segStart + (i-segRecordId)*segStride
		}
		if isNil {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(curVal)
			curVal += segStride
		}
		idx += stride
	}
}

// GetValueMulti gathers arbitrary recids. When the batch is ascending (the
// common case for an index-probe or range-scan batch), it walks the segment
// cursor forward exactly like GetValueRange, recomputing the value with one
// multiply per row (deltas between requested recids aren't necessarily 1)
// but still only touching each crossed segment's recordId/start/stride once.
// A genuinely unordered batch falls back to a fresh local findSegment per
// row — still O(log seqCount) per row like GetValue, but without the shared
// atomic pivot-cache contention.
//
//jitgen:control-flow-stable recids/2 target/1 stride
func (s *StorageSeq) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
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

	idx := 0
	if ascending {
		seg := s.findSegment(recids[0])
		segRecordId, isNil, segStart, segStride := s.segmentAt(seg)
		nextRecordId := s.segmentEnd(seg)
		// Keep one explicit induction variable across the nested segment walk;
		// the range form introduces a second copied-element phi with no benefit.
		for k := 0; k < n; k++ {
			recid := recids[k]
			i := int64(recid)
			for i >= nextRecordId {
				seg++
				segRecordId, isNil, segStart, segStride = s.segmentAt(seg)
				nextRecordId = s.segmentEnd(seg)
			}
			if isNil {
				target[idx] = scm.NewNil()
			} else {
				target[idx] = scm.NewInt(segStart + (i-segRecordId)*segStride)
			}
			idx += stride
		}
		return
	}

	for k := 0; k < n; k++ {
		recid := recids[k]
		seg := s.findSegment(recid)
		segRecordId, isNil, segStart, segStride := s.segmentAt(seg)
		if isNil {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(segStart + (int64(recid)-segRecordId)*segStride)
		}
		idx += stride
	}
}

func (s *StorageSeq) prepare() {
	// set up scan
	s.recordId.prepare()
	s.start.prepare()
	s.stride.prepare()
}
func (s *StorageSeq) scan(i uint32, value scm.Scmer) {
	if value.IsNil() {
		// nil (stride is 0)
		if i == 0 {
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.scan(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.scan(s.seqCount-1, scm.NewNil())
			s.stride.scan(s.seqCount-1, scm.NewInt(0))
		} else if s.lastValueNil {
			// sequence stays the same
		} else {
			// start nil
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.scan(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.scan(s.seqCount-1, scm.NewNil())
			s.stride.scan(s.seqCount-1, scm.NewInt(0))
		}
	} else {
		// integer
		v := value.Int()
		// A value after NULL must start a fresh numeric segment.
		if !s.lastValueNil && s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.scan(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if !s.lastValueNil && i != 0 && v == s.lastValue.Load()+s.lastStride {
			// sequence stays the same
			s.lastValue.Store(v)
		} else {
			// restart with new sequence
			s.seqCount = s.seqCount + 1
			s.lastValue.Store(v)
			s.lastValueFirst = true
			s.lastValueNil = false
			s.recordId.scan(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.scan(s.seqCount-1, value)
		}
	}
}
func (s *StorageSeq) init(i uint32) {
	s.recordId.init(s.seqCount)
	s.start.init(s.seqCount)
	s.stride.init(s.seqCount)
	s.lastValue.Store(0)
	s.lastStride = 0
	s.lastValueNil = false
	s.lastValueFirst = false
	s.count = uint(i)
	s.seqCount = 0
}
func (s *StorageSeq) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		// nil (stride is 0)
		if i == 0 {
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.build(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.build(s.seqCount-1, scm.NewNil())
			s.stride.build(s.seqCount-1, scm.NewInt(0))
		} else if s.lastValueNil {
			// sequence stays the same
		} else {
			// start nil
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.build(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.build(s.seqCount-1, scm.NewNil())
			s.stride.build(s.seqCount-1, scm.NewInt(0))
		}
	} else {
		// integer
		v := value.Int()
		// A value after NULL must start a fresh numeric segment.
		if !s.lastValueNil && s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.build(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if !s.lastValueNil && i != 0 && v == s.lastValue.Load()+s.lastStride {
			// sequence stays the same
			s.lastValue.Store(v)
		} else {
			// restart with new sequence
			s.seqCount = s.seqCount + 1
			s.lastValue.Store(v)
			s.lastValueFirst = true
			s.lastValueNil = false
			s.recordId.build(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.build(s.seqCount-1, value)
		}
	}
}
func (s *StorageSeq) finish() {
	s.recordId.finish()
	s.start.finish()
	s.stride.finish()
	s.storageJITFunctions.finish(s)

	s.lastValue.Store(int64(s.seqCount / 2)) // initialize pivot cache

	/* debug output of the sequence:
	for i := uint(0); i < s.seqCount; i++ {
		fmt.Println(s.recordId.GetValue(i),":",s.start.GetValue(i),":",s.stride.GetValue(i))
	}*/
}
func (s *StorageSeq) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

func (s *StorageSeq) DistinctCount() uint { return uint(s.count) }
