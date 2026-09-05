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

func (s *StorageSeq) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	var d10 scm.JITValueDesc
	_ = d10
	var d11 scm.JITValueDesc
	_ = d11
	var d12 scm.JITValueDesc
	_ = d12
	var d13 scm.JITValueDesc
	_ = d13
	var d14 scm.JITValueDesc
	_ = d14
	var d15 scm.JITValueDesc
	_ = d15
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
	var d53 scm.JITValueDesc
	_ = d53
	var d54 scm.JITValueDesc
	_ = d54
	var d55 scm.JITValueDesc
	_ = d55
	var d111 scm.JITValueDesc
	_ = d111
	var d112 scm.JITValueDesc
	_ = d112
	var d113 scm.JITValueDesc
	_ = d113
	var d114 scm.JITValueDesc
	_ = d114
	var d115 scm.JITValueDesc
	_ = d115
	var d116 scm.JITValueDesc
	_ = d116
	var d117 scm.JITValueDesc
	_ = d117
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
	var d143 scm.JITValueDesc
	_ = d143
	var d230 scm.JITValueDesc
	_ = d230
	var d231 scm.JITValueDesc
	_ = d231
	var d232 scm.JITValueDesc
	_ = d232
	var d233 scm.JITValueDesc
	_ = d233
	var d235 scm.JITValueDesc
	_ = d235
	var d236 scm.JITValueDesc
	_ = d236
	var d237 scm.JITValueDesc
	_ = d237
	var d238 scm.JITValueDesc
	_ = d238
	var d239 scm.JITValueDesc
	_ = d239
	var d240 scm.JITValueDesc
	_ = d240
	var d241 scm.JITValueDesc
	_ = d241
	var d242 scm.JITValueDesc
	_ = d242
	var d244 scm.JITValueDesc
	_ = d244
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
	var d253 scm.JITValueDesc
	_ = d253
	var d357 scm.JITValueDesc
	_ = d357
	var d358 scm.JITValueDesc
	_ = d358
	var d359 scm.JITValueDesc
	_ = d359
	var d360 scm.JITValueDesc
	_ = d360
	var d361 scm.JITValueDesc
	_ = d361
	var d363 scm.JITValueDesc
	_ = d363
	var d364 scm.JITValueDesc
	_ = d364
	var d365 scm.JITValueDesc
	_ = d365
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
	var d371 scm.JITValueDesc
	_ = d371
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
	var d537 scm.JITValueDesc
	_ = d537
	var d538 scm.JITValueDesc
	_ = d538
	var d539 scm.JITValueDesc
	_ = d539
	var d541 scm.JITValueDesc
	_ = d541
	var d542 scm.JITValueDesc
	_ = d542
	var d543 scm.JITValueDesc
	_ = d543
	var d544 scm.JITValueDesc
	_ = d544
	var d545 scm.JITValueDesc
	_ = d545
	var d546 scm.JITValueDesc
	_ = d546
	var d547 scm.JITValueDesc
	_ = d547
	var d549 scm.JITValueDesc
	_ = d549
	var d551 scm.JITValueDesc
	_ = d551
	var d552 scm.JITValueDesc
	_ = d552
	var d553 scm.JITValueDesc
	_ = d553
	var d554 scm.JITValueDesc
	_ = d554
	var d557 scm.JITValueDesc
	_ = d557
	var d713 scm.JITValueDesc
	_ = d713
	var d714 scm.JITValueDesc
	_ = d714
	var d715 scm.JITValueDesc
	_ = d715
	var d716 scm.JITValueDesc
	_ = d716
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
	phiBase0 := ctx.AllocStack(int32(144))
	d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
	_ = d1
	d2 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
	_ = d2
	d3 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
	_ = d3
	d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
	_ = d4
	d5 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
	_ = d5
	d6 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
	_ = d6
	d7 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
	_ = d7
	d8 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
	_ = d8
	d9 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
	_ = d9
	var bbs [14]scm.BBDescriptor
	bbs[1].PhiBase = int32(phiBase0) + int32(0)
	bbs[1].PhiCount = uint16(3)
	bbs[2].PhiBase = int32(phiBase0) + int32(48)
	bbs[2].PhiCount = uint16(1)
	bbs[4].PhiBase = int32(phiBase0) + int32(64)
	bbs[4].PhiCount = uint16(3)
	bbs[8].PhiBase = int32(phiBase0) + int32(112)
	bbs[8].PhiCount = uint16(2)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		ctx.ReclaimUntrackedRegs()
		r2 := ctx.AllocReg()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
			ctx.EmitMovRegMem64(r2, fieldAddr)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			ctx.EmitMovRegMem(r2, thisptr.Reg, off)
		}
		d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
		ctx.BindReg(r2, &d10)
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d10)
		var d11 scm.JITValueDesc
		if d10.Loc == scm.LocImm {
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d10.Imm.Int()))))}
		} else {
			r3 := ctx.AllocReg()
			ctx.EmitMovRegReg(r3, d10.Reg)
			ctx.EmitShlRegImm8(r3, 32)
			ctx.EmitShrRegImm8(r3, 32)
			d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d11)
		}
		ctx.StabilizeDescForControlFlow(&d11)
		ctx.FreeDesc(&d10)
		var d12 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
			val := *(*uint32)(unsafe.Pointer(fieldAddr))
			d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
			r4 := ctx.AllocReg()
			ctx.EmitMovRegMemL(r4, thisptr.Reg, off)
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			ctx.BindReg(r4, &d12)
		}
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d12)
		var d13 scm.JITValueDesc
		if d12.Loc == scm.LocImm {
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d13)
		}
		if d13.Loc == scm.LocImm {
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: d13.Type, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d13.Reg, 32)
			ctx.EmitShrRegImm8(d13.Reg, 32)
		}
		if d13.Loc == scm.LocReg && d12.Loc == scm.LocReg && d13.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d13)
		ctx.EmitStoreToStack(d13, int32(bbs[1].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d13)
		ctx.FreeDesc(&d12)
		if ps.General {
			ctx.SyncDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d14 = d11
			if d14.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d14)
			d15 = d14
			if d15.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: d15.Type, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d15.Reg, 32)
				ctx.EmitShrRegImm8(d15.Reg, 32)
			}
			ctx.EmitStoreToStack(d15, int32(bbs[1].PhiBase)+int32(0))
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(16))
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
		}
		ps16 := scm.PhiState{General: ps.General}
		ps16.OverlayValues = make([]scm.JITValueDesc, 16)
		ps16.OverlayValues[1] = d1
		ps16.OverlayValues[2] = d2
		ps16.OverlayValues[3] = d3
		ps16.OverlayValues[4] = d4
		ps16.OverlayValues[5] = d5
		ps16.OverlayValues[6] = d6
		ps16.OverlayValues[7] = d7
		ps16.OverlayValues[8] = d8
		ps16.OverlayValues[9] = d9
		ps16.OverlayValues[10] = d10
		ps16.OverlayValues[11] = d11
		ps16.OverlayValues[12] = d12
		ps16.OverlayValues[13] = d13
		ps16.OverlayValues[14] = d14
		ps16.OverlayValues[15] = d15
		ps16.PhiValues = make([]scm.JITValueDesc, 3)
		d17 = d11
		ps16.PhiValues[0] = d17
		d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
		ps16.PhiValues[1] = d18
		if ps16.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps16)
		return result
	}
	bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d19 := ps.PhiValues[0]
				ctx.EnsureDesc(&d19)
				ctx.EmitStoreToStack(d19, int32(bbs[1].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d20 := ps.PhiValues[1]
				ctx.EnsureDesc(&d20)
				ctx.EmitStoreToStack(d20, int32(bbs[1].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d21 := ps.PhiValues[2]
				ctx.EnsureDesc(&d21)
				ctx.EmitStoreToStack(d21, int32(bbs[1].PhiBase)+int32(32))
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d1 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d2 = ps.PhiValues[1]
		}
		if !ps.General && len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
			d3 = ps.PhiValues[2]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d1)
		ctx.StabilizeDescForControlFlow(&d2)
		ctx.StabilizeDescForControlFlow(&d3)
		ctx.EnsureDesc(&d1)
		d22 = d1
		_ = d22
		ctx.StabilizeDescForControlFlow(&d22)
		ctx.StabilizeDescForControlFlow(&d1)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl15 := ctx.ReserveLabel()
		_ = lbl15
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl15)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d22)
		var d23 scm.JITValueDesc
		if d22.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d22.Imm.Int()))))}
		} else {
			r5 := ctx.AllocReg()
			ctx.EmitMovRegReg(r5, d22.Reg)
			ctx.EmitShlRegImm8(r5, 32)
			ctx.EmitShrRegImm8(r5, 32)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d23)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d24 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r6 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r6, thisptr.Reg, off)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
			ctx.BindReg(r6, &d24)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		var d25 scm.JITValueDesc
		if d24.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
		} else {
			r7 := ctx.AllocReg()
			ctx.EmitMovRegReg(r7, d24.Reg)
			ctx.EmitShlRegImm8(r7, 56)
			ctx.EmitShrRegImm8(r7, 56)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d25)
		}
		ctx.FreeDesc(&d24)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d23)
		ctx.EnsureDesc(&d25)
		ctx.EnsureDescsTogether(&d23, &d25)
		var d26 scm.JITValueDesc
		if d23.Loc == scm.LocImm && d25.Loc == scm.LocImm {
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() * d25.Imm.Int())}
		} else if d23.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d25.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d23.Imm.Int()))
			ctx.EmitImulInt64(scratch, d25.Reg)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d26)
		} else if d25.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d23.Reg)
			ctx.EmitMovRegReg(scratch, d23.Reg)
			if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d25.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d26)
		} else {
			r8 := ctx.AllocRegExcept(d23.Reg, d25.Reg)
			ctx.EmitMovRegReg(r8, d23.Reg)
			ctx.EmitImulInt64(r8, d25.Reg)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d26)
		}
		if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
			ctx.TransferReg(d23.Reg)
			d23.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d23)
		ctx.FreeDesc(&d25)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		var d27 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() / 64)}
		} else {
			r9 := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegReg(r9, d26.Reg)
			ctx.EmitShrRegImm8(r9, 6)
			d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			ctx.BindReg(r9, &d27)
		}
		if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
			ctx.TransferReg(d26.Reg)
			d26.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		var d28 scm.JITValueDesc
		if d26.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() % 64)}
		} else {
			r10 := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegReg(r10, d26.Reg)
			ctx.EmitAndRegImm32(r10, 63)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d28)
		}
		if d28.Loc == scm.LocReg && d26.Loc == scm.LocReg && d28.Reg == d26.Reg {
			ctx.TransferReg(d26.Reg)
			d26.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d26)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d29 scm.JITValueDesc
		r11 := ctx.AllocReg()
		r12 := ctx.AllocRegExcept(r11)
		r13 := ctx.AllocRegExcept(r11, r12)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r11, uint64(dataPtr))
			ctx.EmitMovRegImm64(r12, uint64(sliceLen))
			ctx.EmitMovRegImm64(r13, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r11, thisptr.Reg, off)
			ctx.EmitMovRegMem(r12, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r13, thisptr.Reg, off+16)
		}
		d29 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r11, Reg2: r12, Reg3: r13}
		ctx.BindReg(r11, &d29)
		ctx.BindReg(r12, &d29)
		ctx.BindReg(r13, &d29)
		ctx.BindReg(r11, &d29)
		ctx.BindReg(r12, &d29)
		ctx.BindReg(r13, &d29)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d27)
		ctx.ReclaimUntrackedRegs()
		d31 = ctx.EmitSliceElementAddress(&d29, &d27, 8)
		ctx.EnsureDesc(&d31)
		ctx.EmitMovRegMem(d31.Reg, d31.Reg, 0)
		d30 = d31
		d30.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d30)
		ctx.EnsureDesc(&d28)
		ctx.EnsureDescsTogether(&d30, &d28)
		var d32 scm.JITValueDesc
		if d30.Loc == scm.LocImm && d28.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) << uint64(d28.Imm.Int())))}
		} else if d28.Loc == scm.LocImm {
			r14 := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegReg(r14, d30.Reg)
			ctx.EmitShlRegImm8(r14, uint8(d28.Imm.Int()))
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d32)
		} else {
			{
				shiftSrc := d30.Reg
				r15 := ctx.AllocRegExcept(d30.Reg, d28.Reg)
				ctx.EmitMovRegReg(r15, d30.Reg)
				shiftSrc = r15
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d28.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d28.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d28.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d32)
			}
		}
		if d32.Loc == scm.LocReg && d30.Loc == scm.LocReg && d32.Reg == d30.Reg {
			ctx.TransferReg(d30.Reg)
			d30.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d27)
		ctx.EnsureDesc(&d27)
		var d33 scm.JITValueDesc
		if d27.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d27.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d27.Reg)
			ctx.EmitMovRegReg(scratch, d27.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d33)
		}
		if d33.Loc == scm.LocReg && d27.Loc == scm.LocReg && d33.Reg == d27.Reg {
			ctx.TransferReg(d27.Reg)
			d27.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d27)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		d35 = ctx.EmitSliceElementAddress(&d29, &d33, 8)
		ctx.EnsureDesc(&d35)
		ctx.EmitMovRegMem(d35.Reg, d35.Reg, 0)
		d34 = d35
		d34.Type = scm.TagInt
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d28)
		ctx.EnsureDescsTogether(&d36, &d28)
		var d37 scm.JITValueDesc
		if d36.Loc == scm.LocImm && d28.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() - d28.Imm.Int())}
		} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
			r16 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r16, d36.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d37)
		} else if d36.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
			ctx.EmitSubInt64(scratch, d28.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else if d28.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(scratch, d36.Reg)
			if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d28.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d28.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d37)
		} else {
			r17 := ctx.AllocRegExcept(d36.Reg, d28.Reg)
			ctx.EmitMovRegReg(r17, d36.Reg)
			ctx.EmitSubInt64(r17, d28.Reg)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d37)
		}
		if d37.Loc == scm.LocReg && d36.Loc == scm.LocReg && d37.Reg == d36.Reg {
			ctx.TransferReg(d36.Reg)
			d36.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d34)
		ctx.EnsureDesc(&d37)
		ctx.EnsureDescsTogether(&d34, &d37)
		var d38 scm.JITValueDesc
		if d34.Loc == scm.LocImm && d37.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d34.Imm.Int()) >> uint64(d37.Imm.Int())))}
		} else if d37.Loc == scm.LocImm {
			r18 := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegReg(r18, d34.Reg)
			ctx.EmitShrRegImm8(r18, uint8(d37.Imm.Int()))
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d38)
		} else {
			{
				shiftSrc := d34.Reg
				r19 := ctx.AllocRegExcept(d34.Reg, d37.Reg)
				ctx.EmitMovRegReg(r19, d34.Reg)
				shiftSrc = r19
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d37.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d37.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d37.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d38)
			}
		}
		if d38.Loc == scm.LocReg && d34.Loc == scm.LocReg && d38.Reg == d34.Reg {
			ctx.TransferReg(d34.Reg)
			d34.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d34)
		ctx.FreeDesc(&d37)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d32)
		ctx.EnsureDesc(&d38)
		var d39 scm.JITValueDesc
		if d32.Loc == scm.LocImm && d38.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() | d38.Imm.Int())}
		} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg}
			ctx.BindReg(d38.Reg, &d39)
		} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
			r20 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r20, d32.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d39)
		} else if d32.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d38.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
			ctx.EmitOrInt64(scratch, d38.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d39)
		} else if d38.Loc == scm.LocImm {
			r21 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r21, d32.Reg)
			if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r21, int32(d38.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
				ctx.EmitOrInt64(r21, scm.RegR11)
			}
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d39)
		} else {
			r22 := ctx.AllocRegExcept(d32.Reg, d38.Reg)
			ctx.EmitMovRegReg(r22, d32.Reg)
			ctx.EmitOrInt64(r22, d38.Reg)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d39)
		}
		if d39.Loc == scm.LocReg && d32.Loc == scm.LocReg && d39.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d32)
		ctx.FreeDesc(&d38)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d40 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r23 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r23, thisptr.Reg, off)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d40)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		ctx.EnsureDesc(&d40)
		var d41 scm.JITValueDesc
		if d40.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d40.Imm.Int()))))}
		} else {
			r24 := ctx.AllocReg()
			ctx.EmitMovRegReg(r24, d40.Reg)
			ctx.EmitShlRegImm8(r24, 56)
			ctx.EmitShrRegImm8(r24, 56)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d41)
		}
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d41)
		ctx.EnsureDescsTogether(&d42, &d41)
		var d43 scm.JITValueDesc
		if d42.Loc == scm.LocImm && d41.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() - d41.Imm.Int())}
		} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
			r25 := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegReg(r25, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d43)
		} else if d42.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d42.Imm.Int()))
			ctx.EmitSubInt64(scratch, d41.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d43)
		} else if d41.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegReg(scratch, d42.Reg)
			if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d41.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d43)
		} else {
			r26 := ctx.AllocRegExcept(d42.Reg, d41.Reg)
			ctx.EmitMovRegReg(r26, d42.Reg)
			ctx.EmitSubInt64(r26, d41.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d43)
		}
		if d43.Loc == scm.LocReg && d42.Loc == scm.LocReg && d43.Reg == d42.Reg {
			ctx.TransferReg(d42.Reg)
			d42.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d43)
		ctx.EnsureDescsTogether(&d39, &d43)
		var d44 scm.JITValueDesc
		if d39.Loc == scm.LocImm && d43.Loc == scm.LocImm {
			d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d39.Imm.Int()) >> uint64(d43.Imm.Int())))}
		} else if d43.Loc == scm.LocImm {
			r27 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r27, d39.Reg)
			ctx.EmitShrRegImm8(r27, uint8(d43.Imm.Int()))
			d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d44)
		} else {
			{
				shiftSrc := d39.Reg
				r28 := ctx.AllocRegExcept(d39.Reg, d43.Reg)
				ctx.EmitMovRegReg(r28, d39.Reg)
				shiftSrc = r28
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d43.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d43.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d43.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d44)
			}
		}
		if d44.Loc == scm.LocReg && d39.Loc == scm.LocReg && d44.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.FreeDesc(&d43)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d44)
		var d45 scm.JITValueDesc
		if d44.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d44.Imm.Int()))))}
		} else {
			r29 := ctx.AllocReg()
			ctx.EmitMovRegReg(r29, d44.Reg)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d45)
		}
		ctx.FreeDesc(&d44)
		var d46 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r30 := ctx.AllocReg()
			ctx.EmitMovRegMem(r30, thisptr.Reg, off)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			ctx.BindReg(r30, &d46)
		}
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d46)
		ctx.EnsureDescsTogether(&d45, &d46)
		var d47 scm.JITValueDesc
		if d45.Loc == scm.LocImm && d46.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() + d46.Imm.Int())}
		} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
			r31 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r31, d45.Reg)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d47)
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d46.Reg}
			ctx.BindReg(d46.Reg, &d47)
		} else if d45.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d45.Imm.Int()))
			ctx.EmitAddInt64(scratch, d46.Reg)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d47)
		} else if d46.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(scratch, d45.Reg)
			if d46.Imm.Int() >= -2147483648 && d46.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d46.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d47)
		} else {
			r32 := ctx.AllocRegExcept(d45.Reg, d46.Reg)
			ctx.EmitMovRegReg(r32, d45.Reg)
			ctx.EmitAddInt64(r32, d46.Reg)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d47)
		}
		if d47.Loc == scm.LocReg && d45.Loc == scm.LocReg && d47.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d45)
		ctx.FreeDesc(&d46)
		ctx.EnsureDesc(&d47)
		ctx.EnsureDesc(&d47)
		var d48 scm.JITValueDesc
		if d47.Loc == scm.LocImm {
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d47.Imm.Int()))))}
		} else {
			r33 := ctx.AllocReg()
			ctx.EmitMovRegReg(r33, d47.Reg)
			ctx.EmitShlRegImm8(r33, 32)
			ctx.EmitShrRegImm8(r33, 32)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			ctx.BindReg(r33, &d48)
		}
		ctx.FreeDesc(&d47)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d48)
		ctx.EnsureDescsTogether(&idxInt, &d48)
		var d49 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d48.Imm.Int()))}
		} else if d48.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(idxInt.Reg)
			if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d48.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r34, scm.CondUnsignedBelow)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r34}
			ctx.BindReg(r34, &d49)
		} else if idxInt.Loc == scm.LocImm {
			r35 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d48.Reg)
			ctx.EmitSetcc(r35, scm.CondUnsignedBelow)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r35}
			ctx.BindReg(r35, &d49)
		} else {
			r36 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d48.Reg)
			ctx.EmitSetcc(r36, scm.CondUnsignedBelow)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r36}
			ctx.BindReg(r36, &d49)
		}
		ctx.FreeDesc(&d48)
		d50 = d49
		ctx.EnsureDesc(&d50)
		if d50.Loc != scm.LocImm && d50.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d50.Loc == scm.LocImm {
			if d50.Imm.Bool() {
				if ps.General {
				}
				ps51 := scm.PhiState{General: ps.General}
				ps51.OverlayValues = make([]scm.JITValueDesc, 51)
				ps51.OverlayValues[1] = d1
				ps51.OverlayValues[2] = d2
				ps51.OverlayValues[3] = d3
				ps51.OverlayValues[4] = d4
				ps51.OverlayValues[5] = d5
				ps51.OverlayValues[6] = d6
				ps51.OverlayValues[7] = d7
				ps51.OverlayValues[8] = d8
				ps51.OverlayValues[9] = d9
				ps51.OverlayValues[10] = d10
				ps51.OverlayValues[11] = d11
				ps51.OverlayValues[12] = d12
				ps51.OverlayValues[13] = d13
				ps51.OverlayValues[14] = d14
				ps51.OverlayValues[15] = d15
				ps51.OverlayValues[17] = d17
				ps51.OverlayValues[18] = d18
				ps51.OverlayValues[19] = d19
				ps51.OverlayValues[20] = d20
				ps51.OverlayValues[21] = d21
				ps51.OverlayValues[22] = d22
				ps51.OverlayValues[23] = d23
				ps51.OverlayValues[24] = d24
				ps51.OverlayValues[25] = d25
				ps51.OverlayValues[26] = d26
				ps51.OverlayValues[27] = d27
				ps51.OverlayValues[28] = d28
				ps51.OverlayValues[29] = d29
				ps51.OverlayValues[30] = d30
				ps51.OverlayValues[31] = d31
				ps51.OverlayValues[32] = d32
				ps51.OverlayValues[33] = d33
				ps51.OverlayValues[34] = d34
				ps51.OverlayValues[35] = d35
				ps51.OverlayValues[36] = d36
				ps51.OverlayValues[37] = d37
				ps51.OverlayValues[38] = d38
				ps51.OverlayValues[39] = d39
				ps51.OverlayValues[40] = d40
				ps51.OverlayValues[41] = d41
				ps51.OverlayValues[42] = d42
				ps51.OverlayValues[43] = d43
				ps51.OverlayValues[44] = d44
				ps51.OverlayValues[45] = d45
				ps51.OverlayValues[46] = d46
				ps51.OverlayValues[47] = d47
				ps51.OverlayValues[48] = d48
				ps51.OverlayValues[49] = d49
				ps51.OverlayValues[50] = d50
				return bbs[3].RenderPS(ps51)
			}
			if ps.General {
			}
			ps52 := scm.PhiState{General: ps.General}
			ps52.OverlayValues = make([]scm.JITValueDesc, 51)
			ps52.OverlayValues[1] = d1
			ps52.OverlayValues[2] = d2
			ps52.OverlayValues[3] = d3
			ps52.OverlayValues[4] = d4
			ps52.OverlayValues[5] = d5
			ps52.OverlayValues[6] = d6
			ps52.OverlayValues[7] = d7
			ps52.OverlayValues[8] = d8
			ps52.OverlayValues[9] = d9
			ps52.OverlayValues[10] = d10
			ps52.OverlayValues[11] = d11
			ps52.OverlayValues[12] = d12
			ps52.OverlayValues[13] = d13
			ps52.OverlayValues[14] = d14
			ps52.OverlayValues[15] = d15
			ps52.OverlayValues[17] = d17
			ps52.OverlayValues[18] = d18
			ps52.OverlayValues[19] = d19
			ps52.OverlayValues[20] = d20
			ps52.OverlayValues[21] = d21
			ps52.OverlayValues[22] = d22
			ps52.OverlayValues[23] = d23
			ps52.OverlayValues[24] = d24
			ps52.OverlayValues[25] = d25
			ps52.OverlayValues[26] = d26
			ps52.OverlayValues[27] = d27
			ps52.OverlayValues[28] = d28
			ps52.OverlayValues[29] = d29
			ps52.OverlayValues[30] = d30
			ps52.OverlayValues[31] = d31
			ps52.OverlayValues[32] = d32
			ps52.OverlayValues[33] = d33
			ps52.OverlayValues[34] = d34
			ps52.OverlayValues[35] = d35
			ps52.OverlayValues[36] = d36
			ps52.OverlayValues[37] = d37
			ps52.OverlayValues[38] = d38
			ps52.OverlayValues[39] = d39
			ps52.OverlayValues[40] = d40
			ps52.OverlayValues[41] = d41
			ps52.OverlayValues[42] = d42
			ps52.OverlayValues[43] = d43
			ps52.OverlayValues[44] = d44
			ps52.OverlayValues[45] = d45
			ps52.OverlayValues[46] = d46
			ps52.OverlayValues[47] = d47
			ps52.OverlayValues[48] = d48
			ps52.OverlayValues[49] = d49
			ps52.OverlayValues[50] = d50
			return bbs[5].RenderPS(ps52)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d53 := ps.PhiValues[0]
				ctx.EnsureDesc(&d53)
				ctx.EmitStoreToStack(d53, int32(bbs[1].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d54 := ps.PhiValues[1]
				ctx.EnsureDesc(&d54)
				ctx.EmitStoreToStack(d54, int32(bbs[1].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d55 := ps.PhiValues[2]
				ctx.EnsureDesc(&d55)
				ctx.EmitStoreToStack(d55, int32(bbs[1].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[1].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d50.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl6)
		ps56 := scm.PhiState{General: true}
		ps56.OverlayValues = make([]scm.JITValueDesc, 56)
		ps56.OverlayValues[1] = d1
		ps56.OverlayValues[2] = d2
		ps56.OverlayValues[3] = d3
		ps56.OverlayValues[4] = d4
		ps56.OverlayValues[5] = d5
		ps56.OverlayValues[6] = d6
		ps56.OverlayValues[7] = d7
		ps56.OverlayValues[8] = d8
		ps56.OverlayValues[9] = d9
		ps56.OverlayValues[10] = d10
		ps56.OverlayValues[11] = d11
		ps56.OverlayValues[12] = d12
		ps56.OverlayValues[13] = d13
		ps56.OverlayValues[14] = d14
		ps56.OverlayValues[15] = d15
		ps56.OverlayValues[17] = d17
		ps56.OverlayValues[18] = d18
		ps56.OverlayValues[19] = d19
		ps56.OverlayValues[20] = d20
		ps56.OverlayValues[21] = d21
		ps56.OverlayValues[22] = d22
		ps56.OverlayValues[23] = d23
		ps56.OverlayValues[24] = d24
		ps56.OverlayValues[25] = d25
		ps56.OverlayValues[26] = d26
		ps56.OverlayValues[27] = d27
		ps56.OverlayValues[28] = d28
		ps56.OverlayValues[29] = d29
		ps56.OverlayValues[30] = d30
		ps56.OverlayValues[31] = d31
		ps56.OverlayValues[32] = d32
		ps56.OverlayValues[33] = d33
		ps56.OverlayValues[34] = d34
		ps56.OverlayValues[35] = d35
		ps56.OverlayValues[36] = d36
		ps56.OverlayValues[37] = d37
		ps56.OverlayValues[38] = d38
		ps56.OverlayValues[39] = d39
		ps56.OverlayValues[40] = d40
		ps56.OverlayValues[41] = d41
		ps56.OverlayValues[42] = d42
		ps56.OverlayValues[43] = d43
		ps56.OverlayValues[44] = d44
		ps56.OverlayValues[45] = d45
		ps56.OverlayValues[46] = d46
		ps56.OverlayValues[47] = d47
		ps56.OverlayValues[48] = d48
		ps56.OverlayValues[49] = d49
		ps56.OverlayValues[50] = d50
		ps56.OverlayValues[53] = d53
		ps56.OverlayValues[54] = d54
		ps56.OverlayValues[55] = d55
		ps57 := scm.PhiState{General: true}
		ps57.OverlayValues = make([]scm.JITValueDesc, 56)
		ps57.OverlayValues[1] = d1
		ps57.OverlayValues[2] = d2
		ps57.OverlayValues[3] = d3
		ps57.OverlayValues[4] = d4
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
		ps57.OverlayValues[17] = d17
		ps57.OverlayValues[18] = d18
		ps57.OverlayValues[19] = d19
		ps57.OverlayValues[20] = d20
		ps57.OverlayValues[21] = d21
		ps57.OverlayValues[22] = d22
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
		ps57.OverlayValues[53] = d53
		ps57.OverlayValues[54] = d54
		ps57.OverlayValues[55] = d55
		snap58 := d1
		snap59 := d2
		snap60 := d3
		snap61 := d4
		snap62 := d5
		snap63 := d6
		snap64 := d7
		snap65 := d8
		snap66 := d9
		snap67 := d10
		snap68 := d11
		snap69 := d12
		snap70 := d13
		snap71 := d14
		snap72 := d15
		snap73 := d17
		snap74 := d18
		snap75 := d19
		snap76 := d20
		snap77 := d21
		snap78 := d22
		snap79 := d23
		snap80 := d24
		snap81 := d25
		snap82 := d26
		snap83 := d27
		snap84 := d28
		snap85 := d29
		snap86 := d30
		snap87 := d31
		snap88 := d32
		snap89 := d33
		snap90 := d34
		snap91 := d35
		snap92 := d36
		snap93 := d37
		snap94 := d38
		snap95 := d39
		snap96 := d40
		snap97 := d41
		snap98 := d42
		snap99 := d43
		snap100 := d44
		snap101 := d45
		snap102 := d46
		snap103 := d47
		snap104 := d48
		snap105 := d49
		snap106 := d50
		snap107 := d53
		snap108 := d54
		snap109 := d55
		alloc110 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps57)
		}
		ctx.RestoreAllocState(alloc110)
		d1 = snap58
		d2 = snap59
		d3 = snap60
		d4 = snap61
		d5 = snap62
		d6 = snap63
		d7 = snap64
		d8 = snap65
		d9 = snap66
		d10 = snap67
		d11 = snap68
		d12 = snap69
		d13 = snap70
		d14 = snap71
		d15 = snap72
		d17 = snap73
		d18 = snap74
		d19 = snap75
		d20 = snap76
		d21 = snap77
		d22 = snap78
		d23 = snap79
		d24 = snap80
		d25 = snap81
		d26 = snap82
		d27 = snap83
		d28 = snap84
		d29 = snap85
		d30 = snap86
		d31 = snap87
		d32 = snap88
		d33 = snap89
		d34 = snap90
		d35 = snap91
		d36 = snap92
		d37 = snap93
		d38 = snap94
		d39 = snap95
		d40 = snap96
		d41 = snap97
		d42 = snap98
		d43 = snap99
		d44 = snap100
		d45 = snap101
		d46 = snap102
		d47 = snap103
		d48 = snap104
		d49 = snap105
		d50 = snap106
		d53 = snap107
		d54 = snap108
		d55 = snap109
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps56)
		}
		return result
		ctx.FreeDesc(&d49)
		return result
	}
	bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d111 := ps.PhiValues[0]
				ctx.EnsureDesc(&d111)
				ctx.EmitStoreToStack(d111, int32(bbs[2].PhiBase)+int32(0))
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d4 = ps.PhiValues[0]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d4)
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d112 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d4.Imm.Int()))))}
		} else {
			r37 := ctx.AllocReg()
			ctx.EmitMovRegReg(r37, d4.Reg)
			ctx.EmitShlRegImm8(r37, 32)
			ctx.EmitShrRegImm8(r37, 32)
			d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			ctx.BindReg(r37, &d112)
		}
		ctx.EnsureDesc(&d112)
		if thisptr.Loc == scm.LocImm {
			baseReg := ctx.AllocReg()
			if d112.Loc == scm.LocReg {
				ctx.FreeReg(baseReg)
				baseReg = ctx.AllocRegExcept(d112.Reg)
			}
			ctx.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
			if d112.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, baseReg, 0)
			} else {
				ctx.EmitStoreRegMem(d112.Reg, baseReg, 0)
			}
			ctx.FreeReg(baseReg)
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
			if d112.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
				ctx.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
			} else {
				ctx.EmitStoreRegMem(d112.Reg, thisptr.Reg, off)
			}
		}
		ctx.FreeDesc(&d112)
		ctx.EnsureDesc(&d4)
		d113 = d4
		_ = d113
		ctx.StabilizeDescForControlFlow(&d113)
		ctx.StabilizeDescForControlFlow(&d4)
		bbpos_2_0 := int32(-1)
		_ = bbpos_2_0
		lbl18 := ctx.ReserveLabel()
		_ = lbl18
		bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl18)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d113)
		ctx.EnsureDesc(&d113)
		var d114 scm.JITValueDesc
		if d113.Loc == scm.LocImm {
			d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d113.Imm.Int()))))}
		} else {
			r38 := ctx.AllocReg()
			ctx.EmitMovRegReg(r38, d113.Reg)
			ctx.EmitShlRegImm8(r38, 32)
			ctx.EmitShrRegImm8(r38, 32)
			d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d114)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d115 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r39 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r39, thisptr.Reg, off)
			d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.BindReg(r39, &d115)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d115)
		ctx.EnsureDesc(&d115)
		var d116 scm.JITValueDesc
		if d115.Loc == scm.LocImm {
			d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d115.Imm.Int()))))}
		} else {
			r40 := ctx.AllocReg()
			ctx.EmitMovRegReg(r40, d115.Reg)
			ctx.EmitShlRegImm8(r40, 56)
			ctx.EmitShrRegImm8(r40, 56)
			d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			ctx.BindReg(r40, &d116)
		}
		ctx.FreeDesc(&d115)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d114)
		ctx.EnsureDesc(&d116)
		ctx.EnsureDescsTogether(&d114, &d116)
		var d117 scm.JITValueDesc
		if d114.Loc == scm.LocImm && d116.Loc == scm.LocImm {
			d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d114.Imm.Int() * d116.Imm.Int())}
		} else if d114.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d116.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d114.Imm.Int()))
			ctx.EmitImulInt64(scratch, d116.Reg)
			d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d117)
		} else if d116.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d114.Reg)
			ctx.EmitMovRegReg(scratch, d114.Reg)
			if d116.Imm.Int() >= -2147483648 && d116.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d116.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d116.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d117)
		} else {
			r41 := ctx.AllocRegExcept(d114.Reg, d116.Reg)
			ctx.EmitMovRegReg(r41, d114.Reg)
			ctx.EmitImulInt64(r41, d116.Reg)
			d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d117)
		}
		if d117.Loc == scm.LocReg && d114.Loc == scm.LocReg && d117.Reg == d114.Reg {
			ctx.TransferReg(d114.Reg)
			d114.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d114)
		ctx.FreeDesc(&d116)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d117)
		var d118 scm.JITValueDesc
		if d117.Loc == scm.LocImm {
			d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() / 64)}
		} else {
			r42 := ctx.AllocRegExcept(d117.Reg)
			ctx.EmitMovRegReg(r42, d117.Reg)
			ctx.EmitShrRegImm8(r42, 6)
			d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d118)
		}
		if d118.Loc == scm.LocReg && d117.Loc == scm.LocReg && d118.Reg == d117.Reg {
			ctx.TransferReg(d117.Reg)
			d117.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d117)
		var d119 scm.JITValueDesc
		if d117.Loc == scm.LocImm {
			d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() % 64)}
		} else {
			r43 := ctx.AllocRegExcept(d117.Reg)
			ctx.EmitMovRegReg(r43, d117.Reg)
			ctx.EmitAndRegImm32(r43, 63)
			d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d119)
		}
		if d119.Loc == scm.LocReg && d117.Loc == scm.LocReg && d119.Reg == d117.Reg {
			ctx.TransferReg(d117.Reg)
			d117.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d117)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d120 scm.JITValueDesc
		r44 := ctx.AllocReg()
		r45 := ctx.AllocRegExcept(r44)
		r46 := ctx.AllocRegExcept(r44, r45)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r44, uint64(dataPtr))
			ctx.EmitMovRegImm64(r45, uint64(sliceLen))
			ctx.EmitMovRegImm64(r46, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
			ctx.EmitMovRegMem(r44, thisptr.Reg, off)
			ctx.EmitMovRegMem(r45, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r46, thisptr.Reg, off+16)
		}
		d120 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r44, Reg2: r45, Reg3: r46}
		ctx.BindReg(r44, &d120)
		ctx.BindReg(r45, &d120)
		ctx.BindReg(r46, &d120)
		ctx.BindReg(r44, &d120)
		ctx.BindReg(r45, &d120)
		ctx.BindReg(r46, &d120)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d118)
		ctx.ReclaimUntrackedRegs()
		d122 = ctx.EmitSliceElementAddress(&d120, &d118, 8)
		ctx.EnsureDesc(&d122)
		ctx.EmitMovRegMem(d122.Reg, d122.Reg, 0)
		d121 = d122
		d121.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d121)
		ctx.EnsureDesc(&d119)
		ctx.EnsureDescsTogether(&d121, &d119)
		var d123 scm.JITValueDesc
		if d121.Loc == scm.LocImm && d119.Loc == scm.LocImm {
			d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d121.Imm.Int()) << uint64(d119.Imm.Int())))}
		} else if d119.Loc == scm.LocImm {
			r47 := ctx.AllocRegExcept(d121.Reg)
			ctx.EmitMovRegReg(r47, d121.Reg)
			ctx.EmitShlRegImm8(r47, uint8(d119.Imm.Int()))
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d123)
		} else {
			{
				shiftSrc := d121.Reg
				r48 := ctx.AllocRegExcept(d121.Reg, d119.Reg)
				ctx.EmitMovRegReg(r48, d121.Reg)
				shiftSrc = r48
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d119.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d119.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d119.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d123)
			}
		}
		if d123.Loc == scm.LocReg && d121.Loc == scm.LocReg && d123.Reg == d121.Reg {
			ctx.TransferReg(d121.Reg)
			d121.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d121)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d118)
		ctx.EnsureDesc(&d118)
		var d124 scm.JITValueDesc
		if d118.Loc == scm.LocImm {
			d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d118.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d118.Reg)
			ctx.EmitMovRegReg(scratch, d118.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d124)
		}
		if d124.Loc == scm.LocReg && d118.Loc == scm.LocReg && d124.Reg == d118.Reg {
			ctx.TransferReg(d118.Reg)
			d118.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d118)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d124)
		ctx.ReclaimUntrackedRegs()
		d126 = ctx.EmitSliceElementAddress(&d120, &d124, 8)
		ctx.EnsureDesc(&d126)
		ctx.EmitMovRegMem(d126.Reg, d126.Reg, 0)
		d125 = d126
		d125.Type = scm.TagInt
		ctx.FreeDesc(&d124)
		ctx.ReclaimUntrackedRegs()
		d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d119)
		ctx.EnsureDescsTogether(&d127, &d119)
		var d128 scm.JITValueDesc
		if d127.Loc == scm.LocImm && d119.Loc == scm.LocImm {
			d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() - d119.Imm.Int())}
		} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
			r49 := ctx.AllocRegExcept(d127.Reg)
			ctx.EmitMovRegReg(r49, d127.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
			ctx.BindReg(r49, &d128)
		} else if d127.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d119.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d127.Imm.Int()))
			ctx.EmitSubInt64(scratch, d119.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d128)
		} else if d119.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d127.Reg)
			ctx.EmitMovRegReg(scratch, d127.Reg)
			if d119.Imm.Int() >= -2147483648 && d119.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d119.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d119.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d128)
		} else {
			r50 := ctx.AllocRegExcept(d127.Reg, d119.Reg)
			ctx.EmitMovRegReg(r50, d127.Reg)
			ctx.EmitSubInt64(r50, d119.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			ctx.BindReg(r50, &d128)
		}
		if d128.Loc == scm.LocReg && d127.Loc == scm.LocReg && d128.Reg == d127.Reg {
			ctx.TransferReg(d127.Reg)
			d127.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d119)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d125)
		ctx.EnsureDesc(&d128)
		ctx.EnsureDescsTogether(&d125, &d128)
		var d129 scm.JITValueDesc
		if d125.Loc == scm.LocImm && d128.Loc == scm.LocImm {
			d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d125.Imm.Int()) >> uint64(d128.Imm.Int())))}
		} else if d128.Loc == scm.LocImm {
			r51 := ctx.AllocRegExcept(d125.Reg)
			ctx.EmitMovRegReg(r51, d125.Reg)
			ctx.EmitShrRegImm8(r51, uint8(d128.Imm.Int()))
			d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			ctx.BindReg(r51, &d129)
		} else {
			{
				shiftSrc := d125.Reg
				r52 := ctx.AllocRegExcept(d125.Reg, d128.Reg)
				ctx.EmitMovRegReg(r52, d125.Reg)
				shiftSrc = r52
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d128.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d128.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d128.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d129)
			}
		}
		if d129.Loc == scm.LocReg && d125.Loc == scm.LocReg && d129.Reg == d125.Reg {
			ctx.TransferReg(d125.Reg)
			d125.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d125)
		ctx.FreeDesc(&d128)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d123)
		ctx.EnsureDesc(&d129)
		var d130 scm.JITValueDesc
		if d123.Loc == scm.LocImm && d129.Loc == scm.LocImm {
			d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() | d129.Imm.Int())}
		} else if d123.Loc == scm.LocImm && d123.Imm.Int() == 0 {
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d129.Reg}
			ctx.BindReg(d129.Reg, &d130)
		} else if d129.Loc == scm.LocImm && d129.Imm.Int() == 0 {
			r53 := ctx.AllocRegExcept(d123.Reg)
			ctx.EmitMovRegReg(r53, d123.Reg)
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			ctx.BindReg(r53, &d130)
		} else if d123.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d129.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d123.Imm.Int()))
			ctx.EmitOrInt64(scratch, d129.Reg)
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d130)
		} else if d129.Loc == scm.LocImm {
			r54 := ctx.AllocRegExcept(d123.Reg)
			ctx.EmitMovRegReg(r54, d123.Reg)
			if d129.Imm.Int() >= -2147483648 && d129.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r54, int32(d129.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
				ctx.EmitOrInt64(r54, scm.RegR11)
			}
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			ctx.BindReg(r54, &d130)
		} else {
			r55 := ctx.AllocRegExcept(d123.Reg, d129.Reg)
			ctx.EmitMovRegReg(r55, d123.Reg)
			ctx.EmitOrInt64(r55, d129.Reg)
			d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d130)
		}
		if d130.Loc == scm.LocReg && d123.Loc == scm.LocReg && d130.Reg == d123.Reg {
			ctx.TransferReg(d123.Reg)
			d123.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d123)
		ctx.FreeDesc(&d129)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d131 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 48)
			r56 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r56, thisptr.Reg, off)
			d131 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
			ctx.BindReg(r56, &d131)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d131)
		ctx.EnsureDesc(&d131)
		var d132 scm.JITValueDesc
		if d131.Loc == scm.LocImm {
			d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d131.Imm.Int()))))}
		} else {
			r57 := ctx.AllocReg()
			ctx.EmitMovRegReg(r57, d131.Reg)
			ctx.EmitShlRegImm8(r57, 56)
			ctx.EmitShrRegImm8(r57, 56)
			d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			ctx.BindReg(r57, &d132)
		}
		ctx.FreeDesc(&d131)
		ctx.ReclaimUntrackedRegs()
		d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d132)
		ctx.EnsureDescsTogether(&d133, &d132)
		var d134 scm.JITValueDesc
		if d133.Loc == scm.LocImm && d132.Loc == scm.LocImm {
			d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d133.Imm.Int() - d132.Imm.Int())}
		} else if d132.Loc == scm.LocImm && d132.Imm.Int() == 0 {
			r58 := ctx.AllocRegExcept(d133.Reg)
			ctx.EmitMovRegReg(r58, d133.Reg)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
			ctx.BindReg(r58, &d134)
		} else if d133.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d132.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d133.Imm.Int()))
			ctx.EmitSubInt64(scratch, d132.Reg)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d134)
		} else if d132.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d133.Reg)
			ctx.EmitMovRegReg(scratch, d133.Reg)
			if d132.Imm.Int() >= -2147483648 && d132.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d132.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d132.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d134)
		} else {
			r59 := ctx.AllocRegExcept(d133.Reg, d132.Reg)
			ctx.EmitMovRegReg(r59, d133.Reg)
			ctx.EmitSubInt64(r59, d132.Reg)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			ctx.BindReg(r59, &d134)
		}
		if d134.Loc == scm.LocReg && d133.Loc == scm.LocReg && d134.Reg == d133.Reg {
			ctx.TransferReg(d133.Reg)
			d133.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d132)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d130)
		ctx.EnsureDesc(&d134)
		ctx.EnsureDescsTogether(&d130, &d134)
		var d135 scm.JITValueDesc
		if d130.Loc == scm.LocImm && d134.Loc == scm.LocImm {
			d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d130.Imm.Int()) >> uint64(d134.Imm.Int())))}
		} else if d134.Loc == scm.LocImm {
			r60 := ctx.AllocRegExcept(d130.Reg)
			ctx.EmitMovRegReg(r60, d130.Reg)
			ctx.EmitShrRegImm8(r60, uint8(d134.Imm.Int()))
			d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			ctx.BindReg(r60, &d135)
		} else {
			{
				shiftSrc := d130.Reg
				r61 := ctx.AllocRegExcept(d130.Reg, d134.Reg)
				ctx.EmitMovRegReg(r61, d130.Reg)
				shiftSrc = r61
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d134.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d134.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d134.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d135)
			}
		}
		if d135.Loc == scm.LocReg && d130.Loc == scm.LocReg && d135.Reg == d130.Reg {
			ctx.TransferReg(d130.Reg)
			d130.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d130)
		ctx.FreeDesc(&d134)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d135)
		ctx.EnsureDesc(&d135)
		ctx.EnsureDesc(&d135)
		var d136 scm.JITValueDesc
		if d135.Loc == scm.LocImm {
			d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d135.Imm.Int()))))}
		} else {
			r62 := ctx.AllocReg()
			ctx.EmitMovRegReg(r62, d135.Reg)
			d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			ctx.BindReg(r62, &d136)
		}
		ctx.FreeDesc(&d135)
		var d137 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
			r63 := ctx.AllocReg()
			ctx.EmitMovRegMem(r63, thisptr.Reg, off)
			d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
			ctx.BindReg(r63, &d137)
		}
		ctx.EnsureDesc(&d136)
		ctx.EnsureDesc(&d137)
		ctx.EnsureDescsTogether(&d136, &d137)
		var d138 scm.JITValueDesc
		if d136.Loc == scm.LocImm && d137.Loc == scm.LocImm {
			d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() + d137.Imm.Int())}
		} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
			r64 := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(r64, d136.Reg)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
			ctx.BindReg(r64, &d138)
		} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d137.Reg}
			ctx.BindReg(d137.Reg, &d138)
		} else if d136.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d137.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d136.Imm.Int()))
			ctx.EmitAddInt64(scratch, d137.Reg)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d138)
		} else if d137.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d136.Reg)
			ctx.EmitMovRegReg(scratch, d136.Reg)
			if d137.Imm.Int() >= -2147483648 && d137.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d137.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d138)
		} else {
			r65 := ctx.AllocRegExcept(d136.Reg, d137.Reg)
			ctx.EmitMovRegReg(r65, d136.Reg)
			ctx.EmitAddInt64(r65, d137.Reg)
			d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			ctx.BindReg(r65, &d138)
		}
		if d138.Loc == scm.LocReg && d136.Loc == scm.LocReg && d138.Reg == d136.Reg {
			ctx.TransferReg(d136.Reg)
			d136.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d138)
		ctx.FreeDesc(&d136)
		ctx.FreeDesc(&d137)
		var d139 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 80)
			r66 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r66, thisptr.Reg, off)
			d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
			ctx.BindReg(r66, &d139)
		}
		d140 = d139
		ctx.EnsureDesc(&d140)
		if d140.Loc != scm.LocImm && d140.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d140.Loc == scm.LocImm {
			if d140.Imm.Bool() {
				if ps.General {
				}
				ps141 := scm.PhiState{General: ps.General}
				ps141.OverlayValues = make([]scm.JITValueDesc, 141)
				ps141.OverlayValues[1] = d1
				ps141.OverlayValues[2] = d2
				ps141.OverlayValues[3] = d3
				ps141.OverlayValues[4] = d4
				ps141.OverlayValues[5] = d5
				ps141.OverlayValues[6] = d6
				ps141.OverlayValues[7] = d7
				ps141.OverlayValues[8] = d8
				ps141.OverlayValues[9] = d9
				ps141.OverlayValues[10] = d10
				ps141.OverlayValues[11] = d11
				ps141.OverlayValues[12] = d12
				ps141.OverlayValues[13] = d13
				ps141.OverlayValues[14] = d14
				ps141.OverlayValues[15] = d15
				ps141.OverlayValues[17] = d17
				ps141.OverlayValues[18] = d18
				ps141.OverlayValues[19] = d19
				ps141.OverlayValues[20] = d20
				ps141.OverlayValues[21] = d21
				ps141.OverlayValues[22] = d22
				ps141.OverlayValues[23] = d23
				ps141.OverlayValues[24] = d24
				ps141.OverlayValues[25] = d25
				ps141.OverlayValues[26] = d26
				ps141.OverlayValues[27] = d27
				ps141.OverlayValues[28] = d28
				ps141.OverlayValues[29] = d29
				ps141.OverlayValues[30] = d30
				ps141.OverlayValues[31] = d31
				ps141.OverlayValues[32] = d32
				ps141.OverlayValues[33] = d33
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
				ps141.OverlayValues[53] = d53
				ps141.OverlayValues[54] = d54
				ps141.OverlayValues[55] = d55
				ps141.OverlayValues[111] = d111
				ps141.OverlayValues[112] = d112
				ps141.OverlayValues[113] = d113
				ps141.OverlayValues[114] = d114
				ps141.OverlayValues[115] = d115
				ps141.OverlayValues[116] = d116
				ps141.OverlayValues[117] = d117
				ps141.OverlayValues[118] = d118
				ps141.OverlayValues[119] = d119
				ps141.OverlayValues[120] = d120
				ps141.OverlayValues[121] = d121
				ps141.OverlayValues[122] = d122
				ps141.OverlayValues[123] = d123
				ps141.OverlayValues[124] = d124
				ps141.OverlayValues[125] = d125
				ps141.OverlayValues[126] = d126
				ps141.OverlayValues[127] = d127
				ps141.OverlayValues[128] = d128
				ps141.OverlayValues[129] = d129
				ps141.OverlayValues[130] = d130
				ps141.OverlayValues[131] = d131
				ps141.OverlayValues[132] = d132
				ps141.OverlayValues[133] = d133
				ps141.OverlayValues[134] = d134
				ps141.OverlayValues[135] = d135
				ps141.OverlayValues[136] = d136
				ps141.OverlayValues[137] = d137
				ps141.OverlayValues[138] = d138
				ps141.OverlayValues[139] = d139
				ps141.OverlayValues[140] = d140
				return bbs[13].RenderPS(ps141)
			}
			if ps.General {
			}
			ps142 := scm.PhiState{General: ps.General}
			ps142.OverlayValues = make([]scm.JITValueDesc, 141)
			ps142.OverlayValues[1] = d1
			ps142.OverlayValues[2] = d2
			ps142.OverlayValues[3] = d3
			ps142.OverlayValues[4] = d4
			ps142.OverlayValues[5] = d5
			ps142.OverlayValues[6] = d6
			ps142.OverlayValues[7] = d7
			ps142.OverlayValues[8] = d8
			ps142.OverlayValues[9] = d9
			ps142.OverlayValues[10] = d10
			ps142.OverlayValues[11] = d11
			ps142.OverlayValues[12] = d12
			ps142.OverlayValues[13] = d13
			ps142.OverlayValues[14] = d14
			ps142.OverlayValues[15] = d15
			ps142.OverlayValues[17] = d17
			ps142.OverlayValues[18] = d18
			ps142.OverlayValues[19] = d19
			ps142.OverlayValues[20] = d20
			ps142.OverlayValues[21] = d21
			ps142.OverlayValues[22] = d22
			ps142.OverlayValues[23] = d23
			ps142.OverlayValues[24] = d24
			ps142.OverlayValues[25] = d25
			ps142.OverlayValues[26] = d26
			ps142.OverlayValues[27] = d27
			ps142.OverlayValues[28] = d28
			ps142.OverlayValues[29] = d29
			ps142.OverlayValues[30] = d30
			ps142.OverlayValues[31] = d31
			ps142.OverlayValues[32] = d32
			ps142.OverlayValues[33] = d33
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
			ps142.OverlayValues[53] = d53
			ps142.OverlayValues[54] = d54
			ps142.OverlayValues[55] = d55
			ps142.OverlayValues[111] = d111
			ps142.OverlayValues[112] = d112
			ps142.OverlayValues[113] = d113
			ps142.OverlayValues[114] = d114
			ps142.OverlayValues[115] = d115
			ps142.OverlayValues[116] = d116
			ps142.OverlayValues[117] = d117
			ps142.OverlayValues[118] = d118
			ps142.OverlayValues[119] = d119
			ps142.OverlayValues[120] = d120
			ps142.OverlayValues[121] = d121
			ps142.OverlayValues[122] = d122
			ps142.OverlayValues[123] = d123
			ps142.OverlayValues[124] = d124
			ps142.OverlayValues[125] = d125
			ps142.OverlayValues[126] = d126
			ps142.OverlayValues[127] = d127
			ps142.OverlayValues[128] = d128
			ps142.OverlayValues[129] = d129
			ps142.OverlayValues[130] = d130
			ps142.OverlayValues[131] = d131
			ps142.OverlayValues[132] = d132
			ps142.OverlayValues[133] = d133
			ps142.OverlayValues[134] = d134
			ps142.OverlayValues[135] = d135
			ps142.OverlayValues[136] = d136
			ps142.OverlayValues[137] = d137
			ps142.OverlayValues[138] = d138
			ps142.OverlayValues[139] = d139
			ps142.OverlayValues[140] = d140
			return bbs[12].RenderPS(ps142)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d143 := ps.PhiValues[0]
				ctx.EnsureDesc(&d143)
				ctx.EmitStoreToStack(d143, int32(bbs[2].PhiBase)+int32(0))
			}
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl19 := ctx.ReserveLabel()
		lbl20 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d140.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl19)
		ctx.EmitJmp(lbl20)
		ctx.MarkLabel(lbl19)
		ctx.EmitJmp(lbl14)
		ctx.MarkLabel(lbl20)
		ctx.EmitJmp(lbl13)
		ps144 := scm.PhiState{General: true}
		ps144.OverlayValues = make([]scm.JITValueDesc, 144)
		ps144.OverlayValues[1] = d1
		ps144.OverlayValues[2] = d2
		ps144.OverlayValues[3] = d3
		ps144.OverlayValues[4] = d4
		ps144.OverlayValues[5] = d5
		ps144.OverlayValues[6] = d6
		ps144.OverlayValues[7] = d7
		ps144.OverlayValues[8] = d8
		ps144.OverlayValues[9] = d9
		ps144.OverlayValues[10] = d10
		ps144.OverlayValues[11] = d11
		ps144.OverlayValues[12] = d12
		ps144.OverlayValues[13] = d13
		ps144.OverlayValues[14] = d14
		ps144.OverlayValues[15] = d15
		ps144.OverlayValues[17] = d17
		ps144.OverlayValues[18] = d18
		ps144.OverlayValues[19] = d19
		ps144.OverlayValues[20] = d20
		ps144.OverlayValues[21] = d21
		ps144.OverlayValues[22] = d22
		ps144.OverlayValues[23] = d23
		ps144.OverlayValues[24] = d24
		ps144.OverlayValues[25] = d25
		ps144.OverlayValues[26] = d26
		ps144.OverlayValues[27] = d27
		ps144.OverlayValues[28] = d28
		ps144.OverlayValues[29] = d29
		ps144.OverlayValues[30] = d30
		ps144.OverlayValues[31] = d31
		ps144.OverlayValues[32] = d32
		ps144.OverlayValues[33] = d33
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
		ps144.OverlayValues[53] = d53
		ps144.OverlayValues[54] = d54
		ps144.OverlayValues[55] = d55
		ps144.OverlayValues[111] = d111
		ps144.OverlayValues[112] = d112
		ps144.OverlayValues[113] = d113
		ps144.OverlayValues[114] = d114
		ps144.OverlayValues[115] = d115
		ps144.OverlayValues[116] = d116
		ps144.OverlayValues[117] = d117
		ps144.OverlayValues[118] = d118
		ps144.OverlayValues[119] = d119
		ps144.OverlayValues[120] = d120
		ps144.OverlayValues[121] = d121
		ps144.OverlayValues[122] = d122
		ps144.OverlayValues[123] = d123
		ps144.OverlayValues[124] = d124
		ps144.OverlayValues[125] = d125
		ps144.OverlayValues[126] = d126
		ps144.OverlayValues[127] = d127
		ps144.OverlayValues[128] = d128
		ps144.OverlayValues[129] = d129
		ps144.OverlayValues[130] = d130
		ps144.OverlayValues[131] = d131
		ps144.OverlayValues[132] = d132
		ps144.OverlayValues[133] = d133
		ps144.OverlayValues[134] = d134
		ps144.OverlayValues[135] = d135
		ps144.OverlayValues[136] = d136
		ps144.OverlayValues[137] = d137
		ps144.OverlayValues[138] = d138
		ps144.OverlayValues[139] = d139
		ps144.OverlayValues[140] = d140
		ps144.OverlayValues[143] = d143
		ps145 := scm.PhiState{General: true}
		ps145.OverlayValues = make([]scm.JITValueDesc, 144)
		ps145.OverlayValues[1] = d1
		ps145.OverlayValues[2] = d2
		ps145.OverlayValues[3] = d3
		ps145.OverlayValues[4] = d4
		ps145.OverlayValues[5] = d5
		ps145.OverlayValues[6] = d6
		ps145.OverlayValues[7] = d7
		ps145.OverlayValues[8] = d8
		ps145.OverlayValues[9] = d9
		ps145.OverlayValues[10] = d10
		ps145.OverlayValues[11] = d11
		ps145.OverlayValues[12] = d12
		ps145.OverlayValues[13] = d13
		ps145.OverlayValues[14] = d14
		ps145.OverlayValues[15] = d15
		ps145.OverlayValues[17] = d17
		ps145.OverlayValues[18] = d18
		ps145.OverlayValues[19] = d19
		ps145.OverlayValues[20] = d20
		ps145.OverlayValues[21] = d21
		ps145.OverlayValues[22] = d22
		ps145.OverlayValues[23] = d23
		ps145.OverlayValues[24] = d24
		ps145.OverlayValues[25] = d25
		ps145.OverlayValues[26] = d26
		ps145.OverlayValues[27] = d27
		ps145.OverlayValues[28] = d28
		ps145.OverlayValues[29] = d29
		ps145.OverlayValues[30] = d30
		ps145.OverlayValues[31] = d31
		ps145.OverlayValues[32] = d32
		ps145.OverlayValues[33] = d33
		ps145.OverlayValues[34] = d34
		ps145.OverlayValues[35] = d35
		ps145.OverlayValues[36] = d36
		ps145.OverlayValues[37] = d37
		ps145.OverlayValues[38] = d38
		ps145.OverlayValues[39] = d39
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
		ps145.OverlayValues[53] = d53
		ps145.OverlayValues[54] = d54
		ps145.OverlayValues[55] = d55
		ps145.OverlayValues[111] = d111
		ps145.OverlayValues[112] = d112
		ps145.OverlayValues[113] = d113
		ps145.OverlayValues[114] = d114
		ps145.OverlayValues[115] = d115
		ps145.OverlayValues[116] = d116
		ps145.OverlayValues[117] = d117
		ps145.OverlayValues[118] = d118
		ps145.OverlayValues[119] = d119
		ps145.OverlayValues[120] = d120
		ps145.OverlayValues[121] = d121
		ps145.OverlayValues[122] = d122
		ps145.OverlayValues[123] = d123
		ps145.OverlayValues[124] = d124
		ps145.OverlayValues[125] = d125
		ps145.OverlayValues[126] = d126
		ps145.OverlayValues[127] = d127
		ps145.OverlayValues[128] = d128
		ps145.OverlayValues[129] = d129
		ps145.OverlayValues[130] = d130
		ps145.OverlayValues[131] = d131
		ps145.OverlayValues[132] = d132
		ps145.OverlayValues[133] = d133
		ps145.OverlayValues[134] = d134
		ps145.OverlayValues[135] = d135
		ps145.OverlayValues[136] = d136
		ps145.OverlayValues[137] = d137
		ps145.OverlayValues[138] = d138
		ps145.OverlayValues[139] = d139
		ps145.OverlayValues[140] = d140
		ps145.OverlayValues[143] = d143
		snap146 := d1
		snap147 := d2
		snap148 := d3
		snap149 := d4
		snap150 := d5
		snap151 := d6
		snap152 := d7
		snap153 := d8
		snap154 := d9
		snap155 := d10
		snap156 := d11
		snap157 := d12
		snap158 := d13
		snap159 := d14
		snap160 := d15
		snap161 := d17
		snap162 := d18
		snap163 := d19
		snap164 := d20
		snap165 := d21
		snap166 := d22
		snap167 := d23
		snap168 := d24
		snap169 := d25
		snap170 := d26
		snap171 := d27
		snap172 := d28
		snap173 := d29
		snap174 := d30
		snap175 := d31
		snap176 := d32
		snap177 := d33
		snap178 := d34
		snap179 := d35
		snap180 := d36
		snap181 := d37
		snap182 := d38
		snap183 := d39
		snap184 := d40
		snap185 := d41
		snap186 := d42
		snap187 := d43
		snap188 := d44
		snap189 := d45
		snap190 := d46
		snap191 := d47
		snap192 := d48
		snap193 := d49
		snap194 := d50
		snap195 := d53
		snap196 := d54
		snap197 := d55
		snap198 := d111
		snap199 := d112
		snap200 := d113
		snap201 := d114
		snap202 := d115
		snap203 := d116
		snap204 := d117
		snap205 := d118
		snap206 := d119
		snap207 := d120
		snap208 := d121
		snap209 := d122
		snap210 := d123
		snap211 := d124
		snap212 := d125
		snap213 := d126
		snap214 := d127
		snap215 := d128
		snap216 := d129
		snap217 := d130
		snap218 := d131
		snap219 := d132
		snap220 := d133
		snap221 := d134
		snap222 := d135
		snap223 := d136
		snap224 := d137
		snap225 := d138
		snap226 := d139
		snap227 := d140
		snap228 := d143
		alloc229 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps145)
		}
		ctx.RestoreAllocState(alloc229)
		d1 = snap146
		d2 = snap147
		d3 = snap148
		d4 = snap149
		d5 = snap150
		d6 = snap151
		d7 = snap152
		d8 = snap153
		d9 = snap154
		d10 = snap155
		d11 = snap156
		d12 = snap157
		d13 = snap158
		d14 = snap159
		d15 = snap160
		d17 = snap161
		d18 = snap162
		d19 = snap163
		d20 = snap164
		d21 = snap165
		d22 = snap166
		d23 = snap167
		d24 = snap168
		d25 = snap169
		d26 = snap170
		d27 = snap171
		d28 = snap172
		d29 = snap173
		d30 = snap174
		d31 = snap175
		d32 = snap176
		d33 = snap177
		d34 = snap178
		d35 = snap179
		d36 = snap180
		d37 = snap181
		d38 = snap182
		d39 = snap183
		d40 = snap184
		d41 = snap185
		d42 = snap186
		d43 = snap187
		d44 = snap188
		d45 = snap189
		d46 = snap190
		d47 = snap191
		d48 = snap192
		d49 = snap193
		d50 = snap194
		d53 = snap195
		d54 = snap196
		d55 = snap197
		d111 = snap198
		d112 = snap199
		d113 = snap200
		d114 = snap201
		d115 = snap202
		d116 = snap203
		d117 = snap204
		d118 = snap205
		d119 = snap206
		d120 = snap207
		d121 = snap208
		d122 = snap209
		d123 = snap210
		d124 = snap211
		d125 = snap212
		d126 = snap213
		d127 = snap214
		d128 = snap215
		d129 = snap216
		d130 = snap217
		d131 = snap218
		d132 = snap219
		d133 = snap220
		d134 = snap221
		d135 = snap222
		d136 = snap223
		d137 = snap224
		d138 = snap225
		d139 = snap226
		d140 = snap227
		d143 = snap228
		if !bbs[13].Rendered {
			return bbs[13].RenderPS(ps144)
		}
		return result
		ctx.FreeDesc(&d139)
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d230 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d230)
		}
		if d230.Loc == scm.LocImm {
			d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: d230.Type, Imm: scm.NewInt(int64(uint64(d230.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d230.Reg, 32)
			ctx.EmitShrRegImm8(d230.Reg, 32)
		}
		if d230.Loc == scm.LocReg && d1.Loc == scm.LocReg && d230.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d230)
		ctx.EmitStoreToStack(d230, int32(bbs[4].PhiBase)+int32(32))
		ctx.StabilizeDescForControlFlow(&d230)
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d231 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d231)
		}
		if d231.Loc == scm.LocImm {
			d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: d231.Type, Imm: scm.NewInt(int64(uint64(d231.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d231.Reg, 32)
			ctx.EmitShrRegImm8(d231.Reg, 32)
		}
		if d231.Loc == scm.LocReg && d1.Loc == scm.LocReg && d231.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d231)
		ctx.EmitStoreToStack(d231, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d231)
		if ps.General {
			ctx.SyncDesc(&d2)
			if d2.Loc == scm.LocReg {
				ctx.ProtectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.ProtectReg(d2.Reg)
				ctx.ProtectReg(d2.Reg2)
			}
			d232 = d2
			if d232.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d232)
			d233 = d232
			if d233.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: d233.Type, Imm: scm.NewInt(int64(uint64(d233.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d233.Reg, 32)
				ctx.EmitShrRegImm8(d233.Reg, 32)
			}
			ctx.EmitStoreToStack(d233, int32(bbs[4].PhiBase)+int32(16))
			if d2.Loc == scm.LocReg {
				ctx.UnprotectReg(d2.Reg)
			} else if d2.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d2.Reg)
				ctx.UnprotectReg(d2.Reg2)
			}
		}
		ps234 := scm.PhiState{General: ps.General}
		ps234.OverlayValues = make([]scm.JITValueDesc, 234)
		ps234.OverlayValues[1] = d1
		ps234.OverlayValues[2] = d2
		ps234.OverlayValues[3] = d3
		ps234.OverlayValues[4] = d4
		ps234.OverlayValues[5] = d5
		ps234.OverlayValues[6] = d6
		ps234.OverlayValues[7] = d7
		ps234.OverlayValues[8] = d8
		ps234.OverlayValues[9] = d9
		ps234.OverlayValues[10] = d10
		ps234.OverlayValues[11] = d11
		ps234.OverlayValues[12] = d12
		ps234.OverlayValues[13] = d13
		ps234.OverlayValues[14] = d14
		ps234.OverlayValues[15] = d15
		ps234.OverlayValues[17] = d17
		ps234.OverlayValues[18] = d18
		ps234.OverlayValues[19] = d19
		ps234.OverlayValues[20] = d20
		ps234.OverlayValues[21] = d21
		ps234.OverlayValues[22] = d22
		ps234.OverlayValues[23] = d23
		ps234.OverlayValues[24] = d24
		ps234.OverlayValues[25] = d25
		ps234.OverlayValues[26] = d26
		ps234.OverlayValues[27] = d27
		ps234.OverlayValues[28] = d28
		ps234.OverlayValues[29] = d29
		ps234.OverlayValues[30] = d30
		ps234.OverlayValues[31] = d31
		ps234.OverlayValues[32] = d32
		ps234.OverlayValues[33] = d33
		ps234.OverlayValues[34] = d34
		ps234.OverlayValues[35] = d35
		ps234.OverlayValues[36] = d36
		ps234.OverlayValues[37] = d37
		ps234.OverlayValues[38] = d38
		ps234.OverlayValues[39] = d39
		ps234.OverlayValues[40] = d40
		ps234.OverlayValues[41] = d41
		ps234.OverlayValues[42] = d42
		ps234.OverlayValues[43] = d43
		ps234.OverlayValues[44] = d44
		ps234.OverlayValues[45] = d45
		ps234.OverlayValues[46] = d46
		ps234.OverlayValues[47] = d47
		ps234.OverlayValues[48] = d48
		ps234.OverlayValues[49] = d49
		ps234.OverlayValues[50] = d50
		ps234.OverlayValues[53] = d53
		ps234.OverlayValues[54] = d54
		ps234.OverlayValues[55] = d55
		ps234.OverlayValues[111] = d111
		ps234.OverlayValues[112] = d112
		ps234.OverlayValues[113] = d113
		ps234.OverlayValues[114] = d114
		ps234.OverlayValues[115] = d115
		ps234.OverlayValues[116] = d116
		ps234.OverlayValues[117] = d117
		ps234.OverlayValues[118] = d118
		ps234.OverlayValues[119] = d119
		ps234.OverlayValues[120] = d120
		ps234.OverlayValues[121] = d121
		ps234.OverlayValues[122] = d122
		ps234.OverlayValues[123] = d123
		ps234.OverlayValues[124] = d124
		ps234.OverlayValues[125] = d125
		ps234.OverlayValues[126] = d126
		ps234.OverlayValues[127] = d127
		ps234.OverlayValues[128] = d128
		ps234.OverlayValues[129] = d129
		ps234.OverlayValues[130] = d130
		ps234.OverlayValues[131] = d131
		ps234.OverlayValues[132] = d132
		ps234.OverlayValues[133] = d133
		ps234.OverlayValues[134] = d134
		ps234.OverlayValues[135] = d135
		ps234.OverlayValues[136] = d136
		ps234.OverlayValues[137] = d137
		ps234.OverlayValues[138] = d138
		ps234.OverlayValues[139] = d139
		ps234.OverlayValues[140] = d140
		ps234.OverlayValues[143] = d143
		ps234.OverlayValues[230] = d230
		ps234.OverlayValues[231] = d231
		ps234.OverlayValues[232] = d232
		ps234.OverlayValues[233] = d233
		ps234.PhiValues = make([]scm.JITValueDesc, 3)
		d235 = d2
		ps234.PhiValues[1] = d235
		if ps234.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps234)
		return result
	}
	bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d236 := ps.PhiValues[0]
				ctx.EnsureDesc(&d236)
				ctx.EmitStoreToStack(d236, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d237 := ps.PhiValues[1]
				ctx.EnsureDesc(&d237)
				ctx.EmitStoreToStack(d237, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d238 := ps.PhiValues[2]
				ctx.EnsureDesc(&d238)
				ctx.EmitStoreToStack(d238, int32(bbs[4].PhiBase)+int32(32))
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
		}
		if len(ps.OverlayValues) > 238 && ps.OverlayValues[238].Loc != scm.LocNone {
			d238 = ps.OverlayValues[238]
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
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d5)
		ctx.StabilizeDescForControlFlow(&d6)
		ctx.StabilizeDescForControlFlow(&d7)
		ctx.EnsureDesc(&d6)
		ctx.EnsureDesc(&d7)
		ctx.EnsureDescsTogether(&d6, &d7)
		var d239 scm.JITValueDesc
		if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
			d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d6.Imm.Int()) == uint64(d7.Imm.Int()))}
		} else if d7.Loc == scm.LocImm {
			r67 := ctx.AllocRegExcept(d6.Reg)
			if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d6.Reg, int32(d7.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitCmpInt64(d6.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r67, scm.CondEqual)
			d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r67}
			ctx.BindReg(r67, &d239)
		} else if d6.Loc == scm.LocImm {
			r68 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d7.Reg)
			ctx.EmitSetcc(r68, scm.CondEqual)
			d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r68}
			ctx.BindReg(r68, &d239)
		} else {
			r69 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitCmpInt64(d6.Reg, d7.Reg)
			ctx.EmitSetcc(r69, scm.CondEqual)
			d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r69}
			ctx.BindReg(r69, &d239)
		}
		d240 = d239
		ctx.EnsureDesc(&d240)
		if d240.Loc != scm.LocImm && d240.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d240.Loc == scm.LocImm {
			if d240.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d6)
					if d6.Loc == scm.LocReg {
						ctx.ProtectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.ProtectReg(d6.Reg)
						ctx.ProtectReg(d6.Reg2)
					}
					d241 = d6
					if d241.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d241)
					d242 = d241
					if d242.Loc == scm.LocImm {
						d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: d242.Type, Imm: scm.NewInt(int64(uint64(d242.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d242.Reg, 32)
						ctx.EmitShrRegImm8(d242.Reg, 32)
					}
					ctx.EmitStoreToStack(d242, int32(bbs[2].PhiBase)+int32(0))
					if d6.Loc == scm.LocReg {
						ctx.UnprotectReg(d6.Reg)
					} else if d6.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d6.Reg)
						ctx.UnprotectReg(d6.Reg2)
					}
				}
				ps243 := scm.PhiState{General: ps.General}
				ps243.OverlayValues = make([]scm.JITValueDesc, 243)
				ps243.OverlayValues[1] = d1
				ps243.OverlayValues[2] = d2
				ps243.OverlayValues[3] = d3
				ps243.OverlayValues[4] = d4
				ps243.OverlayValues[5] = d5
				ps243.OverlayValues[6] = d6
				ps243.OverlayValues[7] = d7
				ps243.OverlayValues[8] = d8
				ps243.OverlayValues[9] = d9
				ps243.OverlayValues[10] = d10
				ps243.OverlayValues[11] = d11
				ps243.OverlayValues[12] = d12
				ps243.OverlayValues[13] = d13
				ps243.OverlayValues[14] = d14
				ps243.OverlayValues[15] = d15
				ps243.OverlayValues[17] = d17
				ps243.OverlayValues[18] = d18
				ps243.OverlayValues[19] = d19
				ps243.OverlayValues[20] = d20
				ps243.OverlayValues[21] = d21
				ps243.OverlayValues[22] = d22
				ps243.OverlayValues[23] = d23
				ps243.OverlayValues[24] = d24
				ps243.OverlayValues[25] = d25
				ps243.OverlayValues[26] = d26
				ps243.OverlayValues[27] = d27
				ps243.OverlayValues[28] = d28
				ps243.OverlayValues[29] = d29
				ps243.OverlayValues[30] = d30
				ps243.OverlayValues[31] = d31
				ps243.OverlayValues[32] = d32
				ps243.OverlayValues[33] = d33
				ps243.OverlayValues[34] = d34
				ps243.OverlayValues[35] = d35
				ps243.OverlayValues[36] = d36
				ps243.OverlayValues[37] = d37
				ps243.OverlayValues[38] = d38
				ps243.OverlayValues[39] = d39
				ps243.OverlayValues[40] = d40
				ps243.OverlayValues[41] = d41
				ps243.OverlayValues[42] = d42
				ps243.OverlayValues[43] = d43
				ps243.OverlayValues[44] = d44
				ps243.OverlayValues[45] = d45
				ps243.OverlayValues[46] = d46
				ps243.OverlayValues[47] = d47
				ps243.OverlayValues[48] = d48
				ps243.OverlayValues[49] = d49
				ps243.OverlayValues[50] = d50
				ps243.OverlayValues[53] = d53
				ps243.OverlayValues[54] = d54
				ps243.OverlayValues[55] = d55
				ps243.OverlayValues[111] = d111
				ps243.OverlayValues[112] = d112
				ps243.OverlayValues[113] = d113
				ps243.OverlayValues[114] = d114
				ps243.OverlayValues[115] = d115
				ps243.OverlayValues[116] = d116
				ps243.OverlayValues[117] = d117
				ps243.OverlayValues[118] = d118
				ps243.OverlayValues[119] = d119
				ps243.OverlayValues[120] = d120
				ps243.OverlayValues[121] = d121
				ps243.OverlayValues[122] = d122
				ps243.OverlayValues[123] = d123
				ps243.OverlayValues[124] = d124
				ps243.OverlayValues[125] = d125
				ps243.OverlayValues[126] = d126
				ps243.OverlayValues[127] = d127
				ps243.OverlayValues[128] = d128
				ps243.OverlayValues[129] = d129
				ps243.OverlayValues[130] = d130
				ps243.OverlayValues[131] = d131
				ps243.OverlayValues[132] = d132
				ps243.OverlayValues[133] = d133
				ps243.OverlayValues[134] = d134
				ps243.OverlayValues[135] = d135
				ps243.OverlayValues[136] = d136
				ps243.OverlayValues[137] = d137
				ps243.OverlayValues[138] = d138
				ps243.OverlayValues[139] = d139
				ps243.OverlayValues[140] = d140
				ps243.OverlayValues[143] = d143
				ps243.OverlayValues[230] = d230
				ps243.OverlayValues[231] = d231
				ps243.OverlayValues[232] = d232
				ps243.OverlayValues[233] = d233
				ps243.OverlayValues[235] = d235
				ps243.OverlayValues[236] = d236
				ps243.OverlayValues[237] = d237
				ps243.OverlayValues[238] = d238
				ps243.OverlayValues[239] = d239
				ps243.OverlayValues[240] = d240
				ps243.OverlayValues[241] = d241
				ps243.OverlayValues[242] = d242
				ps243.PhiValues = make([]scm.JITValueDesc, 1)
				d244 = d6
				ps243.PhiValues[0] = d244
				return bbs[2].RenderPS(ps243)
			}
			if ps.General {
			}
			ps245 := scm.PhiState{General: ps.General}
			ps245.OverlayValues = make([]scm.JITValueDesc, 245)
			ps245.OverlayValues[1] = d1
			ps245.OverlayValues[2] = d2
			ps245.OverlayValues[3] = d3
			ps245.OverlayValues[4] = d4
			ps245.OverlayValues[5] = d5
			ps245.OverlayValues[6] = d6
			ps245.OverlayValues[7] = d7
			ps245.OverlayValues[8] = d8
			ps245.OverlayValues[9] = d9
			ps245.OverlayValues[10] = d10
			ps245.OverlayValues[11] = d11
			ps245.OverlayValues[12] = d12
			ps245.OverlayValues[13] = d13
			ps245.OverlayValues[14] = d14
			ps245.OverlayValues[15] = d15
			ps245.OverlayValues[17] = d17
			ps245.OverlayValues[18] = d18
			ps245.OverlayValues[19] = d19
			ps245.OverlayValues[20] = d20
			ps245.OverlayValues[21] = d21
			ps245.OverlayValues[22] = d22
			ps245.OverlayValues[23] = d23
			ps245.OverlayValues[24] = d24
			ps245.OverlayValues[25] = d25
			ps245.OverlayValues[26] = d26
			ps245.OverlayValues[27] = d27
			ps245.OverlayValues[28] = d28
			ps245.OverlayValues[29] = d29
			ps245.OverlayValues[30] = d30
			ps245.OverlayValues[31] = d31
			ps245.OverlayValues[32] = d32
			ps245.OverlayValues[33] = d33
			ps245.OverlayValues[34] = d34
			ps245.OverlayValues[35] = d35
			ps245.OverlayValues[36] = d36
			ps245.OverlayValues[37] = d37
			ps245.OverlayValues[38] = d38
			ps245.OverlayValues[39] = d39
			ps245.OverlayValues[40] = d40
			ps245.OverlayValues[41] = d41
			ps245.OverlayValues[42] = d42
			ps245.OverlayValues[43] = d43
			ps245.OverlayValues[44] = d44
			ps245.OverlayValues[45] = d45
			ps245.OverlayValues[46] = d46
			ps245.OverlayValues[47] = d47
			ps245.OverlayValues[48] = d48
			ps245.OverlayValues[49] = d49
			ps245.OverlayValues[50] = d50
			ps245.OverlayValues[53] = d53
			ps245.OverlayValues[54] = d54
			ps245.OverlayValues[55] = d55
			ps245.OverlayValues[111] = d111
			ps245.OverlayValues[112] = d112
			ps245.OverlayValues[113] = d113
			ps245.OverlayValues[114] = d114
			ps245.OverlayValues[115] = d115
			ps245.OverlayValues[116] = d116
			ps245.OverlayValues[117] = d117
			ps245.OverlayValues[118] = d118
			ps245.OverlayValues[119] = d119
			ps245.OverlayValues[120] = d120
			ps245.OverlayValues[121] = d121
			ps245.OverlayValues[122] = d122
			ps245.OverlayValues[123] = d123
			ps245.OverlayValues[124] = d124
			ps245.OverlayValues[125] = d125
			ps245.OverlayValues[126] = d126
			ps245.OverlayValues[127] = d127
			ps245.OverlayValues[128] = d128
			ps245.OverlayValues[129] = d129
			ps245.OverlayValues[130] = d130
			ps245.OverlayValues[131] = d131
			ps245.OverlayValues[132] = d132
			ps245.OverlayValues[133] = d133
			ps245.OverlayValues[134] = d134
			ps245.OverlayValues[135] = d135
			ps245.OverlayValues[136] = d136
			ps245.OverlayValues[137] = d137
			ps245.OverlayValues[138] = d138
			ps245.OverlayValues[139] = d139
			ps245.OverlayValues[140] = d140
			ps245.OverlayValues[143] = d143
			ps245.OverlayValues[230] = d230
			ps245.OverlayValues[231] = d231
			ps245.OverlayValues[232] = d232
			ps245.OverlayValues[233] = d233
			ps245.OverlayValues[235] = d235
			ps245.OverlayValues[236] = d236
			ps245.OverlayValues[237] = d237
			ps245.OverlayValues[238] = d238
			ps245.OverlayValues[239] = d239
			ps245.OverlayValues[240] = d240
			ps245.OverlayValues[241] = d241
			ps245.OverlayValues[242] = d242
			ps245.OverlayValues[244] = d244
			return bbs[6].RenderPS(ps245)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d246 := ps.PhiValues[0]
				ctx.EnsureDesc(&d246)
				ctx.EmitStoreToStack(d246, int32(bbs[4].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d247 := ps.PhiValues[1]
				ctx.EnsureDesc(&d247)
				ctx.EmitStoreToStack(d247, int32(bbs[4].PhiBase)+int32(16))
			}
			if len(ps.PhiValues) > 2 && ps.PhiValues[2].Loc != scm.LocNone {
				d248 := ps.PhiValues[2]
				ctx.EnsureDesc(&d248)
				ctx.EmitStoreToStack(d248, int32(bbs[4].PhiBase)+int32(32))
			}
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl21 := ctx.ReserveLabel()
		lbl22 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d240.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl21)
		ctx.EmitJmp(lbl22)
		ctx.MarkLabel(lbl21)
		ctx.SyncDesc(&d6)
		if d6.Loc == scm.LocReg {
			ctx.ProtectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.ProtectReg(d6.Reg)
			ctx.ProtectReg(d6.Reg2)
		}
		d249 = d6
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
		if d6.Loc == scm.LocReg {
			ctx.UnprotectReg(d6.Reg)
		} else if d6.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d6.Reg)
			ctx.UnprotectReg(d6.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl22)
		ctx.EmitJmp(lbl7)
		ps251 := scm.PhiState{General: true}
		ps251.OverlayValues = make([]scm.JITValueDesc, 251)
		ps251.OverlayValues[1] = d1
		ps251.OverlayValues[2] = d2
		ps251.OverlayValues[3] = d3
		ps251.OverlayValues[4] = d4
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
		ps251.OverlayValues[17] = d17
		ps251.OverlayValues[18] = d18
		ps251.OverlayValues[19] = d19
		ps251.OverlayValues[20] = d20
		ps251.OverlayValues[21] = d21
		ps251.OverlayValues[22] = d22
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
		ps251.OverlayValues[53] = d53
		ps251.OverlayValues[54] = d54
		ps251.OverlayValues[55] = d55
		ps251.OverlayValues[111] = d111
		ps251.OverlayValues[112] = d112
		ps251.OverlayValues[113] = d113
		ps251.OverlayValues[114] = d114
		ps251.OverlayValues[115] = d115
		ps251.OverlayValues[116] = d116
		ps251.OverlayValues[117] = d117
		ps251.OverlayValues[118] = d118
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
		ps251.OverlayValues[143] = d143
		ps251.OverlayValues[230] = d230
		ps251.OverlayValues[231] = d231
		ps251.OverlayValues[232] = d232
		ps251.OverlayValues[233] = d233
		ps251.OverlayValues[235] = d235
		ps251.OverlayValues[236] = d236
		ps251.OverlayValues[237] = d237
		ps251.OverlayValues[238] = d238
		ps251.OverlayValues[239] = d239
		ps251.OverlayValues[240] = d240
		ps251.OverlayValues[241] = d241
		ps251.OverlayValues[242] = d242
		ps251.OverlayValues[244] = d244
		ps251.OverlayValues[246] = d246
		ps251.OverlayValues[247] = d247
		ps251.OverlayValues[248] = d248
		ps251.OverlayValues[249] = d249
		ps251.OverlayValues[250] = d250
		ps251.PhiValues = make([]scm.JITValueDesc, 1)
		d253 = d6
		ps251.PhiValues[0] = d253
		ps252 := scm.PhiState{General: true}
		ps252.OverlayValues = make([]scm.JITValueDesc, 254)
		ps252.OverlayValues[1] = d1
		ps252.OverlayValues[2] = d2
		ps252.OverlayValues[3] = d3
		ps252.OverlayValues[4] = d4
		ps252.OverlayValues[5] = d5
		ps252.OverlayValues[6] = d6
		ps252.OverlayValues[7] = d7
		ps252.OverlayValues[8] = d8
		ps252.OverlayValues[9] = d9
		ps252.OverlayValues[10] = d10
		ps252.OverlayValues[11] = d11
		ps252.OverlayValues[12] = d12
		ps252.OverlayValues[13] = d13
		ps252.OverlayValues[14] = d14
		ps252.OverlayValues[15] = d15
		ps252.OverlayValues[17] = d17
		ps252.OverlayValues[18] = d18
		ps252.OverlayValues[19] = d19
		ps252.OverlayValues[20] = d20
		ps252.OverlayValues[21] = d21
		ps252.OverlayValues[22] = d22
		ps252.OverlayValues[23] = d23
		ps252.OverlayValues[24] = d24
		ps252.OverlayValues[25] = d25
		ps252.OverlayValues[26] = d26
		ps252.OverlayValues[27] = d27
		ps252.OverlayValues[28] = d28
		ps252.OverlayValues[29] = d29
		ps252.OverlayValues[30] = d30
		ps252.OverlayValues[31] = d31
		ps252.OverlayValues[32] = d32
		ps252.OverlayValues[33] = d33
		ps252.OverlayValues[34] = d34
		ps252.OverlayValues[35] = d35
		ps252.OverlayValues[36] = d36
		ps252.OverlayValues[37] = d37
		ps252.OverlayValues[38] = d38
		ps252.OverlayValues[39] = d39
		ps252.OverlayValues[40] = d40
		ps252.OverlayValues[41] = d41
		ps252.OverlayValues[42] = d42
		ps252.OverlayValues[43] = d43
		ps252.OverlayValues[44] = d44
		ps252.OverlayValues[45] = d45
		ps252.OverlayValues[46] = d46
		ps252.OverlayValues[47] = d47
		ps252.OverlayValues[48] = d48
		ps252.OverlayValues[49] = d49
		ps252.OverlayValues[50] = d50
		ps252.OverlayValues[53] = d53
		ps252.OverlayValues[54] = d54
		ps252.OverlayValues[55] = d55
		ps252.OverlayValues[111] = d111
		ps252.OverlayValues[112] = d112
		ps252.OverlayValues[113] = d113
		ps252.OverlayValues[114] = d114
		ps252.OverlayValues[115] = d115
		ps252.OverlayValues[116] = d116
		ps252.OverlayValues[117] = d117
		ps252.OverlayValues[118] = d118
		ps252.OverlayValues[119] = d119
		ps252.OverlayValues[120] = d120
		ps252.OverlayValues[121] = d121
		ps252.OverlayValues[122] = d122
		ps252.OverlayValues[123] = d123
		ps252.OverlayValues[124] = d124
		ps252.OverlayValues[125] = d125
		ps252.OverlayValues[126] = d126
		ps252.OverlayValues[127] = d127
		ps252.OverlayValues[128] = d128
		ps252.OverlayValues[129] = d129
		ps252.OverlayValues[130] = d130
		ps252.OverlayValues[131] = d131
		ps252.OverlayValues[132] = d132
		ps252.OverlayValues[133] = d133
		ps252.OverlayValues[134] = d134
		ps252.OverlayValues[135] = d135
		ps252.OverlayValues[136] = d136
		ps252.OverlayValues[137] = d137
		ps252.OverlayValues[138] = d138
		ps252.OverlayValues[139] = d139
		ps252.OverlayValues[140] = d140
		ps252.OverlayValues[143] = d143
		ps252.OverlayValues[230] = d230
		ps252.OverlayValues[231] = d231
		ps252.OverlayValues[232] = d232
		ps252.OverlayValues[233] = d233
		ps252.OverlayValues[235] = d235
		ps252.OverlayValues[236] = d236
		ps252.OverlayValues[237] = d237
		ps252.OverlayValues[238] = d238
		ps252.OverlayValues[239] = d239
		ps252.OverlayValues[240] = d240
		ps252.OverlayValues[241] = d241
		ps252.OverlayValues[242] = d242
		ps252.OverlayValues[244] = d244
		ps252.OverlayValues[246] = d246
		ps252.OverlayValues[247] = d247
		ps252.OverlayValues[248] = d248
		ps252.OverlayValues[249] = d249
		ps252.OverlayValues[250] = d250
		ps252.OverlayValues[253] = d253
		snap254 := d1
		snap255 := d2
		snap256 := d3
		snap257 := d4
		snap258 := d5
		snap259 := d6
		snap260 := d7
		snap261 := d8
		snap262 := d9
		snap263 := d10
		snap264 := d11
		snap265 := d12
		snap266 := d13
		snap267 := d14
		snap268 := d15
		snap269 := d17
		snap270 := d18
		snap271 := d19
		snap272 := d20
		snap273 := d21
		snap274 := d22
		snap275 := d23
		snap276 := d24
		snap277 := d25
		snap278 := d26
		snap279 := d27
		snap280 := d28
		snap281 := d29
		snap282 := d30
		snap283 := d31
		snap284 := d32
		snap285 := d33
		snap286 := d34
		snap287 := d35
		snap288 := d36
		snap289 := d37
		snap290 := d38
		snap291 := d39
		snap292 := d40
		snap293 := d41
		snap294 := d42
		snap295 := d43
		snap296 := d44
		snap297 := d45
		snap298 := d46
		snap299 := d47
		snap300 := d48
		snap301 := d49
		snap302 := d50
		snap303 := d53
		snap304 := d54
		snap305 := d55
		snap306 := d111
		snap307 := d112
		snap308 := d113
		snap309 := d114
		snap310 := d115
		snap311 := d116
		snap312 := d117
		snap313 := d118
		snap314 := d119
		snap315 := d120
		snap316 := d121
		snap317 := d122
		snap318 := d123
		snap319 := d124
		snap320 := d125
		snap321 := d126
		snap322 := d127
		snap323 := d128
		snap324 := d129
		snap325 := d130
		snap326 := d131
		snap327 := d132
		snap328 := d133
		snap329 := d134
		snap330 := d135
		snap331 := d136
		snap332 := d137
		snap333 := d138
		snap334 := d139
		snap335 := d140
		snap336 := d143
		snap337 := d230
		snap338 := d231
		snap339 := d232
		snap340 := d233
		snap341 := d235
		snap342 := d236
		snap343 := d237
		snap344 := d238
		snap345 := d239
		snap346 := d240
		snap347 := d241
		snap348 := d242
		snap349 := d244
		snap350 := d246
		snap351 := d247
		snap352 := d248
		snap353 := d249
		snap354 := d250
		snap355 := d253
		alloc356 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps251)
		}
		ctx.RestoreAllocState(alloc356)
		d1 = snap254
		d2 = snap255
		d3 = snap256
		d4 = snap257
		d5 = snap258
		d6 = snap259
		d7 = snap260
		d8 = snap261
		d9 = snap262
		d10 = snap263
		d11 = snap264
		d12 = snap265
		d13 = snap266
		d14 = snap267
		d15 = snap268
		d17 = snap269
		d18 = snap270
		d19 = snap271
		d20 = snap272
		d21 = snap273
		d22 = snap274
		d23 = snap275
		d24 = snap276
		d25 = snap277
		d26 = snap278
		d27 = snap279
		d28 = snap280
		d29 = snap281
		d30 = snap282
		d31 = snap283
		d32 = snap284
		d33 = snap285
		d34 = snap286
		d35 = snap287
		d36 = snap288
		d37 = snap289
		d38 = snap290
		d39 = snap291
		d40 = snap292
		d41 = snap293
		d42 = snap294
		d43 = snap295
		d44 = snap296
		d45 = snap297
		d46 = snap298
		d47 = snap299
		d48 = snap300
		d49 = snap301
		d50 = snap302
		d53 = snap303
		d54 = snap304
		d55 = snap305
		d111 = snap306
		d112 = snap307
		d113 = snap308
		d114 = snap309
		d115 = snap310
		d116 = snap311
		d117 = snap312
		d118 = snap313
		d119 = snap314
		d120 = snap315
		d121 = snap316
		d122 = snap317
		d123 = snap318
		d124 = snap319
		d125 = snap320
		d126 = snap321
		d127 = snap322
		d128 = snap323
		d129 = snap324
		d130 = snap325
		d131 = snap326
		d132 = snap327
		d133 = snap328
		d134 = snap329
		d135 = snap330
		d136 = snap331
		d137 = snap332
		d138 = snap333
		d139 = snap334
		d140 = snap335
		d143 = snap336
		d230 = snap337
		d231 = snap338
		d232 = snap339
		d233 = snap340
		d235 = snap341
		d236 = snap342
		d237 = snap343
		d238 = snap344
		d239 = snap345
		d240 = snap346
		d241 = snap347
		d242 = snap348
		d244 = snap349
		d246 = snap350
		d247 = snap351
		d248 = snap352
		d249 = snap353
		d250 = snap354
		d253 = snap355
		if !bbs[6].Rendered {
			return bbs[6].RenderPS(ps252)
		}
		return result
		ctx.FreeDesc(&d239)
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d357 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d357 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d357 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d357)
		}
		if d357.Loc == scm.LocImm {
			d357 = scm.JITValueDesc{Loc: scm.LocImm, Type: d357.Type, Imm: scm.NewInt(int64(uint64(d357.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d357.Reg, 32)
			ctx.EmitShrRegImm8(d357.Reg, 32)
		}
		if d357.Loc == scm.LocReg && d1.Loc == scm.LocReg && d357.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d357)
		ctx.EmitStoreToStack(d357, int32(bbs[4].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d357)
		if ps.General {
			ctx.SyncDesc(&d1)
			if d1.Loc == scm.LocReg {
				ctx.ProtectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1.Reg)
				ctx.ProtectReg(d1.Reg2)
			}
			ctx.SyncDesc(&d3)
			if d3.Loc == scm.LocReg {
				ctx.ProtectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.ProtectReg(d3.Reg)
				ctx.ProtectReg(d3.Reg2)
			}
			d358 = d1
			if d358.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d358)
			d359 = d358
			if d359.Loc == scm.LocImm {
				d359 = scm.JITValueDesc{Loc: scm.LocImm, Type: d359.Type, Imm: scm.NewInt(int64(uint64(d359.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d359.Reg, 32)
				ctx.EmitShrRegImm8(d359.Reg, 32)
			}
			ctx.EmitStoreToStack(d359, int32(bbs[4].PhiBase)+int32(16))
			d360 = d3
			if d360.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d360)
			d361 = d360
			if d361.Loc == scm.LocImm {
				d361 = scm.JITValueDesc{Loc: scm.LocImm, Type: d361.Type, Imm: scm.NewInt(int64(uint64(d361.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d361.Reg, 32)
				ctx.EmitShrRegImm8(d361.Reg, 32)
			}
			ctx.EmitStoreToStack(d361, int32(bbs[4].PhiBase)+int32(32))
			if d1.Loc == scm.LocReg {
				ctx.UnprotectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1.Reg)
				ctx.UnprotectReg(d1.Reg2)
			}
			if d3.Loc == scm.LocReg {
				ctx.UnprotectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d3.Reg)
				ctx.UnprotectReg(d3.Reg2)
			}
		}
		ps362 := scm.PhiState{General: ps.General}
		ps362.OverlayValues = make([]scm.JITValueDesc, 362)
		ps362.OverlayValues[1] = d1
		ps362.OverlayValues[2] = d2
		ps362.OverlayValues[3] = d3
		ps362.OverlayValues[4] = d4
		ps362.OverlayValues[5] = d5
		ps362.OverlayValues[6] = d6
		ps362.OverlayValues[7] = d7
		ps362.OverlayValues[8] = d8
		ps362.OverlayValues[9] = d9
		ps362.OverlayValues[10] = d10
		ps362.OverlayValues[11] = d11
		ps362.OverlayValues[12] = d12
		ps362.OverlayValues[13] = d13
		ps362.OverlayValues[14] = d14
		ps362.OverlayValues[15] = d15
		ps362.OverlayValues[17] = d17
		ps362.OverlayValues[18] = d18
		ps362.OverlayValues[19] = d19
		ps362.OverlayValues[20] = d20
		ps362.OverlayValues[21] = d21
		ps362.OverlayValues[22] = d22
		ps362.OverlayValues[23] = d23
		ps362.OverlayValues[24] = d24
		ps362.OverlayValues[25] = d25
		ps362.OverlayValues[26] = d26
		ps362.OverlayValues[27] = d27
		ps362.OverlayValues[28] = d28
		ps362.OverlayValues[29] = d29
		ps362.OverlayValues[30] = d30
		ps362.OverlayValues[31] = d31
		ps362.OverlayValues[32] = d32
		ps362.OverlayValues[33] = d33
		ps362.OverlayValues[34] = d34
		ps362.OverlayValues[35] = d35
		ps362.OverlayValues[36] = d36
		ps362.OverlayValues[37] = d37
		ps362.OverlayValues[38] = d38
		ps362.OverlayValues[39] = d39
		ps362.OverlayValues[40] = d40
		ps362.OverlayValues[41] = d41
		ps362.OverlayValues[42] = d42
		ps362.OverlayValues[43] = d43
		ps362.OverlayValues[44] = d44
		ps362.OverlayValues[45] = d45
		ps362.OverlayValues[46] = d46
		ps362.OverlayValues[47] = d47
		ps362.OverlayValues[48] = d48
		ps362.OverlayValues[49] = d49
		ps362.OverlayValues[50] = d50
		ps362.OverlayValues[53] = d53
		ps362.OverlayValues[54] = d54
		ps362.OverlayValues[55] = d55
		ps362.OverlayValues[111] = d111
		ps362.OverlayValues[112] = d112
		ps362.OverlayValues[113] = d113
		ps362.OverlayValues[114] = d114
		ps362.OverlayValues[115] = d115
		ps362.OverlayValues[116] = d116
		ps362.OverlayValues[117] = d117
		ps362.OverlayValues[118] = d118
		ps362.OverlayValues[119] = d119
		ps362.OverlayValues[120] = d120
		ps362.OverlayValues[121] = d121
		ps362.OverlayValues[122] = d122
		ps362.OverlayValues[123] = d123
		ps362.OverlayValues[124] = d124
		ps362.OverlayValues[125] = d125
		ps362.OverlayValues[126] = d126
		ps362.OverlayValues[127] = d127
		ps362.OverlayValues[128] = d128
		ps362.OverlayValues[129] = d129
		ps362.OverlayValues[130] = d130
		ps362.OverlayValues[131] = d131
		ps362.OverlayValues[132] = d132
		ps362.OverlayValues[133] = d133
		ps362.OverlayValues[134] = d134
		ps362.OverlayValues[135] = d135
		ps362.OverlayValues[136] = d136
		ps362.OverlayValues[137] = d137
		ps362.OverlayValues[138] = d138
		ps362.OverlayValues[139] = d139
		ps362.OverlayValues[140] = d140
		ps362.OverlayValues[143] = d143
		ps362.OverlayValues[230] = d230
		ps362.OverlayValues[231] = d231
		ps362.OverlayValues[232] = d232
		ps362.OverlayValues[233] = d233
		ps362.OverlayValues[235] = d235
		ps362.OverlayValues[236] = d236
		ps362.OverlayValues[237] = d237
		ps362.OverlayValues[238] = d238
		ps362.OverlayValues[239] = d239
		ps362.OverlayValues[240] = d240
		ps362.OverlayValues[241] = d241
		ps362.OverlayValues[242] = d242
		ps362.OverlayValues[244] = d244
		ps362.OverlayValues[246] = d246
		ps362.OverlayValues[247] = d247
		ps362.OverlayValues[248] = d248
		ps362.OverlayValues[249] = d249
		ps362.OverlayValues[250] = d250
		ps362.OverlayValues[253] = d253
		ps362.OverlayValues[357] = d357
		ps362.OverlayValues[358] = d358
		ps362.OverlayValues[359] = d359
		ps362.OverlayValues[360] = d360
		ps362.OverlayValues[361] = d361
		ps362.PhiValues = make([]scm.JITValueDesc, 3)
		d363 = d1
		ps362.PhiValues[1] = d363
		d364 = d3
		ps362.PhiValues[2] = d364
		if ps362.General && bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		return bbs[4].RenderPS(ps362)
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		d365 = d5
		_ = d365
		ctx.StabilizeDescForControlFlow(&d365)
		ctx.StabilizeDescForControlFlow(&d5)
		bbpos_3_0 := int32(-1)
		_ = bbpos_3_0
		lbl23 := ctx.ReserveLabel()
		_ = lbl23
		bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl23)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d365)
		ctx.EnsureDesc(&d365)
		var d366 scm.JITValueDesc
		if d365.Loc == scm.LocImm {
			d366 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d365.Imm.Int()))))}
		} else {
			r70 := ctx.AllocReg()
			ctx.EmitMovRegReg(r70, d365.Reg)
			ctx.EmitShlRegImm8(r70, 32)
			ctx.EmitShrRegImm8(r70, 32)
			d366 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			ctx.BindReg(r70, &d366)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d367 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d367 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r71 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r71, thisptr.Reg, off)
			d367 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
			ctx.BindReg(r71, &d367)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d367)
		ctx.EnsureDesc(&d367)
		var d368 scm.JITValueDesc
		if d367.Loc == scm.LocImm {
			d368 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d367.Imm.Int()))))}
		} else {
			r72 := ctx.AllocReg()
			ctx.EmitMovRegReg(r72, d367.Reg)
			ctx.EmitShlRegImm8(r72, 56)
			ctx.EmitShrRegImm8(r72, 56)
			d368 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			ctx.BindReg(r72, &d368)
		}
		ctx.FreeDesc(&d367)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d366)
		ctx.EnsureDesc(&d368)
		ctx.EnsureDescsTogether(&d366, &d368)
		var d369 scm.JITValueDesc
		if d366.Loc == scm.LocImm && d368.Loc == scm.LocImm {
			d369 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d366.Imm.Int() * d368.Imm.Int())}
		} else if d366.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d368.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d366.Imm.Int()))
			ctx.EmitImulInt64(scratch, d368.Reg)
			d369 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d369)
		} else if d368.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d366.Reg)
			ctx.EmitMovRegReg(scratch, d366.Reg)
			if d368.Imm.Int() >= -2147483648 && d368.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d368.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d368.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d369 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d369)
		} else {
			r73 := ctx.AllocRegExcept(d366.Reg, d368.Reg)
			ctx.EmitMovRegReg(r73, d366.Reg)
			ctx.EmitImulInt64(r73, d368.Reg)
			d369 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			ctx.BindReg(r73, &d369)
		}
		if d369.Loc == scm.LocReg && d366.Loc == scm.LocReg && d369.Reg == d366.Reg {
			ctx.TransferReg(d366.Reg)
			d366.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d366)
		ctx.FreeDesc(&d368)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d369)
		var d370 scm.JITValueDesc
		if d369.Loc == scm.LocImm {
			d370 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d369.Imm.Int() / 64)}
		} else {
			r74 := ctx.AllocRegExcept(d369.Reg)
			ctx.EmitMovRegReg(r74, d369.Reg)
			ctx.EmitShrRegImm8(r74, 6)
			d370 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			ctx.BindReg(r74, &d370)
		}
		if d370.Loc == scm.LocReg && d369.Loc == scm.LocReg && d370.Reg == d369.Reg {
			ctx.TransferReg(d369.Reg)
			d369.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d369)
		var d371 scm.JITValueDesc
		if d369.Loc == scm.LocImm {
			d371 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d369.Imm.Int() % 64)}
		} else {
			r75 := ctx.AllocRegExcept(d369.Reg)
			ctx.EmitMovRegReg(r75, d369.Reg)
			ctx.EmitAndRegImm32(r75, 63)
			d371 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			ctx.BindReg(r75, &d371)
		}
		if d371.Loc == scm.LocReg && d369.Loc == scm.LocReg && d371.Reg == d369.Reg {
			ctx.TransferReg(d369.Reg)
			d369.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d369)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d372 scm.JITValueDesc
		r76 := ctx.AllocReg()
		r77 := ctx.AllocRegExcept(r76)
		r78 := ctx.AllocRegExcept(r76, r77)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r76, uint64(dataPtr))
			ctx.EmitMovRegImm64(r77, uint64(sliceLen))
			ctx.EmitMovRegImm64(r78, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r76, thisptr.Reg, off)
			ctx.EmitMovRegMem(r77, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r78, thisptr.Reg, off+16)
		}
		d372 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r76, Reg2: r77, Reg3: r78}
		ctx.BindReg(r76, &d372)
		ctx.BindReg(r77, &d372)
		ctx.BindReg(r78, &d372)
		ctx.BindReg(r76, &d372)
		ctx.BindReg(r77, &d372)
		ctx.BindReg(r78, &d372)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d370)
		ctx.ReclaimUntrackedRegs()
		d374 = ctx.EmitSliceElementAddress(&d372, &d370, 8)
		ctx.EnsureDesc(&d374)
		ctx.EmitMovRegMem(d374.Reg, d374.Reg, 0)
		d373 = d374
		d373.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d373)
		ctx.EnsureDesc(&d371)
		ctx.EnsureDescsTogether(&d373, &d371)
		var d375 scm.JITValueDesc
		if d373.Loc == scm.LocImm && d371.Loc == scm.LocImm {
			d375 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d373.Imm.Int()) << uint64(d371.Imm.Int())))}
		} else if d371.Loc == scm.LocImm {
			r79 := ctx.AllocRegExcept(d373.Reg)
			ctx.EmitMovRegReg(r79, d373.Reg)
			ctx.EmitShlRegImm8(r79, uint8(d371.Imm.Int()))
			d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			ctx.BindReg(r79, &d375)
		} else {
			{
				shiftSrc := d373.Reg
				r80 := ctx.AllocRegExcept(d373.Reg, d371.Reg)
				ctx.EmitMovRegReg(r80, d373.Reg)
				shiftSrc = r80
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d371.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d371.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d371.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d375 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d375)
			}
		}
		if d375.Loc == scm.LocReg && d373.Loc == scm.LocReg && d375.Reg == d373.Reg {
			ctx.TransferReg(d373.Reg)
			d373.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d373)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d370)
		ctx.EnsureDesc(&d370)
		var d376 scm.JITValueDesc
		if d370.Loc == scm.LocImm {
			d376 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d370.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d370.Reg)
			ctx.EmitMovRegReg(scratch, d370.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d376 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d376)
		}
		if d376.Loc == scm.LocReg && d370.Loc == scm.LocReg && d376.Reg == d370.Reg {
			ctx.TransferReg(d370.Reg)
			d370.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d370)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d376)
		ctx.ReclaimUntrackedRegs()
		d378 = ctx.EmitSliceElementAddress(&d372, &d376, 8)
		ctx.EnsureDesc(&d378)
		ctx.EmitMovRegMem(d378.Reg, d378.Reg, 0)
		d377 = d378
		d377.Type = scm.TagInt
		ctx.FreeDesc(&d376)
		ctx.ReclaimUntrackedRegs()
		d379 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d371)
		ctx.EnsureDescsTogether(&d379, &d371)
		var d380 scm.JITValueDesc
		if d379.Loc == scm.LocImm && d371.Loc == scm.LocImm {
			d380 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d379.Imm.Int() - d371.Imm.Int())}
		} else if d371.Loc == scm.LocImm && d371.Imm.Int() == 0 {
			r81 := ctx.AllocRegExcept(d379.Reg)
			ctx.EmitMovRegReg(r81, d379.Reg)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			ctx.BindReg(r81, &d380)
		} else if d379.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d371.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d379.Imm.Int()))
			ctx.EmitSubInt64(scratch, d371.Reg)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d380)
		} else if d371.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d379.Reg)
			ctx.EmitMovRegReg(scratch, d379.Reg)
			if d371.Imm.Int() >= -2147483648 && d371.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d371.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d371.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d380)
		} else {
			r82 := ctx.AllocRegExcept(d379.Reg, d371.Reg)
			ctx.EmitMovRegReg(r82, d379.Reg)
			ctx.EmitSubInt64(r82, d371.Reg)
			d380 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			ctx.BindReg(r82, &d380)
		}
		if d380.Loc == scm.LocReg && d379.Loc == scm.LocReg && d380.Reg == d379.Reg {
			ctx.TransferReg(d379.Reg)
			d379.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d371)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d377)
		ctx.EnsureDesc(&d380)
		ctx.EnsureDescsTogether(&d377, &d380)
		var d381 scm.JITValueDesc
		if d377.Loc == scm.LocImm && d380.Loc == scm.LocImm {
			d381 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d377.Imm.Int()) >> uint64(d380.Imm.Int())))}
		} else if d380.Loc == scm.LocImm {
			r83 := ctx.AllocRegExcept(d377.Reg)
			ctx.EmitMovRegReg(r83, d377.Reg)
			ctx.EmitShrRegImm8(r83, uint8(d380.Imm.Int()))
			d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			ctx.BindReg(r83, &d381)
		} else {
			{
				shiftSrc := d377.Reg
				r84 := ctx.AllocRegExcept(d377.Reg, d380.Reg)
				ctx.EmitMovRegReg(r84, d377.Reg)
				shiftSrc = r84
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d380.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d380.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d380.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d381 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d381)
			}
		}
		if d381.Loc == scm.LocReg && d377.Loc == scm.LocReg && d381.Reg == d377.Reg {
			ctx.TransferReg(d377.Reg)
			d377.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d377)
		ctx.FreeDesc(&d380)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d375)
		ctx.EnsureDesc(&d381)
		var d382 scm.JITValueDesc
		if d375.Loc == scm.LocImm && d381.Loc == scm.LocImm {
			d382 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d375.Imm.Int() | d381.Imm.Int())}
		} else if d375.Loc == scm.LocImm && d375.Imm.Int() == 0 {
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d381.Reg}
			ctx.BindReg(d381.Reg, &d382)
		} else if d381.Loc == scm.LocImm && d381.Imm.Int() == 0 {
			r85 := ctx.AllocRegExcept(d375.Reg)
			ctx.EmitMovRegReg(r85, d375.Reg)
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			ctx.BindReg(r85, &d382)
		} else if d375.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d381.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d375.Imm.Int()))
			ctx.EmitOrInt64(scratch, d381.Reg)
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d382)
		} else if d381.Loc == scm.LocImm {
			r86 := ctx.AllocRegExcept(d375.Reg)
			ctx.EmitMovRegReg(r86, d375.Reg)
			if d381.Imm.Int() >= -2147483648 && d381.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r86, int32(d381.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d381.Imm.Int()))
				ctx.EmitOrInt64(r86, scm.RegR11)
			}
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			ctx.BindReg(r86, &d382)
		} else {
			r87 := ctx.AllocRegExcept(d375.Reg, d381.Reg)
			ctx.EmitMovRegReg(r87, d375.Reg)
			ctx.EmitOrInt64(r87, d381.Reg)
			d382 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			ctx.BindReg(r87, &d382)
		}
		if d382.Loc == scm.LocReg && d375.Loc == scm.LocReg && d382.Reg == d375.Reg {
			ctx.TransferReg(d375.Reg)
			d375.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d375)
		ctx.FreeDesc(&d381)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d383 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d383 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r88 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r88, thisptr.Reg, off)
			d383 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r88}
			ctx.BindReg(r88, &d383)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d383)
		ctx.EnsureDesc(&d383)
		var d384 scm.JITValueDesc
		if d383.Loc == scm.LocImm {
			d384 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d383.Imm.Int()))))}
		} else {
			r89 := ctx.AllocReg()
			ctx.EmitMovRegReg(r89, d383.Reg)
			ctx.EmitShlRegImm8(r89, 56)
			ctx.EmitShrRegImm8(r89, 56)
			d384 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			ctx.BindReg(r89, &d384)
		}
		ctx.FreeDesc(&d383)
		ctx.ReclaimUntrackedRegs()
		d385 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d384)
		ctx.EnsureDescsTogether(&d385, &d384)
		var d386 scm.JITValueDesc
		if d385.Loc == scm.LocImm && d384.Loc == scm.LocImm {
			d386 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d385.Imm.Int() - d384.Imm.Int())}
		} else if d384.Loc == scm.LocImm && d384.Imm.Int() == 0 {
			r90 := ctx.AllocRegExcept(d385.Reg)
			ctx.EmitMovRegReg(r90, d385.Reg)
			d386 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			ctx.BindReg(r90, &d386)
		} else if d385.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d384.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d385.Imm.Int()))
			ctx.EmitSubInt64(scratch, d384.Reg)
			d386 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d386)
		} else if d384.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d385.Reg)
			ctx.EmitMovRegReg(scratch, d385.Reg)
			if d384.Imm.Int() >= -2147483648 && d384.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d384.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d384.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d386 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d386)
		} else {
			r91 := ctx.AllocRegExcept(d385.Reg, d384.Reg)
			ctx.EmitMovRegReg(r91, d385.Reg)
			ctx.EmitSubInt64(r91, d384.Reg)
			d386 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
			ctx.BindReg(r91, &d386)
		}
		if d386.Loc == scm.LocReg && d385.Loc == scm.LocReg && d386.Reg == d385.Reg {
			ctx.TransferReg(d385.Reg)
			d385.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d384)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d382)
		ctx.EnsureDesc(&d386)
		ctx.EnsureDescsTogether(&d382, &d386)
		var d387 scm.JITValueDesc
		if d382.Loc == scm.LocImm && d386.Loc == scm.LocImm {
			d387 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d382.Imm.Int()) >> uint64(d386.Imm.Int())))}
		} else if d386.Loc == scm.LocImm {
			r92 := ctx.AllocRegExcept(d382.Reg)
			ctx.EmitMovRegReg(r92, d382.Reg)
			ctx.EmitShrRegImm8(r92, uint8(d386.Imm.Int()))
			d387 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
			ctx.BindReg(r92, &d387)
		} else {
			{
				shiftSrc := d382.Reg
				r93 := ctx.AllocRegExcept(d382.Reg, d386.Reg)
				ctx.EmitMovRegReg(r93, d382.Reg)
				shiftSrc = r93
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d386.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d386.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d386.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d387 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d387)
			}
		}
		if d387.Loc == scm.LocReg && d382.Loc == scm.LocReg && d387.Reg == d382.Reg {
			ctx.TransferReg(d382.Reg)
			d382.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d382)
		ctx.FreeDesc(&d386)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d387)
		ctx.EnsureDesc(&d387)
		ctx.EnsureDesc(&d387)
		var d388 scm.JITValueDesc
		if d387.Loc == scm.LocImm {
			d388 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d387.Imm.Int()))))}
		} else {
			r94 := ctx.AllocReg()
			ctx.EmitMovRegReg(r94, d387.Reg)
			d388 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			ctx.BindReg(r94, &d388)
		}
		ctx.FreeDesc(&d387)
		var d389 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d389 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r95 := ctx.AllocReg()
			ctx.EmitMovRegMem(r95, thisptr.Reg, off)
			d389 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
			ctx.BindReg(r95, &d389)
		}
		ctx.EnsureDesc(&d388)
		ctx.EnsureDesc(&d389)
		ctx.EnsureDescsTogether(&d388, &d389)
		var d390 scm.JITValueDesc
		if d388.Loc == scm.LocImm && d389.Loc == scm.LocImm {
			d390 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d388.Imm.Int() + d389.Imm.Int())}
		} else if d389.Loc == scm.LocImm && d389.Imm.Int() == 0 {
			r96 := ctx.AllocRegExcept(d388.Reg)
			ctx.EmitMovRegReg(r96, d388.Reg)
			d390 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			ctx.BindReg(r96, &d390)
		} else if d388.Loc == scm.LocImm && d388.Imm.Int() == 0 {
			d390 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d389.Reg}
			ctx.BindReg(d389.Reg, &d390)
		} else if d388.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d389.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d388.Imm.Int()))
			ctx.EmitAddInt64(scratch, d389.Reg)
			d390 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d390)
		} else if d389.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d388.Reg)
			ctx.EmitMovRegReg(scratch, d388.Reg)
			if d389.Imm.Int() >= -2147483648 && d389.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d389.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d389.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d390 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d390)
		} else {
			r97 := ctx.AllocRegExcept(d388.Reg, d389.Reg)
			ctx.EmitMovRegReg(r97, d388.Reg)
			ctx.EmitAddInt64(r97, d389.Reg)
			d390 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			ctx.BindReg(r97, &d390)
		}
		if d390.Loc == scm.LocReg && d388.Loc == scm.LocReg && d390.Reg == d388.Reg {
			ctx.TransferReg(d388.Reg)
			d388.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d388)
		ctx.FreeDesc(&d389)
		ctx.EnsureDesc(&d390)
		ctx.EnsureDesc(&d390)
		var d391 scm.JITValueDesc
		if d390.Loc == scm.LocImm {
			d391 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d390.Imm.Int()))))}
		} else {
			r98 := ctx.AllocReg()
			ctx.EmitMovRegReg(r98, d390.Reg)
			ctx.EmitShlRegImm8(r98, 32)
			ctx.EmitShrRegImm8(r98, 32)
			d391 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			ctx.BindReg(r98, &d391)
		}
		ctx.FreeDesc(&d390)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d391)
		ctx.EnsureDescsTogether(&idxInt, &d391)
		var d392 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm && d391.Loc == scm.LocImm {
			d392 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d391.Imm.Int()))}
		} else if d391.Loc == scm.LocImm {
			r99 := ctx.AllocRegExcept(idxInt.Reg)
			if d391.Imm.Int() >= -2147483648 && d391.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(idxInt.Reg, int32(d391.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d391.Imm.Int()))
				ctx.EmitCmpInt64(idxInt.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r99, scm.CondUnsignedBelow)
			d392 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r99}
			ctx.BindReg(r99, &d392)
		} else if idxInt.Loc == scm.LocImm {
			r100 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d391.Reg)
			ctx.EmitSetcc(r100, scm.CondUnsignedBelow)
			d392 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r100}
			ctx.BindReg(r100, &d392)
		} else {
			r101 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitCmpInt64(idxInt.Reg, d391.Reg)
			ctx.EmitSetcc(r101, scm.CondUnsignedBelow)
			d392 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r101}
			ctx.BindReg(r101, &d392)
		}
		ctx.FreeDesc(&d391)
		d393 = d392
		ctx.EnsureDesc(&d393)
		if d393.Loc != scm.LocImm && d393.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d393.Loc == scm.LocImm {
			if d393.Imm.Bool() {
				if ps.General {
				}
				ps394 := scm.PhiState{General: ps.General}
				ps394.OverlayValues = make([]scm.JITValueDesc, 394)
				ps394.OverlayValues[1] = d1
				ps394.OverlayValues[2] = d2
				ps394.OverlayValues[3] = d3
				ps394.OverlayValues[4] = d4
				ps394.OverlayValues[5] = d5
				ps394.OverlayValues[6] = d6
				ps394.OverlayValues[7] = d7
				ps394.OverlayValues[8] = d8
				ps394.OverlayValues[9] = d9
				ps394.OverlayValues[10] = d10
				ps394.OverlayValues[11] = d11
				ps394.OverlayValues[12] = d12
				ps394.OverlayValues[13] = d13
				ps394.OverlayValues[14] = d14
				ps394.OverlayValues[15] = d15
				ps394.OverlayValues[17] = d17
				ps394.OverlayValues[18] = d18
				ps394.OverlayValues[19] = d19
				ps394.OverlayValues[20] = d20
				ps394.OverlayValues[21] = d21
				ps394.OverlayValues[22] = d22
				ps394.OverlayValues[23] = d23
				ps394.OverlayValues[24] = d24
				ps394.OverlayValues[25] = d25
				ps394.OverlayValues[26] = d26
				ps394.OverlayValues[27] = d27
				ps394.OverlayValues[28] = d28
				ps394.OverlayValues[29] = d29
				ps394.OverlayValues[30] = d30
				ps394.OverlayValues[31] = d31
				ps394.OverlayValues[32] = d32
				ps394.OverlayValues[33] = d33
				ps394.OverlayValues[34] = d34
				ps394.OverlayValues[35] = d35
				ps394.OverlayValues[36] = d36
				ps394.OverlayValues[37] = d37
				ps394.OverlayValues[38] = d38
				ps394.OverlayValues[39] = d39
				ps394.OverlayValues[40] = d40
				ps394.OverlayValues[41] = d41
				ps394.OverlayValues[42] = d42
				ps394.OverlayValues[43] = d43
				ps394.OverlayValues[44] = d44
				ps394.OverlayValues[45] = d45
				ps394.OverlayValues[46] = d46
				ps394.OverlayValues[47] = d47
				ps394.OverlayValues[48] = d48
				ps394.OverlayValues[49] = d49
				ps394.OverlayValues[50] = d50
				ps394.OverlayValues[53] = d53
				ps394.OverlayValues[54] = d54
				ps394.OverlayValues[55] = d55
				ps394.OverlayValues[111] = d111
				ps394.OverlayValues[112] = d112
				ps394.OverlayValues[113] = d113
				ps394.OverlayValues[114] = d114
				ps394.OverlayValues[115] = d115
				ps394.OverlayValues[116] = d116
				ps394.OverlayValues[117] = d117
				ps394.OverlayValues[118] = d118
				ps394.OverlayValues[119] = d119
				ps394.OverlayValues[120] = d120
				ps394.OverlayValues[121] = d121
				ps394.OverlayValues[122] = d122
				ps394.OverlayValues[123] = d123
				ps394.OverlayValues[124] = d124
				ps394.OverlayValues[125] = d125
				ps394.OverlayValues[126] = d126
				ps394.OverlayValues[127] = d127
				ps394.OverlayValues[128] = d128
				ps394.OverlayValues[129] = d129
				ps394.OverlayValues[130] = d130
				ps394.OverlayValues[131] = d131
				ps394.OverlayValues[132] = d132
				ps394.OverlayValues[133] = d133
				ps394.OverlayValues[134] = d134
				ps394.OverlayValues[135] = d135
				ps394.OverlayValues[136] = d136
				ps394.OverlayValues[137] = d137
				ps394.OverlayValues[138] = d138
				ps394.OverlayValues[139] = d139
				ps394.OverlayValues[140] = d140
				ps394.OverlayValues[143] = d143
				ps394.OverlayValues[230] = d230
				ps394.OverlayValues[231] = d231
				ps394.OverlayValues[232] = d232
				ps394.OverlayValues[233] = d233
				ps394.OverlayValues[235] = d235
				ps394.OverlayValues[236] = d236
				ps394.OverlayValues[237] = d237
				ps394.OverlayValues[238] = d238
				ps394.OverlayValues[239] = d239
				ps394.OverlayValues[240] = d240
				ps394.OverlayValues[241] = d241
				ps394.OverlayValues[242] = d242
				ps394.OverlayValues[244] = d244
				ps394.OverlayValues[246] = d246
				ps394.OverlayValues[247] = d247
				ps394.OverlayValues[248] = d248
				ps394.OverlayValues[249] = d249
				ps394.OverlayValues[250] = d250
				ps394.OverlayValues[253] = d253
				ps394.OverlayValues[357] = d357
				ps394.OverlayValues[358] = d358
				ps394.OverlayValues[359] = d359
				ps394.OverlayValues[360] = d360
				ps394.OverlayValues[361] = d361
				ps394.OverlayValues[363] = d363
				ps394.OverlayValues[364] = d364
				ps394.OverlayValues[365] = d365
				ps394.OverlayValues[366] = d366
				ps394.OverlayValues[367] = d367
				ps394.OverlayValues[368] = d368
				ps394.OverlayValues[369] = d369
				ps394.OverlayValues[370] = d370
				ps394.OverlayValues[371] = d371
				ps394.OverlayValues[372] = d372
				ps394.OverlayValues[373] = d373
				ps394.OverlayValues[374] = d374
				ps394.OverlayValues[375] = d375
				ps394.OverlayValues[376] = d376
				ps394.OverlayValues[377] = d377
				ps394.OverlayValues[378] = d378
				ps394.OverlayValues[379] = d379
				ps394.OverlayValues[380] = d380
				ps394.OverlayValues[381] = d381
				ps394.OverlayValues[382] = d382
				ps394.OverlayValues[383] = d383
				ps394.OverlayValues[384] = d384
				ps394.OverlayValues[385] = d385
				ps394.OverlayValues[386] = d386
				ps394.OverlayValues[387] = d387
				ps394.OverlayValues[388] = d388
				ps394.OverlayValues[389] = d389
				ps394.OverlayValues[390] = d390
				ps394.OverlayValues[391] = d391
				ps394.OverlayValues[392] = d392
				ps394.OverlayValues[393] = d393
				return bbs[7].RenderPS(ps394)
			}
			if ps.General {
			}
			ps395 := scm.PhiState{General: ps.General}
			ps395.OverlayValues = make([]scm.JITValueDesc, 394)
			ps395.OverlayValues[1] = d1
			ps395.OverlayValues[2] = d2
			ps395.OverlayValues[3] = d3
			ps395.OverlayValues[4] = d4
			ps395.OverlayValues[5] = d5
			ps395.OverlayValues[6] = d6
			ps395.OverlayValues[7] = d7
			ps395.OverlayValues[8] = d8
			ps395.OverlayValues[9] = d9
			ps395.OverlayValues[10] = d10
			ps395.OverlayValues[11] = d11
			ps395.OverlayValues[12] = d12
			ps395.OverlayValues[13] = d13
			ps395.OverlayValues[14] = d14
			ps395.OverlayValues[15] = d15
			ps395.OverlayValues[17] = d17
			ps395.OverlayValues[18] = d18
			ps395.OverlayValues[19] = d19
			ps395.OverlayValues[20] = d20
			ps395.OverlayValues[21] = d21
			ps395.OverlayValues[22] = d22
			ps395.OverlayValues[23] = d23
			ps395.OverlayValues[24] = d24
			ps395.OverlayValues[25] = d25
			ps395.OverlayValues[26] = d26
			ps395.OverlayValues[27] = d27
			ps395.OverlayValues[28] = d28
			ps395.OverlayValues[29] = d29
			ps395.OverlayValues[30] = d30
			ps395.OverlayValues[31] = d31
			ps395.OverlayValues[32] = d32
			ps395.OverlayValues[33] = d33
			ps395.OverlayValues[34] = d34
			ps395.OverlayValues[35] = d35
			ps395.OverlayValues[36] = d36
			ps395.OverlayValues[37] = d37
			ps395.OverlayValues[38] = d38
			ps395.OverlayValues[39] = d39
			ps395.OverlayValues[40] = d40
			ps395.OverlayValues[41] = d41
			ps395.OverlayValues[42] = d42
			ps395.OverlayValues[43] = d43
			ps395.OverlayValues[44] = d44
			ps395.OverlayValues[45] = d45
			ps395.OverlayValues[46] = d46
			ps395.OverlayValues[47] = d47
			ps395.OverlayValues[48] = d48
			ps395.OverlayValues[49] = d49
			ps395.OverlayValues[50] = d50
			ps395.OverlayValues[53] = d53
			ps395.OverlayValues[54] = d54
			ps395.OverlayValues[55] = d55
			ps395.OverlayValues[111] = d111
			ps395.OverlayValues[112] = d112
			ps395.OverlayValues[113] = d113
			ps395.OverlayValues[114] = d114
			ps395.OverlayValues[115] = d115
			ps395.OverlayValues[116] = d116
			ps395.OverlayValues[117] = d117
			ps395.OverlayValues[118] = d118
			ps395.OverlayValues[119] = d119
			ps395.OverlayValues[120] = d120
			ps395.OverlayValues[121] = d121
			ps395.OverlayValues[122] = d122
			ps395.OverlayValues[123] = d123
			ps395.OverlayValues[124] = d124
			ps395.OverlayValues[125] = d125
			ps395.OverlayValues[126] = d126
			ps395.OverlayValues[127] = d127
			ps395.OverlayValues[128] = d128
			ps395.OverlayValues[129] = d129
			ps395.OverlayValues[130] = d130
			ps395.OverlayValues[131] = d131
			ps395.OverlayValues[132] = d132
			ps395.OverlayValues[133] = d133
			ps395.OverlayValues[134] = d134
			ps395.OverlayValues[135] = d135
			ps395.OverlayValues[136] = d136
			ps395.OverlayValues[137] = d137
			ps395.OverlayValues[138] = d138
			ps395.OverlayValues[139] = d139
			ps395.OverlayValues[140] = d140
			ps395.OverlayValues[143] = d143
			ps395.OverlayValues[230] = d230
			ps395.OverlayValues[231] = d231
			ps395.OverlayValues[232] = d232
			ps395.OverlayValues[233] = d233
			ps395.OverlayValues[235] = d235
			ps395.OverlayValues[236] = d236
			ps395.OverlayValues[237] = d237
			ps395.OverlayValues[238] = d238
			ps395.OverlayValues[239] = d239
			ps395.OverlayValues[240] = d240
			ps395.OverlayValues[241] = d241
			ps395.OverlayValues[242] = d242
			ps395.OverlayValues[244] = d244
			ps395.OverlayValues[246] = d246
			ps395.OverlayValues[247] = d247
			ps395.OverlayValues[248] = d248
			ps395.OverlayValues[249] = d249
			ps395.OverlayValues[250] = d250
			ps395.OverlayValues[253] = d253
			ps395.OverlayValues[357] = d357
			ps395.OverlayValues[358] = d358
			ps395.OverlayValues[359] = d359
			ps395.OverlayValues[360] = d360
			ps395.OverlayValues[361] = d361
			ps395.OverlayValues[363] = d363
			ps395.OverlayValues[364] = d364
			ps395.OverlayValues[365] = d365
			ps395.OverlayValues[366] = d366
			ps395.OverlayValues[367] = d367
			ps395.OverlayValues[368] = d368
			ps395.OverlayValues[369] = d369
			ps395.OverlayValues[370] = d370
			ps395.OverlayValues[371] = d371
			ps395.OverlayValues[372] = d372
			ps395.OverlayValues[373] = d373
			ps395.OverlayValues[374] = d374
			ps395.OverlayValues[375] = d375
			ps395.OverlayValues[376] = d376
			ps395.OverlayValues[377] = d377
			ps395.OverlayValues[378] = d378
			ps395.OverlayValues[379] = d379
			ps395.OverlayValues[380] = d380
			ps395.OverlayValues[381] = d381
			ps395.OverlayValues[382] = d382
			ps395.OverlayValues[383] = d383
			ps395.OverlayValues[384] = d384
			ps395.OverlayValues[385] = d385
			ps395.OverlayValues[386] = d386
			ps395.OverlayValues[387] = d387
			ps395.OverlayValues[388] = d388
			ps395.OverlayValues[389] = d389
			ps395.OverlayValues[390] = d390
			ps395.OverlayValues[391] = d391
			ps395.OverlayValues[392] = d392
			ps395.OverlayValues[393] = d393
			return bbs[9].RenderPS(ps395)
		}
		if !ps.General {
			ps.General = true
			return bbs[6].RenderPS(ps)
		}
		lbl24 := ctx.ReserveLabel()
		lbl25 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d393.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl24)
		ctx.EmitJmp(lbl25)
		ctx.MarkLabel(lbl24)
		ctx.EmitJmp(lbl8)
		ctx.MarkLabel(lbl25)
		ctx.EmitJmp(lbl10)
		ps396 := scm.PhiState{General: true}
		ps396.OverlayValues = make([]scm.JITValueDesc, 394)
		ps396.OverlayValues[1] = d1
		ps396.OverlayValues[2] = d2
		ps396.OverlayValues[3] = d3
		ps396.OverlayValues[4] = d4
		ps396.OverlayValues[5] = d5
		ps396.OverlayValues[6] = d6
		ps396.OverlayValues[7] = d7
		ps396.OverlayValues[8] = d8
		ps396.OverlayValues[9] = d9
		ps396.OverlayValues[10] = d10
		ps396.OverlayValues[11] = d11
		ps396.OverlayValues[12] = d12
		ps396.OverlayValues[13] = d13
		ps396.OverlayValues[14] = d14
		ps396.OverlayValues[15] = d15
		ps396.OverlayValues[17] = d17
		ps396.OverlayValues[18] = d18
		ps396.OverlayValues[19] = d19
		ps396.OverlayValues[20] = d20
		ps396.OverlayValues[21] = d21
		ps396.OverlayValues[22] = d22
		ps396.OverlayValues[23] = d23
		ps396.OverlayValues[24] = d24
		ps396.OverlayValues[25] = d25
		ps396.OverlayValues[26] = d26
		ps396.OverlayValues[27] = d27
		ps396.OverlayValues[28] = d28
		ps396.OverlayValues[29] = d29
		ps396.OverlayValues[30] = d30
		ps396.OverlayValues[31] = d31
		ps396.OverlayValues[32] = d32
		ps396.OverlayValues[33] = d33
		ps396.OverlayValues[34] = d34
		ps396.OverlayValues[35] = d35
		ps396.OverlayValues[36] = d36
		ps396.OverlayValues[37] = d37
		ps396.OverlayValues[38] = d38
		ps396.OverlayValues[39] = d39
		ps396.OverlayValues[40] = d40
		ps396.OverlayValues[41] = d41
		ps396.OverlayValues[42] = d42
		ps396.OverlayValues[43] = d43
		ps396.OverlayValues[44] = d44
		ps396.OverlayValues[45] = d45
		ps396.OverlayValues[46] = d46
		ps396.OverlayValues[47] = d47
		ps396.OverlayValues[48] = d48
		ps396.OverlayValues[49] = d49
		ps396.OverlayValues[50] = d50
		ps396.OverlayValues[53] = d53
		ps396.OverlayValues[54] = d54
		ps396.OverlayValues[55] = d55
		ps396.OverlayValues[111] = d111
		ps396.OverlayValues[112] = d112
		ps396.OverlayValues[113] = d113
		ps396.OverlayValues[114] = d114
		ps396.OverlayValues[115] = d115
		ps396.OverlayValues[116] = d116
		ps396.OverlayValues[117] = d117
		ps396.OverlayValues[118] = d118
		ps396.OverlayValues[119] = d119
		ps396.OverlayValues[120] = d120
		ps396.OverlayValues[121] = d121
		ps396.OverlayValues[122] = d122
		ps396.OverlayValues[123] = d123
		ps396.OverlayValues[124] = d124
		ps396.OverlayValues[125] = d125
		ps396.OverlayValues[126] = d126
		ps396.OverlayValues[127] = d127
		ps396.OverlayValues[128] = d128
		ps396.OverlayValues[129] = d129
		ps396.OverlayValues[130] = d130
		ps396.OverlayValues[131] = d131
		ps396.OverlayValues[132] = d132
		ps396.OverlayValues[133] = d133
		ps396.OverlayValues[134] = d134
		ps396.OverlayValues[135] = d135
		ps396.OverlayValues[136] = d136
		ps396.OverlayValues[137] = d137
		ps396.OverlayValues[138] = d138
		ps396.OverlayValues[139] = d139
		ps396.OverlayValues[140] = d140
		ps396.OverlayValues[143] = d143
		ps396.OverlayValues[230] = d230
		ps396.OverlayValues[231] = d231
		ps396.OverlayValues[232] = d232
		ps396.OverlayValues[233] = d233
		ps396.OverlayValues[235] = d235
		ps396.OverlayValues[236] = d236
		ps396.OverlayValues[237] = d237
		ps396.OverlayValues[238] = d238
		ps396.OverlayValues[239] = d239
		ps396.OverlayValues[240] = d240
		ps396.OverlayValues[241] = d241
		ps396.OverlayValues[242] = d242
		ps396.OverlayValues[244] = d244
		ps396.OverlayValues[246] = d246
		ps396.OverlayValues[247] = d247
		ps396.OverlayValues[248] = d248
		ps396.OverlayValues[249] = d249
		ps396.OverlayValues[250] = d250
		ps396.OverlayValues[253] = d253
		ps396.OverlayValues[357] = d357
		ps396.OverlayValues[358] = d358
		ps396.OverlayValues[359] = d359
		ps396.OverlayValues[360] = d360
		ps396.OverlayValues[361] = d361
		ps396.OverlayValues[363] = d363
		ps396.OverlayValues[364] = d364
		ps396.OverlayValues[365] = d365
		ps396.OverlayValues[366] = d366
		ps396.OverlayValues[367] = d367
		ps396.OverlayValues[368] = d368
		ps396.OverlayValues[369] = d369
		ps396.OverlayValues[370] = d370
		ps396.OverlayValues[371] = d371
		ps396.OverlayValues[372] = d372
		ps396.OverlayValues[373] = d373
		ps396.OverlayValues[374] = d374
		ps396.OverlayValues[375] = d375
		ps396.OverlayValues[376] = d376
		ps396.OverlayValues[377] = d377
		ps396.OverlayValues[378] = d378
		ps396.OverlayValues[379] = d379
		ps396.OverlayValues[380] = d380
		ps396.OverlayValues[381] = d381
		ps396.OverlayValues[382] = d382
		ps396.OverlayValues[383] = d383
		ps396.OverlayValues[384] = d384
		ps396.OverlayValues[385] = d385
		ps396.OverlayValues[386] = d386
		ps396.OverlayValues[387] = d387
		ps396.OverlayValues[388] = d388
		ps396.OverlayValues[389] = d389
		ps396.OverlayValues[390] = d390
		ps396.OverlayValues[391] = d391
		ps396.OverlayValues[392] = d392
		ps396.OverlayValues[393] = d393
		ps397 := scm.PhiState{General: true}
		ps397.OverlayValues = make([]scm.JITValueDesc, 394)
		ps397.OverlayValues[1] = d1
		ps397.OverlayValues[2] = d2
		ps397.OverlayValues[3] = d3
		ps397.OverlayValues[4] = d4
		ps397.OverlayValues[5] = d5
		ps397.OverlayValues[6] = d6
		ps397.OverlayValues[7] = d7
		ps397.OverlayValues[8] = d8
		ps397.OverlayValues[9] = d9
		ps397.OverlayValues[10] = d10
		ps397.OverlayValues[11] = d11
		ps397.OverlayValues[12] = d12
		ps397.OverlayValues[13] = d13
		ps397.OverlayValues[14] = d14
		ps397.OverlayValues[15] = d15
		ps397.OverlayValues[17] = d17
		ps397.OverlayValues[18] = d18
		ps397.OverlayValues[19] = d19
		ps397.OverlayValues[20] = d20
		ps397.OverlayValues[21] = d21
		ps397.OverlayValues[22] = d22
		ps397.OverlayValues[23] = d23
		ps397.OverlayValues[24] = d24
		ps397.OverlayValues[25] = d25
		ps397.OverlayValues[26] = d26
		ps397.OverlayValues[27] = d27
		ps397.OverlayValues[28] = d28
		ps397.OverlayValues[29] = d29
		ps397.OverlayValues[30] = d30
		ps397.OverlayValues[31] = d31
		ps397.OverlayValues[32] = d32
		ps397.OverlayValues[33] = d33
		ps397.OverlayValues[34] = d34
		ps397.OverlayValues[35] = d35
		ps397.OverlayValues[36] = d36
		ps397.OverlayValues[37] = d37
		ps397.OverlayValues[38] = d38
		ps397.OverlayValues[39] = d39
		ps397.OverlayValues[40] = d40
		ps397.OverlayValues[41] = d41
		ps397.OverlayValues[42] = d42
		ps397.OverlayValues[43] = d43
		ps397.OverlayValues[44] = d44
		ps397.OverlayValues[45] = d45
		ps397.OverlayValues[46] = d46
		ps397.OverlayValues[47] = d47
		ps397.OverlayValues[48] = d48
		ps397.OverlayValues[49] = d49
		ps397.OverlayValues[50] = d50
		ps397.OverlayValues[53] = d53
		ps397.OverlayValues[54] = d54
		ps397.OverlayValues[55] = d55
		ps397.OverlayValues[111] = d111
		ps397.OverlayValues[112] = d112
		ps397.OverlayValues[113] = d113
		ps397.OverlayValues[114] = d114
		ps397.OverlayValues[115] = d115
		ps397.OverlayValues[116] = d116
		ps397.OverlayValues[117] = d117
		ps397.OverlayValues[118] = d118
		ps397.OverlayValues[119] = d119
		ps397.OverlayValues[120] = d120
		ps397.OverlayValues[121] = d121
		ps397.OverlayValues[122] = d122
		ps397.OverlayValues[123] = d123
		ps397.OverlayValues[124] = d124
		ps397.OverlayValues[125] = d125
		ps397.OverlayValues[126] = d126
		ps397.OverlayValues[127] = d127
		ps397.OverlayValues[128] = d128
		ps397.OverlayValues[129] = d129
		ps397.OverlayValues[130] = d130
		ps397.OverlayValues[131] = d131
		ps397.OverlayValues[132] = d132
		ps397.OverlayValues[133] = d133
		ps397.OverlayValues[134] = d134
		ps397.OverlayValues[135] = d135
		ps397.OverlayValues[136] = d136
		ps397.OverlayValues[137] = d137
		ps397.OverlayValues[138] = d138
		ps397.OverlayValues[139] = d139
		ps397.OverlayValues[140] = d140
		ps397.OverlayValues[143] = d143
		ps397.OverlayValues[230] = d230
		ps397.OverlayValues[231] = d231
		ps397.OverlayValues[232] = d232
		ps397.OverlayValues[233] = d233
		ps397.OverlayValues[235] = d235
		ps397.OverlayValues[236] = d236
		ps397.OverlayValues[237] = d237
		ps397.OverlayValues[238] = d238
		ps397.OverlayValues[239] = d239
		ps397.OverlayValues[240] = d240
		ps397.OverlayValues[241] = d241
		ps397.OverlayValues[242] = d242
		ps397.OverlayValues[244] = d244
		ps397.OverlayValues[246] = d246
		ps397.OverlayValues[247] = d247
		ps397.OverlayValues[248] = d248
		ps397.OverlayValues[249] = d249
		ps397.OverlayValues[250] = d250
		ps397.OverlayValues[253] = d253
		ps397.OverlayValues[357] = d357
		ps397.OverlayValues[358] = d358
		ps397.OverlayValues[359] = d359
		ps397.OverlayValues[360] = d360
		ps397.OverlayValues[361] = d361
		ps397.OverlayValues[363] = d363
		ps397.OverlayValues[364] = d364
		ps397.OverlayValues[365] = d365
		ps397.OverlayValues[366] = d366
		ps397.OverlayValues[367] = d367
		ps397.OverlayValues[368] = d368
		ps397.OverlayValues[369] = d369
		ps397.OverlayValues[370] = d370
		ps397.OverlayValues[371] = d371
		ps397.OverlayValues[372] = d372
		ps397.OverlayValues[373] = d373
		ps397.OverlayValues[374] = d374
		ps397.OverlayValues[375] = d375
		ps397.OverlayValues[376] = d376
		ps397.OverlayValues[377] = d377
		ps397.OverlayValues[378] = d378
		ps397.OverlayValues[379] = d379
		ps397.OverlayValues[380] = d380
		ps397.OverlayValues[381] = d381
		ps397.OverlayValues[382] = d382
		ps397.OverlayValues[383] = d383
		ps397.OverlayValues[384] = d384
		ps397.OverlayValues[385] = d385
		ps397.OverlayValues[386] = d386
		ps397.OverlayValues[387] = d387
		ps397.OverlayValues[388] = d388
		ps397.OverlayValues[389] = d389
		ps397.OverlayValues[390] = d390
		ps397.OverlayValues[391] = d391
		ps397.OverlayValues[392] = d392
		ps397.OverlayValues[393] = d393
		snap398 := d1
		snap399 := d2
		snap400 := d3
		snap401 := d4
		snap402 := d5
		snap403 := d6
		snap404 := d7
		snap405 := d8
		snap406 := d9
		snap407 := d10
		snap408 := d11
		snap409 := d12
		snap410 := d13
		snap411 := d14
		snap412 := d15
		snap413 := d17
		snap414 := d18
		snap415 := d19
		snap416 := d20
		snap417 := d21
		snap418 := d22
		snap419 := d23
		snap420 := d24
		snap421 := d25
		snap422 := d26
		snap423 := d27
		snap424 := d28
		snap425 := d29
		snap426 := d30
		snap427 := d31
		snap428 := d32
		snap429 := d33
		snap430 := d34
		snap431 := d35
		snap432 := d36
		snap433 := d37
		snap434 := d38
		snap435 := d39
		snap436 := d40
		snap437 := d41
		snap438 := d42
		snap439 := d43
		snap440 := d44
		snap441 := d45
		snap442 := d46
		snap443 := d47
		snap444 := d48
		snap445 := d49
		snap446 := d50
		snap447 := d53
		snap448 := d54
		snap449 := d55
		snap450 := d111
		snap451 := d112
		snap452 := d113
		snap453 := d114
		snap454 := d115
		snap455 := d116
		snap456 := d117
		snap457 := d118
		snap458 := d119
		snap459 := d120
		snap460 := d121
		snap461 := d122
		snap462 := d123
		snap463 := d124
		snap464 := d125
		snap465 := d126
		snap466 := d127
		snap467 := d128
		snap468 := d129
		snap469 := d130
		snap470 := d131
		snap471 := d132
		snap472 := d133
		snap473 := d134
		snap474 := d135
		snap475 := d136
		snap476 := d137
		snap477 := d138
		snap478 := d139
		snap479 := d140
		snap480 := d143
		snap481 := d230
		snap482 := d231
		snap483 := d232
		snap484 := d233
		snap485 := d235
		snap486 := d236
		snap487 := d237
		snap488 := d238
		snap489 := d239
		snap490 := d240
		snap491 := d241
		snap492 := d242
		snap493 := d244
		snap494 := d246
		snap495 := d247
		snap496 := d248
		snap497 := d249
		snap498 := d250
		snap499 := d253
		snap500 := d357
		snap501 := d358
		snap502 := d359
		snap503 := d360
		snap504 := d361
		snap505 := d363
		snap506 := d364
		snap507 := d365
		snap508 := d366
		snap509 := d367
		snap510 := d368
		snap511 := d369
		snap512 := d370
		snap513 := d371
		snap514 := d372
		snap515 := d373
		snap516 := d374
		snap517 := d375
		snap518 := d376
		snap519 := d377
		snap520 := d378
		snap521 := d379
		snap522 := d380
		snap523 := d381
		snap524 := d382
		snap525 := d383
		snap526 := d384
		snap527 := d385
		snap528 := d386
		snap529 := d387
		snap530 := d388
		snap531 := d389
		snap532 := d390
		snap533 := d391
		snap534 := d392
		snap535 := d393
		alloc536 := ctx.SnapshotAllocState()
		if !bbs[9].Rendered {
			bbs[9].RenderPS(ps397)
		}
		ctx.RestoreAllocState(alloc536)
		d1 = snap398
		d2 = snap399
		d3 = snap400
		d4 = snap401
		d5 = snap402
		d6 = snap403
		d7 = snap404
		d8 = snap405
		d9 = snap406
		d10 = snap407
		d11 = snap408
		d12 = snap409
		d13 = snap410
		d14 = snap411
		d15 = snap412
		d17 = snap413
		d18 = snap414
		d19 = snap415
		d20 = snap416
		d21 = snap417
		d22 = snap418
		d23 = snap419
		d24 = snap420
		d25 = snap421
		d26 = snap422
		d27 = snap423
		d28 = snap424
		d29 = snap425
		d30 = snap426
		d31 = snap427
		d32 = snap428
		d33 = snap429
		d34 = snap430
		d35 = snap431
		d36 = snap432
		d37 = snap433
		d38 = snap434
		d39 = snap435
		d40 = snap436
		d41 = snap437
		d42 = snap438
		d43 = snap439
		d44 = snap440
		d45 = snap441
		d46 = snap442
		d47 = snap443
		d48 = snap444
		d49 = snap445
		d50 = snap446
		d53 = snap447
		d54 = snap448
		d55 = snap449
		d111 = snap450
		d112 = snap451
		d113 = snap452
		d114 = snap453
		d115 = snap454
		d116 = snap455
		d117 = snap456
		d118 = snap457
		d119 = snap458
		d120 = snap459
		d121 = snap460
		d122 = snap461
		d123 = snap462
		d124 = snap463
		d125 = snap464
		d126 = snap465
		d127 = snap466
		d128 = snap467
		d129 = snap468
		d130 = snap469
		d131 = snap470
		d132 = snap471
		d133 = snap472
		d134 = snap473
		d135 = snap474
		d136 = snap475
		d137 = snap476
		d138 = snap477
		d139 = snap478
		d140 = snap479
		d143 = snap480
		d230 = snap481
		d231 = snap482
		d232 = snap483
		d233 = snap484
		d235 = snap485
		d236 = snap486
		d237 = snap487
		d238 = snap488
		d239 = snap489
		d240 = snap490
		d241 = snap491
		d242 = snap492
		d244 = snap493
		d246 = snap494
		d247 = snap495
		d248 = snap496
		d249 = snap497
		d250 = snap498
		d253 = snap499
		d357 = snap500
		d358 = snap501
		d359 = snap502
		d360 = snap503
		d361 = snap504
		d363 = snap505
		d364 = snap506
		d365 = snap507
		d366 = snap508
		d367 = snap509
		d368 = snap510
		d369 = snap511
		d370 = snap512
		d371 = snap513
		d372 = snap514
		d373 = snap515
		d374 = snap516
		d375 = snap517
		d376 = snap518
		d377 = snap519
		d378 = snap520
		d379 = snap521
		d380 = snap522
		d381 = snap523
		d382 = snap524
		d383 = snap525
		d384 = snap526
		d385 = snap527
		d386 = snap528
		d387 = snap529
		d388 = snap530
		d389 = snap531
		d390 = snap532
		d391 = snap533
		d392 = snap534
		d393 = snap535
		if !bbs[7].Rendered {
			return bbs[7].RenderPS(ps396)
		}
		return result
		ctx.FreeDesc(&d392)
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
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
			d365 = ps.OverlayValues[365]
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
		if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
			d371 = ps.OverlayValues[371]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d537 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d537 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() - 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitSubRegImm32(scratch, int32(1))
			d537 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d537)
		}
		if d537.Loc == scm.LocImm {
			d537 = scm.JITValueDesc{Loc: scm.LocImm, Type: d537.Type, Imm: scm.NewInt(int64(uint64(d537.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d537.Reg, 32)
			ctx.EmitShrRegImm8(d537.Reg, 32)
		}
		if d537.Loc == scm.LocReg && d5.Loc == scm.LocReg && d537.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d537)
		ctx.EmitStoreToStack(d537, int32(bbs[8].PhiBase)+int32(16))
		ctx.StabilizeDescForControlFlow(&d537)
		if ps.General {
			ctx.SyncDesc(&d6)
			if d6.Loc == scm.LocReg {
				ctx.ProtectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.ProtectReg(d6.Reg)
				ctx.ProtectReg(d6.Reg2)
			}
			d538 = d6
			if d538.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d538)
			d539 = d538
			if d539.Loc == scm.LocImm {
				d539 = scm.JITValueDesc{Loc: scm.LocImm, Type: d539.Type, Imm: scm.NewInt(int64(uint64(d539.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d539.Reg, 32)
				ctx.EmitShrRegImm8(d539.Reg, 32)
			}
			ctx.EmitStoreToStack(d539, int32(bbs[8].PhiBase)+int32(0))
			if d6.Loc == scm.LocReg {
				ctx.UnprotectReg(d6.Reg)
			} else if d6.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d6.Reg)
				ctx.UnprotectReg(d6.Reg2)
			}
		}
		ps540 := scm.PhiState{General: ps.General}
		ps540.OverlayValues = make([]scm.JITValueDesc, 540)
		ps540.OverlayValues[1] = d1
		ps540.OverlayValues[2] = d2
		ps540.OverlayValues[3] = d3
		ps540.OverlayValues[4] = d4
		ps540.OverlayValues[5] = d5
		ps540.OverlayValues[6] = d6
		ps540.OverlayValues[7] = d7
		ps540.OverlayValues[8] = d8
		ps540.OverlayValues[9] = d9
		ps540.OverlayValues[10] = d10
		ps540.OverlayValues[11] = d11
		ps540.OverlayValues[12] = d12
		ps540.OverlayValues[13] = d13
		ps540.OverlayValues[14] = d14
		ps540.OverlayValues[15] = d15
		ps540.OverlayValues[17] = d17
		ps540.OverlayValues[18] = d18
		ps540.OverlayValues[19] = d19
		ps540.OverlayValues[20] = d20
		ps540.OverlayValues[21] = d21
		ps540.OverlayValues[22] = d22
		ps540.OverlayValues[23] = d23
		ps540.OverlayValues[24] = d24
		ps540.OverlayValues[25] = d25
		ps540.OverlayValues[26] = d26
		ps540.OverlayValues[27] = d27
		ps540.OverlayValues[28] = d28
		ps540.OverlayValues[29] = d29
		ps540.OverlayValues[30] = d30
		ps540.OverlayValues[31] = d31
		ps540.OverlayValues[32] = d32
		ps540.OverlayValues[33] = d33
		ps540.OverlayValues[34] = d34
		ps540.OverlayValues[35] = d35
		ps540.OverlayValues[36] = d36
		ps540.OverlayValues[37] = d37
		ps540.OverlayValues[38] = d38
		ps540.OverlayValues[39] = d39
		ps540.OverlayValues[40] = d40
		ps540.OverlayValues[41] = d41
		ps540.OverlayValues[42] = d42
		ps540.OverlayValues[43] = d43
		ps540.OverlayValues[44] = d44
		ps540.OverlayValues[45] = d45
		ps540.OverlayValues[46] = d46
		ps540.OverlayValues[47] = d47
		ps540.OverlayValues[48] = d48
		ps540.OverlayValues[49] = d49
		ps540.OverlayValues[50] = d50
		ps540.OverlayValues[53] = d53
		ps540.OverlayValues[54] = d54
		ps540.OverlayValues[55] = d55
		ps540.OverlayValues[111] = d111
		ps540.OverlayValues[112] = d112
		ps540.OverlayValues[113] = d113
		ps540.OverlayValues[114] = d114
		ps540.OverlayValues[115] = d115
		ps540.OverlayValues[116] = d116
		ps540.OverlayValues[117] = d117
		ps540.OverlayValues[118] = d118
		ps540.OverlayValues[119] = d119
		ps540.OverlayValues[120] = d120
		ps540.OverlayValues[121] = d121
		ps540.OverlayValues[122] = d122
		ps540.OverlayValues[123] = d123
		ps540.OverlayValues[124] = d124
		ps540.OverlayValues[125] = d125
		ps540.OverlayValues[126] = d126
		ps540.OverlayValues[127] = d127
		ps540.OverlayValues[128] = d128
		ps540.OverlayValues[129] = d129
		ps540.OverlayValues[130] = d130
		ps540.OverlayValues[131] = d131
		ps540.OverlayValues[132] = d132
		ps540.OverlayValues[133] = d133
		ps540.OverlayValues[134] = d134
		ps540.OverlayValues[135] = d135
		ps540.OverlayValues[136] = d136
		ps540.OverlayValues[137] = d137
		ps540.OverlayValues[138] = d138
		ps540.OverlayValues[139] = d139
		ps540.OverlayValues[140] = d140
		ps540.OverlayValues[143] = d143
		ps540.OverlayValues[230] = d230
		ps540.OverlayValues[231] = d231
		ps540.OverlayValues[232] = d232
		ps540.OverlayValues[233] = d233
		ps540.OverlayValues[235] = d235
		ps540.OverlayValues[236] = d236
		ps540.OverlayValues[237] = d237
		ps540.OverlayValues[238] = d238
		ps540.OverlayValues[239] = d239
		ps540.OverlayValues[240] = d240
		ps540.OverlayValues[241] = d241
		ps540.OverlayValues[242] = d242
		ps540.OverlayValues[244] = d244
		ps540.OverlayValues[246] = d246
		ps540.OverlayValues[247] = d247
		ps540.OverlayValues[248] = d248
		ps540.OverlayValues[249] = d249
		ps540.OverlayValues[250] = d250
		ps540.OverlayValues[253] = d253
		ps540.OverlayValues[357] = d357
		ps540.OverlayValues[358] = d358
		ps540.OverlayValues[359] = d359
		ps540.OverlayValues[360] = d360
		ps540.OverlayValues[361] = d361
		ps540.OverlayValues[363] = d363
		ps540.OverlayValues[364] = d364
		ps540.OverlayValues[365] = d365
		ps540.OverlayValues[366] = d366
		ps540.OverlayValues[367] = d367
		ps540.OverlayValues[368] = d368
		ps540.OverlayValues[369] = d369
		ps540.OverlayValues[370] = d370
		ps540.OverlayValues[371] = d371
		ps540.OverlayValues[372] = d372
		ps540.OverlayValues[373] = d373
		ps540.OverlayValues[374] = d374
		ps540.OverlayValues[375] = d375
		ps540.OverlayValues[376] = d376
		ps540.OverlayValues[377] = d377
		ps540.OverlayValues[378] = d378
		ps540.OverlayValues[379] = d379
		ps540.OverlayValues[380] = d380
		ps540.OverlayValues[381] = d381
		ps540.OverlayValues[382] = d382
		ps540.OverlayValues[383] = d383
		ps540.OverlayValues[384] = d384
		ps540.OverlayValues[385] = d385
		ps540.OverlayValues[386] = d386
		ps540.OverlayValues[387] = d387
		ps540.OverlayValues[388] = d388
		ps540.OverlayValues[389] = d389
		ps540.OverlayValues[390] = d390
		ps540.OverlayValues[391] = d391
		ps540.OverlayValues[392] = d392
		ps540.OverlayValues[393] = d393
		ps540.OverlayValues[537] = d537
		ps540.OverlayValues[538] = d538
		ps540.OverlayValues[539] = d539
		ps540.PhiValues = make([]scm.JITValueDesc, 2)
		d541 = d6
		ps540.PhiValues[0] = d541
		if ps540.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps540)
		return result
	}
	bbs[8].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d542 := ps.PhiValues[0]
				ctx.EnsureDesc(&d542)
				ctx.EmitStoreToStack(d542, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d543 := ps.PhiValues[1]
				ctx.EnsureDesc(&d543)
				ctx.EmitStoreToStack(d543, int32(bbs[8].PhiBase)+int32(16))
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
			d365 = ps.OverlayValues[365]
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
		if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
			d371 = ps.OverlayValues[371]
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
		if len(ps.OverlayValues) > 537 && ps.OverlayValues[537].Loc != scm.LocNone {
			d537 = ps.OverlayValues[537]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 542 && ps.OverlayValues[542].Loc != scm.LocNone {
			d542 = ps.OverlayValues[542]
		}
		if len(ps.OverlayValues) > 543 && ps.OverlayValues[543].Loc != scm.LocNone {
			d543 = ps.OverlayValues[543]
		}
		if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
			d8 = ps.PhiValues[0]
		}
		if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
			d9 = ps.PhiValues[1]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.StabilizeDescForControlFlow(&d8)
		ctx.StabilizeDescForControlFlow(&d9)
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDescsTogether(&d8, &d9)
		var d544 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d544 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d8.Imm.Int()) == uint64(d9.Imm.Int()))}
		} else if d9.Loc == scm.LocImm {
			r102 := ctx.AllocRegExcept(d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d8.Reg, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitCmpInt64(d8.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r102, scm.CondEqual)
			d544 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r102}
			ctx.BindReg(r102, &d544)
		} else if d8.Loc == scm.LocImm {
			r103 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d9.Reg)
			ctx.EmitSetcc(r103, scm.CondEqual)
			d544 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r103}
			ctx.BindReg(r103, &d544)
		} else {
			r104 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitCmpInt64(d8.Reg, d9.Reg)
			ctx.EmitSetcc(r104, scm.CondEqual)
			d544 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r104}
			ctx.BindReg(r104, &d544)
		}
		d545 = d544
		ctx.EnsureDesc(&d545)
		if d545.Loc != scm.LocImm && d545.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d545.Loc == scm.LocImm {
			if d545.Imm.Bool() {
				if ps.General {
					ctx.SyncDesc(&d8)
					if d8.Loc == scm.LocReg {
						ctx.ProtectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.ProtectReg(d8.Reg)
						ctx.ProtectReg(d8.Reg2)
					}
					d546 = d8
					if d546.Loc == scm.LocNone {
						panic("jit: phi source has no location")
					}
					ctx.EnsureDesc(&d546)
					d547 = d546
					if d547.Loc == scm.LocImm {
						d547 = scm.JITValueDesc{Loc: scm.LocImm, Type: d547.Type, Imm: scm.NewInt(int64(uint64(d547.Imm.Int()) & 0xffffffff))}
					} else {
						ctx.EmitShlRegImm8(d547.Reg, 32)
						ctx.EmitShrRegImm8(d547.Reg, 32)
					}
					ctx.EmitStoreToStack(d547, int32(bbs[2].PhiBase)+int32(0))
					if d8.Loc == scm.LocReg {
						ctx.UnprotectReg(d8.Reg)
					} else if d8.Loc == scm.LocRegPair {
						ctx.UnprotectReg(d8.Reg)
						ctx.UnprotectReg(d8.Reg2)
					}
				}
				ps548 := scm.PhiState{General: ps.General}
				ps548.OverlayValues = make([]scm.JITValueDesc, 548)
				ps548.OverlayValues[1] = d1
				ps548.OverlayValues[2] = d2
				ps548.OverlayValues[3] = d3
				ps548.OverlayValues[4] = d4
				ps548.OverlayValues[5] = d5
				ps548.OverlayValues[6] = d6
				ps548.OverlayValues[7] = d7
				ps548.OverlayValues[8] = d8
				ps548.OverlayValues[9] = d9
				ps548.OverlayValues[10] = d10
				ps548.OverlayValues[11] = d11
				ps548.OverlayValues[12] = d12
				ps548.OverlayValues[13] = d13
				ps548.OverlayValues[14] = d14
				ps548.OverlayValues[15] = d15
				ps548.OverlayValues[17] = d17
				ps548.OverlayValues[18] = d18
				ps548.OverlayValues[19] = d19
				ps548.OverlayValues[20] = d20
				ps548.OverlayValues[21] = d21
				ps548.OverlayValues[22] = d22
				ps548.OverlayValues[23] = d23
				ps548.OverlayValues[24] = d24
				ps548.OverlayValues[25] = d25
				ps548.OverlayValues[26] = d26
				ps548.OverlayValues[27] = d27
				ps548.OverlayValues[28] = d28
				ps548.OverlayValues[29] = d29
				ps548.OverlayValues[30] = d30
				ps548.OverlayValues[31] = d31
				ps548.OverlayValues[32] = d32
				ps548.OverlayValues[33] = d33
				ps548.OverlayValues[34] = d34
				ps548.OverlayValues[35] = d35
				ps548.OverlayValues[36] = d36
				ps548.OverlayValues[37] = d37
				ps548.OverlayValues[38] = d38
				ps548.OverlayValues[39] = d39
				ps548.OverlayValues[40] = d40
				ps548.OverlayValues[41] = d41
				ps548.OverlayValues[42] = d42
				ps548.OverlayValues[43] = d43
				ps548.OverlayValues[44] = d44
				ps548.OverlayValues[45] = d45
				ps548.OverlayValues[46] = d46
				ps548.OverlayValues[47] = d47
				ps548.OverlayValues[48] = d48
				ps548.OverlayValues[49] = d49
				ps548.OverlayValues[50] = d50
				ps548.OverlayValues[53] = d53
				ps548.OverlayValues[54] = d54
				ps548.OverlayValues[55] = d55
				ps548.OverlayValues[111] = d111
				ps548.OverlayValues[112] = d112
				ps548.OverlayValues[113] = d113
				ps548.OverlayValues[114] = d114
				ps548.OverlayValues[115] = d115
				ps548.OverlayValues[116] = d116
				ps548.OverlayValues[117] = d117
				ps548.OverlayValues[118] = d118
				ps548.OverlayValues[119] = d119
				ps548.OverlayValues[120] = d120
				ps548.OverlayValues[121] = d121
				ps548.OverlayValues[122] = d122
				ps548.OverlayValues[123] = d123
				ps548.OverlayValues[124] = d124
				ps548.OverlayValues[125] = d125
				ps548.OverlayValues[126] = d126
				ps548.OverlayValues[127] = d127
				ps548.OverlayValues[128] = d128
				ps548.OverlayValues[129] = d129
				ps548.OverlayValues[130] = d130
				ps548.OverlayValues[131] = d131
				ps548.OverlayValues[132] = d132
				ps548.OverlayValues[133] = d133
				ps548.OverlayValues[134] = d134
				ps548.OverlayValues[135] = d135
				ps548.OverlayValues[136] = d136
				ps548.OverlayValues[137] = d137
				ps548.OverlayValues[138] = d138
				ps548.OverlayValues[139] = d139
				ps548.OverlayValues[140] = d140
				ps548.OverlayValues[143] = d143
				ps548.OverlayValues[230] = d230
				ps548.OverlayValues[231] = d231
				ps548.OverlayValues[232] = d232
				ps548.OverlayValues[233] = d233
				ps548.OverlayValues[235] = d235
				ps548.OverlayValues[236] = d236
				ps548.OverlayValues[237] = d237
				ps548.OverlayValues[238] = d238
				ps548.OverlayValues[239] = d239
				ps548.OverlayValues[240] = d240
				ps548.OverlayValues[241] = d241
				ps548.OverlayValues[242] = d242
				ps548.OverlayValues[244] = d244
				ps548.OverlayValues[246] = d246
				ps548.OverlayValues[247] = d247
				ps548.OverlayValues[248] = d248
				ps548.OverlayValues[249] = d249
				ps548.OverlayValues[250] = d250
				ps548.OverlayValues[253] = d253
				ps548.OverlayValues[357] = d357
				ps548.OverlayValues[358] = d358
				ps548.OverlayValues[359] = d359
				ps548.OverlayValues[360] = d360
				ps548.OverlayValues[361] = d361
				ps548.OverlayValues[363] = d363
				ps548.OverlayValues[364] = d364
				ps548.OverlayValues[365] = d365
				ps548.OverlayValues[366] = d366
				ps548.OverlayValues[367] = d367
				ps548.OverlayValues[368] = d368
				ps548.OverlayValues[369] = d369
				ps548.OverlayValues[370] = d370
				ps548.OverlayValues[371] = d371
				ps548.OverlayValues[372] = d372
				ps548.OverlayValues[373] = d373
				ps548.OverlayValues[374] = d374
				ps548.OverlayValues[375] = d375
				ps548.OverlayValues[376] = d376
				ps548.OverlayValues[377] = d377
				ps548.OverlayValues[378] = d378
				ps548.OverlayValues[379] = d379
				ps548.OverlayValues[380] = d380
				ps548.OverlayValues[381] = d381
				ps548.OverlayValues[382] = d382
				ps548.OverlayValues[383] = d383
				ps548.OverlayValues[384] = d384
				ps548.OverlayValues[385] = d385
				ps548.OverlayValues[386] = d386
				ps548.OverlayValues[387] = d387
				ps548.OverlayValues[388] = d388
				ps548.OverlayValues[389] = d389
				ps548.OverlayValues[390] = d390
				ps548.OverlayValues[391] = d391
				ps548.OverlayValues[392] = d392
				ps548.OverlayValues[393] = d393
				ps548.OverlayValues[537] = d537
				ps548.OverlayValues[538] = d538
				ps548.OverlayValues[539] = d539
				ps548.OverlayValues[541] = d541
				ps548.OverlayValues[542] = d542
				ps548.OverlayValues[543] = d543
				ps548.OverlayValues[544] = d544
				ps548.OverlayValues[545] = d545
				ps548.OverlayValues[546] = d546
				ps548.OverlayValues[547] = d547
				ps548.PhiValues = make([]scm.JITValueDesc, 1)
				d549 = d8
				ps548.PhiValues[0] = d549
				return bbs[2].RenderPS(ps548)
			}
			if ps.General {
			}
			ps550 := scm.PhiState{General: ps.General}
			ps550.OverlayValues = make([]scm.JITValueDesc, 550)
			ps550.OverlayValues[1] = d1
			ps550.OverlayValues[2] = d2
			ps550.OverlayValues[3] = d3
			ps550.OverlayValues[4] = d4
			ps550.OverlayValues[5] = d5
			ps550.OverlayValues[6] = d6
			ps550.OverlayValues[7] = d7
			ps550.OverlayValues[8] = d8
			ps550.OverlayValues[9] = d9
			ps550.OverlayValues[10] = d10
			ps550.OverlayValues[11] = d11
			ps550.OverlayValues[12] = d12
			ps550.OverlayValues[13] = d13
			ps550.OverlayValues[14] = d14
			ps550.OverlayValues[15] = d15
			ps550.OverlayValues[17] = d17
			ps550.OverlayValues[18] = d18
			ps550.OverlayValues[19] = d19
			ps550.OverlayValues[20] = d20
			ps550.OverlayValues[21] = d21
			ps550.OverlayValues[22] = d22
			ps550.OverlayValues[23] = d23
			ps550.OverlayValues[24] = d24
			ps550.OverlayValues[25] = d25
			ps550.OverlayValues[26] = d26
			ps550.OverlayValues[27] = d27
			ps550.OverlayValues[28] = d28
			ps550.OverlayValues[29] = d29
			ps550.OverlayValues[30] = d30
			ps550.OverlayValues[31] = d31
			ps550.OverlayValues[32] = d32
			ps550.OverlayValues[33] = d33
			ps550.OverlayValues[34] = d34
			ps550.OverlayValues[35] = d35
			ps550.OverlayValues[36] = d36
			ps550.OverlayValues[37] = d37
			ps550.OverlayValues[38] = d38
			ps550.OverlayValues[39] = d39
			ps550.OverlayValues[40] = d40
			ps550.OverlayValues[41] = d41
			ps550.OverlayValues[42] = d42
			ps550.OverlayValues[43] = d43
			ps550.OverlayValues[44] = d44
			ps550.OverlayValues[45] = d45
			ps550.OverlayValues[46] = d46
			ps550.OverlayValues[47] = d47
			ps550.OverlayValues[48] = d48
			ps550.OverlayValues[49] = d49
			ps550.OverlayValues[50] = d50
			ps550.OverlayValues[53] = d53
			ps550.OverlayValues[54] = d54
			ps550.OverlayValues[55] = d55
			ps550.OverlayValues[111] = d111
			ps550.OverlayValues[112] = d112
			ps550.OverlayValues[113] = d113
			ps550.OverlayValues[114] = d114
			ps550.OverlayValues[115] = d115
			ps550.OverlayValues[116] = d116
			ps550.OverlayValues[117] = d117
			ps550.OverlayValues[118] = d118
			ps550.OverlayValues[119] = d119
			ps550.OverlayValues[120] = d120
			ps550.OverlayValues[121] = d121
			ps550.OverlayValues[122] = d122
			ps550.OverlayValues[123] = d123
			ps550.OverlayValues[124] = d124
			ps550.OverlayValues[125] = d125
			ps550.OverlayValues[126] = d126
			ps550.OverlayValues[127] = d127
			ps550.OverlayValues[128] = d128
			ps550.OverlayValues[129] = d129
			ps550.OverlayValues[130] = d130
			ps550.OverlayValues[131] = d131
			ps550.OverlayValues[132] = d132
			ps550.OverlayValues[133] = d133
			ps550.OverlayValues[134] = d134
			ps550.OverlayValues[135] = d135
			ps550.OverlayValues[136] = d136
			ps550.OverlayValues[137] = d137
			ps550.OverlayValues[138] = d138
			ps550.OverlayValues[139] = d139
			ps550.OverlayValues[140] = d140
			ps550.OverlayValues[143] = d143
			ps550.OverlayValues[230] = d230
			ps550.OverlayValues[231] = d231
			ps550.OverlayValues[232] = d232
			ps550.OverlayValues[233] = d233
			ps550.OverlayValues[235] = d235
			ps550.OverlayValues[236] = d236
			ps550.OverlayValues[237] = d237
			ps550.OverlayValues[238] = d238
			ps550.OverlayValues[239] = d239
			ps550.OverlayValues[240] = d240
			ps550.OverlayValues[241] = d241
			ps550.OverlayValues[242] = d242
			ps550.OverlayValues[244] = d244
			ps550.OverlayValues[246] = d246
			ps550.OverlayValues[247] = d247
			ps550.OverlayValues[248] = d248
			ps550.OverlayValues[249] = d249
			ps550.OverlayValues[250] = d250
			ps550.OverlayValues[253] = d253
			ps550.OverlayValues[357] = d357
			ps550.OverlayValues[358] = d358
			ps550.OverlayValues[359] = d359
			ps550.OverlayValues[360] = d360
			ps550.OverlayValues[361] = d361
			ps550.OverlayValues[363] = d363
			ps550.OverlayValues[364] = d364
			ps550.OverlayValues[365] = d365
			ps550.OverlayValues[366] = d366
			ps550.OverlayValues[367] = d367
			ps550.OverlayValues[368] = d368
			ps550.OverlayValues[369] = d369
			ps550.OverlayValues[370] = d370
			ps550.OverlayValues[371] = d371
			ps550.OverlayValues[372] = d372
			ps550.OverlayValues[373] = d373
			ps550.OverlayValues[374] = d374
			ps550.OverlayValues[375] = d375
			ps550.OverlayValues[376] = d376
			ps550.OverlayValues[377] = d377
			ps550.OverlayValues[378] = d378
			ps550.OverlayValues[379] = d379
			ps550.OverlayValues[380] = d380
			ps550.OverlayValues[381] = d381
			ps550.OverlayValues[382] = d382
			ps550.OverlayValues[383] = d383
			ps550.OverlayValues[384] = d384
			ps550.OverlayValues[385] = d385
			ps550.OverlayValues[386] = d386
			ps550.OverlayValues[387] = d387
			ps550.OverlayValues[388] = d388
			ps550.OverlayValues[389] = d389
			ps550.OverlayValues[390] = d390
			ps550.OverlayValues[391] = d391
			ps550.OverlayValues[392] = d392
			ps550.OverlayValues[393] = d393
			ps550.OverlayValues[537] = d537
			ps550.OverlayValues[538] = d538
			ps550.OverlayValues[539] = d539
			ps550.OverlayValues[541] = d541
			ps550.OverlayValues[542] = d542
			ps550.OverlayValues[543] = d543
			ps550.OverlayValues[544] = d544
			ps550.OverlayValues[545] = d545
			ps550.OverlayValues[546] = d546
			ps550.OverlayValues[547] = d547
			ps550.OverlayValues[549] = d549
			return bbs[10].RenderPS(ps550)
		}
		if !ps.General {
			if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d551 := ps.PhiValues[0]
				ctx.EnsureDesc(&d551)
				ctx.EmitStoreToStack(d551, int32(bbs[8].PhiBase)+int32(0))
			}
			if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d552 := ps.PhiValues[1]
				ctx.EnsureDesc(&d552)
				ctx.EmitStoreToStack(d552, int32(bbs[8].PhiBase)+int32(16))
			}
			ps.General = true
			return bbs[8].RenderPS(ps)
		}
		lbl26 := ctx.ReserveLabel()
		lbl27 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d545.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl26)
		ctx.EmitJmp(lbl27)
		ctx.MarkLabel(lbl26)
		ctx.SyncDesc(&d8)
		if d8.Loc == scm.LocReg {
			ctx.ProtectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.ProtectReg(d8.Reg)
			ctx.ProtectReg(d8.Reg2)
		}
		d553 = d8
		if d553.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		ctx.EnsureDesc(&d553)
		d554 = d553
		if d554.Loc == scm.LocImm {
			d554 = scm.JITValueDesc{Loc: scm.LocImm, Type: d554.Type, Imm: scm.NewInt(int64(uint64(d554.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d554.Reg, 32)
			ctx.EmitShrRegImm8(d554.Reg, 32)
		}
		ctx.EmitStoreToStack(d554, int32(bbs[2].PhiBase)+int32(0))
		if d8.Loc == scm.LocReg {
			ctx.UnprotectReg(d8.Reg)
		} else if d8.Loc == scm.LocRegPair {
			ctx.UnprotectReg(d8.Reg)
			ctx.UnprotectReg(d8.Reg2)
		}
		ctx.EmitJmp(lbl3)
		ctx.MarkLabel(lbl27)
		ctx.EmitJmp(lbl11)
		ps555 := scm.PhiState{General: true}
		ps555.OverlayValues = make([]scm.JITValueDesc, 555)
		ps555.OverlayValues[1] = d1
		ps555.OverlayValues[2] = d2
		ps555.OverlayValues[3] = d3
		ps555.OverlayValues[4] = d4
		ps555.OverlayValues[5] = d5
		ps555.OverlayValues[6] = d6
		ps555.OverlayValues[7] = d7
		ps555.OverlayValues[8] = d8
		ps555.OverlayValues[9] = d9
		ps555.OverlayValues[10] = d10
		ps555.OverlayValues[11] = d11
		ps555.OverlayValues[12] = d12
		ps555.OverlayValues[13] = d13
		ps555.OverlayValues[14] = d14
		ps555.OverlayValues[15] = d15
		ps555.OverlayValues[17] = d17
		ps555.OverlayValues[18] = d18
		ps555.OverlayValues[19] = d19
		ps555.OverlayValues[20] = d20
		ps555.OverlayValues[21] = d21
		ps555.OverlayValues[22] = d22
		ps555.OverlayValues[23] = d23
		ps555.OverlayValues[24] = d24
		ps555.OverlayValues[25] = d25
		ps555.OverlayValues[26] = d26
		ps555.OverlayValues[27] = d27
		ps555.OverlayValues[28] = d28
		ps555.OverlayValues[29] = d29
		ps555.OverlayValues[30] = d30
		ps555.OverlayValues[31] = d31
		ps555.OverlayValues[32] = d32
		ps555.OverlayValues[33] = d33
		ps555.OverlayValues[34] = d34
		ps555.OverlayValues[35] = d35
		ps555.OverlayValues[36] = d36
		ps555.OverlayValues[37] = d37
		ps555.OverlayValues[38] = d38
		ps555.OverlayValues[39] = d39
		ps555.OverlayValues[40] = d40
		ps555.OverlayValues[41] = d41
		ps555.OverlayValues[42] = d42
		ps555.OverlayValues[43] = d43
		ps555.OverlayValues[44] = d44
		ps555.OverlayValues[45] = d45
		ps555.OverlayValues[46] = d46
		ps555.OverlayValues[47] = d47
		ps555.OverlayValues[48] = d48
		ps555.OverlayValues[49] = d49
		ps555.OverlayValues[50] = d50
		ps555.OverlayValues[53] = d53
		ps555.OverlayValues[54] = d54
		ps555.OverlayValues[55] = d55
		ps555.OverlayValues[111] = d111
		ps555.OverlayValues[112] = d112
		ps555.OverlayValues[113] = d113
		ps555.OverlayValues[114] = d114
		ps555.OverlayValues[115] = d115
		ps555.OverlayValues[116] = d116
		ps555.OverlayValues[117] = d117
		ps555.OverlayValues[118] = d118
		ps555.OverlayValues[119] = d119
		ps555.OverlayValues[120] = d120
		ps555.OverlayValues[121] = d121
		ps555.OverlayValues[122] = d122
		ps555.OverlayValues[123] = d123
		ps555.OverlayValues[124] = d124
		ps555.OverlayValues[125] = d125
		ps555.OverlayValues[126] = d126
		ps555.OverlayValues[127] = d127
		ps555.OverlayValues[128] = d128
		ps555.OverlayValues[129] = d129
		ps555.OverlayValues[130] = d130
		ps555.OverlayValues[131] = d131
		ps555.OverlayValues[132] = d132
		ps555.OverlayValues[133] = d133
		ps555.OverlayValues[134] = d134
		ps555.OverlayValues[135] = d135
		ps555.OverlayValues[136] = d136
		ps555.OverlayValues[137] = d137
		ps555.OverlayValues[138] = d138
		ps555.OverlayValues[139] = d139
		ps555.OverlayValues[140] = d140
		ps555.OverlayValues[143] = d143
		ps555.OverlayValues[230] = d230
		ps555.OverlayValues[231] = d231
		ps555.OverlayValues[232] = d232
		ps555.OverlayValues[233] = d233
		ps555.OverlayValues[235] = d235
		ps555.OverlayValues[236] = d236
		ps555.OverlayValues[237] = d237
		ps555.OverlayValues[238] = d238
		ps555.OverlayValues[239] = d239
		ps555.OverlayValues[240] = d240
		ps555.OverlayValues[241] = d241
		ps555.OverlayValues[242] = d242
		ps555.OverlayValues[244] = d244
		ps555.OverlayValues[246] = d246
		ps555.OverlayValues[247] = d247
		ps555.OverlayValues[248] = d248
		ps555.OverlayValues[249] = d249
		ps555.OverlayValues[250] = d250
		ps555.OverlayValues[253] = d253
		ps555.OverlayValues[357] = d357
		ps555.OverlayValues[358] = d358
		ps555.OverlayValues[359] = d359
		ps555.OverlayValues[360] = d360
		ps555.OverlayValues[361] = d361
		ps555.OverlayValues[363] = d363
		ps555.OverlayValues[364] = d364
		ps555.OverlayValues[365] = d365
		ps555.OverlayValues[366] = d366
		ps555.OverlayValues[367] = d367
		ps555.OverlayValues[368] = d368
		ps555.OverlayValues[369] = d369
		ps555.OverlayValues[370] = d370
		ps555.OverlayValues[371] = d371
		ps555.OverlayValues[372] = d372
		ps555.OverlayValues[373] = d373
		ps555.OverlayValues[374] = d374
		ps555.OverlayValues[375] = d375
		ps555.OverlayValues[376] = d376
		ps555.OverlayValues[377] = d377
		ps555.OverlayValues[378] = d378
		ps555.OverlayValues[379] = d379
		ps555.OverlayValues[380] = d380
		ps555.OverlayValues[381] = d381
		ps555.OverlayValues[382] = d382
		ps555.OverlayValues[383] = d383
		ps555.OverlayValues[384] = d384
		ps555.OverlayValues[385] = d385
		ps555.OverlayValues[386] = d386
		ps555.OverlayValues[387] = d387
		ps555.OverlayValues[388] = d388
		ps555.OverlayValues[389] = d389
		ps555.OverlayValues[390] = d390
		ps555.OverlayValues[391] = d391
		ps555.OverlayValues[392] = d392
		ps555.OverlayValues[393] = d393
		ps555.OverlayValues[537] = d537
		ps555.OverlayValues[538] = d538
		ps555.OverlayValues[539] = d539
		ps555.OverlayValues[541] = d541
		ps555.OverlayValues[542] = d542
		ps555.OverlayValues[543] = d543
		ps555.OverlayValues[544] = d544
		ps555.OverlayValues[545] = d545
		ps555.OverlayValues[546] = d546
		ps555.OverlayValues[547] = d547
		ps555.OverlayValues[549] = d549
		ps555.OverlayValues[551] = d551
		ps555.OverlayValues[552] = d552
		ps555.OverlayValues[553] = d553
		ps555.OverlayValues[554] = d554
		ps555.PhiValues = make([]scm.JITValueDesc, 1)
		d557 = d8
		ps555.PhiValues[0] = d557
		ps556 := scm.PhiState{General: true}
		ps556.OverlayValues = make([]scm.JITValueDesc, 558)
		ps556.OverlayValues[1] = d1
		ps556.OverlayValues[2] = d2
		ps556.OverlayValues[3] = d3
		ps556.OverlayValues[4] = d4
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
		ps556.OverlayValues[17] = d17
		ps556.OverlayValues[18] = d18
		ps556.OverlayValues[19] = d19
		ps556.OverlayValues[20] = d20
		ps556.OverlayValues[21] = d21
		ps556.OverlayValues[22] = d22
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
		ps556.OverlayValues[53] = d53
		ps556.OverlayValues[54] = d54
		ps556.OverlayValues[55] = d55
		ps556.OverlayValues[111] = d111
		ps556.OverlayValues[112] = d112
		ps556.OverlayValues[113] = d113
		ps556.OverlayValues[114] = d114
		ps556.OverlayValues[115] = d115
		ps556.OverlayValues[116] = d116
		ps556.OverlayValues[117] = d117
		ps556.OverlayValues[118] = d118
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
		ps556.OverlayValues[143] = d143
		ps556.OverlayValues[230] = d230
		ps556.OverlayValues[231] = d231
		ps556.OverlayValues[232] = d232
		ps556.OverlayValues[233] = d233
		ps556.OverlayValues[235] = d235
		ps556.OverlayValues[236] = d236
		ps556.OverlayValues[237] = d237
		ps556.OverlayValues[238] = d238
		ps556.OverlayValues[239] = d239
		ps556.OverlayValues[240] = d240
		ps556.OverlayValues[241] = d241
		ps556.OverlayValues[242] = d242
		ps556.OverlayValues[244] = d244
		ps556.OverlayValues[246] = d246
		ps556.OverlayValues[247] = d247
		ps556.OverlayValues[248] = d248
		ps556.OverlayValues[249] = d249
		ps556.OverlayValues[250] = d250
		ps556.OverlayValues[253] = d253
		ps556.OverlayValues[357] = d357
		ps556.OverlayValues[358] = d358
		ps556.OverlayValues[359] = d359
		ps556.OverlayValues[360] = d360
		ps556.OverlayValues[361] = d361
		ps556.OverlayValues[363] = d363
		ps556.OverlayValues[364] = d364
		ps556.OverlayValues[365] = d365
		ps556.OverlayValues[366] = d366
		ps556.OverlayValues[367] = d367
		ps556.OverlayValues[368] = d368
		ps556.OverlayValues[369] = d369
		ps556.OverlayValues[370] = d370
		ps556.OverlayValues[371] = d371
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
		ps556.OverlayValues[537] = d537
		ps556.OverlayValues[538] = d538
		ps556.OverlayValues[539] = d539
		ps556.OverlayValues[541] = d541
		ps556.OverlayValues[542] = d542
		ps556.OverlayValues[543] = d543
		ps556.OverlayValues[544] = d544
		ps556.OverlayValues[545] = d545
		ps556.OverlayValues[546] = d546
		ps556.OverlayValues[547] = d547
		ps556.OverlayValues[549] = d549
		ps556.OverlayValues[551] = d551
		ps556.OverlayValues[552] = d552
		ps556.OverlayValues[553] = d553
		ps556.OverlayValues[554] = d554
		ps556.OverlayValues[557] = d557
		snap558 := d1
		snap559 := d2
		snap560 := d3
		snap561 := d4
		snap562 := d5
		snap563 := d6
		snap564 := d7
		snap565 := d8
		snap566 := d9
		snap567 := d10
		snap568 := d11
		snap569 := d12
		snap570 := d13
		snap571 := d14
		snap572 := d15
		snap573 := d17
		snap574 := d18
		snap575 := d19
		snap576 := d20
		snap577 := d21
		snap578 := d22
		snap579 := d23
		snap580 := d24
		snap581 := d25
		snap582 := d26
		snap583 := d27
		snap584 := d28
		snap585 := d29
		snap586 := d30
		snap587 := d31
		snap588 := d32
		snap589 := d33
		snap590 := d34
		snap591 := d35
		snap592 := d36
		snap593 := d37
		snap594 := d38
		snap595 := d39
		snap596 := d40
		snap597 := d41
		snap598 := d42
		snap599 := d43
		snap600 := d44
		snap601 := d45
		snap602 := d46
		snap603 := d47
		snap604 := d48
		snap605 := d49
		snap606 := d50
		snap607 := d53
		snap608 := d54
		snap609 := d55
		snap610 := d111
		snap611 := d112
		snap612 := d113
		snap613 := d114
		snap614 := d115
		snap615 := d116
		snap616 := d117
		snap617 := d118
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
		snap640 := d143
		snap641 := d230
		snap642 := d231
		snap643 := d232
		snap644 := d233
		snap645 := d235
		snap646 := d236
		snap647 := d237
		snap648 := d238
		snap649 := d239
		snap650 := d240
		snap651 := d241
		snap652 := d242
		snap653 := d244
		snap654 := d246
		snap655 := d247
		snap656 := d248
		snap657 := d249
		snap658 := d250
		snap659 := d253
		snap660 := d357
		snap661 := d358
		snap662 := d359
		snap663 := d360
		snap664 := d361
		snap665 := d363
		snap666 := d364
		snap667 := d365
		snap668 := d366
		snap669 := d367
		snap670 := d368
		snap671 := d369
		snap672 := d370
		snap673 := d371
		snap674 := d372
		snap675 := d373
		snap676 := d374
		snap677 := d375
		snap678 := d376
		snap679 := d377
		snap680 := d378
		snap681 := d379
		snap682 := d380
		snap683 := d381
		snap684 := d382
		snap685 := d383
		snap686 := d384
		snap687 := d385
		snap688 := d386
		snap689 := d387
		snap690 := d388
		snap691 := d389
		snap692 := d390
		snap693 := d391
		snap694 := d392
		snap695 := d393
		snap696 := d537
		snap697 := d538
		snap698 := d539
		snap699 := d541
		snap700 := d542
		snap701 := d543
		snap702 := d544
		snap703 := d545
		snap704 := d546
		snap705 := d547
		snap706 := d549
		snap707 := d551
		snap708 := d552
		snap709 := d553
		snap710 := d554
		snap711 := d557
		alloc712 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps555)
		}
		ctx.RestoreAllocState(alloc712)
		d1 = snap558
		d2 = snap559
		d3 = snap560
		d4 = snap561
		d5 = snap562
		d6 = snap563
		d7 = snap564
		d8 = snap565
		d9 = snap566
		d10 = snap567
		d11 = snap568
		d12 = snap569
		d13 = snap570
		d14 = snap571
		d15 = snap572
		d17 = snap573
		d18 = snap574
		d19 = snap575
		d20 = snap576
		d21 = snap577
		d22 = snap578
		d23 = snap579
		d24 = snap580
		d25 = snap581
		d26 = snap582
		d27 = snap583
		d28 = snap584
		d29 = snap585
		d30 = snap586
		d31 = snap587
		d32 = snap588
		d33 = snap589
		d34 = snap590
		d35 = snap591
		d36 = snap592
		d37 = snap593
		d38 = snap594
		d39 = snap595
		d40 = snap596
		d41 = snap597
		d42 = snap598
		d43 = snap599
		d44 = snap600
		d45 = snap601
		d46 = snap602
		d47 = snap603
		d48 = snap604
		d49 = snap605
		d50 = snap606
		d53 = snap607
		d54 = snap608
		d55 = snap609
		d111 = snap610
		d112 = snap611
		d113 = snap612
		d114 = snap613
		d115 = snap614
		d116 = snap615
		d117 = snap616
		d118 = snap617
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
		d143 = snap640
		d230 = snap641
		d231 = snap642
		d232 = snap643
		d233 = snap644
		d235 = snap645
		d236 = snap646
		d237 = snap647
		d238 = snap648
		d239 = snap649
		d240 = snap650
		d241 = snap651
		d242 = snap652
		d244 = snap653
		d246 = snap654
		d247 = snap655
		d248 = snap656
		d249 = snap657
		d250 = snap658
		d253 = snap659
		d357 = snap660
		d358 = snap661
		d359 = snap662
		d360 = snap663
		d361 = snap664
		d363 = snap665
		d364 = snap666
		d365 = snap667
		d366 = snap668
		d367 = snap669
		d368 = snap670
		d369 = snap671
		d370 = snap672
		d371 = snap673
		d372 = snap674
		d373 = snap675
		d374 = snap676
		d375 = snap677
		d376 = snap678
		d377 = snap679
		d378 = snap680
		d379 = snap681
		d380 = snap682
		d381 = snap683
		d382 = snap684
		d383 = snap685
		d384 = snap686
		d385 = snap687
		d386 = snap688
		d387 = snap689
		d388 = snap690
		d389 = snap691
		d390 = snap692
		d391 = snap693
		d392 = snap694
		d393 = snap695
		d537 = snap696
		d538 = snap697
		d539 = snap698
		d541 = snap699
		d542 = snap700
		d543 = snap701
		d544 = snap702
		d545 = snap703
		d546 = snap704
		d547 = snap705
		d549 = snap706
		d551 = snap707
		d552 = snap708
		d553 = snap709
		d554 = snap710
		d557 = snap711
		if !bbs[10].Rendered {
			return bbs[10].RenderPS(ps556)
		}
		return result
		ctx.FreeDesc(&d544)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
			d365 = ps.OverlayValues[365]
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
		if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
			d371 = ps.OverlayValues[371]
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
		if len(ps.OverlayValues) > 537 && ps.OverlayValues[537].Loc != scm.LocNone {
			d537 = ps.OverlayValues[537]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 542 && ps.OverlayValues[542].Loc != scm.LocNone {
			d542 = ps.OverlayValues[542]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
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
		if len(ps.OverlayValues) > 554 && ps.OverlayValues[554].Loc != scm.LocNone {
			d554 = ps.OverlayValues[554]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		ctx.ReclaimUntrackedRegs()
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
			d713 = d5
			if d713.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d713)
			d714 = d713
			if d714.Loc == scm.LocImm {
				d714 = scm.JITValueDesc{Loc: scm.LocImm, Type: d714.Type, Imm: scm.NewInt(int64(uint64(d714.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d714.Reg, 32)
				ctx.EmitShrRegImm8(d714.Reg, 32)
			}
			ctx.EmitStoreToStack(d714, int32(bbs[8].PhiBase)+int32(0))
			d715 = d7
			if d715.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d715)
			d716 = d715
			if d716.Loc == scm.LocImm {
				d716 = scm.JITValueDesc{Loc: scm.LocImm, Type: d716.Type, Imm: scm.NewInt(int64(uint64(d716.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d716.Reg, 32)
				ctx.EmitShrRegImm8(d716.Reg, 32)
			}
			ctx.EmitStoreToStack(d716, int32(bbs[8].PhiBase)+int32(16))
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
		ps717 := scm.PhiState{General: ps.General}
		ps717.OverlayValues = make([]scm.JITValueDesc, 717)
		ps717.OverlayValues[1] = d1
		ps717.OverlayValues[2] = d2
		ps717.OverlayValues[3] = d3
		ps717.OverlayValues[4] = d4
		ps717.OverlayValues[5] = d5
		ps717.OverlayValues[6] = d6
		ps717.OverlayValues[7] = d7
		ps717.OverlayValues[8] = d8
		ps717.OverlayValues[9] = d9
		ps717.OverlayValues[10] = d10
		ps717.OverlayValues[11] = d11
		ps717.OverlayValues[12] = d12
		ps717.OverlayValues[13] = d13
		ps717.OverlayValues[14] = d14
		ps717.OverlayValues[15] = d15
		ps717.OverlayValues[17] = d17
		ps717.OverlayValues[18] = d18
		ps717.OverlayValues[19] = d19
		ps717.OverlayValues[20] = d20
		ps717.OverlayValues[21] = d21
		ps717.OverlayValues[22] = d22
		ps717.OverlayValues[23] = d23
		ps717.OverlayValues[24] = d24
		ps717.OverlayValues[25] = d25
		ps717.OverlayValues[26] = d26
		ps717.OverlayValues[27] = d27
		ps717.OverlayValues[28] = d28
		ps717.OverlayValues[29] = d29
		ps717.OverlayValues[30] = d30
		ps717.OverlayValues[31] = d31
		ps717.OverlayValues[32] = d32
		ps717.OverlayValues[33] = d33
		ps717.OverlayValues[34] = d34
		ps717.OverlayValues[35] = d35
		ps717.OverlayValues[36] = d36
		ps717.OverlayValues[37] = d37
		ps717.OverlayValues[38] = d38
		ps717.OverlayValues[39] = d39
		ps717.OverlayValues[40] = d40
		ps717.OverlayValues[41] = d41
		ps717.OverlayValues[42] = d42
		ps717.OverlayValues[43] = d43
		ps717.OverlayValues[44] = d44
		ps717.OverlayValues[45] = d45
		ps717.OverlayValues[46] = d46
		ps717.OverlayValues[47] = d47
		ps717.OverlayValues[48] = d48
		ps717.OverlayValues[49] = d49
		ps717.OverlayValues[50] = d50
		ps717.OverlayValues[53] = d53
		ps717.OverlayValues[54] = d54
		ps717.OverlayValues[55] = d55
		ps717.OverlayValues[111] = d111
		ps717.OverlayValues[112] = d112
		ps717.OverlayValues[113] = d113
		ps717.OverlayValues[114] = d114
		ps717.OverlayValues[115] = d115
		ps717.OverlayValues[116] = d116
		ps717.OverlayValues[117] = d117
		ps717.OverlayValues[118] = d118
		ps717.OverlayValues[119] = d119
		ps717.OverlayValues[120] = d120
		ps717.OverlayValues[121] = d121
		ps717.OverlayValues[122] = d122
		ps717.OverlayValues[123] = d123
		ps717.OverlayValues[124] = d124
		ps717.OverlayValues[125] = d125
		ps717.OverlayValues[126] = d126
		ps717.OverlayValues[127] = d127
		ps717.OverlayValues[128] = d128
		ps717.OverlayValues[129] = d129
		ps717.OverlayValues[130] = d130
		ps717.OverlayValues[131] = d131
		ps717.OverlayValues[132] = d132
		ps717.OverlayValues[133] = d133
		ps717.OverlayValues[134] = d134
		ps717.OverlayValues[135] = d135
		ps717.OverlayValues[136] = d136
		ps717.OverlayValues[137] = d137
		ps717.OverlayValues[138] = d138
		ps717.OverlayValues[139] = d139
		ps717.OverlayValues[140] = d140
		ps717.OverlayValues[143] = d143
		ps717.OverlayValues[230] = d230
		ps717.OverlayValues[231] = d231
		ps717.OverlayValues[232] = d232
		ps717.OverlayValues[233] = d233
		ps717.OverlayValues[235] = d235
		ps717.OverlayValues[236] = d236
		ps717.OverlayValues[237] = d237
		ps717.OverlayValues[238] = d238
		ps717.OverlayValues[239] = d239
		ps717.OverlayValues[240] = d240
		ps717.OverlayValues[241] = d241
		ps717.OverlayValues[242] = d242
		ps717.OverlayValues[244] = d244
		ps717.OverlayValues[246] = d246
		ps717.OverlayValues[247] = d247
		ps717.OverlayValues[248] = d248
		ps717.OverlayValues[249] = d249
		ps717.OverlayValues[250] = d250
		ps717.OverlayValues[253] = d253
		ps717.OverlayValues[357] = d357
		ps717.OverlayValues[358] = d358
		ps717.OverlayValues[359] = d359
		ps717.OverlayValues[360] = d360
		ps717.OverlayValues[361] = d361
		ps717.OverlayValues[363] = d363
		ps717.OverlayValues[364] = d364
		ps717.OverlayValues[365] = d365
		ps717.OverlayValues[366] = d366
		ps717.OverlayValues[367] = d367
		ps717.OverlayValues[368] = d368
		ps717.OverlayValues[369] = d369
		ps717.OverlayValues[370] = d370
		ps717.OverlayValues[371] = d371
		ps717.OverlayValues[372] = d372
		ps717.OverlayValues[373] = d373
		ps717.OverlayValues[374] = d374
		ps717.OverlayValues[375] = d375
		ps717.OverlayValues[376] = d376
		ps717.OverlayValues[377] = d377
		ps717.OverlayValues[378] = d378
		ps717.OverlayValues[379] = d379
		ps717.OverlayValues[380] = d380
		ps717.OverlayValues[381] = d381
		ps717.OverlayValues[382] = d382
		ps717.OverlayValues[383] = d383
		ps717.OverlayValues[384] = d384
		ps717.OverlayValues[385] = d385
		ps717.OverlayValues[386] = d386
		ps717.OverlayValues[387] = d387
		ps717.OverlayValues[388] = d388
		ps717.OverlayValues[389] = d389
		ps717.OverlayValues[390] = d390
		ps717.OverlayValues[391] = d391
		ps717.OverlayValues[392] = d392
		ps717.OverlayValues[393] = d393
		ps717.OverlayValues[537] = d537
		ps717.OverlayValues[538] = d538
		ps717.OverlayValues[539] = d539
		ps717.OverlayValues[541] = d541
		ps717.OverlayValues[542] = d542
		ps717.OverlayValues[543] = d543
		ps717.OverlayValues[544] = d544
		ps717.OverlayValues[545] = d545
		ps717.OverlayValues[546] = d546
		ps717.OverlayValues[547] = d547
		ps717.OverlayValues[549] = d549
		ps717.OverlayValues[551] = d551
		ps717.OverlayValues[552] = d552
		ps717.OverlayValues[553] = d553
		ps717.OverlayValues[554] = d554
		ps717.OverlayValues[557] = d557
		ps717.OverlayValues[713] = d713
		ps717.OverlayValues[714] = d714
		ps717.OverlayValues[715] = d715
		ps717.OverlayValues[716] = d716
		ps717.PhiValues = make([]scm.JITValueDesc, 2)
		d718 = d5
		ps717.PhiValues[0] = d718
		d719 = d7
		ps717.PhiValues[1] = d719
		if ps717.General && bbs[8].Rendered {
			ctx.EmitJmp(lbl9)
			return result
		}
		return bbs[8].RenderPS(ps717)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
			d365 = ps.OverlayValues[365]
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
		if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
			d371 = ps.OverlayValues[371]
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
		if len(ps.OverlayValues) > 537 && ps.OverlayValues[537].Loc != scm.LocNone {
			d537 = ps.OverlayValues[537]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 542 && ps.OverlayValues[542].Loc != scm.LocNone {
			d542 = ps.OverlayValues[542]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
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
		if len(ps.OverlayValues) > 554 && ps.OverlayValues[554].Loc != scm.LocNone {
			d554 = ps.OverlayValues[554]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 713 && ps.OverlayValues[713].Loc != scm.LocNone {
			d713 = ps.OverlayValues[713]
		}
		if len(ps.OverlayValues) > 714 && ps.OverlayValues[714].Loc != scm.LocNone {
			d714 = ps.OverlayValues[714]
		}
		if len(ps.OverlayValues) > 715 && ps.OverlayValues[715].Loc != scm.LocNone {
			d715 = ps.OverlayValues[715]
		}
		if len(ps.OverlayValues) > 716 && ps.OverlayValues[716].Loc != scm.LocNone {
			d716 = ps.OverlayValues[716]
		}
		if len(ps.OverlayValues) > 718 && ps.OverlayValues[718].Loc != scm.LocNone {
			d718 = ps.OverlayValues[718]
		}
		if len(ps.OverlayValues) > 719 && ps.OverlayValues[719].Loc != scm.LocNone {
			d719 = ps.OverlayValues[719]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d9)
		ctx.EnsureDescsTogether(&d8, &d9)
		var d720 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
			d720 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + d9.Imm.Int())}
		} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
			r105 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r105, d8.Reg)
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			ctx.BindReg(r105, &d720)
		} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			ctx.BindReg(d9.Reg, &d720)
		} else if d8.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
			ctx.EmitAddInt64(scratch, d9.Reg)
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d720)
		} else if d9.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(scratch, d8.Reg)
			if d9.Imm.Int() >= -2147483648 && d9.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d9.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d9.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d720)
		} else {
			r106 := ctx.AllocRegExcept(d8.Reg, d9.Reg)
			ctx.EmitMovRegReg(r106, d8.Reg)
			ctx.EmitAddInt64(r106, d9.Reg)
			d720 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			ctx.BindReg(r106, &d720)
		}
		if d720.Loc == scm.LocImm {
			d720 = scm.JITValueDesc{Loc: scm.LocImm, Type: d720.Type, Imm: scm.NewInt(int64(uint64(d720.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d720.Reg, 32)
			ctx.EmitShrRegImm8(d720.Reg, 32)
		}
		if d720.Loc == scm.LocReg && d8.Loc == scm.LocReg && d720.Reg == d8.Reg {
			ctx.TransferReg(d8.Reg)
			d8.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d720)
		var d721 scm.JITValueDesc
		if d720.Loc == scm.LocImm {
			d721 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d720.Imm.Int() / 2)}
		} else {
			r107 := ctx.AllocRegExcept(d720.Reg)
			ctx.EmitMovRegReg(r107, d720.Reg)
			ctx.EmitShrRegImm8(r107, 1)
			d721 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			ctx.BindReg(r107, &d721)
		}
		if d721.Loc == scm.LocImm {
			d721 = scm.JITValueDesc{Loc: scm.LocImm, Type: d721.Type, Imm: scm.NewInt(int64(uint64(d721.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.EmitShlRegImm8(d721.Reg, 32)
			ctx.EmitShrRegImm8(d721.Reg, 32)
		}
		if d721.Loc == scm.LocReg && d720.Loc == scm.LocReg && d721.Reg == d720.Reg {
			ctx.TransferReg(d720.Reg)
			d720.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d721)
		ctx.EmitStoreToStack(d721, int32(bbs[1].PhiBase)+int32(0))
		ctx.StabilizeDescForControlFlow(&d721)
		ctx.FreeDesc(&d720)
		if ps.General {
			ctx.SyncDesc(&d8)
			if d8.Loc == scm.LocReg {
				ctx.ProtectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.ProtectReg(d8.Reg)
				ctx.ProtectReg(d8.Reg2)
			}
			ctx.SyncDesc(&d9)
			if d9.Loc == scm.LocReg {
				ctx.ProtectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.ProtectReg(d9.Reg)
				ctx.ProtectReg(d9.Reg2)
			}
			d722 = d8
			if d722.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d722)
			d723 = d722
			if d723.Loc == scm.LocImm {
				d723 = scm.JITValueDesc{Loc: scm.LocImm, Type: d723.Type, Imm: scm.NewInt(int64(uint64(d723.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d723.Reg, 32)
				ctx.EmitShrRegImm8(d723.Reg, 32)
			}
			ctx.EmitStoreToStack(d723, int32(bbs[1].PhiBase)+int32(16))
			d724 = d9
			if d724.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d724)
			d725 = d724
			if d725.Loc == scm.LocImm {
				d725 = scm.JITValueDesc{Loc: scm.LocImm, Type: d725.Type, Imm: scm.NewInt(int64(uint64(d725.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d725.Reg, 32)
				ctx.EmitShrRegImm8(d725.Reg, 32)
			}
			ctx.EmitStoreToStack(d725, int32(bbs[1].PhiBase)+int32(32))
			if d8.Loc == scm.LocReg {
				ctx.UnprotectReg(d8.Reg)
			} else if d8.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d8.Reg)
				ctx.UnprotectReg(d8.Reg2)
			}
			if d9.Loc == scm.LocReg {
				ctx.UnprotectReg(d9.Reg)
			} else if d9.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d9.Reg)
				ctx.UnprotectReg(d9.Reg2)
			}
		}
		ps726 := scm.PhiState{General: ps.General}
		ps726.OverlayValues = make([]scm.JITValueDesc, 726)
		ps726.OverlayValues[1] = d1
		ps726.OverlayValues[2] = d2
		ps726.OverlayValues[3] = d3
		ps726.OverlayValues[4] = d4
		ps726.OverlayValues[5] = d5
		ps726.OverlayValues[6] = d6
		ps726.OverlayValues[7] = d7
		ps726.OverlayValues[8] = d8
		ps726.OverlayValues[9] = d9
		ps726.OverlayValues[10] = d10
		ps726.OverlayValues[11] = d11
		ps726.OverlayValues[12] = d12
		ps726.OverlayValues[13] = d13
		ps726.OverlayValues[14] = d14
		ps726.OverlayValues[15] = d15
		ps726.OverlayValues[17] = d17
		ps726.OverlayValues[18] = d18
		ps726.OverlayValues[19] = d19
		ps726.OverlayValues[20] = d20
		ps726.OverlayValues[21] = d21
		ps726.OverlayValues[22] = d22
		ps726.OverlayValues[23] = d23
		ps726.OverlayValues[24] = d24
		ps726.OverlayValues[25] = d25
		ps726.OverlayValues[26] = d26
		ps726.OverlayValues[27] = d27
		ps726.OverlayValues[28] = d28
		ps726.OverlayValues[29] = d29
		ps726.OverlayValues[30] = d30
		ps726.OverlayValues[31] = d31
		ps726.OverlayValues[32] = d32
		ps726.OverlayValues[33] = d33
		ps726.OverlayValues[34] = d34
		ps726.OverlayValues[35] = d35
		ps726.OverlayValues[36] = d36
		ps726.OverlayValues[37] = d37
		ps726.OverlayValues[38] = d38
		ps726.OverlayValues[39] = d39
		ps726.OverlayValues[40] = d40
		ps726.OverlayValues[41] = d41
		ps726.OverlayValues[42] = d42
		ps726.OverlayValues[43] = d43
		ps726.OverlayValues[44] = d44
		ps726.OverlayValues[45] = d45
		ps726.OverlayValues[46] = d46
		ps726.OverlayValues[47] = d47
		ps726.OverlayValues[48] = d48
		ps726.OverlayValues[49] = d49
		ps726.OverlayValues[50] = d50
		ps726.OverlayValues[53] = d53
		ps726.OverlayValues[54] = d54
		ps726.OverlayValues[55] = d55
		ps726.OverlayValues[111] = d111
		ps726.OverlayValues[112] = d112
		ps726.OverlayValues[113] = d113
		ps726.OverlayValues[114] = d114
		ps726.OverlayValues[115] = d115
		ps726.OverlayValues[116] = d116
		ps726.OverlayValues[117] = d117
		ps726.OverlayValues[118] = d118
		ps726.OverlayValues[119] = d119
		ps726.OverlayValues[120] = d120
		ps726.OverlayValues[121] = d121
		ps726.OverlayValues[122] = d122
		ps726.OverlayValues[123] = d123
		ps726.OverlayValues[124] = d124
		ps726.OverlayValues[125] = d125
		ps726.OverlayValues[126] = d126
		ps726.OverlayValues[127] = d127
		ps726.OverlayValues[128] = d128
		ps726.OverlayValues[129] = d129
		ps726.OverlayValues[130] = d130
		ps726.OverlayValues[131] = d131
		ps726.OverlayValues[132] = d132
		ps726.OverlayValues[133] = d133
		ps726.OverlayValues[134] = d134
		ps726.OverlayValues[135] = d135
		ps726.OverlayValues[136] = d136
		ps726.OverlayValues[137] = d137
		ps726.OverlayValues[138] = d138
		ps726.OverlayValues[139] = d139
		ps726.OverlayValues[140] = d140
		ps726.OverlayValues[143] = d143
		ps726.OverlayValues[230] = d230
		ps726.OverlayValues[231] = d231
		ps726.OverlayValues[232] = d232
		ps726.OverlayValues[233] = d233
		ps726.OverlayValues[235] = d235
		ps726.OverlayValues[236] = d236
		ps726.OverlayValues[237] = d237
		ps726.OverlayValues[238] = d238
		ps726.OverlayValues[239] = d239
		ps726.OverlayValues[240] = d240
		ps726.OverlayValues[241] = d241
		ps726.OverlayValues[242] = d242
		ps726.OverlayValues[244] = d244
		ps726.OverlayValues[246] = d246
		ps726.OverlayValues[247] = d247
		ps726.OverlayValues[248] = d248
		ps726.OverlayValues[249] = d249
		ps726.OverlayValues[250] = d250
		ps726.OverlayValues[253] = d253
		ps726.OverlayValues[357] = d357
		ps726.OverlayValues[358] = d358
		ps726.OverlayValues[359] = d359
		ps726.OverlayValues[360] = d360
		ps726.OverlayValues[361] = d361
		ps726.OverlayValues[363] = d363
		ps726.OverlayValues[364] = d364
		ps726.OverlayValues[365] = d365
		ps726.OverlayValues[366] = d366
		ps726.OverlayValues[367] = d367
		ps726.OverlayValues[368] = d368
		ps726.OverlayValues[369] = d369
		ps726.OverlayValues[370] = d370
		ps726.OverlayValues[371] = d371
		ps726.OverlayValues[372] = d372
		ps726.OverlayValues[373] = d373
		ps726.OverlayValues[374] = d374
		ps726.OverlayValues[375] = d375
		ps726.OverlayValues[376] = d376
		ps726.OverlayValues[377] = d377
		ps726.OverlayValues[378] = d378
		ps726.OverlayValues[379] = d379
		ps726.OverlayValues[380] = d380
		ps726.OverlayValues[381] = d381
		ps726.OverlayValues[382] = d382
		ps726.OverlayValues[383] = d383
		ps726.OverlayValues[384] = d384
		ps726.OverlayValues[385] = d385
		ps726.OverlayValues[386] = d386
		ps726.OverlayValues[387] = d387
		ps726.OverlayValues[388] = d388
		ps726.OverlayValues[389] = d389
		ps726.OverlayValues[390] = d390
		ps726.OverlayValues[391] = d391
		ps726.OverlayValues[392] = d392
		ps726.OverlayValues[393] = d393
		ps726.OverlayValues[537] = d537
		ps726.OverlayValues[538] = d538
		ps726.OverlayValues[539] = d539
		ps726.OverlayValues[541] = d541
		ps726.OverlayValues[542] = d542
		ps726.OverlayValues[543] = d543
		ps726.OverlayValues[544] = d544
		ps726.OverlayValues[545] = d545
		ps726.OverlayValues[546] = d546
		ps726.OverlayValues[547] = d547
		ps726.OverlayValues[549] = d549
		ps726.OverlayValues[551] = d551
		ps726.OverlayValues[552] = d552
		ps726.OverlayValues[553] = d553
		ps726.OverlayValues[554] = d554
		ps726.OverlayValues[557] = d557
		ps726.OverlayValues[713] = d713
		ps726.OverlayValues[714] = d714
		ps726.OverlayValues[715] = d715
		ps726.OverlayValues[716] = d716
		ps726.OverlayValues[718] = d718
		ps726.OverlayValues[719] = d719
		ps726.OverlayValues[720] = d720
		ps726.OverlayValues[721] = d721
		ps726.OverlayValues[722] = d722
		ps726.OverlayValues[723] = d723
		ps726.OverlayValues[724] = d724
		ps726.OverlayValues[725] = d725
		ps726.PhiValues = make([]scm.JITValueDesc, 3)
		d727 = d8
		ps726.PhiValues[1] = d727
		d728 = d9
		ps726.PhiValues[2] = d728
		if ps726.General && bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		return bbs[1].RenderPS(ps726)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
			d365 = ps.OverlayValues[365]
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
		if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
			d371 = ps.OverlayValues[371]
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
		if len(ps.OverlayValues) > 537 && ps.OverlayValues[537].Loc != scm.LocNone {
			d537 = ps.OverlayValues[537]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 542 && ps.OverlayValues[542].Loc != scm.LocNone {
			d542 = ps.OverlayValues[542]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
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
		if len(ps.OverlayValues) > 554 && ps.OverlayValues[554].Loc != scm.LocNone {
			d554 = ps.OverlayValues[554]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 713 && ps.OverlayValues[713].Loc != scm.LocNone {
			d713 = ps.OverlayValues[713]
		}
		if len(ps.OverlayValues) > 714 && ps.OverlayValues[714].Loc != scm.LocNone {
			d714 = ps.OverlayValues[714]
		}
		if len(ps.OverlayValues) > 715 && ps.OverlayValues[715].Loc != scm.LocNone {
			d715 = ps.OverlayValues[715]
		}
		if len(ps.OverlayValues) > 716 && ps.OverlayValues[716].Loc != scm.LocNone {
			d716 = ps.OverlayValues[716]
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
		if len(ps.OverlayValues) > 722 && ps.OverlayValues[722].Loc != scm.LocNone {
			d722 = ps.OverlayValues[722]
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
		if len(ps.OverlayValues) > 727 && ps.OverlayValues[727].Loc != scm.LocNone {
			d727 = ps.OverlayValues[727]
		}
		if len(ps.OverlayValues) > 728 && ps.OverlayValues[728].Loc != scm.LocNone {
			d728 = ps.OverlayValues[728]
		}
		ctx.ReclaimUntrackedRegs()
		d729 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d730 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d730)
		ctx.BindReg(r1, &d730)
		ctx.EnsureDesc(&d729)
		if d729.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d729, &d730)
		} else {
			switch d729.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d730, d729)
			case scm.TagInt:
				ctx.EmitMakeInt(d730, d729)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d730, d729)
			case scm.TagNil:
				ctx.EmitMakeNil(d730)
			default:
				ctx.EmitMovPairToResult(&d729, &d730)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
			d365 = ps.OverlayValues[365]
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
		if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
			d371 = ps.OverlayValues[371]
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
		if len(ps.OverlayValues) > 537 && ps.OverlayValues[537].Loc != scm.LocNone {
			d537 = ps.OverlayValues[537]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 542 && ps.OverlayValues[542].Loc != scm.LocNone {
			d542 = ps.OverlayValues[542]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
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
		if len(ps.OverlayValues) > 554 && ps.OverlayValues[554].Loc != scm.LocNone {
			d554 = ps.OverlayValues[554]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 713 && ps.OverlayValues[713].Loc != scm.LocNone {
			d713 = ps.OverlayValues[713]
		}
		if len(ps.OverlayValues) > 714 && ps.OverlayValues[714].Loc != scm.LocNone {
			d714 = ps.OverlayValues[714]
		}
		if len(ps.OverlayValues) > 715 && ps.OverlayValues[715].Loc != scm.LocNone {
			d715 = ps.OverlayValues[715]
		}
		if len(ps.OverlayValues) > 716 && ps.OverlayValues[716].Loc != scm.LocNone {
			d716 = ps.OverlayValues[716]
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
		if len(ps.OverlayValues) > 722 && ps.OverlayValues[722].Loc != scm.LocNone {
			d722 = ps.OverlayValues[722]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		d731 = d4
		_ = d731
		ctx.StabilizeDescForControlFlow(&d731)
		ctx.StabilizeDescForControlFlow(&d4)
		bbpos_4_0 := int32(-1)
		_ = bbpos_4_0
		lbl28 := ctx.ReserveLabel()
		_ = lbl28
		bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl28)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d731)
		ctx.EnsureDesc(&d731)
		var d732 scm.JITValueDesc
		if d731.Loc == scm.LocImm {
			d732 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d731.Imm.Int()))))}
		} else {
			r108 := ctx.AllocReg()
			ctx.EmitMovRegReg(r108, d731.Reg)
			ctx.EmitShlRegImm8(r108, 32)
			ctx.EmitShrRegImm8(r108, 32)
			d732 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			ctx.BindReg(r108, &d732)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d733 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d733 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r109 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r109, thisptr.Reg, off)
			d733 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r109}
			ctx.BindReg(r109, &d733)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d733)
		ctx.EnsureDesc(&d733)
		var d734 scm.JITValueDesc
		if d733.Loc == scm.LocImm {
			d734 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d733.Imm.Int()))))}
		} else {
			r110 := ctx.AllocReg()
			ctx.EmitMovRegReg(r110, d733.Reg)
			ctx.EmitShlRegImm8(r110, 56)
			ctx.EmitShrRegImm8(r110, 56)
			d734 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			ctx.BindReg(r110, &d734)
		}
		ctx.FreeDesc(&d733)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d732)
		ctx.EnsureDesc(&d734)
		ctx.EnsureDescsTogether(&d732, &d734)
		var d735 scm.JITValueDesc
		if d732.Loc == scm.LocImm && d734.Loc == scm.LocImm {
			d735 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d732.Imm.Int() * d734.Imm.Int())}
		} else if d732.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d734.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d732.Imm.Int()))
			ctx.EmitImulInt64(scratch, d734.Reg)
			d735 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d735)
		} else if d734.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d732.Reg)
			ctx.EmitMovRegReg(scratch, d732.Reg)
			if d734.Imm.Int() >= -2147483648 && d734.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d734.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d734.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d735 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d735)
		} else {
			r111 := ctx.AllocRegExcept(d732.Reg, d734.Reg)
			ctx.EmitMovRegReg(r111, d732.Reg)
			ctx.EmitImulInt64(r111, d734.Reg)
			d735 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			ctx.BindReg(r111, &d735)
		}
		if d735.Loc == scm.LocReg && d732.Loc == scm.LocReg && d735.Reg == d732.Reg {
			ctx.TransferReg(d732.Reg)
			d732.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d732)
		ctx.FreeDesc(&d734)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d735)
		var d736 scm.JITValueDesc
		if d735.Loc == scm.LocImm {
			d736 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d735.Imm.Int() / 64)}
		} else {
			r112 := ctx.AllocRegExcept(d735.Reg)
			ctx.EmitMovRegReg(r112, d735.Reg)
			ctx.EmitShrRegImm8(r112, 6)
			d736 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			ctx.BindReg(r112, &d736)
		}
		if d736.Loc == scm.LocReg && d735.Loc == scm.LocReg && d736.Reg == d735.Reg {
			ctx.TransferReg(d735.Reg)
			d735.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d735)
		var d737 scm.JITValueDesc
		if d735.Loc == scm.LocImm {
			d737 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d735.Imm.Int() % 64)}
		} else {
			r113 := ctx.AllocRegExcept(d735.Reg)
			ctx.EmitMovRegReg(r113, d735.Reg)
			ctx.EmitAndRegImm32(r113, 63)
			d737 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
			ctx.BindReg(r113, &d737)
		}
		if d737.Loc == scm.LocReg && d735.Loc == scm.LocReg && d737.Reg == d735.Reg {
			ctx.TransferReg(d735.Reg)
			d735.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d735)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d738 scm.JITValueDesc
		r114 := ctx.AllocReg()
		r115 := ctx.AllocRegExcept(r114)
		r116 := ctx.AllocRegExcept(r114, r115)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r114, uint64(dataPtr))
			ctx.EmitMovRegImm64(r115, uint64(sliceLen))
			ctx.EmitMovRegImm64(r116, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
			ctx.EmitMovRegMem(r114, thisptr.Reg, off)
			ctx.EmitMovRegMem(r115, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r116, thisptr.Reg, off+16)
		}
		d738 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r114, Reg2: r115, Reg3: r116}
		ctx.BindReg(r114, &d738)
		ctx.BindReg(r115, &d738)
		ctx.BindReg(r116, &d738)
		ctx.BindReg(r114, &d738)
		ctx.BindReg(r115, &d738)
		ctx.BindReg(r116, &d738)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d736)
		ctx.ReclaimUntrackedRegs()
		d740 = ctx.EmitSliceElementAddress(&d738, &d736, 8)
		ctx.EnsureDesc(&d740)
		ctx.EmitMovRegMem(d740.Reg, d740.Reg, 0)
		d739 = d740
		d739.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d739)
		ctx.EnsureDesc(&d737)
		ctx.EnsureDescsTogether(&d739, &d737)
		var d741 scm.JITValueDesc
		if d739.Loc == scm.LocImm && d737.Loc == scm.LocImm {
			d741 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d739.Imm.Int()) << uint64(d737.Imm.Int())))}
		} else if d737.Loc == scm.LocImm {
			r117 := ctx.AllocRegExcept(d739.Reg)
			ctx.EmitMovRegReg(r117, d739.Reg)
			ctx.EmitShlRegImm8(r117, uint8(d737.Imm.Int()))
			d741 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			ctx.BindReg(r117, &d741)
		} else {
			{
				shiftSrc := d739.Reg
				r118 := ctx.AllocRegExcept(d739.Reg, d737.Reg)
				ctx.EmitMovRegReg(r118, d739.Reg)
				shiftSrc = r118
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d737.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d737.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d737.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d741 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d741)
			}
		}
		if d741.Loc == scm.LocReg && d739.Loc == scm.LocReg && d741.Reg == d739.Reg {
			ctx.TransferReg(d739.Reg)
			d739.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d739)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d736)
		ctx.EnsureDesc(&d736)
		var d742 scm.JITValueDesc
		if d736.Loc == scm.LocImm {
			d742 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d736.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d736.Reg)
			ctx.EmitMovRegReg(scratch, d736.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d742 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d742)
		}
		if d742.Loc == scm.LocReg && d736.Loc == scm.LocReg && d742.Reg == d736.Reg {
			ctx.TransferReg(d736.Reg)
			d736.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d736)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d742)
		ctx.ReclaimUntrackedRegs()
		d744 = ctx.EmitSliceElementAddress(&d738, &d742, 8)
		ctx.EnsureDesc(&d744)
		ctx.EmitMovRegMem(d744.Reg, d744.Reg, 0)
		d743 = d744
		d743.Type = scm.TagInt
		ctx.FreeDesc(&d742)
		ctx.ReclaimUntrackedRegs()
		d745 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d737)
		ctx.EnsureDescsTogether(&d745, &d737)
		var d746 scm.JITValueDesc
		if d745.Loc == scm.LocImm && d737.Loc == scm.LocImm {
			d746 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d745.Imm.Int() - d737.Imm.Int())}
		} else if d737.Loc == scm.LocImm && d737.Imm.Int() == 0 {
			r119 := ctx.AllocRegExcept(d745.Reg)
			ctx.EmitMovRegReg(r119, d745.Reg)
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			ctx.BindReg(r119, &d746)
		} else if d745.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d737.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d745.Imm.Int()))
			ctx.EmitSubInt64(scratch, d737.Reg)
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d746)
		} else if d737.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d745.Reg)
			ctx.EmitMovRegReg(scratch, d745.Reg)
			if d737.Imm.Int() >= -2147483648 && d737.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d737.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d737.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d746)
		} else {
			r120 := ctx.AllocRegExcept(d745.Reg, d737.Reg)
			ctx.EmitMovRegReg(r120, d745.Reg)
			ctx.EmitSubInt64(r120, d737.Reg)
			d746 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			ctx.BindReg(r120, &d746)
		}
		if d746.Loc == scm.LocReg && d745.Loc == scm.LocReg && d746.Reg == d745.Reg {
			ctx.TransferReg(d745.Reg)
			d745.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d737)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d743)
		ctx.EnsureDesc(&d746)
		ctx.EnsureDescsTogether(&d743, &d746)
		var d747 scm.JITValueDesc
		if d743.Loc == scm.LocImm && d746.Loc == scm.LocImm {
			d747 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d743.Imm.Int()) >> uint64(d746.Imm.Int())))}
		} else if d746.Loc == scm.LocImm {
			r121 := ctx.AllocRegExcept(d743.Reg)
			ctx.EmitMovRegReg(r121, d743.Reg)
			ctx.EmitShrRegImm8(r121, uint8(d746.Imm.Int()))
			d747 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
			ctx.BindReg(r121, &d747)
		} else {
			{
				shiftSrc := d743.Reg
				r122 := ctx.AllocRegExcept(d743.Reg, d746.Reg)
				ctx.EmitMovRegReg(r122, d743.Reg)
				shiftSrc = r122
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d746.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d746.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d746.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d747 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d747)
			}
		}
		if d747.Loc == scm.LocReg && d743.Loc == scm.LocReg && d747.Reg == d743.Reg {
			ctx.TransferReg(d743.Reg)
			d743.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d743)
		ctx.FreeDesc(&d746)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d741)
		ctx.EnsureDesc(&d747)
		var d748 scm.JITValueDesc
		if d741.Loc == scm.LocImm && d747.Loc == scm.LocImm {
			d748 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d741.Imm.Int() | d747.Imm.Int())}
		} else if d741.Loc == scm.LocImm && d741.Imm.Int() == 0 {
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d747.Reg}
			ctx.BindReg(d747.Reg, &d748)
		} else if d747.Loc == scm.LocImm && d747.Imm.Int() == 0 {
			r123 := ctx.AllocRegExcept(d741.Reg)
			ctx.EmitMovRegReg(r123, d741.Reg)
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			ctx.BindReg(r123, &d748)
		} else if d741.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d747.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d741.Imm.Int()))
			ctx.EmitOrInt64(scratch, d747.Reg)
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d748)
		} else if d747.Loc == scm.LocImm {
			r124 := ctx.AllocRegExcept(d741.Reg)
			ctx.EmitMovRegReg(r124, d741.Reg)
			if d747.Imm.Int() >= -2147483648 && d747.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r124, int32(d747.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d747.Imm.Int()))
				ctx.EmitOrInt64(r124, scm.RegR11)
			}
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			ctx.BindReg(r124, &d748)
		} else {
			r125 := ctx.AllocRegExcept(d741.Reg, d747.Reg)
			ctx.EmitMovRegReg(r125, d741.Reg)
			ctx.EmitOrInt64(r125, d747.Reg)
			d748 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
			ctx.BindReg(r125, &d748)
		}
		if d748.Loc == scm.LocReg && d741.Loc == scm.LocReg && d748.Reg == d741.Reg {
			ctx.TransferReg(d741.Reg)
			d741.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d741)
		ctx.FreeDesc(&d747)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d749 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d749 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 48)
			r126 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r126, thisptr.Reg, off)
			d749 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.BindReg(r126, &d749)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d749)
		ctx.EnsureDesc(&d749)
		var d750 scm.JITValueDesc
		if d749.Loc == scm.LocImm {
			d750 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d749.Imm.Int()))))}
		} else {
			r127 := ctx.AllocReg()
			ctx.EmitMovRegReg(r127, d749.Reg)
			ctx.EmitShlRegImm8(r127, 56)
			ctx.EmitShrRegImm8(r127, 56)
			d750 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			ctx.BindReg(r127, &d750)
		}
		ctx.FreeDesc(&d749)
		ctx.ReclaimUntrackedRegs()
		d751 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d750)
		ctx.EnsureDescsTogether(&d751, &d750)
		var d752 scm.JITValueDesc
		if d751.Loc == scm.LocImm && d750.Loc == scm.LocImm {
			d752 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d751.Imm.Int() - d750.Imm.Int())}
		} else if d750.Loc == scm.LocImm && d750.Imm.Int() == 0 {
			r128 := ctx.AllocRegExcept(d751.Reg)
			ctx.EmitMovRegReg(r128, d751.Reg)
			d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			ctx.BindReg(r128, &d752)
		} else if d751.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d750.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d751.Imm.Int()))
			ctx.EmitSubInt64(scratch, d750.Reg)
			d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d752)
		} else if d750.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d751.Reg)
			ctx.EmitMovRegReg(scratch, d751.Reg)
			if d750.Imm.Int() >= -2147483648 && d750.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d750.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d750.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d752)
		} else {
			r129 := ctx.AllocRegExcept(d751.Reg, d750.Reg)
			ctx.EmitMovRegReg(r129, d751.Reg)
			ctx.EmitSubInt64(r129, d750.Reg)
			d752 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			ctx.BindReg(r129, &d752)
		}
		if d752.Loc == scm.LocReg && d751.Loc == scm.LocReg && d752.Reg == d751.Reg {
			ctx.TransferReg(d751.Reg)
			d751.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d750)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d748)
		ctx.EnsureDesc(&d752)
		ctx.EnsureDescsTogether(&d748, &d752)
		var d753 scm.JITValueDesc
		if d748.Loc == scm.LocImm && d752.Loc == scm.LocImm {
			d753 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d748.Imm.Int()) >> uint64(d752.Imm.Int())))}
		} else if d752.Loc == scm.LocImm {
			r130 := ctx.AllocRegExcept(d748.Reg)
			ctx.EmitMovRegReg(r130, d748.Reg)
			ctx.EmitShrRegImm8(r130, uint8(d752.Imm.Int()))
			d753 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			ctx.BindReg(r130, &d753)
		} else {
			{
				shiftSrc := d748.Reg
				r131 := ctx.AllocRegExcept(d748.Reg, d752.Reg)
				ctx.EmitMovRegReg(r131, d748.Reg)
				shiftSrc = r131
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d752.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d752.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d752.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d753 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d753)
			}
		}
		if d753.Loc == scm.LocReg && d748.Loc == scm.LocReg && d753.Reg == d748.Reg {
			ctx.TransferReg(d748.Reg)
			d748.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d748)
		ctx.FreeDesc(&d752)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d753)
		ctx.EnsureDesc(&d753)
		ctx.EnsureDesc(&d753)
		var d754 scm.JITValueDesc
		if d753.Loc == scm.LocImm {
			d754 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d753.Imm.Int()))))}
		} else {
			r132 := ctx.AllocReg()
			ctx.EmitMovRegReg(r132, d753.Reg)
			d754 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			ctx.BindReg(r132, &d754)
		}
		ctx.FreeDesc(&d753)
		var d755 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d755 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 56)
			r133 := ctx.AllocReg()
			ctx.EmitMovRegMem(r133, thisptr.Reg, off)
			d755 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r133}
			ctx.BindReg(r133, &d755)
		}
		ctx.EnsureDesc(&d754)
		ctx.EnsureDesc(&d755)
		ctx.EnsureDescsTogether(&d754, &d755)
		var d756 scm.JITValueDesc
		if d754.Loc == scm.LocImm && d755.Loc == scm.LocImm {
			d756 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d754.Imm.Int() + d755.Imm.Int())}
		} else if d755.Loc == scm.LocImm && d755.Imm.Int() == 0 {
			r134 := ctx.AllocRegExcept(d754.Reg)
			ctx.EmitMovRegReg(r134, d754.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			ctx.BindReg(r134, &d756)
		} else if d754.Loc == scm.LocImm && d754.Imm.Int() == 0 {
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d755.Reg}
			ctx.BindReg(d755.Reg, &d756)
		} else if d754.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d755.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d754.Imm.Int()))
			ctx.EmitAddInt64(scratch, d755.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d756)
		} else if d755.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d754.Reg)
			ctx.EmitMovRegReg(scratch, d754.Reg)
			if d755.Imm.Int() >= -2147483648 && d755.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d755.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d755.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d756)
		} else {
			r135 := ctx.AllocRegExcept(d754.Reg, d755.Reg)
			ctx.EmitMovRegReg(r135, d754.Reg)
			ctx.EmitAddInt64(r135, d755.Reg)
			d756 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			ctx.BindReg(r135, &d756)
		}
		if d756.Loc == scm.LocReg && d754.Loc == scm.LocReg && d756.Reg == d754.Reg {
			ctx.TransferReg(d754.Reg)
			d754.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d754)
		ctx.FreeDesc(&d755)
		ctx.EnsureDesc(&d4)
		d757 = d4
		_ = d757
		ctx.StabilizeDescForControlFlow(&d757)
		ctx.StabilizeDescForControlFlow(&d4)
		bbpos_5_0 := int32(-1)
		_ = bbpos_5_0
		lbl29 := ctx.ReserveLabel()
		_ = lbl29
		bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl29)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d757)
		ctx.EnsureDesc(&d757)
		var d758 scm.JITValueDesc
		if d757.Loc == scm.LocImm {
			d758 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d757.Imm.Int()))))}
		} else {
			r136 := ctx.AllocReg()
			ctx.EmitMovRegReg(r136, d757.Reg)
			ctx.EmitShlRegImm8(r136, 32)
			ctx.EmitShrRegImm8(r136, 32)
			d758 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			ctx.BindReg(r136, &d758)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d759 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d759 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r137 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r137, thisptr.Reg, off)
			d759 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
			ctx.BindReg(r137, &d759)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d759)
		ctx.EnsureDesc(&d759)
		var d760 scm.JITValueDesc
		if d759.Loc == scm.LocImm {
			d760 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d759.Imm.Int()))))}
		} else {
			r138 := ctx.AllocReg()
			ctx.EmitMovRegReg(r138, d759.Reg)
			ctx.EmitShlRegImm8(r138, 56)
			ctx.EmitShrRegImm8(r138, 56)
			d760 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			ctx.BindReg(r138, &d760)
		}
		ctx.FreeDesc(&d759)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d758)
		ctx.EnsureDesc(&d760)
		ctx.EnsureDescsTogether(&d758, &d760)
		var d761 scm.JITValueDesc
		if d758.Loc == scm.LocImm && d760.Loc == scm.LocImm {
			d761 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d758.Imm.Int() * d760.Imm.Int())}
		} else if d758.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d760.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d758.Imm.Int()))
			ctx.EmitImulInt64(scratch, d760.Reg)
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d761)
		} else if d760.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d758.Reg)
			ctx.EmitMovRegReg(scratch, d758.Reg)
			if d760.Imm.Int() >= -2147483648 && d760.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d760.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d760.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d761)
		} else {
			r139 := ctx.AllocRegExcept(d758.Reg, d760.Reg)
			ctx.EmitMovRegReg(r139, d758.Reg)
			ctx.EmitImulInt64(r139, d760.Reg)
			d761 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
			ctx.BindReg(r139, &d761)
		}
		if d761.Loc == scm.LocReg && d758.Loc == scm.LocReg && d761.Reg == d758.Reg {
			ctx.TransferReg(d758.Reg)
			d758.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d758)
		ctx.FreeDesc(&d760)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d761)
		var d762 scm.JITValueDesc
		if d761.Loc == scm.LocImm {
			d762 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d761.Imm.Int() / 64)}
		} else {
			r140 := ctx.AllocRegExcept(d761.Reg)
			ctx.EmitMovRegReg(r140, d761.Reg)
			ctx.EmitShrRegImm8(r140, 6)
			d762 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			ctx.BindReg(r140, &d762)
		}
		if d762.Loc == scm.LocReg && d761.Loc == scm.LocReg && d762.Reg == d761.Reg {
			ctx.TransferReg(d761.Reg)
			d761.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d761)
		var d763 scm.JITValueDesc
		if d761.Loc == scm.LocImm {
			d763 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d761.Imm.Int() % 64)}
		} else {
			r141 := ctx.AllocRegExcept(d761.Reg)
			ctx.EmitMovRegReg(r141, d761.Reg)
			ctx.EmitAndRegImm32(r141, 63)
			d763 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
			ctx.BindReg(r141, &d763)
		}
		if d763.Loc == scm.LocReg && d761.Loc == scm.LocReg && d763.Reg == d761.Reg {
			ctx.TransferReg(d761.Reg)
			d761.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d761)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d764 scm.JITValueDesc
		r142 := ctx.AllocReg()
		r143 := ctx.AllocRegExcept(r142)
		r144 := ctx.AllocRegExcept(r142, r143)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r142, uint64(dataPtr))
			ctx.EmitMovRegImm64(r143, uint64(sliceLen))
			ctx.EmitMovRegImm64(r144, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
			ctx.EmitMovRegMem(r142, thisptr.Reg, off)
			ctx.EmitMovRegMem(r143, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r144, thisptr.Reg, off+16)
		}
		d764 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r142, Reg2: r143, Reg3: r144}
		ctx.BindReg(r142, &d764)
		ctx.BindReg(r143, &d764)
		ctx.BindReg(r144, &d764)
		ctx.BindReg(r142, &d764)
		ctx.BindReg(r143, &d764)
		ctx.BindReg(r144, &d764)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d762)
		ctx.ReclaimUntrackedRegs()
		d766 = ctx.EmitSliceElementAddress(&d764, &d762, 8)
		ctx.EnsureDesc(&d766)
		ctx.EmitMovRegMem(d766.Reg, d766.Reg, 0)
		d765 = d766
		d765.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d765)
		ctx.EnsureDesc(&d763)
		ctx.EnsureDescsTogether(&d765, &d763)
		var d767 scm.JITValueDesc
		if d765.Loc == scm.LocImm && d763.Loc == scm.LocImm {
			d767 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d765.Imm.Int()) << uint64(d763.Imm.Int())))}
		} else if d763.Loc == scm.LocImm {
			r145 := ctx.AllocRegExcept(d765.Reg)
			ctx.EmitMovRegReg(r145, d765.Reg)
			ctx.EmitShlRegImm8(r145, uint8(d763.Imm.Int()))
			d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			ctx.BindReg(r145, &d767)
		} else {
			{
				shiftSrc := d765.Reg
				r146 := ctx.AllocRegExcept(d765.Reg, d763.Reg)
				ctx.EmitMovRegReg(r146, d765.Reg)
				shiftSrc = r146
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d763.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d763.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d763.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d767 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d767)
			}
		}
		if d767.Loc == scm.LocReg && d765.Loc == scm.LocReg && d767.Reg == d765.Reg {
			ctx.TransferReg(d765.Reg)
			d765.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d765)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d762)
		ctx.EnsureDesc(&d762)
		var d768 scm.JITValueDesc
		if d762.Loc == scm.LocImm {
			d768 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d762.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d762.Reg)
			ctx.EmitMovRegReg(scratch, d762.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d768 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d768)
		}
		if d768.Loc == scm.LocReg && d762.Loc == scm.LocReg && d768.Reg == d762.Reg {
			ctx.TransferReg(d762.Reg)
			d762.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d762)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d768)
		ctx.ReclaimUntrackedRegs()
		d770 = ctx.EmitSliceElementAddress(&d764, &d768, 8)
		ctx.EnsureDesc(&d770)
		ctx.EmitMovRegMem(d770.Reg, d770.Reg, 0)
		d769 = d770
		d769.Type = scm.TagInt
		ctx.FreeDesc(&d768)
		ctx.ReclaimUntrackedRegs()
		d771 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d763)
		ctx.EnsureDescsTogether(&d771, &d763)
		var d772 scm.JITValueDesc
		if d771.Loc == scm.LocImm && d763.Loc == scm.LocImm {
			d772 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d771.Imm.Int() - d763.Imm.Int())}
		} else if d763.Loc == scm.LocImm && d763.Imm.Int() == 0 {
			r147 := ctx.AllocRegExcept(d771.Reg)
			ctx.EmitMovRegReg(r147, d771.Reg)
			d772 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			ctx.BindReg(r147, &d772)
		} else if d771.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d763.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d771.Imm.Int()))
			ctx.EmitSubInt64(scratch, d763.Reg)
			d772 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d772)
		} else if d763.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d771.Reg)
			ctx.EmitMovRegReg(scratch, d771.Reg)
			if d763.Imm.Int() >= -2147483648 && d763.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d763.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d763.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d772 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d772)
		} else {
			r148 := ctx.AllocRegExcept(d771.Reg, d763.Reg)
			ctx.EmitMovRegReg(r148, d771.Reg)
			ctx.EmitSubInt64(r148, d763.Reg)
			d772 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			ctx.BindReg(r148, &d772)
		}
		if d772.Loc == scm.LocReg && d771.Loc == scm.LocReg && d772.Reg == d771.Reg {
			ctx.TransferReg(d771.Reg)
			d771.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d763)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d769)
		ctx.EnsureDesc(&d772)
		ctx.EnsureDescsTogether(&d769, &d772)
		var d773 scm.JITValueDesc
		if d769.Loc == scm.LocImm && d772.Loc == scm.LocImm {
			d773 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d769.Imm.Int()) >> uint64(d772.Imm.Int())))}
		} else if d772.Loc == scm.LocImm {
			r149 := ctx.AllocRegExcept(d769.Reg)
			ctx.EmitMovRegReg(r149, d769.Reg)
			ctx.EmitShrRegImm8(r149, uint8(d772.Imm.Int()))
			d773 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			ctx.BindReg(r149, &d773)
		} else {
			{
				shiftSrc := d769.Reg
				r150 := ctx.AllocRegExcept(d769.Reg, d772.Reg)
				ctx.EmitMovRegReg(r150, d769.Reg)
				shiftSrc = r150
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d772.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d772.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d772.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d773 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d773)
			}
		}
		if d773.Loc == scm.LocReg && d769.Loc == scm.LocReg && d773.Reg == d769.Reg {
			ctx.TransferReg(d769.Reg)
			d769.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d769)
		ctx.FreeDesc(&d772)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d767)
		ctx.EnsureDesc(&d773)
		var d774 scm.JITValueDesc
		if d767.Loc == scm.LocImm && d773.Loc == scm.LocImm {
			d774 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d767.Imm.Int() | d773.Imm.Int())}
		} else if d767.Loc == scm.LocImm && d767.Imm.Int() == 0 {
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d773.Reg}
			ctx.BindReg(d773.Reg, &d774)
		} else if d773.Loc == scm.LocImm && d773.Imm.Int() == 0 {
			r151 := ctx.AllocRegExcept(d767.Reg)
			ctx.EmitMovRegReg(r151, d767.Reg)
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			ctx.BindReg(r151, &d774)
		} else if d767.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d773.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d767.Imm.Int()))
			ctx.EmitOrInt64(scratch, d773.Reg)
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d774)
		} else if d773.Loc == scm.LocImm {
			r152 := ctx.AllocRegExcept(d767.Reg)
			ctx.EmitMovRegReg(r152, d767.Reg)
			if d773.Imm.Int() >= -2147483648 && d773.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r152, int32(d773.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d773.Imm.Int()))
				ctx.EmitOrInt64(r152, scm.RegR11)
			}
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			ctx.BindReg(r152, &d774)
		} else {
			r153 := ctx.AllocRegExcept(d767.Reg, d773.Reg)
			ctx.EmitMovRegReg(r153, d767.Reg)
			ctx.EmitOrInt64(r153, d773.Reg)
			d774 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			ctx.BindReg(r153, &d774)
		}
		if d774.Loc == scm.LocReg && d767.Loc == scm.LocReg && d774.Reg == d767.Reg {
			ctx.TransferReg(d767.Reg)
			d767.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d767)
		ctx.FreeDesc(&d773)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d775 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d775 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 48)
			r154 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r154, thisptr.Reg, off)
			d775 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d775)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d775)
		ctx.EnsureDesc(&d775)
		var d776 scm.JITValueDesc
		if d775.Loc == scm.LocImm {
			d776 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d775.Imm.Int()))))}
		} else {
			r155 := ctx.AllocReg()
			ctx.EmitMovRegReg(r155, d775.Reg)
			ctx.EmitShlRegImm8(r155, 56)
			ctx.EmitShrRegImm8(r155, 56)
			d776 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			ctx.BindReg(r155, &d776)
		}
		ctx.FreeDesc(&d775)
		ctx.ReclaimUntrackedRegs()
		d777 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d776)
		ctx.EnsureDescsTogether(&d777, &d776)
		var d778 scm.JITValueDesc
		if d777.Loc == scm.LocImm && d776.Loc == scm.LocImm {
			d778 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d777.Imm.Int() - d776.Imm.Int())}
		} else if d776.Loc == scm.LocImm && d776.Imm.Int() == 0 {
			r156 := ctx.AllocRegExcept(d777.Reg)
			ctx.EmitMovRegReg(r156, d777.Reg)
			d778 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			ctx.BindReg(r156, &d778)
		} else if d777.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d776.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d777.Imm.Int()))
			ctx.EmitSubInt64(scratch, d776.Reg)
			d778 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d778)
		} else if d776.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d777.Reg)
			ctx.EmitMovRegReg(scratch, d777.Reg)
			if d776.Imm.Int() >= -2147483648 && d776.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d776.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d776.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d778 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d778)
		} else {
			r157 := ctx.AllocRegExcept(d777.Reg, d776.Reg)
			ctx.EmitMovRegReg(r157, d777.Reg)
			ctx.EmitSubInt64(r157, d776.Reg)
			d778 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			ctx.BindReg(r157, &d778)
		}
		if d778.Loc == scm.LocReg && d777.Loc == scm.LocReg && d778.Reg == d777.Reg {
			ctx.TransferReg(d777.Reg)
			d777.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d776)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d774)
		ctx.EnsureDesc(&d778)
		ctx.EnsureDescsTogether(&d774, &d778)
		var d779 scm.JITValueDesc
		if d774.Loc == scm.LocImm && d778.Loc == scm.LocImm {
			d779 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d774.Imm.Int()) >> uint64(d778.Imm.Int())))}
		} else if d778.Loc == scm.LocImm {
			r158 := ctx.AllocRegExcept(d774.Reg)
			ctx.EmitMovRegReg(r158, d774.Reg)
			ctx.EmitShrRegImm8(r158, uint8(d778.Imm.Int()))
			d779 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			ctx.BindReg(r158, &d779)
		} else {
			{
				shiftSrc := d774.Reg
				r159 := ctx.AllocRegExcept(d774.Reg, d778.Reg)
				ctx.EmitMovRegReg(r159, d774.Reg)
				shiftSrc = r159
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d778.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d778.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d778.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d779 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d779)
			}
		}
		if d779.Loc == scm.LocReg && d774.Loc == scm.LocReg && d779.Reg == d774.Reg {
			ctx.TransferReg(d774.Reg)
			d774.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d774)
		ctx.FreeDesc(&d778)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d779)
		ctx.EnsureDesc(&d779)
		ctx.EnsureDesc(&d779)
		var d780 scm.JITValueDesc
		if d779.Loc == scm.LocImm {
			d780 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d779.Imm.Int()))))}
		} else {
			r160 := ctx.AllocReg()
			ctx.EmitMovRegReg(r160, d779.Reg)
			d780 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			ctx.BindReg(r160, &d780)
		}
		ctx.FreeDesc(&d779)
		var d781 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d781 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 56)
			r161 := ctx.AllocReg()
			ctx.EmitMovRegMem(r161, thisptr.Reg, off)
			d781 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r161}
			ctx.BindReg(r161, &d781)
		}
		ctx.EnsureDesc(&d780)
		ctx.EnsureDesc(&d781)
		ctx.EnsureDescsTogether(&d780, &d781)
		var d782 scm.JITValueDesc
		if d780.Loc == scm.LocImm && d781.Loc == scm.LocImm {
			d782 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d780.Imm.Int() + d781.Imm.Int())}
		} else if d781.Loc == scm.LocImm && d781.Imm.Int() == 0 {
			r162 := ctx.AllocRegExcept(d780.Reg)
			ctx.EmitMovRegReg(r162, d780.Reg)
			d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
			ctx.BindReg(r162, &d782)
		} else if d780.Loc == scm.LocImm && d780.Imm.Int() == 0 {
			d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d781.Reg}
			ctx.BindReg(d781.Reg, &d782)
		} else if d780.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d781.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d780.Imm.Int()))
			ctx.EmitAddInt64(scratch, d781.Reg)
			d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d782)
		} else if d781.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d780.Reg)
			ctx.EmitMovRegReg(scratch, d780.Reg)
			if d781.Imm.Int() >= -2147483648 && d781.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d781.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d781.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d782)
		} else {
			r163 := ctx.AllocRegExcept(d780.Reg, d781.Reg)
			ctx.EmitMovRegReg(r163, d780.Reg)
			ctx.EmitAddInt64(r163, d781.Reg)
			d782 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
			ctx.BindReg(r163, &d782)
		}
		if d782.Loc == scm.LocReg && d780.Loc == scm.LocReg && d782.Reg == d780.Reg {
			ctx.TransferReg(d780.Reg)
			d780.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d780)
		ctx.FreeDesc(&d781)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		var d783 scm.JITValueDesc
		if idxInt.Loc == scm.LocImm {
			d783 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
		} else {
			r164 := ctx.AllocReg()
			ctx.EmitMovRegReg(r164, idxInt.Reg)
			ctx.EmitShlRegImm8(r164, 32)
			ctx.EmitShrRegImm8(r164, 32)
			d783 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
			ctx.BindReg(r164, &d783)
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d783)
		ctx.EnsureDesc(&d782)
		ctx.EnsureDescsTogether(&d783, &d782)
		var d784 scm.JITValueDesc
		if d783.Loc == scm.LocImm && d782.Loc == scm.LocImm {
			d784 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d783.Imm.Int() - d782.Imm.Int())}
		} else if d782.Loc == scm.LocImm && d782.Imm.Int() == 0 {
			r165 := ctx.AllocRegExcept(d783.Reg)
			ctx.EmitMovRegReg(r165, d783.Reg)
			d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
			ctx.BindReg(r165, &d784)
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
			r166 := ctx.AllocRegExcept(d783.Reg, d782.Reg)
			ctx.EmitMovRegReg(r166, d783.Reg)
			ctx.EmitSubInt64(r166, d782.Reg)
			d784 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
			ctx.BindReg(r166, &d784)
		}
		if d784.Loc == scm.LocReg && d783.Loc == scm.LocReg && d784.Reg == d783.Reg {
			ctx.TransferReg(d783.Reg)
			d783.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d783)
		ctx.FreeDesc(&d782)
		ctx.EnsureDesc(&d784)
		ctx.EnsureDesc(&d756)
		ctx.EnsureDescsTogether(&d784, &d756)
		var d785 scm.JITValueDesc
		if d784.Loc == scm.LocImm && d756.Loc == scm.LocImm {
			d785 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d784.Imm.Int() * d756.Imm.Int())}
		} else if d784.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d756.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d784.Imm.Int()))
			ctx.EmitImulInt64(scratch, d756.Reg)
			d785 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d785)
		} else if d756.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d784.Reg)
			ctx.EmitMovRegReg(scratch, d784.Reg)
			if d756.Imm.Int() >= -2147483648 && d756.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d756.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d756.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d785 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d785)
		} else {
			r167 := ctx.AllocRegExcept(d784.Reg, d756.Reg)
			ctx.EmitMovRegReg(r167, d784.Reg)
			ctx.EmitImulInt64(r167, d756.Reg)
			d785 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
			ctx.BindReg(r167, &d785)
		}
		if d785.Loc == scm.LocReg && d784.Loc == scm.LocReg && d785.Reg == d784.Reg {
			ctx.TransferReg(d784.Reg)
			d784.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d784)
		ctx.FreeDesc(&d756)
		ctx.EnsureDesc(&d138)
		ctx.EnsureDesc(&d785)
		ctx.EnsureDescsTogether(&d138, &d785)
		var d786 scm.JITValueDesc
		if d138.Loc == scm.LocImm && d785.Loc == scm.LocImm {
			d786 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d138.Imm.Int() + d785.Imm.Int())}
		} else if d785.Loc == scm.LocImm && d785.Imm.Int() == 0 {
			r168 := ctx.AllocRegExcept(d138.Reg)
			ctx.EmitMovRegReg(r168, d138.Reg)
			d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
			ctx.BindReg(r168, &d786)
		} else if d138.Loc == scm.LocImm && d138.Imm.Int() == 0 {
			d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d785.Reg}
			ctx.BindReg(d785.Reg, &d786)
		} else if d138.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d785.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d138.Imm.Int()))
			ctx.EmitAddInt64(scratch, d785.Reg)
			d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d786)
		} else if d785.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d138.Reg)
			ctx.EmitMovRegReg(scratch, d138.Reg)
			if d785.Imm.Int() >= -2147483648 && d785.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d785.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d785.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d786)
		} else {
			r169 := ctx.AllocRegExcept(d138.Reg, d785.Reg)
			ctx.EmitMovRegReg(r169, d138.Reg)
			ctx.EmitAddInt64(r169, d785.Reg)
			d786 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
			ctx.BindReg(r169, &d786)
		}
		if d786.Loc == scm.LocReg && d138.Loc == scm.LocReg && d786.Reg == d138.Reg {
			ctx.TransferReg(d138.Reg)
			d138.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d785)
		ctx.EnsureDesc(&d786)
		ctx.EnsureDesc(&d786)
		var d787 scm.JITValueDesc
		if d786.Loc == scm.LocImm {
			d787 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d786.Imm.Int()))}
		} else {
			r170 := ctx.AllocRegExcept(d786.Reg)
			ctx.EmitMovRegReg(r170, d786.Reg)
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r170)
			d787 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r170}
			ctx.BindReg(r170, &d787)
		}
		ctx.FreeDesc(&d786)
		ctx.EnsureDesc(&d787)
		d788 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d788)
		ctx.BindReg(r1, &d788)
		ctx.EnsureDesc(&d787)
		ctx.EmitMakeFloat(d788, d787)
		if d787.Loc == scm.LocReg {
			ctx.FreeReg(d787.Reg)
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
		d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(0)}
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(16)}
		d3 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(32)}
		d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(48)}
		d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(64)}
		d6 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(80)}
		d7 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(96)}
		d8 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(112)}
		d9 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase0) + int32(128)}
		if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
			d1 = ps.OverlayValues[1]
		}
		if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
			d2 = ps.OverlayValues[2]
		}
		if !ps.General && len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
			d3 = ps.OverlayValues[3]
		}
		if !ps.General && len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
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
		if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
			d14 = ps.OverlayValues[14]
		}
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
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
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != scm.LocNone {
			d111 = ps.OverlayValues[111]
		}
		if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != scm.LocNone {
			d112 = ps.OverlayValues[112]
		}
		if len(ps.OverlayValues) > 113 && ps.OverlayValues[113].Loc != scm.LocNone {
			d113 = ps.OverlayValues[113]
		}
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
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
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 230 && ps.OverlayValues[230].Loc != scm.LocNone {
			d230 = ps.OverlayValues[230]
		}
		if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
			d231 = ps.OverlayValues[231]
		}
		if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
			d232 = ps.OverlayValues[232]
		}
		if len(ps.OverlayValues) > 233 && ps.OverlayValues[233].Loc != scm.LocNone {
			d233 = ps.OverlayValues[233]
		}
		if len(ps.OverlayValues) > 235 && ps.OverlayValues[235].Loc != scm.LocNone {
			d235 = ps.OverlayValues[235]
		}
		if len(ps.OverlayValues) > 236 && ps.OverlayValues[236].Loc != scm.LocNone {
			d236 = ps.OverlayValues[236]
		}
		if len(ps.OverlayValues) > 237 && ps.OverlayValues[237].Loc != scm.LocNone {
			d237 = ps.OverlayValues[237]
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
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
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
		if len(ps.OverlayValues) > 253 && ps.OverlayValues[253].Loc != scm.LocNone {
			d253 = ps.OverlayValues[253]
		}
		if len(ps.OverlayValues) > 357 && ps.OverlayValues[357].Loc != scm.LocNone {
			d357 = ps.OverlayValues[357]
		}
		if len(ps.OverlayValues) > 358 && ps.OverlayValues[358].Loc != scm.LocNone {
			d358 = ps.OverlayValues[358]
		}
		if len(ps.OverlayValues) > 359 && ps.OverlayValues[359].Loc != scm.LocNone {
			d359 = ps.OverlayValues[359]
		}
		if len(ps.OverlayValues) > 360 && ps.OverlayValues[360].Loc != scm.LocNone {
			d360 = ps.OverlayValues[360]
		}
		if len(ps.OverlayValues) > 361 && ps.OverlayValues[361].Loc != scm.LocNone {
			d361 = ps.OverlayValues[361]
		}
		if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != scm.LocNone {
			d363 = ps.OverlayValues[363]
		}
		if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != scm.LocNone {
			d364 = ps.OverlayValues[364]
		}
		if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != scm.LocNone {
			d365 = ps.OverlayValues[365]
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
		if len(ps.OverlayValues) > 371 && ps.OverlayValues[371].Loc != scm.LocNone {
			d371 = ps.OverlayValues[371]
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
		if len(ps.OverlayValues) > 537 && ps.OverlayValues[537].Loc != scm.LocNone {
			d537 = ps.OverlayValues[537]
		}
		if len(ps.OverlayValues) > 538 && ps.OverlayValues[538].Loc != scm.LocNone {
			d538 = ps.OverlayValues[538]
		}
		if len(ps.OverlayValues) > 539 && ps.OverlayValues[539].Loc != scm.LocNone {
			d539 = ps.OverlayValues[539]
		}
		if len(ps.OverlayValues) > 541 && ps.OverlayValues[541].Loc != scm.LocNone {
			d541 = ps.OverlayValues[541]
		}
		if len(ps.OverlayValues) > 542 && ps.OverlayValues[542].Loc != scm.LocNone {
			d542 = ps.OverlayValues[542]
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
		if len(ps.OverlayValues) > 546 && ps.OverlayValues[546].Loc != scm.LocNone {
			d546 = ps.OverlayValues[546]
		}
		if len(ps.OverlayValues) > 547 && ps.OverlayValues[547].Loc != scm.LocNone {
			d547 = ps.OverlayValues[547]
		}
		if len(ps.OverlayValues) > 549 && ps.OverlayValues[549].Loc != scm.LocNone {
			d549 = ps.OverlayValues[549]
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
		if len(ps.OverlayValues) > 554 && ps.OverlayValues[554].Loc != scm.LocNone {
			d554 = ps.OverlayValues[554]
		}
		if len(ps.OverlayValues) > 557 && ps.OverlayValues[557].Loc != scm.LocNone {
			d557 = ps.OverlayValues[557]
		}
		if len(ps.OverlayValues) > 713 && ps.OverlayValues[713].Loc != scm.LocNone {
			d713 = ps.OverlayValues[713]
		}
		if len(ps.OverlayValues) > 714 && ps.OverlayValues[714].Loc != scm.LocNone {
			d714 = ps.OverlayValues[714]
		}
		if len(ps.OverlayValues) > 715 && ps.OverlayValues[715].Loc != scm.LocNone {
			d715 = ps.OverlayValues[715]
		}
		if len(ps.OverlayValues) > 716 && ps.OverlayValues[716].Loc != scm.LocNone {
			d716 = ps.OverlayValues[716]
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
		if len(ps.OverlayValues) > 722 && ps.OverlayValues[722].Loc != scm.LocNone {
			d722 = ps.OverlayValues[722]
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
		if len(ps.OverlayValues) > 733 && ps.OverlayValues[733].Loc != scm.LocNone {
			d733 = ps.OverlayValues[733]
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
		ctx.ReclaimUntrackedRegs()
		var d789 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d789 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 88)
			r171 := ctx.AllocReg()
			ctx.EmitMovRegMem(r171, thisptr.Reg, off)
			d789 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r171}
			ctx.BindReg(r171, &d789)
		}
		ctx.EnsureDesc(&d789)
		ctx.EnsureDesc(&d789)
		var d790 scm.JITValueDesc
		if d789.Loc == scm.LocImm {
			d790 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d789.Imm.Int()))))}
		} else {
			r172 := ctx.AllocReg()
			ctx.EmitMovRegReg(r172, d789.Reg)
			d790 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
			ctx.BindReg(r172, &d790)
		}
		ctx.FreeDesc(&d789)
		ctx.EnsureDesc(&d138)
		ctx.EnsureDesc(&d790)
		ctx.EnsureDescsTogether(&d138, &d790)
		var d791 scm.JITValueDesc
		if d138.Loc == scm.LocImm && d790.Loc == scm.LocImm {
			d791 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d138.Imm.Int() == d790.Imm.Int())}
		} else if d790.Loc == scm.LocImm {
			r173 := ctx.AllocRegExcept(d138.Reg)
			if d790.Imm.Int() >= -2147483648 && d790.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d138.Reg, int32(d790.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d790.Imm.Int()))
				ctx.EmitCmpInt64(d138.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r173, scm.CondEqual)
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r173}
			ctx.BindReg(r173, &d791)
		} else if d138.Loc == scm.LocImm {
			r174 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d790.Reg)
			ctx.EmitSetcc(r174, scm.CondEqual)
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r174}
			ctx.BindReg(r174, &d791)
		} else {
			r175 := ctx.AllocRegExcept(d138.Reg)
			ctx.EmitCmpInt64(d138.Reg, d790.Reg)
			ctx.EmitSetcc(r175, scm.CondEqual)
			d791 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r175}
			ctx.BindReg(r175, &d791)
		}
		ctx.FreeDesc(&d790)
		d792 = d791
		ctx.EnsureDesc(&d792)
		if d792.Loc != scm.LocImm && d792.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d792.Loc == scm.LocImm {
			if d792.Imm.Bool() {
				if ps.General {
				}
				ps793 := scm.PhiState{General: ps.General}
				ps793.OverlayValues = make([]scm.JITValueDesc, 793)
				ps793.OverlayValues[1] = d1
				ps793.OverlayValues[2] = d2
				ps793.OverlayValues[3] = d3
				ps793.OverlayValues[4] = d4
				ps793.OverlayValues[5] = d5
				ps793.OverlayValues[6] = d6
				ps793.OverlayValues[7] = d7
				ps793.OverlayValues[8] = d8
				ps793.OverlayValues[9] = d9
				ps793.OverlayValues[10] = d10
				ps793.OverlayValues[11] = d11
				ps793.OverlayValues[12] = d12
				ps793.OverlayValues[13] = d13
				ps793.OverlayValues[14] = d14
				ps793.OverlayValues[15] = d15
				ps793.OverlayValues[17] = d17
				ps793.OverlayValues[18] = d18
				ps793.OverlayValues[19] = d19
				ps793.OverlayValues[20] = d20
				ps793.OverlayValues[21] = d21
				ps793.OverlayValues[22] = d22
				ps793.OverlayValues[23] = d23
				ps793.OverlayValues[24] = d24
				ps793.OverlayValues[25] = d25
				ps793.OverlayValues[26] = d26
				ps793.OverlayValues[27] = d27
				ps793.OverlayValues[28] = d28
				ps793.OverlayValues[29] = d29
				ps793.OverlayValues[30] = d30
				ps793.OverlayValues[31] = d31
				ps793.OverlayValues[32] = d32
				ps793.OverlayValues[33] = d33
				ps793.OverlayValues[34] = d34
				ps793.OverlayValues[35] = d35
				ps793.OverlayValues[36] = d36
				ps793.OverlayValues[37] = d37
				ps793.OverlayValues[38] = d38
				ps793.OverlayValues[39] = d39
				ps793.OverlayValues[40] = d40
				ps793.OverlayValues[41] = d41
				ps793.OverlayValues[42] = d42
				ps793.OverlayValues[43] = d43
				ps793.OverlayValues[44] = d44
				ps793.OverlayValues[45] = d45
				ps793.OverlayValues[46] = d46
				ps793.OverlayValues[47] = d47
				ps793.OverlayValues[48] = d48
				ps793.OverlayValues[49] = d49
				ps793.OverlayValues[50] = d50
				ps793.OverlayValues[53] = d53
				ps793.OverlayValues[54] = d54
				ps793.OverlayValues[55] = d55
				ps793.OverlayValues[111] = d111
				ps793.OverlayValues[112] = d112
				ps793.OverlayValues[113] = d113
				ps793.OverlayValues[114] = d114
				ps793.OverlayValues[115] = d115
				ps793.OverlayValues[116] = d116
				ps793.OverlayValues[117] = d117
				ps793.OverlayValues[118] = d118
				ps793.OverlayValues[119] = d119
				ps793.OverlayValues[120] = d120
				ps793.OverlayValues[121] = d121
				ps793.OverlayValues[122] = d122
				ps793.OverlayValues[123] = d123
				ps793.OverlayValues[124] = d124
				ps793.OverlayValues[125] = d125
				ps793.OverlayValues[126] = d126
				ps793.OverlayValues[127] = d127
				ps793.OverlayValues[128] = d128
				ps793.OverlayValues[129] = d129
				ps793.OverlayValues[130] = d130
				ps793.OverlayValues[131] = d131
				ps793.OverlayValues[132] = d132
				ps793.OverlayValues[133] = d133
				ps793.OverlayValues[134] = d134
				ps793.OverlayValues[135] = d135
				ps793.OverlayValues[136] = d136
				ps793.OverlayValues[137] = d137
				ps793.OverlayValues[138] = d138
				ps793.OverlayValues[139] = d139
				ps793.OverlayValues[140] = d140
				ps793.OverlayValues[143] = d143
				ps793.OverlayValues[230] = d230
				ps793.OverlayValues[231] = d231
				ps793.OverlayValues[232] = d232
				ps793.OverlayValues[233] = d233
				ps793.OverlayValues[235] = d235
				ps793.OverlayValues[236] = d236
				ps793.OverlayValues[237] = d237
				ps793.OverlayValues[238] = d238
				ps793.OverlayValues[239] = d239
				ps793.OverlayValues[240] = d240
				ps793.OverlayValues[241] = d241
				ps793.OverlayValues[242] = d242
				ps793.OverlayValues[244] = d244
				ps793.OverlayValues[246] = d246
				ps793.OverlayValues[247] = d247
				ps793.OverlayValues[248] = d248
				ps793.OverlayValues[249] = d249
				ps793.OverlayValues[250] = d250
				ps793.OverlayValues[253] = d253
				ps793.OverlayValues[357] = d357
				ps793.OverlayValues[358] = d358
				ps793.OverlayValues[359] = d359
				ps793.OverlayValues[360] = d360
				ps793.OverlayValues[361] = d361
				ps793.OverlayValues[363] = d363
				ps793.OverlayValues[364] = d364
				ps793.OverlayValues[365] = d365
				ps793.OverlayValues[366] = d366
				ps793.OverlayValues[367] = d367
				ps793.OverlayValues[368] = d368
				ps793.OverlayValues[369] = d369
				ps793.OverlayValues[370] = d370
				ps793.OverlayValues[371] = d371
				ps793.OverlayValues[372] = d372
				ps793.OverlayValues[373] = d373
				ps793.OverlayValues[374] = d374
				ps793.OverlayValues[375] = d375
				ps793.OverlayValues[376] = d376
				ps793.OverlayValues[377] = d377
				ps793.OverlayValues[378] = d378
				ps793.OverlayValues[379] = d379
				ps793.OverlayValues[380] = d380
				ps793.OverlayValues[381] = d381
				ps793.OverlayValues[382] = d382
				ps793.OverlayValues[383] = d383
				ps793.OverlayValues[384] = d384
				ps793.OverlayValues[385] = d385
				ps793.OverlayValues[386] = d386
				ps793.OverlayValues[387] = d387
				ps793.OverlayValues[388] = d388
				ps793.OverlayValues[389] = d389
				ps793.OverlayValues[390] = d390
				ps793.OverlayValues[391] = d391
				ps793.OverlayValues[392] = d392
				ps793.OverlayValues[393] = d393
				ps793.OverlayValues[537] = d537
				ps793.OverlayValues[538] = d538
				ps793.OverlayValues[539] = d539
				ps793.OverlayValues[541] = d541
				ps793.OverlayValues[542] = d542
				ps793.OverlayValues[543] = d543
				ps793.OverlayValues[544] = d544
				ps793.OverlayValues[545] = d545
				ps793.OverlayValues[546] = d546
				ps793.OverlayValues[547] = d547
				ps793.OverlayValues[549] = d549
				ps793.OverlayValues[551] = d551
				ps793.OverlayValues[552] = d552
				ps793.OverlayValues[553] = d553
				ps793.OverlayValues[554] = d554
				ps793.OverlayValues[557] = d557
				ps793.OverlayValues[713] = d713
				ps793.OverlayValues[714] = d714
				ps793.OverlayValues[715] = d715
				ps793.OverlayValues[716] = d716
				ps793.OverlayValues[718] = d718
				ps793.OverlayValues[719] = d719
				ps793.OverlayValues[720] = d720
				ps793.OverlayValues[721] = d721
				ps793.OverlayValues[722] = d722
				ps793.OverlayValues[723] = d723
				ps793.OverlayValues[724] = d724
				ps793.OverlayValues[725] = d725
				ps793.OverlayValues[727] = d727
				ps793.OverlayValues[728] = d728
				ps793.OverlayValues[729] = d729
				ps793.OverlayValues[730] = d730
				ps793.OverlayValues[731] = d731
				ps793.OverlayValues[732] = d732
				ps793.OverlayValues[733] = d733
				ps793.OverlayValues[734] = d734
				ps793.OverlayValues[735] = d735
				ps793.OverlayValues[736] = d736
				ps793.OverlayValues[737] = d737
				ps793.OverlayValues[738] = d738
				ps793.OverlayValues[739] = d739
				ps793.OverlayValues[740] = d740
				ps793.OverlayValues[741] = d741
				ps793.OverlayValues[742] = d742
				ps793.OverlayValues[743] = d743
				ps793.OverlayValues[744] = d744
				ps793.OverlayValues[745] = d745
				ps793.OverlayValues[746] = d746
				ps793.OverlayValues[747] = d747
				ps793.OverlayValues[748] = d748
				ps793.OverlayValues[749] = d749
				ps793.OverlayValues[750] = d750
				ps793.OverlayValues[751] = d751
				ps793.OverlayValues[752] = d752
				ps793.OverlayValues[753] = d753
				ps793.OverlayValues[754] = d754
				ps793.OverlayValues[755] = d755
				ps793.OverlayValues[756] = d756
				ps793.OverlayValues[757] = d757
				ps793.OverlayValues[758] = d758
				ps793.OverlayValues[759] = d759
				ps793.OverlayValues[760] = d760
				ps793.OverlayValues[761] = d761
				ps793.OverlayValues[762] = d762
				ps793.OverlayValues[763] = d763
				ps793.OverlayValues[764] = d764
				ps793.OverlayValues[765] = d765
				ps793.OverlayValues[766] = d766
				ps793.OverlayValues[767] = d767
				ps793.OverlayValues[768] = d768
				ps793.OverlayValues[769] = d769
				ps793.OverlayValues[770] = d770
				ps793.OverlayValues[771] = d771
				ps793.OverlayValues[772] = d772
				ps793.OverlayValues[773] = d773
				ps793.OverlayValues[774] = d774
				ps793.OverlayValues[775] = d775
				ps793.OverlayValues[776] = d776
				ps793.OverlayValues[777] = d777
				ps793.OverlayValues[778] = d778
				ps793.OverlayValues[779] = d779
				ps793.OverlayValues[780] = d780
				ps793.OverlayValues[781] = d781
				ps793.OverlayValues[782] = d782
				ps793.OverlayValues[783] = d783
				ps793.OverlayValues[784] = d784
				ps793.OverlayValues[785] = d785
				ps793.OverlayValues[786] = d786
				ps793.OverlayValues[787] = d787
				ps793.OverlayValues[788] = d788
				ps793.OverlayValues[789] = d789
				ps793.OverlayValues[790] = d790
				ps793.OverlayValues[791] = d791
				ps793.OverlayValues[792] = d792
				return bbs[11].RenderPS(ps793)
			}
			if ps.General {
			}
			ps794 := scm.PhiState{General: ps.General}
			ps794.OverlayValues = make([]scm.JITValueDesc, 793)
			ps794.OverlayValues[1] = d1
			ps794.OverlayValues[2] = d2
			ps794.OverlayValues[3] = d3
			ps794.OverlayValues[4] = d4
			ps794.OverlayValues[5] = d5
			ps794.OverlayValues[6] = d6
			ps794.OverlayValues[7] = d7
			ps794.OverlayValues[8] = d8
			ps794.OverlayValues[9] = d9
			ps794.OverlayValues[10] = d10
			ps794.OverlayValues[11] = d11
			ps794.OverlayValues[12] = d12
			ps794.OverlayValues[13] = d13
			ps794.OverlayValues[14] = d14
			ps794.OverlayValues[15] = d15
			ps794.OverlayValues[17] = d17
			ps794.OverlayValues[18] = d18
			ps794.OverlayValues[19] = d19
			ps794.OverlayValues[20] = d20
			ps794.OverlayValues[21] = d21
			ps794.OverlayValues[22] = d22
			ps794.OverlayValues[23] = d23
			ps794.OverlayValues[24] = d24
			ps794.OverlayValues[25] = d25
			ps794.OverlayValues[26] = d26
			ps794.OverlayValues[27] = d27
			ps794.OverlayValues[28] = d28
			ps794.OverlayValues[29] = d29
			ps794.OverlayValues[30] = d30
			ps794.OverlayValues[31] = d31
			ps794.OverlayValues[32] = d32
			ps794.OverlayValues[33] = d33
			ps794.OverlayValues[34] = d34
			ps794.OverlayValues[35] = d35
			ps794.OverlayValues[36] = d36
			ps794.OverlayValues[37] = d37
			ps794.OverlayValues[38] = d38
			ps794.OverlayValues[39] = d39
			ps794.OverlayValues[40] = d40
			ps794.OverlayValues[41] = d41
			ps794.OverlayValues[42] = d42
			ps794.OverlayValues[43] = d43
			ps794.OverlayValues[44] = d44
			ps794.OverlayValues[45] = d45
			ps794.OverlayValues[46] = d46
			ps794.OverlayValues[47] = d47
			ps794.OverlayValues[48] = d48
			ps794.OverlayValues[49] = d49
			ps794.OverlayValues[50] = d50
			ps794.OverlayValues[53] = d53
			ps794.OverlayValues[54] = d54
			ps794.OverlayValues[55] = d55
			ps794.OverlayValues[111] = d111
			ps794.OverlayValues[112] = d112
			ps794.OverlayValues[113] = d113
			ps794.OverlayValues[114] = d114
			ps794.OverlayValues[115] = d115
			ps794.OverlayValues[116] = d116
			ps794.OverlayValues[117] = d117
			ps794.OverlayValues[118] = d118
			ps794.OverlayValues[119] = d119
			ps794.OverlayValues[120] = d120
			ps794.OverlayValues[121] = d121
			ps794.OverlayValues[122] = d122
			ps794.OverlayValues[123] = d123
			ps794.OverlayValues[124] = d124
			ps794.OverlayValues[125] = d125
			ps794.OverlayValues[126] = d126
			ps794.OverlayValues[127] = d127
			ps794.OverlayValues[128] = d128
			ps794.OverlayValues[129] = d129
			ps794.OverlayValues[130] = d130
			ps794.OverlayValues[131] = d131
			ps794.OverlayValues[132] = d132
			ps794.OverlayValues[133] = d133
			ps794.OverlayValues[134] = d134
			ps794.OverlayValues[135] = d135
			ps794.OverlayValues[136] = d136
			ps794.OverlayValues[137] = d137
			ps794.OverlayValues[138] = d138
			ps794.OverlayValues[139] = d139
			ps794.OverlayValues[140] = d140
			ps794.OverlayValues[143] = d143
			ps794.OverlayValues[230] = d230
			ps794.OverlayValues[231] = d231
			ps794.OverlayValues[232] = d232
			ps794.OverlayValues[233] = d233
			ps794.OverlayValues[235] = d235
			ps794.OverlayValues[236] = d236
			ps794.OverlayValues[237] = d237
			ps794.OverlayValues[238] = d238
			ps794.OverlayValues[239] = d239
			ps794.OverlayValues[240] = d240
			ps794.OverlayValues[241] = d241
			ps794.OverlayValues[242] = d242
			ps794.OverlayValues[244] = d244
			ps794.OverlayValues[246] = d246
			ps794.OverlayValues[247] = d247
			ps794.OverlayValues[248] = d248
			ps794.OverlayValues[249] = d249
			ps794.OverlayValues[250] = d250
			ps794.OverlayValues[253] = d253
			ps794.OverlayValues[357] = d357
			ps794.OverlayValues[358] = d358
			ps794.OverlayValues[359] = d359
			ps794.OverlayValues[360] = d360
			ps794.OverlayValues[361] = d361
			ps794.OverlayValues[363] = d363
			ps794.OverlayValues[364] = d364
			ps794.OverlayValues[365] = d365
			ps794.OverlayValues[366] = d366
			ps794.OverlayValues[367] = d367
			ps794.OverlayValues[368] = d368
			ps794.OverlayValues[369] = d369
			ps794.OverlayValues[370] = d370
			ps794.OverlayValues[371] = d371
			ps794.OverlayValues[372] = d372
			ps794.OverlayValues[373] = d373
			ps794.OverlayValues[374] = d374
			ps794.OverlayValues[375] = d375
			ps794.OverlayValues[376] = d376
			ps794.OverlayValues[377] = d377
			ps794.OverlayValues[378] = d378
			ps794.OverlayValues[379] = d379
			ps794.OverlayValues[380] = d380
			ps794.OverlayValues[381] = d381
			ps794.OverlayValues[382] = d382
			ps794.OverlayValues[383] = d383
			ps794.OverlayValues[384] = d384
			ps794.OverlayValues[385] = d385
			ps794.OverlayValues[386] = d386
			ps794.OverlayValues[387] = d387
			ps794.OverlayValues[388] = d388
			ps794.OverlayValues[389] = d389
			ps794.OverlayValues[390] = d390
			ps794.OverlayValues[391] = d391
			ps794.OverlayValues[392] = d392
			ps794.OverlayValues[393] = d393
			ps794.OverlayValues[537] = d537
			ps794.OverlayValues[538] = d538
			ps794.OverlayValues[539] = d539
			ps794.OverlayValues[541] = d541
			ps794.OverlayValues[542] = d542
			ps794.OverlayValues[543] = d543
			ps794.OverlayValues[544] = d544
			ps794.OverlayValues[545] = d545
			ps794.OverlayValues[546] = d546
			ps794.OverlayValues[547] = d547
			ps794.OverlayValues[549] = d549
			ps794.OverlayValues[551] = d551
			ps794.OverlayValues[552] = d552
			ps794.OverlayValues[553] = d553
			ps794.OverlayValues[554] = d554
			ps794.OverlayValues[557] = d557
			ps794.OverlayValues[713] = d713
			ps794.OverlayValues[714] = d714
			ps794.OverlayValues[715] = d715
			ps794.OverlayValues[716] = d716
			ps794.OverlayValues[718] = d718
			ps794.OverlayValues[719] = d719
			ps794.OverlayValues[720] = d720
			ps794.OverlayValues[721] = d721
			ps794.OverlayValues[722] = d722
			ps794.OverlayValues[723] = d723
			ps794.OverlayValues[724] = d724
			ps794.OverlayValues[725] = d725
			ps794.OverlayValues[727] = d727
			ps794.OverlayValues[728] = d728
			ps794.OverlayValues[729] = d729
			ps794.OverlayValues[730] = d730
			ps794.OverlayValues[731] = d731
			ps794.OverlayValues[732] = d732
			ps794.OverlayValues[733] = d733
			ps794.OverlayValues[734] = d734
			ps794.OverlayValues[735] = d735
			ps794.OverlayValues[736] = d736
			ps794.OverlayValues[737] = d737
			ps794.OverlayValues[738] = d738
			ps794.OverlayValues[739] = d739
			ps794.OverlayValues[740] = d740
			ps794.OverlayValues[741] = d741
			ps794.OverlayValues[742] = d742
			ps794.OverlayValues[743] = d743
			ps794.OverlayValues[744] = d744
			ps794.OverlayValues[745] = d745
			ps794.OverlayValues[746] = d746
			ps794.OverlayValues[747] = d747
			ps794.OverlayValues[748] = d748
			ps794.OverlayValues[749] = d749
			ps794.OverlayValues[750] = d750
			ps794.OverlayValues[751] = d751
			ps794.OverlayValues[752] = d752
			ps794.OverlayValues[753] = d753
			ps794.OverlayValues[754] = d754
			ps794.OverlayValues[755] = d755
			ps794.OverlayValues[756] = d756
			ps794.OverlayValues[757] = d757
			ps794.OverlayValues[758] = d758
			ps794.OverlayValues[759] = d759
			ps794.OverlayValues[760] = d760
			ps794.OverlayValues[761] = d761
			ps794.OverlayValues[762] = d762
			ps794.OverlayValues[763] = d763
			ps794.OverlayValues[764] = d764
			ps794.OverlayValues[765] = d765
			ps794.OverlayValues[766] = d766
			ps794.OverlayValues[767] = d767
			ps794.OverlayValues[768] = d768
			ps794.OverlayValues[769] = d769
			ps794.OverlayValues[770] = d770
			ps794.OverlayValues[771] = d771
			ps794.OverlayValues[772] = d772
			ps794.OverlayValues[773] = d773
			ps794.OverlayValues[774] = d774
			ps794.OverlayValues[775] = d775
			ps794.OverlayValues[776] = d776
			ps794.OverlayValues[777] = d777
			ps794.OverlayValues[778] = d778
			ps794.OverlayValues[779] = d779
			ps794.OverlayValues[780] = d780
			ps794.OverlayValues[781] = d781
			ps794.OverlayValues[782] = d782
			ps794.OverlayValues[783] = d783
			ps794.OverlayValues[784] = d784
			ps794.OverlayValues[785] = d785
			ps794.OverlayValues[786] = d786
			ps794.OverlayValues[787] = d787
			ps794.OverlayValues[788] = d788
			ps794.OverlayValues[789] = d789
			ps794.OverlayValues[790] = d790
			ps794.OverlayValues[791] = d791
			ps794.OverlayValues[792] = d792
			return bbs[12].RenderPS(ps794)
		}
		if !ps.General {
			ps.General = true
			return bbs[13].RenderPS(ps)
		}
		lbl30 := ctx.ReserveLabel()
		lbl31 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d792.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl30)
		ctx.EmitJmp(lbl31)
		ctx.MarkLabel(lbl30)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl31)
		ctx.EmitJmp(lbl13)
		ps795 := scm.PhiState{General: true}
		ps795.OverlayValues = make([]scm.JITValueDesc, 793)
		ps795.OverlayValues[1] = d1
		ps795.OverlayValues[2] = d2
		ps795.OverlayValues[3] = d3
		ps795.OverlayValues[4] = d4
		ps795.OverlayValues[5] = d5
		ps795.OverlayValues[6] = d6
		ps795.OverlayValues[7] = d7
		ps795.OverlayValues[8] = d8
		ps795.OverlayValues[9] = d9
		ps795.OverlayValues[10] = d10
		ps795.OverlayValues[11] = d11
		ps795.OverlayValues[12] = d12
		ps795.OverlayValues[13] = d13
		ps795.OverlayValues[14] = d14
		ps795.OverlayValues[15] = d15
		ps795.OverlayValues[17] = d17
		ps795.OverlayValues[18] = d18
		ps795.OverlayValues[19] = d19
		ps795.OverlayValues[20] = d20
		ps795.OverlayValues[21] = d21
		ps795.OverlayValues[22] = d22
		ps795.OverlayValues[23] = d23
		ps795.OverlayValues[24] = d24
		ps795.OverlayValues[25] = d25
		ps795.OverlayValues[26] = d26
		ps795.OverlayValues[27] = d27
		ps795.OverlayValues[28] = d28
		ps795.OverlayValues[29] = d29
		ps795.OverlayValues[30] = d30
		ps795.OverlayValues[31] = d31
		ps795.OverlayValues[32] = d32
		ps795.OverlayValues[33] = d33
		ps795.OverlayValues[34] = d34
		ps795.OverlayValues[35] = d35
		ps795.OverlayValues[36] = d36
		ps795.OverlayValues[37] = d37
		ps795.OverlayValues[38] = d38
		ps795.OverlayValues[39] = d39
		ps795.OverlayValues[40] = d40
		ps795.OverlayValues[41] = d41
		ps795.OverlayValues[42] = d42
		ps795.OverlayValues[43] = d43
		ps795.OverlayValues[44] = d44
		ps795.OverlayValues[45] = d45
		ps795.OverlayValues[46] = d46
		ps795.OverlayValues[47] = d47
		ps795.OverlayValues[48] = d48
		ps795.OverlayValues[49] = d49
		ps795.OverlayValues[50] = d50
		ps795.OverlayValues[53] = d53
		ps795.OverlayValues[54] = d54
		ps795.OverlayValues[55] = d55
		ps795.OverlayValues[111] = d111
		ps795.OverlayValues[112] = d112
		ps795.OverlayValues[113] = d113
		ps795.OverlayValues[114] = d114
		ps795.OverlayValues[115] = d115
		ps795.OverlayValues[116] = d116
		ps795.OverlayValues[117] = d117
		ps795.OverlayValues[118] = d118
		ps795.OverlayValues[119] = d119
		ps795.OverlayValues[120] = d120
		ps795.OverlayValues[121] = d121
		ps795.OverlayValues[122] = d122
		ps795.OverlayValues[123] = d123
		ps795.OverlayValues[124] = d124
		ps795.OverlayValues[125] = d125
		ps795.OverlayValues[126] = d126
		ps795.OverlayValues[127] = d127
		ps795.OverlayValues[128] = d128
		ps795.OverlayValues[129] = d129
		ps795.OverlayValues[130] = d130
		ps795.OverlayValues[131] = d131
		ps795.OverlayValues[132] = d132
		ps795.OverlayValues[133] = d133
		ps795.OverlayValues[134] = d134
		ps795.OverlayValues[135] = d135
		ps795.OverlayValues[136] = d136
		ps795.OverlayValues[137] = d137
		ps795.OverlayValues[138] = d138
		ps795.OverlayValues[139] = d139
		ps795.OverlayValues[140] = d140
		ps795.OverlayValues[143] = d143
		ps795.OverlayValues[230] = d230
		ps795.OverlayValues[231] = d231
		ps795.OverlayValues[232] = d232
		ps795.OverlayValues[233] = d233
		ps795.OverlayValues[235] = d235
		ps795.OverlayValues[236] = d236
		ps795.OverlayValues[237] = d237
		ps795.OverlayValues[238] = d238
		ps795.OverlayValues[239] = d239
		ps795.OverlayValues[240] = d240
		ps795.OverlayValues[241] = d241
		ps795.OverlayValues[242] = d242
		ps795.OverlayValues[244] = d244
		ps795.OverlayValues[246] = d246
		ps795.OverlayValues[247] = d247
		ps795.OverlayValues[248] = d248
		ps795.OverlayValues[249] = d249
		ps795.OverlayValues[250] = d250
		ps795.OverlayValues[253] = d253
		ps795.OverlayValues[357] = d357
		ps795.OverlayValues[358] = d358
		ps795.OverlayValues[359] = d359
		ps795.OverlayValues[360] = d360
		ps795.OverlayValues[361] = d361
		ps795.OverlayValues[363] = d363
		ps795.OverlayValues[364] = d364
		ps795.OverlayValues[365] = d365
		ps795.OverlayValues[366] = d366
		ps795.OverlayValues[367] = d367
		ps795.OverlayValues[368] = d368
		ps795.OverlayValues[369] = d369
		ps795.OverlayValues[370] = d370
		ps795.OverlayValues[371] = d371
		ps795.OverlayValues[372] = d372
		ps795.OverlayValues[373] = d373
		ps795.OverlayValues[374] = d374
		ps795.OverlayValues[375] = d375
		ps795.OverlayValues[376] = d376
		ps795.OverlayValues[377] = d377
		ps795.OverlayValues[378] = d378
		ps795.OverlayValues[379] = d379
		ps795.OverlayValues[380] = d380
		ps795.OverlayValues[381] = d381
		ps795.OverlayValues[382] = d382
		ps795.OverlayValues[383] = d383
		ps795.OverlayValues[384] = d384
		ps795.OverlayValues[385] = d385
		ps795.OverlayValues[386] = d386
		ps795.OverlayValues[387] = d387
		ps795.OverlayValues[388] = d388
		ps795.OverlayValues[389] = d389
		ps795.OverlayValues[390] = d390
		ps795.OverlayValues[391] = d391
		ps795.OverlayValues[392] = d392
		ps795.OverlayValues[393] = d393
		ps795.OverlayValues[537] = d537
		ps795.OverlayValues[538] = d538
		ps795.OverlayValues[539] = d539
		ps795.OverlayValues[541] = d541
		ps795.OverlayValues[542] = d542
		ps795.OverlayValues[543] = d543
		ps795.OverlayValues[544] = d544
		ps795.OverlayValues[545] = d545
		ps795.OverlayValues[546] = d546
		ps795.OverlayValues[547] = d547
		ps795.OverlayValues[549] = d549
		ps795.OverlayValues[551] = d551
		ps795.OverlayValues[552] = d552
		ps795.OverlayValues[553] = d553
		ps795.OverlayValues[554] = d554
		ps795.OverlayValues[557] = d557
		ps795.OverlayValues[713] = d713
		ps795.OverlayValues[714] = d714
		ps795.OverlayValues[715] = d715
		ps795.OverlayValues[716] = d716
		ps795.OverlayValues[718] = d718
		ps795.OverlayValues[719] = d719
		ps795.OverlayValues[720] = d720
		ps795.OverlayValues[721] = d721
		ps795.OverlayValues[722] = d722
		ps795.OverlayValues[723] = d723
		ps795.OverlayValues[724] = d724
		ps795.OverlayValues[725] = d725
		ps795.OverlayValues[727] = d727
		ps795.OverlayValues[728] = d728
		ps795.OverlayValues[729] = d729
		ps795.OverlayValues[730] = d730
		ps795.OverlayValues[731] = d731
		ps795.OverlayValues[732] = d732
		ps795.OverlayValues[733] = d733
		ps795.OverlayValues[734] = d734
		ps795.OverlayValues[735] = d735
		ps795.OverlayValues[736] = d736
		ps795.OverlayValues[737] = d737
		ps795.OverlayValues[738] = d738
		ps795.OverlayValues[739] = d739
		ps795.OverlayValues[740] = d740
		ps795.OverlayValues[741] = d741
		ps795.OverlayValues[742] = d742
		ps795.OverlayValues[743] = d743
		ps795.OverlayValues[744] = d744
		ps795.OverlayValues[745] = d745
		ps795.OverlayValues[746] = d746
		ps795.OverlayValues[747] = d747
		ps795.OverlayValues[748] = d748
		ps795.OverlayValues[749] = d749
		ps795.OverlayValues[750] = d750
		ps795.OverlayValues[751] = d751
		ps795.OverlayValues[752] = d752
		ps795.OverlayValues[753] = d753
		ps795.OverlayValues[754] = d754
		ps795.OverlayValues[755] = d755
		ps795.OverlayValues[756] = d756
		ps795.OverlayValues[757] = d757
		ps795.OverlayValues[758] = d758
		ps795.OverlayValues[759] = d759
		ps795.OverlayValues[760] = d760
		ps795.OverlayValues[761] = d761
		ps795.OverlayValues[762] = d762
		ps795.OverlayValues[763] = d763
		ps795.OverlayValues[764] = d764
		ps795.OverlayValues[765] = d765
		ps795.OverlayValues[766] = d766
		ps795.OverlayValues[767] = d767
		ps795.OverlayValues[768] = d768
		ps795.OverlayValues[769] = d769
		ps795.OverlayValues[770] = d770
		ps795.OverlayValues[771] = d771
		ps795.OverlayValues[772] = d772
		ps795.OverlayValues[773] = d773
		ps795.OverlayValues[774] = d774
		ps795.OverlayValues[775] = d775
		ps795.OverlayValues[776] = d776
		ps795.OverlayValues[777] = d777
		ps795.OverlayValues[778] = d778
		ps795.OverlayValues[779] = d779
		ps795.OverlayValues[780] = d780
		ps795.OverlayValues[781] = d781
		ps795.OverlayValues[782] = d782
		ps795.OverlayValues[783] = d783
		ps795.OverlayValues[784] = d784
		ps795.OverlayValues[785] = d785
		ps795.OverlayValues[786] = d786
		ps795.OverlayValues[787] = d787
		ps795.OverlayValues[788] = d788
		ps795.OverlayValues[789] = d789
		ps795.OverlayValues[790] = d790
		ps795.OverlayValues[791] = d791
		ps795.OverlayValues[792] = d792
		ps796 := scm.PhiState{General: true}
		ps796.OverlayValues = make([]scm.JITValueDesc, 793)
		ps796.OverlayValues[1] = d1
		ps796.OverlayValues[2] = d2
		ps796.OverlayValues[3] = d3
		ps796.OverlayValues[4] = d4
		ps796.OverlayValues[5] = d5
		ps796.OverlayValues[6] = d6
		ps796.OverlayValues[7] = d7
		ps796.OverlayValues[8] = d8
		ps796.OverlayValues[9] = d9
		ps796.OverlayValues[10] = d10
		ps796.OverlayValues[11] = d11
		ps796.OverlayValues[12] = d12
		ps796.OverlayValues[13] = d13
		ps796.OverlayValues[14] = d14
		ps796.OverlayValues[15] = d15
		ps796.OverlayValues[17] = d17
		ps796.OverlayValues[18] = d18
		ps796.OverlayValues[19] = d19
		ps796.OverlayValues[20] = d20
		ps796.OverlayValues[21] = d21
		ps796.OverlayValues[22] = d22
		ps796.OverlayValues[23] = d23
		ps796.OverlayValues[24] = d24
		ps796.OverlayValues[25] = d25
		ps796.OverlayValues[26] = d26
		ps796.OverlayValues[27] = d27
		ps796.OverlayValues[28] = d28
		ps796.OverlayValues[29] = d29
		ps796.OverlayValues[30] = d30
		ps796.OverlayValues[31] = d31
		ps796.OverlayValues[32] = d32
		ps796.OverlayValues[33] = d33
		ps796.OverlayValues[34] = d34
		ps796.OverlayValues[35] = d35
		ps796.OverlayValues[36] = d36
		ps796.OverlayValues[37] = d37
		ps796.OverlayValues[38] = d38
		ps796.OverlayValues[39] = d39
		ps796.OverlayValues[40] = d40
		ps796.OverlayValues[41] = d41
		ps796.OverlayValues[42] = d42
		ps796.OverlayValues[43] = d43
		ps796.OverlayValues[44] = d44
		ps796.OverlayValues[45] = d45
		ps796.OverlayValues[46] = d46
		ps796.OverlayValues[47] = d47
		ps796.OverlayValues[48] = d48
		ps796.OverlayValues[49] = d49
		ps796.OverlayValues[50] = d50
		ps796.OverlayValues[53] = d53
		ps796.OverlayValues[54] = d54
		ps796.OverlayValues[55] = d55
		ps796.OverlayValues[111] = d111
		ps796.OverlayValues[112] = d112
		ps796.OverlayValues[113] = d113
		ps796.OverlayValues[114] = d114
		ps796.OverlayValues[115] = d115
		ps796.OverlayValues[116] = d116
		ps796.OverlayValues[117] = d117
		ps796.OverlayValues[118] = d118
		ps796.OverlayValues[119] = d119
		ps796.OverlayValues[120] = d120
		ps796.OverlayValues[121] = d121
		ps796.OverlayValues[122] = d122
		ps796.OverlayValues[123] = d123
		ps796.OverlayValues[124] = d124
		ps796.OverlayValues[125] = d125
		ps796.OverlayValues[126] = d126
		ps796.OverlayValues[127] = d127
		ps796.OverlayValues[128] = d128
		ps796.OverlayValues[129] = d129
		ps796.OverlayValues[130] = d130
		ps796.OverlayValues[131] = d131
		ps796.OverlayValues[132] = d132
		ps796.OverlayValues[133] = d133
		ps796.OverlayValues[134] = d134
		ps796.OverlayValues[135] = d135
		ps796.OverlayValues[136] = d136
		ps796.OverlayValues[137] = d137
		ps796.OverlayValues[138] = d138
		ps796.OverlayValues[139] = d139
		ps796.OverlayValues[140] = d140
		ps796.OverlayValues[143] = d143
		ps796.OverlayValues[230] = d230
		ps796.OverlayValues[231] = d231
		ps796.OverlayValues[232] = d232
		ps796.OverlayValues[233] = d233
		ps796.OverlayValues[235] = d235
		ps796.OverlayValues[236] = d236
		ps796.OverlayValues[237] = d237
		ps796.OverlayValues[238] = d238
		ps796.OverlayValues[239] = d239
		ps796.OverlayValues[240] = d240
		ps796.OverlayValues[241] = d241
		ps796.OverlayValues[242] = d242
		ps796.OverlayValues[244] = d244
		ps796.OverlayValues[246] = d246
		ps796.OverlayValues[247] = d247
		ps796.OverlayValues[248] = d248
		ps796.OverlayValues[249] = d249
		ps796.OverlayValues[250] = d250
		ps796.OverlayValues[253] = d253
		ps796.OverlayValues[357] = d357
		ps796.OverlayValues[358] = d358
		ps796.OverlayValues[359] = d359
		ps796.OverlayValues[360] = d360
		ps796.OverlayValues[361] = d361
		ps796.OverlayValues[363] = d363
		ps796.OverlayValues[364] = d364
		ps796.OverlayValues[365] = d365
		ps796.OverlayValues[366] = d366
		ps796.OverlayValues[367] = d367
		ps796.OverlayValues[368] = d368
		ps796.OverlayValues[369] = d369
		ps796.OverlayValues[370] = d370
		ps796.OverlayValues[371] = d371
		ps796.OverlayValues[372] = d372
		ps796.OverlayValues[373] = d373
		ps796.OverlayValues[374] = d374
		ps796.OverlayValues[375] = d375
		ps796.OverlayValues[376] = d376
		ps796.OverlayValues[377] = d377
		ps796.OverlayValues[378] = d378
		ps796.OverlayValues[379] = d379
		ps796.OverlayValues[380] = d380
		ps796.OverlayValues[381] = d381
		ps796.OverlayValues[382] = d382
		ps796.OverlayValues[383] = d383
		ps796.OverlayValues[384] = d384
		ps796.OverlayValues[385] = d385
		ps796.OverlayValues[386] = d386
		ps796.OverlayValues[387] = d387
		ps796.OverlayValues[388] = d388
		ps796.OverlayValues[389] = d389
		ps796.OverlayValues[390] = d390
		ps796.OverlayValues[391] = d391
		ps796.OverlayValues[392] = d392
		ps796.OverlayValues[393] = d393
		ps796.OverlayValues[537] = d537
		ps796.OverlayValues[538] = d538
		ps796.OverlayValues[539] = d539
		ps796.OverlayValues[541] = d541
		ps796.OverlayValues[542] = d542
		ps796.OverlayValues[543] = d543
		ps796.OverlayValues[544] = d544
		ps796.OverlayValues[545] = d545
		ps796.OverlayValues[546] = d546
		ps796.OverlayValues[547] = d547
		ps796.OverlayValues[549] = d549
		ps796.OverlayValues[551] = d551
		ps796.OverlayValues[552] = d552
		ps796.OverlayValues[553] = d553
		ps796.OverlayValues[554] = d554
		ps796.OverlayValues[557] = d557
		ps796.OverlayValues[713] = d713
		ps796.OverlayValues[714] = d714
		ps796.OverlayValues[715] = d715
		ps796.OverlayValues[716] = d716
		ps796.OverlayValues[718] = d718
		ps796.OverlayValues[719] = d719
		ps796.OverlayValues[720] = d720
		ps796.OverlayValues[721] = d721
		ps796.OverlayValues[722] = d722
		ps796.OverlayValues[723] = d723
		ps796.OverlayValues[724] = d724
		ps796.OverlayValues[725] = d725
		ps796.OverlayValues[727] = d727
		ps796.OverlayValues[728] = d728
		ps796.OverlayValues[729] = d729
		ps796.OverlayValues[730] = d730
		ps796.OverlayValues[731] = d731
		ps796.OverlayValues[732] = d732
		ps796.OverlayValues[733] = d733
		ps796.OverlayValues[734] = d734
		ps796.OverlayValues[735] = d735
		ps796.OverlayValues[736] = d736
		ps796.OverlayValues[737] = d737
		ps796.OverlayValues[738] = d738
		ps796.OverlayValues[739] = d739
		ps796.OverlayValues[740] = d740
		ps796.OverlayValues[741] = d741
		ps796.OverlayValues[742] = d742
		ps796.OverlayValues[743] = d743
		ps796.OverlayValues[744] = d744
		ps796.OverlayValues[745] = d745
		ps796.OverlayValues[746] = d746
		ps796.OverlayValues[747] = d747
		ps796.OverlayValues[748] = d748
		ps796.OverlayValues[749] = d749
		ps796.OverlayValues[750] = d750
		ps796.OverlayValues[751] = d751
		ps796.OverlayValues[752] = d752
		ps796.OverlayValues[753] = d753
		ps796.OverlayValues[754] = d754
		ps796.OverlayValues[755] = d755
		ps796.OverlayValues[756] = d756
		ps796.OverlayValues[757] = d757
		ps796.OverlayValues[758] = d758
		ps796.OverlayValues[759] = d759
		ps796.OverlayValues[760] = d760
		ps796.OverlayValues[761] = d761
		ps796.OverlayValues[762] = d762
		ps796.OverlayValues[763] = d763
		ps796.OverlayValues[764] = d764
		ps796.OverlayValues[765] = d765
		ps796.OverlayValues[766] = d766
		ps796.OverlayValues[767] = d767
		ps796.OverlayValues[768] = d768
		ps796.OverlayValues[769] = d769
		ps796.OverlayValues[770] = d770
		ps796.OverlayValues[771] = d771
		ps796.OverlayValues[772] = d772
		ps796.OverlayValues[773] = d773
		ps796.OverlayValues[774] = d774
		ps796.OverlayValues[775] = d775
		ps796.OverlayValues[776] = d776
		ps796.OverlayValues[777] = d777
		ps796.OverlayValues[778] = d778
		ps796.OverlayValues[779] = d779
		ps796.OverlayValues[780] = d780
		ps796.OverlayValues[781] = d781
		ps796.OverlayValues[782] = d782
		ps796.OverlayValues[783] = d783
		ps796.OverlayValues[784] = d784
		ps796.OverlayValues[785] = d785
		ps796.OverlayValues[786] = d786
		ps796.OverlayValues[787] = d787
		ps796.OverlayValues[788] = d788
		ps796.OverlayValues[789] = d789
		ps796.OverlayValues[790] = d790
		ps796.OverlayValues[791] = d791
		ps796.OverlayValues[792] = d792
		snap797 := d1
		snap798 := d2
		snap799 := d3
		snap800 := d4
		snap801 := d5
		snap802 := d6
		snap803 := d7
		snap804 := d8
		snap805 := d9
		snap806 := d10
		snap807 := d11
		snap808 := d12
		snap809 := d13
		snap810 := d14
		snap811 := d15
		snap812 := d17
		snap813 := d18
		snap814 := d19
		snap815 := d20
		snap816 := d21
		snap817 := d22
		snap818 := d23
		snap819 := d24
		snap820 := d25
		snap821 := d26
		snap822 := d27
		snap823 := d28
		snap824 := d29
		snap825 := d30
		snap826 := d31
		snap827 := d32
		snap828 := d33
		snap829 := d34
		snap830 := d35
		snap831 := d36
		snap832 := d37
		snap833 := d38
		snap834 := d39
		snap835 := d40
		snap836 := d41
		snap837 := d42
		snap838 := d43
		snap839 := d44
		snap840 := d45
		snap841 := d46
		snap842 := d47
		snap843 := d48
		snap844 := d49
		snap845 := d50
		snap846 := d53
		snap847 := d54
		snap848 := d55
		snap849 := d111
		snap850 := d112
		snap851 := d113
		snap852 := d114
		snap853 := d115
		snap854 := d116
		snap855 := d117
		snap856 := d118
		snap857 := d119
		snap858 := d120
		snap859 := d121
		snap860 := d122
		snap861 := d123
		snap862 := d124
		snap863 := d125
		snap864 := d126
		snap865 := d127
		snap866 := d128
		snap867 := d129
		snap868 := d130
		snap869 := d131
		snap870 := d132
		snap871 := d133
		snap872 := d134
		snap873 := d135
		snap874 := d136
		snap875 := d137
		snap876 := d138
		snap877 := d139
		snap878 := d140
		snap879 := d143
		snap880 := d230
		snap881 := d231
		snap882 := d232
		snap883 := d233
		snap884 := d235
		snap885 := d236
		snap886 := d237
		snap887 := d238
		snap888 := d239
		snap889 := d240
		snap890 := d241
		snap891 := d242
		snap892 := d244
		snap893 := d246
		snap894 := d247
		snap895 := d248
		snap896 := d249
		snap897 := d250
		snap898 := d253
		snap899 := d357
		snap900 := d358
		snap901 := d359
		snap902 := d360
		snap903 := d361
		snap904 := d363
		snap905 := d364
		snap906 := d365
		snap907 := d366
		snap908 := d367
		snap909 := d368
		snap910 := d369
		snap911 := d370
		snap912 := d371
		snap913 := d372
		snap914 := d373
		snap915 := d374
		snap916 := d375
		snap917 := d376
		snap918 := d377
		snap919 := d378
		snap920 := d379
		snap921 := d380
		snap922 := d381
		snap923 := d382
		snap924 := d383
		snap925 := d384
		snap926 := d385
		snap927 := d386
		snap928 := d387
		snap929 := d388
		snap930 := d389
		snap931 := d390
		snap932 := d391
		snap933 := d392
		snap934 := d393
		snap935 := d537
		snap936 := d538
		snap937 := d539
		snap938 := d541
		snap939 := d542
		snap940 := d543
		snap941 := d544
		snap942 := d545
		snap943 := d546
		snap944 := d547
		snap945 := d549
		snap946 := d551
		snap947 := d552
		snap948 := d553
		snap949 := d554
		snap950 := d557
		snap951 := d713
		snap952 := d714
		snap953 := d715
		snap954 := d716
		snap955 := d718
		snap956 := d719
		snap957 := d720
		snap958 := d721
		snap959 := d722
		snap960 := d723
		snap961 := d724
		snap962 := d725
		snap963 := d727
		snap964 := d728
		snap965 := d729
		snap966 := d730
		snap967 := d731
		snap968 := d732
		snap969 := d733
		snap970 := d734
		snap971 := d735
		snap972 := d736
		snap973 := d737
		snap974 := d738
		snap975 := d739
		snap976 := d740
		snap977 := d741
		snap978 := d742
		snap979 := d743
		snap980 := d744
		snap981 := d745
		snap982 := d746
		snap983 := d747
		snap984 := d748
		snap985 := d749
		snap986 := d750
		snap987 := d751
		snap988 := d752
		snap989 := d753
		snap990 := d754
		snap991 := d755
		snap992 := d756
		snap993 := d757
		snap994 := d758
		snap995 := d759
		snap996 := d760
		snap997 := d761
		snap998 := d762
		snap999 := d763
		snap1000 := d764
		snap1001 := d765
		snap1002 := d766
		snap1003 := d767
		snap1004 := d768
		snap1005 := d769
		snap1006 := d770
		snap1007 := d771
		snap1008 := d772
		snap1009 := d773
		snap1010 := d774
		snap1011 := d775
		snap1012 := d776
		snap1013 := d777
		snap1014 := d778
		snap1015 := d779
		snap1016 := d780
		snap1017 := d781
		snap1018 := d782
		snap1019 := d783
		snap1020 := d784
		snap1021 := d785
		snap1022 := d786
		snap1023 := d787
		snap1024 := d788
		snap1025 := d789
		snap1026 := d790
		snap1027 := d791
		snap1028 := d792
		alloc1029 := ctx.SnapshotAllocState()
		if !bbs[12].Rendered {
			bbs[12].RenderPS(ps796)
		}
		ctx.RestoreAllocState(alloc1029)
		d1 = snap797
		d2 = snap798
		d3 = snap799
		d4 = snap800
		d5 = snap801
		d6 = snap802
		d7 = snap803
		d8 = snap804
		d9 = snap805
		d10 = snap806
		d11 = snap807
		d12 = snap808
		d13 = snap809
		d14 = snap810
		d15 = snap811
		d17 = snap812
		d18 = snap813
		d19 = snap814
		d20 = snap815
		d21 = snap816
		d22 = snap817
		d23 = snap818
		d24 = snap819
		d25 = snap820
		d26 = snap821
		d27 = snap822
		d28 = snap823
		d29 = snap824
		d30 = snap825
		d31 = snap826
		d32 = snap827
		d33 = snap828
		d34 = snap829
		d35 = snap830
		d36 = snap831
		d37 = snap832
		d38 = snap833
		d39 = snap834
		d40 = snap835
		d41 = snap836
		d42 = snap837
		d43 = snap838
		d44 = snap839
		d45 = snap840
		d46 = snap841
		d47 = snap842
		d48 = snap843
		d49 = snap844
		d50 = snap845
		d53 = snap846
		d54 = snap847
		d55 = snap848
		d111 = snap849
		d112 = snap850
		d113 = snap851
		d114 = snap852
		d115 = snap853
		d116 = snap854
		d117 = snap855
		d118 = snap856
		d119 = snap857
		d120 = snap858
		d121 = snap859
		d122 = snap860
		d123 = snap861
		d124 = snap862
		d125 = snap863
		d126 = snap864
		d127 = snap865
		d128 = snap866
		d129 = snap867
		d130 = snap868
		d131 = snap869
		d132 = snap870
		d133 = snap871
		d134 = snap872
		d135 = snap873
		d136 = snap874
		d137 = snap875
		d138 = snap876
		d139 = snap877
		d140 = snap878
		d143 = snap879
		d230 = snap880
		d231 = snap881
		d232 = snap882
		d233 = snap883
		d235 = snap884
		d236 = snap885
		d237 = snap886
		d238 = snap887
		d239 = snap888
		d240 = snap889
		d241 = snap890
		d242 = snap891
		d244 = snap892
		d246 = snap893
		d247 = snap894
		d248 = snap895
		d249 = snap896
		d250 = snap897
		d253 = snap898
		d357 = snap899
		d358 = snap900
		d359 = snap901
		d360 = snap902
		d361 = snap903
		d363 = snap904
		d364 = snap905
		d365 = snap906
		d366 = snap907
		d367 = snap908
		d368 = snap909
		d369 = snap910
		d370 = snap911
		d371 = snap912
		d372 = snap913
		d373 = snap914
		d374 = snap915
		d375 = snap916
		d376 = snap917
		d377 = snap918
		d378 = snap919
		d379 = snap920
		d380 = snap921
		d381 = snap922
		d382 = snap923
		d383 = snap924
		d384 = snap925
		d385 = snap926
		d386 = snap927
		d387 = snap928
		d388 = snap929
		d389 = snap930
		d390 = snap931
		d391 = snap932
		d392 = snap933
		d393 = snap934
		d537 = snap935
		d538 = snap936
		d539 = snap937
		d541 = snap938
		d542 = snap939
		d543 = snap940
		d544 = snap941
		d545 = snap942
		d546 = snap943
		d547 = snap944
		d549 = snap945
		d551 = snap946
		d552 = snap947
		d553 = snap948
		d554 = snap949
		d557 = snap950
		d713 = snap951
		d714 = snap952
		d715 = snap953
		d716 = snap954
		d718 = snap955
		d719 = snap956
		d720 = snap957
		d721 = snap958
		d722 = snap959
		d723 = snap960
		d724 = snap961
		d725 = snap962
		d727 = snap963
		d728 = snap964
		d729 = snap965
		d730 = snap966
		d731 = snap967
		d732 = snap968
		d733 = snap969
		d734 = snap970
		d735 = snap971
		d736 = snap972
		d737 = snap973
		d738 = snap974
		d739 = snap975
		d740 = snap976
		d741 = snap977
		d742 = snap978
		d743 = snap979
		d744 = snap980
		d745 = snap981
		d746 = snap982
		d747 = snap983
		d748 = snap984
		d749 = snap985
		d750 = snap986
		d751 = snap987
		d752 = snap988
		d753 = snap989
		d754 = snap990
		d755 = snap991
		d756 = snap992
		d757 = snap993
		d758 = snap994
		d759 = snap995
		d760 = snap996
		d761 = snap997
		d762 = snap998
		d763 = snap999
		d764 = snap1000
		d765 = snap1001
		d766 = snap1002
		d767 = snap1003
		d768 = snap1004
		d769 = snap1005
		d770 = snap1006
		d771 = snap1007
		d772 = snap1008
		d773 = snap1009
		d774 = snap1010
		d775 = snap1011
		d776 = snap1012
		d777 = snap1013
		d778 = snap1014
		d779 = snap1015
		d780 = snap1016
		d781 = snap1017
		d782 = snap1018
		d783 = snap1019
		d784 = snap1020
		d785 = snap1021
		d786 = snap1022
		d787 = snap1023
		d788 = snap1024
		d789 = snap1025
		d790 = snap1026
		d791 = snap1027
		d792 = snap1028
		if !bbs[11].Rendered {
			return bbs[11].RenderPS(ps795)
		}
		return result
		ctx.FreeDesc(&d791)
		return result
	}
	ps1030 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps1030)
	ctx.MarkLabel(lbl0)
	d1031 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d1031)
	ctx.BindReg(r1, &d1031)
	ctx.EmitMovPairToResult(&d1031, &result)
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
