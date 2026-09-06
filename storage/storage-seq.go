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
	// data
	recordId,
	start,
	stride StorageInt
	count    uint   // number of values
	seqCount uint32 // number of sequences

	// analysis (lastValue also used as atomic pivot cache for concurrent GetValue)
	lastValue      atomic.Int64
	lastStride     int64
	lastValueNil   bool
	lastValueFirst bool
}

func (s *StorageSeq) ComputeSize() uint {
	return s.recordId.ComputeSize() + s.start.ComputeSize() + s.stride.ComputeSize() + 8*8
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

func (s *StorageSeq) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
	var d59 scm.JITValueDesc
	_ = d59
	var d60 scm.JITValueDesc
	_ = d60
	var d61 scm.JITValueDesc
	_ = d61
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
	var d125 scm.JITValueDesc
	_ = d125
	var d126 scm.JITValueDesc
	_ = d126
	var d127 scm.JITValueDesc
	_ = d127
	var d128 scm.JITValueDesc
	_ = d128
	var d129 scm.JITValueDesc
	_ = d129
	var d130 scm.JITValueDesc
	_ = d130
	var d131 scm.JITValueDesc
	_ = d131
	var d132 scm.JITValueDesc
	_ = d132
	var d133 scm.JITValueDesc
	_ = d133
	var d134 scm.JITValueDesc
	_ = d134
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
	var d150 scm.JITValueDesc
	_ = d150
	var d238 scm.JITValueDesc
	_ = d238
	var d239 scm.JITValueDesc
	_ = d239
	var d240 scm.JITValueDesc
	_ = d240
	var d241 scm.JITValueDesc
	_ = d241
	var d243 scm.JITValueDesc
	_ = d243
	var d244 scm.JITValueDesc
	_ = d244
	var d245 scm.JITValueDesc
	_ = d245
	var d246 scm.JITValueDesc
	_ = d246
	var d247 scm.JITValueDesc
	_ = d247
	var d248 scm.JITValueDesc
	_ = d248
	var d249 scm.JITValueDesc
	_ = d249
	var d250 scm.JITValueDesc
	_ = d250
	var d252 scm.JITValueDesc
	_ = d252
	var d254 scm.JITValueDesc
	_ = d254
	var d255 scm.JITValueDesc
	_ = d255
	var d256 scm.JITValueDesc
	_ = d256
	var d257 scm.JITValueDesc
	_ = d257
	var d258 scm.JITValueDesc
	_ = d258
	var d261 scm.JITValueDesc
	_ = d261
	var d366 scm.JITValueDesc
	_ = d366
	var d367 scm.JITValueDesc
	_ = d367
	var d368 scm.JITValueDesc
	_ = d368
	var d369 scm.JITValueDesc
	_ = d369
	var d370 scm.JITValueDesc
	_ = d370
	var d372 scm.JITValueDesc
	_ = d372
	var d373 scm.JITValueDesc
	_ = d373
	var d374 scm.JITValueDesc
	_ = d374
	var d375 scm.JITValueDesc
	_ = d375
	var d376 scm.JITValueDesc
	_ = d376
	var d377 scm.JITValueDesc
	_ = d377
	var d378 scm.JITValueDesc
	_ = d378
	var d379 scm.JITValueDesc
	_ = d379
	var d380 scm.JITValueDesc
	_ = d380
	var d381 scm.JITValueDesc
	_ = d381
	var d382 scm.JITValueDesc
	_ = d382
	var d383 scm.JITValueDesc
	_ = d383
	var d384 scm.JITValueDesc
	_ = d384
	var d385 scm.JITValueDesc
	_ = d385
	var d386 scm.JITValueDesc
	_ = d386
	var d387 scm.JITValueDesc
	_ = d387
	var d388 scm.JITValueDesc
	_ = d388
	var d389 scm.JITValueDesc
	_ = d389
	var d390 scm.JITValueDesc
	_ = d390
	var d391 scm.JITValueDesc
	_ = d391
	var d392 scm.JITValueDesc
	_ = d392
	var d393 scm.JITValueDesc
	_ = d393
	var d394 scm.JITValueDesc
	_ = d394
	var d395 scm.JITValueDesc
	_ = d395
	var d396 scm.JITValueDesc
	_ = d396
	var d397 scm.JITValueDesc
	_ = d397
	var d398 scm.JITValueDesc
	_ = d398
	var d399 scm.JITValueDesc
	_ = d399
	var d400 scm.JITValueDesc
	_ = d400
	var d543 scm.JITValueDesc
	_ = d543
	var d544 scm.JITValueDesc
	_ = d544
	var d545 scm.JITValueDesc
	_ = d545
	var d547 scm.JITValueDesc
	_ = d547
	var d548 scm.JITValueDesc
	_ = d548
	var d549 scm.JITValueDesc
	_ = d549
	var d550 scm.JITValueDesc
	_ = d550
	var d551 scm.JITValueDesc
	_ = d551
	var d552 scm.JITValueDesc
	_ = d552
	var d553 scm.JITValueDesc
	_ = d553
	var d555 scm.JITValueDesc
	_ = d555
	var d557 scm.JITValueDesc
	_ = d557
	var d558 scm.JITValueDesc
	_ = d558
	var d559 scm.JITValueDesc
	_ = d559
	var d560 scm.JITValueDesc
	_ = d560
	var d563 scm.JITValueDesc
	_ = d563
	var d718 scm.JITValueDesc
	_ = d718
	var d719 scm.JITValueDesc
	_ = d719
	var d720 scm.JITValueDesc
	_ = d720
	var d721 scm.JITValueDesc
	_ = d721
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
	var d753 scm.JITValueDesc
	_ = d753
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
	var d759 scm.JITValueDesc
	_ = d759
	var d760 scm.JITValueDesc
	_ = d760
	var d761 scm.JITValueDesc
	_ = d761
	var d762 scm.JITValueDesc
	_ = d762
	var d763 scm.JITValueDesc
	_ = d763
	var d764 scm.JITValueDesc
	_ = d764
	var d765 scm.JITValueDesc
	_ = d765
	var d766 scm.JITValueDesc
	_ = d766
	var d767 scm.JITValueDesc
	_ = d767
	var d768 scm.JITValueDesc
	_ = d768
	var d769 scm.JITValueDesc
	_ = d769
	var d770 scm.JITValueDesc
	_ = d770
	var d771 scm.JITValueDesc
	_ = d771
	var d772 scm.JITValueDesc
	_ = d772
	var d773 scm.JITValueDesc
	_ = d773
	var d774 scm.JITValueDesc
	_ = d774
	var d775 scm.JITValueDesc
	_ = d775
	var d776 scm.JITValueDesc
	_ = d776
	var d777 scm.JITValueDesc
	_ = d777
	var d778 scm.JITValueDesc
	_ = d778
	var d779 scm.JITValueDesc
	_ = d779
	var d780 scm.JITValueDesc
	_ = d780
	var d781 scm.JITValueDesc
	_ = d781
	var d782 scm.JITValueDesc
	_ = d782
	var d783 scm.JITValueDesc
	_ = d783
	var d784 scm.JITValueDesc
	_ = d784
	var d785 scm.JITValueDesc
	_ = d785
	var d786 scm.JITValueDesc
	_ = d786
	var d787 scm.JITValueDesc
	_ = d787
	var d788 scm.JITValueDesc
	_ = d788
	var d789 scm.JITValueDesc
	_ = d789
	var d790 scm.JITValueDesc
	_ = d790
	var d791 scm.JITValueDesc
	_ = d791
	var d792 scm.JITValueDesc
	_ = d792
	var d793 scm.JITValueDesc
	_ = d793
	var d794 scm.JITValueDesc
	_ = d794
	var d795 scm.JITValueDesc
	_ = d795
	var d796 scm.JITValueDesc
	_ = d796
	var d797 scm.JITValueDesc
	_ = d797
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
	phiBase0 := ctx.AllocStack(int32(144))
	var bbs [14]scm.BBDescriptor
	bbs[1].PhiBase = int32(phiBase0) + int32(0)
	bbs[1].PhiCount = uint16(3)
	bbs[2].PhiBase = int32(phiBase0) + int32(48)
	bbs[2].PhiCount = uint16(1)
	bbs[4].PhiBase = int32(phiBase0) + int32(64)
	bbs[4].PhiCount = uint16(3)
	bbs[8].PhiBase = int32(phiBase0) + int32(112)
	bbs[8].PhiCount = uint16(2)
	registerHomes1 := ctx.AllocRegisterHomes(scm.JITRegisterPlan{Slots: [16]scm.JITRegisterSlot{{Color: 0, Width: 1, Cost: 32}, {Color: 1, Width: 1, Cost: 12}, {Color: 2, Width: 1, Cost: 12}}, Count: 3})
	defer ctx.ReleaseRegisterHomes(registerHomes1)
	var r0 scm.Reg
	phiHomeOK2 := registerHomes1.Available&(uint16(1)<<0) == uint16(1)<<0
	if phiHomeOK2 {
		r0 = registerHomes1.Registers[0]
	}
	var r1 scm.Reg
	phiHomeOK3 := registerHomes1.Available&(uint16(1)<<2) == uint16(1)<<2
	if phiHomeOK3 {
		r1 = registerHomes1.Registers[2]
	}
	var r2 scm.Reg
	phiHomeOK4 := registerHomes1.Available&(uint16(1)<<1) == uint16(1)<<1
	if phiHomeOK4 {
		r2 = registerHomes1.Registers[1]
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
	r3 := ctx.AllocReg()
	r4 := ctx.AllocRegExcept(r3)
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
			d13 = ps.OverlayValues[13]
		}
		ctx.ReclaimUntrackedRegs()
		r5 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
			ctx.EmitMovRegMem64(r5, fieldAddr)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			ctx.EmitMovRegMem(r5, thisptr.Reg, off)
		}
		d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
		ctx.BindReg(r5, &d14)
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d14)
		var d15 scm.JITValueDesc
		if d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d14.Imm.Int()))))}
		} else {
			r6 := ctx.AllocReg()
			ctx.EmitMovRegReg(r6, d14.Reg)
			ctx.EmitShlRegImm8(r6, 32)
			ctx.EmitShrRegImm8(r6, 32)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d15)
		}
		ctx.StabilizeDescForControlFlow(&d15)
		ctx.FreeDesc(&d14)
		var d16 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
			r7 := ctx.AllocReg()
			ctx.EmitMovRegMem32(r7, fieldAddr)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			ctx.BindReg(r7, &d16)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
			r8 := ctx.AllocReg()
			ctx.EmitMovRegMemL(r8, thisptr.Reg, off)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			ctx.BindReg(r8, &d16)
		}
		ctx.EnsureDesc(&d16)
		ctx.EnsureDesc(&d16)
		var d17 scm.JITValueDesc
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
			ctx.EmitSubRegImm32(scratch, int32(1))
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		}
		if d17.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: d17.Type, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d17.Reg, 32)
			ctx.EmitShrRegImm8(d17.Reg, 32)
		}
		if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
			ctx.TransferReg(d16.Reg)
			d16.Loc = scm.LocNone
		}
		if ps.General {
			ctx.SyncDesc(&d15)
			if d15.Loc == scm.LocReg {
				ctx.ProtectReg(d15.Reg)
			} else if d15.Loc == scm.LocRegPair {
				ctx.ProtectReg(d15.Reg)
				ctx.ProtectReg(d15.Reg2)
			}
			ctx.SyncDesc(&d17)
			if d17.Loc == scm.LocReg {
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
			d19 = d18
			if d19.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: d19.Type, Imm: scm.NewInt(int64(uint64(d19.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d19.Reg, 32)
				ctx.EmitShrRegImm8(d19.Reg, 32)
			}
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d19)
			} else {
				ctx.EmitStoreToStack(d19, int32(bbs[1].PhiBase)+int32(0))
			}
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)})
			} else {
				ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
			}
			d20 = d17
			if d20.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d20)
			d21 = d20
			if d21.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: d21.Type, Imm: scm.NewInt(int64(uint64(d21.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d21.Reg, 32)
				ctx.EmitShrRegImm8(d21.Reg, 32)
			}
			if phiHomeOK4 {
				ctx.EmitMovToReg(r2, d21)
			} else {
				ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(32))
			}
			if d15.Loc == scm.LocReg {
				ctx.UnprotectReg(d15.Reg)
			} else if d15.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d15.Reg)
				ctx.UnprotectReg(d15.Reg2)
			}
			if d17.Loc == scm.LocReg {
				ctx.UnprotectReg(d17.Reg)
			} else if d17.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d17.Reg)
				ctx.UnprotectReg(d17.Reg2)
			}
		}
		ps22 := scm.PhiState{General: ps.General}
		ps22.OverlayValues = make([]scm.JITValueDesc, 22)
		ps22.OverlayValues[5] = d5
		ps22.OverlayValues[6] = d6
		ps22.OverlayValues[7] = d7
		ps22.OverlayValues[8] = d8
		ps22.OverlayValues[9] = d9
		ps22.OverlayValues[10] = d10
		ps22.OverlayValues[11] = d11
		ps22.OverlayValues[12] = d12
		ps22.OverlayValues[13] = d13
		ps22.OverlayValues[14] = d14
		ps22.OverlayValues[15] = d15
		ps22.OverlayValues[16] = d16
		ps22.OverlayValues[17] = d17
		ps22.OverlayValues[18] = d18
		ps22.OverlayValues[19] = d19
		ps22.OverlayValues[20] = d20
		ps22.OverlayValues[21] = d21
		ps22.PhiValues = make([]scm.JITValueDesc, 3)
		d23 = d15
		ps22.PhiValues[0] = d23
		d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps22.PhiValues[1] = d24
		d25 = d17
		ps22.PhiValues[2] = d25
		if ps22.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps22)
		return result
	}
	bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d26 := ps.PhiValues[0]
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d26)
				} else {
					ctx.EmitStoreToStack(d26, int32(bbs[1].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d27 := ps.PhiValues[1]
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d27)
				} else {
					ctx.EmitStoreToStack(d27, int32(bbs[1].PhiBase)+int32(16))
				}
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d28 := ps.PhiValues[2]
				if phiHomeOK4 {
					ctx.EmitMovToReg(r2, d28)
				} else {
					ctx.EmitStoreToStack(d28, int32(bbs[1].PhiBase)+int32(32))
				}
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d5 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d6 = ps.PhiValues[1]
		}
		if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
			d7 = ps.PhiValues[2]
		}
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
		d29 = d5
		_ = d29
		ctx.StabilizeDescForControlFlow(&d29)
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
		ctx.EnsureDesc(&d29)
		ctx.EnsureDesc(&d29)
		var d30 scm.JITValueDesc
		if d29.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d29.Imm.Int()))))}
		} else {
			r9 := ctx.AllocReg()
			ctx.EmitMovRegReg(r9, d29.Reg)
			ctx.EmitShlRegImm8(r9, 32)
			ctx.EmitShrRegImm8(r9, 32)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d30)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d31 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r10 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r10, fieldAddr)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			ctx.BindReg(r10, &d31)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r11 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r11, thisptr.Reg, off)
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
			ctx.BindReg(r11, &d31)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		var d32 scm.JITValueDesc
		if d31.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d31.Imm.Int()))))}
		} else {
			r12 := ctx.AllocReg()
			ctx.EmitMovRegReg(r12, d31.Reg)
			ctx.EmitShlRegImm8(r12, 56)
			ctx.EmitShrRegImm8(r12, 56)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d32)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d32)
		ctx.EnsureDescsTogether(&d30, &d32)
		var d33 scm.JITValueDesc
		if d30.Loc == scm.LocImm && d32.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() * d32.Imm.Int())}
		} else if d30.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
			ctx.EmitImulInt64(scratch, d32.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d33)
		} else if d32.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegReg(scratch, d30.Reg)
			if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d32.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d33)
		} else {
			r13 := ctx.AllocRegExcept(d30.Reg, d32.Reg)
			ctx.EmitMovRegReg(r13, d30.Reg)
			ctx.EmitImulInt64(r13, d32.Reg)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d33)
		}
		if d33.Loc == scm.LocReg && d30.Loc == scm.LocReg && d33.Reg == d30.Reg {
			ctx.TransferReg(d30.Reg)
			d30.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d30)
		ctx.FreeDesc(&d32)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d33)
		var d34 scm.JITValueDesc
		if d33.Loc == scm.LocImm {
			d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() / 64)}
		} else {
			r14 := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegReg(r14, d33.Reg)
			ctx.EmitShrRegImm8(r14, 6)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d34)
		}
		if d34.Loc == scm.LocReg && d33.Loc == scm.LocReg && d34.Reg == d33.Reg {
			ctx.TransferReg(d33.Reg)
			d33.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d33)
		var d35 scm.JITValueDesc
		if d33.Loc == scm.LocImm {
			d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() % 64)}
		} else {
			r15 := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegReg(r15, d33.Reg)
			ctx.EmitAndRegImm32(r15, 63)
			d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d35)
		}
		if d35.Loc == scm.LocReg && d33.Loc == scm.LocReg && d35.Reg == d33.Reg {
			ctx.TransferReg(d33.Reg)
			d33.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d36 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r16 := ctx.AllocReg()
			r17 := ctx.AllocRegExcept(r16)
			r18 := ctx.AllocRegExcept(r16, r17)
			ctx.EmitMovRegMem64(r16, fieldAddr)
			ctx.EmitMovRegMem64(r17, fieldAddr+8)
			ctx.EmitMovRegMem64(r18, fieldAddr+16)
			d36 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r16, Reg2: r17, Reg3: r18}
			ctx.BindReg(r16, &d36)
			ctx.BindReg(r17, &d36)
			ctx.BindReg(r18, &d36)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r19 := ctx.AllocReg()
			r20 := ctx.AllocRegExcept(r19)
			r21 := ctx.AllocRegExcept(r19, r20)
			ctx.EmitMovRegMem(r19, thisptr.Reg, off)
			ctx.EmitMovRegMem(r20, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r21, thisptr.Reg, off+16)
			d36 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r19, Reg2: r20, Reg3: r21}
			ctx.BindReg(r19, &d36)
			ctx.BindReg(r20, &d36)
			ctx.BindReg(r21, &d36)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.ReclaimUntrackedRegs()
		d38 = ctx.EmitSliceElementAddress(&d36, &d34, 8)
		ctx.EnsureDesc(&d38)
		ctx.EmitMovRegMem(d38.Reg, d38.Reg, 0)
		d37 = d38
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d35)
		var d39 scm.JITValueDesc
		if d37.Loc == scm.LocImm && d35.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d37.Imm.Int()) << uint64(d35.Imm.Int())))}
		} else if d35.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(r22, d37.Reg)
			ctx.EmitShlRegImm8(r22, uint8(d35.Imm.Int()))
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d39)
		} else {
			{
				shiftSrc := d37.Reg
				r23 := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitMovRegReg(r23, d37.Reg)
				shiftSrc = r23
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d35.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d35.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d35.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d39)
			}
		}
		if d39.Loc == scm.LocReg && d37.Loc == scm.LocReg && d39.Reg == d37.Reg {
			ctx.TransferReg(d37.Reg)
			d37.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d34)
		var d40 scm.JITValueDesc
		if d34.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(scratch, d34.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d40)
		}
		if d40.Loc == scm.LocReg && d34.Loc == scm.LocReg && d40.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d34)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		d42 = ctx.EmitSliceElementAddress(&d36, &d40, 8)
		ctx.EnsureDesc(&d42)
		ctx.EmitMovRegMem(d42.Reg, d42.Reg, 0)
		d41 = d42
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d35)
		ctx.EnsureDescsTogether(&d43, &d35)
		var d44 scm.JITValueDesc
		if d43.Loc == scm.LocImm && d35.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() - d35.Imm.Int())}
		} else if d35.Loc == scm.LocImm && d35.Imm.Int() == 0 {
			r24 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r24, d43.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d44)
		} else if d43.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d35.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
			ctx.EmitSubInt64(scratch, d35.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else if d35.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(scratch, d43.Reg)
			if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d35.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d35.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d44)
		} else {
			r25 := ctx.AllocRegExcept(d43.Reg, d35.Reg)
			ctx.EmitMovRegReg(r25, d43.Reg)
			ctx.EmitSubInt64(r25, d35.Reg)
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d44)
		}
		if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d35)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d44)
		var d45 scm.JITValueDesc
		if d41.Loc == scm.LocImm && d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d41.Imm.Int()) >> uint64(d44.Imm.Int())))}
		} else if d44.Loc == scm.LocImm {
			r26 := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(r26, d41.Reg)
			ctx.EmitShrRegImm8(r26, uint8(d44.Imm.Int()))
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d45)
		} else {
			{
				shiftSrc := d41.Reg
				r27 := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(r27, d41.Reg)
				shiftSrc = r27
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d44.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d44.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d44.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d45)
			}
		}
		if d45.Loc == scm.LocReg && d41.Loc == scm.LocReg && d45.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d41)
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d45)
		var d46 scm.JITValueDesc
		if d39.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() | d45.Imm.Int())}
		} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d45.Reg}
			ctx.BindReg(d45.Reg, &d46)
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			r28 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r28, d39.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d46)
		} else if d39.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
			ctx.EmitOrInt64(scratch, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d46)
		} else if d45.Loc == scm.LocImm {
			r29 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r29, d39.Reg)
			if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r29, int32(d45.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.EmitOrInt64(r29, scm.RegR11)
			}
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d46)
		} else {
			r30 := ctx.AllocRegExcept(d39.Reg, d45.Reg)
			ctx.EmitMovRegReg(r30, d39.Reg)
			ctx.EmitOrInt64(r30, d45.Reg)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d46)
		}
		if d46.Loc == scm.LocReg && d39.Loc == scm.LocReg && d46.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		var d47 scm.JITValueDesc
		if d31.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d31.Imm.Int()))))}
		} else {
			r31 := ctx.AllocReg()
			ctx.EmitMovRegReg(r31, d31.Reg)
			ctx.EmitShlRegImm8(r31, 56)
			ctx.EmitShrRegImm8(r31, 56)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d47)
		}
		ctx.ReclaimUntrackedRegs()
		d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d47)
		ctx.EnsureDescsTogether(&d48, &d47)
		var d49 scm.JITValueDesc
		if d48.Loc == scm.LocImm && d47.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() - d47.Imm.Int())}
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			r32 := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegReg(r32, d48.Reg)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d49)
		} else if d48.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d48.Imm.Int()))
			ctx.EmitSubInt64(scratch, d47.Reg)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d49)
		} else if d47.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegReg(scratch, d48.Reg)
			if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d47.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d49)
		} else {
			r33 := ctx.AllocRegExcept(d48.Reg, d47.Reg)
			ctx.EmitMovRegReg(r33, d48.Reg)
			ctx.EmitSubInt64(r33, d47.Reg)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d49)
		}
		if d49.Loc == scm.LocReg && d48.Loc == scm.LocReg && d49.Reg == d48.Reg {
			ctx.TransferReg(d48.Reg)
			d48.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d49)
		var d50 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d49.Loc == scm.LocImm {
			d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d49.Imm.Int())))}
		} else if d49.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r34, d46.Reg)
			ctx.EmitShrRegImm8(r34, uint8(d49.Imm.Int()))
			d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d50)
		} else {
			{
				shiftSrc := d46.Reg
				r35 := ctx.AllocRegExcept(d46.Reg)
				ctx.EmitMovRegReg(r35, d46.Reg)
				shiftSrc = r35
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d49.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d49.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d49.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d50)
			}
		}
		if d50.Loc == scm.LocReg && d46.Loc == scm.LocReg && d50.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.FreeDesc(&d49)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d50)
		ctx.EnsureDesc(&d50)
		ctx.EnsureDesc(&d50)
		var d51 scm.JITValueDesc
		if d50.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d50.Imm.Int()))))}
		} else {
			r36 := ctx.AllocReg()
			ctx.EmitMovRegReg(r36, d50.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d51)
		}
		ctx.FreeDesc(&d50)
		var d52 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r37, fieldAddr)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d52)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
			r38 := ctx.AllocReg()
			ctx.EmitMovRegMem(r38, thisptr.Reg, off)
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d52)
		}
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d52)
		ctx.EnsureDescsTogether(&d51, &d52)
		var d53 scm.JITValueDesc
		if d51.Loc == scm.LocImm && d52.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() + d52.Imm.Int())}
		} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
			r39 := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(r39, d51.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d53)
		} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			ctx.BindReg(d52.Reg, &d53)
		} else if d51.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d51.Imm.Int()))
			ctx.EmitAddInt64(scratch, d52.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d53)
		} else if d52.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d51.Reg)
			ctx.EmitMovRegReg(scratch, d51.Reg)
			if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d52.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d53)
		} else {
			r40 := ctx.AllocRegExcept(d51.Reg, d52.Reg)
			ctx.EmitMovRegReg(r40, d51.Reg)
			ctx.EmitAddInt64(r40, d52.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d53)
		}
		if d53.Loc == scm.LocReg && d51.Loc == scm.LocReg && d53.Reg == d51.Reg {
			ctx.TransferReg(d51.Reg)
			d51.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d51)
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d53)
		var d54 scm.JITValueDesc
		if d53.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d53.Imm.Int()))))}
		} else {
			r41 := ctx.AllocReg()
			ctx.EmitMovRegReg(r41, d53.Reg)
			ctx.EmitShlRegImm8(r41, 32)
			ctx.EmitShrRegImm8(r41, 32)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d54)
		}
		ctx.FreeDesc(&d53)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d54)
		ctx.EnsureDescsTogether(&idxInt, &d54)
		var d55 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d54.Imm.Int()))}
		} else if d54.Loc == scm.LocImm {
			r42 := ctx.AllocRegExcept(idxInt.Reg)
			if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d54.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d54.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r42, scm.CondUnsignedBelow)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r42}
			ctx.BindReg(r42, &d55)
		} else if idxInt.Loc == scm.LocImm {
			r43 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d54.Reg)
			ctx.EmitSetcc(r43, scm.CondUnsignedBelow)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r43}
			ctx.BindReg(r43, &d55)
		} else {
			r44 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d54.Reg)
			ctx.EmitSetcc(r44, scm.CondUnsignedBelow)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r44}
			ctx.BindReg(r44, &d55)
		}
		ctx.FreeDesc(&d54)
		d56 = d55
		ctx.EnsureDesc(&d56)
		if d56.Loc != scm.LocImm && d56.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d56.Loc == scm.LocImm {
			if d56.Imm.Bool() {
				if ps.General {
				}
				ps57 := scm.PhiState{General: ps.General}
				ps57.OverlayValues = make([]scm.JITValueDesc, 57)
				ps57.OverlayValues[5] = d5
				ps57.OverlayValues[6] = d6
				ps57.OverlayValues[7] = d7
				ps57.OverlayValues[8] = d8
				ps57.OverlayValues[9] = d9
				ps57.OverlayValues[10] = d10
				ps57.OverlayValues[11] = d11
				ps57.OverlayValues[12] = d12
				ps57.OverlayValues[13] = d13
				ps57.OverlayValues[14] = d14
				ps57.OverlayValues[15] = d15
				ps57.OverlayValues[16] = d16
				ps57.OverlayValues[17] = d17
				ps57.OverlayValues[18] = d18
				ps57.OverlayValues[19] = d19
				ps57.OverlayValues[20] = d20
				ps57.OverlayValues[21] = d21
				ps57.OverlayValues[23] = d23
				ps57.OverlayValues[24] = d24
				ps57.OverlayValues[25] = d25
				ps57.OverlayValues[26] = d26
				ps57.OverlayValues[27] = d27
				ps57.OverlayValues[28] = d28
				ps57.OverlayValues[29] = d29
				ps57.OverlayValues[30] = d30
				ps57.OverlayValues[31] = d31
				ps57.OverlayValues[32] = d32
				ps57.OverlayValues[33] = d33
				ps57.OverlayValues[34] = d34
				ps57.OverlayValues[35] = d35
				ps57.OverlayValues[36] = d36
				ps57.OverlayValues[37] = d37
				ps57.OverlayValues[38] = d38
				ps57.OverlayValues[39] = d39
				ps57.OverlayValues[40] = d40
				ps57.OverlayValues[41] = d41
				ps57.OverlayValues[42] = d42
				ps57.OverlayValues[43] = d43
				ps57.OverlayValues[44] = d44
				ps57.OverlayValues[45] = d45
				ps57.OverlayValues[46] = d46
				ps57.OverlayValues[47] = d47
				ps57.OverlayValues[48] = d48
				ps57.OverlayValues[49] = d49
				ps57.OverlayValues[50] = d50
				ps57.OverlayValues[51] = d51
				ps57.OverlayValues[52] = d52
				ps57.OverlayValues[53] = d53
				ps57.OverlayValues[54] = d54
				ps57.OverlayValues[55] = d55
				ps57.OverlayValues[56] = d56
				return bbs[3].RenderPS(ps57)
			}
			if ps.General {
			}
			ps58 := scm.PhiState{General: ps.General}
			ps58.OverlayValues = make([]scm.JITValueDesc, 57)
			ps58.OverlayValues[5] = d5
			ps58.OverlayValues[6] = d6
			ps58.OverlayValues[7] = d7
			ps58.OverlayValues[8] = d8
			ps58.OverlayValues[9] = d9
			ps58.OverlayValues[10] = d10
			ps58.OverlayValues[11] = d11
			ps58.OverlayValues[12] = d12
			ps58.OverlayValues[13] = d13
			ps58.OverlayValues[14] = d14
			ps58.OverlayValues[15] = d15
			ps58.OverlayValues[16] = d16
			ps58.OverlayValues[17] = d17
			ps58.OverlayValues[18] = d18
			ps58.OverlayValues[19] = d19
			ps58.OverlayValues[20] = d20
			ps58.OverlayValues[21] = d21
			ps58.OverlayValues[23] = d23
			ps58.OverlayValues[24] = d24
			ps58.OverlayValues[25] = d25
			ps58.OverlayValues[26] = d26
			ps58.OverlayValues[27] = d27
			ps58.OverlayValues[28] = d28
			ps58.OverlayValues[29] = d29
			ps58.OverlayValues[30] = d30
			ps58.OverlayValues[31] = d31
			ps58.OverlayValues[32] = d32
			ps58.OverlayValues[33] = d33
			ps58.OverlayValues[34] = d34
			ps58.OverlayValues[35] = d35
			ps58.OverlayValues[36] = d36
			ps58.OverlayValues[37] = d37
			ps58.OverlayValues[38] = d38
			ps58.OverlayValues[39] = d39
			ps58.OverlayValues[40] = d40
			ps58.OverlayValues[41] = d41
			ps58.OverlayValues[42] = d42
			ps58.OverlayValues[43] = d43
			ps58.OverlayValues[44] = d44
			ps58.OverlayValues[45] = d45
			ps58.OverlayValues[46] = d46
			ps58.OverlayValues[47] = d47
			ps58.OverlayValues[48] = d48
			ps58.OverlayValues[49] = d49
			ps58.OverlayValues[50] = d50
			ps58.OverlayValues[51] = d51
			ps58.OverlayValues[52] = d52
			ps58.OverlayValues[53] = d53
			ps58.OverlayValues[54] = d54
			ps58.OverlayValues[55] = d55
			ps58.OverlayValues[56] = d56
			return bbs[5].RenderPS(ps58)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d59 := ps.PhiValues[0]
				if phiHomeOK2 {
					ctx.EmitMovToReg(r0, d59)
				} else {
					ctx.EmitStoreToStack(d59, int32(bbs[1].PhiBase)+int32(0))
				}
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d60 := ps.PhiValues[1]
				if phiHomeOK3 {
					ctx.EmitMovToReg(r1, d60)
				} else {
					ctx.EmitStoreToStack(d60, int32(bbs[1].PhiBase)+int32(16))
				}
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d61 := ps.PhiValues[2]
				if phiHomeOK4 {
					ctx.EmitMovToReg(r2, d61)
				} else {
					ctx.EmitStoreToStack(d61, int32(bbs[1].PhiBase)+int32(32))
				}
			}
			ps.General = true
			return bbs[1].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d56.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl6)
		ps62 := scm.PhiState{General: true}
		ps62.OverlayValues = make([]scm.JITValueDesc, 62)
		ps62.OverlayValues[5] = d5
		ps62.OverlayValues[6] = d6
		ps62.OverlayValues[7] = d7
		ps62.OverlayValues[8] = d8
		ps62.OverlayValues[9] = d9
		ps62.OverlayValues[10] = d10
		ps62.OverlayValues[11] = d11
		ps62.OverlayValues[12] = d12
		ps62.OverlayValues[13] = d13
		ps62.OverlayValues[14] = d14
		ps62.OverlayValues[15] = d15
		ps62.OverlayValues[16] = d16
		ps62.OverlayValues[17] = d17
		ps62.OverlayValues[18] = d18
		ps62.OverlayValues[19] = d19
		ps62.OverlayValues[20] = d20
		ps62.OverlayValues[21] = d21
		ps62.OverlayValues[23] = d23
		ps62.OverlayValues[24] = d24
		ps62.OverlayValues[25] = d25
		ps62.OverlayValues[26] = d26
		ps62.OverlayValues[27] = d27
		ps62.OverlayValues[28] = d28
		ps62.OverlayValues[29] = d29
		ps62.OverlayValues[30] = d30
		ps62.OverlayValues[31] = d31
		ps62.OverlayValues[32] = d32
		ps62.OverlayValues[33] = d33
		ps62.OverlayValues[34] = d34
		ps62.OverlayValues[35] = d35
		ps62.OverlayValues[36] = d36
		ps62.OverlayValues[37] = d37
		ps62.OverlayValues[38] = d38
		ps62.OverlayValues[39] = d39
		ps62.OverlayValues[40] = d40
		ps62.OverlayValues[41] = d41
		ps62.OverlayValues[42] = d42
		ps62.OverlayValues[43] = d43
		ps62.OverlayValues[44] = d44
		ps62.OverlayValues[45] = d45
		ps62.OverlayValues[46] = d46
		ps62.OverlayValues[47] = d47
		ps62.OverlayValues[48] = d48
		ps62.OverlayValues[49] = d49
		ps62.OverlayValues[50] = d50
		ps62.OverlayValues[51] = d51
		ps62.OverlayValues[52] = d52
		ps62.OverlayValues[53] = d53
		ps62.OverlayValues[54] = d54
		ps62.OverlayValues[55] = d55
		ps62.OverlayValues[56] = d56
		ps62.OverlayValues[59] = d59
		ps62.OverlayValues[60] = d60
		ps62.OverlayValues[61] = d61
		ps63 := scm.PhiState{General: true}
		ps63.OverlayValues = make([]scm.JITValueDesc, 62)
		ps63.OverlayValues[5] = d5
		ps63.OverlayValues[6] = d6
		ps63.OverlayValues[7] = d7
		ps63.OverlayValues[8] = d8
		ps63.OverlayValues[9] = d9
		ps63.OverlayValues[10] = d10
		ps63.OverlayValues[11] = d11
		ps63.OverlayValues[12] = d12
		ps63.OverlayValues[13] = d13
		ps63.OverlayValues[14] = d14
		ps63.OverlayValues[15] = d15
		ps63.OverlayValues[16] = d16
		ps63.OverlayValues[17] = d17
		ps63.OverlayValues[18] = d18
		ps63.OverlayValues[19] = d19
		ps63.OverlayValues[20] = d20
		ps63.OverlayValues[21] = d21
		ps63.OverlayValues[23] = d23
		ps63.OverlayValues[24] = d24
		ps63.OverlayValues[25] = d25
		ps63.OverlayValues[26] = d26
		ps63.OverlayValues[27] = d27
		ps63.OverlayValues[28] = d28
		ps63.OverlayValues[29] = d29
		ps63.OverlayValues[30] = d30
		ps63.OverlayValues[31] = d31
		ps63.OverlayValues[32] = d32
		ps63.OverlayValues[33] = d33
		ps63.OverlayValues[34] = d34
		ps63.OverlayValues[35] = d35
		ps63.OverlayValues[36] = d36
		ps63.OverlayValues[37] = d37
		ps63.OverlayValues[38] = d38
		ps63.OverlayValues[39] = d39
		ps63.OverlayValues[40] = d40
		ps63.OverlayValues[41] = d41
		ps63.OverlayValues[42] = d42
		ps63.OverlayValues[43] = d43
		ps63.OverlayValues[44] = d44
		ps63.OverlayValues[45] = d45
		ps63.OverlayValues[46] = d46
		ps63.OverlayValues[47] = d47
		ps63.OverlayValues[48] = d48
		ps63.OverlayValues[49] = d49
		ps63.OverlayValues[50] = d50
		ps63.OverlayValues[51] = d51
		ps63.OverlayValues[52] = d52
		ps63.OverlayValues[53] = d53
		ps63.OverlayValues[54] = d54
		ps63.OverlayValues[55] = d55
		ps63.OverlayValues[56] = d56
		ps63.OverlayValues[59] = d59
		ps63.OverlayValues[60] = d60
		ps63.OverlayValues[61] = d61
		snap64 := d5
		snap65 := d6
		snap66 := d7
		snap67 := d8
		snap68 := d9
		snap69 := d10
		snap70 := d11
		snap71 := d12
		snap72 := d13
		snap73 := d14
		snap74 := d15
		snap75 := d16
		snap76 := d17
		snap77 := d18
		snap78 := d19
		snap79 := d20
		snap80 := d21
		snap81 := d23
		snap82 := d24
		snap83 := d25
		snap84 := d26
		snap85 := d27
		snap86 := d28
		snap87 := d29
		snap88 := d30
		snap89 := d31
		snap90 := d32
		snap91 := d33
		snap92 := d34
		snap93 := d35
		snap94 := d36
		snap95 := d37
		snap96 := d38
		snap97 := d39
		snap98 := d40
		snap99 := d41
		snap100 := d42
		snap101 := d43
		snap102 := d44
		snap103 := d45
		snap104 := d46
		snap105 := d47
		snap106 := d48
		snap107 := d49
		snap108 := d50
		snap109 := d51
		snap110 := d52
		snap111 := d53
		snap112 := d54
		snap113 := d55
		snap114 := d56
		snap115 := d59
		snap116 := d60
		snap117 := d61
		alloc118 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps63)
		}
		ctx.RestoreAllocState(alloc118)
		d5 = snap64
		d6 = snap65
		d7 = snap66
		d8 = snap67
		d9 = snap68
		d10 = snap69
		d11 = snap70
		d12 = snap71
		d13 = snap72
		d14 = snap73
		d15 = snap74
		d16 = snap75
		d17 = snap76
		d18 = snap77
		d19 = snap78
		d20 = snap79
		d21 = snap80
		d23 = snap81
		d24 = snap82
		d25 = snap83
		d26 = snap84
		d27 = snap85
		d28 = snap86
		d29 = snap87
		d30 = snap88
		d31 = snap89
		d32 = snap90
		d33 = snap91
		d34 = snap92
		d35 = snap93
		d36 = snap94
		d37 = snap95
		d38 = snap96
		d39 = snap97
		d40 = snap98
		d41 = snap99
		d42 = snap100
		d43 = snap101
		d44 = snap102
		d45 = snap103
		d46 = snap104
		d47 = snap105
		d48 = snap106
		d49 = snap107
		d50 = snap108
		d51 = snap109
		d52 = snap110
		d53 = snap111
		d54 = snap112
		d55 = snap113
		d56 = snap114
		d59 = snap115
		d60 = snap116
		d61 = snap117
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps62)
		}
		return result
		ctx.FreeDesc(&d55)
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d119 := ps.PhiValues[0]
				ctx.EmitStoreToStack(d119, int32(bbs[2].PhiBase)+int32(0))
			}
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
			d119 = ps.OverlayValues[119]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d8 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d8)
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d8)
		var d120 scm.JITValueDesc
		if d8.Loc == scm.LocImm {
			d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d8.Imm.Int()))))}
		} else {
			r45 := ctx.AllocReg()
			ctx.EmitMovRegReg(r45, d8.Reg)
			ctx.EmitShlRegImm8(r45, 32)
			ctx.EmitShrRegImm8(r45, 32)
			d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			ctx.BindReg(r45, &d120)
		}
		ctx.EnsureDesc(&d120)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d120.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d120.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d120.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d120.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d120.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d120.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d120)
		ctx.EnsureDesc(&d8)
		d121 = d8
		_ = d121
		ctx.StabilizeDescForControlFlow(&d121)
		ctx.StabilizeDescForControlFlow(&d8)
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl18 := ctx.ReserveLabel()
		_ = lbl18
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl18)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d121)
		ctx.EnsureDesc(&d121)
		var d122 scm.JITValueDesc
		if d121.Loc == scm.LocImm {
			d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d121.Imm.Int()))))}
		} else {
			r46 := ctx.AllocReg()
			ctx.EmitMovRegReg(r46, d121.Reg)
			ctx.EmitShlRegImm8(r46, 32)
			ctx.EmitShrRegImm8(r46, 32)
			d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
			ctx.BindReg(r46, &d122)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d123 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			r47 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r47, fieldAddr)
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			ctx.BindReg(r47, &d123)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			r48 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r48, thisptr.Reg, off)
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
			ctx.BindReg(r48, &d123)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d123)
		ctx.EnsureDesc(&d123)
		var d124 scm.JITValueDesc
		if d123.Loc == scm.LocImm {
			d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d123.Imm.Int()))))}
		} else {
			r49 := ctx.AllocReg()
			ctx.EmitMovRegReg(r49, d123.Reg)
			ctx.EmitShlRegImm8(r49, 56)
			ctx.EmitShrRegImm8(r49, 56)
			d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d124)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d122)
		ctx.EnsureDesc(&d124)
		ctx.EnsureDescsTogether(&d122, &d124)
		var d125 scm.JITValueDesc
		if d122.Loc == scm.LocImm && d124.Loc == scm.LocImm {
			d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() * d124.Imm.Int())}
		} else if d122.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d124.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d122.Imm.Int()))
			ctx.EmitImulInt64(scratch, d124.Reg)
			d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d125)
		} else if d124.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d122.Reg)
			ctx.EmitMovRegReg(scratch, d122.Reg)
			if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d124.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d125)
		} else {
			r50 := ctx.AllocRegExcept(d122.Reg, d124.Reg)
			ctx.EmitMovRegReg(r50, d122.Reg)
			ctx.EmitImulInt64(r50, d124.Reg)
			d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d125)
		}
		if d125.Loc == scm.LocReg && d122.Loc == scm.LocReg && d125.Reg == d122.Reg {
			ctx.TransferReg(d122.Reg)
			d122.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d122)
		ctx.FreeDesc(&d124)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		var d126 scm.JITValueDesc
		if d125.Loc == scm.LocImm {
			d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() / 64)}
		} else {
			r51 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r51, d125.Reg)
			ctx.EmitShrRegImm8(r51, 6)
			d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d126)
		}
		if d126.Loc == scm.LocReg && d125.Loc == scm.LocReg && d126.Reg == d125.Reg {
			ctx.TransferReg(d125.Reg)
			d125.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		var d127 scm.JITValueDesc
		if d125.Loc == scm.LocImm {
			d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() % 64)}
		} else {
			r52 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r52, d125.Reg)
			ctx.EmitAndRegImm32(r52, 63)
			d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
			ctx.BindReg(r52, &d127)
		}
		if d127.Loc == scm.LocReg && d125.Loc == scm.LocReg && d127.Reg == d125.Reg {
			ctx.TransferReg(d125.Reg)
			d125.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d125)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d128 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
			r53 := ctx.AllocReg()
			r54 := ctx.AllocRegExcept(r53)
			r55 := ctx.AllocRegExcept(r53, r54)
			ctx.EmitMovRegMem64(r53, fieldAddr)
			ctx.EmitMovRegMem64(r54, fieldAddr+8)
			ctx.EmitMovRegMem64(r55, fieldAddr+16)
			d128 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r53, Reg2: r54, Reg3: r55}
			ctx.BindReg(r53, &d128)
			ctx.BindReg(r54, &d128)
			ctx.BindReg(r55, &d128)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
			r56 := ctx.AllocReg()
			r57 := ctx.AllocRegExcept(r56)
			r58 := ctx.AllocRegExcept(r56, r57)
			ctx.EmitMovRegMem(r56, thisptr.Reg, off)
			ctx.EmitMovRegMem(r57, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r58, thisptr.Reg, off+16)
			d128 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r56, Reg2: r57, Reg3: r58}
			ctx.BindReg(r56, &d128)
			ctx.BindReg(r57, &d128)
			ctx.BindReg(r58, &d128)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d126)
		ctx.ReclaimUntrackedRegs()
		d130 = ctx.EmitSliceElementAddress(&d128, &d126, 8)
		ctx.EnsureDesc(&d130)
		ctx.EmitMovRegMem(d130.Reg, d130.Reg, 0)
		d129 = d130
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d129)
		ctx.EnsureDesc(&d127)
		var d131 scm.JITValueDesc
		if d129.Loc == scm.LocImm && d127.Loc == scm.LocImm {
			d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d129.Imm.Int()) << uint64(d127.Imm.Int())))}
		} else if d127.Loc == scm.LocImm {
			r59 := ctx.AllocRegExcept(d129.Reg)
			ctx.EmitMovRegReg(r59, d129.Reg)
			ctx.EmitShlRegImm8(r59, uint8(d127.Imm.Int()))
			d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			ctx.BindReg(r59, &d131)
		} else {
			{
				shiftSrc := d129.Reg
				r60 := ctx.AllocRegExcept(d129.Reg)
				ctx.EmitMovRegReg(r60, d129.Reg)
				shiftSrc = r60
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d127.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d127.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d127.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d131)
			}
		}
		if d131.Loc == scm.LocReg && d129.Loc == scm.LocReg && d131.Reg == d129.Reg {
			ctx.TransferReg(d129.Reg)
			d129.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d129)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d126)
		ctx.EnsureDesc(&d126)
		var d132 scm.JITValueDesc
		if d126.Loc == scm.LocImm {
			d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d126.Reg)
			ctx.EmitMovRegReg(scratch, d126.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d132)
		}
		if d132.Loc == scm.LocReg && d126.Loc == scm.LocReg && d132.Reg == d126.Reg {
			ctx.TransferReg(d126.Reg)
			d126.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d126)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d132)
		ctx.ReclaimUntrackedRegs()
		d134 = ctx.EmitSliceElementAddress(&d128, &d132, 8)
		ctx.EnsureDesc(&d134)
		ctx.EmitMovRegMem(d134.Reg, d134.Reg, 0)
		d133 = d134
		ctx.FreeDesc(&d132)
		ctx.ReclaimUntrackedRegs()
		d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d127)
		ctx.EnsureDescsTogether(&d135, &d127)
		var d136 scm.JITValueDesc
		if d135.Loc == scm.LocImm && d127.Loc == scm.LocImm {
			d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() - d127.Imm.Int())}
		} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
			r61 := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitMovRegReg(r61, d135.Reg)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
			ctx.BindReg(r61, &d136)
		} else if d135.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d127.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d135.Imm.Int()))
			ctx.EmitSubInt64(scratch, d127.Reg)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d136)
		} else if d127.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d135.Reg)
			ctx.EmitMovRegReg(scratch, d135.Reg)
			if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d127.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d136)
		} else {
			r62 := ctx.AllocRegExcept(d135.Reg, d127.Reg)
			ctx.EmitMovRegReg(r62, d135.Reg)
			ctx.EmitSubInt64(r62, d127.Reg)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d136)
		}
		if d136.Loc == scm.LocReg && d135.Loc == scm.LocReg && d136.Reg == d135.Reg {
			ctx.TransferReg(d135.Reg)
			d135.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d127)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d133)
		ctx.EnsureDesc(&d136)
		var d137 scm.JITValueDesc
		if d133.Loc == scm.LocImm && d136.Loc == scm.LocImm {
			d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d133.Imm.Int()) >> uint64(d136.Imm.Int())))}
		} else if d136.Loc == scm.LocImm {
			r63 := ctx.AllocRegExcept(d133.Reg)
			ctx.EmitMovRegReg(r63, d133.Reg)
			ctx.EmitShrRegImm8(r63, uint8(d136.Imm.Int()))
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			ctx.BindReg(r63, &d137)
		} else {
			{
				shiftSrc := d133.Reg
				r64 := ctx.AllocRegExcept(d133.Reg)
				ctx.EmitMovRegReg(r64, d133.Reg)
				shiftSrc = r64
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d136.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d136.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d136.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d137)
			}
		}
		if d137.Loc == scm.LocReg && d133.Loc == scm.LocReg && d137.Reg == d133.Reg {
			ctx.TransferReg(d133.Reg)
			d133.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d133)
		ctx.FreeDesc(&d136)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d131)
		ctx.EnsureDesc(&d137)
		var d138 scm.JITValueDesc
		if d131.Loc == scm.LocImm && d137.Loc == scm.LocImm {
			d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d131.Imm.Int() | d137.Imm.Int())}
		} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d137.Reg}
			ctx.BindReg(d137.Reg, &d138)
		} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
			r65 := ctx.AllocRegExcept(d131.Reg)
			ctx.EmitMovRegReg(r65, d131.Reg)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d138)
		} else if d131.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d137.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d131.Imm.Int()))
			ctx.EmitOrInt64(scratch, d137.Reg)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d138)
		} else if d137.Loc == scm.LocImm {
			r66 := ctx.AllocRegExcept(d131.Reg)
			ctx.EmitMovRegReg(r66, d131.Reg)
			if d137.Imm.Int() >= -2147483648 && d137.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r66, int32(d137.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
				ctx.EmitOrInt64(r66, scm.RegR11)
			}
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			ctx.BindReg(r66, &d138)
		} else {
			r67 := ctx.AllocRegExcept(d131.Reg, d137.Reg)
			ctx.EmitMovRegReg(r67, d131.Reg)
			ctx.EmitOrInt64(r67, d137.Reg)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			ctx.BindReg(r67, &d138)
		}
		if d138.Loc == scm.LocReg && d131.Loc == scm.LocReg && d138.Reg == d131.Reg {
			ctx.TransferReg(d131.Reg)
			d131.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d131)
		ctx.FreeDesc(&d137)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d123)
		ctx.EnsureDesc(&d123)
		var d139 scm.JITValueDesc
		if d123.Loc == scm.LocImm {
			d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d123.Imm.Int()))))}
		} else {
			r68 := ctx.AllocReg()
			ctx.EmitMovRegReg(r68, d123.Reg)
			ctx.EmitShlRegImm8(r68, 56)
			ctx.EmitShrRegImm8(r68, 56)
			d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			ctx.BindReg(r68, &d139)
		}
		ctx.ReclaimUntrackedRegs()
		d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d139)
		ctx.EnsureDescsTogether(&d140, &d139)
		var d141 scm.JITValueDesc
		if d140.Loc == scm.LocImm && d139.Loc == scm.LocImm {
			d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() - d139.Imm.Int())}
		} else if d139.Loc == scm.LocImm && d139.Imm.Int() == 0 {
			r69 := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(r69, d140.Reg)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
			ctx.BindReg(r69, &d141)
		} else if d140.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d139.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d140.Imm.Int()))
			ctx.EmitSubInt64(scratch, d139.Reg)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d141)
		} else if d139.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d140.Reg)
			ctx.EmitMovRegReg(scratch, d140.Reg)
			if d139.Imm.Int() >= -2147483648 && d139.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d139.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d139.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d141)
		} else {
			r70 := ctx.AllocRegExcept(d140.Reg, d139.Reg)
			ctx.EmitMovRegReg(r70, d140.Reg)
			ctx.EmitSubInt64(r70, d139.Reg)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			ctx.BindReg(r70, &d141)
		}
		if d141.Loc == scm.LocReg && d140.Loc == scm.LocReg && d141.Reg == d140.Reg {
			ctx.TransferReg(d140.Reg)
			d140.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d139)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d138)
		ctx.EnsureDesc(&d141)
		var d142 scm.JITValueDesc
		if d138.Loc == scm.LocImm && d141.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d138.Imm.Int()) >> uint64(d141.Imm.Int())))}
		} else if d141.Loc == scm.LocImm {
			r71 := ctx.AllocRegExcept(d138.Reg)
			ctx.EmitMovRegReg(r71, d138.Reg)
			ctx.EmitShrRegImm8(r71, uint8(d141.Imm.Int()))
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
			ctx.BindReg(r71, &d142)
		} else {
			{
				shiftSrc := d138.Reg
				r72 := ctx.AllocRegExcept(d138.Reg)
				ctx.EmitMovRegReg(r72, d138.Reg)
				shiftSrc = r72
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d141.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d141.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d141.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d142)
			}
		}
		if d142.Loc == scm.LocReg && d138.Loc == scm.LocReg && d142.Reg == d138.Reg {
			ctx.TransferReg(d138.Reg)
			d138.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d138)
		ctx.FreeDesc(&d141)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d142)
		ctx.EnsureDesc(&d142)
		ctx.EnsureDesc(&d142)
		var d143 scm.JITValueDesc
		if d142.Loc == scm.LocImm {
			d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d142.Imm.Int()))))}
		} else {
			r73 := ctx.AllocReg()
			ctx.EmitMovRegReg(r73, d142.Reg)
			d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d143)
		}
		ctx.FreeDesc(&d142)
		var d144 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
			r74 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r74, fieldAddr)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
			ctx.BindReg(r74, &d144)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
			r75 := ctx.AllocReg()
			ctx.EmitMovRegMem(r75, thisptr.Reg, off)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r75}
			ctx.BindReg(r75, &d144)
		}
		ctx.EnsureDesc(&d143)
		ctx.EnsureDesc(&d144)
		ctx.EnsureDescsTogether(&d143, &d144)
		var d145 scm.JITValueDesc
		if d143.Loc == scm.LocImm && d144.Loc == scm.LocImm {
			d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() + d144.Imm.Int())}
		} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
			r76 := ctx.AllocRegExcept(d143.Reg)
			ctx.EmitMovRegReg(r76, d143.Reg)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			ctx.BindReg(r76, &d145)
		} else if d143.Loc == scm.LocImm && d143.Imm.Int() == 0 {
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d144.Reg}
			ctx.BindReg(d144.Reg, &d145)
		} else if d143.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d144.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d143.Imm.Int()))
			ctx.EmitAddInt64(scratch, d144.Reg)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d145)
		} else if d144.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d143.Reg)
			ctx.EmitMovRegReg(scratch, d143.Reg)
			if d144.Imm.Int() >= -2147483648 && d144.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d144.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d145)
		} else {
			r77 := ctx.AllocRegExcept(d143.Reg, d144.Reg)
			ctx.EmitMovRegReg(r77, d143.Reg)
			ctx.EmitAddInt64(r77, d144.Reg)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
			ctx.BindReg(r77, &d145)
		}
		if d145.Loc == scm.LocReg && d143.Loc == scm.LocReg && d145.Reg == d143.Reg {
			ctx.TransferReg(d143.Reg)
			d143.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d145)
		ctx.FreeDesc(&d143)
		var d146 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			r78 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r78, fieldAddr)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r78}
			ctx.BindReg(r78, &d146)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r79 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r79, thisptr.Reg, off)
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			ctx.BindReg(r79, &d146)
		}
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
				ps148.OverlayValues[5] = d5
				ps148.OverlayValues[6] = d6
				ps148.OverlayValues[7] = d7
				ps148.OverlayValues[8] = d8
				ps148.OverlayValues[9] = d9
				ps148.OverlayValues[10] = d10
				ps148.OverlayValues[11] = d11
				ps148.OverlayValues[12] = d12
				ps148.OverlayValues[13] = d13
				ps148.OverlayValues[14] = d14
				ps148.OverlayValues[15] = d15
				ps148.OverlayValues[16] = d16
				ps148.OverlayValues[17] = d17
				ps148.OverlayValues[18] = d18
				ps148.OverlayValues[19] = d19
				ps148.OverlayValues[20] = d20
				ps148.OverlayValues[21] = d21
				ps148.OverlayValues[23] = d23
				ps148.OverlayValues[24] = d24
				ps148.OverlayValues[25] = d25
				ps148.OverlayValues[26] = d26
				ps148.OverlayValues[27] = d27
				ps148.OverlayValues[28] = d28
				ps148.OverlayValues[29] = d29
				ps148.OverlayValues[30] = d30
				ps148.OverlayValues[31] = d31
				ps148.OverlayValues[32] = d32
				ps148.OverlayValues[33] = d33
				ps148.OverlayValues[34] = d34
				ps148.OverlayValues[35] = d35
				ps148.OverlayValues[36] = d36
				ps148.OverlayValues[37] = d37
				ps148.OverlayValues[38] = d38
				ps148.OverlayValues[39] = d39
				ps148.OverlayValues[40] = d40
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
				ps148.OverlayValues[59] = d59
				ps148.OverlayValues[60] = d60
				ps148.OverlayValues[61] = d61
				ps148.OverlayValues[119] = d119
				ps148.OverlayValues[120] = d120
				ps148.OverlayValues[121] = d121
				ps148.OverlayValues[122] = d122
				ps148.OverlayValues[123] = d123
				ps148.OverlayValues[124] = d124
				ps148.OverlayValues[125] = d125
				ps148.OverlayValues[126] = d126
				ps148.OverlayValues[127] = d127
				ps148.OverlayValues[128] = d128
				ps148.OverlayValues[129] = d129
				ps148.OverlayValues[130] = d130
				ps148.OverlayValues[131] = d131
				ps148.OverlayValues[132] = d132
				ps148.OverlayValues[133] = d133
				ps148.OverlayValues[134] = d134
				ps148.OverlayValues[135] = d135
				ps148.OverlayValues[136] = d136
				ps148.OverlayValues[137] = d137
				ps148.OverlayValues[138] = d138
				ps148.OverlayValues[139] = d139
				ps148.OverlayValues[140] = d140
				ps148.OverlayValues[141] = d141
				ps148.OverlayValues[142] = d142
				ps148.OverlayValues[143] = d143
				ps148.OverlayValues[144] = d144
				ps148.OverlayValues[145] = d145
				ps148.OverlayValues[146] = d146
				ps148.OverlayValues[147] = d147
				return bbs[13].RenderPS(ps148)
			}
			if ps.General {
			}
			ps149 := scm.PhiState{General: ps.General}
			ps149.OverlayValues = make([]scm.JITValueDesc, 148)
			ps149.OverlayValues[5] = d5
			ps149.OverlayValues[6] = d6
			ps149.OverlayValues[7] = d7
			ps149.OverlayValues[8] = d8
			ps149.OverlayValues[9] = d9
			ps149.OverlayValues[10] = d10
			ps149.OverlayValues[11] = d11
			ps149.OverlayValues[12] = d12
			ps149.OverlayValues[13] = d13
			ps149.OverlayValues[14] = d14
			ps149.OverlayValues[15] = d15
			ps149.OverlayValues[16] = d16
			ps149.OverlayValues[17] = d17
			ps149.OverlayValues[18] = d18
			ps149.OverlayValues[19] = d19
			ps149.OverlayValues[20] = d20
			ps149.OverlayValues[21] = d21
			ps149.OverlayValues[23] = d23
			ps149.OverlayValues[24] = d24
			ps149.OverlayValues[25] = d25
			ps149.OverlayValues[26] = d26
			ps149.OverlayValues[27] = d27
			ps149.OverlayValues[28] = d28
			ps149.OverlayValues[29] = d29
			ps149.OverlayValues[30] = d30
			ps149.OverlayValues[31] = d31
			ps149.OverlayValues[32] = d32
			ps149.OverlayValues[33] = d33
			ps149.OverlayValues[34] = d34
			ps149.OverlayValues[35] = d35
			ps149.OverlayValues[36] = d36
			ps149.OverlayValues[37] = d37
			ps149.OverlayValues[38] = d38
			ps149.OverlayValues[39] = d39
			ps149.OverlayValues[40] = d40
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
			ps149.OverlayValues[59] = d59
			ps149.OverlayValues[60] = d60
			ps149.OverlayValues[61] = d61
			ps149.OverlayValues[119] = d119
			ps149.OverlayValues[120] = d120
			ps149.OverlayValues[121] = d121
			ps149.OverlayValues[122] = d122
			ps149.OverlayValues[123] = d123
			ps149.OverlayValues[124] = d124
			ps149.OverlayValues[125] = d125
			ps149.OverlayValues[126] = d126
			ps149.OverlayValues[127] = d127
			ps149.OverlayValues[128] = d128
			ps149.OverlayValues[129] = d129
			ps149.OverlayValues[130] = d130
			ps149.OverlayValues[131] = d131
			ps149.OverlayValues[132] = d132
			ps149.OverlayValues[133] = d133
			ps149.OverlayValues[134] = d134
			ps149.OverlayValues[135] = d135
			ps149.OverlayValues[136] = d136
			ps149.OverlayValues[137] = d137
			ps149.OverlayValues[138] = d138
			ps149.OverlayValues[139] = d139
			ps149.OverlayValues[140] = d140
			ps149.OverlayValues[141] = d141
			ps149.OverlayValues[142] = d142
			ps149.OverlayValues[143] = d143
			ps149.OverlayValues[144] = d144
			ps149.OverlayValues[145] = d145
			ps149.OverlayValues[146] = d146
			ps149.OverlayValues[147] = d147
			return bbs[12].RenderPS(ps149)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d150 := ps.PhiValues[0]
				ctx.EmitStoreToStack(d150, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d147.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl19)
		ctx.EmitJmp(lbl20)
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl13)
		ps151 := scm.PhiState{General: true}
		ps151.OverlayValues = make([]scm.JITValueDesc, 151)
		ps151.OverlayValues[5] = d5
		ps151.OverlayValues[6] = d6
		ps151.OverlayValues[7] = d7
		ps151.OverlayValues[8] = d8
		ps151.OverlayValues[9] = d9
		ps151.OverlayValues[10] = d10
		ps151.OverlayValues[11] = d11
		ps151.OverlayValues[12] = d12
		ps151.OverlayValues[13] = d13
		ps151.OverlayValues[14] = d14
		ps151.OverlayValues[15] = d15
		ps151.OverlayValues[16] = d16
		ps151.OverlayValues[17] = d17
		ps151.OverlayValues[18] = d18
		ps151.OverlayValues[19] = d19
		ps151.OverlayValues[20] = d20
		ps151.OverlayValues[21] = d21
		ps151.OverlayValues[23] = d23
		ps151.OverlayValues[24] = d24
		ps151.OverlayValues[25] = d25
		ps151.OverlayValues[26] = d26
		ps151.OverlayValues[27] = d27
		ps151.OverlayValues[28] = d28
		ps151.OverlayValues[29] = d29
		ps151.OverlayValues[30] = d30
		ps151.OverlayValues[31] = d31
		ps151.OverlayValues[32] = d32
		ps151.OverlayValues[33] = d33
		ps151.OverlayValues[34] = d34
		ps151.OverlayValues[35] = d35
		ps151.OverlayValues[36] = d36
		ps151.OverlayValues[37] = d37
		ps151.OverlayValues[38] = d38
		ps151.OverlayValues[39] = d39
		ps151.OverlayValues[40] = d40
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
		ps151.OverlayValues[59] = d59
		ps151.OverlayValues[60] = d60
		ps151.OverlayValues[61] = d61
		ps151.OverlayValues[119] = d119
		ps151.OverlayValues[120] = d120
		ps151.OverlayValues[121] = d121
		ps151.OverlayValues[122] = d122
		ps151.OverlayValues[123] = d123
		ps151.OverlayValues[124] = d124
		ps151.OverlayValues[125] = d125
		ps151.OverlayValues[126] = d126
		ps151.OverlayValues[127] = d127
		ps151.OverlayValues[128] = d128
		ps151.OverlayValues[129] = d129
		ps151.OverlayValues[130] = d130
		ps151.OverlayValues[131] = d131
		ps151.OverlayValues[132] = d132
		ps151.OverlayValues[133] = d133
		ps151.OverlayValues[134] = d134
		ps151.OverlayValues[135] = d135
		ps151.OverlayValues[136] = d136
		ps151.OverlayValues[137] = d137
		ps151.OverlayValues[138] = d138
		ps151.OverlayValues[139] = d139
		ps151.OverlayValues[140] = d140
		ps151.OverlayValues[141] = d141
		ps151.OverlayValues[142] = d142
		ps151.OverlayValues[143] = d143
		ps151.OverlayValues[144] = d144
		ps151.OverlayValues[145] = d145
		ps151.OverlayValues[146] = d146
		ps151.OverlayValues[147] = d147
		ps151.OverlayValues[150] = d150
		ps152 := scm.PhiState{General: true}
		ps152.OverlayValues = make([]scm.JITValueDesc, 151)
		ps152.OverlayValues[5] = d5
		ps152.OverlayValues[6] = d6
		ps152.OverlayValues[7] = d7
		ps152.OverlayValues[8] = d8
		ps152.OverlayValues[9] = d9
		ps152.OverlayValues[10] = d10
		ps152.OverlayValues[11] = d11
		ps152.OverlayValues[12] = d12
		ps152.OverlayValues[13] = d13
		ps152.OverlayValues[14] = d14
		ps152.OverlayValues[15] = d15
		ps152.OverlayValues[16] = d16
		ps152.OverlayValues[17] = d17
		ps152.OverlayValues[18] = d18
		ps152.OverlayValues[19] = d19
		ps152.OverlayValues[20] = d20
		ps152.OverlayValues[21] = d21
		ps152.OverlayValues[23] = d23
		ps152.OverlayValues[24] = d24
		ps152.OverlayValues[25] = d25
		ps152.OverlayValues[26] = d26
		ps152.OverlayValues[27] = d27
		ps152.OverlayValues[28] = d28
		ps152.OverlayValues[29] = d29
		ps152.OverlayValues[30] = d30
		ps152.OverlayValues[31] = d31
		ps152.OverlayValues[32] = d32
		ps152.OverlayValues[33] = d33
		ps152.OverlayValues[34] = d34
		ps152.OverlayValues[35] = d35
		ps152.OverlayValues[36] = d36
		ps152.OverlayValues[37] = d37
		ps152.OverlayValues[38] = d38
		ps152.OverlayValues[39] = d39
		ps152.OverlayValues[40] = d40
		ps152.OverlayValues[41] = d41
		ps152.OverlayValues[42] = d42
		ps152.OverlayValues[43] = d43
		ps152.OverlayValues[44] = d44
		ps152.OverlayValues[45] = d45
		ps152.OverlayValues[46] = d46
		ps152.OverlayValues[47] = d47
		ps152.OverlayValues[48] = d48
		ps152.OverlayValues[49] = d49
		ps152.OverlayValues[50] = d50
		ps152.OverlayValues[51] = d51
		ps152.OverlayValues[52] = d52
		ps152.OverlayValues[53] = d53
		ps152.OverlayValues[54] = d54
		ps152.OverlayValues[55] = d55
		ps152.OverlayValues[56] = d56
		ps152.OverlayValues[59] = d59
		ps152.OverlayValues[60] = d60
		ps152.OverlayValues[61] = d61
		ps152.OverlayValues[119] = d119
		ps152.OverlayValues[120] = d120
		ps152.OverlayValues[121] = d121
		ps152.OverlayValues[122] = d122
		ps152.OverlayValues[123] = d123
		ps152.OverlayValues[124] = d124
		ps152.OverlayValues[125] = d125
		ps152.OverlayValues[126] = d126
		ps152.OverlayValues[127] = d127
		ps152.OverlayValues[128] = d128
		ps152.OverlayValues[129] = d129
		ps152.OverlayValues[130] = d130
		ps152.OverlayValues[131] = d131
		ps152.OverlayValues[132] = d132
		ps152.OverlayValues[133] = d133
		ps152.OverlayValues[134] = d134
		ps152.OverlayValues[135] = d135
		ps152.OverlayValues[136] = d136
		ps152.OverlayValues[137] = d137
		ps152.OverlayValues[138] = d138
		ps152.OverlayValues[139] = d139
		ps152.OverlayValues[140] = d140
		ps152.OverlayValues[141] = d141
		ps152.OverlayValues[142] = d142
		ps152.OverlayValues[143] = d143
		ps152.OverlayValues[144] = d144
		ps152.OverlayValues[145] = d145
		ps152.OverlayValues[146] = d146
		ps152.OverlayValues[147] = d147
		ps152.OverlayValues[150] = d150
		snap153 := d5
		snap154 := d6
		snap155 := d7
		snap156 := d8
		snap157 := d9
		snap158 := d10
		snap159 := d11
		snap160 := d12
		snap161 := d13
		snap162 := d14
		snap163 := d15
		snap164 := d16
		snap165 := d17
		snap166 := d18
		snap167 := d19
		snap168 := d20
		snap169 := d21
		snap170 := d23
		snap171 := d24
		snap172 := d25
		snap173 := d26
		snap174 := d27
		snap175 := d28
		snap176 := d29
		snap177 := d30
		snap178 := d31
		snap179 := d32
		snap180 := d33
		snap181 := d34
		snap182 := d35
		snap183 := d36
		snap184 := d37
		snap185 := d38
		snap186 := d39
		snap187 := d40
		snap188 := d41
		snap189 := d42
		snap190 := d43
		snap191 := d44
		snap192 := d45
		snap193 := d46
		snap194 := d47
		snap195 := d48
		snap196 := d49
		snap197 := d50
		snap198 := d51
		snap199 := d52
		snap200 := d53
		snap201 := d54
		snap202 := d55
		snap203 := d56
		snap204 := d59
		snap205 := d60
		snap206 := d61
		snap207 := d119
		snap208 := d120
		snap209 := d121
		snap210 := d122
		snap211 := d123
		snap212 := d124
		snap213 := d125
		snap214 := d126
		snap215 := d127
		snap216 := d128
		snap217 := d129
		snap218 := d130
		snap219 := d131
		snap220 := d132
		snap221 := d133
		snap222 := d134
		snap223 := d135
		snap224 := d136
		snap225 := d137
		snap226 := d138
		snap227 := d139
		snap228 := d140
		snap229 := d141
		snap230 := d142
		snap231 := d143
		snap232 := d144
		snap233 := d145
		snap234 := d146
		snap235 := d147
		snap236 := d150
		alloc237 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps152)
		}
		ctx.RestoreAllocState(alloc237)
		d5 = snap153
		d6 = snap154
		d7 = snap155
		d8 = snap156
		d9 = snap157
		d10 = snap158
		d11 = snap159
		d12 = snap160
		d13 = snap161
		d14 = snap162
		d15 = snap163
		d16 = snap164
		d17 = snap165
		d18 = snap166
		d19 = snap167
		d20 = snap168
		d21 = snap169
		d23 = snap170
		d24 = snap171
		d25 = snap172
		d26 = snap173
		d27 = snap174
		d28 = snap175
		d29 = snap176
		d30 = snap177
		d31 = snap178
		d32 = snap179
		d33 = snap180
		d34 = snap181
		d35 = snap182
		d36 = snap183
		d37 = snap184
		d38 = snap185
		d39 = snap186
		d40 = snap187
		d41 = snap188
		d42 = snap189
		d43 = snap190
		d44 = snap191
		d45 = snap192
		d46 = snap193
		d47 = snap194
		d48 = snap195
		d49 = snap196
		d50 = snap197
		d51 = snap198
		d52 = snap199
		d53 = snap200
		d54 = snap201
		d55 = snap202
		d56 = snap203
		d59 = snap204
		d60 = snap205
		d61 = snap206
		d119 = snap207
		d120 = snap208
		d121 = snap209
		d122 = snap210
		d123 = snap211
		d124 = snap212
		d125 = snap213
		d126 = snap214
		d127 = snap215
		d128 = snap216
		d129 = snap217
		d130 = snap218
		d131 = snap219
		d132 = snap220
		d133 = snap221
		d134 = snap222
		d135 = snap223
		d136 = snap224
		d137 = snap225
		d138 = snap226
		d139 = snap227
		d140 = snap228
		d141 = snap229
		d142 = snap230
		d143 = snap231
		d144 = snap232
		d145 = snap233
		d146 = snap234
		d147 = snap235
		d150 = snap236
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps151)
		}
		return result
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d238 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d238)
		}
		if d238.Loc == scm.LocImm {
			d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: d238.Type, Imm: scm.NewInt(int64(uint64(d238.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d238.Reg, 32)
			ctx.EmitShrRegImm8(d238.Reg, 32)
		}
		if d238.Loc == scm.LocReg && d5.Loc == scm.LocReg && d238.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d238)
		ctx.EmitStoreToStack(d238, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d238)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d239 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d239)
		}
		if d239.Loc == scm.LocImm {
			d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: d239.Type, Imm: scm.NewInt(int64(uint64(d239.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d239.Reg, 32)
			ctx.EmitShrRegImm8(d239.Reg, 32)
		}
		if d239.Loc == scm.LocReg && d5.Loc == scm.LocReg && d239.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d239)
		ctx.EmitStoreToStack(d239, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d239)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d240 = d6
			if d240.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d240)
			d241 = d240
			if d241.Loc == scm.LocImm {
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: d241.Type, Imm: scm.NewInt(int64(uint64(d241.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d241.Reg, 32)
				ctx.EmitShrRegImm8(d241.Reg, 32)
			}
			ctx.EmitStoreToStack(d241, int32(bbs[4].PhiBase)+int32(16))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps242 := scm.PhiState{General: ps.General}
		ps242.OverlayValues = make([]scm.JITValueDesc, 242)
		ps242.OverlayValues[5] = d5
		ps242.OverlayValues[6] = d6
		ps242.OverlayValues[7] = d7
		ps242.OverlayValues[8] = d8
		ps242.OverlayValues[9] = d9
		ps242.OverlayValues[10] = d10
		ps242.OverlayValues[11] = d11
		ps242.OverlayValues[12] = d12
		ps242.OverlayValues[13] = d13
		ps242.OverlayValues[14] = d14
		ps242.OverlayValues[15] = d15
		ps242.OverlayValues[16] = d16
		ps242.OverlayValues[17] = d17
		ps242.OverlayValues[18] = d18
		ps242.OverlayValues[19] = d19
		ps242.OverlayValues[20] = d20
		ps242.OverlayValues[21] = d21
		ps242.OverlayValues[23] = d23
		ps242.OverlayValues[24] = d24
		ps242.OverlayValues[25] = d25
		ps242.OverlayValues[26] = d26
		ps242.OverlayValues[27] = d27
		ps242.OverlayValues[28] = d28
		ps242.OverlayValues[29] = d29
		ps242.OverlayValues[30] = d30
		ps242.OverlayValues[31] = d31
		ps242.OverlayValues[32] = d32
		ps242.OverlayValues[33] = d33
		ps242.OverlayValues[34] = d34
		ps242.OverlayValues[35] = d35
		ps242.OverlayValues[36] = d36
		ps242.OverlayValues[37] = d37
		ps242.OverlayValues[38] = d38
		ps242.OverlayValues[39] = d39
		ps242.OverlayValues[40] = d40
		ps242.OverlayValues[41] = d41
		ps242.OverlayValues[42] = d42
		ps242.OverlayValues[43] = d43
		ps242.OverlayValues[44] = d44
		ps242.OverlayValues[45] = d45
		ps242.OverlayValues[46] = d46
		ps242.OverlayValues[47] = d47
		ps242.OverlayValues[48] = d48
		ps242.OverlayValues[49] = d49
		ps242.OverlayValues[50] = d50
		ps242.OverlayValues[51] = d51
		ps242.OverlayValues[52] = d52
		ps242.OverlayValues[53] = d53
		ps242.OverlayValues[54] = d54
		ps242.OverlayValues[55] = d55
		ps242.OverlayValues[56] = d56
		ps242.OverlayValues[59] = d59
		ps242.OverlayValues[60] = d60
		ps242.OverlayValues[61] = d61
		ps242.OverlayValues[119] = d119
		ps242.OverlayValues[120] = d120
		ps242.OverlayValues[121] = d121
		ps242.OverlayValues[122] = d122
		ps242.OverlayValues[123] = d123
		ps242.OverlayValues[124] = d124
		ps242.OverlayValues[125] = d125
		ps242.OverlayValues[126] = d126
		ps242.OverlayValues[127] = d127
		ps242.OverlayValues[128] = d128
		ps242.OverlayValues[129] = d129
		ps242.OverlayValues[130] = d130
		ps242.OverlayValues[131] = d131
		ps242.OverlayValues[132] = d132
		ps242.OverlayValues[133] = d133
		ps242.OverlayValues[134] = d134
		ps242.OverlayValues[135] = d135
		ps242.OverlayValues[136] = d136
		ps242.OverlayValues[137] = d137
		ps242.OverlayValues[138] = d138
		ps242.OverlayValues[139] = d139
		ps242.OverlayValues[140] = d140
		ps242.OverlayValues[141] = d141
		ps242.OverlayValues[142] = d142
		ps242.OverlayValues[143] = d143
		ps242.OverlayValues[144] = d144
		ps242.OverlayValues[145] = d145
		ps242.OverlayValues[146] = d146
		ps242.OverlayValues[147] = d147
		ps242.OverlayValues[150] = d150
		ps242.OverlayValues[238] = d238
		ps242.OverlayValues[239] = d239
		ps242.OverlayValues[240] = d240
		ps242.OverlayValues[241] = d241
		ps242.PhiValues = make([]scm.JITValueDesc, 3)
		d243 = d6
		ps242.PhiValues[1] = d243
		if ps242.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps242)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d244 := ps.PhiValues[0]
				ctx.EmitStoreToStack(d244, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d245 := ps.PhiValues[1]
				ctx.EmitStoreToStack(d245, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d246 := ps.PhiValues[2]
				ctx.EmitStoreToStack(d246, int32(bbs[4].PhiBase)+int32(32))
			}
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d9 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d10 = ps.PhiValues[1]
		}
		if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
			d11 = ps.PhiValues[2]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d9)
		ctx.StabilizeDescForControlFlow(&d10)
		ctx.StabilizeDescForControlFlow(&d11)
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d11)
		ctx.EnsureDescsTogether(&d10, &d11)
		var d247 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d10.Imm.Int()) == uint64(d11.Imm.Int()))}
		} else if d11.Loc == scm.LocImm {
			r80 := ctx.AllocRegExcept(d10.Reg)
			if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d10.Reg, int32(d11.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.EmitCmpInt64(d10.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r80, scm.CondEqual)
			d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r80}
			ctx.BindReg(r80, &d247)
		} else if d10.Loc == scm.LocImm {
			r81 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d11.Reg)
			ctx.EmitSetcc(r81, scm.CondEqual)
			d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r81}
			ctx.BindReg(r81, &d247)
		} else {
			r82 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitCmpInt64(d10.Reg, d11.Reg)
			ctx.EmitSetcc(r82, scm.CondEqual)
			d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r82}
			ctx.BindReg(r82, &d247)
		}
		d248 = d247
		ctx.EnsureDesc(&d248)
		if d248.Loc != scm.LocImm && d248.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d248.Loc == scm.LocImm {
			if d248.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d10)
					if d10.Loc == scm.LocReg {
						ctx.ProtectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.ProtectReg(d10.Reg)
						ctx.ProtectReg(d10.Reg2)
					}
					d249 = d10
					if d249.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d249)
					d250 = d249
					if d250.Loc == scm.LocImm {
						d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: d250.Type, Imm: scm.NewInt(int64(uint64(d250.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d250.Reg, 32)
						ctx.EmitShrRegImm8(d250.Reg, 32)
					}
					ctx.EmitStoreToStack(d250, int32(bbs[2].PhiBase)+int32(0))
					if d10.Loc == scm.LocReg {
						ctx.UnprotectReg(d10.Reg)
					} else if d10.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d10.Reg)
						ctx.UnprotectReg(d10.Reg2)
					}
				}
				ps251 := scm.PhiState{General: ps.General}
				ps251.OverlayValues = make([]scm.JITValueDesc, 251)
				ps251.OverlayValues[5] = d5
				ps251.OverlayValues[6] = d6
				ps251.OverlayValues[7] = d7
				ps251.OverlayValues[8] = d8
				ps251.OverlayValues[9] = d9
				ps251.OverlayValues[10] = d10
				ps251.OverlayValues[11] = d11
				ps251.OverlayValues[12] = d12
				ps251.OverlayValues[13] = d13
				ps251.OverlayValues[14] = d14
				ps251.OverlayValues[15] = d15
				ps251.OverlayValues[16] = d16
				ps251.OverlayValues[17] = d17
				ps251.OverlayValues[18] = d18
				ps251.OverlayValues[19] = d19
				ps251.OverlayValues[20] = d20
				ps251.OverlayValues[21] = d21
				ps251.OverlayValues[23] = d23
				ps251.OverlayValues[24] = d24
				ps251.OverlayValues[25] = d25
				ps251.OverlayValues[26] = d26
				ps251.OverlayValues[27] = d27
				ps251.OverlayValues[28] = d28
				ps251.OverlayValues[29] = d29
				ps251.OverlayValues[30] = d30
				ps251.OverlayValues[31] = d31
				ps251.OverlayValues[32] = d32
				ps251.OverlayValues[33] = d33
				ps251.OverlayValues[34] = d34
				ps251.OverlayValues[35] = d35
				ps251.OverlayValues[36] = d36
				ps251.OverlayValues[37] = d37
				ps251.OverlayValues[38] = d38
				ps251.OverlayValues[39] = d39
				ps251.OverlayValues[40] = d40
				ps251.OverlayValues[41] = d41
				ps251.OverlayValues[42] = d42
				ps251.OverlayValues[43] = d43
				ps251.OverlayValues[44] = d44
				ps251.OverlayValues[45] = d45
				ps251.OverlayValues[46] = d46
				ps251.OverlayValues[47] = d47
				ps251.OverlayValues[48] = d48
				ps251.OverlayValues[49] = d49
				ps251.OverlayValues[50] = d50
				ps251.OverlayValues[51] = d51
				ps251.OverlayValues[52] = d52
				ps251.OverlayValues[53] = d53
				ps251.OverlayValues[54] = d54
				ps251.OverlayValues[55] = d55
				ps251.OverlayValues[56] = d56
				ps251.OverlayValues[59] = d59
				ps251.OverlayValues[60] = d60
				ps251.OverlayValues[61] = d61
				ps251.OverlayValues[119] = d119
				ps251.OverlayValues[120] = d120
				ps251.OverlayValues[121] = d121
				ps251.OverlayValues[122] = d122
				ps251.OverlayValues[123] = d123
				ps251.OverlayValues[124] = d124
				ps251.OverlayValues[125] = d125
				ps251.OverlayValues[126] = d126
				ps251.OverlayValues[127] = d127
				ps251.OverlayValues[128] = d128
				ps251.OverlayValues[129] = d129
				ps251.OverlayValues[130] = d130
				ps251.OverlayValues[131] = d131
				ps251.OverlayValues[132] = d132
				ps251.OverlayValues[133] = d133
				ps251.OverlayValues[134] = d134
				ps251.OverlayValues[135] = d135
				ps251.OverlayValues[136] = d136
				ps251.OverlayValues[137] = d137
				ps251.OverlayValues[138] = d138
				ps251.OverlayValues[139] = d139
				ps251.OverlayValues[140] = d140
				ps251.OverlayValues[141] = d141
				ps251.OverlayValues[142] = d142
				ps251.OverlayValues[143] = d143
				ps251.OverlayValues[144] = d144
				ps251.OverlayValues[145] = d145
				ps251.OverlayValues[146] = d146
				ps251.OverlayValues[147] = d147
				ps251.OverlayValues[150] = d150
				ps251.OverlayValues[238] = d238
				ps251.OverlayValues[239] = d239
				ps251.OverlayValues[240] = d240
				ps251.OverlayValues[241] = d241
				ps251.OverlayValues[243] = d243
				ps251.OverlayValues[244] = d244
				ps251.OverlayValues[245] = d245
				ps251.OverlayValues[246] = d246
				ps251.OverlayValues[247] = d247
				ps251.OverlayValues[248] = d248
				ps251.OverlayValues[249] = d249
				ps251.OverlayValues[250] = d250
				ps251.PhiValues = make([]scm.JITValueDesc, 1)
				d252 = d10
				ps251.PhiValues[0] = d252
				return bbs[2].RenderPS(ps251)
			}
			if ps.General {
			}
			ps253 := scm.PhiState{General: ps.General}
			ps253.OverlayValues = make([]scm.JITValueDesc, 253)
			ps253.OverlayValues[5] = d5
			ps253.OverlayValues[6] = d6
			ps253.OverlayValues[7] = d7
			ps253.OverlayValues[8] = d8
			ps253.OverlayValues[9] = d9
			ps253.OverlayValues[10] = d10
			ps253.OverlayValues[11] = d11
			ps253.OverlayValues[12] = d12
			ps253.OverlayValues[13] = d13
			ps253.OverlayValues[14] = d14
			ps253.OverlayValues[15] = d15
			ps253.OverlayValues[16] = d16
			ps253.OverlayValues[17] = d17
			ps253.OverlayValues[18] = d18
			ps253.OverlayValues[19] = d19
			ps253.OverlayValues[20] = d20
			ps253.OverlayValues[21] = d21
			ps253.OverlayValues[23] = d23
			ps253.OverlayValues[24] = d24
			ps253.OverlayValues[25] = d25
			ps253.OverlayValues[26] = d26
			ps253.OverlayValues[27] = d27
			ps253.OverlayValues[28] = d28
			ps253.OverlayValues[29] = d29
			ps253.OverlayValues[30] = d30
			ps253.OverlayValues[31] = d31
			ps253.OverlayValues[32] = d32
			ps253.OverlayValues[33] = d33
			ps253.OverlayValues[34] = d34
			ps253.OverlayValues[35] = d35
			ps253.OverlayValues[36] = d36
			ps253.OverlayValues[37] = d37
			ps253.OverlayValues[38] = d38
			ps253.OverlayValues[39] = d39
			ps253.OverlayValues[40] = d40
			ps253.OverlayValues[41] = d41
			ps253.OverlayValues[42] = d42
			ps253.OverlayValues[43] = d43
			ps253.OverlayValues[44] = d44
			ps253.OverlayValues[45] = d45
			ps253.OverlayValues[46] = d46
			ps253.OverlayValues[47] = d47
			ps253.OverlayValues[48] = d48
			ps253.OverlayValues[49] = d49
			ps253.OverlayValues[50] = d50
			ps253.OverlayValues[51] = d51
			ps253.OverlayValues[52] = d52
			ps253.OverlayValues[53] = d53
			ps253.OverlayValues[54] = d54
			ps253.OverlayValues[55] = d55
			ps253.OverlayValues[56] = d56
			ps253.OverlayValues[59] = d59
			ps253.OverlayValues[60] = d60
			ps253.OverlayValues[61] = d61
			ps253.OverlayValues[119] = d119
			ps253.OverlayValues[120] = d120
			ps253.OverlayValues[121] = d121
			ps253.OverlayValues[122] = d122
			ps253.OverlayValues[123] = d123
			ps253.OverlayValues[124] = d124
			ps253.OverlayValues[125] = d125
			ps253.OverlayValues[126] = d126
			ps253.OverlayValues[127] = d127
			ps253.OverlayValues[128] = d128
			ps253.OverlayValues[129] = d129
			ps253.OverlayValues[130] = d130
			ps253.OverlayValues[131] = d131
			ps253.OverlayValues[132] = d132
			ps253.OverlayValues[133] = d133
			ps253.OverlayValues[134] = d134
			ps253.OverlayValues[135] = d135
			ps253.OverlayValues[136] = d136
			ps253.OverlayValues[137] = d137
			ps253.OverlayValues[138] = d138
			ps253.OverlayValues[139] = d139
			ps253.OverlayValues[140] = d140
			ps253.OverlayValues[141] = d141
			ps253.OverlayValues[142] = d142
			ps253.OverlayValues[143] = d143
			ps253.OverlayValues[144] = d144
			ps253.OverlayValues[145] = d145
			ps253.OverlayValues[146] = d146
			ps253.OverlayValues[147] = d147
			ps253.OverlayValues[150] = d150
			ps253.OverlayValues[238] = d238
			ps253.OverlayValues[239] = d239
			ps253.OverlayValues[240] = d240
			ps253.OverlayValues[241] = d241
			ps253.OverlayValues[243] = d243
			ps253.OverlayValues[244] = d244
			ps253.OverlayValues[245] = d245
			ps253.OverlayValues[246] = d246
			ps253.OverlayValues[247] = d247
			ps253.OverlayValues[248] = d248
			ps253.OverlayValues[249] = d249
			ps253.OverlayValues[250] = d250
			ps253.OverlayValues[252] = d252
			return bbs[6].RenderPS(ps253)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d254 := ps.PhiValues[0]
				ctx.EmitStoreToStack(d254, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d255 := ps.PhiValues[1]
				ctx.EmitStoreToStack(d255, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d256 := ps.PhiValues[2]
				ctx.EmitStoreToStack(d256, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl21 := ctx.ReserveLabel()
		lbl22 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d248.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl21)
		ctx.EmitJmp(lbl22)
		ctx.MarkLabel(lbl21)
		ctx.SyncDesc(&d10)
		if d10.Loc == scm.LocReg {
			ctx.ProtectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.ProtectReg(d10.Reg)
			ctx.ProtectReg(d10.Reg2)
		}
		d257 = d10
		if d257.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d257)
		d258 = d257
		if d258.Loc == scm.LocImm {
			d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: d258.Type, Imm: scm.NewInt(int64(uint64(d258.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d258.Reg, 32)
			ctx.EmitShrRegImm8(d258.Reg, 32)
		}
		ctx.EmitStoreToStack(d258, int32(bbs[2].PhiBase)+int32(0))
		if d10.Loc == scm.LocReg {
			ctx.UnprotectReg(d10.Reg)
		} else if d10.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d10.Reg)
			ctx.UnprotectReg(d10.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl22)
		ctx.EmitJmp(lbl7)
		ps259 := scm.PhiState{General: true}
		ps259.OverlayValues = make([]scm.JITValueDesc, 259)
		ps259.OverlayValues[5] = d5
		ps259.OverlayValues[6] = d6
		ps259.OverlayValues[7] = d7
		ps259.OverlayValues[8] = d8
		ps259.OverlayValues[9] = d9
		ps259.OverlayValues[10] = d10
		ps259.OverlayValues[11] = d11
		ps259.OverlayValues[12] = d12
		ps259.OverlayValues[13] = d13
		ps259.OverlayValues[14] = d14
		ps259.OverlayValues[15] = d15
		ps259.OverlayValues[16] = d16
		ps259.OverlayValues[17] = d17
		ps259.OverlayValues[18] = d18
		ps259.OverlayValues[19] = d19
		ps259.OverlayValues[20] = d20
		ps259.OverlayValues[21] = d21
		ps259.OverlayValues[23] = d23
		ps259.OverlayValues[24] = d24
		ps259.OverlayValues[25] = d25
		ps259.OverlayValues[26] = d26
		ps259.OverlayValues[27] = d27
		ps259.OverlayValues[28] = d28
		ps259.OverlayValues[29] = d29
		ps259.OverlayValues[30] = d30
		ps259.OverlayValues[31] = d31
		ps259.OverlayValues[32] = d32
		ps259.OverlayValues[33] = d33
		ps259.OverlayValues[34] = d34
		ps259.OverlayValues[35] = d35
		ps259.OverlayValues[36] = d36
		ps259.OverlayValues[37] = d37
		ps259.OverlayValues[38] = d38
		ps259.OverlayValues[39] = d39
		ps259.OverlayValues[40] = d40
		ps259.OverlayValues[41] = d41
		ps259.OverlayValues[42] = d42
		ps259.OverlayValues[43] = d43
		ps259.OverlayValues[44] = d44
		ps259.OverlayValues[45] = d45
		ps259.OverlayValues[46] = d46
		ps259.OverlayValues[47] = d47
		ps259.OverlayValues[48] = d48
		ps259.OverlayValues[49] = d49
		ps259.OverlayValues[50] = d50
		ps259.OverlayValues[51] = d51
		ps259.OverlayValues[52] = d52
		ps259.OverlayValues[53] = d53
		ps259.OverlayValues[54] = d54
		ps259.OverlayValues[55] = d55
		ps259.OverlayValues[56] = d56
		ps259.OverlayValues[59] = d59
		ps259.OverlayValues[60] = d60
		ps259.OverlayValues[61] = d61
		ps259.OverlayValues[119] = d119
		ps259.OverlayValues[120] = d120
		ps259.OverlayValues[121] = d121
		ps259.OverlayValues[122] = d122
		ps259.OverlayValues[123] = d123
		ps259.OverlayValues[124] = d124
		ps259.OverlayValues[125] = d125
		ps259.OverlayValues[126] = d126
		ps259.OverlayValues[127] = d127
		ps259.OverlayValues[128] = d128
		ps259.OverlayValues[129] = d129
		ps259.OverlayValues[130] = d130
		ps259.OverlayValues[131] = d131
		ps259.OverlayValues[132] = d132
		ps259.OverlayValues[133] = d133
		ps259.OverlayValues[134] = d134
		ps259.OverlayValues[135] = d135
		ps259.OverlayValues[136] = d136
		ps259.OverlayValues[137] = d137
		ps259.OverlayValues[138] = d138
		ps259.OverlayValues[139] = d139
		ps259.OverlayValues[140] = d140
		ps259.OverlayValues[141] = d141
		ps259.OverlayValues[142] = d142
		ps259.OverlayValues[143] = d143
		ps259.OverlayValues[144] = d144
		ps259.OverlayValues[145] = d145
		ps259.OverlayValues[146] = d146
		ps259.OverlayValues[147] = d147
		ps259.OverlayValues[150] = d150
		ps259.OverlayValues[238] = d238
		ps259.OverlayValues[239] = d239
		ps259.OverlayValues[240] = d240
		ps259.OverlayValues[241] = d241
		ps259.OverlayValues[243] = d243
		ps259.OverlayValues[244] = d244
		ps259.OverlayValues[245] = d245
		ps259.OverlayValues[246] = d246
		ps259.OverlayValues[247] = d247
		ps259.OverlayValues[248] = d248
		ps259.OverlayValues[249] = d249
		ps259.OverlayValues[250] = d250
		ps259.OverlayValues[252] = d252
		ps259.OverlayValues[254] = d254
		ps259.OverlayValues[255] = d255
		ps259.OverlayValues[256] = d256
		ps259.OverlayValues[257] = d257
		ps259.OverlayValues[258] = d258
		ps259.PhiValues = make([]scm.JITValueDesc, 1)
		d261 = d10
		ps259.PhiValues[0] = d261
		ps260 := scm.PhiState{General: true}
		ps260.OverlayValues = make([]scm.JITValueDesc, 262)
		ps260.OverlayValues[5] = d5
		ps260.OverlayValues[6] = d6
		ps260.OverlayValues[7] = d7
		ps260.OverlayValues[8] = d8
		ps260.OverlayValues[9] = d9
		ps260.OverlayValues[10] = d10
		ps260.OverlayValues[11] = d11
		ps260.OverlayValues[12] = d12
		ps260.OverlayValues[13] = d13
		ps260.OverlayValues[14] = d14
		ps260.OverlayValues[15] = d15
		ps260.OverlayValues[16] = d16
		ps260.OverlayValues[17] = d17
		ps260.OverlayValues[18] = d18
		ps260.OverlayValues[19] = d19
		ps260.OverlayValues[20] = d20
		ps260.OverlayValues[21] = d21
		ps260.OverlayValues[23] = d23
		ps260.OverlayValues[24] = d24
		ps260.OverlayValues[25] = d25
		ps260.OverlayValues[26] = d26
		ps260.OverlayValues[27] = d27
		ps260.OverlayValues[28] = d28
		ps260.OverlayValues[29] = d29
		ps260.OverlayValues[30] = d30
		ps260.OverlayValues[31] = d31
		ps260.OverlayValues[32] = d32
		ps260.OverlayValues[33] = d33
		ps260.OverlayValues[34] = d34
		ps260.OverlayValues[35] = d35
		ps260.OverlayValues[36] = d36
		ps260.OverlayValues[37] = d37
		ps260.OverlayValues[38] = d38
		ps260.OverlayValues[39] = d39
		ps260.OverlayValues[40] = d40
		ps260.OverlayValues[41] = d41
		ps260.OverlayValues[42] = d42
		ps260.OverlayValues[43] = d43
		ps260.OverlayValues[44] = d44
		ps260.OverlayValues[45] = d45
		ps260.OverlayValues[46] = d46
		ps260.OverlayValues[47] = d47
		ps260.OverlayValues[48] = d48
		ps260.OverlayValues[49] = d49
		ps260.OverlayValues[50] = d50
		ps260.OverlayValues[51] = d51
		ps260.OverlayValues[52] = d52
		ps260.OverlayValues[53] = d53
		ps260.OverlayValues[54] = d54
		ps260.OverlayValues[55] = d55
		ps260.OverlayValues[56] = d56
		ps260.OverlayValues[59] = d59
		ps260.OverlayValues[60] = d60
		ps260.OverlayValues[61] = d61
		ps260.OverlayValues[119] = d119
		ps260.OverlayValues[120] = d120
		ps260.OverlayValues[121] = d121
		ps260.OverlayValues[122] = d122
		ps260.OverlayValues[123] = d123
		ps260.OverlayValues[124] = d124
		ps260.OverlayValues[125] = d125
		ps260.OverlayValues[126] = d126
		ps260.OverlayValues[127] = d127
		ps260.OverlayValues[128] = d128
		ps260.OverlayValues[129] = d129
		ps260.OverlayValues[130] = d130
		ps260.OverlayValues[131] = d131
		ps260.OverlayValues[132] = d132
		ps260.OverlayValues[133] = d133
		ps260.OverlayValues[134] = d134
		ps260.OverlayValues[135] = d135
		ps260.OverlayValues[136] = d136
		ps260.OverlayValues[137] = d137
		ps260.OverlayValues[138] = d138
		ps260.OverlayValues[139] = d139
		ps260.OverlayValues[140] = d140
		ps260.OverlayValues[141] = d141
		ps260.OverlayValues[142] = d142
		ps260.OverlayValues[143] = d143
		ps260.OverlayValues[144] = d144
		ps260.OverlayValues[145] = d145
		ps260.OverlayValues[146] = d146
		ps260.OverlayValues[147] = d147
		ps260.OverlayValues[150] = d150
		ps260.OverlayValues[238] = d238
		ps260.OverlayValues[239] = d239
		ps260.OverlayValues[240] = d240
		ps260.OverlayValues[241] = d241
		ps260.OverlayValues[243] = d243
		ps260.OverlayValues[244] = d244
		ps260.OverlayValues[245] = d245
		ps260.OverlayValues[246] = d246
		ps260.OverlayValues[247] = d247
		ps260.OverlayValues[248] = d248
		ps260.OverlayValues[249] = d249
		ps260.OverlayValues[250] = d250
		ps260.OverlayValues[252] = d252
		ps260.OverlayValues[254] = d254
		ps260.OverlayValues[255] = d255
		ps260.OverlayValues[256] = d256
		ps260.OverlayValues[257] = d257
		ps260.OverlayValues[258] = d258
		ps260.OverlayValues[261] = d261
		snap262 := d5
		snap263 := d6
		snap264 := d7
		snap265 := d8
		snap266 := d9
		snap267 := d10
		snap268 := d11
		snap269 := d12
		snap270 := d13
		snap271 := d14
		snap272 := d15
		snap273 := d16
		snap274 := d17
		snap275 := d18
		snap276 := d19
		snap277 := d20
		snap278 := d21
		snap279 := d23
		snap280 := d24
		snap281 := d25
		snap282 := d26
		snap283 := d27
		snap284 := d28
		snap285 := d29
		snap286 := d30
		snap287 := d31
		snap288 := d32
		snap289 := d33
		snap290 := d34
		snap291 := d35
		snap292 := d36
		snap293 := d37
		snap294 := d38
		snap295 := d39
		snap296 := d40
		snap297 := d41
		snap298 := d42
		snap299 := d43
		snap300 := d44
		snap301 := d45
		snap302 := d46
		snap303 := d47
		snap304 := d48
		snap305 := d49
		snap306 := d50
		snap307 := d51
		snap308 := d52
		snap309 := d53
		snap310 := d54
		snap311 := d55
		snap312 := d56
		snap313 := d59
		snap314 := d60
		snap315 := d61
		snap316 := d119
		snap317 := d120
		snap318 := d121
		snap319 := d122
		snap320 := d123
		snap321 := d124
		snap322 := d125
		snap323 := d126
		snap324 := d127
		snap325 := d128
		snap326 := d129
		snap327 := d130
		snap328 := d131
		snap329 := d132
		snap330 := d133
		snap331 := d134
		snap332 := d135
		snap333 := d136
		snap334 := d137
		snap335 := d138
		snap336 := d139
		snap337 := d140
		snap338 := d141
		snap339 := d142
		snap340 := d143
		snap341 := d144
		snap342 := d145
		snap343 := d146
		snap344 := d147
		snap345 := d150
		snap346 := d238
		snap347 := d239
		snap348 := d240
		snap349 := d241
		snap350 := d243
		snap351 := d244
		snap352 := d245
		snap353 := d246
		snap354 := d247
		snap355 := d248
		snap356 := d249
		snap357 := d250
		snap358 := d252
		snap359 := d254
		snap360 := d255
		snap361 := d256
		snap362 := d257
		snap363 := d258
		snap364 := d261
		alloc365 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps259)
		}
		ctx.RestoreAllocState(alloc365)
		d5 = snap262
		d6 = snap263
		d7 = snap264
		d8 = snap265
		d9 = snap266
		d10 = snap267
		d11 = snap268
		d12 = snap269
		d13 = snap270
		d14 = snap271
		d15 = snap272
		d16 = snap273
		d17 = snap274
		d18 = snap275
		d19 = snap276
		d20 = snap277
		d21 = snap278
		d23 = snap279
		d24 = snap280
		d25 = snap281
		d26 = snap282
		d27 = snap283
		d28 = snap284
		d29 = snap285
		d30 = snap286
		d31 = snap287
		d32 = snap288
		d33 = snap289
		d34 = snap290
		d35 = snap291
		d36 = snap292
		d37 = snap293
		d38 = snap294
		d39 = snap295
		d40 = snap296
		d41 = snap297
		d42 = snap298
		d43 = snap299
		d44 = snap300
		d45 = snap301
		d46 = snap302
		d47 = snap303
		d48 = snap304
		d49 = snap305
		d50 = snap306
		d51 = snap307
		d52 = snap308
		d53 = snap309
		d54 = snap310
		d55 = snap311
		d56 = snap312
		d59 = snap313
		d60 = snap314
		d61 = snap315
		d119 = snap316
		d120 = snap317
		d121 = snap318
		d122 = snap319
		d123 = snap320
		d124 = snap321
		d125 = snap322
		d126 = snap323
		d127 = snap324
		d128 = snap325
		d129 = snap326
		d130 = snap327
		d131 = snap328
		d132 = snap329
		d133 = snap330
		d134 = snap331
		d135 = snap332
		d136 = snap333
		d137 = snap334
		d138 = snap335
		d139 = snap336
		d140 = snap337
		d141 = snap338
		d142 = snap339
		d143 = snap340
		d144 = snap341
		d145 = snap342
		d146 = snap343
		d147 = snap344
		d150 = snap345
		d238 = snap346
		d239 = snap347
		d240 = snap348
		d241 = snap349
		d243 = snap350
		d244 = snap351
		d245 = snap352
		d246 = snap353
		d247 = snap354
		d248 = snap355
		d249 = snap356
		d250 = snap357
		d252 = snap358
		d254 = snap359
		d255 = snap360
		d256 = snap361
		d257 = snap362
		d258 = snap363
		d261 = snap364
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps260)
		}
		return result
		ctx.FreeDesc(&d247)
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d366 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d366 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d366 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d366)
		}
		if d366.Loc == scm.LocImm {
			d366 = scm.JITValueDesc{Loc: scm.LocImm, Type: d366.Type, Imm: scm.NewInt(int64(uint64(d366.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d366.Reg, 32)
			ctx.EmitShrRegImm8(d366.Reg, 32)
		}
		if d366.Loc == scm.LocReg && d5.Loc == scm.LocReg && d366.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d366)
		ctx.EmitStoreToStack(d366, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d366)
		if ps.General {
			ctx.SyncDesc(&d5)
			if d5.Loc == scm.LocReg {
				ctx.ProtectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.ProtectReg(d5.Reg)
				ctx.ProtectReg(d5.Reg2)
			}
			ctx.SyncDesc(&d7)
			if d7.Loc == scm.LocReg {
				ctx.ProtectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.ProtectReg(d7.Reg)
				ctx.ProtectReg(d7.Reg2)
			}
			d367 = d5
			if d367.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d367)
			d368 = d367
			if d368.Loc == scm.LocImm {
				d368 = scm.JITValueDesc{Loc: scm.LocImm, Type: d368.Type, Imm: scm.NewInt(int64(uint64(d368.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d368.Reg, 32)
				ctx.EmitShrRegImm8(d368.Reg, 32)
			}
			ctx.EmitStoreToStack(d368, int32(bbs[4].PhiBase)+int32(16))
			d369 = d7
			if d369.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d369)
			d370 = d369
			if d370.Loc == scm.LocImm {
				d370 = scm.JITValueDesc{Loc: scm.LocImm, Type: d370.Type, Imm: scm.NewInt(int64(uint64(d370.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d370.Reg, 32)
				ctx.EmitShrRegImm8(d370.Reg, 32)
			}
			ctx.EmitStoreToStack(d370, int32(bbs[4].PhiBase)+int32(32))
			if d5.Loc == scm.LocReg {
				ctx.UnprotectReg(d5.Reg)
			} else if d5.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d5.Reg)
				ctx.UnprotectReg(d5.Reg2)
			}
			if d7.Loc == scm.LocReg {
				ctx.UnprotectReg(d7.Reg)
			} else if d7.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d7.Reg)
				ctx.UnprotectReg(d7.Reg2)
			}
		}
		ps371 := scm.PhiState{General: ps.General}
		ps371.OverlayValues = make([]scm.JITValueDesc, 371)
		ps371.OverlayValues[5] = d5
		ps371.OverlayValues[6] = d6
		ps371.OverlayValues[7] = d7
		ps371.OverlayValues[8] = d8
		ps371.OverlayValues[9] = d9
		ps371.OverlayValues[10] = d10
		ps371.OverlayValues[11] = d11
		ps371.OverlayValues[12] = d12
		ps371.OverlayValues[13] = d13
		ps371.OverlayValues[14] = d14
		ps371.OverlayValues[15] = d15
		ps371.OverlayValues[16] = d16
		ps371.OverlayValues[17] = d17
		ps371.OverlayValues[18] = d18
		ps371.OverlayValues[19] = d19
		ps371.OverlayValues[20] = d20
		ps371.OverlayValues[21] = d21
		ps371.OverlayValues[23] = d23
		ps371.OverlayValues[24] = d24
		ps371.OverlayValues[25] = d25
		ps371.OverlayValues[26] = d26
		ps371.OverlayValues[27] = d27
		ps371.OverlayValues[28] = d28
		ps371.OverlayValues[29] = d29
		ps371.OverlayValues[30] = d30
		ps371.OverlayValues[31] = d31
		ps371.OverlayValues[32] = d32
		ps371.OverlayValues[33] = d33
		ps371.OverlayValues[34] = d34
		ps371.OverlayValues[35] = d35
		ps371.OverlayValues[36] = d36
		ps371.OverlayValues[37] = d37
		ps371.OverlayValues[38] = d38
		ps371.OverlayValues[39] = d39
		ps371.OverlayValues[40] = d40
		ps371.OverlayValues[41] = d41
		ps371.OverlayValues[42] = d42
		ps371.OverlayValues[43] = d43
		ps371.OverlayValues[44] = d44
		ps371.OverlayValues[45] = d45
		ps371.OverlayValues[46] = d46
		ps371.OverlayValues[47] = d47
		ps371.OverlayValues[48] = d48
		ps371.OverlayValues[49] = d49
		ps371.OverlayValues[50] = d50
		ps371.OverlayValues[51] = d51
		ps371.OverlayValues[52] = d52
		ps371.OverlayValues[53] = d53
		ps371.OverlayValues[54] = d54
		ps371.OverlayValues[55] = d55
		ps371.OverlayValues[56] = d56
		ps371.OverlayValues[59] = d59
		ps371.OverlayValues[60] = d60
		ps371.OverlayValues[61] = d61
		ps371.OverlayValues[119] = d119
		ps371.OverlayValues[120] = d120
		ps371.OverlayValues[121] = d121
		ps371.OverlayValues[122] = d122
		ps371.OverlayValues[123] = d123
		ps371.OverlayValues[124] = d124
		ps371.OverlayValues[125] = d125
		ps371.OverlayValues[126] = d126
		ps371.OverlayValues[127] = d127
		ps371.OverlayValues[128] = d128
		ps371.OverlayValues[129] = d129
		ps371.OverlayValues[130] = d130
		ps371.OverlayValues[131] = d131
		ps371.OverlayValues[132] = d132
		ps371.OverlayValues[133] = d133
		ps371.OverlayValues[134] = d134
		ps371.OverlayValues[135] = d135
		ps371.OverlayValues[136] = d136
		ps371.OverlayValues[137] = d137
		ps371.OverlayValues[138] = d138
		ps371.OverlayValues[139] = d139
		ps371.OverlayValues[140] = d140
		ps371.OverlayValues[141] = d141
		ps371.OverlayValues[142] = d142
		ps371.OverlayValues[143] = d143
		ps371.OverlayValues[144] = d144
		ps371.OverlayValues[145] = d145
		ps371.OverlayValues[146] = d146
		ps371.OverlayValues[147] = d147
		ps371.OverlayValues[150] = d150
		ps371.OverlayValues[238] = d238
		ps371.OverlayValues[239] = d239
		ps371.OverlayValues[240] = d240
		ps371.OverlayValues[241] = d241
		ps371.OverlayValues[243] = d243
		ps371.OverlayValues[244] = d244
		ps371.OverlayValues[245] = d245
		ps371.OverlayValues[246] = d246
		ps371.OverlayValues[247] = d247
		ps371.OverlayValues[248] = d248
		ps371.OverlayValues[249] = d249
		ps371.OverlayValues[250] = d250
		ps371.OverlayValues[252] = d252
		ps371.OverlayValues[254] = d254
		ps371.OverlayValues[255] = d255
		ps371.OverlayValues[256] = d256
		ps371.OverlayValues[257] = d257
		ps371.OverlayValues[258] = d258
		ps371.OverlayValues[261] = d261
		ps371.OverlayValues[366] = d366
		ps371.OverlayValues[367] = d367
		ps371.OverlayValues[368] = d368
		ps371.OverlayValues[369] = d369
		ps371.OverlayValues[370] = d370
		ps371.PhiValues = make([]scm.JITValueDesc, 3)
		d372 = d5
		ps371.PhiValues[1] = d372
		d373 = d7
		ps371.PhiValues[2] = d373
		if ps371.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps371)
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		d374 = d9
		_ = d374
		ctx.StabilizeDescForControlFlow(&d374)
		ctx.StabilizeDescForControlFlow(&d9)
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl23 := ctx.ReserveLabel()
		_ = lbl23
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl23)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d374)
		ctx.EnsureDesc(&d374)
		var d375 scm.JITValueDesc
		if d374.Loc == scm.LocImm {
			d375 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d374.Imm.Int()))))}
		} else {
			r83 := ctx.AllocReg()
			ctx.EmitMovRegReg(r83, d374.Reg)
			ctx.EmitShlRegImm8(r83, 32)
			ctx.EmitShrRegImm8(r83, 32)
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d375)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d376 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r84 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r84, fieldAddr)
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r84}
			ctx.BindReg(r84, &d376)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r85 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r85, thisptr.Reg, off)
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
			ctx.BindReg(r85, &d376)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d376)
		ctx.EnsureDesc(&d376)
		var d377 scm.JITValueDesc
		if d376.Loc == scm.LocImm {
			d377 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d376.Imm.Int()))))}
		} else {
			r86 := ctx.AllocReg()
			ctx.EmitMovRegReg(r86, d376.Reg)
			ctx.EmitShlRegImm8(r86, 56)
			ctx.EmitShrRegImm8(r86, 56)
			d377 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d377)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d375)
		ctx.EnsureDesc(&d377)
		ctx.EnsureDescsTogether(&d375, &d377)
		var d378 scm.JITValueDesc
		if d375.Loc == scm.LocImm && d377.Loc == scm.LocImm {
			d378 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d375.Imm.Int() * d377.Imm.Int())}
		} else if d375.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d377.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d375.Imm.Int()))
			ctx.EmitImulInt64(scratch, d377.Reg)
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d378)
		} else if d377.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d375.Reg)
			ctx.EmitMovRegReg(scratch, d375.Reg)
			if d377.Imm.Int() >= -2147483648 && d377.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d377.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d377.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d378)
		} else {
			r87 := ctx.AllocRegExcept(d375.Reg, d377.Reg)
			ctx.EmitMovRegReg(r87, d375.Reg)
			ctx.EmitImulInt64(r87, d377.Reg)
			d378 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d378)
		}
		if d378.Loc == scm.LocReg && d375.Loc == scm.LocReg && d378.Reg == d375.Reg {
			ctx.TransferReg(d375.Reg)
			d375.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d375)
		ctx.FreeDesc(&d377)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d378)
		var d379 scm.JITValueDesc
		if d378.Loc == scm.LocImm {
			d379 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d378.Imm.Int() / 64)}
		} else {
			r88 := ctx.AllocRegExcept(d378.Reg)
			ctx.EmitMovRegReg(r88, d378.Reg)
			ctx.EmitShrRegImm8(r88, 6)
			d379 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
			ctx.BindReg(r88, &d379)
		}
		if d379.Loc == scm.LocReg && d378.Loc == scm.LocReg && d379.Reg == d378.Reg {
			ctx.TransferReg(d378.Reg)
			d378.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d378)
		var d380 scm.JITValueDesc
		if d378.Loc == scm.LocImm {
			d380 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d378.Imm.Int() % 64)}
		} else {
			r89 := ctx.AllocRegExcept(d378.Reg)
			ctx.EmitMovRegReg(r89, d378.Reg)
			ctx.EmitAndRegImm32(r89, 63)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d380)
		}
		if d380.Loc == scm.LocReg && d378.Loc == scm.LocReg && d380.Reg == d378.Reg {
			ctx.TransferReg(d378.Reg)
			d378.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d378)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d381 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r90 := ctx.AllocReg()
			r91 := ctx.AllocRegExcept(r90)
			r92 := ctx.AllocRegExcept(r90, r91)
			ctx.EmitMovRegMem64(r90, fieldAddr)
			ctx.EmitMovRegMem64(r91, fieldAddr+8)
			ctx.EmitMovRegMem64(r92, fieldAddr+16)
			d381 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r90, Reg2: r91, Reg3: r92}
			ctx.BindReg(r90, &d381)
			ctx.BindReg(r91, &d381)
			ctx.BindReg(r92, &d381)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r93 := ctx.AllocReg()
			r94 := ctx.AllocRegExcept(r93)
			r95 := ctx.AllocRegExcept(r93, r94)
			ctx.EmitMovRegMem(r93, thisptr.Reg, off)
			ctx.EmitMovRegMem(r94, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r95, thisptr.Reg, off+16)
			d381 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r93, Reg2: r94, Reg3: r95}
			ctx.BindReg(r93, &d381)
			ctx.BindReg(r94, &d381)
			ctx.BindReg(r95, &d381)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d379)
		ctx.ReclaimUntrackedRegs()
		d383 = ctx.EmitSliceElementAddress(&d381, &d379, 8)
		ctx.EnsureDesc(&d383)
		ctx.EmitMovRegMem(d383.Reg, d383.Reg, 0)
		d382 = d383
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d382)
		ctx.EnsureDesc(&d380)
		var d384 scm.JITValueDesc
		if d382.Loc == scm.LocImm && d380.Loc == scm.LocImm {
			d384 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d382.Imm.Int()) << uint64(d380.Imm.Int())))}
		} else if d380.Loc == scm.LocImm {
			r96 := ctx.AllocRegExcept(d382.Reg)
			ctx.EmitMovRegReg(r96, d382.Reg)
			ctx.EmitShlRegImm8(r96, uint8(d380.Imm.Int()))
			d384 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d384)
		} else {
			{
				shiftSrc := d382.Reg
				r97 := ctx.AllocRegExcept(d382.Reg)
				ctx.EmitMovRegReg(r97, d382.Reg)
				shiftSrc = r97
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d380.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d380.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d380.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d384 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d384)
			}
		}
		if d384.Loc == scm.LocReg && d382.Loc == scm.LocReg && d384.Reg == d382.Reg {
			ctx.TransferReg(d382.Reg)
			d382.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d382)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d379)
		ctx.EnsureDesc(&d379)
		var d385 scm.JITValueDesc
		if d379.Loc == scm.LocImm {
			d385 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d379.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d379.Reg)
			ctx.EmitMovRegReg(scratch, d379.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d385 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d385)
		}
		if d385.Loc == scm.LocReg && d379.Loc == scm.LocReg && d385.Reg == d379.Reg {
			ctx.TransferReg(d379.Reg)
			d379.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d379)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d385)
		ctx.ReclaimUntrackedRegs()
		d387 = ctx.EmitSliceElementAddress(&d381, &d385, 8)
		ctx.EnsureDesc(&d387)
		ctx.EmitMovRegMem(d387.Reg, d387.Reg, 0)
		d386 = d387
		ctx.FreeDesc(&d385)
		ctx.ReclaimUntrackedRegs()
		d388 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d380)
		ctx.EnsureDescsTogether(&d388, &d380)
		var d389 scm.JITValueDesc
		if d388.Loc == scm.LocImm && d380.Loc == scm.LocImm {
			d389 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d388.Imm.Int() - d380.Imm.Int())}
		} else if d380.Loc == scm.LocImm && d380.Imm.Int() == 0 {
			r98 := ctx.AllocRegExcept(d388.Reg)
			ctx.EmitMovRegReg(r98, d388.Reg)
			d389 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			ctx.BindReg(r98, &d389)
		} else if d388.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d380.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d388.Imm.Int()))
			ctx.EmitSubInt64(scratch, d380.Reg)
			d389 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d389)
		} else if d380.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d388.Reg)
			ctx.EmitMovRegReg(scratch, d388.Reg)
			if d380.Imm.Int() >= -2147483648 && d380.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d380.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d380.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d389 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d389)
		} else {
			r99 := ctx.AllocRegExcept(d388.Reg, d380.Reg)
			ctx.EmitMovRegReg(r99, d388.Reg)
			ctx.EmitSubInt64(r99, d380.Reg)
			d389 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
			ctx.BindReg(r99, &d389)
		}
		if d389.Loc == scm.LocReg && d388.Loc == scm.LocReg && d389.Reg == d388.Reg {
			ctx.TransferReg(d388.Reg)
			d388.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d380)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d386)
		ctx.EnsureDesc(&d389)
		var d390 scm.JITValueDesc
		if d386.Loc == scm.LocImm && d389.Loc == scm.LocImm {
			d390 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d386.Imm.Int()) >> uint64(d389.Imm.Int())))}
		} else if d389.Loc == scm.LocImm {
			r100 := ctx.AllocRegExcept(d386.Reg)
			ctx.EmitMovRegReg(r100, d386.Reg)
			ctx.EmitShrRegImm8(r100, uint8(d389.Imm.Int()))
			d390 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			ctx.BindReg(r100, &d390)
		} else {
			{
				shiftSrc := d386.Reg
				r101 := ctx.AllocRegExcept(d386.Reg)
				ctx.EmitMovRegReg(r101, d386.Reg)
				shiftSrc = r101
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d389.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d389.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d389.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d390 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d390)
			}
		}
		if d390.Loc == scm.LocReg && d386.Loc == scm.LocReg && d390.Reg == d386.Reg {
			ctx.TransferReg(d386.Reg)
			d386.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d386)
		ctx.FreeDesc(&d389)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d384)
		ctx.EnsureDesc(&d390)
		var d391 scm.JITValueDesc
		if d384.Loc == scm.LocImm && d390.Loc == scm.LocImm {
			d391 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d384.Imm.Int() | d390.Imm.Int())}
		} else if d384.Loc == scm.LocImm && d384.Imm.Int() == 0 {
			d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d390.Reg}
			ctx.BindReg(d390.Reg, &d391)
		} else if d390.Loc == scm.LocImm && d390.Imm.Int() == 0 {
			r102 := ctx.AllocRegExcept(d384.Reg)
			ctx.EmitMovRegReg(r102, d384.Reg)
			d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			ctx.BindReg(r102, &d391)
		} else if d384.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d390.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d384.Imm.Int()))
			ctx.EmitOrInt64(scratch, d390.Reg)
			d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d391)
		} else if d390.Loc == scm.LocImm {
			r103 := ctx.AllocRegExcept(d384.Reg)
			ctx.EmitMovRegReg(r103, d384.Reg)
			if d390.Imm.Int() >= -2147483648 && d390.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r103, int32(d390.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d390.Imm.Int()))
				ctx.EmitOrInt64(r103, scm.RegR11)
			}
			d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
			ctx.BindReg(r103, &d391)
		} else {
			r104 := ctx.AllocRegExcept(d384.Reg, d390.Reg)
			ctx.EmitMovRegReg(r104, d384.Reg)
			ctx.EmitOrInt64(r104, d390.Reg)
			d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			ctx.BindReg(r104, &d391)
		}
		if d391.Loc == scm.LocReg && d384.Loc == scm.LocReg && d391.Reg == d384.Reg {
			ctx.TransferReg(d384.Reg)
			d384.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d384)
		ctx.FreeDesc(&d390)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d376)
		ctx.EnsureDesc(&d376)
		var d392 scm.JITValueDesc
		if d376.Loc == scm.LocImm {
			d392 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d376.Imm.Int()))))}
		} else {
			r105 := ctx.AllocReg()
			ctx.EmitMovRegReg(r105, d376.Reg)
			ctx.EmitShlRegImm8(r105, 56)
			ctx.EmitShrRegImm8(r105, 56)
			d392 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			ctx.BindReg(r105, &d392)
		}
		ctx.ReclaimUntrackedRegs()
		d393 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d392)
		ctx.EnsureDescsTogether(&d393, &d392)
		var d394 scm.JITValueDesc
		if d393.Loc == scm.LocImm && d392.Loc == scm.LocImm {
			d394 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d393.Imm.Int() - d392.Imm.Int())}
		} else if d392.Loc == scm.LocImm && d392.Imm.Int() == 0 {
			r106 := ctx.AllocRegExcept(d393.Reg)
			ctx.EmitMovRegReg(r106, d393.Reg)
			d394 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			ctx.BindReg(r106, &d394)
		} else if d393.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d392.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d393.Imm.Int()))
			ctx.EmitSubInt64(scratch, d392.Reg)
			d394 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d394)
		} else if d392.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d393.Reg)
			ctx.EmitMovRegReg(scratch, d393.Reg)
			if d392.Imm.Int() >= -2147483648 && d392.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d392.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d392.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d394 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d394)
		} else {
			r107 := ctx.AllocRegExcept(d393.Reg, d392.Reg)
			ctx.EmitMovRegReg(r107, d393.Reg)
			ctx.EmitSubInt64(r107, d392.Reg)
			d394 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d394)
		}
		if d394.Loc == scm.LocReg && d393.Loc == scm.LocReg && d394.Reg == d393.Reg {
			ctx.TransferReg(d393.Reg)
			d393.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d392)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d391)
		ctx.EnsureDesc(&d394)
		var d395 scm.JITValueDesc
		if d391.Loc == scm.LocImm && d394.Loc == scm.LocImm {
			d395 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d391.Imm.Int()) >> uint64(d394.Imm.Int())))}
		} else if d394.Loc == scm.LocImm {
			r108 := ctx.AllocRegExcept(d391.Reg)
			ctx.EmitMovRegReg(r108, d391.Reg)
			ctx.EmitShrRegImm8(r108, uint8(d394.Imm.Int()))
			d395 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d395)
		} else {
			{
				shiftSrc := d391.Reg
				r109 := ctx.AllocRegExcept(d391.Reg)
				ctx.EmitMovRegReg(r109, d391.Reg)
				shiftSrc = r109
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d394.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d394.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d394.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d395 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d395)
			}
		}
		if d395.Loc == scm.LocReg && d391.Loc == scm.LocReg && d395.Reg == d391.Reg {
			ctx.TransferReg(d391.Reg)
			d391.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d391)
		ctx.FreeDesc(&d394)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d395)
		ctx.EnsureDesc(&d395)
		ctx.EnsureDesc(&d395)
		var d396 scm.JITValueDesc
		if d395.Loc == scm.LocImm {
			d396 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d395.Imm.Int()))))}
		} else {
			r110 := ctx.AllocReg()
			ctx.EmitMovRegReg(r110, d395.Reg)
			d396 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d396)
		}
		ctx.FreeDesc(&d395)
		ctx.EnsureDesc(&d396)
		ctx.EnsureDesc(&d52)
		ctx.EnsureDescsTogether(&d396, &d52)
		var d397 scm.JITValueDesc
		if d396.Loc == scm.LocImm && d52.Loc == scm.LocImm {
			d397 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d396.Imm.Int() + d52.Imm.Int())}
		} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
			r111 := ctx.AllocRegExcept(d396.Reg)
			ctx.EmitMovRegReg(r111, d396.Reg)
			d397 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			ctx.BindReg(r111, &d397)
		} else if d396.Loc == scm.LocImm && d396.Imm.Int() == 0 {
			d397 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			ctx.BindReg(d52.Reg, &d397)
		} else if d396.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d396.Imm.Int()))
			ctx.EmitAddInt64(scratch, d52.Reg)
			d397 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d397)
		} else if d52.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d396.Reg)
			ctx.EmitMovRegReg(scratch, d396.Reg)
			if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d52.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d397 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d397)
		} else {
			r112 := ctx.AllocRegExcept(d396.Reg, d52.Reg)
			ctx.EmitMovRegReg(r112, d396.Reg)
			ctx.EmitAddInt64(r112, d52.Reg)
			d397 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			ctx.BindReg(r112, &d397)
		}
		if d397.Loc == scm.LocReg && d396.Loc == scm.LocReg && d397.Reg == d396.Reg {
			ctx.TransferReg(d396.Reg)
			d396.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d396)
		ctx.EnsureDesc(&d397)
		ctx.EnsureDesc(&d397)
		var d398 scm.JITValueDesc
		if d397.Loc == scm.LocImm {
			d398 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d397.Imm.Int()))))}
		} else {
			r113 := ctx.AllocReg()
			ctx.EmitMovRegReg(r113, d397.Reg)
			ctx.EmitShlRegImm8(r113, 32)
			ctx.EmitShrRegImm8(r113, 32)
			d398 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
			ctx.BindReg(r113, &d398)
		}
		ctx.FreeDesc(&d397)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d398)
		ctx.EnsureDescsTogether(&idxInt, &d398)
		var d399 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d398.Loc == scm.LocImm {
			d399 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d398.Imm.Int()))}
		} else if d398.Loc == scm.LocImm {
			r114 := ctx.AllocRegExcept(idxInt.Reg)
			if d398.Imm.Int() >= -2147483648 && d398.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d398.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d398.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r114, scm.CondUnsignedBelow)
			d399 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r114}
			ctx.BindReg(r114, &d399)
		} else if idxInt.Loc == scm.LocImm {
			r115 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d398.Reg)
			ctx.EmitSetcc(r115, scm.CondUnsignedBelow)
			d399 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r115}
			ctx.BindReg(r115, &d399)
		} else {
			r116 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d398.Reg)
			ctx.EmitSetcc(r116, scm.CondUnsignedBelow)
			d399 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r116}
			ctx.BindReg(r116, &d399)
		}
		ctx.FreeDesc(&d398)
		d400 = d399
		ctx.EnsureDesc(&d400)
		if d400.Loc != scm.LocImm && d400.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d400.Loc == scm.LocImm {
			if d400.Imm.Bool() {
				if ps.General {
				}
				ps401 := scm.PhiState{General: ps.General}
				ps401.OverlayValues = make([]scm.JITValueDesc, 401)
				ps401.OverlayValues[5] = d5
				ps401.OverlayValues[6] = d6
				ps401.OverlayValues[7] = d7
				ps401.OverlayValues[8] = d8
				ps401.OverlayValues[9] = d9
				ps401.OverlayValues[10] = d10
				ps401.OverlayValues[11] = d11
				ps401.OverlayValues[12] = d12
				ps401.OverlayValues[13] = d13
				ps401.OverlayValues[14] = d14
				ps401.OverlayValues[15] = d15
				ps401.OverlayValues[16] = d16
				ps401.OverlayValues[17] = d17
				ps401.OverlayValues[18] = d18
				ps401.OverlayValues[19] = d19
				ps401.OverlayValues[20] = d20
				ps401.OverlayValues[21] = d21
				ps401.OverlayValues[23] = d23
				ps401.OverlayValues[24] = d24
				ps401.OverlayValues[25] = d25
				ps401.OverlayValues[26] = d26
				ps401.OverlayValues[27] = d27
				ps401.OverlayValues[28] = d28
				ps401.OverlayValues[29] = d29
				ps401.OverlayValues[30] = d30
				ps401.OverlayValues[31] = d31
				ps401.OverlayValues[32] = d32
				ps401.OverlayValues[33] = d33
				ps401.OverlayValues[34] = d34
				ps401.OverlayValues[35] = d35
				ps401.OverlayValues[36] = d36
				ps401.OverlayValues[37] = d37
				ps401.OverlayValues[38] = d38
				ps401.OverlayValues[39] = d39
				ps401.OverlayValues[40] = d40
				ps401.OverlayValues[41] = d41
				ps401.OverlayValues[42] = d42
				ps401.OverlayValues[43] = d43
				ps401.OverlayValues[44] = d44
				ps401.OverlayValues[45] = d45
				ps401.OverlayValues[46] = d46
				ps401.OverlayValues[47] = d47
				ps401.OverlayValues[48] = d48
				ps401.OverlayValues[49] = d49
				ps401.OverlayValues[50] = d50
				ps401.OverlayValues[51] = d51
				ps401.OverlayValues[52] = d52
				ps401.OverlayValues[53] = d53
				ps401.OverlayValues[54] = d54
				ps401.OverlayValues[55] = d55
				ps401.OverlayValues[56] = d56
				ps401.OverlayValues[59] = d59
				ps401.OverlayValues[60] = d60
				ps401.OverlayValues[61] = d61
				ps401.OverlayValues[119] = d119
				ps401.OverlayValues[120] = d120
				ps401.OverlayValues[121] = d121
				ps401.OverlayValues[122] = d122
				ps401.OverlayValues[123] = d123
				ps401.OverlayValues[124] = d124
				ps401.OverlayValues[125] = d125
				ps401.OverlayValues[126] = d126
				ps401.OverlayValues[127] = d127
				ps401.OverlayValues[128] = d128
				ps401.OverlayValues[129] = d129
				ps401.OverlayValues[130] = d130
				ps401.OverlayValues[131] = d131
				ps401.OverlayValues[132] = d132
				ps401.OverlayValues[133] = d133
				ps401.OverlayValues[134] = d134
				ps401.OverlayValues[135] = d135
				ps401.OverlayValues[136] = d136
				ps401.OverlayValues[137] = d137
				ps401.OverlayValues[138] = d138
				ps401.OverlayValues[139] = d139
				ps401.OverlayValues[140] = d140
				ps401.OverlayValues[141] = d141
				ps401.OverlayValues[142] = d142
				ps401.OverlayValues[143] = d143
				ps401.OverlayValues[144] = d144
				ps401.OverlayValues[145] = d145
				ps401.OverlayValues[146] = d146
				ps401.OverlayValues[147] = d147
				ps401.OverlayValues[150] = d150
				ps401.OverlayValues[238] = d238
				ps401.OverlayValues[239] = d239
				ps401.OverlayValues[240] = d240
				ps401.OverlayValues[241] = d241
				ps401.OverlayValues[243] = d243
				ps401.OverlayValues[244] = d244
				ps401.OverlayValues[245] = d245
				ps401.OverlayValues[246] = d246
				ps401.OverlayValues[247] = d247
				ps401.OverlayValues[248] = d248
				ps401.OverlayValues[249] = d249
				ps401.OverlayValues[250] = d250
				ps401.OverlayValues[252] = d252
				ps401.OverlayValues[254] = d254
				ps401.OverlayValues[255] = d255
				ps401.OverlayValues[256] = d256
				ps401.OverlayValues[257] = d257
				ps401.OverlayValues[258] = d258
				ps401.OverlayValues[261] = d261
				ps401.OverlayValues[366] = d366
				ps401.OverlayValues[367] = d367
				ps401.OverlayValues[368] = d368
				ps401.OverlayValues[369] = d369
				ps401.OverlayValues[370] = d370
				ps401.OverlayValues[372] = d372
				ps401.OverlayValues[373] = d373
				ps401.OverlayValues[374] = d374
				ps401.OverlayValues[375] = d375
				ps401.OverlayValues[376] = d376
				ps401.OverlayValues[377] = d377
				ps401.OverlayValues[378] = d378
				ps401.OverlayValues[379] = d379
				ps401.OverlayValues[380] = d380
				ps401.OverlayValues[381] = d381
				ps401.OverlayValues[382] = d382
				ps401.OverlayValues[383] = d383
				ps401.OverlayValues[384] = d384
				ps401.OverlayValues[385] = d385
				ps401.OverlayValues[386] = d386
				ps401.OverlayValues[387] = d387
				ps401.OverlayValues[388] = d388
				ps401.OverlayValues[389] = d389
				ps401.OverlayValues[390] = d390
				ps401.OverlayValues[391] = d391
				ps401.OverlayValues[392] = d392
				ps401.OverlayValues[393] = d393
				ps401.OverlayValues[394] = d394
				ps401.OverlayValues[395] = d395
				ps401.OverlayValues[396] = d396
				ps401.OverlayValues[397] = d397
				ps401.OverlayValues[398] = d398
				ps401.OverlayValues[399] = d399
				ps401.OverlayValues[400] = d400
				return bbs[7].RenderPS(ps401)
			}
			if ps.General {
			}
			ps402 := scm.PhiState{General: ps.General}
			ps402.OverlayValues = make([]scm.JITValueDesc, 401)
			ps402.OverlayValues[5] = d5
			ps402.OverlayValues[6] = d6
			ps402.OverlayValues[7] = d7
			ps402.OverlayValues[8] = d8
			ps402.OverlayValues[9] = d9
			ps402.OverlayValues[10] = d10
			ps402.OverlayValues[11] = d11
			ps402.OverlayValues[12] = d12
			ps402.OverlayValues[13] = d13
			ps402.OverlayValues[14] = d14
			ps402.OverlayValues[15] = d15
			ps402.OverlayValues[16] = d16
			ps402.OverlayValues[17] = d17
			ps402.OverlayValues[18] = d18
			ps402.OverlayValues[19] = d19
			ps402.OverlayValues[20] = d20
			ps402.OverlayValues[21] = d21
			ps402.OverlayValues[23] = d23
			ps402.OverlayValues[24] = d24
			ps402.OverlayValues[25] = d25
			ps402.OverlayValues[26] = d26
			ps402.OverlayValues[27] = d27
			ps402.OverlayValues[28] = d28
			ps402.OverlayValues[29] = d29
			ps402.OverlayValues[30] = d30
			ps402.OverlayValues[31] = d31
			ps402.OverlayValues[32] = d32
			ps402.OverlayValues[33] = d33
			ps402.OverlayValues[34] = d34
			ps402.OverlayValues[35] = d35
			ps402.OverlayValues[36] = d36
			ps402.OverlayValues[37] = d37
			ps402.OverlayValues[38] = d38
			ps402.OverlayValues[39] = d39
			ps402.OverlayValues[40] = d40
			ps402.OverlayValues[41] = d41
			ps402.OverlayValues[42] = d42
			ps402.OverlayValues[43] = d43
			ps402.OverlayValues[44] = d44
			ps402.OverlayValues[45] = d45
			ps402.OverlayValues[46] = d46
			ps402.OverlayValues[47] = d47
			ps402.OverlayValues[48] = d48
			ps402.OverlayValues[49] = d49
			ps402.OverlayValues[50] = d50
			ps402.OverlayValues[51] = d51
			ps402.OverlayValues[52] = d52
			ps402.OverlayValues[53] = d53
			ps402.OverlayValues[54] = d54
			ps402.OverlayValues[55] = d55
			ps402.OverlayValues[56] = d56
			ps402.OverlayValues[59] = d59
			ps402.OverlayValues[60] = d60
			ps402.OverlayValues[61] = d61
			ps402.OverlayValues[119] = d119
			ps402.OverlayValues[120] = d120
			ps402.OverlayValues[121] = d121
			ps402.OverlayValues[122] = d122
			ps402.OverlayValues[123] = d123
			ps402.OverlayValues[124] = d124
			ps402.OverlayValues[125] = d125
			ps402.OverlayValues[126] = d126
			ps402.OverlayValues[127] = d127
			ps402.OverlayValues[128] = d128
			ps402.OverlayValues[129] = d129
			ps402.OverlayValues[130] = d130
			ps402.OverlayValues[131] = d131
			ps402.OverlayValues[132] = d132
			ps402.OverlayValues[133] = d133
			ps402.OverlayValues[134] = d134
			ps402.OverlayValues[135] = d135
			ps402.OverlayValues[136] = d136
			ps402.OverlayValues[137] = d137
			ps402.OverlayValues[138] = d138
			ps402.OverlayValues[139] = d139
			ps402.OverlayValues[140] = d140
			ps402.OverlayValues[141] = d141
			ps402.OverlayValues[142] = d142
			ps402.OverlayValues[143] = d143
			ps402.OverlayValues[144] = d144
			ps402.OverlayValues[145] = d145
			ps402.OverlayValues[146] = d146
			ps402.OverlayValues[147] = d147
			ps402.OverlayValues[150] = d150
			ps402.OverlayValues[238] = d238
			ps402.OverlayValues[239] = d239
			ps402.OverlayValues[240] = d240
			ps402.OverlayValues[241] = d241
			ps402.OverlayValues[243] = d243
			ps402.OverlayValues[244] = d244
			ps402.OverlayValues[245] = d245
			ps402.OverlayValues[246] = d246
			ps402.OverlayValues[247] = d247
			ps402.OverlayValues[248] = d248
			ps402.OverlayValues[249] = d249
			ps402.OverlayValues[250] = d250
			ps402.OverlayValues[252] = d252
			ps402.OverlayValues[254] = d254
			ps402.OverlayValues[255] = d255
			ps402.OverlayValues[256] = d256
			ps402.OverlayValues[257] = d257
			ps402.OverlayValues[258] = d258
			ps402.OverlayValues[261] = d261
			ps402.OverlayValues[366] = d366
			ps402.OverlayValues[367] = d367
			ps402.OverlayValues[368] = d368
			ps402.OverlayValues[369] = d369
			ps402.OverlayValues[370] = d370
			ps402.OverlayValues[372] = d372
			ps402.OverlayValues[373] = d373
			ps402.OverlayValues[374] = d374
			ps402.OverlayValues[375] = d375
			ps402.OverlayValues[376] = d376
			ps402.OverlayValues[377] = d377
			ps402.OverlayValues[378] = d378
			ps402.OverlayValues[379] = d379
			ps402.OverlayValues[380] = d380
			ps402.OverlayValues[381] = d381
			ps402.OverlayValues[382] = d382
			ps402.OverlayValues[383] = d383
			ps402.OverlayValues[384] = d384
			ps402.OverlayValues[385] = d385
			ps402.OverlayValues[386] = d386
			ps402.OverlayValues[387] = d387
			ps402.OverlayValues[388] = d388
			ps402.OverlayValues[389] = d389
			ps402.OverlayValues[390] = d390
			ps402.OverlayValues[391] = d391
			ps402.OverlayValues[392] = d392
			ps402.OverlayValues[393] = d393
			ps402.OverlayValues[394] = d394
			ps402.OverlayValues[395] = d395
			ps402.OverlayValues[396] = d396
			ps402.OverlayValues[397] = d397
			ps402.OverlayValues[398] = d398
			ps402.OverlayValues[399] = d399
			ps402.OverlayValues[400] = d400
			return bbs[9].RenderPS(ps402)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d400.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl24)
		ctx.EmitJmp(lbl25)
		ctx.MarkLabel(lbl24)
		ctx.EmitJmp(lbl8)
		ctx.MarkLabel(lbl25)
		ctx.EmitJmp(lbl10)
		ps403 := scm.PhiState{General: true}
		ps403.OverlayValues = make([]scm.JITValueDesc, 401)
		ps403.OverlayValues[5] = d5
		ps403.OverlayValues[6] = d6
		ps403.OverlayValues[7] = d7
		ps403.OverlayValues[8] = d8
		ps403.OverlayValues[9] = d9
		ps403.OverlayValues[10] = d10
		ps403.OverlayValues[11] = d11
		ps403.OverlayValues[12] = d12
		ps403.OverlayValues[13] = d13
		ps403.OverlayValues[14] = d14
		ps403.OverlayValues[15] = d15
		ps403.OverlayValues[16] = d16
		ps403.OverlayValues[17] = d17
		ps403.OverlayValues[18] = d18
		ps403.OverlayValues[19] = d19
		ps403.OverlayValues[20] = d20
		ps403.OverlayValues[21] = d21
		ps403.OverlayValues[23] = d23
		ps403.OverlayValues[24] = d24
		ps403.OverlayValues[25] = d25
		ps403.OverlayValues[26] = d26
		ps403.OverlayValues[27] = d27
		ps403.OverlayValues[28] = d28
		ps403.OverlayValues[29] = d29
		ps403.OverlayValues[30] = d30
		ps403.OverlayValues[31] = d31
		ps403.OverlayValues[32] = d32
		ps403.OverlayValues[33] = d33
		ps403.OverlayValues[34] = d34
		ps403.OverlayValues[35] = d35
		ps403.OverlayValues[36] = d36
		ps403.OverlayValues[37] = d37
		ps403.OverlayValues[38] = d38
		ps403.OverlayValues[39] = d39
		ps403.OverlayValues[40] = d40
		ps403.OverlayValues[41] = d41
		ps403.OverlayValues[42] = d42
		ps403.OverlayValues[43] = d43
		ps403.OverlayValues[44] = d44
		ps403.OverlayValues[45] = d45
		ps403.OverlayValues[46] = d46
		ps403.OverlayValues[47] = d47
		ps403.OverlayValues[48] = d48
		ps403.OverlayValues[49] = d49
		ps403.OverlayValues[50] = d50
		ps403.OverlayValues[51] = d51
		ps403.OverlayValues[52] = d52
		ps403.OverlayValues[53] = d53
		ps403.OverlayValues[54] = d54
		ps403.OverlayValues[55] = d55
		ps403.OverlayValues[56] = d56
		ps403.OverlayValues[59] = d59
		ps403.OverlayValues[60] = d60
		ps403.OverlayValues[61] = d61
		ps403.OverlayValues[119] = d119
		ps403.OverlayValues[120] = d120
		ps403.OverlayValues[121] = d121
		ps403.OverlayValues[122] = d122
		ps403.OverlayValues[123] = d123
		ps403.OverlayValues[124] = d124
		ps403.OverlayValues[125] = d125
		ps403.OverlayValues[126] = d126
		ps403.OverlayValues[127] = d127
		ps403.OverlayValues[128] = d128
		ps403.OverlayValues[129] = d129
		ps403.OverlayValues[130] = d130
		ps403.OverlayValues[131] = d131
		ps403.OverlayValues[132] = d132
		ps403.OverlayValues[133] = d133
		ps403.OverlayValues[134] = d134
		ps403.OverlayValues[135] = d135
		ps403.OverlayValues[136] = d136
		ps403.OverlayValues[137] = d137
		ps403.OverlayValues[138] = d138
		ps403.OverlayValues[139] = d139
		ps403.OverlayValues[140] = d140
		ps403.OverlayValues[141] = d141
		ps403.OverlayValues[142] = d142
		ps403.OverlayValues[143] = d143
		ps403.OverlayValues[144] = d144
		ps403.OverlayValues[145] = d145
		ps403.OverlayValues[146] = d146
		ps403.OverlayValues[147] = d147
		ps403.OverlayValues[150] = d150
		ps403.OverlayValues[238] = d238
		ps403.OverlayValues[239] = d239
		ps403.OverlayValues[240] = d240
		ps403.OverlayValues[241] = d241
		ps403.OverlayValues[243] = d243
		ps403.OverlayValues[244] = d244
		ps403.OverlayValues[245] = d245
		ps403.OverlayValues[246] = d246
		ps403.OverlayValues[247] = d247
		ps403.OverlayValues[248] = d248
		ps403.OverlayValues[249] = d249
		ps403.OverlayValues[250] = d250
		ps403.OverlayValues[252] = d252
		ps403.OverlayValues[254] = d254
		ps403.OverlayValues[255] = d255
		ps403.OverlayValues[256] = d256
		ps403.OverlayValues[257] = d257
		ps403.OverlayValues[258] = d258
		ps403.OverlayValues[261] = d261
		ps403.OverlayValues[366] = d366
		ps403.OverlayValues[367] = d367
		ps403.OverlayValues[368] = d368
		ps403.OverlayValues[369] = d369
		ps403.OverlayValues[370] = d370
		ps403.OverlayValues[372] = d372
		ps403.OverlayValues[373] = d373
		ps403.OverlayValues[374] = d374
		ps403.OverlayValues[375] = d375
		ps403.OverlayValues[376] = d376
		ps403.OverlayValues[377] = d377
		ps403.OverlayValues[378] = d378
		ps403.OverlayValues[379] = d379
		ps403.OverlayValues[380] = d380
		ps403.OverlayValues[381] = d381
		ps403.OverlayValues[382] = d382
		ps403.OverlayValues[383] = d383
		ps403.OverlayValues[384] = d384
		ps403.OverlayValues[385] = d385
		ps403.OverlayValues[386] = d386
		ps403.OverlayValues[387] = d387
		ps403.OverlayValues[388] = d388
		ps403.OverlayValues[389] = d389
		ps403.OverlayValues[390] = d390
		ps403.OverlayValues[391] = d391
		ps403.OverlayValues[392] = d392
		ps403.OverlayValues[393] = d393
		ps403.OverlayValues[394] = d394
		ps403.OverlayValues[395] = d395
		ps403.OverlayValues[396] = d396
		ps403.OverlayValues[397] = d397
		ps403.OverlayValues[398] = d398
		ps403.OverlayValues[399] = d399
		ps403.OverlayValues[400] = d400
		ps404 := scm.PhiState{General: true}
		ps404.OverlayValues = make([]scm.JITValueDesc, 401)
		ps404.OverlayValues[5] = d5
		ps404.OverlayValues[6] = d6
		ps404.OverlayValues[7] = d7
		ps404.OverlayValues[8] = d8
		ps404.OverlayValues[9] = d9
		ps404.OverlayValues[10] = d10
		ps404.OverlayValues[11] = d11
		ps404.OverlayValues[12] = d12
		ps404.OverlayValues[13] = d13
		ps404.OverlayValues[14] = d14
		ps404.OverlayValues[15] = d15
		ps404.OverlayValues[16] = d16
		ps404.OverlayValues[17] = d17
		ps404.OverlayValues[18] = d18
		ps404.OverlayValues[19] = d19
		ps404.OverlayValues[20] = d20
		ps404.OverlayValues[21] = d21
		ps404.OverlayValues[23] = d23
		ps404.OverlayValues[24] = d24
		ps404.OverlayValues[25] = d25
		ps404.OverlayValues[26] = d26
		ps404.OverlayValues[27] = d27
		ps404.OverlayValues[28] = d28
		ps404.OverlayValues[29] = d29
		ps404.OverlayValues[30] = d30
		ps404.OverlayValues[31] = d31
		ps404.OverlayValues[32] = d32
		ps404.OverlayValues[33] = d33
		ps404.OverlayValues[34] = d34
		ps404.OverlayValues[35] = d35
		ps404.OverlayValues[36] = d36
		ps404.OverlayValues[37] = d37
		ps404.OverlayValues[38] = d38
		ps404.OverlayValues[39] = d39
		ps404.OverlayValues[40] = d40
		ps404.OverlayValues[41] = d41
		ps404.OverlayValues[42] = d42
		ps404.OverlayValues[43] = d43
		ps404.OverlayValues[44] = d44
		ps404.OverlayValues[45] = d45
		ps404.OverlayValues[46] = d46
		ps404.OverlayValues[47] = d47
		ps404.OverlayValues[48] = d48
		ps404.OverlayValues[49] = d49
		ps404.OverlayValues[50] = d50
		ps404.OverlayValues[51] = d51
		ps404.OverlayValues[52] = d52
		ps404.OverlayValues[53] = d53
		ps404.OverlayValues[54] = d54
		ps404.OverlayValues[55] = d55
		ps404.OverlayValues[56] = d56
		ps404.OverlayValues[59] = d59
		ps404.OverlayValues[60] = d60
		ps404.OverlayValues[61] = d61
		ps404.OverlayValues[119] = d119
		ps404.OverlayValues[120] = d120
		ps404.OverlayValues[121] = d121
		ps404.OverlayValues[122] = d122
		ps404.OverlayValues[123] = d123
		ps404.OverlayValues[124] = d124
		ps404.OverlayValues[125] = d125
		ps404.OverlayValues[126] = d126
		ps404.OverlayValues[127] = d127
		ps404.OverlayValues[128] = d128
		ps404.OverlayValues[129] = d129
		ps404.OverlayValues[130] = d130
		ps404.OverlayValues[131] = d131
		ps404.OverlayValues[132] = d132
		ps404.OverlayValues[133] = d133
		ps404.OverlayValues[134] = d134
		ps404.OverlayValues[135] = d135
		ps404.OverlayValues[136] = d136
		ps404.OverlayValues[137] = d137
		ps404.OverlayValues[138] = d138
		ps404.OverlayValues[139] = d139
		ps404.OverlayValues[140] = d140
		ps404.OverlayValues[141] = d141
		ps404.OverlayValues[142] = d142
		ps404.OverlayValues[143] = d143
		ps404.OverlayValues[144] = d144
		ps404.OverlayValues[145] = d145
		ps404.OverlayValues[146] = d146
		ps404.OverlayValues[147] = d147
		ps404.OverlayValues[150] = d150
		ps404.OverlayValues[238] = d238
		ps404.OverlayValues[239] = d239
		ps404.OverlayValues[240] = d240
		ps404.OverlayValues[241] = d241
		ps404.OverlayValues[243] = d243
		ps404.OverlayValues[244] = d244
		ps404.OverlayValues[245] = d245
		ps404.OverlayValues[246] = d246
		ps404.OverlayValues[247] = d247
		ps404.OverlayValues[248] = d248
		ps404.OverlayValues[249] = d249
		ps404.OverlayValues[250] = d250
		ps404.OverlayValues[252] = d252
		ps404.OverlayValues[254] = d254
		ps404.OverlayValues[255] = d255
		ps404.OverlayValues[256] = d256
		ps404.OverlayValues[257] = d257
		ps404.OverlayValues[258] = d258
		ps404.OverlayValues[261] = d261
		ps404.OverlayValues[366] = d366
		ps404.OverlayValues[367] = d367
		ps404.OverlayValues[368] = d368
		ps404.OverlayValues[369] = d369
		ps404.OverlayValues[370] = d370
		ps404.OverlayValues[372] = d372
		ps404.OverlayValues[373] = d373
		ps404.OverlayValues[374] = d374
		ps404.OverlayValues[375] = d375
		ps404.OverlayValues[376] = d376
		ps404.OverlayValues[377] = d377
		ps404.OverlayValues[378] = d378
		ps404.OverlayValues[379] = d379
		ps404.OverlayValues[380] = d380
		ps404.OverlayValues[381] = d381
		ps404.OverlayValues[382] = d382
		ps404.OverlayValues[383] = d383
		ps404.OverlayValues[384] = d384
		ps404.OverlayValues[385] = d385
		ps404.OverlayValues[386] = d386
		ps404.OverlayValues[387] = d387
		ps404.OverlayValues[388] = d388
		ps404.OverlayValues[389] = d389
		ps404.OverlayValues[390] = d390
		ps404.OverlayValues[391] = d391
		ps404.OverlayValues[392] = d392
		ps404.OverlayValues[393] = d393
		ps404.OverlayValues[394] = d394
		ps404.OverlayValues[395] = d395
		ps404.OverlayValues[396] = d396
		ps404.OverlayValues[397] = d397
		ps404.OverlayValues[398] = d398
		ps404.OverlayValues[399] = d399
		ps404.OverlayValues[400] = d400
		snap405 := d5
		snap406 := d6
		snap407 := d7
		snap408 := d8
		snap409 := d9
		snap410 := d10
		snap411 := d11
		snap412 := d12
		snap413 := d13
		snap414 := d14
		snap415 := d15
		snap416 := d16
		snap417 := d17
		snap418 := d18
		snap419 := d19
		snap420 := d20
		snap421 := d21
		snap422 := d23
		snap423 := d24
		snap424 := d25
		snap425 := d26
		snap426 := d27
		snap427 := d28
		snap428 := d29
		snap429 := d30
		snap430 := d31
		snap431 := d32
		snap432 := d33
		snap433 := d34
		snap434 := d35
		snap435 := d36
		snap436 := d37
		snap437 := d38
		snap438 := d39
		snap439 := d40
		snap440 := d41
		snap441 := d42
		snap442 := d43
		snap443 := d44
		snap444 := d45
		snap445 := d46
		snap446 := d47
		snap447 := d48
		snap448 := d49
		snap449 := d50
		snap450 := d51
		snap451 := d52
		snap452 := d53
		snap453 := d54
		snap454 := d55
		snap455 := d56
		snap456 := d59
		snap457 := d60
		snap458 := d61
		snap459 := d119
		snap460 := d120
		snap461 := d121
		snap462 := d122
		snap463 := d123
		snap464 := d124
		snap465 := d125
		snap466 := d126
		snap467 := d127
		snap468 := d128
		snap469 := d129
		snap470 := d130
		snap471 := d131
		snap472 := d132
		snap473 := d133
		snap474 := d134
		snap475 := d135
		snap476 := d136
		snap477 := d137
		snap478 := d138
		snap479 := d139
		snap480 := d140
		snap481 := d141
		snap482 := d142
		snap483 := d143
		snap484 := d144
		snap485 := d145
		snap486 := d146
		snap487 := d147
		snap488 := d150
		snap489 := d238
		snap490 := d239
		snap491 := d240
		snap492 := d241
		snap493 := d243
		snap494 := d244
		snap495 := d245
		snap496 := d246
		snap497 := d247
		snap498 := d248
		snap499 := d249
		snap500 := d250
		snap501 := d252
		snap502 := d254
		snap503 := d255
		snap504 := d256
		snap505 := d257
		snap506 := d258
		snap507 := d261
		snap508 := d366
		snap509 := d367
		snap510 := d368
		snap511 := d369
		snap512 := d370
		snap513 := d372
		snap514 := d373
		snap515 := d374
		snap516 := d375
		snap517 := d376
		snap518 := d377
		snap519 := d378
		snap520 := d379
		snap521 := d380
		snap522 := d381
		snap523 := d382
		snap524 := d383
		snap525 := d384
		snap526 := d385
		snap527 := d386
		snap528 := d387
		snap529 := d388
		snap530 := d389
		snap531 := d390
		snap532 := d391
		snap533 := d392
		snap534 := d393
		snap535 := d394
		snap536 := d395
		snap537 := d396
		snap538 := d397
		snap539 := d398
		snap540 := d399
		snap541 := d400
		alloc542 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps404)
		}
		ctx.RestoreAllocState(alloc542)
		d5 = snap405
		d6 = snap406
		d7 = snap407
		d8 = snap408
		d9 = snap409
		d10 = snap410
		d11 = snap411
		d12 = snap412
		d13 = snap413
		d14 = snap414
		d15 = snap415
		d16 = snap416
		d17 = snap417
		d18 = snap418
		d19 = snap419
		d20 = snap420
		d21 = snap421
		d23 = snap422
		d24 = snap423
		d25 = snap424
		d26 = snap425
		d27 = snap426
		d28 = snap427
		d29 = snap428
		d30 = snap429
		d31 = snap430
		d32 = snap431
		d33 = snap432
		d34 = snap433
		d35 = snap434
		d36 = snap435
		d37 = snap436
		d38 = snap437
		d39 = snap438
		d40 = snap439
		d41 = snap440
		d42 = snap441
		d43 = snap442
		d44 = snap443
		d45 = snap444
		d46 = snap445
		d47 = snap446
		d48 = snap447
		d49 = snap448
		d50 = snap449
		d51 = snap450
		d52 = snap451
		d53 = snap452
		d54 = snap453
		d55 = snap454
		d56 = snap455
		d59 = snap456
		d60 = snap457
		d61 = snap458
		d119 = snap459
		d120 = snap460
		d121 = snap461
		d122 = snap462
		d123 = snap463
		d124 = snap464
		d125 = snap465
		d126 = snap466
		d127 = snap467
		d128 = snap468
		d129 = snap469
		d130 = snap470
		d131 = snap471
		d132 = snap472
		d133 = snap473
		d134 = snap474
		d135 = snap475
		d136 = snap476
		d137 = snap477
		d138 = snap478
		d139 = snap479
		d140 = snap480
		d141 = snap481
		d142 = snap482
		d143 = snap483
		d144 = snap484
		d145 = snap485
		d146 = snap486
		d147 = snap487
		d150 = snap488
		d238 = snap489
		d239 = snap490
		d240 = snap491
		d241 = snap492
		d243 = snap493
		d244 = snap494
		d245 = snap495
		d246 = snap496
		d247 = snap497
		d248 = snap498
		d249 = snap499
		d250 = snap500
		d252 = snap501
		d254 = snap502
		d255 = snap503
		d256 = snap504
		d257 = snap505
		d258 = snap506
		d261 = snap507
		d366 = snap508
		d367 = snap509
		d368 = snap510
		d369 = snap511
		d370 = snap512
		d372 = snap513
		d373 = snap514
		d374 = snap515
		d375 = snap516
		d376 = snap517
		d377 = snap518
		d378 = snap519
		d379 = snap520
		d380 = snap521
		d381 = snap522
		d382 = snap523
		d383 = snap524
		d384 = snap525
		d385 = snap526
		d386 = snap527
		d387 = snap528
		d388 = snap529
		d389 = snap530
		d390 = snap531
		d391 = snap532
		d392 = snap533
		d393 = snap534
		d394 = snap535
		d395 = snap536
		d396 = snap537
		d397 = snap538
		d398 = snap539
		d399 = snap540
		d400 = snap541
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps403)
		}
		return result
		ctx.FreeDesc(&d399)
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
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != scm.LocNone {
			d374 = ps.OverlayValues[374]
		}
		if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != scm.LocNone {
			d375 = ps.OverlayValues[375]
		}
		if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != scm.LocNone {
			d376 = ps.OverlayValues[376]
		}
		if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != scm.LocNone {
			d377 = ps.OverlayValues[377]
		}
		if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != scm.LocNone {
			d378 = ps.OverlayValues[378]
		}
		if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != scm.LocNone {
			d379 = ps.OverlayValues[379]
		}
		if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
			d380 = ps.OverlayValues[380]
		}
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
		}
		if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
			d382 = ps.OverlayValues[382]
		}
		if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != scm.LocNone {
			d383 = ps.OverlayValues[383]
		}
		if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != scm.LocNone {
			d384 = ps.OverlayValues[384]
		}
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != scm.LocNone {
			d387 = ps.OverlayValues[387]
		}
		if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != scm.LocNone {
			d388 = ps.OverlayValues[388]
		}
		if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != scm.LocNone {
			d389 = ps.OverlayValues[389]
		}
		if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
			d390 = ps.OverlayValues[390]
		}
		if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != scm.LocNone {
			d391 = ps.OverlayValues[391]
		}
		if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != scm.LocNone {
			d392 = ps.OverlayValues[392]
		}
		if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != scm.LocNone {
			d393 = ps.OverlayValues[393]
		}
		if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != scm.LocNone {
			d394 = ps.OverlayValues[394]
		}
		if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != scm.LocNone {
			d395 = ps.OverlayValues[395]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d9)
		var d543 scm.JITValueDesc
		if d9.Loc == scm.LocImm {
			d543 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(scratch, d9.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d543 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d543)
		}
		if d543.Loc == scm.LocImm {
			d543 = scm.JITValueDesc{Loc: scm.LocImm, Type: d543.Type, Imm: scm.NewInt(int64(uint64(d543.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d543.Reg, 32)
			ctx.EmitShrRegImm8(d543.Reg, 32)
		}
		if d543.Loc == scm.LocReg && d9.Loc == scm.LocReg && d543.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d543)
		ctx.EmitStoreToStack(d543, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d543)
		if ps.General {
			ctx.SyncDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d544 = d10
			if d544.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d544)
			d545 = d544
			if d545.Loc == scm.LocImm {
				d545 = scm.JITValueDesc{Loc: scm.LocImm, Type: d545.Type, Imm: scm.NewInt(int64(uint64(d545.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d545.Reg, 32)
				ctx.EmitShrRegImm8(d545.Reg, 32)
			}
			ctx.EmitStoreToStack(d545, int32(bbs[8].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
		}
		ps546 := scm.PhiState{General: ps.General}
		ps546.OverlayValues = make([]scm.JITValueDesc, 546)
		ps546.OverlayValues[5] = d5
		ps546.OverlayValues[6] = d6
		ps546.OverlayValues[7] = d7
		ps546.OverlayValues[8] = d8
		ps546.OverlayValues[9] = d9
		ps546.OverlayValues[10] = d10
		ps546.OverlayValues[11] = d11
		ps546.OverlayValues[12] = d12
		ps546.OverlayValues[13] = d13
		ps546.OverlayValues[14] = d14
		ps546.OverlayValues[15] = d15
		ps546.OverlayValues[16] = d16
		ps546.OverlayValues[17] = d17
		ps546.OverlayValues[18] = d18
		ps546.OverlayValues[19] = d19
		ps546.OverlayValues[20] = d20
		ps546.OverlayValues[21] = d21
		ps546.OverlayValues[23] = d23
		ps546.OverlayValues[24] = d24
		ps546.OverlayValues[25] = d25
		ps546.OverlayValues[26] = d26
		ps546.OverlayValues[27] = d27
		ps546.OverlayValues[28] = d28
		ps546.OverlayValues[29] = d29
		ps546.OverlayValues[30] = d30
		ps546.OverlayValues[31] = d31
		ps546.OverlayValues[32] = d32
		ps546.OverlayValues[33] = d33
		ps546.OverlayValues[34] = d34
		ps546.OverlayValues[35] = d35
		ps546.OverlayValues[36] = d36
		ps546.OverlayValues[37] = d37
		ps546.OverlayValues[38] = d38
		ps546.OverlayValues[39] = d39
		ps546.OverlayValues[40] = d40
		ps546.OverlayValues[41] = d41
		ps546.OverlayValues[42] = d42
		ps546.OverlayValues[43] = d43
		ps546.OverlayValues[44] = d44
		ps546.OverlayValues[45] = d45
		ps546.OverlayValues[46] = d46
		ps546.OverlayValues[47] = d47
		ps546.OverlayValues[48] = d48
		ps546.OverlayValues[49] = d49
		ps546.OverlayValues[50] = d50
		ps546.OverlayValues[51] = d51
		ps546.OverlayValues[52] = d52
		ps546.OverlayValues[53] = d53
		ps546.OverlayValues[54] = d54
		ps546.OverlayValues[55] = d55
		ps546.OverlayValues[56] = d56
		ps546.OverlayValues[59] = d59
		ps546.OverlayValues[60] = d60
		ps546.OverlayValues[61] = d61
		ps546.OverlayValues[119] = d119
		ps546.OverlayValues[120] = d120
		ps546.OverlayValues[121] = d121
		ps546.OverlayValues[122] = d122
		ps546.OverlayValues[123] = d123
		ps546.OverlayValues[124] = d124
		ps546.OverlayValues[125] = d125
		ps546.OverlayValues[126] = d126
		ps546.OverlayValues[127] = d127
		ps546.OverlayValues[128] = d128
		ps546.OverlayValues[129] = d129
		ps546.OverlayValues[130] = d130
		ps546.OverlayValues[131] = d131
		ps546.OverlayValues[132] = d132
		ps546.OverlayValues[133] = d133
		ps546.OverlayValues[134] = d134
		ps546.OverlayValues[135] = d135
		ps546.OverlayValues[136] = d136
		ps546.OverlayValues[137] = d137
		ps546.OverlayValues[138] = d138
		ps546.OverlayValues[139] = d139
		ps546.OverlayValues[140] = d140
		ps546.OverlayValues[141] = d141
		ps546.OverlayValues[142] = d142
		ps546.OverlayValues[143] = d143
		ps546.OverlayValues[144] = d144
		ps546.OverlayValues[145] = d145
		ps546.OverlayValues[146] = d146
		ps546.OverlayValues[147] = d147
		ps546.OverlayValues[150] = d150
		ps546.OverlayValues[238] = d238
		ps546.OverlayValues[239] = d239
		ps546.OverlayValues[240] = d240
		ps546.OverlayValues[241] = d241
		ps546.OverlayValues[243] = d243
		ps546.OverlayValues[244] = d244
		ps546.OverlayValues[245] = d245
		ps546.OverlayValues[246] = d246
		ps546.OverlayValues[247] = d247
		ps546.OverlayValues[248] = d248
		ps546.OverlayValues[249] = d249
		ps546.OverlayValues[250] = d250
		ps546.OverlayValues[252] = d252
		ps546.OverlayValues[254] = d254
		ps546.OverlayValues[255] = d255
		ps546.OverlayValues[256] = d256
		ps546.OverlayValues[257] = d257
		ps546.OverlayValues[258] = d258
		ps546.OverlayValues[261] = d261
		ps546.OverlayValues[366] = d366
		ps546.OverlayValues[367] = d367
		ps546.OverlayValues[368] = d368
		ps546.OverlayValues[369] = d369
		ps546.OverlayValues[370] = d370
		ps546.OverlayValues[372] = d372
		ps546.OverlayValues[373] = d373
		ps546.OverlayValues[374] = d374
		ps546.OverlayValues[375] = d375
		ps546.OverlayValues[376] = d376
		ps546.OverlayValues[377] = d377
		ps546.OverlayValues[378] = d378
		ps546.OverlayValues[379] = d379
		ps546.OverlayValues[380] = d380
		ps546.OverlayValues[381] = d381
		ps546.OverlayValues[382] = d382
		ps546.OverlayValues[383] = d383
		ps546.OverlayValues[384] = d384
		ps546.OverlayValues[385] = d385
		ps546.OverlayValues[386] = d386
		ps546.OverlayValues[387] = d387
		ps546.OverlayValues[388] = d388
		ps546.OverlayValues[389] = d389
		ps546.OverlayValues[390] = d390
		ps546.OverlayValues[391] = d391
		ps546.OverlayValues[392] = d392
		ps546.OverlayValues[393] = d393
		ps546.OverlayValues[394] = d394
		ps546.OverlayValues[395] = d395
		ps546.OverlayValues[396] = d396
		ps546.OverlayValues[397] = d397
		ps546.OverlayValues[398] = d398
		ps546.OverlayValues[399] = d399
		ps546.OverlayValues[400] = d400
		ps546.OverlayValues[543] = d543
		ps546.OverlayValues[544] = d544
		ps546.OverlayValues[545] = d545
		ps546.PhiValues = make([]scm.JITValueDesc, 2)
		d547 = d10
		ps546.PhiValues[0] = d547
		if ps546.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps546)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d548 := ps.PhiValues[0]
				ctx.EmitStoreToStack(d548, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d549 := ps.PhiValues[1]
				ctx.EmitStoreToStack(d549, int32(bbs[8].PhiBase)+int32(16))
			}
			if bbs[8].VisitCount >= 0 {
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
		}
		bbs[8].VisitCount++
		if ps.General {
			if bbs[8].Rendered {
				ctx.EmitJmp(lbl9)
				return result
			}
			bbs[8].Rendered = true
			bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_8 = bbs[8].Address
			ctx.MarkLabel(lbl9)
			ctx.ResolveFixups()
		}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != scm.LocNone {
			d374 = ps.OverlayValues[374]
		}
		if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != scm.LocNone {
			d375 = ps.OverlayValues[375]
		}
		if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != scm.LocNone {
			d376 = ps.OverlayValues[376]
		}
		if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != scm.LocNone {
			d377 = ps.OverlayValues[377]
		}
		if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != scm.LocNone {
			d378 = ps.OverlayValues[378]
		}
		if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != scm.LocNone {
			d379 = ps.OverlayValues[379]
		}
		if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
			d380 = ps.OverlayValues[380]
		}
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
		}
		if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
			d382 = ps.OverlayValues[382]
		}
		if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != scm.LocNone {
			d383 = ps.OverlayValues[383]
		}
		if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != scm.LocNone {
			d384 = ps.OverlayValues[384]
		}
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != scm.LocNone {
			d387 = ps.OverlayValues[387]
		}
		if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != scm.LocNone {
			d388 = ps.OverlayValues[388]
		}
		if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != scm.LocNone {
			d389 = ps.OverlayValues[389]
		}
		if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
			d390 = ps.OverlayValues[390]
		}
		if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != scm.LocNone {
			d391 = ps.OverlayValues[391]
		}
		if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != scm.LocNone {
			d392 = ps.OverlayValues[392]
		}
		if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != scm.LocNone {
			d393 = ps.OverlayValues[393]
		}
		if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != scm.LocNone {
			d394 = ps.OverlayValues[394]
		}
		if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != scm.LocNone {
			d395 = ps.OverlayValues[395]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 543 && ps.OverlayValues[543].Loc != scm.LocNone {
			d543 = ps.OverlayValues[543]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 545 && ps.OverlayValues[545].Loc != scm.LocNone {
			d545 = ps.OverlayValues[545]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d12 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d13 = ps.PhiValues[1]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d12)
		ctx.StabilizeDescForControlFlow(&d13)
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d12, &d13)
		var d550 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d550 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d12.Imm.Int()) == uint64(d13.Imm.Int()))}
		} else if d13.Loc == scm.LocImm {
			r117 := ctx.AllocRegExcept(d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d12.Reg, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitCmpInt64(d12.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r117, scm.CondEqual)
			d550 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r117}
			ctx.BindReg(r117, &d550)
		} else if d12.Loc == scm.LocImm {
			r118 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d13.Reg)
			ctx.EmitSetcc(r118, scm.CondEqual)
			d550 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
			ctx.BindReg(r118, &d550)
		} else {
			r119 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitCmpInt64(d12.Reg, d13.Reg)
			ctx.EmitSetcc(r119, scm.CondEqual)
			d550 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r119}
			ctx.BindReg(r119, &d550)
		}
		d551 = d550
		ctx.EnsureDesc(&d551)
		if d551.Loc != scm.LocImm && d551.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d551.Loc == scm.LocImm {
			if d551.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d12)
					if d12.Loc == scm.LocReg {
						ctx.ProtectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.ProtectReg(d12.Reg)
						ctx.ProtectReg(d12.Reg2)
					}
					d552 = d12
					if d552.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d552)
					d553 = d552
					if d553.Loc == scm.LocImm {
						d553 = scm.JITValueDesc{Loc: scm.LocImm, Type: d553.Type, Imm: scm.NewInt(int64(uint64(d553.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d553.Reg, 32)
						ctx.EmitShrRegImm8(d553.Reg, 32)
					}
					ctx.EmitStoreToStack(d553, int32(bbs[2].PhiBase)+int32(0))
					if d12.Loc == scm.LocReg {
						ctx.UnprotectReg(d12.Reg)
					} else if d12.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d12.Reg)
						ctx.UnprotectReg(d12.Reg2)
					}
				}
				ps554 := scm.PhiState{General: ps.General}
				ps554.OverlayValues = make([]scm.JITValueDesc, 554)
				ps554.OverlayValues[5] = d5
				ps554.OverlayValues[6] = d6
				ps554.OverlayValues[7] = d7
				ps554.OverlayValues[8] = d8
				ps554.OverlayValues[9] = d9
				ps554.OverlayValues[10] = d10
				ps554.OverlayValues[11] = d11
				ps554.OverlayValues[12] = d12
				ps554.OverlayValues[13] = d13
				ps554.OverlayValues[14] = d14
				ps554.OverlayValues[15] = d15
				ps554.OverlayValues[16] = d16
				ps554.OverlayValues[17] = d17
				ps554.OverlayValues[18] = d18
				ps554.OverlayValues[19] = d19
				ps554.OverlayValues[20] = d20
				ps554.OverlayValues[21] = d21
				ps554.OverlayValues[23] = d23
				ps554.OverlayValues[24] = d24
				ps554.OverlayValues[25] = d25
				ps554.OverlayValues[26] = d26
				ps554.OverlayValues[27] = d27
				ps554.OverlayValues[28] = d28
				ps554.OverlayValues[29] = d29
				ps554.OverlayValues[30] = d30
				ps554.OverlayValues[31] = d31
				ps554.OverlayValues[32] = d32
				ps554.OverlayValues[33] = d33
				ps554.OverlayValues[34] = d34
				ps554.OverlayValues[35] = d35
				ps554.OverlayValues[36] = d36
				ps554.OverlayValues[37] = d37
				ps554.OverlayValues[38] = d38
				ps554.OverlayValues[39] = d39
				ps554.OverlayValues[40] = d40
				ps554.OverlayValues[41] = d41
				ps554.OverlayValues[42] = d42
				ps554.OverlayValues[43] = d43
				ps554.OverlayValues[44] = d44
				ps554.OverlayValues[45] = d45
				ps554.OverlayValues[46] = d46
				ps554.OverlayValues[47] = d47
				ps554.OverlayValues[48] = d48
				ps554.OverlayValues[49] = d49
				ps554.OverlayValues[50] = d50
				ps554.OverlayValues[51] = d51
				ps554.OverlayValues[52] = d52
				ps554.OverlayValues[53] = d53
				ps554.OverlayValues[54] = d54
				ps554.OverlayValues[55] = d55
				ps554.OverlayValues[56] = d56
				ps554.OverlayValues[59] = d59
				ps554.OverlayValues[60] = d60
				ps554.OverlayValues[61] = d61
				ps554.OverlayValues[119] = d119
				ps554.OverlayValues[120] = d120
				ps554.OverlayValues[121] = d121
				ps554.OverlayValues[122] = d122
				ps554.OverlayValues[123] = d123
				ps554.OverlayValues[124] = d124
				ps554.OverlayValues[125] = d125
				ps554.OverlayValues[126] = d126
				ps554.OverlayValues[127] = d127
				ps554.OverlayValues[128] = d128
				ps554.OverlayValues[129] = d129
				ps554.OverlayValues[130] = d130
				ps554.OverlayValues[131] = d131
				ps554.OverlayValues[132] = d132
				ps554.OverlayValues[133] = d133
				ps554.OverlayValues[134] = d134
				ps554.OverlayValues[135] = d135
				ps554.OverlayValues[136] = d136
				ps554.OverlayValues[137] = d137
				ps554.OverlayValues[138] = d138
				ps554.OverlayValues[139] = d139
				ps554.OverlayValues[140] = d140
				ps554.OverlayValues[141] = d141
				ps554.OverlayValues[142] = d142
				ps554.OverlayValues[143] = d143
				ps554.OverlayValues[144] = d144
				ps554.OverlayValues[145] = d145
				ps554.OverlayValues[146] = d146
				ps554.OverlayValues[147] = d147
				ps554.OverlayValues[150] = d150
				ps554.OverlayValues[238] = d238
				ps554.OverlayValues[239] = d239
				ps554.OverlayValues[240] = d240
				ps554.OverlayValues[241] = d241
				ps554.OverlayValues[243] = d243
				ps554.OverlayValues[244] = d244
				ps554.OverlayValues[245] = d245
				ps554.OverlayValues[246] = d246
				ps554.OverlayValues[247] = d247
				ps554.OverlayValues[248] = d248
				ps554.OverlayValues[249] = d249
				ps554.OverlayValues[250] = d250
				ps554.OverlayValues[252] = d252
				ps554.OverlayValues[254] = d254
				ps554.OverlayValues[255] = d255
				ps554.OverlayValues[256] = d256
				ps554.OverlayValues[257] = d257
				ps554.OverlayValues[258] = d258
				ps554.OverlayValues[261] = d261
				ps554.OverlayValues[366] = d366
				ps554.OverlayValues[367] = d367
				ps554.OverlayValues[368] = d368
				ps554.OverlayValues[369] = d369
				ps554.OverlayValues[370] = d370
				ps554.OverlayValues[372] = d372
				ps554.OverlayValues[373] = d373
				ps554.OverlayValues[374] = d374
				ps554.OverlayValues[375] = d375
				ps554.OverlayValues[376] = d376
				ps554.OverlayValues[377] = d377
				ps554.OverlayValues[378] = d378
				ps554.OverlayValues[379] = d379
				ps554.OverlayValues[380] = d380
				ps554.OverlayValues[381] = d381
				ps554.OverlayValues[382] = d382
				ps554.OverlayValues[383] = d383
				ps554.OverlayValues[384] = d384
				ps554.OverlayValues[385] = d385
				ps554.OverlayValues[386] = d386
				ps554.OverlayValues[387] = d387
				ps554.OverlayValues[388] = d388
				ps554.OverlayValues[389] = d389
				ps554.OverlayValues[390] = d390
				ps554.OverlayValues[391] = d391
				ps554.OverlayValues[392] = d392
				ps554.OverlayValues[393] = d393
				ps554.OverlayValues[394] = d394
				ps554.OverlayValues[395] = d395
				ps554.OverlayValues[396] = d396
				ps554.OverlayValues[397] = d397
				ps554.OverlayValues[398] = d398
				ps554.OverlayValues[399] = d399
				ps554.OverlayValues[400] = d400
				ps554.OverlayValues[543] = d543
				ps554.OverlayValues[544] = d544
				ps554.OverlayValues[545] = d545
				ps554.OverlayValues[547] = d547
				ps554.OverlayValues[548] = d548
				ps554.OverlayValues[549] = d549
				ps554.OverlayValues[550] = d550
				ps554.OverlayValues[551] = d551
				ps554.OverlayValues[552] = d552
				ps554.OverlayValues[553] = d553
				ps554.PhiValues = make([]scm.JITValueDesc, 1)
				d555 = d12
				ps554.PhiValues[0] = d555
				return bbs[2].RenderPS(ps554)
			}
			if ps.General {
			}
			ps556 := scm.PhiState{General: ps.General}
			ps556.OverlayValues = make([]scm.JITValueDesc, 556)
			ps556.OverlayValues[5] = d5
			ps556.OverlayValues[6] = d6
			ps556.OverlayValues[7] = d7
			ps556.OverlayValues[8] = d8
			ps556.OverlayValues[9] = d9
			ps556.OverlayValues[10] = d10
			ps556.OverlayValues[11] = d11
			ps556.OverlayValues[12] = d12
			ps556.OverlayValues[13] = d13
			ps556.OverlayValues[14] = d14
			ps556.OverlayValues[15] = d15
			ps556.OverlayValues[16] = d16
			ps556.OverlayValues[17] = d17
			ps556.OverlayValues[18] = d18
			ps556.OverlayValues[19] = d19
			ps556.OverlayValues[20] = d20
			ps556.OverlayValues[21] = d21
			ps556.OverlayValues[23] = d23
			ps556.OverlayValues[24] = d24
			ps556.OverlayValues[25] = d25
			ps556.OverlayValues[26] = d26
			ps556.OverlayValues[27] = d27
			ps556.OverlayValues[28] = d28
			ps556.OverlayValues[29] = d29
			ps556.OverlayValues[30] = d30
			ps556.OverlayValues[31] = d31
			ps556.OverlayValues[32] = d32
			ps556.OverlayValues[33] = d33
			ps556.OverlayValues[34] = d34
			ps556.OverlayValues[35] = d35
			ps556.OverlayValues[36] = d36
			ps556.OverlayValues[37] = d37
			ps556.OverlayValues[38] = d38
			ps556.OverlayValues[39] = d39
			ps556.OverlayValues[40] = d40
			ps556.OverlayValues[41] = d41
			ps556.OverlayValues[42] = d42
			ps556.OverlayValues[43] = d43
			ps556.OverlayValues[44] = d44
			ps556.OverlayValues[45] = d45
			ps556.OverlayValues[46] = d46
			ps556.OverlayValues[47] = d47
			ps556.OverlayValues[48] = d48
			ps556.OverlayValues[49] = d49
			ps556.OverlayValues[50] = d50
			ps556.OverlayValues[51] = d51
			ps556.OverlayValues[52] = d52
			ps556.OverlayValues[53] = d53
			ps556.OverlayValues[54] = d54
			ps556.OverlayValues[55] = d55
			ps556.OverlayValues[56] = d56
			ps556.OverlayValues[59] = d59
			ps556.OverlayValues[60] = d60
			ps556.OverlayValues[61] = d61
			ps556.OverlayValues[119] = d119
			ps556.OverlayValues[120] = d120
			ps556.OverlayValues[121] = d121
			ps556.OverlayValues[122] = d122
			ps556.OverlayValues[123] = d123
			ps556.OverlayValues[124] = d124
			ps556.OverlayValues[125] = d125
			ps556.OverlayValues[126] = d126
			ps556.OverlayValues[127] = d127
			ps556.OverlayValues[128] = d128
			ps556.OverlayValues[129] = d129
			ps556.OverlayValues[130] = d130
			ps556.OverlayValues[131] = d131
			ps556.OverlayValues[132] = d132
			ps556.OverlayValues[133] = d133
			ps556.OverlayValues[134] = d134
			ps556.OverlayValues[135] = d135
			ps556.OverlayValues[136] = d136
			ps556.OverlayValues[137] = d137
			ps556.OverlayValues[138] = d138
			ps556.OverlayValues[139] = d139
			ps556.OverlayValues[140] = d140
			ps556.OverlayValues[141] = d141
			ps556.OverlayValues[142] = d142
			ps556.OverlayValues[143] = d143
			ps556.OverlayValues[144] = d144
			ps556.OverlayValues[145] = d145
			ps556.OverlayValues[146] = d146
			ps556.OverlayValues[147] = d147
			ps556.OverlayValues[150] = d150
			ps556.OverlayValues[238] = d238
			ps556.OverlayValues[239] = d239
			ps556.OverlayValues[240] = d240
			ps556.OverlayValues[241] = d241
			ps556.OverlayValues[243] = d243
			ps556.OverlayValues[244] = d244
			ps556.OverlayValues[245] = d245
			ps556.OverlayValues[246] = d246
			ps556.OverlayValues[247] = d247
			ps556.OverlayValues[248] = d248
			ps556.OverlayValues[249] = d249
			ps556.OverlayValues[250] = d250
			ps556.OverlayValues[252] = d252
			ps556.OverlayValues[254] = d254
			ps556.OverlayValues[255] = d255
			ps556.OverlayValues[256] = d256
			ps556.OverlayValues[257] = d257
			ps556.OverlayValues[258] = d258
			ps556.OverlayValues[261] = d261
			ps556.OverlayValues[366] = d366
			ps556.OverlayValues[367] = d367
			ps556.OverlayValues[368] = d368
			ps556.OverlayValues[369] = d369
			ps556.OverlayValues[370] = d370
			ps556.OverlayValues[372] = d372
			ps556.OverlayValues[373] = d373
			ps556.OverlayValues[374] = d374
			ps556.OverlayValues[375] = d375
			ps556.OverlayValues[376] = d376
			ps556.OverlayValues[377] = d377
			ps556.OverlayValues[378] = d378
			ps556.OverlayValues[379] = d379
			ps556.OverlayValues[380] = d380
			ps556.OverlayValues[381] = d381
			ps556.OverlayValues[382] = d382
			ps556.OverlayValues[383] = d383
			ps556.OverlayValues[384] = d384
			ps556.OverlayValues[385] = d385
			ps556.OverlayValues[386] = d386
			ps556.OverlayValues[387] = d387
			ps556.OverlayValues[388] = d388
			ps556.OverlayValues[389] = d389
			ps556.OverlayValues[390] = d390
			ps556.OverlayValues[391] = d391
			ps556.OverlayValues[392] = d392
			ps556.OverlayValues[393] = d393
			ps556.OverlayValues[394] = d394
			ps556.OverlayValues[395] = d395
			ps556.OverlayValues[396] = d396
			ps556.OverlayValues[397] = d397
			ps556.OverlayValues[398] = d398
			ps556.OverlayValues[399] = d399
			ps556.OverlayValues[400] = d400
			ps556.OverlayValues[543] = d543
			ps556.OverlayValues[544] = d544
			ps556.OverlayValues[545] = d545
			ps556.OverlayValues[547] = d547
			ps556.OverlayValues[548] = d548
			ps556.OverlayValues[549] = d549
			ps556.OverlayValues[550] = d550
			ps556.OverlayValues[551] = d551
			ps556.OverlayValues[552] = d552
			ps556.OverlayValues[553] = d553
			ps556.OverlayValues[555] = d555
			return bbs[10].RenderPS(ps556)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d557 := ps.PhiValues[0]
				ctx.EmitStoreToStack(d557, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d558 := ps.PhiValues[1]
				ctx.EmitStoreToStack(d558, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl26 := ctx.ReserveLabel()
		lbl27 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d551.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl26)
		ctx.EmitJmp(lbl27)
		ctx.MarkLabel(lbl26)
		ctx.SyncDesc(&d12)
		if d12.Loc == scm.LocReg {
			ctx.ProtectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.ProtectReg(d12.Reg)
			ctx.ProtectReg(d12.Reg2)
		}
		d559 = d12
		if d559.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d559)
		d560 = d559
		if d560.Loc == scm.LocImm {
			d560 = scm.JITValueDesc{Loc: scm.LocImm, Type: d560.Type, Imm: scm.NewInt(int64(uint64(d560.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d560.Reg, 32)
			ctx.EmitShrRegImm8(d560.Reg, 32)
		}
		ctx.EmitStoreToStack(d560, int32(bbs[2].PhiBase)+int32(0))
		if d12.Loc == scm.LocReg {
			ctx.UnprotectReg(d12.Reg)
		} else if d12.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d12.Reg)
			ctx.UnprotectReg(d12.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl27)
		ctx.EmitJmp(lbl11)
		ps561 := scm.PhiState{General: true}
		ps561.OverlayValues = make([]scm.JITValueDesc, 561)
		ps561.OverlayValues[5] = d5
		ps561.OverlayValues[6] = d6
		ps561.OverlayValues[7] = d7
		ps561.OverlayValues[8] = d8
		ps561.OverlayValues[9] = d9
		ps561.OverlayValues[10] = d10
		ps561.OverlayValues[11] = d11
		ps561.OverlayValues[12] = d12
		ps561.OverlayValues[13] = d13
		ps561.OverlayValues[14] = d14
		ps561.OverlayValues[15] = d15
		ps561.OverlayValues[16] = d16
		ps561.OverlayValues[17] = d17
		ps561.OverlayValues[18] = d18
		ps561.OverlayValues[19] = d19
		ps561.OverlayValues[20] = d20
		ps561.OverlayValues[21] = d21
		ps561.OverlayValues[23] = d23
		ps561.OverlayValues[24] = d24
		ps561.OverlayValues[25] = d25
		ps561.OverlayValues[26] = d26
		ps561.OverlayValues[27] = d27
		ps561.OverlayValues[28] = d28
		ps561.OverlayValues[29] = d29
		ps561.OverlayValues[30] = d30
		ps561.OverlayValues[31] = d31
		ps561.OverlayValues[32] = d32
		ps561.OverlayValues[33] = d33
		ps561.OverlayValues[34] = d34
		ps561.OverlayValues[35] = d35
		ps561.OverlayValues[36] = d36
		ps561.OverlayValues[37] = d37
		ps561.OverlayValues[38] = d38
		ps561.OverlayValues[39] = d39
		ps561.OverlayValues[40] = d40
		ps561.OverlayValues[41] = d41
		ps561.OverlayValues[42] = d42
		ps561.OverlayValues[43] = d43
		ps561.OverlayValues[44] = d44
		ps561.OverlayValues[45] = d45
		ps561.OverlayValues[46] = d46
		ps561.OverlayValues[47] = d47
		ps561.OverlayValues[48] = d48
		ps561.OverlayValues[49] = d49
		ps561.OverlayValues[50] = d50
		ps561.OverlayValues[51] = d51
		ps561.OverlayValues[52] = d52
		ps561.OverlayValues[53] = d53
		ps561.OverlayValues[54] = d54
		ps561.OverlayValues[55] = d55
		ps561.OverlayValues[56] = d56
		ps561.OverlayValues[59] = d59
		ps561.OverlayValues[60] = d60
		ps561.OverlayValues[61] = d61
		ps561.OverlayValues[119] = d119
		ps561.OverlayValues[120] = d120
		ps561.OverlayValues[121] = d121
		ps561.OverlayValues[122] = d122
		ps561.OverlayValues[123] = d123
		ps561.OverlayValues[124] = d124
		ps561.OverlayValues[125] = d125
		ps561.OverlayValues[126] = d126
		ps561.OverlayValues[127] = d127
		ps561.OverlayValues[128] = d128
		ps561.OverlayValues[129] = d129
		ps561.OverlayValues[130] = d130
		ps561.OverlayValues[131] = d131
		ps561.OverlayValues[132] = d132
		ps561.OverlayValues[133] = d133
		ps561.OverlayValues[134] = d134
		ps561.OverlayValues[135] = d135
		ps561.OverlayValues[136] = d136
		ps561.OverlayValues[137] = d137
		ps561.OverlayValues[138] = d138
		ps561.OverlayValues[139] = d139
		ps561.OverlayValues[140] = d140
		ps561.OverlayValues[141] = d141
		ps561.OverlayValues[142] = d142
		ps561.OverlayValues[143] = d143
		ps561.OverlayValues[144] = d144
		ps561.OverlayValues[145] = d145
		ps561.OverlayValues[146] = d146
		ps561.OverlayValues[147] = d147
		ps561.OverlayValues[150] = d150
		ps561.OverlayValues[238] = d238
		ps561.OverlayValues[239] = d239
		ps561.OverlayValues[240] = d240
		ps561.OverlayValues[241] = d241
		ps561.OverlayValues[243] = d243
		ps561.OverlayValues[244] = d244
		ps561.OverlayValues[245] = d245
		ps561.OverlayValues[246] = d246
		ps561.OverlayValues[247] = d247
		ps561.OverlayValues[248] = d248
		ps561.OverlayValues[249] = d249
		ps561.OverlayValues[250] = d250
		ps561.OverlayValues[252] = d252
		ps561.OverlayValues[254] = d254
		ps561.OverlayValues[255] = d255
		ps561.OverlayValues[256] = d256
		ps561.OverlayValues[257] = d257
		ps561.OverlayValues[258] = d258
		ps561.OverlayValues[261] = d261
		ps561.OverlayValues[366] = d366
		ps561.OverlayValues[367] = d367
		ps561.OverlayValues[368] = d368
		ps561.OverlayValues[369] = d369
		ps561.OverlayValues[370] = d370
		ps561.OverlayValues[372] = d372
		ps561.OverlayValues[373] = d373
		ps561.OverlayValues[374] = d374
		ps561.OverlayValues[375] = d375
		ps561.OverlayValues[376] = d376
		ps561.OverlayValues[377] = d377
		ps561.OverlayValues[378] = d378
		ps561.OverlayValues[379] = d379
		ps561.OverlayValues[380] = d380
		ps561.OverlayValues[381] = d381
		ps561.OverlayValues[382] = d382
		ps561.OverlayValues[383] = d383
		ps561.OverlayValues[384] = d384
		ps561.OverlayValues[385] = d385
		ps561.OverlayValues[386] = d386
		ps561.OverlayValues[387] = d387
		ps561.OverlayValues[388] = d388
		ps561.OverlayValues[389] = d389
		ps561.OverlayValues[390] = d390
		ps561.OverlayValues[391] = d391
		ps561.OverlayValues[392] = d392
		ps561.OverlayValues[393] = d393
		ps561.OverlayValues[394] = d394
		ps561.OverlayValues[395] = d395
		ps561.OverlayValues[396] = d396
		ps561.OverlayValues[397] = d397
		ps561.OverlayValues[398] = d398
		ps561.OverlayValues[399] = d399
		ps561.OverlayValues[400] = d400
		ps561.OverlayValues[543] = d543
		ps561.OverlayValues[544] = d544
		ps561.OverlayValues[545] = d545
		ps561.OverlayValues[547] = d547
		ps561.OverlayValues[548] = d548
		ps561.OverlayValues[549] = d549
		ps561.OverlayValues[550] = d550
		ps561.OverlayValues[551] = d551
		ps561.OverlayValues[552] = d552
		ps561.OverlayValues[553] = d553
		ps561.OverlayValues[555] = d555
		ps561.OverlayValues[557] = d557
		ps561.OverlayValues[558] = d558
		ps561.OverlayValues[559] = d559
		ps561.OverlayValues[560] = d560
		ps561.PhiValues = make([]scm.JITValueDesc, 1)
		d563 = d12
		ps561.PhiValues[0] = d563
		ps562 := scm.PhiState{General: true}
		ps562.OverlayValues = make([]scm.JITValueDesc, 564)
		ps562.OverlayValues[5] = d5
		ps562.OverlayValues[6] = d6
		ps562.OverlayValues[7] = d7
		ps562.OverlayValues[8] = d8
		ps562.OverlayValues[9] = d9
		ps562.OverlayValues[10] = d10
		ps562.OverlayValues[11] = d11
		ps562.OverlayValues[12] = d12
		ps562.OverlayValues[13] = d13
		ps562.OverlayValues[14] = d14
		ps562.OverlayValues[15] = d15
		ps562.OverlayValues[16] = d16
		ps562.OverlayValues[17] = d17
		ps562.OverlayValues[18] = d18
		ps562.OverlayValues[19] = d19
		ps562.OverlayValues[20] = d20
		ps562.OverlayValues[21] = d21
		ps562.OverlayValues[23] = d23
		ps562.OverlayValues[24] = d24
		ps562.OverlayValues[25] = d25
		ps562.OverlayValues[26] = d26
		ps562.OverlayValues[27] = d27
		ps562.OverlayValues[28] = d28
		ps562.OverlayValues[29] = d29
		ps562.OverlayValues[30] = d30
		ps562.OverlayValues[31] = d31
		ps562.OverlayValues[32] = d32
		ps562.OverlayValues[33] = d33
		ps562.OverlayValues[34] = d34
		ps562.OverlayValues[35] = d35
		ps562.OverlayValues[36] = d36
		ps562.OverlayValues[37] = d37
		ps562.OverlayValues[38] = d38
		ps562.OverlayValues[39] = d39
		ps562.OverlayValues[40] = d40
		ps562.OverlayValues[41] = d41
		ps562.OverlayValues[42] = d42
		ps562.OverlayValues[43] = d43
		ps562.OverlayValues[44] = d44
		ps562.OverlayValues[45] = d45
		ps562.OverlayValues[46] = d46
		ps562.OverlayValues[47] = d47
		ps562.OverlayValues[48] = d48
		ps562.OverlayValues[49] = d49
		ps562.OverlayValues[50] = d50
		ps562.OverlayValues[51] = d51
		ps562.OverlayValues[52] = d52
		ps562.OverlayValues[53] = d53
		ps562.OverlayValues[54] = d54
		ps562.OverlayValues[55] = d55
		ps562.OverlayValues[56] = d56
		ps562.OverlayValues[59] = d59
		ps562.OverlayValues[60] = d60
		ps562.OverlayValues[61] = d61
		ps562.OverlayValues[119] = d119
		ps562.OverlayValues[120] = d120
		ps562.OverlayValues[121] = d121
		ps562.OverlayValues[122] = d122
		ps562.OverlayValues[123] = d123
		ps562.OverlayValues[124] = d124
		ps562.OverlayValues[125] = d125
		ps562.OverlayValues[126] = d126
		ps562.OverlayValues[127] = d127
		ps562.OverlayValues[128] = d128
		ps562.OverlayValues[129] = d129
		ps562.OverlayValues[130] = d130
		ps562.OverlayValues[131] = d131
		ps562.OverlayValues[132] = d132
		ps562.OverlayValues[133] = d133
		ps562.OverlayValues[134] = d134
		ps562.OverlayValues[135] = d135
		ps562.OverlayValues[136] = d136
		ps562.OverlayValues[137] = d137
		ps562.OverlayValues[138] = d138
		ps562.OverlayValues[139] = d139
		ps562.OverlayValues[140] = d140
		ps562.OverlayValues[141] = d141
		ps562.OverlayValues[142] = d142
		ps562.OverlayValues[143] = d143
		ps562.OverlayValues[144] = d144
		ps562.OverlayValues[145] = d145
		ps562.OverlayValues[146] = d146
		ps562.OverlayValues[147] = d147
		ps562.OverlayValues[150] = d150
		ps562.OverlayValues[238] = d238
		ps562.OverlayValues[239] = d239
		ps562.OverlayValues[240] = d240
		ps562.OverlayValues[241] = d241
		ps562.OverlayValues[243] = d243
		ps562.OverlayValues[244] = d244
		ps562.OverlayValues[245] = d245
		ps562.OverlayValues[246] = d246
		ps562.OverlayValues[247] = d247
		ps562.OverlayValues[248] = d248
		ps562.OverlayValues[249] = d249
		ps562.OverlayValues[250] = d250
		ps562.OverlayValues[252] = d252
		ps562.OverlayValues[254] = d254
		ps562.OverlayValues[255] = d255
		ps562.OverlayValues[256] = d256
		ps562.OverlayValues[257] = d257
		ps562.OverlayValues[258] = d258
		ps562.OverlayValues[261] = d261
		ps562.OverlayValues[366] = d366
		ps562.OverlayValues[367] = d367
		ps562.OverlayValues[368] = d368
		ps562.OverlayValues[369] = d369
		ps562.OverlayValues[370] = d370
		ps562.OverlayValues[372] = d372
		ps562.OverlayValues[373] = d373
		ps562.OverlayValues[374] = d374
		ps562.OverlayValues[375] = d375
		ps562.OverlayValues[376] = d376
		ps562.OverlayValues[377] = d377
		ps562.OverlayValues[378] = d378
		ps562.OverlayValues[379] = d379
		ps562.OverlayValues[380] = d380
		ps562.OverlayValues[381] = d381
		ps562.OverlayValues[382] = d382
		ps562.OverlayValues[383] = d383
		ps562.OverlayValues[384] = d384
		ps562.OverlayValues[385] = d385
		ps562.OverlayValues[386] = d386
		ps562.OverlayValues[387] = d387
		ps562.OverlayValues[388] = d388
		ps562.OverlayValues[389] = d389
		ps562.OverlayValues[390] = d390
		ps562.OverlayValues[391] = d391
		ps562.OverlayValues[392] = d392
		ps562.OverlayValues[393] = d393
		ps562.OverlayValues[394] = d394
		ps562.OverlayValues[395] = d395
		ps562.OverlayValues[396] = d396
		ps562.OverlayValues[397] = d397
		ps562.OverlayValues[398] = d398
		ps562.OverlayValues[399] = d399
		ps562.OverlayValues[400] = d400
		ps562.OverlayValues[543] = d543
		ps562.OverlayValues[544] = d544
		ps562.OverlayValues[545] = d545
		ps562.OverlayValues[547] = d547
		ps562.OverlayValues[548] = d548
		ps562.OverlayValues[549] = d549
		ps562.OverlayValues[550] = d550
		ps562.OverlayValues[551] = d551
		ps562.OverlayValues[552] = d552
		ps562.OverlayValues[553] = d553
		ps562.OverlayValues[555] = d555
		ps562.OverlayValues[557] = d557
		ps562.OverlayValues[558] = d558
		ps562.OverlayValues[559] = d559
		ps562.OverlayValues[560] = d560
		ps562.OverlayValues[563] = d563
		snap564 := d5
		snap565 := d6
		snap566 := d7
		snap567 := d8
		snap568 := d9
		snap569 := d10
		snap570 := d11
		snap571 := d12
		snap572 := d13
		snap573 := d14
		snap574 := d15
		snap575 := d16
		snap576 := d17
		snap577 := d18
		snap578 := d19
		snap579 := d20
		snap580 := d21
		snap581 := d23
		snap582 := d24
		snap583 := d25
		snap584 := d26
		snap585 := d27
		snap586 := d28
		snap587 := d29
		snap588 := d30
		snap589 := d31
		snap590 := d32
		snap591 := d33
		snap592 := d34
		snap593 := d35
		snap594 := d36
		snap595 := d37
		snap596 := d38
		snap597 := d39
		snap598 := d40
		snap599 := d41
		snap600 := d42
		snap601 := d43
		snap602 := d44
		snap603 := d45
		snap604 := d46
		snap605 := d47
		snap606 := d48
		snap607 := d49
		snap608 := d50
		snap609 := d51
		snap610 := d52
		snap611 := d53
		snap612 := d54
		snap613 := d55
		snap614 := d56
		snap615 := d59
		snap616 := d60
		snap617 := d61
		snap618 := d119
		snap619 := d120
		snap620 := d121
		snap621 := d122
		snap622 := d123
		snap623 := d124
		snap624 := d125
		snap625 := d126
		snap626 := d127
		snap627 := d128
		snap628 := d129
		snap629 := d130
		snap630 := d131
		snap631 := d132
		snap632 := d133
		snap633 := d134
		snap634 := d135
		snap635 := d136
		snap636 := d137
		snap637 := d138
		snap638 := d139
		snap639 := d140
		snap640 := d141
		snap641 := d142
		snap642 := d143
		snap643 := d144
		snap644 := d145
		snap645 := d146
		snap646 := d147
		snap647 := d150
		snap648 := d238
		snap649 := d239
		snap650 := d240
		snap651 := d241
		snap652 := d243
		snap653 := d244
		snap654 := d245
		snap655 := d246
		snap656 := d247
		snap657 := d248
		snap658 := d249
		snap659 := d250
		snap660 := d252
		snap661 := d254
		snap662 := d255
		snap663 := d256
		snap664 := d257
		snap665 := d258
		snap666 := d261
		snap667 := d366
		snap668 := d367
		snap669 := d368
		snap670 := d369
		snap671 := d370
		snap672 := d372
		snap673 := d373
		snap674 := d374
		snap675 := d375
		snap676 := d376
		snap677 := d377
		snap678 := d378
		snap679 := d379
		snap680 := d380
		snap681 := d381
		snap682 := d382
		snap683 := d383
		snap684 := d384
		snap685 := d385
		snap686 := d386
		snap687 := d387
		snap688 := d388
		snap689 := d389
		snap690 := d390
		snap691 := d391
		snap692 := d392
		snap693 := d393
		snap694 := d394
		snap695 := d395
		snap696 := d396
		snap697 := d397
		snap698 := d398
		snap699 := d399
		snap700 := d400
		snap701 := d543
		snap702 := d544
		snap703 := d545
		snap704 := d547
		snap705 := d548
		snap706 := d549
		snap707 := d550
		snap708 := d551
		snap709 := d552
		snap710 := d553
		snap711 := d555
		snap712 := d557
		snap713 := d558
		snap714 := d559
		snap715 := d560
		snap716 := d563
		alloc717 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps561)
		}
		ctx.RestoreAllocState(alloc717)
		d5 = snap564
		d6 = snap565
		d7 = snap566
		d8 = snap567
		d9 = snap568
		d10 = snap569
		d11 = snap570
		d12 = snap571
		d13 = snap572
		d14 = snap573
		d15 = snap574
		d16 = snap575
		d17 = snap576
		d18 = snap577
		d19 = snap578
		d20 = snap579
		d21 = snap580
		d23 = snap581
		d24 = snap582
		d25 = snap583
		d26 = snap584
		d27 = snap585
		d28 = snap586
		d29 = snap587
		d30 = snap588
		d31 = snap589
		d32 = snap590
		d33 = snap591
		d34 = snap592
		d35 = snap593
		d36 = snap594
		d37 = snap595
		d38 = snap596
		d39 = snap597
		d40 = snap598
		d41 = snap599
		d42 = snap600
		d43 = snap601
		d44 = snap602
		d45 = snap603
		d46 = snap604
		d47 = snap605
		d48 = snap606
		d49 = snap607
		d50 = snap608
		d51 = snap609
		d52 = snap610
		d53 = snap611
		d54 = snap612
		d55 = snap613
		d56 = snap614
		d59 = snap615
		d60 = snap616
		d61 = snap617
		d119 = snap618
		d120 = snap619
		d121 = snap620
		d122 = snap621
		d123 = snap622
		d124 = snap623
		d125 = snap624
		d126 = snap625
		d127 = snap626
		d128 = snap627
		d129 = snap628
		d130 = snap629
		d131 = snap630
		d132 = snap631
		d133 = snap632
		d134 = snap633
		d135 = snap634
		d136 = snap635
		d137 = snap636
		d138 = snap637
		d139 = snap638
		d140 = snap639
		d141 = snap640
		d142 = snap641
		d143 = snap642
		d144 = snap643
		d145 = snap644
		d146 = snap645
		d147 = snap646
		d150 = snap647
		d238 = snap648
		d239 = snap649
		d240 = snap650
		d241 = snap651
		d243 = snap652
		d244 = snap653
		d245 = snap654
		d246 = snap655
		d247 = snap656
		d248 = snap657
		d249 = snap658
		d250 = snap659
		d252 = snap660
		d254 = snap661
		d255 = snap662
		d256 = snap663
		d257 = snap664
		d258 = snap665
		d261 = snap666
		d366 = snap667
		d367 = snap668
		d368 = snap669
		d369 = snap670
		d370 = snap671
		d372 = snap672
		d373 = snap673
		d374 = snap674
		d375 = snap675
		d376 = snap676
		d377 = snap677
		d378 = snap678
		d379 = snap679
		d380 = snap680
		d381 = snap681
		d382 = snap682
		d383 = snap683
		d384 = snap684
		d385 = snap685
		d386 = snap686
		d387 = snap687
		d388 = snap688
		d389 = snap689
		d390 = snap690
		d391 = snap691
		d392 = snap692
		d393 = snap693
		d394 = snap694
		d395 = snap695
		d396 = snap696
		d397 = snap697
		d398 = snap698
		d399 = snap699
		d400 = snap700
		d543 = snap701
		d544 = snap702
		d545 = snap703
		d547 = snap704
		d548 = snap705
		d549 = snap706
		d550 = snap707
		d551 = snap708
		d552 = snap709
		d553 = snap710
		d555 = snap711
		d557 = snap712
		d558 = snap713
		d559 = snap714
		d560 = snap715
		d563 = snap716
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps562)
		}
		return result
		ctx.FreeDesc(&d550)
		return result
	}
	bbs[9].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[9].VisitCount >= 0 {
				ps.General = true
				return bbs[9].RenderPS(ps)
			}
		}
		bbs[9].VisitCount++
		if ps.General {
			if bbs[9].Rendered {
				ctx.EmitJmp(lbl10)
				return result
			}
			bbs[9].Rendered = true
			bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_9 = bbs[9].Address
			ctx.MarkLabel(lbl10)
			ctx.ResolveFixups()
		}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != scm.LocNone {
			d374 = ps.OverlayValues[374]
		}
		if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != scm.LocNone {
			d375 = ps.OverlayValues[375]
		}
		if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != scm.LocNone {
			d376 = ps.OverlayValues[376]
		}
		if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != scm.LocNone {
			d377 = ps.OverlayValues[377]
		}
		if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != scm.LocNone {
			d378 = ps.OverlayValues[378]
		}
		if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != scm.LocNone {
			d379 = ps.OverlayValues[379]
		}
		if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
			d380 = ps.OverlayValues[380]
		}
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
		}
		if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
			d382 = ps.OverlayValues[382]
		}
		if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != scm.LocNone {
			d383 = ps.OverlayValues[383]
		}
		if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != scm.LocNone {
			d384 = ps.OverlayValues[384]
		}
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != scm.LocNone {
			d387 = ps.OverlayValues[387]
		}
		if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != scm.LocNone {
			d388 = ps.OverlayValues[388]
		}
		if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != scm.LocNone {
			d389 = ps.OverlayValues[389]
		}
		if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
			d390 = ps.OverlayValues[390]
		}
		if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != scm.LocNone {
			d391 = ps.OverlayValues[391]
		}
		if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != scm.LocNone {
			d392 = ps.OverlayValues[392]
		}
		if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != scm.LocNone {
			d393 = ps.OverlayValues[393]
		}
		if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != scm.LocNone {
			d394 = ps.OverlayValues[394]
		}
		if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != scm.LocNone {
			d395 = ps.OverlayValues[395]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 543 && ps.OverlayValues[543].Loc != scm.LocNone {
			d543 = ps.OverlayValues[543]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 545 && ps.OverlayValues[545].Loc != scm.LocNone {
			d545 = ps.OverlayValues[545]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
		}
		if len(ps.OverlayValues) > 551 && ps.OverlayValues[551].Loc != scm.LocNone {
			d551 = ps.OverlayValues[551]
		}
		if len(ps.OverlayValues) > 552 && ps.OverlayValues[552].Loc != scm.LocNone {
			d552 = ps.OverlayValues[552]
		}
		if len(ps.OverlayValues) > 553 && ps.OverlayValues[553].Loc != scm.LocNone {
			d553 = ps.OverlayValues[553]
		}
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 558 && ps.OverlayValues[558].Loc != scm.LocNone {
			d558 = ps.OverlayValues[558]
		}
		if len(ps.OverlayValues) > 559 && ps.OverlayValues[559].Loc != scm.LocNone {
			d559 = ps.OverlayValues[559]
		}
		if len(ps.OverlayValues) > 560 && ps.OverlayValues[560].Loc != scm.LocNone {
			d560 = ps.OverlayValues[560]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		ctx.ReclaimUntrackedRegs()
		if ps.General {
			ctx.SyncDesc(&d9)
			if d9.Loc == scm.LocReg {
				ctx.ProtectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.ProtectReg(d9.Reg)
				ctx.ProtectReg(d9.Reg2)
			}
			ctx.SyncDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d718 = d9
			if d718.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d718)
			d719 = d718
			if d719.Loc == scm.LocImm {
				d719 = scm.JITValueDesc{Loc: scm.LocImm, Type: d719.Type, Imm: scm.NewInt(int64(uint64(d719.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d719.Reg, 32)
				ctx.EmitShrRegImm8(d719.Reg, 32)
			}
			ctx.EmitStoreToStack(d719, int32(bbs[8].PhiBase)+int32(0))
			d720 = d11
			if d720.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d720)
			d721 = d720
			if d721.Loc == scm.LocImm {
				d721 = scm.JITValueDesc{Loc: scm.LocImm, Type: d721.Type, Imm: scm.NewInt(int64(uint64(d721.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d721.Reg, 32)
				ctx.EmitShrRegImm8(d721.Reg, 32)
			}
			ctx.EmitStoreToStack(d721, int32(bbs[8].PhiBase)+int32(16))
			if d9.Loc == scm.LocReg {
				ctx.UnprotectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d9.Reg)
				ctx.UnprotectReg(d9.Reg2)
			}
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
		}
		ps722 := scm.PhiState{General: ps.General}
		ps722.OverlayValues = make([]scm.JITValueDesc, 722)
		ps722.OverlayValues[5] = d5
		ps722.OverlayValues[6] = d6
		ps722.OverlayValues[7] = d7
		ps722.OverlayValues[8] = d8
		ps722.OverlayValues[9] = d9
		ps722.OverlayValues[10] = d10
		ps722.OverlayValues[11] = d11
		ps722.OverlayValues[12] = d12
		ps722.OverlayValues[13] = d13
		ps722.OverlayValues[14] = d14
		ps722.OverlayValues[15] = d15
		ps722.OverlayValues[16] = d16
		ps722.OverlayValues[17] = d17
		ps722.OverlayValues[18] = d18
		ps722.OverlayValues[19] = d19
		ps722.OverlayValues[20] = d20
		ps722.OverlayValues[21] = d21
		ps722.OverlayValues[23] = d23
		ps722.OverlayValues[24] = d24
		ps722.OverlayValues[25] = d25
		ps722.OverlayValues[26] = d26
		ps722.OverlayValues[27] = d27
		ps722.OverlayValues[28] = d28
		ps722.OverlayValues[29] = d29
		ps722.OverlayValues[30] = d30
		ps722.OverlayValues[31] = d31
		ps722.OverlayValues[32] = d32
		ps722.OverlayValues[33] = d33
		ps722.OverlayValues[34] = d34
		ps722.OverlayValues[35] = d35
		ps722.OverlayValues[36] = d36
		ps722.OverlayValues[37] = d37
		ps722.OverlayValues[38] = d38
		ps722.OverlayValues[39] = d39
		ps722.OverlayValues[40] = d40
		ps722.OverlayValues[41] = d41
		ps722.OverlayValues[42] = d42
		ps722.OverlayValues[43] = d43
		ps722.OverlayValues[44] = d44
		ps722.OverlayValues[45] = d45
		ps722.OverlayValues[46] = d46
		ps722.OverlayValues[47] = d47
		ps722.OverlayValues[48] = d48
		ps722.OverlayValues[49] = d49
		ps722.OverlayValues[50] = d50
		ps722.OverlayValues[51] = d51
		ps722.OverlayValues[52] = d52
		ps722.OverlayValues[53] = d53
		ps722.OverlayValues[54] = d54
		ps722.OverlayValues[55] = d55
		ps722.OverlayValues[56] = d56
		ps722.OverlayValues[59] = d59
		ps722.OverlayValues[60] = d60
		ps722.OverlayValues[61] = d61
		ps722.OverlayValues[119] = d119
		ps722.OverlayValues[120] = d120
		ps722.OverlayValues[121] = d121
		ps722.OverlayValues[122] = d122
		ps722.OverlayValues[123] = d123
		ps722.OverlayValues[124] = d124
		ps722.OverlayValues[125] = d125
		ps722.OverlayValues[126] = d126
		ps722.OverlayValues[127] = d127
		ps722.OverlayValues[128] = d128
		ps722.OverlayValues[129] = d129
		ps722.OverlayValues[130] = d130
		ps722.OverlayValues[131] = d131
		ps722.OverlayValues[132] = d132
		ps722.OverlayValues[133] = d133
		ps722.OverlayValues[134] = d134
		ps722.OverlayValues[135] = d135
		ps722.OverlayValues[136] = d136
		ps722.OverlayValues[137] = d137
		ps722.OverlayValues[138] = d138
		ps722.OverlayValues[139] = d139
		ps722.OverlayValues[140] = d140
		ps722.OverlayValues[141] = d141
		ps722.OverlayValues[142] = d142
		ps722.OverlayValues[143] = d143
		ps722.OverlayValues[144] = d144
		ps722.OverlayValues[145] = d145
		ps722.OverlayValues[146] = d146
		ps722.OverlayValues[147] = d147
		ps722.OverlayValues[150] = d150
		ps722.OverlayValues[238] = d238
		ps722.OverlayValues[239] = d239
		ps722.OverlayValues[240] = d240
		ps722.OverlayValues[241] = d241
		ps722.OverlayValues[243] = d243
		ps722.OverlayValues[244] = d244
		ps722.OverlayValues[245] = d245
		ps722.OverlayValues[246] = d246
		ps722.OverlayValues[247] = d247
		ps722.OverlayValues[248] = d248
		ps722.OverlayValues[249] = d249
		ps722.OverlayValues[250] = d250
		ps722.OverlayValues[252] = d252
		ps722.OverlayValues[254] = d254
		ps722.OverlayValues[255] = d255
		ps722.OverlayValues[256] = d256
		ps722.OverlayValues[257] = d257
		ps722.OverlayValues[258] = d258
		ps722.OverlayValues[261] = d261
		ps722.OverlayValues[366] = d366
		ps722.OverlayValues[367] = d367
		ps722.OverlayValues[368] = d368
		ps722.OverlayValues[369] = d369
		ps722.OverlayValues[370] = d370
		ps722.OverlayValues[372] = d372
		ps722.OverlayValues[373] = d373
		ps722.OverlayValues[374] = d374
		ps722.OverlayValues[375] = d375
		ps722.OverlayValues[376] = d376
		ps722.OverlayValues[377] = d377
		ps722.OverlayValues[378] = d378
		ps722.OverlayValues[379] = d379
		ps722.OverlayValues[380] = d380
		ps722.OverlayValues[381] = d381
		ps722.OverlayValues[382] = d382
		ps722.OverlayValues[383] = d383
		ps722.OverlayValues[384] = d384
		ps722.OverlayValues[385] = d385
		ps722.OverlayValues[386] = d386
		ps722.OverlayValues[387] = d387
		ps722.OverlayValues[388] = d388
		ps722.OverlayValues[389] = d389
		ps722.OverlayValues[390] = d390
		ps722.OverlayValues[391] = d391
		ps722.OverlayValues[392] = d392
		ps722.OverlayValues[393] = d393
		ps722.OverlayValues[394] = d394
		ps722.OverlayValues[395] = d395
		ps722.OverlayValues[396] = d396
		ps722.OverlayValues[397] = d397
		ps722.OverlayValues[398] = d398
		ps722.OverlayValues[399] = d399
		ps722.OverlayValues[400] = d400
		ps722.OverlayValues[543] = d543
		ps722.OverlayValues[544] = d544
		ps722.OverlayValues[545] = d545
		ps722.OverlayValues[547] = d547
		ps722.OverlayValues[548] = d548
		ps722.OverlayValues[549] = d549
		ps722.OverlayValues[550] = d550
		ps722.OverlayValues[551] = d551
		ps722.OverlayValues[552] = d552
		ps722.OverlayValues[553] = d553
		ps722.OverlayValues[555] = d555
		ps722.OverlayValues[557] = d557
		ps722.OverlayValues[558] = d558
		ps722.OverlayValues[559] = d559
		ps722.OverlayValues[560] = d560
		ps722.OverlayValues[563] = d563
		ps722.OverlayValues[718] = d718
		ps722.OverlayValues[719] = d719
		ps722.OverlayValues[720] = d720
		ps722.OverlayValues[721] = d721
		ps722.PhiValues = make([]scm.JITValueDesc, 2)
		d723 = d9
		ps722.PhiValues[0] = d723
		d724 = d11
		ps722.PhiValues[1] = d724
		if ps722.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps722)
		return result
	}
	bbs[10].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[10].VisitCount >= 0 {
				ps.General = true
				return bbs[10].RenderPS(ps)
			}
		}
		bbs[10].VisitCount++
		if ps.General {
			if bbs[10].Rendered {
				ctx.EmitJmp(lbl11)
				return result
			}
			bbs[10].Rendered = true
			bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_10 = bbs[10].Address
			ctx.MarkLabel(lbl11)
			ctx.ResolveFixups()
		}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != scm.LocNone {
			d374 = ps.OverlayValues[374]
		}
		if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != scm.LocNone {
			d375 = ps.OverlayValues[375]
		}
		if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != scm.LocNone {
			d376 = ps.OverlayValues[376]
		}
		if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != scm.LocNone {
			d377 = ps.OverlayValues[377]
		}
		if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != scm.LocNone {
			d378 = ps.OverlayValues[378]
		}
		if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != scm.LocNone {
			d379 = ps.OverlayValues[379]
		}
		if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
			d380 = ps.OverlayValues[380]
		}
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
		}
		if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
			d382 = ps.OverlayValues[382]
		}
		if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != scm.LocNone {
			d383 = ps.OverlayValues[383]
		}
		if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != scm.LocNone {
			d384 = ps.OverlayValues[384]
		}
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != scm.LocNone {
			d387 = ps.OverlayValues[387]
		}
		if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != scm.LocNone {
			d388 = ps.OverlayValues[388]
		}
		if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != scm.LocNone {
			d389 = ps.OverlayValues[389]
		}
		if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
			d390 = ps.OverlayValues[390]
		}
		if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != scm.LocNone {
			d391 = ps.OverlayValues[391]
		}
		if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != scm.LocNone {
			d392 = ps.OverlayValues[392]
		}
		if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != scm.LocNone {
			d393 = ps.OverlayValues[393]
		}
		if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != scm.LocNone {
			d394 = ps.OverlayValues[394]
		}
		if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != scm.LocNone {
			d395 = ps.OverlayValues[395]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 543 && ps.OverlayValues[543].Loc != scm.LocNone {
			d543 = ps.OverlayValues[543]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 545 && ps.OverlayValues[545].Loc != scm.LocNone {
			d545 = ps.OverlayValues[545]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
		}
		if len(ps.OverlayValues) > 551 && ps.OverlayValues[551].Loc != scm.LocNone {
			d551 = ps.OverlayValues[551]
		}
		if len(ps.OverlayValues) > 552 && ps.OverlayValues[552].Loc != scm.LocNone {
			d552 = ps.OverlayValues[552]
		}
		if len(ps.OverlayValues) > 553 && ps.OverlayValues[553].Loc != scm.LocNone {
			d553 = ps.OverlayValues[553]
		}
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 558 && ps.OverlayValues[558].Loc != scm.LocNone {
			d558 = ps.OverlayValues[558]
		}
		if len(ps.OverlayValues) > 559 && ps.OverlayValues[559].Loc != scm.LocNone {
			d559 = ps.OverlayValues[559]
		}
		if len(ps.OverlayValues) > 560 && ps.OverlayValues[560].Loc != scm.LocNone {
			d560 = ps.OverlayValues[560]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 718 && ps.OverlayValues[718].Loc != scm.LocNone {
			d718 = ps.OverlayValues[718]
		}
		if len(ps.OverlayValues) > 719 && ps.OverlayValues[719].Loc != scm.LocNone {
			d719 = ps.OverlayValues[719]
		}
		if len(ps.OverlayValues) > 720 && ps.OverlayValues[720].Loc != scm.LocNone {
			d720 = ps.OverlayValues[720]
		}
		if len(ps.OverlayValues) > 721 && ps.OverlayValues[721].Loc != scm.LocNone {
			d721 = ps.OverlayValues[721]
		}
		if len(ps.OverlayValues) > 723 && ps.OverlayValues[723].Loc != scm.LocNone {
			d723 = ps.OverlayValues[723]
		}
		if len(ps.OverlayValues) > 724 && ps.OverlayValues[724].Loc != scm.LocNone {
			d724 = ps.OverlayValues[724]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d12, &d13)
		var d725 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d725 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() + d13.Imm.Int())}
		} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
			r120 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r120, d12.Reg)
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d725)
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			ctx.BindReg(d13.Reg, &d725)
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitAddInt64(scratch, d13.Reg)
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d725)
		} else if d13.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d13.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d725)
		} else {
			r121 := ctx.AllocRegExcept(d12.Reg, d13.Reg)
			ctx.EmitMovRegReg(r121, d12.Reg)
			ctx.EmitAddInt64(r121, d13.Reg)
			d725 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
			ctx.BindReg(r121, &d725)
		}
		if d725.Loc == scm.LocImm {
			d725 = scm.JITValueDesc{Loc: scm.LocImm, Type: d725.Type, Imm: scm.NewInt(int64(uint64(d725.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d725.Reg, 32)
			ctx.EmitShrRegImm8(d725.Reg, 32)
		}
		if d725.Loc == scm.LocReg && d12.Loc == scm.LocReg && d725.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d725)
		var d726 scm.JITValueDesc
		if d725.Loc == scm.LocImm {
			d726 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d725.Imm.Int() / 2)}
		} else {
			r122 := ctx.AllocRegExcept(d725.Reg)
			ctx.EmitMovRegReg(r122, d725.Reg)
			ctx.EmitShrRegImm8(r122, 1)
			d726 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			ctx.BindReg(r122, &d726)
		}
		if d726.Loc == scm.LocImm {
			d726 = scm.JITValueDesc{Loc: scm.LocImm, Type: d726.Type, Imm: scm.NewInt(int64(uint64(d726.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d726.Reg, 32)
			ctx.EmitShrRegImm8(d726.Reg, 32)
		}
		if d726.Loc == scm.LocReg && d725.Loc == scm.LocReg && d726.Reg == d725.Reg {
			ctx.TransferReg(d725.Reg)
			d725.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d725)
		if ps.General {
			ctx.SyncDesc(&d12)
			if d12.Loc == scm.LocReg {
				ctx.ProtectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.ProtectReg(d12.Reg)
				ctx.ProtectReg(d12.Reg2)
			}
			ctx.SyncDesc(&d13)
			if d13.Loc == scm.LocReg {
				ctx.ProtectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.ProtectReg(d13.Reg)
				ctx.ProtectReg(d13.Reg2)
			}
			ctx.SyncDesc(&d726)
			if d726.Loc == scm.LocReg {
				ctx.ProtectReg(d726.Reg)
			} else if d726.Loc == scm.LocRegPair {
				ctx.ProtectReg(d726.Reg)
				ctx.ProtectReg(d726.Reg2)
			}
			d727 = d726
			if d727.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d727)
			d728 = d727
			if d728.Loc == scm.LocImm {
				d728 = scm.JITValueDesc{Loc: scm.LocImm, Type: d728.Type, Imm: scm.NewInt(int64(uint64(d728.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d728.Reg, 32)
				ctx.EmitShrRegImm8(d728.Reg, 32)
			}
			if phiHomeOK2 {
				ctx.EmitMovToReg(r0, d728)
			} else {
				ctx.EmitStoreToStack(d728, int32(bbs[1].PhiBase)+int32(0))
			}
			d729 = d12
			if d729.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d729)
			d730 = d729
			if d730.Loc == scm.LocImm {
				d730 = scm.JITValueDesc{Loc: scm.LocImm, Type: d730.Type, Imm: scm.NewInt(int64(uint64(d730.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d730.Reg, 32)
				ctx.EmitShrRegImm8(d730.Reg, 32)
			}
			if phiHomeOK3 {
				ctx.EmitMovToReg(r1, d730)
			} else {
				ctx.EmitStoreToStack(d730, int32(bbs[1].PhiBase)+int32(16))
			}
			d731 = d13
			if d731.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d731)
			d732 = d731
			if d732.Loc == scm.LocImm {
				d732 = scm.JITValueDesc{Loc: scm.LocImm, Type: d732.Type, Imm: scm.NewInt(int64(uint64(d732.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d732.Reg, 32)
				ctx.EmitShrRegImm8(d732.Reg, 32)
			}
			if phiHomeOK4 {
				ctx.EmitMovToReg(r2, d732)
			} else {
				ctx.EmitStoreToStack(d732, int32(bbs[1].PhiBase)+int32(32))
			}
			if d12.Loc == scm.LocReg {
				ctx.UnprotectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d12.Reg)
				ctx.UnprotectReg(d12.Reg2)
			}
			if d13.Loc == scm.LocReg {
				ctx.UnprotectReg(d13.Reg)
			} else if d13.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d13.Reg)
				ctx.UnprotectReg(d13.Reg2)
			}
			if d726.Loc == scm.LocReg {
				ctx.UnprotectReg(d726.Reg)
			} else if d726.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d726.Reg)
				ctx.UnprotectReg(d726.Reg2)
			}
		}
		ps733 := scm.PhiState{General: ps.General}
		ps733.OverlayValues = make([]scm.JITValueDesc, 733)
		ps733.OverlayValues[5] = d5
		ps733.OverlayValues[6] = d6
		ps733.OverlayValues[7] = d7
		ps733.OverlayValues[8] = d8
		ps733.OverlayValues[9] = d9
		ps733.OverlayValues[10] = d10
		ps733.OverlayValues[11] = d11
		ps733.OverlayValues[12] = d12
		ps733.OverlayValues[13] = d13
		ps733.OverlayValues[14] = d14
		ps733.OverlayValues[15] = d15
		ps733.OverlayValues[16] = d16
		ps733.OverlayValues[17] = d17
		ps733.OverlayValues[18] = d18
		ps733.OverlayValues[19] = d19
		ps733.OverlayValues[20] = d20
		ps733.OverlayValues[21] = d21
		ps733.OverlayValues[23] = d23
		ps733.OverlayValues[24] = d24
		ps733.OverlayValues[25] = d25
		ps733.OverlayValues[26] = d26
		ps733.OverlayValues[27] = d27
		ps733.OverlayValues[28] = d28
		ps733.OverlayValues[29] = d29
		ps733.OverlayValues[30] = d30
		ps733.OverlayValues[31] = d31
		ps733.OverlayValues[32] = d32
		ps733.OverlayValues[33] = d33
		ps733.OverlayValues[34] = d34
		ps733.OverlayValues[35] = d35
		ps733.OverlayValues[36] = d36
		ps733.OverlayValues[37] = d37
		ps733.OverlayValues[38] = d38
		ps733.OverlayValues[39] = d39
		ps733.OverlayValues[40] = d40
		ps733.OverlayValues[41] = d41
		ps733.OverlayValues[42] = d42
		ps733.OverlayValues[43] = d43
		ps733.OverlayValues[44] = d44
		ps733.OverlayValues[45] = d45
		ps733.OverlayValues[46] = d46
		ps733.OverlayValues[47] = d47
		ps733.OverlayValues[48] = d48
		ps733.OverlayValues[49] = d49
		ps733.OverlayValues[50] = d50
		ps733.OverlayValues[51] = d51
		ps733.OverlayValues[52] = d52
		ps733.OverlayValues[53] = d53
		ps733.OverlayValues[54] = d54
		ps733.OverlayValues[55] = d55
		ps733.OverlayValues[56] = d56
		ps733.OverlayValues[59] = d59
		ps733.OverlayValues[60] = d60
		ps733.OverlayValues[61] = d61
		ps733.OverlayValues[119] = d119
		ps733.OverlayValues[120] = d120
		ps733.OverlayValues[121] = d121
		ps733.OverlayValues[122] = d122
		ps733.OverlayValues[123] = d123
		ps733.OverlayValues[124] = d124
		ps733.OverlayValues[125] = d125
		ps733.OverlayValues[126] = d126
		ps733.OverlayValues[127] = d127
		ps733.OverlayValues[128] = d128
		ps733.OverlayValues[129] = d129
		ps733.OverlayValues[130] = d130
		ps733.OverlayValues[131] = d131
		ps733.OverlayValues[132] = d132
		ps733.OverlayValues[133] = d133
		ps733.OverlayValues[134] = d134
		ps733.OverlayValues[135] = d135
		ps733.OverlayValues[136] = d136
		ps733.OverlayValues[137] = d137
		ps733.OverlayValues[138] = d138
		ps733.OverlayValues[139] = d139
		ps733.OverlayValues[140] = d140
		ps733.OverlayValues[141] = d141
		ps733.OverlayValues[142] = d142
		ps733.OverlayValues[143] = d143
		ps733.OverlayValues[144] = d144
		ps733.OverlayValues[145] = d145
		ps733.OverlayValues[146] = d146
		ps733.OverlayValues[147] = d147
		ps733.OverlayValues[150] = d150
		ps733.OverlayValues[238] = d238
		ps733.OverlayValues[239] = d239
		ps733.OverlayValues[240] = d240
		ps733.OverlayValues[241] = d241
		ps733.OverlayValues[243] = d243
		ps733.OverlayValues[244] = d244
		ps733.OverlayValues[245] = d245
		ps733.OverlayValues[246] = d246
		ps733.OverlayValues[247] = d247
		ps733.OverlayValues[248] = d248
		ps733.OverlayValues[249] = d249
		ps733.OverlayValues[250] = d250
		ps733.OverlayValues[252] = d252
		ps733.OverlayValues[254] = d254
		ps733.OverlayValues[255] = d255
		ps733.OverlayValues[256] = d256
		ps733.OverlayValues[257] = d257
		ps733.OverlayValues[258] = d258
		ps733.OverlayValues[261] = d261
		ps733.OverlayValues[366] = d366
		ps733.OverlayValues[367] = d367
		ps733.OverlayValues[368] = d368
		ps733.OverlayValues[369] = d369
		ps733.OverlayValues[370] = d370
		ps733.OverlayValues[372] = d372
		ps733.OverlayValues[373] = d373
		ps733.OverlayValues[374] = d374
		ps733.OverlayValues[375] = d375
		ps733.OverlayValues[376] = d376
		ps733.OverlayValues[377] = d377
		ps733.OverlayValues[378] = d378
		ps733.OverlayValues[379] = d379
		ps733.OverlayValues[380] = d380
		ps733.OverlayValues[381] = d381
		ps733.OverlayValues[382] = d382
		ps733.OverlayValues[383] = d383
		ps733.OverlayValues[384] = d384
		ps733.OverlayValues[385] = d385
		ps733.OverlayValues[386] = d386
		ps733.OverlayValues[387] = d387
		ps733.OverlayValues[388] = d388
		ps733.OverlayValues[389] = d389
		ps733.OverlayValues[390] = d390
		ps733.OverlayValues[391] = d391
		ps733.OverlayValues[392] = d392
		ps733.OverlayValues[393] = d393
		ps733.OverlayValues[394] = d394
		ps733.OverlayValues[395] = d395
		ps733.OverlayValues[396] = d396
		ps733.OverlayValues[397] = d397
		ps733.OverlayValues[398] = d398
		ps733.OverlayValues[399] = d399
		ps733.OverlayValues[400] = d400
		ps733.OverlayValues[543] = d543
		ps733.OverlayValues[544] = d544
		ps733.OverlayValues[545] = d545
		ps733.OverlayValues[547] = d547
		ps733.OverlayValues[548] = d548
		ps733.OverlayValues[549] = d549
		ps733.OverlayValues[550] = d550
		ps733.OverlayValues[551] = d551
		ps733.OverlayValues[552] = d552
		ps733.OverlayValues[553] = d553
		ps733.OverlayValues[555] = d555
		ps733.OverlayValues[557] = d557
		ps733.OverlayValues[558] = d558
		ps733.OverlayValues[559] = d559
		ps733.OverlayValues[560] = d560
		ps733.OverlayValues[563] = d563
		ps733.OverlayValues[718] = d718
		ps733.OverlayValues[719] = d719
		ps733.OverlayValues[720] = d720
		ps733.OverlayValues[721] = d721
		ps733.OverlayValues[723] = d723
		ps733.OverlayValues[724] = d724
		ps733.OverlayValues[725] = d725
		ps733.OverlayValues[726] = d726
		ps733.OverlayValues[727] = d727
		ps733.OverlayValues[728] = d728
		ps733.OverlayValues[729] = d729
		ps733.OverlayValues[730] = d730
		ps733.OverlayValues[731] = d731
		ps733.OverlayValues[732] = d732
		ps733.PhiValues = make([]scm.JITValueDesc, 3)
		d734 = d726
		ps733.PhiValues[0] = d734
		d735 = d12
		ps733.PhiValues[1] = d735
		d736 = d13
		ps733.PhiValues[2] = d736
		if ps733.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps733)
		return result
	}
	bbs[11].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[11].VisitCount >= 0 {
				ps.General = true
				return bbs[11].RenderPS(ps)
			}
		}
		bbs[11].VisitCount++
		if ps.General {
			if bbs[11].Rendered {
				ctx.EmitJmp(lbl12)
				return result
			}
			bbs[11].Rendered = true
			bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_11 = bbs[11].Address
			ctx.MarkLabel(lbl12)
			ctx.ResolveFixups()
		}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != scm.LocNone {
			d374 = ps.OverlayValues[374]
		}
		if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != scm.LocNone {
			d375 = ps.OverlayValues[375]
		}
		if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != scm.LocNone {
			d376 = ps.OverlayValues[376]
		}
		if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != scm.LocNone {
			d377 = ps.OverlayValues[377]
		}
		if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != scm.LocNone {
			d378 = ps.OverlayValues[378]
		}
		if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != scm.LocNone {
			d379 = ps.OverlayValues[379]
		}
		if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
			d380 = ps.OverlayValues[380]
		}
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
		}
		if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
			d382 = ps.OverlayValues[382]
		}
		if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != scm.LocNone {
			d383 = ps.OverlayValues[383]
		}
		if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != scm.LocNone {
			d384 = ps.OverlayValues[384]
		}
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != scm.LocNone {
			d387 = ps.OverlayValues[387]
		}
		if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != scm.LocNone {
			d388 = ps.OverlayValues[388]
		}
		if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != scm.LocNone {
			d389 = ps.OverlayValues[389]
		}
		if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
			d390 = ps.OverlayValues[390]
		}
		if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != scm.LocNone {
			d391 = ps.OverlayValues[391]
		}
		if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != scm.LocNone {
			d392 = ps.OverlayValues[392]
		}
		if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != scm.LocNone {
			d393 = ps.OverlayValues[393]
		}
		if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != scm.LocNone {
			d394 = ps.OverlayValues[394]
		}
		if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != scm.LocNone {
			d395 = ps.OverlayValues[395]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 543 && ps.OverlayValues[543].Loc != scm.LocNone {
			d543 = ps.OverlayValues[543]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 545 && ps.OverlayValues[545].Loc != scm.LocNone {
			d545 = ps.OverlayValues[545]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
		}
		if len(ps.OverlayValues) > 551 && ps.OverlayValues[551].Loc != scm.LocNone {
			d551 = ps.OverlayValues[551]
		}
		if len(ps.OverlayValues) > 552 && ps.OverlayValues[552].Loc != scm.LocNone {
			d552 = ps.OverlayValues[552]
		}
		if len(ps.OverlayValues) > 553 && ps.OverlayValues[553].Loc != scm.LocNone {
			d553 = ps.OverlayValues[553]
		}
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 558 && ps.OverlayValues[558].Loc != scm.LocNone {
			d558 = ps.OverlayValues[558]
		}
		if len(ps.OverlayValues) > 559 && ps.OverlayValues[559].Loc != scm.LocNone {
			d559 = ps.OverlayValues[559]
		}
		if len(ps.OverlayValues) > 560 && ps.OverlayValues[560].Loc != scm.LocNone {
			d560 = ps.OverlayValues[560]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 718 && ps.OverlayValues[718].Loc != scm.LocNone {
			d718 = ps.OverlayValues[718]
		}
		if len(ps.OverlayValues) > 719 && ps.OverlayValues[719].Loc != scm.LocNone {
			d719 = ps.OverlayValues[719]
		}
		if len(ps.OverlayValues) > 720 && ps.OverlayValues[720].Loc != scm.LocNone {
			d720 = ps.OverlayValues[720]
		}
		if len(ps.OverlayValues) > 721 && ps.OverlayValues[721].Loc != scm.LocNone {
			d721 = ps.OverlayValues[721]
		}
		if len(ps.OverlayValues) > 723 && ps.OverlayValues[723].Loc != scm.LocNone {
			d723 = ps.OverlayValues[723]
		}
		if len(ps.OverlayValues) > 724 && ps.OverlayValues[724].Loc != scm.LocNone {
			d724 = ps.OverlayValues[724]
		}
		if len(ps.OverlayValues) > 725 && ps.OverlayValues[725].Loc != scm.LocNone {
			d725 = ps.OverlayValues[725]
		}
		if len(ps.OverlayValues) > 726 && ps.OverlayValues[726].Loc != scm.LocNone {
			d726 = ps.OverlayValues[726]
		}
		if len(ps.OverlayValues) > 727 && ps.OverlayValues[727].Loc != scm.LocNone {
			d727 = ps.OverlayValues[727]
		}
		if len(ps.OverlayValues) > 728 && ps.OverlayValues[728].Loc != scm.LocNone {
			d728 = ps.OverlayValues[728]
		}
		if len(ps.OverlayValues) > 729 && ps.OverlayValues[729].Loc != scm.LocNone {
			d729 = ps.OverlayValues[729]
		}
		if len(ps.OverlayValues) > 730 && ps.OverlayValues[730].Loc != scm.LocNone {
			d730 = ps.OverlayValues[730]
		}
		if len(ps.OverlayValues) > 731 && ps.OverlayValues[731].Loc != scm.LocNone {
			d731 = ps.OverlayValues[731]
		}
		if len(ps.OverlayValues) > 732 && ps.OverlayValues[732].Loc != scm.LocNone {
			d732 = ps.OverlayValues[732]
		}
		if len(ps.OverlayValues) > 734 && ps.OverlayValues[734].Loc != scm.LocNone {
			d734 = ps.OverlayValues[734]
		}
		if len(ps.OverlayValues) > 735 && ps.OverlayValues[735].Loc != scm.LocNone {
			d735 = ps.OverlayValues[735]
		}
		if len(ps.OverlayValues) > 736 && ps.OverlayValues[736].Loc != scm.LocNone {
			d736 = ps.OverlayValues[736]
		}
		ctx.ReclaimUntrackedRegs()
		d737 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d738 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
		ctx.BindReg(r3, &d738)
		ctx.BindReg(r4, &d738)
		ctx.EnsureDesc(&d737)
		if d737.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d737, &d738)
		} else {
			switch d737.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d738, d737)
			case scm.TagInt:
				ctx.EmitMakeInt(d738, d737)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d738, d737)
			case scm.TagNil:
				ctx.EmitMakeNil(d738)
			default:
				ctx.EmitMovPairToResult(&d737, &d738)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[12].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[12].VisitCount >= 0 {
				ps.General = true
				return bbs[12].RenderPS(ps)
			}
		}
		bbs[12].VisitCount++
		if ps.General {
			if bbs[12].Rendered {
				ctx.EmitJmp(lbl13)
				return result
			}
			bbs[12].Rendered = true
			bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_12 = bbs[12].Address
			ctx.MarkLabel(lbl13)
			ctx.ResolveFixups()
		}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != scm.LocNone {
			d374 = ps.OverlayValues[374]
		}
		if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != scm.LocNone {
			d375 = ps.OverlayValues[375]
		}
		if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != scm.LocNone {
			d376 = ps.OverlayValues[376]
		}
		if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != scm.LocNone {
			d377 = ps.OverlayValues[377]
		}
		if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != scm.LocNone {
			d378 = ps.OverlayValues[378]
		}
		if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != scm.LocNone {
			d379 = ps.OverlayValues[379]
		}
		if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
			d380 = ps.OverlayValues[380]
		}
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
		}
		if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
			d382 = ps.OverlayValues[382]
		}
		if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != scm.LocNone {
			d383 = ps.OverlayValues[383]
		}
		if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != scm.LocNone {
			d384 = ps.OverlayValues[384]
		}
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != scm.LocNone {
			d387 = ps.OverlayValues[387]
		}
		if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != scm.LocNone {
			d388 = ps.OverlayValues[388]
		}
		if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != scm.LocNone {
			d389 = ps.OverlayValues[389]
		}
		if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
			d390 = ps.OverlayValues[390]
		}
		if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != scm.LocNone {
			d391 = ps.OverlayValues[391]
		}
		if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != scm.LocNone {
			d392 = ps.OverlayValues[392]
		}
		if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != scm.LocNone {
			d393 = ps.OverlayValues[393]
		}
		if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != scm.LocNone {
			d394 = ps.OverlayValues[394]
		}
		if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != scm.LocNone {
			d395 = ps.OverlayValues[395]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 543 && ps.OverlayValues[543].Loc != scm.LocNone {
			d543 = ps.OverlayValues[543]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 545 && ps.OverlayValues[545].Loc != scm.LocNone {
			d545 = ps.OverlayValues[545]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
		}
		if len(ps.OverlayValues) > 551 && ps.OverlayValues[551].Loc != scm.LocNone {
			d551 = ps.OverlayValues[551]
		}
		if len(ps.OverlayValues) > 552 && ps.OverlayValues[552].Loc != scm.LocNone {
			d552 = ps.OverlayValues[552]
		}
		if len(ps.OverlayValues) > 553 && ps.OverlayValues[553].Loc != scm.LocNone {
			d553 = ps.OverlayValues[553]
		}
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 558 && ps.OverlayValues[558].Loc != scm.LocNone {
			d558 = ps.OverlayValues[558]
		}
		if len(ps.OverlayValues) > 559 && ps.OverlayValues[559].Loc != scm.LocNone {
			d559 = ps.OverlayValues[559]
		}
		if len(ps.OverlayValues) > 560 && ps.OverlayValues[560].Loc != scm.LocNone {
			d560 = ps.OverlayValues[560]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 718 && ps.OverlayValues[718].Loc != scm.LocNone {
			d718 = ps.OverlayValues[718]
		}
		if len(ps.OverlayValues) > 719 && ps.OverlayValues[719].Loc != scm.LocNone {
			d719 = ps.OverlayValues[719]
		}
		if len(ps.OverlayValues) > 720 && ps.OverlayValues[720].Loc != scm.LocNone {
			d720 = ps.OverlayValues[720]
		}
		if len(ps.OverlayValues) > 721 && ps.OverlayValues[721].Loc != scm.LocNone {
			d721 = ps.OverlayValues[721]
		}
		if len(ps.OverlayValues) > 723 && ps.OverlayValues[723].Loc != scm.LocNone {
			d723 = ps.OverlayValues[723]
		}
		if len(ps.OverlayValues) > 724 && ps.OverlayValues[724].Loc != scm.LocNone {
			d724 = ps.OverlayValues[724]
		}
		if len(ps.OverlayValues) > 725 && ps.OverlayValues[725].Loc != scm.LocNone {
			d725 = ps.OverlayValues[725]
		}
		if len(ps.OverlayValues) > 726 && ps.OverlayValues[726].Loc != scm.LocNone {
			d726 = ps.OverlayValues[726]
		}
		if len(ps.OverlayValues) > 727 && ps.OverlayValues[727].Loc != scm.LocNone {
			d727 = ps.OverlayValues[727]
		}
		if len(ps.OverlayValues) > 728 && ps.OverlayValues[728].Loc != scm.LocNone {
			d728 = ps.OverlayValues[728]
		}
		if len(ps.OverlayValues) > 729 && ps.OverlayValues[729].Loc != scm.LocNone {
			d729 = ps.OverlayValues[729]
		}
		if len(ps.OverlayValues) > 730 && ps.OverlayValues[730].Loc != scm.LocNone {
			d730 = ps.OverlayValues[730]
		}
		if len(ps.OverlayValues) > 731 && ps.OverlayValues[731].Loc != scm.LocNone {
			d731 = ps.OverlayValues[731]
		}
		if len(ps.OverlayValues) > 732 && ps.OverlayValues[732].Loc != scm.LocNone {
			d732 = ps.OverlayValues[732]
		}
		if len(ps.OverlayValues) > 734 && ps.OverlayValues[734].Loc != scm.LocNone {
			d734 = ps.OverlayValues[734]
		}
		if len(ps.OverlayValues) > 735 && ps.OverlayValues[735].Loc != scm.LocNone {
			d735 = ps.OverlayValues[735]
		}
		if len(ps.OverlayValues) > 736 && ps.OverlayValues[736].Loc != scm.LocNone {
			d736 = ps.OverlayValues[736]
		}
		if len(ps.OverlayValues) > 737 && ps.OverlayValues[737].Loc != scm.LocNone {
			d737 = ps.OverlayValues[737]
		}
		if len(ps.OverlayValues) > 738 && ps.OverlayValues[738].Loc != scm.LocNone {
			d738 = ps.OverlayValues[738]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		d739 = d8
		_ = d739
		ctx.StabilizeDescForControlFlow(&d739)
		ctx.StabilizeDescForControlFlow(&d8)
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl28 := ctx.ReserveLabel()
		_ = lbl28
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl28)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d739)
		ctx.EnsureDesc(&d739)
		var d740 scm.JITValueDesc
		if d739.Loc == scm.LocImm {
			d740 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d739.Imm.Int()))))}
		} else {
			r123 := ctx.AllocReg()
			ctx.EmitMovRegReg(r123, d739.Reg)
			ctx.EmitShlRegImm8(r123, 32)
			ctx.EmitShrRegImm8(r123, 32)
			d740 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d740)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d741 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			r124 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r124, fieldAddr)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r124}
			ctx.BindReg(r124, &d741)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			r125 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r125, thisptr.Reg, off)
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r125}
			ctx.BindReg(r125, &d741)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d741)
		ctx.EnsureDesc(&d741)
		var d742 scm.JITValueDesc
		if d741.Loc == scm.LocImm {
			d742 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d741.Imm.Int()))))}
		} else {
			r126 := ctx.AllocReg()
			ctx.EmitMovRegReg(r126, d741.Reg)
			ctx.EmitShlRegImm8(r126, 56)
			ctx.EmitShrRegImm8(r126, 56)
			d742 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
			ctx.BindReg(r126, &d742)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d740)
		ctx.EnsureDesc(&d742)
		ctx.EnsureDescsTogether(&d740, &d742)
		var d743 scm.JITValueDesc
		if d740.Loc == scm.LocImm && d742.Loc == scm.LocImm {
			d743 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d740.Imm.Int() * d742.Imm.Int())}
		} else if d740.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d742.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d740.Imm.Int()))
			ctx.EmitImulInt64(scratch, d742.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d743)
		} else if d742.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d740.Reg)
			ctx.EmitMovRegReg(scratch, d740.Reg)
			if d742.Imm.Int() >= -2147483648 && d742.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d742.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d742.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d743)
		} else {
			r127 := ctx.AllocRegExcept(d740.Reg, d742.Reg)
			ctx.EmitMovRegReg(r127, d740.Reg)
			ctx.EmitImulInt64(r127, d742.Reg)
			d743 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d743)
		}
		if d743.Loc == scm.LocReg && d740.Loc == scm.LocReg && d743.Reg == d740.Reg {
			ctx.TransferReg(d740.Reg)
			d740.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d740)
		ctx.FreeDesc(&d742)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d743)
		var d744 scm.JITValueDesc
		if d743.Loc == scm.LocImm {
			d744 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d743.Imm.Int() / 64)}
		} else {
			r128 := ctx.AllocRegExcept(d743.Reg)
			ctx.EmitMovRegReg(r128, d743.Reg)
			ctx.EmitShrRegImm8(r128, 6)
			d744 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d744)
		}
		if d744.Loc == scm.LocReg && d743.Loc == scm.LocReg && d744.Reg == d743.Reg {
			ctx.TransferReg(d743.Reg)
			d743.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d743)
		var d745 scm.JITValueDesc
		if d743.Loc == scm.LocImm {
			d745 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d743.Imm.Int() % 64)}
		} else {
			r129 := ctx.AllocRegExcept(d743.Reg)
			ctx.EmitMovRegReg(r129, d743.Reg)
			ctx.EmitAndRegImm32(r129, 63)
			d745 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			ctx.BindReg(r129, &d745)
		}
		if d745.Loc == scm.LocReg && d743.Loc == scm.LocReg && d745.Reg == d743.Reg {
			ctx.TransferReg(d743.Reg)
			d743.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d743)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d746 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
			r130 := ctx.AllocReg()
			r131 := ctx.AllocRegExcept(r130)
			r132 := ctx.AllocRegExcept(r130, r131)
			ctx.EmitMovRegMem64(r130, fieldAddr)
			ctx.EmitMovRegMem64(r131, fieldAddr+8)
			ctx.EmitMovRegMem64(r132, fieldAddr+16)
			d746 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r130, Reg2: r131, Reg3: r132}
			ctx.BindReg(r130, &d746)
			ctx.BindReg(r131, &d746)
			ctx.BindReg(r132, &d746)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
			r133 := ctx.AllocReg()
			r134 := ctx.AllocRegExcept(r133)
			r135 := ctx.AllocRegExcept(r133, r134)
			ctx.EmitMovRegMem(r133, thisptr.Reg, off)
			ctx.EmitMovRegMem(r134, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r135, thisptr.Reg, off+16)
			d746 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r133, Reg2: r134, Reg3: r135}
			ctx.BindReg(r133, &d746)
			ctx.BindReg(r134, &d746)
			ctx.BindReg(r135, &d746)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d744)
		ctx.ReclaimUntrackedRegs()
		d748 = ctx.EmitSliceElementAddress(&d746, &d744, 8)
		ctx.EnsureDesc(&d748)
		ctx.EmitMovRegMem(d748.Reg, d748.Reg, 0)
		d747 = d748
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d747)
		ctx.EnsureDesc(&d745)
		var d749 scm.JITValueDesc
		if d747.Loc == scm.LocImm && d745.Loc == scm.LocImm {
			d749 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d747.Imm.Int()) << uint64(d745.Imm.Int())))}
		} else if d745.Loc == scm.LocImm {
			r136 := ctx.AllocRegExcept(d747.Reg)
			ctx.EmitMovRegReg(r136, d747.Reg)
			ctx.EmitShlRegImm8(r136, uint8(d745.Imm.Int()))
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d749)
		} else {
			{
				shiftSrc := d747.Reg
				r137 := ctx.AllocRegExcept(d747.Reg)
				ctx.EmitMovRegReg(r137, d747.Reg)
				shiftSrc = r137
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d745.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d745.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d745.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d749 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d749)
			}
		}
		if d749.Loc == scm.LocReg && d747.Loc == scm.LocReg && d749.Reg == d747.Reg {
			ctx.TransferReg(d747.Reg)
			d747.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d747)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d744)
		ctx.EnsureDesc(&d744)
		var d750 scm.JITValueDesc
		if d744.Loc == scm.LocImm {
			d750 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d744.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d744.Reg)
			ctx.EmitMovRegReg(scratch, d744.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d750 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d750)
		}
		if d750.Loc == scm.LocReg && d744.Loc == scm.LocReg && d750.Reg == d744.Reg {
			ctx.TransferReg(d744.Reg)
			d744.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d744)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d750)
		ctx.ReclaimUntrackedRegs()
		d752 = ctx.EmitSliceElementAddress(&d746, &d750, 8)
		ctx.EnsureDesc(&d752)
		ctx.EmitMovRegMem(d752.Reg, d752.Reg, 0)
		d751 = d752
		ctx.FreeDesc(&d750)
		ctx.ReclaimUntrackedRegs()
		d753 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d745)
		ctx.EnsureDescsTogether(&d753, &d745)
		var d754 scm.JITValueDesc
		if d753.Loc == scm.LocImm && d745.Loc == scm.LocImm {
			d754 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d753.Imm.Int() - d745.Imm.Int())}
		} else if d745.Loc == scm.LocImm && d745.Imm.Int() == 0 {
			r138 := ctx.AllocRegExcept(d753.Reg)
			ctx.EmitMovRegReg(r138, d753.Reg)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d754)
		} else if d753.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d745.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d753.Imm.Int()))
			ctx.EmitSubInt64(scratch, d745.Reg)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d754)
		} else if d745.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d753.Reg)
			ctx.EmitMovRegReg(scratch, d753.Reg)
			if d745.Imm.Int() >= -2147483648 && d745.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d745.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d745.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d754)
		} else {
			r139 := ctx.AllocRegExcept(d753.Reg, d745.Reg)
			ctx.EmitMovRegReg(r139, d753.Reg)
			ctx.EmitSubInt64(r139, d745.Reg)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
			ctx.BindReg(r139, &d754)
		}
		if d754.Loc == scm.LocReg && d753.Loc == scm.LocReg && d754.Reg == d753.Reg {
			ctx.TransferReg(d753.Reg)
			d753.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d745)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d751)
		ctx.EnsureDesc(&d754)
		var d755 scm.JITValueDesc
		if d751.Loc == scm.LocImm && d754.Loc == scm.LocImm {
			d755 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d751.Imm.Int()) >> uint64(d754.Imm.Int())))}
		} else if d754.Loc == scm.LocImm {
			r140 := ctx.AllocRegExcept(d751.Reg)
			ctx.EmitMovRegReg(r140, d751.Reg)
			ctx.EmitShrRegImm8(r140, uint8(d754.Imm.Int()))
			d755 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d755)
		} else {
			{
				shiftSrc := d751.Reg
				r141 := ctx.AllocRegExcept(d751.Reg)
				ctx.EmitMovRegReg(r141, d751.Reg)
				shiftSrc = r141
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d754.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d754.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d754.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d755 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d755)
			}
		}
		if d755.Loc == scm.LocReg && d751.Loc == scm.LocReg && d755.Reg == d751.Reg {
			ctx.TransferReg(d751.Reg)
			d751.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d751)
		ctx.FreeDesc(&d754)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d749)
		ctx.EnsureDesc(&d755)
		var d756 scm.JITValueDesc
		if d749.Loc == scm.LocImm && d755.Loc == scm.LocImm {
			d756 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d749.Imm.Int() | d755.Imm.Int())}
		} else if d749.Loc == scm.LocImm && d749.Imm.Int() == 0 {
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d755.Reg}
			ctx.BindReg(d755.Reg, &d756)
		} else if d755.Loc == scm.LocImm && d755.Imm.Int() == 0 {
			r142 := ctx.AllocRegExcept(d749.Reg)
			ctx.EmitMovRegReg(r142, d749.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			ctx.BindReg(r142, &d756)
		} else if d749.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d755.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d749.Imm.Int()))
			ctx.EmitOrInt64(scratch, d755.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d756)
		} else if d755.Loc == scm.LocImm {
			r143 := ctx.AllocRegExcept(d749.Reg)
			ctx.EmitMovRegReg(r143, d749.Reg)
			if d755.Imm.Int() >= -2147483648 && d755.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r143, int32(d755.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d755.Imm.Int()))
				ctx.EmitOrInt64(r143, scm.RegR11)
			}
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			ctx.BindReg(r143, &d756)
		} else {
			r144 := ctx.AllocRegExcept(d749.Reg, d755.Reg)
			ctx.EmitMovRegReg(r144, d749.Reg)
			ctx.EmitOrInt64(r144, d755.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			ctx.BindReg(r144, &d756)
		}
		if d756.Loc == scm.LocReg && d749.Loc == scm.LocReg && d756.Reg == d749.Reg {
			ctx.TransferReg(d749.Reg)
			d749.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d749)
		ctx.FreeDesc(&d755)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d741)
		ctx.EnsureDesc(&d741)
		var d757 scm.JITValueDesc
		if d741.Loc == scm.LocImm {
			d757 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d741.Imm.Int()))))}
		} else {
			r145 := ctx.AllocReg()
			ctx.EmitMovRegReg(r145, d741.Reg)
			ctx.EmitShlRegImm8(r145, 56)
			ctx.EmitShrRegImm8(r145, 56)
			d757 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d757)
		}
		ctx.ReclaimUntrackedRegs()
		d758 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d757)
		ctx.EnsureDescsTogether(&d758, &d757)
		var d759 scm.JITValueDesc
		if d758.Loc == scm.LocImm && d757.Loc == scm.LocImm {
			d759 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d758.Imm.Int() - d757.Imm.Int())}
		} else if d757.Loc == scm.LocImm && d757.Imm.Int() == 0 {
			r146 := ctx.AllocRegExcept(d758.Reg)
			ctx.EmitMovRegReg(r146, d758.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			ctx.BindReg(r146, &d759)
		} else if d758.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d757.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d758.Imm.Int()))
			ctx.EmitSubInt64(scratch, d757.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d759)
		} else if d757.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d758.Reg)
			ctx.EmitMovRegReg(scratch, d758.Reg)
			if d757.Imm.Int() >= -2147483648 && d757.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d757.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d757.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d759)
		} else {
			r147 := ctx.AllocRegExcept(d758.Reg, d757.Reg)
			ctx.EmitMovRegReg(r147, d758.Reg)
			ctx.EmitSubInt64(r147, d757.Reg)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d759)
		}
		if d759.Loc == scm.LocReg && d758.Loc == scm.LocReg && d759.Reg == d758.Reg {
			ctx.TransferReg(d758.Reg)
			d758.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d757)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d756)
		ctx.EnsureDesc(&d759)
		var d760 scm.JITValueDesc
		if d756.Loc == scm.LocImm && d759.Loc == scm.LocImm {
			d760 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d756.Imm.Int()) >> uint64(d759.Imm.Int())))}
		} else if d759.Loc == scm.LocImm {
			r148 := ctx.AllocRegExcept(d756.Reg)
			ctx.EmitMovRegReg(r148, d756.Reg)
			ctx.EmitShrRegImm8(r148, uint8(d759.Imm.Int()))
			d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d760)
		} else {
			{
				shiftSrc := d756.Reg
				r149 := ctx.AllocRegExcept(d756.Reg)
				ctx.EmitMovRegReg(r149, d756.Reg)
				shiftSrc = r149
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d759.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d759.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d759.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d760)
			}
		}
		if d760.Loc == scm.LocReg && d756.Loc == scm.LocReg && d760.Reg == d756.Reg {
			ctx.TransferReg(d756.Reg)
			d756.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d756)
		ctx.FreeDesc(&d759)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d760)
		ctx.EnsureDesc(&d760)
		ctx.EnsureDesc(&d760)
		var d761 scm.JITValueDesc
		if d760.Loc == scm.LocImm {
			d761 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d760.Imm.Int()))))}
		} else {
			r150 := ctx.AllocReg()
			ctx.EmitMovRegReg(r150, d760.Reg)
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			ctx.BindReg(r150, &d761)
		}
		ctx.FreeDesc(&d760)
		var d762 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
			r151 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r151, fieldAddr)
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r151}
			ctx.BindReg(r151, &d762)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
			r152 := ctx.AllocReg()
			ctx.EmitMovRegMem(r152, thisptr.Reg, off)
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
			ctx.BindReg(r152, &d762)
		}
		ctx.EnsureDesc(&d761)
		ctx.EnsureDesc(&d762)
		ctx.EnsureDescsTogether(&d761, &d762)
		var d763 scm.JITValueDesc
		if d761.Loc == scm.LocImm && d762.Loc == scm.LocImm {
			d763 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d761.Imm.Int() + d762.Imm.Int())}
		} else if d762.Loc == scm.LocImm && d762.Imm.Int() == 0 {
			r153 := ctx.AllocRegExcept(d761.Reg)
			ctx.EmitMovRegReg(r153, d761.Reg)
			d763 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d763)
		} else if d761.Loc == scm.LocImm && d761.Imm.Int() == 0 {
			d763 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d762.Reg}
			ctx.BindReg(d762.Reg, &d763)
		} else if d761.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d762.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d761.Imm.Int()))
			ctx.EmitAddInt64(scratch, d762.Reg)
			d763 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d763)
		} else if d762.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d761.Reg)
			ctx.EmitMovRegReg(scratch, d761.Reg)
			if d762.Imm.Int() >= -2147483648 && d762.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d762.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d762.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d763 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d763)
		} else {
			r154 := ctx.AllocRegExcept(d761.Reg, d762.Reg)
			ctx.EmitMovRegReg(r154, d761.Reg)
			ctx.EmitAddInt64(r154, d762.Reg)
			d763 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			ctx.BindReg(r154, &d763)
		}
		if d763.Loc == scm.LocReg && d761.Loc == scm.LocReg && d763.Reg == d761.Reg {
			ctx.TransferReg(d761.Reg)
			d761.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d761)
		ctx.EnsureDesc(&d8)
		d764 = d8
		_ = d764
		ctx.StabilizeDescForControlFlow(&d764)
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl29 := ctx.ReserveLabel()
		_ = lbl29
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl29)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d764)
		ctx.EnsureDesc(&d764)
		var d765 scm.JITValueDesc
		if d764.Loc == scm.LocImm {
			d765 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d764.Imm.Int()))))}
		} else {
			r155 := ctx.AllocReg()
			ctx.EmitMovRegReg(r155, d764.Reg)
			ctx.EmitShlRegImm8(r155, 32)
			ctx.EmitShrRegImm8(r155, 32)
			d765 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d765)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d766 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			r156 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r156, fieldAddr)
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
			ctx.BindReg(r156, &d766)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			r157 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r157, thisptr.Reg, off)
			d766 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
			ctx.BindReg(r157, &d766)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d766)
		ctx.EnsureDesc(&d766)
		var d767 scm.JITValueDesc
		if d766.Loc == scm.LocImm {
			d767 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d766.Imm.Int()))))}
		} else {
			r158 := ctx.AllocReg()
			ctx.EmitMovRegReg(r158, d766.Reg)
			ctx.EmitShlRegImm8(r158, 56)
			ctx.EmitShrRegImm8(r158, 56)
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d767)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d765)
		ctx.EnsureDesc(&d767)
		ctx.EnsureDescsTogether(&d765, &d767)
		var d768 scm.JITValueDesc
		if d765.Loc == scm.LocImm && d767.Loc == scm.LocImm {
			d768 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d765.Imm.Int() * d767.Imm.Int())}
		} else if d765.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d767.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d765.Imm.Int()))
			ctx.EmitImulInt64(scratch, d767.Reg)
			d768 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d768)
		} else if d767.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d765.Reg)
			ctx.EmitMovRegReg(scratch, d765.Reg)
			if d767.Imm.Int() >= -2147483648 && d767.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d767.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d767.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d768 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d768)
		} else {
			r159 := ctx.AllocRegExcept(d765.Reg, d767.Reg)
			ctx.EmitMovRegReg(r159, d765.Reg)
			ctx.EmitImulInt64(r159, d767.Reg)
			d768 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
			ctx.BindReg(r159, &d768)
		}
		if d768.Loc == scm.LocReg && d765.Loc == scm.LocReg && d768.Reg == d765.Reg {
			ctx.TransferReg(d765.Reg)
			d765.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d765)
		ctx.FreeDesc(&d767)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d768)
		var d769 scm.JITValueDesc
		if d768.Loc == scm.LocImm {
			d769 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d768.Imm.Int() / 64)}
		} else {
			r160 := ctx.AllocRegExcept(d768.Reg)
			ctx.EmitMovRegReg(r160, d768.Reg)
			ctx.EmitShrRegImm8(r160, 6)
			d769 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			ctx.BindReg(r160, &d769)
		}
		if d769.Loc == scm.LocReg && d768.Loc == scm.LocReg && d769.Reg == d768.Reg {
			ctx.TransferReg(d768.Reg)
			d768.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d768)
		var d770 scm.JITValueDesc
		if d768.Loc == scm.LocImm {
			d770 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d768.Imm.Int() % 64)}
		} else {
			r161 := ctx.AllocRegExcept(d768.Reg)
			ctx.EmitMovRegReg(r161, d768.Reg)
			ctx.EmitAndRegImm32(r161, 63)
			d770 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
			ctx.BindReg(r161, &d770)
		}
		if d770.Loc == scm.LocReg && d768.Loc == scm.LocReg && d770.Reg == d768.Reg {
			ctx.TransferReg(d768.Reg)
			d768.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d768)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d771 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
			r162 := ctx.AllocReg()
			r163 := ctx.AllocRegExcept(r162)
			r164 := ctx.AllocRegExcept(r162, r163)
			ctx.EmitMovRegMem64(r162, fieldAddr)
			ctx.EmitMovRegMem64(r163, fieldAddr+8)
			ctx.EmitMovRegMem64(r164, fieldAddr+16)
			d771 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r162, Reg2: r163, Reg3: r164}
			ctx.BindReg(r162, &d771)
			ctx.BindReg(r163, &d771)
			ctx.BindReg(r164, &d771)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
			r165 := ctx.AllocReg()
			r166 := ctx.AllocRegExcept(r165)
			r167 := ctx.AllocRegExcept(r165, r166)
			ctx.EmitMovRegMem(r165, thisptr.Reg, off)
			ctx.EmitMovRegMem(r166, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r167, thisptr.Reg, off+16)
			d771 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r165, Reg2: r166, Reg3: r167}
			ctx.BindReg(r165, &d771)
			ctx.BindReg(r166, &d771)
			ctx.BindReg(r167, &d771)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d769)
		ctx.ReclaimUntrackedRegs()
		d773 = ctx.EmitSliceElementAddress(&d771, &d769, 8)
		ctx.EnsureDesc(&d773)
		ctx.EmitMovRegMem(d773.Reg, d773.Reg, 0)
		d772 = d773
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d772)
		ctx.EnsureDesc(&d770)
		var d774 scm.JITValueDesc
		if d772.Loc == scm.LocImm && d770.Loc == scm.LocImm {
			d774 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d772.Imm.Int()) << uint64(d770.Imm.Int())))}
		} else if d770.Loc == scm.LocImm {
			r168 := ctx.AllocRegExcept(d772.Reg)
			ctx.EmitMovRegReg(r168, d772.Reg)
			ctx.EmitShlRegImm8(r168, uint8(d770.Imm.Int()))
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d774)
		} else {
			{
				shiftSrc := d772.Reg
				r169 := ctx.AllocRegExcept(d772.Reg)
				ctx.EmitMovRegReg(r169, d772.Reg)
				shiftSrc = r169
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d770.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d770.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d770.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d774 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d774)
			}
		}
		if d774.Loc == scm.LocReg && d772.Loc == scm.LocReg && d774.Reg == d772.Reg {
			ctx.TransferReg(d772.Reg)
			d772.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d772)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d769)
		ctx.EnsureDesc(&d769)
		var d775 scm.JITValueDesc
		if d769.Loc == scm.LocImm {
			d775 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d769.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d769.Reg)
			ctx.EmitMovRegReg(scratch, d769.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d775 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d775)
		}
		if d775.Loc == scm.LocReg && d769.Loc == scm.LocReg && d775.Reg == d769.Reg {
			ctx.TransferReg(d769.Reg)
			d769.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d769)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d775)
		ctx.ReclaimUntrackedRegs()
		d777 = ctx.EmitSliceElementAddress(&d771, &d775, 8)
		ctx.EnsureDesc(&d777)
		ctx.EmitMovRegMem(d777.Reg, d777.Reg, 0)
		d776 = d777
		ctx.FreeDesc(&d775)
		ctx.ReclaimUntrackedRegs()
		d778 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d770)
		ctx.EnsureDescsTogether(&d778, &d770)
		var d779 scm.JITValueDesc
		if d778.Loc == scm.LocImm && d770.Loc == scm.LocImm {
			d779 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d778.Imm.Int() - d770.Imm.Int())}
		} else if d770.Loc == scm.LocImm && d770.Imm.Int() == 0 {
			r170 := ctx.AllocRegExcept(d778.Reg)
			ctx.EmitMovRegReg(r170, d778.Reg)
			d779 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
			ctx.BindReg(r170, &d779)
		} else if d778.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d770.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d778.Imm.Int()))
			ctx.EmitSubInt64(scratch, d770.Reg)
			d779 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d779)
		} else if d770.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d778.Reg)
			ctx.EmitMovRegReg(scratch, d778.Reg)
			if d770.Imm.Int() >= -2147483648 && d770.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d770.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d770.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d779 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d779)
		} else {
			r171 := ctx.AllocRegExcept(d778.Reg, d770.Reg)
			ctx.EmitMovRegReg(r171, d778.Reg)
			ctx.EmitSubInt64(r171, d770.Reg)
			d779 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
			ctx.BindReg(r171, &d779)
		}
		if d779.Loc == scm.LocReg && d778.Loc == scm.LocReg && d779.Reg == d778.Reg {
			ctx.TransferReg(d778.Reg)
			d778.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d770)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d776)
		ctx.EnsureDesc(&d779)
		var d780 scm.JITValueDesc
		if d776.Loc == scm.LocImm && d779.Loc == scm.LocImm {
			d780 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d776.Imm.Int()) >> uint64(d779.Imm.Int())))}
		} else if d779.Loc == scm.LocImm {
			r172 := ctx.AllocRegExcept(d776.Reg)
			ctx.EmitMovRegReg(r172, d776.Reg)
			ctx.EmitShrRegImm8(r172, uint8(d779.Imm.Int()))
			d780 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d780)
		} else {
			{
				shiftSrc := d776.Reg
				r173 := ctx.AllocRegExcept(d776.Reg)
				ctx.EmitMovRegReg(r173, d776.Reg)
				shiftSrc = r173
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d779.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d779.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d779.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d780 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d780)
			}
		}
		if d780.Loc == scm.LocReg && d776.Loc == scm.LocReg && d780.Reg == d776.Reg {
			ctx.TransferReg(d776.Reg)
			d776.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d776)
		ctx.FreeDesc(&d779)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d774)
		ctx.EnsureDesc(&d780)
		var d781 scm.JITValueDesc
		if d774.Loc == scm.LocImm && d780.Loc == scm.LocImm {
			d781 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d774.Imm.Int() | d780.Imm.Int())}
		} else if d774.Loc == scm.LocImm && d774.Imm.Int() == 0 {
			d781 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d780.Reg}
			ctx.BindReg(d780.Reg, &d781)
		} else if d780.Loc == scm.LocImm && d780.Imm.Int() == 0 {
			r174 := ctx.AllocRegExcept(d774.Reg)
			ctx.EmitMovRegReg(r174, d774.Reg)
			d781 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
			ctx.BindReg(r174, &d781)
		} else if d774.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d780.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d774.Imm.Int()))
			ctx.EmitOrInt64(scratch, d780.Reg)
			d781 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d781)
		} else if d780.Loc == scm.LocImm {
			r175 := ctx.AllocRegExcept(d774.Reg)
			ctx.EmitMovRegReg(r175, d774.Reg)
			if d780.Imm.Int() >= -2147483648 && d780.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r175, int32(d780.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d780.Imm.Int()))
				ctx.EmitOrInt64(r175, scm.RegR11)
			}
			d781 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
			ctx.BindReg(r175, &d781)
		} else {
			r176 := ctx.AllocRegExcept(d774.Reg, d780.Reg)
			ctx.EmitMovRegReg(r176, d774.Reg)
			ctx.EmitOrInt64(r176, d780.Reg)
			d781 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
			ctx.BindReg(r176, &d781)
		}
		if d781.Loc == scm.LocReg && d774.Loc == scm.LocReg && d781.Reg == d774.Reg {
			ctx.TransferReg(d774.Reg)
			d774.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d774)
		ctx.FreeDesc(&d780)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d766)
		ctx.EnsureDesc(&d766)
		var d782 scm.JITValueDesc
		if d766.Loc == scm.LocImm {
			d782 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d766.Imm.Int()))))}
		} else {
			r177 := ctx.AllocReg()
			ctx.EmitMovRegReg(r177, d766.Reg)
			ctx.EmitShlRegImm8(r177, 56)
			ctx.EmitShrRegImm8(r177, 56)
			d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
			ctx.BindReg(r177, &d782)
		}
		ctx.ReclaimUntrackedRegs()
		d783 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d782)
		ctx.EnsureDescsTogether(&d783, &d782)
		var d784 scm.JITValueDesc
		if d783.Loc == scm.LocImm && d782.Loc == scm.LocImm {
			d784 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d783.Imm.Int() - d782.Imm.Int())}
		} else if d782.Loc == scm.LocImm && d782.Imm.Int() == 0 {
			r178 := ctx.AllocRegExcept(d783.Reg)
			ctx.EmitMovRegReg(r178, d783.Reg)
			d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
			ctx.BindReg(r178, &d784)
		} else if d783.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d782.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d783.Imm.Int()))
			ctx.EmitSubInt64(scratch, d782.Reg)
			d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d784)
		} else if d782.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d783.Reg)
			ctx.EmitMovRegReg(scratch, d783.Reg)
			if d782.Imm.Int() >= -2147483648 && d782.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d782.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d782.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d784)
		} else {
			r179 := ctx.AllocRegExcept(d783.Reg, d782.Reg)
			ctx.EmitMovRegReg(r179, d783.Reg)
			ctx.EmitSubInt64(r179, d782.Reg)
			d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
			ctx.BindReg(r179, &d784)
		}
		if d784.Loc == scm.LocReg && d783.Loc == scm.LocReg && d784.Reg == d783.Reg {
			ctx.TransferReg(d783.Reg)
			d783.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d782)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d781)
		ctx.EnsureDesc(&d784)
		var d785 scm.JITValueDesc
		if d781.Loc == scm.LocImm && d784.Loc == scm.LocImm {
			d785 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d781.Imm.Int()) >> uint64(d784.Imm.Int())))}
		} else if d784.Loc == scm.LocImm {
			r180 := ctx.AllocRegExcept(d781.Reg)
			ctx.EmitMovRegReg(r180, d781.Reg)
			ctx.EmitShrRegImm8(r180, uint8(d784.Imm.Int()))
			d785 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
			ctx.BindReg(r180, &d785)
		} else {
			{
				shiftSrc := d781.Reg
				r181 := ctx.AllocRegExcept(d781.Reg)
				ctx.EmitMovRegReg(r181, d781.Reg)
				shiftSrc = r181
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d784.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d784.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d784.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d785 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d785)
			}
		}
		if d785.Loc == scm.LocReg && d781.Loc == scm.LocReg && d785.Reg == d781.Reg {
			ctx.TransferReg(d781.Reg)
			d781.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d781)
		ctx.FreeDesc(&d784)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d785)
		ctx.FreeDesc(&d8)
		ctx.EnsureDesc(&d785)
		ctx.EnsureDesc(&d785)
		var d786 scm.JITValueDesc
		if d785.Loc == scm.LocImm {
			d786 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d785.Imm.Int()))))}
		} else {
			r182 := ctx.AllocReg()
			ctx.EmitMovRegReg(r182, d785.Reg)
			d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
			ctx.BindReg(r182, &d786)
		}
		ctx.FreeDesc(&d785)
		ctx.EnsureDesc(&d786)
		ctx.EnsureDesc(&d52)
		ctx.EnsureDescsTogether(&d786, &d52)
		var d787 scm.JITValueDesc
		if d786.Loc == scm.LocImm && d52.Loc == scm.LocImm {
			d787 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d786.Imm.Int() + d52.Imm.Int())}
		} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
			r183 := ctx.AllocRegExcept(d786.Reg)
			ctx.EmitMovRegReg(r183, d786.Reg)
			d787 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
			ctx.BindReg(r183, &d787)
		} else if d786.Loc == scm.LocImm && d786.Imm.Int() == 0 {
			d787 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			ctx.BindReg(d52.Reg, &d787)
		} else if d786.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d786.Imm.Int()))
			ctx.EmitAddInt64(scratch, d52.Reg)
			d787 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d787)
		} else if d52.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d786.Reg)
			ctx.EmitMovRegReg(scratch, d786.Reg)
			if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d52.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d787 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d787)
		} else {
			r184 := ctx.AllocRegExcept(d786.Reg, d52.Reg)
			ctx.EmitMovRegReg(r184, d786.Reg)
			ctx.EmitAddInt64(r184, d52.Reg)
			d787 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
			ctx.BindReg(r184, &d787)
		}
		if d787.Loc == scm.LocReg && d786.Loc == scm.LocReg && d787.Reg == d786.Reg {
			ctx.TransferReg(d786.Reg)
			d786.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d786)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d788 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d788 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r185 := ctx.AllocReg()
			ctx.EmitMovRegReg(r185, idxInt.Reg)
			ctx.EmitShlRegImm8(r185, 32)
			ctx.EmitShrRegImm8(r185, 32)
			d788 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
			ctx.BindReg(r185, &d788)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d788)
		ctx.EnsureDesc(&d787)
		ctx.EnsureDescsTogether(&d788, &d787)
		var d789 scm.JITValueDesc
		if d788.Loc == scm.LocImm && d787.Loc == scm.LocImm {
			d789 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d788.Imm.Int() - d787.Imm.Int())}
		} else if d787.Loc == scm.LocImm && d787.Imm.Int() == 0 {
			r186 := ctx.AllocRegExcept(d788.Reg)
			ctx.EmitMovRegReg(r186, d788.Reg)
			d789 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
			ctx.BindReg(r186, &d789)
		} else if d788.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d787.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d788.Imm.Int()))
			ctx.EmitSubInt64(scratch, d787.Reg)
			d789 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d789)
		} else if d787.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d788.Reg)
			ctx.EmitMovRegReg(scratch, d788.Reg)
			if d787.Imm.Int() >= -2147483648 && d787.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d787.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d787.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d789 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d789)
		} else {
			r187 := ctx.AllocRegExcept(d788.Reg, d787.Reg)
			ctx.EmitMovRegReg(r187, d788.Reg)
			ctx.EmitSubInt64(r187, d787.Reg)
			d789 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
			ctx.BindReg(r187, &d789)
		}
		if d789.Loc == scm.LocReg && d788.Loc == scm.LocReg && d789.Reg == d788.Reg {
			ctx.TransferReg(d788.Reg)
			d788.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d788)
		ctx.FreeDesc(&d787)
		ctx.EnsureDesc(&d789)
		ctx.EnsureDesc(&d763)
		ctx.EnsureDescsTogether(&d789, &d763)
		var d790 scm.JITValueDesc
		if d789.Loc == scm.LocImm && d763.Loc == scm.LocImm {
			d790 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d789.Imm.Int() * d763.Imm.Int())}
		} else if d789.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d763.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d789.Imm.Int()))
			ctx.EmitImulInt64(scratch, d763.Reg)
			d790 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d790)
		} else if d763.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d789.Reg)
			ctx.EmitMovRegReg(scratch, d789.Reg)
			if d763.Imm.Int() >= -2147483648 && d763.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d763.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d763.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d790 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d790)
		} else {
			r188 := ctx.AllocRegExcept(d789.Reg, d763.Reg)
			ctx.EmitMovRegReg(r188, d789.Reg)
			ctx.EmitImulInt64(r188, d763.Reg)
			d790 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
			ctx.BindReg(r188, &d790)
		}
		if d790.Loc == scm.LocReg && d789.Loc == scm.LocReg && d790.Reg == d789.Reg {
			ctx.TransferReg(d789.Reg)
			d789.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d789)
		ctx.FreeDesc(&d763)
		ctx.EnsureDesc(&d145)
		ctx.EnsureDesc(&d790)
		ctx.EnsureDescsTogether(&d145, &d790)
		var d791 scm.JITValueDesc
		if d145.Loc == scm.LocImm && d790.Loc == scm.LocImm {
			d791 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() + d790.Imm.Int())}
		} else if d790.Loc == scm.LocImm && d790.Imm.Int() == 0 {
			r189 := ctx.AllocRegExcept(d145.Reg)
			ctx.EmitMovRegReg(r189, d145.Reg)
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
			ctx.BindReg(r189, &d791)
		} else if d145.Loc == scm.LocImm && d145.Imm.Int() == 0 {
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d790.Reg}
			ctx.BindReg(d790.Reg, &d791)
		} else if d145.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d790.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d145.Imm.Int()))
			ctx.EmitAddInt64(scratch, d790.Reg)
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d791)
		} else if d790.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d145.Reg)
			ctx.EmitMovRegReg(scratch, d145.Reg)
			if d790.Imm.Int() >= -2147483648 && d790.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d790.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d790.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d791)
		} else {
			r190 := ctx.AllocRegExcept(d145.Reg, d790.Reg)
			ctx.EmitMovRegReg(r190, d145.Reg)
			ctx.EmitAddInt64(r190, d790.Reg)
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
			ctx.BindReg(r190, &d791)
		}
		if d791.Loc == scm.LocReg && d145.Loc == scm.LocReg && d791.Reg == d145.Reg {
			ctx.TransferReg(d145.Reg)
			d145.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d790)
		ctx.EnsureDesc(&d791)
		ctx.EnsureDesc(&d791)
		var d792 scm.JITValueDesc
		if d791.Loc == scm.LocImm {
			d792 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d791.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d791.Reg)
			d792 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d791.Reg}
			ctx.BindReg(d791.Reg, &d792)
		}
		ctx.FreeDesc(&d791)
		ctx.EnsureDesc(&d792)
		d793 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
		ctx.BindReg(r3, &d793)
		ctx.BindReg(r4, &d793)
		ctx.EnsureDesc(&d792)
		ctx.EmitMakeFloat(d793, d792)
		if d792.Loc == scm.LocReg {
			ctx.FreeReg(d792.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[13].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[13].VisitCount >= 0 {
				ps.General = true
				return bbs[13].RenderPS(ps)
			}
		}
		bbs[13].VisitCount++
		if ps.General {
			if bbs[13].Rendered {
				ctx.EmitJmp(lbl14)
				return result
			}
			bbs[13].Rendered = true
			bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_13 = bbs[13].Address
			ctx.MarkLabel(lbl14)
			ctx.ResolveFixups()
		}
		if phiHomeOK2 {
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0, ID: 0}
		} else {
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		}
		if phiHomeOK3 {
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1, ID: 0}
		} else {
			d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
		}
		if phiHomeOK4 {
			d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2, ID: 0}
		} else {
			d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
		}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
		d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(80)}
		d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(96)}
		d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(112)}
		d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(128)}
		if !ps.General && len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
			d5 = ps.OverlayValues[5]
		}
		if !ps.General && len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
			d6 = ps.OverlayValues[6]
		}
		if !ps.General && len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
			d7 = ps.OverlayValues[7]
		}
		if !ps.General && len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
			d8 = ps.OverlayValues[8]
		}
		if !ps.General && len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
			d9 = ps.OverlayValues[9]
		}
		if !ps.General && len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
			d10 = ps.OverlayValues[10]
		}
		if !ps.General && len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
			d11 = ps.OverlayValues[11]
		}
		if !ps.General && len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
			d12 = ps.OverlayValues[12]
		}
		if !ps.General && len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
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
		if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
			d59 = ps.OverlayValues[59]
		}
		if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
			d60 = ps.OverlayValues[60]
		}
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
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
		if len(ps.OverlayValues) > 125 && ps.OverlayValues[125].Loc != scm.LocNone {
			d125 = ps.OverlayValues[125]
		}
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
			d129 = ps.OverlayValues[129]
		}
		if len(ps.OverlayValues) > 130 && ps.OverlayValues[130].Loc != scm.LocNone {
			d130 = ps.OverlayValues[130]
		}
		if len(ps.OverlayValues) > 131 && ps.OverlayValues[131].Loc != scm.LocNone {
			d131 = ps.OverlayValues[131]
		}
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
			d150 = ps.OverlayValues[150]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
		}
		if len(ps.OverlayValues) > 239 && ps.OverlayValues[239].Loc != scm.LocNone {
			d239 = ps.OverlayValues[239]
		}
		if len(ps.OverlayValues) > 240 && ps.OverlayValues[240].Loc != scm.LocNone {
			d240 = ps.OverlayValues[240]
		}
		if len(ps.OverlayValues) > 241 && ps.OverlayValues[241].Loc != scm.LocNone {
			d241 = ps.OverlayValues[241]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		if len(ps.OverlayValues) > 246 && ps.OverlayValues[246].Loc != scm.LocNone {
			d246 = ps.OverlayValues[246]
		}
		if len(ps.OverlayValues) > 247 && ps.OverlayValues[247].Loc != scm.LocNone {
			d247 = ps.OverlayValues[247]
		}
		if len(ps.OverlayValues) > 248 && ps.OverlayValues[248].Loc != scm.LocNone {
			d248 = ps.OverlayValues[248]
		}
		if len(ps.OverlayValues) > 249 && ps.OverlayValues[249].Loc != scm.LocNone {
			d249 = ps.OverlayValues[249]
		}
		if len(ps.OverlayValues) > 250 && ps.OverlayValues[250].Loc != scm.LocNone {
			d250 = ps.OverlayValues[250]
		}
		if len(ps.OverlayValues) > 252 && ps.OverlayValues[252].Loc != scm.LocNone {
			d252 = ps.OverlayValues[252]
		}
		if len(ps.OverlayValues) > 254 && ps.OverlayValues[254].Loc != scm.LocNone {
			d254 = ps.OverlayValues[254]
		}
		if len(ps.OverlayValues) > 255 && ps.OverlayValues[255].Loc != scm.LocNone {
			d255 = ps.OverlayValues[255]
		}
		if len(ps.OverlayValues) > 256 && ps.OverlayValues[256].Loc != scm.LocNone {
			d256 = ps.OverlayValues[256]
		}
		if len(ps.OverlayValues) > 257 && ps.OverlayValues[257].Loc != scm.LocNone {
			d257 = ps.OverlayValues[257]
		}
		if len(ps.OverlayValues) > 258 && ps.OverlayValues[258].Loc != scm.LocNone {
			d258 = ps.OverlayValues[258]
		}
		if len(ps.OverlayValues) > 261 && ps.OverlayValues[261].Loc != scm.LocNone {
			d261 = ps.OverlayValues[261]
		}
		if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != scm.LocNone {
			d366 = ps.OverlayValues[366]
		}
		if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != scm.LocNone {
			d367 = ps.OverlayValues[367]
		}
		if len(ps.OverlayValues) > 368 && ps.OverlayValues[368].Loc != scm.LocNone {
			d368 = ps.OverlayValues[368]
		}
		if len(ps.OverlayValues) > 369 && ps.OverlayValues[369].Loc != scm.LocNone {
			d369 = ps.OverlayValues[369]
		}
		if len(ps.OverlayValues) > 370 && ps.OverlayValues[370].Loc != scm.LocNone {
			d370 = ps.OverlayValues[370]
		}
		if len(ps.OverlayValues) > 372 && ps.OverlayValues[372].Loc != scm.LocNone {
			d372 = ps.OverlayValues[372]
		}
		if len(ps.OverlayValues) > 373 && ps.OverlayValues[373].Loc != scm.LocNone {
			d373 = ps.OverlayValues[373]
		}
		if len(ps.OverlayValues) > 374 && ps.OverlayValues[374].Loc != scm.LocNone {
			d374 = ps.OverlayValues[374]
		}
		if len(ps.OverlayValues) > 375 && ps.OverlayValues[375].Loc != scm.LocNone {
			d375 = ps.OverlayValues[375]
		}
		if len(ps.OverlayValues) > 376 && ps.OverlayValues[376].Loc != scm.LocNone {
			d376 = ps.OverlayValues[376]
		}
		if len(ps.OverlayValues) > 377 && ps.OverlayValues[377].Loc != scm.LocNone {
			d377 = ps.OverlayValues[377]
		}
		if len(ps.OverlayValues) > 378 && ps.OverlayValues[378].Loc != scm.LocNone {
			d378 = ps.OverlayValues[378]
		}
		if len(ps.OverlayValues) > 379 && ps.OverlayValues[379].Loc != scm.LocNone {
			d379 = ps.OverlayValues[379]
		}
		if len(ps.OverlayValues) > 380 && ps.OverlayValues[380].Loc != scm.LocNone {
			d380 = ps.OverlayValues[380]
		}
		if len(ps.OverlayValues) > 381 && ps.OverlayValues[381].Loc != scm.LocNone {
			d381 = ps.OverlayValues[381]
		}
		if len(ps.OverlayValues) > 382 && ps.OverlayValues[382].Loc != scm.LocNone {
			d382 = ps.OverlayValues[382]
		}
		if len(ps.OverlayValues) > 383 && ps.OverlayValues[383].Loc != scm.LocNone {
			d383 = ps.OverlayValues[383]
		}
		if len(ps.OverlayValues) > 384 && ps.OverlayValues[384].Loc != scm.LocNone {
			d384 = ps.OverlayValues[384]
		}
		if len(ps.OverlayValues) > 385 && ps.OverlayValues[385].Loc != scm.LocNone {
			d385 = ps.OverlayValues[385]
		}
		if len(ps.OverlayValues) > 386 && ps.OverlayValues[386].Loc != scm.LocNone {
			d386 = ps.OverlayValues[386]
		}
		if len(ps.OverlayValues) > 387 && ps.OverlayValues[387].Loc != scm.LocNone {
			d387 = ps.OverlayValues[387]
		}
		if len(ps.OverlayValues) > 388 && ps.OverlayValues[388].Loc != scm.LocNone {
			d388 = ps.OverlayValues[388]
		}
		if len(ps.OverlayValues) > 389 && ps.OverlayValues[389].Loc != scm.LocNone {
			d389 = ps.OverlayValues[389]
		}
		if len(ps.OverlayValues) > 390 && ps.OverlayValues[390].Loc != scm.LocNone {
			d390 = ps.OverlayValues[390]
		}
		if len(ps.OverlayValues) > 391 && ps.OverlayValues[391].Loc != scm.LocNone {
			d391 = ps.OverlayValues[391]
		}
		if len(ps.OverlayValues) > 392 && ps.OverlayValues[392].Loc != scm.LocNone {
			d392 = ps.OverlayValues[392]
		}
		if len(ps.OverlayValues) > 393 && ps.OverlayValues[393].Loc != scm.LocNone {
			d393 = ps.OverlayValues[393]
		}
		if len(ps.OverlayValues) > 394 && ps.OverlayValues[394].Loc != scm.LocNone {
			d394 = ps.OverlayValues[394]
		}
		if len(ps.OverlayValues) > 395 && ps.OverlayValues[395].Loc != scm.LocNone {
			d395 = ps.OverlayValues[395]
		}
		if len(ps.OverlayValues) > 396 && ps.OverlayValues[396].Loc != scm.LocNone {
			d396 = ps.OverlayValues[396]
		}
		if len(ps.OverlayValues) > 397 && ps.OverlayValues[397].Loc != scm.LocNone {
			d397 = ps.OverlayValues[397]
		}
		if len(ps.OverlayValues) > 398 && ps.OverlayValues[398].Loc != scm.LocNone {
			d398 = ps.OverlayValues[398]
		}
		if len(ps.OverlayValues) > 399 && ps.OverlayValues[399].Loc != scm.LocNone {
			d399 = ps.OverlayValues[399]
		}
		if len(ps.OverlayValues) > 400 && ps.OverlayValues[400].Loc != scm.LocNone {
			d400 = ps.OverlayValues[400]
		}
		if len(ps.OverlayValues) > 543 && ps.OverlayValues[543].Loc != scm.LocNone {
			d543 = ps.OverlayValues[543]
		}
		if len(ps.OverlayValues) > 544 && ps.OverlayValues[544].Loc != scm.LocNone {
			d544 = ps.OverlayValues[544]
		}
		if len(ps.OverlayValues) > 545 && ps.OverlayValues[545].Loc != scm.LocNone {
			d545 = ps.OverlayValues[545]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 548 && ps.OverlayValues[548].Loc != scm.LocNone {
			d548 = ps.OverlayValues[548]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
		}
		if len(ps.OverlayValues) > 550 && ps.OverlayValues[550].Loc != scm.LocNone {
			d550 = ps.OverlayValues[550]
		}
		if len(ps.OverlayValues) > 551 && ps.OverlayValues[551].Loc != scm.LocNone {
			d551 = ps.OverlayValues[551]
		}
		if len(ps.OverlayValues) > 552 && ps.OverlayValues[552].Loc != scm.LocNone {
			d552 = ps.OverlayValues[552]
		}
		if len(ps.OverlayValues) > 553 && ps.OverlayValues[553].Loc != scm.LocNone {
			d553 = ps.OverlayValues[553]
		}
		if len(ps.OverlayValues) > 555 && ps.OverlayValues[555].Loc != scm.LocNone {
			d555 = ps.OverlayValues[555]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 558 && ps.OverlayValues[558].Loc != scm.LocNone {
			d558 = ps.OverlayValues[558]
		}
		if len(ps.OverlayValues) > 559 && ps.OverlayValues[559].Loc != scm.LocNone {
			d559 = ps.OverlayValues[559]
		}
		if len(ps.OverlayValues) > 560 && ps.OverlayValues[560].Loc != scm.LocNone {
			d560 = ps.OverlayValues[560]
		}
		if len(ps.OverlayValues) > 563 && ps.OverlayValues[563].Loc != scm.LocNone {
			d563 = ps.OverlayValues[563]
		}
		if len(ps.OverlayValues) > 718 && ps.OverlayValues[718].Loc != scm.LocNone {
			d718 = ps.OverlayValues[718]
		}
		if len(ps.OverlayValues) > 719 && ps.OverlayValues[719].Loc != scm.LocNone {
			d719 = ps.OverlayValues[719]
		}
		if len(ps.OverlayValues) > 720 && ps.OverlayValues[720].Loc != scm.LocNone {
			d720 = ps.OverlayValues[720]
		}
		if len(ps.OverlayValues) > 721 && ps.OverlayValues[721].Loc != scm.LocNone {
			d721 = ps.OverlayValues[721]
		}
		if len(ps.OverlayValues) > 723 && ps.OverlayValues[723].Loc != scm.LocNone {
			d723 = ps.OverlayValues[723]
		}
		if len(ps.OverlayValues) > 724 && ps.OverlayValues[724].Loc != scm.LocNone {
			d724 = ps.OverlayValues[724]
		}
		if len(ps.OverlayValues) > 725 && ps.OverlayValues[725].Loc != scm.LocNone {
			d725 = ps.OverlayValues[725]
		}
		if len(ps.OverlayValues) > 726 && ps.OverlayValues[726].Loc != scm.LocNone {
			d726 = ps.OverlayValues[726]
		}
		if len(ps.OverlayValues) > 727 && ps.OverlayValues[727].Loc != scm.LocNone {
			d727 = ps.OverlayValues[727]
		}
		if len(ps.OverlayValues) > 728 && ps.OverlayValues[728].Loc != scm.LocNone {
			d728 = ps.OverlayValues[728]
		}
		if len(ps.OverlayValues) > 729 && ps.OverlayValues[729].Loc != scm.LocNone {
			d729 = ps.OverlayValues[729]
		}
		if len(ps.OverlayValues) > 730 && ps.OverlayValues[730].Loc != scm.LocNone {
			d730 = ps.OverlayValues[730]
		}
		if len(ps.OverlayValues) > 731 && ps.OverlayValues[731].Loc != scm.LocNone {
			d731 = ps.OverlayValues[731]
		}
		if len(ps.OverlayValues) > 732 && ps.OverlayValues[732].Loc != scm.LocNone {
			d732 = ps.OverlayValues[732]
		}
		if len(ps.OverlayValues) > 734 && ps.OverlayValues[734].Loc != scm.LocNone {
			d734 = ps.OverlayValues[734]
		}
		if len(ps.OverlayValues) > 735 && ps.OverlayValues[735].Loc != scm.LocNone {
			d735 = ps.OverlayValues[735]
		}
		if len(ps.OverlayValues) > 736 && ps.OverlayValues[736].Loc != scm.LocNone {
			d736 = ps.OverlayValues[736]
		}
		if len(ps.OverlayValues) > 737 && ps.OverlayValues[737].Loc != scm.LocNone {
			d737 = ps.OverlayValues[737]
		}
		if len(ps.OverlayValues) > 738 && ps.OverlayValues[738].Loc != scm.LocNone {
			d738 = ps.OverlayValues[738]
		}
		if len(ps.OverlayValues) > 739 && ps.OverlayValues[739].Loc != scm.LocNone {
			d739 = ps.OverlayValues[739]
		}
		if len(ps.OverlayValues) > 740 && ps.OverlayValues[740].Loc != scm.LocNone {
			d740 = ps.OverlayValues[740]
		}
		if len(ps.OverlayValues) > 741 && ps.OverlayValues[741].Loc != scm.LocNone {
			d741 = ps.OverlayValues[741]
		}
		if len(ps.OverlayValues) > 742 && ps.OverlayValues[742].Loc != scm.LocNone {
			d742 = ps.OverlayValues[742]
		}
		if len(ps.OverlayValues) > 743 && ps.OverlayValues[743].Loc != scm.LocNone {
			d743 = ps.OverlayValues[743]
		}
		if len(ps.OverlayValues) > 744 && ps.OverlayValues[744].Loc != scm.LocNone {
			d744 = ps.OverlayValues[744]
		}
		if len(ps.OverlayValues) > 745 && ps.OverlayValues[745].Loc != scm.LocNone {
			d745 = ps.OverlayValues[745]
		}
		if len(ps.OverlayValues) > 746 && ps.OverlayValues[746].Loc != scm.LocNone {
			d746 = ps.OverlayValues[746]
		}
		if len(ps.OverlayValues) > 747 && ps.OverlayValues[747].Loc != scm.LocNone {
			d747 = ps.OverlayValues[747]
		}
		if len(ps.OverlayValues) > 748 && ps.OverlayValues[748].Loc != scm.LocNone {
			d748 = ps.OverlayValues[748]
		}
		if len(ps.OverlayValues) > 749 && ps.OverlayValues[749].Loc != scm.LocNone {
			d749 = ps.OverlayValues[749]
		}
		if len(ps.OverlayValues) > 750 && ps.OverlayValues[750].Loc != scm.LocNone {
			d750 = ps.OverlayValues[750]
		}
		if len(ps.OverlayValues) > 751 && ps.OverlayValues[751].Loc != scm.LocNone {
			d751 = ps.OverlayValues[751]
		}
		if len(ps.OverlayValues) > 752 && ps.OverlayValues[752].Loc != scm.LocNone {
			d752 = ps.OverlayValues[752]
		}
		if len(ps.OverlayValues) > 753 && ps.OverlayValues[753].Loc != scm.LocNone {
			d753 = ps.OverlayValues[753]
		}
		if len(ps.OverlayValues) > 754 && ps.OverlayValues[754].Loc != scm.LocNone {
			d754 = ps.OverlayValues[754]
		}
		if len(ps.OverlayValues) > 755 && ps.OverlayValues[755].Loc != scm.LocNone {
			d755 = ps.OverlayValues[755]
		}
		if len(ps.OverlayValues) > 756 && ps.OverlayValues[756].Loc != scm.LocNone {
			d756 = ps.OverlayValues[756]
		}
		if len(ps.OverlayValues) > 757 && ps.OverlayValues[757].Loc != scm.LocNone {
			d757 = ps.OverlayValues[757]
		}
		if len(ps.OverlayValues) > 758 && ps.OverlayValues[758].Loc != scm.LocNone {
			d758 = ps.OverlayValues[758]
		}
		if len(ps.OverlayValues) > 759 && ps.OverlayValues[759].Loc != scm.LocNone {
			d759 = ps.OverlayValues[759]
		}
		if len(ps.OverlayValues) > 760 && ps.OverlayValues[760].Loc != scm.LocNone {
			d760 = ps.OverlayValues[760]
		}
		if len(ps.OverlayValues) > 761 && ps.OverlayValues[761].Loc != scm.LocNone {
			d761 = ps.OverlayValues[761]
		}
		if len(ps.OverlayValues) > 762 && ps.OverlayValues[762].Loc != scm.LocNone {
			d762 = ps.OverlayValues[762]
		}
		if len(ps.OverlayValues) > 763 && ps.OverlayValues[763].Loc != scm.LocNone {
			d763 = ps.OverlayValues[763]
		}
		if len(ps.OverlayValues) > 764 && ps.OverlayValues[764].Loc != scm.LocNone {
			d764 = ps.OverlayValues[764]
		}
		if len(ps.OverlayValues) > 765 && ps.OverlayValues[765].Loc != scm.LocNone {
			d765 = ps.OverlayValues[765]
		}
		if len(ps.OverlayValues) > 766 && ps.OverlayValues[766].Loc != scm.LocNone {
			d766 = ps.OverlayValues[766]
		}
		if len(ps.OverlayValues) > 767 && ps.OverlayValues[767].Loc != scm.LocNone {
			d767 = ps.OverlayValues[767]
		}
		if len(ps.OverlayValues) > 768 && ps.OverlayValues[768].Loc != scm.LocNone {
			d768 = ps.OverlayValues[768]
		}
		if len(ps.OverlayValues) > 769 && ps.OverlayValues[769].Loc != scm.LocNone {
			d769 = ps.OverlayValues[769]
		}
		if len(ps.OverlayValues) > 770 && ps.OverlayValues[770].Loc != scm.LocNone {
			d770 = ps.OverlayValues[770]
		}
		if len(ps.OverlayValues) > 771 && ps.OverlayValues[771].Loc != scm.LocNone {
			d771 = ps.OverlayValues[771]
		}
		if len(ps.OverlayValues) > 772 && ps.OverlayValues[772].Loc != scm.LocNone {
			d772 = ps.OverlayValues[772]
		}
		if len(ps.OverlayValues) > 773 && ps.OverlayValues[773].Loc != scm.LocNone {
			d773 = ps.OverlayValues[773]
		}
		if len(ps.OverlayValues) > 774 && ps.OverlayValues[774].Loc != scm.LocNone {
			d774 = ps.OverlayValues[774]
		}
		if len(ps.OverlayValues) > 775 && ps.OverlayValues[775].Loc != scm.LocNone {
			d775 = ps.OverlayValues[775]
		}
		if len(ps.OverlayValues) > 776 && ps.OverlayValues[776].Loc != scm.LocNone {
			d776 = ps.OverlayValues[776]
		}
		if len(ps.OverlayValues) > 777 && ps.OverlayValues[777].Loc != scm.LocNone {
			d777 = ps.OverlayValues[777]
		}
		if len(ps.OverlayValues) > 778 && ps.OverlayValues[778].Loc != scm.LocNone {
			d778 = ps.OverlayValues[778]
		}
		if len(ps.OverlayValues) > 779 && ps.OverlayValues[779].Loc != scm.LocNone {
			d779 = ps.OverlayValues[779]
		}
		if len(ps.OverlayValues) > 780 && ps.OverlayValues[780].Loc != scm.LocNone {
			d780 = ps.OverlayValues[780]
		}
		if len(ps.OverlayValues) > 781 && ps.OverlayValues[781].Loc != scm.LocNone {
			d781 = ps.OverlayValues[781]
		}
		if len(ps.OverlayValues) > 782 && ps.OverlayValues[782].Loc != scm.LocNone {
			d782 = ps.OverlayValues[782]
		}
		if len(ps.OverlayValues) > 783 && ps.OverlayValues[783].Loc != scm.LocNone {
			d783 = ps.OverlayValues[783]
		}
		if len(ps.OverlayValues) > 784 && ps.OverlayValues[784].Loc != scm.LocNone {
			d784 = ps.OverlayValues[784]
		}
		if len(ps.OverlayValues) > 785 && ps.OverlayValues[785].Loc != scm.LocNone {
			d785 = ps.OverlayValues[785]
		}
		if len(ps.OverlayValues) > 786 && ps.OverlayValues[786].Loc != scm.LocNone {
			d786 = ps.OverlayValues[786]
		}
		if len(ps.OverlayValues) > 787 && ps.OverlayValues[787].Loc != scm.LocNone {
			d787 = ps.OverlayValues[787]
		}
		if len(ps.OverlayValues) > 788 && ps.OverlayValues[788].Loc != scm.LocNone {
			d788 = ps.OverlayValues[788]
		}
		if len(ps.OverlayValues) > 789 && ps.OverlayValues[789].Loc != scm.LocNone {
			d789 = ps.OverlayValues[789]
		}
		if len(ps.OverlayValues) > 790 && ps.OverlayValues[790].Loc != scm.LocNone {
			d790 = ps.OverlayValues[790]
		}
		if len(ps.OverlayValues) > 791 && ps.OverlayValues[791].Loc != scm.LocNone {
			d791 = ps.OverlayValues[791]
		}
		if len(ps.OverlayValues) > 792 && ps.OverlayValues[792].Loc != scm.LocNone {
			d792 = ps.OverlayValues[792]
		}
		if len(ps.OverlayValues) > 793 && ps.OverlayValues[793].Loc != scm.LocNone {
			d793 = ps.OverlayValues[793]
		}
		ctx.ReclaimUntrackedRegs()
		var d794 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
			r191 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r191, fieldAddr)
			d794 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r191}
			ctx.BindReg(r191, &d794)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
			r192 := ctx.AllocReg()
			ctx.EmitMovRegMem(r192, thisptr.Reg, off)
			d794 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r192}
			ctx.BindReg(r192, &d794)
		}
		ctx.EnsureDesc(&d794)
		ctx.EnsureDesc(&d794)
		var d795 scm.JITValueDesc
		if d794.Loc == scm.LocImm {
			d795 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d794.Imm.Int()))))}
		} else {
			r193 := ctx.AllocReg()
			ctx.EmitMovRegReg(r193, d794.Reg)
			d795 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
			ctx.BindReg(r193, &d795)
		}
		ctx.EnsureDesc(&d145)
		ctx.EnsureDesc(&d795)
		ctx.EnsureDescsTogether(&d145, &d795)
		var d796 scm.JITValueDesc
		if d145.Loc == scm.LocImm && d795.Loc == scm.LocImm {
			d796 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d145.Imm.Int() == d795.Imm.Int())}
		} else if d795.Loc == scm.LocImm {
			r194 := ctx.AllocRegExcept(d145.Reg)
			if d795.Imm.Int() >= -2147483648 && d795.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d145.Reg, int32(d795.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d795.Imm.Int()))
				ctx.EmitCmpInt64(d145.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r194, scm.CondEqual)
			d796 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r194}
			ctx.BindReg(r194, &d796)
		} else if d145.Loc == scm.LocImm {
			r195 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d795.Reg)
			ctx.EmitSetcc(r195, scm.CondEqual)
			d796 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r195}
			ctx.BindReg(r195, &d796)
		} else {
			r196 := ctx.AllocRegExcept(d145.Reg)
			ctx.EmitCmpInt64(d145.Reg, d795.Reg)
			ctx.EmitSetcc(r196, scm.CondEqual)
			d796 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r196}
			ctx.BindReg(r196, &d796)
		}
		ctx.FreeDesc(&d145)
		ctx.FreeDesc(&d795)
		d797 = d796
		ctx.EnsureDesc(&d797)
		if d797.Loc != scm.LocImm && d797.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d797.Loc == scm.LocImm {
			if d797.Imm.Bool() {
				if ps.General {
				}
				ps798 := scm.PhiState{General: ps.General}
				ps798.OverlayValues = make([]scm.JITValueDesc, 798)
				ps798.OverlayValues[5] = d5
				ps798.OverlayValues[6] = d6
				ps798.OverlayValues[7] = d7
				ps798.OverlayValues[8] = d8
				ps798.OverlayValues[9] = d9
				ps798.OverlayValues[10] = d10
				ps798.OverlayValues[11] = d11
				ps798.OverlayValues[12] = d12
				ps798.OverlayValues[13] = d13
				ps798.OverlayValues[14] = d14
				ps798.OverlayValues[15] = d15
				ps798.OverlayValues[16] = d16
				ps798.OverlayValues[17] = d17
				ps798.OverlayValues[18] = d18
				ps798.OverlayValues[19] = d19
				ps798.OverlayValues[20] = d20
				ps798.OverlayValues[21] = d21
				ps798.OverlayValues[23] = d23
				ps798.OverlayValues[24] = d24
				ps798.OverlayValues[25] = d25
				ps798.OverlayValues[26] = d26
				ps798.OverlayValues[27] = d27
				ps798.OverlayValues[28] = d28
				ps798.OverlayValues[29] = d29
				ps798.OverlayValues[30] = d30
				ps798.OverlayValues[31] = d31
				ps798.OverlayValues[32] = d32
				ps798.OverlayValues[33] = d33
				ps798.OverlayValues[34] = d34
				ps798.OverlayValues[35] = d35
				ps798.OverlayValues[36] = d36
				ps798.OverlayValues[37] = d37
				ps798.OverlayValues[38] = d38
				ps798.OverlayValues[39] = d39
				ps798.OverlayValues[40] = d40
				ps798.OverlayValues[41] = d41
				ps798.OverlayValues[42] = d42
				ps798.OverlayValues[43] = d43
				ps798.OverlayValues[44] = d44
				ps798.OverlayValues[45] = d45
				ps798.OverlayValues[46] = d46
				ps798.OverlayValues[47] = d47
				ps798.OverlayValues[48] = d48
				ps798.OverlayValues[49] = d49
				ps798.OverlayValues[50] = d50
				ps798.OverlayValues[51] = d51
				ps798.OverlayValues[52] = d52
				ps798.OverlayValues[53] = d53
				ps798.OverlayValues[54] = d54
				ps798.OverlayValues[55] = d55
				ps798.OverlayValues[56] = d56
				ps798.OverlayValues[59] = d59
				ps798.OverlayValues[60] = d60
				ps798.OverlayValues[61] = d61
				ps798.OverlayValues[119] = d119
				ps798.OverlayValues[120] = d120
				ps798.OverlayValues[121] = d121
				ps798.OverlayValues[122] = d122
				ps798.OverlayValues[123] = d123
				ps798.OverlayValues[124] = d124
				ps798.OverlayValues[125] = d125
				ps798.OverlayValues[126] = d126
				ps798.OverlayValues[127] = d127
				ps798.OverlayValues[128] = d128
				ps798.OverlayValues[129] = d129
				ps798.OverlayValues[130] = d130
				ps798.OverlayValues[131] = d131
				ps798.OverlayValues[132] = d132
				ps798.OverlayValues[133] = d133
				ps798.OverlayValues[134] = d134
				ps798.OverlayValues[135] = d135
				ps798.OverlayValues[136] = d136
				ps798.OverlayValues[137] = d137
				ps798.OverlayValues[138] = d138
				ps798.OverlayValues[139] = d139
				ps798.OverlayValues[140] = d140
				ps798.OverlayValues[141] = d141
				ps798.OverlayValues[142] = d142
				ps798.OverlayValues[143] = d143
				ps798.OverlayValues[144] = d144
				ps798.OverlayValues[145] = d145
				ps798.OverlayValues[146] = d146
				ps798.OverlayValues[147] = d147
				ps798.OverlayValues[150] = d150
				ps798.OverlayValues[238] = d238
				ps798.OverlayValues[239] = d239
				ps798.OverlayValues[240] = d240
				ps798.OverlayValues[241] = d241
				ps798.OverlayValues[243] = d243
				ps798.OverlayValues[244] = d244
				ps798.OverlayValues[245] = d245
				ps798.OverlayValues[246] = d246
				ps798.OverlayValues[247] = d247
				ps798.OverlayValues[248] = d248
				ps798.OverlayValues[249] = d249
				ps798.OverlayValues[250] = d250
				ps798.OverlayValues[252] = d252
				ps798.OverlayValues[254] = d254
				ps798.OverlayValues[255] = d255
				ps798.OverlayValues[256] = d256
				ps798.OverlayValues[257] = d257
				ps798.OverlayValues[258] = d258
				ps798.OverlayValues[261] = d261
				ps798.OverlayValues[366] = d366
				ps798.OverlayValues[367] = d367
				ps798.OverlayValues[368] = d368
				ps798.OverlayValues[369] = d369
				ps798.OverlayValues[370] = d370
				ps798.OverlayValues[372] = d372
				ps798.OverlayValues[373] = d373
				ps798.OverlayValues[374] = d374
				ps798.OverlayValues[375] = d375
				ps798.OverlayValues[376] = d376
				ps798.OverlayValues[377] = d377
				ps798.OverlayValues[378] = d378
				ps798.OverlayValues[379] = d379
				ps798.OverlayValues[380] = d380
				ps798.OverlayValues[381] = d381
				ps798.OverlayValues[382] = d382
				ps798.OverlayValues[383] = d383
				ps798.OverlayValues[384] = d384
				ps798.OverlayValues[385] = d385
				ps798.OverlayValues[386] = d386
				ps798.OverlayValues[387] = d387
				ps798.OverlayValues[388] = d388
				ps798.OverlayValues[389] = d389
				ps798.OverlayValues[390] = d390
				ps798.OverlayValues[391] = d391
				ps798.OverlayValues[392] = d392
				ps798.OverlayValues[393] = d393
				ps798.OverlayValues[394] = d394
				ps798.OverlayValues[395] = d395
				ps798.OverlayValues[396] = d396
				ps798.OverlayValues[397] = d397
				ps798.OverlayValues[398] = d398
				ps798.OverlayValues[399] = d399
				ps798.OverlayValues[400] = d400
				ps798.OverlayValues[543] = d543
				ps798.OverlayValues[544] = d544
				ps798.OverlayValues[545] = d545
				ps798.OverlayValues[547] = d547
				ps798.OverlayValues[548] = d548
				ps798.OverlayValues[549] = d549
				ps798.OverlayValues[550] = d550
				ps798.OverlayValues[551] = d551
				ps798.OverlayValues[552] = d552
				ps798.OverlayValues[553] = d553
				ps798.OverlayValues[555] = d555
				ps798.OverlayValues[557] = d557
				ps798.OverlayValues[558] = d558
				ps798.OverlayValues[559] = d559
				ps798.OverlayValues[560] = d560
				ps798.OverlayValues[563] = d563
				ps798.OverlayValues[718] = d718
				ps798.OverlayValues[719] = d719
				ps798.OverlayValues[720] = d720
				ps798.OverlayValues[721] = d721
				ps798.OverlayValues[723] = d723
				ps798.OverlayValues[724] = d724
				ps798.OverlayValues[725] = d725
				ps798.OverlayValues[726] = d726
				ps798.OverlayValues[727] = d727
				ps798.OverlayValues[728] = d728
				ps798.OverlayValues[729] = d729
				ps798.OverlayValues[730] = d730
				ps798.OverlayValues[731] = d731
				ps798.OverlayValues[732] = d732
				ps798.OverlayValues[734] = d734
				ps798.OverlayValues[735] = d735
				ps798.OverlayValues[736] = d736
				ps798.OverlayValues[737] = d737
				ps798.OverlayValues[738] = d738
				ps798.OverlayValues[739] = d739
				ps798.OverlayValues[740] = d740
				ps798.OverlayValues[741] = d741
				ps798.OverlayValues[742] = d742
				ps798.OverlayValues[743] = d743
				ps798.OverlayValues[744] = d744
				ps798.OverlayValues[745] = d745
				ps798.OverlayValues[746] = d746
				ps798.OverlayValues[747] = d747
				ps798.OverlayValues[748] = d748
				ps798.OverlayValues[749] = d749
				ps798.OverlayValues[750] = d750
				ps798.OverlayValues[751] = d751
				ps798.OverlayValues[752] = d752
				ps798.OverlayValues[753] = d753
				ps798.OverlayValues[754] = d754
				ps798.OverlayValues[755] = d755
				ps798.OverlayValues[756] = d756
				ps798.OverlayValues[757] = d757
				ps798.OverlayValues[758] = d758
				ps798.OverlayValues[759] = d759
				ps798.OverlayValues[760] = d760
				ps798.OverlayValues[761] = d761
				ps798.OverlayValues[762] = d762
				ps798.OverlayValues[763] = d763
				ps798.OverlayValues[764] = d764
				ps798.OverlayValues[765] = d765
				ps798.OverlayValues[766] = d766
				ps798.OverlayValues[767] = d767
				ps798.OverlayValues[768] = d768
				ps798.OverlayValues[769] = d769
				ps798.OverlayValues[770] = d770
				ps798.OverlayValues[771] = d771
				ps798.OverlayValues[772] = d772
				ps798.OverlayValues[773] = d773
				ps798.OverlayValues[774] = d774
				ps798.OverlayValues[775] = d775
				ps798.OverlayValues[776] = d776
				ps798.OverlayValues[777] = d777
				ps798.OverlayValues[778] = d778
				ps798.OverlayValues[779] = d779
				ps798.OverlayValues[780] = d780
				ps798.OverlayValues[781] = d781
				ps798.OverlayValues[782] = d782
				ps798.OverlayValues[783] = d783
				ps798.OverlayValues[784] = d784
				ps798.OverlayValues[785] = d785
				ps798.OverlayValues[786] = d786
				ps798.OverlayValues[787] = d787
				ps798.OverlayValues[788] = d788
				ps798.OverlayValues[789] = d789
				ps798.OverlayValues[790] = d790
				ps798.OverlayValues[791] = d791
				ps798.OverlayValues[792] = d792
				ps798.OverlayValues[793] = d793
				ps798.OverlayValues[794] = d794
				ps798.OverlayValues[795] = d795
				ps798.OverlayValues[796] = d796
				ps798.OverlayValues[797] = d797
				return bbs[11].RenderPS(ps798)
			}
			if ps.General {
			}
			ps799 := scm.PhiState{General: ps.General}
			ps799.OverlayValues = make([]scm.JITValueDesc, 798)
			ps799.OverlayValues[5] = d5
			ps799.OverlayValues[6] = d6
			ps799.OverlayValues[7] = d7
			ps799.OverlayValues[8] = d8
			ps799.OverlayValues[9] = d9
			ps799.OverlayValues[10] = d10
			ps799.OverlayValues[11] = d11
			ps799.OverlayValues[12] = d12
			ps799.OverlayValues[13] = d13
			ps799.OverlayValues[14] = d14
			ps799.OverlayValues[15] = d15
			ps799.OverlayValues[16] = d16
			ps799.OverlayValues[17] = d17
			ps799.OverlayValues[18] = d18
			ps799.OverlayValues[19] = d19
			ps799.OverlayValues[20] = d20
			ps799.OverlayValues[21] = d21
			ps799.OverlayValues[23] = d23
			ps799.OverlayValues[24] = d24
			ps799.OverlayValues[25] = d25
			ps799.OverlayValues[26] = d26
			ps799.OverlayValues[27] = d27
			ps799.OverlayValues[28] = d28
			ps799.OverlayValues[29] = d29
			ps799.OverlayValues[30] = d30
			ps799.OverlayValues[31] = d31
			ps799.OverlayValues[32] = d32
			ps799.OverlayValues[33] = d33
			ps799.OverlayValues[34] = d34
			ps799.OverlayValues[35] = d35
			ps799.OverlayValues[36] = d36
			ps799.OverlayValues[37] = d37
			ps799.OverlayValues[38] = d38
			ps799.OverlayValues[39] = d39
			ps799.OverlayValues[40] = d40
			ps799.OverlayValues[41] = d41
			ps799.OverlayValues[42] = d42
			ps799.OverlayValues[43] = d43
			ps799.OverlayValues[44] = d44
			ps799.OverlayValues[45] = d45
			ps799.OverlayValues[46] = d46
			ps799.OverlayValues[47] = d47
			ps799.OverlayValues[48] = d48
			ps799.OverlayValues[49] = d49
			ps799.OverlayValues[50] = d50
			ps799.OverlayValues[51] = d51
			ps799.OverlayValues[52] = d52
			ps799.OverlayValues[53] = d53
			ps799.OverlayValues[54] = d54
			ps799.OverlayValues[55] = d55
			ps799.OverlayValues[56] = d56
			ps799.OverlayValues[59] = d59
			ps799.OverlayValues[60] = d60
			ps799.OverlayValues[61] = d61
			ps799.OverlayValues[119] = d119
			ps799.OverlayValues[120] = d120
			ps799.OverlayValues[121] = d121
			ps799.OverlayValues[122] = d122
			ps799.OverlayValues[123] = d123
			ps799.OverlayValues[124] = d124
			ps799.OverlayValues[125] = d125
			ps799.OverlayValues[126] = d126
			ps799.OverlayValues[127] = d127
			ps799.OverlayValues[128] = d128
			ps799.OverlayValues[129] = d129
			ps799.OverlayValues[130] = d130
			ps799.OverlayValues[131] = d131
			ps799.OverlayValues[132] = d132
			ps799.OverlayValues[133] = d133
			ps799.OverlayValues[134] = d134
			ps799.OverlayValues[135] = d135
			ps799.OverlayValues[136] = d136
			ps799.OverlayValues[137] = d137
			ps799.OverlayValues[138] = d138
			ps799.OverlayValues[139] = d139
			ps799.OverlayValues[140] = d140
			ps799.OverlayValues[141] = d141
			ps799.OverlayValues[142] = d142
			ps799.OverlayValues[143] = d143
			ps799.OverlayValues[144] = d144
			ps799.OverlayValues[145] = d145
			ps799.OverlayValues[146] = d146
			ps799.OverlayValues[147] = d147
			ps799.OverlayValues[150] = d150
			ps799.OverlayValues[238] = d238
			ps799.OverlayValues[239] = d239
			ps799.OverlayValues[240] = d240
			ps799.OverlayValues[241] = d241
			ps799.OverlayValues[243] = d243
			ps799.OverlayValues[244] = d244
			ps799.OverlayValues[245] = d245
			ps799.OverlayValues[246] = d246
			ps799.OverlayValues[247] = d247
			ps799.OverlayValues[248] = d248
			ps799.OverlayValues[249] = d249
			ps799.OverlayValues[250] = d250
			ps799.OverlayValues[252] = d252
			ps799.OverlayValues[254] = d254
			ps799.OverlayValues[255] = d255
			ps799.OverlayValues[256] = d256
			ps799.OverlayValues[257] = d257
			ps799.OverlayValues[258] = d258
			ps799.OverlayValues[261] = d261
			ps799.OverlayValues[366] = d366
			ps799.OverlayValues[367] = d367
			ps799.OverlayValues[368] = d368
			ps799.OverlayValues[369] = d369
			ps799.OverlayValues[370] = d370
			ps799.OverlayValues[372] = d372
			ps799.OverlayValues[373] = d373
			ps799.OverlayValues[374] = d374
			ps799.OverlayValues[375] = d375
			ps799.OverlayValues[376] = d376
			ps799.OverlayValues[377] = d377
			ps799.OverlayValues[378] = d378
			ps799.OverlayValues[379] = d379
			ps799.OverlayValues[380] = d380
			ps799.OverlayValues[381] = d381
			ps799.OverlayValues[382] = d382
			ps799.OverlayValues[383] = d383
			ps799.OverlayValues[384] = d384
			ps799.OverlayValues[385] = d385
			ps799.OverlayValues[386] = d386
			ps799.OverlayValues[387] = d387
			ps799.OverlayValues[388] = d388
			ps799.OverlayValues[389] = d389
			ps799.OverlayValues[390] = d390
			ps799.OverlayValues[391] = d391
			ps799.OverlayValues[392] = d392
			ps799.OverlayValues[393] = d393
			ps799.OverlayValues[394] = d394
			ps799.OverlayValues[395] = d395
			ps799.OverlayValues[396] = d396
			ps799.OverlayValues[397] = d397
			ps799.OverlayValues[398] = d398
			ps799.OverlayValues[399] = d399
			ps799.OverlayValues[400] = d400
			ps799.OverlayValues[543] = d543
			ps799.OverlayValues[544] = d544
			ps799.OverlayValues[545] = d545
			ps799.OverlayValues[547] = d547
			ps799.OverlayValues[548] = d548
			ps799.OverlayValues[549] = d549
			ps799.OverlayValues[550] = d550
			ps799.OverlayValues[551] = d551
			ps799.OverlayValues[552] = d552
			ps799.OverlayValues[553] = d553
			ps799.OverlayValues[555] = d555
			ps799.OverlayValues[557] = d557
			ps799.OverlayValues[558] = d558
			ps799.OverlayValues[559] = d559
			ps799.OverlayValues[560] = d560
			ps799.OverlayValues[563] = d563
			ps799.OverlayValues[718] = d718
			ps799.OverlayValues[719] = d719
			ps799.OverlayValues[720] = d720
			ps799.OverlayValues[721] = d721
			ps799.OverlayValues[723] = d723
			ps799.OverlayValues[724] = d724
			ps799.OverlayValues[725] = d725
			ps799.OverlayValues[726] = d726
			ps799.OverlayValues[727] = d727
			ps799.OverlayValues[728] = d728
			ps799.OverlayValues[729] = d729
			ps799.OverlayValues[730] = d730
			ps799.OverlayValues[731] = d731
			ps799.OverlayValues[732] = d732
			ps799.OverlayValues[734] = d734
			ps799.OverlayValues[735] = d735
			ps799.OverlayValues[736] = d736
			ps799.OverlayValues[737] = d737
			ps799.OverlayValues[738] = d738
			ps799.OverlayValues[739] = d739
			ps799.OverlayValues[740] = d740
			ps799.OverlayValues[741] = d741
			ps799.OverlayValues[742] = d742
			ps799.OverlayValues[743] = d743
			ps799.OverlayValues[744] = d744
			ps799.OverlayValues[745] = d745
			ps799.OverlayValues[746] = d746
			ps799.OverlayValues[747] = d747
			ps799.OverlayValues[748] = d748
			ps799.OverlayValues[749] = d749
			ps799.OverlayValues[750] = d750
			ps799.OverlayValues[751] = d751
			ps799.OverlayValues[752] = d752
			ps799.OverlayValues[753] = d753
			ps799.OverlayValues[754] = d754
			ps799.OverlayValues[755] = d755
			ps799.OverlayValues[756] = d756
			ps799.OverlayValues[757] = d757
			ps799.OverlayValues[758] = d758
			ps799.OverlayValues[759] = d759
			ps799.OverlayValues[760] = d760
			ps799.OverlayValues[761] = d761
			ps799.OverlayValues[762] = d762
			ps799.OverlayValues[763] = d763
			ps799.OverlayValues[764] = d764
			ps799.OverlayValues[765] = d765
			ps799.OverlayValues[766] = d766
			ps799.OverlayValues[767] = d767
			ps799.OverlayValues[768] = d768
			ps799.OverlayValues[769] = d769
			ps799.OverlayValues[770] = d770
			ps799.OverlayValues[771] = d771
			ps799.OverlayValues[772] = d772
			ps799.OverlayValues[773] = d773
			ps799.OverlayValues[774] = d774
			ps799.OverlayValues[775] = d775
			ps799.OverlayValues[776] = d776
			ps799.OverlayValues[777] = d777
			ps799.OverlayValues[778] = d778
			ps799.OverlayValues[779] = d779
			ps799.OverlayValues[780] = d780
			ps799.OverlayValues[781] = d781
			ps799.OverlayValues[782] = d782
			ps799.OverlayValues[783] = d783
			ps799.OverlayValues[784] = d784
			ps799.OverlayValues[785] = d785
			ps799.OverlayValues[786] = d786
			ps799.OverlayValues[787] = d787
			ps799.OverlayValues[788] = d788
			ps799.OverlayValues[789] = d789
			ps799.OverlayValues[790] = d790
			ps799.OverlayValues[791] = d791
			ps799.OverlayValues[792] = d792
			ps799.OverlayValues[793] = d793
			ps799.OverlayValues[794] = d794
			ps799.OverlayValues[795] = d795
			ps799.OverlayValues[796] = d796
			ps799.OverlayValues[797] = d797
			return bbs[12].RenderPS(ps799)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl30 := ctx.ReserveLabel()
		lbl31 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d797.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl30)
		ctx.EmitJmp(lbl31)
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl31)
		ctx.EmitJmp(lbl13)
		ps800 := scm.PhiState{General: true}
		ps800.OverlayValues = make([]scm.JITValueDesc, 798)
		ps800.OverlayValues[5] = d5
		ps800.OverlayValues[6] = d6
		ps800.OverlayValues[7] = d7
		ps800.OverlayValues[8] = d8
		ps800.OverlayValues[9] = d9
		ps800.OverlayValues[10] = d10
		ps800.OverlayValues[11] = d11
		ps800.OverlayValues[12] = d12
		ps800.OverlayValues[13] = d13
		ps800.OverlayValues[14] = d14
		ps800.OverlayValues[15] = d15
		ps800.OverlayValues[16] = d16
		ps800.OverlayValues[17] = d17
		ps800.OverlayValues[18] = d18
		ps800.OverlayValues[19] = d19
		ps800.OverlayValues[20] = d20
		ps800.OverlayValues[21] = d21
		ps800.OverlayValues[23] = d23
		ps800.OverlayValues[24] = d24
		ps800.OverlayValues[25] = d25
		ps800.OverlayValues[26] = d26
		ps800.OverlayValues[27] = d27
		ps800.OverlayValues[28] = d28
		ps800.OverlayValues[29] = d29
		ps800.OverlayValues[30] = d30
		ps800.OverlayValues[31] = d31
		ps800.OverlayValues[32] = d32
		ps800.OverlayValues[33] = d33
		ps800.OverlayValues[34] = d34
		ps800.OverlayValues[35] = d35
		ps800.OverlayValues[36] = d36
		ps800.OverlayValues[37] = d37
		ps800.OverlayValues[38] = d38
		ps800.OverlayValues[39] = d39
		ps800.OverlayValues[40] = d40
		ps800.OverlayValues[41] = d41
		ps800.OverlayValues[42] = d42
		ps800.OverlayValues[43] = d43
		ps800.OverlayValues[44] = d44
		ps800.OverlayValues[45] = d45
		ps800.OverlayValues[46] = d46
		ps800.OverlayValues[47] = d47
		ps800.OverlayValues[48] = d48
		ps800.OverlayValues[49] = d49
		ps800.OverlayValues[50] = d50
		ps800.OverlayValues[51] = d51
		ps800.OverlayValues[52] = d52
		ps800.OverlayValues[53] = d53
		ps800.OverlayValues[54] = d54
		ps800.OverlayValues[55] = d55
		ps800.OverlayValues[56] = d56
		ps800.OverlayValues[59] = d59
		ps800.OverlayValues[60] = d60
		ps800.OverlayValues[61] = d61
		ps800.OverlayValues[119] = d119
		ps800.OverlayValues[120] = d120
		ps800.OverlayValues[121] = d121
		ps800.OverlayValues[122] = d122
		ps800.OverlayValues[123] = d123
		ps800.OverlayValues[124] = d124
		ps800.OverlayValues[125] = d125
		ps800.OverlayValues[126] = d126
		ps800.OverlayValues[127] = d127
		ps800.OverlayValues[128] = d128
		ps800.OverlayValues[129] = d129
		ps800.OverlayValues[130] = d130
		ps800.OverlayValues[131] = d131
		ps800.OverlayValues[132] = d132
		ps800.OverlayValues[133] = d133
		ps800.OverlayValues[134] = d134
		ps800.OverlayValues[135] = d135
		ps800.OverlayValues[136] = d136
		ps800.OverlayValues[137] = d137
		ps800.OverlayValues[138] = d138
		ps800.OverlayValues[139] = d139
		ps800.OverlayValues[140] = d140
		ps800.OverlayValues[141] = d141
		ps800.OverlayValues[142] = d142
		ps800.OverlayValues[143] = d143
		ps800.OverlayValues[144] = d144
		ps800.OverlayValues[145] = d145
		ps800.OverlayValues[146] = d146
		ps800.OverlayValues[147] = d147
		ps800.OverlayValues[150] = d150
		ps800.OverlayValues[238] = d238
		ps800.OverlayValues[239] = d239
		ps800.OverlayValues[240] = d240
		ps800.OverlayValues[241] = d241
		ps800.OverlayValues[243] = d243
		ps800.OverlayValues[244] = d244
		ps800.OverlayValues[245] = d245
		ps800.OverlayValues[246] = d246
		ps800.OverlayValues[247] = d247
		ps800.OverlayValues[248] = d248
		ps800.OverlayValues[249] = d249
		ps800.OverlayValues[250] = d250
		ps800.OverlayValues[252] = d252
		ps800.OverlayValues[254] = d254
		ps800.OverlayValues[255] = d255
		ps800.OverlayValues[256] = d256
		ps800.OverlayValues[257] = d257
		ps800.OverlayValues[258] = d258
		ps800.OverlayValues[261] = d261
		ps800.OverlayValues[366] = d366
		ps800.OverlayValues[367] = d367
		ps800.OverlayValues[368] = d368
		ps800.OverlayValues[369] = d369
		ps800.OverlayValues[370] = d370
		ps800.OverlayValues[372] = d372
		ps800.OverlayValues[373] = d373
		ps800.OverlayValues[374] = d374
		ps800.OverlayValues[375] = d375
		ps800.OverlayValues[376] = d376
		ps800.OverlayValues[377] = d377
		ps800.OverlayValues[378] = d378
		ps800.OverlayValues[379] = d379
		ps800.OverlayValues[380] = d380
		ps800.OverlayValues[381] = d381
		ps800.OverlayValues[382] = d382
		ps800.OverlayValues[383] = d383
		ps800.OverlayValues[384] = d384
		ps800.OverlayValues[385] = d385
		ps800.OverlayValues[386] = d386
		ps800.OverlayValues[387] = d387
		ps800.OverlayValues[388] = d388
		ps800.OverlayValues[389] = d389
		ps800.OverlayValues[390] = d390
		ps800.OverlayValues[391] = d391
		ps800.OverlayValues[392] = d392
		ps800.OverlayValues[393] = d393
		ps800.OverlayValues[394] = d394
		ps800.OverlayValues[395] = d395
		ps800.OverlayValues[396] = d396
		ps800.OverlayValues[397] = d397
		ps800.OverlayValues[398] = d398
		ps800.OverlayValues[399] = d399
		ps800.OverlayValues[400] = d400
		ps800.OverlayValues[543] = d543
		ps800.OverlayValues[544] = d544
		ps800.OverlayValues[545] = d545
		ps800.OverlayValues[547] = d547
		ps800.OverlayValues[548] = d548
		ps800.OverlayValues[549] = d549
		ps800.OverlayValues[550] = d550
		ps800.OverlayValues[551] = d551
		ps800.OverlayValues[552] = d552
		ps800.OverlayValues[553] = d553
		ps800.OverlayValues[555] = d555
		ps800.OverlayValues[557] = d557
		ps800.OverlayValues[558] = d558
		ps800.OverlayValues[559] = d559
		ps800.OverlayValues[560] = d560
		ps800.OverlayValues[563] = d563
		ps800.OverlayValues[718] = d718
		ps800.OverlayValues[719] = d719
		ps800.OverlayValues[720] = d720
		ps800.OverlayValues[721] = d721
		ps800.OverlayValues[723] = d723
		ps800.OverlayValues[724] = d724
		ps800.OverlayValues[725] = d725
		ps800.OverlayValues[726] = d726
		ps800.OverlayValues[727] = d727
		ps800.OverlayValues[728] = d728
		ps800.OverlayValues[729] = d729
		ps800.OverlayValues[730] = d730
		ps800.OverlayValues[731] = d731
		ps800.OverlayValues[732] = d732
		ps800.OverlayValues[734] = d734
		ps800.OverlayValues[735] = d735
		ps800.OverlayValues[736] = d736
		ps800.OverlayValues[737] = d737
		ps800.OverlayValues[738] = d738
		ps800.OverlayValues[739] = d739
		ps800.OverlayValues[740] = d740
		ps800.OverlayValues[741] = d741
		ps800.OverlayValues[742] = d742
		ps800.OverlayValues[743] = d743
		ps800.OverlayValues[744] = d744
		ps800.OverlayValues[745] = d745
		ps800.OverlayValues[746] = d746
		ps800.OverlayValues[747] = d747
		ps800.OverlayValues[748] = d748
		ps800.OverlayValues[749] = d749
		ps800.OverlayValues[750] = d750
		ps800.OverlayValues[751] = d751
		ps800.OverlayValues[752] = d752
		ps800.OverlayValues[753] = d753
		ps800.OverlayValues[754] = d754
		ps800.OverlayValues[755] = d755
		ps800.OverlayValues[756] = d756
		ps800.OverlayValues[757] = d757
		ps800.OverlayValues[758] = d758
		ps800.OverlayValues[759] = d759
		ps800.OverlayValues[760] = d760
		ps800.OverlayValues[761] = d761
		ps800.OverlayValues[762] = d762
		ps800.OverlayValues[763] = d763
		ps800.OverlayValues[764] = d764
		ps800.OverlayValues[765] = d765
		ps800.OverlayValues[766] = d766
		ps800.OverlayValues[767] = d767
		ps800.OverlayValues[768] = d768
		ps800.OverlayValues[769] = d769
		ps800.OverlayValues[770] = d770
		ps800.OverlayValues[771] = d771
		ps800.OverlayValues[772] = d772
		ps800.OverlayValues[773] = d773
		ps800.OverlayValues[774] = d774
		ps800.OverlayValues[775] = d775
		ps800.OverlayValues[776] = d776
		ps800.OverlayValues[777] = d777
		ps800.OverlayValues[778] = d778
		ps800.OverlayValues[779] = d779
		ps800.OverlayValues[780] = d780
		ps800.OverlayValues[781] = d781
		ps800.OverlayValues[782] = d782
		ps800.OverlayValues[783] = d783
		ps800.OverlayValues[784] = d784
		ps800.OverlayValues[785] = d785
		ps800.OverlayValues[786] = d786
		ps800.OverlayValues[787] = d787
		ps800.OverlayValues[788] = d788
		ps800.OverlayValues[789] = d789
		ps800.OverlayValues[790] = d790
		ps800.OverlayValues[791] = d791
		ps800.OverlayValues[792] = d792
		ps800.OverlayValues[793] = d793
		ps800.OverlayValues[794] = d794
		ps800.OverlayValues[795] = d795
		ps800.OverlayValues[796] = d796
		ps800.OverlayValues[797] = d797
		ps801 := scm.PhiState{General: true}
		ps801.OverlayValues = make([]scm.JITValueDesc, 798)
		ps801.OverlayValues[5] = d5
		ps801.OverlayValues[6] = d6
		ps801.OverlayValues[7] = d7
		ps801.OverlayValues[8] = d8
		ps801.OverlayValues[9] = d9
		ps801.OverlayValues[10] = d10
		ps801.OverlayValues[11] = d11
		ps801.OverlayValues[12] = d12
		ps801.OverlayValues[13] = d13
		ps801.OverlayValues[14] = d14
		ps801.OverlayValues[15] = d15
		ps801.OverlayValues[16] = d16
		ps801.OverlayValues[17] = d17
		ps801.OverlayValues[18] = d18
		ps801.OverlayValues[19] = d19
		ps801.OverlayValues[20] = d20
		ps801.OverlayValues[21] = d21
		ps801.OverlayValues[23] = d23
		ps801.OverlayValues[24] = d24
		ps801.OverlayValues[25] = d25
		ps801.OverlayValues[26] = d26
		ps801.OverlayValues[27] = d27
		ps801.OverlayValues[28] = d28
		ps801.OverlayValues[29] = d29
		ps801.OverlayValues[30] = d30
		ps801.OverlayValues[31] = d31
		ps801.OverlayValues[32] = d32
		ps801.OverlayValues[33] = d33
		ps801.OverlayValues[34] = d34
		ps801.OverlayValues[35] = d35
		ps801.OverlayValues[36] = d36
		ps801.OverlayValues[37] = d37
		ps801.OverlayValues[38] = d38
		ps801.OverlayValues[39] = d39
		ps801.OverlayValues[40] = d40
		ps801.OverlayValues[41] = d41
		ps801.OverlayValues[42] = d42
		ps801.OverlayValues[43] = d43
		ps801.OverlayValues[44] = d44
		ps801.OverlayValues[45] = d45
		ps801.OverlayValues[46] = d46
		ps801.OverlayValues[47] = d47
		ps801.OverlayValues[48] = d48
		ps801.OverlayValues[49] = d49
		ps801.OverlayValues[50] = d50
		ps801.OverlayValues[51] = d51
		ps801.OverlayValues[52] = d52
		ps801.OverlayValues[53] = d53
		ps801.OverlayValues[54] = d54
		ps801.OverlayValues[55] = d55
		ps801.OverlayValues[56] = d56
		ps801.OverlayValues[59] = d59
		ps801.OverlayValues[60] = d60
		ps801.OverlayValues[61] = d61
		ps801.OverlayValues[119] = d119
		ps801.OverlayValues[120] = d120
		ps801.OverlayValues[121] = d121
		ps801.OverlayValues[122] = d122
		ps801.OverlayValues[123] = d123
		ps801.OverlayValues[124] = d124
		ps801.OverlayValues[125] = d125
		ps801.OverlayValues[126] = d126
		ps801.OverlayValues[127] = d127
		ps801.OverlayValues[128] = d128
		ps801.OverlayValues[129] = d129
		ps801.OverlayValues[130] = d130
		ps801.OverlayValues[131] = d131
		ps801.OverlayValues[132] = d132
		ps801.OverlayValues[133] = d133
		ps801.OverlayValues[134] = d134
		ps801.OverlayValues[135] = d135
		ps801.OverlayValues[136] = d136
		ps801.OverlayValues[137] = d137
		ps801.OverlayValues[138] = d138
		ps801.OverlayValues[139] = d139
		ps801.OverlayValues[140] = d140
		ps801.OverlayValues[141] = d141
		ps801.OverlayValues[142] = d142
		ps801.OverlayValues[143] = d143
		ps801.OverlayValues[144] = d144
		ps801.OverlayValues[145] = d145
		ps801.OverlayValues[146] = d146
		ps801.OverlayValues[147] = d147
		ps801.OverlayValues[150] = d150
		ps801.OverlayValues[238] = d238
		ps801.OverlayValues[239] = d239
		ps801.OverlayValues[240] = d240
		ps801.OverlayValues[241] = d241
		ps801.OverlayValues[243] = d243
		ps801.OverlayValues[244] = d244
		ps801.OverlayValues[245] = d245
		ps801.OverlayValues[246] = d246
		ps801.OverlayValues[247] = d247
		ps801.OverlayValues[248] = d248
		ps801.OverlayValues[249] = d249
		ps801.OverlayValues[250] = d250
		ps801.OverlayValues[252] = d252
		ps801.OverlayValues[254] = d254
		ps801.OverlayValues[255] = d255
		ps801.OverlayValues[256] = d256
		ps801.OverlayValues[257] = d257
		ps801.OverlayValues[258] = d258
		ps801.OverlayValues[261] = d261
		ps801.OverlayValues[366] = d366
		ps801.OverlayValues[367] = d367
		ps801.OverlayValues[368] = d368
		ps801.OverlayValues[369] = d369
		ps801.OverlayValues[370] = d370
		ps801.OverlayValues[372] = d372
		ps801.OverlayValues[373] = d373
		ps801.OverlayValues[374] = d374
		ps801.OverlayValues[375] = d375
		ps801.OverlayValues[376] = d376
		ps801.OverlayValues[377] = d377
		ps801.OverlayValues[378] = d378
		ps801.OverlayValues[379] = d379
		ps801.OverlayValues[380] = d380
		ps801.OverlayValues[381] = d381
		ps801.OverlayValues[382] = d382
		ps801.OverlayValues[383] = d383
		ps801.OverlayValues[384] = d384
		ps801.OverlayValues[385] = d385
		ps801.OverlayValues[386] = d386
		ps801.OverlayValues[387] = d387
		ps801.OverlayValues[388] = d388
		ps801.OverlayValues[389] = d389
		ps801.OverlayValues[390] = d390
		ps801.OverlayValues[391] = d391
		ps801.OverlayValues[392] = d392
		ps801.OverlayValues[393] = d393
		ps801.OverlayValues[394] = d394
		ps801.OverlayValues[395] = d395
		ps801.OverlayValues[396] = d396
		ps801.OverlayValues[397] = d397
		ps801.OverlayValues[398] = d398
		ps801.OverlayValues[399] = d399
		ps801.OverlayValues[400] = d400
		ps801.OverlayValues[543] = d543
		ps801.OverlayValues[544] = d544
		ps801.OverlayValues[545] = d545
		ps801.OverlayValues[547] = d547
		ps801.OverlayValues[548] = d548
		ps801.OverlayValues[549] = d549
		ps801.OverlayValues[550] = d550
		ps801.OverlayValues[551] = d551
		ps801.OverlayValues[552] = d552
		ps801.OverlayValues[553] = d553
		ps801.OverlayValues[555] = d555
		ps801.OverlayValues[557] = d557
		ps801.OverlayValues[558] = d558
		ps801.OverlayValues[559] = d559
		ps801.OverlayValues[560] = d560
		ps801.OverlayValues[563] = d563
		ps801.OverlayValues[718] = d718
		ps801.OverlayValues[719] = d719
		ps801.OverlayValues[720] = d720
		ps801.OverlayValues[721] = d721
		ps801.OverlayValues[723] = d723
		ps801.OverlayValues[724] = d724
		ps801.OverlayValues[725] = d725
		ps801.OverlayValues[726] = d726
		ps801.OverlayValues[727] = d727
		ps801.OverlayValues[728] = d728
		ps801.OverlayValues[729] = d729
		ps801.OverlayValues[730] = d730
		ps801.OverlayValues[731] = d731
		ps801.OverlayValues[732] = d732
		ps801.OverlayValues[734] = d734
		ps801.OverlayValues[735] = d735
		ps801.OverlayValues[736] = d736
		ps801.OverlayValues[737] = d737
		ps801.OverlayValues[738] = d738
		ps801.OverlayValues[739] = d739
		ps801.OverlayValues[740] = d740
		ps801.OverlayValues[741] = d741
		ps801.OverlayValues[742] = d742
		ps801.OverlayValues[743] = d743
		ps801.OverlayValues[744] = d744
		ps801.OverlayValues[745] = d745
		ps801.OverlayValues[746] = d746
		ps801.OverlayValues[747] = d747
		ps801.OverlayValues[748] = d748
		ps801.OverlayValues[749] = d749
		ps801.OverlayValues[750] = d750
		ps801.OverlayValues[751] = d751
		ps801.OverlayValues[752] = d752
		ps801.OverlayValues[753] = d753
		ps801.OverlayValues[754] = d754
		ps801.OverlayValues[755] = d755
		ps801.OverlayValues[756] = d756
		ps801.OverlayValues[757] = d757
		ps801.OverlayValues[758] = d758
		ps801.OverlayValues[759] = d759
		ps801.OverlayValues[760] = d760
		ps801.OverlayValues[761] = d761
		ps801.OverlayValues[762] = d762
		ps801.OverlayValues[763] = d763
		ps801.OverlayValues[764] = d764
		ps801.OverlayValues[765] = d765
		ps801.OverlayValues[766] = d766
		ps801.OverlayValues[767] = d767
		ps801.OverlayValues[768] = d768
		ps801.OverlayValues[769] = d769
		ps801.OverlayValues[770] = d770
		ps801.OverlayValues[771] = d771
		ps801.OverlayValues[772] = d772
		ps801.OverlayValues[773] = d773
		ps801.OverlayValues[774] = d774
		ps801.OverlayValues[775] = d775
		ps801.OverlayValues[776] = d776
		ps801.OverlayValues[777] = d777
		ps801.OverlayValues[778] = d778
		ps801.OverlayValues[779] = d779
		ps801.OverlayValues[780] = d780
		ps801.OverlayValues[781] = d781
		ps801.OverlayValues[782] = d782
		ps801.OverlayValues[783] = d783
		ps801.OverlayValues[784] = d784
		ps801.OverlayValues[785] = d785
		ps801.OverlayValues[786] = d786
		ps801.OverlayValues[787] = d787
		ps801.OverlayValues[788] = d788
		ps801.OverlayValues[789] = d789
		ps801.OverlayValues[790] = d790
		ps801.OverlayValues[791] = d791
		ps801.OverlayValues[792] = d792
		ps801.OverlayValues[793] = d793
		ps801.OverlayValues[794] = d794
		ps801.OverlayValues[795] = d795
		ps801.OverlayValues[796] = d796
		ps801.OverlayValues[797] = d797
		snap802 := d5
		snap803 := d6
		snap804 := d7
		snap805 := d8
		snap806 := d9
		snap807 := d10
		snap808 := d11
		snap809 := d12
		snap810 := d13
		snap811 := d14
		snap812 := d15
		snap813 := d16
		snap814 := d17
		snap815 := d18
		snap816 := d19
		snap817 := d20
		snap818 := d21
		snap819 := d23
		snap820 := d24
		snap821 := d25
		snap822 := d26
		snap823 := d27
		snap824 := d28
		snap825 := d29
		snap826 := d30
		snap827 := d31
		snap828 := d32
		snap829 := d33
		snap830 := d34
		snap831 := d35
		snap832 := d36
		snap833 := d37
		snap834 := d38
		snap835 := d39
		snap836 := d40
		snap837 := d41
		snap838 := d42
		snap839 := d43
		snap840 := d44
		snap841 := d45
		snap842 := d46
		snap843 := d47
		snap844 := d48
		snap845 := d49
		snap846 := d50
		snap847 := d51
		snap848 := d52
		snap849 := d53
		snap850 := d54
		snap851 := d55
		snap852 := d56
		snap853 := d59
		snap854 := d60
		snap855 := d61
		snap856 := d119
		snap857 := d120
		snap858 := d121
		snap859 := d122
		snap860 := d123
		snap861 := d124
		snap862 := d125
		snap863 := d126
		snap864 := d127
		snap865 := d128
		snap866 := d129
		snap867 := d130
		snap868 := d131
		snap869 := d132
		snap870 := d133
		snap871 := d134
		snap872 := d135
		snap873 := d136
		snap874 := d137
		snap875 := d138
		snap876 := d139
		snap877 := d140
		snap878 := d141
		snap879 := d142
		snap880 := d143
		snap881 := d144
		snap882 := d145
		snap883 := d146
		snap884 := d147
		snap885 := d150
		snap886 := d238
		snap887 := d239
		snap888 := d240
		snap889 := d241
		snap890 := d243
		snap891 := d244
		snap892 := d245
		snap893 := d246
		snap894 := d247
		snap895 := d248
		snap896 := d249
		snap897 := d250
		snap898 := d252
		snap899 := d254
		snap900 := d255
		snap901 := d256
		snap902 := d257
		snap903 := d258
		snap904 := d261
		snap905 := d366
		snap906 := d367
		snap907 := d368
		snap908 := d369
		snap909 := d370
		snap910 := d372
		snap911 := d373
		snap912 := d374
		snap913 := d375
		snap914 := d376
		snap915 := d377
		snap916 := d378
		snap917 := d379
		snap918 := d380
		snap919 := d381
		snap920 := d382
		snap921 := d383
		snap922 := d384
		snap923 := d385
		snap924 := d386
		snap925 := d387
		snap926 := d388
		snap927 := d389
		snap928 := d390
		snap929 := d391
		snap930 := d392
		snap931 := d393
		snap932 := d394
		snap933 := d395
		snap934 := d396
		snap935 := d397
		snap936 := d398
		snap937 := d399
		snap938 := d400
		snap939 := d543
		snap940 := d544
		snap941 := d545
		snap942 := d547
		snap943 := d548
		snap944 := d549
		snap945 := d550
		snap946 := d551
		snap947 := d552
		snap948 := d553
		snap949 := d555
		snap950 := d557
		snap951 := d558
		snap952 := d559
		snap953 := d560
		snap954 := d563
		snap955 := d718
		snap956 := d719
		snap957 := d720
		snap958 := d721
		snap959 := d723
		snap960 := d724
		snap961 := d725
		snap962 := d726
		snap963 := d727
		snap964 := d728
		snap965 := d729
		snap966 := d730
		snap967 := d731
		snap968 := d732
		snap969 := d734
		snap970 := d735
		snap971 := d736
		snap972 := d737
		snap973 := d738
		snap974 := d739
		snap975 := d740
		snap976 := d741
		snap977 := d742
		snap978 := d743
		snap979 := d744
		snap980 := d745
		snap981 := d746
		snap982 := d747
		snap983 := d748
		snap984 := d749
		snap985 := d750
		snap986 := d751
		snap987 := d752
		snap988 := d753
		snap989 := d754
		snap990 := d755
		snap991 := d756
		snap992 := d757
		snap993 := d758
		snap994 := d759
		snap995 := d760
		snap996 := d761
		snap997 := d762
		snap998 := d763
		snap999 := d764
		snap1000 := d765
		snap1001 := d766
		snap1002 := d767
		snap1003 := d768
		snap1004 := d769
		snap1005 := d770
		snap1006 := d771
		snap1007 := d772
		snap1008 := d773
		snap1009 := d774
		snap1010 := d775
		snap1011 := d776
		snap1012 := d777
		snap1013 := d778
		snap1014 := d779
		snap1015 := d780
		snap1016 := d781
		snap1017 := d782
		snap1018 := d783
		snap1019 := d784
		snap1020 := d785
		snap1021 := d786
		snap1022 := d787
		snap1023 := d788
		snap1024 := d789
		snap1025 := d790
		snap1026 := d791
		snap1027 := d792
		snap1028 := d793
		snap1029 := d794
		snap1030 := d795
		snap1031 := d796
		snap1032 := d797
		alloc1033 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps801)
		}
		ctx.RestoreAllocState(alloc1033)
		d5 = snap802
		d6 = snap803
		d7 = snap804
		d8 = snap805
		d9 = snap806
		d10 = snap807
		d11 = snap808
		d12 = snap809
		d13 = snap810
		d14 = snap811
		d15 = snap812
		d16 = snap813
		d17 = snap814
		d18 = snap815
		d19 = snap816
		d20 = snap817
		d21 = snap818
		d23 = snap819
		d24 = snap820
		d25 = snap821
		d26 = snap822
		d27 = snap823
		d28 = snap824
		d29 = snap825
		d30 = snap826
		d31 = snap827
		d32 = snap828
		d33 = snap829
		d34 = snap830
		d35 = snap831
		d36 = snap832
		d37 = snap833
		d38 = snap834
		d39 = snap835
		d40 = snap836
		d41 = snap837
		d42 = snap838
		d43 = snap839
		d44 = snap840
		d45 = snap841
		d46 = snap842
		d47 = snap843
		d48 = snap844
		d49 = snap845
		d50 = snap846
		d51 = snap847
		d52 = snap848
		d53 = snap849
		d54 = snap850
		d55 = snap851
		d56 = snap852
		d59 = snap853
		d60 = snap854
		d61 = snap855
		d119 = snap856
		d120 = snap857
		d121 = snap858
		d122 = snap859
		d123 = snap860
		d124 = snap861
		d125 = snap862
		d126 = snap863
		d127 = snap864
		d128 = snap865
		d129 = snap866
		d130 = snap867
		d131 = snap868
		d132 = snap869
		d133 = snap870
		d134 = snap871
		d135 = snap872
		d136 = snap873
		d137 = snap874
		d138 = snap875
		d139 = snap876
		d140 = snap877
		d141 = snap878
		d142 = snap879
		d143 = snap880
		d144 = snap881
		d145 = snap882
		d146 = snap883
		d147 = snap884
		d150 = snap885
		d238 = snap886
		d239 = snap887
		d240 = snap888
		d241 = snap889
		d243 = snap890
		d244 = snap891
		d245 = snap892
		d246 = snap893
		d247 = snap894
		d248 = snap895
		d249 = snap896
		d250 = snap897
		d252 = snap898
		d254 = snap899
		d255 = snap900
		d256 = snap901
		d257 = snap902
		d258 = snap903
		d261 = snap904
		d366 = snap905
		d367 = snap906
		d368 = snap907
		d369 = snap908
		d370 = snap909
		d372 = snap910
		d373 = snap911
		d374 = snap912
		d375 = snap913
		d376 = snap914
		d377 = snap915
		d378 = snap916
		d379 = snap917
		d380 = snap918
		d381 = snap919
		d382 = snap920
		d383 = snap921
		d384 = snap922
		d385 = snap923
		d386 = snap924
		d387 = snap925
		d388 = snap926
		d389 = snap927
		d390 = snap928
		d391 = snap929
		d392 = snap930
		d393 = snap931
		d394 = snap932
		d395 = snap933
		d396 = snap934
		d397 = snap935
		d398 = snap936
		d399 = snap937
		d400 = snap938
		d543 = snap939
		d544 = snap940
		d545 = snap941
		d547 = snap942
		d548 = snap943
		d549 = snap944
		d550 = snap945
		d551 = snap946
		d552 = snap947
		d553 = snap948
		d555 = snap949
		d557 = snap950
		d558 = snap951
		d559 = snap952
		d560 = snap953
		d563 = snap954
		d718 = snap955
		d719 = snap956
		d720 = snap957
		d721 = snap958
		d723 = snap959
		d724 = snap960
		d725 = snap961
		d726 = snap962
		d727 = snap963
		d728 = snap964
		d729 = snap965
		d730 = snap966
		d731 = snap967
		d732 = snap968
		d734 = snap969
		d735 = snap970
		d736 = snap971
		d737 = snap972
		d738 = snap973
		d739 = snap974
		d740 = snap975
		d741 = snap976
		d742 = snap977
		d743 = snap978
		d744 = snap979
		d745 = snap980
		d746 = snap981
		d747 = snap982
		d748 = snap983
		d749 = snap984
		d750 = snap985
		d751 = snap986
		d752 = snap987
		d753 = snap988
		d754 = snap989
		d755 = snap990
		d756 = snap991
		d757 = snap992
		d758 = snap993
		d759 = snap994
		d760 = snap995
		d761 = snap996
		d762 = snap997
		d763 = snap998
		d764 = snap999
		d765 = snap1000
		d766 = snap1001
		d767 = snap1002
		d768 = snap1003
		d769 = snap1004
		d770 = snap1005
		d771 = snap1006
		d772 = snap1007
		d773 = snap1008
		d774 = snap1009
		d775 = snap1010
		d776 = snap1011
		d777 = snap1012
		d778 = snap1013
		d779 = snap1014
		d780 = snap1015
		d781 = snap1016
		d782 = snap1017
		d783 = snap1018
		d784 = snap1019
		d785 = snap1020
		d786 = snap1021
		d787 = snap1022
		d788 = snap1023
		d789 = snap1024
		d790 = snap1025
		d791 = snap1026
		d792 = snap1027
		d793 = snap1028
		d794 = snap1029
		d795 = snap1030
		d796 = snap1031
		d797 = snap1032
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps800)
		}
		return result
		ctx.FreeDesc(&d796)
		return result
	}
	ps1034 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1034)
	ctx.MarkLabel(lbl0)
	d1035 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
	ctx.BindReg(r3, &d1035)
	ctx.BindReg(r4, &d1035)
	ctx.EmitMovPairToResult(&d1035, &result)
	ctx.FreeReg(r3)
	ctx.FreeReg(r4)
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

func (s *StorageSeq) GetCachedReader() ColumnReader { return s }

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
	value = int64(s.start.GetValueUInt(min)) + s.start.offset
	if s.start.hasNull && value == int64(s.start.null) {
		return scm.NewNil()
	}
	stride = int64(s.stride.GetValueUInt(min)) + s.stride.offset
	recid := int64(s.recordId.GetValueUInt(min)) + s.recordId.offset
	return scm.NewFloat(float64(value + int64(int64(i)-recid)*stride))

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
			target[idx] = scm.NewFloat(float64(curVal))
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
		for _, recid := range recids {
			i := int64(recid)
			for i >= nextRecordId {
				seg++
				segRecordId, isNil, segStart, segStride = s.segmentAt(seg)
				nextRecordId = s.segmentEnd(seg)
			}
			if isNil {
				target[idx] = scm.NewNil()
			} else {
				target[idx] = scm.NewFloat(float64(segStart + (i-segRecordId)*segStride))
			}
			idx += stride
		}
		return
	}

	for _, recid := range recids {
		seg := s.findSegment(recid)
		segRecordId, isNil, segStart, segStride := s.segmentAt(seg)
		if isNil {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewFloat(float64(segStart + (int64(recid)-segRecordId)*segStride))
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
		if s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.scan(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if i != 0 && v == s.lastValue.Load()+s.lastStride {
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
		if s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.build(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if i != 0 && v == s.lastValue.Load()+s.lastStride {
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
